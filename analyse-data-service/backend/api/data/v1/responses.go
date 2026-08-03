package v1

import "time"

type EventPublicationResult struct {
	ReceiptID    string                              `json:"receipt_id"`
	PackageID    string                              `json:"package_id"`
	ImportedAt   time.Time                           `json:"imported_at"`
	Events       []EventPublicationEventResult       `json:"events"`
	RawDocuments []EventPublicationRawDocumentResult `json:"raw_documents"`
	Counts       EventPublicationCounts              `json:"counts"`
}

type EventPublicationEventResult struct {
	DedupeKey   string `json:"dedupe_key"`
	EventID     string `json:"event_id"`
	Disposition string `json:"disposition"`
}

type EventPublicationRawDocumentResult struct {
	ArtifactID    string `json:"artifact_id"`
	RawDocumentID string `json:"raw_document_id"`
	Disposition   string `json:"disposition"`
}

type EventPublicationCounts struct {
	EventsCreated       int `json:"events_created"`
	EventsReused        int `json:"events_reused"`
	RawDocumentsCreated int `json:"raw_documents_created"`
	RawDocumentsReused  int `json:"raw_documents_reused"`
	EventSourcesCreated int `json:"event_sources_created"`
	EventSourcesReused  int `json:"event_sources_reused"`
	EventTagsCreated    int `json:"event_tags_created"`
	EventTagsReused     int `json:"event_tags_reused"`
}

type EventTagCatalog struct {
	CatalogRevision string                `json:"catalog_revision"`
	CatalogHash     string                `json:"catalog_hash"`
	Tags            []EventTagCatalogItem `json:"tags"`
}

type EventTagCatalogItem struct {
	ID       string `json:"id"`
	TagKind  string `json:"tag_kind"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	IsActive bool   `json:"is_active"`
}

type ResearchThemeImportResult struct {
	ReceiptID                               string                    `json:"receipt_id"`
	AnalysisBatchID                         string                    `json:"analysis_batch_id"`
	PayloadHash                             string                    `json:"payload_hash"`
	ThemeID                                 string                    `json:"theme_id"`
	PublicationMode                         string                    `json:"publication_mode"`
	ReasoningTreeIDsByIndustryChainEntityID map[string]string         `json:"reasoning_tree_ids_by_industry_chain_entity_id"`
	ReasoningTreeIDsByTreeKey               map[string]string         `json:"reasoning_tree_ids_by_tree_key"`
	Counts                                  ResearchThemeImportCounts `json:"counts"`
	PublishedAt                             time.Time                 `json:"published_at"`
	ImportedAt                              time.Time                 `json:"imported_at"`
	Replayed                                bool                      `json:"replayed"`
}

type ResearchThemeImportCounts struct {
	Themes                 int `json:"themes"`
	Impacts                int `json:"impacts"`
	ThemeEventAssociations int `json:"theme_event_associations"`
	ReasoningTrees         int `json:"reasoning_trees"`
	Nodes                  int `json:"nodes"`
	TreeEventAssociations  int `json:"tree_event_associations"`
	SignalAssociations     int `json:"signal_associations"`
	Receipts               int `json:"receipts"`
}

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
	ID                        string                `json:"id"`
	AnalysisBatchID           string                `json:"analysis_batch_id"`
	Title                     string                `json:"title"`
	OneLineConclusion         string                `json:"one_line_conclusion"`
	ConclusionDirection       string                `json:"conclusion_direction"`
	ImpactStrength            string                `json:"impact_strength"`
	AttentionLevel            *string               `json:"attention_level"`
	ConclusionStatus          *string               `json:"conclusion_status"`
	TransmissionStage         string                `json:"transmission_stage"`
	InvestmentGuidanceAction  string                `json:"investment_guidance_action"`
	InvestmentGuidanceSummary string                `json:"investment_guidance_summary"`
	TimeHorizonCategory       string                `json:"time_horizon_category"`
	TimeHorizonSummary        *string               `json:"time_horizon_summary"`
	TransmissionSummary       *string               `json:"transmission_summary"`
	CheckpointSummary         *string               `json:"checkpoint_summary"`
	RiskSummary               *string               `json:"risk_summary"`
	AnalysisAsOf              time.Time             `json:"analysis_as_of"`
	WindowStart               time.Time             `json:"window_start"`
	WindowEnd                 time.Time             `json:"window_end"`
	PublishedAt               time.Time             `json:"published_at"`
	Impacts                   []ResearchThemeImpact `json:"impacts"`
	EvidenceEventCount        int                   `json:"evidence_event_count"`
	ReasoningTreeCount        int                   `json:"reasoning_tree_count"`
}

type ResearchThemeDetail struct {
	ThemeKey                   string          `json:"theme_key"`
	PublicationMode            string          `json:"publication_mode"`
	PublicationContractVersion int             `json:"publication_contract_version"`
	Theme                      ResearchTheme   `json:"theme"`
	Events                     []ResearchEvent `json:"events"`
}

type ResearchThemeImpact struct {
	NodeKey           string  `json:"node_key"`
	DisplayName       string  `json:"display_name"`
	ChainNodeEntityID string  `json:"chain_node_entity_id"`
	Name              string  `json:"name"`
	RelationRole      string  `json:"relation_role"`
	ImpactDirection   string  `json:"impact_direction"`
	ImpactSummary     *string `json:"impact_summary"`
	DisplayOrder      int     `json:"display_order"`
}

type ResearchEvent struct {
	EventID        string     `json:"event_id"`
	EvidenceIDs    []string   `json:"evidence_ids"`
	Title          string     `json:"title"`
	Summary        string     `json:"summary"`
	EventTime      *time.Time `json:"event_time"`
	EvidenceRole   string     `json:"evidence_role"`
	SupportedClaim *string    `json:"supported_claim"`
	DisplayOrder   int        `json:"display_order"`
}

type ResearchReasoningTreeSummary struct {
	TreeKey               string    `json:"tree_key"`
	DisplayName           string    `json:"display_name"`
	ReasoningTreeID       string    `json:"reasoning_tree_id"`
	IndustryChainEntityID string    `json:"industry_chain_entity_id"`
	IndustryChainName     string    `json:"industry_chain_name"`
	Title                 string    `json:"title"`
	DisplayOrder          int       `json:"display_order"`
	EventCount            int       `json:"event_count"`
	PublishedAt           time.Time `json:"published_at"`
}

type ResearchReasoningTreeList struct {
	Theme          ResearchTheme                  `json:"theme"`
	ReasoningTrees []ResearchReasoningTreeSummary `json:"reasoning_trees"`
}

type ResearchReasoningTreeCheckpoint struct {
	Type    string `json:"type"`
	Summary string `json:"summary"`
}

type ResearchReasoningTreeGraphEdge struct {
	ID           string `json:"id"`
	RelationType string `json:"relation_type"`
	ReviewStatus string `json:"review_status"`
	Status       string `json:"status"`
}

type ResearchReasoningTreeSignal struct {
	SignalKey         string  `json:"signal_key"`
	VariableName      *string `json:"variable_name"`
	Direction         *string `json:"direction"`
	VariableSignalKey string  `json:"variable_signal_key"`
	SignalRole        string  `json:"signal_role"`
	SignalDirection   string  `json:"signal_direction"`
	DisplaySummary    string  `json:"display_summary"`
	DisplayOrder      int     `json:"display_order"`
}

type ResearchReasoningTreeNode struct {
	NodeKey                          string                          `json:"node_key"`
	DisplayName                      string                          `json:"display_name"`
	ID                               string                          `json:"id"`
	Position                         int                             `json:"position"`
	ChainNodeEntityID                string                          `json:"chain_node_entity_id"`
	Name                             string                          `json:"name"`
	StateSummary                     *string                         `json:"state_summary"`
	ImpactDirection                  string                          `json:"impact_direction"`
	ImpactStrength                   string                          `json:"impact_strength"`
	ImpactSummary                    *string                         `json:"impact_summary"`
	ReasoningBasisSummary            *string                         `json:"reasoning_basis_summary"`
	EvidenceGapSummary               *string                         `json:"evidence_gap_summary"`
	IncomingIndustryChainGraphEdgeID *string                         `json:"incoming_industry_chain_graph_edge_id"`
	IncomingTransmissionTitle        *string                         `json:"incoming_transmission_title"`
	IncomingTransmissionMechanism    *string                         `json:"incoming_transmission_mechanism"`
	IncomingConditionSummary         *string                         `json:"incoming_condition_summary"`
	IncomingGraphEdge                *ResearchReasoningTreeGraphEdge `json:"incoming_graph_edge"`
	Signals                          []ResearchReasoningTreeSignal   `json:"signals"`
	PrimarySignal                    ResearchReasoningTreeSignal     `json:"primary_signal"`
	SignalDisplaySummary             string                          `json:"signal_display_summary"`
}

type ResearchReasoningTree struct {
	TreeKey                   string                            `json:"tree_key"`
	DisplayName               string                            `json:"display_name"`
	ReasoningTreeID           string                            `json:"reasoning_tree_id"`
	ThemeID                   string                            `json:"theme_id"`
	IndustryChainEntityID     string                            `json:"industry_chain_entity_id"`
	IndustryChainName         string                            `json:"industry_chain_name"`
	Title                     string                            `json:"title"`
	DisplayOrder              int                               `json:"display_order"`
	OneLineConclusion         string                            `json:"one_line_conclusion"`
	FactSummary               *string                           `json:"fact_summary"`
	TransmissionSummary       *string                           `json:"transmission_summary"`
	ImpactDirection           string                            `json:"impact_direction"`
	ImpactStrength            string                            `json:"impact_strength"`
	ImpactSummary             *string                           `json:"impact_summary"`
	ConclusionBoundarySummary *string                           `json:"conclusion_boundary_summary"`
	SupportSummary            *string                           `json:"support_summary"`
	CounterSummary            *string                           `json:"counter_summary"`
	InvalidationConditions    []string                          `json:"invalidation_conditions"`
	Checkpoints               []ResearchReasoningTreeCheckpoint `json:"checkpoints"`
	PublishedAt               time.Time                         `json:"published_at"`
	EventCount                int                               `json:"event_count"`
	Events                    []ResearchEvent                   `json:"events"`
	Nodes                     []ResearchReasoningTreeNode       `json:"nodes"`
}

type ResearchReasoningTreeDetail struct {
	ThemeID                    string                `json:"theme_id"`
	ThemeKey                   string                `json:"theme_key"`
	PublicationMode            string                `json:"publication_mode"`
	PublicationContractVersion int                   `json:"publication_contract_version"`
	ImpactNodeIDs              []string              `json:"impact_node_ids"`
	ReasoningTree              ResearchReasoningTree `json:"reasoning_tree"`
}

type AdminRawDocumentPage struct {
	Items    []AdminRawDocument `json:"items"`
	Total    int                `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
}

type AdminEventPage struct {
	Items    []AdminEvent `json:"items"`
	Total    int          `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
}
