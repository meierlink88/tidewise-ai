package researchreasoningtreeimport

import (
	"context"
	"time"
)

type Store interface {
	InResearchReasoningTreeImportTransaction(context.Context, func(Transaction) error) error
}

type Transaction interface {
	LockResearchReasoningTreeImportTheme(context.Context, string) error
	LockResearchReasoningTreeAnalysisBatch(context.Context, string) error
	ResearchReasoningTreeImportReceipt(context.Context, string) (*Receipt, error)
	ResearchReasoningTreeImportThemePublication(context.Context, string) (*ThemePublication, error)
	ResearchReasoningTreeSignalSnapshots(context.Context, string, []string) (map[string]SignalSnapshot, error)
	ExistingResearchReasoningTreeIndustryChains(context.Context, []string) (map[string]struct{}, error)
	ResearchReasoningTreeChainMemberships(context.Context, []string) (map[string]map[string]struct{}, error)
	ResearchReasoningTreeGraphEdges(context.Context, []string) (map[string]GraphEdgeReference, error)
	InsertResearchReasoningTreeImportReceipt(context.Context, Receipt) error
	InsertResearchReasoningTree(context.Context, ReasoningTreeRecord) error
	InsertResearchReasoningTreeEvent(context.Context, EventRecord) error
	InsertResearchReasoningTreeNode(context.Context, NodeRecord) error
	InsertResearchReasoningTreeNodeSignal(context.Context, SignalRecord) error
	VerifyResearchReasoningTreeImportReceipt(context.Context, Receipt) error
}

type ThemePublication struct {
	ThemeID, AnalysisBatchID, ThemeImportReceiptID, PublisherSubject string
	ImpactNodeIDs, EventIDs                                          map[string]struct{}
}

type GraphEdgeReference struct {
	ID, IndustryChainEntityID, FromChainNodeEntityID, ToChainNodeEntityID string
}

type ReasoningTreeRecord struct {
	ID, ThemeID, ImportReceiptID, IndustryChainEntityID, Title, OneLineConclusion string
	DisplayOrder                                                                  int
	FactSummary, TransmissionSummary, ImpactSummary                               *string
	ImpactDirection, ImpactStrength                                               string
	ConclusionBoundarySummary, SupportSummary, CounterSummary                     *string
	InvalidationConditions                                                        []string
	Checkpoints                                                                   []Checkpoint
}

type EventRecord struct {
	ReasoningTreeID, EventID, EvidenceRole string
	DisplayOrder                           int
	EvidenceIDs                            []string
}

type NodeRecord struct {
	ID, ReasoningTreeID, ChainNodeEntityID, ImpactDirection, ImpactStrength string
	Position                                                                int
	StateSummary, ImpactSummary, ReasoningBasisSummary, EvidenceGapSummary  *string
	IncomingIndustryChainGraphEdgeID, IncomingTransmissionTitle             *string
	IncomingTransmissionMechanism, IncomingConditionSummary                 *string
}

type SignalRecord struct {
	ReasoningTreeNodeID, VariableSignalKey, SignalRole, SignalDirection, DisplaySummary string
	DisplayOrder                                                                        int
}

type SignalSnapshot struct {
	SignalDirection, DisplaySummary string
}

type Counts struct {
	ReasoningTrees     int `json:"reasoning_trees"`
	Nodes              int `json:"nodes"`
	EventAssociations  int `json:"event_associations"`
	SignalAssociations int `json:"signal_associations"`
	Receipts           int `json:"receipts"`
}

type Receipt struct {
	ID, ThemeID, PublisherSubject, PayloadHash string
	ReasoningTreeIDsByIndustryChainEntityID    map[string]string
	Counts                                     Counts
	PublishedAt, ImportedAt                    time.Time
}
