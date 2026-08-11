package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/adminquery"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/model"
)

type RawDocumentListFilter = adminquery.RawDocumentListFilter
type RawDocumentPage = adminquery.RawDocumentStorePage
type AdminQueryRepository = adminquery.Repository

func (r *InMemoryRepository) ListRawDocuments(_ context.Context, filter RawDocumentListFilter) (RawDocumentPage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	title := strings.ToLower(strings.TrimSpace(filter.Title))
	items := make([]model.RawDocument, 0, len(r.documents))
	for _, doc := range r.documents {
		if title != "" && !strings.Contains(strings.ToLower(doc.Title), title) {
			continue
		}
		if filter.SourceRef != "" && doc.SourceRef != filter.SourceRef {
			continue
		}
		if filter.IngestStatus != "" && doc.IngestStatus != filter.IngestStatus {
			continue
		}
		items = append(items, cloneRawDocument(doc))
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CollectedAt.After(items[j].CollectedAt)
	})
	total := len(items)
	items = pageSlice(items, page, pageSize)
	return RawDocumentPage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func cloneRawDocument(document model.RawDocument) model.RawDocument {
	if document.PublishedAt != nil {
		value := *document.PublishedAt
		document.PublishedAt = &value
	}
	return document
}

func scanRawDocument(scanner rawDocumentScanner) (model.RawDocument, error) {
	var document model.RawDocument
	var artifactID sql.NullString
	var sourceRef sql.NullString
	var sourceExternalID sql.NullString
	var publishedAt sql.NullTime
	if err := scanner.Scan(
		&document.ID,
		&document.ContractVersion,
		&artifactID,
		&sourceRef,
		&document.IngestChannel,
		&document.SourceType,
		&document.SourceName,
		&document.SourceURL,
		&sourceExternalID,
		&document.Title,
		&document.ContentText,
		&document.ContentLevel,
		&document.RawObjectURI,
		&document.RawMIMEType,
		&document.Language,
		&publishedAt,
		&document.CollectedAt,
		&document.ContentHash,
		&document.IngestStatus,
	); err != nil {
		return model.RawDocument{}, err
	}
	if artifactID.Valid {
		document.ArtifactID = artifactID.String
	}
	if sourceRef.Valid {
		document.SourceRef = sourceRef.String
	}
	if sourceExternalID.Valid {
		document.SourceExternalID = sourceExternalID.String
	}
	if publishedAt.Valid {
		document.PublishedAt = &publishedAt.Time
	}
	return document, nil
}

func normalizePage(page int, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	return page, pageSize
}

func pageSlice[T any](items []T, page int, pageSize int) []T {
	start := (page - 1) * pageSize
	if start >= len(items) {
		return []T{}
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}

func (r repository) ListRawDocuments(ctx context.Context, filter RawDocumentListFilter) (RawDocumentPage, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	var total int
	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM raw_documents
WHERE ($1 = '' OR title ILIKE '%' || $1 || '%')
  AND ($2 = '' OR source_ref = $2)
  AND ($3 = '' OR ingest_status = $3)
`, filter.Title, filter.SourceRef, string(filter.IngestStatus)).Scan(&total); err != nil {
		return RawDocumentPage{}, fmt.Errorf("count raw documents: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
SELECT id, contract_version, artifact_id, source_ref, ingest_channel, source_type, source_name, source_url,
       source_external_id, title, content_text, content_level, raw_object_uri, raw_mime_type,
       language, published_at, collected_at, content_hash, ingest_status
FROM raw_documents
WHERE ($1 = '' OR title ILIKE '%' || $1 || '%')
  AND ($2 = '' OR source_ref = $2)
  AND ($3 = '' OR ingest_status = $3)
ORDER BY collected_at DESC, id
LIMIT $4 OFFSET $5
`, filter.Title, filter.SourceRef, string(filter.IngestStatus), pageSize, (page-1)*pageSize)
	if err != nil {
		return RawDocumentPage{}, fmt.Errorf("query raw documents: %w", err)
	}
	defer rows.Close()

	items := make([]model.RawDocument, 0)
	for rows.Next() {
		doc, err := scanRawDocument(rows)
		if err != nil {
			return RawDocumentPage{}, err
		}
		items = append(items, doc)
	}
	if err := rows.Err(); err != nil {
		return RawDocumentPage{}, fmt.Errorf("iterate raw documents: %w", err)
	}
	return RawDocumentPage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}
