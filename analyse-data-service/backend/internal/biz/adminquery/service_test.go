package adminquery

import (
	"context"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/model"
)

func TestServiceMapsTransportRequestsToRepositoryQueries(t *testing.T) {
	now := time.Date(2026, 7, 18, 8, 0, 0, 0, time.UTC)
	repository := &fakeRepository{
		rawPage: RawDocumentStorePage{
			Items: []model.RawDocument{{ID: "raw-1", Title: "Raw", CollectedAt: now}},
			Total: 1, Page: 2, PageSize: 10,
		},
	}
	service := NewService(repository)

	rawPage, err := service.ListRawDocuments(context.Background(), RawDocumentListRequest{
		Title: "raw", SourceRef: "source:reuters:world", IngestStatus: model.IngestStatusCollected, Page: 2, PageSize: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if repository.rawFilter.Title != "raw" || repository.rawFilter.SourceRef != "source:reuters:world" || rawPage.Total != 1 || rawPage.Items[0].ID != "raw-1" {
		t.Fatalf("raw query was not mapped: filter=%#v page=%#v", repository.rawFilter, rawPage)
	}

}

type fakeRepository struct {
	rawPage   RawDocumentStorePage
	rawFilter RawDocumentListFilter
}

func (f *fakeRepository) ListRawDocuments(_ context.Context, filter RawDocumentListFilter) (RawDocumentStorePage, error) {
	f.rawFilter = filter
	return f.rawPage, nil
}
