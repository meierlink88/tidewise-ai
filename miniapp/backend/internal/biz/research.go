package biz

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	DefaultResearchWindowHours  = 24
	MinResearchWindowHours      = 1
	MaxResearchWindowHours      = 168
	DefaultResearchLimit        = 20
	DefaultHistoryResearchLimit = 5
	MaxResearchLimit            = 50
	ResearchPeriodToday         = "today"
	ResearchPeriodHistory       = "history"
)

var (
	ErrInvalidResearchRequest = errors.New("invalid research request")
	ErrResearchNotFound       = errors.New("research result not found")
	ErrResearchDataService    = errors.New("research data service failure")
	researchUUIDPattern       = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
)

type ResearchListRequest struct {
	Period      string
	WindowHours int
	Limit       int
	Cursor      string
}
type ResearchDetailRequest struct{ WindowHours int }

type ResearchThemeListResponse struct {
	WindowStart, WindowEnd, AsOf string
	ThemeCount, EventCount       int
	Items                        []ResearchThemeItem
	NextCursor                   *string
}

type ResearchThemeItem struct {
	ID, AnalysisBatchID, Title, OneLineConclusion          string
	ConclusionDirection, ImpactStrength, TransmissionStage string
	InvestmentGuidanceAction, InvestmentGuidanceSummary    string
	TimeHorizonCategory                                    string
	AttentionLevel, ConclusionStatus                       *string
	TimeHorizonSummary, TransmissionSummary                *string
	CheckpointSummary, RiskSummary                         *string
	AnalysisAsOf, WindowStart, WindowEnd, PublishedAt      string
	Impacts                                                []ResearchThemeImpactDTO
	EvidenceEventCount, ReasoningTreeCount                 int
}
type ResearchThemeImpactDTO struct {
	NodeKey, DisplayName                                   string
	ChainNodeEntityID, Name, RelationRole, ImpactDirection string
	ImpactSummary                                          *string
	DisplayOrder                                           int
}
type ResearchThemeDetailResponse struct {
	ResearchThemeItem
	Events []ResearchEventDTO
}
type ResearchEventDTO struct {
	EventID, Title, Summary, EvidenceRole string
	EvidenceIDs                           []string
	EventTime                             *string
	SupportedClaim                        *string
	DisplayOrder                          int
}

type ResearchService struct {
	repo ResearchRepo
	now  func() time.Time
}

func NewResearchService(repo ResearchRepo) *ResearchService {
	return NewResearchServiceWithClock(repo, time.Now)
}

func NewResearchServiceWithClock(repo ResearchRepo, now func() time.Time) *ResearchService {
	if now == nil {
		now = time.Now
	}
	return &ResearchService{repo: repo, now: now}
}

func (s *ResearchService) ListThemes(ctx context.Context, request ResearchListRequest) (ResearchThemeListResponse, error) {
	window, limit, err := normalizeResearchListRequest(request)
	if err != nil {
		return ResearchThemeListResponse{}, err
	}
	if s == nil || s.repo == nil {
		return ResearchThemeListResponse{}, ErrResearchDataService
	}
	query := ResearchListQuery{WindowHours: window, Limit: limit, Cursor: request.Cursor}
	if request.Period != "" {
		bounds, dataCursor, err := s.resolvePeriodQuery(request.Period, request.Cursor)
		if err != nil {
			return ResearchThemeListResponse{}, err
		}
		query.WindowHours = 0
		query.PublishedFrom = &bounds.from
		query.PublishedTo = &bounds.to
		query.Cursor = dataCursor
	}
	page, err := s.repo.ListResearchThemes(ctx, query)
	if err != nil {
		return ResearchThemeListResponse{}, normalizeResearchRepoError(err)
	}
	items := make([]ResearchThemeItem, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, themeItemDTO(item))
	}
	response := ResearchThemeListResponse{
		WindowStart: formatTime(page.WindowStart), WindowEnd: formatTime(page.WindowEnd), AsOf: formatTime(page.AsOf),
		ThemeCount: page.ThemeCount, EventCount: page.EventCount, Items: items, NextCursor: page.NextCursor,
	}
	if request.Period != "" && page.NextCursor != nil {
		wrapped, err := encodePeriodCursor(periodCursor{
			Version: 1, Period: request.Period, PublishedFrom: query.PublishedFrom.UTC(),
			PublishedTo: query.PublishedTo.UTC(), DataCursor: *page.NextCursor,
		})
		if err != nil {
			return ResearchThemeListResponse{}, ErrResearchDataService
		}
		response.NextCursor = &wrapped
	}
	return response, nil
}

func (s *ResearchService) GetTheme(ctx context.Context, id string, request ResearchDetailRequest) (ResearchThemeDetailResponse, error) {
	window, err := normalizeResearchDetailRequest(request)
	if err != nil {
		return ResearchThemeDetailResponse{}, err
	}
	id = strings.TrimSpace(id)
	if !researchUUIDPattern.MatchString(id) {
		return ResearchThemeDetailResponse{}, fmt.Errorf("%w: theme id must be a UUID", ErrInvalidResearchRequest)
	}
	if s == nil || s.repo == nil {
		return ResearchThemeDetailResponse{}, ErrResearchDataService
	}
	detail, err := s.repo.GetResearchTheme(ctx, id, ResearchDetailQuery{WindowHours: window})
	if err != nil {
		return ResearchThemeDetailResponse{}, normalizeResearchRepoError(err)
	}
	return ResearchThemeDetailResponse{ResearchThemeItem: themeItemDTO(detail.Theme), Events: eventDTOs(detail.Events)}, nil
}

func normalizeResearchListRequest(request ResearchListRequest) (int, int, error) {
	if request.Period != "" && request.Period != ResearchPeriodToday && request.Period != ResearchPeriodHistory {
		return 0, 0, fmt.Errorf("%w: period must be today or history", ErrInvalidResearchRequest)
	}
	if request.Period != "" && request.WindowHours != 0 {
		return 0, 0, fmt.Errorf("%w: period cannot be combined with window_hours", ErrInvalidResearchRequest)
	}
	window, limit := request.WindowHours, request.Limit
	if window == 0 && request.Period == "" {
		window = DefaultResearchWindowHours
	}
	if limit == 0 {
		if request.Period == ResearchPeriodHistory {
			limit = DefaultHistoryResearchLimit
		} else {
			limit = DefaultResearchLimit
		}
	}
	if request.Period == "" && (window < MinResearchWindowHours || window > MaxResearchWindowHours) {
		return 0, 0, fmt.Errorf("%w: window_hours out of range", ErrInvalidResearchRequest)
	}
	if limit < 1 || limit > MaxResearchLimit {
		return 0, 0, fmt.Errorf("%w: limit out of range", ErrInvalidResearchRequest)
	}
	return window, limit, nil
}

type publicationBounds struct{ from, to time.Time }

type periodCursor struct {
	Version       int       `json:"v"`
	Period        string    `json:"period"`
	PublishedFrom time.Time `json:"published_from"`
	PublishedTo   time.Time `json:"published_to"`
	DataCursor    string    `json:"data_cursor"`
}

func (s *ResearchService) resolvePeriodQuery(period, encoded string) (publicationBounds, string, error) {
	if strings.TrimSpace(encoded) != "" {
		cursor, err := decodePeriodCursor(encoded)
		if err != nil || cursor.Version != 1 || cursor.Period != period || cursor.DataCursor == "" || !cursor.PublishedFrom.Before(cursor.PublishedTo) {
			return publicationBounds{}, "", fmt.Errorf("%w: invalid cursor", ErrInvalidResearchRequest)
		}
		return publicationBounds{from: cursor.PublishedFrom.UTC(), to: cursor.PublishedTo.UTC()}, cursor.DataCursor, nil
	}
	shanghai := time.FixedZone("Asia/Shanghai", 8*60*60)
	localNow := s.now().In(shanghai)
	today := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, shanghai)
	if period == ResearchPeriodToday {
		return publicationBounds{from: today.UTC(), to: today.AddDate(0, 0, 1).UTC()}, "", nil
	}
	return publicationBounds{from: today.AddDate(0, 0, -30).UTC(), to: today.UTC()}, "", nil
}

func encodePeriodCursor(cursor periodCursor) (string, error) {
	payload, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodePeriodCursor(value string) (periodCursor, error) {
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return periodCursor{}, err
	}
	var cursor periodCursor
	err = json.Unmarshal(payload, &cursor)
	return cursor, err
}
func normalizeResearchDetailRequest(request ResearchDetailRequest) (int, error) {
	window := request.WindowHours
	if window == 0 {
		window = DefaultResearchWindowHours
	}
	if window < MinResearchWindowHours || window > MaxResearchWindowHours {
		return 0, fmt.Errorf("%w: window_hours out of range", ErrInvalidResearchRequest)
	}
	return window, nil
}

func themeItemDTO(value ResearchTheme) ResearchThemeItem {
	impacts := make([]ResearchThemeImpactDTO, 0, len(value.Impacts))
	for _, impact := range value.Impacts {
		impacts = append(impacts, ResearchThemeImpactDTO{
			NodeKey: impact.NodeKey, DisplayName: impact.DisplayName,
			ChainNodeEntityID: impact.ChainNodeEntityID, Name: impact.Name, RelationRole: impact.RelationRole,
			ImpactDirection: impact.ImpactDirection, ImpactSummary: impact.ImpactSummary, DisplayOrder: impact.DisplayOrder,
		})
	}
	return ResearchThemeItem{
		ID: value.ID, AnalysisBatchID: value.AnalysisBatchID, Title: value.Title,
		OneLineConclusion: value.OneLineConclusion, ConclusionDirection: value.ConclusionDirection,
		ImpactStrength: value.ImpactStrength, AttentionLevel: value.AttentionLevel,
		ConclusionStatus: value.ConclusionStatus, TransmissionStage: value.TransmissionStage,
		InvestmentGuidanceAction:  value.InvestmentGuidanceAction,
		InvestmentGuidanceSummary: value.InvestmentGuidanceSummary,
		TimeHorizonCategory:       value.TimeHorizonCategory, TimeHorizonSummary: value.TimeHorizonSummary,
		TransmissionSummary: value.TransmissionSummary, CheckpointSummary: value.CheckpointSummary,
		RiskSummary: value.RiskSummary, AnalysisAsOf: formatTime(value.AnalysisAsOf),
		WindowStart: formatTime(value.WindowStart), WindowEnd: formatTime(value.WindowEnd),
		PublishedAt: formatTime(value.PublishedAt), Impacts: impacts,
		EvidenceEventCount: value.EvidenceEventCount, ReasoningTreeCount: value.ReasoningTreeCount,
	}
}
func eventDTOs(values []ResearchEvent) []ResearchEventDTO {
	result := make([]ResearchEventDTO, 0, len(values))
	for _, value := range values {
		var eventTime *string
		if value.EventTime != nil {
			formatted := formatTime(*value.EventTime)
			eventTime = &formatted
		}
		result = append(result, ResearchEventDTO{
			EventID: value.EventID, Title: value.Title, Summary: value.Summary, EventTime: eventTime,
			EvidenceIDs:  append([]string(nil), value.EvidenceIDs...),
			EvidenceRole: value.EvidenceRole, SupportedClaim: value.SupportedClaim, DisplayOrder: value.DisplayOrder,
		})
	}
	return result
}
func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339) }
func normalizeResearchRepoError(err error) error {
	switch {
	case errors.Is(err, ErrInvalidResearchRequest):
		return ErrInvalidResearchRequest
	case errors.Is(err, ErrResearchNotFound):
		return ErrResearchNotFound
	default:
		return ErrResearchDataService
	}
}
