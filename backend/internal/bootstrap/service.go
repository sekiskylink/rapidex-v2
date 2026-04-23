package bootstrap

import (
	"context"
	"errors"
	"strings"
	"time"

	"basepro/backend/internal/auth"
	"basepro/backend/internal/moduleenablement"
	"basepro/backend/internal/rbac"
	"basepro/backend/internal/settings"
	"basepro/backend/internal/sukumad/userorg"
)

type AppInfo struct {
	Version   string
	Commit    string
	BuildDate string
}

type RuntimeInfo struct {
	Environment       string
	APITokenEnabled   bool
	APIHeaderName     string
	APITokenAllowAuth bool
}

type RuntimeInfoProvider func() RuntimeInfo
type ConfigFlagsProvider func() map[string]bool

type Service struct {
	appInfo            AppInfo
	runtimeInfo        RuntimeInfoProvider
	settingsService    *settings.Service
	moduleFlagsService *moduleenablement.Service
	configFlags        ConfigFlagsProvider
	rbacService        *rbac.Service
	orgUnitScope       *userorg.Service
	now                func() time.Time
}

func NewService(
	appInfo AppInfo,
	runtimeInfo RuntimeInfoProvider,
	settingsService *settings.Service,
	moduleFlagsService *moduleenablement.Service,
	configFlags ConfigFlagsProvider,
	rbacService *rbac.Service,
) *Service {
	if runtimeInfo == nil {
		runtimeInfo = func() RuntimeInfo { return RuntimeInfo{} }
	}
	if configFlags == nil {
		configFlags = func() map[string]bool { return nil }
	}
	return &Service{
		appInfo:            appInfo,
		runtimeInfo:        runtimeInfo,
		settingsService:    settingsService,
		moduleFlagsService: moduleFlagsService,
		configFlags:        configFlags,
		rbacService:        rbacService,
		now:                func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Build(ctx context.Context, principal *auth.Principal) (Response, error) {
	if s.settingsService == nil {
		return Response{}, errors.New("settings service is not configured")
	}

	branding, err := s.settingsService.GetLoginBranding(ctx)
	if err != nil {
		return Response{}, err
	}

	effectiveModules := []moduleenablement.EffectiveModule{}
	if s.moduleFlagsService != nil {
		effectiveModules = s.moduleFlagsService.ListEffective(s.configFlags())
	}

	currentRuntime := s.runtimeInfo()
	summary, canReadSettings, canWriteSettings := s.principalSummary(ctx, principal)

	return Response{
		Version:     1,
		GeneratedAt: s.now(),
		App: AppMetadata{
			Version:   s.appInfo.Version,
			Commit:    s.appInfo.Commit,
			BuildDate: s.appInfo.BuildDate,
		},
		Branding: branding,
		Modules:  effectiveModules,
		Capabilities: CapabilitySummary{
			CanAccessSettings: canWriteSettings || hasAdminRole(summary),
			Auth: AuthCapabilities{
				HasJWTSession:      summary != nil && summary.Type == "user",
				APIHeaderName:      currentRuntime.APIHeaderName,
				APIHeaderAllowAuth: currentRuntime.APITokenAllowAuth,
				APITokenEnabled:    currentRuntime.APITokenEnabled,
			},
			Settings: SettingsCapabilities{
				CanRead:        canReadSettings,
				CanWrite:       canWriteSettings || hasAdminRole(summary),
				WriteRequires:  []string{"admin", rbac.PermissionSettingsWrite},
				EnforcedByAPI:  true,
				Authorization:  "admin_or_settings.write",
				ModuleRequired: "settings",
			},
		},
		Runtime: RuntimeMetadata{
			ServerTimeRFC3339: s.now().Format(time.RFC3339),
			Environment:       normalizeEnvironment(currentRuntime.Environment),
			APIVersion:        "v1",
		},
		Cache: CacheHints{
			Cacheable:          true,
			ContainsSecrets:    false,
			SchemaVersion:      1,
			MaxStaleSeconds:    300,
			OfflineSafePayload: true,
		},
		Principal: summary,
	}, nil
}

func (s *Service) WithUserOrgScope(scope *userorg.Service) *Service {
	if s == nil {
		return s
	}
	s.orgUnitScope = scope
	return s
}

func (s *Service) principalSummary(ctx context.Context, principal *auth.Principal) (*PrincipalSummary, bool, bool) {
	if principal == nil {
		return nil, false, false
	}
	switch principal.Type {
	case "api_token":
		perms := grantsToPermissionList(principal.PermissionGrants)
		return &PrincipalSummary{
			Type:        principal.Type,
			Permissions: perms,
		}, hasPermission(perms, rbac.PermissionSettingsRead), hasPermission(perms, rbac.PermissionSettingsWrite)
	case "user":
		var roles []string
		var perms []string
		assignedOrgUnitIDs := []int64{}
		isOrgUnitScopeRestricted := false
		if s.rbacService != nil {
			if resolvedRoles, err := s.rbacService.RoleNamesForUser(ctx, principal.UserID); err == nil {
				roles = resolvedRoles
			}
			if resolvedPerms, err := s.rbacService.PermissionsForUser(ctx, principal.UserID); err == nil {
				perms = resolvedPerms
			}
		}
		if s.orgUnitScope != nil {
			if ids, err := s.orgUnitScope.GetUserOrgUnitIDs(ctx, principal.UserID); err == nil {
				assignedOrgUnitIDs = ids
			}
			if scope, err := s.orgUnitScope.ResolveScope(ctx, principal.UserID); err == nil {
				isOrgUnitScopeRestricted = scope.Restricted
			}
		}
		return &PrincipalSummary{
			Type:                     principal.Type,
			UserID:                   principal.UserID,
			Username:                 principal.Username,
			Roles:                    roles,
			Permissions:              perms,
			AssignedOrgUnitIDs:       assignedOrgUnitIDs,
			IsOrgUnitScopeRestricted: isOrgUnitScopeRestricted,
		}, hasPermission(perms, rbac.PermissionSettingsRead), hasPermission(perms, rbac.PermissionSettingsWrite)
	default:
		return &PrincipalSummary{Type: principal.Type}, false, false
	}
}

func grantsToPermissionList(grants []auth.PermissionGrant) []string {
	if len(grants) == 0 {
		return nil
	}
	out := make([]string, 0, len(grants))
	for _, grant := range grants {
		if grant.ModuleScope == nil || strings.TrimSpace(*grant.ModuleScope) == "" {
			out = append(out, grant.Permission)
			continue
		}
		out = append(out, grant.Permission+"@"+strings.TrimSpace(*grant.ModuleScope))
	}
	return out
}

func hasPermission(perms []string, target string) bool {
	for _, permission := range perms {
		if permission == target {
			return true
		}
	}
	return false
}

func hasAdminRole(principal *PrincipalSummary) bool {
	if principal == nil {
		return false
	}
	for _, role := range principal.Roles {
		if strings.EqualFold(strings.TrimSpace(role), "admin") {
			return true
		}
	}
	return false
}

func normalizeEnvironment(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "unknown"
	}
	return trimmed
}
