package adminquery

import (
	"context"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/model"
)

type RawDocumentListFilter struct {
	Title, SourceRef string
	IngestStatus     model.IngestStatus
	Page, PageSize   int
}

type RawDocumentStorePage struct {
	Items          []model.RawDocument
	Total          int
	Page, PageSize int
}

type EventListFilter struct {
	Title                      string
	EventStatus                model.EventStatus
	FactStatus                 model.FactStatus
	EventTimeFrom, EventTimeTo *time.Time
	FirstSeenFrom, FirstSeenTo *time.Time
	Page, PageSize             int
}

type EventStorePage struct {
	Items          []model.Event
	Total          int
	Page, PageSize int
}

type Repository interface {
	ListRawDocuments(context.Context, RawDocumentListFilter) (RawDocumentStorePage, error)
	ListEvents(context.Context, EventListFilter) (EventStorePage, error)
}
