// Package postgresstore provides the dedicated raw-import PostgreSQL adapter.
package postgresstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	"github.com/meierlink88/tidewise-ai/backend/services/data/rawimport"
)

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store { return &Store{db: db} }

func (s *Store) InRawImportTransaction(ctx context.Context, fn func(rawimport.Transaction) error) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("raw import database is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin raw import transaction: %w", err)
	}
	wrapper := &transaction{tx: tx}
	if err := fn(wrapper); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit raw import transaction: %w", err)
	}
	return nil
}

func (s *Store) RawImportReceipt(ctx context.Context, caller, key string) (*rawimport.Receipt, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("raw import database is required")
	}
	receipt, err := scanReceipt(s.db.QueryRowContext(ctx, receiptSelectSQL, caller, key))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read raw import receipt: %w", err)
	}
	return &receipt, nil
}

type transaction struct {
	tx *sql.Tx
}

func (t *transaction) LockReceipt(ctx context.Context, lockText, caller, key string) (*rawimport.Receipt, error) {
	if _, err := t.tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, lockText); err != nil {
		return nil, fmt.Errorf("lock raw import receipt: %w", err)
	}
	receipt, err := scanReceipt(t.tx.QueryRowContext(ctx, receiptSelectSQL, caller, key))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read locked raw import receipt: %w", err)
	}
	return &receipt, nil
}

func (t *transaction) Source(ctx context.Context, sourceID string) (domain.SourceCatalog, error) {
	var source domain.SourceCatalog
	var status string
	err := t.tx.QueryRowContext(ctx, `
SELECT id, ingest_channel, source_type, source_name, source_url, status
FROM source_catalogs
WHERE id = $1
	`, sourceID).Scan(&source.ID, &source.IngestChannel, &source.SourceType, &source.SourceName, &source.SourceURL, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.SourceCatalog{}, fmt.Errorf("%w: %s", rawimport.ErrSourceNotFound, sourceID)
	}
	if err != nil {
		return domain.SourceCatalog{}, fmt.Errorf("read raw import source: %w", err)
	}
	source.Status = domain.SourceCatalogStatus(status)
	return source, nil
}

func (t *transaction) LockRawIdentities(ctx context.Context, lockTexts []string) error {
	if !sort.StringsAreSorted(lockTexts) {
		return fmt.Errorf("raw identity locks must be sorted")
	}
	for index, lockText := range lockTexts {
		if lockText == "" || (index > 0 && lockTexts[index-1] == lockText) {
			return fmt.Errorf("raw identity locks must be nonempty and unique")
		}
		if _, err := t.tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, lockText); err != nil {
			return fmt.Errorf("lock raw identity: %w", err)
		}
	}
	return nil
}

func (t *transaction) RawDocumentByExternalID(ctx context.Context, sourceID, externalID string) (string, error) {
	if externalID == "" {
		return "", nil
	}
	return querySingleRawID(ctx, t.tx, `
SELECT id FROM raw_documents
WHERE source_id = $1 AND source_external_id = $2
`, sourceID, externalID)
}

func (t *transaction) RawDocumentByContentHash(ctx context.Context, sourceID, hash string) (string, error) {
	return querySingleRawID(ctx, t.tx, `
SELECT id FROM raw_documents
WHERE source_id = $1 AND content_hash = $2
`, sourceID, hash)
}

func (t *transaction) InsertRawDocument(ctx context.Context, document domain.RawDocument) (bool, error) {
	if err := document.Validate(); err != nil {
		return false, err
	}
	result, err := t.tx.ExecContext(ctx, `
INSERT INTO raw_documents (
    id, source_id, ingest_channel, source_type, source_name, source_url,
    source_external_id, title, content_text, content_level, raw_object_uri, raw_mime_type,
    language, published_at, collected_at, content_hash, ingest_status
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11, $12,
    $13, $14, $15, $16, $17
)
ON CONFLICT DO NOTHING
`, document.ID, document.SourceID, document.IngestChannel, document.SourceType, document.SourceName, document.SourceURL,
		nullString(document.SourceExternalID), document.Title, document.ContentText, document.ContentLevel, document.RawObjectURI, document.RawMIMEType,
		document.Language, nullTime(document.PublishedAt), document.CollectedAt, document.ContentHash, document.IngestStatus)
	if err != nil {
		return false, fmt.Errorf("insert raw document: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read raw document insert result: %w", err)
	}
	return rows == 1, nil
}

func (t *transaction) InsertReceipt(ctx context.Context, receipt rawimport.Receipt) error {
	payload, err := json.Marshal(receipt.Result)
	if err != nil {
		return fmt.Errorf("encode raw import result: %w", err)
	}
	rawIDs, err := json.Marshal(receipt.RawDocumentIDs)
	if err != nil {
		return fmt.Errorf("encode raw import membership: %w", err)
	}
	_, err = t.tx.ExecContext(ctx, `
INSERT INTO raw_document_import_receipts (
    id, caller_identity, idempotency_key, payload_hash,
    raw_document_ids, result_payload, imported_at
) VALUES (
    $1, $2, $3, $4,
    ARRAY(SELECT jsonb_array_elements_text($5::jsonb)::uuid), $6::jsonb, $7
)
`, receipt.ID, receipt.CallerIdentity, receipt.IdempotencyKey, receipt.PayloadHash,
		string(rawIDs), string(payload), receipt.ImportedAt)
	if err != nil {
		return fmt.Errorf("insert raw import receipt: %w", err)
	}
	return nil
}

const receiptSelectSQL = `
SELECT id, caller_identity, idempotency_key, payload_hash,
       array_to_json(raw_document_ids), result_payload, imported_at
FROM raw_document_import_receipts
WHERE caller_identity = $1 AND idempotency_key = $2`

type receiptScanner interface {
	Scan(...any) error
}

func scanReceipt(scanner receiptScanner) (rawimport.Receipt, error) {
	var receipt rawimport.Receipt
	var rawIDsJSON []byte
	var resultJSON []byte
	if err := scanner.Scan(
		&receipt.ID, &receipt.CallerIdentity, &receipt.IdempotencyKey, &receipt.PayloadHash,
		&rawIDsJSON, &resultJSON, &receipt.ImportedAt,
	); err != nil {
		return rawimport.Receipt{}, err
	}
	if err := json.Unmarshal(rawIDsJSON, &receipt.RawDocumentIDs); err != nil {
		return rawimport.Receipt{}, fmt.Errorf("decode raw receipt membership: %w", err)
	}
	if err := json.Unmarshal(resultJSON, &receipt.Result); err != nil {
		return rawimport.Receipt{}, fmt.Errorf("decode raw receipt result: %w", err)
	}
	return receipt, nil
}

type rowQuerier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func querySingleRawID(ctx context.Context, queryer rowQuerier, query string, args ...any) (string, error) {
	rows, err := queryer.QueryContext(ctx, query, args...)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return "", err
		}
		ids = append(ids, id)
		if len(ids) > 1 {
			return "", fmt.Errorf("raw identity resolved to multiple rows")
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(ids) == 0 {
		return "", nil
	}
	return ids[0], nil
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}
