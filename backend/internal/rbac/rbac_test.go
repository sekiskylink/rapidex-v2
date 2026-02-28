package rbac

import (
	"context"
	"testing"
)

type fakeRepository struct {
	rolesByUser map[int64][]Role
	permsByUser map[int64][]Permission
}

func (f *fakeRepository) GetUserRoles(_ context.Context, userID int64) ([]Role, error) {
	return append([]Role{}, f.rolesByUser[userID]...), nil
}

func (f *fakeRepository) GetUserPermissions(_ context.Context, userID int64) ([]Permission, error) {
	return append([]Permission{}, f.permsByUser[userID]...), nil
}

func (f *fakeRepository) EnsureRole(_ context.Context, name string) (Role, error) {
	return Role{Name: name}, nil
}

func (f *fakeRepository) EnsurePermission(_ context.Context, name string, moduleScope *string) (Permission, error) {
	return Permission{Name: name, ModuleScope: moduleScope}, nil
}

func (f *fakeRepository) EnsureRolePermission(context.Context, int64, int64) error {
	return nil
}

func (f *fakeRepository) EnsureUserRole(context.Context, int64, int64) error {
	return nil
}

func (f *fakeRepository) GetRoleByName(_ context.Context, name string) (Role, error) {
	return Role{ID: 1, Name: name}, nil
}

func TestPermissionsForUserIncludesAssignedPermission(t *testing.T) {
	repo := &fakeRepository{
		rolesByUser: map[int64][]Role{10: []Role{{ID: 1, Name: "Manager"}}},
		permsByUser: map[int64][]Permission{10: []Permission{{ID: 7, Name: "users.read"}}},
	}
	service := NewService(repo)

	perms, err := service.PermissionsForUser(context.Background(), 10)
	if err != nil {
		t.Fatalf("permissions for user: %v", err)
	}

	if len(perms) != 1 || perms[0] != "users.read" {
		t.Fatalf("expected [users.read], got %v", perms)
	}
}
