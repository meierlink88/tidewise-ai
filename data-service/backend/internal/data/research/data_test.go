package research

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	researchbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/research"
	entitydata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

const integrationEventID = "EVT11111111-1111-4111-8111-111111111111"

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

func TestPostgresSnapshotAndAtomicEvidenceWorkOnCurrentSchema(t *testing.T) {
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	db := postgresfixture.OpenIsolated(t, "tw_research_snapshot", migrationDir, 0)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, err := db.ExecContext(ctx, `INSERT INTO events (
    id, title, summary, first_seen_at, event_status, fact_status, dedupe_key, fact_payload
) VALUES ($1, 'Snapshot Event', 'Snapshot Event summary', now(), 'confirmed', 'verified',
    'research-snapshot-ledger', '{}'::jsonb)`, integrationEventID); err != nil {
		t.Fatal(err)
	}
	const rawEvidenceID = "RAW11111111-1111-4111-8111-111111111111"
	const atomicEvidenceID = "EVD11111111-1111-4111-8111-111111111111"
	const atomicSemantic = `{"who":"Data","what":"Atomic Evidence semantic survives","when":null,"where":null,"why":null,"how":null}`
	if _, err := db.ExecContext(ctx, `INSERT INTO raw_evidences (
    id, source_id, source_name, source_level, source_url, is_original, raw_text, collected_at, keywords
) VALUES ($1, 'source:ledger', 'Ledger Source', 'L1_OFFICIAL', 'https://example.test/ledger',
    true, 'Atomic Evidence semantic survives', now(), '{}')`, rawEvidenceID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO evidences (
    id, raw_evidence_id, is_split, summary, semantic
) VALUES ($1, $2, false, 'Atomic Evidence semantic survives', $3::jsonb)`,
		atomicEvidenceID, rawEvidenceID, atomicSemantic); err != nil {
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
	var semanticPreserved bool
	if err := db.QueryRowContext(ctx, `SELECT semantic = $2::jsonb FROM evidences WHERE id = $1`,
		atomicEvidenceID, atomicSemantic).Scan(&semanticPreserved); err != nil {
		t.Fatal(err)
	}
	if !semanticPreserved {
		t.Fatal("Atomic Evidence semantic did not round-trip on the current schema")
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
			Events: []researchbiz.SnapshotEvent{{EventID: integrationEventID, EvidenceRole: "driver"}},
		},
		ReasoningTrees: []researchbiz.SnapshotReasoningTree{{
			TreeKey: "tree:ledger", DisplayName: "Ledger Tree", Title: "Ledger Tree",
			DisplayOrder: 1, OneLineConclusion: "Snapshot reasoning", ImpactDirection: "positive",
			ImpactStrength: "medium", InvalidationConditions: []string{}, Checkpoints: []researchbiz.ReasonTreeCheckpoint{},
			Events: []researchbiz.SnapshotTreeEvent{{EventID: integrationEventID, EvidenceRole: "driver", DisplayOrder: 1}},
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
