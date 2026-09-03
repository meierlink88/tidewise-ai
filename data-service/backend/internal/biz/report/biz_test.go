package report_test

import (
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
