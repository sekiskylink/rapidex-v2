package auth

import (
	"net/http"
	"strconv"
	"strings"

	"basepro/backend/internal/apperror"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type logoutRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type createAPITokenRequest struct {
	Name             string   `json:"name"`
	ExpiresInSeconds *int64   `json:"expiresInSeconds"`
	Permissions      []string `json:"permissions"`
	ModuleScope      *string  `json:"moduleScope"`
}

func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" || req.Password == "" {
		apperror.Write(c, apperror.Unauthorized("Invalid credentials"))
		return
	}

	response, err := h.service.Login(c.Request.Context(), req.Username, req.Password, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		apperror.Write(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
		apperror.Write(c, apperror.RefreshInvalid("Refresh token is invalid"))
		return
	}

	response, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		apperror.Write(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) Logout(c *gin.Context) {
	var req logoutRequest
	_ = c.ShouldBindJSON(&req)

	if err := h.service.Logout(c.Request.Context(), req.RefreshToken, c.GetHeader("Authorization"), c.ClientIP(), c.Request.UserAgent()); err != nil {
		apperror.Write(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) Me(c *gin.Context) {
	claims, ok := claimsFromContext(c)
	if !ok {
		apperror.Write(c, apperror.Unauthorized("Missing authorization token"))
		return
	}
	c.JSON(http.StatusOK, h.service.Me(c.Request.Context(), claims))
}

func (h *Handler) ListAPITokens(c *gin.Context) {
	if _, ok := principalFromContext(c); !ok {
		apperror.Write(c, apperror.Unauthorized("Unauthorized"))
		return
	}

	tokens, err := h.service.ListAPITokens(c.Request.Context())
	if err != nil {
		apperror.Write(c, err)
		return
	}

	masked := make([]gin.H, 0, len(tokens))
	for _, token := range tokens {
		masked = append(masked, gin.H{
			"id":         token.ID,
			"name":       token.Name,
			"prefix":     token.Prefix,
			"revokedAt":  token.RevokedAt,
			"expiresAt":  token.ExpiresAt,
			"lastUsedAt": token.LastUsedAt,
			"createdAt":  token.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"items": masked})
}

func (h *Handler) CreateAPIToken(c *gin.Context) {
	principal, ok := principalFromContext(c)
	if !ok {
		apperror.Write(c, apperror.Unauthorized("Unauthorized"))
		return
	}

	var req createAPITokenRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		apperror.Write(c, apperror.Unauthorized("Token name is required"))
		return
	}

	result, err := h.service.CreateAPIToken(c.Request.Context(), actorUserID(principal), APITokenCreateInput{
		Name:             req.Name,
		ExpiresInSeconds: req.ExpiresInSeconds,
		Permissions:      req.Permissions,
		ModuleScope:      req.ModuleScope,
	}, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		apperror.Write(c, err)
		return
	}

	c.JSON(http.StatusCreated, result)
}

func (h *Handler) RevokeAPIToken(c *gin.Context) {
	principal, ok := principalFromContext(c)
	if !ok {
		apperror.Write(c, apperror.Unauthorized("Unauthorized"))
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apperror.Write(c, apperror.Unauthorized("Token not found"))
		return
	}

	token, err := h.service.RevokeAPIToken(c.Request.Context(), actorUserID(principal), id, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		apperror.Write(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         token.ID,
		"name":       token.Name,
		"prefix":     token.Prefix,
		"revokedAt":  token.RevokedAt,
		"expiresAt":  token.ExpiresAt,
		"lastUsedAt": token.LastUsedAt,
		"createdAt":  token.CreatedAt,
	})
}

func claimsFromContext(c *gin.Context) (Claims, bool) {
	value, ok := c.Get(ClaimsContextKey)
	if !ok {
		return Claims{}, false
	}
	claims, ok := value.(Claims)
	return claims, ok
}

func principalFromContext(c *gin.Context) (Principal, bool) {
	value, ok := c.Get(PrincipalContextKey)
	if !ok {
		return Principal{}, false
	}
	principal, ok := value.(Principal)
	return principal, ok
}

func actorUserID(principal Principal) *int64 {
	if principal.Type != "user" {
		return nil
	}
	return &principal.UserID
}
