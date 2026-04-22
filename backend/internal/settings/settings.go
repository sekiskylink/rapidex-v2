package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/audit"
	"github.com/jmoiron/sqlx"
)

const (
	brandingCategory = "branding"
	loginBrandingKey = "login"
)

var ErrNotFound = errors.New("setting not found")

type Repository interface {
	Get(ctx context.Context, category, key string) (json.RawMessage, error)
	Upsert(ctx context.Context, category, key string, value json.RawMessage, updatedByUserID *int64, now time.Time) error
}

type SQLRepository struct {
	db *sqlx.DB
}

func NewSQLRepository(db *sqlx.DB) *SQLRepository {
	return &SQLRepository{db: db}
}

func (r *SQLRepository) Get(ctx context.Context, category, key string) (json.RawMessage, error) {
	var raw []byte
	err := r.db.GetContext(ctx, &raw, `
		SELECT value_json
		FROM app_settings
		WHERE category = $1 AND key = $2
	`, category, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get setting: %w", err)
	}
	return json.RawMessage(raw), nil
}

func (r *SQLRepository) Upsert(ctx context.Context, category, key string, value json.RawMessage, updatedByUserID *int64, now time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO app_settings (category, key, value_json, updated_by_user_id, created_at, updated_at)
		VALUES ($1, $2, $3::jsonb, $4, $5, $5)
		ON CONFLICT (category, key)
		DO UPDATE SET value_json = EXCLUDED.value_json,
		              updated_by_user_id = EXCLUDED.updated_by_user_id,
		              updated_at = EXCLUDED.updated_at
	`, category, key, []byte(value), updatedByUserID, now)
	if err != nil {
		return fmt.Errorf("upsert setting: %w", err)
	}
	return nil
}

type LoginBranding struct {
	ApplicationDisplayName string  `json:"applicationDisplayName"`
	LoginImageURL          *string `json:"loginImageUrl,omitempty"`
	LoginImageAssetPath    *string `json:"loginImageAssetPath,omitempty"`
	ImageConfigured        bool    `json:"imageConfigured"`
}

type LoginBrandingUpdateInput struct {
	ApplicationDisplayName string  `json:"applicationDisplayName"`
	LoginImageURL          *string `json:"loginImageUrl,omitempty"`
	LoginImageAssetPath    *string `json:"loginImageAssetPath,omitempty"`
}

type loginBrandingStored struct {
	ApplicationDisplayName string  `json:"applicationDisplayName"`
	LoginImageURL          *string `json:"loginImageUrl,omitempty"`
	LoginImageAssetPath    *string `json:"loginImageAssetPath,omitempty"`
}

type Service struct {
	repo                  Repository
	auditService          *audit.Service
	runtimeConfigProvider func() map[string]any
	rapidProServerLookup  rapidProServerLookup
	rapidProFieldClient   rapidProFieldClient
}

func NewService(repo Repository, auditService *audit.Service) *Service {
	return &Service{repo: repo, auditService: auditService}
}

func (s *Service) WithRuntimeConfigProvider(provider func() map[string]any) *Service {
	s.runtimeConfigProvider = provider
	return s
}

func (s *Service) GetLoginBranding(ctx context.Context) (LoginBranding, error) {
	raw, err := s.repo.Get(ctx, brandingCategory, loginBrandingKey)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return defaultLoginBranding(), nil
		}
		return LoginBranding{}, err
	}

	var stored loginBrandingStored
	if unmarshalErr := json.Unmarshal(raw, &stored); unmarshalErr != nil {
		return defaultLoginBranding(), nil
	}

	result := LoginBranding{
		ApplicationDisplayName: strings.TrimSpace(stored.ApplicationDisplayName),
		LoginImageURL:          normalizeOptionalString(stored.LoginImageURL),
		LoginImageAssetPath:    normalizeOptionalString(stored.LoginImageAssetPath),
	}
	if result.ApplicationDisplayName == "" {
		result.ApplicationDisplayName = defaultLoginBranding().ApplicationDisplayName
	}
	result.ImageConfigured = result.LoginImageURL != nil || result.LoginImageAssetPath != nil
	return result, nil
}

func (s *Service) GetRuntimeConfig(context.Context) (map[string]any, error) {
	if s.runtimeConfigProvider == nil {
		return map[string]any{}, nil
	}

	return sanitizeRuntimeConfigMap(s.runtimeConfigProvider()), nil
}

func (s *Service) UpdateLoginBranding(ctx context.Context, input LoginBrandingUpdateInput, actorUserID *int64) (LoginBranding, error) {
	displayName := strings.TrimSpace(input.ApplicationDisplayName)
	if displayName == "" {
		return LoginBranding{}, apperror.ValidationWithDetails("validation failed", map[string]any{
			"applicationDisplayName": []string{"is required"},
		})
	}
	if len(displayName) > 120 {
		return LoginBranding{}, apperror.ValidationWithDetails("validation failed", map[string]any{
			"applicationDisplayName": []string{"must be 120 characters or fewer"},
		})
	}

	loginImageURL := normalizeOptionalString(input.LoginImageURL)
	if loginImageURL != nil {
		parsed, parseErr := url.Parse(*loginImageURL)
		if parseErr != nil || !parsed.IsAbs() || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return LoginBranding{}, apperror.ValidationWithDetails("validation failed", map[string]any{
				"loginImageUrl": []string{"must be an absolute http(s) URL"},
			})
		}
	}

	loginImageAssetPath := normalizeOptionalString(input.LoginImageAssetPath)
	if loginImageAssetPath != nil && len(*loginImageAssetPath) > 512 {
		return LoginBranding{}, apperror.ValidationWithDetails("validation failed", map[string]any{
			"loginImageAssetPath": []string{"must be 512 characters or fewer"},
		})
	}

	stored := loginBrandingStored{
		ApplicationDisplayName: displayName,
		LoginImageURL:          loginImageURL,
		LoginImageAssetPath:    loginImageAssetPath,
	}
	payload, err := json.Marshal(stored)
	if err != nil {
		return LoginBranding{}, err
	}

	now := time.Now().UTC()
	if err := s.repo.Upsert(ctx, brandingCategory, loginBrandingKey, payload, actorUserID, now); err != nil {
		return LoginBranding{}, err
	}

	result := LoginBranding{
		ApplicationDisplayName: displayName,
		LoginImageURL:          loginImageURL,
		LoginImageAssetPath:    loginImageAssetPath,
		ImageConfigured:        loginImageURL != nil || loginImageAssetPath != nil,
	}
	s.logAudit(ctx, audit.Event{
		Action:      "settings.login_branding.update",
		ActorUserID: actorUserID,
		EntityType:  "settings",
		EntityID:    strPtr("branding.login"),
		Metadata: map[string]any{
			"applicationDisplayName": displayName,
			"imageConfigured":        result.ImageConfigured,
		},
	})
	return result, nil
}

func (s *Service) logAudit(ctx context.Context, event audit.Event) {
	if s.auditService == nil {
		return
	}
	_ = s.auditService.Log(ctx, event)
}

func normalizeOptionalString(in *string) *string {
	if in == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*in)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func defaultLoginBranding() LoginBranding {
	return LoginBranding{
		ApplicationDisplayName: "BasePro",
		ImageConfigured:        false,
	}
}

func strPtr(v string) *string {
	return &v
}

var sensitiveConfigKeys = map[string]struct{}{
	"password":      {},
	"secret":        {},
	"token":         {},
	"jwt":           {},
	"jwtsigningkey": {},
	"signingkey":    {},
	"signing_key":   {},
	"dsn":           {},
	"authorization": {},
}

var sensitiveConfigKeyParts = map[string]struct{}{
	"password": {},
	"secret":   {},
}

func sanitizeRuntimeConfigMap(input map[string]any) map[string]any {
	return sanitizeConfigValue("", input).(map[string]any)
}

func sanitizeConfigValue(key string, value any) any {
	switch typed := value.(type) {
	case map[string]any:
		sanitized := make(map[string]any, len(typed))
		for childKey, childValue := range typed {
			if isSensitiveConfigKey(childKey) {
				if strings.EqualFold(childKey, "dsn") {
					if text, ok := childValue.(string); ok {
						sanitized[childKey] = maskDSN(text)
						continue
					}
				}
				sanitized[childKey] = "[masked]"
				continue
			}
			sanitized[childKey] = sanitizeConfigValue(childKey, childValue)
		}
		return sanitized
	case []any:
		items := make([]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, sanitizeConfigValue(key, item))
		}
		return items
	case string:
		if strings.EqualFold(key, "dsn") {
			return maskDSN(typed)
		}
		if isSensitiveConfigKey(key) {
			return "[masked]"
		}
		return typed
	default:
		return value
	}
}

func isSensitiveConfigKey(key string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(key))
	normalized := strings.ReplaceAll(strings.ReplaceAll(trimmed, "-", ""), "_", "")
	if _, ok := sensitiveConfigKeys[normalized]; ok {
		return true
	}
	parts := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == '-' || r == '_'
	})
	for _, part := range parts {
		if _, ok := sensitiveConfigKeyParts[part]; ok {
			return true
		}
	}
	for sensitiveKey := range sensitiveConfigKeys {
		candidate := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(sensitiveKey, "-", ""), "_", ""))
		if normalized == candidate {
			return true
		}
	}
	return false
}

func maskDSN(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}

	parsed, err := url.Parse(trimmed)
	if err == nil && parsed.Scheme != "" {
		if parsed.User != nil {
			username := parsed.User.Username()
			if _, hasPassword := parsed.User.Password(); hasPassword {
				parsed.User = url.UserPassword(username, "[masked]")
			}
		}
		query := parsed.Query()
		for key := range query {
			if isSensitiveConfigKey(key) {
				query.Set(key, "[masked]")
			}
		}
		parsed.RawQuery = query.Encode()
		return parsed.String()
	}

	for _, token := range []string{"password=", "passwd=", "pwd=", "token=", "secret="} {
		index := strings.Index(strings.ToLower(trimmed), token)
		if index >= 0 {
			start := index + len(token)
			end := len(trimmed)
			for i := start; i < len(trimmed); i++ {
				if trimmed[i] == ' ' || trimmed[i] == ';' || trimmed[i] == '&' {
					end = i
					break
				}
			}
			return trimmed[:start] + "[masked]" + trimmed[end:]
		}
	}

	return "[masked]"
}
