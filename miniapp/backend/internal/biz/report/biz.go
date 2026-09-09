package report

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
)

const (
	LayerGeopolitics    = "geopolitics"
	LayerMacroeconomics = "macroeconomics"
	SelectionToday      = "today"
	SelectionFallback   = "latest_fallback"
	listPageSize        = 100
	chainPageSize       = 20
	maxListPages        = 100
)

var (
	ErrInvalidRequest        = errors.New("invalid report request")
	ErrReportNotFound        = errors.New("report not found")
	ErrLayerNotFound         = errors.New("report layer not found")
	ErrChainNotFound         = errors.New("report industry chain not found")
	ErrEvidenceScopeNotFound = errors.New("report evidence scope not found")
	ErrDataUnavailable       = errors.New("report data unavailable")
	reportIDPattern          = regexp.MustCompile(`^RPT[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	localKeyPattern          = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
	scopeTokenPattern        = regexp.MustCompile(`^RPE[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
)

type ListQuery struct {
	PublishedFrom, PublishedTo *time.Time
	Limit                      int
	Cursor                     string
}
type ChainListQuery struct {
	ReportID string
	Limit    int
	Cursor   string
}

type Summary struct {
	SchemaVersion            string
	ID, PublisherReportID    string
	GeneratedAt, PublishedAt time.Time
	IndustryChainCount       int
}
type Page struct {
	Items      []Summary
	NextCursor *string
}
type CodedLabel struct{ Code, Label string }
type Confidence struct{ Code, Label string }
type TimeWindow struct{ Code, Label string }
type Reference struct{ Type, LocalKey string }

type LayerUncertainty struct{ Counterevidence, EvidenceGap, Boundary, ReversalCondition *string }
type LayerSummary struct {
	Conclusion         string
	Result             CodedLabel
	Confidence         Confidence
	TimeWindow         TimeWindow
	Transmissions      []Transmission
	Uncertainty        LayerUncertainty
	EvidenceScopeToken *string
}
type LayerSnapshot struct {
	Key, Title string
	Summary    LayerSummary
}
type HomeSnapshot struct {
	Report                      Summary
	Geopolitics, Macroeconomics *LayerSnapshot
}

type Anchor struct {
	LocalKey, Name, CurrentState string
	Result                       CodedLabel
	ConclusionBasis              CodedLabel
	ValidationStatus             CodedLabel
	Reasoning                    string
	TimeWindow                   TimeWindow
	Confidence                   Confidence
	EvidenceScopeToken           *string
}
type ReasoningStep struct {
	LocalKey, Input, Mechanism, Output string
	Confidence                         Confidence
	EvidenceScopeToken                 *string
}
type TransmissionTarget struct {
	Ref    Reference
	Name   string
	Result CodedLabel
}
type Transmission struct {
	LocalKey, SourceConclusion string
	Targets                    []TransmissionTarget
	Logic                      string
	Kind                       CodedLabel
	Confidence                 Confidence
	Status                     CodedLabel
}
type Layer struct {
	Key, Title, Conclusion string
	Result                 CodedLabel
	Confidence             Confidence
	TimeWindow             TimeWindow
	Anchors                []Anchor
	ReasoningSteps         []ReasoningStep
	Transmissions          []Transmission
	Uncertainty            LayerUncertainty
	EvidenceScopeToken     *string
}
type LayerDetail struct {
	Report                Summary
	Layer                 Layer
	RelatedIndustryChains []RelatedIndustryChain
}
type RelatedIndustryChain struct {
	LocalKey, Name string
	Result         CodedLabel
}

type IndustryChainImpactSummary struct {
	LocalKey, Name                    string
	Result                            CodedLabel
	ConclusionBasis, ValidationStatus CodedLabel
	Confidence                        Confidence
	TimeWindow                        TimeWindow
	EvidenceScopeToken                *string
}
type IndustryChainSummary struct {
	LocalKey, Name, Conclusion string
	Result                     CodedLabel
	Confidence                 Confidence
	TimeWindow                 TimeWindow
	ImpactItems                []IndustryChainImpactSummary
	EvidenceScopeToken         *string
}
type IndustryChainPage struct {
	Items      []IndustryChainSummary
	NextCursor *string
}
type IndustryChainNode struct {
	LocalKey, Name, Impact            string
	Result                            CodedLabel
	ConclusionBasis, ValidationStatus CodedLabel
	Reasoning                         string
	TimeWindow                        TimeWindow
	Confidence                        Confidence
	EvidenceScopeToken                *string
}
type IndustryChainEdge struct {
	FromNodeKey, ToNodeKey string
	RelationLabel          string
}
type IndustryChainTopologyNode struct {
	LocalKey, Name string
}
type IndustryChain struct {
	LocalKey, Name, Conclusion             string
	Result                                 CodedLabel
	Confidence                             Confidence
	TimeWindow                             TimeWindow
	PathSummary, AcceptedHypothesisSummary *string
	TopologyNodes                          []IndustryChainTopologyNode
	Nodes                                  []IndustryChainNode
	Edges                                  []IndustryChainEdge
	CounterevidenceAndGap, StopCondition   *string
	EvidenceScopeToken                     *string
}
type IndustryChainDetail struct {
	Report        Summary
	IndustryChain IndustryChain
}

type EvidenceItem struct {
	SemanticTags []EvidenceTag
	PublishedAt  *time.Time
	Summary      string
	Keywords     []string
}
type EvidenceCollection struct {
	ReportID, ScopeToken string
	Items                []EvidenceItem
}

type CardImpactItem struct {
	Ref                               Reference
	Name                              string
	Result                            CodedLabel
	ConclusionBasis, ValidationStatus CodedLabel
	Confidence                        Confidence
	TimeWindow                        TimeWindow
	EvidenceScopeToken                *string
}
type Card struct {
	LocalKey, Kind              string
	DetailRef                   Reference
	Title, Subtitle, Conclusion string
	Result                      CodedLabel
	Confidence                  Confidence
	TimeWindow                  TimeWindow
	ImpactItems                 []CardImpactItem
	EvidenceScopeToken          *string
}
type CardPage struct {
	Items      []Card
	NextCursor *string
}
type Home struct {
	AnalysisGroups []AnalysisGroup
	Report         Summary
	Cards          []Card
	NextCursor     *string
}
type HomeSelection struct{ Mode, Date, Timezone string }
type HomeCollection struct {
	Selection HomeSelection
	Reports   []Home
}

type Repository interface {
	ListAnalyses(context.Context, AnalysisQuery) (AnalysisPage, error)
	GetAnalysis(context.Context, AnalysisQuery) (NormalizedDetailProjection, error)
	GetAnalysisChain(context.Context, AnalysisQuery) (NormalizedChain, error)
	ListReports(context.Context, ListQuery) (Page, error)

	ListEvidences(context.Context, string, string) (EvidenceCollection, error)
}

type UseCase struct {
	repository Repository
	now        func() time.Time
}

func NewUseCase(repository Repository) *UseCase { return NewUseCaseWithClock(repository, time.Now) }
func NewUseCaseWithClock(repository Repository, now func() time.Time) *UseCase {
	if now == nil {
		now = time.Now
	}
	return &UseCase{repository: repository, now: now}
}

func (u *UseCase) Home(ctx context.Context) (HomeCollection, error) {
	if u == nil || u.repository == nil {
		return HomeCollection{}, ErrDataUnavailable
	}
	from, to, date := shanghaiDay(u.now())
	selection := HomeSelection{Mode: SelectionToday, Date: date, Timezone: "Asia/Shanghai"}
	summary, err := u.latestSummary(ctx, ListQuery{PublishedFrom: &from, PublishedTo: &to, Limit: 1})
	if err != nil {
		return HomeCollection{}, err
	}
	if summary == nil {
		summary, err = u.latestSummary(ctx, ListQuery{Limit: 1})
		if err != nil {
			return HomeCollection{}, err
		}
		if summary != nil {
			selection.Mode = SelectionFallback
		}
	}
	if summary == nil {
		return HomeCollection{Selection: selection, Reports: []Home{}}, nil
	}
	home, err := u.readHome(ctx, *summary)
	if err != nil {
		return HomeCollection{}, err
	}
	return HomeCollection{Selection: selection, Reports: []Home{home}}, nil
}

func (u *UseCase) latestSummary(ctx context.Context, query ListQuery) (*Summary, error) {
	page, err := u.repository.ListReports(ctx, query)
	if err != nil {
		return nil, normalizeRepositoryError(err)
	}
	if len(page.Items) > 1 || (len(page.Items) == 0 && page.NextCursor != nil) || validateSummaryOrder(page.Items) != nil {
		return nil, ErrDataUnavailable
	}
	if len(page.Items) == 0 {
		return nil, nil
	}
	return &page.Items[0], nil
}

func (u *UseCase) readHome(ctx context.Context, summary Summary) (Home, error) {
	if summary.SchemaVersion == "report-publication/v4" || summary.SchemaVersion == "report-publication/v5" {
		home := Home{Report: summary, Cards: []Card{}, AnalysisGroups: []AnalysisGroup{}}
		kinds := []string{"geopolitical_stories", "macroeconomic_stories", "concept_analyses"}
		if summary.SchemaVersion == "report-publication/v5" {
			kinds = append(kinds, "industry_chain_analyses")
		}
		for _, kind := range kinds {
			page, err := u.Analyses(ctx, AnalysisQuery{ReportID: summary.ID, Kind: kind, Limit: 20})
			if err != nil {
				return Home{}, err
			}
			home.AnalysisGroups = append(home.AnalysisGroups, AnalysisGroup{Kind: kind, Items: page.Items, NextCursor: page.NextCursor})
		}
		return home, nil
	}
	return Home{}, ErrDataUnavailable
}

func (u *UseCase) Evidences(ctx context.Context, reportID, scopeToken string) (EvidenceCollection, error) {
	if !validReportID(reportID) || !scopeTokenPattern.MatchString(scopeToken) {
		return EvidenceCollection{}, ErrInvalidRequest
	}
	if u == nil || u.repository == nil {
		return EvidenceCollection{}, ErrDataUnavailable
	}
	value, err := u.repository.ListEvidences(ctx, reportID, scopeToken)
	if err != nil {
		return EvidenceCollection{}, normalizeRepositoryError(err)
	}
	if value.ReportID != reportID || value.ScopeToken != scopeToken {
		return EvidenceCollection{}, ErrDataUnavailable
	}
	return value, nil
}

func shanghaiDay(now time.Time) (time.Time, time.Time, string) {
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	local := now.In(location)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	return start.UTC(), start.AddDate(0, 0, 1).UTC(), start.Format("2006-01-02")
}
func validReportID(value string) bool {
	return value == strings.TrimSpace(value) && reportIDPattern.MatchString(value)
}
func validLayer(value string) bool { return value == LayerGeopolitics || value == LayerMacroeconomics }
func validLocalKey(value string) bool {
	return value == strings.TrimSpace(value) && localKeyPattern.MatchString(value)
}

func validateSummaryOrder(items []Summary) error {
	seen := map[string]struct{}{}
	for index, item := range items {
		if !validReportID(item.ID) || strings.TrimSpace(item.PublisherReportID) == "" || item.GeneratedAt.IsZero() || item.PublishedAt.IsZero() || (item.IndustryChainCount < 0 || item.SchemaVersion == "" && item.IndustryChainCount < 1) {
			return ErrDataUnavailable
		}
		if _, duplicate := seen[item.ID]; duplicate {
			return ErrDataUnavailable
		}
		seen[item.ID] = struct{}{}
		if index > 0 {
			previous := items[index-1]
			if item.PublishedAt.After(previous.PublishedAt) || (item.PublishedAt.Equal(previous.PublishedAt) && item.ID < previous.ID) {
				return ErrDataUnavailable
			}
		}
	}
	return nil
}

func normalizeRepositoryError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, ErrInvalidRequest):
		return ErrInvalidRequest
	case errors.Is(err, ErrReportNotFound):
		return ErrReportNotFound
	case errors.Is(err, ErrLayerNotFound):
		return ErrLayerNotFound
	case errors.Is(err, ErrChainNotFound):
		return ErrChainNotFound
	case errors.Is(err, ErrEvidenceScopeNotFound):
		return ErrEvidenceScopeNotFound
	default:
		return ErrDataUnavailable
	}
}

type NormalizedClaim struct {
	Text               string  `json:"text"`
	Basis              string  `json:"basis"`
	EvidenceScopeToken *string `json:"evidence_scope_token"`
	EvidenceCount      int     `json:"evidence_count"`
}
type NormalizedObjections struct {
	Summary               string            `json:"summary"`
	Counterevidence       []NormalizedClaim `json:"counterevidence"`
	Buffers               []NormalizedClaim `json:"buffers"`
	CounterevidenceStatus string            `json:"counterevidence_status"`
	EvidenceGaps          []string          `json:"evidence_gaps"`
	ScopeLimits           []string          `json:"scope_limits"`
}
type NormalizedWindow struct {
	Kind        string  `json:"kind"`
	Description string  `json:"description"`
	StartAt     *string `json:"start_at"`
	EndAt       *string `json:"end_at"`
}
type NormalizedAssessment struct {
	Conclusion         string           `json:"conclusion"`
	Direction          string           `json:"direction"`
	ConclusionBasis    string           `json:"conclusion_basis"`
	ValidationStatus   string           `json:"validation_status"`
	Confidence         *string          `json:"confidence"`
	ForecastWindow     NormalizedWindow `json:"forecast_window"`
	Scope              string           `json:"scope"`
	Conditions         []string         `json:"conditions"`
	FollowUp           []string         `json:"follow_up"`
	TransmissionLogic  string           `json:"transmission_logic"`
	EvidenceScopeToken *string          `json:"evidence_scope_token"`
	EvidenceCount      int              `json:"evidence_count"`
}
type NormalizedNode struct {
	JudgmentOrigin   string                      `json:"judgment_origin,omitempty"`
	ReasoningSources *NormalizedReasoningSources `json:"reasoning_sources,omitempty"`
	VariableSignals  *[]NormalizedSignal         `json:"variable_signals,omitempty"`
	LocalKey         string                      `json:"local_key"`
	SourceID         string                      `json:"source_id"`
	NodeLocalKey     string                      `json:"node_local_key"`
	Name             string                      `json:"name"`
	Assessment       NormalizedAssessment        `json:"assessment"`
	Objections       NormalizedObjections        `json:"objections"`
}
type NormalizedGraph struct {
	Scope string                     `json:"scope,omitempty"`
	Nodes []NormalizedGraphNodesItem `json:"nodes"`
	Edges []NormalizedGraphEdgesItem `json:"edges"`
}
type NormalizedChain struct {
	JudgmentOrigin   string                          `json:"judgment_origin,omitempty"`
	ReasoningSources *NormalizedReasoningSources     `json:"reasoning_sources,omitempty"`
	VariableSignals  *[]NormalizedSignal             `json:"variable_signals,omitempty"`
	LocalKey         string                          `json:"local_key"`
	SourceID         string                          `json:"source_id"`
	Name             string                          `json:"name"`
	Assessment       NormalizedAssessment            `json:"assessment"`
	ReasoningSummary NormalizedChainReasoningSummary `json:"reasoning_summary"`
	Graph            NormalizedGraph                 `json:"graph"`
	AffectedNodes    []NormalizedNode                `json:"affected_nodes"`
	EmptyState       *NormalizedChainEmptyState      `json:"empty_state"`
}
type NormalizedMacro struct {
	JudgmentOrigin   string                      `json:"judgment_origin,omitempty"`
	ReasoningSources *NormalizedReasoningSources `json:"reasoning_sources,omitempty"`
	VariableSignals  *[]NormalizedSignal         `json:"variable_signals,omitempty"`
	LocalKey         string                      `json:"local_key"`
	SourceID         string                      `json:"source_id"`
	Name             string                      `json:"name"`
	Assessment       NormalizedAssessment        `json:"assessment"`
	Objections       NormalizedObjections        `json:"objections"`
}
type NormalizedAnchorRef struct {
	TargetType    string  `json:"target_type"`
	LocalKey      string  `json:"local_key"`
	ChainLocalKey *string `json:"chain_local_key"`
}
type NormalizedGraphNodesItem struct {
	LocalKey string `json:"local_key"`
	SourceID string `json:"source_id"`
	Name     string `json:"name"`
}
type NormalizedGraphEdgesItem struct {
	FromNodeLocalKey string `json:"from_node_local_key"`
	ToNodeLocalKey   string `json:"to_node_local_key"`
	RelationLabel    string `json:"relation_label"`
}
type NormalizedChainReasoningSummary struct {
	Logic      string               `json:"logic"`
	Support    NormalizedClaim      `json:"support"`
	Objections NormalizedObjections `json:"objections"`
}
type NormalizedChainEmptyState struct {
	Code     string   `json:"code"`
	Reason   string   `json:"reason"`
	FollowUp []string `json:"follow_up"`
}
type NormalizedUnitSummary struct {
	Conclusion         string                                `json:"conclusion"`
	TransmissionLogic  string                                `json:"transmission_logic"`
	ImpactAssessment   NormalizedUnitSummaryImpactAssessment `json:"impact_assessment"`
	AffectedRefs       []NormalizedAnchorRef                 `json:"affected_refs"`
	EvidenceScopeToken *string                               `json:"evidence_scope_token"`
	EvidenceCount      int                                   `json:"evidence_count"`
}
type NormalizedUnitSummaryImpactAssessment struct {
	Level              string  `json:"level"`
	Rationale          string  `json:"rationale"`
	EvidenceScopeToken *string `json:"evidence_scope_token"`
	EvidenceCount      int     `json:"evidence_count"`
}

type NormalizedResolvedAnchor struct {
	JudgmentOrigin string               `json:"judgment_origin,omitempty"`
	Reference      NormalizedAnchorRef  `json:"reference"`
	SourceID       string               `json:"source_id"`
	Name           string               `json:"name"`
	Assessment     NormalizedAssessment `json:"assessment"`
}
type NormalizedSummaryProjection struct {
	JudgmentOrigin  string                     `json:"judgment_origin,omitempty"`
	SchemaVersion   string                     `json:"schema_version"`
	LocalKey        string                     `json:"local_key"`
	SourceID        string                     `json:"source_id"`
	Title           string                     `json:"title"`
	Summary         NormalizedUnitSummary      `json:"summary"`
	AffectedAnchors []NormalizedResolvedAnchor `json:"affected_anchors"`
	ChainCount      int                        `json:"chain_count"`
}
type NormalizedChainHeader struct {
	JudgmentOrigin string                     `json:"judgment_origin,omitempty"`
	LocalKey       string                     `json:"local_key"`
	SourceID       string                     `json:"source_id"`
	Name           string                     `json:"name"`
	Assessment     NormalizedAssessment       `json:"assessment"`
	EmptyState     *NormalizedChainEmptyState `json:"empty_state"`
}
type NormalizedDetailProjection struct {
	PublishedAt      *time.Time                  `json:"-"`
	JudgmentOrigin   string                      `json:"judgment_origin,omitempty"`
	ReasoningSources *NormalizedReasoningSources `json:"reasoning_sources,omitempty"`
	VariableSignals  *[]NormalizedSignal         `json:"variable_signals,omitempty"`
	Companies        *[]NormalizedMacro          `json:"companies,omitempty"`
	Summary          NormalizedSummaryProjection `json:"summary"`
	MacroImpacts     []NormalizedMacro           `json:"macro_impacts"`
	IndustryChains   []NormalizedChainHeader     `json:"industry_chains"`
}

type AnalysisPage struct {
	Items      []NormalizedSummaryProjection `json:"items"`
	NextCursor *string                       `json:"next_cursor"`
}
type AnalysisGroup struct {
	Kind       string                        `json:"kind"`
	Items      []NormalizedSummaryProjection `json:"items"`
	NextCursor *string                       `json:"next_cursor"`
}
type AnalysisQuery struct {
	ReportID, Kind, Key, ChainKey, Cursor string
	Limit                                 int
}

func validAnalysisQuery(q AnalysisQuery) bool {
	return validReportID(q.ReportID) && (q.Kind == "geopolitical_stories" || q.Kind == "macroeconomic_stories" || q.Kind == "concept_analyses" || q.Kind == "industry_chain_analyses") && q.Limit >= 0 && q.Limit <= 100 && len(q.Cursor) <= 2048
}
func (u *UseCase) Analyses(ctx context.Context, q AnalysisQuery) (AnalysisPage, error) {
	if u == nil || u.repository == nil || !validAnalysisQuery(q) {
		return AnalysisPage{}, ErrInvalidRequest
	}
	if q.Limit == 0 {
		q.Limit = 20
	}
	p, err := u.repository.ListAnalyses(ctx, q)
	return p, normalizeRepositoryError(err)
}
func (u *UseCase) Analysis(ctx context.Context, q AnalysisQuery) (NormalizedDetailProjection, error) {
	if u == nil || u.repository == nil || !validAnalysisQuery(q) || !localKeyPattern.MatchString(q.Key) {
		return NormalizedDetailProjection{}, ErrInvalidRequest
	}
	p, err := u.repository.GetAnalysis(ctx, q)
	if err != nil {
		return p, normalizeRepositoryError(err)
	}
	publishedAt, err := u.reportPublication(ctx, q.ReportID)
	if err != nil {
		return NormalizedDetailProjection{}, normalizeRepositoryError(err)
	}
	p.PublishedAt = &publishedAt
	return p, nil
}
func (u *UseCase) AnalysisChain(ctx context.Context, q AnalysisQuery) (NormalizedChain, error) {
	if u == nil || u.repository == nil || !validAnalysisQuery(q) || !localKeyPattern.MatchString(q.Key) || !localKeyPattern.MatchString(q.ChainKey) {
		return NormalizedChain{}, ErrInvalidRequest
	}
	p, err := u.repository.GetAnalysisChain(ctx, q)
	return p, normalizeRepositoryError(err)
}

type NormalizedReasoningSources struct {
	SignalIDs    []string                `json:"signal_ids"`
	EventIDs     []string                `json:"event_ids"`
	UpstreamRefs []NormalizedUpstreamRef `json:"upstream_refs"`
}
type NormalizedUpstreamRef struct {
	EntityID  string  `json:"entity_id"`
	LocalKey  string  `json:"local_key"`
	Mechanism *string `json:"mechanism,omitempty"`
	Condition *string `json:"condition,omitempty"`
}
type NormalizedSignal struct {
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

// EvidenceTag is a deterministic reading projection of existing semantic content.
type EvidenceTag struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

// reportPublication combines existing domain summaries for the Miniapp detail view.
// The cursor and request context keep historical lookups tied to the exact report.
func (u *UseCase) reportPublication(ctx context.Context, reportID string) (time.Time, error) {
	query := ListQuery{Limit: 100}
	seen := map[string]bool{}
	for {
		if err := ctx.Err(); err != nil {
			return time.Time{}, err
		}
		page, err := u.repository.ListReports(ctx, query)
		if err != nil {
			return time.Time{}, err
		}
		if len(page.Items) > query.Limit || validateSummaryOrder(page.Items) != nil {
			return time.Time{}, ErrDataUnavailable
		}
		for _, report := range page.Items {
			if report.ID == reportID {
				return report.PublishedAt, nil
			}
		}
		if page.NextCursor == nil {
			return time.Time{}, ErrDataUnavailable
		}
		cursor := *page.NextCursor
		if len(page.Items) == 0 || cursor == "" || len(cursor) > 2048 || seen[cursor] {
			return time.Time{}, ErrDataUnavailable
		}
		seen[cursor] = true
		query.Cursor = cursor
	}
}
