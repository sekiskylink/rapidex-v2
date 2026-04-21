package rapidex

import (
    "net/http"

    "github.com/gin-gonic/gin"
)

// RegisterRoutes registers Rapidex routes.  It expects authentication and
// RBAC middleware to be attached upstream.  The base router group should be
// prefixed with the API version (e.g. `/api/v1`).
func RegisterRoutes(rg *gin.RouterGroup, svc *IntegrationService) {
    // POST /rapidex/webhook
    rg.POST("/rapidex/webhook", func(c *gin.Context) {
        var webhook RapidProWebhook
        if err := c.ShouldBindJSON(&webhook); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        if err := svc.ProcessWebhook(c.Request.Context(), webhook); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        c.Status(http.StatusAccepted)
    })
}