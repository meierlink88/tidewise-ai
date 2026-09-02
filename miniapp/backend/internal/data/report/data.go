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
	localKeyPattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
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
	var wire wireV2Page
	if err := r.get(ctx, path, &wire); err != nil {
		return biz.Page{}, mapReadError(err, readList)
	}
	page, err := mapV2Page(wire)
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
	var wire wireV2Home
	if err := r.get(ctx, reportsPath+"/"+url.PathEscape(reportID)+"/home", &wire); err != nil {
		return biz.Home{}, mapReadError(err, readHome)
	}
	return r.mapV2Home(ctx, wire, reportID)
}

func (r *Repository) GetLayer(ctx context.Context, reportID, layerKey string) (biz.LayerDetail, error) {
	path := reportsPath + "/" + url.PathEscape(reportID) + "/layers/" + url.PathEscape(layerKey)
	var wire wireV2LayerDetail
	if err := r.get(ctx, path, &wire); err != nil {
		return biz.LayerDetail{}, mapReadError(err, readLayer)
	}
	return mapV2LayerDetail(wire, reportID, layerKey)
}

func (r *Repository) GetIndustryChain(ctx context.Context, reportID, chainKey string) (biz.IndustryChainDetail, error) {
	path := reportsPath + "/" + url.PathEscape(reportID) + "/industry-chains/" + url.PathEscape(chainKey)
	var wire wireV2IndustryChainDetail
	if err := r.get(ctx, path, &wire); err != nil {
		return biz.IndustryChainDetail{}, mapReadError(err, readChain)
	}
	return mapV2IndustryChainDetail(wire, reportID, chainKey)
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
			jsonTag := strings.Split(field.Tag.Get("json"), ",")
			name := jsonTag[0]
			if name == "-" {
				continue
			}
			if name == "" {
				name = field.Name
			}
			fieldPayload, exists := object[name]
			if !exists {
				if len(jsonTag) > 1 && jsonTag[1] == "omitempty" {
					continue
				}
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

type wireCheckpoint struct {
	Key          string `json:"key"`
	DisplayOrder int    `json:"display_order"`
	Summary      string `json:"summary"`
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

type wireV2Statistics struct {
	EventCount                  int `json:"event_count"`
	OrdinaryFactCount           int `json:"ordinary_fact_count"`
	SignalFactCount             int `json:"signal_fact_count"`
	TransmissionHypothesisCount int `json:"transmission_hypothesis_count"`
	GeopoliticAnchorCount       int `json:"geopolitic_anchor_count"`
	MacroeconomicAnchorCount    int `json:"macroeconomic_anchor_count"`
	SignaledChainNodeCount      int `json:"signaled_chain_node_count"`
	IndustryChainCount          int `json:"industry_chain_count"`
}

type wireV2Summary struct {
	ID                string           `json:"id"`
	PublisherReportID string           `json:"publisher_report_id"`
	ReportType        string           `json:"report_type"`
	Title             string           `json:"title"`
	GenerationStatus  string           `json:"generation_status"`
	Simulation        bool             `json:"simulation"`
	GeneratedAt       string           `json:"generated_at"`
	Timezone          string           `json:"timezone"`
	HasGeopolitics    bool             `json:"has_geopolitics"`
	HasMacroeconomics bool             `json:"has_macroeconomics"`
	Statistics        wireV2Statistics `json:"statistics"`
	PublishedAt       string           `json:"published_at"`
}

type wireV2Page struct {
	Items      []wireV2Summary `json:"items"`
	NextCursor *string         `json:"next_cursor"`
}

type wireV2Result struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

type wireV2Nature struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

type wireV2Confidence struct {
	Code  string   `json:"code"`
	Label string   `json:"label"`
	Score *float64 `json:"score"`
}

type wireV2TimeWindow struct {
	Horizons []string `json:"horizons"`
	Lag      *string  `json:"lag"`
	Label    string   `json:"label"`
}

type wireV2Effect struct {
	DisplayOrder int    `json:"display_order"`
	Dimension    string `json:"dimension"`
	Direction    string `json:"direction"`
	Confidence   string `json:"confidence"`
}

type wireV2EvidenceReference struct {
	EvidenceID   string `json:"evidence_id"`
	Role         string `json:"role"`
	DisplayOrder int    `json:"display_order"`
}

type wireV2Reference struct {
	Type string `json:"type"`
	Key  string `json:"key"`
}

type wireV2Claim struct {
	Key  string `json:"key"`
	Text string `json:"text"`
}

type wireV2Anchor struct {
	Key          string                    `json:"key"`
	DisplayOrder int                       `json:"display_order"`
	Name         string                    `json:"name"`
	Effects      []wireV2Effect            `json:"effects"`
	Result       wireV2Result              `json:"result"`
	Nature       wireV2Nature              `json:"nature"`
	Reasoning    string                    `json:"reasoning"`
	TimeWindow   wireV2TimeWindow          `json:"time_window"`
	Confidence   wireV2Confidence          `json:"confidence"`
	SourceRef    *string                   `json:"source_ref"`
	EvidenceRefs []wireV2EvidenceReference `json:"evidence_refs"`
}

type wireV2ReasoningStep struct {
	Key          string                    `json:"key"`
	DisplayOrder int                       `json:"display_order"`
	Input        string                    `json:"input"`
	Mechanism    string                    `json:"mechanism"`
	Output       string                    `json:"output"`
	Type         string                    `json:"type"`
	Confidence   wireV2Confidence          `json:"confidence"`
	EvidenceRefs []wireV2EvidenceReference `json:"evidence_refs"`
}

type wireV2NamedResult struct {
	Name   string       `json:"name"`
	Result wireV2Result `json:"result"`
}

type wireV2TransmissionTarget struct {
	Ref     *wireV2Reference    `json:"ref,omitempty"`
	Label   string              `json:"label"`
	Results []wireV2NamedResult `json:"results"`
}

type wireV2Transmission struct {
	Key              string                     `json:"key"`
	DisplayOrder     int                        `json:"display_order"`
	SourceClaimKey   string                     `json:"source_claim_key"`
	SourceConclusion string                     `json:"source_conclusion"`
	Targets          []wireV2TransmissionTarget `json:"targets"`
	Logic            string                     `json:"logic"`
	RelationNature   string                     `json:"relation_nature"`
	Confidence       wireV2Confidence           `json:"confidence"`
	Status           string                     `json:"status"`
	EvidenceRefs     []wireV2EvidenceReference  `json:"evidence_refs"`
}

type wireV2LayerUncertainty struct {
	Counterevidence   *string          `json:"counterevidence"`
	EvidenceGap       *string          `json:"evidence_gap"`
	Boundary          *string          `json:"boundary"`
	ReversalCondition *string          `json:"reversal_condition"`
	Checkpoints       []wireCheckpoint `json:"checkpoints"`
}

type wireV2LayerSummary struct {
	Claim         wireV2Claim               `json:"claim"`
	Transmissions []wireV2Transmission      `json:"transmissions"`
	Uncertainty   wireV2LayerUncertainty    `json:"uncertainty"`
	EvidenceRefs  []wireV2EvidenceReference `json:"evidence_refs"`
}

type wireV2LayerAnalysis struct {
	Anchors          []wireV2Anchor        `json:"anchors"`
	ReasoningSteps   []wireV2ReasoningStep `json:"reasoning_steps"`
	RelatedChainKeys []string              `json:"related_chain_keys"`
}

type wireV2Layer struct {
	Key     string              `json:"key"`
	Title   string              `json:"title"`
	Summary wireV2LayerSummary  `json:"summary"`
	Detail  wireV2LayerAnalysis `json:"detail"`
}

type wireV2LayerSnapshot struct {
	Key     string             `json:"key"`
	Title   string             `json:"title"`
	Summary wireV2LayerSummary `json:"summary"`
}

type wireV2Home struct {
	Report         wireV2Summary        `json:"report"`
	Geopolitics    *wireV2LayerSnapshot `json:"geopolitics"`
	Macroeconomics *wireV2LayerSnapshot `json:"macroeconomics"`
}

type wireV2ChainImpactSummary struct {
	Key           string           `json:"key"`
	DisplayOrder  int              `json:"display_order"`
	NodeKey       string           `json:"node_key"`
	Name          string           `json:"name"`
	Result        wireV2Result     `json:"result"`
	Nature        wireV2Nature     `json:"nature"`
	Confidence    wireV2Confidence `json:"confidence"`
	TimeWindow    wireV2TimeWindow `json:"time_window"`
	EvidenceCount int              `json:"evidence_count"`
}

type wireV2ChainSummary struct {
	Key           string                     `json:"key"`
	DisplayOrder  int                        `json:"display_order"`
	Name          string                     `json:"name"`
	Claim         wireV2Claim                `json:"claim"`
	Status        string                     `json:"status"`
	Result        wireV2Result               `json:"result"`
	Confidence    wireV2Confidence           `json:"confidence"`
	TimeWindow    wireV2TimeWindow           `json:"time_window"`
	ImpactItems   []wireV2ChainImpactSummary `json:"impact_items"`
	EvidenceCount int                        `json:"evidence_count"`
}

type wireV2ChainPage struct {
	Items      []wireV2ChainSummary `json:"items"`
	NextCursor *string              `json:"next_cursor"`
}

type wireV2LayerDetail struct {
	Report                wireV2Summary        `json:"report"`
	Layer                 wireV2Layer          `json:"layer"`
	RelatedIndustryChains []wireV2ChainSummary `json:"related_industry_chains"`
}

type wireV2TopologyNode struct {
	Key          string `json:"key"`
	DisplayOrder int    `json:"display_order"`
	Name         string `json:"name"`
}

type wireV2IndustryChainEdge struct {
	Key           string `json:"key"`
	DisplayOrder  int    `json:"display_order"`
	FromNodeKey   string `json:"from_node_key"`
	ToNodeKey     string `json:"to_node_key"`
	RelationLabel string `json:"relation_label"`
}

type wireV2ChainNode struct {
	Key          string                    `json:"key"`
	DisplayOrder int                       `json:"display_order"`
	NodeKey      string                    `json:"node_key"`
	Effects      []wireV2Effect            `json:"effects"`
	Result       wireV2Result              `json:"result"`
	Nature       wireV2Nature              `json:"nature"`
	Reasoning    string                    `json:"reasoning"`
	TimeWindow   wireV2TimeWindow          `json:"time_window"`
	Confidence   wireV2Confidence          `json:"confidence"`
	EvidenceRefs []wireV2EvidenceReference `json:"evidence_refs"`
}

type wireV2ChainSummaryDetail struct {
	Claim                     wireV2Claim               `json:"claim"`
	Status                    string                    `json:"status"`
	Result                    wireV2Result              `json:"result"`
	Confidence                wireV2Confidence          `json:"confidence"`
	TimeWindow                wireV2TimeWindow          `json:"time_window"`
	Path                      string                    `json:"path"`
	AcceptedHypothesisSummary *string                   `json:"accepted_hypothesis_summary"`
	Graph                     wireV2IndustryChainGraph  `json:"graph"`
	Uncertainty               wireV2ChainUncertainty    `json:"uncertainty"`
	EvidenceRefs              []wireV2EvidenceReference `json:"evidence_refs"`
}

type wireV2ChainUncertainty struct {
	CounterevidenceAndGap string `json:"counterevidence_and_gap"`
	StopCondition         string `json:"stop_condition"`
}

type wireV2IndustryChainGraph struct {
	Nodes []wireV2TopologyNode      `json:"nodes"`
	Edges []wireV2IndustryChainEdge `json:"edges"`
}

type wireV2ChainAnalysis struct {
	NodeImpacts []wireV2ChainNode `json:"node_impacts"`
}

type wireV2IndustryChain struct {
	Key          string                   `json:"key"`
	DisplayOrder int                      `json:"display_order"`
	Name         string                   `json:"name"`
	Summary      wireV2ChainSummaryDetail `json:"summary"`
	Detail       wireV2ChainAnalysis      `json:"detail"`
}

type wireV2IndustryChainDetail struct {
	Report        wireV2Summary       `json:"report"`
	IndustryChain wireV2IndustryChain `json:"industry_chain"`
}

func mapV2Page(wire wireV2Page) (biz.Page, error) {
	if wire.Items == nil {
		return biz.Page{}, biz.ErrDataUnavailable
	}
	items := make([]biz.Summary, len(wire.Items))
	for index, item := range wire.Items {
		mapped, err := mapV2Summary(item)
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

func mapV2Summary(wire wireV2Summary) (biz.Summary, error) {
	generatedAt, err := parseTimestamp(wire.GeneratedAt)
	if err != nil {
		return biz.Summary{}, biz.ErrDataUnavailable
	}
	publishedAt, err := parseTimestamp(wire.PublishedAt)
	if err != nil {
		return biz.Summary{}, biz.ErrDataUnavailable
	}
	counts := []int{wire.Statistics.EventCount, wire.Statistics.OrdinaryFactCount,
		wire.Statistics.SignalFactCount, wire.Statistics.TransmissionHypothesisCount,
		wire.Statistics.GeopoliticAnchorCount, wire.Statistics.MacroeconomicAnchorCount,
		wire.Statistics.SignaledChainNodeCount, wire.Statistics.IndustryChainCount}
	for _, count := range counts {
		if count < 0 {
			return biz.Summary{}, biz.ErrDataUnavailable
		}
	}
	if wire.Statistics.IndustryChainCount < 1 || !reportIDPattern.MatchString(wire.ID) || !validText(wire.PublisherReportID, 200) ||
		!validText(wire.ReportType, 100) || !validText(wire.Title, 500) ||
		!validText(wire.GenerationStatus, 100) || !validText(wire.Timezone, 100) {
		return biz.Summary{}, biz.ErrDataUnavailable
	}
	return biz.Summary{ID: wire.ID, PublisherReportID: wire.PublisherReportID, Title: wire.Title,
		GeneratedAt: generatedAt, PublishedAt: publishedAt}, nil
}

func (r *Repository) mapV2Home(ctx context.Context, wire wireV2Home, expectedReportID string) (biz.Home, error) {
	summary, err := mapV2Summary(wire.Report)
	if err != nil || summary.ID != expectedReportID || wire.Report.HasGeopolitics != (wire.Geopolitics != nil) ||
		wire.Report.HasMacroeconomics != (wire.Macroeconomics != nil) {
		return biz.Home{}, biz.ErrDataUnavailable
	}
	cards := make([]biz.Card, 0, wire.Report.Statistics.IndustryChainCount+2)
	for _, snapshot := range []*wireV2LayerSnapshot{wire.Geopolitics, wire.Macroeconomics} {
		if snapshot == nil {
			continue
		}
		detail, readErr := r.GetLayer(ctx, expectedReportID, snapshot.Key)
		if readErr != nil || detail.Layer.Title != snapshot.Title || detail.Layer.Conclusion != snapshot.Summary.Claim.Text {
			return biz.Home{}, biz.ErrDataUnavailable
		}
		cards = append(cards, cardFromV2Layer(detail.Layer, len(cards)+1))
	}
	chains, err := r.listAllV2ChainSummaries(ctx, expectedReportID)
	if err != nil || len(chains) != wire.Report.Statistics.IndustryChainCount {
		return biz.Home{}, biz.ErrDataUnavailable
	}
	for _, chain := range chains {
		cards = append(cards, cardFromV2ChainSummary(chain, len(cards)+1))
	}
	return biz.Home{Report: summary, IndustryChainCount: len(chains), Cards: cards}, nil
}

func (r *Repository) listAllV2ChainSummaries(ctx context.Context, reportID string) ([]wireV2ChainSummary, error) {
	items := make([]wireV2ChainSummary, 0)
	cursor := ""
	seenCursors := map[string]struct{}{}
	seenKeys := map[string]struct{}{}
	for pageIndex := 0; pageIndex < 100; pageIndex++ {
		values := url.Values{"limit": []string{"100"}}
		if cursor != "" {
			values.Set("cursor", cursor)
		}
		var page wireV2ChainPage
		if err := r.get(ctx, reportsPath+"/"+url.PathEscape(reportID)+"/industry-chains?"+values.Encode(), &page); err != nil {
			return nil, mapReadError(err, readChain)
		}
		if page.Items == nil {
			return nil, biz.ErrDataUnavailable
		}
		for _, item := range page.Items {
			if err := validateV2ChainSummary(item); err != nil {
				return nil, err
			}
			if item.DisplayOrder != len(items)+1 {
				return nil, biz.ErrDataUnavailable
			}
			if _, duplicate := seenKeys[item.Key]; duplicate {
				return nil, biz.ErrDataUnavailable
			}
			seenKeys[item.Key] = struct{}{}
			items = append(items, item)
		}
		if page.NextCursor == nil {
			return items, nil
		}
		next := strings.TrimSpace(*page.NextCursor)
		if next == "" || len(page.Items) == 0 || len(next) > 2048 {
			return nil, biz.ErrDataUnavailable
		}
		if _, duplicate := seenCursors[next]; duplicate {
			return nil, biz.ErrDataUnavailable
		}
		seenCursors[next] = struct{}{}
		cursor = next
	}
	return nil, biz.ErrDataUnavailable
}

func validateV2ChainSummary(wire wireV2ChainSummary) error {
	if !validLocalKey(wire.Key) || wire.DisplayOrder < 1 || !validText(wire.Name, 500) ||
		!validLocalKey(wire.Claim.Key) || !validText(wire.Claim.Text, 10_000) ||
		!validText(wire.Status, 10_000) || wire.ImpactItems == nil || len(wire.ImpactItems) == 0 || wire.EvidenceCount < 0 {
		return biz.ErrDataUnavailable
	}
	if _, err := mapV2Result(wire.Result); err != nil {
		return err
	}
	if _, err := mapV2Confidence(wire.Confidence); err != nil {
		return err
	}
	if !validV2TimeWindow(wire.TimeWindow) {
		return biz.ErrDataUnavailable
	}
	for index, impact := range wire.ImpactItems {
		if impact.DisplayOrder != index+1 || !validLocalKey(impact.Key) || !validLocalKey(impact.NodeKey) ||
			!validText(impact.Name, 500) || impact.EvidenceCount < 0 || !validV2TimeWindow(impact.TimeWindow) {
			return biz.ErrDataUnavailable
		}
		nature, err := mapV2Nature(impact.Nature)
		if err != nil || (nature.Code == "direct_evidence") != (impact.EvidenceCount > 0) {
			return biz.ErrDataUnavailable
		}
		if _, err := mapV2Result(impact.Result); err != nil {
			return err
		}
		if _, err := mapV2Confidence(impact.Confidence); err != nil {
			return err
		}
	}
	return nil
}

func cardFromV2Layer(layer biz.Layer, displayOrder int) biz.Card {
	impacts := make([]biz.CardImpactItem, len(layer.Anchors))
	for index, anchor := range layer.Anchors {
		impacts[index] = biz.CardImpactItem{Ref: biz.Reference{Type: biz.ScopeAnchor, Key: anchor.Key},
			Name: anchor.Name, Result: anchor.Result, Confidence: anchor.Confidence,
			TimeWindow: anchor.TimeWindow, HasEvidence: anchor.HasEvidence}
	}
	subtitle := "增长预期与政策利率"
	if layer.Key == biz.LayerGeopolitics {
		subtitle = "安全对抗与通道可用性"
	}
	return biz.Card{Key: layer.Key, Kind: layer.Key, Order: displayOrder,
		DetailRef: biz.Reference{Type: biz.ScopeLayer, Key: layer.Key}, Title: layer.Title,
		Subtitle: subtitle, Conclusion: layer.Conclusion, Result: layer.Result,
		Confidence: layer.Confidence, TimeWindow: layer.TimeWindow, ImpactItems: impacts,
		HasEvidence: layer.HasEvidence}
}

func cardFromV2ChainSummary(wire wireV2ChainSummary, displayOrder int) biz.Card {
	result, _ := mapV2Result(wire.Result)
	confidence, _ := mapV2Confidence(wire.Confidence)
	impacts := make([]biz.CardImpactItem, len(wire.ImpactItems))
	for index, impact := range wire.ImpactItems {
		impactResult, _ := mapV2Result(impact.Result)
		impactConfidence, _ := mapV2Confidence(impact.Confidence)
		impacts[index] = biz.CardImpactItem{Ref: biz.Reference{Type: biz.ScopeIndustryChainNode, Key: impact.Key},
			Name: impact.Name, Result: impactResult, Confidence: impactConfidence,
			TimeWindow: impact.TimeWindow.Label, HasEvidence: impact.EvidenceCount > 0}
	}
	return biz.Card{Key: wire.Key, Kind: biz.ScopeIndustryChain, Order: displayOrder,
		DetailRef: biz.Reference{Type: biz.ScopeIndustryChain, Key: wire.Key}, Title: wire.Name,
		Subtitle: "产业链", Conclusion: wire.Claim.Text, Result: result, Confidence: confidence,
		TimeWindow: wire.TimeWindow.Label, ImpactItems: impacts, HasEvidence: wire.EvidenceCount > 0}
}

func mapV2LayerDetail(wire wireV2LayerDetail, expectedReportID, expectedLayerKey string) (biz.LayerDetail, error) {
	summary, err := mapV2Summary(wire.Report)
	if err != nil || summary.ID != expectedReportID || wire.Layer.Key != expectedLayerKey || wire.RelatedIndustryChains == nil {
		return biz.LayerDetail{}, biz.ErrDataUnavailable
	}
	layer, err := mapV2Layer(wire.Layer)
	if err != nil {
		return biz.LayerDetail{}, err
	}
	if len(wire.RelatedIndustryChains) != len(wire.Layer.Detail.RelatedChainKeys) {
		return biz.LayerDetail{}, biz.ErrDataUnavailable
	}
	chains := make([]biz.IndustryChainSummary, len(wire.RelatedIndustryChains))
	for index, item := range wire.RelatedIndustryChains {
		if err := validateV2ChainSummary(item); err != nil || item.Key != wire.Layer.Detail.RelatedChainKeys[index] {
			return biz.LayerDetail{}, biz.ErrDataUnavailable
		}
		result, _ := mapV2Result(item.Result)
		confidence, _ := mapV2Confidence(item.Confidence)
		chains[index] = biz.IndustryChainSummary{Key: item.Key, DisplayOrder: item.DisplayOrder,
			Name: item.Name, Conclusion: item.Claim.Text, Status: item.Status, Result: result,
			Confidence: confidence, TimeWindow: item.TimeWindow.Label,
			Scope: biz.EvidenceScope{Type: biz.ScopeIndustryChainSummary, Key: item.Key}, HasEvidence: item.EvidenceCount > 0}
	}
	return biz.LayerDetail{Report: summary, Layer: layer, RelatedIndustryChains: chains}, nil
}

func mapV2Layer(wire wireV2Layer) (biz.Layer, error) {
	if !validLayer(wire.Key) || !validText(wire.Title, 500) || !validLocalKey(wire.Summary.Claim.Key) ||
		!validText(wire.Summary.Claim.Text, 10_000) || wire.Summary.Transmissions == nil ||
		wire.Summary.EvidenceRefs == nil || wire.Detail.Anchors == nil || wire.Detail.ReasoningSteps == nil ||
		wire.Detail.RelatedChainKeys == nil || wire.Summary.Uncertainty.Checkpoints == nil ||
		len(wire.Summary.Transmissions) == 0 || len(wire.Detail.Anchors) == 0 ||
		!validV2EvidenceRefs(wire.Summary.EvidenceRefs, "supports_claim") ||
		wire.Summary.Uncertainty.Counterevidence == nil || !validText(*wire.Summary.Uncertainty.Counterevidence, 10_000) ||
		wire.Summary.Uncertainty.Boundary == nil || !validText(*wire.Summary.Uncertainty.Boundary, 10_000) ||
		wire.Summary.Uncertainty.ReversalCondition == nil || !validText(*wire.Summary.Uncertainty.ReversalCondition, 10_000) ||
		!validNullableText(wire.Summary.Uncertainty.EvidenceGap, 10_000) {
		return biz.Layer{}, biz.ErrDataUnavailable
	}
	anchors := make([]biz.Anchor, len(wire.Detail.Anchors))
	for index, item := range wire.Detail.Anchors {
		if item.DisplayOrder != index+1 || !validLocalKey(item.Key) || !validText(item.Name, 500) ||
			!validText(item.Reasoning, 10_000) || !validV2Effects(item.Effects) ||
			!validV2TimeWindow(item.TimeWindow) {
			return biz.Layer{}, biz.ErrDataUnavailable
		}
		result, err := mapV2Result(item.Result)
		if err != nil {
			return biz.Layer{}, err
		}
		nature, err := mapV2Nature(item.Nature)
		if err != nil || !validV2EvidenceRefs(item.EvidenceRefs, "direct_target") ||
			(nature.Code == "direct_evidence") != (len(item.EvidenceRefs) > 0) {
			return biz.Layer{}, biz.ErrDataUnavailable
		}
		confidence, err := mapV2Confidence(item.Confidence)
		if err != nil {
			return biz.Layer{}, err
		}
		anchors[index] = biz.Anchor{Key: item.Key, DisplayOrder: item.DisplayOrder, Name: item.Name,
			CurrentState: effectsText(item.Effects), Result: result, Nature: nature, Reasoning: item.Reasoning,
			TimeWindow: item.TimeWindow.Label, Confidence: confidence,
			Scope: biz.EvidenceScope{Type: biz.ScopeAnchor, Key: item.Key}, HasEvidence: len(item.EvidenceRefs) > 0}
	}
	steps := make([]biz.ReasoningStep, len(wire.Detail.ReasoningSteps))
	for index, item := range wire.Detail.ReasoningSteps {
		confidence, err := mapV2Confidence(item.Confidence)
		if err != nil || item.DisplayOrder != index+1 || !validLocalKey(item.Key) ||
			!validText(item.Input, 10_000) || !validText(item.Mechanism, 10_000) ||
			!validText(item.Output, 10_000) || !validText(item.Type, 10_000) ||
			!validV2EvidenceRefs(item.EvidenceRefs, "supports_reasoning") {
			return biz.Layer{}, biz.ErrDataUnavailable
		}
		steps[index] = biz.ReasoningStep{Key: item.Key, DisplayOrder: item.DisplayOrder, Input: item.Input,
			Mechanism: item.Mechanism, Output: item.Output, Type: item.Type, Confidence: confidence,
			Scope: biz.EvidenceScope{Type: biz.ScopeReasoningStep, Key: item.Key}, HasEvidence: len(item.EvidenceRefs) > 0}
	}
	paths := make([]biz.TransmissionPath, len(wire.Summary.Transmissions))
	for index, item := range wire.Summary.Transmissions {
		confidence, err := mapV2Confidence(item.Confidence)
		if err != nil || item.DisplayOrder != index+1 || !validLocalKey(item.Key) || item.Targets == nil || len(item.Targets) == 0 ||
			!validText(item.SourceConclusion, 10_000) || !validText(item.Logic, 10_000) ||
			!validText(item.RelationNature, 10_000) || !validText(item.Status, 10_000) ||
			!validV2EvidenceRefs(item.EvidenceRefs, "supports_transmission") {
			return biz.Layer{}, biz.ErrDataUnavailable
		}
		targets := make([]biz.TransmissionTarget, 0)
		for _, target := range item.Targets {
			if !validText(target.Label, 500) || target.Results == nil || len(target.Results) == 0 {
				return biz.Layer{}, biz.ErrDataUnavailable
			}
			for _, named := range target.Results {
				if !validText(named.Name, 500) {
					return biz.Layer{}, biz.ErrDataUnavailable
				}
				result, resultErr := mapV2Result(named.Result)
				if resultErr != nil {
					return biz.Layer{}, resultErr
				}
				label := target.Label
				if len(target.Results) > 1 {
					label += " · " + named.Name
				}
				var ref *biz.Reference
				if target.Ref != nil {
					mapped, refErr := mapV2Reference(*target.Ref)
					if refErr != nil {
						return biz.Layer{}, refErr
					}
					ref = &mapped
				}
				targets = append(targets, biz.TransmissionTarget{Ref: ref, Label: label, Result: result})
			}
		}
		paths[index] = biz.TransmissionPath{Key: item.Key, DisplayOrder: item.DisplayOrder,
			SourceConclusion: item.SourceConclusion, TargetRefs: targets, Logic: item.Logic,
			RelationNature: item.RelationNature, EvidenceRole: "supports_transmission", Confidence: confidence,
			Status: item.Status, Scope: biz.EvidenceScope{Type: biz.ScopeTransmission, Key: item.Key}, HasEvidence: len(item.EvidenceRefs) > 0}
	}
	result := aggregateV2AnchorResult(anchors)
	confidence := aggregateV2LayerConfidence(anchors)
	timeWindow := aggregateV2TimeWindow(anchors)
	return biz.Layer{Key: wire.Key, DisplayOrder: layerDisplayOrder(wire.Key), Title: wire.Title,
		Conclusion: wire.Summary.Claim.Text, Result: result, Confidence: confidence, TimeWindow: timeWindow,
		Anchors: anchors, ReasoningSteps: steps, RelatedAnchorKeys: anchorKeys(anchors),
		RelatedChainKeys: wire.Detail.RelatedChainKeys,
		DownwardTransmission: biz.DownwardTransmission{Summary: wire.Summary.Claim.Text,
			PublishedPaths: paths, CandidateMechanisms: []biz.CandidateMechanism{}, BoundaryNotes: []string{}},
		Uncertainty: biz.LayerUncertainty{Counterevidence: wire.Summary.Uncertainty.Counterevidence,
			EvidenceGap: wire.Summary.Uncertainty.EvidenceGap, Boundary: wire.Summary.Uncertainty.Boundary,
			ReversalCondition: wire.Summary.Uncertainty.ReversalCondition,
			Checkpoints:       mapV2Checkpoints(wire.Summary.Uncertainty.Checkpoints)},
		Scope: biz.EvidenceScope{Type: biz.ScopeSectionSummary, Key: wire.Key}, HasEvidence: len(wire.Summary.EvidenceRefs) > 0}, nil
}

func mapV2IndustryChainDetail(wire wireV2IndustryChainDetail, expectedReportID, expectedChainKey string) (biz.IndustryChainDetail, error) {
	summary, err := mapV2Summary(wire.Report)
	if err != nil || summary.ID != expectedReportID || wire.IndustryChain.Key != expectedChainKey {
		return biz.IndustryChainDetail{}, biz.ErrDataUnavailable
	}
	chain, err := mapV2IndustryChain(wire.IndustryChain)
	if err != nil {
		return biz.IndustryChainDetail{}, err
	}
	return biz.IndustryChainDetail{Report: summary, IndustryChain: chain}, nil
}

func mapV2IndustryChain(wire wireV2IndustryChain) (biz.IndustryChain, error) {
	if !validLocalKey(wire.Key) || wire.DisplayOrder < 1 || !validText(wire.Name, 500) ||
		!validLocalKey(wire.Summary.Claim.Key) || !validText(wire.Summary.Claim.Text, 10_000) ||
		!validText(wire.Summary.Status, 10_000) || !validText(wire.Summary.Path, 10_000) ||
		!validNullableText(wire.Summary.AcceptedHypothesisSummary, 10_000) ||
		!validText(wire.Summary.Uncertainty.CounterevidenceAndGap, 10_000) ||
		!validText(wire.Summary.Uncertainty.StopCondition, 10_000) ||
		!validV2EvidenceRefs(wire.Summary.EvidenceRefs, "supports_claim") ||
		wire.Summary.Graph.Nodes == nil || len(wire.Summary.Graph.Nodes) == 0 || wire.Summary.Graph.Edges == nil ||
		wire.Detail.NodeImpacts == nil || len(wire.Detail.NodeImpacts) == 0 {
		return biz.IndustryChain{}, biz.ErrDataUnavailable
	}
	result, err := mapV2Result(wire.Summary.Result)
	if err != nil {
		return biz.IndustryChain{}, err
	}
	confidence, err := mapV2Confidence(wire.Summary.Confidence)
	if err != nil || !validV2TimeWindow(wire.Summary.TimeWindow) {
		return biz.IndustryChain{}, biz.ErrDataUnavailable
	}
	names := make(map[string]string, len(wire.Summary.Graph.Nodes))
	for index, node := range wire.Summary.Graph.Nodes {
		if node.DisplayOrder != index+1 || !validLocalKey(node.Key) || !validText(node.Name, 500) {
			return biz.IndustryChain{}, biz.ErrDataUnavailable
		}
		if _, duplicate := names[node.Key]; duplicate {
			return biz.IndustryChain{}, biz.ErrDataUnavailable
		}
		names[node.Key] = node.Name
	}
	nodes := make([]biz.IndustryChainNode, len(wire.Detail.NodeImpacts))
	impactNodeKeys := make(map[string]struct{}, len(nodes))
	impactKeys := make(map[string]struct{}, len(nodes))
	for index, item := range wire.Detail.NodeImpacts {
		name, ok := names[item.NodeKey]
		if !ok || item.DisplayOrder != index+1 || !validLocalKey(item.Key) || !validText(item.Reasoning, 10_000) ||
			!validV2Effects(item.Effects) || !validV2TimeWindow(item.TimeWindow) {
			return biz.IndustryChain{}, biz.ErrDataUnavailable
		}
		if _, duplicate := impactKeys[item.Key]; duplicate {
			return biz.IndustryChain{}, biz.ErrDataUnavailable
		}
		if _, duplicate := impactNodeKeys[item.NodeKey]; duplicate {
			return biz.IndustryChain{}, biz.ErrDataUnavailable
		}
		itemResult, resultErr := mapV2Result(item.Result)
		itemNature, natureErr := mapV2Nature(item.Nature)
		itemConfidence, confidenceErr := mapV2Confidence(item.Confidence)
		if resultErr != nil || natureErr != nil || confidenceErr != nil || !validV2EvidenceRefs(item.EvidenceRefs, "direct_target") ||
			(itemNature.Code == "direct_evidence") != (len(item.EvidenceRefs) > 0) {
			return biz.IndustryChain{}, biz.ErrDataUnavailable
		}
		impactKeys[item.Key] = struct{}{}
		impactNodeKeys[item.NodeKey] = struct{}{}
		nodes[index] = biz.IndustryChainNode{Key: item.Key, DisplayOrder: item.DisplayOrder, Name: name,
			Impact: effectsText(item.Effects), Result: itemResult, Nature: itemNature, Reasoning: item.Reasoning,
			TimeWindow: item.TimeWindow.Label, Confidence: itemConfidence,
			Scope: biz.EvidenceScope{Type: biz.ScopeIndustryChainNode, Key: item.Key}, HasEvidence: len(item.EvidenceRefs) > 0}
	}
	edges := make([]biz.IndustryChainEdge, 0, len(wire.Summary.Graph.Edges))
	edgeKeys := make(map[string]struct{}, len(wire.Summary.Graph.Edges))
	for index, item := range wire.Summary.Graph.Edges {
		_, fromExists := names[item.FromNodeKey]
		_, toExists := names[item.ToNodeKey]
		if item.DisplayOrder != index+1 || !validLocalKey(item.Key) || !validText(item.RelationLabel, 500) ||
			!fromExists || !toExists || item.FromNodeKey == item.ToNodeKey {
			return biz.IndustryChain{}, biz.ErrDataUnavailable
		}
		if _, duplicate := edgeKeys[item.Key]; duplicate {
			return biz.IndustryChain{}, biz.ErrDataUnavailable
		}
		edgeKeys[item.Key] = struct{}{}
		if _, ok := impactNodeKeys[item.FromNodeKey]; !ok {
			continue
		}
		if _, ok := impactNodeKeys[item.ToNodeKey]; !ok {
			continue
		}
		edges = append(edges, biz.IndustryChainEdge{Key: item.Key, DisplayOrder: len(edges) + 1,
			FromNodeKey: impactKeyForNode(wire.Detail.NodeImpacts, item.FromNodeKey),
			ToNodeKey:   impactKeyForNode(wire.Detail.NodeImpacts, item.ToNodeKey), RelationLabel: item.RelationLabel})
	}
	path := wire.Summary.Path
	counterevidence := wire.Summary.Uncertainty.CounterevidenceAndGap
	stopCondition := wire.Summary.Uncertainty.StopCondition
	return biz.IndustryChain{Key: wire.Key, ClaimKey: wire.Summary.Claim.Key, DisplayOrder: wire.DisplayOrder,
		Name: wire.Name, Conclusion: wire.Summary.Claim.Text, Status: wire.Summary.Status, Result: result,
		Confidence: confidence, TimeWindow: wire.Summary.TimeWindow.Label, PathSummary: &path,
		AcceptedHypothesisSummary: wire.Summary.AcceptedHypothesisSummary, Nodes: nodes, Edges: edges,
		Uncertainty: biz.ChainUncertainty{CounterevidenceAndGap: &counterevidence,
			StopCondition: &stopCondition, Checkpoints: []biz.Checkpoint{}},
		Scope: biz.EvidenceScope{Type: biz.ScopeIndustryChainSummary, Key: wire.Key}, HasEvidence: len(wire.Summary.EvidenceRefs) > 0}, nil
}

func mapV2Result(wire wireV2Result) (biz.Result, error) {
	return mapResult(wireResult{Code: wire.Code, Label: wire.Label})
}

func mapV2Nature(wire wireV2Nature) (biz.Nature, error) {
	return mapNature(wireNature{Code: wire.Code, Label: wire.Label})
}

func mapV2Confidence(wire wireV2Confidence) (biz.Confidence, error) {
	wantLabel := ""
	switch wire.Code {
	case "high":
		wantLabel = "高"
	case "medium_high":
		wantLabel = "中–高"
	case "medium":
		wantLabel = "中"
	case "low_medium":
		wantLabel = "低–中"
	case "low":
		wantLabel = "低"
	default:
		return biz.Confidence{}, biz.ErrDataUnavailable
	}
	if wire.Label != wantLabel {
		return biz.Confidence{}, biz.ErrDataUnavailable
	}
	return mapConfidence(wireConfidence{Label: wire.Label, Score: wire.Score})
}

func mapV2Reference(wire wireV2Reference) (biz.Reference, error) {
	typeName := wire.Type
	if typeName == "section" {
		typeName = biz.ScopeLayer
	}
	return mapReference(wireReference{Type: typeName, Key: wire.Key})
}

func validV2EvidenceRefs(values []wireV2EvidenceReference, role string) bool {
	if values == nil {
		return false
	}
	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		if !evidenceIDPattern.MatchString(value.EvidenceID) || value.Role != role || value.DisplayOrder != index+1 {
			return false
		}
		if _, duplicate := seen[value.EvidenceID]; duplicate {
			return false
		}
		seen[value.EvidenceID] = struct{}{}
	}
	return true
}

func validV2Effects(values []wireV2Effect) bool {
	if values == nil || len(values) == 0 {
		return false
	}
	for index, value := range values {
		if value.DisplayOrder != index+1 || !validText(value.Dimension, 500) ||
			(value.Direction != "up" && value.Direction != "down" && value.Direction != "stable") ||
			(value.Confidence != "high" && value.Confidence != "medium" && value.Confidence != "low" && value.Confidence != "unknown") {
			return false
		}
	}
	return true
}

func validV2TimeWindow(wire wireV2TimeWindow) bool {
	if wire.Horizons == nil || len(wire.Horizons) == 0 || !validText(wire.Label, 500) || !validNullableText(wire.Lag, 500) {
		return false
	}
	seen := map[string]struct{}{}
	for _, value := range wire.Horizons {
		if value != "immediate" && value != "short" && value != "medium" && value != "long" && value != "future" {
			return false
		}
		if _, duplicate := seen[value]; duplicate {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func effectsText(effects []wireV2Effect) string {
	parts := make([]string, len(effects))
	for index, effect := range effects {
		parts[index] = effect.Dimension + " " + strings.ToUpper(effect.Direction) + "/" + strings.ToUpper(effect.Confidence)
	}
	return strings.Join(parts, "；")
}

func aggregateV2AnchorResult(anchors []biz.Anchor) biz.Result {
	if len(anchors) == 0 {
		return biz.Result{Code: "pending", Label: "待验证"}
	}
	first := anchors[0].Result
	for _, anchor := range anchors[1:] {
		if anchor.Result.Code != first.Code {
			return biz.Result{Code: "diverging", Label: "分化"}
		}
	}
	return first
}

func aggregateV2LayerConfidence(anchors []biz.Anchor) biz.Confidence {
	return anchors[0].Confidence
}

func aggregateV2TimeWindow(anchors []biz.Anchor) string {
	return anchors[0].TimeWindow
}

func layerDisplayOrder(key string) int {
	if key == biz.LayerGeopolitics {
		return 1
	}
	return 2
}

func anchorKeys(anchors []biz.Anchor) []string {
	keys := make([]string, len(anchors))
	for index, anchor := range anchors {
		keys[index] = anchor.Key
	}
	return keys
}

func mapV2Checkpoints(values []wireCheckpoint) []biz.Checkpoint {
	result := make([]biz.Checkpoint, len(values))
	for index, value := range values {
		result[index] = biz.Checkpoint{Key: value.Key, DisplayOrder: value.DisplayOrder, Summary: value.Summary}
	}
	return result
}

func impactKeyForNode(impacts []wireV2ChainNode, nodeKey string) string {
	for _, impact := range impacts {
		if impact.NodeKey == nodeKey {
			return impact.Key
		}
	}
	return ""
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
	case "stable":
		wantLabel = "稳定"
	case "mixed":
		if !validText(wire.Label, 100) {
			return biz.Result{}, biz.ErrDataUnavailable
		}
		return biz.Result{Code: wire.Code, Label: wire.Label}, nil
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
