package devseed

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CleanupSeedData(ctx context.Context, seedTag string, serverCodePrefix string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin cleanup transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	requestIDs := []int64{}
	if err := tx.SelectContext(ctx, &requestIDs, `
		SELECT id
		FROM exchange_requests
		WHERE extras->>'seedTag' = $1
	`, seedTag); err != nil {
		return fmt.Errorf("select seeded requests: %w", err)
	}

	if len(requestIDs) > 0 {
		query, args, err := sqlx.In(`
			DELETE FROM request_dependencies
			WHERE request_id IN (?) OR depends_on_request_id IN (?)
		`, requestIDs, requestIDs)
		if err != nil {
			return fmt.Errorf("build dependency cleanup query: %w", err)
		}
		query = tx.Rebind(query)
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("delete seeded request dependencies: %w", err)
		}

		query, args, err = sqlx.In(`DELETE FROM exchange_requests WHERE id IN (?)`, requestIDs)
		if err != nil {
			return fmt.Errorf("build request cleanup query: %w", err)
		}
		query = tx.Rebind(query)
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("delete seeded requests: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM integration_servers
		WHERE code LIKE $1
	`, serverCodePrefix+"%"); err != nil {
		return fmt.Errorf("delete seeded servers: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit cleanup transaction: %w", err)
	}
	return nil
}
