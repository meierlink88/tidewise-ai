package report

import (
	"context"
	"encoding/json"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

const (
	OperationListReportAnalyses         = "data.v1.listReportAnalyses"
	OperationGetReportAnalysis          = "data.v1.getReportAnalysis"
	OperationGetReportAnalysisChain     = "data.v1.getReportAnalysisChain"
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
	return []string{OperationListReportAnalyses, OperationGetReportAnalysis, OperationGetReportAnalysisChain, OperationPublishReport, OperationListReports, OperationGetReportHome, OperationGetReportLayer, OperationListReportChains, OperationGetReportChain, OperationListReportEvidence}
}

type Service interface {
	ListReportAnalyses(context.Context, *AnalysisRequest) (*v1.Response[AnalysisCollection], error)
	GetReportAnalysis(context.Context, *AnalysisRequest) (*v1.Response[AnalysisUnitDetail], error)
	GetReportAnalysisChain(context.Context, *AnalysisRequest) (*v1.Response[ChainAnalysisDetail], error)
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

type Confidence = CodedLabel

type TimeWindow struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

type EvidenceReference struct {
	EvidenceID string     `json:"evidence_id"`
	Role       CodedLabel `json:"role"`
}

type TransmissionTarget struct {
	TargetType     CodedLabel `json:"target_type"`
	TargetLocalKey string     `json:"target_local_key"`
	TargetName     string     `json:"target_name"`
	Result         CodedLabel `json:"result"`
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

type TransmissionGroup struct {
	Summary string         `json:"summary"`
	Paths   []Transmission `json:"paths"`
}

type DownwardTransmission struct {
	ToMacroeconomics *TransmissionGroup `json:"to_macroeconomics,omitempty"`
	ToIndustryChains *TransmissionGroup `json:"to_industry_chains,omitempty"`
}

type LayerUncertainty struct {
	Counterevidence   *string `json:"counterevidence"`
	EvidenceGap       *string `json:"evidence_gap"`
	Boundary          *string `json:"boundary"`
	ReversalCondition *string `json:"reversal_condition"`
}

type Anchor struct {
	LocalKey         string              `json:"local_key"`
	Name             string              `json:"name"`
	CurrentState     string              `json:"current_state"`
	Result           CodedLabel          `json:"result"`
	ConclusionBasis  CodedLabel          `json:"conclusion_basis"`
	ValidationStatus CodedLabel          `json:"validation_status"`
	Reasoning        string              `json:"reasoning"`
	TimeWindow       TimeWindow          `json:"time_window"`
	Confidence       Confidence          `json:"confidence"`
	EvidenceRefs     []EvidenceReference `json:"evidence_refs"`
}

type ReasoningStep struct {
	LocalKey     string              `json:"local_key"`
	Input        string              `json:"input"`
	Mechanism    string              `json:"mechanism"`
	Output       string              `json:"output"`
	Confidence   Confidence          `json:"confidence"`
	EvidenceRefs []EvidenceReference `json:"evidence_refs"`
}

type Layer struct {
	LocalKey             string               `json:"local_key"`
	Title                string               `json:"title"`
	Conclusion           string               `json:"conclusion"`
	Result               CodedLabel           `json:"result"`
	TimeWindow           TimeWindow           `json:"time_window"`
	Confidence           Confidence           `json:"confidence"`
	AffectedAnchors      []Anchor             `json:"affected_anchors"`
	ReasoningSteps       []ReasoningStep      `json:"reasoning_steps"`
	Uncertainty          LayerUncertainty     `json:"uncertainty"`
	EvidenceRefs         []EvidenceReference  `json:"evidence_refs"`
	DownwardTransmission DownwardTransmission `json:"downward_transmission"`
}

type IndustryChainEdge struct {
	FromNodeLocalKey string `json:"from_node_local_key"`
	ToNodeLocalKey   string `json:"to_node_local_key"`
	RelationLabel    string `json:"relation_label"`
}

type IndustryChainNode struct {
	LocalKey         string              `json:"local_key"`
	Name             string              `json:"name"`
	Impact           string              `json:"impact"`
	Result           CodedLabel          `json:"result"`
	ConclusionBasis  CodedLabel          `json:"conclusion_basis"`
	ValidationStatus CodedLabel          `json:"validation_status"`
	Reasoning        string              `json:"reasoning"`
	TimeWindow       TimeWindow          `json:"time_window"`
	Confidence       Confidence          `json:"confidence"`
	EvidenceRefs     []EvidenceReference `json:"evidence_refs"`
}

type ChainUncertainty struct {
	CounterevidenceAndGap *string `json:"counterevidence_and_gap"`
	StopCondition         *string `json:"stop_condition"`
}

type IndustryChain struct {
	LocalKey                  string              `json:"local_key"`
	Name                      string              `json:"name"`
	Conclusion                string              `json:"conclusion"`
	Result                    CodedLabel          `json:"result"`
	TimeWindow                TimeWindow          `json:"time_window"`
	Confidence                Confidence          `json:"confidence"`
	PathSummary               *string             `json:"path_summary"`
	AcceptedHypothesisSummary *string             `json:"accepted_hypothesis_summary"`
	Nodes                     []IndustryChainNode `json:"nodes"`
	Edges                     []IndustryChainEdge `json:"edges"`
	Uncertainty               ChainUncertainty    `json:"uncertainty"`
	EvidenceRefs              []EvidenceReference `json:"evidence_refs"`
}

type Report struct {
	SchemaVersion        string          `json:"schema_version,omitempty"`
	AnalysisWindow       *AnalysisWindow `json:"analysis_window,omitempty"`
	GeopoliticalStories  []AnalysisUnit  `json:"geopolitical_stories,omitempty"`
	MacroeconomicStories []AnalysisUnit  `json:"macroeconomic_stories,omitempty"`
	ConceptAnalyses      []AnalysisUnit  `json:"concept_analyses,omitempty"`

	ReportType     CodedLabel      `json:"report_type"`
	GeneratedAt    string          `json:"generated_at"`
	Timezone       string          `json:"timezone"`
	Geopolitics    *Layer          `json:"geopolitics,omitempty"`
	Macroeconomics *Layer          `json:"macroeconomics,omitempty"`
	IndustryChains []IndustryChain `json:"industry_chains,omitempty"`
}

type PublicationResult struct {
	ReportID    string `json:"report_id"`
	PublishedAt string `json:"published_at"`
	Replayed    bool   `json:"replayed"`
}

type ListRequest struct {
	SchemaVersion string
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
	SchemaVersion      string          `json:"schema_version,omitempty"`
	AnalysisWindow     *AnalysisWindow `json:"analysis_window,omitempty"`
	ID                 string          `json:"id"`
	PublisherReportID  string          `json:"publisher_report_id"`
	GeneratedAt        string          `json:"generated_at"`
	HasGeopolitics     bool            `json:"has_geopolitics"`
	HasMacroeconomics  bool            `json:"has_macroeconomics"`
	IndustryChainCount int             `json:"industry_chain_count"`
	PublishedAt        string          `json:"published_at"`
}

type Collection struct {
	Items      []Summary `json:"items"`
	NextCursor *string   `json:"next_cursor"`
}

type LayerSummaryProjection struct {
	Conclusion           string               `json:"conclusion"`
	Result               CodedLabel           `json:"result"`
	Confidence           Confidence           `json:"confidence"`
	TimeWindow           TimeWindow           `json:"time_window"`
	DownwardTransmission DownwardTransmission `json:"downward_transmission"`
	Uncertainty          LayerUncertainty     `json:"uncertainty"`
	EvidenceScopeToken   *string              `json:"evidence_scope_token"`
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
	LocalKey           string     `json:"local_key"`
	Name               string     `json:"name"`
	CurrentState       string     `json:"current_state"`
	Result             CodedLabel `json:"result"`
	ConclusionBasis    CodedLabel `json:"conclusion_basis"`
	ValidationStatus   CodedLabel `json:"validation_status"`
	Reasoning          string     `json:"reasoning"`
	TimeWindow         TimeWindow `json:"time_window"`
	Confidence         Confidence `json:"confidence"`
	EvidenceScopeToken *string    `json:"evidence_scope_token"`
}

type ReasoningStepProjection struct {
	LocalKey           string     `json:"local_key"`
	Input              string     `json:"input"`
	Mechanism          string     `json:"mechanism"`
	Output             string     `json:"output"`
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
	LocalKey           string     `json:"local_key"`
	Name               string     `json:"name"`
	Result             CodedLabel `json:"result"`
	ConclusionBasis    CodedLabel `json:"conclusion_basis"`
	ValidationStatus   CodedLabel `json:"validation_status"`
	Confidence         Confidence `json:"confidence"`
	TimeWindow         TimeWindow `json:"time_window"`
	EvidenceScopeToken *string    `json:"evidence_scope_token"`
}

type IndustryChainSummary struct {
	LocalKey           string                       `json:"local_key"`
	Name               string                       `json:"name"`
	Conclusion         string                       `json:"conclusion"`
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
	LocalKey           string     `json:"local_key"`
	Name               string     `json:"name"`
	Impact             string     `json:"impact"`
	Result             CodedLabel `json:"result"`
	ConclusionBasis    CodedLabel `json:"conclusion_basis"`
	ValidationStatus   CodedLabel `json:"validation_status"`
	Reasoning          string     `json:"reasoning"`
	TimeWindow         TimeWindow `json:"time_window"`
	Confidence         Confidence `json:"confidence"`
	EvidenceScopeToken *string    `json:"evidence_scope_token"`
}

type IndustryChainTopologyNode struct {
	LocalKey string `json:"local_key"`
	Name     string `json:"name"`
}

type IndustryChainEdgeProjection struct {
	FromNodeLocalKey string `json:"from_node_local_key"`
	ToNodeLocalKey   string `json:"to_node_local_key"`
	RelationLabel    string `json:"relation_label"`
}

type IndustryChainGraph struct {
	Nodes []IndustryChainTopologyNode   `json:"nodes"`
	Edges []IndustryChainEdgeProjection `json:"edges"`
}

type IndustryChainProjection struct {
	LocalKey                  string                        `json:"local_key"`
	Name                      string                        `json:"name"`
	Conclusion                string                        `json:"conclusion"`
	Result                    CodedLabel                    `json:"result"`
	Confidence                Confidence                    `json:"confidence"`
	TimeWindow                TimeWindow                    `json:"time_window"`
	PathSummary               *string                       `json:"path_summary"`
	AcceptedHypothesisSummary *string                       `json:"accepted_hypothesis_summary"`
	Graph                     IndustryChainGraph            `json:"graph"`
	AffectedNodes             []IndustryChainNodeProjection `json:"affected_nodes"`
	CounterevidenceAndGap     *string                       `json:"counterevidence_and_gap"`
	StopCondition             *string                       `json:"stop_condition"`
	EvidenceScopeToken        *string                       `json:"evidence_scope_token"`
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

// AnalysisWindow records the publisher's observation interval, separately from the forecast horizon.
type AnalysisWindow struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type AnalysisSummary struct {
	Conclusion        string              `json:"conclusion"`
	TransmissionLogic string              `json:"transmission_logic"`
	AnchorKeys        []string            `json:"anchor_keys"`
	EvidenceRefs      []EvidenceReference `json:"evidence_refs"`
}

type AnalysisImpact struct {
	LocalKey           string              `json:"local_key"`
	TargetType         CodedLabel          `json:"target_type"`
	SourceID           string              `json:"source_id"`
	NodeLocalKey       *string             `json:"node_local_key"`
	Name               string              `json:"name"`
	Impact             string              `json:"impact"`
	Result             CodedLabel          `json:"result"`
	ConclusionBasis    CodedLabel          `json:"conclusion_basis"`
	ValidationStatus   CodedLabel          `json:"validation_status"`
	Reasoning          string              `json:"reasoning"`
	TransmissionSignal *string             `json:"transmission_signal"`
	Conditions         []string            `json:"conditions"`
	FollowUp           []string            `json:"follow_up"`
	TimeWindow         TimeWindow          `json:"time_window"`
	Confidence         Confidence          `json:"confidence"`
	EvidenceRefs       []EvidenceReference `json:"evidence_refs"`
}

type AnalysisTopologyNode struct {
	LocalKey string `json:"local_key"`
	SourceID string `json:"source_id"`
	Name     string `json:"name"`
}

type AnalysisGraph struct {
	Nodes []AnalysisTopologyNode `json:"nodes"`
	Edges []IndustryChainEdge    `json:"edges"`
}

type ChainAnalysis struct {
	LocalKey          string              `json:"local_key"`
	SourceID          string              `json:"source_id"`
	Name              string              `json:"name"`
	Conclusion        string              `json:"conclusion"`
	TransmissionLogic string              `json:"transmission_logic"`
	ReasoningSteps    []ReasoningStep     `json:"reasoning_steps"`
	Graph             AnalysisGraph       `json:"graph"`
	AffectedNodes     []AnalysisImpact    `json:"affected_nodes"`
	Uncertainty       LayerUncertainty    `json:"uncertainty"`
	EvidenceRefs      []EvidenceReference `json:"evidence_refs"`
}

type AnalysisDetail struct {
	ReasoningSteps  []ReasoningStep  `json:"reasoning_steps"`
	AffectedAnchors []AnalysisImpact `json:"affected_anchors"`
	Uncertainty     LayerUncertainty `json:"uncertainty"`
	IndustryChains  []ChainAnalysis  `json:"industry_chains"`
}

type AnalysisUnit struct {
	LocalKey string          `json:"local_key"`
	SourceID string          `json:"source_id"`
	Title    string          `json:"title"`
	Summary  AnalysisSummary `json:"summary"`
	Detail   AnalysisDetail  `json:"detail"`
}
type AnalysisImpactProjection struct {
	LocalKey           string     `json:"local_key"`
	TargetType         CodedLabel `json:"target_type"`
	SourceID           string     `json:"source_id"`
	NodeLocalKey       *string    `json:"node_local_key"`
	Name               string     `json:"name"`
	Impact             string     `json:"impact"`
	Result             CodedLabel `json:"result"`
	ConclusionBasis    CodedLabel `json:"conclusion_basis"`
	ValidationStatus   CodedLabel `json:"validation_status"`
	Reasoning          string     `json:"reasoning"`
	TransmissionSignal *string    `json:"transmission_signal"`
	Conditions         []string   `json:"conditions"`
	FollowUp           []string   `json:"follow_up"`
	TimeWindow         TimeWindow `json:"time_window"`
	Confidence         Confidence `json:"confidence"`
	EvidenceScopeToken *string    `json:"evidence_scope_token"`
}

type AnalysisUnitSummary struct {
	LocalKey           string                     `json:"local_key"`
	SourceID           string                     `json:"source_id"`
	Title              string                     `json:"title"`
	Conclusion         string                     `json:"conclusion"`
	TransmissionLogic  string                     `json:"transmission_logic"`
	AffectedAnchors    []AnalysisImpactProjection `json:"affected_anchors"`
	ChainCount         int                        `json:"chain_count"`
	EvidenceScopeToken *string                    `json:"evidence_scope_token"`
	Ordinal            int                        `json:"-"`
}
type ChainAnalysisSummary struct {
	LocalKey   string `json:"local_key"`
	SourceID   string `json:"source_id"`
	Name       string `json:"name"`
	Conclusion string `json:"conclusion"`
}
type AnalysisUnitDetail struct {
	Summary         AnalysisUnitSummary        `json:"summary"`
	ReasoningSteps  []ReasoningStepProjection  `json:"reasoning_steps"`
	AffectedAnchors []AnalysisImpactProjection `json:"affected_anchors"`
	Uncertainty     LayerUncertainty           `json:"uncertainty"`
	IndustryChains  []ChainAnalysisSummary     `json:"industry_chains"`
}
type ChainAnalysisDetail struct {
	LocalKey           string                     `json:"local_key"`
	SourceID           string                     `json:"source_id"`
	Name               string                     `json:"name"`
	Conclusion         string                     `json:"conclusion"`
	TransmissionLogic  string                     `json:"transmission_logic"`
	ReasoningSteps     []ReasoningStepProjection  `json:"reasoning_steps"`
	Graph              AnalysisGraph              `json:"graph"`
	AffectedNodes      []AnalysisImpactProjection `json:"affected_nodes"`
	Uncertainty        LayerUncertainty           `json:"uncertainty"`
	EvidenceScopeToken *string                    `json:"evidence_scope_token"`
}

func (r Report) MarshalJSON() ([]byte, error) {
	type plain Report
	if r.SchemaVersion == "" {
		return json.Marshal(plain(r))
	}
	return json.Marshal(struct {
		plain
		GeopoliticalStories  []AnalysisUnit `json:"geopolitical_stories"`
		MacroeconomicStories []AnalysisUnit `json:"macroeconomic_stories"`
		ConceptAnalyses      []AnalysisUnit `json:"concept_analyses"`
	}{plain(r), r.GeopoliticalStories, r.MacroeconomicStories, r.ConceptAnalyses})
}

type AnalysisRequest struct{ ReportID, Kind, AnalysisKey, ChainKey, Limit, Cursor string }
type AnalysisCollection struct {
	Items      []AnalysisUnitSummary `json:"items"`
	NextCursor *string               `json:"next_cursor"`
}
