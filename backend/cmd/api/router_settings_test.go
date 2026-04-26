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
	values map[string][]byte
}

type fakeRapidProPreviewProvider struct{}

func (f *fakeSettingsRepo) Get(_ context.Context, category string, key string) (json.RawMessage, error) {
	if len(f.values) == 0 {
		return nil, settings.ErrNotFound
	}
	if value, ok := f.values[category+"::"+key]; ok {
		return append(json.RawMessage{}, value...), nil
	}
	return nil, settings.ErrNotFound
}

func (f *fakeSettingsRepo) Upsert(_ context.Context, category string, key string, value json.RawMessage, _ *int64, _ time.Time) error {
	if f.values == nil {
		f.values = map[string][]byte{}
	}
	f.values[category+"::"+key] = append([]byte{}, value...)
	return nil
}

func (fakeRapidProPreviewProvider) ListRapidProReporterSyncPreviewReporters(context.Context) ([]settings.RapidProReporterOption, error) {
	return []settings.RapidProReporterOption{{ID: 11, Name: "Alice Reporter"}}, nil
}

func (fakeRapidProPreviewProvider) BuildRapidProReporterSyncPreview(context.Context, int64) (settings.RapidProReporterSyncPreview, error) {
	return settings.RapidProReporterSyncPreview{
		Reporter: settings.RapidProReporterSyncPreviewReporter{
			ID:            11,
			Name:          "Alice Reporter",
			Telephone:     "+256700000001",
			SyncOperation: "created",
		},
		RequestPath:  "/api/v2/contacts.json",
		RequestQuery: map[string]string{},
		RequestBody: map[string]any{
			"name": "Alice Reporter",
		},
	}, nil
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

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings/login-branding", strings.NewReader(`{"applicationDisplayName":"RapidEx"}`))
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

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings/login-branding", strings.NewReader(`{"applicationDisplayName":"RapidEx Custom","loginImageUrl":"https://cdn.example.com/logo.png"}`))
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

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings/login-branding", strings.NewReader(`{"applicationDisplayName":"RapidEx Admin"}`))
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

func TestRapidProReporterSyncReadRouteAcceptsSettingsReader(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(201, "reader", time.Now().UTC())
	rbacService := rbacServiceWithPermissions(map[int64][]string{
		201: []string{"settings.read"},
	})
	handler := settings.NewHandler(settings.NewService(&fakeSettingsRepo{}, nil))
	router := newRouter(AppDeps{
		JWTManager:      jwt,
		RBACService:     rbacService,
		SettingsHandler: handler,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings/rapidpro-reporter-sync", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "rapidProServerCode") {
		t.Fatalf("expected rapidpro sync settings payload, got %s", w.Body.String())
	}
}

func TestRapidProReporterSyncPreviewRoutesAcceptSettingsReader(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(202, "reader", time.Now().UTC())
	rbacService := rbacServiceWithPermissions(map[int64][]string{
		202: []string{"settings.read"},
	})
	handler := settings.NewHandler(
		settings.NewService(&fakeSettingsRepo{}, nil).
			WithRapidProPreviewProvider(fakeRapidProPreviewProvider{}),
	)
	router := newRouter(AppDeps{
		JWTManager:      jwt,
		RBACService:     rbacService,
		SettingsHandler: handler,
	})

	reportersReq := httptest.NewRequest(http.MethodGet, "/api/v1/settings/rapidpro-reporter-sync/preview-reporters", nil)
	reportersReq.Header.Set("Authorization", "Bearer "+token)
	reportersRes := httptest.NewRecorder()
	router.ServeHTTP(reportersRes, reportersReq)
	if reportersRes.Code != http.StatusOK {
		t.Fatalf("expected 200 from preview reporters, got %d body=%s", reportersRes.Code, reportersRes.Body.String())
	}
	if !strings.Contains(reportersRes.Body.String(), "Alice Reporter") {
		t.Fatalf("expected preview reporters payload, got %s", reportersRes.Body.String())
	}

	previewReq := httptest.NewRequest(http.MethodGet, "/api/v1/settings/rapidpro-reporter-sync/preview?reporterId=11", nil)
	previewReq.Header.Set("Authorization", "Bearer "+token)
	previewRes := httptest.NewRecorder()
	router.ServeHTTP(previewRes, previewReq)
	if previewRes.Code != http.StatusOK {
		t.Fatalf("expected 200 from preview, got %d body=%s", previewRes.Code, previewRes.Body.String())
	}
	if !strings.Contains(previewRes.Body.String(), "contacts.json") {
		t.Fatalf("expected preview payload, got %s", previewRes.Body.String())
	}
}

func TestRapidexWebhookMappingsReadRouteAcceptsSettingsReader(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(203, "reader", time.Now().UTC())
	rbacService := rbacServiceWithPermissions(map[int64][]string{
		203: {"settings.read"},
	})
	repo := &fakeSettingsRepo{
		values: map[string][]byte{
			"rapidex::webhook_mappings": []byte(`{"mappings":[{"flowUuid":"flow-a","dataset":"DATASET_A","orgUnitVar":"facility_code","periodVar":"reporting_period","mappings":[{"field":"indicator_one","dataElement":"DE_1"}]}]}`),
		},
	}
	handler := settings.NewHandler(settings.NewService(repo, nil))
	router := newRouter(AppDeps{
		JWTManager:      jwt,
		RBACService:     rbacService,
		SettingsHandler: handler,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings/rapidex-webhook-mappings", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "flow-a") {
		t.Fatalf("expected rapidex mapping payload, got %s", w.Body.String())
	}
}

func TestRapidexWebhookMappingsUpdateRouteAcceptsSettingsWriter(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(204, "writer", time.Now().UTC())
	rbacService := rbacServiceWithPermissions(map[int64][]string{
		204: {"settings.write"},
	})
	handler := settings.NewHandler(settings.NewService(&fakeSettingsRepo{}, nil))
	router := newRouter(AppDeps{
		JWTManager:      jwt,
		RBACService:     rbacService,
		SettingsHandler: handler,
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings/rapidex-webhook-mappings", strings.NewReader(`{"mappings":[{"flowUuid":"flow-a","flowName":"Weekly Report","dataset":"DATASET_A","orgUnitVar":"facility_code","periodVar":"reporting_period","mappings":[{"field":"indicator_one","dataElement":"DE_1"}]}]}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Weekly Report") {
		t.Fatalf("expected updated rapidex mapping payload, got %s", w.Body.String())
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
