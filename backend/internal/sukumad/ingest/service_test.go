package ingest

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/audit"
	requests "basepro/backend/internal/sukumad/request"
)

type fakeRequestCreator struct {
	createFn func(context.Context, requests.CreateInput) (requests.Record, error)
	inputs   []requests.CreateInput
}

func (f *fakeRequestCreator) CreateRequest(ctx context.Context, input requests.CreateInput) (requests.Record, error) {
	f.inputs = append(f.inputs, input)
	return f.createFn(ctx, input)
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

func TestProcessBatchMovesSuccessfulFileToProcessed(t *testing.T) {
	timestamp := time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC)
	cfg := newRuntimeConfig(t)
	requestCreator := &fakeRequestCreator{
		createFn: func(_ context.Context, input requests.CreateInput) (requests.Record, error) {
			return requests.Record{
				ID:        44,
				UID:       "req-44",
				Status:    requests.StatusPending,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}, nil
		},
	}
	auditRepo := &fakeAuditRepo{}
	service := NewService(NewRepository(), requestCreator, audit.NewService(auditRepo))
	service.now = func() time.Time { return timestamp }

	path := filepath.Join(cfg.InboxPath, "request.json")
	if err := os.WriteFile(path, []byte(`{"sourceSystem":"emr","destinationServerId":7,"idempotencyKey":"abc-1","payload":{"trackedEntity":"1"}}`), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := service.DiscoverPath(context.Background(), path, cfg); err != nil {
		t.Fatalf("discover path: %v", err)
	}

	service.now = func() time.Time { return timestamp.Add(2 * time.Second) }
	if err := service.ProcessBatch(context.Background(), runIDStub{runID: 9}, cfg); err != nil {
		t.Fatalf("process batch: %v", err)
	}

	record, err := service.repo.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("get ingest record: %v", err)
	}
	if record.Status != StatusProcessed {
		t.Fatalf("expected processed status, got %+v", record)
	}
	if record.RequestID == nil || *record.RequestID != 44 {
		t.Fatalf("expected request linkage, got %+v", record)
	}
	if _, err := os.Stat(record.CurrentPath); err != nil {
		t.Fatalf("expected processed file to exist, got %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected inbox file to be moved, got %v", err)
	}
	if len(requestCreator.inputs) != 1 {
		t.Fatalf("expected one request create call, got %d", len(requestCreator.inputs))
	}
	if requestCreator.inputs[0].Extras["ingestSource"] != SourceKindDirectory {
		t.Fatalf("expected ingest extras, got %+v", requestCreator.inputs[0].Extras)
	}
	if len(auditRepo.events) != 1 || auditRepo.events[0].Action != "request.ingested_from_directory" {
		t.Fatalf("unexpected audit events: %+v", auditRepo.events)
	}
}

func TestProcessBatchMovesInvalidFileToFailed(t *testing.T) {
	timestamp := time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC)
	cfg := newRuntimeConfig(t)
	requestCreator := &fakeRequestCreator{
		createFn: func(_ context.Context, input requests.CreateInput) (requests.Record, error) {
			return requests.Record{}, nil
		},
	}
	service := NewService(NewRepository(), requestCreator)
	service.now = func() time.Time { return timestamp }

	path := filepath.Join(cfg.InboxPath, "bad.json")
	if err := os.WriteFile(path, []byte(`{"destinationServerId":7,"payload":{"trackedEntity":"1"}}`), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := service.DiscoverPath(context.Background(), path, cfg); err != nil {
		t.Fatalf("discover path: %v", err)
	}

	service.now = func() time.Time { return timestamp.Add(2 * time.Second) }
	if err := service.ProcessBatch(context.Background(), runIDStub{runID: 10}, cfg); err != nil {
		t.Fatalf("process batch: %v", err)
	}

	record, err := service.repo.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("get ingest record: %v", err)
	}
	if record.Status != StatusFailed || record.LastErrorCode != ErrorCodeInvalidEnvelope {
		t.Fatalf("expected failed invalid-envelope record, got %+v", record)
	}
	if _, err := os.Stat(record.CurrentPath); err != nil {
		t.Fatalf("expected failed file to be archived, got %v", err)
	}
}

func TestProcessBatchRetriesTransientRequestFailure(t *testing.T) {
	timestamp := time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC)
	cfg := newRuntimeConfig(t)
	requestCreator := &fakeRequestCreator{
		createFn: func(_ context.Context, input requests.CreateInput) (requests.Record, error) {
			return requests.Record{}, errors.New("db unavailable")
		},
	}
	service := NewService(NewRepository(), requestCreator)
	service.now = func() time.Time { return timestamp }

	path := filepath.Join(cfg.InboxPath, "retry.json")
	if err := os.WriteFile(path, []byte(`{"destinationServerId":7,"idempotencyKey":"retry-1","payload":{"trackedEntity":"1"}}`), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := service.DiscoverPath(context.Background(), path, cfg); err != nil {
		t.Fatalf("discover path: %v", err)
	}

	service.now = func() time.Time { return timestamp.Add(2 * time.Second) }
	if err := service.ProcessBatch(context.Background(), runIDStub{runID: 11}, cfg); err != nil {
		t.Fatalf("process batch: %v", err)
	}

	record, err := service.repo.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("get ingest record: %v", err)
	}
	if record.Status != StatusRetry || record.LastErrorCode != ErrorCodeRequestCreate {
		t.Fatalf("expected retry status, got %+v", record)
	}
	if filepath.Dir(record.CurrentPath) != cfg.ProcessingPath {
		t.Fatalf("expected file to remain in processing dir, got %s", record.CurrentPath)
	}
	if record.NextAttemptAt == nil || !record.NextAttemptAt.After(timestamp) {
		t.Fatalf("expected retry time, got %+v", record.NextAttemptAt)
	}
}

func TestProcessBatchTreatsValidationErrorAsTerminal(t *testing.T) {
	timestamp := time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC)
	cfg := newRuntimeConfig(t)
	requestCreator := &fakeRequestCreator{
		createFn: func(_ context.Context, input requests.CreateInput) (requests.Record, error) {
			return requests.Record{}, apperror.Validation("bad request")
		},
	}
	service := NewService(NewRepository(), requestCreator)
	service.now = func() time.Time { return timestamp }

	path := filepath.Join(cfg.InboxPath, "validation.json")
	if err := os.WriteFile(path, []byte(`{"destinationServerId":7,"idempotencyKey":"v-1","payload":{"trackedEntity":"1"}}`), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := service.DiscoverPath(context.Background(), path, cfg); err != nil {
		t.Fatalf("discover path: %v", err)
	}

	service.now = func() time.Time { return timestamp.Add(2 * time.Second) }
	if err := service.ProcessBatch(context.Background(), runIDStub{runID: 12}, cfg); err != nil {
		t.Fatalf("process batch: %v", err)
	}

	record, err := service.repo.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("get ingest record: %v", err)
	}
	if record.Status != StatusFailed || record.LastErrorCode != apperror.CodeValidationFailed {
		t.Fatalf("expected validation failure to be terminal, got %+v", record)
	}
}

type runIDStub struct {
	runID int64
}

func (r runIDStub) RunID() int64 {
	return r.runID
}

func newRuntimeConfig(t *testing.T) RuntimeConfig {
	t.Helper()
	root := t.TempDir()
	cfg := RuntimeConfig{
		Enabled:               true,
		InboxPath:             filepath.Join(root, "inbox"),
		ProcessingPath:        filepath.Join(root, "processing"),
		ProcessedPath:         filepath.Join(root, "processed"),
		FailedPath:            filepath.Join(root, "failed"),
		AllowedExtensions:     []string{".json"},
		DefaultSourceSystem:   "directory",
		RequireIdempotencyKey: true,
		Debounce:              time.Second,
		RetryDelay:            30 * time.Second,
		ClaimTimeout:          5 * time.Minute,
		ScanInterval:          time.Minute,
		BatchSize:             4,
	}
	for _, path := range []string{cfg.InboxPath, cfg.ProcessingPath, cfg.ProcessedPath, cfg.FailedPath} {
		if err := os.MkdirAll(path, 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
	}
	return cfg
}
