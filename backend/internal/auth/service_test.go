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

	passwordResetTokensByHash map[string]*PasswordResetToken
	passwordResetTokensByID   map[int64]*PasswordResetToken
	nextPasswordResetTokenID  int64
}

func newFakeRepo(user *User) *fakeRepo {
	r := &fakeRepo{
		usersByID:                 map[int64]*User{},
		usersByUsername:           map[string]*User{},
		tokensByHash:              map[string]*RefreshToken{},
		tokensByID:                map[int64]*RefreshToken{},
		nextTokenID:               1,
		apiTokensByID:             map[int64]*APIToken{},
		apiTokensByHash:           map[string]*APIToken{},
		apiTokenPerms:             map[int64][]APITokenPermission{},
		nextAPITokenID:            1,
		passwordResetTokensByHash: map[string]*PasswordResetToken{},
		passwordResetTokensByID:   map[int64]*PasswordResetToken{},
		nextPasswordResetTokenID:  1,
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

func (r *fakeRepo) GetActiveUserByIdentifier(_ context.Context, identifier string) (*User, error) {
	for _, user := range r.usersByID {
		if !user.IsActive {
			continue
		}
		if user.Username == identifier {
			return user, nil
		}
	}
	return nil, ErrNotFound
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

func (r *fakeRepo) UpdateUserLastLoginAt(_ context.Context, userID int64, at time.Time) error {
	user, ok := r.usersByID[userID]
	if !ok {
		return ErrNotFound
	}
	value := at
	user.LastLoginAt = &value
	return nil
}

func (r *fakeRepo) CreatePasswordResetToken(_ context.Context, token PasswordResetToken) (*PasswordResetToken, error) {
	token.ID = r.nextPasswordResetTokenID
	r.nextPasswordResetTokenID++
	copy := token
	r.passwordResetTokensByHash[token.TokenHash] = &copy
	r.passwordResetTokensByID[token.ID] = &copy
	return &copy, nil
}

func (r *fakeRepo) GetPasswordResetTokenByHash(_ context.Context, hash string) (*PasswordResetToken, error) {
	token, ok := r.passwordResetTokensByHash[hash]
	if !ok {
		return nil, ErrNotFound
	}
	copy := *token
	return &copy, nil
}

func (r *fakeRepo) InvalidateActivePasswordResetTokensForUser(_ context.Context, userID int64, now time.Time) error {
	for _, token := range r.passwordResetTokensByID {
		if token.UserID == userID && token.UsedAt == nil && token.ExpiresAt.After(now) {
			t := now
			token.UsedAt = &t
			token.UpdatedAt = now
		}
	}
	return nil
}

func (r *fakeRepo) MarkPasswordResetTokenUsed(_ context.Context, tokenID int64, now time.Time, consumedFromIP, consumedUserAgent *string) error {
	token, ok := r.passwordResetTokensByID[tokenID]
	if !ok || token.UsedAt != nil {
		return ErrNotFound
	}
	t := now
	token.UsedAt = &t
	token.ConsumedFromIP = consumedFromIP
	token.ConsumedUserAgent = consumedUserAgent
	token.UpdatedAt = now
	return nil
}

func (r *fakeRepo) UpdateUserPasswordHash(_ context.Context, userID int64, passwordHash string, _ time.Time) error {
	user, ok := r.usersByID[userID]
	if !ok {
		return ErrNotFound
	}
	user.PasswordHash = passwordHash
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

func (r *fakeAuditRepo) List(_ context.Context, _ audit.ListFilter) (audit.ListResult, error) {
	return audit.ListResult{}, nil
}

func newTestService(repo *fakeRepo, auditRepo *fakeAuditRepo) *Service {
	return NewService(
		repo,
		audit.NewService(auditRepo),
		NewJWTManager("test-key", 5*time.Minute),
		nil,
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
	if repo.usersByID[user.ID].LastLoginAt == nil {
		t.Fatal("expected last login timestamp to be updated")
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
	repo.usersByID[7] = &User{ID: 7, Username: "svc-reporter", IsActive: true}
	auditRepo := &fakeAuditRepo{}
	service := newTestService(repo, auditRepo)

	expires := int64(3600)
	adminID := int64(1)
	boundUserID := int64(7)
	result, err := service.CreateAPIToken(context.Background(), &adminID, APITokenCreateInput{
		Name:             "ci-token",
		BoundUserID:      &boundUserID,
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
	if stored.BoundUserID == nil || *stored.BoundUserID != boundUserID {
		t.Fatalf("expected bound user id %d, got %+v", boundUserID, stored.BoundUserID)
	}

	perms, _ := repo.GetAPITokenPermissions(context.Background(), result.ID)
	if len(perms) != 1 || perms[0].Permission != "audit.read" {
		t.Fatalf("expected stored permission audit.read, got %+v", perms)
	}
}

func TestCreateAPITokenRejectsInactiveBoundUser(t *testing.T) {
	repo := newFakeRepo(&User{ID: 1, Username: "admin", IsActive: true})
	repo.usersByID[9] = &User{ID: 9, Username: "disabled", IsActive: false}
	service := newTestService(repo, &fakeAuditRepo{})

	adminID := int64(1)
	boundUserID := int64(9)
	_, err := service.CreateAPIToken(context.Background(), &adminID, APITokenCreateInput{
		Name:        "ci-token",
		BoundUserID: &boundUserID,
	}, "127.0.0.1", "test-agent")
	if err == nil {
		t.Fatal("expected inactive bound user to be rejected")
	}
}

func TestAuthenticateAPITokenIncludesBoundUser(t *testing.T) {
	repo := newFakeRepo(&User{ID: 1, Username: "admin", IsActive: true})
	repo.usersByID[5] = &User{ID: 5, Username: "svc", IsActive: true}
	service := newTestService(repo, &fakeAuditRepo{})

	now := time.Now().UTC()
	plain := "plain-token"
	hash := HashAPIToken("test-key", plain)
	repo.apiTokensByHash[hash] = &APIToken{
		ID:          2,
		Name:        "svc-token",
		TokenHash:   hash,
		Prefix:      APITokenPrefix(plain),
		BoundUserID: int64Ptr(5),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	repo.apiTokensByID[2] = repo.apiTokensByHash[hash]
	repo.apiTokenPerms[2] = []APITokenPermission{{APITokenID: 2, Permission: "requests.read"}}

	principal, err := service.AuthenticateAPIToken(context.Background(), plain, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("authenticate api token: %v", err)
	}
	if principal.BoundUserID == nil || *principal.BoundUserID != 5 {
		t.Fatalf("expected bound user id 5, got %+v", principal.BoundUserID)
	}
	if principal.BoundUsername != "svc" {
		t.Fatalf("expected bound username svc, got %q", principal.BoundUsername)
	}
}

func TestForgotPasswordRequestDoesNotLeakUnknownAccount(t *testing.T) {
	repo := newFakeRepo(&User{ID: 1, Username: "admin", IsActive: true})
	service := newTestService(repo, &fakeAuditRepo{})

	result, err := service.RequestPasswordReset(context.Background(), "missing-user", "", "127.0.0.1", "agent")
	if err != nil {
		t.Fatalf("forgot password request: %v", err)
	}
	if result.Status != "accepted" {
		t.Fatalf("expected accepted status, got %q", result.Status)
	}
	if len(repo.passwordResetTokensByID) != 0 {
		t.Fatal("expected no token generated for unknown user")
	}
}

func TestForgotPasswordAndResetPasswordSuccess(t *testing.T) {
	passwordHash, err := HashPassword("old-secret", 4)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	repo := newFakeRepo(&User{ID: 7, Username: "alice", PasswordHash: passwordHash, IsActive: true})
	service := newTestService(repo, &fakeAuditRepo{})
	service.passwordResetTTL = time.Hour

	result, err := service.RequestPasswordReset(context.Background(), "alice", "https://example.com/reset-password", "127.0.0.1", "agent")
	if err != nil {
		t.Fatalf("request reset: %v", err)
	}
	if result.Status != "accepted" {
		t.Fatalf("expected accepted status, got %q", result.Status)
	}
	if len(repo.passwordResetTokensByID) != 1 {
		t.Fatalf("expected one reset token, got %d", len(repo.passwordResetTokensByID))
	}

	var stored *PasswordResetToken
	for _, token := range repo.passwordResetTokensByID {
		stored = token
	}
	if stored == nil {
		t.Fatal("expected stored token")
	}

	plaintext := "plain-reset-token"
	hash := HashToken(plaintext)
	stored.TokenHash = hash
	repo.passwordResetTokensByHash = map[string]*PasswordResetToken{hash: stored}

	if err := service.ResetPasswordWithToken(context.Background(), plaintext, "new-secret", "127.0.0.1", "agent"); err != nil {
		t.Fatalf("reset password: %v", err)
	}

	if ComparePassword(repo.usersByID[7].PasswordHash, "new-secret") != nil {
		t.Fatal("expected password hash to update")
	}
	if stored.UsedAt == nil {
		t.Fatal("expected token marked used")
	}
}

func TestResetPasswordRejectsExpiredOrUsedToken(t *testing.T) {
	repo := newFakeRepo(&User{ID: 8, Username: "bob", PasswordHash: "x", IsActive: true})
	service := newTestService(repo, &fakeAuditRepo{})
	now := time.Now().UTC()
	service.now = func() time.Time { return now }

	expiredPlain := "expired-token"
	expiredHash := HashToken(expiredPlain)
	expired := &PasswordResetToken{
		ID:        1,
		UserID:    8,
		TokenHash: expiredHash,
		ExpiresAt: now.Add(-time.Minute),
		CreatedAt: now.Add(-2 * time.Minute),
		UpdatedAt: now.Add(-2 * time.Minute),
	}
	repo.passwordResetTokensByID[1] = expired
	repo.passwordResetTokensByHash[expiredHash] = expired

	if err := service.ResetPasswordWithToken(context.Background(), expiredPlain, "new-secret", "127.0.0.1", "agent"); err == nil {
		t.Fatal("expected expired token to fail")
	}
	if expired.UsedAt == nil {
		t.Fatal("expected expired token to be invalidated")
	}

	usedPlain := "used-token"
	usedHash := HashToken(usedPlain)
	usedAt := now.Add(-time.Second)
	used := &PasswordResetToken{
		ID:        2,
		UserID:    8,
		TokenHash: usedHash,
		ExpiresAt: now.Add(time.Hour),
		UsedAt:    &usedAt,
		CreatedAt: now.Add(-2 * time.Minute),
		UpdatedAt: now.Add(-time.Second),
	}
	repo.passwordResetTokensByID[2] = used
	repo.passwordResetTokensByHash[usedHash] = used

	if err := service.ResetPasswordWithToken(context.Background(), usedPlain, "new-secret", "127.0.0.1", "agent"); err == nil {
		t.Fatal("expected used token to fail")
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

func int64Ptr(v int64) *int64 {
	return &v
}
