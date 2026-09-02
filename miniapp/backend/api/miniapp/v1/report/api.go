package report

import "context"

const (
	OperationGetHome       = "miniapp.v1.getReportHome"
	OperationGetLayer      = "miniapp.v1.getReportLayer"
	OperationGetChain      = "miniapp.v1.getReportIndustryChain"
	OperationListEvidences = "miniapp.v1.listReportEvidences"
)

type Service interface {
	GetHome(context.Context, *HomeRequest) (*HomeResponse, error)
	GetLayer(context.Context, *LayerRequest) (*LayerDetail, error)
	GetIndustryChain(context.Context, *IndustryChainRequest) (*IndustryChainDetail, error)
	ListEvidences(context.Context, *EvidenceRequest) (*EvidenceCollection, error)
}

type HomeRequest struct{}

type LayerRequest struct {
	ReportID string
	LayerKey string
}

type IndustryChainRequest struct {
	ReportID string
	ChainKey string
}

type EvidenceRequest struct {
	ReportID        string
	ScopeType       string
	ScopeKey        string
	HasUnknownQuery bool
}

type Selection struct {
	Mode     string `json:"mode"`
	Date     string `json:"date"`
	Timezone string `json:"timezone"`
}

type Summary struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	GeneratedAt string `json:"generated_at"`
	PublishedAt string `json:"published_at"`
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

type Reference struct {
	Type string `json:"type"`
	Key  string `json:"key"`
}

type Scope struct {
	Type string `json:"type"`
	Key  string `json:"key"`
}

type CardImpactItem struct {
	Ref         Reference  `json:"ref"`
	Name        string     `json:"name"`
	Result      Result     `json:"result"`
	Confidence  Confidence `json:"confidence"`
	TimeWindow  string     `json:"time_window"`
	HasEvidence bool       `json:"has_evidence"`
}

type Card struct {
	Key          string           `json:"key"`
	Kind         string           `json:"kind"`
	DisplayOrder int              `json:"display_order"`
	DetailRef    Reference        `json:"detail_ref"`
	Title        string           `json:"title"`
	Subtitle     string           `json:"subtitle"`
	Conclusion   string           `json:"conclusion"`
	Result       Result           `json:"result"`
	Confidence   Confidence       `json:"confidence"`
	TimeWindow   string           `json:"time_window"`
	ImpactItems  []CardImpactItem `json:"impact_items"`
	HasEvidence  bool             `json:"has_evidence"`
}

type HomeReport struct {
	Report             Summary `json:"report"`
	IndustryChainCount int     `json:"industry_chain_count"`
	Cards              []Card  `json:"cards"`
}

type HomeResponse struct {
	Selection Selection    `json:"selection"`
	Reports   []HomeReport `json:"reports"`
}

type Anchor struct {
	Key          string     `json:"key"`
	DisplayOrder int        `json:"display_order"`
	Name         string     `json:"name"`
	CurrentState string     `json:"current_state"`
	Result       Result     `json:"result"`
	Nature       Nature     `json:"nature"`
	Reasoning    string     `json:"reasoning"`
	TimeWindow   string     `json:"time_window"`
	Confidence   Confidence `json:"confidence"`
	Scope        Scope      `json:"scope"`
	HasEvidence  bool       `json:"has_evidence"`
}

type ReasoningStep struct {
	Key          string     `json:"key"`
	DisplayOrder int        `json:"display_order"`
	Input        string     `json:"input"`
	Mechanism    string     `json:"mechanism"`
	Output       string     `json:"output"`
	Type         string     `json:"type"`
	Confidence   Confidence `json:"confidence"`
	Scope        Scope      `json:"scope"`
	HasEvidence  bool       `json:"has_evidence"`
}

type TransmissionTarget struct {
	Ref    *Reference `json:"ref"`
	Label  string     `json:"label"`
	Result Result     `json:"result"`
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
	Scope            Scope                `json:"scope"`
	HasEvidence      bool                 `json:"has_evidence"`
}

type CandidateMechanism struct {
	Key          string     `json:"key"`
	DisplayOrder int        `json:"display_order"`
	Mechanism    string     `json:"mechanism"`
	EvidenceGap  *string    `json:"evidence_gap"`
	Confidence   Confidence `json:"confidence"`
	Scope        Scope      `json:"scope"`
	HasEvidence  bool       `json:"has_evidence"`
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
	Scope                Scope                `json:"scope"`
	HasEvidence          bool                 `json:"has_evidence"`
}

type IndustryChainSummary struct {
	Key          string     `json:"key"`
	DisplayOrder int        `json:"display_order"`
	Name         string     `json:"name"`
	Conclusion   string     `json:"conclusion"`
	Status       string     `json:"status"`
	Result       Result     `json:"result"`
	Confidence   Confidence `json:"confidence"`
	TimeWindow   string     `json:"time_window"`
	Scope        Scope      `json:"scope"`
	HasEvidence  bool       `json:"has_evidence"`
}

type LayerDetail struct {
	Report                Summary                `json:"report"`
	Layer                 Layer                  `json:"layer"`
	RelatedIndustryChains []IndustryChainSummary `json:"related_industry_chains"`
}

type IndustryChainNode struct {
	Key          string     `json:"key"`
	DisplayOrder int        `json:"display_order"`
	Name         string     `json:"name"`
	Impact       string     `json:"impact"`
	Result       Result     `json:"result"`
	Nature       Nature     `json:"nature"`
	Reasoning    string     `json:"reasoning"`
	TimeWindow   string     `json:"time_window"`
	Confidence   Confidence `json:"confidence"`
	Scope        Scope      `json:"scope"`
	HasEvidence  bool       `json:"has_evidence"`
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
	Nodes                     []IndustryChainNode `json:"nodes"`
	Edges                     []IndustryChainEdge `json:"edges"`
	Uncertainty               ChainUncertainty    `json:"uncertainty"`
	Scope                     Scope               `json:"scope"`
	HasEvidence               bool                `json:"has_evidence"`
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
	ReportID string         `json:"report_id"`
	Scope    Scope          `json:"scope"`
	Items    []EvidenceItem `json:"items"`
}
