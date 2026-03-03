package audit

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestListPaginationIncludesTotalAndOffset(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()

	repo := NewSQLRepository(sqlx.NewDb(sqlDB, "sqlmock"))
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM audit_logs WHERE action ILIKE $1`)).
		WithArgs("%auth%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(12))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, timestamp, actor_user_id, action, entity_type, entity_id, metadata_json
		FROM audit_logs
	 WHERE action ILIKE $1 ORDER BY timestamp DESC LIMIT $2 OFFSET $3`)).
		WithArgs("%auth%", 5, 5).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "timestamp", "actor_user_id", "action", "entity_type", "entity_id", "metadata_json"}).
				AddRow(int64(10), now, int64(1), "auth.login.success", "user", "1", []byte(`{}`)),
		)

	result, err := repo.List(context.Background(), ListFilter{
		Page:      2,
		PageSize:  5,
		SortField: "timestamp",
		SortOrder: "desc",
		Action:    "auth",
	})
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}

	if result.Total != 12 {
		t.Fatalf("expected total=12, got %d", result.Total)
	}
	if result.Page != 2 || result.PageSize != 5 {
		t.Fatalf("expected page metadata 2/5, got %d/%d", result.Page, result.PageSize)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
