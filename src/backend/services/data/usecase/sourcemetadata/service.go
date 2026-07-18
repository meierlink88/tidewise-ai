// Package sourcemetadata owns the Agent-facing Source Catalog read use case.
package sourcemetadata

import (
	"context"
	"errors"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
	"github.com/meierlink88/tidewise-ai/backend/services/data/repositories"
)

var ErrInvalidCursor = errors.New("source metadata cursor is invalid")

type ListRequest struct {
	Status domain.SourceCatalogStatus
	Limit  int
	Cursor string
}

type Page struct {
	Items      []domain.SourceCatalog
	NextCursor *string
}

type Repository interface {
	ListSourceCatalogs(context.Context, repositories.SourceCatalogListFilter) ([]domain.SourceCatalog, error)
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) List(ctx context.Context, request ListRequest) (Page, error) {
	sources, err := s.repository.ListSourceCatalogs(ctx, repositories.SourceCatalogListFilter{Status: request.Status})
	if err != nil {
		return Page{}, err
	}
	limit := request.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	start := 0
	if request.Cursor != "" {
		found := false
		for index, source := range sources {
			if source.ID == request.Cursor {
				start = index + 1
				found = true
				break
			}
		}
		if !found {
			return Page{}, ErrInvalidCursor
		}
	}
	end := start + limit
	if end > len(sources) {
		end = len(sources)
	}
	items := append([]domain.SourceCatalog(nil), sources[start:end]...)
	var nextCursor *string
	if end < len(sources) && len(items) > 0 {
		value := items[len(items)-1].ID
		nextCursor = &value
	}
	return Page{Items: items, NextCursor: nextCursor}, nil
}
