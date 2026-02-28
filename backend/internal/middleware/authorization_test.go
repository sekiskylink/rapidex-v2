package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"basepro/backend/internal/auth"
	"basepro/backend/internal/rbac"
	"github.com/gin-gonic/gin"
)

type fakeRBACRepo struct {
	rolesByUser map[int64][]rbac.Role
	permsByUser map[int64][]rbac.Permission
}

func (f *fakeRBACRepo) GetUserRoles(_ context.Context, userID int64) ([]rbac.Role, error) {
	return append([]rbac.Role{}, f.rolesByUser[userID]...), nil
}

func (f *fakeRBACRepo) GetUserPermissions(_ context.Context, userID int64) ([]rbac.Permission, error) {
	return append([]rbac.Permission{}, f.permsByUser[userID]...), nil
}

func (f *fakeRBACRepo) EnsureRole(context.Context, string) (rbac.Role, error) {
	return rbac.Role{}, nil
}
func (f *fakeRBACRepo) EnsurePermission(context.Context, string, *string) (rbac.Permission, error) {
	return rbac.Permission{}, nil
}
func (f *fakeRBACRepo) EnsureRolePermission(context.Context, int64, int64) error { return nil }
func (f *fakeRBACRepo) EnsureUserRole(context.Context, int64, int64) error       { return nil }
func (f *fakeRBACRepo) GetRoleByName(context.Context, string) (rbac.Role, error) {
	return rbac.Role{}, nil
}

func TestJWTUserWithoutPermissionGetsForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(22, "alice", time.Now().UTC())

	rbacService := rbac.NewService(&fakeRBACRepo{
		rolesByUser: map[int64][]rbac.Role{22: []rbac.Role{{ID: 1, Name: "Viewer"}}},
		permsByUser: map[int64][]rbac.Permission{22: []rbac.Permission{{ID: 1, Name: "audit.read"}}},
	})

	r := gin.New()
	r.GET("/users", JWTAuth(jwt), RequirePermission(rbacService, "users.read"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
	var body map[string]map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["error"]["code"] != "AUTH_FORBIDDEN" {
		t.Fatalf("expected AUTH_FORBIDDEN, got %q", body["error"]["code"])
	}
}

func TestJWTUserWithPermissionGetsAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(23, "bob", time.Now().UTC())

	rbacService := rbac.NewService(&fakeRBACRepo{
		rolesByUser: map[int64][]rbac.Role{23: []rbac.Role{{ID: 2, Name: "Manager"}}},
		permsByUser: map[int64][]rbac.Permission{23: []rbac.Permission{{ID: 2, Name: "users.read"}}},
	})

	r := gin.New()
	r.GET("/users", JWTAuth(jwt), RequirePermission(rbacService, "users.read"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRequireAuthReturnsUnauthorizedShape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/audit", RequireAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/audit", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	var body map[string]map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["error"]["code"] != "AUTH_UNAUTHORIZED" {
		t.Fatalf("expected AUTH_UNAUTHORIZED, got %q", body["error"]["code"])
	}
}
