package reporter

import (
	"encoding/json"
	"testing"
)

func TestReporterUnmarshalAcceptsLegacyAliases(t *testing.T) {
	var record Reporter
	if err := json.Unmarshal([]byte(`{
		"displayName":"Alice Reporter",
		"phoneNumber":"+256700000001",
		"groupNames":["Lead","VHT"]
	}`), &record); err != nil {
		t.Fatalf("unmarshal reporter: %v", err)
	}

	if record.Name != "Alice Reporter" {
		t.Fatalf("expected displayName alias to populate name, got %q", record.Name)
	}
	if record.Telephone != "+256700000001" {
		t.Fatalf("expected phoneNumber alias to populate telephone, got %q", record.Telephone)
	}
	if len(record.Groups) != 2 {
		t.Fatalf("expected groupNames alias to populate groups, got %#v", record.Groups)
	}
}

func TestNormalizeGroupsDeduplicatesAndTrims(t *testing.T) {
	got := normalizeGroups([]string{" Lead ", "lead", "", "Supervisor"})
	if len(got) != 2 || got[0] != "Lead" || got[1] != "Supervisor" {
		t.Fatalf("unexpected normalized groups: %#v", got)
	}
}
