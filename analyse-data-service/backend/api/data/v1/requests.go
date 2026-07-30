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
	AnalysisBatchID      string                            `json:"analysis_batch_id"`
	AnalysisAsOf         string                            `json:"analysis_as_of"`
	DiscoveryWindowStart string                            `json:"discovery_window_start"`
	DiscoveryWindowEnd   string                            `json:"discovery_window_end"`
	Theme                ResearchThemeImportItem           `json:"theme"`
	ReasoningTrees       []ResearchReasoningTreeImportItem `json:"reasoning_trees"`
}

type ResearchThemeImportItem struct {
	ThemeKey                  string                      `json:"theme_key"`
	Title                     string                      `json:"title"`
	OneLineConclusion         string                      `json:"one_line_conclusion"`
	ConclusionDirection       string                      `json:"conclusion_direction"`
	ImpactStrength            string                      `json:"impact_strength"`
	AttentionLevel            *string                     `json:"attention_level"`
	ConclusionStatus          *string                     `json:"conclusion_status"`
	TransmissionStage         string                      `json:"transmission_stage"`
	InvestmentGuidanceAction  string                      `json:"investment_guidance_action"`
	InvestmentGuidanceSummary string                      `json:"investment_guidance_summary"`
	TimeHorizonCategory       string                      `json:"time_horizon_category"`
	TimeHorizonSummary        *string                     `json:"time_horizon_summary"`
	TransmissionSummary       *string                     `json:"transmission_summary"`
	CheckpointSummary         *string                     `json:"checkpoint_summary"`
	RiskSummary               *string                     `json:"risk_summary"`
	Impacts                   []ResearchThemeImportImpact `json:"impacts"`
	Events                    []ResearchThemeImportEvent  `json:"events"`
}

type ResearchThemeImportImpact struct {
	ChainNodeEntityID string  `json:"chain_node_entity_id"`
	RelationRole      string  `json:"relation_role"`
	ImpactDirection   string  `json:"impact_direction"`
	ImpactSummary     *string `json:"impact_summary"`
	DisplayOrder      int     `json:"display_order"`
}

type ResearchThemeImportEvent struct {
	EventID        string  `json:"event_id"`
	EvidenceRole   string  `json:"evidence_role"`
	SupportedClaim *string `json:"supported_claim"`
}

type ResearchReasoningTreeImportItem struct {
	IndustryChainEntityID     string                                  `json:"industry_chain_entity_id"`
	Title                     string                                  `json:"title"`
	DisplayOrder              int                                     `json:"display_order"`
	OneLineConclusion         string                                  `json:"one_line_conclusion"`
	FactSummary               *string                                 `json:"fact_summary"`
	TransmissionSummary       *string                                 `json:"transmission_summary"`
	ImpactDirection           string                                  `json:"impact_direction"`
	ImpactStrength            string                                  `json:"impact_strength"`
	ImpactSummary             *string                                 `json:"impact_summary"`
	ConclusionBoundarySummary *string                                 `json:"conclusion_boundary_summary"`
	SupportSummary            *string                                 `json:"support_summary"`
	CounterSummary            *string                                 `json:"counter_summary"`
	InvalidationConditions    []string                                `json:"invalidation_conditions"`
	Checkpoints               []ResearchReasoningTreeImportCheckpoint `json:"checkpoints"`
	Events                    []ResearchReasoningTreeImportEvent      `json:"events"`
	Nodes                     []ResearchReasoningTreeImportNode       `json:"nodes"`
}

type ResearchReasoningTreeImportCheckpoint struct {
	Type    string `json:"type"`
	Summary string `json:"summary"`
}

type ResearchReasoningTreeImportEvent struct {
	EventID      string `json:"event_id"`
	EvidenceRole string `json:"evidence_role"`
	DisplayOrder int    `json:"display_order"`
}

type ResearchReasoningTreeImportNode struct {
	Position                         int                                   `json:"position"`
	ChainNodeEntityID                string                                `json:"chain_node_entity_id"`
	StateSummary                     *string                               `json:"state_summary"`
	ImpactDirection                  string                                `json:"impact_direction"`
	ImpactStrength                   string                                `json:"impact_strength"`
	ImpactSummary                    *string                               `json:"impact_summary"`
	ReasoningBasisSummary            *string                               `json:"reasoning_basis_summary"`
	EvidenceGapSummary               *string                               `json:"evidence_gap_summary"`
	IncomingIndustryChainGraphEdgeID *string                               `json:"incoming_industry_chain_graph_edge_id"`
	IncomingTransmissionTitle        *string                               `json:"incoming_transmission_title"`
	IncomingTransmissionMechanism    *string                               `json:"incoming_transmission_mechanism"`
	IncomingConditionSummary         *string                               `json:"incoming_condition_summary"`
	IncomingLineage                  *ResearchReasoningTreeIncomingLineage `json:"incoming_lineage"`
	Signals                          []ResearchReasoningTreeImportSignal   `json:"signals"`
}

type ResearchReasoningTreeImportSignal struct {
	VariableSignalKey string                             `json:"variable_signal_key"`
	SignalRole        string                             `json:"signal_role"`
	SignalDirection   string                             `json:"signal_direction"`
	DisplaySummary    string                             `json:"display_summary"`
	DisplayOrder      int                                `json:"display_order"`
	Lineage           ResearchReasoningTreeSignalLineage `json:"lineage"`
}

type ResearchReasoningTreeSignalLineage struct {
	SourceKind                      string  `json:"source_kind"`
	VariableSignalID                *string `json:"variable_signal_id"`
	SemanticSubmissionID            *string `json:"semantic_submission_id"`
	EvidenceID                      *string `json:"evidence_id"`
	EvidenceHash                    *string `json:"evidence_hash"`
	UpstreamVariableSignalID        *string `json:"upstream_variable_signal_id"`
	UpstreamDirectImpactAssertionID *string `json:"upstream_direct_impact_assertion_id"`
	EntityRelationID                *string `json:"entity_relation_id"`
	IndustryChainGraphEdgeID        *string `json:"industry_chain_graph_edge_id"`
}

type ResearchReasoningTreeIncomingLineage struct {
	SourceKind                      string  `json:"source_kind"`
	DirectImpactAssertionID         *string `json:"direct_impact_assertion_id"`
	SemanticSubmissionID            *string `json:"semantic_submission_id"`
	EvidenceID                      *string `json:"evidence_id"`
	EvidenceHash                    *string `json:"evidence_hash"`
	AffectedVariableKey             *string `json:"affected_variable_key"`
	AffectedDirection               *string `json:"affected_direction"`
	UpstreamVariableSignalID        *string `json:"upstream_variable_signal_id"`
	UpstreamDirectImpactAssertionID *string `json:"upstream_direct_impact_assertion_id"`
	EntityRelationID                *string `json:"entity_relation_id"`
}
