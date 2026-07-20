package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/dataclient"
)

func TestResearchServiceRejectsInvalidInputBeforeCallingDataService(t *testing.T) {
	calls := 0
	client := &dataclient.Fake{
		ListResearchThemesFunc: func(context.Context, dataclient.ResearchListQuery) (dataclient.ResearchThemePage, error) {
			calls++
			return dataclient.ResearchThemePage{}, nil
		},
		GetResearchThemeFunc: func(context.Context, string, dataclient.ResearchDetailQuery) (dataclient.ResearchThemeDetail, error) {
			calls++
			return dataclient.ResearchThemeDetail{}, nil
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
	client := &dataclient.Fake{ListResearchThemesFunc: func(_ context.Context, query dataclient.ResearchListQuery) (dataclient.ResearchThemePage, error) {
		calls++
		if query != (dataclient.ResearchListQuery{WindowHours: 24, Limit: 20, Cursor: "opaque-current"}) {
			t.Fatalf("query = %#v", query)
		}
		return dataclient.ResearchThemePage{
			WindowStart: now.Add(-24 * time.Hour), WindowEnd: now, AsOf: now,
			ThemeCount: 1, EventCount: 2, NextCursor: &nextCursor,
			Items: []dataclient.ResearchTheme{{
				ID: "11111111-1111-4111-8111-111111111111", Name: "主题", OneLineConclusion: "结论",
				ImpactLevel: dataclient.ImpactLevelFocus, TransmissionPath: "政策到产业链",
				TradingDirection: "流动性改善后风险偏好可能回升", TransmissionStage: dataclient.TransmissionStageDiffusion,
				NextCheckpoint: "下周数据", PublishedAt: now,
				AffectedChainNodes:   []dataclient.ResearchThemeChainNode{{ID: "node-1", Name: "算力", RelationRole: "driver", ImpactSummary: "资本开支上升"}},
				RelatedIndices:       []dataclient.ResearchIndex{{ID: "index-1", Name: "指数", ImpactDirection: dataclient.ImpactDirectionNeutral, ImpactSummary: "等待验证"}},
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
		{name: "bad request", err: &dataclient.Error{Kind: dataclient.ErrorKindClient, StatusCode: 400}, want: ErrInvalidResearchRequest},
		{name: "not found", err: &dataclient.Error{Kind: dataclient.ErrorKindClient, StatusCode: 404}, want: ErrResearchNotFound},
		{name: "server", err: &dataclient.Error{Kind: dataclient.ErrorKindServer, StatusCode: 500, RequestID: "safe-upstream-id"}, want: ErrResearchDataService},
		{name: "timeout", err: &dataclient.Error{Kind: dataclient.ErrorKindTimeout}, want: ErrResearchDataService},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := &dataclient.Fake{GetResearchThemeFunc: func(context.Context, string, dataclient.ResearchDetailQuery) (dataclient.ResearchThemeDetail, error) {
				calls++
				return dataclient.ResearchThemeDetail{}, test.err
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
