package researchreasoningtreeimport

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	uuidPattern      = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	signalKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._:-]{0,127}$`)
)

type Publication struct {
	ThemeID        string          `json:"theme_id"`
	ReasoningTrees []ReasoningTree `json:"reasoning_trees"`
}

type ReasoningTree struct {
	IndustryChainEntityID     string       `json:"industry_chain_entity_id"`
	Title                     string       `json:"title"`
	DisplayOrder              int          `json:"display_order"`
	OneLineConclusion         string       `json:"one_line_conclusion"`
	FactSummary               *string      `json:"fact_summary"`
	TransmissionSummary       *string      `json:"transmission_summary"`
	ImpactDirection           string       `json:"impact_direction"`
	ImpactStrength            string       `json:"impact_strength"`
	ImpactSummary             *string      `json:"impact_summary"`
	ConclusionBoundarySummary *string      `json:"conclusion_boundary_summary"`
	SupportSummary            *string      `json:"support_summary"`
	CounterSummary            *string      `json:"counter_summary"`
	InvalidationConditions    []string     `json:"invalidation_conditions"`
	Checkpoints               []Checkpoint `json:"checkpoints"`
	Events                    []Event      `json:"events"`
	Nodes                     []Node       `json:"nodes"`
}

type Checkpoint struct {
	Type    string `json:"type"`
	Summary string `json:"summary"`
}

type Event struct {
	EventID      string `json:"event_id"`
	EvidenceRole string `json:"evidence_role"`
	DisplayOrder int    `json:"display_order"`
}

type Node struct {
	Position                         int      `json:"position"`
	ChainNodeEntityID                string   `json:"chain_node_entity_id"`
	StateSummary                     *string  `json:"state_summary"`
	ImpactDirection                  string   `json:"impact_direction"`
	ImpactStrength                   string   `json:"impact_strength"`
	ImpactSummary                    *string  `json:"impact_summary"`
	ReasoningBasisSummary            *string  `json:"reasoning_basis_summary"`
	EvidenceGapSummary               *string  `json:"evidence_gap_summary"`
	IncomingIndustryChainGraphEdgeID *string  `json:"incoming_industry_chain_graph_edge_id"`
	IncomingTransmissionTitle        *string  `json:"incoming_transmission_title"`
	IncomingTransmissionMechanism    *string  `json:"incoming_transmission_mechanism"`
	IncomingConditionSummary         *string  `json:"incoming_condition_summary"`
	Signals                          []Signal `json:"signals"`
}

type Signal struct {
	VariableSignalKey string `json:"variable_signal_key"`
	SignalRole        string `json:"signal_role"`
	SignalDirection   string `json:"signal_direction"`
	DisplaySummary    string `json:"display_summary"`
	DisplayOrder      int    `json:"display_order"`
}

type ValidationError struct {
	IndustryChainEntityID string
	Path                  string
	Reference             string
	Message               string
}

func (e *ValidationError) Error() string {
	if e == nil {
		return "research Reason Tree publication validation failed"
	}
	location := e.Path
	if e.IndustryChainEntityID != "" {
		location = e.IndustryChainEntityID + ": " + location
	}
	if e.Reference != "" {
		return fmt.Sprintf("%s: %s (%s)", location, e.Message, e.Reference)
	}
	return fmt.Sprintf("%s: %s", location, e.Message)
}

func DecodeStrict(reader io.Reader) (Publication, error) {
	return decodeStrictJSON(reader)
}

func (p Publication) Validate() error {
	if !uuidPattern.MatchString(p.ThemeID) {
		return invalid("", "theme_id", p.ThemeID, "must be a standard lowercase UUID")
	}
	if len(p.ReasoningTrees) == 0 {
		return invalid("", "reasoning_trees", "", "must contain at least one Reason Tree")
	}
	seenChains := make(map[string]struct{}, len(p.ReasoningTrees))
	signalSnapshots := make(map[string]Signal)
	for index, tree := range p.ReasoningTrees {
		path := fmt.Sprintf("reasoning_trees[%d]", index)
		if tree.DisplayOrder != index+1 {
			return invalid(tree.IndustryChainEntityID, path+".display_order", fmt.Sprint(tree.DisplayOrder), "must be contiguous from 1")
		}
		if !uuidPattern.MatchString(tree.IndustryChainEntityID) {
			return invalid(tree.IndustryChainEntityID, path+".industry_chain_entity_id", tree.IndustryChainEntityID, "must be a standard lowercase UUID")
		}
		if _, duplicate := seenChains[tree.IndustryChainEntityID]; duplicate {
			return invalid(tree.IndustryChainEntityID, path+".industry_chain_entity_id", tree.IndustryChainEntityID, "must be unique within the Theme")
		}
		seenChains[tree.IndustryChainEntityID] = struct{}{}
		if err := tree.validate(path, signalSnapshots); err != nil {
			return err
		}
	}
	return nil
}

func (t ReasoningTree) validate(path string, snapshots map[string]Signal) error {
	if err := requiredText(t.IndustryChainEntityID, path+".title", t.Title, 300); err != nil {
		return err
	}
	if err := requiredText(t.IndustryChainEntityID, path+".one_line_conclusion", t.OneLineConclusion, 1000); err != nil {
		return err
	}
	if !oneOf(t.ImpactDirection, "positive", "negative", "mixed", "neutral", "uncertain") {
		return invalid(t.IndustryChainEntityID, path+".impact_direction", t.ImpactDirection, "has an unsupported value")
	}
	if !oneOf(t.ImpactStrength, "strong", "medium", "weak", "unknown") {
		return invalid(t.IndustryChainEntityID, path+".impact_strength", t.ImpactStrength, "has an unsupported value")
	}
	for _, field := range []struct {
		name  string
		value *string
		max   int
	}{
		{"fact_summary", t.FactSummary, 4000},
		{"transmission_summary", t.TransmissionSummary, 4000},
		{"impact_summary", t.ImpactSummary, 2000},
		{"conclusion_boundary_summary", t.ConclusionBoundarySummary, 4000},
		{"support_summary", t.SupportSummary, 4000},
		{"counter_summary", t.CounterSummary, 4000},
	} {
		if err := optionalText(t.IndustryChainEntityID, path+"."+field.name, field.value, field.max); err != nil {
			return err
		}
	}
	for index, condition := range t.InvalidationConditions {
		if err := requiredText(t.IndustryChainEntityID, fmt.Sprintf("%s.invalidation_conditions[%d]", path, index), condition, 2000); err != nil {
			return err
		}
	}
	for index, checkpoint := range t.Checkpoints {
		checkpointPath := fmt.Sprintf("%s.checkpoints[%d]", path, index)
		if !oneOf(checkpoint.Type, "event", "relationship", "metric") {
			return invalid(t.IndustryChainEntityID, checkpointPath+".type", checkpoint.Type, "has an unsupported value")
		}
		if err := requiredText(t.IndustryChainEntityID, checkpointPath+".summary", checkpoint.Summary, 2000); err != nil {
			return err
		}
	}
	seenEvents := make(map[string]struct{}, len(t.Events))
	for index, event := range t.Events {
		eventPath := fmt.Sprintf("%s.events[%d]", path, index)
		if event.DisplayOrder != index+1 {
			return invalid(t.IndustryChainEntityID, eventPath+".display_order", fmt.Sprint(event.DisplayOrder), "must be contiguous from 1")
		}
		if !uuidPattern.MatchString(event.EventID) {
			return invalid(t.IndustryChainEntityID, eventPath+".event_id", event.EventID, "must be a standard lowercase UUID")
		}
		if _, duplicate := seenEvents[event.EventID]; duplicate {
			return invalid(t.IndustryChainEntityID, eventPath+".event_id", event.EventID, "must be unique within the Tree")
		}
		seenEvents[event.EventID] = struct{}{}
		if !oneOf(event.EvidenceRole, "driver", "supporting", "contradicting", "context") {
			return invalid(t.IndustryChainEntityID, eventPath+".evidence_role", event.EvidenceRole, "has an unsupported value")
		}
	}
	if len(t.Nodes) == 0 {
		return invalid(t.IndustryChainEntityID, path+".nodes", "", "must contain at least one Node")
	}
	seenNodes := make(map[string]struct{}, len(t.Nodes))
	for index, node := range t.Nodes {
		nodePath := fmt.Sprintf("%s.nodes[%d]", path, index)
		if node.Position != index+1 {
			return invalid(t.IndustryChainEntityID, nodePath+".position", fmt.Sprint(node.Position), "must be contiguous from 1")
		}
		if !uuidPattern.MatchString(node.ChainNodeEntityID) {
			return invalid(t.IndustryChainEntityID, nodePath+".chain_node_entity_id", node.ChainNodeEntityID, "must be a standard lowercase UUID")
		}
		if _, duplicate := seenNodes[node.ChainNodeEntityID]; duplicate {
			return invalid(t.IndustryChainEntityID, nodePath+".chain_node_entity_id", node.ChainNodeEntityID, "must be unique within the Tree")
		}
		seenNodes[node.ChainNodeEntityID] = struct{}{}
		if !oneOf(node.ImpactDirection, "positive", "negative", "mixed", "neutral", "uncertain") {
			return invalid(t.IndustryChainEntityID, nodePath+".impact_direction", node.ImpactDirection, "has an unsupported value")
		}
		if !oneOf(node.ImpactStrength, "strong", "medium", "weak", "unknown") {
			return invalid(t.IndustryChainEntityID, nodePath+".impact_strength", node.ImpactStrength, "has an unsupported value")
		}
		for _, field := range []struct {
			name  string
			value *string
			max   int
		}{
			{"state_summary", node.StateSummary, 2000},
			{"impact_summary", node.ImpactSummary, 2000},
			{"reasoning_basis_summary", node.ReasoningBasisSummary, 4000},
			{"evidence_gap_summary", node.EvidenceGapSummary, 4000},
		} {
			if err := optionalText(t.IndustryChainEntityID, nodePath+"."+field.name, field.value, field.max); err != nil {
				return err
			}
		}
		if index == 0 {
			if node.IncomingIndustryChainGraphEdgeID != nil || node.IncomingTransmissionTitle != nil ||
				node.IncomingTransmissionMechanism != nil || node.IncomingConditionSummary != nil {
				return invalid(t.IndustryChainEntityID, nodePath+".incoming_*", "", "must all be null for the first Node")
			}
		} else {
			if node.IncomingIndustryChainGraphEdgeID != nil && !uuidPattern.MatchString(*node.IncomingIndustryChainGraphEdgeID) {
				return invalid(t.IndustryChainEntityID, nodePath+".incoming_industry_chain_graph_edge_id", *node.IncomingIndustryChainGraphEdgeID, "must be a standard lowercase UUID")
			}
			for _, field := range []struct {
				name  string
				value *string
			}{
				{"incoming_transmission_title", node.IncomingTransmissionTitle},
				{"incoming_transmission_mechanism", node.IncomingTransmissionMechanism},
				{"incoming_condition_summary", node.IncomingConditionSummary},
			} {
				if field.value == nil {
					return invalid(t.IndustryChainEntityID, nodePath+"."+field.name, "", "is required after the first Node")
				}
				if err := requiredText(t.IndustryChainEntityID, nodePath+"."+field.name, *field.value, 4000); err != nil {
					return err
				}
			}
		}
		if err := validateSignals(t.IndustryChainEntityID, nodePath, node.Signals, snapshots); err != nil {
			return err
		}
	}
	return nil
}

func validateSignals(chainID, nodePath string, signals []Signal, snapshots map[string]Signal) error {
	if len(signals) < 1 || len(signals) > 5 {
		return invalid(chainID, nodePath+".signals", "", "must contain 1..5 Signal snapshots")
	}
	seen := make(map[string]struct{}, len(signals))
	primaryCount := 0
	for index, signal := range signals {
		path := fmt.Sprintf("%s.signals[%d]", nodePath, index)
		if signal.DisplayOrder != index+1 {
			return invalid(chainID, path+".display_order", fmt.Sprint(signal.DisplayOrder), "must be contiguous from 1")
		}
		if !signalKeyPattern.MatchString(signal.VariableSignalKey) {
			return invalid(chainID, path+".variable_signal_key", signal.VariableSignalKey, "must match ^[a-z0-9][a-z0-9._:-]{0,127}$")
		}
		if _, duplicate := seen[signal.VariableSignalKey]; duplicate {
			return invalid(chainID, path+".variable_signal_key", signal.VariableSignalKey, "must be unique within the Node")
		}
		seen[signal.VariableSignalKey] = struct{}{}
		if !oneOf(signal.SignalRole, "primary", "supporting", "contradicting") {
			return invalid(chainID, path+".signal_role", signal.SignalRole, "has an unsupported value")
		}
		if signal.SignalRole == "primary" {
			primaryCount++
			if signal.DisplayOrder != 1 {
				return invalid(chainID, path+".display_order", fmt.Sprint(signal.DisplayOrder), "primary Signal must have display_order 1")
			}
		}
		if !oneOf(signal.SignalDirection, "increase", "decrease", "mixed", "unchanged", "uncertain") {
			return invalid(chainID, path+".signal_direction", signal.SignalDirection, "has an unsupported value")
		}
		trimmed := strings.TrimSpace(signal.DisplaySummary)
		if trimmed == "" || utf8.RuneCountInString(trimmed) > 200 || trimmed != signal.DisplaySummary {
			return invalid(chainID, path+".display_summary", "", "must be a trimmed 1..200 character string")
		}
		if prior, exists := snapshots[signal.VariableSignalKey]; exists {
			if prior.SignalDirection != signal.SignalDirection || prior.DisplaySummary != signal.DisplaySummary {
				return invalid(chainID, path+".variable_signal_key", signal.VariableSignalKey, "must keep the same direction and display summary within the analysis batch")
			}
		} else {
			snapshots[signal.VariableSignalKey] = signal
		}
	}
	if primaryCount != 1 {
		return invalid(chainID, nodePath+".signals", "", "must contain exactly one primary Signal")
	}
	return nil
}

func requiredText(chainID, path, value string, max int) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return invalid(chainID, path, "", "is required")
	}
	if utf8.RuneCountInString(trimmed) > max {
		return invalid(chainID, path, "", fmt.Sprintf("must contain at most %d characters", max))
	}
	return nil
}

func optionalText(chainID, path string, value *string, max int) error {
	if value != nil && utf8.RuneCountInString(strings.TrimSpace(*value)) > max {
		return invalid(chainID, path, "", fmt.Sprintf("must contain at most %d characters", max))
	}
	return nil
}

func invalid(chainID, path, reference, message string) *ValidationError {
	return &ValidationError{IndustryChainEntityID: chainID, Path: path, Reference: reference, Message: message}
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
