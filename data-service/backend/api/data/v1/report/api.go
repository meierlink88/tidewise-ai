package report

import (
	"context"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

const (
	ContractVersion                     = "report-publication.v2"
	OperationPublishReport              = "data.v1.publishReport"
	OperationListReports                = "data.v1.listReports"
	OperationGetReportHome              = "data.v1.getReportHome"
	OperationGetReportLayer             = "data.v1.getReportLayer"
	OperationListReportChains           = "data.v1.listReportIndustryChains"
	OperationGetReportChain             = "data.v1.getReportIndustryChain"
	OperationListReportEvidence         = "data.v1.listReportEvidence"
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
	return []string{OperationPublishReport, OperationListReports, OperationGetReportHome, OperationGetReportLayer, OperationListReportChains, OperationGetReportChain, OperationListReportEvidence}
}

type Service interface {
	PublishReport(context.Context, *PublicationRequest) (*v1.Response[PublicationResult], error)
	ListReports(context.Context, *ListRequest) (*v1.Response[Collection], error)
	GetReportHome(context.Context, *ReportRequest) (*v1.Response[Home], error)
	GetReportLayer(context.Context, *LayerRequest) (*v1.Response[LayerDetail], error)
	ListReportIndustryChains(context.Context, *ChainListRequest) (*v1.Response[IndustryChainCollection], error)
	GetReportIndustryChain(context.Context, *ChainRequest) (*v1.Response[IndustryChainDetail], error)
	ListReportEvidence(context.Context, *EvidenceRequest) (*v1.Response[EvidenceCollection], error)
}

type PublicationRequest struct {
	ContractVersion   string  `json:"contract_version"`
	PublisherReportID string  `json:"publisher_report_id"`
	Content           Content `json:"content"`
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
	Code  string   `json:"code"`
	Label string   `json:"label"`
	Score *float64 `json:"score"`
}
type TimeWindow struct {
	Horizons []string `json:"horizons"`
	Lag      *string  `json:"lag"`
	Label    string   `json:"label"`
}
type Effect struct {
	DisplayOrder int    `json:"display_order"`
	Dimension    string `json:"dimension"`
	Direction    string `json:"direction"`
	Confidence   string `json:"confidence"`
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
type Claim struct {
	Key  string `json:"key"`
	Text string `json:"text"`
}
type Statistics struct {
	EventCount                  int `json:"event_count"`
	OrdinaryFactCount           int `json:"ordinary_fact_count"`
	SignalFactCount             int `json:"signal_fact_count"`
	TransmissionHypothesisCount int `json:"transmission_hypothesis_count"`
	GeopoliticAnchorCount       int `json:"geopolitic_anchor_count"`
	MacroeconomicAnchorCount    int `json:"macroeconomic_anchor_count"`
	SignaledChainNodeCount      int `json:"signaled_chain_node_count"`
	IndustryChainCount          int `json:"industry_chain_count"`
}
type Anchor struct {
	Key          string              `json:"key"`
	DisplayOrder int                 `json:"display_order"`
	Name         string              `json:"name"`
	Effects      []Effect            `json:"effects"`
	Result       Result              `json:"result"`
	Nature       Nature              `json:"nature"`
	Reasoning    string              `json:"reasoning"`
	TimeWindow   TimeWindow          `json:"time_window"`
	Confidence   Confidence          `json:"confidence"`
	SourceRef    *string             `json:"source_ref"`
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
type NamedResult struct {
	Name   string `json:"name"`
	Result Result `json:"result"`
}
type TransmissionTarget struct {
	Ref     *TargetReference `json:"ref,omitempty"`
	Label   string           `json:"label"`
	Results []NamedResult    `json:"results"`
}
type Transmission struct {
	Key              string               `json:"key"`
	DisplayOrder     int                  `json:"display_order"`
	SourceClaimKey   string               `json:"source_claim_key"`
	SourceConclusion string               `json:"source_conclusion"`
	Targets          []TransmissionTarget `json:"targets"`
	Logic            string               `json:"logic"`
	RelationNature   string               `json:"relation_nature"`
	Confidence       Confidence           `json:"confidence"`
	Status           string               `json:"status"`
	EvidenceRefs     []EvidenceReference  `json:"evidence_refs"`
}
type Checkpoint struct {
	Key          string `json:"key"`
	DisplayOrder int    `json:"display_order"`
	Summary      string `json:"summary"`
}
type LayerUncertainty struct {
	Counterevidence   *string      `json:"counterevidence"`
	EvidenceGap       *string      `json:"evidence_gap"`
	Boundary          *string      `json:"boundary"`
	ReversalCondition *string      `json:"reversal_condition"`
	Checkpoints       []Checkpoint `json:"checkpoints"`
}
type LayerSummary struct {
	Claim         Claim               `json:"claim"`
	Transmissions []Transmission      `json:"transmissions"`
	Uncertainty   LayerUncertainty    `json:"uncertainty"`
	EvidenceRefs  []EvidenceReference `json:"evidence_refs"`
}
type LayerAnalysis struct {
	Anchors          []Anchor        `json:"anchors"`
	ReasoningSteps   []ReasoningStep `json:"reasoning_steps"`
	RelatedChainKeys []string        `json:"related_chain_keys"`
}
type Layer struct {
	Key     string        `json:"key"`
	Title   string        `json:"title"`
	Summary LayerSummary  `json:"summary"`
	Detail  LayerAnalysis `json:"detail"`
}
type IndustryChainTopologyNode struct {
	Key          string `json:"key"`
	DisplayOrder int    `json:"display_order"`
	Name         string `json:"name"`
}
type IndustryChainNode struct {
	Key          string              `json:"key"`
	DisplayOrder int                 `json:"display_order"`
	NodeKey      string              `json:"node_key"`
	Effects      []Effect            `json:"effects"`
	Result       Result              `json:"result"`
	Nature       Nature              `json:"nature"`
	Reasoning    string              `json:"reasoning"`
	TimeWindow   TimeWindow          `json:"time_window"`
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
	CounterevidenceAndGap string `json:"counterevidence_and_gap"`
	StopCondition         string `json:"stop_condition"`
}
type IndustryChainGraph struct {
	Nodes []IndustryChainTopologyNode `json:"nodes"`
	Edges []IndustryChainEdge         `json:"edges"`
}
type ChainSummary struct {
	Claim                     Claim               `json:"claim"`
	Status                    string              `json:"status"`
	Result                    Result              `json:"result"`
	Confidence                Confidence          `json:"confidence"`
	TimeWindow                TimeWindow          `json:"time_window"`
	Path                      string              `json:"path"`
	AcceptedHypothesisSummary *string             `json:"accepted_hypothesis_summary"`
	Graph                     IndustryChainGraph  `json:"graph"`
	Uncertainty               ChainUncertainty    `json:"uncertainty"`
	EvidenceRefs              []EvidenceReference `json:"evidence_refs"`
}
type IndustryChainAnalysis struct {
	NodeImpacts []IndustryChainNode `json:"node_impacts"`
}
type IndustryChain struct {
	Key          string                `json:"key"`
	DisplayOrder int                   `json:"display_order"`
	Name         string                `json:"name"`
	Summary      ChainSummary          `json:"summary"`
	Detail       IndustryChainAnalysis `json:"detail"`
}
type AnalysisWindow struct {
	StartedAt string `json:"started_at"`
	EndedAt   string `json:"ended_at"`
}
type TemplateReference struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Role    string `json:"role"`
}
type Provenance struct {
	DerivedFromReportID *string           `json:"derived_from_report_id"`
	FrozenSourceSHA256  *string           `json:"frozen_source_sha256"`
	FrozenSourceCommit  *string           `json:"frozen_source_commit"`
	Template            TemplateReference `json:"template"`
}
type Content struct {
	ReportType       string          `json:"report_type"`
	Title            string          `json:"title"`
	GenerationStatus string          `json:"generation_status"`
	Simulation       bool            `json:"simulation"`
	GeneratedAt      string          `json:"generated_at"`
	AnalysisWindow   AnalysisWindow  `json:"analysis_window"`
	Timezone         string          `json:"timezone"`
	Provenance       Provenance      `json:"provenance"`
	Statistics       Statistics      `json:"statistics"`
	Geopolitics      *Layer          `json:"geopolitics,omitempty"`
	Macroeconomics   *Layer          `json:"macroeconomics,omitempty"`
	IndustryChains   []IndustryChain `json:"industry_chains"`
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
type ChainListRequest struct {
	ReportID        string
	Limit           string
	Cursor          string
	HasUnknownQuery bool
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
	ID                string     `json:"id"`
	PublisherReportID string     `json:"publisher_report_id"`
	ReportType        string     `json:"report_type"`
	Title             string     `json:"title"`
	GenerationStatus  string     `json:"generation_status"`
	Simulation        bool       `json:"simulation"`
	GeneratedAt       string     `json:"generated_at"`
	Timezone          string     `json:"timezone"`
	HasGeopolitics    bool       `json:"has_geopolitics"`
	HasMacroeconomics bool       `json:"has_macroeconomics"`
	Statistics        Statistics `json:"statistics"`
	PublishedAt       string     `json:"published_at"`
}
type Collection struct {
	Items      []Summary `json:"items"`
	NextCursor *string   `json:"next_cursor"`
}
type LayerSnapshot struct {
	Key     string       `json:"key"`
	Title   string       `json:"title"`
	Summary LayerSummary `json:"summary"`
}
type Home struct {
	Report         Summary        `json:"report"`
	Geopolitics    *LayerSnapshot `json:"geopolitics"`
	Macroeconomics *LayerSnapshot `json:"macroeconomics"`
}
type IndustryChainSummary struct {
	Key           string                       `json:"key"`
	DisplayOrder  int                          `json:"display_order"`
	Name          string                       `json:"name"`
	Claim         Claim                        `json:"claim"`
	Status        string                       `json:"status"`
	Result        Result                       `json:"result"`
	Confidence    Confidence                   `json:"confidence"`
	TimeWindow    TimeWindow                   `json:"time_window"`
	ImpactItems   []IndustryChainImpactSummary `json:"impact_items"`
	EvidenceCount int                          `json:"evidence_count"`
}
type IndustryChainImpactSummary struct {
	Key           string     `json:"key"`
	DisplayOrder  int        `json:"display_order"`
	NodeKey       string     `json:"node_key"`
	Name          string     `json:"name"`
	Result        Result     `json:"result"`
	Nature        Nature     `json:"nature"`
	Confidence    Confidence `json:"confidence"`
	TimeWindow    TimeWindow `json:"time_window"`
	EvidenceCount int        `json:"evidence_count"`
}
type IndustryChainCollection struct {
	Items      []IndustryChainSummary `json:"items"`
	NextCursor *string                `json:"next_cursor"`
}
type LayerDetail struct {
	Report                Summary                `json:"report"`
	Layer                 Layer                  `json:"layer"`
	RelatedIndustryChains []IndustryChainSummary `json:"related_industry_chains"`
}
type IndustryChainDetail struct {
	Report        Summary       `json:"report"`
	IndustryChain IndustryChain `json:"industry_chain"`
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
