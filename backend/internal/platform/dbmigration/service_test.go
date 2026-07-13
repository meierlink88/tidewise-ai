package dbmigration

import (
	"context"
	"errors"
	"reflect"
	"strings"
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

func TestServiceForwardsTargetVersionAndReportsOnlyActuallyAppliedMigrations(t *testing.T) {
	executor := &fakeExecutor{
		version:           "14",
		pending:           []Migration{{Version: "000015"}, {Version: "000016"}},
		appliedMigrations: []Migration{{Version: "000015"}},
	}
	service := NewService(executor, &fakeServiceLocker{})

	report, err := service.Check(context.Background(), ServiceOptions{AutoApply: true, TargetVersion: "15"})
	if err != nil {
		t.Fatal(err)
	}
	if executor.targetVersion != "15" {
		t.Fatalf("target version = %q", executor.targetVersion)
	}
	if got, want := migrationVersions(report.Applied), []string{"000015"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("applied = %v, want %v", got, want)
	}
	if got, want := migrationVersions(report.Pending), []string{"000015", "000016"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("pending = %v, want %v", got, want)
	}
	if got, want := migrationVersions(report.Remaining), []string{"000016"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("remaining = %v, want %v", got, want)
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

func TestServiceDoesNotSkipTargetValidationWhenNoMigrationsArePending(t *testing.T) {
	executor := &fakeExecutor{version: "16", applyErr: errors.New("target version 15 is behind current version 16")}
	locker := &fakeServiceLocker{}
	service := NewService(executor, locker)

	_, err := service.Check(context.Background(), ServiceOptions{AutoApply: true, TargetVersion: "15"})
	if err == nil || !strings.Contains(err.Error(), "behind current version") {
		t.Fatalf("error = %v", err)
	}
	if !executor.applied || executor.targetVersion != "15" {
		t.Fatalf("executor = %+v", executor)
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
	targetVersion     string
}

func (e *fakeExecutor) CurrentVersion(context.Context) (string, error) {
	return e.version, e.versionErr
}

func (e *fakeExecutor) Pending(context.Context) ([]Migration, error) {
	return e.pending, e.pendingErr
}

func (e *fakeExecutor) Apply(_ context.Context, targetVersion string) ([]Migration, error) {
	e.applied = true
	e.targetVersion = targetVersion
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
