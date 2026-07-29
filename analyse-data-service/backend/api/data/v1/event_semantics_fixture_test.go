package v1

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestEventSemanticProviderContractAcceptsFrozenConsumerFixtures(t *testing.T) {
	loader := openapi3.NewLoader()
	document, err := loader.LoadFromData(Document())
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
		"supply-resolution.json":          "EventSemanticEntityResolutionEnvelope",
		"supply-targets.json":             "EventSemanticDirectTargetSearchEnvelope",
		"supply-submission-accepted.json": "EventSemanticSubmissionEnvelope",
		"supply-review-accepted.json":     "EventSemanticSubmissionEnvelope",
		"supply-event-semantics.json":     "EventSemanticsEnvelope",
	}
	for name, schemaName := range fixtures {
		payload, err := os.ReadFile(filepath.Join(
			"..", "..", "..", "..", "..", "contracts", "event-semantics", "v1", name,
		))
		if err != nil {
			t.Fatal(err)
		}
		var value any
		if err := json.Unmarshal(payload, &value); err != nil {
			t.Fatalf("decode %s: %v", name, err)
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
