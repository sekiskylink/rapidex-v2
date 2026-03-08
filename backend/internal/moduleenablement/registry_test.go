package moduleenablement

import "testing"

func TestResolveEffectiveUsesDefaultsAndConfigOverrides(t *testing.T) {
	items := ResolveEffective(map[string]bool{
		"modules.settings.enabled": false,
	})

	if len(items) != 3 {
		t.Fatalf("expected 3 module definitions, got %d", len(items))
	}

	if items[0].ModuleID != "dashboard" || !items[0].Enabled || items[0].Source != "default" {
		t.Fatalf("unexpected dashboard effective config: %+v", items[0])
	}

	if items[2].ModuleID != "settings" || items[2].Enabled || items[2].Source != "config" {
		t.Fatalf("expected settings module to be disabled by config override: %+v", items[2])
	}
}

func TestValidateOverridesRejectsUnknownKeys(t *testing.T) {
	err := ValidateOverrides(map[string]bool{
		"modules.unknown.enabled": false,
	})
	if err == nil {
		t.Fatal("expected unknown key validation error")
	}

	if err := ValidateOverrides(map[string]bool{
		"modules.dashboard.enabled": true,
	}); err != nil {
		t.Fatalf("expected known key to pass validation: %v", err)
	}
}
