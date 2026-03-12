package async

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/audit"
)

type Service struct {
	repo         Repository
	auditService *audit.Service
}

func NewService(repository Repository, auditService ...*audit.Service) *Service {
	var auditSvc *audit.Service
	if len(auditService) > 0 {
		auditSvc = auditService[0]
	}
	return &Service{repo: repository, auditService: auditSvc}
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

	if terminalState == StateSucceeded {
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
	}
	if terminalState == StateFailed {
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

func (s *Service) PollDueTasks(ctx context.Context, limit int, poller RemotePoller) error {
	if poller == nil {
		return nil
	}
	due, err := s.repo.ListDueTasks(ctx, time.Now().UTC(), limit)
	if err != nil {
		return err
	}
	for _, task := range due {
		if err := ctx.Err(); err != nil {
			return err
		}
		result, pollErr := poller.Poll(ctx, task)
		if pollErr != nil {
			message := pollErr.Error()
			_, _ = s.RecordPoll(ctx, RecordPollInput{
				AsyncTaskID:  task.ID,
				ErrorMessage: message,
			})
			_, _ = s.UpdateTaskStatus(ctx, UpdateStatusInput{
				ID:            task.ID,
				RemoteStatus:  StatePolling,
				TerminalState: StateFailed,
				RemoteResponse: map[string]any{
					"error": message,
				},
			})
			continue
		}
		if _, err := s.RecordPoll(ctx, RecordPollInput{
			AsyncTaskID:  task.ID,
			StatusCode:   result.StatusCode,
			RemoteStatus: result.RemoteStatus,
			ResponseBody: result.ResponseBody,
			ErrorMessage: result.ErrorMessage,
			DurationMS:   result.DurationMS,
		}); err != nil {
			return err
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
