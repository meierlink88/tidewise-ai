package event

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	eventbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/event"
)

func (s Store) InTransaction(ctx context.Context, fn func(eventbiz.Transaction) error) (resultErr error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin Event Publication transaction: %w", err)
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		failure := recover()
		rollbackErr := tx.Rollback()
		if errors.Is(rollbackErr, sql.ErrTxDone) {
			rollbackErr = nil
		}
		if failure != nil {
			if rollbackErr != nil {
				panic(fmt.Errorf("Event Publication panic (%v) and rollback failed: %w", failure, rollbackErr))
			}
			panic(failure)
		}
		if rollbackErr != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("roll back Event Publication transaction: %w", rollbackErr))
		}
	}()
	wrapper := &transaction{tx: tx}
	if err := fn(wrapper); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit Event Publication transaction: %w", err)
	}
	committed = true
	return nil
}

type transaction struct {
	tx *sql.Tx
}

func (t *transaction) LockIdentities(ctx context.Context, identities []string) error {
	keys := append([]string(nil), identities...)
	sort.Strings(keys)
	for _, key := range keys {
		if _, err := t.tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, key); err != nil {
			return fmt.Errorf("lock Event Publication identity %q: %w", key, err)
		}
	}
	return nil
}

func (t *transaction) StoredEventEvidenceRecord(ctx context.Context, artifactID string) (*eventbiz.StoredEventEvidenceRecord, error) {
	var record eventbiz.StoredEventEvidenceRecord
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
	if err := validateStoredEvidenceRecord(record, artifactID); err != nil {
		return nil, fmt.Errorf("read Event Evidence Record invariant: %w", err)
	}
	return &record, nil
}

func (t *transaction) InsertStoredEventEvidenceRecord(ctx context.Context, record eventbiz.StoredEventEvidenceRecord) error {
	_, err := t.tx.ExecContext(ctx, `
INSERT INTO raw_documents (
    id, contract_version, artifact_id, source_ref, ingest_channel, source_type, source_name,
    source_url, source_external_id, title, content_text, content_level, raw_object_uri,
    raw_mime_type, language, published_at, collected_at, content_hash, ingest_status
) VALUES ($1,2,$2,$3,'',$4,$5,$6,NULL,$7,'','', '',$8,$9,$10,$11,$12,'collected')`,
		record.ID, record.ArtifactID, record.SourceRef, record.SourceType, record.SourceName,
		record.SourceURL, record.Title, record.MIMEType, record.Language,
		nullableTime(record.PublishedAt), record.CollectedAt, record.ContentSHA256,
	)
	if err != nil {
		return fmt.Errorf("insert publication raw document %q: %w", record.ArtifactID, err)
	}
	return nil
}

func (t *transaction) StoredEvent(ctx context.Context, dedupeKey string) (*eventbiz.StoredEvent, error) {
	var record eventbiz.StoredEvent
	var factPayload []byte
	var knowableAt *time.Time
	err := t.tx.QueryRowContext(ctx, `
SELECT id, dedupe_key, title, summary, event_time, fact_payload, first_seen_at, knowable_at,
	   event_status, fact_status
FROM events
WHERE dedupe_key = $1`, dedupeKey).Scan(
		&record.ID, &record.DedupeKey, &record.Title, &record.FactualSummary,
		&record.OccurredAt, &factPayload, &record.FirstSeenAt, &knowableAt,
		&record.EventStatus, &record.FactStatus,
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
	if err := validateStoredPublicationEvent(record, dedupeKey); err != nil {
		return nil, fmt.Errorf("read Event invariant: %w", err)
	}
	return &record, nil
}

func (t *transaction) InsertStoredEvent(ctx context.Context, record eventbiz.StoredEvent) error {
	factPayload, err := json.Marshal(record.FactPayload)
	if err != nil {
		return fmt.Errorf("encode publication Event %q fact payload: %w", record.DedupeKey, err)
	}
	_, err = t.tx.ExecContext(ctx, `
INSERT INTO events (
    id, title, summary, event_time, first_seen_at, knowable_at,
    event_status, fact_status, dedupe_key, fact_payload
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		record.ID, record.Title, record.FactualSummary, nullableTime(record.OccurredAt),
		record.FirstSeenAt, record.KnowableAt, record.EventStatus, record.FactStatus,
		record.DedupeKey, factPayload,
	)
	if err != nil {
		return fmt.Errorf("insert publication Event %q: %w", record.DedupeKey, err)
	}
	return nil
}

func (t *transaction) AdvanceStoredEventObservationTimes(ctx context.Context, eventID string, firstSeenAt, knowableAt time.Time) error {
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

func (t *transaction) StoredEventEvidenceLink(ctx context.Context, eventID, rawDocumentID string) (*eventbiz.StoredEventEvidenceLink, error) {
	var record eventbiz.StoredEventEvidenceLink
	var supportsFieldsJSON []byte
	err := t.tx.QueryRowContext(ctx, `
SELECT id, event_id, raw_document_id, source_level, evidence_statement, evidence_hash,
	   evidence_relation, array_to_json(supports_fields)
FROM event_sources
WHERE contract_version = 3 AND event_id = $1 AND raw_document_id = $2`,
		eventID, rawDocumentID,
	).Scan(
		&record.ID, &record.EventID, &record.RawDocumentID, &record.SourceLevel,
		&record.EvidenceStatement, &record.EvidenceHash, &record.EvidenceRelation,
		&supportsFieldsJSON,
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
	if err := validateStoredEvidenceLink(record, eventID, rawDocumentID); err != nil {
		return nil, fmt.Errorf("read Event Evidence Link invariant: %w", err)
	}
	return &record, nil
}

func (t *transaction) InsertStoredEventEvidenceLink(ctx context.Context, record eventbiz.StoredEventEvidenceLink) error {
	_, err := t.tx.ExecContext(ctx, `
INSERT INTO event_sources (
    id, contract_version, event_id, raw_document_id, source_level, evidence_statement,
	 evidence_hash, evidence_relation, supports_fields
) VALUES ($1,3,$2,$3,$4,$5,$6,$7,$8)`,
		record.ID, record.EventID, record.RawDocumentID, record.SourceLevel,
		record.EvidenceStatement, record.EvidenceHash, record.EvidenceRelation,
		record.SupportsFields,
	)
	if err != nil {
		return fmt.Errorf("insert publication Event Source %q/%q: %w", record.EventID, record.RawDocumentID, err)
	}
	return nil
}

func (t *transaction) PublicationTag(ctx context.Context, tagID string) (*eventbiz.EventTag, error) {
	var tag eventbiz.EventTag
	err := t.tx.QueryRowContext(ctx, `
SELECT id, tag_kind, code, name, is_active
FROM event_tag_defs
WHERE id = $1`, tagID).Scan(&tag.ID, &tag.Kind, &tag.Code, &tag.Name, &tag.Active)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read publication Tag %q: %w", tagID, err)
	}
	if err := validateStoredEventTag(tag, false); err != nil {
		return nil, fmt.Errorf("read Event Tag invariant: %w", err)
	}
	return &tag, nil
}

func (t *transaction) StoredEventTagAssignment(ctx context.Context, eventID, tagID string) (*eventbiz.StoredEventTagAssignment, error) {
	var record eventbiz.StoredEventTagAssignment
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
	if err := validateStoredTagAssignment(record, eventID, tagID); err != nil {
		return nil, fmt.Errorf("read Event Tag Assignment invariant: %w", err)
	}
	return &record, nil
}

func (t *transaction) InsertStoredEventTagAssignment(ctx context.Context, record eventbiz.StoredEventTagAssignment) error {
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

func (t *transaction) InsertEventPublicationReceipt(ctx context.Context, receipt eventbiz.EventPublicationReceipt) error {
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
		return fmt.Errorf("insert Event Publication receipt: %w", err)
	}
	return nil
}

var _ eventbiz.Transaction = (*transaction)(nil)
