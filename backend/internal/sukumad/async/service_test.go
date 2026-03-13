package async

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

func TestServiceCreateUpdateAndRecordPoll(t *testing.T) {
	auditRepo := &fakeAuditRepo{}
	service := NewService(NewRepository(), audit.NewService(auditRepo))

	nextPollAt := time.Now().UTC().Add(time.Minute)
	created, err := service.CreateTask(context.Background(), CreateInput{
		DeliveryAttemptID: 11,
		RemoteJobID:       "remote-11",
		PollURL:           "https://remote/jobs/11",
		RemoteStatus:      StatePending,
		NextPollAt:        &nextPollAt,
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if created.CurrentState != StatePending {
		t.Fatalf("expected pending state, got %+v", created)
	}

	statusCode := 200
	duration := 321
	poll, err := service.RecordPoll(context.Background(), RecordPollInput{
		AsyncTaskID:  created.ID,
		StatusCode:   &statusCode,
		RemoteStatus: StatePolling,
		ResponseBody: `{"state":"running"}`,
		DurationMS:   &duration,
	})
	if err != nil {
		t.Fatalf("record poll: %v", err)
	}
	if poll.AsyncTaskID != created.ID || poll.StatusCode == nil || *poll.StatusCode != 200 {
		t.Fatalf("unexpected poll record: %+v", poll)
	}

	updated, err := service.UpdateTaskStatus(context.Background(), UpdateStatusInput{
		ID:            created.ID,
		RemoteStatus:  StateSucceeded,
		TerminalState: StateSucceeded,
		RemoteResponse: map[string]any{
			"summary": "ok",
		},
	})
	if err != nil {
		t.Fatalf("update status: %v", err)
	}
	if updated.TerminalState != StateSucceeded || updated.CompletedAt == nil {
		t.Fatalf("expected succeeded task, got %+v", updated)
	}
	if len(auditRepo.events) != 2 {
		t.Fatalf("expected create and completion audit events, got %+v", auditRepo.events)
	}
}

func TestServicePollDueTasksUpdatesTaskAndHistory(t *testing.T) {
	service := NewService(NewRepository())
	nextPollAt := time.Now().UTC().Add(-time.Minute)
	created, err := service.CreateTask(context.Background(), CreateInput{
		DeliveryAttemptID: 7,
		RemoteJobID:       "remote-7",
		RemoteStatus:      StatePolling,
		NextPollAt:        &nextPollAt,
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	resultNextPoll := time.Now().UTC().Add(2 * time.Minute)
	if err := service.PollDueTasks(context.Background(), PollExecution{}, 10, StaticPoller{
		Result: RemotePollResult{
			StatusCode:   intPtr(202),
			RemoteStatus: StatePolling,
			ResponseBody: `{"state":"processing"}`,
			DurationMS:   intPtr(123),
			NextPollAt:   &resultNextPoll,
			RemoteResponse: map[string]any{
				"state": "processing",
			},
		},
	}); err != nil {
		t.Fatalf("poll due tasks: %v", err)
	}

	task, err := service.GetTask(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if task.CurrentState != StatePolling || task.NextPollAt == nil {
		t.Fatalf("expected polling task with next poll, got %+v", task)
	}

	polls, err := service.ListPolls(context.Background(), created.ID, ListQuery{Page: 1, PageSize: 25})
	if err != nil {
		t.Fatalf("list polls: %v", err)
	}
	if polls.Total != 1 || len(polls.Items) != 1 {
		t.Fatalf("expected one poll history row, got %+v", polls)
	}
}

type fakeDeliveryUpdater struct {
	succeeded []int64
	failed    []int64
}

func (f *fakeDeliveryUpdater) CompleteFromAsyncSuccess(_ context.Context, deliveryID int64, _ string) error {
	f.succeeded = append(f.succeeded, deliveryID)
	return nil
}

func (f *fakeDeliveryUpdater) CompleteFromAsyncFailure(_ context.Context, deliveryID int64, _ string, _ string) error {
	f.failed = append(f.failed, deliveryID)
	return nil
}

type fakeRequestUpdater struct {
	processing []int64
	completed  []int64
	failed     []int64
}

func (f *fakeRequestUpdater) SetProcessing(_ context.Context, requestID int64) error {
	f.processing = append(f.processing, requestID)
	return nil
}

func (f *fakeRequestUpdater) SetCompleted(_ context.Context, requestID int64) error {
	f.completed = append(f.completed, requestID)
	return nil
}

func (f *fakeRequestUpdater) SetFailed(_ context.Context, requestID int64) error {
	f.failed = append(f.failed, requestID)
	return nil
}

type errorPoller struct{}

func (errorPoller) Poll(context.Context, Record) (RemotePollResult, error) {
	return RemotePollResult{}, errors.New("network timeout")
}

func TestServiceUpdateTaskStatusReconcilesTerminalSuccess(t *testing.T) {
	deliveryUpdater := &fakeDeliveryUpdater{}
	requestUpdater := &fakeRequestUpdater{}
	service := NewService(NewRepository()).WithReconciliation(deliveryUpdater, requestUpdater)

	created, err := service.CreateTask(context.Background(), CreateInput{
		DeliveryAttemptID: 4,
		RemoteJobID:       "remote-4",
		RemoteStatus:      StatePending,
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	if _, err := service.UpdateTaskStatus(context.Background(), UpdateStatusInput{
		ID:            created.ID,
		RemoteStatus:  StateSucceeded,
		TerminalState: StateSucceeded,
		RemoteResponse: map[string]any{
			"status": "OK",
		},
	}); err != nil {
		t.Fatalf("update status: %v", err)
	}

	if len(deliveryUpdater.succeeded) != 1 || deliveryUpdater.succeeded[0] != created.DeliveryAttemptID {
		t.Fatalf("expected delivery success reconciliation, got %+v", deliveryUpdater)
	}
	if len(requestUpdater.completed) != 1 || requestUpdater.completed[0] != created.RequestID {
		t.Fatalf("expected request completion reconciliation, got %+v", requestUpdater)
	}
}

func TestServicePollDueTasksKeepsTaskPollingOnTransientError(t *testing.T) {
	requestUpdater := &fakeRequestUpdater{}
	service := NewService(NewRepository()).WithReconciliation(&fakeDeliveryUpdater{}, requestUpdater)
	nextPollAt := time.Now().UTC().Add(-time.Minute)
	created, err := service.CreateTask(context.Background(), CreateInput{
		DeliveryAttemptID: 5,
		RemoteJobID:       "remote-5",
		RemoteStatus:      StatePolling,
		NextPollAt:        &nextPollAt,
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	if err := service.PollDueTasks(context.Background(), PollExecution{}, 10, errorPoller{}); err != nil {
		t.Fatalf("poll due tasks: %v", err)
	}

	task, err := service.GetTask(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if task.CurrentState != StatePolling || task.TerminalState != "" || task.NextPollAt == nil {
		t.Fatalf("expected task to remain polling after transient error, got %+v", task)
	}
	if len(requestUpdater.processing) == 0 {
		t.Fatalf("expected request processing update after transient error")
	}
}

func TestServicePollDueTasksClaimsTasksAndClearsClaimState(t *testing.T) {
	service := NewService(NewRepository())
	due := time.Now().UTC().Add(-time.Minute)
	created, err := service.CreateTask(context.Background(), CreateInput{
		DeliveryAttemptID: 6,
		RemoteJobID:       "remote-6",
		RemoteStatus:      StatePolling,
		NextPollAt:        &due,
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	counts := map[string]int{}
	if err := service.PollDueTasks(context.Background(), PollExecution{
		WorkerRunID:  17,
		ClaimTimeout: time.Minute,
		Observe: func(name string, delta int) {
			counts[name] += delta
		},
	}, 1, StaticPoller{
		Result: RemotePollResult{
			StatusCode:   intPtr(202),
			RemoteStatus: StatePolling,
			NextPollAt:   nextRetryPollAt(),
			RemoteResponse: map[string]any{
				"state": "polling",
			},
		},
	}); err != nil {
		t.Fatalf("poll due tasks: %v", err)
	}

	task, err := service.GetTask(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if task.PollClaimedAt != nil || task.PollClaimedByRunID != nil {
		t.Fatalf("expected claim to be cleared after polling, got %+v", task)
	}
	if counts["polls_picked"] != 1 || counts["polls_completed"] != 1 {
		t.Fatalf("expected poll counters to increment, got %+v", counts)
	}
}

func TestServiceReconcileTerminalTasksRecoversMissedReconciliation(t *testing.T) {
	repo := NewRepository()
	deliveryUpdater := &fakeDeliveryUpdater{}
	requestUpdater := &fakeRequestUpdater{}
	service := NewService(repo).WithReconciliation(deliveryUpdater, requestUpdater)

	created, err := service.CreateTask(context.Background(), CreateInput{
		DeliveryAttemptID: 10,
		RemoteJobID:       "remote-10",
		RemoteStatus:      StatePolling,
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	if _, err := repo.UpdateTask(context.Background(), UpdateParams{
		ID:            created.ID,
		RemoteJobID:   created.RemoteJobID,
		RemoteStatus:  StateSucceeded,
		TerminalState: StateSucceeded,
		CompletedAt:   func() *time.Time { now := time.Now().UTC(); return &now }(),
		RemoteResponse: map[string]any{
			"status": "OK",
		},
	}); err != nil {
		t.Fatalf("seed terminal task state: %v", err)
	}

	if err := service.ReconcileTerminalTasks(context.Background(), 10); err != nil {
		t.Fatalf("reconcile terminal tasks: %v", err)
	}
	if len(deliveryUpdater.succeeded) != 1 || deliveryUpdater.succeeded[0] != created.DeliveryAttemptID {
		t.Fatalf("expected recovered delivery reconciliation, got %+v", deliveryUpdater)
	}
	if len(requestUpdater.completed) != 1 || requestUpdater.completed[0] != created.RequestID {
		t.Fatalf("expected recovered request completion, got %+v", requestUpdater)
	}
}

func intPtr(value int) *int {
	return &value
}
