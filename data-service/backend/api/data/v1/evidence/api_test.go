package evidence

import (
	"os"
	"strings"
	"testing"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	"gopkg.in/yaml.v3"
)

func TestEvidencePublicationProviderFixturesAreContractNeutralAndTwoPhase(t *testing.T) {
	rawPayload, err := os.ReadFile("testdata/raw-evidence-publication.json")
	if err != nil {
		t.Fatal(err)
	}
	evidencePayload, err := os.ReadFile("testdata/evidence-publication.json")
	if err != nil {
		t.Fatal(err)
	}
	for name, payload := range map[string][]byte{"raw": rawPayload, "evidence": evidencePayload} {
		lower := strings.ToLower(string(payload))
		for _, forbidden := range []string{"agentrun", "agentos", "group_id", "idempotency-key"} {
			if strings.Contains(lower, forbidden) {
				t.Fatalf("%s fixture couples the Data contract to %q", name, forbidden)
			}
		}
	}
	rawRequest, err := decodeRawEvidence(rawPayload)
	if err != nil {
		t.Fatalf("decode Raw Evidence fixture: %v", err)
	}
	evidenceRequest, err := decodeEvidence(evidencePayload)
	if err != nil {
		t.Fatalf("decode Evidence fixture: %v", err)
	}
	if rawRequest.RawEvidence.RawEvidenceID != evidenceRequest.RawEvidenceID {
		t.Fatalf("fixture parent identity mismatch: %q != %q", rawRequest.RawEvidence.RawEvidenceID, evidenceRequest.RawEvidenceID)
	}
	if len(rawRequest.RawEvidence.CategoryIDs) != 2 || rawRequest.RawEvidence.CategoryIDs[0] != "EVCed6e9380-8b20-53d5-b748-fa45c774fa67" || rawRequest.RawEvidence.CategoryIDs[1] != "EVC5b12ffce-178d-56ed-a54f-c01696c486f4" {
		t.Fatalf("fixture Raw Evidence categories = %#v", rawRequest.RawEvidence.CategoryIDs)
	}
	if len(evidenceRequest.Evidences) != 2 ||
		evidenceRequest.Evidences[0].ExpressionKey != evidenceRequest.Evidences[1].ExpressionKey {
		t.Fatalf("fixture must preserve two source Evidences sharing one expression_key: %#v", evidenceRequest.Evidences)
	}
}

func TestEvidencePublicationOpenAPIUsesInternalThreeSecondBudget(t *testing.T) {
	var document map[string]any
	if err := yaml.Unmarshal(v1.Document(), &document); err != nil {
		t.Fatal(err)
	}
	paths := document["paths"].(map[string]any)
	for _, path := range []string{
		v1.APIPrefix + "/raw-evidence-publications",
		v1.APIPrefix + "/evidence-publications",
	} {
		operation := paths[path].(map[string]any)["post"].(map[string]any)
		if got := operation["x-timeout-budget-ms"]; got != 3000 {
			t.Errorf("%s x-timeout-budget-ms = %#v, want 3000", path, got)
		}
	}
}

func TestEvidencePublicationOpenAPISuccessResultsContainOnlyFormalIdentities(t *testing.T) {
	var document map[string]any
	if err := yaml.Unmarshal(v1.Document(), &document); err != nil {
		t.Fatal(err)
	}
	components := document["components"].(map[string]any)["schemas"].(map[string]any)
	for name, expected := range map[string][]string{
		"RawEvidencePublicationResult": {"raw_evidence_id"},
		"EvidencePublicationResult":    {"raw_evidence_id", "evidence_ids"},
	} {
		schema := components[name].(map[string]any)
		properties := schema["properties"].(map[string]any)
		if len(properties) != len(expected) {
			t.Fatalf("%s properties = %#v, want only %#v", name, properties, expected)
		}
		for _, field := range expected {
			if _, ok := properties[field]; !ok {
				t.Fatalf("%s is missing %q", name, field)
			}
		}
		if schema["additionalProperties"] != false {
			t.Fatalf("%s must reject additional response properties", name)
		}
	}
}

func TestRawEvidenceCategoryOpenAPIContract(t *testing.T) {
	var document map[string]any
	if err := yaml.Unmarshal(v1.Document(), &document); err != nil {
		t.Fatal(err)
	}
	components := document["components"].(map[string]any)["schemas"].(map[string]any)
	raw := components["RawEvidence"].(map[string]any)
	categoryIDs := raw["properties"].(map[string]any)["category_ids"].(map[string]any)
	if categoryIDs["type"] != "array" || categoryIDs["uniqueItems"] != true {
		t.Fatalf("RawEvidence category_ids = %#v", categoryIDs)
	}
	category := components["EvidenceCategory"].(map[string]any)
	properties := category["properties"].(map[string]any)
	for _, field := range []string{"id", "code", "name", "description"} {
		if _, exists := properties[field]; !exists {
			t.Fatalf("EvidenceCategory is missing %q", field)
		}
	}
	if len(properties) != 4 || category["additionalProperties"] != false {
		t.Fatalf("EvidenceCategory contract = %#v", category)
	}
	read := components["RawEvidenceRead"].(map[string]any)
	categories := read["properties"].(map[string]any)["categories"].(map[string]any)
	items := categories["items"].(map[string]any)
	if items["$ref"] != "#/components/schemas/EvidenceCategory" {
		t.Fatalf("RawEvidenceRead categories = %#v", categories)
	}
}
