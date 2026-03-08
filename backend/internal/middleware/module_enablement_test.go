package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequireModuleEnabledAllowsEnabledModule(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/guarded", RequireModuleEnabled(func() map[string]bool {
		return map[string]bool{
			"modules.settings.enabled": true,
		}
	}, "settings"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/guarded", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestRequireModuleEnabledBlocksDisabledModule(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/guarded", RequireModuleEnabled(func() map[string]bool {
		return map[string]bool{
			"modules.settings.enabled": false,
		}
	}, "settings"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/guarded", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
	}

	var payload map[string]map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["error"]["code"] != "MODULE_DISABLED" {
		t.Fatalf("expected MODULE_DISABLED, got %v", payload["error"]["code"])
	}
}
