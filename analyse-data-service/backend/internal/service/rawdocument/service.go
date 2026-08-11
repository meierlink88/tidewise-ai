package rawdocument

import (
	"context"
	"fmt"
	"strings"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	api "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1/rawdocument"
	biz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/rawdocument"
)

type Service struct{ useCase *biz.UseCase }

func NewService(useCase *biz.UseCase) (*Service, error) {
	if useCase == nil {
		return nil, fmt.Errorf("RawDocument use case is required")
	}
	return &Service{useCase: useCase}, nil
}

func (s *Service) List(ctx context.Context, request *api.ListRequest) (*v1.Response[api.Page], error) {
	page, err := v1.ParseBoundedInt(request.Page, 1, 1, 1_000_000, "page")
	if err != nil {
		return nil, err
	}
	pageSize, err := v1.ParseBoundedInt(request.PageSize, 50, 1, 100, "page_size")
	if err != nil {
		return nil, err
	}
	status, err := biz.ParseOptionalIngestStatus(request.IngestStatus)
	if err != nil {
		return nil, v1.NewPublicError(v1.StatusBadRequest, "INVALID_REQUEST", "unsupported ingest_status", nil)
	}
	result, err := s.useCase.List(ctx, biz.ListFilter{Title: strings.TrimSpace(request.Title), SourceRef: strings.TrimSpace(request.SourceRef), IngestStatus: status, Page: page, PageSize: pageSize})
	if err != nil {
		return nil, v1.NewPublicError(v1.StatusInternalServerError, "DATA_REPOSITORY_FAILURE", "admin raw-document aggregate failed", nil)
	}
	items := make([]api.Document, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, dto(item))
	}
	return &v1.Response[api.Page]{Status: v1.StatusOK, Result: api.Page{Items: items, Total: result.Total, Page: result.Page, PageSize: result.PageSize}}, nil
}

func dto(item biz.Document) api.Document {
	var publishedAt *string
	if item.PublishedAt != nil {
		value := item.PublishedAt.UTC().Format(time.RFC3339Nano)
		publishedAt = &value
	}
	return api.Document{ID: item.ID, ContractVersion: item.ContractVersion, ArtifactID: item.ArtifactID, SourceRef: item.SourceRef, IngestChannel: item.IngestChannel, SourceType: item.SourceType, SourceName: item.SourceName, SourceURL: item.SourceURL, SourceExternalID: item.SourceExternalID, Title: item.Title, ContentText: item.ContentText, ContentLevel: item.ContentLevel, RawObjectURI: item.RawObjectURI, RawMIMEType: item.RawMIMEType, Language: item.Language, PublishedAt: publishedAt, CollectedAt: item.CollectedAt.UTC().Format(time.RFC3339Nano), IngestStatus: string(item.IngestStatus), ContentSHA256: item.ContentHash}
}

var _ api.Service = (*Service)(nil)
