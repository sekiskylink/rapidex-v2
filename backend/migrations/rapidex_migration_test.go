package migrations

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRapidexMigrationsHaveUpAndDownPairs(t *testing.T) {
	files, err := filepath.Glob("0000{2[6-9],3[0-4]}_*.sql")
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	if len(files) == 0 {
		files, err = filepath.Glob("00002[6-9]_*.sql")
		if err != nil {
			t.Fatalf("glob fallback migrations: %v", err)
		}
		extra, err := filepath.Glob("00003[0-4]_*.sql")
		if err != nil {
			t.Fatalf("glob 00003x migrations: %v", err)
		}
		files = append(files, extra...)
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
		"phone_number VARCHAR(32) NOT NULL UNIQUE",
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

	dropContactUp := readMigration(t, "000030_drop_reporter_contact_uuid.up.sql")
	if !strings.Contains(dropContactUp, "DROP COLUMN IF EXISTS contact_uuid") {
		t.Fatal("expected contact uuid drop migration to remove contact_uuid")
	}

	dropContactDown := readMigration(t, "000030_drop_reporter_contact_uuid.down.sql")
	if !strings.Contains(dropContactDown, "ADD COLUMN IF NOT EXISTS contact_uuid") {
		t.Fatal("expected contact uuid rollback migration to restore contact_uuid")
	}

	reporterGroupCatalogUp := readMigration(t, "000031_create_reporter_group_catalog.up.sql")
	for _, fragment := range []string{
		"CREATE TABLE IF NOT EXISTS reporter_group_catalog",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_reporter_group_catalog_name_unique",
		"INSERT INTO reporter_group_catalog",
	} {
		if !strings.Contains(reporterGroupCatalogUp, fragment) {
			t.Fatalf("expected reporter group catalog migration to contain %q", fragment)
		}
	}

	reporterGroupCatalogDown := readMigration(t, "000031_create_reporter_group_catalog.down.sql")
	if !strings.Contains(reporterGroupCatalogDown, "DROP TABLE IF EXISTS reporter_group_catalog") {
		t.Fatal("expected reporter group catalog rollback to drop reporter_group_catalog")
	}

	reporterBroadcastsUp := readMigration(t, "000032_create_reporter_broadcasts.up.sql")
	if !strings.Contains(reporterBroadcastsUp, "CREATE TABLE reporter_broadcasts") {
		t.Fatal("expected reporter broadcasts migration to create reporter_broadcasts")
	}

	hierarchySyncUp := readMigration(t, "000033_add_dhis2_hierarchy_sync_metadata.up.sql")
	for _, fragment := range []string{
		"CREATE TABLE IF NOT EXISTS org_unit_levels",
		"CREATE TABLE IF NOT EXISTS org_unit_groups",
		"CREATE TABLE IF NOT EXISTS org_unit_attributes",
		"CREATE TABLE IF NOT EXISTS org_unit_group_members",
		"ADD COLUMN IF NOT EXISTS district_level_name",
		"ADD COLUMN IF NOT EXISTS last_counts JSONB",
	} {
		if !strings.Contains(hierarchySyncUp, fragment) {
			t.Fatalf("expected hierarchy sync migration to contain %q", fragment)
		}
	}

	hierarchySyncDown := readMigration(t, "000033_add_dhis2_hierarchy_sync_metadata.down.sql")
	for _, fragment := range []string{
		"DROP TABLE IF EXISTS org_unit_group_members",
		"DROP TABLE IF EXISTS org_unit_attributes",
		"DROP TABLE IF EXISTS org_unit_groups",
		"DROP TABLE IF EXISTS org_unit_levels",
		"DROP COLUMN IF EXISTS district_level_name",
	} {
		if !strings.Contains(hierarchySyncDown, fragment) {
			t.Fatalf("expected hierarchy sync rollback to contain %q", fragment)
		}
	}

	reporterOrphaningUp := readMigration(t, "000034_add_reporter_orphaning.up.sql")
	for _, fragment := range []string{
		"ALTER COLUMN org_unit_id DROP NOT NULL",
		"ON DELETE SET NULL",
		"ADD COLUMN IF NOT EXISTS orphaned_at",
		"ADD COLUMN IF NOT EXISTS last_known_org_unit_uid",
	} {
		if !strings.Contains(reporterOrphaningUp, fragment) {
			t.Fatalf("expected reporter orphaning migration to contain %q", fragment)
		}
	}

	reporterOrphaningDown := readMigration(t, "000034_add_reporter_orphaning.down.sql")
	for _, fragment := range []string{
		"DROP COLUMN IF EXISTS orphaned_at",
		"ON DELETE RESTRICT",
		"ALTER COLUMN org_unit_id SET NOT NULL",
	} {
		if !strings.Contains(reporterOrphaningDown, fragment) {
			t.Fatalf("expected reporter orphaning rollback to contain %q", fragment)
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
