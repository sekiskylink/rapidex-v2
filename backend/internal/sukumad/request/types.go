package request

import (
	"context"
	"time"
)

const (
	StatusPending    = "pending"
	StatusBlocked    = "blocked"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

const (
	TargetStatusPending    = "pending"
	TargetStatusBlocked    = "blocked"
	TargetStatusProcessing = "processing"
	TargetStatusSucceeded  = "succeeded"
	TargetStatusFailed     = "failed"
)

const (
	PayloadFormatJSON = "json"
	PayloadFormatText = "text"
)

const (
	SubmissionBindingBody  = "body"
	SubmissionBindingQuery = "query"
)

type Record struct {
	ID                     int64           `db:"id" json:"id"`
	UID                    string          `db:"uid" json:"uid"`
	SourceSystem           string          `db:"source_system" json:"sourceSystem"`
	DestinationServerID    int64           `db:"destination_server_id" json:"destinationServerId"`
	DestinationServerName  string          `json:"destinationServerName"`
	BatchID                string          `db:"batch_id" json:"batchId"`
	CorrelationID          string          `db:"correlation_id" json:"correlationId"`
	IdempotencyKey         string          `db:"idempotency_key" json:"idempotencyKey"`
	PayloadBody            string          `db:"payload_body" json:"payloadBody"`
	PayloadFormat          string          `db:"payload_format" json:"payloadFormat"`
	SubmissionBinding      string          `db:"submission_binding" json:"submissionBinding"`
	URLSuffix              string          `db:"url_suffix" json:"urlSuffix"`
	Status                 string          `db:"status" json:"status"`
	StatusReason           string          `db:"status_reason" json:"statusReason"`
	DeferredUntil          *time.Time      `db:"deferred_until" json:"deferredUntil,omitempty"`
	Extras                 map[string]any  `json:"extras"`
	CreatedAt              time.Time       `db:"created_at" json:"createdAt"`
	UpdatedAt              time.Time       `db:"updated_at" json:"updatedAt"`
	CreatedBy              *int64          `db:"created_by" json:"createdBy,omitempty"`
	Payload                any             `json:"payload"`
	LatestDeliveryID       *int64          `json:"latestDeliveryId,omitempty"`
	LatestDeliveryUID      string          `json:"latestDeliveryUid"`
	LatestDeliveryStatus   string          `json:"latestDeliveryStatus"`
	LatestAsyncTaskID      *int64          `json:"latestAsyncTaskId,omitempty"`
	LatestAsyncTaskUID     string          `json:"latestAsyncTaskUid"`
	LatestAsyncState       string          `json:"latestAsyncState"`
	LatestAsyncRemoteJobID string          `json:"latestAsyncRemoteJobId"`
	LatestAsyncPollURL     string          `json:"latestAsyncPollUrl"`
	AwaitingAsync          bool            `json:"awaitingAsync"`
	Targets                []TargetRecord  `json:"targets"`
	Dependencies           []DependencyRef `json:"dependencies"`
}

type ListQuery struct {
	Page      int
	PageSize  int
	SortField string
	SortOrder string
	Filter    string
	Status    string
}

type ListResult struct {
	Items    []Record
	Total    int
	Page     int
	PageSize int
}

type CreateParams struct {
	UID                 string
	SourceSystem        string
	DestinationServerID int64
	BatchID             string
	CorrelationID       string
	IdempotencyKey      string
	PayloadBody         string
	PayloadFormat       string
	SubmissionBinding   string
	URLSuffix           string
	Status              string
	StatusReason        string
	DeferredUntil       *time.Time
	Extras              map[string]any
	CreatedBy           *int64
}

type CreateInput struct {
	SourceSystem         string
	DestinationServerID  int64
	DestinationServerIDs []int64
	DependencyRequestIDs []int64
	BatchID              string
	CorrelationID        string
	IdempotencyKey       string
	Payload              any
	PayloadFormat        string
	SubmissionBinding    string
	URLSuffix            string
	Extras               map[string]any
	ActorID              *int64
}

type Repository interface {
	ListRequests(ctx context.Context, query ListQuery) (ListResult, error)
	GetRequestByID(ctx context.Context, id int64) (Record, error)
	CreateRequest(ctx context.Context, params CreateParams) (Record, error)
	UpdateRequestStatus(ctx context.Context, id int64, status string, reason string, deferredUntil *time.Time) (Record, error)
	CreateTargets(ctx context.Context, requestID int64, targets []CreateTargetParams) ([]TargetRecord, error)
	ListTargetsByRequest(ctx context.Context, requestID int64) ([]TargetRecord, error)
	UpdateTarget(ctx context.Context, params UpdateTargetParams) (TargetRecord, error)
	CreateDependencies(ctx context.Context, requestID int64, dependencyIDs []int64) error
	ListDependencies(ctx context.Context, requestID int64) ([]DependencyRef, error)
	ListDependents(ctx context.Context, dependencyRequestID int64) ([]DependencyRef, error)
	GetDependencyStatuses(ctx context.Context, requestID int64) ([]DependencyStatus, error)
	DependencyPathExists(ctx context.Context, fromRequestID int64, toRequestID int64) (bool, error)
}

type TargetRecord struct {
	ID                     int64      `db:"id" json:"id"`
	UID                    string     `db:"uid" json:"uid"`
	RequestID              int64      `db:"request_id" json:"requestId"`
	ServerID               int64      `db:"server_id" json:"serverId"`
	ServerName             string     `json:"serverName"`
	ServerCode             string     `json:"serverCode"`
	TargetKind             string     `db:"target_kind" json:"targetKind"`
	Priority               int        `db:"priority" json:"priority"`
	Status                 string     `db:"status" json:"status"`
	BlockedReason          string     `db:"blocked_reason" json:"blockedReason"`
	DeferredUntil          *time.Time `db:"deferred_until" json:"deferredUntil,omitempty"`
	LastReleasedAt         *time.Time `db:"last_released_at" json:"lastReleasedAt,omitempty"`
	LatestDeliveryID       *int64     `json:"latestDeliveryId,omitempty"`
	LatestDeliveryUID      string     `json:"latestDeliveryUid"`
	LatestDeliveryStatus   string     `json:"latestDeliveryStatus"`
	LatestAsyncTaskID      *int64     `json:"latestAsyncTaskId,omitempty"`
	LatestAsyncTaskUID     string     `json:"latestAsyncTaskUid"`
	LatestAsyncState       string     `json:"latestAsyncState"`
	LatestAsyncRemoteJobID string     `json:"latestAsyncRemoteJobId"`
	LatestAsyncPollURL     string     `json:"latestAsyncPollUrl"`
	AwaitingAsync          bool       `json:"awaitingAsync"`
	CreatedAt              time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt              time.Time  `db:"updated_at" json:"updatedAt"`
}

type DependencyRef struct {
	RequestID                      int64      `db:"request_id" json:"requestId"`
	DependsOnRequestID             int64      `db:"depends_on_request_id" json:"dependsOnRequestId"`
	RequestUID                     string     `db:"request_uid" json:"requestUid"`
	DependsOnUID                   string     `db:"depends_on_uid" json:"dependsOnUid"`
	Status                         string     `db:"status" json:"status"`
	StatusReason                   string     `db:"status_reason" json:"statusReason"`
	DeferredUntil                  *time.Time `db:"deferred_until" json:"deferredUntil,omitempty"`
	DependsOnDestinationServerName string     `db:"depends_on_destination_server_name" json:"dependsOnDestinationServerName"`
}

type DependencyStatus struct {
	RequestID    int64  `db:"request_id"`
	RequestUID   string `db:"request_uid"`
	Status       string `db:"status"`
	StatusReason string `db:"status_reason"`
}

type CreateTargetParams struct {
	UID        string
	ServerID   int64
	TargetKind string
	Priority   int
	Status     string
}

type UpdateTargetParams struct {
	RequestID      int64
	ServerID       int64
	Status         string
	BlockedReason  string
	DeferredUntil  *time.Time
	LastReleasedAt *time.Time
}
