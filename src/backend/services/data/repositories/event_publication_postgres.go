package repositories

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
)

func (r PostgresRepository) InEventPublicationTransaction(ctx context.Context, fn func(EventPublicationTransaction) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin Event Publication V2 transaction: %w", err)
	}
	wrapper := &postgresEventPublicationTx{tx: tx}
	if err := fn(wrapper); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit Event Publication V2 transaction: %w", err)
	}
	return nil
}

type postgresEventPublicationTx struct {
	tx *sql.Tx
}

func (t *postgresEventPublicationTx) LockEventPublicationIdentities(ctx context.Context, identities []string) error {
	keys := append([]string(nil), identities...)
	sort.Strings(keys)
	for _, key := range keys {
		if _, err := t.tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, key); err != nil {
			return fmt.Errorf("lock Event Publication V2 identity %q: %w", key, err)
		}
	}
	return nil
}

func (t *postgresEventPublicationTx) PublicationRawDocument(ctx context.Context, artifactID string) (*PublicationRawDocument, error) {
	var record PublicationRawDocument
	err := t.tx.QueryRowContext(ctx, `
SELECT id, artifact_id, content_hash, source_ref, source_name, source_type, source_url,
       title, published_at, collected_at, language, raw_mime_type
FROM raw_documents
WHERE contract_version = 2 AND artifact_id = $1`, artifactID).Scan(
		&record.ID, &record.ArtifactID, &record.ContentSHA256, &record.SourceRef,
		&record.SourceName, &record.SourceType, &record.SourceURL, &record.Title,
		&record.PublishedAt, &record.CollectedAt, &record.Language, &record.MIMEType,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read publication raw document %q: %w", artifactID, err)
	}
	return &record, nil
}

func (t *postgresEventPublicationTx) InsertPublicationRawDocument(ctx context.Context, record PublicationRawDocument) error {
	_, err := t.tx.ExecContext(ctx, `
INSERT INTO raw_documents (
    id, contract_version, artifact_id, source_ref, ingest_channel, source_type, source_name,
    source_url, source_external_id, title, content_text, content_level, raw_object_uri,
    raw_mime_type, language, published_at, collected_at, content_hash, ingest_status
) VALUES ($1,2,$2,$3,'',$4,$5,$6,NULL,$7,'','', '',$8,$9,$10,$11,$12,'collected')`,
		record.ID, record.ArtifactID, record.SourceRef, record.SourceType, record.SourceName,
		record.SourceURL, record.Title, record.MIMEType, record.Language,
		nullTime(record.PublishedAt), record.CollectedAt, record.ContentSHA256,
	)
	if err != nil {
		return fmt.Errorf("insert publication raw document %q: %w", record.ArtifactID, err)
	}
	return nil
}

func (t *postgresEventPublicationTx) PublicationEvent(ctx context.Context, dedupeKey string) (*PublicationEvent, error) {
	var record PublicationEvent
	var factPayload []byte
	var knowableAt *time.Time
	var primarySourceID *string
	err := t.tx.QueryRowContext(ctx, `
SELECT id, dedupe_key, title, summary, event_time, fact_payload, first_seen_at, knowable_at,
       event_status, fact_status, primary_source_id
FROM events
WHERE dedupe_key = $1`, dedupeKey).Scan(
		&record.ID, &record.DedupeKey, &record.Title, &record.FactualSummary,
		&record.OccurredAt, &factPayload, &record.FirstSeenAt, &knowableAt,
		&record.EventStatus, &record.FactStatus, &primarySourceID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read publication Event %q: %w", dedupeKey, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(factPayload))
	decoder.UseNumber()
	if err := decoder.Decode(&record.FactPayload); err != nil {
		return nil, fmt.Errorf("decode publication Event %q fact payload: %w", dedupeKey, err)
	}
	if knowableAt != nil {
		record.KnowableAt = *knowableAt
	}
	if primarySourceID != nil {
		record.PrimarySourceID = *primarySourceID
	}
	return &record, nil
}

func (t *postgresEventPublicationTx) InsertPublicationEvent(ctx context.Context, record PublicationEvent) error {
	factPayload, err := json.Marshal(record.FactPayload)
	if err != nil {
		return fmt.Errorf("encode publication Event %q fact payload: %w", record.DedupeKey, err)
	}
	_, err = t.tx.ExecContext(ctx, `
INSERT INTO events (
    id, title, summary, event_time, first_seen_at, knowable_at,
    event_status, fact_status, dedupe_key, fact_payload
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		record.ID, record.Title, record.FactualSummary, nullTime(record.OccurredAt),
		record.FirstSeenAt, record.KnowableAt, record.EventStatus, record.FactStatus,
		record.DedupeKey, factPayload,
	)
	if err != nil {
		return fmt.Errorf("insert publication Event %q: %w", record.DedupeKey, err)
	}
	return nil
}

func (t *postgresEventPublicationTx) AdvancePublicationEventObservationTimes(ctx context.Context, eventID string, firstSeenAt, knowableAt time.Time) error {
	_, err := t.tx.ExecContext(ctx, `
UPDATE events
SET first_seen_at = LEAST(first_seen_at, $2),
    knowable_at = LEAST(COALESCE(knowable_at, $3), $3),
    updated_at = now()
WHERE id = $1`, eventID, firstSeenAt, knowableAt)
	if err != nil {
		return fmt.Errorf("advance Event observation times %q: %w", eventID, err)
	}
	return nil
}

func (t *postgresEventPublicationTx) PublicationEventSource(ctx context.Context, eventID, rawDocumentID string) (*PublicationEventSource, error) {
	var record PublicationEventSource
	var supportsFieldsJSON []byte
	err := t.tx.QueryRowContext(ctx, `
SELECT id, event_id, raw_document_id, source_level, evidence_excerpt, evidence_hash,
       evidence_relation, array_to_json(supports_fields), is_primary
FROM event_sources
WHERE contract_version = 2 AND event_id = $1 AND raw_document_id = $2`,
		eventID, rawDocumentID,
	).Scan(
		&record.ID, &record.EventID, &record.RawDocumentID, &record.SourceLevel,
		&record.EvidenceExcerpt, &record.EvidenceHash, &record.EvidenceRelation,
		&supportsFieldsJSON, &record.IsPrimary,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read publication Event Source %q/%q: %w", eventID, rawDocumentID, err)
	}
	if err := json.Unmarshal(supportsFieldsJSON, &record.SupportsFields); err != nil {
		return nil, fmt.Errorf("decode publication Event Source %q/%q supports fields: %w", eventID, rawDocumentID, err)
	}
	return &record, nil
}

func (t *postgresEventPublicationTx) InsertPublicationEventSource(ctx context.Context, record PublicationEventSource) error {
	_, err := t.tx.ExecContext(ctx, `
INSERT INTO event_sources (
    id, contract_version, event_id, raw_document_id, source_level, evidence_excerpt,
    evidence_hash, evidence_relation, supports_fields, is_primary
) VALUES ($1,2,$2,$3,$4,$5,$6,$7,$8,$9)`,
		record.ID, record.EventID, record.RawDocumentID, record.SourceLevel,
		record.EvidenceExcerpt, record.EvidenceHash, record.EvidenceRelation,
		record.SupportsFields, record.IsPrimary,
	)
	if err != nil {
		return fmt.Errorf("insert publication Event Source %q/%q: %w", record.EventID, record.RawDocumentID, err)
	}
	return nil
}

func (t *postgresEventPublicationTx) SetPublicationEventPrimarySource(ctx context.Context, eventID, sourceID string) error {
	result, err := t.tx.ExecContext(ctx, `
UPDATE events
SET primary_source_id = $2, updated_at = now()
WHERE id = $1 AND (primary_source_id IS NULL OR primary_source_id = $2)`, eventID, sourceID)
	if err != nil {
		return fmt.Errorf("set publication Event primary source: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read publication Event primary source result: %w", err)
	}
	if rows != 1 {
		return fmt.Errorf("publication Event %q already has another primary source", eventID)
	}
	return nil
}

func (t *postgresEventPublicationTx) PublicationTag(ctx context.Context, tagID string) (*domain.EventTagDef, error) {
	var tag domain.EventTagDef
	err := t.tx.QueryRowContext(ctx, `
SELECT id, tag_kind, code, name, is_active
FROM event_tag_defs
WHERE id = $1`, tagID).Scan(&tag.ID, &tag.TagKind, &tag.Code, &tag.Name, &tag.IsActive)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read publication Tag %q: %w", tagID, err)
	}
	return &tag, nil
}

func (t *postgresEventPublicationTx) PublicationEventTag(ctx context.Context, eventID, tagID string) (*PublicationEventTag, error) {
	var record PublicationEventTag
	err := t.tx.QueryRowContext(ctx, `
SELECT id, event_id, tag_id, assign_source, review_status, confidence::text, assignment_reason
FROM event_tag_maps
WHERE event_id = $1 AND tag_id = $2`, eventID, tagID).Scan(
		&record.ID, &record.EventID, &record.TagID, &record.AssignSource,
		&record.ReviewStatus, &record.Confidence, &record.AssignmentReason,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read publication Event Tag %q/%q: %w", eventID, tagID, err)
	}
	return &record, nil
}

func (t *postgresEventPublicationTx) InsertPublicationEventTag(ctx context.Context, record PublicationEventTag) error {
	_, err := t.tx.ExecContext(ctx, `
INSERT INTO event_tag_maps (
    id, event_id, tag_id, assign_source, review_status, confidence, assignment_reason
) VALUES ($1,$2,$3,$4,$5,$6::numeric,$7)`,
		record.ID, record.EventID, record.TagID, record.AssignSource,
		record.ReviewStatus, record.Confidence, record.AssignmentReason,
	)
	if err != nil {
		return fmt.Errorf("insert publication Event Tag %q/%q: %w", record.EventID, record.TagID, err)
	}
	return nil
}

func (t *postgresEventPublicationTx) InsertEventPublicationReceipt(ctx context.Context, receipt EventPublicationReceipt) error {
	collectorExecutions, err := json.Marshal(receipt.CollectorExecutions)
	if err != nil {
		return fmt.Errorf("encode publication collector executions: %w", err)
	}
	reviewMetadata, err := json.Marshal(receipt.ReviewMetadata)
	if err != nil {
		return fmt.Errorf("encode publication review metadata: %w", err)
	}
	writeCounts, err := json.Marshal(receipt.WriteCounts)
	if err != nil {
		return fmt.Errorf("encode publication write counts: %w", err)
	}
	_, err = t.tx.ExecContext(ctx, `
INSERT INTO event_publication_receipts (
    id, contract_version, package_id, caller_subject, extractor_execution_id,
    extractor_agent_version, collector_executions, event_ids, raw_document_ids,
    event_source_ids, event_tag_map_ids, review_metadata, write_counts, imported_at
) VALUES ($1,2,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		receipt.ID, receipt.PackageID, receipt.CallerSubject, receipt.ExtractorExecutionID,
		receipt.ExtractorAgentVersion, collectorExecutions, receipt.EventIDs,
		receipt.RawDocumentIDs, receipt.EventSourceIDs, receipt.EventTagMapIDs,
		reviewMetadata, writeCounts, receipt.ImportedAt,
	)
	if err != nil {
		return fmt.Errorf("insert Event Publication V2 receipt: %w", err)
	}
	return nil
}
