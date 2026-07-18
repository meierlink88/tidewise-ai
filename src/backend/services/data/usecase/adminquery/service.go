// Package adminquery owns Data Service queries exposed to the Admin Portal BFF.
package adminquery

import (
	"context"
	"errors"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
	"github.com/meierlink88/tidewise-ai/backend/services/data/repositories"
)

var ErrInvalidSourceID = errors.New("source id must be a UUID")

type RawDocumentListRequest struct {
	Title        string
	SourceID     string
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

type SourceCatalogListRequest struct {
	Status domain.SourceCatalogStatus
}

type Repository interface {
	ListRawDocuments(context.Context, repositories.RawDocumentListFilter) (repositories.RawDocumentPage, error)
	ListEvents(context.Context, repositories.EventListFilter) (repositories.EventPage, error)
	ListSourceCatalogs(context.Context, repositories.SourceCatalogListFilter) ([]domain.SourceCatalog, error)
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) ListRawDocuments(ctx context.Context, request RawDocumentListRequest) (RawDocumentPage, error) {
	if request.SourceID != "" && !repositories.IsUUID(request.SourceID) {
		return RawDocumentPage{}, ErrInvalidSourceID
	}
	page, err := s.repository.ListRawDocuments(ctx, repositories.RawDocumentListFilter{
		Title: request.Title, SourceID: request.SourceID, IngestStatus: request.IngestStatus, Page: request.Page, PageSize: request.PageSize,
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

func (s *Service) ListSourceCatalogs(ctx context.Context, request SourceCatalogListRequest) ([]domain.SourceCatalog, error) {
	return s.repository.ListSourceCatalogs(ctx, repositories.SourceCatalogListFilter{Status: request.Status})
}
