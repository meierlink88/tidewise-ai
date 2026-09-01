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
	ContractVersion = "report-publication.v1"
	DefaultLimit    = 20
	MaxLimit        = 100
)

type ScopeType string

const (
	ScopeReportCard         ScopeType = "report_card"
	ScopeLayer              ScopeType = "layer"
	ScopeAnchor             ScopeType = "anchor"
	ScopeReasoningStep      ScopeType = "reasoning_step"
	ScopeTransmissionPath   ScopeType = "transmission_path"
	ScopeCandidateMechanism ScopeType = "candidate_mechanism"
	ScopeIndustryChain      ScopeType = "industry_chain"
	ScopeIndustryChainNode  ScopeType = "industry_chain_node"
)

type ResultCode string

const (
	ResultWarming   ResultCode = "warming"
	ResultCooling   ResultCode = "cooling"
	ResultDiverging ResultCode = "diverging"
	ResultPending   ResultCode = "pending"
)

type NatureCode string

const (
	NatureDirectEvidence      NatureCode = "direct_evidence"
	NatureReasoningHypothesis NatureCode = "reasoning_hypothesis"
	NaturePendingValidation   NatureCode = "pending_validation"
)

type CardKind string

const (
	CardGeopolitics    CardKind = "geopolitics"
	CardMacroeconomics CardKind = "macroeconomics"
	CardIndustryChain  CardKind = "industry_chain"
)

type TargetType string

const (
	TargetLayer             TargetType = "layer"
	TargetAnchor            TargetType = "anchor"
	TargetIndustryChain     TargetType = "industry_chain"
	TargetIndustryChainNode TargetType = "industry_chain_node"
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
	Label string   `json:"label"`
	Score *float64 `json:"score"`
}

type EvidenceReference struct {
	EvidenceID   string `json:"evidence_id"`
	Role         string `json:"role"`
	DisplayOrder int    `json:"display_order"`
}

type TargetReference struct {
	Type TargetType `json:"type"`
	Key  string     `json:"key"`
}

type ImpactItem struct {
	Ref        TargetReference `json:"ref"`
	Name       string          `json:"name"`
	Result     Result          `json:"result"`
	Confidence Confidence      `json:"confidence"`
	TimeWindow string          `json:"time_window"`
}

type ReportCard struct {
	Key          string              `json:"key"`
	Kind         CardKind            `json:"kind"`
	DisplayOrder int                 `json:"display_order"`
	DetailRef    TargetReference     `json:"detail_ref"`
	Title        string              `json:"title"`
	Subtitle     string              `json:"subtitle"`
	Conclusion   string              `json:"conclusion"`
	Result       Result              `json:"result"`
	Confidence   Confidence          `json:"confidence"`
	TimeWindow   string              `json:"time_window"`
	ImpactItems  []ImpactItem        `json:"impact_items"`
	EvidenceRefs []EvidenceReference `json:"evidence_refs"`
}

type Statistics struct {
	EventCount                           int     `json:"event_count"`
	OrdinaryFactCount                    int     `json:"ordinary_fact_count"`
	SignalFactCount                      int     `json:"signal_fact_count"`
	TransmissionHypothesisCount          int     `json:"transmission_hypothesis_count"`
	RemainingTopologyPendingCount        int     `json:"remaining_topology_pending_count"`
	AdaptiveInclusionThreshold           float64 `json:"adaptive_inclusion_threshold"`
	AdaptiveContinuationThreshold        float64 `json:"adaptive_continuation_threshold"`
	AdaptiveHardMaxHops                  int     `json:"adaptive_hard_max_hops"`
	AdaptiveObservedMaxHops              int     `json:"adaptive_observed_max_hops"`
	AdaptiveStoppedByConfidence          int     `json:"adaptive_stopped_by_confidence"`
	AdaptiveStoppedByNoUnvisitedNeighbor int     `json:"adaptive_stopped_by_no_unvisited_neighbor"`
	AdaptiveRejectedBelowInclusion       int     `json:"adaptive_rejected_below_inclusion"`
	GeopoliticAnchorCount                int     `json:"geopolitic_anchor_count"`
	MacroeconomicAnchorCount             int     `json:"macroeconomic_anchor_count"`
	SignaledChainNodeCount               int     `json:"signaled_chain_node_count"`
	IndustryChainCount                   int     `json:"industry_chain_count"`
	UnmappedChainNodeCount               int     `json:"unmapped_chain_node_count"`
}

type Anchor struct {
	Key          string              `json:"key"`
	DisplayOrder int                 `json:"display_order"`
	Name         string              `json:"name"`
	CurrentState string              `json:"current_state"`
	Result       Result              `json:"result"`
	Nature       Nature              `json:"nature"`
	Reasoning    string              `json:"reasoning"`
	TimeWindow   string              `json:"time_window"`
	Confidence   Confidence          `json:"confidence"`
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
	Ref    TargetReference `json:"ref"`
	Label  string          `json:"label"`
	Result Result          `json:"result"`
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
	EvidenceRefs     []EvidenceReference  `json:"evidence_refs"`
}

type CandidateMechanism struct {
	Key          string              `json:"key"`
	DisplayOrder int                 `json:"display_order"`
	Mechanism    string              `json:"mechanism"`
	EvidenceGap  *string             `json:"evidence_gap"`
	Confidence   Confidence          `json:"confidence"`
	EvidenceRefs []EvidenceReference `json:"evidence_refs"`
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
	EvidenceRefs         []EvidenceReference  `json:"evidence_refs"`
}

type IndustryChainNode struct {
	Key          string              `json:"key"`
	DisplayOrder int                 `json:"display_order"`
	Name         string              `json:"name"`
	Impact       string              `json:"impact"`
	Result       Result              `json:"result"`
	Nature       Nature              `json:"nature"`
	Reasoning    string              `json:"reasoning"`
	TimeWindow   string              `json:"time_window"`
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
	EvidenceRefs              []EvidenceReference `json:"evidence_refs"`
	Nodes                     []IndustryChainNode `json:"nodes"`
	Edges                     []IndustryChainEdge `json:"edges"`
	Uncertainty               ChainUncertainty    `json:"uncertainty"`
}

type CompanyBoundary struct {
	Key          string `json:"key"`
	DisplayOrder int    `json:"display_order"`
	Title        string `json:"title"`
	Published    bool   `json:"published"`
	Boundary     string `json:"boundary"`
}

type Content struct {
	ReportType      string          `json:"report_type"`
	Title           string          `json:"title"`
	Status          string          `json:"status"`
	Simulation      bool            `json:"simulation"`
	GeneratedAt     time.Time       `json:"generated_at"`
	Timezone        string          `json:"timezone"`
	PublishedLayers []string        `json:"published_layers"`
	Statistics      Statistics      `json:"statistics"`
	ReportCards     []ReportCard    `json:"report_cards"`
	Geopolitics     Layer           `json:"geopolitics"`
	Macroeconomics  Layer           `json:"macroeconomics"`
	IndustryChains  []IndustryChain `json:"industry_chains"`
	Company         CompanyBoundary `json:"company"`
}

type Record struct {
	ID              string
	SourceReportID  string
	ContractVersion string
	ContentHash     string
	Content         Content
	PublishedAt     time.Time
}

type EvidenceLink struct {
	ID           string
	ReportID     string
	EvidenceID   string
	ScopeType    ScopeType
	ScopeKey     string
	Role         string
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
	ID              string
	SourceReportID  string
	ReportType      string
	Title           string
	Status          string
	Simulation      bool
	GeneratedAt     time.Time
	Timezone        string
	PublishedLayers []string
	Statistics      Statistics
	PublishedAt     time.Time
}

type IndustryChainSummary struct {
	Key           string     `json:"key"`
	DisplayOrder  int        `json:"display_order"`
	Name          string     `json:"name"`
	Conclusion    string     `json:"conclusion"`
	Status        string     `json:"status"`
	Result        Result     `json:"result"`
	Confidence    Confidence `json:"confidence"`
	TimeWindow    string     `json:"time_window"`
	EvidenceCount int        `json:"evidence_count"`
}

type Home struct {
	Report         Summary
	ReportCards    []ReportCard
	Company        CompanyBoundary
	EvidenceCounts map[TargetReference]int
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
	GetIndustryChain(context.Context, string, string) (Summary, IndustryChain, error)
	ReportScopeExists(context.Context, string, ScopeType, string) (bool, bool, error)
	ListEvidence(context.Context, string, ScopeType, string) ([]Evidence, error)
}

var (
	ErrPublicationConflict   = errors.New("Report source identity conflicts with another payload")
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

func (s *UseCase) Publish(ctx context.Context, contractVersion, sourceReportID string, content Content) (PublicationResult, error) {
	if s == nil || s.store == nil {
		return PublicationResult{}, errors.New("Report store is required")
	}
	if contractVersion != ContractVersion {
		return PublicationResult{}, invalid("contract_version", "must equal "+ContractVersion)
	}
	if err := requiredText("source_report_id", sourceReportID, 200); err != nil {
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
		if err := tx.Lock(ctx, sourceReportID); err != nil {
			return err
		}
		existing, err := tx.ReportBySourceID(ctx, sourceReportID)
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
			ID: reportID, SourceReportID: sourceReportID, ContractVersion: contractVersion,
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
// Report. source_report_id is intentionally not part of this content hash.
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

func (s *UseCase) ListEvidence(ctx context.Context, reportID string, scopeType ScopeType, scopeKey string) ([]Evidence, error) {
	if err := validateReportID(reportID); err != nil {
		return nil, err
	}
	if !validScopeType(scopeType) {
		return nil, invalid("scope_type", "is not supported")
	}
	if scopeKey != strings.TrimSpace(scopeKey) || !localKeyPattern.MatchString(scopeKey) {
		return nil, invalid("scope_key", "must be a lowercase Report-local key")
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

var localKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)

func ValidateContent(content Content) error {
	for _, field := range []struct {
		path  string
		value string
		max   int
	}{
		{"content.report_type", content.ReportType, 100},
		{"content.title", content.Title, 500},
		{"content.status", content.Status, 100},
		{"content.timezone", content.Timezone, 100},
	} {
		if err := requiredText(field.path, field.value, field.max); err != nil {
			return err
		}
	}
	if content.GeneratedAt.IsZero() {
		return invalid("content.generated_at", "must be a timestamp")
	}
	if err := validateStringSet("content.published_layers", content.PublishedLayers,
		map[string]struct{}{"geopolitics": {}, "macroeconomics": {}, "industry_chain": {}, "company": {}}); err != nil {
		return err
	}
	if err := validateStatistics(content.Statistics); err != nil {
		return err
	}
	if content.IndustryChains == nil {
		return invalid("content.industry_chains", "must be an array")
	}
	if content.ReportCards == nil || len(content.ReportCards) == 0 {
		return invalid("content.report_cards", "must contain persisted cards")
	}
	if content.Geopolitics.Key != "geopolitics" || content.Geopolitics.DisplayOrder != 1 {
		return invalid("content.geopolitics", "must use key geopolitics and display_order 1")
	}
	if content.Macroeconomics.Key != "macroeconomics" || content.Macroeconomics.DisplayOrder != 2 {
		return invalid("content.macroeconomics", "must use key macroeconomics and display_order 2")
	}
	if err := validateLayer("content.geopolitics", content.Geopolitics); err != nil {
		return err
	}
	if err := validateLayer("content.macroeconomics", content.Macroeconomics); err != nil {
		return err
	}
	if err := validateOrdered("content.industry_chains", len(content.IndustryChains), func(index int) int {
		return content.IndustryChains[index].DisplayOrder
	}); err != nil {
		return err
	}
	for index, chain := range content.IndustryChains {
		if err := validateIndustryChain(fmt.Sprintf("content.industry_chains[%d]", index), chain); err != nil {
			return err
		}
	}
	if err := validateCompany(content.Company); err != nil {
		return err
	}
	if content.Statistics.GeopoliticAnchorCount != len(content.Geopolitics.Anchors) ||
		content.Statistics.MacroeconomicAnchorCount != len(content.Macroeconomics.Anchors) ||
		content.Statistics.IndustryChainCount != len(content.IndustryChains) {
		return invalid("content.statistics", "structural counts must match the published snapshot")
	}
	index, err := buildContentIndex(content)
	if err != nil {
		return err
	}
	if err := validateRelatedReferences(content, index); err != nil {
		return err
	}
	return validateReportCards(content, index)
}

func ValidateLayerSnapshot(layer Layer) error {
	if layer.Key == "geopolitics" && layer.DisplayOrder != 1 ||
		layer.Key == "macroeconomics" && layer.DisplayOrder != 2 ||
		layer.Key != "geopolitics" && layer.Key != "macroeconomics" {
		return invalid("layer", "does not use a fixed layer key and display order")
	}
	return validateLayer("layer", layer)
}

func ValidateIndustryChainSnapshot(chain IndustryChain) error {
	return validateIndustryChain("industry_chain", chain)
}

type contentIndex struct {
	layers      map[string]Layer
	anchors     map[string]Anchor
	anchorLayer map[string]string
	chains      map[string]IndustryChain
	nodes       map[string]IndustryChainNode
	nodeChain   map[string]string
	scopeKeys   map[ScopeType]map[string]struct{}
}

func buildContentIndex(content Content) (contentIndex, error) {
	index := contentIndex{
		layers: map[string]Layer{}, anchors: map[string]Anchor{}, anchorLayer: map[string]string{},
		chains: map[string]IndustryChain{}, nodes: map[string]IndustryChainNode{}, nodeChain: map[string]string{},
		scopeKeys: map[ScopeType]map[string]struct{}{},
	}
	for _, scope := range []ScopeType{ScopeReportCard, ScopeLayer, ScopeAnchor, ScopeReasoningStep,
		ScopeTransmissionPath, ScopeCandidateMechanism, ScopeIndustryChain, ScopeIndustryChainNode} {
		index.scopeKeys[scope] = map[string]struct{}{}
	}
	add := func(scope ScopeType, key, path string) error {
		if !localKeyPattern.MatchString(key) {
			return invalid(path, "must be a lowercase Report-local key")
		}
		if _, duplicate := index.scopeKeys[scope][key]; duplicate {
			return invalid(path, "duplicates a key in the same Report scope type")
		}
		index.scopeKeys[scope][key] = struct{}{}
		return nil
	}
	for cardIndex, card := range content.ReportCards {
		if err := add(ScopeReportCard, card.Key, fmt.Sprintf("content.report_cards[%d].key", cardIndex)); err != nil {
			return contentIndex{}, err
		}
	}
	for _, layer := range []Layer{content.Geopolitics, content.Macroeconomics} {
		if err := add(ScopeLayer, layer.Key, "content."+layer.Key+".key"); err != nil {
			return contentIndex{}, err
		}
		index.layers[layer.Key] = layer
		for itemIndex, anchor := range layer.Anchors {
			if err := add(ScopeAnchor, anchor.Key, fmt.Sprintf("content.%s.anchors[%d].key", layer.Key, itemIndex)); err != nil {
				return contentIndex{}, err
			}
			index.anchors[anchor.Key], index.anchorLayer[anchor.Key] = anchor, layer.Key
		}
		for itemIndex, step := range layer.ReasoningSteps {
			if err := add(ScopeReasoningStep, step.Key, fmt.Sprintf("content.%s.reasoning_steps[%d].key", layer.Key, itemIndex)); err != nil {
				return contentIndex{}, err
			}
		}
		for itemIndex, path := range layer.DownwardTransmission.PublishedPaths {
			if err := add(ScopeTransmissionPath, path.Key, fmt.Sprintf("content.%s.downward_transmission.published_paths[%d].key", layer.Key, itemIndex)); err != nil {
				return contentIndex{}, err
			}
		}
		for itemIndex, candidate := range layer.DownwardTransmission.CandidateMechanisms {
			if err := add(ScopeCandidateMechanism, candidate.Key, fmt.Sprintf("content.%s.downward_transmission.candidate_mechanisms[%d].key", layer.Key, itemIndex)); err != nil {
				return contentIndex{}, err
			}
		}
	}
	for chainIndex, chain := range content.IndustryChains {
		if err := add(ScopeIndustryChain, chain.Key, fmt.Sprintf("content.industry_chains[%d].key", chainIndex)); err != nil {
			return contentIndex{}, err
		}
		index.chains[chain.Key] = chain
		for nodeIndex, node := range chain.Nodes {
			if err := add(ScopeIndustryChainNode, node.Key, fmt.Sprintf("content.industry_chains[%d].nodes[%d].key", chainIndex, nodeIndex)); err != nil {
				return contentIndex{}, err
			}
			index.nodes[node.Key], index.nodeChain[node.Key] = node, chain.Key
		}
	}
	return index, nil
}

func validateLayer(path string, layer Layer) error {
	for _, field := range []struct{ name, value string }{
		{"title", layer.Title}, {"conclusion", layer.Conclusion}, {"time_window", layer.TimeWindow},
		{"downward_transmission.summary", layer.DownwardTransmission.Summary},
	} {
		if err := requiredText(path+"."+field.name, field.value, 10_000); err != nil {
			return err
		}
	}
	if err := validateResult(path+".result", layer.Result); err != nil {
		return err
	}
	if err := validateConfidence(path+".confidence", layer.Confidence); err != nil {
		return err
	}
	if layer.Anchors == nil || layer.ReasoningSteps == nil || layer.RelatedAnchorKeys == nil ||
		layer.RelatedChainKeys == nil || layer.DownwardTransmission.PublishedPaths == nil ||
		layer.DownwardTransmission.CandidateMechanisms == nil || layer.DownwardTransmission.BoundaryNotes == nil ||
		layer.Uncertainty.Checkpoints == nil || layer.EvidenceRefs == nil {
		return invalid(path, "all collections must be arrays")
	}
	if err := validateOrdered(path+".anchors", len(layer.Anchors), func(index int) int { return layer.Anchors[index].DisplayOrder }); err != nil {
		return err
	}
	for index, anchor := range layer.Anchors {
		anchorPath := fmt.Sprintf("%s.anchors[%d]", path, index)
		for _, field := range []struct{ name, value string }{{"name", anchor.Name}, {"current_state", anchor.CurrentState}, {"reasoning", anchor.Reasoning}, {"time_window", anchor.TimeWindow}} {
			if err := requiredText(anchorPath+"."+field.name, field.value, 10_000); err != nil {
				return err
			}
		}
		if err := validateResult(anchorPath+".result", anchor.Result); err != nil {
			return err
		}
		if err := validateNature(anchorPath+".nature", anchor.Nature); err != nil {
			return err
		}
		if err := validateConfidence(anchorPath+".confidence", anchor.Confidence); err != nil {
			return err
		}
		if err := validateEvidenceRefs(anchorPath+".evidence_refs", anchor.EvidenceRefs); err != nil {
			return err
		}
	}
	if err := validateOrdered(path+".reasoning_steps", len(layer.ReasoningSteps), func(index int) int { return layer.ReasoningSteps[index].DisplayOrder }); err != nil {
		return err
	}
	for index, step := range layer.ReasoningSteps {
		stepPath := fmt.Sprintf("%s.reasoning_steps[%d]", path, index)
		for _, field := range []struct{ name, value string }{{"input", step.Input}, {"mechanism", step.Mechanism}, {"output", step.Output}, {"type", step.Type}} {
			if err := requiredText(stepPath+"."+field.name, field.value, 10_000); err != nil {
				return err
			}
		}
		if err := validateConfidence(stepPath+".confidence", step.Confidence); err != nil {
			return err
		}
		if err := validateEvidenceRefs(stepPath+".evidence_refs", step.EvidenceRefs); err != nil {
			return err
		}
	}
	if err := validateOrdered(path+".downward_transmission.published_paths", len(layer.DownwardTransmission.PublishedPaths), func(index int) int {
		return layer.DownwardTransmission.PublishedPaths[index].DisplayOrder
	}); err != nil {
		return err
	}
	for index, transmission := range layer.DownwardTransmission.PublishedPaths {
		transmissionPath := fmt.Sprintf("%s.downward_transmission.published_paths[%d]", path, index)
		for _, field := range []struct{ name, value string }{{"source_conclusion", transmission.SourceConclusion}, {"logic", transmission.Logic}, {"relation_nature", transmission.RelationNature}, {"evidence_role", transmission.EvidenceRole}, {"status", transmission.Status}} {
			if err := requiredText(transmissionPath+"."+field.name, field.value, 10_000); err != nil {
				return err
			}
		}
		if transmission.TargetRefs == nil || len(transmission.TargetRefs) == 0 {
			return invalid(transmissionPath+".target_refs", "must contain at least one structured target")
		}
		seenTargets := map[TargetReference]struct{}{}
		for targetIndex, target := range transmission.TargetRefs {
			targetPath := fmt.Sprintf("%s.target_refs[%d]", transmissionPath, targetIndex)
			if !validTargetReference(target.Ref) {
				return invalid(targetPath+".ref", "must use a supported type and lowercase Report-local key")
			}
			if _, duplicate := seenTargets[target.Ref]; duplicate {
				return invalid(targetPath+".ref", "duplicates a target in this path")
			}
			seenTargets[target.Ref] = struct{}{}
			if err := requiredText(targetPath+".label", target.Label, 500); err != nil {
				return err
			}
			if err := validateResult(targetPath+".result", target.Result); err != nil {
				return err
			}
		}
		if err := validateConfidence(transmissionPath+".confidence", transmission.Confidence); err != nil {
			return err
		}
		if err := validateEvidenceRefs(transmissionPath+".evidence_refs", transmission.EvidenceRefs); err != nil {
			return err
		}
	}
	if err := validateOrdered(path+".downward_transmission.candidate_mechanisms", len(layer.DownwardTransmission.CandidateMechanisms), func(index int) int {
		return layer.DownwardTransmission.CandidateMechanisms[index].DisplayOrder
	}); err != nil {
		return err
	}
	for index, candidate := range layer.DownwardTransmission.CandidateMechanisms {
		candidatePath := fmt.Sprintf("%s.downward_transmission.candidate_mechanisms[%d]", path, index)
		if err := requiredText(candidatePath+".mechanism", candidate.Mechanism, 10_000); err != nil {
			return err
		}
		if err := optionalText(candidatePath+".evidence_gap", candidate.EvidenceGap, 10_000); err != nil {
			return err
		}
		if err := validateConfidence(candidatePath+".confidence", candidate.Confidence); err != nil {
			return err
		}
		if err := validateEvidenceRefs(candidatePath+".evidence_refs", candidate.EvidenceRefs); err != nil {
			return err
		}
	}
	for index, note := range layer.DownwardTransmission.BoundaryNotes {
		if err := requiredText(fmt.Sprintf("%s.downward_transmission.boundary_notes[%d]", path, index), note, 10_000); err != nil {
			return err
		}
	}
	if err := validateUncertainty(path+".uncertainty", layer.Uncertainty); err != nil {
		return err
	}
	return validateEvidenceRefs(path+".evidence_refs", layer.EvidenceRefs)
}

func validTargetReference(ref TargetReference) bool {
	if !localKeyPattern.MatchString(ref.Key) {
		return false
	}
	switch ref.Type {
	case TargetLayer, TargetAnchor, TargetIndustryChain, TargetIndustryChainNode:
		return true
	default:
		return false
	}
}

func validateIndustryChain(path string, chain IndustryChain) error {
	if !localKeyPattern.MatchString(chain.ClaimKey) {
		return invalid(path+".claim_key", "must be a lowercase Report-local key")
	}
	for _, field := range []struct{ name, value string }{{"name", chain.Name}, {"conclusion", chain.Conclusion}, {"status", chain.Status}, {"time_window", chain.TimeWindow}} {
		if err := requiredText(path+"."+field.name, field.value, 10_000); err != nil {
			return err
		}
	}
	if err := validateResult(path+".result", chain.Result); err != nil {
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
	if chain.Nodes == nil || chain.Edges == nil || chain.Uncertainty.Checkpoints == nil || chain.EvidenceRefs == nil {
		return invalid(path, "all collections must be arrays")
	}
	if err := validateOrdered(path+".nodes", len(chain.Nodes), func(index int) int { return chain.Nodes[index].DisplayOrder }); err != nil {
		return err
	}
	nodes := map[string]struct{}{}
	for index, node := range chain.Nodes {
		nodePath := fmt.Sprintf("%s.nodes[%d]", path, index)
		if !localKeyPattern.MatchString(node.Key) {
			return invalid(nodePath+".key", "must be a lowercase Report-local key")
		}
		if _, duplicate := nodes[node.Key]; duplicate {
			return invalid(nodePath+".key", "duplicates a node in this industry chain")
		}
		nodes[node.Key] = struct{}{}
		for _, field := range []struct{ name, value string }{{"name", node.Name}, {"impact", node.Impact}, {"reasoning", node.Reasoning}, {"time_window", node.TimeWindow}} {
			if err := requiredText(nodePath+"."+field.name, field.value, 10_000); err != nil {
				return err
			}
		}
		if err := validateResult(nodePath+".result", node.Result); err != nil {
			return err
		}
		if err := validateNature(nodePath+".nature", node.Nature); err != nil {
			return err
		}
		if err := validateConfidence(nodePath+".confidence", node.Confidence); err != nil {
			return err
		}
		if err := validateEvidenceRefs(nodePath+".evidence_refs", node.EvidenceRefs); err != nil {
			return err
		}
	}
	if err := validateOrdered(path+".edges", len(chain.Edges), func(index int) int { return chain.Edges[index].DisplayOrder }); err != nil {
		return err
	}
	edges := map[string]struct{}{}
	for index, edge := range chain.Edges {
		edgePath := fmt.Sprintf("%s.edges[%d]", path, index)
		if !localKeyPattern.MatchString(edge.Key) {
			return invalid(edgePath+".key", "must be a lowercase Report-local key")
		}
		if _, duplicate := edges[edge.Key]; duplicate {
			return invalid(edgePath+".key", "duplicates an edge in this industry chain")
		}
		edges[edge.Key] = struct{}{}
		if _, ok := nodes[edge.FromNodeKey]; !ok {
			return invalid(edgePath+".from_node_key", "must reference a node in the same industry chain")
		}
		if _, ok := nodes[edge.ToNodeKey]; !ok {
			return invalid(edgePath+".to_node_key", "must reference a node in the same industry chain")
		}
		if edge.FromNodeKey == edge.ToNodeKey {
			return invalid(edgePath, "must not be a self edge")
		}
		if err := requiredText(edgePath+".relation_label", edge.RelationLabel, 500); err != nil {
			return err
		}
	}
	if err := validateChainUncertainty(path+".uncertainty", chain.Uncertainty); err != nil {
		return err
	}
	return validateEvidenceRefs(path+".evidence_refs", chain.EvidenceRefs)
}

func validateRelatedReferences(content Content, index contentIndex) error {
	for _, layer := range []Layer{content.Geopolitics, content.Macroeconomics} {
		seenAnchors := map[string]struct{}{}
		for itemIndex, key := range layer.RelatedAnchorKeys {
			if _, duplicate := seenAnchors[key]; duplicate {
				return invalid(fmt.Sprintf("content.%s.related_anchor_keys[%d]", layer.Key, itemIndex), "duplicates another related anchor")
			}
			seenAnchors[key] = struct{}{}
			if _, exists := index.anchors[key]; !exists {
				return &ReferenceError{Path: fmt.Sprintf("content.%s.related_anchor_keys[%d]", layer.Key, itemIndex), Reference: key, Message: "does not identify an anchor"}
			}
		}
		seenChains := map[string]struct{}{}
		for itemIndex, key := range layer.RelatedChainKeys {
			if _, duplicate := seenChains[key]; duplicate {
				return invalid(fmt.Sprintf("content.%s.related_chain_keys[%d]", layer.Key, itemIndex), "duplicates another related industry chain")
			}
			seenChains[key] = struct{}{}
			if _, exists := index.chains[key]; !exists {
				return &ReferenceError{Path: fmt.Sprintf("content.%s.related_chain_keys[%d]", layer.Key, itemIndex), Reference: key, Message: "does not identify an industry chain"}
			}
		}
		for pathIndex, transmission := range layer.DownwardTransmission.PublishedPaths {
			for targetIndex, target := range transmission.TargetRefs {
				if !targetExists(index, target.Ref) {
					return &ReferenceError{Path: fmt.Sprintf("content.%s.downward_transmission.published_paths[%d].target_refs[%d].ref", layer.Key, pathIndex, targetIndex), Reference: string(target.Ref.Type) + ":" + target.Ref.Key, Message: "does not identify a Report target"}
				}
			}
		}
	}
	return nil
}

func validateReportCards(content Content, index contentIndex) error {
	if err := validateOrdered("content.report_cards", len(content.ReportCards), func(cardIndex int) int {
		return content.ReportCards[cardIndex].DisplayOrder
	}); err != nil {
		return err
	}
	details := map[TargetReference]struct{}{}
	for cardIndex, card := range content.ReportCards {
		path := fmt.Sprintf("content.report_cards[%d]", cardIndex)
		for _, field := range []struct{ name, value string }{{"title", card.Title}, {"subtitle", card.Subtitle}, {"conclusion", card.Conclusion}, {"time_window", card.TimeWindow}} {
			if err := requiredText(path+"."+field.name, field.value, 10_000); err != nil {
				return err
			}
		}
		if err := validateResult(path+".result", card.Result); err != nil {
			return err
		}
		if err := validateConfidence(path+".confidence", card.Confidence); err != nil {
			return err
		}
		if card.ImpactItems == nil || len(card.ImpactItems) == 0 {
			return invalid(path+".impact_items", "must contain at least one explicit impact item")
		}
		if err := validateEvidenceRefs(path+".evidence_refs", card.EvidenceRefs); err != nil {
			return err
		}
		if _, duplicate := details[card.DetailRef]; duplicate {
			return invalid(path+".detail_ref", "duplicates another card detail target")
		}
		details[card.DetailRef] = struct{}{}
		var detailTitle, detailConclusion, detailTimeWindow string
		var detailResult Result
		var detailConfidence Confidence
		switch card.Kind {
		case CardGeopolitics, CardMacroeconomics:
			expected := "geopolitics"
			if card.Kind == CardMacroeconomics {
				expected = "macroeconomics"
			}
			if card.DetailRef != (TargetReference{Type: TargetLayer, Key: expected}) {
				return invalid(path+".detail_ref", "does not match the card kind")
			}
			detail := index.layers[expected]
			detailTitle, detailConclusion, detailResult, detailConfidence, detailTimeWindow = detail.Title, detail.Conclusion, detail.Result, detail.Confidence, detail.TimeWindow
		case CardIndustryChain:
			if card.DetailRef.Type != TargetIndustryChain {
				return invalid(path+".detail_ref", "must identify an industry chain")
			}
			detail, exists := index.chains[card.DetailRef.Key]
			if !exists {
				return &ReferenceError{Path: path + ".detail_ref", Reference: card.DetailRef.Key, Message: "does not identify an industry chain"}
			}
			detailTitle, detailConclusion, detailResult, detailConfidence, detailTimeWindow = detail.Name, detail.Conclusion, detail.Result, detail.Confidence, detail.TimeWindow
		default:
			return invalid(path+".kind", "is not supported")
		}
		if card.Title != detailTitle || card.Conclusion != detailConclusion || card.Result != detailResult ||
			!sameConfidence(card.Confidence, detailConfidence) || card.TimeWindow != detailTimeWindow {
			return invalid(path, "does not match its detail snapshot")
		}
		seenImpact := map[TargetReference]struct{}{}
		for impactIndex, impact := range card.ImpactItems {
			impactPath := fmt.Sprintf("%s.impact_items[%d]", path, impactIndex)
			if _, duplicate := seenImpact[impact.Ref]; duplicate {
				return invalid(impactPath+".ref", "duplicates another impact item")
			}
			seenImpact[impact.Ref] = struct{}{}
			var name, timeWindow string
			var result Result
			var confidence Confidence
			switch card.Kind {
			case CardGeopolitics, CardMacroeconomics:
				anchor, exists := index.anchors[impact.Ref.Key]
				if impact.Ref.Type != TargetAnchor || !exists || index.anchorLayer[impact.Ref.Key] != card.DetailRef.Key {
					return invalid(impactPath+".ref", "must identify an anchor in the card layer")
				}
				name, result, confidence, timeWindow = anchor.Name, anchor.Result, anchor.Confidence, anchor.TimeWindow
			case CardIndustryChain:
				node, exists := index.nodes[impact.Ref.Key]
				if impact.Ref.Type != TargetIndustryChainNode || !exists || index.nodeChain[impact.Ref.Key] != card.DetailRef.Key {
					return invalid(impactPath+".ref", "must identify a node in the card industry chain")
				}
				name, result, confidence, timeWindow = node.Name, node.Result, node.Confidence, node.TimeWindow
			}
			if impact.Name != name || impact.Result != result || !sameConfidence(impact.Confidence, confidence) || impact.TimeWindow != timeWindow {
				return invalid(impactPath, "does not match its detail snapshot")
			}
		}
	}
	if len(content.IndustryChains) > 0 {
		hasIndustryCard := false
		for ref := range details {
			if ref.Type == TargetIndustryChain {
				hasIndustryCard = true
				break
			}
		}
		if !hasIndustryCard {
			return invalid("content.report_cards", "must contain an industry chain card when industry chains are published")
		}
	}
	for _, ref := range []TargetReference{{Type: TargetLayer, Key: "geopolitics"}, {Type: TargetLayer, Key: "macroeconomics"}} {
		if _, exists := details[ref]; !exists {
			return invalid("content.report_cards", "must contain exactly one card for both fixed layers")
		}
	}
	return nil
}

func targetExists(index contentIndex, ref TargetReference) bool {
	if !localKeyPattern.MatchString(ref.Key) {
		return false
	}
	switch ref.Type {
	case TargetLayer:
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

func validateStatistics(value Statistics) error {
	counts := []struct {
		name  string
		value int
	}{
		{"event_count", value.EventCount}, {"ordinary_fact_count", value.OrdinaryFactCount},
		{"signal_fact_count", value.SignalFactCount}, {"transmission_hypothesis_count", value.TransmissionHypothesisCount},
		{"remaining_topology_pending_count", value.RemainingTopologyPendingCount},
		{"adaptive_hard_max_hops", value.AdaptiveHardMaxHops}, {"adaptive_observed_max_hops", value.AdaptiveObservedMaxHops},
		{"adaptive_stopped_by_confidence", value.AdaptiveStoppedByConfidence},
		{"adaptive_stopped_by_no_unvisited_neighbor", value.AdaptiveStoppedByNoUnvisitedNeighbor},
		{"adaptive_rejected_below_inclusion", value.AdaptiveRejectedBelowInclusion},
		{"geopolitic_anchor_count", value.GeopoliticAnchorCount}, {"macroeconomic_anchor_count", value.MacroeconomicAnchorCount},
		{"signaled_chain_node_count", value.SignaledChainNodeCount}, {"industry_chain_count", value.IndustryChainCount},
		{"unmapped_chain_node_count", value.UnmappedChainNodeCount},
	}
	for _, count := range counts {
		if count.value < 0 {
			return invalid("content.statistics."+count.name, "must be non-negative")
		}
	}
	for _, threshold := range []struct {
		name  string
		value float64
	}{{"adaptive_inclusion_threshold", value.AdaptiveInclusionThreshold}, {"adaptive_continuation_threshold", value.AdaptiveContinuationThreshold}} {
		if math.IsNaN(threshold.value) || math.IsInf(threshold.value, 0) || threshold.value < 0 || threshold.value > 1 {
			return invalid("content.statistics."+threshold.name, "must be between 0 and 1")
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
	if err := requiredText(path+".label", value.Label, 100); err != nil {
		return err
	}
	if value.Score != nil && (math.IsNaN(*value.Score) || math.IsInf(*value.Score, 0) || *value.Score < 0 || *value.Score > 1) {
		return invalid(path+".score", "must be null or between 0 and 1")
	}
	return nil
}

func validateEvidenceRefs(path string, values []EvidenceReference) error {
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
		if err := requiredText(itemPath+".role", value.Role, 200); err != nil {
			return err
		}
	}
	return nil
}

func validateUncertainty(path string, value LayerUncertainty) error {
	for name, field := range map[string]*string{
		"counterevidence": value.Counterevidence, "evidence_gap": value.EvidenceGap,
		"boundary": value.Boundary, "reversal_condition": value.ReversalCondition,
	} {
		if err := optionalText(path+"."+name, field, 10_000); err != nil {
			return err
		}
	}
	return validateCheckpoints(path+".checkpoints", value.Checkpoints)
}

func validateChainUncertainty(path string, value ChainUncertainty) error {
	if err := optionalText(path+".counterevidence_and_gap", value.CounterevidenceAndGap, 10_000); err != nil {
		return err
	}
	if err := optionalText(path+".stop_condition", value.StopCondition, 10_000); err != nil {
		return err
	}
	return validateCheckpoints(path+".checkpoints", value.Checkpoints)
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
			return invalid(itemPath+".key", "must be a lowercase Report-local key")
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

func validateCompany(value CompanyBoundary) error {
	if value.Key != "company" {
		return invalid("content.company.key", "must equal company")
	}
	if value.DisplayOrder != 4 {
		return invalid("content.company.display_order", "must equal 4")
	}
	if value.Published {
		return invalid("content.company.published", "must be false in report-publication.v1")
	}
	if err := requiredText("content.company.title", value.Title, 500); err != nil {
		return err
	}
	return requiredText("content.company.boundary", value.Boundary, 10_000)
}

func validateOrdered(path string, length int, order func(int) int) error {
	for index := 0; index < length; index++ {
		if order(index) != index+1 {
			return invalid(fmt.Sprintf("%s[%d].display_order", path, index), "must be continuous from 1")
		}
	}
	return nil
}

func validateStringSet(path string, values []string, allowed map[string]struct{}) error {
	if values == nil || len(values) == 0 {
		return invalid(path, "must contain at least one value")
	}
	seen := map[string]struct{}{}
	for index, value := range values {
		if _, ok := allowed[value]; !ok {
			return invalid(fmt.Sprintf("%s[%d]", path, index), "is not supported")
		}
		if _, duplicate := seen[value]; duplicate {
			return invalid(fmt.Sprintf("%s[%d]", path, index), "duplicates another value")
		}
		seen[value] = struct{}{}
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

func sameConfidence(left, right Confidence) bool {
	if left.Label != right.Label || (left.Score == nil) != (right.Score == nil) {
		return false
	}
	return left.Score == nil || *left.Score == *right.Score
}

func buildEvidenceLinks(reportID string, content Content) ([]EvidenceLink, error) {
	type scopedRefs struct {
		typeName ScopeType
		key      string
		refs     []EvidenceReference
	}
	values := make([]scopedRefs, 0)
	for _, card := range content.ReportCards {
		values = append(values, scopedRefs{ScopeReportCard, card.Key, card.EvidenceRefs})
	}
	for _, layer := range []Layer{content.Geopolitics, content.Macroeconomics} {
		values = append(values, scopedRefs{ScopeLayer, layer.Key, layer.EvidenceRefs})
		for _, anchor := range layer.Anchors {
			values = append(values, scopedRefs{ScopeAnchor, anchor.Key, anchor.EvidenceRefs})
		}
		for _, step := range layer.ReasoningSteps {
			values = append(values, scopedRefs{ScopeReasoningStep, step.Key, step.EvidenceRefs})
		}
		for _, path := range layer.DownwardTransmission.PublishedPaths {
			values = append(values, scopedRefs{ScopeTransmissionPath, path.Key, path.EvidenceRefs})
		}
		for _, candidate := range layer.DownwardTransmission.CandidateMechanisms {
			values = append(values, scopedRefs{ScopeCandidateMechanism, candidate.Key, candidate.EvidenceRefs})
		}
	}
	for _, chain := range content.IndustryChains {
		values = append(values, scopedRefs{ScopeIndustryChain, chain.Key, chain.EvidenceRefs})
		for _, node := range chain.Nodes {
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
	case ScopeReportCard, ScopeLayer, ScopeAnchor, ScopeReasoningStep, ScopeTransmissionPath,
		ScopeCandidateMechanism, ScopeIndustryChain, ScopeIndustryChainNode:
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
