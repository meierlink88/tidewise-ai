package dbmigration

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRebuildEmptyPostgresSchemaResetsOnlyPublicSchemaAndAppliesTarget58(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectBegin()
	for _, statement := range []string{
		"DROP SCHEMA public CASCADE",
		"CREATE SCHEMA public AUTHORIZATION CURRENT_USER",
		"GRANT USAGE ON SCHEMA public TO public",
	} {
		mock.ExpectExec(regexp.QuoteMeta(statement)).WillReturnResult(sqlmock.NewResult(0, 0))
	}
	mock.ExpectCommit()

	migrations := []Migration{{Version: "000001", Name: "init"}, {Version: "000058", Name: "target"}}
	executor := &fakeExecutor{
		version:           "0",
		pending:           migrations,
		appliedMigrations: migrations,
	}
	locker := &fakeServiceLocker{}
	report, err := rebuildEmptyPostgresSchema(context.Background(), db, executor, locker, "58")
	if err != nil {
		t.Fatal(err)
	}
	if !locker.locked || !locker.unlocked {
		t.Fatalf("schema rebuild did not hold the migration lock: %+v", locker)
	}
	if executor.targetVersion != "58" || report.CurrentVersion != "58" || len(report.Remaining) != 0 {
		t.Fatalf("unexpected schema rebuild result: executor=%+v report=%+v", executor, report)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestValidatePostgresServiceOptionsRejectsIllegalRebuildStates(t *testing.T) {
	for _, options := range []ServiceOptions{
		{RebuildEmptySchema: true},
		{AutoApply: true, RebuildEmptySchema: true},
		{AutoApply: true, TargetVersion: "57", RebuildEmptySchema: true},
	} {
		if err := validatePostgresServiceOptions(options); err == nil {
			t.Fatalf("illegal rebuild options unexpectedly passed: %+v", options)
		}
	}
	if err := validatePostgresServiceOptions(ServiceOptions{AutoApply: true, TargetVersion: "58", RebuildEmptySchema: true}); err != nil {
		t.Fatalf("valid rebuild options failed: %v", err)
	}
}
