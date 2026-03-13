package sukumad

import (
	"basepro/backend/internal/auth"
	"basepro/backend/internal/middleware"
	"basepro/backend/internal/rbac"
	asyncjobs "basepro/backend/internal/sukumad/async"
	"basepro/backend/internal/sukumad/delivery"
	"basepro/backend/internal/sukumad/observability"
	requests "basepro/backend/internal/sukumad/request"
	"basepro/backend/internal/sukumad/server"
	"github.com/gin-gonic/gin"
)

type RouteDeps struct {
	JWTManager           *auth.JWTManager
	RBACService          *rbac.Service
	ModuleFlagsProvider  func() map[string]bool
	ServerHandler        *server.Handler
	RequestHandler       *requests.Handler
	DeliveryHandler      *delivery.Handler
	AsyncHandler         *asyncjobs.Handler
	ObservabilityHandler *observability.Handler
}

func RegisterRoutes(api *gin.RouterGroup, deps RouteDeps) {
	registerServerRoutes(api, deps.ModuleFlagsProvider, deps.JWTManager, deps.RBACService, deps.ServerHandler)
	registerRequestRoutes(api, deps.ModuleFlagsProvider, deps.JWTManager, deps.RBACService, deps.RequestHandler, deps.ObservabilityHandler)
	registerDeliveryRoutes(api, deps.ModuleFlagsProvider, deps.JWTManager, deps.RBACService, deps.DeliveryHandler, deps.ObservabilityHandler)
	registerAsyncRoutes(api, deps.ModuleFlagsProvider, deps.JWTManager, deps.RBACService, deps.AsyncHandler, deps.ObservabilityHandler)
	registerObservabilityRoutes(api, deps.ModuleFlagsProvider, deps.JWTManager, deps.RBACService, deps.ObservabilityHandler)
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
	observabilityHandler *observability.Handler,
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
	if observabilityHandler != nil {
		group.GET("/:id/events", middleware.RequirePermission(rbacService, rbac.PermissionRequestsRead), observabilityHandler.ListRequestEvents)
	}
	group.POST("", middleware.RequirePermission(rbacService, rbac.PermissionRequestsWrite), handler.Create)
}

func registerAsyncRoutes(
	api *gin.RouterGroup,
	moduleFlagsProvider func() map[string]bool,
	jwtManager *auth.JWTManager,
	rbacService *rbac.Service,
	handler *asyncjobs.Handler,
	observabilityHandler *observability.Handler,
) {
	if handler == nil {
		return
	}

	group := api.Group("/jobs")
	group.Use(
		middleware.RequireModuleEnabled(moduleFlagsProvider, "jobs"),
		middleware.ResolveJWTPrincipal(jwtManager),
		middleware.RequireAuth(),
		middleware.RequireJWTUser(),
	)
	group.GET("", middleware.RequirePermission(rbacService, rbac.PermissionJobsRead), handler.List)
	group.GET("/:id", middleware.RequirePermission(rbacService, rbac.PermissionJobsRead), handler.Get)
	if observabilityHandler != nil {
		group.GET("/:id/events", middleware.RequirePermission(rbacService, rbac.PermissionJobsRead), observabilityHandler.ListJobEvents)
	}
	group.GET("/:id/polls", middleware.RequirePermission(rbacService, rbac.PermissionJobsRead), handler.ListPolls)
}

func registerDeliveryRoutes(
	api *gin.RouterGroup,
	moduleFlagsProvider func() map[string]bool,
	jwtManager *auth.JWTManager,
	rbacService *rbac.Service,
	handler *delivery.Handler,
	observabilityHandler *observability.Handler,
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
	if observabilityHandler != nil {
		group.GET("/:id/events", middleware.RequirePermission(rbacService, rbac.PermissionDeliveriesRead), observabilityHandler.ListDeliveryEvents)
	}
	group.POST("/:id/retry", middleware.RequirePermission(rbacService, rbac.PermissionDeliveriesWrite), handler.Retry)
}

func registerObservabilityRoutes(
	api *gin.RouterGroup,
	moduleFlagsProvider func() map[string]bool,
	jwtManager *auth.JWTManager,
	rbacService *rbac.Service,
	handler *observability.Handler,
) {
	if handler == nil {
		return
	}

	group := api.Group("/observability")
	group.Use(
		middleware.RequireModuleEnabled(moduleFlagsProvider, "observability"),
		middleware.ResolveJWTPrincipal(jwtManager),
		middleware.RequireAuth(),
		middleware.RequireJWTUser(),
	)
	group.GET("/workers", middleware.RequirePermission(rbacService, rbac.PermissionObservabilityRead), handler.ListWorkers)
	group.GET("/workers/:id", middleware.RequirePermission(rbacService, rbac.PermissionObservabilityRead), handler.GetWorker)
	group.GET("/rate-limits", middleware.RequirePermission(rbacService, rbac.PermissionObservabilityRead), handler.ListRateLimits)
	group.GET("/events", middleware.RequirePermission(rbacService, rbac.PermissionObservabilityRead), handler.ListEvents)
	group.GET("/events/:id", middleware.RequirePermission(rbacService, rbac.PermissionObservabilityRead), handler.GetEvent)
	group.GET("/trace", middleware.RequirePermission(rbacService, rbac.PermissionObservabilityRead), handler.Trace)
}
