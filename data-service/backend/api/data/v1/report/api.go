package report

import (
	"context"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

const (
	ContractVersion = "report-publication.v1"

	OperationPublishReport      = "data.v1.publishReport"
	OperationListReports        = "data.v1.listReports"
	OperationGetReportHome      = "data.v1.getReportHome"
	OperationGetReportLayer     = "data.v1.getReportLayer"
	OperationGetReportChain     = "data.v1.getReportIndustryChain"
	OperationListReportEvidence = "data.v1.listReportEvidence"

	ErrorInvalidRequest                 = "INVALID_REQUEST"
	ErrorReportNotFound                 = "REPORT_NOT_FOUND"
	ErrorReportLayerNotFound            = "REPORT_LAYER_NOT_FOUND"
	ErrorReportIndustryChainNotFound    = "REPORT_INDUSTRY_CHAIN_NOT_FOUND"
	ErrorReportEvidenceScopeNotFound    = "REPORT_EVIDENCE_SCOPE_NOT_FOUND"
	ErrorReportPublicationConflict      = "REPORT_PUBLICATION_CONFLICT"
	ErrorReportEvidenceReferenceInvalid = "REPORT_EVIDENCE_REFERENCE_INVALID"
	ErrorReportRepositoryFailure        = "REPORT_REPOSITORY_FAILURE"
	ErrorDataServiceNotReady            = "DATA_SERVICE_NOT_READY"
)

func BusinessOperations() []string {
	return []string{
		OperationPublishReport,
		OperationListReports,
		OperationGetReportHome,
		OperationGetReportLayer,
		OperationGetReportChain,
		OperationListReportEvidence,
	}
}

type Service interface {
	PublishReport(context.Context, *PublicationRequest) (*v1.Response[PublicationResult], error)
	ListReports(context.Context, *ListRequest) (*v1.Response[Collection], error)
	GetReportHome(context.Context, *ReportRequest) (*v1.Response[Home], error)
	GetReportLayer(context.Context, *LayerRequest) (*v1.Response[LayerDetail], error)
	GetReportIndustryChain(context.Context, *ChainRequest) (*v1.Response[IndustryChainDetail], error)
	ListReportEvidence(context.Context, *EvidenceRequest) (*v1.Response[EvidenceCollection], error)
}

type PublicationRequest struct {
	ContractVersion string  `json:"contract_version"`
	SourceReportID  string  `json:"source_report_id"`
	Content         Content `json:"content"`
}

type Result struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

type Nature struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

type Confidence struct {
	Label string   `json:"label"`
	Score *float64 `json:"score"`
}

type EvidenceReference struct {
	EvidenceID   string `json:"evidence_id"`
	Role         string `json:"role"`
	DisplayOrder int    `json:"display_order"`
}

type TargetReference struct {
	Type string `json:"type"`
	Key  string `json:"key"`
}

type ImpactItem struct {
	Ref        TargetReference `json:"ref"`
	Name       string          `json:"name"`
	Result     Result          `json:"result"`
	Confidence Confidence      `json:"confidence"`
	TimeWindow string          `json:"time_window"`
}

type ReportCard struct {
	Key          string              `json:"key"`
	Kind         string              `json:"kind"`
	DisplayOrder int                 `json:"display_order"`
	DetailRef    TargetReference     `json:"detail_ref"`
	Title        string              `json:"title"`
	Subtitle     string              `json:"subtitle"`
	Conclusion   string              `json:"conclusion"`
	Result       Result              `json:"result"`
	Confidence   Confidence          `json:"confidence"`
	TimeWindow   string              `json:"time_window"`
	ImpactItems  []ImpactItem        `json:"impact_items"`
	EvidenceRefs []EvidenceReference `json:"evidence_refs"`
}

type Statistics struct {
	EventCount                           int     `json:"event_count"`
	OrdinaryFactCount                    int     `json:"ordinary_fact_count"`
	SignalFactCount                      int     `json:"signal_fact_count"`
	TransmissionHypothesisCount          int     `json:"transmission_hypothesis_count"`
	RemainingTopologyPendingCount        int     `json:"remaining_topology_pending_count"`
	AdaptiveInclusionThreshold           float64 `json:"adaptive_inclusion_threshold"`
	AdaptiveContinuationThreshold        float64 `json:"adaptive_continuation_threshold"`
	AdaptiveHardMaxHops                  int     `json:"adaptive_hard_max_hops"`
	AdaptiveObservedMaxHops              int     `json:"adaptive_observed_max_hops"`
	AdaptiveStoppedByConfidence          int     `json:"adaptive_stopped_by_confidence"`
	AdaptiveStoppedByNoUnvisitedNeighbor int     `json:"adaptive_stopped_by_no_unvisited_neighbor"`
	AdaptiveRejectedBelowInclusion       int     `json:"adaptive_rejected_below_inclusion"`
	GeopoliticAnchorCount                int     `json:"geopolitic_anchor_count"`
	MacroeconomicAnchorCount             int     `json:"macroeconomic_anchor_count"`
	SignaledChainNodeCount               int     `json:"signaled_chain_node_count"`
	IndustryChainCount                   int     `json:"industry_chain_count"`
	UnmappedChainNodeCount               int     `json:"unmapped_chain_node_count"`
}

type Anchor struct {
	Key          string              `json:"key"`
	DisplayOrder int                 `json:"display_order"`
	Name         string              `json:"name"`
	CurrentState string              `json:"current_state"`
	Result       Result              `json:"result"`
	Nature       Nature              `json:"nature"`
	Reasoning    string              `json:"reasoning"`
	TimeWindow   string              `json:"time_window"`
	Confidence   Confidence          `json:"confidence"`
	EvidenceRefs []EvidenceReference `json:"evidence_refs"`
}

type ReasoningStep struct {
	Key          string              `json:"key"`
	DisplayOrder int                 `json:"display_order"`
	Input        string              `json:"input"`
	Mechanism    string              `json:"mechanism"`
	Output       string              `json:"output"`
	Type         string              `json:"type"`
	Confidence   Confidence          `json:"confidence"`
	EvidenceRefs []EvidenceReference `json:"evidence_refs"`
}

type TransmissionTarget struct {
	Ref    TargetReference `json:"ref"`
	Label  string          `json:"label"`
	Result Result          `json:"result"`
}

type TransmissionPath struct {
	Key              string               `json:"key"`
	DisplayOrder     int                  `json:"display_order"`
	SourceConclusion string               `json:"source_conclusion"`
	TargetRefs       []TransmissionTarget `json:"target_refs"`
	Logic            string               `json:"logic"`
	RelationNature   string               `json:"relation_nature"`
	EvidenceRole     string               `json:"evidence_role"`
	Confidence       Confidence           `json:"confidence"`
	Status           string               `json:"status"`
	EvidenceRefs     []EvidenceReference  `json:"evidence_refs"`
}

type CandidateMechanism struct {
	Key          string              `json:"key"`
	DisplayOrder int                 `json:"display_order"`
	Mechanism    string              `json:"mechanism"`
	EvidenceGap  *string             `json:"evidence_gap"`
	Confidence   Confidence          `json:"confidence"`
	EvidenceRefs []EvidenceReference `json:"evidence_refs"`
}

type Checkpoint struct {
	Key          string `json:"key"`
	DisplayOrder int    `json:"display_order"`
	Summary      string `json:"summary"`
}

type DownwardTransmission struct {
	Summary             string               `json:"summary"`
	PublishedPaths      []TransmissionPath   `json:"published_paths"`
	CandidateMechanisms []CandidateMechanism `json:"candidate_mechanisms"`
	BoundaryNotes       []string             `json:"boundary_notes"`
}

type LayerUncertainty struct {
	Counterevidence   *string      `json:"counterevidence"`
	EvidenceGap       *string      `json:"evidence_gap"`
	Boundary          *string      `json:"boundary"`
	ReversalCondition *string      `json:"reversal_condition"`
	Checkpoints       []Checkpoint `json:"checkpoints"`
}

type Layer struct {
	Key                  string               `json:"key"`
	DisplayOrder         int                  `json:"display_order"`
	Title                string               `json:"title"`
	Conclusion           string               `json:"conclusion"`
	Result               Result               `json:"result"`
	Confidence           Confidence           `json:"confidence"`
	TimeWindow           string               `json:"time_window"`
	Anchors              []Anchor             `json:"anchors"`
	ReasoningSteps       []ReasoningStep      `json:"reasoning_steps"`
	RelatedAnchorKeys    []string             `json:"related_anchor_keys"`
	RelatedChainKeys     []string             `json:"related_chain_keys"`
	DownwardTransmission DownwardTransmission `json:"downward_transmission"`
	Uncertainty          LayerUncertainty     `json:"uncertainty"`
	EvidenceRefs         []EvidenceReference  `json:"evidence_refs"`
}

type IndustryChainNode struct {
	Key          string              `json:"key"`
	DisplayOrder int                 `json:"display_order"`
	Name         string              `json:"name"`
	Impact       string              `json:"impact"`
	Result       Result              `json:"result"`
	Nature       Nature              `json:"nature"`
	Reasoning    string              `json:"reasoning"`
	TimeWindow   string              `json:"time_window"`
	Confidence   Confidence          `json:"confidence"`
	EvidenceRefs []EvidenceReference `json:"evidence_refs"`
}

type IndustryChainEdge struct {
	Key           string `json:"key"`
	DisplayOrder  int    `json:"display_order"`
	FromNodeKey   string `json:"from_node_key"`
	ToNodeKey     string `json:"to_node_key"`
	RelationLabel string `json:"relation_label"`
}

type ChainUncertainty struct {
	CounterevidenceAndGap *string      `json:"counterevidence_and_gap"`
	StopCondition         *string      `json:"stop_condition"`
	Checkpoints           []Checkpoint `json:"checkpoints"`
}

type IndustryChain struct {
	Key                       string              `json:"key"`
	ClaimKey                  string              `json:"claim_key"`
	DisplayOrder              int                 `json:"display_order"`
	Name                      string              `json:"name"`
	Conclusion                string              `json:"conclusion"`
	Status                    string              `json:"status"`
	Result                    Result              `json:"result"`
	Confidence                Confidence          `json:"confidence"`
	TimeWindow                string              `json:"time_window"`
	PathSummary               *string             `json:"path_summary"`
	AcceptedHypothesisSummary *string             `json:"accepted_hypothesis_summary"`
	EvidenceRefs              []EvidenceReference `json:"evidence_refs"`
	Nodes                     []IndustryChainNode `json:"nodes"`
	Edges                     []IndustryChainEdge `json:"edges"`
	Uncertainty               ChainUncertainty    `json:"uncertainty"`
}

type CompanyBoundary struct {
	Key          string `json:"key"`
	DisplayOrder int    `json:"display_order"`
	Title        string `json:"title"`
	Published    bool   `json:"published"`
	Boundary     string `json:"boundary"`
}

type Content struct {
	ReportType      string          `json:"report_type"`
	Title           string          `json:"title"`
	Status          string          `json:"status"`
	Simulation      bool            `json:"simulation"`
	GeneratedAt     string          `json:"generated_at"`
	Timezone        string          `json:"timezone"`
	PublishedLayers []string        `json:"published_layers"`
	Statistics      Statistics      `json:"statistics"`
	ReportCards     []ReportCard    `json:"report_cards"`
	Geopolitics     Layer           `json:"geopolitics"`
	Macroeconomics  Layer           `json:"macroeconomics"`
	IndustryChains  []IndustryChain `json:"industry_chains"`
	Company         CompanyBoundary `json:"company"`
}

type PublicationResult struct {
	ReportID    string `json:"report_id"`
	ContentHash string `json:"content_hash"`
	PublishedAt string `json:"published_at"`
	Replayed    bool   `json:"replayed"`
}

type ListRequest struct {
	PublishedFrom string
	PublishedTo   string
	Limit         string
	Cursor        string
}

type ReportRequest struct{ ReportID string }

type LayerRequest struct {
	ReportID string
	LayerKey string
}

type ChainRequest struct {
	ReportID string
	ChainKey string
}

type EvidenceRequest struct {
	ReportID        string
	ScopeType       string
	ScopeKey        string
	HasUnknownQuery bool
}

type Summary struct {
	ID              string     `json:"id"`
	SourceReportID  string     `json:"source_report_id"`
	ReportType      string     `json:"report_type"`
	Title           string     `json:"title"`
	Status          string     `json:"status"`
	Simulation      bool       `json:"simulation"`
	GeneratedAt     string     `json:"generated_at"`
	Timezone        string     `json:"timezone"`
	PublishedLayers []string   `json:"published_layers"`
	Statistics      Statistics `json:"statistics"`
	PublishedAt     string     `json:"published_at"`
}

type Collection struct {
	Items      []Summary `json:"items"`
	NextCursor *string   `json:"next_cursor"`
}

type ImpactItemRead struct {
	Ref           TargetReference `json:"ref"`
	Name          string          `json:"name"`
	Result        Result          `json:"result"`
	Confidence    Confidence      `json:"confidence"`
	TimeWindow    string          `json:"time_window"`
	EvidenceCount int             `json:"evidence_count"`
}

type ReportCardRead struct {
	Key           string           `json:"key"`
	Kind          string           `json:"kind"`
	DisplayOrder  int              `json:"display_order"`
	DetailRef     TargetReference  `json:"detail_ref"`
	Title         string           `json:"title"`
	Subtitle      string           `json:"subtitle"`
	Conclusion    string           `json:"conclusion"`
	Result        Result           `json:"result"`
	Confidence    Confidence       `json:"confidence"`
	TimeWindow    string           `json:"time_window"`
	ImpactItems   []ImpactItemRead `json:"impact_items"`
	EvidenceCount int              `json:"evidence_count"`
}

type AnchorRead struct {
	Key           string     `json:"key"`
	DisplayOrder  int        `json:"display_order"`
	Name          string     `json:"name"`
	CurrentState  string     `json:"current_state"`
	Result        Result     `json:"result"`
	Nature        Nature     `json:"nature"`
	Reasoning     string     `json:"reasoning"`
	TimeWindow    string     `json:"time_window"`
	Confidence    Confidence `json:"confidence"`
	EvidenceCount int        `json:"evidence_count"`
}

type ReasoningStepRead struct {
	Key           string     `json:"key"`
	DisplayOrder  int        `json:"display_order"`
	Input         string     `json:"input"`
	Mechanism     string     `json:"mechanism"`
	Output        string     `json:"output"`
	Type          string     `json:"type"`
	Confidence    Confidence `json:"confidence"`
	EvidenceCount int        `json:"evidence_count"`
}

type TransmissionPathRead struct {
	Key              string               `json:"key"`
	DisplayOrder     int                  `json:"display_order"`
	SourceConclusion string               `json:"source_conclusion"`
	TargetRefs       []TransmissionTarget `json:"target_refs"`
	Logic            string               `json:"logic"`
	RelationNature   string               `json:"relation_nature"`
	EvidenceRole     string               `json:"evidence_role"`
	Confidence       Confidence           `json:"confidence"`
	Status           string               `json:"status"`
	EvidenceCount    int                  `json:"evidence_count"`
}

type CandidateMechanismRead struct {
	Key           string     `json:"key"`
	DisplayOrder  int        `json:"display_order"`
	Mechanism     string     `json:"mechanism"`
	EvidenceGap   *string    `json:"evidence_gap"`
	Confidence    Confidence `json:"confidence"`
	EvidenceCount int        `json:"evidence_count"`
}

type DownwardTransmissionRead struct {
	Summary             string                   `json:"summary"`
	PublishedPaths      []TransmissionPathRead   `json:"published_paths"`
	CandidateMechanisms []CandidateMechanismRead `json:"candidate_mechanisms"`
	BoundaryNotes       []string                 `json:"boundary_notes"`
}

type LayerRead struct {
	Key                  string                   `json:"key"`
	DisplayOrder         int                      `json:"display_order"`
	Title                string                   `json:"title"`
	Conclusion           string                   `json:"conclusion"`
	Result               Result                   `json:"result"`
	Confidence           Confidence               `json:"confidence"`
	TimeWindow           string                   `json:"time_window"`
	Anchors              []AnchorRead             `json:"anchors"`
	ReasoningSteps       []ReasoningStepRead      `json:"reasoning_steps"`
	RelatedAnchorKeys    []string                 `json:"related_anchor_keys"`
	RelatedChainKeys     []string                 `json:"related_chain_keys"`
	DownwardTransmission DownwardTransmissionRead `json:"downward_transmission"`
	Uncertainty          LayerUncertainty         `json:"uncertainty"`
	EvidenceCount        int                      `json:"evidence_count"`
}

type IndustryChainSummary struct {
	Key           string     `json:"key"`
	DisplayOrder  int        `json:"display_order"`
	Name          string     `json:"name"`
	Conclusion    string     `json:"conclusion"`
	Status        string     `json:"status"`
	Result        Result     `json:"result"`
	Confidence    Confidence `json:"confidence"`
	TimeWindow    string     `json:"time_window"`
	EvidenceCount int        `json:"evidence_count"`
}

type IndustryChainNodeRead struct {
	Key           string     `json:"key"`
	DisplayOrder  int        `json:"display_order"`
	Name          string     `json:"name"`
	Impact        string     `json:"impact"`
	Result        Result     `json:"result"`
	Nature        Nature     `json:"nature"`
	Reasoning     string     `json:"reasoning"`
	TimeWindow    string     `json:"time_window"`
	Confidence    Confidence `json:"confidence"`
	EvidenceCount int        `json:"evidence_count"`
}

type IndustryChainRead struct {
	Key                       string                  `json:"key"`
	ClaimKey                  string                  `json:"claim_key"`
	DisplayOrder              int                     `json:"display_order"`
	Name                      string                  `json:"name"`
	Conclusion                string                  `json:"conclusion"`
	Status                    string                  `json:"status"`
	Result                    Result                  `json:"result"`
	Confidence                Confidence              `json:"confidence"`
	TimeWindow                string                  `json:"time_window"`
	PathSummary               *string                 `json:"path_summary"`
	AcceptedHypothesisSummary *string                 `json:"accepted_hypothesis_summary"`
	Nodes                     []IndustryChainNodeRead `json:"nodes"`
	Edges                     []IndustryChainEdge     `json:"edges"`
	Uncertainty               ChainUncertainty        `json:"uncertainty"`
	EvidenceCount             int                     `json:"evidence_count"`
}

type Home struct {
	Report      Summary          `json:"report"`
	ReportCards []ReportCardRead `json:"report_cards"`
	Company     CompanyBoundary  `json:"company"`
}

type LayerDetail struct {
	Report                Summary                `json:"report"`
	Layer                 LayerRead              `json:"layer"`
	RelatedIndustryChains []IndustryChainSummary `json:"related_industry_chains"`
}

type IndustryChainDetail struct {
	Report        Summary           `json:"report"`
	IndustryChain IndustryChainRead `json:"industry_chain"`
}

type EvidenceItem struct {
	EvidenceID   string   `json:"evidence_id"`
	Role         string   `json:"role"`
	DisplayOrder int      `json:"display_order"`
	PublishedAt  *string  `json:"published_at"`
	Summary      string   `json:"summary"`
	Keywords     []string `json:"keywords"`
}

type EvidenceCollection struct {
	ReportID  string         `json:"report_id"`
	ScopeType string         `json:"scope_type"`
	ScopeKey  string         `json:"scope_key"`
	Items     []EvidenceItem `json:"items"`
}
