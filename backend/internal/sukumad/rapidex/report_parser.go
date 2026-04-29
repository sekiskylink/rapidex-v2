package rapidex

import (
	"context"
	"fmt"
	"strings"
	"unicode"
)

type PartialReportParserConfig struct {
	Keyword    string   `json:"keyword" yaml:"keyword"`
	Indicators []string `json:"indicators" yaml:"indicators"`
}

type PartialReportMatchedIndicator struct {
	IndicatorCode      string `json:"indicatorCode"`
	ConfiguredPosition int    `json:"configuredPosition"`
	Value              string `json:"value"`
}

type PartialReportParseResult struct {
	Keyword            string                          `json:"keyword"`
	NormalizedMessage  string                          `json:"normalizedMessage"`
	OrderedValues      []string                        `json:"orderedValues"`
	OrderedValueString string                          `json:"orderedValueString"`
	MatchedIndicators  []PartialReportMatchedIndicator `json:"matchedIndicators"`
	MissingIndicators  []string                        `json:"missingIndicators"`
	UnknownIndicators  []string                        `json:"unknownIndicators"`
}

type PartialReportParserProvider interface {
	GetByKeyword(context.Context, string) (PartialReportParserConfig, bool, error)
}

func ValidatePartialReportParserConfig(cfg PartialReportParserConfig) error {
	keyword := normalizePartialReportToken(cfg.Keyword)
	if keyword == "" {
		return fmt.Errorf("keyword is required")
	}
	if !isSupportedPartialReportToken(keyword) {
		return fmt.Errorf("keyword %q must contain only letters and digits", keyword)
	}
	if len(cfg.Indicators) == 0 {
		return fmt.Errorf("at least one indicator is required")
	}

	seen := map[string]struct{}{}
	for index, indicator := range cfg.Indicators {
		normalized := normalizePartialReportToken(indicator)
		if normalized == "" {
			return fmt.Errorf("indicator[%d] is required", index)
		}
		if !isSupportedPartialReportToken(normalized) {
			return fmt.Errorf("indicator[%d] %q must contain only letters and digits", index, normalized)
		}
		if _, exists := seen[normalized]; exists {
			return fmt.Errorf("duplicate indicator %q", normalized)
		}
		seen[normalized] = struct{}{}
	}

	return nil
}

func NormalizePartialReportParserConfig(input PartialReportParserConfig) PartialReportParserConfig {
	output := PartialReportParserConfig{
		Keyword: normalizePartialReportToken(input.Keyword),
	}
	output.Indicators = make([]string, 0, len(input.Indicators))
	for _, indicator := range input.Indicators {
		normalized := normalizePartialReportToken(indicator)
		if normalized == "" {
			continue
		}
		output.Indicators = append(output.Indicators, normalized)
	}
	return output
}

func ParsePartialReportMessage(message string, cfg PartialReportParserConfig) (PartialReportParseResult, error) {
	normalizedCfg := NormalizePartialReportParserConfig(cfg)
	if err := ValidatePartialReportParserConfig(normalizedCfg); err != nil {
		return PartialReportParseResult{}, err
	}

	tokens := tokenizePartialReportMessage(message)
	if len(tokens) == 0 {
		return PartialReportParseResult{}, fmt.Errorf("message is required")
	}

	keyword := normalizePartialReportToken(tokens[0])
	if keyword == "" {
		return PartialReportParseResult{}, fmt.Errorf("message keyword is required")
	}
	if keyword != normalizedCfg.Keyword {
		return PartialReportParseResult{}, fmt.Errorf("message keyword %q does not match configured keyword %q", keyword, normalizedCfg.Keyword)
	}

	remaining := tokens[1:]
	if len(remaining)%2 != 0 {
		return PartialReportParseResult{}, fmt.Errorf("not all indicators have a value")
	}

	positions := make(map[string]int, len(normalizedCfg.Indicators))
	orderedValues := make([]string, len(normalizedCfg.Indicators))
	for index, indicator := range normalizedCfg.Indicators {
		positions[indicator] = index
		orderedValues[index] = "0"
	}

	matched := make([]PartialReportMatchedIndicator, 0, len(remaining)/2)
	unknown := make([]string, 0)
	unknownSeen := map[string]struct{}{}
	seen := map[string]struct{}{}

	for index := 0; index < len(remaining); index += 2 {
		code := normalizePartialReportToken(remaining[index])
		value := strings.TrimSpace(remaining[index+1])
		if code == "" {
			continue
		}
		position, known := positions[code]
		if !known {
			if _, exists := unknownSeen[code]; !exists {
				unknownSeen[code] = struct{}{}
				unknown = append(unknown, code)
			}
			continue
		}
		if _, exists := seen[code]; exists {
			return PartialReportParseResult{}, fmt.Errorf("indicator %q appears more than once", code)
		}
		seen[code] = struct{}{}
		orderedValues[position] = value
		matched = append(matched, PartialReportMatchedIndicator{
			IndicatorCode:      code,
			ConfiguredPosition: position + 1,
			Value:              value,
		})
	}

	missing := make([]string, 0, len(normalizedCfg.Indicators))
	for _, indicator := range normalizedCfg.Indicators {
		if _, exists := seen[indicator]; !exists {
			missing = append(missing, indicator)
		}
	}

	return PartialReportParseResult{
		Keyword:            keyword,
		NormalizedMessage:  strings.Join(tokens, "."),
		OrderedValues:      orderedValues,
		OrderedValueString: strings.Join(orderedValues, "."),
		MatchedIndicators:  matched,
		MissingIndicators:  missing,
		UnknownIndicators:  unknown,
	}, nil
}

func tokenizePartialReportMessage(message string) []string {
	rawTokens := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(message)), func(r rune) bool {
		return r == '.' || r == ',' || unicode.IsSpace(r)
	})
	tokens := make([]string, 0, len(rawTokens))
	for _, raw := range rawTokens {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		tokens = append(tokens, splitMergedPartialReportToken(raw)...)
	}
	return tokens
}

func splitMergedPartialReportToken(token string) []string {
	for index, r := range token {
		if !unicode.IsDigit(r) {
			continue
		}
		if index == 0 {
			return []string{token}
		}
		code := strings.TrimSpace(token[:index])
		value := strings.TrimSpace(token[index:])
		if code == "" || value == "" {
			return []string{token}
		}
		if !isSupportedPartialReportToken(code) {
			return []string{token}
		}
		return []string{code, value}
	}
	return []string{token}
}

func normalizePartialReportToken(token string) string {
	return strings.ToLower(strings.TrimSpace(token))
}

func isSupportedPartialReportToken(token string) bool {
	if token == "" {
		return false
	}
	for _, r := range token {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
