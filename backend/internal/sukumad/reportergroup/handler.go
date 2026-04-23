package reportergroup

import (
	"net/http"
	"strconv"

	"basepro/backend/internal/apperror"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(c *gin.Context) {
	result, err := h.service.List(c.Request.Context(), ListQuery{
		Page:       parseInt(c.DefaultQuery("page", "0")),
		PageSize:   parseInt(c.DefaultQuery("pageSize", "20")),
		Search:     c.Query("search"),
		ActiveOnly: c.DefaultQuery("activeOnly", "false") == "true",
	})
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) ListOptions(c *gin.Context) {
	items, err := h.service.ListOptions(c.Request.Context())
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) Create(c *gin.Context) {
	var input ReporterGroup
	if err := c.ShouldBindJSON(&input); err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"body": []string{"invalid JSON payload"}}))
		return
	}
	item, err := h.service.Create(c.Request.Context(), input)
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *Handler) Update(c *gin.Context) {
	var input ReporterGroup
	if err := c.ShouldBindJSON(&input); err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"body": []string{"invalid JSON payload"}}))
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"invalid reporter group id"}}))
		return
	}
	input.ID = id
	item, err := h.service.Update(c.Request.Context(), input)
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func parseInt(value string) int {
	number, _ := strconv.Atoi(value)
	return number
}
