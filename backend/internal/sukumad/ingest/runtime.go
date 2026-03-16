package ingest

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"basepro/backend/internal/sukumad/worker"
	"github.com/fsnotify/fsnotify"
)

type watcher interface {
	Ensure(context.Context, string) error
	Drain() []string
	Close() error
}

type Runtime struct {
	service    *Service
	getConfig  func() RuntimeConfig
	watcher    watcher
	lastScanAt time.Time
	mu         sync.Mutex
}

func NewRuntime(service *Service, getConfig func() RuntimeConfig) *Runtime {
	return &Runtime{
		service:   service,
		getConfig: getConfig,
		watcher:   newFSWatcher(),
	}
}

func (r *Runtime) Run(ctx context.Context, exec worker.Execution) error {
	cfg := r.getConfig()
	if !cfg.Enabled {
		return nil
	}
	if err := r.ensureDirectories(cfg); err != nil {
		return err
	}
	if err := r.watcher.Ensure(ctx, cfg.InboxPath); err != nil {
		return err
	}
	if _, err := r.service.RequeueStaleClaims(ctx, cfg); err != nil {
		return err
	}
	for _, path := range r.watcher.Drain() {
		if _, err := r.service.DiscoverPath(ctx, path, cfg); err != nil {
			return err
		}
		exec.Increment("files_seen")
	}
	if r.shouldScan(cfg.ScanInterval) {
		count, err := r.service.DiscoverDirectory(ctx, cfg)
		if err != nil {
			return err
		}
		if count > 0 {
			exec.AddCount("files_seen", count)
		}
	}
	if err := r.service.ProcessBatch(ctx, execAdapter{Execution: exec}, cfg); err != nil {
		return err
	}
	return nil
}

func (r *Runtime) ensureDirectories(cfg RuntimeConfig) error {
	for _, path := range []string{cfg.InboxPath, cfg.ProcessingPath, cfg.ProcessedPath, cfg.FailedPath} {
		if err := os.MkdirAll(path, 0o750); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runtime) shouldScan(interval time.Duration) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	if r.lastScanAt.IsZero() || now.Sub(r.lastScanAt) >= interval {
		r.lastScanAt = now
		return true
	}
	return false
}

type execAdapter struct {
	worker.Execution
}

func (e execAdapter) RunID() int64 {
	return e.Execution.RunID
}

type fsWatcher struct {
	mu      sync.Mutex
	watcher *fsnotify.Watcher
	path    string
	pending map[string]struct{}
	started bool
}

func newFSWatcher() watcher {
	return &fsWatcher{pending: map[string]struct{}{}}
}

func (w *fsWatcher) Ensure(ctx context.Context, path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	cleanPath := filepath.Clean(path)
	if w.started && w.path == cleanPath {
		return nil
	}
	if w.watcher != nil {
		_ = w.watcher.Close()
	}
	next, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	if err := next.Add(cleanPath); err != nil {
		_ = next.Close()
		return err
	}
	w.watcher = next
	w.path = cleanPath
	w.started = true
	go w.loop(ctx, next)
	return nil
}

func (w *fsWatcher) loop(ctx context.Context, current *fsnotify.Watcher) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-current.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename) == 0 {
				continue
			}
			w.mu.Lock()
			w.pending[filepath.Clean(event.Name)] = struct{}{}
			w.mu.Unlock()
		case <-current.Errors:
		}
	}
}

func (w *fsWatcher) Drain() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	items := make([]string, 0, len(w.pending))
	for path := range w.pending {
		items = append(items, path)
	}
	w.pending = map[string]struct{}{}
	return items
}

func (w *fsWatcher) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.watcher == nil {
		return nil
	}
	return w.watcher.Close()
}
