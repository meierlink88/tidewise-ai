package biz

import (
	"context"
	"errors"
	"time"
)

// DataServiceRepo is the Admin-owned boundary for the retained management
// aggregates. Scheduler control is deliberately absent from this port.
type DataServiceRepo interface {
	ListEvents(context.Context, EventListQuery) (EventPage, error)
	ListEvidences(context.Context, EvidenceListQuery) (EvidencePage, error)
	GetRawEvidenceDocument(context.Context, string) (RawEvidenceDocument, error)
	ListEvidenceCategories(context.Context) ([]EvidenceCategory, error)
	ListSources(context.Context) ([]Source, error)
}

type RawEvidenceDocument struct {
	RawText string
}

type CollectionDocument struct {
	Available bool
	URL       string
}

type EvidenceListQuery struct {
	Title, Summary, CategoryID, SourceID, SourceName, SourceLevel string
	IsSplit                                                       *bool
	PublishedFrom, PublishedTo, CollectedFrom, CollectedTo        *time.Time
	Page, PageSize                                                int
}

type EvidencePage struct {
	Items                 []Evidence
	Total, Page, PageSize int
}
type Evidence struct {
	ID, RawEvidenceID       string
	Title                   *string
	Summary                 string
	Semantic                EvidenceSemantic
	Categories              []EvidenceCategory
	SourceID                string
	SourceName, SourceLevel string
	SourceURL               string
	IsOriginal              bool
	QuotedSourceName        *string
	Keywords                []string
	IsSplit                 bool
	PublishedAt             *time.Time
	CollectedAt             time.Time
}
type EvidenceSemantic struct {
	Who   *string
	What  string
	When  *string
	Where *string
	Why   *string
	How   *string
}
type EvidenceCategory struct{ ID, Code, Name, Description string }

type SourceListQuery struct {
	Text, OwnershipType, ChannelType, DefaultSourceLevel string
	Enabled                                              *bool
	Priority                                             *int
	UpdatedFrom, UpdatedTo                               *time.Time
	Page, PageSize                                       int
}
type SourcePage struct {
	Items                 []Source
	Total, Page, PageSize int
}
type Source struct {
	ID, Code, Name, OwnershipType, ChannelType, DefaultSourceLevel string
	Enabled                                                        bool
	Priority                                                       int
	UpdatedAt                                                      time.Time
}

type EventListQuery struct {
	Title         string
	Modality      EventModality
	Status        EventLifecycleStatus
	OccurredFrom  *time.Time
	OccurredTo    *time.Time
	AnnouncedFrom *time.Time
	AnnouncedTo   *time.Time
	Page          int
	PageSize      int
}

type EventModality string

const (
	EventModalityFact EventModality = "FACT"
	EventModalityPlan EventModality = "PLAN"
	EventModalitySpec EventModality = "SPEC"
)

type EventLifecycleStatus string

const (
	EventLifecycleActive     EventLifecycleStatus = "ACTIVE"
	EventLifecycleDeprecated EventLifecycleStatus = "DEPRECATED"
	EventLifecycleArchived   EventLifecycleStatus = "ARCHIVED"
)

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
	Semantic    EventSemantic
	Modality    EventModality
	OccurredAt  *time.Time
	AnnouncedAt *time.Time
	Status      EventLifecycleStatus
}

type EventSemantic struct {
	Who   *string
	What  *string
	When  *string
	Where *string
	Why   *string
	How   *string
}

var ErrFakeMethodNotConfigured = errors.New("data service fake method is not configured")

// FakeDataServiceRepo keeps Admin orchestration tests independent from HTTP and databases.
type FakeDataServiceRepo struct {
	ListEventsFunc             func(context.Context, EventListQuery) (EventPage, error)
	ListEvidencesFunc          func(context.Context, EvidenceListQuery) (EvidencePage, error)
	GetRawEvidenceDocumentFunc func(context.Context, string) (RawEvidenceDocument, error)
	ListEvidenceCategoriesFunc func(context.Context) ([]EvidenceCategory, error)
	ListSourcesFunc            func(context.Context) ([]Source, error)
}

func (f *FakeDataServiceRepo) GetRawEvidenceDocument(ctx context.Context, id string) (RawEvidenceDocument, error) {
	if f == nil || f.GetRawEvidenceDocumentFunc == nil {
		return RawEvidenceDocument{}, ErrFakeMethodNotConfigured
	}
	return f.GetRawEvidenceDocumentFunc(ctx, id)
}

func (f *FakeDataServiceRepo) ListEvidences(ctx context.Context, query EvidenceListQuery) (EvidencePage, error) {
	if f == nil || f.ListEvidencesFunc == nil {
		return EvidencePage{}, ErrFakeMethodNotConfigured
	}
	return f.ListEvidencesFunc(ctx, query)
}

func (f *FakeDataServiceRepo) ListEvidenceCategories(ctx context.Context) ([]EvidenceCategory, error) {
	if f == nil || f.ListEvidenceCategoriesFunc == nil {
		return nil, ErrFakeMethodNotConfigured
	}
	return f.ListEvidenceCategoriesFunc(ctx)
}

func (f *FakeDataServiceRepo) ListSources(ctx context.Context) ([]Source, error) {
	if f == nil || f.ListSourcesFunc == nil {
		return nil, ErrFakeMethodNotConfigured
	}
	return f.ListSourcesFunc(ctx)
}

func (f *FakeDataServiceRepo) ListEvents(ctx context.Context, query EventListQuery) (EventPage, error) {
	if f == nil || f.ListEventsFunc == nil {
		return EventPage{}, ErrFakeMethodNotConfigured
	}
	return f.ListEventsFunc(ctx, query)
}
