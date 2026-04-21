package orgunit

import (
	"context"
	"errors"
	"strings"
	"time"
)

// Service encapsulates business logic for organisation units.  It depends on a Repository.
type Service struct {
	repo Repository
}

// NewService constructs a new Service from a repository implementation.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// List returns a page of OrgUnits matching the provided query.
func (s *Service) List(ctx context.Context, query ListQuery) (ListResult, error) {
	return s.repo.List(ctx, query)
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

// Delete removes an OrgUnit after verifying that it has no children.  The repository is
// expected to enforce referential integrity and return an error if deletion is not allowed.
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
