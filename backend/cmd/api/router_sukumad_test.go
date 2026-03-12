package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"basepro/backend/internal/auth"
	"basepro/backend/internal/rbac"
	"basepro/backend/internal/sukumad/delivery"
	"basepro/backend/internal/sukumad/observability"
	requests "basepro/backend/internal/sukumad/request"
	"basepro/backend/internal/sukumad/server"
	"basepro/backend/internal/sukumad/worker"
)

func TestSukumadRoutesReturnPlaceholderResponses(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(91, "sukumad-reader", time.Now().UTC())
	rbacService := rbacServiceWithPermissions(map[int64][]string{
		91: {
			rbac.PermissionServersRead,
			rbac.PermissionRequestsRead,
			rbac.PermissionDeliveriesRead,
			rbac.PermissionJobsRead,
			rbac.PermissionObservabilityRead,
		},
	})

	router := newRouter(AppDeps{
		JWTManager:           jwt,
		RBACService:          rbacService,
		ServerHandler:        server.NewHandler(server.NewService(server.NewRepository())),
		RequestHandler:       requests.NewHandler(requests.NewService(requests.NewRepository())),
		DeliveryHandler:      delivery.NewHandler(delivery.NewService(delivery.NewRepository())),
		WorkerHandler:        worker.NewHandler(worker.NewService(worker.NewRepository())),
		ObservabilityHandler: observability.NewHandler(observability.NewService(observability.NewRepository())),
	})

	for _, path := range []string{
		"/api/v1/servers",
		"/api/v1/requests",
		"/api/v1/deliveries",
		"/api/v1/jobs",
		"/api/v1/observability",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d body=%s", path, w.Code, w.Body.String())
		}

		var body map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode %s response: %v", path, err)
		}
		if body["message"] != "not implemented" {
			t.Fatalf("expected placeholder response for %s, got %q", path, body["message"])
		}
	}
}

func TestSukumadRoutesRequireMatchingReadPermission(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(92, "limited-reader", time.Now().UTC())
	rbacService := rbacServiceWithPermissions(map[int64][]string{
		92: {rbac.PermissionServersRead},
	})

	router := newRouter(AppDeps{
		JWTManager:           jwt,
		RBACService:          rbacService,
		ServerHandler:        server.NewHandler(server.NewService(server.NewRepository())),
		RequestHandler:       requests.NewHandler(requests.NewService(requests.NewRepository())),
		DeliveryHandler:      delivery.NewHandler(delivery.NewService(delivery.NewRepository())),
		WorkerHandler:        worker.NewHandler(worker.NewService(worker.NewRepository())),
		ObservabilityHandler: observability.NewHandler(observability.NewService(observability.NewRepository())),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/requests", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for missing requests.read permission, got %d body=%s", w.Code, w.Body.String())
	}
}
