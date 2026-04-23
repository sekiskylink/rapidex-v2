package reportergroup

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

type SQLRepository struct {
	db *sqlx.DB
}

func NewSQLRepository(db *sqlx.DB) *SQLRepository {
	return &SQLRepository{db: db}
}

func (r *SQLRepository) List(ctx context.Context, query ListQuery) (ListResult, error) {
	result := ListResult{Page: query.Page, PageSize: query.PageSize}
	where := "WHERE 1=1"
	args := []any{}
	if strings.TrimSpace(query.Search) != "" {
		where += " AND LOWER(name) LIKE LOWER(?)"
		args = append(args, "%"+strings.TrimSpace(query.Search)+"%")
	}
	if query.ActiveOnly {
		where += " AND is_active = TRUE"
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM reporter_group_catalog %s", where)
	if err := r.db.GetContext(ctx, &result.Total, r.db.Rebind(countQuery), args...); err != nil {
		return result, fmt.Errorf("count reporter groups: %w", err)
	}

	limit := query.PageSize
	if limit <= 0 {
		limit = 20
	}
	offset := query.Page * limit
	listQuery := fmt.Sprintf(`
		SELECT id, name, is_active, created_at, updated_at
		FROM reporter_group_catalog
		%s
		ORDER BY LOWER(name) ASC, id ASC
		LIMIT %d OFFSET %d
	`, where, limit, offset)
	rows := []ReporterGroup{}
	if err := r.db.SelectContext(ctx, &rows, r.db.Rebind(listQuery), args...); err != nil {
		return result, fmt.Errorf("list reporter groups: %w", err)
	}
	result.Items = rows
	return result, nil
}

func (r *SQLRepository) GetByID(ctx context.Context, id int64) (ReporterGroup, error) {
	var item ReporterGroup
	if err := r.db.GetContext(ctx, &item, `
		SELECT id, name, is_active, created_at, updated_at
		FROM reporter_group_catalog
		WHERE id = $1
	`, id); err != nil {
		if err == sql.ErrNoRows {
			return ReporterGroup{}, sql.ErrNoRows
		}
		return ReporterGroup{}, fmt.Errorf("get reporter group by id: %w", err)
	}
	return item, nil
}

func (r *SQLRepository) GetByName(ctx context.Context, name string) (ReporterGroup, error) {
	var item ReporterGroup
	if err := r.db.GetContext(ctx, &item, `
		SELECT id, name, is_active, created_at, updated_at
		FROM reporter_group_catalog
		WHERE LOWER(name) = LOWER($1)
	`, strings.TrimSpace(name)); err != nil {
		if err == sql.ErrNoRows {
			return ReporterGroup{}, sql.ErrNoRows
		}
		return ReporterGroup{}, fmt.Errorf("get reporter group by name: %w", err)
	}
	return item, nil
}

func (r *SQLRepository) ListByNames(ctx context.Context, names []string, activeOnly bool) ([]ReporterGroup, error) {
	if len(names) == 0 {
		return []ReporterGroup{}, nil
	}
	normalized := make([]string, 0, len(names))
	for _, name := range names {
		if trimmed := strings.TrimSpace(name); trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	if len(normalized) == 0 {
		return []ReporterGroup{}, nil
	}
	query, args, err := sqlx.In(`
		SELECT id, name, is_active, created_at, updated_at
		FROM reporter_group_catalog
		WHERE LOWER(name) IN (?)
	`, lowerStrings(normalized))
	if err != nil {
		return nil, fmt.Errorf("build reporter groups by names query: %w", err)
	}
	if activeOnly {
		query += " AND is_active = TRUE"
	}
	query += " ORDER BY LOWER(name) ASC, id ASC"
	query = r.db.Rebind(query)
	items := []ReporterGroup{}
	if err := r.db.SelectContext(ctx, &items, query, args...); err != nil {
		return nil, fmt.Errorf("list reporter groups by names: %w", err)
	}
	return items, nil
}

func (r *SQLRepository) Create(ctx context.Context, item ReporterGroup) (ReporterGroup, error) {
	if err := r.db.GetContext(ctx, &item, `
		INSERT INTO reporter_group_catalog (name, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, is_active, created_at, updated_at
	`, item.Name, item.IsActive, item.CreatedAt, item.UpdatedAt); err != nil {
		return ReporterGroup{}, fmt.Errorf("create reporter group: %w", err)
	}
	return item, nil
}

func (r *SQLRepository) Update(ctx context.Context, item ReporterGroup) (ReporterGroup, error) {
	if err := r.db.GetContext(ctx, &item, `
		UPDATE reporter_group_catalog
		SET name = $2, is_active = $3, updated_at = $4
		WHERE id = $1
		RETURNING id, name, is_active, created_at, updated_at
	`, item.ID, item.Name, item.IsActive, item.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return ReporterGroup{}, sql.ErrNoRows
		}
		return ReporterGroup{}, fmt.Errorf("update reporter group: %w", err)
	}
	return item, nil
}

func lowerStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, strings.ToLower(strings.TrimSpace(value)))
	}
	return result
}

var _ Repository = (*SQLRepository)(nil)
