package eventsemantic

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

func TestEventSemanticProviderContractAcceptsV3ConsumerFixtures(t *testing.T) {
	loader := openapi3.NewLoader()
	document, err := loader.LoadFromData(v1.Document())
	if err != nil {
		t.Fatal(err)
	}
	if err := document.Validate(context.Background()); err != nil {
		t.Fatal(err)
	}
	fixtures := map[string]string{
		"supply-eligible-events.json":     "EligibleEventSemanticEventsEnvelope",
		"supply-context-lease.json":       "EventSemanticContextLeaseEnvelope",
		"supply-context.json":             "EventSemanticContextEnvelope",
		"supply-submission-accepted.json": "EventSemanticSubmissionEnvelope",
		"supply-review-accepted.json":     "EventSemanticSubmissionEnvelope",
		"supply-event-semantics.json":     "EventSemanticsEnvelope",
	}
	for name, schemaName := range fixtures {
		payload, err := os.ReadFile(filepath.Join(
			"..", "..", "..", "..", "..", "..", "contracts", "event-semantics", "v3", name,
		))
		if err != nil {
			t.Fatal(err)
		}
		var value any
		if err := json.Unmarshal(payload, &value); err != nil {
			t.Fatalf("decode %s: %v", name, err)
		}
		if name == "supply-context.json" && len(payload) >= 100_000 {
			t.Fatalf("compact context fixture = %d bytes, want < 100000", len(payload))
		}
		schema := document.Components.Schemas[schemaName]
		if schema == nil || schema.Value == nil {
			t.Fatalf("schema %s is missing", schemaName)
		}
		if err := schema.Value.VisitJSON(value); err != nil {
			t.Fatalf("%s violates %s: %v", name, schemaName, err)
		}
	}
}

func TestEventSemanticReadContractKeepsV2HistoricalFixtureReadable(t *testing.T) {
	loader := openapi3.NewLoader()
	document, err := loader.LoadFromData(v1.Document())
	if err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(filepath.Join(
		"..", "..", "..", "..", "..", "..", "contracts", "event-semantics", "v2", "supply-event-semantics.json",
	))
	if err != nil {
		t.Fatal(err)
	}
	var value any
	if err := json.Unmarshal(payload, &value); err != nil {
		t.Fatal(err)
	}
	schema := document.Components.Schemas["EventSemanticsEnvelope"]
	if schema == nil || schema.Value == nil {
		t.Fatal("EventSemanticsEnvelope is missing")
	}
	if err := schema.Value.VisitJSON(value); err != nil {
		t.Fatalf("V2 historical read fixture violates EventSemanticsEnvelope: %v", err)
	}
}

func TestEventSemanticV3SubmissionRequiresProjectedEntityType(t *testing.T) {
	loader := openapi3.NewLoader()
	document, err := loader.LoadFromData(v1.Document())
	if err != nil {
		t.Fatal(err)
	}
	schema := document.Components.Schemas["EventSemanticV3EntityLinkCandidate"]
	if schema == nil || schema.Value == nil {
		t.Fatal("EventSemanticV3EntityLinkCandidate is missing")
	}
	valid := map[string]any{
		"candidate_key": "company", "mention": "某公司",
		"entity_id": "ENT33333333-3333-4333-8333-333333333333", "projected_entity_type": "company",
		"entity_role": "event_subject", "evidence_ids": []any{"22222222-2222-4222-8222-222222222222"},
		"resolution_method": "qdrant_exact",
	}
	if err := schema.Value.VisitJSON(valid); err != nil {
		t.Fatalf("valid V3 candidate: %v", err)
	}
	delete(valid, "projected_entity_type")
	if err := schema.Value.VisitJSON(valid); err == nil {
		t.Fatal("V3 candidate without projected_entity_type passed")
	}
}
