package apperror_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/logging"
	"basepro/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

func TestAuthErrorShapeIncludesDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/auth", func(c *gin.Context) {
		apperror.Write(c, apperror.Unauthorized("Unauthorized"))
	})

	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	var body map[string]map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["error"]["code"] != apperror.CodeAuthUnauthorized {
		t.Fatalf("expected code %s, got %v", apperror.CodeAuthUnauthorized, body["error"]["code"])
	}
	if body["error"]["message"] != "Unauthorized" {
		t.Fatalf("expected message Unauthorized, got %v", body["error"]["message"])
	}
	if details, ok := body["error"]["details"].(map[string]any); !ok || len(details) != 0 {
		t.Fatalf("expected empty object details, got %T %v", body["error"]["details"], body["error"]["details"])
	}
}

func TestValidationErrorShapeHasPopulatedDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/validate", func(c *gin.Context) {
		apperror.Write(c, apperror.Validation("username is required"))
	})

	req := httptest.NewRequest(http.MethodGet, "/validate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var body map[string]map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["error"]["code"] != apperror.CodeValidationFailed {
		t.Fatalf("expected code %s, got %v", apperror.CodeValidationFailed, body["error"]["code"])
	}
	details, ok := body["error"]["details"].(map[string]any)
	if !ok || len(details) == 0 {
		t.Fatalf("expected populated details object, got %T %v", body["error"]["details"], body["error"]["details"])
	}
}

func TestInternalErrorShapeSafeMessageAndRequestIDHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var logOutput bytes.Buffer
	logging.SetOutput(&logOutput)
	logging.ApplyConfig(logging.Config{Level: "info", Format: "json"})
	defer func() {
		logging.SetOutput(nil)
		logging.ApplyConfig(logging.Config{Level: "info", Format: "console"})
	}()

	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/boom", func(c *gin.Context) {
		apperror.Write(c, errors.New("db timeout"))
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
	if w.Header().Get(middleware.RequestIDHeader) == "" {
		t.Fatal("expected X-Request-Id response header")
	}

	var body map[string]map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["error"]["code"] != "INTERNAL_ERROR" {
		t.Fatalf("expected INTERNAL_ERROR code, got %v", body["error"]["code"])
	}
	if body["error"]["message"] != "Internal server error" {
		t.Fatalf("expected safe internal error message, got %v", body["error"]["message"])
	}
	if details, ok := body["error"]["details"].(map[string]any); !ok || len(details) != 0 {
		t.Fatalf("expected empty details object, got %T %v", body["error"]["details"], body["error"]["details"])
	}

	logLine := logOutput.String()
	if strings.Count(logLine, `"request_id"`) != 1 {
		t.Fatalf("expected one request_id in log output, got %q", logLine)
	}
	if !strings.Contains(logLine, `"error":"db timeout"`) {
		t.Fatalf("expected internal error text in log output, got %q", logLine)
	}
}
