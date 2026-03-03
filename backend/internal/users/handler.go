package users

import (
	"net/http"
	"strconv"
	"strings"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/auth"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type createUserRequest struct {
	Username string   `json:"username"`
	Password string   `json:"password"`
	IsActive *bool    `json:"isActive"`
	Roles    []string `json:"roles"`
}

type updateUserRequest struct {
	Username *string   `json:"username"`
	IsActive *bool     `json:"isActive"`
	Roles    *[]string `json:"roles"`
}

type resetPasswordRequest struct {
	Password string `json:"password"`
}

func (h *Handler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "25"))
	sortField, sortOrder := parseSortQuery(c.Query("sort"))
	filterField, filterValue := parseFilterQuery(c.Query("filter"))

	list, err := h.service.ListUsers(c.Request.Context(), ListQuery{
		Page:      page,
		PageSize:  pageSize,
		SortField: sortField,
		SortOrder: sortOrder,
		Filter:    filterForUsers(filterField, filterValue),
	})
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"items":      list.Items,
		"totalCount": list.Total,
		"page":       list.Page,
		"pageSize":   list.PageSize,
	})
}

func (h *Handler) Create(c *gin.Context) {
	principal, ok := principalFromContext(c)
	if !ok {
		apperror.Write(c, apperror.Unauthorized("Unauthorized"))
		return
	}

	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Username) == "" || strings.TrimSpace(req.Password) == "" {
		apperror.Write(c, apperror.Validation("username and password are required"))
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	created, err := h.service.CreateUser(c.Request.Context(), CreateInput{
		Username: req.Username,
		Password: req.Password,
		IsActive: isActive,
		Roles:    normalizeRoles(req.Roles),
		ActorID:  actorUserID(principal),
	})
	if err != nil {
		apperror.Write(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

func (h *Handler) Patch(c *gin.Context) {
	principal, ok := principalFromContext(c)
	if !ok {
		apperror.Write(c, apperror.Unauthorized("Unauthorized"))
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apperror.Write(c, apperror.Validation("invalid user id"))
		return
	}

	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperror.Write(c, apperror.Validation("invalid user payload"))
		return
	}
	if req.Username == nil && req.Roles == nil && req.IsActive == nil {
		apperror.Write(c, apperror.Validation("at least one update field is required"))
		return
	}
	if req.Username != nil && strings.TrimSpace(*req.Username) == "" {
		apperror.Write(c, apperror.Validation("username cannot be empty"))
		return
	}

	var roles *[]string
	if req.Roles != nil {
		cleaned := normalizeRoles(*req.Roles)
		roles = &cleaned
	}

	if err := h.service.UpdateUser(c.Request.Context(), UpdateInput{
		UserID:   id,
		Username: req.Username,
		Roles:    roles,
		IsActive: req.IsActive,
		ActorID:  actorUserID(principal),
	}); err != nil {
		apperror.Write(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) ResetPassword(c *gin.Context) {
	principal, ok := principalFromContext(c)
	if !ok {
		apperror.Write(c, apperror.Unauthorized("Unauthorized"))
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apperror.Write(c, apperror.Validation("invalid user id"))
		return
	}

	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Password) == "" {
		apperror.Write(c, apperror.Validation("password is required"))
		return
	}

	if err := h.service.ResetPassword(c.Request.Context(), actorUserID(principal), id, req.Password); err != nil {
		apperror.Write(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func principalFromContext(c *gin.Context) (auth.Principal, bool) {
	value, ok := c.Get(auth.PrincipalContextKey)
	if !ok {
		return auth.Principal{}, false
	}
	principal, ok := value.(auth.Principal)
	return principal, ok
}

func actorUserID(principal auth.Principal) *int64 {
	if principal.Type != "user" {
		return nil
	}
	return &principal.UserID
}

func parseSortQuery(raw string) (field string, order string) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", ""
	}
	parts := strings.SplitN(value, ":", 2)
	if len(parts) == 1 {
		return strings.TrimSpace(parts[0]), "asc"
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func parseFilterQuery(raw string) (field string, value string) {
	filter := strings.TrimSpace(raw)
	if filter == "" {
		return "", ""
	}
	parts := strings.SplitN(filter, ":", 2)
	if len(parts) == 1 {
		return "", strings.TrimSpace(parts[0])
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func filterForUsers(field, value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	if field == "" || field == "username" {
		return value
	}
	return ""
}

func normalizeRoles(roles []string) []string {
	if len(roles) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(roles))
	for _, role := range roles {
		trimmed := strings.TrimSpace(role)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}
