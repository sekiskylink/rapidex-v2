package delivery

import (
	"context"
	"errors"
	"time"
)

const (
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
	StatusRetrying  = "retrying"
)

var ErrNoEligibleDelivery = errors.New("no eligible delivery")

type Record struct {
	ID                   int64          `db:"id" json:"id"`
	UID                  string         `db:"uid" json:"uid"`
	RequestID            int64          `db:"request_id" json:"requestId"`
	RequestUID           string         `json:"requestUid"`
	CorrelationID        string         `json:"correlationId"`
	ServerID             int64          `db:"server_id" json:"serverId"`
	ServerName           string         `json:"serverName"`
	ServerCode           string         `json:"serverCode"`
	SystemType           string         `json:"systemType"`
	AttemptNumber        int            `db:"attempt_number" json:"attemptNumber"`
	Status               string         `db:"status" json:"status"`
	HTTPStatus           *int           `db:"http_status" json:"httpStatus,omitempty"`
	ResponseBody         string         `db:"response_body" json:"responseBody"`
	ResponseContentType  string         `db:"response_content_type" json:"responseContentType"`
	ResponseBodyFiltered bool           `db:"response_body_filtered" json:"responseBodyFiltered"`
	ResponseSummary      map[string]any `json:"responseSummary"`
	ErrorMessage         string         `db:"error_message" json:"errorMessage"`
	SubmissionHoldReason string         `db:"submission_hold_reason" json:"submissionHoldReason"`
	NextEligibleAt       *time.Time     `db:"next_eligible_at" json:"nextEligibleAt,omitempty"`
	HoldPolicySource     string         `db:"hold_policy_source" json:"holdPolicySource"`
	TerminalReason       string         `db:"terminal_reason" json:"terminalReason"`
	SubmissionMode       string         `json:"submissionMode"`
	AsyncTaskID          *int64         `json:"asyncTaskId,omitempty"`
	AsyncTaskUID         string         `json:"asyncTaskUid"`
	AsyncCurrentState    string         `json:"asyncCurrentState"`
	AsyncRemoteJobID     string         `json:"asyncRemoteJobId"`
	AsyncPollURL         string         `json:"asyncPollUrl"`
	AwaitingAsync        bool           `json:"awaitingAsync"`
	StartedAt            *time.Time     `db:"started_at" json:"startedAt,omitempty"`
	FinishedAt           *time.Time     `db:"finished_at" json:"finishedAt,omitempty"`
	RetryAt              *time.Time     `db:"retry_at" json:"retryAt,omitempty"`
	CreatedAt            time.Time      `db:"created_at" json:"createdAt"`
	UpdatedAt            time.Time      `db:"updated_at" json:"updatedAt"`
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
	UID                  string
	RequestID            int64
	ServerID             int64
	AttemptNumber        int
	Status               string
	HTTPStatus           *int
	ResponseBody         string
	ResponseContentType  string
	ResponseBodyFiltered bool
	ResponseSummary      map[string]any
	ErrorMessage         string
	SubmissionHoldReason string
	NextEligibleAt       *time.Time
	HoldPolicySource     string
	TerminalReason       string
	StartedAt            *time.Time
	FinishedAt           *time.Time
	RetryAt              *time.Time
}

type UpdateParams struct {
	ID                   int64
	Status               string
	HTTPStatus           *int
	ResponseBody         string
	ResponseContentType  string
	ResponseBodyFiltered bool
	ResponseSummary      map[string]any
	ErrorMessage         string
	SubmissionHoldReason string
	NextEligibleAt       *time.Time
	HoldPolicySource     string
	TerminalReason       string
	StartedAt            *time.Time
	FinishedAt           *time.Time
	RetryAt              *time.Time
}

type CreateInput struct {
	RequestID     int64
	ServerID      int64
	CorrelationID string
	ActorID       *int64
}

type CompletionInput struct {
	ID                   int64
	HTTPStatus           *int
	ResponseBody         string
	ResponseContentType  string
	ResponseBodyFiltered bool
	ResponseSummary      map[string]any
	ErrorMessage         string
	ActorID              *int64
}

type ServerSnapshot struct {
	ID                        int64
	Code                      string
	Name                      string
	SystemType                string
	BaseURL                   string
	EndpointType              string
	HTTPMethod                string
	UseAsync                  bool
	Headers                   map[string]string
	URLParams                 map[string]string
	SubmissionWindowStartHour int
	SubmissionWindowEndHour   int
}

type DispatchInput struct {
	DeliveryID    int64
	RequestID     int64
	RequestUID    string
	CorrelationID string
	PayloadBody   string
	URLSuffix     string
	Server        ServerSnapshot
	ActorID       *int64
}

type Repository interface {
	ListDeliveries(ctx context.Context, query ListQuery) (ListResult, error)
	GetDeliveryByID(ctx context.Context, id int64) (Record, error)
	CreateDelivery(ctx context.Context, params CreateParams) (Record, error)
	UpdateDelivery(ctx context.Context, params UpdateParams) (Record, error)
	ClaimNextPendingDelivery(ctx context.Context, now time.Time) (Record, error)
	ClaimNextRetryDelivery(ctx context.Context, now time.Time) (Record, error)
}
