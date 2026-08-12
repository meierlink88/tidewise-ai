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
	InsertThemeImpact(context.Context, PublicationThemeImpactRecord) error
	InsertThemeEvent(context.Context, PublicationThemeEventRecord) error
	InsertSnapshotThemeImpact(context.Context, SnapshotImpactRecord) error
	InsertTreeReceipt(context.Context, ReasonTreeReceipt) error
	InsertSnapshotTreeReceipt(context.Context, SnapshotTreeReceipt) error
	InsertTree(context.Context, ReasonTreeRecord) error
	InsertSnapshotTree(context.Context, SnapshotTreeRecord) error
	InsertTreeEvent(context.Context, ReasonTreeEventRecord) error
	InsertNode(context.Context, NodeRecord) error
	InsertSnapshotNode(context.Context, SnapshotNodeRecord) error
	InsertSignal(context.Context, PublicationSignalRecord) error
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

type PublicationThemeImpactRecord struct {
	ThemeID, ChainNodeEntityID, RelationRole, ImpactDirection string
	ImpactSummary                                             *string
	DisplayOrder                                              int
}

type PublicationThemeEventRecord struct {
	ThemeID, EventID, EvidenceRole string
	SupportedClaim                 *string
	EvidenceIDs                    []string
}

type ReasonTreeRecord struct {
	ID, ThemeID, ImportReceiptID, IndustryChainEntityID, Title, OneLineConclusion string
	DisplayOrder                                                                  int
	FactSummary, TransmissionSummary, ImpactSummary                               *string
	ImpactDirection, ImpactStrength                                               string
	ConclusionBoundarySummary, SupportSummary, CounterSummary                     *string
	InvalidationConditions                                                        []string
	Checkpoints                                                                   []ReasonTreeCheckpoint
}

type ReasonTreeEventRecord struct {
	ReasoningTreeID, EventID, EvidenceRole string
	DisplayOrder                           int
	EvidenceIDs                            []string
}

type ReasonTreeNodeRecord struct {
	ID, ReasoningTreeID, ChainNodeEntityID, ImpactDirection, ImpactStrength string
	Position                                                                int
	StateSummary, ImpactSummary, ReasoningBasisSummary, EvidenceGapSummary  *string
	IncomingIndustryChainGraphEdgeID, IncomingTransmissionTitle             *string
	IncomingTransmissionMechanism, IncomingConditionSummary                 *string
}

type ReasonTreeSignalRecord struct {
	ReasoningTreeNodeID, VariableSignalKey, SignalRole, SignalDirection, DisplaySummary string
	DisplayOrder                                                                        int
}

type ReasonTreeGraphEdgeReference struct {
	ID, IndustryChainEntityID, FromChainNodeEntityID, ToChainNodeEntityID string
}

type ReasonTreeCounts struct {
	ReasoningTrees     int `json:"reasoning_trees"`
	Nodes              int `json:"nodes"`
	EventAssociations  int `json:"event_associations"`
	SignalAssociations int `json:"signal_associations"`
	Receipts           int `json:"receipts"`
}

type ReasonTreeReceipt struct {
	ID, ThemeID, PublisherSubject, PayloadHash string
	ReasoningTreeIDsByIndustryChainEntityID    map[string]string
	Counts                                     ReasonTreeCounts
	PublishedAt, ImportedAt                    time.Time
}

type Receipt struct {
	ID, AnalysisBatchID, PublisherSubject, PayloadHash, ThemeID string
	ThemeKey                                                    string
	ContractVersion                                             int
	PublicationMode                                             string
	ReasoningTreeIDsByIndustryChainEntityID                     map[string]string
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
	ChainNodeIDs, EventIDs, IndustryChainIDs, GraphEdgeIDs, SignalIDs []string
	ImpactIDs, EvidenceIDs, EntityRelationIDs                         []string
	SnapshotEventExistenceOnly                                        bool
}

type ReferenceFacts struct {
	ChainNodeIDs, IndustryChainIDs map[string]TemporalFact
	Events                         map[string]EventFact
	Memberships                    map[string]map[string]TemporalFact
	GraphEdges                     map[string]GraphEdgeFact
	Signals                        map[string]SignalFact
	Impacts                        map[string]ImpactFact
	Evidences                      map[string]EvidenceFact
	EntityRelations                map[string]EntityRelationFact
}

type EventFact struct {
	ID                   string
	KnowledgeAvailableAt time.Time
}

type SignalFact struct {
	ID, SemanticSubmissionID, EventID, SubjectEntityID string
	VariableKey, Direction                             string
	EvidenceIDs                                        map[string]struct{}
	AcceptedAt                                         time.Time
}

type ImpactFact struct {
	ID, SemanticSubmissionID, SourceVariableSignalID, TargetEntityID string
	AffectedVariableKey, AffectedDirection                           string
	SourceEventID, SourceEntityID                                    string
	EvidenceIDs                                                      map[string]struct{}
	AcceptedAt                                                       time.Time
}

type EntityRelationFact struct {
	ID, FromEntityID, ToEntityID string
	TemporalFact
}

type TemporalFact struct {
	CreatedAt, UpdatedAt time.Time
}

type GraphEdgeFact struct {
	ReasonTreeGraphEdgeReference
	TemporalFact
}

type EvidenceFact struct {
	ID, EventID, Hash    string
	KnowledgeAvailableAt time.Time
}

type NodeRecord struct {
	ReasonTreeNodeRecord
	IncomingSourceKind, DirectImpactAssertionID, DirectImpactSemanticSubmissionID *string
	DirectImpactEvidenceID, DirectImpactEvidenceHash                              *string
	DirectImpactAffectedVariableKey, DirectImpactAffectedDirection                *string
	InferenceUpstreamVariableSignalID, InferenceUpstreamDirectImpactAssertionID   *string
	InferenceEntityRelationID                                                     *string
}

type PublicationSignalRecord struct {
	ReasonTreeSignalRecord
	SourceKind                                                       string
	VariableSignalID, SemanticSubmissionID, EvidenceID, EvidenceHash *string
	UpstreamVariableSignalID, UpstreamDirectImpactAssertionID        *string
	EntityRelationID, IndustryChainGraphEdgeID                       *string
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
