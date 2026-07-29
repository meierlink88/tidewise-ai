package v1

import (
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

func TestEventPublicationBindingAllowsArbitraryFactPayloadButRejectsUnknownWireFields(t *testing.T) {
	valid := `{"package_id":"package","provenance":{"extractor_execution_id":"execution","extractor_agent_version":"v1","collector_executions":[]},"raw_documents":[],"events":[{"dedupe_key":"event","title":"title","factual_summary":"summary","fact_payload":{"nested":{"value":[1,true,null]}},"evidence":[],"tags":[],"review":{"review_id":"review","evidence_grade":"A","reasons":[]}}]}`
	if _, err := decodeEventPublication([]byte(valid)); err != nil {
		t.Fatalf("decode valid publication: %v", err)
	}

	unknown := `{"package_id":"package","unexpected":true}`
	if _, err := decodeEventPublication([]byte(unknown)); err == nil {
		t.Fatal("unknown top-level field was accepted")
	}
}
