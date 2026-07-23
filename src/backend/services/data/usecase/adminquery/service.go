// Package adminquery owns Data Service queries exposed to the Admin Portal BFF.
package adminquery

import (
	"context"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
	"github.com/meierlink88/tidewise-ai/backend/services/data/repositories"
)

type RawDocumentListRequest struct {
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

type EventListRequest struct {
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

type Repository interface {
	ListRawDocuments(context.Context, repositories.RawDocumentListFilter) (repositories.RawDocumentPage, error)
	ListEvents(context.Context, repositories.EventListFilter) (repositories.EventPage, error)
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) ListRawDocuments(ctx context.Context, request RawDocumentListRequest) (RawDocumentPage, error) {
	page, err := s.repository.ListRawDocuments(ctx, repositories.RawDocumentListFilter{
		Title: request.Title, SourceRef: request.SourceRef, IngestStatus: request.IngestStatus, Page: request.Page, PageSize: request.PageSize,
	})
	if err != nil {
		return RawDocumentPage{}, err
	}
	return RawDocumentPage{Items: page.Items, Total: page.Total, Page: page.Page, PageSize: page.PageSize}, nil
}

func (s *Service) ListEvents(ctx context.Context, request EventListRequest) (EventPage, error) {
	page, err := s.repository.ListEvents(ctx, repositories.EventListFilter{
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
