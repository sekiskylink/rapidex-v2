package bootstrap

import (
	"time"

	"basepro/backend/internal/moduleenablement"
	"basepro/backend/internal/settings"
)

type AppMetadata struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"buildDate"`
}

type RuntimeMetadata struct {
	ServerTimeRFC3339 string `json:"serverTime"`
	Environment       string `json:"environment"`
	APIVersion        string `json:"apiVersion"`
}

type AuthCapabilities struct {
	HasJWTSession      bool   `json:"hasJwtSession"`
	APIHeaderName      string `json:"apiHeaderName"`
	APIHeaderAllowAuth bool   `json:"apiHeaderAllowBearer"`
	APITokenEnabled    bool   `json:"apiTokenEnabled"`
}

type SettingsCapabilities struct {
	CanRead        bool     `json:"canRead"`
	CanWrite       bool     `json:"canWrite"`
	WriteRequires  []string `json:"writeRequiresAny"`
	EnforcedByAPI  bool     `json:"enforcedByApi"`
	Authorization  string   `json:"authorization"`
	ModuleRequired string   `json:"moduleRequired"`
}

type CapabilitySummary struct {
	CanAccessSettings bool                 `json:"canAccessSettings"`
	Auth              AuthCapabilities     `json:"auth"`
	Settings          SettingsCapabilities `json:"settings"`
}

type CacheHints struct {
	Cacheable          bool `json:"cacheable"`
	ContainsSecrets    bool `json:"containsSecrets"`
	SchemaVersion      int  `json:"schemaVersion"`
	MaxStaleSeconds    int  `json:"maxStaleSeconds"`
	OfflineSafePayload bool `json:"offlineSafePayload"`
}

type PrincipalSummary struct {
	Type                     string   `json:"type"`
	UserID                   int64    `json:"userId,omitempty"`
	Username                 string   `json:"username,omitempty"`
	Roles                    []string `json:"roles,omitempty"`
	Permissions              []string `json:"permissions,omitempty"`
	AssignedOrgUnitIDs       []int64  `json:"assignedOrgUnitIds,omitempty"`
	IsOrgUnitScopeRestricted bool     `json:"isOrgUnitScopeRestricted,omitempty"`
}

type Response struct {
	Version      int                                `json:"version"`
	GeneratedAt  time.Time                          `json:"generatedAt"`
	App          AppMetadata                        `json:"app"`
	Branding     settings.LoginBranding             `json:"branding"`
	Modules      []moduleenablement.EffectiveModule `json:"modules"`
	Capabilities CapabilitySummary                  `json:"capabilities"`
	Runtime      RuntimeMetadata                    `json:"runtime"`
	Cache        CacheHints                         `json:"cache"`
	Principal    *PrincipalSummary                  `json:"principal,omitempty"`
}
