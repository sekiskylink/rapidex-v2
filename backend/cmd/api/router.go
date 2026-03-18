package main

import (
	"context"
	"net/http"
	"time"

	"basepro/backend/internal/audit"
	"basepro/backend/internal/auth"
	"basepro/backend/internal/bootstrap"
	"basepro/backend/internal/middleware"
	"basepro/backend/internal/moduleenablement"
	"basepro/backend/internal/openapi"
	"basepro/backend/internal/rbac"
	"basepro/backend/internal/settings"
	"basepro/backend/internal/sukumad"
	asyncjobs "basepro/backend/internal/sukumad/async"
	"basepro/backend/internal/sukumad/dashboard"
	"basepro/backend/internal/sukumad/delivery"
	"basepro/backend/internal/sukumad/observability"
	requests "basepro/backend/internal/sukumad/request"
	"basepro/backend/internal/sukumad/server"
	"basepro/backend/internal/users"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type AppDeps struct {
	DB                   *sqlx.DB
	Version              string
	Commit               string
	BuildDate            string
	CORSConfig           middleware.CORSConfig
	RateLimitConfig      middleware.RateLimitConfig
	AuthHandler          *auth.Handler
	AuthService          *auth.Service
	JWTManager           *auth.JWTManager
	RBACService          *rbac.Service
	RBACAdminHandler     *rbac.AdminHandler
	AuditHandler         *audit.Handler
	UsersHandler         *users.Handler
	SettingsHandler      *settings.Handler
	BootstrapHandler     *bootstrap.Handler
	ModuleFlagsHandler   *moduleenablement.Handler
	ModuleFlagsProvider  func() map[string]bool
	APITokenHeaderName   string
	APITokenAllowBearer  bool
	ServerHandler        *server.Handler
	RequestHandler       *requests.Handler
	DeliveryHandler      *delivery.Handler
	AsyncHandler         *asyncjobs.Handler
	ObservabilityHandler *observability.Handler
	DashboardHandler     *dashboard.Handler
}

func newRouter(deps AppDeps) *gin.Engine {
	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(middleware.AccessLog())
	r.Use(gin.Recovery())
	openapi.RegisterRoutes(r)
	if deps.CORSConfig.Enabled {
		r.Use(middleware.CORS(deps.CORSConfig))
	}
	if deps.AuthService != nil {
		r.Use(middleware.APITokenAuth(deps.AuthService, deps.APITokenHeaderName, deps.APITokenAllowBearer))
	}

	api := r.Group("/api/v1")
	api.GET("/health", func(c *gin.Context) {
		statusCode := http.StatusOK
		payload := gin.H{
			"status":  "ok",
			"version": deps.Version,
			"db":      "up",
		}

		healthCtx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if deps.DB == nil || deps.DB.PingContext(healthCtx) != nil {
			statusCode = http.StatusServiceUnavailable
			payload["status"] = "degraded"
			payload["db"] = "down"
		}

		c.JSON(statusCode, payload)
	})

	api.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"version":   deps.Version,
			"commit":    deps.Commit,
			"buildDate": deps.BuildDate,
		})
	})

	if deps.ModuleFlagsHandler != nil {
		api.GET("/modules/effective", deps.ModuleFlagsHandler.GetEffective)
	}

	if deps.BootstrapHandler != nil {
		bootstrapHandlers := []gin.HandlerFunc{}
		if deps.JWTManager != nil {
			bootstrapHandlers = append(bootstrapHandlers, middleware.ResolveJWTPrincipal(deps.JWTManager))
		}
		bootstrapHandlers = append(bootstrapHandlers, deps.BootstrapHandler.Get)
		api.GET("/bootstrap", bootstrapHandlers...)
	}

	if deps.AuthHandler != nil && deps.JWTManager != nil {
		authRateLimiter := middleware.NewAuthRateLimiter(deps.RateLimitConfig)
		authGroup := api.Group("/auth")
		authGroup.POST("/login", authRateLimiter.Middleware(), deps.AuthHandler.Login)
		authGroup.POST("/refresh", authRateLimiter.Middleware(), deps.AuthHandler.Refresh)
		authGroup.POST("/forgot-password", authRateLimiter.Middleware(), deps.AuthHandler.ForgotPassword)
		authGroup.POST("/reset-password", authRateLimiter.Middleware(), deps.AuthHandler.ResetPassword)
		authGroup.POST("/logout", deps.AuthHandler.Logout)
		authGroup.GET("/me", middleware.JWTAuth(deps.JWTManager), middleware.RequireJWTUser(), deps.AuthHandler.Me)

		admin := api.Group("/admin/api-tokens")
		admin.Use(middleware.ResolveJWTPrincipal(deps.JWTManager), middleware.RequireAuth())
		admin.GET("", middleware.RequirePermission(deps.RBACService, rbac.PermissionAPITokensRead), deps.AuthHandler.ListAPITokens)
		admin.POST("", middleware.RequirePermission(deps.RBACService, rbac.PermissionAPITokensWrite), deps.AuthHandler.CreateAPIToken)
		admin.POST("/:id/revoke", middleware.RequirePermission(deps.RBACService, rbac.PermissionAPITokensWrite), deps.AuthHandler.RevokeAPIToken)
	}

	if deps.UsersHandler != nil {
		usersGroup := api.Group("/users")
		usersGroup.Use(
			middleware.RequireModuleEnabled(deps.ModuleFlagsProvider, "administration"),
			middleware.ResolveJWTPrincipal(deps.JWTManager),
			middleware.RequireAuth(),
			middleware.RequireJWTUser(),
		)
		usersGroup.GET("", middleware.RequirePermission(deps.RBACService, rbac.PermissionUsersRead), deps.UsersHandler.List)
		usersGroup.GET("/:id", middleware.RequirePermission(deps.RBACService, rbac.PermissionUsersRead), deps.UsersHandler.Get)
		usersGroup.POST("", middleware.RequirePermission(deps.RBACService, rbac.PermissionUsersWrite), deps.UsersHandler.Create)
		usersGroup.PUT("/:id", middleware.RequirePermission(deps.RBACService, rbac.PermissionUsersWrite), deps.UsersHandler.Put)
		usersGroup.PATCH("/:id", middleware.RequirePermission(deps.RBACService, rbac.PermissionUsersWrite), deps.UsersHandler.Patch)
		usersGroup.POST("/:id/reset-password", middleware.RequirePermission(deps.RBACService, rbac.PermissionUsersWrite), deps.UsersHandler.ResetPassword)
	}

	if deps.AuditHandler != nil {
		auditGroup := api.Group("/audit")
		auditGroup.Use(
			middleware.RequireModuleEnabled(deps.ModuleFlagsProvider, "administration"),
			middleware.ResolveJWTPrincipal(deps.JWTManager),
			middleware.RequireAuth(),
		)
		auditGroup.GET("", middleware.RequirePermission(deps.RBACService, rbac.PermissionAuditRead), deps.AuditHandler.List)
	}

	if deps.RBACAdminHandler != nil {
		rolesGroup := api.Group("/admin/roles")
		rolesGroup.Use(
			middleware.RequireModuleEnabled(deps.ModuleFlagsProvider, "administration"),
			middleware.ResolveJWTPrincipal(deps.JWTManager),
			middleware.RequireAuth(),
			middleware.RequireJWTUser(),
		)
		rolesGroup.GET("", middleware.RequirePermission(deps.RBACService, rbac.PermissionUsersRead), deps.RBACAdminHandler.ListRoles)
		rolesGroup.POST("", middleware.RequirePermission(deps.RBACService, rbac.PermissionUsersWrite), deps.RBACAdminHandler.CreateRole)
		rolesGroup.GET("/:id", middleware.RequirePermission(deps.RBACService, rbac.PermissionUsersRead), deps.RBACAdminHandler.GetRole)
		rolesGroup.PUT("/:id", middleware.RequirePermission(deps.RBACService, rbac.PermissionUsersWrite), deps.RBACAdminHandler.PutRole)
		rolesGroup.PATCH("/:id", middleware.RequirePermission(deps.RBACService, rbac.PermissionUsersWrite), deps.RBACAdminHandler.PatchRole)
		rolesGroup.PUT("/:id/permissions", middleware.RequirePermission(deps.RBACService, rbac.PermissionUsersWrite), deps.RBACAdminHandler.UpdateRolePermissions)

		permissionsGroup := api.Group("/admin/permissions")
		permissionsGroup.Use(
			middleware.RequireModuleEnabled(deps.ModuleFlagsProvider, "administration"),
			middleware.ResolveJWTPrincipal(deps.JWTManager),
			middleware.RequireAuth(),
			middleware.RequireJWTUser(),
		)
		permissionsGroup.GET("", middleware.RequirePermission(deps.RBACService, rbac.PermissionUsersRead), deps.RBACAdminHandler.ListPermissions)
	}

	if deps.SettingsHandler != nil {
		api.GET("/settings/public/login-branding", middleware.RequireModuleEnabled(deps.ModuleFlagsProvider, "settings"), deps.SettingsHandler.GetPublicLoginBranding)

		settingsGroup := api.Group("/settings")
		settingsGroup.Use(
			middleware.RequireModuleEnabled(deps.ModuleFlagsProvider, "settings"),
			middleware.ResolveJWTPrincipal(deps.JWTManager),
			middleware.RequireAuth(),
			middleware.RequireJWTUser(),
		)
		settingsGroup.GET("/login-branding", middleware.RequirePermission(deps.RBACService, rbac.PermissionSettingsRead), deps.SettingsHandler.GetLoginBranding)
		settingsGroup.PUT("/login-branding", middleware.RequirePermission(deps.RBACService, rbac.PermissionSettingsWrite, middleware.WithAdminRoleOverride()), deps.SettingsHandler.UpdateLoginBranding)
		if deps.ModuleFlagsHandler != nil {
			settingsGroup.GET("/module-enablement", middleware.RequirePermission(deps.RBACService, rbac.PermissionSettingsRead), deps.ModuleFlagsHandler.GetEffective)
			settingsGroup.PUT("/module-enablement", middleware.RequirePermission(deps.RBACService, rbac.PermissionSettingsWrite, middleware.WithAdminRoleOverride()), deps.ModuleFlagsHandler.UpdateRuntimeOverrides)
		}
	}

	sukumad.RegisterRoutes(api, sukumad.RouteDeps{
		JWTManager:           deps.JWTManager,
		RBACService:          deps.RBACService,
		ModuleFlagsProvider:  deps.ModuleFlagsProvider,
		ServerHandler:        deps.ServerHandler,
		RequestHandler:       deps.RequestHandler,
		DeliveryHandler:      deps.DeliveryHandler,
		AsyncHandler:         deps.AsyncHandler,
		ObservabilityHandler: deps.ObservabilityHandler,
		DashboardHandler:     deps.DashboardHandler,
	})

	return r
}
