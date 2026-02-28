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
	APIBaseURL            string  `json:"apiBaseUrl"`
	AuthMode              string  `json:"authMode"`
	APIToken              *string `json:"apiToken,omitempty"`
	RequestTimeoutSeconds int     `json:"requestTimeoutSeconds"`
}

type SettingsPatch struct {
	APIBaseURL            *string `json:"apiBaseUrl,omitempty"`
	AuthMode              *string `json:"authMode,omitempty"`
	APIToken              *string `json:"apiToken,omitempty"`
	RequestTimeoutSeconds *int    `json:"requestTimeoutSeconds,omitempty"`
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
	if patch.RequestTimeoutSeconds != nil {
		next.RequestTimeoutSeconds = *patch.RequestTimeoutSeconds
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
	return out
}

func validateSettings(in Settings) error {
	if in.AuthMode != "password" && in.AuthMode != "api_token" {
		return errors.New("authMode must be password or api_token")
	}
	if in.RequestTimeoutSeconds < 1 || in.RequestTimeoutSeconds > 300 {
		return errors.New("requestTimeoutSeconds must be between 1 and 300")
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
