package settings

import (
	"net/http"
	"strconv"

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

type updateRapidProReporterSyncRequest struct {
	RapidProServerCode string                         `json:"rapidProServerCode"`
	Mappings           []RapidProReporterFieldMapping `json:"mappings"`
}

type updateRapidexWebhookMappingsRequest struct {
	RapidProServerCode string                        `json:"rapidProServerCode"`
	Dhis2ServerCode    string                        `json:"dhis2ServerCode"`
	Mappings           []RapidexWebhookMappingConfig `json:"mappings"`
}

type importRapidexWebhookMappingsRequest struct {
	YAML string `json:"yaml"`
}

type refreshRapidexWebhookMetadataRequest struct {
	RapidProServerCode string `json:"rapidProServerCode"`
	Dhis2ServerCode    string `json:"dhis2ServerCode"`
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

func (h *Handler) GetRapidProReporterSync(c *gin.Context) {
	settings, err := h.service.GetRapidProReporterSync(c.Request.Context())
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, settings)
}

func (h *Handler) RefreshRapidProReporterSyncFields(c *gin.Context) {
	principal, ok := principalFromContext(c)
	if !ok {
		apperror.Write(c, apperror.Unauthorized("Unauthorized"))
		return
	}
	settings, err := h.service.RefreshRapidProReporterSyncFields(c.Request.Context(), actorUserID(principal))
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, settings)
}

func (h *Handler) UpdateRapidProReporterSync(c *gin.Context) {
	principal, ok := principalFromContext(c)
	if !ok {
		apperror.Write(c, apperror.Unauthorized("Unauthorized"))
		return
	}

	var req updateRapidProReporterSyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{
			"body": []string{"invalid JSON payload"},
		}))
		return
	}

	settings, err := h.service.UpdateRapidProReporterSync(c.Request.Context(), RapidProReporterSyncUpdateInput{
		RapidProServerCode: req.RapidProServerCode,
		Mappings:           req.Mappings,
	}, actorUserID(principal))
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, settings)
}

func (h *Handler) GetRapidexWebhookMappings(c *gin.Context) {
	settings, err := h.service.GetRapidexWebhookMappings(c.Request.Context())
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, settings)
}

func (h *Handler) UpdateRapidexWebhookMappings(c *gin.Context) {
	principal, ok := principalFromContext(c)
	if !ok {
		apperror.Write(c, apperror.Unauthorized("Unauthorized"))
		return
	}

	var req updateRapidexWebhookMappingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{
			"body": []string{"invalid JSON payload"},
		}))
		return
	}

	settings, err := h.service.UpdateRapidexWebhookMappings(c.Request.Context(), RapidexWebhookMappingsUpdateInput{
		RapidProServerCode: req.RapidProServerCode,
		Dhis2ServerCode:    req.Dhis2ServerCode,
		Mappings:           req.Mappings,
	}, actorUserID(principal))
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, settings)
}

func (h *Handler) GetRapidexWebhookMetadata(c *gin.Context) {
	payload, err := h.service.GetRapidexWebhookMetadata(c.Request.Context())
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, payload)
}

func (h *Handler) RefreshRapidexWebhookMetadata(c *gin.Context) {
	principal, ok := principalFromContext(c)
	if !ok {
		apperror.Write(c, apperror.Unauthorized("Unauthorized"))
		return
	}

	var req refreshRapidexWebhookMetadataRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{
			"body": []string{"invalid JSON payload"},
		}))
		return
	}

	payload, err := h.service.RefreshRapidexWebhookMetadata(c.Request.Context(), RapidexWebhookMetadataRefreshInput{
		RapidProServerCode: req.RapidProServerCode,
		Dhis2ServerCode:    req.Dhis2ServerCode,
	}, actorUserID(principal))
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, payload)
}

func (h *Handler) ImportRapidexWebhookMappingsYAML(c *gin.Context) {
	principal, ok := principalFromContext(c)
	if !ok {
		apperror.Write(c, apperror.Unauthorized("Unauthorized"))
		return
	}

	var req importRapidexWebhookMappingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{
			"body": []string{"invalid JSON payload"},
		}))
		return
	}

	settings, err := h.service.ImportRapidexWebhookMappingsYAML(c.Request.Context(), RapidexWebhookMappingsImportInput{
		YAML: req.YAML,
	}, actorUserID(principal))
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, settings)
}

func (h *Handler) ExportRapidexWebhookMappingsYAML(c *gin.Context) {
	payload, err := h.service.ExportRapidexWebhookMappingsYAML(c.Request.Context())
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, payload)
}

func (h *Handler) ListRapidProReporterSyncPreviewReporters(c *gin.Context) {
	reporters, err := h.service.ListRapidProReporterSyncPreviewReporters(c.Request.Context())
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": reporters})
}

func (h *Handler) GetRapidProReporterSyncPreview(c *gin.Context) {
	reporterID, err := strconv.ParseInt(c.Query("reporterId"), 10, 64)
	if err != nil || reporterID <= 0 {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{
			"reporterId": []string{"is required"},
		}))
		return
	}
	preview, err := h.service.GetRapidProReporterSyncPreview(c.Request.Context(), reporterID)
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, preview)
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
