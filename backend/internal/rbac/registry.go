package rbac

const (
	PermissionUsersRead         = "users.read"
	PermissionUsersWrite        = "users.write"
	PermissionAuditRead         = "audit.read"
	PermissionSettingsRead      = "settings.read"
	PermissionSettingsWrite     = "settings.write"
	PermissionAPITokensRead     = "api_tokens.read"
	PermissionAPITokensWrite    = "api_tokens.write"
	PermissionServersRead       = "servers.read"
	PermissionServersWrite      = "servers.write"
	PermissionRequestsRead      = "requests.read"
	PermissionRequestsWrite     = "requests.write"
	PermissionDeliveriesRead    = "deliveries.read"
	PermissionDeliveriesWrite   = "deliveries.write"
	PermissionJobsRead          = "jobs.read"
	PermissionJobsWrite         = "jobs.write"
	PermissionObservabilityRead = "observability.read"
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
		{
			Key:         PermissionServersRead,
			Label:       "Servers: Read",
			Description: "View Sukumad integration servers.",
			Module:      "servers",
			Category:    "Sukumad",
		},
		{
			Key:         PermissionServersWrite,
			Label:       "Servers: Write",
			Description: "Create and update Sukumad integration servers.",
			Module:      "servers",
			Category:    "Sukumad",
		},
		{
			Key:         PermissionRequestsRead,
			Label:       "Requests: Read",
			Description: "View Sukumad exchange requests.",
			Module:      "requests",
			Category:    "Sukumad",
		},
		{
			Key:         PermissionRequestsWrite,
			Label:       "Requests: Write",
			Description: "Create and update Sukumad exchange requests.",
			Module:      "requests",
			Category:    "Sukumad",
		},
		{
			Key:         PermissionDeliveriesRead,
			Label:       "Deliveries: Read",
			Description: "View Sukumad delivery attempts and histories.",
			Module:      "deliveries",
			Category:    "Sukumad",
		},
		{
			Key:         PermissionDeliveriesWrite,
			Label:       "Deliveries: Write",
			Description: "Manage Sukumad delivery retries and operations.",
			Module:      "deliveries",
			Category:    "Sukumad",
		},
		{
			Key:         PermissionJobsRead,
			Label:       "Jobs: Read",
			Description: "View Sukumad worker jobs and background processing status.",
			Module:      "jobs",
			Category:    "Sukumad",
		},
		{
			Key:         PermissionJobsWrite,
			Label:       "Jobs: Write",
			Description: "Manage Sukumad worker jobs and background processing.",
			Module:      "jobs",
			Category:    "Sukumad",
		},
		{
			Key:         PermissionObservabilityRead,
			Label:       "Observability: Read",
			Description: "View Sukumad operational observability surfaces.",
			Module:      "observability",
			Category:    "Sukumad",
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
		{
			ID:       "servers",
			Label:    "Servers",
			NavGroup: "sukumad",
			BasePath: "/servers",
			Permissions: []string{
				PermissionServersRead,
				PermissionServersWrite,
			},
			NavItems: []string{"servers"},
		},
		{
			ID:       "requests",
			Label:    "Requests",
			NavGroup: "sukumad",
			BasePath: "/requests",
			Permissions: []string{
				PermissionRequestsRead,
				PermissionRequestsWrite,
			},
			NavItems: []string{"requests"},
		},
		{
			ID:       "deliveries",
			Label:    "Deliveries",
			NavGroup: "sukumad",
			BasePath: "/deliveries",
			Permissions: []string{
				PermissionDeliveriesRead,
				PermissionDeliveriesWrite,
			},
			NavItems: []string{"deliveries"},
		},
		{
			ID:       "jobs",
			Label:    "Jobs",
			NavGroup: "sukumad",
			BasePath: "/jobs",
			Permissions: []string{
				PermissionJobsRead,
				PermissionJobsWrite,
			},
			NavItems: []string{"jobs"},
		},
		{
			ID:       "observability",
			Label:    "Observability",
			NavGroup: "sukumad",
			BasePath: "/observability",
			Permissions: []string{
				PermissionObservabilityRead,
			},
			NavItems: []string{"observability"},
		},
	}
}

func ModuleIDForPermission(permission string) (string, bool) {
	switch permission {
	case PermissionUsersRead, PermissionUsersWrite, PermissionAuditRead, PermissionAPITokensRead, PermissionAPITokensWrite:
		return "administration", true
	case PermissionSettingsRead, PermissionSettingsWrite:
		return "settings", true
	case PermissionServersRead, PermissionServersWrite:
		return "servers", true
	case PermissionRequestsRead, PermissionRequestsWrite:
		return "requests", true
	case PermissionDeliveriesRead, PermissionDeliveriesWrite:
		return "deliveries", true
	case PermissionJobsRead, PermissionJobsWrite:
		return "jobs", true
	case PermissionObservabilityRead:
		return "observability", true
	default:
		return "", false
	}
}
