package request

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/auth"
	"basepro/backend/internal/config"
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
	SourceSystem            string         `json:"sourceSystem"`
	DestinationServerID     int64          `json:"destinationServerId"`
	DestinationServerIDs    []int64        `json:"destinationServerIds"`
	DependencyRequestIDs    []int64        `json:"dependencyRequestIds"`
	BatchID                 string         `json:"batchId"`
	CorrelationID           string         `json:"correlationId"`
	IdempotencyKey          string         `json:"idempotencyKey"`
	Payload                 any            `json:"payload"`
	PayloadFormat           string         `json:"payloadFormat"`
	SubmissionBinding       string         `json:"submissionBinding"`
	ResponseBodyPersistence string         `json:"responseBodyPersistence"`
	URLSuffix               string         `json:"urlSuffix"`
	Metadata                map[string]any `json:"metadata"`
}

type createExternalRequestRequest struct {
	SourceSystem            string         `json:"sourceSystem"`
	DestinationServerUID    string         `json:"destinationServerUid"`
	DestinationServerUIDs   []string       `json:"destinationServerUids"`
	DependencyRequestUIDs   []string       `json:"dependencyRequestUids"`
	BatchID                 string         `json:"batchId"`
	CorrelationID           string         `json:"correlationId"`
	IdempotencyKey          string         `json:"idempotencyKey"`
	Payload                 any            `json:"payload"`
	PayloadFormat           string         `json:"payloadFormat"`
	SubmissionBinding       string         `json:"submissionBinding"`
	ResponseBodyPersistence string         `json:"responseBodyPersistence"`
	URLSuffix               string         `json:"urlSuffix"`
	Metadata                map[string]any `json:"metadata"`
}

func (h *Handler) List(c *gin.Context) {
	metadataColumns := currentMetadataColumns()
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
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"status": []string{"must be one of pending, blocked, processing, completed, failed"}}))
		return
	}

	search := listquery.ResolveSearch(c.Query("q"), filterField, filterValue, map[string]struct{}{
		"uid":                   {},
		"sourceSystem":          {},
		"destinationServerName": {},
		"correlationId":         {},
		"batchId":               {},
	})

	list, err := h.service.ListRequests(c.Request.Context(), ListQuery{
		Page:            page,
		PageSize:        pageSize,
		SortField:       sortField,
		SortOrder:       sortOrder,
		Filter:          search,
		Status:          status,
		MetadataColumns: metadataColumns,
	})
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, list)
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

func currentMetadataColumns() []MetadataColumn {
	cfg := config.Get()
	columns := make([]MetadataColumn, 0, len(cfg.Sukumad.Requests.MetadataColumns))
	for _, column := range cfg.Sukumad.Requests.MetadataColumns {
		columns = append(columns, MetadataColumn{
			Key:              column.Key,
			Label:            column.Label,
			Type:             column.Type,
			Searchable:       column.Searchable,
			VisibleByDefault: column.VisibleByDefault,
		})
	}
	return normalizeMetadataColumns(columns)
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
		SourceSystem:            req.SourceSystem,
		DestinationServerID:     req.DestinationServerID,
		DestinationServerIDs:    req.DestinationServerIDs,
		DependencyRequestIDs:    req.DependencyRequestIDs,
		BatchID:                 req.BatchID,
		CorrelationID:           req.CorrelationID,
		IdempotencyKey:          req.IdempotencyKey,
		Payload:                 req.Payload,
		PayloadFormat:           req.PayloadFormat,
		SubmissionBinding:       req.SubmissionBinding,
		ResponseBodyPersistence: req.ResponseBodyPersistence,
		URLSuffix:               req.URLSuffix,
		Extras:                  req.Metadata,
		ActorID:                 actorUserID(principal),
	})
	if err != nil {
		apperror.Write(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

func (h *Handler) CreateExternal(c *gin.Context) {
	principal, ok := principalFromContext(c)
	if !ok {
		apperror.Write(c, apperror.Unauthorized("Unauthorized"))
		return
	}

	var req createExternalRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"body": []string{"invalid JSON payload"}}))
		return
	}

	result, err := h.service.CreateExternalRequest(c.Request.Context(), ExternalCreateInput{
		SourceSystem:            req.SourceSystem,
		DestinationServerUID:    req.DestinationServerUID,
		DestinationServerUIDs:   req.DestinationServerUIDs,
		DependencyRequestUIDs:   req.DependencyRequestUIDs,
		BatchID:                 req.BatchID,
		CorrelationID:           req.CorrelationID,
		IdempotencyKey:          req.IdempotencyKey,
		Payload:                 req.Payload,
		PayloadFormat:           req.PayloadFormat,
		SubmissionBinding:       req.SubmissionBinding,
		ResponseBodyPersistence: req.ResponseBodyPersistence,
		URLSuffix:               req.URLSuffix,
		Extras:                  req.Metadata,
		ActorID:                 actorUserID(principal),
	})
	if err != nil {
		apperror.Write(c, err)
		return
	}

	statusCode := http.StatusCreated
	if result.Deduped {
		statusCode = http.StatusOK
	}
	c.JSON(statusCode, toExternalRecord(result.Record))
}

func (h *Handler) GetExternal(c *gin.Context) {
	item, err := h.service.GetRequestByUID(c.Request.Context(), strings.TrimSpace(c.Param("uid")))
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, toExternalRecord(item))
}

func (h *Handler) LookupExternal(c *gin.Context) {
	correlationID := strings.TrimSpace(c.Query("correlationId"))
	sourceSystem := strings.TrimSpace(c.Query("sourceSystem"))
	idempotencyKey := strings.TrimSpace(c.Query("idempotencyKey"))
	batchID := strings.TrimSpace(c.Query("batchId"))

	var (
		items []Record
		err   error
	)

	switch {
	case correlationID != "":
		items, err = h.service.ListRequestsByCorrelationID(c.Request.Context(), correlationID)
	case batchID != "":
		items, err = h.service.ListRequestsByBatchID(c.Request.Context(), batchID)
	case sourceSystem != "" && idempotencyKey != "":
		var item Record
		item, err = h.service.GetRequestBySourceSystemAndIdempotencyKey(c.Request.Context(), sourceSystem, idempotencyKey)
		if err == nil {
			items = []Record{item}
		}
	default:
		err = apperror.ValidationWithDetails("validation failed", map[string]any{
			"lookup": []string{"provide correlationId, batchId, or sourceSystem with idempotencyKey"},
		})
	}
	if err != nil {
		apperror.Write(c, err)
		return
	}

	response := make([]ExternalRecord, 0, len(items))
	for _, item := range items {
		response = append(response, toExternalRecord(item))
	}
	c.JSON(http.StatusOK, gin.H{"items": response, "totalCount": len(response)})
}

func (h *Handler) SummaryExternal(c *gin.Context) {
	destinationServerUID := strings.TrimSpace(c.Query("destinationServerUid"))
	startDate, endDate, details := parseExternalSummaryRange(c.Query("startDate"), c.Query("endDate"))
	if destinationServerUID == "" {
		details["destinationServerUid"] = []string{"is required"}
	}
	if len(details) > 0 {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", details))
		return
	}

	summary, err := h.service.GetExternalSummary(c.Request.Context(), destinationServerUID, startDate, endDate)
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *Handler) Delete(c *gin.Context) {
	principal, ok := principalFromContext(c)
	if !ok {
		apperror.Write(c, apperror.Unauthorized("Unauthorized"))
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"invalid request id"}}))
		return
	}

	if err := h.service.DeleteRequest(c.Request.Context(), actorUserID(principal), id); err != nil {
		apperror.Write(c, err)
		return
	}
	c.Status(http.StatusNoContent)
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

func parseExternalSummaryRange(startValue, endValue string) (time.Time, time.Time, map[string]any) {
	details := map[string]any{}

	startDate, err := time.Parse(time.DateOnly, strings.TrimSpace(startValue))
	if err != nil {
		details["startDate"] = []string{"must be a valid date in YYYY-MM-DD format"}
	}
	endDate, err := time.Parse(time.DateOnly, strings.TrimSpace(endValue))
	if err != nil {
		details["endDate"] = []string{"must be a valid date in YYYY-MM-DD format"}
	}
	if len(details) > 0 {
		return time.Time{}, time.Time{}, details
	}

	startDate = startDate.UTC()
	endDate = endDate.UTC().Add(24*time.Hour - time.Nanosecond)
	if endDate.Before(startDate) {
		details["endDate"] = []string{"must be on or after startDate"}
	}

	return startDate, endDate, details
}

func isValidStatus(value string) bool {
	switch value {
	case StatusPending, StatusBlocked, StatusProcessing, StatusCompleted, StatusFailed:
		return true
	default:
		return false
	}
}

func toExternalRecord(record Record) ExternalRecord {
	targets := make([]ExternalTarget, 0, len(record.Targets))
	for _, target := range record.Targets {
		targets = append(targets, ExternalTarget{
			UID:                   target.UID,
			DestinationServerUID:  target.ServerUID,
			DestinationServerCode: target.ServerCode,
			DestinationServerName: target.ServerName,
			TargetKind:            target.TargetKind,
			Priority:              target.Priority,
			Status:                target.Status,
			BlockedReason:         target.BlockedReason,
			DeferredUntil:         target.DeferredUntil,
			LastReleasedAt:        target.LastReleasedAt,
			LatestDelivery: ExternalDeliveryRef{
				UID:    target.LatestDeliveryUID,
				Status: target.LatestDeliveryStatus,
			},
			LatestAsyncTask: ExternalAsyncRef{
				UID:         target.LatestAsyncTaskUID,
				State:       target.LatestAsyncState,
				RemoteJobID: target.LatestAsyncRemoteJobID,
				PollURL:     target.LatestAsyncPollURL,
			},
			AwaitingAsync: target.AwaitingAsync,
		})
	}

	dependencies := make([]ExternalDependency, 0, len(record.Dependencies))
	for _, dependency := range record.Dependencies {
		dependencies = append(dependencies, ExternalDependency{
			RequestUID:            dependency.RequestUID,
			DependsOnRequestUID:   dependency.DependsOnUID,
			Status:                dependency.Status,
			StatusReason:          dependency.StatusReason,
			DeferredUntil:         dependency.DeferredUntil,
			DestinationServerName: dependency.DependsOnDestinationServerName,
		})
	}

	return ExternalRecord{
		UID:                     record.UID,
		SourceSystem:            record.SourceSystem,
		DestinationServerUID:    record.DestinationServerUID,
		DestinationServerCode:   record.DestinationServerCode,
		DestinationServerName:   record.DestinationServerName,
		BatchID:                 record.BatchID,
		CorrelationID:           record.CorrelationID,
		IdempotencyKey:          record.IdempotencyKey,
		PayloadFormat:           record.PayloadFormat,
		SubmissionBinding:       record.SubmissionBinding,
		ResponseBodyPersistence: record.ResponseBodyPersistence,
		URLSuffix:               record.URLSuffix,
		Status:                  record.Status,
		StatusReason:            record.StatusReason,
		DeferredUntil:           record.DeferredUntil,
		Metadata:                record.Extras,
		AwaitingAsync:           record.AwaitingAsync,
		Targets:                 targets,
		Dependencies:            dependencies,
		CreatedAt:               record.CreatedAt,
		UpdatedAt:               record.UpdatedAt,
	}
}
