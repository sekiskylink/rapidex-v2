package userorg

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

// PgRepository implements Repository using PostgreSQL with sqlx.  It operates
// on a table `user_org_units` that has columns: user_id, org_unit_id,
// created_at, updated_at.  A unique constraint on (user_id, org_unit_id)
// prevents duplicate assignments.
type PgRepository struct {
	db *sqlx.DB
}

// NewPgRepository constructs a new PgRepository.
func NewPgRepository(db *sqlx.DB) *PgRepository {
	return &PgRepository{db: db}
}

// ListByUser returns all org unit IDs assigned to the specified user.
func (r *PgRepository) ListByUser(ctx context.Context, userID int64) ([]int64, error) {
	var ids []int64
	err := r.db.SelectContext(ctx, &ids, "SELECT org_unit_id FROM user_org_units WHERE user_id = $1", userID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return ids, nil
}

// ListAssignmentsByUser returns the assigned org units with lightweight
// hierarchy context for admin UIs and scope resolution.
func (r *PgRepository) ListAssignmentsByUser(ctx context.Context, userID int64) ([]AssignmentDetail, error) {
	details := []AssignmentDetail{}
	err := r.db.SelectContext(ctx, &details, `
        SELECT
            uou.org_unit_id,
            ou.name AS org_unit_name,
            ou.path AS org_unit_path,
            COALESCE(
                (
                    SELECT string_agg(ancestor.name, ' / ' ORDER BY path_parts.ordinality)
                    FROM unnest(string_to_array(trim(ou.path, '/'), '/')) WITH ORDINALITY AS path_parts(uid, ordinality)
                    JOIN org_units ancestor ON ancestor.uid = path_parts.uid
                ),
                ou.name
            ) AS display_path
        FROM user_org_units uou
        JOIN org_units ou ON ou.id = uou.org_unit_id
        WHERE uou.user_id = $1
        ORDER BY ou.name ASC, uou.org_unit_id ASC
    `, userID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return details, nil
}

// Assign inserts or updates an assignment.  It is idempotent: if the row
// already exists, only updated_at is refreshed.
func (r *PgRepository) Assign(ctx context.Context, userID int64, orgUnitID int64) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `INSERT INTO user_org_units (user_id, org_unit_id, created_at, updated_at)
VALUES ($1, $2, $3, $3)
ON CONFLICT (user_id, org_unit_id) DO UPDATE SET updated_at = EXCLUDED.updated_at`, userID, orgUnitID, now)
	return err
}

// Remove deletes a user→org unit assignment.
func (r *PgRepository) Remove(ctx context.Context, userID int64, orgUnitID int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM user_org_units WHERE user_id = $1 AND org_unit_id = $2", userID, orgUnitID)
	return err
}
