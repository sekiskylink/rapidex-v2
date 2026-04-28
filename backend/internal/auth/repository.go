package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

var ErrNotFound = errors.New("not found")

type Repository interface {
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	GetUserByID(ctx context.Context, userID int64) (*User, error)
	GetActiveUserByIdentifier(ctx context.Context, identifier string) (*User, error)
	GetRefreshTokenByHash(ctx context.Context, hash string) (*RefreshToken, error)
	CreateRefreshToken(ctx context.Context, token RefreshToken) (*RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenID int64, replacedByTokenID *int64, now time.Time) error
	RevokeAllActiveRefreshTokensForUser(ctx context.Context, userID int64, now time.Time) error
	UpdateUserLastLoginAt(ctx context.Context, userID int64, at time.Time) error
	CreatePasswordResetToken(ctx context.Context, token PasswordResetToken) (*PasswordResetToken, error)
	GetPasswordResetTokenByHash(ctx context.Context, hash string) (*PasswordResetToken, error)
	InvalidateActivePasswordResetTokensForUser(ctx context.Context, userID int64, now time.Time) error
	MarkPasswordResetTokenUsed(ctx context.Context, tokenID int64, now time.Time, consumedFromIP, consumedUserAgent *string) error
	UpdateUserPasswordHash(ctx context.Context, userID int64, passwordHash string, now time.Time) error

	CreateAPIToken(ctx context.Context, token APIToken, permissions []string, moduleScope *string) (*APIToken, error)
	ListAPITokens(ctx context.Context) ([]APIToken, error)
	GetAPITokenByID(ctx context.Context, tokenID int64) (*APIToken, error)
	GetAPITokenByHash(ctx context.Context, hash string) (*APIToken, error)
	GetAPITokenPermissions(ctx context.Context, tokenID int64) ([]APITokenPermission, error)
	RevokeAPIToken(ctx context.Context, tokenID int64, now time.Time) error
	UpdateAPITokenLastUsed(ctx context.Context, tokenID int64, now time.Time) error

	EnsureUser(ctx context.Context, username, passwordHash string, isActive bool) (*User, error)
}

type SQLRepository struct {
	db *sqlx.DB
}

func NewSQLRepository(db *sqlx.DB) *SQLRepository {
	return &SQLRepository{db: db}
}

func (r *SQLRepository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	var user User
	err := r.db.GetContext(ctx, &user, `
		SELECT id, username, password_hash, is_active, last_login_at
		FROM users
		WHERE username = $1
	`, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return &user, nil
}

func (r *SQLRepository) GetUserByID(ctx context.Context, userID int64) (*User, error) {
	var user User
	err := r.db.GetContext(ctx, &user, `
		SELECT id, username, password_hash, is_active, last_login_at
		FROM users
		WHERE id = $1
	`, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &user, nil
}

func (r *SQLRepository) GetActiveUserByIdentifier(ctx context.Context, identifier string) (*User, error) {
	trimmed := strings.TrimSpace(identifier)
	var user User
	err := r.db.GetContext(ctx, &user, `
		SELECT id, username, password_hash, is_active, last_login_at
		FROM users
		WHERE is_active = TRUE
		  AND (
		    LOWER(username) = LOWER($1)
			OR (email IS NOT NULL AND LOWER(email) = LOWER($1))
		  )
	`, trimmed)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get active user by identifier: %w", err)
	}
	return &user, nil
}

func (r *SQLRepository) GetRefreshTokenByHash(ctx context.Context, hash string) (*RefreshToken, error) {
	var token RefreshToken
	err := r.db.GetContext(ctx, &token, `
		SELECT id, user_id, token_hash, issued_at, expires_at, revoked_at, replaced_by_token_id, created_at, updated_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get refresh token: %w", err)
	}
	return &token, nil
}

func (r *SQLRepository) CreateRefreshToken(ctx context.Context, token RefreshToken) (*RefreshToken, error) {
	var created RefreshToken
	err := r.db.GetContext(ctx, &created, `
		INSERT INTO refresh_tokens (user_id, token_hash, issued_at, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, token_hash, issued_at, expires_at, revoked_at, replaced_by_token_id, created_at, updated_at
	`, token.UserID, token.TokenHash, token.IssuedAt, token.ExpiresAt, token.CreatedAt, token.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create refresh token: %w", err)
	}
	return &created, nil
}

func (r *SQLRepository) RevokeRefreshToken(ctx context.Context, tokenID int64, replacedByTokenID *int64, now time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = COALESCE(revoked_at, $2),
		    replaced_by_token_id = COALESCE($3, replaced_by_token_id),
		    updated_at = $2
		WHERE id = $1
	`, tokenID, now, replacedByTokenID)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

func (r *SQLRepository) RevokeAllActiveRefreshTokensForUser(ctx context.Context, userID int64, now time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = COALESCE(revoked_at, $2), updated_at = $2
		WHERE user_id = $1 AND revoked_at IS NULL
	`, userID, now)
	if err != nil {
		return fmt.Errorf("revoke active refresh tokens: %w", err)
	}
	return nil
}

func (r *SQLRepository) UpdateUserLastLoginAt(ctx context.Context, userID int64, at time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users
		SET last_login_at = $2, updated_at = $2
		WHERE id = $1
	`, userID, at)
	if err != nil {
		return fmt.Errorf("update user last login at: %w", err)
	}
	return nil
}

func (r *SQLRepository) CreatePasswordResetToken(ctx context.Context, token PasswordResetToken) (*PasswordResetToken, error) {
	var created PasswordResetToken
	err := r.db.GetContext(ctx, &created, `
		INSERT INTO password_reset_tokens (
		    user_id, token_hash, expires_at, used_at,
		    requested_from_ip, requested_user_agent,
		    consumed_from_ip, consumed_user_agent,
		    created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, user_id, token_hash, expires_at, used_at,
		          requested_from_ip, requested_user_agent,
		          consumed_from_ip, consumed_user_agent,
		          created_at, updated_at
	`, token.UserID, token.TokenHash, token.ExpiresAt, token.UsedAt, token.RequestedFromIP, token.RequestedUserAgent, token.ConsumedFromIP, token.ConsumedUserAgent, token.CreatedAt, token.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create password reset token: %w", err)
	}
	return &created, nil
}

func (r *SQLRepository) GetPasswordResetTokenByHash(ctx context.Context, hash string) (*PasswordResetToken, error) {
	var token PasswordResetToken
	err := r.db.GetContext(ctx, &token, `
		SELECT id, user_id, token_hash, expires_at, used_at,
		       requested_from_ip, requested_user_agent,
		       consumed_from_ip, consumed_user_agent,
		       created_at, updated_at
		FROM password_reset_tokens
		WHERE token_hash = $1
	`, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get password reset token by hash: %w", err)
	}
	return &token, nil
}

func (r *SQLRepository) InvalidateActivePasswordResetTokensForUser(ctx context.Context, userID int64, now time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE password_reset_tokens
		SET used_at = COALESCE(used_at, $2),
		    updated_at = $2
		WHERE user_id = $1
		  AND used_at IS NULL
		  AND expires_at > $2
	`, userID, now)
	if err != nil {
		return fmt.Errorf("invalidate active password reset tokens: %w", err)
	}
	return nil
}

func (r *SQLRepository) MarkPasswordResetTokenUsed(ctx context.Context, tokenID int64, now time.Time, consumedFromIP, consumedUserAgent *string) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE password_reset_tokens
		SET used_at = $2,
		    consumed_from_ip = $3,
		    consumed_user_agent = $4,
		    updated_at = $2
		WHERE id = $1
		  AND used_at IS NULL
	`, tokenID, now, consumedFromIP, consumedUserAgent)
	if err != nil {
		return fmt.Errorf("mark password reset token used: %w", err)
	}
	rows, rowErr := result.RowsAffected()
	if rowErr == nil && rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *SQLRepository) UpdateUserPasswordHash(ctx context.Context, userID int64, passwordHash string, now time.Time) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE users
		SET password_hash = $2,
		    updated_at = $3
		WHERE id = $1
	`, userID, passwordHash, now)
	if err != nil {
		return fmt.Errorf("update user password hash: %w", err)
	}
	rows, rowErr := result.RowsAffected()
	if rowErr == nil && rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *SQLRepository) CreateAPIToken(ctx context.Context, token APIToken, permissions []string, moduleScope *string) (*APIToken, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin create api token transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var created APIToken
	err = tx.GetContext(ctx, &created, `
		INSERT INTO api_tokens (name, token_hash, prefix, created_by_user_id, bound_user_id, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, name, token_hash, prefix, created_by_user_id, bound_user_id, revoked_at, expires_at, last_used_at, created_at, updated_at
	`, token.Name, token.TokenHash, token.Prefix, token.CreatedByUserID, token.BoundUserID, token.ExpiresAt, token.CreatedAt, token.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert api token: %w", err)
	}

	for _, perm := range permissions {
		clean := strings.TrimSpace(perm)
		if clean == "" {
			continue
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO api_token_permissions (api_token_id, permission, module_scope, created_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT DO NOTHING
		`, created.ID, clean, moduleScope, token.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("insert api token permission: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit create api token transaction: %w", err)
	}

	return &created, nil
}

func (r *SQLRepository) ListAPITokens(ctx context.Context) ([]APIToken, error) {
	var tokens []APIToken
	if err := r.db.SelectContext(ctx, &tokens, `
		SELECT id, name, token_hash, prefix, created_by_user_id, bound_user_id, revoked_at, expires_at, last_used_at, created_at, updated_at
		FROM api_tokens
		ORDER BY created_at DESC
	`); err != nil {
		return nil, fmt.Errorf("list api tokens: %w", err)
	}
	return tokens, nil
}

func (r *SQLRepository) GetAPITokenByID(ctx context.Context, tokenID int64) (*APIToken, error) {
	var token APIToken
	if err := r.db.GetContext(ctx, &token, `
		SELECT id, name, token_hash, prefix, created_by_user_id, bound_user_id, revoked_at, expires_at, last_used_at, created_at, updated_at
		FROM api_tokens
		WHERE id = $1
	`, tokenID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get api token by id: %w", err)
	}
	return &token, nil
}

func (r *SQLRepository) GetAPITokenByHash(ctx context.Context, hash string) (*APIToken, error) {
	var token APIToken
	if err := r.db.GetContext(ctx, &token, `
		SELECT id, name, token_hash, prefix, created_by_user_id, bound_user_id, revoked_at, expires_at, last_used_at, created_at, updated_at
		FROM api_tokens
		WHERE token_hash = $1
	`, hash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get api token by hash: %w", err)
	}
	return &token, nil
}

func (r *SQLRepository) GetAPITokenPermissions(ctx context.Context, tokenID int64) ([]APITokenPermission, error) {
	var permissions []APITokenPermission
	if err := r.db.SelectContext(ctx, &permissions, `
		SELECT api_token_id, permission, module_scope
		FROM api_token_permissions
		WHERE api_token_id = $1
	`, tokenID); err != nil {
		return nil, fmt.Errorf("get api token permissions: %w", err)
	}
	return permissions, nil
}

func (r *SQLRepository) RevokeAPIToken(ctx context.Context, tokenID int64, now time.Time) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE api_tokens
		SET revoked_at = COALESCE(revoked_at, $2), updated_at = $2
		WHERE id = $1
	`, tokenID, now)
	if err != nil {
		return fmt.Errorf("revoke api token: %w", err)
	}
	rows, err := result.RowsAffected()
	if err == nil && rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *SQLRepository) UpdateAPITokenLastUsed(ctx context.Context, tokenID int64, now time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE api_tokens
		SET last_used_at = $2, updated_at = $2
		WHERE id = $1
	`, tokenID, now)
	if err != nil {
		return fmt.Errorf("update api token last used: %w", err)
	}
	return nil
}

func (r *SQLRepository) EnsureUser(ctx context.Context, username, passwordHash string, isActive bool) (*User, error) {
	if user, err := r.GetUserByUsername(ctx, username); err == nil {
		return user, nil
	}

	var created User
	err := r.db.GetContext(ctx, &created, `
		INSERT INTO users (username, password_hash, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id, username, password_hash, is_active, last_login_at
	`, username, passwordHash, isActive)
	if err != nil {
		return nil, fmt.Errorf("ensure user: %w", err)
	}
	return &created, nil
}
