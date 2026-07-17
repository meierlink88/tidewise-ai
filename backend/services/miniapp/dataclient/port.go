package dataclient

import (
	"context"
	"errors"
	"time"
)

const (
	DataAPIPrefix       = "/internal/data/v1"
	ResearchThemesPath  = DataAPIPrefix + "/research/themes"
	ResearchAnchorsPath = DataAPIPrefix + "/research/anchors"
)

// DataServiceClient is the Miniapp-owned boundary for page-level research data.
// Its DTOs intentionally remain local to this consumer.
type DataServiceClient interface {
	ListResearchThemes(context.Context, ResearchListQuery) (ResearchThemePage, error)
	GetResearchTheme(context.Context, string, ResearchDetailQuery) (ResearchThemeDetail, error)
	ListResearchAnchors(context.Context, ResearchListQuery) (ResearchAnchorPage, error)
	GetResearchAnchor(context.Context, string, ResearchDetailQuery) (ResearchAnchorDetail, error)
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
	ImpactLevelHigh   ImpactLevel = "high"
	ImpactLevelMedium ImpactLevel = "medium"
	ImpactLevelLow    ImpactLevel = "low"
)

type TradingDirection string

const (
	TradingDirectionPositive  TradingDirection = "positive"
	TradingDirectionNegative  TradingDirection = "negative"
	TradingDirectionMixed     TradingDirection = "mixed"
	TradingDirectionNeutral   TradingDirection = "neutral"
	TradingDirectionUncertain TradingDirection = "uncertain"
)

type TransmissionStage string

const (
	TransmissionStageEmerging   TransmissionStage = "emerging"
	TransmissionStageDeveloping TransmissionStage = "developing"
	TransmissionStageMature     TransmissionStage = "mature"
	TransmissionStageFading     TransmissionStage = "fading"
)

type AnchorType string

const (
	AnchorTypeEntity        AnchorType = "entity"
	AnchorTypeMarket        AnchorType = "market"
	AnchorTypeIndex         AnchorType = "index"
	AnchorTypePolicy        AnchorType = "policy"
	AnchorTypeIndustryChain AnchorType = "industry_chain"
)

type Importance string

const (
	ImportanceHigh   Importance = "high"
	ImportanceMedium Importance = "medium"
	ImportanceLow    Importance = "low"
)

type EvidenceRole string

const (
	EvidenceRoleSupports    EvidenceRole = "supports"
	EvidenceRoleContradicts EvidenceRole = "contradicts"
	EvidenceRoleContext     EvidenceRole = "context"
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
	ID                      string              `json:"id"`
	Name                    string              `json:"name"`
	OneLineConclusion       string              `json:"one_line_conclusion"`
	ImpactLevel             ImpactLevel         `json:"impact_level"`
	TransmissionPath        string              `json:"transmission_path"`
	TradingDirection        TradingDirection    `json:"trading_direction"`
	TransmissionStage       TransmissionStage   `json:"transmission_stage"`
	NextCheckpoint          string              `json:"next_checkpoint"`
	IndexImpactSummary      string              `json:"index_impact_summary,omitempty"`
	PublishedAt             time.Time           `json:"published_at"`
	AffectedChainNodes      []ResearchChainNode `json:"affected_chain_nodes"`
	RelatedIndices          []ResearchIndex     `json:"related_indices"`
	SupportingEventCount    int                 `json:"supporting_event_count"`
	ContradictingEventCount int                 `json:"contradicting_event_count"`
	HasMoreDetail           bool                `json:"has_more_detail"`
}

type ResearchThemeDetail struct {
	Theme  ResearchTheme   `json:"theme"`
	Events []ResearchEvent `json:"events"`
}

type ResearchAnchorPage struct {
	WindowStart time.Time        `json:"window_start"`
	WindowEnd   time.Time        `json:"window_end"`
	AsOf        time.Time        `json:"as_of"`
	AnchorCount int              `json:"anchor_count"`
	EventCount  int              `json:"event_count"`
	Items       []ResearchAnchor `json:"items"`
	NextCursor  *string          `json:"next_cursor"`
}

type ResearchAnchor struct {
	ID                string              `json:"id"`
	AnchorType        AnchorType          `json:"anchor_type"`
	Name              string              `json:"name"`
	OneLineConclusion string              `json:"one_line_conclusion"`
	Importance        Importance          `json:"importance"`
	TransmissionPath  string              `json:"transmission_path"`
	TradingDirection  TradingDirection    `json:"trading_direction"`
	PublishedAt       time.Time           `json:"published_at"`
	RelatedChainNodes []ResearchChainNode `json:"related_chain_nodes"`
	RelatedIndices    []ResearchIndex     `json:"related_indices"`
	RelatedEventCount int                 `json:"related_event_count"`
}

type ResearchAnchorDetail struct {
	Anchor ResearchAnchor  `json:"anchor"`
	Events []ResearchEvent `json:"events"`
}

type ResearchChainNode struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	RelationRole string `json:"relation_role"`
	Summary      string `json:"impact_summary"`
}

type ResearchIndex struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	ImpactDirection TradingDirection `json:"impact_direction"`
	Summary         string           `json:"impact_summary"`
}

type ResearchEvent struct {
	EventID        string       `json:"event_id"`
	Title          string       `json:"title"`
	Summary        string       `json:"summary"`
	EventTime      *time.Time   `json:"event_time,omitempty"`
	EvidenceRole   EvidenceRole `json:"evidence_role"`
	SupportedClaim string       `json:"supported_claim"`
}

var ErrFakeMethodNotConfigured = errors.New("data service fake method is not configured")

// Fake keeps Miniapp orchestration tests independent from HTTP and databases.
type Fake struct {
	ListResearchThemesFunc  func(context.Context, ResearchListQuery) (ResearchThemePage, error)
	GetResearchThemeFunc    func(context.Context, string, ResearchDetailQuery) (ResearchThemeDetail, error)
	ListResearchAnchorsFunc func(context.Context, ResearchListQuery) (ResearchAnchorPage, error)
	GetResearchAnchorFunc   func(context.Context, string, ResearchDetailQuery) (ResearchAnchorDetail, error)
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

func (f *Fake) ListResearchAnchors(ctx context.Context, query ResearchListQuery) (ResearchAnchorPage, error) {
	if f == nil || f.ListResearchAnchorsFunc == nil {
		return ResearchAnchorPage{}, ErrFakeMethodNotConfigured
	}
	return f.ListResearchAnchorsFunc(ctx, query)
}

func (f *Fake) GetResearchAnchor(ctx context.Context, id string, query ResearchDetailQuery) (ResearchAnchorDetail, error) {
	if f == nil || f.GetResearchAnchorFunc == nil {
		return ResearchAnchorDetail{}, ErrFakeMethodNotConfigured
	}
	return f.GetResearchAnchorFunc(ctx, id, query)
}
