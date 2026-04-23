package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/audit"
	"basepro/backend/internal/sukumad/rapidex/rapidpro"
	sukumadserver "basepro/backend/internal/sukumad/server"
)

const (
	rapidProSyncCategory   = "rapidpro"
	rapidProSyncKey        = "reporter_sync"
	defaultRapidProSrvCode = "rapidpro"
	rapidProTargetName     = "name"
	rapidProTargetURNTel   = "urn.tel"
	rapidProTargetURNWhats = "urn.whatsapp"
	rapidProBuiltInType    = "builtin"
)

var rapidProReporterSourceLabels = map[string]string{
	"name":              "Reporter Name",
	"telephone":         "Telephone",
	"whatsapp":          "WhatsApp",
	"telegram":          "Telegram",
	"reportingLocation": "Reporting Location",
	"facilityName":      "Facility Name",
	"facilityUID":       "Facility UID",
}

var rapidProBuiltInFields = []rapidpro.ContactField{
	{Key: rapidProTargetName, Label: "Contact Name", ValueType: rapidProBuiltInType},
	{Key: rapidProTargetURNTel, Label: "Telephone URN", ValueType: rapidProBuiltInType},
	{Key: rapidProTargetURNWhats, Label: "WhatsApp URN", ValueType: rapidProBuiltInType},
}

type rapidProServerLookup interface {
	GetServerByCode(context.Context, string) (sukumadserver.Record, error)
}

type rapidProFieldClient interface {
	ListContactFields(context.Context, rapidpro.Connection) ([]rapidpro.ContactField, error)
}

type RapidProReporterFieldMapping struct {
	SourceKey        string `json:"sourceKey"`
	SourceLabel      string `json:"sourceLabel"`
	RapidProFieldKey string `json:"rapidProFieldKey"`
}

type RapidProReporterSyncValidation struct {
	IsValid bool     `json:"isValid"`
	Errors  []string `json:"errors,omitempty"`
}

type RapidProReporterSyncSettings struct {
	RapidProServerCode string                         `json:"rapidProServerCode"`
	AvailableFields    []rapidpro.ContactField        `json:"availableFields"`
	Mappings           []RapidProReporterFieldMapping `json:"mappings"`
	LastFetchedAt      *time.Time                     `json:"lastFetchedAt,omitempty"`
	Validation         RapidProReporterSyncValidation `json:"validation"`
}

type RapidProReporterSyncUpdateInput struct {
	RapidProServerCode string                         `json:"rapidProServerCode"`
	Mappings           []RapidProReporterFieldMapping `json:"mappings"`
}

type rapidProReporterSyncStored struct {
	RapidProServerCode string                         `json:"rapidProServerCode"`
	AvailableFields    []rapidpro.ContactField        `json:"availableFields"`
	Mappings           []RapidProReporterFieldMapping `json:"mappings"`
	LastFetchedAt      *time.Time                     `json:"lastFetchedAt,omitempty"`
}

func (s *Service) WithRapidProIntegration(serverLookup rapidProServerLookup, client rapidProFieldClient) *Service {
	s.rapidProServerLookup = serverLookup
	s.rapidProFieldClient = client
	return s
}

func (s *Service) GetRapidProReporterSync(ctx context.Context) (RapidProReporterSyncSettings, error) {
	stored, err := s.getRapidProReporterSyncStored(ctx)
	if err != nil {
		return RapidProReporterSyncSettings{}, err
	}
	return materializeRapidProReporterSync(stored), nil
}

func (s *Service) RefreshRapidProReporterSyncFields(ctx context.Context, actorUserID *int64) (RapidProReporterSyncSettings, error) {
	if s.rapidProServerLookup == nil || s.rapidProFieldClient == nil {
		return RapidProReporterSyncSettings{}, apperror.ValidationWithDetails("validation failed", map[string]any{
			"rapidpro": []string{"RapidPro integration is not configured"},
		})
	}
	stored, err := s.getRapidProReporterSyncStored(ctx)
	if err != nil {
		return RapidProReporterSyncSettings{}, err
	}
	serverCode := normalizeRapidProServerCode(stored.RapidProServerCode)
	record, err := s.rapidProServerLookup.GetServerByCode(ctx, serverCode)
	if err != nil {
		return RapidProReporterSyncSettings{}, err
	}
	if record.Suspended {
		return RapidProReporterSyncSettings{}, apperror.ValidationWithDetails("validation failed", map[string]any{
			"rapidProServerCode": []string{"selected RapidPro server is suspended"},
		})
	}
	fields, err := s.rapidProFieldClient.ListContactFields(ctx, rapidpro.Connection{
		BaseURL: record.BaseURL,
		Headers: record.Headers,
	})
	if err != nil {
		return RapidProReporterSyncSettings{}, err
	}
	stored.RapidProServerCode = serverCode
	stored.AvailableFields = normalizeRapidProFields(fields)
	stored.Mappings = applySuggestedRapidProMappings(stored.Mappings, stored.AvailableFields)
	now := time.Now().UTC()
	stored.LastFetchedAt = &now
	if err := s.saveRapidProReporterSyncStored(ctx, stored, actorUserID); err != nil {
		return RapidProReporterSyncSettings{}, err
	}
	result := materializeRapidProReporterSync(stored)
	s.logAudit(ctx, audit.Event{
		Action:      "settings.rapidpro_reporter_sync.refresh",
		ActorUserID: actorUserID,
		EntityType:  "settings",
		EntityID:    strPtr("rapidpro.reporter_sync"),
		Metadata: map[string]any{
			"rapidProServerCode": serverCode,
			"fieldCount":         len(result.AvailableFields),
		},
	})
	return result, nil
}

func (s *Service) UpdateRapidProReporterSync(ctx context.Context, input RapidProReporterSyncUpdateInput, actorUserID *int64) (RapidProReporterSyncSettings, error) {
	stored, err := s.getRapidProReporterSyncStored(ctx)
	if err != nil {
		return RapidProReporterSyncSettings{}, err
	}
	serverCode := normalizeRapidProServerCode(input.RapidProServerCode)
	mappings, err := validateRapidProMappings(input.Mappings, stored.AvailableFields)
	if err != nil {
		return RapidProReporterSyncSettings{}, err
	}
	stored.RapidProServerCode = serverCode
	stored.Mappings = mappings
	if err := s.saveRapidProReporterSyncStored(ctx, stored, actorUserID); err != nil {
		return RapidProReporterSyncSettings{}, err
	}
	result := materializeRapidProReporterSync(stored)
	s.logAudit(ctx, audit.Event{
		Action:      "settings.rapidpro_reporter_sync.update",
		ActorUserID: actorUserID,
		EntityType:  "settings",
		EntityID:    strPtr("rapidpro.reporter_sync"),
		Metadata: map[string]any{
			"rapidProServerCode": serverCode,
			"mappingCount":       len(result.Mappings),
		},
	})
	return result, nil
}

func (s *Service) getRapidProReporterSyncStored(ctx context.Context) (rapidProReporterSyncStored, error) {
	raw, err := s.repo.Get(ctx, rapidProSyncCategory, rapidProSyncKey)
	if err != nil {
		if err == ErrNotFound {
			return rapidProReporterSyncStored{RapidProServerCode: defaultRapidProSrvCode}, nil
		}
		return rapidProReporterSyncStored{}, err
	}
	var stored rapidProReporterSyncStored
	if unmarshalErr := json.Unmarshal(raw, &stored); unmarshalErr != nil {
		return rapidProReporterSyncStored{RapidProServerCode: defaultRapidProSrvCode}, nil
	}
	stored.RapidProServerCode = normalizeRapidProServerCode(stored.RapidProServerCode)
	stored.AvailableFields = normalizeRapidProFields(stored.AvailableFields)
	stored.Mappings = normalizeRapidProMappings(stored.Mappings)
	return stored, nil
}

func (s *Service) saveRapidProReporterSyncStored(ctx context.Context, stored rapidProReporterSyncStored, actorUserID *int64) error {
	payload, err := json.Marshal(stored)
	if err != nil {
		return fmt.Errorf("marshal rapidpro reporter sync setting: %w", err)
	}
	return s.repo.Upsert(ctx, rapidProSyncCategory, rapidProSyncKey, payload, actorUserID, time.Now().UTC())
}

func materializeRapidProReporterSync(stored rapidProReporterSyncStored) RapidProReporterSyncSettings {
	result := RapidProReporterSyncSettings{
		RapidProServerCode: normalizeRapidProServerCode(stored.RapidProServerCode),
		AvailableFields:    append([]rapidpro.ContactField(nil), stored.AvailableFields...),
		Mappings:           normalizeRapidProMappings(stored.Mappings),
		LastFetchedAt:      stored.LastFetchedAt,
	}
	result.Validation = validateRapidProReporterSyncSettings(result)
	return result
}

func validateRapidProReporterSyncSettings(settings RapidProReporterSyncSettings) RapidProReporterSyncValidation {
	errors := make([]string, 0)
	fieldKeys := make(map[string]struct{}, len(settings.AvailableFields))
	for _, field := range settings.AvailableFields {
		fieldKeys[strings.ToLower(strings.TrimSpace(field.Key))] = struct{}{}
	}
	for _, mapping := range settings.Mappings {
		if mapping.RapidProFieldKey == "" {
			continue
		}
		if _, ok := fieldKeys[strings.ToLower(mapping.RapidProFieldKey)]; !ok {
			errors = append(errors, fmt.Sprintf("Mapped RapidPro field %q is no longer available. Refresh fields and update the mapping.", mapping.RapidProFieldKey))
		}
	}
	return RapidProReporterSyncValidation{
		IsValid: len(errors) == 0,
		Errors:  errors,
	}
}

func validateRapidProMappings(input []RapidProReporterFieldMapping, availableFields []rapidpro.ContactField) ([]RapidProReporterFieldMapping, error) {
	available := make(map[string]struct{}, len(availableFields))
	for _, field := range availableFields {
		available[strings.ToLower(strings.TrimSpace(field.Key))] = struct{}{}
	}
	seenSources := map[string]struct{}{}
	seenFields := map[string]struct{}{}
	result := make([]RapidProReporterFieldMapping, 0, len(input))
	for _, item := range input {
		sourceKey := strings.TrimSpace(item.SourceKey)
		fieldKey := strings.TrimSpace(item.RapidProFieldKey)
		if sourceKey == "" || fieldKey == "" {
			continue
		}
		sourceLabel, ok := rapidProReporterSourceLabels[sourceKey]
		if !ok {
			return nil, apperror.ValidationWithDetails("validation failed", map[string]any{
				"mappings": []string{fmt.Sprintf("unknown source key %q", sourceKey)},
			})
		}
		sourceNorm := strings.ToLower(sourceKey)
		if _, ok := seenSources[sourceNorm]; ok {
			return nil, apperror.ValidationWithDetails("validation failed", map[string]any{
				"mappings": []string{fmt.Sprintf("source %q may only be mapped once", sourceLabel)},
			})
		}
		seenSources[sourceNorm] = struct{}{}
		fieldNorm := strings.ToLower(fieldKey)
		if _, ok := seenFields[fieldNorm]; ok {
			return nil, apperror.ValidationWithDetails("validation failed", map[string]any{
				"mappings": []string{fmt.Sprintf("RapidPro field %q may only be mapped once", fieldKey)},
			})
		}
		seenFields[fieldNorm] = struct{}{}
		if len(available) > 0 {
			if _, ok := available[fieldNorm]; !ok {
				return nil, apperror.ValidationWithDetails("validation failed", map[string]any{
					"mappings": []string{fmt.Sprintf("RapidPro field %q is not available", fieldKey)},
				})
			}
		}
		result = append(result, RapidProReporterFieldMapping{
			SourceKey:        sourceKey,
			SourceLabel:      sourceLabel,
			RapidProFieldKey: fieldKey,
		})
	}
	slices.SortFunc(result, func(left, right RapidProReporterFieldMapping) int {
		return strings.Compare(left.SourceKey, right.SourceKey)
	})
	return result, nil
}

func normalizeRapidProServerCode(value string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return defaultRapidProSrvCode
}

func normalizeRapidProMappings(input []RapidProReporterFieldMapping) []RapidProReporterFieldMapping {
	if len(input) == 0 {
		return nil
	}
	output := make([]RapidProReporterFieldMapping, 0, len(input))
	for _, item := range input {
		sourceKey := strings.TrimSpace(item.SourceKey)
		fieldKey := strings.TrimSpace(item.RapidProFieldKey)
		if sourceKey == "" || fieldKey == "" {
			continue
		}
		label := strings.TrimSpace(item.SourceLabel)
		if label == "" {
			label = rapidProReporterSourceLabels[sourceKey]
		}
		output = append(output, RapidProReporterFieldMapping{
			SourceKey:        sourceKey,
			SourceLabel:      label,
			RapidProFieldKey: fieldKey,
		})
	}
	slices.SortFunc(output, func(left, right RapidProReporterFieldMapping) int {
		return strings.Compare(left.SourceKey, right.SourceKey)
	})
	return output
}

func normalizeRapidProFields(input []rapidpro.ContactField) []rapidpro.ContactField {
	input = append(append([]rapidpro.ContactField(nil), rapidProBuiltInFields...), input...)
	seen := map[string]struct{}{}
	output := make([]rapidpro.ContactField, 0, len(input))
	for _, field := range input {
		key := strings.TrimSpace(field.Key)
		if key == "" {
			continue
		}
		norm := strings.ToLower(key)
		if _, ok := seen[norm]; ok {
			continue
		}
		seen[norm] = struct{}{}
		label := strings.TrimSpace(field.Label)
		if label == "" {
			label = key
		}
		output = append(output, rapidpro.ContactField{
			Key:       key,
			Label:     label,
			ValueType: strings.TrimSpace(field.ValueType),
		})
	}
	if len(output) == 0 {
		return nil
	}
	slices.SortFunc(output, func(left, right rapidpro.ContactField) int {
		leftBuiltIn := strings.EqualFold(left.ValueType, rapidProBuiltInType)
		rightBuiltIn := strings.EqualFold(right.ValueType, rapidProBuiltInType)
		switch {
		case leftBuiltIn && !rightBuiltIn:
			return -1
		case !leftBuiltIn && rightBuiltIn:
			return 1
		default:
			return strings.Compare(strings.ToLower(left.Label), strings.ToLower(right.Label))
		}
	})
	return output
}

func applySuggestedRapidProMappings(current []RapidProReporterFieldMapping, fields []rapidpro.ContactField) []RapidProReporterFieldMapping {
	result := normalizeRapidProMappings(current)
	if len(fields) == 0 {
		return result
	}
	indexBySource := make(map[string]int, len(result))
	for idx, item := range result {
		indexBySource[strings.ToLower(item.SourceKey)] = idx
	}
	for _, field := range fields {
		switch strings.ToLower(strings.TrimSpace(field.Key)) {
		case "facility":
			if _, ok := indexBySource["facilityname"]; !ok {
				result = append(result, RapidProReporterFieldMapping{
					SourceKey:        "facilityName",
					SourceLabel:      rapidProReporterSourceLabels["facilityName"],
					RapidProFieldKey: field.Key,
				})
			}
		case "facilitycode":
			if _, ok := indexBySource["facilityuid"]; !ok {
				result = append(result, RapidProReporterFieldMapping{
					SourceKey:        "facilityUID",
					SourceLabel:      rapidProReporterSourceLabels["facilityUID"],
					RapidProFieldKey: field.Key,
				})
			}
		}
	}
	return normalizeRapidProMappings(result)
}
