package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"basepro/backend/internal/auth"
	"github.com/gin-gonic/gin"
)

type tokenRepo struct {
	tokens      map[string]*auth.APIToken
	permissions map[int64][]auth.APITokenPermission
}

func newTokenRepo() *tokenRepo {
	return &tokenRepo{
		tokens:      map[string]*auth.APIToken{},
		permissions: map[int64][]auth.APITokenPermission{},
	}
}

func (r *tokenRepo) GetUserByUsername(context.Context, string) (*auth.User, error) {
	return nil, auth.ErrNotFound
}
func (r *tokenRepo) GetUserByID(context.Context, int64) (*auth.User, error) {
	return nil, auth.ErrNotFound
}
func (r *tokenRepo) GetRefreshTokenByHash(context.Context, string) (*auth.RefreshToken, error) {
	return nil, auth.ErrNotFound
}
func (r *tokenRepo) CreateRefreshToken(context.Context, auth.RefreshToken) (*auth.RefreshToken, error) {
	return nil, auth.ErrNotFound
}
func (r *tokenRepo) RevokeRefreshToken(context.Context, int64, *int64, time.Time) error { return nil }
func (r *tokenRepo) RevokeAllActiveRefreshTokensForUser(context.Context, int64, time.Time) error {
	return nil
}
func (r *tokenRepo) CreateAPIToken(context.Context, auth.APIToken, []string, *string) (*auth.APIToken, error) {
	return nil, auth.ErrNotFound
}
func (r *tokenRepo) ListAPITokens(context.Context) ([]auth.APIToken, error) { return nil, nil }
func (r *tokenRepo) GetAPITokenByID(context.Context, int64) (*auth.APIToken, error) {
	return nil, auth.ErrNotFound
}
func (r *tokenRepo) GetAPITokenByHash(_ context.Context, hash string) (*auth.APIToken, error) {
	t, ok := r.tokens[hash]
	if !ok {
		return nil, auth.ErrNotFound
	}
	copy := *t
	return &copy, nil
}
func (r *tokenRepo) GetAPITokenPermissions(_ context.Context, tokenID int64) ([]auth.APITokenPermission, error) {
	return append([]auth.APITokenPermission{}, r.permissions[tokenID]...), nil
}
func (r *tokenRepo) RevokeAPIToken(context.Context, int64, time.Time) error { return nil }
func (r *tokenRepo) UpdateAPITokenLastUsed(_ context.Context, tokenID int64, now time.Time) error {
	for _, t := range r.tokens {
		if t.ID == tokenID {
			nt := now
			t.LastUsedAt = &nt
			return nil
		}
	}
	return auth.ErrNotFound
}
func (r *tokenRepo) EnsureUser(context.Context, string, string, bool) (*auth.User, error) {
	return nil, auth.ErrNotFound
}

func TestAPITokenMiddlewareValidTokenGrantsAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newTokenRepo()
	secret := "test-secret"
	plain := "bpt_validtoken"
	hash := auth.HashAPIToken(secret, plain)
	repo.tokens[hash] = &auth.APIToken{ID: 9, Name: "ops", TokenHash: hash, Prefix: auth.APITokenPrefix(plain), CreatedAt: time.Now(), UpdatedAt: time.Now()}
	repo.permissions[9] = []auth.APITokenPermission{{APITokenID: 9, Permission: "audit.read"}}

	service := auth.NewService(repo, nil, auth.NewJWTManager("jwt", time.Minute), nil, time.Minute, time.Hour, time.Hour, secret, true, 4)

	r := gin.New()
	r.GET("/protected", APITokenAuth(service, "X-API-Token", false), RequirePermission(nil, "audit.read"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("X-API-Token", plain)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAPITokenMiddlewareRevokedTokenRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newTokenRepo()
	secret := "test-secret"
	plain := "bpt_revoked"
	hash := auth.HashAPIToken(secret, plain)
	now := time.Now().UTC()
	repo.tokens[hash] = &auth.APIToken{ID: 10, Name: "revoked", TokenHash: hash, Prefix: auth.APITokenPrefix(plain), RevokedAt: &now, CreatedAt: now, UpdatedAt: now}

	service := auth.NewService(repo, nil, auth.NewJWTManager("jwt", time.Minute), nil, time.Minute, time.Hour, time.Hour, secret, true, 4)

	r := gin.New()
	r.GET("/protected", APITokenAuth(service, "X-API-Token", false), RequirePermission(nil, "audit.read"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("X-API-Token", plain)
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

func TestAPITokenMiddlewareExpiredTokenRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newTokenRepo()
	secret := "test-secret"
	plain := "bpt_expired"
	hash := auth.HashAPIToken(secret, plain)
	expired := time.Now().UTC().Add(-time.Minute)
	repo.tokens[hash] = &auth.APIToken{ID: 11, Name: "expired", TokenHash: hash, Prefix: auth.APITokenPrefix(plain), ExpiresAt: &expired, CreatedAt: time.Now(), UpdatedAt: time.Now()}

	service := auth.NewService(repo, nil, auth.NewJWTManager("jwt", time.Minute), nil, time.Minute, time.Hour, time.Hour, secret, true, 4)

	r := gin.New()
	r.GET("/protected", APITokenAuth(service, "X-API-Token", false), RequirePermission(nil, "audit.read"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("X-API-Token", plain)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAPITokenWithoutAuditReadCannotAccessAudit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newTokenRepo()
	secret := "test-secret"
	plain := "bpt_noaudit"
	hash := auth.HashAPIToken(secret, plain)
	repo.tokens[hash] = &auth.APIToken{ID: 12, Name: "svc", TokenHash: hash, Prefix: auth.APITokenPrefix(plain), CreatedAt: time.Now(), UpdatedAt: time.Now()}
	repo.permissions[12] = []auth.APITokenPermission{{APITokenID: 12, Permission: "users.read"}}

	service := auth.NewService(repo, nil, auth.NewJWTManager("jwt", time.Minute), nil, time.Minute, time.Hour, time.Hour, secret, true, 4)

	r := gin.New()
	r.GET("/audit", APITokenAuth(service, "X-API-Token", false), RequirePermission(nil, "audit.read"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/audit", nil)
	req.Header.Set("X-API-Token", plain)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
	var body map[string]map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["error"]["code"] != "AUTH_FORBIDDEN" {
		t.Fatalf("expected AUTH_FORBIDDEN, got %q", body["error"]["code"])
	}
}

func TestAPITokenWithAuditReadCanAccessAudit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newTokenRepo()
	secret := "test-secret"
	plain := "bpt_auditok"
	hash := auth.HashAPIToken(secret, plain)
	repo.tokens[hash] = &auth.APIToken{ID: 13, Name: "svc", TokenHash: hash, Prefix: auth.APITokenPrefix(plain), CreatedAt: time.Now(), UpdatedAt: time.Now()}
	repo.permissions[13] = []auth.APITokenPermission{{APITokenID: 13, Permission: "audit.read"}}

	service := auth.NewService(repo, nil, auth.NewJWTManager("jwt", time.Minute), nil, time.Minute, time.Hour, time.Hour, secret, true, 4)

	r := gin.New()
	r.GET("/audit", APITokenAuth(service, "X-API-Token", false), RequirePermission(nil, "audit.read"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/audit", nil)
	req.Header.Set("X-API-Token", plain)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
