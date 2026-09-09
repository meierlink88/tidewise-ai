package report

import "context"

const (
	OperationGetHome       = "miniapp.v1.getReportHome"
	OperationListEvidences = "miniapp.v1.listReportEvidences"
)

type Service interface {
	ListAnalyses(context.Context, *AnalysisQuery) (*AnalysisPage, error)
	GetAnalysis(context.Context, *AnalysisQuery) (*NormalizedDetailProjection, error)
	GetAnalysisChain(context.Context, *AnalysisQuery) (*NormalizedChain, error)
	GetHome(context.Context, *HomeRequest) (*HomeResponse, error)
	ListEvidences(context.Context, *EvidenceRequest) (*EvidenceCollection, error)
}

type HomeRequest struct{}
type EvidenceRequest struct {
	ReportID, ScopeToken string
	HasUnknownQuery      bool
}

type Selection struct {
	Mode     string `json:"mode"`
	Date     string `json:"date"`
	Timezone string `json:"timezone"`
}

type Summary struct {
	SchemaVersion      string `json:"schema_version,omitempty"`
	ID                 string `json:"id"`
	GeneratedAt        string `json:"generated_at"`
	PublishedAt        string `json:"published_at"`
	IndustryChainCount int    `json:"industry_chain_count"`
}

type CodedLabel struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

type Confidence struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

type TimeWindow struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

type Reference struct {
	Type     string `json:"type"`
	LocalKey string `json:"local_key"`
}

type CardImpactItem struct {
	Ref                Reference  `json:"ref"`
	Name               string     `json:"name"`
	Result             CodedLabel `json:"result"`
	ConclusionBasis    CodedLabel `json:"conclusion_basis"`
	ValidationStatus   CodedLabel `json:"validation_status"`
	Confidence         Confidence `json:"confidence"`
	TimeWindow         TimeWindow `json:"time_window"`
	EvidenceScopeToken *string    `json:"evidence_scope_token"`
}

type Card struct {
	LocalKey           string           `json:"local_key"`
	Kind               string           `json:"kind"`
	DetailRef          Reference        `json:"detail_ref"`
	Title              string           `json:"title"`
	Subtitle           string           `json:"subtitle"`
	Conclusion         string           `json:"conclusion"`
	Result             CodedLabel       `json:"result"`
	Confidence         Confidence       `json:"confidence"`
	TimeWindow         TimeWindow       `json:"time_window"`
	ImpactItems        []CardImpactItem `json:"impact_items"`
	EvidenceScopeToken *string          `json:"evidence_scope_token"`
}

type CardCollection struct {
	Items      []Card  `json:"items"`
	NextCursor *string `json:"next_cursor"`
}

type HomeReport struct {
	AnalysisGroups []AnalysisGroup `json:"analysis_groups,omitempty"`
	Report         Summary         `json:"report"`
	Cards          []Card          `json:"cards"`
	NextCursor     *string         `json:"next_cursor"`
}

type HomeResponse struct {
	Selection Selection    `json:"selection"`
	Reports   []HomeReport `json:"reports"`
}

type Anchor struct {
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

type ReasoningStep struct {
	LocalKey           string     `json:"local_key"`
	Input              string     `json:"input"`
	Mechanism          string     `json:"mechanism"`
	Output             string     `json:"output"`
	Confidence         Confidence `json:"confidence"`
	EvidenceScopeToken *string    `json:"evidence_scope_token"`
}

type TransmissionTarget struct {
	Ref    Reference  `json:"ref"`
	Name   string     `json:"name"`
	Result CodedLabel `json:"result"`
}

type TransmissionPath struct {
	LocalKey         string               `json:"local_key"`
	SourceConclusion string               `json:"source_conclusion"`
	Targets          []TransmissionTarget `json:"targets"`
	Logic            string               `json:"logic"`
	Kind             CodedLabel           `json:"kind"`
	Confidence       Confidence           `json:"confidence"`
	Status           CodedLabel           `json:"status"`
}

type LayerUncertainty struct {
	Counterevidence   *string `json:"counterevidence"`
	EvidenceGap       *string `json:"evidence_gap"`
	Boundary          *string `json:"boundary"`
	ReversalCondition *string `json:"reversal_condition"`
}

type Layer struct {
	Key                string             `json:"key"`
	Title              string             `json:"title"`
	Conclusion         string             `json:"conclusion"`
	Result             CodedLabel         `json:"result"`
	Confidence         Confidence         `json:"confidence"`
	TimeWindow         TimeWindow         `json:"time_window"`
	Anchors            []Anchor           `json:"anchors"`
	ReasoningSteps     []ReasoningStep    `json:"reasoning_steps"`
	Transmissions      []TransmissionPath `json:"transmissions"`
	Uncertainty        LayerUncertainty   `json:"uncertainty"`
	EvidenceScopeToken *string            `json:"evidence_scope_token"`
}

type RelatedIndustryChain struct {
	LocalKey string     `json:"local_key"`
	Name     string     `json:"name"`
	Result   CodedLabel `json:"result"`
}

type LayerDetail struct {
	Report                Summary                `json:"report"`
	Layer                 Layer                  `json:"layer"`
	RelatedIndustryChains []RelatedIndustryChain `json:"related_industry_chains"`
}

type IndustryChainNode struct {
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

type IndustryChainEdge struct {
	FromNodeLocalKey string `json:"from_node_local_key"`
	ToNodeLocalKey   string `json:"to_node_local_key"`
	RelationLabel    string `json:"relation_label"`
}

type IndustryChainTopologyNode struct {
	LocalKey string `json:"local_key"`
	Name     string `json:"name"`
}

type IndustryChain struct {
	LocalKey                  string                      `json:"local_key"`
	Name                      string                      `json:"name"`
	Conclusion                string                      `json:"conclusion"`
	Result                    CodedLabel                  `json:"result"`
	Confidence                Confidence                  `json:"confidence"`
	TimeWindow                TimeWindow                  `json:"time_window"`
	PathSummary               *string                     `json:"path_summary"`
	AcceptedHypothesisSummary *string                     `json:"accepted_hypothesis_summary"`
	TopologyNodes             []IndustryChainTopologyNode `json:"topology_nodes"`
	Nodes                     []IndustryChainNode         `json:"nodes"`
	Edges                     []IndustryChainEdge         `json:"edges"`
	CounterevidenceAndGap     *string                     `json:"counterevidence_and_gap"`
	StopCondition             *string                     `json:"stop_condition"`
	EvidenceScopeToken        *string                     `json:"evidence_scope_token"`
}

type IndustryChainDetail struct {
	Report        Summary       `json:"report"`
	IndustryChain IndustryChain `json:"industry_chain"`
}

type EvidenceItem struct {
	SemanticTags []EvidenceTag `json:"semantic_tags,omitempty"`
	PublishedAt  *string       `json:"published_at"`
	Summary      string        `json:"summary"`
	Keywords     []string      `json:"keywords"`
}

type EvidenceCollection struct {
	ReportID   string         `json:"report_id"`
	ScopeToken string         `json:"scope_token"`
	Items      []EvidenceItem `json:"items"`
}

type NormalizedClaim struct {
	Text               string  `json:"text"`
	Basis              string  `json:"basis"`
	EvidenceScopeToken *string `json:"evidence_scope_token"`
	EvidenceCount      int     `json:"evidence_count"`
}
type NormalizedObjections struct {
	Summary               string            `json:"summary"`
	Counterevidence       []NormalizedClaim `json:"counterevidence"`
	Buffers               []NormalizedClaim `json:"buffers"`
	CounterevidenceStatus string            `json:"counterevidence_status"`
	EvidenceGaps          []string          `json:"evidence_gaps"`
	ScopeLimits           []string          `json:"scope_limits"`
}
type NormalizedWindow struct {
	Kind        string  `json:"kind"`
	Description string  `json:"description"`
	StartAt     *string `json:"start_at"`
	EndAt       *string `json:"end_at"`
}
type NormalizedAssessment struct {
	Conclusion         string           `json:"conclusion"`
	Direction          string           `json:"direction"`
	ConclusionBasis    string           `json:"conclusion_basis"`
	ValidationStatus   string           `json:"validation_status"`
	Confidence         *string          `json:"confidence"`
	ForecastWindow     NormalizedWindow `json:"forecast_window"`
	Scope              string           `json:"scope"`
	Conditions         []string         `json:"conditions"`
	FollowUp           []string         `json:"follow_up"`
	TransmissionLogic  string           `json:"transmission_logic"`
	EvidenceScopeToken *string          `json:"evidence_scope_token"`
	EvidenceCount      int              `json:"evidence_count"`
}
type NormalizedNode struct {
	JudgmentOrigin   string                      `json:"judgment_origin,omitempty"`
	ReasoningSources *NormalizedReasoningSources `json:"reasoning_sources,omitempty"`
	VariableSignals  *[]NormalizedSignal         `json:"variable_signals,omitempty"`
	LocalKey         string                      `json:"local_key"`
	SourceID         string                      `json:"source_id"`
	NodeLocalKey     string                      `json:"node_local_key"`
	Name             string                      `json:"name"`
	Assessment       NormalizedAssessment        `json:"assessment"`
	Objections       NormalizedObjections        `json:"objections"`
}
type NormalizedGraph struct {
	Scope string                     `json:"scope,omitempty"`
	Nodes []NormalizedGraphNodesItem `json:"nodes"`
	Edges []NormalizedGraphEdgesItem `json:"edges"`
}
type NormalizedChain struct {
	JudgmentOrigin   string                          `json:"judgment_origin,omitempty"`
	ReasoningSources *NormalizedReasoningSources     `json:"reasoning_sources,omitempty"`
	VariableSignals  *[]NormalizedSignal             `json:"variable_signals,omitempty"`
	LocalKey         string                          `json:"local_key"`
	SourceID         string                          `json:"source_id"`
	Name             string                          `json:"name"`
	Assessment       NormalizedAssessment            `json:"assessment"`
	ReasoningSummary NormalizedChainReasoningSummary `json:"reasoning_summary"`
	Graph            NormalizedGraph                 `json:"graph"`
	AffectedNodes    []NormalizedNode                `json:"affected_nodes"`
	EmptyState       *NormalizedChainEmptyState      `json:"empty_state"`
}
type NormalizedMacro struct {
	JudgmentOrigin   string                      `json:"judgment_origin,omitempty"`
	ReasoningSources *NormalizedReasoningSources `json:"reasoning_sources,omitempty"`
	VariableSignals  *[]NormalizedSignal         `json:"variable_signals,omitempty"`
	LocalKey         string                      `json:"local_key"`
	SourceID         string                      `json:"source_id"`
	Name             string                      `json:"name"`
	Assessment       NormalizedAssessment        `json:"assessment"`
	Objections       NormalizedObjections        `json:"objections"`
}
type NormalizedAnchorRef struct {
	TargetType    string  `json:"target_type"`
	LocalKey      string  `json:"local_key"`
	ChainLocalKey *string `json:"chain_local_key"`
}
type NormalizedGraphNodesItem struct {
	LocalKey string `json:"local_key"`
	SourceID string `json:"source_id"`
	Name     string `json:"name"`
}
type NormalizedGraphEdgesItem struct {
	FromNodeLocalKey string `json:"from_node_local_key"`
	ToNodeLocalKey   string `json:"to_node_local_key"`
	RelationLabel    string `json:"relation_label"`
}
type NormalizedChainReasoningSummary struct {
	Logic      string               `json:"logic"`
	Support    NormalizedClaim      `json:"support"`
	Objections NormalizedObjections `json:"objections"`
}
type NormalizedChainEmptyState struct {
	Code     string   `json:"code"`
	Reason   string   `json:"reason"`
	FollowUp []string `json:"follow_up"`
}
type NormalizedUnitSummary struct {
	Conclusion         string                                `json:"conclusion"`
	TransmissionLogic  string                                `json:"transmission_logic"`
	ImpactAssessment   NormalizedUnitSummaryImpactAssessment `json:"impact_assessment"`
	AffectedRefs       []NormalizedAnchorRef                 `json:"affected_refs"`
	EvidenceScopeToken *string                               `json:"evidence_scope_token"`
	EvidenceCount      int                                   `json:"evidence_count"`
}
type NormalizedUnitSummaryImpactAssessment struct {
	Level              string  `json:"level"`
	Rationale          string  `json:"rationale"`
	EvidenceScopeToken *string `json:"evidence_scope_token"`
	EvidenceCount      int     `json:"evidence_count"`
}

type NormalizedResolvedAnchor struct {
	JudgmentOrigin string               `json:"judgment_origin,omitempty"`
	Reference      NormalizedAnchorRef  `json:"reference"`
	SourceID       string               `json:"source_id"`
	Name           string               `json:"name"`
	Assessment     NormalizedAssessment `json:"assessment"`
}
type NormalizedSummaryProjection struct {
	JudgmentOrigin  string                     `json:"judgment_origin,omitempty"`
	SchemaVersion   string                     `json:"schema_version"`
	LocalKey        string                     `json:"local_key"`
	SourceID        string                     `json:"source_id"`
	Title           string                     `json:"title"`
	Summary         NormalizedUnitSummary      `json:"summary"`
	AffectedAnchors []NormalizedResolvedAnchor `json:"affected_anchors"`
	ChainCount      int                        `json:"chain_count"`
}
type NormalizedChainHeader struct {
	JudgmentOrigin string                     `json:"judgment_origin,omitempty"`
	LocalKey       string                     `json:"local_key"`
	SourceID       string                     `json:"source_id"`
	Name           string                     `json:"name"`
	Assessment     NormalizedAssessment       `json:"assessment"`
	EmptyState     *NormalizedChainEmptyState `json:"empty_state"`
}
type NormalizedDetailProjection struct {
	JudgmentOrigin   string                      `json:"judgment_origin,omitempty"`
	ReasoningSources *NormalizedReasoningSources `json:"reasoning_sources,omitempty"`
	VariableSignals  *[]NormalizedSignal         `json:"variable_signals,omitempty"`
	Companies        *[]NormalizedMacro          `json:"companies,omitempty"`
	Summary          NormalizedSummaryProjection `json:"summary"`
	MacroImpacts     []NormalizedMacro           `json:"macro_impacts"`
	IndustryChains   []NormalizedChainHeader     `json:"industry_chains"`
}

type AnalysisPage struct {
	Items      []NormalizedSummaryProjection `json:"items"`
	NextCursor *string                       `json:"next_cursor"`
}
type AnalysisGroup struct {
	Kind       string                        `json:"kind"`
	Items      []NormalizedSummaryProjection `json:"items"`
	NextCursor *string                       `json:"next_cursor"`
}
type AnalysisQuery struct {
	ReportID, Kind, Key, ChainKey, Cursor string
	Limit                                 int
}

type NormalizedReasoningSources struct {
	SignalIDs    []string                `json:"signal_ids"`
	EventIDs     []string                `json:"event_ids"`
	UpstreamRefs []NormalizedUpstreamRef `json:"upstream_refs"`
}
type NormalizedUpstreamRef struct {
	EntityID  string  `json:"entity_id"`
	LocalKey  string  `json:"local_key"`
	Mechanism *string `json:"mechanism,omitempty"`
	Condition *string `json:"condition,omitempty"`
}
type NormalizedSignal struct {
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

type EvidenceTag struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}
