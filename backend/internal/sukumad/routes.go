package sukumad

import (
	"net/http"
	"strconv"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/auth"
	"basepro/backend/internal/middleware"
	"basepro/backend/internal/rbac"
	asyncjobs "basepro/backend/internal/sukumad/async"
	"basepro/backend/internal/sukumad/dashboard"
	"basepro/backend/internal/sukumad/delivery"
	documentation "basepro/backend/internal/sukumad/documentation"
	"basepro/backend/internal/sukumad/observability"
	"basepro/backend/internal/sukumad/orgunit"
	"basepro/backend/internal/sukumad/reporter"
	requests "basepro/backend/internal/sukumad/request"
	"basepro/backend/internal/sukumad/scheduler"
	"basepro/backend/internal/sukumad/server"
	"basepro/backend/internal/sukumad/userorg"
	"github.com/gin-gonic/gin"
)

type RouteDeps struct {
	JWTManager           *auth.JWTManager
	RBACService          *rbac.Service
	ModuleFlagsProvider  func() map[string]bool
	ServerHandler        *server.Handler
	RequestHandler       *requests.Handler
	SchedulerHandler     *scheduler.Handler
	DeliveryHandler      *delivery.Handler
	AsyncHandler         *asyncjobs.Handler
	ObservabilityHandler *observability.Handler
	DashboardHandler     *dashboard.Handler
	DocumentationHandler *documentation.Handler
	OrgUnitService       *orgunit.Service
	ReporterService      *reporter.Service
	UserOrgUnitService   *userorg.Service
}

func RegisterRoutes(api *gin.RouterGroup, deps RouteDeps) {
	registerServerRoutes(api, deps.ModuleFlagsProvider, deps.JWTManager, deps.RBACService, deps.ServerHandler)
	registerRequestRoutes(api, deps.ModuleFlagsProvider, deps.JWTManager, deps.RBACService, deps.RequestHandler, deps.ObservabilityHandler)
	registerSchedulerRoutes(api, deps.ModuleFlagsProvider, deps.JWTManager, deps.RBACService, deps.SchedulerHandler)
	registerDeliveryRoutes(api, deps.ModuleFlagsProvider, deps.JWTManager, deps.RBACService, deps.DeliveryHandler, deps.ObservabilityHandler)
	registerAsyncRoutes(api, deps.ModuleFlagsProvider, deps.JWTManager, deps.RBACService, deps.AsyncHandler, deps.ObservabilityHandler)
	registerObservabilityRoutes(api, deps.ModuleFlagsProvider, deps.JWTManager, deps.RBACService, deps.ObservabilityHandler)
	registerDashboardRoutes(api, deps.ModuleFlagsProvider, deps.JWTManager, deps.RBACService, deps.DashboardHandler)
	registerDocumentationRoutes(api, deps.ModuleFlagsProvider, deps.JWTManager, deps.DocumentationHandler)
	registerOrgUnitRoutes(api, deps.ModuleFlagsProvider, deps.JWTManager, deps.RBACService, deps.OrgUnitService)
	registerReporterRoutes(api, deps.ModuleFlagsProvider, deps.JWTManager, deps.RBACService, deps.ReporterService)
	registerUserOrgUnitRoutes(api, deps.ModuleFlagsProvider, deps.JWTManager, deps.RBACService, deps.UserOrgUnitService)
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

	externalGroup := api.Group("/external/servers")
	externalGroup.Use(
		middleware.RequireModuleEnabled(moduleFlagsProvider, "servers"),
		middleware.ResolveJWTPrincipal(jwtManager),
		middleware.RequireAuth(),
	)
	externalGroup.GET("", middleware.RequirePermission(rbacService, rbac.PermissionServersRead), handler.ListExternal)
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
	group.DELETE("/:id", middleware.RequirePermission(rbacService, rbac.PermissionRequestsWrite), handler.Delete)

	externalGroup := api.Group("/external/requests")
	externalGroup.Use(
		middleware.RequireModuleEnabled(moduleFlagsProvider, "requests"),
		middleware.ResolveJWTPrincipal(jwtManager),
		middleware.RequireAuth(),
	)
	externalGroup.GET("/lookup", middleware.RequirePermission(rbacService, rbac.PermissionRequestsRead), handler.LookupExternal)
	externalGroup.GET("/summary", middleware.RequirePermission(rbacService, rbac.PermissionRequestsRead), handler.SummaryExternal)
	externalGroup.GET("/:uid", middleware.RequirePermission(rbacService, rbac.PermissionRequestsRead), handler.GetExternal)
	externalGroup.POST("", middleware.RequirePermission(rbacService, rbac.PermissionRequestsWrite), handler.CreateExternal)
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

func registerSchedulerRoutes(
	api *gin.RouterGroup,
	moduleFlagsProvider func() map[string]bool,
	jwtManager *auth.JWTManager,
	rbacService *rbac.Service,
	handler *scheduler.Handler,
) {
	if handler == nil {
		return
	}

	group := api.Group("/scheduler")
	group.Use(
		middleware.RequireModuleEnabled(moduleFlagsProvider, "scheduler"),
		middleware.ResolveJWTPrincipal(jwtManager),
		middleware.RequireAuth(),
		middleware.RequireJWTUser(),
	)
	group.GET("/jobs", middleware.RequirePermission(rbacService, rbac.PermissionSchedulerRead), handler.ListJobs)
	group.POST("/jobs", middleware.RequirePermission(rbacService, rbac.PermissionSchedulerWrite), handler.CreateJob)
	group.GET("/jobs/:id", middleware.RequirePermission(rbacService, rbac.PermissionSchedulerRead), handler.GetJob)
	group.PUT("/jobs/:id", middleware.RequirePermission(rbacService, rbac.PermissionSchedulerWrite), handler.UpdateJob)
	group.POST("/jobs/:id/enable", middleware.RequirePermission(rbacService, rbac.PermissionSchedulerWrite), handler.EnableJob)
	group.POST("/jobs/:id/disable", middleware.RequirePermission(rbacService, rbac.PermissionSchedulerWrite), handler.DisableJob)
	group.POST("/jobs/:id/run-now", middleware.RequirePermission(rbacService, rbac.PermissionSchedulerWrite), handler.RunNow)
	group.GET("/jobs/:id/runs", middleware.RequirePermission(rbacService, rbac.PermissionSchedulerRead), handler.ListRuns)
	group.GET("/runs/:id", middleware.RequirePermission(rbacService, rbac.PermissionSchedulerRead), handler.GetRun)
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

func registerDashboardRoutes(
	api *gin.RouterGroup,
	moduleFlagsProvider func() map[string]bool,
	jwtManager *auth.JWTManager,
	rbacService *rbac.Service,
	handler *dashboard.Handler,
) {
	if handler == nil {
		return
	}

	group := api.Group("/dashboard")
	group.Use(
		middleware.RequireModuleEnabled(moduleFlagsProvider, "dashboard"),
		middleware.ResolveJWTPrincipal(jwtManager),
		middleware.ResolveJWTPrincipalFromQuery(jwtManager, "access_token"),
		middleware.RequireAuth(),
		middleware.RequireJWTUser(),
	)
	group.GET("/operations", middleware.RequirePermission(rbacService, rbac.PermissionObservabilityRead), handler.GetOperations)
	group.GET("/operations/events", middleware.RequirePermission(rbacService, rbac.PermissionObservabilityRead), handler.StreamOperations)
}

func registerDocumentationRoutes(
	api *gin.RouterGroup,
	moduleFlagsProvider func() map[string]bool,
	jwtManager *auth.JWTManager,
	handler *documentation.Handler,
) {
	if handler == nil {
		return
	}

	group := api.Group("/documentation")
	group.Use(
		middleware.RequireModuleEnabled(moduleFlagsProvider, "documentation"),
		middleware.ResolveJWTPrincipal(jwtManager),
		middleware.RequireAuth(),
		middleware.RequireJWTUser(),
	)
	group.GET("", handler.List)
	group.GET("/:slug", handler.Get)
}

func registerOrgUnitRoutes(
	api *gin.RouterGroup,
	moduleFlagsProvider func() map[string]bool,
	jwtManager *auth.JWTManager,
	rbacService *rbac.Service,
	service *orgunit.Service,
) {
	if service == nil {
		return
	}

	group := api.Group("")
	group.Use(
		middleware.RequireModuleEnabled(moduleFlagsProvider, "orgunits"),
		middleware.ResolveJWTPrincipal(jwtManager),
		middleware.RequireAuth(),
		middleware.RequireJWTUser(),
	)
	group.GET("/orgunits", middleware.RequirePermission(rbacService, rbac.PermissionOrgUnitsRead), func(c *gin.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("page", "0"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
		search := c.Query("search")
		var parentID *int64
		if pid := c.Query("parentId"); pid != "" {
			if id64, err := strconv.ParseInt(pid, 10, 64); err == nil {
				parentID = &id64
			}
		}
		result, err := service.List(c.Request.Context(), orgunit.ListQuery{Page: page, PageSize: pageSize, Search: search, ParentID: parentID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})
	group.POST("/orgunits", middleware.RequirePermission(rbacService, rbac.PermissionOrgUnitsWrite), func(c *gin.Context) {
		var input orgunit.OrgUnit
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		created, err := service.Create(c.Request.Context(), input)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, created)
	})
	group.PUT("/orgunits/:id", middleware.RequirePermission(rbacService, rbac.PermissionOrgUnitsWrite), func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var input orgunit.OrgUnit
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		input.ID = id
		updated, err := service.Update(c.Request.Context(), input)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, updated)
	})
	group.DELETE("/orgunits/:id", middleware.RequirePermission(rbacService, rbac.PermissionOrgUnitsWrite), func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if err := service.Delete(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	})
}

func registerReporterRoutes(
	api *gin.RouterGroup,
	moduleFlagsProvider func() map[string]bool,
	jwtManager *auth.JWTManager,
	rbacService *rbac.Service,
	service *reporter.Service,
) {
	if service == nil {
		return
	}

	group := api.Group("")
	group.Use(
		middleware.RequireModuleEnabled(moduleFlagsProvider, "reporters"),
		middleware.ResolveJWTPrincipal(jwtManager),
		middleware.RequireAuth(),
		middleware.RequireJWTUser(),
	)
	group.GET("/reporters", middleware.RequirePermission(rbacService, rbac.PermissionReportersRead), func(c *gin.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("page", "0"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
		search := c.Query("search")
		var orgUnitID *int64
		if oid := c.Query("orgUnitId"); oid != "" {
			if id64, err := strconv.ParseInt(oid, 10, 64); err == nil {
				orgUnitID = &id64
			}
		}
		onlyActive := c.DefaultQuery("onlyActive", "false") == "true"
		result, err := service.List(c.Request.Context(), reporter.ListQuery{Page: page, PageSize: pageSize, Search: search, OrgUnitID: orgUnitID, OnlyActive: onlyActive})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})
	group.POST("/reporters", middleware.RequirePermission(rbacService, rbac.PermissionReportersWrite), func(c *gin.Context) {
		var input reporter.Reporter
		if err := c.ShouldBindJSON(&input); err != nil {
			apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"body": []string{"invalid JSON payload"}}))
			return
		}
		created, err := service.Create(c.Request.Context(), input)
		if err != nil {
			apperror.Write(c, err)
			return
		}
		c.JSON(http.StatusCreated, created)
	})
	group.PUT("/reporters/:id", middleware.RequirePermission(rbacService, rbac.PermissionReportersWrite), func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"invalid reporter id"}}))
			return
		}
		var input reporter.Reporter
		if err := c.ShouldBindJSON(&input); err != nil {
			apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"body": []string{"invalid JSON payload"}}))
			return
		}
		input.ID = id
		updated, err := service.Update(c.Request.Context(), input)
		if err != nil {
			apperror.Write(c, err)
			return
		}
		c.JSON(http.StatusOK, updated)
	})
	group.DELETE("/reporters/:id", middleware.RequirePermission(rbacService, rbac.PermissionReportersWrite), func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"invalid reporter id"}}))
			return
		}
		if err := service.Delete(c.Request.Context(), id); err != nil {
			apperror.Write(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})
	group.POST("/reporters/:id/sync", middleware.RequirePermission(rbacService, rbac.PermissionReportersWrite), func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"invalid reporter id"}}))
			return
		}
		result, err := service.SyncReporter(c.Request.Context(), id)
		if err != nil {
			apperror.Write(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	})
	group.POST("/reporters/:id/send-message", middleware.RequirePermission(rbacService, rbac.PermissionReportersWrite), func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"invalid reporter id"}}))
			return
		}
		var payload struct {
			Text string `json:"text"`
		}
		if err := c.ShouldBindJSON(&payload); err != nil {
			apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"body": []string{"invalid JSON payload"}}))
			return
		}
		result, err := service.SendMessage(c.Request.Context(), id, payload.Text)
		if err != nil {
			apperror.Write(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	})
	group.POST("/reporters/bulk/sync", middleware.RequirePermission(rbacService, rbac.PermissionReportersWrite), func(c *gin.Context) {
		var payload struct {
			ReporterIDs []int64 `json:"reporterIds"`
		}
		if err := c.ShouldBindJSON(&payload); err != nil {
			apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"body": []string{"invalid JSON payload"}}))
			return
		}
		result, err := service.SyncReporters(c.Request.Context(), payload.ReporterIDs)
		if err != nil {
			apperror.Write(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	})
	group.POST("/reporters/bulk/broadcast", middleware.RequirePermission(rbacService, rbac.PermissionReportersWrite), func(c *gin.Context) {
		var payload struct {
			ReporterIDs []int64 `json:"reporterIds"`
			Text        string  `json:"text"`
		}
		if err := c.ShouldBindJSON(&payload); err != nil {
			apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"body": []string{"invalid JSON payload"}}))
			return
		}
		result, err := service.BroadcastMessage(c.Request.Context(), payload.ReporterIDs, payload.Text)
		if err != nil {
			apperror.Write(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	})
}

func registerUserOrgUnitRoutes(
	api *gin.RouterGroup,
	moduleFlagsProvider func() map[string]bool,
	jwtManager *auth.JWTManager,
	rbacService *rbac.Service,
	service *userorg.Service,
) {
	if service == nil {
		return
	}

	group := api.Group("")
	group.Use(
		middleware.RequireModuleEnabled(moduleFlagsProvider, "orgunits"),
		middleware.ResolveJWTPrincipal(jwtManager),
		middleware.RequireAuth(),
		middleware.RequireJWTUser(),
	)
	group.GET("/user-org-units/:userId", middleware.RequirePermission(rbacService, rbac.PermissionOrgUnitsRead), func(c *gin.Context) {
		userID, err := strconv.ParseInt(c.Param("userId"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
			return
		}
		ids, err := service.GetUserOrgUnitIDs(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"orgUnitIds": ids})
	})
	group.POST("/user-org-units", middleware.RequirePermission(rbacService, rbac.PermissionOrgUnitsWrite), func(c *gin.Context) {
		var req userorg.AssignmentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.UserID == 0 || req.OrgUnitID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "userId and orgUnitId are required"})
			return
		}
		if err := service.Assign(c.Request.Context(), req.UserID, req.OrgUnitID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusCreated)
	})
	group.DELETE("/user-org-units/:userId/:orgUnitId", middleware.RequirePermission(rbacService, rbac.PermissionOrgUnitsWrite), func(c *gin.Context) {
		userID, err1 := strconv.ParseInt(c.Param("userId"), 10, 64)
		orgID, err2 := strconv.ParseInt(c.Param("orgUnitId"), 10, 64)
		if err1 != nil || err2 != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if err := service.Remove(c.Request.Context(), userID, orgID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	})
}
