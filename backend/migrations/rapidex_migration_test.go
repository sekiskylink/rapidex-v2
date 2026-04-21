package migrations

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRapidexMigrationsHaveUpAndDownPairs(t *testing.T) {
	files, err := filepath.Glob("00002[6-9]_*.sql")
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected numbered rapidex migrations")
	}

	seen := map[string]map[string]bool{}
	for _, file := range files {
		base := filepath.Base(file)
		version := strings.TrimSuffix(strings.TrimSuffix(base, ".up.sql"), ".down.sql")
		if _, ok := seen[version]; !ok {
			seen[version] = map[string]bool{}
		}
		if strings.HasSuffix(base, ".up.sql") {
			seen[version]["up"] = true
			continue
		}
		if strings.HasSuffix(base, ".down.sql") {
			seen[version]["down"] = true
			continue
		}
		t.Fatalf("migration %s does not use .up.sql/.down.sql suffix", base)
	}

	for version, pair := range seen {
		if !pair["up"] || !pair["down"] {
			t.Fatalf("migration %s missing up/down pair: %#v", version, pair)
		}
	}
}

func TestRapidexMigrationDeclaresRequiredTablesAndRollback(t *testing.T) {
	firstUp := readMigration(t, "000026_create_orgunits_and_reporters.up.sql")
	for _, fragment := range []string{
		"CREATE TABLE org_units",
		"id BIGSERIAL PRIMARY KEY",
		"parent_id BIGINT REFERENCES org_units(id) ON DELETE RESTRICT",
		"CREATE INDEX idx_org_units_path ON org_units (path)",
		"CREATE TABLE reporters",
		"org_unit_id BIGINT NOT NULL REFERENCES org_units(id) ON DELETE RESTRICT",
	} {
		if !strings.Contains(firstUp, fragment) {
			t.Fatalf("expected first rapidex up migration to contain %q", fragment)
		}
	}

	firstDown := readMigration(t, "000026_create_orgunits_and_reporters.down.sql")
	for _, fragment := range []string{
		"DROP TABLE IF EXISTS reporters",
		"DROP TABLE IF EXISTS org_units",
	} {
		if !strings.Contains(firstDown, fragment) {
			t.Fatalf("expected first rapidex down migration to contain %q", fragment)
		}
	}

	userOrgUp := readMigration(t, "000028_create_user_org_units.up.sql")
	for _, fragment := range []string{
		"CREATE TABLE user_org_units",
		"user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE",
		"org_unit_id BIGINT NOT NULL REFERENCES org_units(id) ON DELETE CASCADE",
	} {
		if !strings.Contains(userOrgUp, fragment) {
			t.Fatalf("expected user org units migration to contain %q", fragment)
		}
	}

	enrichedUp := readMigration(t, "000029_enrich_orgunits_and_reporters.up.sql")
	for _, fragment := range []string{
		"ADD COLUMN IF NOT EXISTS short_name",
		"ADD COLUMN IF NOT EXISTS hierarchy_level",
		"ADD COLUMN IF NOT EXISTS attribute_values JSONB",
		"CREATE TABLE IF NOT EXISTS reporter_groups",
		"ADD COLUMN IF NOT EXISTS whatsapp",
		"ADD COLUMN IF NOT EXISTS rapidpro_uuid",
	} {
		if !strings.Contains(enrichedUp, fragment) {
			t.Fatalf("expected enriched rapidex migration to contain %q", fragment)
		}
	}

	enrichedDown := readMigration(t, "000029_enrich_orgunits_and_reporters.down.sql")
	for _, fragment := range []string{
		"DROP TABLE IF EXISTS reporter_groups",
		"DROP COLUMN IF EXISTS hierarchy_level",
		"DROP COLUMN IF EXISTS rapidpro_uuid",
	} {
		if !strings.Contains(enrichedDown, fragment) {
			t.Fatalf("expected enriched rapidex rollback to contain %q", fragment)
		}
	}
}

func readMigration(t *testing.T, name string) string {
	t.Helper()
	content, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read migration %s: %v", name, err)
	}
	return string(content)
}
