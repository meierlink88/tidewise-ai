package biz

import (
	"context"
	"errors"
	"time"
)

// DataServiceRepo is the Admin-owned boundary for the retained management
// aggregates. Scheduler control is deliberately absent from this port.
type DataServiceRepo interface {
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
	Items    []RawDocument
	Total    int
	Page     int
	PageSize int
}

type RawDocument struct {
	ID               string
	ContractVersion  int
	ArtifactID       string
	SourceRef        string
	IngestChannel    string
	SourceType       string
	SourceName       string
	SourceURL        string
	SourceExternalID string
	Title            string
	ContentText      string
	ContentLevel     string
	RawObjectURI     string
	RawMIMEType      string
	Language         string
	PublishedAt      *time.Time
	CollectedAt      time.Time
	IngestStatus     IngestStatus
	ContentSHA256    string
}

type EventPage struct {
	Items    []Event
	Total    int
	Page     int
	PageSize int
}

type Event struct {
	ID          string
	Title       string
	Summary     string
	EventTime   *time.Time
	FirstSeenAt time.Time
	KnowableAt  *time.Time
	EventStatus EventStatus
	FactStatus  FactStatus
	DedupeKey   string
}

var ErrFakeMethodNotConfigured = errors.New("data service fake method is not configured")

// FakeDataServiceRepo keeps Admin orchestration tests independent from HTTP and databases.
type FakeDataServiceRepo struct {
	ListRawDocumentsFunc func(context.Context, RawDocumentListQuery) (RawDocumentPage, error)
	ListEventsFunc       func(context.Context, EventListQuery) (EventPage, error)
}

func (f *FakeDataServiceRepo) ListRawDocuments(ctx context.Context, query RawDocumentListQuery) (RawDocumentPage, error) {
	if f == nil || f.ListRawDocumentsFunc == nil {
		return RawDocumentPage{}, ErrFakeMethodNotConfigured
	}
	return f.ListRawDocumentsFunc(ctx, query)
}

func (f *FakeDataServiceRepo) ListEvents(ctx context.Context, query EventListQuery) (EventPage, error) {
	if f == nil || f.ListEventsFunc == nil {
		return EventPage{}, ErrFakeMethodNotConfigured
	}
	return f.ListEventsFunc(ctx, query)
}
