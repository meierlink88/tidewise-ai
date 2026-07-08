package dbmigration

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestRunnerReportsNoPendingMigrations(t *testing.T) {
	source := fakeSource{
		migrations: []Migration{
			{Version: "000001", Name: "000001_init.sql"},
		},
	}
	store := &fakeStore{
		appliedVersions: map[string]bool{"000001": true},
	}
	runner := NewRunner(source, store, fakeLocker{})

	report, err := runner.Check(context.Background(), RunnerOptions{AutoApply: true})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if len(report.Pending) != 0 {
		t.Fatalf("pending migrations = %v, want none", report.Pending)
	}
	if len(store.applied) != 0 {
		t.Fatalf("applied migrations = %v, want none", store.applied)
	}
}

func TestRunnerDoesNotApplyPendingMigrationsWhenAutoApplyDisabled(t *testing.T) {
	source := fakeSource{
		migrations: []Migration{
			{Version: "000001", Name: "000001_init.sql"},
			{Version: "000002", Name: "000002_next.sql"},
		},
	}
	store := &fakeStore{
		appliedVersions: map[string]bool{"000001": true},
	}
	locker := &recordingLocker{}
	runner := NewRunner(source, store, locker)

	report, err := runner.Check(context.Background(), RunnerOptions{AutoApply: false})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if got, want := migrationVersions(report.Pending), []string{"000002"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("pending versions = %v, want %v", got, want)
	}
	if len(store.applied) != 0 {
		t.Fatalf("applied migrations = %v, want none", store.applied)
	}
	if locker.locked {
		t.Fatal("locker should not be acquired when auto apply is disabled")
	}
}

func TestRunnerAppliesPendingMigrationsWithLock(t *testing.T) {
	source := fakeSource{
		migrations: []Migration{
			{Version: "000001", Name: "000001_init.sql"},
			{Version: "000002", Name: "000002_next.sql"},
		},
	}
	store := &fakeStore{
		appliedVersions: map[string]bool{"000001": true},
	}
	locker := &recordingLocker{}
	runner := NewRunner(source, store, locker)

	report, err := runner.Check(context.Background(), RunnerOptions{AutoApply: true})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if got, want := migrationVersions(report.Applied), []string{"000002"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("applied versions = %v, want %v", got, want)
	}
	if got, want := store.applied, []string{"000002"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("store applied versions = %v, want %v", got, want)
	}
	if !locker.locked || !locker.unlocked {
		t.Fatalf("lock lifecycle locked=%v unlocked=%v, want both true", locker.locked, locker.unlocked)
	}
}

func TestRunnerReleasesLockWhenApplyFails(t *testing.T) {
	expectedErr := errors.New("apply failed")
	source := fakeSource{
		migrations: []Migration{
			{Version: "000001", Name: "000001_init.sql"},
		},
	}
	store := &fakeStore{applyErr: expectedErr}
	locker := &recordingLocker{}
	runner := NewRunner(source, store, locker)

	if _, err := runner.Check(context.Background(), RunnerOptions{AutoApply: true}); !errors.Is(err, expectedErr) {
		t.Fatalf("Check() error = %v, want %v", err, expectedErr)
	}
	if !locker.unlocked {
		t.Fatal("lock must be released when apply fails")
	}
}

func TestRunnerReturnsLockErrorBeforeApplying(t *testing.T) {
	expectedErr := errors.New("lock failed")
	source := fakeSource{
		migrations: []Migration{
			{Version: "000001", Name: "000001_init.sql"},
		},
	}
	store := &fakeStore{}
	locker := &recordingLocker{lockErr: expectedErr}
	runner := NewRunner(source, store, locker)

	if _, err := runner.Check(context.Background(), RunnerOptions{AutoApply: true}); !errors.Is(err, expectedErr) {
		t.Fatalf("Check() error = %v, want %v", err, expectedErr)
	}
	if len(store.applied) != 0 {
		t.Fatalf("applied migrations = %v, want none", store.applied)
	}
}

type fakeSource struct {
	migrations []Migration
	err        error
}

func (s fakeSource) ListMigrations(context.Context) ([]Migration, error) {
	return s.migrations, s.err
}

type fakeStore struct {
	appliedVersions map[string]bool
	applied         []string
	applyErr        error
}

func (s *fakeStore) AppliedVersions(context.Context) (map[string]bool, error) {
	versions := map[string]bool{}
	for version, applied := range s.appliedVersions {
		versions[version] = applied
	}
	return versions, nil
}

func (s *fakeStore) Apply(_ context.Context, migration Migration) error {
	if s.applyErr != nil {
		return s.applyErr
	}
	s.applied = append(s.applied, migration.Version)
	return nil
}

type fakeLocker struct{}

func (fakeLocker) Lock(context.Context) error {
	return nil
}

func (fakeLocker) Unlock(context.Context) error {
	return nil
}

type recordingLocker struct {
	locked   bool
	unlocked bool
	lockErr  error
}

func (l *recordingLocker) Lock(context.Context) error {
	if l.lockErr != nil {
		return l.lockErr
	}
	l.locked = true
	return nil
}

func (l *recordingLocker) Unlock(context.Context) error {
	l.unlocked = true
	return nil
}

func migrationVersions(migrations []Migration) []string {
	versions := make([]string, 0, len(migrations))
	for _, migration := range migrations {
		versions = append(versions, migration.Version)
	}
	return versions
}
