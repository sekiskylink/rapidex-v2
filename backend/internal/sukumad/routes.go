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
	registerListRoute(api, deps.ModuleFlagsProvider, deps.JWTManager, deps.RBACService, "servers", "/servers", rbac.PermissionServersRead, deps.ServerHandler.List)
	registerListRoute(api, deps.ModuleFlagsProvider, deps.JWTManager, deps.RBACService, "requests", "/requests", rbac.PermissionRequestsRead, deps.RequestHandler.List)
	registerListRoute(api, deps.ModuleFlagsProvider, deps.JWTManager, deps.RBACService, "deliveries", "/deliveries", rbac.PermissionDeliveriesRead, deps.DeliveryHandler.List)
	registerListRoute(api, deps.ModuleFlagsProvider, deps.JWTManager, deps.RBACService, "jobs", "/jobs", rbac.PermissionJobsRead, deps.WorkerHandler.List)
	registerListRoute(api, deps.ModuleFlagsProvider, deps.JWTManager, deps.RBACService, "observability", "/observability", rbac.PermissionObservabilityRead, deps.ObservabilityHandler.List)
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
