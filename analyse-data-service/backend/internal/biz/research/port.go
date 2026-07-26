package research

import (
	"context"
	"errors"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/model"
)

var (
	ErrResearchNotFound               = errors.New("research result not found")
	ErrResearchThemeNotFound          = errors.New("research theme not found")
	ErrResearchReasoningTreesNotFound = errors.New("research reasoning trees not found")
	ErrResearchReasoningTreeNotFound  = errors.New("research reasoning tree not found")
	ErrResearchReasoningTreeInvariant = errors.New("research reasoning tree invariant violation")
)

type ChainNodeRecord struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	RelationRole string `json:"relation_role"`
	Summary      string `json:"impact_summary"`
}

type IndexRecord struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	ImpactDirection string `json:"impact_direction"`
	Summary         string `json:"impact_summary"`
}

type EventRecord struct {
	EventID        string     `json:"event_id"`
	Title          string     `json:"title"`
	Summary        string     `json:"summary"`
	EventTime      *time.Time `json:"event_time,omitempty"`
	EvidenceRole   string     `json:"evidence_role"`
	SupportedClaim string     `json:"supported_claim"`
}

type ThemeSummaryRecord struct {
	ID, Name, OneLineConclusion                   string
	ImpactLevel                                   model.ImpactLevel
	TransmissionPath, TradingDirection            string
	TransmissionStage                             model.TransmissionStage
	NextCheckpoint, MarketConfirmationSummary     string
	PublishedAt                                   time.Time
	ChainNodes                                    []ChainNodeRecord
	Indices                                       []IndexRecord
	SupportingEventCount, ContradictingEventCount int
}

type ThemeDetailRecord struct {
	ThemeSummaryRecord
	Events []EventRecord
}

type ThemeListFilter struct {
	WindowStart, AsOf time.Time
	Limit, CursorRank int
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
	AnchorID            string `json:"anchor_id"`
	CenterChainNodeID   string `json:"center_chain_node_id"`
	CenterChainNodeName string `json:"center_chain_node_name"`
}

type ReasoningTreeListRecord struct {
	Theme          ThemeSummaryRecord
	ReasoningTrees []ReasoningTreeSummaryRecord
}

type ReasoningTreeEventRecord struct {
	EventID         string     `json:"event_id"`
	Title           string     `json:"title"`
	Summary         string     `json:"summary"`
	EventTime       *time.Time `json:"event_time"`
	EvidenceRole    string     `json:"evidence_role"`
	EvidenceSummary string     `json:"evidence_summary"`
}

type ReasoningTreePathNodeRecord struct {
	Position                      int     `json:"position"`
	ChainNodeID                   string  `json:"chain_node_id"`
	Name                          string  `json:"name"`
	ChangeDirection               string  `json:"change_direction"`
	ChangeSummary                 string  `json:"change_summary"`
	ImpactSummary                 string  `json:"impact_summary"`
	IncomingTransmissionMechanism *string `json:"incoming_transmission_mechanism"`
}

type ReasoningTreeRecord struct {
	AnchorID, CenterChainNodeID, CenterChainNodeName    string
	OneLineConclusion, FactSummary, NetDirectionSummary string
	SupportSummary                                      string
	CounterSummary                                      *string
	TradingDirection, NextCheckpoint                    string
	Events                                              []ReasoningTreeEventRecord
	PathNodes                                           []ReasoningTreePathNodeRecord
}

type ReasoningTreeDetailRecord struct {
	ThemeID       string
	ReasoningTree ReasoningTreeRecord
}

type Repository interface {
	ListResearchThemes(context.Context, ThemeListFilter) (ThemeStorePage, error)
	GetResearchTheme(context.Context, string, DetailFilter) (ThemeDetailRecord, error)
	ListResearchThemeReasoningTrees(context.Context, string) (ReasoningTreeListRecord, error)
	GetResearchThemeReasoningTree(context.Context, string, string) (ReasoningTreeDetailRecord, error)
}
