package rapidex

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"basepro/backend/internal/apperror"
	sukumadserver "basepro/backend/internal/sukumad/server"
)

type fakeMappingProvider struct {
	binding WebhookBinding
	ok      bool
	err     error
}

func (f fakeMappingProvider) GetByFlowUUID(context.Context, string) (WebhookBinding, bool, error) {
	return f.binding, f.ok, f.err
}

type fakeExternalRequestCreator struct {
	calls     []ExternalRequestInput
	seenKeys  map[string]struct{}
	created   int
	deduped   int
	lastKey   string
	returnErr error
}

func (f *fakeExternalRequestCreator) CreateExternalRequest(_ context.Context, input ExternalRequestInput) error {
	if f.returnErr != nil {
		return f.returnErr
	}
	f.calls = append(f.calls, input)
	if f.seenKeys == nil {
		f.seenKeys = map[string]struct{}{}
	}
	f.lastKey = input.IdempotencyKey
	if _, ok := f.seenKeys[input.IdempotencyKey]; ok {
		f.deduped++
		return nil
	}
	f.seenKeys[input.IdempotencyKey] = struct{}{}
	f.created++
	return nil
}

type fakeServerResolver struct {
	record sukumadserver.Record
	err    error
	code   string
}

func (f *fakeServerResolver) GetServerByCode(_ context.Context, code string) (sukumadserver.Record, error) {
	f.code = code
	if f.err != nil {
		return sukumadserver.Record{}, f.err
	}
	return f.record, nil
}

func TestProcessWebhookQueuesAggregatePayload(t *testing.T) {
	requestCreator := &fakeExternalRequestCreator{}
	serverResolver := &fakeServerResolver{record: sukumadserver.Record{UID: "dhis2-uid"}}
	service := NewIntegrationService(fakeMappingProvider{
		ok: true,
		binding: WebhookBinding{
			MappingConfig: MappingConfig{
				FlowUUID:   "flow-1",
				FlowName:   "Weekly",
				Dataset:    "ds-1",
				OrgUnitVar: "facility",
				PeriodVar:  "period",
				Mappings: []DataValueMapping{
					{Field: "value_a", DataElement: "de-1", CategoryOptionCombo: "coc-1"},
				},
			},
			RapidProServerCode: "rapidpro-main",
			DHIS2ServerCode:    "dhis2-main",
		},
	}, nil, requestCreator, serverResolver)

	webhook := RapidProWebhook{
		FlowUUID: "flow-1",
		Results: map[string]interface{}{
			"facility": "OU_123",
			"period":   "202604",
			"value_a":  "17",
		},
	}
	webhook.Contact.UUID = "contact-1"
	webhook.Contact.URNs = []string{"tel:+256782820208"}

	if err := service.ProcessWebhook(context.Background(), webhook); err != nil {
		t.Fatalf("process webhook: %v", err)
	}
	if len(requestCreator.calls) != 1 {
		t.Fatalf("expected 1 request create call, got %d", len(requestCreator.calls))
	}
	call := requestCreator.calls[0]
	if call.SourceSystem != "rapidex-webhook" {
		t.Fatalf("unexpected source system: %q", call.SourceSystem)
	}
	if call.DestinationServerUID != "dhis2-uid" {
		t.Fatalf("unexpected destination uid: %q", call.DestinationServerUID)
	}
	if call.PayloadFormat != "json" || call.SubmissionBinding != "body" {
		t.Fatalf("unexpected transport config: %+v", call)
	}
	if call.URLSuffix != "/api/dataValueSets" {
		t.Fatalf("unexpected url suffix: %q", call.URLSuffix)
	}
	if call.CorrelationID != "rapidex:flow-1:contact-1" {
		t.Fatalf("unexpected correlation id: %q", call.CorrelationID)
	}
	if call.IdempotencyKey == "" {
		t.Fatal("expected idempotency key to be set")
	}
	if got := call.Extras["msisdn"]; got != "+256782820208" {
		t.Fatalf("expected msisdn extra, got %#v", got)
	}
	payload, ok := call.Payload.(AggregatePayload)
	if !ok {
		t.Fatalf("expected aggregate payload, got %T", call.Payload)
	}
	if payload.DataSet != "ds-1" || payload.OrgUnit != "OU_123" || payload.Period != "202604" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if len(payload.DataValues) != 1 || payload.DataValues[0].Value != "17" {
		t.Fatalf("unexpected data values: %+v", payload.DataValues)
	}
	if serverResolver.code != "dhis2-main" {
		t.Fatalf("expected lookup by dhis2-main, got %q", serverResolver.code)
	}
}

func TestProcessWebhookUsesNestedFlowUUIDFallback(t *testing.T) {
	requestCreator := &fakeExternalRequestCreator{}
	serverResolver := &fakeServerResolver{record: sukumadserver.Record{UID: "dhis2-uid"}}
	service := NewIntegrationService(fakeMappingProvider{
		ok: true,
		binding: WebhookBinding{
			MappingConfig: MappingConfig{
				FlowUUID:   "flow-1",
				FlowName:   "Weekly",
				Dataset:    "ds-1",
				OrgUnitVar: "facility",
				PeriodVar:  "period",
				Mappings: []DataValueMapping{
					{Field: "value_a", DataElement: "de-1"},
				},
			},
			RapidProServerCode: "rapidpro-main",
			DHIS2ServerCode:    "dhis2-main",
		},
	}, nil, requestCreator, serverResolver)

	webhook := RapidProWebhook{
		Results: map[string]interface{}{
			"facility": "OU_123",
			"period":   "202604",
			"value_a":  "17",
		},
	}
	webhook.Flow.UUID = "flow-1"
	webhook.Contact.UUID = "contact-1"

	if err := service.ProcessWebhook(context.Background(), webhook); err != nil {
		t.Fatalf("process webhook: %v", err)
	}
	if len(requestCreator.calls) != 1 {
		t.Fatalf("expected 1 request create call, got %d", len(requestCreator.calls))
	}
	call := requestCreator.calls[0]
	if call.CorrelationID != "rapidex:flow-1:contact-1" {
		t.Fatalf("unexpected correlation id: %q", call.CorrelationID)
	}
	if got := call.Extras["flowUuid"]; got != "flow-1" {
		t.Fatalf("expected resolved flowUuid extra, got %#v", got)
	}
	if serverResolver.code != "dhis2-main" {
		t.Fatalf("expected lookup by dhis2-main, got %q", serverResolver.code)
	}
}

func TestProcessWebhookUnwrapsNestedValueFields(t *testing.T) {
	requestCreator := &fakeExternalRequestCreator{}
	service := NewIntegrationService(fakeMappingProvider{
		ok: true,
		binding: WebhookBinding{
			MappingConfig: MappingConfig{
				FlowUUID:   "flow-1",
				Dataset:    "ds-1",
				OrgUnitVar: "facility",
				PeriodVar:  "period",
				PayloadAOC: "HllvX50cXC0",
				Mappings: []DataValueMapping{
					{Field: "value_a", DataElement: "de-1", CategoryOptionCombo: "HllvX50cXC0"},
					{Field: "value_b", DataElement: "de-2", CategoryOptionCombo: "HllvX50cXC0"},
				},
			},
			DHIS2ServerCode: "dhis2-main",
		},
	}, nil, requestCreator, &fakeServerResolver{record: sukumadserver.Record{UID: "dhis2-uid"}})

	webhook := RapidProWebhook{
		FlowUUID: "flow-1",
		Results: map[string]interface{}{
			"facility": map[string]interface{}{
				"category": "All Responses",
				"value":    "FvewOonC8lS",
			},
			"period": map[string]interface{}{
				"category": "All Responses",
				"value":    "2026W17",
			},
			"value_a": map[string]interface{}{
				"category": "OPD New Attendance",
				"value":    "1",
			},
			"value_b": map[string]interface{}{
				"category": "OPD Total Attendance",
				"value":    "3",
			},
		},
	}

	if err := service.ProcessWebhook(context.Background(), webhook); err != nil {
		t.Fatalf("process webhook: %v", err)
	}
	if len(requestCreator.calls) != 1 {
		t.Fatalf("expected 1 request create call, got %d", len(requestCreator.calls))
	}

	payload, ok := requestCreator.calls[0].Payload.(AggregatePayload)
	if !ok {
		t.Fatalf("expected aggregate payload, got %T", requestCreator.calls[0].Payload)
	}
	if payload.OrgUnit != "FvewOonC8lS" {
		t.Fatalf("expected mapped orgUnit value, got %q", payload.OrgUnit)
	}
	if payload.Period != "2026W17" {
		t.Fatalf("expected mapped period value, got %q", payload.Period)
	}
	if len(payload.DataValues) != 2 {
		t.Fatalf("expected 2 data values, got %+v", payload.DataValues)
	}
	if payload.DataValues[0].Value != "1" {
		t.Fatalf("expected first nested value to unwrap, got %q", payload.DataValues[0].Value)
	}
	if payload.DataValues[1].Value != "3" {
		t.Fatalf("expected second nested value to unwrap, got %q", payload.DataValues[1].Value)
	}
}

func TestProcessWebhookRejectsInvalidMappedPayload(t *testing.T) {
	service := NewIntegrationService(fakeMappingProvider{
		ok: true,
		binding: WebhookBinding{
			MappingConfig: MappingConfig{
				FlowUUID:   "flow-1",
				Dataset:    "ds-1",
				OrgUnitVar: "facility",
				PeriodVar:  "period",
				Mappings: []DataValueMapping{
					{Field: "value_a", DataElement: "de-1"},
				},
			},
			DHIS2ServerCode: "dhis2-main",
		},
	}, nil, &fakeExternalRequestCreator{}, &fakeServerResolver{record: sukumadserver.Record{UID: "dhis2-uid"}})

	err := service.ProcessWebhook(context.Background(), RapidProWebhook{
		FlowUUID: "flow-1",
		Results: map[string]interface{}{
			"period": "202604",
		},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Code != apperror.CodeValidationFailed {
		t.Fatalf("unexpected error code: %s", appErr.Code)
	}
	if _, ok := appErr.Details["orgUnit"]; !ok {
		t.Fatalf("expected orgUnit validation error, got %+v", appErr.Details)
	}
	if _, ok := appErr.Details["dataValues"]; !ok {
		t.Fatalf("expected dataValues validation error, got %+v", appErr.Details)
	}
}

func TestProcessWebhookRejectsMissingFlowUUIDAcrossAllLocations(t *testing.T) {
	service := NewIntegrationService(fakeMappingProvider{}, nil, &fakeExternalRequestCreator{}, &fakeServerResolver{})

	err := service.ProcessWebhook(context.Background(), RapidProWebhook{
		FlowUUID: "   ",
		Results: map[string]interface{}{
			"facility": "OU_123",
		},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Code != apperror.CodeValidationFailed {
		t.Fatalf("unexpected error code: %s", appErr.Code)
	}
	if got := appErr.Details["flow_uuid"]; got == nil {
		t.Fatalf("expected flow_uuid validation details, got %+v", appErr.Details)
	}
}

func TestProcessWebhookUsesStableIdempotencyKeyForDuplicates(t *testing.T) {
	requestCreator := &fakeExternalRequestCreator{}
	service := NewIntegrationService(fakeMappingProvider{
		ok: true,
		binding: WebhookBinding{
			MappingConfig: MappingConfig{
				FlowUUID:   "flow-1",
				Dataset:    "ds-1",
				OrgUnitVar: "facility",
				PeriodVar:  "period",
				Mappings: []DataValueMapping{
					{Field: "value_a", DataElement: "de-1"},
				},
			},
			DHIS2ServerCode: "dhis2-main",
		},
	}, nil, requestCreator, &fakeServerResolver{record: sukumadserver.Record{UID: "dhis2-uid"}})

	webhook := RapidProWebhook{
		FlowUUID: "flow-1",
		Results: map[string]interface{}{
			"facility": "OU_123",
			"period":   "202604",
			"value_a":  "17",
		},
	}

	if err := service.ProcessWebhook(context.Background(), webhook); err != nil {
		t.Fatalf("first process webhook: %v", err)
	}
	firstKey := requestCreator.lastKey
	if err := service.ProcessWebhook(context.Background(), webhook); err != nil {
		t.Fatalf("second process webhook: %v", err)
	}
	if requestCreator.lastKey != firstKey {
		t.Fatalf("expected stable idempotency key, first=%q second=%q", firstKey, requestCreator.lastKey)
	}
	if requestCreator.created != 1 || requestCreator.deduped != 1 {
		t.Fatalf("expected one create and one dedupe, got created=%d deduped=%d", requestCreator.created, requestCreator.deduped)
	}
}

func TestProcessWebhookRejectsUnknownDHIS2Server(t *testing.T) {
	service := NewIntegrationService(fakeMappingProvider{
		ok: true,
		binding: WebhookBinding{
			MappingConfig: MappingConfig{
				FlowUUID:   "flow-1",
				Dataset:    "ds-1",
				OrgUnitVar: "facility",
				PeriodVar:  "period",
				Mappings: []DataValueMapping{
					{Field: "value_a", DataElement: "de-1"},
				},
			},
			DHIS2ServerCode: "dhis2-main",
		},
	}, nil, &fakeExternalRequestCreator{}, &fakeServerResolver{err: sql.ErrNoRows})

	err := service.ProcessWebhook(context.Background(), RapidProWebhook{
		FlowUUID: "flow-1",
		Results: map[string]interface{}{
			"facility": "OU_123",
			"period":   "202604",
			"value_a":  "17",
		},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error, got %T", err)
	}
	if _, ok := appErr.Details["dhis2ServerCode"]; !ok {
		t.Fatalf("expected dhis2ServerCode validation error, got %+v", appErr.Details)
	}
}
