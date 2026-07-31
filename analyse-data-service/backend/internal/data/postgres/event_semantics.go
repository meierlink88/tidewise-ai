package postgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventsemantics"
)

const (
	eventSemanticsOntologyVersion     = "event-semantics.phase-one@1"
	eventSemanticsPolicyVersion       = "event-semantics.phase-one@1"
	eventSemanticsManifestVersion     = "event-semantic-context-manifest.v1"
	eventSemanticsRouteVersion        = "event-semantic-anchor-routes.v1"
	eventSemanticsRoutePartitionLimit = 50
)

const eventSemanticInputEligibilitySQL = `
	e.event_status = 'confirmed'
	AND e.fact_status = 'verified'
	AND e.event_time IS NOT NULL
	AND EXISTS (
	    SELECT 1
	    FROM event_sources evidence
	    JOIN raw_documents document ON document.id = evidence.raw_document_id
	    WHERE evidence.event_id = e.id
	      AND COALESCE(evidence.evidence_hash, '') ~ '^[0-9a-f]{64}$'
	      AND COALESCE(btrim(evidence.evidence_excerpt), '') <> ''
	      AND evidence.source_level IN ('primary', 'secondary')
	      AND evidence.evidence_relation IN ('supports', 'contradicts', 'context')
	      AND COALESCE(evidence.supports_fields, ARRAY[]::text[])
	          <@ ARRAY['title', 'factual_summary', 'occurred_at', 'fact_payload']::text[]
	      AND array_position(evidence.supports_fields, NULL::text) IS NULL
	      AND (
	          evidence.evidence_relation = 'context'
	          OR COALESCE(cardinality(evidence.supports_fields), 0) > 0
	      )
	      AND COALESCE(btrim(document.source_name), '') <> ''
	      AND COALESCE(btrim(document.source_type), '') <> ''
	      AND COALESCE(btrim(document.title), '') <> ''
	      AND document.collected_at IS NOT NULL
	      AND evidence.created_at IS NOT NULL
	)
	AND NOT EXISTS (
	    SELECT 1
	    FROM event_sources evidence
	    JOIN raw_documents document ON document.id = evidence.raw_document_id
	    WHERE evidence.event_id = e.id
	      AND (
	          COALESCE(evidence.evidence_hash, '') !~ '^[0-9a-f]{64}$'
	          OR COALESCE(btrim(evidence.evidence_excerpt), '') = ''
	          OR COALESCE(evidence.source_level, '') NOT IN ('primary', 'secondary')
	          OR COALESCE(evidence.evidence_relation, '') NOT IN ('supports', 'contradicts', 'context')
	          OR NOT (
	              COALESCE(evidence.supports_fields, ARRAY[]::text[])
	              <@ ARRAY['title', 'factual_summary', 'occurred_at', 'fact_payload']::text[]
	          )
	          OR array_position(evidence.supports_fields, NULL::text) IS NOT NULL
	          OR (
	              evidence.evidence_relation <> 'context'
	              AND COALESCE(cardinality(evidence.supports_fields), 0) = 0
	          )
	          OR COALESCE(btrim(document.source_name), '') = ''
	          OR COALESCE(btrim(document.source_type), '') = ''
	          OR COALESCE(btrim(document.title), '') = ''
	          OR document.collected_at IS NULL
	          OR evidence.created_at IS NULL
	      )
	)
`

type semanticQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (r repository) ListEligibleEvents(
	ctx context.Context,
	limit int,
	after *eventsemantics.EligibleEventCursor,
) ([]eventsemantics.EligibleEvent, error) {
	var afterTime any
	var afterEventID any
	if after != nil {
		afterTime = after.FirstSeenAt.UTC()
		afterEventID = after.EventID
	}
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT e.id, e.first_seen_at
		FROM events e
		WHERE %s
		  AND NOT EXISTS (
		      SELECT 1 FROM event_semantic_context_leases lease
		      WHERE lease.event_id = e.id
		        AND lease.status = 'active'
		        AND lease.lease_expires_at > now()
		  )
		  AND NOT EXISTS (
		      SELECT 1 FROM event_semantic_submissions submission
		      WHERE submission.event_id = e.id
		        AND submission.status <> 'superseded'
		  )
		  AND (
		      $1::timestamptz IS NULL
		      OR (e.first_seen_at, e.id) > ($1::timestamptz, $2::uuid)
		  )
		ORDER BY e.first_seen_at, e.id
		LIMIT $3
	`, eventSemanticInputEligibilitySQL), afterTime, afterEventID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]eventsemantics.EligibleEvent, 0)
	for rows.Next() {
		var item eventsemantics.EligibleEvent
		if err := rows.Scan(&item.EventID, &item.FirstSeenAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r repository) CreateContextLease(
	ctx context.Context,
	request eventsemantics.ContextLeaseRequest,
) (eventsemantics.ContextLease, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return eventsemantics.ContextLease{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
		UPDATE event_semantic_context_leases
		SET status = 'expired'
		WHERE status = 'active' AND lease_expires_at <= now()
	`); err != nil {
		return eventsemantics.ContextLease{}, err
	}
	var existing eventsemantics.ContextLease
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
			return eventsemantics.ContextLease{}, &eventsemantics.ConflictError{
				Reason: "agent_execution_id is bound to a different Context Lease identity",
			}
		}
		if submissionStatus != "" && submissionStatus != string(eventsemantics.StatusPendingReview) &&
			submissionStatus != string(eventsemantics.StatusNeedsReanalysis) {
			return eventsemantics.ContextLease{}, &eventsemantics.ConflictError{
				Reason: "agent_execution_id already reached a terminal Semantic Submission",
			}
		}
		existing.EventID = existingEventID
		existing.SupersedesSubmissionID = existingSupersedes
		existing.Status = "active"
		existing.LeaseExpiresAt = time.Now().UTC().Add(request.Lease)
		var manifest eventsemantics.ContextManifest
		if len(existingManifest) == 0 {
			manifest, err = buildEventSemanticManifest(
				ctx, tx, existing.ID, existing.EventID, request.AgentExecutionID,
				request.WorkerID, existing.LeaseExpiresAt,
			)
			if err != nil {
				return eventsemantics.ContextLease{}, err
			}
		} else if err := json.Unmarshal(existingManifest, &manifest); err != nil {
			return eventsemantics.ContextLease{}, err
		} else if err := validateEventSemanticManifestFingerprint(manifest); err != nil {
			return eventsemantics.ContextLease{}, err
		}
		manifest.LeaseStatus = "active"
		manifest.LeaseExpiresAt = existing.LeaseExpiresAt
		manifest.ManifestFingerprint, err = eventSemanticManifestFingerprint(manifest)
		if err != nil {
			return eventsemantics.ContextLease{}, err
		}
		existingManifest, err = json.Marshal(manifest)
		if err != nil {
			return eventsemantics.ContextLease{}, err
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE event_semantic_context_leases
			SET status = 'active', lease_expires_at = $2, consumed_at = NULL,
			    context_manifest = $3::jsonb
			WHERE id = $1
		`, existing.ID, existing.LeaseExpiresAt, existingManifest); err != nil {
			return eventsemantics.ContextLease{}, err
		}
		if err := tx.Commit(); err != nil {
			return eventsemantics.ContextLease{}, err
		}
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return eventsemantics.ContextLease{}, err
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
			return eventsemantics.ContextLease{}, err
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
		return eventsemantics.ContextLease{}, classifyEventSemanticLeaseEligibility(
			ctx, tx, request.EventID,
		)
	}
	if err != nil {
		return eventsemantics.ContextLease{}, err
	}
	if request.SupersedesSubmissionID == "" {
		var exists bool
		if err := tx.QueryRowContext(ctx, `
			SELECT EXISTS (
			    SELECT 1 FROM event_semantic_submissions
			    WHERE event_id = $1 AND status <> 'superseded'
			)
		`, eventID).Scan(&exists); err != nil {
			return eventsemantics.ContextLease{}, err
		}
		if exists {
			return eventsemantics.ContextLease{}, &eventsemantics.ConflictError{
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
			return eventsemantics.ContextLease{}, &eventsemantics.NotFoundError{
				Resource: "superseded Event Semantic Submission",
			}
		} else if err != nil {
			return eventsemantics.ContextLease{}, err
		}
		if priorEventID != eventID ||
			(priorStatus != "needs_reanalysis" && priorStatus != "accepted" &&
				priorStatus != "rejected" && priorStatus != "quarantined") {
			return eventsemantics.ContextLease{}, &eventsemantics.ConflictError{
				Reason: "supersedes_submission_id is not an active terminal Submission for this Event",
			}
		}
	}
	contextLease := eventsemantics.ContextLease{
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
		return eventsemantics.ContextLease{}, err
	}
	manifestPayload, err := json.Marshal(manifest)
	if err != nil {
		return eventsemantics.ContextLease{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO event_semantic_context_leases(
		    id, event_id, supersedes_submission_id, agent_execution_id, worker_id,
		    status, lease_expires_at, context_snapshot, context_manifest
		) VALUES ($1, $2, NULLIF($3, '')::uuid, $4, $5, 'active', $6, NULL, $7)
	`, contextLease.ID, contextLease.EventID, contextLease.SupersedesSubmissionID,
		request.AgentExecutionID, request.WorkerID, contextLease.LeaseExpiresAt,
		manifestPayload); err != nil {
		return eventsemantics.ContextLease{}, err
	}
	if err := tx.Commit(); err != nil {
		return eventsemantics.ContextLease{}, err
	}
	return contextLease, nil
}

func classifyEventSemanticLeaseEligibility(
	ctx context.Context,
	tx *sql.Tx,
	eventID string,
) error {
	var status, factStatus string
	var hasEventTime, inputValid, activeLease bool
	err := tx.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT e.event_status, e.fact_status, e.event_time IS NOT NULL,
		       (%s),
		       EXISTS (
		           SELECT 1
		           FROM event_semantic_context_leases lease
		           WHERE lease.event_id = e.id AND lease.status = 'active'
		       )
		FROM events e
		WHERE e.id = $1
	`, eventSemanticInputEligibilitySQL), eventID).Scan(
		&status, &factStatus, &hasEventTime, &inputValid, &activeLease,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return &eventsemantics.NotFoundError{Resource: "Event"}
	}
	if err != nil {
		return err
	}
	if status != "confirmed" || factStatus != "verified" || !hasEventTime {
		return &eventsemantics.NotRequiredError{
			Reason: "Event no longer requires initial Semantic processing",
		}
	}
	if !inputValid {
		return &eventsemantics.InputInvalidError{
			Reason: "Event does not satisfy the Event Semantic input contract",
		}
	}
	if activeLease {
		return &eventsemantics.ConflictError{
			Reason: "Event already has an active Context Lease",
		}
	}
	return &eventsemantics.NotFoundError{Resource: "eligible Event"}
}

func (r repository) Context(ctx context.Context, contextLeaseID string) (eventsemantics.Context, error) {
	var payload []byte
	err := r.db.QueryRowContext(ctx, `
		SELECT context_manifest
		FROM event_semantic_context_leases
		WHERE id = $1 AND status = 'active' AND lease_expires_at > now()
		  AND context_manifest IS NOT NULL
	`, contextLeaseID).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return eventsemantics.Context{}, &eventsemantics.NotFoundError{Resource: "active Event Semantic Context Lease"}
	}
	if err != nil {
		return eventsemantics.Context{}, err
	}
	var manifest eventsemantics.ContextManifest
	if err := json.Unmarshal(payload, &manifest); err != nil {
		return eventsemantics.Context{}, err
	}
	return eventSemanticContextFromManifest(ctx, r.db, manifest)
}

func (r repository) SubmissionContext(
	ctx context.Context,
	contextLeaseID string,
	submission eventsemantics.Submission,
) (eventsemantics.Context, error) {
	result, err := r.Context(ctx, contextLeaseID)
	if err != nil {
		return eventsemantics.Context{}, err
	}
	return hydrateEventSemanticSubmissionContext(ctx, r.db, result, submission, false)
}

func hydrateEventSemanticSubmissionContext(
	ctx context.Context,
	query semanticQueryer,
	result eventsemantics.Context,
	submission eventsemantics.Submission,
	lockSelectedFacts bool,
) (eventsemantics.Context, error) {
	lockClause := ""
	if lockSelectedFacts {
		lockClause = " FOR SHARE"
	}
	entityIDs := make([]string, 0, len(submission.EntityLinks)+len(submission.DirectImpacts))
	for _, link := range submission.EntityLinks {
		entityIDs = append(entityIDs, link.EntityID)
	}
	for _, impact := range submission.DirectImpacts {
		entityIDs = append(entityIDs, impact.TargetEntityID)
	}
	if len(entityIDs) > 0 {
		rows, err := query.QueryContext(ctx, `
			SELECT id, entity_type, name, canonical_name, array_to_json(aliases), status
			FROM entity_nodes
			WHERE id = ANY($1::uuid[])
			ORDER BY entity_type, canonical_name, id
		`+lockClause, entityIDs)
		if err != nil {
			return eventsemantics.Context{}, err
		}
		for rows.Next() {
			var item eventsemantics.Entity
			var aliases []byte
			if err := rows.Scan(
				&item.ID, &item.Type, &item.Name, &item.CanonicalName, &aliases, &item.Status,
			); err != nil {
				rows.Close()
				return eventsemantics.Context{}, err
			}
			if err := json.Unmarshal(aliases, &item.Aliases); err != nil {
				rows.Close()
				return eventsemantics.Context{}, err
			}
			result.Entities = append(result.Entities, item)
		}
		if err := rows.Close(); err != nil {
			return eventsemantics.Context{}, err
		}
	}
	relationIDs := make([]string, 0, len(submission.DirectImpacts))
	for _, impact := range submission.DirectImpacts {
		if impact.EntityRelationID != "" {
			relationIDs = append(relationIDs, impact.EntityRelationID)
		}
	}
	if len(relationIDs) > 0 {
		rows, err := query.QueryContext(ctx, `
			SELECT id, from_entity_id, to_entity_id, relation_type, status
			FROM entity_edges
			WHERE id = ANY($1::uuid[])
			ORDER BY relation_type, id
		`+lockClause, relationIDs)
		if err != nil {
			return eventsemantics.Context{}, err
		}
		for rows.Next() {
			var item eventsemantics.EntityRelation
			if err := rows.Scan(
				&item.ID, &item.FromEntityID, &item.ToEntityID, &item.Type, &item.Status,
			); err != nil {
				rows.Close()
				return eventsemantics.Context{}, err
			}
			result.Relations = append(result.Relations, item)
		}
		if err := rows.Close(); err != nil {
			return eventsemantics.Context{}, err
		}
	}
	return result, nil
}

func buildEventSemanticContext(
	ctx context.Context,
	query semanticQueryer,
	contextLeaseID string,
	eventID string,
	agentExecutionID string,
	workerID string,
	leaseExpiresAt time.Time,
) (eventsemantics.Context, error) {
	result := eventsemantics.Context{
		ContextLeaseID: contextLeaseID, AgentExecutionID: agentExecutionID,
		WorkerID: workerID, LeaseExpiresAt: leaseExpiresAt.UTC(),
		ManifestContractVersion: eventSemanticsManifestVersion,
		OntologyVersion:         eventSemanticsOntologyVersion,
		PolicyVersion:           eventSemanticsPolicyVersion, RouteContractVersion: eventSemanticsRouteVersion,
	}
	err := query.QueryRowContext(ctx, `
		SELECT id, title, summary, event_time, event_status, fact_status
		FROM events WHERE id = $1
	`, eventID).Scan(
		&result.Event.ID, &result.Event.Title, &result.Event.Summary, &result.Event.OccurredAt,
		&result.Event.Status, &result.Event.FactStatus,
	)
	if err != nil {
		return eventsemantics.Context{}, err
	}
	if result.Evidence, err = eventSemanticEvidence(ctx, query, eventID, true, nil); err != nil {
		return eventsemantics.Context{}, err
	}
	if result.EntityTypes, err = eventSemanticEntityTypes(ctx, query, true, nil); err != nil {
		return eventsemantics.Context{}, err
	}
	if result.Variables, err = eventSemanticVariables(ctx, query, true, nil); err != nil {
		return eventsemantics.Context{}, err
	}
	if result.Rules, err = eventSemanticRules(ctx, query, true, nil); err != nil {
		return eventsemantics.Context{}, err
	}
	result.EventFingerprint, err = eventSemanticFingerprint(result.Event)
	if err != nil {
		return eventsemantics.Context{}, err
	}
	result.EvidenceFingerprint, err = eventSemanticFingerprint(result.Evidence)
	if err != nil {
		return eventsemantics.Context{}, err
	}
	stableContext := result
	stableContext.LeaseExpiresAt = time.Time{}
	stableContext.ContextFingerprint = ""
	result.ContextFingerprint, err = eventSemanticFingerprint(stableContext)
	if err != nil {
		return eventsemantics.Context{}, err
	}
	return result, nil
}

func buildEventSemanticManifest(
	ctx context.Context,
	query semanticQueryer,
	contextLeaseID string,
	eventID string,
	agentExecutionID string,
	workerID string,
	leaseExpiresAt time.Time,
) (eventsemantics.ContextManifest, error) {
	contextValue, err := buildEventSemanticContext(
		ctx, query, contextLeaseID, eventID, agentExecutionID, workerID, leaseExpiresAt,
	)
	if err != nil {
		return eventsemantics.ContextManifest{}, err
	}
	manifest := eventsemantics.ContextManifest{
		ContextLeaseID: contextLeaseID, AgentExecutionID: agentExecutionID, WorkerID: workerID,
		LeaseStatus: "active", LeaseExpiresAt: leaseExpiresAt.UTC(),
		ManifestContractVersion: eventSemanticsManifestVersion,
		ContextFingerprint:      contextValue.ContextFingerprint,
		EventID:                 eventID, EventFingerprint: contextValue.EventFingerprint,
		EvidenceFingerprint: contextValue.EvidenceFingerprint,
		OntologyVersion:     contextValue.OntologyVersion, PolicyVersion: contextValue.PolicyVersion,
		RouteContractVersion: contextValue.RouteContractVersion,
		Evidence:             make([]eventsemantics.EvidenceReference, 0, len(contextValue.Evidence)),
		EntityTypes:          make([]eventsemantics.VersionReference, 0, len(contextValue.EntityTypes)),
		Variables:            make([]eventsemantics.VersionReference, 0, len(contextValue.Variables)),
		Rules:                make([]eventsemantics.VersionReference, 0, len(contextValue.Rules)),
	}
	for _, evidence := range contextValue.Evidence {
		fingerprint, err := eventSemanticFingerprint(evidence)
		if err != nil {
			return eventsemantics.ContextManifest{}, err
		}
		manifest.Evidence = append(manifest.Evidence, eventsemantics.EvidenceReference{
			EvidenceID: evidence.ID, Fingerprint: fingerprint,
		})
	}
	for _, definition := range contextValue.EntityTypes {
		manifest.EntityTypes = append(manifest.EntityTypes, eventsemantics.VersionReference{Key: definition.TypeKey, Version: definition.Version})
	}
	for _, definition := range contextValue.Variables {
		manifest.Variables = append(manifest.Variables, eventsemantics.VersionReference{Key: definition.Key, Version: definition.Version})
	}
	for _, rule := range contextValue.Rules {
		manifest.Rules = append(manifest.Rules, eventsemantics.VersionReference{Key: rule.Key, Version: rule.Version})
	}
	manifest.ManifestFingerprint, err = eventSemanticManifestFingerprint(manifest)
	if err != nil {
		return eventsemantics.ContextManifest{}, err
	}
	return manifest, nil
}

func eventSemanticContextFromManifest(
	ctx context.Context,
	query semanticQueryer,
	manifest eventsemantics.ContextManifest,
) (eventsemantics.Context, error) {
	if err := validateEventSemanticManifestFingerprint(manifest); err != nil {
		return eventsemantics.Context{}, err
	}
	result := eventsemantics.Context{
		ContextLeaseID: manifest.ContextLeaseID, AgentExecutionID: manifest.AgentExecutionID,
		WorkerID: manifest.WorkerID, LeaseExpiresAt: manifest.LeaseExpiresAt,
		ManifestContractVersion: manifest.ManifestContractVersion,
		ContextFingerprint:      manifest.ContextFingerprint, EventFingerprint: manifest.EventFingerprint,
		EvidenceFingerprint: manifest.EvidenceFingerprint, OntologyVersion: manifest.OntologyVersion,
		PolicyVersion: manifest.PolicyVersion, RouteContractVersion: manifest.RouteContractVersion,
	}
	if err := query.QueryRowContext(ctx, `
		SELECT id, title, summary, event_time, event_status, fact_status
		FROM events WHERE id = $1
	`, manifest.EventID).Scan(
		&result.Event.ID, &result.Event.Title, &result.Event.Summary, &result.Event.OccurredAt,
		&result.Event.Status, &result.Event.FactStatus,
	); err != nil {
		return eventsemantics.Context{}, err
	}
	evidenceIDs := make([]string, 0, len(manifest.Evidence))
	for _, reference := range manifest.Evidence {
		evidenceIDs = append(evidenceIDs, reference.EvidenceID)
	}
	allEvidence, err := eventSemanticEvidence(ctx, query, manifest.EventID, false, evidenceIDs)
	if err != nil {
		return eventsemantics.Context{}, err
	}
	evidenceByID := make(map[string]eventsemantics.Evidence, len(allEvidence))
	for _, evidence := range allEvidence {
		evidenceByID[evidence.ID] = evidence
	}
	result.Evidence = make([]eventsemantics.Evidence, 0, len(manifest.Evidence))
	for _, reference := range manifest.Evidence {
		evidence, ok := evidenceByID[reference.EvidenceID]
		if !ok {
			return eventsemantics.Context{}, &eventsemantics.ContextDriftError{Reason: "pinned Event Evidence is unavailable"}
		}
		current, err := eventSemanticFingerprint(evidence)
		if err != nil {
			return eventsemantics.Context{}, err
		}
		if current != reference.Fingerprint {
			return eventsemantics.Context{}, &eventsemantics.ContextDriftError{Reason: "pinned Event Evidence changed"}
		}
		result.Evidence = append(result.Evidence, evidence)
	}
	if result.EntityTypes, err = eventSemanticEntityTypes(ctx, query, false, manifest.EntityTypes); err != nil {
		return eventsemantics.Context{}, err
	}
	result.EntityTypes, err = selectEntityTypeReferences(result.EntityTypes, manifest.EntityTypes)
	if err != nil {
		return eventsemantics.Context{}, err
	}
	if result.Variables, err = eventSemanticVariables(ctx, query, false, manifest.Variables); err != nil {
		return eventsemantics.Context{}, err
	}
	result.Variables, err = selectVariableReferences(result.Variables, manifest.Variables)
	if err != nil {
		return eventsemantics.Context{}, err
	}
	if result.Rules, err = eventSemanticRules(ctx, query, false, manifest.Rules); err != nil {
		return eventsemantics.Context{}, err
	}
	result.Rules, err = selectRuleReferences(result.Rules, manifest.Rules)
	if err != nil {
		return eventsemantics.Context{}, err
	}
	eventFingerprint, err := eventSemanticFingerprint(result.Event)
	if err != nil {
		return eventsemantics.Context{}, err
	}
	evidenceFingerprint, err := eventSemanticFingerprint(result.Evidence)
	if err != nil {
		return eventsemantics.Context{}, err
	}
	stableContext := result
	stableContext.LeaseExpiresAt = time.Time{}
	stableContext.ContextFingerprint = ""
	contextFingerprint, err := eventSemanticFingerprint(stableContext)
	if err != nil {
		return eventsemantics.Context{}, err
	}
	if eventFingerprint != manifest.EventFingerprint || evidenceFingerprint != manifest.EvidenceFingerprint ||
		contextFingerprint != manifest.ContextFingerprint {
		return eventsemantics.Context{}, &eventsemantics.ContextDriftError{Reason: "pinned Event Semantic Context changed"}
	}
	return result, nil
}

func eventSemanticManifestFingerprint(manifest eventsemantics.ContextManifest) (string, error) {
	manifest.ManifestFingerprint = ""
	return eventSemanticFingerprint(manifest)
}

func validateEventSemanticManifestFingerprint(manifest eventsemantics.ContextManifest) error {
	fingerprint, err := eventSemanticManifestFingerprint(manifest)
	if err != nil {
		return err
	}
	if manifest.ManifestContractVersion != eventSemanticsManifestVersion ||
		manifest.ManifestFingerprint == "" || fingerprint != manifest.ManifestFingerprint {
		return &eventsemantics.ContextDriftError{Reason: "Event Semantic Context Manifest identity changed"}
	}
	return nil
}

func selectEntityTypeReferences(values []eventsemantics.EntityTypeDefinition, references []eventsemantics.VersionReference) ([]eventsemantics.EntityTypeDefinition, error) {
	selected := make([]eventsemantics.EntityTypeDefinition, 0, len(references))
	for _, reference := range references {
		found := false
		for _, value := range values {
			if value.TypeKey == reference.Key && value.Version == reference.Version {
				selected = append(selected, value)
				found = true
				break
			}
		}
		if !found {
			return nil, &eventsemantics.ContextDriftError{Reason: "pinned Entity Type Definition is unavailable"}
		}
	}
	return selected, nil
}

func selectVariableReferences(values []eventsemantics.VariableDefinition, references []eventsemantics.VersionReference) ([]eventsemantics.VariableDefinition, error) {
	selected := make([]eventsemantics.VariableDefinition, 0, len(references))
	for _, reference := range references {
		found := false
		for _, value := range values {
			if value.Key == reference.Key && value.Version == reference.Version {
				selected = append(selected, value)
				found = true
				break
			}
		}
		if !found {
			return nil, &eventsemantics.ContextDriftError{Reason: "pinned Variable Definition is unavailable"}
		}
	}
	return selected, nil
}

func selectRuleReferences(values []eventsemantics.DirectTransmissionRule, references []eventsemantics.VersionReference) ([]eventsemantics.DirectTransmissionRule, error) {
	selected := make([]eventsemantics.DirectTransmissionRule, 0, len(references))
	for _, reference := range references {
		found := false
		for _, value := range values {
			if value.Key == reference.Key && value.Version == reference.Version {
				selected = append(selected, value)
				found = true
				break
			}
		}
		if !found {
			return nil, &eventsemantics.ContextDriftError{Reason: "pinned Direct Transmission Rule is unavailable"}
		}
	}
	return selected, nil
}

func eventSemanticFingerprint(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(payload)), nil
}

func eventSemanticEntityTypes(
	ctx context.Context,
	query semanticQueryer,
	includeAll bool,
	references []eventsemantics.VersionReference,
) ([]eventsemantics.EntityTypeDefinition, error) {
	keys, versions := semanticVersionReferenceArrays(references)
	rows, err := query.QueryContext(ctx, `
		SELECT type_key, version, signal_subject_allowed, direct_target_mode, status
		FROM entity_type_definitions
		WHERE status = 'active'
		  AND ($1 OR EXISTS (
		    SELECT 1 FROM unnest($2::text[], $3::integer[]) requested(key, version)
		    WHERE requested.key = type_key AND requested.version = entity_type_definitions.version
		  ))
		ORDER BY type_key, version
	`, includeAll, keys, versions)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []eventsemantics.EntityTypeDefinition
	for rows.Next() {
		var item eventsemantics.EntityTypeDefinition
		if err := rows.Scan(
			&item.TypeKey, &item.Version, &item.SignalSubjectAllowed,
			&item.DirectTargetMode, &item.Status,
		); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func eventSemanticEvidence(
	ctx context.Context,
	query semanticQueryer,
	eventID string,
	includeAll bool,
	evidenceIDs []string,
) ([]eventsemantics.Evidence, error) {
	rows, err := query.QueryContext(ctx, `
		SELECT es.id, es.evidence_hash, es.evidence_excerpt, es.source_level,
		       es.evidence_relation, array_to_json(es.supports_fields),
		       COALESCE(es.is_primary, false), es.raw_document_id,
		       rd.source_name, rd.source_type, rd.source_url, rd.title,
		       rd.published_at, rd.collected_at,
		       GREATEST(COALESCE(rd.published_at, rd.collected_at), rd.collected_at),
		       es.created_at,
		       COALESCE(NULLIF(event.fact_payload ->> 'statement_source', ''), '')
		FROM event_sources es
		JOIN raw_documents rd ON rd.id = es.raw_document_id
		JOIN events event ON event.id = es.event_id
		WHERE es.event_id = $1 AND ($2 OR es.id = ANY($3::uuid[]))
		ORDER BY COALESCE(es.is_primary, false) DESC, es.id
	`, eventID, includeAll, evidenceIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []eventsemantics.Evidence
	for rows.Next() {
		var item eventsemantics.Evidence
		var supported []byte
		if err := rows.Scan(
			&item.ID, &item.Hash, &item.Excerpt, &item.SourceLevel, &item.Relation,
			&supported, &item.IsPrimary, &item.RawDocumentID, &item.SourceName,
			&item.SourceType, &item.SourceURL, &item.Title, &item.PublishedAt,
			&item.FirstSeenAt, &item.KnowledgeAvailableAt, &item.AcceptedAt,
			&item.StatementSource,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(supported, &item.SupportsFields); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func eventSemanticEntities(ctx context.Context, query semanticQueryer) ([]eventsemantics.Entity, error) {
	rows, err := query.QueryContext(ctx, `
		SELECT id, entity_type, name, canonical_name, array_to_json(aliases), status
		FROM entity_nodes
		WHERE status = 'active'
		  AND entity_type IN (
		      SELECT type_key FROM entity_type_definitions
		      WHERE version = 1 AND status = 'active'
		  )
		ORDER BY entity_type, canonical_name, id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []eventsemantics.Entity
	for rows.Next() {
		var item eventsemantics.Entity
		var aliases []byte
		if err := rows.Scan(&item.ID, &item.Type, &item.Name, &item.CanonicalName, &aliases, &item.Status); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(aliases, &item.Aliases); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func eventSemanticRelations(ctx context.Context, query semanticQueryer) ([]eventsemantics.EntityRelation, error) {
	rows, err := query.QueryContext(ctx, `
		SELECT edge.id, edge.from_entity_id, edge.to_entity_id, edge.relation_type, edge.status
		FROM entity_edges edge
		JOIN entity_nodes source ON source.id = edge.from_entity_id AND source.status = 'active'
		JOIN entity_nodes target ON target.id = edge.to_entity_id AND target.status = 'active'
		WHERE edge.status = 'active'
		ORDER BY edge.relation_type, edge.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []eventsemantics.EntityRelation
	for rows.Next() {
		var item eventsemantics.EntityRelation
		if err := rows.Scan(&item.ID, &item.FromEntityID, &item.ToEntityID, &item.Type, &item.Status); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func eventSemanticVariables(
	ctx context.Context,
	query semanticQueryer,
	includeAll bool,
	references []eventsemantics.VersionReference,
) ([]eventsemantics.VariableDefinition, error) {
	keys, versions := semanticVersionReferenceArrays(references)
	rows, err := query.QueryContext(ctx, `
		SELECT definition.variable_key, definition.version, definition.name_zh, definition.name_en,
		       definition.domain, definition.value_type, definition.status,
		       array_to_json(definition.allowed_directions),
		       array_to_json(array_agg(applicable.entity_type ORDER BY applicable.entity_type))
		FROM variable_definitions definition
		JOIN variable_definition_entity_types applicable
		  ON applicable.variable_key = definition.variable_key
		 AND applicable.variable_version = definition.version
		WHERE definition.status = 'active'
		  AND ($1 OR EXISTS (
		    SELECT 1 FROM unnest($2::text[], $3::integer[]) requested(key, version)
		    WHERE requested.key = definition.variable_key AND requested.version = definition.version
		  ))
		GROUP BY definition.variable_key, definition.version
		ORDER BY definition.variable_key, definition.version
	`, includeAll, keys, versions)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []eventsemantics.VariableDefinition
	for rows.Next() {
		var item eventsemantics.VariableDefinition
		var directions, applicable []byte
		if err := rows.Scan(
			&item.Key, &item.Version, &item.NameZH, &item.NameEN, &item.Domain, &item.ValueType,
			&item.Status, &directions, &applicable,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(directions, &item.AllowedDirections); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(applicable, &item.ApplicableEntityTypes); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func eventSemanticRules(
	ctx context.Context,
	query semanticQueryer,
	includeAll bool,
	references []eventsemantics.VersionReference,
) ([]eventsemantics.DirectTransmissionRule, error) {
	keys, versions := semanticVersionReferenceArrays(references)
	rows, err := query.QueryContext(ctx, `
		SELECT rule_key, version, status, source_entity_type, source_variable_key,
		       source_variable_version, source_direction, relation_type, target_entity_type,
		       affected_variable_key, affected_variable_version, affected_direction,
		       condition_summary, mechanism_template
		FROM direct_transmission_rules
		WHERE status = 'approved'
		  AND ($1 OR EXISTS (
		    SELECT 1 FROM unnest($2::text[], $3::integer[]) requested(key, version)
		    WHERE requested.key = rule_key AND requested.version = direct_transmission_rules.version
		  ))
		ORDER BY rule_key, version
	`, includeAll, keys, versions)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []eventsemantics.DirectTransmissionRule
	for rows.Next() {
		var item eventsemantics.DirectTransmissionRule
		if err := rows.Scan(
			&item.Key, &item.Version, &item.Status, &item.SourceEntityType, &item.SourceVariableKey,
			&item.SourceVariableVersion, &item.SourceDirection, &item.RelationType,
			&item.TargetEntityType, &item.AffectedVariableKey, &item.AffectedVariableVersion,
			&item.AffectedDirection, &item.ConditionSummary, &item.MechanismTemplate,
		); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func semanticVersionReferenceArrays(
	references []eventsemantics.VersionReference,
) ([]string, []int32) {
	keys := make([]string, 0, len(references))
	versions := make([]int32, 0, len(references))
	for _, reference := range references {
		keys = append(keys, reference.Key)
		versions = append(versions, int32(reference.Version))
	}
	return keys, versions
}

func (r repository) Resolve(
	ctx context.Context,
	contextLeaseID string,
	mentions []eventsemantics.EntityMention,
) ([]eventsemantics.EntityResolution, error) {
	if _, err := r.Context(ctx, contextLeaseID); err != nil {
		return nil, err
	}
	result := make([]eventsemantics.EntityResolution, 0, len(mentions))
	for _, mention := range mentions {
		resolution := eventsemantics.EntityResolution{Mention: mention.Mention}
		rows, err := r.db.QueryContext(ctx, `
			SELECT id, entity_type, name, canonical_name, array_to_json(aliases), status
			FROM entity_nodes
			WHERE status = 'active'
			  AND entity_type = ANY($1)
			  AND (lower(name) = lower($2) OR lower(canonical_name) = lower($2)
			       OR EXISTS (SELECT 1 FROM unnest(aliases) alias WHERE lower(alias) = lower($2)))
			ORDER BY (lower(canonical_name) = lower($2)) DESC, canonical_name, id
			LIMIT 10
		`, mention.AllowedEntityTypes, mention.Mention)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var entity eventsemantics.Entity
			var aliases []byte
			if err := rows.Scan(
				&entity.ID, &entity.Type, &entity.Name, &entity.CanonicalName,
				&aliases, &entity.Status,
			); err != nil {
				rows.Close()
				return nil, err
			}
			if err := json.Unmarshal(aliases, &entity.Aliases); err != nil {
				rows.Close()
				return nil, err
			}
			resolution.Candidates = append(resolution.Candidates, entity)
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
		resolution.Ambiguous = len(resolution.Candidates) > 1
		result = append(result, resolution)
	}
	return result, nil
}

func (r repository) SearchDirectTargets(
	ctx context.Context,
	contextLeaseID string,
	subjectEntityID string,
	allowedTargetTypes []string,
) ([]eventsemantics.DirectTarget, error) {
	if _, err := r.Context(ctx, contextLeaseID); err != nil {
		return nil, err
	}
	var result []eventsemantics.DirectTarget
	rows, err := r.db.QueryContext(ctx, `
		SELECT target.id, target.entity_type, target.name, target.canonical_name,
		       array_to_json(target.aliases), target.status,
		       edge.id, edge.from_entity_id, edge.to_entity_id, edge.relation_type, edge.status
		FROM entity_edges edge
		JOIN entity_nodes target ON target.id = edge.to_entity_id AND target.status = 'active'
		WHERE edge.from_entity_id = $1 AND edge.status = 'active'
		  AND target.entity_type = ANY($2)
		ORDER BY edge.relation_type, target.canonical_name, target.id
		LIMIT 50
	`, subjectEntityID, allowedTargetTypes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item eventsemantics.DirectTarget
		var aliases []byte
		if err := rows.Scan(
			&item.Entity.ID, &item.Entity.Type, &item.Entity.Name, &item.Entity.CanonicalName,
			&aliases, &item.Entity.Status, &item.Relation.ID, &item.Relation.FromEntityID,
			&item.Relation.ToEntityID, &item.Relation.Type, &item.Relation.Status,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(aliases, &item.Entity.Aliases); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r repository) ListResolutionRoutes(
	ctx context.Context,
	contextLeaseID string,
	targetEntityType string,
) ([]eventsemantics.ResolutionRoute, error) {
	if _, err := r.Context(ctx, contextLeaseID); err != nil {
		return nil, err
	}
	if targetEntityType != "chain_node" {
		return nil, &eventsemantics.ValidationError{Reason: "target Entity Type has no bounded resolution route"}
	}
	industryPartitions, industryLabels, err := r.eventSemanticIndustryPartitions(ctx)
	if err != nil {
		return nil, err
	}
	conceptPartitions, conceptLabels, err := r.eventSemanticConceptPartitions(ctx)
	if err != nil {
		return nil, err
	}
	var routes []eventsemantics.ResolutionRoute
	if len(industryPartitions) > 0 {
		routes = append(routes, eventsemantics.ResolutionRoute{
			ID: "chain-node-via-industry.v1", ContractVersion: eventSemanticsRouteVersion,
			TargetEntityType: "chain_node", AnchorEntityType: "industry",
			MappingRelationType: "mapped_to_industry", Partitions: industryPartitions,
			PartitionLabels: industryLabels,
			Direction:       "industry_to_industry_chain_to_chain_node",
			Purpose:         "Resolve a formal ChainNode through an approved Industry anchor",
			NextOperation:   "list_resolution_anchors", OrderingContract: "canonical_name_entity_id.v1",
		})
	}
	if len(conceptPartitions) > 0 {
		routes = append(routes, eventsemantics.ResolutionRoute{
			ID: "chain-node-via-concept.v1", ContractVersion: eventSemanticsRouteVersion,
			TargetEntityType: "chain_node", AnchorEntityType: "concept",
			MappingRelationType: "mapped_to_concept", Partitions: conceptPartitions,
			PartitionLabels: conceptLabels,
			Direction:       "concept_to_industry_chain_to_chain_node",
			Purpose:         "Resolve a formal ChainNode through an approved Concept anchor",
			NextOperation:   "list_resolution_anchors", OrderingContract: "canonical_name_entity_id.v1",
		})
	}
	return routes, nil
}

func (r repository) eventSemanticIndustryPartitions(ctx context.Context) ([]string, map[string]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT profile.entity_id::text, entity.name
		FROM industry_profiles profile
		JOIN entity_nodes entity ON entity.id = profile.entity_id AND entity.status = 'active'
		WHERE profile.review_status = 'approved' AND profile.classification_level = 1
		ORDER BY entity.canonical_name, profile.entity_id
		LIMIT $1
	`, eventSemanticsRoutePartitionLimit)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var result []string
	labels := make(map[string]string)
	for rows.Next() {
		var partition, label string
		if err := rows.Scan(&partition, &label); err != nil {
			return nil, nil, err
		}
		result = append(result, partition)
		labels[partition] = label
	}
	return result, labels, rows.Err()
}

func (r repository) eventSemanticConceptPartitions(ctx context.Context) ([]string, map[string]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT concept_type
		FROM concept_profiles profile
		JOIN entity_nodes entity ON entity.id = profile.entity_id AND entity.status = 'active'
		WHERE profile.review_status = 'approved'
		ORDER BY concept_type
		LIMIT $1
	`, eventSemanticsRoutePartitionLimit)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var result []string
	labels := make(map[string]string)
	for rows.Next() {
		var partition string
		if err := rows.Scan(&partition); err != nil {
			return nil, nil, err
		}
		result = append(result, partition)
		labels[partition] = strings.ReplaceAll(partition, "_", " ")
	}
	return result, labels, rows.Err()
}

func (r repository) ListResolutionAnchors(
	ctx context.Context,
	contextLeaseID string,
	routeID string,
	partition string,
	parentAnchorIDs []string,
	limit int,
	after *eventsemantics.ResolutionKeyset,
) ([]eventsemantics.ResolutionAnchor, error) {
	if _, err := r.Context(ctx, contextLeaseID); err != nil {
		return nil, err
	}
	if parentAnchorIDs == nil {
		parentAnchorIDs = []string{}
	}
	var afterName, afterID any
	if after != nil {
		afterName, afterID = after.CanonicalName, after.EntityID
	}
	var rows *sql.Rows
	var err error
	switch routeID {
	case "chain-node-via-industry.v1":
		var validPartition bool
		if err := r.db.QueryRowContext(ctx, `
			SELECT EXISTS (
			  SELECT 1 FROM industry_profiles profile
			  JOIN entity_nodes entity ON entity.id = profile.entity_id AND entity.status = 'active'
			  WHERE profile.entity_id = $1::uuid AND profile.classification_level = 1
			    AND profile.review_status = 'approved'
			)
		`, partition).Scan(&validPartition); err != nil {
			return nil, err
		}
		if !validPartition {
			return nil, &eventsemantics.ValidationError{Reason: "Industry partition is unknown, inactive or unapproved"}
		}
		var validParentCount int
		if len(parentAnchorIDs) > 0 {
			if err := r.db.QueryRowContext(ctx, `
				SELECT count(DISTINCT child.entity_id)
				FROM industry_profiles root
				JOIN industry_profiles child
				  ON child.entity_id = ANY($2::uuid[]) AND child.review_status = 'approved'
				JOIN entity_nodes entity ON entity.id = child.entity_id AND entity.status = 'active'
				WHERE root.entity_id = $1::uuid AND root.classification_level = 1
				  AND root.review_status = 'approved'
				  AND child.hierarchy_path_codes[1] = root.industry_code
			`, partition, parentAnchorIDs).Scan(&validParentCount); err != nil {
				return nil, err
			}
			if validParentCount != len(parentAnchorIDs) {
				return nil, &eventsemantics.ValidationError{Reason: "parent_anchor_ids contain an unknown or out-of-partition Industry"}
			}
		}
		rows, err = r.db.QueryContext(ctx, `
			SELECT entity.id, entity.entity_type, entity.name, entity.canonical_name,
			       array_to_json(entity.aliases), entity.status, $1::text,
			       profile.definition,
			       profile.classification_system || ':' || profile.classification_version || ':' ||
			         array_to_string(profile.hierarchy_path_codes, '/')
			FROM industry_profiles profile
			JOIN entity_nodes entity ON entity.id = profile.entity_id AND entity.status = 'active'
			JOIN industry_profiles root
			  ON root.entity_id = $1::uuid AND root.classification_level = 1
			 AND root.review_status = 'approved'
			WHERE profile.review_status = 'approved'
			  AND profile.hierarchy_path_codes[1] = root.industry_code
			  AND (
			    cardinality($2::uuid[]) = 0 OR EXISTS (
			      SELECT 1 FROM industry_profiles parent
			      WHERE parent.entity_id = ANY($2::uuid[]) AND parent.review_status = 'approved'
			        AND profile.hierarchy_path_codes[1:cardinality(parent.hierarchy_path_codes)] =
			            parent.hierarchy_path_codes
			    )
			  )
			  AND EXISTS (
			    SELECT 1
			    FROM entity_edges mapping
			    JOIN entity_nodes chain ON chain.id = mapping.from_entity_id AND chain.status = 'active'
			    JOIN industry_chain_definitions definition
			      ON definition.entity_id = chain.id AND definition.review_status = 'approved'
			    JOIN industry_chain_node_memberships membership
			      ON membership.industry_chain_entity_id = chain.id
			     AND membership.status = 'active' AND membership.review_status = 'approved'
			    JOIN chain_node_profiles node_profile
			      ON node_profile.entity_id = membership.chain_node_entity_id
			     AND node_profile.review_status = 'approved'
			    JOIN entity_nodes node
			      ON node.id = membership.chain_node_entity_id AND node.status = 'active'
			    WHERE mapping.to_entity_id = profile.entity_id
			      AND mapping.relation_type = 'mapped_to_industry' AND mapping.status = 'active'
			  )
			  AND ($3::text IS NULL OR (entity.canonical_name, entity.id) > ($3::text, $4::uuid))
			ORDER BY entity.canonical_name, entity.id
			LIMIT $5
		`, partition, parentAnchorIDs, afterName, afterID, limit)
	case "chain-node-via-concept.v1":
		var validPartition bool
		if err := r.db.QueryRowContext(ctx, `
			SELECT EXISTS (
			  SELECT 1 FROM concept_profiles profile
			  JOIN entity_nodes entity ON entity.id = profile.entity_id AND entity.status = 'active'
			  WHERE profile.concept_type = $1 AND profile.review_status = 'approved'
			)
		`, partition).Scan(&validPartition); err != nil {
			return nil, err
		}
		if !validPartition {
			return nil, &eventsemantics.ValidationError{Reason: "Concept partition is unknown, inactive or unapproved"}
		}
		rows, err = r.db.QueryContext(ctx, `
			SELECT entity.id, entity.entity_type, entity.name, entity.canonical_name,
			       array_to_json(entity.aliases), entity.status, profile.concept_type,
			       profile.definition, profile.concept_type || ':' || entity.id::text
			FROM concept_profiles profile
			JOIN entity_nodes entity ON entity.id = profile.entity_id AND entity.status = 'active'
			WHERE profile.review_status = 'approved' AND profile.concept_type = $1
			  AND EXISTS (
			    SELECT 1
			    FROM entity_edges mapping
			    JOIN entity_nodes chain ON chain.id = mapping.from_entity_id AND chain.status = 'active'
			    JOIN industry_chain_definitions definition
			      ON definition.entity_id = chain.id AND definition.review_status = 'approved'
			    JOIN industry_chain_node_memberships membership
			      ON membership.industry_chain_entity_id = chain.id
			     AND membership.status = 'active' AND membership.review_status = 'approved'
			    JOIN chain_node_profiles node_profile
			      ON node_profile.entity_id = membership.chain_node_entity_id
			     AND node_profile.review_status = 'approved'
			    JOIN entity_nodes node
			      ON node.id = membership.chain_node_entity_id AND node.status = 'active'
			    WHERE mapping.to_entity_id = profile.entity_id
			      AND mapping.relation_type = 'mapped_to_concept' AND mapping.status = 'active'
			  )
			  AND ($2::text IS NULL OR (entity.canonical_name, entity.id) > ($2::text, $3::uuid))
			ORDER BY entity.canonical_name, entity.id
			LIMIT $4
		`, partition, afterName, afterID, limit)
	default:
		return nil, &eventsemantics.ValidationError{Reason: "route_id is not supported"}
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]eventsemantics.ResolutionAnchor, 0)
	for rows.Next() {
		var item eventsemantics.ResolutionAnchor
		var aliases []byte
		if err := rows.Scan(
			&item.Entity.ID, &item.Entity.Type, &item.Entity.Name, &item.Entity.CanonicalName,
			&aliases, &item.Entity.Status, &item.Partition, &item.Description, &item.HierarchyIdentity,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(aliases, &item.Entity.Aliases); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r repository) ResolveChainNodeCandidates(
	ctx context.Context,
	contextLeaseID string,
	routeID string,
	anchorEntityIDs []string,
	limit int,
	after *eventsemantics.ResolutionKeyset,
) ([]eventsemantics.ResolutionCandidate, error) {
	manifest, err := r.Context(ctx, contextLeaseID)
	if err != nil {
		return nil, err
	}
	relationType, anchorEntityType := "", ""
	switch routeID {
	case "chain-node-via-industry.v1":
		relationType, anchorEntityType = "mapped_to_industry", "industry"
	case "chain-node-via-concept.v1":
		relationType, anchorEntityType = "mapped_to_concept", "concept"
	default:
		return nil, &eventsemantics.ValidationError{Reason: "route_id is not supported"}
	}
	var validAnchorCount int
	if err := r.db.QueryRowContext(ctx, `
		SELECT count(DISTINCT anchor.id)
		FROM entity_nodes anchor
		WHERE anchor.id = ANY($1::uuid[]) AND anchor.status = 'active' AND anchor.entity_type = $2
		  AND (
		    ($2 = 'industry' AND EXISTS (
		      SELECT 1 FROM industry_profiles profile
		      WHERE profile.entity_id = anchor.id AND profile.review_status = 'approved'
		    ))
		    OR ($2 = 'concept' AND EXISTS (
		      SELECT 1 FROM concept_profiles profile
		      WHERE profile.entity_id = anchor.id AND profile.review_status = 'approved'
		    ))
		  )
	`, anchorEntityIDs, anchorEntityType).Scan(&validAnchorCount); err != nil {
		return nil, err
	}
	if validAnchorCount != len(anchorEntityIDs) {
		return nil, &eventsemantics.ValidationError{Reason: "anchor_entity_ids contain an unknown, inactive, wrong-type or unapproved anchor"}
	}
	var afterName, afterID any
	if after != nil {
		afterName, afterID = after.CanonicalName, after.EntityID
	}
	rows, err := r.db.QueryContext(ctx, `
		WITH target_page AS (
		  SELECT node.id AS target_id, node.canonical_name, node_profile.definition
		  FROM entity_edges mapping
		  JOIN entity_nodes anchor ON anchor.id = mapping.to_entity_id AND anchor.status = 'active'
		  JOIN entity_nodes chain ON chain.id = mapping.from_entity_id AND chain.status = 'active'
		  JOIN industry_chain_definitions definition
		    ON definition.entity_id = chain.id AND definition.review_status = 'approved'
		  JOIN industry_chain_node_memberships membership
		    ON membership.industry_chain_entity_id = chain.id
		   AND membership.status = 'active' AND membership.review_status = 'approved'
		  JOIN chain_node_profiles node_profile
		    ON node_profile.entity_id = membership.chain_node_entity_id
		   AND node_profile.review_status = 'approved'
		  JOIN entity_nodes node ON node.id = membership.chain_node_entity_id AND node.status = 'active'
		  WHERE mapping.relation_type = $1 AND mapping.status = 'active'
		    AND anchor.id = ANY($2::uuid[])
		    AND ($3::text IS NULL OR (node.canonical_name, node.id) > ($3::text, $4::uuid))
		  GROUP BY node.id, node.canonical_name, node_profile.definition
		  ORDER BY node.canonical_name, node.id
		  LIMIT $5
		)
		SELECT path.anchor_id, path.chain_id, path.mapping_id, node.id, node.entity_type,
		       node.name, node.canonical_name, array_to_json(node.aliases), node.status,
		       path.position, path.membership_updated_at, page.definition,
		       path.chain_name, path.anchor_updated_at, path.chain_updated_at,
		       path.mapping_updated_at, node.updated_at, array_to_json(matched.anchor_ids)
		FROM target_page page
		JOIN entity_nodes node ON node.id = page.target_id
		JOIN LATERAL (
		  SELECT anchor.id AS anchor_id, chain.id AS chain_id, mapping.id AS mapping_id,
		         membership.position, membership.updated_at AS membership_updated_at,
		         membership.contextual_stage, chain.name AS chain_name,
		         anchor.updated_at AS anchor_updated_at, chain.updated_at AS chain_updated_at,
		         mapping.updated_at AS mapping_updated_at
		  FROM entity_edges mapping
		  JOIN entity_nodes anchor ON anchor.id = mapping.to_entity_id AND anchor.status = 'active'
		  JOIN entity_nodes chain ON chain.id = mapping.from_entity_id AND chain.status = 'active'
		  JOIN industry_chain_definitions definition
		    ON definition.entity_id = chain.id AND definition.review_status = 'approved'
		  JOIN industry_chain_node_memberships membership
		    ON membership.industry_chain_entity_id = chain.id
		   AND membership.chain_node_entity_id = page.target_id
		   AND membership.status = 'active' AND membership.review_status = 'approved'
		  WHERE mapping.relation_type = $1 AND mapping.status = 'active'
		    AND anchor.id = ANY($2::uuid[])
		  ORDER BY anchor.id, chain.canonical_name, membership.position, chain.id, mapping.id
		  LIMIT 1
		) path ON true
		JOIN LATERAL (
		  SELECT array_agg(DISTINCT mapping.to_entity_id::text ORDER BY mapping.to_entity_id::text) AS anchor_ids
		  FROM entity_edges mapping
		  JOIN entity_nodes chain ON chain.id = mapping.from_entity_id AND chain.status = 'active'
		  JOIN industry_chain_definitions definition
		    ON definition.entity_id = chain.id AND definition.review_status = 'approved'
		  JOIN industry_chain_node_memberships membership
		    ON membership.industry_chain_entity_id = chain.id
		   AND membership.chain_node_entity_id = page.target_id
		   AND membership.status = 'active' AND membership.review_status = 'approved'
		  WHERE mapping.relation_type = $1 AND mapping.status = 'active'
		    AND mapping.to_entity_id = ANY($2::uuid[])
		) matched ON true
		ORDER BY node.canonical_name, node.id
	`, relationType, anchorEntityIDs, afterName, afterID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]eventsemantics.ResolutionCandidate, 0)
	for rows.Next() {
		var item eventsemantics.ResolutionCandidate
		var aliases []byte
		var matchedAnchorIDs []byte
		var membershipUpdatedAt time.Time
		var anchorUpdatedAt, chainUpdatedAt, mappingUpdatedAt, targetUpdatedAt time.Time
		if err := rows.Scan(
			&item.Receipt.AnchorEntityID, &item.Receipt.IndustryChainEntityID,
			&item.Receipt.MappingRelationID, &item.Entity.ID, &item.Entity.Type,
			&item.Entity.Name, &item.Entity.CanonicalName, &aliases, &item.Entity.Status,
			&item.Receipt.MembershipPosition, &membershipUpdatedAt, &item.Description,
			&item.IndustryChainEntityName, &anchorUpdatedAt, &chainUpdatedAt, &mappingUpdatedAt,
			&targetUpdatedAt, &matchedAnchorIDs,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(aliases, &item.Entity.Aliases); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(matchedAnchorIDs, &item.MatchedAnchorEntityIDs); err != nil {
			return nil, err
		}
		item.Receipt.RouteID = routeID
		item.Receipt.RouteContractVersion = manifest.RouteContractVersion
		item.Receipt.TargetEntityID = item.Entity.ID
		item.Receipt.MembershipUpdatedAt = membershipUpdatedAt.UTC().Format(time.RFC3339Nano)
		item.Receipt.PathFingerprint, err = eventSemanticResolutionFingerprint(item.Receipt, resolutionPathVersions{
			AnchorUpdatedAt:  anchorUpdatedAt.UTC().Format(time.RFC3339Nano),
			ChainUpdatedAt:   chainUpdatedAt.UTC().Format(time.RFC3339Nano),
			MappingUpdatedAt: mappingUpdatedAt.UTC().Format(time.RFC3339Nano),
			TargetUpdatedAt:  targetUpdatedAt.UTC().Format(time.RFC3339Nano),
		})
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func validateEventSemanticResolutionReceipts(
	ctx context.Context,
	query semanticQueryer,
	manifest eventsemantics.Context,
	submission eventsemantics.Submission,
) error {
	for _, link := range submission.EntityLinks {
		if link.ResolutionReceipt == nil {
			continue
		}
		receipt := *link.ResolutionReceipt
		relationType, profileJoin := "", ""
		switch receipt.RouteID {
		case "chain-node-via-industry.v1":
			relationType = "mapped_to_industry"
			profileJoin = `JOIN industry_profiles anchor_profile
			  ON anchor_profile.entity_id = anchor.id AND anchor_profile.review_status = 'approved'`
		case "chain-node-via-concept.v1":
			relationType = "mapped_to_concept"
			profileJoin = `JOIN concept_profiles anchor_profile
			  ON anchor_profile.entity_id = anchor.id AND anchor_profile.review_status = 'approved'`
		default:
			return &eventsemantics.ContextDriftError{Reason: "selected anchor route is no longer valid"}
		}
		if receipt.RouteContractVersion != manifest.RouteContractVersion ||
			receipt.TargetEntityID != link.EntityID {
			return &eventsemantics.ContextDriftError{Reason: "selected anchor route version or target changed"}
		}
		var position int
		var updatedAt, anchorUpdatedAt, chainUpdatedAt, mappingUpdatedAt, targetUpdatedAt time.Time
		err := query.QueryRowContext(ctx, fmt.Sprintf(`
			SELECT membership.position, membership.updated_at, anchor.updated_at,
			       chain.updated_at, mapping.updated_at, node.updated_at
			FROM entity_edges mapping
			JOIN entity_nodes anchor ON anchor.id = mapping.to_entity_id AND anchor.status = 'active'
			%s
			JOIN entity_nodes chain ON chain.id = mapping.from_entity_id AND chain.status = 'active'
			JOIN industry_chain_definitions definition
			  ON definition.entity_id = mapping.from_entity_id AND definition.review_status = 'approved'
			JOIN industry_chain_node_memberships membership
			  ON membership.industry_chain_entity_id = mapping.from_entity_id
			 AND membership.status = 'active' AND membership.review_status = 'approved'
			JOIN chain_node_profiles node_profile
			  ON node_profile.entity_id = membership.chain_node_entity_id
			 AND node_profile.review_status = 'approved'
			JOIN entity_nodes node ON node.id = membership.chain_node_entity_id AND node.status = 'active'
			WHERE mapping.id = $1 AND mapping.relation_type = $2 AND mapping.status = 'active'
			  AND mapping.to_entity_id = $3 AND mapping.from_entity_id = $4
			  AND membership.chain_node_entity_id = $5
			FOR SHARE OF mapping, anchor, anchor_profile, chain, definition, membership, node_profile, node
		`, profileJoin), receipt.MappingRelationID, relationType, receipt.AnchorEntityID,
			receipt.IndustryChainEntityID, receipt.TargetEntityID).Scan(
			&position, &updatedAt, &anchorUpdatedAt, &chainUpdatedAt, &mappingUpdatedAt, &targetUpdatedAt,
		)
		if errors.Is(err, sql.ErrNoRows) {
			return &eventsemantics.ContextDriftError{Reason: "selected anchor path changed or disappeared"}
		}
		if err != nil {
			return err
		}
		current := receipt
		current.MembershipPosition = position
		current.MembershipUpdatedAt = updatedAt.UTC().Format(time.RFC3339Nano)
		fingerprint, err := eventSemanticResolutionFingerprint(current, resolutionPathVersions{
			AnchorUpdatedAt:  anchorUpdatedAt.UTC().Format(time.RFC3339Nano),
			ChainUpdatedAt:   chainUpdatedAt.UTC().Format(time.RFC3339Nano),
			MappingUpdatedAt: mappingUpdatedAt.UTC().Format(time.RFC3339Nano),
			TargetUpdatedAt:  targetUpdatedAt.UTC().Format(time.RFC3339Nano),
		})
		if err != nil {
			return err
		}
		if receipt.MembershipPosition != current.MembershipPosition ||
			receipt.MembershipUpdatedAt != current.MembershipUpdatedAt ||
			receipt.PathFingerprint != fingerprint {
			return &eventsemantics.ContextDriftError{Reason: "selected anchor path changed after candidate resolution"}
		}
	}
	return nil
}

type resolutionPathVersions struct {
	AnchorUpdatedAt  string
	ChainUpdatedAt   string
	MappingUpdatedAt string
	TargetUpdatedAt  string
}

func eventSemanticResolutionFingerprint(
	receipt eventsemantics.ResolutionReceipt,
	versions resolutionPathVersions,
) (string, error) {
	return eventSemanticFingerprint(struct {
		RouteID, RouteVersion, AnchorID, ChainID, MappingID, TargetID, UpdatedAt string
		AnchorUpdatedAt, ChainUpdatedAt, MappingUpdatedAt, TargetUpdatedAt       string
		Position                                                                 int
	}{
		RouteID: receipt.RouteID, RouteVersion: receipt.RouteContractVersion,
		AnchorID: receipt.AnchorEntityID, ChainID: receipt.IndustryChainEntityID,
		MappingID: receipt.MappingRelationID, TargetID: receipt.TargetEntityID,
		UpdatedAt: receipt.MembershipUpdatedAt, Position: receipt.MembershipPosition,
		AnchorUpdatedAt: versions.AnchorUpdatedAt, ChainUpdatedAt: versions.ChainUpdatedAt,
		MappingUpdatedAt: versions.MappingUpdatedAt, TargetUpdatedAt: versions.TargetUpdatedAt,
	})
}

func (r repository) CreateSubmission(
	ctx context.Context,
	submission eventsemantics.Submission,
	precheck eventsemantics.PrecheckResult,
	payload []byte,
	hash string,
) (eventsemantics.SubmissionResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return eventsemantics.SubmissionResult{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if existing, found, err := existingEventSemanticSubmission(ctx, tx, submission.AgentExecutionID); err != nil {
		return eventsemantics.SubmissionResult{}, err
	} else if found {
		if existing.CanonicalPayloadHash != hash {
			return eventsemantics.SubmissionResult{}, &eventsemantics.ConflictError{Reason: "agent_execution_id is bound to a different canonical payload"}
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
		return eventsemantics.SubmissionResult{}, &eventsemantics.NotFoundError{Resource: "Event Semantic Context Lease"}
	} else if err != nil {
		return eventsemantics.SubmissionResult{}, err
	}
	if eventID != submission.EventID || status != "active" {
		return eventsemantics.SubmissionResult{}, &eventsemantics.ConflictError{
			Reason: "context lease is not active for this Event",
		}
	}
	if leaseAgentExecutionID != submission.AgentExecutionID {
		return eventsemantics.SubmissionResult{}, &eventsemantics.ConflictError{
			Reason: "Submission agent_execution_id differs from its Context Lease",
		}
	}
	if leaseSupersedesSubmissionID.String != submission.SupersedesSubmissionID {
		return eventsemantics.SubmissionResult{}, &eventsemantics.ConflictError{
			Reason: "Submission supersedes identity differs from its Context Lease",
		}
	}
	var manifestReference eventsemantics.ContextManifest
	if err := json.Unmarshal(manifestPayload, &manifestReference); err != nil {
		return eventsemantics.SubmissionResult{}, err
	}
	manifest, err := eventSemanticContextFromManifest(ctx, tx, manifestReference)
	if err != nil {
		return eventsemantics.SubmissionResult{}, err
	}
	if err := validateEventSemanticResolutionReceipts(ctx, tx, manifest, submission); err != nil {
		return eventsemantics.SubmissionResult{}, err
	}
	transactionContext, err := hydrateEventSemanticSubmissionContext(
		ctx, tx, manifest, submission, true,
	)
	if err != nil {
		return eventsemantics.SubmissionResult{}, err
	}
	precheck = eventsemantics.Precheck(transactionContext, submission)
	if submission.SupersedesSubmissionID != "" {
		var supersededEventID string
		var supersededStatus eventsemantics.ReviewStatus
		if err := tx.QueryRowContext(ctx, `
			SELECT event_id, status
			FROM event_semantic_submissions
			WHERE id = $1
			FOR UPDATE
		`, submission.SupersedesSubmissionID).Scan(&supersededEventID, &supersededStatus); errors.Is(err, sql.ErrNoRows) {
			return eventsemantics.SubmissionResult{}, &eventsemantics.NotFoundError{Resource: "superseded Event Semantic Submission"}
		} else if err != nil {
			return eventsemantics.SubmissionResult{}, err
		}
		if supersededEventID != submission.EventID || supersededStatus == eventsemantics.StatusSuperseded {
			return eventsemantics.SubmissionResult{}, &eventsemantics.ConflictError{
				Reason: "supersedes_submission_id must reference the current Event's active prior Submission",
			}
		}
	}
	submissionID, snapshotID := uuid.NewString(), uuid.NewString()
	decisionPayload, err := json.Marshal(precheck)
	if err != nil {
		return eventsemantics.SubmissionResult{}, err
	}
	counts, _ := json.Marshal(map[string]int{
		"entity_links":     len(submission.EntityLinks),
		"variable_signals": len(submission.VariableSignals),
		"direct_impacts":   len(submission.DirectImpacts),
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
		    'event-semantics.phase-one',1,$15,$16,$17,$18
	    )
	`, submissionID, submission.ContextLeaseID, submission.EventID, submission.AgentExecutionID,
		submission.AgentKey, submission.AgentVersion, nullString(submission.SupersedesSubmissionID),
		submission.GeneratorPromptHash, submission.GeneratorModel,
		submission.ReviewerPromptHash, submission.ReviewerModel,
		nullString(submission.AdjudicatorPromptHash), nullString(submission.AdjudicatorModel),
		submission.OntologyVersion, hash, submissionStatus, counts, decisionPayload,
	); err != nil {
		return eventsemantics.SubmissionResult{}, err
	}
	for _, link := range submission.EntityLinks {
		if link.ResolutionReceipt == nil {
			continue
		}
		receiptPayload, err := json.Marshal(link.ResolutionReceipt)
		if err != nil {
			return eventsemantics.SubmissionResult{}, err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO event_semantic_resolution_bindings(
			    id, semantic_submission_id, context_lease_id, candidate_key, mention,
			    anchor_entity_id, target_entity_id, route_id, route_contract_version,
			    path_fingerprint, resolution_receipt
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		`, uuid.NewString(), submissionID, submission.ContextLeaseID, link.Key, link.Mention,
			link.ResolutionReceipt.AnchorEntityID, link.EntityID, link.ResolutionReceipt.RouteID,
			link.ResolutionReceipt.RouteContractVersion, link.ResolutionReceipt.PathFingerprint,
			receiptPayload); err != nil {
			return eventsemantics.SubmissionResult{}, err
		}
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO event_semantic_candidate_snapshots(id, semantic_submission_id, payload, canonical_payload_hash)
		VALUES ($1,$2,$3,$4)
	`, snapshotID, submissionID, payload, hash); err != nil {
		return eventsemantics.SubmissionResult{}, err
	}
	if err := insertReviewableSemanticCandidates(ctx, tx, submissionID, submission, precheck); err != nil {
		return eventsemantics.SubmissionResult{}, err
	}
	if submissionStatus != eventsemantics.StatusPendingReview &&
		submissionStatus != eventsemantics.StatusNeedsReanalysis {
		if _, err := tx.ExecContext(ctx, `
			UPDATE event_semantic_context_leases
			SET status = 'consumed', consumed_at = now()
			WHERE id = $1
		`, submission.ContextLeaseID); err != nil {
			return eventsemantics.SubmissionResult{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return eventsemantics.SubmissionResult{}, err
	}
	return eventsemantics.SubmissionResult{
		SubmissionID: submissionID, EventID: submission.EventID, Status: submissionStatus,
		CanonicalPayloadHash: hash, Precheck: precheck,
	}, nil
}

func (r repository) ReplaySubmission(
	ctx context.Context,
	executionID string,
	canonicalPayloadHash string,
) (eventsemantics.SubmissionResult, bool, error) {
	existing, found, err := queryEventSemanticSubmission(ctx, r.db.QueryRowContext(ctx, `
		SELECT id,event_id,status,canonical_payload_hash,decision_summary
		FROM event_semantic_submissions
		WHERE agent_execution_id = $1
	`, executionID))
	if err != nil || !found {
		return existing, found, err
	}
	if existing.CanonicalPayloadHash != canonicalPayloadHash {
		return eventsemantics.SubmissionResult{}, false, &eventsemantics.ConflictError{
			Reason: "agent_execution_id is bound to a different canonical payload",
		}
	}
	existing.Replayed = true
	return existing, true, nil
}

func insertReviewableSemanticCandidates(
	ctx context.Context,
	tx *sql.Tx,
	submissionID string,
	submission eventsemantics.Submission,
	precheck eventsemantics.PrecheckResult,
) error {
	linkDecisions := decisionsByKey(precheck.EntityLinks)
	linkIDs := make(map[string]string)
	for _, candidate := range submission.EntityLinks {
		decision := linkDecisions[candidate.Key]
		if decision.Status == eventsemantics.StatusRejected {
			continue
		}
		id := uuid.NewString()
		linkIDs[candidate.Key] = id
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO event_entity_links(
			    id,event_id,entity_id,entity_role,assign_source,review_status,evidence_note,
			    semantic_submission_id,candidate_key,resolved_mention,resolution_method,
			    resolution_confidence,evidence_ids,provenance,reason_code
			) VALUES ($1,$2,$3,$4,'ai',$5,'',$6,$7,$8,$9,$10,$11,'semantic',$12)
		`, id, submission.EventID, candidate.EntityID, candidate.EntityRole, decision.Status,
			submissionID, candidate.Key, candidate.Mention, candidate.ResolutionMethod,
			nullString(candidate.ResolutionConfidence), candidate.EvidenceIDs, nullString(decision.ReasonCode),
		); err != nil {
			return err
		}
	}
	signalDecisions := decisionsByKey(precheck.VariableSignals)
	signalIDs := make(map[string]string)
	for _, candidate := range submission.VariableSignals {
		decision := signalDecisions[candidate.Key]
		linkID := linkIDs[candidate.SubjectLinkKey]
		if decision.Status == eventsemantics.StatusRejected || linkID == "" {
			continue
		}
		id := uuid.NewString()
		signalIDs[candidate.Key] = id
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO variable_signals(
			    id,semantic_submission_id,candidate_key,source_event_id,subject_event_entity_link_id,
			    variable_key,variable_version,direction,assertion_modality,evidence_ids,
			    statement_at,valid_from,valid_until,forecast_period_start,forecast_period_end,
			    extraction_confidence,review_status,reason_code
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
		`, id, submissionID, candidate.Key, submission.EventID, linkID, candidate.VariableKey,
			candidate.VariableVersion, candidate.Direction, candidate.AssertionModality,
			candidate.EvidenceIDs, candidate.StatementAt, candidate.ValidFrom, candidate.ValidUntil,
			candidate.ForecastPeriodStart, candidate.ForecastPeriodEnd,
			nullString(candidate.ExtractionConfidence), decision.Status, nullString(decision.ReasonCode),
		); err != nil {
			return err
		}
		for _, measurement := range candidate.Measurements {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO variable_signal_measurements(
				    id,variable_signal_id,measurement_role,value_shape,raw_value,raw_lower,raw_upper,
				    raw_unit,canonical_value,canonical_lower,canonical_upper,canonical_unit,currency,
				    scale,comparison_basis,comparison_period,raw_text,is_approximate,evidence_id
				) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
			`, uuid.NewString(), id, measurement.Role, measurement.Shape,
				nullableDecimal(measurement.RawValue), nullableDecimal(measurement.RawLower),
				nullableDecimal(measurement.RawUpper), nullString(measurement.RawUnit),
				nullableDecimal(measurement.CanonicalValue), nullableDecimal(measurement.CanonicalLower),
				nullableDecimal(measurement.CanonicalUpper), nullString(measurement.CanonicalUnit),
				nullString(measurement.Currency), nullString(measurement.Scale),
				nullString(measurement.ComparisonBasis), nullString(measurement.ComparisonPeriod),
				measurement.RawText, measurement.IsApproximate, measurement.EvidenceID,
			); err != nil {
				return err
			}
		}
	}
	impactDecisions := decisionsByKey(precheck.DirectImpacts)
	for _, candidate := range submission.DirectImpacts {
		decision := impactDecisions[candidate.Key]
		signalID := signalIDs[candidate.SourceSignalKey]
		if decision.Status == eventsemantics.StatusRejected || signalID == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO direct_impact_assertions(
			    id,semantic_submission_id,candidate_key,source_variable_signal_id,target_entity_id,
			    affected_variable_key,affected_variable_version,affected_direction,derivation_type,
			    mechanism_summary,evidence_ids,entity_relation_id,rule_key,rule_version,
			    assertion_confidence,review_status,reason_code
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		`, uuid.NewString(), submissionID, candidate.Key, signalID, candidate.TargetEntityID,
			candidate.AffectedVariableKey, candidate.AffectedVariableVersion,
			candidate.AffectedDirection, candidate.DerivationType, candidate.MechanismSummary,
			candidate.EvidenceIDs, nullString(candidate.EntityRelationID), nullString(candidate.RuleKey),
			nullablePositiveInt(candidate.RuleVersion), nullString(candidate.AssertionConfidence),
			decision.Status, nullString(decision.ReasonCode),
		); err != nil {
			return err
		}
	}
	return nil
}

func (r repository) SubmitReview(
	ctx context.Context,
	submission eventsemantics.ReviewSubmission,
	payload []byte,
	hash string,
) (eventsemantics.SubmissionResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return eventsemantics.SubmissionResult{}, err
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
		return eventsemantics.SubmissionResult{}, &eventsemantics.NotFoundError{Resource: "Event Semantic Submission"}
	} else if err != nil {
		return eventsemantics.SubmissionResult{}, err
	}
	if !reviewIdentity.matches(submission) {
		return eventsemantics.SubmissionResult{}, &eventsemantics.ConflictError{
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
			return eventsemantics.SubmissionResult{}, &eventsemantics.ConflictError{Reason: "reviewer_execution_key is bound to a different payload"}
		}
		result, found, err := eventSemanticSubmissionByID(ctx, tx, submission.SubmissionID)
		if err != nil {
			return eventsemantics.SubmissionResult{}, err
		}
		if !found {
			return eventsemantics.SubmissionResult{}, &eventsemantics.NotFoundError{Resource: "Event Semantic Submission"}
		}
		result.Replayed = true
		return result, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return eventsemantics.SubmissionResult{}, err
	}
	result, found, err := eventSemanticSubmissionByID(ctx, tx, submission.SubmissionID)
	if err != nil {
		return eventsemantics.SubmissionResult{}, err
	}
	if !found {
		return eventsemantics.SubmissionResult{}, &eventsemantics.NotFoundError{Resource: "Event Semantic Submission"}
	}
	if result.Status != eventsemantics.StatusPendingReview && result.Status != eventsemantics.StatusNeedsReanalysis {
		return eventsemantics.SubmissionResult{}, &eventsemantics.ConflictError{Reason: "Event Semantic Submission is not reviewable"}
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO event_semantic_review_snapshots(
		    id,semantic_submission_id,reviewer_execution_key,payload,canonical_payload_hash
		) VALUES ($1,$2,$3,$4,$5)
	`, uuid.NewString(), submission.SubmissionID, submission.ReviewerExecutionKey, payload, hash); err != nil {
		return eventsemantics.SubmissionResult{}, err
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
		return eventsemantics.SubmissionResult{}, err
	}
	if err := applySemanticReview(
		ctx, tx, submission, &result.Precheck, priorReviewCount > retryBudget,
	); err != nil {
		return eventsemantics.SubmissionResult{}, err
	}
	status := summarizeSemanticSubmission(result.Precheck)
	decisionPayload, err := json.Marshal(result.Precheck)
	if err != nil {
		return eventsemantics.SubmissionResult{}, err
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
		return eventsemantics.SubmissionResult{}, err
	}
	if status == eventsemantics.StatusAccepted ||
		status == eventsemantics.StatusRejected ||
		status == eventsemantics.StatusQuarantined {
		if _, err := tx.ExecContext(ctx, `
			UPDATE event_semantic_context_leases
			SET status = 'consumed', consumed_at = now()
			WHERE id = (SELECT context_lease_id FROM event_semantic_submissions WHERE id = $1)
		`, submission.SubmissionID); err != nil {
			return eventsemantics.SubmissionResult{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return eventsemantics.SubmissionResult{}, err
	}
	result.Status = status
	return result, nil
}

func applySemanticReview(
	ctx context.Context,
	tx *sql.Tx,
	submission eventsemantics.ReviewSubmission,
	precheck *eventsemantics.PrecheckResult,
	quarantineIndeterminate bool,
) error {
	pending := map[string]*eventsemantics.CandidateDecision{}
	candidateEvidence := make(map[string][]string)
	register := func(kind string, items []eventsemantics.CandidateDecision) {
		for index := range items {
			if items[index].Status == eventsemantics.StatusPendingReview ||
				items[index].Status == eventsemantics.StatusNeedsReanalysis {
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
		return &eventsemantics.ValidationError{Reason: "review must decide every reviewable candidate exactly once"}
	}
	seen := make(map[string]struct{}, len(submission.Items))
	for _, item := range submission.Items {
		identity := item.CandidateType + ":" + item.CandidateKey
		decision, exists := pending[identity]
		if !exists {
			return &eventsemantics.ConflictError{Reason: "review references a non-reviewable candidate"}
		}
		if _, duplicate := seen[identity]; duplicate {
			return &eventsemantics.ValidationError{Reason: "review candidate identities must be unique"}
		}
		if !reviewEvidenceMatchesCandidate(item.EvidenceIDs, candidateEvidence[identity]) {
			return &eventsemantics.ValidationError{Reason: "review Evidence must cite the candidate Event Evidence"}
		}
		seen[identity] = struct{}{}
		switch item.Decision {
		case "pass":
			decision.Status, decision.ReasonCode = eventsemantics.StatusAccepted, ""
		case "fail":
			decision.Status, decision.ReasonCode = eventsemantics.StatusRejected, firstReason(item.ReasonCodes, "reviewer_failed")
		case "indeterminate":
			if quarantineIndeterminate {
				decision.Status, decision.ReasonCode = eventsemantics.StatusQuarantined, "unresolved_after_retry_budget"
			} else {
				decision.Status, decision.ReasonCode = eventsemantics.StatusNeedsReanalysis, firstReason(item.ReasonCodes, "reviewer_indeterminate")
			}
		}
	}
	propagateSemanticReview(precheck)
	if summarizeSemanticSubmission(*precheck) == eventsemantics.StatusAccepted {
		if err := supersedePriorSemanticSubmission(ctx, tx, submission.SubmissionID); err != nil {
			return err
		}
	}
	return persistSemanticDecisions(ctx, tx, submission.SubmissionID, *precheck)
}

func supersedePriorSemanticSubmission(ctx context.Context, tx *sql.Tx, runID string) error {
	var priorSubmissionID sql.NullString
	if err := tx.QueryRowContext(ctx, `
		SELECT supersedes_submission_id::text
		FROM event_semantic_submissions
		WHERE id = $1
	`, runID).Scan(&priorSubmissionID); err != nil {
		return err
	}
	if !priorSubmissionID.Valid {
		return nil
	}
	for _, table := range []string{"event_entity_links", "variable_signals", "direct_impact_assertions"} {
		query := fmt.Sprintf(`
			UPDATE %s
			SET review_status = 'superseded', reason_code = 'superseded_by_reanalysis', updated_at = now()
			WHERE semantic_submission_id = $1 AND review_status <> 'superseded'
		`, table)
		if _, err := tx.ExecContext(ctx, query, priorSubmissionID.String); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE event_semantic_submissions
		SET status = 'superseded', finalized_at = now()
		WHERE id = $1 AND status <> 'superseded'
	`, priorSubmissionID.String); err != nil {
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

func (identity semanticReviewIdentity) matches(submission eventsemantics.ReviewSubmission) bool {
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

func propagateSemanticReview(precheck *eventsemantics.PrecheckResult) {
	linkStatus := make(map[string]eventsemantics.CandidateDecision, len(precheck.EntityLinks))
	for _, item := range precheck.EntityLinks {
		linkStatus[item.CandidateKey] = item
	}
	signalStatus := make(map[string]eventsemantics.CandidateDecision, len(precheck.VariableSignals))
	signalByKey := make(map[string]eventsemantics.VariableSignalCandidate, len(precheck.ReviewerWorkPackage.VariableSignals))
	for _, item := range precheck.ReviewerWorkPackage.VariableSignals {
		signalByKey[item.Key] = item
	}
	for index := range precheck.VariableSignals {
		candidate := signalByKey[precheck.VariableSignals[index].CandidateKey]
		upstream := linkStatus[candidate.SubjectLinkKey]
		if upstream.Status == eventsemantics.StatusRejected {
			precheck.VariableSignals[index].Status = eventsemantics.StatusRejected
			precheck.VariableSignals[index].ReasonCode = "upstream_rejected"
		} else if upstream.Status == eventsemantics.StatusQuarantined {
			precheck.VariableSignals[index].Status = eventsemantics.StatusQuarantined
			precheck.VariableSignals[index].ReasonCode = "upstream_quarantined"
		} else if upstream.Status == eventsemantics.StatusNeedsReanalysis {
			precheck.VariableSignals[index].Status = eventsemantics.StatusNeedsReanalysis
			precheck.VariableSignals[index].ReasonCode = "upstream_pending"
		}
		signalStatus[precheck.VariableSignals[index].CandidateKey] = precheck.VariableSignals[index]
	}
	impactByKey := make(map[string]eventsemantics.DirectImpactCandidate, len(precheck.ReviewerWorkPackage.DirectImpacts))
	for _, item := range precheck.ReviewerWorkPackage.DirectImpacts {
		impactByKey[item.Key] = item
	}
	for index := range precheck.DirectImpacts {
		candidate := impactByKey[precheck.DirectImpacts[index].CandidateKey]
		upstream := signalStatus[candidate.SourceSignalKey]
		if upstream.Status == eventsemantics.StatusRejected {
			precheck.DirectImpacts[index].Status = eventsemantics.StatusRejected
			precheck.DirectImpacts[index].ReasonCode = "upstream_rejected"
		} else if upstream.Status == eventsemantics.StatusQuarantined {
			precheck.DirectImpacts[index].Status = eventsemantics.StatusQuarantined
			precheck.DirectImpacts[index].ReasonCode = "upstream_quarantined"
		} else if upstream.Status == eventsemantics.StatusNeedsReanalysis {
			precheck.DirectImpacts[index].Status = eventsemantics.StatusNeedsReanalysis
			precheck.DirectImpacts[index].ReasonCode = "upstream_pending"
		}
	}
}

func summarizeSemanticSubmission(precheck eventsemantics.PrecheckResult) eventsemantics.ReviewStatus {
	hasAccepted, hasPending, hasNeeds, hasQuarantined := false, false, false, false
	for _, group := range [][]eventsemantics.CandidateDecision{
		precheck.EntityLinks, precheck.VariableSignals, precheck.DirectImpacts,
	} {
		for _, item := range group {
			hasAccepted = hasAccepted || item.Status == eventsemantics.StatusAccepted
			hasPending = hasPending || item.Status == eventsemantics.StatusPendingReview
			hasNeeds = hasNeeds || item.Status == eventsemantics.StatusNeedsReanalysis
			hasQuarantined = hasQuarantined || item.Status == eventsemantics.StatusQuarantined
		}
	}
	if hasPending {
		return eventsemantics.StatusPendingReview
	}
	if hasNeeds {
		return eventsemantics.StatusNeedsReanalysis
	}
	if hasAccepted {
		return eventsemantics.StatusAccepted
	}
	if hasQuarantined {
		return eventsemantics.StatusQuarantined
	}
	return eventsemantics.StatusRejected
}

func persistSemanticDecisions(
	ctx context.Context,
	tx *sql.Tx,
	runID string,
	precheck eventsemantics.PrecheckResult,
) error {
	for table, items := range map[string][]eventsemantics.CandidateDecision{
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

func (r repository) GetEventSemantics(ctx context.Context, eventID string) (eventsemantics.EventSemanticsResult, error) {
	var exists bool
	if err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM events WHERE id = $1)`, eventID).Scan(&exists); err != nil {
		return eventsemantics.EventSemanticsResult{}, err
	}
	if !exists {
		return eventsemantics.EventSemanticsResult{}, &eventsemantics.NotFoundError{Resource: "Event"}
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT run.id, run.event_id, run.status, run.canonical_payload_hash, run.decision_summary,
		       run.context_lease_id::text, run.agent_execution_id, run.agent_key, run.agent_version,
		       COALESCE(run.supersedes_submission_id::text, ''),
		       run.generator_prompt_hash, run.generator_model,
		       run.reviewer_prompt_hash, run.reviewer_model,
		       COALESCE(run.adjudicator_prompt_hash, ''), COALESCE(run.adjudicator_model, ''),
		       run.ontology_version,
		       run.acceptance_policy_key || '@' || run.acceptance_policy_version::text,
		       snapshot.payload, run.created_at, run.finalized_at
		FROM event_semantic_submissions run
		JOIN event_semantic_candidate_snapshots snapshot ON snapshot.semantic_submission_id = run.id
		WHERE run.event_id = $1
		ORDER BY run.created_at, run.id
	`, eventID)
	if err != nil {
		return eventsemantics.EventSemanticsResult{}, err
	}
	defer rows.Close()
	result := eventsemantics.EventSemanticsResult{EventID: eventID}
	for rows.Next() {
		var run eventsemantics.SubmissionResult
		var decisions []byte
		var candidateSnapshot []byte
		if err := rows.Scan(
			&run.SubmissionID, &run.EventID, &run.Status, &run.CanonicalPayloadHash, &decisions,
			&run.ContextLeaseID, &run.AgentExecutionID, &run.AgentKey, &run.AgentVersion,
			&run.SupersedesSubmissionID, &run.GeneratorPromptHash, &run.GeneratorModel,
			&run.ReviewerPromptHash, &run.ReviewerModel,
			&run.AdjudicatorPromptHash, &run.AdjudicatorModel,
			&run.OntologyVersion, &run.AcceptancePolicyVersion,
			&candidateSnapshot, &run.CreatedAt, &run.FinalizedAt,
		); err != nil {
			return eventsemantics.EventSemanticsResult{}, err
		}
		if err := json.Unmarshal(decisions, &run.Precheck); err != nil {
			return eventsemantics.EventSemanticsResult{}, err
		}
		run.CandidateSnapshot = append(json.RawMessage(nil), candidateSnapshot...)
		if run.ReviewSnapshots, err = r.eventSemanticReviewSnapshots(ctx, run.SubmissionID); err != nil {
			return eventsemantics.EventSemanticsResult{}, err
		}
		if err := r.populateSemanticRecordIDs(ctx, run.SubmissionID, &run.Precheck); err != nil {
			return eventsemantics.EventSemanticsResult{}, err
		}
		result.Submissions = append(result.Submissions, run)
	}
	return result, rows.Err()
}

func (r repository) eventSemanticReviewSnapshots(
	ctx context.Context,
	runID string,
) ([]eventsemantics.ReviewSnapshot, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT reviewer_execution_key, canonical_payload_hash, payload, created_at
		FROM event_semantic_review_snapshots
		WHERE semantic_submission_id = $1
		ORDER BY created_at, id
	`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []eventsemantics.ReviewSnapshot
	for rows.Next() {
		var item eventsemantics.ReviewSnapshot
		var payload []byte
		if err := rows.Scan(
			&item.ReviewerExecutionKey, &item.CanonicalPayloadHash, &payload, &item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.Payload = append(json.RawMessage(nil), payload...)
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r repository) populateSemanticRecordIDs(
	ctx context.Context,
	runID string,
	precheck *eventsemantics.PrecheckResult,
) error {
	groups := []struct {
		table string
		items *[]eventsemantics.CandidateDecision
	}{
		{table: "event_entity_links", items: &precheck.EntityLinks},
		{table: "variable_signals", items: &precheck.VariableSignals},
		{table: "direct_impact_assertions", items: &precheck.DirectImpacts},
	}
	for _, group := range groups {
		query := fmt.Sprintf(`
			SELECT candidate_key, id::text
			FROM %s
			WHERE semantic_submission_id = $1
		`, group.table)
		rows, err := r.db.QueryContext(ctx, query, runID)
		if err != nil {
			return err
		}
		ids := make(map[string]string)
		for rows.Next() {
			var key, id string
			if err := rows.Scan(&key, &id); err != nil {
				rows.Close()
				return err
			}
			ids[key] = id
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
		for index := range *group.items {
			(*group.items)[index].RecordID = ids[(*group.items)[index].CandidateKey]
		}
	}
	return nil
}

func existingEventSemanticSubmission(ctx context.Context, tx *sql.Tx, executionID string) (eventsemantics.SubmissionResult, bool, error) {
	return queryEventSemanticSubmission(ctx, tx.QueryRowContext(ctx, `
		SELECT id,event_id,status,canonical_payload_hash,decision_summary
		FROM event_semantic_submissions
		WHERE agent_execution_id = $1
	`, executionID))
}

func eventSemanticSubmissionByID(ctx context.Context, tx *sql.Tx, runID string) (eventsemantics.SubmissionResult, bool, error) {
	return queryEventSemanticSubmission(ctx, tx.QueryRowContext(ctx, `
		SELECT id,event_id,status,canonical_payload_hash,decision_summary
		FROM event_semantic_submissions
		WHERE id = $1
		FOR UPDATE
	`, runID))
}

type semanticSubmissionRow interface {
	Scan(...any) error
}

func queryEventSemanticSubmission(_ context.Context, row semanticSubmissionRow) (eventsemantics.SubmissionResult, bool, error) {
	var result eventsemantics.SubmissionResult
	var decisions []byte
	err := row.Scan(&result.SubmissionID, &result.EventID, &result.Status, &result.CanonicalPayloadHash, &decisions)
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
		return eventsemantics.SubmissionResult{}, false, nil
	}
	if err != nil {
		return eventsemantics.SubmissionResult{}, false, err
	}
	if err := json.Unmarshal(decisions, &result.Precheck); err != nil {
		return eventsemantics.SubmissionResult{}, false, err
	}
	return result, true, nil
}

func decisionsByKey(items []eventsemantics.CandidateDecision) map[string]eventsemantics.CandidateDecision {
	result := make(map[string]eventsemantics.CandidateDecision, len(items))
	for _, item := range items {
		result[item.CandidateKey] = item
	}
	return result
}

func nullableDecimal(value *string) any {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	return strings.TrimSpace(*value)
}

func firstReason(reasons []string, fallback string) string {
	for _, reason := range reasons {
		if strings.TrimSpace(reason) != "" {
			return strings.TrimSpace(reason)
		}
	}
	return fallback
}
