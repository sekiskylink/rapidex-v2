package server

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestSQLRepositoryListServers(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer sqlDB.Close()

	repo := NewSQLRepository(sqlx.NewDb(sqlDB, "sqlmock"))
	now := time.Now().UTC()
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	dataRows := sqlmock.NewRows([]string{
		"id", "uid", "name", "code", "system_type", "base_url", "endpoint_type", "http_method",
		"use_async", "parse_responses", "response_body_persistence", "headers", "url_params", "suspended", "created_at", "updated_at", "created_by",
	}).AddRow(
		1, "11111111-1111-1111-1111-111111111111", "DHIS2 Uganda", "dhis2-ug", "dhis2", "https://dhis.example.com", "http", "POST",
		true, true, "filter", []byte(`{"Authorization":"Bearer token"}`), []byte(`{"orgUnit":"OU_123"}`), false, now, now, int64(7),
	)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM integration_servers WHERE name ILIKE $1 OR code ILIKE $1 OR system_type ILIKE $1 OR base_url ILIKE $1`)).
		WithArgs("%dhis%").
		WillReturnRows(countRows)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, uid::text AS uid, name, code, system_type, base_url, endpoint_type, http_method,
		       use_async, parse_responses, response_body_persistence, headers, url_params, suspended, created_at, updated_at, created_by
		FROM integration_servers
	 WHERE name ILIKE $1 OR code ILIKE $1 OR system_type ILIKE $1 OR base_url ILIKE $1 ORDER BY name ASC LIMIT $2 OFFSET $3`)).
		WithArgs("%dhis%", 25, 0).
		WillReturnRows(dataRows)

	result, err := repo.ListServers(context.Background(), ListQuery{Page: 1, PageSize: 25, SortField: "name", SortOrder: "asc", Filter: "dhis"})
	if err != nil {
		t.Fatalf("list servers: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("expected one server, got total=%d items=%d", result.Total, len(result.Items))
	}
	if result.Items[0].Code != "dhis2-ug" {
		t.Fatalf("expected server code dhis2-ug, got %q", result.Items[0].Code)
	}
	if result.Items[0].Headers["Authorization"] != "Bearer token" {
		t.Fatalf("expected decoded headers")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSQLRepositoryGetServerByIDNotFound(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer sqlDB.Close()

	repo := NewSQLRepository(sqlx.NewDb(sqlDB, "sqlmock"))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, uid::text AS uid, name, code, system_type, base_url, endpoint_type, http_method,
		       use_async, parse_responses, response_body_persistence, headers, url_params, suspended, created_at, updated_at, created_by
		FROM integration_servers
		WHERE id = $1
	`)).
		WithArgs(int64(44)).
		WillReturnError(sql.ErrNoRows)

	if _, err := repo.GetServerByID(context.Background(), 44); err == nil {
		t.Fatal("expected not found error")
	}
}

func TestSQLRepositoryCreateServer(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer sqlDB.Close()

	repo := NewSQLRepository(sqlx.NewDb(sqlDB, "sqlmock"))
	now := time.Now().UTC()
	rows := sqlmock.NewRows([]string{
		"id", "uid", "name", "code", "system_type", "base_url", "endpoint_type", "http_method",
		"use_async", "parse_responses", "response_body_persistence", "headers", "url_params", "suspended", "created_at", "updated_at", "created_by",
	}).AddRow(
		9, "11111111-1111-1111-1111-111111111111", "OpenHIM", "openhim", "api", "https://openhim.example.com", "http", "POST",
		false, true, "save", []byte(`{"X-Api-Key":"abc"}`), []byte(`{}`), false, now, now, int64(3),
	)

	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO integration_servers (
			uid, name, code, system_type, base_url, endpoint_type, http_method,
			use_async, parse_responses, response_body_persistence, headers, url_params, suspended, created_at, updated_at, created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb, $12::jsonb, $13, NOW(), NOW(), $14)
		RETURNING id, uid::text AS uid, name, code, system_type, base_url, endpoint_type, http_method,
		          use_async, parse_responses, response_body_persistence, headers, url_params, suspended, created_at, updated_at, created_by
	`)).
		WithArgs("11111111-1111-1111-1111-111111111111", "OpenHIM", "openhim", "api", "https://openhim.example.com", "http", "POST", false, true, "save", `{"X-Api-Key":"abc"}`, `{}`, false, int64(3)).
		WillReturnRows(rows)

	record, err := repo.CreateServer(context.Background(), CreateParams{
		UID:                     "11111111-1111-1111-1111-111111111111",
		Name:                    "OpenHIM",
		Code:                    "openhim",
		SystemType:              "api",
		BaseURL:                 "https://openhim.example.com",
		EndpointType:            "http",
		HTTPMethod:              "POST",
		UseAsync:                false,
		ParseResponses:          true,
		ResponseBodyPersistence: "save",
		Headers:                 map[string]string{"X-Api-Key": "abc"},
		URLParams:               map[string]string{},
		Suspended:               false,
		CreatedBy:               int64Ptr(3),
	})
	if err != nil {
		t.Fatalf("create server: %v", err)
	}
	if record.ID != 9 || record.Code != "openhim" {
		t.Fatalf("unexpected record: %+v", record)
	}
}

func TestSQLRepositoryDeleteServerNotFound(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer sqlDB.Close()

	repo := NewSQLRepository(sqlx.NewDb(sqlDB, "sqlmock"))
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM integration_servers WHERE id = $1`)).
		WithArgs(int64(55)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := repo.DeleteServer(context.Background(), 55); err == nil {
		t.Fatal("expected delete not found error")
	}
}

func int64Ptr(v int64) *int64 {
	return &v
}
