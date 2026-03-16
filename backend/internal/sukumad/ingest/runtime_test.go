package ingest

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	requests "basepro/backend/internal/sukumad/request"
	"basepro/backend/internal/sukumad/worker"
)

type fakeWatcher struct {
	paths      []string
	ensurePath string
}

func (f *fakeWatcher) Ensure(_ context.Context, path string) error {
	f.ensurePath = path
	return nil
}

func (f *fakeWatcher) Drain() []string {
	items := append([]string{}, f.paths...)
	f.paths = nil
	return items
}

func (f *fakeWatcher) Close() error {
	return nil
}

func TestRuntimeDrainsWatcherAndProcessesFile(t *testing.T) {
	cfg := newRuntimeConfig(t)
	cfg.Debounce = 0
	path := filepath.Join(cfg.InboxPath, "runtime.json")
	if err := os.WriteFile(path, []byte(`{"destinationServerId":7,"idempotencyKey":"runtime-1","payload":{"trackedEntity":"1"}}`), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	now := time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC)
	service := NewService(NewRepository(), &fakeRequestCreator{
		createFn: func(_ context.Context, input requests.CreateInput) (requests.Record, error) {
			return requests.Record{ID: 15, UID: "req-runtime"}, nil
		},
	})
	service.now = func() time.Time { return now.Add(2 * time.Second) }
	rt := NewRuntime(service, func() RuntimeConfig { return cfg })
	fw := &fakeWatcher{paths: []string{path}}
	rt.watcher = fw

	exec := worker.Execution{
		RunID: 5,
		AddCount: func(name string, delta int) {
			if name != "files_seen" || delta <= 0 {
				t.Fatalf("unexpected count update %s=%d", name, delta)
			}
		},
		SetMeta: func(string, any) {},
	}

	if err := rt.Run(context.Background(), exec); err != nil {
		t.Fatalf("runtime run: %v", err)
	}
	if fw.ensurePath != cfg.InboxPath {
		t.Fatalf("expected watcher to be ensured on inbox, got %s", fw.ensurePath)
	}
	record, err := service.repo.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("get ingest record: %v", err)
	}
	if record.Status != StatusProcessed {
		t.Fatalf("expected processed record, got %+v", record)
	}
}
