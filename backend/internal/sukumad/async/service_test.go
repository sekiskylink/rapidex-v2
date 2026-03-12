package async

import (
	"context"
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
	if err := service.PollDueTasks(context.Background(), 10, StaticPoller{
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

func intPtr(value int) *int {
	return &value
}
