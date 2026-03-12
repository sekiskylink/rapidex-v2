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
	"github.com/jackc/pgx/v5/pgconn"
)

const retryDelay = 5 * time.Minute

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
		UID:           newUID(),
		RequestID:     input.RequestID,
		ServerID:      input.ServerID,
		AttemptNumber: 1,
		Status:        StatusPending,
		ResponseBody:  "",
		ErrorMessage:  "",
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
	return s.repo.UpdateDelivery(ctx, UpdateParams{
		ID:           id,
		Status:       StatusRunning,
		HTTPStatus:   nil,
		ResponseBody: record.ResponseBody,
		ErrorMessage: "",
		StartedAt:    &now,
		FinishedAt:   nil,
		RetryAt:      nil,
	})
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
		ID:           input.ID,
		Status:       StatusSucceeded,
		HTTPStatus:   input.HTTPStatus,
		ResponseBody: strings.TrimSpace(input.ResponseBody),
		ErrorMessage: "",
		StartedAt:    record.StartedAt,
		FinishedAt:   &now,
		RetryAt:      nil,
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
		ID:           input.ID,
		Status:       StatusFailed,
		HTTPStatus:   input.HTTPStatus,
		ResponseBody: strings.TrimSpace(input.ResponseBody),
		ErrorMessage: strings.TrimSpace(input.ErrorMessage),
		StartedAt:    record.StartedAt,
		FinishedAt:   &now,
		RetryAt:      nil,
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

	retryAt := time.Now().UTC().Add(retryDelay)
	created, err := s.repo.CreateDelivery(ctx, CreateParams{
		UID:           newUID(),
		RequestID:     record.RequestID,
		ServerID:      record.ServerID,
		AttemptNumber: record.AttemptNumber + 1,
		Status:        StatusRetrying,
		ResponseBody:  "",
		ErrorMessage:  "",
		RetryAt:       &retryAt,
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

	return created, nil
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
