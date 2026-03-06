package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// App struct
type App struct {
	ctx context.Context
}

type Settings struct {
	APIBaseURL            string                `json:"apiBaseUrl"`
	AuthMode              string                `json:"authMode"`
	APIToken              *string               `json:"apiToken,omitempty"`
	RefreshToken          *string               `json:"refreshToken,omitempty"`
	RequestTimeoutSeconds int                   `json:"requestTimeoutSeconds"`
	UIPrefs               UIPrefs               `json:"uiPrefs"`
	TablePrefs            map[string]TablePrefs `json:"tablePrefs"`
}

type SettingsPatch struct {
	APIBaseURL            *string                `json:"apiBaseUrl,omitempty"`
	AuthMode              *string                `json:"authMode,omitempty"`
	APIToken              *string                `json:"apiToken,omitempty"`
	RefreshToken          *string                `json:"refreshToken,omitempty"`
	RequestTimeoutSeconds *int                   `json:"requestTimeoutSeconds,omitempty"`
	UIPrefs               *UIPrefsPatch          `json:"uiPrefs,omitempty"`
	TablePrefs            *map[string]TablePrefs `json:"tablePrefs,omitempty"`
}

type UIPrefs struct {
	ThemeMode             string `json:"themeMode"`
	PalettePreset         string `json:"palettePreset"`
	NavCollapsed          bool   `json:"navCollapsed"`
	PinActionsColumnRight bool   `json:"pinActionsColumnRight"`
	DataGridBorderRadius  int    `json:"dataGridBorderRadius"`
}

type UIPrefsPatch struct {
	ThemeMode             *string `json:"themeMode,omitempty"`
	PalettePreset         *string `json:"palettePreset,omitempty"`
	NavCollapsed          *bool   `json:"navCollapsed,omitempty"`
	PinActionsColumnRight *bool   `json:"pinActionsColumnRight,omitempty"`
	DataGridBorderRadius  *int    `json:"dataGridBorderRadius,omitempty"`
}

type TablePrefs struct {
	Version          int              `json:"version"`
	PageSize         int              `json:"pageSize"`
	Density          string           `json:"density"`
	ColumnVisibility map[string]bool  `json:"columnVisibility"`
	ColumnOrder      []string         `json:"columnOrder"`
	PinnedColumns    TablePinnedModel `json:"pinnedColumns"`
}

type TablePinnedModel struct {
	Left  []string `json:"left"`
	Right []string `json:"right"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) LoadSettings() (Settings, error) {
	return a.readSettings()
}

func (a *App) SaveSettings(patch SettingsPatch) (Settings, error) {
	current, err := a.readSettings()
	if err != nil {
		return Settings{}, err
	}

	next := current
	if patch.APIBaseURL != nil {
		next.APIBaseURL = strings.TrimSpace(*patch.APIBaseURL)
	}
	if patch.AuthMode != nil {
		next.AuthMode = *patch.AuthMode
	}
	if patch.APIToken != nil {
		token := strings.TrimSpace(*patch.APIToken)
		if token == "" {
			next.APIToken = nil
		} else {
			next.APIToken = &token
		}
	}
	if patch.RefreshToken != nil {
		token := strings.TrimSpace(*patch.RefreshToken)
		if token == "" {
			next.RefreshToken = nil
		} else {
			next.RefreshToken = &token
		}
	}
	if patch.RequestTimeoutSeconds != nil {
		next.RequestTimeoutSeconds = *patch.RequestTimeoutSeconds
	}
	if patch.UIPrefs != nil {
		if patch.UIPrefs.ThemeMode != nil {
			next.UIPrefs.ThemeMode = *patch.UIPrefs.ThemeMode
		}
		if patch.UIPrefs.PalettePreset != nil {
			next.UIPrefs.PalettePreset = strings.TrimSpace(*patch.UIPrefs.PalettePreset)
		}
		if patch.UIPrefs.NavCollapsed != nil {
			next.UIPrefs.NavCollapsed = *patch.UIPrefs.NavCollapsed
		}
		if patch.UIPrefs.PinActionsColumnRight != nil {
			next.UIPrefs.PinActionsColumnRight = *patch.UIPrefs.PinActionsColumnRight
		}
		if patch.UIPrefs.DataGridBorderRadius != nil {
			next.UIPrefs.DataGridBorderRadius = *patch.UIPrefs.DataGridBorderRadius
		}
	}
	if patch.TablePrefs != nil {
		next.TablePrefs = *patch.TablePrefs
	}

	next = normalizeSettings(next)
	if err := validateSettings(next); err != nil {
		return Settings{}, err
	}
	if err := a.writeSettings(next); err != nil {
		return Settings{}, err
	}
	return next, nil
}

func (a *App) ResetSettings() (Settings, error) {
	defaults := defaultSettings()
	if err := a.writeSettings(defaults); err != nil {
		return Settings{}, err
	}
	return defaults, nil
}

func defaultSettings() Settings {
	return Settings{
		APIBaseURL:            "",
		AuthMode:              "password",
		RequestTimeoutSeconds: 15,
		UIPrefs: UIPrefs{
			ThemeMode:             "system",
			PalettePreset:         "ocean",
			NavCollapsed:          false,
			PinActionsColumnRight: true,
			DataGridBorderRadius:  12,
		},
		TablePrefs: map[string]TablePrefs{},
	}
}

func normalizeSettings(in Settings) Settings {
	out := in
	out.APIBaseURL = strings.TrimSpace(out.APIBaseURL)
	if out.AuthMode == "" {
		out.AuthMode = "password"
	}
	if out.RequestTimeoutSeconds <= 0 {
		out.RequestTimeoutSeconds = 15
	}
	if out.UIPrefs.ThemeMode == "" {
		out.UIPrefs.ThemeMode = "system"
	}
	out.UIPrefs.PalettePreset = strings.TrimSpace(out.UIPrefs.PalettePreset)
	if out.UIPrefs.PalettePreset == "" {
		out.UIPrefs.PalettePreset = "ocean"
	}
	if out.UIPrefs.DataGridBorderRadius <= 0 {
		out.UIPrefs.DataGridBorderRadius = 12
		if !out.UIPrefs.PinActionsColumnRight {
			out.UIPrefs.PinActionsColumnRight = true
		}
	}
	if out.AuthMode != "api_token" {
		out.APIToken = nil
	} else if out.APIToken != nil {
		token := strings.TrimSpace(*out.APIToken)
		if token == "" {
			out.APIToken = nil
		} else {
			out.APIToken = &token
		}
	}
	if out.RefreshToken != nil {
		token := strings.TrimSpace(*out.RefreshToken)
		if token == "" {
			out.RefreshToken = nil
		} else {
			out.RefreshToken = &token
		}
	}
	if out.TablePrefs == nil {
		out.TablePrefs = map[string]TablePrefs{}
	}
	normalizedTablePrefs := make(map[string]TablePrefs, len(out.TablePrefs))
	for key, pref := range out.TablePrefs {
		storageKey := strings.TrimSpace(key)
		if storageKey == "" {
			continue
		}
		normalizedTablePrefs[storageKey] = normalizeTablePrefs(pref)
	}
	out.TablePrefs = normalizedTablePrefs
	return out
}

func normalizeTablePrefs(in TablePrefs) TablePrefs {
	out := in
	if out.Version <= 0 {
		out.Version = 1
	}
	if out.PageSize <= 0 {
		out.PageSize = 25
	}
	if out.Density != "compact" && out.Density != "comfortable" {
		out.Density = "standard"
	}
	if out.ColumnVisibility == nil {
		out.ColumnVisibility = map[string]bool{}
	}
	out.ColumnOrder = normalizeStringSlice(out.ColumnOrder)
	out.PinnedColumns.Left = normalizeStringSlice(out.PinnedColumns.Left)
	out.PinnedColumns.Right = normalizeStringSlice(out.PinnedColumns.Right)
	return out
}

func normalizeStringSlice(input []string) []string {
	if len(input) == 0 {
		return []string{}
	}
	result := make([]string, 0, len(input))
	for _, item := range input {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		result = append(result, value)
	}
	return result
}

func validateSettings(in Settings) error {
	if in.AuthMode != "password" && in.AuthMode != "api_token" {
		return errors.New("authMode must be password or api_token")
	}
	if in.RequestTimeoutSeconds < 1 || in.RequestTimeoutSeconds > 300 {
		return errors.New("requestTimeoutSeconds must be between 1 and 300")
	}
	if in.UIPrefs.ThemeMode != "light" && in.UIPrefs.ThemeMode != "dark" && in.UIPrefs.ThemeMode != "system" {
		return errors.New("uiPrefs.themeMode must be light, dark, or system")
	}
	if strings.TrimSpace(in.UIPrefs.PalettePreset) == "" {
		return errors.New("uiPrefs.palettePreset is required")
	}
	if in.UIPrefs.DataGridBorderRadius < 4 || in.UIPrefs.DataGridBorderRadius > 32 {
		return errors.New("uiPrefs.dataGridBorderRadius must be between 4 and 32")
	}
	return nil
}

func (a *App) readSettings() (Settings, error) {
	defaults := defaultSettings()
	path, err := settingsFilePath()
	if err != nil {
		return Settings{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaults, nil
		}
		return Settings{}, err
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return Settings{}, err
	}
	settings = normalizeSettings(settings)
	if err := validateSettings(settings); err != nil {
		return Settings{}, err
	}
	return settings, nil
}

func (a *App) writeSettings(in Settings) error {
	path, err := settingsFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	payload, err := json.MarshalIndent(in, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func settingsFilePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "basepro-desktop", "settings.json"), nil
}
