package auth

import "time"

const (
	ClaimsContextKey    = "auth_claims"
	PrincipalContextKey = "principal"
)

type User struct {
	ID           int64  `db:"id"`
	Username     string `db:"username"`
	PasswordHash string `db:"password_hash"`
	IsActive     bool   `db:"is_active"`
}

type RefreshToken struct {
	ID                int64      `db:"id"`
	UserID            int64      `db:"user_id"`
	TokenHash         string     `db:"token_hash"`
	IssuedAt          time.Time  `db:"issued_at"`
	ExpiresAt         time.Time  `db:"expires_at"`
	RevokedAt         *time.Time `db:"revoked_at"`
	ReplacedByTokenID *int64     `db:"replaced_by_token_id"`
	CreatedAt         time.Time  `db:"created_at"`
	UpdatedAt         time.Time  `db:"updated_at"`
}

type APIToken struct {
	ID              int64      `db:"id"`
	Name            string     `db:"name"`
	TokenHash       string     `db:"token_hash"`
	Prefix          string     `db:"prefix"`
	CreatedByUserID *int64     `db:"created_by_user_id"`
	RevokedAt       *time.Time `db:"revoked_at"`
	ExpiresAt       *time.Time `db:"expires_at"`
	LastUsedAt      *time.Time `db:"last_used_at"`
	CreatedAt       time.Time  `db:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"`
}

type APITokenPermission struct {
	APITokenID  int64   `db:"api_token_id"`
	Permission  string  `db:"permission"`
	ModuleScope *string `db:"module_scope"`
}

type PermissionGrant struct {
	Permission  string  `json:"permission"`
	ModuleScope *string `json:"moduleScope,omitempty"`
}

type APITokenCreateInput struct {
	Name             string
	CreatedByUserID  *int64
	ExpiresInSeconds *int64
	Permissions      []string
	ModuleScope      *string
}

type APITokenCreateResult struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Prefix      string     `json:"prefix"`
	Token       string     `json:"token"`
	ExpiresAt   *time.Time `json:"expiresAt"`
	Permissions []string   `json:"permissions"`
}

type AuthResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
}

type Claims struct {
	UserID    int64
	Username  string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

type Principal struct {
	Type             string
	ID               string
	UserID           int64
	Username         string
	APITokenID       int64
	Roles            []string
	Permissions      []string
	PermissionGrants []PermissionGrant
}
