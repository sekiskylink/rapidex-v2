package reporter

import (
	"context"
	"testing"
	"time"

	"basepro/backend/internal/sukumad/rapidex/rapidpro"
	sukumadserver "basepro/backend/internal/sukumad/server"
)

type reporterServiceRepo struct {
	byID              map[int64]Reporter
	updatedSinceItems []Reporter
	updateCalls       int
}

func (r *reporterServiceRepo) List(context.Context, ListQuery) (ListResult, error) {
	return ListResult{}, nil
}
func (r *reporterServiceRepo) GetByID(_ context.Context, id int64) (Reporter, error) {
	return r.byID[id], nil
}
func (r *reporterServiceRepo) GetByUID(context.Context, string) (Reporter, error) {
	return Reporter{}, nil
}
func (r *reporterServiceRepo) GetByRapidProUUID(context.Context, string) (Reporter, error) {
	return Reporter{}, nil
}
func (r *reporterServiceRepo) GetByPhoneNumber(context.Context, string) (Reporter, error) {
	return Reporter{}, nil
}
func (r *reporterServiceRepo) ListByIDs(_ context.Context, ids []int64) ([]Reporter, error) {
	items := make([]Reporter, 0, len(ids))
	for _, id := range ids {
		items = append(items, r.byID[id])
	}
	return items, nil
}
func (r *reporterServiceRepo) ListUpdatedSince(context.Context, *time.Time, int, bool) ([]Reporter, error) {
	return append([]Reporter(nil), r.updatedSinceItems...), nil
}
func (r *reporterServiceRepo) UpdateRapidProStatus(_ context.Context, id int64, rapidProUUID string, synced bool) (Reporter, error) {
	item := r.byID[id]
	item.RapidProUUID = rapidProUUID
	item.Synced = synced
	r.byID[id] = item
	r.updateCalls++
	return item, nil
}
func (r *reporterServiceRepo) Create(context.Context, Reporter) (Reporter, error) {
	return Reporter{}, nil
}
func (r *reporterServiceRepo) Update(context.Context, Reporter) (Reporter, error) {
	return Reporter{}, nil
}
func (r *reporterServiceRepo) Delete(context.Context, int64) error {
	return nil
}

type reporterServerLookup struct {
	record sukumadserver.Record
}

func (s reporterServerLookup) GetServerByCode(context.Context, string) (sukumadserver.Record, error) {
	return s.record, nil
}

type reporterRapidProClient struct {
	lookupByUUIDFound bool
	messageCalls      int
}

func (c *reporterRapidProClient) LookupContactByUUID(context.Context, rapidpro.Connection, string) (rapidpro.Contact, bool, error) {
	if !c.lookupByUUIDFound {
		return rapidpro.Contact{}, false, nil
	}
	return rapidpro.Contact{UUID: "contact-existing"}, true, nil
}
func (c *reporterRapidProClient) LookupContactByURN(context.Context, rapidpro.Connection, string) (rapidpro.Contact, bool, error) {
	return rapidpro.Contact{}, false, nil
}
func (c *reporterRapidProClient) UpsertContact(_ context.Context, _ rapidpro.Connection, input rapidpro.UpsertContactInput) (rapidpro.Contact, error) {
	if input.UUID != "" {
		return rapidpro.Contact{UUID: input.UUID}, nil
	}
	return rapidpro.Contact{UUID: "contact-created"}, nil
}
func (c *reporterRapidProClient) EnsureGroup(_ context.Context, _ rapidpro.Connection, name string) (rapidpro.Group, bool, error) {
	return rapidpro.Group{UUID: "group-" + name, Name: name}, true, nil
}
func (c *reporterRapidProClient) SendMessage(context.Context, rapidpro.Connection, string, string) (rapidpro.Message, error) {
	c.messageCalls++
	return rapidpro.Message{}, nil
}
func (c *reporterRapidProClient) SendBroadcast(context.Context, rapidpro.Connection, []string, string) (rapidpro.Broadcast, error) {
	return rapidpro.Broadcast{}, nil
}

func TestSyncReporterUpdatesExistingRapidProContact(t *testing.T) {
	repo := &reporterServiceRepo{
		byID: map[int64]Reporter{
			1: {
				ID:           1,
				Name:         "Alice Reporter",
				Telephone:    "+256700000001",
				OrgUnitID:    2,
				Groups:       []string{"Lead"},
				RapidProUUID: "contact-existing",
				IsActive:     true,
			},
		},
	}
	client := &reporterRapidProClient{lookupByUUIDFound: true}
	service := NewService(repo).
		WithRapidProIntegration(reporterServerLookup{
			record: sukumadserver.Record{
				Code:    "rapidpro",
				BaseURL: "https://rapidpro.example.com",
				Headers: map[string]string{"Authorization": "Token secret"},
			},
		}, client)

	result, err := service.SyncReporter(context.Background(), 1)
	if err != nil {
		t.Fatalf("sync reporter: %v", err)
	}
	if result.Operation != "updated" {
		t.Fatalf("expected update operation, got %q", result.Operation)
	}
	if result.Reporter.RapidProUUID != "contact-existing" {
		t.Fatalf("expected persisted contact uuid, got %q", result.Reporter.RapidProUUID)
	}
	if !result.Reporter.Synced {
		t.Fatalf("expected reporter to be marked synced")
	}
	if repo.updateCalls != 1 {
		t.Fatalf("expected one rapidpro status update, got %d", repo.updateCalls)
	}
}

func TestSyncReporterRequiresRapidProUUID(t *testing.T) {
	repo := &reporterServiceRepo{
		byID: map[int64]Reporter{
			1: {
				ID:        1,
				Name:      "Alice Reporter",
				Telephone: "+256700000001",
				OrgUnitID: 2,
				IsActive:  true,
			},
		},
	}
	client := &reporterRapidProClient{lookupByUUIDFound: true}
	service := NewService(repo).
		WithRapidProIntegration(reporterServerLookup{
			record: sukumadserver.Record{
				Code:    "rapidpro",
				BaseURL: "https://rapidpro.example.com",
			},
		}, client)

	if _, err := service.SyncReporter(context.Background(), 1); err == nil {
		t.Fatal("expected sync reporter to reject empty rapidpro uuid")
	}
	if repo.updateCalls != 0 {
		t.Fatalf("expected no persistence updates, got %d", repo.updateCalls)
	}
}

func TestSyncUpdatedSinceDryRunSkipsRemoteMutation(t *testing.T) {
	now := time.Date(2026, time.April, 21, 12, 0, 0, 0, time.UTC)
	repo := &reporterServiceRepo{
		byID: map[int64]Reporter{},
		updatedSinceItems: []Reporter{
			{ID: 1, Name: "Alice", Telephone: "+256700000001", OrgUnitID: 2, IsActive: true},
		},
	}
	service := NewService(repo).WithClock(func() time.Time { return now })

	result, err := service.SyncUpdatedSince(context.Background(), &now, 50, true, true)
	if err != nil {
		t.Fatalf("dry run sync updated since: %v", err)
	}
	if result.Scanned != 1 || result.Synced != 0 {
		t.Fatalf("unexpected dry-run counts: %+v", result)
	}
	if repo.updateCalls != 0 {
		t.Fatalf("expected dry run to avoid persistence, got %d updates", repo.updateCalls)
	}
	if result.WatermarkTo == nil || !result.WatermarkTo.Equal(now) {
		t.Fatalf("expected watermark to equal clock time, got %+v", result.WatermarkTo)
	}
}
