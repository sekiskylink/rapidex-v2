package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/websocket"
)

func TestHandlerGetOperationsReturnsSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fixed := time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC)
	handler := NewHandler(NewService(&stubRepository{
		snapshot: Snapshot{
			GeneratedAt: fixed,
			Health:      Health{Status: "ok"},
			KPIs:        KPIs{RequestsToday: 3},
		},
	}).WithClock(func() time.Time { return fixed }))

	router := gin.New()
	router.GET("/dashboard/operations", handler.GetOperations)

	req := httptest.NewRequest(http.MethodGet, "/dashboard/operations", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var snapshot Snapshot
	if err := json.Unmarshal(w.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	if snapshot.KPIs.RequestsToday != 3 {
		t.Fatalf("expected requestsToday 3, got %+v", snapshot.KPIs)
	}
	if snapshot.Health.Status != "ok" {
		t.Fatalf("expected health ok, got %+v", snapshot.Health)
	}
}

func TestHandlerStreamOperationsPublishesEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := NewService(&stubRepository{})
	handler := NewHandler(service)

	router := gin.New()
	router.GET("/dashboard/operations/events", handler.StreamOperations)

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + server.URL[len("http"):] + "/dashboard/operations/events"
	config, err := websocket.NewConfig(wsURL, "http://example.com")
	if err != nil {
		t.Fatalf("create websocket config: %v", err)
	}

	conn, err := websocket.DialConfig(config)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	requestID := int64(7)
	service.PublishSourceEvent(context.Background(), SourceEvent{
		Type:       "request.created",
		Timestamp:  time.Date(2026, 3, 18, 11, 0, 0, 0, time.UTC),
		Severity:   "info",
		Message:    "Request accepted",
		RequestID:  &requestID,
		RequestUID: "req_7",
	})

	var event StreamEvent
	if err := websocket.JSON.Receive(conn, &event); err != nil {
		t.Fatalf("receive stream event: %v", err)
	}
	if event.Type != "request.created" || event.EntityType != "request" || event.EntityID != requestID {
		t.Fatalf("unexpected stream event: %+v", event)
	}
}
