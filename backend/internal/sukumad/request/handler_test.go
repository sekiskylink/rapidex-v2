package request

import (
	"bytes"
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
