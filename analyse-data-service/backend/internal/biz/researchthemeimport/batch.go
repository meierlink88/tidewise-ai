package researchthemeimport

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	themeKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._:-]{0,127}$`)
	uuidPattern     = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
)

type Batch struct {
	AnalysisBatchID string  `json:"analysis_batch_id"`
	AnalysisAsOf    string  `json:"analysis_as_of"`
	WindowStart     string  `json:"window_start"`
	WindowEnd       string  `json:"window_end"`
	Themes          []Theme `json:"themes"`
}

type Theme struct {
	ThemeKey                  string   `json:"theme_key"`
	Title                     string   `json:"title"`
	OneLineConclusion         string   `json:"one_line_conclusion"`
	ConclusionDirection       string   `json:"conclusion_direction"`
	ImpactStrength            string   `json:"impact_strength"`
	AttentionLevel            *string  `json:"attention_level"`
	ConclusionStatus          *string  `json:"conclusion_status"`
	TransmissionStage         string   `json:"transmission_stage"`
	InvestmentGuidanceAction  string   `json:"investment_guidance_action"`
	InvestmentGuidanceSummary string   `json:"investment_guidance_summary"`
	TimeHorizonCategory       string   `json:"time_horizon_category"`
	TimeHorizonSummary        *string  `json:"time_horizon_summary"`
	TransmissionSummary       *string  `json:"transmission_summary"`
	CheckpointSummary         *string  `json:"checkpoint_summary"`
	RiskSummary               *string  `json:"risk_summary"`
	Impacts                   []Impact `json:"impacts"`
	Events                    []Event  `json:"events"`
}

type Impact struct {
	ChainNodeEntityID           string  `json:"chain_node_entity_id"`
	RelationRole                string  `json:"relation_role"`
	ImpactDirection             string  `json:"impact_direction"`
	ImpactSummary               *string `json:"impact_summary"`
	PrimarySignalDisplaySummary string  `json:"primary_signal_display_summary"`
	DisplayOrder                int     `json:"display_order"`
}

type Event struct {
	EventID        string  `json:"event_id"`
	EvidenceRole   string  `json:"evidence_role"`
	SupportedClaim *string `json:"supported_claim"`
}

type Window struct {
	AnalysisAsOf time.Time
	Start        time.Time
	End          time.Time
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
	if value := strings.TrimSpace(b.AnalysisBatchID); value == "" || utf8.RuneCountInString(value) > 200 {
		return Window{}, invalid("", "analysis_batch_id", "", "must contain 1..200 characters")
	}
	analysisAsOf, err := parseUTCTime("analysis_as_of", b.AnalysisAsOf)
	if err != nil {
		return Window{}, err
	}
	start, err := parseUTCTime("window_start", b.WindowStart)
	if err != nil {
		return Window{}, err
	}
	end, err := parseUTCTime("window_end", b.WindowEnd)
	if err != nil {
		return Window{}, err
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
	return Window{AnalysisAsOf: analysisAsOf, Start: start, End: end}, nil
}

func (t Theme) validate(path string) error {
	required := []struct {
		name  string
		value string
		max   int
	}{
		{"title", t.Title, 300},
		{"one_line_conclusion", t.OneLineConclusion, 1000},
		{"investment_guidance_summary", t.InvestmentGuidanceSummary, 2000},
	}
	for _, field := range required {
		if err := validateRequiredText(t.ThemeKey, path+"."+field.name, field.value, field.max); err != nil {
			return err
		}
	}
	if !oneOf(t.ConclusionDirection, "positive", "negative", "mixed", "neutral", "uncertain") {
		return invalid(t.ThemeKey, path+".conclusion_direction", t.ConclusionDirection, "has an unsupported value")
	}
	if !oneOf(t.ImpactStrength, "strong", "medium", "weak", "unknown") {
		return invalid(t.ThemeKey, path+".impact_strength", t.ImpactStrength, "has an unsupported value")
	}
	if t.AttentionLevel != nil && !oneOf(*t.AttentionLevel, "high", "medium", "low") {
		return invalid(t.ThemeKey, path+".attention_level", *t.AttentionLevel, "has an unsupported value")
	}
	if t.ConclusionStatus != nil && !oneOf(*t.ConclusionStatus, "supported", "partial", "conflicted") {
		return invalid(t.ThemeKey, path+".conclusion_status", *t.ConclusionStatus, "has an unsupported value")
	}
	if !oneOf(t.TransmissionStage, "identification", "validation", "diffusion", "dampening") {
		return invalid(t.ThemeKey, path+".transmission_stage", t.TransmissionStage, "has an unsupported value")
	}
	if !oneOf(t.InvestmentGuidanceAction, "focus", "avoid", "observe", "differentiate") {
		return invalid(t.ThemeKey, path+".investment_guidance_action", t.InvestmentGuidanceAction, "has an unsupported value")
	}
	if !oneOf(t.TimeHorizonCategory, "short_term", "medium_term", "long_term", "custom") {
		return invalid(t.ThemeKey, path+".time_horizon_category", t.TimeHorizonCategory, "has an unsupported value")
	}
	for _, field := range []struct {
		name  string
		value *string
		max   int
	}{
		{"time_horizon_summary", t.TimeHorizonSummary, 1000},
		{"transmission_summary", t.TransmissionSummary, 4000},
		{"checkpoint_summary", t.CheckpointSummary, 4000},
		{"risk_summary", t.RiskSummary, 4000},
	} {
		if err := validateOptionalText(t.ThemeKey, path+"."+field.name, field.value, field.max); err != nil {
			return err
		}
	}
	if len(t.Impacts) == 0 {
		return invalid(t.ThemeKey, path+".impacts", "", "must contain at least one Theme Impact")
	}
	seenImpacts := make(map[string]struct{}, len(t.Impacts))
	for index, impact := range t.Impacts {
		impactPath := fmt.Sprintf("%s.impacts[%d]", path, index)
		if impact.DisplayOrder != index+1 {
			return invalid(t.ThemeKey, impactPath+".display_order", fmt.Sprint(impact.DisplayOrder), "must be contiguous from 1")
		}
		if !uuidPattern.MatchString(impact.ChainNodeEntityID) {
			return invalid(t.ThemeKey, impactPath+".chain_node_entity_id", impact.ChainNodeEntityID, "must be a standard lowercase UUID")
		}
		if _, duplicate := seenImpacts[impact.ChainNodeEntityID]; duplicate {
			return invalid(t.ThemeKey, impactPath+".chain_node_entity_id", impact.ChainNodeEntityID, "must be unique within the Theme")
		}
		seenImpacts[impact.ChainNodeEntityID] = struct{}{}
		if !oneOf(impact.RelationRole, "driver", "beneficiary", "constraint", "exposure") {
			return invalid(t.ThemeKey, impactPath+".relation_role", impact.RelationRole, "has an unsupported value")
		}
		if !oneOf(impact.ImpactDirection, "positive", "negative", "mixed", "neutral", "uncertain") {
			return invalid(t.ThemeKey, impactPath+".impact_direction", impact.ImpactDirection, "has an unsupported value")
		}
		if err := validateOptionalText(t.ThemeKey, impactPath+".impact_summary", impact.ImpactSummary, 2000); err != nil {
			return err
		}
		if err := validateRequiredTrimmedText(
			t.ThemeKey,
			impactPath+".primary_signal_display_summary",
			impact.PrimarySignalDisplaySummary,
			200,
		); err != nil {
			return err
		}
	}
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
		if err := validateOptionalText(t.ThemeKey, eventPath+".supported_claim", event.SupportedClaim, 2000); err != nil {
			return err
		}
	}
	return nil
}

func parseUTCTime(path, value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, invalid("", path, value, "must be an RFC3339 UTC timestamp")
	}
	_, offset := parsed.Zone()
	if offset != 0 {
		return time.Time{}, invalid("", path, value, "must use UTC")
	}
	return parsed.UTC(), nil
}

func validateRequiredText(themeKey, path, value string, max int) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return invalid(themeKey, path, "", "is required")
	}
	if utf8.RuneCountInString(trimmed) > max {
		return invalid(themeKey, path, "", fmt.Sprintf("must contain at most %d characters", max))
	}
	return nil
}

func validateRequiredTrimmedText(themeKey, path, value string, max int) error {
	if err := validateRequiredText(themeKey, path, value, max); err != nil {
		return err
	}
	if value != strings.TrimSpace(value) {
		return invalid(themeKey, path, "", "must not contain leading or trailing whitespace")
	}
	return nil
}

func validateOptionalText(themeKey, path string, value *string, max int) error {
	if value != nil && utf8.RuneCountInString(strings.TrimSpace(*value)) > max {
		return invalid(themeKey, path, "", fmt.Sprintf("must contain at most %d characters", max))
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
