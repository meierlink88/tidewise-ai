package eventsemantic

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	eventbiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventsemantic"
)

func (r Store) CreateContextLease(
	ctx context.Context,
	request eventbiz.ContextLeaseRequest,
) (eventbiz.ContextLease, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return eventbiz.ContextLease{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
		UPDATE event_semantic_context_leases
		SET status = 'expired'
		WHERE status = 'active' AND lease_expires_at <= now()
	`); err != nil {
		return eventbiz.ContextLease{}, err
	}
	var existing eventbiz.ContextLease
	var existingWorkerID, existingEventID, existingSupersedes, submissionStatus string
	var existingManifest []byte
	err = tx.QueryRowContext(ctx, `
		SELECT lease.id, lease.event_id, COALESCE(lease.supersedes_submission_id::text, ''),
		       lease.worker_id, lease.status, lease.lease_expires_at,
		       COALESCE(submission.status, ''), lease.context_manifest
		FROM event_semantic_context_leases lease
		LEFT JOIN event_semantic_submissions submission ON submission.context_lease_id = lease.id
		WHERE lease.agent_execution_id = $1
		FOR UPDATE OF lease
	`, request.AgentExecutionID).Scan(
		&existing.ID, &existingEventID, &existingSupersedes, &existingWorkerID,
		&existing.Status, &existing.LeaseExpiresAt, &submissionStatus, &existingManifest,
	)
	if err == nil {
		if existingEventID != request.EventID || existingWorkerID != request.WorkerID ||
			existingSupersedes != request.SupersedesSubmissionID {
			return eventbiz.ContextLease{}, &eventbiz.ConflictError{
				Reason: "agent_execution_id is bound to a different Context Lease identity",
			}
		}
		if submissionStatus != "" && submissionStatus != string(eventbiz.StatusPendingReview) &&
			submissionStatus != string(eventbiz.StatusNeedsReanalysis) {
			return eventbiz.ContextLease{}, &eventbiz.ConflictError{
				Reason: "agent_execution_id already reached a terminal Semantic Submission",
			}
		}
		existing.EventID = existingEventID
		existing.SupersedesSubmissionID = existingSupersedes
		existing.Status = "active"
		existing.LeaseExpiresAt = time.Now().UTC().Add(request.Lease)
		var manifest eventbiz.ContextManifest
		if len(existingManifest) == 0 {
			manifest, err = buildEventSemanticManifest(
				ctx, tx, existing.ID, existing.EventID, request.AgentExecutionID,
				request.WorkerID, existing.LeaseExpiresAt,
			)
			if err != nil {
				return eventbiz.ContextLease{}, err
			}
		} else if err := json.Unmarshal(existingManifest, &manifest); err != nil {
			return eventbiz.ContextLease{}, err
		} else if err := validateEventSemanticManifestFingerprint(manifest); err != nil {
			return eventbiz.ContextLease{}, err
		}
		manifest.LeaseStatus = "active"
		manifest.LeaseExpiresAt = existing.LeaseExpiresAt
		manifest.ManifestFingerprint, err = eventSemanticManifestFingerprint(manifest)
		if err != nil {
			return eventbiz.ContextLease{}, err
		}
		existingManifest, err = json.Marshal(manifest)
		if err != nil {
			return eventbiz.ContextLease{}, err
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE event_semantic_context_leases
			SET status = 'active', lease_expires_at = $2, consumed_at = NULL,
			    context_manifest = $3::jsonb
			WHERE id = $1
		`, existing.ID, existing.LeaseExpiresAt, existingManifest); err != nil {
			return eventbiz.ContextLease{}, err
		}
		if err := tx.Commit(); err != nil {
			return eventbiz.ContextLease{}, err
		}
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return eventbiz.ContextLease{}, err
	}
	if request.SupersedesSubmissionID != "" {
		if _, err := tx.ExecContext(ctx, `
			UPDATE event_semantic_context_leases
			SET status = 'consumed', consumed_at = now()
			WHERE status = 'active'
			  AND id = (
			      SELECT context_lease_id
			      FROM event_semantic_submissions
			      WHERE id = $1
			  )
		`, request.SupersedesSubmissionID); err != nil {
			return eventbiz.ContextLease{}, err
		}
	}
	var eventID string
	err = tx.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT e.id
		FROM events e
		WHERE e.id = $1
		  AND %s
		  AND NOT EXISTS (
		      SELECT 1 FROM event_semantic_context_leases lease
		      WHERE lease.event_id = e.id AND lease.status = 'active'
		  )
		FOR UPDATE
	`, eventSemanticInputEligibilitySQL), request.EventID).Scan(&eventID)
	if errors.Is(err, sql.ErrNoRows) {
		return eventbiz.ContextLease{}, classifyEventSemanticLeaseEligibility(
			ctx, tx, request.EventID,
		)
	}
	if err != nil {
		return eventbiz.ContextLease{}, err
	}
	if request.SupersedesSubmissionID == "" {
		var exists bool
		if err := tx.QueryRowContext(ctx, `
			SELECT EXISTS (
			    SELECT 1 FROM event_semantic_submissions
			    WHERE event_id = $1 AND status <> 'superseded'
			)
		`, eventID).Scan(&exists); err != nil {
			return eventbiz.ContextLease{}, err
		}
		if exists {
			return eventbiz.ContextLease{}, &eventbiz.ConflictError{
				Reason: "Event already has an active Semantic Submission",
			}
		}
	} else {
		var priorEventID, priorStatus string
		if err := tx.QueryRowContext(ctx, `
			SELECT event_id, status
			FROM event_semantic_submissions
			WHERE id = $1
			FOR UPDATE
		`, request.SupersedesSubmissionID).Scan(&priorEventID, &priorStatus); errors.Is(err, sql.ErrNoRows) {
			return eventbiz.ContextLease{}, &eventbiz.NotFoundError{
				Resource: "superseded Event Semantic Submission",
			}
		} else if err != nil {
			return eventbiz.ContextLease{}, err
		}
		if priorEventID != eventID ||
			(priorStatus != "needs_reanalysis" && priorStatus != "accepted" &&
				priorStatus != "rejected" && priorStatus != "quarantined") {
			return eventbiz.ContextLease{}, &eventbiz.ConflictError{
				Reason: "supersedes_submission_id is not an active terminal Submission for this Event",
			}
		}
	}
	contextLease := eventbiz.ContextLease{
		ID: uuid.NewString(), EventID: eventID,
		SupersedesSubmissionID: request.SupersedesSubmissionID,
		Status:                 "active",
		LeaseExpiresAt:         time.Now().UTC().Add(request.Lease),
	}
	manifest, err := buildEventSemanticManifest(
		ctx, tx, contextLease.ID, contextLease.EventID, request.AgentExecutionID,
		request.WorkerID, contextLease.LeaseExpiresAt,
	)
	if err != nil {
		return eventbiz.ContextLease{}, err
	}
	manifestPayload, err := json.Marshal(manifest)
	if err != nil {
		return eventbiz.ContextLease{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO event_semantic_context_leases(
		    id, event_id, supersedes_submission_id, agent_execution_id, worker_id,
		    status, lease_expires_at, context_snapshot, context_manifest
		) VALUES ($1, $2, NULLIF($3, '')::uuid, $4, $5, 'active', $6, NULL, $7)
	`, contextLease.ID, contextLease.EventID, contextLease.SupersedesSubmissionID,
		request.AgentExecutionID, request.WorkerID, contextLease.LeaseExpiresAt,
		manifestPayload); err != nil {
		return eventbiz.ContextLease{}, err
	}
	if err := tx.Commit(); err != nil {
		return eventbiz.ContextLease{}, err
	}
	return contextLease, nil
}

func classifyEventSemanticLeaseEligibility(
	ctx context.Context,
	tx *sql.Tx,
	eventID string,
) error {
	var status, factStatus string
	var inputValid, activeLease bool
	err := tx.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT e.event_status, e.fact_status, (%s),
		       EXISTS (
		           SELECT 1
		           FROM event_semantic_context_leases lease
		           WHERE lease.event_id = e.id AND lease.status = 'active'
		       )
		FROM events e
		WHERE e.id = $1
	`, eventSemanticInputEligibilitySQL), eventID).Scan(
		&status, &factStatus, &inputValid, &activeLease,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return &eventbiz.NotFoundError{Resource: "Event"}
	}
	if err != nil {
		return err
	}
	if status != "confirmed" || factStatus != "verified" {
		return &eventbiz.NotRequiredError{
			Reason: "Event no longer requires initial Semantic processing",
		}
	}
	if !inputValid {
		return &eventbiz.InputInvalidError{
			Reason: "Event does not satisfy the Event Semantic input contract",
		}
	}
	if activeLease {
		return &eventbiz.ConflictError{
			Reason: "Event already has an active Context Lease",
		}
	}
	return &eventbiz.NotFoundError{Resource: "eligible Event"}
}

func (r Store) CreateSubmission(
	ctx context.Context,
	submission eventbiz.Submission,
	precheck eventbiz.PrecheckResult,
	payload []byte,
	hash string,
) (eventbiz.SubmissionResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return eventbiz.SubmissionResult{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if existing, found, err := existingEventSemanticSubmission(ctx, tx, submission.AgentExecutionID); err != nil {
		return eventbiz.SubmissionResult{}, err
	} else if found {
		if existing.CanonicalPayloadHash != hash {
			return eventbiz.SubmissionResult{}, &eventbiz.ConflictError{Reason: "agent_execution_id is bound to a different canonical payload"}
		}
		existing.Replayed = true
		return existing, nil
	}
	var eventID, leaseAgentExecutionID, status string
	var leaseSupersedesSubmissionID sql.NullString
	var manifestPayload []byte
	if err := tx.QueryRowContext(ctx, `
		SELECT event_id, agent_execution_id, status, supersedes_submission_id, context_manifest
		FROM event_semantic_context_leases
		WHERE id = $1 AND lease_expires_at > now() AND context_manifest IS NOT NULL
		FOR UPDATE
	`, submission.ContextLeaseID).Scan(
		&eventID, &leaseAgentExecutionID, &status, &leaseSupersedesSubmissionID, &manifestPayload,
	); errors.Is(err, sql.ErrNoRows) {
		return eventbiz.SubmissionResult{}, &eventbiz.NotFoundError{Resource: "Event Semantic Context Lease"}
	} else if err != nil {
		return eventbiz.SubmissionResult{}, err
	}
	if eventID != submission.EventID || status != "active" {
		return eventbiz.SubmissionResult{}, &eventbiz.ConflictError{
			Reason: "context lease is not active for this Event",
		}
	}
	if leaseAgentExecutionID != submission.AgentExecutionID {
		return eventbiz.SubmissionResult{}, &eventbiz.ConflictError{
			Reason: "Submission agent_execution_id differs from its Context Lease",
		}
	}
	if leaseSupersedesSubmissionID.String != submission.SupersedesSubmissionID {
		return eventbiz.SubmissionResult{}, &eventbiz.ConflictError{
			Reason: "Submission supersedes identity differs from its Context Lease",
		}
	}
	var manifestReference eventbiz.ContextManifest
	if err := json.Unmarshal(manifestPayload, &manifestReference); err != nil {
		return eventbiz.SubmissionResult{}, err
	}
	manifest, err := eventSemanticContextFromManifest(ctx, tx, manifestReference)
	if err != nil {
		return eventbiz.SubmissionResult{}, err
	}
	transactionContext, err := hydrateEventSemanticSubmissionContext(
		ctx, tx, manifest, submission, true,
	)
	if err != nil {
		return eventbiz.SubmissionResult{}, err
	}
	precheck = eventbiz.Precheck(transactionContext, submission)
	if submission.SupersedesSubmissionID != "" {
		var supersededEventID string
		var supersededStatus eventbiz.ReviewStatus
		if err := tx.QueryRowContext(ctx, `
			SELECT event_id, status
			FROM event_semantic_submissions
			WHERE id = $1
			FOR UPDATE
		`, submission.SupersedesSubmissionID).Scan(&supersededEventID, &supersededStatus); errors.Is(err, sql.ErrNoRows) {
			return eventbiz.SubmissionResult{}, &eventbiz.NotFoundError{Resource: "superseded Event Semantic Submission"}
		} else if err != nil {
			return eventbiz.SubmissionResult{}, err
		}
		if supersededEventID != submission.EventID || supersededStatus == eventbiz.StatusSuperseded {
			return eventbiz.SubmissionResult{}, &eventbiz.ConflictError{
				Reason: "supersedes_submission_id must reference the current Event's active prior Submission",
			}
		}
	}
	submissionID, snapshotID := uuid.NewString(), uuid.NewString()
	decisionPayload, err := json.Marshal(precheck)
	if err != nil {
		return eventbiz.SubmissionResult{}, err
	}
	counts, _ := json.Marshal(map[string]int{
		"entity_links":     len(submission.EntityLinks),
		"variable_signals": len(submission.VariableSignals),
	})
	submissionStatus := summarizeSemanticSubmission(precheck)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO event_semantic_submissions(
		    id, context_lease_id, event_id, agent_execution_id, agent_key, agent_version,
		    supersedes_submission_id, generator_prompt_hash, generator_model,
		    reviewer_prompt_hash, reviewer_model, adjudicator_prompt_hash,
		    adjudicator_model, ontology_version, acceptance_policy_key,
		    acceptance_policy_version, canonical_payload_hash, status,
		    candidate_counts, decision_summary
		) VALUES (
		    $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,
		    'event-semantics.objective-v2',1,$15,$16,$17,$18
	    )
	`, submissionID, submission.ContextLeaseID, submission.EventID, submission.AgentExecutionID,
		submission.AgentKey, submission.AgentVersion, nullString(submission.SupersedesSubmissionID),
		submission.GeneratorPromptHash, submission.GeneratorModel,
		submission.ReviewerPromptHash, submission.ReviewerModel,
		nullString(submission.AdjudicatorPromptHash), nullString(submission.AdjudicatorModel),
		submission.OntologyVersion, hash, submissionStatus, counts, decisionPayload,
	); err != nil {
		return eventbiz.SubmissionResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO event_semantic_candidate_snapshots(id, semantic_submission_id, payload, canonical_payload_hash)
		VALUES ($1,$2,$3,$4)
	`, snapshotID, submissionID, payload, hash); err != nil {
		return eventbiz.SubmissionResult{}, err
	}
	if err := insertReviewableSemanticCandidates(ctx, tx, submissionID, submission, precheck); err != nil {
		return eventbiz.SubmissionResult{}, err
	}
	if submissionStatus != eventbiz.StatusPendingReview &&
		submissionStatus != eventbiz.StatusNeedsReanalysis {
		if _, err := tx.ExecContext(ctx, `
			UPDATE event_semantic_context_leases
			SET status = 'consumed', consumed_at = now()
			WHERE id = $1
		`, submission.ContextLeaseID); err != nil {
			return eventbiz.SubmissionResult{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return eventbiz.SubmissionResult{}, err
	}
	return eventbiz.SubmissionResult{
		SubmissionID: submissionID, EventID: submission.EventID, Status: submissionStatus,
		CanonicalPayloadHash: hash, Precheck: precheck,
	}, nil
}

func (r Store) SubmitReview(
	ctx context.Context,
	submission eventbiz.ReviewSubmission,
	payload []byte,
	hash string,
) (eventbiz.SubmissionResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return eventbiz.SubmissionResult{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var reviewIdentity semanticReviewIdentity
	if err := tx.QueryRowContext(ctx, `
		SELECT agent_execution_id, reviewer_prompt_hash, reviewer_model,
		       COALESCE(adjudicator_prompt_hash, ''), COALESCE(adjudicator_model, '')
		FROM event_semantic_submissions
		WHERE id = $1
		FOR UPDATE
	`, submission.SubmissionID).Scan(
		&reviewIdentity.AgentExecutionID,
		&reviewIdentity.ReviewerPromptHash, &reviewIdentity.ReviewerModel,
		&reviewIdentity.AdjudicatorPromptHash, &reviewIdentity.AdjudicatorModel,
	); errors.Is(err, sql.ErrNoRows) {
		return eventbiz.SubmissionResult{}, &eventbiz.NotFoundError{Resource: "Event Semantic Submission"}
	} else if err != nil {
		return eventbiz.SubmissionResult{}, err
	}
	if !reviewIdentity.matches(submission) {
		return eventbiz.SubmissionResult{}, &eventbiz.ConflictError{
			Reason: "review prompt or model does not match the frozen Submission identity",
		}
	}
	var existingHash string
	err = tx.QueryRowContext(ctx, `
		SELECT canonical_payload_hash
		FROM event_semantic_review_snapshots
		WHERE semantic_submission_id = $1 AND reviewer_execution_key = $2
	`, submission.SubmissionID, submission.ReviewerExecutionKey).Scan(&existingHash)
	if err == nil {
		if existingHash != hash {
			return eventbiz.SubmissionResult{}, &eventbiz.ConflictError{Reason: "reviewer_execution_key is bound to a different payload"}
		}
		result, found, err := eventSemanticSubmissionByID(ctx, tx, submission.SubmissionID)
		if err != nil {
			return eventbiz.SubmissionResult{}, err
		}
		if !found {
			return eventbiz.SubmissionResult{}, &eventbiz.NotFoundError{Resource: "Event Semantic Submission"}
		}
		result.Replayed = true
		return result, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return eventbiz.SubmissionResult{}, err
	}
	result, found, err := eventSemanticSubmissionByID(ctx, tx, submission.SubmissionID)
	if err != nil {
		return eventbiz.SubmissionResult{}, err
	}
	if !found {
		return eventbiz.SubmissionResult{}, &eventbiz.NotFoundError{Resource: "Event Semantic Submission"}
	}
	if result.Status != eventbiz.StatusPendingReview && result.Status != eventbiz.StatusNeedsReanalysis {
		return eventbiz.SubmissionResult{}, &eventbiz.ConflictError{Reason: "Event Semantic Submission is not reviewable"}
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO event_semantic_review_snapshots(
		    id,semantic_submission_id,reviewer_execution_key,payload,canonical_payload_hash
		) VALUES ($1,$2,$3,$4,$5)
	`, uuid.NewString(), submission.SubmissionID, submission.ReviewerExecutionKey, payload, hash); err != nil {
		return eventbiz.SubmissionResult{}, err
	}
	var priorReviewCount, retryBudget int
	if err := tx.QueryRowContext(ctx, `
		SELECT
		    (SELECT count(*) FROM event_semantic_review_snapshots WHERE semantic_submission_id = $1),
		    policy.retry_budget
		FROM event_semantic_submissions run
		JOIN event_semantic_acceptance_policies policy
		  ON policy.policy_key = run.acceptance_policy_key
		 AND policy.version = run.acceptance_policy_version
		WHERE run.id = $1
	`, submission.SubmissionID).Scan(&priorReviewCount, &retryBudget); err != nil {
		return eventbiz.SubmissionResult{}, err
	}
	if err := applySemanticReview(
		ctx, tx, submission, &result.Precheck, priorReviewCount > retryBudget,
	); err != nil {
		return eventbiz.SubmissionResult{}, err
	}
	status := summarizeSemanticSubmission(result.Precheck)
	decisionPayload, err := json.Marshal(result.Precheck)
	if err != nil {
		return eventbiz.SubmissionResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE event_semantic_submissions
		SET status = $2::varchar, decision_summary = $3,
		    finalized_at = CASE
		        WHEN $2::text IN ('accepted','rejected','quarantined') THEN now()
		        ELSE NULL
		    END
		WHERE id = $1
	`, submission.SubmissionID, status, decisionPayload); err != nil {
		return eventbiz.SubmissionResult{}, err
	}
	if status == eventbiz.StatusAccepted ||
		status == eventbiz.StatusRejected ||
		status == eventbiz.StatusQuarantined {
		if _, err := tx.ExecContext(ctx, `
			UPDATE event_semantic_context_leases
			SET status = 'consumed', consumed_at = now()
			WHERE id = (SELECT context_lease_id FROM event_semantic_submissions WHERE id = $1)
		`, submission.SubmissionID); err != nil {
			return eventbiz.SubmissionResult{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return eventbiz.SubmissionResult{}, err
	}
	result.Status = status
	return result, nil
}

func applySemanticReview(
	ctx context.Context,
	tx *sql.Tx,
	submission eventbiz.ReviewSubmission,
	precheck *eventbiz.PrecheckResult,
	quarantineIndeterminate bool,
) error {
	pending := map[string]*eventbiz.CandidateDecision{}
	candidateEvidence := make(map[string][]string)
	register := func(kind string, items []eventbiz.CandidateDecision) {
		for index := range items {
			if items[index].Status == eventbiz.StatusPendingReview ||
				items[index].Status == eventbiz.StatusNeedsReanalysis {
				pending[kind+":"+items[index].CandidateKey] = &items[index]
			}
		}
	}
	register("entity_link", precheck.EntityLinks)
	register("variable_signal", precheck.VariableSignals)
	register("direct_impact", precheck.DirectImpacts)
	for _, candidate := range precheck.ReviewerWorkPackage.EntityLinks {
		candidateEvidence["entity_link:"+candidate.Key] = candidate.EvidenceIDs
	}
	for _, candidate := range precheck.ReviewerWorkPackage.VariableSignals {
		candidateEvidence["variable_signal:"+candidate.Key] = candidate.EvidenceIDs
	}
	for _, candidate := range precheck.ReviewerWorkPackage.DirectImpacts {
		candidateEvidence["direct_impact:"+candidate.Key] = candidate.EvidenceIDs
	}
	if len(submission.Items) != len(pending) {
		return &eventbiz.ValidationError{Reason: "review must decide every reviewable candidate exactly once"}
	}
	seen := make(map[string]struct{}, len(submission.Items))
	for _, item := range submission.Items {
		identity := item.CandidateType + ":" + item.CandidateKey
		decision, exists := pending[identity]
		if !exists {
			return &eventbiz.ConflictError{Reason: "review references a non-reviewable candidate"}
		}
		if _, duplicate := seen[identity]; duplicate {
			return &eventbiz.ValidationError{Reason: "review candidate identities must be unique"}
		}
		if !reviewEvidenceMatchesCandidate(item.EvidenceIDs, candidateEvidence[identity]) {
			return &eventbiz.ValidationError{Reason: "review Evidence must cite the candidate Event Evidence"}
		}
		seen[identity] = struct{}{}
		switch item.Decision {
		case "pass":
			decision.Status, decision.ReasonCode = eventbiz.StatusAccepted, ""
		case "fail":
			decision.Status, decision.ReasonCode = eventbiz.StatusRejected, firstReason(item.ReasonCodes, "reviewer_failed")
		case "indeterminate":
			if quarantineIndeterminate {
				decision.Status, decision.ReasonCode = eventbiz.StatusQuarantined, "unresolved_after_retry_budget"
			} else {
				decision.Status, decision.ReasonCode = eventbiz.StatusNeedsReanalysis, firstReason(item.ReasonCodes, "reviewer_indeterminate")
			}
		}
	}
	propagateSemanticReview(precheck)
	if summarizeSemanticSubmission(*precheck) == eventbiz.StatusAccepted {
		if err := supersedePriorSemanticSubmission(ctx, tx, submission.SubmissionID); err != nil {
			return err
		}
	}
	return persistSemanticDecisions(ctx, tx, submission.SubmissionID, *precheck)
}

func supersedePriorSemanticSubmission(ctx context.Context, tx *sql.Tx, runID string) error {
	for _, table := range []string{"event_entity_links", "variable_signals", "direct_impact_assertions"} {
		query := fmt.Sprintf(`
			WITH RECURSIVE ancestors(id) AS (
			    SELECT supersedes_submission_id
			    FROM event_semantic_submissions
			    WHERE id = $1 AND supersedes_submission_id IS NOT NULL
			    UNION ALL
			    SELECT run.supersedes_submission_id
			    FROM event_semantic_submissions run
			    JOIN ancestors ON run.id = ancestors.id
			    WHERE run.supersedes_submission_id IS NOT NULL
			)
			UPDATE %s
			SET review_status = 'superseded', reason_code = 'superseded_by_reanalysis', updated_at = now()
			WHERE semantic_submission_id IN (SELECT id FROM ancestors)
			  AND review_status <> 'superseded'
		`, table)
		if _, err := tx.ExecContext(ctx, query, runID); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `
		WITH RECURSIVE ancestors(id) AS (
		    SELECT supersedes_submission_id
		    FROM event_semantic_submissions
		    WHERE id = $1 AND supersedes_submission_id IS NOT NULL
		    UNION ALL
		    SELECT run.supersedes_submission_id
		    FROM event_semantic_submissions run
		    JOIN ancestors ON run.id = ancestors.id
		    WHERE run.supersedes_submission_id IS NOT NULL
		)
		UPDATE event_semantic_submissions
		SET status = 'superseded', finalized_at = now()
		WHERE id IN (SELECT id FROM ancestors) AND status <> 'superseded'
	`, runID); err != nil {
		return err
	}
	return nil
}

type semanticReviewIdentity struct {
	AgentExecutionID      string
	ReviewerPromptHash    string
	ReviewerModel         string
	AdjudicatorPromptHash string
	AdjudicatorModel      string
}

func (identity semanticReviewIdentity) matches(submission eventbiz.ReviewSubmission) bool {
	switch {
	case submission.ReviewerExecutionKey == identity.AgentExecutionID+":reviewer":
		return submission.PromptHash == identity.ReviewerPromptHash &&
			submission.Model == identity.ReviewerModel
	case submission.ReviewerExecutionKey == identity.AgentExecutionID+":adjudicator":
		return identity.AdjudicatorPromptHash != "" &&
			submission.PromptHash == identity.AdjudicatorPromptHash &&
			submission.Model == identity.AdjudicatorModel
	default:
		return false
	}
}

func reviewEvidenceMatchesCandidate(reviewed, candidate []string) bool {
	if len(reviewed) == 0 || len(candidate) == 0 {
		return false
	}
	allowed := make(map[string]struct{}, len(candidate))
	for _, evidenceID := range candidate {
		allowed[evidenceID] = struct{}{}
	}
	for _, evidenceID := range reviewed {
		if _, exists := allowed[evidenceID]; !exists {
			return false
		}
	}
	return true
}

func propagateSemanticReview(precheck *eventbiz.PrecheckResult) {
	linkStatus := make(map[string]eventbiz.CandidateDecision, len(precheck.EntityLinks))
	for _, item := range precheck.EntityLinks {
		linkStatus[item.CandidateKey] = item
	}
	signalStatus := make(map[string]eventbiz.CandidateDecision, len(precheck.VariableSignals))
	signalByKey := make(map[string]eventbiz.VariableSignalCandidate, len(precheck.ReviewerWorkPackage.VariableSignals))
	for _, item := range precheck.ReviewerWorkPackage.VariableSignals {
		signalByKey[item.Key] = item
	}
	for index := range precheck.VariableSignals {
		candidate := signalByKey[precheck.VariableSignals[index].CandidateKey]
		upstream := linkStatus[candidate.SubjectLinkKey]
		if upstream.Status == eventbiz.StatusRejected {
			precheck.VariableSignals[index].Status = eventbiz.StatusRejected
			precheck.VariableSignals[index].ReasonCode = "upstream_rejected"
		} else if upstream.Status == eventbiz.StatusQuarantined {
			precheck.VariableSignals[index].Status = eventbiz.StatusQuarantined
			precheck.VariableSignals[index].ReasonCode = "upstream_quarantined"
		} else if upstream.Status == eventbiz.StatusNeedsReanalysis {
			precheck.VariableSignals[index].Status = eventbiz.StatusNeedsReanalysis
			precheck.VariableSignals[index].ReasonCode = "upstream_pending"
		}
		signalStatus[precheck.VariableSignals[index].CandidateKey] = precheck.VariableSignals[index]
	}
	impactByKey := make(map[string]eventbiz.DirectImpactCandidate, len(precheck.ReviewerWorkPackage.DirectImpacts))
	for _, item := range precheck.ReviewerWorkPackage.DirectImpacts {
		impactByKey[item.Key] = item
	}
	for index := range precheck.DirectImpacts {
		candidate := impactByKey[precheck.DirectImpacts[index].CandidateKey]
		upstream := signalStatus[candidate.SourceSignalKey]
		if upstream.Status == eventbiz.StatusRejected {
			precheck.DirectImpacts[index].Status = eventbiz.StatusRejected
			precheck.DirectImpacts[index].ReasonCode = "upstream_rejected"
		} else if upstream.Status == eventbiz.StatusQuarantined {
			precheck.DirectImpacts[index].Status = eventbiz.StatusQuarantined
			precheck.DirectImpacts[index].ReasonCode = "upstream_quarantined"
		} else if upstream.Status == eventbiz.StatusNeedsReanalysis {
			precheck.DirectImpacts[index].Status = eventbiz.StatusNeedsReanalysis
			precheck.DirectImpacts[index].ReasonCode = "upstream_pending"
		}
	}
}

func summarizeSemanticSubmission(precheck eventbiz.PrecheckResult) eventbiz.ReviewStatus {
	hasAccepted, hasPending, hasNeeds, hasQuarantined := false, false, false, false
	for _, group := range [][]eventbiz.CandidateDecision{
		precheck.EntityLinks, precheck.VariableSignals, precheck.DirectImpacts,
	} {
		for _, item := range group {
			hasAccepted = hasAccepted || item.Status == eventbiz.StatusAccepted
			hasPending = hasPending || item.Status == eventbiz.StatusPendingReview
			hasNeeds = hasNeeds || item.Status == eventbiz.StatusNeedsReanalysis
			hasQuarantined = hasQuarantined || item.Status == eventbiz.StatusQuarantined
		}
	}
	if hasPending {
		return eventbiz.StatusPendingReview
	}
	if hasNeeds {
		return eventbiz.StatusNeedsReanalysis
	}
	if hasAccepted {
		return eventbiz.StatusAccepted
	}
	if hasQuarantined {
		return eventbiz.StatusQuarantined
	}
	return eventbiz.StatusRejected
}

func persistSemanticDecisions(
	ctx context.Context,
	tx *sql.Tx,
	runID string,
	precheck eventbiz.PrecheckResult,
) error {
	for table, items := range map[string][]eventbiz.CandidateDecision{
		"event_entity_links":       precheck.EntityLinks,
		"variable_signals":         precheck.VariableSignals,
		"direct_impact_assertions": precheck.DirectImpacts,
	} {
		for _, item := range items {
			query := fmt.Sprintf(`
				UPDATE %s
				SET review_status = $3, reason_code = $4, updated_at = now()
				WHERE semantic_submission_id = $1 AND candidate_key = $2
			`, table)
			if _, err := tx.ExecContext(
				ctx, query, runID, item.CandidateKey, item.Status, nullString(item.ReasonCode),
			); err != nil {
				return err
			}
		}
	}
	return nil
}
