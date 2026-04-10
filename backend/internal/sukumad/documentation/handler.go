package docbrowser

import (
	"net/http"

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
	items, err := h.service.ListDocuments(c.Request.Context())
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) Get(c *gin.Context) {
	item, err := h.service.GetDocument(c.Request.Context(), c.Param("slug"))
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}
