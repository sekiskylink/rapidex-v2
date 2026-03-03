package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"basepro/backend/internal/config"
	"basepro/backend/internal/logging"
	"basepro/backend/internal/migrateutil"
)

func main() {
	if len(os.Args) < 2 {
		logging.L().Error("migrate_usage_error", slog.String("message", "usage: migrate [up|down|create]"))
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "up":
		runUpDown(true)
	case "down":
		runUpDown(false)
	case "create":
		runCreate()
	default:
		logging.L().Error("migrate_unknown_command", slog.String("command", command))
		os.Exit(1)
	}
}

func runUpDown(isUp bool) {
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	configFile := fs.String("config", "", "path to config file")
	if err := fs.Parse(os.Args[2:]); err != nil {
		logging.L().Error("migrate_parse_flags_failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if _, err := config.Load(config.Options{ConfigFile: *configFile}); err != nil {
		logging.L().Error("migrate_load_config_failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	cfg := config.Get()
	if isUp {
		if err := migrateutil.Up(cfg.Database.DSN, "./migrations"); err != nil {
			logging.L().Error("migrate_up_failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
		fmt.Println("migrations applied")
		return
	}

	if err := migrateutil.DownOne(cfg.Database.DSN, "./migrations"); err != nil {
		logging.L().Error("migrate_down_failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	fmt.Println("one migration rolled back")
}

func runCreate() {
	fs := flag.NewFlagSet("create", flag.ExitOnError)
	name := fs.String("name", "", "migration name")
	if err := fs.Parse(os.Args[2:]); err != nil {
		logging.L().Error("migrate_parse_flags_failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	up, down, err := migrateutil.CreatePair("./migrations", *name)
	if err != nil {
		logging.L().Error("migrate_create_failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	fmt.Printf("created %s\n", up)
	fmt.Printf("created %s\n", down)
}
