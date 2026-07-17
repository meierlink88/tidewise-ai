// Package research owns Data Service research aggregate queries and transport DTOs.
package research

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

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
	ErrInvalidRequest = errors.New("invalid research request")
	ErrRepository     = errors.New("research repository failure")
	ErrNotFound       = errors.New("research aggregate not found")
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

type ResearchThemePage struct {
	WindowStart time.Time       `json:"window_start"`
	WindowEnd   time.Time       `json:"window_end"`
	AsOf        time.Time       `json:"as_of"`
	ThemeCount  int             `json:"theme_count"`
	EventCount  int             `json:"event_count"`
	Items       []ResearchTheme `json:"items"`
	NextCursor  *string         `json:"next_cursor"`
}

type ResearchTheme struct {
	ID                      string                   `json:"id"`
	Name                    string                   `json:"name"`
	OneLineConclusion       string                   `json:"one_line_conclusion"`
	ImpactLevel             domain.ImpactLevel       `json:"impact_level"`
	TransmissionPath        string                   `json:"transmission_path"`
	TradingDirection        string                   `json:"trading_direction"`
	TransmissionStage       domain.TransmissionStage `json:"transmission_stage"`
	NextCheckpoint          string                   `json:"next_checkpoint"`
	IndexImpactSummary      string                   `json:"index_impact_summary,omitempty"`
	PublishedAt             time.Time                `json:"published_at"`
	AffectedChainNodes      []ResearchThemeChainNode `json:"affected_chain_nodes"`
	RelatedIndices          []ResearchIndex          `json:"related_indices"`
	SupportingEventCount    int                      `json:"supporting_event_count"`
	ContradictingEventCount int                      `json:"contradicting_event_count"`
	HasMoreDetail           bool                     `json:"has_more_detail"`
}

type ResearchThemeDetail struct {
	Theme  ResearchTheme   `json:"theme"`
	Events []ResearchEvent `json:"events"`
}

type ResearchAnchorPage struct {
	WindowStart time.Time        `json:"window_start"`
	WindowEnd   time.Time        `json:"window_end"`
	AsOf        time.Time        `json:"as_of"`
	AnchorCount int              `json:"anchor_count"`
	EventCount  int              `json:"event_count"`
	Items       []ResearchAnchor `json:"items"`
	NextCursor  *string          `json:"next_cursor"`
}

type ResearchAnchor struct {
	ID                string                    `json:"id"`
	AnchorType        domain.AnchorType         `json:"anchor_type"`
	Name              string                    `json:"name"`
	OneLineConclusion string                    `json:"one_line_conclusion"`
	Importance        domain.ResearchImportance `json:"importance"`
	TransmissionPath  string                    `json:"transmission_path"`
	TradingDirection  string                    `json:"trading_direction"`
	PublishedAt       time.Time                 `json:"published_at"`
	RelatedChainNodes []ResearchAnchorChainNode `json:"related_chain_nodes"`
	RelatedIndices    []ResearchIndex           `json:"related_indices"`
	RelatedEventCount int                       `json:"related_event_count"`
}

type ResearchAnchorDetail struct {
	Anchor ResearchAnchor  `json:"anchor"`
	Events []ResearchEvent `json:"events"`
}

type ResearchThemeChainNode struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	RelationRole  string `json:"relation_role"`
	ImpactSummary string `json:"impact_summary"`
}

type ResearchAnchorChainNode struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	RelationRole    string `json:"relation_role"`
	RelationSummary string `json:"relation_summary"`
}

type ResearchIndex struct {
	ID              string                         `json:"id"`
	Name            string                         `json:"name"`
	ImpactDirection domain.ResearchImpactDirection `json:"impact_direction"`
	ImpactSummary   string                         `json:"impact_summary"`
}

type ResearchEvent struct {
	EventID        string                      `json:"event_id"`
	Title          string                      `json:"title"`
	Summary        string                      `json:"summary"`
	EventTime      *time.Time                  `json:"event_time,omitempty"`
	EvidenceRole   domain.ResearchEvidenceRole `json:"evidence_role"`
	SupportedClaim string                      `json:"supported_claim"`
}

type Service struct {
	repository repositories.ResearchReadRepository
	now        func() time.Time
}

func NewService(repository repositories.ResearchReadRepository, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{repository: repository, now: now}
}

func (s *Service) ListThemes(ctx context.Context, request ResearchListRequest) (ResearchThemePage, error) {
	windowHours, limit, err := normalizeListRequest(request)
	if err != nil {
		return ResearchThemePage{}, err
	}
	asOf, windowStart, cursor, err := s.prepareCursor("themes", windowHours, request.Cursor)
	if err != nil {
		return ResearchThemePage{}, err
	}
	page, err := s.repository.ListResearchThemes(ctx, repositories.ResearchThemeListFilter{
		WindowStart: windowStart, AsOf: asOf, Limit: limit, CursorRank: cursor.Rank,
		CursorPublishedAt: cursor.PublishedAtPtr(), CursorID: cursor.ID,
	})
	if err != nil {
		return ResearchThemePage{}, mapRepositoryError(err)
	}
	response := ResearchThemePage{
		WindowStart: page.WindowStart.UTC(), WindowEnd: page.WindowEnd.UTC(), AsOf: page.AsOf.UTC(),
		ThemeCount: page.ThemeCount, EventCount: page.EventCount,
		Items: make([]ResearchTheme, 0, len(page.Items)),
	}
	for _, item := range page.Items {
		response.Items = append(response.Items, themeDTO(item))
	}
	if page.HasMore && len(page.Items) > 0 {
		last := page.Items[len(page.Items)-1]
		next, err := encodeResearchCursor(researchCursor{
			Version: 1, Kind: "themes", WindowHours: windowHours, AsOf: asOf,
			Rank: impactRank(last.ImpactLevel), PublishedAt: last.PublishedAt, ID: last.ID,
		})
		if err != nil {
			return ResearchThemePage{}, fmt.Errorf("encode research cursor: %w", err)
		}
		response.NextCursor = &next
	}
	return response, nil
}

func (s *Service) GetTheme(ctx context.Context, id string, request ResearchDetailRequest) (ResearchThemeDetail, error) {
	windowHours, err := normalizeDetailRequest(request)
	if err != nil {
		return ResearchThemeDetail{}, err
	}
	if !researchUUIDPattern.MatchString(strings.TrimSpace(id)) {
		return ResearchThemeDetail{}, fmt.Errorf("%w: theme id must be a UUID", ErrInvalidRequest)
	}
	asOf := s.now().UTC()
	item, err := s.repository.GetResearchTheme(ctx, id, repositories.ResearchDetailFilter{
		WindowStart: asOf.Add(-time.Duration(windowHours) * time.Hour), AsOf: asOf,
	})
	if err != nil {
		return ResearchThemeDetail{}, mapRepositoryError(err)
	}
	return ResearchThemeDetail{Theme: themeDTO(item.ResearchThemeSummary), Events: eventDTOs(item.Events)}, nil
}

func (s *Service) ListAnchors(ctx context.Context, request ResearchListRequest) (ResearchAnchorPage, error) {
	windowHours, limit, err := normalizeListRequest(request)
	if err != nil {
		return ResearchAnchorPage{}, err
	}
	asOf, windowStart, cursor, err := s.prepareCursor("anchors", windowHours, request.Cursor)
	if err != nil {
		return ResearchAnchorPage{}, err
	}
	page, err := s.repository.ListResearchAnchors(ctx, repositories.ResearchAnchorListFilter{
		WindowStart: windowStart, AsOf: asOf, Limit: limit, CursorRank: cursor.Rank,
		CursorPublishedAt: cursor.PublishedAtPtr(), CursorID: cursor.ID,
	})
	if err != nil {
		return ResearchAnchorPage{}, mapRepositoryError(err)
	}
	response := ResearchAnchorPage{
		WindowStart: page.WindowStart.UTC(), WindowEnd: page.WindowEnd.UTC(), AsOf: page.AsOf.UTC(),
		AnchorCount: page.AnchorCount, EventCount: page.EventCount,
		Items: make([]ResearchAnchor, 0, len(page.Items)),
	}
	for _, item := range page.Items {
		response.Items = append(response.Items, anchorDTO(item))
	}
	if page.HasMore && len(page.Items) > 0 {
		last := page.Items[len(page.Items)-1]
		next, err := encodeResearchCursor(researchCursor{
			Version: 1, Kind: "anchors", WindowHours: windowHours, AsOf: asOf,
			Rank: importanceRank(last.Importance), PublishedAt: last.PublishedAt, ID: last.ID,
		})
		if err != nil {
			return ResearchAnchorPage{}, fmt.Errorf("encode research cursor: %w", err)
		}
		response.NextCursor = &next
	}
	return response, nil
}

func (s *Service) GetAnchor(ctx context.Context, id string, request ResearchDetailRequest) (ResearchAnchorDetail, error) {
	windowHours, err := normalizeDetailRequest(request)
	if err != nil {
		return ResearchAnchorDetail{}, err
	}
	if !researchUUIDPattern.MatchString(strings.TrimSpace(id)) {
		return ResearchAnchorDetail{}, fmt.Errorf("%w: anchor id must be a UUID", ErrInvalidRequest)
	}
	asOf := s.now().UTC()
	item, err := s.repository.GetResearchAnchor(ctx, id, repositories.ResearchDetailFilter{
		WindowStart: asOf.Add(-time.Duration(windowHours) * time.Hour), AsOf: asOf,
	})
	if err != nil {
		return ResearchAnchorDetail{}, mapRepositoryError(err)
	}
	return ResearchAnchorDetail{Anchor: anchorDTO(item.ResearchAnchorSummary), Events: eventDTOs(item.Events)}, nil
}

func normalizeListRequest(request ResearchListRequest) (int, int, error) {
	windowHours := request.WindowHours
	if windowHours == 0 {
		windowHours = DefaultResearchWindowHours
	}
	if windowHours < MinResearchWindowHours || windowHours > MaxResearchWindowHours {
		return 0, 0, fmt.Errorf("%w: window_hours must be between %d and %d", ErrInvalidRequest, MinResearchWindowHours, MaxResearchWindowHours)
	}
	limit := request.Limit
	if limit == 0 {
		limit = DefaultResearchLimit
	}
	if limit < 1 || limit > MaxResearchLimit {
		return 0, 0, fmt.Errorf("%w: limit must be between 1 and %d", ErrInvalidRequest, MaxResearchLimit)
	}
	return windowHours, limit, nil
}

func normalizeDetailRequest(request ResearchDetailRequest) (int, error) {
	windowHours := request.WindowHours
	if windowHours == 0 {
		windowHours = DefaultResearchWindowHours
	}
	if windowHours < MinResearchWindowHours || windowHours > MaxResearchWindowHours {
		return 0, fmt.Errorf("%w: window_hours must be between %d and %d", ErrInvalidRequest, MinResearchWindowHours, MaxResearchWindowHours)
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
	if c.ID == "" {
		return nil
	}
	value := c.PublishedAt
	return &value
}

func (s *Service) prepareCursor(kind string, windowHours int, encoded string) (time.Time, time.Time, researchCursor, error) {
	if strings.TrimSpace(encoded) == "" {
		asOf := s.now().UTC()
		return asOf, asOf.Add(-time.Duration(windowHours) * time.Hour), researchCursor{}, nil
	}
	cursor, err := decodeResearchCursor(encoded)
	if err != nil || cursor.Kind != kind || cursor.WindowHours != windowHours || cursor.Version != 1 || cursor.ID == "" {
		return time.Time{}, time.Time{}, researchCursor{}, fmt.Errorf("%w: invalid cursor", ErrInvalidRequest)
	}
	asOf := cursor.AsOf.UTC()
	return asOf, asOf.Add(-time.Duration(windowHours) * time.Hour), cursor, nil
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

func themeDTO(item repositories.ResearchThemeSummary) ResearchTheme {
	return ResearchTheme{
		ID: item.ID, Name: item.Name, OneLineConclusion: item.OneLineConclusion,
		ImpactLevel: item.ImpactLevel, TransmissionPath: item.TransmissionPath, TradingDirection: item.TradingDirection,
		TransmissionStage: item.TransmissionStage, NextCheckpoint: item.NextCheckpoint,
		IndexImpactSummary: item.IndexImpactSummary, PublishedAt: item.PublishedAt.UTC(),
		AffectedChainNodes: themeChainNodeDTOs(item.ChainNodes), RelatedIndices: indexDTOs(item.Indices),
		SupportingEventCount: item.SupportingEventCount, ContradictingEventCount: item.ContradictingEventCount,
		HasMoreDetail: true,
	}
}

func anchorDTO(item repositories.ResearchAnchorSummary) ResearchAnchor {
	return ResearchAnchor{
		ID: item.ID, AnchorType: item.AnchorType, Name: item.Name, OneLineConclusion: item.OneLineConclusion,
		Importance: item.Importance, TransmissionPath: item.TransmissionPath, TradingDirection: item.TradingDirection,
		PublishedAt: item.PublishedAt.UTC(), RelatedChainNodes: anchorChainNodeDTOs(item.ChainNodes),
		RelatedIndices: indexDTOs(item.Indices), RelatedEventCount: item.RelatedEventCount,
	}
}

func themeChainNodeDTOs(values []repositories.ResearchChainNode) []ResearchThemeChainNode {
	result := make([]ResearchThemeChainNode, 0, len(values))
	for _, value := range values {
		result = append(result, ResearchThemeChainNode{
			ID: value.ID, Name: value.Name, RelationRole: value.RelationRole, ImpactSummary: value.Summary,
		})
	}
	return result
}

func anchorChainNodeDTOs(values []repositories.ResearchChainNode) []ResearchAnchorChainNode {
	result := make([]ResearchAnchorChainNode, 0, len(values))
	for _, value := range values {
		result = append(result, ResearchAnchorChainNode{
			ID: value.ID, Name: value.Name, RelationRole: value.RelationRole, RelationSummary: value.Summary,
		})
	}
	return result
}

func indexDTOs(values []repositories.ResearchIndex) []ResearchIndex {
	result := make([]ResearchIndex, 0, len(values))
	for _, value := range values {
		result = append(result, ResearchIndex{
			ID: value.ID, Name: value.Name,
			ImpactDirection: domain.ResearchImpactDirection(value.ImpactDirection), ImpactSummary: value.Summary,
		})
	}
	return result
}

func eventDTOs(values []repositories.ResearchEvent) []ResearchEvent {
	result := make([]ResearchEvent, 0, len(values))
	for _, value := range values {
		var eventTime *time.Time
		if value.EventTime != nil {
			formatted := value.EventTime.UTC()
			eventTime = &formatted
		}
		result = append(result, ResearchEvent{
			EventID: value.EventID, Title: value.Title, Summary: value.Summary, EventTime: eventTime,
			EvidenceRole: domain.ResearchEvidenceRole(value.EvidenceRole), SupportedClaim: value.SupportedClaim,
		})
	}
	return result
}

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

func mapRepositoryError(err error) error {
	if errors.Is(err, repositories.ErrResearchNotFound) {
		return ErrNotFound
	}
	return fmt.Errorf("%w: %v", ErrRepository, err)
}
