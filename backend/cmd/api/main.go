package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"basepro/backend/internal/audit"
	"basepro/backend/internal/auth"
	"basepro/backend/internal/config"
	"basepro/backend/internal/db"
	"basepro/backend/internal/logging"
	"basepro/backend/internal/middleware"
	"basepro/backend/internal/migrate"
	"basepro/backend/internal/rbac"
	"basepro/backend/internal/users"
)

var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

func main() {
	if err := run(); err != nil {
		logging.L().Error("api_start_failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	flags := newFlags()
	if err := flags.fs.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	_, err := config.Load(config.Options{
		ConfigFile: flags.configFile,
		Overrides:  flags.overrides(),
		Watch:      true,
	})
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	cfg := config.Get()
	logging.ApplyConfig(logging.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
	})
	config.RegisterOnChange(func(next config.Config) {
		logging.ApplyConfig(logging.Config{
			Level:  next.Logging.Level,
			Format: next.Logging.Format,
		})
	})

	database, err := db.Open(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := database.Close(); closeErr != nil {
			logging.L().Warn("database_close_failed", slog.String("error", closeErr.Error()))
		}
	}()

	startupMigrator := migrate.NewRunner()
	if err := startupMigrator.Run(ctx, migrate.Config{
		AutoMigrate: cfg.Database.AutoMigrate,
		LockTimeout: time.Duration(cfg.Database.AutoMigrateLockTimeoutSeconds) * time.Second,
	}, database, cfg.Database.DSN); err != nil {
		return fmt.Errorf("startup migrations: %w", err)
	}

	jwtManager := auth.NewJWTManager(cfg.Auth.JWTSigningKey, time.Duration(cfg.Auth.AccessTokenTTLSeconds)*time.Second)
	auditService := audit.NewService(audit.NewSQLRepository(database))
	rbacService := rbac.NewService(rbac.NewSQLRepository(database))
	authService := auth.NewService(
		auth.NewSQLRepository(database),
		auditService,
		jwtManager,
		rbacService,
		time.Duration(cfg.Auth.AccessTokenTTLSeconds)*time.Second,
		time.Duration(cfg.Auth.RefreshTokenTTLSeconds)*time.Second,
		time.Duration(cfg.Auth.APITokenTTLSeconds)*time.Second,
		cfg.Auth.JWTSigningKey,
		cfg.Auth.APITokenEnabled,
		cfg.Auth.PasswordHashCost,
	)
	usersService := users.NewService(users.NewSQLRepository(database), rbacService, auditService, cfg.Auth.PasswordHashCost)

	seedRBAC := cfg.Seed.EnableDevBootstrap || flags.seedDevAdmin
	if seedRBAC {
		if err := rbacService.EnsureBaseRBAC(ctx); err != nil {
			return fmt.Errorf("seed base rbac: %w", err)
		}
	}

	if seedRBAC {
		username := os.Getenv("BASEPRO_DEV_ADMIN_USERNAME")
		if username == "" {
			username = "admin"
		}
		password := os.Getenv("BASEPRO_DEV_ADMIN_PASSWORD")
		if password == "" {
			password = "admin123!"
		}
		user, seedErr := authService.SeedDevAdmin(ctx, username, password)
		if seedErr != nil {
			return fmt.Errorf("seed dev admin: %w", seedErr)
		}
		if err := rbacService.AssignRoleToUser(ctx, user.ID, "Admin"); err != nil {
			return fmt.Errorf("assign admin role: %w", err)
		}
		logging.L().Info("dev_admin_seeded", slog.String("username", user.Username), slog.Int64("user_id", user.ID))
	}

	srv := &http.Server{
		Addr: cfg.Server.Port,
		Handler: newRouter(AppDeps{
			DB:        database,
			Version:   Version,
			Commit:    Commit,
			BuildDate: BuildDate,
			CORSConfig: middleware.CORSConfig{
				Enabled:          cfg.Security.CORS.Enabled,
				AllowedOrigins:   cfg.Security.CORS.AllowedOrigins,
				AllowedMethods:   cfg.Security.CORS.AllowedMethods,
				AllowedHeaders:   cfg.Security.CORS.AllowedHeaders,
				AllowCredentials: cfg.Security.CORS.AllowCredentials,
			},
			RateLimitConfig: middleware.RateLimitConfig{
				Enabled:           cfg.Security.RateLimit.Enabled,
				RequestsPerSecond: cfg.Security.RateLimit.RequestsPerSecond,
				Burst:             cfg.Security.RateLimit.Burst,
			},
			AuthHandler:         auth.NewHandler(authService),
			AuthService:         authService,
			JWTManager:          jwtManager,
			RBACService:         rbacService,
			AuditHandler:        audit.NewHandler(auditService),
			UsersHandler:        users.NewHandler(usersService),
			APITokenHeaderName:  cfg.Auth.APITokenHeaderName,
			APITokenAllowBearer: cfg.Auth.APITokenAllowBearer,
		}),
	}

	shutdownTimeout := time.Duration(cfg.Server.ShutdownTimeoutSeconds) * time.Second
	return runServer(ctx, srv, shutdownTimeout)
}

type cliFlags struct {
	fs               *flag.FlagSet
	configFile       string
	loggingLevel     string
	loggingFormat    string
	serverPort       string
	shutdownTimeout  int
	databaseDSN      string
	maxOpenConns     int
	maxIdleConns     int
	autoMigrate      bool
	autoMigrateLock  int
	authAccessTTL    int
	authRefreshTTL   int
	authSigningKey   string
	passwordHashCost int
	apiTokenEnabled  bool
	apiTokenHeader   string
	apiTokenTTL      int
	apiTokenBearer   bool
	rateLimitEnabled bool
	rateLimitRPS     float64
	rateLimitBurst   int
	corsEnabled      bool
	corsCredentials  bool
	seedDevAdmin     bool
}

func newFlags() *cliFlags {
	f := &cliFlags{fs: flag.NewFlagSet(os.Args[0], flag.ContinueOnError)}
	f.fs.StringVar(&f.configFile, "config", "", "path to config file")
	f.fs.StringVar(&f.loggingLevel, "logging-level", "", "logging level (debug|info|warn|error)")
	f.fs.StringVar(&f.loggingFormat, "logging-format", "", "logging format (json|console)")
	f.fs.StringVar(&f.serverPort, "server-port", "", "server listen address")
	f.fs.IntVar(&f.shutdownTimeout, "shutdown-timeout", 0, "shutdown timeout in seconds")
	f.fs.StringVar(&f.databaseDSN, "database-dsn", "", "database DSN")
	f.fs.IntVar(&f.maxOpenConns, "database-max-open-conns", 0, "max open DB connections")
	f.fs.IntVar(&f.maxIdleConns, "database-max-idle-conns", 0, "max idle DB connections")
	f.fs.BoolVar(&f.autoMigrate, "database-auto-migrate", false, "auto-run migrations on startup")
	f.fs.IntVar(&f.autoMigrateLock, "database-auto-migrate-lock-timeout", 0, "migration advisory lock timeout in seconds")
	f.fs.IntVar(&f.authAccessTTL, "auth-access-ttl", 0, "access token TTL in seconds")
	f.fs.IntVar(&f.authRefreshTTL, "auth-refresh-ttl", 0, "refresh token TTL in seconds")
	f.fs.StringVar(&f.authSigningKey, "auth-jwt-signing-key", "", "JWT signing key")
	f.fs.IntVar(&f.passwordHashCost, "auth-password-hash-cost", 0, "bcrypt password hash cost")
	f.fs.BoolVar(&f.apiTokenEnabled, "auth-api-token-enabled", false, "enable API token auth")
	f.fs.StringVar(&f.apiTokenHeader, "auth-api-token-header", "", "API token header name")
	f.fs.IntVar(&f.apiTokenTTL, "auth-api-token-ttl", 0, "default API token TTL in seconds")
	f.fs.BoolVar(&f.apiTokenBearer, "auth-api-token-allow-bearer", false, "allow Authorization bearer for API token")
	f.fs.BoolVar(&f.rateLimitEnabled, "security-rate-limit-enabled", false, "enable auth endpoint rate limiting")
	f.fs.Float64Var(&f.rateLimitRPS, "security-rate-limit-rps", 0, "rate limit requests per second")
	f.fs.IntVar(&f.rateLimitBurst, "security-rate-limit-burst", 0, "rate limit burst size")
	f.fs.BoolVar(&f.corsEnabled, "security-cors-enabled", false, "enable CORS middleware")
	f.fs.BoolVar(&f.corsCredentials, "security-cors-allow-credentials", false, "allow CORS credentials")
	f.fs.BoolVar(&f.seedDevAdmin, "seed-dev-admin", false, "seed a dev admin user")
	return f
}

func (f *cliFlags) overrides() map[string]any {
	overrides := make(map[string]any)
	f.fs.Visit(func(fl *flag.Flag) {
		switch fl.Name {
		case "logging-level":
			overrides["logging.level"] = f.loggingLevel
		case "logging-format":
			overrides["logging.format"] = f.loggingFormat
		case "server-port":
			overrides["server.port"] = f.serverPort
		case "shutdown-timeout":
			overrides["server.shutdown_timeout_seconds"] = f.shutdownTimeout
		case "database-dsn":
			overrides["database.dsn"] = f.databaseDSN
		case "database-max-open-conns":
			overrides["database.max_open_conns"] = f.maxOpenConns
		case "database-max-idle-conns":
			overrides["database.max_idle_conns"] = f.maxIdleConns
		case "database-auto-migrate":
			overrides["database.auto_migrate"] = f.autoMigrate
		case "database-auto-migrate-lock-timeout":
			overrides["database.auto_migrate_lock_timeout_seconds"] = f.autoMigrateLock
		case "auth-access-ttl":
			overrides["auth.access_token_ttl_seconds"] = f.authAccessTTL
		case "auth-refresh-ttl":
			overrides["auth.refresh_token_ttl_seconds"] = f.authRefreshTTL
		case "auth-jwt-signing-key":
			overrides["auth.jwt_signing_key"] = f.authSigningKey
		case "auth-password-hash-cost":
			overrides["auth.password_hash_cost"] = f.passwordHashCost
		case "auth-api-token-enabled":
			overrides["auth.api_token_enabled"] = f.apiTokenEnabled
		case "auth-api-token-header":
			overrides["auth.api_token_header_name"] = f.apiTokenHeader
		case "auth-api-token-ttl":
			overrides["auth.api_token_ttl_seconds"] = f.apiTokenTTL
		case "auth-api-token-allow-bearer":
			overrides["auth.api_token_allow_bearer"] = f.apiTokenBearer
		case "security-rate-limit-enabled":
			overrides["security.rate_limit.enabled"] = f.rateLimitEnabled
		case "security-rate-limit-rps":
			overrides["security.rate_limit.requests_per_second"] = f.rateLimitRPS
		case "security-rate-limit-burst":
			overrides["security.rate_limit.burst"] = f.rateLimitBurst
		case "security-cors-enabled":
			overrides["security.cors.enabled"] = f.corsEnabled
		case "security-cors-allow-credentials":
			overrides["security.cors.allow_credentials"] = f.corsCredentials
		}
	})
	return overrides
}
