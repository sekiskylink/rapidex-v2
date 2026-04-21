package userorg

import "context"

// Service wraps the repository with higher‑level functionality.  It may be
// extended to compute descendant scopes or enforce role‑based rules.
type Service struct {
    repo Repository
}

// NewService constructs a Service from the given repository.
func NewService(repo Repository) *Service {
    return &Service{repo: repo}
}

// GetUserOrgUnitIDs returns the list of org unit IDs assigned to the given user.
func (s *Service) GetUserOrgUnitIDs(ctx context.Context, userID int64) ([]int64, error) {
    return s.repo.ListByUser(ctx, userID)
}

// Assign assigns a user to an organisation unit.
func (s *Service) Assign(ctx context.Context, userID, orgUnitID int64) error {
    return s.repo.Assign(ctx, userID, orgUnitID)
}

// Remove removes an assignment between a user and an organisation unit.
func (s *Service) Remove(ctx context.Context, userID, orgUnitID int64) error {
    return s.repo.Remove(ctx, userID, orgUnitID)
}