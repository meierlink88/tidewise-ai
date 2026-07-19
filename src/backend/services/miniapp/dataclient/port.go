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

type AnchorType string

const (
	AnchorTypePolicy          AnchorType = "policy"
	AnchorTypeSupply          AnchorType = "supply"
	AnchorTypeDemand          AnchorType = "demand"
	AnchorTypeTechnology      AnchorType = "technology"
	AnchorTypeCost            AnchorType = "cost"
	AnchorTypeGeopolitics     AnchorType = "geopolitics"
	AnchorTypeMarketStructure AnchorType = "market_structure"
)

type Importance string

const (
	ImportancePrimary    Importance = "primary"
	ImportanceSecondary  Importance = "secondary"
	ImportanceContextual Importance = "contextual"
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
	ID                string                    `json:"id"`
	AnchorType        AnchorType                `json:"anchor_type"`
	Name              string                    `json:"name"`
	OneLineConclusion string                    `json:"one_line_conclusion"`
	Importance        Importance                `json:"importance"`
	TransmissionPath  string                    `json:"transmission_path"`
	TradingDirection  string                    `json:"trading_direction"`
	PublishedAt       time.Time                 `json:"published_at"`
	RelatedChainNodes []ResearchAnchorChainNode `json:"related_chain_nodes"`
	RelatedIndices    []ResearchIndex           `json:"related_indices"`
	RelatedEventCount int                       `json:"related_event_count"`
}

type ResearchAnchorDetail struct {
	Anchor ResearchAnchor  `json:"anchor"`
	Events []ResearchEvent `json:"events"`
}

type ResearchThemeChainNode struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	RelationRole  string `json:"relation_role"`
	ImpactSummary string `json:"impact_summary"`
}

type ResearchAnchorChainNode struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	RelationRole    string `json:"relation_role"`
	RelationSummary string `json:"relation_summary"`
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
