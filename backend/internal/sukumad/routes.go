package sukumad

import (
	"basepro/backend/internal/auth"
	"basepro/backend/internal/middleware"
	"basepro/backend/internal/rbac"
	"basepro/backend/internal/sukumad/delivery"
	"basepro/backend/internal/sukumad/observability"
	requests "basepro/backend/internal/sukumad/request"
	"basepro/backend/internal/sukumad/server"
	"basepro/backend/internal/sukumad/worker"
	"github.com/gin-gonic/gin"
)

type RouteDeps struct {
	JWTManager           *auth.JWTManager
	RBACService          *rbac.Service
	ModuleFlagsProvider  func() map[string]bool
	ServerHandler        *server.Handler
	RequestHandler       *requests.Handler
	DeliveryHandler      *delivery.Handler
	WorkerHandler        *worker.Handler
	ObservabilityHandler *observability.Handler
}

func RegisterRoutes(api *gin.RouterGroup, deps RouteDeps) {
	registerServerRoutes(api, deps.ModuleFlagsProvider, deps.JWTManager, deps.RBACService, deps.ServerHandler)
	registerRequestRoutes(api, deps.ModuleFlagsProvider, deps.JWTManager, deps.RBACService, deps.RequestHandler)
	registerDeliveryRoutes(api, deps.ModuleFlagsProvider, deps.JWTManager, deps.RBACService, deps.DeliveryHandler)
	registerListRoute(api, deps.ModuleFlagsProvider, deps.JWTManager, deps.RBACService, "jobs", "/jobs", rbac.PermissionJobsRead, deps.WorkerHandler.List)
	registerListRoute(api, deps.ModuleFlagsProvider, deps.JWTManager, deps.RBACService, "observability", "/observability", rbac.PermissionObservabilityRead, deps.ObservabilityHandler.List)
}

func registerServerRoutes(
	api *gin.RouterGroup,
	moduleFlagsProvider func() map[string]bool,
	jwtManager *auth.JWTManager,
	rbacService *rbac.Service,
	handler *server.Handler,
) {
	if handler == nil {
		return
	}

	group := api.Group("/servers")
	group.Use(
		middleware.RequireModuleEnabled(moduleFlagsProvider, "servers"),
		middleware.ResolveJWTPrincipal(jwtManager),
		middleware.RequireAuth(),
		middleware.RequireJWTUser(),
	)
	group.GET("", middleware.RequirePermission(rbacService, rbac.PermissionServersRead), handler.List)
	group.GET("/:id", middleware.RequirePermission(rbacService, rbac.PermissionServersRead), handler.Get)
	group.POST("", middleware.RequirePermission(rbacService, rbac.PermissionServersWrite), handler.Create)
	group.PUT("/:id", middleware.RequirePermission(rbacService, rbac.PermissionServersWrite), handler.Update)
	group.DELETE("/:id", middleware.RequirePermission(rbacService, rbac.PermissionServersWrite), handler.Delete)
}

func registerRequestRoutes(
	api *gin.RouterGroup,
	moduleFlagsProvider func() map[string]bool,
	jwtManager *auth.JWTManager,
	rbacService *rbac.Service,
	handler *requests.Handler,
) {
	if handler == nil {
		return
	}

	group := api.Group("/requests")
	group.Use(
		middleware.RequireModuleEnabled(moduleFlagsProvider, "requests"),
		middleware.ResolveJWTPrincipal(jwtManager),
		middleware.RequireAuth(),
		middleware.RequireJWTUser(),
	)
	group.GET("", middleware.RequirePermission(rbacService, rbac.PermissionRequestsRead), handler.List)
	group.GET("/:id", middleware.RequirePermission(rbacService, rbac.PermissionRequestsRead), handler.Get)
	group.POST("", middleware.RequirePermission(rbacService, rbac.PermissionRequestsWrite), handler.Create)
}

func registerListRoute(
	api *gin.RouterGroup,
	moduleFlagsProvider func() map[string]bool,
	jwtManager *auth.JWTManager,
	rbacService *rbac.Service,
	moduleID string,
	path string,
	permission string,
	handler func(*gin.Context),
) {
	if handler == nil {
		return
	}

	group := api.Group(path)
	group.Use(
		middleware.RequireModuleEnabled(moduleFlagsProvider, moduleID),
		middleware.ResolveJWTPrincipal(jwtManager),
		middleware.RequireAuth(),
		middleware.RequireJWTUser(),
	)
	group.GET("", middleware.RequirePermission(rbacService, permission), handler)
}

func registerDeliveryRoutes(
	api *gin.RouterGroup,
	moduleFlagsProvider func() map[string]bool,
	jwtManager *auth.JWTManager,
	rbacService *rbac.Service,
	handler *delivery.Handler,
) {
	if handler == nil {
		return
	}

	group := api.Group("/deliveries")
	group.Use(
		middleware.RequireModuleEnabled(moduleFlagsProvider, "deliveries"),
		middleware.ResolveJWTPrincipal(jwtManager),
		middleware.RequireAuth(),
		middleware.RequireJWTUser(),
	)
	group.GET("", middleware.RequirePermission(rbacService, rbac.PermissionDeliveriesRead), handler.List)
	group.GET("/:id", middleware.RequirePermission(rbacService, rbac.PermissionDeliveriesRead), handler.Get)
	group.POST("/:id/retry", middleware.RequirePermission(rbacService, rbac.PermissionDeliveriesWrite), handler.Retry)
}
