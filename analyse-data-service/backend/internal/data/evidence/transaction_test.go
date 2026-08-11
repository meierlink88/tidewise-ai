package evidence

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	evidencebiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/evidence"
)

func TestEvidenceTransactionRejectsInvalidPersistedRawEvidence(t *testing.T) {
	const rawEvidenceID = "RAW_persisted_000000000000000"
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectBegin()
	mock.ExpectQuery("FROM raw_evidences").
		WithArgs(rawEvidenceID).
		WillReturnRows(sqlmock.NewRows([]string{
			"raw_evidence_id", "source_id", "source_name", "source_level", "source_url", "is_original",
			"quoted_source_id", "quoted_source_name", "title", "raw_text", "published_at", "collected_at",
			"content_hash", "keywords",
		}).AddRow(
			rawEvidenceID, "SRC_0000000000000000000000000000", "Source", "INVALID",
			"https://example.test/article", true, nil, nil, nil, "hello", nil,
			time.Date(2026, 8, 11, 1, 2, 3, 0, time.UTC),
			"2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824", []byte(`[]`),
		))
	mock.ExpectRollback()

	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	accepted := errors.New("invalid persisted Raw Evidence was accepted")
	ctx := context.Background()
	err = store.InTransaction(ctx, func(tx evidencebiz.Transaction) error {
		_, readErr := tx.RawEvidence(ctx, rawEvidenceID)
		if readErr == nil {
			return accepted
		}
		return readErr
	})
	var invariantErr *persistedInvariantError
	if errors.Is(err, accepted) || !errors.As(err, &invariantErr) || invariantErr.field != "source_level" {
		t.Fatalf("RawEvidence() error = %v, want persisted source_level invariant error", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEvidenceTransactionRejectsInvalidPersistedEvidenceSet(t *testing.T) {
	const rawEvidenceID = "RAW_persisted_000000000000000"
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectBegin()
	rows := sqlmock.NewRows([]string{
		"evidence_id", "raw_evidence_id", "split_order", "is_split", "layer_type",
		"source_who", "source_what", "source_when", "source_when_raw", "source_where", "source_why", "source_how",
		"source_who_core", "source_what_core", "source_when_core", "source_when_raw_core",
		"source_where_core", "source_why_core", "source_how_core",
		"expression_fingerprint", "expression_key", "fingerprint_version",
	})
	rows.AddRow(persistedEvidenceRow("EVD_persisted_000000000000000", rawEvidenceID, 0, "first fact")...)
	rows.AddRow(persistedEvidenceRow("EVD_persisted_000000000000002", rawEvidenceID, 2, "second fact")...)
	mock.ExpectQuery("FROM evidences").
		WithArgs(rawEvidenceID).
		WillReturnRows(rows)
	mock.ExpectRollback()

	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	accepted := errors.New("invalid persisted Evidence set was accepted")
	ctx := context.Background()
	err = store.InTransaction(ctx, func(tx evidencebiz.Transaction) error {
		_, readErr := tx.EvidencesByRawEvidence(ctx, rawEvidenceID)
		if readErr == nil {
			return accepted
		}
		return readErr
	})
	var invariantErr *persistedInvariantError
	if errors.Is(err, accepted) || !errors.As(err, &invariantErr) || invariantErr.field != "split_order" {
		t.Fatalf("EvidencesByRawEvidence() error = %v, want persisted split_order invariant error", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func persistedEvidenceRow(id, rawEvidenceID string, splitOrder int, sourceWhat string) []driver.Value {
	return []driver.Value{
		id, rawEvidenceID, splitOrder, true, "SINGLE",
		nil, sourceWhat, nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil, nil, nil,
		sourceWhat + " normalized", id + "-key", "v1",
	}
}

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
