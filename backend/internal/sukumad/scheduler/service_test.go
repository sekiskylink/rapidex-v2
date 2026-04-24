package scheduler

import (
	"bytes"
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"basepro/backend/internal/logging"
)

func TestCalculateNextRunInterval(t *testing.T) {
	svc := NewService(NewRepository()).WithClock(func() time.Time {
		return time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	})

	next, err := svc.CalculateNextRun(ScheduleTypeInterval, "15m", "UTC", svc.clock())
	if err != nil {
		t.Fatalf("calculate interval next run: %v", err)
	}
	if next == nil || !next.Equal(time.Date(2026, 4, 18, 12, 15, 0, 0, time.UTC)) {
		t.Fatalf("unexpected next run: %v", next)
	}
}

func TestCalculateNextRunCron(t *testing.T) {
	svc := NewService(NewRepository())
	reference := time.Date(2026, 4, 18, 12, 7, 0, 0, time.UTC)

	next, err := svc.CalculateNextRun(ScheduleTypeCron, "30 12 * * *", "UTC", reference)
	if err != nil {
		t.Fatalf("calculate cron next run: %v", err)
	}
	if next == nil || !next.Equal(time.Date(2026, 4, 18, 12, 30, 0, 0, time.UTC)) {
		t.Fatalf("unexpected cron next run: %v", next)
	}
}

func TestCreateScheduledJobRejectsInvalidTimezone(t *testing.T) {
	svc := NewService(NewRepository())

	_, err := svc.CreateScheduledJob(context.Background(), CreateInput{
		Code:         "nightly-sync",
		Name:         "Nightly Sync",
		JobCategory:  JobCategoryIntegration,
		JobType:      "dhis2.sync",
		ScheduleType: ScheduleTypeInterval,
		ScheduleExpr: "30m",
		Timezone:     "Mars/Phobos",
		Enabled:      true,
		Config:       map[string]any{"serverCode": "dhis2"},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestCreateScheduledJobRejectsInvalidMaintenanceConfig(t *testing.T) {
	svc := NewService(NewRepository())

	_, err := svc.CreateScheduledJob(context.Background(), CreateInput{
		Code:         "purge-logs",
		Name:         "Purge Logs",
		JobCategory:  JobCategoryMaintenance,
		JobType:      "purge_old_logs",
		ScheduleType: ScheduleTypeCron,
		ScheduleExpr: "0 2 * * *",
		Timezone:     "UTC",
		Enabled:      true,
		Config: map[string]any{
			"dryRun":     false,
			"batchSize":  0,
			"maxAgeDays": -1,
		},
	})
	if err == nil {
		t.Fatal("expected validation error for maintenance config")
	}
}

func TestRunNowCreatesPendingManualRun(t *testing.T) {
	now := time.Date(2026, 4, 18, 13, 0, 0, 0, time.UTC)
	svc := NewService(NewRepository()).WithClock(func() time.Time { return now })

	job, err := svc.CreateScheduledJob(context.Background(), CreateInput{
		Code:                "nightly-sync",
		Name:                "Nightly Sync",
		JobCategory:         JobCategoryIntegration,
		JobType:             "dhis2.sync",
		ScheduleType:        ScheduleTypeInterval,
		ScheduleExpr:        "30m",
		Timezone:            "UTC",
		Enabled:             true,
		AllowConcurrentRuns: false,
		Config:              map[string]any{"serverCode": "dhis2"},
	})
	if err != nil {
		t.Fatalf("create scheduled job: %v", err)
	}

	run, err := svc.RunNow(context.Background(), nil, job.ID)
	if err != nil {
		t.Fatalf("run now: %v", err)
	}
	if run.TriggerMode != TriggerModeManual {
		t.Fatalf("expected manual trigger, got %s", run.TriggerMode)
	}
	if run.Status != RunStatusPending {
		t.Fatalf("expected pending status, got %s", run.Status)
	}
	if !run.ScheduledFor.Equal(now) {
		t.Fatalf("expected scheduled_for %s, got %s", now, run.ScheduledFor)
	}
}

func TestDispatchDueJobsQueuesScheduledRunAndAdvancesNextRun(t *testing.T) {
	now := time.Date(2026, 4, 18, 13, 0, 0, 0, time.UTC)
	svc := NewService(NewRepository()).WithClock(func() time.Time { return now })

	job, err := svc.CreateScheduledJob(context.Background(), CreateInput{
		Code:         "metadata-sync",
		Name:         "Metadata Sync",
		JobCategory:  JobCategoryIntegration,
		JobType:      "metadata_sync",
		ScheduleType: ScheduleTypeInterval,
		ScheduleExpr: "15m",
		Timezone:     "UTC",
		Enabled:      true,
	})
	if err != nil {
		t.Fatalf("create scheduled job: %v", err)
	}
	pastDue := now.Add(-15 * time.Minute)
	repo := svc.repo.(*memoryRepository)
	repo.jobs[0].NextRunAt = &pastDue

	result, err := svc.DispatchDueJobs(context.Background(), 10)
	if err != nil {
		t.Fatalf("dispatch due jobs: %v", err)
	}
	if len(result.CreatedRuns) != 1 {
		t.Fatalf("expected one created run, got %+v", result)
	}
	if result.CreatedRuns[0].TriggerMode != TriggerModeScheduled {
		t.Fatalf("expected scheduled trigger mode, got %+v", result.CreatedRuns[0])
	}
	updatedJob, err := svc.GetScheduledJob(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("reload scheduled job: %v", err)
	}
	if updatedJob.NextRunAt == nil || !updatedJob.NextRunAt.After(now) {
		t.Fatalf("expected next run after %s, got %v", now, updatedJob.NextRunAt)
	}
}

func TestDispatchDueJobsPreventsDuplicateDispatch(t *testing.T) {
	now := time.Date(2026, 4, 18, 13, 0, 0, 0, time.UTC)
	svc := NewService(NewRepository()).WithClock(func() time.Time { return now })

	job, err := svc.CreateScheduledJob(context.Background(), CreateInput{
		Code:         "nightly-reconcile",
		Name:         "Nightly Reconcile",
		JobCategory:  JobCategoryIntegration,
		JobType:      "reconciliation_pull",
		ScheduleType: ScheduleTypeInterval,
		ScheduleExpr: "30m",
		Timezone:     "UTC",
		Enabled:      true,
	})
	if err != nil {
		t.Fatalf("create scheduled job: %v", err)
	}
	pastDue := now.Add(-30 * time.Minute)
	repo := svc.repo.(*memoryRepository)
	repo.jobs[0].NextRunAt = &pastDue

	first, err := svc.DispatchDueJobs(context.Background(), 10)
	if err != nil {
		t.Fatalf("first dispatch: %v", err)
	}
	second, err := svc.DispatchDueJobs(context.Background(), 10)
	if err != nil {
		t.Fatalf("second dispatch: %v", err)
	}
	if len(first.CreatedRuns) != 1 || len(second.CreatedRuns) != 0 {
		t.Fatalf("expected only one dispatch across cycles, first=%+v second=%+v", first, second)
	}

	runs, err := svc.ListJobRuns(context.Background(), job.ID, RunListQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runs.Items) != 1 {
		t.Fatalf("expected one persisted run, got %+v", runs.Items)
	}
}

func TestClaimNextPendingRunRespectsAllowConcurrentRuns(t *testing.T) {
	repo := NewRepository().(*memoryRepository)
	now := time.Date(2026, 4, 18, 13, 0, 0, 0, time.UTC)
	job, err := repo.CreateScheduledJob(context.Background(), CreateParams{
		UID:                 newUID(),
		Code:                "archive-old",
		Name:                "Archive Old",
		JobCategory:         JobCategoryMaintenance,
		JobType:             "archive_old_requests",
		ScheduleType:        ScheduleTypeInterval,
		ScheduleExpr:        "1h",
		Timezone:            "UTC",
		Enabled:             true,
		AllowConcurrentRuns: false,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	for index := 0; index < 2; index++ {
		if _, err := repo.CreateJobRun(context.Background(), CreateRunParams{
			UID:            newUID(),
			ScheduledJobID: job.ID,
			TriggerMode:    TriggerModeManual,
			ScheduledFor:   now.Add(time.Duration(index) * time.Minute),
			Status:         RunStatusPending,
		}); err != nil {
			t.Fatalf("create run %d: %v", index, err)
		}
	}

	first, err := repo.ClaimNextPendingRun(context.Background(), now, 77)
	if err != nil {
		t.Fatalf("claim first run: %v", err)
	}
	if first.Status != RunStatusRunning {
		t.Fatalf("expected running status after first claim, got %+v", first)
	}
	if _, err := repo.ClaimNextPendingRun(context.Background(), now.Add(time.Minute), 77); err == nil || err != sql.ErrNoRows {
		t.Fatalf("expected no second claim while first run is active, got %v", err)
	}
}

func TestRunPendingSchedulerRunsTransitionsLifecycle(t *testing.T) {
	now := time.Date(2026, 4, 18, 13, 0, 0, 0, time.UTC)
	svc := NewService(NewRepository()).WithClock(func() time.Time { return now })

	job, err := svc.CreateScheduledJob(context.Background(), CreateInput{
		Code:         "metadata-sync",
		Name:         "Metadata Sync",
		JobCategory:  JobCategoryIntegration,
		JobType:      "metadata_sync",
		ScheduleType: ScheduleTypeInterval,
		ScheduleExpr: "15m",
		Timezone:     "UTC",
		Enabled:      true,
		Config: map[string]any{
			"serverCode": "dhis2",
			"batchSize":  25,
			"dryRun":     true,
		},
	})
	if err != nil {
		t.Fatalf("create scheduled job: %v", err)
	}
	run, err := svc.RunNow(context.Background(), nil, job.ID)
	if err != nil {
		t.Fatalf("run now: %v", err)
	}

	if err := svc.RunPendingSchedulerRuns(context.Background(), 42, 1); err != nil {
		t.Fatalf("run pending scheduler runs: %v", err)
	}
	finishedRun, err := svc.GetRun(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if finishedRun.Status != RunStatusSucceeded || finishedRun.StartedAt == nil || finishedRun.FinishedAt == nil {
		t.Fatalf("expected completed scheduler run lifecycle, got %+v", finishedRun)
	}
	if finishedRun.WorkerID == nil || *finishedRun.WorkerID != 42 {
		t.Fatalf("expected worker id to be attached, got %+v", finishedRun)
	}
	if finishedRun.ResultSummary["jobType"] != "metadata_sync" {
		t.Fatalf("expected structured result summary, got %+v", finishedRun.ResultSummary)
	}
	updatedJob, err := svc.GetScheduledJob(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("reload job: %v", err)
	}
	if updatedJob.LastRunAt == nil || updatedJob.LastSuccessAt == nil || updatedJob.LatestRunStatus != RunStatusSucceeded {
		t.Fatalf("expected scheduler job lifecycle timestamps, got %+v", updatedJob)
	}
}

func TestRunPendingSchedulerRunsFailsUnknownJobType(t *testing.T) {
	now := time.Date(2026, 4, 18, 13, 0, 0, 0, time.UTC)
	svc := NewService(NewRepository()).WithClock(func() time.Time { return now })

	job, err := svc.CreateScheduledJob(context.Background(), CreateInput{
		Code:         "mystery-job",
		Name:         "Mystery Job",
		JobCategory:  JobCategoryMaintenance,
		JobType:      "mystery_type",
		ScheduleType: ScheduleTypeInterval,
		ScheduleExpr: "15m",
		Timezone:     "UTC",
		Enabled:      true,
	})
	if err != nil {
		t.Fatalf("create scheduled job: %v", err)
	}
	run, err := svc.RunNow(context.Background(), nil, job.ID)
	if err != nil {
		t.Fatalf("run now: %v", err)
	}

	if err := svc.RunPendingSchedulerRuns(context.Background(), 99, 1); err != nil {
		t.Fatalf("run pending scheduler runs: %v", err)
	}
	finishedRun, err := svc.GetRun(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if finishedRun.Status != RunStatusFailed {
		t.Fatalf("expected failed run for unknown job type, got %+v", finishedRun)
	}
	if finishedRun.ErrorMessage == "" {
		t.Fatalf("expected error message for unknown job type, got %+v", finishedRun)
	}
}

func TestDispatchDueJobsAndRunPendingSchedulerRunsEmitLifecycleLogs(t *testing.T) {
	now := time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)
	logOutput := captureSchedulerLogs(t)
	svc := NewService(NewRepository()).WithClock(func() time.Time { return now })

	_, err := svc.CreateScheduledJob(context.Background(), CreateInput{
		Code:         "metadata-sync",
		Name:         "Metadata Sync",
		JobCategory:  JobCategoryIntegration,
		JobType:      "metadata_sync",
		ScheduleType: ScheduleTypeInterval,
		ScheduleExpr: "15m",
		Timezone:     "UTC",
		Enabled:      true,
		Config: map[string]any{
			"serverCode": "dhis2",
			"batchSize":  25,
			"dryRun":     true,
		},
	})
	if err != nil {
		t.Fatalf("create scheduled job: %v", err)
	}
	now = now.Add(16 * time.Minute)
	if _, err := svc.DispatchDueJobs(context.Background(), 1); err != nil {
		t.Fatalf("dispatch due jobs: %v", err)
	}
	if err := svc.RunPendingSchedulerRuns(context.Background(), 42, 1); err != nil {
		t.Fatalf("run pending scheduler runs: %v", err)
	}

	assertSchedulerLogContains(t, logOutput.String(),
		"scheduler_run_queued",
		"\"job_code\":\"metadata-sync\"",
		"scheduler_run_claimed",
		"\"worker_id\":42",
		"scheduler_run_started",
		"scheduler_run_finished",
		"\"status\":\"succeeded\"",
	)
}

func captureSchedulerLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var logOutput bytes.Buffer
	logging.SetOutput(&logOutput)
	logging.ApplyConfig(logging.Config{Level: "info", Format: "json"})
	t.Cleanup(func() {
		logging.SetOutput(nil)
		logging.ApplyConfig(logging.Config{Level: "info", Format: "console"})
	})
	return &logOutput
}

func assertSchedulerLogContains(t *testing.T, logs string, fragments ...string) {
	t.Helper()
	for _, fragment := range fragments {
		if !strings.Contains(logs, fragment) {
			t.Fatalf("expected logs to contain %q, got:\n%s", fragment, logs)
		}
	}
}
