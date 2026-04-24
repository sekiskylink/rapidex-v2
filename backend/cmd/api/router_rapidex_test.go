package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"basepro/backend/internal/auth"
	"basepro/backend/internal/rbac"
	"basepro/backend/internal/sukumad/orgunit"
	"basepro/backend/internal/sukumad/reporter"
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

	for _, path := range []string{"/api/v1/reporters/1/rapidpro-contact", "/api/v1/reporters/1/chat-history"} {
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
