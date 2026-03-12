package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestServiceCreateAndResolvePolicy(t *testing.T) {
	service := NewService(NewRepository())

	policy, err := service.CreatePolicy(context.Background(), CreateParams{
		Name:           "Global policy",
		ScopeType:      "global",
		ScopeRef:       "",
		RPS:            10,
		Burst:          20,
		MaxConcurrency: 3,
		TimeoutMS:      500,
		IsActive:       true,
	})
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	if policy.ID == 0 || policy.UID == "" {
		t.Fatalf("unexpected policy: %+v", policy)
	}

	resolved, ok, err := service.ResolveActivePolicy(context.Background(), "global", "")
	if err != nil {
		t.Fatalf("resolve active policy: %v", err)
	}
	if !ok || resolved.ID != policy.ID {
		t.Fatalf("unexpected resolved policy: ok=%v policy=%+v", ok, resolved)
	}
}

func TestLimiterAcquireRelease(t *testing.T) {
	limiter := NewLimiter(Policy{
		RPS:            100,
		Burst:          1,
		MaxConcurrency: 1,
		TimeoutMS:      100,
	})

	lease, err := limiter.Acquire(context.Background())
	if err != nil {
		t.Fatalf("acquire first lease: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		_, acquireErr := limiter.Acquire(ctx)
		done <- acquireErr
	}()

	if err := <-done; err == nil {
		t.Fatal("expected second acquire to time out while concurrency slot is held")
	}

	lease.Release()
}
