package ratelimit

import (
	"context"

	"basepro/backend/internal/apperror"
)

type Service struct {
	repo Repository
}

func NewService(repository Repository) *Service {
	return &Service{repo: repository}
}

func (s *Service) ListPolicies(ctx context.Context, query ListQuery) (ListResult, error) {
	return s.repo.ListPolicies(ctx, query)
}

func (s *Service) GetPolicy(ctx context.Context, id int64) (Policy, error) {
	item, err := s.repo.GetPolicyByID(ctx, id)
	if err != nil {
		return Policy{}, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"rate limit policy not found"}})
	}
	return item, nil
}

func (s *Service) CreatePolicy(ctx context.Context, params CreateParams) (Policy, error) {
	if params.Name == "" {
		return Policy{}, apperror.ValidationWithDetails("validation failed", map[string]any{"name": []string{"is required"}})
	}
	if params.ScopeType == "" {
		return Policy{}, apperror.ValidationWithDetails("validation failed", map[string]any{"scopeType": []string{"is required"}})
	}
	if params.RPS <= 0 || params.Burst <= 0 || params.MaxConcurrency <= 0 || params.TimeoutMS <= 0 {
		return Policy{}, apperror.ValidationWithDetails("validation failed", map[string]any{
			"rps":            []string{"must be > 0"},
			"burst":          []string{"must be > 0"},
			"maxConcurrency": []string{"must be > 0"},
			"timeoutMs":      []string{"must be > 0"},
		})
	}
	if params.UID == "" {
		params.UID = newUID()
	}
	return s.repo.CreatePolicy(ctx, params)
}

func (s *Service) ResolveActivePolicy(ctx context.Context, scopeType string, scopeRef string) (Policy, bool, error) {
	return s.repo.FindActivePolicy(ctx, scopeType, scopeRef)
}
