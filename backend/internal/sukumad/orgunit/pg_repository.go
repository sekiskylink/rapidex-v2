package orgunit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

type PgRepository struct {
	db *sqlx.DB
}

const orgUnitDisplayPathSQL = `
	COALESCE(
		(
			SELECT string_agg(ancestor.name, ' / ' ORDER BY path_parts.ordinality)
			FROM unnest(string_to_array(trim(org_units.path, '/'), '/')) WITH ORDINALITY AS path_parts(uid, ordinality)
			JOIN org_units ancestor ON ancestor.uid = path_parts.uid
			WHERE ancestor.uid <> org_units.uid
		),
		''
	) AS display_path`

type orgUnitRow struct {
	ID              int64      `db:"id"`
	UID             string     `db:"uid"`
	Code            string     `db:"code"`
	Name            string     `db:"name"`
	ShortName       string     `db:"short_name"`
	Description     string     `db:"description"`
	ParentID        *int64     `db:"parent_id"`
	HierarchyLevel  int        `db:"hierarchy_level"`
	Path            string     `db:"path"`
	DisplayPath     string     `db:"display_path"`
	Address         string     `db:"address"`
	Email           string     `db:"email"`
	URL             string     `db:"url"`
	PhoneNumber     string     `db:"phone_number"`
	Extras          JSONMap    `db:"extras"`
	AttributeValues JSONMap    `db:"attribute_values"`
	OpeningDate     *time.Time `db:"opening_date"`
	Deleted         bool       `db:"deleted"`
	LastSyncDate    *time.Time `db:"last_sync_date"`
	HasChildren     bool       `db:"has_children"`
	CreatedAt       time.Time  `db:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"`
}

func NewPgRepository(db *sqlx.DB) *PgRepository {
	return &PgRepository{db: db}
}

func (r *PgRepository) List(ctx context.Context, query ListQuery) (ListResult, error) {
	result := ListResult{Page: query.Page, PageSize: query.PageSize}
	where := "WHERE 1=1"
	args := []interface{}{}

	if query.ScopeRestricted && len(query.ScopePaths) == 0 && !(query.RootsOnly && len(query.ScopeRootIDs) > 0) {
		result.Total = 0
		result.Items = []OrgUnit{}
		return result, nil
	}

	if strings.TrimSpace(query.Search) != "" {
		where += " AND (LOWER(code) LIKE LOWER(?) OR LOWER(name) LIKE LOWER(?) OR LOWER(short_name) LIKE LOWER(?))"
		search := fmt.Sprintf("%%%s%%", strings.TrimSpace(query.Search))
		args = append(args, search, search, search)
	}
	if query.ParentID != nil {
		where += " AND parent_id = ?"
		args = append(args, *query.ParentID)
	} else if query.RootsOnly {
		if query.ScopeRestricted {
			if len(query.ScopeRootIDs) == 0 {
				result.Total = 0
				result.Items = []OrgUnit{}
				return result, nil
			}
			rootPlaceholders := make([]string, 0, len(query.ScopeRootIDs))
			for _, rootID := range query.ScopeRootIDs {
				rootPlaceholders = append(rootPlaceholders, "?")
				args = append(args, rootID)
			}
			where += " AND id IN (" + strings.Join(rootPlaceholders, ", ") + ")"
		} else {
			where += " AND parent_id IS NULL"
		}
	}
	if query.LeafOnly {
		where += " AND NOT EXISTS (SELECT 1 FROM org_units child WHERE child.parent_id = org_units.id)"
	}
	if query.ScopeRestricted && len(query.ScopePaths) > 0 {
		pathClauses := make([]string, 0, len(query.ScopePaths))
		for _, path := range query.ScopePaths {
			pathClauses = append(pathClauses, "org_units.path LIKE ?")
			args = append(args, path+"%")
		}
		where += " AND (" + strings.Join(pathClauses, " OR ") + ")"
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
		SELECT id, uid, code, name, short_name, description, parent_id, hierarchy_level, path,
		       %s,
		       address, email, url, phone_number, extras, attribute_values, opening_date, deleted,
		       EXISTS (SELECT 1 FROM org_units child WHERE child.parent_id = org_units.id) AS has_children,
		       last_sync_date, created_at, updated_at
		FROM org_units
		%s
		ORDER BY path ASC, name ASC
		LIMIT %d OFFSET %d
	`, orgUnitDisplayPathSQL, where, limit, offset)

	rows := []orgUnitRow{}
	if err := r.db.SelectContext(ctx, &rows, r.db.Rebind(listQuery), args...); err != nil {
		return result, err
	}

	result.Total = total
	result.Items = convertOrgUnitRows(rows)
	return result, nil
}

func (r *PgRepository) GetByID(ctx context.Context, id int64) (OrgUnit, error) {
	return r.getByWhere(ctx, "id = $1", id)
}

func (r *PgRepository) GetByUID(ctx context.Context, uid string) (OrgUnit, error) {
	return r.getByWhere(ctx, "uid = $1", uid)
}

func (r *PgRepository) GetByCode(ctx context.Context, code string) (OrgUnit, error) {
	return r.getByWhere(ctx, "code = $1", code)
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

	parent, err := r.loadParentForMutation(ctx, tx, unit.ParentID, "")
	if err != nil {
		return OrgUnit{}, err
	}

	path := buildPath(parent, unit.UID)
	level := 1
	if parent != nil {
		level = parent.HierarchyLevel + 1
	}

	query := `
		INSERT INTO org_units (
			uid, code, name, short_name, description, parent_id, hierarchy_level, path,
			address, email, url, phone_number, extras, attribute_values, opening_date,
			deleted, last_sync_date, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19
		)
		RETURNING id
	`
	if err := tx.QueryRowxContext(
		ctx,
		query,
		unit.UID,
		unit.Code,
		unit.Name,
		unit.ShortName,
		unit.Description,
		unit.ParentID,
		level,
		path,
		unit.Address,
		unit.Email,
		unit.URL,
		unit.PhoneNumber,
		unit.Extras,
		unit.AttributeValues,
		unit.OpeningDate,
		unit.Deleted,
		unit.LastSyncDate,
		unit.CreatedAt,
		unit.UpdatedAt,
	).Scan(&unit.ID); err != nil {
		return OrgUnit{}, err
	}

	if err := tx.Commit(); err != nil {
		return OrgUnit{}, err
	}
	tx = nil
	return r.GetByID(ctx, unit.ID)
}

func (r *PgRepository) Update(ctx context.Context, unit OrgUnit) (OrgUnit, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return OrgUnit{}, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	existing, err := r.getByIDTx(ctx, tx, unit.ID)
	if err != nil {
		return OrgUnit{}, err
	}

	parent, err := r.loadParentForMutation(ctx, tx, unit.ParentID, existing.UID)
	if err != nil {
		return OrgUnit{}, err
	}

	level := 1
	if parent != nil {
		level = parent.HierarchyLevel + 1
	}
	newPath := buildPath(parent, existing.UID)
	oldPath := existing.Path
	levelDelta := level - existing.HierarchyLevel

	res, err := tx.ExecContext(ctx, `
		UPDATE org_units
		SET code = $1,
		    name = $2,
		    short_name = $3,
		    description = $4,
		    parent_id = $5,
		    hierarchy_level = $6,
		    path = $7,
		    address = $8,
		    email = $9,
		    url = $10,
		    phone_number = $11,
		    extras = $12,
		    attribute_values = $13,
		    opening_date = $14,
		    deleted = $15,
		    last_sync_date = $16,
		    updated_at = $17
		WHERE id = $18
	`,
		unit.Code,
		unit.Name,
		unit.ShortName,
		unit.Description,
		unit.ParentID,
		level,
		newPath,
		unit.Address,
		unit.Email,
		unit.URL,
		unit.PhoneNumber,
		unit.Extras,
		unit.AttributeValues,
		unit.OpeningDate,
		unit.Deleted,
		unit.LastSyncDate,
		unit.UpdatedAt,
		unit.ID,
	)
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

	if newPath != oldPath || levelDelta != 0 {
		if _, err := tx.ExecContext(ctx, `
			UPDATE org_units
			SET path = $1 || substring(path FROM $2),
			    hierarchy_level = hierarchy_level + $3
			WHERE path LIKE $4
			  AND id <> $5
		`, newPath, len(oldPath)+1, levelDelta, oldPath+"%", unit.ID); err != nil {
			return OrgUnit{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return OrgUnit{}, err
	}
	tx = nil
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
	var reporterCount int
	if err := r.db.GetContext(ctx, &reporterCount, `SELECT COUNT(*) FROM reporters WHERE org_unit_id = $1`, id); err != nil {
		return err
	}
	if reporterCount > 0 {
		return fmt.Errorf("organisation unit has assigned reporters")
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

func (r *PgRepository) getByWhere(ctx context.Context, where string, arg any) (OrgUnit, error) {
	row := orgUnitRow{}
	query := `
		SELECT id, uid, code, name, short_name, description, parent_id, hierarchy_level, path,
		       ` + orgUnitDisplayPathSQL + `,
		       address, email, url, phone_number, extras, attribute_values, opening_date, deleted,
		       EXISTS (SELECT 1 FROM org_units child WHERE child.parent_id = org_units.id) AS has_children,
		       last_sync_date, created_at, updated_at
		FROM org_units
		WHERE ` + where
	if err := r.db.GetContext(ctx, &row, query, arg); err != nil {
		return OrgUnit{}, err
	}
	return row.toOrgUnit(), nil
}

func (r *PgRepository) getByIDTx(ctx context.Context, tx *sqlx.Tx, id int64) (OrgUnit, error) {
	row := orgUnitRow{}
	if err := tx.GetContext(ctx, &row, `
		SELECT id, uid, code, name, short_name, description, parent_id, hierarchy_level, path,
		       `+orgUnitDisplayPathSQL+`,
		       address, email, url, phone_number, extras, attribute_values, opening_date, deleted,
		       EXISTS (SELECT 1 FROM org_units child WHERE child.parent_id = org_units.id) AS has_children,
		       last_sync_date, created_at, updated_at
		FROM org_units
		WHERE id = $1
	`, id); err != nil {
		return OrgUnit{}, err
	}
	return row.toOrgUnit(), nil
}

func (r *PgRepository) loadParentForMutation(ctx context.Context, tx *sqlx.Tx, parentID *int64, ownUID string) (*OrgUnit, error) {
	if parentID == nil {
		return nil, nil
	}

	parent, err := r.getByIDTx(ctx, tx, *parentID)
	if err != nil {
		return nil, err
	}
	if ownUID != "" {
		if parent.ID == *parentID && parent.UID == ownUID {
			return nil, errors.New("organisation unit cannot be its own parent")
		}
		if strings.Contains(parent.Path, "/"+ownUID+"/") {
			return nil, errors.New("organisation unit cannot be moved under its own descendant")
		}
	}
	return &parent, nil
}

func buildPath(parent *OrgUnit, uid string) string {
	if parent == nil {
		return "/" + uid + "/"
	}
	return parent.Path + uid + "/"
}

func convertOrgUnitRows(rows []orgUnitRow) []OrgUnit {
	items := make([]OrgUnit, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toOrgUnit())
	}
	return items
}

func (r orgUnitRow) toOrgUnit() OrgUnit {
	return OrgUnit{
		ID:              r.ID,
		UID:             r.UID,
		Code:            r.Code,
		Name:            r.Name,
		ShortName:       r.ShortName,
		Description:     r.Description,
		ParentID:        r.ParentID,
		HierarchyLevel:  r.HierarchyLevel,
		Path:            r.Path,
		DisplayPath:     r.DisplayPath,
		Address:         r.Address,
		Email:           r.Email,
		URL:             r.URL,
		PhoneNumber:     r.PhoneNumber,
		Extras:          defaultJSONMap(r.Extras),
		AttributeValues: defaultJSONMap(r.AttributeValues),
		OpeningDate:     r.OpeningDate,
		Deleted:         r.Deleted,
		LastSyncDate:    r.LastSyncDate,
		HasChildren:     r.HasChildren,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}
}

func defaultJSONMap(input JSONMap) JSONMap {
	if input == nil {
		return JSONMap{}
	}
	return input
}
