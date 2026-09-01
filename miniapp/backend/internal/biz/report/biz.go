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

	ScopeReportCard         = "report_card"
	ScopeLayer              = "layer"
	ScopeAnchor             = "anchor"
	ScopeReasoningStep      = "reasoning_step"
	ScopeTransmissionPath   = "transmission_path"
	ScopeCandidateMechanism = "candidate_mechanism"
	ScopeIndustryChain      = "industry_chain"
	ScopeIndustryChainNode  = "industry_chain_node"

	SelectionToday          = "today"
	SelectionLatestFallback = "latest_fallback"

	listPageSize        = 100
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

	reportIDPattern = regexp.MustCompile(`^RPT[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	localKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)
)

type ListQuery struct {
	PublishedFrom *time.Time
	PublishedTo   *time.Time
	Limit         int
	Cursor        string
}

type Summary struct {
	ID             string
	SourceReportID string
	Title          string
	GeneratedAt    time.Time
	PublishedAt    time.Time
}

type Page struct {
	Items      []Summary
	NextCursor *string
}

type Result struct {
	Code  string
	Label string
}

type Nature struct {
	Code  string
	Label string
}

type Confidence struct {
	Label string
	Score *float64
}

// Reference identifies only a target inside one immutable Report snapshot.
type Reference struct {
	Type string
	Key  string
}

type EvidenceScope struct {
	Type string
	Key  string
}

type CardImpactItem struct {
	Ref         Reference
	Name        string
	Result      Result
	Confidence  Confidence
	TimeWindow  string
	HasEvidence bool
}

type Card struct {
	Key         string
	Kind        string
	Order       int
	DetailRef   Reference
	Title       string
	Subtitle    string
	Conclusion  string
	Result      Result
	Confidence  Confidence
	TimeWindow  string
	ImpactItems []CardImpactItem
	HasEvidence bool
}

type CompanyBoundary struct {
	Key          string
	DisplayOrder int
	Title        string
	Published    bool
	Boundary     string
}

type Home struct {
	Report  Summary
	Cards   []Card
	Company CompanyBoundary
}

type HomeSelection struct {
	Mode     string
	Date     string
	Timezone string
}

type HomeCollection struct {
	Selection HomeSelection
	Reports   []Home
}

type Anchor struct {
	Key          string
	DisplayOrder int
	Name         string
	CurrentState string
	Result       Result
	Nature       Nature
	Reasoning    string
	TimeWindow   string
	Confidence   Confidence
	Scope        EvidenceScope
	HasEvidence  bool
}

type ReasoningStep struct {
	Key          string
	DisplayOrder int
	Input        string
	Mechanism    string
	Output       string
	Type         string
	Confidence   Confidence
	Scope        EvidenceScope
	HasEvidence  bool
}

type TransmissionTarget struct {
	Ref    Reference
	Label  string
	Result Result
}

type TransmissionPath struct {
	Key              string
	DisplayOrder     int
	SourceConclusion string
	TargetRefs       []TransmissionTarget
	Logic            string
	RelationNature   string
	EvidenceRole     string
	Confidence       Confidence
	Status           string
	Scope            EvidenceScope
	HasEvidence      bool
}

type CandidateMechanism struct {
	Key          string
	DisplayOrder int
	Mechanism    string
	EvidenceGap  *string
	Confidence   Confidence
	Scope        EvidenceScope
	HasEvidence  bool
}

type Checkpoint struct {
	Key          string
	DisplayOrder int
	Summary      string
}

type DownwardTransmission struct {
	Summary             string
	PublishedPaths      []TransmissionPath
	CandidateMechanisms []CandidateMechanism
	BoundaryNotes       []string
}

type LayerUncertainty struct {
	Counterevidence   *string
	EvidenceGap       *string
	Boundary          *string
	ReversalCondition *string
	Checkpoints       []Checkpoint
}

type Layer struct {
	Key                  string
	DisplayOrder         int
	Title                string
	Conclusion           string
	Result               Result
	Confidence           Confidence
	TimeWindow           string
	Anchors              []Anchor
	ReasoningSteps       []ReasoningStep
	RelatedAnchorKeys    []string
	RelatedChainKeys     []string
	DownwardTransmission DownwardTransmission
	Uncertainty          LayerUncertainty
	Scope                EvidenceScope
	HasEvidence          bool
}

type IndustryChainSummary struct {
	Key          string
	DisplayOrder int
	Name         string
	Conclusion   string
	Status       string
	Result       Result
	Confidence   Confidence
	TimeWindow   string
	Scope        EvidenceScope
	HasEvidence  bool
}

type LayerDetail struct {
	Report                Summary
	Layer                 Layer
	RelatedIndustryChains []IndustryChainSummary
}

type IndustryChainNode struct {
	Key          string
	DisplayOrder int
	Name         string
	Impact       string
	Result       Result
	Nature       Nature
	Reasoning    string
	TimeWindow   string
	Confidence   Confidence
	Scope        EvidenceScope
	HasEvidence  bool
}

type IndustryChainEdge struct {
	Key           string
	DisplayOrder  int
	FromNodeKey   string
	ToNodeKey     string
	RelationLabel string
}

type ChainUncertainty struct {
	CounterevidenceAndGap *string
	StopCondition         *string
	Checkpoints           []Checkpoint
}

type IndustryChain struct {
	Key                       string
	ClaimKey                  string
	DisplayOrder              int
	Name                      string
	Conclusion                string
	Status                    string
	Result                    Result
	Confidence                Confidence
	TimeWindow                string
	PathSummary               *string
	AcceptedHypothesisSummary *string
	Nodes                     []IndustryChainNode
	Edges                     []IndustryChainEdge
	Uncertainty               ChainUncertainty
	Scope                     EvidenceScope
	HasEvidence               bool
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
	ReportID string
	Scope    EvidenceScope
	Items    []EvidenceItem
}

type Repository interface {
	ListReports(context.Context, ListQuery) (Page, error)
	GetHome(context.Context, string) (Home, error)
	GetLayer(context.Context, string, string) (LayerDetail, error)
	GetIndustryChain(context.Context, string, string) (IndustryChainDetail, error)
	ListEvidences(context.Context, string, EvidenceScope) (EvidenceCollection, error)
}

type UseCase struct {
	repository Repository
	now        func() time.Time
}

func NewUseCase(repository Repository) *UseCase {
	return NewUseCaseWithClock(repository, time.Now)
}

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
			selection.Mode = SelectionLatestFallback
		}
	}
	if err := validateSummaryOrder(summaries); err != nil {
		return HomeCollection{}, ErrDataUnavailable
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
	workerCount := homeReadConcurrency
	if len(summaries) < workerCount {
		workerCount = len(summaries)
	}
	readContext, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan int)
	var workers sync.WaitGroup
	var failOnce sync.Once
	var readErr error
	workers.Add(workerCount)
	for worker := 0; worker < workerCount; worker++ {
		go func() {
			defer workers.Done()
			for index := range jobs {
				home, err := u.repository.GetHome(readContext, summaries[index].ID)
				if err != nil || !sameSummary(home.Report, summaries[index]) {
					failOnce.Do(func() {
						// A list/detail mismatch is a downstream snapshot failure for
						// the whole homepage, never a partial per-Report success.
						readErr = ErrDataUnavailable
						cancel()
					})
					continue
				}
				homes[index] = home
			}
		}()
	}
sendJobs:
	for index := range summaries {
		select {
		case jobs <- index:
		case <-readContext.Done():
			break sendJobs
		}
	}
	close(jobs)
	workers.Wait()
	if readErr != nil || ctx.Err() != nil {
		return nil, ErrDataUnavailable
	}
	return homes, nil
}

func (u *UseCase) listAll(ctx context.Context, first ListQuery) ([]Summary, error) {
	query := first
	items := make([]Summary, 0)
	seenCursors := make(map[string]struct{})
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
		if _, duplicate := seenCursors[next]; duplicate {
			return nil, ErrDataUnavailable
		}
		seenCursors[next] = struct{}{}
		query.Cursor = next
	}
	return nil, ErrDataUnavailable
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
	return value, nil
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
	if value.Report.ID != reportID || value.IndustryChain.Key != chainKey {
		return IndustryChainDetail{}, ErrDataUnavailable
	}
	return value, nil
}

func (u *UseCase) Evidences(ctx context.Context, reportID string, scope EvidenceScope) (EvidenceCollection, error) {
	if !validReportID(reportID) || !validEvidenceScope(scope) {
		return EvidenceCollection{}, ErrInvalidRequest
	}
	if u == nil || u.repository == nil {
		return EvidenceCollection{}, ErrDataUnavailable
	}
	value, err := u.repository.ListEvidences(ctx, reportID, scope)
	if err != nil {
		return EvidenceCollection{}, normalizeRepositoryError(err)
	}
	if value.ReportID != reportID || value.Scope != scope {
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

func validLayer(value string) bool {
	return value == LayerGeopolitics || value == LayerMacroeconomics
}

func validLocalKey(value string) bool {
	return value == strings.TrimSpace(value) && localKeyPattern.MatchString(value)
}

func ValidReference(ref Reference) bool {
	switch ref.Type {
	case ScopeLayer:
		return validLayer(ref.Key)
	case ScopeAnchor, ScopeIndustryChain, ScopeIndustryChainNode:
		return validLocalKey(ref.Key)
	default:
		return false
	}
}

func validEvidenceScope(scope EvidenceScope) bool {
	switch scope.Type {
	case ScopeLayer:
		return validLayer(scope.Key)
	case ScopeReportCard, ScopeAnchor, ScopeReasoningStep, ScopeTransmissionPath,
		ScopeCandidateMechanism, ScopeIndustryChain, ScopeIndustryChainNode:
		return validLocalKey(scope.Key)
	default:
		return false
	}
}

func validateSummaryOrder(items []Summary) error {
	seen := make(map[string]struct{}, len(items))
	for index, item := range items {
		if !validReportID(item.ID) || item.SourceReportID == "" || item.Title == "" ||
			item.GeneratedAt.IsZero() || item.PublishedAt.IsZero() || item.GeneratedAt.Location() != time.UTC ||
			item.PublishedAt.Location() != time.UTC {
			return ErrDataUnavailable
		}
		if _, duplicate := seen[item.ID]; duplicate {
			return ErrDataUnavailable
		}
		seen[item.ID] = struct{}{}
		if index == 0 {
			continue
		}
		previous := items[index-1]
		if item.PublishedAt.After(previous.PublishedAt) ||
			(item.PublishedAt.Equal(previous.PublishedAt) && item.ID < previous.ID) {
			return ErrDataUnavailable
		}
	}
	return nil
}

func sameSummary(left, right Summary) bool {
	return left.ID == right.ID && left.SourceReportID == right.SourceReportID && left.Title == right.Title &&
		left.GeneratedAt.Equal(right.GeneratedAt) && left.PublishedAt.Equal(right.PublishedAt)
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
