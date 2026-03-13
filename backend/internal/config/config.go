package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"basepro/backend/internal/logging"
	"basepro/backend/internal/moduleenablement"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Config is the typed runtime configuration snapshot used across the backend.
type Config struct {
	Logging struct {
		Level  string `mapstructure:"level"`
		Format string `mapstructure:"format"`
	} `mapstructure:"logging"`
	Server struct {
		Port                   string   `mapstructure:"port"`
		ShutdownTimeoutSeconds int      `mapstructure:"shutdown_timeout_seconds"`
		CORSAllowedOrigins     []string `mapstructure:"cors_allowed_origins"`
	} `mapstructure:"server"`
	Database struct {
		DSN                           string `mapstructure:"dsn"`
		MaxOpenConns                  int    `mapstructure:"max_open_conns"`
		MaxIdleConns                  int    `mapstructure:"max_idle_conns"`
		AutoMigrate                   bool   `mapstructure:"auto_migrate"`
		AutoMigrateLockTimeoutSeconds int    `mapstructure:"auto_migrate_lock_timeout_seconds"`
	} `mapstructure:"database"`
	Auth struct {
		AccessTokenTTLSeconds  int    `mapstructure:"access_token_ttl_seconds"`
		RefreshTokenTTLSeconds int    `mapstructure:"refresh_token_ttl_seconds"`
		JWTSigningKey          string `mapstructure:"jwt_signing_key"`
		PasswordHashCost       int    `mapstructure:"password_hash_cost"`
		APITokenEnabled        bool   `mapstructure:"api_token_enabled"`
		APITokenHeaderName     string `mapstructure:"api_token_header_name"`
		APITokenTTLSeconds     int    `mapstructure:"api_token_ttl_seconds"`
		APITokenAllowBearer    bool   `mapstructure:"api_token_allow_bearer"`
	} `mapstructure:"auth"`
	Security struct {
		RateLimit struct {
			Enabled           bool    `mapstructure:"enabled"`
			RequestsPerSecond float64 `mapstructure:"requests_per_second"`
			Burst             int     `mapstructure:"burst"`
		} `mapstructure:"rate_limit"`
		CORS struct {
			Enabled          bool     `mapstructure:"enabled"`
			AllowedOrigins   []string `mapstructure:"allowed_origins"`
			AllowedMethods   []string `mapstructure:"allowed_methods"`
			AllowedHeaders   []string `mapstructure:"allowed_headers"`
			AllowCredentials bool     `mapstructure:"allow_credentials"`
		} `mapstructure:"cors"`
	} `mapstructure:"security"`
	Seed struct {
		EnableDevBootstrap bool `mapstructure:"enable_dev_bootstrap"`
	} `mapstructure:"seed"`
	Modules struct {
		Flags map[string]bool `mapstructure:"flags"`
	} `mapstructure:"modules"`
	Sukumad struct {
		SubmissionWindow struct {
			Default struct {
				StartHour int `mapstructure:"start_hour"`
				EndHour   int `mapstructure:"end_hour"`
			} `mapstructure:"default"`
			Destinations map[string]struct {
				StartHour int `mapstructure:"start_hour"`
				EndHour   int `mapstructure:"end_hour"`
			} `mapstructure:"destinations"`
		} `mapstructure:"submission_window"`
		Retry struct {
			Default struct {
				MaxRetries int `mapstructure:"max_retries"`
			} `mapstructure:"default"`
			Destinations map[string]struct {
				MaxRetries int `mapstructure:"max_retries"`
			} `mapstructure:"destinations"`
		} `mapstructure:"retry"`
		ResponseFilter struct {
			Default struct {
				AllowedContentTypes []string `mapstructure:"allowed_content_types"`
				AllowUnknown        bool     `mapstructure:"allow_unknown"`
			} `mapstructure:"default"`
			Destinations map[string]struct {
				AllowedContentTypes []string `mapstructure:"allowed_content_types"`
				AllowUnknown        bool     `mapstructure:"allow_unknown"`
			} `mapstructure:"destinations"`
		} `mapstructure:"response_filter"`
		Retention struct {
			Enabled         bool `mapstructure:"enabled"`
			DryRun          bool `mapstructure:"dry_run"`
			TerminalAgeDays int `mapstructure:"terminal_age_days"`
			BatchSize       int `mapstructure:"batch_size"`
		} `mapstructure:"retention"`
		RateLimit struct {
			Default struct {
				RequestsPerSecond float64 `mapstructure:"requests_per_second"`
				Burst             int     `mapstructure:"burst"`
			} `mapstructure:"default"`
			Destinations map[string]struct {
				RequestsPerSecond float64 `mapstructure:"requests_per_second"`
				Burst             int     `mapstructure:"burst"`
			} `mapstructure:"destinations"`
		} `mapstructure:"rate_limit"`
	} `mapstructure:"sukumad"`
}

type Options struct {
	ConfigFile string
	Overrides  map[string]any
	Watch      bool
}

var activeConfig atomic.Value
var configChangeCallbacks []func(Config)
var callbackMu sync.Mutex

func init() {
	activeConfig.Store(defaultConfig())
}

// Get returns the latest validated config snapshot.
func Get() Config {
	cfg, ok := activeConfig.Load().(Config)
	if !ok {
		return defaultConfig()
	}
	return cfg
}

// Load reads config from file + env + overrides and sets up optional hot reload.
func Load(opts Options) (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./config")
	v.AddConfigPath(".")

	if opts.ConfigFile != "" {
		v.SetConfigFile(opts.ConfigFile)
	}

	setDefaults(v)
	v.SetEnvPrefix("BASEPRO")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	for key, value := range opts.Overrides {
		v.Set(key, value)
	}

	cfg, err := decodeAndValidate(v)
	if err != nil {
		return nil, err
	}
	activeConfig.Store(cfg)
	notifyConfigChange(cfg)

	if opts.Watch {
		v.OnConfigChange(func(event fsnotify.Event) {
			newCfg, decodeErr := decodeAndValidate(v)
			if decodeErr != nil {
				logging.L().Warn("config_reload_rejected", slog.String("file", event.Name), slog.String("error", decodeErr.Error()))
				return
			}

			activeConfig.Store(newCfg)
			notifyConfigChange(newCfg)
			logging.L().Info("config_reload_applied", slog.String("file", event.Name))
		})
		v.WatchConfig()
	}

	return v, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "console")
	v.SetDefault("server.port", ":8080")
	v.SetDefault("server.shutdown_timeout_seconds", 10)
	v.SetDefault("server.cors_allowed_origins", []string{
		"http://localhost:5173",
		"http://127.0.0.1:5173",
		"wails://wails.localhost",
		"wails://wails.localhost:*",
	})
	v.SetDefault("database.max_open_conns", 10)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.auto_migrate", false)
	v.SetDefault("database.auto_migrate_lock_timeout_seconds", 30)
	v.SetDefault("auth.access_token_ttl_seconds", 900)
	v.SetDefault("auth.refresh_token_ttl_seconds", 604800)
	v.SetDefault("auth.password_hash_cost", 12)
	v.SetDefault("auth.api_token_enabled", true)
	v.SetDefault("auth.api_token_header_name", "X-API-Token")
	v.SetDefault("auth.api_token_ttl_seconds", 2592000)
	v.SetDefault("auth.api_token_allow_bearer", false)
	v.SetDefault("security.rate_limit.enabled", false)
	v.SetDefault("security.rate_limit.requests_per_second", 5.0)
	v.SetDefault("security.rate_limit.burst", 10)
	v.SetDefault("security.cors.enabled", false)
	v.SetDefault("security.cors.allowed_origins", []string{})
	v.SetDefault("security.cors.allowed_methods", []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"})
	v.SetDefault("security.cors.allowed_headers", []string{"Authorization", "Content-Type", "X-API-Token", "X-Request-Id"})
	v.SetDefault("security.cors.allow_credentials", false)
	v.SetDefault("seed.enable_dev_bootstrap", false)
	v.SetDefault("modules.flags", map[string]bool{})
	v.SetDefault("sukumad.submission_window.default.start_hour", 0)
	v.SetDefault("sukumad.submission_window.default.end_hour", 0)
	v.SetDefault("sukumad.submission_window.destinations", map[string]any{})
	v.SetDefault("sukumad.retry.default.max_retries", 2)
	v.SetDefault("sukumad.retry.destinations", map[string]any{})
	v.SetDefault("sukumad.response_filter.default.allowed_content_types", []string{"application/json", "application/*+json", "application/xml", "text/xml"})
	v.SetDefault("sukumad.response_filter.default.allow_unknown", false)
	v.SetDefault("sukumad.response_filter.destinations", map[string]any{})
	v.SetDefault("sukumad.retention.enabled", false)
	v.SetDefault("sukumad.retention.dry_run", true)
	v.SetDefault("sukumad.retention.terminal_age_days", 30)
	v.SetDefault("sukumad.retention.batch_size", 100)
	v.SetDefault("sukumad.rate_limit.default.requests_per_second", 2.0)
	v.SetDefault("sukumad.rate_limit.default.burst", 2)
	v.SetDefault("sukumad.rate_limit.destinations", map[string]any{})
}

func defaultConfig() Config {
	cfg := Config{}
	cfg.Logging.Level = "info"
	cfg.Logging.Format = "console"
	cfg.Server.Port = ":8080"
	cfg.Server.ShutdownTimeoutSeconds = 10
	cfg.Server.CORSAllowedOrigins = []string{
		"http://localhost:5173",
		"http://127.0.0.1:5173",
		"wails://wails.localhost",
		"wails://wails.localhost:*",
	}
	cfg.Database.MaxOpenConns = 10
	cfg.Database.MaxIdleConns = 5
	cfg.Database.AutoMigrate = false
	cfg.Database.AutoMigrateLockTimeoutSeconds = 30
	cfg.Auth.AccessTokenTTLSeconds = 900
	cfg.Auth.RefreshTokenTTLSeconds = 604800
	cfg.Auth.PasswordHashCost = 12
	cfg.Auth.APITokenEnabled = true
	cfg.Auth.APITokenHeaderName = "X-API-Token"
	cfg.Auth.APITokenTTLSeconds = 2592000
	cfg.Auth.APITokenAllowBearer = false
	cfg.Security.RateLimit.Enabled = false
	cfg.Security.RateLimit.RequestsPerSecond = 5
	cfg.Security.RateLimit.Burst = 10
	cfg.Security.CORS.Enabled = false
	cfg.Security.CORS.AllowedOrigins = []string{}
	cfg.Security.CORS.AllowedMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	cfg.Security.CORS.AllowedHeaders = []string{"Authorization", "Content-Type", "X-API-Token", "X-Request-Id"}
	cfg.Security.CORS.AllowCredentials = false
	cfg.Seed.EnableDevBootstrap = false
	cfg.Modules.Flags = map[string]bool{}
	cfg.Sukumad.SubmissionWindow.Default.StartHour = 0
	cfg.Sukumad.SubmissionWindow.Default.EndHour = 0
	cfg.Sukumad.SubmissionWindow.Destinations = map[string]struct {
		StartHour int `mapstructure:"start_hour"`
		EndHour   int `mapstructure:"end_hour"`
	}{}
	cfg.Sukumad.Retry.Default.MaxRetries = 2
	cfg.Sukumad.Retry.Destinations = map[string]struct {
		MaxRetries int `mapstructure:"max_retries"`
	}{}
	cfg.Sukumad.ResponseFilter.Default.AllowedContentTypes = []string{"application/json", "application/*+json", "application/xml", "text/xml"}
	cfg.Sukumad.ResponseFilter.Default.AllowUnknown = false
	cfg.Sukumad.ResponseFilter.Destinations = map[string]struct {
		AllowedContentTypes []string `mapstructure:"allowed_content_types"`
		AllowUnknown        bool     `mapstructure:"allow_unknown"`
	}{}
	cfg.Sukumad.Retention.Enabled = false
	cfg.Sukumad.Retention.DryRun = true
	cfg.Sukumad.Retention.TerminalAgeDays = 30
	cfg.Sukumad.Retention.BatchSize = 100
	cfg.Sukumad.RateLimit.Default.RequestsPerSecond = 2
	cfg.Sukumad.RateLimit.Default.Burst = 2
	cfg.Sukumad.RateLimit.Destinations = map[string]struct {
		RequestsPerSecond float64 `mapstructure:"requests_per_second"`
		Burst             int     `mapstructure:"burst"`
	}{}
	return cfg
}

func decodeAndValidate(v *viper.Viper) (Config, error) {
	cfg := defaultConfig()
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	if err := validate(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func validate(cfg Config) error {
	switch strings.ToLower(strings.TrimSpace(cfg.Logging.Level)) {
	case "debug", "info", "warn", "error":
	default:
		return errors.New("logging.level must be one of: debug, info, warn, error")
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Logging.Format)) {
	case "json", "console":
	default:
		return errors.New("logging.format must be one of: json, console")
	}
	if cfg.Server.Port == "" {
		return errors.New("server.port must not be empty")
	}
	if cfg.Server.ShutdownTimeoutSeconds <= 0 {
		return errors.New("server.shutdown_timeout_seconds must be > 0")
	}
	for _, origin := range cfg.Server.CORSAllowedOrigins {
		if strings.TrimSpace(origin) == "" {
			return errors.New("server.cors_allowed_origins must not contain empty values")
		}
	}
	if cfg.Database.DSN == "" {
		return errors.New("database.dsn must not be empty")
	}
	if cfg.Database.MaxOpenConns <= 0 {
		return errors.New("database.max_open_conns must be > 0")
	}
	if cfg.Database.MaxIdleConns <= 0 {
		return errors.New("database.max_idle_conns must be > 0")
	}
	if cfg.Database.AutoMigrateLockTimeoutSeconds <= 0 {
		return errors.New("database.auto_migrate_lock_timeout_seconds must be > 0")
	}
	if cfg.Auth.AccessTokenTTLSeconds <= 0 {
		return errors.New("auth.access_token_ttl_seconds must be > 0")
	}
	if cfg.Auth.RefreshTokenTTLSeconds <= 0 {
		return errors.New("auth.refresh_token_ttl_seconds must be > 0")
	}
	if cfg.Auth.JWTSigningKey == "" {
		return errors.New("auth.jwt_signing_key must not be empty")
	}
	if cfg.Auth.PasswordHashCost < 4 || cfg.Auth.PasswordHashCost > 31 {
		return errors.New("auth.password_hash_cost must be between 4 and 31")
	}
	if cfg.Auth.APITokenHeaderName == "" {
		return errors.New("auth.api_token_header_name must not be empty")
	}
	if cfg.Auth.APITokenTTLSeconds <= 0 {
		return errors.New("auth.api_token_ttl_seconds must be > 0")
	}
	if cfg.Security.RateLimit.Enabled {
		if cfg.Security.RateLimit.RequestsPerSecond <= 0 {
			return errors.New("security.rate_limit.requests_per_second must be > 0 when rate limiting is enabled")
		}
		if cfg.Security.RateLimit.Burst <= 0 {
			return errors.New("security.rate_limit.burst must be > 0 when rate limiting is enabled")
		}
	}
	if cfg.Security.CORS.Enabled {
		if len(cfg.Security.CORS.AllowedOrigins) == 0 {
			return errors.New("security.cors.allowed_origins must not be empty when CORS is enabled")
		}
		if len(cfg.Security.CORS.AllowedMethods) == 0 {
			return errors.New("security.cors.allowed_methods must not be empty when CORS is enabled")
		}
		if len(cfg.Security.CORS.AllowedHeaders) == 0 {
			return errors.New("security.cors.allowed_headers must not be empty when CORS is enabled")
		}
		for _, origin := range cfg.Security.CORS.AllowedOrigins {
			trimmed := strings.TrimSpace(origin)
			if trimmed == "" {
				return errors.New("security.cors.allowed_origins must not contain empty values")
			}
			if cfg.Security.CORS.AllowCredentials && strings.Contains(trimmed, "*") {
				return errors.New("security.cors.allowed_origins must not contain wildcards when allow_credentials is true")
			}
		}
	}
	if cfg.Sukumad.RateLimit.Default.RequestsPerSecond <= 0 {
		return errors.New("sukumad.rate_limit.default.requests_per_second must be > 0")
	}
	if cfg.Sukumad.RateLimit.Default.Burst <= 0 {
		return errors.New("sukumad.rate_limit.default.burst must be > 0")
	}
	if err := validateWindow(cfg.Sukumad.SubmissionWindow.Default.StartHour, cfg.Sukumad.SubmissionWindow.Default.EndHour, "sukumad.submission_window.default"); err != nil {
		return err
	}
	for key, destination := range cfg.Sukumad.SubmissionWindow.Destinations {
		if err := validateWindow(destination.StartHour, destination.EndHour, "sukumad.submission_window.destinations."+key); err != nil {
			return err
		}
	}
	if cfg.Sukumad.Retry.Default.MaxRetries < 0 {
		return errors.New("sukumad.retry.default.max_retries must be >= 0")
	}
	for key, destination := range cfg.Sukumad.Retry.Destinations {
		if destination.MaxRetries < 0 {
			return fmt.Errorf("sukumad.retry.destinations.%s.max_retries must be >= 0", key)
		}
	}
	if cfg.Sukumad.Retention.TerminalAgeDays <= 0 {
		return errors.New("sukumad.retention.terminal_age_days must be > 0")
	}
	if cfg.Sukumad.Retention.BatchSize <= 0 {
		return errors.New("sukumad.retention.batch_size must be > 0")
	}
	for key, destination := range cfg.Sukumad.RateLimit.Destinations {
		if strings.TrimSpace(key) == "" {
			return errors.New("sukumad.rate_limit.destinations must not contain empty keys")
		}
		if destination.RequestsPerSecond <= 0 {
			return fmt.Errorf("sukumad.rate_limit.destinations.%s.requests_per_second must be > 0", key)
		}
		if destination.Burst <= 0 {
			return fmt.Errorf("sukumad.rate_limit.destinations.%s.burst must be > 0", key)
		}
	}
	if err := moduleenablement.ValidateOverrides(cfg.Modules.Flags); err != nil {
		return err
	}
	return nil
}

func validateWindow(start int, end int, prefix string) error {
	if start == 0 && end == 0 {
		return nil
	}
	if start < 0 || start > 23 {
		return fmt.Errorf("%s.start_hour must be between 0 and 23", prefix)
	}
	if end < 0 || end > 23 {
		return fmt.Errorf("%s.end_hour must be between 0 and 23", prefix)
	}
	if start == end {
		return fmt.Errorf("%s start_hour and end_hour must differ", prefix)
	}
	return nil
}

// RegisterOnChange registers a callback that receives each validated config
// snapshot, including the initial load.
func RegisterOnChange(callback func(Config)) {
	if callback == nil {
		return
	}
	callbackMu.Lock()
	defer callbackMu.Unlock()
	configChangeCallbacks = append(configChangeCallbacks, callback)
}

func notifyConfigChange(cfg Config) {
	callbackMu.Lock()
	callbacks := append([]func(Config){}, configChangeCallbacks...)
	callbackMu.Unlock()
	for _, callback := range callbacks {
		callback(cfg)
	}
}

// SafeWriteFile writes to a temp file and atomically renames it over the target.
func SafeWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp config file: %w", err)
	}

	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp config file: %w", err)
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temp config file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp config file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename temp config file: %w", err)
	}

	return nil
}
