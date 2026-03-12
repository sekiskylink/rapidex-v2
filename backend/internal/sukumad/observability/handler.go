package observability

import (
	"net/http"
	"strconv"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/listquery"
	"basepro/backend/internal/sukumad/ratelimit"
	"basepro/backend/internal/sukumad/worker"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListWorkers(c *gin.Context) {
	page, err := listquery.ParseInt(c.Query("page"), 1, 1, 100000, "page")
	if err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"page": []string{err.Error()}}))
		return
	}
	pageSize, err := listquery.ParseInt(c.Query("pageSize"), 25, 1, 200, "pageSize")
	if err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"pageSize": []string{err.Error()}}))
		return
	}
	sortField, sortOrder, err := listquery.ParseSort(c.Query("sort"))
	if err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"sort": []string{err.Error()}}))
		return
	}

	list, err := h.service.ListWorkers(c.Request.Context(), worker.ListQuery{
		Page:      page,
		PageSize:  pageSize,
		SortField: sortField,
		SortOrder: sortOrder,
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

func (h *Handler) GetWorker(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"invalid worker id"}}))
		return
	}
	item, err := h.service.GetWorker(c.Request.Context(), id)
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) ListRateLimits(c *gin.Context) {
	page, err := listquery.ParseInt(c.Query("page"), 1, 1, 100000, "page")
	if err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"page": []string{err.Error()}}))
		return
	}
	pageSize, err := listquery.ParseInt(c.Query("pageSize"), 25, 1, 200, "pageSize")
	if err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"pageSize": []string{err.Error()}}))
		return
	}
	sortField, sortOrder, err := listquery.ParseSort(c.Query("sort"))
	if err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"sort": []string{err.Error()}}))
		return
	}

	list, err := h.service.ListRateLimits(c.Request.Context(), ratelimit.ListQuery{
		Page:      page,
		PageSize:  pageSize,
		SortField: sortField,
		SortOrder: sortOrder,
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
