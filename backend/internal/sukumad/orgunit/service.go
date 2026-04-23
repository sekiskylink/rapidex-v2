package orgunit

import (
	"context"
	"errors"
	"strings"
	"time"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/sukumad/userorg"
)

type orgUnitScopeResolver interface {
	ResolveScope(context.Context, int64) (userorg.Scope, error)
}

// Service encapsulates business logic for organisation units.  It depends on a Repository.
type Service struct {
	repo          Repository
	scopeResolver orgUnitScopeResolver
}

// NewService constructs a new Service from a repository implementation.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) WithScopeResolver(resolver orgUnitScopeResolver) *Service {
	if s == nil {
		return s
	}
	s.scopeResolver = resolver
	return s
}

// List returns a page of OrgUnits matching the provided query.
func (s *Service) List(ctx context.Context, query ListQuery) (ListResult, error) {
	return s.repo.List(ctx, query)
}

func (s *Service) ListForUser(ctx context.Context, userID int64, query ListQuery) (ListResult, error) {
	scopedQuery, err := s.applyScope(ctx, userID, query)
	if err != nil {
		return ListResult{}, err
	}
	return s.repo.List(ctx, scopedQuery)
}

// Get fetches an OrgUnit by its numeric ID.
func (s *Service) Get(ctx context.Context, id int64) (OrgUnit, error) {
	return s.repo.GetByID(ctx, id)
}

// Create validates and persists a new OrgUnit.  It sets timestamps and generates a UID if
// necessary.  Code must be unique across all units; this check is delegated to the repository.
func (s *Service) Create(ctx context.Context, unit OrgUnit) (OrgUnit, error) {
	if strings.TrimSpace(unit.Code) == "" || strings.TrimSpace(unit.Name) == "" {
		return OrgUnit{}, errors.New("code and name are required")
	}
	uid, err := ensureDHIS2UID(unit.UID)
	if err != nil {
		return OrgUnit{}, err
	}
	unit.UID = uid
	if strings.TrimSpace(unit.ShortName) == "" {
		unit.ShortName = unit.Name
	}
	if unit.Extras == nil {
		unit.Extras = JSONMap{}
	}
	if unit.AttributeValues == nil {
		unit.AttributeValues = JSONMap{}
	}
	now := time.Now().UTC()
	unit.CreatedAt = now
	unit.UpdatedAt = now
	return s.repo.Create(ctx, unit)
}

func (s *Service) CreateForUser(ctx context.Context, userID int64, unit OrgUnit) (OrgUnit, error) {
	scope, err := s.resolveScope(ctx, userID)
	if err != nil {
		return OrgUnit{}, err
	}
	if scope.Restricted {
		if unit.ParentID == nil {
			return OrgUnit{}, apperror.Forbidden("Organisation unit is outside your assigned jurisdiction")
		}
		parent, err := s.repo.GetByID(ctx, *unit.ParentID)
		if err != nil {
			return OrgUnit{}, err
		}
		if !userorg.ScopeContainsPath(scope, parent.Path) {
			return OrgUnit{}, apperror.Forbidden("Organisation unit is outside your assigned jurisdiction")
		}
	}
	return s.Create(ctx, unit)
}

// Update validates and persists modifications to an existing OrgUnit.
func (s *Service) Update(ctx context.Context, unit OrgUnit) (OrgUnit, error) {
	if unit.ID == 0 {
		return OrgUnit{}, errors.New("id is required for update")
	}
	if strings.TrimSpace(unit.Code) == "" || strings.TrimSpace(unit.Name) == "" {
		return OrgUnit{}, errors.New("code and name are required")
	}
	if _, err := ensureDHIS2UID(unit.UID); unit.UID != "" && err != nil {
		return OrgUnit{}, err
	}
	if strings.TrimSpace(unit.ShortName) == "" {
		unit.ShortName = unit.Name
	}
	if unit.Extras == nil {
		unit.Extras = JSONMap{}
	}
	if unit.AttributeValues == nil {
		unit.AttributeValues = JSONMap{}
	}
	unit.UpdatedAt = time.Now().UTC()
	return s.repo.Update(ctx, unit)
}

func (s *Service) UpdateForUser(ctx context.Context, userID int64, unit OrgUnit) (OrgUnit, error) {
	scope, err := s.resolveScope(ctx, userID)
	if err != nil {
		return OrgUnit{}, err
	}
	if scope.Restricted {
		existing, err := s.repo.GetByID(ctx, unit.ID)
		if err != nil {
			return OrgUnit{}, err
		}
		if !userorg.ScopeContainsPath(scope, existing.Path) {
			return OrgUnit{}, apperror.Forbidden("Organisation unit is outside your assigned jurisdiction")
		}
		if unit.ParentID == nil {
			if existing.ParentID != nil {
				return OrgUnit{}, apperror.Forbidden("Organisation unit is outside your assigned jurisdiction")
			}
			return s.Update(ctx, unit)
		}
		parent, err := s.repo.GetByID(ctx, *unit.ParentID)
		if err != nil {
			return OrgUnit{}, err
		}
		if !userorg.ScopeContainsPath(scope, parent.Path) {
			return OrgUnit{}, apperror.Forbidden("Organisation unit is outside your assigned jurisdiction")
		}
	}
	return s.Update(ctx, unit)
}

// Delete removes an OrgUnit after verifying that it has no children.  The repository is
// expected to enforce referential integrity and return an error if deletion is not allowed.
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) DeleteForUser(ctx context.Context, userID, id int64) error {
	scope, err := s.resolveScope(ctx, userID)
	if err != nil {
		return err
	}
	if scope.Restricted {
		item, err := s.repo.GetByID(ctx, id)
		if err != nil {
			return err
		}
		if !userorg.ScopeContainsPath(scope, item.Path) {
			return apperror.Forbidden("Organisation unit is outside your assigned jurisdiction")
		}
	}
	return s.repo.Delete(ctx, id)
}

func (s *Service) applyScope(ctx context.Context, userID int64, query ListQuery) (ListQuery, error) {
	scope, err := s.resolveScope(ctx, userID)
	if err != nil {
		return ListQuery{}, err
	}
	if !scope.Restricted {
		return query, nil
	}
	query.ScopeRestricted = true
	query.ScopePaths = append([]string(nil), scope.PathPrefixes...)
	query.ScopeRootIDs = append([]int64(nil), scope.AssignedOrgUnitIDs...)
	return query, nil
}

func (s *Service) resolveScope(ctx context.Context, userID int64) (userorg.Scope, error) {
	if s.scopeResolver == nil {
		return userorg.Scope{}, nil
	}
	return s.scopeResolver.ResolveScope(ctx, userID)
}
