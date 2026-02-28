package rbac

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"

	"github.com/jmoiron/sqlx"
)

type Role struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
}

type Permission struct {
	ID          int64   `db:"id"`
	Name        string  `db:"name"`
	ModuleScope *string `db:"module_scope"`
}

type Repository interface {
	GetUserRoles(ctx context.Context, userID int64) ([]Role, error)
	GetUserPermissions(ctx context.Context, userID int64) ([]Permission, error)
	EnsureRole(ctx context.Context, name string) (Role, error)
	EnsurePermission(ctx context.Context, name string, moduleScope *string) (Permission, error)
	EnsureRolePermission(ctx context.Context, roleID, permissionID int64) error
	EnsureUserRole(ctx context.Context, userID, roleID int64) error
	GetRoleByName(ctx context.Context, name string) (Role, error)
}

type SQLRepository struct {
	db *sqlx.DB
}

func NewSQLRepository(db *sqlx.DB) *SQLRepository {
	return &SQLRepository{db: db}
}

func (r *SQLRepository) GetUserRoles(ctx context.Context, userID int64) ([]Role, error) {
	roles := []Role{}
	err := r.db.SelectContext(ctx, &roles, `
		SELECT r.id, r.name
		FROM roles r
		JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1
		ORDER BY r.name ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("get user roles: %w", err)
	}
	return roles, nil
}

func (r *SQLRepository) GetUserPermissions(ctx context.Context, userID int64) ([]Permission, error) {
	permissions := []Permission{}
	err := r.db.SelectContext(ctx, &permissions, `
		SELECT DISTINCT p.id, p.name, p.module_scope
		FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id
		JOIN user_roles ur ON ur.role_id = rp.role_id
		WHERE ur.user_id = $1
		ORDER BY p.name ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("get user permissions: %w", err)
	}
	return permissions, nil
}

func (r *SQLRepository) EnsureRole(ctx context.Context, name string) (Role, error) {
	var role Role
	err := r.db.GetContext(ctx, &role, `
		INSERT INTO roles (name, created_at, updated_at)
		VALUES ($1, NOW(), NOW())
		ON CONFLICT (name) DO UPDATE SET updated_at = roles.updated_at
		RETURNING id, name
	`, strings.TrimSpace(name))
	if err != nil {
		return Role{}, fmt.Errorf("ensure role: %w", err)
	}
	return role, nil
}

func (r *SQLRepository) EnsurePermission(ctx context.Context, name string, moduleScope *string) (Permission, error) {
	var permission Permission
	err := r.db.GetContext(ctx, &permission, `
		INSERT INTO permissions (name, module_scope, created_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (name) DO UPDATE SET module_scope = EXCLUDED.module_scope
		RETURNING id, name, module_scope
	`, strings.TrimSpace(name), moduleScope)
	if err != nil {
		return Permission{}, fmt.Errorf("ensure permission: %w", err)
	}
	return permission, nil
}

func (r *SQLRepository) EnsureRolePermission(ctx context.Context, roleID, permissionID int64) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO role_permissions (role_id, permission_id, created_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (role_id, permission_id) DO NOTHING
	`, roleID, permissionID)
	if err != nil {
		return fmt.Errorf("ensure role permission: %w", err)
	}
	return nil
}

func (r *SQLRepository) EnsureUserRole(ctx context.Context, userID, roleID int64) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO user_roles (user_id, role_id, created_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id, role_id) DO NOTHING
	`, userID, roleID)
	if err != nil {
		return fmt.Errorf("ensure user role: %w", err)
	}
	return nil
}

func (r *SQLRepository) GetRoleByName(ctx context.Context, name string) (Role, error) {
	var role Role
	err := r.db.GetContext(ctx, &role, `SELECT id, name FROM roles WHERE name = $1`, strings.TrimSpace(name))
	if err != nil {
		if err == sql.ErrNoRows {
			return Role{}, sql.ErrNoRows
		}
		return Role{}, fmt.Errorf("get role by name: %w", err)
	}
	return role, nil
}

type Service struct {
	repo Repository

	mu    sync.RWMutex
	cache map[int64]cachedPermissions
}

type cachedPermissions struct {
	roles       []Role
	permissions []Permission
}

func NewService(repo Repository) *Service {
	return &Service{
		repo:  repo,
		cache: map[int64]cachedPermissions{},
	}
}

func (s *Service) GetUserRoles(ctx context.Context, userID int64) ([]Role, error) {
	cached, ok := s.readCache(userID)
	if ok {
		return append([]Role{}, cached.roles...), nil
	}
	roles, err := s.repo.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, err
	}
	perms, err := s.repo.GetUserPermissions(ctx, userID)
	if err != nil {
		return nil, err
	}
	s.writeCache(userID, cachedPermissions{roles: roles, permissions: perms})
	return append([]Role{}, roles...), nil
}

func (s *Service) GetUserPermissions(ctx context.Context, userID int64) ([]Permission, error) {
	cached, ok := s.readCache(userID)
	if ok {
		return append([]Permission{}, cached.permissions...), nil
	}
	roles, err := s.repo.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, err
	}
	permissions, err := s.repo.GetUserPermissions(ctx, userID)
	if err != nil {
		return nil, err
	}
	s.writeCache(userID, cachedPermissions{roles: roles, permissions: permissions})
	return append([]Permission{}, permissions...), nil
}

func (s *Service) HasPermission(ctx context.Context, userID int64, perm string, moduleScope *string) (bool, error) {
	permissions, err := s.GetUserPermissions(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, p := range permissions {
		if p.Name != perm {
			continue
		}
		if moduleScope == nil {
			return true, nil
		}
		if p.ModuleScope == nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(*p.ModuleScope), strings.TrimSpace(*moduleScope)) {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) PermissionsForUser(ctx context.Context, userID int64) ([]string, error) {
	permissions, err := s.GetUserPermissions(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(permissions))
	for _, perm := range permissions {
		if perm.ModuleScope == nil || strings.TrimSpace(*perm.ModuleScope) == "" {
			result = append(result, perm.Name)
			continue
		}
		result = append(result, perm.Name+"@"+strings.TrimSpace(*perm.ModuleScope))
	}
	return result, nil
}

func (s *Service) RoleNamesForUser(ctx context.Context, userID int64) ([]string, error) {
	roles, err := s.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(roles))
	for _, role := range roles {
		result = append(result, role.Name)
	}
	return result, nil
}

func (s *Service) EnsureBaseRBAC(ctx context.Context) error {
	roleDefinitions := []string{"Admin", "Manager", "Staff", "Viewer"}
	permissionDefinitions := []Permission{
		{Name: "users.read"},
		{Name: "users.write"},
		{Name: "audit.read"},
		{Name: "settings.read"},
		{Name: "settings.write"},
		{Name: "api_tokens.read"},
		{Name: "api_tokens.write"},
	}

	roleMap := map[string]Role{}
	for _, name := range roleDefinitions {
		role, err := s.repo.EnsureRole(ctx, name)
		if err != nil {
			return err
		}
		roleMap[name] = role
	}

	permMap := map[string]Permission{}
	for _, definition := range permissionDefinitions {
		perm, err := s.repo.EnsurePermission(ctx, definition.Name, definition.ModuleScope)
		if err != nil {
			return err
		}
		permMap[definition.Name] = perm
	}

	mappings := map[string][]string{
		"Admin":   {"users.read", "users.write", "audit.read", "settings.read", "settings.write", "api_tokens.read", "api_tokens.write"},
		"Manager": {"users.read", "audit.read", "settings.read"},
		"Staff":   {"settings.read"},
		"Viewer":  {"users.read", "audit.read", "settings.read"},
	}

	for roleName, perms := range mappings {
		role := roleMap[roleName]
		for _, permName := range perms {
			if err := s.repo.EnsureRolePermission(ctx, role.ID, permMap[permName].ID); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Service) AssignRoleToUser(ctx context.Context, userID int64, roleName string) error {
	role, err := s.repo.GetRoleByName(ctx, roleName)
	if err != nil {
		return err
	}
	if err := s.repo.EnsureUserRole(ctx, userID, role.ID); err != nil {
		return err
	}
	s.InvalidateUser(userID)
	return nil
}

func (s *Service) InvalidateUser(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cache, userID)
}

func (s *Service) readCache(userID int64) (cachedPermissions, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.cache[userID]
	return entry, ok
}

func (s *Service) writeCache(userID int64, value cachedPermissions) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache[userID] = value
}
