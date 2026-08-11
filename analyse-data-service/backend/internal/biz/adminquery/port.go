package adminquery

import (
	"context"

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

type Repository interface {
	ListRawDocuments(context.Context, RawDocumentListFilter) (RawDocumentStorePage, error)
}
