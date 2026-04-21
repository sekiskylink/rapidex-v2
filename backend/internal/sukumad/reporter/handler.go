package reporter

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
)

// RegisterRoutes registers the reporter API handlers.
// It expects authentication and RBAC middleware to be attached upstream.
func RegisterRoutes(rg *gin.RouterGroup, svc *Service) {
    rg.GET("/reporters", func(c *gin.Context) {
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
        result, err := svc.List(c.Request.Context(), ListQuery{Page: page, PageSize: pageSize, Search: search, OrgUnitID: orgUnitID, OnlyActive: onlyActive})
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusOK, result)
    })
    rg.POST("/reporters", func(c *gin.Context) {
        var input Reporter
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
    rg.PUT("/reporters/:id", func(c *gin.Context) {
        id, err := strconv.ParseInt(c.Param("id"), 10, 64)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
            return
        }
        var input Reporter
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
    rg.DELETE("/reporters/:id", func(c *gin.Context) {
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
