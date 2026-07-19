package researchthemeimport

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

var (
	themeKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._:-]{0,127}$`)
	uuidPattern     = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
)

type Batch struct {
	AnalysisBatchID string  `json:"analysis_batch_id"`
	WindowStart     string  `json:"window_start"`
	WindowEnd       string  `json:"window_end"`
	Themes          []Theme `json:"themes"`
}

type Theme struct {
	ThemeKey                  string      `json:"theme_key"`
	Name                      string      `json:"name"`
	OneLineConclusion         string      `json:"one_line_conclusion"`
	ImpactLevel               string      `json:"impact_level"`
	TransmissionPath          string      `json:"transmission_path"`
	TradingDirection          string      `json:"trading_direction"`
	TransmissionStage         string      `json:"transmission_stage"`
	NextCheckpoint            string      `json:"next_checkpoint"`
	MarketConfirmationSummary string      `json:"market_confirmation_summary"`
	ChainNodes                []ChainNode `json:"chain_nodes"`
	Events                    []Event     `json:"events"`
}

type ChainNode struct {
	ChainNodeID   string `json:"chain_node_id"`
	RelationRole  string `json:"relation_role"`
	ImpactSummary string `json:"impact_summary"`
}

type Event struct {
	EventID        string `json:"event_id"`
	EvidenceRole   string `json:"evidence_role"`
	SupportedClaim string `json:"supported_claim"`
}

type Window struct {
	Start time.Time
	End   time.Time
}

type ValidationError struct {
	ThemeKey  string
	Path      string
	Reference string
	Message   string
}

func (e *ValidationError) Error() string {
	if e == nil {
		return "research theme publication validation failed"
	}
	location := e.Path
	if e.ThemeKey != "" {
		location = e.ThemeKey + ": " + location
	}
	if e.Reference != "" {
		return fmt.Sprintf("%s: %s (%s)", location, e.Message, e.Reference)
	}
	return fmt.Sprintf("%s: %s", location, e.Message)
}

func DecodeStrict(reader io.Reader) (Batch, error) {
	return decodeStrictJSON(reader)
}

func (b Batch) Validate() (Window, error) {
	if value := strings.TrimSpace(b.AnalysisBatchID); value == "" || len(value) > 200 {
		return Window{}, invalid("", "analysis_batch_id", "", "must contain 1..200 characters")
	}
	start, err := time.Parse(time.RFC3339, b.WindowStart)
	if err != nil {
		return Window{}, invalid("", "window_start", b.WindowStart, "must be an RFC3339 UTC timestamp")
	}
	if _, offset := start.Zone(); offset != 0 {
		return Window{}, invalid("", "window_start", b.WindowStart, "must use UTC")
	}
	end, err := time.Parse(time.RFC3339, b.WindowEnd)
	if err != nil {
		return Window{}, invalid("", "window_end", b.WindowEnd, "must be an RFC3339 UTC timestamp")
	}
	if _, offset := end.Zone(); offset != 0 {
		return Window{}, invalid("", "window_end", b.WindowEnd, "must use UTC")
	}
	if !start.Before(end) {
		return Window{}, invalid("", "window_end", b.WindowEnd, "must be greater than window_start")
	}
	if len(b.Themes) == 0 {
		return Window{}, invalid("", "themes", "", "must contain at least one Theme")
	}
	for index := range b.Themes {
		theme := &b.Themes[index]
		path := fmt.Sprintf("themes[%d]", index)
		if !themeKeyPattern.MatchString(theme.ThemeKey) {
			return Window{}, invalid(theme.ThemeKey, path+".theme_key", theme.ThemeKey, "must match ^[a-z0-9][a-z0-9._:-]{0,127}$")
		}
		if index > 0 && theme.ThemeKey <= b.Themes[index-1].ThemeKey {
			message := "must be sorted in ascending ASCII order"
			if theme.ThemeKey == b.Themes[index-1].ThemeKey {
				message = "must be unique within the batch"
			}
			return Window{}, invalid(theme.ThemeKey, path+".theme_key", theme.ThemeKey, message)
		}
		if err := theme.validate(path); err != nil {
			return Window{}, err
		}
	}
	return Window{Start: start.UTC(), End: end.UTC()}, nil
}

func (t Theme) validate(path string) error {
	required := []struct {
		name  string
		value string
	}{
		{"name", t.Name},
		{"one_line_conclusion", t.OneLineConclusion},
		{"transmission_path", t.TransmissionPath},
		{"trading_direction", t.TradingDirection},
		{"next_checkpoint", t.NextCheckpoint},
		{"market_confirmation_summary", t.MarketConfirmationSummary},
	}
	for _, field := range required {
		if strings.TrimSpace(field.value) == "" {
			return invalid(t.ThemeKey, path+"."+field.name, "", "is required")
		}
	}
	if !oneOf(t.ImpactLevel, "high", "focus", "watch") {
		return invalid(t.ThemeKey, path+".impact_level", t.ImpactLevel, "must be high, focus, or watch")
	}
	if !oneOf(t.TransmissionStage, "identification", "validation", "diffusion", "dampening") {
		return invalid(t.ThemeKey, path+".transmission_stage", t.TransmissionStage, "has an unsupported value")
	}
	if len(t.ChainNodes) == 0 {
		return invalid(t.ThemeKey, path+".chain_nodes", "", "must contain at least one association")
	}
	for index, node := range t.ChainNodes {
		nodePath := fmt.Sprintf("%s.chain_nodes[%d]", path, index)
		if !uuidPattern.MatchString(node.ChainNodeID) {
			return invalid(t.ThemeKey, nodePath+".chain_node_id", node.ChainNodeID, "must be a standard lowercase UUID")
		}
		if index > 0 && node.ChainNodeID <= t.ChainNodes[index-1].ChainNodeID {
			message := "must be sorted by chain_node_id"
			if node.ChainNodeID == t.ChainNodes[index-1].ChainNodeID {
				message = "must be unique within the Theme"
			}
			return invalid(t.ThemeKey, nodePath+".chain_node_id", node.ChainNodeID, message)
		}
		if !oneOf(node.RelationRole, "driver", "beneficiary", "constraint", "exposure") {
			return invalid(t.ThemeKey, nodePath+".relation_role", node.RelationRole, "has an unsupported value")
		}
		if strings.TrimSpace(node.ImpactSummary) == "" {
			return invalid(t.ThemeKey, nodePath+".impact_summary", "", "is required")
		}
	}
	if len(t.Events) == 0 {
		return invalid(t.ThemeKey, path+".events", "", "must contain at least one evidence association")
	}
	hasDriver := false
	for index, event := range t.Events {
		eventPath := fmt.Sprintf("%s.events[%d]", path, index)
		if !uuidPattern.MatchString(event.EventID) {
			return invalid(t.ThemeKey, eventPath+".event_id", event.EventID, "must be a standard lowercase UUID")
		}
		if index > 0 && event.EventID <= t.Events[index-1].EventID {
			message := "must be sorted by event_id"
			if event.EventID == t.Events[index-1].EventID {
				message = "must be unique within the Theme"
			}
			return invalid(t.ThemeKey, eventPath+".event_id", event.EventID, message)
		}
		if !oneOf(event.EvidenceRole, "driver", "supporting", "contradicting", "context") {
			return invalid(t.ThemeKey, eventPath+".evidence_role", event.EvidenceRole, "has an unsupported value")
		}
		if event.EvidenceRole == "driver" {
			hasDriver = true
		}
		if strings.TrimSpace(event.SupportedClaim) == "" {
			return invalid(t.ThemeKey, eventPath+".supported_claim", "", "is required")
		}
	}
	if !hasDriver {
		return invalid(t.ThemeKey, path+".events", "", "must contain at least one driver Event")
	}
	return nil
}

func invalid(themeKey, path, reference, message string) *ValidationError {
	return &ValidationError{ThemeKey: themeKey, Path: path, Reference: reference, Message: message}
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
