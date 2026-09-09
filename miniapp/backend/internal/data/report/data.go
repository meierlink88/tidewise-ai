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
		for _, tag := range item.SemanticTags {
			if (tag.Kind != "actor" && tag.Kind != "action" && tag.Kind != "object" && tag.Kind != "metric") || !validText(tag.Text, 2000) {
				return biz.EvidenceCollection{}, biz.ErrDataUnavailable
			}
		}
		items[index].SemanticTags = item.SemanticTags
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
	readAnalysisPage
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
	case "INVALID_REQUEST":
		if operation == readAnalysisPage {
			return biz.ErrInvalidRequest
		}
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
	SchemaVersion  string `json:"schema_version,omitempty"`
	AnalysisWindow *struct {
		Start string `json:"start"`
		End   string `json:"end"`
	} `json:"analysis_window,omitempty"`
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
	SemanticTags []biz.EvidenceTag `json:"semantic_tags,omitempty"`
	PublishedAt  *string           `json:"published_at"`
	Summary      string            `json:"summary"`
	Keywords     []string          `json:"keywords"`
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
	if !reportIDPattern.MatchString(wire.ID) || !validText(wire.PublisherReportID, 200) || (wire.IndustryChainCount < 0 || (wire.SchemaVersion == "" && wire.IndustryChainCount < 1)) {
		return biz.Summary{}, biz.ErrDataUnavailable
	}
	return biz.Summary{SchemaVersion: wire.SchemaVersion, ID: wire.ID, PublisherReportID: wire.PublisherReportID, GeneratedAt: generated, PublishedAt: published, IndustryChainCount: wire.IndustryChainCount}, nil
}

func validLocalKey(value string) bool {
	return value == strings.TrimSpace(value) && localKeyPattern.MatchString(value)
}
func validToken(value *string) bool { return value == nil || scopeTokenPattern.MatchString(*value) }

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

func analysisPath(q biz.AnalysisQuery) string {
	return reportsPath + "/" + url.PathEscape(q.ReportID) + "/analyses/" + url.PathEscape(q.Kind)
}
func (r *Repository) ListAnalyses(ctx context.Context, q biz.AnalysisQuery) (biz.AnalysisPage, error) {
	var p biz.AnalysisPage
	values := url.Values{"limit": {strconv.Itoa(q.Limit)}}
	if q.Cursor != "" {
		values.Set("cursor", q.Cursor)
	}
	if err := r.get(ctx, analysisPath(q)+"?"+values.Encode(), &p); err != nil {
		return p, mapReadError(err, readAnalysisPage)
	}
	seen := map[string]bool{}
	for _, u := range p.Items {
		if !validNormalizedSummary(u) || seen[u.LocalKey] {
			return p, biz.ErrDataUnavailable
		}
		seen[u.LocalKey] = true
	}
	if len(p.Items) > q.Limit || p.NextCursor != nil && (!validText(*p.NextCursor, 2048) || len(p.Items) == 0) {
		return p, biz.ErrDataUnavailable
	}
	return p, nil
}
func (r *Repository) GetAnalysis(ctx context.Context, q biz.AnalysisQuery) (biz.NormalizedDetailProjection, error) {
	var p biz.NormalizedDetailProjection
	if err := r.get(ctx, analysisPath(q)+"/"+url.PathEscape(q.Key), &p); err != nil {
		return p, mapReadError(err, readLayer)
	}
	if p.Summary.LocalKey != q.Key || !validNormalizedSummary(p.Summary) {
		return p, biz.ErrDataUnavailable
	}
	if (p.Summary.SchemaVersion == "report-publication/v5" && p.Companies == nil) || !validNormalizedProvenance(p.JudgmentOrigin, p.ReasoningSources, p.VariableSignals, p.Summary.SchemaVersion == "report-publication/v5") {
		return p, biz.ErrDataUnavailable
	}
	if p.Companies != nil {
		for _, m := range *p.Companies {
			if !validNormalizedMacro(m) || !validJudgmentOrigin(m.JudgmentOrigin, p.Summary.SchemaVersion == "report-publication/v5") {
				return p, biz.ErrDataUnavailable
			}
		}
	}
	for _, m := range p.MacroImpacts {
		if !validNormalizedMacro(m) || !validJudgmentOrigin(m.JudgmentOrigin, p.Summary.SchemaVersion == "report-publication/v5") {
			return p, biz.ErrDataUnavailable
		}
	}
	for _, c := range p.IndustryChains {
		if !validNormalizedAssessment(c.Assessment) || !validJudgmentOrigin(c.JudgmentOrigin, p.Summary.SchemaVersion == "report-publication/v5") {
			return p, biz.ErrDataUnavailable
		}
	}
	return p, nil
}
func (r *Repository) GetAnalysisChain(ctx context.Context, q biz.AnalysisQuery) (biz.NormalizedChain, error) {
	var p biz.NormalizedChain
	if err := r.get(ctx, analysisPath(q)+"/"+url.PathEscape(q.Key)+"/industry-chains/"+url.PathEscape(q.ChainKey), &p); err != nil {
		return p, mapReadError(err, readChain)
	}
	if p.LocalKey != q.ChainKey || !validNormalizedChain(p) {
		return p, biz.ErrDataUnavailable
	}
	return p, nil
}

// Evidence counts describe exactly the token's readable list, including the empty scope.
func validNormalizedScope(token *string, count int) bool {
	return count >= 0 && validToken(token) && ((count == 0) == (token == nil))
}
func validNormalizedAssessment(a biz.NormalizedAssessment) bool {
	if !validNormalizedScope(a.EvidenceScopeToken, a.EvidenceCount) {
		return false
	}
	switch a.Direction {
	case "warming", "cooling", "diverging", "pending":
	default:
		return false
	}
	if a.ConclusionBasis == "observation_only" {
		return a.Confidence == nil && a.Direction == "pending"
	}
	return a.ConclusionBasis == "reasoning_hypothesis" && a.Confidence != nil && (*a.Confidence == "low" || *a.Confidence == "medium" || *a.Confidence == "high")
}
func validNormalizedObjections(o biz.NormalizedObjections) bool {
	for _, claims := range [][]biz.NormalizedClaim{o.Counterevidence, o.Buffers} {
		for _, c := range claims {
			if !validNormalizedScope(c.EvidenceScopeToken, c.EvidenceCount) {
				return false
			}
		}
	}
	return true
}
func validNormalizedSummary(u biz.NormalizedSummaryProjection) bool {
	if (u.SchemaVersion != "report-publication/v4" && u.SchemaVersion != "report-publication/v5") || !validLocalKey(u.LocalKey) || !validText(u.Title, 10000) || u.ChainCount < 0 || !validNormalizedScope(u.Summary.EvidenceScopeToken, u.Summary.EvidenceCount) || !validNormalizedScope(u.Summary.ImpactAssessment.EvidenceScopeToken, u.Summary.ImpactAssessment.EvidenceCount) {
		return false
	}
	if !validJudgmentOrigin(u.JudgmentOrigin, u.SchemaVersion == "report-publication/v5") {
		return false
	}
	for _, a := range u.AffectedAnchors {
		if !validNormalizedAssessment(a.Assessment) || !validJudgmentOrigin(a.JudgmentOrigin, u.SchemaVersion == "report-publication/v5") {
			return false
		}
	}
	return true
}
func validNormalizedChain(c biz.NormalizedChain) bool {
	if !validNormalizedProvenance(c.JudgmentOrigin, c.ReasoningSources, c.VariableSignals, c.Graph.Scope != "") {
		return false
	}
	if c.JudgmentOrigin != "" && c.Graph.Scope != "assessed_nodes_only" {
		return false
	}
	if c.Graph.Scope != "" && (c.Graph.Scope != "assessed_nodes_only" || len(c.Graph.Nodes) != len(c.AffectedNodes) || c.EmptyState != nil) {
		return false
	}
	if !validNormalizedAssessment(c.Assessment) || !validNormalizedScope(c.ReasoningSummary.Support.EvidenceScopeToken, c.ReasoningSummary.Support.EvidenceCount) || !validNormalizedObjections(c.ReasoningSummary.Objections) {
		return false
	}
	nodes := map[string]bool{}
	for _, n := range c.Graph.Nodes {
		if !validLocalKey(n.LocalKey) || nodes[n.LocalKey] {
			return false
		}
		nodes[n.LocalKey] = true
	}
	for _, e := range c.Graph.Edges {
		if !nodes[e.FromNodeLocalKey] || !nodes[e.ToNodeLocalKey] {
			return false
		}
	}
	assessed := map[string]bool{}
	for _, n := range c.AffectedNodes {
		if !validNormalizedProvenance(n.JudgmentOrigin, n.ReasoningSources, n.VariableSignals, c.JudgmentOrigin != "") || !nodes[n.NodeLocalKey] || assessed[n.NodeLocalKey] || !validNormalizedAssessment(n.Assessment) || !validNormalizedObjections(n.Objections) {
			return false
		}
		assessed[n.NodeLocalKey] = true
	}
	return (c.EmptyState != nil) == (c.Assessment.ConclusionBasis == "observation_only") && (c.EmptyState == nil || len(c.AffectedNodes) == 0)
}

func validJudgmentOrigin(origin string, required bool) bool {
	return origin == "direct" || origin == "inferred" || (!required && origin == "")
}
func validNormalizedMacro(m biz.NormalizedMacro) bool {
	return validNormalizedAssessment(m.Assessment) && validNormalizedObjections(m.Objections) && validNormalizedProvenance(m.JudgmentOrigin, m.ReasoningSources, m.VariableSignals, false)
}

// Validate the scoped published provenance without reconstructing it from live facts.
func validNormalizedProvenance(origin string, sources *biz.NormalizedReasoningSources, signals *[]biz.NormalizedSignal, required bool) bool {
	if origin == "" {
		return !required && sources == nil && signals == nil
	}
	if !validJudgmentOrigin(origin, true) || sources == nil || signals == nil || *signals == nil || sources.SignalIDs == nil || sources.EventIDs == nil || sources.UpstreamRefs == nil {
		return false
	}
	if (origin == "direct") != (len(*signals) > 0) || len(sources.SignalIDs) != len(*signals) || len(sources.EventIDs)+len(sources.UpstreamRefs) == 0 {
		return false
	}
	events := map[string]bool{}
	for _, id := range sources.EventIDs {
		if !validText(id, 128) || events[id] {
			return false
		}
		events[id] = true
	}
	seen := map[string]bool{}
	for i, s := range *signals {
		if !validText(s.SignalID, 128) || seen[s.SignalID] || s.SignalID != sources.SignalIDs[i] || !validText(s.VariableID, 128) || !validText(s.VariableName, 16000) || !validText(s.Signal, 16000) || !validText(s.Qualification, 16000) || !validNormalizedScope(s.EvidenceScopeToken, s.EvidenceCount) || s.EvidenceCount == 0 || len(s.EventIDs) == 0 {
			return false
		}
		seen[s.SignalID] = true
		switch s.SourceDirection {
		case "UP", "DOWN", "STABLE", "MIXED", "UNKNOWN":
		default:
			return false
		}
		if s.Adoption != "adopted" && s.Adoption != "qualified" {
			return false
		}
		signalEvents := map[string]bool{}
		for _, id := range s.EventIDs {
			if !events[id] || signalEvents[id] {
				return false
			}
			signalEvents[id] = true
		}
	}
	refs := map[string]bool{}
	for _, r := range sources.UpstreamRefs {
		if !validLocalKey(r.LocalKey) || refs[r.LocalKey] || !validNullableText(r.Mechanism, 16000) || !validNullableText(r.Condition, 16000) {
			return false
		}
		refs[r.LocalKey] = true
	}
	return true
}
