package dashboard

import (
	"context"
	"testing"
	"time"
)

type stubRepository struct {
	snapshot   Snapshot
	err        error
	observedAt time.Time
}

func (r *stubRepository) GetSnapshot(_ context.Context, now time.Time) (Snapshot, error) {
	r.observedAt = now
	return r.snapshot, r.err
}

func TestServiceGetOperationsSnapshotUsesClock(t *testing.T) {
	fixed := time.Date(2026, 3, 18, 9, 30, 0, 0, time.UTC)
	repo := &stubRepository{snapshot: Snapshot{GeneratedAt: fixed}}
	service := NewService(repo).WithClock(func() time.Time { return fixed })

	snapshot, err := service.GetOperationsSnapshot(context.Background())
	if err != nil {
		t.Fatalf("get operations snapshot: %v", err)
	}
	if !repo.observedAt.Equal(fixed) {
		t.Fatalf("expected repository to observe %s, got %s", fixed, repo.observedAt)
	}
	if !snapshot.GeneratedAt.Equal(fixed) {
		t.Fatalf("expected snapshot generatedAt %s, got %s", fixed, snapshot.GeneratedAt)
	}
}
