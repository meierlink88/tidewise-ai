package v1

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResearchThemeBindingRejectsDuplicateAndUnknownFieldsWithPath(t *testing.T) {
	for _, test := range []struct {
		name    string
		payload string
		path    string
	}{
		{
			name:    "duplicate nested field",
			payload: `{"analysis_batch_id":"batch","analysis_as_of":"as-of","discovery_window_start":"start","discovery_window_end":"end","theme":{"theme_key":"theme-1","theme_key":"theme-2"},"reasoning_trees":[]}`,
			path:    "theme.theme_key",
		},
		{
			name:    "unknown nested field",
			payload: `{"analysis_batch_id":"batch","analysis_as_of":"as-of","discovery_window_start":"start","discovery_window_end":"end","theme":{"theme_key":"theme-1","unexpected":true},"reasoning_trees":[]}`,
			path:    "theme.unexpected",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := decodeResearchThemeImport([]byte(test.payload))
			publicError, ok := err.(*PublicError)
			if !ok {
				t.Fatalf("error = %T %v, want *PublicError", err, err)
			}
			details, ok := publicError.Details.(map[string]any)
			if !ok || details["path"] != test.path {
				t.Fatalf("details = %#v, want path %q", publicError.Details, test.path)
			}
		})
	}
}

func TestResearchThemeBindingAcceptsPreparedUATAnalystSnapshotFixture(t *testing.T) {
	payload, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "testdata", "research-theme-analyst-snapshot-v3", "01-uat-at01-prepared-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	request, err := decodeResearchThemeImport(payload)
	if err != nil || request.Snapshot == nil || len(request.Snapshot.ReasoningTrees) != 2 {
		t.Fatalf("decode prepared UAT fixture = %#v, %v", request, err)
	}
}

func TestResearchThemeBindingAcceptsIsolatedAnalystSnapshotAndRejectsFormalIDs(t *testing.T) {
	request := ResearchThemeSnapshotImportRequest{
		PublicationMode: "analyst_snapshot", AnalysisBatchID: "batch-snapshot",
		AnalysisAsOf: "2026-08-03T11:00:00Z", DiscoveryWindowStart: "2026-08-03T03:00:00Z",
		DiscoveryWindowEnd: "2026-08-03T07:00:00Z",
		Theme: ResearchThemeSnapshotItem{
			ThemeKey: "theme:snapshot", Title: "Theme", OneLineConclusion: "Conclusion",
			ConclusionDirection: "uncertain", ImpactStrength: "unknown", TransmissionStage: "validation",
			InvestmentGuidanceAction: "observe", InvestmentGuidanceSummary: "Observe",
			TimeHorizonCategory: "medium_term",
			Impacts:             []ResearchThemeSnapshotImpact{{NodeKey: "node:a", DisplayName: "Focus name", RelationRole: "driver", ImpactDirection: "uncertain", DisplayOrder: 1}},
			Events:              []ResearchThemeSnapshotEvent{{EventID: "11111111-1111-4111-8111-111111111111", EvidenceRole: "driver"}},
		},
		ReasoningTrees: []ResearchReasoningTreeSnapshotImportItem{{
			TreeKey: "tree:a", DisplayName: "Analysis path", Title: "Tree", DisplayOrder: 1,
			OneLineConclusion: "Tree conclusion", ImpactDirection: "uncertain", ImpactStrength: "unknown",
			InvalidationConditions: []string{}, Checkpoints: []ResearchReasoningTreeImportCheckpoint{},
			Events: []ResearchReasoningTreeSnapshotEvent{{EventID: "11111111-1111-4111-8111-111111111111", EvidenceRole: "driver", DisplayOrder: 1}},
			Nodes: []ResearchReasoningTreeSnapshotNode{{
				NodeKey: "node:a", DisplayName: "Detailed node name", Position: 1,
				ImpactDirection: "uncertain", ImpactStrength: "unknown",
				Signals: []ResearchReasoningTreeSnapshotSignal{{SignalKey: "signal:a", DisplaySummary: "完成流片", Role: "primary", DisplayOrder: 1}},
			}},
		}},
	}
	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeResearchThemeImport(payload)
	if err != nil || decoded.Snapshot == nil {
		t.Fatalf("decode analyst_snapshot = %#v, %v", decoded, err)
	}

	formalID := `,"chain_node_entity_id":"33333333-3333-4333-8333-333333333333"`
	mixed := strings.Replace(string(payload), `"node_key":"node:a"`, `"node_key":"node:a"`+formalID, 1)
	if _, err := decodeResearchThemeImport([]byte(mixed)); err == nil {
		t.Fatal("analyst_snapshot carrying formal ontology ID was accepted")
	}
}

func TestResearchThemeBindingRequiresAtomicTreeAndLineageShape(t *testing.T) {
	payload := `{"analysis_batch_id":"batch","analysis_as_of":"2026-07-29T00:00:00Z","discovery_window_start":"2026-07-28T00:00:00Z","discovery_window_end":"2026-07-29T00:00:00Z","theme":{},"reasoning_trees":[{"industry_chain_entity_id":"22222222-2222-4222-8222-222222222222"}]}`
	_, err := decodeResearchThemeImport([]byte(payload))
	publicError, ok := err.(*PublicError)
	if !ok {
		t.Fatalf("error = %T %v, want *PublicError", err, err)
	}
	details, ok := publicError.Details.(map[string]any)
	if !ok || details["path"] != "reasoning_trees[0].title" {
		t.Fatalf("details = %#v", publicError.Details)
	}
}

func TestResearchThemeBindingAcceptsAtomicAggregateShape(t *testing.T) {
	payload := `{
		"analysis_batch_id":"batch","analysis_as_of":"2026-07-02T00:00:00Z",
		"discovery_window_start":"2026-07-01T00:00:00Z","discovery_window_end":"2026-07-02T00:00:00Z",
		"theme":{},"reasoning_trees":[{
			"industry_chain_entity_id":"22222222-2222-4222-8222-222222222222",
			"title":"tree","display_order":1,"one_line_conclusion":"conclusion",
			"fact_summary":null,"transmission_summary":null,"impact_direction":"positive",
			"impact_strength":"medium","impact_summary":null,"conclusion_boundary_summary":null,
			"support_summary":null,"counter_summary":null,"invalidation_conditions":[],
			"checkpoints":[],"events":[],"nodes":[{
				"position":1,"chain_node_entity_id":"33333333-3333-4333-8333-333333333333",
				"state_summary":null,"impact_direction":"positive","impact_strength":"medium",
				"impact_summary":null,"reasoning_basis_summary":null,"evidence_gap_summary":null,
				"incoming_industry_chain_graph_edge_id":null,"incoming_transmission_title":null,
				"incoming_transmission_mechanism":null,"incoming_condition_summary":null,
				"incoming_lineage":null,"signals":[]
			}]
		}]
	}`
	if _, err := decodeResearchThemeImport([]byte(payload)); err != nil {
		t.Fatalf("decode atomic aggregate: %#v", err)
	}
}
