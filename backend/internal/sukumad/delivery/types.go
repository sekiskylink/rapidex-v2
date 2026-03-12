package delivery

import (
	"context"
	"time"
)

const (
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
	StatusRetrying  = "retrying"
)

type Record struct {
	ID            int64      `db:"id" json:"id"`
	UID           string     `db:"uid" json:"uid"`
	RequestID     int64      `db:"request_id" json:"requestId"`
	RequestUID    string     `json:"requestUid"`
	ServerID      int64      `db:"server_id" json:"serverId"`
	ServerName    string     `json:"serverName"`
	AttemptNumber int        `db:"attempt_number" json:"attemptNumber"`
	Status        string     `db:"status" json:"status"`
	HTTPStatus    *int       `db:"http_status" json:"httpStatus,omitempty"`
	ResponseBody  string     `db:"response_body" json:"responseBody"`
	ErrorMessage  string     `db:"error_message" json:"errorMessage"`
	StartedAt     *time.Time `db:"started_at" json:"startedAt,omitempty"`
	FinishedAt    *time.Time `db:"finished_at" json:"finishedAt,omitempty"`
	RetryAt       *time.Time `db:"retry_at" json:"retryAt,omitempty"`
	CreatedAt     time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt     time.Time  `db:"updated_at" json:"updatedAt"`
}

type ListQuery struct {
	Page      int
	PageSize  int
	SortField string
	SortOrder string
	Filter    string
	Status    string
	Server    string
	Date      *time.Time
}

type ListResult struct {
	Items    []Record
	Total    int
	Page     int
	PageSize int
}

type CreateParams struct {
	UID           string
	RequestID     int64
	ServerID      int64
	AttemptNumber int
	Status        string
	HTTPStatus    *int
	ResponseBody  string
	ErrorMessage  string
	StartedAt     *time.Time
	FinishedAt    *time.Time
	RetryAt       *time.Time
}

type UpdateParams struct {
	ID           int64
	Status       string
	HTTPStatus   *int
	ResponseBody string
	ErrorMessage string
	StartedAt    *time.Time
	FinishedAt   *time.Time
	RetryAt      *time.Time
}

type CreateInput struct {
	RequestID int64
	ServerID  int64
	ActorID   *int64
}

type CompletionInput struct {
	ID           int64
	HTTPStatus   *int
	ResponseBody string
	ErrorMessage string
	ActorID      *int64
}

type Repository interface {
	ListDeliveries(ctx context.Context, query ListQuery) (ListResult, error)
	GetDeliveryByID(ctx context.Context, id int64) (Record, error)
	CreateDelivery(ctx context.Context, params CreateParams) (Record, error)
	UpdateDelivery(ctx context.Context, params UpdateParams) (Record, error)
}
