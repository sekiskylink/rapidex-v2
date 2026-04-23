package auth

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/audit"
	"basepro/backend/internal/rbac"
	"basepro/backend/internal/sukumad/userorg"
)

type orgUnitAssignmentProvider interface {
	GetUserOrgUnitIDs(context.Context, int64) ([]int64, error)
	ResolveScope(context.Context, int64) (userorg.Scope, error)
}

type Service struct {
	repo             Repository
	auditService     *audit.Service
	jwt              *JWTManager
	rbacService      *rbac.Service
	accessTTL        time.Duration
	refreshTTL       time.Duration
	apiTokenTTL      time.Duration
	apiTokenSecret   string
	apiTokenEnabled  bool
	passwordHashCost int
	passwordResetTTL time.Duration
	resetNotifier    PasswordResetNotifier
	orgUnitScope     orgUnitAssignmentProvider
	now              func() time.Time
}

type PasswordResetNotification struct {
	UserID             int64
	Username           string
	ResetToken         string
	ResetURL           *string
	ExpiresAt          time.Time
	RequestedFromIP    string
	RequestedUserAgent string
}

type PasswordResetNotifier interface {
	SendPasswordReset(ctx context.Context, notification PasswordResetNotification) error
}

func NewService(
	repo Repository,
	auditService *audit.Service,
	jwt *JWTManager,
	rbacService *rbac.Service,
	accessTTL, refreshTTL, apiTokenTTL time.Duration,
	apiTokenSecret string,
	apiTokenEnabled bool,
	passwordHashCost int,
) *Service {
	return &Service{
		repo:             repo,
		auditService:     auditService,
		jwt:              jwt,
		rbacService:      rbacService,
		accessTTL:        accessTTL,
		refreshTTL:       refreshTTL,
		apiTokenTTL:      apiTokenTTL,
		apiTokenSecret:   apiTokenSecret,
		apiTokenEnabled:  apiTokenEnabled,
		passwordHashCost: passwordHashCost,
		passwordResetTTL: 30 * time.Minute,
		now:              func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) SetPasswordResetNotifier(notifier PasswordResetNotifier) {
	s.resetNotifier = notifier
}

func (s *Service) WithUserOrgScope(provider orgUnitAssignmentProvider) *Service {
	if s == nil {
		return s
	}
	s.orgUnitScope = provider
	return s
}

func (s *Service) RequestPasswordReset(ctx context.Context, identifier, resetURLBase, ip, userAgent string) (PasswordResetRequestResult, error) {
	normalizedIdentifier := strings.TrimSpace(identifier)
	if normalizedIdentifier == "" {
		return PasswordResetRequestResult{}, apperror.ValidationWithDetails("validation failed", map[string]any{
			"identifier": []string{"is required"},
		})
	}

	var preparedResetURL *string
	if strings.TrimSpace(resetURLBase) != "" {
		u, err := url.Parse(strings.TrimSpace(resetURLBase))
		if err != nil || !u.IsAbs() || (u.Scheme != "http" && u.Scheme != "https") {
			return PasswordResetRequestResult{}, apperror.ValidationWithDetails("validation failed", map[string]any{
				"resetUrl": []string{"must be an absolute http(s) URL"},
			})
		}
		prepared := u.String()
		preparedResetURL = &prepared
	}

	user, err := s.repo.GetActiveUserByIdentifier(ctx, normalizedIdentifier)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			s.logAudit(ctx, audit.Event{
				Action:     "auth.password_reset.requested",
				EntityType: "auth",
				Metadata: map[string]any{
					"identifier": normalizedIdentifier,
					"result":     "accepted",
					"ip":         ip,
					"user_agent": userAgent,
				},
			})
			return PasswordResetRequestResult{Status: "accepted"}, nil
		}
		return PasswordResetRequestResult{}, err
	}

	now := s.now()
	if err := s.repo.InvalidateActivePasswordResetTokensForUser(ctx, user.ID, now); err != nil {
		return PasswordResetRequestResult{}, err
	}

	plainToken, err := GeneratePasswordResetToken()
	if err != nil {
		return PasswordResetRequestResult{}, err
	}

	tokenRecord, err := s.repo.CreatePasswordResetToken(ctx, PasswordResetToken{
		UserID:             user.ID,
		TokenHash:          HashToken(plainToken),
		ExpiresAt:          now.Add(s.passwordResetTTL),
		RequestedFromIP:    optionalString(ip),
		RequestedUserAgent: optionalString(userAgent),
		CreatedAt:          now,
		UpdatedAt:          now,
	})
	if err != nil {
		return PasswordResetRequestResult{}, err
	}

	var finalURL *string
	if preparedResetURL != nil {
		parsed, _ := url.Parse(*preparedResetURL)
		query := parsed.Query()
		query.Set("token", plainToken)
		parsed.RawQuery = query.Encode()
		urlValue := parsed.String()
		finalURL = &urlValue
	}

	if s.resetNotifier != nil {
		_ = s.resetNotifier.SendPasswordReset(ctx, PasswordResetNotification{
			UserID:             user.ID,
			Username:           user.Username,
			ResetToken:         plainToken,
			ResetURL:           finalURL,
			ExpiresAt:          tokenRecord.ExpiresAt,
			RequestedFromIP:    ip,
			RequestedUserAgent: userAgent,
		})
	}

	s.logAudit(ctx, audit.Event{
		Action:      "auth.password_reset.requested",
		ActorUserID: &user.ID,
		EntityType:  "auth",
		EntityID:    strPtr(user.Username),
		Metadata: map[string]any{
			"result":       "accepted",
			"expires_at":   tokenRecord.ExpiresAt.Format(time.RFC3339),
			"ip":           ip,
			"user_agent":   userAgent,
			"delivery":     "pending",
			"has_reset":    finalURL != nil,
			"identifier":   normalizedIdentifier,
			"token_stored": true,
		},
	})

	return PasswordResetRequestResult{Status: "accepted"}, nil
}

func (s *Service) ResetPasswordWithToken(ctx context.Context, token, newPassword, ip, userAgent string) error {
	normalizedToken := strings.TrimSpace(token)
	if normalizedToken == "" {
		return apperror.ValidationWithDetails("validation failed", map[string]any{
			"token": []string{"is required"},
		})
	}
	if strings.TrimSpace(newPassword) == "" {
		return apperror.ValidationWithDetails("validation failed", map[string]any{
			"password": []string{"is required"},
		})
	}
	if len(newPassword) > 256 {
		return apperror.ValidationWithDetails("validation failed", map[string]any{
			"password": []string{"must be 256 characters or fewer"},
		})
	}

	now := s.now()
	resetToken, err := s.repo.GetPasswordResetTokenByHash(ctx, HashToken(normalizedToken))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return apperror.ValidationWithDetails("validation failed", map[string]any{
				"token": []string{"is invalid, expired, or already used"},
			})
		}
		return err
	}
	if resetToken.UsedAt != nil {
		return apperror.ValidationWithDetails("validation failed", map[string]any{
			"token": []string{"is invalid, expired, or already used"},
		})
	}
	if now.After(resetToken.ExpiresAt) {
		_ = s.repo.MarkPasswordResetTokenUsed(ctx, resetToken.ID, now, optionalString(ip), optionalString(userAgent))
		return apperror.ValidationWithDetails("validation failed", map[string]any{
			"token": []string{"is invalid, expired, or already used"},
		})
	}

	passwordHash, err := HashPassword(newPassword, s.passwordHashCost)
	if err != nil {
		return err
	}

	if err := s.repo.MarkPasswordResetTokenUsed(ctx, resetToken.ID, now, optionalString(ip), optionalString(userAgent)); err != nil {
		return apperror.ValidationWithDetails("validation failed", map[string]any{
			"token": []string{"is invalid, expired, or already used"},
		})
	}
	if err := s.repo.UpdateUserPasswordHash(ctx, resetToken.UserID, passwordHash, now); err != nil {
		return err
	}
	if err := s.repo.RevokeAllActiveRefreshTokensForUser(ctx, resetToken.UserID, now); err != nil {
		return err
	}
	if err := s.repo.InvalidateActivePasswordResetTokensForUser(ctx, resetToken.UserID, now); err != nil {
		return err
	}

	s.logAudit(ctx, audit.Event{
		Action:      "auth.password_reset.success",
		ActorUserID: &resetToken.UserID,
		EntityType:  "auth",
		EntityID:    strPtr(strconv.FormatInt(resetToken.UserID, 10)),
		Metadata: map[string]any{
			"ip":         ip,
			"user_agent": userAgent,
		},
	})
	return nil
}

func (s *Service) Login(ctx context.Context, username, password, ip, userAgent string) (AuthResponse, error) {
	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil || !user.IsActive || ComparePassword(user.PasswordHash, password) != nil {
		s.logAudit(ctx, audit.Event{
			Action:     "auth.login.failure",
			EntityType: "auth",
			Metadata: map[string]any{
				"username":   username,
				"reason":     "invalid_credentials",
				"ip":         ip,
				"user_agent": userAgent,
			},
		})
		return AuthResponse{}, apperror.Unauthorized("Invalid credentials")
	}

	response, err := s.issueTokens(ctx, user.ID, user.Username)
	if err != nil {
		return AuthResponse{}, err
	}
	_ = s.repo.UpdateUserLastLoginAt(ctx, user.ID, s.now())

	s.logAudit(ctx, audit.Event{
		Action:      "auth.login.success",
		ActorUserID: &user.ID,
		EntityType:  "auth",
		EntityID:    strPtr(user.Username),
		Metadata: map[string]any{
			"ip":         ip,
			"user_agent": userAgent,
		},
	})

	return response, nil
}

func (s *Service) Refresh(ctx context.Context, presentedToken, ip, userAgent string) (AuthResponse, error) {
	now := s.now()
	hash := HashToken(presentedToken)
	token, err := s.repo.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		s.logAudit(ctx, audit.Event{
			Action:     "auth.refresh.failure",
			EntityType: "auth",
			Metadata: map[string]any{
				"reason":     "invalid_refresh_token",
				"ip":         ip,
				"user_agent": userAgent,
			},
		})
		return AuthResponse{}, apperror.RefreshInvalid("Refresh token is invalid")
	}

	if token.RevokedAt != nil {
		_ = s.repo.RevokeAllActiveRefreshTokensForUser(ctx, token.UserID, now)
		s.logAudit(ctx, audit.Event{
			Action:      "auth.refresh.reused",
			ActorUserID: &token.UserID,
			EntityType:  "auth",
			Metadata: map[string]any{
				"ip":         ip,
				"user_agent": userAgent,
			},
		})
		return AuthResponse{}, apperror.RefreshReused("Refresh token has been reused")
	}

	if now.After(token.ExpiresAt) {
		s.logAudit(ctx, audit.Event{
			Action:      "auth.refresh.failure",
			ActorUserID: &token.UserID,
			EntityType:  "auth",
			Metadata: map[string]any{
				"reason":     "expired_refresh_token",
				"ip":         ip,
				"user_agent": userAgent,
			},
		})
		return AuthResponse{}, apperror.RefreshInvalid("Refresh token is invalid")
	}

	newPlain, err := GenerateRefreshToken()
	if err != nil {
		return AuthResponse{}, errors.New("failed to generate refresh token")
	}
	newHash := HashToken(newPlain)

	newRecord, err := s.repo.CreateRefreshToken(ctx, RefreshToken{
		UserID:    token.UserID,
		TokenHash: newHash,
		IssuedAt:  now,
		ExpiresAt: now.Add(s.refreshTTL),
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return AuthResponse{}, err
	}

	if err := s.repo.RevokeRefreshToken(ctx, token.ID, &newRecord.ID, now); err != nil {
		return AuthResponse{}, err
	}

	user, err := s.repo.GetUserByID(ctx, token.UserID)
	if err != nil {
		return AuthResponse{}, apperror.Unauthorized("Invalid credentials")
	}

	accessToken, expiresIn, err := s.jwt.GenerateAccessToken(token.UserID, user.Username, now)
	if err != nil {
		return AuthResponse{}, err
	}

	s.logAudit(ctx, audit.Event{
		Action:      "auth.refresh.success",
		ActorUserID: &token.UserID,
		EntityType:  "auth",
		Metadata: map[string]any{
			"ip":         ip,
			"user_agent": userAgent,
		},
	})

	return AuthResponse{AccessToken: accessToken, RefreshToken: newPlain, ExpiresIn: expiresIn}, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken, authHeader, ip, userAgent string) error {
	now := s.now()
	var actor *int64

	if refreshToken != "" {
		hash := HashToken(refreshToken)
		token, err := s.repo.GetRefreshTokenByHash(ctx, hash)
		if err == nil {
			actor = &token.UserID
			_ = s.repo.RevokeRefreshToken(ctx, token.ID, nil, now)
		}
	} else if claims, ok := s.parseBearer(authHeader); ok {
		actor = &claims.UserID
		_ = s.repo.RevokeAllActiveRefreshTokensForUser(ctx, claims.UserID, now)
	}

	s.logAudit(ctx, audit.Event{
		Action:      "auth.logout",
		ActorUserID: actor,
		EntityType:  "auth",
		Metadata: map[string]any{
			"ip":         ip,
			"user_agent": userAgent,
		},
	})

	return nil
}

func (s *Service) Me(ctx context.Context, claims Claims) map[string]any {
	roles := []string{}
	permissions := []string{}
	assignedOrgUnitIDs := []int64{}
	isOrgUnitScopeRestricted := false
	if s.rbacService != nil {
		if resolvedRoles, err := s.rbacService.RoleNamesForUser(ctx, claims.UserID); err == nil {
			roles = resolvedRoles
		}
		if resolvedPerms, err := s.rbacService.PermissionsForUser(ctx, claims.UserID); err == nil {
			permissions = resolvedPerms
		}
	}
	if s.orgUnitScope != nil {
		if ids, err := s.orgUnitScope.GetUserOrgUnitIDs(ctx, claims.UserID); err == nil {
			assignedOrgUnitIDs = ids
		}
		if scope, err := s.orgUnitScope.ResolveScope(ctx, claims.UserID); err == nil {
			isOrgUnitScopeRestricted = scope.Restricted
		}
	}
	return map[string]any{
		"id":                       claims.UserID,
		"username":                 claims.Username,
		"roles":                    roles,
		"permissions":              permissions,
		"assignedOrgUnitIds":       assignedOrgUnitIDs,
		"isOrgUnitScopeRestricted": isOrgUnitScopeRestricted,
	}
}

func (s *Service) ListAPITokens(ctx context.Context) ([]APIToken, error) {
	return s.repo.ListAPITokens(ctx)
}

func (s *Service) CreateAPIToken(ctx context.Context, actorUserID *int64, input APITokenCreateInput, ip, userAgent string) (APITokenCreateResult, error) {
	if !s.apiTokenEnabled {
		return APITokenCreateResult{}, apperror.Unauthorized("API token auth is disabled")
	}
	if strings.TrimSpace(input.Name) == "" {
		return APITokenCreateResult{}, apperror.Unauthorized("Token name is required")
	}

	now := s.now()
	plaintext, err := GenerateAPIToken()
	if err != nil {
		return APITokenCreateResult{}, err
	}

	var expiresAt *time.Time
	ttl := s.apiTokenTTL
	if input.ExpiresInSeconds != nil && *input.ExpiresInSeconds > 0 {
		ttl = time.Duration(*input.ExpiresInSeconds) * time.Second
	}
	if ttl > 0 {
		t := now.Add(ttl)
		expiresAt = &t
	}

	created, err := s.repo.CreateAPIToken(ctx, APIToken{
		Name:            strings.TrimSpace(input.Name),
		TokenHash:       HashAPIToken(s.apiTokenSecret, plaintext),
		Prefix:          APITokenPrefix(plaintext),
		CreatedByUserID: actorUserID,
		ExpiresAt:       expiresAt,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, input.Permissions, input.ModuleScope)
	if err != nil {
		return APITokenCreateResult{}, err
	}

	s.logAudit(ctx, audit.Event{
		Action:      "api_token.create",
		ActorUserID: actorUserID,
		EntityType:  "api_token",
		EntityID:    strPtr(created.Name),
		Metadata: map[string]any{
			"token_id":    created.ID,
			"prefix":      created.Prefix,
			"permissions": input.Permissions,
			"ip":          ip,
			"user_agent":  userAgent,
		},
	})

	return APITokenCreateResult{
		ID:          created.ID,
		Name:        created.Name,
		Prefix:      created.Prefix,
		Token:       plaintext,
		ExpiresAt:   created.ExpiresAt,
		Permissions: input.Permissions,
	}, nil
}

func (s *Service) RevokeAPIToken(ctx context.Context, actorUserID *int64, tokenID int64, ip, userAgent string) (*APIToken, error) {
	now := s.now()
	if err := s.repo.RevokeAPIToken(ctx, tokenID, now); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, apperror.Unauthorized("Token not found")
		}
		return nil, err
	}
	token, err := s.repo.GetAPITokenByID(ctx, tokenID)
	if err != nil {
		return nil, err
	}

	s.logAudit(ctx, audit.Event{
		Action:      "api_token.revoke",
		ActorUserID: actorUserID,
		EntityType:  "api_token",
		EntityID:    strPtr(token.Name),
		Metadata: map[string]any{
			"token_id":   token.ID,
			"prefix":     token.Prefix,
			"ip":         ip,
			"user_agent": userAgent,
		},
	})

	return token, nil
}

func (s *Service) AuthenticateAPIToken(ctx context.Context, plaintext, ip, userAgent string) (Principal, error) {
	if !s.apiTokenEnabled {
		return Principal{}, ErrNotFound
	}
	if plaintext == "" {
		return Principal{}, ErrNotFound
	}

	now := s.now()
	hash := HashAPIToken(s.apiTokenSecret, plaintext)
	token, err := s.repo.GetAPITokenByHash(ctx, hash)
	if err != nil {
		return Principal{}, apperror.Unauthorized("Invalid API token")
	}
	if token.RevokedAt != nil {
		return Principal{}, apperror.Unauthorized("Invalid API token")
	}
	if token.ExpiresAt != nil && now.After(*token.ExpiresAt) {
		return Principal{}, apperror.Unauthorized("Invalid API token")
	}
	permissions, err := s.repo.GetAPITokenPermissions(ctx, token.ID)
	if err != nil {
		return Principal{}, err
	}
	grants := make([]PermissionGrant, 0, len(permissions))
	names := make([]string, 0, len(permissions))
	for _, perm := range permissions {
		grants = append(grants, PermissionGrant{
			Permission:  perm.Permission,
			ModuleScope: perm.ModuleScope,
		})
		names = append(names, perm.Permission)
	}

	go func(tokenID int64, at time.Time) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.repo.UpdateAPITokenLastUsed(ctx, tokenID, at)
	}(token.ID, now)

	return Principal{
		Type:             "api_token",
		ID:               strconv.FormatInt(token.ID, 10),
		APITokenID:       token.ID,
		Permissions:      names,
		PermissionGrants: grants,
	}, nil
}

func (s *Service) SeedDevAdmin(ctx context.Context, username, password string) (*User, error) {
	if strings.TrimSpace(username) == "" || strings.TrimSpace(password) == "" {
		return nil, errors.New("seed credentials must not be empty")
	}
	hash, err := HashPassword(password, s.passwordHashCost)
	if err != nil {
		return nil, err
	}
	return s.repo.EnsureUser(ctx, username, hash, true)
}

func (s *Service) issueTokens(ctx context.Context, userID int64, username string) (AuthResponse, error) {
	now := s.now()
	accessToken, expiresIn, err := s.jwt.GenerateAccessToken(userID, username, now)
	if err != nil {
		return AuthResponse{}, err
	}

	refreshToken, err := GenerateRefreshToken()
	if err != nil {
		return AuthResponse{}, err
	}

	_, err = s.repo.CreateRefreshToken(ctx, RefreshToken{
		UserID:    userID,
		TokenHash: HashToken(refreshToken),
		IssuedAt:  now,
		ExpiresAt: now.Add(s.refreshTTL),
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return AuthResponse{}, err
	}

	return AuthResponse{AccessToken: accessToken, RefreshToken: refreshToken, ExpiresIn: expiresIn}, nil
}

func (s *Service) parseBearer(header string) (Claims, bool) {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return Claims{}, false
	}
	claims, err := s.jwt.ParseAccessToken(parts[1])
	if err != nil {
		return Claims{}, false
	}
	return claims, true
}

func (s *Service) logAudit(ctx context.Context, event audit.Event) {
	if s.auditService == nil {
		return
	}
	_ = s.auditService.Log(ctx, event)
}

func strPtr(value string) *string {
	return &value
}

func optionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
