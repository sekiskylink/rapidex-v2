package async

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/audit"
	"basepro/backend/internal/sukumad/traceevent"
)

type Service struct {
	repo            Repository
	auditService    *audit.Service
	deliveryUpdater interface {
		CompleteFromAsyncSuccess(context.Context, int64, string) error
		CompleteFromAsyncFailure(context.Context, int64, string, string) error
	}
	requestStatusUpdater interface {
		SetProcessing(context.Context, int64) error
		SetCompleted(context.Context, int64) error
		SetFailed(context.Context, int64) error
	}
	eventWriter traceevent.Writer
}

func NewService(repository Repository, auditService ...*audit.Service) *Service {
	var auditSvc *audit.Service
	if len(auditService) > 0 {
		auditSvc = auditService[0]
	}
	return &Service{repo: repository, auditService: auditSvc}
}

func (s *Service) WithReconciliation(
	deliveryUpdater interface {
		CompleteFromAsyncSuccess(context.Context, int64, string) error
		CompleteFromAsyncFailure(context.Context, int64, string, string) error
	},
	requestStatusUpdater interface {
		SetProcessing(context.Context, int64) error
		SetCompleted(context.Context, int64) error
		SetFailed(context.Context, int64) error
	},
) *Service {
	s.deliveryUpdater = deliveryUpdater
	s.requestStatusUpdater = requestStatusUpdater
	return s
}

func (s *Service) WithEventWriter(eventWriter traceevent.Writer) *Service {
	s.eventWriter = eventWriter
	return s
}

func (s *Service) ListTasks(ctx context.Context, query ListQuery) (ListResult, error) {
	return s.repo.ListTasks(ctx, query)
}

func (s *Service) GetTask(ctx context.Context, id int64) (Record, error) {
	record, err := s.repo.GetTaskByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"job not found"}})
		}
		return Record{}, err
	}
	return record, nil
}

func (s *Service) ListPolls(ctx context.Context, id int64, query ListQuery) (PollListResult, error) {
	if _, err := s.GetTask(ctx, id); err != nil {
		return PollListResult{}, err
	}
	return s.repo.ListPolls(ctx, id, query)
}

func (s *Service) CreateTask(ctx context.Context, input CreateInput) (Record, error) {
	if input.DeliveryAttemptID <= 0 {
		return Record{}, apperror.ValidationWithDetails("validation failed", map[string]any{"deliveryAttemptId": []string{"is required"}})
	}

	remoteStatus := normalizeState(input.RemoteStatus, StatePending)
	created, err := s.repo.CreateTask(ctx, CreateParams{
		UID:               newUID(),
		DeliveryAttemptID: input.DeliveryAttemptID,
		RemoteJobID:       strings.TrimSpace(input.RemoteJobID),
		PollURL:           strings.TrimSpace(input.PollURL),
		RemoteStatus:      remoteStatus,
		NextPollAt:        cloneTimePtr(input.NextPollAt),
		RemoteResponse:    cloneJSONMap(input.RemoteResponse),
	})
	if err != nil {
		return Record{}, err
	}

	s.logAudit(ctx, audit.Event{
		Action:      "async_task.created",
		ActorUserID: input.ActorID,
		EntityType:  "async_task",
		EntityID:    strPtr(fmt.Sprintf("%d", created.ID)),
		Metadata: map[string]any{
			"uid":               created.UID,
			"deliveryAttemptId": created.DeliveryAttemptID,
			"deliveryUid":       created.DeliveryUID,
			"requestUid":        created.RequestUID,
			"remoteJobId":       created.RemoteJobID,
			"currentState":      created.CurrentState,
		},
	})
	s.appendEvent(ctx, traceevent.WriteInput{
		RequestID:         &created.RequestID,
		DeliveryAttemptID: &created.DeliveryAttemptID,
		AsyncTaskID:       &created.ID,
		EventType:         traceevent.EventAsyncCreated,
		EventLevel:        "info",
		Message:           traceevent.Message("Async task created", "Async task %s created", created.UID),
		CorrelationID:     created.CorrelationID,
		Actor:             traceevent.Actor{Type: traceevent.ActorUser, UserID: input.ActorID},
		SourceComponent:   "async.service",
		EventData: map[string]any{
			"asyncTaskUid": created.UID,
			"deliveryUid":  created.DeliveryUID,
			"requestUid":   created.RequestUID,
			"remoteJobId":  created.RemoteJobID,
			"state":        created.CurrentState,
		},
	})

	return created, nil
}

func (s *Service) UpdateTaskStatus(ctx context.Context, input UpdateStatusInput) (Record, error) {
	current, err := s.GetTask(ctx, input.ID)
	if err != nil {
		return Record{}, err
	}

	terminalState := strings.ToLower(strings.TrimSpace(input.TerminalState))
	if terminalState != "" && terminalState != StateSucceeded && terminalState != StateFailed {
		return Record{}, apperror.ValidationWithDetails("validation failed", map[string]any{"terminalState": []string{"must be succeeded or failed"}})
	}

	remoteStatus := normalizeState(input.RemoteStatus, current.RemoteStatus)
	completedAt := current.CompletedAt
	if terminalState != "" {
		now := time.Now().UTC()
		completedAt = &now
	}

	updated, err := s.repo.UpdateTask(ctx, UpdateParams{
		ID:             input.ID,
		RemoteJobID:    firstNonEmpty(strings.TrimSpace(input.RemoteJobID), current.RemoteJobID),
		PollURL:        firstNonEmpty(strings.TrimSpace(input.PollURL), current.PollURL),
		RemoteStatus:   remoteStatus,
		TerminalState:  terminalState,
		NextPollAt:     cloneTimePtr(input.NextPollAt),
		CompletedAt:    cloneTimePtr(completedAt),
		RemoteResponse: cloneJSONMap(input.RemoteResponse),
	})
	if err != nil {
		return Record{}, err
	}

	if terminalState == "" && s.requestStatusUpdater != nil {
		_ = s.requestStatusUpdater.SetProcessing(ctx, updated.RequestID)
	}

	if terminalState == StateSucceeded {
		s.reconcileTerminalState(ctx, updated)
		s.logAudit(ctx, audit.Event{
			Action:      "async_task.completed",
			ActorUserID: input.ActorID,
			EntityType:  "async_task",
			EntityID:    strPtr(fmt.Sprintf("%d", updated.ID)),
			Metadata: map[string]any{
				"uid":          updated.UID,
				"requestUid":   updated.RequestUID,
				"deliveryUid":  updated.DeliveryUID,
				"remoteStatus": updated.RemoteStatus,
			},
		})
		s.appendEvent(ctx, traceevent.WriteInput{
			RequestID:         &updated.RequestID,
			DeliveryAttemptID: &updated.DeliveryAttemptID,
			AsyncTaskID:       &updated.ID,
			EventType:         traceevent.EventAsyncCompleted,
			EventLevel:        "info",
			Message:           traceevent.Message("Async task completed", "Async task %s completed", updated.UID),
			CorrelationID:     updated.CorrelationID,
			Actor:             traceevent.Actor{Type: traceevent.ActorUser, UserID: input.ActorID},
			SourceComponent:   "async.service",
			EventData:         map[string]any{"asyncTaskUid": updated.UID, "remoteStatus": updated.RemoteStatus},
		})
	}
	if terminalState == StateFailed {
		s.reconcileTerminalState(ctx, updated)
		s.logAudit(ctx, audit.Event{
			Action:      "async_task.failed",
			ActorUserID: input.ActorID,
			EntityType:  "async_task",
			EntityID:    strPtr(fmt.Sprintf("%d", updated.ID)),
			Metadata: map[string]any{
				"uid":          updated.UID,
				"requestUid":   updated.RequestUID,
				"deliveryUid":  updated.DeliveryUID,
				"remoteStatus": updated.RemoteStatus,
			},
		})
		s.appendEvent(ctx, traceevent.WriteInput{
			RequestID:         &updated.RequestID,
			DeliveryAttemptID: &updated.DeliveryAttemptID,
			AsyncTaskID:       &updated.ID,
			EventType:         traceevent.EventAsyncFailed,
			EventLevel:        "error",
			Message:           traceevent.Message("Async task failed", "Async task %s failed", updated.UID),
			CorrelationID:     updated.CorrelationID,
			Actor:             traceevent.Actor{Type: traceevent.ActorUser, UserID: input.ActorID},
			SourceComponent:   "async.service",
			EventData:         map[string]any{"asyncTaskUid": updated.UID, "remoteStatus": updated.RemoteStatus},
		})
	}

	return updated, nil
}

func (s *Service) RecordPoll(ctx context.Context, input RecordPollInput) (PollRecord, error) {
	if input.AsyncTaskID <= 0 {
		return PollRecord{}, apperror.ValidationWithDetails("validation failed", map[string]any{"asyncTaskId": []string{"is required"}})
	}
	if _, err := s.GetTask(ctx, input.AsyncTaskID); err != nil {
		return PollRecord{}, err
	}
	record, err := s.repo.RecordPoll(ctx, input)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PollRecord{}, apperror.ValidationWithDetails("validation failed", map[string]any{"asyncTaskId": []string{"job not found"}})
		}
		return PollRecord{}, err
	}
	return record, nil
}

func (s *Service) PollDueTasks(ctx context.Context, exec PollExecution, limit int, poller RemotePoller) error {
	if poller == nil {
		return nil
	}
	if limit <= 0 {
		limit = 10
	}
	claimTimeout := exec.ClaimTimeout
	if claimTimeout <= 0 {
		claimTimeout = time.Minute
	}
	for i := 0; i < limit; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		task, err := s.repo.ClaimNextDueTask(ctx, time.Now().UTC(), claimTimeout, exec.WorkerRunID)
		if err != nil {
			if errors.Is(err, ErrNoEligibleTask) {
				return nil
			}
			return err
		}
		exec.Increment("polls_picked")
		s.appendEvent(ctx, traceevent.WriteInput{
			RequestID:         &task.RequestID,
			DeliveryAttemptID: &task.DeliveryAttemptID,
			AsyncTaskID:       &task.ID,
			WorkerRunID:       workerRunIDPtr(exec.WorkerRunID),
			EventType:         traceevent.EventAsyncPollStarted,
			EventLevel:        "info",
			Message:           traceevent.Message("Async poll started", "Polling async task %s", task.UID),
			CorrelationID:     task.CorrelationID,
			Actor:             traceevent.Actor{Type: traceevent.ActorSystem},
			SourceComponent:   "async.poller",
			EventData:         map[string]any{"asyncTaskUid": task.UID, "remoteJobId": task.RemoteJobID},
		})
		result, pollErr := poller.Poll(ctx, task)
		if pollErr != nil {
			message := pollErr.Error()
			_, _ = s.RecordPoll(ctx, RecordPollInput{
				AsyncTaskID:  task.ID,
				ErrorMessage: message,
			})
			_, _ = s.UpdateTaskStatus(ctx, UpdateStatusInput{
				ID:           task.ID,
				RemoteStatus: StatePolling,
				NextPollAt:   nextRetryPollAt(),
				RemoteResponse: map[string]any{
					"error": message,
				},
			})
			s.appendEvent(ctx, traceevent.WriteInput{
				RequestID:         &task.RequestID,
				DeliveryAttemptID: &task.DeliveryAttemptID,
				AsyncTaskID:       &task.ID,
				WorkerRunID:       workerRunIDPtr(exec.WorkerRunID),
				EventType:         traceevent.EventAsyncPollFailed,
				EventLevel:        "warning",
				Message:           traceevent.Message("Async poll failed", "Polling async task %s failed", task.UID),
				CorrelationID:     task.CorrelationID,
				Actor:             traceevent.Actor{Type: traceevent.ActorSystem},
				SourceComponent:   "async.poller",
				EventData:         map[string]any{"asyncTaskUid": task.UID, "error": message},
			})
			exec.Increment("polls_failed")
			continue
		}
		if _, err := s.RecordPoll(ctx, RecordPollInput{
			AsyncTaskID:          task.ID,
			StatusCode:           result.StatusCode,
			RemoteStatus:         result.RemoteStatus,
			ResponseBody:         result.ResponseBody,
			ResponseContentType:  result.ResponseContentType,
			ResponseBodyFiltered: result.ResponseBodyFiltered,
			ErrorMessage:         result.ErrorMessage,
			DurationMS:           result.DurationMS,
		}); err != nil {
			return err
		}
		if result.ResponseBodyFiltered {
			s.appendEvent(ctx, traceevent.WriteInput{
				RequestID:         &task.RequestID,
				DeliveryAttemptID: &task.DeliveryAttemptID,
				AsyncTaskID:       &task.ID,
				WorkerRunID:       workerRunIDPtr(exec.WorkerRunID),
				EventType:         "async.poll.filtered_content_type",
				EventLevel:        "warning",
				Message:           traceevent.Message("Async poll content filtered", "Async task %s poll response content was filtered", task.UID),
				CorrelationID:     task.CorrelationID,
				Actor:             traceevent.Actor{Type: traceevent.ActorSystem},
				SourceComponent:   "async.poller",
				EventData: map[string]any{
					"asyncTaskUid":        task.UID,
					"responseContentType": result.ResponseContentType,
					"filtered":            true,
				},
			})
		}
		if _, err := s.UpdateTaskStatus(ctx, UpdateStatusInput{
			ID:             task.ID,
			RemoteStatus:   normalizeState(result.RemoteStatus, StatePolling),
			TerminalState:  strings.ToLower(strings.TrimSpace(result.TerminalState)),
			NextPollAt:     cloneTimePtr(result.NextPollAt),
			RemoteResponse: cloneJSONMap(result.RemoteResponse),
		}); err != nil {
			return err
		}
		s.appendEvent(ctx, traceevent.WriteInput{
			RequestID:         &task.RequestID,
			DeliveryAttemptID: &task.DeliveryAttemptID,
			AsyncTaskID:       &task.ID,
			WorkerRunID:       workerRunIDPtr(exec.WorkerRunID),
			EventType:         traceevent.EventAsyncPollSucceeded,
			EventLevel:        "info",
			Message:           traceevent.Message("Async poll succeeded", "Polling async task %s succeeded", task.UID),
			CorrelationID:     task.CorrelationID,
			Actor:             traceevent.Actor{Type: traceevent.ActorSystem},
			SourceComponent:   "async.poller",
			EventData: map[string]any{
				"asyncTaskUid": task.UID,
				"remoteStatus": result.RemoteStatus,
				"statusCode":   result.StatusCode,
			},
		})
		exec.Increment("polls_completed")
	}
	return nil
}

func (s *Service) ReconcileTerminalTasks(ctx context.Context, limit int) error {
	tasks, err := s.repo.ListTerminalTasksForRecovery(ctx, limit)
	if err != nil {
		return err
	}
	for _, task := range tasks {
		if err := ctx.Err(); err != nil {
			return err
		}
		s.reconcileTerminalState(ctx, task)
		s.appendEvent(ctx, traceevent.WriteInput{
			RequestID:         &task.RequestID,
			DeliveryAttemptID: &task.DeliveryAttemptID,
			AsyncTaskID:       &task.ID,
			EventType:         "async.recovery.reconciled",
			EventLevel:        "warning",
			Message:           traceevent.Message("Recovered async reconciliation", "Recovered async task %s reconciliation", task.UID),
			CorrelationID:     task.CorrelationID,
			Actor:             traceevent.Actor{Type: traceevent.ActorSystem},
			SourceComponent:   "async.service",
			EventData: map[string]any{
				"asyncTaskUid":  task.UID,
				"terminalState": task.TerminalState,
				"recovery":      true,
			},
		})
	}
	return nil
}

func (s *Service) logAudit(ctx context.Context, event audit.Event) {
	if s.auditService == nil {
		return
	}
	_ = s.auditService.Log(ctx, event)
}

func normalizeState(value string, fallback string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "", StatePending, StatePolling, StateSucceeded, StateFailed:
		if normalized == "" {
			return fallback
		}
		return normalized
	default:
		return fallback
	}
}

func firstNonEmpty(value string, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func strPtr(v string) *string {
	return &v
}

func (s *Service) reconcileTerminalState(ctx context.Context, record Record) {
	if s.requestStatusUpdater != nil && record.CurrentState == StatePolling {
		_ = s.requestStatusUpdater.SetProcessing(ctx, record.RequestID)
	}
	if record.TerminalState == "" {
		return
	}
	if s.deliveryUpdater != nil {
		if record.TerminalState == StateSucceeded {
			_ = s.deliveryUpdater.CompleteFromAsyncSuccess(ctx, record.DeliveryAttemptID, marshalRemoteResponse(record.RemoteResponse))
		}
		if record.TerminalState == StateFailed {
			_ = s.deliveryUpdater.CompleteFromAsyncFailure(
				ctx,
				record.DeliveryAttemptID,
				marshalRemoteResponse(record.RemoteResponse),
				firstNonEmpty(extractErrorMessage(record.RemoteResponse), "dhis2 async task failed"),
			)
		}
	}
	if s.requestStatusUpdater != nil {
		if record.TerminalState == StateSucceeded {
			_ = s.requestStatusUpdater.SetCompleted(ctx, record.RequestID)
		}
		if record.TerminalState == StateFailed {
			_ = s.requestStatusUpdater.SetFailed(ctx, record.RequestID)
		}
	}
}

func nextRetryPollAt() *time.Time {
	next := time.Now().UTC().Add(30 * time.Second)
	return &next
}

func marshalRemoteResponse(input map[string]any) string {
	if len(input) == 0 {
		return ""
	}
	bytes, err := json.Marshal(input)
	if err != nil {
		return ""
	}
	return string(bytes)
}

func extractErrorMessage(input map[string]any) string {
	for _, key := range []string{"error", "message", "description"} {
		value, ok := input[key]
		if ok {
			if text, isText := value.(string); isText {
				return text
			}
		}
	}
	return ""
}

func (s *Service) appendEvent(ctx context.Context, input traceevent.WriteInput) {
	if s.eventWriter == nil {
		return
	}
	_ = s.eventWriter.AppendEvent(ctx, input)
}

func workerRunIDPtr(runID int64) *int64 {
	if runID <= 0 {
		return nil
	}
	return &runID
}
