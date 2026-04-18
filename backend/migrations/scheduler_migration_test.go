package migrations

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSchedulerMigrationDeclaresRequiredTablesAndColumns(t *testing.T) {
	path := filepath.Join("000025_create_scheduler_tables.up.sql")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	sql := string(content)
	requiredFragments := []string{
		"CREATE TABLE scheduled_jobs",
		"job_category TEXT NOT NULL CHECK (job_category IN ('integration', 'maintenance'))",
		"schedule_type TEXT NOT NULL CHECK (schedule_type IN ('cron', 'interval'))",
		"allow_concurrent_runs BOOLEAN NOT NULL DEFAULT FALSE",
		"config JSONB NOT NULL DEFAULT '{}'::jsonb",
		"CREATE TABLE scheduled_job_runs",
		"scheduled_job_id BIGINT NOT NULL REFERENCES scheduled_jobs(id) ON DELETE CASCADE",
		"trigger_mode TEXT NOT NULL CHECK (trigger_mode IN ('scheduled', 'manual'))",
		"status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'succeeded', 'failed', 'cancelled', 'skipped'))",
		"result_summary JSONB NOT NULL DEFAULT '{}'::jsonb",
	}

	for _, fragment := range requiredFragments {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("expected migration to contain %q", fragment)
		}
	}
}
