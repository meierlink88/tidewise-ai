package sourcemetadata

import (
	"context"
	"errors"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
	"github.com/meierlink88/tidewise-ai/backend/services/data/repositories"
)

func TestServicePaginatesSourceMetadataByStableCursor(t *testing.T) {
	repository := &fakeRepository{sources: []domain.SourceCatalog{{ID: "source-a"}, {ID: "source-b"}, {ID: "source-c"}}}
	service := NewService(repository)

	page, err := service.List(context.Background(), ListRequest{Status: domain.SourceCatalogStatusActive, Limit: 1, Cursor: "source-a"})
	if err != nil {
		t.Fatal(err)
	}
	if repository.filter.Status != domain.SourceCatalogStatusActive || len(page.Items) != 1 || page.Items[0].ID != "source-b" || page.NextCursor == nil || *page.NextCursor != "source-b" {
		t.Fatalf("page = %#v filter = %#v", page, repository.filter)
	}

	_, err = service.List(context.Background(), ListRequest{Limit: 20, Cursor: "missing"})
	if !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("error = %v, want ErrInvalidCursor", err)
	}
}

type fakeRepository struct {
	sources []domain.SourceCatalog
	filter  repositories.SourceCatalogListFilter
}

func (f *fakeRepository) ListSourceCatalogs(_ context.Context, filter repositories.SourceCatalogListFilter) ([]domain.SourceCatalog, error) {
	f.filter = filter
	return f.sources, nil
}
