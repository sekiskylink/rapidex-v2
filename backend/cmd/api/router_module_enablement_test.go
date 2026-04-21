package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"basepro/backend/internal/auth"
	"basepro/backend/internal/moduleenablement"
	"basepro/backend/internal/rbac"
	"basepro/backend/internal/settings"
)

type modulesEffectiveResponse struct {
	Modules []moduleenablement.EffectiveModule `json:"modules"`
}

func TestSettingsPublicRouteBlockedWhenSettingsModuleDisabled(t *testing.T) {
	router := newRouter(AppDeps{
		SettingsHandler: settings.NewHandler(settings.NewService(&fakeSettingsRepo{}, nil)),
		ModuleFlagsProvider: func() map[string]bool {
			return map[string]bool{
				"modules.settings.enabled": false,
			}
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings/public/login-branding", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when settings module is disabled, got %d body=%s", w.Code, w.Body.String())
	}

	var body map[string]map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["error"]["code"] != "MODULE_DISABLED" {
		t.Fatalf("expected MODULE_DISABLED code, got %v", body["error"]["code"])
	}
}

func TestModulesEffectiveEndpointReturnsResolvedModuleFlags(t *testing.T) {
	moduleService := moduleenablement.NewService(&fakeSettingsRepo{}, nil)
	router := newRouter(AppDeps{
		ModuleFlagsHandler: moduleenablement.NewHandler(moduleService, func() map[string]bool {
			return map[string]bool{
				"modules.settings.enabled": false,
			}
		}),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/modules/effective", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var body modulesEffectiveResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(body.Modules) != 12 {
		t.Fatalf("expected 12 modules, got %d", len(body.Modules))
	}

	var settingsItem *moduleenablement.EffectiveModule
	for i := range body.Modules {
		if body.Modules[i].ModuleID == "settings" {
			settingsItem = &body.Modules[i]
			break
		}
	}
	if settingsItem == nil {
		t.Fatal("expected settings module in payload")
	}
	if settingsItem.Enabled {
		t.Fatalf("expected settings module disabled by config override: %+v", *settingsItem)
	}
	if settingsItem.Source != "config" {
		t.Fatalf("expected settings module source to be config, got %q", settingsItem.Source)
	}
}

func TestSettingsModuleEnablementUpdateRouteRequiresWriterPermission(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(55, "reader", time.Now().UTC())
	rbacService := rbacServiceWithPermissions(map[int64][]string{
		55: []string{"settings.read"},
	})
	moduleService := moduleenablement.NewService(&fakeSettingsRepo{}, nil)
	router := newRouter(AppDeps{
		JWTManager:         jwt,
		RBACService:        rbacService,
		SettingsHandler:    settings.NewHandler(settings.NewService(&fakeSettingsRepo{}, nil)),
		ModuleFlagsHandler: moduleenablement.NewHandler(moduleService, nil),
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings/module-enablement", strings.NewReader(`{"modules":[{"moduleId":"administration","enabled":false}]}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestSettingsModuleEnablementUpdateAcceptsAuthorizedWriter(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(56, "writer", time.Now().UTC())
	rbacService := rbacServiceWithPermissions(map[int64][]string{
		56: []string{"settings.read", "settings.write"},
	})
	moduleService := moduleenablement.NewService(&fakeSettingsRepo{}, nil)
	router := newRouter(AppDeps{
		JWTManager:         jwt,
		RBACService:        rbacService,
		SettingsHandler:    settings.NewHandler(settings.NewService(&fakeSettingsRepo{}, nil)),
		ModuleFlagsHandler: moduleenablement.NewHandler(moduleService, nil),
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings/module-enablement", strings.NewReader(`{"modules":[{"moduleId":"administration","enabled":false}]}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var body modulesEffectiveResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	var administrationItem *moduleenablement.EffectiveModule
	for i := range body.Modules {
		if body.Modules[i].ModuleID == "administration" {
			administrationItem = &body.Modules[i]
			break
		}
	}
	if administrationItem == nil {
		t.Fatal("expected administration module in payload")
	}
	if administrationItem.Enabled {
		t.Fatalf("expected administration module disabled after update: %+v", *administrationItem)
	}
	if administrationItem.Source != "runtime" {
		t.Fatalf("expected runtime source, got %s", administrationItem.Source)
	}
}

func TestSettingsModuleEnablementUpdateAcceptsAdminWithoutExplicitSettingsWrite(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(57, "admin", time.Now().UTC())
	rbacService := rbac.NewService(&fakeRBACRepo{
		rolesByUser: map[int64][]rbac.Role{
			57: {{ID: 1, Name: "Admin"}},
		},
		permsByUser: map[int64][]rbac.Permission{
			57: {{ID: 1, Name: "settings.read"}},
		},
	})
	moduleService := moduleenablement.NewService(&fakeSettingsRepo{}, nil)
	router := newRouter(AppDeps{
		JWTManager:         jwt,
		RBACService:        rbacService,
		SettingsHandler:    settings.NewHandler(settings.NewService(&fakeSettingsRepo{}, nil)),
		ModuleFlagsHandler: moduleenablement.NewHandler(moduleService, nil),
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings/module-enablement", strings.NewReader(`{"modules":[{"moduleId":"administration","enabled":false}]}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}
