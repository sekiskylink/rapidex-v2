package scheduler

import (
	"context"
	"testing"
	"time"
)

func TestArchiveOldRequestsHandler(t *testing.T) {
	repo := &fakeMaintenanceRepository{
		archivableRequests: []maintenanceRequestCandidate{
			{RequestID: 11, RequestUID: "req-11"},
			{RequestID: 12, RequestUID: "req-12"},
		},
	}
	registry := NewDefaultHandlerRegistry()
	for jobType, registration := range newMaintenanceHandlers(maintenanceHandlerDependencies{
		repo: repo,
		now:  func() time.Time { return time.Date(2026, 4, 19, 9, 0, 0, 0, time.UTC) },
	}) {
		registry.Register(jobType, registration.handler)
	}

	handler := registry.Lookup("archive_old_requests")
	result, err := handler.Execute(context.Background(), JobExecution{
		Job: Record{
			JobType: "archive_old_requests",
			Config: map[string]any{
				"dryRun":     false,
				"batchSize":  50,
				"maxAgeDays": 30,
			},
		},
		Run: RunRecord{UID: "run-archive"},
	})
	if err != nil {
		t.Fatalf("execute archive handler: %v", err)
	}
	if result.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %+v", result)
	}
	if got := result.ResultSummary["archived_count"]; got != 2 {
		t.Fatalf("expected archived_count=2, got %+v", got)
	}
	if len(repo.archivedIDs) != 2 {
		t.Fatalf("expected archived ids to be recorded, got %+v", repo.archivedIDs)
	}
}

func TestPurgeOldLogsHandler(t *testing.T) {
	repo := &fakeMaintenanceRepository{
		purgeResult: logPurgeResult{
			AuditLogs:      4,
			RequestEvents:  3,
			AsyncTaskPolls: 2,
			WorkerRuns:     1,
		},
	}
	registry := NewDefaultHandlerRegistry()
	for jobType, registration := range newMaintenanceHandlers(maintenanceHandlerDependencies{
		repo: repo,
		now:  func() time.Time { return time.Date(2026, 4, 19, 9, 0, 0, 0, time.UTC) },
	}) {
		registry.Register(jobType, registration.handler)
	}

	result, err := registry.Lookup("purge_old_logs").Execute(context.Background(), JobExecution{
		Job: Record{
			JobType: "purge_old_logs",
			Config: map[string]any{
				"dryRun":     false,
				"batchSize":  25,
				"maxAgeDays": 14,
			},
		},
		Run: RunRecord{UID: "run-purge"},
	})
	if err != nil {
		t.Fatalf("execute purge handler: %v", err)
	}
	if got := result.ResultSummary["deleted_count"]; got != 10 {
		t.Fatalf("expected deleted_count=10, got %+v", got)
	}
}

func TestMarkStuckRequestsHandlerDryRun(t *testing.T) {
	repo := &fakeMaintenanceRepository{
		stuckRequests: []maintenanceRequestCandidate{
			{RequestID: 21, RequestUID: "req-21"},
			{RequestID: 22, RequestUID: "req-22"},
		},
	}
	registry := NewDefaultHandlerRegistry()
	for jobType, registration := range newMaintenanceHandlers(maintenanceHandlerDependencies{
		repo: repo,
		now:  func() time.Time { return time.Date(2026, 4, 19, 9, 0, 0, 0, time.UTC) },
	}) {
		registry.Register(jobType, registration.handler)
	}

	result, err := registry.Lookup("mark_stuck_requests").Execute(context.Background(), JobExecution{
		Job: Record{
			JobType: "mark_stuck_requests",
			Config: map[string]any{
				"dryRun":             true,
				"batchSize":          20,
				"staleCutoffMinutes": 45,
			},
		},
		Run: RunRecord{UID: "run-stuck"},
	})
	if err != nil {
		t.Fatalf("execute mark stuck handler: %v", err)
	}
	if got := result.ResultSummary["affected_count"]; got != 2 {
		t.Fatalf("expected affected_count=2, got %+v", got)
	}
	if len(repo.stuckIDs) != 0 {
		t.Fatalf("expected dry run to avoid updates, got %+v", repo.stuckIDs)
	}
}

func TestCleanupOrphanedRecordsHandler(t *testing.T) {
	repo := &fakeMaintenanceRepository{
		orphanedIngest: []ingestFileCandidate{
			{ID: 31, UID: "ing-31"},
		},
	}
	registry := NewDefaultHandlerRegistry()
	for jobType, registration := range newMaintenanceHandlers(maintenanceHandlerDependencies{
		repo: repo,
		now:  func() time.Time { return time.Date(2026, 4, 19, 9, 0, 0, 0, time.UTC) },
	}) {
		registry.Register(jobType, registration.handler)
	}

	result, err := registry.Lookup("cleanup_orphaned_records").Execute(context.Background(), JobExecution{
		Job: Record{
			JobType: "cleanup_orphaned_records",
			Config: map[string]any{
				"dryRun":     false,
				"batchSize":  10,
				"maxAgeDays": 7,
			},
		},
		Run: RunRecord{UID: "run-cleanup"},
	})
	if err != nil {
		t.Fatalf("execute cleanup handler: %v", err)
	}
	if got := result.ResultSummary["deleted_count"]; got != 1 {
		t.Fatalf("expected deleted_count=1, got %+v", got)
	}
	if len(repo.deletedIDs) != 1 || repo.deletedIDs[0] != 31 {
		t.Fatalf("expected deleted ingest file id 31, got %+v", repo.deletedIDs)
	}
}
