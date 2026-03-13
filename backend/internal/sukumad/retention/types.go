package retention

import (
	"context"
	"time"
)

type Candidate struct {
	RequestID   int64     `db:"request_id"`
	RequestUID  string    `db:"request_uid"`
	Status      string    `db:"status"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
	TargetCount int       `db:"target_count"`
}

type PurgeCounts struct {
	AsyncTaskPolls  int `json:"asyncTaskPolls"`
	AsyncTasks      int `json:"asyncTasks"`
	RequestEvents   int `json:"requestEvents"`
	DeliveryAttempts int `json:"deliveryAttempts"`
	RequestTargets  int `json:"requestTargets"`
	Dependencies    int `json:"dependencies"`
	Requests        int `json:"requests"`
}

type RunInput struct {
	Cutoff    time.Time
	BatchSize int
	DryRun    bool
}

type RunResult struct {
	Cutoff            time.Time    `json:"cutoff"`
	BatchSize         int          `json:"batchSize"`
	DryRun            bool         `json:"dryRun"`
	Scanned           int          `json:"scanned"`
	CandidateRequests []Candidate  `json:"candidateRequests"`
	Counts            PurgeCounts  `json:"counts"`
}

type Repository interface {
	ListPurgeCandidates(context.Context, time.Time, int) ([]Candidate, error)
	PurgeRequest(context.Context, int64) (PurgeCounts, error)
}
