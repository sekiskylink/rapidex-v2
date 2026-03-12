package request

import (
	"context"
	"encoding/json"
	"time"
)

const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

type Record struct {
	ID                    int64           `db:"id" json:"id"`
	UID                   string          `db:"uid" json:"uid"`
	SourceSystem          string          `db:"source_system" json:"sourceSystem"`
	DestinationServerID   int64           `db:"destination_server_id" json:"destinationServerId"`
	DestinationServerName string          `json:"destinationServerName"`
	BatchID               string          `db:"batch_id" json:"batchId"`
	CorrelationID         string          `db:"correlation_id" json:"correlationId"`
	IdempotencyKey        string          `db:"idempotency_key" json:"idempotencyKey"`
	PayloadBody           string          `db:"payload_body" json:"payloadBody"`
	PayloadFormat         string          `db:"payload_format" json:"payloadFormat"`
	URLSuffix             string          `db:"url_suffix" json:"urlSuffix"`
	Status                string          `db:"status" json:"status"`
	Extras                map[string]any  `json:"extras"`
	CreatedAt             time.Time       `db:"created_at" json:"createdAt"`
	UpdatedAt             time.Time       `db:"updated_at" json:"updatedAt"`
	CreatedBy             *int64          `db:"created_by" json:"createdBy,omitempty"`
	Payload               json.RawMessage `json:"payload"`
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
	URLSuffix           string
	Status              string
	Extras              map[string]any
	CreatedBy           *int64
}

type CreateInput struct {
	SourceSystem        string
	DestinationServerID int64
	BatchID             string
	CorrelationID       string
	IdempotencyKey      string
	Payload             json.RawMessage
	URLSuffix           string
	Extras              map[string]any
	ActorID             *int64
}

type Repository interface {
	ListRequests(ctx context.Context, query ListQuery) (ListResult, error)
	GetRequestByID(ctx context.Context, id int64) (Record, error)
	CreateRequest(ctx context.Context, params CreateParams) (Record, error)
}
