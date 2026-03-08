package middleware

import (
	"net/http"

	"basepro/backend/internal/moduleenablement"
	"github.com/gin-gonic/gin"
)

func RequireModuleEnabled(getOverrides func() map[string]bool, moduleID string) gin.HandlerFunc {
	if getOverrides == nil {
		getOverrides = func() map[string]bool { return nil }
	}

	return func(c *gin.Context) {
		if moduleenablement.IsModuleEnabled(moduleID, getOverrides()) {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "MODULE_DISABLED",
				"message": "Module is disabled",
				"details": gin.H{"moduleId": moduleID},
			},
		})
	}
}
