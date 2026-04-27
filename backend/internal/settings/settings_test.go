package settings

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"basepro/backend/internal/audit"
	"basepro/backend/internal/sukumad/rapidex"
	dhis2metadata "basepro/backend/internal/sukumad/rapidex/dhis2metadata"
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

type fakeRapidexMetadataServerCatalog struct {
	records map[string]sukumadserver.Record
	items   []sukumadserver.Record
}

func (f fakeRapidexMetadataServerCatalog) ListServers(context.Context, sukumadserver.ListQuery) (sukumadserver.ListResult, error) {
	return sukumadserver.ListResult{Items: append([]sukumadserver.Record(nil), f.items...)}, nil
}

func (f fakeRapidexMetadataServerCatalog) GetServerByCode(_ context.Context, code string) (sukumadserver.Record, error) {
	record, ok := f.records[code]
	if !ok {
		return sukumadserver.Record{}, errors.New("server not found")
	}
	return record, nil
}

type fakeRapidexFlowClient struct {
	flows  []rapidpro.Flow
	fields []rapidpro.ContactField
}

func (f fakeRapidexFlowClient) ListFlows(context.Context, rapidpro.Connection) ([]rapidpro.Flow, error) {
	return append([]rapidpro.Flow(nil), f.flows...), nil
}

func (f fakeRapidexFlowClient) ListContactFields(context.Context, rapidpro.Connection) ([]rapidpro.ContactField, error) {
	return append([]rapidpro.ContactField(nil), f.fields...), nil
}

type fakeRapidexDHIS2MetadataClient struct {
	lastConn            ConnectionCapture
	requestedDatasetIDs []string
}

type ConnectionCapture struct {
	BaseURL   string
	Headers   map[string]string
	URLParams map[string]string
}

func (f *fakeRapidexDHIS2MetadataClient) capture(conn dhis2metadata.Connection) {
	f.lastConn = ConnectionCapture{
		BaseURL:   conn.BaseURL,
		Headers:   cloneTestStringMap(conn.Headers),
		URLParams: cloneTestStringMap(conn.URLParams),
	}
}

func (f *fakeRapidexDHIS2MetadataClient) ListDataSets(_ context.Context, conn dhis2metadata.Connection) ([]dhis2metadata.DataSet, error) {
	f.capture(conn)
	return []dhis2metadata.DataSet{{ID: "ds1", Name: "Dataset 1"}}, nil
}

func (f *fakeRapidexDHIS2MetadataClient) GetDataSet(_ context.Context, conn dhis2metadata.Connection, datasetID string) (dhis2metadata.DataSet, error) {
	f.capture(conn)
	f.requestedDatasetIDs = append(f.requestedDatasetIDs, datasetID)
	return dhis2metadata.DataSet{
		ID:         datasetID,
		Name:       "Dataset " + datasetID,
		PeriodType: "Monthly",
		DataSetElements: []dhis2metadata.DataSetElementItem{
			{
				DataElement: dhis2metadata.DataElement{
					ID:        "de1",
					Name:      "Data Element 1",
					ValueType: "NUMBER",
					CategoryCombo: dhis2metadata.CategoryCombo{
						ID:   "cc1",
						Name: "Default",
						CategoryOptionCombos: []dhis2metadata.CategoryOptionCombo{
							{ID: "coc1", Name: "Default"},
						},
					},
				},
			},
		},
	}, nil
}

func (f *fakeAuditRepo) Insert(_ context.Context, event audit.Event) error {
	f.events = append(f.events, event)
	return nil
}

func cloneTestStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
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

func TestUpdateRapidexWebhookMappingsNormalizesAndSortsMappings(t *testing.T) {
	repo := newFakeRepo()
	service := NewService(repo, nil)

	got, err := service.UpdateRapidexWebhookMappings(context.Background(), RapidexWebhookMappingsUpdateInput{
		Mappings: []rapidex.MappingConfig{
			{
				FlowUUID:   " flow-b ",
				FlowName:   " Beta ",
				Dataset:    " DATASET_B ",
				OrgUnitVar: " facility_b ",
				PeriodVar:  " period_b ",
				Mappings: []rapidex.DataValueMapping{
					{Field: " indicator_b ", DataElement: " DE_B "},
				},
			},
			{
				FlowUUID:   "flow-a",
				FlowName:   "Alpha",
				Dataset:    "DATASET_A",
				OrgUnitVar: "facility_a",
				PeriodVar:  "period_a",
				Mappings: []rapidex.DataValueMapping{
					{Field: "indicator_a", DataElement: "DE_A"},
				},
			},
		},
	}, nil)
	if err != nil {
		t.Fatalf("update rapidex webhook mappings: %v", err)
	}
	if len(got.Mappings) != 2 {
		t.Fatalf("expected 2 mappings, got %#v", got.Mappings)
	}
	if got.Mappings[0].FlowUUID != "flow-a" || got.Mappings[1].FlowUUID != "flow-b" {
		t.Fatalf("expected mappings sorted by name/uuid, got %#v", got.Mappings)
	}
	if got.Mappings[1].Dataset != "DATASET_B" || got.Mappings[1].Mappings[0].Field != "indicator_b" {
		t.Fatalf("expected trimmed mapping values, got %#v", got.Mappings[1])
	}
	if !got.Validation.IsValid {
		t.Fatalf("expected valid settings, got %#v", got.Validation)
	}
}

func TestUpdateRapidexWebhookMappingsRejectsDuplicateFlowUUID(t *testing.T) {
	service := NewService(newFakeRepo(), nil)

	_, err := service.UpdateRapidexWebhookMappings(context.Background(), RapidexWebhookMappingsUpdateInput{
		Mappings: []rapidex.MappingConfig{
			{FlowUUID: "flow-a", Dataset: "dataset-a", OrgUnitVar: "facility", PeriodVar: "period", Mappings: []rapidex.DataValueMapping{{Field: "one", DataElement: "DE1"}}},
			{FlowUUID: "flow-a", Dataset: "dataset-b", OrgUnitVar: "facility", PeriodVar: "period", Mappings: []rapidex.DataValueMapping{{Field: "two", DataElement: "DE2"}}},
		},
	}, nil)
	if err == nil {
		t.Fatal("expected duplicate flow uuid validation error")
	}
}

func TestImportAndExportRapidexWebhookMappingsYAML(t *testing.T) {
	repo := newFakeRepo()
	service := NewService(repo, nil)

	_, err := service.ImportRapidexWebhookMappingsYAML(context.Background(), RapidexWebhookMappingsImportInput{
		YAML: `flow_uuid: 11111111-2222-3333-4444-555555555555
flow_name: Weekly Report
dataset: DATASET_A
org_unit_var: facility_code
period_var: reporting_period
mappings:
  - field: indicator_one
    data_element: DE_1
---
flow_uuid: 66666666-7777-8888-9999-000000000000
flow_name: Monthly Report
dataset: DATASET_B
org_unit_var: facility_uid
period_var: reporting_month
mappings:
  - field: indicator_two
    data_element: DE_2
`,
	}, nil)
	if err != nil {
		t.Fatalf("import rapidex webhook mappings: %v", err)
	}

	exported, err := service.ExportRapidexWebhookMappingsYAML(context.Background())
	if err != nil {
		t.Fatalf("export rapidex webhook mappings: %v", err)
	}
	if !strings.Contains(exported.YAML, "flow_uuid: 11111111-2222-3333-4444-555555555555") || !strings.Contains(exported.YAML, "---") {
		t.Fatalf("expected exported yaml to include both mapping docs, got %q", exported.YAML)
	}
}

func TestRapidexWebhookMappingProviderReturnsMappingByFlowUUID(t *testing.T) {
	repo := newFakeRepo()
	payload := rapidexWebhookMappingsStored{
		Mappings: []rapidex.MappingConfig{
			{FlowUUID: "flow-a", Dataset: "dataset-a", OrgUnitVar: "facility", PeriodVar: "period", Mappings: []rapidex.DataValueMapping{{Field: "one", DataElement: "DE1"}}},
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal test payload: %v", err)
	}
	repo.items["rapidex::webhook_mappings"] = raw

	provider := NewRapidexWebhookMappingProvider(repo)
	got, ok, err := provider.GetByFlowUUID(context.Background(), "flow-a")
	if err != nil {
		t.Fatalf("get by flow uuid: %v", err)
	}
	if !ok {
		t.Fatal("expected mapping to be found")
	}
	if got.FlowUUID != "flow-a" || got.Dataset != "dataset-a" {
		t.Fatalf("unexpected mapping %#v", got)
	}
}

func TestRefreshRapidexWebhookMetadataPassesDHIS2URLParams(t *testing.T) {
	repo := newFakeRepo()
	catalog := fakeRapidexMetadataServerCatalog{
		records: map[string]sukumadserver.Record{
			"rapidpro": {Code: "rapidpro", BaseURL: "https://rapidpro.example.com"},
			"dhis2": {
				Code:      "dhis2",
				BaseURL:   "https://dhis.example.com/api",
				URLParams: map[string]string{"locale": "en", "strategy": "children"},
				Headers:   map[string]string{"Authorization": "ApiToken test"},
			},
		},
		items: []sukumadserver.Record{
			{Code: "rapidpro", Name: "RapidPro", SystemType: "rapidpro"},
			{Code: "dhis2", Name: "DHIS2", SystemType: "dhis2"},
		},
	}
	dhis2Client := &fakeRapidexDHIS2MetadataClient{}
	service := NewService(repo, nil).
		WithRapidexMetadataIntegration(
			catalog,
			fakeRapidexFlowClient{
				flows:  []rapidpro.Flow{{UUID: "flow-1", Name: "Facility Flow"}},
				fields: []rapidpro.ContactField{{Key: "facility_code", Label: "Facility Code"}},
			},
			dhis2Client,
		)

	_, err := service.RefreshRapidexWebhookMetadata(context.Background(), RapidexWebhookMetadataRefreshInput{
		RapidProServerCode: "rapidpro",
		Dhis2ServerCode:    "dhis2",
		Scope:              "catalog",
	}, nil)
	if err != nil {
		t.Fatalf("refresh rapidex webhook metadata: %v", err)
	}
	if dhis2Client.lastConn.BaseURL != "https://dhis.example.com/api" {
		t.Fatalf("expected dhis2 base url to pass through, got %q", dhis2Client.lastConn.BaseURL)
	}
	if dhis2Client.lastConn.URLParams["locale"] != "en" || dhis2Client.lastConn.URLParams["strategy"] != "children" {
		t.Fatalf("expected dhis2 url params to pass through, got %#v", dhis2Client.lastConn.URLParams)
	}
	if dhis2Client.lastConn.Headers["Authorization"] != "ApiToken test" {
		t.Fatalf("expected dhis2 headers to pass through, got %#v", dhis2Client.lastConn.Headers)
	}
}

func TestRefreshRapidexWebhookMetadataAliasesAOCOptionsFromCategoryOptionCombos(t *testing.T) {
	repo := newFakeRepo()
	catalog := fakeRapidexMetadataServerCatalog{
		records: map[string]sukumadserver.Record{
			"rapidpro": {Code: "rapidpro", BaseURL: "https://rapidpro.example.com"},
			"dhis2":    {Code: "dhis2", BaseURL: "https://dhis.example.com"},
		},
		items: []sukumadserver.Record{
			{Code: "rapidpro", Name: "RapidPro", SystemType: "rapidpro"},
			{Code: "dhis2", Name: "DHIS2", SystemType: "dhis2"},
		},
	}
	dhis2Client := &fakeRapidexDHIS2MetadataClient{}
	service := NewService(repo, nil).
		WithRapidexMetadataIntegration(
			catalog,
			fakeRapidexFlowClient{
				flows:  []rapidpro.Flow{{UUID: "flow-1", Name: "Facility Flow"}},
				fields: []rapidpro.ContactField{{Key: "facility_code", Label: "Facility Code"}},
			},
			dhis2Client,
		)

	got, err := service.RefreshRapidexWebhookMetadata(context.Background(), RapidexWebhookMetadataRefreshInput{
		RapidProServerCode: "rapidpro",
		Dhis2ServerCode:    "dhis2",
		Scope:              "datasets",
		DatasetIDs:         []string{"ds1"},
	}, nil)
	if err != nil {
		t.Fatalf("refresh rapidex webhook metadata: %v", err)
	}
	metadata, ok := got.Snapshot.Dhis2DatasetMetadataByID["ds1"]
	if !ok {
		t.Fatalf("expected dataset metadata for ds1, got %#v", got.Snapshot.Dhis2DatasetMetadataByID)
	}
	if len(metadata.CategoryOptionCombos) != 1 || metadata.CategoryOptionCombos[0].ID != "coc1" {
		t.Fatalf("expected category option combos in dataset metadata, got %#v", metadata.CategoryOptionCombos)
	}
	if len(metadata.AttributeOptionCombos) != 1 || metadata.AttributeOptionCombos[0].ID != "coc1" {
		t.Fatalf("expected attribute option combos to mirror category option combos, got %#v", metadata.AttributeOptionCombos)
	}
}

func TestRefreshRapidexWebhookMetadataRequiresDatasetIDsForDatasetScope(t *testing.T) {
	repo := newFakeRepo()
	service := NewService(repo, nil).
		WithRapidexMetadataIntegration(
			fakeRapidexMetadataServerCatalog{
				records: map[string]sukumadserver.Record{
					"rapidpro": {Code: "rapidpro", BaseURL: "https://rapidpro.example.com"},
					"dhis2":    {Code: "dhis2", BaseURL: "https://dhis.example.com"},
				},
			},
			fakeRapidexFlowClient{},
			&fakeRapidexDHIS2MetadataClient{},
		)

	_, err := service.RefreshRapidexWebhookMetadata(context.Background(), RapidexWebhookMetadataRefreshInput{
		RapidProServerCode: "rapidpro",
		Dhis2ServerCode:    "dhis2",
		Scope:              "datasets",
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "validation failed") {
		t.Fatalf("expected dataset validation error, got %v", err)
	}
}

func TestRefreshRapidexWebhookMetadataLoadsOnlyRequestedDatasets(t *testing.T) {
	repo := newFakeRepo()
	catalog := fakeRapidexMetadataServerCatalog{
		records: map[string]sukumadserver.Record{
			"rapidpro": {Code: "rapidpro", BaseURL: "https://rapidpro.example.com"},
			"dhis2":    {Code: "dhis2", BaseURL: "https://dhis.example.com"},
		},
	}
	dhis2Client := &fakeRapidexDHIS2MetadataClient{}
	service := NewService(repo, nil).
		WithRapidexMetadataIntegration(catalog, fakeRapidexFlowClient{}, dhis2Client)

	_, err := service.RefreshRapidexWebhookMetadata(context.Background(), RapidexWebhookMetadataRefreshInput{
		RapidProServerCode: "rapidpro",
		Dhis2ServerCode:    "dhis2",
		Scope:              "datasets",
		DatasetIDs:         []string{"ds2"},
	}, nil)
	if err != nil {
		t.Fatalf("refresh rapidex webhook metadata: %v", err)
	}
	if len(dhis2Client.requestedDatasetIDs) != 1 || dhis2Client.requestedDatasetIDs[0] != "ds2" {
		t.Fatalf("expected only ds2 to be requested, got %#v", dhis2Client.requestedDatasetIDs)
	}
}
