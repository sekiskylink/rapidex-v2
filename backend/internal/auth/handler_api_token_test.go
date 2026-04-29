package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestCreateAPITokenEndpointReturnsPlaintextOnceAndStoresHash(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newFakeRepo(&User{ID: 1, Username: "admin", IsActive: true})
	repo.usersByID[7] = &User{ID: 7, Username: "svc-user", IsActive: true}
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
		"boundUserId": int64(7),
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
	if resp.BoundUserID == nil || *resp.BoundUserID != 7 {
		t.Fatalf("expected bound user id 7, got %+v", resp.BoundUserID)
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
	if stored.BoundUserID == nil || *stored.BoundUserID != 7 {
		t.Fatalf("expected stored bound user id 7, got %+v", stored.BoundUserID)
	}

	permissions, err := repo.GetAPITokenPermissions(req.Context(), resp.ID)
	if err != nil {
		t.Fatalf("permissions lookup failed: %v", err)
	}
	if len(permissions) != 1 || permissions[0].Permission != "audit.read" {
		t.Fatalf("expected permissions [audit.read], got %v", permissions)
	}
}

func TestListMyActiveAPITokensEndpointReturnsOwnedActiveTokens(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newFakeRepo(&User{ID: 1, Username: "admin", IsActive: true})
	now := time.Date(2026, 4, 29, 11, 45, 0, 0, time.UTC)
	activeExpiry := now.Add(24 * time.Hour)
	repo.apiTokensByID[1] = &APIToken{
		ID:              1,
		Name:            "owned-token",
		Prefix:          "bpt_12",
		CreatedByUserID: int64Ptr(1),
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now.Add(-time.Hour),
		ExpiresAt:       &activeExpiry,
	}
	repo.apiTokensByID[2] = &APIToken{
		ID:              2,
		Name:            "other-token",
		Prefix:          "bpt_34",
		CreatedByUserID: int64Ptr(7),
		CreatedAt:       now.Add(-2 * time.Hour),
		UpdatedAt:       now.Add(-2 * time.Hour),
	}

	service := newTestService(repo, &fakeAuditRepo{})
	service.now = func() time.Time { return now }
	handler := NewHandler(service)

	r := gin.New()
	r.GET("/api/v1/admin/api-tokens/mine", func(c *gin.Context) {
		c.Set(PrincipalContextKey, Principal{Type: "user", UserID: 1, Username: "admin"})
		handler.ListMyActiveAPITokens(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/api-tokens/mine", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Items []APITokenSummary `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 token, got %d", len(resp.Items))
	}
	if resp.Items[0].Name != "owned-token" {
		t.Fatalf("expected owned-token, got %+v", resp.Items[0])
	}
}

func TestRevokeAPITokenEndpointRevokesToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newFakeRepo(&User{ID: 1, Username: "admin", IsActive: true})
	now := time.Date(2026, 4, 29, 11, 45, 0, 0, time.UTC)
	repo.apiTokensByID[5] = &APIToken{
		ID:              5,
		Name:            "owned-token",
		Prefix:          "bpt_12",
		CreatedByUserID: int64Ptr(1),
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now.Add(-time.Hour),
	}

	service := newTestService(repo, &fakeAuditRepo{})
	service.now = func() time.Time { return now }
	handler := NewHandler(service)

	r := gin.New()
	r.POST("/api/v1/admin/api-tokens/:id/revoke", func(c *gin.Context) {
		c.Set(PrincipalContextKey, Principal{Type: "user", UserID: 1, Username: "admin"})
		handler.RevokeAPIToken(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/api-tokens/5/revoke", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	stored, err := repo.GetAPITokenByID(req.Context(), 5)
	if err != nil {
		t.Fatalf("stored token lookup failed: %v", err)
	}
	if stored.RevokedAt == nil {
		t.Fatal("expected token to be revoked")
	}
}
