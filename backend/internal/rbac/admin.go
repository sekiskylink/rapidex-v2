package rbac

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/audit"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

type RoleRecord struct {
	ID        int64     `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

type RoleSummary struct {
	RoleRecord
	PermissionCount int `db:"permission_count" json:"permissionCount"`
	UserCount       int `db:"user_count" json:"userCount"`
}

type PermissionRecord struct {
	ID          int64      `db:"id" json:"id"`
	Name        string     `db:"name" json:"name"`
	ModuleScope *string    `db:"module_scope" json:"moduleScope,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"createdAt"`
	AssignedAt  *time.Time `db:"assigned_at" json:"assignedAt,omitempty"`
}

type RoleUserRecord struct {
	ID       int64  `db:"id" json:"id"`
	Username string `db:"username" json:"username"`
	IsActive bool   `db:"is_active" json:"isActive"`
}

type RoleDetail struct {
	RoleRecord
	Permissions []PermissionRecord `json:"permissions"`
	Users       []RoleUserRecord   `json:"users,omitempty"`
}

type RoleListQuery struct {
	Page      int
	PageSize  int
	SortField string
	SortOrder string
	Filter    string
}

type PermissionListQuery struct {
	Page            int
	PageSize        int
	SortField       string
	SortOrder       string
	Query           string
	ModuleScope     *string
	AllowedNames    []string
	FilterToAllowed bool
}

type RoleListResult struct {
	Items    []RoleSummary
	Total    int
	Page     int
	PageSize int
}

type PermissionListResult struct {
	Items    []PermissionRecord
	Total    int
	Page     int
	PageSize int
}

type RoleCreateInput struct {
	Name        string
	Permissions []string
	ActorUserID *int64
}

type RoleUpdateInput struct {
	RoleID      int64
	Name        *string
	Permissions *[]string
	ActorUserID *int64
}

type AdminRepository interface {
	ListRoles(ctx context.Context, query RoleListQuery) (RoleListResult, error)
	CreateRole(ctx context.Context, name string) (RoleRecord, error)
	UpdateRoleName(ctx context.Context, roleID int64, name string) (RoleRecord, error)
	GetRoleByID(ctx context.Context, roleID int64) (RoleRecord, error)
	ListRolePermissions(ctx context.Context, roleID int64) ([]PermissionRecord, error)
	ListRoleUsers(ctx context.Context, roleID int64) ([]RoleUserRecord, error)
	ListPermissions(ctx context.Context, query PermissionListQuery) (PermissionListResult, error)
	GetPermissionsByNames(ctx context.Context, names []string) ([]PermissionRecord, error)
	ReplaceRolePermissions(ctx context.Context, roleID int64, permissionIDs []int64) error
}

func normalizeRoleListQuery(query RoleListQuery) RoleListQuery {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 25
	}
	sortField := strings.TrimSpace(query.SortField)
	switch sortField {
	case "id", "name", "created_at", "updated_at", "permission_count", "user_count":
	default:
		sortField = "name"
	}
	sortOrder := strings.ToLower(strings.TrimSpace(query.SortOrder))
	if sortOrder != "desc" {
		sortOrder = "asc"
	}
	return RoleListQuery{
		Page:      page,
		PageSize:  pageSize,
		SortField: sortField,
		SortOrder: sortOrder,
		Filter:    strings.TrimSpace(query.Filter),
	}
}

func normalizePermissionListQuery(query PermissionListQuery) PermissionListQuery {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 25
	}
	sortField := strings.TrimSpace(query.SortField)
	switch sortField {
	case "id", "name", "module_scope", "created_at":
	default:
		sortField = "name"
	}
	sortOrder := strings.ToLower(strings.TrimSpace(query.SortOrder))
	if sortOrder != "desc" {
		sortOrder = "asc"
	}
	var moduleScope *string
	if query.ModuleScope != nil {
		trimmed := strings.TrimSpace(*query.ModuleScope)
		if trimmed != "" {
			moduleScope = &trimmed
		}
	}
	return PermissionListQuery{
		Page:            page,
		PageSize:        pageSize,
		SortField:       sortField,
		SortOrder:       sortOrder,
		Query:           strings.TrimSpace(query.Query),
		ModuleScope:     moduleScope,
		AllowedNames:    uniqueStrings(query.AllowedNames),
		FilterToAllowed: query.FilterToAllowed,
	}
}

func (r *SQLRepository) ListRoles(ctx context.Context, query RoleListQuery) (RoleListResult, error) {
	q := normalizeRoleListQuery(query)
	offset := (q.Page - 1) * q.PageSize

	filterArgs := []any{}
	where := ""
	if q.Filter != "" {
		filterArgs = append(filterArgs, "%"+q.Filter+"%")
		where = fmt.Sprintf(" WHERE r.name ILIKE $%d", len(filterArgs))
	}

	total := 0
	countQuery := `SELECT COUNT(*) FROM roles r` + where
	if err := r.db.GetContext(ctx, &total, countQuery, filterArgs...); err != nil {
		return RoleListResult{}, fmt.Errorf("count roles: %w", err)
	}

	args := append(filterArgs, q.PageSize, offset)
	rows := []RoleSummary{}
	listQuery := `
		SELECT
			r.id,
			r.name,
			r.created_at,
			r.updated_at,
			COUNT(DISTINCT rp.permission_id) AS permission_count,
			COUNT(DISTINCT ur.user_id) AS user_count
		FROM roles r
		LEFT JOIN role_permissions rp ON rp.role_id = r.id
		LEFT JOIN user_roles ur ON ur.role_id = r.id
	` + where + `
		GROUP BY r.id, r.name, r.created_at, r.updated_at
		ORDER BY ` + q.SortField + ` ` + strings.ToUpper(q.SortOrder) + `
		LIMIT $` + fmt.Sprintf("%d", len(args)-1) + ` OFFSET $` + fmt.Sprintf("%d", len(args))

	if err := r.db.SelectContext(ctx, &rows, listQuery, args...); err != nil {
		return RoleListResult{}, fmt.Errorf("list roles: %w", err)
	}

	return RoleListResult{
		Items:    rows,
		Total:    total,
		Page:     q.Page,
		PageSize: q.PageSize,
	}, nil
}

func (r *SQLRepository) CreateRole(ctx context.Context, name string) (RoleRecord, error) {
	var role RoleRecord
	err := r.db.GetContext(ctx, &role, `
		INSERT INTO roles (name, created_at, updated_at)
		VALUES ($1, NOW(), NOW())
		RETURNING id, name, created_at, updated_at
	`, strings.TrimSpace(name))
	if err != nil {
		return RoleRecord{}, fmt.Errorf("create role: %w", err)
	}
	return role, nil
}

func (r *SQLRepository) UpdateRoleName(ctx context.Context, roleID int64, name string) (RoleRecord, error) {
	var role RoleRecord
	err := r.db.GetContext(ctx, &role, `
		UPDATE roles
		SET name = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, created_at, updated_at
	`, roleID, strings.TrimSpace(name))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RoleRecord{}, sql.ErrNoRows
		}
		return RoleRecord{}, fmt.Errorf("update role: %w", err)
	}
	return role, nil
}

func (r *SQLRepository) GetRoleByID(ctx context.Context, roleID int64) (RoleRecord, error) {
	var role RoleRecord
	err := r.db.GetContext(ctx, &role, `SELECT id, name, created_at, updated_at FROM roles WHERE id = $1`, roleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RoleRecord{}, sql.ErrNoRows
		}
		return RoleRecord{}, fmt.Errorf("get role by id: %w", err)
	}
	return role, nil
}

func (r *SQLRepository) ListRolePermissions(ctx context.Context, roleID int64) ([]PermissionRecord, error) {
	items := []PermissionRecord{}
	err := r.db.SelectContext(ctx, &items, `
		SELECT p.id, p.name, p.module_scope, p.created_at, rp.created_at AS assigned_at
		FROM role_permissions rp
		JOIN permissions p ON p.id = rp.permission_id
		WHERE rp.role_id = $1
		ORDER BY p.name ASC
	`, roleID)
	if err != nil {
		return nil, fmt.Errorf("list role permissions: %w", err)
	}
	return items, nil
}

func (r *SQLRepository) ListRoleUsers(ctx context.Context, roleID int64) ([]RoleUserRecord, error) {
	items := []RoleUserRecord{}
	err := r.db.SelectContext(ctx, &items, `
		SELECT u.id, u.username, u.is_active
		FROM user_roles ur
		JOIN users u ON u.id = ur.user_id
		WHERE ur.role_id = $1
		ORDER BY u.username ASC
	`, roleID)
	if err != nil {
		return nil, fmt.Errorf("list role users: %w", err)
	}
	return items, nil
}

func (r *SQLRepository) ListPermissions(ctx context.Context, query PermissionListQuery) (PermissionListResult, error) {
	q := normalizePermissionListQuery(query)
	if q.FilterToAllowed && len(q.AllowedNames) == 0 {
		return PermissionListResult{
			Items:    []PermissionRecord{},
			Total:    0,
			Page:     q.Page,
			PageSize: q.PageSize,
		}, nil
	}

	offset := (q.Page - 1) * q.PageSize

	conditions := []string{}
	args := []any{}
	if q.Query != "" {
		args = append(args, "%"+q.Query+"%")
		idx := len(args)
		conditions = append(conditions, fmt.Sprintf("(p.name ILIKE $%d OR COALESCE(p.module_scope, '') ILIKE $%d)", idx, idx))
	}
	if q.ModuleScope != nil {
		args = append(args, *q.ModuleScope)
		conditions = append(conditions, fmt.Sprintf("p.module_scope = $%d", len(args)))
	}
	if q.FilterToAllowed {
		placeholders := make([]string, 0, len(q.AllowedNames))
		for _, name := range q.AllowedNames {
			args = append(args, name)
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)))
		}
		conditions = append(conditions, fmt.Sprintf("p.name IN (%s)", strings.Join(placeholders, ", ")))
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	total := 0
	countQuery := `SELECT COUNT(*) FROM permissions p` + where
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return PermissionListResult{}, fmt.Errorf("count permissions: %w", err)
	}

	args = append(args, q.PageSize, offset)
	items := []PermissionRecord{}
	listQuery := `
		SELECT p.id, p.name, p.module_scope, p.created_at, NULL::timestamptz AS assigned_at
		FROM permissions p
	` + where + `
		ORDER BY p.` + q.SortField + ` ` + strings.ToUpper(q.SortOrder) + `
		LIMIT $` + fmt.Sprintf("%d", len(args)-1) + ` OFFSET $` + fmt.Sprintf("%d", len(args))

	if err := r.db.SelectContext(ctx, &items, listQuery, args...); err != nil {
		return PermissionListResult{}, fmt.Errorf("list permissions: %w", err)
	}

	return PermissionListResult{
		Items:    items,
		Total:    total,
		Page:     q.Page,
		PageSize: q.PageSize,
	}, nil
}

func (r *SQLRepository) GetPermissionsByNames(ctx context.Context, names []string) ([]PermissionRecord, error) {
	cleaned := uniqueStrings(names)
	if len(cleaned) == 0 {
		return []PermissionRecord{}, nil
	}

	query, args, err := sqlx.In(`SELECT id, name, module_scope, created_at, NULL::timestamptz AS assigned_at FROM permissions WHERE name IN (?)`, cleaned)
	if err != nil {
		return nil, fmt.Errorf("build permission lookup query: %w", err)
	}
	query = r.db.Rebind(query)

	items := []PermissionRecord{}
	if err := r.db.SelectContext(ctx, &items, query, args...); err != nil {
		return nil, fmt.Errorf("get permissions by names: %w", err)
	}
	return items, nil
}

func (r *SQLRepository) ReplaceRolePermissions(ctx context.Context, roleID int64, permissionIDs []int64) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin replace role permissions tx: %w", err)
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, `DELETE FROM role_permissions WHERE role_id = $1`, roleID); err != nil {
		return fmt.Errorf("clear role permissions: %w", err)
	}
	for _, permissionID := range permissionIDs {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO role_permissions (role_id, permission_id, created_at)
			VALUES ($1, $2, NOW())
		`, roleID, permissionID); err != nil {
			return fmt.Errorf("insert role permission: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit replace role permissions tx: %w", err)
	}
	tx = nil
	return nil
}

type AdminService struct {
	repo                AdminRepository
	auditService        *audit.Service
	isPermissionEnabled func(permissionName string) bool
}

type AdminServiceOption func(*AdminService)

func WithPermissionEnablementFilter(filter func(permissionName string) bool) AdminServiceOption {
	return func(service *AdminService) {
		if filter == nil {
			return
		}
		service.isPermissionEnabled = filter
	}
}

func NewAdminService(repo AdminRepository, auditService *audit.Service, opts ...AdminServiceOption) *AdminService {
	service := &AdminService{
		repo:         repo,
		auditService: auditService,
		isPermissionEnabled: func(_ string) bool {
			return true
		},
	}
	for _, opt := range opts {
		opt(service)
	}
	return service
}

func (s *AdminService) ListRoles(ctx context.Context, query RoleListQuery) (RoleListResult, error) {
	return s.repo.ListRoles(ctx, query)
}

func (s *AdminService) GetRoleDetail(ctx context.Context, roleID int64, includeUsers bool) (RoleDetail, error) {
	role, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RoleDetail{}, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"role not found"}})
		}
		return RoleDetail{}, err
	}

	permissions, err := s.repo.ListRolePermissions(ctx, roleID)
	if err != nil {
		return RoleDetail{}, err
	}
	permissions = s.filterPermissionRecords(permissions)

	detail := RoleDetail{
		RoleRecord:  role,
		Permissions: permissions,
		Users:       []RoleUserRecord{},
	}
	if includeUsers {
		users, usersErr := s.repo.ListRoleUsers(ctx, roleID)
		if usersErr != nil {
			return RoleDetail{}, usersErr
		}
		detail.Users = users
	}

	return detail, nil
}

func (s *AdminService) CreateRole(ctx context.Context, in RoleCreateInput) (RoleDetail, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return RoleDetail{}, apperror.ValidationWithDetails("validation failed", map[string]any{"name": []string{"is required"}})
	}

	resolvedPermissionNames, resolvedPermissionIDs, err := s.resolvePermissionNames(ctx, in.Permissions)
	if err != nil {
		return RoleDetail{}, err
	}

	role, err := s.repo.CreateRole(ctx, name)
	if err != nil {
		if mapped := mapRoleConstraintError(err); mapped != nil {
			return RoleDetail{}, mapped
		}
		return RoleDetail{}, err
	}

	if err := s.repo.ReplaceRolePermissions(ctx, role.ID, resolvedPermissionIDs); err != nil {
		return RoleDetail{}, err
	}

	permissions, err := s.repo.ListRolePermissions(ctx, role.ID)
	if err != nil {
		return RoleDetail{}, err
	}
	permissions = s.filterPermissionRecords(permissions)

	s.logAudit(ctx, audit.Event{
		Action:      "roles.create",
		ActorUserID: in.ActorUserID,
		EntityType:  "role",
		EntityID:    strPtr(fmt.Sprintf("%d", role.ID)),
		Metadata: map[string]any{
			"name":        role.Name,
			"permissions": resolvedPermissionNames,
		},
	})

	return RoleDetail{
		RoleRecord:  role,
		Permissions: permissions,
		Users:       []RoleUserRecord{},
	}, nil
}

func (s *AdminService) UpdateRole(ctx context.Context, in RoleUpdateInput) (RoleDetail, error) {
	if in.Name == nil && in.Permissions == nil {
		return RoleDetail{}, apperror.ValidationWithDetails("validation failed", map[string]any{"body": []string{"at least one update field is required"}})
	}

	role, err := s.repo.GetRoleByID(ctx, in.RoleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RoleDetail{}, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"role not found"}})
		}
		return RoleDetail{}, err
	}

	metadata := map[string]any{}
	if in.Name != nil {
		trimmed := strings.TrimSpace(*in.Name)
		if trimmed == "" {
			return RoleDetail{}, apperror.ValidationWithDetails("validation failed", map[string]any{"name": []string{"cannot be empty"}})
		}
		updatedRole, updateErr := s.repo.UpdateRoleName(ctx, in.RoleID, trimmed)
		if updateErr != nil {
			if mapped := mapRoleConstraintError(updateErr); mapped != nil {
				return RoleDetail{}, mapped
			}
			return RoleDetail{}, updateErr
		}
		role = updatedRole
		metadata["name"] = trimmed
	}

	if in.Permissions != nil {
		resolvedPermissionNames, resolvedPermissionIDs, resolveErr := s.resolvePermissionNames(ctx, *in.Permissions)
		if resolveErr != nil {
			return RoleDetail{}, resolveErr
		}
		if replaceErr := s.repo.ReplaceRolePermissions(ctx, in.RoleID, resolvedPermissionIDs); replaceErr != nil {
			return RoleDetail{}, replaceErr
		}
		metadata["permissions"] = resolvedPermissionNames
	}

	permissions, err := s.repo.ListRolePermissions(ctx, in.RoleID)
	if err != nil {
		return RoleDetail{}, err
	}
	permissions = s.filterPermissionRecords(permissions)
	detail := RoleDetail{
		RoleRecord:  role,
		Permissions: permissions,
		Users:       []RoleUserRecord{},
	}

	if len(metadata) > 0 {
		s.logAudit(ctx, audit.Event{
			Action:      "roles.update",
			ActorUserID: in.ActorUserID,
			EntityType:  "role",
			EntityID:    strPtr(fmt.Sprintf("%d", in.RoleID)),
			Metadata:    metadata,
		})
	}

	return detail, nil
}

func (s *AdminService) ListPermissions(ctx context.Context, query PermissionListQuery) (PermissionListResult, error) {
	query.AllowedNames = s.allowedPermissionNames()
	query.FilterToAllowed = true
	return s.repo.ListPermissions(ctx, query)
}

func (s *AdminService) resolvePermissionNames(ctx context.Context, names []string) ([]string, []int64, error) {
	clean := uniqueStrings(names)
	if len(clean) > 0 {
		allowed := map[string]struct{}{}
		for _, name := range s.allowedPermissionNames() {
			allowed[name] = struct{}{}
		}
		for _, requested := range clean {
			if _, ok := allowed[requested]; !ok {
				return nil, nil, apperror.ValidationWithDetails("validation failed", map[string]any{"permissions": []string{"one or more permissions are invalid"}})
			}
		}
	}
	permissions, err := s.repo.GetPermissionsByNames(ctx, clean)
	if err != nil {
		return nil, nil, err
	}
	if len(permissions) != len(clean) {
		return nil, nil, apperror.ValidationWithDetails("validation failed", map[string]any{"permissions": []string{"one or more permissions are invalid"}})
	}
	ids := make([]int64, 0, len(permissions))
	for _, permission := range permissions {
		ids = append(ids, permission.ID)
	}
	return clean, ids, nil
}

func (s *AdminService) logAudit(ctx context.Context, event audit.Event) {
	if s.auditService == nil {
		return
	}
	_ = s.auditService.Log(ctx, event)
}

func (s *AdminService) allowedPermissionNames() []string {
	definitions := BasePermissionRegistry()
	names := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		if s.isPermissionEnabled(definition.Key) {
			names = append(names, definition.Key)
		}
	}
	return names
}

func (s *AdminService) filterPermissionRecords(items []PermissionRecord) []PermissionRecord {
	if len(items) == 0 {
		return items
	}
	filtered := make([]PermissionRecord, 0, len(items))
	for _, item := range items {
		if s.isPermissionEnabled(item.Name) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func mapRoleConstraintError(err error) *apperror.AppError {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return nil
	}
	if pgErr.Code != "23505" {
		return nil
	}
	if strings.Contains(pgErr.ConstraintName, "roles_name") || strings.Contains(pgErr.ConstraintName, "roles_name_key") {
		return apperror.ValidationWithDetails("validation failed", map[string]any{"name": []string{"must be unique"}})
	}
	return apperror.ValidationWithDetails("validation failed", map[string]any{"record": []string{"already exists"}})
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func strPtr(v string) *string {
	return &v
}
