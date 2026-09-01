package report

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	biz "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/biz/report"
	dataapi "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/data"
)

const reportsPath = dataapi.DataAPIPrefix + "/reports"

var (
	reportIDPattern   = regexp.MustCompile(`^RPT[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	evidenceIDPattern = regexp.MustCompile(`^EVD[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	localKeyPattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)
	metadataPattern   = regexp.MustCompile(`^[A-Za-z0-9_.:-]+$`)
)

type Repository struct {
	client *dataapi.HTTPClient
}

func NewRepository(client *dataapi.HTTPClient) (*Repository, error) {
	if client == nil {
		return nil, errors.New("Report Data API client is required")
	}
	return &Repository{client: client}, nil
}

func (r *Repository) ListReports(ctx context.Context, query biz.ListQuery) (biz.Page, error) {
	values := make(url.Values)
	if query.PublishedFrom != nil {
		values.Set("published_from", query.PublishedFrom.UTC().Format(time.RFC3339Nano))
	}
	if query.PublishedTo != nil {
		values.Set("published_to", query.PublishedTo.UTC().Format(time.RFC3339Nano))
	}
	if query.Limit > 0 {
		values.Set("limit", strconv.Itoa(query.Limit))
	}
	if query.Cursor != "" {
		values.Set("cursor", query.Cursor)
	}
	path := reportsPath
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var wire wirePage
	if err := r.get(ctx, path, &wire); err != nil {
		return biz.Page{}, mapReadError(err, readList)
	}
	page, err := mapPage(wire)
	if err != nil {
		return biz.Page{}, err
	}
	if query.Limit > 0 && len(page.Items) > query.Limit {
		return biz.Page{}, biz.ErrDataUnavailable
	}
	for _, item := range page.Items {
		if query.PublishedFrom != nil && item.PublishedAt.Before(*query.PublishedFrom) {
			return biz.Page{}, biz.ErrDataUnavailable
		}
		if query.PublishedTo != nil && !item.PublishedAt.Before(*query.PublishedTo) {
			return biz.Page{}, biz.ErrDataUnavailable
		}
	}
	return page, nil
}

func (r *Repository) GetHome(ctx context.Context, reportID string) (biz.Home, error) {
	var wire wireHome
	if err := r.get(ctx, reportsPath+"/"+url.PathEscape(reportID)+"/home", &wire); err != nil {
		return biz.Home{}, mapReadError(err, readHome)
	}
	return mapHome(wire, reportID)
}

func (r *Repository) GetLayer(ctx context.Context, reportID, layerKey string) (biz.LayerDetail, error) {
	path := reportsPath + "/" + url.PathEscape(reportID) + "/layers/" + url.PathEscape(layerKey)
	var wire wireLayerDetail
	if err := r.get(ctx, path, &wire); err != nil {
		return biz.LayerDetail{}, mapReadError(err, readLayer)
	}
	return mapLayerDetail(wire, reportID, layerKey)
}

func (r *Repository) GetIndustryChain(ctx context.Context, reportID, chainKey string) (biz.IndustryChainDetail, error) {
	path := reportsPath + "/" + url.PathEscape(reportID) + "/industry-chains/" + url.PathEscape(chainKey)
	var wire wireIndustryChainDetail
	if err := r.get(ctx, path, &wire); err != nil {
		return biz.IndustryChainDetail{}, mapReadError(err, readChain)
	}
	return mapIndustryChainDetail(wire, reportID, chainKey)
}

func (r *Repository) ListEvidences(ctx context.Context, reportID string, scope biz.EvidenceScope) (biz.EvidenceCollection, error) {
	values := url.Values{"scope_type": []string{scope.Type}, "scope_key": []string{scope.Key}}
	path := reportsPath + "/" + url.PathEscape(reportID) + "/evidences?" + values.Encode()
	var wire wireEvidenceCollection
	if err := r.get(ctx, path, &wire); err != nil {
		return biz.EvidenceCollection{}, mapReadError(err, readEvidence)
	}
	return mapEvidenceCollection(wire, reportID, scope)
}

func (r *Repository) get(ctx context.Context, path string, target any) error {
	if r == nil || r.client == nil {
		return biz.ErrDataUnavailable
	}
	var envelope strictEnvelope
	if err := r.client.GetJSON(ctx, path, &envelope); err != nil {
		return err
	}
	if !validMetadata(envelope.RequestID, 128) || len(envelope.Result) == 0 || bytes.Equal(envelope.Result, []byte("null")) {
		return biz.ErrDataUnavailable
	}
	if err := decodeExact(envelope.Result, target); err != nil {
		return biz.ErrDataUnavailable
	}
	return nil
}

type strictEnvelope struct {
	RequestID string
	Result    json.RawMessage
}

func (e *strictEnvelope) UnmarshalJSON(payload []byte) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(payload, &object); err != nil {
		return err
	}
	if len(object) != 2 || object["request_id"] == nil || object["result"] == nil {
		return errors.New("invalid Data response envelope")
	}
	if err := json.Unmarshal(object["request_id"], &e.RequestID); err != nil {
		return err
	}
	e.Result = append(e.Result[:0], object["result"]...)
	return nil
}

func decodeExact(payload []byte, target any) error {
	if err := validateRequiredJSON(payload, reflect.TypeOf(target)); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("Data response has trailing JSON")
	}
	return nil
}

func validateRequiredJSON(payload json.RawMessage, targetType reflect.Type) error {
	for targetType.Kind() == reflect.Pointer {
		if bytes.Equal(bytes.TrimSpace(payload), []byte("null")) {
			return nil
		}
		targetType = targetType.Elem()
	}
	if bytes.Equal(bytes.TrimSpace(payload), []byte("null")) {
		return errors.New("required Data field is null")
	}
	switch targetType.Kind() {
	case reflect.Struct:
		var object map[string]json.RawMessage
		if err := json.Unmarshal(payload, &object); err != nil || object == nil {
			return errors.New("required Data object is invalid")
		}
		for fieldIndex := 0; fieldIndex < targetType.NumField(); fieldIndex++ {
			field := targetType.Field(fieldIndex)
			if !field.IsExported() {
				continue
			}
			name := strings.Split(field.Tag.Get("json"), ",")[0]
			if name == "-" {
				continue
			}
			if name == "" {
				name = field.Name
			}
			fieldPayload, exists := object[name]
			if !exists {
				return errors.New("required Data field is missing")
			}
			if err := validateRequiredJSON(fieldPayload, field.Type); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		var items []json.RawMessage
		if err := json.Unmarshal(payload, &items); err != nil || items == nil {
			return errors.New("required Data array is invalid")
		}
		for _, item := range items {
			if err := validateRequiredJSON(item, targetType.Elem()); err != nil {
				return err
			}
		}
	}
	return nil
}

type readOperation int

const (
	readList readOperation = iota
	readHome
	readLayer
	readChain
	readEvidence
)

func mapReadError(err error, operation readOperation) error {
	if errors.Is(err, biz.ErrDataUnavailable) {
		return biz.ErrDataUnavailable
	}
	var clientError *dataapi.Error
	if !errors.As(err, &clientError) {
		return biz.ErrDataUnavailable
	}
	switch clientError.Code {
	case "REPORT_NOT_FOUND":
		return biz.ErrReportNotFound
	case "REPORT_LAYER_NOT_FOUND":
		return biz.ErrLayerNotFound
	case "REPORT_INDUSTRY_CHAIN_NOT_FOUND":
		return biz.ErrChainNotFound
	case "REPORT_EVIDENCE_SCOPE_NOT_FOUND":
		if operation == readEvidence {
			return biz.ErrEvidenceScopeNotFound
		}
	}
	return biz.ErrDataUnavailable
}

type wireStatistics struct {
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

type wireSummary struct {
	ID              string         `json:"id"`
	SourceReportID  string         `json:"source_report_id"`
	ReportType      string         `json:"report_type"`
	Title           string         `json:"title"`
	Status          string         `json:"status"`
	Simulation      bool           `json:"simulation"`
	GeneratedAt     string         `json:"generated_at"`
	Timezone        string         `json:"timezone"`
	PublishedLayers []string       `json:"published_layers"`
	Statistics      wireStatistics `json:"statistics"`
	PublishedAt     string         `json:"published_at"`
}

type wirePage struct {
	Items      []wireSummary `json:"items"`
	NextCursor *string       `json:"next_cursor"`
}

type wireResult struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

type wireNature struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

type wireConfidence struct {
	Label string   `json:"label"`
	Score *float64 `json:"score"`
}

type wireReference struct {
	Type string `json:"type"`
	Key  string `json:"key"`
}

type wireCardImpactItem struct {
	Ref           wireReference  `json:"ref"`
	Name          string         `json:"name"`
	Result        wireResult     `json:"result"`
	Confidence    wireConfidence `json:"confidence"`
	TimeWindow    string         `json:"time_window"`
	EvidenceCount int            `json:"evidence_count"`
}

type wireCard struct {
	Key           string               `json:"key"`
	Kind          string               `json:"kind"`
	DisplayOrder  int                  `json:"display_order"`
	DetailRef     wireReference        `json:"detail_ref"`
	Title         string               `json:"title"`
	Subtitle      string               `json:"subtitle"`
	Conclusion    string               `json:"conclusion"`
	Result        wireResult           `json:"result"`
	Confidence    wireConfidence       `json:"confidence"`
	TimeWindow    string               `json:"time_window"`
	ImpactItems   []wireCardImpactItem `json:"impact_items"`
	EvidenceCount int                  `json:"evidence_count"`
}

type wireCompanyBoundary struct {
	Key          string `json:"key"`
	DisplayOrder int    `json:"display_order"`
	Title        string `json:"title"`
	Published    bool   `json:"published"`
	Boundary     string `json:"boundary"`
}

type wireHome struct {
	Report      wireSummary         `json:"report"`
	ReportCards []wireCard          `json:"report_cards"`
	Company     wireCompanyBoundary `json:"company"`
}

type wireAnchor struct {
	Key           string         `json:"key"`
	DisplayOrder  int            `json:"display_order"`
	Name          string         `json:"name"`
	CurrentState  string         `json:"current_state"`
	Result        wireResult     `json:"result"`
	Nature        wireNature     `json:"nature"`
	Reasoning     string         `json:"reasoning"`
	TimeWindow    string         `json:"time_window"`
	Confidence    wireConfidence `json:"confidence"`
	EvidenceCount int            `json:"evidence_count"`
}

type wireReasoningStep struct {
	Key           string         `json:"key"`
	DisplayOrder  int            `json:"display_order"`
	Input         string         `json:"input"`
	Mechanism     string         `json:"mechanism"`
	Output        string         `json:"output"`
	Type          string         `json:"type"`
	Confidence    wireConfidence `json:"confidence"`
	EvidenceCount int            `json:"evidence_count"`
}

type wireTransmissionTarget struct {
	Ref    wireReference `json:"ref"`
	Label  string        `json:"label"`
	Result wireResult    `json:"result"`
}

type wireTransmissionPath struct {
	Key              string                   `json:"key"`
	DisplayOrder     int                      `json:"display_order"`
	SourceConclusion string                   `json:"source_conclusion"`
	TargetRefs       []wireTransmissionTarget `json:"target_refs"`
	Logic            string                   `json:"logic"`
	RelationNature   string                   `json:"relation_nature"`
	EvidenceRole     string                   `json:"evidence_role"`
	Confidence       wireConfidence           `json:"confidence"`
	Status           string                   `json:"status"`
	EvidenceCount    int                      `json:"evidence_count"`
}

type wireCandidateMechanism struct {
	Key           string         `json:"key"`
	DisplayOrder  int            `json:"display_order"`
	Mechanism     string         `json:"mechanism"`
	EvidenceGap   *string        `json:"evidence_gap"`
	Confidence    wireConfidence `json:"confidence"`
	EvidenceCount int            `json:"evidence_count"`
}

type wireCheckpoint struct {
	Key          string `json:"key"`
	DisplayOrder int    `json:"display_order"`
	Summary      string `json:"summary"`
}

type wireDownwardTransmission struct {
	Summary             string                   `json:"summary"`
	PublishedPaths      []wireTransmissionPath   `json:"published_paths"`
	CandidateMechanisms []wireCandidateMechanism `json:"candidate_mechanisms"`
	BoundaryNotes       []string                 `json:"boundary_notes"`
}

type wireLayerUncertainty struct {
	Counterevidence   *string          `json:"counterevidence"`
	EvidenceGap       *string          `json:"evidence_gap"`
	Boundary          *string          `json:"boundary"`
	ReversalCondition *string          `json:"reversal_condition"`
	Checkpoints       []wireCheckpoint `json:"checkpoints"`
}

type wireLayer struct {
	Key                  string                   `json:"key"`
	DisplayOrder         int                      `json:"display_order"`
	Title                string                   `json:"title"`
	Conclusion           string                   `json:"conclusion"`
	Result               wireResult               `json:"result"`
	Confidence           wireConfidence           `json:"confidence"`
	TimeWindow           string                   `json:"time_window"`
	Anchors              []wireAnchor             `json:"anchors"`
	ReasoningSteps       []wireReasoningStep      `json:"reasoning_steps"`
	RelatedAnchorKeys    []string                 `json:"related_anchor_keys"`
	RelatedChainKeys     []string                 `json:"related_chain_keys"`
	DownwardTransmission wireDownwardTransmission `json:"downward_transmission"`
	Uncertainty          wireLayerUncertainty     `json:"uncertainty"`
	EvidenceCount        int                      `json:"evidence_count"`
}

type wireIndustryChainSummary struct {
	Key           string         `json:"key"`
	DisplayOrder  int            `json:"display_order"`
	Name          string         `json:"name"`
	Conclusion    string         `json:"conclusion"`
	Status        string         `json:"status"`
	Result        wireResult     `json:"result"`
	Confidence    wireConfidence `json:"confidence"`
	TimeWindow    string         `json:"time_window"`
	EvidenceCount int            `json:"evidence_count"`
}

type wireLayerDetail struct {
	Report                wireSummary                `json:"report"`
	Layer                 wireLayer                  `json:"layer"`
	RelatedIndustryChains []wireIndustryChainSummary `json:"related_industry_chains"`
}

type wireIndustryChainNode struct {
	Key           string         `json:"key"`
	DisplayOrder  int            `json:"display_order"`
	Name          string         `json:"name"`
	Impact        string         `json:"impact"`
	Result        wireResult     `json:"result"`
	Nature        wireNature     `json:"nature"`
	Reasoning     string         `json:"reasoning"`
	TimeWindow    string         `json:"time_window"`
	Confidence    wireConfidence `json:"confidence"`
	EvidenceCount int            `json:"evidence_count"`
}

type wireIndustryChainEdge struct {
	Key           string `json:"key"`
	DisplayOrder  int    `json:"display_order"`
	FromNodeKey   string `json:"from_node_key"`
	ToNodeKey     string `json:"to_node_key"`
	RelationLabel string `json:"relation_label"`
}

type wireChainUncertainty struct {
	CounterevidenceAndGap *string          `json:"counterevidence_and_gap"`
	StopCondition         *string          `json:"stop_condition"`
	Checkpoints           []wireCheckpoint `json:"checkpoints"`
}

type wireIndustryChain struct {
	Key                       string                  `json:"key"`
	ClaimKey                  string                  `json:"claim_key"`
	DisplayOrder              int                     `json:"display_order"`
	Name                      string                  `json:"name"`
	Conclusion                string                  `json:"conclusion"`
	Status                    string                  `json:"status"`
	Result                    wireResult              `json:"result"`
	Confidence                wireConfidence          `json:"confidence"`
	TimeWindow                string                  `json:"time_window"`
	PathSummary               *string                 `json:"path_summary"`
	AcceptedHypothesisSummary *string                 `json:"accepted_hypothesis_summary"`
	Nodes                     []wireIndustryChainNode `json:"nodes"`
	Edges                     []wireIndustryChainEdge `json:"edges"`
	Uncertainty               wireChainUncertainty    `json:"uncertainty"`
	EvidenceCount             int                     `json:"evidence_count"`
}

type wireIndustryChainDetail struct {
	Report        wireSummary       `json:"report"`
	IndustryChain wireIndustryChain `json:"industry_chain"`
}

type wireEvidenceItem struct {
	EvidenceID   string   `json:"evidence_id"`
	Role         string   `json:"role"`
	DisplayOrder int      `json:"display_order"`
	PublishedAt  *string  `json:"published_at"`
	Summary      string   `json:"summary"`
	Keywords     []string `json:"keywords"`
}

type wireEvidenceCollection struct {
	ReportID  string             `json:"report_id"`
	ScopeType string             `json:"scope_type"`
	ScopeKey  string             `json:"scope_key"`
	Items     []wireEvidenceItem `json:"items"`
}

func mapPage(wire wirePage) (biz.Page, error) {
	if wire.Items == nil {
		return biz.Page{}, biz.ErrDataUnavailable
	}
	items := make([]biz.Summary, len(wire.Items))
	for index, item := range wire.Items {
		mapped, err := mapSummary(item)
		if err != nil {
			return biz.Page{}, err
		}
		items[index] = mapped
	}
	if wire.NextCursor != nil && (!validText(*wire.NextCursor, 2048) || strings.TrimSpace(*wire.NextCursor) != *wire.NextCursor) {
		return biz.Page{}, biz.ErrDataUnavailable
	}
	return biz.Page{Items: items, NextCursor: wire.NextCursor}, nil
}

func mapSummary(wire wireSummary) (biz.Summary, error) {
	generatedAt, err := parseTimestamp(wire.GeneratedAt)
	if err != nil {
		return biz.Summary{}, biz.ErrDataUnavailable
	}
	publishedAt, err := parseTimestamp(wire.PublishedAt)
	if err != nil {
		return biz.Summary{}, biz.ErrDataUnavailable
	}
	if !reportIDPattern.MatchString(wire.ID) || !validText(wire.SourceReportID, 200) ||
		!validText(wire.ReportType, 100) || !validText(wire.Title, 500) || !validText(wire.Status, 100) ||
		!validText(wire.Timezone, 100) || !validPublishedLayers(wire.PublishedLayers) ||
		!validStatistics(wire.Statistics) {
		return biz.Summary{}, biz.ErrDataUnavailable
	}
	return biz.Summary{ID: wire.ID, SourceReportID: wire.SourceReportID, Title: wire.Title,
		GeneratedAt: generatedAt, PublishedAt: publishedAt}, nil
}

func mapHome(wire wireHome, expectedReportID string) (biz.Home, error) {
	summary, err := mapSummary(wire.Report)
	if err != nil || summary.ID != expectedReportID || wire.ReportCards == nil || len(wire.ReportCards) == 0 {
		return biz.Home{}, biz.ErrDataUnavailable
	}
	cards := make([]biz.Card, len(wire.ReportCards))
	seenKeys := make(map[string]struct{}, len(cards))
	seenDetails := make(map[string]struct{}, len(cards))
	layerCardCounts := map[string]int{
		biz.LayerGeopolitics:    0,
		biz.LayerMacroeconomics: 0,
	}
	for index, card := range wire.ReportCards {
		mapped, mapErr := mapCard(card)
		if mapErr != nil || card.DisplayOrder != index+1 {
			return biz.Home{}, biz.ErrDataUnavailable
		}
		if _, duplicate := seenKeys[mapped.Key]; duplicate {
			return biz.Home{}, biz.ErrDataUnavailable
		}
		seenKeys[mapped.Key] = struct{}{}
		detailIdentity := mapped.DetailRef.Type + ":" + mapped.DetailRef.Key
		if _, duplicate := seenDetails[detailIdentity]; duplicate {
			return biz.Home{}, biz.ErrDataUnavailable
		}
		seenDetails[detailIdentity] = struct{}{}
		if mapped.Kind == biz.LayerGeopolitics || mapped.Kind == biz.LayerMacroeconomics {
			layerCardCounts[mapped.Kind]++
		}
		cards[index] = mapped
	}
	if layerCardCounts[biz.LayerGeopolitics] != 1 || layerCardCounts[biz.LayerMacroeconomics] != 1 {
		return biz.Home{}, biz.ErrDataUnavailable
	}
	company := wire.Company
	if company.Key != "company" || company.DisplayOrder != 4 || company.Published ||
		!validText(company.Title, 500) || !validText(company.Boundary, 10_000) {
		return biz.Home{}, biz.ErrDataUnavailable
	}
	return biz.Home{Report: summary, Cards: cards, Company: biz.CompanyBoundary{
		Key: company.Key, DisplayOrder: company.DisplayOrder, Title: company.Title,
		Published: company.Published, Boundary: company.Boundary,
	}}, nil
}

func mapCard(wire wireCard) (biz.Card, error) {
	if !validLocalKey(wire.Key) || wire.DisplayOrder < 1 || wire.EvidenceCount < 0 ||
		wire.ImpactItems == nil || len(wire.ImpactItems) == 0 ||
		!validText(wire.Title, 10_000) || !validText(wire.Subtitle, 10_000) || !validText(wire.Conclusion, 10_000) ||
		!validText(wire.TimeWindow, 10_000) || !validCardKind(wire.Kind) {
		return biz.Card{}, biz.ErrDataUnavailable
	}
	detailRef, err := mapReference(wire.DetailRef)
	if err != nil || (detailRef.Type != biz.ScopeLayer && detailRef.Type != biz.ScopeIndustryChain) ||
		(wire.Kind == biz.LayerGeopolitics && detailRef != (biz.Reference{Type: biz.ScopeLayer, Key: biz.LayerGeopolitics})) ||
		(wire.Kind == biz.LayerMacroeconomics && detailRef != (biz.Reference{Type: biz.ScopeLayer, Key: biz.LayerMacroeconomics})) ||
		(wire.Kind == biz.ScopeIndustryChain && detailRef.Type != biz.ScopeIndustryChain) {
		return biz.Card{}, biz.ErrDataUnavailable
	}
	result, err := mapResult(wire.Result)
	if err != nil {
		return biz.Card{}, err
	}
	confidence, err := mapConfidence(wire.Confidence)
	if err != nil {
		return biz.Card{}, err
	}
	impacts := make([]biz.CardImpactItem, len(wire.ImpactItems))
	seenRefs := make(map[string]struct{}, len(impacts))
	for index, impact := range wire.ImpactItems {
		ref, refErr := mapReference(impact.Ref)
		validImpactType := (wire.Kind == biz.ScopeIndustryChain && ref.Type == biz.ScopeIndustryChainNode) ||
			(wire.Kind != biz.ScopeIndustryChain && ref.Type == biz.ScopeAnchor)
		if refErr != nil || !validImpactType ||
			!validText(impact.Name, 10_000) || !validText(impact.TimeWindow, 10_000) || impact.EvidenceCount < 0 {
			return biz.Card{}, biz.ErrDataUnavailable
		}
		identity := ref.Type + ":" + ref.Key
		if _, duplicate := seenRefs[identity]; duplicate {
			return biz.Card{}, biz.ErrDataUnavailable
		}
		seenRefs[identity] = struct{}{}
		impactResult, resultErr := mapResult(impact.Result)
		if resultErr != nil {
			return biz.Card{}, resultErr
		}
		impactConfidence, confidenceErr := mapConfidence(impact.Confidence)
		if confidenceErr != nil {
			return biz.Card{}, confidenceErr
		}
		impacts[index] = biz.CardImpactItem{Ref: ref, Name: impact.Name, Result: impactResult,
			Confidence: impactConfidence, TimeWindow: impact.TimeWindow, HasEvidence: impact.EvidenceCount > 0}
	}
	return biz.Card{Key: wire.Key, Kind: wire.Kind, Order: wire.DisplayOrder, DetailRef: detailRef,
		Title: wire.Title, Subtitle: wire.Subtitle, Conclusion: wire.Conclusion, Result: result,
		Confidence: confidence, TimeWindow: wire.TimeWindow, ImpactItems: impacts,
		HasEvidence: wire.EvidenceCount > 0}, nil
}

func mapLayerDetail(wire wireLayerDetail, expectedReportID, expectedLayerKey string) (biz.LayerDetail, error) {
	summary, err := mapSummary(wire.Report)
	if err != nil || summary.ID != expectedReportID || wire.Layer.Key != expectedLayerKey {
		return biz.LayerDetail{}, biz.ErrDataUnavailable
	}
	layer, err := mapLayer(wire.Layer)
	if err != nil {
		return biz.LayerDetail{}, err
	}
	if wire.RelatedIndustryChains == nil {
		return biz.LayerDetail{}, biz.ErrDataUnavailable
	}
	chains := make([]biz.IndustryChainSummary, len(wire.RelatedIndustryChains))
	chainKeys := make(map[string]struct{}, len(chains))
	for index, chain := range wire.RelatedIndustryChains {
		mapped, mapErr := mapChainSummary(chain)
		if mapErr != nil {
			return biz.LayerDetail{}, biz.ErrDataUnavailable
		}
		if _, duplicate := chainKeys[mapped.Key]; duplicate {
			return biz.LayerDetail{}, biz.ErrDataUnavailable
		}
		chainKeys[mapped.Key] = struct{}{}
		chains[index] = mapped
	}
	if len(layer.RelatedChainKeys) != len(chains) {
		return biz.LayerDetail{}, biz.ErrDataUnavailable
	}
	for index, key := range layer.RelatedChainKeys {
		if _, exists := chainKeys[key]; !exists || chains[index].Key != key {
			return biz.LayerDetail{}, biz.ErrDataUnavailable
		}
	}
	return biz.LayerDetail{Report: summary, Layer: layer, RelatedIndustryChains: chains}, nil
}

func mapLayer(wire wireLayer) (biz.Layer, error) {
	if !validLayer(wire.Key) || wire.DisplayOrder < 1 || !validText(wire.Title, 10_000) ||
		!validText(wire.Conclusion, 10_000) || !validText(wire.TimeWindow, 10_000) || wire.EvidenceCount < 0 ||
		wire.Anchors == nil || wire.ReasoningSteps == nil || wire.RelatedAnchorKeys == nil ||
		wire.RelatedChainKeys == nil || wire.DownwardTransmission.PublishedPaths == nil ||
		wire.DownwardTransmission.CandidateMechanisms == nil || wire.DownwardTransmission.BoundaryNotes == nil ||
		wire.Uncertainty.Checkpoints == nil {
		return biz.Layer{}, biz.ErrDataUnavailable
	}
	result, err := mapResult(wire.Result)
	if err != nil {
		return biz.Layer{}, err
	}
	confidence, err := mapConfidence(wire.Confidence)
	if err != nil {
		return biz.Layer{}, err
	}
	anchors := make([]biz.Anchor, len(wire.Anchors))
	anchorKeys := make(map[string]struct{}, len(anchors))
	for index, item := range wire.Anchors {
		if item.DisplayOrder != index+1 || !validLocalKey(item.Key) || !validText(item.Name, 10_000) ||
			!validText(item.CurrentState, 10_000) || !validText(item.Reasoning, 10_000) ||
			!validText(item.TimeWindow, 10_000) || item.EvidenceCount < 0 {
			return biz.Layer{}, biz.ErrDataUnavailable
		}
		if _, duplicate := anchorKeys[item.Key]; duplicate {
			return biz.Layer{}, biz.ErrDataUnavailable
		}
		anchorKeys[item.Key] = struct{}{}
		itemResult, resultErr := mapResult(item.Result)
		itemNature, natureErr := mapNature(item.Nature)
		itemConfidence, confidenceErr := mapConfidence(item.Confidence)
		if resultErr != nil || natureErr != nil || confidenceErr != nil {
			return biz.Layer{}, biz.ErrDataUnavailable
		}
		anchors[index] = biz.Anchor{Key: item.Key, DisplayOrder: item.DisplayOrder, Name: item.Name,
			CurrentState: item.CurrentState, Result: itemResult, Nature: itemNature, Reasoning: item.Reasoning,
			TimeWindow: item.TimeWindow, Confidence: itemConfidence,
			Scope:       biz.EvidenceScope{Type: biz.ScopeAnchor, Key: item.Key},
			HasEvidence: item.EvidenceCount > 0}
	}
	steps, err := mapReasoningSteps(wire.Key, wire.ReasoningSteps)
	if err != nil {
		return biz.Layer{}, err
	}
	paths, err := mapTransmissionPaths(wire.Key, wire.DownwardTransmission.PublishedPaths)
	if err != nil {
		return biz.Layer{}, err
	}
	candidates, err := mapCandidateMechanisms(wire.Key, wire.DownwardTransmission.CandidateMechanisms)
	if err != nil {
		return biz.Layer{}, err
	}
	checkpoints, err := mapCheckpoints(wire.Uncertainty.Checkpoints)
	if err != nil {
		return biz.Layer{}, err
	}
	if !validText(wire.DownwardTransmission.Summary, 10_000) ||
		!validStringArray(wire.DownwardTransmission.BoundaryNotes, 0, 10_000) ||
		!validNullableText(wire.Uncertainty.Counterevidence, 10_000) ||
		!validNullableText(wire.Uncertainty.EvidenceGap, 10_000) ||
		!validNullableText(wire.Uncertainty.Boundary, 10_000) ||
		!validNullableText(wire.Uncertainty.ReversalCondition, 10_000) ||
		!validLocalKeyArray(wire.RelatedAnchorKeys) || !validLocalKeyArray(wire.RelatedChainKeys) {
		return biz.Layer{}, biz.ErrDataUnavailable
	}
	return biz.Layer{Key: wire.Key, DisplayOrder: wire.DisplayOrder, Title: wire.Title,
		Conclusion: wire.Conclusion, Result: result, Confidence: confidence, TimeWindow: wire.TimeWindow,
		Anchors: anchors, ReasoningSteps: steps, RelatedAnchorKeys: wire.RelatedAnchorKeys,
		RelatedChainKeys: wire.RelatedChainKeys, DownwardTransmission: biz.DownwardTransmission{
			Summary: wire.DownwardTransmission.Summary, PublishedPaths: paths,
			CandidateMechanisms: candidates, BoundaryNotes: wire.DownwardTransmission.BoundaryNotes,
		}, Uncertainty: biz.LayerUncertainty{Counterevidence: wire.Uncertainty.Counterevidence,
			EvidenceGap: wire.Uncertainty.EvidenceGap, Boundary: wire.Uncertainty.Boundary,
			ReversalCondition: wire.Uncertainty.ReversalCondition, Checkpoints: checkpoints},
		Scope: biz.EvidenceScope{Type: biz.ScopeLayer, Key: wire.Key}, HasEvidence: wire.EvidenceCount > 0}, nil
}

func mapReasoningSteps(layerKey string, wire []wireReasoningStep) ([]biz.ReasoningStep, error) {
	items := make([]biz.ReasoningStep, len(wire))
	seen := make(map[string]struct{}, len(wire))
	for index, item := range wire {
		if item.DisplayOrder != index+1 || !validLocalKey(item.Key) || !validText(item.Input, 10_000) ||
			!validText(item.Mechanism, 10_000) || !validText(item.Output, 10_000) || !validText(item.Type, 10_000) || item.EvidenceCount < 0 {
			return nil, biz.ErrDataUnavailable
		}
		if _, duplicate := seen[item.Key]; duplicate {
			return nil, biz.ErrDataUnavailable
		}
		seen[item.Key] = struct{}{}
		confidence, err := mapConfidence(item.Confidence)
		if err != nil {
			return nil, err
		}
		items[index] = biz.ReasoningStep{Key: item.Key, DisplayOrder: item.DisplayOrder, Input: item.Input,
			Mechanism: item.Mechanism, Output: item.Output, Type: item.Type, Confidence: confidence,
			Scope:       biz.EvidenceScope{Type: biz.ScopeReasoningStep, Key: item.Key},
			HasEvidence: item.EvidenceCount > 0}
	}
	return items, nil
}

func mapTransmissionPaths(layerKey string, wire []wireTransmissionPath) ([]biz.TransmissionPath, error) {
	items := make([]biz.TransmissionPath, len(wire))
	seen := make(map[string]struct{}, len(wire))
	for index, item := range wire {
		if item.DisplayOrder != index+1 || !validLocalKey(item.Key) || !validText(item.SourceConclusion, 10_000) ||
			!validText(item.Logic, 10_000) || !validText(item.RelationNature, 10_000) ||
			!validText(item.EvidenceRole, 10_000) || !validText(item.Status, 10_000) ||
			item.EvidenceCount < 0 || item.TargetRefs == nil || len(item.TargetRefs) == 0 {
			return nil, biz.ErrDataUnavailable
		}
		if _, duplicate := seen[item.Key]; duplicate {
			return nil, biz.ErrDataUnavailable
		}
		seen[item.Key] = struct{}{}
		targets := make([]biz.TransmissionTarget, len(item.TargetRefs))
		seenTargets := make(map[string]struct{}, len(targets))
		for targetIndex, target := range item.TargetRefs {
			ref, err := mapReference(target.Ref)
			if err != nil || !validText(target.Label, 500) {
				return nil, biz.ErrDataUnavailable
			}
			identity := ref.Type + ":" + ref.Key
			if _, duplicate := seenTargets[identity]; duplicate {
				return nil, biz.ErrDataUnavailable
			}
			seenTargets[identity] = struct{}{}
			result, resultErr := mapResult(target.Result)
			if resultErr != nil {
				return nil, resultErr
			}
			targets[targetIndex] = biz.TransmissionTarget{Ref: ref, Label: target.Label, Result: result}
		}
		confidence, err := mapConfidence(item.Confidence)
		if err != nil {
			return nil, err
		}
		items[index] = biz.TransmissionPath{Key: item.Key, DisplayOrder: item.DisplayOrder,
			SourceConclusion: item.SourceConclusion, TargetRefs: targets, Logic: item.Logic,
			RelationNature: item.RelationNature, EvidenceRole: item.EvidenceRole, Confidence: confidence,
			Status: item.Status, Scope: biz.EvidenceScope{Type: biz.ScopeTransmissionPath, Key: item.Key},
			HasEvidence: item.EvidenceCount > 0}
	}
	return items, nil
}

func mapCandidateMechanisms(layerKey string, wire []wireCandidateMechanism) ([]biz.CandidateMechanism, error) {
	items := make([]biz.CandidateMechanism, len(wire))
	seen := make(map[string]struct{}, len(wire))
	for index, item := range wire {
		if item.DisplayOrder != index+1 || !validLocalKey(item.Key) || !validText(item.Mechanism, 10_000) ||
			!validNullableText(item.EvidenceGap, 10_000) || item.EvidenceCount < 0 {
			return nil, biz.ErrDataUnavailable
		}
		if _, duplicate := seen[item.Key]; duplicate {
			return nil, biz.ErrDataUnavailable
		}
		seen[item.Key] = struct{}{}
		confidence, err := mapConfidence(item.Confidence)
		if err != nil {
			return nil, err
		}
		items[index] = biz.CandidateMechanism{Key: item.Key, DisplayOrder: item.DisplayOrder,
			Mechanism: item.Mechanism, EvidenceGap: item.EvidenceGap, Confidence: confidence,
			Scope:       biz.EvidenceScope{Type: biz.ScopeCandidateMechanism, Key: item.Key},
			HasEvidence: item.EvidenceCount > 0}
	}
	return items, nil
}

func mapChainSummary(wire wireIndustryChainSummary) (biz.IndustryChainSummary, error) {
	if !validLocalKey(wire.Key) || wire.DisplayOrder < 1 || !validText(wire.Name, 10_000) ||
		!validText(wire.Conclusion, 10_000) || !validText(wire.Status, 10_000) ||
		!validText(wire.TimeWindow, 10_000) || wire.EvidenceCount < 0 {
		return biz.IndustryChainSummary{}, biz.ErrDataUnavailable
	}
	result, err := mapResult(wire.Result)
	if err != nil {
		return biz.IndustryChainSummary{}, err
	}
	confidence, err := mapConfidence(wire.Confidence)
	if err != nil {
		return biz.IndustryChainSummary{}, err
	}
	return biz.IndustryChainSummary{Key: wire.Key, DisplayOrder: wire.DisplayOrder, Name: wire.Name,
		Conclusion: wire.Conclusion, Status: wire.Status, Result: result, Confidence: confidence,
		TimeWindow: wire.TimeWindow, Scope: biz.EvidenceScope{Type: biz.ScopeIndustryChain, Key: wire.Key},
		HasEvidence: wire.EvidenceCount > 0}, nil
}

func mapIndustryChainDetail(wire wireIndustryChainDetail, expectedReportID, expectedChainKey string) (biz.IndustryChainDetail, error) {
	summary, err := mapSummary(wire.Report)
	if err != nil || summary.ID != expectedReportID || wire.IndustryChain.Key != expectedChainKey {
		return biz.IndustryChainDetail{}, biz.ErrDataUnavailable
	}
	chain, err := mapIndustryChain(wire.IndustryChain)
	if err != nil {
		return biz.IndustryChainDetail{}, err
	}
	return biz.IndustryChainDetail{Report: summary, IndustryChain: chain}, nil
}

func mapIndustryChain(wire wireIndustryChain) (biz.IndustryChain, error) {
	if !validLocalKey(wire.Key) || !validLocalKey(wire.ClaimKey) || wire.DisplayOrder < 1 ||
		!validText(wire.Name, 10_000) || !validText(wire.Conclusion, 10_000) || !validText(wire.Status, 10_000) ||
		!validText(wire.TimeWindow, 10_000) || !validNullableText(wire.PathSummary, 10_000) ||
		!validNullableText(wire.AcceptedHypothesisSummary, 10_000) || wire.EvidenceCount < 0 ||
		wire.Nodes == nil || wire.Edges == nil || wire.Uncertainty.Checkpoints == nil ||
		!validNullableText(wire.Uncertainty.CounterevidenceAndGap, 10_000) ||
		!validNullableText(wire.Uncertainty.StopCondition, 10_000) {
		return biz.IndustryChain{}, biz.ErrDataUnavailable
	}
	result, err := mapResult(wire.Result)
	if err != nil {
		return biz.IndustryChain{}, err
	}
	confidence, err := mapConfidence(wire.Confidence)
	if err != nil {
		return biz.IndustryChain{}, err
	}
	nodes := make([]biz.IndustryChainNode, len(wire.Nodes))
	nodeKeys := make(map[string]struct{}, len(nodes))
	for index, item := range wire.Nodes {
		if item.DisplayOrder != index+1 || !validLocalKey(item.Key) || !validText(item.Name, 10_000) ||
			!validText(item.Impact, 10_000) || !validText(item.Reasoning, 10_000) ||
			!validText(item.TimeWindow, 10_000) || item.EvidenceCount < 0 {
			return biz.IndustryChain{}, biz.ErrDataUnavailable
		}
		if _, duplicate := nodeKeys[item.Key]; duplicate {
			return biz.IndustryChain{}, biz.ErrDataUnavailable
		}
		nodeKeys[item.Key] = struct{}{}
		itemResult, resultErr := mapResult(item.Result)
		itemNature, natureErr := mapNature(item.Nature)
		itemConfidence, confidenceErr := mapConfidence(item.Confidence)
		if resultErr != nil || natureErr != nil || confidenceErr != nil {
			return biz.IndustryChain{}, biz.ErrDataUnavailable
		}
		nodes[index] = biz.IndustryChainNode{Key: item.Key, DisplayOrder: item.DisplayOrder,
			Name: item.Name, Impact: item.Impact, Result: itemResult, Nature: itemNature,
			Reasoning: item.Reasoning, TimeWindow: item.TimeWindow, Confidence: itemConfidence,
			Scope:       biz.EvidenceScope{Type: biz.ScopeIndustryChainNode, Key: item.Key},
			HasEvidence: item.EvidenceCount > 0}
	}
	edges := make([]biz.IndustryChainEdge, len(wire.Edges))
	edgeKeys := make(map[string]struct{}, len(edges))
	for index, item := range wire.Edges {
		if item.DisplayOrder != index+1 || !validLocalKey(item.Key) || !validLocalKey(item.FromNodeKey) ||
			!validLocalKey(item.ToNodeKey) || item.FromNodeKey == item.ToNodeKey || !validText(item.RelationLabel, 500) {
			return biz.IndustryChain{}, biz.ErrDataUnavailable
		}
		if _, fromExists := nodeKeys[item.FromNodeKey]; !fromExists {
			return biz.IndustryChain{}, biz.ErrDataUnavailable
		}
		if _, toExists := nodeKeys[item.ToNodeKey]; !toExists {
			return biz.IndustryChain{}, biz.ErrDataUnavailable
		}
		if _, duplicate := edgeKeys[item.Key]; duplicate {
			return biz.IndustryChain{}, biz.ErrDataUnavailable
		}
		edgeKeys[item.Key] = struct{}{}
		edges[index] = biz.IndustryChainEdge{Key: item.Key, DisplayOrder: item.DisplayOrder,
			FromNodeKey: item.FromNodeKey, ToNodeKey: item.ToNodeKey, RelationLabel: item.RelationLabel}
	}
	checkpoints, err := mapCheckpoints(wire.Uncertainty.Checkpoints)
	if err != nil {
		return biz.IndustryChain{}, err
	}
	return biz.IndustryChain{Key: wire.Key, ClaimKey: wire.ClaimKey, DisplayOrder: wire.DisplayOrder,
		Name: wire.Name, Conclusion: wire.Conclusion, Status: wire.Status, Result: result,
		Confidence: confidence, TimeWindow: wire.TimeWindow, PathSummary: wire.PathSummary,
		AcceptedHypothesisSummary: wire.AcceptedHypothesisSummary, Nodes: nodes, Edges: edges,
		Uncertainty: biz.ChainUncertainty{CounterevidenceAndGap: wire.Uncertainty.CounterevidenceAndGap,
			StopCondition: wire.Uncertainty.StopCondition, Checkpoints: checkpoints},
		Scope: biz.EvidenceScope{Type: biz.ScopeIndustryChain, Key: wire.Key}, HasEvidence: wire.EvidenceCount > 0}, nil
}

func mapCheckpoints(wire []wireCheckpoint) ([]biz.Checkpoint, error) {
	items := make([]biz.Checkpoint, len(wire))
	seen := make(map[string]struct{}, len(wire))
	for index, item := range wire {
		if item.DisplayOrder != index+1 || !validLocalKey(item.Key) || !validText(item.Summary, 10_000) {
			return nil, biz.ErrDataUnavailable
		}
		if _, duplicate := seen[item.Key]; duplicate {
			return nil, biz.ErrDataUnavailable
		}
		seen[item.Key] = struct{}{}
		items[index] = biz.Checkpoint{Key: item.Key, DisplayOrder: item.DisplayOrder, Summary: item.Summary}
	}
	return items, nil
}

func mapEvidenceCollection(wire wireEvidenceCollection, expectedReportID string, expectedScope biz.EvidenceScope) (biz.EvidenceCollection, error) {
	if wire.ReportID != expectedReportID || wire.ScopeType != expectedScope.Type || wire.ScopeKey != expectedScope.Key || wire.Items == nil {
		return biz.EvidenceCollection{}, biz.ErrDataUnavailable
	}
	items := make([]biz.EvidenceItem, len(wire.Items))
	seenIDs := make(map[string]struct{}, len(items))
	for index, item := range wire.Items {
		if !evidenceIDPattern.MatchString(item.EvidenceID) || !validText(item.Role, 200) ||
			item.DisplayOrder != index+1 || !validText(item.Summary, 200) || len(item.Keywords) < 1 ||
			!validStringArray(item.Keywords, 5, 6) {
			return biz.EvidenceCollection{}, biz.ErrDataUnavailable
		}
		if _, duplicate := seenIDs[item.EvidenceID]; duplicate {
			return biz.EvidenceCollection{}, biz.ErrDataUnavailable
		}
		seenIDs[item.EvidenceID] = struct{}{}
		var publishedAt *time.Time
		if item.PublishedAt != nil {
			parsed, err := parseTimestamp(*item.PublishedAt)
			if err != nil {
				return biz.EvidenceCollection{}, biz.ErrDataUnavailable
			}
			publishedAt = &parsed
		}
		items[index] = biz.EvidenceItem{PublishedAt: publishedAt, Summary: item.Summary, Keywords: item.Keywords}
	}
	return biz.EvidenceCollection{ReportID: wire.ReportID, Scope: expectedScope, Items: items}, nil
}

func mapReference(wire wireReference) (biz.Reference, error) {
	ref := biz.Reference{Type: wire.Type, Key: wire.Key}
	if !biz.ValidReference(ref) {
		return biz.Reference{}, biz.ErrDataUnavailable
	}
	return ref, nil
}

func mapResult(wire wireResult) (biz.Result, error) {
	wantLabel := ""
	switch wire.Code {
	case "warming":
		wantLabel = "升温"
	case "cooling":
		wantLabel = "降温"
	case "diverging":
		wantLabel = "分化"
	case "pending":
		wantLabel = "待验证"
	default:
		return biz.Result{}, biz.ErrDataUnavailable
	}
	if wire.Label != wantLabel {
		return biz.Result{}, biz.ErrDataUnavailable
	}
	return biz.Result{Code: wire.Code, Label: wire.Label}, nil
}

func mapNature(wire wireNature) (biz.Nature, error) {
	wantLabel := ""
	switch wire.Code {
	case "direct_evidence":
		wantLabel = "直接证据"
	case "reasoning_hypothesis":
		wantLabel = "推理假设"
	case "pending_validation":
		wantLabel = "待验证"
	default:
		return biz.Nature{}, biz.ErrDataUnavailable
	}
	if wire.Label != wantLabel {
		return biz.Nature{}, biz.ErrDataUnavailable
	}
	return biz.Nature{Code: wire.Code, Label: wire.Label}, nil
}

func mapConfidence(wire wireConfidence) (biz.Confidence, error) {
	if !validText(wire.Label, 100) || (wire.Score != nil && (math.IsNaN(*wire.Score) || math.IsInf(*wire.Score, 0) || *wire.Score < 0 || *wire.Score > 1)) {
		return biz.Confidence{}, biz.ErrDataUnavailable
	}
	return biz.Confidence{Label: wire.Label, Score: wire.Score}, nil
}

func validStatistics(value wireStatistics) bool {
	counts := []int{value.EventCount, value.OrdinaryFactCount, value.SignalFactCount,
		value.TransmissionHypothesisCount, value.RemainingTopologyPendingCount, value.AdaptiveHardMaxHops,
		value.AdaptiveObservedMaxHops, value.AdaptiveStoppedByConfidence,
		value.AdaptiveStoppedByNoUnvisitedNeighbor, value.AdaptiveRejectedBelowInclusion,
		value.GeopoliticAnchorCount, value.MacroeconomicAnchorCount, value.SignaledChainNodeCount,
		value.IndustryChainCount, value.UnmappedChainNodeCount}
	for _, count := range counts {
		if count < 0 {
			return false
		}
	}
	for _, threshold := range []float64{value.AdaptiveInclusionThreshold, value.AdaptiveContinuationThreshold} {
		if math.IsNaN(threshold) || math.IsInf(threshold, 0) || threshold < 0 || threshold > 1 {
			return false
		}
	}
	return true
}

func validCardKind(value string) bool {
	return value == biz.LayerGeopolitics || value == biz.LayerMacroeconomics || value == biz.ScopeIndustryChain
}

func validPublishedLayers(values []string) bool {
	if len(values) == 0 || len(values) > 4 {
		return false
	}
	allowed := map[string]struct{}{
		"geopolitics": {}, "macroeconomics": {}, "industry_chain": {}, "company": {},
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, exists := allowed[value]; !exists {
			return false
		}
		if _, duplicate := seen[value]; duplicate {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func validLayer(value string) bool {
	return value == biz.LayerGeopolitics || value == biz.LayerMacroeconomics
}

func validLocalKey(value string) bool {
	return localKeyPattern.MatchString(value)
}

func validLocalKeyArray(values []string) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !validLocalKey(value) {
			return false
		}
		if _, duplicate := seen[value]; duplicate {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func validStringArray(values []string, maxItems, maxText int) bool {
	if values == nil || (maxItems > 0 && len(values) > maxItems) {
		return false
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !validText(value, maxText) {
			return false
		}
		if _, duplicate := seen[value]; duplicate {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func validNullableText(value *string, maximum int) bool {
	return value == nil || validText(*value, maximum)
}

func validText(value string, maximum int) bool {
	return value == strings.TrimSpace(value) && value != "" && utf8.ValidString(value) && utf8.RuneCountInString(value) <= maximum
}

func validMetadata(value string, maximum int) bool {
	return len(value) > 0 && len(value) <= maximum && metadataPattern.MatchString(value)
}

func parseTimestamp(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

var _ biz.Repository = (*Repository)(nil)
