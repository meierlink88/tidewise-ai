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

func TestResearchServiceUsesShanghaiNaturalDayForTodayThemes(t *testing.T) {
	now := time.Date(2026, 8, 4, 15, 30, 0, 0, time.UTC)
	repo := &Fake{ListResearchThemesFunc: func(_ context.Context, query ResearchListQuery) (ResearchThemePage, error) {
		wantFrom := time.Date(2026, 8, 3, 16, 0, 0, 0, time.UTC)
		wantTo := time.Date(2026, 8, 4, 16, 0, 0, 0, time.UTC)
		if query.PublishedFrom == nil || query.PublishedTo == nil || !query.PublishedFrom.Equal(wantFrom) || !query.PublishedTo.Equal(wantTo) {
			t.Fatalf("publication range = [%v, %v), want [%s, %s)", query.PublishedFrom, query.PublishedTo, wantFrom, wantTo)
		}
		if query.WindowHours != 0 || query.Limit != 20 {
			t.Fatalf("query = %#v", query)
		}
		return ResearchThemePage{WindowStart: wantFrom, WindowEnd: wantTo, AsOf: now}, nil
	}}

	_, err := NewResearchServiceWithClock(repo, func() time.Time { return now }).ListThemes(
		context.Background(), ResearchListRequest{Period: ResearchPeriodToday},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestResearchServiceUsesPreviousThirtyCalendarDaysAndFiveItemPagesForHistory(t *testing.T) {
	now := time.Date(2026, 8, 4, 3, 0, 0, 0, time.UTC)
	repo := &Fake{ListResearchThemesFunc: func(_ context.Context, query ResearchListQuery) (ResearchThemePage, error) {
		wantFrom := time.Date(2026, 7, 4, 16, 0, 0, 0, time.UTC)
		wantTo := time.Date(2026, 8, 3, 16, 0, 0, 0, time.UTC)
		if query.PublishedFrom == nil || query.PublishedTo == nil || !query.PublishedFrom.Equal(wantFrom) || !query.PublishedTo.Equal(wantTo) || query.Limit != 5 {
			t.Fatalf("query = %#v", query)
		}
		return ResearchThemePage{WindowStart: wantFrom, WindowEnd: wantTo, AsOf: now}, nil
	}}

	_, err := NewResearchServiceWithClock(repo, func() time.Time { return now }).ListThemes(
		context.Background(), ResearchListRequest{Period: ResearchPeriodHistory},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestResearchServiceCursorFreezesHistoryPublicationRange(t *testing.T) {
	now := time.Date(2026, 8, 4, 15, 30, 0, 0, time.UTC)
	dataCursor := "data-page-two"
	var queries []ResearchListQuery
	repo := &Fake{ListResearchThemesFunc: func(_ context.Context, query ResearchListQuery) (ResearchThemePage, error) {
		queries = append(queries, query)
		if len(queries) == 1 {
			return ResearchThemePage{NextCursor: &dataCursor}, nil
		}
		return ResearchThemePage{}, nil
	}}
	service := NewResearchServiceWithClock(repo, func() time.Time { return now })

	first, err := service.ListThemes(context.Background(), ResearchListRequest{Period: ResearchPeriodHistory})
	if err != nil || first.NextCursor == nil {
		t.Fatalf("first page cursor/error = %v/%v", first.NextCursor, err)
	}
	now = now.Add(2 * time.Hour)
	_, err = service.ListThemes(context.Background(), ResearchListRequest{
		Period: ResearchPeriodHistory, Cursor: *first.NextCursor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(queries) != 2 || queries[1].Cursor != dataCursor || !queries[1].PublishedFrom.Equal(*queries[0].PublishedFrom) || !queries[1].PublishedTo.Equal(*queries[0].PublishedTo) {
		t.Fatalf("queries = %#v", queries)
	}
}

func TestResearchServiceRejectsCursorWithClientChosenPublicationRange(t *testing.T) {
	now := time.Date(2026, 8, 4, 3, 0, 0, 0, time.UTC)
	forged, err := encodePeriodCursor(periodCursor{
		Version: 1, Period: ResearchPeriodHistory,
		PublishedFrom: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		PublishedTo:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		DataCursor:    "forged-data-cursor",
	})
	if err != nil {
		t.Fatal(err)
	}
	service := NewResearchServiceWithClock(&Fake{}, func() time.Time { return now })

	_, err = service.ListThemes(context.Background(), ResearchListRequest{
		Period: ResearchPeriodHistory, Cursor: forged,
	})
	if !errors.Is(err, ErrInvalidResearchRequest) {
		t.Fatalf("error = %v, want %v", err, ErrInvalidResearchRequest)
	}
}

func TestResearchServiceRejectsInvalidInputAndMapsRepoErrors(t *testing.T) {
	service := NewResearchService(&Fake{GetResearchThemeFunc: func(context.Context, string) (ResearchThemeDetail, error) {
		return ResearchThemeDetail{}, ErrResearchNotFound
	}})
	if _, err := service.GetTheme(context.Background(), "bad", ResearchDetailRequest{}); !errors.Is(err, ErrInvalidResearchRequest) {
		t.Fatalf("invalid error=%v", err)
	}
	if _, err := service.GetTheme(context.Background(), "11111111-1111-4111-8111-111111111111", ResearchDetailRequest{}); !errors.Is(err, ErrResearchNotFound) {
		t.Fatalf("not found=%v", err)
	}
}
