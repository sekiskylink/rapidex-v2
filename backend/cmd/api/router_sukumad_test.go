package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"basepro/backend/internal/audit"
	"basepro/backend/internal/auth"
	"basepro/backend/internal/rbac"
	asyncjobs "basepro/backend/internal/sukumad/async"
	"basepro/backend/internal/sukumad/delivery"
	"basepro/backend/internal/sukumad/observability"
	"basepro/backend/internal/sukumad/ratelimit"
	requests "basepro/backend/internal/sukumad/request"
	"basepro/backend/internal/sukumad/server"
	"basepro/backend/internal/sukumad/worker"
)

func newSukumadTestAppDeps(jwt *auth.JWTManager, rbacService *rbac.Service) AppDeps {
	asyncService := asyncjobs.NewService(asyncjobs.NewRepository(), audit.NewService(&fakeAuditRepo{}))
	workerService := worker.NewService(worker.NewRepository(), audit.NewService(&fakeAuditRepo{}))
	rateLimitService := ratelimit.NewService(ratelimit.NewRepository())
	return AppDeps{
		JWTManager:           jwt,
		RBACService:          rbacService,
		ServerHandler:        server.NewHandler(server.NewService(server.NewRepository())),
		RequestHandler:       requests.NewHandler(requests.NewService(requests.NewRepository())),
		DeliveryHandler:      delivery.NewHandler(delivery.NewService(delivery.NewRepository())),
		AsyncHandler:         asyncjobs.NewHandler(asyncService),
		ObservabilityHandler: observability.NewHandler(observability.NewService(observability.NewRepository(nil, workerService, rateLimitService))),
	}
}

func TestServerRoutesCRUD(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(91, "sukumad-reader", time.Now().UTC())
	rbacService := rbacServiceWithPermissions(map[int64][]string{
		91: {
			rbac.PermissionServersRead,
			rbac.PermissionServersWrite,
		},
	})
	serverHandler := server.NewHandler(server.NewService(server.NewRepository(), audit.NewService(&fakeAuditRepo{})))

	deps := newSukumadTestAppDeps(jwt, rbacService)
	deps.ServerHandler = serverHandler
	router := newRouter(deps)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/servers", bytes.NewReader([]byte(`{
		"name":"DHIS2 Production",
		"code":"dhis2-prod",
		"systemType":"dhis2",
		"baseUrl":"https://dhis.example.com",
		"endpointType":"http",
		"httpMethod":"post",
		"useAsync":true,
		"parseResponses":true,
		"headers":{"Authorization":"Bearer masked"},
		"urlParams":{"orgUnit":"OU_123"},
		"suspended":false
	}`)))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	router.ServeHTTP(createW, createReq)

	if createW.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", createW.Code, createW.Body.String())
	}

	var created map[string]any
	if err := json.Unmarshal(createW.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	id := int(created["id"].(float64))
	idStr := strconv.Itoa(id)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/servers?page=1&pageSize=25", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listW := httptest.NewRecorder()
	router.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200 from list, got %d body=%s", listW.Code, listW.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+idStr, nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200 from get, got %d body=%s", getW.Code, getW.Body.String())
	}

	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/servers/"+idStr, bytes.NewReader([]byte(`{
		"name":"DHIS2 Production",
		"code":"dhis2-prod",
		"systemType":"dhis2",
		"baseUrl":"https://dhis.example.com/api",
		"endpointType":"http",
		"httpMethod":"post",
		"useAsync":true,
		"parseResponses":true,
		"headers":{"Authorization":"Bearer masked"},
		"urlParams":{"orgUnit":"OU_123"},
		"suspended":true
	}`)))
	updateReq.Header.Set("Authorization", "Bearer "+token)
	updateReq.Header.Set("Content-Type", "application/json")
	updateW := httptest.NewRecorder()
	router.ServeHTTP(updateW, updateReq)
	if updateW.Code != http.StatusOK {
		t.Fatalf("expected 200 from update, got %d body=%s", updateW.Code, updateW.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/servers/"+idStr, nil)
	deleteReq.Header.Set("Authorization", "Bearer "+token)
	deleteW := httptest.NewRecorder()
	router.ServeHTTP(deleteW, deleteReq)
	if deleteW.Code != http.StatusNoContent {
		t.Fatalf("expected 204 from delete, got %d body=%s", deleteW.Code, deleteW.Body.String())
	}
}

func TestRequestRoutesCreateListAndGet(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(95, "request-writer", time.Now().UTC())
	rbacService := rbacServiceWithPermissions(map[int64][]string{
		95: {
			rbac.PermissionRequestsRead,
			rbac.PermissionRequestsWrite,
			rbac.PermissionDeliveriesRead,
		},
	})
	deliveryService := delivery.NewService(delivery.NewRepository(), audit.NewService(&fakeAuditRepo{}))
	requestHandler := requests.NewHandler(requests.NewService(requests.NewRepository(), audit.NewService(&fakeAuditRepo{})).WithDeliveryService(deliveryService))

	deps := newSukumadTestAppDeps(jwt, rbacService)
	deps.RequestHandler = requestHandler
	deps.DeliveryHandler = delivery.NewHandler(deliveryService)
	router := newRouter(deps)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/requests", bytes.NewReader([]byte(`{
		"sourceSystem":"emr",
		"destinationServerId":3,
		"correlationId":"corr-123",
		"payload":{"trackedEntity":"123"},
		"metadata":{"priority":"high"}
	}`)))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	router.ServeHTTP(createW, createReq)

	if createW.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", createW.Code, createW.Body.String())
	}

	var created map[string]any
	if err := json.Unmarshal(createW.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	id := int(created["id"].(float64))
	idStr := strconv.Itoa(id)
	if created["status"] != requests.StatusPending {
		t.Fatalf("expected pending status, got %+v", created)
	}
	if _, ok := created["uid"].(string); !ok {
		t.Fatalf("expected uid in response, got %+v", created)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/requests?page=1&pageSize=25", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listW := httptest.NewRecorder()
	router.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200 from list, got %d body=%s", listW.Code, listW.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/requests/"+idStr, nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200 from get, got %d body=%s", getW.Code, getW.Body.String())
	}

	deliveryListReq := httptest.NewRequest(http.MethodGet, "/api/v1/deliveries?page=1&pageSize=25", nil)
	deliveryListReq.Header.Set("Authorization", "Bearer "+token)
	deliveryListW := httptest.NewRecorder()
	router.ServeHTTP(deliveryListW, deliveryListReq)
	if deliveryListW.Code != http.StatusOK {
		t.Fatalf("expected 200 from delivery list, got %d body=%s", deliveryListW.Code, deliveryListW.Body.String())
	}
}

func TestDeliveryRoutesListGetAndRetry(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(97, "delivery-operator", time.Now().UTC())
	rbacService := rbacServiceWithPermissions(map[int64][]string{
		97: {
			rbac.PermissionDeliveriesRead,
			rbac.PermissionDeliveriesWrite,
		},
	})
	deliveryService := delivery.NewService(delivery.NewRepository(), audit.NewService(&fakeAuditRepo{}))
	created, err := deliveryService.CreatePendingDelivery(nil, delivery.CreateInput{
		RequestID: 11,
		ServerID:  7,
	})
	if err != nil {
		t.Fatalf("seed delivery: %v", err)
	}
	if _, err := deliveryService.MarkRunning(nil, created.ID); err != nil {
		t.Fatalf("mark running: %v", err)
	}
	if _, err := deliveryService.MarkFailed(nil, delivery.CompletionInput{
		ID:           created.ID,
		ErrorMessage: "timeout",
	}); err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	deps := newSukumadTestAppDeps(jwt, rbacService)
	deps.DeliveryHandler = delivery.NewHandler(deliveryService)
	router := newRouter(deps)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/deliveries?page=1&pageSize=25", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listW := httptest.NewRecorder()
	router.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200 from delivery list, got %d body=%s", listW.Code, listW.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/deliveries/1", nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200 from delivery get, got %d body=%s", getW.Code, getW.Body.String())
	}

	retryReq := httptest.NewRequest(http.MethodPost, "/api/v1/deliveries/1/retry", nil)
	retryReq.Header.Set("Authorization", "Bearer "+token)
	retryW := httptest.NewRecorder()
	router.ServeHTTP(retryW, retryReq)
	if retryW.Code != http.StatusCreated {
		t.Fatalf("expected 201 from retry, got %d body=%s", retryW.Code, retryW.Body.String())
	}
}

func TestJobsAndObservabilityRoutes(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(99, "ops-reader", time.Now().UTC())
	rbacService := rbacServiceWithPermissions(map[int64][]string{
		99: {
			rbac.PermissionJobsRead,
			rbac.PermissionObservabilityRead,
		},
	})

	asyncService := asyncjobs.NewService(asyncjobs.NewRepository(), audit.NewService(&fakeAuditRepo{}))
	workerService := worker.NewService(worker.NewRepository(), audit.NewService(&fakeAuditRepo{}))
	rateLimitService := ratelimit.NewService(ratelimit.NewRepository())

	job, err := asyncService.CreateTask(nil, asyncjobs.CreateInput{
		DeliveryAttemptID: 21,
		RemoteJobID:       "remote-21",
		RemoteStatus:      asyncjobs.StatePolling,
	})
	if err != nil {
		t.Fatalf("seed async task: %v", err)
	}
	if _, err := asyncService.RecordPoll(nil, asyncjobs.RecordPollInput{
		AsyncTaskID:  job.ID,
		RemoteStatus: asyncjobs.StatePolling,
		ResponseBody: `{"state":"processing"}`,
	}); err != nil {
		t.Fatalf("seed poll: %v", err)
	}

	if _, err := workerService.StartRun(nil, worker.Definition{Type: worker.TypePoll, Name: "poll-worker"}); err != nil {
		t.Fatalf("seed worker run: %v", err)
	}
	if _, err := rateLimitService.CreatePolicy(nil, ratelimit.CreateParams{
		Name:           "Global",
		ScopeType:      "global",
		RPS:            10,
		Burst:          20,
		MaxConcurrency: 2,
		TimeoutMS:      500,
		IsActive:       true,
	}); err != nil {
		t.Fatalf("seed rate limit: %v", err)
	}

	deps := newSukumadTestAppDeps(jwt, rbacService)
	deps.AsyncHandler = asyncjobs.NewHandler(asyncService)
	deps.ObservabilityHandler = observability.NewHandler(observability.NewService(observability.NewRepository(nil, workerService, rateLimitService)))
	router := newRouter(deps)

	for _, path := range []string{
		"/api/v1/jobs?page=1&pageSize=25",
		"/api/v1/jobs/1",
		"/api/v1/jobs/1/polls?page=1&pageSize=25",
		"/api/v1/observability/workers?page=1&pageSize=25",
		"/api/v1/observability/workers/1",
		"/api/v1/observability/rate-limits?page=1&pageSize=25",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d body=%s", path, w.Code, w.Body.String())
		}
	}
}

func TestSukumadRoutesRequireMatchingReadPermission(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(92, "limited-reader", time.Now().UTC())
	rbacService := rbacServiceWithPermissions(map[int64][]string{
		92: {rbac.PermissionServersRead},
	})

	router := newRouter(newSukumadTestAppDeps(jwt, rbacService))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/requests", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for missing requests.read permission, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestDeliveryRoutesRequireWritePermissionForRetry(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(98, "delivery-reader", time.Now().UTC())
	rbacService := rbacServiceWithPermissions(map[int64][]string{
		98: {rbac.PermissionDeliveriesRead},
	})
	deliveryService := delivery.NewService(delivery.NewRepository(), audit.NewService(&fakeAuditRepo{}))
	created, err := deliveryService.CreatePendingDelivery(nil, delivery.CreateInput{
		RequestID: 2,
		ServerID:  4,
	})
	if err != nil {
		t.Fatalf("seed delivery: %v", err)
	}
	if _, err := deliveryService.MarkRunning(nil, created.ID); err != nil {
		t.Fatalf("mark running: %v", err)
	}
	if _, err := deliveryService.MarkFailed(nil, delivery.CompletionInput{
		ID:           created.ID,
		ErrorMessage: "failed",
	}); err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	deps := newSukumadTestAppDeps(jwt, rbacService)
	deps.DeliveryHandler = delivery.NewHandler(deliveryService)
	router := newRouter(deps)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/deliveries/1/retry", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for missing deliveries.write permission, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestJobsAndObservabilityRoutesRequireReadPermissions(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(100, "limited-ops", time.Now().UTC())
	rbacService := rbacServiceWithPermissions(map[int64][]string{
		100: {rbac.PermissionServersRead},
	})

	router := newRouter(newSukumadTestAppDeps(jwt, rbacService))

	for _, path := range []string{
		"/api/v1/jobs",
		"/api/v1/jobs/1/events",
		"/api/v1/requests/1/events",
		"/api/v1/deliveries/1/events",
		"/api/v1/observability/workers",
		"/api/v1/observability/rate-limits",
		"/api/v1/observability/events",
		"/api/v1/observability/trace?correlationId=test",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for %s, got %d body=%s", path, w.Code, w.Body.String())
		}
	}
}

func TestServerRoutesRequireWritePermissionForMutation(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(93, "server-reader", time.Now().UTC())
	rbacService := rbacServiceWithPermissions(map[int64][]string{
		93: {rbac.PermissionServersRead},
	})

	router := newRouter(newSukumadTestAppDeps(jwt, rbacService))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/servers", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for missing servers.write permission, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestRequestRoutesRequireWritePermissionForMutation(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(96, "request-reader", time.Now().UTC())
	rbacService := rbacServiceWithPermissions(map[int64][]string{
		96: {rbac.PermissionRequestsRead},
	})

	router := newRouter(newSukumadTestAppDeps(jwt, rbacService))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/requests", bytes.NewReader([]byte(`{"destinationServerId":1,"payload":{}}`)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for missing requests.write permission, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestServerRoutesReturnValidationErrors(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(94, "server-writer", time.Now().UTC())
	rbacService := rbacServiceWithPermissions(map[int64][]string{
		94: {rbac.PermissionServersRead, rbac.PermissionServersWrite},
	})

	router := newRouter(newSukumadTestAppDeps(jwt, rbacService))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/servers", bytes.NewReader([]byte(`{
		"name":"",
		"code":"BAD CODE",
		"systemType":"",
		"baseUrl":"bad-url",
		"endpointType":"",
		"httpMethod":"TRACE",
		"useAsync":false,
		"parseResponses":true,
		"headers":{},
		"urlParams":{},
		"suspended":false
	}`)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 validation error, got %d body=%s", w.Code, w.Body.String())
	}
}
