package async

import (
	"net/http"
	"strconv"
	"strings"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/listquery"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(c *gin.Context) {
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
	filterField, filterValue, err := listquery.ParseFilter(c.Query("filter"))
	if err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"filter": []string{err.Error()}}))
		return
	}

	status := strings.ToLower(strings.TrimSpace(c.Query("status")))
	if status == "" && filterField == "status" {
		status = strings.ToLower(strings.TrimSpace(filterValue))
	}

	search := listquery.ResolveSearch(c.Query("q"), filterField, filterValue, map[string]struct{}{
		"uid":          {},
		"deliveryUid":  {},
		"requestUid":   {},
		"remoteJobId":  {},
		"remoteStatus": {},
	})

	list, err := h.service.ListTasks(c.Request.Context(), ListQuery{
		Page:      page,
		PageSize:  pageSize,
		SortField: sortField,
		SortOrder: sortOrder,
		Filter:    search,
		Status:    status,
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

func (h *Handler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"invalid job id"}}))
		return
	}
	item, err := h.service.GetTask(c.Request.Context(), id)
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) ListPolls(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"invalid job id"}}))
		return
	}
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
	list, err := h.service.ListPolls(c.Request.Context(), id, ListQuery{
		Page:     page,
		PageSize: pageSize,
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
