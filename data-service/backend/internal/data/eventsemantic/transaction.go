package eventsemantic

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	eventbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/eventsemantic"
	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

func (r Store) InTransaction(
	ctx context.Context,
	fn func(eventbiz.Transaction) error,
) (resultErr error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin Event Semantic transaction: %w", err)
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		failure := recover()
		rollbackErr := tx.Rollback()
		if errors.Is(rollbackErr, sql.ErrTxDone) {
			rollbackErr = nil
		}
		if failure != nil {
			if rollbackErr != nil {
				panic(fmt.Errorf("Event Semantic panic (%v) and rollback failed: %w", failure, rollbackErr))
			}
			panic(failure)
		}
		if rollbackErr != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("roll back Event Semantic transaction: %w", rollbackErr))
		}
	}()
	if err := fn(&transaction{tx: tx}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit Event Semantic transaction: %w", err)
	}
	committed = true
	return nil
}

type transaction struct {
	tx *sql.Tx
}

func (t *transaction) LoadContextLeaseState(
	ctx context.Context,
	request eventbiz.ContextLeaseRequest,
	observedAt time.Time,
) (eventbiz.ContextLeaseTransactionState, error) {
	var state eventbiz.ContextLeaseTransactionState
	expiredRows, err := t.tx.QueryContext(ctx, `
SELECT id::text
FROM event_semantic_context_leases
WHERE status = 'active' AND lease_expires_at <= $1
ORDER BY id
FOR UPDATE
`, observedAt)
	if err != nil {
		return state, err
	}
	for expiredRows.Next() {
		var id string
		if err := expiredRows.Scan(&id); err != nil {
			expiredRows.Close()
			return eventbiz.ContextLeaseTransactionState{}, err
		}
		if !coreid.Is(id, coreid.EventSemanticContextLease) {
			expiredRows.Close()
			return eventbiz.ContextLeaseTransactionState{}, invalidPersistedEventSemantic("expired Context Lease reference")
		}
		state.ExpiredLeaseIDs = append(state.ExpiredLeaseIDs, id)
	}
	if err := expiredRows.Err(); err != nil {
		expiredRows.Close()
		return eventbiz.ContextLeaseTransactionState{}, err
	}
	if err := expiredRows.Close(); err != nil {
		return eventbiz.ContextLeaseTransactionState{}, err
	}

	var existing eventbiz.StoredContextLease
	var submissionStatus string
	err = t.tx.QueryRowContext(ctx, `
SELECT lease.id, lease.event_id, COALESCE(lease.supersedes_submission_id::text, ''),
       lease.agent_execution_id, lease.worker_id, lease.status, lease.lease_expires_at,
       COALESCE(submission.status, '')
FROM event_semantic_context_leases lease
LEFT JOIN event_semantic_submissions submission ON submission.context_lease_id = lease.id
WHERE lease.agent_execution_id = $1
FOR UPDATE OF lease
`, request.AgentExecutionID).Scan(
		&existing.ID, &existing.EventID, &existing.SupersedesSubmissionID,
		&existing.AgentExecutionID, &existing.WorkerID, &existing.Status,
		&existing.LeaseExpiresAt, &submissionStatus,
	)
	if err == nil {
		existing.SubmissionStatus = eventbiz.ReviewStatus(submissionStatus)
		if err := validatePersistedStoredContextLease(existing); err != nil {
			return eventbiz.ContextLeaseTransactionState{}, err
		}
		state.Existing = &existing
		return state, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return eventbiz.ContextLeaseTransactionState{}, err
	}

	state.Event.EventID = request.EventID
	err = t.tx.QueryRowContext(ctx, fmt.Sprintf(`
SELECT e.id, e.event_status, e.fact_status, (%s)
FROM events e
WHERE e.id = $1
FOR UPDATE
`, eventSemanticInputEligibilitySQL), request.EventID).Scan(
		&state.Event.EventID,
		&state.Event.EventStatus,
		&state.Event.FactStatus,
		&state.Event.InputValid,
	)
	if errors.Is(err, sql.ErrNoRows) {
		state.Event = eventbiz.LeaseEventState{}
		return state, nil
	}
	if err != nil {
		return eventbiz.ContextLeaseTransactionState{}, err
	}
	state.Event.Found = true
	if err := validatePersistedLeaseEventState(state.Event); err != nil {
		return eventbiz.ContextLeaseTransactionState{}, err
	}

	rows, err := t.tx.QueryContext(ctx, `
SELECT id::text
FROM event_semantic_context_leases
WHERE event_id = $1 AND status = 'active' AND lease_expires_at > $2
ORDER BY id
FOR UPDATE
`, request.EventID, observedAt)
	if err != nil {
		return eventbiz.ContextLeaseTransactionState{}, err
	}
	activeLeaseIDs := make([]string, 0, 1)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return eventbiz.ContextLeaseTransactionState{}, err
		}
		activeLeaseIDs = append(activeLeaseIDs, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return eventbiz.ContextLeaseTransactionState{}, err
	}
	if err := rows.Close(); err != nil {
		return eventbiz.ContextLeaseTransactionState{}, err
	}
	if len(activeLeaseIDs) > 1 {
		return eventbiz.ContextLeaseTransactionState{}, invalidPersistedEventSemantic("active Context Lease set")
	}
	if len(activeLeaseIDs) == 1 {
		if !coreid.Is(activeLeaseIDs[0], coreid.EventSemanticContextLease) {
			return eventbiz.ContextLeaseTransactionState{}, invalidPersistedEventSemantic("active Context Lease reference")
		}
		state.ActiveLeaseID = activeLeaseIDs[0]
	}

	submissionRows, err := t.tx.QueryContext(ctx, `
SELECT id::text, status
FROM event_semantic_submissions
WHERE event_id = $1
ORDER BY id
FOR UPDATE
`, request.EventID)
	if err != nil {
		return eventbiz.ContextLeaseTransactionState{}, err
	}
	for submissionRows.Next() {
		var id string
		var status eventbiz.ReviewStatus
		if err := submissionRows.Scan(&id, &status); err != nil {
			submissionRows.Close()
			return eventbiz.ContextLeaseTransactionState{}, err
		}
		if !coreid.Is(id, coreid.EventSemanticSubmission) || !validPersistedReviewStatus(status) {
			submissionRows.Close()
			return eventbiz.ContextLeaseTransactionState{}, invalidPersistedEventSemantic("Event Submission reference")
		}
		if status != eventbiz.StatusSuperseded {
			state.HasActiveSubmission = true
		}
	}
	if err := submissionRows.Err(); err != nil {
		submissionRows.Close()
		return eventbiz.ContextLeaseTransactionState{}, err
	}
	if err := submissionRows.Close(); err != nil {
		return eventbiz.ContextLeaseTransactionState{}, err
	}
	if request.SupersedesSubmissionID != "" {
		prior, err := loadSubmissionReference(ctx, t.tx, request.SupersedesSubmissionID)
		if err != nil {
			return eventbiz.ContextLeaseTransactionState{}, err
		}
		state.SupersededSubmission = prior
	}
	return state, nil
}

func (t *transaction) SaveContextLease(
	ctx context.Context,
	write eventbiz.ContextLeaseWrite,
) error {
	if len(write.ExpireLeaseIDs) > 0 {
		result, err := t.tx.ExecContext(ctx, `
UPDATE event_semantic_context_leases
SET status = 'expired'
WHERE id = ANY($1::text[]) AND status = 'active' AND lease_expires_at <= $2
`, write.ExpireLeaseIDs, write.TransitionedAt)
		if err != nil {
			return err
		}
		updated, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if updated != int64(len(write.ExpireLeaseIDs)) {
			return invalidPersistedEventSemantic("expired Context Lease command")
		}
	}
	if write.Refresh {
		var manifestPayload []byte
		if err := t.tx.QueryRowContext(ctx, `
SELECT context_manifest
FROM event_semantic_context_leases
WHERE id = $1
FOR UPDATE
`, write.Lease.ID).Scan(&manifestPayload); err != nil {
			return err
		}
		manifest, err := contextLeaseManifest(
			ctx, t.tx, write, manifestPayload,
		)
		if err != nil {
			return err
		}
		payload, err := json.Marshal(manifest)
		if err != nil {
			return err
		}
		_, err = t.tx.ExecContext(ctx, `
UPDATE event_semantic_context_leases
SET status = $2, lease_expires_at = $3, consumed_at = NULL,
    context_manifest = $4::jsonb
WHERE id = $1
`, write.Lease.ID, write.Lease.Status, write.Lease.LeaseExpiresAt, payload)
		return err
	}
	if write.ConsumeSupersededLease {
		if write.Lease.SupersedesSubmissionID == "" {
			return invalidPersistedEventSemantic("superseded Context Lease command")
		}
		if _, err := t.tx.ExecContext(ctx, `
UPDATE event_semantic_context_leases
SET status = 'consumed', consumed_at = $2
WHERE status = 'active'
  AND id = (
      SELECT context_lease_id
      FROM event_semantic_submissions
      WHERE id = $1
  )
`, write.Lease.SupersedesSubmissionID, write.TransitionedAt); err != nil {
			return err
		}
	}
	manifest, err := contextLeaseManifest(ctx, t.tx, write, nil)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	_, err = t.tx.ExecContext(ctx, `
INSERT INTO event_semantic_context_leases(
    id, event_id, supersedes_submission_id, agent_execution_id, worker_id,
    status, lease_expires_at, context_snapshot, context_manifest, leased_at
) VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, $7, NULL, $8, $9)
`, write.Lease.ID, write.Lease.EventID, write.Lease.SupersedesSubmissionID,
		write.AgentExecutionID, write.WorkerID, write.Lease.Status,
		write.Lease.LeaseExpiresAt, payload, write.TransitionedAt)
	return err
}

func contextLeaseManifest(
	ctx context.Context,
	tx *sql.Tx,
	write eventbiz.ContextLeaseWrite,
	existingPayload []byte,
) (eventbiz.ContextManifest, error) {
	if len(existingPayload) == 0 {
		return buildEventSemanticManifest(
			ctx, tx, write.Lease.ID, write.Lease.EventID,
			write.AgentExecutionID, write.WorkerID, write.Lease.LeaseExpiresAt,
		)
	}
	var manifest eventbiz.ContextManifest
	if err := json.Unmarshal(existingPayload, &manifest); err != nil {
		return eventbiz.ContextManifest{}, err
	}
	if err := validateEventSemanticManifestFingerprint(manifest); err != nil {
		return eventbiz.ContextManifest{}, err
	}
	manifest.LeaseStatus = write.Lease.Status
	manifest.LeaseExpiresAt = write.Lease.LeaseExpiresAt
	fingerprint, err := eventSemanticManifestFingerprint(manifest)
	if err != nil {
		return eventbiz.ContextManifest{}, err
	}
	manifest.ManifestFingerprint = fingerprint
	return manifest, nil
}

func (t *transaction) LoadSubmissionState(
	ctx context.Context,
	submission eventbiz.Submission,
) (eventbiz.SubmissionTransactionState, error) {
	var state eventbiz.SubmissionTransactionState
	if existing, found, err := existingEventSemanticSubmission(
		ctx, t.tx, submission.AgentExecutionID,
	); err != nil {
		return state, err
	} else if found {
		state.Existing = &existing
		return state, nil
	}

	var supersedes sql.NullString
	var manifestPayload []byte
	err := t.tx.QueryRowContext(ctx, `
SELECT event_id, agent_execution_id, status, lease_expires_at,
       supersedes_submission_id, context_manifest
FROM event_semantic_context_leases
WHERE id = $1 AND context_manifest IS NOT NULL
FOR UPDATE
`, submission.ContextLeaseID).Scan(
		&state.Lease.EventID, &state.Lease.AgentExecutionID,
		&state.Lease.Status, &state.Lease.LeaseExpiresAt, &supersedes, &manifestPayload,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	state.Lease.Found = true
	state.Lease.SupersedesSubmissionID = supersedes.String
	if err := validatePersistedSubmissionLeaseState(state.Lease); err != nil {
		return eventbiz.SubmissionTransactionState{}, err
	}

	var manifest eventbiz.ContextManifest
	if err := json.Unmarshal(manifestPayload, &manifest); err != nil {
		return eventbiz.SubmissionTransactionState{}, err
	}
	contextValue, err := eventSemanticContextFromManifest(ctx, t.tx, manifest)
	if err != nil {
		return eventbiz.SubmissionTransactionState{}, err
	}
	state.Context, err = hydrateEventSemanticSubmissionContext(
		ctx, t.tx, contextValue, submission, true,
	)
	if err != nil {
		return eventbiz.SubmissionTransactionState{}, err
	}
	if submission.SupersedesSubmissionID != "" {
		state.SupersededSubmission, err = loadSubmissionReference(
			ctx, t.tx, submission.SupersedesSubmissionID,
		)
		if err != nil {
			return eventbiz.SubmissionTransactionState{}, err
		}
	}
	return state, nil
}

func (t *transaction) SaveSubmission(
	ctx context.Context,
	write eventbiz.SubmissionWrite,
) error {
	decisionPayload, err := json.Marshal(write.Precheck)
	if err != nil {
		return err
	}
	counts, err := json.Marshal(map[string]int{
		"entity_links":     len(write.Submission.EntityLinks),
		"variable_signals": len(write.Submission.VariableSignals),
	})
	if err != nil {
		return err
	}
	_, err = t.tx.ExecContext(ctx, `
INSERT INTO event_semantic_submissions(
    id, context_lease_id, event_id, agent_execution_id, agent_key, agent_version,
    supersedes_submission_id, generator_prompt_hash, generator_model,
    reviewer_prompt_hash, reviewer_model, adjudicator_prompt_hash,
    adjudicator_model, ontology_version, acceptance_policy_key,
    acceptance_policy_version, canonical_payload_hash, status,
    candidate_counts, decision_summary, created_at
) VALUES (
    $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,
    'event-semantics.objective-v2',1,$15,$16,$17,$18,$19
)
`, write.SubmissionID, write.Submission.ContextLeaseID, write.Submission.EventID,
		write.Submission.AgentExecutionID, write.Submission.AgentKey, write.Submission.AgentVersion,
		nullString(write.Submission.SupersedesSubmissionID),
		write.Submission.GeneratorPromptHash, write.Submission.GeneratorModel,
		write.Submission.ReviewerPromptHash, write.Submission.ReviewerModel,
		nullString(write.Submission.AdjudicatorPromptHash),
		nullString(write.Submission.AdjudicatorModel),
		write.Submission.OntologyVersion, write.PayloadHash, write.Status, counts, decisionPayload,
		write.TransitionedAt,
	)
	if err != nil {
		return err
	}
	if _, err := t.tx.ExecContext(ctx, `
INSERT INTO event_semantic_candidate_snapshots(
    id, semantic_submission_id, payload, canonical_payload_hash, created_at
) VALUES ($1,$2,$3,$4,$5)
`, write.SnapshotID, write.SubmissionID, write.Payload, write.PayloadHash, write.TransitionedAt); err != nil {
		return err
	}
	if err := insertReviewableSemanticCandidates(
		ctx, t.tx, write.SubmissionID, write.Submission, write.Precheck, write.CandidateIDs,
	); err != nil {
		return err
	}
	if write.ConsumeLease {
		if _, err := t.tx.ExecContext(ctx, `
UPDATE event_semantic_context_leases
SET status = 'consumed', consumed_at = $2
WHERE id = $1
`, write.Submission.ContextLeaseID, write.TransitionedAt); err != nil {
			return err
		}
	}
	return nil
}

func (t *transaction) LoadReviewState(
	ctx context.Context,
	submission eventbiz.ReviewSubmission,
) (eventbiz.ReviewTransactionState, error) {
	var state eventbiz.ReviewTransactionState
	err := t.tx.QueryRowContext(ctx, `
SELECT agent_execution_id, reviewer_prompt_hash, reviewer_model,
       COALESCE(adjudicator_prompt_hash, ''), COALESCE(adjudicator_model, '')
FROM event_semantic_submissions
WHERE id = $1
FOR UPDATE
`, submission.SubmissionID).Scan(
		&state.Identity.AgentExecutionID,
		&state.Identity.ReviewerPromptHash,
		&state.Identity.ReviewerModel,
		&state.Identity.AdjudicatorPromptHash,
		&state.Identity.AdjudicatorModel,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	state.Found = true
	if err := validatePersistedReviewIdentity(state.Identity); err != nil {
		return eventbiz.ReviewTransactionState{}, err
	}

	var snapshot eventbiz.ReviewSnapshot
	var payload []byte
	err = t.tx.QueryRowContext(ctx, `
SELECT canonical_payload_hash, payload, created_at
FROM event_semantic_review_snapshots
WHERE semantic_submission_id = $1 AND reviewer_execution_key = $2
`, submission.SubmissionID, submission.ReviewerExecutionKey).Scan(
		&snapshot.CanonicalPayloadHash, &payload, &snapshot.CreatedAt,
	)
	if err == nil {
		snapshot.ReviewerExecutionKey = submission.ReviewerExecutionKey
		snapshot.Payload = append(json.RawMessage(nil), payload...)
		state.ExistingSnapshot = &snapshot
	} else if !errors.Is(err, sql.ErrNoRows) {
		return eventbiz.ReviewTransactionState{}, err
	}

	result, found, err := eventSemanticSubmissionByID(ctx, t.tx, submission.SubmissionID)
	if err != nil {
		return eventbiz.ReviewTransactionState{}, err
	}
	if !found {
		return state, nil
	}
	state.Submission = &result
	if state.ExistingSnapshot != nil {
		if err := validatePersistedReviewSnapshot(
			*state.ExistingSnapshot, submission.SubmissionID, result.Precheck,
		); err != nil {
			return eventbiz.ReviewTransactionState{}, err
		}
		return state, nil
	}
	if err := t.tx.QueryRowContext(ctx, `
SELECT
    (SELECT count(*) FROM event_semantic_review_snapshots WHERE semantic_submission_id = $1),
    policy.retry_budget
FROM event_semantic_submissions run
JOIN event_semantic_acceptance_policies policy
  ON policy.policy_key = run.acceptance_policy_key
 AND policy.version = run.acceptance_policy_version
WHERE run.id = $1
`, submission.SubmissionID).Scan(&state.ReviewCount, &state.RetryBudget); err != nil {
		return eventbiz.ReviewTransactionState{}, err
	}
	if state.ReviewCount < 0 || state.RetryBudget < 0 || state.RetryBudget > 3 || state.ReviewCount > state.RetryBudget+1 {
		return eventbiz.ReviewTransactionState{}, invalidPersistedEventSemantic("Review retry state")
	}
	return state, nil
}

func (t *transaction) SaveReview(
	ctx context.Context,
	write eventbiz.ReviewWrite,
) error {
	if _, err := t.tx.ExecContext(ctx, `
INSERT INTO event_semantic_review_snapshots(
    id, semantic_submission_id, reviewer_execution_key, payload,
    canonical_payload_hash, created_at
) VALUES ($1,$2,$3,$4,$5,$6)
`, write.SnapshotID, write.Submission.SubmissionID,
		write.Submission.ReviewerExecutionKey, write.Payload,
		write.PayloadHash, write.TransitionedAt); err != nil {
		return err
	}
	if write.SupersedePrior {
		if err := supersedePriorSemanticSubmission(
			ctx, t.tx, write.Submission.SubmissionID, write.TransitionedAt,
		); err != nil {
			return err
		}
	}
	if err := persistSemanticDecisions(
		ctx, t.tx, write.Submission.SubmissionID, write.Precheck, write.TransitionedAt,
	); err != nil {
		return err
	}
	decisionPayload, err := json.Marshal(write.Precheck)
	if err != nil {
		return err
	}
	if _, err := t.tx.ExecContext(ctx, `
UPDATE event_semantic_submissions
SET status = $2::varchar, decision_summary = $3, finalized_at = $4
WHERE id = $1
`, write.Submission.SubmissionID, write.Status, decisionPayload, write.FinalizedAt); err != nil {
		return err
	}
	if write.ConsumeLease {
		if _, err := t.tx.ExecContext(ctx, `
UPDATE event_semantic_context_leases
SET status = 'consumed', consumed_at = $2
WHERE id = (
    SELECT context_lease_id
    FROM event_semantic_submissions
    WHERE id = $1
)
`, write.Submission.SubmissionID, write.TransitionedAt); err != nil {
			return err
		}
	}
	return nil
}

func loadSubmissionReference(
	ctx context.Context,
	tx *sql.Tx,
	submissionID string,
) (*eventbiz.SubmissionReference, error) {
	var reference eventbiz.SubmissionReference
	err := tx.QueryRowContext(ctx, `
SELECT id::text, event_id::text, context_lease_id::text, status
FROM event_semantic_submissions
WHERE id = $1
FOR UPDATE
`, submissionID).Scan(
		&reference.SubmissionID, &reference.EventID,
		&reference.ContextLeaseID, &reference.Status,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := validatePersistedSubmissionReference(reference); err != nil {
		return nil, err
	}
	return &reference, nil
}

func supersedePriorSemanticSubmission(
	ctx context.Context,
	tx *sql.Tx,
	runID string,
	transitionedAt time.Time,
) error {
	for _, table := range []string{
		"event_entity_links", "variable_signals", "direct_impact_assertions",
	} {
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
SET review_status = 'superseded',
    reason_code = 'superseded_by_reanalysis',
    updated_at = $2
WHERE semantic_submission_id IN (SELECT id FROM ancestors)
  AND review_status <> 'superseded'
`, table)
		if _, err := tx.ExecContext(ctx, query, runID, transitionedAt); err != nil {
			return err
		}
	}
	_, err := tx.ExecContext(ctx, `
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
SET status = 'superseded', finalized_at = $2
WHERE id IN (SELECT id FROM ancestors) AND status <> 'superseded'
`, runID, transitionedAt)
	return err
}

func persistSemanticDecisions(
	ctx context.Context,
	tx *sql.Tx,
	runID string,
	precheck eventbiz.PrecheckResult,
	transitionedAt time.Time,
) error {
	for table, items := range map[string][]eventbiz.CandidateDecision{
		"event_entity_links":       precheck.EntityLinks,
		"variable_signals":         precheck.VariableSignals,
		"direct_impact_assertions": precheck.DirectImpacts,
	} {
		for _, item := range items {
			query := fmt.Sprintf(`
UPDATE %s
SET review_status = $3, reason_code = $4, updated_at = $5
WHERE semantic_submission_id = $1 AND candidate_key = $2
`, table)
			if _, err := tx.ExecContext(
				ctx, query, runID, item.CandidateKey,
				item.Status, nullString(item.ReasonCode), transitionedAt,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

var _ eventbiz.Transaction = (*transaction)(nil)
