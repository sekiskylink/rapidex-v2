package rbac

const (
	PermissionUsersRead      = "users.read"
	PermissionUsersWrite     = "users.write"
	PermissionAuditRead      = "audit.read"
	PermissionSettingsRead   = "settings.read"
	PermissionSettingsWrite  = "settings.write"
	PermissionAPITokensRead  = "api_tokens.read"
	PermissionAPITokensWrite = "api_tokens.write"
)

type PermissionDefinition struct {
	Key         string
	Label       string
	Description string
	Module      string
	Category    string
}

type ModuleDefinition struct {
	ID          string
	Label       string
	NavGroup    string
	BasePath    string
	Permissions []string
	NavItems    []string
}

func BasePermissionRegistry() []PermissionDefinition {
	return []PermissionDefinition{
		{
			Key:         PermissionUsersRead,
			Label:       "Users: Read",
			Description: "View users and administration listings that depend on user read access.",
			Module:      "users",
			Category:    "Administration",
		},
		{
			Key:         PermissionUsersWrite,
			Label:       "Users: Write",
			Description: "Create and update users, roles, and role-permission mappings.",
			Module:      "users",
			Category:    "Administration",
		},
		{
			Key:         PermissionAuditRead,
			Label:       "Audit: Read",
			Description: "View audit log entries and related metadata.",
			Module:      "audit",
			Category:    "Administration",
		},
		{
			Key:         PermissionSettingsRead,
			Label:       "Settings: Read",
			Description: "View platform settings such as login branding and configuration.",
			Module:      "settings",
			Category:    "Settings",
		},
		{
			Key:         PermissionSettingsWrite,
			Label:       "Settings: Write",
			Description: "Update platform settings such as login branding and configuration.",
			Module:      "settings",
			Category:    "Settings",
		},
		{
			Key:         PermissionAPITokensRead,
			Label:       "API Tokens: Read",
			Description: "View API token records.",
			Module:      "api_tokens",
			Category:    "Administration",
		},
		{
			Key:         PermissionAPITokensWrite,
			Label:       "API Tokens: Write",
			Description: "Create and revoke API tokens.",
			Module:      "api_tokens",
			Category:    "Administration",
		},
	}
}

func BaseModuleRegistry() []ModuleDefinition {
	return []ModuleDefinition{
		{
			ID:          "dashboard",
			Label:       "Dashboard",
			NavGroup:    "dashboard",
			BasePath:    "/dashboard",
			Permissions: []string{},
			NavItems:    []string{"dashboard"},
		},
		{
			ID:       "administration",
			Label:    "Administration",
			NavGroup: "administration",
			BasePath: "/users",
			Permissions: []string{
				PermissionUsersRead,
				PermissionUsersWrite,
				PermissionAuditRead,
			},
			NavItems: []string{"users", "roles", "permissions", "audit"},
		},
		{
			ID:       "settings",
			Label:    "Settings",
			NavGroup: "settings",
			BasePath: "/settings",
			Permissions: []string{
				PermissionSettingsRead,
				PermissionSettingsWrite,
			},
			NavItems: []string{"settings"},
		},
	}
}

func ModuleIDForPermission(permission string) (string, bool) {
	switch permission {
	case PermissionUsersRead, PermissionUsersWrite, PermissionAuditRead, PermissionAPITokensRead, PermissionAPITokensWrite:
		return "administration", true
	case PermissionSettingsRead, PermissionSettingsWrite:
		return "settings", true
	default:
		return "", false
	}
}
