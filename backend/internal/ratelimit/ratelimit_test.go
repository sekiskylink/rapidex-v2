package ratelimit

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestRegistryThrottlesSameDestination(t *testing.T) {
	registry := NewRegistry(func(string) Policy {
		return Policy{RequestsPerSecond: 20, Burst: 1}
	})

	start := time.Now()
	if err := registry.Wait(context.Background(), "dhis2-ug"); err != nil {
		t.Fatalf("first wait: %v", err)
	}
	if err := registry.Wait(context.Background(), "dhis2-ug"); err != nil {
		t.Fatalf("second wait: %v", err)
	}

	if elapsed := time.Since(start); elapsed < 45*time.Millisecond {
		t.Fatalf("expected throttling delay, got %v", elapsed)
	}
}

func TestRegistrySeparatesDestinations(t *testing.T) {
	registry := NewRegistry(func(string) Policy {
		return Policy{RequestsPerSecond: 10, Burst: 1}
	})

	start := time.Now()
	var wg sync.WaitGroup
	for _, key := range []string{"dhis2-a", "rapidpro-b"} {
		wg.Add(1)
		go func(destinationKey string) {
			defer wg.Done()
			if err := registry.Wait(context.Background(), destinationKey); err != nil {
				t.Errorf("wait %s: %v", destinationKey, err)
			}
		}(key)
	}
	wg.Wait()

	if elapsed := time.Since(start); elapsed > 75*time.Millisecond {
		t.Fatalf("expected different destinations to proceed independently, got %v", elapsed)
	}
}

func TestRegistryRespectsContextCancellation(t *testing.T) {
	registry := NewRegistry(func(string) Policy {
		return Policy{RequestsPerSecond: 2, Burst: 1}
	})

	if err := registry.Wait(context.Background(), "dhis2-ug"); err != nil {
		t.Fatalf("seed wait: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	if err := registry.Wait(ctx, "dhis2-ug"); err == nil {
		t.Fatal("expected wait cancellation error")
	}
}

func TestRegistryHandlesConcurrentWorkers(t *testing.T) {
	registry := NewRegistry(func(string) Policy {
		return Policy{RequestsPerSecond: 25, Burst: 1}
	})

	start := time.Now()
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := registry.Wait(context.Background(), "dhis2-ug"); err != nil {
				t.Errorf("wait: %v", err)
			}
		}()
	}
	wg.Wait()

	if elapsed := time.Since(start); elapsed < 110*time.Millisecond {
		t.Fatalf("expected concurrent workers to share throttling, got %v", elapsed)
	}
}

func TestRegistryUsesDefaultsForMissingPolicy(t *testing.T) {
	registry := NewRegistry(func(string) Policy { return Policy{} })

	if err := registry.Wait(context.Background(), "unconfigured-destination"); err != nil {
		t.Fatalf("wait with default policy: %v", err)
	}
}
