package v1

import (
	"encoding/json"
	"time"
)

type EventPublicationRequest struct {
	PackageID    string                        `json:"package_id"`
	Provenance   EventPublicationProvenance    `json:"provenance"`
	RawDocuments []EventPublicationRawDocument `json:"raw_documents"`
	Events       []EventPublicationEvent       `json:"events"`
}

type EventPublicationProvenance struct {
	ExtractorExecutionID  string                               `json:"extractor_execution_id"`
	ExtractorAgentVersion string                               `json:"extractor_agent_version"`
	CollectorExecutions   []EventPublicationCollectorExecution `json:"collector_executions"`
}

type EventPublicationCollectorExecution struct {
	ArtifactID           string `json:"artifact_id"`
	CollectorExecutionID string `json:"collector_execution_id"`
}

type EventPublicationRawDocument struct {
	ArtifactID    string     `json:"artifact_id"`
	ContentSHA256 string     `json:"content_sha256"`
	SourceRef     string     `json:"source_ref"`
	SourceName    string     `json:"source_name"`
	SourceType    string     `json:"source_type"`
	SourceURL     string     `json:"source_url,omitempty"`
	Title         string     `json:"title"`
	PublishedAt   *time.Time `json:"published_at,omitempty"`
	CollectedAt   time.Time  `json:"collected_at"`
	Language      string     `json:"language,omitempty"`
	MIMEType      string     `json:"mime_type,omitempty"`
}

type EventPublicationEvent struct {
	DedupeKey      string                     `json:"dedupe_key"`
	Title          string                     `json:"title"`
	FactualSummary string                     `json:"factual_summary"`
	OccurredAt     *time.Time                 `json:"occurred_at,omitempty"`
	FactPayload    map[string]any             `json:"fact_payload"`
	Evidence       []EventPublicationEvidence `json:"evidence"`
	Tags           []EventPublicationTag      `json:"tags"`
	Review         EventPublicationReview     `json:"review"`
}

type EventPublicationEvidence struct {
	ArtifactID       string   `json:"artifact_id"`
	EvidenceRelation string   `json:"evidence_relation"`
	EvidenceExcerpt  string   `json:"evidence_excerpt"`
	SupportsFields   []string `json:"supports_fields"`
	SourceLevel      string   `json:"source_level"`
	IsPrimary        bool     `json:"is_primary"`
}

type EventPublicationTag struct {
	TagID            string      `json:"tag_id"`
	TagKind          string      `json:"tag_kind"`
	TagCode          string      `json:"tag_code"`
	Confidence       json.Number `json:"confidence"`
	AssignmentReason string      `json:"assignment_reason"`
	AssignSource     string      `json:"assign_source"`
}

type EventPublicationReview struct {
	ReviewID      string   `json:"review_id"`
	EvidenceGrade string   `json:"evidence_grade"`
	Reasons       []string `json:"reasons"`
}

type ResearchThemeImportRequest struct {
	AnalysisBatchID string                    `json:"analysis_batch_id"`
	WindowStart     string                    `json:"window_start"`
	WindowEnd       string                    `json:"window_end"`
	Themes          []ResearchThemeImportItem `json:"themes"`
}

type ResearchThemeImportItem struct {
	ThemeKey                  string                         `json:"theme_key"`
	Name                      string                         `json:"name"`
	OneLineConclusion         string                         `json:"one_line_conclusion"`
	ImpactLevel               string                         `json:"impact_level"`
	TransmissionPath          string                         `json:"transmission_path"`
	TradingDirection          string                         `json:"trading_direction"`
	TransmissionStage         string                         `json:"transmission_stage"`
	NextCheckpoint            string                         `json:"next_checkpoint"`
	MarketConfirmationSummary string                         `json:"market_confirmation_summary"`
	ChainNodes                []ResearchThemeImportChainNode `json:"chain_nodes"`
	Events                    []ResearchThemeImportEvent     `json:"events"`
}

type ResearchThemeImportChainNode struct {
	ChainNodeID   string `json:"chain_node_id"`
	RelationRole  string `json:"relation_role"`
	ImpactSummary string `json:"impact_summary"`
}

type ResearchThemeImportEvent struct {
	EventID        string `json:"event_id"`
	EvidenceRole   string `json:"evidence_role"`
	SupportedClaim string `json:"supported_claim"`
}

type ResearchAnchorImportRequest struct {
	ThemeID string                     `json:"theme_id"`
	Anchors []ResearchAnchorImportItem `json:"anchors"`
}

type ResearchAnchorImportItem struct {
	CenterChainNodeID   string                         `json:"center_chain_node_id"`
	OneLineConclusion   string                         `json:"one_line_conclusion"`
	FactSummary         string                         `json:"fact_summary"`
	NetDirectionSummary string                         `json:"net_direction_summary"`
	SupportSummary      string                         `json:"support_summary"`
	CounterSummary      *string                        `json:"counter_summary"`
	TradingDirection    string                         `json:"trading_direction"`
	NextCheckpoint      string                         `json:"next_checkpoint"`
	Events              []ResearchAnchorImportEvent    `json:"events"`
	PathNodes           []ResearchAnchorImportPathNode `json:"path_nodes"`
}

type ResearchAnchorImportEvent struct {
	EventID         string `json:"event_id"`
	EvidenceRole    string `json:"evidence_role"`
	EvidenceSummary string `json:"evidence_summary"`
}

type ResearchAnchorImportPathNode struct {
	ChainNodeID                   string  `json:"chain_node_id"`
	ChangeDirection               string  `json:"change_direction"`
	ChangeSummary                 string  `json:"change_summary"`
	ImpactSummary                 string  `json:"impact_summary"`
	IncomingTransmissionMechanism *string `json:"incoming_transmission_mechanism"`
}
