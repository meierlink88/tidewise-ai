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

func TestValidateContentAcceptsSelectedCardsAcrossManyIndustryChains(t *testing.T) {
	content := reportfixture.ContentWithManyChains(54)
	if len(content.ReportCards) != 4 {
		t.Fatalf("Report card count = %d, want 4", len(content.ReportCards))
	}
	if err := reportbiz.ValidateContent(content); err != nil {
		t.Fatalf("ValidateContent() error = %v", err)
	}
}

func TestValidateContentRejectsBrokenClosedWorldAndSnapshotContracts(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*reportbiz.Content)
		path   string
	}{
		{name: "duplicate anchor key across layers", mutate: func(content *reportbiz.Content) {
			content.Macroeconomics.Anchors[0].Key = content.Geopolitics.Anchors[0].Key
			content.Macroeconomics.RelatedAnchorKeys[0] = content.Geopolitics.Anchors[0].Key
			content.ReportCards[1].ImpactItems[0].Ref.Key = content.Geopolitics.Anchors[0].Key
		}, path: "content.macroeconomics.anchors[0].key"},
		{name: "result label mismatch", mutate: func(content *reportbiz.Content) {
			content.Geopolitics.Result.Label = "降温"
		}, path: "content.geopolitics.result.label"},
		{name: "nature label mismatch", mutate: func(content *reportbiz.Content) {
			content.Geopolitics.Anchors[0].Nature.Label = "推理假设"
		}, path: "content.geopolitics.anchors[0].nature.label"},
		{name: "company layer accidentally published", mutate: func(content *reportbiz.Content) {
			content.Company.Published = true
		}, path: "content.company.published"},
		{name: "empty impact items", mutate: func(content *reportbiz.Content) {
			content.ReportCards[0].ImpactItems = []reportbiz.ImpactItem{}
		}, path: "content.report_cards[0].impact_items"},
		{name: "duplicate related chain", mutate: func(content *reportbiz.Content) {
			content.Geopolitics.RelatedChainKeys = []string{"chain-01", "chain-01"}
		}, path: "content.geopolitics.related_chain_keys[1]"},
		{name: "unknown structured target", mutate: func(content *reportbiz.Content) {
			content.Geopolitics.DownwardTransmission.PublishedPaths[0].TargetRefs[0].Ref.Key = "unknown-chain"
		}, path: "content.geopolitics.downward_transmission.published_paths[0].target_refs[0].ref"},
		{name: "card differs from detail snapshot", mutate: func(content *reportbiz.Content) {
			content.ReportCards[0].Conclusion = "不一致"
		}, path: "content.report_cards[0]"},
		{name: "no industry card", mutate: func(content *reportbiz.Content) {
			content.ReportCards = content.ReportCards[:2]
		}, path: "content.report_cards"},
		{name: "source text contains boundary whitespace", mutate: func(content *reportbiz.Content) {
			content.Title = " 每日推理报告"
		}, path: "content.title"},
	} {
		t.Run(test.name, func(t *testing.T) {
			content := reportfixture.Content()
			test.mutate(&content)
			err := reportbiz.ValidateContent(content)
			if err == nil || !strings.Contains(err.Error(), test.path) {
				t.Fatalf("ValidateContent() error = %v, want path %s", err, test.path)
			}
		})
	}
}

func TestPublishCreatesImmutableReportAndExactEvidenceLinksThenReplays(t *testing.T) {
	store := newFakeStore(reportfixture.EvidenceOne, reportfixture.EvidenceTwo)
	clock := time.Date(2026, 9, 1, 1, 2, 3, 456789000, time.UTC)
	useCase, err := reportbiz.NewUseCase(store, func() time.Time { return clock })
	if err != nil {
		t.Fatal(err)
	}
	content := reportfixture.Content()
	created, err := useCase.Publish(context.Background(), reportbiz.ContractVersion, "agentos-report-2026-09-01-a", content)
	if err != nil {
		t.Fatal(err)
	}
	if created.Replayed || created.Record.ID == "" || created.Record.ContentHash != created.ContentHash || !created.Record.PublishedAt.Equal(clock) {
		t.Fatalf("created = %#v", created)
	}
	if len(store.reports) != 1 || len(store.links) == 0 {
		t.Fatalf("writes reports=%d links=%d", len(store.reports), len(store.links))
	}
	wantScopes := map[reportbiz.ScopeType]bool{
		reportbiz.ScopeReportCard: false, reportbiz.ScopeLayer: false, reportbiz.ScopeAnchor: false,
		reportbiz.ScopeReasoningStep: false, reportbiz.ScopeTransmissionPath: false,
		reportbiz.ScopeCandidateMechanism: false, reportbiz.ScopeIndustryChain: false,
		reportbiz.ScopeIndustryChainNode: false,
	}
	for _, link := range store.links {
		wantScopes[link.ScopeType] = true
		if strings.Contains(link.ScopeKey, "/") {
			t.Fatalf("scope key %q was composed instead of preserving the Report-local key", link.ScopeKey)
		}
		if link.ReportID != created.Record.ID || link.DisplayOrder < 1 || strings.TrimSpace(link.Role) == "" {
			t.Fatalf("invalid link = %#v", link)
		}
	}
	for scope, seen := range wantScopes {
		if !seen {
			t.Errorf("scope %s did not produce a relationship row", scope)
		}
	}

	replayed, err := useCase.Publish(context.Background(), reportbiz.ContractVersion, "agentos-report-2026-09-01-a", content)
	if err != nil {
		t.Fatal(err)
	}
	if !replayed.Replayed || replayed.Record.ID != created.Record.ID || len(store.reports) != 1 {
		t.Fatalf("replayed = %#v reports=%d", replayed, len(store.reports))
	}

	changed := reportfixture.Content()
	changed.Title = "另一份报告"
	_, err = useCase.Publish(context.Background(), reportbiz.ContractVersion, "agentos-report-2026-09-01-a", changed)
	if !errors.Is(err, reportbiz.ErrPublicationConflict) || len(store.reports) != 1 {
		t.Fatalf("conflict error = %v reports=%d", err, len(store.reports))
	}
}

func TestPublishRejectsMissingEvidenceAtomicallyAndDoesNotNormalizeSourceIdentity(t *testing.T) {
	store := newFakeStore(reportfixture.EvidenceOne)
	useCase, err := reportbiz.NewUseCase(store, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = useCase.Publish(context.Background(), reportbiz.ContractVersion, "agentos-report-2026-09-01-a", reportfixture.Content())
	var reference *reportbiz.ReferenceError
	if !errors.As(err, &reference) || reference.Reference != reportfixture.EvidenceTwo {
		t.Fatalf("missing Evidence error = %v", err)
	}
	if len(store.reports) != 0 || len(store.links) != 0 {
		t.Fatalf("missing Evidence mutated reports=%d links=%d", len(store.reports), len(store.links))
	}
	_, err = useCase.Publish(context.Background(), reportbiz.ContractVersion, " agentos-report-2026-09-01-a", reportfixture.Content())
	var validation *reportbiz.ValidationError
	if !errors.As(err, &validation) || validation.Path != "source_report_id" {
		t.Fatalf("whitespace source_report_id error = %v", err)
	}
}

func TestListProducesFilterBoundStableCursor(t *testing.T) {
	store := newFakeStore()
	publishedAt := time.Date(2026, 9, 1, 2, 0, 0, 0, time.UTC)
	store.page = reportbiz.StorePage{Items: []reportbiz.Summary{{
		ID: reportfixture.ReportOne, PublishedAt: publishedAt,
	}}, HasMore: true}
	useCase, err := reportbiz.NewUseCase(store, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	from := publishedAt.Add(-time.Hour)
	page, err := useCase.List(context.Background(), reportbiz.ListRequest{PublishedFrom: &from, Limit: 1})
	if err != nil || page.NextCursor == nil {
		t.Fatalf("List() page=%#v error=%v", page, err)
	}
	if _, err := useCase.List(context.Background(), reportbiz.ListRequest{Limit: 1, Cursor: *page.NextCursor}); err == nil {
		t.Fatal("cursor was accepted with a different date filter")
	}
}

func TestReadIdentitiesAndScopeKeysDoNotNormalizeBoundaryWhitespace(t *testing.T) {
	useCase, err := reportbiz.NewUseCase(newFakeStore(), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := useCase.GetHome(ctx, " "+reportfixture.ReportOne); err == nil {
		t.Fatal("GetHome accepted a Report ID with boundary whitespace")
	}
	if _, _, _, err := useCase.GetLayer(ctx, reportfixture.ReportOne, " geopolitics"); !errors.Is(err, reportbiz.ErrLayerNotFound) {
		t.Fatalf("GetLayer whitespace error = %v", err)
	}
	if _, _, err := useCase.GetIndustryChain(ctx, reportfixture.ReportOne, " chain-01"); !errors.Is(err, reportbiz.ErrChainNotFound) {
		t.Fatalf("GetIndustryChain whitespace error = %v", err)
	}
	if _, err := useCase.ListEvidence(ctx, reportfixture.ReportOne, reportbiz.ScopeAnchor, " geo-anchor"); err == nil {
		t.Fatal("ListEvidence accepted a scope key with boundary whitespace")
	}
}

type fakeStore struct {
	existing map[string]struct{}
	bySource map[string]reportbiz.Record
	reports  []reportbiz.Record
	links    []reportbiz.EvidenceLink
	page     reportbiz.StorePage
}

func newFakeStore(evidenceIDs ...string) *fakeStore {
	store := &fakeStore{existing: map[string]struct{}{}, bySource: map[string]reportbiz.Record{}}
	for _, id := range evidenceIDs {
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

func (s *fakeStore) GetLayer(context.Context, string, string) (reportbiz.Summary, reportbiz.Layer, []reportbiz.IndustryChainSummary, error) {
	return reportbiz.Summary{}, reportbiz.Layer{}, []reportbiz.IndustryChainSummary{}, reportbiz.ErrReportNotFound
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

func (s *fakeTransaction) ReportBySourceID(_ context.Context, sourceID string) (*reportbiz.Record, error) {
	record, ok := s.bySource[sourceID]
	if !ok {
		return nil, nil
	}
	return &record, nil
}

func (s *fakeTransaction) ExistingEvidenceIDs(_ context.Context, ids []string) ([]string, error) {
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, ok := s.existing[id]; ok {
			result = append(result, id)
		}
	}
	return result, nil
}

func (s *fakeTransaction) InsertReport(_ context.Context, record reportbiz.Record) error {
	s.reports = append(s.reports, record)
	s.bySource[record.SourceReportID] = record
	return nil
}

func (s *fakeTransaction) InsertEvidenceLinks(_ context.Context, links []reportbiz.EvidenceLink) error {
	s.links = append(s.links, links...)
	return nil
}

var _ reportbiz.Store = (*fakeStore)(nil)
