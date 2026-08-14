package rawdocument

import (
	"context"
	"database/sql"
	"fmt"

	biz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/rawdocument"
)

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, fmt.Errorf("RawDocument database is required")
	}
	return &Store{db: db}, nil
}

func (s *Store) List(ctx context.Context, filter biz.ListFilter) (biz.StorePage, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM raw_documents WHERE ($1 = '' OR title ILIKE '%' || $1 || '%') AND ($2 = '' OR source_ref = $2) AND ($3 = '' OR ingest_status = $3)`, filter.Title, filter.SourceRef, string(filter.IngestStatus)).Scan(&total); err != nil {
		return biz.StorePage{}, fmt.Errorf("count raw documents: %w", err)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, contract_version, artifact_id, source_ref, ingest_channel, source_type, source_name, source_url, source_external_id, title, content_text, content_level, raw_object_uri, raw_mime_type, language, published_at, collected_at, content_hash, ingest_status FROM raw_documents WHERE ($1 = '' OR title ILIKE '%' || $1 || '%') AND ($2 = '' OR source_ref = $2) AND ($3 = '' OR ingest_status = $3) ORDER BY collected_at DESC, id LIMIT $4 OFFSET $5`, filter.Title, filter.SourceRef, string(filter.IngestStatus), pageSize, (page-1)*pageSize)
	if err != nil {
		return biz.StorePage{}, fmt.Errorf("query raw documents: %w", err)
	}
	defer rows.Close()
	items := make([]biz.Document, 0)
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return biz.StorePage{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return biz.StorePage{}, fmt.Errorf("iterate raw documents: %w", err)
	}
	return biz.StorePage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

type scanner interface{ Scan(...any) error }

func scan(row scanner) (biz.Document, error) {
	var item biz.Document
	var artifactID, sourceRef, sourceExternalID sql.NullString
	var publishedAt sql.NullTime
	if err := row.Scan(&item.ID, &item.ContractVersion, &artifactID, &sourceRef, &item.IngestChannel, &item.SourceType, &item.SourceName, &item.SourceURL, &sourceExternalID, &item.Title, &item.ContentText, &item.ContentLevel, &item.RawObjectURI, &item.RawMIMEType, &item.Language, &publishedAt, &item.CollectedAt, &item.ContentHash, &item.IngestStatus); err != nil {
		return biz.Document{}, fmt.Errorf("scan raw document: %w", err)
	}
	if artifactID.Valid {
		item.ArtifactID = artifactID.String
	}
	if sourceRef.Valid {
		item.SourceRef = sourceRef.String
	}
	if sourceExternalID.Valid {
		item.SourceExternalID = sourceExternalID.String
	}
	if publishedAt.Valid {
		item.PublishedAt = &publishedAt.Time
	}
	if err := item.Validate(); err != nil {
		return biz.Document{}, fmt.Errorf("validate persisted raw document: %w", err)
	}
	return item, nil
}

func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	return page, pageSize
}

var _ biz.Store = (*Store)(nil)
