package observability

import (
	"net/http"
	"strconv"
	"strings"
	"time"

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

func (h *Handler) ListEvents(c *gin.Context) {
	query, err := parseEventListQuery(c)
	if err != nil {
		apperror.Write(c, err)
		return
	}
	list, err := h.service.ListEvents(c.Request.Context(), query)
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

func (h *Handler) GetEvent(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"invalid event id"}}))
		return
	}
	item, err := h.service.GetEvent(c.Request.Context(), id)
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) ListRequestEvents(c *gin.Context) {
	id, err := parseEntityID(c.Param("id"), "request")
	if err != nil {
		apperror.Write(c, err)
		return
	}
	query, err := parseEventListQuery(c)
	if err != nil {
		apperror.Write(c, err)
		return
	}
	list, err := h.service.ListRequestEvents(c.Request.Context(), id, query)
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": list.Items, "totalCount": list.Total, "page": list.Page, "pageSize": list.PageSize})
}

func (h *Handler) ListDeliveryEvents(c *gin.Context) {
	id, err := parseEntityID(c.Param("id"), "delivery")
	if err != nil {
		apperror.Write(c, err)
		return
	}
	query, err := parseEventListQuery(c)
	if err != nil {
		apperror.Write(c, err)
		return
	}
	list, err := h.service.ListDeliveryEvents(c.Request.Context(), id, query)
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": list.Items, "totalCount": list.Total, "page": list.Page, "pageSize": list.PageSize})
}

func (h *Handler) ListJobEvents(c *gin.Context) {
	id, err := parseEntityID(c.Param("id"), "job")
	if err != nil {
		apperror.Write(c, err)
		return
	}
	query, err := parseEventListQuery(c)
	if err != nil {
		apperror.Write(c, err)
		return
	}
	list, err := h.service.ListJobEvents(c.Request.Context(), id, query)
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": list.Items, "totalCount": list.Total, "page": list.Page, "pageSize": list.PageSize})
}

func (h *Handler) Trace(c *gin.Context) {
	trace, err := h.service.TraceByCorrelationID(c.Request.Context(), strings.TrimSpace(c.Query("correlationId")))
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, trace)
}

func parseEntityID(raw string, label string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"invalid " + label + " id"}})
	}
	return id, nil
}

func parseEventListQuery(c *gin.Context) (EventListQuery, error) {
	page, err := listquery.ParseInt(c.Query("page"), 1, 1, 100000, "page")
	if err != nil {
		return EventListQuery{}, apperror.ValidationWithDetails("validation failed", map[string]any{"page": []string{err.Error()}})
	}
	pageSize, err := listquery.ParseInt(c.Query("pageSize"), 25, 1, 200, "pageSize")
	if err != nil {
		return EventListQuery{}, apperror.ValidationWithDetails("validation failed", map[string]any{"pageSize": []string{err.Error()}})
	}
	sortField, sortOrder, err := listquery.ParseSort(c.Query("sort"))
	if err != nil {
		return EventListQuery{}, apperror.ValidationWithDetails("validation failed", map[string]any{"sort": []string{err.Error()}})
	}
	if sortField != "" && sortField != "createdAt" {
		return EventListQuery{}, apperror.ValidationWithDetails("validation failed", map[string]any{"sort": []string{"only createdAt sorting is supported"}})
	}

	var from *time.Time
	if value := strings.TrimSpace(c.Query("from")); value != "" {
		parsed, parseErr := time.Parse(time.RFC3339, value)
		if parseErr != nil {
			return EventListQuery{}, apperror.ValidationWithDetails("validation failed", map[string]any{"from": []string{"must be RFC3339"}})
		}
		from = &parsed
	}
	var to *time.Time
	if value := strings.TrimSpace(c.Query("to")); value != "" {
		parsed, parseErr := time.Parse(time.RFC3339, value)
		if parseErr != nil {
			return EventListQuery{}, apperror.ValidationWithDetails("validation failed", map[string]any{"to": []string{"must be RFC3339"}})
		}
		to = &parsed
	}

	requestID, err := parseOptionalInt64(c.Query("requestId"), "requestId")
	if err != nil {
		return EventListQuery{}, err
	}
	deliveryID, err := parseOptionalInt64(c.Query("deliveryId"), "deliveryId")
	if err != nil {
		return EventListQuery{}, err
	}
	jobID, err := parseOptionalInt64(c.Query("jobId"), "jobId")
	if err != nil {
		return EventListQuery{}, err
	}
	workerID, err := parseOptionalInt64(c.Query("workerId"), "workerId")
	if err != nil {
		return EventListQuery{}, err
	}

	return EventListQuery{
		Page:              page,
		PageSize:          pageSize,
		RequestID:         requestID,
		DeliveryAttemptID: deliveryID,
		AsyncTaskID:       jobID,
		WorkerRunID:       workerID,
		CorrelationID:     strings.TrimSpace(c.Query("correlationId")),
		EventType:         strings.TrimSpace(c.Query("eventType")),
		Level:             strings.TrimSpace(c.Query("level")),
		From:              from,
		To:                to,
		SortOrder:         sortOrder,
	}, nil
}

func parseOptionalInt64(raw string, field string) (*int64, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil, apperror.ValidationWithDetails("validation failed", map[string]any{field: []string{"must be an integer"}})
	}
	return &parsed, nil
}
