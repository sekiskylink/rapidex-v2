package migrate

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	gomigrate "github.com/golang-migrate/migrate/v4"
	"github.com/jmoiron/sqlx"
)

type fakeLocker struct {
	acquireFn    func(ctx context.Context, db *sqlx.DB, lockID int64) error
	releaseFn    func(ctx context.Context, db *sqlx.DB, lockID int64) error
	acquireCalls int
	releaseCalls int
}

func (f *fakeLocker) Acquire(ctx context.Context, db *sqlx.DB, lockID int64) error {
	f.acquireCalls++
	if f.acquireFn != nil {
		return f.acquireFn(ctx, db, lockID)
	}
	return nil
}

func (f *fakeLocker) Release(ctx context.Context, db *sqlx.DB, lockID int64) error {
	f.releaseCalls++
	if f.releaseFn != nil {
		return f.releaseFn(ctx, db, lockID)
	}
	return nil
}

type fakeUpRunner struct {
	upFn    func(databaseDSN string) error
	upCalls int
}

func (f *fakeUpRunner) Up(databaseDSN string) error {
	f.upCalls++
	if f.upFn != nil {
		return f.upFn(databaseDSN)
	}
	return nil
}

func TestRunSkipsWhenAutoMigrateDisabled(t *testing.T) {
	locker := &fakeLocker{}
	up := &fakeUpRunner{}
	runner := &Runner{locker: locker, up: up, lockID: 1}

	err := runner.Run(context.Background(), Config{
		AutoMigrate: false,
		LockTimeout: 30 * time.Second,
	}, nil, "postgres://test")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if locker.acquireCalls != 0 {
		t.Fatalf("expected no lock acquire, got %d", locker.acquireCalls)
	}
	if up.upCalls != 0 {
		t.Fatalf("expected no migrations run, got %d", up.upCalls)
	}
}

func TestRunTreatsErrNoChangeAsSuccess(t *testing.T) {
	locker := &fakeLocker{}
	up := &fakeUpRunner{
		upFn: func(databaseDSN string) error {
			return gomigrate.ErrNoChange
		},
	}
	runner := &Runner{locker: locker, up: up, lockID: 1}

	err := runner.Run(context.Background(), Config{
		AutoMigrate: true,
		LockTimeout: 30 * time.Second,
	}, nil, "postgres://test")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if locker.acquireCalls != 1 {
		t.Fatalf("expected lock acquire once, got %d", locker.acquireCalls)
	}
	if locker.releaseCalls != 1 {
		t.Fatalf("expected lock release once, got %d", locker.releaseCalls)
	}
	if up.upCalls != 1 {
		t.Fatalf("expected migrations run once, got %d", up.upCalls)
	}
}

func TestRunLockAcquisitionRespectsTimeout(t *testing.T) {
	locker := &fakeLocker{
		acquireFn: func(ctx context.Context, db *sqlx.DB, lockID int64) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}
	up := &fakeUpRunner{}
	runner := &Runner{locker: locker, up: up, lockID: 1}

	err := runner.Run(context.Background(), Config{
		AutoMigrate: true,
		LockTimeout: 25 * time.Millisecond,
	}, nil, "postgres://test")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
	if !strings.Contains(err.Error(), "acquire migration advisory lock") {
		t.Fatalf("expected acquire advisory lock context in error, got %v", err)
	}
	if up.upCalls != 0 {
		t.Fatalf("expected no migration run on lock timeout, got %d", up.upCalls)
	}
}
