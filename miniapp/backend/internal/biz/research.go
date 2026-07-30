package biz

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	DefaultResearchWindowHours = 24
	MinResearchWindowHours     = 1
	MaxResearchWindowHours     = 168
	DefaultResearchLimit       = 20
	MaxResearchLimit           = 50
)

var (
	ErrInvalidResearchRequest = errors.New("invalid research request")
	ErrResearchNotFound       = errors.New("research result not found")
	ErrResearchDataService    = errors.New("research data service failure")
	researchUUIDPattern       = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
)

type ResearchListRequest struct {
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
	EventTime                             *string
	SupportedClaim                        *string
	DisplayOrder                          int
}

type ResearchService struct{ repo ResearchRepo }

func NewResearchService(repo ResearchRepo) *ResearchService { return &ResearchService{repo: repo} }

func (s *ResearchService) ListThemes(ctx context.Context, request ResearchListRequest) (ResearchThemeListResponse, error) {
	window, limit, err := normalizeResearchListRequest(request)
	if err != nil {
		return ResearchThemeListResponse{}, err
	}
	if s == nil || s.repo == nil {
		return ResearchThemeListResponse{}, ErrResearchDataService
	}
	page, err := s.repo.ListResearchThemes(ctx, ResearchListQuery{WindowHours: window, Limit: limit, Cursor: request.Cursor})
	if err != nil {
		return ResearchThemeListResponse{}, normalizeResearchRepoError(err)
	}
	items := make([]ResearchThemeItem, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, themeItemDTO(item))
	}
	return ResearchThemeListResponse{
		WindowStart: formatTime(page.WindowStart), WindowEnd: formatTime(page.WindowEnd), AsOf: formatTime(page.AsOf),
		ThemeCount: page.ThemeCount, EventCount: page.EventCount, Items: items, NextCursor: page.NextCursor,
	}, nil
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
	window, limit := request.WindowHours, request.Limit
	if window == 0 {
		window = DefaultResearchWindowHours
	}
	if limit == 0 {
		limit = DefaultResearchLimit
	}
	if window < MinResearchWindowHours || window > MaxResearchWindowHours {
		return 0, 0, fmt.Errorf("%w: window_hours out of range", ErrInvalidResearchRequest)
	}
	if limit < 1 || limit > MaxResearchLimit {
		return 0, 0, fmt.Errorf("%w: limit out of range", ErrInvalidResearchRequest)
	}
	return window, limit, nil
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
