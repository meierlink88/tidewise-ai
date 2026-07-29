package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventsemantics"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/postgres"
)

func TestPostgresEventSemanticsReanalysisSupersedesPriorSubmissionAndReadsCompleteAudit(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	seedEventSemanticScenario(t, db)
	semanticService := eventsemantics.NewService(postgres.NewEventSemanticsStore(db))
	ctx := context.Background()

	firstLease, err := semanticService.CreateContextLease(ctx, eventsemantics.ContextLeaseRequest{
		EventID: semanticEventID, AgentExecutionID: "semantic-execution-1",
		WorkerID: "semantic-integration-worker", Lease: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	replayedLease, err := semanticService.CreateContextLease(ctx, eventsemantics.ContextLeaseRequest{
		EventID: semanticEventID, AgentExecutionID: "semantic-execution-1",
		WorkerID: "semantic-integration-worker", Lease: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	if replayedLease.ID != firstLease.ID ||
		replayedLease.LeaseExpiresAt.Before(firstLease.LeaseExpiresAt) {
		t.Fatalf("replayed Context Lease = %#v, want %#v", replayedLease, firstLease)
	}
	contextSnapshot, err := semanticService.Context(ctx, firstLease.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(contextSnapshot.Evidence) != 1 || len(contextSnapshot.Rules) < 1 {
		t.Fatalf("context snapshot = %#v", contextSnapshot)
	}
	if _, err := db.Exec(`
		UPDATE entity_nodes SET aliases = array_append(aliases, 'Late Alias') WHERE id = $1
	`, semanticCompanyID); err != nil {
		t.Fatal(err)
	}
	lateResolution, err := semanticService.Resolve(ctx, firstLease.ID, []eventsemantics.EntityMention{{
		Mention: "Late Alias", AllowedEntityTypes: []string{"company"},
	}})
	if err != nil || len(lateResolution) != 1 || len(lateResolution[0].Candidates) != 0 {
		t.Fatalf("lease observed post-snapshot Entity mutation: %#v, err=%v", lateResolution, err)
	}
	resolutions, err := semanticService.Resolve(ctx, firstLease.ID, []eventsemantics.EntityMention{{
		Mention: "Integration Wafer Fab", AllowedEntityTypes: []string{"company"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(resolutions) != 1 || len(resolutions[0].Candidates) != 1 ||
		resolutions[0].Candidates[0].ID != semanticCompanyID {
		t.Fatalf("resolutions = %#v", resolutions)
	}
	targets, err := semanticService.SearchDirectTargets(
		ctx, firstLease.ID, semanticCompanyID, []string{"product"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 || targets[0].Entity.ID != semanticProductID {
		t.Fatalf("targets = %#v", targets)
	}

	firstRequest := semanticSubmission(firstLease.ID, "semantic-execution-1", "")
	firstSubmission, err := semanticService.CreateSubmission(ctx, firstRequest)
	if err != nil {
		t.Fatal(err)
	}
	if firstSubmission.Status != eventsemantics.StatusPendingReview {
		t.Fatalf("first status = %q", firstSubmission.Status)
	}
	replayedSubmission, err := semanticService.CreateSubmission(ctx, firstRequest)
	if err != nil {
		t.Fatal(err)
	}
	if !replayedSubmission.Replayed || replayedSubmission.SubmissionID != firstSubmission.SubmissionID {
		t.Fatalf("replayedSubmission run = %#v", replayedSubmission)
	}
	firstReview, err := semanticService.SubmitReview(ctx, eventsemantics.ReviewSubmission{
		SubmissionID: firstSubmission.SubmissionID, ReviewerExecutionKey: "semantic-execution-1:reviewer",
		PromptHash: strings.Repeat("b", 64), Model: "fixture-reviewer",
		Items: semanticReviewItems("indeterminate"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if firstReview.Status != eventsemantics.StatusNeedsReanalysis {
		t.Fatalf("first review status = %q", firstReview.Status)
	}
	if _, err := db.Exec(`
		UPDATE event_semantic_context_leases
		SET status = 'expired', lease_expires_at = now() - interval '1 minute'
		WHERE id = $1
	`, firstLease.ID); err != nil {
		t.Fatal(err)
	}
	recoveredLease, err := semanticService.CreateContextLease(ctx, eventsemantics.ContextLeaseRequest{
		EventID: semanticEventID, AgentExecutionID: "semantic-execution-1",
		WorkerID: "semantic-integration-worker", Lease: 15 * time.Minute,
	})
	if err != nil || recoveredLease.ID != firstLease.ID || recoveredLease.Status != "active" {
		t.Fatalf("recovered Context Lease = %#v, err=%v", recoveredLease, err)
	}
	recoveredContext, err := semanticService.Context(ctx, recoveredLease.ID)
	if err != nil || recoveredContext.Event.ID != contextSnapshot.Event.ID ||
		len(recoveredContext.Entities) != len(contextSnapshot.Entities) {
		t.Fatalf("recovered frozen Context = %#v, err=%v", recoveredContext, err)
	}

	secondLease, err := semanticService.CreateContextLease(ctx, eventsemantics.ContextLeaseRequest{
		EventID: semanticEventID, SupersedesSubmissionID: firstSubmission.SubmissionID,
		AgentExecutionID: "semantic-execution-2", WorkerID: "semantic-integration-worker",
		Lease: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	secondRequest := semanticSubmission(
		secondLease.ID, "semantic-execution-2", firstSubmission.SubmissionID,
	)
	secondSubmission, err := semanticService.CreateSubmission(ctx, secondRequest)
	if err != nil {
		t.Fatal(err)
	}
	secondReview, err := semanticService.SubmitReview(ctx, eventsemantics.ReviewSubmission{
		SubmissionID: secondSubmission.SubmissionID, ReviewerExecutionKey: "semantic-execution-2:reviewer",
		PromptHash: strings.Repeat("b", 64), Model: "fixture-reviewer",
		Items: semanticReviewItems("pass"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if secondReview.Status != eventsemantics.StatusAccepted {
		t.Fatalf("second review status = %q", secondReview.Status)
	}
	replayedSubmissionAcceptedSubmission, err := semanticService.CreateSubmission(ctx, secondRequest)
	if err != nil {
		t.Fatalf("replay accepted Submission after Context Lease consumption: %v", err)
	}
	if !replayedSubmissionAcceptedSubmission.Replayed || replayedSubmissionAcceptedSubmission.SubmissionID != secondSubmission.SubmissionID ||
		replayedSubmissionAcceptedSubmission.Status != eventsemantics.StatusAccepted {
		t.Fatalf("replayedSubmission accepted Run = %#v", replayedSubmissionAcceptedSubmission)
	}
	acceptedReanalysisLease, err := semanticService.CreateContextLease(ctx, eventsemantics.ContextLeaseRequest{
		EventID: semanticEventID, SupersedesSubmissionID: secondSubmission.SubmissionID,
		AgentExecutionID: "semantic-execution-3", WorkerID: "semantic-integration-worker",
		Lease: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	thirdRequest := semanticSubmission(
		acceptedReanalysisLease.ID, "semantic-execution-3", secondSubmission.SubmissionID,
	)
	thirdSubmission, err := semanticService.CreateSubmission(ctx, thirdRequest)
	if err != nil {
		t.Fatal(err)
	}
	thirdReview, err := semanticService.SubmitReview(ctx, eventsemantics.ReviewSubmission{
		SubmissionID: thirdSubmission.SubmissionID, ReviewerExecutionKey: "semantic-execution-3:reviewer",
		PromptHash: strings.Repeat("b", 64), Model: "fixture-reviewer",
		Items: semanticReviewItems("pass"),
	})
	if err != nil {
		t.Fatalf("accepted-to-accepted reanalysis: %v", err)
	}
	if thirdReview.Status != eventsemantics.StatusAccepted {
		t.Fatalf("third review status = %q", thirdReview.Status)
	}

	readResult, err := semanticService.Get(ctx, semanticEventID)
	if err != nil {
		t.Fatal(err)
	}
	if len(readResult.Submissions) != 3 {
		t.Fatalf("read submissions = %#v", readResult.Submissions)
	}
	if readResult.Submissions[0].Status != eventsemantics.StatusSuperseded ||
		readResult.Submissions[1].Status != eventsemantics.StatusSuperseded ||
		readResult.Submissions[2].Status != eventsemantics.StatusAccepted {
		t.Fatalf(
			"read statuses = %q, %q, %q",
			readResult.Submissions[0].Status, readResult.Submissions[1].Status, readResult.Submissions[2].Status,
		)
	}
	accepted := readResult.Submissions[2]
	if accepted.SupersedesSubmissionID != secondSubmission.SubmissionID ||
		len(accepted.ReviewSnapshots) != 1 ||
		len(accepted.CandidateSnapshot) == 0 ||
		len(accepted.Precheck.ReviewerWorkPackage.Evidence) != 1 ||
		accepted.Precheck.EntityLinks[0].RecordID == "" ||
		accepted.Precheck.VariableSignals[0].RecordID == "" ||
		accepted.Precheck.DirectImpacts[0].RecordID == "" {
		t.Fatalf("complete accepted audit = %#v", accepted)
	}
	for _, table := range []string{
		"event_entity_links", "variable_signals", "direct_impact_assertions",
	} {
		var count int
		query := fmt.Sprintf(
			`SELECT count(*) FROM %s WHERE semantic_submission_id = $1 AND review_status = 'superseded'`,
			table,
		)
		if err := db.QueryRow(query, firstSubmission.SubmissionID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("%s superseded count = %d", table, count)
		}
	}
}

const (
	semanticRawDocumentID = "10000000-0000-4000-8000-000000000001"
	semanticEventID       = "10000000-0000-4000-8000-000000000002"
	semanticEvidenceID    = "10000000-0000-4000-8000-000000000003"
	semanticCompanyID     = "10000000-0000-4000-8000-000000000004"
	semanticProductID     = "10000000-0000-4000-8000-000000000005"
	semanticRelationID    = "10000000-0000-4000-8000-000000000006"
)

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
	if _, err := db.Exec(
		`UPDATE events SET primary_source_id = $2 WHERE id = $1`,
		semanticEventID, semanticEvidenceID,
	); err != nil {
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

func semanticSubmission(
	claimID string,
	executionID string,
	supersedesSubmissionID string,
) eventsemantics.Submission {
	value := "-10"
	return eventsemantics.Submission{
		ContextLeaseID: claimID, EventID: semanticEventID, AgentExecutionID: executionID,
		AgentKey: "event-semantic-enricher", AgentVersion: "event-semantic-enricher.v1",
		SupersedesSubmissionID: supersedesSubmissionID,
		GeneratorPromptHash:    strings.Repeat("a", 64), GeneratorModel: "fixture-generator",
		ReviewerPromptHash: strings.Repeat("b", 64), ReviewerModel: "fixture-reviewer",
		OntologyVersion:         "event-semantics.phase-one@1",
		AcceptancePolicyVersion: "event-semantics.phase-one@1",
		EntityLinks: []eventsemantics.EntityLinkCandidate{{
			Key: "company", Mention: "Integration Wafer Fab", EntityID: semanticCompanyID,
			EntityRole: "subject", EvidenceIDs: []string{semanticEvidenceID},
			ResolutionMethod: "data_service_resolution", ResolutionConfidence: "0.99000",
		}},
		VariableSignals: []eventsemantics.VariableSignalCandidate{{
			Key: "production", SubjectLinkKey: "company",
			VariableKey: "production_volume", VariableVersion: 1,
			Direction: "decrease", AssertionModality: "actual",
			EvidenceIDs: []string{semanticEvidenceID},
			Measurements: []eventsemantics.MeasurementValue{{
				Role: "relative_change", Shape: "exact", RawValue: &value, RawUnit: "%",
				CanonicalValue: &value, CanonicalUnit: "percent",
				RawText: "production fell 10%", EvidenceID: semanticEvidenceID,
			}},
			ExtractionConfidence: "0.98000",
		}},
		DirectImpacts: []eventsemantics.DirectImpactCandidate{{
			Key: "supply", SourceSignalKey: "production", TargetEntityID: semanticProductID,
			AffectedVariableKey: "market_supply", AffectedVariableVersion: 1,
			AffectedDirection: "decrease", DerivationType: "rule_inferred",
			MechanismSummary: "Producer output fell, reducing direct Product supply.",
			EntityRelationID: semanticRelationID,
			RuleKey:          "production_decrease_reduces_product_supply", RuleVersion: 1,
			EvidenceIDs: []string{semanticEvidenceID}, AssertionConfidence: "0.96000",
		}},
	}
}

func semanticReviewItems(decision string) []eventsemantics.ReviewItem {
	return []eventsemantics.ReviewItem{
		{CandidateType: "entity_link", CandidateKey: "company", Decision: decision,
			ReasonCodes: []string{"fixture_review"}, EvidenceIDs: []string{semanticEvidenceID}},
		{CandidateType: "variable_signal", CandidateKey: "production", Decision: decision,
			ReasonCodes: []string{"fixture_review"}, EvidenceIDs: []string{semanticEvidenceID}},
		{CandidateType: "direct_impact", CandidateKey: "supply", Decision: decision,
			ReasonCodes: []string{"fixture_review"}, EvidenceIDs: []string{semanticEvidenceID}},
	}
}
