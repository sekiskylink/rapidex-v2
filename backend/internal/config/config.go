package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Config is the typed runtime configuration snapshot used across the backend.
type Config struct {
	Server struct {
		Port                   string `mapstructure:"port"`
		ShutdownTimeoutSeconds int    `mapstructure:"shutdown_timeout_seconds"`
	} `mapstructure:"server"`
	Database struct {
		DSN          string `mapstructure:"dsn"`
		MaxOpenConns int    `mapstructure:"max_open_conns"`
		MaxIdleConns int    `mapstructure:"max_idle_conns"`
		AutoMigrate  bool   `mapstructure:"auto_migrate"`
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
	Seed struct {
		EnableDevBootstrap bool `mapstructure:"enable_dev_bootstrap"`
	} `mapstructure:"seed"`
}

type Options struct {
	ConfigFile string
	Overrides  map[string]any
	Watch      bool
}

var activeConfig atomic.Value

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

	if opts.Watch {
		v.OnConfigChange(func(event fsnotify.Event) {
			newCfg, decodeErr := decodeAndValidate(v)
			if decodeErr != nil {
				log.Printf("config reload rejected (%s): %v", event.Name, decodeErr)
				return
			}

			activeConfig.Store(newCfg)
			log.Printf("config reload applied (%s)", event.Name)
		})
		v.WatchConfig()
	}

	return v, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.port", ":8080")
	v.SetDefault("server.shutdown_timeout_seconds", 10)
	v.SetDefault("database.max_open_conns", 10)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.auto_migrate", true)
	v.SetDefault("auth.access_token_ttl_seconds", 900)
	v.SetDefault("auth.refresh_token_ttl_seconds", 604800)
	v.SetDefault("auth.password_hash_cost", 12)
	v.SetDefault("auth.api_token_enabled", true)
	v.SetDefault("auth.api_token_header_name", "X-API-Token")
	v.SetDefault("auth.api_token_ttl_seconds", 2592000)
	v.SetDefault("auth.api_token_allow_bearer", false)
	v.SetDefault("seed.enable_dev_bootstrap", false)
}

func defaultConfig() Config {
	cfg := Config{}
	cfg.Server.Port = ":8080"
	cfg.Server.ShutdownTimeoutSeconds = 10
	cfg.Database.MaxOpenConns = 10
	cfg.Database.MaxIdleConns = 5
	cfg.Database.AutoMigrate = true
	cfg.Auth.AccessTokenTTLSeconds = 900
	cfg.Auth.RefreshTokenTTLSeconds = 604800
	cfg.Auth.PasswordHashCost = 12
	cfg.Auth.APITokenEnabled = true
	cfg.Auth.APITokenHeaderName = "X-API-Token"
	cfg.Auth.APITokenTTLSeconds = 2592000
	cfg.Auth.APITokenAllowBearer = false
	cfg.Seed.EnableDevBootstrap = false
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
	if cfg.Server.Port == "" {
		return errors.New("server.port must not be empty")
	}
	if cfg.Server.ShutdownTimeoutSeconds <= 0 {
		return errors.New("server.shutdown_timeout_seconds must be > 0")
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
	return nil
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
