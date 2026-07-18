package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

type RawDocumentRepository interface {
	UpsertRawDocument(context.Context, domain.RawDocument) (RawDocumentWriteResult, error)
	UpdateRawDocumentStatus(context.Context, string, domain.IngestStatus) error
}

type RawDocumentWriteResult struct {
	Document    domain.RawDocument
	Created     bool
	DuplicateOf string
}

func (r *InMemoryRepository) UpsertRawDocument(_ context.Context, doc domain.RawDocument) (RawDocumentWriteResult, error) {
	if err := doc.Validate(); err != nil {
		return RawDocumentWriteResult{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.findDuplicate(doc); ok {
		return RawDocumentWriteResult{
			Document:    existing,
			Created:     false,
			DuplicateOf: existing.ID,
		}, nil
	}

	r.documents[doc.ID] = doc
	return RawDocumentWriteResult{
		Document: doc,
		Created:  true,
	}, nil
}

func (r *InMemoryRepository) UpdateRawDocumentStatus(_ context.Context, id string, status domain.IngestStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	doc, ok := r.documents[id]
	if !ok {
		return fmt.Errorf("raw document %q not found", id)
	}
	doc.IngestStatus = status
	r.documents[id] = doc

	return nil
}

func (r *InMemoryRepository) RawDocument(id string) (domain.RawDocument, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	doc, ok := r.documents[id]
	return doc, ok
}

func (r *InMemoryRepository) RawDocumentCount(_ context.Context, sourceID string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	count := 0
	for _, doc := range r.documents {
		if sourceID == "" || doc.SourceID == sourceID {
			count++
		}
	}
	return count, nil
}

func (r *InMemoryRepository) findDuplicate(doc domain.RawDocument) (domain.RawDocument, bool) {
	for _, existing := range r.documents {
		if existing.SourceID != doc.SourceID {
			continue
		}
		if doc.SourceExternalID != "" && existing.SourceExternalID == doc.SourceExternalID {
			return existing, true
		}
		if existing.ContentHash == doc.ContentHash {
			return existing, true
		}
	}

	return domain.RawDocument{}, false
}

func cloneRawDocument(doc domain.RawDocument) domain.RawDocument {
	if doc.PublishedAt != nil {
		value := *doc.PublishedAt
		doc.PublishedAt = &value
	}
	return doc
}

func (r PostgresRepository) UpsertRawDocument(ctx context.Context, doc domain.RawDocument) (RawDocumentWriteResult, error) {
	doc = normalizeRawDocument(doc)
	if err := doc.Validate(); err != nil {
		return RawDocumentWriteResult{}, err
	}

	existing, ok, err := r.findDuplicate(ctx, doc)
	if err != nil {
		return RawDocumentWriteResult{}, err
	}
	if ok {
		return RawDocumentWriteResult{
			Document:    existing,
			Created:     false,
			DuplicateOf: existing.ID,
		}, nil
	}

	_, err = r.db.ExecContext(ctx, `
INSERT INTO raw_documents (
    id, source_id, ingest_channel, source_type, source_name, source_url,
    source_external_id, title, content_text, content_level, raw_object_uri, raw_mime_type,
    language, published_at, collected_at, content_hash, ingest_status
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11, $12,
    $13, $14, $15, $16, $17
)
`, doc.ID, doc.SourceID, doc.IngestChannel, doc.SourceType, doc.SourceName, doc.SourceURL,
		nullString(doc.SourceExternalID), doc.Title, doc.ContentText, doc.ContentLevel, doc.RawObjectURI, doc.RawMIMEType,
		doc.Language, nullTime(doc.PublishedAt), doc.CollectedAt, doc.ContentHash, doc.IngestStatus)
	if err != nil {
		return RawDocumentWriteResult{}, fmt.Errorf("insert raw document: %w", err)
	}

	return RawDocumentWriteResult{
		Document: doc,
		Created:  true,
	}, nil
}

func (r PostgresRepository) UpdateRawDocumentStatus(ctx context.Context, id string, status domain.IngestStatus) error {
	normalizedID := NormalizeUUID(id)
	result, err := r.db.ExecContext(ctx, `
UPDATE raw_documents
SET ingest_status = $2, updated_at = now()
WHERE id = $1
`, normalizedID, status)
	if err != nil {
		return fmt.Errorf("update raw document status: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("raw document %q not found", id)
	}
	return nil
}

func (r PostgresRepository) RawDocumentCount(ctx context.Context, sourceID string) (int, error) {
	var count int
	if sourceID == "" {
		err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM raw_documents
`).Scan(&count)
		if err != nil {
			return 0, fmt.Errorf("count raw documents: %w", err)
		}
		return count, nil
	}

	err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM raw_documents
WHERE source_id = $1
`, NormalizeUUID(sourceID)).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count raw documents: %w", err)
	}
	return count, nil
}

func (r PostgresRepository) findDuplicate(ctx context.Context, doc domain.RawDocument) (domain.RawDocument, bool, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, source_id, ingest_channel, source_type, source_name, source_url,
       source_external_id, title, content_text, content_level, raw_object_uri, raw_mime_type,
       language, published_at, collected_at, content_hash, ingest_status
FROM raw_documents
WHERE source_id = $1
  AND (
    ($2 <> '' AND source_external_id = $2)
    OR content_hash = $3
  )
LIMIT 1
`, doc.SourceID, doc.SourceExternalID, doc.ContentHash)

	existing, err := scanRawDocument(row)
	if err == sql.ErrNoRows {
		return domain.RawDocument{}, false, nil
	}
	if err != nil {
		return domain.RawDocument{}, false, fmt.Errorf("find duplicate raw document: %w", err)
	}

	return existing, true, nil
}

func scanRawDocument(scanner rawDocumentScanner) (domain.RawDocument, error) {
	var doc domain.RawDocument
	var sourceExternalID sql.NullString
	var publishedAt sql.NullTime
	if err := scanner.Scan(
		&doc.ID,
		&doc.SourceID,
		&doc.IngestChannel,
		&doc.SourceType,
		&doc.SourceName,
		&doc.SourceURL,
		&sourceExternalID,
		&doc.Title,
		&doc.ContentText,
		&doc.ContentLevel,
		&doc.RawObjectURI,
		&doc.RawMIMEType,
		&doc.Language,
		&publishedAt,
		&doc.CollectedAt,
		&doc.ContentHash,
		&doc.IngestStatus,
	); err != nil {
		return domain.RawDocument{}, err
	}
	if sourceExternalID.Valid {
		doc.SourceExternalID = sourceExternalID.String
	}
	if publishedAt.Valid {
		doc.PublishedAt = &publishedAt.Time
	}
	return doc, nil
}

func normalizeRawDocument(doc domain.RawDocument) domain.RawDocument {
	doc.SourceID = NormalizeUUID(doc.SourceID)
	doc.ID = RawDocumentUUID(doc.SourceID, doc.ID, doc.SourceExternalID, doc.ContentHash)
	if doc.CollectedAt.IsZero() {
		doc.CollectedAt = time.Now()
	}
	if doc.IngestStatus == "" {
		doc.IngestStatus = domain.IngestStatusCollected
	}
	return doc
}
