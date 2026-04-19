package scheduler

import (
	"context"
	"testing"
)

func TestEnsureDefaultMaintenanceJobsSeedsMissingJobs(t *testing.T) {
	svc := NewService(NewRepository())

	created, err := svc.EnsureDefaultMaintenanceJobs(context.Background())
	if err != nil {
		t.Fatalf("ensure default maintenance jobs: %v", err)
	}
	if len(created) != 3 {
		t.Fatalf("expected three seeded jobs, got %+v", created)
	}

	secondRun, err := svc.EnsureDefaultMaintenanceJobs(context.Background())
	if err != nil {
		t.Fatalf("ensure default maintenance jobs second run: %v", err)
	}
	if len(secondRun) != 0 {
		t.Fatalf("expected idempotent seeding, got %+v", secondRun)
	}
}
