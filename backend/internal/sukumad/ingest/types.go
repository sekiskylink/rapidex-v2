package ingest

import (
	"context"
	"errors"
	"time"

	requests "basepro/backend/internal/sukumad/request"
)

const (
	StatusDiscovered = "discovered"
	StatusRetry      = "retry"
	StatusProcessing = "processing"
	StatusProcessed  = "processed"
	StatusFailed     = "failed"
)

const (
	SourceKindDirectory = "directory"
)

const (
	ErrorCodeInvalidEnvelope = "INGEST_INVALID_ENVELOPE"
	ErrorCodeFileRead        = "INGEST_FILE_READ_ERROR"
	ErrorCodeFileMove        = "INGEST_FILE_MOVE_ERROR"
	ErrorCodeRequestCreate   = "INGEST_REQUEST_CREATE_ERROR"
	ErrorCodeDuplicate       = "INGEST_DUPLICATE_FILE"
)

var ErrNoEligibleFile = errors.New("no eligible ingest file")

type Record struct {
	ID                 int64          `db:"id" json:"id"`
	UID                string         `db:"uid" json:"uid"`
	SourceKind         string         `db:"source_kind" json:"sourceKind"`
	OriginalName       string         `db:"original_name" json:"originalName"`
	SourcePath         string         `db:"source_path" json:"sourcePath"`
	CurrentPath        string         `db:"current_path" json:"currentPath"`
	ArchivedPath       string         `db:"archived_path" json:"archivedPath"`
	Status             string         `db:"status" json:"status"`
	FileSize           int64          `db:"file_size" json:"fileSize"`
	ModifiedAt         *time.Time     `db:"modified_at" json:"modifiedAt,omitempty"`
	FirstSeenAt        time.Time      `db:"first_seen_at" json:"firstSeenAt"`
	LastSeenAt         time.Time      `db:"last_seen_at" json:"lastSeenAt"`
	ClaimedAt          *time.Time     `db:"claimed_at" json:"claimedAt,omitempty"`
	ClaimedByWorkerRun *int64         `db:"claimed_by_worker_run_id" json:"claimedByWorkerRunId,omitempty"`
	AttemptCount       int            `db:"attempt_count" json:"attemptCount"`
	NextAttemptAt      *time.Time     `db:"next_attempt_at" json:"nextAttemptAt,omitempty"`
	RequestID          *int64         `db:"request_id" json:"requestId,omitempty"`
	ChecksumSHA256     string         `db:"checksum_sha256" json:"checksumSha256"`
	IdempotencyKey     string         `db:"idempotency_key" json:"idempotencyKey"`
	LastErrorCode      string         `db:"last_error_code" json:"lastErrorCode"`
	LastErrorMessage   string         `db:"last_error_message" json:"lastErrorMessage"`
	ProcessedAt        *time.Time     `db:"processed_at" json:"processedAt,omitempty"`
	FailedAt           *time.Time     `db:"failed_at" json:"failedAt,omitempty"`
	Meta               map[string]any `json:"meta"`
	CreatedAt          time.Time      `db:"created_at" json:"createdAt"`
	UpdatedAt          time.Time      `db:"updated_at" json:"updatedAt"`
}

type UpsertParams struct {
	UID          string
	SourceKind   string
	OriginalName string
	SourcePath   string
	CurrentPath  string
	FileSize     int64
	ModifiedAt   *time.Time
	ObservedAt   time.Time
}

type ClaimParams struct {
	WorkerRunID int64
	ReadyBefore time.Time
	RetryBefore time.Time
	ClaimedAt   time.Time
}

type SetCurrentPathParams struct {
	ID          int64
	CurrentPath string
}

type MarkProcessedParams struct {
	ID             int64
	CurrentPath    string
	ArchivedPath   string
	RequestID      int64
	ChecksumSHA256 string
	IdempotencyKey string
	Meta           map[string]any
}

type MarkFailedParams struct {
	ID               int64
	CurrentPath      string
	ArchivedPath     string
	ChecksumSHA256   string
	IdempotencyKey   string
	LastErrorCode    string
	LastErrorMessage string
	Meta             map[string]any
}

type MarkRetryParams struct {
	ID               int64
	CurrentPath      string
	ChecksumSHA256   string
	IdempotencyKey   string
	LastErrorCode    string
	LastErrorMessage string
	NextAttemptAt    time.Time
	Meta             map[string]any
}

type RequeueParams struct {
	StaleBefore time.Time
	RetryAt     time.Time
}

type Repository interface {
	UpsertDiscovered(context.Context, UpsertParams) (Record, error)
	ClaimNextReady(context.Context, ClaimParams) (Record, error)
	SetCurrentPath(context.Context, SetCurrentPathParams) (Record, error)
	MarkProcessed(context.Context, MarkProcessedParams) (Record, error)
	MarkFailed(context.Context, MarkFailedParams) (Record, error)
	MarkRetry(context.Context, MarkRetryParams) (Record, error)
	RequeueStaleClaims(context.Context, RequeueParams) (int, error)
	GetByID(context.Context, int64) (Record, error)
}

type RuntimeConfig struct {
	Enabled               bool
	InboxPath             string
	ProcessingPath        string
	ProcessedPath         string
	FailedPath            string
	AllowedExtensions     []string
	DefaultSourceSystem   string
	RequireIdempotencyKey bool
	Debounce              time.Duration
	RetryDelay            time.Duration
	ClaimTimeout          time.Duration
	ScanInterval          time.Duration
	BatchSize             int
}

type requestCreator interface {
	CreateRequest(context.Context, requests.CreateInput) (requests.Record, error)
}
