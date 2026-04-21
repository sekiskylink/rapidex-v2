package rbac

import (
	"testing"
)

func TestBasePermissionRegistry(t *testing.T) {
	definitions := BasePermissionRegistry()
	if len(definitions) == 0 {
		t.Fatalf("expected permission definitions")
	}

	seen := map[string]struct{}{}
	for _, definition := range definitions {
		if definition.Key == "" {
			t.Fatalf("permission key must not be empty")
		}
		if _, exists := seen[definition.Key]; exists {
			t.Fatalf("duplicate permission key: %s", definition.Key)
		}
		seen[definition.Key] = struct{}{}
	}

	required := []string{
		PermissionUsersRead,
		PermissionUsersWrite,
		PermissionAuditRead,
		PermissionSettingsRead,
		PermissionSettingsWrite,
		PermissionServersRead,
		PermissionServersWrite,
		PermissionRequestsRead,
		PermissionRequestsWrite,
		PermissionDeliveriesRead,
		PermissionDeliveriesWrite,
		PermissionJobsRead,
		PermissionJobsWrite,
		PermissionSchedulerRead,
		PermissionSchedulerWrite,
		PermissionObservabilityRead,
		PermissionOrgUnitsRead,
		PermissionOrgUnitsWrite,
		PermissionReportersRead,
		PermissionReportersWrite,
	}
	for _, key := range required {
		if _, exists := seen[key]; !exists {
			t.Fatalf("missing required permission: %s", key)
		}
	}
}

func TestBaseModuleRegistry(t *testing.T) {
	modules := BaseModuleRegistry()
	if len(modules) != 12 {
		t.Fatalf("expected 12 base modules, got %d", len(modules))
	}

	ids := map[string]struct{}{}
	for _, module := range modules {
		if module.ID == "" {
			t.Fatalf("module id must not be empty")
		}
		if _, exists := ids[module.ID]; exists {
			t.Fatalf("duplicate module id: %s", module.ID)
		}
		ids[module.ID] = struct{}{}
	}

	if _, ok := ids["administration"]; !ok {
		t.Fatalf("expected administration module")
	}
	if _, ok := ids["servers"]; !ok {
		t.Fatalf("expected servers module")
	}
	if _, ok := ids["scheduler"]; !ok {
		t.Fatalf("expected scheduler module")
	}
	if _, ok := ids["documentation"]; !ok {
		t.Fatalf("expected documentation module")
	}
	if _, ok := ids["orgunits"]; !ok {
		t.Fatalf("expected orgunits module")
	}
	if _, ok := ids["reporters"]; !ok {
		t.Fatalf("expected reporters module")
	}
}
