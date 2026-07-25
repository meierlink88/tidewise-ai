package biz

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestResearchServiceRejectsInvalidInputBeforeCallingDataService(t *testing.T) {
	calls := 0
	client := &Fake{
		ListResearchThemesFunc: func(context.Context, ResearchListQuery) (ResearchThemePage, error) {
			calls++
			return ResearchThemePage{}, nil
		},
		GetResearchThemeFunc: func(context.Context, string, ResearchDetailQuery) (ResearchThemeDetail, error) {
			calls++
			return ResearchThemeDetail{}, nil
		},
	}
	service := NewResearchService(client)

	if _, err := service.ListThemes(context.Background(), ResearchListRequest{WindowHours: 169, Limit: 20}); !errors.Is(err, ErrInvalidResearchRequest) {
		t.Fatalf("ListThemes() error = %v, want invalid request", err)
	}
	if _, err := service.GetTheme(context.Background(), "theme-1", ResearchDetailRequest{WindowHours: 24}); !errors.Is(err, ErrInvalidResearchRequest) {
		t.Fatalf("GetTheme() error = %v, want invalid request", err)
	}
	if calls != 0 {
		t.Fatalf("Data Service calls = %d, want 0", calls)
	}
}

func TestResearchServiceListsThemesWithOneAggregateCallAndPreservesCursorAndDTO(t *testing.T) {
	now := time.Date(2026, 7, 17, 1, 0, 0, 0, time.UTC)
	nextCursor := "opaque-next"
	calls := 0
	client := &Fake{ListResearchThemesFunc: func(_ context.Context, query ResearchListQuery) (ResearchThemePage, error) {
		calls++
		if query != (ResearchListQuery{WindowHours: 24, Limit: 20, Cursor: "opaque-current"}) {
			t.Fatalf("query = %#v", query)
		}
		return ResearchThemePage{
			WindowStart: now.Add(-24 * time.Hour), WindowEnd: now, AsOf: now,
			ThemeCount: 1, EventCount: 2, NextCursor: &nextCursor,
			Items: []ResearchTheme{{
				ID: "11111111-1111-4111-8111-111111111111", Name: "主题", OneLineConclusion: "结论",
				ImpactLevel: ImpactLevelFocus, TransmissionPath: "政策到产业链",
				TradingDirection: "流动性改善后风险偏好可能回升", TransmissionStage: TransmissionStageDiffusion,
				NextCheckpoint: "下周数据", PublishedAt: now,
				AffectedChainNodes:   []ResearchThemeChainNode{{ID: "node-1", Name: "算力", RelationRole: "driver", ImpactSummary: "资本开支上升"}},
				RelatedIndices:       []ResearchIndex{{ID: "index-1", Name: "指数", ImpactDirection: ImpactDirectionNeutral, ImpactSummary: "等待验证"}},
				SupportingEventCount: 2, ContradictingEventCount: 1,
			}},
		}, nil
	}}

	result, err := NewResearchService(client).ListThemes(context.Background(), ResearchListRequest{WindowHours: 24, Limit: 20, Cursor: "opaque-current"})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("aggregate calls = %d, want 1", calls)
	}
	if result.NextCursor == nil || *result.NextCursor != nextCursor || result.Items[0].ImpactLevel != "focus" || result.Items[0].TradingDirection != "流动性改善后风险偏好可能回升" {
		t.Fatalf("result = %#v", result)
	}
	if result.Items[0].AffectedChainNodes[0].Summary != "资本开支上升" || result.Items[0].RelatedIndices[0].ImpactDirection != "neutral" {
		t.Fatalf("relations = %#v/%#v", result.Items[0].AffectedChainNodes, result.Items[0].RelatedIndices)
	}
}

func TestResearchServiceMapsDataErrorsToStablePublicErrors(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		want error
	}{
		{name: "invalid request", err: ErrInvalidResearchRequest, want: ErrInvalidResearchRequest},
		{name: "not found", err: ErrResearchNotFound, want: ErrResearchNotFound},
		{name: "data service", err: ErrResearchDataService, want: ErrResearchDataService},
		{name: "unknown adapter failure", err: errors.New("must not cross the Biz port"), want: ErrResearchDataService},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := &Fake{GetResearchThemeFunc: func(context.Context, string, ResearchDetailQuery) (ResearchThemeDetail, error) {
				calls++
				return ResearchThemeDetail{}, test.err
			}}
			_, err := NewResearchService(client).GetTheme(context.Background(), "11111111-1111-4111-8111-111111111111", ResearchDetailRequest{WindowHours: 24})
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
			if calls != 1 {
				t.Fatalf("calls = %d, want 1", calls)
			}
			if errors.Is(err, ErrResearchDataService) && err.Error() != ErrResearchDataService.Error() {
				t.Fatalf("internal error leaked: %q", err)
			}
		})
	}
}
