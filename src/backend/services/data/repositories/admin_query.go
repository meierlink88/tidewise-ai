package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
)

type RawDocumentListFilter struct {
	Title        string
	SourceRef    string
	IngestStatus domain.IngestStatus
	Page         int
	PageSize     int
}

type RawDocumentPage struct {
	Items    []domain.RawDocument
	Total    int
	Page     int
	PageSize int
}

type EventListFilter struct {
	Title         string
	EventStatus   domain.EventStatus
	FactStatus    domain.FactStatus
	EventTimeFrom *time.Time
	EventTimeTo   *time.Time
	FirstSeenFrom *time.Time
	FirstSeenTo   *time.Time
	Page          int
	PageSize      int
}

type EventPage struct {
	Items    []domain.Event
	Total    int
	Page     int
	PageSize int
}

type AdminQueryRepository interface {
	ListRawDocuments(context.Context, RawDocumentListFilter) (RawDocumentPage, error)
	ListEvents(context.Context, EventListFilter) (EventPage, error)
}

func (r *InMemoryRepository) SeedEvent(_ context.Context, event domain.Event) error {
	if err := event.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.events[event.ID] = cloneEvent(event)
	return nil
}

func (r *InMemoryRepository) ListRawDocuments(_ context.Context, filter RawDocumentListFilter) (RawDocumentPage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	title := strings.ToLower(strings.TrimSpace(filter.Title))
	items := make([]domain.RawDocument, 0, len(r.documents))
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

func (r *InMemoryRepository) ListEvents(_ context.Context, filter EventListFilter) (EventPage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	title := strings.ToLower(strings.TrimSpace(filter.Title))
	items := make([]domain.Event, 0, len(r.events))
	for _, event := range r.events {
		if title != "" && !strings.Contains(strings.ToLower(event.Title), title) {
			continue
		}
		if filter.EventStatus != "" && event.EventStatus != filter.EventStatus {
			continue
		}
		if filter.FactStatus != "" && event.FactStatus != filter.FactStatus {
			continue
		}
		if filter.EventTimeFrom != nil {
			if event.EventTime == nil || event.EventTime.Before(*filter.EventTimeFrom) {
				continue
			}
		}
		if filter.EventTimeTo != nil {
			if event.EventTime == nil || event.EventTime.After(*filter.EventTimeTo) {
				continue
			}
		}
		if filter.FirstSeenFrom != nil && event.FirstSeenAt.Before(*filter.FirstSeenFrom) {
			continue
		}
		if filter.FirstSeenTo != nil && event.FirstSeenAt.After(*filter.FirstSeenTo) {
			continue
		}
		items = append(items, cloneEvent(event))
	}
	sort.Slice(items, func(i, j int) bool {
		if !items[i].FirstSeenAt.Equal(items[j].FirstSeenAt) {
			return items[i].FirstSeenAt.After(items[j].FirstSeenAt)
		}
		if items[i].EventTime == nil {
			return false
		}
		if items[j].EventTime == nil {
			return true
		}
		return items[i].EventTime.After(*items[j].EventTime)
	})
	total := len(items)
	items = pageSlice(items, page, pageSize)
	return EventPage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func cloneEvent(event domain.Event) domain.Event {
	if event.EventTime != nil {
		value := *event.EventTime
		event.EventTime = &value
	}
	if event.KnowableAt != nil {
		value := *event.KnowableAt
		event.KnowableAt = &value
	}
	event.FactPayload = cloneFactPayload(event.FactPayload)
	return event
}

func cloneRawDocument(document domain.RawDocument) domain.RawDocument {
	if document.PublishedAt != nil {
		value := *document.PublishedAt
		document.PublishedAt = &value
	}
	return document
}

func scanRawDocument(scanner rawDocumentScanner) (domain.RawDocument, error) {
	var document domain.RawDocument
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
		return domain.RawDocument{}, err
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

func (r PostgresRepository) ListRawDocuments(ctx context.Context, filter RawDocumentListFilter) (RawDocumentPage, error) {
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

	items := make([]domain.RawDocument, 0)
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

func (r PostgresRepository) ListEvents(ctx context.Context, filter EventListFilter) (EventPage, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	var total int
	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM events
WHERE ($1 = '' OR title ILIKE '%' || $1 || '%')
  AND ($2 = '' OR event_status = $2)
  AND ($3 = '' OR fact_status = $3)
  AND ($4::timestamptz IS NULL OR event_time >= $4)
  AND ($5::timestamptz IS NULL OR event_time <= $5)
  AND ($6::timestamptz IS NULL OR first_seen_at >= $6)
  AND ($7::timestamptz IS NULL OR first_seen_at <= $7)
`, filter.Title, string(filter.EventStatus), string(filter.FactStatus), nullTime(filter.EventTimeFrom), nullTime(filter.EventTimeTo), nullTime(filter.FirstSeenFrom), nullTime(filter.FirstSeenTo)).Scan(&total); err != nil {
		return EventPage{}, fmt.Errorf("count events: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
SELECT id, title, summary, event_time, first_seen_at, knowable_at,
       event_status, fact_status, dedupe_key, primary_source_id
FROM events
WHERE ($1 = '' OR title ILIKE '%' || $1 || '%')
  AND ($2 = '' OR event_status = $2)
  AND ($3 = '' OR fact_status = $3)
  AND ($4::timestamptz IS NULL OR event_time >= $4)
  AND ($5::timestamptz IS NULL OR event_time <= $5)
  AND ($6::timestamptz IS NULL OR first_seen_at >= $6)
  AND ($7::timestamptz IS NULL OR first_seen_at <= $7)
ORDER BY first_seen_at DESC, event_time DESC NULLS LAST, id
LIMIT $8 OFFSET $9
`, filter.Title, string(filter.EventStatus), string(filter.FactStatus), nullTime(filter.EventTimeFrom), nullTime(filter.EventTimeTo), nullTime(filter.FirstSeenFrom), nullTime(filter.FirstSeenTo), pageSize, (page-1)*pageSize)
	if err != nil {
		return EventPage{}, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Event, 0)
	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
			return EventPage{}, err
		}
		items = append(items, event)
	}
	if err := rows.Err(); err != nil {
		return EventPage{}, fmt.Errorf("iterate events: %w", err)
	}
	return EventPage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func scanEvent(scanner rawDocumentScanner) (domain.Event, error) {
	var event domain.Event
	var eventTime sql.NullTime
	var knowableAt sql.NullTime
	var primarySourceID sql.NullString
	if err := scanner.Scan(
		&event.ID,
		&event.Title,
		&event.Summary,
		&eventTime,
		&event.FirstSeenAt,
		&knowableAt,
		&event.EventStatus,
		&event.FactStatus,
		&event.DedupeKey,
		&primarySourceID,
	); err != nil {
		return domain.Event{}, fmt.Errorf("scan event: %w", err)
	}
	if eventTime.Valid {
		event.EventTime = &eventTime.Time
	}
	if knowableAt.Valid {
		event.KnowableAt = &knowableAt.Time
	}
	if primarySourceID.Valid {
		event.PrimarySourceID = primarySourceID.String
	}
	return event, nil
}
