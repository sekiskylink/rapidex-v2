package delivery

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/audit"
	asyncjobs "basepro/backend/internal/sukumad/async"
	"basepro/backend/internal/sukumad/traceevent"
	"github.com/jackc/pgx/v5/pgconn"
)

const retryDelay = 5 * time.Minute

type Service struct {
	repo         Repository
	auditService *audit.Service
	dispatcher   interface {
		Submit(context.Context, DispatchInput) (DispatchResult, error)
	}
	asyncService interface {
		CreateTask(context.Context, asyncjobs.CreateInput) (asyncjobs.Record, error)
	}
	requestStatusUpdater interface {
		SetProcessing(context.Context, int64) error
		SetCompleted(context.Context, int64) error
		SetFailed(context.Context, int64) error
		SetBlocked(context.Context, int64, string, *time.Time) error
	}
	targetUpdater interface {
		SetTargetPending(context.Context, int64, int64) error
		SetTargetBlocked(context.Context, int64, int64, string, *time.Time) error
		SetTargetProcessing(context.Context, int64, int64) error
		SetTargetSucceeded(context.Context, int64, int64) error
		SetTargetFailed(context.Context, int64, int64, string) error
	}
	eventWriter traceevent.Writer
}

type DispatchResult struct {
	HTTPStatus           *int
	ResponseBody         string
	ResponseContentType  string
	ResponseBodyFiltered bool
	ResponseSummary      map[string]any
	ErrorMessage         string
	RemoteJobID          string
	PollURL              string
	RemoteStatus         string
	RemoteResponse       map[string]any
	Async                bool
	Terminal             bool
	Succeeded            bool
}

func NewService(repository Repository, auditService ...*audit.Service) *Service {
	var auditSvc *audit.Service
	if len(auditService) > 0 {
		auditSvc = auditService[0]
	}
	return &Service{repo: repository, auditService: auditSvc}
}

func (s *Service) WithDispatcher(dispatcher interface {
	Submit(context.Context, DispatchInput) (DispatchResult, error)
}) *Service {
	s.dispatcher = dispatcher
	return s
}

func (s *Service) WithAsyncService(asyncService interface {
	CreateTask(context.Context, asyncjobs.CreateInput) (asyncjobs.Record, error)
}) *Service {
	s.asyncService = asyncService
	return s
}

func (s *Service) WithRequestStatusUpdater(updater interface {
	SetProcessing(context.Context, int64) error
	SetCompleted(context.Context, int64) error
	SetFailed(context.Context, int64) error
	SetBlocked(context.Context, int64, string, *time.Time) error
}) *Service {
	s.requestStatusUpdater = updater
	return s
}

func (s *Service) WithTargetUpdater(updater interface {
	SetTargetPending(context.Context, int64, int64) error
	SetTargetBlocked(context.Context, int64, int64, string, *time.Time) error
	SetTargetProcessing(context.Context, int64, int64) error
	SetTargetSucceeded(context.Context, int64, int64) error
	SetTargetFailed(context.Context, int64, int64, string) error
}) *Service {
	s.targetUpdater = updater
	return s
}

func (s *Service) WithEventWriter(eventWriter traceevent.Writer) *Service {
	s.eventWriter = eventWriter
	return s
}

func (s *Service) ListDeliveries(ctx context.Context, query ListQuery) (ListResult, error) {
	return s.repo.ListDeliveries(ctx, query)
}

func (s *Service) GetDelivery(ctx context.Context, id int64) (Record, error) {
	record, err := s.repo.GetDeliveryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"delivery not found"}})
		}
		return Record{}, err
	}
	return record, nil
}

func (s *Service) CreatePendingDelivery(ctx context.Context, input CreateInput) (Record, error) {
	if input.RequestID <= 0 || input.ServerID <= 0 {
		return Record{}, apperror.ValidationWithDetails("validation failed", map[string]any{
			"requestId": []string{"is required"},
			"serverId":  []string{"is required"},
		})
	}

	created, err := s.repo.CreateDelivery(ctx, CreateParams{
		UID:                  newUID(),
		RequestID:            input.RequestID,
		ServerID:             input.ServerID,
		AttemptNumber:        1,
		Status:               StatusPending,
		ResponseBody:         "",
		ResponseContentType:  "",
		ResponseBodyFiltered: false,
		ResponseSummary:      map[string]any{},
		ErrorMessage:         "",
		SubmissionHoldReason: "",
		HoldPolicySource:     "",
		TerminalReason:       "",
	})
	if err != nil {
		if mapped := mapConstraintError(err); mapped != nil {
			return Record{}, mapped
		}
		return Record{}, err
	}

	s.logAudit(ctx, audit.Event{
		Action:      "delivery.created",
		ActorUserID: input.ActorID,
		EntityType:  "delivery",
		EntityID:    strPtr(fmt.Sprintf("%d", created.ID)),
		Metadata: map[string]any{
			"uid":           created.UID,
			"requestId":     created.RequestID,
			"requestUid":    created.RequestUID,
			"serverId":      created.ServerID,
			"serverName":    created.ServerName,
			"attemptNumber": created.AttemptNumber,
			"status":        created.Status,
		},
	})
	if s.targetUpdater != nil {
		_ = s.targetUpdater.SetTargetPending(ctx, created.RequestID, created.ServerID)
	}
	s.appendEvent(ctx, traceevent.WriteInput{
		RequestID:         &created.RequestID,
		DeliveryAttemptID: &created.ID,
		EventType:         traceevent.EventDeliveryCreated,
		EventLevel:        "info",
		Message:           traceevent.Message("Delivery created", "Delivery %s created for request %s", created.UID, created.RequestUID),
		Actor:             traceevent.Actor{Type: traceevent.ActorUser, UserID: input.ActorID},
		SourceComponent:   "delivery.service",
		CorrelationID:     input.CorrelationID,
		EventData: map[string]any{
			"deliveryUid":   created.UID,
			"requestUid":    created.RequestUID,
			"serverName":    created.ServerName,
			"attemptNumber": created.AttemptNumber,
			"status":        created.Status,
		},
	})

	return created, nil
}

func (s *Service) MarkRunning(ctx context.Context, id int64) (Record, error) {
	record, err := s.mustLoadForTransition(ctx, id)
	if err != nil {
		return Record{}, err
	}
	if record.Status != StatusPending && record.Status != StatusRetrying {
		return Record{}, apperror.ValidationWithDetails("validation failed", map[string]any{"status": []string{"delivery can only start from pending or retrying"}})
	}
	now := time.Now().UTC()
	updated, err := s.repo.UpdateDelivery(ctx, UpdateParams{
		ID:                  id,
		Status:              StatusRunning,
		HTTPStatus:          nil,
		ResponseBody:        record.ResponseBody,
		ResponseContentType: record.ResponseContentType,
		ResponseBodyFiltered: record.ResponseBodyFiltered,
		ResponseSummary:     cloneJSONMap(record.ResponseSummary),
		ErrorMessage:        "",
		SubmissionHoldReason: "",
		NextEligibleAt:      nil,
		HoldPolicySource:    "",
		TerminalReason:      "",
		StartedAt:           &now,
		FinishedAt:          nil,
		RetryAt:             nil,
	})
	if err != nil {
		return Record{}, err
	}
	eventType := traceevent.EventDeliveryStarted
	message := traceevent.Message("Delivery started", "Delivery %s started", updated.UID)
	if record.Status == StatusRetrying {
		eventType = traceevent.EventDeliveryRetryStarted
		message = traceevent.Message("Delivery retry started", "Delivery retry %s started", updated.UID)
	}
	s.appendEvent(ctx, traceevent.WriteInput{
		RequestID:         &updated.RequestID,
		DeliveryAttemptID: &updated.ID,
		EventType:         eventType,
		EventLevel:        "info",
		Message:           message,
		SourceComponent:   "delivery.service",
		CorrelationID:     updated.CorrelationID,
		Actor:             traceevent.Actor{Type: traceevent.ActorSystem},
		EventData: map[string]any{
			"deliveryUid":   updated.UID,
			"requestUid":    updated.RequestUID,
			"attemptNumber": updated.AttemptNumber,
			"status":        updated.Status,
		},
	})
	if s.targetUpdater != nil {
		_ = s.targetUpdater.SetTargetProcessing(ctx, updated.RequestID, updated.ServerID)
	}
	return updated, nil
}

func (s *Service) MarkSucceeded(ctx context.Context, input CompletionInput) (Record, error) {
	record, err := s.mustLoadForTransition(ctx, input.ID)
	if err != nil {
		return Record{}, err
	}
	if record.Status != StatusRunning {
		return Record{}, apperror.ValidationWithDetails("validation failed", map[string]any{"status": []string{"delivery can only succeed from running"}})
	}
	now := time.Now().UTC()
	updated, err := s.repo.UpdateDelivery(ctx, UpdateParams{
		ID:                  input.ID,
		Status:              StatusSucceeded,
		HTTPStatus:          input.HTTPStatus,
		ResponseBody:        strings.TrimSpace(input.ResponseBody),
		ResponseContentType: firstNonEmpty(input.ResponseContentType, record.ResponseContentType),
		ResponseBodyFiltered: input.ResponseBodyFiltered || record.ResponseBodyFiltered,
		ResponseSummary:     mergeResponseSummary(record.ResponseSummary, input.ResponseSummary),
		ErrorMessage:        "",
		SubmissionHoldReason: "",
		NextEligibleAt:      nil,
		HoldPolicySource:    "",
		TerminalReason:      "",
		StartedAt:           record.StartedAt,
		FinishedAt:          &now,
		RetryAt:             nil,
	})
	if err != nil {
		return Record{}, err
	}

	s.logAudit(ctx, audit.Event{
		Action:      "delivery.succeeded",
		ActorUserID: input.ActorID,
		EntityType:  "delivery",
		EntityID:    strPtr(fmt.Sprintf("%d", updated.ID)),
		Metadata: map[string]any{
			"uid":           updated.UID,
			"requestUid":    updated.RequestUID,
			"serverName":    updated.ServerName,
			"attemptNumber": updated.AttemptNumber,
			"httpStatus":    updated.HTTPStatus,
		},
	})
	s.appendEvent(ctx, traceevent.WriteInput{
		RequestID:         &updated.RequestID,
		DeliveryAttemptID: &updated.ID,
		EventType:         traceevent.EventDeliveryResponse,
		EventLevel:        "info",
		Message:           traceevent.Message("Delivery response received", "Delivery %s received a response", updated.UID),
		SourceComponent:   "delivery.service",
		CorrelationID:     updated.CorrelationID,
		Actor:             traceevent.Actor{Type: traceevent.ActorSystem},
		EventData: map[string]any{
			"deliveryUid":  updated.UID,
			"httpStatus":   updated.HTTPStatus,
			"responseBody": updated.ResponseBody,
		},
	})
	s.appendEvent(ctx, traceevent.WriteInput{
		RequestID:         &updated.RequestID,
		DeliveryAttemptID: &updated.ID,
		EventType:         traceevent.EventDeliverySucceeded,
		EventLevel:        "info",
		Message:           traceevent.Message("Delivery succeeded", "Delivery %s succeeded", updated.UID),
		SourceComponent:   "delivery.service",
		CorrelationID:     updated.CorrelationID,
		Actor:             traceevent.Actor{Type: traceevent.ActorUser, UserID: input.ActorID},
		EventData: map[string]any{
			"deliveryUid":   updated.UID,
			"httpStatus":    updated.HTTPStatus,
			"attemptNumber": updated.AttemptNumber,
		},
	})
	if s.targetUpdater != nil {
		_ = s.targetUpdater.SetTargetSucceeded(ctx, updated.RequestID, updated.ServerID)
	}

	return updated, nil
}

func (s *Service) MarkFailed(ctx context.Context, input CompletionInput) (Record, error) {
	record, err := s.mustLoadForTransition(ctx, input.ID)
	if err != nil {
		return Record{}, err
	}
	if record.Status != StatusRunning {
		return Record{}, apperror.ValidationWithDetails("validation failed", map[string]any{"status": []string{"delivery can only fail from running"}})
	}
	now := time.Now().UTC()
	updated, err := s.repo.UpdateDelivery(ctx, UpdateParams{
		ID:                  input.ID,
		Status:              StatusFailed,
		HTTPStatus:          input.HTTPStatus,
		ResponseBody:        strings.TrimSpace(input.ResponseBody),
		ResponseContentType: record.ResponseContentType,
		ResponseBodyFiltered: record.ResponseBodyFiltered,
		ResponseSummary:     cloneJSONMap(record.ResponseSummary),
		ErrorMessage:        strings.TrimSpace(input.ErrorMessage),
		SubmissionHoldReason: "",
		NextEligibleAt:      nil,
		HoldPolicySource:    "",
		TerminalReason:      strings.TrimSpace(input.ErrorMessage),
		StartedAt:           record.StartedAt,
		FinishedAt:          &now,
		RetryAt:             nil,
	})
	if err != nil {
		return Record{}, err
	}

	s.logAudit(ctx, audit.Event{
		Action:      "delivery.failed",
		ActorUserID: input.ActorID,
		EntityType:  "delivery",
		EntityID:    strPtr(fmt.Sprintf("%d", updated.ID)),
		Metadata: map[string]any{
			"uid":           updated.UID,
			"requestUid":    updated.RequestUID,
			"serverName":    updated.ServerName,
			"attemptNumber": updated.AttemptNumber,
			"httpStatus":    updated.HTTPStatus,
			"errorMessage":  updated.ErrorMessage,
		},
	})
	s.appendEvent(ctx, traceevent.WriteInput{
		RequestID:         &updated.RequestID,
		DeliveryAttemptID: &updated.ID,
		EventType:         traceevent.EventDeliveryResponse,
		EventLevel:        "warning",
		Message:           traceevent.Message("Delivery response received", "Delivery %s received a failure response", updated.UID),
		SourceComponent:   "delivery.service",
		CorrelationID:     updated.CorrelationID,
		Actor:             traceevent.Actor{Type: traceevent.ActorSystem},
		EventData: map[string]any{
			"deliveryUid":  updated.UID,
			"httpStatus":   updated.HTTPStatus,
			"responseBody": updated.ResponseBody,
		},
	})
	s.appendEvent(ctx, traceevent.WriteInput{
		RequestID:         &updated.RequestID,
		DeliveryAttemptID: &updated.ID,
		EventType:         traceevent.EventDeliveryFailed,
		EventLevel:        "error",
		Message:           traceevent.Message("Delivery failed", "Delivery %s failed", updated.UID),
		SourceComponent:   "delivery.service",
		CorrelationID:     updated.CorrelationID,
		Actor:             traceevent.Actor{Type: traceevent.ActorUser, UserID: input.ActorID},
		EventData: map[string]any{
			"deliveryUid":   updated.UID,
			"httpStatus":    updated.HTTPStatus,
			"errorMessage":  updated.ErrorMessage,
			"attemptNumber": updated.AttemptNumber,
		},
	})
	if s.targetUpdater != nil {
		_ = s.targetUpdater.SetTargetFailed(ctx, updated.RequestID, updated.ServerID, updated.ErrorMessage)
	}

	return updated, nil
}

func (s *Service) RetryDelivery(ctx context.Context, actorID *int64, id int64) (Record, error) {
	record, err := s.mustLoadForTransition(ctx, id)
	if err != nil {
		return Record{}, err
	}
	if record.Status != StatusFailed {
		return Record{}, apperror.ValidationWithDetails("validation failed", map[string]any{"status": []string{"only failed deliveries can be retried"}})
	}
	maxRetries := ResolveMaxRetries(record.ServerCode)
	if (record.AttemptNumber - 1) >= maxRetries {
		s.appendEvent(ctx, traceevent.WriteInput{
			RequestID:         &record.RequestID,
			DeliveryAttemptID: &record.ID,
			EventType:         "delivery.retry.rejected.max_retries",
			EventLevel:        "warning",
			Message:           traceevent.Message("Delivery retry rejected", "Delivery %s exceeded max retries", record.UID),
			SourceComponent:   "delivery.service",
			CorrelationID:     record.CorrelationID,
			Actor:             traceevent.Actor{Type: traceevent.ActorUser, UserID: actorID},
			EventData: map[string]any{
				"deliveryUid":   record.UID,
				"attemptNumber": record.AttemptNumber,
				"maxRetries":    maxRetries,
				"serverCode":    record.ServerCode,
			},
		})
		return Record{}, apperror.ValidationWithDetails("validation failed", map[string]any{"status": []string{"max retries reached"}})
	}

	retryAt := time.Now().UTC().Add(retryDelay)
	created, err := s.repo.CreateDelivery(ctx, CreateParams{
		UID:                  newUID(),
		RequestID:            record.RequestID,
		ServerID:             record.ServerID,
		AttemptNumber:        record.AttemptNumber + 1,
		Status:               StatusRetrying,
		ResponseBody:         "",
		ResponseContentType:  "",
		ResponseBodyFiltered: false,
		ResponseSummary:      map[string]any{},
		ErrorMessage:         "",
		SubmissionHoldReason: "",
		HoldPolicySource:     "",
		TerminalReason:       "",
		RetryAt:              &retryAt,
	})
	if err != nil {
		if mapped := mapConstraintError(err); mapped != nil {
			return Record{}, mapped
		}
		return Record{}, err
	}

	s.logAudit(ctx, audit.Event{
		Action:      "delivery.retry",
		ActorUserID: actorID,
		EntityType:  "delivery",
		EntityID:    strPtr(fmt.Sprintf("%d", created.ID)),
		Metadata: map[string]any{
			"uid":                 created.UID,
			"requestUid":          created.RequestUID,
			"serverName":          created.ServerName,
			"attemptNumber":       created.AttemptNumber,
			"retryAt":             created.RetryAt,
			"sourceDeliveryId":    record.ID,
			"sourceDeliveryUid":   record.UID,
			"sourceAttemptNumber": record.AttemptNumber,
		},
	})
	s.appendEvent(ctx, traceevent.WriteInput{
		RequestID:         &created.RequestID,
		DeliveryAttemptID: &created.ID,
		EventType:         traceevent.EventDeliveryRetrySched,
		EventLevel:        "warning",
		Message:           traceevent.Message("Delivery retry scheduled", "Delivery retry %s scheduled", created.UID),
		SourceComponent:   "delivery.service",
		CorrelationID:     created.CorrelationID,
		Actor:             traceevent.Actor{Type: traceevent.ActorUser, UserID: actorID},
		EventData: map[string]any{
			"deliveryUid":         created.UID,
			"requestUid":          created.RequestUID,
			"retryAt":             created.RetryAt,
			"sourceDeliveryId":    record.ID,
			"sourceDeliveryUid":   record.UID,
			"sourceAttemptNumber": record.AttemptNumber,
		},
	})

	return created, nil
}

func (s *Service) SubmitDHIS2Delivery(ctx context.Context, input DispatchInput) (Record, error) {
	if input.DeliveryID <= 0 {
		return Record{}, apperror.ValidationWithDetails("validation failed", map[string]any{"deliveryId": []string{"is required"}})
	}
	if strings.TrimSpace(input.Server.SystemType) == "" {
		return Record{}, apperror.ValidationWithDetails("validation failed", map[string]any{"systemType": []string{"is required"}})
	}
	if s.dispatcher == nil {
		return s.GetDelivery(ctx, input.DeliveryID)
	}
	now := time.Now().UTC()
	windowPolicy := ResolveSubmissionWindow(input.Server.Code)
	allowed, nextEligibleAt := windowPolicy.Evaluate(now)
	if !allowed {
		record, err := s.repo.UpdateDelivery(ctx, UpdateParams{
			ID:                  input.DeliveryID,
			Status:              StatusPending,
			HTTPStatus:          nil,
			ResponseBody:        "",
			ResponseContentType: "",
			ResponseBodyFiltered: false,
			ResponseSummary:     map[string]any{},
			ErrorMessage:        "",
			SubmissionHoldReason: "window_closed",
			NextEligibleAt:      nextEligibleAt,
			HoldPolicySource:    windowPolicy.Source,
			TerminalReason:      "",
			StartedAt:           nil,
			FinishedAt:          nil,
			RetryAt:             nil,
		})
		if err != nil {
			return Record{}, err
		}
		if s.targetUpdater != nil {
			_ = s.targetUpdater.SetTargetBlocked(ctx, record.RequestID, record.ServerID, "window_closed", nextEligibleAt)
		}
		if s.requestStatusUpdater != nil {
			_ = s.requestStatusUpdater.SetBlocked(ctx, input.RequestID, "window_closed", nextEligibleAt)
		}
		s.appendEvent(ctx, traceevent.WriteInput{
			RequestID:         &record.RequestID,
			DeliveryAttemptID: &record.ID,
			EventType:         "delivery.deferred.window",
			EventLevel:        "info",
			Message:           traceevent.Message("Delivery deferred by submission window", "Delivery %s deferred by submission window", record.UID),
			SourceComponent:   "delivery.service",
			CorrelationID:     record.CorrelationID,
			Actor:             traceevent.Actor{Type: traceevent.ActorUser, UserID: input.ActorID},
			EventData: map[string]any{
				"deliveryUid":     record.UID,
				"serverCode":      input.Server.Code,
				"policySource":    windowPolicy.Source,
				"startHour":       windowPolicy.StartHour,
				"endHour":         windowPolicy.EndHour,
				"nextEligibleAt":  nextEligibleAt,
				"deferReason":     "window_closed",
			},
		})
		return record, nil
	}

	running, err := s.MarkRunning(ctx, input.DeliveryID)
	if err != nil {
		return Record{}, err
	}

	result, err := s.dispatcher.Submit(ctx, input)
	if err != nil {
		failed, failErr := s.MarkFailed(ctx, CompletionInput{
			ID:           input.DeliveryID,
			ResponseBody: "",
			ErrorMessage: err.Error(),
			ActorID:      input.ActorID,
		})
		if failErr != nil {
			return Record{}, failErr
		}
		if s.requestStatusUpdater != nil {
			_ = s.requestStatusUpdater.SetFailed(ctx, input.RequestID)
		}
		s.logAudit(ctx, audit.Event{
			Action:      "dhis2.submission.failed",
			ActorUserID: input.ActorID,
			EntityType:  "delivery",
			EntityID:    strPtr(fmt.Sprintf("%d", failed.ID)),
			Metadata: map[string]any{
				"deliveryUid": failed.UID,
				"requestUid":  input.RequestUID,
				"serverName":  input.Server.Name,
			},
		})
		return failed, nil
	}

	s.logAudit(ctx, audit.Event{
		Action:      "dhis2.submission.started",
		ActorUserID: input.ActorID,
		EntityType:  "delivery",
		EntityID:    strPtr(fmt.Sprintf("%d", running.ID)),
		Metadata: map[string]any{
			"deliveryUid": running.UID,
			"requestUid":  input.RequestUID,
			"serverName":  input.Server.Name,
			"systemType":  input.Server.SystemType,
		},
	})

	if result.Async {
		if s.asyncService == nil {
			return Record{}, apperror.ValidationWithDetails("validation failed", map[string]any{"async": []string{"async service is not configured"}})
		}
		if _, err := s.repo.UpdateDelivery(ctx, UpdateParams{
			ID:                  running.ID,
			Status:              StatusRunning,
			HTTPStatus:          result.HTTPStatus,
			ResponseBody:        strings.TrimSpace(result.ResponseBody),
			ResponseContentType: result.ResponseContentType,
			ResponseBodyFiltered: result.ResponseBodyFiltered,
			ResponseSummary:     cloneJSONMap(result.ResponseSummary),
			ErrorMessage:        strings.TrimSpace(result.ErrorMessage),
			SubmissionHoldReason: "",
			NextEligibleAt:      nil,
			HoldPolicySource:    "",
			TerminalReason:      "",
			StartedAt:           running.StartedAt,
			FinishedAt:          nil,
			RetryAt:             nil,
		}); err != nil {
			return Record{}, err
		}
		if result.ResponseBodyFiltered {
			s.appendEvent(ctx, traceevent.WriteInput{
				RequestID:         &running.RequestID,
				DeliveryAttemptID: &running.ID,
				EventType:         "delivery.response.filtered_content_type",
				EventLevel:        "warning",
				Message:           traceevent.Message("Delivery response content filtered", "Delivery %s response content was filtered", running.UID),
				SourceComponent:   "delivery.service",
				CorrelationID:     running.CorrelationID,
				Actor:             traceevent.Actor{Type: traceevent.ActorSystem},
				EventData: map[string]any{
					"deliveryUid":         running.UID,
					"responseContentType": result.ResponseContentType,
					"httpStatus":          result.HTTPStatus,
					"filtered":            true,
					"summary":             cloneJSONMap(result.ResponseSummary),
				},
			})
		}
		task, err := s.asyncService.CreateTask(ctx, asyncjobs.CreateInput{
			DeliveryAttemptID: running.ID,
			RemoteJobID:       strings.TrimSpace(result.RemoteJobID),
			PollURL:           strings.TrimSpace(result.PollURL),
			RemoteStatus:      firstNonEmpty(strings.TrimSpace(result.RemoteStatus), asyncjobs.StatePending),
			NextPollAt:        nextPollTime(),
			RemoteResponse:    cloneJSONMap(result.RemoteResponse),
			ActorID:           input.ActorID,
		})
		if err != nil {
			return Record{}, err
		}
		if s.requestStatusUpdater != nil {
			_ = s.requestStatusUpdater.SetProcessing(ctx, input.RequestID)
		}
		s.logAudit(ctx, audit.Event{
			Action:      "dhis2.async_task.created",
			ActorUserID: input.ActorID,
			EntityType:  "async_task",
			EntityID:    strPtr(fmt.Sprintf("%d", task.ID)),
			Metadata: map[string]any{
				"deliveryUid":  running.UID,
				"requestUid":   input.RequestUID,
				"remoteJobId":  task.RemoteJobID,
				"asyncTaskUid": task.UID,
			},
		})
		return s.GetDelivery(ctx, running.ID)
	}

	if result.Terminal && result.Succeeded {
		running.ResponseContentType = result.ResponseContentType
		running.ResponseBodyFiltered = result.ResponseBodyFiltered
		running.ResponseSummary = cloneJSONMap(result.ResponseSummary)
		record, err := s.MarkSucceeded(ctx, CompletionInput{
			ID:                  running.ID,
			HTTPStatus:          result.HTTPStatus,
			ResponseBody:        result.ResponseBody,
			ResponseContentType: result.ResponseContentType,
			ResponseBodyFiltered: result.ResponseBodyFiltered,
			ResponseSummary:     cloneJSONMap(result.ResponseSummary),
			ActorID:             input.ActorID,
		})
		if err != nil {
			return Record{}, err
		}
		if s.requestStatusUpdater != nil {
			_ = s.requestStatusUpdater.SetCompleted(ctx, input.RequestID)
		}
		s.logAudit(ctx, audit.Event{
			Action:      "dhis2.submission.succeeded",
			ActorUserID: input.ActorID,
			EntityType:  "delivery",
			EntityID:    strPtr(fmt.Sprintf("%d", record.ID)),
			Metadata: map[string]any{
				"deliveryUid": record.UID,
				"requestUid":  input.RequestUID,
				"serverName":  input.Server.Name,
				"httpStatus":  result.HTTPStatus,
			},
		})
		return record, nil
	}

	running.ResponseContentType = result.ResponseContentType
	running.ResponseBodyFiltered = result.ResponseBodyFiltered
	running.ResponseSummary = cloneJSONMap(result.ResponseSummary)
	record, err := s.MarkFailed(ctx, CompletionInput{
		ID:                  running.ID,
		HTTPStatus:          result.HTTPStatus,
		ResponseBody:        result.ResponseBody,
		ResponseContentType: result.ResponseContentType,
		ResponseBodyFiltered: result.ResponseBodyFiltered,
		ResponseSummary:     cloneJSONMap(result.ResponseSummary),
		ErrorMessage:        firstNonEmpty(result.ErrorMessage, "dhis2 submission failed"),
		ActorID:             input.ActorID,
	})
	if err != nil {
		return Record{}, err
	}
	if s.requestStatusUpdater != nil {
		_ = s.requestStatusUpdater.SetFailed(ctx, input.RequestID)
	}
	s.logAudit(ctx, audit.Event{
		Action:      "dhis2.submission.failed",
		ActorUserID: input.ActorID,
		EntityType:  "delivery",
		EntityID:    strPtr(fmt.Sprintf("%d", record.ID)),
		Metadata: map[string]any{
			"deliveryUid": record.UID,
			"requestUid":  input.RequestUID,
			"serverName":  input.Server.Name,
			"httpStatus":  result.HTTPStatus,
		},
	})
	return record, nil
}

func (s *Service) CompleteFromAsyncSuccess(ctx context.Context, deliveryID int64, responseBody string) error {
	_, err := s.MarkSucceeded(ctx, CompletionInput{
		ID:           deliveryID,
		ResponseBody: responseBody,
	})
	return err
}

func (s *Service) CompleteFromAsyncFailure(ctx context.Context, deliveryID int64, responseBody string, errorMessage string) error {
	_, err := s.MarkFailed(ctx, CompletionInput{
		ID:           deliveryID,
		ResponseBody: responseBody,
		ErrorMessage: errorMessage,
	})
	return err
}

func (s *Service) appendEvent(ctx context.Context, input traceevent.WriteInput) {
	if s.eventWriter == nil {
		return
	}
	_ = s.eventWriter.AppendEvent(ctx, input)
}

func (s *Service) mustLoadForTransition(ctx context.Context, id int64) (Record, error) {
	record, err := s.repo.GetDeliveryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"delivery not found"}})
		}
		return Record{}, err
	}
	return record, nil
}

func (s *Service) logAudit(ctx context.Context, event audit.Event) {
	if s.auditService == nil {
		return
	}
	_ = s.auditService.Log(ctx, event)
}

func mapConstraintError(err error) *apperror.AppError {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return nil
	}
	switch pgErr.Code {
	case "23503":
		if strings.Contains(pgErr.ConstraintName, "request_id") {
			return apperror.ValidationWithDetails("validation failed", map[string]any{"requestId": []string{"request not found"}})
		}
		if strings.Contains(pgErr.ConstraintName, "server_id") {
			return apperror.ValidationWithDetails("validation failed", map[string]any{"serverId": []string{"server not found"}})
		}
	case "23505":
		if strings.Contains(pgErr.ConstraintName, "uid") {
			return apperror.ValidationWithDetails("validation failed", map[string]any{"uid": []string{"must be unique"}})
		}
		if strings.Contains(pgErr.ConstraintName, "attempt") {
			return apperror.ValidationWithDetails("validation failed", map[string]any{"attemptNumber": []string{"must be unique for this request"}})
		}
	}
	return nil
}

func strPtr(v string) *string {
	return &v
}

func firstNonEmpty(value string, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func nextPollTime() *time.Time {
	next := time.Now().UTC().Add(15 * time.Second)
	return &next
}

func cloneJSONMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func mergeResponseSummary(existing map[string]any, next map[string]any) map[string]any {
	if len(next) == 0 {
		return cloneJSONMap(existing)
	}
	return cloneJSONMap(next)
}
