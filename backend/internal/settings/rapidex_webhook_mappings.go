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
	"basepro/backend/internal/sukumad/rapidex"
)

const (
	rapidexWebhookMappingsCategory = "rapidex"
	rapidexWebhookMappingsKey      = "webhook_mappings"
)

type RapidexWebhookMappingsValidation struct {
	IsValid bool     `json:"isValid"`
	Errors  []string `json:"errors,omitempty"`
}

type RapidexWebhookMappingConfig = rapidex.MappingConfig

type RapidexWebhookMappingsSettings struct {
	RapidProServerCode string                           `json:"rapidProServerCode"`
	Dhis2ServerCode    string                           `json:"dhis2ServerCode"`
	Mappings           []rapidex.MappingConfig          `json:"mappings"`
	Validation         RapidexWebhookMappingsValidation `json:"validation"`
}

type RapidexWebhookMappingsUpdateInput struct {
	RapidProServerCode string                  `json:"rapidProServerCode"`
	Dhis2ServerCode    string                  `json:"dhis2ServerCode"`
	Mappings           []rapidex.MappingConfig `json:"mappings"`
}

type RapidexWebhookMappingsImportInput struct {
	YAML string `json:"yaml"`
}

type RapidexWebhookMappingsExport struct {
	YAML string `json:"yaml"`
}

type rapidexWebhookMappingsStored struct {
	RapidProServerCode string                  `json:"rapidProServerCode"`
	Dhis2ServerCode    string                  `json:"dhis2ServerCode"`
	Mappings           []rapidex.MappingConfig `json:"mappings"`
}

type RapidexWebhookMappingProvider struct {
	repo Repository
}

func NewRapidexWebhookMappingProvider(repo Repository) *RapidexWebhookMappingProvider {
	return &RapidexWebhookMappingProvider{repo: repo}
}

func (p *RapidexWebhookMappingProvider) GetByFlowUUID(ctx context.Context, flowUUID string) (rapidex.WebhookBinding, bool, error) {
	stored, err := p.getStored(ctx)
	if err != nil {
		return rapidex.WebhookBinding{}, false, err
	}
	normalizedFlowUUID := strings.TrimSpace(flowUUID)
	for _, cfg := range stored.Mappings {
		if strings.TrimSpace(cfg.FlowUUID) == normalizedFlowUUID {
			return rapidex.WebhookBinding{
				MappingConfig:      cfg,
				RapidProServerCode: stored.RapidProServerCode,
				DHIS2ServerCode:    stored.Dhis2ServerCode,
			}, true, nil
		}
	}
	return rapidex.WebhookBinding{}, false, nil
}

func (p *RapidexWebhookMappingProvider) getStored(ctx context.Context) (rapidexWebhookMappingsStored, error) {
	raw, err := p.repo.Get(ctx, rapidexWebhookMappingsCategory, rapidexWebhookMappingsKey)
	if err != nil {
		if err == ErrNotFound {
			return rapidexWebhookMappingsStored{}, nil
		}
		return rapidexWebhookMappingsStored{}, err
	}
	var stored rapidexWebhookMappingsStored
	if unmarshalErr := json.Unmarshal(raw, &stored); unmarshalErr != nil {
		return rapidexWebhookMappingsStored{}, nil
	}
	stored.RapidProServerCode = normalizeRapidProServerCode(stored.RapidProServerCode)
	stored.Dhis2ServerCode = normalizeRapidexDHIS2ServerCode(stored.Dhis2ServerCode)
	stored.Mappings = normalizeRapidexWebhookMappings(stored.Mappings)
	return stored, nil
}

func (s *Service) GetRapidexWebhookMappings(ctx context.Context) (RapidexWebhookMappingsSettings, error) {
	stored, err := s.getRapidexWebhookMappingsStored(ctx)
	if err != nil {
		return RapidexWebhookMappingsSettings{}, err
	}
	return materializeRapidexWebhookMappings(stored), nil
}

func (s *Service) UpdateRapidexWebhookMappings(ctx context.Context, input RapidexWebhookMappingsUpdateInput, actorUserID *int64) (RapidexWebhookMappingsSettings, error) {
	mappings, err := validateRapidexWebhookMappings(input.Mappings)
	if err != nil {
		return RapidexWebhookMappingsSettings{}, err
	}
	stored := rapidexWebhookMappingsStored{
		RapidProServerCode: normalizeRapidProServerCode(input.RapidProServerCode),
		Dhis2ServerCode:    normalizeRapidexDHIS2ServerCode(input.Dhis2ServerCode),
		Mappings:           mappings,
	}
	if err := s.saveRapidexWebhookMappingsStored(ctx, stored, actorUserID); err != nil {
		return RapidexWebhookMappingsSettings{}, err
	}
	result := materializeRapidexWebhookMappings(stored)
	s.logAudit(ctx, audit.Event{
		Action:      "settings.rapidex_webhook_mappings.update",
		ActorUserID: actorUserID,
		EntityType:  "settings",
		EntityID:    strPtr("rapidex.webhook_mappings"),
		Metadata: map[string]any{
			"mappingCount": len(result.Mappings),
		},
	})
	return result, nil
}

func (s *Service) ImportRapidexWebhookMappingsYAML(ctx context.Context, input RapidexWebhookMappingsImportInput, actorUserID *int64) (RapidexWebhookMappingsSettings, error) {
	if strings.TrimSpace(input.YAML) == "" {
		return RapidexWebhookMappingsSettings{}, apperror.ValidationWithDetails("validation failed", map[string]any{
			"yaml": []string{"is required"},
		})
	}
	mappings, err := rapidex.ParseMappingConfigsYAML(input.YAML)
	if err != nil {
		return RapidexWebhookMappingsSettings{}, apperror.ValidationWithDetails("validation failed", map[string]any{
			"yaml": []string{fmt.Sprintf("invalid RapidEx mapping YAML: %v", err)},
		})
	}
	stored, err := s.getRapidexWebhookMappingsStored(ctx)
	if err != nil {
		return RapidexWebhookMappingsSettings{}, err
	}
	return s.UpdateRapidexWebhookMappings(ctx, RapidexWebhookMappingsUpdateInput{
		RapidProServerCode: stored.RapidProServerCode,
		Dhis2ServerCode:    stored.Dhis2ServerCode,
		Mappings:           mappings,
	}, actorUserID)
}

func (s *Service) ExportRapidexWebhookMappingsYAML(ctx context.Context) (RapidexWebhookMappingsExport, error) {
	stored, err := s.getRapidexWebhookMappingsStored(ctx)
	if err != nil {
		return RapidexWebhookMappingsExport{}, err
	}
	yamlText, err := rapidex.MarshalMappingConfigsYAML(stored.Mappings)
	if err != nil {
		return RapidexWebhookMappingsExport{}, err
	}
	return RapidexWebhookMappingsExport{YAML: yamlText}, nil
}

func (s *Service) getRapidexWebhookMappingsStored(ctx context.Context) (rapidexWebhookMappingsStored, error) {
	return NewRapidexWebhookMappingProvider(s.repo).getStored(ctx)
}

func (s *Service) saveRapidexWebhookMappingsStored(ctx context.Context, stored rapidexWebhookMappingsStored, actorUserID *int64) error {
	payload, err := json.Marshal(stored)
	if err != nil {
		return fmt.Errorf("marshal rapidex webhook mappings setting: %w", err)
	}
	return s.repo.Upsert(ctx, rapidexWebhookMappingsCategory, rapidexWebhookMappingsKey, payload, actorUserID, time.Now().UTC())
}

func materializeRapidexWebhookMappings(stored rapidexWebhookMappingsStored) RapidexWebhookMappingsSettings {
	result := RapidexWebhookMappingsSettings{
		RapidProServerCode: normalizeRapidProServerCode(stored.RapidProServerCode),
		Dhis2ServerCode:    normalizeRapidexDHIS2ServerCode(stored.Dhis2ServerCode),
		Mappings:           normalizeRapidexWebhookMappings(stored.Mappings),
	}
	result.Validation = validateRapidexWebhookMappingsForResponse(result.Mappings)
	return result
}

func validateRapidexWebhookMappingsForResponse(input []rapidex.MappingConfig) RapidexWebhookMappingsValidation {
	_, err := validateRapidexWebhookMappings(input)
	if err != nil {
		return RapidexWebhookMappingsValidation{IsValid: false, Errors: []string{err.Error()}}
	}
	return RapidexWebhookMappingsValidation{IsValid: true}
}

func validateRapidexWebhookMappings(input []rapidex.MappingConfig) ([]rapidex.MappingConfig, error) {
	if len(input) == 0 {
		return nil, apperror.ValidationWithDetails("validation failed", map[string]any{
			"mappings": []string{"at least one mapping is required"},
		})
	}
	result := make([]rapidex.MappingConfig, 0, len(input))
	seenFlowUUIDs := map[string]struct{}{}
	for index, item := range input {
		normalized := normalizeRapidexWebhookMapping(item)
		if err := rapidex.ValidateMappingConfig(normalized); err != nil {
			return nil, apperror.ValidationWithDetails("validation failed", map[string]any{
				"mappings": []string{fmt.Sprintf("mapping[%d]: %v", index, err)},
			})
		}
		flowUUID := strings.TrimSpace(normalized.FlowUUID)
		if _, exists := seenFlowUUIDs[flowUUID]; exists {
			return nil, apperror.ValidationWithDetails("validation failed", map[string]any{
				"mappings": []string{fmt.Sprintf("duplicate flow_uuid %q", flowUUID)},
			})
		}
		seenFlowUUIDs[flowUUID] = struct{}{}
		result = append(result, normalized)
	}
	slices.SortFunc(result, func(left, right rapidex.MappingConfig) int {
		leftName := strings.ToLower(strings.TrimSpace(left.FlowName))
		rightName := strings.ToLower(strings.TrimSpace(right.FlowName))
		if leftName == rightName {
			return strings.Compare(strings.ToLower(strings.TrimSpace(left.FlowUUID)), strings.ToLower(strings.TrimSpace(right.FlowUUID)))
		}
		return strings.Compare(leftName, rightName)
	})
	return result, nil
}

func normalizeRapidexWebhookMappings(input []rapidex.MappingConfig) []rapidex.MappingConfig {
	if len(input) == 0 {
		return []rapidex.MappingConfig{}
	}
	output := make([]rapidex.MappingConfig, 0, len(input))
	for _, item := range input {
		normalized := normalizeRapidexWebhookMapping(item)
		if strings.TrimSpace(normalized.FlowUUID) == "" && strings.TrimSpace(normalized.Dataset) == "" && len(normalized.Mappings) == 0 {
			continue
		}
		output = append(output, normalized)
	}
	slices.SortFunc(output, func(left, right rapidex.MappingConfig) int {
		leftName := strings.ToLower(strings.TrimSpace(left.FlowName))
		rightName := strings.ToLower(strings.TrimSpace(right.FlowName))
		if leftName == rightName {
			return strings.Compare(strings.ToLower(strings.TrimSpace(left.FlowUUID)), strings.ToLower(strings.TrimSpace(right.FlowUUID)))
		}
		return strings.Compare(leftName, rightName)
	})
	return output
}

func normalizeRapidexWebhookMapping(input rapidex.MappingConfig) rapidex.MappingConfig {
	output := rapidex.MappingConfig{
		FlowUUID:   strings.TrimSpace(input.FlowUUID),
		FlowName:   strings.TrimSpace(input.FlowName),
		Dataset:    strings.TrimSpace(input.Dataset),
		OrgUnitVar: strings.TrimSpace(input.OrgUnitVar),
		PeriodVar:  strings.TrimSpace(input.PeriodVar),
		PayloadAOC: strings.TrimSpace(input.PayloadAOC),
		Mappings:   make([]rapidex.DataValueMapping, 0, len(input.Mappings)),
	}
	for _, item := range input.Mappings {
		output.Mappings = append(output.Mappings, rapidex.DataValueMapping{
			Field:                strings.TrimSpace(item.Field),
			DataElement:          strings.TrimSpace(item.DataElement),
			CategoryOptionCombo:  strings.TrimSpace(item.CategoryOptionCombo),
			AttributeOptionCombo: strings.TrimSpace(item.AttributeOptionCombo),
		})
	}
	return output
}
