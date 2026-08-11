package eventsemantic

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
	eventbiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventsemantic"
)

const (
	eventSemanticsOntologyVersion     = "event-semantics.objective-v3@1"
	eventSemanticsPolicyVersion       = "event-semantics.objective-v2@1"
	eventSemanticsManifestVersion     = "event-semantic-context-manifest.v3"
	eventSemanticsRouteVersion        = "event-semantic-anchor-routes.v1"
	eventSemanticsRoutePartitionLimit = 50
)

const eventSemanticInputEligibilitySQL = `
	e.event_status = 'confirmed'
	AND e.fact_status = 'verified'
	AND EXISTS (
	    SELECT 1
	    FROM event_sources evidence
	    JOIN raw_documents document ON document.id = evidence.raw_document_id
	    WHERE evidence.event_id = e.id
	      AND COALESCE(evidence.evidence_hash, '') ~ '^[0-9a-f]{64}$'
	      AND COALESCE(btrim(evidence.evidence_statement), '') <> ''
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
	          OR COALESCE(btrim(evidence.evidence_statement), '') = ''
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

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (Store, error) {
	if db == nil {
		return Store{}, errors.New("Event Semantic database is required")
	}
	return Store{db: db}, nil
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

type semanticQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (r Store) ListEligibleEvents(
	ctx context.Context,
	limit int,
	after *eventbiz.EligibleEventCursor,
) ([]eventbiz.EligibleEvent, error) {
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
	result := make([]eventbiz.EligibleEvent, 0)
	for rows.Next() {
		var item eventbiz.EligibleEvent
		if err := rows.Scan(&item.EventID, &item.FirstSeenAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r Store) Context(ctx context.Context, contextLeaseID string) (eventbiz.Context, error) {
	var payload []byte
	err := r.db.QueryRowContext(ctx, `
		SELECT context_manifest
		FROM event_semantic_context_leases
		WHERE id = $1 AND status = 'active' AND lease_expires_at > now()
		  AND context_manifest IS NOT NULL
	`, contextLeaseID).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return eventbiz.Context{}, &eventbiz.NotFoundError{Resource: "active Event Semantic Context Lease"}
	}
	if err != nil {
		return eventbiz.Context{}, err
	}
	var manifest eventbiz.ContextManifest
	if err := json.Unmarshal(payload, &manifest); err != nil {
		return eventbiz.Context{}, err
	}
	return eventSemanticContextFromManifest(ctx, r.db, manifest)
}

func (r Store) SubmissionContext(
	ctx context.Context,
	contextLeaseID string,
	submission eventbiz.Submission,
) (eventbiz.Context, error) {
	result, err := r.Context(ctx, contextLeaseID)
	if err != nil {
		return eventbiz.Context{}, err
	}
	return hydrateEventSemanticSubmissionContext(ctx, r.db, result, submission, false)
}

func hydrateEventSemanticSubmissionContext(
	ctx context.Context,
	query semanticQueryer,
	result eventbiz.Context,
	submission eventbiz.Submission,
	lockSelectedFacts bool,
) (eventbiz.Context, error) {
	lockClause := ""
	if lockSelectedFacts {
		lockClause = " FOR SHARE"
	}
	entityIDs := make([]string, 0, len(submission.EntityLinks))
	for _, link := range submission.EntityLinks {
		entityIDs = append(entityIDs, link.EntityID)
	}
	if len(entityIDs) > 0 {
		rows, err := query.QueryContext(ctx, `
			SELECT id, entity_type, name, canonical_name, array_to_json(aliases), status
			FROM entity_nodes
			WHERE id = ANY($1::uuid[])
			ORDER BY entity_type, canonical_name, id
		`+lockClause, entityIDs)
		if err != nil {
			return eventbiz.Context{}, err
		}
		for rows.Next() {
			var item eventbiz.Entity
			var aliases []byte
			if err := rows.Scan(
				&item.ID, &item.Type, &item.Name, &item.CanonicalName, &aliases, &item.Status,
			); err != nil {
				rows.Close()
				return eventbiz.Context{}, err
			}
			if err := json.Unmarshal(aliases, &item.Aliases); err != nil {
				rows.Close()
				return eventbiz.Context{}, err
			}
			result.Entities = append(result.Entities, item)
		}
		if err := rows.Close(); err != nil {
			return eventbiz.Context{}, err
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
) (eventbiz.Context, error) {
	result := eventbiz.Context{
		ContextLeaseID: contextLeaseID, AgentExecutionID: agentExecutionID,
		WorkerID: workerID, LeaseExpiresAt: leaseExpiresAt.UTC(),
		ManifestContractVersion: eventSemanticsManifestVersion,
		OntologyVersion:         eventSemanticsOntologyVersion,
		PolicyVersion:           eventSemanticsPolicyVersion,
	}
	err := query.QueryRowContext(ctx, `
		SELECT id, title, summary, event_time, event_status, fact_status
		FROM events WHERE id = $1
	`, eventID).Scan(
		&result.Event.ID, &result.Event.Title, &result.Event.Summary, &result.Event.OccurredAt,
		&result.Event.Status, &result.Event.FactStatus,
	)
	if err != nil {
		return eventbiz.Context{}, err
	}
	if result.Evidence, err = eventSemanticEvidence(ctx, query, eventID, true, nil); err != nil {
		return eventbiz.Context{}, err
	}
	if result.EntityTypes, err = eventSemanticEntityTypes(ctx, query, true, nil); err != nil {
		return eventbiz.Context{}, err
	}
	if result.Variables, err = eventSemanticVariables(ctx, query, true, nil); err != nil {
		return eventbiz.Context{}, err
	}
	if err := hydrateEventSemanticPolicy(ctx, query, &result); err != nil {
		return eventbiz.Context{}, err
	}
	result.EventFingerprint, err = eventSemanticFingerprint(result.Event)
	if err != nil {
		return eventbiz.Context{}, err
	}
	result.EvidenceFingerprint, err = eventSemanticFingerprint(result.Evidence)
	if err != nil {
		return eventbiz.Context{}, err
	}
	stableContext := result
	stableContext.LeaseExpiresAt = time.Time{}
	stableContext.ContextFingerprint = ""
	result.ContextFingerprint, err = eventSemanticFingerprint(stableContext)
	if err != nil {
		return eventbiz.Context{}, err
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
) (eventbiz.ContextManifest, error) {
	contextValue, err := buildEventSemanticContext(
		ctx, query, contextLeaseID, eventID, agentExecutionID, workerID, leaseExpiresAt,
	)
	if err != nil {
		return eventbiz.ContextManifest{}, err
	}
	manifest := eventbiz.ContextManifest{
		ContextLeaseID: contextLeaseID, AgentExecutionID: agentExecutionID, WorkerID: workerID,
		LeaseStatus: "active", LeaseExpiresAt: leaseExpiresAt.UTC(),
		ManifestContractVersion: eventSemanticsManifestVersion,
		ContextFingerprint:      contextValue.ContextFingerprint,
		EventID:                 eventID, EventFingerprint: contextValue.EventFingerprint,
		EvidenceFingerprint: contextValue.EvidenceFingerprint,
		OntologyVersion:     contextValue.OntologyVersion, PolicyVersion: contextValue.PolicyVersion,
		Evidence:    make([]eventbiz.EvidenceReference, 0, len(contextValue.Evidence)),
		EntityTypes: make([]eventbiz.VersionReference, 0, len(contextValue.EntityTypes)),
		Variables:   make([]eventbiz.VersionReference, 0, len(contextValue.Variables)),
	}
	for _, evidence := range contextValue.Evidence {
		fingerprint, err := eventSemanticFingerprint(evidence)
		if err != nil {
			return eventbiz.ContextManifest{}, err
		}
		manifest.Evidence = append(manifest.Evidence, eventbiz.EvidenceReference{
			EvidenceID: evidence.ID, Fingerprint: fingerprint,
		})
	}
	for _, definition := range contextValue.EntityTypes {
		manifest.EntityTypes = append(manifest.EntityTypes, eventbiz.VersionReference{Key: definition.TypeKey, Version: definition.Version})
	}
	for _, definition := range contextValue.Variables {
		manifest.Variables = append(manifest.Variables, eventbiz.VersionReference{Key: definition.Key, Version: definition.Version})
	}
	manifest.ManifestFingerprint, err = eventSemanticManifestFingerprint(manifest)
	if err != nil {
		return eventbiz.ContextManifest{}, err
	}
	return manifest, nil
}

func eventSemanticContextFromManifest(
	ctx context.Context,
	query semanticQueryer,
	manifest eventbiz.ContextManifest,
) (eventbiz.Context, error) {
	if err := validateEventSemanticManifestFingerprint(manifest); err != nil {
		return eventbiz.Context{}, err
	}
	result := eventbiz.Context{
		ContextLeaseID: manifest.ContextLeaseID, AgentExecutionID: manifest.AgentExecutionID,
		WorkerID: manifest.WorkerID, LeaseExpiresAt: manifest.LeaseExpiresAt,
		ManifestContractVersion: manifest.ManifestContractVersion,
		ContextFingerprint:      manifest.ContextFingerprint, EventFingerprint: manifest.EventFingerprint,
		EvidenceFingerprint: manifest.EvidenceFingerprint, OntologyVersion: manifest.OntologyVersion,
		PolicyVersion: manifest.PolicyVersion,
	}
	if err := query.QueryRowContext(ctx, `
		SELECT id, title, summary, event_time, event_status, fact_status
		FROM events WHERE id = $1
	`, manifest.EventID).Scan(
		&result.Event.ID, &result.Event.Title, &result.Event.Summary, &result.Event.OccurredAt,
		&result.Event.Status, &result.Event.FactStatus,
	); err != nil {
		return eventbiz.Context{}, err
	}
	evidenceIDs := make([]string, 0, len(manifest.Evidence))
	for _, reference := range manifest.Evidence {
		evidenceIDs = append(evidenceIDs, reference.EvidenceID)
	}
	allEvidence, err := eventSemanticEvidence(ctx, query, manifest.EventID, false, evidenceIDs)
	if err != nil {
		return eventbiz.Context{}, err
	}
	evidenceByID := make(map[string]eventbiz.Evidence, len(allEvidence))
	for _, evidence := range allEvidence {
		evidenceByID[evidence.ID] = evidence
	}
	result.Evidence = make([]eventbiz.Evidence, 0, len(manifest.Evidence))
	for _, reference := range manifest.Evidence {
		evidence, ok := evidenceByID[reference.EvidenceID]
		if !ok {
			return eventbiz.Context{}, &eventbiz.ContextDriftError{Reason: "pinned Event Evidence is unavailable"}
		}
		current, err := eventSemanticFingerprint(evidence)
		if err != nil {
			return eventbiz.Context{}, err
		}
		if current != reference.Fingerprint {
			return eventbiz.Context{}, &eventbiz.ContextDriftError{Reason: "pinned Event Evidence changed"}
		}
		result.Evidence = append(result.Evidence, evidence)
	}
	if result.EntityTypes, err = eventSemanticEntityTypes(ctx, query, false, manifest.EntityTypes); err != nil {
		return eventbiz.Context{}, err
	}
	result.EntityTypes, err = selectEntityTypeReferences(result.EntityTypes, manifest.EntityTypes)
	if err != nil {
		return eventbiz.Context{}, err
	}
	if result.Variables, err = eventSemanticVariables(ctx, query, false, manifest.Variables); err != nil {
		return eventbiz.Context{}, err
	}
	result.Variables, err = selectVariableReferences(result.Variables, manifest.Variables)
	if err != nil {
		return eventbiz.Context{}, err
	}
	if err := hydrateEventSemanticPolicy(ctx, query, &result); err != nil {
		return eventbiz.Context{}, err
	}
	eventFingerprint, err := eventSemanticFingerprint(result.Event)
	if err != nil {
		return eventbiz.Context{}, err
	}
	evidenceFingerprint, err := eventSemanticFingerprint(result.Evidence)
	if err != nil {
		return eventbiz.Context{}, err
	}
	stableContext := result
	stableContext.LeaseExpiresAt = time.Time{}
	stableContext.ContextFingerprint = ""
	contextFingerprint, err := eventSemanticFingerprint(stableContext)
	if err != nil {
		return eventbiz.Context{}, err
	}
	if eventFingerprint != manifest.EventFingerprint || evidenceFingerprint != manifest.EvidenceFingerprint ||
		contextFingerprint != manifest.ContextFingerprint {
		return eventbiz.Context{}, &eventbiz.ContextDriftError{Reason: "pinned Event Semantic Context changed"}
	}
	return result, nil
}

func eventSemanticManifestFingerprint(manifest eventbiz.ContextManifest) (string, error) {
	manifest.ManifestFingerprint = ""
	return eventSemanticFingerprint(manifest)
}

func validateEventSemanticManifestFingerprint(manifest eventbiz.ContextManifest) error {
	fingerprint, err := eventSemanticManifestFingerprint(manifest)
	if err != nil {
		return err
	}
	if manifest.ManifestContractVersion != eventSemanticsManifestVersion ||
		manifest.ManifestFingerprint == "" || fingerprint != manifest.ManifestFingerprint {
		return &eventbiz.ContextDriftError{Reason: "Event Semantic Context Manifest identity changed"}
	}
	return nil
}

func selectEntityTypeReferences(values []eventbiz.EntityTypeDefinition, references []eventbiz.VersionReference) ([]eventbiz.EntityTypeDefinition, error) {
	selected := make([]eventbiz.EntityTypeDefinition, 0, len(references))
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
			return nil, &eventbiz.ContextDriftError{Reason: "pinned Entity Type Definition is unavailable"}
		}
	}
	return selected, nil
}

func selectVariableReferences(values []eventbiz.VariableDefinition, references []eventbiz.VersionReference) ([]eventbiz.VariableDefinition, error) {
	selected := make([]eventbiz.VariableDefinition, 0, len(references))
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
			return nil, &eventbiz.ContextDriftError{Reason: "pinned Variable Definition is unavailable"}
		}
	}
	return selected, nil
}

func selectRuleReferences(values []eventbiz.DirectTransmissionRule, references []eventbiz.VersionReference) ([]eventbiz.DirectTransmissionRule, error) {
	selected := make([]eventbiz.DirectTransmissionRule, 0, len(references))
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
			return nil, &eventbiz.ContextDriftError{Reason: "pinned Direct Transmission Rule is unavailable"}
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
	references []eventbiz.VersionReference,
) ([]eventbiz.EntityTypeDefinition, error) {
	keys, versions := semanticVersionReferenceArrays(references)
	rows, err := query.QueryContext(ctx, `
		SELECT type_key, version, name_zh, name_en, business_definition,
		       array_to_json(inclusion_criteria), array_to_json(exclusion_criteria),
		       event_link_allowed, signal_subject_allowed,
		       array_to_json(allowed_event_roles), status
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
	var result []eventbiz.EntityTypeDefinition
	for rows.Next() {
		var item eventbiz.EntityTypeDefinition
		var inclusionCriteria, exclusionCriteria, allowedRoles []byte
		if err := rows.Scan(
			&item.TypeKey, &item.Version, &item.NameZH, &item.NameEN,
			&item.BusinessDefinition, &inclusionCriteria, &exclusionCriteria,
			&item.EventLinkAllowed, &item.SignalSubjectAllowed,
			&allowedRoles, &item.Status,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(inclusionCriteria, &item.InclusionCriteria); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(exclusionCriteria, &item.ExclusionCriteria); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(allowedRoles, &item.AllowedEventRoles); err != nil {
			return nil, err
		}
		if !validEventSemanticEntityTypeDefinition(item) {
			return nil, errors.New("Event Semantic Entity Type Definition is invalid")
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func validEventSemanticEntityTypeDefinition(item eventbiz.EntityTypeDefinition) bool {
	if strings.TrimSpace(item.TypeKey) == "" || item.Version <= 0 ||
		strings.TrimSpace(item.NameZH) == "" || strings.TrimSpace(item.NameEN) == "" ||
		strings.TrimSpace(item.BusinessDefinition) == "" || item.Status != "active" ||
		len(item.InclusionCriteria) == 0 || len(item.ExclusionCriteria) == 0 ||
		len(item.AllowedEventRoles) == 0 {
		return false
	}
	for _, values := range [][]string{item.InclusionCriteria, item.ExclusionCriteria, item.AllowedEventRoles} {
		for _, value := range values {
			if strings.TrimSpace(value) == "" {
				return false
			}
		}
	}
	return true
}

func eventSemanticEvidence(
	ctx context.Context,
	query semanticQueryer,
	eventID string,
	includeAll bool,
	evidenceIDs []string,
) ([]eventbiz.Evidence, error) {
	rows, err := query.QueryContext(ctx, `
		SELECT es.id, es.evidence_hash, es.evidence_statement, es.source_level,
		       es.evidence_relation, array_to_json(es.supports_fields), es.raw_document_id,
		       rd.source_name, rd.source_type, rd.source_url, rd.title,
		       rd.published_at, rd.collected_at,
		       GREATEST(COALESCE(rd.published_at, rd.collected_at), rd.collected_at),
		       es.created_at,
		       COALESCE(NULLIF(event.fact_payload ->> 'statement_source', ''), '')
		FROM event_sources es
		JOIN raw_documents rd ON rd.id = es.raw_document_id
		JOIN events event ON event.id = es.event_id
		WHERE es.event_id = $1 AND ($2 OR es.id = ANY($3::uuid[]))
		ORDER BY es.created_at, es.id
	`, eventID, includeAll, evidenceIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []eventbiz.Evidence
	for rows.Next() {
		var item eventbiz.Evidence
		var supported []byte
		if err := rows.Scan(
			&item.ID, &item.Hash, &item.Statement, &item.SourceLevel, &item.Relation,
			&supported, &item.RawDocumentID, &item.SourceName,
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

func eventSemanticEntities(ctx context.Context, query semanticQueryer) ([]eventbiz.Entity, error) {
	rows, err := query.QueryContext(ctx, `
		SELECT id, entity_type, name, canonical_name, array_to_json(aliases), status
		FROM entity_nodes
		WHERE status = 'active'
		  AND entity_type IN (
		      SELECT type_key FROM entity_type_definitions
		      WHERE status = 'active' AND event_link_allowed
		  )
		ORDER BY entity_type, canonical_name, id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []eventbiz.Entity
	for rows.Next() {
		var item eventbiz.Entity
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

func eventSemanticRelations(ctx context.Context, query semanticQueryer) ([]eventbiz.EntityRelation, error) {
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
	var result []eventbiz.EntityRelation
	for rows.Next() {
		var item eventbiz.EntityRelation
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
	references []eventbiz.VersionReference,
) ([]eventbiz.VariableDefinition, error) {
	keys, versions := semanticVersionReferenceArrays(references)
	rows, err := query.QueryContext(ctx, `
		SELECT definition.variable_key, definition.version, definition.name_zh, definition.name_en,
		       definition.domain, definition.business_definition, definition.value_type, definition.status,
		       array_to_json(definition.allowed_directions),
		       array_to_json(definition.allowed_units),
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
	var result []eventbiz.VariableDefinition
	for rows.Next() {
		var item eventbiz.VariableDefinition
		var directions, units, applicable []byte
		if err := rows.Scan(
			&item.Key, &item.Version, &item.NameZH, &item.NameEN, &item.Domain,
			&item.BusinessDefinition, &item.ValueType, &item.Status, &directions, &units, &applicable,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(directions, &item.AllowedDirections); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(units, &item.AllowedUnits); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(applicable, &item.ApplicableEntityTypes); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func hydrateEventSemanticPolicy(
	ctx context.Context,
	query semanticQueryer,
	result *eventbiz.Context,
) error {
	var payload []byte
	if err := query.QueryRowContext(ctx, `
		SELECT policy
		FROM event_semantic_acceptance_policies
		WHERE policy_key = 'event-semantics.objective-v2'
		  AND version = 1
		  AND status = 'active'
	`).Scan(&payload); err != nil {
		return err
	}
	var policy struct {
		AssertionModalities []string                     `json:"assertion_modalities"`
		MeasurementContract eventbiz.MeasurementContract `json:"measurement_contract"`
	}
	if err := json.Unmarshal(payload, &policy); err != nil {
		return err
	}
	if len(policy.AssertionModalities) == 0 ||
		policy.MeasurementContract.Representation != "evidence_grounded_narrative" ||
		policy.MeasurementContract.MaxItemsPerSignal <= 0 ||
		policy.MeasurementContract.MaxTextCharacters <= 0 ||
		!policy.MeasurementContract.RequiresEvidenceIDs ||
		policy.MeasurementContract.NumericValidation {
		return errors.New("Event Semantic V2 acceptance policy is invalid")
	}
	result.AssertionModalities = policy.AssertionModalities
	result.MeasurementContract = policy.MeasurementContract
	return nil
}

func eventSemanticRules(
	ctx context.Context,
	query semanticQueryer,
	includeAll bool,
	references []eventbiz.VersionReference,
) ([]eventbiz.DirectTransmissionRule, error) {
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
	var result []eventbiz.DirectTransmissionRule
	for rows.Next() {
		var item eventbiz.DirectTransmissionRule
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
	references []eventbiz.VersionReference,
) ([]string, []int32) {
	keys := make([]string, 0, len(references))
	versions := make([]int32, 0, len(references))
	for _, reference := range references {
		keys = append(keys, reference.Key)
		versions = append(versions, int32(reference.Version))
	}
	return keys, versions
}

func (r Store) Resolve(
	ctx context.Context,
	contextLeaseID string,
	mentions []eventbiz.EntityMention,
) ([]eventbiz.EntityResolution, error) {
	if _, err := r.Context(ctx, contextLeaseID); err != nil {
		return nil, err
	}
	result := make([]eventbiz.EntityResolution, 0, len(mentions))
	for _, mention := range mentions {
		resolution := eventbiz.EntityResolution{Mention: mention.Mention}
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
			var entity eventbiz.Entity
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

func (r Store) SearchDirectTargets(
	ctx context.Context,
	contextLeaseID string,
	subjectEntityID string,
	allowedTargetTypes []string,
) ([]eventbiz.DirectTarget, error) {
	if _, err := r.Context(ctx, contextLeaseID); err != nil {
		return nil, err
	}
	var result []eventbiz.DirectTarget
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
		var item eventbiz.DirectTarget
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

func (r Store) ListResolutionRoutes(
	ctx context.Context,
	contextLeaseID string,
	targetEntityType string,
) ([]eventbiz.ResolutionRoute, error) {
	if _, err := r.Context(ctx, contextLeaseID); err != nil {
		return nil, err
	}
	if targetEntityType != "chain_node" {
		return nil, &eventbiz.ValidationError{Reason: "target Entity Type has no bounded resolution route"}
	}
	industryPartitions, industryLabels, err := r.eventSemanticIndustryPartitions(ctx)
	if err != nil {
		return nil, err
	}
	conceptPartitions, conceptLabels, err := r.eventSemanticConceptPartitions(ctx)
	if err != nil {
		return nil, err
	}
	var routes []eventbiz.ResolutionRoute
	if len(industryPartitions) > 0 {
		routes = append(routes, eventbiz.ResolutionRoute{
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
		routes = append(routes, eventbiz.ResolutionRoute{
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

func (r Store) eventSemanticIndustryPartitions(ctx context.Context) ([]string, map[string]string, error) {
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

func (r Store) eventSemanticConceptPartitions(ctx context.Context) ([]string, map[string]string, error) {
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

func (r Store) ListResolutionAnchors(
	ctx context.Context,
	contextLeaseID string,
	routeID string,
	partition string,
	parentAnchorIDs []string,
	limit int,
	after *eventbiz.ResolutionKeyset,
) ([]eventbiz.ResolutionAnchor, error) {
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
			return nil, &eventbiz.ValidationError{Reason: "Industry partition is unknown, inactive or unapproved"}
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
				return nil, &eventbiz.ValidationError{Reason: "parent_anchor_ids contain an unknown or out-of-partition Industry"}
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
			return nil, &eventbiz.ValidationError{Reason: "Concept partition is unknown, inactive or unapproved"}
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
		return nil, &eventbiz.ValidationError{Reason: "route_id is not supported"}
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]eventbiz.ResolutionAnchor, 0)
	for rows.Next() {
		var item eventbiz.ResolutionAnchor
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

func (r Store) ResolveChainNodeCandidates(
	ctx context.Context,
	contextLeaseID string,
	routeID string,
	anchorEntityIDs []string,
	limit int,
	after *eventbiz.ResolutionKeyset,
) ([]eventbiz.ResolutionCandidate, error) {
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
		return nil, &eventbiz.ValidationError{Reason: "route_id is not supported"}
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
		return nil, &eventbiz.ValidationError{Reason: "anchor_entity_ids contain an unknown, inactive, wrong-type or unapproved anchor"}
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
	result := make([]eventbiz.ResolutionCandidate, 0)
	for rows.Next() {
		var item eventbiz.ResolutionCandidate
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
	manifest eventbiz.Context,
	submission eventbiz.Submission,
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
			return &eventbiz.ContextDriftError{Reason: "selected anchor route is no longer valid"}
		}
		if receipt.RouteContractVersion != manifest.RouteContractVersion ||
			receipt.TargetEntityID != link.EntityID {
			return &eventbiz.ContextDriftError{Reason: "selected anchor route version or target changed"}
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
			return &eventbiz.ContextDriftError{Reason: "selected anchor path changed or disappeared"}
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
			return &eventbiz.ContextDriftError{Reason: "selected anchor path changed after candidate resolution"}
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
	receipt eventbiz.ResolutionReceipt,
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

func (r Store) ReplaySubmission(
	ctx context.Context,
	executionID string,
	canonicalPayloadHash string,
) (eventbiz.SubmissionResult, bool, error) {
	existing, found, err := queryEventSemanticSubmission(ctx, r.db.QueryRowContext(ctx, `
		SELECT id,event_id,status,canonical_payload_hash,decision_summary
		FROM event_semantic_submissions
		WHERE agent_execution_id = $1
	`, executionID))
	if err != nil || !found {
		return existing, found, err
	}
	if existing.CanonicalPayloadHash != canonicalPayloadHash {
		return eventbiz.SubmissionResult{}, false, &eventbiz.ConflictError{
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
	submission eventbiz.Submission,
	precheck eventbiz.PrecheckResult,
) error {
	linkDecisions := decisionsByKey(precheck.EntityLinks)
	linkIDs := make(map[string]string)
	for _, candidate := range submission.EntityLinks {
		decision := linkDecisions[candidate.Key]
		if decision.Status == eventbiz.StatusRejected {
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
	for _, candidate := range submission.VariableSignals {
		decision := signalDecisions[candidate.Key]
		linkID := linkIDs[candidate.SubjectLinkKey]
		if decision.Status == eventbiz.StatusRejected || linkID == "" {
			continue
		}
		id := uuid.NewString()
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
			primaryEvidenceID := measurement.EvidenceIDs[0]
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO variable_signal_measurements(
				    id,variable_signal_id,measurement_role,value_shape,raw_value,raw_lower,raw_upper,
				    raw_unit,canonical_value,canonical_lower,canonical_upper,canonical_unit,currency,
				    scale,comparison_basis,comparison_period,raw_text,is_approximate,evidence_id,evidence_ids
				) VALUES ($1,$2,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,$3,false,$4,$5)
			`, uuid.NewString(), id, measurement.Text, primaryEvidenceID, measurement.EvidenceIDs,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r Store) GetEventSemantics(ctx context.Context, eventID string) (eventbiz.EventSemanticsResult, error) {
	var exists bool
	if err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM events WHERE id = $1)`, eventID).Scan(&exists); err != nil {
		return eventbiz.EventSemanticsResult{}, err
	}
	if !exists {
		return eventbiz.EventSemanticsResult{}, &eventbiz.NotFoundError{Resource: "Event"}
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
		return eventbiz.EventSemanticsResult{}, err
	}
	defer rows.Close()
	result := eventbiz.EventSemanticsResult{EventID: eventID}
	for rows.Next() {
		var run eventbiz.SubmissionResult
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
			return eventbiz.EventSemanticsResult{}, err
		}
		if err := json.Unmarshal(decisions, &run.Precheck); err != nil {
			return eventbiz.EventSemanticsResult{}, err
		}
		run.CandidateSnapshot = append(json.RawMessage(nil), candidateSnapshot...)
		if run.ReviewSnapshots, err = r.eventSemanticReviewSnapshots(ctx, run.SubmissionID); err != nil {
			return eventbiz.EventSemanticsResult{}, err
		}
		if err := r.populateSemanticRecordIDs(ctx, run.SubmissionID, &run.Precheck); err != nil {
			return eventbiz.EventSemanticsResult{}, err
		}
		result.Submissions = append(result.Submissions, run)
	}
	return result, rows.Err()
}

func (r Store) eventSemanticReviewSnapshots(
	ctx context.Context,
	runID string,
) ([]eventbiz.ReviewSnapshot, error) {
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
	var result []eventbiz.ReviewSnapshot
	for rows.Next() {
		var item eventbiz.ReviewSnapshot
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

func (r Store) populateSemanticRecordIDs(
	ctx context.Context,
	runID string,
	precheck *eventbiz.PrecheckResult,
) error {
	groups := []struct {
		table string
		items *[]eventbiz.CandidateDecision
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

func existingEventSemanticSubmission(ctx context.Context, tx *sql.Tx, executionID string) (eventbiz.SubmissionResult, bool, error) {
	return queryEventSemanticSubmission(ctx, tx.QueryRowContext(ctx, `
		SELECT id,event_id,status,canonical_payload_hash,decision_summary
		FROM event_semantic_submissions
		WHERE agent_execution_id = $1
	`, executionID))
}

func eventSemanticSubmissionByID(ctx context.Context, tx *sql.Tx, runID string) (eventbiz.SubmissionResult, bool, error) {
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

func queryEventSemanticSubmission(_ context.Context, row semanticSubmissionRow) (eventbiz.SubmissionResult, bool, error) {
	var result eventbiz.SubmissionResult
	var decisions []byte
	err := row.Scan(&result.SubmissionID, &result.EventID, &result.Status, &result.CanonicalPayloadHash, &decisions)
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
		return eventbiz.SubmissionResult{}, false, nil
	}
	if err != nil {
		return eventbiz.SubmissionResult{}, false, err
	}
	if err := json.Unmarshal(decisions, &result.Precheck); err != nil {
		return eventbiz.SubmissionResult{}, false, err
	}
	return result, true, nil
}

func decisionsByKey(items []eventbiz.CandidateDecision) map[string]eventbiz.CandidateDecision {
	result := make(map[string]eventbiz.CandidateDecision, len(items))
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

const HistoricalEventSemanticManifestVersion = "event-semantic-history-audit.v1"

type HistoricalEventSemanticManifest struct {
	Version         string    `json:"version"`
	GeneratedAt     time.Time `json:"generated_at"`
	ValidEventIDs   []string  `json:"valid_event_ids"`
	InvalidEventIDs []string  `json:"invalid_event_ids"`
}

func AuditHistoricalEventSemantics(
	ctx context.Context,
	db *sql.DB,
	generatedAt time.Time,
) (HistoricalEventSemanticManifest, error) {
	if db == nil {
		return HistoricalEventSemanticManifest{}, fmt.Errorf(
			"Data database is required",
		)
	}
	rows, err := db.QueryContext(ctx, fmt.Sprintf(`
		SELECT e.id, (%s) AS input_valid
		FROM events e
		WHERE EXISTS (
		    SELECT 1
		    FROM event_sources historical_evidence
		    WHERE historical_evidence.event_id = e.id
		      AND historical_evidence.contract_version = 1
		)
		ORDER BY e.first_seen_at, e.id
	`, eventSemanticInputEligibilitySQL))
	if err != nil {
		return HistoricalEventSemanticManifest{}, fmt.Errorf(
			"audit historical Event Semantic inputs: %w", err,
		)
	}
	defer rows.Close()
	manifest := HistoricalEventSemanticManifest{
		Version:         HistoricalEventSemanticManifestVersion,
		GeneratedAt:     generatedAt.UTC(),
		ValidEventIDs:   []string{},
		InvalidEventIDs: []string{},
	}
	for rows.Next() {
		var eventID string
		var valid bool
		if err := rows.Scan(&eventID, &valid); err != nil {
			return HistoricalEventSemanticManifest{}, fmt.Errorf(
				"scan historical Event Semantic input: %w", err,
			)
		}
		if valid {
			manifest.ValidEventIDs = append(manifest.ValidEventIDs, eventID)
		} else {
			manifest.InvalidEventIDs = append(manifest.InvalidEventIDs, eventID)
		}
	}
	if err := rows.Err(); err != nil {
		return HistoricalEventSemanticManifest{}, fmt.Errorf(
			"audit historical Event Semantic inputs: %w", err,
		)
	}
	return manifest, nil
}
