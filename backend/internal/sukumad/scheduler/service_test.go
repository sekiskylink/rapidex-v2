package scheduler

import (
	"context"
	"testing"
	"time"
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
