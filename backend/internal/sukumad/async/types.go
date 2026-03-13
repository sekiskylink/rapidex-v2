package async

import (
	"context"
	"errors"
	"time"
)

const (
	StatePending   = "pending"
	StatePolling   = "polling"
	StateSucceeded = "succeeded"
	StateFailed    = "failed"
)

var ErrNoEligibleTask = errors.New("no eligible async task")

type Record struct {
	ID                 int64          `db:"id" json:"id"`
	UID                string         `db:"uid" json:"uid"`
	DeliveryAttemptID  int64          `db:"delivery_attempt_id" json:"deliveryAttemptId"`
	DeliveryUID        string         `json:"deliveryUid"`
	RequestID          int64          `json:"requestId"`
	RequestUID         string         `json:"requestUid"`
	CorrelationID      string         `json:"correlationId"`
	DestinationCode    string         `json:"destinationCode"`
	RemoteJobID        string         `db:"remote_job_id" json:"remoteJobId"`
	PollURL            string         `db:"poll_url" json:"pollUrl"`
	RemoteStatus       string         `db:"remote_status" json:"remoteStatus"`
	TerminalState      string         `db:"terminal_state" json:"terminalState"`
	CurrentState       string         `json:"currentState"`
	NextPollAt         *time.Time     `db:"next_poll_at" json:"nextPollAt,omitempty"`
	CompletedAt        *time.Time     `db:"completed_at" json:"completedAt,omitempty"`
	PollClaimedAt      *time.Time     `db:"poll_claimed_at" json:"pollClaimedAt,omitempty"`
	PollClaimedByRunID *int64         `db:"poll_claimed_by_worker_run_id" json:"pollClaimedByWorkerRunId,omitempty"`
	RemoteResponse     map[string]any `json:"remoteResponse"`
	CreatedAt          time.Time      `db:"created_at" json:"createdAt"`
	UpdatedAt          time.Time      `db:"updated_at" json:"updatedAt"`
}

type PollRecord struct {
	ID                   int64     `db:"id" json:"id"`
	AsyncTaskID          int64     `db:"async_task_id" json:"asyncTaskId"`
	PolledAt             time.Time `db:"polled_at" json:"polledAt"`
	StatusCode           *int      `db:"status_code" json:"statusCode,omitempty"`
	RemoteStatus         string    `db:"remote_status" json:"remoteStatus"`
	ResponseBody         string    `db:"response_body" json:"responseBody"`
	ResponseContentType  string    `db:"response_content_type" json:"responseContentType"`
	ResponseBodyFiltered bool      `db:"response_body_filtered" json:"responseBodyFiltered"`
	ErrorMessage         string    `db:"error_message" json:"errorMessage"`
	DurationMS           *int      `db:"duration_ms" json:"durationMs,omitempty"`
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

type PollListResult struct {
	Items    []PollRecord
	Total    int
	Page     int
	PageSize int
}

type CreateParams struct {
	UID               string
	DeliveryAttemptID int64
	RemoteJobID       string
	PollURL           string
	RemoteStatus      string
	TerminalState     string
	NextPollAt        *time.Time
	CompletedAt       *time.Time
	RemoteResponse    map[string]any
}

type UpdateParams struct {
	ID             int64
	RemoteJobID    string
	PollURL        string
	RemoteStatus   string
	TerminalState  string
	NextPollAt     *time.Time
	CompletedAt    *time.Time
	RemoteResponse map[string]any
}

type CreateInput struct {
	DeliveryAttemptID int64
	RemoteJobID       string
	PollURL           string
	RemoteStatus      string
	NextPollAt        *time.Time
	RemoteResponse    map[string]any
	ActorID           *int64
}

type UpdateStatusInput struct {
	ID             int64
	RemoteJobID    string
	PollURL        string
	RemoteStatus   string
	TerminalState  string
	NextPollAt     *time.Time
	RemoteResponse map[string]any
	ActorID        *int64
}

type RecordPollInput struct {
	AsyncTaskID          int64
	StatusCode           *int
	RemoteStatus         string
	ResponseBody         string
	ResponseContentType  string
	ResponseBodyFiltered bool
	ErrorMessage         string
	DurationMS           *int
}

type RemotePollResult struct {
	StatusCode           *int
	RemoteStatus         string
	TerminalState        string
	ResponseBody         string
	ResponseContentType  string
	ResponseBodyFiltered bool
	ErrorMessage         string
	DurationMS           *int
	NextPollAt           *time.Time
	RemoteResponse       map[string]any
}

type RemotePoller interface {
	Poll(context.Context, Record) (RemotePollResult, error)
}

type PollExecution struct {
	WorkerRunID  int64
	ClaimTimeout time.Duration
	Observe      func(string, int)
}

func (p PollExecution) Increment(name string) {
	if p.Observe != nil && name != "" {
		p.Observe(name, 1)
	}
}

type Repository interface {
	ListTasks(context.Context, ListQuery) (ListResult, error)
	GetTaskByID(context.Context, int64) (Record, error)
	CreateTask(context.Context, CreateParams) (Record, error)
	UpdateTask(context.Context, UpdateParams) (Record, error)
	ListPolls(context.Context, int64, ListQuery) (PollListResult, error)
	RecordPoll(context.Context, RecordPollInput) (PollRecord, error)
	ListDueTasks(context.Context, time.Time, int) ([]Record, error)
	ClaimNextDueTask(context.Context, time.Time, time.Duration, int64) (Record, error)
	ListTerminalTasksForRecovery(context.Context, int) ([]Record, error)
}
