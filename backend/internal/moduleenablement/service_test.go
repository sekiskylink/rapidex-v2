package moduleenablement

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"basepro/backend/internal/audit"
	"basepro/backend/internal/settings"
)

type fakeSettingsRepository struct {
	value []byte
}

func (f *fakeSettingsRepository) Get(context.Context, string, string) (json.RawMessage, error) {
	if len(f.value) == 0 {
		return nil, settings.ErrNotFound
	}
	return append([]byte(nil), f.value...), nil
}

func (f *fakeSettingsRepository) Upsert(_ context.Context, _, _ string, value json.RawMessage, _ *int64, _ time.Time) error {
	f.value = append([]byte(nil), value...)
	return nil
}

type fakeAuditRepository struct {
	events []audit.Event
}

func (f *fakeAuditRepository) Insert(_ context.Context, event audit.Event) error {
	f.events = append(f.events, event)
	return nil
}

func (f *fakeAuditRepository) List(context.Context, audit.ListFilter) (audit.ListResult, error) {
	return audit.ListResult{}, nil
}

func TestUpdateRuntimeOverridesRejectsNonEditableModules(t *testing.T) {
	svc := NewService(&fakeSettingsRepository{}, nil)
	_, err := svc.UpdateRuntimeOverrides(context.Background(), []RuntimeModuleOverride{
		{ModuleID: "settings", Enabled: false},
	}, nil, nil)
	if err == nil {
		t.Fatal("expected validation error for non-editable module")
	}
}

func TestUpdateRuntimeOverridesPersistsAndLogsAudit(t *testing.T) {
	settingsRepo := &fakeSettingsRepository{}
	auditRepo := &fakeAuditRepository{}
	svc := NewService(settingsRepo, audit.NewService(auditRepo))

	actorID := int64(42)
	modules, err := svc.UpdateRuntimeOverrides(context.Background(), []RuntimeModuleOverride{
		{ModuleID: "administration", Enabled: false},
	}, nil, &actorID)
	if err != nil {
		t.Fatalf("update runtime overrides: %v", err)
	}

	if len(modules) != 8 {
		t.Fatalf("expected 8 modules, got %d", len(modules))
	}
	if len(auditRepo.events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(auditRepo.events))
	}
	if auditRepo.events[0].Action != "settings.module_enablement.update" {
		t.Fatalf("unexpected audit action: %s", auditRepo.events[0].Action)
	}

	var stored runtimeOverridesStored
	if err := json.Unmarshal(settingsRepo.value, &stored); err != nil {
		t.Fatalf("decode stored overrides: %v", err)
	}
	if enabled, ok := stored.Flags["modules.administration.enabled"]; !ok || enabled {
		t.Fatalf("expected administration runtime override false, got %+v", stored.Flags)
	}
}
