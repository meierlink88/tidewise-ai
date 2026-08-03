package researchpublication

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/identity"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchreasoningtreeimport"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchthemeimport"
)

const SnapshotPublicationMode = "analyst_snapshot"

var (
	snapshotKeyPattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9._:-]{0,127}$`)
	snapshotUUIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
)

type SnapshotAggregate struct {
	PublicationMode      string                  `json:"publication_mode"`
	AnalysisBatchID      string                  `json:"analysis_batch_id"`
	AnalysisAsOf         string                  `json:"analysis_as_of"`
	DiscoveryWindowStart string                  `json:"discovery_window_start"`
	DiscoveryWindowEnd   string                  `json:"discovery_window_end"`
	Theme                SnapshotTheme           `json:"theme"`
	ReasoningTrees       []SnapshotReasoningTree `json:"reasoning_trees"`
}

type SnapshotTheme struct {
	ThemeKey                  string           `json:"theme_key"`
	Title                     string           `json:"title"`
	OneLineConclusion         string           `json:"one_line_conclusion"`
	ConclusionDirection       string           `json:"conclusion_direction"`
	ImpactStrength            string           `json:"impact_strength"`
	AttentionLevel            *string          `json:"attention_level"`
	ConclusionStatus          *string          `json:"conclusion_status"`
	TransmissionStage         string           `json:"transmission_stage"`
	InvestmentGuidanceAction  string           `json:"investment_guidance_action"`
	InvestmentGuidanceSummary string           `json:"investment_guidance_summary"`
	TimeHorizonCategory       string           `json:"time_horizon_category"`
	TimeHorizonSummary        *string          `json:"time_horizon_summary"`
	TransmissionSummary       *string          `json:"transmission_summary"`
	CheckpointSummary         *string          `json:"checkpoint_summary"`
	RiskSummary               *string          `json:"risk_summary"`
	Impacts                   []SnapshotImpact `json:"impacts"`
	Events                    []SnapshotEvent  `json:"events"`
}

type SnapshotImpact struct {
	NodeKey         string  `json:"node_key"`
	DisplayName     string  `json:"display_name"`
	RelationRole    string  `json:"relation_role"`
	ImpactDirection string  `json:"impact_direction"`
	ImpactSummary   *string `json:"impact_summary"`
	DisplayOrder    int     `json:"display_order"`
}

type SnapshotEvent struct {
	EventID        string   `json:"event_id"`
	EvidenceIDs    []string `json:"evidence_ids,omitempty"`
	EvidenceRole   string   `json:"evidence_role"`
	SupportedClaim *string  `json:"supported_claim"`
}

type SnapshotReasoningTree struct {
	TreeKey                   string                                   `json:"tree_key"`
	DisplayName               string                                   `json:"display_name"`
	Title                     string                                   `json:"title"`
	DisplayOrder              int                                      `json:"display_order"`
	OneLineConclusion         string                                   `json:"one_line_conclusion"`
	FactSummary               *string                                  `json:"fact_summary"`
	TransmissionSummary       *string                                  `json:"transmission_summary"`
	ImpactDirection           string                                   `json:"impact_direction"`
	ImpactStrength            string                                   `json:"impact_strength"`
	ImpactSummary             *string                                  `json:"impact_summary"`
	ConclusionBoundarySummary *string                                  `json:"conclusion_boundary_summary"`
	SupportSummary            *string                                  `json:"support_summary"`
	CounterSummary            *string                                  `json:"counter_summary"`
	InvalidationConditions    []string                                 `json:"invalidation_conditions"`
	Checkpoints               []researchreasoningtreeimport.Checkpoint `json:"checkpoints"`
	Events                    []SnapshotTreeEvent                      `json:"events"`
	Nodes                     []SnapshotNode                           `json:"nodes"`
}

type SnapshotTreeEvent struct {
	EventID      string   `json:"event_id"`
	EvidenceIDs  []string `json:"evidence_ids,omitempty"`
	EvidenceRole string   `json:"evidence_role"`
	DisplayOrder int      `json:"display_order"`
}

type SnapshotNode struct {
	NodeKey               string                        `json:"node_key"`
	DisplayName           string                        `json:"display_name"`
	Position              int                           `json:"position"`
	StateSummary          *string                       `json:"state_summary"`
	ImpactDirection       string                        `json:"impact_direction"`
	ImpactStrength        string                        `json:"impact_strength"`
	ImpactSummary         *string                       `json:"impact_summary"`
	ReasoningBasisSummary *string                       `json:"reasoning_basis_summary"`
	EvidenceGapSummary    *string                       `json:"evidence_gap_summary"`
	IncomingTransmission  *SnapshotIncomingTransmission `json:"incoming_transmission"`
	Signals               []SnapshotSignal              `json:"signals"`
}

type SnapshotIncomingTransmission struct {
	Title            *string `json:"title"`
	Mechanism        string  `json:"mechanism"`
	ConditionSummary *string `json:"condition_summary"`
}

type SnapshotSignal struct {
	SignalKey      string  `json:"signal_key"`
	DisplaySummary string  `json:"display_summary"`
	Role           string  `json:"role"`
	DisplayOrder   int     `json:"display_order"`
	VariableName   *string `json:"variable_name"`
	Direction      *string `json:"direction"`
}

func (a SnapshotAggregate) Validate() (time.Time, string, error) {
	if a.PublicationMode != SnapshotPublicationMode {
		return time.Time{}, "", invalid("publication_mode", a.PublicationMode, "must be analyst_snapshot")
	}
	if value := strings.TrimSpace(a.AnalysisBatchID); value == "" || utf8.RuneCountInString(value) > 200 {
		return time.Time{}, "", invalid("analysis_batch_id", "", "must contain 1..200 characters")
	}
	asOf, err := snapshotUTCTime("analysis_as_of", a.AnalysisAsOf)
	if err != nil {
		return time.Time{}, "", err
	}
	windowStart, err := snapshotUTCTime("discovery_window_start", a.DiscoveryWindowStart)
	if err != nil {
		return time.Time{}, "", err
	}
	windowEnd, err := snapshotUTCTime("discovery_window_end", a.DiscoveryWindowEnd)
	if err != nil {
		return time.Time{}, "", err
	}
	if !windowStart.Before(windowEnd) {
		return time.Time{}, "", invalid("discovery_window_end", a.DiscoveryWindowEnd, "must be greater than discovery_window_start")
	}
	if windowEnd.After(asOf) {
		return time.Time{}, "", invalid("discovery_window_end", a.DiscoveryWindowEnd, "must not be later than analysis_as_of")
	}
	if !snapshotKeyPattern.MatchString(a.Theme.ThemeKey) {
		return time.Time{}, "", invalid("theme.theme_key", a.Theme.ThemeKey, "must match the local key pattern")
	}
	if err := a.Theme.validate(); err != nil {
		return time.Time{}, "", err
	}
	if len(a.ReasoningTrees) == 0 {
		return time.Time{}, "", invalid("reasoning_trees", "", "must contain at least one Reason Tree")
	}
	themeEvents := make(map[string]struct{}, len(a.Theme.Events))
	for _, event := range a.Theme.Events {
		themeEvents[event.EventID] = struct{}{}
	}
	treeKeys := make(map[string]struct{}, len(a.ReasoningTrees))
	coveredImpacts := make(map[string]struct{}, len(a.Theme.Impacts))
	for treeIndex, tree := range a.ReasoningTrees {
		path := fmt.Sprintf("reasoning_trees[%d]", treeIndex)
		if tree.DisplayOrder != treeIndex+1 {
			return time.Time{}, "", invalid(path+".display_order", fmt.Sprint(tree.DisplayOrder), "must be contiguous from 1")
		}
		if !snapshotKeyPattern.MatchString(tree.TreeKey) {
			return time.Time{}, "", invalid(path+".tree_key", tree.TreeKey, "must match the local key pattern")
		}
		if _, duplicate := treeKeys[tree.TreeKey]; duplicate {
			return time.Time{}, "", invalid(path+".tree_key", tree.TreeKey, "must be unique within the aggregate")
		}
		treeKeys[tree.TreeKey] = struct{}{}
		nodeKeys, err := tree.validate(path, themeEvents)
		if err != nil {
			return time.Time{}, "", err
		}
		for _, impact := range a.Theme.Impacts {
			if _, ok := nodeKeys[impact.NodeKey]; ok {
				coveredImpacts[impact.NodeKey] = struct{}{}
			}
		}
	}
	for _, impact := range a.Theme.Impacts {
		if _, ok := coveredImpacts[impact.NodeKey]; !ok {
			return time.Time{}, "", invalid("theme.impacts", impact.NodeKey, "must be covered by at least one Reason Tree")
		}
	}
	return asOf, researchthemeimport.ThemeID(a.AnalysisBatchID, a.Theme.ThemeKey), nil
}

func (t SnapshotTheme) validate() error {
	for _, field := range []struct {
		path  string
		value string
		max   int
	}{{"theme.title", t.Title, 300}, {"theme.one_line_conclusion", t.OneLineConclusion, 1000}, {"theme.investment_guidance_summary", t.InvestmentGuidanceSummary, 2000}} {
		if err := snapshotRequiredText(field.path, field.value, field.max); err != nil {
			return err
		}
	}
	if !snapshotOneOf(t.ConclusionDirection, "positive", "negative", "mixed", "neutral", "uncertain") {
		return invalid("theme.conclusion_direction", t.ConclusionDirection, "has an unsupported value")
	}
	if !snapshotOneOf(t.ImpactStrength, "strong", "medium", "weak", "unknown") {
		return invalid("theme.impact_strength", t.ImpactStrength, "has an unsupported value")
	}
	if t.AttentionLevel != nil && !snapshotOneOf(*t.AttentionLevel, "high", "medium", "low") {
		return invalid("theme.attention_level", *t.AttentionLevel, "has an unsupported value")
	}
	if t.ConclusionStatus != nil && !snapshotOneOf(*t.ConclusionStatus, "supported", "partial", "conflicted") {
		return invalid("theme.conclusion_status", *t.ConclusionStatus, "has an unsupported value")
	}
	if !snapshotOneOf(t.TransmissionStage, "identification", "validation", "diffusion", "dampening") {
		return invalid("theme.transmission_stage", t.TransmissionStage, "has an unsupported value")
	}
	if !snapshotOneOf(t.InvestmentGuidanceAction, "focus", "avoid", "observe", "differentiate") {
		return invalid("theme.investment_guidance_action", t.InvestmentGuidanceAction, "has an unsupported value")
	}
	if !snapshotOneOf(t.TimeHorizonCategory, "short_term", "medium_term", "long_term", "custom") {
		return invalid("theme.time_horizon_category", t.TimeHorizonCategory, "has an unsupported value")
	}
	for _, field := range []struct {
		path  string
		value *string
		max   int
	}{{"theme.time_horizon_summary", t.TimeHorizonSummary, 1000}, {"theme.transmission_summary", t.TransmissionSummary, 4000}, {"theme.checkpoint_summary", t.CheckpointSummary, 4000}, {"theme.risk_summary", t.RiskSummary, 4000}} {
		if err := snapshotOptionalText(field.path, field.value, field.max); err != nil {
			return err
		}
	}
	if len(t.Impacts) == 0 {
		return invalid("theme.impacts", "", "must contain at least one Theme Impact")
	}
	impactKeys := make(map[string]struct{}, len(t.Impacts))
	for index, impact := range t.Impacts {
		path := fmt.Sprintf("theme.impacts[%d]", index)
		if impact.DisplayOrder != index+1 {
			return invalid(path+".display_order", fmt.Sprint(impact.DisplayOrder), "must be contiguous from 1")
		}
		if !snapshotKeyPattern.MatchString(impact.NodeKey) {
			return invalid(path+".node_key", impact.NodeKey, "must match the local key pattern")
		}
		if _, duplicate := impactKeys[impact.NodeKey]; duplicate {
			return invalid(path+".node_key", impact.NodeKey, "must be unique within the Theme")
		}
		impactKeys[impact.NodeKey] = struct{}{}
		if err := snapshotRequiredText(path+".display_name", impact.DisplayName, 300); err != nil {
			return err
		}
		if !snapshotOneOf(impact.RelationRole, "driver", "beneficiary", "constraint", "exposure") {
			return invalid(path+".relation_role", impact.RelationRole, "has an unsupported value")
		}
		if !snapshotOneOf(impact.ImpactDirection, "positive", "negative", "mixed", "neutral", "uncertain") {
			return invalid(path+".impact_direction", impact.ImpactDirection, "has an unsupported value")
		}
		if err := snapshotOptionalText(path+".impact_summary", impact.ImpactSummary, 2000); err != nil {
			return err
		}
	}
	if len(t.Events) == 0 {
		return invalid("theme.events", "", "must contain at least one Event")
	}
	for index, event := range t.Events {
		path := fmt.Sprintf("theme.events[%d]", index)
		if err := validateSnapshotEvent(path, event.EventID, event.EvidenceIDs, event.EvidenceRole); err != nil {
			return err
		}
		if index > 0 && event.EventID <= t.Events[index-1].EventID {
			return invalid(path+".event_id", event.EventID, "must be unique and sorted by event_id")
		}
		if err := snapshotOptionalText(path+".supported_claim", event.SupportedClaim, 2000); err != nil {
			return err
		}
	}
	return nil
}

func (t SnapshotReasoningTree) validate(path string, themeEvents map[string]struct{}) (map[string]struct{}, error) {
	if err := snapshotRequiredText(path+".display_name", t.DisplayName, 300); err != nil {
		return nil, err
	}
	if err := snapshotRequiredText(path+".title", t.Title, 300); err != nil {
		return nil, err
	}
	if err := snapshotRequiredText(path+".one_line_conclusion", t.OneLineConclusion, 1000); err != nil {
		return nil, err
	}
	if !snapshotOneOf(t.ImpactDirection, "positive", "negative", "mixed", "neutral", "uncertain") {
		return nil, invalid(path+".impact_direction", t.ImpactDirection, "has an unsupported value")
	}
	if !snapshotOneOf(t.ImpactStrength, "strong", "medium", "weak", "unknown") {
		return nil, invalid(path+".impact_strength", t.ImpactStrength, "has an unsupported value")
	}
	for _, field := range []struct {
		name  string
		value *string
		max   int
	}{{"fact_summary", t.FactSummary, 4000}, {"transmission_summary", t.TransmissionSummary, 4000}, {"impact_summary", t.ImpactSummary, 2000}, {"conclusion_boundary_summary", t.ConclusionBoundarySummary, 4000}, {"support_summary", t.SupportSummary, 4000}, {"counter_summary", t.CounterSummary, 4000}} {
		if err := snapshotOptionalText(path+"."+field.name, field.value, field.max); err != nil {
			return nil, err
		}
	}
	for index, condition := range t.InvalidationConditions {
		if err := snapshotRequiredText(fmt.Sprintf("%s.invalidation_conditions[%d]", path, index), condition, 2000); err != nil {
			return nil, err
		}
	}
	for index, checkpoint := range t.Checkpoints {
		checkpointPath := fmt.Sprintf("%s.checkpoints[%d]", path, index)
		if !snapshotOneOf(checkpoint.Type, "event", "relationship", "metric") {
			return nil, invalid(checkpointPath+".type", checkpoint.Type, "has an unsupported value")
		}
		if err := snapshotRequiredText(checkpointPath+".summary", checkpoint.Summary, 2000); err != nil {
			return nil, err
		}
	}
	if len(t.Events) == 0 {
		return nil, invalid(path+".events", "", "must contain at least one Event")
	}
	seenEvents := make(map[string]struct{}, len(t.Events))
	for index, event := range t.Events {
		eventPath := fmt.Sprintf("%s.events[%d]", path, index)
		if event.DisplayOrder != index+1 {
			return nil, invalid(eventPath+".display_order", fmt.Sprint(event.DisplayOrder), "must be contiguous from 1")
		}
		if err := validateSnapshotEvent(eventPath, event.EventID, event.EvidenceIDs, event.EvidenceRole); err != nil {
			return nil, err
		}
		if _, duplicate := seenEvents[event.EventID]; duplicate {
			return nil, invalid(eventPath+".event_id", event.EventID, "must be unique within the Tree")
		}
		seenEvents[event.EventID] = struct{}{}
		if _, ok := themeEvents[event.EventID]; !ok {
			return nil, invalid(eventPath+".event_id", event.EventID, "must belong to Theme events")
		}
	}
	if len(t.Nodes) == 0 {
		return nil, invalid(path+".nodes", "", "must contain at least one Node")
	}
	nodeKeys := make(map[string]struct{}, len(t.Nodes))
	for index, node := range t.Nodes {
		nodePath := fmt.Sprintf("%s.nodes[%d]", path, index)
		if node.Position != index+1 {
			return nil, invalid(nodePath+".position", fmt.Sprint(node.Position), "must be contiguous from 1")
		}
		if !snapshotKeyPattern.MatchString(node.NodeKey) {
			return nil, invalid(nodePath+".node_key", node.NodeKey, "must match the local key pattern")
		}
		if _, duplicate := nodeKeys[node.NodeKey]; duplicate {
			return nil, invalid(nodePath+".node_key", node.NodeKey, "must be unique within the Tree")
		}
		nodeKeys[node.NodeKey] = struct{}{}
		if err := node.validate(nodePath, index == 0); err != nil {
			return nil, err
		}
	}
	return nodeKeys, nil
}

func (n SnapshotNode) validate(path string, root bool) error {
	if err := snapshotRequiredText(path+".display_name", n.DisplayName, 300); err != nil {
		return err
	}
	if !snapshotOneOf(n.ImpactDirection, "positive", "negative", "mixed", "neutral", "uncertain") {
		return invalid(path+".impact_direction", n.ImpactDirection, "has an unsupported value")
	}
	if !snapshotOneOf(n.ImpactStrength, "strong", "medium", "weak", "unknown") {
		return invalid(path+".impact_strength", n.ImpactStrength, "has an unsupported value")
	}
	for _, field := range []struct {
		name  string
		value *string
		max   int
	}{{"state_summary", n.StateSummary, 2000}, {"impact_summary", n.ImpactSummary, 2000}, {"reasoning_basis_summary", n.ReasoningBasisSummary, 4000}, {"evidence_gap_summary", n.EvidenceGapSummary, 4000}} {
		if err := snapshotOptionalText(path+"."+field.name, field.value, field.max); err != nil {
			return err
		}
	}
	if root && n.IncomingTransmission != nil {
		return invalid(path+".incoming_transmission", "", "must be null for the first Node")
	}
	if !root {
		if n.IncomingTransmission == nil {
			return invalid(path+".incoming_transmission", "", "is required after the first Node")
		}
		if err := snapshotRequiredText(path+".incoming_transmission.mechanism", n.IncomingTransmission.Mechanism, 4000); err != nil {
			return err
		}
		if err := snapshotOptionalText(path+".incoming_transmission.title", n.IncomingTransmission.Title, 4000); err != nil {
			return err
		}
		if err := snapshotOptionalText(path+".incoming_transmission.condition_summary", n.IncomingTransmission.ConditionSummary, 4000); err != nil {
			return err
		}
	}
	return validateSnapshotSignals(path, n.Signals)
}

func validateSnapshotSignals(nodePath string, signals []SnapshotSignal) error {
	if len(signals) < 1 || len(signals) > 5 {
		return invalid(nodePath+".signals", "", "must contain 1..5 Signal snapshots")
	}
	seen := make(map[string]struct{}, len(signals))
	primaryCount := 0
	for index, signal := range signals {
		path := fmt.Sprintf("%s.signals[%d]", nodePath, index)
		if signal.DisplayOrder != index+1 {
			return invalid(path+".display_order", fmt.Sprint(signal.DisplayOrder), "must be contiguous from 1")
		}
		if !snapshotKeyPattern.MatchString(signal.SignalKey) {
			return invalid(path+".signal_key", signal.SignalKey, "must match the local key pattern")
		}
		if _, duplicate := seen[signal.SignalKey]; duplicate {
			return invalid(path+".signal_key", signal.SignalKey, "must be unique within the Node")
		}
		seen[signal.SignalKey] = struct{}{}
		if err := snapshotRequiredText(path+".display_summary", signal.DisplaySummary, 200); err != nil {
			return err
		}
		if !snapshotOneOf(signal.Role, "primary", "supporting", "contradicting") {
			return invalid(path+".role", signal.Role, "has an unsupported value")
		}
		if signal.Role == "primary" {
			primaryCount++
			if signal.DisplayOrder != 1 {
				return invalid(path+".display_order", fmt.Sprint(signal.DisplayOrder), "primary Signal must be first")
			}
		}
		if err := snapshotOptionalText(path+".variable_name", signal.VariableName, 200); err != nil {
			return err
		}
		if signal.Direction != nil && !snapshotOneOf(*signal.Direction, "increase", "decrease", "mixed", "unchanged", "uncertain") {
			return invalid(path+".direction", *signal.Direction, "has an unsupported value")
		}
	}
	if primaryCount != 1 {
		return invalid(nodePath+".signals", "", "must contain exactly one primary Signal")
	}
	return nil
}

func validateSnapshotEvent(path, eventID string, evidenceIDs []string, role string) error {
	if !snapshotUUIDPattern.MatchString(eventID) {
		return invalid(path+".event_id", eventID, "must be a standard lowercase UUID")
	}
	if !snapshotOneOf(role, "driver", "supporting", "contradicting", "context") {
		return invalid(path+".evidence_role", role, "has an unsupported value")
	}
	for index, evidenceID := range evidenceIDs {
		if !snapshotUUIDPattern.MatchString(evidenceID) {
			return invalid(fmt.Sprintf("%s.evidence_ids[%d]", path, index), evidenceID, "must be a standard lowercase UUID")
		}
		if index > 0 && evidenceID <= evidenceIDs[index-1] {
			return invalid(fmt.Sprintf("%s.evidence_ids[%d]", path, index), evidenceID, "must be unique and sorted")
		}
	}
	return nil
}

func SnapshotTreeID(themeID, treeKey string) string {
	return identity.NormalizeUUID("research_reasoning_tree_snapshot", themeID, treeKey)
}

func SnapshotNodeID(treeID, nodeKey string) string {
	return identity.NormalizeUUID("research_reasoning_tree_node_snapshot", treeID, nodeKey)
}

func CanonicalSnapshotHash(value SnapshotAggregate) (string, error) {
	return researchthemeimport.CanonicalHashValue(value, "research Theme analyst snapshot V3")
}

func snapshotUTCTime(path, value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, invalid(path, value, "must be an RFC3339 UTC timestamp")
	}
	_, offset := parsed.Zone()
	if offset != 0 {
		return time.Time{}, invalid(path, value, "must use UTC")
	}
	return parsed.UTC(), nil
}

func snapshotRequiredText(path, value string, max int) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return invalid(path, "", "is required")
	}
	if trimmed != value || utf8.RuneCountInString(value) > max {
		return invalid(path, "", fmt.Sprintf("must be a trimmed 1..%d character string", max))
	}
	return nil
}

func snapshotOptionalText(path string, value *string, max int) error {
	if value == nil {
		return nil
	}
	if strings.TrimSpace(*value) != *value || utf8.RuneCountInString(*value) > max {
		return invalid(path, "", fmt.Sprintf("must be a trimmed string containing at most %d characters", max))
	}
	return nil
}

func snapshotOneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
