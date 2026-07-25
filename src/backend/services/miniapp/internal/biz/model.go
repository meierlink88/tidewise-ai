package biz

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
)

const (
	DataAPIPrefix      = "/api/data/v1"
	ResearchThemesPath = DataAPIPrefix + "/research/themes"
)

// ResearchRepo is the Miniapp-owned boundary for page-level research data.
// The Data Service HTTP adapter implements this port.
type ResearchRepo interface {
	ListResearchThemes(context.Context, ResearchListQuery) (ResearchThemePage, error)
	GetResearchTheme(context.Context, string, ResearchDetailQuery) (ResearchThemeDetail, error)
	ListResearchThemeReasoningTrees(context.Context, string) (ResearchReasoningTreeList, error)
	GetResearchThemeReasoningTree(context.Context, string, string) (ResearchReasoningTreeDetail, error)
}

type ResearchListQuery struct {
	WindowHours int
	Limit       int
	Cursor      string
}

type ResearchDetailQuery struct {
	WindowHours int
}

type ImpactLevel string

const (
	ImpactLevelHigh  ImpactLevel = "high"
	ImpactLevelFocus ImpactLevel = "focus"
	ImpactLevelWatch ImpactLevel = "watch"
)

type TransmissionStage string

const (
	TransmissionStageIdentification TransmissionStage = "identification"
	TransmissionStageValidation     TransmissionStage = "validation"
	TransmissionStageDiffusion      TransmissionStage = "diffusion"
	TransmissionStageDampening      TransmissionStage = "dampening"
)

type EvidenceRole string

const (
	EvidenceRoleDriver        EvidenceRole = "driver"
	EvidenceRoleSupporting    EvidenceRole = "supporting"
	EvidenceRoleContradicting EvidenceRole = "contradicting"
	EvidenceRoleContext       EvidenceRole = "context"
)

type ImpactDirection string

const (
	ImpactDirectionPositive ImpactDirection = "positive"
	ImpactDirectionNegative ImpactDirection = "negative"
	ImpactDirectionMixed    ImpactDirection = "mixed"
	ImpactDirectionNeutral  ImpactDirection = "neutral"
)

type ChangeDirection string

const (
	ChangeDirectionIncrease  ChangeDirection = "increase"
	ChangeDirectionDecrease  ChangeDirection = "decrease"
	ChangeDirectionMixed     ChangeDirection = "mixed"
	ChangeDirectionUnchanged ChangeDirection = "unchanged"
	ChangeDirectionUncertain ChangeDirection = "uncertain"
)

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
	ID                        string                   `json:"id"`
	Name                      string                   `json:"name"`
	OneLineConclusion         string                   `json:"one_line_conclusion"`
	ImpactLevel               ImpactLevel              `json:"impact_level"`
	TransmissionPath          string                   `json:"transmission_path"`
	TradingDirection          string                   `json:"trading_direction"`
	TransmissionStage         TransmissionStage        `json:"transmission_stage"`
	NextCheckpoint            string                   `json:"next_checkpoint"`
	MarketConfirmationSummary string                   `json:"market_confirmation_summary"`
	PublishedAt               time.Time                `json:"published_at"`
	AffectedChainNodes        []ResearchThemeChainNode `json:"affected_chain_nodes"`
	RelatedIndices            []ResearchIndex          `json:"related_indices"`
	SupportingEventCount      int                      `json:"supporting_event_count"`
	ContradictingEventCount   int                      `json:"contradicting_event_count"`
}

type ResearchThemeDetail struct {
	Theme  ResearchTheme   `json:"theme"`
	Events []ResearchEvent `json:"events"`
}

type ResearchThemeChainNode struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	RelationRole  string `json:"relation_role"`
	ImpactSummary string `json:"impact_summary"`
}

type ResearchIndex struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	ImpactDirection ImpactDirection `json:"impact_direction"`
	ImpactSummary   string          `json:"impact_summary"`
}

type ResearchEvent struct {
	EventID        string       `json:"event_id"`
	Title          string       `json:"title"`
	Summary        string       `json:"summary"`
	EventTime      *time.Time   `json:"event_time,omitempty"`
	EvidenceRole   EvidenceRole `json:"evidence_role"`
	SupportedClaim string       `json:"supported_claim"`
}

type ResearchReasoningTreeChainNode struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ResearchReasoningTreeSummary struct {
	AnchorID        string                         `json:"anchor_id"`
	CenterChainNode ResearchReasoningTreeChainNode `json:"center_chain_node"`
}

type ResearchReasoningTreeList struct {
	Theme          ResearchTheme                  `json:"theme"`
	ReasoningTrees []ResearchReasoningTreeSummary `json:"reasoning_trees"`
}

type ResearchReasoningTreeEvent struct {
	EventID         string       `json:"event_id"`
	Title           string       `json:"title"`
	Summary         string       `json:"summary"`
	EventTime       *time.Time   `json:"event_time,omitempty"`
	EvidenceRole    EvidenceRole `json:"evidence_role"`
	EvidenceSummary string       `json:"evidence_summary"`
}

type ResearchReasoningTreePathNode struct {
	ChainNodeID                   string          `json:"chain_node_id"`
	Name                          string          `json:"name"`
	ChangeDirection               ChangeDirection `json:"change_direction"`
	ChangeSummary                 string          `json:"change_summary"`
	ImpactSummary                 string          `json:"impact_summary"`
	IncomingTransmissionMechanism *string         `json:"incoming_transmission_mechanism"`
}

type ResearchReasoningTree struct {
	AnchorID            string                          `json:"anchor_id"`
	CenterChainNode     ResearchReasoningTreeChainNode  `json:"center_chain_node"`
	OneLineConclusion   string                          `json:"one_line_conclusion"`
	FactSummary         string                          `json:"fact_summary"`
	NetDirectionSummary string                          `json:"net_direction_summary"`
	SupportSummary      string                          `json:"support_summary"`
	CounterSummary      *string                         `json:"counter_summary"`
	TradingDirection    string                          `json:"trading_direction"`
	NextCheckpoint      string                          `json:"next_checkpoint"`
	EventCount          int                             `json:"event_count"`
	Events              []ResearchReasoningTreeEvent    `json:"events"`
	PathNodes           []ResearchReasoningTreePathNode `json:"path_nodes"`
}

type ResearchReasoningTreeDetail struct {
	ThemeID       string                `json:"theme_id"`
	ReasoningTree ResearchReasoningTree `json:"reasoning_tree"`
}

var ErrFakeMethodNotConfigured = errors.New("data service fake method is not configured")

type ErrorKind string

const (
	ErrorKindClient     ErrorKind = "client"
	ErrorKindConflict   ErrorKind = "conflict"
	ErrorKindServer     ErrorKind = "server"
	ErrorKindConnection ErrorKind = "connection"
	ErrorKindTimeout    ErrorKind = "timeout"
	ErrorKindCanceled   ErrorKind = "canceled"
	ErrorKindProtocol   ErrorKind = "protocol"
	ErrorKindEncode     ErrorKind = "encode"
	ErrorKindDecode     ErrorKind = "decode"
)

// Error is the sanitized failure contract between the ResearchRepo adapter and
// the Miniapp business layer.
type Error struct {
	Kind       ErrorKind
	StatusCode int
	Code       string
	RequestID  string
}

func (e *Error) Error() string {
	if e == nil {
		return "data service request failed"
	}
	message := "data service request failed: kind=" + string(e.Kind)
	if e.StatusCode != 0 {
		message += " status=" + strconv.Itoa(e.StatusCode)
	}
	if code := safeMetadata(e.Code, 100); code != "" {
		message += " code=" + code
	}
	if requestID := safeMetadata(e.RequestID, 128); requestID != "" {
		message += " request_id=" + requestID
	}
	return message
}

func safeMetadata(value string, maxLength int) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > maxLength {
		return ""
	}
	for _, character := range value {
		switch {
		case character >= 'a' && character <= 'z':
		case character >= 'A' && character <= 'Z':
		case character >= '0' && character <= '9':
		case character == '-', character == '_', character == '.', character == ':':
		default:
			return ""
		}
	}
	return value
}

// Fake keeps Miniapp orchestration tests independent from HTTP and databases.
type Fake struct {
	ListResearchThemesFunc              func(context.Context, ResearchListQuery) (ResearchThemePage, error)
	GetResearchThemeFunc                func(context.Context, string, ResearchDetailQuery) (ResearchThemeDetail, error)
	ListResearchThemeReasoningTreesFunc func(context.Context, string) (ResearchReasoningTreeList, error)
	GetResearchThemeReasoningTreeFunc   func(context.Context, string, string) (ResearchReasoningTreeDetail, error)
}

func (f *Fake) ListResearchThemes(ctx context.Context, query ResearchListQuery) (ResearchThemePage, error) {
	if f == nil || f.ListResearchThemesFunc == nil {
		return ResearchThemePage{}, ErrFakeMethodNotConfigured
	}
	return f.ListResearchThemesFunc(ctx, query)
}

func (f *Fake) GetResearchTheme(ctx context.Context, id string, query ResearchDetailQuery) (ResearchThemeDetail, error) {
	if f == nil || f.GetResearchThemeFunc == nil {
		return ResearchThemeDetail{}, ErrFakeMethodNotConfigured
	}
	return f.GetResearchThemeFunc(ctx, id, query)
}

func (f *Fake) ListResearchThemeReasoningTrees(ctx context.Context, themeID string) (ResearchReasoningTreeList, error) {
	if f == nil || f.ListResearchThemeReasoningTreesFunc == nil {
		return ResearchReasoningTreeList{}, ErrFakeMethodNotConfigured
	}
	return f.ListResearchThemeReasoningTreesFunc(ctx, themeID)
}

func (f *Fake) GetResearchThemeReasoningTree(ctx context.Context, themeID, anchorID string) (ResearchReasoningTreeDetail, error) {
	if f == nil || f.GetResearchThemeReasoningTreeFunc == nil {
		return ResearchReasoningTreeDetail{}, ErrFakeMethodNotConfigured
	}
	return f.GetResearchThemeReasoningTreeFunc(ctx, themeID, anchorID)
}
