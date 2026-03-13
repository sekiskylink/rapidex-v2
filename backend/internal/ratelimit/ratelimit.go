package ratelimit

import (
	"context"
	"sync"

	xrate "golang.org/x/time/rate"
)

const (
	DefaultRequestsPerSecond = 2.0
	DefaultBurst             = 2
)

type Limiter interface {
	Wait(ctx context.Context, destinationKey string) error
}

type Policy struct {
	RequestsPerSecond float64
	Burst             int
}

type PolicyProvider func(destinationKey string) Policy

type Registry struct {
	mu       sync.RWMutex
	provider PolicyProvider
	limiters map[string]*registryEntry
}

type registryEntry struct {
	policy  Policy
	limiter *xrate.Limiter
}

func NewRegistry(provider PolicyProvider) *Registry {
	return &Registry{
		provider: provider,
		limiters: map[string]*registryEntry{},
	}
}

func (r *Registry) Wait(ctx context.Context, destinationKey string) error {
	if r == nil {
		return nil
	}
	entry := r.entryFor(destinationKey)
	return entry.limiter.Wait(ctx)
}

func (r *Registry) entryFor(destinationKey string) *registryEntry {
	key := normalizeKey(destinationKey)
	policy := normalizePolicy(resolvePolicy(r.provider, key))

	r.mu.RLock()
	existing, ok := r.limiters[key]
	if ok && existing.policy == policy {
		r.mu.RUnlock()
		return existing
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok = r.limiters[key]
	if ok && existing.policy == policy {
		return existing
	}

	entry := &registryEntry{
		policy:  policy,
		limiter: xrate.NewLimiter(xrate.Limit(policy.RequestsPerSecond), policy.Burst),
	}
	r.limiters[key] = entry
	return entry
}

func normalizeKey(value string) string {
	if value == "" {
		return "default"
	}
	return value
}

func resolvePolicy(provider PolicyProvider, destinationKey string) Policy {
	if provider == nil {
		return Policy{}
	}
	return provider(destinationKey)
}

func normalizePolicy(policy Policy) Policy {
	if policy.RequestsPerSecond <= 0 {
		policy.RequestsPerSecond = DefaultRequestsPerSecond
	}
	if policy.Burst <= 0 {
		policy.Burst = DefaultBurst
	}
	return policy
}
