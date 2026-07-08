package dbmigration

import (
	"context"
	"errors"
	"testing"
)

func TestServiceCheckOnlyReportsPending(t *testing.T) {
	executor := &fakeExecutor{
		version: "0",
		pending: []Migration{
			{Version: "000001", Name: "init"},
		},
	}
	locker := &fakeServiceLocker{}
	service := NewService(executor, locker)

	report, err := service.Check(context.Background(), ServiceOptions{AutoApply: false})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if report.CurrentVersion != "0" {
		t.Fatalf("CurrentVersion = %q, want 0", report.CurrentVersion)
	}
	if len(report.Pending) != 1 {
		t.Fatalf("Pending length = %d, want 1", len(report.Pending))
	}
	if len(report.Applied) != 0 {
		t.Fatalf("Applied length = %d, want 0", len(report.Applied))
	}
	if locker.locked || locker.unlocked {
		t.Fatal("check-only must not acquire migration lock")
	}
	if executor.applied {
		t.Fatal("check-only must not apply migrations")
	}
}

func TestServiceAutoApplyUsesLock(t *testing.T) {
	executor := &fakeExecutor{
		version: "0",
		pending: []Migration{
			{Version: "000001", Name: "init"},
		},
		appliedMigrations: []Migration{
			{Version: "000001", Name: "init"},
		},
	}
	locker := &fakeServiceLocker{}
	service := NewService(executor, locker)

	report, err := service.Check(context.Background(), ServiceOptions{AutoApply: true})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if !locker.locked || !locker.unlocked {
		t.Fatal("auto apply must lock and unlock migration execution")
	}
	if !executor.applied {
		t.Fatal("auto apply must apply pending migrations")
	}
	if len(report.Applied) != 1 {
		t.Fatalf("Applied length = %d, want 1", len(report.Applied))
	}
}

func TestServiceSkipsLockWhenNoPending(t *testing.T) {
	executor := &fakeExecutor{version: "000001"}
	locker := &fakeServiceLocker{}
	service := NewService(executor, locker)

	report, err := service.Check(context.Background(), ServiceOptions{AutoApply: true})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if len(report.Pending) != 0 || len(report.Applied) != 0 {
		t.Fatalf("report = %+v, want no pending or applied migrations", report)
	}
	if locker.locked || locker.unlocked {
		t.Fatal("no pending migrations must not lock")
	}
}

func TestServiceReturnsLockError(t *testing.T) {
	lockErr := errors.New("lock failed")
	executor := &fakeExecutor{
		version: "0",
		pending: []Migration{{Version: "000001", Name: "init"}},
	}
	service := NewService(executor, &fakeServiceLocker{lockErr: lockErr})

	if _, err := service.Check(context.Background(), ServiceOptions{AutoApply: true}); !errors.Is(err, lockErr) {
		t.Fatalf("Check() error = %v, want lock error", err)
	}
	if executor.applied {
		t.Fatal("service must not apply migrations when lock fails")
	}
}

func TestServiceReturnsApplyErrorAndUnlocks(t *testing.T) {
	applyErr := errors.New("apply failed")
	executor := &fakeExecutor{
		version:  "0",
		pending:  []Migration{{Version: "000001", Name: "init"}},
		applyErr: applyErr,
	}
	locker := &fakeServiceLocker{}
	service := NewService(executor, locker)

	if _, err := service.Check(context.Background(), ServiceOptions{AutoApply: true}); !errors.Is(err, applyErr) {
		t.Fatalf("Check() error = %v, want apply error", err)
	}
	if !locker.unlocked {
		t.Fatal("service must unlock after apply failure")
	}
}

type fakeExecutor struct {
	version           string
	pending           []Migration
	appliedMigrations []Migration
	versionErr        error
	pendingErr        error
	applyErr          error
	applied           bool
}

func (e *fakeExecutor) CurrentVersion(context.Context) (string, error) {
	return e.version, e.versionErr
}

func (e *fakeExecutor) Pending(context.Context) ([]Migration, error) {
	return e.pending, e.pendingErr
}

func (e *fakeExecutor) Apply(context.Context) ([]Migration, error) {
	e.applied = true
	return e.appliedMigrations, e.applyErr
}

type fakeServiceLocker struct {
	locked    bool
	unlocked  bool
	lockErr   error
	unlockErr error
}

func (l *fakeServiceLocker) Lock(context.Context) error {
	l.locked = true
	return l.lockErr
}

func (l *fakeServiceLocker) Unlock(context.Context) error {
	l.unlocked = true
	return l.unlockErr
}
