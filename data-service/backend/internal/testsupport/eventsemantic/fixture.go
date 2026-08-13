package eventsemanticfixture

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	eventbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/eventsemantic"
)

const (
	RawDocumentID     = "10000000-0000-4000-8000-000000000001"
	EventID           = "10000000-0000-4000-8000-000000000002"
	EvidenceID        = "10000000-0000-4000-8000-000000000003"
	CompanyID         = "10000000-0000-4000-8000-000000000004"
	ProductID         = "10000000-0000-4000-8000-000000000005"
	RelationID        = "10000000-0000-4000-8000-000000000006"
	EvidenceStatement = "Integration Wafer Fab production fell 10%"
)

func SeedScenario(t *testing.T, db *sql.DB, currentEvidenceContract bool) {
	t.Helper()
	evidenceHash := fmt.Sprintf("%x", sha256.Sum256([]byte(EvidenceStatement)))
	if _, err := db.Exec(`
INSERT INTO raw_documents (
  id, ingest_channel, source_type, source_name, source_url, title, content_text,
  raw_mime_type, language, published_at, collected_at, content_hash, ingest_status
) VALUES (
  $1, 'integration', 'news', 'Integration Primary Source', 'https://example.test/wafer',
  'Wafer production update', 'Integration source content', 'text/plain', 'en',
  '2026-07-28T08:00:00Z', '2026-07-28T08:01:00Z', $2, 'collected'
)`, RawDocumentID, strings.Repeat("1", 64)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO events (
  id, title, summary, event_time, first_seen_at, knowable_at,
  event_status, fact_status, dedupe_key, fact_payload
) VALUES (
  $1, 'Integration Wafer Fab production fell 10%',
  'Integration Wafer Fab confirmed a 10% production decline.',
  '2026-07-28T08:00:00Z', '2026-07-28T08:01:00Z', '2026-07-28T08:01:00Z',
  'confirmed', 'verified', 'event:semantic:integration', '{"production_change_percent":-10}'::jsonb
)`, EventID); err != nil {
		t.Fatal(err)
	}
	if currentEvidenceContract {
		if _, err := db.Exec(`
INSERT INTO event_sources (
  id, event_id, raw_document_id, source_level, evidence_statement, evidence_hash,
  evidence_relation, supports_fields, contract_version
) VALUES (
  $1, $2, $3, 'primary', 'Integration Wafer Fab production fell 10%', $4,
  'supports', ARRAY['title','factual_summary','occurred_at','fact_payload'], 3
)`, EvidenceID, EventID, RawDocumentID, evidenceHash); err != nil {
			t.Fatal(err)
		}
	} else {
		if _, err := db.Exec(`
INSERT INTO event_sources (
  id, event_id, raw_document_id, source_level, evidence_excerpt, evidence_hash,
  evidence_relation, supports_fields, is_primary, contract_version
) VALUES (
  $1, $2, $3, 'primary', 'Integration Wafer Fab production fell 10%', $4,
  'supports', ARRAY['title','factual_summary','occurred_at','fact_payload'], true, 2
)`, EvidenceID, EventID, RawDocumentID, evidenceHash); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(
			`UPDATE events SET primary_source_id = $2 WHERE id = $1`,
			EventID,
			EvidenceID,
		); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`
INSERT INTO entity_nodes (
  id, entity_key, entity_type, layer_code, name, canonical_name, aliases, status
) VALUES
  ($1, 'company:integration-wafer-fab', 'company', 'company',
   'Integration Wafer Fab', 'Integration Wafer Fab', ARRAY['IWF'], 'active'),
  ($2, 'product:integration-wafer', 'product', 'product',
   'Integration Wafer', 'Integration Wafer', ARRAY['8-inch wafer'], 'active')
`, CompanyID, ProductID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO entity_edges (
  id, from_entity_id, to_entity_id, relation_type, evidence_note, status
) VALUES ($1, $2, $3, 'produces', 'Integration evidence', 'active')
`, RelationID, CompanyID, ProductID); err != nil {
		t.Fatal(err)
	}
}

func Submission(leaseID, executionID, supersedesSubmissionID string) eventbiz.Submission {
	return eventbiz.Submission{
		ContextLeaseID: leaseID, EventID: EventID, AgentExecutionID: executionID,
		AgentKey: "event-semantic-enricher", AgentVersion: "event-semantic-enricher.v3",
		SupersedesSubmissionID: supersedesSubmissionID,
		GeneratorPromptHash:    strings.Repeat("a", 64), GeneratorModel: "fixture-generator",
		ReviewerPromptHash: strings.Repeat("b", 64), ReviewerModel: "fixture-reviewer",
		AdjudicatorPromptHash: strings.Repeat("b", 64), AdjudicatorModel: "fixture-reviewer",
		OntologyVersion: "event-semantics.objective-v3@1", AcceptancePolicyVersion: "event-semantics.objective-v2@1",
		EntityLinks: []eventbiz.EntityLinkCandidate{{
			Key: "company", Mention: "Integration Wafer Fab", EntityID: CompanyID,
			ProjectedEntityType: "company",
			EntityRole:          "event_subject", EvidenceIDs: []string{EvidenceID},
			ResolutionMethod: "qdrant_exact", ResolutionConfidence: "1.00000",
		}},
		VariableSignals: []eventbiz.VariableSignalCandidate{{
			Key: "production", SubjectLinkKey: "company",
			VariableKey: "production_volume", VariableVersion: 1,
			Direction: "decrease", AssertionModality: "actual",
			EvidenceIDs: []string{EvidenceID},
			Measurements: []eventbiz.MeasurementValue{{
				Text: "production fell 10%", EvidenceIDs: []string{EvidenceID},
			}},
			ExtractionConfidence: "0.98000",
		}},
	}
}

func ReviewItems(decision eventbiz.ReviewDecision) []eventbiz.ReviewItem {
	return []eventbiz.ReviewItem{
		{CandidateType: "entity_link", CandidateKey: "company", Decision: decision,
			ReasonCodes: []string{"fixture_review"}, EvidenceIDs: []string{EvidenceID}},
		{CandidateType: "variable_signal", CandidateKey: "production", Decision: decision,
			ReasonCodes: []string{"fixture_review"}, EvidenceIDs: []string{EvidenceID}},
	}
}
