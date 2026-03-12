package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"basepro/backend/internal/audit"
)

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

func TestServiceStartRunAndHeartbeat(t *testing.T) {
	auditRepo := &fakeAuditRepo{}
	service := NewService(NewRepository(), audit.NewService(auditRepo))

	run, err := service.StartRun(context.Background(), Definition{Type: TypePoll, Name: "poll-worker"})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}
	if run.Status != StatusRunning {
		t.Fatalf("expected running status, got %+v", run)
	}
	heartbeat, err := service.Heartbeat(context.Background(), run.ID, map[string]any{"tick": 1})
	if err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	if heartbeat.LastHeartbeatAt == nil || heartbeat.Meta["tick"] != 1 {
		t.Fatalf("unexpected heartbeat payload: %+v", heartbeat)
	}
	if len(auditRepo.events) != 1 || auditRepo.events[0].Action != "worker.started" {
		t.Fatalf("unexpected audit events: %+v", auditRepo.events)
	}
}

func TestManagerHandlesContextCancellationAndFailures(t *testing.T) {
	service := NewService(NewRepository())

	ctx, cancel := context.WithCancel(context.Background())
	manager := NewManager(service, Definition{
		Type:              TypeRetry,
		Name:              "retry-worker",
		Interval:          10 * time.Millisecond,
		HeartbeatInterval: 5 * time.Millisecond,
		Run: func(context.Context, Execution) error {
			cancel()
			return nil
		},
	})

	var errs []error
	for err := range manager.Start(ctx) {
		errs = append(errs, err)
	}

	if len(errs) > 0 && !errors.Is(errs[0], context.Canceled) {
		t.Fatalf("expected context cancellation, got %+v", errs)
	}

	list, err := service.ListRuns(context.Background(), ListQuery{Page: 1, PageSize: 25})
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if list.Total != 1 || list.Items[0].Status != StatusStopped {
		t.Fatalf("unexpected worker list after shutdown: %+v", list)
	}
}
