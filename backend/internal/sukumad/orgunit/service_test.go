package orgunit

import (
	"context"
	"testing"
)

type stubRepository struct {
	created OrgUnit
	updated OrgUnit
}

func (s *stubRepository) List(context.Context, ListQuery) (ListResult, error) {
	return ListResult{}, nil
}
func (s *stubRepository) GetByID(context.Context, int64) (OrgUnit, error)    { return OrgUnit{}, nil }
func (s *stubRepository) GetByUID(context.Context, string) (OrgUnit, error)  { return OrgUnit{}, nil }
func (s *stubRepository) GetByCode(context.Context, string) (OrgUnit, error) { return OrgUnit{}, nil }
func (s *stubRepository) Delete(context.Context, int64) error                { return nil }

func (s *stubRepository) Create(_ context.Context, unit OrgUnit) (OrgUnit, error) {
	s.created = unit
	return unit, nil
}

func (s *stubRepository) Update(_ context.Context, unit OrgUnit) (OrgUnit, error) {
	s.updated = unit
	return unit, nil
}

func TestServiceCreateGeneratesDHIS2UIDAndDefaults(t *testing.T) {
	repo := &stubRepository{}
	service := NewService(repo)

	created, err := service.Create(context.Background(), OrgUnit{
		Code: "FAC-001",
		Name: "Facility One",
	})
	if err != nil {
		t.Fatalf("create org unit: %v", err)
	}
	if len(created.UID) != dhis2UIDLength {
		t.Fatalf("expected generated DHIS2 uid length %d, got %q", dhis2UIDLength, created.UID)
	}
	if !dhis2UIDPattern.MatchString(created.UID) {
		t.Fatalf("expected generated uid to match DHIS2 pattern, got %q", created.UID)
	}
	if repo.created.ShortName != "Facility One" {
		t.Fatalf("expected short name default to name, got %q", repo.created.ShortName)
	}
	if repo.created.Extras == nil || repo.created.AttributeValues == nil {
		t.Fatalf("expected json maps to be initialized")
	}
}

func TestServiceCreateRejectsMalformedUID(t *testing.T) {
	service := NewService(&stubRepository{})
	_, err := service.Create(context.Background(), OrgUnit{
		UID:  "123-invalid",
		Code: "FAC-002",
		Name: "Facility Two",
	})
	if err == nil {
		t.Fatal("expected malformed DHIS2 uid to be rejected")
	}
}

func TestServiceCreateAllowsBlankCode(t *testing.T) {
	repo := &stubRepository{}
	service := NewService(repo)

	created, err := service.Create(context.Background(), OrgUnit{
		Name: "Facility Without Code",
	})
	if err != nil {
		t.Fatalf("create org unit: %v", err)
	}
	if created.Code != "" {
		t.Fatalf("expected blank code, got %q", created.Code)
	}
	if repo.created.Code != "" {
		t.Fatalf("expected repository create to receive blank code, got %q", repo.created.Code)
	}
}

func TestServiceCreateRequiresName(t *testing.T) {
	service := NewService(&stubRepository{})
	_, err := service.Create(context.Background(), OrgUnit{Code: "FAC-003"})
	if err == nil || err.Error() != "name is required" {
		t.Fatalf("expected name is required error, got %v", err)
	}
}
