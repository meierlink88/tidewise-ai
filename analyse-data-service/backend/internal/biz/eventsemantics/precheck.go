package eventsemantics

import (
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var decimalPattern = regexp.MustCompile(`^[+-]?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?$`)

func Precheck(context Context, submission Submission) PrecheckResult {
	result := PrecheckResult{
		ReviewerWorkPackage: ReviewerWorkPackage{
			Event: context.Event, Evidence: append([]Evidence(nil), context.Evidence...),
		},
	}
	evidence := indexEvidence(context.Evidence)
	entities := indexEntities(context.Entities)
	variables := indexVariables(context.Variables)
	relations := indexRelations(context.Relations)
	rules := indexRules(context.Rules)

	linkByKey := make(map[string]EntityLinkCandidate, len(submission.EntityLinks))
	linkStatus := make(map[string]ReviewStatus, len(submission.EntityLinks))
	for _, candidate := range submission.EntityLinks {
		reason := validateLink(context, candidate, evidence, entities)
		decision := decision(candidate.Key, reason)
		result.EntityLinks = append(result.EntityLinks, decision)
		linkByKey[candidate.Key], linkStatus[candidate.Key] = candidate, decision.Status
		if decision.Status == StatusPendingReview {
			result.ReviewerWorkPackage.EntityLinks = append(result.ReviewerWorkPackage.EntityLinks, candidate)
		}
	}

	signalByKey := make(map[string]VariableSignalCandidate, len(submission.VariableSignals))
	signalStatus := make(map[string]ReviewStatus, len(submission.VariableSignals))
	for _, candidate := range submission.VariableSignals {
		reason := validateSignal(candidate, evidence, entities, variables, linkByKey, linkStatus)
		decision := decision(candidate.Key, reason)
		result.VariableSignals = append(result.VariableSignals, decision)
		signalByKey[candidate.Key], signalStatus[candidate.Key] = candidate, decision.Status
		if decision.Status == StatusPendingReview {
			result.ReviewerWorkPackage.VariableSignals = append(result.ReviewerWorkPackage.VariableSignals, candidate)
		}
	}

	for _, candidate := range submission.DirectImpacts {
		reason := validateImpact(candidate, evidence, entities, variables, relations, rules, linkByKey, signalByKey, signalStatus)
		decision := decision(candidate.Key, reason)
		result.DirectImpacts = append(result.DirectImpacts, decision)
		if decision.Status == StatusPendingReview {
			result.ReviewerWorkPackage.DirectImpacts = append(result.ReviewerWorkPackage.DirectImpacts, candidate)
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
	if !contains([]string{
		"event_subject", "actor", "affected_entity", "statement_source", "event_object", "context",
	}, candidate.EntityRole) {
		return "entity_role_invalid"
	}
	entity, exists := entities[candidate.EntityID]
	if !exists || entity.Status != "active" {
		return "entity_not_found"
	}
	if candidate.ResolutionMethod != "data_service_resolution" ||
		!entityMentionMatches(entity, candidate.Mention) ||
		countEntityMentionMatches(context.Entities, candidate.Mention) != 1 {
		return "entity_resolution_not_unique"
	}
	if !allEvidenceExists(candidate.EvidenceIDs, evidence) {
		return "evidence_not_in_event"
	}
	if !validConfidence(candidate.ResolutionConfidence) {
		return "confidence_invalid"
	}
	return ""
}

func validateSignal(
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
	if !contains([]string{"actual", "stated_intent", "source_forecast"}, candidate.AssertionModality) {
		return "assertion_modality_invalid"
	}
	if !allEvidenceExists(candidate.EvidenceIDs, evidence) {
		return "evidence_not_in_event"
	}
	if !validConfidence(candidate.ExtractionConfidence) {
		return "confidence_invalid"
	}
	if (candidate.AssertionModality == "stated_intent" || candidate.AssertionModality == "source_forecast") &&
		candidate.StatementAt == nil {
		return "statement_time_missing"
	}
	if candidate.AssertionModality == "stated_intent" && candidate.ValidFrom == nil {
		return "effective_period_missing"
	}
	if candidate.AssertionModality == "source_forecast" &&
		(candidate.ForecastPeriodStart == nil || candidate.ForecastPeriodEnd == nil) {
		return "forecast_period_missing"
	}
	if invalidTimeRange(candidate.ValidFrom, candidate.ValidUntil) ||
		invalidTimeRange(candidate.ForecastPeriodStart, candidate.ForecastPeriodEnd) {
		return "signal_time_invalid"
	}
	for _, measurement := range candidate.Measurements {
		if !contains([]string{"absolute_level", "absolute_change", "relative_change", "percentage_point_change"}, measurement.Role) ||
			!contains([]string{"exact", "range", "lower_bound", "upper_bound"}, measurement.Shape) {
			return "measurement_invalid"
		}
		if _, ok := evidence[measurement.EvidenceID]; !ok {
			return "evidence_not_in_event"
		}
		if !validMeasurement(measurement) {
			return "measurement_value_invalid"
		}
	}
	if measurementDirectionConflicts(candidate.Direction, candidate.Measurements) {
		return "measurement_direction_conflict"
	}
	return ""
}

func validateImpact(
	candidate DirectImpactCandidate,
	evidence map[string]Evidence,
	entities map[string]Entity,
	variables map[string]VariableDefinition,
	relations map[string]EntityRelation,
	rules map[string]DirectTransmissionRule,
	links map[string]EntityLinkCandidate,
	signals map[string]VariableSignalCandidate,
	signalStatus map[string]ReviewStatus,
) string {
	signal, exists := signals[candidate.SourceSignalKey]
	if !exists {
		return "source_signal_not_found"
	}
	if signalStatus[candidate.SourceSignalKey] == StatusRejected {
		return "upstream_rejected"
	}
	link := links[signal.SubjectLinkKey]
	subject := entities[link.EntityID]
	target, exists := entities[candidate.TargetEntityID]
	if !exists || target.Status != "active" {
		return "target_not_found"
	}
	if subject.ID == target.ID {
		return "target_equals_subject"
	}
	if !contains([]string{"commodity", "product", "chain_node", "company", "industry"}, target.Type) {
		return "target_type_not_allowed"
	}
	affected, exists := variables[definitionIdentity(candidate.AffectedVariableKey, candidate.AffectedVariableVersion)]
	if !exists || affected.Status != "active" {
		return "affected_variable_not_found"
	}
	if !contains(affected.ApplicableEntityTypes, target.Type) || !contains(affected.AllowedDirections, candidate.AffectedDirection) {
		return "affected_variable_not_applicable"
	}
	if !allEvidenceExists(candidate.EvidenceIDs, evidence) {
		return "evidence_not_in_event"
	}
	if !validConfidence(candidate.AssertionConfidence) {
		return "confidence_invalid"
	}
	switch candidate.DerivationType {
	case "event_explicit":
		if strings.TrimSpace(candidate.MechanismSummary) == "" {
			return "explicit_mechanism_missing"
		}
	case "rule_inferred":
		relation, relationExists := relations[candidate.EntityRelationID]
		rule, ruleExists := rules[definitionIdentity(candidate.RuleKey, candidate.RuleVersion)]
		if !relationExists || relation.Status != "active" ||
			relation.FromEntityID != subject.ID || relation.ToEntityID != target.ID {
			return "relation_not_found"
		}
		if !ruleExists || rule.Status != "approved" {
			return "rule_not_approved"
		}
		if rule.SourceEntityType != subject.Type || rule.SourceVariableKey != signal.VariableKey ||
			rule.SourceVariableVersion != signal.VariableVersion ||
			rule.SourceDirection != signal.Direction || rule.RelationType != relation.Type ||
			rule.TargetEntityType != target.Type || rule.AffectedVariableKey != candidate.AffectedVariableKey ||
			rule.AffectedVariableVersion != candidate.AffectedVariableVersion ||
			rule.AffectedDirection != candidate.AffectedDirection {
			return "rule_not_matched"
		}
	default:
		return "derivation_type_invalid"
	}
	return ""
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

func indexRelations(items []EntityRelation) map[string]EntityRelation {
	result := make(map[string]EntityRelation, len(items))
	for _, item := range items {
		result[item.ID] = item
	}
	return result
}

func indexRules(items []DirectTransmissionRule) map[string]DirectTransmissionRule {
	result := make(map[string]DirectTransmissionRule, len(items))
	for _, item := range items {
		result[definitionIdentity(item.Key, item.Version)] = item
	}
	return result
}

func allEvidenceExists(ids []string, evidence map[string]Evidence) bool {
	if len(ids) == 0 {
		return false
	}
	for _, id := range ids {
		if _, exists := evidence[id]; !exists {
			return false
		}
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
	if strings.TrimSpace(value) == "" {
		return true
	}
	parsed, ok := parseDecimal(value)
	return ok && parsed.Sign() >= 0 && parsed.Cmp(big.NewRat(1, 1)) <= 0
}

func invalidTimeRange(start, end *time.Time) bool {
	return start != nil && end != nil && end.Before(*start)
}

func validMeasurement(value MeasurementValue) bool {
	if strings.TrimSpace(value.RawText) == "" {
		return false
	}
	for _, candidate := range []*string{
		value.RawValue, value.RawLower, value.RawUpper,
		value.CanonicalValue, value.CanonicalLower, value.CanonicalUpper,
	} {
		if candidate != nil && !validDecimal(*candidate) {
			return false
		}
	}
	switch value.Shape {
	case "exact":
		if value.RawValue == nil || value.CanonicalValue == nil ||
			value.RawLower != nil || value.RawUpper != nil ||
			value.CanonicalLower != nil || value.CanonicalUpper != nil {
			return false
		}
	case "range":
		if value.RawLower == nil || value.RawUpper == nil ||
			value.CanonicalLower == nil || value.CanonicalUpper == nil ||
			value.RawValue != nil || value.CanonicalValue != nil ||
			(value.RawLower == nil) != (value.RawUpper == nil) {
			return false
		}
		if greaterDecimal(value.CanonicalLower, value.CanonicalUpper) ||
			(value.RawLower != nil && greaterDecimal(value.RawLower, value.RawUpper)) {
			return false
		}
	case "lower_bound":
		if value.RawLower == nil || value.CanonicalLower == nil ||
			value.RawValue != nil || value.RawUpper != nil ||
			value.CanonicalValue != nil || value.CanonicalUpper != nil {
			return false
		}
	case "upper_bound":
		if value.RawUpper == nil || value.CanonicalUpper == nil ||
			value.RawValue != nil || value.RawLower != nil ||
			value.CanonicalValue != nil || value.CanonicalLower != nil {
			return false
		}
	default:
		return false
	}
	if !validMeasurementUnitConversion(value) {
		return false
	}
	switch value.Role {
	case "relative_change":
		return value.CanonicalUnit == "percent"
	case "percentage_point_change":
		return value.CanonicalUnit == "percentage_point"
	default:
		return true
	}
}

func entityMentionMatches(entity Entity, mention string) bool {
	normalized := strings.ToLower(strings.TrimSpace(mention))
	if normalized == "" {
		return false
	}
	if strings.ToLower(strings.TrimSpace(entity.Name)) == normalized ||
		strings.ToLower(strings.TrimSpace(entity.CanonicalName)) == normalized {
		return true
	}
	for _, alias := range entity.Aliases {
		if strings.ToLower(strings.TrimSpace(alias)) == normalized {
			return true
		}
	}
	return false
}

func countEntityMentionMatches(entities []Entity, mention string) int {
	count := 0
	for _, entity := range entities {
		if entity.Status == "active" && entityMentionMatches(entity, mention) {
			count++
		}
	}
	return count
}

func validMeasurementUnitConversion(value MeasurementValue) bool {
	rawUnit := strings.ToLower(strings.TrimSpace(value.RawUnit))
	canonicalUnit := strings.ToLower(strings.TrimSpace(value.CanonicalUnit))
	if rawUnit == "" || canonicalUnit == "" {
		return false
	}
	allowed := rawUnit == canonicalUnit ||
		((rawUnit == "%" || rawUnit == "percent") && canonicalUnit == "percent") ||
		((rawUnit == "pp" || rawUnit == "percentage_point" || rawUnit == "个百分点") &&
			canonicalUnit == "percentage_point")
	if !allowed {
		return false
	}
	for _, pair := range [][2]*string{
		{value.RawValue, value.CanonicalValue},
		{value.RawLower, value.CanonicalLower},
		{value.RawUpper, value.CanonicalUpper},
	} {
		if pair[0] == nil && pair[1] == nil {
			continue
		}
		if pair[0] == nil || pair[1] == nil || !equalDecimal(*pair[0], *pair[1]) {
			return false
		}
	}
	return true
}

func equalDecimal(left, right string) bool {
	leftValue, leftOK := parseDecimal(left)
	rightValue, rightOK := parseDecimal(right)
	return leftOK && rightOK && leftValue.Cmp(rightValue) == 0
}

func validDecimal(value string) bool {
	trimmed := strings.TrimSpace(value)
	if !decimalPattern.MatchString(trimmed) {
		return false
	}
	_, ok := new(big.Rat).SetString(trimmed)
	return ok
}

func greaterDecimal(lower, upper *string) bool {
	left, leftOK := parseDecimal(*lower)
	right, rightOK := parseDecimal(*upper)
	return !leftOK || !rightOK || left.Cmp(right) > 0
}

func parseDecimal(value string) (*big.Rat, bool) {
	trimmed := strings.TrimSpace(value)
	if !decimalPattern.MatchString(trimmed) {
		return nil, false
	}
	parsed, ok := new(big.Rat).SetString(trimmed)
	return parsed, ok
}

func measurementDirectionConflicts(direction string, measurements []MeasurementValue) bool {
	changeMeasurements := make([]MeasurementValue, 0, len(measurements))
	hasAbsoluteLevel := false
	for _, measurement := range measurements {
		if measurement.Role == "absolute_level" {
			hasAbsoluteLevel = true
			continue
		}
		changeMeasurements = append(changeMeasurements, measurement)
	}
	if len(changeMeasurements) == 0 {
		return hasAbsoluteLevel && direction != "uncertain"
	}

	signs := make([]int, 0, len(changeMeasurements))
	for _, measurement := range changeMeasurements {
		sign, certain := measurementChangeSign(measurement)
		if !certain {
			return direction != "uncertain"
		}
		signs = append(signs, sign)
	}
	allSame := true
	for _, sign := range signs[1:] {
		if sign != signs[0] {
			allSame = false
			break
		}
	}
	if allSame {
		expected := map[int]string{-1: "decrease", 0: "unchanged", 1: "increase"}[signs[0]]
		return direction != expected
	}
	if hasOppositeSigns(signs) && measurementsHaveDistinctComparisonContext(changeMeasurements) {
		return direction != "mixed"
	}
	return direction != "uncertain"
}

func measurementChangeSign(measurement MeasurementValue) (int, bool) {
	values := []*string{
		measurement.CanonicalValue, measurement.CanonicalLower, measurement.CanonicalUpper,
	}
	sign := 0
	seen := false
	for _, value := range values {
		if value == nil {
			continue
		}
		parsed, ok := parseDecimal(*value)
		if !ok {
			return 0, false
		}
		current := parsed.Sign()
		if !seen {
			sign, seen = current, true
			continue
		}
		if current != sign {
			return 0, false
		}
	}
	return sign, seen
}

func hasOppositeSigns(signs []int) bool {
	hasPositive, hasNegative := false, false
	for _, sign := range signs {
		hasPositive = hasPositive || sign > 0
		hasNegative = hasNegative || sign < 0
	}
	return hasPositive && hasNegative
}

func measurementsHaveDistinctComparisonContext(measurements []MeasurementValue) bool {
	contexts := make(map[string]struct{}, len(measurements))
	for _, measurement := range measurements {
		basis := strings.TrimSpace(measurement.ComparisonBasis)
		period := strings.TrimSpace(measurement.ComparisonPeriod)
		if basis == "" && period == "" {
			return false
		}
		contexts[basis+"\x00"+period] = struct{}{}
	}
	return len(contexts) > 1
}
