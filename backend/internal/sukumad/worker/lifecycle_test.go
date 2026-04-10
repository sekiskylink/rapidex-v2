package worker

import (
	"context"
	"testing"
	"time"

	asyncjobs "basepro/backend/internal/sukumad/async"
	"basepro/backend/internal/sukumad/delivery"
	requests "basepro/backend/internal/sukumad/request"
	"basepro/backend/internal/sukumad/server"
)

type scriptedDispatcher struct {
	results []delivery.DispatchResult
}

func (s *scriptedDispatcher) Submit(_ context.Context, _ delivery.DispatchInput) (delivery.DispatchResult, error) {
	if len(s.results) == 0 {
		return delivery.DispatchResult{}, nil
	}
	result := s.results[0]
	s.results = s.results[1:]
	return result, nil
}

type staticServerService struct {
	items map[int64]server.Record
}

func (s staticServerService) GetServer(_ context.Context, id int64) (server.Record, error) {
	return s.items[id], nil
}

type staticPoller struct {
	result asyncjobs.RemotePollResult
}

func (p staticPoller) Poll(context.Context, asyncjobs.Record) (asyncjobs.RemotePollResult, error) {
	return p.result, nil
}

func TestWorkerDrivenLifecycleAsyncAcceptedThenSucceeded(t *testing.T) {
	ctx := context.Background()
	requestRepo := requests.NewRepository()
	deliveryRepo := delivery.NewRepository()
	asyncRepo := asyncjobs.NewRepository()

	requestService := requests.NewService(requestRepo)
	dispatcher := &scriptedDispatcher{
		results: []delivery.DispatchResult{{
			HTTPStatus:   intPtr(202),
			ResponseBody: `{"status":"PENDING"}`,
			RemoteJobID:  "job-1",
			PollURL:      "https://dhis.example.com/jobs/job-1",
			RemoteStatus: asyncjobs.StatePending,
			RemoteResponse: map[string]any{
				"status": "PENDING",
			},
			Async: true,
		}},
	}
	deliveryService := delivery.NewService(deliveryRepo).
		WithDispatcher(dispatcher).
		WithRequestStatusUpdater(requestService).
		WithTargetUpdater(requestService)
	asyncService := asyncjobs.NewService(asyncRepo).
		WithReconciliation(deliveryService, requestService)
	deliveryService.WithAsyncService(asyncService)
	requestService.WithDeliveryService(deliveryService)

	executor := NewDeliveryExecutor(
		deliveryRepo,
		requestService,
		staticServerService{items: map[int64]server.Record{
			3: {
				ID:         3,
				Code:       "dhis2-ug",
				Name:       "DHIS2 Uganda",
				SystemType: "dhis2",
				BaseURL:    "https://dhis.example.com",
				HTTPMethod: "POST",
				UseAsync:   true,
			},
		}},
		deliveryService,
	)

	created, err := requestService.CreateRequest(ctx, requests.CreateInput{
		DestinationServerID: 3,
		Payload:             []byte(`{"trackedEntity":"123"}`),
	})
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if err := executor.RunSendBatch(ctx, Execution{}, 1); err != nil {
		t.Fatalf("run send batch: %v", err)
	}

	processing, err := requestService.GetRequest(ctx, created.ID)
	if err != nil {
		t.Fatalf("reload processing request: %v", err)
	}
	if processing.Status != requests.StatusProcessing || len(processing.Targets) != 1 || processing.Targets[0].Status != requests.TargetStatusProcessing {
		t.Fatalf("expected processing request awaiting async, got %+v", processing)
	}

	tasks, err := asyncService.ListTasks(ctx, asyncjobs.ListQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list async tasks: %v", err)
	}
	if len(tasks.Items) != 1 {
		t.Fatalf("expected one async task, got %+v", tasks)
	}
	due := time.Now().UTC().Add(-time.Second)
	if _, err := asyncService.UpdateTaskStatus(ctx, asyncjobs.UpdateStatusInput{
		ID:             tasks.Items[0].ID,
		RemoteJobID:    tasks.Items[0].RemoteJobID,
		PollURL:        tasks.Items[0].PollURL,
		RemoteStatus:   tasks.Items[0].RemoteStatus,
		NextPollAt:     &due,
		RemoteResponse: tasks.Items[0].RemoteResponse,
	}); err != nil {
		t.Fatalf("mark task due: %v", err)
	}
	if err := asyncService.PollDueTasks(ctx, asyncjobs.PollExecution{
		WorkerRunID:  22,
		ClaimTimeout: time.Minute,
	}, 1, staticPoller{result: asyncjobs.RemotePollResult{
		StatusCode:    intPtr(200),
		RemoteStatus:  asyncjobs.StateSucceeded,
		TerminalState: asyncjobs.StateSucceeded,
		RemoteResponse: map[string]any{
			"status": "OK",
		},
	}}); err != nil {
		t.Fatalf("poll due tasks: %v", err)
	}

	completed, err := requestService.GetRequest(ctx, created.ID)
	if err != nil {
		t.Fatalf("reload completed request: %v", err)
	}
	if completed.Status != requests.StatusCompleted || len(completed.Targets) != 1 || completed.Targets[0].Status != requests.TargetStatusSucceeded {
		t.Fatalf("expected completed request after async reconciliation, got %+v", completed)
	}
}

func TestWorkerDrivenLifecycleRetryCompletesAfterInitialFailure(t *testing.T) {
	ctx := context.Background()
	requestRepo := requests.NewRepository()
	deliveryRepo := delivery.NewRepository()
	asyncRepo := asyncjobs.NewRepository()

	requestService := requests.NewService(requestRepo)
	dispatcher := &scriptedDispatcher{
		results: []delivery.DispatchResult{
			{
				HTTPStatus:   intPtr(400),
				ResponseBody: `{"status":"ERROR"}`,
				ErrorMessage: "validation failed",
				Terminal:     true,
			},
			{
				HTTPStatus:   intPtr(200),
				ResponseBody: `{"status":"OK"}`,
				Terminal:     true,
				Succeeded:    true,
			},
		},
	}
	deliveryService := delivery.NewService(deliveryRepo).
		WithDispatcher(dispatcher).
		WithRequestStatusUpdater(requestService).
		WithTargetUpdater(requestService)
	asyncService := asyncjobs.NewService(asyncRepo).
		WithReconciliation(deliveryService, requestService)
	deliveryService.WithAsyncService(asyncService)
	requestService.WithDeliveryService(deliveryService)

	executor := NewDeliveryExecutor(
		deliveryRepo,
		requestService,
		staticServerService{items: map[int64]server.Record{
			4: {
				ID:         4,
				Code:       "dhis2-ug",
				Name:       "DHIS2 Uganda",
				SystemType: "dhis2",
				BaseURL:    "https://dhis.example.com",
				HTTPMethod: "POST",
			},
		}},
		deliveryService,
	)

	created, err := requestService.CreateRequest(ctx, requests.CreateInput{
		DestinationServerID: 4,
		Payload:             []byte(`{"trackedEntity":"123"}`),
	})
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if err := executor.RunSendBatch(ctx, Execution{}, 1); err != nil {
		t.Fatalf("run initial send batch: %v", err)
	}

	failedRequest, err := requestService.GetRequest(ctx, created.ID)
	if err != nil {
		t.Fatalf("reload failed request: %v", err)
	}
	if failedRequest.Status != requests.StatusFailed || len(failedRequest.Targets) != 1 || failedRequest.Targets[0].Status != requests.TargetStatusFailed {
		t.Fatalf("expected failed request after first send, got %+v", failedRequest)
	}
	deliveries, err := deliveryService.ListDeliveries(ctx, delivery.ListQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list deliveries: %v", err)
	}
	var failedDeliveryID int64
	for _, item := range deliveries.Items {
		if item.RequestID == created.ID && item.Status == delivery.StatusFailed {
			failedDeliveryID = item.ID
			break
		}
	}
	if failedDeliveryID == 0 {
		t.Fatalf("expected failed delivery for request %d, got %+v", created.ID, deliveries.Items)
	}

	retryDelivery, err := deliveryService.RetryDelivery(ctx, nil, failedDeliveryID)
	if err != nil {
		t.Fatalf("schedule retry delivery: %v", err)
	}
	due := time.Now().UTC().Add(-time.Second)
	if _, err := deliveryRepo.UpdateDelivery(ctx, delivery.UpdateParams{
		ID:                   retryDelivery.ID,
		Status:               delivery.StatusRetrying,
		HTTPStatus:           nil,
		ResponseBody:         retryDelivery.ResponseBody,
		ResponseContentType:  retryDelivery.ResponseContentType,
		ResponseBodyFiltered: retryDelivery.ResponseBodyFiltered,
		ResponseSummary:      retryDelivery.ResponseSummary,
		ErrorMessage:         retryDelivery.ErrorMessage,
		SubmissionHoldReason: retryDelivery.SubmissionHoldReason,
		NextEligibleAt:       retryDelivery.NextEligibleAt,
		HoldPolicySource:     retryDelivery.HoldPolicySource,
		TerminalReason:       retryDelivery.TerminalReason,
		StartedAt:            nil,
		FinishedAt:           nil,
		RetryAt:              &due,
	}); err != nil {
		t.Fatalf("make retry delivery due: %v", err)
	}

	if err := executor.RunRetryBatch(ctx, Execution{}, 1); err != nil {
		t.Fatalf("run retry batch: %v", err)
	}

	completed, err := requestService.GetRequest(ctx, created.ID)
	if err != nil {
		t.Fatalf("reload completed request: %v", err)
	}
	if completed.Status != requests.StatusCompleted || len(completed.Targets) != 1 || completed.Targets[0].Status != requests.TargetStatusSucceeded {
		t.Fatalf("expected request completion after retry success, got %+v", completed)
	}
}

func TestWorkerRestartPicksDurablePendingDelivery(t *testing.T) {
	ctx := context.Background()
	requestRepo := requests.NewRepository()
	deliveryRepo := delivery.NewRepository()
	requestService := requests.NewService(requestRepo)
	dispatcher := &scriptedDispatcher{
		results: []delivery.DispatchResult{{
			HTTPStatus:   intPtr(200),
			ResponseBody: `{"status":"OK"}`,
			Terminal:     true,
			Succeeded:    true,
		}},
	}
	deliveryService := delivery.NewService(deliveryRepo).
		WithDispatcher(dispatcher).
		WithRequestStatusUpdater(requestService).
		WithTargetUpdater(requestService)
	requestService.WithDeliveryService(deliveryService)

	created, err := requestService.CreateRequest(ctx, requests.CreateInput{
		DestinationServerID: 5,
		Payload:             []byte(`{"trackedEntity":"restart"}`),
	})
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if created.Status != requests.StatusPending {
		t.Fatalf("expected durable pending request before worker restart, got %+v", created)
	}

	// Simulate a fresh worker process by creating a new executor against the same durable repos.
	executor := NewDeliveryExecutor(
		deliveryRepo,
		requestService,
		staticServerService{items: map[int64]server.Record{
			5: {
				ID:         5,
				Code:       "dhis2-ug",
				Name:       "DHIS2 Uganda",
				SystemType: "dhis2",
				BaseURL:    "https://dhis.example.com",
				HTTPMethod: "POST",
			},
		}},
		deliveryService,
	)
	if err := executor.RunSendBatch(ctx, Execution{}, 1); err != nil {
		t.Fatalf("run send batch after restart: %v", err)
	}

	completed, err := requestService.GetRequest(ctx, created.ID)
	if err != nil {
		t.Fatalf("reload request: %v", err)
	}
	if completed.Status != requests.StatusCompleted {
		t.Fatalf("expected restarted worker to complete durable pending work, got %+v", completed)
	}
	if len(completed.Targets) != 1 || completed.Targets[0].Status != requests.TargetStatusSucceeded {
		t.Fatalf("expected restarted worker to mark target succeeded, got %+v", completed.Targets)
	}
}

func intPtr(value int) *int {
	return &value
}
