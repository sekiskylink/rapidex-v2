package rapidex

import (
	"net/http"

	"basepro/backend/internal/apperror"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers RapidEx routes. It expects authentication and
// RBAC middleware to be attached upstream. The base router group should be
// prefixed with the API version (for example `/api/v1`).
func RegisterRoutes(rg *gin.RouterGroup, svc *IntegrationService) {
	rg.POST("/webhook", func(c *gin.Context) {
		var webhook RapidProWebhook
		if err := c.ShouldBindJSON(&webhook); err != nil {
			apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{
				"body": []string{err.Error()},
			}))
			return
		}
		if err := svc.ProcessWebhook(c.Request.Context(), webhook); err != nil {
			apperror.Write(c, err)
			return
		}
		c.Status(http.StatusAccepted)
	})
}
