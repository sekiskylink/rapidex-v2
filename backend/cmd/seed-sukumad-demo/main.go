package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"basepro/backend/internal/config"
	"basepro/backend/internal/db"
	"basepro/backend/internal/logging"
	asyncjobs "basepro/backend/internal/sukumad/async"
	"basepro/backend/internal/sukumad/delivery"
	"basepro/backend/internal/sukumad/devseed"
	"basepro/backend/internal/sukumad/request"
	"basepro/backend/internal/sukumad/server"
)

func main() {
	if err := run(); err != nil {
		logging.L().Error("seed_sukumad_demo_failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	var configFile string
	flags := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flags.StringVar(&configFile, "config", "", "path to config file")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	if _, err := config.Load(config.Options{
		ConfigFile: configFile,
		Watch:      false,
	}); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	cfg := config.Get()
	logging.ApplyConfig(logging.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
	})

	database, err := db.Open(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() {
		_ = database.Close()
	}()

	serverService := server.NewService(server.NewRepository(database))
	requestService := request.NewService(request.NewRepository(database))
	deliveryService := delivery.NewService(delivery.NewRepository(database)).
		WithRequestStatusUpdater(requestService).
		WithTargetUpdater(requestService)
	requestService.WithDeliveryService(deliveryService)
	asyncService := asyncjobs.NewService(asyncjobs.NewRepository(database)).
		WithReconciliation(deliveryService, requestService)

	seeder := devseed.NewService(
		devseed.NewRepository(database),
		serverService,
		requestService,
		deliveryService,
		asyncService,
	)

	summary, err := seeder.Seed(ctx)
	if err != nil {
		return err
	}

	logging.L().Info(
		"seed_sukumad_demo_completed",
		slog.String("seed_tag", summary.SeedTag),
		slog.Int("servers_seeded", summary.ServersSeeded),
		slog.Int("requests_seeded", summary.RequestsSeeded),
	)
	return nil
}
