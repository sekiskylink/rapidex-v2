package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"basepro/backend/internal/logging"
	"github.com/gin-gonic/gin"
)

func TestRequestIDGeneratedWhenMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	requestID := strings.TrimSpace(w.Header().Get(RequestIDHeader))
	if requestID == "" {
		t.Fatal("expected generated request id header")
	}
}

func TestRequestIDPreservedFromHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set(RequestIDHeader, "client-provided-id")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get(RequestIDHeader); got != "client-provided-id" {
		t.Fatalf("expected request id to be preserved, got %q", got)
	}
}

func TestAccessLogIncludesRequestMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var buf bytes.Buffer
	logging.SetOutput(&buf)
	logging.ApplyConfig(logging.Config{Level: "info", Format: "json"})
	defer func() {
		logging.SetOutput(nil)
		logging.ApplyConfig(logging.Config{Level: "info", Format: "console"})
	}()

	r := gin.New()
	r.Use(RequestID(), AccessLog())
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set(RequestIDHeader, "abc-123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		t.Fatal("expected log output")
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &payload); err != nil {
		t.Fatalf("decode log line: %v", err)
	}

	if payload["msg"] != "http_request" {
		t.Fatalf("expected http_request log, got %v", payload["msg"])
	}
	if payload["request_id"] != "abc-123" {
		t.Fatalf("expected request_id abc-123, got %v", payload["request_id"])
	}
	if payload["method"] != http.MethodGet {
		t.Fatalf("expected method GET, got %v", payload["method"])
	}
	if payload["path"] != "/ping" {
		t.Fatalf("expected path /ping, got %v", payload["path"])
	}
	if status, ok := payload["status"].(float64); !ok || int(status) != http.StatusCreated {
		t.Fatalf("expected status 201, got %v", payload["status"])
	}
	if _, ok := payload["duration_ms"]; !ok {
		t.Fatal("expected duration_ms in access log")
	}
}
