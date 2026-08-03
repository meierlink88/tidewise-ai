package researchpublication

import (
	"context"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchreasoningtreeimport"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchthemeimport"
)

type Store interface {
	InResearchPublicationTransaction(context.Context, func(Transaction) error) error
}

type Transaction interface {
	Lock(context.Context, string) error
	Receipt(context.Context, string) (*Receipt, error)
	ReferenceFacts(context.Context, ReferenceQuery) (ReferenceFacts, error)
	InsertThemeReceipt(context.Context, Receipt) error
	InsertTheme(context.Context, researchthemeimport.ThemeRecord) error
	InsertThemeImpact(context.Context, researchthemeimport.ImpactRecord) error
	InsertThemeEvent(context.Context, researchthemeimport.EventRecord) error
	InsertSnapshotThemeImpact(context.Context, SnapshotImpactRecord) error
	InsertTreeReceipt(context.Context, researchreasoningtreeimport.Receipt) error
	InsertSnapshotTreeReceipt(context.Context, SnapshotTreeReceipt) error
	InsertTree(context.Context, researchreasoningtreeimport.ReasoningTreeRecord) error
	InsertSnapshotTree(context.Context, SnapshotTreeRecord) error
	InsertTreeEvent(context.Context, researchreasoningtreeimport.EventRecord) error
	InsertNode(context.Context, NodeRecord) error
	InsertSnapshotNode(context.Context, SnapshotNodeRecord) error
	InsertSignal(context.Context, SignalRecord) error
	InsertSnapshotSignal(context.Context, SnapshotSignalRecord) error
	Verify(context.Context, Receipt) error
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
	researchreasoningtreeimport.GraphEdgeReference
	TemporalFact
}

type EvidenceFact struct {
	ID, EventID, Hash    string
	KnowledgeAvailableAt time.Time
}

type NodeRecord struct {
	researchreasoningtreeimport.NodeRecord
	IncomingSourceKind, DirectImpactAssertionID, DirectImpactSemanticSubmissionID *string
	DirectImpactEvidenceID, DirectImpactEvidenceHash                              *string
	DirectImpactAffectedVariableKey, DirectImpactAffectedDirection                *string
	InferenceUpstreamVariableSignalID, InferenceUpstreamDirectImpactAssertionID   *string
	InferenceEntityRelationID                                                     *string
}

type SignalRecord struct {
	researchreasoningtreeimport.SignalRecord
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
	Counts                                     researchreasoningtreeimport.Counts
	PublishedAt, ImportedAt                    time.Time
}

type SnapshotTreeRecord struct {
	ID, ThemeID, ImportReceiptID, TreeKey, DisplayName, Title, OneLineConclusion string
	DisplayOrder                                                                 int
	FactSummary, TransmissionSummary, ImpactSummary                              *string
	ImpactDirection, ImpactStrength                                              string
	ConclusionBoundarySummary, SupportSummary, CounterSummary                    *string
	InvalidationConditions                                                       []string
	Checkpoints                                                                  []researchreasoningtreeimport.Checkpoint
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
