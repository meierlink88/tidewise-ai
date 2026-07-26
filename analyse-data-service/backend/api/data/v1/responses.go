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
	ReceiptID       string                    `json:"receipt_id"`
	AnalysisBatchID string                    `json:"analysis_batch_id"`
	PayloadHash     string                    `json:"payload_hash"`
	ThemeIDsByKey   map[string]string         `json:"theme_ids_by_key"`
	Counts          ResearchThemeImportCounts `json:"counts"`
	PublishedAt     time.Time                 `json:"published_at"`
	ImportedAt      time.Time                 `json:"imported_at"`
	Replayed        bool                      `json:"replayed"`
}

type ResearchThemeImportCounts struct {
	Themes                int `json:"themes"`
	ChainNodeAssociations int `json:"chain_node_associations"`
	EventAssociations     int `json:"event_associations"`
	Receipts              int `json:"receipts"`
}

type ResearchAnchorImportResult struct {
	ReceiptID                    string                     `json:"receipt_id"`
	ThemeID                      string                     `json:"theme_id"`
	PayloadHash                  string                     `json:"payload_hash"`
	AnchorIDsByCenterChainNodeID map[string]string          `json:"anchor_ids_by_center_chain_node_id"`
	Counts                       ResearchAnchorImportCounts `json:"counts"`
	PublishedAt                  time.Time                  `json:"published_at"`
	ImportedAt                   time.Time                  `json:"imported_at"`
	Replayed                     bool                       `json:"replayed"`
}

type ResearchAnchorImportCounts struct {
	Anchors           int `json:"anchors"`
	EventAssociations int `json:"event_associations"`
	PathNodes         int `json:"path_nodes"`
	Receipts          int `json:"receipts"`
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
	ID                        string                   `json:"id"`
	Name                      string                   `json:"name"`
	OneLineConclusion         string                   `json:"one_line_conclusion"`
	ImpactLevel               string                   `json:"impact_level"`
	TransmissionPath          string                   `json:"transmission_path"`
	TradingDirection          string                   `json:"trading_direction"`
	TransmissionStage         string                   `json:"transmission_stage"`
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

type ResearchThemeChainNode struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	RelationRole  string `json:"relation_role"`
	ImpactSummary string `json:"impact_summary"`
}

type ResearchIndex struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	ImpactDirection string `json:"impact_direction"`
	ImpactSummary   string `json:"impact_summary"`
}

type ResearchEvent struct {
	EventID        string     `json:"event_id"`
	Title          string     `json:"title"`
	Summary        string     `json:"summary"`
	EventTime      *time.Time `json:"event_time,omitempty"`
	EvidenceRole   string     `json:"evidence_role"`
	SupportedClaim string     `json:"supported_claim"`
}

type ResearchReasoningTreeChainNode struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ResearchReasoningTreeSummary struct {
	AnchorID        string                         `json:"anchor_id"`
	CenterChainNode ResearchReasoningTreeChainNode `json:"center_chain_node"`
}

type ResearchReasoningTreeList struct {
	Theme          ResearchTheme                  `json:"theme"`
	ReasoningTrees []ResearchReasoningTreeSummary `json:"reasoning_trees"`
}

type ResearchReasoningTreeEvent struct {
	EventID         string     `json:"event_id"`
	Title           string     `json:"title"`
	Summary         string     `json:"summary"`
	EventTime       *time.Time `json:"event_time,omitempty"`
	EvidenceRole    string     `json:"evidence_role"`
	EvidenceSummary string     `json:"evidence_summary"`
}

type ResearchReasoningTreePathNode struct {
	ChainNodeID                   string  `json:"chain_node_id"`
	Name                          string  `json:"name"`
	ChangeDirection               string  `json:"change_direction"`
	ChangeSummary                 string  `json:"change_summary"`
	ImpactSummary                 string  `json:"impact_summary"`
	IncomingTransmissionMechanism *string `json:"incoming_transmission_mechanism"`
}

type ResearchReasoningTree struct {
	AnchorID            string                          `json:"anchor_id"`
	CenterChainNode     ResearchReasoningTreeChainNode  `json:"center_chain_node"`
	OneLineConclusion   string                          `json:"one_line_conclusion"`
	FactSummary         string                          `json:"fact_summary"`
	NetDirectionSummary string                          `json:"net_direction_summary"`
	SupportSummary      string                          `json:"support_summary"`
	CounterSummary      *string                         `json:"counter_summary"`
	TradingDirection    string                          `json:"trading_direction"`
	NextCheckpoint      string                          `json:"next_checkpoint"`
	EventCount          int                             `json:"event_count"`
	Events              []ResearchReasoningTreeEvent    `json:"events"`
	PathNodes           []ResearchReasoningTreePathNode `json:"path_nodes"`
}

type ResearchReasoningTreeDetail struct {
	ThemeID       string                `json:"theme_id"`
	ReasoningTree ResearchReasoningTree `json:"reasoning_tree"`
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
