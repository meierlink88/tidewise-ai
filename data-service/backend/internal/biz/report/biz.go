package report

import (
	"bytes"
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
	ScopeStorySummary         ScopeType = "story_summary"
	ScopeConceptSummary       ScopeType = "concept_summary"
	ScopeChainReasoningStep   ScopeType = "chain_reasoning_step"
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
	V4                   *V4Report       `json:"-"`
	SchemaVersion        string          `json:"schema_version,omitempty"`
	AnalysisWindow       *AnalysisWindow `json:"analysis_window,omitempty"`
	GeopoliticalStories  []AnalysisUnit  `json:"geopolitical_stories,omitempty"`
	MacroeconomicStories []AnalysisUnit  `json:"macroeconomic_stories,omitempty"`
	ConceptAnalyses      []AnalysisUnit  `json:"concept_analyses,omitempty"`

	ReportType     CodedLabel      `json:"report_type"`
	GeneratedAt    time.Time       `json:"generated_at"`
	Timezone       string          `json:"timezone"`
	Geopolitics    *Layer          `json:"geopolitics,omitempty"`
	Macroeconomics *Layer          `json:"macroeconomics,omitempty"`
	IndustryChains []IndustryChain `json:"industry_chains,omitempty"`
}

type Record struct {
	EvidenceCounts    map[string]int
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
	SemanticTags []EvidenceTag
	PublishedAt  *time.Time
	Summary      string
	Keywords     []string
}

type Summary struct {
	SchemaVersion                          string
	AnalysisWindowStart, AnalysisWindowEnd string
	ID                                     string
	PublisherReportID                      string
	GeneratedAt                            time.Time
	HasGeopolitics                         bool
	HasMacroeconomics                      bool
	IndustryChainCount                     int
	PublishedAt                            time.Time
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
	SchemaVersion              string
	PublishedFrom, PublishedTo *time.Time
	Limit                      int
	Cursor                     string
}

type ListFilter struct {
	SchemaVersion                                 string
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
	ListAnalyses(context.Context, AnalysisListFilter) (AnalysisStorePage, error)
	GetAnalysis(context.Context, string, string, string) (AnalysisUnitDetail, error)
	GetAnalysisChain(context.Context, string, string, string, string) (ChainAnalysisDetail, error)
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
		// An immutable snapshot can be replayed even when later publication rules tighten.
		if err := ValidateReport(report); err != nil {
			return err
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
		record := Record{EvidenceCounts: countEvidenceScopes(links), ID: reportID, PublisherReportID: publisherReportID, ContentHash: payloadHash, Report: report, PublishedAt: s.now().UTC()}
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

// Counts are server-owned metadata, excluded from the publisher payload and replay hash.
func countEvidenceScopes(links []EvidenceLink) map[string]int {
	scopes := map[string]map[string]struct{}{}
	for _, link := range links {
		if scopes[link.ScopePath] == nil {
			scopes[link.ScopePath] = map[string]struct{}{}
		}
		scopes[link.ScopePath][link.EvidenceID] = struct{}{}
	}
	counts := map[string]int{}
	for path, ids := range scopes {
		counts[path] = len(ids)
	}
	return counts
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
	if request.SchemaVersion != "" && request.SchemaVersion != "legacy" && request.SchemaVersion != AnalysisSchemaVersion && request.SchemaVersion != NormalizedSchemaVersion && request.SchemaVersion != SignalSchemaVersion {
		return Page{}, invalid("schema_version", "unsupported version")
	}
	selection := request.SchemaVersion
	if selection == "" {
		selection = "all"
	}
	filter := ListFilter{SchemaVersion: selection, PublishedFrom: cloneTime(request.PublishedFrom), PublishedTo: cloneTime(request.PublishedTo), Limit: request.Limit}
	if strings.TrimSpace(request.Cursor) != "" {
		cursor, err := decodeReportCursor(request.Cursor)
		if err != nil || cursor.Version != 1 || cursor.SchemaVersion != selection || !coreid.Is(cursor.ID, coreid.Report) || !sameOptionalTime(cursor.PublishedFrom, request.PublishedFrom) || !sameOptionalTime(cursor.PublishedTo, request.PublishedTo) {
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
		encoded, err := encodeCursor(reportCursor{SchemaVersion: selection, Version: 1, PublishedFrom: cloneTime(request.PublishedFrom), PublishedTo: cloneTime(request.PublishedTo), PublishedAt: last.PublishedAt.UTC(), ID: last.ID})
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
	SchemaVersion string     `json:"schema_version,omitempty"`
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
	impactLevelLabels        = map[string]string{ImpactLevelHigh: "高影响", ImpactLevelMedium: "中影响", ImpactLevelLow: "低影响", ImpactLevelPending: "待评估"}
	targetTypeLabels         = map[string]string{
		"macro_anchor": "宏观经济锚点", "industry_chain": "产业链", "industry_chain_node": "产业链节点",
	}
)

func ValidateReport(report Report) error {
	if report.V4 != nil {
		return validateNormalizedReport(*report.V4)
	}
	if err := validateMappedLabel("report.report_type", report.ReportType, map[string]string{"investment_reasoning": "投研推理报告"}); err != nil {
		return err
	}
	if report.GeneratedAt.IsZero() {
		return invalid("report.generated_at", "must be a timestamp")
	}
	if report.Timezone != "Asia/Shanghai" {
		return invalid("report.timezone", "must be Asia/Shanghai")
	}
	if report.SchemaVersion != "" {
		return validateAnalysisReport(report)
	}
	if report.AnalysisWindow != nil || report.GeopoliticalStories != nil || report.MacroeconomicStories != nil || report.ConceptAnalyses != nil {
		return invalid("report.schema_version", "required for analysis fields")
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
	if report.V4 != nil {
		return normalizedEvidenceLinks(reportID, *report.V4)
	}
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

	for _, group := range []struct {
		kind  string
		units []AnalysisUnit
	}{{"geopolitical_stories", report.GeopoliticalStories}, {"macroeconomic_stories", report.MacroeconomicStories}, {"concept_analyses", report.ConceptAnalyses}} {
		for _, unit := range group.units {
			prefix := group.kind + "/" + unit.LocalKey
			scope := ScopeStorySummary
			if group.kind == "concept_analyses" {
				scope = ScopeConceptSummary
			}
			values = append(values, scopedRefs{scope, prefix + "/summary/evidence_refs", unit.Summary.EvidenceRefs})
			if impact := unit.Summary.ImpactAssessment; impact != nil {
				values = append(values, scopedRefs{scope, prefix + "/summary/impact_assessment/evidence_refs", impact.EvidenceRefs})
			}
			for _, a := range unit.Detail.AffectedAnchors {
				values = append(values, scopedRefs{ScopeAnchor, prefix + "/detail/affected_anchors/" + a.LocalKey + "/evidence_refs", a.EvidenceRefs})
			}
			for _, step := range unit.Detail.ReasoningSteps {
				values = append(values, scopedRefs{ScopeReasoningStep, prefix + "/detail/reasoning_steps/" + step.LocalKey + "/evidence_refs", step.EvidenceRefs})
			}
			for _, c := range unit.Detail.IndustryChains {
				cp := prefix + "/detail/industry_chains/" + c.LocalKey
				values = append(values, scopedRefs{ScopeIndustryChainSummary, cp + "/evidence_refs", c.EvidenceRefs})
				for _, a := range c.AffectedNodes {
					values = append(values, scopedRefs{ScopeIndustryChainNode, cp + "/affected_nodes/" + a.LocalKey + "/evidence_refs", a.EvidenceRefs})
				}
				for _, step := range c.ReasoningSteps {
					values = append(values, scopedRefs{ScopeChainReasoningStep, cp + "/reasoning_steps/" + step.LocalKey + "/evidence_refs", step.EvidenceRefs})
				}
			}
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

// AnalysisWindow records the publisher's observation interval, separately from the forecast horizon.
type AnalysisWindow struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

const (
	ImpactLevelHigh    = "high"
	ImpactLevelMedium  = "medium"
	ImpactLevelLow     = "low"
	ImpactLevelPending = "pending"
)

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

const AnalysisSchemaVersion = "report-publication/v3"

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

func validateAnalysisReport(r Report) error {
	if r.SchemaVersion != AnalysisSchemaVersion {
		return invalid("report.schema_version", "unsupported version")
	}
	if r.Geopolitics != nil || r.Macroeconomics != nil || r.IndustryChains != nil {
		return invalid("report", "cannot mix legacy and analysis contracts")
	}
	if r.AnalysisWindow == nil {
		return invalid("report.analysis_window", "is required")
	}
	start, e1 := time.Parse(time.RFC3339Nano, r.AnalysisWindow.Start)
	end, e2 := time.Parse(time.RFC3339Nano, r.AnalysisWindow.End)
	if e1 != nil || e2 != nil || !start.Before(end) {
		return invalid("report.analysis_window", "requires ordered RFC3339 timestamps")
	}
	if r.GeopoliticalStories == nil || r.MacroeconomicStories == nil || r.ConceptAnalyses == nil {
		return invalid("report", "analysis arrays must be present, possibly empty")
	}
	if len(r.GeopoliticalStories)+len(r.MacroeconomicStories)+len(r.ConceptAnalyses) == 0 {
		return invalid("report", "requires at least one analysis")
	}
	index := reportIndex{allKeys: map[string]struct{}{}}
	for _, group := range []struct {
		kind  string
		units []AnalysisUnit
	}{{"geopolitical_stories", r.GeopoliticalStories}, {"macroeconomic_stories", r.MacroeconomicStories}, {"concept_analyses", r.ConceptAnalyses}} {
		sources := map[string]bool{}
		for _, unit := range group.units {
			if sources[unit.SourceID] {
				return invalid(group.kind, "duplicate source within analysis group")
			}
			sources[unit.SourceID] = true
			if err := validateAnalysisUnit(group.kind, unit, &index); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateAnalysisUnit(kind string, u AnalysisUnit, index *reportIndex) error {
	p := kind + "/" + u.LocalKey
	if err := index.add(p, u.LocalKey); err != nil {
		return err
	}
	if err := requiredText(p+"/source_id", u.SourceID, 200); err != nil {
		return err
	}
	for field, value := range map[string]string{"source_id": u.SourceID, "title": u.Title, "summary.conclusion": u.Summary.Conclusion, "summary.transmission_logic": u.Summary.TransmissionLogic} {
		if err := requiredText(p+"."+field, value, 10000); err != nil {
			return err
		}
	}
	if err := validateAnalysisEvidenceRefs(p+"/summary/evidence_refs", u.Summary.EvidenceRefs, "summary_support"); err != nil {
		return err
	}
	if a := u.Summary.ImpactAssessment; a != nil {
		if err := validateMappedLabel(p+"/summary/impact_assessment/level", a.Level, impactLevelLabels); err != nil {
			return err
		}
		if err := requiredText(p+"/summary/impact_assessment/rationale", a.Rationale, 10000); err != nil {
			return err
		}
		if a.Level.Code != ImpactLevelPending && len(a.EvidenceRefs) == 0 {
			return invalid(p+"/summary/impact_assessment/evidence_refs", "rated impact requires supporting Evidence")
		}
		if err := validateAnalysisEvidenceRefs(p+"/summary/impact_assessment/evidence_refs", a.EvidenceRefs, "summary_support"); err != nil {
			return err
		}
	}
	if err := validateAnalysisSteps(p+"/detail", u.Detail.ReasoningSteps, index); err != nil {
		return err
	}
	if err := validateAnalysisUncertainty(p, u.Detail.Uncertainty); err != nil {
		return err
	}
	if u.Summary.AnchorKeys == nil || u.Detail.AffectedAnchors == nil || u.Detail.IndustryChains == nil {
		return invalid(p, "arrays must not be null")
	}
	impacts := map[string]bool{}
	storyChains := map[[2]string]bool{}
	for _, a := range u.Detail.AffectedAnchors {
		if err := validateAnalysisImpact(p, a, index, false); err != nil {
			return err
		}
		if kind == "geopolitical_stories" && a.TargetType.Code != "macro_anchor" && a.TargetType.Code != "industry_chain" || kind == "macroeconomic_stories" && a.TargetType.Code != "industry_chain" {
			return invalid(p, "story anchors must target the permitted downstream layer")
		}
		if a.TargetType.Code == "industry_chain" {
			storyChains[[2]string{a.SourceID, a.Name}] = true
		}
		impacts[a.LocalKey] = true
	}
	if kind == "concept_analyses" {
		if len(u.Detail.IndustryChains) == 0 || len(u.Detail.AffectedAnchors) != 0 {
			return invalid(p, "Concept requires chains; impacts belong to chain nodes")
		}
	}
	chainSources := map[string]bool{}
	for _, c := range u.Detail.IndustryChains {
		if chainSources[c.SourceID] {
			return invalid(p, "duplicate chain source within analysis")
		}
		chainSources[c.SourceID] = true
		if kind != "concept_analyses" && !storyChains[[2]string{c.SourceID, c.Name}] {
			return invalid(p, "story chain must match an affected industry-chain anchor source and name")
		}
		if err := validateChainAnalysis(p, c, index); err != nil {
			return err
		}
		if kind == "concept_analyses" {
			for _, a := range c.AffectedNodes {
				impacts[a.LocalKey] = true
			}
		}
	}

	seen := map[string]bool{}
	for _, k := range u.Summary.AnchorKeys {
		if !impacts[k] || seen[k] {
			return invalid(p+"/summary/anchor_keys", "must uniquely reference this unit's impacts")
		}
		seen[k] = true
	}
	return nil
}

func validateAnalysisUncertainty(p string, u LayerUncertainty) error {
	for field, value := range map[string]*string{"counterevidence": u.Counterevidence, "evidence_gap": u.EvidenceGap, "boundary": u.Boundary, "reversal_condition": u.ReversalCondition} {
		if err := optionalText(p+"/uncertainty/"+field, value, 10000); err != nil {
			return err
		}
	}
	return nil
}

func validateAnalysisSteps(p string, steps []ReasoningStep, index *reportIndex) error {
	if steps == nil {
		return invalid(p+"/reasoning_steps", "must not be null")
	}
	for _, s := range steps {
		if err := index.add(p, s.LocalKey); err != nil {
			return err
		}
		for field, value := range map[string]string{"input": s.Input, "mechanism": s.Mechanism, "output": s.Output} {
			if err := requiredText(p+"/"+field, value, 10000); err != nil {
				return err
			}
		}
		if err := validateConfidence(p, s.Confidence); err != nil {
			return err
		}
		if err := validateAnalysisEvidenceRefs(p, s.EvidenceRefs, "reasoning_support"); err != nil {
			return err
		}
	}
	return nil
}

func validateAnalysisImpact(p string, a AnalysisImpact, index *reportIndex, node bool) error {
	if err := index.add(p, a.LocalKey); err != nil {
		return err
	}
	if err := requiredText(p+"/source_id", a.SourceID, 200); err != nil {
		return err
	}
	for field, value := range map[string]string{"source_id": a.SourceID, "name": a.Name, "impact": a.Impact, "reasoning": a.Reasoning} {
		if err := requiredText(p+"/"+field, value, 10000); err != nil {
			return err
		}
	}
	if err := validateMappedLabel(p+"/target_type", a.TargetType, targetTypeLabels); err != nil {
		return err
	}
	if node && (a.TargetType.Code != "industry_chain_node" || a.NodeLocalKey == nil) {
		return invalid(p, "chain impact requires a topology node reference")
	}
	if !node && a.NodeLocalKey != nil {
		return invalid(p, "story anchor cannot reference a chain-local topology node")
	}
	if err := validateResult(p, a.Result); err != nil {
		return err
	}
	if err := validateConfidence(p, a.Confidence); err != nil {
		return err
	}
	if err := validateTimeWindow(p, a.TimeWindow); err != nil {
		return err
	}
	if err := validateMappedLabel(p, a.ConclusionBasis, basisLabels); err != nil {
		return err
	}
	if err := validateMappedLabel(p, a.ValidationStatus, validationLabels); err != nil {
		return err
	}
	role := "reasoning_support"
	if a.ConclusionBasis.Code == BasisDirectEvidence {
		role = "direct_support"
		if a.ValidationStatus.Code != ValidationConfirmed || len(a.EvidenceRefs) == 0 {
			return invalid(p, "direct evidence requires confirmed status and Evidence")
		}
	} else if a.ValidationStatus.Code != ValidationPending {
		return invalid(p, "hypotheses must remain pending validation")
	}
	if a.ConclusionBasis.Code == BasisNoDirectional && a.Result.Code != ResultPending {
		return invalid(p, "no directional conclusion requires pending result")
	}
	if err := validateAnalysisEvidenceRefs(p, a.EvidenceRefs, role); err != nil {
		return err
	}
	if err := optionalText(p, a.TransmissionSignal, 10000); err != nil {
		return err
	}
	if a.Conditions == nil || a.FollowUp == nil {
		return invalid(p, "conditions and follow_up must be arrays")
	}
	for _, list := range [][]string{a.Conditions, a.FollowUp} {
		for _, value := range list {
			if err := requiredText(p, value, 10000); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateChainAnalysis(p string, c ChainAnalysis, index *reportIndex) error {
	if err := index.add(p, c.LocalKey); err != nil {
		return err
	}
	if err := requiredText(p+"/source_id", c.SourceID, 200); err != nil {
		return err
	}
	for field, value := range map[string]string{"source_id": c.SourceID, "name": c.Name, "conclusion": c.Conclusion, "transmission_logic": c.TransmissionLogic} {
		if err := requiredText(p+"/"+field, value, 10000); err != nil {
			return err
		}
	}
	if err := validateAnalysisEvidenceRefs(p, c.EvidenceRefs, "summary_support"); err != nil {
		return err
	}
	if err := validateAnalysisSteps(p, c.ReasoningSteps, index); err != nil {
		return err
	}
	if err := validateAnalysisUncertainty(p, c.Uncertainty); err != nil {
		return err
	}
	if c.Graph.Nodes == nil || c.Graph.Edges == nil || c.AffectedNodes == nil {
		return invalid(p, "graph and affected_nodes must not be null")
	}
	nodes := map[string]AnalysisTopologyNode{}
	for _, n := range c.Graph.Nodes {
		if err := index.add(p, n.LocalKey); err != nil {
			return err
		}
		if err := requiredText(p, n.Name, 10000); err != nil {
			return err
		}
		if err := requiredText(p, n.SourceID, 200); err != nil {
			return err
		}
		nodes[n.LocalKey] = n
	}
	edges := map[string]bool{}
	for _, e := range c.Graph.Edges {
		_, from := nodes[e.FromNodeLocalKey]
		_, to := nodes[e.ToNodeLocalKey]
		key := e.FromNodeLocalKey + "/" + e.ToNodeLocalKey + "/" + e.RelationLabel
		if !from || !to || e.FromNodeLocalKey == e.ToNodeLocalKey || edges[key] {
			return invalid(p, "edge endpoints must close within chain and edges must be unique")
		}
		edges[key] = true
		if err := requiredText(p, e.RelationLabel, 1000); err != nil {
			return err
		}
	}
	for _, a := range c.AffectedNodes {
		if err := validateAnalysisImpact(p, a, index, true); err != nil {
			return err
		}
		n, ok := nodes[*a.NodeLocalKey]
		if !ok || n.SourceID != a.SourceID || n.Name != a.Name {
			return invalid(p, "impact must match its chain topology node snapshot")
		}
	}
	return nil
}

type AnalysisListRequest struct {
	ReportID, Kind, Cursor string
	Limit                  int
}
type AnalysisListFilter struct {
	ReportID, Kind      string
	AfterOrdinal, Limit int
}
type AnalysisStorePage struct {
	Items   []AnalysisUnitSummary
	HasMore bool
}
type AnalysisPage struct {
	Items      []AnalysisUnitSummary `json:"items"`
	NextCursor *string               `json:"next_cursor"`
}
type analysisCursor struct {
	Version        int
	ReportID, Kind string
	Ordinal        int
}

func validateAnalysisKind(kind string) error {
	switch kind {
	case "geopolitical_stories", "macroeconomic_stories", "concept_analyses", "industry_chain_analyses", "company_analyses":
		return nil
	}
	return invalid("kind", "must identify a story or Concept analysis collection")
}
func (s *UseCase) ListAnalyses(ctx context.Context, r AnalysisListRequest) (AnalysisPage, error) {
	if err := validateReportID(r.ReportID); err != nil {
		return AnalysisPage{}, err
	}
	if err := validateAnalysisKind(r.Kind); err != nil {
		return AnalysisPage{}, err
	}
	if r.Limit == 0 {
		r.Limit = DefaultLimit
	}
	if r.Limit < 1 || r.Limit > MaxLimit {
		return AnalysisPage{}, invalid("limit", "out of range")
	}
	f := AnalysisListFilter{ReportID: r.ReportID, Kind: r.Kind, Limit: r.Limit}
	if r.Cursor != "" {
		payload, err := base64.RawURLEncoding.DecodeString(r.Cursor)
		var c analysisCursor
		if err != nil || json.Unmarshal(payload, &c) != nil || c.Version != 1 || c.ReportID != r.ReportID || c.Kind != r.Kind || c.Ordinal < 1 {
			return AnalysisPage{}, invalid("cursor", "invalid for analysis query")
		}
		f.AfterOrdinal = c.Ordinal
	}
	page, err := s.store.ListAnalyses(ctx, f)
	if err != nil {
		return AnalysisPage{}, err
	}
	result := AnalysisPage{Items: page.Items}
	if page.HasMore && len(page.Items) > 0 {
		c, err := encodeCursor(analysisCursor{Version: 1, ReportID: r.ReportID, Kind: r.Kind, Ordinal: page.Items[len(page.Items)-1].Ordinal})
		if err != nil {
			return AnalysisPage{}, err
		}
		result.NextCursor = &c
	}
	return result, nil
}
func (s *UseCase) GetAnalysis(ctx context.Context, reportID, kind, key string) (AnalysisUnitDetail, error) {
	if err := validateReportID(reportID); err != nil {
		return AnalysisUnitDetail{}, err
	}
	if err := validateAnalysisKind(kind); err != nil {
		return AnalysisUnitDetail{}, err
	}
	if !localKeyPattern.MatchString(key) {
		return AnalysisUnitDetail{}, invalid("analysis_key", "invalid local key")
	}
	return s.store.GetAnalysis(ctx, reportID, kind, key)
}
func (s *UseCase) GetAnalysisChain(ctx context.Context, reportID, kind, analysisKey, chainKey string) (ChainAnalysisDetail, error) {
	if err := validateReportID(reportID); err != nil {
		return ChainAnalysisDetail{}, err
	}
	if err := validateAnalysisKind(kind); err != nil {
		return ChainAnalysisDetail{}, err
	}
	if !localKeyPattern.MatchString(analysisKey) || !localKeyPattern.MatchString(chainKey) {
		return ChainAnalysisDetail{}, invalid("local_key", "invalid analysis or chain key")
	}
	return s.store.GetAnalysisChain(ctx, reportID, kind, analysisKey, chainKey)
}

func validateAnalysisEvidenceRefs(path string, refs []EvidenceReference, role string) error {
	if refs == nil {
		return invalid(path, "evidence_refs must be an array")
	}
	return validateEvidenceRefs(path, refs, role)
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
		stamp, err := time.Parse(time.RFC3339Nano, parsed.GeneratedAt)
		if err != nil {
			return err
		}
		*r = Report{V4: &parsed, SchemaVersion: parsed.SchemaVersion, ReportType: CodedLabel{Code: parsed.ReportType.Code, Label: parsed.ReportType.Label}, GeneratedAt: stamp, Timezone: parsed.Timezone}
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

func validateV4CodedLabel(p string, v V4CodedLabel) error {
	if err := requiredText(p+"/code", v.Code, 16000); err != nil {
		return err
	}
	if err := requiredText(p+"/label", v.Label, 16000); err != nil {
		return err
	}
	return nil
}
func validateV4Claim(p string, v V4Claim) error {
	if err := requiredText(p+"/text", v.Text, 16000); err != nil {
		return err
	}
	if err := requiredText(p+"/basis", v.Basis, 16000); err != nil {
		return err
	}
	if !v4OneOf(v.Basis, "source_fact", "inference", "hypothetical_buffer") {
		return invalid(p+"/basis", "unsupported value")
	}
	if v.EvidenceIDs == nil {
		return invalid(p+"/evidence_ids", "must be an array")
	}
	for i, value := range v.EvidenceIDs {
		if err := requiredText(p+"/evidence_ids"+fmt.Sprintf("/%d", i), value, 16000); err != nil {
			return err
		}
		if !regexp.MustCompile("^EVD[0-9a-f-]{36}$").MatchString(value) {
			return invalid(p+"/evidence_ids"+fmt.Sprintf("/%d", i), "invalid object ID")
		}
	}
	if !v4Unique(v.EvidenceIDs) {
		return invalid(p+"/evidence_ids", "duplicate values")
	}
	return nil
}
func validateV4Objections(p string, v V4Objections) error {
	if err := requiredText(p+"/summary", v.Summary, 16000); err != nil {
		return err
	}
	if v.Counterevidence == nil {
		return invalid(p+"/counterevidence", "must be an array")
	}
	for i, value := range v.Counterevidence {
		if err := validateV4Claim(p+"/counterevidence"+fmt.Sprintf("/%d", i), value); err != nil {
			return err
		}
	}
	if v.Buffers == nil {
		return invalid(p+"/buffers", "must be an array")
	}
	for i, value := range v.Buffers {
		if err := validateV4Claim(p+"/buffers"+fmt.Sprintf("/%d", i), value); err != nil {
			return err
		}
	}
	if err := requiredText(p+"/counterevidence_status", v.CounterevidenceStatus, 16000); err != nil {
		return err
	}
	if !v4OneOf(v.CounterevidenceStatus, "identified", "none_identified") {
		return invalid(p+"/counterevidence_status", "unsupported value")
	}
	if v.EvidenceGaps == nil {
		return invalid(p+"/evidence_gaps", "must be an array")
	}
	for i, value := range v.EvidenceGaps {
		if err := requiredText(p+"/evidence_gaps"+fmt.Sprintf("/%d", i), value, 16000); err != nil {
			return err
		}
	}
	if v.ScopeLimits == nil {
		return invalid(p+"/scope_limits", "must be an array")
	}
	for i, value := range v.ScopeLimits {
		if err := requiredText(p+"/scope_limits"+fmt.Sprintf("/%d", i), value, 16000); err != nil {
			return err
		}
	}
	return nil
}
func validateV4Window(p string, v V4Window) error {
	if err := requiredText(p+"/kind", v.Kind, 16000); err != nil {
		return err
	}
	if !v4OneOf(v.Kind, "relative", "calendar", "stage", "mixed", "not_applicable") {
		return invalid(p+"/kind", "unsupported value")
	}
	if err := requiredText(p+"/description", v.Description, 16000); err != nil {
		return err
	}
	if v.StartAt != nil {
		if err := requiredText(p+"/start_at", (*v.StartAt), 16000); err != nil {
			return err
		}
		if _, err := time.Parse(time.RFC3339Nano, (*v.StartAt)); err != nil {
			return invalid(p+"/start_at", "must be RFC3339")
		}
	}
	if v.EndAt != nil {
		if err := requiredText(p+"/end_at", (*v.EndAt), 16000); err != nil {
			return err
		}
		if _, err := time.Parse(time.RFC3339Nano, (*v.EndAt)); err != nil {
			return invalid(p+"/end_at", "must be RFC3339")
		}
	}
	return nil
}
func validateV4Assessment(p string, v V4Assessment) error {
	if err := requiredText(p+"/conclusion", v.Conclusion, 16000); err != nil {
		return err
	}
	if err := requiredText(p+"/direction", v.Direction, 16000); err != nil {
		return err
	}
	if !v4OneOf(v.Direction, "warming", "cooling", "diverging", "pending") {
		return invalid(p+"/direction", "unsupported value")
	}
	if err := requiredText(p+"/conclusion_basis", v.ConclusionBasis, 16000); err != nil {
		return err
	}
	if !v4OneOf(v.ConclusionBasis, "reasoning_hypothesis", "observation_only") {
		return invalid(p+"/conclusion_basis", "unsupported value")
	}
	if err := requiredText(p+"/validation_status", v.ValidationStatus, 16000); err != nil {
		return err
	}
	if !v4OneOf(v.ValidationStatus, "pending_validation", "insufficient_evidence") {
		return invalid(p+"/validation_status", "unsupported value")
	}
	if v.Confidence != nil {
		if err := requiredText(p+"/confidence", (*v.Confidence), 16000); err != nil {
			return err
		}
		if !v4OneOf((*v.Confidence), "low", "medium", "high") {
			return invalid(p+"/confidence", "unsupported value")
		}
	}
	if err := validateV4Window(p+"/forecast_window", v.ForecastWindow); err != nil {
		return err
	}
	if err := requiredText(p+"/scope", v.Scope, 16000); err != nil {
		return err
	}
	if v.Conditions == nil {
		return invalid(p+"/conditions", "must be an array")
	}
	for i, value := range v.Conditions {
		if err := requiredText(p+"/conditions"+fmt.Sprintf("/%d", i), value, 16000); err != nil {
			return err
		}
	}
	if v.FollowUp == nil {
		return invalid(p+"/follow_up", "must be an array")
	}
	for i, value := range v.FollowUp {
		if err := requiredText(p+"/follow_up"+fmt.Sprintf("/%d", i), value, 16000); err != nil {
			return err
		}
	}
	if err := requiredText(p+"/transmission_logic", v.TransmissionLogic, 16000); err != nil {
		return err
	}
	if v.EvidenceIDs == nil {
		return invalid(p+"/evidence_ids", "must be an array")
	}
	for i, value := range v.EvidenceIDs {
		if err := requiredText(p+"/evidence_ids"+fmt.Sprintf("/%d", i), value, 16000); err != nil {
			return err
		}
		if !regexp.MustCompile("^EVD[0-9a-f-]{36}$").MatchString(value) {
			return invalid(p+"/evidence_ids"+fmt.Sprintf("/%d", i), "invalid object ID")
		}
	}
	if !v4Unique(v.EvidenceIDs) {
		return invalid(p+"/evidence_ids", "duplicate values")
	}
	return nil
}
func validateV4Node(p string, v V4Node) error {
	if err := requiredText(p+"/local_key", v.LocalKey, 16000); err != nil {
		return err
	}
	if err := requiredText(p+"/source_id", v.SourceID, 16000); err != nil {
		return err
	}
	if !regexp.MustCompile("^CND[0-9a-f-]{36}$").MatchString(v.SourceID) {
		return invalid(p+"/source_id", "invalid object ID")
	}
	if err := requiredText(p+"/node_local_key", v.NodeLocalKey, 16000); err != nil {
		return err
	}
	if err := requiredText(p+"/name", v.Name, 16000); err != nil {
		return err
	}
	if err := validateV4Assessment(p+"/assessment", v.Assessment); err != nil {
		return err
	}
	if err := validateV4Objections(p+"/objections", v.Objections); err != nil {
		return err
	}
	return nil
}
func validateV4Graph(p string, v V4Graph) error {
	if v.Nodes == nil {
		return invalid(p+"/nodes", "must be an array")
	}
	for i, value := range v.Nodes {
		if err := validateV4GraphNodesItem(p+"/nodes"+fmt.Sprintf("/%d", i), value); err != nil {
			return err
		}
	}
	if v.Edges == nil {
		return invalid(p+"/edges", "must be an array")
	}
	for i, value := range v.Edges {
		if err := validateV4GraphEdgesItem(p+"/edges"+fmt.Sprintf("/%d", i), value); err != nil {
			return err
		}
	}
	return nil
}
func validateV4Chain(p string, v V4Chain) error {
	if err := requiredText(p+"/local_key", v.LocalKey, 16000); err != nil {
		return err
	}
	if err := requiredText(p+"/source_id", v.SourceID, 16000); err != nil {
		return err
	}
	if !regexp.MustCompile("^ICH[0-9a-f-]{36}$").MatchString(v.SourceID) {
		return invalid(p+"/source_id", "invalid object ID")
	}
	if err := requiredText(p+"/name", v.Name, 16000); err != nil {
		return err
	}
	if err := validateV4Assessment(p+"/assessment", v.Assessment); err != nil {
		return err
	}
	if err := validateV4ChainReasoningSummary(p+"/reasoning_summary", v.ReasoningSummary); err != nil {
		return err
	}
	if err := validateV4Graph(p+"/graph", v.Graph); err != nil {
		return err
	}
	if v.AffectedNodes == nil {
		return invalid(p+"/affected_nodes", "must be an array")
	}
	for i, value := range v.AffectedNodes {
		if err := validateV4Node(p+"/affected_nodes"+fmt.Sprintf("/%d", i), value); err != nil {
			return err
		}
	}
	if v.EmptyState != nil {
		if err := validateV4ChainEmptyState(p+"/empty_state", (*v.EmptyState)); err != nil {
			return err
		}
	}
	return nil
}
func validateV4Macro(p string, v V4Macro) error {
	if err := requiredText(p+"/local_key", v.LocalKey, 16000); err != nil {
		return err
	}
	if err := requiredText(p+"/source_id", v.SourceID, 16000); err != nil {
		return err
	}
	if !regexp.MustCompile("^MEC[0-9a-f-]{36}$").MatchString(v.SourceID) {
		return invalid(p+"/source_id", "invalid object ID")
	}
	if err := requiredText(p+"/name", v.Name, 16000); err != nil {
		return err
	}
	if err := validateV4Assessment(p+"/assessment", v.Assessment); err != nil {
		return err
	}
	if err := validateV4Objections(p+"/objections", v.Objections); err != nil {
		return err
	}
	return nil
}
func validateV4AnchorRef(p string, v V4AnchorRef) error {
	if err := requiredText(p+"/target_type", v.TargetType, 16000); err != nil {
		return err
	}
	if !v4OneOf(v.TargetType, "macroeconomic_story", "industry_chain", "industry_chain_node") {
		return invalid(p+"/target_type", "unsupported value")
	}
	if err := requiredText(p+"/local_key", v.LocalKey, 16000); err != nil {
		return err
	}
	if v.ChainLocalKey != nil {
		if err := requiredText(p+"/chain_local_key", (*v.ChainLocalKey), 16000); err != nil {
			return err
		}
	}
	return nil
}
func validateV4Unit(p string, v V4Unit) error {
	if err := requiredText(p+"/local_key", v.LocalKey, 16000); err != nil {
		return err
	}
	if err := requiredText(p+"/source_id", v.SourceID, 16000); err != nil {
		return err
	}
	if err := requiredText(p+"/title", v.Title, 16000); err != nil {
		return err
	}
	if err := validateV4UnitSummary(p+"/summary", v.Summary); err != nil {
		return err
	}
	if err := validateV4UnitDetail(p+"/detail", v.Detail); err != nil {
		return err
	}
	return nil
}
func validateV4Report(p string, v V4Report) error {
	if err := requiredText(p+"/schema_version", v.SchemaVersion, 16000); err != nil {
		return err
	}
	if !v4OneOf(v.SchemaVersion, "report-publication/v4") {
		return invalid(p+"/schema_version", "unsupported value")
	}
	if err := validateV4CodedLabel(p+"/report_type", v.ReportType); err != nil {
		return err
	}
	if err := requiredText(p+"/generated_at", v.GeneratedAt, 16000); err != nil {
		return err
	}
	if _, err := time.Parse(time.RFC3339Nano, v.GeneratedAt); err != nil {
		return invalid(p+"/generated_at", "must be RFC3339")
	}
	if err := requiredText(p+"/timezone", v.Timezone, 16000); err != nil {
		return err
	}
	if !v4OneOf(v.Timezone, "Asia/Shanghai") {
		return invalid(p+"/timezone", "unsupported value")
	}
	if err := validateV4ReportAnalysisWindow(p+"/analysis_window", v.AnalysisWindow); err != nil {
		return err
	}
	if v.GeopoliticalStories == nil {
		return invalid(p+"/geopolitical_stories", "must be an array")
	}
	for i, value := range v.GeopoliticalStories {
		if err := validateV4Unit(p+"/geopolitical_stories"+fmt.Sprintf("/%d", i), value); err != nil {
			return err
		}
	}
	if v.MacroeconomicStories == nil {
		return invalid(p+"/macroeconomic_stories", "must be an array")
	}
	for i, value := range v.MacroeconomicStories {
		if err := validateV4Unit(p+"/macroeconomic_stories"+fmt.Sprintf("/%d", i), value); err != nil {
			return err
		}
	}
	if v.ConceptAnalyses == nil {
		return invalid(p+"/concept_analyses", "must be an array")
	}
	for i, value := range v.ConceptAnalyses {
		if err := validateV4Unit(p+"/concept_analyses"+fmt.Sprintf("/%d", i), value); err != nil {
			return err
		}
	}
	if v.Observations == nil {
		return invalid(p+"/observations", "must be an array")
	}
	for i, value := range v.Observations {
		if err := validateV4ReportObservationsItem(p+"/observations"+fmt.Sprintf("/%d", i), value); err != nil {
			return err
		}
	}
	if v.Limitations == nil {
		return invalid(p+"/limitations", "must be an array")
	}
	for i, value := range v.Limitations {
		if err := requiredText(p+"/limitations"+fmt.Sprintf("/%d", i), value, 16000); err != nil {
			return err
		}
	}
	return nil
}
func validateV4GraphNodesItem(p string, v V4GraphNodesItem) error {
	if err := requiredText(p+"/local_key", v.LocalKey, 16000); err != nil {
		return err
	}
	if err := requiredText(p+"/source_id", v.SourceID, 16000); err != nil {
		return err
	}
	if !regexp.MustCompile("^CND[0-9a-f-]{36}$").MatchString(v.SourceID) {
		return invalid(p+"/source_id", "invalid object ID")
	}
	if err := requiredText(p+"/name", v.Name, 16000); err != nil {
		return err
	}
	return nil
}
func validateV4GraphEdgesItem(p string, v V4GraphEdgesItem) error {
	if err := requiredText(p+"/from_node_local_key", v.FromNodeLocalKey, 16000); err != nil {
		return err
	}
	if err := requiredText(p+"/to_node_local_key", v.ToNodeLocalKey, 16000); err != nil {
		return err
	}
	if err := requiredText(p+"/relation_label", v.RelationLabel, 16000); err != nil {
		return err
	}
	return nil
}
func validateV4ChainReasoningSummary(p string, v V4ChainReasoningSummary) error {
	if err := requiredText(p+"/logic", v.Logic, 16000); err != nil {
		return err
	}
	if err := validateV4Claim(p+"/support", v.Support); err != nil {
		return err
	}
	if err := validateV4Objections(p+"/objections", v.Objections); err != nil {
		return err
	}
	return nil
}
func validateV4ChainEmptyState(p string, v V4ChainEmptyState) error {
	if err := requiredText(p+"/code", v.Code, 16000); err != nil {
		return err
	}
	if !v4OneOf(v.Code, "observation_only") {
		return invalid(p+"/code", "unsupported value")
	}
	if err := requiredText(p+"/reason", v.Reason, 16000); err != nil {
		return err
	}
	if v.FollowUp == nil {
		return invalid(p+"/follow_up", "must be an array")
	}
	for i, value := range v.FollowUp {
		if err := requiredText(p+"/follow_up"+fmt.Sprintf("/%d", i), value, 16000); err != nil {
			return err
		}
	}
	return nil
}
func validateV4UnitSummary(p string, v V4UnitSummary) error {
	if err := requiredText(p+"/conclusion", v.Conclusion, 16000); err != nil {
		return err
	}
	if err := requiredText(p+"/transmission_logic", v.TransmissionLogic, 16000); err != nil {
		return err
	}
	if err := validateV4UnitSummaryImpactAssessment(p+"/impact_assessment", v.ImpactAssessment); err != nil {
		return err
	}
	if v.AffectedRefs == nil {
		return invalid(p+"/affected_refs", "must be an array")
	}
	for i, value := range v.AffectedRefs {
		if err := validateV4AnchorRef(p+"/affected_refs"+fmt.Sprintf("/%d", i), value); err != nil {
			return err
		}
	}
	if v.EvidenceIDs == nil {
		return invalid(p+"/evidence_ids", "must be an array")
	}
	for i, value := range v.EvidenceIDs {
		if err := requiredText(p+"/evidence_ids"+fmt.Sprintf("/%d", i), value, 16000); err != nil {
			return err
		}
		if !regexp.MustCompile("^EVD[0-9a-f-]{36}$").MatchString(value) {
			return invalid(p+"/evidence_ids"+fmt.Sprintf("/%d", i), "invalid object ID")
		}
	}
	if !v4Unique(v.EvidenceIDs) {
		return invalid(p+"/evidence_ids", "duplicate values")
	}
	return nil
}
func validateV4UnitDetail(p string, v V4UnitDetail) error {
	if v.MacroImpacts == nil {
		return invalid(p+"/macro_impacts", "must be an array")
	}
	for i, value := range v.MacroImpacts {
		if err := validateV4Macro(p+"/macro_impacts"+fmt.Sprintf("/%d", i), value); err != nil {
			return err
		}
	}
	if v.IndustryChains == nil {
		return invalid(p+"/industry_chains", "must be an array")
	}
	for i, value := range v.IndustryChains {
		if err := validateV4Chain(p+"/industry_chains"+fmt.Sprintf("/%d", i), value); err != nil {
			return err
		}
	}
	return nil
}
func validateV4ReportAnalysisWindow(p string, v V4ReportAnalysisWindow) error {
	if err := requiredText(p+"/start", v.Start, 16000); err != nil {
		return err
	}
	if _, err := time.Parse(time.RFC3339Nano, v.Start); err != nil {
		return invalid(p+"/start", "must be RFC3339")
	}
	if err := requiredText(p+"/end", v.End, 16000); err != nil {
		return err
	}
	if _, err := time.Parse(time.RFC3339Nano, v.End); err != nil {
		return invalid(p+"/end", "must be RFC3339")
	}
	return nil
}
func validateV4ReportObservationsItem(p string, v V4ReportObservationsItem) error {
	if err := requiredText(p+"/local_key", v.LocalKey, 16000); err != nil {
		return err
	}
	if err := requiredText(p+"/title", v.Title, 16000); err != nil {
		return err
	}
	if err := requiredText(p+"/text", v.Text, 16000); err != nil {
		return err
	}
	if v.EvidenceIDs == nil {
		return invalid(p+"/evidence_ids", "must be an array")
	}
	for i, value := range v.EvidenceIDs {
		if err := requiredText(p+"/evidence_ids"+fmt.Sprintf("/%d", i), value, 16000); err != nil {
			return err
		}
		if !regexp.MustCompile("^EVD[0-9a-f-]{36}$").MatchString(value) {
			return invalid(p+"/evidence_ids"+fmt.Sprintf("/%d", i), "invalid object ID")
		}
	}
	if !v4Unique(v.EvidenceIDs) {
		return invalid(p+"/evidence_ids", "duplicate values")
	}
	return nil
}
func validateV4UnitSummaryImpactAssessment(p string, v V4UnitSummaryImpactAssessment) error {
	if err := requiredText(p+"/level", v.Level, 16000); err != nil {
		return err
	}
	if !v4OneOf(v.Level, "high", "medium", "low", "pending") {
		return invalid(p+"/level", "unsupported value")
	}
	if err := requiredText(p+"/rationale", v.Rationale, 16000); err != nil {
		return err
	}
	if v.EvidenceIDs == nil {
		return invalid(p+"/evidence_ids", "must be an array")
	}
	for i, value := range v.EvidenceIDs {
		if err := requiredText(p+"/evidence_ids"+fmt.Sprintf("/%d", i), value, 16000); err != nil {
			return err
		}
		if !regexp.MustCompile("^EVD[0-9a-f-]{36}$").MatchString(value) {
			return invalid(p+"/evidence_ids"+fmt.Sprintf("/%d", i), "invalid object ID")
		}
	}
	if !v4Unique(v.EvidenceIDs) {
		return invalid(p+"/evidence_ids", "duplicate values")
	}
	return nil
}
func v4OneOf(s string, options ...string) bool {
	for _, v := range options {
		if s == v {
			return true
		}
	}
	return false
}
func v4Unique(values []string) bool {
	seen := map[string]bool{}
	for _, v := range values {
		if seen[v] {
			return false
		}
		seen[v] = true
	}
	return true
}

func validateNormalizedReport(r V4Report) error {
	if r.SchemaVersion == SignalSchemaVersion {
		return validateSignalReport(r)
	}
	raw, _ := json.Marshal(r)
	var tree any
	_ = json.Unmarshal(raw, &tree)
	if hasSignalFields(tree) {
		return invalid("report", "v5 fields require v5 version")
	}
	return validateNormalizedBaseReport(r)
}
func validateNormalizedBaseReport(r V4Report) error {
	if err := validateV4Report("report", r); err != nil {
		return err
	}
	if r.ReportType.Code != "investment_reasoning" || r.ReportType.Label != "投研推理报告" {
		return invalid("report_type", "invalid report type")
	}
	start, _ := time.Parse(time.RFC3339Nano, r.AnalysisWindow.Start)
	end, _ := time.Parse(time.RFC3339Nano, r.AnalysisWindow.End)
	if !start.Before(end) {
		return invalid("analysis_window", "start must precede end")
	}
	seen := map[string]bool{}
	total := 0
	for _, g := range []struct {
		kind  string
		units []V4Unit
	}{{"geopolitical_stories", r.GeopoliticalStories}, {"macroeconomic_stories", r.MacroeconomicStories}, {"concept_analyses", r.ConceptAnalyses}} {
		sources := map[string]bool{}
		unitKeys := map[string]bool{}
		for _, u := range g.units {
			total++
			if sources[u.SourceID] {
				return invalid(g.kind, "duplicate source")
			}
			sources[u.SourceID] = true
			if err := validateNormalizedUnit(g.kind, u, unitKeys); err != nil {
				return err
			}
		}
	}
	if total == 0 {
		return invalid("report", "at least one unit required")
	}
	for _, o := range r.Observations {
		if err := normalizedKey(o.LocalKey, seen); err != nil {
			return err
		}
	}
	return nil
}
func normalizedKey(key string, seen map[string]bool) error {
	if !localKeyPattern.MatchString(key) || seen[key] {
		return invalid("local_key", "invalid or duplicate local key")
	}
	seen[key] = true
	return nil
}
func validateNormalizedAssessment(a V4Assessment) error { return validateAssessmentState(a, false) }
func validateAssessmentState(a V4Assessment, allowPending bool) error {
	w := a.ForecastWindow
	if w.StartAt != nil && w.EndAt != nil {
		start, _ := time.Parse(time.RFC3339Nano, *w.StartAt)
		end, _ := time.Parse(time.RFC3339Nano, *w.EndAt)
		if !start.Before(end) {
			return invalid("forecast_window", "invalid interval")
		}
	}
	if a.ConclusionBasis == "observation_only" {
		if a.Direction != "pending" || a.ValidationStatus != "insufficient_evidence" || a.Confidence != nil || w.Kind != "not_applicable" || w.StartAt != nil || w.EndAt != nil {
			return invalid("assessment", "inconsistent observation state")
		}
	} else if (a.Direction == "pending" && !allowPending) || a.ValidationStatus != "pending_validation" || a.Confidence == nil || w.Kind == "not_applicable" || len(a.Conditions) == 0 || len(a.FollowUp) == 0 || len(a.EvidenceIDs) == 0 {
		return invalid("assessment", "directional inference requires conditions, follow-up, confidence and Evidence")
	}
	return nil
}
func validateNormalizedObjections(o V4Objections) error {
	if (len(o.Counterevidence) > 0) != (o.CounterevidenceStatus == "identified") {
		return invalid("counterevidence_status", "does not match counterevidence")
	}
	for _, c := range o.Counterevidence {
		if c.Basis != "source_fact" || len(c.EvidenceIDs) == 0 {
			return invalid("counterevidence", "requires sourced counterfact")
		}
	}
	for _, c := range o.Buffers {
		if c.Basis == "source_fact" && len(c.EvidenceIDs) == 0 {
			return invalid("buffers", "source fact requires Evidence")
		}
	}
	return nil
}
func validateNormalizedUnit(kind string, u V4Unit, seen map[string]bool) error {
	if kind == "industry_chain_analyses" {
		kind = "concept_analyses"
	}
	if err := normalizedKey(u.LocalKey, seen); err != nil {
		return err
	}
	if u.Summary.ImpactAssessment.Level != "pending" && len(u.Summary.ImpactAssessment.EvidenceIDs) == 0 {
		return invalid("impact_assessment", "rated impact requires Evidence")
	}
	if kind != "geopolitical_stories" && len(u.Detail.MacroImpacts) > 0 {
		return invalid("macro_impacts", "only geopolitical stories may target macro objects")
	}
	seen = map[string]bool{}
	macros := map[string]bool{}
	chains := map[string]map[string]bool{}
	sources := map[string]bool{}
	for _, m := range u.Detail.MacroImpacts {
		if err := normalizedKey(m.LocalKey, seen); err != nil {
			return err
		}
		macros[m.LocalKey] = true
		if err := validateVersionedAssessment(m.Assessment, m.JudgmentOrigin != ""); err != nil {
			return err
		}
		if err := validateNormalizedObjections(m.Objections); err != nil {
			return err
		}
	}
	seen = map[string]bool{}
	for _, c := range u.Detail.IndustryChains {
		if sources[c.SourceID] {
			return invalid("industry_chains", "duplicate source")
		}
		sources[c.SourceID] = true
		nodes, err := validateNormalizedChain(c, seen)
		if err != nil {
			return err
		}
		chains[c.LocalKey] = nodes
	}
	refs := map[string]bool{}
	for _, a := range u.Summary.AffectedRefs {
		key := a.TargetType + "/" + a.LocalKey
		if a.ChainLocalKey != nil {
			key += "/" + *a.ChainLocalKey
		}
		if refs[key] {
			return invalid("affected_refs", "duplicate reference")
		}
		refs[key] = true
		valid := false
		switch a.TargetType {
		case "macroeconomic_story":
			valid = kind == "geopolitical_stories" && a.ChainLocalKey == nil && macros[a.LocalKey]
		case "industry_chain":
			_, ok := chains[a.LocalKey]
			valid = kind != "concept_analyses" && a.ChainLocalKey == nil && ok
		case "industry_chain_node":
			if a.ChainLocalKey != nil {
				valid = kind == "concept_analyses" && chains[*a.ChainLocalKey][a.LocalKey]
			}
		}
		if !valid {
			return invalid("affected_refs", "reference does not close in its unit")
		}
	}
	return nil
}

// V4 scopes use the exact typed snapshot traversal, shared with token projection.
type NormalizedEvidenceScope struct {
	Path string
	IDs  []string
}

func NormalizedEvidenceScopes(r V4Report) []NormalizedEvidenceScope {
	if r.SchemaVersion == SignalSchemaVersion {
		return signalEvidenceScopes(r)
	}
	scopes := []NormalizedEvidenceScope{}
	add := func(p string, ids []string) {
		scopes = append(scopes, NormalizedEvidenceScope{p + "/evidence_ids", ids})
	}
	objections := func(p string, o V4Objections) {
		for i, c := range o.Counterevidence {
			add(fmt.Sprintf("%s/counterevidence/%d", p, i), c.EvidenceIDs)
		}
		for i, c := range o.Buffers {
			add(fmt.Sprintf("%s/buffers/%d", p, i), c.EvidenceIDs)
		}
	}
	for _, g := range []struct {
		kind  string
		units []V4Unit
	}{{"geopolitical_stories", r.GeopoliticalStories}, {"macroeconomic_stories", r.MacroeconomicStories}, {"concept_analyses", r.ConceptAnalyses}} {
		for _, u := range g.units {
			p := g.kind + "/" + u.LocalKey
			add(p+"/summary", u.Summary.EvidenceIDs)
			add(p+"/summary/impact_assessment", u.Summary.ImpactAssessment.EvidenceIDs)
			for _, m := range u.Detail.MacroImpacts {
				mp := p + "/detail/macro_impacts/" + m.LocalKey
				add(mp+"/assessment", m.Assessment.EvidenceIDs)
				objections(mp+"/objections", m.Objections)
			}
			for _, c := range u.Detail.IndustryChains {
				cp := p + "/detail/industry_chains/" + c.LocalKey
				add(cp+"/assessment", c.Assessment.EvidenceIDs)
				add(cp+"/reasoning_summary/support", c.ReasoningSummary.Support.EvidenceIDs)
				objections(cp+"/reasoning_summary/objections", c.ReasoningSummary.Objections)
				for _, n := range c.AffectedNodes {
					np := cp + "/affected_nodes/" + n.LocalKey
					add(np+"/assessment", n.Assessment.EvidenceIDs)
					objections(np+"/objections", n.Objections)
				}
			}
		}
	}
	for _, o := range r.Observations {
		add("observations/"+o.LocalKey, o.EvidenceIDs)
	}
	return scopes
}
func normalizedEvidenceLinks(id string, r V4Report) ([]EvidenceLink, error) {
	links := []EvidenceLink{}
	for _, scope := range NormalizedEvidenceScopes(r) {
		for i, ev := range scope.IDs {
			key, err := coreid.New(coreid.ReportEvidenceLink)
			if err != nil {
				return nil, err
			}
			links = append(links, EvidenceLink{ID: key, ReportID: id, EvidenceID: ev, ScopeType: ScopeType("normalized_report_evidence"), ScopePath: scope.Path, Position: i + 1})
		}
	}
	return links, nil
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
func (v ChainAnalysisDetail) MarshalJSON() ([]byte, error) {
	if v.V4 != nil {
		return json.Marshal(v.V4)
	}
	type plain ChainAnalysisDetail
	return json.Marshal(plain(v))
}
func (v Home) MarshalJSON() ([]byte, error) {
	if v.V4 != nil {
		return json.Marshal(v.V4)
	}
	type plain Home
	return json.Marshal(plain(v))
}

// ValidateNormalizedUnit guards selected immutable JSONB units at the read boundary.
func ValidateNormalizedUnit(kind string, u V4Unit) error {
	if err := validateV4Unit("unit", u); err != nil {
		return err
	}
	return validateNormalizedUnit(kind, u, map[string]bool{})
}

func validateNormalizedChain(c V4Chain, seen map[string]bool) (map[string]bool, error) {
	if err := normalizedKey(c.LocalKey, seen); err != nil {
		return nil, err
	}
	if err := validateVersionedAssessment(c.Assessment, c.JudgmentOrigin != ""); err != nil {
		return nil, err
	}
	if err := validateNormalizedObjections(c.ReasoningSummary.Objections); err != nil {
		return nil, err
	}
	if c.ReasoningSummary.Logic != c.Assessment.TransmissionLogic {
		return nil, invalid("reasoning_summary.logic", "must match chain assessment")
	}
	if c.ReasoningSummary.Support.Basis == "source_fact" && len(c.ReasoningSummary.Support.EvidenceIDs) == 0 {
		return nil, invalid("support", "source fact requires Evidence")
	}
	if (c.EmptyState != nil) != (len(c.AffectedNodes) == 0) || (c.EmptyState != nil) != (c.Assessment.ConclusionBasis == "observation_only") {
		return nil, invalid("empty_state", "does not match node results")
	}
	if c.EmptyState != nil && len(c.EmptyState.FollowUp) == 0 {
		return nil, invalid("empty_state", "requires follow-up")
	}
	graphKeys := map[string]bool{}
	nodeKeys := map[string]bool{}
	graph := map[string]V4GraphNodesItem{}
	graphIDs := map[string]bool{}
	for _, n := range c.Graph.Nodes {
		if err := normalizedKey(n.LocalKey, graphKeys); err != nil {
			return nil, err
		}
		if graphIDs[n.SourceID] {
			return nil, invalid("graph", "duplicate source node")
		}
		graphIDs[n.SourceID] = true
		graph[n.LocalKey] = n
	}
	edges := map[string]bool{}
	for _, e := range c.Graph.Edges {
		_, a := graph[e.FromNodeLocalKey]
		_, b := graph[e.ToNodeLocalKey]
		key := e.FromNodeLocalKey + "/" + e.ToNodeLocalKey + "/" + e.RelationLabel
		if !a || !b || e.FromNodeLocalKey == e.ToNodeLocalKey || edges[key] {
			return nil, invalid("graph.edges", "invalid or duplicate edge")
		}
		edges[key] = true
	}
	nodes := map[string]bool{}
	targets := map[string]bool{}
	for _, n := range c.AffectedNodes {
		if err := normalizedKey(n.LocalKey, nodeKeys); err != nil {
			return nil, err
		}
		g, ok := graph[n.NodeLocalKey]
		if !ok || g.SourceID != n.SourceID || g.Name != n.Name || targets[n.NodeLocalKey] {
			return nil, invalid("affected_nodes", "node identity mismatch or duplicate")
		}
		targets[n.NodeLocalKey] = true
		nodes[n.LocalKey] = true
		if err := validateVersionedAssessment(n.Assessment, n.JudgmentOrigin != ""); err != nil {
			return nil, err
		}
		if err := validateNormalizedObjections(n.Objections); err != nil {
			return nil, err
		}
		if n.Assessment.ConclusionBasis != "reasoning_hypothesis" {
			return nil, invalid("affected_nodes", "observation belongs in empty_state")
		}
	}
	return nodes, nil
}
func ValidateNormalizedChain(c V4Chain) error {
	if err := validateV4Chain("chain", c); err != nil {
		return err
	}
	_, err := validateNormalizedChain(c, map[string]bool{})
	return err
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

func hasSignalFields(v any) bool {
	switch x := v.(type) {
	case map[string]any:
		if _, ok := x["nodes"]; ok {
			if _, has := x["scope"]; has {
				return true
			}
		}
		for k, z := range x {
			if v4OneOf(k, "variable_signals", "reasoning_sources", "judgment_origin", "industry_chain_analyses", "company_analyses", "companies") {
				return true
			}
			if hasSignalFields(z) {
				return true
			}
		}
	case []any:
		for _, z := range x {
			if hasSignalFields(z) {
				return true
			}
		}
	}
	return false
}
func validateVersionedAssessment(a V4Assessment, signalVersion bool) error {
	if signalVersion && a.ConclusionBasis != "reasoning_hypothesis" {
		return invalid("assessment", "v5 anchors require a bounded judgment")
	}
	// A bounded v5 judgment may explicitly leave direction pending; all other inference requirements remain mandatory.
	return validateAssessmentState(a, signalVersion)
}

type signalJudgment struct {
	key, id, origin string
	sources         *V5ReasoningSources
	signals         *[]V5Signal
}

func validateSignalJudgments(values []signalJudgment) error {
	objects := map[string]string{}
	for _, v := range values {
		if err := normalizedKey(v.key, mapKeys(objects)); err != nil {
			return err
		}
		objects[v.key] = v.id
	}
	for _, v := range values {
		if v.signals == nil || *v.signals == nil || v.sources == nil {
			return invalid(v.key, "signal arrays and reasoning sources are required")
		}
		src := v.sources
		if src.SignalIDs == nil || src.EventIDs == nil || src.UpstreamRefs == nil || !v4Unique(src.SignalIDs) || !v4Unique(src.EventIDs) {
			return invalid(v.key, "invalid reasoning source arrays")
		}
		if !v4OneOf(v.origin, "direct", "inferred") || (v.origin == "direct") != (len(*v.signals) > 0) {
			return invalid(v.key, "direct judgment must have own signals; inferred judgment must not")
		}
		if len(src.EventIDs) == 0 && len(src.UpstreamRefs) == 0 {
			return invalid(v.key, "judgment requires Event or upstream provenance")
		}
		if len(src.SignalIDs) != len(*v.signals) {
			return invalid(v.key, "signal provenance does not match owned signals")
		}
		for _, id := range src.EventIDs {
			if !validSignalID(id, "EVT") {
				return invalid(v.key, "invalid Event ID")
			}
		}
		for i, row := range *v.signals {
			if row.SignalID != src.SignalIDs[i] || !validSignalID(row.SignalID, "") || !validSignalID(row.VariableID, "") {
				return invalid(v.key, "invalid variable or signal identity")
			}
			if !v4OneOf(row.SourceDirection, "UP", "DOWN", "STABLE", "MIXED", "UNKNOWN") || !v4OneOf(row.Adoption, "adopted", "qualified") {
				return invalid(v.key, "invalid signal direction or adoption")
			}
			for _, t := range []string{row.VariableName, row.Signal, row.Qualification} {
				if err := requiredText(v.key, t, 16000); err != nil {
					return err
				}
			}
			if len(row.EventIDs) == 0 || len(row.EvidenceIDs) == 0 || !v4Unique(row.EventIDs) || !v4Unique(row.EvidenceIDs) {
				return invalid(v.key, "signals require unique Event and Evidence references")
			}
			for _, id := range row.EventIDs {
				if !validSignalID(id, "EVT") || !v4OneOf(id, src.EventIDs...) {
					return invalid(v.key, "signal Event missing from judgment provenance")
				}
			}
			for _, id := range row.EvidenceIDs {
				if !validSignalID(id, "EVD") {
					return invalid(v.key, "invalid Evidence ID")
				}
			}
		}
		refs := map[string]bool{}
		for _, ref := range src.UpstreamRefs {
			targetID, exists := objects[ref.LocalKey]
			if !exists || targetID != ref.EntityID || ref.LocalKey == v.key || refs[ref.LocalKey] {
				return invalid(v.key, "upstream reference does not close in judgment scope")
			}
			refs[ref.LocalKey] = true
			if ref.Mechanism != nil {
				if err := requiredText(v.key, *ref.Mechanism, 16000); err != nil {
					return err
				}
			}
			if ref.Condition != nil {
				if err := requiredText(v.key, *ref.Condition, 16000); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
func mapKeys(m map[string]string) map[string]bool {
	r := map[string]bool{}
	for k := range m {
		r[k] = true
	}
	return r
}
func validSignalID(id, prefix string) bool {
	return regexp.MustCompile("^" + prefix + "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$").MatchString(id)
}
func ValidateSignalCompany(c V4Macro) error {
	if !validSignalID(c.SourceID, "COM") {
		return invalid(c.LocalKey, "invalid Company ID")
	}
	if err := requiredText(c.LocalKey, c.Name, 16000); err != nil {
		return err
	}
	if err := validateV4Assessment(c.LocalKey, c.Assessment); err != nil {
		return err
	}
	if err := validateVersionedAssessment(c.Assessment, true); err != nil {
		return err
	}
	if err := validateV4Objections(c.LocalKey, c.Objections); err != nil {
		return err
	}
	return validateNormalizedObjections(c.Objections)
}
func ValidateSignalUnit(kind string, u V4Unit) error {
	if err := ValidateNormalizedUnit(kind, u); err != nil {
		return err
	}
	prefix := map[string]string{"geopolitical_stories": "GPR", "macroeconomic_stories": "MEC", "concept_analyses": "CON", "industry_chain_analyses": "ICH"}[kind]
	if prefix == "" || !validSignalID(u.SourceID, prefix) {
		return invalid(u.LocalKey, "unit source ID does not match kind")
	}
	if u.Detail.Companies == nil || *u.Detail.Companies == nil {
		return invalid(u.LocalKey, "companies must be an array")
	}
	values := []signalJudgment{{u.LocalKey, u.SourceID, u.JudgmentOrigin, u.ReasoningSources, u.Detail.VariableSignals}}
	for _, m := range u.Detail.MacroImpacts {
		values = append(values, signalJudgment{m.LocalKey, m.SourceID, m.JudgmentOrigin, m.ReasoningSources, m.VariableSignals})
	}
	for _, c := range *u.Detail.Companies {
		if err := ValidateSignalCompany(c); err != nil {
			return err
		}
		values = append(values, signalJudgment{c.LocalKey, c.SourceID, c.JudgmentOrigin, c.ReasoningSources, c.VariableSignals})
	}
	for _, c := range u.Detail.IndustryChains {
		if c.Graph.Scope != "assessed_nodes_only" || c.EmptyState != nil || len(c.Graph.Nodes) != len(c.AffectedNodes) {
			return invalid(c.LocalKey, "v5 graph must contain exactly assessed nodes")
		}
		values = append(values, signalJudgment{c.LocalKey, c.SourceID, c.JudgmentOrigin, c.ReasoningSources, c.VariableSignals})
		for _, n := range c.AffectedNodes {
			values = append(values, signalJudgment{n.LocalKey, n.SourceID, n.JudgmentOrigin, n.ReasoningSources, n.VariableSignals})
		}
	}
	return validateSignalJudgments(values)
}
func validateSignalReport(r V4Report) error {
	if r.IndustryChainAnalyses == nil || *r.IndustryChainAnalyses == nil || r.CompanyAnalyses == nil || *r.CompanyAnalyses == nil {
		return invalid("report", "v5 collections must be arrays")
	}
	// Reuse normalized structural and assessment rules without rewriting the persisted snapshot.
	base := r
	base.SchemaVersion = NormalizedSchemaVersion
	base.ConceptAnalyses = append(append([]V4Unit{}, r.ConceptAnalyses...), (*r.IndustryChainAnalyses)...)
	if len(base.GeopoliticalStories)+len(base.MacroeconomicStories)+len(base.ConceptAnalyses) > 0 {
		if err := validateNormalizedBaseReport(base); err != nil {
			return err
		}
	} else {
		if err := validateV4Report("report", base); err != nil {
			return err
		}
		if len(*r.CompanyAnalyses) == 0 {
			return invalid("report", "at least one judgment required")
		}
	}
	if r.ReportType.Code != "investment_reasoning" || r.ReportType.Label != "投研推理报告" {
		return invalid("report_type", "invalid report type")
	}
	start, _ := time.Parse(time.RFC3339Nano, r.AnalysisWindow.Start)
	end, _ := time.Parse(time.RFC3339Nano, r.AnalysisWindow.End)
	if !start.Before(end) {
		return invalid("analysis_window", "start must precede end")
	}
	observationKeys := map[string]bool{}
	for _, o := range r.Observations {
		if err := normalizedKey(o.LocalKey, observationKeys); err != nil {
			return err
		}
	}
	companyKeys := map[string]bool{}
	companySources := map[string]bool{}
	for _, c := range *r.CompanyAnalyses {
		if err := normalizedKey(c.LocalKey, companyKeys); err != nil {
			return err
		}
		if companySources[c.SourceID] {
			return invalid("company_analyses", "duplicate company source")
		}
		companySources[c.SourceID] = true
	}
	owners := map[string]string{}
	for _, g := range []struct {
		kind  string
		units []V4Unit
	}{{"geopolitical_stories", r.GeopoliticalStories}, {"macroeconomic_stories", r.MacroeconomicStories}, {"concept_analyses", r.ConceptAnalyses}, {"industry_chain_analyses", *r.IndustryChainAnalyses}} {
		for _, u := range g.units {
			if err := ValidateSignalUnit(g.kind, u); err != nil {
				return err
			}
		}
	}
	for _, c := range *r.CompanyAnalyses {
		if err := ValidateSignalCompany(c); err != nil {
			return err
		}
		if err := validateSignalJudgments([]signalJudgment{{c.LocalKey, c.SourceID, c.JudgmentOrigin, c.ReasoningSources, c.VariableSignals}}); err != nil {
			return err
		}
	}
	// Signal ownership is stable across repeated entity appearances within the report.
	var walk func(any) error
	walk = func(v any) error {
		switch x := v.(type) {
		case map[string]any:
			if rows, ok := x["variable_signals"].([]any); ok {
				id, _ := x["source_id"].(string)
				if id != "" {
					for _, row := range rows {
						sid := row.(map[string]any)["signal_id"].(string)
						if prior := owners[sid]; prior != "" && prior != id {
							return invalid("variable_signals", "signal assigned to multiple entities")
						}
						owners[sid] = id
					}
				}
			}
			// Story signals belong to their containing unit.
			if detail, ok := x["detail"].(map[string]any); ok {
				detail["source_id"] = x["source_id"]
			}
			for _, z := range x {
				if err := walk(z); err != nil {
					return err
				}
			}
		case []any:
			for _, z := range x {
				if err := walk(z); err != nil {
					return err
				}
			}
		}
		return nil
	}
	raw, _ := json.Marshal(r)
	var tree any
	_ = json.Unmarshal(raw, &tree)
	return walk(tree)
}
func signalEvidenceScopes(r V4Report) []NormalizedEvidenceScope {
	// Exact object paths use local keys; signal and claim arrays use their frozen ordinal.
	raw, _ := json.Marshal(r)
	var tree any
	_ = json.Unmarshal(raw, &tree)
	scopes := []NormalizedEvidenceScope{}
	var walk func(any, string)
	walk = func(v any, p string) {
		switch x := v.(type) {
		case map[string]any:
			if raw, ok := x["evidence_ids"].([]any); ok {
				ids := []string{}
				for _, id := range raw {
					ids = append(ids, id.(string))
				}
				scopes = append(scopes, NormalizedEvidenceScope{strings.TrimPrefix(p+"/evidence_ids", "/"), ids})
			}
			keys := []string{}
			for k := range x {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				walk(x[k], p+"/"+k)
			}
		case []any:
			for i, z := range x {
				key := fmt.Sprint(i)
				if o, ok := z.(map[string]any); ok {
					if k, ok := o["local_key"].(string); ok {
						key = k
					}
				}
				walk(z, p+"/"+key)
			}
		}
	}
	walk(tree, "")
	return scopes
}

func ValidateStandaloneSignalCompany(c V4Macro) error {
	if err := ValidateSignalCompany(c); err != nil {
		return err
	}
	return validateSignalJudgments([]signalJudgment{{c.LocalKey, c.SourceID, c.JudgmentOrigin, c.ReasoningSources, c.VariableSignals}})
}

// EvidenceTag is a deterministic reading projection of existing semantic content.
type EvidenceTag struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

// Report readers consume four semantic dimensions without the full Evidence JSON.
type EvidenceSemanticProjection struct {
	Actors  []string                   `json:"actors"`
	Action  string                     `json:"action"`
	Objects []string                   `json:"objects"`
	Metrics []EvidenceMetricProjection `json:"metrics"`
}
type EvidenceMetricProjection struct {
	Name   string  `json:"name"`
	Value  *string `json:"value"`
	Unit   *string `json:"unit"`
	Change *string `json:"change"`
	Period *string `json:"period"`
}

func ProjectEvidenceTags(value EvidenceSemanticProjection) []EvidenceTag {
	tags := make([]EvidenceTag, 0)
	appendTag := func(kind, text string) {
		if strings.TrimSpace(text) != "" {
			tags = append(tags, EvidenceTag{Kind: kind, Text: text})
		}
	}
	for _, text := range value.Actors {
		appendTag("actor", text)
	}
	appendTag("action", value.Action)
	for _, text := range value.Objects {
		appendTag("object", text)
	}
	for _, metric := range value.Metrics {
		parts := []string{}
		for _, text := range []*string{&metric.Name, metric.Value, metric.Unit, metric.Change, metric.Period} {
			if text != nil && strings.TrimSpace(*text) != "" {
				parts = append(parts, *text)
			}
		}
		appendTag("metric", strings.Join(parts, " · "))
	}
	return tags
}

// AnalysisIdentity is the stable, Data-owned identity shared by one summary/detail pair.
// Source IDs remain published snapshot references, never foreign keys to live graph objects.
func AnalysisIdentity(reportID, kind, localKey string) (string, error) {
	return coreid.Derive(coreid.ReportAnalysis, "report-analysis", reportID, kind, localKey)
}
