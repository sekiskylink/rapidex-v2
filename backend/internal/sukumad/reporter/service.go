package reporter

import (
	"context"
	"errors"
	"strings"
	"time"
)

// Service encapsulates business logic for reporters and depends on a Repository.
type Service struct {
	repo Repository
}

// NewService constructs a new Service with the provided repository.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// List returns a page of reporters matching the provided query.
func (s *Service) List(ctx context.Context, query ListQuery) (ListResult, error) {
	return s.repo.List(ctx, query)
}

// Get fetches a reporter by ID.
func (s *Service) Get(ctx context.Context, id int64) (Reporter, error) {
	return s.repo.GetByID(ctx, id)
}

// Create validates and persists a new Reporter.
func (s *Service) Create(ctx context.Context, r Reporter) (Reporter, error) {
	if strings.TrimSpace(r.Name) == "" {
		return Reporter{}, errors.New("name is required")
	}
	if strings.TrimSpace(r.Telephone) == "" {
		return Reporter{}, errors.New("telephone is required")
	}
	if r.OrgUnitID == 0 {
		return Reporter{}, errors.New("org unit is required")
	}
	now := time.Now().UTC()
	r.CreatedAt = now
	r.UpdatedAt = now
	r.Groups = normalizeGroups(r.Groups)
	return s.repo.Create(ctx, r)
}

// Update validates and updates an existing reporter.
func (s *Service) Update(ctx context.Context, r Reporter) (Reporter, error) {
	if r.ID == 0 {
		return Reporter{}, errors.New("id is required for update")
	}
	if strings.TrimSpace(r.Name) == "" {
		return Reporter{}, errors.New("name is required")
	}
	if strings.TrimSpace(r.Telephone) == "" {
		return Reporter{}, errors.New("telephone is required")
	}
	if r.OrgUnitID == 0 {
		return Reporter{}, errors.New("org unit is required")
	}
	r.Groups = normalizeGroups(r.Groups)
	r.UpdatedAt = time.Now().UTC()
	return s.repo.Update(ctx, r)
}

// Delete removes a reporter by ID.
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func normalizeGroups(groups []string) []string {
	if len(groups) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(groups))
	normalized := make([]string, 0, len(groups))
	for _, group := range groups {
		value := strings.TrimSpace(group)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, value)
	}
	return normalized
}
