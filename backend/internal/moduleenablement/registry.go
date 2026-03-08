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
}

var registry = []Definition{
	{
		ModuleID:         "dashboard",
		FlagKey:          "modules.dashboard.enabled",
		EnabledByDefault: true,
		Description:      "Authenticated dashboard shell entry.",
		Scope:            ScopeFullStack,
	},
	{
		ModuleID:         "administration",
		FlagKey:          "modules.administration.enabled",
		EnabledByDefault: true,
		Description:      "RBAC and audit administration surfaces.",
		Scope:            ScopeFullStack,
	},
	{
		ModuleID:         "settings",
		FlagKey:          "modules.settings.enabled",
		EnabledByDefault: true,
		Description:      "System configuration and branding surfaces.",
		Scope:            ScopeFullStack,
	},
}

func Definitions() []Definition {
	out := make([]Definition, len(registry))
	copy(out, registry)
	return out
}

func ResolveEffective(overrides map[string]bool) []EffectiveModule {
	items := make([]EffectiveModule, 0, len(registry))
	for _, definition := range registry {
		enabled := definition.EnabledByDefault
		source := "default"
		if overrides != nil {
			if value, ok := overrides[definition.FlagKey]; ok {
				enabled = value
				source = "config"
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
		})
	}
	return items
}

func IsModuleEnabled(moduleID string, overrides map[string]bool) bool {
	for _, definition := range registry {
		if definition.ModuleID != moduleID {
			continue
		}
		if overrides != nil {
			if value, ok := overrides[definition.FlagKey]; ok {
				return value
			}
		}
		return definition.EnabledByDefault
	}
	return true
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
