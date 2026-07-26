package service

import (
	"context"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventtagcatalog"
)

func (s *DataService) ListActiveEventTags(ctx context.Context, request *v1.EventTagCatalogRequest) (*v1.Response[v1.EventTagCatalog], error) {
	if request == nil || !request.Active {
		return nil, publicError(v1.StatusBadRequest, "INVALID_REQUEST", "active must be exactly true")
	}
	if s == nil || s.dependencies.EventTagCatalog == nil {
		return nil, publicError(v1.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "Event Tag Catalog service is unavailable")
	}
	catalog, err := s.dependencies.EventTagCatalog.Active(ctx)
	if err != nil {
		return nil, publicError(v1.StatusInternalServerError, "EVENT_TAG_CATALOG_FAILED", "Event Tag Catalog is unavailable")
	}
	tags := make([]v1.EventTagCatalogItem, len(catalog.Tags))
	for position, tag := range catalog.Tags {
		tags[position] = eventTagCatalogItemDTO(tag)
	}
	return &v1.Response[v1.EventTagCatalog]{
		Status: v1.StatusOK,
		Result: v1.EventTagCatalog{
			CatalogRevision: catalog.Revision,
			CatalogHash:     catalog.Hash,
			Tags:            tags,
		},
	}, nil
}

func eventTagCatalogItemDTO(tag eventtagcatalog.Tag) v1.EventTagCatalogItem {
	return v1.EventTagCatalogItem{
		ID:       tag.ID,
		TagKind:  tag.Kind,
		Code:     tag.Code,
		Name:     tag.Name,
		IsActive: tag.Active,
	}
}
