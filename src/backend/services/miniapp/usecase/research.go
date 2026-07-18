package usecase

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/dataclient"
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
)

var researchUUIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

type ResearchListRequest struct {
	WindowHours int
	Limit       int
	Cursor      string
}

type ResearchDetailRequest struct {
	WindowHours int
}

type ResearchThemeListResponse struct {
	WindowStart string              `json:"window_start"`
	WindowEnd   string              `json:"window_end"`
	AsOf        string              `json:"as_of"`
	ThemeCount  int                 `json:"theme_count"`
	EventCount  int                 `json:"event_count"`
	Items       []ResearchThemeItem `json:"items"`
	NextCursor  *string             `json:"next_cursor"`
}

type ResearchThemeItem struct {
	ID                      string                 `json:"id"`
	Name                    string                 `json:"name"`
	OneLineConclusion       string                 `json:"one_line_conclusion"`
	ImpactLevel             string                 `json:"impact_level"`
	TransmissionPath        string                 `json:"transmission_path"`
	TradingDirection        string                 `json:"trading_direction"`
	TransmissionStage       string                 `json:"transmission_stage"`
	NextCheckpoint          string                 `json:"next_checkpoint"`
	IndexImpactSummary      string                 `json:"index_impact_summary,omitempty"`
	PublishedAt             string                 `json:"published_at"`
	AffectedChainNodes      []ResearchChainNodeDTO `json:"affected_chain_nodes"`
	RelatedIndices          []ResearchIndexDTO     `json:"related_indices"`
	SupportingEventCount    int                    `json:"supporting_event_count"`
	ContradictingEventCount int                    `json:"contradicting_event_count"`
}

type ResearchThemeDetailResponse struct {
	ResearchThemeItem
	Events []ResearchEventDTO `json:"events"`
}

type ResearchAnchorListResponse struct {
	WindowStart string               `json:"window_start"`
	WindowEnd   string               `json:"window_end"`
	AsOf        string               `json:"as_of"`
	AnchorCount int                  `json:"anchor_count"`
	EventCount  int                  `json:"event_count"`
	Items       []ResearchAnchorItem `json:"items"`
	NextCursor  *string              `json:"next_cursor"`
}

type ResearchAnchorItem struct {
	ID                string                       `json:"id"`
	AnchorType        string                       `json:"anchor_type"`
	Name              string                       `json:"name"`
	OneLineConclusion string                       `json:"one_line_conclusion"`
	Importance        string                       `json:"importance"`
	TransmissionPath  string                       `json:"transmission_path"`
	TradingDirection  string                       `json:"trading_direction"`
	PublishedAt       string                       `json:"published_at"`
	RelatedChainNodes []ResearchAnchorChainNodeDTO `json:"related_chain_nodes"`
	RelatedIndices    []ResearchIndexDTO           `json:"related_indices"`
	RelatedEventCount int                          `json:"related_event_count"`
}

type ResearchAnchorDetailResponse struct {
	ResearchAnchorItem
	Events []ResearchEventDTO `json:"events"`
}

type ResearchChainNodeDTO struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	RelationRole string `json:"relation_role"`
	Summary      string `json:"impact_summary"`
}

type ResearchAnchorChainNodeDTO struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	RelationRole    string `json:"relation_role"`
	RelationSummary string `json:"relation_summary"`
}

type ResearchIndexDTO struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	ImpactDirection string `json:"impact_direction"`
	Summary         string `json:"impact_summary"`
}

type ResearchEventDTO struct {
	EventID        string  `json:"event_id"`
	Title          string  `json:"title"`
	Summary        string  `json:"summary"`
	EventTime      *string `json:"event_time,omitempty"`
	EvidenceRole   string  `json:"evidence_role"`
	SupportedClaim string  `json:"supported_claim"`
}

type ResearchService struct {
	client dataclient.DataServiceClient
}

func NewResearchService(client dataclient.DataServiceClient) *ResearchService {
	return &ResearchService{client: client}
}

func (s *ResearchService) ListThemes(ctx context.Context, request ResearchListRequest) (ResearchThemeListResponse, error) {
	windowHours, limit, err := normalizeResearchListRequest(request)
	if err != nil {
		return ResearchThemeListResponse{}, err
	}
	if s == nil || s.client == nil {
		return ResearchThemeListResponse{}, ErrResearchDataService
	}
	page, err := s.client.ListResearchThemes(ctx, dataclient.ResearchListQuery{WindowHours: windowHours, Limit: limit, Cursor: request.Cursor})
	if err != nil {
		return ResearchThemeListResponse{}, mapDataServiceError(err)
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
	windowHours, err := normalizeResearchDetailRequest(request)
	if err != nil {
		return ResearchThemeDetailResponse{}, err
	}
	id = strings.TrimSpace(id)
	if !researchUUIDPattern.MatchString(id) {
		return ResearchThemeDetailResponse{}, fmt.Errorf("%w: theme id must be a UUID", ErrInvalidResearchRequest)
	}
	if s == nil || s.client == nil {
		return ResearchThemeDetailResponse{}, ErrResearchDataService
	}
	detail, err := s.client.GetResearchTheme(ctx, id, dataclient.ResearchDetailQuery{WindowHours: windowHours})
	if err != nil {
		return ResearchThemeDetailResponse{}, mapDataServiceError(err)
	}
	return ResearchThemeDetailResponse{ResearchThemeItem: themeItemDTO(detail.Theme), Events: eventDTOs(detail.Events)}, nil
}

func (s *ResearchService) ListAnchors(ctx context.Context, request ResearchListRequest) (ResearchAnchorListResponse, error) {
	windowHours, limit, err := normalizeResearchListRequest(request)
	if err != nil {
		return ResearchAnchorListResponse{}, err
	}
	if s == nil || s.client == nil {
		return ResearchAnchorListResponse{}, ErrResearchDataService
	}
	page, err := s.client.ListResearchAnchors(ctx, dataclient.ResearchListQuery{WindowHours: windowHours, Limit: limit, Cursor: request.Cursor})
	if err != nil {
		return ResearchAnchorListResponse{}, mapDataServiceError(err)
	}
	items := make([]ResearchAnchorItem, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, anchorItemDTO(item))
	}
	return ResearchAnchorListResponse{
		WindowStart: formatTime(page.WindowStart), WindowEnd: formatTime(page.WindowEnd), AsOf: formatTime(page.AsOf),
		AnchorCount: page.AnchorCount, EventCount: page.EventCount, Items: items, NextCursor: page.NextCursor,
	}, nil
}

func (s *ResearchService) GetAnchor(ctx context.Context, id string, request ResearchDetailRequest) (ResearchAnchorDetailResponse, error) {
	windowHours, err := normalizeResearchDetailRequest(request)
	if err != nil {
		return ResearchAnchorDetailResponse{}, err
	}
	id = strings.TrimSpace(id)
	if !researchUUIDPattern.MatchString(id) {
		return ResearchAnchorDetailResponse{}, fmt.Errorf("%w: anchor id must be a UUID", ErrInvalidResearchRequest)
	}
	if s == nil || s.client == nil {
		return ResearchAnchorDetailResponse{}, ErrResearchDataService
	}
	detail, err := s.client.GetResearchAnchor(ctx, id, dataclient.ResearchDetailQuery{WindowHours: windowHours})
	if err != nil {
		return ResearchAnchorDetailResponse{}, mapDataServiceError(err)
	}
	return ResearchAnchorDetailResponse{ResearchAnchorItem: anchorItemDTO(detail.Anchor), Events: eventDTOs(detail.Events)}, nil
}

func normalizeResearchListRequest(request ResearchListRequest) (int, int, error) {
	windowHours := request.WindowHours
	if windowHours == 0 {
		windowHours = DefaultResearchWindowHours
	}
	if windowHours < MinResearchWindowHours || windowHours > MaxResearchWindowHours {
		return 0, 0, fmt.Errorf("%w: window_hours must be between %d and %d", ErrInvalidResearchRequest, MinResearchWindowHours, MaxResearchWindowHours)
	}
	limit := request.Limit
	if limit == 0 {
		limit = DefaultResearchLimit
	}
	if limit < 1 || limit > MaxResearchLimit {
		return 0, 0, fmt.Errorf("%w: limit must be between 1 and %d", ErrInvalidResearchRequest, MaxResearchLimit)
	}
	return windowHours, limit, nil
}

func normalizeResearchDetailRequest(request ResearchDetailRequest) (int, error) {
	windowHours := request.WindowHours
	if windowHours == 0 {
		windowHours = DefaultResearchWindowHours
	}
	if windowHours < MinResearchWindowHours || windowHours > MaxResearchWindowHours {
		return 0, fmt.Errorf("%w: window_hours must be between %d and %d", ErrInvalidResearchRequest, MinResearchWindowHours, MaxResearchWindowHours)
	}
	return windowHours, nil
}

func themeItemDTO(item dataclient.ResearchTheme) ResearchThemeItem {
	return ResearchThemeItem{
		ID: item.ID, Name: item.Name, OneLineConclusion: item.OneLineConclusion, ImpactLevel: string(item.ImpactLevel),
		TransmissionPath: item.TransmissionPath, TradingDirection: item.TradingDirection, TransmissionStage: string(item.TransmissionStage),
		NextCheckpoint: item.NextCheckpoint, IndexImpactSummary: item.IndexImpactSummary, PublishedAt: formatTime(item.PublishedAt),
		AffectedChainNodes: themeChainNodeDTOs(item.AffectedChainNodes), RelatedIndices: indexDTOs(item.RelatedIndices),
		SupportingEventCount: item.SupportingEventCount, ContradictingEventCount: item.ContradictingEventCount,
	}
}

func anchorItemDTO(item dataclient.ResearchAnchor) ResearchAnchorItem {
	return ResearchAnchorItem{
		ID: item.ID, AnchorType: string(item.AnchorType), Name: item.Name, OneLineConclusion: item.OneLineConclusion,
		Importance: string(item.Importance), TransmissionPath: item.TransmissionPath, TradingDirection: item.TradingDirection,
		PublishedAt: formatTime(item.PublishedAt), RelatedChainNodes: anchorChainNodeDTOs(item.RelatedChainNodes),
		RelatedIndices: indexDTOs(item.RelatedIndices), RelatedEventCount: item.RelatedEventCount,
	}
}

func themeChainNodeDTOs(values []dataclient.ResearchThemeChainNode) []ResearchChainNodeDTO {
	result := make([]ResearchChainNodeDTO, 0, len(values))
	for _, value := range values {
		result = append(result, ResearchChainNodeDTO{ID: value.ID, Name: value.Name, RelationRole: value.RelationRole, Summary: value.ImpactSummary})
	}
	return result
}

func anchorChainNodeDTOs(values []dataclient.ResearchAnchorChainNode) []ResearchAnchorChainNodeDTO {
	result := make([]ResearchAnchorChainNodeDTO, 0, len(values))
	for _, value := range values {
		result = append(result, ResearchAnchorChainNodeDTO{ID: value.ID, Name: value.Name, RelationRole: value.RelationRole, RelationSummary: value.RelationSummary})
	}
	return result
}

func indexDTOs(values []dataclient.ResearchIndex) []ResearchIndexDTO {
	result := make([]ResearchIndexDTO, 0, len(values))
	for _, value := range values {
		result = append(result, ResearchIndexDTO{ID: value.ID, Name: value.Name, ImpactDirection: string(value.ImpactDirection), Summary: value.ImpactSummary})
	}
	return result
}

func eventDTOs(values []dataclient.ResearchEvent) []ResearchEventDTO {
	result := make([]ResearchEventDTO, 0, len(values))
	for _, value := range values {
		var eventTime *string
		if value.EventTime != nil {
			formatted := formatTime(*value.EventTime)
			eventTime = &formatted
		}
		result = append(result, ResearchEventDTO{
			EventID: value.EventID, Title: value.Title, Summary: value.Summary, EventTime: eventTime,
			EvidenceRole: string(value.EvidenceRole), SupportedClaim: value.SupportedClaim,
		})
	}
	return result
}

func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339) }

func mapDataServiceError(err error) error {
	var clientErr *dataclient.Error
	if errors.As(err, &clientErr) {
		switch clientErr.StatusCode {
		case http.StatusBadRequest:
			return ErrInvalidResearchRequest
		case http.StatusNotFound:
			return ErrResearchNotFound
		}
	}
	return ErrResearchDataService
}
