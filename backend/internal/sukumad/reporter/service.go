package reporter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/audit"
	"basepro/backend/internal/settings"
	"basepro/backend/internal/sukumad/orgunit"
	"basepro/backend/internal/sukumad/rapidex/rapidpro"
	sukumadserver "basepro/backend/internal/sukumad/server"
)

const rapidProServerCode = "rapidpro"

type rapidProServerLookup interface {
	GetServerByCode(context.Context, string) (sukumadserver.Record, error)
}

type rapidProClient interface {
	LookupContactByUUID(context.Context, rapidpro.Connection, string) (rapidpro.Contact, bool, error)
	LookupContactByURN(context.Context, rapidpro.Connection, string) (rapidpro.Contact, bool, error)
	UpsertContact(context.Context, rapidpro.Connection, rapidpro.UpsertContactInput) (rapidpro.Contact, error)
	EnsureGroup(context.Context, rapidpro.Connection, string) (rapidpro.Group, bool, error)
	SendMessage(context.Context, rapidpro.Connection, string, string) (rapidpro.Message, error)
	SendBroadcast(context.Context, rapidpro.Connection, []string, string) (rapidpro.Broadcast, error)
}

type rapidProReporterSyncSettingsProvider interface {
	GetRapidProReporterSync(context.Context) (settings.RapidProReporterSyncSettings, error)
}

type orgUnitLookup interface {
	Get(context.Context, int64) (orgunit.OrgUnit, error)
}

// Service encapsulates business logic for reporters and depends on a Repository.
type Service struct {
	repo             Repository
	auditService     *audit.Service
	serverLookup     rapidProServerLookup
	rapidProClient   rapidProClient
	rapidProSettings rapidProReporterSyncSettingsProvider
	orgUnitLookup    orgUnitLookup
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

// List returns a page of reporters matching the provided query.
func (s *Service) List(ctx context.Context, query ListQuery) (ListResult, error) {
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

// Create validates and persists a new Reporter.
func (s *Service) Create(ctx context.Context, r Reporter) (Reporter, error) {
	if err := validateReporter(r, false); err != nil {
		return Reporter{}, err
	}
	now := s.clock()
	r.CreatedAt = now
	r.UpdatedAt = now
	r.Groups = normalizeGroups(r.Groups)
	return s.repo.Create(ctx, r)
}

// Update validates and updates an existing reporter.
func (s *Service) Update(ctx context.Context, r Reporter) (Reporter, error) {
	if err := validateReporter(r, true); err != nil {
		return Reporter{}, err
	}
	r.Groups = normalizeGroups(r.Groups)
	r.UpdatedAt = s.clock()
	updated, err := s.repo.Update(ctx, r)
	if err != nil {
		return Reporter{}, mapReporterLookupError(err)
	}
	return updated, nil
}

// Delete removes a reporter by ID.
func (s *Service) Delete(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return mapReporterLookupError(err)
	}
	return nil
}

func (s *Service) SyncReporter(ctx context.Context, id int64) (SyncResult, error) {
	reporter, err := s.Get(ctx, id)
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
		return MessageResult{}, err
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
		return BroadcastResult{}, err
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

func (s *Service) syncReporterBatch(ctx context.Context, reporters []Reporter, dryRun bool, since *time.Time, onlyActive bool) (SyncBatchResult, error) {
	result := SyncBatchResult{
		Requested:     len(reporters),
		Scanned:       len(reporters),
		Reporters:     make([]Reporter, 0, len(reporters)),
		WatermarkFrom: since,
		DryRun:        dryRun,
		OnlyActive:    onlyActive,
	}
	if len(reporters) == 0 {
		now := s.clock()
		result.WatermarkTo = &now
		return result, nil
	}
	if dryRun {
		now := s.clock()
		result.WatermarkTo = &now
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
		return result, fmt.Errorf("%d reporter syncs failed: %w", result.Failed, batchErr)
	}
	return result, nil
}

func (s *Service) syncReporter(ctx context.Context, reporter Reporter) (SyncResult, error) {
	reporter.RapidProUUID = strings.TrimSpace(reporter.RapidProUUID)
	conn, err := s.rapidProConnection(ctx)
	if err != nil {
		return SyncResult{}, err
	}
	groupUUIDs, _, err := s.ensureRapidProGroups(ctx, conn, reporter.Groups)
	if err != nil {
		return SyncResult{}, err
	}
	urn := phoneURN(reporter.Telephone)
	if urn == "" {
		return SyncResult{}, apperror.ValidationWithDetails("validation failed", map[string]any{"telephone": []string{"must resolve to a RapidPro tel: URN"}})
	}

	operation := "updated"
	resolvedUUID := reporter.RapidProUUID
	upsertInput := rapidpro.UpsertContactInput{
		Name:   strings.TrimSpace(reporter.Name),
		URN:    urn,
		URNs:   []string{urn},
		Groups: groupUUIDs,
	}
	mapped, err := s.rapidProContactData(ctx, reporter)
	if err != nil {
		return SyncResult{}, err
	}
	if mapped.Name != "" {
		upsertInput.Name = mapped.Name
	}
	if len(mapped.URNs) > 0 {
		upsertInput.URNs = mapped.URNs
	}
	upsertInput.Fields = mapped.Fields
	if resolvedUUID != "" {
		contact, found, lookupErr := s.rapidProClient.LookupContactByUUID(ctx, conn, resolvedUUID)
		if lookupErr != nil {
			return SyncResult{}, lookupErr
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
			return SyncResult{}, lookupErr
		}
		if found {
			resolvedUUID = contact.UUID
		} else {
			operation = "created"
		}
	}
	upsertInput.UUID = resolvedUUID
	contact, err := s.rapidProClient.UpsertContact(ctx, conn, upsertInput)
	if err != nil {
		return SyncResult{}, err
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
			"operation":    operation,
			"rapidProUuid": updatedReporter.RapidProUUID,
			"groupCount":   len(groupUUIDs),
		},
	})
	return SyncResult{Reporter: updatedReporter, Operation: operation, GroupCount: len(groupUUIDs)}, nil
}

type rapidProContactData struct {
	Name   string
	URNs   []string
	Fields map[string]string
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
	}

	values := make(map[string]string, len(config.Mappings))
	for _, mapping := range config.Mappings {
		resolved := strings.TrimSpace(resolveRapidProSourceValue(mapping.SourceKey, reporter, unit))
		if resolved == "" {
			return rapidProContactData{}, apperror.ValidationWithDetails("validation failed", map[string]any{
				"rapidproMapping": []string{fmt.Sprintf("%s is required for RapidPro field %q", mapping.SourceLabel, mapping.RapidProFieldKey)},
			})
		}
		switch strings.ToLower(strings.TrimSpace(mapping.RapidProFieldKey)) {
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

func (s *Service) ensureRapidProGroups(ctx context.Context, conn rapidpro.Connection, groups []string) ([]string, int, error) {
	normalized := normalizeGroups(groups)
	if len(normalized) == 0 {
		return nil, 0, nil
	}
	groupUUIDs := make([]string, 0, len(normalized))
	createdCount := 0
	for _, group := range normalized {
		resolved, created, err := s.rapidProClient.EnsureGroup(ctx, conn, group)
		if err != nil {
			return nil, 0, err
		}
		if created {
			createdCount++
		}
		groupUUIDs = append(groupUUIDs, resolved.UUID)
	}
	return groupUUIDs, createdCount, nil
}

func (s *Service) rapidProConnection(ctx context.Context) (rapidpro.Connection, error) {
	if s.serverLookup == nil || s.rapidProClient == nil {
		return rapidpro.Connection{}, apperror.ValidationWithDetails("validation failed", map[string]any{"rapidpro": []string{"RapidPro integration is not configured"}})
	}
	record, err := s.serverLookup.GetServerByCode(ctx, rapidProServerCode)
	if err != nil {
		return rapidpro.Connection{}, err
	}
	if record.Suspended {
		return rapidpro.Connection{}, apperror.ValidationWithDetails("validation failed", map[string]any{"rapidpro": []string{"RapidPro server is suspended"}})
	}
	if strings.TrimSpace(record.BaseURL) == "" {
		return rapidpro.Connection{}, apperror.ValidationWithDetails("validation failed", map[string]any{"rapidpro": []string{"RapidPro server base URL is required"}})
	}
	return rapidpro.Connection{BaseURL: record.BaseURL, Headers: record.Headers}, nil
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
