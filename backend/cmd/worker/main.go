package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"basepro/backend/internal/audit"
	"basepro/backend/internal/config"
	"basepro/backend/internal/db"
	"basepro/backend/internal/logging"
	"basepro/backend/internal/migrate"
	outboundratelimit "basepro/backend/internal/ratelimit"
	asyncjobs "basepro/backend/internal/sukumad/async"
	"basepro/backend/internal/sukumad/delivery"
	"basepro/backend/internal/sukumad/dhis2"
	"basepro/backend/internal/sukumad/ingest"
	"basepro/backend/internal/sukumad/observability"
	requests "basepro/backend/internal/sukumad/request"
	"basepro/backend/internal/sukumad/retention"
	"basepro/backend/internal/sukumad/server"
	"basepro/backend/internal/sukumad/worker"
)

func main() {
	if err := run(); err != nil {
		logging.L().Error("worker_start_failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var configFile string
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.StringVar(&configFile, "config", "", "path to config file")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	_, err := config.Load(config.Options{
		ConfigFile: configFile,
		Watch:      true,
	})
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	cfg := config.Get()
	logging.ApplyConfig(logging.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
	})
	config.RegisterOnChange(func(next config.Config) {
		logging.ApplyConfig(logging.Config{
			Level:  next.Logging.Level,
			Format: next.Logging.Format,
		})
	})

	database, err := db.Open(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := database.Close(); closeErr != nil {
			logging.L().Warn("worker_database_close_failed", slog.String("error", closeErr.Error()))
		}
	}()

	startupMigrator := migrate.NewRunner()
	if err := startupMigrator.Run(ctx, migrate.Config{
		AutoMigrate: cfg.Database.AutoMigrate,
		LockTimeout: time.Duration(cfg.Database.AutoMigrateLockTimeoutSeconds) * time.Second,
	}, database, cfg.Database.DSN); err != nil {
		return fmt.Errorf("startup migrations: %w", err)
	}

	auditService := audit.NewService(audit.NewSQLRepository(database))
	outboundLimiter := outboundratelimit.NewRegistry(func(destinationKey string) outboundratelimit.Policy {
		nextCfg := config.Get()
		policy := outboundratelimit.Policy{
			RequestsPerSecond: nextCfg.Sukumad.RateLimit.Default.RequestsPerSecond,
			Burst:             nextCfg.Sukumad.RateLimit.Default.Burst,
		}
		if destination, ok := nextCfg.Sukumad.RateLimit.Destinations[destinationKey]; ok {
			policy.RequestsPerSecond = destination.RequestsPerSecond
			policy.Burst = destination.Burst
		}
		return policy
	})
	sukumadDHIS2Service := dhis2.NewService(nil, outboundLimiter).WithOutboundLoggingConfig(func() dhis2.OutboundLoggingConfig {
		nextCfg := config.Get().Sukumad.Workers.OutboundLogging
		return dhis2.OutboundLoggingConfig{
			Enabled:          nextCfg.Enabled,
			BodyPreviewBytes: nextCfg.BodyPreviewBytes,
		}
	})
	sukumadServerService := server.NewService(server.NewRepository(database), auditService)
	sukumadRequestService := requests.NewService(requests.NewRepository(database), auditService)
	sukumadIngestService := ingest.NewService(ingest.NewRepository(database), sukumadRequestService, auditService)
	sukumadDeliveryService := delivery.NewService(delivery.NewRepository(database), auditService).
		WithDispatcher(sukumadDHIS2Service).
		WithRequestStatusUpdater(sukumadRequestService).
		WithTargetUpdater(sukumadRequestService)
	sukumadAsyncService := asyncjobs.NewService(asyncjobs.NewRepository(database), auditService)
	sukumadWorkerService := worker.NewService(worker.NewRepository(database), auditService)
	sukumadRetentionService := retention.NewService(retention.NewRepository(database), auditService)
	sukumadObservabilityService := observability.NewService(observability.NewRepository(database, sukumadWorkerService, nil))
	sukumadDeliveryService.WithAsyncService(sukumadAsyncService).WithEventWriter(sukumadObservabilityService)
	sukumadRequestService.WithDeliveryService(sukumadDeliveryService).WithEventWriter(sukumadObservabilityService)
	sukumadIngestService.WithEventWriter(sukumadObservabilityService)
	sukumadAsyncService.WithReconciliation(sukumadDeliveryService, sukumadRequestService).WithEventWriter(sukumadObservabilityService)
	sukumadWorkerService.WithEventWriter(sukumadObservabilityService)
	sukumadRetentionService.WithEventWriter(sukumadObservabilityService)

	executor := worker.NewDeliveryExecutor(
		delivery.NewRepository(database),
		sukumadRequestService,
		sukumadServerService,
		sukumadDeliveryService,
	).WithEventWriter(sukumadObservabilityService)

	workerCfg := config.Get().Sukumad.Workers
	recoveryExec := worker.Execution{
		RunID:    0,
		AddCount: func(string, int) {},
		SetMeta:  func(string, any) {},
	}
	if err := executor.RecoverStaleRunning(ctx, recoveryExec, time.Duration(workerCfg.Recovery.StaleDeliveryAfterSeconds)*time.Second); err != nil {
		return fmt.Errorf("recover stale running deliveries: %w", err)
	}
	if err := sukumadAsyncService.ReconcileTerminalTasks(ctx, workerCfg.Poll.BatchSize); err != nil {
		return fmt.Errorf("reconcile terminal async tasks: %w", err)
	}

	sendDef := worker.NewSendDefinition(executor, workerCfg.Send.BatchSize)
	sendDef.Interval = time.Duration(workerCfg.Send.IntervalSeconds) * time.Second
	sendDef.HeartbeatInterval = time.Duration(workerCfg.HeartbeatSeconds) * time.Second

	retryDef := worker.NewRetryDefinition(executor, workerCfg.Retry.BatchSize)
	retryDef.Interval = time.Duration(workerCfg.Retry.IntervalSeconds) * time.Second
	retryDef.HeartbeatInterval = time.Duration(workerCfg.HeartbeatSeconds) * time.Second

	pollDef := worker.NewPollDefinition(
		sukumadAsyncService,
		sukumadDHIS2Service,
		workerCfg.Poll.BatchSize,
		time.Duration(workerCfg.Poll.ClaimTimeoutSeconds)*time.Second,
	)
	pollDef.Interval = time.Duration(workerCfg.Poll.IntervalSeconds) * time.Second
	pollDef.HeartbeatInterval = time.Duration(workerCfg.HeartbeatSeconds) * time.Second

	retentionDef := worker.NewRetentionDefinition(
		sukumadRetentionService,
		cfg.Sukumad.Retention.Enabled,
		func() time.Time {
			nextCfg := config.Get()
			return time.Now().UTC().Add(-time.Duration(nextCfg.Sukumad.Retention.TerminalAgeDays) * 24 * time.Hour)
		},
		cfg.Sukumad.Retention.BatchSize,
		cfg.Sukumad.Retention.DryRun,
	)
	retentionDef.Interval = time.Duration(workerCfg.Retention.IntervalSeconds) * time.Second
	retentionDef.HeartbeatInterval = time.Duration(workerCfg.HeartbeatSeconds) * time.Second

	ingestRuntime := ingest.NewRuntime(sukumadIngestService, func() ingest.RuntimeConfig {
		nextCfg := config.Get().Sukumad.Ingest.Directory
		return ingest.RuntimeConfig{
			Enabled:               nextCfg.Enabled,
			InboxPath:             nextCfg.InboxPath,
			ProcessingPath:        nextCfg.ProcessingPath,
			ProcessedPath:         nextCfg.ProcessedPath,
			FailedPath:            nextCfg.FailedPath,
			AllowedExtensions:     append([]string{}, nextCfg.AllowedExtensions...),
			DefaultSourceSystem:   nextCfg.DefaultSourceSystem,
			RequireIdempotencyKey: nextCfg.RequireIdempotencyKey,
			Debounce:              time.Duration(nextCfg.DebounceMilliseconds) * time.Millisecond,
			RetryDelay:            time.Duration(nextCfg.RetryDelaySeconds) * time.Second,
			ClaimTimeout:          time.Duration(nextCfg.ClaimTimeoutSeconds) * time.Second,
			ScanInterval:          time.Duration(nextCfg.ScanIntervalSeconds) * time.Second,
			BatchSize:             nextCfg.BatchSize,
		}
	})
	ingestDef := worker.NewIngestDefinition(ingestRuntime.Run, config.Get().Sukumad.Ingest.Directory.BatchSize)
	ingestDef.Interval = time.Second
	ingestDef.HeartbeatInterval = time.Duration(workerCfg.HeartbeatSeconds) * time.Second

	manager := worker.NewManager(sukumadWorkerService, ingestDef, sendDef, retryDef, pollDef, retentionDef)
	errCh := manager.Start(ctx)

	for {
		select {
		case err, ok := <-errCh:
			if !ok {
				return nil
			}
			if err != nil && !errors.Is(err, context.Canceled) {
				stop()
				return err
			}
		case <-ctx.Done():
			for err := range errCh {
				if err != nil && !errors.Is(err, context.Canceled) {
					return err
				}
			}
			return nil
		}
	}
}
