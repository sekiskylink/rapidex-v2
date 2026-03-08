package moduleenablement

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	getOverrides func() map[string]bool
}

func NewHandler(getOverrides func() map[string]bool) *Handler {
	if getOverrides == nil {
		getOverrides = func() map[string]bool { return nil }
	}
	return &Handler{getOverrides: getOverrides}
}

func (h *Handler) GetEffective(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"modules": ResolveEffective(h.getOverrides()),
	})
}
