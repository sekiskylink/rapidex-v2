package request

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/audit"
	"basepro/backend/internal/sukumad/delivery"
	sukumadserver "basepro/backend/internal/sukumad/server"
	"basepro/backend/internal/sukumad/traceevent"
	"github.com/jackc/pgx/v5/pgconn"
)

type Service struct {
	repo         Repository
	auditService *audit.Service
	serverSvc    interface {
		GetServerByUID(context.Context, string) (sukumadserver.Record, error)
	}
	deliverySvc interface {
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

func (s *Service) WithServerService(serverSvc interface {
	GetServerByUID(context.Context, string) (sukumadserver.Record, error)
}) *Service {
	s.serverSvc = serverSvc
	return s
}

func (s *Service) WithEventWriter(eventWriter traceevent.Writer) *Service {
	s.eventWriter = eventWriter
	return s
}

func (s *Service) ListRequests(ctx context.Context, query ListQuery) (ListResult, error) {
	result, err := s.repo.ListRequests(ctx, query)
	if err != nil {
		return ListResult{}, err
	}
	result.MetadataColumns = normalizeMetadataColumns(query.MetadataColumns)
	applyMetadataColumns(result.Items, result.MetadataColumns)
	return result, nil
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

func (s *Service) GetRequestByUID(ctx context.Context, uid string) (Record, error) {
	record, err := s.repo.GetRequestByUID(ctx, strings.TrimSpace(uid))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, apperror.ValidationWithDetails("validation failed", map[string]any{"uid": []string{"request not found"}})
		}
		return Record{}, err
	}
	return record, nil
}

func (s *Service) ListRequestsByBatchID(ctx context.Context, batchID string) ([]Record, error) {
	return s.repo.ListRequestsByBatchID(ctx, strings.TrimSpace(batchID))
}

func (s *Service) ListRequestsByCorrelationID(ctx context.Context, correlationID string) ([]Record, error) {
	return s.repo.ListRequestsByCorrelationID(ctx, strings.TrimSpace(correlationID))
}

func (s *Service) GetRequestBySourceSystemAndIdempotencyKey(ctx context.Context, sourceSystem string, idempotencyKey string) (Record, error) {
	record, err := s.repo.GetRequestBySourceSystemAndIdempotencyKey(ctx, strings.TrimSpace(sourceSystem), strings.TrimSpace(idempotencyKey))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, apperror.ValidationWithDetails("validation failed", map[string]any{"idempotencyKey": []string{"request not found"}})
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

	created, err := s.createRequestNormalized(ctx, normalized)
	if err != nil {
		if mapped := mapConstraintError(err); mapped != nil {
			return Record{}, mapped
		}
		return Record{}, err
	}
	return created, nil
}

func (s *Service) CreateExternalRequest(ctx context.Context, input ExternalCreateInput) (CreateResult, error) {
	normalized, details := s.normalizeExternalCreateInput(ctx, input)
	if len(details) > 0 {
		return CreateResult{}, apperror.ValidationWithDetails("validation failed", details)
	}

	if normalized.IdempotencyKey != "" {
		existing, err := s.repo.GetRequestBySourceSystemAndIdempotencyKey(ctx, normalized.SourceSystem, normalized.IdempotencyKey)
		if err == nil {
			return CreateResult{Record: existing, Deduped: true, Created: false}, nil
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return CreateResult{}, err
		}
	}

	created, err := s.createRequestNormalized(ctx, normalized)
	if err != nil {
		if normalized.IdempotencyKey != "" && isIdempotencyConstraintError(err) {
			existing, lookupErr := s.repo.GetRequestBySourceSystemAndIdempotencyKey(ctx, normalized.SourceSystem, normalized.IdempotencyKey)
			if lookupErr == nil {
				return CreateResult{Record: existing, Deduped: true, Created: false}, nil
			}
		}
		if mapped := mapConstraintError(err); mapped != nil {
			return CreateResult{}, mapped
		}
		return CreateResult{}, err
	}
	return CreateResult{Record: created, Created: true}, nil
}

func (s *Service) createRequestNormalized(ctx context.Context, normalized CreateInput) (Record, error) {
	created, err := s.repo.CreateRequest(ctx, CreateParams{
		UID:                     newUID(),
		SourceSystem:            normalized.SourceSystem,
		DestinationServerID:     normalized.DestinationServerID,
		BatchID:                 normalized.BatchID,
		CorrelationID:           normalized.CorrelationID,
		IdempotencyKey:          normalized.IdempotencyKey,
		PayloadBody:             string(normalized.Payload.([]byte)),
		PayloadFormat:           normalized.PayloadFormat,
		SubmissionBinding:       normalized.SubmissionBinding,
		ResponseBodyPersistence: normalized.ResponseBodyPersistence,
		URLSuffix:               normalized.URLSuffix,
		Status:                  StatusPending,
		Extras:                  normalized.Extras,
		CreatedBy:               normalized.ActorID,
	})
	if err != nil {
		return Record{}, err
	}
	s.appendEvent(ctx, traceevent.WriteInput{
		RequestID:       &created.ID,
		EventType:       traceevent.EventRequestCreated,
		EventLevel:      "info",
		Message:         traceevent.Message("Request created", "Request %s created", created.UID),
		CorrelationID:   created.CorrelationID,
		Actor:           traceevent.Actor{Type: traceevent.ActorUser, UserID: normalized.ActorID},
		SourceComponent: "request.service",
		EventData: map[string]any{
			"requestUid":              created.UID,
			"status":                  created.Status,
			"destinationServerId":     created.DestinationServerID,
			"destinationServerName":   created.DestinationServerName,
			"sourceSystem":            created.SourceSystem,
			"awaitingAsync":           created.AwaitingAsync,
			"responseBodyPersistence": created.ResponseBodyPersistence,
		},
	})

	s.logAudit(ctx, audit.Event{
		Action:      "request.created",
		ActorUserID: normalized.ActorID,
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
			Actor:           traceevent.Actor{Type: traceevent.ActorUser, UserID: normalized.ActorID},
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
				Actor:           traceevent.Actor{Type: traceevent.ActorUser, UserID: normalized.ActorID},
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
				ActorID:       normalized.ActorID,
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
				Actor:             traceevent.Actor{Type: traceevent.ActorUser, UserID: normalized.ActorID},
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
					Actor:             traceevent.Actor{Type: traceevent.ActorUser, UserID: normalized.ActorID},
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

func (s *Service) normalizeExternalCreateInput(ctx context.Context, input ExternalCreateInput) (CreateInput, map[string]any) {
	normalized := ExternalCreateInput{
		SourceSystem:          strings.TrimSpace(input.SourceSystem),
		DestinationServerUID:  strings.TrimSpace(input.DestinationServerUID),
		DestinationServerUIDs: normalizeUIDs(input.DestinationServerUIDs),
		DependencyRequestUIDs: normalizeUIDs(input.DependencyRequestUIDs),
		BatchID:               strings.TrimSpace(input.BatchID),
		CorrelationID:         strings.TrimSpace(input.CorrelationID),
		IdempotencyKey:        strings.TrimSpace(input.IdempotencyKey),
		Payload:               input.Payload,
		PayloadFormat:         input.PayloadFormat,
		SubmissionBinding:     input.SubmissionBinding,
		URLSuffix:             strings.TrimSpace(input.URLSuffix),
		Extras:                cloneExtras(input.Extras),
		ActorID:               input.ActorID,
	}

	details := map[string]any{}
	if normalized.SourceSystem == "" {
		details["sourceSystem"] = []string{"is required"}
	}
	if normalized.DestinationServerUID == "" {
		details["destinationServerUid"] = []string{"is required"}
	}
	if s.serverSvc == nil {
		details["destinationServerUid"] = []string{"server lookup is not configured"}
	}
	if err := validateExtras(normalized.Extras); err != nil {
		details["metadata"] = []string{err.Error()}
	}
	if len(details) > 0 {
		return CreateInput{}, details
	}

	primaryServer, err := s.serverSvc.GetServerByUID(ctx, normalized.DestinationServerUID)
	if err != nil {
		details["destinationServerUid"] = []string{"server not found"}
		return CreateInput{}, details
	}

	destinationIDs := make([]int64, 0, len(normalized.DestinationServerUIDs))
	for _, uid := range normalized.DestinationServerUIDs {
		serverRecord, lookupErr := s.serverSvc.GetServerByUID(ctx, uid)
		if lookupErr != nil {
			details["destinationServerUids"] = []string{"must reference valid server uids"}
			return CreateInput{}, details
		}
		destinationIDs = append(destinationIDs, serverRecord.ID)
	}

	dependencyIDs := make([]int64, 0, len(normalized.DependencyRequestUIDs))
	for _, uid := range normalized.DependencyRequestUIDs {
		record, lookupErr := s.repo.GetRequestByUID(ctx, uid)
		if lookupErr != nil {
			details["dependencyRequestUids"] = []string{"must reference valid request uids"}
			return CreateInput{}, details
		}
		dependencyIDs = append(dependencyIDs, record.ID)
	}

	internal := CreateInput{
		SourceSystem:            normalized.SourceSystem,
		DestinationServerID:     primaryServer.ID,
		DestinationServerIDs:    destinationIDs,
		DependencyRequestIDs:    dependencyIDs,
		BatchID:                 normalized.BatchID,
		CorrelationID:           normalized.CorrelationID,
		IdempotencyKey:          normalized.IdempotencyKey,
		Payload:                 normalized.Payload,
		PayloadFormat:           normalized.PayloadFormat,
		SubmissionBinding:       normalized.SubmissionBinding,
		ResponseBodyPersistence: normalized.ResponseBodyPersistence,
		URLSuffix:               normalized.URLSuffix,
		Extras:                  normalized.Extras,
		ActorID:                 normalized.ActorID,
	}

	internal, validationDetails := normalizeCreateInput(internal)
	for key, value := range validationDetails {
		details[key] = value
	}
	if len(details) > 0 {
		return CreateInput{}, details
	}
	return internal, nil
}

func normalizeUIDs(input []string) []string {
	items := make([]string, 0, len(input))
	seen := map[string]struct{}{}
	for _, raw := range input {
		uid := strings.TrimSpace(raw)
		if uid == "" {
			continue
		}
		if _, ok := seen[uid]; ok {
			continue
		}
		seen[uid] = struct{}{}
		items = append(items, uid)
	}
	return items
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
		SourceSystem:            strings.TrimSpace(input.SourceSystem),
		DestinationServerID:     input.DestinationServerID,
		DestinationServerIDs:    append([]int64{}, input.DestinationServerIDs...),
		DependencyRequestIDs:    append([]int64{}, input.DependencyRequestIDs...),
		BatchID:                 strings.TrimSpace(input.BatchID),
		CorrelationID:           strings.TrimSpace(input.CorrelationID),
		IdempotencyKey:          strings.TrimSpace(input.IdempotencyKey),
		PayloadFormat:           normalizePayloadFormat(input.PayloadFormat),
		SubmissionBinding:       normalizeSubmissionBinding(input.SubmissionBinding),
		ResponseBodyPersistence: normalizeResponseBodyPersistence(input.ResponseBodyPersistence, true),
		URLSuffix:               strings.TrimSpace(input.URLSuffix),
		Extras:                  cloneExtras(input.Extras),
		ActorID:                 input.ActorID,
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

	payload, err := normalizePayload(input.Payload, normalized.PayloadFormat, normalized.SubmissionBinding)
	if err != nil {
		details["payload"] = []string{err.Error()}
	} else {
		normalized.Payload = payload
	}

	if err := validateExtras(normalized.Extras); err != nil {
		details["metadata"] = []string{err.Error()}
	}
	if !isValidResponseBodyPersistence(normalized.ResponseBodyPersistence, true) {
		details["responseBodyPersistence"] = []string{"must be one of server default, filter, save, or discard"}
	}

	return normalized, details
}

func (s *Service) DeleteRequest(ctx context.Context, actorID *int64, id int64) error {
	existing, err := s.GetRequest(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteRequest(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"request not found"}})
		}
		return err
	}
	s.logAudit(ctx, audit.Event{
		Action:      "request.deleted",
		ActorUserID: actorID,
		EntityType:  "request",
		EntityID:    strPtr(fmt.Sprintf("%d", id)),
		Metadata: map[string]any{
			"uid":                   existing.UID,
			"status":                existing.Status,
			"destinationServerId":   existing.DestinationServerID,
			"destinationServerName": existing.DestinationServerName,
			"correlationId":         existing.CorrelationID,
		},
	})
	return nil
}

func normalizePayloadFormat(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", PayloadFormatJSON:
		return PayloadFormatJSON
	case PayloadFormatText:
		return PayloadFormatText
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func normalizeSubmissionBinding(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", SubmissionBindingBody:
		return SubmissionBindingBody
	case SubmissionBindingQuery:
		return SubmissionBindingQuery
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func normalizeResponseBodyPersistence(value string, allowDefault bool) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "server_default" || normalized == "server-default" || normalized == "default" {
		return ""
	}
	if normalized == "" && !allowDefault {
		return "filter"
	}
	return normalized
}

func isValidResponseBodyPersistence(value string, allowDefault bool) bool {
	switch value {
	case "filter", "save", "discard":
		return true
	case "":
		return allowDefault
	default:
		return false
	}
}

func normalizePayload(input any, payloadFormat string, submissionBinding string) ([]byte, error) {
	if payloadFormat != PayloadFormatJSON && payloadFormat != PayloadFormatText {
		return nil, errors.New("payloadFormat must be one of json or text")
	}
	if submissionBinding != SubmissionBindingBody && submissionBinding != SubmissionBindingQuery {
		return nil, errors.New("submissionBinding must be one of body or query")
	}
	if payloadFormat == PayloadFormatJSON {
		return normalizeJSONPayload(input, submissionBinding)
	}
	return normalizeTextPayload(input, submissionBinding)
}

func normalizeJSONPayload(input any, submissionBinding string) ([]byte, error) {
	encoded, err := marshalPayloadInput(input)
	if err != nil {
		return nil, errors.New("must be valid JSON")
	}
	trimmed := bytes.TrimSpace(encoded)
	if len(trimmed) == 0 {
		return nil, errors.New("is required")
	}
	if !json.Valid(trimmed) {
		return nil, errors.New("must be valid JSON")
	}
	if submissionBinding == SubmissionBindingQuery {
		if err := validateJSONQueryPayload(trimmed); err != nil {
			return nil, err
		}
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, trimmed); err != nil {
		return nil, errors.New("must be valid JSON")
	}
	return compact.Bytes(), nil
}

func normalizeTextPayload(input any, submissionBinding string) ([]byte, error) {
	value, ok := input.(string)
	if !ok {
		return nil, errors.New("must be a text value")
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, errors.New("is required")
	}
	if submissionBinding == SubmissionBindingQuery {
		if _, err := url.ParseQuery(trimmed); err != nil {
			return nil, errors.New("must be a valid query string")
		}
	}
	return []byte(trimmed), nil
}

func marshalPayloadInput(input any) ([]byte, error) {
	switch value := input.(type) {
	case nil:
		return nil, nil
	case []byte:
		return value, nil
	default:
		return json.Marshal(value)
	}
}

func validateJSONQueryPayload(payload []byte) error {
	var parsed map[string]any
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return errors.New("must be a JSON object when sent as query params")
	}
	if len(parsed) == 0 {
		return errors.New("must include at least one query param")
	}
	for key, value := range parsed {
		if strings.TrimSpace(key) == "" {
			return errors.New("query param names must be non-empty")
		}
		if !isQueryParamValue(value) {
			return errors.New("query param values must be strings, numbers, booleans, null, or arrays of those values")
		}
	}
	return nil
}

func isQueryParamValue(value any) bool {
	switch typed := value.(type) {
	case nil, string, bool, float64:
		return true
	case []any:
		for _, item := range typed {
			if !isQueryParamScalar(item) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func isQueryParamScalar(value any) bool {
	switch value.(type) {
	case nil, string, bool, float64:
		return true
	default:
		return false
	}
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
		if strings.Contains(pgErr.ConstraintName, "idempotency") {
			return apperror.ValidationWithDetails("validation failed", map[string]any{"idempotencyKey": []string{"must be unique within the source system"}})
		}
	}
	return nil
}

func isIdempotencyConstraintError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && strings.Contains(pgErr.ConstraintName, "idempotency")
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
	updated.Payload = clonePayloadValue(updated.Payload)
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
