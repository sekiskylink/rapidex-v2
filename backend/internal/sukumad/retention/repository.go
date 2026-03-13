package retention

import (
	"context"
	"database/sql"
	"fmt"
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

func (r *SQLRepository) ListPurgeCandidates(ctx context.Context, cutoff time.Time, limit int) ([]Candidate, error) {
	if limit <= 0 {
		limit = 100
	}
	rows := []Candidate{}
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT r.id AS request_id, r.uid::text AS request_uid, r.status, r.created_at, r.updated_at,
		       COALESCE(COUNT(t.id), 0) AS target_count
		FROM exchange_requests r
		LEFT JOIN request_targets t ON t.request_id = r.id
		WHERE r.updated_at <= $1
		  AND r.status IN ('completed', 'failed')
		  AND NOT EXISTS (
			  SELECT 1
			  FROM request_targets rt
			  WHERE rt.request_id = r.id
			    AND rt.status NOT IN ('succeeded', 'failed')
		  )
		  AND NOT EXISTS (
			  SELECT 1
			  FROM delivery_attempts d
			  JOIN async_tasks a ON a.delivery_attempt_id = d.id
			  WHERE d.request_id = r.id
			    AND COALESCE(a.terminal_state, '') = ''
		  )
		GROUP BY r.id
		ORDER BY r.updated_at ASC, r.id ASC
		LIMIT $2
	`, cutoff, limit); err != nil {
		return nil, fmt.Errorf("list retention candidates: %w", err)
	}
	return rows, nil
}

func (r *SQLRepository) PurgeRequest(ctx context.Context, requestID int64) (PurgeCounts, error) {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return PurgeCounts{}, fmt.Errorf("begin retention transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	counts := PurgeCounts{}
	steps := []struct {
		query string
		dst   *int
		args  []any
	}{
		{
			query: `
				WITH target_async_tasks AS (
					SELECT a.id
					FROM async_tasks a
					JOIN delivery_attempts d ON d.id = a.delivery_attempt_id
					WHERE d.request_id = $1
				)
				DELETE FROM async_task_polls
				WHERE async_task_id IN (SELECT id FROM target_async_tasks)
			`,
			dst:  &counts.AsyncTaskPolls,
			args: []any{requestID},
		},
		{
			query: `
				DELETE FROM async_tasks
				WHERE delivery_attempt_id IN (
					SELECT id FROM delivery_attempts WHERE request_id = $1
				)
			`,
			dst:  &counts.AsyncTasks,
			args: []any{requestID},
		},
		{
			query: `DELETE FROM request_events WHERE request_id = $1`,
			dst:   &counts.RequestEvents,
			args:  []any{requestID},
		},
		{
			query: `DELETE FROM delivery_attempts WHERE request_id = $1`,
			dst:   &counts.DeliveryAttempts,
			args:  []any{requestID},
		},
		{
			query: `DELETE FROM request_targets WHERE request_id = $1`,
			dst:   &counts.RequestTargets,
			args:  []any{requestID},
		},
		{
			query: `DELETE FROM request_dependencies WHERE request_id = $1 OR depends_on_request_id = $1`,
			dst:   &counts.Dependencies,
			args:  []any{requestID},
		},
		{
			query: `DELETE FROM exchange_requests WHERE id = $1`,
			dst:   &counts.Requests,
			args:  []any{requestID},
		},
	}

	for _, step := range steps {
		result, err := tx.ExecContext(ctx, step.query, step.args...)
		if err != nil {
			return PurgeCounts{}, fmt.Errorf("purge request %d: %w", requestID, err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return PurgeCounts{}, fmt.Errorf("read purge rows affected for request %d: %w", requestID, err)
		}
		*step.dst = int(affected)
	}

	if err := tx.Commit(); err != nil {
		return PurgeCounts{}, fmt.Errorf("commit retention transaction: %w", err)
	}
	return counts, nil
}

type memoryRepository struct {
	mu         sync.Mutex
	candidates []Candidate
	purged     map[int64]PurgeCounts
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{purged: map[int64]PurgeCounts{}}
}

func (r *memoryRepository) ListPurgeCandidates(_ context.Context, cutoff time.Time, limit int) ([]Candidate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if limit <= 0 || limit > len(r.candidates) {
		limit = len(r.candidates)
	}
	items := make([]Candidate, 0, limit)
	for _, candidate := range r.candidates {
		if candidate.UpdatedAt.After(cutoff) {
			continue
		}
		items = append(items, candidate)
		if len(items) == limit {
			break
		}
	}
	return items, nil
}

func (r *memoryRepository) PurgeRequest(_ context.Context, requestID int64) (PurgeCounts, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	counts, ok := r.purged[requestID]
	if !ok {
		counts = PurgeCounts{Requests: 1}
	}
	return counts, nil
}
