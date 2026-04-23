package settings

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"basepro/backend/internal/audit"
	"basepro/backend/internal/sukumad/rapidex/rapidpro"
	sukumadserver "basepro/backend/internal/sukumad/server"
)

type fakeRepo struct {
	items map[string]json.RawMessage
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{items: map[string]json.RawMessage{}}
}

func (f *fakeRepo) key(category, key string) string {
	return category + "::" + key
}

func (f *fakeRepo) Get(_ context.Context, category, key string) (json.RawMessage, error) {
	value, ok := f.items[f.key(category, key)]
	if !ok {
		return nil, ErrNotFound
	}
	return append(json.RawMessage{}, value...), nil
}

func (f *fakeRepo) Upsert(_ context.Context, category, key string, value json.RawMessage, _ *int64, _ time.Time) error {
	f.items[f.key(category, key)] = append(json.RawMessage{}, value...)
	return nil
}

type fakeAuditRepo struct {
	events []audit.Event
}

type fakeRapidProServerLookup struct {
	record sukumadserver.Record
}

func (f fakeRapidProServerLookup) GetServerByCode(context.Context, string) (sukumadserver.Record, error) {
	return f.record, nil
}

type fakeRapidProFieldClient struct {
	fields []rapidpro.ContactField
}

func (f fakeRapidProFieldClient) ListContactFields(context.Context, rapidpro.Connection) ([]rapidpro.ContactField, error) {
	return append([]rapidpro.ContactField(nil), f.fields...), nil
}

func (f *fakeAuditRepo) Insert(_ context.Context, event audit.Event) error {
	f.events = append(f.events, event)
	return nil
}

func (f *fakeAuditRepo) List(context.Context, audit.ListFilter) (audit.ListResult, error) {
	return audit.ListResult{}, nil
}

func TestGetLoginBrandingDefaultsWhenMissing(t *testing.T) {
	service := NewService(newFakeRepo(), nil)

	got, err := service.GetLoginBranding(context.Background())
	if err != nil {
		t.Fatalf("get branding: %v", err)
	}
	if got.ApplicationDisplayName != "RapidEx" {
		t.Fatalf("expected RapidEx default, got %q", got.ApplicationDisplayName)
	}
	if got.ImageConfigured {
		t.Fatal("expected imageConfigured false")
	}
}

func TestUpdateAndGetLoginBranding(t *testing.T) {
	repo := newFakeRepo()
	auditRepo := &fakeAuditRepo{}
	service := NewService(repo, audit.NewService(auditRepo))
	actor := int64(42)

	url := "https://cdn.example.com/brand.png"
	asset := "assets/login/custom.png"
	updated, err := service.UpdateLoginBranding(context.Background(), LoginBrandingUpdateInput{
		ApplicationDisplayName: "RapidEx Ops",
		LoginImageURL:          &url,
		LoginImageAssetPath:    &asset,
	}, &actor)
	if err != nil {
		t.Fatalf("update branding: %v", err)
	}
	if updated.ApplicationDisplayName != "RapidEx Ops" {
		t.Fatalf("unexpected display name: %q", updated.ApplicationDisplayName)
	}
	if !updated.ImageConfigured {
		t.Fatal("expected imageConfigured true")
	}

	got, err := service.GetLoginBranding(context.Background())
	if err != nil {
		t.Fatalf("get branding: %v", err)
	}
	if got.ApplicationDisplayName != "RapidEx Ops" {
		t.Fatalf("expected persisted display name, got %q", got.ApplicationDisplayName)
	}
	if got.LoginImageURL == nil || *got.LoginImageURL != url {
		t.Fatalf("expected persisted loginImageUrl %q", url)
	}

	if len(auditRepo.events) != 1 || auditRepo.events[0].Action != "settings.login_branding.update" {
		t.Fatalf("expected settings.login_branding.update audit event, got %+v", auditRepo.events)
	}
}

func TestUpdateLoginBrandingRejectsBadURL(t *testing.T) {
	service := NewService(newFakeRepo(), nil)
	bad := "javascript:alert(1)"
	_, err := service.UpdateLoginBranding(context.Background(), LoginBrandingUpdateInput{
		ApplicationDisplayName: "RapidEx",
		LoginImageURL:          &bad,
	}, nil)
	if err == nil {
		t.Fatal("expected validation error for bad URL")
	}
}

func TestGetRuntimeConfigMasksSensitiveFields(t *testing.T) {
	service := NewService(newFakeRepo(), nil).WithRuntimeConfigProvider(func() map[string]any {
		return map[string]any{
			"database": map[string]any{
				"dsn": "postgres://dbuser:dbpass@db.example.com:5432/sukumad?sslmode=disable&password=querysecret",
			},
			"auth": map[string]any{
				"jwt_signing_key":       "super-secret-key",
				"api_token_header_name": "X-API-Token",
				"api_token_enabled":     true,
			},
		}
	})
	got, err := service.GetRuntimeConfig(context.Background())
	if err != nil {
		t.Fatalf("get runtime config: %v", err)
	}

	database, ok := got["database"].(map[string]any)
	if !ok {
		t.Fatalf("expected database map, got %#v", got["database"])
	}
	dsn, ok := database["dsn"].(string)
	if !ok {
		t.Fatalf("expected dsn string, got %#v", database["dsn"])
	}
	for _, forbidden := range []string{"dbpass", "querysecret"} {
		if strings.Contains(dsn, forbidden) {
			t.Fatalf("expected dsn to mask %q, got %q", forbidden, dsn)
		}
	}
	if !strings.Contains(dsn, "db.example.com:5432/sukumad") {
		t.Fatalf("expected dsn to preserve host/db context, got %q", dsn)
	}

	authSection, ok := got["auth"].(map[string]any)
	if !ok {
		t.Fatalf("expected auth map, got %#v", got["auth"])
	}
	if authSection["jwt_signing_key"] != "[masked]" {
		t.Fatalf("expected jwt_signing_key masked, got %#v", authSection["jwt_signing_key"])
	}
	if authSection["api_token_header_name"] != "X-API-Token" {
		t.Fatalf("expected api_token_header_name preserved, got %#v", authSection["api_token_header_name"])
	}
	if authSection["api_token_enabled"] != true {
		t.Fatalf("expected api_token_enabled preserved, got %#v", authSection["api_token_enabled"])
	}
}

func TestMaskDSNReturnsMaskedFallbackForOpaqueValues(t *testing.T) {
	masked := maskDSN("Server=localhost;User Id=sa;Password=super-secret;Database=basepro")
	if strings.Contains(masked, "super-secret") {
		t.Fatalf("expected password masked, got %q", masked)
	}
	if !strings.Contains(masked, "Password=[masked]") {
		t.Fatalf("expected masked password placeholder, got %q", masked)
	}
}

func TestRefreshRapidProReporterSyncFieldsAppliesFacilitySuggestions(t *testing.T) {
	repo := newFakeRepo()
	service := NewService(repo, nil).
		WithRapidProIntegration(
			fakeRapidProServerLookup{record: sukumadserver.Record{Code: "rapidpro", BaseURL: "https://rapidpro.example.com"}},
			fakeRapidProFieldClient{fields: []rapidpro.ContactField{
				{Key: "Facility", Label: "Facility"},
				{Key: "FacilityCode", Label: "FacilityCode"},
				{Key: "District", Label: "District"},
			}},
		)

	got, err := service.RefreshRapidProReporterSyncFields(context.Background(), nil)
	if err != nil {
		t.Fatalf("refresh rapidpro fields: %v", err)
	}
	if got.RapidProServerCode != "rapidpro" {
		t.Fatalf("expected default server code, got %q", got.RapidProServerCode)
	}
	if len(got.AvailableFields) != 6 {
		t.Fatalf("expected 6 fields including built-ins, got %d", len(got.AvailableFields))
	}
	if got.AvailableFields[0].Key != "name" || got.AvailableFields[0].ValueType != "builtin" {
		t.Fatalf("expected built-in name field first, got %#v", got.AvailableFields[0])
	}
	if got.AvailableFields[1].Key != "urn.tel" || got.AvailableFields[1].ValueType != "builtin" {
		t.Fatalf("expected built-in tel field second, got %#v", got.AvailableFields[1])
	}
	if got.AvailableFields[2].Key != "urn.whatsapp" || got.AvailableFields[2].ValueType != "builtin" {
		t.Fatalf("expected built-in whatsapp field third, got %#v", got.AvailableFields[2])
	}
	if len(got.Mappings) != 2 {
		t.Fatalf("expected 2 suggested mappings, got %#v", got.Mappings)
	}
	if got.Mappings[0].SourceKey != "facilityName" || got.Mappings[0].RapidProFieldKey != "Facility" {
		t.Fatalf("expected Facility suggestion, got %#v", got.Mappings[0])
	}
	if got.Mappings[1].SourceKey != "facilityUID" || got.Mappings[1].RapidProFieldKey != "FacilityCode" {
		t.Fatalf("expected FacilityCode suggestion, got %#v", got.Mappings[1])
	}
	if got.LastFetchedAt == nil {
		t.Fatal("expected lastFetchedAt to be populated")
	}
	if !got.Validation.IsValid {
		t.Fatalf("expected valid settings, got %#v", got.Validation)
	}
}

func TestUpdateRapidProReporterSyncRejectsUnknownField(t *testing.T) {
	repo := newFakeRepo()
	repo.items["rapidpro::reporter_sync"] = json.RawMessage(`{
		"rapidProServerCode":"rapidpro",
		"availableFields":[{"key":"Facility","label":"Facility"}]
	}`)
	service := NewService(repo, nil)

	_, err := service.UpdateRapidProReporterSync(context.Background(), RapidProReporterSyncUpdateInput{
		RapidProServerCode: "rapidpro",
		Mappings: []RapidProReporterFieldMapping{
			{SourceKey: "facilityName", RapidProFieldKey: "FacilityCode"},
		},
	}, nil)
	if err == nil {
		t.Fatal("expected validation error for unavailable field")
	}
}

func TestUpdateRapidProReporterSyncAcceptsBuiltInTarget(t *testing.T) {
	repo := newFakeRepo()
	service := NewService(repo, nil)

	got, err := service.UpdateRapidProReporterSync(context.Background(), RapidProReporterSyncUpdateInput{
		RapidProServerCode: "rapidpro",
		Mappings: []RapidProReporterFieldMapping{
			{SourceKey: "telephone", RapidProFieldKey: "urn.tel"},
		},
	}, nil)
	if err != nil {
		t.Fatalf("expected built-in target to validate, got %v", err)
	}
	if len(got.Mappings) != 1 || got.Mappings[0].RapidProFieldKey != "urn.tel" {
		t.Fatalf("expected built-in mapping to persist, got %#v", got.Mappings)
	}
}
