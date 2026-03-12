package request

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/audit"
	"basepro/backend/internal/sukumad/delivery"
	"github.com/jackc/pgx/v5/pgconn"
)

type Service struct {
	repo         Repository
	auditService *audit.Service
	deliverySvc  interface {
		CreatePendingDelivery(context.Context, delivery.CreateInput) (delivery.Record, error)
	}
}

func NewService(repository Repository, auditService ...*audit.Service) *Service {
	var auditSvc *audit.Service
	if len(auditService) > 0 {
		auditSvc = auditService[0]
	}
	return &Service{repo: repository, auditService: auditSvc}
}

func (s *Service) WithDeliveryService(deliverySvc interface {
	CreatePendingDelivery(context.Context, delivery.CreateInput) (delivery.Record, error)
}) *Service {
	s.deliverySvc = deliverySvc
	return s
}

func (s *Service) ListRequests(ctx context.Context, query ListQuery) (ListResult, error) {
	return s.repo.ListRequests(ctx, query)
}

func (s *Service) GetRequest(ctx context.Context, id int64) (Record, error) {
	record, err := s.repo.GetRequestByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"request not found"}})
		}
		return Record{}, err
	}
	return record, nil
}

func (s *Service) CreateRequest(ctx context.Context, input CreateInput) (Record, error) {
	normalized, details := normalizeCreateInput(input)
	if len(details) > 0 {
		return Record{}, apperror.ValidationWithDetails("validation failed", details)
	}

	created, err := s.repo.CreateRequest(ctx, CreateParams{
		UID:                 newUID(),
		SourceSystem:        normalized.SourceSystem,
		DestinationServerID: normalized.DestinationServerID,
		BatchID:             normalized.BatchID,
		CorrelationID:       normalized.CorrelationID,
		IdempotencyKey:      normalized.IdempotencyKey,
		PayloadBody:         string(normalized.Payload),
		PayloadFormat:       "json",
		URLSuffix:           normalized.URLSuffix,
		Status:              StatusPending,
		Extras:              normalized.Extras,
		CreatedBy:           input.ActorID,
	})
	if err != nil {
		if mapped := mapConstraintError(err); mapped != nil {
			return Record{}, mapped
		}
		return Record{}, err
	}

	if s.deliverySvc != nil {
		if _, err := s.deliverySvc.CreatePendingDelivery(ctx, delivery.CreateInput{
			RequestID: created.ID,
			ServerID:  created.DestinationServerID,
			ActorID:   input.ActorID,
		}); err != nil {
			return Record{}, err
		}
	}

	s.logAudit(ctx, audit.Event{
		Action:      "request.created",
		ActorUserID: input.ActorID,
		EntityType:  "request",
		EntityID:    strPtr(fmt.Sprintf("%d", created.ID)),
		Metadata: map[string]any{
			"uid":                   created.UID,
			"status":                created.Status,
			"destinationServerId":   created.DestinationServerID,
			"destinationServerName": created.DestinationServerName,
			"correlationId":         created.CorrelationID,
		},
	})

	return created, nil
}

func normalizeCreateInput(input CreateInput) (CreateInput, map[string]any) {
	normalized := CreateInput{
		SourceSystem:        strings.TrimSpace(input.SourceSystem),
		DestinationServerID: input.DestinationServerID,
		BatchID:             strings.TrimSpace(input.BatchID),
		CorrelationID:       strings.TrimSpace(input.CorrelationID),
		IdempotencyKey:      strings.TrimSpace(input.IdempotencyKey),
		URLSuffix:           strings.TrimSpace(input.URLSuffix),
		Extras:              cloneExtras(input.Extras),
		ActorID:             input.ActorID,
	}

	details := map[string]any{}
	if normalized.DestinationServerID <= 0 {
		details["destinationServerId"] = []string{"is required"}
	}

	payload, err := normalizePayload(input.Payload)
	if err != nil {
		details["payload"] = []string{err.Error()}
	} else {
		normalized.Payload = payload
	}

	if err := validateExtras(normalized.Extras); err != nil {
		details["metadata"] = []string{err.Error()}
	}

	return normalized, details
}

func normalizePayload(input json.RawMessage) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(input)
	if len(trimmed) == 0 {
		return nil, errors.New("is required")
	}
	if !json.Valid(trimmed) {
		return nil, errors.New("must be valid JSON")
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, trimmed); err != nil {
		return nil, errors.New("must be valid JSON")
	}
	return json.RawMessage(compact.Bytes()), nil
}

func validateExtras(extras map[string]any) error {
	if extras == nil {
		return nil
	}
	if _, err := json.Marshal(extras); err != nil {
		return errors.New("must be valid JSON")
	}
	return nil
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
		if strings.Contains(pgErr.ConstraintName, "destination_server_id") {
			return apperror.ValidationWithDetails("validation failed", map[string]any{"destinationServerId": []string{"server not found"}})
		}
	case "23505":
		if strings.Contains(pgErr.ConstraintName, "uid") {
			return apperror.ValidationWithDetails("validation failed", map[string]any{"uid": []string{"must be unique"}})
		}
	}
	return nil
}

func strPtr(v string) *string {
	return &v
}
