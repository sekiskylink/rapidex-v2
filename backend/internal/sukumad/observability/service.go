package observability

import (
	"context"
	"database/sql"
	"errors"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/sukumad/ratelimit"
	"basepro/backend/internal/sukumad/worker"
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) ListWorkers(ctx context.Context, query worker.ListQuery) (worker.ListResult, error) {
	return s.repository.workers.ListRuns(ctx, query)
}

func (s *Service) GetWorker(ctx context.Context, id int64) (worker.Record, error) {
	return s.repository.workers.GetRun(ctx, id)
}

func (s *Service) ListRateLimits(ctx context.Context, query ratelimit.ListQuery) (ratelimit.ListResult, error) {
	return s.repository.rateLimits.ListPolicies(ctx, query)
}

func (s *Service) AppendEvent(ctx context.Context, input EventWriteInput) error {
	if normalizeLevel(input.EventLevel) == "" {
		return apperror.ValidationWithDetails("validation failed", map[string]any{"level": []string{"must be info, warning, or error"}})
	}
	if input.EventType == "" {
		return apperror.ValidationWithDetails("validation failed", map[string]any{"eventType": []string{"is required"}})
	}
	input.EventLevel = normalizeLevel(input.EventLevel)
	input.Actor.Type = normalizeActorType(input.Actor.Type)
	input.EventData = sanitizeEventData(input.EventData)
	_, err := s.repository.AppendEvent(ctx, input)
	return err
}

func (s *Service) GetEvent(ctx context.Context, id int64) (EventRecord, error) {
	item, err := s.repository.GetEvent(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return EventRecord{}, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"event not found"}})
		}
		return EventRecord{}, err
	}
	return item, nil
}

func (s *Service) ListEvents(ctx context.Context, query EventListQuery) (EventListResult, error) {
	if query.Level != "" && normalizeLevel(query.Level) == "" {
		return EventListResult{}, apperror.ValidationWithDetails("validation failed", map[string]any{"level": []string{"must be info, warning, or error"}})
	}
	if query.From != nil && query.To != nil && query.To.Before(*query.From) {
		return EventListResult{}, apperror.ValidationWithDetails("validation failed", map[string]any{"to": []string{"must be after from"}})
	}
	return s.repository.ListEvents(ctx, query)
}

func (s *Service) ListRequestEvents(ctx context.Context, requestID int64, query EventListQuery) (EventListResult, error) {
	exists, err := s.repository.HasRequest(ctx, requestID)
	if err != nil {
		return EventListResult{}, err
	}
	if !exists {
		return EventListResult{}, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"request not found"}})
	}
	query.RequestID = &requestID
	return s.repository.ListEvents(ctx, query)
}

func (s *Service) ListDeliveryEvents(ctx context.Context, deliveryID int64, query EventListQuery) (EventListResult, error) {
	exists, err := s.repository.HasDelivery(ctx, deliveryID)
	if err != nil {
		return EventListResult{}, err
	}
	if !exists {
		return EventListResult{}, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"delivery not found"}})
	}
	query.DeliveryAttemptID = &deliveryID
	return s.repository.ListEvents(ctx, query)
}

func (s *Service) ListJobEvents(ctx context.Context, jobID int64, query EventListQuery) (EventListResult, error) {
	exists, err := s.repository.HasJob(ctx, jobID)
	if err != nil {
		return EventListResult{}, err
	}
	if !exists {
		return EventListResult{}, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"job not found"}})
	}
	query.AsyncTaskID = &jobID
	return s.repository.ListEvents(ctx, query)
}

func (s *Service) TraceByCorrelationID(ctx context.Context, correlationID string) (TraceResult, error) {
	if correlationID == "" {
		return TraceResult{}, apperror.ValidationWithDetails("validation failed", map[string]any{"correlationId": []string{"is required"}})
	}
	return s.repository.TraceByCorrelationID(ctx, correlationID)
}
