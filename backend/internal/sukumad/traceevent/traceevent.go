package traceevent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	ActorSystem = "system"
	ActorWorker = "worker"
	ActorUser   = "user"
)

const (
	EventRequestCreated       = "request.created"
	EventRequestSubmitted     = "request.submitted"
	EventRequestStatusChanged = "request.status_changed"
	EventRequestCompleted     = "request.completed"
	EventRequestFailed        = "request.failed"
	EventDeliveryCreated      = "delivery.created"
	EventDeliveryStarted      = "delivery.started"
	EventDeliverySucceeded    = "delivery.succeeded"
	EventDeliveryFailed       = "delivery.failed"
	EventDeliveryRetrySched   = "delivery.retry_scheduled"
	EventDeliveryRetryStarted = "delivery.retry_started"
	EventDeliveryResponse     = "delivery.response_received"
	EventAsyncCreated         = "async.created"
	EventAsyncPollStarted     = "async.poll_started"
	EventAsyncPollSucceeded   = "async.poll_succeeded"
	EventAsyncPollFailed      = "async.poll_failed"
	EventAsyncCompleted       = "async.completed"
	EventAsyncFailed          = "async.failed"
	EventWorkerStarted        = "worker.started"
	EventWorkerHeartbeat      = "worker.heartbeat"
	EventWorkerStopped        = "worker.stopped"
	EventWorkerError          = "worker.error"
)

type Actor struct {
	Type   string
	UserID *int64
	Name   string
}

type WriteInput struct {
	RequestID         *int64
	DeliveryAttemptID *int64
	AsyncTaskID       *int64
	WorkerRunID       *int64
	EventType         string
	EventLevel        string
	EventData         map[string]any
	Message           string
	CorrelationID     string
	Actor             Actor
	SourceComponent   string
}

type Writer interface {
	AppendEvent(context.Context, WriteInput) error
}

func SanitizeData(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	sanitized := make(map[string]any, len(input))
	for key, value := range input {
		if IsSensitiveKey(key) {
			sanitized[key] = "[masked]"
			continue
		}
		switch typed := value.(type) {
		case map[string]any:
			sanitized[key] = SanitizeData(typed)
		case []any:
			items := make([]any, 0, len(typed))
			for _, item := range typed {
				if nested, ok := item.(map[string]any); ok {
					items = append(items, SanitizeData(nested))
					continue
				}
				items = append(items, item)
			}
			sanitized[key] = items
		default:
			sanitized[key] = value
		}
	}
	return sanitized
}

func PreviewData(input map[string]any) string {
	if len(input) == 0 {
		return ""
	}
	encoded, err := json.Marshal(input)
	if err != nil {
		return "{...}"
	}
	if len(encoded) > 180 {
		return string(encoded[:177]) + "..."
	}
	return string(encoded)
}

func NormalizeLevel(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "info":
		return "info"
	case "warning", "warn":
		return "warning"
	case "error":
		return "error"
	default:
		return ""
	}
}

func NormalizeActorType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", ActorSystem:
		return ActorSystem
	case ActorWorker:
		return ActorWorker
	case ActorUser:
		return ActorUser
	default:
		return ActorSystem
	}
}

func IsSensitiveKey(key string) bool {
	needle := strings.ToLower(strings.TrimSpace(key))
	for _, part := range []string{"token", "password", "secret", "authorization", "api_key", "apikey", "refresh", "access"} {
		if strings.Contains(needle, part) {
			return true
		}
	}
	return false
}

func Message(fallback string, format string, args ...any) string {
	if strings.TrimSpace(format) == "" {
		return fallback
	}
	return fmt.Sprintf(format, args...)
}
