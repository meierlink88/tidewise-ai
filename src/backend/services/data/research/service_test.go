package research

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

type fakeRepository struct {
	themePage    repositories.ResearchThemePage
	anchorPage   repositories.ResearchAnchorPage
	themeDetail  repositories.ResearchThemeDetail
	anchorDetail repositories.ResearchAnchorDetail
	err          error
	themeFilter  repositories.ResearchThemeListFilter
	anchorFilter repositories.ResearchAnchorListFilter
}

func (f *fakeRepository) ListResearchThemes(_ context.Context, filter repositories.ResearchThemeListFilter) (repositories.ResearchThemePage, error) {
	f.themeFilter = filter
	return f.themePage, f.err
}

func (f *fakeRepository) GetResearchTheme(context.Context, string, repositories.ResearchDetailFilter) (repositories.ResearchThemeDetail, error) {
	return f.themeDetail, f.err
}

func (f *fakeRepository) ListResearchAnchors(_ context.Context, filter repositories.ResearchAnchorListFilter) (repositories.ResearchAnchorPage, error) {
	f.anchorFilter = filter
	return f.anchorPage, f.err
}

func (f *fakeRepository) GetResearchAnchor(context.Context, string, repositories.ResearchDetailFilter) (repositories.ResearchAnchorDetail, error) {
	return f.anchorDetail, f.err
}

func TestServiceKeepsStableThemeCursorAndAuthoritativeFields(t *testing.T) {
	now := time.Date(2026, 7, 17, 1, 0, 0, 0, time.UTC)
	repository := &fakeRepository{themePage: repositories.ResearchThemePage{
		AsOf: now, WindowStart: now.Add(-24 * time.Hour), WindowEnd: now,
		ThemeCount: 1, EventCount: 2, HasMore: true,
		Items: []repositories.ResearchThemeSummary{{
			ID: "11111111-1111-4111-8111-111111111111", Name: "主题", OneLineConclusion: "结论",
			ImpactLevel: domain.ImpactLevelFocus, TransmissionPath: "供给传导", TradingDirection: "关注成本回落后的利润修复",
			TransmissionStage: domain.TransmissionStageInfrastructure, NextCheckpoint: "下次数据", IndexImpactSummary: "宽基震荡",
			PublishedAt:          now,
			ChainNodes:           []repositories.ResearchChainNode{{ID: "22222222-2222-4222-8222-222222222222", Name: "节点", RelationRole: "driver", Summary: "主题影响"}},
			Indices:              []repositories.ResearchIndex{{ID: "33333333-3333-4333-8333-333333333333", Name: "指数", ImpactDirection: "mixed", Summary: "指数影响"}},
			SupportingEventCount: 1, ContradictingEventCount: 1,
		}},
	}}
	service := NewService(repository, func() time.Time { return now })

	page, err := service.ListThemes(context.Background(), ResearchListRequest{WindowHours: 24, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if page.NextCursor == nil {
		t.Fatal("next cursor is nil")
	}
	if page.Items[0].ImpactLevel != domain.ImpactLevelFocus || page.Items[0].TransmissionStage != domain.TransmissionStageInfrastructure {
		t.Fatalf("authoritative theme enums drifted: %#v", page.Items[0])
	}
	if page.Items[0].TradingDirection != "关注成本回落后的利润修复" {
		t.Fatalf("trading_direction = %q", page.Items[0].TradingDirection)
	}
	if got := page.Items[0].AffectedChainNodes[0].ImpactSummary; got != "主题影响" {
		t.Fatalf("theme impact_summary = %q", got)
	}
	cursor, err := decodeResearchCursor(*page.NextCursor)
	if err != nil {
		t.Fatal(err)
	}
	if cursor.Kind != "themes" || cursor.Rank != 2 || cursor.ID != page.Items[0].ID || !cursor.AsOf.Equal(now) {
		t.Fatalf("cursor = %#v", cursor)
	}

	if _, err := service.ListThemes(context.Background(), ResearchListRequest{WindowHours: 24, Limit: 20, Cursor: *page.NextCursor}); err != nil {
		t.Fatal(err)
	}
	if repository.themeFilter.CursorRank != 2 || repository.themeFilter.CursorID != page.Items[0].ID || !repository.themeFilter.AsOf.Equal(now) {
		t.Fatalf("cursor repository filter = %#v", repository.themeFilter)
	}
}

func TestServiceKeepsAnchorRelationSummarySeparate(t *testing.T) {
	now := time.Date(2026, 7, 17, 1, 0, 0, 0, time.UTC)
	repository := &fakeRepository{anchorDetail: repositories.ResearchAnchorDetail{
		ResearchAnchorSummary: repositories.ResearchAnchorSummary{
			ID: "11111111-1111-4111-8111-111111111111", AnchorType: domain.AnchorTypeMarketStructure,
			Name: "锚点", OneLineConclusion: "结论", Importance: domain.ResearchImportanceContextual,
			TransmissionPath: "制度传导", TradingDirection: "等待确认后再评估", PublishedAt: now,
			ChainNodes: []repositories.ResearchChainNode{{ID: "22222222-2222-4222-8222-222222222222", Name: "节点", RelationRole: "constraint", Summary: "锚点关系"}},
			Indices:    []repositories.ResearchIndex{},
		},
		Events: []repositories.ResearchEvent{},
	}}
	service := NewService(repository, func() time.Time { return now })

	detail, err := service.GetAnchor(context.Background(), "11111111-1111-4111-8111-111111111111", ResearchDetailRequest{WindowHours: 24})
	if err != nil {
		t.Fatal(err)
	}
	if got := detail.Anchor.RelatedChainNodes[0].RelationSummary; got != "锚点关系" {
		t.Fatalf("anchor relation_summary = %q", got)
	}
	if detail.Anchor.RelatedChainNodes == nil || detail.Anchor.RelatedIndices == nil || detail.Events == nil {
		t.Fatal("empty collections must remain JSON arrays")
	}
}

func TestServiceKeepsStableAnchorCursorRank(t *testing.T) {
	now := time.Date(2026, 7, 17, 1, 0, 0, 0, time.UTC)
	repository := &fakeRepository{anchorPage: repositories.ResearchAnchorPage{
		AsOf: now, WindowStart: now.Add(-24 * time.Hour), WindowEnd: now, HasMore: true,
		Items: []repositories.ResearchAnchorSummary{{
			ID: "11111111-1111-4111-8111-111111111111", Importance: domain.ResearchImportanceSecondary,
			PublishedAt: now, ChainNodes: []repositories.ResearchChainNode{}, Indices: []repositories.ResearchIndex{},
		}},
	}}
	service := NewService(repository, func() time.Time { return now })
	page, err := service.ListAnchors(context.Background(), ResearchListRequest{WindowHours: 24, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if page.NextCursor == nil {
		t.Fatal("next cursor is nil")
	}
	cursor, err := decodeResearchCursor(*page.NextCursor)
	if err != nil {
		t.Fatal(err)
	}
	if cursor.Kind != "anchors" || cursor.Rank != 2 || cursor.ID != page.Items[0].ID {
		t.Fatalf("anchor cursor = %#v", cursor)
	}
}

func TestServiceValidatesRequestsAndClassifiesRepositoryErrors(t *testing.T) {
	service := NewService(&fakeRepository{}, time.Now)
	if _, err := service.ListThemes(context.Background(), ResearchListRequest{WindowHours: 169, Limit: 20}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("invalid window error = %v", err)
	}
	if _, err := service.GetTheme(context.Background(), "not-a-uuid", ResearchDetailRequest{WindowHours: 24}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("invalid UUID error = %v", err)
	}

	service = NewService(&fakeRepository{err: repositories.ErrResearchNotFound}, time.Now)
	if _, err := service.GetTheme(context.Background(), "11111111-1111-4111-8111-111111111111", ResearchDetailRequest{WindowHours: 24}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("not found error = %v", err)
	}
	service = NewService(&fakeRepository{err: errors.New("database unavailable")}, time.Now)
	if _, err := service.ListAnchors(context.Background(), ResearchListRequest{WindowHours: 24, Limit: 20}); !errors.Is(err, ErrRepository) {
		t.Fatalf("repository error = %v", err)
	}
}

var _ repositories.ResearchReadRepository = (*fakeRepository)(nil)
