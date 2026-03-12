package delivery

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"basepro/backend/internal/auth"
	"github.com/gin-gonic/gin"
)

func newTestHandler() *Handler {
	repo := NewRepository()
	return NewHandler(NewService(repo))
}

func TestHandlerGetRejectsInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newTestHandler()
	router := gin.New()
	router.GET("/deliveries/:id", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/deliveries/nope", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandlerListReturnsPaginatedPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newTestHandler()
	_, err := handler.service.CreatePendingDelivery(nil, CreateInput{
		RequestID: 9,
		ServerID:  3,
	})
	if err != nil {
		t.Fatalf("seed delivery: %v", err)
	}

	router := gin.New()
	router.GET("/deliveries", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/deliveries?page=1&pageSize=25", nil)
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

func TestHandlerRetryReturnsCreatedRetryAttempt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newTestHandler()
	created, err := handler.service.CreatePendingDelivery(nil, CreateInput{
		RequestID: 14,
		ServerID:  6,
	})
	if err != nil {
		t.Fatalf("create delivery: %v", err)
	}
	if _, err := handler.service.MarkRunning(nil, created.ID); err != nil {
		t.Fatalf("mark running: %v", err)
	}
	if _, err := handler.service.MarkFailed(nil, CompletionInput{
		ID:           created.ID,
		ErrorMessage: "upstream failed",
	}); err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	router := gin.New()
	router.POST("/deliveries/:id/retry", func(c *gin.Context) {
		c.Set(auth.PrincipalContextKey, auth.Principal{Type: "user", UserID: 1})
		handler.Retry(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/deliveries/1/retry", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["status"] != StatusRetrying || payload["attemptNumber"].(float64) != 2 {
		t.Fatalf("unexpected retry payload: %+v", payload)
	}
}
