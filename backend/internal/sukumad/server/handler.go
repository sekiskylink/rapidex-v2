package server

import (
	"net/http"
	"strconv"

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

type upsertServerRequest struct {
	Name                    string            `json:"name"`
	Code                    string            `json:"code"`
	SystemType              string            `json:"systemType"`
	BaseURL                 string            `json:"baseUrl"`
	EndpointType            string            `json:"endpointType"`
	HTTPMethod              string            `json:"httpMethod"`
	UseAsync                bool              `json:"useAsync"`
	ParseResponses          bool              `json:"parseResponses"`
	ResponseBodyPersistence string            `json:"responseBodyPersistence"`
	Headers                 map[string]string `json:"headers"`
	URLParams               map[string]string `json:"urlParams"`
	Suspended               bool              `json:"suspended"`
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
	search := listquery.ResolveSearch(c.Query("q"), filterField, filterValue, map[string]struct{}{
		"name":       {},
		"code":       {},
		"systemType": {},
		"baseUrl":    {},
	})

	list, err := h.service.ListServers(c.Request.Context(), ListQuery{
		Page:      page,
		PageSize:  pageSize,
		SortField: sortField,
		SortOrder: sortOrder,
		Filter:    search,
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
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"invalid server id"}}))
		return
	}

	item, err := h.service.GetServer(c.Request.Context(), id)
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

	var req upsertServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"body": []string{"invalid JSON payload"}}))
		return
	}

	created, err := h.service.CreateServer(c.Request.Context(), CreateInput{
		Name:                    req.Name,
		Code:                    req.Code,
		SystemType:              req.SystemType,
		BaseURL:                 req.BaseURL,
		EndpointType:            req.EndpointType,
		HTTPMethod:              req.HTTPMethod,
		UseAsync:                req.UseAsync,
		ParseResponses:          req.ParseResponses,
		ResponseBodyPersistence: req.ResponseBodyPersistence,
		Headers:                 req.Headers,
		URLParams:               req.URLParams,
		Suspended:               req.Suspended,
		ActorID:                 actorUserID(principal),
	})
	if err != nil {
		apperror.Write(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

func (h *Handler) Update(c *gin.Context) {
	principal, ok := principalFromContext(c)
	if !ok {
		apperror.Write(c, apperror.Unauthorized("Unauthorized"))
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"invalid server id"}}))
		return
	}

	var req upsertServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"body": []string{"invalid JSON payload"}}))
		return
	}

	updated, err := h.service.UpdateServer(c.Request.Context(), UpdateInput{
		ID:                      id,
		Name:                    req.Name,
		Code:                    req.Code,
		SystemType:              req.SystemType,
		BaseURL:                 req.BaseURL,
		EndpointType:            req.EndpointType,
		HTTPMethod:              req.HTTPMethod,
		UseAsync:                req.UseAsync,
		ParseResponses:          req.ParseResponses,
		ResponseBodyPersistence: req.ResponseBodyPersistence,
		Headers:                 req.Headers,
		URLParams:               req.URLParams,
		Suspended:               req.Suspended,
		ActorID:                 actorUserID(principal),
	})
	if err != nil {
		apperror.Write(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

func (h *Handler) Delete(c *gin.Context) {
	principal, ok := principalFromContext(c)
	if !ok {
		apperror.Write(c, apperror.Unauthorized("Unauthorized"))
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apperror.Write(c, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"invalid server id"}}))
		return
	}

	if err := h.service.DeleteServer(c.Request.Context(), actorUserID(principal), id); err != nil {
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
	if principal.Type != "user" {
		return nil
	}
	return &principal.UserID
}
