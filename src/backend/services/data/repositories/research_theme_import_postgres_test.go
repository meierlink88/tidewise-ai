package repositories

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestResearchThemeImportBatchLockUsesTransactionAdvisoryLock(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	repositoryTx := &postgresResearchThemeImportTx{tx: tx}

	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_xact_lock(hashtextextended($1, 0))")).
		WithArgs("batch-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repositoryTx.LockResearchThemeImportBatch(context.Background(), "batch-1"); err != nil {
		t.Fatal(err)
	}
	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestResearchThemeImportReceiptDecodesFrozenResult(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	repositoryTx := &postgresResearchThemeImportTx{tx: tx}
	now := time.Date(2026, 7, 19, 9, 30, 0, 0, time.UTC)
	query := `SELECT id, analysis_batch_id, publisher_subject, payload_hash,
       theme_ids_by_key, write_counts, published_at, imported_at
FROM research_theme_import_receipts WHERE analysis_batch_id = $1`
	mock.ExpectQuery(regexp.QuoteMeta(query)).WithArgs("batch-1").WillReturnRows(sqlmock.NewRows([]string{
		"id", "analysis_batch_id", "publisher_subject", "payload_hash", "theme_ids_by_key", "write_counts", "published_at", "imported_at",
	}).AddRow(
		"11111111-1111-4111-8111-111111111111", "batch-1", "agent-run",
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		[]byte(`{"theme:a":"22222222-2222-4222-8222-222222222222"}`),
		[]byte(`{"themes":1,"chain_node_associations":2,"event_associations":3,"receipts":1}`), now, now,
	))

	receipt, err := repositoryTx.ResearchThemeImportReceipt(context.Background(), "batch-1")
	if err != nil {
		t.Fatal(err)
	}
	if receipt == nil || receipt.ThemeIDsByKey["theme:a"] != "22222222-2222-4222-8222-222222222222" {
		t.Fatalf("receipt = %#v", receipt)
	}
	if receipt.Counts != (ResearchThemeImportCounts{Themes: 1, ChainNodeAssociations: 2, EventAssociations: 3, Receipts: 1}) {
		t.Fatalf("counts = %#v", receipt.Counts)
	}
	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyResearchThemeImportReceiptScopesEveryQueryToReceipt(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	repositoryTx := &postgresResearchThemeImportTx{tx: tx}
	receipt := ResearchThemeImportReceipt{
		ID: "11111111-1111-4111-8111-111111111111",
		ThemeIDsByKey: map[string]string{
			"theme:a": "22222222-2222-4222-8222-222222222222",
		},
		Counts: ResearchThemeImportCounts{Themes: 1, ChainNodeAssociations: 2, EventAssociations: 3, Receipts: 1},
	}

	mock.ExpectQuery("SELECT COALESCE\\(jsonb_object_agg").
		WithArgs(receipt.ID).
		WillReturnRows(sqlmock.NewRows([]string{"theme_ids"}).AddRow([]byte(`{"theme:a":"22222222-2222-4222-8222-222222222222"}`)))
	mock.ExpectQuery("SELECT").
		WithArgs(receipt.ID).
		WillReturnRows(sqlmock.NewRows([]string{"themes", "nodes", "events"}).AddRow(1, 2, 3))

	if err := repositoryTx.VerifyResearchThemeImportReceipt(context.Background(), receipt); err != nil {
		t.Fatal(err)
	}
	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
