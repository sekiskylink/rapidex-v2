package users

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"basepro/backend/internal/audit"
	"basepro/backend/internal/rbac"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestListUsersPaginationIncludesTotalAndOffset(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()

	repo := NewSQLRepository(sqlx.NewDb(sqlDB, "sqlmock"))
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM users WHERE username ILIKE $1`)).
		WithArgs("%ali%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, username, is_active, created_at, updated_at
		FROM users
	 WHERE username ILIKE $3 ORDER BY created_at DESC LIMIT $1 OFFSET $2`)).
		WithArgs(2, 2, "%ali%").
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "username", "is_active", "created_at", "updated_at"}).
				AddRow(int64(3), "alice", true, now, now).
				AddRow(int64(4), "alina", true, now, now),
		)

	result, err := repo.ListUsers(context.Background(), ListQuery{
		Page:      2,
		PageSize:  2,
		SortField: "created_at",
		SortOrder: "desc",
		Filter:    "ali",
	})
	if err != nil {
		t.Fatalf("list users: %v", err)
	}

	if result.Total != 5 {
		t.Fatalf("expected total=5, got %d", result.Total)
	}
	if result.Page != 2 || result.PageSize != 2 {
		t.Fatalf("expected page metadata 2/2, got %d/%d", result.Page, result.PageSize)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

type fakeRepo struct {
	created UserRecord
}

func (f *fakeRepo) ListUsers(context.Context, ListQuery) (ListResult, error) {
	return ListResult{}, nil
}

func (f *fakeRepo) CreateUser(_ context.Context, username, _ string, isActive bool) (UserRecord, error) {
	f.created = UserRecord{
		ID:        100,
		Username:  username,
		IsActive:  isActive,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	return f.created, nil
}

func (f *fakeRepo) SetUserActive(context.Context, int64, bool) error {
	return nil
}

func (f *fakeRepo) SetUsername(context.Context, int64, string) error {
	return nil
}

func (f *fakeRepo) SetPassword(context.Context, int64, string) error {
	return nil
}

type fakeRBACRepo struct {
	rolesByUser map[int64][]rbac.Role
}

func (f *fakeRBACRepo) GetUserRoles(_ context.Context, userID int64) ([]rbac.Role, error) {
	return append([]rbac.Role{}, f.rolesByUser[userID]...), nil
}

func (f *fakeRBACRepo) GetUserPermissions(context.Context, int64) ([]rbac.Permission, error) {
	return []rbac.Permission{}, nil
}

func (f *fakeRBACRepo) EnsureRole(context.Context, string) (rbac.Role, error) {
	return rbac.Role{}, nil
}

func (f *fakeRBACRepo) EnsurePermission(context.Context, string, *string) (rbac.Permission, error) {
	return rbac.Permission{}, nil
}

func (f *fakeRBACRepo) EnsureRolePermission(context.Context, int64, int64) error {
	return nil
}

func (f *fakeRBACRepo) EnsureUserRole(context.Context, int64, int64) error {
	return nil
}

func (f *fakeRBACRepo) GetRoleByName(_ context.Context, name string) (rbac.Role, error) {
	switch name {
	case "Viewer":
		return rbac.Role{ID: 4, Name: "Viewer"}, nil
	default:
		return rbac.Role{}, sql.ErrNoRows
	}
}

func (f *fakeRBACRepo) ReplaceUserRoles(_ context.Context, userID int64, roleIDs []int64) error {
	roles := make([]rbac.Role, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		if roleID == 4 {
			roles = append(roles, rbac.Role{ID: 4, Name: "Viewer"})
		}
	}
	f.rolesByUser[userID] = roles
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

func TestCreateUserWritesAuditLog(t *testing.T) {
	repo := &fakeRepo{}
	rbacService := rbac.NewService(&fakeRBACRepo{
		rolesByUser: map[int64][]rbac.Role{
			100: []rbac.Role{{ID: 4, Name: "Viewer"}},
		},
	})
	auditRepo := &fakeAuditRepo{}
	service := NewService(repo, rbacService, audit.NewService(auditRepo), 4)

	actorID := int64(1)
	_, err := service.CreateUser(context.Background(), CreateInput{
		Username: "newuser",
		Password: "TempPass123!",
		IsActive: true,
		Roles:    []string{"Viewer"},
		ActorID:  &actorID,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	if len(auditRepo.events) == 0 {
		t.Fatalf("expected at least one audit event")
	}
	if auditRepo.events[0].Action != "users.create" {
		t.Fatalf("expected users.create audit action, got %s", auditRepo.events[0].Action)
	}
}
