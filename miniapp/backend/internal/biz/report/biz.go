package report

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"sync"
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
	homeReadConcurrency = 4
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
	PublishedAt *time.Time
	Summary     string
	Keywords    []string
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
	Report     Summary
	Cards      []Card
	NextCursor *string
}
type HomeSelection struct{ Mode, Date, Timezone string }
type HomeCollection struct {
	Selection HomeSelection
	Reports   []Home
}

type Repository interface {
	ListReports(context.Context, ListQuery) (Page, error)
	GetHome(context.Context, string) (HomeSnapshot, error)
	ListIndustryChains(context.Context, ChainListQuery) (IndustryChainPage, error)
	GetLayer(context.Context, string, string) (LayerDetail, error)
	GetIndustryChain(context.Context, string, string) (IndustryChainDetail, error)
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
	summaries, err := u.listAll(ctx, ListQuery{PublishedFrom: &from, PublishedTo: &to, Limit: listPageSize})
	if err != nil {
		return HomeCollection{}, normalizeRepositoryError(err)
	}
	if len(summaries) == 0 {
		page, listErr := u.repository.ListReports(ctx, ListQuery{Limit: 1})
		if listErr != nil {
			return HomeCollection{}, normalizeRepositoryError(listErr)
		}
		if len(page.Items) > 1 || (len(page.Items) == 0 && page.NextCursor != nil) {
			return HomeCollection{}, ErrDataUnavailable
		}
		summaries = page.Items
		if len(summaries) > 0 {
			selection.Mode = SelectionFallback
		}
	}
	if err := validateSummaryOrder(summaries); err != nil {
		return HomeCollection{}, err
	}
	if len(summaries) == 0 {
		return HomeCollection{Selection: selection, Reports: []Home{}}, nil
	}
	homes, err := u.readHomes(ctx, summaries)
	if err != nil {
		return HomeCollection{}, err
	}
	return HomeCollection{Selection: selection, Reports: homes}, nil
}

func (u *UseCase) readHomes(ctx context.Context, summaries []Summary) ([]Home, error) {
	homes := make([]Home, len(summaries))
	workersCount := homeReadConcurrency
	if len(summaries) < workersCount {
		workersCount = len(summaries)
	}
	readContext, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan int)
	var workers sync.WaitGroup
	var failOnce sync.Once
	var readErr error
	workers.Add(workersCount)
	for worker := 0; worker < workersCount; worker++ {
		go func() {
			defer workers.Done()
			for index := range jobs {
				home, err := u.readHome(readContext, summaries[index])
				if err != nil {
					failOnce.Do(func() { readErr = err; cancel() })
					continue
				}
				homes[index] = home
			}
		}()
	}
send:
	for index := range summaries {
		select {
		case jobs <- index:
		case <-readContext.Done():
			break send
		}
	}
	close(jobs)
	workers.Wait()
	if readErr != nil {
		return nil, readErr
	}
	if ctx.Err() != nil {
		return nil, ErrDataUnavailable
	}
	return homes, nil
}

func (u *UseCase) readHome(ctx context.Context, summary Summary) (Home, error) {
	snapshot, err := u.repository.GetHome(ctx, summary.ID)
	if err != nil || !sameSummary(snapshot.Report, summary) {
		return Home{}, ErrDataUnavailable
	}
	cards := make([]Card, 0, 2+chainPageSize)
	for _, layer := range []*LayerSnapshot{snapshot.Geopolitics, snapshot.Macroeconomics} {
		if layer == nil {
			continue
		}
		detail, detailErr := u.repository.GetLayer(ctx, summary.ID, layer.Key)
		if detailErr != nil || !sameSummary(detail.Report, summary) || !sameLayerSummary(detail.Layer, *layer) {
			return Home{}, ErrDataUnavailable
		}
		cards = append(cards, layerCard(detail.Layer))
	}
	chains, err := u.repository.ListIndustryChains(ctx, ChainListQuery{ReportID: summary.ID, Limit: chainPageSize})
	if err != nil {
		return Home{}, normalizeRepositoryError(err)
	}
	for _, chain := range chains.Items {
		cards = append(cards, chainCard(chain))
	}
	return Home{Report: summary, Cards: cards, NextCursor: chains.NextCursor}, nil
}

func (u *UseCase) IndustryChains(ctx context.Context, reportID string, limit int, cursor string) (CardPage, error) {
	if !validReportID(reportID) || limit < 0 || limit > 100 || len(cursor) > 2048 {
		return CardPage{}, ErrInvalidRequest
	}
	if u == nil || u.repository == nil {
		return CardPage{}, ErrDataUnavailable
	}
	if limit == 0 {
		limit = chainPageSize
	}
	page, err := u.repository.ListIndustryChains(ctx, ChainListQuery{ReportID: reportID, Limit: limit, Cursor: cursor})
	if err != nil {
		return CardPage{}, normalizeRepositoryError(err)
	}
	items := make([]Card, len(page.Items))
	for index, item := range page.Items {
		items[index] = chainCard(item)
	}
	return CardPage{Items: items, NextCursor: page.NextCursor}, nil
}

func (u *UseCase) Layer(ctx context.Context, reportID, layerKey string) (LayerDetail, error) {
	if !validReportID(reportID) || !validLayer(layerKey) {
		return LayerDetail{}, ErrInvalidRequest
	}
	if u == nil || u.repository == nil {
		return LayerDetail{}, ErrDataUnavailable
	}
	value, err := u.repository.GetLayer(ctx, reportID, layerKey)
	if err != nil {
		return LayerDetail{}, normalizeRepositoryError(err)
	}
	if value.Report.ID != reportID || value.Layer.Key != layerKey {
		return LayerDetail{}, ErrDataUnavailable
	}
	chains, err := u.listAllIndustryChains(ctx, reportID)
	if err != nil || len(chains) != value.Report.IndustryChainCount {
		return LayerDetail{}, ErrDataUnavailable
	}
	value.RelatedIndustryChains = make([]RelatedIndustryChain, len(chains))
	for index, chain := range chains {
		value.RelatedIndustryChains[index] = RelatedIndustryChain{LocalKey: chain.LocalKey, Name: chain.Name, Result: chain.Result}
	}
	return value, nil
}

func (u *UseCase) listAllIndustryChains(ctx context.Context, reportID string) ([]IndustryChainSummary, error) {
	query := ChainListQuery{ReportID: reportID, Limit: listPageSize}
	items := []IndustryChainSummary{}
	seenCursors := map[string]struct{}{}
	seenKeys := map[string]struct{}{}
	for pageIndex := 0; pageIndex < maxListPages; pageIndex++ {
		page, err := u.repository.ListIndustryChains(ctx, query)
		if err != nil {
			return nil, normalizeRepositoryError(err)
		}
		for _, item := range page.Items {
			if !validLocalKey(item.LocalKey) {
				return nil, ErrDataUnavailable
			}
			if _, duplicate := seenKeys[item.LocalKey]; duplicate {
				return nil, ErrDataUnavailable
			}
			seenKeys[item.LocalKey] = struct{}{}
			items = append(items, item)
		}
		if page.NextCursor == nil {
			return items, nil
		}
		next := strings.TrimSpace(*page.NextCursor)
		if next == "" || len(page.Items) == 0 || len(next) > 2048 {
			return nil, ErrDataUnavailable
		}
		if _, duplicate := seenCursors[next]; duplicate {
			return nil, ErrDataUnavailable
		}
		seenCursors[next] = struct{}{}
		query.Cursor = next
	}
	return nil, ErrDataUnavailable
}

func (u *UseCase) IndustryChain(ctx context.Context, reportID, chainKey string) (IndustryChainDetail, error) {
	if !validReportID(reportID) || !validLocalKey(chainKey) {
		return IndustryChainDetail{}, ErrInvalidRequest
	}
	if u == nil || u.repository == nil {
		return IndustryChainDetail{}, ErrDataUnavailable
	}
	value, err := u.repository.GetIndustryChain(ctx, reportID, chainKey)
	if err != nil {
		return IndustryChainDetail{}, normalizeRepositoryError(err)
	}
	if value.Report.ID != reportID || value.IndustryChain.LocalKey != chainKey {
		return IndustryChainDetail{}, ErrDataUnavailable
	}
	return value, nil
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

func (u *UseCase) listAll(ctx context.Context, first ListQuery) ([]Summary, error) {
	query := first
	items := []Summary{}
	seen := map[string]struct{}{}
	for pageIndex := 0; pageIndex < maxListPages; pageIndex++ {
		page, err := u.repository.ListReports(ctx, query)
		if err != nil {
			return nil, err
		}
		items = append(items, page.Items...)
		if page.NextCursor == nil {
			return items, nil
		}
		next := strings.TrimSpace(*page.NextCursor)
		if next == "" || len(page.Items) == 0 || len(next) > 2048 {
			return nil, ErrDataUnavailable
		}
		if _, duplicate := seen[next]; duplicate {
			return nil, ErrDataUnavailable
		}
		seen[next] = struct{}{}
		query.Cursor = next
	}
	return nil, ErrDataUnavailable
}

func layerCard(layer Layer) Card {
	impacts := make([]CardImpactItem, len(layer.Anchors))
	for index, anchor := range layer.Anchors {
		impacts[index] = CardImpactItem{
			Ref: Reference{Type: "anchor", LocalKey: anchor.LocalKey}, Name: anchor.Name, Result: anchor.Result,
			ConclusionBasis: anchor.ConclusionBasis, ValidationStatus: anchor.ValidationStatus,
			Confidence: anchor.Confidence, TimeWindow: anchor.TimeWindow, EvidenceScopeToken: anchor.EvidenceScopeToken,
		}
	}
	return Card{
		LocalKey: layer.Key, Kind: layer.Key, DetailRef: Reference{Type: "layer", LocalKey: layer.Key},
		Title: layer.Title, Conclusion: layer.Conclusion, Result: layer.Result, Confidence: layer.Confidence,
		TimeWindow: layer.TimeWindow, ImpactItems: impacts, EvidenceScopeToken: layer.EvidenceScopeToken,
	}
}

func chainCard(chain IndustryChainSummary) Card {
	impacts := make([]CardImpactItem, len(chain.ImpactItems))
	for index, item := range chain.ImpactItems {
		impacts[index] = CardImpactItem{
			Ref: Reference{Type: "industry_chain_node", LocalKey: item.LocalKey}, Name: item.Name, Result: item.Result,
			ConclusionBasis: item.ConclusionBasis, ValidationStatus: item.ValidationStatus,
			Confidence: item.Confidence, TimeWindow: item.TimeWindow, EvidenceScopeToken: item.EvidenceScopeToken,
		}
	}
	return Card{
		LocalKey: chain.LocalKey, Kind: "industry_chain", DetailRef: Reference{Type: "industry_chain", LocalKey: chain.LocalKey},
		Title: chain.Name, Conclusion: chain.Conclusion, Result: chain.Result,
		Confidence: chain.Confidence, TimeWindow: chain.TimeWindow, ImpactItems: impacts, EvidenceScopeToken: chain.EvidenceScopeToken,
	}
}

func sameLayerSummary(layer Layer, snapshot LayerSnapshot) bool {
	return layer.Key == snapshot.Key && layer.Title == snapshot.Title && layer.Conclusion == snapshot.Summary.Conclusion &&
		layer.Result == snapshot.Summary.Result && layer.TimeWindow == snapshot.Summary.TimeWindow
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
		if !validReportID(item.ID) || strings.TrimSpace(item.PublisherReportID) == "" || item.GeneratedAt.IsZero() || item.PublishedAt.IsZero() || item.IndustryChainCount < 1 {
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
func sameSummary(left, right Summary) bool {
	return left.ID == right.ID && left.PublisherReportID == right.PublisherReportID && left.GeneratedAt.Equal(right.GeneratedAt) &&
		left.PublishedAt.Equal(right.PublishedAt) && left.IndustryChainCount == right.IndustryChainCount
}
func normalizeRepositoryError(err error) error {
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
