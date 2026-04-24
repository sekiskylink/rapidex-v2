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

func TestEnsureDefaultIntegrationJobsSeedsDefaultIntegrationJobs(t *testing.T) {
	svc := NewService(NewRepository())

	created, err := svc.EnsureDefaultIntegrationJobs(context.Background())
	if err != nil {
		t.Fatalf("ensure default integration jobs: %v", err)
	}
	if len(created) != 2 {
		t.Fatalf("expected two seeded integration jobs, got %+v", created)
	}
	jobTypes := map[string]bool{}
	for _, item := range created {
		jobTypes[item.JobType] = true
	}
	if !jobTypes[JobTypeRapidProReporterSync] || !jobTypes[JobTypeDHIS2OrgUnitRefresh] {
		t.Fatalf("expected rapidpro and dhis2 org unit refresh jobs, got %+v", created)
	}

	secondRun, err := svc.EnsureDefaultIntegrationJobs(context.Background())
	if err != nil {
		t.Fatalf("ensure default integration jobs second run: %v", err)
	}
	if len(secondRun) != 0 {
		t.Fatalf("expected idempotent seeding, got %+v", secondRun)
	}
}
