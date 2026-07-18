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
	locker := &fakeServiceLocker{}
	executor := &auditedExecutor{
		locker:          locker,
		versions:        []string{"14", "15"},
		pendingStates:   [][]Migration{{{Version: "000015"}, {Version: "000016"}}, {{Version: "000016"}}},
		predictedResult: []Migration{{Version: "000015"}, {Version: "000016"}},
	}
	service := NewService(executor, locker)

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
	if report.CurrentVersion != "15" {
		t.Fatalf("current version = %q, want post-apply 15", report.CurrentVersion)
	}
}

func TestServiceAutoApplyWithoutTargetDerivesAppliedFromLockedSnapshots(t *testing.T) {
	locker := &fakeServiceLocker{}
	executor := &auditedExecutor{
		locker:        locker,
		versions:      []string{"14", "16"},
		pendingStates: [][]Migration{{{Version: "000015"}, {Version: "000016"}}, nil},
	}
	report, err := NewService(executor, locker).Check(context.Background(), ServiceOptions{AutoApply: true})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := migrationVersions(report.Applied), []string{"000015", "000016"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("applied = %v, want %v", got, want)
	}
	if len(report.Remaining) != 0 || report.CurrentVersion != "16" {
		t.Fatalf("report = %+v", report)
	}
}

func TestServiceRejectsSuccessfulApplyThatDidNotReachTarget(t *testing.T) {
	locker := &fakeServiceLocker{}
	executor := &auditedExecutor{
		locker:        locker,
		versions:      []string{"14", "14"},
		pendingStates: [][]Migration{{{Version: "000015"}, {Version: "000016"}}, {{Version: "000015"}, {Version: "000016"}}},
	}
	_, err := NewService(executor, locker).Check(context.Background(), ServiceOptions{AutoApply: true, TargetVersion: "15"})
	if err == nil || !strings.Contains(err.Error(), "did not reach target") {
		t.Fatalf("Check() error = %v", err)
	}
}

func TestServiceDoesNotClaimMigrationAbovePostApplyVersion(t *testing.T) {
	locker := &fakeServiceLocker{}
	executor := &auditedExecutor{
		locker:          locker,
		versions:        []string{"14", "15"},
		pendingStates:   [][]Migration{{{Version: "000015"}, {Version: "000016"}}, nil},
		predictedResult: []Migration{{Version: "000015"}, {Version: "000016"}},
	}
	report, err := NewService(executor, locker).Check(context.Background(), ServiceOptions{AutoApply: true, TargetVersion: "15"})
	if err == nil || !strings.Contains(err.Error(), "post-apply migration state is inconsistent") {
		t.Fatalf("Check() error = %v", err)
	}
	if got, want := migrationVersions(report.Applied), []string{"000015"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("applied = %v, want %v", got, want)
	}
}

func TestServiceReturnsUnlockError(t *testing.T) {
	unlockErr := errors.New("unlock failed")
	locker := &fakeServiceLocker{unlockErr: unlockErr}
	executor := &auditedExecutor{locker: locker, versions: []string{"14", "15"}, pendingStates: [][]Migration{{{Version: "000015"}}, nil}}
	_, err := NewService(executor, locker).Check(context.Background(), ServiceOptions{AutoApply: true})
	if !errors.Is(err, unlockErr) {
		t.Fatalf("Check() error = %v, want unlock error", err)
	}
}

func TestServiceLocksBeforeConfirmingAutoApplyHasNoPendingMigrations(t *testing.T) {
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
	if !locker.locked || !locker.unlocked {
		t.Fatal("auto apply must establish a locked state snapshot before reporting no pending migrations")
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
	if e.applyErr == nil {
		appliedVersions := map[string]struct{}{}
		for _, migration := range e.appliedMigrations {
			appliedVersions[migration.Version] = struct{}{}
			e.version = strings.TrimLeft(migration.Version, "0")
			if e.version == "" {
				e.version = "0"
			}
		}
		remaining := make([]Migration, 0, len(e.pending))
		for _, migration := range e.pending {
			if _, applied := appliedVersions[migration.Version]; !applied {
				remaining = append(remaining, migration)
			}
		}
		e.pending = remaining
	}
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

type auditedExecutor struct {
	locker          *fakeServiceLocker
	versions        []string
	pendingStates   [][]Migration
	predictedResult []Migration
	versionCalls    int
	pendingCalls    int
	applied         bool
	targetVersion   string
}

func (e *auditedExecutor) CurrentVersion(context.Context) (string, error) {
	if !e.locker.locked {
		return "", errors.New("current version read before migration lock")
	}
	index := e.versionCalls
	if index >= len(e.versions) {
		index = len(e.versions) - 1
	}
	e.versionCalls++
	return e.versions[index], nil
}

func (e *auditedExecutor) Pending(context.Context) ([]Migration, error) {
	if !e.locker.locked {
		return nil, errors.New("pending migrations read before migration lock")
	}
	index := e.pendingCalls
	if index >= len(e.pendingStates) {
		index = len(e.pendingStates) - 1
	}
	e.pendingCalls++
	return append([]Migration(nil), e.pendingStates[index]...), nil
}

func (e *auditedExecutor) Apply(_ context.Context, targetVersion string) ([]Migration, error) {
	e.applied = true
	e.targetVersion = targetVersion
	return append([]Migration(nil), e.predictedResult...), nil
}
