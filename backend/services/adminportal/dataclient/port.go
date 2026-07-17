package dataclient

import (
	"context"
	"errors"
	"time"
)

const (
	DataAPIPrefix           = "/internal/data/v1"
	AdminRawDocumentsPath   = DataAPIPrefix + "/admin/raw-documents"
	AdminEventsPath         = DataAPIPrefix + "/admin/events"
	AdminSourceCatalogsPath = DataAPIPrefix + "/admin/source-catalogs"
)

// DataServiceClient is the Admin-owned boundary for the retained management
// aggregates. Scheduler control is deliberately absent from this port.
type DataServiceClient interface {
	ListRawDocuments(context.Context, RawDocumentListQuery) (RawDocumentPage, error)
	ListEvents(context.Context, EventListQuery) (EventPage, error)
	ListSourceCatalogs(context.Context, SourceCatalogListQuery) (SourceCatalogCollection, error)
}

type RawDocumentListQuery struct {
	Title        string
	SourceID     string
	IngestStatus IngestStatus
	Page         int
	PageSize     int
}

type EventListQuery struct {
	Title         string
	EventStatus   EventStatus
	FactStatus    FactStatus
	EventTimeFrom *time.Time
	EventTimeTo   *time.Time
	FirstSeenFrom *time.Time
	FirstSeenTo   *time.Time
	Page          int
	PageSize      int
}

type SourceCatalogListQuery struct {
	Status SourceStatus
}

type IngestStatus string

const (
	IngestStatusCollected      IngestStatus = "collected"
	IngestStatusDuplicate      IngestStatus = "duplicate"
	IngestStatusFailed         IngestStatus = "failed"
	IngestStatusPendingExtract IngestStatus = "pending_extract"
)

type EventStatus string

const (
	EventStatusCandidate EventStatus = "candidate"
	EventStatusConfirmed EventStatus = "confirmed"
	EventStatusRejected  EventStatus = "rejected"
)

type FactStatus string

const (
	FactStatusUnverified FactStatus = "unverified"
	FactStatusVerified   FactStatus = "verified"
	FactStatusDisputed   FactStatus = "disputed"
)

type SourceStatus string

const (
	SourceStatusActive   SourceStatus = "active"
	SourceStatusInactive SourceStatus = "inactive"
	SourceStatusDisabled SourceStatus = "disabled"
)

type RawDocumentPage struct {
	Items    []RawDocument `json:"items"`
	Total    int           `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}

type RawDocument struct {
	ID               string       `json:"id"`
	SourceID         string       `json:"source_id"`
	IngestChannel    string       `json:"ingest_channel"`
	SourceType       string       `json:"source_type"`
	SourceName       string       `json:"source_name"`
	SourceURL        string       `json:"source_url"`
	SourceExternalID string       `json:"source_external_id,omitempty"`
	Title            string       `json:"title"`
	ContentText      string       `json:"content_text"`
	ContentLevel     string       `json:"content_level"`
	RawObjectURI     string       `json:"raw_object_uri"`
	RawMIMEType      string       `json:"raw_mime_type"`
	Language         string       `json:"language"`
	PublishedAt      *time.Time   `json:"published_at,omitempty"`
	CollectedAt      time.Time    `json:"collected_at"`
	IngestStatus     IngestStatus `json:"ingest_status"`
}

type EventPage struct {
	Items    []Event `json:"items"`
	Total    int     `json:"total"`
	Page     int     `json:"page"`
	PageSize int     `json:"page_size"`
}

type Event struct {
	ID              string      `json:"id"`
	Title           string      `json:"title"`
	Summary         string      `json:"summary"`
	EventTime       *time.Time  `json:"event_time,omitempty"`
	FirstSeenAt     time.Time   `json:"first_seen_at"`
	KnowableAt      *time.Time  `json:"knowable_at,omitempty"`
	EventStatus     EventStatus `json:"event_status"`
	FactStatus      FactStatus  `json:"fact_status"`
	DedupeKey       string      `json:"dedupe_key"`
	PrimarySourceID *string     `json:"primary_source_id,omitempty"`
}

type SourceCatalogCollection struct {
	Items []SourceCatalog `json:"items"`
}

type SourceCatalog struct {
	ID            string       `json:"id"`
	IngestChannel string       `json:"ingest_channel"`
	ProviderKey   string       `json:"provider_key"`
	ConnectorKey  string       `json:"connector_key"`
	ParserKey     string       `json:"parser_key"`
	SourceType    string       `json:"source_type"`
	SourceName    string       `json:"source_name"`
	SourceURL     string       `json:"source_url"`
	SourceLevel   string       `json:"source_level"`
	TopicHint     string       `json:"topic_hint"`
	UsagePolicy   string       `json:"usage_policy"`
	Status        SourceStatus `json:"status"`
}

var ErrFakeMethodNotConfigured = errors.New("data service fake method is not configured")

// Fake keeps Admin orchestration tests independent from HTTP and databases.
type Fake struct {
	ListRawDocumentsFunc   func(context.Context, RawDocumentListQuery) (RawDocumentPage, error)
	ListEventsFunc         func(context.Context, EventListQuery) (EventPage, error)
	ListSourceCatalogsFunc func(context.Context, SourceCatalogListQuery) (SourceCatalogCollection, error)
}

func (f *Fake) ListRawDocuments(ctx context.Context, query RawDocumentListQuery) (RawDocumentPage, error) {
	if f == nil || f.ListRawDocumentsFunc == nil {
		return RawDocumentPage{}, ErrFakeMethodNotConfigured
	}
	return f.ListRawDocumentsFunc(ctx, query)
}

func (f *Fake) ListEvents(ctx context.Context, query EventListQuery) (EventPage, error) {
	if f == nil || f.ListEventsFunc == nil {
		return EventPage{}, ErrFakeMethodNotConfigured
	}
	return f.ListEventsFunc(ctx, query)
}

func (f *Fake) ListSourceCatalogs(ctx context.Context, query SourceCatalogListQuery) (SourceCatalogCollection, error) {
	if f == nil || f.ListSourceCatalogsFunc == nil {
		return SourceCatalogCollection{}, ErrFakeMethodNotConfigured
	}
	return f.ListSourceCatalogsFunc(ctx, query)
}
