package dashboard

import (
	"time"

	"basepro/backend/internal/sukumad/traceevent"
)

func mapSourceEvent(input SourceEvent) StreamEvent {
	event := StreamEvent{
		Type:          input.Type,
		Timestamp:     input.Timestamp.UTC(),
		Severity:      normalizeSeverity(input.Severity),
		Summary:       input.Message,
		CorrelationID: input.CorrelationID,
		RequestID:     input.RequestID,
		DeliveryID:    input.DeliveryID,
		JobID:         input.JobID,
		WorkerID:      input.WorkerID,
		ServerID:      extractServerID(input.Payload),
		Patch:         patchForEventType(input.Type),
		Invalidations: invalidationsForEventType(input.Type),
		Payload:       clonePayload(input.Payload),
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	switch {
	case input.DeliveryID != nil:
		event.EntityType = "delivery"
		event.EntityID = *input.DeliveryID
		event.EntityUID = input.DeliveryUID
	case input.JobID != nil:
		event.EntityType = "job"
		event.EntityID = *input.JobID
		event.EntityUID = input.JobUID
	case input.WorkerID != nil:
		event.EntityType = "worker"
		event.EntityID = *input.WorkerID
		event.EntityUID = input.WorkerUID
	case input.RequestID != nil:
		event.EntityType = "request"
		event.EntityID = *input.RequestID
		event.EntityUID = input.RequestUID
	default:
		event.EntityType = "system"
	}
	return event
}

func normalizeSeverity(value string) string {
	switch value {
	case "warning", "error":
		return value
	default:
		return "info"
	}
}

func clonePayload(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}
	return traceevent.SanitizeData(input)
}

func extractServerID(payload map[string]any) *int64 {
	for _, key := range []string{"serverId", "server_id"} {
		if value, ok := payload[key]; ok {
			if parsed, ok := toInt64(value); ok {
				return &parsed
			}
		}
	}
	return nil
}

func toInt64(value any) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int32:
		return int64(typed), true
	case int64:
		return typed, true
	case float64:
		return int64(typed), true
	default:
		return 0, false
	}
}

func patchForEventType(eventType string) *StreamPatch {
	switch eventType {
	case "request.created", "request.submitted":
		return &StreamPatch{KPI: "pendingRequests", Op: "increment", Value: 1}
	case "request.completed", "request.failed":
		return &StreamPatch{KPI: "pendingRequests", Op: "decrement", Value: 1}
	case "delivery.created", "delivery.retry_scheduled":
		return &StreamPatch{KPI: "pendingDeliveries", Op: "increment", Value: 1}
	case "delivery.started", "delivery.retry_started":
		return &StreamPatch{KPI: "runningDeliveries", Op: "increment", Value: 1}
	case "delivery.succeeded":
		return &StreamPatch{KPI: "runningDeliveries", Op: "decrement", Value: 1}
	case "delivery.failed":
		return &StreamPatch{KPI: "failedDeliveriesLastHour", Op: "increment", Value: 1}
	case "async.poll_started":
		return &StreamPatch{KPI: "pollingJobs", Op: "increment", Value: 1}
	case "async.completed", "async.failed":
		return &StreamPatch{KPI: "pollingJobs", Op: "decrement", Value: 1}
	default:
		return nil
	}
}

func invalidationsForEventType(eventType string) []string {
	switch eventType {
	case "request.created", "request.submitted", "request.completed", "request.failed", "request.status_changed":
		return []string{"kpis", "recentEvents", "processingGraph", "attention.failedDeliveries"}
	case "delivery.created", "delivery.started", "delivery.succeeded", "delivery.failed", "delivery.retry_scheduled", "delivery.retry_started":
		return []string{"kpis", "recentEvents", "attention.failedDeliveries", "attention.staleRunningDeliveries", "trends.deliveriesByStatus", "trends.failuresByServer"}
	case "async.created", "async.poll_started", "async.poll_succeeded", "async.poll_failed", "async.completed", "async.failed":
		return []string{"kpis", "recentEvents", "attention.stuckJobs", "trends.jobsByState"}
	case "worker.started", "worker.heartbeat", "worker.stopped", "worker.error":
		return []string{"kpis", "recentEvents", "workers", "attention.unhealthyWorkers"}
	default:
		return []string{"recentEvents"}
	}
}
