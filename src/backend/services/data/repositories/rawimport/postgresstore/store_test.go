package postgresstore

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/meierlink88/tidewise-ai/backend/services/data/repositories"
	"github.com/meierlink88/tidewise-ai/backend/services/data/usecase/rawimport"
)

const (
	sourceID = "22222222-2222-5222-8222-222222222222"
)

var rawID = repositories.RawDocumentUUID(sourceID, "", "story-1", strings.Repeat("a", 64))

func TestLockReceiptUsesAdvisoryTransactionLockBeforePlainSelect(t *testing.T) {
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
	storeTx := &transaction{tx: tx}

	lockText := "raw-receipt:v1|9|agent-run|batch-1"
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_xact_lock(hashtextextended($1, 0))")).
		WithArgs(lockText).
		WillReturnResult(sqlmock.NewResult(0, 1))
	plainSelect := "^" + regexp.QuoteMeta(strings.TrimSpace(receiptSelectSQL)) + "$"
	mock.ExpectQuery(plainSelect).
		WithArgs("agent-run", "batch-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "caller_identity", "idempotency_key", "payload_hash", "raw_document_ids", "result_payload", "imported_at"}))

	receipt, err := storeTx.LockReceipt(context.Background(), lockText, "agent-run", "batch-1")
	if err != nil {
		t.Fatal(err)
	}
	if receipt != nil {
		t.Fatalf("receipt = %#v, want nil", receipt)
	}
	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("LockReceipt must issue advisory lock then plain SELECT without FOR UPDATE/FOR SHARE: %v", err)
	}
}

func TestRawImportUsesOneTransactionAndConflictSafeWinnerRead(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	caller := "agent-run"
	key := "batch-1"
	receiptLock := "raw-receipt:v1|9|agent-run|batch-1"
	externalLock := "raw-external:v1|" + sourceID + "|7|story-1"
	hash := strings.Repeat("a", 64)
	hashLock := "raw-hash:v1|" + sourceID + "|" + hash
	importedAt := time.Date(2026, 7, 17, 3, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_xact_lock(hashtextextended($1, 0))")).WithArgs(receiptLock).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT id, caller_identity, idempotency_key, payload_hash").WithArgs(caller, key).WillReturnError(sqlmock.ErrCancelled)
	mock.ExpectRollback()

	service := rawimport.NewService(New(db), func() time.Time { return importedAt })
	_, err = service.Import(context.Background(), caller, key, batch(hash))
	if !errors.Is(err, sqlmock.ErrCancelled) {
		t.Fatalf("receipt read error = %v, want sqlmock cancellation", err)
	}
	mock.ExpectationsWereMet()

	// Recreate the mock so the successful path has a precise, readable order.
	db.Close()
	db, mock, err = sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_xact_lock(hashtextextended($1, 0))")).WithArgs(receiptLock).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT id, caller_identity, idempotency_key, payload_hash").WithArgs(caller, key).WillReturnRows(sqlmock.NewRows([]string{"id", "caller_identity", "idempotency_key", "payload_hash", "raw_document_ids", "result_payload", "imported_at"}))
	mock.ExpectQuery("SELECT id, ingest_channel, source_type, source_name, source_url, status").WithArgs(sourceID).WillReturnRows(sqlmock.NewRows([]string{"id", "ingest_channel", "source_type", "source_name", "source_url", "status"}).AddRow(sourceID, "rss", "news", "Example", "https://example.test/feed", "active"))
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_xact_lock(hashtextextended($1, 0))")).WithArgs(externalLock).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_xact_lock(hashtextextended($1, 0))")).WithArgs(hashLock).WillReturnResult(sqlmock.NewResult(0, 1))
	expectExternalLookup(mock, sourceID, "story-1", nil)
	expectHashLookup(mock, sourceID, hash, nil)
	mock.ExpectExec("(?s)INSERT INTO raw_documents .*ON CONFLICT DO NOTHING").WillReturnResult(sqlmock.NewResult(0, 1))
	expectExternalLookup(mock, sourceID, "story-1", []string{rawID})
	expectHashLookup(mock, sourceID, hash, []string{rawID})
	mock.ExpectExec("(?s)INSERT INTO raw_document_import_receipts").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	service = rawimport.NewService(New(db), func() time.Time { return importedAt })
	result, err := service.Import(context.Background(), caller, key, batch(hash))
	if err != nil {
		t.Fatal(err)
	}
	if result.RawDocumentIDs[0] != rawID || result.Items[0].Disposition != rawimport.DispositionCreated {
		t.Fatalf("result = %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRawImportIdentityConflictRollsBackWithoutInsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	hash := strings.Repeat("a", 64)
	mock.ExpectBegin()
	mock.ExpectExec("SELECT pg_advisory_xact_lock").WithArgs("raw-receipt:v1|9|agent-run|batch-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT id, caller_identity, idempotency_key, payload_hash").WithArgs("agent-run", "batch-1").WillReturnRows(sqlmock.NewRows([]string{"id", "caller_identity", "idempotency_key", "payload_hash", "raw_document_ids", "result_payload", "imported_at"}))
	mock.ExpectQuery("SELECT id, ingest_channel, source_type, source_name, source_url, status").WithArgs(sourceID).WillReturnRows(sqlmock.NewRows([]string{"id", "ingest_channel", "source_type", "source_name", "source_url", "status"}).AddRow(sourceID, "rss", "news", "Example", "https://example.test/feed", "active"))
	mock.ExpectExec("SELECT pg_advisory_xact_lock").WithArgs("raw-external:v1|" + sourceID + "|7|story-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("SELECT pg_advisory_xact_lock").WithArgs("raw-hash:v1|" + sourceID + "|" + hash).WillReturnResult(sqlmock.NewResult(0, 1))
	expectExternalLookup(mock, sourceID, "story-1", []string{"aaaaaaaa-aaaa-5aaa-8aaa-aaaaaaaaaaaa"})
	expectHashLookup(mock, sourceID, hash, []string{"bbbbbbbb-bbbb-5bbb-8bbb-bbbbbbbbbbbb"})
	mock.ExpectRollback()

	service := rawimport.NewService(New(db), nil)
	_, err = service.Import(context.Background(), "agent-run", "batch-1", batch(hash))
	if rawimport.ErrorCode(err) != rawimport.CodeIdentityConflict {
		t.Fatalf("error = %v, code = %q", err, rawimport.ErrorCode(err))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRawImportSourceNotFoundIsANameOnlyValidationError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	hash := strings.Repeat("a", 64)
	mock.ExpectBegin()
	mock.ExpectExec("SELECT pg_advisory_xact_lock").WithArgs("raw-receipt:v1|9|agent-run|batch-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT id, caller_identity, idempotency_key, payload_hash").WithArgs("agent-run", "batch-1").WillReturnRows(sqlmock.NewRows([]string{"id", "caller_identity", "idempotency_key", "payload_hash", "raw_document_ids", "result_payload", "imported_at"}))
	mock.ExpectQuery("SELECT id, ingest_channel, source_type, source_name, source_url, status").WithArgs(sourceID).WillReturnRows(sqlmock.NewRows([]string{"id", "ingest_channel", "source_type", "source_name", "source_url", "status"}))
	mock.ExpectRollback()

	_, err = rawimport.NewService(New(db), nil).Import(context.Background(), "agent-run", "batch-1", batch(hash))
	if rawimport.ErrorCode(err) != rawimport.CodeInvalidRequest || !strings.Contains(err.Error(), "source") {
		t.Fatalf("source-not-found error = %v, code = %q", err, rawimport.ErrorCode(err))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRawImportStatusDecodesImmutableStoredResult(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	importedAt := time.Date(2026, 7, 17, 3, 0, 0, 0, time.UTC)
	service := rawimport.NewService(New(db), nil)
	plan, err := service.Plan("agent-run", "batch-1", batch(strings.Repeat("a", 64)))
	if err != nil {
		t.Fatal(err)
	}
	resultJSON := `{"receipt_id":"` + plan.ReceiptID + `","payload_hash":"` + plan.PayloadHash + `","raw_document_ids":["` + rawID + `"],"items":[{"raw_document_id":"` + rawID + `","disposition":"created"}],"imported_at":"2026-07-17T03:00:00Z"}`
	mock.ExpectQuery("SELECT id, caller_identity, idempotency_key, payload_hash").WithArgs("agent-run", "batch-1").WillReturnRows(sqlmock.NewRows([]string{"id", "caller_identity", "idempotency_key", "payload_hash", "raw_document_ids", "result_payload", "imported_at"}).AddRow(plan.ReceiptID, "agent-run", "batch-1", plan.PayloadHash, `[`+`"`+rawID+`"`+`]`, resultJSON, importedAt))

	status, err := service.Status(context.Background(), "agent-run", "batch-1")
	if err != nil {
		t.Fatal(err)
	}
	if status.State != rawimport.StatusCompleted || status.Result == nil || status.Result.ReceiptID != plan.ReceiptID {
		t.Fatalf("status = %#v", status)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func batch(hash string) rawimport.Batch {
	published := time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC)
	return rawimport.Batch{Items: []rawimport.Candidate{{
		SourceID: sourceID, SourceExternalID: "story-1",
		IngestChannel: "rss", SourceType: "news", SourceName: "Example", SourceURL: "https://example.test/feed",
		Title: "Title", ContentText: "Body", ContentLevel: "full", Language: "en",
		PublishedAt: &published, CollectedAt: time.Date(2026, 7, 17, 1, 0, 0, 0, time.UTC), ContentHash: hash,
	}}}
}

func expectExternalLookup(mock sqlmock.Sqlmock, sourceID, externalID string, ids []string) {
	rows := sqlmock.NewRows([]string{"id"})
	for _, id := range ids {
		rows.AddRow(id)
	}
	mock.ExpectQuery("SELECT id FROM raw_documents[[:space:]]+WHERE source_id = \\$1 AND source_external_id = \\$2").WithArgs(sourceID, externalID).WillReturnRows(rows)
}

func expectHashLookup(mock sqlmock.Sqlmock, sourceID, hash string, ids []string) {
	rows := sqlmock.NewRows([]string{"id"})
	for _, id := range ids {
		rows.AddRow(id)
	}
	mock.ExpectQuery("SELECT id FROM raw_documents[[:space:]]+WHERE source_id = \\$1 AND content_hash = \\$2").WithArgs(sourceID, hash).WillReturnRows(rows)
}
