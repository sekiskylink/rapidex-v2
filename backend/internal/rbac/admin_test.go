package rbac

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/audit"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeAdminRepo struct {
	rolesByID           map[int64]RoleRecord
	permissionsByRoleID map[int64][]PermissionRecord
	roleUsersByID       map[int64][]RoleUserRecord
	permissionsByName   map[string]PermissionRecord
	nextRoleID          int64
}

func newFakeAdminRepo() *fakeAdminRepo {
	now := time.Now().UTC()
	viewer := PermissionRecord{ID: 10, Name: "users.read", CreatedAt: now}
	editor := PermissionRecord{ID: 11, Name: "users.write", CreatedAt: now}
	return &fakeAdminRepo{
		rolesByID: map[int64]RoleRecord{
			1: {ID: 1, Name: "Viewer", CreatedAt: now, UpdatedAt: now},
		},
		permissionsByRoleID: map[int64][]PermissionRecord{
			1: {viewer},
		},
		roleUsersByID: map[int64][]RoleUserRecord{
			1: {{ID: 50, Username: "alice", IsActive: true}},
		},
		permissionsByName: map[string]PermissionRecord{
			"users.read":  viewer,
			"users.write": editor,
		},
		nextRoleID: 1,
	}
}

func (f *fakeAdminRepo) ListRoles(_ context.Context, query RoleListQuery) (RoleListResult, error) {
	items := []RoleSummary{}
	for _, role := range f.rolesByID {
		if query.Filter != "" && !strings.Contains(strings.ToLower(role.Name), strings.ToLower(query.Filter)) {
			continue
		}
		items = append(items, RoleSummary{
			RoleRecord:      role,
			PermissionCount: len(f.permissionsByRoleID[role.ID]),
			UserCount:       len(f.roleUsersByID[role.ID]),
		})
	}
	return RoleListResult{Items: items, Total: len(items), Page: query.Page, PageSize: query.PageSize}, nil
}

func (f *fakeAdminRepo) CreateRole(_ context.Context, name string) (RoleRecord, error) {
	for _, role := range f.rolesByID {
		if role.Name == name {
			return RoleRecord{}, &pgconn.PgError{Code: "23505", ConstraintName: "roles_name_key"}
		}
	}
	f.nextRoleID++
	now := time.Now().UTC()
	role := RoleRecord{ID: f.nextRoleID, Name: name, CreatedAt: now, UpdatedAt: now}
	f.rolesByID[role.ID] = role
	return role, nil
}

func (f *fakeAdminRepo) UpdateRoleName(_ context.Context, roleID int64, name string) (RoleRecord, error) {
	role, ok := f.rolesByID[roleID]
	if !ok {
		return RoleRecord{}, sql.ErrNoRows
	}
	role.Name = name
	role.UpdatedAt = time.Now().UTC()
	f.rolesByID[roleID] = role
	return role, nil
}

func (f *fakeAdminRepo) GetRoleByID(_ context.Context, roleID int64) (RoleRecord, error) {
	role, ok := f.rolesByID[roleID]
	if !ok {
		return RoleRecord{}, sql.ErrNoRows
	}
	return role, nil
}

func (f *fakeAdminRepo) ListRolePermissions(_ context.Context, roleID int64) ([]PermissionRecord, error) {
	return append([]PermissionRecord{}, f.permissionsByRoleID[roleID]...), nil
}

func (f *fakeAdminRepo) ListRoleUsers(_ context.Context, roleID int64) ([]RoleUserRecord, error) {
	return append([]RoleUserRecord{}, f.roleUsersByID[roleID]...), nil
}

func (f *fakeAdminRepo) ListPermissions(_ context.Context, query PermissionListQuery) (PermissionListResult, error) {
	items := []PermissionRecord{}
	allowed := map[string]struct{}{}
	if query.FilterToAllowed {
		for _, name := range query.AllowedNames {
			allowed[name] = struct{}{}
		}
	}
	for _, permission := range f.permissionsByName {
		if query.FilterToAllowed {
			if _, ok := allowed[permission.Name]; !ok {
				continue
			}
		}
		if query.Query != "" && !strings.Contains(strings.ToLower(permission.Name), strings.ToLower(query.Query)) {
			continue
		}
		items = append(items, permission)
	}
	return PermissionListResult{Items: items, Total: len(items), Page: query.Page, PageSize: query.PageSize}, nil
}

func (f *fakeAdminRepo) GetPermissionsByNames(_ context.Context, names []string) ([]PermissionRecord, error) {
	items := make([]PermissionRecord, 0, len(names))
	for _, name := range names {
		permission, ok := f.permissionsByName[name]
		if !ok {
			continue
		}
		items = append(items, permission)
	}
	return items, nil
}

func (f *fakeAdminRepo) ReplaceRolePermissions(_ context.Context, roleID int64, permissionIDs []int64) error {
	permissions := make([]PermissionRecord, 0, len(permissionIDs))
	for _, id := range permissionIDs {
		for _, permission := range f.permissionsByName {
			if permission.ID == id {
				permissions = append(permissions, permission)
			}
		}
	}
	f.permissionsByRoleID[roleID] = permissions
	return nil
}

type fakeAuditRepo struct {
	events []audit.Event
}

func (f *fakeAuditRepo) Insert(_ context.Context, event audit.Event) error {
	f.events = append(f.events, event)
	return nil
}

func (f *fakeAuditRepo) List(context.Context, audit.ListFilter) (audit.ListResult, error) {
	return audit.ListResult{}, nil
}

func TestAdminServiceCreateRoleLogsAudit(t *testing.T) {
	repo := newFakeAdminRepo()
	auditRepo := &fakeAuditRepo{}
	service := NewAdminService(repo, audit.NewService(auditRepo))

	actorID := int64(7)
	created, err := service.CreateRole(context.Background(), RoleCreateInput{
		Name:        "Operators",
		Permissions: []string{"users.read", "users.write"},
		ActorUserID: &actorID,
	})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}
	if created.Name != "Operators" {
		t.Fatalf("expected role name Operators, got %s", created.Name)
	}
	if len(created.Permissions) != 2 {
		t.Fatalf("expected 2 permissions, got %d", len(created.Permissions))
	}
	if len(auditRepo.events) != 1 || auditRepo.events[0].Action != "roles.create" {
		t.Fatalf("expected roles.create audit event")
	}
}

func TestAdminServiceCreateRoleRejectsInvalidPermission(t *testing.T) {
	repo := newFakeAdminRepo()
	service := NewAdminService(repo, audit.NewService(&fakeAuditRepo{}))

	_, err := service.CreateRole(context.Background(), RoleCreateInput{
		Name:        "Operators",
		Permissions: []string{"users.read", "unknown.permission"},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}

	var appErr *apperror.AppError
	if !errorsAs(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != apperror.CodeValidationFailed {
		t.Fatalf("expected validation code, got %s", appErr.Code)
	}
}

func TestAdminServiceGetRoleDetailIncludesUsersWhenRequested(t *testing.T) {
	repo := newFakeAdminRepo()
	service := NewAdminService(repo, audit.NewService(&fakeAuditRepo{}))

	result, err := service.GetRoleDetail(context.Background(), 1, true)
	if err != nil {
		t.Fatalf("get role detail: %v", err)
	}
	if len(result.Users) != 1 || result.Users[0].Username != "alice" {
		t.Fatalf("expected role users to be included")
	}
}

func TestAdminHandlerListPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newFakeAdminRepo()
	handler := NewAdminHandler(NewAdminService(repo, audit.NewService(&fakeAuditRepo{})))

	r := gin.New()
	r.GET("/permissions", handler.ListPermissions)
	req := httptest.NewRequest(http.MethodGet, "/permissions?page=1&pageSize=25&q=users.read", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	items, ok := body["items"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("expected list items in response")
	}
}

func TestAdminServiceListPermissionsFiltersDisabledModulePermissions(t *testing.T) {
	repo := newFakeAdminRepo()
	repo.permissionsByName["settings.read"] = PermissionRecord{ID: 12, Name: "settings.read", CreatedAt: time.Now().UTC()}
	service := NewAdminService(
		repo,
		audit.NewService(&fakeAuditRepo{}),
		WithPermissionEnablementFilter(func(permissionName string) bool {
			moduleID, ok := ModuleIDForPermission(permissionName)
			if !ok {
				return true
			}
			return moduleID != "administration"
		}),
	)

	result, err := service.ListPermissions(context.Background(), PermissionListQuery{
		Page:     1,
		PageSize: 25,
	})
	if err != nil {
		t.Fatalf("list permissions: %v", err)
	}

	for _, item := range result.Items {
		if item.Name == "users.read" || item.Name == "users.write" {
			t.Fatalf("expected administration permissions filtered out, got %s", item.Name)
		}
	}
	if len(result.Items) != 1 || result.Items[0].Name != "settings.read" {
		t.Fatalf("expected only settings.read visible, got %+v", result.Items)
	}
}

func TestAdminServiceRejectsDisabledModulePermissionsOnUpdate(t *testing.T) {
	repo := newFakeAdminRepo()
	service := NewAdminService(
		repo,
		audit.NewService(&fakeAuditRepo{}),
		WithPermissionEnablementFilter(func(permissionName string) bool {
			moduleID, ok := ModuleIDForPermission(permissionName)
			if !ok {
				return true
			}
			return moduleID != "administration"
		}),
	)

	_, err := service.UpdateRole(context.Background(), RoleUpdateInput{
		RoleID:      1,
		Permissions: &[]string{"users.read"},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}

	var appErr *apperror.AppError
	if !errorsAs(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != apperror.CodeValidationFailed {
		t.Fatalf("expected validation code, got %s", appErr.Code)
	}
}

func TestAdminHandlerListRolesSupportsQSearch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newFakeAdminRepo()
	handler := NewAdminHandler(NewAdminService(repo, audit.NewService(&fakeAuditRepo{})))

	r := gin.New()
	r.GET("/roles", handler.ListRoles)
	req := httptest.NewRequest(http.MethodGet, "/roles?page=1&pageSize=25&q=view", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	items, ok := body["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one filtered role, got %v", body["items"])
	}
}

func TestAdminHandlerListRolesRejectsInvalidSort(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newFakeAdminRepo()
	handler := NewAdminHandler(NewAdminService(repo, audit.NewService(&fakeAuditRepo{})))

	r := gin.New()
	r.GET("/roles", handler.ListRoles)
	req := httptest.NewRequest(http.MethodGet, "/roles?sort=name:sideways", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestAdminHandlerUpdateRolePermissionsValidatesRoleID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newFakeAdminRepo()
	handler := NewAdminHandler(NewAdminService(repo, audit.NewService(&fakeAuditRepo{})))

	r := gin.New()
	r.PUT("/roles/:id/permissions", handler.UpdateRolePermissions)

	payload := []byte(`{"permissions":["users.read"]}`)
	req := httptest.NewRequest(http.MethodPut, "/roles/bad/permissions", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAdminHandlerCreateRoleInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newFakeAdminRepo()
	handler := NewAdminHandler(NewAdminService(repo, audit.NewService(&fakeAuditRepo{})))

	r := gin.New()
	r.POST("/roles", handler.CreateRole)

	req := httptest.NewRequest(http.MethodPost, "/roles", bytes.NewReader([]byte(`{`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func errorsAs(err error, target **apperror.AppError) bool {
	if err == nil {
		return false
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		return false
	}
	*target = appErr
	return true
}
