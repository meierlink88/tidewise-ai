// Package adminquery owns Data Service queries exposed to the Admin Portal BFF.
package adminquery

import (
	"context"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/model"
)

type RawDocumentListRequest struct {
	Title        string
	SourceRef    string
	IngestStatus model.IngestStatus
	Page         int
	PageSize     int
}

type RawDocumentPage struct {
	Items    []model.RawDocument
	Total    int
	Page     int
	PageSize int
}

type EventListRequest struct {
	Title         string
	EventStatus   model.EventStatus
	FactStatus    model.FactStatus
	EventTimeFrom *time.Time
	EventTimeTo   *time.Time
	FirstSeenFrom *time.Time
	FirstSeenTo   *time.Time
	Page          int
	PageSize      int
}

type EventPage struct {
	Items    []model.Event
	Total    int
	Page     int
	PageSize int
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) ListRawDocuments(ctx context.Context, request RawDocumentListRequest) (RawDocumentPage, error) {
	page, err := s.repository.ListRawDocuments(ctx, RawDocumentListFilter{
		Title: request.Title, SourceRef: request.SourceRef, IngestStatus: request.IngestStatus, Page: request.Page, PageSize: request.PageSize,
	})
	if err != nil {
		return RawDocumentPage{}, err
	}
	return RawDocumentPage{Items: page.Items, Total: page.Total, Page: page.Page, PageSize: page.PageSize}, nil
}

func (s *Service) ListEvents(ctx context.Context, request EventListRequest) (EventPage, error) {
	page, err := s.repository.ListEvents(ctx, EventListFilter{
		Title: request.Title, EventStatus: request.EventStatus, FactStatus: request.FactStatus,
		EventTimeFrom: request.EventTimeFrom, EventTimeTo: request.EventTimeTo,
		FirstSeenFrom: request.FirstSeenFrom, FirstSeenTo: request.FirstSeenTo,
		Page: request.Page, PageSize: request.PageSize,
	})
	if err != nil {
		return EventPage{}, err
	}
	return EventPage{Items: page.Items, Total: page.Total, Page: page.Page, PageSize: page.PageSize}, nil
}
