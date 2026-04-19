package rbac

import (
	"context"
	"testing"
)

type fakeRepository struct {
	rolesByUser map[int64][]Role
	permsByUser map[int64][]Permission
	roleSeq      int64
	permSeq      int64
	rolesByName  map[string]Role
	permsByName  map[string]Permission
	rolePerms    map[string]map[string]struct{}
}

func (f *fakeRepository) GetUserRoles(_ context.Context, userID int64) ([]Role, error) {
	return append([]Role{}, f.rolesByUser[userID]...), nil
}

func (f *fakeRepository) GetUserPermissions(_ context.Context, userID int64) ([]Permission, error) {
	return append([]Permission{}, f.permsByUser[userID]...), nil
}

func (f *fakeRepository) EnsureRole(_ context.Context, name string) (Role, error) {
	if f.rolesByName == nil {
		f.rolesByName = map[string]Role{}
	}
	if role, ok := f.rolesByName[name]; ok {
		return role, nil
	}
	f.roleSeq++
	role := Role{ID: f.roleSeq, Name: name}
	f.rolesByName[name] = role
	return role, nil
}

func (f *fakeRepository) EnsurePermission(_ context.Context, name string, moduleScope *string) (Permission, error) {
	if f.permsByName == nil {
		f.permsByName = map[string]Permission{}
	}
	if perm, ok := f.permsByName[name]; ok {
		return perm, nil
	}
	f.permSeq++
	perm := Permission{ID: f.permSeq, Name: name, ModuleScope: moduleScope}
	f.permsByName[name] = perm
	return perm, nil
}

func (f *fakeRepository) EnsureRolePermission(_ context.Context, roleID, permissionID int64) error {
	if f.rolePerms == nil {
		f.rolePerms = map[string]map[string]struct{}{}
	}
	var roleName string
	for name, role := range f.rolesByName {
		if role.ID == roleID {
			roleName = name
			break
		}
	}
	var permName string
	for name, perm := range f.permsByName {
		if perm.ID == permissionID {
			permName = name
			break
		}
	}
	if roleName == "" || permName == "" {
		return nil
	}
	if f.rolePerms[roleName] == nil {
		f.rolePerms[roleName] = map[string]struct{}{}
	}
	f.rolePerms[roleName][permName] = struct{}{}
	return nil
}

func (f *fakeRepository) EnsureUserRole(context.Context, int64, int64) error {
	return nil
}

func (f *fakeRepository) GetRoleByName(_ context.Context, name string) (Role, error) {
	return Role{ID: 1, Name: name}, nil
}

func (f *fakeRepository) ReplaceUserRoles(context.Context, int64, []int64) error {
	return nil
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

func TestEnsureBaseRBACSeedsSchedulerPermissionsForDefaultRoles(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo)

	if err := service.EnsureBaseRBAC(context.Background()); err != nil {
		t.Fatalf("ensure base rbac: %v", err)
	}

	assertRoleHasPermission := func(roleName, permissionName string) {
		t.Helper()
		perms := repo.rolePerms[roleName]
		if perms == nil {
			t.Fatalf("expected seeded permissions for role %q", roleName)
		}
		if _, ok := perms[permissionName]; !ok {
			t.Fatalf("expected role %q to include permission %q; got %v", roleName, permissionName, perms)
		}
	}

	assertRoleHasPermission("Admin", PermissionSchedulerRead)
	assertRoleHasPermission("Admin", PermissionSchedulerWrite)
	assertRoleHasPermission("Manager", PermissionSchedulerRead)
	assertRoleHasPermission("Viewer", PermissionSchedulerRead)
}
