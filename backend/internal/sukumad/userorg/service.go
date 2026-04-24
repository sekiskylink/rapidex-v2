package userorg

import (
	"context"
	"strings"
)

type roleLookup interface {
	RoleNamesForUser(context.Context, int64) ([]string, error)
}

// Service wraps the repository with higher‑level functionality.  It may be
// extended to compute descendant scopes or enforce role‑based rules.
type Service struct {
	repo       Repository
	roleLookup roleLookup
}

// NewService constructs a Service from the given repository.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) WithRoleLookup(lookup roleLookup) *Service {
	if s == nil {
		return s
	}
	s.roleLookup = lookup
	return s
}

// GetUserOrgUnitIDs returns the list of org unit IDs assigned to the given user.
func (s *Service) GetUserOrgUnitIDs(ctx context.Context, userID int64) ([]int64, error) {
	return s.repo.ListByUser(ctx, userID)
}

func (s *Service) GetUserAssignments(ctx context.Context, userID int64) ([]AssignmentDetail, error) {
	return s.repo.ListAssignmentsByUser(ctx, userID)
}

func (s *Service) ResolveScope(ctx context.Context, userID int64) (Scope, error) {
	if s == nil {
		return Scope{}, nil
	}
	if s.roleLookup != nil {
		roles, err := s.roleLookup.RoleNamesForUser(ctx, userID)
		if err != nil {
			return Scope{}, err
		}
		for _, role := range roles {
			if strings.EqualFold(strings.TrimSpace(role), "admin") {
				return Scope{Restricted: false}, nil
			}
		}
	}

	assignments, err := s.repo.ListAssignmentsByUser(ctx, userID)
	if err != nil {
		return Scope{}, err
	}
	scope := Scope{
		Restricted:         true,
		AssignedOrgUnitIDs: make([]int64, 0, len(assignments)),
		PathPrefixes:       make([]string, 0, len(assignments)),
	}
	for _, assignment := range assignments {
		scope.AssignedOrgUnitIDs = append(scope.AssignedOrgUnitIDs, assignment.OrgUnitID)
		trimmed := strings.TrimSpace(assignment.OrgUnitPath)
		if trimmed != "" {
			scope.PathPrefixes = append(scope.PathPrefixes, trimmed)
		}
	}
	return scope, nil
}

func ScopeContainsPath(scope Scope, path string) bool {
	if !scope.Restricted {
		return true
	}
	normalizedPath := normalizeScopePath(path)
	if normalizedPath == "" || len(scope.PathPrefixes) == 0 {
		return false
	}
	for _, prefix := range scope.PathPrefixes {
		if pathHasPrefix(normalizedPath, prefix) {
			return true
		}
	}
	return false
}

func pathHasPrefix(path string, prefix string) bool {
	normalizedPath := normalizeScopePath(path)
	normalizedPrefix := normalizeScopePath(prefix)
	if normalizedPath == "" || normalizedPrefix == "" {
		return false
	}
	return normalizedPath == normalizedPrefix || strings.HasPrefix(normalizedPath, normalizedPrefix+"/")
}

func normalizeScopePath(path string) string {
	trimmed := strings.Trim(strings.TrimSpace(path), "/")
	if trimmed == "" {
		return ""
	}
	return "/" + trimmed
}

// Assign assigns a user to an organisation unit.
func (s *Service) Assign(ctx context.Context, userID, orgUnitID int64) error {
	return s.repo.Assign(ctx, userID, orgUnitID)
}

// Remove removes an assignment between a user and an organisation unit.
func (s *Service) Remove(ctx context.Context, userID, orgUnitID int64) error {
	return s.repo.Remove(ctx, userID, orgUnitID)
}
