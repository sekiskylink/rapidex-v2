package reporter

import (
    "context"
    "database/sql"
    "fmt"

    "github.com/jmoiron/sqlx"
)

// PgRepository implements the Reporter repository backed by a PostgreSQL database.  It
// uses sqlx for convenient struct scanning and parameter binding.  This
// implementation is intentionally simple; it does not optimise for performance or
// transaction management but provides a complete set of CRUD operations.  It
// should be wrapped in higher‑level services or middleware to enforce
// permissions and logging.
//
// NOTE: This repository assumes a table `reporters` with columns matching the
// Reporter struct tags defined in types.go.  It also expects unique
// constraints on contact_uuid and phone_number.  Migrations must be applied
// before using this repository.
type PgRepository struct {
    db *sqlx.DB
}

// NewPgRepository constructs a new PgRepository.  The provided db must be a
// live connection.  The caller is responsible for managing the lifecycle of
// the connection.
func NewPgRepository(db *sqlx.DB) *PgRepository {
    return &PgRepository{db: db}
}

// List returns a paginated list of reporters.  Filters on search and
// organisation unit are applied when provided.  Search terms match against
// phone numbers and display names case‑insensitively.
func (r *PgRepository) List(ctx context.Context, query ListQuery) (ListResult, error) {
    result := ListResult{Page: query.Page, PageSize: query.PageSize}
    // Build base query and parameters
    where := "WHERE 1=1"
    args := []interface{}{}
    if query.Search != "" {
        where += " AND (LOWER(phone_number) LIKE LOWER(?) OR LOWER(display_name) LIKE LOWER(?))"
        s := fmt.Sprintf("%%%s%%", query.Search)
        args = append(args, s, s)
    }
    if query.OrgUnitID != nil {
        where += " AND org_unit_id = ?"
        args = append(args, *query.OrgUnitID)
    }
    if query.OnlyActive {
        where += " AND is_active = TRUE"
    }
    // Count total
    countQuery := fmt.Sprintf("SELECT COUNT(*) FROM reporters %s", where)
    var total int
    if err := r.db.GetContext(ctx, &total, r.db.Rebind(countQuery), args...); err != nil {
        return result, err
    }
    result.Total = total
    // Pagination
    limit := query.PageSize
    if limit <= 0 {
        limit = 20
    }
    offset := query.Page * limit
    listQuery := fmt.Sprintf("SELECT id, uid, contact_uuid, phone_number, display_name, org_unit_id, is_active, created_at, updated_at FROM reporters %s ORDER BY id LIMIT %d OFFSET %d", where, limit, offset)
    rows := []Reporter{}
    if err := r.db.SelectContext(ctx, &rows, r.db.Rebind(listQuery), args...); err != nil {
        return result, err
    }
    result.Items = rows
    return result, nil
}

// GetByID fetches a reporter by its numeric ID.
func (r *PgRepository) GetByID(ctx context.Context, id int64) (Reporter, error) {
    var rp Reporter
    err := r.db.GetContext(ctx, &rp, "SELECT id, uid, contact_uuid, phone_number, display_name, org_unit_id, is_active, created_at, updated_at FROM reporters WHERE id = $1", id)
    return rp, err
}

// GetByUID fetches a reporter by its UID.
func (r *PgRepository) GetByUID(ctx context.Context, uid string) (Reporter, error) {
    var rp Reporter
    err := r.db.GetContext(ctx, &rp, "SELECT id, uid, contact_uuid, phone_number, display_name, org_unit_id, is_active, created_at, updated_at FROM reporters WHERE uid = $1", uid)
    return rp, err
}

// GetByContactUUID fetches a reporter by its RapidPro contact UUID.
func (r *PgRepository) GetByContactUUID(ctx context.Context, contactUUID string) (Reporter, error) {
    var rp Reporter
    err := r.db.GetContext(ctx, &rp, "SELECT id, uid, contact_uuid, phone_number, display_name, org_unit_id, is_active, created_at, updated_at FROM reporters WHERE contact_uuid = $1", contactUUID)
    return rp, err
}

// GetByPhoneNumber fetches a reporter by its phone number.
func (r *PgRepository) GetByPhoneNumber(ctx context.Context, phone string) (Reporter, error) {
    var rp Reporter
    err := r.db.GetContext(ctx, &rp, "SELECT id, uid, contact_uuid, phone_number, display_name, org_unit_id, is_active, created_at, updated_at FROM reporters WHERE phone_number = $1", phone)
    return rp, err
}

// Create inserts a new reporter and returns the persisted record.  It generates
// a new UID when none is provided using a simple UUID function in the
// database.  Unique constraint violations are returned as errors.
func (r *PgRepository) Create(ctx context.Context, reporter Reporter) (Reporter, error) {
    // Insert and return ID + UID
    query := `INSERT INTO reporters (uid, contact_uuid, phone_number, display_name, org_unit_id, is_active, created_at, updated_at)
VALUES (COALESCE(NULLIF($1, ''), gen_random_uuid()::text), $2, $3, $4, $5, $6, $7, $8)
RETURNING id, uid`
    var id int64
    var uid string
    err := r.db.QueryRowxContext(ctx, query,
        reporter.UID,
        reporter.ContactUUID,
        reporter.PhoneNumber,
        reporter.DisplayName,
        reporter.OrgUnitID,
        reporter.IsActive,
        reporter.CreatedAt,
        reporter.UpdatedAt,
    ).Scan(&id, &uid)
    if err != nil {
        return Reporter{}, err
    }
    reporter.ID = id
    reporter.UID = uid
    return reporter, nil
}

// Update modifies an existing reporter.  Only mutable fields are updated.  An
// error is returned when no rows are affected (e.g. reporter not found).
func (r *PgRepository) Update(ctx context.Context, reporter Reporter) (Reporter, error) {
    res, err := r.db.ExecContext(ctx, `UPDATE reporters SET phone_number = $1, display_name = $2, org_unit_id = $3, is_active = $4, updated_at = $5 WHERE id = $6`,
        reporter.PhoneNumber,
        reporter.DisplayName,
        reporter.OrgUnitID,
        reporter.IsActive,
        reporter.UpdatedAt,
        reporter.ID,
    )
    if err != nil {
        return Reporter{}, err
    }
    rows, err := res.RowsAffected()
    if err != nil {
        return Reporter{}, err
    }
    if rows == 0 {
        return Reporter{}, sql.ErrNoRows
    }
    return reporter, nil
}

// Delete removes a reporter by ID.  It returns an error if no row is deleted.
func (r *PgRepository) Delete(ctx context.Context, id int64) error {
    res, err := r.db.ExecContext(ctx, "DELETE FROM reporters WHERE id = $1", id)
    if err != nil {
        return err
    }
    rows, err := res.RowsAffected()
    if err != nil {
        return err
    }
    if rows == 0 {
        return sql.ErrNoRows
    }
    return nil
}