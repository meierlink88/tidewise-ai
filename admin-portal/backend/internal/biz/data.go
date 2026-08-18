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
	ListEventsFunc func(context.Context, EventListQuery) (EventPage, error)
}

func (f *FakeDataServiceRepo) ListEvents(ctx context.Context, query EventListQuery) (EventPage, error) {
	if f == nil || f.ListEventsFunc == nil {
		return EventPage{}, ErrFakeMethodNotConfigured
	}
	return f.ListEventsFunc(ctx, query)
}
