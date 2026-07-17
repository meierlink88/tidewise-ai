package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func (r PostgresRepository) InTransaction(ctx context.Context, fn func(EventImportTransaction) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin event import transaction: %w", err)
	}
	wrapper := &postgresEventImportTx{tx: tx}
	if err := fn(wrapper); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit event import transaction: %w", err)
	}
	return nil
}

type postgresEventImportTx struct{ tx *sql.Tx }

func (t *postgresEventImportTx) VerifyReceiptResults(ctx context.Context, receipt EventImportReceipt) error {
	var count int
	if err := t.tx.QueryRowContext(ctx, `SELECT count(*) FROM events WHERE id = $1`, receipt.EventID).Scan(&count); err != nil {
		return fmt.Errorf("verify receipt event: %w", err)
	}
	if count != 1 {
		return fmt.Errorf("receipt event %q does not exist", receipt.EventID)
	}
	if err := t.tx.QueryRowContext(ctx, `SELECT count(*) FROM raw_documents WHERE id = ANY($1::uuid[])`, receipt.RawDocumentIDs).Scan(&count); err != nil {
		return fmt.Errorf("verify receipt raw documents: %w", err)
	}
	if count != len(receipt.RawDocumentIDs) {
		return fmt.Errorf("receipt raw document IDs are not all present")
	}
	if err := t.tx.QueryRowContext(ctx, `SELECT count(*) FROM event_sources WHERE id = ANY($1::uuid[]) AND event_id = $2`, receipt.EventSourceIDs, receipt.EventID).Scan(&count); err != nil {
		return fmt.Errorf("verify receipt event sources: %w", err)
	}
	if count != len(receipt.EventSourceIDs) {
		return fmt.Errorf("receipt event source IDs are not all present for event %q", receipt.EventID)
	}
	if err := t.tx.QueryRowContext(ctx, `SELECT count(*) FROM event_tag_maps WHERE id = ANY($1::uuid[]) AND event_id = $2`, receipt.EventTagMapIDs, receipt.EventID).Scan(&count); err != nil {
		return fmt.Errorf("verify receipt event tag maps: %w", err)
	}
	if count != len(receipt.EventTagMapIDs) {
		return fmt.Errorf("receipt event tag map IDs are not all present for event %q", receipt.EventID)
	}
	return nil
}

func (t *postgresEventImportTx) LockReceipt(ctx context.Context, key string) (*EventImportReceipt, error) {
	if _, err := t.tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, key); err != nil {
		return nil, fmt.Errorf("lock event import idempotency key: %w", err)
	}
	var receipt EventImportReceipt
	var rawDocumentIDsJSON []byte
	var eventSourceIDsJSON []byte
	var eventTagMapIDsJSON []byte
	var metadata []byte
	err := t.tx.QueryRowContext(ctx, `
SELECT id, idempotency_key, package_id, review_id, review_decision, payload_hash,
       event_id, array_to_json(raw_document_ids), array_to_json(event_source_ids), array_to_json(event_tag_map_ids), review_metadata, imported_at
FROM event_import_receipts WHERE idempotency_key = $1`, key).Scan(
		&receipt.ID, &receipt.IdempotencyKey, &receipt.PackageID, &receipt.ReviewID, &receipt.ReviewDecision, &receipt.PayloadHash,
		&receipt.EventID, &rawDocumentIDsJSON, &eventSourceIDsJSON, &eventTagMapIDsJSON, &metadata, &receipt.ImportedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read event import receipt: %w", err)
	}
	if err := json.Unmarshal(metadata, &receipt.ReviewMetadata); err != nil {
		return nil, fmt.Errorf("decode event import review metadata: %w", err)
	}
	if receipt.RawDocumentIDs, err = decodeReceiptUUIDJSONArray("raw_document_ids", rawDocumentIDsJSON); err != nil {
		return nil, err
	}
	if receipt.EventSourceIDs, err = decodeReceiptUUIDJSONArray("event_source_ids", eventSourceIDsJSON); err != nil {
		return nil, err
	}
	if receipt.EventTagMapIDs, err = decodeReceiptUUIDJSONArray("event_tag_map_ids", eventTagMapIDsJSON); err != nil {
		return nil, err
	}
	return &receipt, nil
}

func decodeReceiptUUIDJSONArray(field string, data []byte) ([]string, error) {
	var ids []string
	if err := json.Unmarshal(data, &ids); err != nil {
		return nil, fmt.Errorf("decode receipt %s: %w", field, err)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("decode receipt %s: array must not be empty", field)
	}
	for _, id := range ids {
		if id == "" {
			return nil, fmt.Errorf("decode receipt %s: array contains empty ID", field)
		}
	}
	return ids, nil
}

func (t *postgresEventImportTx) Source(ctx context.Context, id string) (domain.SourceCatalog, error) {
	var source domain.SourceCatalog
	var status string
	var sourceConfig []byte
	err := t.tx.QueryRowContext(ctx, `SELECT id, ingest_channel, source_type, status, source_config FROM source_catalogs WHERE id = $1`, id).Scan(&source.ID, &source.IngestChannel, &source.SourceType, &status, &sourceConfig)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.SourceCatalog{}, fmt.Errorf("source %q not found", id)
	}
	if err != nil {
		return domain.SourceCatalog{}, err
	}
	source.Status = domain.SourceCatalogStatus(status)
	if len(sourceConfig) > 0 {
		if err := json.Unmarshal(sourceConfig, &source.SourceConfig); err != nil {
			return domain.SourceCatalog{}, fmt.Errorf("decode source config: %w", err)
		}
	}
	return source, nil
}

func (t *postgresEventImportTx) UpsertRawDocument(ctx context.Context, doc domain.RawDocument) (string, error) {
	var id string
	err := t.tx.QueryRowContext(ctx, `
SELECT id FROM raw_documents
WHERE source_id = $1 AND (($2 <> '' AND source_external_id = $2) OR content_hash = $3)
LIMIT 1`, doc.SourceID, doc.SourceExternalID, doc.ContentHash).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("find duplicate raw document: %w", err)
	}
	_, err = t.tx.ExecContext(ctx, `
INSERT INTO raw_documents (
 id, source_id, ingest_channel, source_type, source_name, source_url, source_external_id,
 title, content_text, content_level, published_at, collected_at, content_hash, ingest_status
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		doc.ID, doc.SourceID, doc.IngestChannel, doc.SourceType, doc.SourceName, doc.SourceURL, doc.SourceExternalID,
		doc.Title, doc.ContentText, doc.ContentLevel, nullTime(doc.PublishedAt), doc.CollectedAt, doc.ContentHash, doc.IngestStatus)
	if err != nil {
		return "", fmt.Errorf("insert raw document: %w", err)
	}
	return doc.ID, nil
}

func (t *postgresEventImportTx) UpsertEvent(ctx context.Context, event domain.Event) (string, error) {
	factPayload, err := json.Marshal(event.FactPayload)
	if err != nil {
		return "", fmt.Errorf("encode event fact payload: %w", err)
	}
	_, err = t.tx.ExecContext(ctx, `
INSERT INTO events (id,title,summary,event_time,first_seen_at,knowable_at,event_status,fact_status,dedupe_key,fact_payload)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
ON CONFLICT (dedupe_key) DO NOTHING`, event.ID, event.Title, event.Summary, event.EventTime, event.FirstSeenAt, event.KnowableAt, event.EventStatus, event.FactStatus, event.DedupeKey, factPayload)
	if err != nil {
		return "", fmt.Errorf("upsert event: %w", err)
	}
	var id string
	if err := t.tx.QueryRowContext(ctx, `SELECT id FROM events WHERE dedupe_key = $1`, event.DedupeKey).Scan(&id); err != nil {
		return "", fmt.Errorf("read event after upsert: %w", err)
	}
	return id, nil
}

func (t *postgresEventImportTx) AddEventSource(ctx context.Context, source domain.EventSource) (string, error) {
	_, err := t.tx.ExecContext(ctx, `
INSERT INTO event_sources (id,event_id,raw_document_id,source_level,evidence_excerpt,evidence_hash,evidence_relation,supports_fields)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT (event_id,raw_document_id,evidence_hash) DO NOTHING`, source.ID, source.EventID, source.RawDocumentID, source.SourceLevel, source.EvidenceExcerpt, source.EvidenceHash, source.EvidenceRelation, source.SupportsFields)
	if err != nil {
		return "", fmt.Errorf("upsert event source: %w", err)
	}
	var id string
	if err := t.tx.QueryRowContext(ctx, `SELECT id FROM event_sources WHERE event_id=$1 AND raw_document_id=$2 AND evidence_hash=$3`, source.EventID, source.RawDocumentID, source.EvidenceHash).Scan(&id); err != nil {
		return "", fmt.Errorf("read event source after upsert: %w", err)
	}
	return id, nil
}

func (t *postgresEventImportTx) Tag(ctx context.Context, id, kind, code string) (domain.EventTagDef, error) {
	var definition domain.EventTagDef
	var active bool
	err := t.tx.QueryRowContext(ctx, `SELECT id, tag_kind, code, name, is_active FROM event_tag_defs WHERE id=$1 AND tag_kind=$2 AND code=$3`, id, kind, code).Scan(&definition.ID, &definition.TagKind, &definition.Code, &definition.Name, &active)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.EventTagDef{}, fmt.Errorf("tag %q/%q/%q not found", id, kind, code)
	}
	if err != nil {
		return domain.EventTagDef{}, err
	}
	if !active {
		return domain.EventTagDef{}, fmt.Errorf("tag %q is inactive", code)
	}
	return definition, nil
}

func (t *postgresEventImportTx) AssignEventTag(ctx context.Context, tagMap domain.EventTagMap) (string, error) {
	_, err := t.tx.ExecContext(ctx, `
INSERT INTO event_tag_maps (id,event_id,tag_id,assign_source,review_status,confidence,assignment_reason)
VALUES ($1,$2,$3,$4,$5,$6,$7)
ON CONFLICT (event_id,tag_id) DO NOTHING`, tagMap.ID, tagMap.EventID, tagMap.TagID, tagMap.AssignSource, tagMap.ReviewStatus, tagMap.Confidence, tagMap.AssignmentReason)
	if err != nil {
		return "", fmt.Errorf("upsert event tag: %w", err)
	}
	var id string
	if err := t.tx.QueryRowContext(ctx, `SELECT id FROM event_tag_maps WHERE event_id=$1 AND tag_id=$2`, tagMap.EventID, tagMap.TagID).Scan(&id); err != nil {
		return "", fmt.Errorf("read event tag after upsert: %w", err)
	}
	return id, nil
}

func (t *postgresEventImportTx) InsertReceipt(ctx context.Context, receipt EventImportReceipt) error {
	metadata, err := json.Marshal(receipt.ReviewMetadata)
	if err != nil {
		return err
	}
	_, err = t.tx.ExecContext(ctx, `
INSERT INTO event_import_receipts (id,idempotency_key,package_id,review_id,review_decision,payload_hash,event_id,raw_document_ids,event_source_ids,event_tag_map_ids,review_metadata,imported_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, receipt.ID, receipt.IdempotencyKey, receipt.PackageID, receipt.ReviewID, receipt.ReviewDecision, receipt.PayloadHash, receipt.EventID, receipt.RawDocumentIDs, receipt.EventSourceIDs, receipt.EventTagMapIDs, metadata, receipt.ImportedAt)
	return err
}
