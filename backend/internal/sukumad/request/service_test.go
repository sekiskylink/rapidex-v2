package request

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"basepro/backend/internal/audit"
	"basepro/backend/internal/sukumad/traceevent"
)

type fakeRepo struct {
	listFn   func(ctx context.Context, query ListQuery) (ListResult, error)
	getFn    func(ctx context.Context, id int64) (Record, error)
	createFn func(ctx context.Context, params CreateParams) (Record, error)
	updateFn func(ctx context.Context, id int64, status string, reason string, deferredUntil *time.Time) (Record, error)
}

func (f *fakeRepo) ListRequests(ctx context.Context, query ListQuery) (ListResult, error) {
	return f.listFn(ctx, query)
}

func (f *fakeRepo) GetRequestByID(ctx context.Context, id int64) (Record, error) {
	return f.getFn(ctx, id)
}

func (f *fakeRepo) CreateRequest(ctx context.Context, params CreateParams) (Record, error) {
	return f.createFn(ctx, params)
}

func (f *fakeRepo) UpdateRequestStatus(ctx context.Context, id int64, status string, reason string, deferredUntil *time.Time) (Record, error) {
	if f.updateFn == nil {
		return Record{}, sql.ErrNoRows
	}
	return f.updateFn(ctx, id, status, reason, deferredUntil)
}

func (f *fakeRepo) CreateTargets(context.Context, int64, []CreateTargetParams) ([]TargetRecord, error) {
	return []TargetRecord{}, nil
}

func (f *fakeRepo) ListTargetsByRequest(context.Context, int64) ([]TargetRecord, error) {
	return []TargetRecord{}, nil
}

func (f *fakeRepo) UpdateTarget(context.Context, UpdateTargetParams) (TargetRecord, error) {
	return TargetRecord{}, nil
}

func (f *fakeRepo) CreateDependencies(context.Context, int64, []int64) error {
	return nil
}

func (f *fakeRepo) ListDependencies(context.Context, int64) ([]DependencyRef, error) {
	return []DependencyRef{}, nil
}

func (f *fakeRepo) ListDependents(context.Context, int64) ([]DependencyRef, error) {
	return []DependencyRef{}, nil
}

func (f *fakeRepo) GetDependencyStatuses(context.Context, int64) ([]DependencyStatus, error) {
	return []DependencyStatus{}, nil
}

func (f *fakeRepo) DependencyPathExists(context.Context, int64, int64) (bool, error) {
	return false, nil
}

type fakeAuditRepo struct {
	events []audit.Event
}

type fakeEventWriter struct {
	events []traceevent.WriteInput
}

func (f *fakeEventWriter) AppendEvent(_ context.Context, input traceevent.WriteInput) error {
	f.events = append(f.events, input)
	return nil
}

func (f *fakeAuditRepo) Insert(_ context.Context, event audit.Event) error {
	f.events = append(f.events, event)
	return nil
}

func (f *fakeAuditRepo) List(_ context.Context, _ audit.ListFilter) (audit.ListResult, error) {
	return audit.ListResult{}, nil
}

func TestServiceCreateRequestValidatesInput(t *testing.T) {
	service := NewService(&fakeRepo{}, audit.NewService(&fakeAuditRepo{}))

	_, err := service.CreateRequest(context.Background(), CreateInput{
		DestinationServerID: 0,
		Payload:             []byte(`not-json`),
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestServiceCreateRequestWritesAuditEvent(t *testing.T) {
	auditRepo := &fakeAuditRepo{}
	eventWriter := &fakeEventWriter{}
	service := NewService(&fakeRepo{
		createFn: func(_ context.Context, params CreateParams) (Record, error) {
			return Record{
				ID:                    11,
				UID:                   params.UID,
				DestinationServerID:   params.DestinationServerID,
				DestinationServerName: "DHIS2 Uganda",
				Status:                params.Status,
				CorrelationID:         params.CorrelationID,
				PayloadBody:           params.PayloadBody,
				Payload:               []byte(params.PayloadBody),
				CreatedAt:             time.Now().UTC(),
				UpdatedAt:             time.Now().UTC(),
				CreatedBy:             params.CreatedBy,
			}, nil
		},
	}, audit.NewService(auditRepo)).WithEventWriter(eventWriter)

	actorID := int64(8)
	created, err := service.CreateRequest(context.Background(), CreateInput{
		SourceSystem:        "emr",
		DestinationServerID: 3,
		CorrelationID:       "corr-1",
		Payload:             []byte(`{"trackedEntity":"123"}`),
		Extras:              map[string]any{"priority": "high"},
		ActorID:             &actorID,
	})
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if created.Status != StatusPending {
		t.Fatalf("expected pending status, got %s", created.Status)
	}
	if len(auditRepo.events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(auditRepo.events))
	}
	if auditRepo.events[0].Action != "request.created" {
		t.Fatalf("expected request.created, got %s", auditRepo.events[0].Action)
	}
	if len(eventWriter.events) != 1 || eventWriter.events[0].EventType != traceevent.EventRequestCreated {
		t.Fatalf("expected request.created event write, got %+v", eventWriter.events)
	}
}

func TestServiceGetRequestNotFound(t *testing.T) {
	service := NewService(&fakeRepo{
		getFn: func(_ context.Context, _ int64) (Record, error) {
			return Record{}, sql.ErrNoRows
		},
	}, audit.NewService(&fakeAuditRepo{}))

	if _, err := service.GetRequest(context.Background(), 99); err == nil {
		t.Fatal("expected not found error")
	}
}
