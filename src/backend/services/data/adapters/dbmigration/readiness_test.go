package dbmigration

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRequirePostgresReadyReadOnlyUsesExistingLedgerWithoutGooseMutation(t *testing.T) {
	directory := t.TempDir()
	for _, name := range []string{"000001_first.sql", "000002_second.sql"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte("-- migration"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(readOnlyLedgerExistsSQL)).WillReturnRows(sqlmock.NewRows([]string{"to_regclass"}).AddRow("goose_db_version"))
	mock.ExpectQuery(regexp.QuoteMeta(readOnlyLedgerRowsSQL)).WillReturnRows(sqlmock.NewRows([]string{"version_id", "is_applied"}).
		AddRow(int64(0), true).
		AddRow(int64(1), true).
		AddRow(int64(2), true))
	mock.ExpectCommit()

	report, err := RequirePostgresReadyReadOnly(context.Background(), db, directory)
	if err != nil {
		t.Fatal(err)
	}
	if report.CurrentVersion != "000002" || len(report.Pending) != 0 || len(report.Remaining) != 0 {
		t.Fatalf("read-only readiness report = %#v", report)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRequirePostgresReadyReadOnlyFailsClosedForMissingLedgerOrPendingMigration(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "000001_first.sql"), []byte("-- migration"), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Run("missing ledger", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(readOnlyLedgerExistsSQL)).WillReturnRows(sqlmock.NewRows([]string{"to_regclass"}).AddRow(nil))
		mock.ExpectRollback()
		if _, err := RequirePostgresReadyReadOnly(context.Background(), db, directory); err == nil || !strings.Contains(err.Error(), "ledger") {
			t.Fatalf("missing-ledger error = %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("pending", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(readOnlyLedgerExistsSQL)).WillReturnRows(sqlmock.NewRows([]string{"to_regclass"}).AddRow("goose_db_version"))
		mock.ExpectQuery(regexp.QuoteMeta(readOnlyLedgerRowsSQL)).WillReturnRows(sqlmock.NewRows([]string{"version_id", "is_applied"}).AddRow(int64(0), true))
		mock.ExpectRollback()
		report, err := RequirePostgresReadyReadOnly(context.Background(), db, directory)
		if err == nil || !strings.Contains(err.Error(), "pending") || len(report.Pending) != 1 || report.Pending[0].Version != "000001" {
			t.Fatalf("pending report=%#v error=%v", report, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}
