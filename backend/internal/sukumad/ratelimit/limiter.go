package ratelimit

import (
	"context"
	"sync"
	"time"
)

type Lease interface {
	Release()
}

type limiterLease struct {
	release func()
}

func (l limiterLease) Release() {
	if l.release != nil {
		l.release()
	}
}

type Limiter struct {
	mu          sync.Mutex
	rps         int
	nextAllowed time.Time
	concurrency chan struct{}
	timeout     time.Duration
}

func NewLimiter(policy Policy) *Limiter {
	return &Limiter{
		rps:         policy.RPS,
		concurrency: make(chan struct{}, policy.MaxConcurrency),
		timeout:     time.Duration(policy.TimeoutMS) * time.Millisecond,
	}
}

func (l *Limiter) Acquire(ctx context.Context) (Lease, error) {
	if l == nil {
		return limiterLease{}, nil
	}
	waitCtx := ctx
	if l.timeout > 0 {
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithTimeout(ctx, l.timeout)
		defer cancel()
	}
	if err := l.wait(waitCtx); err != nil {
		return nil, err
	}
	select {
	case l.concurrency <- struct{}{}:
		return limiterLease{release: func() { <-l.concurrency }}, nil
	case <-waitCtx.Done():
		return nil, waitCtx.Err()
	}
}

func (l *Limiter) wait(ctx context.Context) error {
	if l.rps <= 0 {
		return nil
	}
	l.mu.Lock()
	now := time.Now()
	interval := time.Second / time.Duration(l.rps)
	waitFor := time.Duration(0)
	if l.nextAllowed.After(now) {
		waitFor = l.nextAllowed.Sub(now)
	}
	l.nextAllowed = now.Add(waitFor + interval)
	l.mu.Unlock()

	if waitFor <= 0 {
		return nil
	}

	timer := time.NewTimer(waitFor)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
