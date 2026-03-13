package retention

import (
	"context"
	"errors"
	"testing"
	"time"

	"basepro/backend/internal/audit"
	"basepro/backend/internal/sukumad/traceevent"
)

type fakeRepository struct {
	listCandidatesFn func(context.Context, time.Time, int) ([]Candidate, error)
	purgeFn          func(context.Context, int64) (PurgeCounts, error)
}

func (f *fakeRepository) ListPurgeCandidates(ctx context.Context, cutoff time.Time, limit int) ([]Candidate, error) {
	return f.listCandidatesFn(ctx, cutoff, limit)
}

func (f *fakeRepository) PurgeRequest(ctx context.Context, requestID int64) (PurgeCounts, error) {
	return f.purgeFn(ctx, requestID)
}

type fakeAuditRepo struct {
	events []audit.Event
}

func (f *fakeAuditRepo) Insert(_ context.Context, event audit.Event) error {
	f.events = append(f.events, event)
	return nil
}

func (f *fakeAuditRepo) List(_ context.Context, _ audit.ListFilter) (audit.ListResult, error) {
	return audit.ListResult{}, nil
}

type fakeEventWriter struct {
	events []traceevent.WriteInput
}

func (f *fakeEventWriter) AppendEvent(_ context.Context, input traceevent.WriteInput) error {
	f.events = append(f.events, input)
	return nil
}

func TestServiceRunDryRunReportsCandidatesWithoutDeleting(t *testing.T) {
	now := time.Now().UTC()
	repo := &fakeRepository{
		listCandidatesFn: func(_ context.Context, cutoff time.Time, limit int) ([]Candidate, error) {
			if limit != 2 {
				t.Fatalf("expected batch size 2, got %d", limit)
			}
			if cutoff.IsZero() {
				t.Fatal("expected cutoff")
			}
			return []Candidate{
				{RequestID: 10, RequestUID: "req-10", Status: "completed", UpdatedAt: now},
				{RequestID: 11, RequestUID: "req-11", Status: "failed", UpdatedAt: now},
			}, nil
		},
		purgeFn: func(_ context.Context, _ int64) (PurgeCounts, error) {
			t.Fatal("purge should not run during dry-run")
			return PurgeCounts{}, nil
		},
	}
	auditRepo := &fakeAuditRepo{}
	eventWriter := &fakeEventWriter{}
	service := NewService(repo, audit.NewService(auditRepo)).WithEventWriter(eventWriter)

	result, err := service.Run(context.Background(), RunInput{
		Cutoff:    now.Add(-24 * time.Hour),
		BatchSize: 2,
		DryRun:    true,
	})
	if err != nil {
		t.Fatalf("run retention: %v", err)
	}
	if result.Scanned != 2 || len(result.CandidateRequests) != 2 {
		t.Fatalf("unexpected dry-run result: %+v", result)
	}
	if result.Counts.Requests != 2 {
		t.Fatalf("expected request count to reflect candidates in dry-run, got %+v", result.Counts)
	}
	if len(auditRepo.events) != 1 || auditRepo.events[0].Action != "retention.run" {
		t.Fatalf("unexpected audit events: %+v", auditRepo.events)
	}
	if len(eventWriter.events) < 2 {
		t.Fatalf("expected retention events, got %+v", eventWriter.events)
	}
}

func TestServiceRunPurgesCandidatesAndAggregatesCounts(t *testing.T) {
	now := time.Now().UTC()
	repo := &fakeRepository{
		listCandidatesFn: func(_ context.Context, _ time.Time, _ int) ([]Candidate, error) {
			return []Candidate{
				{RequestID: 20, RequestUID: "req-20", Status: "completed", UpdatedAt: now},
				{RequestID: 21, RequestUID: "req-21", Status: "failed", UpdatedAt: now},
			}, nil
		},
		purgeFn: func(_ context.Context, requestID int64) (PurgeCounts, error) {
			return PurgeCounts{Requests: 1, DeliveryAttempts: int(requestID - 18)}, nil
		},
	}
	service := NewService(repo)

	result, err := service.Run(context.Background(), RunInput{
		Cutoff:    now,
		BatchSize: 10,
		DryRun:    false,
	})
	if err != nil {
		t.Fatalf("run retention: %v", err)
	}
	if result.Counts.Requests != 2 || result.Counts.DeliveryAttempts != 5 {
		t.Fatalf("unexpected aggregate counts: %+v", result.Counts)
	}
}

func TestServiceRunStopsOnCancellation(t *testing.T) {
	now := time.Now().UTC()
	ctx, cancel := context.WithCancel(context.Background())
	repo := &fakeRepository{
		listCandidatesFn: func(_ context.Context, _ time.Time, _ int) ([]Candidate, error) {
			return []Candidate{
				{RequestID: 30, RequestUID: "req-30", Status: "completed", UpdatedAt: now},
			}, nil
		},
		purgeFn: func(_ context.Context, _ int64) (PurgeCounts, error) {
			cancel()
			return PurgeCounts{}, context.Canceled
		},
	}
	service := NewService(repo)

	if _, err := service.Run(ctx, RunInput{Cutoff: now, BatchSize: 1}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}
