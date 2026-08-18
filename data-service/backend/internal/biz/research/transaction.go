package research

import (
	"context"
	"time"
)

type PublicationStore interface {
	InResearchPublicationTransaction(context.Context, func(PublicationTransaction) error) error
}

type PublicationTransaction interface {
	Lock(context.Context, string) error
	Receipt(context.Context, string) (*Receipt, error)
	ReferenceFacts(context.Context, ReferenceQuery) (ReferenceFacts, error)
	InsertThemeReceipt(context.Context, Receipt) error
	InsertTheme(context.Context, PublicationThemeRecord) error
	InsertSnapshotThemeImpact(context.Context, SnapshotImpactRecord) error
	InsertThemeEvent(context.Context, PublicationThemeEventRecord) error
	InsertSnapshotTreeReceipt(context.Context, SnapshotTreeReceipt) error
	InsertSnapshotTree(context.Context, SnapshotTreeRecord) error
	InsertTreeEvent(context.Context, ReasonTreeEventRecord) error
	InsertSnapshotNode(context.Context, SnapshotNodeRecord) error
	InsertSnapshotSignal(context.Context, SnapshotSignalRecord) error
	Verify(context.Context, Receipt) error
}

type PublicationThemeRecord struct {
	ID, ImportReceiptID, AnalysisBatchID, ThemeKey, Title, OneLineConclusion string
	ConclusionDirection, ImpactStrength, TransmissionStage                   string
	InvestmentGuidanceAction, InvestmentGuidanceSummary                      string
	TimeHorizonCategory                                                      string
	AttentionLevel, ConclusionStatus                                         *string
	TimeHorizonSummary, TransmissionSummary, CheckpointSummary, RiskSummary  *string
	AnalysisAsOf, WindowStart, WindowEnd, PublishedAt                        time.Time
}

type PublicationThemeEventRecord struct {
	ThemeID, EventID, EvidenceRole string
	SupportedClaim                 *string
	EvidenceIDs                    []string
}

type ReasonTreeEventRecord struct {
	ReasoningTreeID, EventID, EvidenceRole string
	DisplayOrder                           int
	EvidenceIDs                            []string
}

type ReasonTreeCounts struct {
	ReasoningTrees     int `json:"reasoning_trees"`
	Nodes              int `json:"nodes"`
	EventAssociations  int `json:"event_associations"`
	SignalAssociations int `json:"signal_associations"`
	Receipts           int `json:"receipts"`
}

type Receipt struct {
	ID, AnalysisBatchID, PublisherSubject, PayloadHash, ThemeID string
	ThemeKey                                                    string
	ContractVersion                                             int
	PublicationMode                                             string
	ReasoningTreeIDsByTreeKey                                   map[string]string
	Counts                                                      Counts
	PublishedAt, ImportedAt                                     time.Time
}

type Counts struct {
	Themes                 int `json:"themes"`
	Impacts                int `json:"impacts"`
	ThemeEventAssociations int `json:"theme_event_associations"`
	ReasoningTrees         int `json:"reasoning_trees"`
	Nodes                  int `json:"nodes"`
	TreeEventAssociations  int `json:"tree_event_associations"`
	SignalAssociations     int `json:"signal_associations"`
	Receipts               int `json:"receipts"`
}

type ReferenceQuery struct {
	EventIDs, EvidenceIDs []string
}

type ReferenceFacts struct {
	Events    map[string]EventFact
	Evidences map[string]EvidenceFact
}

type EventFact struct {
	ID                   string
	KnowledgeAvailableAt time.Time
}

type EvidenceFact struct {
	ID, EventID, Hash    string
	KnowledgeAvailableAt time.Time
}

type SnapshotImpactRecord struct {
	ThemeID, NodeKey, DisplayName, RelationRole, ImpactDirection string
	ImpactSummary                                                *string
	DisplayOrder                                                 int
}

type SnapshotTreeReceipt struct {
	ID, ThemeID, PublisherSubject, PayloadHash string
	ReasoningTreeIDsByTreeKey                  map[string]string
	Counts                                     ReasonTreeCounts
	PublishedAt, ImportedAt                    time.Time
}

type SnapshotTreeRecord struct {
	ID, ThemeID, ImportReceiptID, TreeKey, DisplayName, Title, OneLineConclusion string
	DisplayOrder                                                                 int
	FactSummary, TransmissionSummary, ImpactSummary                              *string
	ImpactDirection, ImpactStrength                                              string
	ConclusionBoundarySummary, SupportSummary, CounterSummary                    *string
	InvalidationConditions                                                       []string
	Checkpoints                                                                  []ReasonTreeCheckpoint
}

type SnapshotNodeRecord struct {
	ID, ReasoningTreeID, NodeKey, DisplayName, ImpactDirection, ImpactStrength string
	Position                                                                   int
	StateSummary, ImpactSummary, ReasoningBasisSummary, EvidenceGapSummary     *string
	IncomingTransmissionTitle, IncomingConditionSummary                        *string
	IncomingTransmissionMechanism                                              *string
}

type SnapshotSignalRecord struct {
	ReasoningTreeNodeID, SignalKey, SignalRole, DisplaySummary string
	VariableName, SignalDirection                              *string
	DisplayOrder                                               int
}
