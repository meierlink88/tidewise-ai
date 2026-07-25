package researchanchorimport

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

type Publication struct {
	ThemeID string   `json:"theme_id"`
	Anchors []Anchor `json:"anchors"`
}

type Anchor struct {
	CenterChainNodeID   string     `json:"center_chain_node_id"`
	OneLineConclusion   string     `json:"one_line_conclusion"`
	FactSummary         string     `json:"fact_summary"`
	NetDirectionSummary string     `json:"net_direction_summary"`
	SupportSummary      string     `json:"support_summary"`
	CounterSummary      *string    `json:"counter_summary"`
	TradingDirection    string     `json:"trading_direction"`
	NextCheckpoint      string     `json:"next_checkpoint"`
	Events              []Event    `json:"events"`
	PathNodes           []PathNode `json:"path_nodes"`
}

type Event struct {
	EventID         string `json:"event_id"`
	EvidenceRole    string `json:"evidence_role"`
	EvidenceSummary string `json:"evidence_summary"`
}

type PathNode struct {
	ChainNodeID                   string  `json:"chain_node_id"`
	ChangeDirection               string  `json:"change_direction"`
	ChangeSummary                 string  `json:"change_summary"`
	ImpactSummary                 string  `json:"impact_summary"`
	IncomingTransmissionMechanism *string `json:"incoming_transmission_mechanism"`
}

type ValidationError struct {
	CenterChainNodeID string
	Path              string
	Reference         string
	Message           string
}

func (e *ValidationError) Error() string {
	if e == nil {
		return "research Anchor publication validation failed"
	}
	if e.Reference != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Path, e.Message, e.Reference)
	}
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

func DecodeStrict(reader io.Reader) (Publication, error) {
	return decodeStrictJSON(reader)
}

func (p Publication) Validate() error {
	if !uuidPattern.MatchString(p.ThemeID) {
		return invalid("", "theme_id", p.ThemeID, "must be a standard lowercase UUID")
	}
	if len(p.Anchors) == 0 {
		return invalid("", "anchors", "", "must contain at least one Anchor")
	}
	for index := range p.Anchors {
		anchor := &p.Anchors[index]
		path := fmt.Sprintf("anchors[%d]", index)
		if !uuidPattern.MatchString(anchor.CenterChainNodeID) {
			return invalid(anchor.CenterChainNodeID, path+".center_chain_node_id", anchor.CenterChainNodeID, "must be a standard lowercase UUID")
		}
		if index > 0 && anchor.CenterChainNodeID <= p.Anchors[index-1].CenterChainNodeID {
			message := "must be sorted by center_chain_node_id"
			if anchor.CenterChainNodeID == p.Anchors[index-1].CenterChainNodeID {
				message = "must be unique within the Theme"
			}
			return invalid(anchor.CenterChainNodeID, path+".center_chain_node_id", anchor.CenterChainNodeID, message)
		}
		if err := anchor.validate(path); err != nil {
			return err
		}
	}
	return nil
}

func (a Anchor) validate(path string) error {
	required := []struct {
		name  string
		value string
	}{
		{"one_line_conclusion", a.OneLineConclusion},
		{"fact_summary", a.FactSummary},
		{"net_direction_summary", a.NetDirectionSummary},
		{"support_summary", a.SupportSummary},
		{"trading_direction", a.TradingDirection},
		{"next_checkpoint", a.NextCheckpoint},
	}
	for _, field := range required {
		if strings.TrimSpace(field.value) == "" {
			return invalid(a.CenterChainNodeID, path+"."+field.name, "", "is required")
		}
	}
	if len(a.Events) == 0 {
		return invalid(a.CenterChainNodeID, path+".events", "", "must contain at least one evidence association")
	}
	hasDriver := false
	hasContradiction := false
	for index, event := range a.Events {
		eventPath := fmt.Sprintf("%s.events[%d]", path, index)
		if !uuidPattern.MatchString(event.EventID) {
			return invalid(a.CenterChainNodeID, eventPath+".event_id", event.EventID, "must be a standard lowercase UUID")
		}
		if index > 0 && event.EventID <= a.Events[index-1].EventID {
			message := "must be sorted by event_id"
			if event.EventID == a.Events[index-1].EventID {
				message = "must be unique within the Anchor"
			}
			return invalid(a.CenterChainNodeID, eventPath+".event_id", event.EventID, message)
		}
		if !oneOf(event.EvidenceRole, "driver", "supporting", "contradicting", "context") {
			return invalid(a.CenterChainNodeID, eventPath+".evidence_role", event.EvidenceRole, "has an unsupported value")
		}
		if event.EvidenceRole == "driver" {
			hasDriver = true
		}
		if event.EvidenceRole == "contradicting" {
			hasContradiction = true
		}
		if strings.TrimSpace(event.EvidenceSummary) == "" {
			return invalid(a.CenterChainNodeID, eventPath+".evidence_summary", "", "is required")
		}
	}
	if !hasDriver {
		return invalid(a.CenterChainNodeID, path+".events", "", "must contain at least one driver Event")
	}
	if hasContradiction && (a.CounterSummary == nil || strings.TrimSpace(*a.CounterSummary) == "") {
		return invalid(a.CenterChainNodeID, path+".counter_summary", "", "is required when the Anchor contains a contradicting Event")
	}
	if !hasContradiction && a.CounterSummary != nil {
		return invalid(a.CenterChainNodeID, path+".counter_summary", "", "must be null when the Anchor has no contradicting Event")
	}
	if len(a.PathNodes) < 2 {
		return invalid(a.CenterChainNodeID, path+".path_nodes", "", "must contain at least two Path Nodes")
	}
	seen := make(map[string]struct{}, len(a.PathNodes))
	centerCount := 0
	for index, node := range a.PathNodes {
		nodePath := fmt.Sprintf("%s.path_nodes[%d]", path, index)
		if !uuidPattern.MatchString(node.ChainNodeID) {
			return invalid(a.CenterChainNodeID, nodePath+".chain_node_id", node.ChainNodeID, "must be a standard lowercase UUID")
		}
		if _, duplicate := seen[node.ChainNodeID]; duplicate {
			return invalid(a.CenterChainNodeID, nodePath+".chain_node_id", node.ChainNodeID, "must not repeat within the ordered path")
		}
		seen[node.ChainNodeID] = struct{}{}
		if node.ChainNodeID == a.CenterChainNodeID {
			centerCount++
		}
		if !oneOf(node.ChangeDirection, "increase", "decrease", "mixed", "unchanged", "uncertain") {
			return invalid(a.CenterChainNodeID, nodePath+".change_direction", node.ChangeDirection, "has an unsupported value")
		}
		if strings.TrimSpace(node.ChangeSummary) == "" {
			return invalid(a.CenterChainNodeID, nodePath+".change_summary", "", "is required")
		}
		if strings.TrimSpace(node.ImpactSummary) == "" {
			return invalid(a.CenterChainNodeID, nodePath+".impact_summary", "", "is required")
		}
		if index == 0 && node.IncomingTransmissionMechanism != nil {
			return invalid(a.CenterChainNodeID, nodePath+".incoming_transmission_mechanism", "", "must be null for the first Path Node")
		}
		if index > 0 && (node.IncomingTransmissionMechanism == nil || strings.TrimSpace(*node.IncomingTransmissionMechanism) == "") {
			return invalid(a.CenterChainNodeID, nodePath+".incoming_transmission_mechanism", "", "must be nonblank after the first Path Node")
		}
	}
	if centerCount != 1 {
		return invalid(a.CenterChainNodeID, path+".path_nodes", a.CenterChainNodeID, "must contain the center Chain Node exactly once")
	}
	return nil
}

func invalid(centerChainNodeID, path, reference, message string) *ValidationError {
	return &ValidationError{CenterChainNodeID: centerChainNodeID, Path: path, Reference: reference, Message: message}
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
