package report_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	reportbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/report"
	reportfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/report"
)

func TestValidateContentAcceptsIndustryOnlyReportWithFiftyFourChains(t *testing.T) {
	content := reportfixture.ContentWithManyChains(54)
	if content.Geopolitics != nil || content.Macroeconomics != nil {
		t.Fatal("optional upper sections were materialized")
	}
	if err := reportbiz.ValidateContent(content); err != nil {
		t.Fatalf("ValidateContent() error = %v", err)
	}
}

func TestValidateContentPreservesMixedResultLabelFromReport(t *testing.T) {
	content := reportfixture.Content()
	content.Geopolitics.Detail.Anchors[0].Result = reportbiz.Result{Code: reportbiz.ResultMixed, Label: "升温 / 局部稳定"}
	if err := reportbiz.ValidateContent(content); err != nil {
		t.Fatalf("ValidateContent() error = %v", err)
	}
}

func TestValidateContentRejectsDomainContractViolations(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*reportbiz.Content)
		path   string
	}{
		{"no industry analysis", func(c *reportbiz.Content) {
			c.IndustryChains = []reportbiz.IndustryChain{}
			c.Statistics.IndustryChainCount = 0
		}, "content.industry_chains"},
		{"structural count mismatch", func(c *reportbiz.Content) { c.Statistics.IndustryChainCount = 2 }, "content.statistics"},
		{"unknown related chain", func(c *reportbiz.Content) { c.Geopolitics.Detail.RelatedChainKeys = []string{"missing"} }, "content.geopolitics.detail.related_chain_keys[0]"},
		{"present layer has no downward transmission", func(c *reportbiz.Content) { c.Geopolitics.Summary.Transmissions = []reportbiz.Transmission{} }, "content.geopolitics.summary.transmissions"},
		{"present layer has no affected anchor", func(c *reportbiz.Content) { c.Geopolitics.Detail.Anchors = []reportbiz.Anchor{} }, "content.geopolitics.detail.anchors"},
		{"present layer has no boundary", func(c *reportbiz.Content) { c.Geopolitics.Summary.Uncertainty.Boundary = nil }, "content.geopolitics.summary.uncertainty.boundary"},
		{"chain summary has no graph nodes", func(c *reportbiz.Content) {
			c.IndustryChains[0].Summary.Graph.Nodes = []reportbiz.IndustryChainTopologyNode{}
		}, "content.industry_chains[0]"},
		{"chain summary has no counterevidence", func(c *reportbiz.Content) { c.IndustryChains[0].Summary.Uncertainty.CounterevidenceAndGap = "" }, "content.industry_chains[0].summary.uncertainty.counterevidence_and_gap"},
		{"chain detail has no affected nodes", func(c *reportbiz.Content) { c.IndustryChains[0].Detail.NodeImpacts = []reportbiz.IndustryChainNode{} }, "content.industry_chains[0]"},
		{"topology and impact mixed", func(c *reportbiz.Content) { c.IndustryChains[0].Detail.NodeImpacts[0].NodeKey = "missing" }, "content.industry_chains[0].detail.node_impacts[0].node_key"},
		{"hypothesis cites direct evidence", func(c *reportbiz.Content) {
			c.IndustryChains[0].Detail.NodeImpacts[0].Nature = reportbiz.Nature{Code: reportbiz.NatureReasoningHypothesis, Label: "推理假设"}
		}, "content.industry_chains[0].detail.node_impacts[0].evidence_refs"},
		{"direct conclusion has no evidence", func(c *reportbiz.Content) {
			c.IndustryChains[0].Detail.NodeImpacts[0].EvidenceRefs = []reportbiz.EvidenceReference{}
		}, "content.industry_chains[0].detail.node_impacts[0].evidence_refs"},
		{"claim label drift", func(c *reportbiz.Content) { c.IndustryChains[0].Summary.Result.Label = "降温" }, "content.industry_chains[0].summary.result.label"},
		{"duplicate claim key", func(c *reportbiz.Content) {
			c.IndustryChains[0].Summary.Claim.Key = c.Geopolitics.Summary.Claim.Key
		}, "content.industry_chains[0].summary.claim.key"},
	} {
		t.Run(test.name, func(t *testing.T) {
			content := reportfixture.Content()
			test.mutate(&content)
			err := reportbiz.ValidateContent(content)
			if err == nil || !strings.Contains(err.Error(), test.path) {
				t.Fatalf("error=%v want path %s", err, test.path)
			}
		})
	}
}

func TestPublishCreatesImmutableReportAndExactEvidenceLinksThenReplays(t *testing.T) {
	store := newFakeStore(reportfixture.EvidenceOne, reportfixture.EvidenceTwo)
	clock := time.Date(2026, 9, 2, 1, 2, 3, 456789000, time.UTC)
	useCase, err := reportbiz.NewUseCase(store, func() time.Time { return clock })
	if err != nil {
		t.Fatal(err)
	}
	created, err := useCase.Publish(context.Background(), reportbiz.ContractVersion, "publisher-report-2026-09-02", reportfixture.Content())
	if err != nil {
		t.Fatal(err)
	}
	if created.Replayed || created.Record.PublisherReportID != "publisher-report-2026-09-02" || !created.Record.PublishedAt.Equal(clock) {
		t.Fatalf("created=%#v", created)
	}
	want := map[reportbiz.ScopeType]bool{reportbiz.ScopeSectionSummary: false, reportbiz.ScopeAnchor: false, reportbiz.ScopeTransmission: false, reportbiz.ScopeIndustryChainSummary: false, reportbiz.ScopeIndustryChainNode: false}
	for _, link := range store.links {
		want[link.ScopeType] = true
	}
	for scope, seen := range want {
		if !seen {
			t.Errorf("scope %s did not produce a link", scope)
		}
	}
	replayed, err := useCase.Publish(context.Background(), reportbiz.ContractVersion, "publisher-report-2026-09-02", reportfixture.Content())
	if err != nil {
		t.Fatal(err)
	}
	if !replayed.Replayed || replayed.Record.ID != created.Record.ID || len(store.reports) != 1 {
		t.Fatalf("replayed=%#v", replayed)
	}
	changed := reportfixture.Content()
	changed.Title = "另一份报告"
	_, err = useCase.Publish(context.Background(), reportbiz.ContractVersion, "publisher-report-2026-09-02", changed)
	if !errors.Is(err, reportbiz.ErrPublicationConflict) {
		t.Fatalf("conflict error=%v", err)
	}
}

func TestPublishRejectsMissingEvidenceAtomically(t *testing.T) {
	store := newFakeStore(reportfixture.EvidenceOne)
	useCase, _ := reportbiz.NewUseCase(store, time.Now)
	_, err := useCase.Publish(context.Background(), reportbiz.ContractVersion, "publisher-report", reportfixture.Content())
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
	store.chainPage = reportbiz.IndustryChainStorePage{Items: []reportbiz.IndustryChainSummary{{Key: "chain-01", DisplayOrder: 1}}, HasMore: true}
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
	s := &fakeStore{existing: map[string]struct{}{}, byPublisher: map[string]reportbiz.Record{}}
	for _, id := range ids {
		s.existing[id] = struct{}{}
	}
	return s
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
func (s *fakeStore) GetLayer(context.Context, string, string) (reportbiz.Summary, reportbiz.Layer, []reportbiz.IndustryChainSummary, error) {
	return reportbiz.Summary{}, reportbiz.Layer{}, nil, reportbiz.ErrReportNotFound
}
func (s *fakeStore) ListIndustryChains(context.Context, reportbiz.IndustryChainListFilter) (reportbiz.IndustryChainStorePage, error) {
	return s.chainPage, nil
}
func (s *fakeStore) GetIndustryChain(context.Context, string, string) (reportbiz.Summary, reportbiz.IndustryChain, error) {
	return reportbiz.Summary{}, reportbiz.IndustryChain{}, reportbiz.ErrReportNotFound
}
func (s *fakeStore) ReportScopeExists(context.Context, string, reportbiz.ScopeType, string) (bool, bool, error) {
	return false, false, nil
}
func (s *fakeStore) ListEvidence(context.Context, string, reportbiz.ScopeType, string) ([]reportbiz.Evidence, error) {
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
