package moduleenablement

import "testing"

func TestResolveEffectiveUsesDefaultsAndConfigOverrides(t *testing.T) {
	items := ResolveEffective(
		map[string]bool{"modules.settings.enabled": false},
		map[string]bool{"modules.administration.enabled": false},
	)

	if len(items) != 12 {
		t.Fatalf("expected 12 module definitions, got %d", len(items))
	}

	if items[0].ModuleID != "dashboard" || !items[0].Enabled || items[0].Source != "default" || items[0].Editable {
		t.Fatalf("unexpected dashboard effective config: %+v", items[0])
	}

	if items[1].ModuleID != "administration" || items[1].Enabled || items[1].Source != "runtime" || !items[1].Editable {
		t.Fatalf("expected administration module to be disabled by runtime override: %+v", items[1])
	}

	if items[2].ModuleID != "settings" || items[2].Enabled || items[2].Source != "config" || items[2].Editable {
		t.Fatalf("expected settings module to be disabled by config override: %+v", items[2])
	}

	for _, moduleID := range []string{"servers", "requests", "deliveries", "jobs", "scheduler", "observability", "documentation", "orgunits", "reporters"} {
		found := false
		for _, item := range items {
			if item.ModuleID != moduleID {
				continue
			}
			found = true
			if !item.Enabled || item.Source != "default" || !item.Editable {
				t.Fatalf("expected %s module enabled by default and runtime-editable: %+v", moduleID, item)
			}
		}
		if !found {
			t.Fatalf("expected module definition for %s", moduleID)
		}
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
