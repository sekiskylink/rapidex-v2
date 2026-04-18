package scheduler

import (
	"context"
	"log/slog"
	"time"

	"basepro/backend/internal/logging"
)

type Runtime struct {
	service   *Service
	getConfig func() RuntimeConfig
}

func NewRuntime(service *Service, getConfig func() RuntimeConfig) *Runtime {
	return &Runtime{service: service, getConfig: getConfig}
}

func (r *Runtime) RunDispatcher(ctx context.Context) error {
	if r == nil || r.service == nil || r.getConfig == nil {
		<-ctx.Done()
		return ctx.Err()
	}

	for {
		cfg := r.getConfig()
		if cfg.DispatchEnabled {
			result, err := r.service.DispatchDueJobs(ctx, cfg.DispatchBatch)
			if err != nil {
				return err
			}
			if len(result.CreatedRuns) > 0 || len(result.SkippedJobs) > 0 {
				logging.ForContext(ctx).Info("scheduler_dispatch_cycle",
					slog.Int("created_runs", len(result.CreatedRuns)),
					slog.Int("skipped_jobs", len(result.SkippedJobs)),
				)
			}
		}

		wait := cfg.DispatchEvery
		if wait <= 0 {
			wait = time.Second
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
