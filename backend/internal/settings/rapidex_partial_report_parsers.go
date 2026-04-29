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
	rapidexPartialReportParsersCategory = "rapidex"
	rapidexPartialReportParsersKey      = "partial_report_parsers"
)

type RapidexPartialReportParsersValidation struct {
	IsValid bool     `json:"isValid"`
	Errors  []string `json:"errors,omitempty"`
}

type RapidexPartialReportParserConfig = rapidex.PartialReportParserConfig

type RapidexPartialReportParsersSettings struct {
	Parsers    []rapidex.PartialReportParserConfig   `json:"parsers"`
	Validation RapidexPartialReportParsersValidation `json:"validation"`
}

type RapidexPartialReportParsersUpdateInput struct {
	Parsers []rapidex.PartialReportParserConfig `json:"parsers"`
}

type rapidexPartialReportParsersStored struct {
	Parsers []rapidex.PartialReportParserConfig `json:"parsers"`
}

type RapidexPartialReportParserProvider struct {
	repo Repository
}

func NewRapidexPartialReportParserProvider(repo Repository) *RapidexPartialReportParserProvider {
	return &RapidexPartialReportParserProvider{repo: repo}
}

func (p *RapidexPartialReportParserProvider) GetByKeyword(ctx context.Context, keyword string) (rapidex.PartialReportParserConfig, bool, error) {
	stored, err := p.getStored(ctx)
	if err != nil {
		return rapidex.PartialReportParserConfig{}, false, err
	}
	normalizedKeyword := strings.ToLower(strings.TrimSpace(keyword))
	for _, cfg := range stored.Parsers {
		if strings.EqualFold(strings.TrimSpace(cfg.Keyword), normalizedKeyword) {
			return cfg, true, nil
		}
	}
	return rapidex.PartialReportParserConfig{}, false, nil
}

func (p *RapidexPartialReportParserProvider) getStored(ctx context.Context) (rapidexPartialReportParsersStored, error) {
	raw, err := p.repo.Get(ctx, rapidexPartialReportParsersCategory, rapidexPartialReportParsersKey)
	if err != nil {
		if err == ErrNotFound {
			return rapidexPartialReportParsersStored{}, nil
		}
		return rapidexPartialReportParsersStored{}, err
	}
	var stored rapidexPartialReportParsersStored
	if unmarshalErr := json.Unmarshal(raw, &stored); unmarshalErr != nil {
		return rapidexPartialReportParsersStored{}, nil
	}
	stored.Parsers = normalizeRapidexPartialReportParsers(stored.Parsers)
	return stored, nil
}

func (s *Service) GetRapidexPartialReportParsers(ctx context.Context) (RapidexPartialReportParsersSettings, error) {
	stored, err := s.getRapidexPartialReportParsersStored(ctx)
	if err != nil {
		return RapidexPartialReportParsersSettings{}, err
	}
	return materializeRapidexPartialReportParsers(stored), nil
}

func (s *Service) UpdateRapidexPartialReportParsers(ctx context.Context, input RapidexPartialReportParsersUpdateInput, actorUserID *int64) (RapidexPartialReportParsersSettings, error) {
	parsers, err := validateRapidexPartialReportParsers(input.Parsers)
	if err != nil {
		return RapidexPartialReportParsersSettings{}, err
	}
	stored := rapidexPartialReportParsersStored{Parsers: parsers}
	if err := s.saveRapidexPartialReportParsersStored(ctx, stored, actorUserID); err != nil {
		return RapidexPartialReportParsersSettings{}, err
	}
	result := materializeRapidexPartialReportParsers(stored)
	s.logAudit(ctx, audit.Event{
		Action:      "settings.rapidex_partial_report_parsers.update",
		ActorUserID: actorUserID,
		EntityType:  "settings",
		EntityID:    strPtr("rapidex.partial_report_parsers"),
		Metadata: map[string]any{
			"parserCount": len(result.Parsers),
		},
	})
	return result, nil
}

func (s *Service) getRapidexPartialReportParsersStored(ctx context.Context) (rapidexPartialReportParsersStored, error) {
	return NewRapidexPartialReportParserProvider(s.repo).getStored(ctx)
}

func (s *Service) saveRapidexPartialReportParsersStored(ctx context.Context, stored rapidexPartialReportParsersStored, actorUserID *int64) error {
	payload, err := json.Marshal(stored)
	if err != nil {
		return fmt.Errorf("marshal rapidex partial report parsers setting: %w", err)
	}
	return s.repo.Upsert(ctx, rapidexPartialReportParsersCategory, rapidexPartialReportParsersKey, payload, actorUserID, time.Now().UTC())
}

func materializeRapidexPartialReportParsers(stored rapidexPartialReportParsersStored) RapidexPartialReportParsersSettings {
	result := RapidexPartialReportParsersSettings{
		Parsers: normalizeRapidexPartialReportParsers(stored.Parsers),
	}
	result.Validation = validateRapidexPartialReportParsersForResponse(result.Parsers)
	return result
}

func validateRapidexPartialReportParsersForResponse(input []rapidex.PartialReportParserConfig) RapidexPartialReportParsersValidation {
	_, err := validateRapidexPartialReportParsers(input)
	if err != nil {
		return RapidexPartialReportParsersValidation{IsValid: false, Errors: []string{err.Error()}}
	}
	return RapidexPartialReportParsersValidation{IsValid: true}
}

func validateRapidexPartialReportParsers(input []rapidex.PartialReportParserConfig) ([]rapidex.PartialReportParserConfig, error) {
	if len(input) == 0 {
		return []rapidex.PartialReportParserConfig{}, nil
	}
	result := make([]rapidex.PartialReportParserConfig, 0, len(input))
	seenKeywords := map[string]struct{}{}
	for index, item := range input {
		normalized := normalizeRapidexPartialReportParser(item)
		if err := rapidex.ValidatePartialReportParserConfig(normalized); err != nil {
			return nil, apperror.ValidationWithDetails("validation failed", map[string]any{
				"parsers": []string{fmt.Sprintf("parser[%d]: %v", index, err)},
			})
		}
		if _, exists := seenKeywords[normalized.Keyword]; exists {
			return nil, apperror.ValidationWithDetails("validation failed", map[string]any{
				"parsers": []string{fmt.Sprintf("duplicate keyword %q", normalized.Keyword)},
			})
		}
		seenKeywords[normalized.Keyword] = struct{}{}
		result = append(result, normalized)
	}
	slices.SortFunc(result, func(left, right rapidex.PartialReportParserConfig) int {
		return strings.Compare(left.Keyword, right.Keyword)
	})
	return result, nil
}

func normalizeRapidexPartialReportParsers(input []rapidex.PartialReportParserConfig) []rapidex.PartialReportParserConfig {
	if len(input) == 0 {
		return []rapidex.PartialReportParserConfig{}
	}
	output := make([]rapidex.PartialReportParserConfig, 0, len(input))
	for _, item := range input {
		normalized := normalizeRapidexPartialReportParser(item)
		if normalized.Keyword == "" && len(normalized.Indicators) == 0 {
			continue
		}
		output = append(output, normalized)
	}
	slices.SortFunc(output, func(left, right rapidex.PartialReportParserConfig) int {
		return strings.Compare(left.Keyword, right.Keyword)
	})
	return output
}

func normalizeRapidexPartialReportParser(input rapidex.PartialReportParserConfig) rapidex.PartialReportParserConfig {
	return rapidex.NormalizePartialReportParserConfig(input)
}
