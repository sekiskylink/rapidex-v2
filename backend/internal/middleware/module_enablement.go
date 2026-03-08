package middleware

import (
	"basepro/backend/internal/apperror"
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

		apperror.Write(c, apperror.ModuleDisabled(moduleID))
		c.Abort()
	}
}
