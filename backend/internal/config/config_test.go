package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHotReloadAtomicSwapAndValidation(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	initial := []byte(`
server:
  port: ":8080"
  shutdown_timeout_seconds: 10
database:
  dsn: "postgres://basepro:basepro@127.0.0.1:5432/basepro_dev?sslmode=disable"
  max_open_conns: 10
  max_idle_conns: 5
  auto_migrate: false
auth:
  access_token_ttl_seconds: 900
  refresh_token_ttl_seconds: 604800
  jwt_signing_key: "test-signing-key"
  password_hash_cost: 12
  api_token_enabled: true
  api_token_header_name: "X-API-Token"
  api_token_ttl_seconds: 2592000
  api_token_allow_bearer: false
`)
	if err := SafeWriteFile(cfgPath, initial, 0o600); err != nil {
		t.Fatalf("write initial config: %v", err)
	}

	_, err := Load(Options{ConfigFile: cfgPath, Watch: true})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if got := Get().Server.Port; got != ":8080" {
		t.Fatalf("expected initial port :8080, got %q", got)
	}

	updated := []byte(`
server:
  port: ":9090"
  shutdown_timeout_seconds: 10
database:
  dsn: "postgres://basepro:basepro@127.0.0.1:5432/basepro_dev?sslmode=disable"
  max_open_conns: 10
  max_idle_conns: 5
  auto_migrate: false
auth:
  access_token_ttl_seconds: 900
  refresh_token_ttl_seconds: 604800
  jwt_signing_key: "test-signing-key"
  password_hash_cost: 12
  api_token_enabled: true
  api_token_header_name: "X-API-Token"
  api_token_ttl_seconds: 2592000
  api_token_allow_bearer: false
`)
	if err := os.WriteFile(cfgPath, updated, 0o600); err != nil {
		t.Fatalf("write updated config: %v", err)
	}

	if !waitForConfig(t, func(cfg Config) bool { return cfg.Server.Port == ":9090" }) {
		t.Fatalf("expected config port to swap to :9090, got %q", Get().Server.Port)
	}

	invalid := []byte(`
server:
  port: ""
  shutdown_timeout_seconds: 0
database:
  dsn: ""
  max_open_conns: 0
  max_idle_conns: 0
auth:
  access_token_ttl_seconds: 0
  refresh_token_ttl_seconds: 0
  jwt_signing_key: ""
  password_hash_cost: 1
  api_token_enabled: true
  api_token_header_name: ""
  api_token_ttl_seconds: 0
  api_token_allow_bearer: false
`)
	if err := os.WriteFile(cfgPath, invalid, 0o600); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}

	time.Sleep(300 * time.Millisecond)
	if got := Get().Server.Port; got != ":9090" {
		t.Fatalf("invalid reload should keep previous config, got %q", got)
	}
}

func waitForConfig(t *testing.T, check func(Config) bool) bool {
	t.Helper()
	timeout := time.After(3 * time.Second)
	tick := time.NewTicker(25 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-timeout:
			return false
		case <-tick.C:
			if check(Get()) {
				return true
			}
		}
	}
}

func TestValidateLoggingConfig(t *testing.T) {
	cfg := defaultConfig()
	cfg.Database.DSN = "postgres://basepro:basepro@127.0.0.1:5432/basepro_dev?sslmode=disable"
	cfg.Auth.JWTSigningKey = "test-signing-key"

	cfg.Logging.Level = "debug"
	cfg.Logging.Format = "json"
	if err := validate(cfg); err != nil {
		t.Fatalf("expected valid logging config: %v", err)
	}

	cfg.Logging.Level = "verbose"
	if err := validate(cfg); err == nil {
		t.Fatal("expected invalid logging.level to fail validation")
	}

	cfg = defaultConfig()
	cfg.Database.DSN = "postgres://basepro:basepro@127.0.0.1:5432/basepro_dev?sslmode=disable"
	cfg.Auth.JWTSigningKey = "test-signing-key"
	cfg.Logging.Format = "pretty"
	if err := validate(cfg); err == nil {
		t.Fatal("expected invalid logging.format to fail validation")
	}
}

func TestValidateSecurityConfig(t *testing.T) {
	cfg := defaultConfig()
	cfg.Database.DSN = "postgres://basepro:basepro@127.0.0.1:5432/basepro_dev?sslmode=disable"
	cfg.Auth.JWTSigningKey = "test-signing-key"
	cfg.Security.CORS.Enabled = true
	cfg.Security.CORS.AllowedOrigins = []string{"http://localhost:5173"}
	cfg.Security.CORS.AllowCredentials = true
	if err := validate(cfg); err != nil {
		t.Fatalf("expected valid security config, got %v", err)
	}

	cfg.Security.CORS.AllowedOrigins = []string{"*"}
	if err := validate(cfg); err == nil {
		t.Fatal("expected wildcard origin with credentials to fail validation")
	}

	cfg = defaultConfig()
	cfg.Database.DSN = "postgres://basepro:basepro@127.0.0.1:5432/basepro_dev?sslmode=disable"
	cfg.Auth.JWTSigningKey = "test-signing-key"
	cfg.Security.RateLimit.Enabled = true
	cfg.Security.RateLimit.RequestsPerSecond = 0
	if err := validate(cfg); err == nil {
		t.Fatal("expected invalid rate limit config to fail validation")
	}
}

func TestLoadInvalidConfigFails(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "invalid-config.yaml")
	content := []byte(`
server:
  port: ""
database:
  dsn: ""
auth:
  jwt_signing_key: ""
`)
	if err := os.WriteFile(cfgPath, content, 0o600); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}

	if _, err := Load(Options{ConfigFile: cfgPath, Watch: false}); err == nil {
		t.Fatal("expected invalid config load to fail")
	}
}

func TestHotReloadRejectsInvalidCorsAndKeepsPrevious(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	initial := []byte(`
server:
  port: ":8181"
  shutdown_timeout_seconds: 10
database:
  dsn: "postgres://basepro:basepro@127.0.0.1:5432/basepro_dev?sslmode=disable"
  max_open_conns: 10
  max_idle_conns: 5
  auto_migrate: false
auth:
  access_token_ttl_seconds: 900
  refresh_token_ttl_seconds: 604800
  jwt_signing_key: "test-signing-key"
  password_hash_cost: 12
  api_token_enabled: true
  api_token_header_name: "X-API-Token"
  api_token_ttl_seconds: 2592000
  api_token_allow_bearer: false
security:
  cors:
    enabled: true
    allowed_origins:
      - "http://localhost:5173"
    allowed_methods:
      - "GET"
      - "POST"
      - "OPTIONS"
    allowed_headers:
      - "Authorization"
      - "Content-Type"
    allow_credentials: true
`)
	if err := SafeWriteFile(cfgPath, initial, 0o600); err != nil {
		t.Fatalf("write initial config: %v", err)
	}
	if _, err := Load(Options{ConfigFile: cfgPath, Watch: true}); err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := Get().Server.Port; got != ":8181" {
		t.Fatalf("expected initial port :8181, got %q", got)
	}

	invalidReload := []byte(`
server:
  port: ":9191"
  shutdown_timeout_seconds: 10
database:
  dsn: "postgres://basepro:basepro@127.0.0.1:5432/basepro_dev?sslmode=disable"
  max_open_conns: 10
  max_idle_conns: 5
  auto_migrate: false
auth:
  access_token_ttl_seconds: 900
  refresh_token_ttl_seconds: 604800
  jwt_signing_key: "test-signing-key"
  password_hash_cost: 12
  api_token_enabled: true
  api_token_header_name: "X-API-Token"
  api_token_ttl_seconds: 2592000
  api_token_allow_bearer: false
security:
  cors:
    enabled: true
    allowed_origins:
      - "*"
    allowed_methods:
      - "GET"
      - "POST"
      - "OPTIONS"
    allowed_headers:
      - "Authorization"
      - "Content-Type"
    allow_credentials: true
`)
	if err := os.WriteFile(cfgPath, invalidReload, 0o600); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}

	time.Sleep(300 * time.Millisecond)
	if got := Get().Server.Port; got != ":8181" {
		t.Fatalf("invalid reload should keep previous config, got %q", got)
	}
}

func TestSecureDefaults(t *testing.T) {
	cfg := defaultConfig()
	if cfg.Database.AutoMigrate {
		t.Fatal("expected database.auto_migrate default to false")
	}
	if cfg.Logging.Level != "info" {
		t.Fatalf("expected logging.level default info, got %q", cfg.Logging.Level)
	}
	if cfg.Security.RateLimit.Enabled {
		t.Fatal("expected security.rate_limit.enabled default false")
	}
	if cfg.Security.CORS.Enabled {
		t.Fatal("expected security.cors.enabled default false")
	}
}
