package eventsemantics

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var confidencePattern = regexp.MustCompile(`^(0(\.\d{1,5})?|1(\.0{1,5})?)$`)

func Precheck(context Context, submission Submission) PrecheckResult {
	result := PrecheckResult{
		ReviewerWorkPackage: ReviewerWorkPackage{
			Event: context.Event, Evidence: append([]Evidence(nil), context.Evidence...),
		},
	}
	evidence := indexEvidence(context.Evidence)
	entities := indexEntities(context.Entities)
	variables := indexVariables(context.Variables)

	linkByKey := make(map[string]EntityLinkCandidate, len(submission.EntityLinks))
	linkStatus := make(map[string]ReviewStatus, len(submission.EntityLinks))
	seenEntityIDs := make(map[string]struct{}, len(submission.EntityLinks))
	reviewerEntityIDs := make(map[string]struct{}, len(submission.EntityLinks))
	for _, candidate := range submission.EntityLinks {
		reason := validateLink(context, candidate, evidence, entities)
		if reason == "" {
			if _, duplicate := seenEntityIDs[candidate.EntityID]; duplicate {
				reason = "duplicate_entity_link"
			} else {
				seenEntityIDs[candidate.EntityID] = struct{}{}
			}
		}
		decision := decision(candidate.Key, reason)
		result.EntityLinks = append(result.EntityLinks, decision)
		linkByKey[candidate.Key], linkStatus[candidate.Key] = candidate, decision.Status
		if decision.Status == StatusPendingReview {
			result.ReviewerWorkPackage.EntityLinks = append(result.ReviewerWorkPackage.EntityLinks, candidate)
			if _, exists := reviewerEntityIDs[candidate.EntityID]; !exists {
				result.ReviewerWorkPackage.ResolvedEntities = append(
					result.ReviewerWorkPackage.ResolvedEntities, entities[candidate.EntityID],
				)
				reviewerEntityIDs[candidate.EntityID] = struct{}{}
			}
		}
	}

	for _, candidate := range submission.VariableSignals {
		reason := validateSignal(context, candidate, evidence, entities, variables, linkByKey, linkStatus)
		decision := decision(candidate.Key, reason)
		result.VariableSignals = append(result.VariableSignals, decision)
		if decision.Status == StatusPendingReview {
			result.ReviewerWorkPackage.VariableSignals = append(result.ReviewerWorkPackage.VariableSignals, candidate)
		}
	}
	return result
}

func decision(key, reason string) CandidateDecision {
	if reason != "" {
		return CandidateDecision{CandidateKey: key, Status: StatusRejected, ReasonCode: reason}
	}
	return CandidateDecision{CandidateKey: key, Status: StatusPendingReview}
}

func validateLink(context Context, candidate EntityLinkCandidate, evidence map[string]Evidence, entities map[string]Entity) string {
	if context.Event.Status != "confirmed" || context.Event.FactStatus != "verified" {
		return "event_not_eligible"
	}
	if strings.TrimSpace(candidate.Key) == "" || strings.TrimSpace(candidate.Mention) == "" ||
		strings.TrimSpace(candidate.EntityRole) == "" || strings.TrimSpace(candidate.ResolutionMethod) == "" {
		return "link_invalid"
	}
	entity, exists := entities[candidate.EntityID]
	if !exists || entity.Status != "active" {
		return "entity_not_found"
	}
	if strings.TrimSpace(candidate.ProjectedEntityType) == "" || candidate.ProjectedEntityType != entity.Type {
		return "entity_projection_type_mismatch"
	}
	entityType, exists := activeEntityType(context.EntityTypes, entity.Type)
	if !exists || !entityType.EventLinkAllowed || !contains(entityType.AllowedEventRoles, candidate.EntityRole) {
		return "entity_role_invalid"
	}
	if !allEvidenceExists(candidate.EvidenceIDs, evidence) {
		return "evidence_not_in_event"
	}
	if !contains([]string{"qdrant_exact", "qdrant_vector"}, candidate.ResolutionMethod) {
		return "entity_resolution_method_invalid"
	}
	if !mentionGrounded(context, candidate) {
		return "entity_mention_not_in_evidence"
	}
	if !validConfidence(candidate.ResolutionConfidence) {
		return "confidence_invalid"
	}
	return ""
}

func validateSignal(
	context Context,
	candidate VariableSignalCandidate,
	evidence map[string]Evidence,
	entities map[string]Entity,
	variables map[string]VariableDefinition,
	links map[string]EntityLinkCandidate,
	linkStatus map[string]ReviewStatus,
) string {
	link, exists := links[candidate.SubjectLinkKey]
	if !exists {
		return "subject_link_not_found"
	}
	if linkStatus[candidate.SubjectLinkKey] == StatusRejected {
		return "upstream_rejected"
	}
	entity := entities[link.EntityID]
	entityType, exists := activeEntityType(context.EntityTypes, entity.Type)
	if !exists || !entityType.SignalSubjectAllowed {
		return "signal_subject_not_allowed"
	}
	variable, exists := variables[definitionIdentity(candidate.VariableKey, candidate.VariableVersion)]
	if !exists || variable.Status != "active" {
		return "variable_not_found"
	}
	if !contains(variable.ApplicableEntityTypes, entity.Type) {
		return "variable_not_applicable"
	}
	if !contains(variable.AllowedDirections, candidate.Direction) {
		return "direction_not_allowed"
	}
	if !contains(context.AssertionModalities, candidate.AssertionModality) {
		return "assertion_modality_invalid"
	}
	if !allEvidenceExists(candidate.EvidenceIDs, evidence) {
		return "evidence_not_in_event"
	}
	if !validConfidence(candidate.ExtractionConfidence) {
		return "confidence_invalid"
	}
	if invalidTimeRange(candidate.ValidFrom, candidate.ValidUntil) ||
		invalidTimeRange(candidate.ForecastPeriodStart, candidate.ForecastPeriodEnd) {
		return "signal_time_invalid"
	}
	if len(candidate.Measurements) > context.MeasurementContract.MaxItemsPerSignal {
		return "measurement_count_invalid"
	}
	for _, measurement := range candidate.Measurements {
		if text := strings.TrimSpace(measurement.Text); text == "" ||
			len([]rune(text)) > context.MeasurementContract.MaxTextCharacters {
			return "measurement_text_invalid"
		}
		if !allEvidenceExists(measurement.EvidenceIDs, evidence) {
			return "evidence_not_in_event"
		}
	}
	return ""
}

func activeEntityType(items []EntityTypeDefinition, typeKey string) (EntityTypeDefinition, bool) {
	for _, item := range items {
		if item.TypeKey == typeKey && item.Status == "active" {
			return item, true
		}
	}
	return EntityTypeDefinition{}, false
}

func mentionGrounded(context Context, candidate EntityLinkCandidate) bool {
	mention := strings.ToLower(strings.TrimSpace(candidate.Mention))
	if mention == "" || len(candidate.EvidenceIDs) == 0 {
		return false
	}
	eventContainsMention := strings.Contains(strings.ToLower(context.Event.Title), mention) ||
		strings.Contains(strings.ToLower(context.Event.Summary), mention)
	evidenceByID := indexEvidence(context.Evidence)
	mentionedInEvidence := false
	hasPrimarySupportingLineage := false
	for _, evidenceID := range candidate.EvidenceIDs {
		item, ok := evidenceByID[evidenceID]
		if !ok {
			return false
		}
		if strings.Contains(strings.ToLower(item.Excerpt), mention) ||
			strings.Contains(strings.ToLower(item.Title), mention) {
			mentionedInEvidence = true
		}
		if item.IsPrimary && item.Relation == "supports" {
			hasPrimarySupportingLineage = true
		}
	}
	if mentionedInEvidence {
		return true
	}
	return eventContainsMention && hasPrimarySupportingLineage
}

func indexEvidence(items []Evidence) map[string]Evidence {
	result := make(map[string]Evidence, len(items))
	for _, item := range items {
		result[item.ID] = item
	}
	return result
}

func indexEntities(items []Entity) map[string]Entity {
	result := make(map[string]Entity, len(items))
	for _, item := range items {
		result[item.ID] = item
	}
	return result
}

func indexVariables(items []VariableDefinition) map[string]VariableDefinition {
	result := make(map[string]VariableDefinition, len(items))
	for _, item := range items {
		result[definitionIdentity(item.Key, item.Version)] = item
	}
	return result
}

func allEvidenceExists(ids []string, evidence map[string]Evidence) bool {
	if len(ids) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if _, exists := evidence[id]; !exists {
			return false
		}
		if _, duplicate := seen[id]; duplicate {
			return false
		}
		seen[id] = struct{}{}
	}
	return true
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func definitionIdentity(key string, version int) string {
	return key + "@" + strconv.Itoa(version)
}

func validConfidence(value string) bool {
	if value == "" {
		return true
	}
	return confidencePattern.MatchString(value)
}

func invalidTimeRange(start, end *time.Time) bool {
	return start != nil && end != nil && end.Before(*start)
}
