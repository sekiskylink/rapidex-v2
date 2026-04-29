package rapidex

import (
	"strings"
	"testing"
)

func TestParsePartialReportMessageOrdersConfiguredIndicators(t *testing.T) {
	cfg := PartialReportParserConfig{Keyword: "cases", Indicators: []string{"ma", "dy", "tf"}}

	got, err := ParsePartialReportMessage("cases.tf.1.dy.4.ma.2", cfg)
	if err != nil {
		t.Fatalf("parse report: %v", err)
	}
	if got.OrderedValueString != "2.4.1" {
		t.Fatalf("expected ordered value string 2.4.1, got %q", got.OrderedValueString)
	}
	if got.NormalizedMessage != "cases.tf.1.dy.4.ma.2" {
		t.Fatalf("unexpected normalized message: %q", got.NormalizedMessage)
	}
}

func TestParsePartialReportMessageFillsMissingIndicatorsWithZero(t *testing.T) {
	cfg := PartialReportParserConfig{Keyword: "cases", Indicators: []string{"ma", "dy", "tf"}}

	got, err := ParsePartialReportMessage("cases.dy.4.ma.2", cfg)
	if err != nil {
		t.Fatalf("parse report: %v", err)
	}
	if got.OrderedValueString != "2.4.0" {
		t.Fatalf("expected ordered value string 2.4.0, got %q", got.OrderedValueString)
	}
	if len(got.MissingIndicators) != 1 || got.MissingIndicators[0] != "tf" {
		t.Fatalf("expected tf to be missing, got %#v", got.MissingIndicators)
	}
}

func TestParsePartialReportMessageSupportsMixedDelimitersAndMergedTokens(t *testing.T) {
	cfg := PartialReportParserConfig{Keyword: "cases", Indicators: []string{"ma", "dy", "tf"}}

	got, err := ParsePartialReportMessage("cases, tf1 dy4 ma2", cfg)
	if err != nil {
		t.Fatalf("parse report: %v", err)
	}
	if got.OrderedValueString != "2.4.1" {
		t.Fatalf("expected ordered value string 2.4.1, got %q", got.OrderedValueString)
	}
}

func TestParsePartialReportMessageIgnoresUnknownIndicatorsButReportsThem(t *testing.T) {
	cfg := PartialReportParserConfig{Keyword: "cases", Indicators: []string{"ma", "dy", "tf"}}

	got, err := ParsePartialReportMessage("cases.zz.9.tf.1.dy.4.ma.2", cfg)
	if err != nil {
		t.Fatalf("parse report: %v", err)
	}
	if got.OrderedValueString != "2.4.1" {
		t.Fatalf("expected ordered value string 2.4.1, got %q", got.OrderedValueString)
	}
	if len(got.UnknownIndicators) != 1 || got.UnknownIndicators[0] != "zz" {
		t.Fatalf("expected unknown indicator zz, got %#v", got.UnknownIndicators)
	}
}

func TestParsePartialReportMessageRejectsOddIndicatorValueTokens(t *testing.T) {
	cfg := PartialReportParserConfig{Keyword: "cases", Indicators: []string{"ma", "dy", "tf"}}

	_, err := ParsePartialReportMessage("cases.ma.2.dy", cfg)
	if err == nil || !strings.Contains(err.Error(), "not all indicators have a value") {
		t.Fatalf("expected odd-token validation error, got %v", err)
	}
}

func TestParsePartialReportMessageRejectsDuplicateIndicator(t *testing.T) {
	cfg := PartialReportParserConfig{Keyword: "cases", Indicators: []string{"ma", "dy", "tf"}}

	_, err := ParsePartialReportMessage("cases.ma.2.ma.5", cfg)
	if err == nil || !strings.Contains(err.Error(), `indicator "ma" appears more than once`) {
		t.Fatalf("expected duplicate indicator validation error, got %v", err)
	}
}

func TestValidatePartialReportParserConfigRejectsDuplicates(t *testing.T) {
	err := ValidatePartialReportParserConfig(PartialReportParserConfig{
		Keyword:    "cases",
		Indicators: []string{"ma", "ma"},
	})
	if err == nil || !strings.Contains(err.Error(), `duplicate indicator "ma"`) {
		t.Fatalf("expected duplicate indicator config error, got %v", err)
	}
}
