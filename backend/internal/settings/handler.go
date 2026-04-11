package settings

import (
	"net/http"

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

type updateLoginBrandingRequest struct {
	ApplicationDisplayName string  `json:"applicationDisplayName"`
	LoginImageURL          *string `json:"loginImageUrl"`
	LoginImageAssetPath    *string `json:"loginImageAssetPath"`
}

func (h *Handler) GetPublicLoginBranding(c *gin.Context) {
	branding, err := h.service.GetLoginBranding(c.Request.Context())
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, branding)
}

func (h *Handler) GetLoginBranding(c *gin.Context) {
	branding, err := h.service.GetLoginBranding(c.Request.Context())
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, branding)
}

func (h *Handler) GetRuntimeConfig(c *gin.Context) {
	config, err := h.service.GetRuntimeConfig(c.Request.Context())
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"config": config})
}

func (h *Handler) UpdateLoginBranding(c *gin.Context) {
	principal, ok := principalFromContext(c)
	if !ok {
		apperror.Write(c, apperror.Unauthorized("Unauthorized"))
		return
	}

	var req updateLoginBrandingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{
			"body": []string{"invalid JSON payload"},
		}))
		return
	}

	branding, err := h.service.UpdateLoginBranding(c.Request.Context(), LoginBrandingUpdateInput{
		ApplicationDisplayName: req.ApplicationDisplayName,
		LoginImageURL:          req.LoginImageURL,
		LoginImageAssetPath:    req.LoginImageAssetPath,
	}, actorUserID(principal))
	if err != nil {
		apperror.Write(c, err)
		return
	}

	c.JSON(http.StatusOK, branding)
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
