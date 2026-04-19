package scheduler

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

const stuckRequestReason = "stuck_request_timeout"

type jobConfigValidator interface {
	ValidateConfig(map[string]any) map[string]any
}

type maintenanceRepository interface {
	ListArchivableRequests(context.Context, time.Time, int) ([]maintenanceRequestCandidate, error)
	ArchiveRequests(context.Context, []int64, time.Time, string) (int, error)
	ListStuckRequests(context.Context, time.Time, int) ([]maintenanceRequestCandidate, error)
	MarkRequestsStuck(context.Context, []int64, string) (int, error)
	ListOrphanedIngestFiles(context.Context, time.Time, int) ([]ingestFileCandidate, error)
	DeleteIngestFiles(context.Context, []int64) (int, error)
	PurgeOldLogs(context.Context, time.Time, int) (logPurgeResult, error)
}

type maintenanceRequestCandidate struct {
	RequestID  int64     `db:"request_id"`
	RequestUID string    `db:"request_uid"`
	Status     string    `db:"status"`
	UpdatedAt  time.Time `db:"updated_at"`
}

type ingestFileCandidate struct {
	ID        int64     `db:"id"`
	UID       string    `db:"uid"`
	Status    string    `db:"status"`
	UpdatedAt time.Time `db:"updated_at"`
}

type logPurgeResult struct {
	AuditLogs      int `json:"auditLogs"`
	RequestEvents  int `json:"requestEvents"`
	AsyncTaskPolls int `json:"asyncTaskPolls"`
	WorkerRuns     int `json:"workerRuns"`
}

type maintenanceSQLRepository struct {
	db *sqlx.DB
}

func newMaintenanceRepository(repository Repository) maintenanceRepository {
	sqlRepo, ok := repository.(*SQLRepository)
	if !ok || sqlRepo == nil || sqlRepo.db == nil {
		return nil
	}
	return &maintenanceSQLRepository{db: sqlRepo.db}
}

func (r *maintenanceSQLRepository) ListArchivableRequests(ctx context.Context, cutoff time.Time, limit int) ([]maintenanceRequestCandidate, error) {
	if limit <= 0 {
		limit = 100
	}
	items := []maintenanceRequestCandidate{}
	if err := r.db.SelectContext(ctx, &items, `
		SELECT r.id AS request_id,
		       r.uid::text AS request_uid,
		       r.status,
		       r.updated_at
		FROM exchange_requests r
		WHERE r.status IN ('completed', 'failed')
		  AND r.updated_at <= $1
		  AND COALESCE(r.extras->'maintenance'->>'archived', 'false') <> 'true'
		ORDER BY r.updated_at ASC, r.id ASC
		LIMIT $2
	`, cutoff.UTC(), limit); err != nil {
		return nil, fmt.Errorf("list archivable requests: %w", err)
	}
	return items, nil
}

func (r *maintenanceSQLRepository) ArchiveRequests(ctx context.Context, requestIDs []int64, archivedAt time.Time, runUID string) (int, error) {
	if len(requestIDs) == 0 {
		return 0, nil
	}
	query, args, err := sqlx.In(`
		UPDATE exchange_requests
		SET extras = COALESCE(extras, '{}'::jsonb) || jsonb_build_object(
			'maintenance',
			COALESCE(extras->'maintenance', '{}'::jsonb) || jsonb_build_object(
				'archived', true,
				'archivedAt', ?,
				'archiveRunUid', ?,
				'archiveSource', 'scheduler'
			)
		),
		    updated_at = NOW()
		WHERE id IN (?)
		  AND COALESCE(extras->'maintenance'->>'archived', 'false') <> 'true'
	`, archivedAt.UTC().Format(time.RFC3339), strings.TrimSpace(runUID), requestIDs)
	if err != nil {
		return 0, fmt.Errorf("build archive requests query: %w", err)
	}
	query = r.db.Rebind(query)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("archive requests: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("archive requests rows affected: %w", err)
	}
	return int(affected), nil
}

func (r *maintenanceSQLRepository) ListStuckRequests(ctx context.Context, cutoff time.Time, limit int) ([]maintenanceRequestCandidate, error) {
	if limit <= 0 {
		limit = 100
	}
	items := []maintenanceRequestCandidate{}
	if err := r.db.SelectContext(ctx, &items, `
		SELECT DISTINCT r.id AS request_id,
		       r.uid::text AS request_uid,
		       r.status,
		       r.updated_at
		FROM exchange_requests r
		INNER JOIN request_targets t ON t.request_id = r.id
		WHERE r.status IN ('pending', 'processing')
		  AND r.updated_at <= $1
		  AND t.status IN ('pending', 'processing')
		ORDER BY r.updated_at ASC, r.id ASC
		LIMIT $2
	`, cutoff.UTC(), limit); err != nil {
		return nil, fmt.Errorf("list stuck requests: %w", err)
	}
	return items, nil
}

func (r *maintenanceSQLRepository) MarkRequestsStuck(ctx context.Context, requestIDs []int64, reason string) (int, error) {
	if len(requestIDs) == 0 {
		return 0, nil
	}
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin mark stuck tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	targetQuery, targetArgs, err := sqlx.In(`
		UPDATE request_targets
		SET status = 'blocked',
		    blocked_reason = ?,
		    deferred_until = NULL,
		    updated_at = NOW()
		WHERE request_id IN (?)
		  AND status IN ('pending', 'processing')
	`, reason, requestIDs)
	if err != nil {
		return 0, fmt.Errorf("build mark stuck targets query: %w", err)
	}
	targetQuery = tx.Rebind(targetQuery)
	if _, err := tx.ExecContext(ctx, targetQuery, targetArgs...); err != nil {
		return 0, fmt.Errorf("mark stuck targets: %w", err)
	}

	requestQuery, requestArgs, err := sqlx.In(`
		UPDATE exchange_requests
		SET status = 'blocked',
		    status_reason = ?,
		    deferred_until = NULL,
		    updated_at = NOW()
		WHERE id IN (?)
		  AND status IN ('pending', 'processing')
	`, reason, requestIDs)
	if err != nil {
		return 0, fmt.Errorf("build mark stuck requests query: %w", err)
	}
	requestQuery = tx.Rebind(requestQuery)
	result, err := tx.ExecContext(ctx, requestQuery, requestArgs...)
	if err != nil {
		return 0, fmt.Errorf("mark stuck requests: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("mark stuck requests rows affected: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit mark stuck tx: %w", err)
	}
	return int(affected), nil
}

func (r *maintenanceSQLRepository) ListOrphanedIngestFiles(ctx context.Context, cutoff time.Time, limit int) ([]ingestFileCandidate, error) {
	if limit <= 0 {
		limit = 100
	}
	items := []ingestFileCandidate{}
	if err := r.db.SelectContext(ctx, &items, `
		SELECT i.id, i.uid::text AS uid, i.status, i.updated_at
		FROM ingest_files i
		WHERE i.request_id IS NULL
		  AND i.status IN ('processed', 'failed')
		  AND i.updated_at <= $1
		ORDER BY i.updated_at ASC, i.id ASC
		LIMIT $2
	`, cutoff.UTC(), limit); err != nil {
		return nil, fmt.Errorf("list orphaned ingest files: %w", err)
	}
	return items, nil
}

func (r *maintenanceSQLRepository) DeleteIngestFiles(ctx context.Context, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	query, args, err := sqlx.In(`DELETE FROM ingest_files WHERE id IN (?)`, ids)
	if err != nil {
		return 0, fmt.Errorf("build delete ingest files query: %w", err)
	}
	query = r.db.Rebind(query)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("delete orphaned ingest files: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("delete orphaned ingest files rows affected: %w", err)
	}
	return int(affected), nil
}

func (r *maintenanceSQLRepository) PurgeOldLogs(ctx context.Context, cutoff time.Time, batchSize int) (logPurgeResult, error) {
	if batchSize <= 0 {
		batchSize = 100
	}
	remaining := batchSize
	result := logPurgeResult{}

	deleteWithLimit := func(query string, limit int, args ...any) (int, error) {
		if limit <= 0 {
			return 0, nil
		}
		row := r.db.QueryRowxContext(ctx, query, append(args, limit)...)
		var deleted int
		if err := row.Scan(&deleted); err != nil {
			return 0, err
		}
		return deleted, nil
	}

	var err error
	result.AuditLogs, err = deleteWithLimit(`
		WITH doomed AS (
			SELECT id
			FROM audit_logs
			WHERE timestamp <= $1
			ORDER BY timestamp ASC, id ASC
			LIMIT $2
		), deleted AS (
			DELETE FROM audit_logs
			WHERE id IN (SELECT id FROM doomed)
		)
		SELECT COUNT(*) FROM deleted
	`, remaining, cutoff.UTC())
	if err != nil {
		return logPurgeResult{}, fmt.Errorf("purge audit logs: %w", err)
	}
	remaining -= result.AuditLogs

	result.RequestEvents, err = deleteWithLimit(`
		WITH doomed AS (
			SELECT id
			FROM request_events
			WHERE created_at <= $1
			ORDER BY created_at ASC, id ASC
			LIMIT $2
		), deleted AS (
			DELETE FROM request_events
			WHERE id IN (SELECT id FROM doomed)
		)
		SELECT COUNT(*) FROM deleted
	`, remaining, cutoff.UTC())
	if err != nil {
		return logPurgeResult{}, fmt.Errorf("purge request events: %w", err)
	}
	remaining -= result.RequestEvents

	result.AsyncTaskPolls, err = deleteWithLimit(`
		WITH doomed AS (
			SELECT id
			FROM async_task_polls
			WHERE polled_at <= $1
			ORDER BY polled_at ASC, id ASC
			LIMIT $2
		), deleted AS (
			DELETE FROM async_task_polls
			WHERE id IN (SELECT id FROM doomed)
		)
		SELECT COUNT(*) FROM deleted
	`, remaining, cutoff.UTC())
	if err != nil {
		return logPurgeResult{}, fmt.Errorf("purge async task polls: %w", err)
	}
	remaining -= result.AsyncTaskPolls

	result.WorkerRuns, err = deleteWithLimit(`
		WITH doomed AS (
			SELECT id
			FROM worker_runs
			WHERE status <> 'running'
			  AND COALESCE(stopped_at, updated_at, started_at) <= $1
			ORDER BY COALESCE(stopped_at, updated_at, started_at) ASC, id ASC
			LIMIT $2
		), deleted AS (
			DELETE FROM worker_runs
			WHERE id IN (SELECT id FROM doomed)
		)
		SELECT COUNT(*) FROM deleted
	`, remaining, cutoff.UTC())
	if err != nil {
		return logPurgeResult{}, fmt.Errorf("purge worker runs: %w", err)
	}

	return result, nil
}

type maintenanceHandlerDependencies struct {
	repo maintenanceRepository
	now  func() time.Time
}

func newMaintenanceHandlers(deps maintenanceHandlerDependencies) map[string]typedRegistration {
	now := deps.now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return map[string]typedRegistration{
		"archive_old_requests": typedRegistration{
			handler: typedHandlerWithValidate[archiveOldRequestsConfig](
				func(ctx context.Context, exec JobExecution, cfg archiveOldRequestsConfig) (JobResult, error) {
					return runArchiveOldRequests(ctx, exec, cfg, deps.repo, now)
				},
				validateArchiveOldRequestsConfig,
			),
		},
		"purge_old_logs": typedRegistration{
			handler: typedHandlerWithValidate[purgeOldLogsConfig](
				func(ctx context.Context, exec JobExecution, cfg purgeOldLogsConfig) (JobResult, error) {
					return runPurgeOldLogs(ctx, exec, cfg, deps.repo, now)
				},
				validatePurgeOldLogsConfig,
			),
		},
		"mark_stuck_requests": typedRegistration{
			handler: typedHandlerWithValidate[markStuckRequestsConfig](
				func(ctx context.Context, exec JobExecution, cfg markStuckRequestsConfig) (JobResult, error) {
					return runMarkStuckRequests(ctx, exec, cfg, deps.repo, now)
				},
				validateMarkStuckRequestsConfig,
			),
		},
		"cleanup_orphaned_records": typedRegistration{
			handler: typedHandlerWithValidate[cleanupOrphanedRecordsConfig](
				func(ctx context.Context, exec JobExecution, cfg cleanupOrphanedRecordsConfig) (JobResult, error) {
					return runCleanupOrphanedRecords(ctx, exec, cfg, deps.repo, now)
				},
				validateCleanupOrphanedRecordsConfig,
			),
		},
	}
}

func runArchiveOldRequests(ctx context.Context, exec JobExecution, cfg archiveOldRequestsConfig, repo maintenanceRepository, now func() time.Time) (JobResult, error) {
	if repo == nil {
		return JobResult{}, errNoMaintenanceRepository
	}
	cutoff := now().AddDate(0, 0, -cfg.MaxAgeDays).UTC()
	candidates, err := repo.ListArchivableRequests(ctx, cutoff, cfg.BatchSize)
	if err != nil {
		return JobResult{}, err
	}
	summary := baseMaintenanceSummary(exec.Job.JobType, exec.Run.UID, cfg.DryRun, cfg.BatchSize)
	summary["cutoff"] = cutoff.Format(time.RFC3339)
	summary["maxAgeDays"] = cfg.MaxAgeDays
	summary["scanned_count"] = len(candidates)
	summary["affected_count"] = len(candidates)
	if cfg.DryRun {
		summary["skipped_count"] = len(candidates)
		return JobResult{Status: RunStatusSucceeded, ResultSummary: summary}, nil
	}
	ids := candidateIDs(candidates)
	archivedCount, err := repo.ArchiveRequests(ctx, ids, now().UTC(), exec.Run.UID)
	if err != nil {
		return JobResult{}, err
	}
	summary["archived_count"] = archivedCount
	summary["skipped_count"] = maxInt(0, len(candidates)-archivedCount)
	return JobResult{Status: RunStatusSucceeded, ResultSummary: summary}, nil
}

func runPurgeOldLogs(ctx context.Context, exec JobExecution, cfg purgeOldLogsConfig, repo maintenanceRepository, now func() time.Time) (JobResult, error) {
	if repo == nil {
		return JobResult{}, errNoMaintenanceRepository
	}
	cutoff := now().AddDate(0, 0, -cfg.MaxAgeDays).UTC()
	summary := baseMaintenanceSummary(exec.Job.JobType, exec.Run.UID, cfg.DryRun, cfg.BatchSize)
	summary["cutoff"] = cutoff.Format(time.RFC3339)
	summary["maxAgeDays"] = cfg.MaxAgeDays
	if cfg.DryRun {
		summary["skipped_count"] = cfg.BatchSize
		return JobResult{Status: RunStatusSucceeded, ResultSummary: summary}, nil
	}
	counts, err := repo.PurgeOldLogs(ctx, cutoff, cfg.BatchSize)
	if err != nil {
		return JobResult{}, err
	}
	deleted := counts.AuditLogs + counts.RequestEvents + counts.AsyncTaskPolls + counts.WorkerRuns
	summary["scanned_count"] = deleted
	summary["affected_count"] = deleted
	summary["deleted_count"] = deleted
	summary["deletedByTable"] = map[string]int{
		"auditLogs":      counts.AuditLogs,
		"requestEvents":  counts.RequestEvents,
		"asyncTaskPolls": counts.AsyncTaskPolls,
		"workerRuns":     counts.WorkerRuns,
	}
	return JobResult{Status: RunStatusSucceeded, ResultSummary: summary}, nil
}

func runMarkStuckRequests(ctx context.Context, exec JobExecution, cfg markStuckRequestsConfig, repo maintenanceRepository, now func() time.Time) (JobResult, error) {
	if repo == nil {
		return JobResult{}, errNoMaintenanceRepository
	}
	cutoff := now().Add(-cfg.staleCutoff()).UTC()
	candidates, err := repo.ListStuckRequests(ctx, cutoff, cfg.BatchSize)
	if err != nil {
		return JobResult{}, err
	}
	summary := baseMaintenanceSummary(exec.Job.JobType, exec.Run.UID, cfg.DryRun, cfg.BatchSize)
	summary["cutoff"] = cutoff.Format(time.RFC3339)
	summary["staleCutoffMinutes"] = cfg.effectiveCutoffMinutes()
	summary["scanned_count"] = len(candidates)
	summary["affected_count"] = len(candidates)
	if cfg.DryRun {
		summary["skipped_count"] = len(candidates)
		return JobResult{Status: RunStatusSucceeded, ResultSummary: summary}, nil
	}
	affected, err := repo.MarkRequestsStuck(ctx, candidateIDs(candidates), stuckRequestReason)
	if err != nil {
		return JobResult{}, err
	}
	summary["skipped_count"] = maxInt(0, len(candidates)-affected)
	return JobResult{Status: RunStatusSucceeded, ResultSummary: summary}, nil
}

func runCleanupOrphanedRecords(ctx context.Context, exec JobExecution, cfg cleanupOrphanedRecordsConfig, repo maintenanceRepository, now func() time.Time) (JobResult, error) {
	if repo == nil {
		return JobResult{}, errNoMaintenanceRepository
	}
	cutoff := now().AddDate(0, 0, -cfg.MaxAgeDays).UTC()
	candidates, err := repo.ListOrphanedIngestFiles(ctx, cutoff, cfg.BatchSize)
	if err != nil {
		return JobResult{}, err
	}
	summary := baseMaintenanceSummary(exec.Job.JobType, exec.Run.UID, cfg.DryRun, cfg.BatchSize)
	summary["cutoff"] = cutoff.Format(time.RFC3339)
	summary["maxAgeDays"] = cfg.MaxAgeDays
	summary["scanned_count"] = len(candidates)
	summary["affected_count"] = len(candidates)
	if cfg.DryRun {
		summary["skipped_count"] = len(candidates)
		return JobResult{Status: RunStatusSucceeded, ResultSummary: summary}, nil
	}
	deleted, err := repo.DeleteIngestFiles(ctx, ingestFileIDs(candidates))
	if err != nil {
		return JobResult{}, err
	}
	summary["deleted_count"] = deleted
	summary["skipped_count"] = maxInt(0, len(candidates)-deleted)
	return JobResult{Status: RunStatusSucceeded, ResultSummary: summary}, nil
}

func baseMaintenanceSummary(jobType string, runUID string, dryRun bool, batchSize int) map[string]any {
	return map[string]any{
		"jobType":        jobType,
		"runUid":         runUID,
		"dryRun":         dryRun,
		"batchSize":      batchSize,
		"scanned_count":  0,
		"affected_count": 0,
		"archived_count": 0,
		"deleted_count":  0,
		"skipped_count":  0,
	}
}

func candidateIDs(items []maintenanceRequestCandidate) []int64 {
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		if item.RequestID > 0 {
			ids = append(ids, item.RequestID)
		}
	}
	return ids
}

func ingestFileIDs(items []ingestFileCandidate) []int64 {
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		if item.ID > 0 {
			ids = append(ids, item.ID)
		}
	}
	return ids
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

type typedRegistration struct {
	handler JobHandler
}

func validateArchiveOldRequestsConfig(cfg archiveOldRequestsConfig) map[string]any {
	details := map[string]any{}
	if cfg.BatchSize <= 0 {
		details["config.batchSize"] = []string{"must be greater than zero"}
	}
	if cfg.MaxAgeDays <= 0 {
		details["config.maxAgeDays"] = []string{"must be greater than zero"}
	}
	return details
}

func validatePurgeOldLogsConfig(cfg purgeOldLogsConfig) map[string]any {
	details := map[string]any{}
	if cfg.BatchSize <= 0 {
		details["config.batchSize"] = []string{"must be greater than zero"}
	}
	if cfg.MaxAgeDays <= 0 {
		details["config.maxAgeDays"] = []string{"must be greater than zero"}
	}
	return details
}

func validateMarkStuckRequestsConfig(cfg markStuckRequestsConfig) map[string]any {
	details := map[string]any{}
	if cfg.BatchSize <= 0 {
		details["config.batchSize"] = []string{"must be greater than zero"}
	}
	if cfg.effectiveCutoffMinutes() <= 0 {
		details["config.staleCutoffMinutes"] = []string{"must be greater than zero"}
	}
	return details
}

func validateCleanupOrphanedRecordsConfig(cfg cleanupOrphanedRecordsConfig) map[string]any {
	details := map[string]any{}
	if cfg.BatchSize <= 0 {
		details["config.batchSize"] = []string{"must be greater than zero"}
	}
	if cfg.MaxAgeDays <= 0 {
		details["config.maxAgeDays"] = []string{"must be greater than zero"}
	}
	return details
}

type fakeMaintenanceRepository struct {
	archivableRequests []maintenanceRequestCandidate
	stuckRequests      []maintenanceRequestCandidate
	orphanedIngest     []ingestFileCandidate
	purgeResult        logPurgeResult
	archivedIDs        []int64
	stuckIDs           []int64
	deletedIDs         []int64
}

func (f *fakeMaintenanceRepository) ListArchivableRequests(_ context.Context, _ time.Time, _ int) ([]maintenanceRequestCandidate, error) {
	return append([]maintenanceRequestCandidate{}, f.archivableRequests...), nil
}

func (f *fakeMaintenanceRepository) ArchiveRequests(_ context.Context, ids []int64, _ time.Time, _ string) (int, error) {
	f.archivedIDs = append([]int64{}, ids...)
	return len(ids), nil
}

func (f *fakeMaintenanceRepository) ListStuckRequests(_ context.Context, _ time.Time, _ int) ([]maintenanceRequestCandidate, error) {
	return append([]maintenanceRequestCandidate{}, f.stuckRequests...), nil
}

func (f *fakeMaintenanceRepository) MarkRequestsStuck(_ context.Context, ids []int64, _ string) (int, error) {
	f.stuckIDs = append([]int64{}, ids...)
	return len(ids), nil
}

func (f *fakeMaintenanceRepository) ListOrphanedIngestFiles(_ context.Context, _ time.Time, _ int) ([]ingestFileCandidate, error) {
	return append([]ingestFileCandidate{}, f.orphanedIngest...), nil
}

func (f *fakeMaintenanceRepository) DeleteIngestFiles(_ context.Context, ids []int64) (int, error) {
	f.deletedIDs = append([]int64{}, ids...)
	return len(ids), nil
}

func (f *fakeMaintenanceRepository) PurgeOldLogs(_ context.Context, _ time.Time, _ int) (logPurgeResult, error) {
	return f.purgeResult, nil
}

var errNoMaintenanceRepository = fmt.Errorf("maintenance repository unavailable")
