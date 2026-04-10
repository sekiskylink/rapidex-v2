package request

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestSQLRepositoryListRequests(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer sqlDB.Close()

	repo := NewSQLRepository(sqlx.NewDb(sqlDB, "sqlmock"))
	now := time.Now().UTC()
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	dataRows := sqlmock.NewRows([]string{
		"id", "uid", "source_system", "destination_server_id", "destination_server_uid", "destination_server_name", "destination_server_code", "batch_id", "correlation_id",
		"idempotency_key", "payload_body", "payload_format", "submission_binding", "response_body_persistence", "url_suffix", "status", "status_reason", "deferred_until", "extras", "created_at", "updated_at", "created_by",
		"latest_delivery_id", "latest_delivery_uid", "latest_delivery_status", "latest_async_task_id", "latest_async_task_uid", "latest_async_state", "latest_async_remote_job_id", "latest_async_poll_url",
	}).AddRow(
		8, "11111111-1111-1111-1111-111111111111", "emr", 3, "srv-1", "DHIS2 Uganda", "dhis2-ug", "batch-1", "corr-1",
		"idem-1", `{"trackedEntity":"123"}`, "json", "body", "", "/api/data", "pending", "", nil, []byte(`{"priority":"high"}`), now, now, int64(7),
		nil, "", "", nil, "", "", "", "",
	)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) 
		FROM exchange_requests r
		LEFT JOIN integration_servers s ON s.id = r.destination_server_id
		LEFT JOIN LATERAL (
			SELECT d.id,
			       d.uid::text AS uid,
			       d.status
			FROM delivery_attempts d
			WHERE d.request_id = r.id
			ORDER BY d.attempt_number DESC, d.created_at DESC
			LIMIT 1
		) ld ON TRUE
		LEFT JOIN async_tasks a ON a.delivery_attempt_id = ld.id
		 WHERE (
			r.uid::text ILIKE $1 OR
			COALESCE(r.source_system, '') ILIKE $1 OR
			COALESCE(r.correlation_id, '') ILIKE $1 OR
			COALESCE(r.batch_id, '') ILIKE $1 OR
			COALESCE(r.idempotency_key, '') ILIKE $1 OR
			COALESCE(r.url_suffix, '') ILIKE $1 OR
			COALESCE(s.name, '') ILIKE $1 OR
			COALESCE(s.code, '') ILIKE $1
		) AND r.status = $2`)).
		WithArgs("%dhis%", "pending").
		WillReturnRows(countRows)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT r.id, r.uid::text AS uid, r.source_system, r.destination_server_id,
		       COALESCE(s.uid::text, '') AS destination_server_uid,
		       COALESCE(s.name, '') AS destination_server_name,
		       COALESCE(s.code, '') AS destination_server_code,
		       COALESCE(r.batch_id, '') AS batch_id, COALESCE(r.correlation_id, '') AS correlation_id, COALESCE(r.idempotency_key, '') AS idempotency_key,
		       r.payload_body, r.payload_format, r.submission_binding, COALESCE(r.response_body_persistence, '') AS response_body_persistence, COALESCE(r.url_suffix, '') AS url_suffix, r.status, COALESCE(r.status_reason, '') AS status_reason, r.deferred_until,
		       r.extras, r.created_at, r.updated_at, r.created_by,
		       ld.id AS latest_delivery_id,
		       COALESCE(ld.uid, '') AS latest_delivery_uid,
		       COALESCE(ld.status, '') AS latest_delivery_status,
		       a.id AS latest_async_task_id,
		       COALESCE(a.uid::text, '') AS latest_async_task_uid,
		       COALESCE(a.terminal_state, CASE WHEN COALESCE(a.remote_status, '') = '' THEN '' ELSE a.remote_status END) AS latest_async_state,
		       COALESCE(a.remote_job_id, '') AS latest_async_remote_job_id,
		       COALESCE(a.poll_url, '') AS latest_async_poll_url
		FROM exchange_requests r
		LEFT JOIN integration_servers s ON s.id = r.destination_server_id
		LEFT JOIN LATERAL (
			SELECT d.id,
			       d.uid::text AS uid,
			       d.status
			FROM delivery_attempts d
			WHERE d.request_id = r.id
			ORDER BY d.attempt_number DESC, d.created_at DESC
			LIMIT 1
		) ld ON TRUE
		LEFT JOIN async_tasks a ON a.delivery_attempt_id = ld.id
		 WHERE (
			r.uid::text ILIKE $1 OR
			COALESCE(r.source_system, '') ILIKE $1 OR
			COALESCE(r.correlation_id, '') ILIKE $1 OR
			COALESCE(r.batch_id, '') ILIKE $1 OR
			COALESCE(r.idempotency_key, '') ILIKE $1 OR
			COALESCE(r.url_suffix, '') ILIKE $1 OR
			COALESCE(s.name, '') ILIKE $1 OR
			COALESCE(s.code, '') ILIKE $1
		) AND r.status = $2 ORDER BY r.created_at DESC LIMIT $3 OFFSET $4`)).
		WithArgs("%dhis%", "pending", 25, 0).
		WillReturnRows(dataRows)
	mock.ExpectQuery("(?s)SELECT t.id, t.uid::text AS uid, t.request_id, t.server_id, .*WHERE t.request_id IN \\(\\?\\).*").
		WithArgs(int64(8)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "uid", "request_id", "server_id", "server_uid", "server_name", "server_code", "target_kind", "priority", "status", "blocked_reason", "deferred_until", "last_released_at",
			"latest_delivery_id", "latest_delivery_uid", "latest_delivery_status", "latest_async_task_id", "latest_async_task_uid", "latest_async_state", "latest_async_remote_job_id", "latest_async_poll_url", "created_at", "updated_at",
		}))
	mock.ExpectQuery("(?s)SELECT d.request_id, d.depends_on_request_id, .*WHERE d.request_id IN \\(\\?\\).*").
		WithArgs(int64(8)).
		WillReturnRows(sqlmock.NewRows([]string{
			"request_id", "depends_on_request_id", "request_uid", "depends_on_uid", "status", "status_reason", "deferred_until", "depends_on_destination_server_name",
		}))

	result, err := repo.ListRequests(context.Background(), ListQuery{
		Page:      1,
		PageSize:  25,
		SortField: "createdAt",
		SortOrder: "desc",
		Filter:    "dhis",
		Status:    "pending",
	})
	if err != nil {
		t.Fatalf("list requests: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("expected one request, got total=%d items=%d", result.Total, len(result.Items))
	}
	if result.Items[0].DestinationServerName != "DHIS2 Uganda" {
		t.Fatalf("expected destination server name, got %+v", result.Items[0])
	}
	payload, ok := result.Items[0].Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected decoded json payload, got %#v", result.Items[0].Payload)
	}
	if payload["trackedEntity"] != "123" {
		t.Fatalf("expected decoded trackedEntity payload, got %+v", payload)
	}
}

func TestSQLRepositoryGetRequestByIDNotFound(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer sqlDB.Close()

	repo := NewSQLRepository(sqlx.NewDb(sqlDB, "sqlmock"))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT r.id, r.uid::text AS uid, r.source_system, r.destination_server_id,
		       COALESCE(s.uid::text, '') AS destination_server_uid,
		       COALESCE(s.name, '') AS destination_server_name,
		       COALESCE(s.code, '') AS destination_server_code,
		       COALESCE(r.batch_id, '') AS batch_id, COALESCE(r.correlation_id, '') AS correlation_id, COALESCE(r.idempotency_key, '') AS idempotency_key,
		       r.payload_body, r.payload_format, r.submission_binding, COALESCE(r.response_body_persistence, '') AS response_body_persistence, COALESCE(r.url_suffix, '') AS url_suffix, r.status, COALESCE(r.status_reason, '') AS status_reason, r.deferred_until,
		       r.extras, r.created_at, r.updated_at, r.created_by,
		       ld.id AS latest_delivery_id,
		       COALESCE(ld.uid, '') AS latest_delivery_uid,
		       COALESCE(ld.status, '') AS latest_delivery_status,
		       a.id AS latest_async_task_id,
		       COALESCE(a.uid::text, '') AS latest_async_task_uid,
		       COALESCE(a.terminal_state, CASE WHEN COALESCE(a.remote_status, '') = '' THEN '' ELSE a.remote_status END) AS latest_async_state,
		       COALESCE(a.remote_job_id, '') AS latest_async_remote_job_id,
		       COALESCE(a.poll_url, '') AS latest_async_poll_url
		FROM exchange_requests r
		LEFT JOIN integration_servers s ON s.id = r.destination_server_id
		LEFT JOIN LATERAL (
			SELECT d.id,
			       d.uid::text AS uid,
			       d.status
			FROM delivery_attempts d
			WHERE d.request_id = r.id
			ORDER BY d.attempt_number DESC, d.created_at DESC
			LIMIT 1
		) ld ON TRUE
		LEFT JOIN async_tasks a ON a.delivery_attempt_id = ld.id
		WHERE r.id = $1
	`)).
		WithArgs(int64(44)).
		WillReturnError(sql.ErrNoRows)

	if _, err := repo.GetRequestByID(context.Background(), 44); err == nil {
		t.Fatal("expected not found error")
	}
}

func TestSQLRepositoryCreateRequest(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer sqlDB.Close()

	repo := NewSQLRepository(sqlx.NewDb(sqlDB, "sqlmock"))
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO exchange_requests (
			uid, source_system, destination_server_id, batch_id, correlation_id, idempotency_key,
			payload_body, payload_format, submission_binding, response_body_persistence, url_suffix, status, status_reason, deferred_until, extras, created_at, updated_at, created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15::jsonb, NOW(), NOW(), $16)
		RETURNING id
	`)).
		WithArgs(
			"11111111-1111-1111-1111-111111111111",
			"emr",
			int64(3),
			"",
			"",
			"",
			`{"trackedEntity":"123"}`,
			"json",
			"body",
			"",
			"/api/data",
			"pending",
			"",
			nil,
			`{"priority":"high"}`,
			int64(9),
		).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(15))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT r.id, r.uid::text AS uid, r.source_system, r.destination_server_id,
		       COALESCE(s.uid::text, '') AS destination_server_uid,
		       COALESCE(s.name, '') AS destination_server_name,
		       COALESCE(s.code, '') AS destination_server_code,
		       COALESCE(r.batch_id, '') AS batch_id, COALESCE(r.correlation_id, '') AS correlation_id, COALESCE(r.idempotency_key, '') AS idempotency_key,
		       r.payload_body, r.payload_format, r.submission_binding, COALESCE(r.response_body_persistence, '') AS response_body_persistence, COALESCE(r.url_suffix, '') AS url_suffix, r.status, COALESCE(r.status_reason, '') AS status_reason, r.deferred_until,
		       r.extras, r.created_at, r.updated_at, r.created_by,
		       ld.id AS latest_delivery_id,
		       COALESCE(ld.uid, '') AS latest_delivery_uid,
		       COALESCE(ld.status, '') AS latest_delivery_status,
		       a.id AS latest_async_task_id,
		       COALESCE(a.uid::text, '') AS latest_async_task_uid,
		       COALESCE(a.terminal_state, CASE WHEN COALESCE(a.remote_status, '') = '' THEN '' ELSE a.remote_status END) AS latest_async_state,
		       COALESCE(a.remote_job_id, '') AS latest_async_remote_job_id,
		       COALESCE(a.poll_url, '') AS latest_async_poll_url
		FROM exchange_requests r
		LEFT JOIN integration_servers s ON s.id = r.destination_server_id
		LEFT JOIN LATERAL (
			SELECT d.id,
			       d.uid::text AS uid,
			       d.status
			FROM delivery_attempts d
			WHERE d.request_id = r.id
			ORDER BY d.attempt_number DESC, d.created_at DESC
			LIMIT 1
		) ld ON TRUE
		LEFT JOIN async_tasks a ON a.delivery_attempt_id = ld.id
		WHERE r.id = $1
	`)).
		WithArgs(int64(15)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "uid", "source_system", "destination_server_id", "destination_server_uid", "destination_server_name", "destination_server_code", "batch_id", "correlation_id",
			"idempotency_key", "payload_body", "payload_format", "submission_binding", "response_body_persistence", "url_suffix", "status", "status_reason", "deferred_until", "extras", "created_at", "updated_at", "created_by",
			"latest_delivery_id", "latest_delivery_uid", "latest_delivery_status", "latest_async_task_id", "latest_async_task_uid", "latest_async_state", "latest_async_remote_job_id", "latest_async_poll_url",
		}).AddRow(
			15, "11111111-1111-1111-1111-111111111111", "emr", 3, "srv-1", "DHIS2 Uganda", "dhis2-ug", "", "",
			"", `{"trackedEntity":"123"}`, "json", "body", "", "/api/data", "pending", "", nil, []byte(`{"priority":"high"}`), now, now, int64(9),
			nil, "", "", nil, "", "", "", "",
		))
	mock.ExpectQuery("(?s)SELECT t.id, t.uid::text AS uid, t.request_id, t.server_id, .*WHERE t.request_id IN \\(\\?\\).*").
		WithArgs(int64(15)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "uid", "request_id", "server_id", "server_uid", "server_name", "server_code", "target_kind", "priority", "status", "blocked_reason", "deferred_until", "last_released_at",
			"latest_delivery_id", "latest_delivery_uid", "latest_delivery_status", "latest_async_task_id", "latest_async_task_uid", "latest_async_state", "latest_async_remote_job_id", "latest_async_poll_url", "created_at", "updated_at",
		}))
	mock.ExpectQuery("(?s)SELECT d.request_id, d.depends_on_request_id, .*WHERE d.request_id IN \\(\\?\\).*").
		WithArgs(int64(15)).
		WillReturnRows(sqlmock.NewRows([]string{
			"request_id", "depends_on_request_id", "request_uid", "depends_on_uid", "status", "status_reason", "deferred_until", "depends_on_destination_server_name",
		}))

	record, err := repo.CreateRequest(context.Background(), CreateParams{
		UID:                 "11111111-1111-1111-1111-111111111111",
		SourceSystem:        "emr",
		DestinationServerID: 3,
		BatchID:             "",
		CorrelationID:       "",
		IdempotencyKey:      "",
		PayloadBody:         `{"trackedEntity":"123"}`,
		PayloadFormat:       "json",
		SubmissionBinding:   "body",
		URLSuffix:           "/api/data",
		Status:              "pending",
		Extras:              map[string]any{"priority": "high"},
		CreatedBy:           int64Ptr(9),
	})
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if record.ID != 15 || record.Status != "pending" {
		t.Fatalf("unexpected record: %+v", record)
	}
	if record.BatchID != "" || record.CorrelationID != "" || record.IdempotencyKey != "" {
		t.Fatalf("expected empty optional metadata strings, got batch=%q correlation=%q idempotency=%q", record.BatchID, record.CorrelationID, record.IdempotencyKey)
	}
}

func int64Ptr(v int64) *int64 {
	return &v
}
