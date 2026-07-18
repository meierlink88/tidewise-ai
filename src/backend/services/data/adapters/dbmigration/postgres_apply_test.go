package dbmigration

import (
	"context"
	"database/sql"
	"reflect"
	"strings"
	"testing"
)

func TestGooseExecutorApplyStopsAtTargetAndLeavesLaterMigrationPending(t *testing.T) {
	operations := &fakeGooseOperations{
		current: 14,
		migrations: []Migration{
			{Version: "000015", Name: "cleanup"},
			{Version: "000016", Name: "external_identifiers"},
		},
	}
	executor := GooseExecutor{operations: operations}

	applied, err := executor.Apply(context.Background(), "15")
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if got, want := migrationVersions(applied), []string{"000015"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("applied = %v, want %v", got, want)
	}
	if operations.upCalls != 0 || operations.upToCalls != 1 || operations.upToTarget != 15 {
		t.Fatalf("operations = %+v", operations)
	}
	pending, err := executor.Pending(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := migrationVersions(pending), []string{"000016"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("pending = %v, want %v", got, want)
	}
}

func TestGooseExecutorApplyWithoutTargetKeepsExistingAllPendingBehavior(t *testing.T) {
	operations := &fakeGooseOperations{
		current: 14,
		migrations: []Migration{
			{Version: "000015", Name: "cleanup"},
			{Version: "000016", Name: "external_identifiers"},
		},
	}
	executor := GooseExecutor{operations: operations}

	applied, err := executor.Apply(context.Background(), "")
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if got, want := migrationVersions(applied), []string{"000015", "000016"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("applied = %v, want %v", got, want)
	}
	if operations.upCalls != 1 || operations.upToCalls != 0 {
		t.Fatalf("operations = %+v", operations)
	}
}

func TestGooseExecutorApplyRejectsInvalidRollbackAndUnknownTargets(t *testing.T) {
	for _, test := range []struct {
		name    string
		current int64
		target  string
		want    string
	}{
		{name: "invalid", current: 14, target: "not-a-version", want: "invalid target version"},
		{name: "rollback", current: 15, target: "14", want: "behind current version"},
		{name: "rollback with no pending", current: 16, target: "15", want: "behind current version"},
		{name: "unknown jump", current: 14, target: "17", want: "does not match an available migration"},
	} {
		t.Run(test.name, func(t *testing.T) {
			operations := &fakeGooseOperations{current: test.current, migrations: []Migration{{Version: "000015"}, {Version: "000016"}}}
			_, err := (GooseExecutor{operations: operations}).Apply(context.Background(), test.target)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Apply() error = %v, want containing %q", err, test.want)
			}
			if operations.upCalls != 0 || operations.upToCalls != 0 {
				t.Fatalf("invalid target invoked apply: %+v", operations)
			}
		})
	}
}

type fakeGooseOperations struct {
	current    int64
	migrations []Migration
	upCalls    int
	upToCalls  int
	upToTarget int64
}

func (o *fakeGooseOperations) currentVersion(context.Context, *sql.DB) (int64, error) {
	return o.current, nil
}

func (o *fakeGooseOperations) pending(context.Context, *sql.DB, string, int64) ([]Migration, error) {
	result := make([]Migration, 0, len(o.migrations))
	for _, migration := range o.migrations {
		if migrationVersionNumber(migration.Version) > o.current {
			result = append(result, migration)
		}
	}
	return result, nil
}

func (o *fakeGooseOperations) up(context.Context, *sql.DB, string) error {
	o.upCalls++
	for _, migration := range o.migrations {
		if version := migrationVersionNumber(migration.Version); version > o.current {
			o.current = version
		}
	}
	return nil
}

func (o *fakeGooseOperations) upTo(_ context.Context, _ *sql.DB, _ string, target int64) error {
	o.upToCalls++
	o.upToTarget = target
	o.current = target
	return nil
}

func migrationVersionNumber(version string) int64 {
	var result int64
	for _, digit := range version {
		result = result*10 + int64(digit-'0')
	}
	return result
}
