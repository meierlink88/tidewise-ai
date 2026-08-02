package dataclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic"
)

func TestEventSemanticClientConsumesV3ContextWithoutSearchContracts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/data/v1/event-semantics/context-leases/11111111-1111-4111-8111-111111111111/context" {
			t.Fatalf("unexpected Data path %s", request.URL.Path)
		}
		requestID := request.Header.Get("X-Request-ID")
		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("X-Request-ID", requestID)
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"request_id": requestID,
			"result": map[string]any{
				"context_lease_id":   "11111111-1111-4111-8111-111111111111",
				"agent_execution_id": "execution", "worker_id": "worker",
				"lease_expires_at":          "2026-08-01T10:00:00Z",
				"manifest_contract_version": "event-semantic-context-manifest.v3",
				"context_fingerprint":       strings.Repeat("a", 64), "event_fingerprint": strings.Repeat("b", 64), "evidence_fingerprint": strings.Repeat("c", 64),
				"ontology_version":          "event-semantics.objective-v3@1",
				"acceptance_policy_version": "event-semantics.objective-v2@1",
				"event":                     map[string]any{"id": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "title": "title", "summary": "summary", "occurred_at": nil, "event_status": "confirmed", "fact_status": "verified"},
				"evidence": []any{map[string]any{
					"evidence_id": "22222222-2222-4222-8222-222222222222", "evidence_hash": strings.Repeat("d", 64),
					"evidence_statement": "title", "source_level": "primary", "relation": "supports", "supports_fields": []string{"title"},
					"raw_document_id": "33333333-3333-4333-8333-333333333333", "source_name": "fixture", "source_type": "news", "title": "title",
					"first_seen_at": "2026-08-01T09:00:00Z", "knowledge_available_at": "2026-08-01T09:00:00Z", "accepted_at": "2026-08-01T09:00:00Z", "statement_source": "",
				}},
				"entity_type_definitions": []any{map[string]any{
					"type_key": "company", "version": 1, "name_zh": "企业", "name_en": "Company",
					"business_definition": "依法设立的企业主体", "inclusion_criteria": []string{"公司"},
					"exclusion_criteria": []string{"产品"}, "event_link_allowed": true,
					"signal_subject_allowed": true, "allowed_event_roles": []string{"actor"}, "status": "active",
				}},
				"variable_definitions": []any{map[string]any{"key": "revenue", "version": 1, "name_zh": "收入", "name_en": "Revenue", "domain": "finance", "business_definition": "正式收入", "value_type": "narrative", "status": "active", "allowed_directions": []string{"increase"}, "allowed_units": []string{}, "applicable_entity_types": []string{"company"}}},
				"assertion_modalities": []string{"actual"},
				"measurement_contract": map[string]any{"representation": "evidence_grounded_narrative", "max_items_per_signal": 8, "max_text_characters": 2000, "requires_evidence_ids": true, "numeric_validation": false},
			},
		})
	}))
	defer server.Close()
	client, err := New(Config{BaseURL: server.URL, ServiceToken: "token", Timeout: time.Second, MaxResponseBytes: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	contextValue, err := client.Context(context.Background(), "11111111-1111-4111-8111-111111111111")
	if err != nil {
		t.Fatal(err)
	}
	if contextValue.ManifestContractVersion != "event-semantic-context-manifest.v3" || contextValue.MeasurementContract.NumericValidation {
		t.Fatalf("context = %#v", contextValue)
	}
	invalid := contextValue
	invalid.ContextFingerprint = "not-a-sha256"
	if validSemanticContext(invalid, contextValue.ContextLeaseID) {
		t.Fatal("consumer accepted invalid Context fingerprint")
	}
	invalid = contextValue
	invalid.Evidence = nil
	if validSemanticContext(invalid, contextValue.ContextLeaseID) {
		t.Fatal("consumer accepted Context without Evidence")
	}
	var _ eventsemantic.DataClient = client
}

func TestEventSemanticSubmissionWireContainsNoDirectImpact(t *testing.T) {
	payload, err := json.Marshal(eventsemantic.SubmissionRequest{
		AgentKey: eventsemantic.AgentKey, AgentVersion: eventsemantic.AgentVersion,
		EntityLinks: []eventsemantic.EntityLinkCandidate{{
			CandidateKey: "company", EntityID: "33333333-3333-4333-8333-333333333333",
			ProjectedEntityType: "company",
		}}, VariableSignals: []eventsemantic.VariableSignalCandidate{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) == "" || json.Valid(payload) == false {
		t.Fatalf("payload = %s", payload)
	}
	var value map[string]any
	_ = json.Unmarshal(payload, &value)
	if _, exists := value["direct_impacts"]; exists {
		t.Fatalf("V3 payload unexpectedly contains direct_impacts: %s", payload)
	}
	links := value["entity_links"].([]any)
	if links[0].(map[string]any)["projected_entity_type"] != "company" {
		t.Fatalf("V3 payload omitted projected Entity Type: %s", payload)
	}
}

func TestHistoricalAuditWorkPackageMayLackResolvedEntitiesButResumableReviewMayNot(t *testing.T) {
	work := &eventsemantic.ReviewerWorkPackage{EntityLinks: []eventsemantic.EntityLinkCandidate{{
		CandidateKey: "company", Mention: "公司", EntityID: "33333333-3333-4333-8333-333333333333",
		EvidenceIDs: []string{"22222222-2222-4222-8222-222222222222"},
	}}}
	base := eventsemantic.SubmissionResult{
		SubmissionID:         "11111111-1111-4111-8111-111111111111",
		EventID:              "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		Status:               "accepted",
		CanonicalPayloadHash: strings.Repeat("a", 64),
		AuditWorkPackage:     work,
	}
	if !validSemanticSubmission(base) {
		t.Fatal("historical terminal audit package without resolved_entities was rejected")
	}
	base.AuditWorkPackage = nil
	base.ReviewerWorkPackage = work
	base.Status = "pending_review"
	if validSemanticSubmission(base) {
		t.Fatal("resumable reviewer work package without resolved_entities was accepted")
	}
}
