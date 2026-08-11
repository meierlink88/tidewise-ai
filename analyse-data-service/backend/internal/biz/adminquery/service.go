// Package adminquery owns Data Service queries exposed to the Admin Portal BFF.
package adminquery

import (
	"context"

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
