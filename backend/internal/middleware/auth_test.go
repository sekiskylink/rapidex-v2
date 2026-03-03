package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"basepro/backend/internal/auth"
	"github.com/gin-gonic/gin"
)

func TestJWTAuthMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtManager := auth.NewJWTManager("test-signing-key", time.Minute)

	r := gin.New()
	r.GET("/protected", JWTAuth(jwtManager), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	var body map[string]map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["error"]["code"] != "AUTH_UNAUTHORIZED" {
		t.Fatalf("expected AUTH_UNAUTHORIZED, got %q", body["error"]["code"])
	}
}

func TestJWTAuthExpiredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtManager := auth.NewJWTManager("test-signing-key", time.Second)

	token, _, err := jwtManager.GenerateAccessToken(1, "alice", time.Now().Add(-2*time.Second))
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	r := gin.New()
	r.GET("/protected", JWTAuth(jwtManager), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	var body map[string]map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["error"]["code"] != "AUTH_EXPIRED" {
		t.Fatalf("expected AUTH_EXPIRED, got %q", body["error"]["code"])
	}
}
