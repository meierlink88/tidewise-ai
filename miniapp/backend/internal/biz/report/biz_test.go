package report

import (
	"context"
	"testing"
	"time"
)

const testReportID = "RPT11111111-1111-4111-8111-111111111111"
const testScopeToken = "RPE11111111-1111-4111-8111-111111111111"

func TestHomeLoadsUpperDetailsAndOnlyFirstIndustryChainPage(t *testing.T) {
	summary := validSummary()
	next := "next-page"
	repository := &fakeRepository{
		listPage:  Page{Items: []Summary{summary}},
		home:      HomeSnapshot{Report: summary, Geopolitics: &LayerSnapshot{Key: LayerGeopolitics, Title: "地缘政治", Summary: sampleLayer().summary()}},
		layer:     LayerDetail{Report: summary, Layer: sampleLayer()},
		chainPage: IndustryChainPage{Items: []IndustryChainSummary{validChainSummary()}, NextCursor: &next},
	}
	useCase := NewUseCaseWithClock(repository, func() time.Time { return time.Date(2026, 9, 2, 9, 0, 0, 0, time.FixedZone("CST", 8*3600)) })
	home, err := useCase.Home(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(home.Reports) != 1 || len(home.Reports[0].Cards) != 2 || home.Reports[0].NextCursor == nil || repository.chainQueries[0].Limit != 20 {
		t.Fatalf("home=%#v queries=%#v", home, repository.chainQueries)
	}
	if home.Reports[0].Cards[0].ImpactItems[0].ConclusionBasis.Code != "direct_evidence" {
		t.Fatalf("card=%#v", home.Reports[0].Cards[0])
	}
}

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

func TestIndustryChainsForwardsOpaqueCursorAndDoesNotLoadDetails(t *testing.T) {
	next := "next"
	repository := &fakeRepository{chainPage: IndustryChainPage{Items: []IndustryChainSummary{validChainSummary()}, NextCursor: &next}}
	useCase := NewUseCase(repository)
	page, err := useCase.IndustryChains(context.Background(), testReportID, 12, "cursor")
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].DetailRef.LocalKey != "chain-01" || repository.chainQueries[0].Cursor != "cursor" || repository.chainQueries[0].Limit != 12 {
		t.Fatalf("page=%#v queries=%#v", page, repository.chainQueries)
	}
}

func TestLayerLoadsEveryReportIndustryChainThroughDataPagination(t *testing.T) {
	summary := validSummary()
	summary.IndustryChainCount = 3
	secondCursor := "second-page"
	first := validChainSummary()
	second := validChainSummary()
	second.LocalKey, second.Name = "chain-02", "产业链二"
	third := validChainSummary()
	third.LocalKey, third.Name = "chain-03", "产业链三"
	repository := &fakeRepository{
		layer: LayerDetail{Report: summary, Layer: sampleLayer()},
		chainPages: map[string]IndustryChainPage{
			"":           {Items: []IndustryChainSummary{first, second}, NextCursor: &secondCursor},
			secondCursor: {Items: []IndustryChainSummary{third}},
		},
	}
	value, err := NewUseCase(repository).Layer(context.Background(), testReportID, LayerGeopolitics)
	if err != nil {
		t.Fatal(err)
	}
	if len(value.RelatedIndustryChains) != 3 || value.RelatedIndustryChains[2].LocalKey != "chain-03" ||
		len(repository.chainQueries) != 2 || repository.chainQueries[0].Limit != 100 {
		t.Fatalf("value=%#v queries=%#v", value, repository.chainQueries)
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
	return Summary{ID: testReportID, PublisherReportID: "publisher", GeneratedAt: time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC), PublishedAt: time.Date(2026, 9, 2, 1, 0, 0, 0, time.UTC), IndustryChainCount: 54}
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
	listPage     Page
	listPages    []Page
	listQueries  []ListQuery
	home         HomeSnapshot
	homes        map[string]HomeSnapshot
	layer        LayerDetail
	chainPage    IndustryChainPage
	chainPages   map[string]IndustryChainPage
	chain        IndustryChainDetail
	evidence     EvidenceCollection
	chainQueries []ChainListQuery
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
	return page, nil
}
func (f *fakeRepository) GetHome(_ context.Context, reportID string) (HomeSnapshot, error) {
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
