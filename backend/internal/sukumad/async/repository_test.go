package async

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestSQLRepositoryListTasks(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()

	repo := NewSQLRepository(sqlx.NewDb(sqlDB, "sqlmock"))
	now := time.Now().UTC()
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	dataRows := sqlmock.NewRows([]string{
		"id", "uid", "delivery_attempt_id", "delivery_uid", "request_id", "request_uid", "correlation_id",
		"remote_job_id", "poll_url", "remote_status", "terminal_state", "next_poll_at", "completed_at", "remote_response", "created_at", "updated_at",
	}).AddRow(
		7, "job-uid", 3, "delivery-uid", 5, "request-uid", "corr-1",
		"remote-7", "https://remote/jobs/7", StatePolling, "", now, nil, []byte(`{"status":"processing"}`), now, now,
	)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) 
		FROM async_tasks a
		LEFT JOIN delivery_attempts d ON d.id = a.delivery_attempt_id
		LEFT JOIN exchange_requests rq ON rq.id = d.request_id
		 WHERE (
			a.uid::text ILIKE $1 OR
			COALESCE(d.uid::text, '') ILIKE $1 OR
			COALESCE(rq.uid::text, '') ILIKE $1 OR
			COALESCE(a.remote_job_id, '') ILIKE $1 OR
			COALESCE(a.remote_status, '') ILIKE $1 OR
			COALESCE(a.terminal_state, '') ILIKE $1
		) AND (COALESCE(a.terminal_state, CASE WHEN COALESCE(a.remote_status, '') = '' THEN 'pending' ELSE a.remote_status END) = $2)`)).
		WithArgs("%remote%", StatePolling).
		WillReturnRows(countRows)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT a.id, a.uid::text AS uid, a.delivery_attempt_id, COALESCE(d.uid::text, '') AS delivery_uid,
		       COALESCE(d.request_id, 0) AS request_id, COALESCE(rq.uid::text, '') AS request_uid, COALESCE(rq.correlation_id, '') AS correlation_id,
		       COALESCE(a.remote_job_id, '') AS remote_job_id, COALESCE(a.poll_url, '') AS poll_url,
		       COALESCE(a.remote_status, '') AS remote_status, COALESCE(a.terminal_state, '') AS terminal_state,
		       a.next_poll_at, a.completed_at, a.remote_response, a.created_at, a.updated_at
		
		FROM async_tasks a
		LEFT JOIN delivery_attempts d ON d.id = a.delivery_attempt_id
		LEFT JOIN exchange_requests rq ON rq.id = d.request_id
		 WHERE (
			a.uid::text ILIKE $1 OR
			COALESCE(d.uid::text, '') ILIKE $1 OR
			COALESCE(rq.uid::text, '') ILIKE $1 OR
			COALESCE(a.remote_job_id, '') ILIKE $1 OR
			COALESCE(a.remote_status, '') ILIKE $1 OR
			COALESCE(a.terminal_state, '') ILIKE $1
		) AND (COALESCE(a.terminal_state, CASE WHEN COALESCE(a.remote_status, '') = '' THEN 'pending' ELSE a.remote_status END) = $2) ORDER BY a.created_at DESC LIMIT $3 OFFSET $4`)).
		WithArgs("%remote%", StatePolling, 25, 0).
		WillReturnRows(dataRows)

	result, err := repo.ListTasks(context.Background(), ListQuery{
		Page:      1,
		PageSize:  25,
		Filter:    "remote",
		Status:    StatePolling,
		SortField: "createdAt",
		SortOrder: "desc",
	})
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("unexpected list result: %+v", result)
	}
	if result.Items[0].DeliveryUID != "delivery-uid" || result.Items[0].RequestUID != "request-uid" {
		t.Fatalf("unexpected task row: %+v", result.Items[0])
	}
}

func TestSQLRepositoryCreateUpdateAndRecordPoll(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()

	repo := NewSQLRepository(sqlx.NewDb(sqlDB, "sqlmock"))
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO async_tasks (
			uid, delivery_attempt_id, remote_job_id, poll_url, remote_status, terminal_state,
			next_poll_at, completed_at, remote_response, created_at, updated_at
		)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), $7, $8, $9::jsonb, NOW(), NOW())
		RETURNING id
	`)).
		WithArgs("job-uid", int64(3), "remote-3", "https://remote/jobs/3", StatePending, "", nil, nil, `{"state":"pending"}`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(9))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT a.id, a.uid::text AS uid, a.delivery_attempt_id, COALESCE(d.uid::text, '') AS delivery_uid,
		       COALESCE(d.request_id, 0) AS request_id, COALESCE(rq.uid::text, '') AS request_uid, COALESCE(rq.correlation_id, '') AS correlation_id,
		       COALESCE(a.remote_job_id, '') AS remote_job_id, COALESCE(a.poll_url, '') AS poll_url,
		       COALESCE(a.remote_status, '') AS remote_status, COALESCE(a.terminal_state, '') AS terminal_state,
		       a.next_poll_at, a.completed_at, a.remote_response, a.created_at, a.updated_at
		FROM async_tasks a
		LEFT JOIN delivery_attempts d ON d.id = a.delivery_attempt_id
		LEFT JOIN exchange_requests rq ON rq.id = d.request_id
		WHERE a.id = $1
	`)).
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "uid", "delivery_attempt_id", "delivery_uid", "request_id", "request_uid", "correlation_id",
			"remote_job_id", "poll_url", "remote_status", "terminal_state", "next_poll_at", "completed_at", "remote_response", "created_at", "updated_at",
		}).AddRow(9, "job-uid", 3, "delivery-uid", 5, "request-uid", "corr-1", "remote-3", "https://remote/jobs/3", StatePending, "", nil, nil, []byte(`{"state":"pending"}`), now, now))

	record, err := repo.CreateTask(context.Background(), CreateParams{
		UID:               "job-uid",
		DeliveryAttemptID: 3,
		RemoteJobID:       "remote-3",
		PollURL:           "https://remote/jobs/3",
		RemoteStatus:      StatePending,
		RemoteResponse: map[string]any{
			"state": "pending",
		},
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if record.ID != 9 || record.RemoteJobID != "remote-3" {
		t.Fatalf("unexpected created task: %+v", record)
	}

	completedAt := now
	mock.ExpectQuery(regexp.QuoteMeta(`
		UPDATE async_tasks
		SET remote_job_id = NULLIF($2, ''),
		    poll_url = NULLIF($3, ''),
		    remote_status = NULLIF($4, ''),
		    terminal_state = NULLIF($5, ''),
		    next_poll_at = $6,
		    completed_at = $7,
		    remote_response = $8::jsonb,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`)).
		WithArgs(int64(9), "remote-3", "https://remote/jobs/3", StateSucceeded, StateSucceeded, nil, &completedAt, `{"state":"done"}`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(9))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT a.id, a.uid::text AS uid, a.delivery_attempt_id, COALESCE(d.uid::text, '') AS delivery_uid,
		       COALESCE(d.request_id, 0) AS request_id, COALESCE(rq.uid::text, '') AS request_uid, COALESCE(rq.correlation_id, '') AS correlation_id,
		       COALESCE(a.remote_job_id, '') AS remote_job_id, COALESCE(a.poll_url, '') AS poll_url,
		       COALESCE(a.remote_status, '') AS remote_status, COALESCE(a.terminal_state, '') AS terminal_state,
		       a.next_poll_at, a.completed_at, a.remote_response, a.created_at, a.updated_at
		FROM async_tasks a
		LEFT JOIN delivery_attempts d ON d.id = a.delivery_attempt_id
		LEFT JOIN exchange_requests rq ON rq.id = d.request_id
		WHERE a.id = $1
	`)).
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "uid", "delivery_attempt_id", "delivery_uid", "request_id", "request_uid", "correlation_id",
			"remote_job_id", "poll_url", "remote_status", "terminal_state", "next_poll_at", "completed_at", "remote_response", "created_at", "updated_at",
		}).AddRow(9, "job-uid", 3, "delivery-uid", 5, "request-uid", "corr-1", "remote-3", "https://remote/jobs/3", StateSucceeded, StateSucceeded, nil, now, []byte(`{"state":"done"}`), now, now))

	updated, err := repo.UpdateTask(context.Background(), UpdateParams{
		ID:             9,
		RemoteJobID:    "remote-3",
		PollURL:        "https://remote/jobs/3",
		RemoteStatus:   StateSucceeded,
		TerminalState:  StateSucceeded,
		CompletedAt:    &completedAt,
		RemoteResponse: map[string]any{"state": "done"},
	})
	if err != nil {
		t.Fatalf("update task: %v", err)
	}
	if updated.TerminalState != StateSucceeded {
		t.Fatalf("unexpected updated task: %+v", updated)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO async_task_polls (
			async_task_id, polled_at, status_code, remote_status, response_body, error_message, duration_ms
		)
		VALUES ($1, NOW(), $2, NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''), $6)
		RETURNING id, async_task_id, polled_at, status_code, COALESCE(remote_status, '') AS remote_status,
		          COALESCE(response_body, '') AS response_body, COALESCE(error_message, '') AS error_message, duration_ms
	`)).
		WithArgs(int64(9), intPtr(200), StateSucceeded, `{"done":true}`, "", intPtr(99)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "async_task_id", "polled_at", "status_code", "remote_status", "response_body", "error_message", "duration_ms"}).
			AddRow(int64(1), int64(9), now, 200, StateSucceeded, `{"done":true}`, "", 99))

	poll, err := repo.RecordPoll(context.Background(), RecordPollInput{
		AsyncTaskID:  9,
		StatusCode:   intPtr(200),
		RemoteStatus: StateSucceeded,
		ResponseBody: `{"done":true}`,
		DurationMS:   intPtr(99),
	})
	if err != nil {
		t.Fatalf("record poll: %v", err)
	}
	if poll.AsyncTaskID != 9 {
		t.Fatalf("unexpected poll record: %+v", poll)
	}
}

func TestSQLRepositoryGetTaskNotFound(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()

	repo := NewSQLRepository(sqlx.NewDb(sqlDB, "sqlmock"))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT a.id, a.uid::text AS uid, a.delivery_attempt_id, COALESCE(d.uid::text, '') AS delivery_uid,
		       COALESCE(d.request_id, 0) AS request_id, COALESCE(rq.uid::text, '') AS request_uid, COALESCE(rq.correlation_id, '') AS correlation_id,
		       COALESCE(a.remote_job_id, '') AS remote_job_id, COALESCE(a.poll_url, '') AS poll_url,
		       COALESCE(a.remote_status, '') AS remote_status, COALESCE(a.terminal_state, '') AS terminal_state,
		       a.next_poll_at, a.completed_at, a.remote_response, a.created_at, a.updated_at
		FROM async_tasks a
		LEFT JOIN delivery_attempts d ON d.id = a.delivery_attempt_id
		LEFT JOIN exchange_requests rq ON rq.id = d.request_id
		WHERE a.id = $1
	`)).
		WithArgs(int64(99)).
		WillReturnError(sql.ErrNoRows)

	if _, err := repo.GetTaskByID(context.Background(), 99); err == nil {
		t.Fatal("expected not found error")
	}
}
