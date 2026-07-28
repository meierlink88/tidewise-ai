package biz

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestResearchServiceMapsThemeV1WithOneAggregateCall(t *testing.T) {
	now := time.Date(2026, 7, 28, 8, 0, 0, 0, time.UTC)
	calls := 0
	repo := &Fake{ListResearchThemesFunc: func(_ context.Context, q ResearchListQuery) (ResearchThemePage, error) {
		calls++
		if q.WindowHours != 24 || q.Limit != 20 {
			t.Fatalf("query=%#v", q)
		}
		return ResearchThemePage{WindowStart: now.Add(-24 * time.Hour), WindowEnd: now, AsOf: now, ThemeCount: 1, Items: []ResearchTheme{{ID: "theme", Title: "高速光模块需求验证", ConclusionDirection: "positive", ImpactStrength: "medium", TransmissionStage: "validation", InvestmentGuidanceAction: "focus", InvestmentGuidanceSummary: "关注订单", TimeHorizonCategory: "short_term", AnalysisAsOf: now, WindowStart: now.Add(-24 * time.Hour), WindowEnd: now, PublishedAt: now, Impacts: []ResearchThemeImpact{{ChainNodeEntityID: "node", Name: "高速光模块", DisplayOrder: 1}}, ReasoningTreeCount: 2}}}, nil
	}}
	result, err := NewResearchService(repo).ListThemes(context.Background(), ResearchListRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || result.Items[0].Title != "高速光模块需求验证" || result.Items[0].Impacts[0].Name != "高速光模块" || result.Items[0].ReasoningTreeCount != 2 {
		t.Fatalf("result=%#v calls=%d", result, calls)
	}
}

func TestResearchServiceRejectsInvalidInputAndMapsRepoErrors(t *testing.T) {
	service := NewResearchService(&Fake{GetResearchThemeFunc: func(context.Context, string, ResearchDetailQuery) (ResearchThemeDetail, error) {
		return ResearchThemeDetail{}, ErrResearchNotFound
	}})
	if _, err := service.GetTheme(context.Background(), "bad", ResearchDetailRequest{}); !errors.Is(err, ErrInvalidResearchRequest) {
		t.Fatalf("invalid error=%v", err)
	}
	if _, err := service.GetTheme(context.Background(), "11111111-1111-4111-8111-111111111111", ResearchDetailRequest{}); !errors.Is(err, ErrResearchNotFound) {
		t.Fatalf("not found=%v", err)
	}
}
