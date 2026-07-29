package researchpublication

import (
	"fmt"
	"regexp"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchreasoningtreeimport"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchthemeimport"
)

var hashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type Aggregate struct {
	AnalysisBatchID      string                    `json:"analysis_batch_id"`
	AnalysisAsOf         string                    `json:"analysis_as_of"`
	DiscoveryWindowStart string                    `json:"discovery_window_start"`
	DiscoveryWindowEnd   string                    `json:"discovery_window_end"`
	Theme                researchthemeimport.Theme `json:"theme"`
	ReasoningTrees       []ReasoningTree           `json:"reasoning_trees"`
}

type ReasoningTree struct {
	researchreasoningtreeimport.ReasoningTree
	Nodes []Node `json:"nodes"`
}

type Node struct {
	Position                         int              `json:"position"`
	ChainNodeEntityID                string           `json:"chain_node_entity_id"`
	StateSummary                     *string          `json:"state_summary"`
	ImpactDirection                  string           `json:"impact_direction"`
	ImpactStrength                   string           `json:"impact_strength"`
	ImpactSummary                    *string          `json:"impact_summary"`
	ReasoningBasisSummary            *string          `json:"reasoning_basis_summary"`
	EvidenceGapSummary               *string          `json:"evidence_gap_summary"`
	IncomingIndustryChainGraphEdgeID *string          `json:"incoming_industry_chain_graph_edge_id"`
	IncomingTransmissionTitle        *string          `json:"incoming_transmission_title"`
	IncomingTransmissionMechanism    *string          `json:"incoming_transmission_mechanism"`
	IncomingConditionSummary         *string          `json:"incoming_condition_summary"`
	IncomingLineage                  *IncomingLineage `json:"incoming_lineage"`
	Signals                          []Signal         `json:"signals"`
}

type Signal struct {
	VariableSignalKey string        `json:"variable_signal_key"`
	SignalRole        string        `json:"signal_role"`
	SignalDirection   string        `json:"signal_direction"`
	DisplaySummary    string        `json:"display_summary"`
	DisplayOrder      int           `json:"display_order"`
	Lineage           SignalLineage `json:"lineage"`
}

type SignalLineage struct {
	SourceKind                      string  `json:"source_kind"`
	VariableSignalID                *string `json:"variable_signal_id"`
	SemanticSubmissionID            *string `json:"semantic_submission_id"`
	EvidenceID                      *string `json:"evidence_id"`
	EvidenceHash                    *string `json:"evidence_hash"`
	UpstreamVariableSignalID        *string `json:"upstream_variable_signal_id"`
	UpstreamDirectImpactAssertionID *string `json:"upstream_direct_impact_assertion_id"`
	EntityRelationID                *string `json:"entity_relation_id"`
	IndustryChainGraphEdgeID        *string `json:"industry_chain_graph_edge_id"`
}

type IncomingLineage struct {
	SourceKind                      string  `json:"source_kind"`
	DirectImpactAssertionID         *string `json:"direct_impact_assertion_id"`
	SemanticSubmissionID            *string `json:"semantic_submission_id"`
	EvidenceID                      *string `json:"evidence_id"`
	EvidenceHash                    *string `json:"evidence_hash"`
	AffectedVariableKey             *string `json:"affected_variable_key"`
	AffectedDirection               *string `json:"affected_direction"`
	UpstreamVariableSignalID        *string `json:"upstream_variable_signal_id"`
	UpstreamDirectImpactAssertionID *string `json:"upstream_direct_impact_assertion_id"`
	EntityRelationID                *string `json:"entity_relation_id"`
}

type ValidationError struct {
	Path, Reference, Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s (%s)", e.Path, e.Message, e.Reference)
}

func invalid(path, reference, message string) *ValidationError {
	return &ValidationError{Path: path, Reference: reference, Message: message}
}

type ReferenceError struct {
	Path, Reference, Message string
}

func (e *ReferenceError) Error() string {
	return fmt.Sprintf("%s: %s (%s)", e.Path, e.Message, e.Reference)
}

func invalidReference(path, reference, message string) *ReferenceError {
	return &ReferenceError{Path: path, Reference: reference, Message: message}
}

func (a Aggregate) Validate() (time.Time, string, error) {
	window, err := (researchthemeimport.Batch{
		AnalysisBatchID: a.AnalysisBatchID,
		AnalysisAsOf:    a.AnalysisAsOf,
		WindowStart:     a.DiscoveryWindowStart,
		WindowEnd:       a.DiscoveryWindowEnd,
		Themes:          []researchthemeimport.Theme{a.Theme},
	}).Validate()
	if err != nil {
		return time.Time{}, "", err
	}
	if len(a.ReasoningTrees) == 0 {
		return time.Time{}, "", invalid("reasoning_trees", "", "must contain at least one Reason Tree")
	}
	themeID := researchthemeimport.ThemeID(a.AnalysisBatchID, a.Theme.ThemeKey)
	legacyTrees := make([]researchreasoningtreeimport.ReasoningTree, 0, len(a.ReasoningTrees))
	for treeIndex, tree := range a.ReasoningTrees {
		legacyNodes := make([]researchreasoningtreeimport.Node, 0, len(tree.Nodes))
		for nodeIndex, node := range tree.Nodes {
			legacySignals := make([]researchreasoningtreeimport.Signal, 0, len(node.Signals))
			for signalIndex, signal := range node.Signals {
				path := fmt.Sprintf("reasoning_trees[%d].nodes[%d].signals[%d].lineage", treeIndex, nodeIndex, signalIndex)
				if err := signal.Lineage.validate(path); err != nil {
					return time.Time{}, "", err
				}
				legacySignals = append(legacySignals, researchreasoningtreeimport.Signal{
					VariableSignalKey: signal.VariableSignalKey,
					SignalRole:        signal.SignalRole, SignalDirection: signal.SignalDirection,
					DisplaySummary: signal.DisplaySummary, DisplayOrder: signal.DisplayOrder,
				})
			}
			if nodeIndex == 0 && node.IncomingLineage != nil {
				return time.Time{}, "", invalid(
					fmt.Sprintf("reasoning_trees[%d].nodes[%d].incoming_lineage", treeIndex, nodeIndex),
					"", "must be null for the first Node",
				)
			}
			if nodeIndex > 0 {
				if node.IncomingLineage == nil {
					return time.Time{}, "", invalid(
						fmt.Sprintf("reasoning_trees[%d].nodes[%d].incoming_lineage", treeIndex, nodeIndex),
						"", "is required for every non-root Node",
					)
				}
				if err := node.IncomingLineage.validate(
					fmt.Sprintf("reasoning_trees[%d].nodes[%d].incoming_lineage", treeIndex, nodeIndex),
					node.IncomingIndustryChainGraphEdgeID,
				); err != nil {
					return time.Time{}, "", err
				}
			}
			legacyNodes = append(legacyNodes, researchreasoningtreeimport.Node{
				Position: node.Position, ChainNodeEntityID: node.ChainNodeEntityID,
				StateSummary: node.StateSummary, ImpactDirection: node.ImpactDirection,
				ImpactStrength: node.ImpactStrength, ImpactSummary: node.ImpactSummary,
				ReasoningBasisSummary: node.ReasoningBasisSummary, EvidenceGapSummary: node.EvidenceGapSummary,
				IncomingIndustryChainGraphEdgeID: node.IncomingIndustryChainGraphEdgeID,
				IncomingTransmissionTitle:        node.IncomingTransmissionTitle,
				IncomingTransmissionMechanism:    node.IncomingTransmissionMechanism,
				IncomingConditionSummary:         node.IncomingConditionSummary, Signals: legacySignals,
			})
		}
		legacyTree := tree.ReasoningTree
		legacyTree.Nodes = legacyNodes
		legacyTrees = append(legacyTrees, legacyTree)
	}
	if err := (researchreasoningtreeimport.Publication{
		ThemeID: themeID, ReasoningTrees: legacyTrees,
	}).Validate(); err != nil {
		return time.Time{}, "", err
	}
	themeEvents := make(map[string]struct{}, len(a.Theme.Events))
	themeImpacts := make(map[string]struct{}, len(a.Theme.Impacts))
	coveredImpacts := make(map[string]struct{}, len(a.Theme.Impacts))
	for _, impact := range a.Theme.Impacts {
		themeImpacts[impact.ChainNodeEntityID] = struct{}{}
	}
	for _, event := range a.Theme.Events {
		themeEvents[event.EventID] = struct{}{}
	}
	for treeIndex, tree := range a.ReasoningTrees {
		treeImpactCount := 0
		for _, node := range tree.Nodes {
			if _, ok := themeImpacts[node.ChainNodeEntityID]; ok {
				coveredImpacts[node.ChainNodeEntityID] = struct{}{}
				treeImpactCount++
			}
		}
		if treeImpactCount == 0 {
			return time.Time{}, "", invalid(
				fmt.Sprintf("reasoning_trees[%d].nodes", treeIndex), "",
				"must contain at least one Theme Impact Node",
			)
		}
		for eventIndex, event := range tree.Events {
			if _, ok := themeEvents[event.EventID]; !ok {
				return time.Time{}, "", invalid(
					fmt.Sprintf("reasoning_trees[%d].events[%d].event_id", treeIndex, eventIndex),
					event.EventID, "must belong to Theme events",
				)
			}
		}
	}
	for impact := range themeImpacts {
		if _, ok := coveredImpacts[impact]; !ok {
			return time.Time{}, "", invalid(
				"theme.impacts", impact, "must be covered by at least one Reason Tree",
			)
		}
	}
	return window.AnalysisAsOf, themeID, nil
}

func (l SignalLineage) validate(path string) error {
	switch l.SourceKind {
	case "formal_signal":
		if !allUUID(l.VariableSignalID, l.SemanticSubmissionID, l.EvidenceID) ||
			l.EvidenceHash == nil || !hashPattern.MatchString(*l.EvidenceHash) {
			return invalid(path, l.SourceKind, "formal_signal requires Signal, Submission, Evidence UUIDs and Evidence hash")
		}
		if anySet(l.UpstreamVariableSignalID, l.UpstreamDirectImpactAssertionID, l.EntityRelationID, l.IndustryChainGraphEdgeID) {
			return invalid(path, l.SourceKind, "formal_signal cannot carry analyst inference references")
		}
	case "analyst_inference":
		if anySet(l.VariableSignalID, l.SemanticSubmissionID, l.EvidenceID, l.EvidenceHash) {
			return invalid(path, l.SourceKind, "analyst_inference cannot claim a formal Signal or Evidence")
		}
		if !oneUUID(l.UpstreamVariableSignalID, l.UpstreamDirectImpactAssertionID) ||
			!oneUUID(l.EntityRelationID, l.IndustryChainGraphEdgeID) {
			return invalid(path, l.SourceKind, "analyst_inference requires one formal upstream fact and one formal relation")
		}
	default:
		return invalid(path+".source_kind", l.SourceKind, "must be formal_signal or analyst_inference")
	}
	return nil
}

func (l IncomingLineage) validate(path string, graphEdgeID *string) error {
	switch l.SourceKind {
	case "formal_direct_impact":
		if !allUUID(l.DirectImpactAssertionID, l.SemanticSubmissionID, l.EvidenceID) ||
			l.EvidenceHash == nil || !hashPattern.MatchString(*l.EvidenceHash) ||
			l.AffectedVariableKey == nil || *l.AffectedVariableKey == "" ||
			l.AffectedDirection == nil || *l.AffectedDirection == "" {
			return invalid(path, l.SourceKind, "formal_direct_impact requires Impact, Submission, Evidence and affected-variable snapshots")
		}
		if anySet(l.UpstreamVariableSignalID, l.UpstreamDirectImpactAssertionID, l.EntityRelationID) {
			return invalid(path, l.SourceKind, "formal_direct_impact cannot carry analyst inference references")
		}
	case "analyst_inference":
		if anySet(l.DirectImpactAssertionID, l.SemanticSubmissionID, l.EvidenceID, l.EvidenceHash, l.AffectedVariableKey, l.AffectedDirection) {
			return invalid(path, l.SourceKind, "analyst_inference cannot claim a formal Direct Impact")
		}
		if !oneUUID(l.UpstreamVariableSignalID, l.UpstreamDirectImpactAssertionID) ||
			!(validUUID(l.EntityRelationID) || validUUID(graphEdgeID)) {
			return invalid(path, l.SourceKind, "analyst_inference requires one formal upstream fact and one formal incoming relation")
		}
	default:
		return invalid(path+".source_kind", l.SourceKind, "must be formal_direct_impact or analyst_inference")
	}
	return nil
}

func allUUID(values ...*string) bool {
	for _, value := range values {
		if !validUUID(value) {
			return false
		}
	}
	return true
}

func oneUUID(values ...*string) bool {
	count := 0
	for _, value := range values {
		if validUUID(value) {
			count++
		}
	}
	return count == 1
}

func validUUID(value *string) bool {
	return value != nil && regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(*value)
}

func anySet(values ...*string) bool {
	for _, value := range values {
		if value != nil {
			return true
		}
	}
	return false
}

func CanonicalHash(value Aggregate) (string, error) {
	return researchthemeimport.CanonicalHashValue(value, "research Theme Aggregate V2")
}
