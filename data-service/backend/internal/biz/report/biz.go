package report

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
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
	ResultStable             = "stable"
	ResultMixed              = "mixed"
	ResultPending            = "pending"
	BasisDirectEvidence      = "direct_evidence"
	BasisReasoningHypothesis = "reasoning_hypothesis"
	ValidationPending        = "pending_validation"
	TransmissionCrossLayer   = "cross_layer_reasoning"
	TransmissionSameSource   = "same_source_signal"
)

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
	GeneratedAt    time.Time       `json:"generated_at"`
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
	Report         Summary
	Geopolitics    *LayerSnapshot
	Macroeconomics *LayerSnapshot
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
	Ordinal            int                          `json:"-"`
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
			return &ReferenceError{Path: "report.evidence_ids", Reference: missing, Message: "does not identify an existing Atomic Evidence"}
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

func ValidateReport(report Report) error {
	if report.GeneratedAt.IsZero() {
		return invalid("report.generated_at", "must be a timestamp")
	}
	if len(report.IndustryChains) == 0 {
		return invalid("report.industry_chains", "must contain at least one industry-chain analysis")
	}
	index := reportIndex{sections: map[string]struct{}{}, anchors: map[string]struct{}{}, chains: map[string]struct{}{}, nodes: map[string]struct{}{}, allKeys: map[string]struct{}{}}
	if report.Geopolitics != nil {
		index.sections["geopolitics"] = struct{}{}
		if err := validateLayer("report.geopolitics", *report.Geopolitics, &index); err != nil {
			return err
		}
	}
	if report.Macroeconomics != nil {
		index.sections["macroeconomics"] = struct{}{}
		if err := validateLayer("report.macroeconomics", *report.Macroeconomics, &index); err != nil {
			return err
		}
	}
	for i, chain := range report.IndustryChains {
		if err := validateIndustryChain(fmt.Sprintf("report.industry_chains[%d]", i), chain, &index); err != nil {
			return err
		}
	}
	return validateTransmissionTargets(report, index)
}

type reportIndex struct {
	sections map[string]struct{}
	anchors  map[string]struct{}
	chains   map[string]struct{}
	nodes    map[string]struct{}
	allKeys  map[string]struct{}
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

func validateLayer(path string, layer Layer, index *reportIndex) error {
	if err := requiredText(path+".title", layer.Title, 500); err != nil {
		return err
	}
	if err := validateLayerSummary(path+".summary", layer.Summary); err != nil {
		return err
	}
	if layer.Detail.AffectedAnchors == nil || layer.Detail.ReasoningSteps == nil {
		return invalid(path+".detail", "collections must be arrays")
	}
	for i, anchor := range layer.Detail.AffectedAnchors {
		itemPath := fmt.Sprintf("%s.detail.affected_anchors[%d]", path, i)
		if err := index.add(itemPath+".local_key", anchor.LocalKey); err != nil {
			return err
		}
		index.anchors[anchor.LocalKey] = struct{}{}
		if err := validateAssessment(itemPath, "current_state", anchor.Name, anchor.CurrentState, anchor.TransmissionLogic, anchor.Result, anchor.ConclusionBasis, anchor.ValidationStatus, anchor.TimeWindow, anchor.Confidence, anchor.EvidenceIDs); err != nil {
			return err
		}
	}
	for i, step := range layer.Detail.ReasoningSteps {
		itemPath := fmt.Sprintf("%s.detail.reasoning_steps[%d]", path, i)
		for name, value := range map[string]string{"input": step.Input, "mechanism": step.Mechanism, "output": step.Output} {
			if err := requiredText(itemPath+"."+name, value, 10_000); err != nil {
				return err
			}
		}
		if err := validateOpenCodedLabel(itemPath+".reasoning_type", step.ReasoningType); err != nil {
			return err
		}
		if err := validateConfidence(itemPath+".confidence", step.Confidence); err != nil {
			return err
		}
		if err := validateEvidenceIDs(itemPath+".evidence_ids", step.EvidenceIDs); err != nil {
			return err
		}
	}
	for i, transmission := range layer.Summary.DownwardTransmission {
		itemPath := fmt.Sprintf("%s.summary.downward_transmission[%d]", path, i)
		if err := index.add(itemPath+".local_key", transmission.LocalKey); err != nil {
			return err
		}
		if err := requiredText(itemPath+".source_conclusion", transmission.SourceConclusion, 10_000); err != nil {
			return err
		}
		if err := requiredText(itemPath+".transmission_logic", transmission.TransmissionLogic, 10_000); err != nil {
			return err
		}
		if len(transmission.Targets) == 0 {
			return invalid(itemPath+".targets", "must contain at least one target")
		}
		if err := validateMappedLabel(itemPath+".transmission_kind", transmission.TransmissionKind, map[string]string{TransmissionCrossLayer: "跨层推理", TransmissionSameSource: "同源信号"}); err != nil {
			return err
		}
		if err := validateConfidence(itemPath+".confidence", transmission.Confidence); err != nil {
			return err
		}
		if err := validateOpenCodedLabel(itemPath+".status", transmission.Status); err != nil {
			return err
		}
		for j, target := range transmission.Targets {
			targetPath := fmt.Sprintf("%s.targets[%d]", itemPath, j)
			if !validTargetType(target.Type) || !localKeyPattern.MatchString(target.LocalKey) {
				return invalid(targetPath, "must identify a supported Report-local target")
			}
			if err := requiredText(targetPath+".name", target.Name, 500); err != nil {
				return err
			}
			if err := validateResult(targetPath+".result", target.Result); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateLayerSummary(path string, summary LayerSummary) error {
	if err := requiredText(path+".conclusion", summary.Conclusion, 10_000); err != nil {
		return err
	}
	if err := validateResult(path+".result", summary.Result); err != nil {
		return err
	}
	if err := validateConfidence(path+".confidence", summary.Confidence); err != nil {
		return err
	}
	if err := validateTimeWindow(path+".time_window", summary.TimeWindow); err != nil {
		return err
	}
	if summary.DownwardTransmission == nil {
		return invalid(path+".downward_transmission", "must be an array")
	}
	for name, value := range map[string]*string{"counterevidence": summary.Uncertainty.Counterevidence, "evidence_gap": summary.Uncertainty.EvidenceGap, "boundary": summary.Uncertainty.Boundary, "reversal_condition": summary.Uncertainty.ReversalCondition} {
		if err := optionalText(path+".uncertainty."+name, value, 10_000); err != nil {
			return err
		}
	}
	return validateEvidenceIDs(path+".evidence_ids", summary.EvidenceIDs)
}

func validateIndustryChain(path string, chain IndustryChain, index *reportIndex) error {
	if err := index.add(path+".local_key", chain.LocalKey); err != nil {
		return err
	}
	index.chains[chain.LocalKey] = struct{}{}
	if err := requiredText(path+".name", chain.Name, 500); err != nil {
		return err
	}
	summary := chain.Summary
	for name, value := range map[string]string{"conclusion": summary.Conclusion, "status": summary.Status, "path": summary.Path, "counterevidence_and_gap": summary.CounterevidenceAndGap, "stop_condition": summary.StopCondition} {
		if err := requiredText(path+".summary."+name, value, 10_000); err != nil {
			return err
		}
	}
	if err := validateResult(path+".summary.result", summary.Result); err != nil {
		return err
	}
	if err := validateConfidence(path+".summary.confidence", summary.Confidence); err != nil {
		return err
	}
	if err := validateTimeWindow(path+".summary.time_window", summary.TimeWindow); err != nil {
		return err
	}
	if err := validateEvidenceIDs(path+".summary.evidence_ids", summary.EvidenceIDs); err != nil {
		return err
	}
	if len(summary.Graph.Nodes) == 0 || summary.Graph.Edges == nil || len(chain.Detail.AffectedNodes) == 0 {
		return invalid(path, "graph nodes and affected nodes must be non-empty; edges must be an array")
	}
	topology := map[string]string{}
	for i, node := range summary.Graph.Nodes {
		nodePath := fmt.Sprintf("%s.summary.graph.nodes[%d]", path, i)
		if !localKeyPattern.MatchString(node.LocalKey) {
			return invalid(nodePath+".local_key", "must be a Report-local graph node key")
		}
		if _, exists := topology[node.LocalKey]; exists {
			return invalid(nodePath+".local_key", "duplicates a graph node")
		}
		if err := requiredText(nodePath+".name", node.Name, 500); err != nil {
			return err
		}
		topology[node.LocalKey] = node.Name
	}
	for i, edge := range summary.Graph.Edges {
		edgePath := fmt.Sprintf("%s.summary.graph.edges[%d]", path, i)
		if _, exists := topology[edge.FromNodeKey]; !exists {
			return invalid(edgePath+".from_node_key", "must reference this chain graph")
		}
		if _, exists := topology[edge.ToNodeKey]; !exists {
			return invalid(edgePath+".to_node_key", "must reference this chain graph")
		}
		if edge.FromNodeKey == edge.ToNodeKey {
			return invalid(edgePath, "must not be a self edge")
		}
		if err := validateOpenCodedLabel(edgePath+".relation", edge.Relation); err != nil {
			return err
		}
	}
	seenNodes := map[string]struct{}{}
	for i, node := range chain.Detail.AffectedNodes {
		nodePath := fmt.Sprintf("%s.detail.affected_nodes[%d]", path, i)
		if err := index.add(nodePath+".local_key", node.LocalKey); err != nil {
			return err
		}
		index.nodes[node.LocalKey] = struct{}{}
		if _, exists := topology[node.NodeLocalKey]; !exists {
			return invalid(nodePath+".node_local_key", "must reference this chain graph")
		}
		if _, exists := seenNodes[node.NodeLocalKey]; exists {
			return invalid(nodePath+".node_local_key", "duplicates an affected graph node")
		}
		seenNodes[node.NodeLocalKey] = struct{}{}
		if err := validateAssessment(nodePath, "impact", node.Name, node.Impact, node.TransmissionLogic, node.Result, node.ConclusionBasis, node.ValidationStatus, node.TimeWindow, node.Confidence, node.EvidenceIDs); err != nil {
			return err
		}
	}
	return nil
}

func validateAssessment(path, stateField, name, state, logic string, result CodedLabel, basis, validation *CodedLabel, window TimeWindow, confidence Confidence, evidenceIDs []string) error {
	for field, value := range map[string]string{"name": name, stateField: state, "transmission_logic": logic} {
		if err := requiredText(path+"."+field, value, 10_000); err != nil {
			return err
		}
	}
	if err := validateResult(path+".result", result); err != nil {
		return err
	}
	if err := validateOptionalMappedLabel(path+".conclusion_basis", basis, map[string]string{BasisDirectEvidence: "直接证据", BasisReasoningHypothesis: "推理假设"}); err != nil {
		return err
	}
	if err := validateOptionalMappedLabel(path+".validation_status", validation, map[string]string{ValidationPending: "待验证"}); err != nil {
		return err
	}
	if err := validateTimeWindow(path+".time_window", window); err != nil {
		return err
	}
	if err := validateConfidence(path+".confidence", confidence); err != nil {
		return err
	}
	if err := validateEvidenceIDs(path+".evidence_ids", evidenceIDs); err != nil {
		return err
	}
	if basis != nil && basis.Code == BasisDirectEvidence && len(evidenceIDs) == 0 {
		return invalid(path+".evidence_ids", "direct evidence requires at least one Evidence ID")
	}
	if (basis == nil || basis.Code != BasisDirectEvidence) && len(evidenceIDs) != 0 {
		return invalid(path+".evidence_ids", "only direct-evidence assessments may expose Evidence")
	}
	return nil
}

func validateTransmissionTargets(report Report, index reportIndex) error {
	for sectionName, layer := range map[string]*Layer{"geopolitics": report.Geopolitics, "macroeconomics": report.Macroeconomics} {
		if layer == nil {
			continue
		}
		for i, transmission := range layer.Summary.DownwardTransmission {
			for j, target := range transmission.Targets {
				exists := false
				switch target.Type {
				case "section":
					_, exists = index.sections[target.LocalKey]
				case "anchor":
					_, exists = index.anchors[target.LocalKey]
				case "industry_chain":
					_, exists = index.chains[target.LocalKey]
				case "industry_chain_node":
					_, exists = index.nodes[target.LocalKey]
				}
				if !exists {
					return &ReferenceError{Path: fmt.Sprintf("report.%s.summary.downward_transmission[%d].targets[%d]", sectionName, i, j), Reference: target.Type + ":" + target.LocalKey, Message: "does not identify a Report-local target"}
				}
			}
		}
	}
	return nil
}

func validTargetType(value string) bool {
	return value == "section" || value == "anchor" || value == "industry_chain" || value == "industry_chain_node"
}

func validateResult(path string, value CodedLabel) error {
	return validateMappedLabel(path, value, map[string]string{
		ResultWarming: "升温", ResultCooling: "降温", ResultDiverging: "分化",
		ResultStable: "稳定", ResultMixed: "混合", ResultPending: "待验证",
	})
}

func validateConfidence(path string, value Confidence) error {
	if err := validateMappedLabel(path, CodedLabel{Code: value.Code, Label: value.Label}, map[string]string{
		"high": "高", "medium_high": "中–高", "medium": "中", "low_medium": "低–中", "low": "低",
	}); err != nil {
		return err
	}
	if value.Score != nil && (math.IsNaN(*value.Score) || math.IsInf(*value.Score, 0) || *value.Score < 0 || *value.Score > 1) {
		return invalid(path+".score", "must be null or between 0 and 1")
	}
	return nil
}

func validateTimeWindow(path string, value TimeWindow) error {
	if err := requiredText(path+".code", value.Code, 100); err != nil {
		return err
	}
	return requiredText(path+".label", value.Label, 500)
}

func validateOptionalMappedLabel(path string, value *CodedLabel, values map[string]string) error {
	if value == nil {
		return nil
	}
	return validateMappedLabel(path, *value, values)
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

func validateOpenCodedLabel(path string, value CodedLabel) error {
	if err := requiredText(path+".code", value.Code, 100); err != nil {
		return err
	}
	return requiredText(path+".label", value.Label, 500)
}

func validateEvidenceIDs(path string, values []string) error {
	if values == nil {
		return invalid(path, "must be an array")
	}
	seen := map[string]struct{}{}
	for i, id := range values {
		if !coreid.Is(id, coreid.Evidence) {
			return &ReferenceError{Path: fmt.Sprintf("%s[%d]", path, i), Reference: id, Message: "must be a canonical Atomic Evidence ID"}
		}
		if _, duplicate := seen[id]; duplicate {
			return invalid(fmt.Sprintf("%s[%d]", path, i), "duplicates an Evidence in this scope")
		}
		seen[id] = struct{}{}
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
	type scopedIDs struct {
		typeName ScopeType
		path     string
		ids      []string
	}
	values := make([]scopedIDs, 0)
	for sectionName, layer := range map[string]*Layer{"geopolitics": report.Geopolitics, "macroeconomics": report.Macroeconomics} {
		if layer == nil {
			continue
		}
		values = append(values, scopedIDs{ScopeSectionSummary, sectionName + "/summary", layer.Summary.EvidenceIDs})
		for _, anchor := range layer.Detail.AffectedAnchors {
			values = append(values, scopedIDs{ScopeAnchor, sectionName + "/detail/affected_anchors/" + anchor.LocalKey, anchor.EvidenceIDs})
		}
		for i, step := range layer.Detail.ReasoningSteps {
			values = append(values, scopedIDs{ScopeReasoningStep, fmt.Sprintf("%s/detail/reasoning_steps/%d", sectionName, i+1), step.EvidenceIDs})
		}
	}
	for _, chain := range report.IndustryChains {
		prefix := "industry_chains/" + chain.LocalKey
		values = append(values, scopedIDs{ScopeIndustryChainSummary, prefix + "/summary", chain.Summary.EvidenceIDs})
		for _, node := range chain.Detail.AffectedNodes {
			values = append(values, scopedIDs{ScopeIndustryChainNode, prefix + "/detail/affected_nodes/" + node.LocalKey, node.EvidenceIDs})
		}
	}
	links := make([]EvidenceLink, 0)
	for _, value := range values {
		for position, evidenceID := range value.ids {
			linkID, err := coreid.New(coreid.ReportEvidenceLink)
			if err != nil {
				return nil, fmt.Errorf("generate Report Evidence Link ID: %w", err)
			}
			links = append(links, EvidenceLink{ID: linkID, ReportID: reportID, EvidenceID: evidenceID, ScopeType: value.typeName, ScopePath: value.path, Position: position + 1})
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
