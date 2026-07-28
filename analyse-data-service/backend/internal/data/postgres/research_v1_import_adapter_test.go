package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestResearchThemeImportTransactionIsAtomic(t *testing.T) {
	for _, test := range []struct {
		name      string
		callback  error
		expectEnd func(sqlmock.Sqlmock)
	}{
		{
			name:      "commit",
			expectEnd: func(mock sqlmock.Sqlmock) { mock.ExpectCommit() },
		},
		{
			name:      "rollback",
			callback:  errors.New("validation failed"),
			expectEnd: func(mock sqlmock.Sqlmock) { mock.ExpectRollback() },
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			mock.ExpectBegin()
			test.expectEnd(mock)
			repository := repository{db: db}

			err = repository.InResearchThemeImportTransaction(context.Background(), func(ResearchThemeImportTransaction) error {
				return test.callback
			})
			if !errors.Is(err, test.callback) {
				t.Fatalf("transaction error = %v, want %v", err, test.callback)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestResearchReasoningTreeImportTransactionRollsBackChildFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectRollback()
	repository := repository{db: db}
	childFailure := errors.New("node insert failed")

	err = repository.InResearchReasoningTreeImportTransaction(context.Background(), func(ResearchReasoningTreeImportTransaction) error {
		return childFailure
	})
	if !errors.Is(err, childFailure) {
		t.Fatalf("transaction error = %v, want %v", err, childFailure)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
