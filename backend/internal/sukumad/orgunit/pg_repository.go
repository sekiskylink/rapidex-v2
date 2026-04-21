package orgunit

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type PgRepository struct {
	db *sqlx.DB
}

func NewPgRepository(db *sqlx.DB) *PgRepository {
	return &PgRepository{db: db}
}

func (r *PgRepository) List(ctx context.Context, query ListQuery) (ListResult, error) {
	result := ListResult{Page: query.Page, PageSize: query.PageSize}
	where := "WHERE 1=1"
	args := []interface{}{}

	if query.Search != "" {
		where += " AND (LOWER(code) LIKE LOWER(?) OR LOWER(name) LIKE LOWER(?))"
		search := fmt.Sprintf("%%%s%%", query.Search)
		args = append(args, search, search)
	}
	if query.ParentID != nil {
		where += " AND parent_id = ?"
		args = append(args, *query.ParentID)
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM org_units %s", where)
	if err := r.db.GetContext(ctx, &total, r.db.Rebind(countQuery), args...); err != nil {
		return result, err
	}

	limit := query.PageSize
	if limit <= 0 {
		limit = 20
	}
	offset := query.Page * limit
	listQuery := fmt.Sprintf(`
		SELECT id, uid, code, name, description, parent_id, path, created_at, updated_at
		FROM org_units
		%s
		ORDER BY path ASC, name ASC
		LIMIT %d OFFSET %d
	`, where, limit, offset)

	items := []OrgUnit{}
	if err := r.db.SelectContext(ctx, &items, r.db.Rebind(listQuery), args...); err != nil {
		return result, err
	}

	result.Total = total
	result.Items = items
	return result, nil
}

func (r *PgRepository) GetByID(ctx context.Context, id int64) (OrgUnit, error) {
	var unit OrgUnit
	err := r.db.GetContext(ctx, &unit, `
		SELECT id, uid, code, name, description, parent_id, path, created_at, updated_at
		FROM org_units
		WHERE id = $1
	`, id)
	return unit, err
}

func (r *PgRepository) GetByUID(ctx context.Context, uid string) (OrgUnit, error) {
	var unit OrgUnit
	err := r.db.GetContext(ctx, &unit, `
		SELECT id, uid, code, name, description, parent_id, path, created_at, updated_at
		FROM org_units
		WHERE uid = $1
	`, uid)
	return unit, err
}

func (r *PgRepository) GetByCode(ctx context.Context, code string) (OrgUnit, error) {
	var unit OrgUnit
	err := r.db.GetContext(ctx, &unit, `
		SELECT id, uid, code, name, description, parent_id, path, created_at, updated_at
		FROM org_units
		WHERE code = $1
	`, code)
	return unit, err
}

func (r *PgRepository) Create(ctx context.Context, unit OrgUnit) (OrgUnit, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return OrgUnit{}, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	parentPath := "/"
	if unit.ParentID != nil {
		var parent OrgUnit
		if err := tx.GetContext(ctx, &parent, `
			SELECT id, uid, code, name, description, parent_id, path, created_at, updated_at
			FROM org_units
			WHERE id = $1
		`, *unit.ParentID); err != nil {
			return OrgUnit{}, err
		}
		parentPath = parent.Path
	}

	err = tx.QueryRowxContext(ctx, `
		INSERT INTO org_units (uid, code, name, description, parent_id, path, created_at, updated_at)
		VALUES (COALESCE(NULLIF($1, ''), gen_random_uuid()::text), $2, $3, $4, $5, '', $6, $7)
		RETURNING id, uid
	`, unit.UID, unit.Code, unit.Name, unit.Description, unit.ParentID, unit.CreatedAt, unit.UpdatedAt).Scan(&unit.ID, &unit.UID)
	if err != nil {
		return OrgUnit{}, err
	}

	unit.Path = fmt.Sprintf("%s%d/", parentPath, unit.ID)
	if _, err := tx.ExecContext(ctx, `UPDATE org_units SET path = $1 WHERE id = $2`, unit.Path, unit.ID); err != nil {
		return OrgUnit{}, err
	}

	if err := tx.Commit(); err != nil {
		return OrgUnit{}, err
	}
	tx = nil
	return unit, nil
}

func (r *PgRepository) Update(ctx context.Context, unit OrgUnit) (OrgUnit, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE org_units
		SET code = $1, name = $2, description = $3, parent_id = $4, updated_at = $5
		WHERE id = $6
	`, unit.Code, unit.Name, unit.Description, unit.ParentID, unit.UpdatedAt, unit.ID)
	if err != nil {
		return OrgUnit{}, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return OrgUnit{}, err
	}
	if rows == 0 {
		return OrgUnit{}, sql.ErrNoRows
	}
	return r.GetByID(ctx, unit.ID)
}

func (r *PgRepository) Delete(ctx context.Context, id int64) error {
	var childCount int
	if err := r.db.GetContext(ctx, &childCount, `SELECT COUNT(*) FROM org_units WHERE parent_id = $1`, id); err != nil {
		return err
	}
	if childCount > 0 {
		return fmt.Errorf("organisation unit has child units")
	}

	res, err := r.db.ExecContext(ctx, `DELETE FROM org_units WHERE id = $1`, id)
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
