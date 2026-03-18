package dashboard

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestSQLRepositoryGetSnapshot(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer sqlDB.Close()

	repo := NewRepository(sqlx.NewDb(sqlDB, "sqlmock"))
	now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT
			(SELECT COUNT(*) FROM exchange_requests WHERE created_at >= $1) AS requests_today,
			(SELECT COUNT(*) FROM exchange_requests WHERE status IN ('pending', 'blocked')) AS pending_requests,
			(SELECT COUNT(*) FROM delivery_attempts WHERE status IN ('pending', 'retrying')) AS pending_deliveries,
			(SELECT COUNT(*) FROM delivery_attempts WHERE status = 'running') AS running_deliveries,
			(SELECT COUNT(*) FROM delivery_attempts WHERE status = 'failed' AND COALESCE(finished_at, updated_at, created_at) >= $2) AS failed_deliveries_last_hour,
			(SELECT COUNT(*) FROM async_tasks WHERE COALESCE(terminal_state, '') = '' AND LOWER(COALESCE(remote_status, '')) = 'polling') AS polling_jobs,
			(SELECT COUNT(*) FROM ingest_files WHERE status IN ('discovered', 'retry', 'processing')) AS ingest_backlog,
			(SELECT COUNT(*) FROM worker_runs WHERE status = 'running' AND COALESCE(last_heartbeat_at, started_at) >= $3) AS healthy_workers,
			(SELECT COUNT(*) FROM worker_runs WHERE NOT (status = 'running' AND COALESCE(last_heartbeat_at, started_at) >= $3)) AS unhealthy_workers`)).
		WithArgs(now.Truncate(24*time.Hour), now.Add(-time.Hour), now.Add(-workerHeartbeatFreshness)).
		WillReturnRows(sqlmock.NewRows([]string{
			"requests_today", "pending_requests", "pending_deliveries", "running_deliveries",
			"failed_deliveries_last_hour", "polling_jobs", "ingest_backlog", "healthy_workers", "unhealthy_workers",
		}).AddRow(12, 3, 4, 1, 2, 5, 6, 2, 1))

	mock.ExpectQuery("(?s)FROM delivery_attempts d.*WHERE d.status = 'failed'.*LIMIT \\$1").
		WithArgs(attentionLimit).
		WillReturnRows(sqlmock.NewRows([]string{
			"total_count", "id", "uid", "request_id", "request_uid", "server_id", "server_name", "correlation_id", "status", "error_message", "started_at", "finished_at", "next_eligible_at", "updated_at",
		}).AddRow(1, 10, "del-10", 20, "req-20", 30, "DHIS2", "corr-1", "failed", "timeout", now.Add(-30*time.Minute), now.Add(-29*time.Minute), now.Add(10*time.Minute), now.Add(-29*time.Minute)))

	mock.ExpectQuery("(?s)FROM delivery_attempts d.*WHERE d.status = 'running'.*LIMIT \\$2").
		WithArgs(now.Add(-staleRunningDeliveryAfter), attentionLimit).
		WillReturnRows(sqlmock.NewRows([]string{
			"total_count", "id", "uid", "request_id", "request_uid", "server_id", "server_name", "correlation_id", "status", "error_message", "started_at", "finished_at", "next_eligible_at", "updated_at",
		}))

	mock.ExpectQuery("(?s)FROM async_tasks a.*LOWER\\(COALESCE\\(a.remote_status, ''\\)\\) = 'polling'.*LIMIT \\$2").
		WithArgs(now.Add(-stuckJobAfter), attentionLimit).
		WillReturnRows(sqlmock.NewRows([]string{
			"total_count", "id", "uid", "delivery_id", "delivery_uid", "request_id", "request_uid", "correlation_id", "remote_job_id", "remote_status", "current_state", "next_poll_at", "updated_at",
		}).AddRow(1, 40, "job-40", 10, "del-10", 20, "req-20", "corr-1", "remote-1", "polling", "polling", now.Add(-15*time.Minute), now.Add(-11*time.Minute)))

	mock.ExpectQuery("(?s)FROM ingest_files i.*WHERE i.status IN \\('failed', 'retry'\\).*LIMIT \\$2").
		WithArgs(now.Add(-recentIngestFailureWindow), attentionLimit).
		WillReturnRows(sqlmock.NewRows([]string{
			"total_count", "id", "uid", "original_name", "current_path", "status", "last_error_code", "last_error_message", "request_id", "failed_at", "updated_at",
		}).AddRow(1, 50, "ing-50", "batch.json", "/tmp/batch.json", "failed", "INGEST_FILE_READ_ERROR", "read failed", nil, now.Add(-20*time.Minute), now.Add(-20*time.Minute)))

	mock.ExpectQuery("(?s)FROM worker_runs w.*WHERE NOT \\(w.status = 'running' AND COALESCE\\(w.last_heartbeat_at, w.started_at\\) >= \\$1\\).*LIMIT \\$2").
		WithArgs(now.Add(-workerHeartbeatFreshness), attentionLimit).
		WillReturnRows(sqlmock.NewRows([]string{
			"total_count", "id", "uid", "worker_type", "worker_name", "status", "last_heartbeat_at", "started_at", "updated_at",
		}).AddRow(1, 60, "wrk-60", "poll", "poll-worker", "failed", now.Add(-5*time.Minute), now.Add(-30*time.Minute), now.Add(-2*time.Minute)))

	mock.ExpectQuery("(?s)FROM worker_runs w.*ORDER BY w.updated_at DESC, w.id DESC.*LIMIT \\$1").
		WithArgs(workerSummaryLimit).
		WillReturnRows(sqlmock.NewRows([]string{
			"total_count", "id", "uid", "worker_type", "worker_name", "status", "last_heartbeat_at", "started_at", "updated_at",
		}).AddRow(0, 61, "wrk-61", "send", "send-worker", "running", now.Add(-time.Minute), now.Add(-45*time.Minute), now.Add(-time.Minute)))

	mock.ExpectQuery("(?s)FROM exchange_requests.*GROUP BY 1.*ORDER BY 1 ASC").
		WithArgs(now.Add(-trendWindow)).
		WillReturnRows(sqlmock.NewRows([]string{"bucket_start", "count"}).AddRow(now.Add(-2*time.Hour), 7))

	mock.ExpectQuery("(?s)FROM delivery_attempts.*GROUP BY 1, 2.*ORDER BY 1 ASC, 2 ASC").
		WithArgs(now.Add(-trendWindow)).
		WillReturnRows(sqlmock.NewRows([]string{"bucket_start", "status", "count"}).AddRow(now.Add(-2*time.Hour), "failed", 2))

	mock.ExpectQuery("(?s)FROM async_tasks.*GROUP BY 1, 2.*ORDER BY 1 ASC, 2 ASC").
		WithArgs(now.Add(-trendWindow)).
		WillReturnRows(sqlmock.NewRows([]string{"bucket_start", "status", "count"}).AddRow(now.Add(-2*time.Hour), "polling", 3))

	mock.ExpectQuery("(?s)FROM delivery_attempts d.*WHERE d.status = 'failed'.*LIMIT 5").
		WithArgs(now.Add(-trendWindow)).
		WillReturnRows(sqlmock.NewRows([]string{"server_id", "server_name", "count"}).AddRow(30, "DHIS2", 2))

	mock.ExpectQuery("(?s)FROM request_events e.*ORDER BY e.created_at DESC, e.id DESC.*LIMIT \\$1").
		WithArgs(workerSummaryLimit).
		WillReturnRows(sqlmock.NewRows([]string{
			"event_type", "created_at", "event_level", "message", "correlation_id", "request_id", "request_uid", "delivery_attempt_id", "delivery_uid", "async_task_id", "async_task_uid", "worker_run_id", "worker_run_uid",
		}).AddRow("delivery.failed", now.Add(-5*time.Minute), "error", "Delivery failed", "corr-1", 20, "req-20", 10, "del-10", nil, "", nil, ""))

	snapshot, err := repo.GetSnapshot(context.Background(), now)
	if err != nil {
		t.Fatalf("get snapshot: %v", err)
	}
	if snapshot.KPIs.RequestsToday != 12 {
		t.Fatalf("expected requestsToday 12, got %+v", snapshot.KPIs)
	}
	if snapshot.Health.Status != "degraded" {
		t.Fatalf("expected degraded health, got %+v", snapshot.Health)
	}
	if snapshot.Attention.FailedDeliveries.Total != 1 || len(snapshot.Attention.FailedDeliveries.Items) != 1 {
		t.Fatalf("expected failed delivery attention item, got %+v", snapshot.Attention.FailedDeliveries)
	}
	if len(snapshot.RecentEvents) != 1 || snapshot.RecentEvents[0].EntityType != "delivery" {
		t.Fatalf("expected delivery recent event, got %+v", snapshot.RecentEvents)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}
