package report

import (
	"context"
	"testing"
	"time"
)

const testReportID = "RPT11111111-1111-4111-8111-111111111111"
const testScopeToken = "RPE11111111-1111-4111-8111-111111111111"

func TestHomeSelectsOnlyLatestReportPublishedToday(t *testing.T) {
	latest := validSummary()
	latest.ID = "RPT22222222-2222-4222-8222-222222222222"
	latest.PublishedAt = time.Date(2026, 9, 2, 2, 0, 0, 0, time.UTC)
	older := validSummary()
	repository := &fakeRepository{
		listPage: Page{Items: []Summary{latest, older}},
		homes: map[string]HomeSnapshot{
			latest.ID: {Report: latest},
			older.ID:  {Report: older},
		},
		chainPage: IndustryChainPage{Items: []IndustryChainSummary{validChainSummary()}},
	}
	useCase := NewUseCaseWithClock(repository, func() time.Time {
		return time.Date(2026, 9, 2, 18, 0, 0, 0, time.FixedZone("CST", 8*3600))
	})

	home, err := useCase.Home(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(home.Reports) != 1 || home.Reports[0].Report.ID != latest.ID {
		t.Fatalf("reports=%#v", home.Reports)
	}
	if len(repository.listQueries) != 1 || repository.listQueries[0].Limit != 1 ||
		repository.listQueries[0].PublishedFrom == nil || repository.listQueries[0].PublishedTo == nil {
		t.Fatalf("queries=%#v", repository.listQueries)
	}
}

func TestHomeFallsBackToLatestHistoricalReportWhenTodayIsEmpty(t *testing.T) {
	historical := validSummary()
	repository := &fakeRepository{
		listPages: []Page{{Items: []Summary{}}, {Items: []Summary{historical}}},
		homes:     map[string]HomeSnapshot{historical.ID: {Report: historical}},
		chainPage: IndustryChainPage{Items: []IndustryChainSummary{validChainSummary()}},
	}
	useCase := NewUseCaseWithClock(repository, func() time.Time {
		return time.Date(2026, 9, 3, 18, 0, 0, 0, time.FixedZone("CST", 8*3600))
	})

	home, err := useCase.Home(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if home.Selection.Mode != SelectionFallback || len(home.Reports) != 1 ||
		home.Reports[0].Report.ID != historical.ID {
		t.Fatalf("home=%#v", home)
	}
	if len(repository.listQueries) != 2 || repository.listQueries[0].Limit != 1 ||
		repository.listQueries[1].Limit != 1 || repository.listQueries[1].PublishedFrom != nil ||
		repository.listQueries[1].PublishedTo != nil {
		t.Fatalf("queries=%#v", repository.listQueries)
	}
}

func TestEvidenceRequiresOpaqueScopeToken(t *testing.T) {
	repository := &fakeRepository{evidence: EvidenceCollection{ReportID: testReportID, ScopeToken: testScopeToken, Items: []EvidenceItem{}}}
	useCase := NewUseCase(repository)
	if _, err := useCase.Evidences(context.Background(), testReportID, "anchor"); err != ErrInvalidRequest {
		t.Fatalf("invalid token error=%v", err)
	}
	value, err := useCase.Evidences(context.Background(), testReportID, testScopeToken)
	if err != nil || value.ScopeToken != testScopeToken {
		t.Fatalf("value=%#v err=%v", value, err)
	}
}

func validSummary() Summary {
	return Summary{SchemaVersion: "report-publication/v5", ID: testReportID, PublisherReportID: "publisher", GeneratedAt: time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC), PublishedAt: time.Date(2026, 9, 2, 1, 0, 0, 0, time.UTC), IndustryChainCount: 54}
}
func sampleLayer() Layer {
	return Layer{Key: LayerGeopolitics, Title: "地缘政治", Conclusion: "地缘风险升温", Result: CodedLabel{Code: "warming", Label: "升温"}, Confidence: Confidence{Code: "high", Label: "高"}, TimeWindow: TimeWindow{Code: "short", Label: "短期"}, Anchors: []Anchor{{LocalKey: "anchor-01", Name: "锚点", CurrentState: "UP", Result: CodedLabel{Code: "warming", Label: "升温"}, ConclusionBasis: CodedLabel{Code: "direct_evidence", Label: "直接证据"}, ValidationStatus: CodedLabel{Code: "confirmed", Label: "已确认"}, Reasoning: "逻辑", TimeWindow: TimeWindow{Code: "short", Label: "短期"}, Confidence: Confidence{Code: "high", Label: "高"}, EvidenceScopeToken: stringPointer(testScopeToken)}}, ReasoningSteps: []ReasoningStep{}, Transmissions: []Transmission{}, Uncertainty: LayerUncertainty{}, EvidenceScopeToken: stringPointer(testScopeToken)}
}
func (l Layer) summary() LayerSummary {
	return LayerSummary{Conclusion: l.Conclusion, Result: l.Result, Confidence: l.Confidence, TimeWindow: l.TimeWindow, Transmissions: l.Transmissions, Uncertainty: l.Uncertainty, EvidenceScopeToken: l.EvidenceScopeToken}
}
func validChainSummary() IndustryChainSummary {
	return IndustryChainSummary{LocalKey: "chain-01", Name: "产业链", Conclusion: "结论", Result: CodedLabel{Code: "warming", Label: "升温"}, Confidence: Confidence{Code: "medium", Label: "中"}, TimeWindow: TimeWindow{Code: "medium", Label: "中期"}, ImpactItems: []IndustryChainImpactSummary{}, EvidenceScopeToken: stringPointer(testScopeToken)}
}
func stringPointer(value string) *string { return &value }

type fakeRepository struct {
	analysisQueries []AnalysisQuery
	analysisPage    AnalysisPage
	analysisErr     error
	analysisDetail  NormalizedDetailProjection
	listErr         error
	homeCalls       int
	listPage        Page
	listPages       []Page
	listQueries     []ListQuery
	home            HomeSnapshot
	homes           map[string]HomeSnapshot
	layer           LayerDetail
	chainPage       IndustryChainPage
	chainPages      map[string]IndustryChainPage
	chain           IndustryChainDetail
	evidence        EvidenceCollection
	chainQueries    []ChainListQuery
}

func (f *fakeRepository) ListReports(_ context.Context, query ListQuery) (Page, error) {
	f.listQueries = append(f.listQueries, query)
	page := f.listPage
	if len(f.listPages) >= len(f.listQueries) {
		page = f.listPages[len(f.listQueries)-1]
	}
	if query.Limit > 0 && len(page.Items) > query.Limit {
		page.Items = page.Items[:query.Limit]
	}
	return page, f.listErr
}
func (f *fakeRepository) GetHome(_ context.Context, reportID string) (HomeSnapshot, error) {
	f.homeCalls++
	if f.homes != nil {
		return f.homes[reportID], nil
	}
	return f.home, nil
}
func (f *fakeRepository) ListIndustryChains(_ context.Context, query ChainListQuery) (IndustryChainPage, error) {
	f.chainQueries = append(f.chainQueries, query)
	if f.chainPages != nil {
		return f.chainPages[query.Cursor], nil
	}
	return f.chainPage, nil
}
func (f *fakeRepository) GetLayer(context.Context, string, string) (LayerDetail, error) {
	return f.layer, nil
}
func (f *fakeRepository) GetIndustryChain(context.Context, string, string) (IndustryChainDetail, error) {
	return f.chain, nil
}
func (f *fakeRepository) ListEvidences(context.Context, string, string) (EvidenceCollection, error) {
	return f.evidence, nil
}

var _ Repository = (*fakeRepository)(nil)

func (f *fakeRepository) ListAnalyses(_ context.Context, q AnalysisQuery) (AnalysisPage, error) {
	f.analysisQueries = append(f.analysisQueries, q)
	return f.analysisPage, f.analysisErr
}

func (f *fakeRepository) GetAnalysis(context.Context, AnalysisQuery) (NormalizedDetailProjection, error) {
	return f.analysisDetail, f.analysisErr
}

func (*fakeRepository) GetAnalysisChain(context.Context, AnalysisQuery) (NormalizedChain, error) {
	return NormalizedChain{}, ErrDataUnavailable
}

func TestNormalizedHomeUsesSelectedReportAndIndependentGroupPages(t *testing.T) {
	s := validSummary()
	s.SchemaVersion = "report-publication/v4"
	s.IndustryChainCount = 0
	cursor := "opaque-first-page"
	r := &fakeRepository{listPage: Page{Items: []Summary{s}}, analysisPage: AnalysisPage{Items: []NormalizedSummaryProjection{}, NextCursor: &cursor}}
	home, err := NewUseCase(r).Home(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(home.Reports) != 1 || len(home.Reports[0].AnalysisGroups) != 3 || len(home.Reports[0].Cards) != 0 || r.homeCalls != 0 || len(r.chainQueries) != 0 {
		t.Fatalf("home=%+v", home)
	}
	for i, k := range []string{"geopolitical_stories", "macroeconomic_stories", "concept_analyses"} {
		q := r.analysisQueries[i]
		if q.Kind != k || q.ReportID != s.ID || q.Limit != 20 || q.Cursor != "" || *home.Reports[0].AnalysisGroups[i].NextCursor != cursor {
			t.Fatalf("query=%+v", q)
		}
	}
	_, err = NewUseCase(r).Analyses(context.Background(), AnalysisQuery{ReportID: s.ID, Kind: "concept_analyses", Limit: 13, Cursor: cursor})
	if err != nil {
		t.Fatal(err)
	}
	if q := r.analysisQueries[3]; q.Cursor != cursor || q.Limit != 13 {
		t.Fatalf("query=%+v", q)
	}
	r.analysisErr = ErrDataUnavailable
	if _, err = NewUseCase(r).Home(context.Background()); err != ErrDataUnavailable {
		t.Fatalf("expected latest report error: %v", err)
	}
	if r.homeCalls != 0 {
		t.Fatal("silently fell back to legacy home")
	}
}

func TestV5HomeUsesNormalizedGroups(t *testing.T) {
	s := validSummary()
	s.SchemaVersion = "report-publication/v5"
	r := &fakeRepository{listPage: Page{Items: []Summary{s}}, analysisPage: AnalysisPage{Items: []NormalizedSummaryProjection{}}}
	home, err := NewUseCase(r).Home(context.Background())
	if err != nil || len(home.Reports) != 1 || len(home.Reports[0].AnalysisGroups) != 4 || r.homeCalls != 0 {
		t.Fatalf("v5 home failed: %+v %v", home, err)
	}
	if q := r.analysisQueries[3]; q.Kind != "industry_chain_analyses" || q.ReportID != s.ID || q.Limit != 20 {
		t.Fatalf("industry query=%+v", q)
	}
}

func TestHomeRejectsRetiredSnapshotWithoutReadingLegacyEndpoints(t *testing.T) {
	summary := validSummary()
	summary.SchemaVersion = ""
	repository := &fakeRepository{listPage: Page{Items: []Summary{summary}}}
	_, err := NewUseCase(repository).Home(context.Background())
	if err != ErrDataUnavailable || len(repository.analysisQueries) != 0 {
		t.Fatalf("err=%v queries=%v", err, repository.analysisQueries)
	}
}

func TestAnalysisAssociatesPublicationFromHistoricalReportPage(t *testing.T) {
	target := validSummary()
	newer := target
	newer.ID = "RPT22222222-2222-4222-8222-222222222222"
	newer.PublishedAt = target.PublishedAt.Add(time.Hour)
	repo := &fakeRepository{listPages: []Page{
		{Items: []Summary{newer}, NextCursor: stringPointer("next-page")},
		{Items: []Summary{target}},
	}}
	got, err := NewUseCase(repo).Analysis(context.Background(), AnalysisQuery{ReportID: target.ID, Kind: "geopolitical_stories", Key: "story"})
	if err != nil || got.PublishedAt == nil || !got.PublishedAt.Equal(target.PublishedAt) {
		t.Fatalf("publication=%v err=%v", got.PublishedAt, err)
	}
	if len(repo.listQueries) != 2 || repo.listQueries[1].Cursor != "next-page" || repo.listQueries[0].Limit != 100 {
		t.Fatalf("queries=%#v", repo.listQueries)
	}
}

func TestAnalysisPublicationLookupFailsExplicitly(t *testing.T) {
	for _, tc := range []struct {
		name string
		repo *fakeRepository
	}{
		{"absent", &fakeRepository{}},
		{"upstream failure", &fakeRepository{listErr: ErrDataUnavailable}},
		{"repeated cursor", &fakeRepository{listPage: Page{Items: []Summary{validSummary()}, NextCursor: stringPointer("loop")}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewUseCase(tc.repo).Analysis(context.Background(), AnalysisQuery{ReportID: "RPT22222222-2222-4222-8222-222222222222", Kind: "geopolitical_stories", Key: "story"})
			if err == nil {
				t.Fatal("expected explicit metadata lookup failure")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	repo := &fakeRepository{}
	_, err := NewUseCase(repo).Analysis(ctx, AnalysisQuery{ReportID: testReportID, Kind: "geopolitical_stories", Key: "story"})
	if err != ErrDataUnavailable || len(repo.listQueries) != 0 {
		t.Fatalf("err=%v queries=%v", err, repo.listQueries)
	}
}
