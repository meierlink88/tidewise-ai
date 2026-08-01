package service

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventsemantics"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/postgres"
)

const (
	semanticRawDocumentID = "10000000-0000-4000-8000-000000000001"
	semanticEventID       = "10000000-0000-4000-8000-000000000002"
	semanticEvidenceID    = "10000000-0000-4000-8000-000000000003"
	semanticCompanyID     = "10000000-0000-4000-8000-000000000004"
	semanticProductID     = "10000000-0000-4000-8000-000000000005"
	semanticRelationID    = "10000000-0000-4000-8000-000000000006"
)

func TestEventSemanticV2MigrationBackfillsDeprecatedEntityTypeRoles(t *testing.T) {
	db := openEventPublicationTestDatabaseAt(t, 35)
	if _, err := db.Exec(`
		INSERT INTO entity_type_definitions(
			type_key, version, signal_subject_allowed, direct_target_mode, status
		) VALUES ('deprecated_fixture', 1, false, 'deny', 'deprecated')
	`); err != nil {
		t.Fatal(err)
	}
	applyEventPublicationMigration(t, db, 36)
	var roleCount int
	if err := db.QueryRow(`
		SELECT cardinality(allowed_event_roles) FROM entity_type_definitions
		WHERE type_key = 'deprecated_fixture' AND version = 1
	`).Scan(&roleCount); err != nil {
		t.Fatal(err)
	}
	if roleCount == 0 {
		t.Fatal("deprecated Entity Type received no Event roles during V2 migration")
	}
}

func TestEventSemanticV2MigrationBackfillsAndKeepsLegacyMeasurementWritesSafe(t *testing.T) {
	db := openEventPublicationTestDatabaseAt(t, 35)
	seedEventSemanticScenario(t, db)
	const (
		leaseID       = "40000000-0000-4000-8000-000000000001"
		submissionID  = "40000000-0000-4000-8000-000000000002"
		linkID        = "40000000-0000-4000-8000-000000000003"
		signalID      = "40000000-0000-4000-8000-000000000004"
		historicalID  = "40000000-0000-4000-8000-000000000005"
		legacyWriteID = "40000000-0000-4000-8000-000000000006"
	)
	if _, err := db.Exec(`
		INSERT INTO event_semantic_context_leases(
			id,event_id,agent_execution_id,worker_id,status,lease_expires_at,context_snapshot,consumed_at
		) VALUES ($1,$2,'migration-compat','migration-test','consumed',now() + interval '1 hour','{}',now())
	`, leaseID, semanticEventID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO event_semantic_submissions(
			id,context_lease_id,event_id,agent_execution_id,agent_key,agent_version,
			generator_prompt_hash,generator_model,reviewer_prompt_hash,reviewer_model,
			ontology_version,acceptance_policy_key,acceptance_policy_version,
			canonical_payload_hash,status,candidate_counts,decision_summary,finalized_at
		) VALUES (
			$1,$2,$3,'migration-compat','event-semantic-enricher','legacy-version',
			$4,'legacy-generator',$5,'legacy-reviewer','legacy-ontology',
			'event-semantics.phase-one',1,$6,'accepted','{}','{}',now()
		)
	`, submissionID, leaseID, semanticEventID, strings.Repeat("a", 64), strings.Repeat("b", 64), strings.Repeat("c", 64)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO event_entity_links(
			id,event_id,entity_id,entity_role,assign_source,review_status,evidence_note,
			semantic_submission_id,candidate_key,resolved_mention,resolution_method,
			resolution_confidence,evidence_ids,provenance
		) VALUES ($1,$2,$3,'event_subject','ai','accepted','',$4,'company','Integration Wafer Fab',
			'qdrant_exact',1.0,ARRAY[$5::uuid],'semantic')
	`, linkID, semanticEventID, semanticCompanyID, submissionID, semanticEvidenceID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO variable_signals(
			id,semantic_submission_id,candidate_key,source_event_id,subject_event_entity_link_id,
			variable_key,variable_version,direction,assertion_modality,evidence_ids,review_status
		) VALUES ($1,$2,'production',$3,$4,'production_volume',1,'decrease','actual',ARRAY[$5::uuid],'accepted')
	`, signalID, submissionID, semanticEventID, linkID, semanticEvidenceID); err != nil {
		t.Fatal(err)
	}
	insertLegacyMeasurement := func(id string) {
		t.Helper()
		if _, err := db.Exec(`
			INSERT INTO variable_signal_measurements(
				id,variable_signal_id,measurement_role,value_shape,raw_value,raw_unit,
				canonical_value,canonical_unit,raw_text,is_approximate,evidence_id
			) VALUES ($1,$2,'relative_change','exact',10,'%',10,'percent','production fell 10%',false,$3)
		`, id, signalID, semanticEvidenceID); err != nil {
			t.Fatal(err)
		}
	}
	insertLegacyMeasurement(historicalID)

	applyEventPublicationMigration(t, db, 36)
	assertMeasurementEvidenceIDs(t, db, historicalID, semanticEvidenceID)
	insertLegacyMeasurement(legacyWriteID)
	assertMeasurementEvidenceIDs(t, db, legacyWriteID, semanticEvidenceID)
}

func assertMeasurementEvidenceIDs(t *testing.T, db *sql.DB, measurementID, evidenceID string) {
	t.Helper()
	var evidenceIDs string
	if err := db.QueryRow(`
		SELECT evidence_ids::text FROM variable_signal_measurements WHERE id = $1
	`, measurementID).Scan(&evidenceIDs); err != nil {
		t.Fatal(err)
	}
	if evidenceIDs != "{"+evidenceID+"}" {
		t.Fatalf("measurement %s evidence_ids = %q", measurementID, evidenceIDs)
	}
}

func TestPostgresEventSemanticV2PersistsNarrativeMeasurementWithoutDirectImpact(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	seedEventSemanticScenario(t, db)
	service := eventsemantics.NewService(postgres.NewEventSemanticsStore(db))
	lease, err := service.CreateContextLease(context.Background(), eventsemantics.ContextLeaseRequest{
		EventID: semanticEventID, AgentExecutionID: "semantic-v2-persistence",
		WorkerID: "semantic-v2-fixture", Lease: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	submission, err := service.CreateSubmission(context.Background(), semanticSubmission(
		lease.ID, "semantic-v2-persistence", "",
	))
	if err != nil {
		t.Fatal(err)
	}

	var text string
	var evidenceIDs string
	var role, shape, rawValue, rawUnit, canonicalValue, canonicalUnit sql.NullString
	if err := db.QueryRow(`
		SELECT measurement.raw_text, measurement.evidence_ids,
		       measurement.measurement_role, measurement.value_shape,
		       measurement.raw_value::text, measurement.raw_unit,
		       measurement.canonical_value::text, measurement.canonical_unit
		FROM variable_signal_measurements measurement
		JOIN variable_signals signal ON signal.id = measurement.variable_signal_id
		WHERE signal.semantic_submission_id = $1
	`, submission.SubmissionID).Scan(
		&text, &evidenceIDs, &role, &shape, &rawValue, &rawUnit, &canonicalValue, &canonicalUnit,
	); err != nil {
		t.Fatal(err)
	}
	if text != "production fell 10%" || evidenceIDs != "{"+semanticEvidenceID+"}" ||
		role.Valid || shape.Valid || rawValue.Valid || rawUnit.Valid || canonicalValue.Valid || canonicalUnit.Valid {
		t.Fatalf("narrative measurement text=%q evidence=%v structured=%#v/%#v/%#v/%#v/%#v/%#v",
			text, evidenceIDs, role, shape, rawValue, rawUnit, canonicalValue, canonicalUnit)
	}
	var directImpacts int
	if err := db.QueryRow(`SELECT count(*) FROM direct_impact_assertions WHERE semantic_submission_id = $1`, submission.SubmissionID).Scan(&directImpacts); err != nil {
		t.Fatal(err)
	}
	if directImpacts != 0 {
		t.Fatalf("direct impacts = %d, want 0", directImpacts)
	}
	finalized, err := service.SubmitReview(context.Background(), eventsemantics.ReviewSubmission{
		SubmissionID: submission.SubmissionID, ReviewerExecutionKey: "semantic-v2-persistence:reviewer",
		PromptHash: strings.Repeat("b", 64), Model: "fixture-reviewer",
		Items: semanticReviewItems("pass"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if finalized.Status != eventsemantics.StatusAccepted {
		t.Fatalf("finalized status = %q, want accepted", finalized.Status)
	}
	reanalysisLease, err := service.CreateContextLease(context.Background(), eventsemantics.ContextLeaseRequest{
		EventID: semanticEventID, SupersedesSubmissionID: submission.SubmissionID,
		AgentExecutionID: "semantic-v2-reanalysis", WorkerID: "semantic-v2-fixture",
		Lease: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	reanalysis, err := service.CreateSubmission(context.Background(), semanticSubmission(
		reanalysisLease.ID, "semantic-v2-reanalysis", submission.SubmissionID,
	))
	if err != nil {
		t.Fatal(err)
	}
	refinalized, err := service.SubmitReview(context.Background(), eventsemantics.ReviewSubmission{
		SubmissionID: reanalysis.SubmissionID, ReviewerExecutionKey: "semantic-v2-reanalysis:reviewer",
		PromptHash: strings.Repeat("b", 64), Model: "fixture-reviewer",
		Items: semanticReviewItems("fail"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if refinalized.Status != eventsemantics.StatusRejected {
		t.Fatalf("reanalysis status = %q, want rejected", refinalized.Status)
	}
	thirdLease, err := service.CreateContextLease(context.Background(), eventsemantics.ContextLeaseRequest{
		EventID: semanticEventID, SupersedesSubmissionID: reanalysis.SubmissionID,
		AgentExecutionID: "semantic-v2-third", WorkerID: "semantic-v2-fixture", Lease: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	third, err := service.CreateSubmission(context.Background(), semanticSubmission(
		thirdLease.ID, "semantic-v2-third", reanalysis.SubmissionID,
	))
	if err != nil {
		t.Fatal(err)
	}
	thirdFinalized, err := service.SubmitReview(context.Background(), eventsemantics.ReviewSubmission{
		SubmissionID: third.SubmissionID, ReviewerExecutionKey: "semantic-v2-third:reviewer",
		PromptHash: strings.Repeat("b", 64), Model: "fixture-reviewer", Items: semanticReviewItems("pass"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if thirdFinalized.Status != eventsemantics.StatusAccepted {
		t.Fatalf("third status = %q, want accepted", thirdFinalized.Status)
	}
	var firstStatus, secondStatus string
	if err := db.QueryRow(`SELECT status FROM event_semantic_submissions WHERE id = $1`, submission.SubmissionID).Scan(&firstStatus); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT status FROM event_semantic_submissions WHERE id = $1`, reanalysis.SubmissionID).Scan(&secondStatus); err != nil {
		t.Fatal(err)
	}
	if firstStatus != "superseded" || secondStatus != "superseded" {
		t.Fatalf("ancestor statuses = %q/%q, want superseded/superseded", firstStatus, secondStatus)
	}
	indeterminateLease, err := service.CreateContextLease(context.Background(), eventsemantics.ContextLeaseRequest{
		EventID: semanticEventID, SupersedesSubmissionID: third.SubmissionID,
		AgentExecutionID: "semantic-v2-indeterminate", WorkerID: "semantic-v2-fixture", Lease: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	indeterminate, err := service.CreateSubmission(context.Background(), semanticSubmission(
		indeterminateLease.ID, "semantic-v2-indeterminate", third.SubmissionID,
	))
	if err != nil {
		t.Fatal(err)
	}
	needsReanalysis, err := service.SubmitReview(context.Background(), eventsemantics.ReviewSubmission{
		SubmissionID:         indeterminate.SubmissionID,
		ReviewerExecutionKey: "semantic-v2-indeterminate:reviewer",
		PromptHash:           strings.Repeat("b", 64), Model: "fixture-reviewer",
		Items: semanticReviewItems("indeterminate"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if needsReanalysis.Status != eventsemantics.StatusNeedsReanalysis {
		t.Fatalf("first indeterminate status = %q", needsReanalysis.Status)
	}
	quarantined, err := service.SubmitReview(context.Background(), eventsemantics.ReviewSubmission{
		SubmissionID:         indeterminate.SubmissionID,
		ReviewerExecutionKey: "semantic-v2-indeterminate:adjudicator",
		PromptHash:           strings.Repeat("b", 64), Model: "fixture-reviewer",
		Items: semanticReviewItems("indeterminate"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if quarantined.Status != eventsemantics.StatusQuarantined {
		t.Fatalf("second indeterminate status = %q, want quarantined", quarantined.Status)
	}
}

func seedEventSemanticScenario(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`
INSERT INTO raw_documents (
  id, ingest_channel, source_type, source_name, source_url, title, content_text,
  raw_mime_type, language, published_at, collected_at, content_hash, ingest_status
) VALUES (
  $1, 'integration', 'news', 'Integration Primary Source', 'https://example.test/wafer',
  'Wafer production update', 'Integration source content', 'text/plain', 'en',
  '2026-07-28T08:00:00Z', '2026-07-28T08:01:00Z', $2, 'collected'
)`, semanticRawDocumentID, strings.Repeat("1", 64)); err != nil {
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
)`, semanticEventID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO event_sources (
  id, event_id, raw_document_id, source_level, evidence_excerpt, evidence_hash,
  evidence_relation, supports_fields, is_primary
) VALUES (
  $1, $2, $3, 'primary', 'Integration Wafer Fab production fell 10%', $4,
  'supports', ARRAY['title','factual_summary','occurred_at','fact_payload'], true
)`, semanticEvidenceID, semanticEventID, semanticRawDocumentID, strings.Repeat("2", 64)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE events SET primary_source_id = $2 WHERE id = $1`, semanticEventID, semanticEvidenceID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO entity_nodes (
  id, entity_key, entity_type, layer_code, name, canonical_name, aliases, status
) VALUES
  ($1, 'company:integration-wafer-fab', 'company', 'company',
   'Integration Wafer Fab', 'Integration Wafer Fab', ARRAY['IWF'], 'active'),
  ($2, 'product:integration-wafer', 'product', 'product',
   'Integration Wafer', 'Integration Wafer', ARRAY['8-inch wafer'], 'active')
`, semanticCompanyID, semanticProductID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO entity_edges (
  id, from_entity_id, to_entity_id, relation_type, evidence_note, status
) VALUES ($1, $2, $3, 'produces', 'Integration evidence', 'active')
`, semanticRelationID, semanticCompanyID, semanticProductID); err != nil {
		t.Fatal(err)
	}
}

func semanticSubmission(leaseID, executionID, supersedesSubmissionID string) eventsemantics.Submission {
	return eventsemantics.Submission{
		ContextLeaseID: leaseID, EventID: semanticEventID, AgentExecutionID: executionID,
		AgentKey: "event-semantic-enricher", AgentVersion: "event-semantic-enricher.v3",
		SupersedesSubmissionID: supersedesSubmissionID,
		GeneratorPromptHash:    strings.Repeat("a", 64), GeneratorModel: "fixture-generator",
		ReviewerPromptHash: strings.Repeat("b", 64), ReviewerModel: "fixture-reviewer",
		AdjudicatorPromptHash: strings.Repeat("b", 64), AdjudicatorModel: "fixture-reviewer",
		OntologyVersion: "event-semantics.objective-v3@1", AcceptancePolicyVersion: "event-semantics.objective-v2@1",
		EntityLinks: []eventsemantics.EntityLinkCandidate{{
			Key: "company", Mention: "Integration Wafer Fab", EntityID: semanticCompanyID,
			ProjectedEntityType: "company",
			EntityRole:          "event_subject", EvidenceIDs: []string{semanticEvidenceID},
			ResolutionMethod: "qdrant_exact", ResolutionConfidence: "1.00000",
		}},
		VariableSignals: []eventsemantics.VariableSignalCandidate{{
			Key: "production", SubjectLinkKey: "company",
			VariableKey: "production_volume", VariableVersion: 1,
			Direction: "decrease", AssertionModality: "actual",
			EvidenceIDs: []string{semanticEvidenceID},
			Measurements: []eventsemantics.MeasurementValue{{
				Text: "production fell 10%", EvidenceIDs: []string{semanticEvidenceID},
			}},
			ExtractionConfidence: "0.98000",
		}},
	}
}

func semanticReviewItems(decision string) []eventsemantics.ReviewItem {
	return []eventsemantics.ReviewItem{
		{CandidateType: "entity_link", CandidateKey: "company", Decision: decision,
			ReasonCodes: []string{"fixture_review"}, EvidenceIDs: []string{semanticEvidenceID}},
		{CandidateType: "variable_signal", CandidateKey: "production", Decision: decision,
			ReasonCodes: []string{"fixture_review"}, EvidenceIDs: []string{semanticEvidenceID}},
	}
}
