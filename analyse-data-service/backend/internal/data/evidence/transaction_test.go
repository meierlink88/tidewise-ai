package evidence

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	evidencebiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/evidence"
)

func TestEvidencePublicationTransactionRollsBackOnPanic(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectBegin()
	mock.ExpectRollback()

	deferred := false
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	func() {
		defer func() {
			if recovered := recover(); recovered != "test panic" {
				t.Fatalf("recovered panic = %#v", recovered)
			}
			deferred = true
		}()
		_ = store.InTransaction(context.Background(), func(evidencebiz.Transaction) error {
			panic("test panic")
		})
	}()
	if !deferred {
		t.Fatal("transaction panic did not reach caller")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEvidenceTransactionRollsBackWhenExecutionBudgetExpires(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectBegin()
	mock.ExpectRollback()
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	err = store.InTransaction(ctx, func(evidencebiz.Transaction) error {
		return context.DeadlineExceeded
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("InTransaction() error = %v, want deadline exceeded", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
