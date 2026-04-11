package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"basepro/backend/internal/auth"
	"basepro/backend/internal/rbac"
	"basepro/backend/internal/settings"
)

type fakeSettingsRepo struct {
	value []byte
}

func (f *fakeSettingsRepo) Get(context.Context, string, string) (json.RawMessage, error) {
	if len(f.value) == 0 {
		return nil, settings.ErrNotFound
	}
	return append(json.RawMessage{}, f.value...), nil
}

func (f *fakeSettingsRepo) Upsert(_ context.Context, _ string, _ string, value json.RawMessage, _ *int64, _ time.Time) error {
	f.value = append([]byte{}, value...)
	return nil
}

func TestLoginBrandingPublicRouteAccessibleWithoutAuth(t *testing.T) {
	handler := settings.NewHandler(settings.NewService(&fakeSettingsRepo{}, nil))
	router := newRouter(AppDeps{SettingsHandler: handler})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings/public/login-branding", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "applicationDisplayName") {
		t.Fatalf("expected branding payload, got %s", w.Body.String())
	}
}

func TestLoginBrandingUpdateRouteRequiresAuth(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	rbacService := rbacServiceWithPermissions(map[int64][]string{})
	handler := settings.NewHandler(settings.NewService(&fakeSettingsRepo{}, nil))
	router := newRouter(AppDeps{
		JWTManager:      jwt,
		RBACService:     rbacService,
		SettingsHandler: handler,
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings/login-branding", strings.NewReader(`{"applicationDisplayName":"BasePro"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestLoginBrandingUpdateRouteAcceptsAuthorizedWriter(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(99, "writer", time.Now().UTC())
	rbacService := rbacServiceWithPermissions(map[int64][]string{
		99: []string{"settings.write"},
	})
	handler := settings.NewHandler(settings.NewService(&fakeSettingsRepo{}, nil))
	router := newRouter(AppDeps{
		JWTManager:      jwt,
		RBACService:     rbacService,
		SettingsHandler: handler,
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings/login-branding", strings.NewReader(`{"applicationDisplayName":"BasePro Custom","loginImageUrl":"https://cdn.example.com/logo.png"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestLoginBrandingUpdateRouteAcceptsAdminWithoutExplicitSettingsWrite(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(100, "admin", time.Now().UTC())
	rbacService := rbac.NewService(&fakeRBACRepo{
		rolesByUser: map[int64][]rbac.Role{
			100: {{ID: 1, Name: "Admin"}},
		},
		permsByUser: map[int64][]rbac.Permission{
			100: {{ID: 1, Name: "settings.read"}},
		},
	})
	handler := settings.NewHandler(settings.NewService(&fakeSettingsRepo{}, nil))
	router := newRouter(AppDeps{
		JWTManager:      jwt,
		RBACService:     rbacService,
		SettingsHandler: handler,
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings/login-branding", strings.NewReader(`{"applicationDisplayName":"BasePro Admin"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestRuntimeConfigRouteRequiresAuth(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	rbacService := rbacServiceWithPermissions(map[int64][]string{})
	handler := settings.NewHandler(settings.NewService(&fakeSettingsRepo{}, nil))
	router := newRouter(AppDeps{
		JWTManager:      jwt,
		RBACService:     rbacService,
		SettingsHandler: handler,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings/runtime-config", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestRuntimeConfigRouteAcceptsSettingsReader(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(101, "reader", time.Now().UTC())
	rbacService := rbacServiceWithPermissions(map[int64][]string{
		101: []string{"settings.read"},
	})
	handler := settings.NewHandler(
		settings.NewService(&fakeSettingsRepo{}, nil).WithRuntimeConfigProvider(func() map[string]any {
			return map[string]any{
				"database": map[string]any{
					"dsn": "postgres://reader:reader-secret@127.0.0.1:5432/basepro?sslmode=disable",
				},
			}
		}),
	)
	router := newRouter(AppDeps{
		JWTManager:      jwt,
		RBACService:     rbacService,
		SettingsHandler: handler,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings/runtime-config", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `"config"`) {
		t.Fatalf("expected config envelope, got %s", body)
	}
	if strings.Contains(body, "reader-secret") {
		t.Fatalf("expected runtime config to mask dsn credentials, got %s", body)
	}
}

func rbacServiceWithPermissions(perms map[int64][]string) *rbac.Service {
	roleMap := map[int64][]rbac.Role{}
	permMap := map[int64][]rbac.Permission{}
	for userID, names := range perms {
		roleMap[userID] = []rbac.Role{{ID: 1, Name: "Custom"}}
		entries := make([]rbac.Permission, 0, len(names))
		for idx, name := range names {
			entries = append(entries, rbac.Permission{ID: int64(idx + 1), Name: name})
		}
		permMap[userID] = entries
	}
	return rbac.NewService(&fakeRBACRepo{rolesByUser: roleMap, permsByUser: permMap})
}
