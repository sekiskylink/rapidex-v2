package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/auth"
	"basepro/backend/internal/rbac"
	"basepro/backend/internal/sukumad/orgunit"
	"basepro/backend/internal/sukumad/rapidex"
	"basepro/backend/internal/sukumad/reporter"
	request "basepro/backend/internal/sukumad/request"
	sukumadserver "basepro/backend/internal/sukumad/server"
)

func TestRapidexRoutesRequireAuthAndPermissions(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(120, "rapidex-reader", time.Now().UTC())

	deps := AppDeps{
		JWTManager:          jwt,
		RBACService:         rbacServiceWithPermissions(map[int64][]string{120: {rbac.PermissionOrgUnitsRead, rbac.PermissionReportersRead}}),
		ModuleFlagsProvider: func() map[string]bool { return map[string]bool{} },
		OrgUnitService:      orgunit.NewService(&rapidexOrgUnitRepo{}),
		ReporterService:     reporter.NewService(&rapidexReporterRepo{}),
	}
	router := newRouter(deps)

	for _, path := range []string{"/api/v1/orgunits", "/api/v1/reporters"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected %s without auth to return 401, got %d", path, w.Code)
		}

		req = httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected %s with permission to return 200, got %d body=%s", path, w.Code, w.Body.String())
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reporters/broadcasts", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected /api/v1/reporters/broadcasts without auth to return 401, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/reporters/broadcasts?page=0&pageSize=10", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected /api/v1/reporters/broadcasts with permission to return 200, got %d body=%s", w.Code, w.Body.String())
	}

	for _, path := range []string{"/api/v1/reporters/1/rapidpro-contact", "/api/v1/reporters/1/chat-history", "/api/v1/reporters/1/reports"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected %s without auth to return 401, got %d", path, w.Code)
		}
	}
}

func TestRapidexRoutesRejectMissingPermission(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(121, "rapidex-viewer", time.Now().UTC())

	router := newRouter(AppDeps{
		JWTManager:          jwt,
		RBACService:         rbacServiceWithPermissions(map[int64][]string{121: {}}),
		ModuleFlagsProvider: func() map[string]bool { return map[string]bool{} },
		OrgUnitService:      orgunit.NewService(&rapidexOrgUnitRepo{}),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgunits", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected orgunits without permission to return 403, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestRapidexOrgUnitRoutePassesHierarchyQueryFlags(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(122, "rapidex-query-reader", time.Now().UTC())
	repo := &rapidexOrgUnitRepo{}

	router := newRouter(AppDeps{
		JWTManager:          jwt,
		RBACService:         rbacServiceWithPermissions(map[int64][]string{122: {rbac.PermissionOrgUnitsRead}}),
		ModuleFlagsProvider: func() map[string]bool { return map[string]bool{} },
		OrgUnitService:      orgunit.NewService(repo),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgunits?page=2&pageSize=15&search=hospital&rootsOnly=true&leafOnly=true", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	if repo.lastQuery.Page != 2 || repo.lastQuery.PageSize != 15 {
		t.Fatalf("expected pagination to be passed through, got %+v", repo.lastQuery)
	}
	if repo.lastQuery.Search != "hospital" {
		t.Fatalf("expected search to be passed through, got %+v", repo.lastQuery)
	}
	if !repo.lastQuery.RootsOnly || !repo.lastQuery.LeafOnly {
		t.Fatalf("expected rootsOnly and leafOnly to be true, got %+v", repo.lastQuery)
	}

	var body struct {
		Items []orgunit.OrgUnit `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(body.Items) != 1 || !body.Items[0].HasChildren {
		t.Fatalf("expected response to include org unit with hasChildren=true, got %+v", body.Items)
	}
}

func TestRapidexWebhookRouteAcceptsAPITokenAndQueuesRequest(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	deps := newSukumadTestAppDeps(jwt, rbacServiceWithPermissions(nil))
	deps.ModuleFlagsProvider = func() map[string]bool { return map[string]bool{"requests": true} }

	tokenRepo := newAPITokenRepo()
	secret := "test-secret"
	plain := "bpt_requestswrite"
	hash := auth.HashAPIToken(secret, plain)
	tokenRepo.tokens[hash] = &auth.APIToken{
		ID:        51,
		Name:      "rapidex-webhook",
		TokenHash: hash,
		Prefix:    auth.APITokenPrefix(plain),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	tokenRepo.permissions[51] = []auth.APITokenPermission{{APITokenID: 51, Permission: rbac.PermissionRequestsWrite}}
	deps.AuthService = auth.NewService(tokenRepo, nil, jwt, nil, time.Minute, time.Hour, time.Hour, secret, true, 4)
	deps.APITokenHeaderName = "X-API-Token"

	requestCreator := &rapidexRouteRequestCreator{}
	deps.RapidexService = rapidex.NewIntegrationService(rapidexRouteMappingProvider{
		binding: rapidex.WebhookBinding{
			MappingConfig: rapidex.MappingConfig{
				FlowUUID:   "flow-1",
				Dataset:    "ds-1",
				OrgUnitVar: "facility",
				PeriodVar:  "period",
				Mappings: []rapidex.DataValueMapping{
					{Field: "value_a", DataElement: "de-1"},
				},
			},
			DHIS2ServerCode: "dhis2-main",
		},
		ok: true,
	}, nil, requestCreator, rapidexRouteServerResolver{
		record: sukumadserver.Record{UID: "dhis2-uid"},
	})

	router := newRouter(deps)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rapidex/webhook", bytes.NewReader([]byte(`{
		"flow_uuid":"flow-1",
		"results":{"facility":"OU_123","period":"202604","value_a":"17"},
		"contact":{"uuid":"contact-1","urns":["tel:+256782820208"]}
	}`)))
	req.Header.Set("X-API-Token", plain)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", w.Code, w.Body.String())
	}
	if len(requestCreator.calls) != 1 {
		t.Fatalf("expected request queue call, got %d", len(requestCreator.calls))
	}
}

func TestRapidexWebhookRouteRequiresAuthentication(t *testing.T) {
	router := newRouter(AppDeps{
		ModuleFlagsProvider: func() map[string]bool { return map[string]bool{"requests": true} },
		RapidexService: rapidex.NewIntegrationService(rapidexRouteMappingProvider{
			binding: rapidex.WebhookBinding{
				MappingConfig: rapidex.MappingConfig{
					FlowUUID:   "flow-1",
					Dataset:    "ds-1",
					OrgUnitVar: "facility",
					PeriodVar:  "period",
					Mappings:   []rapidex.DataValueMapping{{Field: "value_a", DataElement: "de-1"}},
				},
				DHIS2ServerCode: "dhis2-main",
			},
			ok: true,
		}, nil, &rapidexRouteRequestCreator{}, rapidexRouteServerResolver{
			record: sukumadserver.Record{UID: "dhis2-uid"},
		}),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rapidex/webhook", bytes.NewReader([]byte(`{"flow_uuid":"flow-1"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestRapidexWebhookRouteReturnsValidationError(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	deps := newSukumadTestAppDeps(jwt, rbacServiceWithPermissions(nil))
	deps.ModuleFlagsProvider = func() map[string]bool { return map[string]bool{"requests": true} }

	tokenRepo := newAPITokenRepo()
	secret := "test-secret"
	plain := "bpt_requestswrite_validation"
	hash := auth.HashAPIToken(secret, plain)
	tokenRepo.tokens[hash] = &auth.APIToken{
		ID:        52,
		Name:      "rapidex-webhook",
		TokenHash: hash,
		Prefix:    auth.APITokenPrefix(plain),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	tokenRepo.permissions[52] = []auth.APITokenPermission{{APITokenID: 52, Permission: rbac.PermissionRequestsWrite}}
	deps.AuthService = auth.NewService(tokenRepo, nil, jwt, nil, time.Minute, time.Hour, time.Hour, secret, true, 4)
	deps.APITokenHeaderName = "X-API-Token"
	deps.RapidexService = rapidex.NewIntegrationService(rapidexRouteMappingProvider{
		binding: rapidex.WebhookBinding{
			MappingConfig: rapidex.MappingConfig{
				FlowUUID:   "flow-1",
				Dataset:    "ds-1",
				OrgUnitVar: "facility",
				PeriodVar:  "period",
				Mappings:   []rapidex.DataValueMapping{{Field: "value_a", DataElement: "de-1"}},
			},
			DHIS2ServerCode: "dhis2-main",
		},
		ok: true,
	}, nil, &rapidexRouteRequestCreator{}, rapidexRouteServerResolver{
		record: sukumadserver.Record{UID: "dhis2-uid"},
	})

	router := newRouter(deps)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/rapidex/webhook", bytes.NewReader([]byte(`{
		"flow_uuid":"flow-1",
		"results":{"period":"202604"}
	}`)))
	req.Header.Set("X-API-Token", plain)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Error struct {
			Code    string         `json:"code"`
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error.Code != apperror.CodeValidationFailed {
		t.Fatalf("expected validation code, got %+v", body.Error)
	}
	if _, ok := body.Error.Details["orgUnit"]; !ok {
		t.Fatalf("expected orgUnit validation details, got %+v", body.Error.Details)
	}
}

type rapidexRouteMappingProvider struct {
	binding rapidex.WebhookBinding
	ok      bool
	err     error
}

func (p rapidexRouteMappingProvider) GetByFlowUUID(context.Context, string) (rapidex.WebhookBinding, bool, error) {
	return p.binding, p.ok, p.err
}

type rapidexRouteRequestCreator struct {
	calls []rapidex.ExternalRequestInput
}

func (r *rapidexRouteRequestCreator) CreateExternalRequest(_ context.Context, input rapidex.ExternalRequestInput) error {
	r.calls = append(r.calls, input)
	return nil
}

type rapidexRouteServerResolver struct {
	record sukumadserver.Record
	err    error
}

func (r rapidexRouteServerResolver) GetServerByCode(context.Context, string) (sukumadserver.Record, error) {
	if r.err != nil {
		return sukumadserver.Record{}, r.err
	}
	return r.record, nil
}

type rapidexOrgUnitRepo struct {
	lastQuery orgunit.ListQuery
}

func (r *rapidexOrgUnitRepo) List(_ context.Context, query orgunit.ListQuery) (orgunit.ListResult, error) {
	r.lastQuery = query
	return orgunit.ListResult{
		Items: []orgunit.OrgUnit{{ID: 1, Name: "Kampala Health Centre", HasChildren: true}},
		Total: 1, Page: query.Page, PageSize: query.PageSize,
	}, nil
}
func (r *rapidexOrgUnitRepo) GetByID(context.Context, int64) (orgunit.OrgUnit, error) {
	return orgunit.OrgUnit{}, nil
}
func (r *rapidexOrgUnitRepo) GetByUID(context.Context, string) (orgunit.OrgUnit, error) {
	return orgunit.OrgUnit{}, nil
}
func (r *rapidexOrgUnitRepo) GetByCode(context.Context, string) (orgunit.OrgUnit, error) {
	return orgunit.OrgUnit{}, nil
}
func (r *rapidexOrgUnitRepo) Create(_ context.Context, unit orgunit.OrgUnit) (orgunit.OrgUnit, error) {
	unit.ID = 1
	return unit, nil
}
func (r *rapidexOrgUnitRepo) Update(_ context.Context, unit orgunit.OrgUnit) (orgunit.OrgUnit, error) {
	return unit, nil
}
func (r *rapidexOrgUnitRepo) Delete(context.Context, int64) error {
	return nil
}

type rapidexReporterRepo struct{}

func (r *rapidexReporterRepo) List(context.Context, reporter.ListQuery) (reporter.ListResult, error) {
	return reporter.ListResult{Items: []reporter.Reporter{}, Total: 0, Page: 0, PageSize: 20}, nil
}
func (r *rapidexReporterRepo) ListBroadcasts(context.Context, reporter.BroadcastListQuery) (reporter.BroadcastListResult, error) {
	return reporter.BroadcastListResult{
		Items:    []reporter.JurisdictionBroadcastRecord{{ID: 31, Status: reporter.BroadcastStatusQueued, ReporterGroup: "Lead"}},
		Total:    1,
		Page:     0,
		PageSize: 10,
	}, nil
}
func (r *rapidexReporterRepo) GetByID(context.Context, int64) (reporter.Reporter, error) {
	return reporter.Reporter{}, nil
}
func (r *rapidexReporterRepo) GetByUID(context.Context, string) (reporter.Reporter, error) {
	return reporter.Reporter{}, nil
}
func (r *rapidexReporterRepo) GetByRapidProUUID(context.Context, string) (reporter.Reporter, error) {
	return reporter.Reporter{}, nil
}
func (r *rapidexReporterRepo) GetByPhoneNumber(context.Context, string) (reporter.Reporter, error) {
	return reporter.Reporter{}, nil
}
func (r *rapidexReporterRepo) ListByIDs(context.Context, []int64) ([]reporter.Reporter, error) {
	return []reporter.Reporter{}, nil
}
func (r *rapidexReporterRepo) ListUpdatedSince(context.Context, *time.Time, int, bool) ([]reporter.Reporter, error) {
	return []reporter.Reporter{}, nil
}
func (r *rapidexReporterRepo) CountBroadcastRecipients(context.Context, reporter.BroadcastRecipientQuery) (int, error) {
	return 0, nil
}
func (r *rapidexReporterRepo) ListBroadcastRecipients(context.Context, reporter.BroadcastRecipientQuery) ([]reporter.Reporter, error) {
	return []reporter.Reporter{}, nil
}
func (r *rapidexReporterRepo) GetRecentPendingBroadcastByDedupeKey(context.Context, string, time.Time) (reporter.JurisdictionBroadcastRecord, error) {
	return reporter.JurisdictionBroadcastRecord{}, sql.ErrNoRows
}
func (r *rapidexReporterRepo) ListRecentReporterReports(context.Context, request.ReporterRecentReportsQuery) ([]request.Record, error) {
	return []request.Record{}, nil
}
func (r *rapidexReporterRepo) CreateJurisdictionBroadcast(_ context.Context, item reporter.JurisdictionBroadcastRecord) (reporter.JurisdictionBroadcastRecord, error) {
	item.ID = 1
	return item, nil
}
func (r *rapidexReporterRepo) ClaimNextJurisdictionBroadcast(context.Context, time.Time, time.Duration, int64) (reporter.JurisdictionBroadcastRecord, error) {
	return reporter.JurisdictionBroadcastRecord{}, reporter.ErrNoEligibleBroadcast
}
func (r *rapidexReporterRepo) UpdateJurisdictionBroadcastResult(_ context.Context, id int64, status string, sentCount int, failedCount int, lastError string, finishedAt time.Time) (reporter.JurisdictionBroadcastRecord, error) {
	return reporter.JurisdictionBroadcastRecord{ID: id, Status: status, SentCount: sentCount, FailedCount: failedCount, LastError: lastError, FinishedAt: &finishedAt}, nil
}
func (r *rapidexReporterRepo) UpdateRapidProStatus(_ context.Context, id int64, rapidProUUID string, synced bool) (reporter.Reporter, error) {
	return reporter.Reporter{ID: id, RapidProUUID: rapidProUUID, Synced: synced}, nil
}
func (r *rapidexReporterRepo) MarkForSync(_ context.Context, id int64) (reporter.Reporter, error) {
	return reporter.Reporter{ID: id, Synced: false}, nil
}
func (r *rapidexReporterRepo) Create(_ context.Context, item reporter.Reporter) (reporter.Reporter, error) {
	item.ID = 1
	return item, nil
}
func (r *rapidexReporterRepo) Update(_ context.Context, item reporter.Reporter) (reporter.Reporter, error) {
	return item, nil
}
func (r *rapidexReporterRepo) Delete(context.Context, int64) error {
	return nil
}
