package service

import (
	"context"
	"database/sql"
	"errors"
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
	var snapshotIsNull, manifestHasEntities, manifestHasRelations bool
	var manifestHasEvent, manifestHasEvidence, manifestHasEntityTypes, manifestHasVariables, manifestHasRules bool
	var manifestBytes int
	if err := db.QueryRow(`
		SELECT context_snapshot IS NULL,
		       context_manifest ? 'entities', context_manifest ? 'relations',
		       context_manifest ? 'event', context_manifest ? 'evidence',
		       context_manifest ? 'entity_type_definitions',
		       context_manifest ? 'variable_definitions',
		       context_manifest ? 'direct_transmission_rules',
		       octet_length(context_manifest::text)
		FROM event_semantic_context_leases WHERE id = $1
	`, firstLease.ID).Scan(
		&snapshotIsNull, &manifestHasEntities, &manifestHasRelations,
		&manifestHasEvent, &manifestHasEvidence, &manifestHasEntityTypes,
		&manifestHasVariables, &manifestHasRules, &manifestBytes,
	); err != nil {
		t.Fatal(err)
	}
	if !snapshotIsNull || manifestHasEntities || manifestHasRelations || manifestHasEvent ||
		manifestHasEvidence || manifestHasEntityTypes || manifestHasVariables || manifestHasRules ||
		manifestBytes >= 100_000 {
		t.Fatalf(
			"compact manifest snapshotNull=%v entities=%v relations=%v bytes=%d",
			snapshotIsNull, manifestHasEntities, manifestHasRelations, manifestBytes,
		)
	}
	if _, err := db.Exec(`
		UPDATE entity_nodes SET aliases = array_append(aliases, 'Late Alias') WHERE id = $1
	`, semanticCompanyID); err != nil {
		t.Fatal(err)
	}
	lateResolution, err := semanticService.Resolve(ctx, firstLease.ID, []eventsemantics.EntityMention{{
		Mention: "Late Alias", AllowedEntityTypes: []string{"company"},
	}})
	if err != nil || len(lateResolution) != 1 || len(lateResolution[0].Candidates) != 1 {
		t.Fatalf("compact lease did not resolve current formal Entity: %#v, err=%v", lateResolution, err)
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

func TestPostgresMigration000035PreservesLegacyLeaseAndSupportsMixedVersionReplay(t *testing.T) {
	db := openEventPublicationTestDatabaseAt(t, 34)
	seedEventSemanticScenario(t, db)
	legacyLeaseID := "10000000-0000-4000-8000-000000000035"
	request := eventsemantics.ContextLeaseRequest{
		EventID: semanticEventID, AgentExecutionID: "semantic-pre-000035-execution",
		WorkerID: "semantic-pre-000035-worker", Lease: 15 * time.Minute,
	}
	if _, err := db.Exec(`
		INSERT INTO event_semantic_context_leases(
		  id, event_id, agent_execution_id, worker_id, status, lease_expires_at, context_snapshot
		) VALUES ($1, $2, $3, $4, 'active', now() + interval '15 minutes', '{"legacy":"snapshot"}'::jsonb)
	`, legacyLeaseID, request.EventID, request.AgentExecutionID, request.WorkerID); err != nil {
		t.Fatal(err)
	}
	applyEventPublicationMigration(t, db, 35)
	var legacySnapshotPresent, legacyManifestAbsent bool
	if err := db.QueryRow(`
		SELECT context_snapshot = '{"legacy":"snapshot"}'::jsonb, context_manifest IS NULL
		FROM event_semantic_context_leases WHERE id = $1
	`, legacyLeaseID).Scan(&legacySnapshotPresent, &legacyManifestAbsent); err != nil {
		t.Fatal(err)
	}
	if !legacySnapshotPresent || !legacyManifestAbsent {
		t.Fatalf("legacy row after migration snapshot=%v manifestAbsent=%v", legacySnapshotPresent, legacyManifestAbsent)
	}
	semanticService := eventsemantics.NewService(postgres.NewEventSemanticsStore(db))
	replayed, err := semanticService.CreateContextLease(context.Background(), request)
	if err != nil || replayed.ID != legacyLeaseID {
		t.Fatalf("mixed-version replay=%#v err=%v", replayed, err)
	}
	contextValue, err := semanticService.Context(context.Background(), replayed.ID)
	if err != nil || contextValue.Event.ID != semanticEventID || len(contextValue.Evidence) != 1 {
		t.Fatalf("replayed Context=%#v err=%v", contextValue, err)
	}
	if _, err := db.Exec(`UPDATE event_semantic_context_leases SET status='consumed', consumed_at=now() WHERE id=$1`, replayed.ID); err != nil {
		t.Fatal(err)
	}
	newLease, err := semanticService.CreateContextLease(context.Background(), eventsemantics.ContextLeaseRequest{
		EventID: semanticEventID, AgentExecutionID: "semantic-post-000035-execution",
		WorkerID: "semantic-post-000035-worker", Lease: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	var snapshotIsNull, manifestIsObject bool
	if err := db.QueryRow(`
		SELECT context_snapshot IS NULL, jsonb_typeof(context_manifest) = 'object'
		FROM event_semantic_context_leases WHERE id = $1
	`, newLease.ID).Scan(&snapshotIsNull, &manifestIsObject); err != nil {
		t.Fatal(err)
	}
	if !snapshotIsNull || !manifestIsObject {
		t.Fatalf("new lease snapshotIsNull=%v manifestIsObject=%v", snapshotIsNull, manifestIsObject)
	}
}

func TestPostgresEventSemanticsReplayUpgradesLegacySnapshotLeaseToCompactManifest(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	seedEventSemanticScenario(t, db)
	semanticService := eventsemantics.NewService(postgres.NewEventSemanticsStore(db))
	ctx := context.Background()
	request := eventsemantics.ContextLeaseRequest{
		EventID: semanticEventID, AgentExecutionID: "semantic-legacy-replay",
		WorkerID: "semantic-integration-worker", Lease: 15 * time.Minute,
	}
	lease, err := semanticService.CreateContextLease(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		UPDATE event_semantic_context_leases
		SET context_snapshot = '{}'::jsonb, context_manifest = NULL
		WHERE id = $1
	`, lease.ID); err != nil {
		t.Fatal(err)
	}
	replayed, err := semanticService.CreateContextLease(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.ID != lease.ID {
		t.Fatalf("replayed lease id = %q, want %q", replayed.ID, lease.ID)
	}
	manifest, err := semanticService.Context(ctx, lease.ID)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.ManifestContractVersion != "event-semantic-context-manifest.v1" || len(manifest.Evidence) == 0 {
		t.Fatalf("upgraded manifest = %#v", manifest)
	}
	var snapshotPreserved, manifestCreated bool
	if err := db.QueryRow(`
		SELECT context_snapshot IS NOT NULL, context_manifest IS NOT NULL
		FROM event_semantic_context_leases WHERE id = $1
	`, lease.ID).Scan(&snapshotPreserved, &manifestCreated); err != nil {
		t.Fatal(err)
	}
	if !snapshotPreserved || !manifestCreated {
		t.Fatalf("legacy replay snapshotPreserved=%v manifestCreated=%v", snapshotPreserved, manifestCreated)
	}
}

func TestPostgresEligibleEventPaginationAndLeaseShareTheInputContract(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	seedEventSemanticScenario(t, db)
	secondEventID := "10000000-0000-4000-8000-000000000012"
	secondEvidenceID := "10000000-0000-4000-8000-000000000013"
	invalidEventID := "10000000-0000-4000-8000-000000000022"
	invalidEvidenceID := "10000000-0000-4000-8000-000000000023"
	legacyNullableEventID := "10000000-0000-4000-8000-000000000032"
	legacyNullableEvidenceID := "10000000-0000-4000-8000-000000000033"
	for _, row := range []struct {
		eventID, evidenceID, excerpt, firstSeen, relation string
		supportsFields                                    []string
	}{
		{
			secondEventID, secondEvidenceID, "second valid context evidence",
			"2026-07-28T08:02:00Z", "context", []string{},
		},
		{
			invalidEventID, invalidEvidenceID, "",
			"2026-07-28T08:03:00Z", "supports", []string{"title"},
		},
	} {
		if _, err := db.Exec(`
			INSERT INTO events (
			  id, title, summary, event_time, first_seen_at, knowable_at,
			  event_status, fact_status, dedupe_key, fact_payload
			) VALUES (
			  $1::uuid, 'Semantic pagination Event', 'Semantic pagination Event summary',
			  '2026-07-28T08:00:00Z', $2, $2, 'confirmed', 'verified',
			  'event:semantic:' || $1::text, '{}'::jsonb
			)
		`, row.eventID, row.firstSeen); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`
			INSERT INTO event_sources (
			  id, event_id, raw_document_id, source_level, evidence_excerpt, evidence_hash,
			  evidence_relation, supports_fields, is_primary
			) VALUES (
			  $2, $1, $3, 'primary', $4, $5,
			  $6, $7, true
			)
		`, row.eventID, row.evidenceID, semanticRawDocumentID,
			row.excerpt, strings.Repeat("3", 64), row.relation,
			row.supportsFields); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(
			`UPDATE events SET primary_source_id = $2 WHERE id = $1`,
			row.eventID, row.evidenceID,
		); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`
		INSERT INTO events (
		  id, title, summary, event_time, first_seen_at, knowable_at,
		  event_status, fact_status, dedupe_key, fact_payload
		) VALUES (
		  $1::uuid, 'Legacy nullable Evidence Event', 'Legacy nullable Evidence summary',
		  '2026-07-28T08:00:00Z', '2026-07-28T08:04:00Z',
		  '2026-07-28T08:04:00Z', 'confirmed', 'verified',
		  'event:semantic:' || $1::text, '{}'::jsonb
		)
	`, legacyNullableEventID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO event_sources (
		  id, event_id, raw_document_id, source_level, evidence_excerpt,
		  evidence_hash, is_primary
		) VALUES (
		  $2, $1, $3, 'primary', 'legacy evidence without relation',
		  $4, true
		)
	`, legacyNullableEventID, legacyNullableEvidenceID, semanticRawDocumentID,
		strings.Repeat("4", 64)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(
		`UPDATE events SET primary_source_id = $2 WHERE id = $1`,
		legacyNullableEventID, legacyNullableEvidenceID,
	); err != nil {
		t.Fatal(err)
	}
	semanticService := eventsemantics.NewService(postgres.NewEventSemanticsStore(db))

	first, err := semanticService.ListEligibleEvents(context.Background(), 1, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Events) != 1 || first.NextCursor == "" {
		t.Fatalf("first page = %#v", first)
	}
	second, err := semanticService.ListEligibleEvents(
		context.Background(), 1, first.NextCursor,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Events) != 1 || second.Events[0].EventID != secondEventID ||
		second.NextCursor != "" {
		t.Fatalf("second page = %#v", second)
	}
	for _, page := range []eventsemantics.EligibleEventPage{first, second} {
		for _, item := range page.Events {
			if item.EventID == invalidEventID ||
				item.EventID == legacyNullableEventID {
				t.Fatalf("invalid historical Event appeared in eligible page: %#v", page)
			}
		}
	}
	_, err = semanticService.CreateContextLease(
		context.Background(),
		eventsemantics.ContextLeaseRequest{
			EventID: invalidEventID, AgentExecutionID: "invalid-semantic-execution",
			WorkerID: "semantic-integration-worker", Lease: 15 * time.Minute,
		},
	)
	var inputInvalid *eventsemantics.InputInvalidError
	if !errors.As(err, &inputInvalid) {
		t.Fatalf("invalid Event lease error = %T %v", err, err)
	}
	_, err = semanticService.CreateContextLease(
		context.Background(),
		eventsemantics.ContextLeaseRequest{
			EventID:          legacyNullableEventID,
			AgentExecutionID: "legacy-nullable-semantic-execution",
			WorkerID:         "semantic-integration-worker",
			Lease:            15 * time.Minute,
		},
	)
	if !errors.As(err, &inputInvalid) {
		t.Fatalf("legacy nullable Event lease error = %T %v", err, err)
	}
	manifest, err := postgres.AuditHistoricalEventSemantics(
		context.Background(), db, time.Date(2026, 7, 31, 9, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !containsID(manifest.ValidEventIDs, semanticEventID) ||
		!containsID(manifest.ValidEventIDs, secondEventID) ||
		!containsID(manifest.InvalidEventIDs, invalidEventID) ||
		!containsID(manifest.InvalidEventIDs, legacyNullableEventID) {
		t.Fatalf("historical manifest = %#v", manifest)
	}
}

func TestPostgresEventSemanticAnchorResolutionPersistsOnlySelectedBindingAndDetectsPathDrift(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	seedEventSemanticScenario(t, db)
	seedEventSemanticAnchorScenario(t, db)
	semanticService := eventsemantics.NewService(postgres.NewEventSemanticsStore(db))
	ctx := context.Background()
	lease, err := semanticService.CreateContextLease(ctx, eventsemantics.ContextLeaseRequest{
		EventID: semanticEventID, AgentExecutionID: "semantic-anchor-execution",
		WorkerID: "semantic-anchor-worker", Lease: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	routes, err := semanticService.ListResolutionRoutes(ctx, lease.ID, "chain_node")
	if err != nil || len(routes) != 1 {
		t.Fatalf("routes=%#v err=%v", routes, err)
	}
	_, err = semanticService.ListResolutionAnchors(
		ctx, lease.ID, "chain-node-via-industry.v1", semanticCompanyID, nil, 50, "",
	)
	var invalidPartition *eventsemantics.ValidationError
	if !errors.As(err, &invalidPartition) {
		t.Fatalf("unknown partition error = %T %v", err, err)
	}
	anchors, err := semanticService.ListResolutionAnchors(
		ctx, lease.ID, "chain-node-via-industry.v1", semanticIndustryRootID, nil, 50, "",
	)
	if err != nil || len(anchors.Anchors) != 1 || anchors.Anchors[0].Entity.ID != semanticIndustryID {
		t.Fatalf("anchors=%#v err=%v", anchors, err)
	}
	_, err = semanticService.ResolveChainNodeCandidates(
		ctx, lease.ID, "chain-node-via-industry.v1", []string{semanticCompanyID}, 50, "",
	)
	var invalidAnchor *eventsemantics.ValidationError
	if !errors.As(err, &invalidAnchor) {
		t.Fatalf("wrong-type anchor error = %T %v", err, err)
	}
	candidates, err := semanticService.ResolveChainNodeCandidates(
		ctx, lease.ID, "chain-node-via-industry.v1", []string{semanticIndustryID}, 50, "",
	)
	if err != nil || len(candidates.Candidates) != 1 || candidates.Candidates[0].Entity.ID != semanticChainNodeID {
		t.Fatalf("candidates=%#v err=%v", candidates, err)
	}
	receipt := candidates.Candidates[0].Receipt
	request := semanticAnchorSubmission(lease.ID, receipt)
	if _, err := db.Exec(`
		UPDATE industry_chain_node_memberships SET updated_at = updated_at + interval '1 second'
		WHERE industry_chain_entity_id = $1 AND chain_node_entity_id = $2
	`, semanticIndustryChainID, semanticChainNodeID); err != nil {
		t.Fatal(err)
	}
	_, err = semanticService.CreateSubmission(ctx, request)
	var drift *eventsemantics.ContextDriftError
	if !errors.As(err, &drift) {
		t.Fatalf("drift error = %T %v", err, err)
	}
	refreshed, err := semanticService.ResolveChainNodeCandidates(
		ctx, lease.ID, "chain-node-via-industry.v1", []string{semanticIndustryID}, 50, "",
	)
	if err != nil || len(refreshed.Candidates) != 1 {
		t.Fatalf("refreshed=%#v err=%v", refreshed, err)
	}
	request.EntityLinks[0].ResolutionReceipt = &refreshed.Candidates[0].Receipt
	if _, err := db.Exec(`
		UPDATE entity_nodes SET updated_at = updated_at + interval '1 second' WHERE id = $1
	`, semanticChainNodeID); err != nil {
		t.Fatal(err)
	}
	_, err = semanticService.CreateSubmission(ctx, request)
	if !errors.As(err, &drift) {
		t.Fatalf("selected Entity drift error = %T %v", err, err)
	}
	refreshed, err = semanticService.ResolveChainNodeCandidates(
		ctx, lease.ID, "chain-node-via-industry.v1", []string{semanticIndustryID}, 50, "",
	)
	if err != nil || len(refreshed.Candidates) != 1 {
		t.Fatalf("Entity-refreshed=%#v err=%v", refreshed, err)
	}
	request.EntityLinks[0].ResolutionReceipt = &refreshed.Candidates[0].Receipt
	submission, err := semanticService.CreateSubmission(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	var bindingCount int
	if err := db.QueryRow(`
		SELECT count(*) FROM event_semantic_resolution_bindings
		WHERE semantic_submission_id = $1
	`, submission.SubmissionID).Scan(&bindingCount); err != nil {
		t.Fatal(err)
	}
	if bindingCount != 1 {
		t.Fatalf("selected resolution binding count = %d", bindingCount)
	}
}

func TestPostgresEventSemanticResolutionUsesKeysetPagesForL3AnchorsAndCandidates(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	seedEventSemanticScenario(t, db)
	seedEventSemanticAnchorScenario(t, db)
	for _, statement := range []string{
		`INSERT INTO entity_nodes(id, entity_key, entity_type, layer_code, name, canonical_name, aliases, status)
		 VALUES ('10000000-0000-4000-8000-000000000019', 'industry:semantic-fixture-l3-z', 'industry', 'industry', 'Z Fixture Industry', 'Z Fixture Industry', '{}', 'active'),
		        ('10000000-0000-4000-8000-000000000020', 'chain-node:semantic-fixture-z', 'chain_node', 'chain_node', 'Z Formal Node', 'Z Formal Node', '{}', 'active')`,
		`INSERT INTO industry_profiles(entity_id, classification_system, classification_version, industry_code, classification_level, parent_industry_entity_id, hierarchy_path_codes, definition, boundary_note, review_status)
		 VALUES ('10000000-0000-4000-8000-000000000019', 'fixture', 'v1', 'F010102', 3, '` + semanticIndustryL2ID + `', ARRAY['F01','F0101','F010102'], 'Z fixture', 'Z fixture', 'approved')`,
		`INSERT INTO chain_node_profiles(entity_id, definition, boundary_note, review_status)
		 VALUES ('10000000-0000-4000-8000-000000000020', 'Z formal node', NULL, 'approved')`,
		`INSERT INTO industry_chain_node_memberships(industry_chain_entity_id, chain_node_entity_id, position, contextual_stage, review_status, status, inclusion_reason, evidence_ids, source_name, source_url, verified_at)
		 VALUES ('` + semanticIndustryChainID + `', '10000000-0000-4000-8000-000000000020', 2, 'midstream', 'approved', 'active', 'Z fixture', ARRAY['fixture:z'], 'integration', 'artifact://event-semantic-anchor/z', now())`,
		`INSERT INTO entity_edges(id, from_entity_id, to_entity_id, relation_type, evidence_note, status)
		 VALUES ('10000000-0000-4000-8000-000000000021', '` + semanticIndustryChainID + `', '10000000-0000-4000-8000-000000000019', 'mapped_to_industry', 'Z fixture mapping', 'active')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	service := eventsemantics.NewService(postgres.NewEventSemanticsStore(db))
	lease, err := service.CreateContextLease(context.Background(), eventsemantics.ContextLeaseRequest{
		EventID: semanticEventID, AgentExecutionID: "semantic-keyset-execution",
		WorkerID: "semantic-keyset-worker", Lease: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	firstAnchors, err := service.ListResolutionAnchors(context.Background(), lease.ID, "chain-node-via-industry.v1", semanticIndustryRootID, nil, 1, "")
	if err != nil || len(firstAnchors.Anchors) != 1 || firstAnchors.NextCursor == "" || firstAnchors.Anchors[0].Entity.ID != semanticIndustryID {
		t.Fatalf("first anchor page=%#v err=%v", firstAnchors, err)
	}
	secondAnchors, err := service.ListResolutionAnchors(context.Background(), lease.ID, "chain-node-via-industry.v1", semanticIndustryRootID, nil, 1, firstAnchors.NextCursor)
	if err != nil || len(secondAnchors.Anchors) != 1 || secondAnchors.NextCursor != "" || secondAnchors.Anchors[0].Entity.ID == firstAnchors.Anchors[0].Entity.ID {
		t.Fatalf("second anchor page=%#v err=%v", secondAnchors, err)
	}
	firstCandidates, err := service.ResolveChainNodeCandidates(context.Background(), lease.ID, "chain-node-via-industry.v1", []string{semanticIndustryID}, 1, "")
	if err != nil || len(firstCandidates.Candidates) != 1 || firstCandidates.NextCursor == "" || firstCandidates.Candidates[0].Entity.ID != semanticChainNodeID {
		t.Fatalf("first candidate page=%#v err=%v", firstCandidates, err)
	}
	secondCandidates, err := service.ResolveChainNodeCandidates(context.Background(), lease.ID, "chain-node-via-industry.v1", []string{semanticIndustryID}, 1, firstCandidates.NextCursor)
	if err != nil || len(secondCandidates.Candidates) != 1 || secondCandidates.NextCursor != "" || secondCandidates.Candidates[0].Entity.ID == firstCandidates.Candidates[0].Entity.ID {
		t.Fatalf("second candidate page=%#v err=%v", secondCandidates, err)
	}
}

func containsID(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

const (
	semanticRawDocumentID   = "10000000-0000-4000-8000-000000000001"
	semanticEventID         = "10000000-0000-4000-8000-000000000002"
	semanticEvidenceID      = "10000000-0000-4000-8000-000000000003"
	semanticCompanyID       = "10000000-0000-4000-8000-000000000004"
	semanticProductID       = "10000000-0000-4000-8000-000000000005"
	semanticRelationID      = "10000000-0000-4000-8000-000000000006"
	semanticIndustryRootID  = "10000000-0000-4000-8000-000000000007"
	semanticIndustryL2ID    = "10000000-0000-4000-8000-000000000017"
	semanticIndustryID      = "10000000-0000-4000-8000-000000000018"
	semanticIndustryChainID = "10000000-0000-4000-8000-000000000008"
	semanticChainNodeID     = "10000000-0000-4000-8000-000000000009"
	semanticMappingID       = "10000000-0000-4000-8000-000000000010"
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
			EntityRole: "event_subject", EvidenceIDs: []string{semanticEvidenceID},
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

func seedEventSemanticAnchorScenario(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, statement := range []string{
		`INSERT INTO entity_nodes (
		    id, entity_key, entity_type, layer_code, name, canonical_name, aliases, status
		) VALUES
		    ('` + semanticIndustryRootID + `', 'industry:semantic-fixture-root', 'industry', 'industry', 'Fixture Industry', 'Fixture Industry', ARRAY['Fixture Sector'], 'active'),
		    ('` + semanticIndustryL2ID + `', 'industry:semantic-fixture-l2', 'industry', 'industry', 'Fixture Components', 'Fixture Components', ARRAY['Fixture Component Sector'], 'active'),
		    ('` + semanticIndustryID + `', 'industry:semantic-fixture-l3', 'industry', 'industry', 'Fixture Wafer Capacity', 'Fixture Wafer Capacity', ARRAY['Fixture Wafer'], 'active'),
		    ('` + semanticIndustryChainID + `', 'industry-chain:semantic-fixture', 'industry_chain', 'industry_chain', 'Fixture Chain', 'Fixture Chain', '{}', 'active'),
		    ('` + semanticChainNodeID + `', 'chain-node:semantic-fixture', 'chain_node', 'chain_node', 'Formal Manufacturing Node', 'Formal Manufacturing Node', ARRAY['critical manufacturing stage'], 'active')`,
		`INSERT INTO industry_profiles (
		    entity_id, classification_system, classification_version, industry_code,
		    classification_level, parent_industry_entity_id, hierarchy_path_codes,
		    definition, boundary_note, review_status
		) VALUES
		    ('` + semanticIndustryRootID + `', 'fixture', 'v1', 'F01', 1, NULL,
		     ARRAY['F01'], 'Fixture Industry definition', 'Fixture Industry boundary', 'approved'),
		    ('` + semanticIndustryL2ID + `', 'fixture', 'v1', 'F0101', 2, '` + semanticIndustryRootID + `',
		     ARRAY['F01','F0101'], 'Fixture Components definition', 'Fixture Components boundary', 'approved'),
		    ('` + semanticIndustryID + `', 'fixture', 'v1', 'F010101', 3, '` + semanticIndustryL2ID + `',
		     ARRAY['F01','F0101','F010101'], 'Fixture Wafer definition', 'Fixture Wafer boundary', 'approved')`,
		`INSERT INTO chain_node_profiles (entity_id, definition, boundary_note, review_status)
		 VALUES ('` + semanticChainNodeID + `', 'Formal node definition', 'Formal node boundary', 'approved')`,
		`INSERT INTO industry_chain_definitions (
		    entity_id, scope, target_output, end_use, observable_variables,
		    geography, as_of_date, review_status
		) VALUES ('` + semanticIndustryChainID + `', 'Fixture chain scope', 'Fixture output',
		    'Fixture end use', ARRAY['production_volume'], 'CN', CURRENT_DATE, 'approved')`,
		`INSERT INTO industry_chain_node_memberships (
		    industry_chain_entity_id, chain_node_entity_id, position, contextual_stage,
		    review_status, status, inclusion_reason, evidence_ids, source_name, source_url, verified_at
		) VALUES ('` + semanticIndustryChainID + `', '` + semanticChainNodeID + `', 1, 'upstream',
		    'approved', 'active', 'Fixture membership', ARRAY['fixture:evidence'],
		    'integration', 'artifact://event-semantic-anchor', now())`,
		`INSERT INTO entity_edges (
		    id, from_entity_id, to_entity_id, relation_type, evidence_note, status
		) VALUES ('` + semanticMappingID + `', '` + semanticIndustryChainID + `', '` + semanticIndustryID + `',
		    'mapped_to_industry', 'Fixture approved mapping', 'active')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
}

func semanticAnchorSubmission(
	leaseID string,
	receipt eventsemantics.ResolutionReceipt,
) eventsemantics.Submission {
	return eventsemantics.Submission{
		ContextLeaseID: leaseID, EventID: semanticEventID,
		AgentExecutionID: "semantic-anchor-execution",
		AgentKey:         "event-semantic-enricher", AgentVersion: "event-semantic-enricher.v1",
		GeneratorPromptHash: strings.Repeat("a", 64), GeneratorModel: "fixture-generator",
		ReviewerPromptHash: strings.Repeat("b", 64), ReviewerModel: "fixture-reviewer",
		OntologyVersion:         "event-semantics.phase-one@1",
		AcceptancePolicyVersion: "event-semantics.phase-one@1",
		EntityLinks: []eventsemantics.EntityLinkCandidate{{
			Key: "node", Mention: "upstream critical manufacturing stage",
			EntityID: semanticChainNodeID, EntityRole: "event_subject",
			EvidenceIDs:          []string{semanticEvidenceID},
			ResolutionMethod:     "data_service_anchor_resolution",
			ResolutionConfidence: "0.90000", ResolutionReceipt: &receipt,
		}},
		VariableSignals: []eventsemantics.VariableSignalCandidate{{
			Key: "production", SubjectLinkKey: "node", VariableKey: "production_volume",
			VariableVersion: 1, Direction: "decrease", AssertionModality: "actual",
			EvidenceIDs: []string{semanticEvidenceID}, ExtractionConfidence: "0.90000",
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
