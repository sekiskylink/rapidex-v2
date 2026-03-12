package moduleenablement

import "fmt"

type Scope string

const (
	ScopeBackend   Scope = "backend"
	ScopeDesktop   Scope = "desktop"
	ScopeWeb       Scope = "web"
	ScopeFullStack Scope = "full-stack"
)

type Definition struct {
	ModuleID         string
	FlagKey          string
	EnabledByDefault bool
	Description      string
	Scope            Scope
	Experimental     bool
	AdminControl     AdminControl
}

type EffectiveModule struct {
	ModuleID         string `json:"moduleId"`
	FlagKey          string `json:"flagKey"`
	Enabled          bool   `json:"enabled"`
	EnabledByDefault bool   `json:"enabledByDefault"`
	Description      string `json:"description,omitempty"`
	Scope            Scope  `json:"scope,omitempty"`
	Experimental     bool   `json:"experimental,omitempty"`
	Source           string `json:"source"`
	AdminControl     string `json:"adminControl"`
	Editable         bool   `json:"editable"`
}

type AdminControl string

const (
	AdminControlStatic   AdminControl = "static"
	AdminControlRuntime  AdminControl = "runtime"
	AdminControlReadOnly AdminControl = "read-only"
)

var registry = []Definition{
	{
		ModuleID:         "dashboard",
		FlagKey:          "modules.dashboard.enabled",
		EnabledByDefault: true,
		Description:      "Authenticated dashboard shell entry.",
		Scope:            ScopeFullStack,
		AdminControl:     AdminControlStatic,
	},
	{
		ModuleID:         "administration",
		FlagKey:          "modules.administration.enabled",
		EnabledByDefault: true,
		Description:      "RBAC and audit administration surfaces.",
		Scope:            ScopeFullStack,
		AdminControl:     AdminControlRuntime,
	},
	{
		ModuleID:         "settings",
		FlagKey:          "modules.settings.enabled",
		EnabledByDefault: true,
		Description:      "System configuration and branding surfaces.",
		Scope:            ScopeFullStack,
		AdminControl:     AdminControlStatic,
	},
	{
		ModuleID:         "servers",
		FlagKey:          "modules.servers.enabled",
		EnabledByDefault: true,
		Description:      "Sukumad integration server management surfaces.",
		Scope:            ScopeFullStack,
		AdminControl:     AdminControlStatic,
	},
	{
		ModuleID:         "requests",
		FlagKey:          "modules.requests.enabled",
		EnabledByDefault: true,
		Description:      "Sukumad request lifecycle surfaces.",
		Scope:            ScopeFullStack,
		AdminControl:     AdminControlStatic,
	},
	{
		ModuleID:         "deliveries",
		FlagKey:          "modules.deliveries.enabled",
		EnabledByDefault: true,
		Description:      "Sukumad delivery monitoring surfaces.",
		Scope:            ScopeFullStack,
		AdminControl:     AdminControlStatic,
	},
	{
		ModuleID:         "jobs",
		FlagKey:          "modules.jobs.enabled",
		EnabledByDefault: true,
		Description:      "Sukumad worker and job monitoring surfaces.",
		Scope:            ScopeFullStack,
		AdminControl:     AdminControlStatic,
	},
	{
		ModuleID:         "observability",
		FlagKey:          "modules.observability.enabled",
		EnabledByDefault: true,
		Description:      "Sukumad observability surfaces.",
		Scope:            ScopeFullStack,
		AdminControl:     AdminControlStatic,
	},
}

func Definitions() []Definition {
	out := make([]Definition, len(registry))
	copy(out, registry)
	return out
}

func ResolveEffective(configOverrides map[string]bool, runtimeOverrides map[string]bool) []EffectiveModule {
	items := make([]EffectiveModule, 0, len(registry))
	for _, definition := range registry {
		enabled := definition.EnabledByDefault
		source := "default"
		if configOverrides != nil {
			if value, ok := configOverrides[definition.FlagKey]; ok {
				enabled = value
				source = "config"
			}
		}
		if runtimeOverrides != nil && isRuntimeEditable(definition) {
			if value, ok := runtimeOverrides[definition.FlagKey]; ok {
				enabled = value
				source = "runtime"
			}
		}
		items = append(items, EffectiveModule{
			ModuleID:         definition.ModuleID,
			FlagKey:          definition.FlagKey,
			Enabled:          enabled,
			EnabledByDefault: definition.EnabledByDefault,
			Description:      definition.Description,
			Scope:            definition.Scope,
			Experimental:     definition.Experimental,
			Source:           source,
			AdminControl:     string(resolveAdminControl(definition)),
			Editable:         isRuntimeEditable(definition),
		})
	}
	return items
}

func IsModuleEnabled(moduleID string, configOverrides map[string]bool, runtimeOverrides map[string]bool) bool {
	for _, definition := range registry {
		if definition.ModuleID != moduleID {
			continue
		}
		if runtimeOverrides != nil && isRuntimeEditable(definition) {
			if value, ok := runtimeOverrides[definition.FlagKey]; ok {
				return value
			}
		}
		if configOverrides != nil {
			if value, ok := configOverrides[definition.FlagKey]; ok {
				return value
			}
		}
		return definition.EnabledByDefault
	}
	return true
}

func EditableDefinitionByModuleID(moduleID string) (Definition, bool) {
	for _, definition := range registry {
		if definition.ModuleID == moduleID && isRuntimeEditable(definition) {
			return definition, true
		}
	}
	return Definition{}, false
}

func resolveAdminControl(definition Definition) AdminControl {
	if definition.Experimental && definition.AdminControl == AdminControlRuntime {
		return AdminControlReadOnly
	}
	return definition.AdminControl
}

func isRuntimeEditable(definition Definition) bool {
	return resolveAdminControl(definition) == AdminControlRuntime
}

func ValidateOverrides(overrides map[string]bool) error {
	if len(overrides) == 0 {
		return nil
	}

	allowed := make(map[string]struct{}, len(registry))
	for _, definition := range registry {
		allowed[definition.FlagKey] = struct{}{}
	}

	for key := range overrides {
		if _, ok := allowed[key]; !ok {
			return fmt.Errorf("modules.flags contains unknown key %q", key)
		}
	}
	return nil
}
