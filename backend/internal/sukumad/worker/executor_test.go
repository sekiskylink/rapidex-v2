package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"basepro/backend/internal/sukumad/delivery"
	requests "basepro/backend/internal/sukumad/request"
	"basepro/backend/internal/sukumad/server"
	"basepro/backend/internal/sukumad/traceevent"
)

type fakeClaimRepo struct {
	pending []delivery.Record
	retries []delivery.Record
}

func (f *fakeClaimRepo) ClaimNextPendingDelivery(_ context.Context, _ time.Time) (delivery.Record, error) {
	if len(f.pending) == 0 {
		return delivery.Record{}, delivery.ErrNoEligibleDelivery
	}
	record := f.pending[0]
	f.pending = f.pending[1:]
	return record, nil
}

func (f *fakeClaimRepo) ClaimNextRetryDelivery(_ context.Context, _ time.Time) (delivery.Record, error) {
	if len(f.retries) == 0 {
		return delivery.Record{}, delivery.ErrNoEligibleDelivery
	}
	record := f.retries[0]
	f.retries = f.retries[1:]
	return record, nil
}

type fakeRequestService struct {
	items map[int64]requests.Record
}

func (f *fakeRequestService) GetRequest(_ context.Context, id int64) (requests.Record, error) {
	item, ok := f.items[id]
	if !ok {
		return requests.Record{}, errors.New("request not found")
	}
	return item, nil
}

type fakeServerService struct {
	items map[int64]server.Record
}

func (f *fakeServerService) GetServer(_ context.Context, id int64) (server.Record, error) {
	item, ok := f.items[id]
	if !ok {
		return server.Record{}, errors.New("server not found")
	}
	return item, nil
}

type fakeSubmitter struct {
	inputs  []delivery.DispatchInput
	results []delivery.Record
}

func (f *fakeSubmitter) SubmitDHIS2Delivery(_ context.Context, input delivery.DispatchInput) (delivery.Record, error) {
	f.inputs = append(f.inputs, input)
	if len(f.results) == 0 {
		return delivery.Record{ID: input.DeliveryID, RequestID: input.RequestID, UID: "delivery", Status: delivery.StatusSucceeded}, nil
	}
	result := f.results[0]
	f.results = f.results[1:]
	return result, nil
}

type fakeEventWriter struct {
	events []traceevent.WriteInput
}

func (f *fakeEventWriter) AppendEvent(_ context.Context, input traceevent.WriteInput) error {
	f.events = append(f.events, input)
	return nil
}

func TestDeliveryExecutorRunSendBatchUsesSharedSubmissionPath(t *testing.T) {
	claimRepo := &fakeClaimRepo{
		pending: []delivery.Record{{
			ID:            41,
			UID:           "delivery-41",
			RequestID:     7,
			CorrelationID: "corr-1",
			ServerID:      3,
			Status:        delivery.StatusRunning,
		}},
	}
	requestService := &fakeRequestService{
		items: map[int64]requests.Record{
			7: {
				ID:            7,
				UID:           "request-7",
				CorrelationID: "corr-1",
				PayloadBody:   `{"trackedEntity":"123"}`,
				URLSuffix:     "/tracker",
			},
		},
	}
	serverService := &fakeServerService{
		items: map[int64]server.Record{
			3: {
				ID:         3,
				Code:       "dhis2-ug",
				Name:       "DHIS2 Uganda",
				SystemType: "dhis2",
				BaseURL:    "https://dhis.example.com",
				HTTPMethod: "POST",
			},
		},
	}
	submitter := &fakeSubmitter{
		results: []delivery.Record{{
			ID:                   41,
			RequestID:            7,
			UID:                  "delivery-41",
			Status:               delivery.StatusPending,
			SubmissionHoldReason: "window_closed",
		}},
	}
	eventWriter := &fakeEventWriter{}
	executor := NewDeliveryExecutor(claimRepo, requestService, serverService, submitter).WithEventWriter(eventWriter)

	if err := executor.RunSendBatch(context.Background(), Execution{RunID: 99}, 1); err != nil {
		t.Fatalf("run send batch: %v", err)
	}
	if len(submitter.inputs) != 1 {
		t.Fatalf("expected one shared submit call, got %d", len(submitter.inputs))
	}
	if submitter.inputs[0].DeliveryID != 41 || submitter.inputs[0].RequestUID != "request-7" || submitter.inputs[0].Server.Code != "dhis2-ug" {
		t.Fatalf("unexpected submit input: %+v", submitter.inputs[0])
	}
	if len(eventWriter.events) < 4 {
		t.Fatalf("expected worker events to be recorded, got %d", len(eventWriter.events))
	}
	if got := eventWriter.events[len(eventWriter.events)-1].EventType; got != "delivery.submission.deferred" {
		t.Fatalf("expected deferred completion event, got %q", got)
	}
}

func TestDeliveryExecutorRunRetryBatchUsesSameSubmissionPath(t *testing.T) {
	claimRepo := &fakeClaimRepo{
		retries: []delivery.Record{{
			ID:            51,
			UID:           "delivery-51",
			RequestID:     8,
			CorrelationID: "corr-2",
			ServerID:      4,
			Status:        delivery.StatusRunning,
		}},
	}
	requestService := &fakeRequestService{
		items: map[int64]requests.Record{
			8: {
				ID:            8,
				UID:           "request-8",
				CorrelationID: "corr-2",
				PayloadBody:   `{"dataValues":[]}`,
			},
		},
	}
	serverService := &fakeServerService{
		items: map[int64]server.Record{
			4: {
				ID:         4,
				Code:       "dhis2-ug",
				Name:       "DHIS2 Uganda",
				SystemType: "dhis2",
				BaseURL:    "https://dhis.example.com",
				HTTPMethod: "POST",
			},
		},
	}
	submitter := &fakeSubmitter{
		results: []delivery.Record{{
			ID:        51,
			RequestID: 8,
			UID:       "delivery-51",
			Status:    delivery.StatusSucceeded,
		}},
	}
	executor := NewDeliveryExecutor(claimRepo, requestService, serverService, submitter)

	if err := executor.RunRetryBatch(context.Background(), Execution{RunID: 101}, 1); err != nil {
		t.Fatalf("run retry batch: %v", err)
	}
	if len(submitter.inputs) != 1 {
		t.Fatalf("expected one retry submit call, got %d", len(submitter.inputs))
	}
	if submitter.inputs[0].DeliveryID != 51 || submitter.inputs[0].Server.Code != "dhis2-ug" {
		t.Fatalf("unexpected retry submit input: %+v", submitter.inputs[0])
	}
}
