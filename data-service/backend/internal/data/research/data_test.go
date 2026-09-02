package research

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	eventbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/event"
	evidencebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/evidence"
	researchbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/research"
	entitydata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity"
	eventdata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/event"
	evidencedata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/evidence"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

func TestResearchThemeAdapterRejectsMalformedPersistedRows(t *testing.T) {
	now := time.Date(2026, 8, 12, 8, 0, 0, 0, time.UTC)
	valid := ResearchThemeSummary{
		ID: "RTH11111111-1111-4111-8111-111111111111", AnalysisBatchID: "batch:one",
		Title: "Theme", OneLineConclusion: "Conclusion", ConclusionDirection: "positive",
		ImpactStrength: "medium", TransmissionStage: "validation", InvestmentGuidanceAction: "observe",
		InvestmentGuidanceSummary: "Observe", TimeHorizonCategory: "medium_term",
		AnalysisAsOf: now, WindowStart: now.Add(-time.Hour), WindowEnd: now, PublishedAt: now,
		Impacts: []ResearchThemeImpact{{NodeKey: "node:one", DisplayName: "Node", RelationRole: "driver", ImpactDirection: "positive", DisplayOrder: 1}},
	}
	if err := validatePersistedResearchThemeSummary(valid); err != nil {
		t.Fatalf("valid persisted Theme rejected: %v", err)
	}
	invalid := valid
	invalid.ConclusionDirection = "invented"
	if err := validatePersistedResearchThemeSummary(invalid); err == nil {
		t.Fatal("malformed persisted Theme enum was accepted")
	}
	invalid = valid
	invalid.Impacts = append(append([]ResearchThemeImpact(nil), valid.Impacts...), valid.Impacts[0])
	invalid.Impacts[1].DisplayOrder = 2
	if err := validatePersistedResearchThemeSummary(invalid); err == nil {
		t.Fatal("duplicated persisted Impact identity was accepted")
	}
}

func TestResearchReasoningTreeAdapterRejectsMalformedSnapshotRows(t *testing.T) {
	now := time.Date(2026, 8, 12, 8, 0, 0, 0, time.UTC)
	treeID := "RRT11111111-1111-4111-8111-111111111111"
	nodeID := "RRN55555555-5555-4555-8555-555555555555"
	publication := researchReasoningTreePublication{
		ReceiptID: "RRI33333333-3333-4333-8333-333333333333",
		Mapping:   map[string]string{"tree:one": treeID},
		Counts:    researchbiz.ReasonTreeCounts{ReasoningTrees: 1, Nodes: 1, SignalAssociations: 1, Receipts: 1},
		Trees: []ResearchReasoningTreeSummary{{
			ReasoningTreeID: treeID, TreeKey: "tree:one", DisplayName: "Chain",
			Title: "Tree", DisplayOrder: 1, PublishedAt: now,
		}},
	}
	if !validReasoningTreePublication(publication, 1, 0, 1) {
		t.Fatal("valid persisted snapshot Reasoning Tree publication was rejected")
	}
	invalidPublication := publication
	invalidPublication.Trees = append([]ResearchReasoningTreeSummary(nil), publication.Trees...)
	invalidPublication.Trees[0].ReasoningTreeID = "not-an-object-id"
	if validReasoningTreePublication(invalidPublication, 1, 0, 1) {
		t.Fatal("malformed persisted Reasoning Tree identity was accepted")
	}

	detail := ResearchReasoningTreeDetail{
		ThemeKey: "theme:one", PublicationMode: researchbiz.SnapshotPublicationMode, PublicationContractVersion: 3,
	}
	tree := ResearchReasoningTree{
		ReasoningTreeID: treeID, ThemeID: "RTH44444444-4444-4444-8444-444444444444",
		TreeKey: "tree:one", DisplayName: "Chain", Title: "Tree", OneLineConclusion: "Conclusion",
		ImpactDirection: "positive", ImpactStrength: "medium", DisplayOrder: 1, PublishedAt: now,
		Nodes: []researchbiz.ReasoningTreeNodeRecord{{
			ID: nodeID, NodeKey: "node:one", DisplayName: "Node", Position: 1,
			ImpactDirection: "positive", ImpactStrength: "medium",
			Signals: []researchbiz.SignalRecord{{
				SignalKey: "signal:one", SignalRole: "primary", Direction: stringPointer("increase"),
				DisplaySummary: "Signal", DisplayOrder: 1,
			}},
		}},
	}
	if !validReasoningTreeDetail(detail, tree, []string{"node:one"}) {
		t.Fatal("valid persisted snapshot Reasoning Tree detail was rejected")
	}
	invalidTree := tree
	invalidTree.Nodes = append([]researchbiz.ReasoningTreeNodeRecord(nil), tree.Nodes...)
	invalidTree.Nodes[0].Signals = append([]researchbiz.SignalRecord(nil), tree.Nodes[0].Signals...)
	invalidTree.Nodes[0].Signals[0].SignalRole = "invented"
	if validReasoningTreeDetail(detail, invalidTree, []string{"node:one"}) {
		t.Fatal("malformed persisted snapshot Signal enum was accepted")
	}
}

func TestPostgresSnapshotPublicationWorksOnCurrentSchema(t *testing.T) {
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	db := postgresfixture.OpenIsolated(t, "tw_research_snapshot", migrationDir, 0)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	evidenceStore, err := evidencedata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	evidenceUseCase, err := evidencebiz.NewUseCase(evidenceStore)
	if err != nil {
		t.Fatal(err)
	}
	publishedAt := time.Now().UTC().Add(-2 * time.Hour)
	raw, err := evidenceUseCase.PublishRawEvidence(ctx, evidencebiz.RawEvidence{
		PublicationKey: "research-snapshot-ledger", SourceID: "SRC_research_snapshot", SourceName: "Research Source",
		SourceLevel: evidencebiz.SourceLevelOfficial, SourceURL: "https://example.test/research", IsOriginal: true,
		RawText: "Research snapshot source.", PublishedAt: &publishedAt, CollectedAt: publishedAt.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := evidenceUseCase.PublishEvidence(ctx, raw.ID, []evidencebiz.Evidence{{
		Summary: "Research snapshot evidence.", Keywords: []string{"研究快照"},
		Semantic: evidencebiz.Semantic{
			Actors: []string{"Research Source"}, Action: "supports", Objects: []string{"Research snapshot"},
			Stage: evidencebiz.EvidenceStageOccurred, Modality: evidencebiz.EvidenceModalityFact,
			Time: evidencebiz.EvidenceTime{Precision: evidencebiz.EvidenceTimeUnknown}, Jurisdictions: []string{}, Metrics: []evidencebiz.EvidenceMetric{},
			Attribution: &evidencebiz.EvidenceAttribution{},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	eventStore, err := eventdata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	eventUseCase, err := eventbiz.NewUseCase(eventStore)
	if err != nil {
		t.Fatal(err)
	}
	event, err := eventUseCase.Create(ctx, eventbiz.CreateInput{
		Title: "Snapshot Event", Summary: "Snapshot Event summary.",
		Semantic: eventbiz.Semantic{
			Actors:        []string{"Research actor"},
			Action:        "publishes",
			Objects:       []string{"Research snapshot"},
			Stage:         eventbiz.EventStageOccurred,
			Modality:      eventbiz.ModalityFact,
			Time:          eventbiz.EventTime{OccurredAt: &publishedAt, Precision: eventbiz.TimePrecisionDay},
			Jurisdictions: []string{},
			Metrics:       []eventbiz.Metric{},
		},
		Evidence: []eventbiz.EvidenceLinkInput{{EvidenceID: evidence.IDs[0], ContributionWeight: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	graphStore, err := entitydata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := researchbiz.NewUseCase(store, store, graphStore, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	aggregate := integrationSnapshotAggregate(time.Now().UTC().Truncate(time.Second))
	aggregate.Theme.Events[0].EventID = event.Event.ID
	aggregate.Theme.Events[0].EvidenceIDs = []string{event.Evidence[0].ID}
	aggregate.ReasoningTrees[0].Events[0].EventID = event.Event.ID
	aggregate.ReasoningTrees[0].Events[0].EvidenceIDs = []string{event.Evidence[0].ID}
	first, err := useCase.PublishSnapshot(ctx, "integration-analyst", aggregate)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := useCase.PublishSnapshot(ctx, "integration-analyst", aggregate)
	if err != nil {
		t.Fatal(err)
	}
	if first.PublicationMode != researchbiz.SnapshotPublicationMode || !replayed.Replayed ||
		first.ThemeID != replayed.ThemeID {
		t.Fatalf("first=%#v replay=%#v", first, replayed)
	}
	detail, err := useCase.GetTheme(ctx, first.ThemeID, researchbiz.ResearchDetailRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if detail.PublicationMode != researchbiz.SnapshotPublicationMode ||
		detail.PublicationContractVersion != 3 || detail.ThemeKey != "theme:ledger" {
		t.Fatalf("snapshot detail = %#v", detail)
	}
	treeID := first.ReasoningTreeIDsByTreeKey["tree:ledger"]
	tree, err := useCase.GetReasoningTree(ctx, first.ThemeID, treeID)
	if err != nil {
		t.Fatal(err)
	}
	if tree.PublicationMode != researchbiz.SnapshotPublicationMode ||
		tree.ReasoningTree.TreeKey != "tree:ledger" ||
		tree.ReasoningTree.Nodes[0].Signals[0].SignalKey != "signal:ledger" {
		t.Fatalf("snapshot tree = %#v", tree)
	}
}

func integrationSnapshotAggregate(asOf time.Time) researchbiz.SnapshotAggregate {
	return researchbiz.SnapshotAggregate{
		PublicationMode:      researchbiz.SnapshotPublicationMode,
		AnalysisBatchID:      "snapshot-ledger",
		AnalysisAsOf:         asOf.Format(time.RFC3339),
		DiscoveryWindowStart: asOf.Add(-time.Hour).Format(time.RFC3339),
		DiscoveryWindowEnd:   asOf.Format(time.RFC3339),
		Theme: researchbiz.SnapshotTheme{
			ThemeKey: "theme:ledger", Title: "Ledger Theme", OneLineConclusion: "Snapshot remains publishable",
			ConclusionDirection: "positive", ImpactStrength: "medium", TransmissionStage: "validation",
			InvestmentGuidanceAction: "observe", InvestmentGuidanceSummary: "Observe the snapshot",
			TimeHorizonCategory: "medium_term",
			Impacts: []researchbiz.SnapshotImpact{{
				NodeKey: "node:ledger", DisplayName: "Ledger Node", RelationRole: "driver",
				ImpactDirection: "positive", DisplayOrder: 1,
			}},
			Events: []researchbiz.SnapshotEvent{{EventID: "EVT11111111-1111-4111-8111-111111111111", EvidenceRole: "driver"}},
		},
		ReasoningTrees: []researchbiz.SnapshotReasoningTree{{
			TreeKey: "tree:ledger", DisplayName: "Ledger Tree", Title: "Ledger Tree",
			DisplayOrder: 1, OneLineConclusion: "Snapshot reasoning", ImpactDirection: "positive",
			ImpactStrength: "medium", InvalidationConditions: []string{}, Checkpoints: []researchbiz.ReasonTreeCheckpoint{},
			Events: []researchbiz.SnapshotTreeEvent{{EventID: "EVT11111111-1111-4111-8111-111111111111", EvidenceRole: "driver", DisplayOrder: 1}},
			Nodes: []researchbiz.SnapshotNode{{
				NodeKey: "node:ledger", DisplayName: "Ledger Node", Position: 1,
				ImpactDirection: "positive", ImpactStrength: "medium",
				Signals: []researchbiz.SnapshotSignal{{
					SignalKey: "signal:ledger", DisplaySummary: "Snapshot signal", Role: "primary", DisplayOrder: 1,
				}},
			}},
		}},
	}
}

func stringPointer(value string) *string { return &value }

func openResearchTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, "tw_research_transaction", migrationDir, 0)
}
