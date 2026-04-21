package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"basepro/backend/internal/sukumad/delivery"
	"basepro/backend/internal/sukumad/reporter"
	requests "basepro/backend/internal/sukumad/request"
	sukumadserver "basepro/backend/internal/sukumad/server"
)

const (
	JobTypeURLCall              = "url_call"
	JobTypeRequestExchange      = "request_exchange"
	JobTypeRapidProReporterSync = "rapidpro_reporter_sync"
)

type integrationHandlerDependencies struct {
	serverLookup interface {
		GetServerByUID(context.Context, string) (sukumadserver.Record, error)
	}
	requestCreator interface {
		CreateExternalRequest(context.Context, requests.ExternalCreateInput) (requests.CreateResult, error)
	}
	submitter interface {
		Submit(context.Context, delivery.DispatchInput) (delivery.DispatchResult, error)
	}
	reporterSyncer interface {
		SyncUpdatedSince(context.Context, *time.Time, int, bool, bool) (reporter.SyncBatchResult, error)
	}
}

type urlCallConfig struct {
	DestinationServerUID    string `json:"destinationServerUid"`
	URLSuffix               string `json:"urlSuffix"`
	Payload                 any    `json:"payload"`
	PayloadFormat           string `json:"payloadFormat"`
	SubmissionBinding       string `json:"submissionBinding"`
	ResponseBodyPersistence string `json:"responseBodyPersistence"`
}

type requestExchangeConfig struct {
	SourceSystem            string         `json:"sourceSystem"`
	DestinationServerUID    string         `json:"destinationServerUid"`
	DestinationServerUIDs   []string       `json:"destinationServerUids"`
	BatchID                 string         `json:"batchId"`
	CorrelationID           string         `json:"correlationId"`
	IdempotencyKeyPrefix    string         `json:"idempotencyKeyPrefix"`
	Payload                 any            `json:"payload"`
	PayloadFormat           string         `json:"payloadFormat"`
	SubmissionBinding       string         `json:"submissionBinding"`
	ResponseBodyPersistence string         `json:"responseBodyPersistence"`
	URLSuffix               string         `json:"urlSuffix"`
	Metadata                map[string]any `json:"metadata"`
}

type rapidProReporterSyncConfig struct {
	BatchSize  int  `json:"batchSize"`
	DryRun     bool `json:"dryRun"`
	OnlyActive bool `json:"onlyActive"`
}

func newIntegrationHandlers(deps integrationHandlerDependencies) map[string]typedRegistration {
	return map[string]typedRegistration{
		JobTypeURLCall: {
			handler: typedHandlerWithValidate[urlCallConfig](func(ctx context.Context, exec JobExecution, cfg urlCallConfig) (JobResult, error) {
				return runURLCall(ctx, exec, cfg, deps)
			}, validateURLCallConfig),
		},
		JobTypeRequestExchange: {
			handler: typedHandlerWithValidate[requestExchangeConfig](func(ctx context.Context, exec JobExecution, cfg requestExchangeConfig) (JobResult, error) {
				return runRequestExchange(ctx, exec, cfg, deps)
			}, validateRequestExchangeConfig),
		},
		JobTypeRapidProReporterSync: {
			handler: typedHandlerWithValidate[rapidProReporterSyncConfig](func(ctx context.Context, exec JobExecution, cfg rapidProReporterSyncConfig) (JobResult, error) {
				return runRapidProReporterSync(ctx, exec, cfg, deps)
			}, validateRapidProReporterSyncConfig),
		},
	}
}

func runURLCall(ctx context.Context, exec JobExecution, cfg urlCallConfig, deps integrationHandlerDependencies) (JobResult, error) {
	if deps.serverLookup == nil {
		return JobResult{}, fmt.Errorf("server lookup is not configured")
	}
	if deps.submitter == nil {
		return JobResult{}, fmt.Errorf("url call submitter is not configured")
	}

	serverRecord, err := deps.serverLookup.GetServerByUID(ctx, strings.TrimSpace(cfg.DestinationServerUID))
	if err != nil {
		return JobResult{}, err
	}
	payloadBody, err := encodeSchedulerPayload(cfg.Payload, normalizeSchedulerPayloadFormat(cfg.PayloadFormat))
	if err != nil {
		return JobResult{}, err
	}

	result, err := deps.submitter.Submit(ctx, delivery.DispatchInput{
		CorrelationID:           schedulerRunCorrelation(exec),
		PayloadBody:             payloadBody,
		PayloadFormat:           normalizeSchedulerPayloadFormat(cfg.PayloadFormat),
		SubmissionBinding:       normalizeSchedulerSubmissionBinding(cfg.SubmissionBinding),
		ResponseBodyPersistence: normalizeSchedulerResponseBodyPersistence(cfg.ResponseBodyPersistence),
		URLSuffix:               strings.TrimSpace(cfg.URLSuffix),
		Server:                  serverSnapshot(serverRecord),
	})
	if err != nil {
		return JobResult{ResultSummary: baseURLCallSummary(exec, serverRecord, result)}, err
	}

	status := RunStatusSucceeded
	if result.Terminal && !result.Succeeded {
		status = RunStatusFailed
	}
	summary := baseURLCallSummary(exec, serverRecord, result)
	return JobResult{Status: status, ResultSummary: summary}, nil
}

func runRequestExchange(ctx context.Context, exec JobExecution, cfg requestExchangeConfig, deps integrationHandlerDependencies) (JobResult, error) {
	if deps.requestCreator == nil {
		return JobResult{}, fmt.Errorf("request exchange creator is not configured")
	}

	correlationID := strings.TrimSpace(cfg.CorrelationID)
	if correlationID == "" {
		correlationID = schedulerRunCorrelation(exec)
	}
	idempotencyKey := ""
	if prefix := strings.TrimSpace(cfg.IdempotencyKeyPrefix); prefix != "" {
		idempotencyKey = prefix + ":" + exec.Run.UID
	}

	result, err := deps.requestCreator.CreateExternalRequest(ctx, requests.ExternalCreateInput{
		SourceSystem:            strings.TrimSpace(cfg.SourceSystem),
		DestinationServerUID:    strings.TrimSpace(cfg.DestinationServerUID),
		DestinationServerUIDs:   normalizeStringList(cfg.DestinationServerUIDs),
		BatchID:                 strings.TrimSpace(cfg.BatchID),
		CorrelationID:           correlationID,
		IdempotencyKey:          idempotencyKey,
		Payload:                 cfg.Payload,
		PayloadFormat:           normalizeSchedulerPayloadFormat(cfg.PayloadFormat),
		SubmissionBinding:       normalizeSchedulerSubmissionBinding(cfg.SubmissionBinding),
		ResponseBodyPersistence: normalizeSchedulerResponseBodyPersistence(cfg.ResponseBodyPersistence),
		URLSuffix:               strings.TrimSpace(cfg.URLSuffix),
		Extras:                  cloneJSONMap(cfg.Metadata),
	})
	if err != nil {
		return JobResult{}, err
	}

	record := result.Record
	return JobResult{
		Status: RunStatusSucceeded,
		ResultSummary: map[string]any{
			"jobType":              JobTypeRequestExchange,
			"runUid":               exec.Run.UID,
			"requestId":            record.ID,
			"requestUid":           record.UID,
			"requestStatus":        record.Status,
			"correlationId":        record.CorrelationID,
			"destinationServerUid": record.DestinationServerUID,
			"destinationServer":    record.DestinationServerCode,
			"deliveryCount":        len(record.Targets),
			"deduped":              result.Deduped,
			"created":              result.Created,
		},
	}, nil
}

func runRapidProReporterSync(ctx context.Context, exec JobExecution, cfg rapidProReporterSyncConfig, deps integrationHandlerDependencies) (JobResult, error) {
	if deps.reporterSyncer == nil {
		return JobResult{}, fmt.Errorf("rapidpro reporter sync service is not configured")
	}
	result, err := deps.reporterSyncer.SyncUpdatedSince(ctx, exec.Job.LastSuccessAt, cfg.BatchSize, cfg.OnlyActive, cfg.DryRun)
	summary := map[string]any{
		"jobType":       JobTypeRapidProReporterSync,
		"runUid":        exec.Run.UID,
		"watermarkFrom": exec.Job.LastSuccessAt,
		"watermarkTo":   result.WatermarkTo,
		"scannedCount":  result.Scanned,
		"syncedCount":   result.Synced,
		"createdCount":  result.Created,
		"updatedCount":  result.Updated,
		"failedCount":   result.Failed,
		"dryRun":        cfg.DryRun,
		"onlyActive":    cfg.OnlyActive,
	}
	if err != nil {
		return JobResult{ResultSummary: summary}, err
	}
	return JobResult{Status: RunStatusSucceeded, ResultSummary: summary}, nil
}

func validateURLCallConfig(cfg urlCallConfig) map[string]any {
	details := map[string]any{}
	if strings.TrimSpace(cfg.DestinationServerUID) == "" {
		details["config.destinationServerUid"] = []string{"is required"}
	}
	validateSchedulerRequestConfig(details, cfg.Payload, cfg.PayloadFormat, cfg.SubmissionBinding, cfg.ResponseBodyPersistence)
	return detailsOrNil(details)
}

func validateRequestExchangeConfig(cfg requestExchangeConfig) map[string]any {
	details := map[string]any{}
	if strings.TrimSpace(cfg.SourceSystem) == "" {
		details["config.sourceSystem"] = []string{"is required"}
	}
	if strings.TrimSpace(cfg.DestinationServerUID) == "" {
		details["config.destinationServerUid"] = []string{"is required"}
	}
	if strings.TrimSpace(cfg.IdempotencyKeyPrefix) != "" && strings.Contains(strings.TrimSpace(cfg.IdempotencyKeyPrefix), " ") {
		details["config.idempotencyKeyPrefix"] = []string{"must not contain spaces"}
	}
	validateSchedulerRequestConfig(details, cfg.Payload, cfg.PayloadFormat, cfg.SubmissionBinding, cfg.ResponseBodyPersistence)
	if cfg.Metadata != nil {
		if _, err := json.Marshal(cfg.Metadata); err != nil {
			details["config.metadata"] = []string{"must be valid JSON"}
		}
	}
	return detailsOrNil(details)
}

func validateRapidProReporterSyncConfig(cfg rapidProReporterSyncConfig) map[string]any {
	details := map[string]any{}
	if cfg.BatchSize <= 0 {
		details["config.batchSize"] = []string{"must be greater than zero"}
	}
	return detailsOrNil(details)
}

func validateSchedulerRequestConfig(details map[string]any, payload any, payloadFormat string, submissionBinding string, responseBodyPersistence string) {
	format := normalizeSchedulerPayloadFormat(payloadFormat)
	if format != requests.PayloadFormatJSON && format != requests.PayloadFormatText {
		details["config.payloadFormat"] = []string{"must be one of json or text"}
	}
	binding := normalizeSchedulerSubmissionBinding(submissionBinding)
	if binding != requests.SubmissionBindingBody && binding != requests.SubmissionBindingQuery {
		details["config.submissionBinding"] = []string{"must be one of body or query"}
	}
	if payload == nil {
		details["config.payload"] = []string{"is required"}
	} else if payloadBody, err := encodeSchedulerPayload(payload, format); err != nil {
		details["config.payload"] = []string{err.Error()}
	} else if err := validateSchedulerPayloadBinding(payloadBody, format, binding); err != nil {
		details["config.payload"] = []string{err.Error()}
	}
	switch strings.ToLower(strings.TrimSpace(responseBodyPersistence)) {
	case "", "default", "filter", "save", "discard":
	default:
		details["config.responseBodyPersistence"] = []string{"must be one of default, filter, save, or discard"}
	}
}

func validateSchedulerPayloadBinding(payloadBody string, payloadFormat string, submissionBinding string) error {
	if submissionBinding != requests.SubmissionBindingQuery {
		return nil
	}
	switch payloadFormat {
	case requests.PayloadFormatJSON:
		var parsed map[string]any
		if err := json.Unmarshal([]byte(payloadBody), &parsed); err != nil {
			return fmt.Errorf("must be a JSON object when sent as query params")
		}
		if len(parsed) == 0 {
			return fmt.Errorf("must include at least one query param")
		}
		for key, value := range parsed {
			if strings.TrimSpace(key) == "" {
				return fmt.Errorf("query param names must be non-empty")
			}
			if !isSchedulerQueryParamValue(value) {
				return fmt.Errorf("query param values must be strings, numbers, booleans, null, or arrays of those values")
			}
		}
	case requests.PayloadFormatText:
		if _, err := url.ParseQuery(strings.TrimSpace(payloadBody)); err != nil {
			return fmt.Errorf("must be a valid query string")
		}
	}
	return nil
}

func isSchedulerQueryParamValue(value any) bool {
	switch typed := value.(type) {
	case nil, string, bool, float64:
		return true
	case []any:
		for _, item := range typed {
			if !isSchedulerQueryParamValue(item) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func encodeSchedulerPayload(payload any, payloadFormat string) (string, error) {
	switch payloadFormat {
	case "", requests.PayloadFormatJSON:
		bytes, err := json.Marshal(payload)
		if err != nil {
			return "", fmt.Errorf("must be valid JSON")
		}
		if strings.TrimSpace(string(bytes)) == "" || string(bytes) == "null" {
			return "", fmt.Errorf("is required")
		}
		return string(bytes), nil
	case requests.PayloadFormatText:
		value, ok := payload.(string)
		if !ok {
			return "", fmt.Errorf("must be a text value")
		}
		value = strings.TrimSpace(value)
		if value == "" {
			return "", fmt.Errorf("is required")
		}
		return value, nil
	default:
		return "", fmt.Errorf("payloadFormat must be one of json or text")
	}
}

func normalizeSchedulerPayloadFormat(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", requests.PayloadFormatJSON:
		return requests.PayloadFormatJSON
	case requests.PayloadFormatText:
		return requests.PayloadFormatText
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func normalizeSchedulerSubmissionBinding(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", requests.SubmissionBindingBody:
		return requests.SubmissionBindingBody
	case requests.SubmissionBindingQuery:
		return requests.SubmissionBindingQuery
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func normalizeSchedulerResponseBodyPersistence(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "server_default", "server-default", "default":
		return ""
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func serverSnapshot(record sukumadserver.Record) delivery.ServerSnapshot {
	return delivery.ServerSnapshot{
		ID:                      record.ID,
		Code:                    record.Code,
		Name:                    record.Name,
		SystemType:              record.SystemType,
		BaseURL:                 record.BaseURL,
		EndpointType:            record.EndpointType,
		HTTPMethod:              record.HTTPMethod,
		UseAsync:                record.UseAsync,
		ResponseBodyPersistence: record.ResponseBodyPersistence,
		Headers:                 cloneStringMap(record.Headers),
		URLParams:               cloneStringMap(record.URLParams),
	}
}

func baseURLCallSummary(exec JobExecution, serverRecord sukumadserver.Record, result delivery.DispatchResult) map[string]any {
	summary := map[string]any{
		"jobType":               JobTypeURLCall,
		"runUid":                exec.Run.UID,
		"destinationServerUid":  serverRecord.UID,
		"destinationServerCode": serverRecord.Code,
		"destinationServerName": serverRecord.Name,
		"async":                 result.Async,
		"terminal":              result.Terminal,
		"succeeded":             result.Succeeded,
		"responseBodyFiltered":  result.ResponseBodyFiltered,
		"responseSummary":       cloneJSONMap(result.ResponseSummary),
	}
	if result.HTTPStatus != nil {
		summary["httpStatus"] = *result.HTTPStatus
	}
	if strings.TrimSpace(result.ResponseContentType) != "" {
		summary["responseContentType"] = strings.TrimSpace(result.ResponseContentType)
	}
	if strings.TrimSpace(result.RemoteStatus) != "" {
		summary["remoteStatus"] = strings.TrimSpace(result.RemoteStatus)
	}
	if strings.TrimSpace(result.RemoteJobID) != "" {
		summary["remoteJobId"] = strings.TrimSpace(result.RemoteJobID)
	}
	if strings.TrimSpace(result.ErrorMessage) != "" {
		summary["errorMessage"] = strings.TrimSpace(result.ErrorMessage)
	}
	return summary
}

func schedulerRunCorrelation(exec JobExecution) string {
	return "scheduler:" + strings.TrimSpace(exec.Job.Code) + ":" + strings.TrimSpace(exec.Run.UID)
}

func normalizeStringList(input []string) []string {
	items := make([]string, 0, len(input))
	for _, raw := range input {
		value := strings.TrimSpace(raw)
		if value != "" {
			items = append(items, value)
		}
	}
	return items
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return map[string]string{}
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func detailsOrNil(details map[string]any) map[string]any {
	if len(details) == 0 {
		return nil
	}
	return details
}
