package postgres

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/evidencepublication"
)

func TestEvidencePublicationTransactionRollsBackOnPanic(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectBegin()
	mock.ExpectRollback()
	mock.ExpectClose()

	deferred := false
	func() {
		defer func() {
			if recovered := recover(); recovered != "test panic" {
				t.Fatalf("recovered panic = %#v", recovered)
			}
			deferred = true
		}()
		_ = newRepository(db).InTransaction(context.Background(), func(evidencepublication.Transaction) error {
			panic("test panic")
		})
	}()
	if !deferred {
		t.Fatal("transaction panic did not reach caller")
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close SQL mock: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
