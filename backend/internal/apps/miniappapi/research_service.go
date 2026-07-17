package miniappapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
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
	ErrResearchRepository     = errors.New("research repository failure")
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
	HasMoreDetail           bool                   `json:"has_more_detail"`
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
	repository repositories.ResearchReadRepository
	now        func() time.Time
}

func NewResearchService(repository repositories.ResearchReadRepository, now func() time.Time) *ResearchService {
	if now == nil {
		now = time.Now
	}
	return &ResearchService{repository: repository, now: now}
}

func (s *ResearchService) ListThemes(ctx context.Context, request ResearchListRequest) (ResearchThemeListResponse, error) {
	windowHours, limit, err := normalizeResearchListRequest(request)
	if err != nil {
		return ResearchThemeListResponse{}, err
	}
	asOf, start, cursor, err := s.prepareCursor("themes", windowHours, request.Cursor)
	if err != nil {
		return ResearchThemeListResponse{}, err
	}
	page, err := s.repository.ListResearchThemes(ctx, repositories.ResearchThemeListFilter{WindowStart: start, AsOf: asOf, Limit: limit, CursorRank: cursor.Rank, CursorPublishedAt: cursor.PublishedAtPtr(), CursorID: cursor.ID})
	if err != nil {
		return ResearchThemeListResponse{}, mapResearchRepositoryError(err)
	}
	response := ResearchThemeListResponse{WindowStart: formatTime(page.WindowStart), WindowEnd: formatTime(page.WindowEnd), AsOf: formatTime(page.AsOf), ThemeCount: page.ThemeCount, EventCount: page.EventCount, Items: make([]ResearchThemeItem, 0, len(page.Items))}
	for _, item := range page.Items {
		response.Items = append(response.Items, themeItemDTO(item))
	}
	if page.HasMore && len(page.Items) > 0 {
		last := page.Items[len(page.Items)-1]
		next, err := encodeResearchCursor(researchCursor{Version: 1, Kind: "themes", WindowHours: windowHours, AsOf: asOf, Rank: impactRank(last.ImpactLevel), PublishedAt: last.PublishedAt, ID: last.ID})
		if err != nil {
			return ResearchThemeListResponse{}, fmt.Errorf("encode research cursor: %w", err)
		}
		response.NextCursor = &next
	}
	return response, nil
}

func (s *ResearchService) GetTheme(ctx context.Context, id string, request ResearchDetailRequest) (ResearchThemeDetailResponse, error) {
	windowHours, err := normalizeResearchDetailRequest(request)
	if err != nil {
		return ResearchThemeDetailResponse{}, err
	}
	if !researchUUIDPattern.MatchString(strings.TrimSpace(id)) {
		return ResearchThemeDetailResponse{}, fmt.Errorf("%w: theme id must be a UUID", ErrInvalidResearchRequest)
	}
	asOf := s.now().UTC()
	item, err := s.repository.GetResearchTheme(ctx, id, repositories.ResearchDetailFilter{WindowStart: asOf.Add(-time.Duration(windowHours) * time.Hour), AsOf: asOf})
	if err != nil {
		return ResearchThemeDetailResponse{}, mapResearchRepositoryError(err)
	}
	result := ResearchThemeDetailResponse{ResearchThemeItem: themeItemDTO(item.ResearchThemeSummary), Events: eventDTOs(item.Events)}
	return result, nil
}

func (s *ResearchService) ListAnchors(ctx context.Context, request ResearchListRequest) (ResearchAnchorListResponse, error) {
	windowHours, limit, err := normalizeResearchListRequest(request)
	if err != nil {
		return ResearchAnchorListResponse{}, err
	}
	asOf, start, cursor, err := s.prepareCursor("anchors", windowHours, request.Cursor)
	if err != nil {
		return ResearchAnchorListResponse{}, err
	}
	page, err := s.repository.ListResearchAnchors(ctx, repositories.ResearchAnchorListFilter{WindowStart: start, AsOf: asOf, Limit: limit, CursorRank: cursor.Rank, CursorPublishedAt: cursor.PublishedAtPtr(), CursorID: cursor.ID})
	if err != nil {
		return ResearchAnchorListResponse{}, mapResearchRepositoryError(err)
	}
	response := ResearchAnchorListResponse{WindowStart: formatTime(page.WindowStart), WindowEnd: formatTime(page.WindowEnd), AsOf: formatTime(page.AsOf), AnchorCount: page.AnchorCount, EventCount: page.EventCount, Items: make([]ResearchAnchorItem, 0, len(page.Items))}
	for _, item := range page.Items {
		response.Items = append(response.Items, anchorItemDTO(item))
	}
	if page.HasMore && len(page.Items) > 0 {
		last := page.Items[len(page.Items)-1]
		next, err := encodeResearchCursor(researchCursor{Version: 1, Kind: "anchors", WindowHours: windowHours, AsOf: asOf, Rank: importanceRank(last.Importance), PublishedAt: last.PublishedAt, ID: last.ID})
		if err != nil {
			return ResearchAnchorListResponse{}, fmt.Errorf("encode research cursor: %w", err)
		}
		response.NextCursor = &next
	}
	return response, nil
}

func (s *ResearchService) GetAnchor(ctx context.Context, id string, request ResearchDetailRequest) (ResearchAnchorDetailResponse, error) {
	windowHours, err := normalizeResearchDetailRequest(request)
	if err != nil {
		return ResearchAnchorDetailResponse{}, err
	}
	if !researchUUIDPattern.MatchString(strings.TrimSpace(id)) {
		return ResearchAnchorDetailResponse{}, fmt.Errorf("%w: anchor id must be a UUID", ErrInvalidResearchRequest)
	}
	asOf := s.now().UTC()
	item, err := s.repository.GetResearchAnchor(ctx, id, repositories.ResearchDetailFilter{WindowStart: asOf.Add(-time.Duration(windowHours) * time.Hour), AsOf: asOf})
	if err != nil {
		return ResearchAnchorDetailResponse{}, mapResearchRepositoryError(err)
	}
	return ResearchAnchorDetailResponse{ResearchAnchorItem: anchorItemDTO(item.ResearchAnchorSummary), Events: eventDTOs(item.Events)}, nil
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

type researchCursor struct {
	Version     int       `json:"v"`
	Kind        string    `json:"kind"`
	WindowHours int       `json:"window_hours"`
	AsOf        time.Time `json:"as_of"`
	Rank        int       `json:"rank"`
	PublishedAt time.Time `json:"published_at"`
	ID          string    `json:"id"`
}

func (c researchCursor) PublishedAtPtr() *time.Time {
	value := c.PublishedAt
	if c.ID == "" {
		return nil
	}
	return &value
}

func (s *ResearchService) prepareCursor(kind string, windowHours int, encoded string) (time.Time, time.Time, researchCursor, error) {
	if strings.TrimSpace(encoded) == "" {
		asOf := s.now().UTC()
		return asOf, asOf.Add(-time.Duration(windowHours) * time.Hour), researchCursor{}, nil
	}
	cursor, err := decodeResearchCursor(encoded)
	if err != nil || cursor.Kind != kind || cursor.WindowHours != windowHours || cursor.Version != 1 || cursor.ID == "" {
		return time.Time{}, time.Time{}, researchCursor{}, fmt.Errorf("%w: invalid cursor", ErrInvalidResearchRequest)
	}
	return cursor.AsOf.UTC(), cursor.AsOf.UTC().Add(-time.Duration(windowHours) * time.Hour), cursor, nil
}

func encodeResearchCursor(cursor researchCursor) (string, error) {
	payload, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}
func decodeResearchCursor(value string) (researchCursor, error) {
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return researchCursor{}, err
	}
	var cursor researchCursor
	if err := json.Unmarshal(payload, &cursor); err != nil {
		return researchCursor{}, err
	}
	return cursor, nil
}

func themeItemDTO(item repositories.ResearchThemeSummary) ResearchThemeItem {
	return ResearchThemeItem{ID: item.ID, Name: item.Name, OneLineConclusion: item.OneLineConclusion, ImpactLevel: string(item.ImpactLevel), TransmissionPath: item.TransmissionPath, TradingDirection: item.TradingDirection, TransmissionStage: string(item.TransmissionStage), NextCheckpoint: item.NextCheckpoint, IndexImpactSummary: item.IndexImpactSummary, PublishedAt: formatTime(item.PublishedAt), AffectedChainNodes: chainNodeDTOs(item.ChainNodes), RelatedIndices: indexDTOs(item.Indices), SupportingEventCount: item.SupportingEventCount, ContradictingEventCount: item.ContradictingEventCount, HasMoreDetail: true}
}
func anchorItemDTO(item repositories.ResearchAnchorSummary) ResearchAnchorItem {
	return ResearchAnchorItem{ID: item.ID, AnchorType: string(item.AnchorType), Name: item.Name, OneLineConclusion: item.OneLineConclusion, Importance: string(item.Importance), TransmissionPath: item.TransmissionPath, TradingDirection: item.TradingDirection, PublishedAt: formatTime(item.PublishedAt), RelatedChainNodes: anchorChainNodeDTOs(item.ChainNodes), RelatedIndices: indexDTOs(item.Indices), RelatedEventCount: item.RelatedEventCount}
}
func chainNodeDTOs(values []repositories.ResearchChainNode) []ResearchChainNodeDTO {
	result := make([]ResearchChainNodeDTO, 0, len(values))
	for _, value := range values {
		result = append(result, ResearchChainNodeDTO{ID: value.ID, Name: value.Name, RelationRole: value.RelationRole, Summary: value.Summary})
	}
	return result
}
func indexDTOs(values []repositories.ResearchIndex) []ResearchIndexDTO {
	result := make([]ResearchIndexDTO, 0, len(values))
	for _, value := range values {
		result = append(result, ResearchIndexDTO{ID: value.ID, Name: value.Name, ImpactDirection: value.ImpactDirection, Summary: value.Summary})
	}
	return result
}

func anchorChainNodeDTOs(values []repositories.ResearchChainNode) []ResearchAnchorChainNodeDTO {
	result := make([]ResearchAnchorChainNodeDTO, 0, len(values))
	for _, value := range values {
		result = append(result, ResearchAnchorChainNodeDTO{ID: value.ID, Name: value.Name, RelationRole: value.RelationRole, RelationSummary: value.Summary})
	}
	return result
}
func eventDTOs(values []repositories.ResearchEvent) []ResearchEventDTO {
	result := make([]ResearchEventDTO, 0, len(values))
	for _, value := range values {
		var eventTime *string
		if value.EventTime != nil {
			formatted := formatTime(*value.EventTime)
			eventTime = &formatted
		}
		result = append(result, ResearchEventDTO{EventID: value.EventID, Title: value.Title, Summary: value.Summary, EventTime: eventTime, EvidenceRole: value.EvidenceRole, SupportedClaim: value.SupportedClaim})
	}
	return result
}
func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339) }
func impactRank(value domain.ImpactLevel) int {
	switch value {
	case domain.ImpactLevelHigh:
		return 3
	case domain.ImpactLevelFocus:
		return 2
	default:
		return 1
	}
}
func importanceRank(value domain.ResearchImportance) int {
	switch value {
	case domain.ResearchImportancePrimary:
		return 3
	case domain.ResearchImportanceSecondary:
		return 2
	default:
		return 1
	}
}
func mapResearchRepositoryError(err error) error {
	if errors.Is(err, repositories.ErrResearchNotFound) {
		return repositories.ErrResearchNotFound
	}
	return fmt.Errorf("%w: %v", ErrResearchRepository, err)
}

func RegisterResearchRoutes(group *gin.RouterGroup, service *ResearchService) {
	research := group.Group("/miniapp/research")
	research.GET("/themes", func(ctx *gin.Context) { handleListThemes(ctx, service) })
	research.GET("/themes/:theme_id", func(ctx *gin.Context) { handleGetTheme(ctx, service) })
	research.GET("/anchors", func(ctx *gin.Context) { handleListAnchors(ctx, service) })
	research.GET("/anchors/:anchor_id", func(ctx *gin.Context) { handleGetAnchor(ctx, service) })
}

func handleListThemes(ctx *gin.Context, service *ResearchService) {
	response, err := service.ListThemes(ctx.Request.Context(), parseResearchListRequest(ctx))
	writeResearchResponse(ctx, response, err)
}
func handleGetTheme(ctx *gin.Context, service *ResearchService) {
	response, err := service.GetTheme(ctx.Request.Context(), ctx.Param("theme_id"), parseResearchDetailRequest(ctx))
	writeResearchResponse(ctx, response, err)
}
func handleListAnchors(ctx *gin.Context, service *ResearchService) {
	response, err := service.ListAnchors(ctx.Request.Context(), parseResearchListRequest(ctx))
	writeResearchResponse(ctx, response, err)
}
func handleGetAnchor(ctx *gin.Context, service *ResearchService) {
	response, err := service.GetAnchor(ctx.Request.Context(), ctx.Param("anchor_id"), parseResearchDetailRequest(ctx))
	writeResearchResponse(ctx, response, err)
}

func parseResearchListRequest(ctx *gin.Context) ResearchListRequest {
	return ResearchListRequest{WindowHours: parseIntQuery(ctx, "window_hours"), Limit: parseIntQuery(ctx, "limit"), Cursor: ctx.Query("cursor")}
}
func parseResearchDetailRequest(ctx *gin.Context) ResearchDetailRequest {
	return ResearchDetailRequest{WindowHours: parseIntQuery(ctx, "window_hours")}
}
func parseIntQuery(ctx *gin.Context, key string) int {
	value := strings.TrimSpace(ctx.Query(key))
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return -1
	}
	return parsed
}
func writeResearchResponse(ctx *gin.Context, response any, err error) {
	if err == nil {
		ctx.JSON(http.StatusOK, response)
		return
	}
	status := http.StatusInternalServerError
	if errors.Is(err, ErrInvalidResearchRequest) {
		status = http.StatusBadRequest
	} else if errors.Is(err, repositories.ErrResearchNotFound) {
		status = http.StatusNotFound
	}
	ctx.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
}
