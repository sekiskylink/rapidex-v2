package migrate

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"basepro/backend/internal/logging"
	gomigrate "github.com/golang-migrate/migrate/v4"
	"github.com/jmoiron/sqlx"
)

const defaultAdvisoryLockID int64 = 987654321

type Config struct {
	AutoMigrate bool
	LockTimeout time.Duration
}

type Locker interface {
	Acquire(ctx context.Context, db *sqlx.DB, lockID int64) error
	Release(ctx context.Context, db *sqlx.DB, lockID int64) error
}

type UpRunner interface {
	Up(databaseDSN string) error
}

type Runner struct {
	locker Locker
	up     UpRunner
	lockID int64
}

func NewRunner() *Runner {
	return &Runner{
		locker: NewPostgresLocker(100 * time.Millisecond),
		up:     golangMigrateRunner{},
		lockID: defaultAdvisoryLockID,
	}
}

func (r *Runner) Run(ctx context.Context, cfg Config, db *sqlx.DB, databaseDSN string) error {
	if !cfg.AutoMigrate {
		logging.L().Info("startup_migrations_skipped")
		return nil
	}
	if cfg.LockTimeout <= 0 {
		return errors.New("auto-migrate lock timeout must be > 0")
	}

	lockCtx, cancel := context.WithTimeout(ctx, cfg.LockTimeout)
	defer cancel()

	logging.L().Info("startup_migrations_lock_acquire")
	if err := r.locker.Acquire(lockCtx, db, r.lockID); err != nil {
		return fmt.Errorf("acquire migration advisory lock: %w", err)
	}
	defer func() {
		releaseCtx, releaseCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer releaseCancel()
		if err := r.locker.Release(releaseCtx, db, r.lockID); err != nil {
			logging.L().Warn("startup_migrations_lock_release_failed", slog.String("error", err.Error()))
		}
	}()

	logging.L().Info("startup_migrations_running")
	if err := r.up.Up(databaseDSN); err != nil {
		if errors.Is(err, gomigrate.ErrNoChange) {
			logging.L().Info("startup_migrations_no_change")
			return nil
		}
		return fmt.Errorf("run migrations: %w", err)
	}

	logging.L().Info("startup_migrations_complete")
	return nil
}
