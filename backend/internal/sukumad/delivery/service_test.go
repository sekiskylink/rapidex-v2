package delivery

import (
	"context"
	"testing"
	"time"

	"basepro/backend/internal/audit"
	asyncjobs "basepro/backend/internal/sukumad/async"
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

func TestServiceCreatePendingDeliveryWritesAuditEvent(t *testing.T) {
	auditRepo := &fakeAuditRepo{}
	service := NewService(NewRepository(), audit.NewService(auditRepo))

	actorID := int64(4)
	created, err := service.CreatePendingDelivery(context.Background(), CreateInput{
		RequestID: 8,
		ServerID:  12,
		ActorID:   &actorID,
	})
	if err != nil {
		t.Fatalf("create pending delivery: %v", err)
	}
	if created.Status != StatusPending || created.AttemptNumber != 1 {
		t.Fatalf("unexpected created delivery: %+v", created)
	}
	if len(auditRepo.events) != 1 || auditRepo.events[0].Action != "delivery.created" {
		t.Fatalf("expected delivery.created audit event, got %+v", auditRepo.events)
	}
}

func TestServiceStatusTransitionsAndRetryScheduling(t *testing.T) {
	auditRepo := &fakeAuditRepo{}
	service := NewService(NewRepository(), audit.NewService(auditRepo))

	actorID := int64(9)
	created, err := service.CreatePendingDelivery(context.Background(), CreateInput{
		RequestID: 11,
		ServerID:  3,
		ActorID:   &actorID,
	})
	if err != nil {
		t.Fatalf("create pending delivery: %v", err)
	}

	running, err := service.MarkRunning(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("mark running: %v", err)
	}
	if running.Status != StatusRunning || running.StartedAt == nil {
		t.Fatalf("expected running delivery, got %+v", running)
	}

	httpStatus := 504
	failed, err := service.MarkFailed(context.Background(), CompletionInput{
		ID:           created.ID,
		HTTPStatus:   &httpStatus,
		ErrorMessage: "gateway timeout",
		ActorID:      &actorID,
	})
	if err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	if failed.Status != StatusFailed || failed.FinishedAt == nil || failed.ErrorMessage != "gateway timeout" {
		t.Fatalf("expected failed delivery, got %+v", failed)
	}

	retried, err := service.RetryDelivery(context.Background(), &actorID, failed.ID)
	if err != nil {
		t.Fatalf("retry delivery: %v", err)
	}
	if retried.Status != StatusRetrying || retried.AttemptNumber != 2 || retried.RetryAt == nil {
		t.Fatalf("expected retrying delivery, got %+v", retried)
	}
	if retried.RetryAt.Before(time.Now().UTC()) {
		t.Fatalf("expected retry schedule in the future, got %v", retried.RetryAt)
	}

	if len(auditRepo.events) != 3 {
		t.Fatalf("expected 3 audit events, got %d", len(auditRepo.events))
	}
	if auditRepo.events[1].Action != "delivery.failed" || auditRepo.events[2].Action != "delivery.retry" {
		t.Fatalf("unexpected audit sequence: %+v", auditRepo.events)
	}
}

func TestServiceRetryRequiresFailedStatus(t *testing.T) {
	service := NewService(NewRepository(), audit.NewService(&fakeAuditRepo{}))

	created, err := service.CreatePendingDelivery(context.Background(), CreateInput{
		RequestID: 2,
		ServerID:  5,
	})
	if err != nil {
		t.Fatalf("create pending delivery: %v", err)
	}

	if _, err := service.RetryDelivery(context.Background(), nil, created.ID); err == nil {
		t.Fatal("expected retry validation error")
	}
}

type fakeDispatcher struct {
	result DispatchResult
	err    error
	keys   []string
}

func (f *fakeDispatcher) Submit(_ context.Context, input DispatchInput) (DispatchResult, error) {
	f.keys = append(f.keys, input.Server.Code)
	return f.result, f.err
}

type fakeAsyncService struct {
	created []asyncjobs.CreateInput
}

func (f *fakeAsyncService) CreateTask(_ context.Context, input asyncjobs.CreateInput) (asyncjobs.Record, error) {
	f.created = append(f.created, input)
	return asyncjobs.Record{
		ID:                51,
		UID:               "job-51",
		DeliveryAttemptID: input.DeliveryAttemptID,
		RemoteJobID:       input.RemoteJobID,
		PollURL:           input.PollURL,
		RemoteStatus:      input.RemoteStatus,
		CurrentState:      input.RemoteStatus,
	}, nil
}

type fakeRequestStatusUpdater struct {
	processing []int64
	completed  []int64
	failed     []int64
}

func (f *fakeRequestStatusUpdater) SetProcessing(_ context.Context, requestID int64) error {
	f.processing = append(f.processing, requestID)
	return nil
}

func (f *fakeRequestStatusUpdater) SetCompleted(_ context.Context, requestID int64) error {
	f.completed = append(f.completed, requestID)
	return nil
}

func (f *fakeRequestStatusUpdater) SetFailed(_ context.Context, requestID int64) error {
	f.failed = append(f.failed, requestID)
	return nil
}

func TestServiceSubmitDHIS2DeliveryCreatesAsyncTask(t *testing.T) {
	asyncService := &fakeAsyncService{}
	requestUpdater := &fakeRequestStatusUpdater{}
	dispatcher := &fakeDispatcher{
		result: DispatchResult{
			HTTPStatus:     intPtr(202),
			ResponseBody:   `{"status":"PENDING"}`,
			RemoteJobID:    "remote-22",
			PollURL:        "https://dhis.example.com/jobs/22",
			RemoteStatus:   asyncjobs.StatePending,
			RemoteResponse: map[string]any{"status": "PENDING"},
			Async:          true,
		},
	}
	service := NewService(NewRepository()).
		WithDispatcher(dispatcher).
		WithAsyncService(asyncService).
		WithRequestStatusUpdater(requestUpdater)

	created, err := service.CreatePendingDelivery(context.Background(), CreateInput{
		RequestID: 22,
		ServerID:  4,
	})
	if err != nil {
		t.Fatalf("create delivery: %v", err)
	}

	record, err := service.SubmitDHIS2Delivery(context.Background(), DispatchInput{
		DeliveryID:  created.ID,
		RequestID:   created.RequestID,
		RequestUID:  created.RequestUID,
		PayloadBody: `{"trackedEntity":"123"}`,
		URLSuffix:   "/tracker",
		Server: ServerSnapshot{
			ID:         created.ServerID,
			Code:       "dhis2-ug",
			Name:       created.ServerName,
			SystemType: "dhis2",
			BaseURL:    "https://dhis.example.com",
			HTTPMethod: "POST",
			UseAsync:   true,
		},
	})
	if err != nil {
		t.Fatalf("submit dhis2 delivery: %v", err)
	}
	if record.Status != StatusRunning {
		t.Fatalf("expected running delivery awaiting async, got %+v", record)
	}
	if len(asyncService.created) != 1 || asyncService.created[0].RemoteJobID != "remote-22" {
		t.Fatalf("unexpected async task creation: %+v", asyncService.created)
	}
	if len(dispatcher.keys) != 1 || dispatcher.keys[0] != "dhis2-ug" {
		t.Fatalf("expected dispatch call to capture destination key, got %+v", dispatcher.keys)
	}
	if len(requestUpdater.processing) != 1 || requestUpdater.processing[0] != created.RequestID {
		t.Fatalf("expected request processing update, got %+v", requestUpdater)
	}
}

func TestServiceSubmitDHIS2DeliveryMarksSyncFailure(t *testing.T) {
	requestUpdater := &fakeRequestStatusUpdater{}
	dispatcher := &fakeDispatcher{
		result: DispatchResult{
			HTTPStatus:   intPtr(400),
			ResponseBody: `{"status":"ERROR"}`,
			ErrorMessage: "validation failed",
			Terminal:     true,
		},
	}
	service := NewService(NewRepository()).
		WithDispatcher(dispatcher).
		WithRequestStatusUpdater(requestUpdater)

	created, err := service.CreatePendingDelivery(context.Background(), CreateInput{
		RequestID: 9,
		ServerID:  4,
	})
	if err != nil {
		t.Fatalf("create delivery: %v", err)
	}

	record, err := service.SubmitDHIS2Delivery(context.Background(), DispatchInput{
		DeliveryID:  created.ID,
		RequestID:   created.RequestID,
		RequestUID:  created.RequestUID,
		PayloadBody: `{"trackedEntity":"123"}`,
		Server: ServerSnapshot{
			Code:       "dhis2-ug",
			SystemType: "dhis2",
			BaseURL:    "https://dhis.example.com",
			HTTPMethod: "POST",
		},
	})
	if err != nil {
		t.Fatalf("submit dhis2 delivery: %v", err)
	}
	if record.Status != StatusFailed {
		t.Fatalf("expected failed delivery, got %+v", record)
	}
	if len(requestUpdater.failed) != 1 || requestUpdater.failed[0] != created.RequestID {
		t.Fatalf("expected request failure update, got %+v", requestUpdater)
	}
}

func TestServiceRetrySubmissionUsesSameRateLimitedDispatcher(t *testing.T) {
	dispatcher := &fakeDispatcher{
		result: DispatchResult{
			HTTPStatus:   intPtr(200),
			ResponseBody: `{"status":"OK","response":{"status":"SUCCESS"}}`,
			Terminal:     true,
			Succeeded:    true,
		},
	}
	service := NewService(NewRepository()).WithDispatcher(dispatcher)

	created, err := service.CreatePendingDelivery(context.Background(), CreateInput{
		RequestID: 30,
		ServerID:  8,
	})
	if err != nil {
		t.Fatalf("create delivery: %v", err)
	}

	input := DispatchInput{
		RequestID:   created.RequestID,
		RequestUID:  created.RequestUID,
		PayloadBody: `{"trackedEntity":"123"}`,
		Server: ServerSnapshot{
			ID:         created.ServerID,
			Code:       "dhis2-ug",
			Name:       created.ServerName,
			SystemType: "dhis2",
			BaseURL:    "https://dhis.example.com",
			HTTPMethod: "POST",
		},
	}

	input.DeliveryID = created.ID
	if _, err := service.SubmitDHIS2Delivery(context.Background(), input); err != nil {
		t.Fatalf("submit original delivery: %v", err)
	}

	failed, err := service.CreatePendingDelivery(context.Background(), CreateInput{
		RequestID: created.RequestID,
		ServerID:  created.ServerID,
	})
	if err != nil {
		t.Fatalf("create second delivery: %v", err)
	}
	if _, err := service.MarkRunning(context.Background(), failed.ID); err != nil {
		t.Fatalf("mark running: %v", err)
	}
	if _, err := service.MarkFailed(context.Background(), CompletionInput{
		ID:           failed.ID,
		ErrorMessage: "timeout",
	}); err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	retried, err := service.RetryDelivery(context.Background(), nil, failed.ID)
	if err != nil {
		t.Fatalf("retry delivery: %v", err)
	}

	input.DeliveryID = retried.ID
	if _, err := service.SubmitDHIS2Delivery(context.Background(), input); err != nil {
		t.Fatalf("submit retried delivery: %v", err)
	}

	if got := dispatcher.keys; len(got) != 2 || got[0] != "dhis2-ug" || got[1] != "dhis2-ug" {
		t.Fatalf("expected original and retry submissions to use the same destination-scoped dispatcher path, got %+v", got)
	}
}

func intPtr(value int) *int {
	return &value
}
