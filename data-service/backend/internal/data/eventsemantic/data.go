package eventsemantic

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"bytes"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	entitybiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity"
	eventbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/eventsemantic"
)

const (
	eventSemanticsOntologyVersion     = "event-semantics.objective-v3@1"
	eventSemanticsPolicyVersion       = "event-semantics.objective-v2@1"
	eventSemanticsManifestVersion     = "event-semantic-context-manifest.v4"
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

func invalidPersistedEventSemantic(resource string) error {
	return fmt.Errorf("persisted Event Semantic %s is invalid", resource)
}

func validPersistedUUID(value string) bool {
	_, err := uuid.Parse(value)
	return err == nil
}

func validPersistedObjectID(value string) bool {
	return validPersistedUUID(value) || entitybiz.IsCountryID(value) || entitybiz.IsOrganizationID(value)
}

func validPersistedSHA256(value string) bool {
	if len(value) != sha256.Size*2 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validPersistedReviewStatus(status eventbiz.ReviewStatus) bool {
	switch status {
	case eventbiz.StatusPendingReview, eventbiz.StatusNeedsReanalysis, eventbiz.StatusQuarantined,
		eventbiz.StatusAccepted, eventbiz.StatusRejected, eventbiz.StatusSuperseded:
		return true
	default:
		return false
	}
}

func validPersistedContextLeaseStatus(status string) bool {
	switch status {
	case "active", "consumed", "expired":
		return true
	default:
		return false
	}
}

func validatePersistedStoredContextLease(lease eventbiz.StoredContextLease) error {
	if !validPersistedUUID(lease.ID) || !validPersistedUUID(lease.EventID) ||
		(lease.SupersedesSubmissionID != "" && !validPersistedUUID(lease.SupersedesSubmissionID)) ||
		strings.TrimSpace(lease.AgentExecutionID) == "" || strings.TrimSpace(lease.WorkerID) == "" ||
		!validPersistedContextLeaseStatus(lease.Status) || lease.LeaseExpiresAt.IsZero() ||
		(lease.SubmissionStatus != "" && !validPersistedReviewStatus(lease.SubmissionStatus)) {
		return invalidPersistedEventSemantic("Context Lease")
	}
	return nil
}

func validatePersistedLeaseEventState(state eventbiz.LeaseEventState) error {
	validEventStatus := state.EventStatus == "candidate" || state.EventStatus == "confirmed" || state.EventStatus == "rejected"
	validFactStatus := state.FactStatus == "unverified" || state.FactStatus == "verified" || state.FactStatus == "disputed"
	if !validPersistedUUID(state.EventID) || !validEventStatus || !validFactStatus {
		return invalidPersistedEventSemantic("Event state")
	}
	return nil
}

func validatePersistedSubmissionLeaseState(state eventbiz.SubmissionLeaseState) error {
	if !validPersistedUUID(state.EventID) || strings.TrimSpace(state.AgentExecutionID) == "" ||
		!validPersistedContextLeaseStatus(state.Status) || state.LeaseExpiresAt.IsZero() ||
		(state.SupersedesSubmissionID != "" && !validPersistedUUID(state.SupersedesSubmissionID)) {
		return invalidPersistedEventSemantic("Submission Context Lease")
	}
	return nil
}

func validatePersistedSubmissionReference(reference eventbiz.SubmissionReference) error {
	if !validPersistedUUID(reference.SubmissionID) || !validPersistedUUID(reference.EventID) ||
		!validPersistedUUID(reference.ContextLeaseID) || !validPersistedReviewStatus(reference.Status) {
		return invalidPersistedEventSemantic("Submission reference")
	}
	return nil
}

func validatePersistedReviewIdentity(identity eventbiz.ReviewIdentity) error {
	if strings.TrimSpace(identity.AgentExecutionID) == "" ||
		!validPersistedSHA256(identity.ReviewerPromptHash) || strings.TrimSpace(identity.ReviewerModel) == "" ||
		(identity.AdjudicatorPromptHash == "") != (identity.AdjudicatorModel == "") ||
		(identity.AdjudicatorPromptHash != "" && !validPersistedSHA256(identity.AdjudicatorPromptHash)) {
		return invalidPersistedEventSemantic("Review identity")
	}
	return nil
}

func validPersistedStringSet(values []string, required bool) bool {
	if required && len(values) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return false
		}
		if _, duplicate := seen[value]; duplicate {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func validPersistedUUIDSet(values []string, required bool) bool {
	if !validPersistedStringSet(values, required) {
		return false
	}
	for _, value := range values {
		if !validPersistedUUID(value) {
			return false
		}
	}
	return true
}

func validatePersistedEvent(item eventbiz.Event) error {
	if !validPersistedUUID(item.ID) || strings.TrimSpace(item.Title) == "" ||
		strings.TrimSpace(item.Summary) == "" || item.Status != "confirmed" || item.FactStatus != "verified" {
		return invalidPersistedEventSemantic("Event")
	}
	return nil
}

func validatePersistedEvidence(item eventbiz.Evidence) error {
	allowedFields := map[string]struct{}{
		"title": {}, "factual_summary": {}, "occurred_at": {}, "fact_payload": {},
	}
	expectedHash := fmt.Sprintf("%x", sha256.Sum256([]byte(item.Statement)))
	if !validPersistedUUID(item.ID) || !validPersistedUUID(item.RawDocumentID) ||
		!validPersistedSHA256(item.Hash) || strings.TrimSpace(item.Statement) == "" ||
		item.Hash != expectedHash ||
		(item.SourceLevel != "primary" && item.SourceLevel != "secondary") ||
		(item.Relation != "supports" && item.Relation != "contradicts" && item.Relation != "context") ||
		strings.TrimSpace(item.SourceName) == "" || strings.TrimSpace(item.SourceType) == "" ||
		strings.TrimSpace(item.Title) == "" || item.FirstSeenAt.IsZero() ||
		item.KnowledgeAvailableAt.IsZero() || item.AcceptedAt.IsZero() ||
		!validPersistedStringSet(item.SupportsFields, item.Relation != "context") {
		return invalidPersistedEventSemantic("Evidence")
	}
	for _, field := range item.SupportsFields {
		if _, ok := allowedFields[field]; !ok {
			return invalidPersistedEventSemantic("Evidence supports_fields")
		}
	}
	return nil
}

func validatePersistedEntity(item eventbiz.Entity) error {
	if !validPersistedObjectID(item.ID) || strings.TrimSpace(item.Type) == "" ||
		(entitybiz.IsCountryID(item.ID) != (item.Type == entitybiz.ObjectTypeCountry)) ||
		(entitybiz.IsOrganizationID(item.ID) != (item.Type == entitybiz.ObjectTypeOrganization)) ||
		strings.TrimSpace(item.Name) == "" || strings.TrimSpace(item.CanonicalName) == "" ||
		item.Status != "active" || !validPersistedStringSet(item.Aliases, false) {
		return invalidPersistedEventSemantic("Entity")
	}
	return nil
}

func validatePersistedEntityRelation(item eventbiz.EntityRelation) error {
	if !validPersistedUUID(item.ID) || !validPersistedUUID(item.FromEntityID) ||
		!validPersistedUUID(item.ToEntityID) || strings.TrimSpace(item.Type) == "" || item.Status != "active" {
		return invalidPersistedEventSemantic("Entity Relation")
	}
	return nil
}

func validatePersistedVariableDefinition(item eventbiz.VariableDefinition) error {
	allowedDirections := map[string]struct{}{
		"increase": {}, "decrease": {}, "unchanged": {}, "mixed": {}, "uncertain": {},
	}
	if strings.TrimSpace(item.Key) == "" || item.Version <= 0 || strings.TrimSpace(item.NameZH) == "" ||
		strings.TrimSpace(item.NameEN) == "" || strings.TrimSpace(item.Domain) == "" ||
		strings.TrimSpace(item.BusinessDefinition) == "" || strings.TrimSpace(item.ValueType) == "" ||
		item.Status != "active" || !validPersistedStringSet(item.AllowedDirections, true) ||
		!validPersistedStringSet(item.AllowedUnits, false) ||
		!validPersistedStringSet(item.ApplicableEntityTypes, true) {
		return invalidPersistedEventSemantic("Variable Definition")
	}
	for _, direction := range item.AllowedDirections {
		if _, ok := allowedDirections[direction]; !ok {
			return invalidPersistedEventSemantic("Variable Definition direction")
		}
	}
	return nil
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
		if !validPersistedUUID(item.EventID) || item.FirstSeenAt.IsZero() {
			return nil, invalidPersistedEventSemantic("eligible Event")
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
	countryIDs := make([]string, 0, len(submission.EntityLinks))
	organizationIDs := make([]string, 0, len(submission.EntityLinks))
	for _, link := range submission.EntityLinks {
		if entitybiz.IsCountryID(link.EntityID) {
			countryIDs = append(countryIDs, link.EntityID)
		} else if entitybiz.IsOrganizationID(link.EntityID) {
			organizationIDs = append(organizationIDs, link.EntityID)
		} else {
			entityIDs = append(entityIDs, link.EntityID)
		}
	}
	if len(entityIDs)+len(countryIDs)+len(organizationIDs) > 0 {
		rows, err := query.QueryContext(ctx, `
			WITH selected_entities AS MATERIALIZED (
				SELECT id::text, entity_type::text, name, canonical_name,
				       array_to_json(aliases) aliases, status::text
				FROM entity_nodes
				WHERE id = ANY($1::uuid[])
				`+lockClause+`
			), selected_countries AS MATERIALIZED (
				SELECT id, 'country' object_type, name, name canonical_name,
				       array_to_json(ARRAY[name_en]) aliases, 'active' status
				FROM countries
				WHERE id = ANY($2::text[])
				`+lockClause+`
			), selected_organizations AS MATERIALIZED (
				SELECT id, 'organization' object_type, name, name canonical_name,
				       array_to_json(ARRAY[name_en]) aliases, 'active' status
				FROM organizations
				WHERE id = ANY($3::text[])
				`+lockClause+`
			)
			SELECT * FROM selected_entities
			UNION ALL
			SELECT * FROM selected_countries
			UNION ALL
			SELECT * FROM selected_organizations
			ORDER BY 2, 4, 1
		`, entityIDs, countryIDs, organizationIDs)
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
			if err := validatePersistedEntity(item); err != nil {
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
	if err := validatePersistedEvent(result.Event); err != nil {
		return eventbiz.Context{}, err
	}
	if result.Evidence, err = eventSemanticEvidence(ctx, query, eventID, true, nil); err != nil {
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
		Evidence:  make([]eventbiz.EvidenceReference, 0, len(contextValue.Evidence)),
		Variables: make([]eventbiz.VersionReference, 0, len(contextValue.Variables)),
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
	if err := validatePersistedEvent(result.Event); err != nil {
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
	if err := validatePersistedEventSemanticManifest(manifest); err != nil {
		return err
	}
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

func validatePersistedEventSemanticManifest(manifest eventbiz.ContextManifest) error {
	if !validPersistedUUID(manifest.ContextLeaseID) || !validPersistedUUID(manifest.EventID) ||
		strings.TrimSpace(manifest.AgentExecutionID) == "" || strings.TrimSpace(manifest.WorkerID) == "" ||
		manifest.LeaseStatus != "active" || manifest.LeaseExpiresAt.IsZero() ||
		manifest.ManifestContractVersion != eventSemanticsManifestVersion ||
		manifest.OntologyVersion != eventSemanticsOntologyVersion || manifest.PolicyVersion != eventSemanticsPolicyVersion ||
		!validPersistedSHA256(manifest.ManifestFingerprint) || !validPersistedSHA256(manifest.ContextFingerprint) ||
		!validPersistedSHA256(manifest.EventFingerprint) || !validPersistedSHA256(manifest.EvidenceFingerprint) ||
		len(manifest.Evidence) == 0 || len(manifest.Variables) == 0 {
		return invalidPersistedEventSemantic("Context Manifest")
	}
	evidenceIDs := make([]string, 0, len(manifest.Evidence))
	for _, reference := range manifest.Evidence {
		if !validPersistedSHA256(reference.Fingerprint) {
			return invalidPersistedEventSemantic("Context Manifest Evidence reference")
		}
		evidenceIDs = append(evidenceIDs, reference.EvidenceID)
	}
	if !validPersistedUUIDSet(evidenceIDs, true) ||
		!validPersistedVersionReferences(manifest.Variables, true) ||
		!validPersistedVersionReferences(manifest.Rules, false) {
		return invalidPersistedEventSemantic("Context Manifest reference")
	}
	return nil
}

func validPersistedVersionReferences(references []eventbiz.VersionReference, required bool) bool {
	if required && len(references) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(references))
	for _, reference := range references {
		key := strings.TrimSpace(reference.Key)
		if key == "" || reference.Version <= 0 {
			return false
		}
		identity := fmt.Sprintf("%s@%d", key, reference.Version)
		if _, duplicate := seen[identity]; duplicate {
			return false
		}
		seen[identity] = struct{}{}
	}
	return true
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

func eventSemanticFingerprint(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(payload)), nil
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
		if err := validatePersistedEvidence(item); err != nil {
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
		if err := validatePersistedVariableDefinition(item); err != nil {
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
		!validPersistedStringSet(policy.AssertionModalities, true) ||
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
			SELECT candidate.id, candidate.object_type, candidate.name,
			       candidate.canonical_name, candidate.aliases, candidate.status
			FROM (
				SELECT id::text, entity_type::text object_type, name, canonical_name,
				       array_to_json(aliases) aliases, status::text
				FROM entity_nodes
				WHERE status = 'active'
				  AND entity_type = ANY($1)
				  AND (lower(name) = lower($2) OR lower(canonical_name) = lower($2)
				       OR EXISTS (SELECT 1 FROM unnest(aliases) alias WHERE lower(alias) = lower($2)))
				UNION ALL
				SELECT id, 'country', name, name, array_to_json(ARRAY[name_en]), 'active'
				FROM countries
				WHERE 'country' = ANY($1)
				  AND (lower(name) = lower($2) OR lower(name_en) = lower($2))
				UNION ALL
				SELECT id, 'organization', name, name, array_to_json(ARRAY[name_en]), 'active'
				FROM organizations
				WHERE 'organization' = ANY($1)
				  AND (lower(name) = lower($2) OR lower(name_en) = lower($2))
			) candidate
			ORDER BY (lower(candidate.canonical_name) = lower($2)) DESC,
			         candidate.canonical_name, candidate.id
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
			if err := validatePersistedEntity(entity); err != nil {
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
		if err := validatePersistedEntity(item.Entity); err != nil {
			return nil, err
		}
		if err := validatePersistedEntityRelation(item.Relation); err != nil {
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
		if !validPersistedUUID(partition) || strings.TrimSpace(label) == "" {
			return nil, nil, invalidPersistedEventSemantic("Industry partition")
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
		if strings.TrimSpace(partition) == "" {
			return nil, nil, invalidPersistedEventSemantic("Concept partition")
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
		if err := validatePersistedEntity(item.Entity); err != nil {
			return nil, err
		}
		if strings.TrimSpace(item.Partition) == "" || strings.TrimSpace(item.Description) == "" ||
			strings.TrimSpace(item.HierarchyIdentity) == "" {
			return nil, invalidPersistedEventSemantic("Resolution Anchor")
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
		if err := validatePersistedEntity(item.Entity); err != nil {
			return nil, err
		}
		if !validPersistedUUID(item.Receipt.AnchorEntityID) ||
			!validPersistedUUID(item.Receipt.IndustryChainEntityID) ||
			!validPersistedUUID(item.Receipt.MappingRelationID) ||
			!validPersistedUUIDSet(item.MatchedAnchorEntityIDs, true) ||
			strings.TrimSpace(item.Description) == "" || strings.TrimSpace(item.IndustryChainEntityName) == "" ||
			membershipUpdatedAt.IsZero() || anchorUpdatedAt.IsZero() || chainUpdatedAt.IsZero() ||
			mappingUpdatedAt.IsZero() || targetUpdatedAt.IsZero() {
			return nil, invalidPersistedEventSemantic("Resolution Candidate")
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
		entityID, countryID, organizationID := any(candidate.EntityID), any(nil), any(nil)
		if candidate.ProjectedEntityType == entitybiz.ObjectTypeCountry {
			entityID, countryID = nil, candidate.EntityID
		} else if candidate.ProjectedEntityType == entitybiz.ObjectTypeOrganization {
			entityID, organizationID = nil, candidate.EntityID
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO event_entity_links(
			    id,event_id,entity_id,country_id,organization_id,entity_role,assign_source,review_status,evidence_note,
			    semantic_submission_id,candidate_key,resolved_mention,resolution_method,
			    resolution_confidence,evidence_ids,provenance,reason_code
			) VALUES ($1,$2,$3,$4,$5,$6,'ai',$7,'',$8,$9,$10,$11,$12,$13,'semantic',$14)
		`, id, submission.EventID, entityID, countryID, organizationID, candidate.EntityRole, decision.Status,
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

func validatePersistedCandidateDecisions(items []eventbiz.CandidateDecision) (map[string]struct{}, error) {
	keys := make(map[string]struct{}, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item.CandidateKey)
		if key == "" || !validPersistedReviewStatus(item.Status) {
			return nil, invalidPersistedEventSemantic("Candidate Decision")
		}
		if _, duplicate := keys[key]; duplicate {
			return nil, invalidPersistedEventSemantic("Candidate Decision set")
		}
		keys[key] = struct{}{}
	}
	return keys, nil
}

func validatePersistedEntityLinkCandidate(item eventbiz.EntityLinkCandidate) error {
	if strings.TrimSpace(item.Key) == "" || strings.TrimSpace(item.Mention) == "" ||
		!validPersistedObjectID(item.EntityID) || strings.TrimSpace(item.ProjectedEntityType) == "" ||
		strings.TrimSpace(item.EntityRole) == "" || strings.TrimSpace(item.ResolutionMethod) == "" ||
		!validPersistedUUIDSet(item.EvidenceIDs, true) {
		return invalidPersistedEventSemantic("Entity Link candidate")
	}
	return nil
}

func validatePersistedVariableSignalCandidate(item eventbiz.VariableSignalCandidate) error {
	allowedDirections := map[string]struct{}{
		"increase": {}, "decrease": {}, "unchanged": {}, "mixed": {}, "uncertain": {},
	}
	allowedModalities := map[string]struct{}{
		"actual": {}, "stated_intent": {}, "source_forecast": {},
	}
	if strings.TrimSpace(item.Key) == "" || strings.TrimSpace(item.SubjectLinkKey) == "" ||
		strings.TrimSpace(item.VariableKey) == "" || item.VariableVersion <= 0 ||
		!validPersistedUUIDSet(item.EvidenceIDs, true) {
		return invalidPersistedEventSemantic("Variable Signal candidate")
	}
	if _, ok := allowedDirections[item.Direction]; !ok {
		return invalidPersistedEventSemantic("Variable Signal direction")
	}
	if _, ok := allowedModalities[item.AssertionModality]; !ok {
		return invalidPersistedEventSemantic("Variable Signal modality")
	}
	if item.ValidFrom != nil && item.ValidUntil != nil && item.ValidUntil.Before(*item.ValidFrom) {
		return invalidPersistedEventSemantic("Variable Signal validity range")
	}
	if item.ForecastPeriodStart != nil && item.ForecastPeriodEnd != nil && item.ForecastPeriodEnd.Before(*item.ForecastPeriodStart) {
		return invalidPersistedEventSemantic("Variable Signal forecast range")
	}
	for _, measurement := range item.Measurements {
		if strings.TrimSpace(measurement.Text) == "" || !validPersistedUUIDSet(measurement.EvidenceIDs, true) {
			return invalidPersistedEventSemantic("Variable Signal measurement")
		}
	}
	return nil
}

func validatePersistedDirectImpactCandidate(item eventbiz.DirectImpactCandidate) error {
	if strings.TrimSpace(item.Key) == "" || strings.TrimSpace(item.SourceSignalKey) == "" ||
		!validPersistedUUID(item.TargetEntityID) || strings.TrimSpace(item.AffectedVariableKey) == "" ||
		item.AffectedVariableVersion <= 0 || strings.TrimSpace(item.AffectedDirection) == "" ||
		(item.DerivationType != "event_explicit" && item.DerivationType != "rule_inferred") ||
		!validPersistedUUIDSet(item.EvidenceIDs, true) {
		return invalidPersistedEventSemantic("Direct Impact candidate")
	}
	return nil
}

func validatePersistedPrecheck(precheck eventbiz.PrecheckResult) error {
	decisionGroups := [][]eventbiz.CandidateDecision{
		precheck.EntityLinks, precheck.VariableSignals, precheck.DirectImpacts,
	}
	for _, group := range decisionGroups {
		if _, err := validatePersistedCandidateDecisions(group); err != nil {
			return err
		}
	}
	if precheck.ReviewerWorkPackage.Event.ID != "" {
		if err := validatePersistedEvent(precheck.ReviewerWorkPackage.Event); err != nil {
			return err
		}
	}
	for _, evidence := range precheck.ReviewerWorkPackage.Evidence {
		if err := validatePersistedEvidence(evidence); err != nil {
			return err
		}
	}
	for _, entity := range precheck.ReviewerWorkPackage.ResolvedEntities {
		if err := validatePersistedEntity(entity); err != nil {
			return err
		}
	}
	for _, candidate := range precheck.ReviewerWorkPackage.EntityLinks {
		if err := validatePersistedEntityLinkCandidate(candidate); err != nil {
			return err
		}
	}
	for _, candidate := range precheck.ReviewerWorkPackage.VariableSignals {
		if err := validatePersistedVariableSignalCandidate(candidate); err != nil {
			return err
		}
	}
	for _, candidate := range precheck.ReviewerWorkPackage.DirectImpacts {
		if err := validatePersistedDirectImpactCandidate(candidate); err != nil {
			return err
		}
	}
	return nil
}

func validatePersistedSubmissionSummary(result eventbiz.SubmissionResult) error {
	if !validPersistedUUID(result.SubmissionID) || !validPersistedUUID(result.EventID) ||
		!validPersistedSHA256(result.CanonicalPayloadHash) || !validPersistedReviewStatus(result.Status) {
		return invalidPersistedEventSemantic("Submission")
	}
	if err := validatePersistedPrecheck(result.Precheck); err != nil {
		return err
	}
	if result.Status != eventbiz.StatusSuperseded && eventbiz.SummarizeSubmission(result.Precheck) != result.Status {
		return invalidPersistedEventSemantic("Submission status")
	}
	return nil
}

func validatePersistedSubmissionIdentity(result eventbiz.SubmissionResult) error {
	if err := validatePersistedSubmissionSummary(result); err != nil {
		return err
	}
	if !validPersistedUUID(result.ContextLeaseID) || strings.TrimSpace(result.AgentExecutionID) == "" ||
		strings.TrimSpace(result.AgentKey) == "" || strings.TrimSpace(result.AgentVersion) == "" ||
		!validPersistedSHA256(result.GeneratorPromptHash) || strings.TrimSpace(result.GeneratorModel) == "" ||
		!validPersistedSHA256(result.ReviewerPromptHash) || strings.TrimSpace(result.ReviewerModel) == "" ||
		(result.AdjudicatorPromptHash != "" && !validPersistedSHA256(result.AdjudicatorPromptHash)) ||
		(result.AdjudicatorPromptHash == "") != (result.AdjudicatorModel == "") ||
		strings.TrimSpace(result.OntologyVersion) == "" || strings.TrimSpace(result.AcceptancePolicyVersion) == "" ||
		result.CreatedAt.IsZero() || (result.SupersedesSubmissionID != "" && !validPersistedUUID(result.SupersedesSubmissionID)) {
		return invalidPersistedEventSemantic("Submission identity")
	}
	requiresFinalization := result.Status == eventbiz.StatusAccepted ||
		result.Status == eventbiz.StatusQuarantined || result.Status == eventbiz.StatusSuperseded
	forbidsFinalization := result.Status == eventbiz.StatusPendingReview ||
		result.Status == eventbiz.StatusNeedsReanalysis
	if (requiresFinalization && result.FinalizedAt == nil) ||
		(forbidsFinalization && result.FinalizedAt != nil) {
		return invalidPersistedEventSemantic("Submission finalization")
	}
	return nil
}

func validatePersistedCandidateSnapshot(
	payload []byte,
	hash string,
	result eventbiz.SubmissionResult,
) (eventbiz.Submission, error) {
	return validatePersistedCandidateSnapshotPayload(payload, hash, result, true)
}

func validatePersistedCandidateSnapshotPayload(
	payload []byte,
	hash string,
	result eventbiz.SubmissionResult,
	validateFullIdentity bool,
) (eventbiz.Submission, error) {
	if hash != result.CanonicalPayloadHash || !validPersistedSHA256(hash) {
		return eventbiz.Submission{}, invalidPersistedEventSemantic("Candidate Snapshot hash")
	}
	var submission eventbiz.Submission
	if err := json.Unmarshal(payload, &submission); err != nil {
		return eventbiz.Submission{}, err
	}
	canonicalHash, err := eventSemanticFingerprint(submission)
	if err != nil {
		return eventbiz.Submission{}, err
	}
	if canonicalHash != hash {
		return eventbiz.Submission{}, invalidPersistedEventSemantic("Candidate Snapshot hash")
	}
	if submission.EventID != result.EventID {
		return eventbiz.Submission{}, invalidPersistedEventSemantic("Candidate Snapshot identity")
	}
	if validateFullIdentity &&
		(submission.ContextLeaseID != result.ContextLeaseID ||
			submission.AgentExecutionID != result.AgentExecutionID || submission.AgentKey != result.AgentKey ||
			submission.AgentVersion != result.AgentVersion || submission.SupersedesSubmissionID != result.SupersedesSubmissionID ||
			submission.GeneratorPromptHash != result.GeneratorPromptHash || submission.GeneratorModel != result.GeneratorModel ||
			submission.ReviewerPromptHash != result.ReviewerPromptHash || submission.ReviewerModel != result.ReviewerModel ||
			submission.AdjudicatorPromptHash != result.AdjudicatorPromptHash || submission.AdjudicatorModel != result.AdjudicatorModel ||
			submission.OntologyVersion != result.OntologyVersion || submission.AcceptancePolicyVersion != result.AcceptancePolicyVersion) {
		return eventbiz.Submission{}, invalidPersistedEventSemantic("Candidate Snapshot identity")
	}
	entityKeys := make(map[string]struct{}, len(submission.EntityLinks))
	for _, candidate := range submission.EntityLinks {
		if err := validatePersistedEntityLinkCandidate(candidate); err != nil {
			return eventbiz.Submission{}, err
		}
		if _, duplicate := entityKeys[candidate.Key]; duplicate {
			return eventbiz.Submission{}, invalidPersistedEventSemantic("Entity Link candidate set")
		}
		entityKeys[candidate.Key] = struct{}{}
	}
	signalKeys := make(map[string]struct{}, len(submission.VariableSignals))
	for _, candidate := range submission.VariableSignals {
		if err := validatePersistedVariableSignalCandidate(candidate); err != nil {
			return eventbiz.Submission{}, err
		}
		if _, duplicate := signalKeys[candidate.Key]; duplicate {
			return eventbiz.Submission{}, invalidPersistedEventSemantic("Variable Signal candidate set")
		}
		if _, ok := entityKeys[candidate.SubjectLinkKey]; !ok {
			return eventbiz.Submission{}, invalidPersistedEventSemantic("Variable Signal subject reference")
		}
		signalKeys[candidate.Key] = struct{}{}
	}
	decisionEntityKeys, err := validatePersistedCandidateDecisions(result.Precheck.EntityLinks)
	if err != nil {
		return eventbiz.Submission{}, err
	}
	decisionSignalKeys, err := validatePersistedCandidateDecisions(result.Precheck.VariableSignals)
	if err != nil {
		return eventbiz.Submission{}, err
	}
	if !samePersistedKeys(entityKeys, decisionEntityKeys) || !samePersistedKeys(signalKeys, decisionSignalKeys) {
		return eventbiz.Submission{}, invalidPersistedEventSemantic("Candidate Snapshot decision set")
	}
	return submission, nil
}

func samePersistedKeys(left, right map[string]struct{}) bool {
	if len(left) != len(right) {
		return false
	}
	for key := range left {
		if _, ok := right[key]; !ok {
			return false
		}
	}
	return true
}

func validatePersistedSubmissionAggregate(
	ctx context.Context,
	query semanticQueryer,
	result *eventbiz.SubmissionResult,
) error {
	var payload []byte
	var hash string
	if err := query.QueryRowContext(ctx, `
		SELECT payload, canonical_payload_hash
		FROM event_semantic_candidate_snapshots
		WHERE semantic_submission_id = $1
	`, result.SubmissionID).Scan(&payload, &hash); err != nil {
		return err
	}
	if err := validatePersistedSubmissionIdentity(*result); err != nil {
		return err
	}
	submission, err := validatePersistedCandidateSnapshotPayload(payload, hash, *result, true)
	if err != nil {
		return err
	}
	if err := validatePersistedReviewerWorkPackageCandidates(submission, result.Precheck); err != nil {
		return err
	}
	return validatePersistedCandidateRecords(ctx, query, result.SubmissionID, result.Status, &result.Precheck, false)
}

func validatePersistedReviewerWorkPackageCandidates(
	submission eventbiz.Submission,
	precheck eventbiz.PrecheckResult,
) error {
	if err := validatePersistedCandidateSubset(
		submission.EntityLinks,
		precheck.ReviewerWorkPackage.EntityLinks,
		func(item eventbiz.EntityLinkCandidate) string { return item.Key },
	); err != nil {
		return invalidPersistedEventSemantic("Reviewer Work Package Entity Link candidate set")
	}
	if err := validatePersistedCandidateSubset(
		submission.VariableSignals,
		precheck.ReviewerWorkPackage.VariableSignals,
		func(item eventbiz.VariableSignalCandidate) string { return item.Key },
	); err != nil {
		return invalidPersistedEventSemantic("Reviewer Work Package Variable Signal candidate set")
	}
	return nil
}

func validatePersistedCandidateSubset[T any](
	snapshot []T,
	workPackage []T,
	key func(T) string,
) error {
	byKey := make(map[string]T, len(snapshot))
	for _, candidate := range snapshot {
		byKey[key(candidate)] = candidate
	}
	seen := make(map[string]struct{}, len(workPackage))
	for _, candidate := range workPackage {
		candidateKey := key(candidate)
		if _, duplicate := seen[candidateKey]; duplicate {
			return errors.New("duplicate work-package candidate")
		}
		seen[candidateKey] = struct{}{}
		snapshotCandidate, ok := byKey[candidateKey]
		if !ok {
			return errors.New("candidate missing from snapshot")
		}
		snapshotHash, err := eventSemanticFingerprint(snapshotCandidate)
		if err != nil {
			return err
		}
		workPackageHash, err := eventSemanticFingerprint(candidate)
		if err != nil {
			return err
		}
		if snapshotHash != workPackageHash {
			return errors.New("candidate content differs from snapshot")
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
		       snapshot.payload, snapshot.canonical_payload_hash, run.created_at, run.finalized_at
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
		var candidateSnapshotHash string
		if err := rows.Scan(
			&run.SubmissionID, &run.EventID, &run.Status, &run.CanonicalPayloadHash, &decisions,
			&run.ContextLeaseID, &run.AgentExecutionID, &run.AgentKey, &run.AgentVersion,
			&run.SupersedesSubmissionID, &run.GeneratorPromptHash, &run.GeneratorModel,
			&run.ReviewerPromptHash, &run.ReviewerModel,
			&run.AdjudicatorPromptHash, &run.AdjudicatorModel,
			&run.OntologyVersion, &run.AcceptancePolicyVersion,
			&candidateSnapshot, &candidateSnapshotHash, &run.CreatedAt, &run.FinalizedAt,
		); err != nil {
			return eventbiz.EventSemanticsResult{}, err
		}
		if err := json.Unmarshal(decisions, &run.Precheck); err != nil {
			return eventbiz.EventSemanticsResult{}, err
		}
		if run.EventID != eventID {
			return eventbiz.EventSemanticsResult{}, invalidPersistedEventSemantic("Submission Event reference")
		}
		if err := validatePersistedSubmissionIdentity(run); err != nil {
			return eventbiz.EventSemanticsResult{}, err
		}
		candidateSubmission, err := validatePersistedCandidateSnapshot(candidateSnapshot, candidateSnapshotHash, run)
		if err != nil {
			return eventbiz.EventSemanticsResult{}, err
		}
		if err := validatePersistedReviewerWorkPackageCandidates(candidateSubmission, run.Precheck); err != nil {
			return eventbiz.EventSemanticsResult{}, err
		}
		run.CandidateSnapshot = append(json.RawMessage(nil), candidateSnapshot...)
		if run.ReviewSnapshots, err = r.eventSemanticReviewSnapshots(ctx, run.SubmissionID, run.Precheck); err != nil {
			return eventbiz.EventSemanticsResult{}, err
		}
		if err := r.populateSemanticRecordIDs(ctx, run.SubmissionID, run.Status, &run.Precheck); err != nil {
			return eventbiz.EventSemanticsResult{}, err
		}
		result.Submissions = append(result.Submissions, run)
	}
	return result, rows.Err()
}

func (r Store) eventSemanticReviewSnapshots(
	ctx context.Context,
	runID string,
	precheck eventbiz.PrecheckResult,
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
		if err := validatePersistedReviewSnapshot(item, runID, precheck); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func validatePersistedReviewSnapshot(
	item eventbiz.ReviewSnapshot,
	runID string,
	precheck eventbiz.PrecheckResult,
) error {
	if strings.TrimSpace(item.ReviewerExecutionKey) == "" || !validPersistedSHA256(item.CanonicalPayloadHash) || item.CreatedAt.IsZero() {
		return invalidPersistedEventSemantic("Review Snapshot")
	}
	var review eventbiz.ReviewSubmission
	if err := json.Unmarshal(item.Payload, &review); err != nil {
		return err
	}
	canonicalHash, err := eventSemanticFingerprint(review)
	if err != nil {
		return err
	}
	if canonicalHash != item.CanonicalPayloadHash {
		return invalidPersistedEventSemantic("Review Snapshot hash")
	}
	if review.SubmissionID != runID || review.ReviewerExecutionKey != item.ReviewerExecutionKey ||
		!validPersistedSHA256(review.PromptHash) || strings.TrimSpace(review.Model) == "" || len(review.Items) == 0 {
		return invalidPersistedEventSemantic("Review Snapshot identity")
	}
	candidateSets := map[eventbiz.CandidateType]map[string]struct{}{}
	if candidateSets[eventbiz.CandidateTypeEntityLink], err = validatePersistedCandidateDecisions(precheck.EntityLinks); err != nil {
		return err
	}
	if candidateSets[eventbiz.CandidateTypeVariableSignal], err = validatePersistedCandidateDecisions(precheck.VariableSignals); err != nil {
		return err
	}
	if candidateSets[eventbiz.CandidateTypeDirectImpact], err = validatePersistedCandidateDecisions(precheck.DirectImpacts); err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(review.Items))
	for _, reviewItem := range review.Items {
		candidates, ok := candidateSets[reviewItem.CandidateType]
		if !ok || strings.TrimSpace(reviewItem.CandidateKey) == "" ||
			(reviewItem.Decision != eventbiz.ReviewDecisionPass && reviewItem.Decision != eventbiz.ReviewDecisionFail &&
				reviewItem.Decision != eventbiz.ReviewDecisionIndeterminate) ||
			!validPersistedUUIDSet(reviewItem.EvidenceIDs, true) {
			return invalidPersistedEventSemantic("Review Snapshot item")
		}
		if _, ok := candidates[reviewItem.CandidateKey]; !ok {
			return invalidPersistedEventSemantic("Review Snapshot candidate reference")
		}
		identity := string(reviewItem.CandidateType) + ":" + reviewItem.CandidateKey
		if _, duplicate := seen[identity]; duplicate {
			return invalidPersistedEventSemantic("Review Snapshot candidate set")
		}
		seen[identity] = struct{}{}
	}
	return nil
}

func (r Store) populateSemanticRecordIDs(
	ctx context.Context,
	runID string,
	status eventbiz.ReviewStatus,
	precheck *eventbiz.PrecheckResult,
) error {
	return validatePersistedCandidateRecords(ctx, r.db, runID, status, precheck, true)
}

func validatePersistedCandidateRecords(
	ctx context.Context,
	queryer semanticQueryer,
	runID string,
	submissionStatus eventbiz.ReviewStatus,
	precheck *eventbiz.PrecheckResult,
	populate bool,
) error {
	groups := []struct {
		table            string
		items            *[]eventbiz.CandidateDecision
		expectedKeys     map[string]struct{}
		expectedEvidence map[string][]string
	}{
		{table: "event_entity_links", items: &precheck.EntityLinks, expectedKeys: entityLinkCandidateKeys(precheck.ReviewerWorkPackage.EntityLinks), expectedEvidence: entityLinkCandidateEvidence(precheck.ReviewerWorkPackage.EntityLinks)},
		{table: "variable_signals", items: &precheck.VariableSignals, expectedKeys: variableSignalCandidateKeys(precheck.ReviewerWorkPackage.VariableSignals), expectedEvidence: variableSignalCandidateEvidence(precheck.ReviewerWorkPackage.VariableSignals)},
		{table: "direct_impact_assertions", items: &precheck.DirectImpacts, expectedKeys: directImpactCandidateKeys(precheck.ReviewerWorkPackage.DirectImpacts), expectedEvidence: directImpactCandidateEvidence(precheck.ReviewerWorkPackage.DirectImpacts)},
	}
	for _, group := range groups {
		decisionKeys, err := validatePersistedCandidateDecisions(*group.items)
		if err != nil {
			return err
		}
		decisionByKey := decisionsByKey(*group.items)
		query := fmt.Sprintf(`
			SELECT candidate_key, id::text, review_status, array_to_json(evidence_ids)
			FROM %s
			WHERE semantic_submission_id = $1
		`, group.table)
		rows, err := queryer.QueryContext(ctx, query, runID)
		if err != nil {
			return err
		}
		ids := make(map[string]string)
		for rows.Next() {
			var key, id string
			var status eventbiz.ReviewStatus
			var evidencePayload []byte
			if err := rows.Scan(&key, &id, &status, &evidencePayload); err != nil {
				rows.Close()
				return err
			}
			if strings.TrimSpace(key) == "" || !validPersistedUUID(id) {
				rows.Close()
				return invalidPersistedEventSemantic("candidate record reference")
			}
			if _, ok := decisionKeys[key]; !ok {
				rows.Close()
				return invalidPersistedEventSemantic("candidate record decision reference")
			}
			if _, ok := group.expectedKeys[key]; !ok {
				rows.Close()
				return invalidPersistedEventSemantic("candidate record set")
			}
			var evidenceIDs []string
			if err := json.Unmarshal(evidencePayload, &evidenceIDs); err != nil {
				rows.Close()
				return err
			}
			expectedStatus := decisionByKey[key].Status
			if submissionStatus == eventbiz.StatusSuperseded {
				expectedStatus = eventbiz.StatusSuperseded
			}
			if status != expectedStatus || !samePersistedStringSet(evidenceIDs, group.expectedEvidence[key]) {
				rows.Close()
				return invalidPersistedEventSemantic("candidate record content")
			}
			if _, duplicate := ids[key]; duplicate {
				rows.Close()
				return invalidPersistedEventSemantic("candidate record uniqueness")
			}
			ids[key] = id
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
		if len(ids) != len(group.expectedKeys) {
			return invalidPersistedEventSemantic("candidate record completeness")
		}
		if populate {
			for index := range *group.items {
				if id, ok := ids[(*group.items)[index].CandidateKey]; ok {
					(*group.items)[index].RecordID = id
				}
			}
		}
	}
	return nil
}

func samePersistedStringSet(left, right []string) bool {
	if !validPersistedUUIDSet(left, true) || !validPersistedUUIDSet(right, true) || len(left) != len(right) {
		return false
	}
	values := make(map[string]struct{}, len(left))
	for _, value := range left {
		values[value] = struct{}{}
	}
	for _, value := range right {
		if _, ok := values[value]; !ok {
			return false
		}
	}
	return true
}

func entityLinkCandidateKeys(items []eventbiz.EntityLinkCandidate) map[string]struct{} {
	result := make(map[string]struct{}, len(items))
	for _, item := range items {
		result[item.Key] = struct{}{}
	}
	return result
}

func entityLinkCandidateEvidence(items []eventbiz.EntityLinkCandidate) map[string][]string {
	result := make(map[string][]string, len(items))
	for _, item := range items {
		result[item.Key] = item.EvidenceIDs
	}
	return result
}

func variableSignalCandidateKeys(items []eventbiz.VariableSignalCandidate) map[string]struct{} {
	result := make(map[string]struct{}, len(items))
	for _, item := range items {
		result[item.Key] = struct{}{}
	}
	return result
}

func variableSignalCandidateEvidence(items []eventbiz.VariableSignalCandidate) map[string][]string {
	result := make(map[string][]string, len(items))
	for _, item := range items {
		result[item.Key] = item.EvidenceIDs
	}
	return result
}

func directImpactCandidateKeys(items []eventbiz.DirectImpactCandidate) map[string]struct{} {
	result := make(map[string]struct{}, len(items))
	for _, item := range items {
		result[item.Key] = struct{}{}
	}
	return result
}

func directImpactCandidateEvidence(items []eventbiz.DirectImpactCandidate) map[string][]string {
	result := make(map[string][]string, len(items))
	for _, item := range items {
		result[item.Key] = item.EvidenceIDs
	}
	return result
}

func existingEventSemanticSubmission(ctx context.Context, tx *sql.Tx, executionID string) (eventbiz.SubmissionResult, bool, error) {
	result, found, err := queryEventSemanticSubmission(ctx, tx.QueryRowContext(ctx, `
		SELECT id,event_id,status,canonical_payload_hash,decision_summary,
		       context_lease_id::text,agent_execution_id,agent_key,agent_version,
		       COALESCE(supersedes_submission_id::text,''),generator_prompt_hash,generator_model,
		       reviewer_prompt_hash,reviewer_model,COALESCE(adjudicator_prompt_hash,''),
		       COALESCE(adjudicator_model,''),ontology_version,
		       acceptance_policy_key || '@' || acceptance_policy_version::text,
		       created_at,finalized_at
		FROM event_semantic_submissions
		WHERE agent_execution_id = $1
		FOR UPDATE
	`, executionID))
	if err != nil || !found {
		return result, found, err
	}
	if err := validatePersistedSubmissionAggregate(ctx, tx, &result); err != nil {
		return eventbiz.SubmissionResult{}, false, err
	}
	return result, true, nil
}

func eventSemanticSubmissionByID(ctx context.Context, tx *sql.Tx, runID string) (eventbiz.SubmissionResult, bool, error) {
	result, found, err := queryEventSemanticSubmission(ctx, tx.QueryRowContext(ctx, `
		SELECT id,event_id,status,canonical_payload_hash,decision_summary,
		       context_lease_id::text,agent_execution_id,agent_key,agent_version,
		       COALESCE(supersedes_submission_id::text,''),generator_prompt_hash,generator_model,
		       reviewer_prompt_hash,reviewer_model,COALESCE(adjudicator_prompt_hash,''),
		       COALESCE(adjudicator_model,''),ontology_version,
		       acceptance_policy_key || '@' || acceptance_policy_version::text,
		       created_at,finalized_at
		FROM event_semantic_submissions
		WHERE id = $1
		FOR UPDATE
	`, runID))
	if err != nil || !found {
		return result, found, err
	}
	if err := validatePersistedSubmissionAggregate(ctx, tx, &result); err != nil {
		return eventbiz.SubmissionResult{}, false, err
	}
	return result, true, nil
}

type semanticSubmissionRow interface {
	Scan(...any) error
}

func queryEventSemanticSubmission(_ context.Context, row semanticSubmissionRow) (eventbiz.SubmissionResult, bool, error) {
	var result eventbiz.SubmissionResult
	var decisions []byte
	err := row.Scan(
		&result.SubmissionID, &result.EventID, &result.Status, &result.CanonicalPayloadHash, &decisions,
		&result.ContextLeaseID, &result.AgentExecutionID, &result.AgentKey, &result.AgentVersion,
		&result.SupersedesSubmissionID, &result.GeneratorPromptHash, &result.GeneratorModel,
		&result.ReviewerPromptHash, &result.ReviewerModel,
		&result.AdjudicatorPromptHash, &result.AdjudicatorModel,
		&result.OntologyVersion, &result.AcceptancePolicyVersion,
		&result.CreatedAt, &result.FinalizedAt,
	)
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
		return eventbiz.SubmissionResult{}, false, nil
	}
	if err != nil {
		return eventbiz.SubmissionResult{}, false, err
	}
	if err := json.Unmarshal(decisions, &result.Precheck); err != nil {
		return eventbiz.SubmissionResult{}, false, err
	}
	if err := validatePersistedSubmissionIdentity(result); err != nil {
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
		if !validPersistedUUID(eventID) {
			return HistoricalEventSemanticManifest{}, invalidPersistedEventSemantic("historical Event reference")
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

const researchSemanticClosureCTE = `
WITH
requested_variables(variable_key, version) AS (SELECT * FROM unnest($2::text[], $3::integer[])),
requested_rules(rule_key, version) AS (SELECT * FROM unnest($4::text[], $5::integer[])),
requested_submissions(id) AS (SELECT unnest($6::uuid[])),
selected_rules AS MATERIALIZED (
    SELECT rule.* FROM direct_transmission_rules rule
    JOIN requested_rules requested ON requested.rule_key = rule.rule_key AND requested.version = rule.version
    WHERE rule.status = 'approved' AND rule.created_at <= $1 AND COALESCE(rule.reviewed_at, rule.created_at) <= $1
),
selected_variable_keys(variable_key, version) AS MATERIALIZED (
    SELECT variable_key, version FROM requested_variables
    UNION SELECT source_variable_key, source_variable_version FROM selected_rules
    UNION SELECT affected_variable_key, affected_variable_version FROM selected_rules
),
selected_variable_definitions AS MATERIALIZED (
    SELECT definition.* FROM variable_definitions definition
    JOIN selected_variable_keys selected ON selected.variable_key = definition.variable_key AND selected.version = definition.version
    WHERE definition.status = 'active' AND definition.created_at <= $1
),
selected_applicable_entity_types AS MATERIALIZED (
    SELECT applicable.* FROM variable_definition_entity_types applicable
    JOIN selected_variable_keys selected ON selected.variable_key = applicable.variable_key AND selected.version = applicable.variable_version
    WHERE applicable.created_at <= $1
),
selected_policy_keys(policy_key, version) AS MATERIALIZED (
    SELECT submission.acceptance_policy_key, submission.acceptance_policy_version
    FROM event_semantic_submissions submission JOIN requested_submissions requested ON requested.id = submission.id
    WHERE submission.status = 'accepted' AND COALESCE(submission.finalized_at, submission.created_at) <= $1
),
selected_policies AS MATERIALIZED (
    SELECT policy.* FROM event_semantic_acceptance_policies policy
    JOIN selected_policy_keys selected ON selected.policy_key = policy.policy_key AND selected.version = policy.version
    WHERE policy.status = 'active' AND policy.created_at <= $1
)
`

type researchSemanticClosureParameters struct {
	variableKeys          []string
	variableVersions      []int32
	ruleKeys              []string
	ruleVersions          []int32
	semanticSubmissionIDs []string
}

func (s Store) ListResearchSemantics(
	ctx context.Context,
	query eventbiz.ResearchSemanticQuery,
) ([]eventbiz.ResearchSemanticRecord, error) {
	if s.db == nil {
		return nil, errors.New("Event Semantic database is required")
	}
	var historicalGap bool
	if err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS (
		    SELECT 1
		    FROM event_semantic_submissions submission
		    JOIN events event ON event.id = submission.event_id
		    WHERE submission.status = 'superseded'
		      AND submission.created_at <= $3
		      AND submission.finalized_at > $3
		      AND COALESCE(event.knowable_at, event.first_seen_at) >= $1
		      AND COALESCE(event.knowable_at, event.first_seen_at) < $2
		      AND COALESCE(event.knowable_at, event.first_seen_at) <= $3
		)
	`, query.DiscoveryWindowStart, query.DiscoveryWindowEnd, query.AnalysisAsOf).Scan(&historicalGap); err != nil {
		return nil, err
	}
	if historicalGap {
		return nil, eventbiz.ErrResearchHistoricalSemanticsUnavailable
	}
	result := make([]eventbiz.ResearchSemanticRecord, 0, len(query.EventIDs))
	for _, eventID := range query.EventIDs {
		payload, err := s.researchSemanticRecord(ctx, eventID, query.AnalysisAsOf)
		if err != nil {
			return nil, err
		}
		result = append(result, payload)
	}
	return result, nil
}

func (s *Store) preflightReferenceClosureBudget(
	ctx context.Context,
	query eventbiz.ResearchSemanticClosureQuery,
	parameters researchSemanticClosureParameters,
) error {
	var rows, bytes int64
	err := s.db.QueryRowContext(
		ctx,
		researchSemanticClosureCTE+`
		SELECT count(*)::bigint, COALESCE(sum(pg_column_size(item)), 0)::bigint
		FROM (
		    SELECT to_jsonb(definition) item FROM selected_variable_definitions definition
		    UNION ALL
		    SELECT to_jsonb(applicable) FROM selected_applicable_entity_types applicable
		    UNION ALL
		    SELECT to_jsonb(rule) FROM selected_rules rule
		    UNION ALL
		    SELECT to_jsonb(policy) FROM selected_policies policy
		) records
	`,
		researchSemanticClosureArgs(query.AnalysisAsOf, parameters)...,
	).Scan(&rows, &bytes)
	if err != nil {
		return err
	}
	if rows > eventbiz.ResearchMaxDictionaryRows ||
		bytes > eventbiz.ResearchMaxDictionaryBytes {
		return &eventbiz.ResearchResourceLimitError{
			Reason:        "Research Analysis Context reference closure exceeds the preflight budget",
			Component:     "reference_closure",
			ActualRows:    int64Pointer(rows),
			MaxRows:       int64Pointer(eventbiz.ResearchMaxDictionaryRows),
			ActualBytes:   int64Pointer(bytes),
			MaxBytes:      int64Pointer(eventbiz.ResearchMaxDictionaryBytes),
			RetryGuidance: "reduce_page_size",
		}
	}
	return nil
}

func (s *Store) researchSemanticRecord(
	ctx context.Context,
	eventID string,
	analysisAsOf time.Time,
) (eventbiz.ResearchSemanticRecord, error) {
	var payload []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT jsonb_build_object(
		    'entity_links', COALESCE((
		        SELECT jsonb_agg(jsonb_build_object(
		            'event_entity_link_id', link.id,
		            'semantic_submission_id', link.semantic_submission_id,
			            'entity_id', COALESCE(link.country_id, link.organization_id, link.entity_id::text),
		            'entity_role', link.entity_role,
		            'resolved_mention', link.resolved_mention,
		            'resolution_method', link.resolution_method,
		            'resolution_confidence', link.resolution_confidence,
		            'evidence_ids', link.evidence_ids,
		            'review_status', link.review_status
			        ) ORDER BY link.entity_role, COALESCE(link.country_id, link.organization_id, link.entity_id::text), link.id)
		        FROM event_entity_links link
		        JOIN event_semantic_submissions submission
		          ON submission.id = link.semantic_submission_id
		        WHERE link.event_id = $1::uuid
		          AND link.review_status = 'accepted'
		          AND submission.status = 'accepted'
		          AND link.updated_at <= $2
		          AND COALESCE(submission.finalized_at, submission.created_at) <= $2
		    ), '[]'::jsonb),
		    'variable_signals', COALESCE((
		        SELECT jsonb_agg(jsonb_build_object(
		            'variable_signal_id', signal.id,
		            'semantic_submission_id', signal.semantic_submission_id,
		            'source_event_id', signal.source_event_id,
		            'subject_event_entity_link_id', signal.subject_event_entity_link_id,
			            'subject_entity_id', COALESCE(subject.country_id, subject.organization_id, subject.entity_id::text),
		            'variable_key', signal.variable_key,
		            'variable_version', signal.variable_version,
		            'direction', signal.direction,
		            'assertion_modality', signal.assertion_modality,
		            'evidence_ids', signal.evidence_ids,
		            'statement_at', signal.statement_at,
		            'valid_from', signal.valid_from,
		            'valid_until', signal.valid_until,
		            'forecast_period_start', signal.forecast_period_start,
		            'forecast_period_end', signal.forecast_period_end,
		            'extraction_confidence', signal.extraction_confidence,
		            'review_status', signal.review_status,
		            'measurements', COALESCE((
		                SELECT jsonb_agg(jsonb_build_object(
		                    'measurement_id', measurement.id,
		                    'measurement_role', measurement.measurement_role,
		                    'value_shape', measurement.value_shape,
		                    'raw_value', measurement.raw_value::text,
		                    'raw_lower', measurement.raw_lower::text,
		                    'raw_upper', measurement.raw_upper::text,
		                    'raw_unit', measurement.raw_unit,
		                    'canonical_value', measurement.canonical_value::text,
		                    'canonical_lower', measurement.canonical_lower::text,
		                    'canonical_upper', measurement.canonical_upper::text,
		                    'canonical_unit', measurement.canonical_unit,
		                    'currency', measurement.currency,
		                    'scale', measurement.scale,
		                    'comparison_basis', measurement.comparison_basis,
		                    'comparison_period', measurement.comparison_period,
		                    'raw_text', measurement.raw_text,
		                    'is_approximate', measurement.is_approximate,
		                    'evidence_id', measurement.evidence_id
		                ) ORDER BY measurement.id)
		                FROM variable_signal_measurements measurement
		                WHERE measurement.variable_signal_id = signal.id
		            ), '[]'::jsonb),
		            'direct_impacts', COALESCE((
		                SELECT jsonb_agg(jsonb_build_object(
		                    'direct_impact_assertion_id', impact.id,
		                    'semantic_submission_id', impact.semantic_submission_id,
		                    'source_variable_signal_id', impact.source_variable_signal_id,
		                    'target_entity_id', impact.target_entity_id,
		                    'affected_variable_key', impact.affected_variable_key,
		                    'affected_variable_version', impact.affected_variable_version,
		                    'affected_direction', impact.affected_direction,
		                    'derivation_type', impact.derivation_type,
		                    'mechanism_summary', impact.mechanism_summary,
		                    'evidence_ids', impact.evidence_ids,
		                    'entity_relation_id', impact.entity_relation_id,
		                    'rule_key', impact.rule_key,
		                    'rule_version', impact.rule_version,
		                    'assertion_confidence', impact.assertion_confidence,
		                    'effective_from', impact.effective_from,
		                    'effective_to', impact.effective_to,
		                    'review_status', impact.review_status
		                ) ORDER BY impact.target_entity_id, impact.affected_variable_key, impact.id)
		                FROM direct_impact_assertions impact
		                JOIN event_semantic_submissions impact_submission
		                  ON impact_submission.id = impact.semantic_submission_id
		                WHERE impact.source_variable_signal_id = signal.id
		                  AND impact.review_status = 'accepted'
		                  AND impact_submission.status = 'accepted'
		                  AND impact.updated_at <= $2
		                  AND COALESCE(
		                      impact_submission.finalized_at,
		                      impact_submission.created_at
		                  ) <= $2
		            ), '[]'::jsonb)
		        ) ORDER BY signal.created_at, signal.id)
		        FROM variable_signals signal
		        JOIN event_entity_links subject
		          ON subject.id = signal.subject_event_entity_link_id
		        JOIN event_semantic_submissions submission
		          ON submission.id = signal.semantic_submission_id
		        WHERE signal.source_event_id = $1::uuid
		          AND signal.review_status = 'accepted'
		          AND submission.status = 'accepted'
		          AND signal.updated_at <= $2
		          AND COALESCE(submission.finalized_at, submission.created_at) <= $2
		    ), '[]'::jsonb)
		)
	`, eventID, analysisAsOf).Scan(&payload)
	if err != nil {
		return eventbiz.ResearchSemanticRecord{}, err
	}
	var record eventbiz.ResearchSemanticRecord
	if err := strictDecodeResearchContext(payload, &record); err != nil {
		return eventbiz.ResearchSemanticRecord{}, err
	}
	record.EventID = eventID
	if err := validateResearchSemanticRecord(record); err != nil {
		return eventbiz.ResearchSemanticRecord{}, err
	}
	return record, nil
}

func validateResearchSemanticRecord(record eventbiz.ResearchSemanticRecord) error {
	if _, err := uuid.Parse(record.EventID); err != nil {
		return errors.New("persisted Research semantic Event reference is invalid")
	}
	links := make(map[string]struct{}, len(record.EntityLinks))
	for _, link := range record.EntityLinks {
		if !researchUUID(link.EventEntityLinkID) || !researchUUID(link.SemanticSubmissionID) ||
			!researchUUID(link.EntityID) || strings.TrimSpace(link.EntityRole) == "" ||
			link.ReviewStatus != "accepted" || !validResearchUUIDSet(link.EvidenceIDs) {
			return errors.New("persisted Research Entity Link violates invariants")
		}
		if _, duplicate := links[link.EventEntityLinkID]; duplicate {
			return errors.New("persisted Research Entity Link is duplicated")
		}
		links[link.EventEntityLinkID] = struct{}{}
	}
	signals := make(map[string]struct{}, len(record.VariableSignals))
	for _, signal := range record.VariableSignals {
		if !researchUUID(signal.VariableSignalID) || !researchUUID(signal.SemanticSubmissionID) ||
			signal.SourceEventID != record.EventID || !researchUUID(signal.SubjectEntityID) ||
			signal.VariableVersion < 1 || strings.TrimSpace(signal.VariableKey) == "" ||
			!researchOneOf(signal.Direction, "increase", "decrease", "unchanged", "mixed", "uncertain") ||
			!researchOneOf(signal.AssertionModality, "actual", "stated_intent", "source_forecast") ||
			signal.ReviewStatus != "accepted" || !validResearchUUIDSet(signal.EvidenceIDs) {
			return errors.New("persisted Research Variable Signal violates invariants")
		}
		if _, ok := links[signal.SubjectEventEntityLinkID]; !ok {
			return errors.New("persisted Research Variable Signal subject link is unavailable")
		}
		if _, duplicate := signals[signal.VariableSignalID]; duplicate {
			return errors.New("persisted Research Variable Signal is duplicated")
		}
		signals[signal.VariableSignalID] = struct{}{}
		for _, measurement := range signal.Measurements {
			if !researchUUID(measurement.MeasurementID) || measurement.EvidenceID == "" ||
				!researchUUID(measurement.EvidenceID) || strings.TrimSpace(measurement.RawText) == "" {
				return errors.New("persisted Research measurement violates invariants")
			}
			if measurement.MeasurementRole != "" && !researchOneOf(measurement.MeasurementRole, "absolute_level", "absolute_change", "relative_change", "percentage_point_change") {
				return errors.New("persisted Research measurement role is invalid")
			}
			if measurement.ValueShape != "" && !researchOneOf(measurement.ValueShape, "exact", "range", "lower_bound", "upper_bound") {
				return errors.New("persisted Research measurement shape is invalid")
			}
		}
		for _, impact := range signal.DirectImpacts {
			if !researchUUID(impact.DirectImpactAssertionID) || !researchUUID(impact.SemanticSubmissionID) ||
				impact.SourceVariableSignalID != signal.VariableSignalID || !researchUUID(impact.TargetEntityID) ||
				impact.AffectedVariableVersion < 1 || strings.TrimSpace(impact.AffectedVariableKey) == "" ||
				!researchOneOf(impact.AffectedDirection, "increase", "decrease", "unchanged", "mixed", "uncertain") ||
				!researchOneOf(impact.DerivationType, "event_explicit", "rule_inferred") ||
				strings.TrimSpace(impact.MechanismSummary) == "" || impact.ReviewStatus != "accepted" ||
				!validResearchUUIDSet(impact.EvidenceIDs) {
				return errors.New("persisted Research Direct Impact violates invariants")
			}
		}
	}
	return nil
}

func researchUUID(value string) bool {
	_, err := uuid.Parse(value)
	return err == nil
}

func validResearchUUIDSet(values []string) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !researchUUID(value) {
			return false
		}
		if _, duplicate := seen[value]; duplicate {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func researchOneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func (s Store) ResearchSemanticClosure(
	ctx context.Context,
	query eventbiz.ResearchSemanticClosureQuery,
) (eventbiz.ResearchSemanticDictionaries, error) {
	parameters := buildResearchSemanticClosureParameters(query)
	policiesResolve, err := s.referenceClosurePoliciesResolve(ctx, query.AnalysisAsOf, parameters.semanticSubmissionIDs)
	if err != nil {
		return eventbiz.ResearchSemanticDictionaries{}, err
	}
	if !policiesResolve {
		return eventbiz.ResearchSemanticDictionaries{}, eventbiz.ErrResearchReferenceClosureInconsistent
	}
	if err := s.preflightReferenceClosureBudget(ctx, query, parameters); err != nil {
		return eventbiz.ResearchSemanticDictionaries{}, err
	}
	var payload []byte
	err = s.db.QueryRowContext(ctx, researchSemanticClosureCTE+`
SELECT jsonb_build_object(
    'variable_definitions', COALESCE((SELECT jsonb_agg(jsonb_build_object(
        'key', definition.variable_key, 'version', definition.version,
        'name_zh', definition.name_zh, 'name_en', definition.name_en,
        'domain', definition.domain, 'business_definition', definition.business_definition,
        'value_type', definition.value_type, 'allowed_directions', definition.allowed_directions,
        'canonical_unit', definition.canonical_unit, 'status', definition.status,
        'applicable_entity_types', COALESCE((SELECT jsonb_agg(applicable.entity_type ORDER BY applicable.entity_type)
            FROM variable_definition_entity_types applicable
            WHERE applicable.variable_key = definition.variable_key
              AND applicable.variable_version = definition.version
              AND applicable.created_at <= $1), '[]'::jsonb)
    ) ORDER BY definition.variable_key, definition.version) FROM selected_variable_definitions definition), '[]'::jsonb),
    'direct_transmission_rules', COALESCE((SELECT jsonb_agg(jsonb_build_object(
        'rule_key', rule.rule_key, 'version', rule.version,
        'source_entity_type', rule.source_entity_type, 'source_variable_key', rule.source_variable_key,
        'source_variable_version', rule.source_variable_version, 'source_direction', rule.source_direction,
        'relation_type', rule.relation_type, 'target_entity_type', rule.target_entity_type,
        'affected_variable_key', rule.affected_variable_key, 'affected_variable_version', rule.affected_variable_version,
        'affected_direction', rule.affected_direction, 'condition_summary', rule.condition_summary,
        'mechanism_template', rule.mechanism_template, 'status', rule.status
    ) ORDER BY rule.rule_key, rule.version) FROM selected_rules rule), '[]'::jsonb),
    'acceptance_policies', COALESCE((SELECT jsonb_agg(jsonb_build_object(
        'policy_key', policy.policy_key, 'version', policy.version,
        'retry_budget', policy.retry_budget, 'status', policy.status, 'policy', policy.policy
    ) ORDER BY policy.policy_key, policy.version) FROM selected_policies policy), '[]'::jsonb)
)`, researchSemanticClosureArgs(query.AnalysisAsOf, parameters)...).Scan(&payload)
	if err != nil {
		return eventbiz.ResearchSemanticDictionaries{}, err
	}
	var dictionaries eventbiz.ResearchSemanticDictionaries
	if err := strictDecodeResearchContext(payload, &dictionaries); err != nil {
		return eventbiz.ResearchSemanticDictionaries{}, err
	}
	if err := validateResearchSemanticDictionaries(dictionaries); err != nil {
		return eventbiz.ResearchSemanticDictionaries{}, err
	}
	return dictionaries, nil
}

func validateResearchSemanticDictionaries(value eventbiz.ResearchSemanticDictionaries) error {
	variables := make(map[string]struct{}, len(value.VariableDefinitions))
	for _, definition := range value.VariableDefinitions {
		if strings.TrimSpace(definition.Key) == "" || definition.Version < 1 ||
			strings.TrimSpace(definition.NameZH) == "" || strings.TrimSpace(definition.BusinessDefinition) == "" ||
			definition.Status != "active" || len(definition.AllowedDirections) == 0 {
			return errors.New("persisted Research Variable Definition violates invariants")
		}
		variables[fmt.Sprintf("%s\x00%d", definition.Key, definition.Version)] = struct{}{}
	}
	for _, rule := range value.DirectTransmissionRules {
		if strings.TrimSpace(rule.RuleKey) == "" || rule.Version < 1 || rule.Status != "approved" ||
			strings.TrimSpace(rule.RelationType) == "" || strings.TrimSpace(rule.SourceEntityType) == "" ||
			strings.TrimSpace(rule.TargetEntityType) == "" || strings.TrimSpace(rule.ConditionSummary) == "" ||
			strings.TrimSpace(rule.MechanismTemplate) == "" {
			return errors.New("persisted Research Direct Transmission Rule violates invariants")
		}
		if _, ok := variables[fmt.Sprintf("%s\x00%d", rule.SourceVariableKey, rule.SourceVariableVersion)]; !ok {
			return errors.New("persisted Research rule source variable is unavailable")
		}
		if _, ok := variables[fmt.Sprintf("%s\x00%d", rule.AffectedVariableKey, rule.AffectedVariableVersion)]; !ok {
			return errors.New("persisted Research rule target variable is unavailable")
		}
	}
	for _, policy := range value.AcceptancePolicies {
		if strings.TrimSpace(policy.PolicyKey) == "" || policy.Version < 1 || policy.RetryBudget < 0 ||
			policy.Status != "active" || len(policy.Policy) == 0 || !json.Valid(policy.Policy) {
			return errors.New("persisted Research Acceptance Policy violates invariants")
		}
	}
	return nil
}

func (s *Store) referenceClosurePoliciesResolve(
	ctx context.Context,
	analysisAsOf time.Time,
	submissionIDs []string,
) (bool, error) {
	var resolves bool
	err := s.db.QueryRowContext(ctx, `
		WITH requested_submissions(id) AS (
		    SELECT unnest($2::uuid[])
		)
		SELECT NOT EXISTS (
		    SELECT 1
		    FROM requested_submissions requested
		    LEFT JOIN event_semantic_submissions submission
		      ON submission.id = requested.id
		     AND submission.status = 'accepted'
		     AND COALESCE(submission.finalized_at, submission.created_at) <= $1
		    LEFT JOIN event_semantic_acceptance_policies policy
		      ON policy.policy_key = submission.acceptance_policy_key
		     AND policy.version = submission.acceptance_policy_version
		     AND policy.status = 'active'
		     AND policy.created_at <= $1
		    WHERE submission.id IS NULL
		       OR policy.policy_key IS NULL
		)
	`, analysisAsOf, submissionIDs).Scan(&resolves)
	return resolves, err
}

func buildResearchSemanticClosureParameters(
	query eventbiz.ResearchSemanticClosureQuery,
) researchSemanticClosureParameters {
	parameters := researchSemanticClosureParameters{
		semanticSubmissionIDs: append([]string(nil), query.SemanticSubmissionIDs...),
		variableKeys:          make([]string, 0, len(query.VariableDefinitions)),
		variableVersions:      make([]int32, 0, len(query.VariableDefinitions)),
		ruleKeys:              make([]string, 0, len(query.DirectTransmissionRules)),
		ruleVersions:          make([]int32, 0, len(query.DirectTransmissionRules)),
	}
	for _, reference := range query.VariableDefinitions {
		parameters.variableKeys = append(parameters.variableKeys, reference.Key)
		parameters.variableVersions = append(parameters.variableVersions, int32(reference.Version))
	}
	for _, reference := range query.DirectTransmissionRules {
		parameters.ruleKeys = append(parameters.ruleKeys, reference.Key)
		parameters.ruleVersions = append(parameters.ruleVersions, int32(reference.Version))
	}
	return parameters
}

func researchSemanticClosureArgs(
	analysisAsOf time.Time,
	parameters researchSemanticClosureParameters,
) []any {
	return []any{
		analysisAsOf,
		parameters.variableKeys,
		parameters.variableVersions,
		parameters.ruleKeys,
		parameters.ruleVersions,
		parameters.semanticSubmissionIDs,
	}
}

func strictDecodeResearchContext(payload []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode typed Research Analysis Context: %w", err)
	}
	return nil
}

func nullUUID(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func int64Pointer(value int64) *int64 {
	return &value
}
