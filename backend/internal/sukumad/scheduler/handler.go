package scheduler

import (
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

type upsertRequest struct {
	Code                string         `json:"code"`
	Name                string         `json:"name"`
	Description         string         `json:"description"`
	JobCategory         string         `json:"jobCategory"`
	JobType             string         `json:"jobType"`
	ScheduleType        string         `json:"scheduleType"`
	ScheduleExpr        string         `json:"scheduleExpr"`
	Timezone            string         `json:"timezone"`
	Enabled             bool           `json:"enabled"`
	AllowConcurrentRuns bool           `json:"allowConcurrentRuns"`
	Config              map[string]any `json:"config"`
}

func (h *Handler) ListJobs(c *gin.Context) {
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

	result, err := h.service.ListScheduledJobs(c.Request.Context(), ListQuery{
		Page:      page,
		PageSize:  pageSize,
		SortField: sortField,
		SortOrder: sortOrder,
		Filter:    strings.TrimSpace(c.Query("q")),
		Category:  strings.ToLower(strings.TrimSpace(c.Query("category"))),
	})
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetJob(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"invalid scheduled job id"}}))
		return
	}

	record, err := h.service.GetScheduledJob(c.Request.Context(), id)
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, record)
}

func (h *Handler) CreateJob(c *gin.Context) {
	principal, ok := principalFromContext(c)
	if !ok {
		apperror.Write(c, apperror.Unauthorized("Unauthorized"))
		return
	}

	var req upsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"body": []string{"invalid JSON payload"}}))
		return
	}

	record, err := h.service.CreateScheduledJob(c.Request.Context(), CreateInput{
		Code:                req.Code,
		Name:                req.Name,
		Description:         req.Description,
		JobCategory:         req.JobCategory,
		JobType:             req.JobType,
		ScheduleType:        req.ScheduleType,
		ScheduleExpr:        req.ScheduleExpr,
		Timezone:            req.Timezone,
		Enabled:             req.Enabled,
		AllowConcurrentRuns: req.AllowConcurrentRuns,
		Config:              req.Config,
		ActorID:             actorUserID(principal),
	})
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, record)
}

func (h *Handler) UpdateJob(c *gin.Context) {
	principal, ok := principalFromContext(c)
	if !ok {
		apperror.Write(c, apperror.Unauthorized("Unauthorized"))
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"invalid scheduled job id"}}))
		return
	}

	var req upsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"body": []string{"invalid JSON payload"}}))
		return
	}

	record, err := h.service.UpdateScheduledJob(c.Request.Context(), UpdateInput{
		ID:                  id,
		Code:                req.Code,
		Name:                req.Name,
		Description:         req.Description,
		JobCategory:         req.JobCategory,
		JobType:             req.JobType,
		ScheduleType:        req.ScheduleType,
		ScheduleExpr:        req.ScheduleExpr,
		Timezone:            req.Timezone,
		Enabled:             req.Enabled,
		AllowConcurrentRuns: req.AllowConcurrentRuns,
		Config:              req.Config,
		ActorID:             actorUserID(principal),
	})
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, record)
}

func (h *Handler) EnableJob(c *gin.Context) {
	h.setEnabled(c, true)
}

func (h *Handler) DisableJob(c *gin.Context) {
	h.setEnabled(c, false)
}

func (h *Handler) setEnabled(c *gin.Context, enabled bool) {
	principal, ok := principalFromContext(c)
	if !ok {
		apperror.Write(c, apperror.Unauthorized("Unauthorized"))
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"invalid scheduled job id"}}))
		return
	}

	record, err := h.service.SetScheduledJobEnabled(c.Request.Context(), actorUserID(principal), id, enabled)
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, record)
}

func (h *Handler) RunNow(c *gin.Context) {
	principal, ok := principalFromContext(c)
	if !ok {
		apperror.Write(c, apperror.Unauthorized("Unauthorized"))
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"invalid scheduled job id"}}))
		return
	}

	record, err := h.service.RunNow(c.Request.Context(), actorUserID(principal), id)
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, record)
}

func (h *Handler) ListRuns(c *gin.Context) {
	jobID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"invalid scheduled job id"}}))
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
	sortField, sortOrder, err := listquery.ParseSort(c.Query("sort"))
	if err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"sort": []string{err.Error()}}))
		return
	}

	result, err := h.service.ListJobRuns(c.Request.Context(), jobID, RunListQuery{
		Page:      page,
		PageSize:  pageSize,
		SortField: sortField,
		SortOrder: sortOrder,
		Status:    strings.ToLower(strings.TrimSpace(c.Query("status"))),
	})
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetRun(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"invalid scheduled job run id"}}))
		return
	}

	record, err := h.service.GetRun(c.Request.Context(), id)
	if err != nil {
		apperror.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, record)
}

func principalFromContext(c *gin.Context) (auth.Principal, bool) {
	value, ok := c.Get(auth.PrincipalContextKey)
	if !ok {
		return auth.Principal{}, false
	}
	principal, ok := value.(auth.Principal)
	if !ok {
		return auth.Principal{}, false
	}
	return principal, true
}

func actorUserID(principal auth.Principal) *int64 {
	userID, ok := principal.EffectiveUserID()
	if !ok {
		return nil
	}
	return &userID
}
