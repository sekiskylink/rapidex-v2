package request

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"basepro/backend/internal/auth"
	"github.com/gin-gonic/gin"
)

func newTestHandler() *Handler {
	repo := NewRepository()
	return NewHandler(NewService(repo))
}

func TestHandlerCreateRejectsInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newTestHandler()
	router := gin.New()
	router.POST("/requests", func(c *gin.Context) {
		c.Set(auth.PrincipalContextKey, auth.Principal{Type: "user", UserID: 1})
		handler.Create(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/requests", bytes.NewReader([]byte(`{`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandlerGetRejectsInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newTestHandler()
	router := gin.New()
	router.GET("/requests/:id", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/requests/nope", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandlerDeleteReturnsNoContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := NewRepository()
	service := NewService(repo)
	created, err := service.CreateRequest(context.Background(), CreateInput{
		DestinationServerID: 3,
		Payload:             []byte(`{"trackedEntity":"123"}`),
	})
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	handler := NewHandler(service)
	router := gin.New()
	router.DELETE("/requests/:id", func(c *gin.Context) {
		c.Set(auth.PrincipalContextKey, auth.Principal{Type: "user", UserID: 1})
		handler.Delete(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/requests/"+strconv.FormatInt(created.ID, 10), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d body=%s", w.Code, w.Body.String())
	}
	if _, err := service.GetRequest(context.Background(), created.ID); err == nil {
		t.Fatal("expected request to be deleted")
	}
}

func TestHandlerDeleteRejectsInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newTestHandler()
	router := gin.New()
	router.DELETE("/requests/:id", func(c *gin.Context) {
		c.Set(auth.PrincipalContextKey, auth.Principal{Type: "user", UserID: 1})
		handler.Delete(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/requests/nope", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandlerCreateReturnsAcceptAndPersistRequestState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := NewRepository()
	deliverySvc := &fakeRequestDeliveryService{repo: repo.(*memoryRepository)}
	handler := NewHandler(NewService(repo).WithDeliveryService(deliverySvc))
	router := gin.New()
	router.POST("/requests", func(c *gin.Context) {
		c.Set(auth.PrincipalContextKey, auth.Principal{Type: "user", UserID: 1})
		handler.Create(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/requests", bytes.NewReader([]byte(`{
		"destinationServerId": 3,
		"payload": {"trackedEntity":"123"}
	}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
	}

	var created Record
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if created.Status != StatusPending {
		t.Fatalf("expected pending request for worker pickup, got %+v", created)
	}
	if len(created.Targets) != 1 || created.Targets[0].Status != TargetStatusPending {
		t.Fatalf("expected pending target, got %+v", created.Targets)
	}
	if created.Targets[0].LatestDeliveryID == nil || created.Targets[0].LatestDeliveryStatus != "pending" {
		t.Fatalf("expected pending delivery metadata, got %+v", created.Targets[0])
	}
	if len(deliverySvc.created) != 1 {
		t.Fatalf("expected a single durable pending delivery, got %d", len(deliverySvc.created))
	}
}

func TestHandlerListReturnsPaginatedPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newTestHandler()
	service := handler.service
	_, _ = service.CreateRequest(nil, CreateInput{
		SourceSystem:        "emr",
		DestinationServerID: 3,
		CorrelationID:       "corr-1",
		Payload:             []byte(`{"trackedEntity":"123"}`),
	})

	router := gin.New()
	router.GET("/requests", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/requests?page=1&pageSize=25", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["totalCount"].(float64) != 1 {
		t.Fatalf("expected totalCount 1, got %+v", payload)
	}
}

func TestHandlerCreateExternalReturnsUIDOnlyContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := NewRepository().(*memoryRepository)
	deliverySvc := &fakeRequestDeliveryService{repo: repo}
	handler := NewHandler(NewService(repo).
		WithDeliveryService(deliverySvc).
		WithServerService(&fakeServerResolver{items: map[string]int64{"srv-primary": 3}}))
	router := gin.New()
	router.POST("/external/requests", func(c *gin.Context) {
		c.Set(auth.PrincipalContextKey, auth.Principal{Type: "api_token"})
		handler.CreateExternal(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/external/requests", bytes.NewReader([]byte(`{
		"sourceSystem": "emr",
		"destinationServerUid": "srv-primary",
		"idempotencyKey": "idem-1",
		"payload": {"trackedEntity":"123"}
	}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := payload["id"]; ok {
		t.Fatalf("expected external response to omit internal id, got %+v", payload)
	}
	if payload["uid"] == "" || payload["destinationServerUid"] != "server-uid-3" {
		t.Fatalf("unexpected external payload %+v", payload)
	}
}
