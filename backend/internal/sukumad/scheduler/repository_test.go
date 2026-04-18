package scheduler

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestSQLRepositoryListScheduledJobs(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer sqlDB.Close()

	repo := NewSQLRepository(sqlx.NewDb(sqlDB, "sqlmock"))
	now := time.Now().UTC()
	nextRun := now.Add(15 * time.Minute)
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	dataRows := sqlmock.NewRows([]string{
		"id", "uid", "code", "name", "description", "job_category", "job_type", "schedule_type", "schedule_expr",
		"timezone", "enabled", "allow_concurrent_runs", "config", "last_run_at", "next_run_at", "last_success_at", "last_failure_at", "latest_run_status",
		"created_at", "updated_at",
	}).AddRow(
		1, "11111111-1111-1111-1111-111111111111", "nightly-sync", "Nightly Sync", "sync nightly", "integration", "dhis2.sync", "interval", "15m",
		"UTC", true, false, []byte(`{"serverCode":"dhis2"}`), nil, nextRun, nil, nil, "succeeded", now, now,
	)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM scheduled_jobs WHERE (code ILIKE $1 OR name ILIKE $1 OR description ILIKE $1 OR job_type ILIKE $1) AND job_category = $2`)).
		WithArgs("%sync%", "integration").
		WillReturnRows(countRows)
	mock.ExpectQuery(regexp.QuoteMeta(`
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
	 WHERE (code ILIKE $1 OR name ILIKE $1 OR description ILIKE $1 OR job_type ILIKE $1) AND job_category = $2 ORDER BY name ASC LIMIT $3 OFFSET $4`)).
		WithArgs("%sync%", "integration", 25, 0).
		WillReturnRows(dataRows)

	result, err := repo.ListScheduledJobs(context.Background(), ListQuery{
		Page:      1,
		PageSize:  25,
		SortField: "name",
		SortOrder: "asc",
		Filter:    "sync",
		Category:  "integration",
	})
	if err != nil {
		t.Fatalf("list scheduled jobs: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("unexpected list result: %+v", result)
	}
	if result.Items[0].Code != "nightly-sync" {
		t.Fatalf("expected nightly-sync code, got %q", result.Items[0].Code)
	}
	if result.Items[0].Config["serverCode"] != "dhis2" {
		t.Fatalf("expected config to decode, got %+v", result.Items[0].Config)
	}
}

func TestSQLRepositoryCreateJobRun(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer sqlDB.Close()

	repo := NewSQLRepository(sqlx.NewDb(sqlDB, "sqlmock"))
	now := time.Now().UTC()
	insertRows := sqlmock.NewRows([]string{
		"id", "uid", "scheduled_job_id", "scheduled_job_uid", "scheduled_job_code", "scheduled_job_name",
		"trigger_mode", "scheduled_for", "started_at", "finished_at", "status", "worker_id", "error_message", "result_summary", "created_at", "updated_at",
	}).AddRow(
		4, "22222222-2222-2222-2222-222222222222", 2, "", "", "", "manual", now, nil, nil, "pending", nil, "", []byte(`{"message":"queued"}`), now, now,
	)
	getRows := sqlmock.NewRows([]string{
		"id", "uid", "scheduled_job_id", "scheduled_job_uid", "scheduled_job_code", "scheduled_job_name",
		"trigger_mode", "scheduled_for", "started_at", "finished_at", "status", "worker_id", "error_message", "result_summary", "created_at", "updated_at",
	}).AddRow(
		4, "22222222-2222-2222-2222-222222222222", 2, "11111111-1111-1111-1111-111111111111", "nightly-sync", "Nightly Sync", "manual", now, nil, nil, "pending", nil, "", []byte(`{"message":"queued"}`), now, now,
	)

	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO scheduled_job_runs (
			uid, scheduled_job_id, trigger_mode, scheduled_for, status, result_summary, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, NOW(), NOW())
		RETURNING id, uid::text AS uid, scheduled_job_id, '' AS scheduled_job_uid, '' AS scheduled_job_code, '' AS scheduled_job_name,
		          trigger_mode, scheduled_for, started_at, finished_at, status, worker_id, COALESCE(error_message, '') AS error_message,
		          result_summary, created_at, updated_at
	`)).
		WithArgs("22222222-2222-2222-2222-222222222222", int64(2), "manual", now, "pending", `{"message":"queued"}`).
		WillReturnRows(insertRows)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT r.id, r.uid::text AS uid, r.scheduled_job_id,
		       j.uid::text AS scheduled_job_uid, j.code AS scheduled_job_code, j.name AS scheduled_job_name,
		       r.trigger_mode, r.scheduled_for, r.started_at, r.finished_at, r.status, r.worker_id,
		       COALESCE(r.error_message, '') AS error_message, r.result_summary, r.created_at, r.updated_at
		FROM scheduled_job_runs r
		INNER JOIN scheduled_jobs j ON j.id = r.scheduled_job_id
		WHERE r.id = $1
	`)).
		WithArgs(int64(4)).
		WillReturnRows(getRows)

	record, err := repo.CreateJobRun(context.Background(), CreateRunParams{
		UID:            "22222222-2222-2222-2222-222222222222",
		ScheduledJobID: 2,
		TriggerMode:    TriggerModeManual,
		ScheduledFor:   now,
		Status:         RunStatusPending,
		ResultSummary:  map[string]any{"message": "queued"},
	})
	if err != nil {
		t.Fatalf("create job run: %v", err)
	}
	if record.ScheduledJobCode != "nightly-sync" {
		t.Fatalf("expected joined job code, got %q", record.ScheduledJobCode)
	}
}
