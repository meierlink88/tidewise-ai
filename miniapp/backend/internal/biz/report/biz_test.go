package report

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type Fake struct {
	ListReportsFunc      func(context.Context, ListQuery) (Page, error)
	GetHomeFunc          func(context.Context, string) (Home, error)
	GetLayerFunc         func(context.Context, string, string) (LayerDetail, error)
	GetIndustryChainFunc func(context.Context, string, string) (IndustryChainDetail, error)
	ListEvidencesFunc    func(context.Context, string, EvidenceScope) (EvidenceCollection, error)
}

func (f *Fake) ListReports(ctx context.Context, query ListQuery) (Page, error) {
	if f.ListReportsFunc == nil {
		return Page{Items: []Summary{}}, nil
	}
	return f.ListReportsFunc(ctx, query)
}

func (f *Fake) GetHome(ctx context.Context, reportID string) (Home, error) {
	if f.GetHomeFunc == nil {
		return Home{}, nil
	}
	return f.GetHomeFunc(ctx, reportID)
}

func (f *Fake) GetLayer(ctx context.Context, reportID, layerKey string) (LayerDetail, error) {
	if f.GetLayerFunc == nil {
		return LayerDetail{}, nil
	}
	return f.GetLayerFunc(ctx, reportID, layerKey)
}

func (f *Fake) GetIndustryChain(ctx context.Context, reportID, chainKey string) (IndustryChainDetail, error) {
	if f.GetIndustryChainFunc == nil {
		return IndustryChainDetail{}, nil
	}
	return f.GetIndustryChainFunc(ctx, reportID, chainKey)
}

func (f *Fake) ListEvidences(ctx context.Context, reportID string, scope EvidenceScope) (EvidenceCollection, error) {
	if f.ListEvidencesFunc == nil {
		return EvidenceCollection{Items: []EvidenceItem{}}, nil
	}
	return f.ListEvidencesFunc(ctx, reportID, scope)
}

const (
	testReportID1 = "RPT11111111-1111-4111-8111-111111111111"
	testReportID2 = "RPT22222222-2222-4222-8222-222222222222"
	testReportID3 = "RPT33333333-3333-4333-8333-333333333333"
)

func TestUseCaseHomeReadsEveryShanghaiTodayPageAndKeepsStableOrder(t *testing.T) {
	now := time.Date(2026, 8, 31, 17, 30, 0, 0, time.UTC)
	published := time.Date(2026, 9, 1, 1, 0, 0, 0, time.UTC)
	summaries := []Summary{
		testSummary(testReportID1, published.Add(2*time.Minute)),
		testSummary(testReportID2, published.Add(time.Minute)),
		testSummary(testReportID3, published),
	}
	var mutex sync.Mutex
	queries := make([]ListQuery, 0, 2)
	repo := &Fake{
		ListReportsFunc: func(_ context.Context, query ListQuery) (Page, error) {
			mutex.Lock()
			defer mutex.Unlock()
			queries = append(queries, query)
			if query.Cursor == "" {
				next := "page-two"
				return Page{Items: summaries[:2], NextCursor: &next}, nil
			}
			if query.Cursor != "page-two" {
				t.Fatalf("cursor = %q", query.Cursor)
			}
			return Page{Items: summaries[2:]}, nil
		},
		GetHomeFunc: func(_ context.Context, reportID string) (Home, error) {
			// Deliberately finish in reverse order; output must remain Data order.
			switch reportID {
			case testReportID1:
				time.Sleep(15 * time.Millisecond)
			case testReportID2:
				time.Sleep(5 * time.Millisecond)
			}
			for _, summary := range summaries {
				if summary.ID == reportID {
					return Home{Report: summary, Cards: []Card{}}, nil
				}
			}
			return Home{}, ErrReportNotFound
		},
	}

	result, err := NewUseCaseWithClock(repo, func() time.Time { return now }).Home(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Selection != (HomeSelection{Mode: SelectionToday, Date: "2026-09-01", Timezone: "Asia/Shanghai"}) {
		t.Fatalf("selection = %#v", result.Selection)
	}
	if len(result.Reports) != 3 {
		t.Fatalf("reports = %#v", result.Reports)
	}
	for index, report := range result.Reports {
		if report.Report.ID != summaries[index].ID {
			t.Fatalf("reports[%d] = %q, want %q", index, report.Report.ID, summaries[index].ID)
		}
	}
	if len(queries) != 2 || queries[0].Limit != listPageSize || queries[0].PublishedFrom == nil || queries[0].PublishedTo == nil || queries[1].Cursor != "page-two" {
		t.Fatalf("queries = %#v", queries)
	}
	wantFrom := time.Date(2026, 8, 31, 16, 0, 0, 0, time.UTC)
	wantTo := time.Date(2026, 9, 1, 16, 0, 0, 0, time.UTC)
	if !queries[0].PublishedFrom.Equal(wantFrom) || !queries[0].PublishedTo.Equal(wantTo) {
		t.Fatalf("today bounds = %s..%s, want %s..%s", queries[0].PublishedFrom, queries[0].PublishedTo, wantFrom, wantTo)
	}
}

func TestUseCaseHomeFallsBackOnlyAfterEmptyTodayAndKeepsExplicitEmpty(t *testing.T) {
	t.Run("fallback one latest", func(t *testing.T) {
		queries := make([]ListQuery, 0, 2)
		summary := testSummary(testReportID1, time.Date(2026, 8, 31, 4, 0, 0, 0, time.UTC))
		repo := &Fake{
			ListReportsFunc: func(_ context.Context, query ListQuery) (Page, error) {
				queries = append(queries, query)
				if len(queries) == 1 {
					return Page{Items: []Summary{}}, nil
				}
				next := "older-history-exists"
				return Page{Items: []Summary{summary}, NextCursor: &next}, nil
			},
			GetHomeFunc: func(context.Context, string) (Home, error) {
				return Home{Report: summary, Cards: []Card{}}, nil
			},
		}

		result, err := NewUseCaseWithClock(repo, func() time.Time {
			return time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
		}).Home(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if result.Selection.Mode != SelectionLatestFallback || len(result.Reports) != 1 || len(queries) != 2 {
			t.Fatalf("result/queries = %#v/%#v", result, queries)
		}
		if queries[1].Limit != 1 || queries[1].PublishedFrom != nil || queries[1].PublishedTo != nil {
			t.Fatalf("fallback query = %#v", queries[1])
		}
	})

	t.Run("all empty", func(t *testing.T) {
		homeCalls := 0
		repo := &Fake{
			ListReportsFunc: func(context.Context, ListQuery) (Page, error) {
				return Page{Items: []Summary{}}, nil
			},
			GetHomeFunc: func(context.Context, string) (Home, error) {
				homeCalls++
				return Home{}, nil
			},
		}
		result, err := NewUseCaseWithClock(repo, func() time.Time {
			return time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
		}).Home(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if result.Selection.Mode != SelectionToday || result.Reports == nil || len(result.Reports) != 0 || homeCalls != 0 {
			t.Fatalf("result/home calls = %#v/%d", result, homeCalls)
		}
	})

	t.Run("empty fallback page cannot advertise more history", func(t *testing.T) {
		listCalls := 0
		repo := &Fake{ListReportsFunc: func(context.Context, ListQuery) (Page, error) {
			listCalls++
			if listCalls == 1 {
				return Page{Items: []Summary{}}, nil
			}
			next := "impossible-older-page"
			return Page{Items: []Summary{}, NextCursor: &next}, nil
		}}

		_, err := NewUseCaseWithClock(repo, func() time.Time {
			return time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
		}).Home(context.Background())
		if !errors.Is(err, ErrDataUnavailable) {
			t.Fatalf("fallback cursor error = %v", err)
		}
	})
}

func TestUseCaseHomeFailsClosedOnCursorLoopAndDataErrors(t *testing.T) {
	cursor := "same-cursor"
	repo := &Fake{ListReportsFunc: func(context.Context, ListQuery) (Page, error) {
		return Page{Items: []Summary{testSummary(testReportID1, time.Now().UTC())}, NextCursor: &cursor}, nil
	}}
	if _, err := NewUseCase(repo).Home(context.Background()); !errors.Is(err, ErrDataUnavailable) {
		t.Fatalf("cursor loop error = %v", err)
	}

	repo.ListReportsFunc = func(context.Context, ListQuery) (Page, error) {
		return Page{}, errors.New("secret downstream body")
	}
	if _, err := NewUseCase(repo).Home(context.Background()); !errors.Is(err, ErrDataUnavailable) || stringsContain(err.Error(), "secret") {
		t.Fatalf("downstream error = %v", err)
	}
}

func TestUseCaseValidatesDirectReadsAndEvidenceScopes(t *testing.T) {
	calls := 0
	repo := &Fake{GetLayerFunc: func(context.Context, string, string) (LayerDetail, error) {
		calls++
		return LayerDetail{}, errors.New("postgres password=must-not-leak")
	}}
	useCase := NewUseCase(repo)

	if _, err := useCase.Layer(context.Background(), "not-a-report-id", LayerGeopolitics); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("invalid report ID error = %v", err)
	}
	if _, err := useCase.Layer(context.Background(), testReportID1, "company"); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("invalid layer error = %v", err)
	}
	if calls != 0 {
		t.Fatalf("repository calls = %d, want 0", calls)
	}
	if _, err := useCase.Layer(context.Background(), testReportID1, LayerGeopolitics); !errors.Is(err, ErrDataUnavailable) {
		t.Fatalf("repository error = %v", err)
	}
	for _, scope := range []EvidenceScope{
		{Type: ScopeSectionSummary, Key: LayerGeopolitics},
		{Type: ScopeAnchor, Key: "anchor-1"},
		{Type: ScopeIndustryChainSummary, Key: "chain-1"},
		{Type: ScopeIndustryChainNode, Key: "node-1"},
	} {
		if !validEvidenceScope(scope) {
			t.Fatalf("valid scope rejected: %#v", scope)
		}
	}
	for _, scope := range []EvidenceScope{
		{Type: ScopeSectionSummary, Key: "geo-card"},
		{Type: ScopeAnchor, Key: "anchor/1"},
		{Type: "event", Key: "event-1"},
	} {
		if validEvidenceScope(scope) {
			t.Fatalf("invalid scope accepted: %#v", scope)
		}
	}
}

func testSummary(id string, publishedAt time.Time) Summary {
	return Summary{
		ID: id, PublisherReportID: "publisher-" + id[3:11], Title: "Report " + id[3:11],
		GeneratedAt: publishedAt.Add(-time.Minute).UTC(), PublishedAt: publishedAt.UTC(),
	}
}

func stringsContain(value, part string) bool {
	for index := 0; index+len(part) <= len(value); index++ {
		if value[index:index+len(part)] == part {
			return true
		}
	}
	return false
}
