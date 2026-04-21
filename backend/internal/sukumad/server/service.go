package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/audit"
	"github.com/jackc/pgx/v5/pgconn"
)

var codePattern = regexp.MustCompile(`^[a-z0-9._-]+$`)

type Service struct {
	repo         Repository
	auditService *audit.Service
}

func NewService(repository Repository, auditService ...*audit.Service) *Service {
	var auditSvc *audit.Service
	if len(auditService) > 0 {
		auditSvc = auditService[0]
	}
	return &Service{repo: repository, auditService: auditSvc}
}

func (s *Service) ListServers(ctx context.Context, query ListQuery) (ListResult, error) {
	return s.repo.ListServers(ctx, query)
}

func (s *Service) GetServer(ctx context.Context, id int64) (Record, error) {
	record, err := s.repo.GetServerByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"server not found"}})
		}
		return Record{}, err
	}
	return record, nil
}

func (s *Service) GetServerByUID(ctx context.Context, uid string) (Record, error) {
	record, err := s.repo.GetServerByUID(ctx, uid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, apperror.ValidationWithDetails("validation failed", map[string]any{"destinationServerUid": []string{"server not found"}})
		}
		return Record{}, err
	}
	return record, nil
}

func (s *Service) GetServerByCode(ctx context.Context, code string) (Record, error) {
	record, err := s.repo.GetServerByCode(ctx, code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, apperror.ValidationWithDetails("validation failed", map[string]any{"code": []string{"server not found"}})
		}
		return Record{}, err
	}
	return record, nil
}

func (s *Service) CreateServer(ctx context.Context, input CreateInput) (Record, error) {
	normalized, details := normalizeInput(input)
	if len(details) > 0 {
		return Record{}, apperror.ValidationWithDetails("validation failed", details)
	}

	created, err := s.repo.CreateServer(ctx, CreateParams{
		UID:                     newUID(),
		Name:                    normalized.Name,
		Code:                    normalized.Code,
		SystemType:              normalized.SystemType,
		BaseURL:                 normalized.BaseURL,
		EndpointType:            normalized.EndpointType,
		HTTPMethod:              normalized.HTTPMethod,
		UseAsync:                normalized.UseAsync,
		ParseResponses:          normalized.ParseResponses,
		ResponseBodyPersistence: normalized.ResponseBodyPersistence,
		Headers:                 normalized.Headers,
		URLParams:               normalized.URLParams,
		Suspended:               normalized.Suspended,
		CreatedBy:               input.ActorID,
	})
	if err != nil {
		if mapped := mapConstraintError(err); mapped != nil {
			return Record{}, mapped
		}
		return Record{}, err
	}

	s.logAudit(ctx, audit.Event{
		Action:      "server.created",
		ActorUserID: input.ActorID,
		EntityType:  "server",
		EntityID:    strPtr(fmt.Sprintf("%d", created.ID)),
		Metadata: map[string]any{
			"name":                    created.Name,
			"code":                    created.Code,
			"systemType":              created.SystemType,
			"baseUrl":                 created.BaseURL,
			"suspended":               created.Suspended,
			"responseBodyPersistence": created.ResponseBodyPersistence,
		},
	})

	return created, nil
}

func (s *Service) UpdateServer(ctx context.Context, input UpdateInput) (Record, error) {
	existing, err := s.repo.GetServerByID(ctx, input.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"server not found"}})
		}
		return Record{}, err
	}

	normalized, details := normalizeInput(CreateInput{
		Name:                    input.Name,
		Code:                    input.Code,
		SystemType:              input.SystemType,
		BaseURL:                 input.BaseURL,
		EndpointType:            input.EndpointType,
		HTTPMethod:              input.HTTPMethod,
		UseAsync:                input.UseAsync,
		ParseResponses:          input.ParseResponses,
		ResponseBodyPersistence: input.ResponseBodyPersistence,
		Headers:                 input.Headers,
		URLParams:               input.URLParams,
		Suspended:               input.Suspended,
	})
	if len(details) > 0 {
		return Record{}, apperror.ValidationWithDetails("validation failed", details)
	}

	updated, err := s.repo.UpdateServer(ctx, UpdateParams{
		ID:                      input.ID,
		Name:                    normalized.Name,
		Code:                    normalized.Code,
		SystemType:              normalized.SystemType,
		BaseURL:                 normalized.BaseURL,
		EndpointType:            normalized.EndpointType,
		HTTPMethod:              normalized.HTTPMethod,
		UseAsync:                normalized.UseAsync,
		ParseResponses:          normalized.ParseResponses,
		ResponseBodyPersistence: normalized.ResponseBodyPersistence,
		Headers:                 normalized.Headers,
		URLParams:               normalized.URLParams,
		Suspended:               normalized.Suspended,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"server not found"}})
		}
		if mapped := mapConstraintError(err); mapped != nil {
			return Record{}, mapped
		}
		return Record{}, err
	}

	s.logAudit(ctx, audit.Event{
		Action:      "server.updated",
		ActorUserID: input.ActorID,
		EntityType:  "server",
		EntityID:    strPtr(fmt.Sprintf("%d", updated.ID)),
		Metadata: map[string]any{
			"name":                    updated.Name,
			"code":                    updated.Code,
			"systemType":              updated.SystemType,
			"baseUrl":                 updated.BaseURL,
			"endpointType":            updated.EndpointType,
			"httpMethod":              updated.HTTPMethod,
			"useAsync":                updated.UseAsync,
			"parseResponses":          updated.ParseResponses,
			"responseBodyPersistence": updated.ResponseBodyPersistence,
			"suspended":               updated.Suspended,
		},
	})

	if existing.Suspended != updated.Suspended {
		action := "server.activated"
		if updated.Suspended {
			action = "server.suspended"
		}
		s.logAudit(ctx, audit.Event{
			Action:      action,
			ActorUserID: input.ActorID,
			EntityType:  "server",
			EntityID:    strPtr(fmt.Sprintf("%d", updated.ID)),
			Metadata: map[string]any{
				"code":      updated.Code,
				"suspended": updated.Suspended,
			},
		})
	}

	return updated, nil
}

func (s *Service) DeleteServer(ctx context.Context, actorID *int64, id int64) error {
	existing, err := s.repo.GetServerByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"server not found"}})
		}
		return err
	}
	if err := s.repo.DeleteServer(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"server not found"}})
		}
		return err
	}

	s.logAudit(ctx, audit.Event{
		Action:      "server.deleted",
		ActorUserID: actorID,
		EntityType:  "server",
		EntityID:    strPtr(fmt.Sprintf("%d", id)),
		Metadata: map[string]any{
			"name": existing.Name,
			"code": existing.Code,
		},
	})
	return nil
}

func normalizeInput(input CreateInput) (CreateInput, map[string]any) {
	normalized := CreateInput{
		Name:                    strings.TrimSpace(input.Name),
		Code:                    strings.ToLower(strings.TrimSpace(input.Code)),
		SystemType:              strings.ToLower(strings.TrimSpace(input.SystemType)),
		BaseURL:                 strings.TrimRight(strings.TrimSpace(input.BaseURL), "/"),
		EndpointType:            strings.ToLower(strings.TrimSpace(input.EndpointType)),
		HTTPMethod:              strings.ToUpper(strings.TrimSpace(input.HTTPMethod)),
		UseAsync:                input.UseAsync,
		ParseResponses:          input.ParseResponses,
		ResponseBodyPersistence: normalizeResponseBodyPersistence(input.ResponseBodyPersistence),
		Headers:                 sanitizeMap(input.Headers),
		URLParams:               sanitizeMap(input.URLParams),
		Suspended:               input.Suspended,
	}

	details := map[string]any{}
	if normalized.Name == "" {
		details["name"] = []string{"is required"}
	}
	if normalized.Code == "" {
		details["code"] = []string{"is required"}
	} else if !codePattern.MatchString(normalized.Code) {
		details["code"] = []string{"must contain only lowercase letters, numbers, dots, underscores, or hyphens"}
	}
	if normalized.SystemType == "" {
		details["systemType"] = []string{"is required"}
	}
	if normalized.EndpointType == "" {
		details["endpointType"] = []string{"is required"}
	}
	if normalized.HTTPMethod == "" {
		details["httpMethod"] = []string{"is required"}
	} else if !isSupportedHTTPMethod(normalized.HTTPMethod) {
		details["httpMethod"] = []string{"must be one of GET, POST, PUT, PATCH, DELETE"}
	}
	if normalized.BaseURL == "" {
		details["baseUrl"] = []string{"is required"}
	} else if err := validateBaseURL(normalized.BaseURL); err != nil {
		details["baseUrl"] = []string{err.Error()}
	}
	if err := validateStringMap(normalized.Headers); err != nil {
		details["headers"] = []string{err.Error()}
	}
	if err := validateStringMap(normalized.URLParams); err != nil {
		details["urlParams"] = []string{err.Error()}
	}
	if !isValidResponseBodyPersistence(normalized.ResponseBodyPersistence) {
		details["responseBodyPersistence"] = []string{"must be one of filter, save, or discard"}
	}

	return normalized, details
}

func normalizeResponseBodyPersistence(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return "filter"
	}
	return normalized
}

func isValidResponseBodyPersistence(value string) bool {
	switch value {
	case "filter", "save", "discard":
		return true
	default:
		return false
	}
}

func sanitizeMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		out[trimmedKey] = strings.TrimSpace(value)
	}
	return out
}

func validateBaseURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return errors.New("must be a valid URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("must use http or https")
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return errors.New("must include a host")
	}
	return nil
}

func validateStringMap(values map[string]string) error {
	for key := range values {
		if strings.TrimSpace(key) == "" {
			return errors.New("keys cannot be empty")
		}
	}
	return nil
}

func isSupportedHTTPMethod(method string) bool {
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE":
		return true
	default:
		return false
	}
}

func (s *Service) logAudit(ctx context.Context, event audit.Event) {
	if s.auditService == nil {
		return
	}
	_ = s.auditService.Log(ctx, event)
}

func mapConstraintError(err error) *apperror.AppError {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return nil
	}
	if pgErr.Code != "23505" {
		return nil
	}
	if strings.Contains(pgErr.ConstraintName, "code") {
		return apperror.ValidationWithDetails("validation failed", map[string]any{"code": []string{"must be unique"}})
	}
	if strings.Contains(pgErr.ConstraintName, "uid") {
		return apperror.ValidationWithDetails("validation failed", map[string]any{"uid": []string{"must be unique"}})
	}
	return apperror.ValidationWithDetails("validation failed", map[string]any{"record": []string{"already exists"}})
}

func strPtr(v string) *string {
	return &v
}
