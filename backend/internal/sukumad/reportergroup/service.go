package reportergroup

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/settings"
	"basepro/backend/internal/sukumad/rapidex/rapidpro"
	sukumadserver "basepro/backend/internal/sukumad/server"
)

const defaultRapidProServerCode = "rapidpro"

type rapidProServerLookup interface {
	GetServerByCode(context.Context, string) (sukumadserver.Record, error)
}

type rapidProClient interface {
	LookupGroupByName(context.Context, rapidpro.Connection, string) (rapidpro.Group, bool, error)
	CreateGroup(context.Context, rapidpro.Connection, string) (rapidpro.Group, error)
}

type rapidProReporterSyncSettings interface {
	GetRapidProReporterSync(context.Context) (settings.RapidProReporterSyncSettings, error)
}

type Service struct {
	repo             Repository
	serverLookup     rapidProServerLookup
	rapidProClient   rapidProClient
	rapidProSettings rapidProReporterSyncSettings
	clock            func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		clock: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *Service) WithRapidProIntegration(serverLookup rapidProServerLookup, client rapidProClient) *Service {
	if s == nil {
		return s
	}
	s.serverLookup = serverLookup
	s.rapidProClient = client
	return s
}

func (s *Service) WithRapidProSettings(provider rapidProReporterSyncSettings) *Service {
	if s == nil {
		return s
	}
	s.rapidProSettings = provider
	return s
}

func (s *Service) List(ctx context.Context, query ListQuery) (ListResult, error) {
	return s.repo.List(ctx, query)
}

func (s *Service) ListOptions(ctx context.Context) ([]Option, error) {
	result, err := s.repo.List(ctx, ListQuery{Page: 0, PageSize: 500, ActiveOnly: true})
	if err != nil {
		return nil, err
	}
	options := make([]Option, 0, len(result.Items))
	for _, item := range result.Items {
		options = append(options, Option{ID: item.ID, Name: item.Name})
	}
	return options, nil
}

func (s *Service) Create(ctx context.Context, item ReporterGroup) (ReporterGroup, error) {
	item.Name = strings.TrimSpace(item.Name)
	if item.Name == "" {
		return ReporterGroup{}, apperror.ValidationWithDetails("validation failed", map[string]any{"name": []string{"is required"}})
	}
	if existing, err := s.repo.GetByName(ctx, item.Name); err == nil && existing.ID != 0 {
		return ReporterGroup{}, apperror.ValidationWithDetails("validation failed", map[string]any{"name": []string{"must be unique"}})
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ReporterGroup{}, err
	}
	if item.IsActive {
		if _, err := s.ensureRapidProGroup(ctx, item.Name); err != nil {
			return ReporterGroup{}, err
		}
	}
	now := s.clock()
	item.CreatedAt = now
	item.UpdatedAt = now
	return s.repo.Create(ctx, item)
}

func (s *Service) Update(ctx context.Context, item ReporterGroup) (ReporterGroup, error) {
	if item.ID == 0 {
		return ReporterGroup{}, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"is required"}})
	}
	item.Name = strings.TrimSpace(item.Name)
	if item.Name == "" {
		return ReporterGroup{}, apperror.ValidationWithDetails("validation failed", map[string]any{"name": []string{"is required"}})
	}
	existing, err := s.repo.GetByID(ctx, item.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ReporterGroup{}, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"reporter group not found"}})
		}
		return ReporterGroup{}, err
	}
	if byName, err := s.repo.GetByName(ctx, item.Name); err == nil && byName.ID != existing.ID {
		return ReporterGroup{}, apperror.ValidationWithDetails("validation failed", map[string]any{"name": []string{"must be unique"}})
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ReporterGroup{}, err
	}
	if item.IsActive {
		if _, err := s.ensureRapidProGroup(ctx, item.Name); err != nil {
			return ReporterGroup{}, err
		}
	}
	item.CreatedAt = existing.CreatedAt
	item.UpdatedAt = s.clock()
	return s.repo.Update(ctx, item)
}

func (s *Service) ValidateActiveNames(ctx context.Context, names []string) ([]string, error) {
	normalized := normalizeNames(names)
	if len(normalized) == 0 {
		return []string{}, nil
	}
	items, err := s.repo.ListByNames(ctx, normalized, true)
	if err != nil {
		return nil, err
	}
	available := make(map[string]string, len(items))
	for _, item := range items {
		available[strings.ToLower(strings.TrimSpace(item.Name))] = item.Name
	}
	missing := make([]string, 0)
	result := make([]string, 0, len(normalized))
	for _, name := range normalized {
		if canonical, ok := available[strings.ToLower(name)]; ok {
			result = append(result, canonical)
			continue
		}
		missing = append(missing, name)
	}
	if len(missing) > 0 {
		return nil, apperror.ValidationWithDetails("validation failed", map[string]any{
			"groups": []string{fmt.Sprintf("Unknown or inactive reporter groups: %s", strings.Join(missing, ", "))},
		})
	}
	return result, nil
}

func (s *Service) EnsureRapidProGroups(ctx context.Context, names []string) ([]rapidpro.Group, error) {
	validated, err := s.ValidateActiveNames(ctx, names)
	if err != nil {
		return nil, err
	}
	resolved := make([]rapidpro.Group, 0, len(validated))
	for _, name := range validated {
		group, ensureErr := s.ensureRapidProGroup(ctx, name)
		if ensureErr != nil {
			return nil, ensureErr
		}
		resolved = append(resolved, group)
	}
	return resolved, nil
}

func (s *Service) ensureRapidProGroup(ctx context.Context, name string) (rapidpro.Group, error) {
	conn, err := s.rapidProConnection(ctx)
	if err != nil {
		return rapidpro.Group{}, err
	}
	match, found, err := s.rapidProClient.LookupGroupByName(ctx, conn, name)
	if err != nil {
		return rapidpro.Group{}, mapRapidProRequestError(err)
	}
	if found {
		return match, nil
	}
	created, err := s.rapidProClient.CreateGroup(ctx, conn, name)
	if err != nil {
		return rapidpro.Group{}, mapRapidProRequestError(err)
	}
	return created, nil
}

func (s *Service) rapidProConnection(ctx context.Context) (rapidpro.Connection, error) {
	if s.serverLookup == nil || s.rapidProClient == nil {
		return rapidpro.Connection{}, apperror.ValidationWithDetails("validation failed", map[string]any{"rapidpro": []string{"RapidPro integration is not configured"}})
	}
	serverCode := defaultRapidProServerCode
	if s.rapidProSettings != nil {
		config, err := s.rapidProSettings.GetRapidProReporterSync(ctx)
		if err != nil {
			return rapidpro.Connection{}, err
		}
		if trimmed := strings.TrimSpace(config.RapidProServerCode); trimmed != "" {
			serverCode = trimmed
		}
	}
	record, err := s.serverLookup.GetServerByCode(ctx, serverCode)
	if err != nil {
		return rapidpro.Connection{}, err
	}
	if record.Suspended {
		return rapidpro.Connection{}, apperror.ValidationWithDetails("validation failed", map[string]any{
			"rapidpro":           []string{"RapidPro server is suspended"},
			"rapidProServerCode": []string{serverCode},
		})
	}
	if strings.TrimSpace(record.BaseURL) == "" {
		return rapidpro.Connection{}, apperror.ValidationWithDetails("validation failed", map[string]any{
			"rapidpro":           []string{"RapidPro server base URL is required"},
			"rapidProServerCode": []string{serverCode},
		})
	}
	return rapidpro.Connection{BaseURL: record.BaseURL, Headers: record.Headers}, nil
}

func mapRapidProRequestError(err error) error {
	var requestErr *rapidpro.RequestError
	if errors.As(err, &requestErr) {
		detail := fmt.Sprintf("RapidPro rejected the request (status %d)", requestErr.StatusCode)
		if body := strings.TrimSpace(requestErr.Body); body != "" {
			detail = fmt.Sprintf("%s: %s", detail, body)
		}
		return apperror.ValidationWithDetails("validation failed", map[string]any{"rapidpro": []string{detail}})
	}
	return err
}

func normalizeNames(names []string) []string {
	if len(names) == 0 {
		return []string{}
	}
	seen := map[string]struct{}{}
	result := make([]string, 0, len(names))
	for _, name := range names {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}
