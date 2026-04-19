package scheduler

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

type recordRow struct {
	ID                  int64           `db:"id"`
	UID                 string          `db:"uid"`
	Code                string          `db:"code"`
	Name                string          `db:"name"`
	Description         string          `db:"description"`
	JobCategory         string          `db:"job_category"`
	JobType             string          `db:"job_type"`
	ScheduleType        string          `db:"schedule_type"`
	ScheduleExpr        string          `db:"schedule_expr"`
	Timezone            string          `db:"timezone"`
	Enabled             bool            `db:"enabled"`
	AllowConcurrentRuns bool            `db:"allow_concurrent_runs"`
	Config              json.RawMessage `db:"config"`
	LastRunAt           *time.Time      `db:"last_run_at"`
	NextRunAt           *time.Time      `db:"next_run_at"`
	LastSuccessAt       *time.Time      `db:"last_success_at"`
	LastFailureAt       *time.Time      `db:"last_failure_at"`
	LatestRunStatus     string          `db:"latest_run_status"`
	CreatedAt           time.Time       `db:"created_at"`
	UpdatedAt           time.Time       `db:"updated_at"`
}

type runRow struct {
	ID               int64           `db:"id"`
	UID              string          `db:"uid"`
	ScheduledJobID   int64           `db:"scheduled_job_id"`
	ScheduledJobUID  string          `db:"scheduled_job_uid"`
	ScheduledJobCode string          `db:"scheduled_job_code"`
	ScheduledJobName string          `db:"scheduled_job_name"`
	TriggerMode      string          `db:"trigger_mode"`
	ScheduledFor     time.Time       `db:"scheduled_for"`
	StartedAt        *time.Time      `db:"started_at"`
	FinishedAt       *time.Time      `db:"finished_at"`
	Status           string          `db:"status"`
	WorkerID         *int64          `db:"worker_id"`
	ErrorMessage     string          `db:"error_message"`
	ResultSummary    json.RawMessage `db:"result_summary"`
	CreatedAt        time.Time       `db:"created_at"`
	UpdatedAt        time.Time       `db:"updated_at"`
}

func normalizeListQuery(query ListQuery) ListQuery {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 25
	}

	sortField := strings.TrimSpace(query.SortField)
	switch sortField {
	case "code", "name", "jobCategory", "jobType", "scheduleType", "enabled", "nextRunAt", "lastRunAt", "updatedAt", "createdAt":
	default:
		sortField = "name"
	}

	sortOrder := strings.ToLower(strings.TrimSpace(query.SortOrder))
	if sortOrder != "desc" {
		sortOrder = "asc"
	}

	return ListQuery{
		Page:      page,
		PageSize:  pageSize,
		SortField: sortField,
		SortOrder: sortOrder,
		Filter:    strings.TrimSpace(query.Filter),
		Category:  strings.ToLower(strings.TrimSpace(query.Category)),
	}
}

func normalizeRunListQuery(query RunListQuery) RunListQuery {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 25
	}
	sortField := strings.TrimSpace(query.SortField)
	switch sortField {
	case "scheduledFor", "status", "startedAt", "finishedAt", "createdAt":
	default:
		sortField = "scheduledFor"
	}
	sortOrder := strings.ToLower(strings.TrimSpace(query.SortOrder))
	if sortOrder != "asc" {
		sortOrder = "desc"
	}

	return RunListQuery{
		Page:      page,
		PageSize:  pageSize,
		SortField: sortField,
		SortOrder: sortOrder,
		Status:    strings.ToLower(strings.TrimSpace(query.Status)),
	}
}

func (r *SQLRepository) ListScheduledJobs(ctx context.Context, query ListQuery) (ListResult, error) {
	q := normalizeListQuery(query)
	offset := (q.Page - 1) * q.PageSize

	conditions := make([]string, 0, 2)
	args := make([]any, 0, 4)
	if q.Filter != "" {
		args = append(args, "%"+q.Filter+"%")
		bind := fmt.Sprintf("$%d", len(args))
		conditions = append(conditions, `(code ILIKE `+bind+` OR name ILIKE `+bind+` OR description ILIKE `+bind+` OR job_type ILIKE `+bind+`)`)
	}
	if q.Category != "" {
		args = append(args, q.Category)
		bind := fmt.Sprintf("$%d", len(args))
		conditions = append(conditions, `job_category = `+bind)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM scheduled_jobs`+whereClause, args...); err != nil {
		return ListResult{}, fmt.Errorf("count scheduled jobs: %w", err)
	}

	rows := []recordRow{}
	selectArgs := append([]any{}, args...)
	selectArgs = append(selectArgs, q.PageSize, offset)
	querySQL := `
		SELECT j.id, j.uid::text AS uid, j.code, j.name, j.description, j.job_category, j.job_type, j.schedule_type, j.schedule_expr,
		       j.timezone, j.enabled, j.allow_concurrent_runs, j.config, j.last_run_at, j.next_run_at, j.last_success_at, j.last_failure_at,
		       COALESCE(latest.status, '') AS latest_run_status, j.created_at, j.updated_at
		FROM scheduled_jobs j
		LEFT JOIN LATERAL (
			SELECT r.status
			FROM scheduled_job_runs r
			WHERE r.scheduled_job_id = j.id
			ORDER BY COALESCE(r.finished_at, r.started_at, r.created_at) DESC, r.id DESC
			LIMIT 1
		) latest ON TRUE
	` + whereClause + fmt.Sprintf(" ORDER BY %s %s LIMIT $%d OFFSET $%d",
		resolveJobSortColumn(q.SortField),
		strings.ToUpper(q.SortOrder),
		len(selectArgs)-1,
		len(selectArgs),
	)
	if err := r.db.SelectContext(ctx, &rows, querySQL, selectArgs...); err != nil {
		return ListResult{}, fmt.Errorf("list scheduled jobs: %w", err)
	}

	items, err := decodeRows(rows)
	if err != nil {
		return ListResult{}, err
	}
	return ListResult{Items: items, Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}

func (r *SQLRepository) GetScheduledJobByID(ctx context.Context, id int64) (Record, error) {
	var row recordRow
	if err := r.db.GetContext(ctx, &row, `
		SELECT j.id, j.uid::text AS uid, j.code, j.name, j.description, j.job_category, j.job_type, j.schedule_type, j.schedule_expr,
		       j.timezone, j.enabled, j.allow_concurrent_runs, j.config, j.last_run_at, j.next_run_at, j.last_success_at, j.last_failure_at,
		       COALESCE(latest.status, '') AS latest_run_status, j.created_at, j.updated_at
		FROM scheduled_jobs j
		LEFT JOIN LATERAL (
			SELECT r.status
			FROM scheduled_job_runs r
			WHERE r.scheduled_job_id = j.id
			ORDER BY COALESCE(r.finished_at, r.started_at, r.created_at) DESC, r.id DESC
			LIMIT 1
		) latest ON TRUE
		WHERE j.id = $1
	`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, sql.ErrNoRows
		}
		return Record{}, fmt.Errorf("get scheduled job: %w", err)
	}
	return decodeRow(row)
}

func (r *SQLRepository) GetScheduledJobByCode(ctx context.Context, code string) (Record, error) {
	var row recordRow
	if err := r.db.GetContext(ctx, &row, `
		SELECT j.id, j.uid::text AS uid, j.code, j.name, j.description, j.job_category, j.job_type, j.schedule_type, j.schedule_expr,
		       j.timezone, j.enabled, j.allow_concurrent_runs, j.config, j.last_run_at, j.next_run_at, j.last_success_at, j.last_failure_at,
		       COALESCE(latest.status, '') AS latest_run_status, j.created_at, j.updated_at
		FROM scheduled_jobs j
		LEFT JOIN LATERAL (
			SELECT r.status
			FROM scheduled_job_runs r
			WHERE r.scheduled_job_id = j.id
			ORDER BY COALESCE(r.finished_at, r.started_at, r.created_at) DESC, r.id DESC
			LIMIT 1
		) latest ON TRUE
		WHERE j.code = $1
	`, strings.TrimSpace(code)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, sql.ErrNoRows
		}
		return Record{}, fmt.Errorf("get scheduled job by code: %w", err)
	}
	return decodeRow(row)
}

func (r *SQLRepository) CreateScheduledJob(ctx context.Context, params CreateParams) (Record, error) {
	configValue, err := json.Marshal(cloneJSONMap(params.Config))
	if err != nil {
		return Record{}, fmt.Errorf("marshal scheduler config: %w", err)
	}

	var row recordRow
	if err := r.db.GetContext(ctx, &row, `
		INSERT INTO scheduled_jobs (
			uid, code, name, description, job_category, job_type, schedule_type, schedule_expr, timezone,
			enabled, allow_concurrent_runs, config, next_run_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12::jsonb, $13, NOW(), NOW())
		RETURNING id, uid::text AS uid, code, name, description, job_category, job_type, schedule_type, schedule_expr,
		          timezone, enabled, allow_concurrent_runs, config, last_run_at, next_run_at, last_success_at, last_failure_at,
		          '' AS latest_run_status, created_at, updated_at
	`,
		params.UID,
		params.Code,
		params.Name,
		params.Description,
		params.JobCategory,
		params.JobType,
		params.ScheduleType,
		params.ScheduleExpr,
		params.Timezone,
		params.Enabled,
		params.AllowConcurrentRuns,
		string(configValue),
		params.NextRunAt,
	); err != nil {
		return Record{}, fmt.Errorf("create scheduled job: %w", err)
	}
	return decodeRow(row)
}

func (r *SQLRepository) UpdateScheduledJob(ctx context.Context, params UpdateParams) (Record, error) {
	configValue, err := json.Marshal(cloneJSONMap(params.Config))
	if err != nil {
		return Record{}, fmt.Errorf("marshal scheduler config: %w", err)
	}

	var row recordRow
	if err := r.db.GetContext(ctx, &row, `
		UPDATE scheduled_jobs
		SET code = $2,
		    name = $3,
		    description = $4,
		    job_category = $5,
		    job_type = $6,
		    schedule_type = $7,
		    schedule_expr = $8,
		    timezone = $9,
		    enabled = $10,
		    allow_concurrent_runs = $11,
		    config = $12::jsonb,
		    next_run_at = $13,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, uid::text AS uid, code, name, description, job_category, job_type, schedule_type, schedule_expr,
		          timezone, enabled, allow_concurrent_runs, config, last_run_at, next_run_at, last_success_at, last_failure_at,
		          '' AS latest_run_status, created_at, updated_at
	`,
		params.ID,
		params.Code,
		params.Name,
		params.Description,
		params.JobCategory,
		params.JobType,
		params.ScheduleType,
		params.ScheduleExpr,
		params.Timezone,
		params.Enabled,
		params.AllowConcurrentRuns,
		string(configValue),
		params.NextRunAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, sql.ErrNoRows
		}
		return Record{}, fmt.Errorf("update scheduled job: %w", err)
	}
	return decodeRow(row)
}

func (r *SQLRepository) SetScheduledJobEnabled(ctx context.Context, params SetEnabledParams) (Record, error) {
	var row recordRow
	if err := r.db.GetContext(ctx, &row, `
		UPDATE scheduled_jobs
		SET enabled = $2,
		    next_run_at = $3,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, uid::text AS uid, code, name, description, job_category, job_type, schedule_type, schedule_expr,
		          timezone, enabled, allow_concurrent_runs, config, last_run_at, next_run_at, last_success_at, last_failure_at,
		          '' AS latest_run_status, created_at, updated_at
	`, params.ID, params.Enabled, params.NextRunAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, sql.ErrNoRows
		}
		return Record{}, fmt.Errorf("set scheduled job enabled: %w", err)
	}
	return decodeRow(row)
}

func (r *SQLRepository) ListJobRuns(ctx context.Context, jobID int64, query RunListQuery) (RunListResult, error) {
	q := normalizeRunListQuery(query)
	offset := (q.Page - 1) * q.PageSize

	conditions := []string{"r.scheduled_job_id = $1"}
	args := []any{jobID}
	if q.Status != "" {
		args = append(args, q.Status)
		conditions = append(conditions, fmt.Sprintf("r.status = $%d", len(args)))
	}
	whereClause := " WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM scheduled_job_runs r`+whereClause, args...); err != nil {
		return RunListResult{}, fmt.Errorf("count scheduler runs: %w", err)
	}

	rows := []runRow{}
	selectArgs := append([]any{}, args...)
	selectArgs = append(selectArgs, q.PageSize, offset)
	querySQL := `
		SELECT r.id, r.uid::text AS uid, r.scheduled_job_id,
		       j.uid::text AS scheduled_job_uid, j.code AS scheduled_job_code, j.name AS scheduled_job_name,
		       r.trigger_mode, r.scheduled_for, r.started_at, r.finished_at, r.status, r.worker_id,
		       COALESCE(r.error_message, '') AS error_message, r.result_summary, r.created_at, r.updated_at
		FROM scheduled_job_runs r
		INNER JOIN scheduled_jobs j ON j.id = r.scheduled_job_id
	` + whereClause + fmt.Sprintf(" ORDER BY %s %s LIMIT $%d OFFSET $%d",
		resolveRunSortColumn(q.SortField),
		strings.ToUpper(q.SortOrder),
		len(selectArgs)-1,
		len(selectArgs),
	)
	if err := r.db.SelectContext(ctx, &rows, querySQL, selectArgs...); err != nil {
		return RunListResult{}, fmt.Errorf("list scheduler runs: %w", err)
	}

	items, err := decodeRunRows(rows)
	if err != nil {
		return RunListResult{}, err
	}
	return RunListResult{Items: items, Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}

func (r *SQLRepository) GetRunByID(ctx context.Context, id int64) (RunRecord, error) {
	var row runRow
	if err := r.db.GetContext(ctx, &row, `
		SELECT r.id, r.uid::text AS uid, r.scheduled_job_id,
		       j.uid::text AS scheduled_job_uid, j.code AS scheduled_job_code, j.name AS scheduled_job_name,
		       r.trigger_mode, r.scheduled_for, r.started_at, r.finished_at, r.status, r.worker_id,
		       COALESCE(r.error_message, '') AS error_message, r.result_summary, r.created_at, r.updated_at
		FROM scheduled_job_runs r
		INNER JOIN scheduled_jobs j ON j.id = r.scheduled_job_id
		WHERE r.id = $1
	`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RunRecord{}, sql.ErrNoRows
		}
		return RunRecord{}, fmt.Errorf("get scheduler run: %w", err)
	}
	return decodeRunRow(row)
}

func (r *SQLRepository) CreateJobRun(ctx context.Context, params CreateRunParams) (RunRecord, error) {
	resultSummary, err := json.Marshal(cloneJSONMap(params.ResultSummary))
	if err != nil {
		return RunRecord{}, fmt.Errorf("marshal run summary: %w", err)
	}

	var row runRow
	if err := r.db.GetContext(ctx, &row, `
		INSERT INTO scheduled_job_runs (
			uid, scheduled_job_id, trigger_mode, scheduled_for, status, result_summary, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, NOW(), NOW())
		RETURNING id, uid::text AS uid, scheduled_job_id, '' AS scheduled_job_uid, '' AS scheduled_job_code, '' AS scheduled_job_name,
		          trigger_mode, scheduled_for, started_at, finished_at, status, worker_id, COALESCE(error_message, '') AS error_message,
		          result_summary, created_at, updated_at
	`, params.UID, params.ScheduledJobID, params.TriggerMode, params.ScheduledFor, params.Status, string(resultSummary)); err != nil {
		return RunRecord{}, fmt.Errorf("create scheduler run: %w", err)
	}

	return r.GetRunByID(ctx, row.ID)
}

func (r *SQLRepository) ListDueScheduledJobs(ctx context.Context, now time.Time, limit int) ([]Record, error) {
	if limit <= 0 {
		limit = 1
	}
	rows := []recordRow{}
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT j.id, j.uid::text AS uid, j.code, j.name, j.description, j.job_category, j.job_type, j.schedule_type, j.schedule_expr,
		       j.timezone, j.enabled, j.allow_concurrent_runs, j.config, j.last_run_at, j.next_run_at, j.last_success_at, j.last_failure_at,
		       COALESCE(latest.status, '') AS latest_run_status, j.created_at, j.updated_at
		FROM scheduled_jobs j
		LEFT JOIN LATERAL (
			SELECT r.status
			FROM scheduled_job_runs r
			WHERE r.scheduled_job_id = j.id
			ORDER BY COALESCE(r.finished_at, r.started_at, r.created_at) DESC, r.id DESC
			LIMIT 1
		) latest ON TRUE
		WHERE j.enabled = TRUE
		  AND j.next_run_at IS NOT NULL
		  AND j.next_run_at <= $1
		ORDER BY j.next_run_at ASC, j.id ASC
		LIMIT $2
	`, now.UTC(), limit); err != nil {
		return nil, fmt.Errorf("list due scheduled jobs: %w", err)
	}
	return decodeRows(rows)
}

func (r *SQLRepository) HasActiveJobRuns(ctx context.Context, jobID int64) (bool, error) {
	var active bool
	if err := r.db.GetContext(ctx, &active, `
		SELECT EXISTS(
			SELECT 1
			FROM scheduled_job_runs
			WHERE scheduled_job_id = $1
			  AND status IN ('pending', 'running')
		)
	`, jobID); err != nil {
		return false, fmt.Errorf("check active scheduler runs: %w", err)
	}
	return active, nil
}

func (r *SQLRepository) DispatchScheduledJob(ctx context.Context, params DispatchJobParams) (RunRecord, bool, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return RunRecord{}, false, fmt.Errorf("begin scheduler dispatch tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var job recordRow
	if err := tx.GetContext(ctx, &job, `
		SELECT id, uid::text AS uid, code, name, description, job_category, job_type, schedule_type, schedule_expr,
		       timezone, enabled, allow_concurrent_runs, config, last_run_at, next_run_at, last_success_at, last_failure_at,
		       '' AS latest_run_status, created_at, updated_at
		FROM scheduled_jobs
		WHERE id = $1
		  AND enabled = TRUE
		  AND next_run_at IS NOT NULL
		  AND next_run_at <= $2
		FOR UPDATE SKIP LOCKED
	`, params.JobID, params.ScheduledFor.UTC()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RunRecord{}, false, nil
		}
		return RunRecord{}, false, fmt.Errorf("lock scheduled job for dispatch: %w", err)
	}

	if !job.AllowConcurrentRuns {
		var active bool
		if err := tx.GetContext(ctx, &active, `
			SELECT EXISTS(
				SELECT 1
				FROM scheduled_job_runs
				WHERE scheduled_job_id = $1
				  AND status IN ('pending', 'running')
			)
		`, params.JobID); err != nil {
			return RunRecord{}, false, fmt.Errorf("check active scheduler runs in dispatch: %w", err)
		}
		if active {
			if err := tx.Commit(); err != nil {
				return RunRecord{}, false, fmt.Errorf("commit skipped scheduler dispatch: %w", err)
			}
			return RunRecord{}, false, nil
		}
	}

	resultSummary, err := json.Marshal(cloneJSONMap(params.ResultSummary))
	if err != nil {
		return RunRecord{}, false, fmt.Errorf("marshal dispatch result summary: %w", err)
	}

	var runID int64
	if err := tx.GetContext(ctx, &runID, `
		INSERT INTO scheduled_job_runs (
			uid, scheduled_job_id, trigger_mode, scheduled_for, status, result_summary, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, 'pending', $5::jsonb, NOW(), NOW())
		RETURNING id
	`, params.RunUID, params.JobID, params.TriggerMode, params.ScheduledFor.UTC(), string(resultSummary)); err != nil {
		return RunRecord{}, false, fmt.Errorf("insert scheduled job run during dispatch: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE scheduled_jobs
		SET next_run_at = $2,
		    updated_at = NOW()
		WHERE id = $1
	`, params.JobID, params.NextRunAt); err != nil {
		return RunRecord{}, false, fmt.Errorf("update next run during dispatch: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return RunRecord{}, false, fmt.Errorf("commit scheduler dispatch tx: %w", err)
	}
	record, err := r.GetRunByID(ctx, runID)
	if err != nil {
		return RunRecord{}, false, err
	}
	return record, true, nil
}

func (r *SQLRepository) ClaimNextPendingRun(ctx context.Context, now time.Time, workerID int64) (RunRecord, error) {
	var row runRow
	if err := r.db.GetContext(ctx, &row, `
		WITH candidate AS (
			SELECT r.id
			FROM scheduled_job_runs r
			INNER JOIN scheduled_jobs j ON j.id = r.scheduled_job_id
			WHERE r.status = 'pending'
			  AND (
				j.allow_concurrent_runs = TRUE OR NOT EXISTS (
					SELECT 1
					FROM scheduled_job_runs active
					WHERE active.scheduled_job_id = r.scheduled_job_id
					  AND active.status = 'running'
					  AND active.id <> r.id
				)
			  )
			ORDER BY r.scheduled_for ASC, r.created_at ASC, r.id ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		), claimed AS (
			UPDATE scheduled_job_runs r
			SET status = 'running',
			    worker_id = NULLIF($2, 0),
			    started_at = $1,
			    updated_at = NOW()
			WHERE r.id IN (SELECT id FROM candidate)
			RETURNING r.id, r.scheduled_job_id
		), touched_jobs AS (
			UPDATE scheduled_jobs j
			SET last_run_at = $1,
			    updated_at = NOW()
			WHERE j.id IN (SELECT scheduled_job_id FROM claimed)
		)
		SELECT r.id, r.uid::text AS uid, r.scheduled_job_id,
		       j.uid::text AS scheduled_job_uid, j.code AS scheduled_job_code, j.name AS scheduled_job_name,
		       r.trigger_mode, r.scheduled_for, r.started_at, r.finished_at, r.status, r.worker_id,
		       COALESCE(r.error_message, '') AS error_message, r.result_summary, r.created_at, r.updated_at
		FROM scheduled_job_runs r
		INNER JOIN scheduled_jobs j ON j.id = r.scheduled_job_id
		INNER JOIN claimed c ON c.id = r.id
	`, now.UTC(), workerID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RunRecord{}, sql.ErrNoRows
		}
		return RunRecord{}, fmt.Errorf("claim pending scheduler run: %w", err)
	}
	return decodeRunRow(row)
}

func (r *SQLRepository) FinalizeJobRun(ctx context.Context, params FinalizeRunParams) (RunRecord, error) {
	resultSummary, err := json.Marshal(cloneJSONMap(params.ResultSummary))
	if err != nil {
		return RunRecord{}, fmt.Errorf("marshal scheduler run final summary: %w", err)
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return RunRecord{}, fmt.Errorf("begin scheduler finalize tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var jobID int64
	if err := tx.GetContext(ctx, &jobID, `
		UPDATE scheduled_job_runs
		SET status = $2,
		    finished_at = $3,
		    error_message = $4,
		    result_summary = $5::jsonb,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING scheduled_job_id
	`, params.RunID, params.Status, params.FinishedAt.UTC(), params.ErrorMessage, string(resultSummary)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RunRecord{}, sql.ErrNoRows
		}
		return RunRecord{}, fmt.Errorf("update scheduler run final state: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE scheduled_jobs
		SET last_run_at = $2,
		    last_success_at = CASE WHEN $3::timestamptz IS NULL THEN last_success_at ELSE $3 END,
		    last_failure_at = CASE WHEN $4::timestamptz IS NULL THEN last_failure_at ELSE $4 END,
		    updated_at = NOW()
		WHERE id = $1
	`, jobID, params.LastRunAt.UTC(), params.LastSuccessAt, params.LastFailureAt); err != nil {
		return RunRecord{}, fmt.Errorf("update scheduler job final timestamps: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return RunRecord{}, fmt.Errorf("commit scheduler finalize tx: %w", err)
	}
	return r.GetRunByID(ctx, params.RunID)
}

func resolveJobSortColumn(field string) string {
	switch field {
	case "code":
		return "code"
	case "jobCategory":
		return "job_category"
	case "jobType":
		return "job_type"
	case "scheduleType":
		return "schedule_type"
	case "enabled":
		return "enabled"
	case "nextRunAt":
		return "next_run_at"
	case "lastRunAt":
		return "last_run_at"
	case "updatedAt":
		return "updated_at"
	case "createdAt":
		return "created_at"
	default:
		return "name"
	}
}

func resolveRunSortColumn(field string) string {
	switch field {
	case "status":
		return "r.status"
	case "startedAt":
		return "r.started_at"
	case "finishedAt":
		return "r.finished_at"
	case "createdAt":
		return "r.created_at"
	default:
		return "r.scheduled_for"
	}
}

func decodeRows(rows []recordRow) ([]Record, error) {
	items := make([]Record, 0, len(rows))
	for _, row := range rows {
		item, err := decodeRow(row)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func decodeRow(row recordRow) (Record, error) {
	configValue := map[string]any{}
	if len(row.Config) > 0 {
		if err := json.Unmarshal(row.Config, &configValue); err != nil {
			return Record{}, fmt.Errorf("decode scheduler config: %w", err)
		}
	}
	return Record{
		ID:                  row.ID,
		UID:                 row.UID,
		Code:                row.Code,
		Name:                row.Name,
		Description:         row.Description,
		JobCategory:         row.JobCategory,
		JobType:             row.JobType,
		ScheduleType:        row.ScheduleType,
		ScheduleExpr:        row.ScheduleExpr,
		Timezone:            row.Timezone,
		Enabled:             row.Enabled,
		AllowConcurrentRuns: row.AllowConcurrentRuns,
		Config:              configValue,
		LastRunAt:           row.LastRunAt,
		NextRunAt:           row.NextRunAt,
		LastSuccessAt:       row.LastSuccessAt,
		LastFailureAt:       row.LastFailureAt,
		LatestRunStatus:     row.LatestRunStatus,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}, nil
}

func decodeRunRows(rows []runRow) ([]RunRecord, error) {
	items := make([]RunRecord, 0, len(rows))
	for _, row := range rows {
		item, err := decodeRunRow(row)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func decodeRunRow(row runRow) (RunRecord, error) {
	summary := map[string]any{}
	if len(row.ResultSummary) > 0 {
		if err := json.Unmarshal(row.ResultSummary, &summary); err != nil {
			return RunRecord{}, fmt.Errorf("decode run summary: %w", err)
		}
	}
	return RunRecord{
		ID:               row.ID,
		UID:              row.UID,
		ScheduledJobID:   row.ScheduledJobID,
		ScheduledJobUID:  row.ScheduledJobUID,
		ScheduledJobCode: row.ScheduledJobCode,
		ScheduledJobName: row.ScheduledJobName,
		TriggerMode:      row.TriggerMode,
		ScheduledFor:     row.ScheduledFor,
		StartedAt:        row.StartedAt,
		FinishedAt:       row.FinishedAt,
		Status:           row.Status,
		WorkerID:         row.WorkerID,
		ErrorMessage:     row.ErrorMessage,
		ResultSummary:    summary,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}, nil
}

type memoryRepository struct {
	mu        sync.RWMutex
	nextID    int64
	nextRunID int64
	jobs      []Record
	runs      []RunRecord
}

func newMemoryRepository() Repository {
	return &memoryRepository{
		nextID:    1,
		nextRunID: 1,
		jobs:      []Record{},
		runs:      []RunRecord{},
	}
}

func (r *memoryRepository) ListScheduledJobs(_ context.Context, query ListQuery) (ListResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	q := normalizeListQuery(query)
	items := make([]Record, 0, len(r.jobs))
	for _, job := range r.jobs {
		if q.Filter != "" && !matchesJobFilter(job, q.Filter) {
			continue
		}
		if q.Category != "" && job.JobCategory != q.Category {
			continue
		}
		cloned := cloneRecord(job)
		cloned.LatestRunStatus = r.latestRunStatusLocked(cloned.ID)
		items = append(items, cloned)
	}

	sortJobs(items, q.SortField, q.SortOrder)
	total := len(items)
	start := min(total, (q.Page-1)*q.PageSize)
	end := min(total, start+q.PageSize)
	return ListResult{Items: items[start:end], Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}

func (r *memoryRepository) GetScheduledJobByID(_ context.Context, id int64) (Record, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, item := range r.jobs {
		if item.ID == id {
			cloned := cloneRecord(item)
			cloned.LatestRunStatus = r.latestRunStatusLocked(cloned.ID)
			return cloned, nil
		}
	}
	return Record{}, sql.ErrNoRows
}

func (r *memoryRepository) GetScheduledJobByCode(_ context.Context, code string) (Record, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, item := range r.jobs {
		if item.Code == strings.TrimSpace(code) {
			cloned := cloneRecord(item)
			cloned.LatestRunStatus = r.latestRunStatusLocked(cloned.ID)
			return cloned, nil
		}
	}
	return Record{}, sql.ErrNoRows
}

func (r *memoryRepository) CreateScheduledJob(_ context.Context, params CreateParams) (Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	record := Record{
		ID:                  r.nextID,
		UID:                 params.UID,
		Code:                params.Code,
		Name:                params.Name,
		Description:         params.Description,
		JobCategory:         params.JobCategory,
		JobType:             params.JobType,
		ScheduleType:        params.ScheduleType,
		ScheduleExpr:        params.ScheduleExpr,
		Timezone:            params.Timezone,
		Enabled:             params.Enabled,
		AllowConcurrentRuns: params.AllowConcurrentRuns,
		Config:              cloneJSONMap(params.Config),
		NextRunAt:           cloneTimePtr(params.NextRunAt),
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	r.nextID++
	r.jobs = append(r.jobs, record)
	return cloneRecord(record), nil
}

func (r *memoryRepository) UpdateScheduledJob(_ context.Context, params UpdateParams) (Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for index := range r.jobs {
		if r.jobs[index].ID != params.ID {
			continue
		}
		r.jobs[index].Code = params.Code
		r.jobs[index].Name = params.Name
		r.jobs[index].Description = params.Description
		r.jobs[index].JobCategory = params.JobCategory
		r.jobs[index].JobType = params.JobType
		r.jobs[index].ScheduleType = params.ScheduleType
		r.jobs[index].ScheduleExpr = params.ScheduleExpr
		r.jobs[index].Timezone = params.Timezone
		r.jobs[index].Enabled = params.Enabled
		r.jobs[index].AllowConcurrentRuns = params.AllowConcurrentRuns
		r.jobs[index].Config = cloneJSONMap(params.Config)
		r.jobs[index].NextRunAt = cloneTimePtr(params.NextRunAt)
		r.jobs[index].UpdatedAt = time.Now().UTC()
		cloned := cloneRecord(r.jobs[index])
		cloned.LatestRunStatus = r.latestRunStatusLocked(cloned.ID)
		return cloned, nil
	}
	return Record{}, sql.ErrNoRows
}

func (r *memoryRepository) SetScheduledJobEnabled(_ context.Context, params SetEnabledParams) (Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for index := range r.jobs {
		if r.jobs[index].ID != params.ID {
			continue
		}
		r.jobs[index].Enabled = params.Enabled
		r.jobs[index].NextRunAt = cloneTimePtr(params.NextRunAt)
		r.jobs[index].UpdatedAt = time.Now().UTC()
		cloned := cloneRecord(r.jobs[index])
		cloned.LatestRunStatus = r.latestRunStatusLocked(cloned.ID)
		return cloned, nil
	}
	return Record{}, sql.ErrNoRows
}

func (r *memoryRepository) ListJobRuns(_ context.Context, jobID int64, query RunListQuery) (RunListResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	q := normalizeRunListQuery(query)
	items := make([]RunRecord, 0, len(r.runs))
	for _, run := range r.runs {
		if run.ScheduledJobID != jobID {
			continue
		}
		if q.Status != "" && run.Status != q.Status {
			continue
		}
		items = append(items, cloneRunRecord(run))
	}
	sortRuns(items, q.SortField, q.SortOrder)
	total := len(items)
	start := min(total, (q.Page-1)*q.PageSize)
	end := min(total, start+q.PageSize)
	return RunListResult{Items: items[start:end], Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}

func (r *memoryRepository) GetRunByID(_ context.Context, id int64) (RunRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, run := range r.runs {
		if run.ID == id {
			return cloneRunRecord(run), nil
		}
	}
	return RunRecord{}, sql.ErrNoRows
}

func (r *memoryRepository) CreateJobRun(_ context.Context, params CreateRunParams) (RunRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var job Record
	found := false
	for _, item := range r.jobs {
		if item.ID == params.ScheduledJobID {
			job = item
			found = true
			break
		}
	}
	if !found {
		return RunRecord{}, sql.ErrNoRows
	}

	now := time.Now().UTC()
	record := RunRecord{
		ID:               r.nextRunID,
		UID:              params.UID,
		ScheduledJobID:   params.ScheduledJobID,
		ScheduledJobUID:  job.UID,
		ScheduledJobCode: job.Code,
		ScheduledJobName: job.Name,
		TriggerMode:      params.TriggerMode,
		ScheduledFor:     params.ScheduledFor,
		Status:           params.Status,
		ErrorMessage:     "",
		ResultSummary:    cloneJSONMap(params.ResultSummary),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	r.nextRunID++
	r.runs = append(r.runs, record)
	return cloneRunRecord(record), nil
}

func (r *memoryRepository) ListDueScheduledJobs(_ context.Context, now time.Time, limit int) ([]Record, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 1
	}
	items := make([]Record, 0, limit)
	for _, job := range r.jobs {
		if !job.Enabled || job.NextRunAt == nil || job.NextRunAt.After(now.UTC()) {
			continue
		}
		cloned := cloneRecord(job)
		cloned.LatestRunStatus = r.latestRunStatusLocked(cloned.ID)
		items = append(items, cloned)
	}
	sortJobs(items, "nextRunAt", "asc")
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (r *memoryRepository) HasActiveJobRuns(_ context.Context, jobID int64) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.hasActiveRunsLocked(jobID), nil
}

func (r *memoryRepository) DispatchScheduledJob(_ context.Context, params DispatchJobParams) (RunRecord, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for index := range r.jobs {
		job := &r.jobs[index]
		if job.ID != params.JobID {
			continue
		}
		if !job.Enabled || job.NextRunAt == nil || job.NextRunAt.After(params.ScheduledFor.UTC()) {
			return RunRecord{}, false, nil
		}
		if !job.AllowConcurrentRuns && r.hasActiveRunsLocked(job.ID) {
			return RunRecord{}, false, nil
		}
		now := time.Now().UTC()
		record := RunRecord{
			ID:               r.nextRunID,
			UID:              params.RunUID,
			ScheduledJobID:   job.ID,
			ScheduledJobUID:  job.UID,
			ScheduledJobCode: job.Code,
			ScheduledJobName: job.Name,
			TriggerMode:      params.TriggerMode,
			ScheduledFor:     params.ScheduledFor.UTC(),
			Status:           RunStatusPending,
			ResultSummary:    cloneJSONMap(params.ResultSummary),
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		r.nextRunID++
		r.runs = append(r.runs, record)
		job.NextRunAt = cloneTimePtr(params.NextRunAt)
		job.UpdatedAt = now
		return cloneRunRecord(record), true, nil
	}
	return RunRecord{}, false, nil
}

func (r *memoryRepository) ClaimNextPendingRun(_ context.Context, now time.Time, workerID int64) (RunRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var selectedIndex = -1
	for index := range r.runs {
		run := r.runs[index]
		if run.Status != RunStatusPending {
			continue
		}
		job := r.lookupJobLocked(run.ScheduledJobID)
		if job == nil {
			continue
		}
		if !job.AllowConcurrentRuns && r.hasRunningRunLocked(run.ScheduledJobID, run.ID) {
			continue
		}
		if selectedIndex < 0 ||
			run.ScheduledFor.Before(r.runs[selectedIndex].ScheduledFor) ||
			(run.ScheduledFor.Equal(r.runs[selectedIndex].ScheduledFor) && run.ID < r.runs[selectedIndex].ID) {
			selectedIndex = index
		}
	}
	if selectedIndex < 0 {
		return RunRecord{}, sql.ErrNoRows
	}
	startedAt := now.UTC()
	r.runs[selectedIndex].Status = RunStatusRunning
	r.runs[selectedIndex].WorkerID = &workerID
	r.runs[selectedIndex].StartedAt = &startedAt
	r.runs[selectedIndex].UpdatedAt = startedAt
	if job := r.lookupJobLocked(r.runs[selectedIndex].ScheduledJobID); job != nil {
		job.LastRunAt = &startedAt
		job.UpdatedAt = startedAt
	}
	return cloneRunRecord(r.runs[selectedIndex]), nil
}

func (r *memoryRepository) FinalizeJobRun(_ context.Context, params FinalizeRunParams) (RunRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for index := range r.runs {
		if r.runs[index].ID != params.RunID {
			continue
		}
		finishedAt := params.FinishedAt.UTC()
		r.runs[index].Status = params.Status
		r.runs[index].FinishedAt = &finishedAt
		r.runs[index].ErrorMessage = params.ErrorMessage
		r.runs[index].ResultSummary = cloneJSONMap(params.ResultSummary)
		r.runs[index].UpdatedAt = finishedAt
		if job := r.lookupJobLocked(r.runs[index].ScheduledJobID); job != nil {
			lastRunAt := params.LastRunAt.UTC()
			job.LastRunAt = &lastRunAt
			job.LastSuccessAt = cloneTimePtr(params.LastSuccessAt)
			job.LastFailureAt = cloneTimePtr(params.LastFailureAt)
			job.UpdatedAt = finishedAt
		}
		return cloneRunRecord(r.runs[index]), nil
	}
	return RunRecord{}, sql.ErrNoRows
}

func matchesJobFilter(job Record, filter string) bool {
	needle := strings.ToLower(strings.TrimSpace(filter))
	return strings.Contains(strings.ToLower(job.Code), needle) ||
		strings.Contains(strings.ToLower(job.Name), needle) ||
		strings.Contains(strings.ToLower(job.Description), needle) ||
		strings.Contains(strings.ToLower(job.JobType), needle)
}

func sortJobs(items []Record, field string, order string) {
	desc := strings.ToLower(order) == "desc"
	slices.SortFunc(items, func(left Record, right Record) int {
		var cmp int
		switch field {
		case "code":
			cmp = strings.Compare(left.Code, right.Code)
		case "jobCategory":
			cmp = strings.Compare(left.JobCategory, right.JobCategory)
		case "jobType":
			cmp = strings.Compare(left.JobType, right.JobType)
		case "scheduleType":
			cmp = strings.Compare(left.ScheduleType, right.ScheduleType)
		case "enabled":
			if left.Enabled == right.Enabled {
				cmp = 0
			} else if left.Enabled {
				cmp = 1
			} else {
				cmp = -1
			}
		case "nextRunAt":
			cmp = compareTimePtr(left.NextRunAt, right.NextRunAt)
		case "lastRunAt":
			cmp = compareTimePtr(left.LastRunAt, right.LastRunAt)
		case "updatedAt":
			cmp = left.UpdatedAt.Compare(right.UpdatedAt)
		case "createdAt":
			cmp = left.CreatedAt.Compare(right.CreatedAt)
		default:
			cmp = strings.Compare(left.Name, right.Name)
		}
		if desc {
			return -cmp
		}
		return cmp
	})
}

func sortRuns(items []RunRecord, field string, order string) {
	desc := strings.ToLower(order) == "desc"
	slices.SortFunc(items, func(left RunRecord, right RunRecord) int {
		var cmp int
		switch field {
		case "status":
			cmp = strings.Compare(left.Status, right.Status)
		case "startedAt":
			cmp = compareTimePtr(left.StartedAt, right.StartedAt)
		case "finishedAt":
			cmp = compareTimePtr(left.FinishedAt, right.FinishedAt)
		case "createdAt":
			cmp = left.CreatedAt.Compare(right.CreatedAt)
		default:
			cmp = left.ScheduledFor.Compare(right.ScheduledFor)
		}
		if desc {
			return -cmp
		}
		return cmp
	})
}

func compareTimePtr(left *time.Time, right *time.Time) int {
	switch {
	case left == nil && right == nil:
		return 0
	case left == nil:
		return -1
	case right == nil:
		return 1
	default:
		return left.Compare(*right)
	}
}

func (r *memoryRepository) lookupJobLocked(jobID int64) *Record {
	for index := range r.jobs {
		if r.jobs[index].ID == jobID {
			return &r.jobs[index]
		}
	}
	return nil
}

func (r *memoryRepository) hasActiveRunsLocked(jobID int64) bool {
	for _, run := range r.runs {
		if run.ScheduledJobID == jobID && (run.Status == RunStatusPending || run.Status == RunStatusRunning) {
			return true
		}
	}
	return false
}

func (r *memoryRepository) hasRunningRunLocked(jobID int64, excludeID int64) bool {
	for _, run := range r.runs {
		if run.ScheduledJobID == jobID && run.ID != excludeID && run.Status == RunStatusRunning {
			return true
		}
	}
	return false
}

func (r *memoryRepository) latestRunStatusLocked(jobID int64) string {
	var latest *RunRecord
	for index := range r.runs {
		run := &r.runs[index]
		if run.ScheduledJobID != jobID {
			continue
		}
		if latest == nil || run.UpdatedAt.After(latest.UpdatedAt) || (run.UpdatedAt.Equal(latest.UpdatedAt) && run.ID > latest.ID) {
			latest = run
		}
	}
	if latest == nil {
		return ""
	}
	return latest.Status
}

func cloneRecord(item Record) Record {
	item.Config = cloneJSONMap(item.Config)
	item.LastRunAt = cloneTimePtr(item.LastRunAt)
	item.NextRunAt = cloneTimePtr(item.NextRunAt)
	item.LastSuccessAt = cloneTimePtr(item.LastSuccessAt)
	item.LastFailureAt = cloneTimePtr(item.LastFailureAt)
	return item
}

func cloneRunRecord(item RunRecord) RunRecord {
	item.StartedAt = cloneTimePtr(item.StartedAt)
	item.FinishedAt = cloneTimePtr(item.FinishedAt)
	item.ResultSummary = cloneJSONMap(item.ResultSummary)
	if item.WorkerID != nil {
		value := *item.WorkerID
		item.WorkerID = &value
	}
	return item
}

func cloneJSONMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	clone := make(map[string]any, len(value))
	for key, item := range value {
		clone[key] = item
	}
	return clone
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	next := *value
	return &next
}

func min(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func newUID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err == nil {
		raw[6] = (raw[6] & 0x0f) | 0x40
		raw[8] = (raw[8] & 0x3f) | 0x80
		return fmt.Sprintf(
			"%s-%s-%s-%s-%s",
			hex.EncodeToString(raw[0:4]),
			hex.EncodeToString(raw[4:6]),
			hex.EncodeToString(raw[6:8]),
			hex.EncodeToString(raw[8:10]),
			hex.EncodeToString(raw[10:16]),
		)
	}

	now := time.Now().UTC().UnixNano()
	fallback := make([]byte, 16)
	for index := range fallback {
		fallback[index] = byte(now >> ((index % 8) * 8))
	}
	return fmt.Sprintf(
		"%s-%s-%s-%s-%s",
		hex.EncodeToString(fallback[0:4]),
		hex.EncodeToString(fallback[4:6]),
		hex.EncodeToString(fallback[6:8]),
		hex.EncodeToString(fallback[8:10]),
		hex.EncodeToString(fallback[10:16]),
	)
}
