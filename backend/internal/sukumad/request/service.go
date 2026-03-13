package request

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/audit"
	"basepro/backend/internal/sukumad/delivery"
	"basepro/backend/internal/sukumad/traceevent"
	"github.com/jackc/pgx/v5/pgconn"
)

type Service struct {
	repo         Repository
	auditService *audit.Service
	deliverySvc  interface {
		CreatePendingDelivery(context.Context, delivery.CreateInput) (delivery.Record, error)
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
}) *Service {
	s.deliverySvc = deliverySvc
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

	targetIDs := normalizeDestinationIDs(normalized.DestinationServerID, normalized.DestinationServerIDs)
	targetParams := make([]CreateTargetParams, 0, len(targetIDs))
	for index, targetID := range targetIDs {
		targetKind := "cc"
		if index == 0 {
			targetKind = "primary"
		}
		targetParams = append(targetParams, CreateTargetParams{
			UID:        newUID(),
			ServerID:   targetID,
			TargetKind: targetKind,
			Priority:   index + 1,
			Status:     TargetStatusPending,
		})
	}
	createdTargets, err := s.repo.CreateTargets(ctx, created.ID, targetParams)
	if err != nil {
		return Record{}, err
	}
	if err := s.repo.CreateDependencies(ctx, created.ID, normalized.DependencyRequestIDs); err != nil {
		return Record{}, err
	}
	for _, target := range createdTargets {
		s.appendEvent(ctx, traceevent.WriteInput{
			RequestID:       &created.ID,
			EventType:       "request.target.created",
			EventLevel:      "info",
			Message:         traceevent.Message("Request target created", "Request %s target %s created", created.UID, target.UID),
			CorrelationID:   created.CorrelationID,
			Actor:           traceevent.Actor{Type: traceevent.ActorUser, UserID: input.ActorID},
			SourceComponent: "request.service",
			EventData: map[string]any{
				"requestUid": created.UID,
				"targetUid":  target.UID,
				"serverId":   target.ServerID,
				"serverCode": target.ServerCode,
				"targetKind": target.TargetKind,
				"status":     target.Status,
			},
		})
	}

	dependencyStatuses, err := s.repo.GetDependencyStatuses(ctx, created.ID)
	if err != nil {
		return Record{}, err
	}
	blockedByDependency := false
	for _, dependency := range dependencyStatuses {
		if dependency.Status == StatusFailed {
			if err := s.SetFailed(ctx, created.ID); err != nil {
				return Record{}, err
			}
			s.appendEvent(ctx, traceevent.WriteInput{
				RequestID:       &created.ID,
				EventType:       "request.failed.dependency",
				EventLevel:      "warning",
				Message:         traceevent.Message("Request failed by dependency", "Request %s failed because dependency %s failed", created.UID, dependency.RequestUID),
				CorrelationID:   created.CorrelationID,
				Actor:           traceevent.Actor{Type: traceevent.ActorUser, UserID: input.ActorID},
				SourceComponent: "request.service",
				EventData: map[string]any{
					"requestUid":        created.UID,
					"dependencyRequest": dependency.RequestUID,
				},
			})
			return s.GetRequest(ctx, created.ID)
		}
		if dependency.Status != StatusCompleted {
			blockedByDependency = true
		}
	}

	if s.deliverySvc != nil {
		for _, targetID := range targetIDs {
			deliveryRecord, err := s.deliverySvc.CreatePendingDelivery(ctx, delivery.CreateInput{
				RequestID:     created.ID,
				ServerID:      targetID,
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
					"serverId":      targetID,
				},
			})

			if blockedByDependency {
				if _, err := s.updateTargetAndRollup(ctx, UpdateTargetParams{
					RequestID:     created.ID,
					ServerID:      targetID,
					Status:        TargetStatusBlocked,
					BlockedReason: "dependency_blocked",
				}); err != nil {
					return Record{}, err
				}
				s.appendEvent(ctx, traceevent.WriteInput{
					RequestID:         &created.ID,
					DeliveryAttemptID: &deliveryRecord.ID,
					EventType:         "request.blocked.dependency",
					EventLevel:        "info",
					Message:           traceevent.Message("Request blocked by dependency", "Request %s is waiting on dependencies", created.UID),
					CorrelationID:     created.CorrelationID,
					Actor:             traceevent.Actor{Type: traceevent.ActorUser, UserID: input.ActorID},
					SourceComponent:   "request.service",
					EventData:         map[string]any{"requestUid": created.UID},
				})
				continue
			}
		}
		return s.GetRequest(ctx, created.ID)
	}

	return created, nil
}

func (s *Service) SetProcessing(ctx context.Context, requestID int64) error {
	if record, err := s.GetRequest(ctx, requestID); err == nil && len(record.Targets) > 0 {
		_, err = s.reconcileTargetRollup(ctx, requestID)
		return err
	}
	_, err := s.updateStatus(ctx, requestID, StatusProcessing, "", nil)
	return err
}

func (s *Service) SetCompleted(ctx context.Context, requestID int64) error {
	if record, err := s.GetRequest(ctx, requestID); err == nil && len(record.Targets) > 0 {
		_, err = s.reconcileTargetRollup(ctx, requestID)
		return err
	}
	_, err := s.updateStatus(ctx, requestID, StatusCompleted, "", nil)
	return err
}

func (s *Service) SetFailed(ctx context.Context, requestID int64) error {
	if record, err := s.GetRequest(ctx, requestID); err == nil && len(record.Targets) > 0 {
		_, err = s.reconcileTargetRollup(ctx, requestID)
		return err
	}
	_, err := s.updateStatus(ctx, requestID, StatusFailed, "", nil)
	return err
}

func (s *Service) SetBlocked(ctx context.Context, requestID int64, reason string, deferredUntil *time.Time) error {
	if record, err := s.GetRequest(ctx, requestID); err == nil && len(record.Targets) > 0 {
		_, err = s.reconcileTargetRollup(ctx, requestID)
		return err
	}
	_, err := s.updateStatus(ctx, requestID, StatusBlocked, reason, deferredUntil)
	return err
}

func normalizeCreateInput(input CreateInput) (CreateInput, map[string]any) {
	normalized := CreateInput{
		SourceSystem:         strings.TrimSpace(input.SourceSystem),
		DestinationServerID:  input.DestinationServerID,
		DestinationServerIDs: append([]int64{}, input.DestinationServerIDs...),
		DependencyRequestIDs: append([]int64{}, input.DependencyRequestIDs...),
		BatchID:              strings.TrimSpace(input.BatchID),
		CorrelationID:        strings.TrimSpace(input.CorrelationID),
		IdempotencyKey:       strings.TrimSpace(input.IdempotencyKey),
		URLSuffix:            strings.TrimSpace(input.URLSuffix),
		Extras:               cloneExtras(input.Extras),
		ActorID:              input.ActorID,
	}

	details := map[string]any{}
	if normalized.DestinationServerID <= 0 {
		details["destinationServerId"] = []string{"is required"}
	}
	for _, destinationID := range normalizeDestinationIDs(normalized.DestinationServerID, normalized.DestinationServerIDs) {
		if destinationID <= 0 {
			details["destinationServerIds"] = []string{"must contain valid server ids"}
			break
		}
	}
	for _, dependencyID := range normalized.DependencyRequestIDs {
		if dependencyID <= 0 {
			details["dependencyRequestIds"] = []string{"must contain valid request ids"}
			break
		}
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

func normalizeDestinationIDs(primary int64, extra []int64) []int64 {
	items := make([]int64, 0, len(extra)+1)
	seen := map[int64]struct{}{}
	if primary > 0 {
		seen[primary] = struct{}{}
		items = append(items, primary)
	}
	for _, item := range extra {
		if item <= 0 {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		items = append(items, item)
	}
	return items
}

func (s *Service) logAudit(ctx context.Context, event audit.Event) {
	if s.auditService == nil {
		return
	}
	_ = s.auditService.Log(ctx, event)
}

func (s *Service) updateStatus(ctx context.Context, requestID int64, status string, reason string, deferredUntil *time.Time) (Record, error) {
	current, err := s.GetRequest(ctx, requestID)
	if err != nil {
		return Record{}, err
	}
	record, err := s.repo.UpdateRequestStatus(ctx, requestID, status, reason, deferredUntil)
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
			"reason":     reason,
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

func (s *Service) SetTargetPending(ctx context.Context, requestID int64, serverID int64) error {
	_, err := s.updateTargetAndRollup(ctx, UpdateTargetParams{
		RequestID:     requestID,
		ServerID:      serverID,
		Status:        TargetStatusPending,
		BlockedReason: "",
	})
	return err
}

func (s *Service) SetTargetBlocked(ctx context.Context, requestID int64, serverID int64, reason string, deferredUntil *time.Time) error {
	_, err := s.updateTargetAndRollup(ctx, UpdateTargetParams{
		RequestID:     requestID,
		ServerID:      serverID,
		Status:        TargetStatusBlocked,
		BlockedReason: reason,
		DeferredUntil: deferredUntil,
	})
	return err
}

func (s *Service) SetTargetProcessing(ctx context.Context, requestID int64, serverID int64) error {
	_, err := s.updateTargetAndRollup(ctx, UpdateTargetParams{
		RequestID:     requestID,
		ServerID:      serverID,
		Status:        TargetStatusProcessing,
		BlockedReason: "",
		DeferredUntil: nil,
	})
	return err
}

func (s *Service) SetTargetSucceeded(ctx context.Context, requestID int64, serverID int64) error {
	_, err := s.updateTargetAndRollup(ctx, UpdateTargetParams{
		RequestID:     requestID,
		ServerID:      serverID,
		Status:        TargetStatusSucceeded,
		BlockedReason: "",
		DeferredUntil: nil,
	})
	return err
}

func (s *Service) SetTargetFailed(ctx context.Context, requestID int64, serverID int64, reason string) error {
	_, err := s.updateTargetAndRollup(ctx, UpdateTargetParams{
		RequestID:     requestID,
		ServerID:      serverID,
		Status:        TargetStatusFailed,
		BlockedReason: reason,
		DeferredUntil: nil,
	})
	return err
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

func (s *Service) updateTargetAndRollup(ctx context.Context, params UpdateTargetParams) (Record, error) {
	target, err := s.repo.UpdateTarget(ctx, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, apperror.ValidationWithDetails("validation failed", map[string]any{"target": []string{"request target not found"}})
		}
		return Record{}, err
	}
	requestRecord, err := s.reconcileTargetRollup(ctx, params.RequestID)
	if err != nil {
		return Record{}, err
	}

	eventType := "request.rollup.updated"
	eventLevel := "info"
	switch target.Status {
	case TargetStatusSucceeded:
		eventType = "request.target.completed"
	case TargetStatusFailed:
		eventType = "request.target.failed"
		eventLevel = "error"
	case TargetStatusBlocked:
		eventType = "request.target.blocked"
	case TargetStatusProcessing:
		eventType = "request.target.processing"
	}
	s.appendEvent(ctx, traceevent.WriteInput{
		RequestID:       &requestRecord.ID,
		EventType:       eventType,
		EventLevel:      eventLevel,
		Message:         traceevent.Message("Request target updated", "Request %s target %s moved to %s", requestRecord.UID, target.UID, target.Status),
		CorrelationID:   requestRecord.CorrelationID,
		Actor:           traceevent.Actor{Type: traceevent.ActorSystem},
		SourceComponent: "request.service",
		EventData: map[string]any{
			"requestUid":      requestRecord.UID,
			"targetUid":       target.UID,
			"serverId":        target.ServerID,
			"serverCode":      target.ServerCode,
			"targetStatus":    target.Status,
			"blockedReason":   target.BlockedReason,
			"deferredUntil":   target.DeferredUntil,
			"requestRollup":   requestRecord.Status,
			"requestReason":   requestRecord.StatusReason,
			"requestDeferred": requestRecord.DeferredUntil,
		},
	})
	return requestRecord, nil
}

func (s *Service) reconcileTargetRollup(ctx context.Context, requestID int64) (Record, error) {
	requestRecord, err := s.GetRequest(ctx, requestID)
	if err != nil {
		return Record{}, err
	}
	if len(requestRecord.Targets) == 0 {
		return requestRecord, nil
	}

	nextStatus, nextReason, nextDeferredUntil := deriveRollup(requestRecord.Targets)
	if requestRecord.Status == nextStatus && requestRecord.StatusReason == nextReason && sameTimePtr(requestRecord.DeferredUntil, nextDeferredUntil) {
		return requestRecord, nil
	}

	updated, err := s.repo.UpdateRequestStatus(ctx, requestID, nextStatus, nextReason, nextDeferredUntil)
	if err != nil {
		return Record{}, err
	}
	updated.Targets = cloneTargets(requestRecord.Targets)
	updated.Dependencies = cloneDependencies(requestRecord.Dependencies)
	updated.Payload = append(json.RawMessage(nil), updated.Payload...)
	updated.Extras = cloneExtras(updated.Extras)
	s.appendEvent(ctx, traceevent.WriteInput{
		RequestID:       &updated.ID,
		EventType:       "request.rollup.updated",
		EventLevel:      levelForRequestStatus(updated.Status),
		Message:         traceevent.Message("Request roll-up updated", "Request %s roll-up is now %s", updated.UID, updated.Status),
		CorrelationID:   updated.CorrelationID,
		Actor:           traceevent.Actor{Type: traceevent.ActorSystem},
		SourceComponent: "request.service",
		EventData: map[string]any{
			"requestUid":    updated.UID,
			"status":        updated.Status,
			"reason":        updated.StatusReason,
			"deferredUntil": updated.DeferredUntil,
		},
	})
	if updated.Status == StatusCompleted || updated.Status == StatusFailed {
		if err := s.reconcileDependents(ctx, updated); err != nil {
			return Record{}, err
		}
	}
	return updated, nil
}

func (s *Service) reconcileDependents(ctx context.Context, dependency Record) error {
	dependents, err := s.repo.ListDependents(ctx, dependency.ID)
	if err != nil {
		return err
	}
	seen := make(map[int64]struct{}, len(dependents))
	for _, dependent := range dependents {
		if dependent.RequestID <= 0 {
			continue
		}
		if _, ok := seen[dependent.RequestID]; ok {
			continue
		}
		seen[dependent.RequestID] = struct{}{}
		if err := s.reevaluateDependencyState(ctx, dependency, dependent.RequestID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) reevaluateDependencyState(ctx context.Context, dependency Record, requestID int64) error {
	record, err := s.GetRequest(ctx, requestID)
	if err != nil {
		return err
	}
	if record.Status == StatusCompleted || record.Status == StatusFailed {
		return nil
	}

	dependencyStatuses, err := s.repo.GetDependencyStatuses(ctx, requestID)
	if err != nil {
		return err
	}
	for _, status := range dependencyStatuses {
		if status.Status == StatusFailed {
			return s.failBlockedRequestByDependency(ctx, record, status.RequestUID)
		}
		if status.Status != StatusCompleted {
			return nil
		}
	}

	return s.releaseDependencyBlockedRequest(ctx, record, dependency.UID)
}

func (s *Service) failBlockedRequestByDependency(ctx context.Context, record Record, failedDependencyUID string) error {
	for _, target := range record.Targets {
		if target.Status == TargetStatusSucceeded || target.Status == TargetStatusFailed {
			continue
		}
		if _, err := s.updateTargetAndRollup(ctx, UpdateTargetParams{
			RequestID:     record.ID,
			ServerID:      target.ServerID,
			Status:        TargetStatusFailed,
			BlockedReason: "dependency_failed",
		}); err != nil {
			return err
		}
	}

	s.appendEvent(ctx, traceevent.WriteInput{
		RequestID:       &record.ID,
		EventType:       "request.failed.dependency",
		EventLevel:      "warning",
		Message:         traceevent.Message("Request failed by dependency", "Request %s failed because dependency %s failed", record.UID, failedDependencyUID),
		CorrelationID:   record.CorrelationID,
		Actor:           traceevent.Actor{Type: traceevent.ActorSystem},
		SourceComponent: "request.service",
		EventData: map[string]any{
			"requestUid":        record.UID,
			"dependencyRequest": failedDependencyUID,
		},
	})
	return nil
}

func (s *Service) releaseDependencyBlockedRequest(ctx context.Context, record Record, dependencyUID string) error {
	releasedAt := time.Now().UTC()
	releasedTargets := 0
	for _, target := range record.Targets {
		if target.Status != TargetStatusBlocked || target.BlockedReason != "dependency_blocked" {
			continue
		}
		releasedTargets++
		if _, err := s.updateTargetAndRollup(ctx, UpdateTargetParams{
			RequestID:      record.ID,
			ServerID:       target.ServerID,
			Status:         TargetStatusPending,
			BlockedReason:  "",
			DeferredUntil:  nil,
			LastReleasedAt: &releasedAt,
		}); err != nil {
			return err
		}
	}
	if releasedTargets == 0 {
		return nil
	}

	s.appendEvent(ctx, traceevent.WriteInput{
		RequestID:       &record.ID,
		EventType:       "request.unblocked.dependency",
		EventLevel:      "info",
		Message:         traceevent.Message("Request unblocked by dependency", "Request %s was released after dependencies completed", record.UID),
		CorrelationID:   record.CorrelationID,
		Actor:           traceevent.Actor{Type: traceevent.ActorSystem},
		SourceComponent: "request.service",
		EventData: map[string]any{
			"requestUid":        record.UID,
			"releasedTargets":   releasedTargets,
			"dependencyRequest": dependencyUID,
		},
	})
	return nil
}

func deriveRollup(targets []TargetRecord) (string, string, *time.Time) {
	allSucceeded := len(targets) > 0
	hasBlocked := false
	var blockedReason string
	var deferredUntil *time.Time

	for _, target := range targets {
		switch target.Status {
		case TargetStatusFailed:
			return StatusFailed, firstTargetReason(target), nil
		case TargetStatusProcessing:
			return StatusProcessing, "", nil
		case TargetStatusBlocked:
			hasBlocked = true
			if blockedReason == "" {
				blockedReason = target.BlockedReason
			}
			if deferredUntil == nil || earlierTime(target.DeferredUntil, deferredUntil) {
				deferredUntil = cloneTimePtr(target.DeferredUntil)
			}
			allSucceeded = false
		case TargetStatusPending:
			allSucceeded = false
			if blockedReason == "" && target.BlockedReason != "" {
				blockedReason = target.BlockedReason
			}
			if deferredUntil == nil || earlierTime(target.DeferredUntil, deferredUntil) {
				deferredUntil = cloneTimePtr(target.DeferredUntil)
			}
		case TargetStatusSucceeded:
		default:
			allSucceeded = false
		}
	}

	if allSucceeded {
		return StatusCompleted, "", nil
	}
	if hasBlocked || blockedReason != "" || deferredUntil != nil {
		return StatusBlocked, blockedReason, cloneTimePtr(deferredUntil)
	}
	return StatusPending, "", nil
}

func firstTargetReason(target TargetRecord) string {
	return strings.TrimSpace(target.BlockedReason)
}

func sameTimePtr(left *time.Time, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

func earlierTime(candidate *time.Time, current *time.Time) bool {
	if candidate == nil {
		return false
	}
	if current == nil {
		return true
	}
	return candidate.Before(*current)
}
