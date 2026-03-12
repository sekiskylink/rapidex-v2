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
		"id", "uid", "source_system", "destination_server_id", "destination_server_name", "batch_id", "correlation_id",
		"idempotency_key", "payload_body", "payload_format", "url_suffix", "status", "extras", "created_at", "updated_at", "created_by",
	}).AddRow(
		8, "11111111-1111-1111-1111-111111111111", "emr", 3, "DHIS2 Uganda", "batch-1", "corr-1",
		"idem-1", `{"trackedEntity":"123"}`, "json", "/api/data", "pending", []byte(`{"priority":"high"}`), now, now, int64(7),
	)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) 
		FROM exchange_requests r
		LEFT JOIN integration_servers s ON s.id = r.destination_server_id
		 WHERE (
			r.uid ILIKE $1 OR
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
		SELECT r.id, r.uid, r.source_system, r.destination_server_id,
		       COALESCE(s.name, '') AS destination_server_name,
		       r.batch_id, r.correlation_id, r.idempotency_key,
		       r.payload_body, r.payload_format, r.url_suffix, r.status,
		       r.extras, r.created_at, r.updated_at, r.created_by
		FROM exchange_requests r
		LEFT JOIN integration_servers s ON s.id = r.destination_server_id
		 WHERE (
			r.uid ILIKE $1 OR
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
	if string(result.Items[0].Payload) != `{"trackedEntity":"123"}` {
		t.Fatalf("expected payload raw json, got %s", string(result.Items[0].Payload))
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
		SELECT r.id, r.uid, r.source_system, r.destination_server_id,
		       COALESCE(s.name, '') AS destination_server_name,
		       r.batch_id, r.correlation_id, r.idempotency_key,
		       r.payload_body, r.payload_format, r.url_suffix, r.status,
		       r.extras, r.created_at, r.updated_at, r.created_by
		FROM exchange_requests r
		LEFT JOIN integration_servers s ON s.id = r.destination_server_id
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
			payload_body, payload_format, url_suffix, status, extras, created_at, updated_at, created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb, NOW(), NOW(), $12)
		RETURNING id
	`)).
		WithArgs(
			"11111111-1111-1111-1111-111111111111",
			"emr",
			int64(3),
			"batch-1",
			"corr-1",
			"idem-1",
			`{"trackedEntity":"123"}`,
			"json",
			"/api/data",
			"pending",
			`{"priority":"high"}`,
			int64(9),
		).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(15))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT r.id, r.uid, r.source_system, r.destination_server_id,
		       COALESCE(s.name, '') AS destination_server_name,
		       r.batch_id, r.correlation_id, r.idempotency_key,
		       r.payload_body, r.payload_format, r.url_suffix, r.status,
		       r.extras, r.created_at, r.updated_at, r.created_by
		FROM exchange_requests r
		LEFT JOIN integration_servers s ON s.id = r.destination_server_id
		WHERE r.id = $1
	`)).
		WithArgs(int64(15)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "uid", "source_system", "destination_server_id", "destination_server_name", "batch_id", "correlation_id",
			"idempotency_key", "payload_body", "payload_format", "url_suffix", "status", "extras", "created_at", "updated_at", "created_by",
		}).AddRow(
			15, "11111111-1111-1111-1111-111111111111", "emr", 3, "DHIS2 Uganda", "batch-1", "corr-1",
			"idem-1", `{"trackedEntity":"123"}`, "json", "/api/data", "pending", []byte(`{"priority":"high"}`), now, now, int64(9),
		))

	record, err := repo.CreateRequest(context.Background(), CreateParams{
		UID:                 "11111111-1111-1111-1111-111111111111",
		SourceSystem:        "emr",
		DestinationServerID: 3,
		BatchID:             "batch-1",
		CorrelationID:       "corr-1",
		IdempotencyKey:      "idem-1",
		PayloadBody:         `{"trackedEntity":"123"}`,
		PayloadFormat:       "json",
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
}

func int64Ptr(v int64) *int64 {
	return &v
}
