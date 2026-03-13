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
	"basepro/backend/internal/sukumad/server"
	"basepro/backend/internal/sukumad/traceevent"
	"github.com/jackc/pgx/v5/pgconn"
)

type Service struct {
	repo         Repository
	auditService *audit.Service
	deliverySvc  interface {
		CreatePendingDelivery(context.Context, delivery.CreateInput) (delivery.Record, error)
		SubmitDHIS2Delivery(context.Context, delivery.DispatchInput) (delivery.Record, error)
	}
	serverSvc interface {
		GetServer(context.Context, int64) (server.Record, error)
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

func (s *Service) WithDeliveryService(deliverySvc interface {
	CreatePendingDelivery(context.Context, delivery.CreateInput) (delivery.Record, error)
	SubmitDHIS2Delivery(context.Context, delivery.DispatchInput) (delivery.Record, error)
}) *Service {
	s.deliverySvc = deliverySvc
	return s
}

func (s *Service) WithServerService(serverSvc interface {
	GetServer(context.Context, int64) (server.Record, error)
}) *Service {
	s.serverSvc = serverSvc
	return s
}

func (s *Service) WithEventWriter(eventWriter traceevent.Writer) *Service {
	s.eventWriter = eventWriter
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
	s.appendEvent(ctx, traceevent.WriteInput{
		RequestID:       &created.ID,
		EventType:       traceevent.EventRequestCreated,
		EventLevel:      "info",
		Message:         traceevent.Message("Request created", "Request %s created", created.UID),
		CorrelationID:   created.CorrelationID,
		Actor:           traceevent.Actor{Type: traceevent.ActorUser, UserID: input.ActorID},
		SourceComponent: "request.service",
		EventData: map[string]any{
			"requestUid":            created.UID,
			"status":                created.Status,
			"destinationServerId":   created.DestinationServerID,
			"destinationServerName": created.DestinationServerName,
			"sourceSystem":          created.SourceSystem,
			"awaitingAsync":         created.AwaitingAsync,
		},
	})

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

	if s.deliverySvc != nil {
		deliveryRecord, err := s.deliverySvc.CreatePendingDelivery(ctx, delivery.CreateInput{
			RequestID:     created.ID,
			ServerID:      created.DestinationServerID,
			CorrelationID: created.CorrelationID,
			ActorID:       input.ActorID,
		})
		if err != nil {
			return Record{}, err
		}
		s.appendEvent(ctx, traceevent.WriteInput{
			RequestID:         &created.ID,
			DeliveryAttemptID: &deliveryRecord.ID,
			EventType:         traceevent.EventRequestSubmitted,
			EventLevel:        "info",
			Message:           traceevent.Message("Request submitted to delivery", "Request %s submitted to delivery %s", created.UID, deliveryRecord.UID),
			CorrelationID:     created.CorrelationID,
			Actor:             traceevent.Actor{Type: traceevent.ActorUser, UserID: input.ActorID},
			SourceComponent:   "request.service",
			EventData: map[string]any{
				"requestUid":    created.UID,
				"deliveryUid":   deliveryRecord.UID,
				"deliveryState": deliveryRecord.Status,
			},
		})

		if s.serverSvc != nil {
			serverRecord, err := s.serverSvc.GetServer(ctx, created.DestinationServerID)
			if err != nil {
				return Record{}, err
			}
			if _, err := s.deliverySvc.SubmitDHIS2Delivery(ctx, delivery.DispatchInput{
				DeliveryID:    deliveryRecord.ID,
				RequestID:     created.ID,
				RequestUID:    created.UID,
				CorrelationID: created.CorrelationID,
				PayloadBody:   created.PayloadBody,
				URLSuffix:     created.URLSuffix,
				Server: delivery.ServerSnapshot{
					ID:           serverRecord.ID,
					Name:         serverRecord.Name,
					SystemType:   serverRecord.SystemType,
					BaseURL:      serverRecord.BaseURL,
					EndpointType: serverRecord.EndpointType,
					HTTPMethod:   serverRecord.HTTPMethod,
					UseAsync:     serverRecord.UseAsync,
					Headers:      serverRecord.Headers,
					URLParams:    serverRecord.URLParams,
				},
				ActorID: input.ActorID,
			}); err != nil {
				return Record{}, err
			}
			return s.GetRequest(ctx, created.ID)
		}
	}

	return created, nil
}

func (s *Service) SetProcessing(ctx context.Context, requestID int64) error {
	_, err := s.updateStatus(ctx, requestID, StatusProcessing)
	return err
}

func (s *Service) SetCompleted(ctx context.Context, requestID int64) error {
	_, err := s.updateStatus(ctx, requestID, StatusCompleted)
	return err
}

func (s *Service) SetFailed(ctx context.Context, requestID int64) error {
	_, err := s.updateStatus(ctx, requestID, StatusFailed)
	return err
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

func (s *Service) updateStatus(ctx context.Context, requestID int64, status string) (Record, error) {
	current, err := s.GetRequest(ctx, requestID)
	if err != nil {
		return Record{}, err
	}
	record, err := s.repo.UpdateRequestStatus(ctx, requestID, status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"request not found"}})
		}
		return Record{}, err
	}
	s.appendEvent(ctx, traceevent.WriteInput{
		RequestID:       &record.ID,
		EventType:       traceevent.EventRequestStatusChanged,
		EventLevel:      levelForRequestStatus(status),
		Message:         traceevent.Message("Request status changed", "Request %s moved from %s to %s", record.UID, current.Status, status),
		CorrelationID:   record.CorrelationID,
		Actor:           traceevent.Actor{Type: traceevent.ActorSystem},
		SourceComponent: "request.service",
		EventData: map[string]any{
			"requestUid": record.UID,
			"fromStatus": current.Status,
			"toStatus":   status,
		},
	})
	switch status {
	case StatusCompleted:
		s.appendEvent(ctx, traceevent.WriteInput{
			RequestID:       &record.ID,
			EventType:       traceevent.EventRequestCompleted,
			EventLevel:      "info",
			Message:         traceevent.Message("Request completed", "Request %s completed", record.UID),
			CorrelationID:   record.CorrelationID,
			Actor:           traceevent.Actor{Type: traceevent.ActorSystem},
			SourceComponent: "request.service",
			EventData:       map[string]any{"requestUid": record.UID},
		})
	case StatusFailed:
		s.appendEvent(ctx, traceevent.WriteInput{
			RequestID:       &record.ID,
			EventType:       traceevent.EventRequestFailed,
			EventLevel:      "error",
			Message:         traceevent.Message("Request failed", "Request %s failed", record.UID),
			CorrelationID:   record.CorrelationID,
			Actor:           traceevent.Actor{Type: traceevent.ActorSystem},
			SourceComponent: "request.service",
			EventData:       map[string]any{"requestUid": record.UID},
		})
	}
	return record, nil
}

func (s *Service) appendEvent(ctx context.Context, input traceevent.WriteInput) {
	if s.eventWriter == nil {
		return
	}
	_ = s.eventWriter.AppendEvent(ctx, input)
}

func levelForRequestStatus(status string) string {
	if status == StatusFailed {
		return "error"
	}
	return "info"
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
