package report

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

type ScopeType string

const (
	ScopeSectionSummary       ScopeType = "section_summary"
	ScopeAnchor               ScopeType = "anchor"
	ScopeReasoningStep        ScopeType = "reasoning_step"
	ScopeIndustryChainSummary ScopeType = "industry_chain_summary"
	ScopeIndustryChainNode    ScopeType = "industry_chain_node"
)

const (
	ResultWarming            = "warming"
	ResultCooling            = "cooling"
	ResultDiverging          = "diverging"
	ResultPending            = "pending"
	BasisDirectEvidence      = "direct_evidence"
	BasisReasoningHypothesis = "reasoning_hypothesis"
	BasisNoDirectional       = "no_directional_conclusion"
	ValidationConfirmed      = "confirmed"
	ValidationPending        = "pending_validation"
	TransmissionCrossLayer   = "cross_layer_reasoning"
	TransmissionSameSource   = "same_source_signal"
)

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
	ReportType     CodedLabel      `json:"report_type"`
	GeneratedAt    time.Time       `json:"generated_at"`
	Timezone       string          `json:"timezone"`
	Geopolitics    *Layer          `json:"geopolitics,omitempty"`
	Macroeconomics *Layer          `json:"macroeconomics,omitempty"`
	IndustryChains []IndustryChain `json:"industry_chains"`
}

type Record struct {
	ID                string
	PublisherReportID string
	ContentHash       string
	Report            Report
	PublishedAt       time.Time
}

type EvidenceLink struct {
	ID         string
	ReportID   string
	EvidenceID string
	ScopeType  ScopeType
	ScopePath  string
	Position   int
}

type Evidence struct {
	PublishedAt *time.Time
	Summary     string
	Keywords    []string
}

type Summary struct {
	ID                 string
	PublisherReportID  string
	GeneratedAt        time.Time
	HasGeopolitics     bool
	HasMacroeconomics  bool
	IndustryChainCount int
	PublishedAt        time.Time
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
	Report         Summary
	Geopolitics    *LayerSnapshot
	Macroeconomics *LayerSnapshot
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
	Ordinal            int                          `json:"-"`
	LocalKey           string                       `json:"local_key"`
	Name               string                       `json:"name"`
	Conclusion         string                       `json:"conclusion"`
	Result             CodedLabel                   `json:"result"`
	Confidence         Confidence                   `json:"confidence"`
	TimeWindow         TimeWindow                   `json:"time_window"`
	ImpactItems        []IndustryChainImpactSummary `json:"impact_items"`
	EvidenceScopeToken *string                      `json:"evidence_scope_token"`
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

type IndustryChainListRequest struct {
	ReportID string
	Limit    int
	Cursor   string
}

type IndustryChainListFilter struct {
	ReportID     string
	AfterOrdinal int
	Limit        int
}

type IndustryChainStorePage struct {
	Items   []IndustryChainSummary
	HasMore bool
}

type IndustryChainPage struct {
	Items      []IndustryChainSummary
	NextCursor *string
}

type ListRequest struct {
	PublishedFrom, PublishedTo *time.Time
	Limit                      int
	Cursor                     string
}

type ListFilter struct {
	PublishedFrom, PublishedTo, CursorPublishedAt *time.Time
	CursorID                                      string
	Limit                                         int
}

type StorePage struct {
	Items   []Summary
	HasMore bool
}

type Page struct {
	Items      []Summary
	NextCursor *string
}

type PublicationResult struct {
	Record   Record
	Replayed bool
}

type Store interface {
	PublicationStore
	ListReports(context.Context, ListFilter) (StorePage, error)
	GetReport(context.Context, string) (Record, error)
	GetHome(context.Context, string) (Home, error)
	GetLayer(context.Context, string, string) (Summary, LayerProjection, error)
	ListIndustryChains(context.Context, IndustryChainListFilter) (IndustryChainStorePage, error)
	GetIndustryChain(context.Context, string, string) (Summary, IndustryChainProjection, error)
	ListEvidence(context.Context, string, string) ([]Evidence, error)
}

var (
	ErrPublicationConflict   = errors.New("Report publisher identity conflicts with another payload")
	ErrReportNotFound        = errors.New("Report was not found")
	ErrLayerNotFound         = errors.New("Report layer was not found")
	ErrChainNotFound         = errors.New("Report industry chain was not found")
	ErrEvidenceScopeNotFound = errors.New("Report Evidence scope was not found")
)

type ValidationError struct {
	Path    string
	Message string
}

func (e *ValidationError) Error() string { return e.Path + ": " + e.Message }

type ReferenceError struct {
	Path      string
	Reference string
	Message   string
}

func (e *ReferenceError) Error() string {
	return fmt.Sprintf("%s: %s (%s)", e.Path, e.Message, e.Reference)
}

func invalid(path, message string) error { return &ValidationError{Path: path, Message: message} }

type UseCase struct {
	store Store
	now   func() time.Time
}

func NewUseCase(store Store, now func() time.Time) (*UseCase, error) {
	if store == nil {
		return nil, errors.New("Report store is required")
	}
	if now == nil {
		return nil, errors.New("Report clock is required")
	}
	return &UseCase{store: store, now: now}, nil
}

func (s *UseCase) Publish(ctx context.Context, publisherReportID string, report Report) (PublicationResult, error) {
	if s == nil || s.store == nil {
		return PublicationResult{}, errors.New("Report store is required")
	}
	if err := requiredText("publisher_report_id", publisherReportID, 200); err != nil {
		return PublicationResult{}, err
	}
	if err := ValidateReport(report); err != nil {
		return PublicationResult{}, err
	}
	payloadHash, err := ContentHash(report)
	if err != nil {
		return PublicationResult{}, fmt.Errorf("canonicalize Report publication: %w", err)
	}
	var result PublicationResult
	err = s.store.InPublicationTransaction(ctx, func(tx PublicationTransaction) error {
		if err := tx.Lock(ctx, publisherReportID); err != nil {
			return err
		}
		existing, err := tx.ReportByPublisherID(ctx, publisherReportID)
		if err != nil {
			return err
		}
		if existing != nil {
			if existing.ContentHash != payloadHash {
				return ErrPublicationConflict
			}
			result = PublicationResult{Record: *existing, Replayed: true}
			return nil
		}
		reportID, err := coreid.New(coreid.Report)
		if err != nil {
			return fmt.Errorf("generate Report ID: %w", err)
		}
		links, err := buildEvidenceLinks(reportID, report)
		if err != nil {
			return err
		}
		ids := uniqueEvidenceIDs(links)
		existingIDs, err := tx.ExistingEvidenceIDs(ctx, ids)
		if err != nil {
			return err
		}
		if missing := firstMissingEvidence(ids, existingIDs); missing != "" {
			return &ReferenceError{Path: "report.evidence_refs", Reference: missing, Message: "does not identify an existing Atomic Evidence"}
		}
		record := Record{ID: reportID, PublisherReportID: publisherReportID, ContentHash: payloadHash, Report: report, PublishedAt: s.now().UTC()}
		if err := tx.InsertReport(ctx, record); err != nil {
			return err
		}
		if err := tx.InsertEvidenceLinks(ctx, links); err != nil {
			return err
		}
		result = PublicationResult{Record: record}
		return nil
	})
	return result, err
}

func ContentHash(report Report) (string, error) { return canonicalPayloadHash(report) }

func (s *UseCase) List(ctx context.Context, request ListRequest) (Page, error) {
	if request.Limit == 0 {
		request.Limit = DefaultLimit
	}
	if request.Limit < 1 || request.Limit > MaxLimit {
		return Page{}, invalid("limit", fmt.Sprintf("must be between 1 and %d", MaxLimit))
	}
	if request.PublishedFrom != nil && request.PublishedTo != nil && !request.PublishedFrom.Before(*request.PublishedTo) {
		return Page{}, invalid("published_from", "must be before published_to")
	}
	filter := ListFilter{PublishedFrom: cloneTime(request.PublishedFrom), PublishedTo: cloneTime(request.PublishedTo), Limit: request.Limit}
	if strings.TrimSpace(request.Cursor) != "" {
		cursor, err := decodeReportCursor(request.Cursor)
		if err != nil || cursor.Version != 1 || !coreid.Is(cursor.ID, coreid.Report) || !sameOptionalTime(cursor.PublishedFrom, request.PublishedFrom) || !sameOptionalTime(cursor.PublishedTo, request.PublishedTo) {
			return Page{}, invalid("cursor", "is invalid for this Report query")
		}
		filter.CursorPublishedAt, filter.CursorID = cloneTime(&cursor.PublishedAt), cursor.ID
	}
	page, err := s.store.ListReports(ctx, filter)
	if err != nil {
		return Page{}, err
	}
	result := Page{Items: page.Items}
	if page.HasMore && len(page.Items) > 0 {
		last := page.Items[len(page.Items)-1]
		encoded, err := encodeCursor(reportCursor{Version: 1, PublishedFrom: cloneTime(request.PublishedFrom), PublishedTo: cloneTime(request.PublishedTo), PublishedAt: last.PublishedAt.UTC(), ID: last.ID})
		if err != nil {
			return Page{}, fmt.Errorf("encode Report cursor: %w", err)
		}
		result.NextCursor = &encoded
	}
	return result, nil
}

func (s *UseCase) GetHome(ctx context.Context, reportID string) (Home, error) {
	if err := validateReportID(reportID); err != nil {
		return Home{}, err
	}
	return s.store.GetHome(ctx, reportID)
}

func (s *UseCase) Get(ctx context.Context, reportID string) (Record, error) {
	if err := validateReportID(reportID); err != nil {
		return Record{}, err
	}
	return s.store.GetReport(ctx, reportID)
}

func (s *UseCase) GetLayer(ctx context.Context, reportID, layerKey string) (Summary, LayerProjection, error) {
	if err := validateReportID(reportID); err != nil {
		return Summary{}, LayerProjection{}, err
	}
	if layerKey != "geopolitics" && layerKey != "macroeconomics" {
		return Summary{}, LayerProjection{}, ErrLayerNotFound
	}
	return s.store.GetLayer(ctx, reportID, layerKey)
}

func (s *UseCase) GetIndustryChain(ctx context.Context, reportID, chainKey string) (Summary, IndustryChainProjection, error) {
	if err := validateReportID(reportID); err != nil {
		return Summary{}, IndustryChainProjection{}, err
	}
	if !localKeyPattern.MatchString(chainKey) {
		return Summary{}, IndustryChainProjection{}, ErrChainNotFound
	}
	return s.store.GetIndustryChain(ctx, reportID, chainKey)
}

func (s *UseCase) ListIndustryChains(ctx context.Context, request IndustryChainListRequest) (IndustryChainPage, error) {
	if err := validateReportID(request.ReportID); err != nil {
		return IndustryChainPage{}, err
	}
	if request.Limit == 0 {
		request.Limit = DefaultLimit
	}
	if request.Limit < 1 || request.Limit > MaxLimit {
		return IndustryChainPage{}, invalid("limit", fmt.Sprintf("must be between 1 and %d", MaxLimit))
	}
	filter := IndustryChainListFilter{ReportID: request.ReportID, Limit: request.Limit}
	if request.Cursor != "" {
		cursor, err := decodeIndustryChainCursor(request.Cursor)
		if err != nil || cursor.Version != 1 || cursor.ReportID != request.ReportID || cursor.Ordinal < 1 {
			return IndustryChainPage{}, invalid("cursor", "is invalid for this Report industry-chain query")
		}
		filter.AfterOrdinal = cursor.Ordinal
	}
	page, err := s.store.ListIndustryChains(ctx, filter)
	if err != nil {
		return IndustryChainPage{}, err
	}
	result := IndustryChainPage{Items: page.Items}
	if page.HasMore && len(page.Items) > 0 {
		last := page.Items[len(page.Items)-1]
		encoded, err := encodeCursor(industryChainCursor{Version: 1, ReportID: request.ReportID, Ordinal: last.Ordinal})
		if err != nil {
			return IndustryChainPage{}, fmt.Errorf("encode Report industry-chain cursor: %w", err)
		}
		result.NextCursor = &encoded
	}
	return result, nil
}

func (s *UseCase) ListEvidence(ctx context.Context, reportID, scopeToken string) ([]Evidence, error) {
	if err := validateReportID(reportID); err != nil {
		return nil, err
	}
	if !coreid.Is(scopeToken, coreid.ReportEvidenceLink) {
		return nil, invalid("scope_token", "must be an opaque Report Evidence scope token")
	}
	return s.store.ListEvidence(ctx, reportID, scopeToken)
}

func validateReportID(value string) error {
	if !coreid.Is(value, coreid.Report) {
		return invalid("report_id", "must be a Report ID")
	}
	return nil
}

type reportCursor struct {
	Version       int        `json:"v"`
	PublishedFrom *time.Time `json:"published_from"`
	PublishedTo   *time.Time `json:"published_to"`
	PublishedAt   time.Time  `json:"published_at"`
	ID            string     `json:"id"`
}

type industryChainCursor struct {
	Version  int    `json:"v"`
	ReportID string `json:"report_id"`
	Ordinal  int    `json:"ordinal"`
}

func encodeCursor(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeReportCursor(value string) (reportCursor, error) {
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return reportCursor{}, err
	}
	var cursor reportCursor
	err = json.Unmarshal(payload, &cursor)
	return cursor, err
}

func decodeIndustryChainCursor(value string) (industryChainCursor, error) {
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return industryChainCursor{}, err
	}
	var cursor industryChainCursor
	err = json.Unmarshal(payload, &cursor)
	return cursor, err
}

var localKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

var (
	resultLabels = map[string]string{
		ResultWarming: "升温", ResultCooling: "降温", ResultDiverging: "分化", ResultPending: "待验证",
	}
	confidenceLabels = map[string]string{"low": "低", "medium": "中", "high": "高"}
	timeWindowLabels = map[string]string{
		"short": "短期", "medium": "中期", "long": "长期", "short_medium": "短期–中期",
		"short_long": "短期–长期", "medium_long": "中期–长期",
		"short_medium_long": "短期–中期–长期", "follow_up": "后续周期",
	}
	basisLabels = map[string]string{
		BasisDirectEvidence: "直接证据", BasisReasoningHypothesis: "推理假设",
		BasisNoDirectional: "无方向性结论",
	}
	validationLabels   = map[string]string{ValidationConfirmed: "已确认", ValidationPending: "待验证"}
	evidenceRoleLabels = map[string]string{
		"direct_support": "直接依据", "reasoning_support": "推导依据", "summary_support": "核心依据",
	}
	transmissionKindLabels = map[string]string{
		TransmissionCrossLayer: "跨层推理", TransmissionSameSource: "同源信号",
	}
	transmissionStatusLabels = map[string]string{"established": "已形成传导"}
	targetTypeLabels         = map[string]string{
		"macro_anchor": "宏观经济锚点", "industry_chain": "产业链", "industry_chain_node": "产业链节点",
	}
)

func ValidateReport(report Report) error {
	if err := validateMappedLabel("report.report_type", report.ReportType, map[string]string{"investment_reasoning": "投研推理报告"}); err != nil {
		return err
	}
	if report.GeneratedAt.IsZero() {
		return invalid("report.generated_at", "must be a timestamp")
	}
	if report.Timezone != "Asia/Shanghai" {
		return invalid("report.timezone", "must be Asia/Shanghai")
	}
	if len(report.IndustryChains) == 0 {
		return invalid("report.industry_chains", "must contain at least one industry-chain analysis")
	}
	index := reportIndex{
		anchors: map[string]string{}, chains: map[string]struct{}{},
		nodes: map[string]struct{}{}, allKeys: map[string]struct{}{},
	}
	if report.Geopolitics != nil {
		if err := validateLayer("report.geopolitics", "geopolitics", *report.Geopolitics, true, &index); err != nil {
			return err
		}
	}
	if report.Macroeconomics != nil {
		if err := validateLayer("report.macroeconomics", "macroeconomics", *report.Macroeconomics, false, &index); err != nil {
			return err
		}
	}
	for position, chain := range report.IndustryChains {
		if err := validateIndustryChain(fmt.Sprintf("report.industry_chains[%d]", position), chain, &index); err != nil {
			return err
		}
	}
	return validateTransmissionTargets(report, index)
}

type reportIndex struct {
	anchors map[string]string
	chains  map[string]struct{}
	nodes   map[string]struct{}
	allKeys map[string]struct{}
}

func (i *reportIndex) add(path, key string) error {
	if !localKeyPattern.MatchString(key) {
		return invalid(path, "must be a Report-local key")
	}
	if _, exists := i.allKeys[key]; exists {
		return invalid(path, "duplicates a Report-local key")
	}
	i.allKeys[key] = struct{}{}
	return nil
}

func validateLayer(path, expectedKey string, layer Layer, geopolitics bool, index *reportIndex) error {
	if layer.LocalKey != expectedKey {
		return invalid(path+".local_key", "does not match its Report layer")
	}
	for field, value := range map[string]string{
		"title": layer.Title, "conclusion": layer.Conclusion,
	} {
		if err := requiredText(path+"."+field, value, 10_000); err != nil {
			return err
		}
	}
	if err := validateResult(path+".result", layer.Result); err != nil {
		return err
	}
	if err := validateTimeWindow(path+".time_window", layer.TimeWindow); err != nil {
		return err
	}
	if err := validateConfidence(path+".confidence", layer.Confidence); err != nil {
		return err
	}
	if layer.AffectedAnchors == nil || layer.ReasoningSteps == nil {
		return invalid(path, "affected_anchors and reasoning_steps must be arrays")
	}
	for position, anchor := range layer.AffectedAnchors {
		itemPath := fmt.Sprintf("%s.affected_anchors[%d]", path, position)
		if err := index.add(itemPath+".local_key", anchor.LocalKey); err != nil {
			return err
		}
		index.anchors[anchor.LocalKey] = expectedKey
		if err := validateAssessment(itemPath, anchor.Name, "current_state", anchor.CurrentState, anchor.Reasoning,
			anchor.Result, anchor.ConclusionBasis, anchor.ValidationStatus, anchor.TimeWindow,
			anchor.Confidence, anchor.EvidenceRefs); err != nil {
			return err
		}
	}
	for position, step := range layer.ReasoningSteps {
		itemPath := fmt.Sprintf("%s.reasoning_steps[%d]", path, position)
		if err := index.add(itemPath+".local_key", step.LocalKey); err != nil {
			return err
		}
		for field, value := range map[string]string{
			"input": step.Input, "mechanism": step.Mechanism, "output": step.Output,
		} {
			if err := requiredText(itemPath+"."+field, value, 10_000); err != nil {
				return err
			}
		}
		if err := validateConfidence(itemPath+".confidence", step.Confidence); err != nil {
			return err
		}
		if err := validateEvidenceRefs(itemPath+".evidence_refs", step.EvidenceRefs, "reasoning_support"); err != nil {
			return err
		}
	}
	for field, value := range map[string]*string{
		"counterevidence":    layer.Uncertainty.Counterevidence,
		"evidence_gap":       layer.Uncertainty.EvidenceGap,
		"boundary":           layer.Uncertainty.Boundary,
		"reversal_condition": layer.Uncertainty.ReversalCondition,
	} {
		if err := optionalText(path+".uncertainty."+field, value, 10_000); err != nil {
			return err
		}
	}
	if err := validateEvidenceRefs(path+".evidence_refs", layer.EvidenceRefs, "summary_support"); err != nil {
		return err
	}
	if geopolitics {
		if layer.DownwardTransmission.ToMacroeconomics == nil || layer.DownwardTransmission.ToIndustryChains == nil {
			return invalid(path+".downward_transmission", "must contain to_macroeconomics and to_industry_chains")
		}
		if err := validateTransmissionGroup(path+".downward_transmission.to_macroeconomics", *layer.DownwardTransmission.ToMacroeconomics); err != nil {
			return err
		}
	} else if layer.DownwardTransmission.ToMacroeconomics != nil {
		return invalid(path+".downward_transmission.to_macroeconomics", "is not supported for macroeconomics")
	} else if layer.DownwardTransmission.ToIndustryChains == nil {
		return invalid(path+".downward_transmission.to_industry_chains", "is required")
	}
	return validateTransmissionGroup(path+".downward_transmission.to_industry_chains", *layer.DownwardTransmission.ToIndustryChains)
}

func validateTransmissionGroup(path string, group TransmissionGroup) error {
	if err := requiredText(path+".summary", group.Summary, 10_000); err != nil {
		return err
	}
	if group.Paths == nil {
		return invalid(path+".paths", "must be an array")
	}
	seen := make(map[string]struct{}, len(group.Paths))
	for position, transmission := range group.Paths {
		itemPath := fmt.Sprintf("%s.paths[%d]", path, position)
		if !localKeyPattern.MatchString(transmission.LocalKey) {
			return invalid(itemPath+".local_key", "must be a Report-local key")
		}
		if _, duplicate := seen[transmission.LocalKey]; duplicate {
			return invalid(itemPath+".local_key", "duplicates a local key in this transmission group")
		}
		seen[transmission.LocalKey] = struct{}{}
		for field, value := range map[string]string{
			"source_conclusion":  transmission.SourceConclusion,
			"transmission_logic": transmission.TransmissionLogic,
		} {
			if err := requiredText(itemPath+"."+field, value, 10_000); err != nil {
				return err
			}
		}
		if len(transmission.Targets) == 0 {
			return invalid(itemPath+".targets", "must contain at least one target")
		}
		if err := validateMappedLabel(itemPath+".transmission_kind", transmission.TransmissionKind, transmissionKindLabels); err != nil {
			return err
		}
		if err := validateConfidence(itemPath+".confidence", transmission.Confidence); err != nil {
			return err
		}
		if err := validateMappedLabel(itemPath+".status", transmission.Status, transmissionStatusLabels); err != nil {
			return err
		}
		for targetPosition, target := range transmission.Targets {
			targetPath := fmt.Sprintf("%s.targets[%d]", itemPath, targetPosition)
			if err := validateMappedLabel(targetPath+".target_type", target.TargetType, targetTypeLabels); err != nil {
				return err
			}
			if !localKeyPattern.MatchString(target.TargetLocalKey) {
				return invalid(targetPath+".target_local_key", "must be a Report-local key")
			}
			if err := requiredText(targetPath+".target_name", target.TargetName, 500); err != nil {
				return err
			}
			if err := validateResult(targetPath+".result", target.Result); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateIndustryChain(path string, chain IndustryChain, index *reportIndex) error {
	if err := index.add(path+".local_key", chain.LocalKey); err != nil {
		return err
	}
	index.chains[chain.LocalKey] = struct{}{}
	for field, value := range map[string]string{"name": chain.Name, "conclusion": chain.Conclusion} {
		if err := requiredText(path+"."+field, value, 10_000); err != nil {
			return err
		}
	}
	if err := validateResult(path+".result", chain.Result); err != nil {
		return err
	}
	if err := validateTimeWindow(path+".time_window", chain.TimeWindow); err != nil {
		return err
	}
	if err := validateConfidence(path+".confidence", chain.Confidence); err != nil {
		return err
	}
	if err := optionalText(path+".path_summary", chain.PathSummary, 10_000); err != nil {
		return err
	}
	if err := optionalText(path+".accepted_hypothesis_summary", chain.AcceptedHypothesisSummary, 10_000); err != nil {
		return err
	}
	if len(chain.Nodes) == 0 || chain.Edges == nil {
		return invalid(path, "nodes must be non-empty and edges must be an array")
	}
	if err := validateEvidenceRefs(path+".evidence_refs", chain.EvidenceRefs, "summary_support"); err != nil {
		return err
	}
	if err := optionalText(path+".uncertainty.counterevidence_and_gap", chain.Uncertainty.CounterevidenceAndGap, 10_000); err != nil {
		return err
	}
	if err := optionalText(path+".uncertainty.stop_condition", chain.Uncertainty.StopCondition, 10_000); err != nil {
		return err
	}
	chainNodeKeys := make(map[string]struct{}, len(chain.Nodes))
	for position, node := range chain.Nodes {
		nodePath := fmt.Sprintf("%s.nodes[%d]", path, position)
		if err := index.add(nodePath+".local_key", node.LocalKey); err != nil {
			return err
		}
		index.nodes[node.LocalKey] = struct{}{}
		chainNodeKeys[node.LocalKey] = struct{}{}
		if err := validateAssessment(nodePath, node.Name, "impact", node.Impact, node.Reasoning,
			node.Result, node.ConclusionBasis, node.ValidationStatus, node.TimeWindow,
			node.Confidence, node.EvidenceRefs); err != nil {
			return err
		}
	}
	for position, edge := range chain.Edges {
		edgePath := fmt.Sprintf("%s.edges[%d]", path, position)
		if _, exists := chainNodeKeys[edge.FromNodeLocalKey]; !exists {
			return invalid(edgePath+".from_node_local_key", "must reference this chain")
		}
		if _, exists := chainNodeKeys[edge.ToNodeLocalKey]; !exists {
			return invalid(edgePath+".to_node_local_key", "must reference this chain")
		}
		if edge.FromNodeLocalKey == edge.ToNodeLocalKey {
			return invalid(edgePath, "must not be a self edge")
		}
		if err := requiredText(edgePath+".relation_label", edge.RelationLabel, 500); err != nil {
			return err
		}
	}
	return nil
}

func validateAssessment(path, name, stateField, state, reasoning string, result CodedLabel, basis, validation CodedLabel,
	window TimeWindow, confidence Confidence, evidenceRefs []EvidenceReference,
) error {
	for field, value := range map[string]string{"name": name, "reasoning": reasoning, stateField: state} {
		if err := requiredText(path+"."+field, value, 10_000); err != nil {
			return err
		}
	}
	if err := validateResult(path+".result", result); err != nil {
		return err
	}
	if err := validateMappedLabel(path+".conclusion_basis", basis, basisLabels); err != nil {
		return err
	}
	if err := validateMappedLabel(path+".validation_status", validation, validationLabels); err != nil {
		return err
	}
	if err := validateTimeWindow(path+".time_window", window); err != nil {
		return err
	}
	if err := validateConfidence(path+".confidence", confidence); err != nil {
		return err
	}
	if err := validateEvidenceRefs(path+".evidence_refs", evidenceRefs, "direct_support"); err != nil {
		return err
	}
	switch basis.Code {
	case BasisDirectEvidence:
		if validation.Code != ValidationConfirmed {
			return invalid(path+".validation_status", "direct evidence must be confirmed")
		}
		if len(evidenceRefs) == 0 {
			return invalid(path+".evidence_refs", "direct evidence requires at least one Evidence reference")
		}
	case BasisReasoningHypothesis, BasisNoDirectional:
		if validation.Code != ValidationPending {
			return invalid(path+".validation_status", "non-direct conclusions must be pending validation")
		}
		if len(evidenceRefs) != 0 {
			return invalid(path+".evidence_refs", "non-direct conclusions cannot expose direct Evidence")
		}
	}
	return nil
}

func validateTransmissionTargets(report Report, index reportIndex) error {
	validateLayerTargets := func(path string, layer *Layer) error {
		if layer == nil {
			return nil
		}
		groups := []*TransmissionGroup{
			layer.DownwardTransmission.ToMacroeconomics,
			layer.DownwardTransmission.ToIndustryChains,
		}
		for _, group := range groups {
			if group == nil {
				continue
			}
			for pathPosition, transmission := range group.Paths {
				for targetPosition, target := range transmission.Targets {
					exists := false
					switch target.TargetType.Code {
					case "macro_anchor":
						section, ok := index.anchors[target.TargetLocalKey]
						exists = ok && section == "macroeconomics"
					case "industry_chain":
						_, exists = index.chains[target.TargetLocalKey]
					case "industry_chain_node":
						_, exists = index.nodes[target.TargetLocalKey]
					}
					if !exists {
						return &ReferenceError{
							Path:      fmt.Sprintf("%s.downward_transmission.paths[%d].targets[%d]", path, pathPosition, targetPosition),
							Reference: target.TargetType.Code + ":" + target.TargetLocalKey,
							Message:   "does not identify a Report-local target",
						}
					}
				}
			}
		}
		return nil
	}
	if err := validateLayerTargets("report.geopolitics", report.Geopolitics); err != nil {
		return err
	}
	return validateLayerTargets("report.macroeconomics", report.Macroeconomics)
}

func validateResult(path string, value CodedLabel) error {
	return validateMappedLabel(path, value, resultLabels)
}

func validateConfidence(path string, value Confidence) error {
	return validateMappedLabel(path, value, confidenceLabels)
}

func validateTimeWindow(path string, value TimeWindow) error {
	return validateMappedLabel(path, CodedLabel(value), timeWindowLabels)
}

func validateMappedLabel(path string, value CodedLabel, values map[string]string) error {
	want, ok := values[value.Code]
	if !ok {
		return invalid(path+".code", "is not supported")
	}
	if value.Label != want {
		return invalid(path+".label", "does not match code")
	}
	return nil
}

func validateEvidenceRefs(path string, values []EvidenceReference, requiredRole string) error {
	if values == nil {
		return invalid(path, "must be an array")
	}
	seen := make(map[string]struct{}, len(values))
	for position, reference := range values {
		itemPath := fmt.Sprintf("%s[%d]", path, position)
		if !coreid.Is(reference.EvidenceID, coreid.Evidence) {
			return &ReferenceError{Path: itemPath + ".evidence_id", Reference: reference.EvidenceID, Message: "must be a canonical Atomic Evidence ID"}
		}
		if _, duplicate := seen[reference.EvidenceID]; duplicate {
			return invalid(itemPath+".evidence_id", "duplicates an Evidence in this scope")
		}
		seen[reference.EvidenceID] = struct{}{}
		if err := validateMappedLabel(itemPath+".role", reference.Role, evidenceRoleLabels); err != nil {
			return err
		}
		if reference.Role.Code != requiredRole {
			return invalid(itemPath+".role.code", "does not match this Evidence scope")
		}
	}
	return nil
}

func requiredText(path, value string, max int) error {
	if strings.TrimSpace(value) == "" || value != strings.TrimSpace(value) {
		return invalid(path, "must not be blank or padded")
	}
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > max {
		return invalid(path, fmt.Sprintf("must contain at most %d valid UTF-8 characters", max))
	}
	return nil
}

func optionalText(path string, value *string, max int) error {
	if value == nil {
		return nil
	}
	return requiredText(path, *value, max)
}

func buildEvidenceLinks(reportID string, report Report) ([]EvidenceLink, error) {
	type scopedRefs struct {
		typeName ScopeType
		path     string
		refs     []EvidenceReference
	}
	values := make([]scopedRefs, 0)
	for _, section := range []struct {
		name  string
		layer *Layer
	}{{"geopolitics", report.Geopolitics}, {"macroeconomics", report.Macroeconomics}} {
		if section.layer == nil {
			continue
		}
		values = append(values, scopedRefs{ScopeSectionSummary, section.name + "/evidence_refs", section.layer.EvidenceRefs})
		for _, anchor := range section.layer.AffectedAnchors {
			values = append(values, scopedRefs{ScopeAnchor, section.name + "/affected_anchors/" + anchor.LocalKey + "/evidence_refs", anchor.EvidenceRefs})
		}
		for _, step := range section.layer.ReasoningSteps {
			values = append(values, scopedRefs{ScopeReasoningStep, section.name + "/reasoning_steps/" + step.LocalKey + "/evidence_refs", step.EvidenceRefs})
		}
	}
	for _, chain := range report.IndustryChains {
		prefix := "industry_chains/" + chain.LocalKey
		values = append(values, scopedRefs{ScopeIndustryChainSummary, prefix + "/evidence_refs", chain.EvidenceRefs})
		for _, node := range chain.Nodes {
			values = append(values, scopedRefs{ScopeIndustryChainNode, prefix + "/nodes/" + node.LocalKey + "/evidence_refs", node.EvidenceRefs})
		}
	}
	links := make([]EvidenceLink, 0)
	for _, value := range values {
		for position, reference := range value.refs {
			linkID, err := coreid.New(coreid.ReportEvidenceLink)
			if err != nil {
				return nil, fmt.Errorf("generate Report Evidence Link ID: %w", err)
			}
			links = append(links, EvidenceLink{
				ID: linkID, ReportID: reportID, EvidenceID: reference.EvidenceID,
				ScopeType: value.typeName, ScopePath: value.path, Position: position + 1,
			})
		}
	}
	return links, nil
}

func uniqueEvidenceIDs(links []EvidenceLink) []string {
	seen := make(map[string]struct{}, len(links))
	for _, link := range links {
		seen[link.EvidenceID] = struct{}{}
	}
	result := make([]string, 0, len(seen))
	for id := range seen {
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}

func firstMissingEvidence(want, got []string) string {
	existing := make(map[string]struct{}, len(got))
	for _, id := range got {
		existing[id] = struct{}{}
	}
	for _, id := range want {
		if _, ok := existing[id]; !ok {
			return id
		}
	}
	return ""
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	result := value.UTC()
	return &result
}

func sameOptionalTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}
