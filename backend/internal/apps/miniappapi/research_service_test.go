package miniappapi

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

type fakeResearchReadRepository struct {
	themePage    repositories.ResearchThemePage
	anchorPage   repositories.ResearchAnchorPage
	themeDetail  *repositories.ResearchThemeDetail
	anchorDetail *repositories.ResearchAnchorDetail
	err          error
	lastTheme    repositories.ResearchThemeListFilter
}

func (f *fakeResearchReadRepository) ListResearchThemes(_ context.Context, filter repositories.ResearchThemeListFilter) (repositories.ResearchThemePage, error) {
	f.lastTheme = filter
	if f.err != nil {
		return repositories.ResearchThemePage{}, f.err
	}
	return f.themePage, nil
}
func (f *fakeResearchReadRepository) GetResearchTheme(_ context.Context, _ string, _ repositories.ResearchDetailFilter) (repositories.ResearchThemeDetail, error) {
	if f.err != nil {
		return repositories.ResearchThemeDetail{}, f.err
	}
	if f.themeDetail == nil {
		return repositories.ResearchThemeDetail{}, repositories.ErrResearchNotFound
	}
	return *f.themeDetail, nil
}
func (f *fakeResearchReadRepository) ListResearchAnchors(_ context.Context, _ repositories.ResearchAnchorListFilter) (repositories.ResearchAnchorPage, error) {
	if f.err != nil {
		return repositories.ResearchAnchorPage{}, f.err
	}
	return f.anchorPage, nil
}
func (f *fakeResearchReadRepository) GetResearchAnchor(_ context.Context, _ string, _ repositories.ResearchDetailFilter) (repositories.ResearchAnchorDetail, error) {
	if f.err != nil {
		return repositories.ResearchAnchorDetail{}, f.err
	}
	if f.anchorDetail == nil {
		return repositories.ResearchAnchorDetail{}, repositories.ErrResearchNotFound
	}
	return *f.anchorDetail, nil
}

func TestResearchServiceRejectsInvalidListParameters(t *testing.T) {
	service := NewResearchService(&fakeResearchReadRepository{}, func() time.Time { return time.Date(2026, 7, 17, 1, 0, 0, 0, time.UTC) })
	if _, err := service.ListThemes(context.Background(), ResearchListRequest{WindowHours: 169, Limit: 20}); !errors.Is(err, ErrInvalidResearchRequest) {
		t.Fatalf("ListThemes() error = %v, want invalid request", err)
	}
}

func TestResearchServiceRejectsInvalidUUID(t *testing.T) {
	service := NewResearchService(&fakeResearchReadRepository{}, time.Now)
	if _, err := service.GetTheme(context.Background(), "theme-1", ResearchDetailRequest{WindowHours: 24}); !errors.Is(err, ErrInvalidResearchRequest) {
		t.Fatalf("GetTheme() error = %v, want invalid request", err)
	}
}

func TestResearchServiceEncodesStableCursorAndMapsEmptyCollections(t *testing.T) {
	now := time.Date(2026, 7, 17, 1, 0, 0, 0, time.UTC)
	repo := &fakeResearchReadRepository{themePage: repositories.ResearchThemePage{
		AsOf: now, WindowStart: now.Add(-24 * time.Hour), WindowEnd: now,
		ThemeCount: 1, EventCount: 0, Items: []repositories.ResearchThemeSummary{{ID: "theme-1", Name: "主题", PublishedAt: now, ChainNodes: []repositories.ResearchChainNode{}, Indices: []repositories.ResearchIndex{}}}, HasMore: true,
	}}
	service := NewResearchService(repo, func() time.Time { return now })
	result, err := service.ListThemes(context.Background(), ResearchListRequest{WindowHours: 24, Limit: 20})
	if err != nil {
		t.Fatalf("ListThemes() error = %v", err)
	}
	if result.NextCursor == nil || *result.NextCursor == "" {
		t.Fatal("NextCursor is empty, want opaque cursor")
	}
	if result.Items[0].AffectedChainNodes == nil || result.Items[0].RelatedIndices == nil {
		t.Fatal("empty collections must be non-nil")
	}
	if repo.lastTheme.AsOf != now || repo.lastTheme.Limit != 20 {
		t.Fatalf("repository filter = %+v", repo.lastTheme)
	}
}

func TestResearchServiceMapsNotFoundAndRepositoryFailure(t *testing.T) {
	for name, errValue := range map[string]error{
		"not found":          repositories.ErrResearchNotFound,
		"repository failure": errors.New("database unavailable"),
	} {
		t.Run(name, func(t *testing.T) {
			service := NewResearchService(&fakeResearchReadRepository{err: errValue}, time.Now)
			_, err := service.GetTheme(context.Background(), "11111111-1111-4111-8111-111111111111", ResearchDetailRequest{WindowHours: 24})
			if err == nil {
				t.Fatal("GetTheme() error = nil")
			}
		})
	}
}

func TestResearchServiceLeavesCursorNilWhenPageHasNoMoreRows(t *testing.T) {
	now := time.Date(2026, 7, 17, 1, 0, 0, 0, time.UTC)
	repo := &fakeResearchReadRepository{themePage: repositories.ResearchThemePage{AsOf: now, WindowStart: now.Add(-24 * time.Hour), WindowEnd: now}}
	service := NewResearchService(repo, func() time.Time { return now })
	result, err := service.ListThemes(context.Background(), ResearchListRequest{WindowHours: 24, Limit: 20})
	if err != nil {
		t.Fatalf("ListThemes() error = %v", err)
	}
	if result.NextCursor != nil {
		t.Fatal("NextCursor is non-nil without more rows")
	}
}
