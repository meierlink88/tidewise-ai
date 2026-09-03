package report

import "context"

const (
	OperationGetHome       = "miniapp.v1.getReportHome"
	OperationListChains    = "miniapp.v1.listReportIndustryChains"
	OperationGetLayer      = "miniapp.v1.getReportLayer"
	OperationGetChain      = "miniapp.v1.getReportIndustryChain"
	OperationListEvidences = "miniapp.v1.listReportEvidences"
)

type Service interface {
	GetHome(context.Context, *HomeRequest) (*HomeResponse, error)
	ListIndustryChains(context.Context, *IndustryChainListRequest) (*CardCollection, error)
	GetLayer(context.Context, *LayerRequest) (*LayerDetail, error)
	GetIndustryChain(context.Context, *IndustryChainRequest) (*IndustryChainDetail, error)
	ListEvidences(context.Context, *EvidenceRequest) (*EvidenceCollection, error)
}

type HomeRequest struct{}
type LayerRequest struct{ ReportID, LayerKey string }
type IndustryChainRequest struct{ ReportID, ChainKey string }
type IndustryChainListRequest struct {
	ReportID, Limit, Cursor string
	HasUnknownQuery         bool
}
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
	Report     Summary `json:"report"`
	Cards      []Card  `json:"cards"`
	NextCursor *string `json:"next_cursor"`
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
	PublishedAt *string  `json:"published_at"`
	Summary     string   `json:"summary"`
	Keywords    []string `json:"keywords"`
}

type EvidenceCollection struct {
	ReportID   string         `json:"report_id"`
	ScopeToken string         `json:"scope_token"`
	Items      []EvidenceItem `json:"items"`
}
