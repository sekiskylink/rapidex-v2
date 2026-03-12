package worker

import (
	"context"
	"time"
)

const (
	TypeSend  = "send"
	TypePoll  = "poll"
	TypeRetry = "retry"
)

const (
	StatusStarting = "starting"
	StatusRunning  = "running"
	StatusStopped  = "stopped"
	StatusFailed   = "failed"
)

type Record struct {
	ID              int64          `db:"id" json:"id"`
	UID             string         `db:"uid" json:"uid"`
	WorkerType      string         `db:"worker_type" json:"workerType"`
	WorkerName      string         `db:"worker_name" json:"workerName"`
	Status          string         `db:"status" json:"status"`
	StartedAt       time.Time      `db:"started_at" json:"startedAt"`
	StoppedAt       *time.Time     `db:"stopped_at" json:"stoppedAt,omitempty"`
	LastHeartbeatAt *time.Time     `db:"last_heartbeat_at" json:"lastHeartbeatAt,omitempty"`
	Meta            map[string]any `json:"meta"`
	CreatedAt       time.Time      `db:"created_at" json:"createdAt"`
	UpdatedAt       time.Time      `db:"updated_at" json:"updatedAt"`
}

type CreateParams struct {
	UID        string
	WorkerType string
	WorkerName string
	Status     string
	StartedAt  time.Time
	Meta       map[string]any
}

type UpdateParams struct {
	ID              int64
	Status          string
	StoppedAt       *time.Time
	LastHeartbeatAt *time.Time
	Meta            map[string]any
}

type ListQuery struct {
	Page      int
	PageSize  int
	SortField string
	SortOrder string
}

type ListResult struct {
	Items    []Record
	Total    int
	Page     int
	PageSize int
}

type Repository interface {
	ListRuns(context.Context, ListQuery) (ListResult, error)
	GetRunByID(context.Context, int64) (Record, error)
	CreateRun(context.Context, CreateParams) (Record, error)
	UpdateRun(context.Context, UpdateParams) (Record, error)
}

type Execution struct {
	RunID int64
}

type Definition struct {
	Type              string
	Name              string
	Interval          time.Duration
	HeartbeatInterval time.Duration
	Run               func(context.Context, Execution) error
	Meta              map[string]any
}
