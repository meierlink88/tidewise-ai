package report

import (
	"bytes"
	"context"
	"encoding/json"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

const (
	OperationGetReportAnalysisUnitChain = "data.v1.getReportAnalysisUnitChain"
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
	return []string{OperationGetReportAnalysisUnitChain, OperationListReportAnalyses, OperationGetReportAnalysis, OperationGetReportAnalysisChain, OperationPublishReport, OperationListReports, OperationGetReportHome, OperationGetReportLayer, OperationListReportChains, OperationGetReportChain, OperationListReportEvidence}
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
	V4                   *V4Report       `json:"-"`
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
	V4             *V4HomeProjection `json:"-"`
	Report         Summary           `json:"report"`
	Geopolitics    *LayerSnapshot    `json:"geopolitics"`
	Macroeconomics *LayerSnapshot    `json:"macroeconomics"`
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

// ImpactAssessment describes conditional consequence magnitude, independently of direction and confidence.
type ImpactAssessment struct {
	Level        CodedLabel          `json:"level"`
	Rationale    string              `json:"rationale"`
	EvidenceRefs []EvidenceReference `json:"evidence_refs"`
}

type ImpactAssessmentProjection struct {
	Level              CodedLabel `json:"level"`
	Rationale          string     `json:"rationale"`
	EvidenceScopeToken *string    `json:"evidence_scope_token"`
}

type AnalysisSummary struct {
	ImpactAssessment  *ImpactAssessment   `json:"impact_assessment,omitempty"`
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
	Company            *V5CompanyProjection        `json:"-"`
	V4                 *V4SummaryProjection        `json:"-"`
	ImpactAssessment   *ImpactAssessmentProjection `json:"impact_assessment,omitempty"`
	LocalKey           string                      `json:"local_key"`
	SourceID           string                      `json:"source_id"`
	Title              string                      `json:"title"`
	Conclusion         string                      `json:"conclusion"`
	TransmissionLogic  string                      `json:"transmission_logic"`
	AffectedAnchors    []AnalysisImpactProjection  `json:"affected_anchors"`
	ChainCount         int                         `json:"chain_count"`
	EvidenceScopeToken *string                     `json:"evidence_scope_token"`
	Ordinal            int                         `json:"-"`
}
type ChainAnalysisSummary struct {
	LocalKey   string `json:"local_key"`
	SourceID   string `json:"source_id"`
	Name       string `json:"name"`
	Conclusion string `json:"conclusion"`
}
type AnalysisUnitDetail struct {
	Company         *V5CompanyProjection       `json:"-"`
	V4              *V4DetailProjection        `json:"-"`
	Summary         AnalysisUnitSummary        `json:"summary"`
	ReasoningSteps  []ReasoningStepProjection  `json:"reasoning_steps"`
	AffectedAnchors []AnalysisImpactProjection `json:"affected_anchors"`
	Uncertainty     LayerUncertainty           `json:"uncertainty"`
	IndustryChains  []ChainAnalysisSummary     `json:"industry_chains"`
}
type ChainAnalysisDetail struct {
	V4                 *V4ReadChain               `json:"-"`
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
	if r.V4 != nil {
		return json.Marshal(r.V4)
	}
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

// V4 contracts implement the approved normalized report, independently of legacy snapshots.
const NormalizedSchemaVersion = "report-publication/v4"
const SignalSchemaVersion = "report-publication/v5"

type V4CodedLabel struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}
type V4Claim struct {
	Text        string   `json:"text"`
	Basis       string   `json:"basis"`
	EvidenceIDs []string `json:"evidence_ids"`
}
type V4Objections struct {
	Summary               string    `json:"summary"`
	Counterevidence       []V4Claim `json:"counterevidence"`
	Buffers               []V4Claim `json:"buffers"`
	CounterevidenceStatus string    `json:"counterevidence_status"`
	EvidenceGaps          []string  `json:"evidence_gaps"`
	ScopeLimits           []string  `json:"scope_limits"`
}
type V4Window struct {
	Kind        string  `json:"kind"`
	Description string  `json:"description"`
	StartAt     *string `json:"start_at"`
	EndAt       *string `json:"end_at"`
}
type V4Assessment struct {
	Conclusion        string   `json:"conclusion"`
	Direction         string   `json:"direction"`
	ConclusionBasis   string   `json:"conclusion_basis"`
	ValidationStatus  string   `json:"validation_status"`
	Confidence        *string  `json:"confidence"`
	ForecastWindow    V4Window `json:"forecast_window"`
	Scope             string   `json:"scope"`
	Conditions        []string `json:"conditions"`
	FollowUp          []string `json:"follow_up"`
	TransmissionLogic string   `json:"transmission_logic"`
	EvidenceIDs       []string `json:"evidence_ids"`
}
type V4Node struct {
	JudgmentOrigin   string              `json:"judgment_origin,omitempty"`
	ReasoningSources *V5ReasoningSources `json:"reasoning_sources,omitempty"`
	VariableSignals  *[]V5Signal         `json:"variable_signals,omitempty"`
	LocalKey         string              `json:"local_key"`
	SourceID         string              `json:"source_id"`
	NodeLocalKey     string              `json:"node_local_key"`
	Name             string              `json:"name"`
	Assessment       V4Assessment        `json:"assessment"`
	Objections       V4Objections        `json:"objections"`
}
type V4Graph struct {
	Scope string             `json:"scope,omitempty"`
	Nodes []V4GraphNodesItem `json:"nodes"`
	Edges []V4GraphEdgesItem `json:"edges"`
}
type V4Chain struct {
	JudgmentOrigin   string                  `json:"judgment_origin,omitempty"`
	ReasoningSources *V5ReasoningSources     `json:"reasoning_sources,omitempty"`
	VariableSignals  *[]V5Signal             `json:"variable_signals,omitempty"`
	LocalKey         string                  `json:"local_key"`
	SourceID         string                  `json:"source_id"`
	Name             string                  `json:"name"`
	Assessment       V4Assessment            `json:"assessment"`
	ReasoningSummary V4ChainReasoningSummary `json:"reasoning_summary"`
	Graph            V4Graph                 `json:"graph"`
	AffectedNodes    []V4Node                `json:"affected_nodes"`
	EmptyState       *V4ChainEmptyState      `json:"empty_state"`
}
type V4Macro struct {
	JudgmentOrigin   string              `json:"judgment_origin,omitempty"`
	ReasoningSources *V5ReasoningSources `json:"reasoning_sources,omitempty"`
	VariableSignals  *[]V5Signal         `json:"variable_signals,omitempty"`
	LocalKey         string              `json:"local_key"`
	SourceID         string              `json:"source_id"`
	Name             string              `json:"name"`
	Assessment       V4Assessment        `json:"assessment"`
	Objections       V4Objections        `json:"objections"`
}
type V4AnchorRef struct {
	TargetType    string  `json:"target_type"`
	LocalKey      string  `json:"local_key"`
	ChainLocalKey *string `json:"chain_local_key"`
}
type V4Unit struct {
	JudgmentOrigin   string              `json:"judgment_origin,omitempty"`
	ReasoningSources *V5ReasoningSources `json:"reasoning_sources,omitempty"`
	LocalKey         string              `json:"local_key"`
	SourceID         string              `json:"source_id"`
	Title            string              `json:"title"`
	Summary          V4UnitSummary       `json:"summary"`
	Detail           V4UnitDetail        `json:"detail"`
}
type V4Report struct {
	IndustryChainAnalyses *[]V4Unit                  `json:"industry_chain_analyses,omitempty"`
	CompanyAnalyses       *[]V4Macro                 `json:"company_analyses,omitempty"`
	SchemaVersion         string                     `json:"schema_version"`
	ReportType            V4CodedLabel               `json:"report_type"`
	GeneratedAt           string                     `json:"generated_at"`
	Timezone              string                     `json:"timezone"`
	AnalysisWindow        V4ReportAnalysisWindow     `json:"analysis_window"`
	GeopoliticalStories   []V4Unit                   `json:"geopolitical_stories"`
	MacroeconomicStories  []V4Unit                   `json:"macroeconomic_stories"`
	ConceptAnalyses       []V4Unit                   `json:"concept_analyses"`
	Observations          []V4ReportObservationsItem `json:"observations"`
	Limitations           []string                   `json:"limitations"`
}
type V4GraphNodesItem struct {
	LocalKey string `json:"local_key"`
	SourceID string `json:"source_id"`
	Name     string `json:"name"`
}
type V4GraphEdgesItem struct {
	FromNodeLocalKey string `json:"from_node_local_key"`
	ToNodeLocalKey   string `json:"to_node_local_key"`
	RelationLabel    string `json:"relation_label"`
}
type V4ChainReasoningSummary struct {
	Logic      string       `json:"logic"`
	Support    V4Claim      `json:"support"`
	Objections V4Objections `json:"objections"`
}
type V4ChainEmptyState struct {
	Code     string   `json:"code"`
	Reason   string   `json:"reason"`
	FollowUp []string `json:"follow_up"`
}
type V4UnitSummary struct {
	Conclusion        string                        `json:"conclusion"`
	TransmissionLogic string                        `json:"transmission_logic"`
	ImpactAssessment  V4UnitSummaryImpactAssessment `json:"impact_assessment"`
	AffectedRefs      []V4AnchorRef                 `json:"affected_refs"`
	EvidenceIDs       []string                      `json:"evidence_ids"`
}
type V4UnitDetail struct {
	VariableSignals *[]V5Signal `json:"variable_signals,omitempty"`
	Companies       *[]V4Macro  `json:"companies,omitempty"`
	MacroImpacts    []V4Macro   `json:"macro_impacts"`
	IndustryChains  []V4Chain   `json:"industry_chains"`
}
type V4ReportAnalysisWindow struct {
	Start string `json:"start"`
	End   string `json:"end"`
}
type V4ReportObservationsItem struct {
	LocalKey    string   `json:"local_key"`
	Title       string   `json:"title"`
	Text        string   `json:"text"`
	EvidenceIDs []string `json:"evidence_ids"`
}
type V4UnitSummaryImpactAssessment struct {
	Level       string   `json:"level"`
	Rationale   string   `json:"rationale"`
	EvidenceIDs []string `json:"evidence_ids"`
}

func (r *Report) UnmarshalJSON(payload []byte) error {
	var probe struct {
		SchemaVersion string `json:"schema_version"`
	}
	if err := json.Unmarshal(payload, &probe); err != nil {
		return err
	}
	if probe.SchemaVersion == NormalizedSchemaVersion || probe.SchemaVersion == SignalSchemaVersion {
		var parsed V4Report
		decoder := json.NewDecoder(bytes.NewReader(payload))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&parsed); err != nil {
			return err
		}
		*r = Report{V4: &parsed, SchemaVersion: parsed.SchemaVersion, ReportType: CodedLabel{Code: parsed.ReportType.Code, Label: parsed.ReportType.Label}, GeneratedAt: parsed.GeneratedAt, Timezone: parsed.Timezone}
		return nil
	}
	type plain Report
	var v plain
	d := json.NewDecoder(bytes.NewReader(payload))
	d.DisallowUnknownFields()
	if err := d.Decode(&v); err != nil {
		return err
	}
	*r = Report(v)
	return nil
}

type V4ReadCodedLabel struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}
type V4ReadClaim struct {
	Text               string  `json:"text"`
	Basis              string  `json:"basis"`
	EvidenceScopeToken *string `json:"evidence_scope_token"`
	EvidenceCount      int     `json:"evidence_count"`
}
type V4ReadObjections struct {
	Summary               string        `json:"summary"`
	Counterevidence       []V4ReadClaim `json:"counterevidence"`
	Buffers               []V4ReadClaim `json:"buffers"`
	CounterevidenceStatus string        `json:"counterevidence_status"`
	EvidenceGaps          []string      `json:"evidence_gaps"`
	ScopeLimits           []string      `json:"scope_limits"`
}
type V4ReadWindow struct {
	Kind        string  `json:"kind"`
	Description string  `json:"description"`
	StartAt     *string `json:"start_at"`
	EndAt       *string `json:"end_at"`
}
type V4ReadAssessment struct {
	Conclusion         string       `json:"conclusion"`
	Direction          string       `json:"direction"`
	ConclusionBasis    string       `json:"conclusion_basis"`
	ValidationStatus   string       `json:"validation_status"`
	Confidence         *string      `json:"confidence"`
	ForecastWindow     V4ReadWindow `json:"forecast_window"`
	Scope              string       `json:"scope"`
	Conditions         []string     `json:"conditions"`
	FollowUp           []string     `json:"follow_up"`
	TransmissionLogic  string       `json:"transmission_logic"`
	EvidenceScopeToken *string      `json:"evidence_scope_token"`
	EvidenceCount      int          `json:"evidence_count"`
}
type V4ReadNode struct {
	JudgmentOrigin   string              `json:"judgment_origin,omitempty"`
	ReasoningSources *V5ReasoningSources `json:"reasoning_sources,omitempty"`
	VariableSignals  *[]V5ReadSignal     `json:"variable_signals,omitempty"`
	LocalKey         string              `json:"local_key"`
	SourceID         string              `json:"source_id"`
	NodeLocalKey     string              `json:"node_local_key"`
	Name             string              `json:"name"`
	Assessment       V4ReadAssessment    `json:"assessment"`
	Objections       V4ReadObjections    `json:"objections"`
}
type V4ReadGraph struct {
	Scope string                 `json:"scope,omitempty"`
	Nodes []V4ReadGraphNodesItem `json:"nodes"`
	Edges []V4ReadGraphEdgesItem `json:"edges"`
}
type V4ReadChain struct {
	JudgmentOrigin   string                      `json:"judgment_origin,omitempty"`
	ReasoningSources *V5ReasoningSources         `json:"reasoning_sources,omitempty"`
	VariableSignals  *[]V5ReadSignal             `json:"variable_signals,omitempty"`
	LocalKey         string                      `json:"local_key"`
	SourceID         string                      `json:"source_id"`
	Name             string                      `json:"name"`
	Assessment       V4ReadAssessment            `json:"assessment"`
	ReasoningSummary V4ReadChainReasoningSummary `json:"reasoning_summary"`
	Graph            V4ReadGraph                 `json:"graph"`
	AffectedNodes    []V4ReadNode                `json:"affected_nodes"`
	EmptyState       *V4ReadChainEmptyState      `json:"empty_state"`
}
type V4ReadMacro struct {
	JudgmentOrigin   string              `json:"judgment_origin,omitempty"`
	ReasoningSources *V5ReasoningSources `json:"reasoning_sources,omitempty"`
	VariableSignals  *[]V5ReadSignal     `json:"variable_signals,omitempty"`
	LocalKey         string              `json:"local_key"`
	SourceID         string              `json:"source_id"`
	Name             string              `json:"name"`
	Assessment       V4ReadAssessment    `json:"assessment"`
	Objections       V4ReadObjections    `json:"objections"`
}
type V4ReadAnchorRef struct {
	TargetType    string  `json:"target_type"`
	LocalKey      string  `json:"local_key"`
	ChainLocalKey *string `json:"chain_local_key"`
}
type V4ReadUnit struct {
	JudgmentOrigin   string              `json:"judgment_origin,omitempty"`
	ReasoningSources *V5ReasoningSources `json:"reasoning_sources,omitempty"`
	LocalKey         string              `json:"local_key"`
	SourceID         string              `json:"source_id"`
	Title            string              `json:"title"`
	Summary          V4ReadUnitSummary   `json:"summary"`
	Detail           V4ReadUnitDetail    `json:"detail"`
}
type V4ReadReport struct {
	IndustryChainAnalyses *[]V4ReadUnit                  `json:"industry_chain_analyses,omitempty"`
	CompanyAnalyses       *[]V4ReadMacro                 `json:"company_analyses,omitempty"`
	SchemaVersion         string                         `json:"schema_version"`
	ReportType            V4ReadCodedLabel               `json:"report_type"`
	GeneratedAt           string                         `json:"generated_at"`
	Timezone              string                         `json:"timezone"`
	AnalysisWindow        V4ReadReportAnalysisWindow     `json:"analysis_window"`
	GeopoliticalStories   []V4ReadUnit                   `json:"geopolitical_stories"`
	MacroeconomicStories  []V4ReadUnit                   `json:"macroeconomic_stories"`
	ConceptAnalyses       []V4ReadUnit                   `json:"concept_analyses"`
	Observations          []V4ReadReportObservationsItem `json:"observations"`
	Limitations           []string                       `json:"limitations"`
}
type V4ReadGraphNodesItem struct {
	LocalKey string `json:"local_key"`
	SourceID string `json:"source_id"`
	Name     string `json:"name"`
}
type V4ReadGraphEdgesItem struct {
	FromNodeLocalKey string `json:"from_node_local_key"`
	ToNodeLocalKey   string `json:"to_node_local_key"`
	RelationLabel    string `json:"relation_label"`
}
type V4ReadChainReasoningSummary struct {
	Logic      string           `json:"logic"`
	Support    V4ReadClaim      `json:"support"`
	Objections V4ReadObjections `json:"objections"`
}
type V4ReadChainEmptyState struct {
	Code     string   `json:"code"`
	Reason   string   `json:"reason"`
	FollowUp []string `json:"follow_up"`
}
type V4ReadUnitSummary struct {
	Conclusion         string                            `json:"conclusion"`
	TransmissionLogic  string                            `json:"transmission_logic"`
	ImpactAssessment   V4ReadUnitSummaryImpactAssessment `json:"impact_assessment"`
	AffectedRefs       []V4ReadAnchorRef                 `json:"affected_refs"`
	EvidenceScopeToken *string                           `json:"evidence_scope_token"`
	EvidenceCount      int                               `json:"evidence_count"`
}
type V4ReadUnitDetail struct {
	VariableSignals *[]V5ReadSignal `json:"variable_signals,omitempty"`
	Companies       *[]V4ReadMacro  `json:"companies,omitempty"`
	MacroImpacts    []V4ReadMacro   `json:"macro_impacts"`
	IndustryChains  []V4ReadChain   `json:"industry_chains"`
}
type V4ReadReportAnalysisWindow struct {
	Start string `json:"start"`
	End   string `json:"end"`
}
type V4ReadReportObservationsItem struct {
	LocalKey           string  `json:"local_key"`
	Title              string  `json:"title"`
	Text               string  `json:"text"`
	EvidenceScopeToken *string `json:"evidence_scope_token"`
	EvidenceCount      int     `json:"evidence_count"`
}
type V4ReadUnitSummaryImpactAssessment struct {
	Level              string  `json:"level"`
	Rationale          string  `json:"rationale"`
	EvidenceScopeToken *string `json:"evidence_scope_token"`
	EvidenceCount      int     `json:"evidence_count"`
}

type V4ResolvedAnchor struct {
	JudgmentOrigin string           `json:"judgment_origin,omitempty"`
	Reference      V4AnchorRef      `json:"reference"`
	SourceID       string           `json:"source_id"`
	Name           string           `json:"name"`
	Assessment     V4ReadAssessment `json:"assessment"`
}
type V4SummaryProjection struct {
	JudgmentOrigin  string             `json:"judgment_origin,omitempty"`
	SchemaVersion   string             `json:"schema_version"`
	LocalKey        string             `json:"local_key"`
	SourceID        string             `json:"source_id"`
	Title           string             `json:"title"`
	Summary         V4ReadUnitSummary  `json:"summary"`
	AffectedAnchors []V4ResolvedAnchor `json:"affected_anchors"`
	ChainCount      int                `json:"chain_count"`
}
type V4ChainHeader struct {
	JudgmentOrigin string                 `json:"judgment_origin,omitempty"`
	LocalKey       string                 `json:"local_key"`
	SourceID       string                 `json:"source_id"`
	Name           string                 `json:"name"`
	Assessment     V4ReadAssessment       `json:"assessment"`
	EmptyState     *V4ReadChainEmptyState `json:"empty_state"`
}
type V4DetailProjection struct {
	JudgmentOrigin   string              `json:"judgment_origin,omitempty"`
	ReasoningSources *V5ReasoningSources `json:"reasoning_sources,omitempty"`
	VariableSignals  *[]V5ReadSignal     `json:"variable_signals,omitempty"`
	Companies        *[]V4ReadMacro      `json:"companies,omitempty"`
	Summary          V4SummaryProjection `json:"summary"`
	MacroImpacts     []V4ReadMacro       `json:"macro_impacts"`
	IndustryChains   []V4ChainHeader     `json:"industry_chains"`
}
type V4HomeProjection struct {
	ReportType     V4ReadCodedLabel               `json:"report_type"`
	SchemaVersion  string                         `json:"schema_version"`
	GeneratedAt    string                         `json:"generated_at"`
	Timezone       string                         `json:"timezone"`
	AnalysisWindow V4ReadReportAnalysisWindow     `json:"analysis_window"`
	Observations   []V4ReadReportObservationsItem `json:"observations"`
	Limitations    []string                       `json:"limitations"`
}

func (v AnalysisUnitSummary) MarshalJSON() ([]byte, error) {
	if v.Company != nil {
		return json.Marshal(v.Company)
	}
	if v.V4 != nil {
		return json.Marshal(v.V4)
	}
	type plain AnalysisUnitSummary
	return json.Marshal(plain(v))
}
func (v *AnalysisUnitSummary) UnmarshalJSON(b []byte) error {
	var companyProbe struct {
		Company json.RawMessage `json:"company"`
	}
	if err := json.Unmarshal(b, &companyProbe); err != nil {
		return err
	}
	if companyProbe.Company != nil {
		var c V5CompanyProjection
		if err := json.Unmarshal(b, &c); err != nil {
			return err
		}
		*v = AnalysisUnitSummary{Company: &c}
		return nil
	}
	var probe struct {
		SchemaVersion string `json:"schema_version"`
		Summary       struct {
			SchemaVersion string `json:"schema_version"`
		} `json:"summary"`
		Assessment json.RawMessage `json:"assessment"`
	}
	if err := json.Unmarshal(b, &probe); err != nil {
		return err
	}
	if probe.SchemaVersion == NormalizedSchemaVersion || probe.SchemaVersion == SignalSchemaVersion {
		var n V4SummaryProjection
		if err := json.Unmarshal(b, &n); err != nil {
			return err
		}
		*v = AnalysisUnitSummary{V4: &n}
		return nil
	}
	type plain AnalysisUnitSummary
	var n plain
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*v = AnalysisUnitSummary(n)
	return nil
}
func (v AnalysisUnitDetail) MarshalJSON() ([]byte, error) {
	if v.Company != nil {
		return json.Marshal(v.Company)
	}
	if v.V4 != nil {
		return json.Marshal(v.V4)
	}
	type plain AnalysisUnitDetail
	return json.Marshal(plain(v))
}
func (v *AnalysisUnitDetail) UnmarshalJSON(b []byte) error {
	var companyProbe struct {
		Company json.RawMessage `json:"company"`
	}
	if err := json.Unmarshal(b, &companyProbe); err != nil {
		return err
	}
	if companyProbe.Company != nil {
		var c V5CompanyProjection
		if err := json.Unmarshal(b, &c); err != nil {
			return err
		}
		*v = AnalysisUnitDetail{Company: &c}
		return nil
	}
	var probe struct {
		SchemaVersion string `json:"schema_version"`
		Summary       struct {
			SchemaVersion string `json:"schema_version"`
		} `json:"summary"`
		Assessment json.RawMessage `json:"assessment"`
	}
	if err := json.Unmarshal(b, &probe); err != nil {
		return err
	}
	if probe.Summary.SchemaVersion == NormalizedSchemaVersion || probe.Summary.SchemaVersion == SignalSchemaVersion {
		var n V4DetailProjection
		if err := json.Unmarshal(b, &n); err != nil {
			return err
		}
		*v = AnalysisUnitDetail{V4: &n}
		return nil
	}
	type plain AnalysisUnitDetail
	var n plain
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*v = AnalysisUnitDetail(n)
	return nil
}
func (v ChainAnalysisDetail) MarshalJSON() ([]byte, error) {
	if v.V4 != nil {
		return json.Marshal(v.V4)
	}
	type plain ChainAnalysisDetail
	return json.Marshal(plain(v))
}
func (v *ChainAnalysisDetail) UnmarshalJSON(b []byte) error {
	var probe struct {
		SchemaVersion string `json:"schema_version"`
		Summary       struct {
			SchemaVersion string `json:"schema_version"`
		} `json:"summary"`
		Assessment json.RawMessage `json:"assessment"`
	}
	if err := json.Unmarshal(b, &probe); err != nil {
		return err
	}
	if probe.Assessment != nil {
		var n V4ReadChain
		if err := json.Unmarshal(b, &n); err != nil {
			return err
		}
		*v = ChainAnalysisDetail{V4: &n}
		return nil
	}
	type plain ChainAnalysisDetail
	var n plain
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*v = ChainAnalysisDetail(n)
	return nil
}
func (v Home) MarshalJSON() ([]byte, error) {
	if v.V4 != nil {
		return json.Marshal(v.V4)
	}
	type plain Home
	return json.Marshal(plain(v))
}
func (v *Home) UnmarshalJSON(b []byte) error {
	var probe struct {
		SchemaVersion string `json:"schema_version"`
		Summary       struct {
			SchemaVersion string `json:"schema_version"`
		} `json:"summary"`
		Assessment json.RawMessage `json:"assessment"`
	}
	if err := json.Unmarshal(b, &probe); err != nil {
		return err
	}
	if probe.SchemaVersion == NormalizedSchemaVersion || probe.SchemaVersion == SignalSchemaVersion {
		var n V4HomeProjection
		if err := json.Unmarshal(b, &n); err != nil {
			return err
		}
		*v = Home{V4: &n}
		return nil
	}
	type plain Home
	var n plain
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*v = Home(n)
	return nil
}

// V5 signal provenance is publisher-authored and never joined to live Event or graph facts.
type V5ReasoningSources struct {
	SignalIDs    []string        `json:"signal_ids"`
	EventIDs     []string        `json:"event_ids"`
	UpstreamRefs []V5UpstreamRef `json:"upstream_refs"`
}
type V5UpstreamRef struct {
	EntityID  string  `json:"entity_id"`
	LocalKey  string  `json:"local_key"`
	Mechanism *string `json:"mechanism,omitempty"`
	Condition *string `json:"condition,omitempty"`
}
type V5Signal struct {
	VariableID      string   `json:"variable_id"`
	VariableName    string   `json:"variable_name"`
	SignalID        string   `json:"signal_id"`
	Signal          string   `json:"signal"`
	SourceDirection string   `json:"source_direction"`
	Adoption        string   `json:"adoption"`
	Qualification   string   `json:"qualification"`
	EventIDs        []string `json:"event_ids"`
	EvidenceIDs     []string `json:"evidence_ids"`
}
type V5ReadSignal struct {
	VariableID         string   `json:"variable_id"`
	VariableName       string   `json:"variable_name"`
	SignalID           string   `json:"signal_id"`
	Signal             string   `json:"signal"`
	SourceDirection    string   `json:"source_direction"`
	Adoption           string   `json:"adoption"`
	Qualification      string   `json:"qualification"`
	EventIDs           []string `json:"event_ids"`
	EvidenceScopeToken *string  `json:"evidence_scope_token"`
	EvidenceCount      int      `json:"evidence_count"`
}
type V5CompanyProjection struct {
	SchemaVersion string      `json:"schema_version"`
	Company       V4ReadMacro `json:"company"`
}
