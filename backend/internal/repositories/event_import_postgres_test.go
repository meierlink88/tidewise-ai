package repositories

import (
	"context"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestEventImportLockReceiptUsesAdvisoryTransactionLockBeforePlainSelect(t *testing.T) {
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
	repositoryTx := &postgresEventImportTx{tx: tx}

	key := "reviewed-event-batch-1"
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_xact_lock(hashtextextended($1, 0))")).
		WithArgs(key).
		WillReturnResult(sqlmock.NewResult(0, 1))
	plainReceiptSelect := strings.TrimSpace(`
SELECT id, idempotency_key, package_id, review_id, review_decision, payload_hash,
       event_id, array_to_json(raw_document_ids), array_to_json(event_source_ids), array_to_json(event_tag_map_ids), review_metadata, imported_at
FROM event_import_receipts WHERE idempotency_key = $1`)
	mock.ExpectQuery("^" + regexp.QuoteMeta(plainReceiptSelect) + "$").
		WithArgs(key).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "idempotency_key", "package_id", "review_id", "review_decision", "payload_hash",
			"event_id", "raw_document_ids", "event_source_ids", "event_tag_map_ids", "review_metadata", "imported_at",
		}))

	receipt, err := repositoryTx.LockReceipt(context.Background(), key)
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

func TestDecodeReceiptUUIDJSONArray(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		data string
		want []string
	}{
		{"raw documents", `["11111111-1111-1111-1111-111111111111"]`, []string{"11111111-1111-1111-1111-111111111111"}},
		{"event sources", `["22222222-2222-2222-2222-222222222222","33333333-3333-3333-3333-333333333333"]`, []string{"22222222-2222-2222-2222-222222222222", "33333333-3333-3333-3333-333333333333"}},
		{"event tag maps", `["44444444-4444-4444-4444-444444444444","55555555-5555-5555-5555-555555555555","66666666-6666-6666-6666-666666666666"]`, []string{"44444444-4444-4444-4444-444444444444", "55555555-5555-5555-5555-555555555555", "66666666-6666-6666-6666-666666666666"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decodeReceiptUUIDJSONArray(tc.name, []byte(tc.data))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("decoded IDs = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestDecodeReceiptUUIDJSONArrayRejectsEmptyAndMalformedValues(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		data string
	}{
		{"empty", `[]`},
		{"malformed JSON", `not-json`},
		{"wrong JSON type", `{"id":"11111111-1111-1111-1111-111111111111"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := decodeReceiptUUIDJSONArray("raw_document_ids", []byte(tc.data))
			if err == nil || !strings.Contains(err.Error(), "raw_document_ids") {
				t.Fatalf("decode error = %v, want raw_document_ids context", err)
			}
		})
	}
}
