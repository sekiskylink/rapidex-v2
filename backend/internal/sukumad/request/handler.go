package request

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

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

type createRequestRequest struct {
	SourceSystem        string          `json:"sourceSystem"`
	DestinationServerID int64           `json:"destinationServerId"`
	BatchID             string          `json:"batchId"`
	CorrelationID       string          `json:"correlationId"`
	IdempotencyKey      string          `json:"idempotencyKey"`
	Payload             json.RawMessage `json:"payload"`
	URLSuffix           string          `json:"urlSuffix"`
	Metadata            map[string]any  `json:"metadata"`
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

	status := ""
	if filterField == "status" {
		status = strings.ToLower(strings.TrimSpace(filterValue))
		if !isValidStatus(status) {
			apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"status": []string{"must be one of pending, processing, completed, failed"}}))
			return
		}
	}

	search := listquery.ResolveSearch(c.Query("q"), filterField, filterValue, map[string]struct{}{
		"uid":                   {},
		"sourceSystem":          {},
		"destinationServerName": {},
		"correlationId":         {},
		"batchId":               {},
	})

	list, err := h.service.ListRequests(c.Request.Context(), ListQuery{
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
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"invalid request id"}}))
		return
	}

	item, err := h.service.GetRequest(c.Request.Context(), id)
	if err != nil {
		apperror.Write(c, err)
		return
	}

	c.JSON(http.StatusOK, item)
}

func (h *Handler) Create(c *gin.Context) {
	principal, ok := principalFromContext(c)
	if !ok {
		apperror.Write(c, apperror.Unauthorized("Unauthorized"))
		return
	}

	var req createRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"body": []string{"invalid JSON payload"}}))
		return
	}

	created, err := h.service.CreateRequest(c.Request.Context(), CreateInput{
		SourceSystem:        req.SourceSystem,
		DestinationServerID: req.DestinationServerID,
		BatchID:             req.BatchID,
		CorrelationID:       req.CorrelationID,
		IdempotencyKey:      req.IdempotencyKey,
		Payload:             req.Payload,
		URLSuffix:           req.URLSuffix,
		Extras:              req.Metadata,
		ActorID:             actorUserID(principal),
	})
	if err != nil {
		apperror.Write(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
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

func isValidStatus(value string) bool {
	switch value {
	case StatusPending, StatusProcessing, StatusCompleted, StatusFailed:
		return true
	default:
		return false
	}
}
