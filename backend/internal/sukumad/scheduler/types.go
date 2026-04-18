package scheduler

import (
	"context"
	"time"
)

const (
	JobCategoryIntegration = "integration"
	JobCategoryMaintenance = "maintenance"
)

const (
	ScheduleTypeCron     = "cron"
	ScheduleTypeInterval = "interval"
)

const (
	TriggerModeScheduled = "scheduled"
	TriggerModeManual    = "manual"
)

const (
	RunStatusPending   = "pending"
	RunStatusRunning   = "running"
	RunStatusSucceeded = "succeeded"
	RunStatusFailed    = "failed"
	RunStatusCancelled = "cancelled"
	RunStatusSkipped   = "skipped"
)

type Record struct {
	ID                  int64          `db:"id" json:"id"`
	UID                 string         `db:"uid" json:"uid"`
	Code                string         `db:"code" json:"code"`
	Name                string         `db:"name" json:"name"`
	Description         string         `db:"description" json:"description"`
	JobCategory         string         `db:"job_category" json:"jobCategory"`
	JobType             string         `db:"job_type" json:"jobType"`
	ScheduleType        string         `db:"schedule_type" json:"scheduleType"`
	ScheduleExpr        string         `db:"schedule_expr" json:"scheduleExpr"`
	Timezone            string         `db:"timezone" json:"timezone"`
	Enabled             bool           `db:"enabled" json:"enabled"`
	AllowConcurrentRuns bool           `db:"allow_concurrent_runs" json:"allowConcurrentRuns"`
	Config              map[string]any `json:"config"`
	LastRunAt           *time.Time     `db:"last_run_at" json:"lastRunAt,omitempty"`
	NextRunAt           *time.Time     `db:"next_run_at" json:"nextRunAt,omitempty"`
	LastSuccessAt       *time.Time     `db:"last_success_at" json:"lastSuccessAt,omitempty"`
	LastFailureAt       *time.Time     `db:"last_failure_at" json:"lastFailureAt,omitempty"`
	CreatedAt           time.Time      `db:"created_at" json:"createdAt"`
	UpdatedAt           time.Time      `db:"updated_at" json:"updatedAt"`
}

type RunRecord struct {
	ID               int64          `db:"id" json:"id"`
	UID              string         `db:"uid" json:"uid"`
	ScheduledJobID   int64          `db:"scheduled_job_id" json:"scheduledJobId"`
	ScheduledJobUID  string         `json:"scheduledJobUid"`
	ScheduledJobCode string         `json:"scheduledJobCode"`
	ScheduledJobName string         `json:"scheduledJobName"`
	TriggerMode      string         `db:"trigger_mode" json:"triggerMode"`
	ScheduledFor     time.Time      `db:"scheduled_for" json:"scheduledFor"`
	StartedAt        *time.Time     `db:"started_at" json:"startedAt,omitempty"`
	FinishedAt       *time.Time     `db:"finished_at" json:"finishedAt,omitempty"`
	Status           string         `db:"status" json:"status"`
	WorkerID         *int64         `db:"worker_id" json:"workerId,omitempty"`
	ErrorMessage     string         `db:"error_message" json:"errorMessage"`
	ResultSummary    map[string]any `json:"resultSummary"`
	CreatedAt        time.Time      `db:"created_at" json:"createdAt"`
	UpdatedAt        time.Time      `db:"updated_at" json:"updatedAt"`
}

type ListQuery struct {
	Page      int
	PageSize  int
	SortField string
	SortOrder string
	Filter    string
	Category  string
}

type ListResult struct {
	Items    []Record `json:"items"`
	Total    int      `json:"totalCount"`
	Page     int      `json:"page"`
	PageSize int      `json:"pageSize"`
}

type RunListQuery struct {
	Page      int
	PageSize  int
	SortField string
	SortOrder string
	Status    string
}

type RunListResult struct {
	Items    []RunRecord `json:"items"`
	Total    int         `json:"totalCount"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}

type CreateParams struct {
	UID                 string
	Code                string
	Name                string
	Description         string
	JobCategory         string
	JobType             string
	ScheduleType        string
	ScheduleExpr        string
	Timezone            string
	Enabled             bool
	AllowConcurrentRuns bool
	Config              map[string]any
	NextRunAt           *time.Time
}

type UpdateParams struct {
	ID                  int64
	Code                string
	Name                string
	Description         string
	JobCategory         string
	JobType             string
	ScheduleType        string
	ScheduleExpr        string
	Timezone            string
	Enabled             bool
	AllowConcurrentRuns bool
	Config              map[string]any
	NextRunAt           *time.Time
}

type SetEnabledParams struct {
	ID        int64
	Enabled   bool
	NextRunAt *time.Time
}

type CreateRunParams struct {
	UID            string
	ScheduledJobID int64
	TriggerMode    string
	ScheduledFor   time.Time
	Status         string
	ResultSummary  map[string]any
}

type CreateInput struct {
	Code                string
	Name                string
	Description         string
	JobCategory         string
	JobType             string
	ScheduleType        string
	ScheduleExpr        string
	Timezone            string
	Enabled             bool
	AllowConcurrentRuns bool
	Config              map[string]any
	ActorID             *int64
}

type UpdateInput struct {
	ID                  int64
	Code                string
	Name                string
	Description         string
	JobCategory         string
	JobType             string
	ScheduleType        string
	ScheduleExpr        string
	Timezone            string
	Enabled             bool
	AllowConcurrentRuns bool
	Config              map[string]any
	ActorID             *int64
}

type Repository interface {
	ListScheduledJobs(ctx context.Context, query ListQuery) (ListResult, error)
	GetScheduledJobByID(ctx context.Context, id int64) (Record, error)
	CreateScheduledJob(ctx context.Context, params CreateParams) (Record, error)
	UpdateScheduledJob(ctx context.Context, params UpdateParams) (Record, error)
	SetScheduledJobEnabled(ctx context.Context, params SetEnabledParams) (Record, error)
	ListJobRuns(ctx context.Context, jobID int64, query RunListQuery) (RunListResult, error)
	GetRunByID(ctx context.Context, id int64) (RunRecord, error)
	CreateJobRun(ctx context.Context, params CreateRunParams) (RunRecord, error)
}
