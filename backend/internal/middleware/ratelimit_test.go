package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAuthRateLimiterDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter := NewAuthRateLimiter(RateLimitConfig{
		Enabled:           false,
		RequestsPerSecond: 1,
		Burst:             1,
	})

	r := gin.New()
	r.POST("/api/v1/auth/login", limiter.Middleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 when disabled, got %d", w.Code)
		}
	}
}

func TestAuthRateLimiterEnabledReturns429Shape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter := NewAuthRateLimiter(RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 1,
		Burst:             1,
	})

	r := gin.New()
	r.POST("/api/v1/auth/refresh", limiter.Middleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req1.RemoteAddr = "10.0.0.1:1234"
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected first request 200, got %d", w1.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req2.RemoteAddr = "10.0.0.1:1234"
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request 429, got %d", w2.Code)
	}

	var body map[string]map[string]any
	if err := json.Unmarshal(w2.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["error"]["code"] != "RATE_LIMITED" {
		t.Fatalf("expected RATE_LIMITED, got %v", body["error"]["code"])
	}
	if details, ok := body["error"]["details"].(map[string]any); !ok || len(details) != 0 {
		t.Fatalf("expected empty details object, got %T %v", body["error"]["details"], body["error"]["details"])
	}
}
