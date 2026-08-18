package research

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	entitybiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity"
	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
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

func publicationThemeID(analysisBatchID, themeKey string) string {
	return mustDeriveResearchID(coreid.ResearchTheme, "research-theme", analysisBatchID, themeKey)
}

func mustDeriveResearchID(kind coreid.Kind, namespace string, parts ...string) string {
	value, err := coreid.Derive(kind, namespace, parts...)
	if err != nil {
		panic(fmt.Sprintf("derive reviewed Research ID contract: %v", err))
	}
	return value
}

var researchKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._:-]{0,127}$`)

type ReasonTreeCheckpoint struct {
	Type    string `json:"type"`
	Summary string `json:"summary"`
}

type ResearchResourceLimitError struct {
	Reason, Component     string
	ActualRows, MaxRows   *int64
	ActualBytes, MaxBytes *int64
	RetryGuidance         string
}

func (e *ResearchResourceLimitError) Error() string {
	return e.Reason
}

func sortedSet(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func isAllowedValue(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func payloadFingerprint(value any) (string, error) {
	return canonicalPublicationHashValue(value, "research payload")
}

func encodeCursor(cursor researchCursor) (string, error) {
	payload, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

var (
	ErrResearchNotFound               = errors.New("research result not found")
	ErrResearchThemeNotFound          = errors.New("research theme not found")
	ErrResearchReasoningTreesNotFound = errors.New("research reasoning trees not found")
	ErrResearchReasoningTreeNotFound  = errors.New("research reasoning tree not found")
	ErrResearchReasoningTreeInvariant = errors.New("research reasoning tree invariant violation")
)

type ThemeImpactRecord struct {
	NodeKey         string  `json:"node_key"`
	DisplayName     string  `json:"display_name"`
	RelationRole    string  `json:"relation_role"`
	ImpactDirection string  `json:"impact_direction"`
	ImpactSummary   *string `json:"impact_summary"`
	DisplayOrder    int     `json:"display_order"`
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
	ReasoningTreeID, TreeKey, DisplayName, Title string
	DisplayOrder, EventCount                     int
	PublishedAt                                  time.Time
}

type ReasoningTreeListRecord struct {
	Theme          ThemeSummaryRecord
	ReasoningTrees []ReasoningTreeSummaryRecord
}

type CheckpointRecord struct {
	Type, Summary string
}

type SignalRecord struct {
	SignalKey, SignalRole, DisplaySummary string
	VariableName, Direction               *string
	DisplayOrder                          int
}

type ReasoningTreeNodeRecord struct {
	ID, NodeKey, DisplayName, ImpactDirection, ImpactStrength string
	Position                                                  int
	StateSummary, ImpactSummary, ReasoningBasisSummary        *string
	EvidenceGapSummary                                        *string
	IncomingTransmissionTitle                                 *string
	IncomingTransmissionMechanism, IncomingConditionSummary   *string
	Signals                                                   []SignalRecord
}

type ReasoningTreeRecord struct {
	ReasoningTreeID, ThemeID, TreeKey, DisplayName            string
	Title, OneLineConclusion, ImpactDirection, ImpactStrength string
	DisplayOrder, EventCount                                  int
	FactSummary, TransmissionSummary, ImpactSummary           *string
	ConclusionBoundarySummary, SupportSummary, CounterSummary *string
	InvalidationConditions                                    []string
	Checkpoints                                               []CheckpointRecord
	PublishedAt                                               time.Time
	Events                                                    []EventRecord
	Nodes                                                     []ReasoningTreeNodeRecord
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
	TreeKey         string    `json:"tree_key"`
	DisplayName     string    `json:"display_name"`
	ReasoningTreeID string    `json:"reasoning_tree_id"`
	Title           string    `json:"title"`
	DisplayOrder    int       `json:"display_order"`
	EventCount      int       `json:"event_count"`
	PublishedAt     time.Time `json:"published_at"`
}

type ResearchCheckpoint struct {
	Type, Summary string
}

type ResearchSignal struct {
	SignalKey      string  `json:"signal_key"`
	VariableName   *string `json:"variable_name"`
	Direction      *string `json:"direction"`
	SignalRole     string  `json:"signal_role"`
	DisplaySummary string  `json:"display_summary"`
	DisplayOrder   int     `json:"display_order"`
}

type ResearchReasoningTreeNode struct {
	NodeKey                       string           `json:"node_key"`
	DisplayName                   string           `json:"display_name"`
	ID                            string           `json:"id"`
	Position                      int              `json:"position"`
	StateSummary                  *string          `json:"state_summary"`
	ImpactDirection               string           `json:"impact_direction"`
	ImpactStrength                string           `json:"impact_strength"`
	ImpactSummary                 *string          `json:"impact_summary"`
	ReasoningBasisSummary         *string          `json:"reasoning_basis_summary"`
	EvidenceGapSummary            *string          `json:"evidence_gap_summary"`
	IncomingTransmissionTitle     *string          `json:"incoming_transmission_title"`
	IncomingTransmissionMechanism *string          `json:"incoming_transmission_mechanism"`
	IncomingConditionSummary      *string          `json:"incoming_condition_summary"`
	Signals                       []ResearchSignal `json:"signals"`
	PrimarySignal                 ResearchSignal   `json:"primary_signal"`
	SignalDisplaySummary          string           `json:"signal_display_summary"`
}

type ResearchReasoningTree struct {
	ReasoningTreeID, ThemeID, TreeKey, DisplayName            string
	Title, OneLineConclusion, ImpactDirection, ImpactStrength string
	DisplayOrder, EventCount                                  int
	FactSummary, TransmissionSummary, ImpactSummary           *string
	ConclusionBoundarySummary, SupportSummary, CounterSummary *string
	InvalidationConditions                                    []string
	Checkpoints                                               []ResearchCheckpoint
	PublishedAt                                               time.Time
	Events                                                    []ResearchEvent
	Nodes                                                     []ResearchReasoningTreeNode
}

type ResearchReasoningTreeDetail struct {
	ThemeID, ThemeKey, PublicationMode string
	PublicationContractVersion         int
	ImpactNodeIDs                      []string
	ReasoningTree                      ResearchReasoningTree
}

func (s *UseCase) ListReasoningTrees(ctx context.Context, themeID string) (ResearchReasoningTreeList, error) {
	themeID = strings.TrimSpace(themeID)
	if !coreid.Is(themeID, coreid.ResearchTheme) {
		return ResearchReasoningTreeList{}, fmt.Errorf("%w: theme id must be a Research Theme ID", ErrInvalidRequest)
	}
	result, err := s.repository.ListResearchThemeReasoningTrees(ctx, themeID)
	if err != nil {
		return ResearchReasoningTreeList{}, mapReasoningTreeRepositoryError(err)
	}
	summaries := make([]ResearchReasoningTreeSummary, 0, len(result.ReasoningTrees))
	for _, value := range result.ReasoningTrees {
		summaries = append(summaries, ResearchReasoningTreeSummary{
			TreeKey: value.TreeKey, DisplayName: value.DisplayName,
			ReasoningTreeID: value.ReasoningTreeID, Title: value.Title,
			DisplayOrder: value.DisplayOrder, EventCount: value.EventCount,
			PublishedAt: value.PublishedAt.UTC(),
		})
	}
	return ResearchReasoningTreeList{Theme: themeDTO(result.Theme), ReasoningTrees: summaries}, nil
}

func (s *UseCase) GetReasoningTree(ctx context.Context, themeID, reasoningTreeID string) (ResearchReasoningTreeDetail, error) {
	themeID = strings.TrimSpace(themeID)
	reasoningTreeID = strings.TrimSpace(reasoningTreeID)
	if !coreid.Is(themeID, coreid.ResearchTheme) {
		return ResearchReasoningTreeDetail{}, fmt.Errorf("%w: theme id must be a Research Theme ID", ErrInvalidRequest)
	}
	if !coreid.Is(reasoningTreeID, coreid.ResearchReasoningTree) {
		return ResearchReasoningTreeDetail{}, fmt.Errorf("%w: reasoning tree id must be a Research Reasoning Tree ID", ErrInvalidRequest)
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
				SignalRole: signal.SignalRole, DisplaySummary: signal.DisplaySummary,
				DisplayOrder: signal.DisplayOrder,
			}
			signals = append(signals, item)
			if signal.SignalRole == "primary" {
				primary = item
			} else {
				secondary = append(secondary, signal.DisplaySummary)
			}
		}
		nodes = append(nodes, ResearchReasoningTreeNode{
			NodeKey: node.NodeKey, DisplayName: node.DisplayName,
			ID: node.ID, Position: node.Position,
			StateSummary: node.StateSummary, ImpactDirection: node.ImpactDirection,
			ImpactStrength: node.ImpactStrength, ImpactSummary: node.ImpactSummary,
			ReasoningBasisSummary: node.ReasoningBasisSummary, EvidenceGapSummary: node.EvidenceGapSummary,
			IncomingTransmissionTitle:     node.IncomingTransmissionTitle,
			IncomingTransmissionMechanism: node.IncomingTransmissionMechanism,
			IncomingConditionSummary:      node.IncomingConditionSummary,
			Signals:                       signals, PrimarySignal: primary, SignalDisplaySummary: strings.Join(secondary, " · "),
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
	NodeKey         string  `json:"node_key"`
	DisplayName     string  `json:"display_name"`
	RelationRole    string  `json:"relation_role"`
	ImpactDirection string  `json:"impact_direction"`
	ImpactSummary   *string `json:"impact_summary"`
	DisplayOrder    int     `json:"display_order"`
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
	graphStore       GraphStore
	now              func() time.Time
}

func NewUseCase(
	repository Repository,
	publicationStore PublicationStore,
	graphStore GraphStore,
	now func() time.Time,
) (*UseCase, error) {
	if repository == nil || publicationStore == nil || graphStore == nil {
		return nil, errors.New("Research use case dependencies are required")
	}
	if now == nil {
		now = time.Now
	}
	return &UseCase{
		repository:       repository,
		publicationStore: publicationStore,
		graphStore:       graphStore,
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
	if !coreid.Is(strings.TrimSpace(id), coreid.ResearchTheme) {
		return ResearchThemeDetail{}, fmt.Errorf("%w: theme id must be a Research Theme ID", ErrInvalidRequest)
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
	ReasoningTreeIDsByTreeKey                        map[string]string
	Counts                                           Counts
	PublishedAt, ImportedAt                          time.Time
	Replayed                                         bool
}

func resultFromReceipt(r Receipt, replayed bool) Result {
	return Result{
		ReceiptID: r.ID, AnalysisBatchID: r.AnalysisBatchID, PayloadHash: r.PayloadHash,
		ThemeID: r.ThemeID, PublicationMode: r.PublicationMode,
		ReasoningTreeIDsByTreeKey: cloneMap(r.ReasoningTreeIDsByTreeKey),
		Counts:                    r.Counts, PublishedAt: r.PublishedAt, ImportedAt: r.ImportedAt, Replayed: replayed,
	}
}

func cloneMap(input map[string]string) map[string]string {
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
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
			ID:      mustDeriveResearchID(coreid.ResearchReasoningTreeReceipt, "research-reasoning-tree-import-receipt", themeID),
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
		ID:              mustDeriveResearchID(coreid.ResearchThemeReceipt, "research-theme-import-receipt", a.AnalysisBatchID),
		AnalysisBatchID: a.AnalysisBatchID, PayloadHash: payloadHash, ThemeID: themeID,
		ThemeKey: a.Theme.ThemeKey, ContractVersion: 3, PublicationMode: SnapshotPublicationMode,
		ReasoningTreeIDsByTreeKey: treeIDs, Counts: counts,
	}
}

func snapshotReferenceQuery(a SnapshotAggregate) ReferenceQuery {
	query := ReferenceQuery{}
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
	if !coreid.Is(eventID, coreid.Event) {
		return invalid(path+".event_id", eventID, "must be an Event ID")
	}
	if !isAllowedValue(role, "driver", "supporting", "contradicting", "context") {
		return invalid(path+".evidence_role", role, "has an unsupported value")
	}
	for index, evidenceID := range evidenceIDs {
		if !coreid.Is(evidenceID, coreid.EventEvidenceLink) {
			return invalid(fmt.Sprintf("%s.evidence_ids[%d]", path, index), evidenceID, "must be an Event Evidence Link ID")
		}
		if index > 0 && evidenceID <= evidenceIDs[index-1] {
			return invalid(fmt.Sprintf("%s.evidence_ids[%d]", path, index), evidenceID, "must be unique and sorted")
		}
	}
	return nil
}

func SnapshotTreeID(themeID, treeKey string) string {
	return mustDeriveResearchID(coreid.ResearchReasoningTree, "research-reasoning-tree-snapshot", themeID, treeKey)
}

func SnapshotNodeID(treeID, nodeKey string) string {
	return mustDeriveResearchID(coreid.ResearchReasoningTreeNode, "research-reasoning-tree-node-snapshot", treeID, nodeKey)
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
	AnalysisAsOf    string           `json:"analysis_as_of"`
	SeedEntityIDs   []string         `json:"seed_entity_ids"`
	RelationFilters []RelationFilter `json:"relation_filters"`
	MaxDepth        int              `json:"max_depth"`
	IndustryChainID *string          `json:"industry_chain_id,omitempty"`
	NodeBudget      int              `json:"node_budget"`
	EdgeBudget      int              `json:"edge_budget"`
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
	IndustryChainID            *string          `json:"industry_chain_id,omitempty"`
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
		if !entitybiz.IsObjectID(id) {
			return GraphQuery{}, normalizedGraphSearchRequest{}, &GraphValidationError{Reason: "seed_entity_ids contains an invalid Object ID"}
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
	if request.IndustryChainID != nil &&
		!entitybiz.IsIndustryChainID(*request.IndustryChainID) {
		return GraphQuery{}, normalizedGraphSearchRequest{}, &GraphValidationError{
			Reason: "industry_chain_id must be an IndustryChain ID",
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
		IndustryChainID:            request.IndustryChainID,
		NodeBudget:                 request.NodeBudget,
		EdgeBudget:                 request.EdgeBudget,
	}
	return GraphQuery{
		AnalysisAsOf:    asOf,
		SeedEntityIDs:   seeds,
		RelationFilters: filters,
		MaxDepth:        request.MaxDepth,
		IndustryChainID: request.IndustryChainID,
		NodeBudget:      request.NodeBudget,
		EdgeBudget:      request.EdgeBudget,
		FactPolicy:      entitybiz.ApprovedActiveResearchGraphFactPolicy(),
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
		if _, ok := entities[chain.IndustryChainID]; !ok {
			return false
		}
		chains[chain.IndustryChainID] = struct{}{}
	}
	memberships := map[string]struct{}{}
	for _, membership := range graph.IndustryChainMemberships {
		if _, ok := chains[membership.IndustryChainID]; !ok {
			return false
		}
		if _, ok := entities[membership.ChainNodeID]; !ok {
			return false
		}
		memberships[membership.IndustryChainID+"\x00"+membership.ChainNodeID] = struct{}{}
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
		if _, ok := memberships[edge.IndustryChainID+"\x00"+edge.FromChainNodeID]; !ok {
			return false
		}
		if _, ok := memberships[edge.IndustryChainID+"\x00"+edge.ToChainNodeID]; !ok {
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
