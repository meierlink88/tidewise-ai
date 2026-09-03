package report

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
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
	localKeyPattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
	scopeTokenPattern = regexp.MustCompile(`^RPE[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	metadataPattern   = regexp.MustCompile(`^[A-Za-z0-9_.:-]+$`)
)

type Repository struct{ client *dataapi.HTTPClient }

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
	items := make([]biz.Summary, len(wire.Items))
	for index, item := range wire.Items {
		mapped, err := mapSummary(item)
		if err != nil {
			return biz.Page{}, err
		}
		items[index] = mapped
	}
	if query.Limit > 0 && len(items) > query.Limit {
		return biz.Page{}, biz.ErrDataUnavailable
	}
	return biz.Page{Items: items, NextCursor: wire.NextCursor}, nil
}

func (r *Repository) GetHome(ctx context.Context, reportID string) (biz.HomeSnapshot, error) {
	var wire wireHome
	if err := r.get(ctx, reportsPath+"/"+url.PathEscape(reportID)+"/home", &wire); err != nil {
		return biz.HomeSnapshot{}, mapReadError(err, readHome)
	}
	summary, err := mapSummary(wire.Report)
	if err != nil || summary.ID != reportID {
		return biz.HomeSnapshot{}, biz.ErrDataUnavailable
	}
	result := biz.HomeSnapshot{Report: summary}
	if wire.Geopolitics != nil {
		value, err := mapLayerSnapshot(*wire.Geopolitics)
		if err != nil || value.Key != biz.LayerGeopolitics {
			return biz.HomeSnapshot{}, biz.ErrDataUnavailable
		}
		result.Geopolitics = &value
	}
	if wire.Macroeconomics != nil {
		value, err := mapLayerSnapshot(*wire.Macroeconomics)
		if err != nil || value.Key != biz.LayerMacroeconomics {
			return biz.HomeSnapshot{}, biz.ErrDataUnavailable
		}
		result.Macroeconomics = &value
	}
	if (result.Geopolitics != nil) != summaryHasLayer(wire.Report.HasGeopolitics) || (result.Macroeconomics != nil) != summaryHasLayer(wire.Report.HasMacroeconomics) {
		return biz.HomeSnapshot{}, biz.ErrDataUnavailable
	}
	return result, nil
}

func (r *Repository) ListIndustryChains(ctx context.Context, query biz.ChainListQuery) (biz.IndustryChainPage, error) {
	values := make(url.Values)
	if query.Limit > 0 {
		values.Set("limit", strconv.Itoa(query.Limit))
	}
	if query.Cursor != "" {
		values.Set("cursor", query.Cursor)
	}
	path := reportsPath + "/" + url.PathEscape(query.ReportID) + "/industry-chains"
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var wire wireChainPage
	if err := r.get(ctx, path, &wire); err != nil {
		return biz.IndustryChainPage{}, mapReadError(err, readChainPage)
	}
	items := make([]biz.IndustryChainSummary, len(wire.Items))
	for index, item := range wire.Items {
		mapped, err := mapChainSummary(item)
		if err != nil {
			return biz.IndustryChainPage{}, err
		}
		items[index] = mapped
	}
	if query.Limit > 0 && len(items) > query.Limit {
		return biz.IndustryChainPage{}, biz.ErrDataUnavailable
	}
	return biz.IndustryChainPage{Items: items, NextCursor: wire.NextCursor}, nil
}

func (r *Repository) GetLayer(ctx context.Context, reportID, layerKey string) (biz.LayerDetail, error) {
	var wire wireLayerDetail
	path := reportsPath + "/" + url.PathEscape(reportID) + "/layers/" + url.PathEscape(layerKey)
	if err := r.get(ctx, path, &wire); err != nil {
		return biz.LayerDetail{}, mapReadError(err, readLayer)
	}
	summary, err := mapSummary(wire.Report)
	if err != nil || summary.ID != reportID {
		return biz.LayerDetail{}, biz.ErrDataUnavailable
	}
	layer, err := mapLayer(wire.Layer)
	if err != nil || layer.Key != layerKey {
		return biz.LayerDetail{}, biz.ErrDataUnavailable
	}
	return biz.LayerDetail{Report: summary, Layer: layer, RelatedIndustryChains: []biz.RelatedIndustryChain{}}, nil
}

func (r *Repository) GetIndustryChain(ctx context.Context, reportID, chainKey string) (biz.IndustryChainDetail, error) {
	var wire wireIndustryChainDetail
	path := reportsPath + "/" + url.PathEscape(reportID) + "/industry-chains/" + url.PathEscape(chainKey)
	if err := r.get(ctx, path, &wire); err != nil {
		return biz.IndustryChainDetail{}, mapReadError(err, readChain)
	}
	summary, err := mapSummary(wire.Report)
	if err != nil || summary.ID != reportID {
		return biz.IndustryChainDetail{}, biz.ErrDataUnavailable
	}
	chain, err := mapIndustryChain(wire.IndustryChain)
	if err != nil || chain.LocalKey != chainKey {
		return biz.IndustryChainDetail{}, biz.ErrDataUnavailable
	}
	return biz.IndustryChainDetail{Report: summary, IndustryChain: chain}, nil
}

func (r *Repository) ListEvidences(ctx context.Context, reportID, scopeToken string) (biz.EvidenceCollection, error) {
	values := url.Values{"scope_token": []string{scopeToken}}
	path := reportsPath + "/" + url.PathEscape(reportID) + "/evidences?" + values.Encode()
	var wire wireEvidenceCollection
	if err := r.get(ctx, path, &wire); err != nil {
		return biz.EvidenceCollection{}, mapReadError(err, readEvidence)
	}
	if wire.ReportID != reportID || wire.ScopeToken != scopeToken || wire.Items == nil {
		return biz.EvidenceCollection{}, biz.ErrDataUnavailable
	}
	items := make([]biz.EvidenceItem, len(wire.Items))
	for index, item := range wire.Items {
		if !validText(item.Summary, 500) || !validStringArray(item.Keywords, 20, 100) {
			return biz.EvidenceCollection{}, biz.ErrDataUnavailable
		}
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
	return biz.EvidenceCollection{ReportID: reportID, ScopeToken: scopeToken, Items: items}, nil
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
		for index := 0; index < targetType.NumField(); index++ {
			field := targetType.Field(index)
			if !field.IsExported() {
				continue
			}
			tag := strings.Split(field.Tag.Get("json"), ",")
			name := tag[0]
			if name == "-" {
				continue
			}
			if name == "" {
				name = field.Name
			}
			value, exists := object[name]
			if !exists {
				if len(tag) > 1 && tag[1] == "omitempty" {
					continue
				}
				return errors.New("required Data field is missing")
			}
			if err := validateRequiredJSON(value, field.Type); err != nil {
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
	readChainPage
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

type wireCodedLabel struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}
type wireConfidence struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}
type wireTimeWindow struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}
type wireUncertainty struct {
	Counterevidence   *string `json:"counterevidence"`
	EvidenceGap       *string `json:"evidence_gap"`
	Boundary          *string `json:"boundary"`
	ReversalCondition *string `json:"reversal_condition"`
}
type wireSummary struct {
	ID                 string `json:"id"`
	PublisherReportID  string `json:"publisher_report_id"`
	GeneratedAt        string `json:"generated_at"`
	HasGeopolitics     bool   `json:"has_geopolitics"`
	HasMacroeconomics  bool   `json:"has_macroeconomics"`
	IndustryChainCount int    `json:"industry_chain_count"`
	PublishedAt        string `json:"published_at"`
}
type wirePage struct {
	Items      []wireSummary `json:"items"`
	NextCursor *string       `json:"next_cursor"`
}
type wireTransmissionTarget struct {
	TargetType     wireCodedLabel `json:"target_type"`
	TargetLocalKey string         `json:"target_local_key"`
	TargetName     string         `json:"target_name"`
	Result         wireCodedLabel `json:"result"`
}
type wireTransmission struct {
	LocalKey          string                   `json:"local_key"`
	SourceConclusion  string                   `json:"source_conclusion"`
	Targets           []wireTransmissionTarget `json:"targets"`
	TransmissionLogic string                   `json:"transmission_logic"`
	TransmissionKind  wireCodedLabel           `json:"transmission_kind"`
	Confidence        wireConfidence           `json:"confidence"`
	Status            wireCodedLabel           `json:"status"`
}
type wireLayerSummary struct {
	Conclusion           string                   `json:"conclusion"`
	Result               wireCodedLabel           `json:"result"`
	Confidence           wireConfidence           `json:"confidence"`
	TimeWindow           wireTimeWindow           `json:"time_window"`
	DownwardTransmission wireDownwardTransmission `json:"downward_transmission"`
	Uncertainty          wireUncertainty          `json:"uncertainty"`
	EvidenceScopeToken   *string                  `json:"evidence_scope_token"`
}
type wireTransmissionGroup struct {
	Summary string             `json:"summary"`
	Paths   []wireTransmission `json:"paths"`
}
type wireDownwardTransmission struct {
	ToMacroeconomics *wireTransmissionGroup `json:"to_macroeconomics,omitempty"`
	ToIndustryChains *wireTransmissionGroup `json:"to_industry_chains,omitempty"`
}
type wireLayerSnapshot struct {
	Key     string           `json:"key"`
	Title   string           `json:"title"`
	Summary wireLayerSummary `json:"summary"`
}
type wireHome struct {
	Report         wireSummary        `json:"report"`
	Geopolitics    *wireLayerSnapshot `json:"geopolitics"`
	Macroeconomics *wireLayerSnapshot `json:"macroeconomics"`
}
type wireAnchor struct {
	LocalKey           string         `json:"local_key"`
	Name               string         `json:"name"`
	CurrentState       string         `json:"current_state"`
	Result             wireCodedLabel `json:"result"`
	ConclusionBasis    wireCodedLabel `json:"conclusion_basis"`
	ValidationStatus   wireCodedLabel `json:"validation_status"`
	Reasoning          string         `json:"reasoning"`
	TimeWindow         wireTimeWindow `json:"time_window"`
	Confidence         wireConfidence `json:"confidence"`
	EvidenceScopeToken *string        `json:"evidence_scope_token"`
}
type wireReasoningStep struct {
	LocalKey           string         `json:"local_key"`
	Input              string         `json:"input"`
	Mechanism          string         `json:"mechanism"`
	Output             string         `json:"output"`
	Confidence         wireConfidence `json:"confidence"`
	EvidenceScopeToken *string        `json:"evidence_scope_token"`
}
type wireLayer struct {
	Key             string              `json:"key"`
	Title           string              `json:"title"`
	Summary         wireLayerSummary    `json:"summary"`
	AffectedAnchors []wireAnchor        `json:"affected_anchors"`
	ReasoningSteps  []wireReasoningStep `json:"reasoning_steps"`
}
type wireLayerDetail struct {
	Report wireSummary `json:"report"`
	Layer  wireLayer   `json:"layer"`
}
type wireImpactSummary struct {
	LocalKey           string         `json:"local_key"`
	Name               string         `json:"name"`
	Result             wireCodedLabel `json:"result"`
	ConclusionBasis    wireCodedLabel `json:"conclusion_basis"`
	ValidationStatus   wireCodedLabel `json:"validation_status"`
	Confidence         wireConfidence `json:"confidence"`
	TimeWindow         wireTimeWindow `json:"time_window"`
	EvidenceScopeToken *string        `json:"evidence_scope_token"`
}
type wireChainSummary struct {
	LocalKey           string              `json:"local_key"`
	Name               string              `json:"name"`
	Conclusion         string              `json:"conclusion"`
	Result             wireCodedLabel      `json:"result"`
	Confidence         wireConfidence      `json:"confidence"`
	TimeWindow         wireTimeWindow      `json:"time_window"`
	ImpactItems        []wireImpactSummary `json:"impact_items"`
	EvidenceScopeToken *string             `json:"evidence_scope_token"`
}
type wireChainPage struct {
	Items      []wireChainSummary `json:"items"`
	NextCursor *string            `json:"next_cursor"`
}
type wireGraphNode struct {
	LocalKey string `json:"local_key"`
	Name     string `json:"name"`
}
type wireGraphEdge struct {
	FromNodeLocalKey string `json:"from_node_local_key"`
	ToNodeLocalKey   string `json:"to_node_local_key"`
	RelationLabel    string `json:"relation_label"`
}
type wireGraph struct {
	Nodes []wireGraphNode `json:"nodes"`
	Edges []wireGraphEdge `json:"edges"`
}
type wireChainNode struct {
	LocalKey           string         `json:"local_key"`
	Name               string         `json:"name"`
	Impact             string         `json:"impact"`
	Result             wireCodedLabel `json:"result"`
	ConclusionBasis    wireCodedLabel `json:"conclusion_basis"`
	ValidationStatus   wireCodedLabel `json:"validation_status"`
	Reasoning          string         `json:"reasoning"`
	TimeWindow         wireTimeWindow `json:"time_window"`
	Confidence         wireConfidence `json:"confidence"`
	EvidenceScopeToken *string        `json:"evidence_scope_token"`
}
type wireIndustryChain struct {
	LocalKey                  string          `json:"local_key"`
	Name                      string          `json:"name"`
	Conclusion                string          `json:"conclusion"`
	Result                    wireCodedLabel  `json:"result"`
	Confidence                wireConfidence  `json:"confidence"`
	TimeWindow                wireTimeWindow  `json:"time_window"`
	PathSummary               *string         `json:"path_summary"`
	AcceptedHypothesisSummary *string         `json:"accepted_hypothesis_summary"`
	Graph                     wireGraph       `json:"graph"`
	AffectedNodes             []wireChainNode `json:"affected_nodes"`
	CounterevidenceAndGap     *string         `json:"counterevidence_and_gap"`
	StopCondition             *string         `json:"stop_condition"`
	EvidenceScopeToken        *string         `json:"evidence_scope_token"`
}
type wireIndustryChainDetail struct {
	Report        wireSummary       `json:"report"`
	IndustryChain wireIndustryChain `json:"industry_chain"`
}
type wireEvidenceItem struct {
	PublishedAt *string  `json:"published_at"`
	Summary     string   `json:"summary"`
	Keywords    []string `json:"keywords"`
}
type wireEvidenceCollection struct {
	ReportID   string             `json:"report_id"`
	ScopeToken string             `json:"scope_token"`
	Items      []wireEvidenceItem `json:"items"`
}

func mapSummary(wire wireSummary) (biz.Summary, error) {
	generated, err := parseTimestamp(wire.GeneratedAt)
	if err != nil {
		return biz.Summary{}, biz.ErrDataUnavailable
	}
	published, err := parseTimestamp(wire.PublishedAt)
	if err != nil {
		return biz.Summary{}, biz.ErrDataUnavailable
	}
	if !reportIDPattern.MatchString(wire.ID) || !validText(wire.PublisherReportID, 200) || wire.IndustryChainCount < 1 {
		return biz.Summary{}, biz.ErrDataUnavailable
	}
	return biz.Summary{ID: wire.ID, PublisherReportID: wire.PublisherReportID, GeneratedAt: generated, PublishedAt: published, IndustryChainCount: wire.IndustryChainCount}, nil
}

func mapLayerSnapshot(wire wireLayerSnapshot) (biz.LayerSnapshot, error) {
	if !validLayer(wire.Key) || !validText(wire.Title, 500) {
		return biz.LayerSnapshot{}, biz.ErrDataUnavailable
	}
	if wire.Key == biz.LayerGeopolitics && (wire.Summary.DownwardTransmission.ToMacroeconomics == nil || wire.Summary.DownwardTransmission.ToIndustryChains == nil) {
		return biz.LayerSnapshot{}, biz.ErrDataUnavailable
	}
	if wire.Key == biz.LayerMacroeconomics && (wire.Summary.DownwardTransmission.ToMacroeconomics != nil || wire.Summary.DownwardTransmission.ToIndustryChains == nil) {
		return biz.LayerSnapshot{}, biz.ErrDataUnavailable
	}
	summary, err := mapLayerSummary(wire.Summary)
	if err != nil {
		return biz.LayerSnapshot{}, err
	}
	return biz.LayerSnapshot{Key: wire.Key, Title: wire.Title, Summary: summary}, nil
}

func mapLayerSummary(wire wireLayerSummary) (biz.LayerSummary, error) {
	result, err := mapCoded(wire.Result)
	if err != nil {
		return biz.LayerSummary{}, err
	}
	confidence, err := mapConfidence(wire.Confidence)
	if err != nil {
		return biz.LayerSummary{}, err
	}
	window, err := mapTimeWindow(wire.TimeWindow)
	if err != nil {
		return biz.LayerSummary{}, err
	}
	transmissions, err := mapDownwardTransmission(wire.DownwardTransmission)
	if err != nil {
		return biz.LayerSummary{}, err
	}
	if !validText(wire.Conclusion, 10_000) || !validUncertainty(wire.Uncertainty) || !validToken(wire.EvidenceScopeToken) {
		return biz.LayerSummary{}, biz.ErrDataUnavailable
	}
	return biz.LayerSummary{Conclusion: wire.Conclusion, Result: result, Confidence: confidence, TimeWindow: window, Transmissions: transmissions, Uncertainty: mapUncertainty(wire.Uncertainty), EvidenceScopeToken: wire.EvidenceScopeToken}, nil
}

func mapLayer(wire wireLayer) (biz.Layer, error) {
	snapshot, err := mapLayerSnapshot(wireLayerSnapshot{Key: wire.Key, Title: wire.Title, Summary: wire.Summary})
	if err != nil {
		return biz.Layer{}, err
	}
	anchors := make([]biz.Anchor, len(wire.AffectedAnchors))
	for index, item := range wire.AffectedAnchors {
		result, err := mapCoded(item.Result)
		if err != nil {
			return biz.Layer{}, err
		}
		basis, err := mapCoded(item.ConclusionBasis)
		if err != nil {
			return biz.Layer{}, err
		}
		status, err := mapCoded(item.ValidationStatus)
		if err != nil {
			return biz.Layer{}, err
		}
		confidence, err := mapConfidence(item.Confidence)
		if err != nil {
			return biz.Layer{}, err
		}
		window, err := mapTimeWindow(item.TimeWindow)
		if err != nil {
			return biz.Layer{}, err
		}
		if !validLocalKey(item.LocalKey) || !validText(item.Name, 500) || !validText(item.CurrentState, 10_000) || !validText(item.Reasoning, 10_000) || !validToken(item.EvidenceScopeToken) {
			return biz.Layer{}, biz.ErrDataUnavailable
		}
		anchors[index] = biz.Anchor{LocalKey: item.LocalKey, Name: item.Name, CurrentState: item.CurrentState, Result: result, ConclusionBasis: basis, ValidationStatus: status, Reasoning: item.Reasoning, TimeWindow: window, Confidence: confidence, EvidenceScopeToken: item.EvidenceScopeToken}
	}
	steps := make([]biz.ReasoningStep, len(wire.ReasoningSteps))
	for index, item := range wire.ReasoningSteps {
		confidence, err := mapConfidence(item.Confidence)
		if err != nil {
			return biz.Layer{}, err
		}
		if !validLocalKey(item.LocalKey) || !validText(item.Input, 10_000) || !validText(item.Mechanism, 10_000) || !validText(item.Output, 10_000) || !validToken(item.EvidenceScopeToken) {
			return biz.Layer{}, biz.ErrDataUnavailable
		}
		steps[index] = biz.ReasoningStep{LocalKey: item.LocalKey, Input: item.Input, Mechanism: item.Mechanism, Output: item.Output, Confidence: confidence, EvidenceScopeToken: item.EvidenceScopeToken}
	}
	return biz.Layer{Key: snapshot.Key, Title: snapshot.Title, Conclusion: snapshot.Summary.Conclusion, Result: snapshot.Summary.Result, Confidence: snapshot.Summary.Confidence, TimeWindow: snapshot.Summary.TimeWindow, Anchors: anchors, ReasoningSteps: steps, Transmissions: snapshot.Summary.Transmissions, Uncertainty: snapshot.Summary.Uncertainty, EvidenceScopeToken: snapshot.Summary.EvidenceScopeToken}, nil
}

func mapTransmissions(values []wireTransmission) ([]biz.Transmission, error) {
	if values == nil {
		return nil, biz.ErrDataUnavailable
	}
	result := make([]biz.Transmission, len(values))
	for index, item := range values {
		kind, err := mapCoded(item.TransmissionKind)
		if err != nil {
			return nil, err
		}
		confidence, err := mapConfidence(item.Confidence)
		if err != nil {
			return nil, err
		}
		status, err := mapCoded(item.Status)
		if err != nil {
			return nil, err
		}
		if !validLocalKey(item.LocalKey) || !validText(item.SourceConclusion, 10_000) || !validText(item.TransmissionLogic, 10_000) || len(item.Targets) == 0 {
			return nil, biz.ErrDataUnavailable
		}
		targets := make([]biz.TransmissionTarget, len(item.Targets))
		for targetIndex, target := range item.Targets {
			mappedResult, err := mapCoded(target.Result)
			targetType, typeErr := mapCoded(target.TargetType)
			if err != nil || typeErr != nil || !validReference(targetType.Code, target.TargetLocalKey) || !validText(target.TargetName, 500) {
				return nil, biz.ErrDataUnavailable
			}
			targets[targetIndex] = biz.TransmissionTarget{Ref: biz.Reference{Type: targetType.Code, LocalKey: target.TargetLocalKey}, Name: target.TargetName, Result: mappedResult}
		}
		result[index] = biz.Transmission{LocalKey: item.LocalKey, SourceConclusion: item.SourceConclusion, Targets: targets, Logic: item.TransmissionLogic, Kind: kind, Confidence: confidence, Status: status}
	}
	return result, nil
}

func mapDownwardTransmission(value wireDownwardTransmission) ([]biz.Transmission, error) {
	result := []biz.Transmission{}
	for _, group := range []*wireTransmissionGroup{value.ToMacroeconomics, value.ToIndustryChains} {
		if group == nil {
			continue
		}
		if !validText(group.Summary, 10_000) {
			return nil, biz.ErrDataUnavailable
		}
		paths, err := mapTransmissions(group.Paths)
		if err != nil {
			return nil, err
		}
		result = append(result, paths...)
	}
	return result, nil
}

func mapChainSummary(wire wireChainSummary) (biz.IndustryChainSummary, error) {
	result, err := mapCoded(wire.Result)
	if err != nil {
		return biz.IndustryChainSummary{}, err
	}
	confidence, err := mapConfidence(wire.Confidence)
	if err != nil {
		return biz.IndustryChainSummary{}, err
	}
	window, err := mapTimeWindow(wire.TimeWindow)
	if err != nil {
		return biz.IndustryChainSummary{}, err
	}
	if !validLocalKey(wire.LocalKey) || !validText(wire.Name, 500) || !validText(wire.Conclusion, 10_000) || wire.ImpactItems == nil || !validToken(wire.EvidenceScopeToken) {
		return biz.IndustryChainSummary{}, biz.ErrDataUnavailable
	}
	impacts := make([]biz.IndustryChainImpactSummary, len(wire.ImpactItems))
	for index, item := range wire.ImpactItems {
		itemResult, err := mapCoded(item.Result)
		if err != nil {
			return biz.IndustryChainSummary{}, err
		}
		basis, err := mapCoded(item.ConclusionBasis)
		if err != nil {
			return biz.IndustryChainSummary{}, err
		}
		status, err := mapCoded(item.ValidationStatus)
		if err != nil {
			return biz.IndustryChainSummary{}, err
		}
		itemConfidence, err := mapConfidence(item.Confidence)
		if err != nil {
			return biz.IndustryChainSummary{}, err
		}
		itemWindow, err := mapTimeWindow(item.TimeWindow)
		if err != nil {
			return biz.IndustryChainSummary{}, err
		}
		if !validLocalKey(item.LocalKey) || !validText(item.Name, 500) || !validToken(item.EvidenceScopeToken) {
			return biz.IndustryChainSummary{}, biz.ErrDataUnavailable
		}
		impacts[index] = biz.IndustryChainImpactSummary{LocalKey: item.LocalKey, Name: item.Name, Result: itemResult, ConclusionBasis: basis, ValidationStatus: status, Confidence: itemConfidence, TimeWindow: itemWindow, EvidenceScopeToken: item.EvidenceScopeToken}
	}
	return biz.IndustryChainSummary{LocalKey: wire.LocalKey, Name: wire.Name, Conclusion: wire.Conclusion, Result: result, Confidence: confidence, TimeWindow: window, ImpactItems: impacts, EvidenceScopeToken: wire.EvidenceScopeToken}, nil
}

func mapIndustryChain(wire wireIndustryChain) (biz.IndustryChain, error) {
	result, err := mapCoded(wire.Result)
	if err != nil {
		return biz.IndustryChain{}, err
	}
	confidence, err := mapConfidence(wire.Confidence)
	if err != nil {
		return biz.IndustryChain{}, err
	}
	window, err := mapTimeWindow(wire.TimeWindow)
	if err != nil {
		return biz.IndustryChain{}, err
	}
	if !validLocalKey(wire.LocalKey) || !validText(wire.Name, 500) || !validText(wire.Conclusion, 10_000) || !validNullableText(wire.PathSummary, 10_000) || !validNullableText(wire.AcceptedHypothesisSummary, 10_000) || !validNullableText(wire.CounterevidenceAndGap, 10_000) || !validNullableText(wire.StopCondition, 10_000) || wire.Graph.Nodes == nil || wire.Graph.Edges == nil || wire.AffectedNodes == nil || !validToken(wire.EvidenceScopeToken) {
		return biz.IndustryChain{}, biz.ErrDataUnavailable
	}
	topology := map[string]struct{}{}
	topologyNodes := make([]biz.IndustryChainTopologyNode, len(wire.Graph.Nodes))
	for index, node := range wire.Graph.Nodes {
		if !validLocalKey(node.LocalKey) || !validText(node.Name, 500) {
			return biz.IndustryChain{}, biz.ErrDataUnavailable
		}
		if _, duplicate := topology[node.LocalKey]; duplicate {
			return biz.IndustryChain{}, biz.ErrDataUnavailable
		}
		topology[node.LocalKey] = struct{}{}
		topologyNodes[index] = biz.IndustryChainTopologyNode{LocalKey: node.LocalKey, Name: node.Name}
	}
	edges := make([]biz.IndustryChainEdge, len(wire.Graph.Edges))
	for index, edge := range wire.Graph.Edges {
		_, from := topology[edge.FromNodeLocalKey]
		_, to := topology[edge.ToNodeLocalKey]
		if !validText(edge.RelationLabel, 500) || !from || !to || edge.FromNodeLocalKey == edge.ToNodeLocalKey {
			return biz.IndustryChain{}, biz.ErrDataUnavailable
		}
		edges[index] = biz.IndustryChainEdge{FromNodeKey: edge.FromNodeLocalKey, ToNodeKey: edge.ToNodeLocalKey, RelationLabel: edge.RelationLabel}
	}
	nodes := make([]biz.IndustryChainNode, len(wire.AffectedNodes))
	assessedTopology := map[string]struct{}{}
	assessmentKeys := map[string]struct{}{}
	for index, item := range wire.AffectedNodes {
		if _, ok := topology[item.LocalKey]; !ok {
			return biz.IndustryChain{}, biz.ErrDataUnavailable
		}
		if _, duplicate := assessedTopology[item.LocalKey]; duplicate {
			return biz.IndustryChain{}, biz.ErrDataUnavailable
		}
		if _, duplicate := assessmentKeys[item.LocalKey]; duplicate {
			return biz.IndustryChain{}, biz.ErrDataUnavailable
		}
		assessedTopology[item.LocalKey] = struct{}{}
		assessmentKeys[item.LocalKey] = struct{}{}
		itemResult, err := mapCoded(item.Result)
		if err != nil {
			return biz.IndustryChain{}, err
		}
		basis, err := mapCoded(item.ConclusionBasis)
		if err != nil {
			return biz.IndustryChain{}, err
		}
		status, err := mapCoded(item.ValidationStatus)
		if err != nil {
			return biz.IndustryChain{}, err
		}
		itemConfidence, err := mapConfidence(item.Confidence)
		if err != nil {
			return biz.IndustryChain{}, err
		}
		itemWindow, err := mapTimeWindow(item.TimeWindow)
		if err != nil {
			return biz.IndustryChain{}, err
		}
		if !validLocalKey(item.LocalKey) || !validText(item.Name, 500) || !validText(item.Impact, 10_000) || !validText(item.Reasoning, 10_000) || !validToken(item.EvidenceScopeToken) {
			return biz.IndustryChain{}, biz.ErrDataUnavailable
		}
		nodes[index] = biz.IndustryChainNode{LocalKey: item.LocalKey, Name: item.Name, Impact: item.Impact, Result: itemResult, ConclusionBasis: basis, ValidationStatus: status, Reasoning: item.Reasoning, TimeWindow: itemWindow, Confidence: itemConfidence, EvidenceScopeToken: item.EvidenceScopeToken}
	}
	return biz.IndustryChain{LocalKey: wire.LocalKey, Name: wire.Name, Conclusion: wire.Conclusion, Result: result, Confidence: confidence, TimeWindow: window, PathSummary: wire.PathSummary, AcceptedHypothesisSummary: wire.AcceptedHypothesisSummary, TopologyNodes: topologyNodes, Nodes: nodes, Edges: edges, CounterevidenceAndGap: wire.CounterevidenceAndGap, StopCondition: wire.StopCondition, EvidenceScopeToken: wire.EvidenceScopeToken}, nil
}

func mapCoded(wire wireCodedLabel) (biz.CodedLabel, error) {
	if !validText(wire.Code, 100) || !validText(wire.Label, 500) {
		return biz.CodedLabel{}, biz.ErrDataUnavailable
	}
	return biz.CodedLabel{Code: wire.Code, Label: wire.Label}, nil
}
func mapConfidence(wire wireConfidence) (biz.Confidence, error) {
	if !validText(wire.Code, 100) || !validText(wire.Label, 500) {
		return biz.Confidence{}, biz.ErrDataUnavailable
	}
	return biz.Confidence{Code: wire.Code, Label: wire.Label}, nil
}
func mapTimeWindow(wire wireTimeWindow) (biz.TimeWindow, error) {
	if !validText(wire.Code, 100) || !validText(wire.Label, 500) {
		return biz.TimeWindow{}, biz.ErrDataUnavailable
	}
	return biz.TimeWindow{Code: wire.Code, Label: wire.Label}, nil
}
func mapUncertainty(wire wireUncertainty) biz.LayerUncertainty {
	return biz.LayerUncertainty{Counterevidence: wire.Counterevidence, EvidenceGap: wire.EvidenceGap, Boundary: wire.Boundary, ReversalCondition: wire.ReversalCondition}
}
func validUncertainty(value wireUncertainty) bool {
	return validNullableText(value.Counterevidence, 10_000) && validNullableText(value.EvidenceGap, 10_000) && validNullableText(value.Boundary, 10_000) && validNullableText(value.ReversalCondition, 10_000)
}
func validReference(kind, key string) bool {
	switch kind {
	case "section":
		return validLayer(key)
	case "macro_anchor", "industry_chain", "industry_chain_node":
		return validLocalKey(key)
	default:
		return false
	}
}
func validLayer(value string) bool {
	return value == biz.LayerGeopolitics || value == biz.LayerMacroeconomics
}
func validLocalKey(value string) bool {
	return value == strings.TrimSpace(value) && localKeyPattern.MatchString(value)
}
func validToken(value *string) bool   { return value == nil || scopeTokenPattern.MatchString(*value) }
func summaryHasLayer(value bool) bool { return value }
func validStringArray(values []string, maxItems, maxText int) bool {
	if values == nil || (maxItems > 0 && len(values) > maxItems) {
		return false
	}
	seen := map[string]struct{}{}
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
