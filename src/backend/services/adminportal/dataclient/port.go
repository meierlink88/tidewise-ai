package dataclient

import (
	"context"
	"errors"
	"time"
)

const (
	DataAPIPrefix         = "/api/data/v1"
	AdminRawDocumentsPath = DataAPIPrefix + "/raw-documents"
	AdminEventsPath       = DataAPIPrefix + "/events"
)

// DataServiceClient is the Admin-owned boundary for the retained management
// aggregates. Scheduler control is deliberately absent from this port.
type DataServiceClient interface {
	ListRawDocuments(context.Context, RawDocumentListQuery) (RawDocumentPage, error)
	ListEvents(context.Context, EventListQuery) (EventPage, error)
}

type RawDocumentListQuery struct {
	Title        string
	SourceRef    string
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

type RawDocumentPage struct {
	Items    []RawDocument `json:"items"`
	Total    int           `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}

type RawDocument struct {
	ID               string       `json:"id"`
	ContractVersion  int          `json:"contract_version"`
	ArtifactID       string       `json:"artifact_id,omitempty"`
	SourceRef        string       `json:"source_ref,omitempty"`
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
	ContentSHA256    string       `json:"content_sha256"`
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

var ErrFakeMethodNotConfigured = errors.New("data service fake method is not configured")

// Fake keeps Admin orchestration tests independent from HTTP and databases.
type Fake struct {
	ListRawDocumentsFunc func(context.Context, RawDocumentListQuery) (RawDocumentPage, error)
	ListEventsFunc       func(context.Context, EventListQuery) (EventPage, error)
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
