package rapidex

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"basepro/backend/internal/apperror"
	sukumadserver "basepro/backend/internal/sukumad/server"
)

// IntegrationService coordinates the processing of RapidPro webhooks. It
// fetches the appropriate mapping configuration, transforms the webhook
// payload into a DHIS2 aggregate payload, and passes it to the Sukumad
// request engine.
type IntegrationService struct {
	mappingProvider MappingProvider
	// reporterSvc remains reserved for later reporter-based org unit override
	// work. The current milestone intentionally queues using the saved mapping.
	reporterSvc interface{}
	requestSvc  externalRequestCreator
	serverSvc   serverResolver
}

type MappingProvider interface {
	GetByFlowUUID(context.Context, string) (WebhookBinding, bool, error)
}

type externalRequestCreator interface {
	CreateExternalRequest(context.Context, ExternalRequestInput) error
}

type serverResolver interface {
	GetServerByCode(context.Context, string) (sukumadserver.Record, error)
}

// NewIntegrationService constructs a new service. Passing nil for reporterSvc
// is allowed. requestSvc and serverSvc must be configured for queueing.
func NewIntegrationService(provider MappingProvider, reporterSvc interface{}, requestSvc externalRequestCreator, serverSvc serverResolver) *IntegrationService {
	return &IntegrationService{
		mappingProvider: provider,
		reporterSvc:     reporterSvc,
		requestSvc:      requestSvc,
		serverSvc:       serverSvc,
	}
}

// ProcessWebhook processes a RapidPro webhook event. It looks up the mapping
// configuration by flow UUID, transforms the event into an AggregatePayload,
// validates the queueing prerequisites, and creates a Sukumad exchange request.
func (s *IntegrationService) ProcessWebhook(ctx context.Context, webhook RapidProWebhook) error {
	flowUUID := strings.TrimSpace(webhook.FlowUUID)
	if flowUUID == "" {
		return apperror.ValidationWithDetails("validation failed", map[string]any{
			"flow_uuid": []string{"is required"},
		})
	}

	binding, ok, err := s.mappingProvider.GetByFlowUUID(ctx, flowUUID)
	if err != nil {
		return fmt.Errorf("load mapping: %w", err)
	}
	if !ok {
		return apperror.ValidationWithDetails("validation failed", map[string]any{
			"flow_uuid": []string{fmt.Sprintf("no mapping configured for flow %s", flowUUID)},
		})
	}

	payload, err := MapToAggregate(webhook, binding.MappingConfig)
	if err != nil {
		return fmt.Errorf("failed to map webhook: %w", err)
	}

	if details := validateAggregatePayload(payload, binding); len(details) > 0 {
		return apperror.ValidationWithDetails("validation failed", details)
	}
	if s.requestSvc == nil {
		return fmt.Errorf("rapidex request service is not configured")
	}
	if s.serverSvc == nil {
		return fmt.Errorf("rapidex server lookup is not configured")
	}

	destinationServer, err := s.serverSvc.GetServerByCode(ctx, strings.TrimSpace(binding.DHIS2ServerCode))
	if err != nil {
		if err == sql.ErrNoRows {
			return apperror.ValidationWithDetails("validation failed", map[string]any{
				"dhis2ServerCode": []string{"server not found"},
			})
		}
		var appErr *apperror.AppError
		if errors.As(err, &appErr) && appErr.Code == apperror.CodeValidationFailed {
			return apperror.ValidationWithDetails("validation failed", map[string]any{
				"dhis2ServerCode": []string{"server not found"},
			})
		}
		return fmt.Errorf("resolve dhis2 server: %w", err)
	}

	if err := s.requestSvc.CreateExternalRequest(ctx, ExternalRequestInput{
		SourceSystem:         "rapidex-webhook",
		DestinationServerUID: destinationServer.UID,
		CorrelationID:        webhookCorrelationID(webhook),
		IdempotencyKey:       webhookIdempotencyKey(webhook),
		Payload:              payload,
		PayloadFormat:        "json",
		SubmissionBinding:    "body",
		URLSuffix:            "/api/dataValueSets",
		Extras:               webhookExtras(webhook, binding, payload),
	}); err != nil {
		return err
	}
	return nil
}

func validateAggregatePayload(payload AggregatePayload, binding WebhookBinding) map[string]any {
	details := map[string]any{}
	if strings.TrimSpace(binding.DHIS2ServerCode) == "" {
		details["dhis2ServerCode"] = []string{"is required in RapidEx webhook settings"}
	}
	if strings.TrimSpace(payload.DataSet) == "" {
		details["dataSet"] = []string{"is required"}
	}
	if strings.TrimSpace(payload.OrgUnit) == "" {
		details["orgUnit"] = []string{"is required"}
	}
	if strings.TrimSpace(payload.Period) == "" {
		details["period"] = []string{"is required"}
	}
	if len(payload.DataValues) == 0 {
		details["dataValues"] = []string{"at least one mapped data value is required"}
	}
	return details
}

func webhookCorrelationID(webhook RapidProWebhook) string {
	flowUUID := strings.TrimSpace(webhook.FlowUUID)
	contactUUID := strings.TrimSpace(webhook.Contact.UUID)
	if contactUUID != "" {
		return "rapidex:" + flowUUID + ":" + contactUUID
	}
	sum := sha256.Sum256(mustMarshalWebhook(webhook))
	return "rapidex:" + flowUUID + ":" + hex.EncodeToString(sum[:8])
}

func webhookIdempotencyKey(webhook RapidProWebhook) string {
	sum := sha256.Sum256(mustMarshalWebhook(webhook))
	return "rapidex-webhook:" + hex.EncodeToString(sum[:])
}

func webhookExtras(webhook RapidProWebhook, binding WebhookBinding, payload AggregatePayload) map[string]any {
	extras := map[string]any{
		"flowUuid":         strings.TrimSpace(webhook.FlowUUID),
		"flowName":         strings.TrimSpace(binding.MappingConfig.FlowName),
		"contactUuid":      strings.TrimSpace(webhook.Contact.UUID),
		"rapidProServer":   strings.TrimSpace(binding.RapidProServerCode),
		"dhis2Server":      strings.TrimSpace(binding.DHIS2ServerCode),
		"mappedDataset":    strings.TrimSpace(payload.DataSet),
		"mappedOrgUnit":    strings.TrimSpace(payload.OrgUnit),
		"mappedPeriod":     strings.TrimSpace(payload.Period),
		"mappedValueCount": len(payload.DataValues),
	}
	if msisdn := extractMSISDN(webhook); msisdn != "" {
		extras["msisdn"] = msisdn
	}
	return extras
}

func extractMSISDN(webhook RapidProWebhook) string {
	candidates := make([]string, 0, len(webhook.Contact.URNs)+1)
	if urn := strings.TrimSpace(webhook.Contact.URN); urn != "" {
		candidates = append(candidates, urn)
	}
	candidates = append(candidates, webhook.Contact.URNs...)
	for _, candidate := range candidates {
		value := strings.TrimSpace(candidate)
		if value == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(value), "tel:") {
			value = strings.TrimSpace(value[4:])
		}
		if strings.HasPrefix(value, "+") {
			return value
		}
	}
	return ""
}

func mustMarshalWebhook(webhook RapidProWebhook) []byte {
	payload, err := json.Marshal(webhook)
	if err != nil {
		return []byte("{}")
	}
	return payload
}
