package reporter

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"sort"
	"strings"
	"time"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/audit"
	"basepro/backend/internal/logging"
	"basepro/backend/internal/settings"
	"basepro/backend/internal/sukumad/orgunit"
	"basepro/backend/internal/sukumad/rapidex/rapidpro"
	"basepro/backend/internal/sukumad/reportergroup"
	sukumadrequest "basepro/backend/internal/sukumad/request"
	sukumadserver "basepro/backend/internal/sukumad/server"
	"basepro/backend/internal/sukumad/userorg"
)

const (
	defaultRapidProServerCode = "rapidpro"
	maxRapidProHistoryItems   = 50
	maxRapidProHistoryPages   = 5
	broadcastDuplicateWindow  = 15 * time.Minute
	broadcastClaimTimeout     = time.Minute
)

type rapidProServerLookup interface {
	GetServerByCode(context.Context, string) (sukumadserver.Record, error)
}

type rapidProClient interface {
	LookupContactByUUID(context.Context, rapidpro.Connection, string) (rapidpro.Contact, bool, error)
	LookupContactByURN(context.Context, rapidpro.Connection, string) (rapidpro.Contact, bool, error)
	UpsertContact(context.Context, rapidpro.Connection, rapidpro.UpsertContactInput) (rapidpro.Contact, error)
	LookupGroupByName(context.Context, rapidpro.Connection, string) (rapidpro.Group, bool, error)
	CreateGroup(context.Context, rapidpro.Connection, string) (rapidpro.Group, error)
	SendMessage(context.Context, rapidpro.Connection, string, string) (rapidpro.Message, error)
	SendBroadcast(context.Context, rapidpro.Connection, []string, string) (rapidpro.Broadcast, error)
	ListMessages(context.Context, rapidpro.Connection, map[string]string) ([]rapidpro.Message, string, error)
}

type rapidProReporterSyncSettingsProvider interface {
	GetRapidProReporterSync(context.Context) (settings.RapidProReporterSyncSettings, error)
}

type orgUnitLookup interface {
	Get(context.Context, int64) (orgunit.OrgUnit, error)
}

type reporterGroupCatalog interface {
	ValidateActiveNames(context.Context, []string) ([]string, error)
	EnsureRapidProGroups(context.Context, []string) ([]rapidpro.Group, error)
}

type reporterScopeResolver interface {
	ResolveScope(context.Context, int64) (userorg.Scope, error)
}

type reporterRecentReportsLookup interface {
	ListRecentReporterReports(context.Context, sukumadrequest.ReporterRecentReportsQuery) ([]sukumadrequest.Record, error)
}

// Service encapsulates business logic for reporters and depends on a Repository.
type Service struct {
	repo             Repository
	auditService     *audit.Service
	serverLookup     rapidProServerLookup
	rapidProClient   rapidProClient
	rapidProSettings rapidProReporterSyncSettingsProvider
	orgUnitLookup    orgUnitLookup
	groupCatalog     reporterGroupCatalog
	scopeResolver    reporterScopeResolver
	recentReports    reporterRecentReportsLookup
	clock            func() time.Time
}

// NewService constructs a new Service with the provided repository.
func NewService(repo Repository, auditService ...*audit.Service) *Service {
	var auditSvc *audit.Service
	if len(auditService) > 0 {
		auditSvc = auditService[0]
	}
	return &Service{
		repo:         repo,
		auditService: auditSvc,
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

func (s *Service) WithRapidProSettings(provider rapidProReporterSyncSettingsProvider) *Service {
	if s == nil {
		return s
	}
	s.rapidProSettings = provider
	return s
}

func (s *Service) WithOrgUnitLookup(lookup orgUnitLookup) *Service {
	if s == nil {
		return s
	}
	s.orgUnitLookup = lookup
	return s
}

func (s *Service) WithClock(clock func() time.Time) *Service {
	if s == nil || clock == nil {
		return s
	}
	s.clock = clock
	return s
}

func (s *Service) WithReporterGroupCatalog(catalog reporterGroupCatalog) *Service {
	if s == nil {
		return s
	}
	s.groupCatalog = catalog
	return s
}

func (s *Service) WithScopeResolver(resolver reporterScopeResolver) *Service {
	if s == nil {
		return s
	}
	s.scopeResolver = resolver
	return s
}

func (s *Service) WithRecentReportsLookup(lookup reporterRecentReportsLookup) *Service {
	if s == nil {
		return s
	}
	s.recentReports = lookup
	return s
}

// List returns a page of reporters matching the provided query.
func (s *Service) List(ctx context.Context, query ListQuery) (ListResult, error) {
	return s.repo.List(ctx, query)
}

func (s *Service) ListBroadcasts(ctx context.Context, query BroadcastListQuery) (BroadcastListResult, error) {
	return s.repo.ListBroadcasts(ctx, query)
}

func (s *Service) ListForUser(ctx context.Context, userID int64, query ListQuery) (ListResult, error) {
	scope, err := s.resolveScope(ctx, userID)
	if err != nil {
		return ListResult{}, err
	}
	if scope.Restricted {
		query.ScopeRestricted = true
		query.ScopePaths = append([]string(nil), scope.PathPrefixes...)
	}
	return s.repo.List(ctx, query)
}

// Get fetches a reporter by ID.
func (s *Service) Get(ctx context.Context, id int64) (Reporter, error) {
	reporter, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Reporter{}, mapReporterLookupError(err)
	}
	return reporter, nil
}

func (s *Service) GetForUser(ctx context.Context, userID, id int64) (Reporter, error) {
	scope, err := s.resolveScope(ctx, userID)
	if err != nil {
		return Reporter{}, err
	}
	reporter, err := s.Get(ctx, id)
	if err != nil {
		return Reporter{}, err
	}
	if err := s.ensureReporterInScope(ctx, scope, reporter); err != nil {
		return Reporter{}, err
	}
	return reporter, nil
}

func (s *Service) ListRapidProReporterSyncPreviewReporters(ctx context.Context) ([]settings.RapidProReporterOption, error) {
	result, err := s.repo.List(ctx, ListQuery{Page: 0, PageSize: 200})
	if err != nil {
		return nil, err
	}
	options := make([]settings.RapidProReporterOption, 0, len(result.Items))
	for _, reporter := range result.Items {
		options = append(options, settings.RapidProReporterOption{
			ID:   reporter.ID,
			Name: reporter.Name,
		})
	}
	return options, nil
}

func (s *Service) BuildRapidProReporterSyncPreview(ctx context.Context, id int64) (settings.RapidProReporterSyncPreview, error) {
	reporter, err := s.Get(ctx, id)
	if err != nil {
		return settings.RapidProReporterSyncPreview{}, err
	}
	built, err := s.buildRapidProSync(ctx, reporter)
	if err != nil {
		return settings.RapidProReporterSyncPreview{}, err
	}
	preview := settings.RapidProReporterSyncPreview{
		Reporter: settings.RapidProReporterSyncPreviewReporter{
			ID:            reporter.ID,
			Name:          reporter.Name,
			Telephone:     reporter.Telephone,
			RapidProUUID:  reporter.RapidProUUID,
			Groups:        append([]string(nil), reporter.Groups...),
			FacilityName:  built.ContactData.FacilityName,
			FacilityUID:   built.ContactData.FacilityUID,
			SyncOperation: built.Operation,
		},
		RequestPath:    "/api/v2/contacts.json",
		RequestQuery:   built.RequestQuery,
		RequestBody:    built.RequestBody,
		ResolvedGroups: built.ResolvedGroups,
	}
	return preview, nil
}

func (s *Service) GetRapidProContactDetails(ctx context.Context, id int64) (RapidProContactDetailsResult, error) {
	reporter, err := s.Get(ctx, id)
	if err != nil {
		return RapidProContactDetailsResult{}, err
	}
	conn, err := s.rapidProConnection(ctx)
	if err != nil {
		return RapidProContactDetailsResult{}, err
	}
	contact, found, err := s.lookupRapidProContact(ctx, conn, reporter)
	if err != nil {
		return RapidProContactDetailsResult{}, err
	}
	result := RapidProContactDetailsResult{
		Reporter: reporter,
		Found:    found,
	}
	if found {
		snapshot := toRapidProContactSnapshot(contact)
		result.Contact = &snapshot
	}
	return result, nil
}

func (s *Service) GetRapidProMessageHistory(ctx context.Context, id int64) (RapidProMessageHistoryResult, error) {
	reporter, err := s.Get(ctx, id)
	if err != nil {
		return RapidProMessageHistoryResult{}, err
	}
	conn, err := s.rapidProConnection(ctx)
	if err != nil {
		return RapidProMessageHistoryResult{}, err
	}
	contact, found, err := s.lookupRapidProContact(ctx, conn, reporter)
	if err != nil {
		return RapidProMessageHistoryResult{}, err
	}
	result := RapidProMessageHistoryResult{
		Reporter: reporter,
		Found:    found,
		Items:    []RapidProMessageRecord{},
	}
	if !found {
		return result, nil
	}
	targetURNs := make(map[string]struct{}, len(contact.URNs))
	for _, urn := range contact.URNs {
		normalized := strings.ToLower(strings.TrimSpace(urn))
		if normalized == "" {
			continue
		}
		targetURNs[normalized] = struct{}{}
	}
	var (
		query        map[string]string
		next         string
		pagesScanned int
	)
	for pagesScanned < maxRapidProHistoryPages && len(result.Items) < maxRapidProHistoryItems {
		messages, pageNext, err := s.rapidProClient.ListMessages(ctx, conn, query)
		if err != nil {
			return RapidProMessageHistoryResult{}, mapRapidProRequestError(err)
		}
		pagesScanned++
		next = pageNext
		for _, message := range messages {
			if !messageMatchesReporter(message, contact.UUID, targetURNs) {
				continue
			}
			result.Items = append(result.Items, toRapidProMessageRecord(message))
			if len(result.Items) >= maxRapidProHistoryItems {
				break
			}
		}
		if next == "" {
			break
		}
		nextQuery, ok := rapidProNextQuery(next)
		if !ok {
			break
		}
		query = nextQuery
	}
	sort.SliceStable(result.Items, func(i, j int) bool {
		left := messageRecordSortTime(result.Items[i])
		right := messageRecordSortTime(result.Items[j])
		if left.Equal(right) {
			return result.Items[i].ID < result.Items[j].ID
		}
		return left.Before(right)
	})
	result.Next = next
	return result, nil
}

// Create validates and persists a new Reporter.
func (s *Service) Create(ctx context.Context, r Reporter) (Reporter, error) {
	if err := validateReporter(r, false); err != nil {
		return Reporter{}, err
	}
	if s.groupCatalog != nil {
		validated, err := s.groupCatalog.ValidateActiveNames(ctx, r.Groups)
		if err != nil {
			return Reporter{}, err
		}
		r.Groups = validated
	}
	now := s.clock()
	r.CreatedAt = now
	r.UpdatedAt = now
	r.Groups = normalizeGroups(r.Groups)
	return s.repo.Create(ctx, r)
}

func (s *Service) CreateForUser(ctx context.Context, userID int64, r Reporter) (Reporter, error) {
	scope, err := s.resolveScope(ctx, userID)
	if err != nil {
		return Reporter{}, err
	}
	if err := s.ensureOrgUnitInScope(ctx, scope, r.OrgUnitID); err != nil {
		return Reporter{}, err
	}
	return s.Create(ctx, r)
}

// Update validates and updates an existing reporter.
func (s *Service) Update(ctx context.Context, r Reporter) (Reporter, error) {
	if err := validateReporter(r, true); err != nil {
		return Reporter{}, err
	}
	if s.groupCatalog != nil {
		validated, err := s.groupCatalog.ValidateActiveNames(ctx, r.Groups)
		if err != nil {
			return Reporter{}, err
		}
		r.Groups = validated
	}
	r.Groups = normalizeGroups(r.Groups)
	r.UpdatedAt = s.clock()
	updated, err := s.repo.Update(ctx, r)
	if err != nil {
		return Reporter{}, mapReporterLookupError(err)
	}
	return updated, nil
}

func (s *Service) UpdateForUser(ctx context.Context, userID int64, r Reporter) (Reporter, error) {
	scope, err := s.resolveScope(ctx, userID)
	if err != nil {
		return Reporter{}, err
	}
	existing, err := s.Get(ctx, r.ID)
	if err != nil {
		return Reporter{}, err
	}
	if err := s.ensureReporterInScope(ctx, scope, existing); err != nil {
		return Reporter{}, err
	}
	if err := s.ensureOrgUnitInScope(ctx, scope, r.OrgUnitID); err != nil {
		return Reporter{}, err
	}
	return s.Update(ctx, r)
}

// Delete removes a reporter by ID.
func (s *Service) Delete(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return mapReporterLookupError(err)
	}
	return nil
}

func (s *Service) DeleteForUser(ctx context.Context, userID, id int64) error {
	if _, err := s.GetForUser(ctx, userID, id); err != nil {
		return err
	}
	return s.Delete(ctx, id)
}

func (s *Service) SyncReporter(ctx context.Context, id int64) (SyncResult, error) {
	reporter, err := s.Get(ctx, id)
	if err != nil {
		return SyncResult{}, err
	}
	return s.syncReporter(ctx, reporter)
}

func (s *Service) SyncReporterForUser(ctx context.Context, userID, id int64) (SyncResult, error) {
	reporter, err := s.GetForUser(ctx, userID, id)
	if err != nil {
		return SyncResult{}, err
	}
	return s.syncReporter(ctx, reporter)
}

func (s *Service) SyncReporters(ctx context.Context, ids []int64) (SyncBatchResult, error) {
	reporters, err := s.loadByIDs(ctx, ids)
	if err != nil {
		return SyncBatchResult{}, err
	}
	return s.syncReporterBatch(ctx, reporters, false, nil, false)
}

func (s *Service) SyncReportersForUser(ctx context.Context, userID int64, ids []int64) (SyncBatchResult, error) {
	reporters, err := s.loadByIDsForUser(ctx, userID, ids)
	if err != nil {
		return SyncBatchResult{}, err
	}
	return s.syncReporterBatch(ctx, reporters, false, nil, false)
}

func (s *Service) SyncUpdatedSince(ctx context.Context, since *time.Time, limit int, onlyActive bool, dryRun bool) (SyncBatchResult, error) {
	reporters, err := s.repo.ListUpdatedSince(ctx, since, limit, onlyActive)
	if err != nil {
		return SyncBatchResult{}, err
	}
	return s.syncReporterBatch(ctx, reporters, dryRun, since, onlyActive)
}

func (s *Service) SendMessage(ctx context.Context, id int64, text string) (MessageResult, error) {
	message := strings.TrimSpace(text)
	if message == "" {
		return MessageResult{}, apperror.ValidationWithDetails("validation failed", map[string]any{"text": []string{"is required"}})
	}
	reporter, err := s.Get(ctx, id)
	if err != nil {
		return MessageResult{}, err
	}
	syncResult, err := s.syncReporter(ctx, reporter)
	if err != nil {
		return MessageResult{}, err
	}
	conn, err := s.rapidProConnection(ctx)
	if err != nil {
		return MessageResult{}, err
	}
	if _, err := s.rapidProClient.SendMessage(ctx, conn, syncResult.Reporter.RapidProUUID, message); err != nil {
		return MessageResult{}, mapRapidProRequestError(err)
	}
	s.logAudit(ctx, audit.Event{
		Action:     "reporter.message.sent",
		EntityType: "reporter",
		EntityID:   strPtr(fmt.Sprintf("%d", syncResult.Reporter.ID)),
		Metadata: map[string]any{
			"name":         syncResult.Reporter.Name,
			"rapidProUuid": syncResult.Reporter.RapidProUUID,
		},
	})
	return MessageResult{Reporter: syncResult.Reporter, Message: message}, nil
}

func (s *Service) SendMessageForUser(ctx context.Context, userID, id int64, text string) (MessageResult, error) {
	message := strings.TrimSpace(text)
	if message == "" {
		return MessageResult{}, apperror.ValidationWithDetails("validation failed", map[string]any{"text": []string{"is required"}})
	}
	reporter, err := s.GetForUser(ctx, userID, id)
	if err != nil {
		return MessageResult{}, err
	}
	syncResult, err := s.syncReporter(ctx, reporter)
	if err != nil {
		return MessageResult{}, err
	}
	conn, err := s.rapidProConnection(ctx)
	if err != nil {
		return MessageResult{}, err
	}
	if _, err := s.rapidProClient.SendMessage(ctx, conn, syncResult.Reporter.RapidProUUID, message); err != nil {
		return MessageResult{}, mapRapidProRequestError(err)
	}
	return MessageResult{Reporter: syncResult.Reporter, Message: message}, nil
}

func (s *Service) BroadcastMessage(ctx context.Context, ids []int64, text string) (BroadcastResult, error) {
	message := strings.TrimSpace(text)
	if message == "" {
		return BroadcastResult{}, apperror.ValidationWithDetails("validation failed", map[string]any{"text": []string{"is required"}})
	}
	reporters, err := s.loadByIDs(ctx, ids)
	if err != nil {
		return BroadcastResult{}, err
	}
	syncResult, err := s.syncReporterBatch(ctx, reporters, false, nil, false)
	if err != nil {
		return BroadcastResult{}, err
	}
	if len(syncResult.Reporters) == 0 {
		return BroadcastResult{}, apperror.ValidationWithDetails("validation failed", map[string]any{"reporterIds": []string{"at least one reporter is required"}})
	}
	conn, err := s.rapidProConnection(ctx)
	if err != nil {
		return BroadcastResult{}, err
	}
	contactUUIDs := make([]string, 0, len(syncResult.Reporters))
	reporterIDs := make([]int64, 0, len(syncResult.Reporters))
	for _, reporter := range syncResult.Reporters {
		contactUUIDs = append(contactUUIDs, reporter.RapidProUUID)
		reporterIDs = append(reporterIDs, reporter.ID)
	}
	if _, err := s.rapidProClient.SendBroadcast(ctx, conn, contactUUIDs, message); err != nil {
		return BroadcastResult{}, mapRapidProRequestError(err)
	}
	s.logAudit(ctx, audit.Event{
		Action:     "reporter.broadcast.sent",
		EntityType: "reporter",
		Metadata: map[string]any{
			"reporterIds": reporterIDs,
			"count":       len(reporterIDs),
		},
	})
	return BroadcastResult{ReporterIDs: reporterIDs, Message: message}, nil
}

func (s *Service) BroadcastMessageForUser(ctx context.Context, userID int64, ids []int64, text string) (BroadcastResult, error) {
	message := strings.TrimSpace(text)
	if message == "" {
		return BroadcastResult{}, apperror.ValidationWithDetails("validation failed", map[string]any{"text": []string{"is required"}})
	}
	reporters, err := s.loadByIDsForUser(ctx, userID, ids)
	if err != nil {
		return BroadcastResult{}, err
	}
	syncResult, err := s.syncReporterBatch(ctx, reporters, false, nil, false)
	if err != nil {
		return BroadcastResult{}, err
	}
	if len(syncResult.Reporters) == 0 {
		return BroadcastResult{}, apperror.ValidationWithDetails("validation failed", map[string]any{"reporterIds": []string{"at least one reporter is required"}})
	}
	conn, err := s.rapidProConnection(ctx)
	if err != nil {
		return BroadcastResult{}, err
	}
	contactUUIDs := make([]string, 0, len(syncResult.Reporters))
	reporterIDs := make([]int64, 0, len(syncResult.Reporters))
	for _, reporter := range syncResult.Reporters {
		contactUUIDs = append(contactUUIDs, reporter.RapidProUUID)
		reporterIDs = append(reporterIDs, reporter.ID)
	}
	if _, err := s.rapidProClient.SendBroadcast(ctx, conn, contactUUIDs, message); err != nil {
		return BroadcastResult{}, mapRapidProRequestError(err)
	}
	return BroadcastResult{ReporterIDs: reporterIDs, Message: message}, nil
}

func (s *Service) QueueJurisdictionBroadcastForUser(ctx context.Context, userID int64, input JurisdictionBroadcastInput) (JurisdictionBroadcastQueueResult, error) {
	message := strings.TrimSpace(input.Text)
	if message == "" {
		return JurisdictionBroadcastQueueResult{}, apperror.ValidationWithDetails("validation failed", map[string]any{"text": []string{"is required"}})
	}
	group := strings.TrimSpace(input.ReporterGroup)
	if group == "" {
		return JurisdictionBroadcastQueueResult{}, apperror.ValidationWithDetails("validation failed", map[string]any{"reporterGroup": []string{"is required"}})
	}
	orgUnitIDs := normalizeInt64IDs(input.OrgUnitIDs)
	if len(orgUnitIDs) == 0 {
		return JurisdictionBroadcastQueueResult{}, apperror.ValidationWithDetails("validation failed", map[string]any{"orgUnitIds": []string{"at least one organisation unit is required"}})
	}
	if s.groupCatalog != nil {
		validGroups, err := s.groupCatalog.ValidateActiveNames(ctx, []string{group})
		if err != nil {
			return JurisdictionBroadcastQueueResult{}, err
		}
		if len(validGroups) == 0 {
			return JurisdictionBroadcastQueueResult{}, apperror.ValidationWithDetails("validation failed", map[string]any{"reporterGroup": []string{"must be an active reporter group"}})
		}
		group = validGroups[0]
	}

	scope, err := s.resolveScope(ctx, userID)
	if err != nil {
		return JurisdictionBroadcastQueueResult{}, err
	}
	paths, err := s.broadcastOrgUnitPaths(ctx, scope, orgUnitIDs)
	if err != nil {
		return JurisdictionBroadcastQueueResult{}, err
	}
	matchedCount, err := s.repo.CountBroadcastRecipients(ctx, BroadcastRecipientQuery{
		OrgUnitPaths:  paths,
		ReporterGroup: group,
		OnlyActive:    true,
	})
	if err != nil {
		return JurisdictionBroadcastQueueResult{}, err
	}
	if matchedCount == 0 {
		return JurisdictionBroadcastQueueResult{}, apperror.ValidationWithDetails("validation failed", map[string]any{
			"orgUnitIds":    []string{"No active reporters matched the selected organisation units and reporter group"},
			"reporterGroup": []string{"No active reporters matched the selected organisation units and reporter group"},
		})
	}

	dedupeKey := jurisdictionBroadcastDedupeKey(userID, orgUnitIDs, group, message)
	since := s.clock().Add(-broadcastDuplicateWindow)
	existing, err := s.repo.GetRecentPendingBroadcastByDedupeKey(ctx, dedupeKey, since)
	if err == nil {
		return JurisdictionBroadcastQueueResult{
			Status:    BroadcastQueueResultDuplicate,
			Message:   "An identical reporter broadcast is already being processed. Please wait for it to finish.",
			Broadcast: existing,
		}, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return JurisdictionBroadcastQueueResult{}, err
	}

	now := s.clock()
	record, err := s.repo.CreateJurisdictionBroadcast(ctx, JurisdictionBroadcastRecord{
		UID:               newUID(),
		RequestedByUserID: userID,
		OrgUnitIDs:        orgUnitIDs,
		ReporterGroup:     group,
		MessageText:       message,
		DedupeKey:         dedupeKey,
		MatchedCount:      matchedCount,
		Status:            BroadcastStatusQueued,
		RequestedAt:       now,
		CreatedAt:         now,
		UpdatedAt:         now,
	})
	if err != nil {
		return JurisdictionBroadcastQueueResult{}, err
	}
	s.logAudit(ctx, audit.Event{
		Action:     "reporter.broadcast.queued",
		EntityType: "reporter_broadcast",
		EntityID:   strPtr(fmt.Sprintf("%d", record.ID)),
		Metadata: map[string]any{
			"requestedByUserId": userID,
			"orgUnitIds":        append([]int64(nil), record.OrgUnitIDs...),
			"reporterGroup":     record.ReporterGroup,
			"matchedCount":      record.MatchedCount,
		},
	})
	return JurisdictionBroadcastQueueResult{
		Status:    BroadcastQueueResultQueued,
		Message:   "Reporter broadcast queued. Delivery will continue in the background.",
		Broadcast: record,
	}, nil
}

func (s *Service) RunQueuedBroadcasts(ctx context.Context, workerRunID int64, observe func(string, int), batchSize int, claimTimeout time.Duration) error {
	if batchSize <= 0 {
		batchSize = 10
	}
	if claimTimeout <= 0 {
		claimTimeout = broadcastClaimTimeout
	}
	for i := 0; i < batchSize; i++ {
		record, err := s.repo.ClaimNextJurisdictionBroadcast(ctx, s.clock(), claimTimeout, workerRunID)
		if err != nil {
			if errors.Is(err, ErrNoEligibleBroadcast) {
				return nil
			}
			return err
		}
		if observe != nil {
			observe("claimed", 1)
		}
		if err := s.processJurisdictionBroadcast(ctx, record, observe); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) processJurisdictionBroadcast(ctx context.Context, record JurisdictionBroadcastRecord, observe func(string, int)) error {
	paths, err := s.broadcastOrgUnitPaths(ctx, userorg.Scope{}, record.OrgUnitIDs)
	if err != nil {
		if _, updateErr := s.repo.UpdateJurisdictionBroadcastResult(ctx, record.ID, BroadcastStatusFailed, 0, record.MatchedCount, err.Error(), s.clock()); updateErr != nil {
			return errors.Join(err, updateErr)
		}
		return nil
	}
	reporters, err := s.repo.ListBroadcastRecipients(ctx, BroadcastRecipientQuery{
		OrgUnitPaths:  paths,
		ReporterGroup: record.ReporterGroup,
		OnlyActive:    true,
	})
	if err != nil {
		if _, updateErr := s.repo.UpdateJurisdictionBroadcastResult(ctx, record.ID, BroadcastStatusFailed, 0, record.MatchedCount, err.Error(), s.clock()); updateErr != nil {
			return errors.Join(err, updateErr)
		}
		return nil
	}
	if len(reporters) == 0 {
		_, err = s.repo.UpdateJurisdictionBroadcastResult(ctx, record.ID, BroadcastStatusFailed, 0, 0, "No active reporters matched at processing time.", s.clock())
		return err
	}

	syncResult, syncErr := s.syncReporterBatch(ctx, reporters, false, nil, true)
	if len(syncResult.Reporters) == 0 {
		errText := "No reporters could be synchronized for broadcast delivery."
		if syncErr != nil {
			errText = syncErr.Error()
		}
		_, err = s.repo.UpdateJurisdictionBroadcastResult(ctx, record.ID, BroadcastStatusFailed, 0, len(reporters), errText, s.clock())
		return err
	}

	conn, err := s.rapidProConnection(ctx)
	if err != nil {
		if _, updateErr := s.repo.UpdateJurisdictionBroadcastResult(ctx, record.ID, BroadcastStatusFailed, 0, len(reporters), err.Error(), s.clock()); updateErr != nil {
			return errors.Join(err, updateErr)
		}
		return nil
	}
	contactUUIDs := make([]string, 0, len(syncResult.Reporters))
	for _, reporter := range syncResult.Reporters {
		if uuid := strings.TrimSpace(reporter.RapidProUUID); uuid != "" {
			contactUUIDs = append(contactUUIDs, uuid)
		}
	}
	if len(contactUUIDs) == 0 {
		_, err = s.repo.UpdateJurisdictionBroadcastResult(ctx, record.ID, BroadcastStatusFailed, 0, len(reporters), "No synchronized RapidPro contacts were available for broadcast delivery.", s.clock())
		return err
	}
	if _, err := s.rapidProClient.SendBroadcast(ctx, conn, contactUUIDs, record.MessageText); err != nil {
		mapped := mapRapidProRequestError(err)
		if _, updateErr := s.repo.UpdateJurisdictionBroadcastResult(ctx, record.ID, BroadcastStatusFailed, 0, len(reporters), mapped.Error(), s.clock()); updateErr != nil {
			return errors.Join(mapped, updateErr)
		}
		return nil
	}
	if observe != nil {
		observe("completed", 1)
	}
	failedCount := len(reporters) - len(contactUUIDs)
	if syncResult.Failed > failedCount {
		failedCount = syncResult.Failed
	}
	if _, err := s.repo.UpdateJurisdictionBroadcastResult(ctx, record.ID, BroadcastStatusCompleted, len(contactUUIDs), failedCount, "", s.clock()); err != nil {
		return err
	}
	s.logAudit(ctx, audit.Event{
		Action:     "reporter.broadcast.sent.background",
		EntityType: "reporter_broadcast",
		EntityID:   strPtr(fmt.Sprintf("%d", record.ID)),
		Metadata: map[string]any{
			"matchedCount":  len(reporters),
			"sentCount":     len(contactUUIDs),
			"failedCount":   failedCount,
			"reporterGroup": record.ReporterGroup,
		},
	})
	return nil
}

func (s *Service) GetRapidProContactDetailsForUser(ctx context.Context, userID, id int64) (RapidProContactDetailsResult, error) {
	reporter, err := s.GetForUser(ctx, userID, id)
	if err != nil {
		return RapidProContactDetailsResult{}, err
	}
	conn, err := s.rapidProConnection(ctx)
	if err != nil {
		return RapidProContactDetailsResult{}, err
	}
	contact, found, err := s.lookupRapidProContact(ctx, conn, reporter)
	if err != nil {
		return RapidProContactDetailsResult{}, err
	}
	result := RapidProContactDetailsResult{Reporter: reporter, Found: found}
	if found {
		snapshot := toRapidProContactSnapshot(contact)
		result.Contact = &snapshot
	}
	return result, nil
}

func (s *Service) GetRapidProMessageHistoryForUser(ctx context.Context, userID, id int64) (RapidProMessageHistoryResult, error) {
	reporter, err := s.GetForUser(ctx, userID, id)
	if err != nil {
		return RapidProMessageHistoryResult{}, err
	}
	return s.getRapidProMessageHistoryForReporter(ctx, reporter)
}

func (s *Service) GetRecentReportsForUser(ctx context.Context, userID, id int64) (ReporterRecentReportsResult, error) {
	reporter, err := s.GetForUser(ctx, userID, id)
	if err != nil {
		return ReporterRecentReportsResult{}, err
	}
	if s.recentReports == nil {
		return ReporterRecentReportsResult{Reporter: reporter, Items: []ReporterRecentReportRecord{}}, nil
	}
	facilityName := ""
	if s.orgUnitLookup != nil && reporter.OrgUnitID > 0 {
		unit, lookupErr := s.orgUnitLookup.Get(ctx, reporter.OrgUnitID)
		if lookupErr != nil {
			return ReporterRecentReportsResult{}, lookupErr
		}
		facilityName = strings.TrimSpace(unit.Name)
	}
	items, err := s.recentReports.ListRecentReporterReports(ctx, sukumadrequest.ReporterRecentReportsQuery{
		MSISDN:   strings.TrimSpace(reporter.Telephone),
		Facility: facilityName,
		Limit:    5,
	})
	if err != nil {
		return ReporterRecentReportsResult{}, err
	}
	resultItems := make([]ReporterRecentReportRecord, 0, len(items))
	for _, item := range items {
		resultItems = append(resultItems, ReporterRecentReportRecord{
			ID:             item.ID,
			UID:            item.UID,
			Status:         item.Status,
			CreatedAt:      item.CreatedAt,
			PayloadBody:    item.PayloadBody,
			PayloadPreview: buildPayloadPreview(item.PayloadBody, 20),
		})
	}
	return ReporterRecentReportsResult{
		Reporter: reporter,
		Items:    resultItems,
	}, nil
}

func (s *Service) syncReporterBatch(ctx context.Context, reporters []Reporter, dryRun bool, since *time.Time, onlyActive bool) (SyncBatchResult, error) {
	result := SyncBatchResult{
		Requested:     len(reporters),
		Scanned:       len(reporters),
		Reporters:     make([]Reporter, 0, len(reporters)),
		WatermarkFrom: since,
		DryRun:        dryRun,
		OnlyActive:    onlyActive,
	}
	logger := logging.ForContext(ctx).With(
		slog.Int("requested_count", result.Requested),
		slog.Bool("dry_run", dryRun),
		slog.Bool("only_active", onlyActive),
		slog.Any("watermark_from", since),
	)
	logger.Info("rapidpro_reporter_sync_scan_started")
	if len(reporters) == 0 {
		now := s.clock()
		result.WatermarkTo = &now
		logger.Info("rapidpro_reporter_sync_scan_completed",
			slog.Int("scanned_count", result.Scanned),
			slog.Int("synced_count", result.Synced),
			slog.Int("created_count", result.Created),
			slog.Int("updated_count", result.Updated),
			slog.Int("failed_count", result.Failed),
			slog.Any("watermark_to", result.WatermarkTo),
		)
		return result, nil
	}
	if dryRun {
		now := s.clock()
		result.WatermarkTo = &now
		logger.Info("rapidpro_reporter_sync_scan_completed",
			slog.Int("scanned_count", result.Scanned),
			slog.Int("synced_count", result.Synced),
			slog.Int("created_count", result.Created),
			slog.Int("updated_count", result.Updated),
			slog.Int("failed_count", result.Failed),
			slog.Any("watermark_to", result.WatermarkTo),
		)
		return result, nil
	}
	var batchErr error
	for _, reporter := range reporters {
		syncResult, err := s.syncReporter(ctx, reporter)
		if err != nil {
			result.Failed++
			result.FailedIDs = append(result.FailedIDs, reporter.ID)
			result.FailedNames = append(result.FailedNames, reporter.Name)
			batchErr = errors.Join(batchErr, err)
			logger.Error("rapidpro_reporter_sync_reporter_failed",
				slog.Int64("reporter_id", reporter.ID),
				slog.String("reporter_name", reporter.Name),
				slog.String("error", err.Error()),
			)
			continue
		}
		result.Reporters = append(result.Reporters, syncResult.Reporter)
		result.Synced++
		switch syncResult.Operation {
		case "created":
			result.Created++
		default:
			result.Updated++
		}
	}
	now := s.clock()
	result.WatermarkTo = &now
	if batchErr != nil {
		logger.Error("rapidpro_reporter_sync_scan_failed",
			slog.Int("scanned_count", result.Scanned),
			slog.Int("synced_count", result.Synced),
			slog.Int("created_count", result.Created),
			slog.Int("updated_count", result.Updated),
			slog.Int("failed_count", result.Failed),
			slog.Any("watermark_to", result.WatermarkTo),
		)
		return result, fmt.Errorf("%d reporter syncs failed: %w", result.Failed, batchErr)
	}
	logger.Info("rapidpro_reporter_sync_scan_completed",
		slog.Int("scanned_count", result.Scanned),
		slog.Int("synced_count", result.Synced),
		slog.Int("created_count", result.Created),
		slog.Int("updated_count", result.Updated),
		slog.Int("failed_count", result.Failed),
		slog.Any("watermark_to", result.WatermarkTo),
	)
	return result, nil
}

func (s *Service) syncReporter(ctx context.Context, reporter Reporter) (SyncResult, error) {
	built, err := s.buildRapidProSync(ctx, reporter)
	if err != nil {
		return SyncResult{}, err
	}
	contact, err := s.rapidProClient.UpsertContact(ctx, built.Connection, built.UpsertInput)
	if err != nil {
		return SyncResult{}, mapRapidProRequestError(err)
	}
	updatedReporter, err := s.repo.UpdateRapidProStatus(ctx, reporter.ID, contact.UUID, true)
	if err != nil {
		return SyncResult{}, err
	}
	s.logAudit(ctx, audit.Event{
		Action:     "reporter.synced",
		EntityType: "reporter",
		EntityID:   strPtr(fmt.Sprintf("%d", updatedReporter.ID)),
		Metadata: map[string]any{
			"name":         updatedReporter.Name,
			"operation":    built.Operation,
			"rapidProUuid": updatedReporter.RapidProUUID,
			"groupCount":   len(built.ResolvedGroups),
		},
	})
	return SyncResult{Reporter: updatedReporter, Operation: built.Operation, GroupCount: len(built.ResolvedGroups)}, nil
}

func (s *Service) lookupRapidProContact(ctx context.Context, conn rapidpro.Connection, reporter Reporter) (rapidpro.Contact, bool, error) {
	reporter.RapidProUUID = strings.TrimSpace(reporter.RapidProUUID)
	if reporter.RapidProUUID != "" {
		contact, found, err := s.rapidProClient.LookupContactByUUID(ctx, conn, reporter.RapidProUUID)
		if err != nil {
			return rapidpro.Contact{}, false, mapRapidProRequestError(err)
		}
		if found {
			return contact, true, nil
		}
	}
	urn := phoneURN(reporter.Telephone)
	if urn == "" {
		return rapidpro.Contact{}, false, nil
	}
	contact, found, err := s.rapidProClient.LookupContactByURN(ctx, conn, urn)
	if err != nil {
		return rapidpro.Contact{}, false, mapRapidProRequestError(err)
	}
	return contact, found, nil
}

func (s *Service) getRapidProMessageHistoryForReporter(ctx context.Context, reporter Reporter) (RapidProMessageHistoryResult, error) {
	conn, err := s.rapidProConnection(ctx)
	if err != nil {
		return RapidProMessageHistoryResult{}, err
	}
	contact, found, err := s.lookupRapidProContact(ctx, conn, reporter)
	if err != nil {
		return RapidProMessageHistoryResult{}, err
	}
	result := RapidProMessageHistoryResult{Reporter: reporter, Found: found, Items: []RapidProMessageRecord{}}
	if !found {
		return result, nil
	}
	targetURNs := make(map[string]struct{}, len(contact.URNs))
	for _, urn := range contact.URNs {
		normalized := strings.ToLower(strings.TrimSpace(urn))
		if normalized == "" {
			continue
		}
		targetURNs[normalized] = struct{}{}
	}
	var (
		query        map[string]string
		next         string
		pagesScanned int
	)
	for pagesScanned < maxRapidProHistoryPages && len(result.Items) < maxRapidProHistoryItems {
		messages, pageNext, err := s.rapidProClient.ListMessages(ctx, conn, query)
		if err != nil {
			return RapidProMessageHistoryResult{}, mapRapidProRequestError(err)
		}
		pagesScanned++
		next = pageNext
		for _, message := range messages {
			if !messageMatchesReporter(message, contact.UUID, targetURNs) {
				continue
			}
			result.Items = append(result.Items, toRapidProMessageRecord(message))
			if len(result.Items) >= maxRapidProHistoryItems {
				break
			}
		}
		if next == "" {
			break
		}
		nextQuery, ok := rapidProNextQuery(next)
		if !ok {
			break
		}
		query = nextQuery
	}
	sort.SliceStable(result.Items, func(i, j int) bool {
		left := messageRecordSortTime(result.Items[i])
		right := messageRecordSortTime(result.Items[j])
		if left.Equal(right) {
			return result.Items[i].ID < result.Items[j].ID
		}
		return left.Before(right)
	})
	result.Next = next
	return result, nil
}

func (s *Service) loadByIDsForUser(ctx context.Context, userID int64, ids []int64) ([]Reporter, error) {
	scope, err := s.resolveScope(ctx, userID)
	if err != nil {
		return nil, err
	}
	reporters, err := s.loadByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	if !scope.Restricted {
		return reporters, nil
	}
	filtered := make([]Reporter, 0, len(reporters))
	for _, item := range reporters {
		if err := s.ensureReporterInScope(ctx, scope, item); err == nil {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}

func (s *Service) ensureReporterInScope(ctx context.Context, scope userorg.Scope, reporter Reporter) error {
	if !scope.Restricted {
		return nil
	}
	return s.ensureOrgUnitInScope(ctx, scope, reporter.OrgUnitID)
}

func (s *Service) ensureOrgUnitInScope(ctx context.Context, scope userorg.Scope, orgUnitID int64) error {
	if !scope.Restricted {
		return nil
	}
	if s.orgUnitLookup == nil {
		return apperror.Forbidden("Organisation unit is outside your assigned jurisdiction")
	}
	unit, err := s.orgUnitLookup.Get(ctx, orgUnitID)
	if err != nil {
		return err
	}
	if !userorg.ScopeContainsPath(scope, unit.Path) {
		return apperror.Forbidden("Organisation unit is outside your assigned jurisdiction")
	}
	return nil
}

func (s *Service) resolveScope(ctx context.Context, userID int64) (userorg.Scope, error) {
	if s.scopeResolver == nil {
		return userorg.Scope{}, nil
	}
	return s.scopeResolver.ResolveScope(ctx, userID)
}

func buildPayloadPreview(payload string, limit int) string {
	trimmed := strings.TrimSpace(payload)
	if limit <= 0 {
		limit = 20
	}
	runes := []rune(trimmed)
	if len(runes) <= limit {
		return trimmed
	}
	return string(runes[:limit]) + "..."
}

type rapidProContactData struct {
	Name         string
	URNs         []string
	Fields       map[string]string
	FacilityName string
	FacilityUID  string
}

type rapidProSyncBuild struct {
	Connection     rapidpro.Connection
	Operation      string
	UpsertInput    rapidpro.UpsertContactInput
	RequestQuery   map[string]string
	RequestBody    map[string]any
	ResolvedGroups []settings.RapidProResolvedGroup
	ContactData    rapidProContactData
}

func (s *Service) buildRapidProSync(ctx context.Context, reporter Reporter) (rapidProSyncBuild, error) {
	reporter.RapidProUUID = strings.TrimSpace(reporter.RapidProUUID)
	conn, err := s.rapidProConnection(ctx)
	if err != nil {
		return rapidProSyncBuild{}, err
	}
	urn := phoneURN(reporter.Telephone)
	if urn == "" {
		return rapidProSyncBuild{}, apperror.ValidationWithDetails("validation failed", map[string]any{"telephone": []string{"must resolve to a RapidPro tel: URN"}})
	}
	mapped, err := s.rapidProContactData(ctx, reporter)
	if err != nil {
		return rapidProSyncBuild{}, err
	}
	resolvedGroups, err := s.lookupRapidProGroups(ctx, conn, reporter.Groups)
	if err != nil {
		return rapidProSyncBuild{}, err
	}
	if resolvedGroups == nil {
		resolvedGroups = []settings.RapidProResolvedGroup{}
	}
	groupUUIDs := make([]string, 0, len(resolvedGroups))
	for _, group := range resolvedGroups {
		groupUUIDs = append(groupUUIDs, group.UUID)
	}

	operation := "updated"
	resolvedUUID := reporter.RapidProUUID
	if resolvedUUID != "" {
		contact, found, lookupErr := s.rapidProClient.LookupContactByUUID(ctx, conn, resolvedUUID)
		if lookupErr != nil {
			return rapidProSyncBuild{}, mapRapidProRequestError(lookupErr)
		}
		if found {
			resolvedUUID = contact.UUID
		} else {
			resolvedUUID = ""
		}
	}
	if resolvedUUID == "" {
		contact, found, lookupErr := s.rapidProClient.LookupContactByURN(ctx, conn, urn)
		if lookupErr != nil {
			return rapidProSyncBuild{}, mapRapidProRequestError(lookupErr)
		}
		if found {
			resolvedUUID = contact.UUID
		} else {
			operation = "created"
		}
	}

	upsertInput := rapidpro.UpsertContactInput{
		UUID:   resolvedUUID,
		Name:   mapped.Name,
		URNs:   mapped.URNs,
		Groups: groupUUIDs,
		Fields: mapped.Fields,
	}
	requestQuery := map[string]string{}
	if resolvedUUID != "" {
		requestQuery["uuid"] = resolvedUUID
	}
	requestBody := map[string]any{
		"name":   upsertInput.Name,
		"urns":   append([]string(nil), upsertInput.URNs...),
		"groups": []string{},
	}
	if len(upsertInput.Groups) > 0 {
		requestBody["groups"] = append([]string(nil), upsertInput.Groups...)
	}
	if len(mapped.Fields) > 0 {
		requestBody["fields"] = cloneStringMap(mapped.Fields)
	}

	return rapidProSyncBuild{
		Connection:     conn,
		Operation:      operation,
		UpsertInput:    upsertInput,
		RequestQuery:   requestQuery,
		RequestBody:    requestBody,
		ResolvedGroups: resolvedGroups,
		ContactData:    mapped,
	}, nil
}

func (s *Service) rapidProContactData(ctx context.Context, reporter Reporter) (rapidProContactData, error) {
	result := rapidProContactData{
		Name: strings.TrimSpace(reporter.Name),
	}
	if urn := phoneURN(reporter.Telephone); urn != "" {
		result.URNs = []string{urn}
	}
	if s.rapidProSettings == nil {
		return result, nil
	}
	config, err := s.rapidProSettings.GetRapidProReporterSync(ctx)
	if err != nil {
		return rapidProContactData{}, err
	}
	if !config.Validation.IsValid {
		return rapidProContactData{}, apperror.ValidationWithDetails("validation failed", map[string]any{
			"rapidproMapping": config.Validation.Errors,
		})
	}
	if len(config.Mappings) == 0 {
		return result, nil
	}

	needsFacility := false
	for _, mapping := range config.Mappings {
		switch mapping.SourceKey {
		case "facilityName", "facilityUID":
			needsFacility = true
		}
	}

	var unit orgunit.OrgUnit
	if needsFacility {
		if s.orgUnitLookup == nil {
			return rapidProContactData{}, apperror.ValidationWithDetails("validation failed", map[string]any{
				"rapidproMapping": []string{"Facility mapping is configured, but org unit lookup is not available"},
			})
		}
		loaded, getErr := s.orgUnitLookup.Get(ctx, reporter.OrgUnitID)
		if getErr != nil {
			return rapidProContactData{}, apperror.ValidationWithDetails("validation failed", map[string]any{
				"rapidproMapping": []string{"Reporter facility could not be loaded for RapidPro field mapping"},
			})
		}
		unit = loaded
		result.FacilityName = strings.TrimSpace(unit.Name)
		result.FacilityUID = strings.TrimSpace(unit.UID)
	}

	values := make(map[string]string, len(config.Mappings))
	for _, mapping := range config.Mappings {
		resolved := strings.TrimSpace(resolveRapidProSourceValue(mapping.SourceKey, reporter, unit))
		targetKey := strings.ToLower(strings.TrimSpace(mapping.RapidProFieldKey))
		if resolved == "" {
			if targetKey == settingsTargetURNWhats() {
				continue
			}
			return rapidProContactData{}, apperror.ValidationWithDetails("validation failed", map[string]any{
				"rapidproMapping": []string{fmt.Sprintf("%s is required for RapidPro field %q", mapping.SourceLabel, mapping.RapidProFieldKey)},
			})
		}
		switch targetKey {
		case settingsTargetName():
			result.Name = resolved
		case settingsTargetURNTel():
			urn := phoneURN(resolved)
			if urn == "" {
				return rapidProContactData{}, apperror.ValidationWithDetails("validation failed", map[string]any{
					"rapidproMapping": []string{fmt.Sprintf("%s must resolve to a RapidPro tel: URN", mapping.SourceLabel)},
				})
			}
			result.URNs = append(result.URNs, urn)
		case settingsTargetURNWhats():
			urn := whatsappURN(resolved)
			if urn == "" {
				return rapidProContactData{}, apperror.ValidationWithDetails("validation failed", map[string]any{
					"rapidproMapping": []string{fmt.Sprintf("%s must resolve to a RapidPro whatsapp: URN", mapping.SourceLabel)},
				})
			}
			result.URNs = append(result.URNs, urn)
		default:
			values[mapping.RapidProFieldKey] = resolved
		}
	}
	result.URNs = normalizeURNs(result.URNs)
	if len(values) > 0 {
		result.Fields = values
	}
	return result, nil
}

func resolveRapidProSourceValue(sourceKey string, reporter Reporter, unit orgunit.OrgUnit) string {
	switch sourceKey {
	case "name":
		return reporter.Name
	case "telephone":
		return reporter.Telephone
	case "whatsapp":
		return reporter.WhatsApp
	case "telegram":
		return reporter.Telegram
	case "reportingLocation":
		return reporter.ReportingLocation
	case "facilityName":
		return unit.Name
	case "facilityUID":
		return unit.UID
	default:
		return ""
	}
}

func (s *Service) lookupRapidProGroups(ctx context.Context, conn rapidpro.Connection, groups []string) ([]settings.RapidProResolvedGroup, error) {
	normalized := normalizeGroups(groups)
	if len(normalized) == 0 {
		return nil, nil
	}
	if s.groupCatalog != nil {
		groups, err := s.groupCatalog.EnsureRapidProGroups(ctx, normalized)
		if err != nil {
			return nil, err
		}
		resolved := make([]settings.RapidProResolvedGroup, 0, len(groups))
		for _, group := range groups {
			resolved = append(resolved, settings.RapidProResolvedGroup{
				Name: group.Name,
				UUID: group.UUID,
			})
		}
		return resolved, nil
	}
	resolved := make([]settings.RapidProResolvedGroup, 0, len(normalized))
	missing := make([]string, 0)
	for _, group := range normalized {
		match, found, err := s.rapidProClient.LookupGroupByName(ctx, conn, group)
		if err != nil {
			return nil, mapRapidProRequestError(err)
		}
		if !found {
			missing = append(missing, group)
			continue
		}
		resolved = append(resolved, settings.RapidProResolvedGroup{
			Name: match.Name,
			UUID: match.UUID,
		})
	}
	if len(missing) > 0 {
		return nil, apperror.ValidationWithDetails("validation failed", map[string]any{
			"rapidproGroups": []string{fmt.Sprintf("RapidPro groups must exist before sync: %s", strings.Join(missing, ", "))},
		})
	}
	return resolved, nil
}

var _ reporterGroupCatalog = (*reportergroup.Service)(nil)

func (s *Service) rapidProConnection(ctx context.Context) (rapidpro.Connection, error) {
	if s.serverLookup == nil || s.rapidProClient == nil {
		return rapidpro.Connection{}, apperror.ValidationWithDetails("validation failed", map[string]any{"rapidpro": []string{"RapidPro integration is not configured"}})
	}
	serverCode, err := s.rapidProServerCode(ctx)
	if err != nil {
		return rapidpro.Connection{}, err
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

func (s *Service) rapidProServerCode(ctx context.Context) (string, error) {
	if s.rapidProSettings == nil {
		return defaultRapidProServerCode, nil
	}
	config, err := s.rapidProSettings.GetRapidProReporterSync(ctx)
	if err != nil {
		return "", err
	}
	if code := strings.TrimSpace(config.RapidProServerCode); code != "" {
		return code, nil
	}
	return defaultRapidProServerCode, nil
}

func (s *Service) loadByIDs(ctx context.Context, ids []int64) ([]Reporter, error) {
	normalized := make([]int64, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	if len(normalized) == 0 {
		return nil, apperror.ValidationWithDetails("validation failed", map[string]any{"reporterIds": []string{"at least one reporter is required"}})
	}
	reporters, err := s.repo.ListByIDs(ctx, normalized)
	if err != nil {
		return nil, err
	}
	if len(reporters) != len(normalized) {
		return nil, apperror.ValidationWithDetails("validation failed", map[string]any{"reporterIds": []string{"one or more reporters were not found"}})
	}
	return reporters, nil
}

func validateReporter(r Reporter, requireID bool) error {
	if requireID && r.ID == 0 {
		return apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"is required"}})
	}
	if strings.TrimSpace(r.Name) == "" {
		return apperror.ValidationWithDetails("validation failed", map[string]any{"name": []string{"is required"}})
	}
	if strings.TrimSpace(r.Telephone) == "" {
		return apperror.ValidationWithDetails("validation failed", map[string]any{"telephone": []string{"is required"}})
	}
	if r.OrgUnitID == 0 {
		return apperror.ValidationWithDetails("validation failed", map[string]any{"orgUnitId": []string{"is required"}})
	}
	return nil
}

func mapReporterLookupError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"reporter not found"}})
	}
	return mapRapidProRequestError(err)
}

func mapRapidProRequestError(err error) error {
	if err == nil {
		return nil
	}
	var requestErr *rapidpro.RequestError
	if errors.As(err, &requestErr) && requestErr.StatusCode >= 400 && requestErr.StatusCode < 500 {
		detail := fmt.Sprintf("RapidPro rejected the request (status %d)", requestErr.StatusCode)
		if requestErr.Body != "" {
			detail += ": " + requestErr.Body
		}
		return apperror.ValidationWithDetails("validation failed", map[string]any{
			"rapidpro": []string{detail},
		})
	}
	return err
}

func phoneURN(phone string) string {
	value := strings.TrimSpace(phone)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(value), "tel:") {
		return value
	}
	return "tel:" + value
}

func whatsappURN(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(trimmed), "whatsapp:") {
		return trimmed
	}
	return "whatsapp:" + trimmed
}

func normalizeURNs(input []string) []string {
	if len(input) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(input))
	output := make([]string, 0, len(input))
	for _, item := range input {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		output = append(output, value)
	}
	if len(output) == 0 {
		return nil
	}
	return output
}

func (s *Service) broadcastOrgUnitPaths(ctx context.Context, scope userorg.Scope, orgUnitIDs []int64) ([]string, error) {
	if s.orgUnitLookup == nil {
		return nil, apperror.ValidationWithDetails("validation failed", map[string]any{"orgUnitIds": []string{"Organisation unit lookup is not configured"}})
	}
	paths := make([]string, 0, len(orgUnitIDs))
	for _, id := range orgUnitIDs {
		unit, err := s.orgUnitLookup.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		if scope.Restricted && !userorg.ScopeContainsPath(scope, unit.Path) {
			return nil, apperror.Forbidden("Organisation unit is outside your assigned jurisdiction")
		}
		path := strings.TrimSpace(unit.Path)
		if path == "" {
			return nil, apperror.ValidationWithDetails("validation failed", map[string]any{"orgUnitIds": []string{"Selected organisation unit is missing hierarchy path data"}})
		}
		paths = append(paths, path)
	}
	return normalizeStringValues(paths), nil
}

func normalizeInt64IDs(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	values := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		values = append(values, id)
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	return values
}

func normalizeStringValues(input []string) []string {
	if len(input) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(input))
	values := make([]string, 0, len(input))
	for _, item := range input {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		values = append(values, trimmed)
	}
	sort.Strings(values)
	return values
}

func jurisdictionBroadcastDedupeKey(userID int64, orgUnitIDs []int64, reporterGroup string, message string) string {
	parts := []string{
		fmt.Sprintf("user:%d", userID),
		"orgunits:" + joinInt64IDs(orgUnitIDs),
		"group:" + strings.ToLower(strings.TrimSpace(reporterGroup)),
		"text:" + normalizeBroadcastText(message),
	}
	digest := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(digest[:])
}

func normalizeBroadcastText(message string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(message)), " ")
}

func joinInt64IDs(values []int64) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%d", value))
	}
	return strings.Join(parts, ",")
}

func newUID() string {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%d-%d", time.Now().UTC().UnixNano(), time.Now().UTC().Unix())))
	return hex.EncodeToString(digest[:12])
}

func settingsTargetName() string {
	return "name"
}

func settingsTargetURNTel() string {
	return "urn.tel"
}

func settingsTargetURNWhats() string {
	return "urn.whatsapp"
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

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func toRapidProContactSnapshot(contact rapidpro.Contact) RapidProContactSnapshot {
	result := RapidProContactSnapshot{
		UUID:       strings.TrimSpace(contact.UUID),
		Name:       strings.TrimSpace(contact.Name),
		Status:     strings.TrimSpace(contact.Status),
		Language:   strings.TrimSpace(contact.Language),
		URNs:       append([]string(nil), contact.URNs...),
		Groups:     make([]RapidProGroup, 0, len(contact.Groups)),
		Fields:     cloneStringMap(contact.Fields),
		CreatedOn:  strings.TrimSpace(contact.CreatedOn),
		ModifiedOn: strings.TrimSpace(contact.ModifiedOn),
		LastSeenOn: strings.TrimSpace(contact.LastSeenOn),
	}
	for _, group := range contact.Groups {
		result.Groups = append(result.Groups, RapidProGroup{
			UUID: strings.TrimSpace(group.UUID),
			Name: strings.TrimSpace(group.Name),
		})
	}
	if contact.Flow != nil {
		result.Flow = &RapidProFlow{
			UUID: strings.TrimSpace(contact.Flow.UUID),
			Name: strings.TrimSpace(contact.Flow.Name),
		}
	}
	return result
}

func toRapidProMessageRecord(message rapidpro.Message) RapidProMessageRecord {
	result := RapidProMessageRecord{
		ID:          message.ID,
		BroadcastID: message.BroadcastID,
		Direction:   strings.TrimSpace(message.Direction),
		Type:        strings.TrimSpace(message.Type),
		Status:      strings.TrimSpace(message.Status),
		Visibility:  strings.TrimSpace(message.Visibility),
		Text:        strings.TrimSpace(message.Text),
		URN:         strings.TrimSpace(message.URN),
		CreatedOn:   strings.TrimSpace(message.CreatedOn),
		SentOn:      strings.TrimSpace(message.SentOn),
		ModifiedOn:  strings.TrimSpace(message.ModifiedOn),
	}
	if message.Channel != nil {
		result.Channel = &RapidProFlow{
			UUID: strings.TrimSpace(message.Channel.UUID),
			Name: strings.TrimSpace(message.Channel.Name),
		}
	}
	if message.Flow != nil {
		result.Flow = &RapidProFlow{
			UUID: strings.TrimSpace(message.Flow.UUID),
			Name: strings.TrimSpace(message.Flow.Name),
		}
	}
	return result
}

func messageMatchesReporter(message rapidpro.Message, contactUUID string, urns map[string]struct{}) bool {
	messageContactUUID := strings.TrimSpace(message.Contact.UUID)
	targetUUID := strings.TrimSpace(contactUUID)
	if messageContactUUID != "" && strings.EqualFold(messageContactUUID, targetUUID) {
		return true
	}
	_, ok := urns[strings.ToLower(strings.TrimSpace(message.URN))]
	return ok
}

func rapidProNextQuery(next string) (map[string]string, bool) {
	trimmed := strings.TrimSpace(next)
	if trimmed == "" {
		return nil, false
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, false
	}
	values := parsed.Query()
	if len(values) == 0 {
		return nil, false
	}
	query := make(map[string]string, len(values))
	for key, items := range values {
		if len(items) == 0 {
			continue
		}
		query[key] = items[0]
	}
	if len(query) == 0 {
		return nil, false
	}
	return query, true
}

func messageRecordSortTime(item RapidProMessageRecord) time.Time {
	for _, value := range []string{item.SentOn, item.CreatedOn, item.ModifiedOn} {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		parsed, err := time.Parse(time.RFC3339, trimmed)
		if err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func (s *Service) logAudit(ctx context.Context, event audit.Event) {
	if s == nil || s.auditService == nil {
		return
	}
	_ = s.auditService.Log(ctx, event)
}

func strPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}
