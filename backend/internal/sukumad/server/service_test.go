package server

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"basepro/backend/internal/audit"
)

type fakeRepo struct {
	listFn   func(ctx context.Context, query ListQuery) (ListResult, error)
	getFn    func(ctx context.Context, id int64) (Record, error)
	createFn func(ctx context.Context, params CreateParams) (Record, error)
	updateFn func(ctx context.Context, params UpdateParams) (Record, error)
	deleteFn func(ctx context.Context, id int64) error
}

func (f *fakeRepo) ListServers(ctx context.Context, query ListQuery) (ListResult, error) {
	return f.listFn(ctx, query)
}
func (f *fakeRepo) GetServerByID(ctx context.Context, id int64) (Record, error) {
	return f.getFn(ctx, id)
}
func (f *fakeRepo) CreateServer(ctx context.Context, params CreateParams) (Record, error) {
	return f.createFn(ctx, params)
}
func (f *fakeRepo) UpdateServer(ctx context.Context, params UpdateParams) (Record, error) {
	return f.updateFn(ctx, params)
}
func (f *fakeRepo) DeleteServer(ctx context.Context, id int64) error {
	return f.deleteFn(ctx, id)
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

func TestServiceCreateServerValidatesInput(t *testing.T) {
	service := NewService(&fakeRepo{}, audit.NewService(&fakeAuditRepo{}))

	_, err := service.CreateServer(context.Background(), CreateInput{
		Name:           "",
		Code:           "INVALID CODE",
		SystemType:     "",
		BaseURL:        "ftp://bad",
		EndpointType:   "",
		HTTPMethod:     "TRACE",
		ParseResponses: true,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestServiceUpdateServerWritesSuspendAuditEvent(t *testing.T) {
	auditRepo := &fakeAuditRepo{}
	service := NewService(&fakeRepo{
		getFn: func(_ context.Context, id int64) (Record, error) {
			return Record{
				ID:             id,
				Name:           "DHIS2",
				Code:           "dhis2",
				SystemType:     "dhis2",
				BaseURL:        "https://dhis.example.com",
				EndpointType:   "http",
				HTTPMethod:     "POST",
				ParseResponses: true,
				Headers:        map[string]string{},
				URLParams:      map[string]string{},
				Suspended:      false,
			}, nil
		},
		updateFn: func(_ context.Context, params UpdateParams) (Record, error) {
			return Record{
				ID:             params.ID,
				Name:           params.Name,
				Code:           params.Code,
				SystemType:     params.SystemType,
				BaseURL:        params.BaseURL,
				EndpointType:   params.EndpointType,
				HTTPMethod:     params.HTTPMethod,
				UseAsync:       params.UseAsync,
				ParseResponses: params.ParseResponses,
				Headers:        params.Headers,
				URLParams:      params.URLParams,
				Suspended:      params.Suspended,
				UpdatedAt:      time.Now().UTC(),
			}, nil
		},
	}, audit.NewService(auditRepo))

	actorID := int64(8)
	updated, err := service.UpdateServer(context.Background(), UpdateInput{
		ID:             21,
		Name:           "DHIS2",
		Code:           "dhis2",
		SystemType:     "dhis2",
		BaseURL:        "https://dhis.example.com",
		EndpointType:   "http",
		HTTPMethod:     "POST",
		ParseResponses: true,
		Headers:        map[string]string{},
		URLParams:      map[string]string{},
		Suspended:      true,
		ActorID:        &actorID,
	})
	if err != nil {
		t.Fatalf("update server: %v", err)
	}
	if !updated.Suspended {
		t.Fatal("expected suspended server")
	}
	if len(auditRepo.events) != 2 {
		t.Fatalf("expected 2 audit events, got %d", len(auditRepo.events))
	}
	if auditRepo.events[0].Action != "server.updated" {
		t.Fatalf("expected server.updated, got %s", auditRepo.events[0].Action)
	}
	if auditRepo.events[1].Action != "server.suspended" {
		t.Fatalf("expected server.suspended, got %s", auditRepo.events[1].Action)
	}
}

func TestServiceDeleteServerNotFound(t *testing.T) {
	service := NewService(&fakeRepo{
		getFn: func(_ context.Context, _ int64) (Record, error) {
			return Record{}, sql.ErrNoRows
		},
	}, audit.NewService(&fakeAuditRepo{}))

	if err := service.DeleteServer(context.Background(), nil, 99); err == nil {
		t.Fatal("expected delete not found error")
	}
}
