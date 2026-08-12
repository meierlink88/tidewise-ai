package research

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	entitybiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/entity"
	eventbiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/event"
	eventsemanticbiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventsemantic"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/identity"
)

func canonicalPublicationHashValue(value any, label string) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode %s: %w", label, err)
	}

	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return "", fmt.Errorf("decode %s: %w", label, err)
	}

	var canonical bytes.Buffer
	if err := writePublicationCanonicalJSON(&canonical, decoded); err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical.Bytes())
	return hex.EncodeToString(sum[:]), nil
}

func writePublicationCanonicalJSON(writer *bytes.Buffer, value any) error {
	switch typed := value.(type) {
	case nil:
		writer.WriteString("null")
	case string:
		return writePublicationCanonicalString(writer, typed)
	case json.Number:
		writer.WriteString(typed.String())
	case []any:
		writer.WriteByte('[')
		for index, item := range typed {
			if index > 0 {
				writer.WriteByte(',')
			}
			if err := writePublicationCanonicalJSON(writer, item); err != nil {
				return err
			}
		}
		writer.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		writer.WriteByte('{')
		for index, key := range keys {
			if index > 0 {
				writer.WriteByte(',')
			}
			if err := writePublicationCanonicalString(writer, key); err != nil {
				return err
			}
			writer.WriteByte(':')
			if err := writePublicationCanonicalJSON(writer, typed[key]); err != nil {
				return err
			}
		}
		writer.WriteByte('}')
	default:
		return fmt.Errorf("unsupported V1 canonical JSON value %T", value)
	}
	return nil
}

func writePublicationCanonicalString(writer *bytes.Buffer, value string) error {
	if !utf8.ValidString(value) {
		return fmt.Errorf("canonical JSON string contains invalid UTF-8")
	}
	const hexadecimal = "0123456789abcdef"
	writer.WriteByte('"')
	for index := 0; index < len(value); index++ {
		character := value[index]
		switch character {
		case '"', '\\':
			writer.WriteByte('\\')
			writer.WriteByte(character)
		case '\b':
			writer.WriteString(`\b`)
		case '\t':
			writer.WriteString(`\t`)
		case '\n':
			writer.WriteString(`\n`)
		case '\f':
			writer.WriteString(`\f`)
		case '\r':
			writer.WriteString(`\r`)
		default:
			if character < 0x20 {
				writer.WriteString(`\u00`)
				writer.WriteByte(hexadecimal[character>>4])
				writer.WriteByte(hexadecimal[character&0x0f])
				continue
			}
			writer.WriteByte(character)
		}
	}
	writer.WriteByte('"')
	return nil
}

var (
	publicationReasonTreeNamespace = [16]byte{
		0x33, 0xf3, 0xa1, 0x72, 0x5c, 0x45, 0x55, 0x85,
		0x8b, 0xcd, 0x19, 0x52, 0x7b, 0x34, 0x61, 0x93,
	}
	publicationReasonTreeNodeNamespace = [16]byte{
		0x7e, 0x8c, 0xb1, 0x31, 0x70, 0x3b, 0x5c, 0xaf,
		0x98, 0xaa, 0x1f, 0x4d, 0x8a, 0xa6, 0xcb, 0x1e,
	}
)

func publicationReasonTreeID(themeID, industryChainEntityID string) string {
	return publicationUUIDV5(publicationReasonTreeNamespace, themeID+"\x00"+industryChainEntityID)
}

func publicationReasonTreeNodeID(reasoningTreeID string, position int, chainNodeEntityID string) string {
	return publicationUUIDV5(publicationReasonTreeNodeNamespace, reasoningTreeID+"\x00"+strconv.Itoa(position)+"\x00"+chainNodeEntityID)
}

func publicationUUIDV5(namespace [16]byte, name string) string {
	hash := sha1.New()
	_, _ = hash.Write(namespace[:])
	_, _ = hash.Write([]byte(name))
	identifier := hash.Sum(nil)[:16]
	identifier[6] = (identifier[6] & 0x0f) | 0x50
	identifier[8] = (identifier[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		identifier[0:4], identifier[4:6], identifier[6:8], identifier[8:10], identifier[10:16])
}

var publicationThemeNamespace = [16]byte{
	0x7b, 0x95, 0x0d, 0x74, 0x76, 0x8c, 0x57, 0xe0,
	0x97, 0xb5, 0xea, 0x4f, 0x3d, 0xa1, 0xbc, 0x88,
}

func publicationThemeID(analysisBatchID, themeKey string) string {
	return publicationUUIDV5(publicationThemeNamespace, analysisBatchID+"\x00"+themeKey)
}

var (
	researchKeyPattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9._:-]{0,127}$`)
	lowercaseUUIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
)

type ThemeBatch struct {
	AnalysisBatchID string       `json:"analysis_batch_id"`
	AnalysisAsOf    string       `json:"analysis_as_of"`
	WindowStart     string       `json:"window_start"`
	WindowEnd       string       `json:"window_end"`
	Themes          []ThemeInput `json:"themes"`
}

type ThemeInput struct {
	ThemeKey                  string             `json:"theme_key"`
	Title                     string             `json:"title"`
	OneLineConclusion         string             `json:"one_line_conclusion"`
	ConclusionDirection       string             `json:"conclusion_direction"`
	ImpactStrength            string             `json:"impact_strength"`
	AttentionLevel            *string            `json:"attention_level"`
	ConclusionStatus          *string            `json:"conclusion_status"`
	TransmissionStage         string             `json:"transmission_stage"`
	InvestmentGuidanceAction  string             `json:"investment_guidance_action"`
	InvestmentGuidanceSummary string             `json:"investment_guidance_summary"`
	TimeHorizonCategory       string             `json:"time_horizon_category"`
	TimeHorizonSummary        *string            `json:"time_horizon_summary"`
	TransmissionSummary       *string            `json:"transmission_summary"`
	CheckpointSummary         *string            `json:"checkpoint_summary"`
	RiskSummary               *string            `json:"risk_summary"`
	Impacts                   []ThemeImpactInput `json:"impacts"`
	Events                    []ThemeEventInput  `json:"events"`
}

type ThemeImpactInput struct {
	ChainNodeEntityID string  `json:"chain_node_entity_id"`
	RelationRole      string  `json:"relation_role"`
	ImpactDirection   string  `json:"impact_direction"`
	ImpactSummary     *string `json:"impact_summary"`
	DisplayOrder      int     `json:"display_order"`
}

type ThemeEventInput struct {
	EventID        string  `json:"event_id"`
	EvidenceRole   string  `json:"evidence_role"`
	SupportedClaim *string `json:"supported_claim"`
}

type ThemeWindow struct {
	AnalysisAsOf time.Time
	Start        time.Time
	End          time.Time
}

type ThemeValidationError struct {
	ThemeKey  string
	Path      string
	Reference string
	Message   string
}

func (e *ThemeValidationError) Error() string {
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

func (b ThemeBatch) Validate() (ThemeWindow, error) {
	if value := strings.TrimSpace(b.AnalysisBatchID); value == "" || utf8.RuneCountInString(value) > 200 {
		return ThemeWindow{}, invalidTheme("", "analysis_batch_id", "", "must contain 1..200 characters")
	}
	analysisAsOf, err := parseThemeUTCTime("analysis_as_of", b.AnalysisAsOf)
	if err != nil {
		return ThemeWindow{}, err
	}
	start, err := parseThemeUTCTime("window_start", b.WindowStart)
	if err != nil {
		return ThemeWindow{}, err
	}
	end, err := parseThemeUTCTime("window_end", b.WindowEnd)
	if err != nil {
		return ThemeWindow{}, err
	}
	if !start.Before(end) {
		return ThemeWindow{}, invalidTheme("", "window_end", b.WindowEnd, "must be greater than window_start")
	}
	if len(b.Themes) == 0 {
		return ThemeWindow{}, invalidTheme("", "themes", "", "must contain at least one ThemeInput")
	}
	for index := range b.Themes {
		theme := &b.Themes[index]
		path := fmt.Sprintf("themes[%d]", index)
		if !researchKeyPattern.MatchString(theme.ThemeKey) {
			return ThemeWindow{}, invalidTheme(theme.ThemeKey, path+".theme_key", theme.ThemeKey, "must match ^[a-z0-9][a-z0-9._:-]{0,127}$")
		}
		if index > 0 && theme.ThemeKey <= b.Themes[index-1].ThemeKey {
			message := "must be sorted in ascending ASCII order"
			if theme.ThemeKey == b.Themes[index-1].ThemeKey {
				message = "must be unique within the batch"
			}
			return ThemeWindow{}, invalidTheme(theme.ThemeKey, path+".theme_key", theme.ThemeKey, message)
		}
		if err := theme.validate(path); err != nil {
			return ThemeWindow{}, err
		}
	}
	return ThemeWindow{AnalysisAsOf: analysisAsOf, Start: start, End: end}, nil
}

func (t ThemeInput) validate(path string) error {
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
		if err := validateThemeRequiredText(t.ThemeKey, path+"."+field.name, field.value, field.max); err != nil {
			return err
		}
	}
	if !isAllowedValue(t.ConclusionDirection, "positive", "negative", "mixed", "neutral", "uncertain") {
		return invalidTheme(t.ThemeKey, path+".conclusion_direction", t.ConclusionDirection, "has an unsupported value")
	}
	if !isAllowedValue(t.ImpactStrength, "strong", "medium", "weak", "unknown") {
		return invalidTheme(t.ThemeKey, path+".impact_strength", t.ImpactStrength, "has an unsupported value")
	}
	if t.AttentionLevel != nil && !isAllowedValue(*t.AttentionLevel, "high", "medium", "low") {
		return invalidTheme(t.ThemeKey, path+".attention_level", *t.AttentionLevel, "has an unsupported value")
	}
	if t.ConclusionStatus != nil && !isAllowedValue(*t.ConclusionStatus, "supported", "partial", "conflicted") {
		return invalidTheme(t.ThemeKey, path+".conclusion_status", *t.ConclusionStatus, "has an unsupported value")
	}
	if !isAllowedValue(t.TransmissionStage, "identification", "validation", "diffusion", "dampening") {
		return invalidTheme(t.ThemeKey, path+".transmission_stage", t.TransmissionStage, "has an unsupported value")
	}
	if !isAllowedValue(t.InvestmentGuidanceAction, "focus", "avoid", "observe", "differentiate") {
		return invalidTheme(t.ThemeKey, path+".investment_guidance_action", t.InvestmentGuidanceAction, "has an unsupported value")
	}
	if !isAllowedValue(t.TimeHorizonCategory, "short_term", "medium_term", "long_term", "custom") {
		return invalidTheme(t.ThemeKey, path+".time_horizon_category", t.TimeHorizonCategory, "has an unsupported value")
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
		if err := validateThemeOptionalText(t.ThemeKey, path+"."+field.name, field.value, field.max); err != nil {
			return err
		}
	}
	if len(t.Impacts) == 0 {
		return invalidTheme(t.ThemeKey, path+".impacts", "", "must contain at least one ThemeInput ThemeImpactInput")
	}
	seenImpacts := make(map[string]struct{}, len(t.Impacts))
	for index, impact := range t.Impacts {
		impactPath := fmt.Sprintf("%s.impacts[%d]", path, index)
		if impact.DisplayOrder != index+1 {
			return invalidTheme(t.ThemeKey, impactPath+".display_order", fmt.Sprint(impact.DisplayOrder), "must be contiguous from 1")
		}
		if !lowercaseUUIDPattern.MatchString(impact.ChainNodeEntityID) {
			return invalidTheme(t.ThemeKey, impactPath+".chain_node_entity_id", impact.ChainNodeEntityID, "must be a standard lowercase UUID")
		}
		if _, duplicate := seenImpacts[impact.ChainNodeEntityID]; duplicate {
			return invalidTheme(t.ThemeKey, impactPath+".chain_node_entity_id", impact.ChainNodeEntityID, "must be unique within the ThemeInput")
		}
		seenImpacts[impact.ChainNodeEntityID] = struct{}{}
		if !isAllowedValue(impact.RelationRole, "driver", "beneficiary", "constraint", "exposure") {
			return invalidTheme(t.ThemeKey, impactPath+".relation_role", impact.RelationRole, "has an unsupported value")
		}
		if !isAllowedValue(impact.ImpactDirection, "positive", "negative", "mixed", "neutral", "uncertain") {
			return invalidTheme(t.ThemeKey, impactPath+".impact_direction", impact.ImpactDirection, "has an unsupported value")
		}
		if err := validateThemeOptionalText(t.ThemeKey, impactPath+".impact_summary", impact.ImpactSummary, 2000); err != nil {
			return err
		}
	}
	for index, event := range t.Events {
		eventPath := fmt.Sprintf("%s.events[%d]", path, index)
		if !lowercaseUUIDPattern.MatchString(event.EventID) {
			return invalidTheme(t.ThemeKey, eventPath+".event_id", event.EventID, "must be a standard lowercase UUID")
		}
		if index > 0 && event.EventID <= t.Events[index-1].EventID {
			message := "must be sorted by event_id"
			if event.EventID == t.Events[index-1].EventID {
				message = "must be unique within the ThemeInput"
			}
			return invalidTheme(t.ThemeKey, eventPath+".event_id", event.EventID, message)
		}
		if !isAllowedValue(event.EvidenceRole, "driver", "supporting", "contradicting", "context") {
			return invalidTheme(t.ThemeKey, eventPath+".evidence_role", event.EvidenceRole, "has an unsupported value")
		}
		if err := validateThemeOptionalText(t.ThemeKey, eventPath+".supported_claim", event.SupportedClaim, 2000); err != nil {
			return err
		}
	}
	return nil
}

func parseThemeUTCTime(path, value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, invalidTheme("", path, value, "must be an RFC3339 UTC timestamp")
	}
	_, offset := parsed.Zone()
	if offset != 0 {
		return time.Time{}, invalidTheme("", path, value, "must use UTC")
	}
	return parsed.UTC(), nil
}

func validateThemeRequiredText(themeKey, path, value string, max int) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return invalidTheme(themeKey, path, "", "is required")
	}
	if utf8.RuneCountInString(trimmed) > max {
		return invalidTheme(themeKey, path, "", fmt.Sprintf("must contain at most %d characters", max))
	}
	return nil
}

func validateThemeOptionalText(themeKey, path string, value *string, max int) error {
	if value != nil && utf8.RuneCountInString(strings.TrimSpace(*value)) > max {
		return invalidTheme(themeKey, path, "", fmt.Sprintf("must contain at most %d characters", max))
	}
	return nil
}

func invalidTheme(themeKey, path, reference, message string) *ThemeValidationError {
	return &ThemeValidationError{ThemeKey: themeKey, Path: path, Reference: reference, Message: message}
}

func isAllowedValue(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

type ReasonTreePublication struct {
	ThemeID        string            `json:"theme_id"`
	ReasoningTrees []ReasonTreeInput `json:"reasoning_trees"`
}

type ReasonTreeInput struct {
	IndustryChainEntityID     string                 `json:"industry_chain_entity_id"`
	Title                     string                 `json:"title"`
	DisplayOrder              int                    `json:"display_order"`
	OneLineConclusion         string                 `json:"one_line_conclusion"`
	FactSummary               *string                `json:"fact_summary"`
	TransmissionSummary       *string                `json:"transmission_summary"`
	ImpactDirection           string                 `json:"impact_direction"`
	ImpactStrength            string                 `json:"impact_strength"`
	ImpactSummary             *string                `json:"impact_summary"`
	ConclusionBoundarySummary *string                `json:"conclusion_boundary_summary"`
	SupportSummary            *string                `json:"support_summary"`
	CounterSummary            *string                `json:"counter_summary"`
	InvalidationConditions    []string               `json:"invalidation_conditions"`
	Checkpoints               []ReasonTreeCheckpoint `json:"checkpoints"`
	Events                    []ReasonTreeEventInput `json:"events"`
	Nodes                     []ReasonTreeNodeInput  `json:"nodes"`
}

type ReasonTreeCheckpoint struct {
	Type    string `json:"type"`
	Summary string `json:"summary"`
}

type ReasonTreeEventInput struct {
	EventID      string `json:"event_id"`
	EvidenceRole string `json:"evidence_role"`
	DisplayOrder int    `json:"display_order"`
}

type ReasonTreeNodeInput struct {
	Position                         int                     `json:"position"`
	ChainNodeEntityID                string                  `json:"chain_node_entity_id"`
	StateSummary                     *string                 `json:"state_summary"`
	ImpactDirection                  string                  `json:"impact_direction"`
	ImpactStrength                   string                  `json:"impact_strength"`
	ImpactSummary                    *string                 `json:"impact_summary"`
	ReasoningBasisSummary            *string                 `json:"reasoning_basis_summary"`
	EvidenceGapSummary               *string                 `json:"evidence_gap_summary"`
	IncomingIndustryChainGraphEdgeID *string                 `json:"incoming_industry_chain_graph_edge_id"`
	IncomingTransmissionTitle        *string                 `json:"incoming_transmission_title"`
	IncomingTransmissionMechanism    *string                 `json:"incoming_transmission_mechanism"`
	IncomingConditionSummary         *string                 `json:"incoming_condition_summary"`
	Signals                          []ReasonTreeSignalInput `json:"signals"`
}

type ReasonTreeSignalInput struct {
	VariableSignalKey string `json:"variable_signal_key"`
	SignalRole        string `json:"signal_role"`
	SignalDirection   string `json:"signal_direction"`
	DisplaySummary    string `json:"display_summary"`
	DisplayOrder      int    `json:"display_order"`
}

type ReasonTreeValidationError struct {
	IndustryChainEntityID string
	Path                  string
	Reference             string
	Message               string
}

func (e *ReasonTreeValidationError) Error() string {
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

func (p ReasonTreePublication) Validate() error {
	if !lowercaseUUIDPattern.MatchString(p.ThemeID) {
		return invalidReasonTree("", "theme_id", p.ThemeID, "must be a standard lowercase UUID")
	}
	if len(p.ReasoningTrees) == 0 {
		return invalidReasonTree("", "reasoning_trees", "", "must contain at least one Reason Tree")
	}
	seenChains := make(map[string]struct{}, len(p.ReasoningTrees))
	signalSnapshots := make(map[string]ReasonTreeSignalInput)
	for index, tree := range p.ReasoningTrees {
		path := fmt.Sprintf("reasoning_trees[%d]", index)
		if tree.DisplayOrder != index+1 {
			return invalidReasonTree(tree.IndustryChainEntityID, path+".display_order", fmt.Sprint(tree.DisplayOrder), "must be contiguous from 1")
		}
		if !lowercaseUUIDPattern.MatchString(tree.IndustryChainEntityID) {
			return invalidReasonTree(tree.IndustryChainEntityID, path+".industry_chain_entity_id", tree.IndustryChainEntityID, "must be a standard lowercase UUID")
		}
		if _, duplicate := seenChains[tree.IndustryChainEntityID]; duplicate {
			return invalidReasonTree(tree.IndustryChainEntityID, path+".industry_chain_entity_id", tree.IndustryChainEntityID, "must be unique within the Theme")
		}
		seenChains[tree.IndustryChainEntityID] = struct{}{}
		if err := tree.validate(path, signalSnapshots); err != nil {
			return err
		}
	}
	return nil
}

func (t ReasonTreeInput) validate(path string, snapshots map[string]ReasonTreeSignalInput) error {
	if err := requiredReasonTreeText(t.IndustryChainEntityID, path+".title", t.Title, 300); err != nil {
		return err
	}
	if err := requiredReasonTreeText(t.IndustryChainEntityID, path+".one_line_conclusion", t.OneLineConclusion, 1000); err != nil {
		return err
	}
	if !isAllowedValue(t.ImpactDirection, "positive", "negative", "mixed", "neutral", "uncertain") {
		return invalidReasonTree(t.IndustryChainEntityID, path+".impact_direction", t.ImpactDirection, "has an unsupported value")
	}
	if !isAllowedValue(t.ImpactStrength, "strong", "medium", "weak", "unknown") {
		return invalidReasonTree(t.IndustryChainEntityID, path+".impact_strength", t.ImpactStrength, "has an unsupported value")
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
		if err := optionalReasonTreeText(t.IndustryChainEntityID, path+"."+field.name, field.value, field.max); err != nil {
			return err
		}
	}
	for index, condition := range t.InvalidationConditions {
		if err := requiredReasonTreeText(t.IndustryChainEntityID, fmt.Sprintf("%s.invalidation_conditions[%d]", path, index), condition, 2000); err != nil {
			return err
		}
	}
	for index, checkpoint := range t.Checkpoints {
		checkpointPath := fmt.Sprintf("%s.checkpoints[%d]", path, index)
		if !isAllowedValue(checkpoint.Type, "event", "relationship", "metric") {
			return invalidReasonTree(t.IndustryChainEntityID, checkpointPath+".type", checkpoint.Type, "has an unsupported value")
		}
		if err := requiredReasonTreeText(t.IndustryChainEntityID, checkpointPath+".summary", checkpoint.Summary, 2000); err != nil {
			return err
		}
	}
	seenEvents := make(map[string]struct{}, len(t.Events))
	for index, event := range t.Events {
		eventPath := fmt.Sprintf("%s.events[%d]", path, index)
		if event.DisplayOrder != index+1 {
			return invalidReasonTree(t.IndustryChainEntityID, eventPath+".display_order", fmt.Sprint(event.DisplayOrder), "must be contiguous from 1")
		}
		if !lowercaseUUIDPattern.MatchString(event.EventID) {
			return invalidReasonTree(t.IndustryChainEntityID, eventPath+".event_id", event.EventID, "must be a standard lowercase UUID")
		}
		if _, duplicate := seenEvents[event.EventID]; duplicate {
			return invalidReasonTree(t.IndustryChainEntityID, eventPath+".event_id", event.EventID, "must be unique within the Tree")
		}
		seenEvents[event.EventID] = struct{}{}
		if !isAllowedValue(event.EvidenceRole, "driver", "supporting", "contradicting", "context") {
			return invalidReasonTree(t.IndustryChainEntityID, eventPath+".evidence_role", event.EvidenceRole, "has an unsupported value")
		}
	}
	if len(t.Nodes) == 0 {
		return invalidReasonTree(t.IndustryChainEntityID, path+".nodes", "", "must contain at least one ReasonTreeNodeInput")
	}
	seenNodes := make(map[string]struct{}, len(t.Nodes))
	for index, node := range t.Nodes {
		nodePath := fmt.Sprintf("%s.nodes[%d]", path, index)
		if node.Position != index+1 {
			return invalidReasonTree(t.IndustryChainEntityID, nodePath+".position", fmt.Sprint(node.Position), "must be contiguous from 1")
		}
		if !lowercaseUUIDPattern.MatchString(node.ChainNodeEntityID) {
			return invalidReasonTree(t.IndustryChainEntityID, nodePath+".chain_node_entity_id", node.ChainNodeEntityID, "must be a standard lowercase UUID")
		}
		if _, duplicate := seenNodes[node.ChainNodeEntityID]; duplicate {
			return invalidReasonTree(t.IndustryChainEntityID, nodePath+".chain_node_entity_id", node.ChainNodeEntityID, "must be unique within the Tree")
		}
		seenNodes[node.ChainNodeEntityID] = struct{}{}
		if !isAllowedValue(node.ImpactDirection, "positive", "negative", "mixed", "neutral", "uncertain") {
			return invalidReasonTree(t.IndustryChainEntityID, nodePath+".impact_direction", node.ImpactDirection, "has an unsupported value")
		}
		if !isAllowedValue(node.ImpactStrength, "strong", "medium", "weak", "unknown") {
			return invalidReasonTree(t.IndustryChainEntityID, nodePath+".impact_strength", node.ImpactStrength, "has an unsupported value")
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
			if err := optionalReasonTreeText(t.IndustryChainEntityID, nodePath+"."+field.name, field.value, field.max); err != nil {
				return err
			}
		}
		if index == 0 {
			if node.IncomingIndustryChainGraphEdgeID != nil || node.IncomingTransmissionTitle != nil ||
				node.IncomingTransmissionMechanism != nil || node.IncomingConditionSummary != nil {
				return invalidReasonTree(t.IndustryChainEntityID, nodePath+".incoming_*", "", "must all be null for the first ReasonTreeNodeInput")
			}
		} else {
			if node.IncomingIndustryChainGraphEdgeID != nil && !lowercaseUUIDPattern.MatchString(*node.IncomingIndustryChainGraphEdgeID) {
				return invalidReasonTree(t.IndustryChainEntityID, nodePath+".incoming_industry_chain_graph_edge_id", *node.IncomingIndustryChainGraphEdgeID, "must be a standard lowercase UUID")
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
					return invalidReasonTree(t.IndustryChainEntityID, nodePath+"."+field.name, "", "is required after the first ReasonTreeNodeInput")
				}
				if err := requiredReasonTreeText(t.IndustryChainEntityID, nodePath+"."+field.name, *field.value, 4000); err != nil {
					return err
				}
			}
		}
		if err := validateReasonTreeSignals(t.IndustryChainEntityID, nodePath, node.Signals, snapshots); err != nil {
			return err
		}
	}
	return nil
}

func validateReasonTreeSignals(chainID, nodePath string, signals []ReasonTreeSignalInput, snapshots map[string]ReasonTreeSignalInput) error {
	if len(signals) < 1 || len(signals) > 5 {
		return invalidReasonTree(chainID, nodePath+".signals", "", "must contain 1..5 ReasonTreeSignalInput snapshots")
	}
	seen := make(map[string]struct{}, len(signals))
	primaryCount := 0
	for index, signal := range signals {
		path := fmt.Sprintf("%s.signals[%d]", nodePath, index)
		if signal.DisplayOrder != index+1 {
			return invalidReasonTree(chainID, path+".display_order", fmt.Sprint(signal.DisplayOrder), "must be contiguous from 1")
		}
		if !researchKeyPattern.MatchString(signal.VariableSignalKey) {
			return invalidReasonTree(chainID, path+".variable_signal_key", signal.VariableSignalKey, "must match ^[a-z0-9][a-z0-9._:-]{0,127}$")
		}
		if _, duplicate := seen[signal.VariableSignalKey]; duplicate {
			return invalidReasonTree(chainID, path+".variable_signal_key", signal.VariableSignalKey, "must be unique within the ReasonTreeNodeInput")
		}
		seen[signal.VariableSignalKey] = struct{}{}
		if !isAllowedValue(signal.SignalRole, "primary", "supporting", "contradicting") {
			return invalidReasonTree(chainID, path+".signal_role", signal.SignalRole, "has an unsupported value")
		}
		if signal.SignalRole == "primary" {
			primaryCount++
			if signal.DisplayOrder != 1 {
				return invalidReasonTree(chainID, path+".display_order", fmt.Sprint(signal.DisplayOrder), "primary ReasonTreeSignalInput must have display_order 1")
			}
		}
		if !isAllowedValue(signal.SignalDirection, "increase", "decrease", "mixed", "unchanged", "uncertain") {
			return invalidReasonTree(chainID, path+".signal_direction", signal.SignalDirection, "has an unsupported value")
		}
		trimmed := strings.TrimSpace(signal.DisplaySummary)
		if trimmed == "" || utf8.RuneCountInString(trimmed) > 200 || trimmed != signal.DisplaySummary {
			return invalidReasonTree(chainID, path+".display_summary", "", "must be a trimmed 1..200 character string")
		}
		if prior, exists := snapshots[signal.VariableSignalKey]; exists {
			if prior.SignalDirection != signal.SignalDirection || prior.DisplaySummary != signal.DisplaySummary {
				return invalidReasonTree(chainID, path+".variable_signal_key", signal.VariableSignalKey, "must keep the same direction and display summary within the analysis batch")
			}
		} else {
			snapshots[signal.VariableSignalKey] = signal
		}
	}
	if primaryCount != 1 {
		return invalidReasonTree(chainID, nodePath+".signals", "", "must contain exactly one primary ReasonTreeSignalInput")
	}
	return nil
}

func requiredReasonTreeText(chainID, path, value string, max int) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return invalidReasonTree(chainID, path, "", "is required")
	}
	if utf8.RuneCountInString(trimmed) > max {
		return invalidReasonTree(chainID, path, "", fmt.Sprintf("must contain at most %d characters", max))
	}
	return nil
}

func optionalReasonTreeText(chainID, path string, value *string, max int) error {
	if value != nil && utf8.RuneCountInString(strings.TrimSpace(*value)) > max {
		return invalidReasonTree(chainID, path, "", fmt.Sprintf("must contain at most %d characters", max))
	}
	return nil
}

func invalidReasonTree(chainID, path, reference, message string) *ReasonTreeValidationError {
	return &ReasonTreeValidationError{IndustryChainEntityID: chainID, Path: path, Reference: reference, Message: message}
}

type EventSemanticBundle struct {
	Event           Event            `json:"event"`
	Evidence        []Evidence       `json:"evidence"`
	EntityLinks     []EntityLink     `json:"entity_links"`
	VariableSignals []VariableSignal `json:"variable_signals"`
}

type Event = eventbiz.ResearchEventFact
type Evidence = eventbiz.ResearchEvidenceFact
type EntityLink = eventsemanticbiz.ResearchEntityLink
type VariableSignal = eventsemanticbiz.ResearchVariableSignal
type Measurement = eventsemanticbiz.ResearchMeasurement
type DirectImpact = eventsemanticbiz.ResearchDirectImpact
type Entity = entitybiz.ResearchGraphEntity
type RelationDefinition = entitybiz.ResearchGraphRelation
type EntityRelation = entitybiz.ResearchGraphEntityRelation
type IndustryChain = entitybiz.ResearchGraphIndustryChain
type IndustryChainMembership = entitybiz.ResearchGraphMembership
type IndustryChainGraphEdge = entitybiz.ResearchGraphIndustryEdge
type EntityTypeContext = entitybiz.ResearchEntityTypeContext
type VariableDefinition = eventsemanticbiz.ResearchVariableDefinition
type DirectTransmissionRule = eventsemanticbiz.ResearchDirectTransmissionRule
type AcceptancePolicy = eventsemanticbiz.ResearchAcceptancePolicy

type Dictionaries struct {
	Entities                 []Entity                  `json:"entities"`
	RelationDefinitions      []RelationDefinition      `json:"relation_definitions"`
	EntityRelations          []EntityRelation          `json:"entity_relations"`
	IndustryChains           []IndustryChain           `json:"industry_chains"`
	IndustryChainMemberships []IndustryChainMembership `json:"industry_chain_memberships"`
	IndustryChainGraphEdges  []IndustryChainGraphEdge  `json:"industry_chain_graph_edges"`
	EntityTypeDefinitions    []EntityTypeContext       `json:"entity_type_definitions"`
	VariableDefinitions      []VariableDefinition      `json:"variable_definitions"`
	DirectTransmissionRules  []DirectTransmissionRule  `json:"direct_transmission_rules"`
	AcceptancePolicies       []AcceptancePolicy        `json:"acceptance_policies"`
}

var ErrHistoricalSemanticsUnavailable = eventsemanticbiz.ErrResearchHistoricalSemanticsUnavailable
var ErrReferenceClosureInconsistent = eventsemanticbiz.ErrResearchReferenceClosureInconsistent

type EventProvider interface {
	ListResearchEvents(context.Context, eventbiz.ResearchEventQuery) (eventbiz.ResearchEventPage, error)
}

type SemanticProvider interface {
	ListResearchSemantics(context.Context, eventsemanticbiz.ResearchSemanticQuery) ([]eventsemanticbiz.ResearchSemanticRecord, error)
	ResearchSemanticClosure(context.Context, eventsemanticbiz.ResearchSemanticClosureQuery) (eventsemanticbiz.ResearchSemanticDictionaries, error)
}

type EntityProvider interface {
	ResearchReferenceClosure(context.Context, entitybiz.ResearchReferenceQuery) (entitybiz.ResearchReferenceDictionaries, error)
	SearchResearchGraph(context.Context, GraphQuery) (GraphSubgraph, error)
}

type AnalysisContextStoreQuery struct {
	DiscoveryWindowStart      time.Time
	DiscoveryWindowEnd        time.Time
	AnalysisAsOf              time.Time
	PredictionHorizonStart    *time.Time
	PredictionHorizonEnd      *time.Time
	PageSize                  int
	AfterKnowledgeAvailableAt *time.Time
	AfterEventID              string
}

type BundleRecord struct {
	KnowledgeAvailableAt time.Time
	EventID              string
	Bundle               EventSemanticBundle
}

type AnalysisContextStorePage struct {
	Bundles      []BundleRecord
	Dictionaries Dictionaries
	HasMore      bool
}

type VersionedReference = eventsemanticbiz.ResearchVersionedReference

type ReferenceClosureQuery struct {
	AnalysisAsOf            time.Time
	EntityIDs               []string
	EntityRelationIDs       []string
	VariableDefinitions     []VersionedReference
	DirectTransmissionRules []VersionedReference
	SemanticSubmissionIDs   []string
}

const (
	AnalysisContextContractVersion       = "research-analysis-context.v1"
	AnalysisContextTBoxContractVersion   = "event-semantics.phase-one@1"
	AnalysisContextStableOrderingVersion = "knowledge-available-at-event-id.v1"
	AnalysisContextTemporalSemantics     = "retrospective_reconstruction"
	AnalysisContextTemporalLimitation    = "Event semantics are filtered by analysis_as_of; TBox and relationship dictionaries are a current-state reconstruction and do not claim strict historical replay"
	MaxDiscoveryWindow                   = 366 * 24 * time.Hour
	MaxEventSemanticBundleBytes          = 512 * 1024
	MaxDictionaryBytes                   = 4 * 1024 * 1024
	MaxPageBytes                         = 8 * 1024 * 1024
	MaxEventSemanticBundleRows           = 1_000
	MaxDictionaryRows                    = 50_000
)

type AnalysisContextRequest struct {
	DiscoveryWindowStart   string
	DiscoveryWindowEnd     string
	AnalysisAsOf           string
	PredictionHorizonStart *string
	PredictionHorizonEnd   *string
	PageSize               int
	Cursor                 string
}

type AnalysisContextResult struct {
	ContractVersion             string                `json:"contract_version"`
	TBoxContractVersion         string                `json:"tbox_contract_version"`
	TemporalSemantics           string                `json:"temporal_semantics"`
	TemporalLimitation          string                `json:"temporal_limitation"`
	EventPageFingerprint        string                `json:"event_page_fingerprint"`
	ReferenceClosureFingerprint string                `json:"reference_closure_fingerprint"`
	DiscoveryWindowStart        string                `json:"discovery_window_start"`
	DiscoveryWindowEnd          string                `json:"discovery_window_end"`
	AnalysisAsOf                string                `json:"analysis_as_of"`
	PredictionHorizonStart      *string               `json:"prediction_horizon_start,omitempty"`
	PredictionHorizonEnd        *string               `json:"prediction_horizon_end,omitempty"`
	EventSemanticBundles        []EventSemanticBundle `json:"event_semantic_bundles"`
	Dictionaries                Dictionaries          `json:"dictionaries"`
	NextCursor                  string                `json:"next_cursor,omitempty"`
	HasMore                     bool                  `json:"has_more"`
}

type AnalysisContextValidationError struct {
	Reason string
}

type ResearchResourceLimitError struct {
	Reason        string
	Component     string
	ActualRows    *int64
	MaxRows       *int64
	ActualBytes   *int64
	MaxBytes      *int64
	RetryGuidance string
}

func (e *ResearchResourceLimitError) Error() string {
	return e.Reason
}

func (e *AnalysisContextValidationError) Error() string {
	return e.Reason
}

func (s *UseCase) List(ctx context.Context, request AnalysisContextRequest) (AnalysisContextResult, error) {
	if s == nil || s.eventProvider == nil || s.semanticProvider == nil || s.entityProvider == nil {
		return AnalysisContextResult{}, errors.New("Research Analysis Context providers are required")
	}
	query, normalized, fingerprint, err := validateAnalysisContextRequest(request)
	if err != nil {
		return AnalysisContextResult{}, err
	}
	if request.Cursor != "" {
		decoded, err := decodeCursor(request.Cursor)
		if err != nil || decoded.Version != 2 ||
			decoded.Fingerprint != fingerprint {
			return AnalysisContextResult{}, &AnalysisContextValidationError{Reason: "cursor does not match the Analysis Context query"}
		}
		after, err := parseUTC("cursor.knowledge_available_at", decoded.KnowledgeAvailableAt)
		if err != nil || !identity.IsUUID(decoded.EventID) {
			return AnalysisContextResult{}, &AnalysisContextValidationError{Reason: "cursor is invalid"}
		}
		query.AfterKnowledgeAvailableAt = &after
		query.AfterEventID = decoded.EventID
	}
	eventPage, err := s.eventProvider.ListResearchEvents(ctx, eventbiz.ResearchEventQuery{
		DiscoveryWindowStart:      query.DiscoveryWindowStart,
		DiscoveryWindowEnd:        query.DiscoveryWindowEnd,
		AnalysisAsOf:              query.AnalysisAsOf,
		PageSize:                  query.PageSize,
		AfterKnowledgeAvailableAt: query.AfterKnowledgeAvailableAt,
		AfterEventID:              query.AfterEventID,
	})
	if err != nil {
		return AnalysisContextResult{}, err
	}
	eventIDs := make([]string, 0, len(eventPage.Events))
	for _, event := range eventPage.Events {
		eventIDs = append(eventIDs, event.Event.ID)
	}
	semanticRecords, err := s.semanticProvider.ListResearchSemantics(ctx, eventsemanticbiz.ResearchSemanticQuery{
		EventIDs: eventIDs, DiscoveryWindowStart: query.DiscoveryWindowStart,
		DiscoveryWindowEnd: query.DiscoveryWindowEnd, AnalysisAsOf: query.AnalysisAsOf,
	})
	if err != nil {
		return AnalysisContextResult{}, mapResearchSemanticProviderError(err)
	}
	page, err := assembleAnalysisContextPage(eventPage, semanticRecords)
	if err != nil {
		return AnalysisContextResult{}, err
	}
	closure := buildReferenceClosureQuery(query.AnalysisAsOf, page.Bundles)
	semanticDictionaries, err := s.semanticProvider.ResearchSemanticClosure(ctx, eventsemanticbiz.ResearchSemanticClosureQuery{
		AnalysisAsOf:            closure.AnalysisAsOf,
		VariableDefinitions:     closure.VariableDefinitions,
		DirectTransmissionRules: closure.DirectTransmissionRules,
		SemanticSubmissionIDs:   closure.SemanticSubmissionIDs,
	})
	if err != nil {
		return AnalysisContextResult{}, mapResearchSemanticProviderError(err)
	}
	relationTypes := make([]string, 0, len(semanticDictionaries.DirectTransmissionRules))
	entityTypes := make([]string, 0)
	for _, definition := range semanticDictionaries.VariableDefinitions {
		entityTypes = append(entityTypes, definition.ApplicableEntityTypes...)
	}
	for _, rule := range semanticDictionaries.DirectTransmissionRules {
		relationTypes = append(relationTypes, rule.RelationType)
		entityTypes = append(entityTypes, rule.SourceEntityType, rule.TargetEntityType)
	}
	entityDictionaries, err := s.entityProvider.ResearchReferenceClosure(ctx, entitybiz.ResearchReferenceQuery{
		AnalysisAsOf:      closure.AnalysisAsOf,
		EntityIDs:         closure.EntityIDs,
		EntityRelationIDs: closure.EntityRelationIDs,
		RelationTypes:     sortedUniqueStrings(relationTypes),
		EntityTypes:       sortedUniqueStrings(entityTypes),
	})
	if err != nil {
		if errors.Is(err, entitybiz.ErrResearchHistoricalReferencesUnavailable) {
			return AnalysisContextResult{}, ErrHistoricalSemanticsUnavailable
		}
		return AnalysisContextResult{}, err
	}
	page.Dictionaries = normalizeDictionaries(Dictionaries{
		Entities:                 entityDictionaries.Entities,
		RelationDefinitions:      entityDictionaries.RelationDefinitions,
		EntityRelations:          entityDictionaries.EntityRelations,
		IndustryChains:           entityDictionaries.IndustryChains,
		IndustryChainMemberships: entityDictionaries.IndustryChainMemberships,
		IndustryChainGraphEdges:  entityDictionaries.IndustryChainGraphEdges,
		EntityTypeDefinitions:    entityDictionaries.EntityTypeDefinitions,
		VariableDefinitions:      semanticDictionaries.VariableDefinitions,
		DirectTransmissionRules:  semanticDictionaries.DirectTransmissionRules,
		AcceptancePolicies:       semanticDictionaries.AcceptancePolicies,
	})
	if !researchContextReferencesResolve(page) {
		return AnalysisContextResult{}, ErrReferenceClosureInconsistent
	}
	result := AnalysisContextResult{
		ContractVersion:        AnalysisContextContractVersion,
		TBoxContractVersion:    AnalysisContextTBoxContractVersion,
		TemporalSemantics:      AnalysisContextTemporalSemantics,
		TemporalLimitation:     AnalysisContextTemporalLimitation,
		DiscoveryWindowStart:   normalized.DiscoveryWindowStart,
		DiscoveryWindowEnd:     normalized.DiscoveryWindowEnd,
		AnalysisAsOf:           normalized.AnalysisAsOf,
		PredictionHorizonStart: normalized.PredictionHorizonStart,
		PredictionHorizonEnd:   normalized.PredictionHorizonEnd,
		EventSemanticBundles:   make([]EventSemanticBundle, 0, len(page.Bundles)),
		Dictionaries:           page.Dictionaries,
		HasMore:                page.HasMore,
	}
	dictionaryPayload, err := json.Marshal(result.Dictionaries)
	if err != nil {
		return AnalysisContextResult{}, errors.New("research Analysis Context dictionaries are invalid")
	}
	dictionaryRowCount := dictionaryRows(result.Dictionaries)
	if dictionaryRowCount > MaxDictionaryRows || len(dictionaryPayload) > MaxDictionaryBytes {
		return AnalysisContextResult{}, &ResearchResourceLimitError{
			Reason:        "Research Analysis Context reference closure exceeds the response budget",
			Component:     "reference_closure",
			ActualRows:    int64Reference(int64(dictionaryRowCount)),
			MaxRows:       int64Reference(MaxDictionaryRows),
			ActualBytes:   int64Reference(int64(len(dictionaryPayload))),
			MaxBytes:      int64Reference(MaxDictionaryBytes),
			RetryGuidance: "reduce_page_size",
		}
	}
	pageBytes := len(dictionaryPayload)
	for _, bundle := range page.Bundles {
		if bundle.KnowledgeAvailableAt.IsZero() || !identity.IsUUID(bundle.EventID) ||
			bundle.Bundle.Event.ID != bundle.EventID {
			return AnalysisContextResult{}, errors.New("research Analysis Context bundle is invalid")
		}
		bundlePayload, err := json.Marshal(bundle.Bundle)
		if err != nil {
			return AnalysisContextResult{}, errors.New("research Analysis Context bundle is invalid")
		}
		bundleRowCount := eventSemanticBundleRows(bundle.Bundle)
		if bundleRowCount > MaxEventSemanticBundleRows || len(bundlePayload) > MaxEventSemanticBundleBytes {
			return AnalysisContextResult{}, &ResearchResourceLimitError{
				Reason:        "an Event Semantic Bundle exceeds the response budget",
				Component:     "event_semantic_bundle",
				ActualRows:    int64Reference(int64(bundleRowCount)),
				MaxRows:       int64Reference(MaxEventSemanticBundleRows),
				ActualBytes:   int64Reference(int64(len(bundlePayload))),
				MaxBytes:      int64Reference(MaxEventSemanticBundleBytes),
				RetryGuidance: "event_bundle_requires_provider_remediation",
			}
		}
		pageBytes += len(bundlePayload)
		if pageBytes > MaxPageBytes {
			return AnalysisContextResult{}, &ResearchResourceLimitError{
				Reason:        "Research Analysis Context page exceeds the response budget",
				Component:     "analysis_context_page",
				ActualBytes:   int64Reference(int64(pageBytes)),
				MaxBytes:      int64Reference(MaxPageBytes),
				RetryGuidance: "reduce_page_size",
			}
		}
		result.EventSemanticBundles = append(result.EventSemanticBundles, bundle.Bundle)
	}
	result.EventPageFingerprint, err = payloadFingerprint(result.EventSemanticBundles)
	if err != nil {
		return AnalysisContextResult{}, errors.New("research Analysis Context Event page is invalid")
	}
	result.ReferenceClosureFingerprint, err = payloadFingerprint(result.Dictionaries)
	if err != nil {
		return AnalysisContextResult{}, errors.New("research Analysis Context reference closure is invalid")
	}
	if page.HasMore {
		if len(page.Bundles) == 0 {
			return AnalysisContextResult{}, errors.New("research Analysis Context continuation has no terminal bundle")
		}
		last := page.Bundles[len(page.Bundles)-1]
		result.NextCursor, err = encodeCursor(contextCursor{
			Version:              2,
			Fingerprint:          fingerprint,
			KnowledgeAvailableAt: last.KnowledgeAvailableAt.UTC().Format(time.RFC3339Nano),
			EventID:              last.EventID,
		})
		if err != nil {
			return AnalysisContextResult{}, err
		}
	}
	return result, nil
}

func mapResearchSemanticProviderError(err error) error {
	var limit *eventsemanticbiz.ResearchResourceLimitError
	if !errors.As(err, &limit) {
		return err
	}
	return &ResearchResourceLimitError{
		Reason: limit.Reason, Component: limit.Component,
		ActualRows: limit.ActualRows, MaxRows: limit.MaxRows,
		ActualBytes: limit.ActualBytes, MaxBytes: limit.MaxBytes,
		RetryGuidance: limit.RetryGuidance,
	}
}

func assembleAnalysisContextPage(
	eventPage eventbiz.ResearchEventPage,
	semanticRecords []eventsemanticbiz.ResearchSemanticRecord,
) (AnalysisContextStorePage, error) {
	semantics := make(map[string]eventsemanticbiz.ResearchSemanticRecord, len(semanticRecords))
	for _, record := range semanticRecords {
		if !identity.IsUUID(record.EventID) {
			return AnalysisContextStorePage{}, errors.New("Research semantic provider returned an invalid Event reference")
		}
		if _, duplicate := semantics[record.EventID]; duplicate {
			return AnalysisContextStorePage{}, errors.New("Research semantic provider returned a duplicate Event")
		}
		semantics[record.EventID] = record
	}
	page := AnalysisContextStorePage{
		Bundles: make([]BundleRecord, 0, len(eventPage.Events)),
		HasMore: eventPage.HasMore,
	}
	for _, facts := range eventPage.Events {
		semantic, ok := semantics[facts.Event.ID]
		if !ok {
			return AnalysisContextStorePage{}, errors.New("Research semantic provider omitted an Event")
		}
		links, signals := filterSemanticsByEvidence(facts.Evidence, semantic.EntityLinks, semantic.VariableSignals)
		page.Bundles = append(page.Bundles, BundleRecord{
			KnowledgeAvailableAt: facts.KnowledgeAvailableAt,
			EventID:              facts.Event.ID,
			Bundle: EventSemanticBundle{
				Event: facts.Event, Evidence: facts.Evidence,
				EntityLinks: links, VariableSignals: signals,
			},
		})
	}
	if len(semantics) != len(eventPage.Events) {
		return AnalysisContextStorePage{}, errors.New("Research semantic provider returned an unexpected Event")
	}
	return page, nil
}

func filterSemanticsByEvidence(
	evidence []Evidence,
	links []EntityLink,
	signals []VariableSignal,
) ([]EntityLink, []VariableSignal) {
	available := make(map[string]struct{}, len(evidence))
	for _, item := range evidence {
		available[item.EvidenceID] = struct{}{}
	}
	filteredLinks := make([]EntityLink, 0, len(links))
	linkIDs := make(map[string]struct{}, len(links))
	for _, link := range links {
		if stringSetAvailable(link.EvidenceIDs, available) {
			filteredLinks = append(filteredLinks, link)
			linkIDs[link.EventEntityLinkID] = struct{}{}
		}
	}
	filteredSignals := make([]VariableSignal, 0, len(signals))
	for _, signal := range signals {
		if _, ok := linkIDs[signal.SubjectEventEntityLinkID]; !ok || !stringSetAvailable(signal.EvidenceIDs, available) {
			continue
		}
		measurements := make([]Measurement, 0, len(signal.Measurements))
		for _, measurement := range signal.Measurements {
			if _, ok := available[measurement.EvidenceID]; ok {
				measurements = append(measurements, measurement)
			}
		}
		impacts := make([]DirectImpact, 0, len(signal.DirectImpacts))
		for _, impact := range signal.DirectImpacts {
			if stringSetAvailable(impact.EvidenceIDs, available) {
				impacts = append(impacts, impact)
			}
		}
		signal.Measurements = measurements
		signal.DirectImpacts = impacts
		filteredSignals = append(filteredSignals, signal)
	}
	return filteredLinks, filteredSignals
}

func stringSetAvailable(values []string, available map[string]struct{}) bool {
	for _, value := range values {
		if _, ok := available[value]; !ok {
			return false
		}
	}
	return true
}

func sortedUniqueStrings(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return sortedSet(set)
}

func buildReferenceClosureQuery(
	analysisAsOf time.Time,
	bundles []BundleRecord,
) ReferenceClosureQuery {
	entityIDs := map[string]struct{}{}
	relationIDs := map[string]struct{}{}
	variableDefinitions := map[string]VersionedReference{}
	rules := map[string]VersionedReference{}
	submissionIDs := map[string]struct{}{}
	for _, record := range bundles {
		for _, link := range record.Bundle.EntityLinks {
			entityIDs[link.EntityID] = struct{}{}
			if link.SemanticSubmissionID != "" {
				submissionIDs[link.SemanticSubmissionID] = struct{}{}
			}
		}
		for _, signal := range record.Bundle.VariableSignals {
			entityIDs[signal.SubjectEntityID] = struct{}{}
			variable := VersionedReference{Key: signal.VariableKey, Version: signal.VariableVersion}
			variableDefinitions[versionedKey(variable.Key, variable.Version)] = variable
			if signal.SemanticSubmissionID != "" {
				submissionIDs[signal.SemanticSubmissionID] = struct{}{}
			}
			for _, impact := range signal.DirectImpacts {
				entityIDs[impact.TargetEntityID] = struct{}{}
				affected := VersionedReference{
					Key: impact.AffectedVariableKey, Version: impact.AffectedVariableVersion,
				}
				variableDefinitions[versionedKey(affected.Key, affected.Version)] = affected
				if impact.EntityRelationID != nil {
					relationIDs[*impact.EntityRelationID] = struct{}{}
				}
				if impact.RuleKey != nil && impact.RuleVersion != nil {
					rule := VersionedReference{Key: *impact.RuleKey, Version: *impact.RuleVersion}
					rules[versionedKey(rule.Key, rule.Version)] = rule
				}
				if impact.SemanticSubmissionID != "" {
					submissionIDs[impact.SemanticSubmissionID] = struct{}{}
				}
			}
		}
	}
	return ReferenceClosureQuery{
		AnalysisAsOf:            analysisAsOf,
		EntityIDs:               sortedSet(entityIDs),
		EntityRelationIDs:       sortedSet(relationIDs),
		VariableDefinitions:     sortedVersionedReferences(variableDefinitions),
		DirectTransmissionRules: sortedVersionedReferences(rules),
		SemanticSubmissionIDs:   sortedSet(submissionIDs),
	}
}

func sortedSet(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func sortedVersionedReferences(
	values map[string]VersionedReference,
) []VersionedReference {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]VersionedReference, 0, len(keys))
	for _, key := range keys {
		result = append(result, values[key])
	}
	return result
}

func researchContextReferencesResolve(page AnalysisContextStorePage) bool {
	entityTypes := make(map[string]struct{}, len(page.Dictionaries.EntityTypeDefinitions))
	for _, definition := range page.Dictionaries.EntityTypeDefinitions {
		entityTypes[definition.TypeKey] = struct{}{}
	}
	entities := make(map[string]struct{}, len(page.Dictionaries.Entities))
	for _, entity := range page.Dictionaries.Entities {
		if !containsID(entityTypes, entity.EntityType) {
			return false
		}
		entities[entity.EntityID] = struct{}{}
	}
	relationTypes := make(map[string]struct{}, len(page.Dictionaries.RelationDefinitions))
	for _, definition := range page.Dictionaries.RelationDefinitions {
		relationTypes[definition.RelationType] = struct{}{}
	}
	relations := make(map[string]EntityRelation, len(page.Dictionaries.EntityRelations))
	for _, relation := range page.Dictionaries.EntityRelations {
		if !containsID(entities, relation.FromEntityID) ||
			!containsID(entities, relation.ToEntityID) ||
			!containsID(relationTypes, relation.RelationType) {
			return false
		}
		relations[relation.EntityRelationID] = relation
	}
	variables := make(map[string]struct{}, len(page.Dictionaries.VariableDefinitions))
	for _, definition := range page.Dictionaries.VariableDefinitions {
		for _, entityType := range definition.ApplicableEntityTypes {
			if !containsID(entityTypes, entityType) {
				return false
			}
		}
		variables[versionedKey(definition.Key, definition.Version)] = struct{}{}
	}
	rules := make(map[string]struct{}, len(page.Dictionaries.DirectTransmissionRules))
	for _, rule := range page.Dictionaries.DirectTransmissionRules {
		if !containsID(variables, versionedKey(rule.SourceVariableKey, rule.SourceVariableVersion)) ||
			!containsID(variables, versionedKey(rule.AffectedVariableKey, rule.AffectedVariableVersion)) ||
			!containsID(entityTypes, rule.SourceEntityType) ||
			!containsID(entityTypes, rule.TargetEntityType) ||
			!containsID(relationTypes, rule.RelationType) {
			return false
		}
		rules[versionedKey(rule.RuleKey, rule.Version)] = struct{}{}
	}
	chains := make(map[string]struct{}, len(page.Dictionaries.IndustryChains))
	for _, chain := range page.Dictionaries.IndustryChains {
		if !containsID(entities, chain.IndustryChainEntityID) {
			return false
		}
		chains[chain.IndustryChainEntityID] = struct{}{}
	}
	for _, entity := range page.Dictionaries.Entities {
		if entity.EntityType == "industry_chain" &&
			!containsID(chains, entity.EntityID) {
			return false
		}
	}
	memberships := make(map[string]struct{}, len(page.Dictionaries.IndustryChainMemberships))
	for _, membership := range page.Dictionaries.IndustryChainMemberships {
		if !containsID(chains, membership.IndustryChainEntityID) ||
			!containsID(entities, membership.ChainNodeEntityID) {
			return false
		}
		memberships[membership.IndustryChainEntityID+"\x00"+membership.ChainNodeEntityID] = struct{}{}
	}
	for _, edge := range page.Dictionaries.IndustryChainGraphEdges {
		if !containsID(chains, edge.IndustryChainEntityID) ||
			!containsID(memberships, edge.IndustryChainEntityID+"\x00"+edge.FromChainNodeEntityID) ||
			!containsID(memberships, edge.IndustryChainEntityID+"\x00"+edge.ToChainNodeEntityID) {
			return false
		}
	}
	for _, record := range page.Bundles {
		bundle := record.Bundle
		evidence := make(map[string]struct{}, len(bundle.Evidence))
		for _, item := range bundle.Evidence {
			evidence[item.EvidenceID] = struct{}{}
		}
		links := make(map[string]EntityLink, len(bundle.EntityLinks))
		for _, link := range bundle.EntityLinks {
			if !containsID(entities, link.EntityID) || !allIDsResolve(evidence, link.EvidenceIDs) {
				return false
			}
			links[link.EventEntityLinkID] = link
		}
		signals := make(map[string]struct{}, len(bundle.VariableSignals))
		for _, signal := range bundle.VariableSignals {
			link, ok := links[signal.SubjectEventEntityLinkID]
			if !ok || link.EntityID != signal.SubjectEntityID ||
				!containsID(entities, signal.SubjectEntityID) ||
				!containsID(variables, versionedKey(signal.VariableKey, signal.VariableVersion)) ||
				!allIDsResolve(evidence, signal.EvidenceIDs) {
				return false
			}
			for _, measurement := range signal.Measurements {
				if !containsID(evidence, measurement.EvidenceID) {
					return false
				}
			}
			signals[signal.VariableSignalID] = struct{}{}
			for _, impact := range signal.DirectImpacts {
				if impact.SourceVariableSignalID != signal.VariableSignalID ||
					!containsID(entities, impact.TargetEntityID) ||
					!containsID(variables, versionedKey(
						impact.AffectedVariableKey, impact.AffectedVariableVersion,
					)) ||
					!allIDsResolve(evidence, impact.EvidenceIDs) {
					return false
				}
				if impact.EntityRelationID != nil {
					if _, ok := relations[*impact.EntityRelationID]; !ok {
						return false
					}
				}
				if impact.RuleKey != nil && impact.RuleVersion != nil &&
					!containsID(rules, versionedKey(*impact.RuleKey, *impact.RuleVersion)) {
					return false
				}
			}
		}
		if len(signals) != len(bundle.VariableSignals) {
			return false
		}
	}
	return true
}

func versionedKey(key string, version int) string {
	return fmt.Sprintf("%s@%d", key, version)
}

func containsID(set map[string]struct{}, id string) bool {
	_, ok := set[id]
	return ok
}

func allIDsResolve(set map[string]struct{}, ids []string) bool {
	for _, id := range ids {
		if !containsID(set, id) {
			return false
		}
	}
	return true
}

type normalizedAnalysisContextRequest struct {
	DiscoveryWindowStart   string
	DiscoveryWindowEnd     string
	AnalysisAsOf           string
	PredictionHorizonStart *string
	PredictionHorizonEnd   *string
}

func validateAnalysisContextRequest(
	request AnalysisContextRequest,
) (AnalysisContextStoreQuery, normalizedAnalysisContextRequest, string, error) {
	if request.PageSize < 1 || request.PageSize > 50 {
		return AnalysisContextStoreQuery{}, normalizedAnalysisContextRequest{}, "", &AnalysisContextValidationError{
			Reason: "page_size must be between 1 and 50",
		}
	}
	start, err := parseUTC("discovery_window_start", request.DiscoveryWindowStart)
	if err != nil {
		return AnalysisContextStoreQuery{}, normalizedAnalysisContextRequest{}, "", err
	}
	end, err := parseUTC("discovery_window_end", request.DiscoveryWindowEnd)
	if err != nil {
		return AnalysisContextStoreQuery{}, normalizedAnalysisContextRequest{}, "", err
	}
	asOf, err := parseUTC("analysis_as_of", request.AnalysisAsOf)
	if err != nil {
		return AnalysisContextStoreQuery{}, normalizedAnalysisContextRequest{}, "", err
	}
	if !start.Before(end) {
		return AnalysisContextStoreQuery{}, normalizedAnalysisContextRequest{}, "", &AnalysisContextValidationError{
			Reason: "discovery_window_end must be after discovery_window_start",
		}
	}
	if end.Sub(start) > MaxDiscoveryWindow {
		return AnalysisContextStoreQuery{}, normalizedAnalysisContextRequest{}, "", &ResearchResourceLimitError{
			Reason:        "discovery window exceeds the maximum technical budget of 366 days",
			Component:     "analysis_context_query",
			RetryGuidance: "reduce_discovery_window",
		}
	}
	if end.After(asOf) {
		return AnalysisContextStoreQuery{}, normalizedAnalysisContextRequest{}, "", &AnalysisContextValidationError{
			Reason: "discovery_window_end must not be after analysis_as_of",
		}
	}
	predictionStart, predictionEnd, err := predictionWindow(
		request.PredictionHorizonStart, request.PredictionHorizonEnd, asOf,
	)
	if err != nil {
		return AnalysisContextStoreQuery{}, normalizedAnalysisContextRequest{}, "", err
	}
	normalized := normalizedAnalysisContextRequest{
		DiscoveryWindowStart: start.Format(time.RFC3339Nano),
		DiscoveryWindowEnd:   end.Format(time.RFC3339Nano),
		AnalysisAsOf:         asOf.Format(time.RFC3339Nano),
	}
	if predictionStart != nil {
		value := predictionStart.Format(time.RFC3339Nano)
		normalized.PredictionHorizonStart = &value
		value = predictionEnd.Format(time.RFC3339Nano)
		normalized.PredictionHorizonEnd = &value
	}
	fingerprint, err := queryFingerprint(normalized, request.PageSize)
	if err != nil {
		return AnalysisContextStoreQuery{}, normalizedAnalysisContextRequest{}, "", err
	}
	return AnalysisContextStoreQuery{
		DiscoveryWindowStart: start, DiscoveryWindowEnd: end, AnalysisAsOf: asOf,
		PredictionHorizonStart: predictionStart, PredictionHorizonEnd: predictionEnd,
		PageSize: request.PageSize,
	}, normalized, fingerprint, nil
}

func predictionWindow(
	rawStart *string,
	rawEnd *string,
	asOf time.Time,
) (*time.Time, *time.Time, error) {
	if (rawStart == nil) != (rawEnd == nil) {
		return nil, nil, &AnalysisContextValidationError{
			Reason: "prediction horizon start and end must be provided together",
		}
	}
	if rawStart == nil {
		return nil, nil, nil
	}
	start, err := parseUTC("prediction_horizon_start", *rawStart)
	if err != nil {
		return nil, nil, err
	}
	end, err := parseUTC("prediction_horizon_end", *rawEnd)
	if err != nil {
		return nil, nil, err
	}
	if start.Before(asOf) || !start.Before(end) {
		return nil, nil, &AnalysisContextValidationError{
			Reason: "prediction horizon must start at or after analysis_as_of and end after its start",
		}
	}
	return &start, &end, nil
}

func parseUTC(name, value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, &AnalysisContextValidationError{Reason: fmt.Sprintf("%s must be an RFC3339 UTC timestamp", name)}
	}
	_, offset := parsed.Zone()
	if offset != 0 {
		return time.Time{}, &AnalysisContextValidationError{Reason: fmt.Sprintf("%s must use UTC", name)}
	}
	return parsed.UTC(), nil
}

func queryFingerprint(request normalizedAnalysisContextRequest, pageSize int) (string, error) {
	payload, err := json.Marshal(struct {
		AnalysisContextContractVersion       string                           `json:"contract_version"`
		AnalysisContextTBoxContractVersion   string                           `json:"tbox_contract_version"`
		AnalysisContextStableOrderingVersion string                           `json:"stable_ordering_version"`
		AnalysisContextRequest               normalizedAnalysisContextRequest `json:"request"`
		PageSize                             int                              `json:"page_size"`
	}{
		AnalysisContextContractVersion:       AnalysisContextContractVersion,
		AnalysisContextTBoxContractVersion:   AnalysisContextTBoxContractVersion,
		AnalysisContextStableOrderingVersion: AnalysisContextStableOrderingVersion,
		AnalysisContextRequest:               request,
		PageSize:                             pageSize,
	})
	if err != nil {
		return "", fmt.Errorf("encode Research Analysis Context query fingerprint: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func payloadFingerprint(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func normalizeDictionaries(value Dictionaries) Dictionaries {
	if value.Entities == nil {
		value.Entities = []Entity{}
	}
	if value.RelationDefinitions == nil {
		value.RelationDefinitions = []RelationDefinition{}
	}
	if value.EntityRelations == nil {
		value.EntityRelations = []EntityRelation{}
	}
	if value.IndustryChains == nil {
		value.IndustryChains = []IndustryChain{}
	}
	if value.IndustryChainMemberships == nil {
		value.IndustryChainMemberships = []IndustryChainMembership{}
	}
	if value.IndustryChainGraphEdges == nil {
		value.IndustryChainGraphEdges = []IndustryChainGraphEdge{}
	}
	if value.EntityTypeDefinitions == nil {
		value.EntityTypeDefinitions = []EntityTypeContext{}
	}
	if value.VariableDefinitions == nil {
		value.VariableDefinitions = []VariableDefinition{}
	}
	if value.DirectTransmissionRules == nil {
		value.DirectTransmissionRules = []DirectTransmissionRule{}
	}
	if value.AcceptancePolicies == nil {
		value.AcceptancePolicies = []AcceptancePolicy{}
	}
	return value
}

type contextCursor struct {
	Version              int    `json:"v"`
	Fingerprint          string `json:"fingerprint"`
	KnowledgeAvailableAt string `json:"knowledge_available_at"`
	EventID              string `json:"event_id"`
}

func encodeCursor[T contextCursor | researchCursor](cursor T) (string, error) {
	payload, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeCursor(raw string) (contextCursor, error) {
	payload, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return contextCursor{}, err
	}
	var cursor contextCursor
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cursor); err != nil {
		return contextCursor{}, err
	}
	return cursor, nil
}

func dictionaryRows(value Dictionaries) int {
	rows := len(value.Entities) +
		len(value.RelationDefinitions) +
		len(value.EntityRelations) +
		len(value.IndustryChains) +
		len(value.IndustryChainMemberships) +
		len(value.IndustryChainGraphEdges) +
		len(value.EntityTypeDefinitions) +
		len(value.VariableDefinitions) +
		len(value.DirectTransmissionRules) +
		len(value.AcceptancePolicies)
	for _, definition := range value.VariableDefinitions {
		rows += len(definition.ApplicableEntityTypes)
	}
	return rows
}

func eventSemanticBundleRows(value EventSemanticBundle) int {
	rows := len(value.Evidence) + len(value.EntityLinks) + len(value.VariableSignals)
	for _, signal := range value.VariableSignals {
		rows += len(signal.Measurements) + len(signal.DirectImpacts)
	}
	return rows
}

func int64Reference(value int64) *int64 {
	return &value
}

var (
	ErrResearchNotFound               = errors.New("research result not found")
	ErrResearchThemeNotFound          = errors.New("research theme not found")
	ErrResearchReasoningTreesNotFound = errors.New("research reasoning trees not found")
	ErrResearchReasoningTreeNotFound  = errors.New("research reasoning tree not found")
	ErrResearchReasoningTreeInvariant = errors.New("research reasoning tree invariant violation")
)

type ThemeImpactRecord struct {
	NodeKey           string  `json:"node_key"`
	DisplayName       string  `json:"display_name"`
	ChainNodeEntityID string  `json:"chain_node_entity_id"`
	Name              string  `json:"name"`
	RelationRole      string  `json:"relation_role"`
	ImpactDirection   string  `json:"impact_direction"`
	ImpactSummary     *string `json:"impact_summary"`
	DisplayOrder      int     `json:"display_order"`
}

type EventRecord struct {
	EventID        string     `json:"event_id"`
	EvidenceIDs    []string   `json:"evidence_ids"`
	Title          string     `json:"title"`
	Summary        string     `json:"summary"`
	EventTime      *time.Time `json:"event_time"`
	EvidenceRole   string     `json:"evidence_role"`
	SupportedClaim *string    `json:"supported_claim"`
	DisplayOrder   int        `json:"display_order,omitempty"`
}

type ThemeSummaryRecord struct {
	ID, AnalysisBatchID, Title, OneLineConclusion              string
	ConclusionDirection, ImpactStrength, TransmissionStage     string
	InvestmentGuidanceAction, InvestmentGuidanceSummary        string
	TimeHorizonCategory                                        string
	AttentionLevel, ConclusionStatus                           *string
	TimeHorizonSummary, TransmissionSummary, CheckpointSummary *string
	RiskSummary                                                *string
	AnalysisAsOf, WindowStart, WindowEnd, PublishedAt          time.Time
	Impacts                                                    []ThemeImpactRecord
	EvidenceEventCount, ReasoningTreeCount                     int
}

type ThemeDetailRecord struct {
	ThemeSummaryRecord
	ThemeKey, PublicationMode  string
	PublicationContractVersion int
	Events                     []EventRecord
}

type ThemeListFilter struct {
	WindowStart, WindowEnd, AsOf time.Time
	Limit                        int
	CursorPublishedAt            *time.Time
	CursorID                     string
}

type ThemeStorePage struct {
	AsOf, WindowStart, WindowEnd time.Time
	ThemeCount, EventCount       int
	Items                        []ThemeSummaryRecord
	HasMore                      bool
}

type ReasoningTreeSummaryRecord struct {
	ReasoningTreeID, IndustryChainEntityID, IndustryChainName, Title string
	TreeKey, DisplayName                                             string
	DisplayOrder, EventCount                                         int
	PublishedAt                                                      time.Time
}

type ReasoningTreeListRecord struct {
	Theme          ThemeSummaryRecord
	ReasoningTrees []ReasoningTreeSummaryRecord
}

type CheckpointRecord struct {
	Type, Summary string
}

type GraphEdgeRecord struct {
	ID, RelationType, ReviewStatus, Status string
}

type SignalRecord struct {
	VariableSignalKey, SignalRole, SignalDirection, DisplaySummary string
	SignalKey                                                      string
	VariableName, Direction                                        *string
	DisplayOrder                                                   int
}

type ReasoningTreeNodeRecord struct {
	ID, ChainNodeEntityID, Name, ImpactDirection, ImpactStrength string
	NodeKey, DisplayName                                         string
	Position                                                     int
	StateSummary, ImpactSummary, ReasoningBasisSummary           *string
	EvidenceGapSummary                                           *string
	IncomingIndustryChainGraphEdgeID, IncomingTransmissionTitle  *string
	IncomingTransmissionMechanism, IncomingConditionSummary      *string
	IncomingGraphEdge                                            *GraphEdgeRecord
	Signals                                                      []SignalRecord
}

type ReasoningTreeRecord struct {
	ReasoningTreeID, ThemeID, IndustryChainEntityID, IndustryChainName string
	TreeKey, DisplayName                                               string
	Title, OneLineConclusion, ImpactDirection, ImpactStrength          string
	DisplayOrder, EventCount                                           int
	FactSummary, TransmissionSummary, ImpactSummary                    *string
	ConclusionBoundarySummary, SupportSummary, CounterSummary          *string
	InvalidationConditions                                             []string
	Checkpoints                                                        []CheckpointRecord
	PublishedAt                                                        time.Time
	Events                                                             []EventRecord
	Nodes                                                              []ReasoningTreeNodeRecord
}

type ReasoningTreeDetailRecord struct {
	ThemeID, ThemeKey, PublicationMode string
	PublicationContractVersion         int
	ImpactNodeIDs                      []string
	ReasoningTree                      ReasoningTreeRecord
}

type Repository interface {
	ListResearchThemes(context.Context, ThemeListFilter) (ThemeStorePage, error)
	GetResearchTheme(context.Context, string) (ThemeDetailRecord, error)
	ListResearchThemeReasoningTrees(context.Context, string) (ReasoningTreeListRecord, error)
	GetResearchThemeReasoningTree(context.Context, string, string) (ReasoningTreeDetailRecord, error)
}

var (
	ErrThemeNotFound                   = errors.New("research Theme not found")
	ErrReasoningTreesNotFound          = errors.New("research Theme has no published reasoning trees")
	ErrReasoningTreeNotFound           = errors.New("research reasoning tree not found")
	ErrReasoningTreeInvariantViolation = errors.New("research reasoning tree invariant violation")
)

type ResearchReasoningTreeList struct {
	Theme          ResearchTheme                  `json:"theme"`
	ReasoningTrees []ResearchReasoningTreeSummary `json:"reasoning_trees"`
}

type ResearchReasoningTreeSummary struct {
	TreeKey               string    `json:"tree_key"`
	DisplayName           string    `json:"display_name"`
	ReasoningTreeID       string    `json:"reasoning_tree_id"`
	IndustryChainEntityID string    `json:"industry_chain_entity_id"`
	IndustryChainName     string    `json:"industry_chain_name"`
	Title                 string    `json:"title"`
	DisplayOrder          int       `json:"display_order"`
	EventCount            int       `json:"event_count"`
	PublishedAt           time.Time `json:"published_at"`
}

type ResearchCheckpoint struct {
	Type, Summary string
}

type ResearchGraphEdge struct {
	ID, RelationType, ReviewStatus, Status string
}

type ResearchSignal struct {
	SignalKey         string  `json:"signal_key"`
	VariableName      *string `json:"variable_name"`
	Direction         *string `json:"direction"`
	VariableSignalKey string  `json:"variable_signal_key"`
	SignalRole        string  `json:"signal_role"`
	SignalDirection   string  `json:"signal_direction"`
	DisplaySummary    string  `json:"display_summary"`
	DisplayOrder      int     `json:"display_order"`
}

type ResearchReasoningTreeNode struct {
	NodeKey                          string             `json:"node_key"`
	DisplayName                      string             `json:"display_name"`
	ID                               string             `json:"id"`
	Position                         int                `json:"position"`
	ChainNodeEntityID                string             `json:"chain_node_entity_id"`
	Name                             string             `json:"name"`
	StateSummary                     *string            `json:"state_summary"`
	ImpactDirection                  string             `json:"impact_direction"`
	ImpactStrength                   string             `json:"impact_strength"`
	ImpactSummary                    *string            `json:"impact_summary"`
	ReasoningBasisSummary            *string            `json:"reasoning_basis_summary"`
	EvidenceGapSummary               *string            `json:"evidence_gap_summary"`
	IncomingIndustryChainGraphEdgeID *string            `json:"incoming_industry_chain_graph_edge_id"`
	IncomingTransmissionTitle        *string            `json:"incoming_transmission_title"`
	IncomingTransmissionMechanism    *string            `json:"incoming_transmission_mechanism"`
	IncomingConditionSummary         *string            `json:"incoming_condition_summary"`
	IncomingGraphEdge                *ResearchGraphEdge `json:"incoming_graph_edge"`
	Signals                          []ResearchSignal   `json:"signals"`
	PrimarySignal                    ResearchSignal     `json:"primary_signal"`
	SignalDisplaySummary             string             `json:"signal_display_summary"`
}

type ResearchReasoningTree struct {
	ReasoningTreeID, ThemeID, IndustryChainEntityID, IndustryChainName string
	TreeKey, DisplayName                                               string
	Title, OneLineConclusion, ImpactDirection, ImpactStrength          string
	DisplayOrder, EventCount                                           int
	FactSummary, TransmissionSummary, ImpactSummary                    *string
	ConclusionBoundarySummary, SupportSummary, CounterSummary          *string
	InvalidationConditions                                             []string
	Checkpoints                                                        []ResearchCheckpoint
	PublishedAt                                                        time.Time
	Events                                                             []ResearchEvent
	Nodes                                                              []ResearchReasoningTreeNode
}

type ResearchReasoningTreeDetail struct {
	ThemeID, ThemeKey, PublicationMode string
	PublicationContractVersion         int
	ImpactNodeIDs                      []string
	ReasoningTree                      ResearchReasoningTree
}

func (s *UseCase) ListReasoningTrees(ctx context.Context, themeID string) (ResearchReasoningTreeList, error) {
	themeID = strings.ToLower(strings.TrimSpace(themeID))
	if !researchUUIDPattern.MatchString(themeID) {
		return ResearchReasoningTreeList{}, fmt.Errorf("%w: theme id must be a UUID", ErrInvalidRequest)
	}
	result, err := s.repository.ListResearchThemeReasoningTrees(ctx, themeID)
	if err != nil {
		return ResearchReasoningTreeList{}, mapReasoningTreeRepositoryError(err)
	}
	summaries := make([]ResearchReasoningTreeSummary, 0, len(result.ReasoningTrees))
	for _, value := range result.ReasoningTrees {
		summaries = append(summaries, ResearchReasoningTreeSummary{
			TreeKey: value.TreeKey, DisplayName: value.DisplayName,
			ReasoningTreeID:       value.ReasoningTreeID,
			IndustryChainEntityID: value.IndustryChainEntityID,
			IndustryChainName:     value.IndustryChainName, Title: value.Title,
			DisplayOrder: value.DisplayOrder, EventCount: value.EventCount,
			PublishedAt: value.PublishedAt.UTC(),
		})
	}
	return ResearchReasoningTreeList{Theme: themeDTO(result.Theme), ReasoningTrees: summaries}, nil
}

func (s *UseCase) GetReasoningTree(ctx context.Context, themeID, reasoningTreeID string) (ResearchReasoningTreeDetail, error) {
	themeID = strings.ToLower(strings.TrimSpace(themeID))
	reasoningTreeID = strings.ToLower(strings.TrimSpace(reasoningTreeID))
	if !researchUUIDPattern.MatchString(themeID) {
		return ResearchReasoningTreeDetail{}, fmt.Errorf("%w: theme id must be a UUID", ErrInvalidRequest)
	}
	if !researchUUIDPattern.MatchString(reasoningTreeID) {
		return ResearchReasoningTreeDetail{}, fmt.Errorf("%w: reasoning tree id must be a UUID", ErrInvalidRequest)
	}
	result, err := s.repository.GetResearchThemeReasoningTree(ctx, themeID, reasoningTreeID)
	if err != nil {
		return ResearchReasoningTreeDetail{}, mapReasoningTreeRepositoryError(err)
	}
	tree := result.ReasoningTree
	nodes := make([]ResearchReasoningTreeNode, 0, len(tree.Nodes))
	for _, node := range tree.Nodes {
		signals := make([]ResearchSignal, 0, len(node.Signals))
		var primary ResearchSignal
		secondary := make([]string, 0, len(node.Signals)-1)
		for _, signal := range node.Signals {
			item := ResearchSignal{
				SignalKey: signal.SignalKey, VariableName: signal.VariableName, Direction: signal.Direction,
				VariableSignalKey: signal.VariableSignalKey, SignalRole: signal.SignalRole,
				SignalDirection: signal.SignalDirection, DisplaySummary: signal.DisplaySummary,
				DisplayOrder: signal.DisplayOrder,
			}
			signals = append(signals, item)
			if signal.SignalRole == "primary" {
				primary = item
			} else {
				secondary = append(secondary, signal.DisplaySummary)
			}
		}
		var graphEdge *ResearchGraphEdge
		if node.IncomingGraphEdge != nil {
			graphEdge = &ResearchGraphEdge{
				ID: node.IncomingGraphEdge.ID, RelationType: node.IncomingGraphEdge.RelationType,
				ReviewStatus: node.IncomingGraphEdge.ReviewStatus, Status: node.IncomingGraphEdge.Status,
			}
		}
		nodes = append(nodes, ResearchReasoningTreeNode{
			NodeKey: node.NodeKey, DisplayName: node.DisplayName,
			ID: node.ID, Position: node.Position, ChainNodeEntityID: node.ChainNodeEntityID,
			Name: node.Name, StateSummary: node.StateSummary, ImpactDirection: node.ImpactDirection,
			ImpactStrength: node.ImpactStrength, ImpactSummary: node.ImpactSummary,
			ReasoningBasisSummary: node.ReasoningBasisSummary, EvidenceGapSummary: node.EvidenceGapSummary,
			IncomingIndustryChainGraphEdgeID: node.IncomingIndustryChainGraphEdgeID,
			IncomingTransmissionTitle:        node.IncomingTransmissionTitle,
			IncomingTransmissionMechanism:    node.IncomingTransmissionMechanism,
			IncomingConditionSummary:         node.IncomingConditionSummary, IncomingGraphEdge: graphEdge,
			Signals: signals, PrimarySignal: primary, SignalDisplaySummary: strings.Join(secondary, " · "),
		})
	}
	checkpoints := make([]ResearchCheckpoint, 0, len(tree.Checkpoints))
	for _, checkpoint := range tree.Checkpoints {
		checkpoints = append(checkpoints, ResearchCheckpoint{Type: checkpoint.Type, Summary: checkpoint.Summary})
	}
	return ResearchReasoningTreeDetail{
		ThemeID: result.ThemeID, ThemeKey: result.ThemeKey, PublicationMode: result.PublicationMode,
		PublicationContractVersion: result.PublicationContractVersion,
		ImpactNodeIDs:              append([]string(nil), result.ImpactNodeIDs...),
		ReasoningTree: ResearchReasoningTree{
			TreeKey: tree.TreeKey, DisplayName: tree.DisplayName,
			ReasoningTreeID: tree.ReasoningTreeID, ThemeID: tree.ThemeID,
			IndustryChainEntityID: tree.IndustryChainEntityID, IndustryChainName: tree.IndustryChainName,
			Title: tree.Title, DisplayOrder: tree.DisplayOrder, OneLineConclusion: tree.OneLineConclusion,
			FactSummary: tree.FactSummary, TransmissionSummary: tree.TransmissionSummary,
			ImpactDirection: tree.ImpactDirection, ImpactStrength: tree.ImpactStrength,
			ImpactSummary: tree.ImpactSummary, ConclusionBoundarySummary: tree.ConclusionBoundarySummary,
			SupportSummary: tree.SupportSummary, CounterSummary: tree.CounterSummary,
			InvalidationConditions: append([]string(nil), tree.InvalidationConditions...),
			Checkpoints:            checkpoints, PublishedAt: tree.PublishedAt.UTC(),
			EventCount: tree.EventCount, Events: eventDTOs(tree.Events), Nodes: nodes,
		},
	}, nil
}

func mapReasoningTreeRepositoryError(err error) error {
	switch {
	case errors.Is(err, ErrResearchThemeNotFound):
		return ErrThemeNotFound
	case errors.Is(err, ErrResearchReasoningTreesNotFound):
		return ErrReasoningTreesNotFound
	case errors.Is(err, ErrResearchReasoningTreeNotFound):
		return ErrReasoningTreeNotFound
	case errors.Is(err, ErrResearchReasoningTreeInvariant):
		return ErrReasoningTreeInvariantViolation
	default:
		return fmt.Errorf("%w: %v", ErrRepository, err)
	}
}

const (
	DefaultResearchWindowHours = 24
	MinResearchWindowHours     = 1
	MaxResearchWindowHours     = 168
	DefaultResearchLimit       = 20
	MaxResearchLimit           = 50
)

var (
	ErrInvalidRequest = errors.New("invalid research request")
	ErrRepository     = errors.New("research repository failure")
	ErrNotFound       = errors.New("research aggregate not found")
)

var researchUUIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

type ResearchListRequest struct {
	WindowHours   int
	PublishedFrom *time.Time
	PublishedTo   *time.Time
	Limit         int
	Cursor        string
}

type ResearchDetailRequest struct{ WindowHours int }

type ResearchThemePage struct {
	WindowStart time.Time       `json:"window_start"`
	WindowEnd   time.Time       `json:"window_end"`
	AsOf        time.Time       `json:"as_of"`
	ThemeCount  int             `json:"theme_count"`
	EventCount  int             `json:"event_count"`
	Items       []ResearchTheme `json:"items"`
	NextCursor  *string         `json:"next_cursor"`
}

type ResearchTheme struct {
	ID                        string                `json:"id"`
	AnalysisBatchID           string                `json:"analysis_batch_id"`
	Title                     string                `json:"title"`
	OneLineConclusion         string                `json:"one_line_conclusion"`
	ConclusionDirection       string                `json:"conclusion_direction"`
	ImpactStrength            string                `json:"impact_strength"`
	AttentionLevel            *string               `json:"attention_level"`
	ConclusionStatus          *string               `json:"conclusion_status"`
	TransmissionStage         string                `json:"transmission_stage"`
	InvestmentGuidanceAction  string                `json:"investment_guidance_action"`
	InvestmentGuidanceSummary string                `json:"investment_guidance_summary"`
	TimeHorizonCategory       string                `json:"time_horizon_category"`
	TimeHorizonSummary        *string               `json:"time_horizon_summary"`
	TransmissionSummary       *string               `json:"transmission_summary"`
	CheckpointSummary         *string               `json:"checkpoint_summary"`
	RiskSummary               *string               `json:"risk_summary"`
	AnalysisAsOf              time.Time             `json:"analysis_as_of"`
	WindowStart               time.Time             `json:"window_start"`
	WindowEnd                 time.Time             `json:"window_end"`
	PublishedAt               time.Time             `json:"published_at"`
	Impacts                   []ResearchThemeImpact `json:"impacts"`
	EvidenceEventCount        int                   `json:"evidence_event_count"`
	ReasoningTreeCount        int                   `json:"reasoning_tree_count"`
}

type ResearchThemeImpact struct {
	NodeKey           string  `json:"node_key"`
	DisplayName       string  `json:"display_name"`
	ChainNodeEntityID string  `json:"chain_node_entity_id"`
	Name              string  `json:"name"`
	RelationRole      string  `json:"relation_role"`
	ImpactDirection   string  `json:"impact_direction"`
	ImpactSummary     *string `json:"impact_summary"`
	DisplayOrder      int     `json:"display_order"`
}

type ResearchThemeDetail struct {
	ThemeKey                   string          `json:"theme_key"`
	PublicationMode            string          `json:"publication_mode"`
	PublicationContractVersion int             `json:"publication_contract_version"`
	Theme                      ResearchTheme   `json:"theme"`
	Events                     []ResearchEvent `json:"events"`
}

type ResearchEvent struct {
	EventID        string     `json:"event_id"`
	EvidenceIDs    []string   `json:"evidence_ids"`
	Title          string     `json:"title"`
	Summary        string     `json:"summary"`
	EventTime      *time.Time `json:"event_time"`
	EvidenceRole   string     `json:"evidence_role"`
	SupportedClaim *string    `json:"supported_claim,omitempty"`
	DisplayOrder   int        `json:"display_order,omitempty"`
}

type UseCase struct {
	repository       Repository
	publicationStore PublicationStore
	eventProvider    EventProvider
	semanticProvider SemanticProvider
	entityProvider   EntityProvider
	graphStore       GraphStore
	now              func() time.Time
}

func NewUseCase(
	repository Repository,
	publicationStore PublicationStore,
	eventProvider EventProvider,
	semanticProvider SemanticProvider,
	entityProvider EntityProvider,
	now func() time.Time,
) (*UseCase, error) {
	if repository == nil || publicationStore == nil || eventProvider == nil || semanticProvider == nil || entityProvider == nil {
		return nil, errors.New("Research use case dependencies are required")
	}
	if now == nil {
		now = time.Now
	}
	return &UseCase{
		repository:       repository,
		publicationStore: publicationStore,
		eventProvider:    eventProvider,
		semanticProvider: semanticProvider,
		entityProvider:   entityProvider,
		graphStore:       entityProvider,
		now:              now,
	}, nil
}

func (s *UseCase) ListThemes(ctx context.Context, request ResearchListRequest) (ResearchThemePage, error) {
	windowHours, limit, explicitRange, err := normalizeListRequest(request)
	if err != nil {
		return ResearchThemePage{}, err
	}
	asOf, windowStart, windowEnd, cursor, err := s.prepareCursor("themes", request, windowHours, explicitRange)
	if err != nil {
		return ResearchThemePage{}, err
	}
	page, err := s.repository.ListResearchThemes(ctx, ThemeListFilter{
		WindowStart: windowStart, WindowEnd: windowEnd, AsOf: asOf, Limit: limit,
		CursorPublishedAt: cursor.PublishedAtPtr(), CursorID: cursor.ID,
	})
	if err != nil {
		return ResearchThemePage{}, mapRepositoryError(err)
	}
	response := ResearchThemePage{
		WindowStart: page.WindowStart.UTC(), WindowEnd: page.WindowEnd.UTC(), AsOf: page.AsOf.UTC(),
		ThemeCount: page.ThemeCount, EventCount: page.EventCount,
		Items: make([]ResearchTheme, 0, len(page.Items)),
	}
	for _, item := range page.Items {
		response.Items = append(response.Items, themeDTO(item))
	}
	if page.HasMore && len(page.Items) > 0 {
		last := page.Items[len(page.Items)-1]
		nextCursor := researchCursor{
			Version: 1, Kind: "themes", WindowHours: windowHours, AsOf: asOf,
			PublishedAt: last.PublishedAt, ID: last.ID,
		}
		if explicitRange {
			nextCursor.Version = 2
			nextCursor.WindowStart = windowStart
			nextCursor.WindowEnd = windowEnd
		}
		next, err := encodeCursor(nextCursor)
		if err != nil {
			return ResearchThemePage{}, fmt.Errorf("encode research cursor: %w", err)
		}
		response.NextCursor = &next
	}
	return response, nil
}

func (s *UseCase) GetTheme(ctx context.Context, id string, request ResearchDetailRequest) (ResearchThemeDetail, error) {
	if _, err := normalizeDetailRequest(request); err != nil {
		return ResearchThemeDetail{}, err
	}
	if !researchUUIDPattern.MatchString(strings.TrimSpace(id)) {
		return ResearchThemeDetail{}, fmt.Errorf("%w: theme id must be a UUID", ErrInvalidRequest)
	}
	item, err := s.repository.GetResearchTheme(ctx, id)
	if err != nil {
		return ResearchThemeDetail{}, mapRepositoryError(err)
	}
	return ResearchThemeDetail{
		ThemeKey: item.ThemeKey, PublicationMode: item.PublicationMode,
		PublicationContractVersion: item.PublicationContractVersion,
		Theme:                      themeDTO(item.ThemeSummaryRecord), Events: eventDTOs(item.Events),
	}, nil
}

func normalizeListRequest(request ResearchListRequest) (int, int, bool, error) {
	explicitRange := request.PublishedFrom != nil || request.PublishedTo != nil
	if explicitRange && (request.PublishedFrom == nil || request.PublishedTo == nil) {
		return 0, 0, false, fmt.Errorf("%w: published_from and published_to must be provided together", ErrInvalidRequest)
	}
	if explicitRange && request.WindowHours != 0 {
		return 0, 0, false, fmt.Errorf("%w: window_hours cannot be combined with published_from and published_to", ErrInvalidRequest)
	}
	windowHours := request.WindowHours
	if windowHours == 0 && !explicitRange {
		windowHours = DefaultResearchWindowHours
	}
	if !explicitRange && (windowHours < MinResearchWindowHours || windowHours > MaxResearchWindowHours) {
		return 0, 0, false, fmt.Errorf("%w: window_hours must be between %d and %d", ErrInvalidRequest, MinResearchWindowHours, MaxResearchWindowHours)
	}
	if explicitRange && !request.PublishedFrom.Before(*request.PublishedTo) {
		return 0, 0, false, fmt.Errorf("%w: published_from must be before published_to", ErrInvalidRequest)
	}
	limit := request.Limit
	if limit == 0 {
		limit = DefaultResearchLimit
	}
	if limit < 1 || limit > MaxResearchLimit {
		return 0, 0, false, fmt.Errorf("%w: limit must be between 1 and %d", ErrInvalidRequest, MaxResearchLimit)
	}
	return windowHours, limit, explicitRange, nil
}

func normalizeDetailRequest(request ResearchDetailRequest) (int, error) {
	windowHours := request.WindowHours
	if windowHours == 0 {
		windowHours = DefaultResearchWindowHours
	}
	if windowHours < MinResearchWindowHours || windowHours > MaxResearchWindowHours {
		return 0, fmt.Errorf("%w: window_hours must be between %d and %d", ErrInvalidRequest, MinResearchWindowHours, MaxResearchWindowHours)
	}
	return windowHours, nil
}

type researchCursor struct {
	Version     int       `json:"v"`
	Kind        string    `json:"kind"`
	WindowHours int       `json:"window_hours"`
	AsOf        time.Time `json:"as_of"`
	WindowStart time.Time `json:"window_start,omitempty"`
	WindowEnd   time.Time `json:"window_end,omitempty"`
	PublishedAt time.Time `json:"published_at"`
	ID          string    `json:"id"`
}

func (c researchCursor) PublishedAtPtr() *time.Time {
	if c.ID == "" {
		return nil
	}
	value := c.PublishedAt
	return &value
}

func (s *UseCase) prepareCursor(kind string, request ResearchListRequest, windowHours int, explicitRange bool) (time.Time, time.Time, time.Time, researchCursor, error) {
	if strings.TrimSpace(request.Cursor) == "" {
		asOf := s.now().UTC()
		if explicitRange {
			return asOf, request.PublishedFrom.UTC(), request.PublishedTo.UTC(), researchCursor{}, nil
		}
		return asOf, asOf.Add(-time.Duration(windowHours) * time.Hour), asOf, researchCursor{}, nil
	}
	cursor, err := decodeResearchCursor(request.Cursor)
	if err != nil || cursor.Kind != kind || cursor.ID == "" {
		return time.Time{}, time.Time{}, time.Time{}, researchCursor{}, fmt.Errorf("%w: invalid cursor", ErrInvalidRequest)
	}
	if explicitRange {
		if cursor.Version != 2 || !cursor.WindowStart.Equal(request.PublishedFrom.UTC()) || !cursor.WindowEnd.Equal(request.PublishedTo.UTC()) {
			return time.Time{}, time.Time{}, time.Time{}, researchCursor{}, fmt.Errorf("%w: invalid cursor", ErrInvalidRequest)
		}
		return cursor.AsOf.UTC(), cursor.WindowStart.UTC(), cursor.WindowEnd.UTC(), cursor, nil
	}
	if cursor.Version != 1 || cursor.WindowHours != windowHours {
		return time.Time{}, time.Time{}, time.Time{}, researchCursor{}, fmt.Errorf("%w: invalid cursor", ErrInvalidRequest)
	}
	asOf := cursor.AsOf.UTC()
	return asOf, asOf.Add(-time.Duration(windowHours) * time.Hour), asOf, cursor, nil
}

func decodeResearchCursor(value string) (researchCursor, error) {
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return researchCursor{}, err
	}
	var cursor researchCursor
	if err := json.Unmarshal(payload, &cursor); err != nil {
		return researchCursor{}, err
	}
	return cursor, nil
}

func themeDTO(item ThemeSummaryRecord) ResearchTheme {
	return ResearchTheme{
		ID: item.ID, AnalysisBatchID: item.AnalysisBatchID, Title: item.Title,
		OneLineConclusion: item.OneLineConclusion, ConclusionDirection: item.ConclusionDirection,
		ImpactStrength: item.ImpactStrength, AttentionLevel: item.AttentionLevel,
		ConclusionStatus: item.ConclusionStatus, TransmissionStage: item.TransmissionStage,
		InvestmentGuidanceAction:  item.InvestmentGuidanceAction,
		InvestmentGuidanceSummary: item.InvestmentGuidanceSummary,
		TimeHorizonCategory:       item.TimeHorizonCategory, TimeHorizonSummary: item.TimeHorizonSummary,
		TransmissionSummary: item.TransmissionSummary, CheckpointSummary: item.CheckpointSummary,
		RiskSummary: item.RiskSummary, AnalysisAsOf: item.AnalysisAsOf.UTC(),
		WindowStart: item.WindowStart.UTC(), WindowEnd: item.WindowEnd.UTC(),
		PublishedAt: item.PublishedAt.UTC(), Impacts: impactDTOs(item.Impacts),
		EvidenceEventCount: item.EvidenceEventCount, ReasoningTreeCount: item.ReasoningTreeCount,
	}
}

func impactDTOs(values []ThemeImpactRecord) []ResearchThemeImpact {
	result := make([]ResearchThemeImpact, 0, len(values))
	for _, value := range values {
		result = append(result, ResearchThemeImpact{
			NodeKey: value.NodeKey, DisplayName: value.DisplayName,
			ChainNodeEntityID: value.ChainNodeEntityID, Name: value.Name,
			RelationRole: value.RelationRole, ImpactDirection: value.ImpactDirection,
			ImpactSummary: value.ImpactSummary, DisplayOrder: value.DisplayOrder,
		})
	}
	return result
}

func eventDTOs(values []EventRecord) []ResearchEvent {
	result := make([]ResearchEvent, 0, len(values))
	for index, value := range values {
		var eventTime *time.Time
		if value.EventTime != nil {
			formatted := value.EventTime.UTC()
			eventTime = &formatted
		}
		displayOrder := value.DisplayOrder
		if displayOrder == 0 {
			displayOrder = index + 1
		}
		result = append(result, ResearchEvent{
			EvidenceIDs: append([]string(nil), value.EvidenceIDs...),
			EventID:     value.EventID, Title: value.Title, Summary: value.Summary,
			EventTime: eventTime, EvidenceRole: value.EvidenceRole,
			SupportedClaim: value.SupportedClaim, DisplayOrder: displayOrder,
		})
	}
	return result
}

func mapRepositoryError(err error) error {
	switch {
	case errors.Is(err, ErrResearchNotFound), errors.Is(err, ErrResearchThemeNotFound):
		return ErrNotFound
	default:
		return fmt.Errorf("%w: %v", ErrRepository, err)
	}
}

var (
	ErrPayloadConflict   = errors.New("analysis batch conflicts with the published aggregate")
	ErrPublisherConflict = errors.New("analysis batch belongs to another publisher subject")
)

type Result struct {
	ReceiptID, AnalysisBatchID, PayloadHash, ThemeID string
	PublicationMode                                  string
	ReasoningTreeIDsByIndustryChainEntityID          map[string]string
	ReasoningTreeIDsByTreeKey                        map[string]string
	Counts                                           Counts
	PublishedAt, ImportedAt                          time.Time
	Replayed                                         bool
}

func (s *UseCase) Publish(ctx context.Context, publisher string, aggregate Aggregate) (Result, error) {
	if s == nil || s.publicationStore == nil {
		return Result{}, errors.New("research publication store is required")
	}
	publisher = strings.TrimSpace(publisher)
	if publisher == "" || len(publisher) > 200 {
		return Result{}, errors.New("publisher subject must contain 1..200 characters")
	}
	analysisAsOf, themeID, err := aggregate.Validate()
	if err != nil {
		return Result{}, err
	}
	payloadHash, err := CanonicalHash(aggregate)
	if err != nil {
		return Result{}, fmt.Errorf("hash research publication: %w", err)
	}
	plan := publicationPlan(aggregate, themeID, payloadHash)
	var result Result
	err = s.publicationStore.InResearchPublicationTransaction(ctx, func(tx PublicationTransaction) error {
		if err := tx.Lock(ctx, aggregate.AnalysisBatchID); err != nil {
			return fmt.Errorf("lock research publication: %w", err)
		}
		existing, err := tx.Receipt(ctx, aggregate.AnalysisBatchID)
		if err != nil {
			return fmt.Errorf("load research publication receipt: %w", err)
		}
		if existing != nil {
			if existing.ContractVersion != 2 {
				return ErrPayloadConflict
			}
			if existing.PublisherSubject != publisher {
				return ErrPublisherConflict
			}
			if existing.PayloadHash != payloadHash {
				return ErrPayloadConflict
			}
			if existing.ThemeID != plan.ThemeID ||
				!reflect.DeepEqual(existing.ReasoningTreeIDsByIndustryChainEntityID, plan.ReasoningTreeIDsByIndustryChainEntityID) ||
				existing.Counts != plan.Counts {
				return errors.New("research publication receipt does not match deterministic plan")
			}
			if err := tx.Verify(ctx, *existing); err != nil {
				return fmt.Errorf("verify research publication replay: %w", err)
			}
			result = resultFromReceipt(*existing, true)
			return nil
		}

		query := referenceQuery(aggregate)
		facts, err := tx.ReferenceFacts(ctx, query)
		if err != nil {
			return fmt.Errorf("load research publication references: %w", err)
		}
		if err := validateReferences(aggregate, analysisAsOf, facts); err != nil {
			return err
		}

		publishedAt := s.now().UTC().Truncate(time.Microsecond)
		receipt := plan
		receipt.PublisherSubject = publisher
		receipt.PublishedAt, receipt.ImportedAt = publishedAt, publishedAt
		if err := tx.InsertThemeReceipt(ctx, receipt); err != nil {
			return fmt.Errorf("insert aggregate receipt: %w", err)
		}
		windowStart, _ := time.Parse(time.RFC3339, aggregate.DiscoveryWindowStart)
		windowEnd, _ := time.Parse(time.RFC3339, aggregate.DiscoveryWindowEnd)
		theme := aggregate.Theme
		if err := tx.InsertTheme(ctx, PublicationThemeRecord{
			ID: themeID, ImportReceiptID: receipt.ID, AnalysisBatchID: aggregate.AnalysisBatchID,
			ThemeKey: theme.ThemeKey, Title: theme.Title, OneLineConclusion: theme.OneLineConclusion,
			ConclusionDirection: theme.ConclusionDirection, ImpactStrength: theme.ImpactStrength,
			AttentionLevel: theme.AttentionLevel, ConclusionStatus: theme.ConclusionStatus,
			TransmissionStage:         theme.TransmissionStage,
			InvestmentGuidanceAction:  theme.InvestmentGuidanceAction,
			InvestmentGuidanceSummary: theme.InvestmentGuidanceSummary,
			TimeHorizonCategory:       theme.TimeHorizonCategory, TimeHorizonSummary: theme.TimeHorizonSummary,
			TransmissionSummary: theme.TransmissionSummary, CheckpointSummary: theme.CheckpointSummary,
			RiskSummary: theme.RiskSummary, AnalysisAsOf: analysisAsOf,
			WindowStart: windowStart, WindowEnd: windowEnd, PublishedAt: publishedAt,
		}); err != nil {
			return fmt.Errorf("insert Theme: %w", err)
		}
		for _, impact := range theme.Impacts {
			if err := tx.InsertThemeImpact(ctx, PublicationThemeImpactRecord{
				ThemeID: themeID, ChainNodeEntityID: impact.ChainNodeEntityID,
				RelationRole: impact.RelationRole, ImpactDirection: impact.ImpactDirection,
				ImpactSummary: impact.ImpactSummary, DisplayOrder: impact.DisplayOrder,
			}); err != nil {
				return fmt.Errorf("insert Theme Impact: %w", err)
			}
		}
		for _, event := range theme.Events {
			if err := tx.InsertThemeEvent(ctx, PublicationThemeEventRecord{
				ThemeID: themeID, EventID: event.EventID, EvidenceRole: event.EvidenceRole,
				SupportedClaim: event.SupportedClaim,
			}); err != nil {
				return fmt.Errorf("insert Theme Event: %w", err)
			}
		}

		treeReceipt := ReasonTreeReceipt{
			ID:      identity.NormalizeUUID("research_reasoning_tree_import_receipt", themeID),
			ThemeID: themeID, PublisherSubject: publisher, PayloadHash: payloadHash,
			ReasoningTreeIDsByIndustryChainEntityID: cloneMap(plan.ReasoningTreeIDsByIndustryChainEntityID),
			Counts: ReasonTreeCounts{
				ReasoningTrees: plan.Counts.ReasoningTrees, Nodes: plan.Counts.Nodes,
				EventAssociations:  plan.Counts.TreeEventAssociations,
				SignalAssociations: plan.Counts.SignalAssociations, Receipts: 1,
			},
			PublishedAt: publishedAt, ImportedAt: publishedAt,
		}
		if err := tx.InsertTreeReceipt(ctx, treeReceipt); err != nil {
			return fmt.Errorf("insert Reason Tree receipt: %w", err)
		}
		for _, tree := range aggregate.ReasoningTrees {
			treeID := plan.ReasoningTreeIDsByIndustryChainEntityID[tree.IndustryChainEntityID]
			if err := tx.InsertTree(ctx, ReasonTreeRecord{
				ID: treeID, ThemeID: themeID, ImportReceiptID: treeReceipt.ID,
				IndustryChainEntityID: tree.IndustryChainEntityID, Title: tree.Title,
				DisplayOrder: tree.DisplayOrder, OneLineConclusion: tree.OneLineConclusion,
				FactSummary: tree.FactSummary, TransmissionSummary: tree.TransmissionSummary,
				ImpactDirection: tree.ImpactDirection, ImpactStrength: tree.ImpactStrength,
				ImpactSummary: tree.ImpactSummary, ConclusionBoundarySummary: tree.ConclusionBoundarySummary,
				SupportSummary: tree.SupportSummary, CounterSummary: tree.CounterSummary,
				InvalidationConditions: append([]string(nil), tree.InvalidationConditions...),
				Checkpoints:            append([]ReasonTreeCheckpoint(nil), tree.Checkpoints...),
			}); err != nil {
				return fmt.Errorf("insert Reason Tree: %w", err)
			}
			for _, event := range tree.Events {
				if err := tx.InsertTreeEvent(ctx, ReasonTreeEventRecord{
					ReasoningTreeID: treeID, EventID: event.EventID,
					EvidenceRole: event.EvidenceRole, DisplayOrder: event.DisplayOrder,
				}); err != nil {
					return fmt.Errorf("insert Reason Tree Event: %w", err)
				}
			}
			for _, node := range tree.Nodes {
				nodeID := publicationReasonTreeNodeID(treeID, node.Position, node.ChainNodeEntityID)
				record := NodeRecord{ReasonTreeNodeRecord: ReasonTreeNodeRecord{
					ID: nodeID, ReasoningTreeID: treeID, Position: node.Position,
					ChainNodeEntityID: node.ChainNodeEntityID, StateSummary: node.StateSummary,
					ImpactDirection: node.ImpactDirection, ImpactStrength: node.ImpactStrength,
					ImpactSummary: node.ImpactSummary, ReasoningBasisSummary: node.ReasoningBasisSummary,
					EvidenceGapSummary:               node.EvidenceGapSummary,
					IncomingIndustryChainGraphEdgeID: node.IncomingIndustryChainGraphEdgeID,
					IncomingTransmissionTitle:        node.IncomingTransmissionTitle,
					IncomingTransmissionMechanism:    node.IncomingTransmissionMechanism,
					IncomingConditionSummary:         node.IncomingConditionSummary,
				}}
				if node.IncomingLineage != nil {
					lineage := node.IncomingLineage
					record.IncomingSourceKind = &lineage.SourceKind
					record.DirectImpactAssertionID = lineage.DirectImpactAssertionID
					record.DirectImpactSemanticSubmissionID = lineage.SemanticSubmissionID
					record.DirectImpactEvidenceID = lineage.EvidenceID
					record.DirectImpactEvidenceHash = lineage.EvidenceHash
					record.DirectImpactAffectedVariableKey = lineage.AffectedVariableKey
					record.DirectImpactAffectedDirection = lineage.AffectedDirection
					record.InferenceUpstreamVariableSignalID = lineage.UpstreamVariableSignalID
					record.InferenceUpstreamDirectImpactAssertionID = lineage.UpstreamDirectImpactAssertionID
					record.InferenceEntityRelationID = lineage.EntityRelationID
				}
				if err := tx.InsertNode(ctx, record); err != nil {
					return fmt.Errorf("insert Reason Tree Node: %w", err)
				}
				for _, signal := range node.Signals {
					if err := tx.InsertSignal(ctx, PublicationSignalRecord{
						ReasonTreeSignalRecord: ReasonTreeSignalRecord{
							ReasoningTreeNodeID: nodeID, VariableSignalKey: signal.VariableSignalKey,
							SignalRole: signal.SignalRole, SignalDirection: signal.SignalDirection,
							DisplaySummary: signal.DisplaySummary, DisplayOrder: signal.DisplayOrder,
						},
						SourceKind:           signal.Lineage.SourceKind,
						VariableSignalID:     signal.Lineage.VariableSignalID,
						SemanticSubmissionID: signal.Lineage.SemanticSubmissionID,
						EvidenceID:           signal.Lineage.EvidenceID, EvidenceHash: signal.Lineage.EvidenceHash,
						UpstreamVariableSignalID:        signal.Lineage.UpstreamVariableSignalID,
						UpstreamDirectImpactAssertionID: signal.Lineage.UpstreamDirectImpactAssertionID,
						EntityRelationID:                signal.Lineage.EntityRelationID,
						IndustryChainGraphEdgeID:        signal.Lineage.IndustryChainGraphEdgeID,
					}); err != nil {
						return fmt.Errorf("insert Reason Tree Signal: %w", err)
					}
				}
			}
		}
		if err := tx.Verify(ctx, receipt); err != nil {
			return fmt.Errorf("verify research publication: %w", err)
		}
		result = resultFromReceipt(receipt, false)
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

func publicationPlan(a Aggregate, themeID, payloadHash string) Receipt {
	treeIDs := make(map[string]string, len(a.ReasoningTrees))
	counts := Counts{Themes: 1, Impacts: len(a.Theme.Impacts), ThemeEventAssociations: len(a.Theme.Events), Receipts: 2}
	for _, tree := range a.ReasoningTrees {
		treeIDs[tree.IndustryChainEntityID] = publicationReasonTreeID(themeID, tree.IndustryChainEntityID)
		counts.ReasoningTrees++
		counts.TreeEventAssociations += len(tree.Events)
		counts.Nodes += len(tree.Nodes)
		for _, node := range tree.Nodes {
			counts.SignalAssociations += len(node.Signals)
		}
	}
	return Receipt{
		ID:              identity.NormalizeUUID("research_theme_import_receipt", a.AnalysisBatchID),
		AnalysisBatchID: a.AnalysisBatchID, PayloadHash: payloadHash, ThemeID: themeID,
		ThemeKey: a.Theme.ThemeKey, ContractVersion: 2, PublicationMode: "formal",
		ReasoningTreeIDsByIndustryChainEntityID: treeIDs, Counts: counts,
	}
}

func referenceQuery(a Aggregate) ReferenceQuery {
	q := ReferenceQuery{}
	for _, impact := range a.Theme.Impacts {
		q.ChainNodeIDs = append(q.ChainNodeIDs, impact.ChainNodeEntityID)
	}
	for _, event := range a.Theme.Events {
		q.EventIDs = append(q.EventIDs, event.EventID)
	}
	for _, tree := range a.ReasoningTrees {
		q.IndustryChainIDs = append(q.IndustryChainIDs, tree.IndustryChainEntityID)
		for _, event := range tree.Events {
			q.EventIDs = append(q.EventIDs, event.EventID)
		}
		for _, node := range tree.Nodes {
			q.ChainNodeIDs = append(q.ChainNodeIDs, node.ChainNodeEntityID)
			if node.IncomingIndustryChainGraphEdgeID != nil {
				q.GraphEdgeIDs = append(q.GraphEdgeIDs, *node.IncomingIndustryChainGraphEdgeID)
			}
			if l := node.IncomingLineage; l != nil {
				appendLineageQuery(&q, l.DirectImpactAssertionID, l.UpstreamVariableSignalID, l.UpstreamDirectImpactAssertionID, l.EvidenceID, l.EntityRelationID)
			}
			for _, signal := range node.Signals {
				l := signal.Lineage
				appendLineageQuery(&q, nil, choose(l.VariableSignalID, l.UpstreamVariableSignalID), l.UpstreamDirectImpactAssertionID, l.EvidenceID, l.EntityRelationID)
				if l.IndustryChainGraphEdgeID != nil {
					q.GraphEdgeIDs = append(q.GraphEdgeIDs, *l.IndustryChainGraphEdgeID)
				}
			}
		}
	}
	return q
}

func appendLineageQuery(q *ReferenceQuery, impact, signal, upstreamImpact, evidence, relation *string) {
	if impact != nil {
		q.ImpactIDs = append(q.ImpactIDs, *impact)
	}
	if signal != nil {
		q.SignalIDs = append(q.SignalIDs, *signal)
	}
	if upstreamImpact != nil {
		q.ImpactIDs = append(q.ImpactIDs, *upstreamImpact)
	}
	if evidence != nil {
		q.EvidenceIDs = append(q.EvidenceIDs, *evidence)
	}
	if relation != nil {
		q.EntityRelationIDs = append(q.EntityRelationIDs, *relation)
	}
}

func choose(first, second *string) *string {
	if first != nil {
		return first
	}
	return second
}

func validateReferences(a Aggregate, asOf time.Time, facts ReferenceFacts) error {
	windowStart, _ := time.Parse(time.RFC3339, a.DiscoveryWindowStart)
	windowEnd, _ := time.Parse(time.RFC3339, a.DiscoveryWindowEnd)
	for index, impact := range a.Theme.Impacts {
		temporal, ok := facts.ChainNodeIDs[impact.ChainNodeEntityID]
		if !ok || !temporalFactAvailableAt(temporal, asOf) {
			return invalidReference(fmt.Sprintf("theme.impacts[%d].chain_node_entity_id", index), impact.ChainNodeEntityID, "active approved Chain Node does not exist")
		}
	}
	for index, event := range a.Theme.Events {
		fact, ok := facts.Events[event.EventID]
		if !ok {
			return invalidReference(fmt.Sprintf("theme.events[%d].event_id", index), event.EventID, "confirmed verified Event does not exist")
		}
		if fact.KnowledgeAvailableAt.Before(windowStart) ||
			!fact.KnowledgeAvailableAt.Before(windowEnd) ||
			fact.KnowledgeAvailableAt.After(asOf) {
			return invalidReference(
				fmt.Sprintf("theme.events[%d].event_id", index), event.EventID,
				"Event was not knowable inside the declared discovery window by analysis_as_of",
			)
		}
	}
	for treeIndex, tree := range a.ReasoningTrees {
		treePath := fmt.Sprintf("reasoning_trees[%d]", treeIndex)
		chainTemporal, ok := facts.IndustryChainIDs[tree.IndustryChainEntityID]
		if !ok || !temporalFactAvailableAt(chainTemporal, asOf) {
			return invalidReference(treePath+".industry_chain_entity_id", tree.IndustryChainEntityID, "active approved Industry Chain does not exist")
		}
		treeEvents := make(map[string]struct{}, len(tree.Events))
		for _, event := range tree.Events {
			treeEvents[event.EventID] = struct{}{}
		}
		for nodeIndex, node := range tree.Nodes {
			nodePath := fmt.Sprintf("%s.nodes[%d]", treePath, nodeIndex)
			step := inferenceStep{
				CurrentNodeEntityID:   node.ChainNodeEntityID,
				IndustryChainEntityID: tree.IndustryChainEntityID,
			}
			if nodeIndex > 0 {
				step.PreviousNodeEntityID = &tree.Nodes[nodeIndex-1].ChainNodeEntityID
			}
			membership, ok := facts.Memberships[tree.IndustryChainEntityID][node.ChainNodeEntityID]
			if !ok || !temporalFactAvailableAt(membership, asOf) {
				return invalidReference(nodePath+".chain_node_entity_id", node.ChainNodeEntityID, "Node is not an active approved member of the Industry Chain")
			}
			if node.IncomingIndustryChainGraphEdgeID != nil {
				edge, ok := facts.GraphEdges[*node.IncomingIndustryChainGraphEdgeID]
				if !ok || !temporalFactAvailableAt(edge.TemporalFact, asOf) ||
					nodeIndex == 0 || edge.IndustryChainEntityID != tree.IndustryChainEntityID ||
					!connectsEntities(
						edge.FromChainNodeEntityID,
						edge.ToChainNodeEntityID,
						tree.Nodes[nodeIndex-1].ChainNodeEntityID,
						node.ChainNodeEntityID,
					) {
					return invalidReference(nodePath+".incoming_industry_chain_graph_edge_id", *node.IncomingIndustryChainGraphEdgeID, "Graph Edge does not connect the adjacent Tree Nodes")
				}
			}
			if node.IncomingLineage != nil {
				if err := validateIncomingLineage(
					nodePath+".incoming_lineage",
					*node.IncomingLineage,
					step,
					node.IncomingIndustryChainGraphEdgeID,
					treeEvents,
					asOf,
					facts,
				); err != nil {
					return err
				}
			}
			for signalIndex, signal := range node.Signals {
				if err := validateSignalLineage(
					fmt.Sprintf("%s.signals[%d].lineage", nodePath, signalIndex),
					signal,
					step,
					treeEvents,
					asOf,
					facts,
				); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

type inferenceStep struct {
	PreviousNodeEntityID  *string
	CurrentNodeEntityID   string
	IndustryChainEntityID string
}

func validateSignalLineage(
	path string,
	input Signal,
	step inferenceStep,
	treeEvents map[string]struct{},
	asOf time.Time,
	facts ReferenceFacts,
) error {
	l := input.Lineage
	if l.SourceKind == "formal_signal" {
		fact, ok := facts.Signals[*l.VariableSignalID]
		if !ok {
			return invalidReference(path+".variable_signal_id", *l.VariableSignalID, "accepted latest Signal does not exist")
		}
		if fact.SemanticSubmissionID != *l.SemanticSubmissionID {
			return invalidReference(path+".semantic_submission_id", *l.SemanticSubmissionID, "does not own the Signal")
		}
		if fact.SubjectEntityID != step.CurrentNodeEntityID {
			return invalidReference(path+".variable_signal_id", fact.ID, "Signal subject must equal the Node entity")
		}
		if fact.VariableKey != input.VariableSignalKey || fact.Direction != input.SignalDirection {
			return invalidReference(path+".variable_signal_id", fact.ID, "Signal display snapshot does not match the formal Signal")
		}
		if _, ok := treeEvents[fact.EventID]; !ok {
			return invalidReference(path+".variable_signal_id", fact.ID, "Signal source Event is not covered by Reason Tree events")
		}
		if err := validateEvidence(path, *l.EvidenceID, *l.EvidenceHash, fact.EventID, fact.EvidenceIDs, asOf, facts); err != nil {
			return err
		}
		if fact.AcceptedAt.After(asOf) {
			return invalidReference(path+".variable_signal_id", fact.ID, "Signal was not accepted by analysis_as_of")
		}
		return nil
	}
	return validateInference(
		path, l.UpstreamVariableSignalID, l.UpstreamDirectImpactAssertionID,
		l.EntityRelationID, l.IndustryChainGraphEdgeID,
		step,
		treeEvents, asOf, facts,
	)
}

func validateIncomingLineage(
	path string,
	l IncomingLineage,
	step inferenceStep,
	graphEdgeID *string,
	treeEvents map[string]struct{},
	asOf time.Time,
	facts ReferenceFacts,
) error {
	if l.SourceKind == "formal_direct_impact" {
		fact, ok := facts.Impacts[*l.DirectImpactAssertionID]
		if !ok {
			return invalidReference(path+".direct_impact_assertion_id", *l.DirectImpactAssertionID, "accepted latest Direct Impact does not exist")
		}
		if fact.SemanticSubmissionID != *l.SemanticSubmissionID {
			return invalidReference(path+".semantic_submission_id", *l.SemanticSubmissionID, "does not own the Direct Impact")
		}
		if fact.TargetEntityID != step.CurrentNodeEntityID {
			return invalidReference(path+".direct_impact_assertion_id", fact.ID, "Direct Impact target must equal the downstream Node")
		}
		if step.PreviousNodeEntityID == nil || fact.SourceEntityID != *step.PreviousNodeEntityID {
			return invalidReference(path+".direct_impact_assertion_id", fact.ID, "Direct Impact source Signal subject must equal the previous Node")
		}
		if fact.AffectedVariableKey != *l.AffectedVariableKey || fact.AffectedDirection != *l.AffectedDirection {
			return invalidReference(path+".direct_impact_assertion_id", fact.ID, "affected-variable snapshot does not match the formal Direct Impact")
		}
		if _, ok := treeEvents[fact.SourceEventID]; !ok {
			return invalidReference(path+".direct_impact_assertion_id", fact.ID, "Direct Impact source Event is not covered by Reason Tree events")
		}
		if err := validateEvidence(path, *l.EvidenceID, *l.EvidenceHash, fact.SourceEventID, fact.EvidenceIDs, asOf, facts); err != nil {
			return err
		}
		if fact.AcceptedAt.After(asOf) {
			return invalidReference(path+".direct_impact_assertion_id", fact.ID, "Direct Impact was not accepted by analysis_as_of")
		}
		return nil
	}
	return validateInference(
		path, l.UpstreamVariableSignalID, l.UpstreamDirectImpactAssertionID,
		l.EntityRelationID, graphEdgeID,
		step,
		treeEvents, asOf, facts,
	)
}

func validateInference(
	path string,
	signalID, impactID, entityRelationID, graphEdgeID *string,
	step inferenceStep,
	treeEvents map[string]struct{},
	asOf time.Time,
	facts ReferenceFacts,
) error {
	upstreamEntityIDs := make(map[string]struct{}, 2)
	if signalID != nil {
		fact, ok := facts.Signals[*signalID]
		if !ok || fact.AcceptedAt.After(asOf) {
			return invalidReference(path+".upstream_variable_signal_id", *signalID, "accepted latest upstream Signal does not exist at analysis_as_of")
		}
		if _, ok := treeEvents[fact.EventID]; !ok {
			return invalidReference(path+".upstream_variable_signal_id", *signalID, "upstream Signal source Event is not covered by Reason Tree events")
		}
		upstreamEntityIDs[fact.SubjectEntityID] = struct{}{}
	}
	if impactID != nil {
		fact, ok := facts.Impacts[*impactID]
		if !ok || fact.AcceptedAt.After(asOf) {
			return invalidReference(path+".upstream_direct_impact_assertion_id", *impactID, "accepted latest upstream Direct Impact does not exist at analysis_as_of")
		}
		if _, ok := treeEvents[fact.SourceEventID]; !ok {
			return invalidReference(path+".upstream_direct_impact_assertion_id", *impactID, "upstream Direct Impact source Event is not covered by Reason Tree events")
		}
		upstreamEntityIDs[fact.SourceEntityID] = struct{}{}
		upstreamEntityIDs[fact.TargetEntityID] = struct{}{}
	}
	if entityRelationID != nil {
		relation, ok := facts.EntityRelations[*entityRelationID]
		connectsStep := relationConnectsInferenceStep(
			relation.FromEntityID,
			relation.ToEntityID,
			step,
			upstreamEntityIDs,
		)
		if !ok || !temporalFactAvailableAt(relation.TemporalFact, asOf) ||
			!connectsStep {
			return invalidReference(path+".entity_relation_id", *entityRelationID, "active Entity Relation does not connect the adjacent Tree Nodes")
		}
	}
	if graphEdgeID != nil {
		edge, ok := facts.GraphEdges[*graphEdgeID]
		connectsStep := relationConnectsInferenceStep(
			edge.FromChainNodeEntityID,
			edge.ToChainNodeEntityID,
			step,
			upstreamEntityIDs,
		)
		if !ok || !temporalFactAvailableAt(edge.TemporalFact, asOf) ||
			edge.IndustryChainEntityID != step.IndustryChainEntityID || !connectsStep {
			return invalidReference(path+".industry_chain_graph_edge_id", *graphEdgeID, "active approved Industry Chain Graph Edge does not connect the adjacent Tree Nodes")
		}
	}
	return nil
}

func relationConnectsInferenceStep(
	fromEntityID, toEntityID string,
	step inferenceStep,
	upstreamEntityIDs map[string]struct{},
) bool {
	if step.PreviousNodeEntityID != nil {
		return connectsEntities(fromEntityID, toEntityID, *step.PreviousNodeEntityID, step.CurrentNodeEntityID)
	}
	for upstreamEntityID := range upstreamEntityIDs {
		if connectsEntities(fromEntityID, toEntityID, upstreamEntityID, step.CurrentNodeEntityID) {
			return true
		}
	}
	return false
}

func connectsEntities(fromEntityID, toEntityID, firstEntityID, secondEntityID string) bool {
	return fromEntityID == firstEntityID && toEntityID == secondEntityID ||
		fromEntityID == secondEntityID && toEntityID == firstEntityID
}

func temporalFactAvailableAt(fact TemporalFact, asOf time.Time) bool {
	return !fact.CreatedAt.IsZero() &&
		!fact.UpdatedAt.IsZero() &&
		!fact.CreatedAt.After(asOf) &&
		!fact.UpdatedAt.After(asOf)
}

func validateEvidence(path, evidenceID, hash, eventID string, allowed map[string]struct{}, asOf time.Time, facts ReferenceFacts) error {
	if _, ok := allowed[evidenceID]; !ok {
		return invalidReference(path+".evidence_id", evidenceID, "Evidence is not attached to the formal fact")
	}
	evidence, ok := facts.Evidences[evidenceID]
	if !ok || evidence.EventID != eventID || evidence.Hash != hash {
		return invalidReference(path+".evidence_id", evidenceID, "Evidence ID/hash/Event lineage does not match")
	}
	if evidence.KnowledgeAvailableAt.After(asOf) {
		return invalidReference(path+".evidence_id", evidenceID, "Evidence was not knowable by analysis_as_of")
	}
	return nil
}

func resultFromReceipt(r Receipt, replayed bool) Result {
	return Result{
		ReceiptID: r.ID, AnalysisBatchID: r.AnalysisBatchID, PayloadHash: r.PayloadHash,
		ThemeID: r.ThemeID, PublicationMode: r.PublicationMode,
		ReasoningTreeIDsByIndustryChainEntityID: cloneMap(r.ReasoningTreeIDsByIndustryChainEntityID),
		ReasoningTreeIDsByTreeKey:               cloneMap(r.ReasoningTreeIDsByTreeKey),
		Counts:                                  r.Counts, PublishedAt: r.PublishedAt, ImportedAt: r.ImportedAt, Replayed: replayed,
	}
}

func cloneMap(input map[string]string) map[string]string {
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

var hashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type Aggregate struct {
	AnalysisBatchID      string          `json:"analysis_batch_id"`
	AnalysisAsOf         string          `json:"analysis_as_of"`
	DiscoveryWindowStart string          `json:"discovery_window_start"`
	DiscoveryWindowEnd   string          `json:"discovery_window_end"`
	Theme                ThemeInput      `json:"theme"`
	ReasoningTrees       []ReasoningTree `json:"reasoning_trees"`
}

type ReasoningTree struct {
	ReasonTreeInput
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
	window, err := (ThemeBatch{
		AnalysisBatchID: a.AnalysisBatchID,
		AnalysisAsOf:    a.AnalysisAsOf,
		WindowStart:     a.DiscoveryWindowStart,
		WindowEnd:       a.DiscoveryWindowEnd,
		Themes:          []ThemeInput{a.Theme},
	}).Validate()
	if err != nil {
		return time.Time{}, "", err
	}
	if len(a.ReasoningTrees) == 0 {
		return time.Time{}, "", invalid("reasoning_trees", "", "must contain at least one Reason Tree")
	}
	themeID := publicationThemeID(a.AnalysisBatchID, a.Theme.ThemeKey)
	legacyTrees := make([]ReasonTreeInput, 0, len(a.ReasoningTrees))
	for treeIndex, tree := range a.ReasoningTrees {
		legacyNodes := make([]ReasonTreeNodeInput, 0, len(tree.Nodes))
		for nodeIndex, node := range tree.Nodes {
			legacySignals := make([]ReasonTreeSignalInput, 0, len(node.Signals))
			for signalIndex, signal := range node.Signals {
				path := fmt.Sprintf("reasoning_trees[%d].nodes[%d].signals[%d].lineage", treeIndex, nodeIndex, signalIndex)
				if err := signal.Lineage.validate(path); err != nil {
					return time.Time{}, "", err
				}
				legacySignals = append(legacySignals, ReasonTreeSignalInput{
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
			legacyNodes = append(legacyNodes, ReasonTreeNodeInput{
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
		legacyTree := tree.ReasonTreeInput
		legacyTree.Nodes = legacyNodes
		legacyTrees = append(legacyTrees, legacyTree)
	}
	if err := (ReasonTreePublication{
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
	return value != nil && lowercaseUUIDPattern.MatchString(*value)
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
	return canonicalPublicationHashValue(value, "research Theme Aggregate V2")
}
func (s *UseCase) PublishSnapshot(ctx context.Context, publisher string, aggregate SnapshotAggregate) (Result, error) {
	if s == nil || s.publicationStore == nil {
		return Result{}, errors.New("research publication store is required")
	}
	publisher = strings.TrimSpace(publisher)
	if publisher == "" || len(publisher) > 200 {
		return Result{}, errors.New("publisher subject must contain 1..200 characters")
	}
	analysisAsOf, themeID, err := aggregate.Validate()
	if err != nil {
		return Result{}, err
	}
	payloadHash, err := CanonicalSnapshotHash(aggregate)
	if err != nil {
		return Result{}, fmt.Errorf("hash research snapshot publication: %w", err)
	}
	plan := snapshotPublicationPlan(aggregate, themeID, payloadHash)
	var result Result
	err = s.publicationStore.InResearchPublicationTransaction(ctx, func(tx PublicationTransaction) error {
		if err := tx.Lock(ctx, aggregate.AnalysisBatchID); err != nil {
			return fmt.Errorf("lock research publication: %w", err)
		}
		existing, err := tx.Receipt(ctx, aggregate.AnalysisBatchID)
		if err != nil {
			return fmt.Errorf("load research publication receipt: %w", err)
		}
		if existing != nil {
			if existing.ContractVersion != 3 || existing.PublicationMode != SnapshotPublicationMode {
				return ErrPayloadConflict
			}
			if existing.PublisherSubject != publisher {
				return ErrPublisherConflict
			}
			if existing.PayloadHash != payloadHash {
				return ErrPayloadConflict
			}
			if existing.ThemeID != plan.ThemeID ||
				!reflect.DeepEqual(existing.ReasoningTreeIDsByTreeKey, plan.ReasoningTreeIDsByTreeKey) ||
				existing.Counts != plan.Counts {
				return errors.New("research snapshot publication receipt does not match deterministic plan")
			}
			if err := tx.Verify(ctx, *existing); err != nil {
				return fmt.Errorf("verify research snapshot publication replay: %w", err)
			}
			result = resultFromReceipt(*existing, true)
			return nil
		}

		facts, err := tx.ReferenceFacts(ctx, snapshotReferenceQuery(aggregate))
		if err != nil {
			return fmt.Errorf("load research snapshot references: %w", err)
		}
		if err := validateSnapshotReferences(aggregate, facts); err != nil {
			return err
		}

		publishedAt := s.now().UTC().Truncate(time.Microsecond)
		receipt := plan
		receipt.PublisherSubject = publisher
		receipt.PublishedAt, receipt.ImportedAt = publishedAt, publishedAt
		if err := tx.InsertThemeReceipt(ctx, receipt); err != nil {
			return fmt.Errorf("insert aggregate receipt: %w", err)
		}
		windowStart, _ := time.Parse(time.RFC3339, aggregate.DiscoveryWindowStart)
		windowEnd, _ := time.Parse(time.RFC3339, aggregate.DiscoveryWindowEnd)
		theme := aggregate.Theme
		if err := tx.InsertTheme(ctx, PublicationThemeRecord{
			ID: themeID, ImportReceiptID: receipt.ID, AnalysisBatchID: aggregate.AnalysisBatchID,
			ThemeKey: theme.ThemeKey, Title: theme.Title, OneLineConclusion: theme.OneLineConclusion,
			ConclusionDirection: theme.ConclusionDirection, ImpactStrength: theme.ImpactStrength,
			AttentionLevel: theme.AttentionLevel, ConclusionStatus: theme.ConclusionStatus,
			TransmissionStage: theme.TransmissionStage, InvestmentGuidanceAction: theme.InvestmentGuidanceAction,
			InvestmentGuidanceSummary: theme.InvestmentGuidanceSummary,
			TimeHorizonCategory:       theme.TimeHorizonCategory, TimeHorizonSummary: theme.TimeHorizonSummary,
			TransmissionSummary: theme.TransmissionSummary, CheckpointSummary: theme.CheckpointSummary,
			RiskSummary: theme.RiskSummary, AnalysisAsOf: analysisAsOf,
			WindowStart: windowStart, WindowEnd: windowEnd, PublishedAt: publishedAt,
		}); err != nil {
			return fmt.Errorf("insert Theme: %w", err)
		}
		for _, impact := range theme.Impacts {
			if err := tx.InsertSnapshotThemeImpact(ctx, SnapshotImpactRecord{
				ThemeID: themeID, NodeKey: impact.NodeKey, DisplayName: impact.DisplayName,
				RelationRole: impact.RelationRole, ImpactDirection: impact.ImpactDirection,
				ImpactSummary: impact.ImpactSummary, DisplayOrder: impact.DisplayOrder,
			}); err != nil {
				return fmt.Errorf("insert snapshot Theme Impact: %w", err)
			}
		}
		for _, event := range theme.Events {
			if err := tx.InsertThemeEvent(ctx, PublicationThemeEventRecord{
				ThemeID: themeID, EventID: event.EventID, EvidenceRole: event.EvidenceRole,
				SupportedClaim: event.SupportedClaim, EvidenceIDs: append([]string(nil), event.EvidenceIDs...),
			}); err != nil {
				return fmt.Errorf("insert Theme Event: %w", err)
			}
		}

		treeReceipt := SnapshotTreeReceipt{
			ID:      identity.NormalizeUUID("research_reasoning_tree_import_receipt", themeID),
			ThemeID: themeID, PublisherSubject: publisher, PayloadHash: payloadHash,
			ReasoningTreeIDsByTreeKey: cloneMap(plan.ReasoningTreeIDsByTreeKey),
			Counts: ReasonTreeCounts{
				ReasoningTrees: plan.Counts.ReasoningTrees, Nodes: plan.Counts.Nodes,
				EventAssociations:  plan.Counts.TreeEventAssociations,
				SignalAssociations: plan.Counts.SignalAssociations, Receipts: 1,
			},
			PublishedAt: publishedAt, ImportedAt: publishedAt,
		}
		if err := tx.InsertSnapshotTreeReceipt(ctx, treeReceipt); err != nil {
			return fmt.Errorf("insert snapshot Reason Tree receipt: %w", err)
		}
		for _, tree := range aggregate.ReasoningTrees {
			treeID := plan.ReasoningTreeIDsByTreeKey[tree.TreeKey]
			if err := tx.InsertSnapshotTree(ctx, SnapshotTreeRecord{
				ID: treeID, ThemeID: themeID, ImportReceiptID: treeReceipt.ID,
				TreeKey: tree.TreeKey, DisplayName: tree.DisplayName, Title: tree.Title,
				DisplayOrder: tree.DisplayOrder, OneLineConclusion: tree.OneLineConclusion,
				FactSummary: tree.FactSummary, TransmissionSummary: tree.TransmissionSummary,
				ImpactDirection: tree.ImpactDirection, ImpactStrength: tree.ImpactStrength,
				ImpactSummary: tree.ImpactSummary, ConclusionBoundarySummary: tree.ConclusionBoundarySummary,
				SupportSummary: tree.SupportSummary, CounterSummary: tree.CounterSummary,
				InvalidationConditions: append([]string(nil), tree.InvalidationConditions...),
				Checkpoints:            append([]ReasonTreeCheckpoint(nil), tree.Checkpoints...),
			}); err != nil {
				return fmt.Errorf("insert snapshot Reason Tree: %w", err)
			}
			for _, event := range tree.Events {
				if err := tx.InsertTreeEvent(ctx, ReasonTreeEventRecord{
					ReasoningTreeID: treeID, EventID: event.EventID, EvidenceRole: event.EvidenceRole,
					DisplayOrder: event.DisplayOrder, EvidenceIDs: append([]string(nil), event.EvidenceIDs...),
				}); err != nil {
					return fmt.Errorf("insert snapshot Reason Tree Event: %w", err)
				}
			}
			for _, node := range tree.Nodes {
				nodeID := SnapshotNodeID(treeID, node.NodeKey)
				record := SnapshotNodeRecord{
					ID: nodeID, ReasoningTreeID: treeID, NodeKey: node.NodeKey, DisplayName: node.DisplayName,
					Position: node.Position, StateSummary: node.StateSummary,
					ImpactDirection: node.ImpactDirection, ImpactStrength: node.ImpactStrength,
					ImpactSummary: node.ImpactSummary, ReasoningBasisSummary: node.ReasoningBasisSummary,
					EvidenceGapSummary: node.EvidenceGapSummary,
				}
				if node.IncomingTransmission != nil {
					record.IncomingTransmissionTitle = node.IncomingTransmission.Title
					record.IncomingTransmissionMechanism = &node.IncomingTransmission.Mechanism
					record.IncomingConditionSummary = node.IncomingTransmission.ConditionSummary
				}
				if err := tx.InsertSnapshotNode(ctx, record); err != nil {
					return fmt.Errorf("insert snapshot Reason Tree Node: %w", err)
				}
				for _, signal := range node.Signals {
					if err := tx.InsertSnapshotSignal(ctx, SnapshotSignalRecord{
						ReasoningTreeNodeID: nodeID, SignalKey: signal.SignalKey,
						SignalRole: signal.Role, DisplaySummary: signal.DisplaySummary,
						VariableName: signal.VariableName, SignalDirection: signal.Direction,
						DisplayOrder: signal.DisplayOrder,
					}); err != nil {
						return fmt.Errorf("insert snapshot Reason Tree Signal: %w", err)
					}
				}
			}
		}
		if err := tx.Verify(ctx, receipt); err != nil {
			return fmt.Errorf("verify research snapshot publication: %w", err)
		}
		result = resultFromReceipt(receipt, false)
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

func snapshotPublicationPlan(a SnapshotAggregate, themeID, payloadHash string) Receipt {
	treeIDs := make(map[string]string, len(a.ReasoningTrees))
	counts := Counts{Themes: 1, Impacts: len(a.Theme.Impacts), ThemeEventAssociations: len(a.Theme.Events), Receipts: 2}
	for _, tree := range a.ReasoningTrees {
		treeIDs[tree.TreeKey] = SnapshotTreeID(themeID, tree.TreeKey)
		counts.ReasoningTrees++
		counts.TreeEventAssociations += len(tree.Events)
		counts.Nodes += len(tree.Nodes)
		for _, node := range tree.Nodes {
			counts.SignalAssociations += len(node.Signals)
		}
	}
	return Receipt{
		ID:              identity.NormalizeUUID("research_theme_import_receipt", a.AnalysisBatchID),
		AnalysisBatchID: a.AnalysisBatchID, PayloadHash: payloadHash, ThemeID: themeID,
		ThemeKey: a.Theme.ThemeKey, ContractVersion: 3, PublicationMode: SnapshotPublicationMode,
		ReasoningTreeIDsByIndustryChainEntityID: map[string]string{},
		ReasoningTreeIDsByTreeKey:               treeIDs, Counts: counts,
	}
}

func snapshotReferenceQuery(a SnapshotAggregate) ReferenceQuery {
	query := ReferenceQuery{SnapshotEventExistenceOnly: true}
	for _, event := range a.Theme.Events {
		query.EventIDs = append(query.EventIDs, event.EventID)
		query.EvidenceIDs = append(query.EvidenceIDs, event.EvidenceIDs...)
	}
	for _, tree := range a.ReasoningTrees {
		for _, event := range tree.Events {
			query.EventIDs = append(query.EventIDs, event.EventID)
			query.EvidenceIDs = append(query.EvidenceIDs, event.EvidenceIDs...)
		}
	}
	return query
}

func validateSnapshotReferences(a SnapshotAggregate, facts ReferenceFacts) error {
	validate := func(path string, eventID string, evidenceIDs []string) error {
		if _, ok := facts.Events[eventID]; !ok {
			return invalidReference(path+".event_id", eventID, "Event does not exist")
		}
		for index, evidenceID := range evidenceIDs {
			evidence, ok := facts.Evidences[evidenceID]
			if !ok || evidence.EventID != eventID {
				return invalidReference(fmt.Sprintf("%s.evidence_ids[%d]", path, index), evidenceID, "Evidence does not belong to Event")
			}
		}
		return nil
	}
	for index, event := range a.Theme.Events {
		if err := validate(fmt.Sprintf("theme.events[%d]", index), event.EventID, event.EvidenceIDs); err != nil {
			return err
		}
	}
	for treeIndex, tree := range a.ReasoningTrees {
		for eventIndex, event := range tree.Events {
			if err := validate(fmt.Sprintf("reasoning_trees[%d].events[%d]", treeIndex, eventIndex), event.EventID, event.EvidenceIDs); err != nil {
				return err
			}
		}
	}
	return nil
}

const SnapshotPublicationMode = "analyst_snapshot"

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
	TreeKey                   string                 `json:"tree_key"`
	DisplayName               string                 `json:"display_name"`
	Title                     string                 `json:"title"`
	DisplayOrder              int                    `json:"display_order"`
	OneLineConclusion         string                 `json:"one_line_conclusion"`
	FactSummary               *string                `json:"fact_summary"`
	TransmissionSummary       *string                `json:"transmission_summary"`
	ImpactDirection           string                 `json:"impact_direction"`
	ImpactStrength            string                 `json:"impact_strength"`
	ImpactSummary             *string                `json:"impact_summary"`
	ConclusionBoundarySummary *string                `json:"conclusion_boundary_summary"`
	SupportSummary            *string                `json:"support_summary"`
	CounterSummary            *string                `json:"counter_summary"`
	InvalidationConditions    []string               `json:"invalidation_conditions"`
	Checkpoints               []ReasonTreeCheckpoint `json:"checkpoints"`
	Events                    []SnapshotTreeEvent    `json:"events"`
	Nodes                     []SnapshotNode         `json:"nodes"`
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
	if !researchKeyPattern.MatchString(a.Theme.ThemeKey) {
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
		if !researchKeyPattern.MatchString(tree.TreeKey) {
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
	return asOf, publicationThemeID(a.AnalysisBatchID, a.Theme.ThemeKey), nil
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
	if !isAllowedValue(t.ConclusionDirection, "positive", "negative", "mixed", "neutral", "uncertain") {
		return invalid("theme.conclusion_direction", t.ConclusionDirection, "has an unsupported value")
	}
	if !isAllowedValue(t.ImpactStrength, "strong", "medium", "weak", "unknown") {
		return invalid("theme.impact_strength", t.ImpactStrength, "has an unsupported value")
	}
	if t.AttentionLevel != nil && !isAllowedValue(*t.AttentionLevel, "high", "medium", "low") {
		return invalid("theme.attention_level", *t.AttentionLevel, "has an unsupported value")
	}
	if t.ConclusionStatus != nil && !isAllowedValue(*t.ConclusionStatus, "supported", "partial", "conflicted") {
		return invalid("theme.conclusion_status", *t.ConclusionStatus, "has an unsupported value")
	}
	if !isAllowedValue(t.TransmissionStage, "identification", "validation", "diffusion", "dampening") {
		return invalid("theme.transmission_stage", t.TransmissionStage, "has an unsupported value")
	}
	if !isAllowedValue(t.InvestmentGuidanceAction, "focus", "avoid", "observe", "differentiate") {
		return invalid("theme.investment_guidance_action", t.InvestmentGuidanceAction, "has an unsupported value")
	}
	if !isAllowedValue(t.TimeHorizonCategory, "short_term", "medium_term", "long_term", "custom") {
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
		if !researchKeyPattern.MatchString(impact.NodeKey) {
			return invalid(path+".node_key", impact.NodeKey, "must match the local key pattern")
		}
		if _, duplicate := impactKeys[impact.NodeKey]; duplicate {
			return invalid(path+".node_key", impact.NodeKey, "must be unique within the Theme")
		}
		impactKeys[impact.NodeKey] = struct{}{}
		if err := snapshotRequiredText(path+".display_name", impact.DisplayName, 300); err != nil {
			return err
		}
		if !isAllowedValue(impact.RelationRole, "driver", "beneficiary", "constraint", "exposure") {
			return invalid(path+".relation_role", impact.RelationRole, "has an unsupported value")
		}
		if !isAllowedValue(impact.ImpactDirection, "positive", "negative", "mixed", "neutral", "uncertain") {
			return invalid(path+".impact_direction", impact.ImpactDirection, "has an unsupported value")
		}
		if err := snapshotOptionalText(path+".impact_summary", impact.ImpactSummary, 2000); err != nil {
			return err
		}
	}
	if len(t.Events) == 0 {
		return invalid("theme.events", "", "must contain at least one Event")
	}
	seenEvents := make(map[string]struct{}, len(t.Events))
	for index, event := range t.Events {
		path := fmt.Sprintf("theme.events[%d]", index)
		if err := validateSnapshotEvent(path, event.EventID, event.EvidenceIDs, event.EvidenceRole); err != nil {
			return err
		}
		if _, duplicate := seenEvents[event.EventID]; duplicate {
			return invalid(path+".event_id", event.EventID, "must be unique within the Theme")
		}
		seenEvents[event.EventID] = struct{}{}
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
	if !isAllowedValue(t.ImpactDirection, "positive", "negative", "mixed", "neutral", "uncertain") {
		return nil, invalid(path+".impact_direction", t.ImpactDirection, "has an unsupported value")
	}
	if !isAllowedValue(t.ImpactStrength, "strong", "medium", "weak", "unknown") {
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
		if !isAllowedValue(checkpoint.Type, "event", "relationship", "metric") {
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
		if !researchKeyPattern.MatchString(node.NodeKey) {
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
	if !isAllowedValue(n.ImpactDirection, "positive", "negative", "mixed", "neutral", "uncertain") {
		return invalid(path+".impact_direction", n.ImpactDirection, "has an unsupported value")
	}
	if !isAllowedValue(n.ImpactStrength, "strong", "medium", "weak", "unknown") {
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
		if !researchKeyPattern.MatchString(signal.SignalKey) {
			return invalid(path+".signal_key", signal.SignalKey, "must match the local key pattern")
		}
		if _, duplicate := seen[signal.SignalKey]; duplicate {
			return invalid(path+".signal_key", signal.SignalKey, "must be unique within the Node")
		}
		seen[signal.SignalKey] = struct{}{}
		if err := snapshotRequiredText(path+".display_summary", signal.DisplaySummary, 200); err != nil {
			return err
		}
		if !isAllowedValue(signal.Role, "primary", "supporting", "contradicting") {
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
		if signal.Direction != nil && !isAllowedValue(*signal.Direction, "increase", "decrease", "mixed", "unchanged", "uncertain") {
			return invalid(path+".direction", *signal.Direction, "has an unsupported value")
		}
	}
	if primaryCount != 1 {
		return invalid(nodePath+".signals", "", "must contain exactly one primary Signal")
	}
	return nil
}

func validateSnapshotEvent(path, eventID string, evidenceIDs []string, role string) error {
	if !lowercaseUUIDPattern.MatchString(eventID) {
		return invalid(path+".event_id", eventID, "must be a standard lowercase UUID")
	}
	if !isAllowedValue(role, "driver", "supporting", "contradicting", "context") {
		return invalid(path+".evidence_role", role, "has an unsupported value")
	}
	for index, evidenceID := range evidenceIDs {
		if !lowercaseUUIDPattern.MatchString(evidenceID) {
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
	return canonicalPublicationHashValue(canonicalSnapshotAggregate(value), "research Theme analyst snapshot V3")
}

func canonicalSnapshotAggregate(value SnapshotAggregate) SnapshotAggregate {
	canonical := value
	canonical.Theme = value.Theme
	canonical.Theme.Events = append([]SnapshotEvent(nil), value.Theme.Events...)
	sort.Slice(canonical.Theme.Events, func(i, j int) bool {
		return canonical.Theme.Events[i].EventID < canonical.Theme.Events[j].EventID
	})
	return canonical
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

type GraphSubgraph = entitybiz.ResearchGraphSubgraph
type GraphEntity = entitybiz.ResearchGraphEntity
type GraphRelationDefinition = entitybiz.ResearchGraphRelation
type GraphEntityRelation = entitybiz.ResearchGraphEntityRelation
type GraphIndustryChain = entitybiz.ResearchGraphIndustryChain
type GraphIndustryChainMembership = entitybiz.ResearchGraphMembership
type GraphIndustryChainEdge = entitybiz.ResearchGraphIndustryEdge

type GraphStore interface {
	SearchResearchGraph(context.Context, GraphQuery) (GraphSubgraph, error)
}

type GraphQuery = entitybiz.ResearchGraphQuery

const (
	GraphContractVersion       = "research-graph-search.v1"
	GraphStableOrderingVersion = "depth-seed-relation-endpoints-edge-id.v1"
	GraphMaxDepth              = 5
	GraphMaxSeedEntities       = 20
	GraphMaxRelationFilters    = 20
	GraphMaxNodeBudget         = 500
	GraphMaxEdgeBudget         = 1_000
	GraphMaxResultBytes        = 4 * 1024 * 1024
)

type Direction = entitybiz.ResearchGraphDirection

const (
	DirectionOutgoing = entitybiz.ResearchGraphDirectionOutgoing
	DirectionIncoming = entitybiz.ResearchGraphDirectionIncoming
	DirectionBoth     = entitybiz.ResearchGraphDirectionBoth
)

type RelationFilter = entitybiz.ResearchGraphRelationFilter

type GraphSearchRequest struct {
	AnalysisAsOf          string           `json:"analysis_as_of"`
	SeedEntityIDs         []string         `json:"seed_entity_ids"`
	RelationFilters       []RelationFilter `json:"relation_filters"`
	MaxDepth              int              `json:"max_depth"`
	IndustryChainEntityID *string          `json:"industry_chain_entity_id,omitempty"`
	NodeBudget            int              `json:"node_budget"`
	EdgeBudget            int              `json:"edge_budget"`
}

type GraphSearchResult struct {
	ContractVersion          string                                  `json:"contract_version"`
	AnalysisAsOf             string                                  `json:"analysis_as_of"`
	QueryFingerprint         string                                  `json:"query_fingerprint"`
	GraphFingerprint         string                                  `json:"graph_fingerprint"`
	ActualDepth              int                                     `json:"actual_depth"`
	Entities                 []entitybiz.ResearchGraphEntity         `json:"entities"`
	RelationDefinitions      []entitybiz.ResearchGraphRelation       `json:"relation_definitions"`
	EntityRelations          []entitybiz.ResearchGraphEntityRelation `json:"entity_relations"`
	IndustryChains           []entitybiz.ResearchGraphIndustryChain  `json:"industry_chains"`
	IndustryChainMemberships []entitybiz.ResearchGraphMembership     `json:"industry_chain_memberships"`
	IndustryChainGraphEdges  []entitybiz.ResearchGraphIndustryEdge   `json:"industry_chain_graph_edges"`
}

type GraphValidationError = entitybiz.ResearchGraphValidationError

func (s *UseCase) Search(
	ctx context.Context,
	request GraphSearchRequest,
) (GraphSearchResult, error) {
	if s == nil || s.graphStore == nil {
		return GraphSearchResult{}, errors.New("research graph store is required")
	}
	query, normalized, err := validateGraphSearchRequest(request)
	if err != nil {
		return GraphSearchResult{}, err
	}
	graph, err := s.graphStore.SearchResearchGraph(ctx, query)
	if err != nil {
		var limit *entitybiz.ResearchGraphResourceLimitError
		if errors.As(err, &limit) {
			return GraphSearchResult{}, &ResearchResourceLimitError{
				Reason: limit.Reason, Component: limit.Component,
				ActualRows: limit.ActualRows, MaxRows: limit.MaxRows,
				ActualBytes: limit.ActualBytes, MaxBytes: limit.MaxBytes,
				RetryGuidance: limit.RetryGuidance,
			}
		}
		return GraphSearchResult{}, err
	}
	graph = normalizeSubgraph(graph)
	if graph.ActualDepth < 0 || graph.ActualDepth > query.MaxDepth ||
		!referencesResolve(graph) {
		return GraphSearchResult{}, errors.New("research graph result is not reference complete")
	}
	edgeCount := len(graph.EntityRelations) + len(graph.IndustryChainGraphEdges)
	if len(graph.Entities) > query.NodeBudget || edgeCount > query.EdgeBudget {
		component := "research_graph_nodes"
		actual := int64(len(graph.Entities))
		maximum := int64(query.NodeBudget)
		reason := "research graph result exceeds the requested node budget"
		if len(graph.Entities) <= query.NodeBudget {
			component = "research_graph_edges"
			actual = int64(edgeCount)
			maximum = int64(query.EdgeBudget)
			reason = "research graph result exceeds the requested edge budget"
		}
		return GraphSearchResult{}, &ResearchResourceLimitError{
			Reason:     reason,
			Component:  component,
			ActualRows: &actual, MaxRows: &maximum,
			RetryGuidance: "reduce_depth_relation_types_or_chain_scope",
		}
	}
	graphPayload, err := json.Marshal(graph)
	if err != nil {
		return GraphSearchResult{}, errors.New("research graph result is invalid")
	}
	if len(graphPayload) > GraphMaxResultBytes {
		actual := int64(len(graphPayload))
		maximum := int64(GraphMaxResultBytes)
		return GraphSearchResult{}, &ResearchResourceLimitError{
			Reason:      "research graph result exceeds the response budget",
			Component:   "research_graph_result",
			ActualBytes: &actual, MaxBytes: &maximum,
			RetryGuidance: "reduce_depth_relation_types_or_chain_scope",
		}
	}
	queryFingerprint, err := payloadFingerprint(normalized)
	if err != nil {
		return GraphSearchResult{}, err
	}
	graphFingerprint, err := payloadFingerprint(graph)
	if err != nil {
		return GraphSearchResult{}, err
	}
	return GraphSearchResult{
		ContractVersion:          GraphContractVersion,
		AnalysisAsOf:             normalized.AnalysisAsOf,
		QueryFingerprint:         queryFingerprint,
		GraphFingerprint:         graphFingerprint,
		ActualDepth:              graph.ActualDepth,
		Entities:                 graph.Entities,
		RelationDefinitions:      graph.RelationDefinitions,
		EntityRelations:          graph.EntityRelations,
		IndustryChains:           graph.IndustryChains,
		IndustryChainMemberships: graph.IndustryChainMemberships,
		IndustryChainGraphEdges:  graph.IndustryChainGraphEdges,
	}, nil
}

type normalizedGraphSearchRequest struct {
	GraphContractVersion       string           `json:"contract_version"`
	GraphStableOrderingVersion string           `json:"stable_ordering_version"`
	AnalysisAsOf               string           `json:"analysis_as_of"`
	SeedEntityIDs              []string         `json:"seed_entity_ids"`
	RelationFilters            []RelationFilter `json:"relation_filters"`
	MaxDepth                   int              `json:"max_depth"`
	IndustryChainEntityID      *string          `json:"industry_chain_entity_id,omitempty"`
	NodeBudget                 int              `json:"node_budget"`
	EdgeBudget                 int              `json:"edge_budget"`
}

func validateGraphSearchRequest(request GraphSearchRequest) (GraphQuery, normalizedGraphSearchRequest, error) {
	asOf, err := time.Parse(time.RFC3339, request.AnalysisAsOf)
	if err != nil {
		return GraphQuery{}, normalizedGraphSearchRequest{}, &GraphValidationError{
			Reason: "analysis_as_of must be an RFC3339 UTC timestamp",
		}
	}
	_, offset := asOf.Zone()
	if offset != 0 {
		return GraphQuery{}, normalizedGraphSearchRequest{}, &GraphValidationError{Reason: "analysis_as_of must use UTC"}
	}
	if len(request.SeedEntityIDs) < 1 || len(request.SeedEntityIDs) > GraphMaxSeedEntities {
		return GraphQuery{}, normalizedGraphSearchRequest{}, &GraphValidationError{
			Reason: fmt.Sprintf("seed_entity_ids must contain between 1 and %d IDs", GraphMaxSeedEntities),
		}
	}
	seedSet := map[string]struct{}{}
	for _, id := range request.SeedEntityIDs {
		if !identity.IsUUID(id) {
			return GraphQuery{}, normalizedGraphSearchRequest{}, &GraphValidationError{Reason: "seed_entity_ids contains an invalid UUID"}
		}
		if _, exists := seedSet[id]; exists {
			return GraphQuery{}, normalizedGraphSearchRequest{}, &GraphValidationError{
				Reason: "seed_entity_ids must contain unique IDs",
			}
		}
		seedSet[id] = struct{}{}
	}
	seeds := sortedSet(seedSet)
	if len(request.RelationFilters) < 1 || len(request.RelationFilters) > GraphMaxRelationFilters {
		return GraphQuery{}, normalizedGraphSearchRequest{}, &GraphValidationError{
			Reason: fmt.Sprintf("relation_filters must contain between 1 and %d filters", GraphMaxRelationFilters),
		}
	}
	filterSet := map[string]RelationFilter{}
	for _, filter := range request.RelationFilters {
		filter.RelationType = strings.TrimSpace(filter.RelationType)
		if filter.RelationType == "" ||
			(filter.Direction != DirectionOutgoing &&
				filter.Direction != DirectionIncoming &&
				filter.Direction != DirectionBoth) {
			return GraphQuery{}, normalizedGraphSearchRequest{}, &GraphValidationError{Reason: "relation_filters is invalid"}
		}
		if _, exists := filterSet[filter.RelationType]; exists {
			return GraphQuery{}, normalizedGraphSearchRequest{}, &GraphValidationError{
				Reason: "relation_filters must configure each relation_type exactly once",
			}
		}
		filterSet[filter.RelationType] = filter
	}
	filterKeys := make([]string, 0, len(filterSet))
	for key := range filterSet {
		filterKeys = append(filterKeys, key)
	}
	sort.Strings(filterKeys)
	filters := make([]RelationFilter, 0, len(filterKeys))
	for _, key := range filterKeys {
		filters = append(filters, filterSet[key])
	}
	if request.MaxDepth < 1 || request.MaxDepth > GraphMaxDepth {
		return GraphQuery{}, normalizedGraphSearchRequest{}, &GraphValidationError{
			Reason: fmt.Sprintf("max_depth must be between 1 and %d", GraphMaxDepth),
		}
	}
	if request.NodeBudget < 1 || request.NodeBudget > GraphMaxNodeBudget ||
		request.EdgeBudget < 1 || request.EdgeBudget > GraphMaxEdgeBudget {
		return GraphQuery{}, normalizedGraphSearchRequest{}, &GraphValidationError{
			Reason: "node_budget or edge_budget exceeds the supported range",
		}
	}
	if request.IndustryChainEntityID != nil &&
		!identity.IsUUID(*request.IndustryChainEntityID) {
		return GraphQuery{}, normalizedGraphSearchRequest{}, &GraphValidationError{
			Reason: "industry_chain_entity_id must be a UUID",
		}
	}
	asOf = asOf.UTC()
	normalized := normalizedGraphSearchRequest{
		GraphContractVersion:       GraphContractVersion,
		GraphStableOrderingVersion: GraphStableOrderingVersion,
		AnalysisAsOf:               asOf.Format(time.RFC3339Nano),
		SeedEntityIDs:              seeds,
		RelationFilters:            filters,
		MaxDepth:                   request.MaxDepth,
		IndustryChainEntityID:      request.IndustryChainEntityID,
		NodeBudget:                 request.NodeBudget,
		EdgeBudget:                 request.EdgeBudget,
	}
	return GraphQuery{
		AnalysisAsOf:          asOf,
		SeedEntityIDs:         seeds,
		RelationFilters:       filters,
		MaxDepth:              request.MaxDepth,
		IndustryChainEntityID: request.IndustryChainEntityID,
		NodeBudget:            request.NodeBudget,
		EdgeBudget:            request.EdgeBudget,
		FactPolicy:            entitybiz.ApprovedActiveResearchGraphFactPolicy(),
	}, normalized, nil
}

func referencesResolve(graph GraphSubgraph) bool {
	entities := map[string]struct{}{}
	for _, entity := range graph.Entities {
		entities[entity.EntityID] = struct{}{}
	}
	relationTypes := map[string]struct{}{}
	for _, definition := range graph.RelationDefinitions {
		relationTypes[definition.RelationType] = struct{}{}
	}
	chains := map[string]struct{}{}
	for _, chain := range graph.IndustryChains {
		if _, ok := entities[chain.IndustryChainEntityID]; !ok {
			return false
		}
		chains[chain.IndustryChainEntityID] = struct{}{}
	}
	memberships := map[string]struct{}{}
	for _, membership := range graph.IndustryChainMemberships {
		if _, ok := chains[membership.IndustryChainEntityID]; !ok {
			return false
		}
		if _, ok := entities[membership.ChainNodeEntityID]; !ok {
			return false
		}
		memberships[membership.IndustryChainEntityID+"\x00"+membership.ChainNodeEntityID] = struct{}{}
	}
	for _, relation := range graph.EntityRelations {
		if _, ok := entities[relation.FromEntityID]; !ok {
			return false
		}
		if _, ok := entities[relation.ToEntityID]; !ok {
			return false
		}
		if _, ok := relationTypes[relation.RelationType]; !ok {
			return false
		}
	}
	for _, edge := range graph.IndustryChainGraphEdges {
		if _, ok := relationTypes[edge.RelationType]; !ok {
			return false
		}
		if _, ok := memberships[edge.IndustryChainEntityID+"\x00"+edge.FromChainNodeEntityID]; !ok {
			return false
		}
		if _, ok := memberships[edge.IndustryChainEntityID+"\x00"+edge.ToChainNodeEntityID]; !ok {
			return false
		}
	}
	return true
}

func normalizeSubgraph(graph GraphSubgraph) GraphSubgraph {
	if graph.Entities == nil {
		graph.Entities = []entitybiz.ResearchGraphEntity{}
	}
	if graph.RelationDefinitions == nil {
		graph.RelationDefinitions = []entitybiz.ResearchGraphRelation{}
	}
	if graph.EntityRelations == nil {
		graph.EntityRelations = []entitybiz.ResearchGraphEntityRelation{}
	}
	if graph.IndustryChains == nil {
		graph.IndustryChains = []entitybiz.ResearchGraphIndustryChain{}
	}
	if graph.IndustryChainMemberships == nil {
		graph.IndustryChainMemberships = []entitybiz.ResearchGraphMembership{}
	}
	if graph.IndustryChainGraphEdges == nil {
		graph.IndustryChainGraphEdges = []entitybiz.ResearchGraphIndustryEdge{}
	}
	return graph
}
