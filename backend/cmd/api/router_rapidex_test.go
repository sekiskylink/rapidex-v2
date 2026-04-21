package main

import (
	"context"
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

type rapidexOrgUnitRepo struct{}

func (r *rapidexOrgUnitRepo) List(context.Context, orgunit.ListQuery) (orgunit.ListResult, error) {
	return orgunit.ListResult{Items: []orgunit.OrgUnit{}, Total: 0, Page: 0, PageSize: 20}, nil
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
func (r *rapidexReporterRepo) GetByID(context.Context, int64) (reporter.Reporter, error) {
	return reporter.Reporter{}, nil
}
func (r *rapidexReporterRepo) GetByUID(context.Context, string) (reporter.Reporter, error) {
	return reporter.Reporter{}, nil
}
func (r *rapidexReporterRepo) GetByContactUUID(context.Context, string) (reporter.Reporter, error) {
	return reporter.Reporter{}, nil
}
func (r *rapidexReporterRepo) GetByPhoneNumber(context.Context, string) (reporter.Reporter, error) {
	return reporter.Reporter{}, nil
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
