package reporter

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
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

func TestListUpdatedSinceIncludesReportersWithoutRapidProUUID(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer sqlDB.Close()

	repo := NewPgRepository(sqlx.NewDb(sqlDB, "sqlmock"))
	since := time.Date(2026, time.April, 24, 10, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{
		"id", "uid", "name", "telephone", "whatsapp", "telegram", "org_unit_id", "reporting_location",
		"district_id", "orphaned_at", "orphan_reason", "last_known_org_unit_uid", "last_known_org_unit_name", "total_reports", "last_reporting_date", "sms_code", "sms_code_expires_at",
		"mtuuid", "synced", "rapidpro_uuid", "is_active", "created_at", "updated_at", "last_login_at",
	}).AddRow(
		1, "rep-1", "Alice Reporter", "+256700000001", "", "", 4, "Kampala", nil, nil, "", "ou-4", "Kampala", 0, nil, "", nil,
		"", false, "", true, since.Add(-time.Hour), since.Add(time.Minute), nil,
	)
	mock.ExpectQuery(`(?s)SELECT id, uid, name, telephone, whatsapp, telegram, org_unit_id, reporting_location,.*FROM reporters.*WHERE updated_at > \? AND is_active = TRUE.*ORDER BY updated_at ASC, id ASC.*LIMIT 10`).
		WithArgs(since.UTC()).
		WillReturnRows(rows)
	mock.ExpectQuery(`(?s)SELECT reporter_id, group_name FROM reporter_groups WHERE reporter_id IN \(\?\)`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"reporter_id", "group_name"}))

	items, err := repo.ListUpdatedSince(context.Background(), &since, 10, true)
	if err != nil {
		t.Fatalf("list updated since: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one reporter, got %d", len(items))
	}
	if items[0].RapidProUUID != "" {
		t.Fatalf("expected blank rapidpro uuid, got %q", items[0].RapidProUUID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}
