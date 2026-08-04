package biz

import (
	"context"
	"errors"
	"time"
)

type ResearchRepo interface {
	ListResearchThemes(context.Context, ResearchListQuery) (ResearchThemePage, error)
	GetResearchTheme(context.Context, string) (ResearchThemeDetail, error)
	ListResearchThemeReasoningTrees(context.Context, string) (ResearchReasoningTreeList, error)
	GetResearchThemeReasoningTree(context.Context, string, string) (ResearchReasoningTreeDetail, error)
}

type ResearchListQuery struct {
	WindowHours   int
	PublishedFrom *time.Time
	PublishedTo   *time.Time
	Limit         int
	Cursor        string
}

type ResearchThemePage struct {
	WindowStart, WindowEnd, AsOf time.Time
	ThemeCount, EventCount       int
	Items                        []ResearchTheme
	NextCursor                   *string
}

type ResearchTheme struct {
	ID, AnalysisBatchID, Title, OneLineConclusion          string
	ConclusionDirection, ImpactStrength, TransmissionStage string
	InvestmentGuidanceAction, InvestmentGuidanceSummary    string
	TimeHorizonCategory                                    string
	AttentionLevel, ConclusionStatus                       *string
	TimeHorizonSummary, TransmissionSummary                *string
	CheckpointSummary, RiskSummary                         *string
	AnalysisAsOf, WindowStart, WindowEnd, PublishedAt      time.Time
	Impacts                                                []ResearchThemeImpact
	EvidenceEventCount, ReasoningTreeCount                 int
}

type ResearchThemeImpact struct {
	NodeKey, DisplayName                                   string
	ChainNodeEntityID, Name, RelationRole, ImpactDirection string
	ImpactSummary                                          *string
	DisplayOrder                                           int
}

type ResearchThemeDetail struct {
	Theme  ResearchTheme
	Events []ResearchEvent
}

type ResearchEvent struct {
	EventID, Title, Summary, EvidenceRole string
	EvidenceIDs                           []string
	EventTime                             *time.Time
	SupportedClaim                        *string
	DisplayOrder                          int
}

type ResearchReasoningTreeSummary struct {
	TreeKey, DisplayName                                             string
	ReasoningTreeID, IndustryChainEntityID, IndustryChainName, Title string
	DisplayOrder, EventCount                                         int
	PublishedAt                                                      time.Time
}

type ResearchReasoningTreeList struct {
	Theme          ResearchTheme
	ReasoningTrees []ResearchReasoningTreeSummary
}

type ResearchCheckpoint struct{ Type, Summary string }
type ResearchGraphEdge struct{ ID, RelationType, ReviewStatus, Status string }
type ResearchSignal struct {
	SignalKey                                                      string
	VariableName, Direction                                        *string
	VariableSignalKey, SignalRole, SignalDirection, DisplaySummary string
	DisplayOrder                                                   int
}

type ResearchReasoningTreeNode struct {
	NodeKey, DisplayName                                         string
	ID, ChainNodeEntityID, Name, ImpactDirection, ImpactStrength string
	Position                                                     int
	StateSummary, ImpactSummary, ReasoningBasisSummary           *string
	EvidenceGapSummary                                           *string
	IncomingIndustryChainGraphEdgeID, IncomingTransmissionTitle  *string
	IncomingTransmissionMechanism, IncomingConditionSummary      *string
	IncomingGraphEdge                                            *ResearchGraphEdge
	Signals                                                      []ResearchSignal
	PrimarySignal                                                ResearchSignal
	SignalDisplaySummary                                         string
}

type ResearchReasoningTree struct {
	TreeKey, DisplayName                                               string
	ReasoningTreeID, ThemeID, IndustryChainEntityID, IndustryChainName string
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
	ThemeID       string
	ImpactNodeIDs []string
	ReasoningTree ResearchReasoningTree
}

var ErrFakeMethodNotConfigured = errors.New("data service fake method is not configured")

type Fake struct {
	ListResearchThemesFunc              func(context.Context, ResearchListQuery) (ResearchThemePage, error)
	GetResearchThemeFunc                func(context.Context, string) (ResearchThemeDetail, error)
	ListResearchThemeReasoningTreesFunc func(context.Context, string) (ResearchReasoningTreeList, error)
	GetResearchThemeReasoningTreeFunc   func(context.Context, string, string) (ResearchReasoningTreeDetail, error)
}

func (f *Fake) ListResearchThemes(ctx context.Context, query ResearchListQuery) (ResearchThemePage, error) {
	if f == nil || f.ListResearchThemesFunc == nil {
		return ResearchThemePage{}, ErrFakeMethodNotConfigured
	}
	return f.ListResearchThemesFunc(ctx, query)
}
func (f *Fake) GetResearchTheme(ctx context.Context, id string) (ResearchThemeDetail, error) {
	if f == nil || f.GetResearchThemeFunc == nil {
		return ResearchThemeDetail{}, ErrFakeMethodNotConfigured
	}
	return f.GetResearchThemeFunc(ctx, id)
}
func (f *Fake) ListResearchThemeReasoningTrees(ctx context.Context, id string) (ResearchReasoningTreeList, error) {
	if f == nil || f.ListResearchThemeReasoningTreesFunc == nil {
		return ResearchReasoningTreeList{}, ErrFakeMethodNotConfigured
	}
	return f.ListResearchThemeReasoningTreesFunc(ctx, id)
}
func (f *Fake) GetResearchThemeReasoningTree(ctx context.Context, themeID, treeID string) (ResearchReasoningTreeDetail, error) {
	if f == nil || f.GetResearchThemeReasoningTreeFunc == nil {
		return ResearchReasoningTreeDetail{}, ErrFakeMethodNotConfigured
	}
	return f.GetResearchThemeReasoningTreeFunc(ctx, themeID, treeID)
}
