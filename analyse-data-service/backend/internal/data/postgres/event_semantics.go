package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventsemantics"
)

const (
	eventSemanticsOntologyVersion = "event-semantics.phase-one@1"
	eventSemanticsPolicyVersion   = "event-semantics.phase-one@1"
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
	      AND evidence.evidence_hash ~ '^[0-9a-f]{64}$'
	      AND btrim(evidence.evidence_excerpt) <> ''
	      AND btrim(document.source_name) <> ''
	      AND btrim(document.source_type) <> ''
	      AND btrim(document.title) <> ''
	      AND document.collected_at IS NOT NULL
	      AND evidence.created_at IS NOT NULL
	)
	AND NOT EXISTS (
	    SELECT 1
	    FROM event_sources evidence
	    JOIN raw_documents document ON document.id = evidence.raw_document_id
	    WHERE evidence.event_id = e.id
	      AND (
	          evidence.evidence_hash !~ '^[0-9a-f]{64}$'
	          OR btrim(evidence.evidence_excerpt) = ''
	          OR btrim(document.source_name) = ''
	          OR btrim(document.source_type) = ''
	          OR btrim(document.title) = ''
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
	err = tx.QueryRowContext(ctx, `
		SELECT lease.id, lease.event_id, COALESCE(lease.supersedes_submission_id::text, ''),
		       lease.worker_id, lease.status, lease.lease_expires_at,
		       COALESCE(submission.status, '')
		FROM event_semantic_context_leases lease
		LEFT JOIN event_semantic_submissions submission ON submission.context_lease_id = lease.id
		WHERE lease.agent_execution_id = $1
		FOR UPDATE OF lease
	`, request.AgentExecutionID).Scan(
		&existing.ID, &existingEventID, &existingSupersedes, &existingWorkerID,
		&existing.Status, &existing.LeaseExpiresAt, &submissionStatus,
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
		if _, err := tx.ExecContext(ctx, `
			UPDATE event_semantic_context_leases
			SET status = 'active', lease_expires_at = $2, consumed_at = NULL
			WHERE id = $1
		`, existing.ID, existing.LeaseExpiresAt); err != nil {
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
	snapshot, err := buildEventSemanticContext(ctx, tx, contextLease.ID, contextLease.EventID)
	if err != nil {
		return eventsemantics.ContextLease{}, err
	}
	snapshotPayload, err := json.Marshal(snapshot)
	if err != nil {
		return eventsemantics.ContextLease{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO event_semantic_context_leases(
		    id, event_id, supersedes_submission_id, agent_execution_id, worker_id,
		    status, lease_expires_at, context_snapshot
		) VALUES ($1, $2, NULLIF($3, '')::uuid, $4, $5, 'active', $6, $7)
	`, contextLease.ID, contextLease.EventID, contextLease.SupersedesSubmissionID,
		request.AgentExecutionID, request.WorkerID, contextLease.LeaseExpiresAt,
		snapshotPayload); err != nil {
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
		SELECT context_snapshot
		FROM event_semantic_context_leases
		WHERE id = $1 AND status = 'active' AND lease_expires_at > now()
	`, contextLeaseID).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return eventsemantics.Context{}, &eventsemantics.NotFoundError{Resource: "active Event Semantic Context Lease"}
	}
	if err != nil {
		return eventsemantics.Context{}, err
	}
	var result eventsemantics.Context
	if err := json.Unmarshal(payload, &result); err != nil {
		return eventsemantics.Context{}, err
	}
	return result, nil
}

func buildEventSemanticContext(
	ctx context.Context,
	query semanticQueryer,
	contextLeaseID string,
	eventID string,
) (eventsemantics.Context, error) {
	result := eventsemantics.Context{
		ContextLeaseID: contextLeaseID, OntologyVersion: eventSemanticsOntologyVersion,
		PolicyVersion: eventSemanticsPolicyVersion,
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
	if result.Evidence, err = eventSemanticEvidence(ctx, query, eventID); err != nil {
		return eventsemantics.Context{}, err
	}
	if result.Entities, err = eventSemanticEntities(ctx, query); err != nil {
		return eventsemantics.Context{}, err
	}
	if result.Relations, err = eventSemanticRelations(ctx, query); err != nil {
		return eventsemantics.Context{}, err
	}
	if result.Variables, err = eventSemanticVariables(ctx, query); err != nil {
		return eventsemantics.Context{}, err
	}
	if result.Rules, err = eventSemanticRules(ctx, query); err != nil {
		return eventsemantics.Context{}, err
	}
	return result, nil
}

func eventSemanticEvidence(ctx context.Context, query semanticQueryer, eventID string) ([]eventsemantics.Evidence, error) {
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
		WHERE es.event_id = $1
		ORDER BY COALESCE(es.is_primary, false) DESC, es.id
	`, eventID)
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

func eventSemanticVariables(ctx context.Context, query semanticQueryer) ([]eventsemantics.VariableDefinition, error) {
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
		GROUP BY definition.variable_key, definition.version
		ORDER BY definition.variable_key, definition.version
	`)
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

func eventSemanticRules(ctx context.Context, query semanticQueryer) ([]eventsemantics.DirectTransmissionRule, error) {
	rows, err := query.QueryContext(ctx, `
		SELECT rule_key, version, status, source_entity_type, source_variable_key,
		       source_variable_version, source_direction, relation_type, target_entity_type,
		       affected_variable_key, affected_variable_version, affected_direction,
		       condition_summary, mechanism_template
		FROM direct_transmission_rules
		WHERE status = 'approved'
		ORDER BY rule_key, version
	`)
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

func (r repository) Resolve(
	ctx context.Context,
	contextLeaseID string,
	mentions []eventsemantics.EntityMention,
) ([]eventsemantics.EntityResolution, error) {
	snapshot, err := r.Context(ctx, contextLeaseID)
	if err != nil {
		return nil, err
	}
	result := make([]eventsemantics.EntityResolution, 0, len(mentions))
	for _, mention := range mentions {
		resolution := eventsemantics.EntityResolution{Mention: mention.Mention}
		for _, entity := range snapshot.Entities {
			if entity.Status == "active" && containsString(mention.AllowedEntityTypes, entity.Type) &&
				entityMatchesMention(entity, mention.Mention) {
				resolution.Candidates = append(resolution.Candidates, entity)
			}
		}
		sort.Slice(resolution.Candidates, func(i, j int) bool {
			leftExact := strings.EqualFold(resolution.Candidates[i].CanonicalName, mention.Mention)
			rightExact := strings.EqualFold(resolution.Candidates[j].CanonicalName, mention.Mention)
			if leftExact != rightExact {
				return leftExact
			}
			if resolution.Candidates[i].CanonicalName != resolution.Candidates[j].CanonicalName {
				return resolution.Candidates[i].CanonicalName < resolution.Candidates[j].CanonicalName
			}
			return resolution.Candidates[i].ID < resolution.Candidates[j].ID
		})
		if len(resolution.Candidates) > 10 {
			resolution.Candidates = resolution.Candidates[:10]
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
	snapshot, err := r.Context(ctx, contextLeaseID)
	if err != nil {
		return nil, err
	}
	entities := make(map[string]eventsemantics.Entity, len(snapshot.Entities))
	for _, entity := range snapshot.Entities {
		entities[entity.ID] = entity
	}
	var result []eventsemantics.DirectTarget
	for _, relation := range snapshot.Relations {
		target, exists := entities[relation.ToEntityID]
		if relation.Status == "active" && relation.FromEntityID == subjectEntityID &&
			exists && target.Status == "active" && containsString(allowedTargetTypes, target.Type) {
			result = append(result, eventsemantics.DirectTarget{Entity: target, Relation: relation})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Relation.Type != result[j].Relation.Type {
			return result[i].Relation.Type < result[j].Relation.Type
		}
		if result[i].Entity.CanonicalName != result[j].Entity.CanonicalName {
			return result[i].Entity.CanonicalName < result[j].Entity.CanonicalName
		}
		return result[i].Entity.ID < result[j].Entity.ID
	})
	if len(result) > 50 {
		result = result[:50]
	}
	return result, nil
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func entityMatchesMention(entity eventsemantics.Entity, mention string) bool {
	if strings.EqualFold(entity.Name, mention) || strings.EqualFold(entity.CanonicalName, mention) {
		return true
	}
	for _, alias := range entity.Aliases {
		if strings.EqualFold(alias, mention) {
			return true
		}
	}
	return false
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
	if err := tx.QueryRowContext(ctx, `
		SELECT event_id, agent_execution_id, status, supersedes_submission_id
		FROM event_semantic_context_leases
		WHERE id = $1 AND lease_expires_at > now()
		FOR UPDATE
	`, submission.ContextLeaseID).Scan(
		&eventID, &leaseAgentExecutionID, &status, &leaseSupersedesSubmissionID,
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
