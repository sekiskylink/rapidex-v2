package reporter

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/logging"
	"basepro/backend/internal/settings"
	"basepro/backend/internal/sukumad/orgunit"
	"basepro/backend/internal/sukumad/rapidex/rapidpro"
	sukumadrequest "basepro/backend/internal/sukumad/request"
	sukumadserver "basepro/backend/internal/sukumad/server"
	"basepro/backend/internal/sukumad/userorg"
)

type reporterServiceRepo struct {
	byID                     map[int64]Reporter
	broadcastListResult      BroadcastListResult
	updatedSinceItems        []Reporter
	updateCalls              int
	countBroadcastRecipients int
	broadcastRecipients      []Reporter
	recentBroadcast          *JurisdictionBroadcastRecord
	recentReports            []sukumadrequest.Record
	recentReportsQuery       *sukumadrequest.ReporterRecentReportsQuery
	createdBroadcasts        []JurisdictionBroadcastRecord
	claimedBroadcasts        []JurisdictionBroadcastRecord
	updatedBroadcasts        []JurisdictionBroadcastRecord
}

func (r *reporterServiceRepo) List(context.Context, ListQuery) (ListResult, error) {
	return ListResult{}, nil
}
func (r *reporterServiceRepo) ListBroadcasts(context.Context, BroadcastListQuery) (BroadcastListResult, error) {
	return r.broadcastListResult, nil
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
func (r *reporterServiceRepo) CountBroadcastRecipients(context.Context, BroadcastRecipientQuery) (int, error) {
	return r.countBroadcastRecipients, nil
}
func (r *reporterServiceRepo) ListBroadcastRecipients(context.Context, BroadcastRecipientQuery) ([]Reporter, error) {
	return append([]Reporter(nil), r.broadcastRecipients...), nil
}
func (r *reporterServiceRepo) GetRecentPendingBroadcastByDedupeKey(context.Context, string, time.Time) (JurisdictionBroadcastRecord, error) {
	if r.recentBroadcast != nil {
		return *r.recentBroadcast, nil
	}
	return JurisdictionBroadcastRecord{}, sql.ErrNoRows
}
func (r *reporterServiceRepo) ListRecentReporterReports(_ context.Context, query sukumadrequest.ReporterRecentReportsQuery) ([]sukumadrequest.Record, error) {
	copyQuery := query
	r.recentReportsQuery = &copyQuery
	return append([]sukumadrequest.Record(nil), r.recentReports...), nil
}
func (r *reporterServiceRepo) CreateJurisdictionBroadcast(_ context.Context, record JurisdictionBroadcastRecord) (JurisdictionBroadcastRecord, error) {
	record.ID = int64(len(r.createdBroadcasts) + 1)
	r.createdBroadcasts = append(r.createdBroadcasts, record)
	return record, nil
}
func (r *reporterServiceRepo) ClaimNextJurisdictionBroadcast(context.Context, time.Time, time.Duration, int64) (JurisdictionBroadcastRecord, error) {
	if len(r.claimedBroadcasts) > 0 {
		item := r.claimedBroadcasts[0]
		r.claimedBroadcasts = r.claimedBroadcasts[1:]
		return item, nil
	}
	return JurisdictionBroadcastRecord{}, ErrNoEligibleBroadcast
}
func (r *reporterServiceRepo) UpdateJurisdictionBroadcastResult(_ context.Context, id int64, status string, sentCount int, failedCount int, lastError string, finishedAt time.Time) (JurisdictionBroadcastRecord, error) {
	record := JurisdictionBroadcastRecord{ID: id, Status: status, SentCount: sentCount, FailedCount: failedCount, LastError: lastError, FinishedAt: &finishedAt}
	r.updatedBroadcasts = append(r.updatedBroadcasts, record)
	return record, nil
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
	record         sukumadserver.Record
	records        map[string]sukumadserver.Record
	requestedCodes []string
}

func (s *reporterServerLookup) GetServerByCode(_ context.Context, code string) (sukumadserver.Record, error) {
	s.requestedCodes = append(s.requestedCodes, code)
	if len(s.records) > 0 {
		record, ok := s.records[code]
		if !ok {
			return sukumadserver.Record{}, errors.New("server not found")
		}
		return record, nil
	}
	return s.record, nil
}

type reporterRapidProClient struct {
	lookupByUUIDFound bool
	lookupByURNFound  bool
	lookupGroupFound  bool
	groupLookupSet    bool
	messageCalls      int
	lastUpsertInput   rapidpro.UpsertContactInput
	uuidContact       rapidpro.Contact
	urnContact        rapidpro.Contact
	messages          []rapidpro.Message
	messagesNext      string
	messagePages      map[string]rapidProMessagePage
}

type rapidProMessagePage struct {
	messages []rapidpro.Message
	next     string
}

func (c *reporterRapidProClient) LookupContactByUUID(context.Context, rapidpro.Connection, string) (rapidpro.Contact, bool, error) {
	if !c.lookupByUUIDFound {
		return rapidpro.Contact{}, false, nil
	}
	if strings.TrimSpace(c.uuidContact.UUID) != "" {
		return c.uuidContact, true, nil
	}
	return rapidpro.Contact{UUID: "contact-existing"}, true, nil
}
func (c *reporterRapidProClient) LookupContactByURN(context.Context, rapidpro.Connection, string) (rapidpro.Contact, bool, error) {
	if !c.lookupByURNFound {
		return rapidpro.Contact{}, false, nil
	}
	if strings.TrimSpace(c.urnContact.UUID) != "" {
		return c.urnContact, true, nil
	}
	return rapidpro.Contact{UUID: "contact-by-urn"}, true, nil
}
func (c *reporterRapidProClient) UpsertContact(_ context.Context, _ rapidpro.Connection, input rapidpro.UpsertContactInput) (rapidpro.Contact, error) {
	c.lastUpsertInput = input
	if input.UUID != "" {
		return rapidpro.Contact{UUID: input.UUID}, nil
	}
	return rapidpro.Contact{UUID: "contact-created"}, nil
}
func (c *reporterRapidProClient) LookupGroupByName(_ context.Context, _ rapidpro.Connection, name string) (rapidpro.Group, bool, error) {
	if c.groupLookupSet && !c.lookupGroupFound {
		return rapidpro.Group{}, false, nil
	}
	return rapidpro.Group{UUID: "group-" + name, Name: name}, true, nil
}
func (c *reporterRapidProClient) CreateGroup(_ context.Context, _ rapidpro.Connection, name string) (rapidpro.Group, error) {
	return rapidpro.Group{UUID: "group-" + name, Name: name}, nil
}
func (c *reporterRapidProClient) SendMessage(context.Context, rapidpro.Connection, string, string) (rapidpro.Message, error) {
	c.messageCalls++
	return rapidpro.Message{}, nil
}
func (c *reporterRapidProClient) SendBroadcast(context.Context, rapidpro.Connection, []string, string) (rapidpro.Broadcast, error) {
	return rapidpro.Broadcast{}, nil
}
func (c *reporterRapidProClient) ListMessages(_ context.Context, _ rapidpro.Connection, query map[string]string) ([]rapidpro.Message, string, error) {
	c.messageCalls++
	cursor := ""
	if query != nil {
		cursor = strings.TrimSpace(query["cursor"])
	}
	if len(c.messagePages) > 0 {
		page, ok := c.messagePages[cursor]
		if !ok {
			return nil, "", nil
		}
		return append([]rapidpro.Message(nil), page.messages...), page.next, nil
	}
	return append([]rapidpro.Message(nil), c.messages...), c.messagesNext, nil
}

type reporterSettingsProvider struct {
	config settings.RapidProReporterSyncSettings
}

func (p reporterSettingsProvider) GetRapidProReporterSync(context.Context) (settings.RapidProReporterSyncSettings, error) {
	return p.config, nil
}

type reporterOrgUnitLookup struct {
	item  orgunit.OrgUnit
	items map[int64]orgunit.OrgUnit
}

func (l reporterOrgUnitLookup) Get(_ context.Context, id int64) (orgunit.OrgUnit, error) {
	if len(l.items) > 0 {
		if item, ok := l.items[id]; ok {
			return item, nil
		}
	}
	return l.item, nil
}

type reporterScopeResolverStub struct {
	scope userorg.Scope
	err   error
}

func (s reporterScopeResolverStub) ResolveScope(context.Context, int64) (userorg.Scope, error) {
	return s.scope, s.err
}

type reporterGroupCatalogStub struct {
	validatedGroups []string
}

func (s *reporterGroupCatalogStub) ValidateActiveNames(_ context.Context, names []string) ([]string, error) {
	s.validatedGroups = append([]string(nil), names...)
	return append([]string(nil), names...), nil
}

func (s *reporterGroupCatalogStub) EnsureRapidProGroups(_ context.Context, names []string) ([]rapidpro.Group, error) {
	result := make([]rapidpro.Group, 0, len(names))
	for _, name := range names {
		result = append(result, rapidpro.Group{UUID: "group-" + name, Name: name})
	}
	return result, nil
}

func TestQueueJurisdictionBroadcastForUserQueuesBackgroundBroadcast(t *testing.T) {
	repo := &reporterServiceRepo{countBroadcastRecipients: 3}
	groupCatalog := &reporterGroupCatalogStub{}
	service := NewService(repo).
		WithReporterGroupCatalog(groupCatalog).
		WithOrgUnitLookup(reporterOrgUnitLookup{
			items: map[int64]orgunit.OrgUnit{
				9: {ID: 9, Name: "Kampala District", Path: "/UG/Kampala/"},
			},
		}).
		WithClock(func() time.Time { return time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC) })

	result, err := service.QueueJurisdictionBroadcastForUser(context.Background(), 7, JurisdictionBroadcastInput{
		OrgUnitIDs:    []int64{9, 9},
		ReporterGroup: "Lead",
		Text:          "Please submit today",
	})
	if err != nil {
		t.Fatalf("queue jurisdiction broadcast: %v", err)
	}
	if result.Status != BroadcastQueueResultQueued {
		t.Fatalf("expected queued status, got %q", result.Status)
	}
	if len(repo.createdBroadcasts) != 1 {
		t.Fatalf("expected one queued broadcast, got %d", len(repo.createdBroadcasts))
	}
	created := repo.createdBroadcasts[0]
	if created.MatchedCount != 3 {
		t.Fatalf("expected matched count 3, got %d", created.MatchedCount)
	}
	if len(created.OrgUnitIDs) != 1 || created.OrgUnitIDs[0] != 9 {
		t.Fatalf("expected normalized org unit ids, got %#v", created.OrgUnitIDs)
	}
	if created.ReporterGroup != "Lead" {
		t.Fatalf("expected reporter group Lead, got %q", created.ReporterGroup)
	}
	if got := strings.TrimSpace(created.MessageText); got != "Please submit today" {
		t.Fatalf("expected message text to be persisted, got %q", got)
	}
	if len(groupCatalog.validatedGroups) != 1 || groupCatalog.validatedGroups[0] != "Lead" {
		t.Fatalf("expected group validation for Lead, got %#v", groupCatalog.validatedGroups)
	}
}

func TestListBroadcastsReturnsRepositoryResult(t *testing.T) {
	repo := &reporterServiceRepo{
		broadcastListResult: BroadcastListResult{
			Items: []JurisdictionBroadcastRecord{
				{ID: 31, Status: BroadcastStatusCompleted, ReporterGroup: "Lead", MatchedCount: 4, SentCount: 4},
			},
			Total:    1,
			Page:     0,
			PageSize: 10,
		},
	}
	service := NewService(repo)

	result, err := service.ListBroadcasts(context.Background(), BroadcastListQuery{Page: 0, PageSize: 10})
	if err != nil {
		t.Fatalf("list broadcasts: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}
	if len(result.Items) != 1 || result.Items[0].ID != 31 {
		t.Fatalf("unexpected broadcast items: %+v", result.Items)
	}
}

func TestQueueJurisdictionBroadcastForUserReturnsDuplicatePending(t *testing.T) {
	repo := &reporterServiceRepo{
		countBroadcastRecipients: 2,
		recentBroadcast: &JurisdictionBroadcastRecord{
			ID:           11,
			UID:          "rb-11",
			MatchedCount: 2,
			Status:       BroadcastStatusRunning,
		},
	}
	service := NewService(repo).
		WithOrgUnitLookup(reporterOrgUnitLookup{
			items: map[int64]orgunit.OrgUnit{
				9: {ID: 9, Name: "Kampala District", Path: "/UG/Kampala/"},
			},
		}).
		WithClock(func() time.Time { return time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC) })

	result, err := service.QueueJurisdictionBroadcastForUser(context.Background(), 7, JurisdictionBroadcastInput{
		OrgUnitIDs:    []int64{9},
		ReporterGroup: "Lead",
		Text:          "Please submit today",
	})
	if err != nil {
		t.Fatalf("queue duplicate jurisdiction broadcast: %v", err)
	}
	if result.Status != BroadcastQueueResultDuplicate {
		t.Fatalf("expected duplicate pending status, got %q", result.Status)
	}
	if len(repo.createdBroadcasts) != 0 {
		t.Fatalf("expected no new broadcast to be created, got %d", len(repo.createdBroadcasts))
	}
}

func TestRunQueuedBroadcastsProcessesClaimedBroadcast(t *testing.T) {
	repo := &reporterServiceRepo{
		byID: map[int64]Reporter{
			1: {ID: 1, Name: "Alice Reporter", Telephone: "+256700000001", OrgUnitID: 9, IsActive: true},
			2: {ID: 2, Name: "Bob Reporter", Telephone: "+256700000002", OrgUnitID: 9, IsActive: true},
		},
		broadcastRecipients: []Reporter{
			{ID: 1, Name: "Alice Reporter", Telephone: "+256700000001", OrgUnitID: 9, IsActive: true},
			{ID: 2, Name: "Bob Reporter", Telephone: "+256700000002", OrgUnitID: 9, IsActive: true},
		},
		claimedBroadcasts: []JurisdictionBroadcastRecord{
			{
				ID:            21,
				UID:           "rb-21",
				OrgUnitIDs:    []int64{9},
				ReporterGroup: "Lead",
				MessageText:   "Background hello",
				MatchedCount:  2,
				Status:        BroadcastStatusRunning,
			},
		},
	}
	client := &reporterRapidProClient{}
	service := NewService(repo).
		WithRapidProIntegration(&reporterServerLookup{
			record: sukumadserver.Record{Code: "rapidpro", BaseURL: "https://rapidpro.example.com"},
		}, client).
		WithOrgUnitLookup(reporterOrgUnitLookup{
			items: map[int64]orgunit.OrgUnit{
				9: {ID: 9, Name: "Kampala District", Path: "/UG/Kampala/"},
			},
		}).
		WithClock(func() time.Time { return time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC) })

	var observed []string
	err := service.RunQueuedBroadcasts(context.Background(), 55, func(name string, delta int) {
		observed = append(observed, fmt.Sprintf("%s:%d", name, delta))
	}, 5, time.Minute)
	if err != nil {
		t.Fatalf("run queued broadcasts: %v", err)
	}
	if len(repo.updatedBroadcasts) == 0 {
		t.Fatal("expected broadcast result update")
	}
	updated := repo.updatedBroadcasts[len(repo.updatedBroadcasts)-1]
	if updated.Status != BroadcastStatusCompleted {
		t.Fatalf("expected completed status, got %q", updated.Status)
	}
	if updated.SentCount != 2 || updated.FailedCount != 0 {
		t.Fatalf("expected sent=2 failed=0, got sent=%d failed=%d", updated.SentCount, updated.FailedCount)
	}
	if !strings.Contains(strings.Join(observed, ","), "claimed:1") || !strings.Contains(strings.Join(observed, ","), "completed:1") {
		t.Fatalf("expected worker observations, got %#v", observed)
	}
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
	serverLookup := &reporterServerLookup{
		record: sukumadserver.Record{
			Code:    "rapidpro",
			BaseURL: "https://rapidpro.example.com",
			Headers: map[string]string{"Authorization": "Token secret"},
		},
	}
	service := NewService(repo).
		WithRapidProIntegration(serverLookup, client)

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
	if len(serverLookup.requestedCodes) != 1 || serverLookup.requestedCodes[0] != "rapidpro" {
		t.Fatalf("expected default server code lookup, got %#v", serverLookup.requestedCodes)
	}
}

func TestSyncReporterIncludesConfiguredRapidProContactFields(t *testing.T) {
	repo := &reporterServiceRepo{
		byID: map[int64]Reporter{
			1: {
				ID:                1,
				Name:              "Alice Reporter",
				Telephone:         "+256700000001",
				ReportingLocation: "Kampala",
				OrgUnitID:         2,
				IsActive:          true,
			},
		},
	}
	client := &reporterRapidProClient{}
	service := NewService(repo).
		WithRapidProIntegration(&reporterServerLookup{
			record: sukumadserver.Record{
				Code:    "rapidpro",
				BaseURL: "https://rapidpro.example.com",
			},
		}, client).
		WithRapidProSettings(reporterSettingsProvider{
			config: settings.RapidProReporterSyncSettings{
				Mappings: []settings.RapidProReporterFieldMapping{
					{SourceKey: "facilityName", SourceLabel: "Facility Name", RapidProFieldKey: "Facility"},
					{SourceKey: "facilityUID", SourceLabel: "Facility UID", RapidProFieldKey: "FacilityCode"},
				},
				Validation: settings.RapidProReporterSyncValidation{IsValid: true},
			},
		}).
		WithOrgUnitLookup(reporterOrgUnitLookup{
			item: orgunit.OrgUnit{ID: 2, UID: "ou-2", Name: "Kampala Health Centre"},
		})

	result, err := service.SyncReporter(context.Background(), 1)
	if err != nil {
		t.Fatalf("sync reporter: %v", err)
	}
	if result.Reporter.RapidProUUID != "contact-created" {
		t.Fatalf("expected created contact uuid to be persisted, got %q", result.Reporter.RapidProUUID)
	}
	if got := client.lastUpsertInput.Fields["Facility"]; got != "Kampala Health Centre" {
		t.Fatalf("expected Facility field to use org unit name, got %q", got)
	}
	if got := client.lastUpsertInput.Fields["FacilityCode"]; got != "ou-2" {
		t.Fatalf("expected FacilityCode field to use org unit uid, got %q", got)
	}
}

func TestSyncReporterMapsBuiltInRapidProTargets(t *testing.T) {
	repo := &reporterServiceRepo{
		byID: map[int64]Reporter{
			1: {
				ID:                1,
				Name:              "Alice Reporter",
				Telephone:         "+256700000001",
				WhatsApp:          "+256700000099",
				ReportingLocation: "Kampala",
				OrgUnitID:         2,
				IsActive:          true,
			},
		},
	}
	client := &reporterRapidProClient{}
	service := NewService(repo).
		WithRapidProIntegration(&reporterServerLookup{
			record: sukumadserver.Record{
				Code:    "rapidpro",
				BaseURL: "https://rapidpro.example.com",
			},
		}, client).
		WithRapidProSettings(reporterSettingsProvider{
			config: settings.RapidProReporterSyncSettings{
				Mappings: []settings.RapidProReporterFieldMapping{
					{SourceKey: "facilityName", SourceLabel: "Facility Name", RapidProFieldKey: "name"},
					{SourceKey: "telephone", SourceLabel: "Telephone", RapidProFieldKey: "urn.tel"},
					{SourceKey: "whatsapp", SourceLabel: "WhatsApp", RapidProFieldKey: "urn.whatsapp"},
					{SourceKey: "facilityUID", SourceLabel: "Facility UID", RapidProFieldKey: "FacilityCode"},
				},
				Validation: settings.RapidProReporterSyncValidation{IsValid: true},
			},
		}).
		WithOrgUnitLookup(reporterOrgUnitLookup{
			item: orgunit.OrgUnit{ID: 2, UID: "ou-2", Name: "Kampala Health Centre"},
		})

	_, err := service.SyncReporter(context.Background(), 1)
	if err != nil {
		t.Fatalf("sync reporter: %v", err)
	}
	if client.lastUpsertInput.Name != "Kampala Health Centre" {
		t.Fatalf("expected built-in name mapping, got %q", client.lastUpsertInput.Name)
	}
	if len(client.lastUpsertInput.URNs) != 2 {
		t.Fatalf("expected both tel and whatsapp URNs, got %#v", client.lastUpsertInput.URNs)
	}
	if client.lastUpsertInput.URNs[0] != "tel:+256700000001" {
		t.Fatalf("expected tel urn first, got %#v", client.lastUpsertInput.URNs)
	}
	if client.lastUpsertInput.URNs[1] != "whatsapp:+256700000099" {
		t.Fatalf("expected whatsapp urn second, got %#v", client.lastUpsertInput.URNs)
	}
	if got := client.lastUpsertInput.Fields["FacilityCode"]; got != "ou-2" {
		t.Fatalf("expected custom field to remain in Fields, got %q", got)
	}
}

func TestSyncReporterAllowsMissingWhatsAppForOptionalURNMapping(t *testing.T) {
	repo := &reporterServiceRepo{
		byID: map[int64]Reporter{
			1: {
				ID:                1,
				Name:              "Alice Reporter",
				Telephone:         "+256700000001",
				ReportingLocation: "Kampala",
				OrgUnitID:         2,
				IsActive:          true,
			},
		},
	}
	client := &reporterRapidProClient{}
	service := NewService(repo).
		WithRapidProIntegration(&reporterServerLookup{
			record: sukumadserver.Record{
				Code:    "rapidpro",
				BaseURL: "https://rapidpro.example.com",
			},
		}, client).
		WithRapidProSettings(reporterSettingsProvider{
			config: settings.RapidProReporterSyncSettings{
				Mappings: []settings.RapidProReporterFieldMapping{
					{SourceKey: "telephone", SourceLabel: "Telephone", RapidProFieldKey: "urn.tel"},
					{SourceKey: "whatsapp", SourceLabel: "WhatsApp", RapidProFieldKey: "urn.whatsapp"},
				},
				Validation: settings.RapidProReporterSyncValidation{IsValid: true},
			},
		})

	_, err := service.SyncReporter(context.Background(), 1)
	if err != nil {
		t.Fatalf("sync reporter: %v", err)
	}
	if len(client.lastUpsertInput.URNs) != 1 {
		t.Fatalf("expected only telephone URN when whatsapp is missing, got %#v", client.lastUpsertInput.URNs)
	}
	if client.lastUpsertInput.URNs[0] != "tel:+256700000001" {
		t.Fatalf("expected telephone URN to remain, got %#v", client.lastUpsertInput.URNs)
	}
}

func TestSyncReporterUsesRapidProServerCodeFromSettings(t *testing.T) {
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
	client := &reporterRapidProClient{}
	serverLookup := &reporterServerLookup{
		records: map[string]sukumadserver.Record{
			"rapidpro-custom": {
				Code:    "rapidpro-custom",
				BaseURL: "https://custom.example.com",
			},
		},
	}
	service := NewService(repo).
		WithRapidProIntegration(serverLookup, client).
		WithRapidProSettings(reporterSettingsProvider{
			config: settings.RapidProReporterSyncSettings{
				RapidProServerCode: "rapidpro-custom",
				Validation:         settings.RapidProReporterSyncValidation{IsValid: true},
			},
		})

	_, err := service.SyncReporter(context.Background(), 1)
	if err != nil {
		t.Fatalf("sync reporter: %v", err)
	}
	if len(serverLookup.requestedCodes) == 0 || serverLookup.requestedCodes[0] != "rapidpro-custom" {
		t.Fatalf("expected custom server code lookup, got %#v", serverLookup.requestedCodes)
	}
}

func TestSyncReporterFallsBackToDefaultServerCodeWhenSettingsBlank(t *testing.T) {
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
	client := &reporterRapidProClient{}
	serverLookup := &reporterServerLookup{
		record: sukumadserver.Record{
			Code:    "rapidpro",
			BaseURL: "https://rapidpro.example.com",
		},
	}
	service := NewService(repo).
		WithRapidProIntegration(serverLookup, client).
		WithRapidProSettings(reporterSettingsProvider{
			config: settings.RapidProReporterSyncSettings{
				RapidProServerCode: "   ",
				Validation:         settings.RapidProReporterSyncValidation{IsValid: true},
			},
		})

	_, err := service.SyncReporter(context.Background(), 1)
	if err != nil {
		t.Fatalf("sync reporter: %v", err)
	}
	if len(serverLookup.requestedCodes) == 0 || serverLookup.requestedCodes[0] != "rapidpro" {
		t.Fatalf("expected default server code lookup, got %#v", serverLookup.requestedCodes)
	}
}

func TestSyncReporterReturnsValidationErrorForRapidProClient400(t *testing.T) {
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
	client := &reporterRapidProClient{}
	serverLookup := &reporterServerLookup{
		record: sukumadserver.Record{
			Code:    "rapidpro",
			BaseURL: "https://rapidpro.example.com",
		},
	}
	clientError := &rapidpro.RequestError{StatusCode: 400, Body: `{"fields":["Facility is invalid"]}`}
	service := NewService(repo).
		WithRapidProIntegration(serverLookup, client).
		WithRapidProSettings(reporterSettingsProvider{
			config: settings.RapidProReporterSyncSettings{
				Validation: settings.RapidProReporterSyncValidation{IsValid: true},
			},
		})
	client.lastUpsertInput = rapidpro.UpsertContactInput{}
	originalClient := service.rapidProClient
	service.rapidProClient = &reporterRapidProClientWithUpsertError{delegate: originalClient, err: clientError}

	_, err := service.SyncReporter(context.Background(), 1)
	if err == nil {
		t.Fatal("expected sync reporter to fail")
	}
	var typed *apperror.AppError
	if !errors.As(err, &typed) {
		t.Fatalf("expected app error, got %T", err)
	}
	if typed.Code != apperror.CodeValidationFailed {
		t.Fatalf("expected validation code, got %+v", typed)
	}
	message := strings.Join(typed.Details["rapidpro"].([]string), " ")
	if !strings.Contains(message, "Facility is invalid") {
		t.Fatalf("expected rapidpro validation detail, got %#v", typed.Details)
	}
}

type reporterRapidProClientWithUpsertError struct {
	delegate rapidProClient
	err      error
}

func (c *reporterRapidProClientWithUpsertError) LookupContactByUUID(ctx context.Context, conn rapidpro.Connection, uuid string) (rapidpro.Contact, bool, error) {
	return c.delegate.LookupContactByUUID(ctx, conn, uuid)
}

func (c *reporterRapidProClientWithUpsertError) LookupContactByURN(ctx context.Context, conn rapidpro.Connection, urn string) (rapidpro.Contact, bool, error) {
	return c.delegate.LookupContactByURN(ctx, conn, urn)
}

func (c *reporterRapidProClientWithUpsertError) UpsertContact(context.Context, rapidpro.Connection, rapidpro.UpsertContactInput) (rapidpro.Contact, error) {
	return rapidpro.Contact{}, c.err
}

func (c *reporterRapidProClientWithUpsertError) LookupGroupByName(ctx context.Context, conn rapidpro.Connection, name string) (rapidpro.Group, bool, error) {
	return c.delegate.LookupGroupByName(ctx, conn, name)
}

func (c *reporterRapidProClientWithUpsertError) CreateGroup(ctx context.Context, conn rapidpro.Connection, name string) (rapidpro.Group, error) {
	return c.delegate.CreateGroup(ctx, conn, name)
}

func (c *reporterRapidProClientWithUpsertError) SendMessage(ctx context.Context, conn rapidpro.Connection, contactUUID string, text string) (rapidpro.Message, error) {
	return c.delegate.SendMessage(ctx, conn, contactUUID, text)
}

func (c *reporterRapidProClientWithUpsertError) SendBroadcast(ctx context.Context, conn rapidpro.Connection, contactUUIDs []string, text string) (rapidpro.Broadcast, error) {
	return c.delegate.SendBroadcast(ctx, conn, contactUUIDs, text)
}

func (c *reporterRapidProClientWithUpsertError) ListMessages(ctx context.Context, conn rapidpro.Connection, query map[string]string) ([]rapidpro.Message, string, error) {
	return c.delegate.ListMessages(ctx, conn, query)
}

func TestSyncReporterFailsWhenRapidProFieldMappingIsInvalid(t *testing.T) {
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
	service := NewService(repo).
		WithRapidProIntegration(&reporterServerLookup{
			record: sukumadserver.Record{
				Code:    "rapidpro",
				BaseURL: "https://rapidpro.example.com",
			},
		}, &reporterRapidProClient{}).
		WithRapidProSettings(reporterSettingsProvider{
			config: settings.RapidProReporterSyncSettings{
				Validation: settings.RapidProReporterSyncValidation{
					IsValid: false,
					Errors:  []string{`Mapped RapidPro field "Facility" is no longer available.`},
				},
			},
		})

	_, err := service.SyncReporter(context.Background(), 1)
	if err == nil {
		t.Fatal("expected sync to fail for invalid rapidpro mapping")
	}
}

func TestSyncReporterLooksUpByURNWhenRapidProUUIDMissing(t *testing.T) {
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
	client := &reporterRapidProClient{lookupByURNFound: true}
	service := NewService(repo).
		WithRapidProIntegration(&reporterServerLookup{
			record: sukumadserver.Record{
				Code:    "rapidpro",
				BaseURL: "https://rapidpro.example.com",
			},
		}, client)

	result, err := service.SyncReporter(context.Background(), 1)
	if err != nil {
		t.Fatalf("sync reporter: %v", err)
	}
	if result.Operation != "updated" {
		t.Fatalf("expected update operation after URN match, got %q", result.Operation)
	}
	if result.Reporter.RapidProUUID != "contact-by-urn" {
		t.Fatalf("expected URN lookup uuid to be persisted, got %q", result.Reporter.RapidProUUID)
	}
	if client.lastUpsertInput.UUID != "contact-by-urn" {
		t.Fatalf("expected upsert to reuse URN lookup uuid, got %q", client.lastUpsertInput.UUID)
	}
	if repo.updateCalls != 1 {
		t.Fatalf("expected one persistence update, got %d", repo.updateCalls)
	}
}

func TestSyncReporterCreatesRapidProContactWhenLookupMisses(t *testing.T) {
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
	client := &reporterRapidProClient{}
	service := NewService(repo).
		WithRapidProIntegration(&reporterServerLookup{
			record: sukumadserver.Record{
				Code:    "rapidpro",
				BaseURL: "https://rapidpro.example.com",
			},
		}, client)

	result, err := service.SyncReporter(context.Background(), 1)
	if err != nil {
		t.Fatalf("sync reporter: %v", err)
	}
	if result.Operation != "created" {
		t.Fatalf("expected create operation, got %q", result.Operation)
	}
	if result.Reporter.RapidProUUID != "contact-created" {
		t.Fatalf("expected created contact uuid to be persisted, got %q", result.Reporter.RapidProUUID)
	}
	if client.lastUpsertInput.UUID != "" {
		t.Fatalf("expected create path to upsert without preset uuid, got %q", client.lastUpsertInput.UUID)
	}
	if repo.updateCalls != 1 {
		t.Fatalf("expected one persistence update, got %d", repo.updateCalls)
	}
}

func TestSyncReporterFallsBackToURNWhenStoredRapidProUUIDIsStale(t *testing.T) {
	repo := &reporterServiceRepo{
		byID: map[int64]Reporter{
			1: {
				ID:           1,
				Name:         "Alice Reporter",
				Telephone:    "+256700000001",
				OrgUnitID:    2,
				RapidProUUID: "contact-stale",
				IsActive:     true,
			},
		},
	}
	client := &reporterRapidProClient{lookupByURNFound: true}
	service := NewService(repo).
		WithRapidProIntegration(&reporterServerLookup{
			record: sukumadserver.Record{
				Code:    "rapidpro",
				BaseURL: "https://rapidpro.example.com",
			},
		}, client)

	result, err := service.SyncReporter(context.Background(), 1)
	if err != nil {
		t.Fatalf("sync reporter: %v", err)
	}
	if result.Operation != "updated" {
		t.Fatalf("expected update operation after stale uuid fallback, got %q", result.Operation)
	}
	if result.Reporter.RapidProUUID != "contact-by-urn" {
		t.Fatalf("expected fallback uuid to be persisted, got %q", result.Reporter.RapidProUUID)
	}
	if client.lastUpsertInput.UUID != "contact-by-urn" {
		t.Fatalf("expected upsert to use fallback uuid, got %q", client.lastUpsertInput.UUID)
	}
	if repo.updateCalls != 1 {
		t.Fatalf("expected one persistence update, got %d", repo.updateCalls)
	}
}

func TestSyncReporterFailsWhenRapidProGroupDoesNotExist(t *testing.T) {
	repo := &reporterServiceRepo{
		byID: map[int64]Reporter{
			1: {
				ID:        1,
				Name:      "Alice Reporter",
				Telephone: "+256700000001",
				OrgUnitID: 2,
				Groups:    []string{"Lead"},
				IsActive:  true,
			},
		},
	}
	client := &reporterRapidProClient{groupLookupSet: true, lookupGroupFound: false}
	service := NewService(repo).
		WithRapidProIntegration(&reporterServerLookup{
			record: sukumadserver.Record{
				Code:    "rapidpro",
				BaseURL: "https://rapidpro.example.com",
			},
		}, client)

	_, err := service.SyncReporter(context.Background(), 1)
	if err == nil {
		t.Fatal("expected missing rapidpro group validation")
	}
	var typed *apperror.AppError
	if !errors.As(err, &typed) {
		t.Fatalf("expected app error, got %T", err)
	}
	message := strings.Join(typed.Details["rapidproGroups"].([]string), " ")
	if !strings.Contains(message, "Lead") {
		t.Fatalf("expected missing group name in error, got %#v", typed.Details)
	}
}

func TestBuildRapidProReporterSyncPreviewReturnsRequestShape(t *testing.T) {
	repo := &reporterServiceRepo{
		byID: map[int64]Reporter{
			1: {
				ID:        1,
				Name:      "Alice Reporter",
				Telephone: "+256700000001",
				OrgUnitID: 2,
				Groups:    []string{"Lead"},
				IsActive:  true,
			},
		},
	}
	client := &reporterRapidProClient{}
	service := NewService(repo).
		WithRapidProIntegration(&reporterServerLookup{
			record: sukumadserver.Record{
				Code:    "rapidpro",
				BaseURL: "https://rapidpro.example.com",
			},
		}, client).
		WithRapidProSettings(reporterSettingsProvider{
			config: settings.RapidProReporterSyncSettings{
				Mappings: []settings.RapidProReporterFieldMapping{
					{SourceKey: "facilityName", SourceLabel: "Facility Name", RapidProFieldKey: "Facility"},
				},
				Validation: settings.RapidProReporterSyncValidation{IsValid: true},
			},
		}).
		WithOrgUnitLookup(reporterOrgUnitLookup{
			item: orgunit.OrgUnit{ID: 2, UID: "ou-2", Name: "Kampala Health Centre"},
		})

	preview, err := service.BuildRapidProReporterSyncPreview(context.Background(), 1)
	if err != nil {
		t.Fatalf("build preview: %v", err)
	}
	if preview.Reporter.SyncOperation != "created" {
		t.Fatalf("expected create preview operation, got %q", preview.Reporter.SyncOperation)
	}
	if preview.RequestPath != "/api/v2/contacts.json" {
		t.Fatalf("unexpected preview path %q", preview.RequestPath)
	}
	urns, ok := preview.RequestBody["urns"].([]string)
	if !ok || len(urns) != 1 || urns[0] != "tel:+256700000001" {
		t.Fatalf("expected preview urns, got %#v", preview.RequestBody["urns"])
	}
	fields, ok := preview.RequestBody["fields"].(map[string]string)
	if !ok || fields["Facility"] != "Kampala Health Centre" {
		t.Fatalf("expected preview facility field, got %#v", preview.RequestBody["fields"])
	}
	if len(preview.ResolvedGroups) != 1 || preview.ResolvedGroups[0].UUID != "group-Lead" {
		t.Fatalf("expected resolved group preview, got %#v", preview.ResolvedGroups)
	}
}

func TestGetRapidProContactDetailsUsesUUIDAndMapsSnapshot(t *testing.T) {
	repo := &reporterServiceRepo{
		byID: map[int64]Reporter{
			1: {
				ID:           1,
				Name:         "Alice Reporter",
				Telephone:    "+256700000001",
				OrgUnitID:    2,
				RapidProUUID: "contact-existing",
				IsActive:     true,
			},
		},
	}
	client := &reporterRapidProClient{
		lookupByUUIDFound: true,
		uuidContact: rapidpro.Contact{
			UUID:       "contact-existing",
			Name:       "Alice Reporter",
			Status:     "active",
			Language:   "eng",
			URNs:       []string{"tel:+256700000001"},
			Groups:     []rapidpro.Group{{UUID: "group-1", Name: "Lead"}},
			Fields:     map[string]string{"Facility": "Kampala Health Centre"},
			Flow:       &rapidpro.Flow{UUID: "flow-1", Name: "Registration"},
			CreatedOn:  "2026-04-20T09:00:00Z",
			ModifiedOn: "2026-04-22T09:00:00Z",
			LastSeenOn: "2026-04-23T09:00:00Z",
		},
	}
	service := NewService(repo).
		WithRapidProIntegration(&reporterServerLookup{
			record: sukumadserver.Record{
				Code:    "rapidpro",
				BaseURL: "https://rapidpro.example.com",
			},
		}, client)

	details, err := service.GetRapidProContactDetails(context.Background(), 1)
	if err != nil {
		t.Fatalf("get rapidpro contact details: %v", err)
	}
	if !details.Found || details.Contact == nil {
		t.Fatalf("expected remote contact to be found, got %+v", details)
	}
	if details.Contact.Status != "active" || details.Contact.Language != "eng" {
		t.Fatalf("unexpected contact snapshot: %+v", details.Contact)
	}
	if details.Contact.Flow == nil || details.Contact.Flow.Name != "Registration" {
		t.Fatalf("expected flow to be mapped, got %+v", details.Contact)
	}
	if details.Contact.Fields["Facility"] != "Kampala Health Centre" {
		t.Fatalf("expected custom fields to be preserved, got %+v", details.Contact.Fields)
	}
}

func TestGetRecentReportsForUserMatchesTelephoneAndFacility(t *testing.T) {
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
		recentReports: []sukumadrequest.Record{
			{
				ID:          88,
				UID:         "req-88",
				Status:      sukumadrequest.StatusCompleted,
				PayloadBody: `{"foo":"bar","value":"12345678901234567890ABCDE"}`,
				CreatedAt:   time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC),
			},
		},
	}
	service := NewService(repo).
		WithOrgUnitLookup(reporterOrgUnitLookup{
			item: orgunit.OrgUnit{ID: 2, Name: "Kampala Health Centre"},
		}).
		WithRecentReportsLookup(repo)

	result, err := service.GetRecentReportsForUser(context.Background(), 7, 1)
	if err != nil {
		t.Fatalf("get recent reports: %v", err)
	}
	if repo.recentReportsQuery == nil {
		t.Fatal("expected recent reports lookup query to be captured")
	}
	if repo.recentReportsQuery.MSISDN != "+256700000001" {
		t.Fatalf("expected msisdn query, got %+v", repo.recentReportsQuery)
	}
	if repo.recentReportsQuery.Facility != "Kampala Health Centre" {
		t.Fatalf("expected facility query, got %+v", repo.recentReportsQuery)
	}
	if repo.recentReportsQuery.Limit != 5 {
		t.Fatalf("expected limit 5, got %+v", repo.recentReportsQuery)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected one recent report, got %+v", result)
	}
	if result.Items[0].PayloadPreview != `{"foo":"bar","value"...` {
		t.Fatalf("expected truncated payload preview, got %q", result.Items[0].PayloadPreview)
	}
}

func TestGetRecentReportsForUserRespectsScope(t *testing.T) {
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
	service := NewService(repo).
		WithOrgUnitLookup(reporterOrgUnitLookup{
			item: orgunit.OrgUnit{ID: 2, Name: "Kampala Health Centre", Path: "/UG/Kampala/KHC/"},
		}).
		WithScopeResolver(reporterScopeResolverStub{
			scope: userorg.Scope{Restricted: true, PathPrefixes: []string{"/UG/Gulu/"}},
		})

	_, err := service.GetRecentReportsForUser(context.Background(), 7, 1)
	if err == nil {
		t.Fatal("expected scope error")
	}
	var typed *apperror.AppError
	if !errors.As(err, &typed) || typed.Code != apperror.CodeAuthForbidden {
		t.Fatalf("expected forbidden app error, got %v", err)
	}
}

func TestGetRapidProMessageHistoryCollectsRecentConversationAcrossPages(t *testing.T) {
	repo := &reporterServiceRepo{
		byID: map[int64]Reporter{
			1: {
				ID:           1,
				Name:         "Alice Reporter",
				Telephone:    "+256700000001",
				OrgUnitID:    2,
				RapidProUUID: "contact-existing",
				IsActive:     true,
			},
		},
	}
	client := &reporterRapidProClient{
		lookupByUUIDFound: true,
		uuidContact: rapidpro.Contact{
			UUID: "contact-existing",
			Name: "Alice Reporter",
			URNs: []string{"tel:+256700000001"},
		},
		messagePages: map[string]rapidProMessagePage{
			"": {
				messages: []rapidpro.Message{
					{
						ID:         11,
						Direction:  "outgoing",
						Status:     "sent",
						Text:       "Thanks",
						URN:        "tel:+256700000001",
						CreatedOn:  "2026-04-22T10:05:00Z",
						ModifiedOn: "2026-04-22T10:06:00Z",
						Contact:    rapidpro.Contact{UUID: "contact-existing", Name: "Alice Reporter"},
						Channel:    &rapidpro.Channel{UUID: "chan-1", Name: "Vonage"},
					},
					{
						ID:         12,
						Direction:  "incoming",
						Status:     "handled",
						Text:       "Ignore me",
						URN:        "tel:+256700000099",
						CreatedOn:  "2026-04-22T10:10:00Z",
						ModifiedOn: "2026-04-22T10:11:00Z",
						Contact:    rapidpro.Contact{UUID: "contact-other", Name: "Other"},
					},
				},
				next: "https://rapidpro.example.com/api/v2/messages.json?cursor=next-page",
			},
			"next-page": {
				messages: []rapidpro.Message{
					{
						ID:         10,
						Direction:  "incoming",
						Status:     "handled",
						Text:       "Hello there",
						URN:        "tel:+256700000001",
						CreatedOn:  "2026-04-22T10:00:00Z",
						ModifiedOn: "2026-04-22T10:01:00Z",
						Contact:    rapidpro.Contact{UUID: "contact-existing", Name: "Alice Reporter"},
					},
				},
				next: "https://rapidpro.example.com/api/v2/messages.json?cursor=more-history",
			},
			"more-history": {
				messages: nil,
				next:     "",
			},
		},
	}
	service := NewService(repo).
		WithRapidProIntegration(&reporterServerLookup{
			record: sukumadserver.Record{
				Code:    "rapidpro",
				BaseURL: "https://rapidpro.example.com",
			},
		}, client)

	history, err := service.GetRapidProMessageHistory(context.Background(), 1)
	if err != nil {
		t.Fatalf("get rapidpro message history: %v", err)
	}
	if !history.Found {
		t.Fatalf("expected reporter contact to be found")
	}
	if len(history.Items) != 2 {
		t.Fatalf("expected two matching messages, got %+v", history.Items)
	}
	if history.Items[0].Direction != "incoming" || history.Items[1].Direction != "outgoing" {
		t.Fatalf("unexpected directions: %+v", history.Items)
	}
	if history.Items[0].Text != "Hello there" || history.Items[1].Text != "Thanks" {
		t.Fatalf("expected oldest-to-newest ordering, got %+v", history.Items)
	}
	if history.Items[1].Channel == nil || history.Items[1].Channel.Name != "Vonage" {
		t.Fatalf("expected channel metadata, got %+v", history.Items[1])
	}
	if history.Next != "" {
		t.Fatalf("expected history to exhaust available test pages, got next %q", history.Next)
	}
	if client.messageCalls != 3 {
		t.Fatalf("expected three rapidpro message page calls, got %d", client.messageCalls)
	}
}

func TestGetRapidProMessageHistoryStopsAfterConfiguredHistoryCap(t *testing.T) {
	repo := &reporterServiceRepo{
		byID: map[int64]Reporter{
			1: {
				ID:           1,
				Name:         "Alice Reporter",
				Telephone:    "+256700000001",
				OrgUnitID:    2,
				RapidProUUID: "contact-existing",
				IsActive:     true,
			},
		},
	}
	pageMessages := make([]rapidpro.Message, 0, 10)
	for index := 0; index < 10; index++ {
		pageMessages = append(pageMessages, rapidpro.Message{
			ID:         int64(index + 1),
			Direction:  "incoming",
			Status:     "handled",
			Text:       fmt.Sprintf("Message %02d", index+1),
			URN:        "tel:+256700000001",
			CreatedOn:  fmt.Sprintf("2026-04-22T10:%02d:00Z", index),
			ModifiedOn: fmt.Sprintf("2026-04-22T10:%02d:30Z", index),
			Contact:    rapidpro.Contact{UUID: "contact-existing", Name: "Alice Reporter"},
		})
	}
	client := &reporterRapidProClient{
		lookupByUUIDFound: true,
		uuidContact: rapidpro.Contact{
			UUID: "contact-existing",
			Name: "Alice Reporter",
			URNs: []string{"tel:+256700000001"},
		},
		messagePages: map[string]rapidProMessagePage{
			"":       {messages: append([]rapidpro.Message(nil), pageMessages...), next: "https://rapidpro.example.com/api/v2/messages.json?cursor=page-2"},
			"page-2": {messages: append([]rapidpro.Message(nil), pageMessages...), next: "https://rapidpro.example.com/api/v2/messages.json?cursor=page-3"},
			"page-3": {messages: append([]rapidpro.Message(nil), pageMessages...), next: "https://rapidpro.example.com/api/v2/messages.json?cursor=page-4"},
			"page-4": {messages: append([]rapidpro.Message(nil), pageMessages...), next: "https://rapidpro.example.com/api/v2/messages.json?cursor=page-5"},
			"page-5": {messages: append([]rapidpro.Message(nil), pageMessages...), next: "https://rapidpro.example.com/api/v2/messages.json?cursor=page-6"},
			"page-6": {messages: append([]rapidpro.Message(nil), pageMessages...), next: ""},
		},
	}
	service := NewService(repo).
		WithRapidProIntegration(&reporterServerLookup{
			record: sukumadserver.Record{
				Code:    "rapidpro",
				BaseURL: "https://rapidpro.example.com",
			},
		}, client)

	history, err := service.GetRapidProMessageHistory(context.Background(), 1)
	if err != nil {
		t.Fatalf("get rapidpro message history: %v", err)
	}
	if len(history.Items) != maxRapidProHistoryItems {
		t.Fatalf("expected history to be capped at %d items, got %d", maxRapidProHistoryItems, len(history.Items))
	}
	if client.messageCalls != maxRapidProHistoryPages {
		t.Fatalf("expected %d rapidpro page calls, got %d", maxRapidProHistoryPages, client.messageCalls)
	}
	if history.Next != "https://rapidpro.example.com/api/v2/messages.json?cursor=page-6" {
		t.Fatalf("expected next cursor to be preserved")
	}
}

func TestGetRapidProMessageHistoryReturnsEmptyWhenContactMissing(t *testing.T) {
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
	service := NewService(repo).
		WithRapidProIntegration(&reporterServerLookup{
			record: sukumadserver.Record{
				Code:    "rapidpro",
				BaseURL: "https://rapidpro.example.com",
			},
		}, &reporterRapidProClient{})

	history, err := service.GetRapidProMessageHistory(context.Background(), 1)
	if err != nil {
		t.Fatalf("get rapidpro message history: %v", err)
	}
	if history.Found {
		t.Fatalf("expected missing remote contact")
	}
	if len(history.Items) != 0 {
		t.Fatalf("expected no history items, got %+v", history.Items)
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

func TestSyncUpdatedSinceEmitsBatchAndReporterFailureLogs(t *testing.T) {
	now := time.Date(2026, time.April, 24, 12, 0, 0, 0, time.UTC)
	logOutput := captureReporterLogs(t)
	repo := &reporterServiceRepo{
		byID: map[int64]Reporter{
			1: {ID: 1, Name: "Alice", Telephone: "+256700000001", OrgUnitID: 2, IsActive: true},
		},
		updatedSinceItems: []Reporter{
			{ID: 1, Name: "Alice", Telephone: "+256700000001", OrgUnitID: 2, IsActive: true},
		},
	}
	service := NewService(repo).
		WithClock(func() time.Time { return now }).
		WithRapidProIntegration(&reporterServerLookup{
			record: sukumadserver.Record{
				Code:    "rapidpro",
				BaseURL: "https://rapidpro.example.com",
			},
		}, &reporterRapidProClientWithUpsertError{
			delegate: &reporterRapidProClient{},
			err:      &rapidpro.RequestError{StatusCode: 400, Body: `{"fields":["Facility is invalid"]}`},
		}).
		WithRapidProSettings(reporterSettingsProvider{
			config: settings.RapidProReporterSyncSettings{
				Validation: settings.RapidProReporterSyncValidation{IsValid: true},
			},
		})

	result, err := service.SyncUpdatedSince(context.Background(), nil, 50, true, false)
	if err == nil {
		t.Fatal("expected sync updated since to fail")
	}
	if result.Failed != 1 {
		t.Fatalf("expected one failed reporter, got %+v", result)
	}
	assertReporterLogContains(t, logOutput.String(),
		"rapidpro_reporter_sync_scan_started",
		"\"requested_count\":1",
		"rapidpro_reporter_sync_reporter_failed",
		"\"reporter_name\":\"Alice\"",
		"rapidpro_reporter_sync_scan_failed",
		"\"failed_count\":1",
	)
}

func captureReporterLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var logOutput bytes.Buffer
	logging.SetOutput(&logOutput)
	logging.ApplyConfig(logging.Config{Level: "info", Format: "json"})
	t.Cleanup(func() {
		logging.SetOutput(nil)
		logging.ApplyConfig(logging.Config{Level: "info", Format: "console"})
	})
	return &logOutput
}

func assertReporterLogContains(t *testing.T, logs string, fragments ...string) {
	t.Helper()
	for _, fragment := range fragments {
		if !strings.Contains(logs, fragment) {
			t.Fatalf("expected logs to contain %q, got:\n%s", fragment, logs)
		}
	}
}
