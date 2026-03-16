package ingest

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
)

type SQLRepository struct {
	db *sqlx.DB
}

func NewSQLRepository(db *sqlx.DB) *SQLRepository {
	return &SQLRepository{db: db}
}

func NewRepository(db ...*sqlx.DB) Repository {
	if len(db) > 0 && db[0] != nil {
		return NewSQLRepository(db[0])
	}
	return newMemoryRepository()
}

type row struct {
	ID                 int64           `db:"id"`
	UID                string          `db:"uid"`
	SourceKind         string          `db:"source_kind"`
	OriginalName       string          `db:"original_name"`
	SourcePath         string          `db:"source_path"`
	CurrentPath        string          `db:"current_path"`
	ArchivedPath       sql.NullString  `db:"archived_path"`
	Status             string          `db:"status"`
	FileSize           int64           `db:"file_size"`
	ModifiedAt         *time.Time      `db:"modified_at"`
	FirstSeenAt        time.Time       `db:"first_seen_at"`
	LastSeenAt         time.Time       `db:"last_seen_at"`
	ClaimedAt          *time.Time      `db:"claimed_at"`
	ClaimedByWorkerRun *int64          `db:"claimed_by_worker_run_id"`
	AttemptCount       int             `db:"attempt_count"`
	NextAttemptAt      *time.Time      `db:"next_attempt_at"`
	RequestID          *int64          `db:"request_id"`
	ChecksumSHA256     sql.NullString  `db:"checksum_sha256"`
	IdempotencyKey     sql.NullString  `db:"idempotency_key"`
	LastErrorCode      sql.NullString  `db:"last_error_code"`
	LastErrorMessage   sql.NullString  `db:"last_error_message"`
	ProcessedAt        *time.Time      `db:"processed_at"`
	FailedAt           *time.Time      `db:"failed_at"`
	Meta               json.RawMessage `db:"meta"`
	CreatedAt          time.Time       `db:"created_at"`
	UpdatedAt          time.Time       `db:"updated_at"`
}

const selectColumns = `
	SELECT id, uid::text AS uid, source_kind, original_name, source_path, current_path, archived_path, status, file_size,
	       modified_at, first_seen_at, last_seen_at, claimed_at, claimed_by_worker_run_id, attempt_count, next_attempt_at,
	       request_id, checksum_sha256, idempotency_key, last_error_code, last_error_message, processed_at, failed_at,
	       meta, created_at, updated_at
	FROM ingest_files
`

func (r *SQLRepository) UpsertDiscovered(ctx context.Context, params UpsertParams) (Record, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return Record{}, fmt.Errorf("begin ingest upsert: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var existing row
	err = tx.GetContext(ctx, &existing, selectColumns+`
		WHERE current_path = $1 AND status IN ('discovered', 'retry', 'processing')
		ORDER BY id DESC
		LIMIT 1
		FOR UPDATE
	`, params.CurrentPath)
	if err == nil {
		sameFile := existing.FileSize == params.FileSize && sameTime(existing.ModifiedAt, params.ModifiedAt)
		if sameFile {
			return decodeRow(existing)
		}
		var id int64
		if err := tx.GetContext(ctx, &id, `
			UPDATE ingest_files
			SET source_kind = $2,
			    original_name = $3,
			    source_path = $4,
			    file_size = $5,
			    modified_at = $6,
			    status = $7,
			    last_seen_at = $8,
			    next_attempt_at = NULL,
			    claimed_at = NULL,
			    claimed_by_worker_run_id = NULL,
			    last_error_code = NULL,
			    last_error_message = NULL,
			    checksum_sha256 = NULL,
			    idempotency_key = NULL,
			    meta = '{}'::jsonb,
			    updated_at = NOW()
			WHERE id = $1
			RETURNING id
		`, existing.ID, params.SourceKind, params.OriginalName, params.SourcePath, params.FileSize, params.ModifiedAt, StatusDiscovered, params.ObservedAt); err != nil {
			return Record{}, fmt.Errorf("update discovered ingest file: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return Record{}, fmt.Errorf("commit ingest upsert: %w", err)
		}
		return r.GetByID(ctx, id)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Record{}, fmt.Errorf("lookup ingest file: %w", err)
	}

	var id int64
	if err := tx.GetContext(ctx, &id, `
		INSERT INTO ingest_files (
			uid, source_kind, original_name, source_path, current_path, status, file_size, modified_at, first_seen_at,
			last_seen_at, attempt_count, meta, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9, 0, '{}'::jsonb, NOW(), NOW())
		RETURNING id
	`, firstNonEmpty(params.UID, newUID()), params.SourceKind, params.OriginalName, params.SourcePath, params.CurrentPath, StatusDiscovered, params.FileSize, params.ModifiedAt, params.ObservedAt); err != nil {
		return Record{}, fmt.Errorf("insert ingest file: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Record{}, fmt.Errorf("commit ingest insert: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *SQLRepository) ClaimNextReady(ctx context.Context, params ClaimParams) (Record, error) {
	var id int64
	err := r.db.GetContext(ctx, &id, `
		WITH candidate AS (
			SELECT id
			FROM ingest_files
			WHERE status IN ('discovered', 'retry')
			  AND last_seen_at <= $1
			  AND COALESCE(next_attempt_at, TIMESTAMPTZ '-infinity') <= $2
			ORDER BY COALESCE(next_attempt_at, first_seen_at), id
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		UPDATE ingest_files AS f
		SET status = 'processing',
		    claimed_at = $3,
		    claimed_by_worker_run_id = $4,
		    attempt_count = attempt_count + 1,
		    updated_at = NOW()
		FROM candidate
		WHERE f.id = candidate.id
		RETURNING f.id
	`, params.ReadyBefore, params.RetryBefore, params.ClaimedAt, params.WorkerRunID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, ErrNoEligibleFile
		}
		return Record{}, fmt.Errorf("claim ingest file: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *SQLRepository) SetCurrentPath(ctx context.Context, params SetCurrentPathParams) (Record, error) {
	var id int64
	err := r.db.GetContext(ctx, &id, `
		UPDATE ingest_files
		SET current_path = $2,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`, params.ID, params.CurrentPath)
	if err != nil {
		return Record{}, fmt.Errorf("set ingest current path: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *SQLRepository) MarkProcessed(ctx context.Context, params MarkProcessedParams) (Record, error) {
	return r.updateTerminal(ctx, params.ID, `
		UPDATE ingest_files
		SET current_path = $2,
		    archived_path = $3,
		    status = 'processed',
		    request_id = $4,
		    checksum_sha256 = NULLIF($5, ''),
		    idempotency_key = NULLIF($6, ''),
		    processed_at = NOW(),
		    claimed_at = NULL,
		    claimed_by_worker_run_id = NULL,
		    next_attempt_at = NULL,
		    last_error_code = NULL,
		    last_error_message = NULL,
		    meta = $7::jsonb,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`, params.CurrentPath, params.ArchivedPath, params.RequestID, params.ChecksumSHA256, params.IdempotencyKey, cloneMetaJSON(params.Meta))
}

func (r *SQLRepository) MarkFailed(ctx context.Context, params MarkFailedParams) (Record, error) {
	return r.updateTerminal(ctx, params.ID, `
		UPDATE ingest_files
		SET current_path = $2,
		    archived_path = $3,
		    status = 'failed',
		    checksum_sha256 = NULLIF($4, ''),
		    idempotency_key = NULLIF($5, ''),
		    failed_at = NOW(),
		    claimed_at = NULL,
		    claimed_by_worker_run_id = NULL,
		    next_attempt_at = NULL,
		    last_error_code = NULLIF($6, ''),
		    last_error_message = NULLIF($7, ''),
		    meta = $8::jsonb,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`, params.CurrentPath, params.ArchivedPath, params.ChecksumSHA256, params.IdempotencyKey, params.LastErrorCode, params.LastErrorMessage, cloneMetaJSON(params.Meta))
}

func (r *SQLRepository) MarkRetry(ctx context.Context, params MarkRetryParams) (Record, error) {
	var id int64
	err := r.db.GetContext(ctx, &id, `
		UPDATE ingest_files
		SET current_path = $2,
		    status = 'retry',
		    checksum_sha256 = NULLIF($3, ''),
		    idempotency_key = NULLIF($4, ''),
		    next_attempt_at = $5,
		    claimed_at = NULL,
		    claimed_by_worker_run_id = NULL,
		    last_error_code = NULLIF($6, ''),
		    last_error_message = NULLIF($7, ''),
		    meta = $8::jsonb,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`, params.ID, params.CurrentPath, params.ChecksumSHA256, params.IdempotencyKey, params.NextAttemptAt, params.LastErrorCode, params.LastErrorMessage, cloneMetaJSON(params.Meta))
	if err != nil {
		return Record{}, fmt.Errorf("mark ingest retry: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *SQLRepository) RequeueStaleClaims(ctx context.Context, params RequeueParams) (int, error) {
	result, err := r.db.ExecContext(ctx, `
		UPDATE ingest_files
		SET status = 'retry',
		    next_attempt_at = $2,
		    claimed_at = NULL,
		    claimed_by_worker_run_id = NULL,
		    last_error_code = $3,
		    last_error_message = $4,
		    updated_at = NOW()
		WHERE status = 'processing'
		  AND claimed_at IS NOT NULL
		  AND claimed_at <= $1
	`, params.StaleBefore, params.RetryAt, ErrorCodeFileMove, "stale processing claim requeued")
	if err != nil {
		return 0, fmt.Errorf("requeue stale ingest claims: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count stale ingest requeue rows: %w", err)
	}
	return int(count), nil
}

func (r *SQLRepository) GetByID(ctx context.Context, id int64) (Record, error) {
	var item row
	if err := r.db.GetContext(ctx, &item, selectColumns+` WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, sql.ErrNoRows
		}
		return Record{}, fmt.Errorf("get ingest file: %w", err)
	}
	return decodeRow(item)
}

func (r *SQLRepository) updateTerminal(ctx context.Context, id int64, query string, args ...any) (Record, error) {
	params := append([]any{id}, args...)
	var updatedID int64
	if err := r.db.GetContext(ctx, &updatedID, query, params...); err != nil {
		return Record{}, fmt.Errorf("update terminal ingest file: %w", err)
	}
	return r.GetByID(ctx, updatedID)
}

func decodeRow(item row) (Record, error) {
	meta := map[string]any{}
	if len(item.Meta) > 0 {
		if err := json.Unmarshal(item.Meta, &meta); err != nil {
			return Record{}, fmt.Errorf("decode ingest meta: %w", err)
		}
	}
	return Record{
		ID:                 item.ID,
		UID:                item.UID,
		SourceKind:         item.SourceKind,
		OriginalName:       item.OriginalName,
		SourcePath:         item.SourcePath,
		CurrentPath:        item.CurrentPath,
		ArchivedPath:       item.ArchivedPath.String,
		Status:             item.Status,
		FileSize:           item.FileSize,
		ModifiedAt:         cloneTimePtr(item.ModifiedAt),
		FirstSeenAt:        item.FirstSeenAt,
		LastSeenAt:         item.LastSeenAt,
		ClaimedAt:          cloneTimePtr(item.ClaimedAt),
		ClaimedByWorkerRun: cloneInt64Ptr(item.ClaimedByWorkerRun),
		AttemptCount:       item.AttemptCount,
		NextAttemptAt:      cloneTimePtr(item.NextAttemptAt),
		RequestID:          cloneInt64Ptr(item.RequestID),
		ChecksumSHA256:     item.ChecksumSHA256.String,
		IdempotencyKey:     item.IdempotencyKey.String,
		LastErrorCode:      item.LastErrorCode.String,
		LastErrorMessage:   item.LastErrorMessage.String,
		ProcessedAt:        cloneTimePtr(item.ProcessedAt),
		FailedAt:           cloneTimePtr(item.FailedAt),
		Meta:               cloneJSONMap(meta),
		CreatedAt:          item.CreatedAt,
		UpdatedAt:          item.UpdatedAt,
	}, nil
}

type memoryRepository struct {
	mu     sync.RWMutex
	nextID int64
	items  map[int64]Record
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{nextID: 1, items: map[int64]Record{}}
}

func (r *memoryRepository) UpsertDiscovered(_ context.Context, params UpsertParams) (Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, item := range r.items {
		if item.CurrentPath != params.CurrentPath || !isActive(item.Status) {
			continue
		}
		sameFile := item.FileSize == params.FileSize && sameTime(item.ModifiedAt, params.ModifiedAt)
		if sameFile {
			return cloneRecord(item), nil
		}
		item.SourceKind = params.SourceKind
		item.OriginalName = params.OriginalName
		item.SourcePath = params.SourcePath
		item.FileSize = params.FileSize
		item.ModifiedAt = cloneTimePtr(params.ModifiedAt)
		item.Status = StatusDiscovered
		item.LastSeenAt = params.ObservedAt
		item.NextAttemptAt = nil
		item.ClaimedAt = nil
		item.ClaimedByWorkerRun = nil
		item.LastErrorCode = ""
		item.LastErrorMessage = ""
		item.ChecksumSHA256 = ""
		item.IdempotencyKey = ""
		item.Meta = map[string]any{}
		item.UpdatedAt = time.Now().UTC()
		r.items[id] = item
		return cloneRecord(item), nil
	}
	now := time.Now().UTC()
	id := r.nextID
	r.nextID++
	item := Record{
		ID:           id,
		UID:          firstNonEmpty(params.UID, newUID()),
		SourceKind:   params.SourceKind,
		OriginalName: params.OriginalName,
		SourcePath:   params.SourcePath,
		CurrentPath:  params.CurrentPath,
		Status:       StatusDiscovered,
		FileSize:     params.FileSize,
		ModifiedAt:   cloneTimePtr(params.ModifiedAt),
		FirstSeenAt:  params.ObservedAt,
		LastSeenAt:   params.ObservedAt,
		Meta:         map[string]any{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	r.items[id] = item
	return cloneRecord(item), nil
}

func (r *memoryRepository) ClaimNextReady(_ context.Context, params ClaimParams) (Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := make([]Record, 0, len(r.items))
	for _, item := range r.items {
		if item.Status != StatusDiscovered && item.Status != StatusRetry {
			continue
		}
		if item.LastSeenAt.After(params.ReadyBefore) {
			continue
		}
		if item.NextAttemptAt != nil && item.NextAttemptAt.After(params.RetryBefore) {
			continue
		}
		items = append(items, cloneRecord(item))
	}
	slices.SortFunc(items, func(a, b Record) int {
		left := a.FirstSeenAt
		if a.NextAttemptAt != nil {
			left = *a.NextAttemptAt
		}
		right := b.FirstSeenAt
		if b.NextAttemptAt != nil {
			right = *b.NextAttemptAt
		}
		return left.Compare(right)
	})
	if len(items) == 0 {
		return Record{}, ErrNoEligibleFile
	}
	item := items[0]
	stamp := params.ClaimedAt
	item.Status = StatusProcessing
	item.ClaimedAt = &stamp
	item.ClaimedByWorkerRun = int64Ptr(params.WorkerRunID)
	item.AttemptCount++
	item.UpdatedAt = time.Now().UTC()
	r.items[item.ID] = item
	return cloneRecord(item), nil
}

func (r *memoryRepository) SetCurrentPath(_ context.Context, params SetCurrentPathParams) (Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[params.ID]
	if !ok {
		return Record{}, sql.ErrNoRows
	}
	item.CurrentPath = params.CurrentPath
	item.UpdatedAt = time.Now().UTC()
	r.items[params.ID] = item
	return cloneRecord(item), nil
}

func (r *memoryRepository) MarkProcessed(_ context.Context, params MarkProcessedParams) (Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[params.ID]
	if !ok {
		return Record{}, sql.ErrNoRows
	}
	now := time.Now().UTC()
	item.CurrentPath = params.CurrentPath
	item.ArchivedPath = params.ArchivedPath
	item.Status = StatusProcessed
	item.RequestID = int64Ptr(params.RequestID)
	item.ChecksumSHA256 = params.ChecksumSHA256
	item.IdempotencyKey = params.IdempotencyKey
	item.ProcessedAt = &now
	item.ClaimedAt = nil
	item.ClaimedByWorkerRun = nil
	item.NextAttemptAt = nil
	item.LastErrorCode = ""
	item.LastErrorMessage = ""
	item.Meta = cloneJSONMap(params.Meta)
	item.UpdatedAt = now
	r.items[params.ID] = item
	return cloneRecord(item), nil
}

func (r *memoryRepository) MarkFailed(_ context.Context, params MarkFailedParams) (Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[params.ID]
	if !ok {
		return Record{}, sql.ErrNoRows
	}
	now := time.Now().UTC()
	item.CurrentPath = params.CurrentPath
	item.ArchivedPath = params.ArchivedPath
	item.Status = StatusFailed
	item.ChecksumSHA256 = params.ChecksumSHA256
	item.IdempotencyKey = params.IdempotencyKey
	item.LastErrorCode = params.LastErrorCode
	item.LastErrorMessage = params.LastErrorMessage
	item.FailedAt = &now
	item.ClaimedAt = nil
	item.ClaimedByWorkerRun = nil
	item.NextAttemptAt = nil
	item.Meta = cloneJSONMap(params.Meta)
	item.UpdatedAt = now
	r.items[params.ID] = item
	return cloneRecord(item), nil
}

func (r *memoryRepository) MarkRetry(_ context.Context, params MarkRetryParams) (Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[params.ID]
	if !ok {
		return Record{}, sql.ErrNoRows
	}
	item.CurrentPath = params.CurrentPath
	item.Status = StatusRetry
	item.ChecksumSHA256 = params.ChecksumSHA256
	item.IdempotencyKey = params.IdempotencyKey
	item.LastErrorCode = params.LastErrorCode
	item.LastErrorMessage = params.LastErrorMessage
	nextAttempt := params.NextAttemptAt
	item.NextAttemptAt = &nextAttempt
	item.ClaimedAt = nil
	item.ClaimedByWorkerRun = nil
	item.Meta = cloneJSONMap(params.Meta)
	item.UpdatedAt = time.Now().UTC()
	r.items[params.ID] = item
	return cloneRecord(item), nil
}

func (r *memoryRepository) RequeueStaleClaims(_ context.Context, params RequeueParams) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for id, item := range r.items {
		if item.Status != StatusProcessing || item.ClaimedAt == nil || item.ClaimedAt.After(params.StaleBefore) {
			continue
		}
		next := params.RetryAt
		item.Status = StatusRetry
		item.NextAttemptAt = &next
		item.ClaimedAt = nil
		item.ClaimedByWorkerRun = nil
		item.LastErrorCode = ErrorCodeFileMove
		item.LastErrorMessage = "stale processing claim requeued"
		item.UpdatedAt = time.Now().UTC()
		r.items[id] = item
		count++
	}
	return count, nil
}

func (r *memoryRepository) GetByID(_ context.Context, id int64) (Record, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[id]
	if !ok {
		return Record{}, sql.ErrNoRows
	}
	return cloneRecord(item), nil
}

func cloneRecord(input Record) Record {
	out := input
	out.ModifiedAt = cloneTimePtr(input.ModifiedAt)
	out.ClaimedAt = cloneTimePtr(input.ClaimedAt)
	out.ClaimedByWorkerRun = cloneInt64Ptr(input.ClaimedByWorkerRun)
	out.NextAttemptAt = cloneTimePtr(input.NextAttemptAt)
	out.RequestID = cloneInt64Ptr(input.RequestID)
	out.ProcessedAt = cloneTimePtr(input.ProcessedAt)
	out.FailedAt = cloneTimePtr(input.FailedAt)
	out.Meta = cloneJSONMap(input.Meta)
	return out
}

func cloneTimePtr(input *time.Time) *time.Time {
	if input == nil {
		return nil
	}
	cloned := input.UTC()
	return &cloned
}

func cloneInt64Ptr(input *int64) *int64 {
	if input == nil {
		return nil
	}
	cloned := *input
	return &cloned
}

func cloneJSONMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func cloneMetaJSON(input map[string]any) string {
	encoded, _ := json.Marshal(cloneJSONMap(input))
	if len(encoded) == 0 {
		return "{}"
	}
	return string(encoded)
}

func sameTime(left *time.Time, right *time.Time) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return left.UTC().Equal(right.UTC())
	}
}

func isActive(status string) bool {
	switch strings.TrimSpace(status) {
	case StatusDiscovered, StatusRetry, StatusProcessing:
		return true
	default:
		return false
	}
}

func newUID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	hexed := hex.EncodeToString(buf)
	return fmt.Sprintf("%s-%s-%s-%s-%s", hexed[:8], hexed[8:12], hexed[12:16], hexed[16:20], hexed[20:32])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func int64Ptr(value int64) *int64 {
	return &value
}
