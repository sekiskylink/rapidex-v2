package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/audit"
)

type fakeRepo struct {
	usersByID       map[int64]*User
	usersByUsername map[string]*User
	tokensByHash    map[string]*RefreshToken
	tokensByID      map[int64]*RefreshToken
	nextTokenID     int64

	apiTokensByID   map[int64]*APIToken
	apiTokensByHash map[string]*APIToken
	apiTokenPerms   map[int64][]APITokenPermission
	nextAPITokenID  int64
}

func newFakeRepo(user *User) *fakeRepo {
	r := &fakeRepo{
		usersByID:       map[int64]*User{},
		usersByUsername: map[string]*User{},
		tokensByHash:    map[string]*RefreshToken{},
		tokensByID:      map[int64]*RefreshToken{},
		nextTokenID:     1,
		apiTokensByID:   map[int64]*APIToken{},
		apiTokensByHash: map[string]*APIToken{},
		apiTokenPerms:   map[int64][]APITokenPermission{},
		nextAPITokenID:  1,
	}
	if user != nil {
		r.usersByID[user.ID] = user
		r.usersByUsername[user.Username] = user
	}
	return r
}

func (r *fakeRepo) GetUserByUsername(_ context.Context, username string) (*User, error) {
	user, ok := r.usersByUsername[username]
	if !ok {
		return nil, ErrNotFound
	}
	return user, nil
}

func (r *fakeRepo) GetUserByID(_ context.Context, userID int64) (*User, error) {
	user, ok := r.usersByID[userID]
	if !ok {
		return nil, ErrNotFound
	}
	return user, nil
}

func (r *fakeRepo) GetRefreshTokenByHash(_ context.Context, hash string) (*RefreshToken, error) {
	token, ok := r.tokensByHash[hash]
	if !ok {
		return nil, ErrNotFound
	}
	copy := *token
	return &copy, nil
}

func (r *fakeRepo) CreateRefreshToken(_ context.Context, token RefreshToken) (*RefreshToken, error) {
	token.ID = r.nextTokenID
	r.nextTokenID++
	copy := token
	r.tokensByHash[token.TokenHash] = &copy
	r.tokensByID[token.ID] = &copy
	return &copy, nil
}

func (r *fakeRepo) RevokeRefreshToken(_ context.Context, tokenID int64, replacedByTokenID *int64, now time.Time) error {
	token, ok := r.tokensByID[tokenID]
	if !ok {
		return ErrNotFound
	}
	if token.RevokedAt == nil {
		t := now
		token.RevokedAt = &t
	}
	token.ReplacedByTokenID = replacedByTokenID
	token.UpdatedAt = now
	return nil
}

func (r *fakeRepo) RevokeAllActiveRefreshTokensForUser(_ context.Context, userID int64, now time.Time) error {
	for _, token := range r.tokensByID {
		if token.UserID == userID && token.RevokedAt == nil {
			t := now
			token.RevokedAt = &t
			token.UpdatedAt = now
		}
	}
	return nil
}

func (r *fakeRepo) CreateAPIToken(_ context.Context, token APIToken, permissions []string, _ *string) (*APIToken, error) {
	token.ID = r.nextAPITokenID
	r.nextAPITokenID++
	copy := token
	r.apiTokensByID[token.ID] = &copy
	r.apiTokensByHash[token.TokenHash] = &copy
	perms := make([]APITokenPermission, 0, len(permissions))
	for _, permission := range permissions {
		perms = append(perms, APITokenPermission{APITokenID: token.ID, Permission: permission})
	}
	r.apiTokenPerms[token.ID] = perms
	return &copy, nil
}

func (r *fakeRepo) ListAPITokens(_ context.Context) ([]APIToken, error) {
	items := make([]APIToken, 0, len(r.apiTokensByID))
	for _, t := range r.apiTokensByID {
		items = append(items, *t)
	}
	return items, nil
}

func (r *fakeRepo) GetAPITokenByID(_ context.Context, tokenID int64) (*APIToken, error) {
	token, ok := r.apiTokensByID[tokenID]
	if !ok {
		return nil, ErrNotFound
	}
	copy := *token
	return &copy, nil
}

func (r *fakeRepo) GetAPITokenByHash(_ context.Context, hash string) (*APIToken, error) {
	token, ok := r.apiTokensByHash[hash]
	if !ok {
		return nil, ErrNotFound
	}
	copy := *token
	return &copy, nil
}

func (r *fakeRepo) GetAPITokenPermissions(_ context.Context, tokenID int64) ([]APITokenPermission, error) {
	return append([]APITokenPermission{}, r.apiTokenPerms[tokenID]...), nil
}

func (r *fakeRepo) RevokeAPIToken(_ context.Context, tokenID int64, now time.Time) error {
	token, ok := r.apiTokensByID[tokenID]
	if !ok {
		return ErrNotFound
	}
	if token.RevokedAt == nil {
		t := now
		token.RevokedAt = &t
	}
	token.UpdatedAt = now
	return nil
}

func (r *fakeRepo) UpdateAPITokenLastUsed(_ context.Context, tokenID int64, now time.Time) error {
	token, ok := r.apiTokensByID[tokenID]
	if !ok {
		return ErrNotFound
	}
	t := now
	token.LastUsedAt = &t
	token.UpdatedAt = now
	return nil
}

func (r *fakeRepo) EnsureUser(_ context.Context, username, passwordHash string, isActive bool) (*User, error) {
	if existing, ok := r.usersByUsername[username]; ok {
		return existing, nil
	}
	id := int64(len(r.usersByID) + 1)
	user := &User{ID: id, Username: username, PasswordHash: passwordHash, IsActive: isActive}
	r.usersByID[id] = user
	r.usersByUsername[username] = user
	return user, nil
}

type fakeAuditRepo struct {
	events []audit.Event
}

func (r *fakeAuditRepo) Insert(_ context.Context, event audit.Event) error {
	r.events = append(r.events, event)
	return nil
}

func (r *fakeAuditRepo) List(_ context.Context, _ audit.ListFilter) ([]audit.Record, error) {
	return nil, nil
}

func newTestService(repo *fakeRepo, auditRepo *fakeAuditRepo) *Service {
	return NewService(
		repo,
		audit.NewService(auditRepo),
		NewJWTManager("test-key", 5*time.Minute),
		5*time.Minute,
		24*time.Hour,
		24*time.Hour,
		"test-key",
		true,
		4,
	)
}

func TestLoginSuccessReturnsTokensAndStoresRefreshHash(t *testing.T) {
	passwordHash, err := HashPassword("secret", 4)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	user := &User{ID: 10, Username: "alice", PasswordHash: passwordHash, IsActive: true}
	repo := newFakeRepo(user)
	auditRepo := &fakeAuditRepo{}
	service := newTestService(repo, auditRepo)

	resp, err := service.Login(context.Background(), "alice", "secret", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatal("expected access and refresh tokens")
	}
	if resp.ExpiresIn <= 0 {
		t.Fatalf("expected positive expiresIn, got %d", resp.ExpiresIn)
	}

	hash := HashToken(resp.RefreshToken)
	stored, err := repo.GetRefreshTokenByHash(context.Background(), hash)
	if err != nil {
		t.Fatalf("stored refresh token not found: %v", err)
	}
	if stored.TokenHash == resp.RefreshToken {
		t.Fatal("refresh token must be stored as hash, not plaintext")
	}
}

func TestLoginFailureReturnsUnauthorized(t *testing.T) {
	passwordHash, err := HashPassword("secret", 4)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	user := &User{ID: 1, Username: "alice", PasswordHash: passwordHash, IsActive: true}
	repo := newFakeRepo(user)
	service := newTestService(repo, &fakeAuditRepo{})

	_, err = service.Login(context.Background(), "alice", "wrong", "127.0.0.1", "test-agent")
	if err == nil {
		t.Fatal("expected login error")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != apperror.CodeAuthUnauthorized {
		t.Fatalf("expected %s, got %s", apperror.CodeAuthUnauthorized, appErr.Code)
	}
}

func TestRefreshSuccessRotatesTokenAndNewTokenWorks(t *testing.T) {
	passwordHash, err := HashPassword("secret", 4)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	user := &User{ID: 2, Username: "bob", PasswordHash: passwordHash, IsActive: true}
	repo := newFakeRepo(user)
	service := newTestService(repo, &fakeAuditRepo{})

	loginResp, err := service.Login(context.Background(), "bob", "secret", "127.0.0.1", "agent")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	oldRecord, err := repo.GetRefreshTokenByHash(context.Background(), HashToken(loginResp.RefreshToken))
	if err != nil {
		t.Fatalf("get old record: %v", err)
	}

	refreshResp, err := service.Refresh(context.Background(), loginResp.RefreshToken, "127.0.0.1", "agent")
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}

	updatedOld, err := repo.GetRefreshTokenByHash(context.Background(), HashToken(loginResp.RefreshToken))
	if err != nil {
		t.Fatalf("get updated old record: %v", err)
	}
	if updatedOld.RevokedAt == nil {
		t.Fatal("expected old token to be revoked")
	}
	if updatedOld.ReplacedByTokenID == nil || *updatedOld.ReplacedByTokenID == oldRecord.ID {
		t.Fatal("expected old token replaced_by_token_id to point to new token")
	}

	if _, err := service.Refresh(context.Background(), refreshResp.RefreshToken, "127.0.0.1", "agent"); err != nil {
		t.Fatalf("newly issued refresh token should work: %v", err)
	}
}

func TestRefreshReuseDetectionRevokesActiveTokens(t *testing.T) {
	passwordHash, err := HashPassword("secret", 4)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	user := &User{ID: 3, Username: "carol", PasswordHash: passwordHash, IsActive: true}
	repo := newFakeRepo(user)
	service := newTestService(repo, &fakeAuditRepo{})

	loginResp, err := service.Login(context.Background(), "carol", "secret", "127.0.0.1", "agent")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	rotatedResp, err := service.Refresh(context.Background(), loginResp.RefreshToken, "127.0.0.1", "agent")
	if err != nil {
		t.Fatalf("rotate token: %v", err)
	}

	_, err = service.Refresh(context.Background(), loginResp.RefreshToken, "127.0.0.1", "agent")
	if err == nil {
		t.Fatal("expected reuse detection error")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != apperror.CodeAuthRefreshReuse {
		t.Fatalf("expected %s, got %s", apperror.CodeAuthRefreshReuse, appErr.Code)
	}

	active, err := repo.GetAPITokenByID(context.Background(), 999)
	if err == nil || active != nil {
		// noop guard for interface expansion; not relevant to refresh chain assertions.
	}

	refreshActive, err := repo.GetRefreshTokenByHash(context.Background(), HashToken(rotatedResp.RefreshToken))
	if err != nil {
		t.Fatalf("lookup active token: %v", err)
	}
	if refreshActive.RevokedAt == nil {
		t.Fatal("expected all active user tokens to be revoked on reuse detection")
	}
}

func TestCreateAPITokenStoresHashAndPrefix(t *testing.T) {
	repo := newFakeRepo(&User{ID: 1, Username: "admin", IsActive: true})
	auditRepo := &fakeAuditRepo{}
	service := newTestService(repo, auditRepo)

	expires := int64(3600)
	adminID := int64(1)
	result, err := service.CreateAPIToken(context.Background(), &adminID, APITokenCreateInput{
		Name:             "ci-token",
		ExpiresInSeconds: &expires,
		Permissions:      []string{"audit.read"},
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("create api token: %v", err)
	}

	if result.Token == "" {
		t.Fatal("expected plaintext token on create")
	}
	if result.Prefix != APITokenPrefix(result.Token) {
		t.Fatalf("unexpected token prefix: %s", result.Prefix)
	}

	stored, err := repo.GetAPITokenByID(context.Background(), result.ID)
	if err != nil {
		t.Fatalf("stored token missing: %v", err)
	}
	if stored.TokenHash == result.Token {
		t.Fatal("api token stored as plaintext")
	}
	if stored.TokenHash != HashAPIToken("test-key", result.Token) {
		t.Fatal("stored hash does not match expected HMAC")
	}

	perms, _ := repo.GetAPITokenPermissions(context.Background(), result.ID)
	if len(perms) != 1 || perms[0].Permission != "audit.read" {
		t.Fatalf("expected stored permission audit.read, got %+v", perms)
	}
}

func TestAPITokenCreateAndRevokeProduceAuditLogs(t *testing.T) {
	repo := newFakeRepo(&User{ID: 1, Username: "admin", IsActive: true})
	auditRepo := &fakeAuditRepo{}
	service := newTestService(repo, auditRepo)

	adminID := int64(1)
	result, err := service.CreateAPIToken(context.Background(), &adminID, APITokenCreateInput{Name: "ops"}, "127.0.0.1", "agent")
	if err != nil {
		t.Fatalf("create api token: %v", err)
	}

	if _, err := service.RevokeAPIToken(context.Background(), &adminID, result.ID, "127.0.0.1", "agent"); err != nil {
		t.Fatalf("revoke api token: %v", err)
	}

	if len(auditRepo.events) < 2 {
		t.Fatalf("expected at least 2 audit events, got %d", len(auditRepo.events))
	}
	if auditRepo.events[0].Action != "api_token.create" {
		t.Fatalf("expected first audit action api_token.create, got %s", auditRepo.events[0].Action)
	}
	if auditRepo.events[1].Action != "api_token.revoke" {
		t.Fatalf("expected second audit action api_token.revoke, got %s", auditRepo.events[1].Action)
	}
}
