package report

import (
	"context"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

const (
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
	PublisherReportID string `json:"publisher_report_id"`
	Report            Report `json:"report"`
}

type CodedLabel struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

type Confidence struct {
	Code  string   `json:"code"`
	Label string   `json:"label"`
	Score *float64 `json:"score"`
}

type TimeWindow struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

type TransmissionTarget struct {
	Type     string     `json:"type"`
	LocalKey string     `json:"local_key"`
	Name     string     `json:"name"`
	Result   CodedLabel `json:"result"`
}

type Transmission struct {
	LocalKey          string               `json:"local_key"`
	SourceConclusion  string               `json:"source_conclusion"`
	Targets           []TransmissionTarget `json:"targets"`
	TransmissionLogic string               `json:"transmission_logic"`
	TransmissionKind  CodedLabel           `json:"transmission_kind"`
	Confidence        Confidence           `json:"confidence"`
	Status            CodedLabel           `json:"status"`
}

type LayerUncertainty struct {
	Counterevidence   *string `json:"counterevidence"`
	EvidenceGap       *string `json:"evidence_gap"`
	Boundary          *string `json:"boundary"`
	ReversalCondition *string `json:"reversal_condition"`
}

type LayerSummary struct {
	Conclusion           string           `json:"conclusion"`
	Result               CodedLabel       `json:"result"`
	Confidence           Confidence       `json:"confidence"`
	TimeWindow           TimeWindow       `json:"time_window"`
	DownwardTransmission []Transmission   `json:"downward_transmission"`
	Uncertainty          LayerUncertainty `json:"uncertainty"`
	EvidenceIDs          []string         `json:"evidence_ids"`
}

type Anchor struct {
	LocalKey          string      `json:"local_key"`
	Name              string      `json:"name"`
	CurrentState      string      `json:"current_state"`
	Result            CodedLabel  `json:"result"`
	ConclusionBasis   *CodedLabel `json:"conclusion_basis"`
	ValidationStatus  *CodedLabel `json:"validation_status"`
	TransmissionLogic string      `json:"transmission_logic"`
	TimeWindow        TimeWindow  `json:"time_window"`
	Confidence        Confidence  `json:"confidence"`
	EvidenceIDs       []string    `json:"evidence_ids"`
}

type ReasoningStep struct {
	Input         string     `json:"input"`
	Mechanism     string     `json:"mechanism"`
	Output        string     `json:"output"`
	ReasoningType CodedLabel `json:"reasoning_type"`
	Confidence    Confidence `json:"confidence"`
	EvidenceIDs   []string   `json:"evidence_ids"`
}

type LayerAnalysis struct {
	AffectedAnchors []Anchor        `json:"affected_anchors"`
	ReasoningSteps  []ReasoningStep `json:"reasoning_steps"`
}

type Layer struct {
	Title   string        `json:"title"`
	Summary LayerSummary  `json:"summary"`
	Detail  LayerAnalysis `json:"detail"`
}

type IndustryChainTopologyNode struct {
	LocalKey string `json:"local_key"`
	Name     string `json:"name"`
}

type IndustryChainEdge struct {
	FromNodeKey string     `json:"from_node_key"`
	ToNodeKey   string     `json:"to_node_key"`
	Relation    CodedLabel `json:"relation"`
}

type IndustryChainGraph struct {
	Nodes []IndustryChainTopologyNode `json:"nodes"`
	Edges []IndustryChainEdge         `json:"edges"`
}

type ChainSummary struct {
	Conclusion            string             `json:"conclusion"`
	Status                string             `json:"status"`
	Result                CodedLabel         `json:"result"`
	Confidence            Confidence         `json:"confidence"`
	TimeWindow            TimeWindow         `json:"time_window"`
	Path                  string             `json:"path"`
	Graph                 IndustryChainGraph `json:"graph"`
	CounterevidenceAndGap string             `json:"counterevidence_and_gap"`
	StopCondition         string             `json:"stop_condition"`
	EvidenceIDs           []string           `json:"evidence_ids"`
}

type IndustryChainNode struct {
	LocalKey          string      `json:"local_key"`
	NodeLocalKey      string      `json:"node_local_key"`
	Name              string      `json:"name"`
	Impact            string      `json:"impact"`
	Result            CodedLabel  `json:"result"`
	ConclusionBasis   *CodedLabel `json:"conclusion_basis"`
	ValidationStatus  *CodedLabel `json:"validation_status"`
	TransmissionLogic string      `json:"transmission_logic"`
	TimeWindow        TimeWindow  `json:"time_window"`
	Confidence        Confidence  `json:"confidence"`
	EvidenceIDs       []string    `json:"evidence_ids"`
}

type IndustryChainAnalysis struct {
	AffectedNodes []IndustryChainNode `json:"affected_nodes"`
}

type IndustryChain struct {
	LocalKey string                `json:"local_key"`
	Name     string                `json:"name"`
	Summary  ChainSummary          `json:"summary"`
	Detail   IndustryChainAnalysis `json:"detail"`
}

type Report struct {
	GeneratedAt    string          `json:"generated_at"`
	Geopolitics    *Layer          `json:"geopolitics,omitempty"`
	Macroeconomics *Layer          `json:"macroeconomics,omitempty"`
	IndustryChains []IndustryChain `json:"industry_chains"`
}

type PublicationResult struct {
	ReportID    string `json:"report_id"`
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
	ScopeToken      string
	HasUnknownQuery bool
}

type Summary struct {
	ID                 string `json:"id"`
	PublisherReportID  string `json:"publisher_report_id"`
	GeneratedAt        string `json:"generated_at"`
	HasGeopolitics     bool   `json:"has_geopolitics"`
	HasMacroeconomics  bool   `json:"has_macroeconomics"`
	IndustryChainCount int    `json:"industry_chain_count"`
	PublishedAt        string `json:"published_at"`
}

type Collection struct {
	Items      []Summary `json:"items"`
	NextCursor *string   `json:"next_cursor"`
}

type LayerSummaryProjection struct {
	Conclusion           string           `json:"conclusion"`
	Result               CodedLabel       `json:"result"`
	Confidence           Confidence       `json:"confidence"`
	TimeWindow           TimeWindow       `json:"time_window"`
	DownwardTransmission []Transmission   `json:"downward_transmission"`
	Uncertainty          LayerUncertainty `json:"uncertainty"`
	EvidenceScopeToken   *string          `json:"evidence_scope_token"`
}

type LayerSnapshot struct {
	Key     string                 `json:"key"`
	Title   string                 `json:"title"`
	Summary LayerSummaryProjection `json:"summary"`
}

type Home struct {
	Report         Summary        `json:"report"`
	Geopolitics    *LayerSnapshot `json:"geopolitics"`
	Macroeconomics *LayerSnapshot `json:"macroeconomics"`
}

type AnchorProjection struct {
	LocalKey           string      `json:"local_key"`
	Name               string      `json:"name"`
	CurrentState       string      `json:"current_state"`
	Result             CodedLabel  `json:"result"`
	ConclusionBasis    *CodedLabel `json:"conclusion_basis"`
	ValidationStatus   *CodedLabel `json:"validation_status"`
	TransmissionLogic  string      `json:"transmission_logic"`
	TimeWindow         TimeWindow  `json:"time_window"`
	Confidence         Confidence  `json:"confidence"`
	EvidenceScopeToken *string     `json:"evidence_scope_token"`
}

type ReasoningStepProjection struct {
	Input              string     `json:"input"`
	Mechanism          string     `json:"mechanism"`
	Output             string     `json:"output"`
	ReasoningType      CodedLabel `json:"reasoning_type"`
	Confidence         Confidence `json:"confidence"`
	EvidenceScopeToken *string    `json:"evidence_scope_token"`
}

type LayerProjection struct {
	Key             string                    `json:"key"`
	Title           string                    `json:"title"`
	Summary         LayerSummaryProjection    `json:"summary"`
	AffectedAnchors []AnchorProjection        `json:"affected_anchors"`
	ReasoningSteps  []ReasoningStepProjection `json:"reasoning_steps"`
}

type LayerDetail struct {
	Report Summary         `json:"report"`
	Layer  LayerProjection `json:"layer"`
}

type IndustryChainImpactSummary struct {
	LocalKey           string      `json:"local_key"`
	Name               string      `json:"name"`
	Result             CodedLabel  `json:"result"`
	ConclusionBasis    *CodedLabel `json:"conclusion_basis"`
	ValidationStatus   *CodedLabel `json:"validation_status"`
	Confidence         Confidence  `json:"confidence"`
	TimeWindow         TimeWindow  `json:"time_window"`
	EvidenceScopeToken *string     `json:"evidence_scope_token"`
}

type IndustryChainSummary struct {
	LocalKey           string                       `json:"local_key"`
	Name               string                       `json:"name"`
	Conclusion         string                       `json:"conclusion"`
	Status             string                       `json:"status"`
	Result             CodedLabel                   `json:"result"`
	Confidence         Confidence                   `json:"confidence"`
	TimeWindow         TimeWindow                   `json:"time_window"`
	ImpactItems        []IndustryChainImpactSummary `json:"impact_items"`
	EvidenceScopeToken *string                      `json:"evidence_scope_token"`
}

type IndustryChainCollection struct {
	Items      []IndustryChainSummary `json:"items"`
	NextCursor *string                `json:"next_cursor"`
}

type IndustryChainNodeProjection struct {
	LocalKey           string      `json:"local_key"`
	NodeLocalKey       string      `json:"node_local_key"`
	Name               string      `json:"name"`
	Impact             string      `json:"impact"`
	Result             CodedLabel  `json:"result"`
	ConclusionBasis    *CodedLabel `json:"conclusion_basis"`
	ValidationStatus   *CodedLabel `json:"validation_status"`
	TransmissionLogic  string      `json:"transmission_logic"`
	TimeWindow         TimeWindow  `json:"time_window"`
	Confidence         Confidence  `json:"confidence"`
	EvidenceScopeToken *string     `json:"evidence_scope_token"`
}

type IndustryChainProjection struct {
	LocalKey              string                        `json:"local_key"`
	Name                  string                        `json:"name"`
	Conclusion            string                        `json:"conclusion"`
	Status                string                        `json:"status"`
	Result                CodedLabel                    `json:"result"`
	Confidence            Confidence                    `json:"confidence"`
	TimeWindow            TimeWindow                    `json:"time_window"`
	Path                  string                        `json:"path"`
	Graph                 IndustryChainGraph            `json:"graph"`
	AffectedNodes         []IndustryChainNodeProjection `json:"affected_nodes"`
	CounterevidenceAndGap string                        `json:"counterevidence_and_gap"`
	StopCondition         string                        `json:"stop_condition"`
	EvidenceScopeToken    *string                       `json:"evidence_scope_token"`
}

type IndustryChainDetail struct {
	Report        Summary                 `json:"report"`
	IndustryChain IndustryChainProjection `json:"industry_chain"`
}

type EvidenceItem struct {
	PublishedAt *string  `json:"published_at"`
	Summary     string   `json:"summary"`
	Keywords    []string `json:"keywords"`
}

type EvidenceCollection struct {
	ReportID   string         `json:"report_id"`
	ScopeToken string         `json:"scope_token"`
	Items      []EvidenceItem `json:"items"`
}
