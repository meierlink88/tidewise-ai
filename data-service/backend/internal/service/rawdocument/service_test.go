package rawdocument

import (
	"context"
	"testing"
	"time"

	api "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/rawdocument"
	biz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/rawdocument"
)

func TestServiceListsDocumentsThroughUseCase(t *testing.T) {
	now := time.Date(2026, 8, 12, 3, 4, 5, 0, time.UTC)
	store := &storeStub{page: biz.StorePage{Items: []biz.Document{{ID: "raw-1", ContractVersion: 2, ArtifactID: "artifact", SourceRef: "source", SourceType: "news", SourceName: "wire", Title: "title", ContentHash: "hash", CollectedAt: now}}, Total: 1, Page: 2, PageSize: 10}}
	useCase, err := biz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(useCase)
	if err != nil {
		t.Fatal(err)
	}
	response, err := service.List(context.Background(), &api.ListRequest{Title: " title ", SourceRef: " source ", IngestStatus: "collected", Page: "2", PageSize: "10"})
	if err != nil {
		t.Fatal(err)
	}
	if store.filter.Title != "title" || store.filter.SourceRef != "source" || store.filter.Page != 2 || store.filter.PageSize != 10 {
		t.Fatalf("filter = %#v", store.filter)
	}
	if len(response.Result.Items) != 1 || response.Result.Items[0].CollectedAt != "2026-08-12T03:04:05Z" {
		t.Fatalf("response = %#v", response.Result)
	}
}

func TestServiceRejectsInvalidStatusAndDependency(t *testing.T) {
	if _, err := NewService(nil); err == nil {
		t.Fatal("NewService(nil) error = nil")
	}
	useCase, _ := biz.NewUseCase(&storeStub{})
	service, _ := NewService(useCase)
	if _, err := service.List(context.Background(), &api.ListRequest{IngestStatus: "invalid"}); err == nil {
		t.Fatal("List() error = nil")
	}
}

type storeStub struct {
	filter biz.ListFilter
	page   biz.StorePage
	err    error
}

func (s *storeStub) List(_ context.Context, filter biz.ListFilter) (biz.StorePage, error) {
	s.filter = filter
	return s.page, s.err
}
