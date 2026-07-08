package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) PostgresRepository {
	return PostgresRepository{db: db}
}

func (r PostgresRepository) SeedSource(ctx context.Context, source domain.SourceCatalog) error {
	source = normalizeSource(source)
	policy, err := json.Marshal(source.RateLimitPolicy)
	if err != nil {
		return fmt.Errorf("marshal rate limit policy: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
INSERT INTO source_catalogs (
    id, ingest_channel, provider_key, connector_key, parser_key, source_type,
    source_name, source_url, source_level, topic_hint, route_template, code_style,
    auth_required, auth_type, credential_ref, rate_limit_policy, usage_policy, status
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11, $12,
    $13, $14, $15, $16::jsonb, $17, $18
) ON CONFLICT (id) DO UPDATE SET
    ingest_channel = EXCLUDED.ingest_channel,
    provider_key = EXCLUDED.provider_key,
    connector_key = EXCLUDED.connector_key,
    parser_key = EXCLUDED.parser_key,
    source_type = EXCLUDED.source_type,
    source_name = EXCLUDED.source_name,
    source_url = EXCLUDED.source_url,
    source_level = EXCLUDED.source_level,
    topic_hint = EXCLUDED.topic_hint,
    route_template = EXCLUDED.route_template,
    code_style = EXCLUDED.code_style,
    auth_required = EXCLUDED.auth_required,
    auth_type = EXCLUDED.auth_type,
    credential_ref = EXCLUDED.credential_ref,
    rate_limit_policy = EXCLUDED.rate_limit_policy,
    usage_policy = EXCLUDED.usage_policy,
    status = EXCLUDED.status,
    updated_at = now()
`, source.ID, source.IngestChannel, source.ProviderKey, source.ConnectorKey, source.ParserKey, source.SourceType,
		source.SourceName, source.SourceURL, source.SourceLevel, source.TopicHint, source.RouteTemplate, source.CodeStyle,
		source.AuthRequired, source.AuthType, source.CredentialRef, string(policy), source.UsagePolicy, source.Status)
	if err != nil {
		return fmt.Errorf("seed source catalog: %w", err)
	}

	return nil
}

func (r PostgresRepository) ActiveSources(ctx context.Context, filter SourceCatalogFilter) ([]domain.SourceCatalog, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, ingest_channel, provider_key, connector_key, parser_key, source_type,
       source_name, source_url, source_level, topic_hint, route_template, code_style,
       auth_required, auth_type, credential_ref, rate_limit_policy, usage_policy, status
FROM source_catalogs
WHERE status = 'active'
  AND ($1 = '' OR provider_key = $1)
  AND ($2 = '' OR ingest_channel = $2)
ORDER BY source_name, id
`, filter.ProviderKey, filter.IngestChannel)
	if err != nil {
		return nil, fmt.Errorf("query active source catalogs: %w", err)
	}
	defer rows.Close()

	var sources []domain.SourceCatalog
	for rows.Next() {
		source, err := scanSource(rows)
		if err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate source catalogs: %w", err)
	}

	return sources, nil
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
    source_external_id, title, content_text, raw_object_uri, raw_mime_type,
    language, published_at, collected_at, content_hash, ingest_status
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11,
    $12, $13, $14, $15, $16
)
`, doc.ID, doc.SourceID, doc.IngestChannel, doc.SourceType, doc.SourceName, doc.SourceURL,
		nullString(doc.SourceExternalID), doc.Title, doc.ContentText, doc.RawObjectURI, doc.RawMIMEType,
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
       source_external_id, title, content_text, raw_object_uri, raw_mime_type,
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

type sourceScanner interface {
	Scan(dest ...any) error
}

func scanSource(scanner sourceScanner) (domain.SourceCatalog, error) {
	var source domain.SourceCatalog
	var policyBytes []byte
	if err := scanner.Scan(
		&source.ID,
		&source.IngestChannel,
		&source.ProviderKey,
		&source.ConnectorKey,
		&source.ParserKey,
		&source.SourceType,
		&source.SourceName,
		&source.SourceURL,
		&source.SourceLevel,
		&source.TopicHint,
		&source.RouteTemplate,
		&source.CodeStyle,
		&source.AuthRequired,
		&source.AuthType,
		&source.CredentialRef,
		&policyBytes,
		&source.UsagePolicy,
		&source.Status,
	); err != nil {
		return domain.SourceCatalog{}, fmt.Errorf("scan source catalog: %w", err)
	}
	if len(policyBytes) > 0 {
		if err := json.Unmarshal(policyBytes, &source.RateLimitPolicy); err != nil {
			return domain.SourceCatalog{}, fmt.Errorf("parse rate limit policy: %w", err)
		}
	}
	if source.RateLimitPolicy == nil {
		source.RateLimitPolicy = map[string]any{}
	}
	return source, nil
}

type rawDocumentScanner interface {
	Scan(dest ...any) error
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

func normalizeSource(source domain.SourceCatalog) domain.SourceCatalog {
	source.ID = NormalizeUUID(source.ID)
	if source.SourceLevel == "" {
		source.SourceLevel = "secondary"
	}
	if source.AuthType == "" {
		source.AuthType = "none"
	}
	if source.Status == "" {
		source.Status = domain.SourceCatalogStatusActive
	}
	if source.RateLimitPolicy == nil {
		source.RateLimitPolicy = map[string]any{}
	}
	return source
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
