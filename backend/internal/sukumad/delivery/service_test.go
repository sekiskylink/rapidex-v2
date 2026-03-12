package delivery

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
