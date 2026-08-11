package v1

import (
	"os"
	"strings"
	"testing"
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
	rawRequest, err := decodeRawEvidencePublication(rawPayload)
	if err != nil {
		t.Fatalf("decode Raw Evidence fixture: %v", err)
	}
	evidenceRequest, err := decodeEvidencePublication(evidencePayload)
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
