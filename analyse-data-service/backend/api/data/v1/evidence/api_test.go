package evidence

import (
	"os"
	"strings"
	"testing"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
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
