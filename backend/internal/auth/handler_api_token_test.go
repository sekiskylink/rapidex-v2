package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCreateAPITokenEndpointReturnsPlaintextOnceAndStoresHash(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newFakeRepo(&User{ID: 1, Username: "admin", IsActive: true})
	auditRepo := &fakeAuditRepo{}
	service := newTestService(repo, auditRepo)
	handler := NewHandler(service)

	r := gin.New()
	r.POST("/api/v1/admin/api-tokens", func(c *gin.Context) {
		c.Set(PrincipalContextKey, Principal{Type: "user", UserID: 1, Username: "admin"})
		handler.CreateAPIToken(c)
	})

	payload := map[string]any{
		"name":        "automation",
		"permissions": []string{"audit.read"},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/api-tokens", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var resp APITokenCreateResult
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Token == "" {
		t.Fatal("expected plaintext token in create response")
	}
	if resp.Prefix != APITokenPrefix(resp.Token) {
		t.Fatalf("expected prefix %s, got %s", APITokenPrefix(resp.Token), resp.Prefix)
	}

	stored, err := repo.GetAPITokenByID(req.Context(), resp.ID)
	if err != nil {
		t.Fatalf("stored token not found: %v", err)
	}
	if stored.TokenHash == resp.Token {
		t.Fatal("stored token hash must not equal plaintext token")
	}
	if stored.TokenHash != HashAPIToken("test-key", resp.Token) {
		t.Fatal("stored hash does not match configured HMAC hash")
	}

	permissions, err := repo.GetAPITokenPermissions(req.Context(), resp.ID)
	if err != nil {
		t.Fatalf("permissions lookup failed: %v", err)
	}
	if len(permissions) != 1 || permissions[0].Permission != "audit.read" {
		t.Fatalf("expected permissions [audit.read], got %v", permissions)
	}
}
