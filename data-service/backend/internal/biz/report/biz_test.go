package report_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	reportbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/report"
	reportfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/report"
)

func TestValidateReportAcceptsExactAgentOSFixture(t *testing.T) {
	payload, err := os.ReadFile(filepath.Join("..", "..", "..", "api", "data", "v1", "report", "testdata", "investment-report-publication-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	var request struct {
		Report reportbiz.Report `json:"report"`
	}
	if err := json.Unmarshal(payload, &request); err != nil {
		t.Fatal(err)
	}
	if err := reportbiz.ValidateReport(request.Report); err != nil {
		t.Fatalf("ValidateReport() error = %v", err)
	}
}

func TestValidateReportAcceptsIndustryOnlyReportWithFiftyFourChains(t *testing.T) {
	report := reportfixture.ReportWithManyChains(54)
	if report.Geopolitics != nil || report.Macroeconomics != nil {
		t.Fatal("optional upper sections were materialized")
	}
	if err := reportbiz.ValidateReport(report); err != nil {
		t.Fatalf("ValidateReport() error = %v", err)
	}
}

func TestFrozenScaleBaselineCardinalityAndEvidenceScopes(t *testing.T) {
	report := reportfixture.FrozenScaleBaselineReport()
	if err := reportbiz.ValidateReport(report); err != nil {
		t.Fatalf("ValidateReport() error = %v", err)
	}
	affectedNodes := 0
	for _, chain := range report.IndustryChains {
		affectedNodes += len(chain.Nodes)
	}
	if len(report.IndustryChains) != 54 || affectedNodes != 157 {
		t.Fatalf("chains=%d affected_nodes=%d", len(report.IndustryChains), affectedNodes)
	}

	evidenceIDs := reportfixture.FrozenScaleBaselineEvidenceIDs()
	store := newFakeStore(evidenceIDs...)
	useCase, err := reportbiz.NewUseCase(store, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := useCase.Publish(context.Background(), "publisher-frozen-scale-baseline", report); err != nil {
		t.Fatal(err)
	}
	unique := map[string]struct{}{}
	for _, link := range store.links {
		unique[link.EvidenceID] = struct{}{}
	}
	if len(unique) != 43 || len(store.links) != 265 {
		t.Fatalf("unique_evidence=%d links=%d", len(unique), len(store.links))
	}
}

func TestValidateReportAcceptsFrozenCodeLabelCatalog(t *testing.T) {
	report := reportfixture.Report()
	report.Geopolitics.AffectedAnchors[0].ValidationStatus = reportbiz.CodedLabel{Code: reportbiz.ValidationConfirmed, Label: "已确认"}
	if err := reportbiz.ValidateReport(report); err != nil {
		t.Fatalf("ValidateReport() error = %v", err)
	}
}

func TestValidateReportRejectsDomainContractViolations(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*reportbiz.Report)
		path   string
	}{
		{"no industry analysis", func(r *reportbiz.Report) { r.IndustryChains = []reportbiz.IndustryChain{} }, "report.industry_chains"},
		{"unknown transmission target", func(r *reportbiz.Report) {
			r.Geopolitics.DownwardTransmission.ToIndustryChains.Paths[0].Targets[0].TargetLocalKey = "missing"
		}, "targets[0]"},
		{"graph endpoint outside chain", func(r *reportbiz.Report) {
			r.IndustryChains[0].Edges = []reportbiz.IndustryChainEdge{{FromNodeLocalKey: "missing", ToNodeLocalKey: r.IndustryChains[0].Nodes[0].LocalKey, RelationLabel: "组成"}}
		}, "from_node_local_key"},
		{"hypothesis exposes evidence", func(r *reportbiz.Report) {
			r.IndustryChains[0].Nodes[0].ConclusionBasis = reportbiz.CodedLabel{Code: reportbiz.BasisReasoningHypothesis, Label: "推理假设"}
			r.IndustryChains[0].Nodes[0].ValidationStatus = reportbiz.CodedLabel{Code: reportbiz.ValidationPending, Label: "待验证"}
		}, "evidence_refs"},
		{"direct conclusion has no evidence", func(r *reportbiz.Report) { r.IndustryChains[0].Nodes[0].EvidenceRefs = []reportbiz.EvidenceReference{} }, "evidence_refs"},
		{"label drift", func(r *reportbiz.Report) { r.IndustryChains[0].Result.Label = "降温" }, "result.label"},
	} {
		t.Run(test.name, func(t *testing.T) {
			report := reportfixture.Report()
			test.mutate(&report)
			err := reportbiz.ValidateReport(report)
			if err == nil || !strings.Contains(err.Error(), test.path) {
				t.Fatalf("error=%v want path %s", err, test.path)
			}
		})
	}
}

func TestPublishCreatesImmutableReportAndScopedEvidenceThenReplays(t *testing.T) {
	store := newFakeStore(reportfixture.EvidenceOne, reportfixture.EvidenceTwo)
	clock := time.Date(2026, 9, 2, 1, 2, 3, 456789000, time.UTC)
	useCase, err := reportbiz.NewUseCase(store, func() time.Time { return clock })
	if err != nil {
		t.Fatal(err)
	}
	created, err := useCase.Publish(context.Background(), "publisher-report-2026-09-02", reportfixture.Report())
	if err != nil {
		t.Fatal(err)
	}
	if created.Replayed || created.Record.PublisherReportID != "publisher-report-2026-09-02" || !created.Record.PublishedAt.Equal(clock) {
		t.Fatalf("created=%#v", created)
	}
	want := map[reportbiz.ScopeType]bool{
		reportbiz.ScopeSectionSummary: false, reportbiz.ScopeAnchor: false,
		reportbiz.ScopeIndustryChainSummary: false, reportbiz.ScopeIndustryChainNode: false,
	}
	for _, link := range store.links {
		want[link.ScopeType] = true
		if link.ScopePath == "" || link.Position < 1 {
			t.Fatalf("invalid Evidence link %#v", link)
		}
	}
	for scope, seen := range want {
		if !seen {
			t.Errorf("scope %s did not produce a link", scope)
		}
	}
	replayed, err := useCase.Publish(context.Background(), "publisher-report-2026-09-02", reportfixture.Report())
	if err != nil || !replayed.Replayed || replayed.Record.ID != created.Record.ID || len(store.reports) != 1 {
		t.Fatalf("replayed=%#v err=%v", replayed, err)
	}
	changed := reportfixture.Report()
	changed.IndustryChains[0].Name = "另一条产业链"
	_, err = useCase.Publish(context.Background(), "publisher-report-2026-09-02", changed)
	if !errors.Is(err, reportbiz.ErrPublicationConflict) {
		t.Fatalf("conflict error=%v", err)
	}
}

func TestPublishRejectsMissingEvidenceAtomically(t *testing.T) {
	store := newFakeStore(reportfixture.EvidenceOne)
	useCase, _ := reportbiz.NewUseCase(store, time.Now)
	_, err := useCase.Publish(context.Background(), "publisher-report", reportfixture.Report())
	var reference *reportbiz.ReferenceError
	if !errors.As(err, &reference) || reference.Reference != reportfixture.EvidenceTwo {
		t.Fatalf("error=%v", err)
	}
	if len(store.reports) != 0 || len(store.links) != 0 {
		t.Fatal("failed publication mutated store")
	}
}

func TestIndustryChainCursorIsReportBound(t *testing.T) {
	store := newFakeStore()
	store.chainPage = reportbiz.IndustryChainStorePage{Items: []reportbiz.IndustryChainSummary{{LocalKey: "chain-01", Ordinal: 1}}, HasMore: true}
	useCase, _ := reportbiz.NewUseCase(store, time.Now)
	page, err := useCase.ListIndustryChains(context.Background(), reportbiz.IndustryChainListRequest{ReportID: reportfixture.ReportOne, Limit: 1})
	if err != nil || page.NextCursor == nil {
		t.Fatalf("page=%#v err=%v", page, err)
	}
	_, err = useCase.ListIndustryChains(context.Background(), reportbiz.IndustryChainListRequest{ReportID: "RPT22222222-2222-4222-8222-222222222222", Limit: 1, Cursor: *page.NextCursor})
	if err == nil {
		t.Fatal("cursor was accepted for another report")
	}
}

type fakeStore struct {
	existing    map[string]struct{}
	byPublisher map[string]reportbiz.Record
	reports     []reportbiz.Record
	links       []reportbiz.EvidenceLink
	page        reportbiz.StorePage
	chainPage   reportbiz.IndustryChainStorePage
}

func newFakeStore(ids ...string) *fakeStore {
	store := &fakeStore{existing: map[string]struct{}{}, byPublisher: map[string]reportbiz.Record{}}
	for _, id := range ids {
		store.existing[id] = struct{}{}
	}
	return store
}

func (s *fakeStore) InPublicationTransaction(ctx context.Context, fn func(reportbiz.PublicationTransaction) error) error {
	return fn((*fakeTransaction)(s))
}
func (s *fakeStore) ListReports(context.Context, reportbiz.ListFilter) (reportbiz.StorePage, error) {
	return s.page, nil
}
func (s *fakeStore) GetReport(context.Context, string) (reportbiz.Record, error) {
	return reportbiz.Record{}, reportbiz.ErrReportNotFound
}
func (s *fakeStore) GetHome(context.Context, string) (reportbiz.Home, error) {
	return reportbiz.Home{}, reportbiz.ErrReportNotFound
}
func (s *fakeStore) GetLayer(context.Context, string, string) (reportbiz.Summary, reportbiz.LayerProjection, error) {
	return reportbiz.Summary{}, reportbiz.LayerProjection{}, reportbiz.ErrReportNotFound
}
func (s *fakeStore) ListIndustryChains(context.Context, reportbiz.IndustryChainListFilter) (reportbiz.IndustryChainStorePage, error) {
	return s.chainPage, nil
}
func (s *fakeStore) GetIndustryChain(context.Context, string, string) (reportbiz.Summary, reportbiz.IndustryChainProjection, error) {
	return reportbiz.Summary{}, reportbiz.IndustryChainProjection{}, reportbiz.ErrReportNotFound
}
func (s *fakeStore) ListEvidence(context.Context, string, string) ([]reportbiz.Evidence, error) {
	return []reportbiz.Evidence{}, nil
}

type fakeTransaction fakeStore

func (*fakeTransaction) Lock(context.Context, string) error { return nil }
func (s *fakeTransaction) ReportByPublisherID(_ context.Context, id string) (*reportbiz.Record, error) {
	record, ok := s.byPublisher[id]
	if !ok {
		return nil, nil
	}
	return &record, nil
}
func (s *fakeTransaction) ExistingEvidenceIDs(_ context.Context, ids []string) ([]string, error) {
	result := []string{}
	for _, id := range ids {
		if _, ok := s.existing[id]; ok {
			result = append(result, id)
		}
	}
	return result, nil
}
func (s *fakeTransaction) InsertReport(_ context.Context, record reportbiz.Record) error {
	s.reports = append(s.reports, record)
	s.byPublisher[record.PublisherReportID] = record
	return nil
}
func (s *fakeTransaction) InsertEvidenceLinks(_ context.Context, links []reportbiz.EvidenceLink) error {
	s.links = append(s.links, links...)
	return nil
}

var _ reportbiz.Store = (*fakeStore)(nil)

func TestStoryConceptReportValidation(t *testing.T) {
	payload, err := os.ReadFile("../../../api/data/v1/report/testdata/story-concept-publication-request.json")
	if err != nil {
		t.Fatal(err)
	}
	read := func() reportbiz.Report {
		var request struct {
			Report reportbiz.Report `json:"report"`
		}
		if err := json.Unmarshal(payload, &request); err != nil {
			t.Fatal(err)
		}
		return request.Report
	}
	if err := reportbiz.ValidateReport(read()); err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(*reportbiz.Report){
		"unknown version":    func(r *reportbiz.Report) { r.SchemaVersion = "future" },
		"missing version":    func(r *reportbiz.Report) { r.SchemaVersion = "" },
		"cross-unit summary": func(r *reportbiz.Report) { r.GeopoliticalStories[0].Summary.AnchorKeys = []string{"geo-b-impact"} },
		"duplicate unit":     func(r *reportbiz.Report) { r.GeopoliticalStories[1].LocalKey = "geo-a" },
		"wrong hypothesis role": func(r *reportbiz.Report) {
			r.GeopoliticalStories[0].Detail.AffectedAnchors[0].EvidenceRefs[0].Role = reportbiz.CodedLabel{Code: "direct_support", Label: "直接依据"}
		},
		"cross-chain node": func(r *reportbiz.Report) {
			k := "chain-b-node"
			r.ConceptAnalyses[0].Detail.IndustryChains[0].AffectedNodes[0].NodeLocalKey = &k
		},
		"null collection": func(r *reportbiz.Report) { r.MacroeconomicStories = nil },
		"empty report": func(r *reportbiz.Report) {
			r.GeopoliticalStories = []reportbiz.AnalysisUnit{}
			r.ConceptAnalyses = []reportbiz.AnalysisUnit{}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			r := read()
			mutate(&r)
			if reportbiz.ValidateReport(r) == nil {
				t.Fatal("accepted invalid report")
			}
		})
	}
	r := read()
	r.ConceptAnalyses = []reportbiz.AnalysisUnit{}
	if err := reportbiz.ValidateReport(r); err != nil {
		t.Fatalf("story-only report: %v", err)
	}
	store := newFakeStore("EVD11111111-1111-4111-8111-111111111111")
	uc, _ := reportbiz.NewUseCase(store, time.Now)
	_, err = uc.Publish(context.Background(), "v3-test", read())
	if err != nil {
		t.Fatal(err)
	}
	if len(store.links) != 14 {
		t.Fatalf("scoped Evidence links=%d want 14", len(store.links))
	}
}

func (*fakeStore) ListAnalyses(context.Context, reportbiz.AnalysisListFilter) (reportbiz.AnalysisStorePage, error) {
	return reportbiz.AnalysisStorePage{}, nil
}
func (*fakeStore) GetAnalysis(context.Context, string, string, string) (reportbiz.AnalysisUnitDetail, error) {
	return reportbiz.AnalysisUnitDetail{}, nil
}
func (*fakeStore) GetAnalysisChain(context.Context, string, string, string, string) (reportbiz.ChainAnalysisDetail, error) {
	return reportbiz.ChainAnalysisDetail{}, nil
}

func TestStoryChainPublicationValidation(t *testing.T) {
	payload, err := os.ReadFile("../../../api/data/v1/report/testdata/story-chain-publication-request.json")
	if err != nil {
		t.Fatal(err)
	}
	read := func() reportbiz.Report {
		var r struct {
			Report reportbiz.Report `json:"report"`
		}
		if err := json.Unmarshal(payload, &r); err != nil {
			t.Fatal(err)
		}
		return r.Report
	}
	if err := reportbiz.ValidateReport(read()); err != nil {
		t.Fatal(err)
	}

	t.Run("anchor order", func(t *testing.T) {
		for _, reverse := range []bool{false, true} {
			r := read()
			u := &r.GeopoliticalStories[0]
			a := u.Detail.AffectedAnchors[0]
			a.LocalKey = "alternate-anchor"
			a.Name = "alternate snapshot name"
			u.Detail.AffectedAnchors = append(u.Detail.AffectedAnchors, a)
			if reverse {
				u.Detail.AffectedAnchors[0], u.Detail.AffectedAnchors[1] = u.Detail.AffectedAnchors[1], u.Detail.AffectedAnchors[0]
			}
			if err := reportbiz.ValidateReport(r); err != nil {
				t.Fatal(err)
			}
		}
	})
	t.Run("historical exact replay", func(t *testing.T) {
		r := read()
		u := &r.GeopoliticalStories[0]
		u.Detail.IndustryChains = []reportbiz.ChainAnalysis{}
		u.Detail.AffectedAnchors[0].TargetType = reportbiz.CodedLabel{Code: "industry_chain_node", Label: "产业链节点"}
		hash, err := reportbiz.ContentHash(r)
		if err != nil {
			t.Fatal(err)
		}
		store := newFakeStore()
		store.byPublisher["historical"] = reportbiz.Record{PublisherReportID: "historical", ContentHash: hash, Report: r}
		uc, _ := reportbiz.NewUseCase(store, time.Now)
		result, err := uc.Publish(context.Background(), "historical", r)
		if err != nil || !result.Replayed {
			t.Fatalf("historical replay=%+v err=%v", result, err)
		}
		if _, err := uc.Publish(context.Background(), "new-invalid", r); err == nil {
			t.Fatal("new invalid hierarchy accepted")
		}
		r.GeopoliticalStories[0].Summary.Conclusion = "changed"
		if _, err := uc.Publish(context.Background(), "historical", r); !errors.Is(err, reportbiz.ErrPublicationConflict) {
			t.Fatalf("historical conflict=%v", err)
		}
	})
	for name, mutate := range map[string]func(*reportbiz.Report){
		"story summary node": func(r *reportbiz.Report) {
			r.GeopoliticalStories[0].Summary.AnchorKeys = []string{r.GeopoliticalStories[0].Detail.IndustryChains[0].AffectedNodes[0].LocalKey}
		},
		"unanchored chain":    func(r *reportbiz.Report) { r.GeopoliticalStories[0].Detail.IndustryChains[0].SourceID = "other" },
		"chain name mismatch": func(r *reportbiz.Report) { r.GeopoliticalStories[0].Detail.IndustryChains[0].Name = "other" },
		"macro targets macro": func(r *reportbiz.Report) {
			r.MacroeconomicStories[0].Detail.AffectedAnchors[0].TargetType = reportbiz.CodedLabel{Code: "macro_anchor", Label: "宏观经济锚点"}
		},
		"geo targets node": func(r *reportbiz.Report) {
			r.GeopoliticalStories[0].Detail.AffectedAnchors[0].TargetType = reportbiz.CodedLabel{Code: "industry_chain_node", Label: "产业链节点"}
		},
		"cross story node": func(r *reportbiz.Report) {
			k := r.MacroeconomicStories[0].Detail.IndustryChains[0].Graph.Nodes[0].LocalKey
			r.GeopoliticalStories[0].Detail.IndustryChains[0].AffectedNodes[0].NodeLocalKey = &k
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := read()
			mutate(&r)
			if reportbiz.ValidateReport(r) == nil {
				t.Fatal("invalid story chain accepted")
			}
		})
	}
}

func TestImpactAssessmentValidation(t *testing.T) {
	payload, err := os.ReadFile("../../../api/data/v1/report/testdata/story-chain-publication-request.json")
	if err != nil {
		t.Fatal(err)
	}
	read := func() reportbiz.Report {
		var r struct {
			Report reportbiz.Report `json:"report"`
		}
		if err := json.Unmarshal(payload, &r); err != nil {
			t.Fatal(err)
		}
		return r.Report
	}
	for _, kind := range []string{"geo", "macro", "concept"} {
		for code, label := range map[string]string{"high": "高影响", "medium": "中影响", "low": "低影响", "pending": "待评估"} {
			r := read()
			a := r.GeopoliticalStories[0].Summary.ImpactAssessment
			if kind == "macro" {
				a = r.MacroeconomicStories[0].Summary.ImpactAssessment
			}
			if kind == "concept" {
				a = r.ConceptAnalyses[0].Summary.ImpactAssessment
			}
			a.Level = reportbiz.CodedLabel{Code: code, Label: label}
			if code == "pending" {
				a.EvidenceRefs = []reportbiz.EvidenceReference{}
			}
			if err := reportbiz.ValidateReport(r); err != nil {
				t.Fatal(err)
			}
		}
	}
	for name, mutate := range map[string]func(*reportbiz.ImpactAssessment){
		"unknown level":          func(a *reportbiz.ImpactAssessment) { a.Level.Code = "severe" },
		"wrong label":            func(a *reportbiz.ImpactAssessment) { a.Level.Label = "低影响" },
		"blank rationale":        func(a *reportbiz.ImpactAssessment) { a.Rationale = " " },
		"long rationale":         func(a *reportbiz.ImpactAssessment) { a.Rationale = strings.Repeat("a", 10001) },
		"missing rated Evidence": func(a *reportbiz.ImpactAssessment) { a.EvidenceRefs = []reportbiz.EvidenceReference{} },
		"null pending Evidence": func(a *reportbiz.ImpactAssessment) {
			a.Level = reportbiz.CodedLabel{Code: "pending", Label: "待评估"}
			a.EvidenceRefs = nil
		},
		"wrong role": func(a *reportbiz.ImpactAssessment) {
			a.EvidenceRefs[0].Role = reportbiz.CodedLabel{Code: "direct_support", Label: "直接依据"}
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := read()
			mutate(r.GeopoliticalStories[0].Summary.ImpactAssessment)
			if reportbiz.ValidateReport(r) == nil {
				t.Fatal("invalid assessment accepted")
			}
		})
	}
	r := read()
	r.GeopoliticalStories[0].Summary.ImpactAssessment = nil
	r.MacroeconomicStories[0].Summary.ImpactAssessment = nil
	r.ConceptAnalyses[0].Summary.ImpactAssessment = nil
	wire, err := json.Marshal(r)
	if err != nil || strings.Contains(string(wire), "impact_assessment") {
		t.Fatalf("legacy wire changed: %v", err)
	}
	if err := reportbiz.ValidateReport(r); err != nil {
		t.Fatal(err)
	}
}

func normalizedFixture(t *testing.T) reportbiz.Report {
	t.Helper()
	payload, err := os.ReadFile("../../../api/data/v1/report/testdata/normalized-publication-request.json")
	if err != nil {
		t.Fatal(err)
	}
	var req struct {
		Report reportbiz.Report `json:"report"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatal(err)
	}
	return req.Report
}
func TestNormalizedReportContractAndReferenceRules(t *testing.T) {
	r := normalizedFixture(t)
	if err := reportbiz.ValidateReport(r); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		mutate func(*reportbiz.V4Report)
	}{
		{"cross unit reference", func(r *reportbiz.V4Report) { r.GeopoliticalStories[0].Summary.AffectedRefs[0].LocalKey = "missing" }},
		{"rated impact without evidence", func(r *reportbiz.V4Report) {
			r.GeopoliticalStories[0].Summary.ImpactAssessment.EvidenceIDs = []string{}
		}},
		{"unknown enum", func(r *reportbiz.V4Report) {
			r.ConceptAnalyses[0].Detail.IndustryChains[0].Assessment.Direction = "invented"
		}},
		{"false counterfact status", func(r *reportbiz.V4Report) {
			r.GeopoliticalStories[0].Detail.IndustryChains[0].ReasoningSummary.Objections.CounterevidenceStatus = "identified"
		}},
		{"node identity mismatch", func(r *reportbiz.V4Report) {
			r.GeopoliticalStories[0].Detail.IndustryChains[0].AffectedNodes[0].Name = "wrong"
		}},
		{"empty state mismatch", func(r *reportbiz.V4Report) { r.ConceptAnalyses[0].Detail.IndustryChains[2].EmptyState = nil }},
		{"missing conditions", func(r *reportbiz.V4Report) {
			r.GeopoliticalStories[0].Detail.IndustryChains[0].AffectedNodes[0].Assessment.Conditions = []string{}
		}},
		{"macro targets under macro", func(r *reportbiz.V4Report) {
			r.MacroeconomicStories[0].Detail.MacroImpacts = r.GeopoliticalStories[0].Detail.MacroImpacts
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := normalizedFixture(t)
			tc.mutate(r.V4)
			if err := reportbiz.ValidateReport(r); err == nil {
				t.Fatal("invalid report accepted")
			}
		})
	}
	payload, _ := json.Marshal(r)
	var decoded reportbiz.Report
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	a, _ := reportbiz.ContentHash(r)
	b, _ := reportbiz.ContentHash(decoded)
	if a != b {
		t.Fatal("normalized canonical round trip changed")
	}
}

func TestNormalizedLocalKeysAreScopedToContainers(t *testing.T) {
	r := normalizedFixture(t)
	// A second story may reuse every detail key; references stay within that story.
	clone := normalizedFixture(t).V4.GeopoliticalStories[0]
	clone.LocalKey = "second-story"
	clone.SourceID = "GPR22222222-2222-4222-8222-222222222222"
	r.V4.GeopoliticalStories = append(r.V4.GeopoliticalStories, clone)
	if err := reportbiz.ValidateReport(r); err != nil {
		t.Fatal(err)
	}
	r.V4.GeopoliticalStories[1].LocalKey = r.V4.GeopoliticalStories[0].LocalKey
	if err := reportbiz.ValidateReport(r); err == nil {
		t.Fatal("duplicate sibling key accepted")
	}
}

func TestPublicationComputesScopeCountsAndKeepsReplayHash(t *testing.T) {
	r := normalizedFixture(t)
	store := newFakeStore(reportfixture.EvidenceOne)
	uc, _ := reportbiz.NewUseCase(store, time.Now)
	first, err := uc.Publish(context.Background(), "scope-counts", r)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Record.EvidenceCounts) == 0 {
		t.Fatal("missing counts")
	}
	for _, n := range first.Record.EvidenceCounts {
		if n != 1 {
			t.Fatalf("scope count %d", n)
		}
	}
	replay, err := uc.Publish(context.Background(), "scope-counts", r)
	if err != nil || !replay.Replayed || replay.Record.ContentHash != first.Record.ContentHash {
		t.Fatalf("replay %v %v", replay, err)
	}
	raw, _ := json.Marshal(r)
	raw = bytes.Replace(raw, []byte(`"summary":{`), []byte(`"summary":{"evidence_count":999,`), 1)
	var supplied reportbiz.Report
	if err := json.Unmarshal(raw, &supplied); err == nil {
		t.Fatal("publisher supplied derived count accepted")
	}
}
