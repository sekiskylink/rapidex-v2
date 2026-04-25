package orgunit

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

var testTime = time.Date(2026, time.April, 25, 10, 0, 0, 0, time.UTC)

func TestPgRepositoryListOrdersRootsAlphabetically(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer sqlDB.Close()

	repo := NewPgRepository(sqlx.NewDb(sqlDB, "sqlmock"))

	countRows := sqlmock.NewRows([]string{"count"}).AddRow(2)
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM org_units WHERE 1=1 AND parent_id IS NULL`).
		WillReturnRows(countRows)

	rows := sqlmock.NewRows([]string{
		"id", "uid", "code", "name", "short_name", "description", "parent_id", "hierarchy_level", "path",
		"display_path", "address", "email", "url", "phone_number", "extras", "attribute_values",
		"opening_date", "deleted", "has_children", "last_sync_date", "created_at", "updated_at",
	}).
		AddRow(2, "uid-2", "B", "Alpha District", "Alpha District", "", nil, 1, "/uid-2", "", "", "", "", "", []byte(`{}`), []byte(`{}`), nil, false, true, nil, testTime, testTime).
		AddRow(1, "uid-1", "A", "Zulu District", "Zulu District", "", nil, 1, "/uid-1", "", "", "", "", "", []byte(`{}`), []byte(`{}`), nil, false, true, nil, testTime, testTime)

	mock.ExpectQuery(`(?s)SELECT id, uid, code, name, short_name, description, parent_id, hierarchy_level, path,.*FROM org_units.*WHERE 1=1 AND parent_id IS NULL.*ORDER BY hierarchy_level ASC, COALESCE\(parent_id, 0\) ASC, LOWER\(name\) ASC, id ASC.*LIMIT 20 OFFSET 0`).
		WillReturnRows(rows)

	result, err := repo.List(context.Background(), ListQuery{RootsOnly: true})
	if err != nil {
		t.Fatalf("list org units: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}
	if result.Items[0].Name != "Alpha District" || result.Items[1].Name != "Zulu District" {
		t.Fatalf("expected alphabetical root ordering, got %q then %q", result.Items[0].Name, result.Items[1].Name)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

func TestPgRepositoryListOrdersChildrenAlphabetically(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer sqlDB.Close()

	repo := NewPgRepository(sqlx.NewDb(sqlDB, "sqlmock"))

	countRows := sqlmock.NewRows([]string{"count"}).AddRow(2)
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM org_units WHERE 1=1 AND parent_id = \?`).
		WithArgs(int64(9)).
		WillReturnRows(countRows)

	rows := sqlmock.NewRows([]string{
		"id", "uid", "code", "name", "short_name", "description", "parent_id", "hierarchy_level", "path",
		"display_path", "address", "email", "url", "phone_number", "extras", "attribute_values",
		"opening_date", "deleted", "has_children", "last_sync_date", "created_at", "updated_at",
	}).
		AddRow(5, "uid-5", "B", "Alpha Health Centre", "Alpha Health Centre", "", 9, 2, "/uid-9/uid-5", "Uganda / Kampala", "", "", "", "", []byte(`{}`), []byte(`{}`), nil, false, false, nil, testTime, testTime).
		AddRow(4, "uid-4", "A", "Zulu Health Centre", "Zulu Health Centre", "", 9, 2, "/uid-9/uid-4", "Uganda / Kampala", "", "", "", "", []byte(`{}`), []byte(`{}`), nil, false, false, nil, testTime, testTime)

	mock.ExpectQuery(`(?s)SELECT id, uid, code, name, short_name, description, parent_id, hierarchy_level, path,.*FROM org_units.*WHERE 1=1 AND parent_id = \?.*ORDER BY hierarchy_level ASC, COALESCE\(parent_id, 0\) ASC, LOWER\(name\) ASC, id ASC.*LIMIT 20 OFFSET 0`).
		WithArgs(int64(9)).
		WillReturnRows(rows)

	parentID := int64(9)
	result, err := repo.List(context.Background(), ListQuery{ParentID: &parentID})
	if err != nil {
		t.Fatalf("list org units: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}
	if result.Items[0].Name != "Alpha Health Centre" || result.Items[1].Name != "Zulu Health Centre" {
		t.Fatalf("expected alphabetical child ordering, got %q then %q", result.Items[0].Name, result.Items[1].Name)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}
