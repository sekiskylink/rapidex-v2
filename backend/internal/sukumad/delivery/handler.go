package delivery

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/auth"
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
	if status != "" && !isValidStatus(status) {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"status": []string{"must be one of pending, running, succeeded, failed, retrying"}}))
		return
	}

	serverFilter := strings.TrimSpace(c.Query("server"))
	if serverFilter == "" && filterField == "server" {
		serverFilter = strings.TrimSpace(filterValue)
	}

	var dateFilter *time.Time
	dateValue := strings.TrimSpace(c.Query("date"))
	if dateValue != "" {
		parsed, parseErr := time.Parse("2006-01-02", dateValue)
		if parseErr != nil {
			apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"date": []string{"must use YYYY-MM-DD"}}))
			return
		}
		dateFilter = &parsed
	}

	search := listquery.ResolveSearch(c.Query("q"), filterField, filterValue, map[string]struct{}{
		"uid":        {},
		"requestUid": {},
		"server":     {},
		"serverName": {},
	})

	list, err := h.service.ListDeliveries(c.Request.Context(), ListQuery{
		Page:      page,
		PageSize:  pageSize,
		SortField: sortField,
		SortOrder: sortOrder,
		Filter:    search,
		Status:    status,
		Server:    serverFilter,
		Date:      dateFilter,
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
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"invalid delivery id"}}))
		return
	}

	item, err := h.service.GetDelivery(c.Request.Context(), id)
	if err != nil {
		apperror.Write(c, err)
		return
	}

	c.JSON(http.StatusOK, item)
}

func (h *Handler) Retry(c *gin.Context) {
	principal, ok := principalFromContext(c)
	if !ok {
		apperror.Write(c, apperror.Unauthorized("Unauthorized"))
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"invalid delivery id"}}))
		return
	}

	record, err := h.service.RetryDelivery(c.Request.Context(), actorUserID(principal), id)
	if err != nil {
		apperror.Write(c, err)
		return
	}

	c.JSON(http.StatusCreated, record)
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
	userID, ok := principal.EffectiveUserID()
	if !ok {
		return nil
	}
	return &userID
}

func isValidStatus(value string) bool {
	switch value {
	case StatusPending, StatusRunning, StatusSucceeded, StatusFailed, StatusRetrying:
		return true
	default:
		return false
	}
}
