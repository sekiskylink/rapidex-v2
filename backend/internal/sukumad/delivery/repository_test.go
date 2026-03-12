package delivery

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestSQLRepositoryListDeliveries(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer sqlDB.Close()

	repo := NewSQLRepository(sqlx.NewDb(sqlDB, "sqlmock"))
	now := time.Now().UTC()
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	dataRows := sqlmock.NewRows([]string{
		"id", "uid", "request_id", "request_uid", "server_id", "server_name", "attempt_number",
		"status", "http_status", "response_body", "error_message", "started_at", "finished_at", "retry_at", "created_at", "updated_at",
	}).AddRow(
		7, "delivery-uid", 3, "request-uid", 9, "DHIS2 Uganda", 1,
		StatusFailed, 502, "{}", "timeout", now, now, nil, now, now,
	)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) 
		FROM delivery_attempts d
		LEFT JOIN exchange_requests r ON r.id = d.request_id
		LEFT JOIN integration_servers s ON s.id = d.server_id
		 WHERE (
			d.uid::text ILIKE $1 OR
			COALESCE(r.uid::text, '') ILIKE $1 OR
			COALESCE(s.name, '') ILIKE $1 OR
			COALESCE(s.code, '') ILIKE $1 OR
			COALESCE(d.error_message, '') ILIKE $1
		) AND d.status = $2 AND (COALESCE(s.name, '') ILIKE $3 OR COALESCE(s.code, '') ILIKE $3) AND DATE(d.created_at) = $4::date`)).
		WithArgs("%dhis%", StatusFailed, "%dhis%", now.Format("2006-01-02")).
		WillReturnRows(countRows)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT d.id, d.uid::text AS uid, d.request_id, COALESCE(r.uid::text, '') AS request_uid,
		       d.server_id, COALESCE(s.name, '') AS server_name,
		       d.attempt_number, d.status, d.http_status, d.response_body, d.error_message,
		       d.started_at, d.finished_at, d.retry_at, d.created_at, d.updated_at
		
		FROM delivery_attempts d
		LEFT JOIN exchange_requests r ON r.id = d.request_id
		LEFT JOIN integration_servers s ON s.id = d.server_id
		 WHERE (
			d.uid::text ILIKE $1 OR
			COALESCE(r.uid::text, '') ILIKE $1 OR
			COALESCE(s.name, '') ILIKE $1 OR
			COALESCE(s.code, '') ILIKE $1 OR
			COALESCE(d.error_message, '') ILIKE $1
		) AND d.status = $2 AND (COALESCE(s.name, '') ILIKE $3 OR COALESCE(s.code, '') ILIKE $3) AND DATE(d.created_at) = $4::date ORDER BY d.created_at DESC LIMIT $5 OFFSET $6`)).
		WithArgs("%dhis%", StatusFailed, "%dhis%", now.Format("2006-01-02"), 25, 0).
		WillReturnRows(dataRows)

	result, err := repo.ListDeliveries(context.Background(), ListQuery{
		Page:      1,
		PageSize:  25,
		Filter:    "dhis",
		Status:    StatusFailed,
		Server:    "dhis",
		Date:      &now,
		SortField: "createdAt",
		SortOrder: "desc",
	})
	if err != nil {
		t.Fatalf("list deliveries: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("expected one delivery, got total=%d items=%d", result.Total, len(result.Items))
	}
	if result.Items[0].RequestUID != "request-uid" || result.Items[0].ServerName != "DHIS2 Uganda" {
		t.Fatalf("unexpected record: %+v", result.Items[0])
	}
}

func TestSQLRepositoryGetDeliveryByIDNotFound(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer sqlDB.Close()

	repo := NewSQLRepository(sqlx.NewDb(sqlDB, "sqlmock"))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT d.id, d.uid::text AS uid, d.request_id, COALESCE(r.uid::text, '') AS request_uid,
		       d.server_id, COALESCE(s.name, '') AS server_name,
		       d.attempt_number, d.status, d.http_status, d.response_body, d.error_message,
		       d.started_at, d.finished_at, d.retry_at, d.created_at, d.updated_at
		FROM delivery_attempts d
		LEFT JOIN exchange_requests r ON r.id = d.request_id
		LEFT JOIN integration_servers s ON s.id = d.server_id
		WHERE d.id = $1
	`)).
		WithArgs(int64(41)).
		WillReturnError(sql.ErrNoRows)

	if _, err := repo.GetDeliveryByID(context.Background(), 41); err == nil {
		t.Fatal("expected not found error")
	}
}

func TestSQLRepositoryCreateAndUpdateDelivery(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer sqlDB.Close()

	repo := NewSQLRepository(sqlx.NewDb(sqlDB, "sqlmock"))
	now := time.Now().UTC()
	started := now.Add(-time.Minute)
	finished := now
	httpStatus := 200

	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO delivery_attempts (
			uid, request_id, server_id, attempt_number, status, http_status, response_body, error_message,
			started_at, finished_at, retry_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		RETURNING id
	`)).
		WithArgs("delivery-uid", int64(4), int64(8), 1, StatusPending, nil, "", "", nil, nil, nil).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT d.id, d.uid::text AS uid, d.request_id, COALESCE(r.uid::text, '') AS request_uid,
		       d.server_id, COALESCE(s.name, '') AS server_name,
		       d.attempt_number, d.status, d.http_status, d.response_body, d.error_message,
		       d.started_at, d.finished_at, d.retry_at, d.created_at, d.updated_at
		FROM delivery_attempts d
		LEFT JOIN exchange_requests r ON r.id = d.request_id
		LEFT JOIN integration_servers s ON s.id = d.server_id
		WHERE d.id = $1
	`)).
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "uid", "request_id", "request_uid", "server_id", "server_name", "attempt_number",
			"status", "http_status", "response_body", "error_message", "started_at", "finished_at", "retry_at", "created_at", "updated_at",
		}).AddRow(
			10, "delivery-uid", 4, "request-uid", 8, "DHIS2 Uganda", 1,
			StatusPending, nil, "", "", nil, nil, nil, now, now,
		))

	record, err := repo.CreateDelivery(context.Background(), CreateParams{
		UID:           "delivery-uid",
		RequestID:     4,
		ServerID:      8,
		AttemptNumber: 1,
		Status:        StatusPending,
		ResponseBody:  "",
		ErrorMessage:  "",
	})
	if err != nil {
		t.Fatalf("create delivery: %v", err)
	}
	if record.ID != 10 || record.Status != StatusPending {
		t.Fatalf("unexpected created record: %+v", record)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`
		UPDATE delivery_attempts
		SET status = $2,
		    http_status = $3,
		    response_body = $4,
		    error_message = $5,
		    started_at = $6,
		    finished_at = $7,
		    retry_at = $8,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`)).
		WithArgs(int64(10), StatusSucceeded, &httpStatus, `{"status":"ok"}`, "", &started, &finished, nil).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT d.id, d.uid::text AS uid, d.request_id, COALESCE(r.uid::text, '') AS request_uid,
		       d.server_id, COALESCE(s.name, '') AS server_name,
		       d.attempt_number, d.status, d.http_status, d.response_body, d.error_message,
		       d.started_at, d.finished_at, d.retry_at, d.created_at, d.updated_at
		FROM delivery_attempts d
		LEFT JOIN exchange_requests r ON r.id = d.request_id
		LEFT JOIN integration_servers s ON s.id = d.server_id
		WHERE d.id = $1
	`)).
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "uid", "request_id", "request_uid", "server_id", "server_name", "attempt_number",
			"status", "http_status", "response_body", "error_message", "started_at", "finished_at", "retry_at", "created_at", "updated_at",
		}).AddRow(
			10, "delivery-uid", 4, "request-uid", 8, "DHIS2 Uganda", 1,
			StatusSucceeded, httpStatus, `{"status":"ok"}`, "", started, finished, nil, now, now,
		))

	updated, err := repo.UpdateDelivery(context.Background(), UpdateParams{
		ID:           10,
		Status:       StatusSucceeded,
		HTTPStatus:   &httpStatus,
		ResponseBody: `{"status":"ok"}`,
		ErrorMessage: "",
		StartedAt:    &started,
		FinishedAt:   &finished,
	})
	if err != nil {
		t.Fatalf("update delivery: %v", err)
	}
	if updated.Status != StatusSucceeded || updated.HTTPStatus == nil || *updated.HTTPStatus != 200 {
		t.Fatalf("unexpected updated record: %+v", updated)
	}
}
