package observability

import (
	"time"

	"basepro/backend/internal/sukumad/traceevent"
)

type Summary struct {
	Workers    int `json:"workers"`
	RateLimits int `json:"rateLimits"`
}

type EventActor = traceevent.Actor
type EventWriteInput = traceevent.WriteInput

type EventRecord struct {
	ID                int64          `db:"id" json:"id"`
	UID               string         `db:"uid" json:"uid"`
	RequestID         *int64         `db:"request_id" json:"requestId,omitempty"`
	RequestUID        string         `json:"requestUid"`
	DeliveryAttemptID *int64         `db:"delivery_attempt_id" json:"deliveryAttemptId,omitempty"`
	DeliveryUID       string         `json:"deliveryUid"`
	AsyncTaskID       *int64         `db:"async_task_id" json:"asyncTaskId,omitempty"`
	AsyncTaskUID      string         `json:"asyncTaskUid"`
	WorkerRunID       *int64         `db:"worker_run_id" json:"workerRunId,omitempty"`
	WorkerRunUID      string         `json:"workerRunUid"`
	EventType         string         `db:"event_type" json:"eventType"`
	EventLevel        string         `db:"event_level" json:"eventLevel"`
	EventData         map[string]any `json:"eventData,omitempty"`
	EventDataPreview  string         `json:"eventDataPreview"`
	Message           string         `db:"message" json:"message"`
	CorrelationID     string         `db:"correlation_id" json:"correlationId"`
	ActorType         string         `db:"actor_type" json:"actorType"`
	ActorUserID       *int64         `db:"actor_user_id" json:"actorUserId,omitempty"`
	ActorName         string         `db:"actor_name" json:"actorName"`
	SourceComponent   string         `db:"source_component" json:"sourceComponent"`
	CreatedAt         time.Time      `db:"created_at" json:"createdAt"`
}

type EventListQuery struct {
	Page              int
	PageSize          int
	RequestID         *int64
	DeliveryAttemptID *int64
	AsyncTaskID       *int64
	WorkerRunID       *int64
	CorrelationID     string
	EventType         string
	Level             string
	From              *time.Time
	To                *time.Time
	SortOrder         string
}

type EventListResult struct {
	Items    []EventRecord
	Total    int
	Page     int
	PageSize int
}

type TraceReference struct {
	ID        int64     `json:"id"`
	UID       string    `json:"uid"`
	CreatedAt time.Time `json:"createdAt"`
}

type TraceSummary struct {
	Requests   []TraceReference `json:"requests"`
	Deliveries []TraceReference `json:"deliveries"`
	Jobs       []TraceReference `json:"jobs"`
	Workers    []TraceReference `json:"workers"`
}

type TraceResult struct {
	CorrelationID string        `json:"correlationId"`
	Summary       TraceSummary  `json:"summary"`
	Events        []EventRecord `json:"events"`
}
