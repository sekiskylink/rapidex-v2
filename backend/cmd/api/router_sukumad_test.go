package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"basepro/backend/internal/audit"
	"basepro/backend/internal/auth"
	"basepro/backend/internal/rbac"
	asyncjobs "basepro/backend/internal/sukumad/async"
	"basepro/backend/internal/sukumad/dashboard"
	"basepro/backend/internal/sukumad/delivery"
	documentation "basepro/backend/internal/sukumad/documentation"
	"basepro/backend/internal/sukumad/observability"
	"basepro/backend/internal/sukumad/ratelimit"
	requests "basepro/backend/internal/sukumad/request"
	"basepro/backend/internal/sukumad/scheduler"
	"basepro/backend/internal/sukumad/server"
	"basepro/backend/internal/sukumad/worker"
	"golang.org/x/net/websocket"
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
		SchedulerHandler:     scheduler.NewHandler(scheduler.NewService(scheduler.NewRepository())),
		DeliveryHandler:      delivery.NewHandler(delivery.NewService(delivery.NewRepository())),
		AsyncHandler:         asyncjobs.NewHandler(asyncService),
		ObservabilityHandler: observability.NewHandler(observability.NewService(observability.NewRepository(nil, workerService, rateLimitService))),
		DashboardHandler: dashboard.NewHandler(dashboard.NewService(&dashboardTestRepository{
			snapshot: dashboard.Snapshot{
				GeneratedAt: time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC),
				Health:      dashboard.Health{Status: "ok"},
				KPIs:        dashboard.KPIs{RequestsToday: 1},
			},
		})),
		DocumentationHandler: documentation.NewHandler(documentation.NewService(func() documentation.SourceConfig {
			return documentation.SourceConfig{}
		})),
	}
}

type dashboardTestRepository struct {
	snapshot dashboard.Snapshot
}

func (r *dashboardTestRepository) GetSnapshot(_ context.Context, _ time.Time) (dashboard.Snapshot, error) {
	return r.snapshot, nil
}

type apiTokenRepo struct {
	tokens      map[string]*auth.APIToken
	permissions map[int64][]auth.APITokenPermission
}

func newAPITokenRepo() *apiTokenRepo {
	return &apiTokenRepo{
		tokens:      map[string]*auth.APIToken{},
		permissions: map[int64][]auth.APITokenPermission{},
	}
}

func (r *apiTokenRepo) GetUserByUsername(context.Context, string) (*auth.User, error) {
	return nil, auth.ErrNotFound
}
func (r *apiTokenRepo) GetUserByID(context.Context, int64) (*auth.User, error) {
	return nil, auth.ErrNotFound
}
func (r *apiTokenRepo) GetActiveUserByIdentifier(context.Context, string) (*auth.User, error) {
	return nil, auth.ErrNotFound
}
func (r *apiTokenRepo) GetRefreshTokenByHash(context.Context, string) (*auth.RefreshToken, error) {
	return nil, auth.ErrNotFound
}
func (r *apiTokenRepo) CreateRefreshToken(context.Context, auth.RefreshToken) (*auth.RefreshToken, error) {
	return nil, auth.ErrNotFound
}
func (r *apiTokenRepo) RevokeRefreshToken(context.Context, int64, *int64, time.Time) error {
	return nil
}
func (r *apiTokenRepo) RevokeAllActiveRefreshTokensForUser(context.Context, int64, time.Time) error {
	return nil
}
func (r *apiTokenRepo) UpdateUserLastLoginAt(context.Context, int64, time.Time) error { return nil }
func (r *apiTokenRepo) CreatePasswordResetToken(context.Context, auth.PasswordResetToken) (*auth.PasswordResetToken, error) {
	return nil, auth.ErrNotFound
}
func (r *apiTokenRepo) GetPasswordResetTokenByHash(context.Context, string) (*auth.PasswordResetToken, error) {
	return nil, auth.ErrNotFound
}
func (r *apiTokenRepo) InvalidateActivePasswordResetTokensForUser(context.Context, int64, time.Time) error {
	return nil
}
func (r *apiTokenRepo) MarkPasswordResetTokenUsed(context.Context, int64, time.Time, *string, *string) error {
	return auth.ErrNotFound
}
func (r *apiTokenRepo) UpdateUserPasswordHash(context.Context, int64, string, time.Time) error {
	return auth.ErrNotFound
}
func (r *apiTokenRepo) CreateAPIToken(context.Context, auth.APIToken, []string, *string) (*auth.APIToken, error) {
	return nil, auth.ErrNotFound
}
func (r *apiTokenRepo) ListAPITokens(context.Context) ([]auth.APIToken, error) { return nil, nil }
func (r *apiTokenRepo) GetAPITokenByID(context.Context, int64) (*auth.APIToken, error) {
	return nil, auth.ErrNotFound
}
func (r *apiTokenRepo) GetAPITokenByHash(_ context.Context, hash string) (*auth.APIToken, error) {
	token, ok := r.tokens[hash]
	if !ok {
		return nil, auth.ErrNotFound
	}
	copy := *token
	return &copy, nil
}
func (r *apiTokenRepo) GetAPITokenPermissions(_ context.Context, tokenID int64) ([]auth.APITokenPermission, error) {
	return append([]auth.APITokenPermission{}, r.permissions[tokenID]...), nil
}
func (r *apiTokenRepo) RevokeAPIToken(context.Context, int64, time.Time) error { return nil }
func (r *apiTokenRepo) UpdateAPITokenLastUsed(_ context.Context, tokenID int64, now time.Time) error {
	for _, token := range r.tokens {
		if token.ID == tokenID {
			copyNow := now
			token.LastUsedAt = &copyNow
			return nil
		}
	}
	return auth.ErrNotFound
}
func (r *apiTokenRepo) EnsureUser(context.Context, string, string, bool) (*auth.User, error) {
	return nil, auth.ErrNotFound
}

func TestDashboardOperationsEventsRequiresObservabilityRead(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(111, "dashboard-reader", time.Now().UTC())
	service := dashboard.NewService(&dashboardTestRepository{})

	deps := newSukumadTestAppDeps(jwt, rbacServiceWithPermissions(map[int64][]string{
		111: {},
	}))
	deps.DashboardHandler = dashboard.NewHandler(service)
	server := httptest.NewServer(newRouter(deps))
	defer server.Close()

	wsURL := "ws" + server.URL[len("http"):] + "/api/v1/dashboard/operations/events"
	config, err := websocket.NewConfig(wsURL, "http://example.com")
	if err != nil {
		t.Fatalf("create websocket config: %v", err)
	}
	config.Location.RawQuery = "access_token=" + token

	conn, err := websocket.DialConfig(config)
	if err == nil {
		conn.Close()
		t.Fatal("expected websocket dial to fail without observability.read")
	}
}

func TestDashboardOperationsEventsStreamsPublishedEvent(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(112, "dashboard-observer", time.Now().UTC())
	service := dashboard.NewService(&dashboardTestRepository{})

	deps := newSukumadTestAppDeps(jwt, rbacServiceWithPermissions(map[int64][]string{
		112: {rbac.PermissionObservabilityRead},
	}))
	deps.DashboardHandler = dashboard.NewHandler(service)
	server := httptest.NewServer(newRouter(deps))
	defer server.Close()

	wsURL := "ws" + server.URL[len("http"):] + "/api/v1/dashboard/operations/events"
	config, err := websocket.NewConfig(wsURL, "http://example.com")
	if err != nil {
		t.Fatalf("create websocket config: %v", err)
	}
	config.Location.RawQuery = "access_token=" + token

	conn, err := websocket.DialConfig(config)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	workerID := int64(5)
	service.PublishSourceEvent(context.Background(), dashboard.SourceEvent{
		Type:      "worker.heartbeat",
		Timestamp: time.Date(2026, 3, 18, 12, 5, 0, 0, time.UTC),
		Severity:  "info",
		Message:   "Worker heartbeat",
		WorkerID:  &workerID,
		WorkerUID: "wrk_5",
	})

	var event dashboard.StreamEvent
	if err := websocket.JSON.Receive(conn, &event); err != nil {
		t.Fatalf("receive websocket event: %v", err)
	}
	if event.Type != "worker.heartbeat" || event.EntityType != "worker" || event.EntityID != workerID {
		t.Fatalf("unexpected event payload: %+v", event)
	}
}

func TestDocumentationRoutesRequireAuthenticationAndServeConfiguredMarkdown(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "overview.md"), []byte("# Overview\n\nBody"), 0o600); err != nil {
		t.Fatalf("write doc: %v", err)
	}

	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(115, "docs-reader", time.Now().UTC())
	deps := newSukumadTestAppDeps(jwt, rbacServiceWithPermissions(map[int64][]string{
		115: {},
	}))
	deps.DocumentationHandler = documentation.NewHandler(documentation.NewService(func() documentation.SourceConfig {
		return documentation.SourceConfig{
			RootPath: root,
			Files: []documentation.SourceFile{
				{Slug: "overview", Title: "Overview", Path: "overview.md"},
			},
		}
	}))
	router := newRouter(deps)

	unauthReq := httptest.NewRequest(http.MethodGet, "/api/v1/documentation", nil)
	unauthW := httptest.NewRecorder()
	router.ServeHTTP(unauthW, unauthReq)
	if unauthW.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d body=%s", unauthW.Code, unauthW.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/documentation", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listW := httptest.NewRecorder()
	router.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200 from list, got %d body=%s", listW.Code, listW.Body.String())
	}

	var list struct {
		Items []documentation.DocumentSummary `json:"items"`
	}
	if err := json.Unmarshal(listW.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list.Items) != 1 || list.Items[0].Slug != "overview" {
		t.Fatalf("unexpected list response: %+v", list)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/documentation/overview", nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200 from get, got %d body=%s", getW.Code, getW.Body.String())
	}

	var detail documentation.DocumentDetail
	if err := json.Unmarshal(getW.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if detail.Content != "# Overview\n\nBody" {
		t.Fatalf("unexpected detail response: %+v", detail)
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

func TestExternalServerListRouteAcceptsAPITokenAndReturnsSafeFields(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	rbacService := rbacServiceWithPermissions(map[int64][]string{
		91: {
			rbac.PermissionServersRead,
			rbac.PermissionServersWrite,
		},
	})
	serverHandler := server.NewHandler(server.NewService(server.NewRepository(), audit.NewService(&fakeAuditRepo{})))

	deps := newSukumadTestAppDeps(jwt, rbacService)
	deps.ServerHandler = serverHandler

	tokenRepo := newAPITokenRepo()
	secret := "test-secret"
	plain := "bpt_serversread"
	hash := auth.HashAPIToken(secret, plain)
	tokenRepo.tokens[hash] = &auth.APIToken{
		ID:        21,
		Name:      "server-reader",
		TokenHash: hash,
		Prefix:    auth.APITokenPrefix(plain),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	tokenRepo.permissions[21] = []auth.APITokenPermission{{APITokenID: 21, Permission: rbac.PermissionServersRead}}
	deps.AuthService = auth.NewService(tokenRepo, nil, jwt, nil, time.Minute, time.Hour, time.Hour, secret, true, 4)
	deps.APITokenHeaderName = "X-API-Token"

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
	userToken, _, _ := jwt.GenerateAccessToken(91, "sukumad-reader", time.Now().UTC())
	createReq.Header.Set("Authorization", "Bearer "+userToken)
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	router.ServeHTTP(createW, createReq)
	if createW.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating server, got %d body=%s", createW.Code, createW.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/external/servers?page=1&pageSize=25", nil)
	req.Header.Set("X-API-Token", plain)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var body struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Items) != 1 {
		t.Fatalf("expected 1 item, got %+v", body.Items)
	}
	item := body.Items[0]
	if item["uid"] == "" || item["code"] != "dhis2-prod" || item["name"] != "DHIS2 Production" {
		t.Fatalf("unexpected item payload: %+v", item)
	}
	if _, ok := item["id"]; ok {
		t.Fatalf("expected external payload to omit id: %+v", item)
	}
	if _, ok := item["baseUrl"]; ok {
		t.Fatalf("expected external payload to omit baseUrl: %+v", item)
	}
	if _, ok := item["headers"]; ok {
		t.Fatalf("expected external payload to omit headers: %+v", item)
	}
}

func TestExternalServerListRouteRejectsTokenWithoutServersRead(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	rbacService := rbacServiceWithPermissions(nil)

	deps := newSukumadTestAppDeps(jwt, rbacService)
	deps.ServerHandler = server.NewHandler(server.NewService(server.NewRepository(), audit.NewService(&fakeAuditRepo{})))

	tokenRepo := newAPITokenRepo()
	secret := "test-secret"
	plain := "bpt_noserversread"
	hash := auth.HashAPIToken(secret, plain)
	tokenRepo.tokens[hash] = &auth.APIToken{
		ID:        22,
		Name:      "denied-reader",
		TokenHash: hash,
		Prefix:    auth.APITokenPrefix(plain),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	tokenRepo.permissions[22] = []auth.APITokenPermission{{APITokenID: 22, Permission: rbac.PermissionRequestsRead}}
	deps.AuthService = auth.NewService(tokenRepo, nil, jwt, nil, time.Minute, time.Hour, time.Hour, secret, true, 4)
	deps.APITokenHeaderName = "X-API-Token"

	router := newRouter(deps)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/external/servers", nil)
	req.Header.Set("X-API-Token", plain)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestExternalServerListRouteRequiresAuthentication(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	rbacService := rbacServiceWithPermissions(nil)

	deps := newSukumadTestAppDeps(jwt, rbacService)
	deps.ServerHandler = server.NewHandler(server.NewService(server.NewRepository(), audit.NewService(&fakeAuditRepo{})))

	router := newRouter(deps)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/external/servers", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestExternalRequestSummaryRouteUsesSummaryHandlerPath(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, err := jwt.GenerateAccessToken(11, "summary-reader", time.Now().UTC())
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}

	deps := newSukumadTestAppDeps(jwt, rbacServiceWithPermissions(map[int64][]string{
		11: {rbac.PermissionRequestsRead},
	}))
	deps.RequestHandler = requests.NewHandler(requests.NewService(requests.NewRepository()))
	router := newRouter(deps)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/external/requests/summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 from summary handler validation, got %d body=%s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("destinationServerUid")) {
		t.Fatalf("expected summary validation details, got %s", w.Body.String())
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
		"/api/v1/dashboard/operations",
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

func TestDashboardOperationsRouteReturnsSnapshot(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(101, "dashboard-reader", time.Now().UTC())
	rbacService := rbacServiceWithPermissions(map[int64][]string{
		101: {rbac.PermissionObservabilityRead},
	})

	router := newRouter(newSukumadTestAppDeps(jwt, rbacService))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/operations", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var snapshot map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	if _, ok := snapshot["kpis"]; !ok {
		t.Fatalf("expected kpis in response, got %+v", snapshot)
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

func TestSchedulerRoutesRequireSchedulerPermissions(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(211, "scheduler-reader", time.Now().UTC())
	rbacService := rbacServiceWithPermissions(map[int64][]string{
		211: {rbac.PermissionRequestsRead},
	})

	router := newRouter(newSukumadTestAppDeps(jwt, rbacService))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scheduler/jobs?page=1&pageSize=25", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for missing scheduler.read permission, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestSchedulerRoutesListCreateAndRunNow(t *testing.T) {
	jwt := auth.NewJWTManager("jwt-secret", time.Minute)
	token, _, _ := jwt.GenerateAccessToken(212, "scheduler-admin", time.Now().UTC())
	rbacService := rbacServiceWithPermissions(map[int64][]string{
		212: {rbac.PermissionSchedulerRead, rbac.PermissionSchedulerWrite},
	})

	router := newRouter(newSukumadTestAppDeps(jwt, rbacService))

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/scheduler/jobs", bytes.NewBufferString(`{
		"code":"nightly-sync",
		"name":"Nightly Sync",
		"description":"Nightly integration sync",
		"jobCategory":"integration",
		"jobType":"dhis2.sync",
		"scheduleType":"interval",
		"scheduleExpr":"15m",
		"timezone":"UTC",
		"enabled":true,
		"allowConcurrentRuns":false,
		"config":{"serverCode":"dhis2"}
	}`))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	router.ServeHTTP(createW, createReq)
	if createW.Code != http.StatusCreated {
		t.Fatalf("expected 201 create, got %d body=%s", createW.Code, createW.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/scheduler/jobs?page=1&pageSize=25", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listW := httptest.NewRecorder()
	router.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200 list, got %d body=%s", listW.Code, listW.Body.String())
	}

	runReq := httptest.NewRequest(http.MethodPost, "/api/v1/scheduler/jobs/1/run-now", nil)
	runReq.Header.Set("Authorization", "Bearer "+token)
	runW := httptest.NewRecorder()
	router.ServeHTTP(runW, runReq)
	if runW.Code != http.StatusCreated {
		t.Fatalf("expected 201 run-now, got %d body=%s", runW.Code, runW.Body.String())
	}

	runsReq := httptest.NewRequest(http.MethodGet, "/api/v1/scheduler/jobs/1/runs?page=1&pageSize=25", nil)
	runsReq.Header.Set("Authorization", "Bearer "+token)
	runsW := httptest.NewRecorder()
	router.ServeHTTP(runsW, runsReq)
	if runsW.Code != http.StatusOK {
		t.Fatalf("expected 200 runs list, got %d body=%s", runsW.Code, runsW.Body.String())
	}
}
