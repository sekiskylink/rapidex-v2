package orgunit

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
)

// RegisterRoutes registers the organisation unit API handlers under the provided router group.
// It expects the group to be configured with authentication and RBAC middleware upstream.
func RegisterRoutes(rg *gin.RouterGroup, svc *Service) {
    rg.GET("/orgunits", func(c *gin.Context) {
        page, _ := strconv.Atoi(c.DefaultQuery("page", "0"))
        pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
        search := c.Query("search")
        // ParentID is optional; parse if provided
        var parentID *int64
        if pid := c.Query("parentId"); pid != "" {
            if id64, err := strconv.ParseInt(pid, 10, 64); err == nil {
                parentID = &id64
            }
        }
        result, err := svc.List(c.Request.Context(), ListQuery{Page: page, PageSize: pageSize, Search: search, ParentID: parentID})
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusOK, result)
    })
    rg.POST("/orgunits", func(c *gin.Context) {
        var input OrgUnit
        if err := c.ShouldBindJSON(&input); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        created, err := svc.Create(c.Request.Context(), input)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusCreated, created)
    })
    rg.PUT("/orgunits/:id", func(c *gin.Context) {
        id, err := strconv.ParseInt(c.Param("id"), 10, 64)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
            return
        }
        var input OrgUnit
        if err := c.ShouldBindJSON(&input); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        input.ID = id
        updated, err := svc.Update(c.Request.Context(), input)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusOK, updated)
    })
    rg.DELETE("/orgunits/:id", func(c *gin.Context) {
        id, err := strconv.ParseInt(c.Param("id"), 10, 64)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
            return
        }
        if err := svc.Delete(c.Request.Context(), id); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        c.Status(http.StatusNoContent)
    })
}
