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
	IsActive *bool    `json:"isActive"`
	Roles    []string `json:"roles"`
}

type resetPasswordRequest struct {
	Password string `json:"password"`
}

func (h *Handler) List(c *gin.Context) {
	items, err := h.service.ListUsers(c.Request.Context())
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) Create(c *gin.Context) {
	principal, ok := principalFromContext(c)
	if !ok {
		apperror.Write(c, apperror.Unauthorized("Unauthorized"))
		return
	}

	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Username) == "" || strings.TrimSpace(req.Password) == "" {
		apperror.Write(c, apperror.Unauthorized("Invalid user payload"))
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
		Roles:    req.Roles,
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
		apperror.Write(c, apperror.Unauthorized("Invalid user id"))
		return
	}

	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperror.Write(c, apperror.Unauthorized("Invalid user payload"))
		return
	}

	if err := h.service.UpdateUser(c.Request.Context(), UpdateInput{
		UserID:   id,
		Roles:    req.Roles,
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
		apperror.Write(c, apperror.Unauthorized("Invalid user id"))
		return
	}

	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Password) == "" {
		apperror.Write(c, apperror.Unauthorized("Password is required"))
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
