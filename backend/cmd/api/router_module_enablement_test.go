package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"basepro/backend/internal/moduleenablement"
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
	router := newRouter(AppDeps{
		ModuleFlagsHandler: moduleenablement.NewHandler(func() map[string]bool {
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

	if len(body.Modules) != 3 {
		t.Fatalf("expected 3 modules, got %d", len(body.Modules))
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
