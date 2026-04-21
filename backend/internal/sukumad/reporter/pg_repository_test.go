package reporter

import (
	"testing"
	"time"
)

func TestPreserveSystemManagedFields(t *testing.T) {
	lastReportingDate := time.Date(2026, time.April, 10, 12, 0, 0, 0, time.UTC)
	smsCodeExpiresAt := time.Date(2026, time.April, 11, 12, 0, 0, 0, time.UTC)
	lastLoginAt := time.Date(2026, time.April, 12, 12, 0, 0, 0, time.UTC)

	existing := Reporter{
		ID:                44,
		UID:               "uid-44",
		TotalReports:      17,
		LastReportingDate: &lastReportingDate,
		SMSCode:           "1234",
		SMSCodeExpiresAt:  &smsCodeExpiresAt,
		MTUUID:            "mt-uuid-44",
		Synced:            true,
		LastLoginAt:       &lastLoginAt,
	}
	incoming := Reporter{
		ID:           44,
		Name:         "Alice",
		Telephone:    "+256700000001",
		OrgUnitID:    5,
		IsActive:     true,
		RapidProUUID: "rapidpro-44",
	}

	got := preserveSystemManagedFields(existing, incoming)

	if got.UID != existing.UID {
		t.Fatalf("expected uid %q, got %q", existing.UID, got.UID)
	}
	if got.TotalReports != existing.TotalReports {
		t.Fatalf("expected total reports %d, got %d", existing.TotalReports, got.TotalReports)
	}
	if got.LastReportingDate != existing.LastReportingDate {
		t.Fatalf("expected last reporting date to be preserved")
	}
	if got.SMSCode != existing.SMSCode {
		t.Fatalf("expected sms code %q, got %q", existing.SMSCode, got.SMSCode)
	}
	if got.SMSCodeExpiresAt != existing.SMSCodeExpiresAt {
		t.Fatalf("expected sms code expiry to be preserved")
	}
	if got.MTUUID != existing.MTUUID {
		t.Fatalf("expected mtuuid %q, got %q", existing.MTUUID, got.MTUUID)
	}
	if !got.Synced {
		t.Fatalf("expected synced to remain true")
	}
	if got.LastLoginAt != existing.LastLoginAt {
		t.Fatalf("expected last login to be preserved")
	}
}
