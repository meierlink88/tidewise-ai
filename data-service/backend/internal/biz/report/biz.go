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
	ContractVersion = "report-publication.v2"
	DefaultLimit    = 20
	MaxLimit        = 100
)

type ScopeType string

const (
	ScopeSectionSummary       ScopeType = "section_summary"
	ScopeAnchor               ScopeType = "anchor"
	ScopeReasoningStep        ScopeType = "reasoning_step"
	ScopeTransmission         ScopeType = "transmission"
	ScopeIndustryChainSummary ScopeType = "industry_chain_summary"
	ScopeIndustryChainNode    ScopeType = "industry_chain_node"
)

type ResultCode string

const (
	ResultWarming   ResultCode = "warming"
	ResultCooling   ResultCode = "cooling"
	ResultDiverging ResultCode = "diverging"
	ResultStable    ResultCode = "stable"
	ResultMixed     ResultCode = "mixed"
	ResultPending   ResultCode = "pending"
)

type NatureCode string

const (
	NatureDirectEvidence      NatureCode = "direct_evidence"
	NatureReasoningHypothesis NatureCode = "reasoning_hypothesis"
	NaturePendingValidation   NatureCode = "pending_validation"
)

type TargetType string

const (
	TargetSection           TargetType = "section"
	TargetAnchor            TargetType = "anchor"
	TargetIndustryChain     TargetType = "industry_chain"
	TargetIndustryChainNode TargetType = "industry_chain_node"
)

type ConfidenceCode string

const (
	ConfidenceHigh       ConfidenceCode = "high"
	ConfidenceMediumHigh ConfidenceCode = "medium_high"
	ConfidenceMedium     ConfidenceCode = "medium"
	ConfidenceLowMedium  ConfidenceCode = "low_medium"
	ConfidenceLow        ConfidenceCode = "low"
)

type DirectionCode string

const (
	DirectionUp     DirectionCode = "up"
	DirectionDown   DirectionCode = "down"
	DirectionStable DirectionCode = "stable"
)

type SignalConfidenceCode string

const (
	SignalConfidenceHigh    SignalConfidenceCode = "high"
	SignalConfidenceMedium  SignalConfidenceCode = "medium"
	SignalConfidenceLow     SignalConfidenceCode = "low"
	SignalConfidenceUnknown SignalConfidenceCode = "unknown"
)

type HorizonCode string

const (
	HorizonImmediate HorizonCode = "immediate"
	HorizonShort     HorizonCode = "short"
	HorizonMedium    HorizonCode = "medium"
	HorizonLong      HorizonCode = "long"
	HorizonFuture    HorizonCode = "future"
)

type EvidenceRoleCode string

const EvidenceRoleDirectTarget EvidenceRoleCode = "direct_target"

const (
	EvidenceRoleSupportsClaim        EvidenceRoleCode = "supports_claim"
	EvidenceRoleSupportsReasoning    EvidenceRoleCode = "supports_reasoning"
	EvidenceRoleSupportsTransmission EvidenceRoleCode = "supports_transmission"
)

type Result struct {
	Code  ResultCode `json:"code"`
	Label string     `json:"label"`
}

type Nature struct {
	Code  NatureCode `json:"code"`
	Label string     `json:"label"`
}

type Confidence struct {
	Code  ConfidenceCode `json:"code"`
	Label string         `json:"label"`
	Score *float64       `json:"score"`
}

type TimeWindow struct {
	Horizons []HorizonCode `json:"horizons"`
	Lag      *string       `json:"lag"`
	Label    string        `json:"label"`
}

type Effect struct {
	DisplayOrder int                  `json:"display_order"`
	Dimension    string               `json:"dimension"`
	Direction    DirectionCode        `json:"direction"`
	Confidence   SignalConfidenceCode `json:"confidence"`
}

type EvidenceReference struct {
	EvidenceID   string           `json:"evidence_id"`
	Role         EvidenceRoleCode `json:"role"`
	DisplayOrder int              `json:"display_order"`
}

type TargetReference struct {
	Type TargetType `json:"type"`
	Key  string     `json:"key"`
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

type TransmissionTarget struct {
	Ref     *TargetReference `json:"ref,omitempty"`
	Label   string           `json:"label"`
	Results []NamedResult    `json:"results"`
}

type NamedResult struct {
	Name   string `json:"name"`
	Result Result `json:"result"`
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
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
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
	GeneratedAt      time.Time       `json:"generated_at"`
	AnalysisWindow   AnalysisWindow  `json:"analysis_window"`
	Timezone         string          `json:"timezone"`
	Provenance       Provenance      `json:"provenance"`
	Statistics       Statistics      `json:"statistics"`
	Geopolitics      *Layer          `json:"geopolitics,omitempty"`
	Macroeconomics   *Layer          `json:"macroeconomics,omitempty"`
	IndustryChains   []IndustryChain `json:"industry_chains"`
}

type Record struct {
	ID                string
	PublisherReportID string
	ContractVersion   string
	ContentHash       string
	Content           Content
	PublishedAt       time.Time
}

type EvidenceLink struct {
	ID           string
	ReportID     string
	EvidenceID   string
	ScopeType    ScopeType
	ScopeKey     string
	Role         EvidenceRoleCode
	DisplayOrder int
}

type Evidence struct {
	EvidenceID   string
	Role         string
	DisplayOrder int
	PublishedAt  *time.Time
	Summary      string
	Keywords     []string
}

type Summary struct {
	ID                string
	PublisherReportID string
	ReportType        string
	Title             string
	GenerationStatus  string
	Simulation        bool
	GeneratedAt       time.Time
	Timezone          string
	HasGeopolitics    bool
	HasMacroeconomics bool
	Statistics        Statistics
	PublishedAt       time.Time
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

// IndustryChainImpactSummary is a read projection for a paged Miniapp card.
// It is derived from node impacts plus topology names and is not persisted as
// a second Report fact.
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

type LayerSnapshot struct {
	Key     string       `json:"key"`
	Title   string       `json:"title"`
	Summary LayerSummary `json:"summary"`
}

type Home struct {
	Report         Summary
	Geopolitics    *LayerSnapshot
	Macroeconomics *LayerSnapshot
}

type IndustryChainListRequest struct {
	ReportID string
	Limit    int
	Cursor   string
}

type IndustryChainListFilter struct {
	ReportID          string
	AfterDisplayOrder int
	Limit             int
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
	Record      Record
	ContentHash string
	Replayed    bool
}

type Store interface {
	PublicationStore
	ListReports(context.Context, ListFilter) (StorePage, error)
	GetReport(context.Context, string) (Record, error)
	GetHome(context.Context, string) (Home, error)
	GetLayer(context.Context, string, string) (Summary, Layer, []IndustryChainSummary, error)
	ListIndustryChains(context.Context, IndustryChainListFilter) (IndustryChainStorePage, error)
	GetIndustryChain(context.Context, string, string) (Summary, IndustryChain, error)
	ReportScopeExists(context.Context, string, ScopeType, string) (bool, bool, error)
	ListEvidence(context.Context, string, ScopeType, string) ([]Evidence, error)
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

func (s *UseCase) Publish(ctx context.Context, contractVersion, publisherReportID string, content Content) (PublicationResult, error) {
	if s == nil || s.store == nil {
		return PublicationResult{}, errors.New("Report store is required")
	}
	if contractVersion != ContractVersion {
		return PublicationResult{}, invalid("contract_version", "must equal "+ContractVersion)
	}
	if err := requiredText("publisher_report_id", publisherReportID, 200); err != nil {
		return PublicationResult{}, err
	}
	if err := ValidateContent(content); err != nil {
		return PublicationResult{}, err
	}
	payloadHash, err := ContentHash(contractVersion, content)
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
			if existing.ContractVersion != contractVersion || existing.ContentHash != payloadHash {
				return ErrPublicationConflict
			}
			result = PublicationResult{Record: *existing, ContentHash: existing.ContentHash, Replayed: true}
			return nil
		}
		reportID, err := coreid.New(coreid.Report)
		if err != nil {
			return fmt.Errorf("generate Report ID: %w", err)
		}
		links, err := buildEvidenceLinks(reportID, content)
		if err != nil {
			return err
		}
		evidenceIDs := uniqueEvidenceIDs(links)
		existingIDs, err := tx.ExistingEvidenceIDs(ctx, evidenceIDs)
		if err != nil {
			return err
		}
		if missing := firstMissingEvidence(evidenceIDs, existingIDs); missing != "" {
			return &ReferenceError{Path: "content.evidence_refs", Reference: missing,
				Message: "references unpublished Atomic Evidence"}
		}
		record := Record{
			ID: reportID, PublisherReportID: publisherReportID, ContractVersion: contractVersion,
			ContentHash: payloadHash, Content: content,
			PublishedAt: s.now().UTC().Truncate(time.Microsecond),
		}
		if err := tx.InsertReport(ctx, record); err != nil {
			return err
		}
		if err := tx.InsertEvidenceLinks(ctx, links); err != nil {
			return err
		}
		result = PublicationResult{Record: record, ContentHash: payloadHash}
		return nil
	})
	if err != nil {
		return PublicationResult{}, err
	}
	return result, nil
}

// ContentHash returns the versioned canonical hash stored with an immutable
// Report. publisher_report_id is intentionally not part of this content hash.
func ContentHash(contractVersion string, content Content) (string, error) {
	return canonicalPayloadHash(struct {
		ContractVersion string  `json:"contract_version"`
		Content         Content `json:"content"`
	}{ContractVersion: contractVersion, Content: content})
}

func (s *UseCase) List(ctx context.Context, request ListRequest) (Page, error) {
	if s == nil || s.store == nil {
		return Page{}, errors.New("Report store is required")
	}
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
		cursor, err := decodeCursor(request.Cursor)
		if err != nil || cursor.Version != 1 || !coreid.Is(cursor.ID, coreid.Report) ||
			!sameOptionalTime(cursor.PublishedFrom, request.PublishedFrom) ||
			!sameOptionalTime(cursor.PublishedTo, request.PublishedTo) {
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
		encoded, err := encodeCursor(reportCursor{Version: 1, PublishedFrom: cloneTime(request.PublishedFrom),
			PublishedTo: cloneTime(request.PublishedTo), PublishedAt: last.PublishedAt.UTC(), ID: last.ID})
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

func (s *UseCase) GetLayer(ctx context.Context, reportID, layerKey string) (Summary, Layer, []IndustryChainSummary, error) {
	if err := validateReportID(reportID); err != nil {
		return Summary{}, Layer{}, nil, err
	}
	if layerKey != strings.TrimSpace(layerKey) || (layerKey != "geopolitics" && layerKey != "macroeconomics") {
		return Summary{}, Layer{}, nil, ErrLayerNotFound
	}
	return s.store.GetLayer(ctx, reportID, layerKey)
}

func (s *UseCase) GetIndustryChain(ctx context.Context, reportID, chainKey string) (Summary, IndustryChain, error) {
	if err := validateReportID(reportID); err != nil {
		return Summary{}, IndustryChain{}, err
	}
	if chainKey != strings.TrimSpace(chainKey) || !localKeyPattern.MatchString(chainKey) {
		return Summary{}, IndustryChain{}, ErrChainNotFound
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
	if strings.TrimSpace(request.Cursor) != "" {
		cursor, err := decodeIndustryChainCursor(request.Cursor)
		if err != nil || cursor.Version != 1 || cursor.ReportID != request.ReportID || cursor.DisplayOrder < 1 {
			return IndustryChainPage{}, invalid("cursor", "is invalid for this Report industry-chain query")
		}
		filter.AfterDisplayOrder = cursor.DisplayOrder
	}
	page, err := s.store.ListIndustryChains(ctx, filter)
	if err != nil {
		return IndustryChainPage{}, err
	}
	result := IndustryChainPage{Items: page.Items}
	if page.HasMore && len(page.Items) > 0 {
		last := page.Items[len(page.Items)-1]
		encoded, err := encodeIndustryChainCursor(industryChainCursor{Version: 1, ReportID: request.ReportID, DisplayOrder: last.DisplayOrder})
		if err != nil {
			return IndustryChainPage{}, fmt.Errorf("encode Report industry-chain cursor: %w", err)
		}
		result.NextCursor = &encoded
	}
	return result, nil
}

func (s *UseCase) ListEvidence(ctx context.Context, reportID string, scopeType ScopeType, scopeKey string) ([]Evidence, error) {
	if err := validateReportID(reportID); err != nil {
		return nil, err
	}
	if !validScopeType(scopeType) {
		return nil, invalid("scope_type", "is not supported")
	}
	if scopeKey != strings.TrimSpace(scopeKey) || !localKeyPattern.MatchString(scopeKey) {
		return nil, invalid("scope_key", "must be a Report-local key")
	}
	reportExists, scopeExists, err := s.store.ReportScopeExists(ctx, reportID, scopeType, scopeKey)
	if err != nil {
		return nil, err
	}
	if !reportExists {
		return nil, ErrReportNotFound
	}
	if !scopeExists {
		return nil, ErrEvidenceScopeNotFound
	}
	return s.store.ListEvidence(ctx, reportID, scopeType, scopeKey)
}

func validateReportID(value string) error {
	if value != strings.TrimSpace(value) || !coreid.Is(value, coreid.Report) {
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
	Version      int    `json:"v"`
	ReportID     string `json:"report_id"`
	DisplayOrder int    `json:"display_order"`
}

func encodeCursor(cursor reportCursor) (string, error) {
	payload, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeCursor(value string) (reportCursor, error) {
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return reportCursor{}, err
	}
	var cursor reportCursor
	err = json.Unmarshal(payload, &cursor)
	return cursor, err
}

func encodeIndustryChainCursor(cursor industryChainCursor) (string, error) {
	payload, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
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

func ValidateContent(content Content) error {
	for _, field := range []struct {
		path, value string
		max         int
	}{
		{"content.report_type", content.ReportType, 100},
		{"content.title", content.Title, 500},
		{"content.generation_status", content.GenerationStatus, 100},
		{"content.timezone", content.Timezone, 100},
		{"content.provenance.template.name", content.Provenance.Template.Name, 500},
		{"content.provenance.template.version", content.Provenance.Template.Version, 100},
		{"content.provenance.template.role", content.Provenance.Template.Role, 500},
	} {
		if err := requiredText(field.path, field.value, field.max); err != nil {
			return err
		}
	}
	if content.GeneratedAt.IsZero() {
		return invalid("content.generated_at", "must be a timestamp")
	}
	if content.AnalysisWindow.StartedAt.IsZero() || content.AnalysisWindow.EndedAt.IsZero() ||
		!content.AnalysisWindow.StartedAt.Before(content.AnalysisWindow.EndedAt) {
		return invalid("content.analysis_window", "must contain a non-empty half-open time range")
	}
	for path, value := range map[string]*string{
		"content.provenance.derived_from_report_id": content.Provenance.DerivedFromReportID,
		"content.provenance.frozen_source_sha256":   content.Provenance.FrozenSourceSHA256,
		"content.provenance.frozen_source_commit":   content.Provenance.FrozenSourceCommit,
	} {
		if err := optionalText(path, value, 500); err != nil {
			return err
		}
	}
	if err := validateStatistics(content.Statistics); err != nil {
		return err
	}
	if content.IndustryChains == nil || len(content.IndustryChains) == 0 {
		return invalid("content.industry_chains", "must contain at least one industry-chain analysis")
	}
	index := newContentIndex()
	for name, layer := range map[string]*Layer{"geopolitics": content.Geopolitics, "macroeconomics": content.Macroeconomics} {
		if layer == nil {
			continue
		}
		if layer.Key != name {
			return invalid("content."+name+".key", "must equal "+name)
		}
		if err := validateLayer("content."+name, *layer, &index); err != nil {
			return err
		}
		index.layers[name] = struct{}{}
	}
	if err := validateOrdered("content.industry_chains", len(content.IndustryChains), func(i int) int { return content.IndustryChains[i].DisplayOrder }); err != nil {
		return err
	}
	for i, chain := range content.IndustryChains {
		if err := validateIndustryChain(fmt.Sprintf("content.industry_chains[%d]", i), chain, &index); err != nil {
			return err
		}
	}
	if content.Statistics.GeopoliticAnchorCount != optionalAnchorCount(content.Geopolitics) ||
		content.Statistics.MacroeconomicAnchorCount != optionalAnchorCount(content.Macroeconomics) ||
		content.Statistics.IndustryChainCount != len(content.IndustryChains) {
		return invalid("content.statistics", "structural counts must match the published snapshot")
	}
	return validateCrossReferences(content, index)
}

func ValidateLayerSnapshot(layer Layer) error {
	if layer.Key != "geopolitics" && layer.Key != "macroeconomics" {
		return invalid("layer.key", "is not a supported section")
	}
	index := newContentIndex()
	return validateLayer("layer", layer, &index)
}

func ValidateIndustryChainSnapshot(chain IndustryChain) error {
	index := newContentIndex()
	return validateIndustryChain("industry_chain", chain, &index)
}

func ValidateIndustryChainSummaryProjection(summary IndustryChainSummary) error {
	if !localKeyPattern.MatchString(summary.Key) || summary.DisplayOrder < 1 {
		return invalid("industry_chain_summary", "has an invalid identity or display order")
	}
	if err := requiredText("industry_chain_summary.name", summary.Name, 500); err != nil {
		return err
	}
	if err := validateClaim("industry_chain_summary.claim", summary.Claim); err != nil {
		return err
	}
	if err := requiredText("industry_chain_summary.status", summary.Status, 10_000); err != nil {
		return err
	}
	if err := validateResult("industry_chain_summary.result", summary.Result); err != nil {
		return err
	}
	if err := validateConfidence("industry_chain_summary.confidence", summary.Confidence); err != nil {
		return err
	}
	if err := validateTimeWindow("industry_chain_summary.time_window", summary.TimeWindow); err != nil {
		return err
	}
	if summary.ImpactItems == nil || len(summary.ImpactItems) == 0 || summary.EvidenceCount < 0 {
		return invalid("industry_chain_summary", "impact_items must be non-empty and evidence_count must be non-negative")
	}
	if err := validateOrdered("industry_chain_summary.impact_items", len(summary.ImpactItems), func(i int) int { return summary.ImpactItems[i].DisplayOrder }); err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for index, item := range summary.ImpactItems {
		path := fmt.Sprintf("industry_chain_summary.impact_items[%d]", index)
		if !localKeyPattern.MatchString(item.Key) || !localKeyPattern.MatchString(item.NodeKey) || item.EvidenceCount < 0 {
			return invalid(path, "has an invalid identity or Evidence count")
		}
		if _, duplicate := seen[item.NodeKey]; duplicate {
			return invalid(path+".node_key", "duplicates an impacted topology node")
		}
		seen[item.NodeKey] = struct{}{}
		if err := requiredText(path+".name", item.Name, 500); err != nil {
			return err
		}
		if err := validateResult(path+".result", item.Result); err != nil {
			return err
		}
		if err := validateNature(path+".nature", item.Nature); err != nil {
			return err
		}
		if item.Nature.Code == NatureDirectEvidence && item.EvidenceCount == 0 {
			return invalid(path+".evidence_count", "direct evidence must be available")
		}
		if item.Nature.Code != NatureDirectEvidence && item.EvidenceCount != 0 {
			return invalid(path+".evidence_count", "hypothesis or pending validation must not expose direct Evidence")
		}
		if err := validateConfidence(path+".confidence", item.Confidence); err != nil {
			return err
		}
		if err := validateTimeWindow(path+".time_window", item.TimeWindow); err != nil {
			return err
		}
	}
	return nil
}

type contentIndex struct {
	layers  map[string]struct{}
	anchors map[string]struct{}
	chains  map[string]struct{}
	nodes   map[string]string
	claims  map[string]struct{}
	keys    map[ScopeType]map[string]struct{}
}

func newContentIndex() contentIndex {
	result := contentIndex{layers: map[string]struct{}{}, anchors: map[string]struct{}{}, chains: map[string]struct{}{}, nodes: map[string]string{}, claims: map[string]struct{}{}, keys: map[ScopeType]map[string]struct{}{}}
	for _, scope := range []ScopeType{ScopeSectionSummary, ScopeAnchor, ScopeReasoningStep, ScopeTransmission, ScopeIndustryChainSummary, ScopeIndustryChainNode} {
		result.keys[scope] = map[string]struct{}{}
	}
	return result
}

func (i *contentIndex) addClaim(key, path string) error {
	if _, duplicate := i.claims[key]; duplicate {
		return invalid(path, "duplicates a claim key in the same Report")
	}
	i.claims[key] = struct{}{}
	return nil
}

func (i *contentIndex) add(scope ScopeType, key, path string) error {
	if !localKeyPattern.MatchString(key) {
		return invalid(path, "must be a Report-local key")
	}
	if _, duplicate := i.keys[scope][key]; duplicate {
		return invalid(path, "duplicates a key in the same Report scope type")
	}
	i.keys[scope][key] = struct{}{}
	return nil
}

func optionalAnchorCount(layer *Layer) int {
	if layer == nil {
		return 0
	}
	return len(layer.Detail.Anchors)
}

func validateLayer(path string, layer Layer, index *contentIndex) error {
	if err := requiredText(path+".title", layer.Title, 500); err != nil {
		return err
	}
	if err := index.add(ScopeSectionSummary, layer.Key, path+".key"); err != nil {
		return err
	}
	if err := validateClaim(path+".summary.claim", layer.Summary.Claim); err != nil {
		return err
	}
	if err := index.addClaim(layer.Summary.Claim.Key, path+".summary.claim.key"); err != nil {
		return err
	}
	if err := validateEvidenceRefs(path+".summary.evidence_refs", layer.Summary.EvidenceRefs, EvidenceRoleSupportsClaim); err != nil {
		return err
	}
	if err := validateLayerUncertainty(path+".summary.uncertainty", layer.Summary.Uncertainty); err != nil {
		return err
	}
	if layer.Summary.Transmissions == nil || layer.Detail.Anchors == nil || layer.Detail.ReasoningSteps == nil || layer.Detail.RelatedChainKeys == nil {
		return invalid(path, "all collections must be arrays")
	}
	if len(layer.Summary.Transmissions) == 0 {
		return invalid(path+".summary.transmissions", "must contain at least one downward transmission")
	}
	if len(layer.Detail.Anchors) == 0 {
		return invalid(path+".detail.anchors", "must contain at least one affected anchor")
	}
	if err := validateOrdered(path+".summary.transmissions", len(layer.Summary.Transmissions), func(i int) int { return layer.Summary.Transmissions[i].DisplayOrder }); err != nil {
		return err
	}
	for i, transmission := range layer.Summary.Transmissions {
		itemPath := fmt.Sprintf("%s.summary.transmissions[%d]", path, i)
		if err := index.add(ScopeTransmission, transmission.Key, itemPath+".key"); err != nil {
			return err
		}
		for _, field := range []struct{ name, value string }{{"source_claim_key", transmission.SourceClaimKey}, {"source_conclusion", transmission.SourceConclusion}, {"logic", transmission.Logic}, {"relation_nature", transmission.RelationNature}, {"status", transmission.Status}} {
			if err := requiredText(itemPath+"."+field.name, field.value, 10_000); err != nil {
				return err
			}
		}
		if transmission.Targets == nil || len(transmission.Targets) == 0 {
			return invalid(itemPath+".targets", "must contain at least one target")
		}
		for j, target := range transmission.Targets {
			targetPath := fmt.Sprintf("%s.targets[%d]", itemPath, j)
			if err := requiredText(targetPath+".label", target.Label, 500); err != nil {
				return err
			}
			if target.Ref != nil && !validTargetReference(*target.Ref) {
				return invalid(targetPath+".ref", "is not a supported Report target")
			}
			if target.Results == nil || len(target.Results) == 0 {
				return invalid(targetPath+".results", "must contain at least one named result")
			}
			for k, result := range target.Results {
				if err := requiredText(fmt.Sprintf("%s.results[%d].name", targetPath, k), result.Name, 500); err != nil {
					return err
				}
				if err := validateResult(fmt.Sprintf("%s.results[%d].result", targetPath, k), result.Result); err != nil {
					return err
				}
			}
		}
		if err := validateConfidence(itemPath+".confidence", transmission.Confidence); err != nil {
			return err
		}
		if err := validateEvidenceRefs(itemPath+".evidence_refs", transmission.EvidenceRefs, EvidenceRoleSupportsTransmission); err != nil {
			return err
		}
	}
	if err := validateOrdered(path+".detail.anchors", len(layer.Detail.Anchors), func(i int) int { return layer.Detail.Anchors[i].DisplayOrder }); err != nil {
		return err
	}
	for i, anchor := range layer.Detail.Anchors {
		itemPath := fmt.Sprintf("%s.detail.anchors[%d]", path, i)
		if err := index.add(ScopeAnchor, anchor.Key, itemPath+".key"); err != nil {
			return err
		}
		index.anchors[anchor.Key] = struct{}{}
		if err := requiredText(itemPath+".name", anchor.Name, 500); err != nil {
			return err
		}
		if err := requiredText(itemPath+".reasoning", anchor.Reasoning, 10_000); err != nil {
			return err
		}
		if err := validateEffects(itemPath+".effects", anchor.Effects); err != nil {
			return err
		}
		if err := validateResult(itemPath+".result", anchor.Result); err != nil {
			return err
		}
		if err := validateNatureAndEvidence(itemPath, anchor.Nature, anchor.EvidenceRefs); err != nil {
			return err
		}
		if err := validateTimeWindow(itemPath+".time_window", anchor.TimeWindow); err != nil {
			return err
		}
		if err := validateConfidence(itemPath+".confidence", anchor.Confidence); err != nil {
			return err
		}
		if err := optionalText(itemPath+".source_ref", anchor.SourceRef, 500); err != nil {
			return err
		}
	}
	if err := validateOrdered(path+".detail.reasoning_steps", len(layer.Detail.ReasoningSteps), func(i int) int { return layer.Detail.ReasoningSteps[i].DisplayOrder }); err != nil {
		return err
	}
	for i, step := range layer.Detail.ReasoningSteps {
		itemPath := fmt.Sprintf("%s.detail.reasoning_steps[%d]", path, i)
		if err := index.add(ScopeReasoningStep, step.Key, itemPath+".key"); err != nil {
			return err
		}
		for _, field := range []struct{ name, value string }{{"input", step.Input}, {"mechanism", step.Mechanism}, {"output", step.Output}, {"type", step.Type}} {
			if err := requiredText(itemPath+"."+field.name, field.value, 10_000); err != nil {
				return err
			}
		}
		if err := validateConfidence(itemPath+".confidence", step.Confidence); err != nil {
			return err
		}
		if err := validateEvidenceRefs(itemPath+".evidence_refs", step.EvidenceRefs, EvidenceRoleSupportsReasoning); err != nil {
			return err
		}
	}
	return nil
}

func validateIndustryChain(path string, chain IndustryChain, index *contentIndex) error {
	if err := index.add(ScopeIndustryChainSummary, chain.Key, path+".key"); err != nil {
		return err
	}
	if _, duplicate := index.chains[chain.Key]; duplicate {
		return invalid(path+".key", "duplicates an industry chain")
	}
	index.chains[chain.Key] = struct{}{}
	if err := requiredText(path+".name", chain.Name, 500); err != nil {
		return err
	}
	if err := validateClaim(path+".summary.claim", chain.Summary.Claim); err != nil {
		return err
	}
	if err := index.addClaim(chain.Summary.Claim.Key, path+".summary.claim.key"); err != nil {
		return err
	}
	for _, field := range []struct{ name, value string }{{"status", chain.Summary.Status}, {"path", chain.Summary.Path}} {
		if err := requiredText(path+".summary."+field.name, field.value, 10_000); err != nil {
			return err
		}
	}
	if err := optionalText(path+".summary.accepted_hypothesis_summary", chain.Summary.AcceptedHypothesisSummary, 10_000); err != nil {
		return err
	}
	if err := validateResult(path+".summary.result", chain.Summary.Result); err != nil {
		return err
	}
	if err := validateConfidence(path+".summary.confidence", chain.Summary.Confidence); err != nil {
		return err
	}
	if err := validateTimeWindow(path+".summary.time_window", chain.Summary.TimeWindow); err != nil {
		return err
	}
	if err := validateEvidenceRefs(path+".summary.evidence_refs", chain.Summary.EvidenceRefs, EvidenceRoleSupportsClaim); err != nil {
		return err
	}
	if chain.Summary.Graph.Nodes == nil || len(chain.Summary.Graph.Nodes) == 0 || chain.Summary.Graph.Edges == nil || chain.Detail.NodeImpacts == nil || len(chain.Detail.NodeImpacts) == 0 {
		return invalid(path, "summary.graph.nodes and detail.node_impacts must be non-empty; all collections must be arrays")
	}
	if err := validateOrdered(path+".summary.graph.nodes", len(chain.Summary.Graph.Nodes), func(i int) int { return chain.Summary.Graph.Nodes[i].DisplayOrder }); err != nil {
		return err
	}
	topologyNodes := map[string]struct{}{}
	for i, node := range chain.Summary.Graph.Nodes {
		itemPath := fmt.Sprintf("%s.summary.graph.nodes[%d]", path, i)
		if !localKeyPattern.MatchString(node.Key) {
			return invalid(itemPath+".key", "must be a Report-local key")
		}
		if _, duplicate := topologyNodes[node.Key]; duplicate {
			return invalid(itemPath+".key", "duplicates a topology node")
		}
		topologyNodes[node.Key] = struct{}{}
		if err := requiredText(itemPath+".name", node.Name, 500); err != nil {
			return err
		}
	}
	if err := validateOrdered(path+".summary.graph.edges", len(chain.Summary.Graph.Edges), func(i int) int { return chain.Summary.Graph.Edges[i].DisplayOrder }); err != nil {
		return err
	}
	edges := map[string]struct{}{}
	for i, edge := range chain.Summary.Graph.Edges {
		itemPath := fmt.Sprintf("%s.summary.graph.edges[%d]", path, i)
		if !localKeyPattern.MatchString(edge.Key) {
			return invalid(itemPath+".key", "must be a Report-local key")
		}
		if _, duplicate := edges[edge.Key]; duplicate {
			return invalid(itemPath+".key", "duplicates an edge")
		}
		edges[edge.Key] = struct{}{}
		if _, ok := topologyNodes[edge.FromNodeKey]; !ok {
			return invalid(itemPath+".from_node_key", "must reference this chain topology")
		}
		if _, ok := topologyNodes[edge.ToNodeKey]; !ok {
			return invalid(itemPath+".to_node_key", "must reference this chain topology")
		}
		if edge.FromNodeKey == edge.ToNodeKey {
			return invalid(itemPath, "must not be a self edge")
		}
		if err := requiredText(itemPath+".relation_label", edge.RelationLabel, 500); err != nil {
			return err
		}
	}
	if err := validateOrdered(path+".detail.node_impacts", len(chain.Detail.NodeImpacts), func(i int) int { return chain.Detail.NodeImpacts[i].DisplayOrder }); err != nil {
		return err
	}
	seenImpacts := map[string]struct{}{}
	for i, impact := range chain.Detail.NodeImpacts {
		itemPath := fmt.Sprintf("%s.detail.node_impacts[%d]", path, i)
		if err := index.add(ScopeIndustryChainNode, impact.Key, itemPath+".key"); err != nil {
			return err
		}
		if _, ok := topologyNodes[impact.NodeKey]; !ok {
			return invalid(itemPath+".node_key", "must reference this chain topology")
		}
		if _, duplicate := seenImpacts[impact.NodeKey]; duplicate {
			return invalid(itemPath+".node_key", "duplicates an impacted topology node")
		}
		seenImpacts[impact.NodeKey] = struct{}{}
		index.nodes[impact.Key] = chain.Key
		if err := requiredText(itemPath+".reasoning", impact.Reasoning, 10_000); err != nil {
			return err
		}
		if err := validateEffects(itemPath+".effects", impact.Effects); err != nil {
			return err
		}
		if err := validateResult(itemPath+".result", impact.Result); err != nil {
			return err
		}
		if err := validateNatureAndEvidence(itemPath, impact.Nature, impact.EvidenceRefs); err != nil {
			return err
		}
		if err := validateTimeWindow(itemPath+".time_window", impact.TimeWindow); err != nil {
			return err
		}
		if err := validateConfidence(itemPath+".confidence", impact.Confidence); err != nil {
			return err
		}
	}
	if err := requiredText(path+".summary.uncertainty.counterevidence_and_gap", chain.Summary.Uncertainty.CounterevidenceAndGap, 10_000); err != nil {
		return err
	}
	return requiredText(path+".summary.uncertainty.stop_condition", chain.Summary.Uncertainty.StopCondition, 10_000)
}

func validateCrossReferences(content Content, index contentIndex) error {
	for sectionName, layer := range map[string]*Layer{"geopolitics": content.Geopolitics, "macroeconomics": content.Macroeconomics} {
		if layer == nil {
			continue
		}
		for i, key := range layer.Detail.RelatedChainKeys {
			if _, ok := index.chains[key]; !ok {
				return &ReferenceError{Path: fmt.Sprintf("content.%s.detail.related_chain_keys[%d]", sectionName, i), Reference: key, Message: "does not identify an industry chain"}
			}
		}
		for i, transmission := range layer.Summary.Transmissions {
			if transmission.SourceClaimKey != layer.Summary.Claim.Key {
				return invalid(fmt.Sprintf("content.%s.summary.transmissions[%d].source_claim_key", sectionName, i), "must reference this section claim")
			}
			for j, target := range transmission.Targets {
				if target.Ref != nil && !targetExists(index, *target.Ref) {
					return &ReferenceError{Path: fmt.Sprintf("content.%s.summary.transmissions[%d].targets[%d].ref", sectionName, i, j), Reference: string(target.Ref.Type) + ":" + target.Ref.Key, Message: "does not identify a Report target"}
				}
			}
		}
	}
	return nil
}

func targetExists(index contentIndex, ref TargetReference) bool {
	switch ref.Type {
	case TargetSection:
		_, ok := index.layers[ref.Key]
		return ok
	case TargetAnchor:
		_, ok := index.anchors[ref.Key]
		return ok
	case TargetIndustryChain:
		_, ok := index.chains[ref.Key]
		return ok
	case TargetIndustryChainNode:
		_, ok := index.nodes[ref.Key]
		return ok
	default:
		return false
	}
}

func validTargetReference(ref TargetReference) bool {
	if !localKeyPattern.MatchString(ref.Key) {
		return false
	}
	switch ref.Type {
	case TargetSection, TargetAnchor, TargetIndustryChain, TargetIndustryChainNode:
		return true
	default:
		return false
	}
}

func validateClaim(path string, value Claim) error {
	if !localKeyPattern.MatchString(value.Key) {
		return invalid(path+".key", "must be a Report-local key")
	}
	return requiredText(path+".text", value.Text, 10_000)
}

func validateEffects(path string, values []Effect) error {
	if values == nil || len(values) == 0 {
		return invalid(path, "must contain at least one structured effect")
	}
	if err := validateOrdered(path, len(values), func(i int) int { return values[i].DisplayOrder }); err != nil {
		return err
	}
	for i, effect := range values {
		itemPath := fmt.Sprintf("%s[%d]", path, i)
		if err := requiredText(itemPath+".dimension", effect.Dimension, 500); err != nil {
			return err
		}
		switch effect.Direction {
		case DirectionUp, DirectionDown, DirectionStable:
		default:
			return invalid(itemPath+".direction", "is not supported")
		}
		switch effect.Confidence {
		case SignalConfidenceHigh, SignalConfidenceMedium, SignalConfidenceLow, SignalConfidenceUnknown:
		default:
			return invalid(itemPath+".confidence", "is not supported")
		}
	}
	return nil
}

func validateTimeWindow(path string, value TimeWindow) error {
	if value.Horizons == nil || len(value.Horizons) == 0 {
		return invalid(path+".horizons", "must contain at least one horizon")
	}
	seen := map[HorizonCode]struct{}{}
	for i, horizon := range value.Horizons {
		switch horizon {
		case HorizonImmediate, HorizonShort, HorizonMedium, HorizonLong, HorizonFuture:
		default:
			return invalid(fmt.Sprintf("%s.horizons[%d]", path, i), "is not supported")
		}
		if _, duplicate := seen[horizon]; duplicate {
			return invalid(fmt.Sprintf("%s.horizons[%d]", path, i), "duplicates a horizon")
		}
		seen[horizon] = struct{}{}
	}
	if err := optionalText(path+".lag", value.Lag, 500); err != nil {
		return err
	}
	return requiredText(path+".label", value.Label, 500)
}

func validateLayerUncertainty(path string, value LayerUncertainty) error {
	for name, field := range map[string]*string{"counterevidence": value.Counterevidence, "boundary": value.Boundary, "reversal_condition": value.ReversalCondition} {
		if field == nil {
			return invalid(path+"."+name, "is required")
		}
		if err := requiredText(path+"."+name, *field, 10_000); err != nil {
			return err
		}
	}
	if err := optionalText(path+".evidence_gap", value.EvidenceGap, 10_000); err != nil {
		return err
	}
	return validateCheckpoints(path+".checkpoints", value.Checkpoints)
}

func validateNatureAndEvidence(path string, nature Nature, refs []EvidenceReference) error {
	if err := validateNature(path+".nature", nature); err != nil {
		return err
	}
	if err := validateEvidenceRefs(path+".evidence_refs", refs, EvidenceRoleDirectTarget); err != nil {
		return err
	}
	if nature.Code == NatureDirectEvidence && len(refs) == 0 {
		return invalid(path+".evidence_refs", "direct evidence requires at least one Evidence reference")
	}
	if nature.Code != NatureDirectEvidence && len(refs) != 0 {
		return invalid(path+".evidence_refs", "hypothesis or pending validation must not cite direct target Evidence")
	}
	return nil
}

func validateStatistics(value Statistics) error {
	counts := []struct {
		name  string
		value int
	}{{"event_count", value.EventCount}, {"ordinary_fact_count", value.OrdinaryFactCount}, {"signal_fact_count", value.SignalFactCount}, {"transmission_hypothesis_count", value.TransmissionHypothesisCount}, {"geopolitic_anchor_count", value.GeopoliticAnchorCount}, {"macroeconomic_anchor_count", value.MacroeconomicAnchorCount}, {"signaled_chain_node_count", value.SignaledChainNodeCount}, {"industry_chain_count", value.IndustryChainCount}}
	for _, count := range counts {
		if count.value < 0 {
			return invalid("content.statistics."+count.name, "must be non-negative")
		}
	}
	return nil
}

func validateResult(path string, value Result) error {
	wantLabel := ""
	switch value.Code {
	case ResultWarming:
		wantLabel = "升温"
	case ResultCooling:
		wantLabel = "降温"
	case ResultDiverging:
		wantLabel = "分化"
	case ResultStable:
		wantLabel = "稳定"
	case ResultMixed:
		return requiredText(path+".label", value.Label, 100)
	case ResultPending:
		wantLabel = "待验证"
	default:
		return invalid(path+".code", "is not supported")
	}
	if value.Label != wantLabel {
		return invalid(path+".label", "does not match result code")
	}
	return nil
}

func validateNature(path string, value Nature) error {
	wantLabel := ""
	switch value.Code {
	case NatureDirectEvidence:
		wantLabel = "直接证据"
	case NatureReasoningHypothesis:
		wantLabel = "推理假设"
	case NaturePendingValidation:
		wantLabel = "待验证"
	default:
		return invalid(path+".code", "is not supported")
	}
	if value.Label != wantLabel {
		return invalid(path+".label", "does not match nature code")
	}
	return nil
}

func validateConfidence(path string, value Confidence) error {
	wantLabel := ""
	switch value.Code {
	case ConfidenceHigh:
		wantLabel = "高"
	case ConfidenceMediumHigh:
		wantLabel = "中–高"
	case ConfidenceMedium:
		wantLabel = "中"
	case ConfidenceLowMedium:
		wantLabel = "低–中"
	case ConfidenceLow:
		wantLabel = "低"
	default:
		return invalid(path+".code", "is not supported")
	}
	if err := requiredText(path+".label", value.Label, 100); err != nil {
		return err
	}
	if value.Label != wantLabel {
		return invalid(path+".label", "does not match confidence code")
	}
	if value.Score != nil && (math.IsNaN(*value.Score) || math.IsInf(*value.Score, 0) || *value.Score < 0 || *value.Score > 1) {
		return invalid(path+".score", "must be null or between 0 and 1")
	}
	return nil
}

func validateEvidenceRefs(path string, values []EvidenceReference, allowedRoles ...EvidenceRoleCode) error {
	if values == nil {
		return invalid(path, "must be an array")
	}
	seen := map[string]struct{}{}
	for index, value := range values {
		itemPath := fmt.Sprintf("%s[%d]", path, index)
		if value.DisplayOrder != index+1 {
			return invalid(itemPath+".display_order", "must be continuous from 1")
		}
		if !coreid.Is(value.EvidenceID, coreid.Evidence) {
			return &ReferenceError{Path: itemPath + ".evidence_id", Reference: value.EvidenceID, Message: "must be a canonical Atomic Evidence ID"}
		}
		if _, duplicate := seen[value.EvidenceID]; duplicate {
			return invalid(itemPath+".evidence_id", "duplicates an Evidence in this scope")
		}
		seen[value.EvidenceID] = struct{}{}
		allowed := len(allowedRoles) == 0
		for _, role := range allowedRoles {
			if value.Role == role {
				allowed = true
				break
			}
		}
		if !allowed {
			return invalid(itemPath+".role", "is not valid for this Report scope")
		}
	}
	return nil
}

func validateCheckpoints(path string, values []Checkpoint) error {
	if values == nil {
		return invalid(path, "must be an array")
	}
	if err := validateOrdered(path, len(values), func(index int) int { return values[index].DisplayOrder }); err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for index, value := range values {
		itemPath := fmt.Sprintf("%s[%d]", path, index)
		if !localKeyPattern.MatchString(value.Key) {
			return invalid(itemPath+".key", "must be a Report-local key")
		}
		if _, duplicate := seen[value.Key]; duplicate {
			return invalid(itemPath+".key", "duplicates a checkpoint")
		}
		seen[value.Key] = struct{}{}
		if err := requiredText(itemPath+".summary", value.Summary, 10_000); err != nil {
			return err
		}
	}
	return nil
}

func validateOrdered(path string, length int, order func(int) int) error {
	for index := 0; index < length; index++ {
		if order(index) != index+1 {
			return invalid(fmt.Sprintf("%s[%d].display_order", path, index), "must be continuous from 1")
		}
	}
	return nil
}

func requiredText(path, value string, max int) error {
	if strings.TrimSpace(value) == "" {
		return invalid(path, "must not be blank")
	}
	if value != strings.TrimSpace(value) {
		return invalid(path, "must not contain leading or trailing whitespace")
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

func buildEvidenceLinks(reportID string, content Content) ([]EvidenceLink, error) {
	type scopedRefs struct {
		typeName ScopeType
		key      string
		refs     []EvidenceReference
	}
	values := make([]scopedRefs, 0)
	for _, layer := range []*Layer{content.Geopolitics, content.Macroeconomics} {
		if layer == nil {
			continue
		}
		values = append(values, scopedRefs{ScopeSectionSummary, layer.Key, layer.Summary.EvidenceRefs})
		for _, anchor := range layer.Detail.Anchors {
			values = append(values, scopedRefs{ScopeAnchor, anchor.Key, anchor.EvidenceRefs})
		}
		for _, step := range layer.Detail.ReasoningSteps {
			values = append(values, scopedRefs{ScopeReasoningStep, step.Key, step.EvidenceRefs})
		}
		for _, transmission := range layer.Summary.Transmissions {
			values = append(values, scopedRefs{ScopeTransmission, transmission.Key, transmission.EvidenceRefs})
		}
	}
	for _, chain := range content.IndustryChains {
		values = append(values, scopedRefs{ScopeIndustryChainSummary, chain.Key, chain.Summary.EvidenceRefs})
		for _, node := range chain.Detail.NodeImpacts {
			values = append(values, scopedRefs{ScopeIndustryChainNode, node.Key, node.EvidenceRefs})
		}
	}
	links := make([]EvidenceLink, 0)
	for _, value := range values {
		for _, reference := range value.refs {
			linkID, err := coreid.New(coreid.ReportEvidenceLink)
			if err != nil {
				return nil, fmt.Errorf("generate Report Evidence Link ID: %w", err)
			}
			links = append(links, EvidenceLink{ID: linkID, ReportID: reportID, EvidenceID: reference.EvidenceID,
				ScopeType: value.typeName, ScopeKey: value.key, Role: reference.Role, DisplayOrder: reference.DisplayOrder})
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

func validScopeType(value ScopeType) bool {
	switch value {
	case ScopeSectionSummary, ScopeAnchor, ScopeReasoningStep, ScopeTransmission,
		ScopeIndustryChainSummary, ScopeIndustryChainNode:
		return true
	default:
		return false
	}
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
