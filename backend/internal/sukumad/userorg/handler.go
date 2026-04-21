package userorg

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
)

// RegisterRoutes registers API endpoints for managing user organisation unit
// assignments.  These endpoints assume authentication and RBAC middleware
// upstream.  Only administrators should be allowed to assign or remove
// assignments.
func RegisterRoutes(rg *gin.RouterGroup, svc *Service) {
    // GET /user_org_units/:userId → list assigned org units
    rg.GET("/user_org_units/:userId", func(c *gin.Context) {
        userID, err := strconv.ParseInt(c.Param("userId"), 10, 64)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
            return
        }
        ids, err := svc.GetUserOrgUnitIDs(c.Request.Context(), userID)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusOK, gin.H{"orgUnitIds": ids})
    })
    // POST /user_org_units → assign user to org unit
    rg.POST("/user_org_units", func(c *gin.Context) {
        var req AssignmentRequest
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        if req.UserID == 0 || req.OrgUnitID == 0 {
            c.JSON(http.StatusBadRequest, gin.H{"error": "userId and orgUnitId are required"})
            return
        }
        if err := svc.Assign(c.Request.Context(), req.UserID, req.OrgUnitID); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        c.Status(http.StatusCreated)
    })
    // DELETE /user_org_units/:userId/:orgUnitId → remove assignment
    rg.DELETE("/user_org_units/:userId/:orgUnitId", func(c *gin.Context) {
        userID, err1 := strconv.ParseInt(c.Param("userId"), 10, 64)
        orgID, err2 := strconv.ParseInt(c.Param("orgUnitId"), 10, 64)
        if err1 != nil || err2 != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
            return
        }
        if err := svc.Remove(c.Request.Context(), userID, orgID); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        c.Status(http.StatusNoContent)
    })
}