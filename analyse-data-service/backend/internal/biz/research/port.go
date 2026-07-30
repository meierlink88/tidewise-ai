package research

import (
	"context"
	"errors"
	"time"
)

var (
	ErrResearchNotFound               = errors.New("research result not found")
	ErrResearchThemeNotFound          = errors.New("research theme not found")
	ErrResearchReasoningTreesNotFound = errors.New("research reasoning trees not found")
	ErrResearchReasoningTreeNotFound  = errors.New("research reasoning tree not found")
	ErrResearchReasoningTreeInvariant = errors.New("research reasoning tree invariant violation")
)

type ThemeImpactRecord struct {
	ChainNodeEntityID           string  `json:"chain_node_entity_id"`
	Name                        string  `json:"name"`
	RelationRole                string  `json:"relation_role"`
	ImpactDirection             string  `json:"impact_direction"`
	ImpactSummary               *string `json:"impact_summary"`
	PrimarySignalDisplaySummary *string `json:"primary_signal_display_summary"`
	DisplayOrder                int     `json:"display_order"`
}

type EventRecord struct {
	EventID        string     `json:"event_id"`
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
	Events []EventRecord
}

type ThemeListFilter struct {
	WindowStart, AsOf time.Time
	Limit             int
	CursorPublishedAt *time.Time
	CursorID          string
}

type DetailFilter struct {
	WindowStart, AsOf time.Time
}

type ThemeStorePage struct {
	AsOf, WindowStart, WindowEnd time.Time
	ThemeCount, EventCount       int
	Items                        []ThemeSummaryRecord
	HasMore                      bool
}

type ReasoningTreeSummaryRecord struct {
	ReasoningTreeID, IndustryChainEntityID, IndustryChainName, Title string
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
	DisplayOrder                                                   int
}

type ReasoningTreeNodeRecord struct {
	ID, ChainNodeEntityID, Name, ImpactDirection, ImpactStrength string
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
	ThemeID       string
	ImpactNodeIDs []string
	ReasoningTree ReasoningTreeRecord
}

type Repository interface {
	ListResearchThemes(context.Context, ThemeListFilter) (ThemeStorePage, error)
	GetResearchTheme(context.Context, string, DetailFilter) (ThemeDetailRecord, error)
	ListResearchThemeReasoningTrees(context.Context, string) (ReasoningTreeListRecord, error)
	GetResearchThemeReasoningTree(context.Context, string, string) (ReasoningTreeDetailRecord, error)
}
