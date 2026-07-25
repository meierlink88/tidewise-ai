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
			payload: `{"analysis_batch_id":"batch","window_start":"start","window_end":"end","themes":[{"theme_key":"theme-1","theme_key":"theme-2"}]}`,
			path:    "themes[0].theme_key",
		},
		{
			name:    "unknown nested field",
			payload: `{"analysis_batch_id":"batch","window_start":"start","window_end":"end","themes":[{"theme_key":"theme-1","unexpected":true}]}`,
			path:    "themes[0].unexpected",
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

func TestResearchAnchorBindingEnforcesRequiredTypedWireContract(t *testing.T) {
	for _, test := range []struct {
		name    string
		payload string
		path    string
	}{
		{
			name:    "missing required field",
			payload: `{"theme_id":"11111111-1111-4111-8111-111111111111","anchors":[{"center_chain_node_id":"22222222-2222-4222-8222-222222222222"}]}`,
			path:    "anchors[0].one_line_conclusion",
		},
		{
			name:    "wrong scalar type",
			payload: `{"theme_id":42,"anchors":[]}`,
			path:    "theme_id",
		},
		{
			name:    "invalid uuid",
			payload: `{"theme_id":"not-a-uuid","anchors":[]}`,
			path:    "theme_id",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := decodeResearchAnchorImport([]byte(test.payload))
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
