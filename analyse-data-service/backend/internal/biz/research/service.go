// Package research owns Data Service research aggregate queries and business result models.
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

type ResearchDetailRequest struct{ WindowHours int }

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
	ID                        string                `json:"id"`
	AnalysisBatchID           string                `json:"analysis_batch_id"`
	Title                     string                `json:"title"`
	OneLineConclusion         string                `json:"one_line_conclusion"`
	ConclusionDirection       string                `json:"conclusion_direction"`
	ImpactStrength            string                `json:"impact_strength"`
	AttentionLevel            *string               `json:"attention_level"`
	ConclusionStatus          *string               `json:"conclusion_status"`
	TransmissionStage         string                `json:"transmission_stage"`
	InvestmentGuidanceAction  string                `json:"investment_guidance_action"`
	InvestmentGuidanceSummary string                `json:"investment_guidance_summary"`
	TimeHorizonCategory       string                `json:"time_horizon_category"`
	TimeHorizonSummary        *string               `json:"time_horizon_summary"`
	TransmissionSummary       *string               `json:"transmission_summary"`
	CheckpointSummary         *string               `json:"checkpoint_summary"`
	RiskSummary               *string               `json:"risk_summary"`
	AnalysisAsOf              time.Time             `json:"analysis_as_of"`
	WindowStart               time.Time             `json:"window_start"`
	WindowEnd                 time.Time             `json:"window_end"`
	PublishedAt               time.Time             `json:"published_at"`
	Impacts                   []ResearchThemeImpact `json:"impacts"`
	EvidenceEventCount        int                   `json:"evidence_event_count"`
	ReasoningTreeCount        int                   `json:"reasoning_tree_count"`
}

type ResearchThemeImpact struct {
	ChainNodeEntityID string  `json:"chain_node_entity_id"`
	Name              string  `json:"name"`
	RelationRole      string  `json:"relation_role"`
	ImpactDirection   string  `json:"impact_direction"`
	ImpactSummary     *string `json:"impact_summary"`
	DisplayOrder      int     `json:"display_order"`
}

type ResearchThemeDetail struct {
	Theme  ResearchTheme   `json:"theme"`
	Events []ResearchEvent `json:"events"`
}

type ResearchEvent struct {
	EventID        string     `json:"event_id"`
	Title          string     `json:"title"`
	Summary        string     `json:"summary"`
	EventTime      *time.Time `json:"event_time"`
	EvidenceRole   string     `json:"evidence_role"`
	SupportedClaim *string    `json:"supported_claim,omitempty"`
	DisplayOrder   int        `json:"display_order,omitempty"`
}

type Service struct {
	repository Repository
	now        func() time.Time
}

func NewService(repository Repository, now func() time.Time) *Service {
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
	page, err := s.repository.ListResearchThemes(ctx, ThemeListFilter{
		WindowStart: windowStart, AsOf: asOf, Limit: limit,
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
			PublishedAt: last.PublishedAt, ID: last.ID,
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
	item, err := s.repository.GetResearchTheme(ctx, id, DetailFilter{
		WindowStart: asOf.Add(-time.Duration(windowHours) * time.Hour), AsOf: asOf,
	})
	if err != nil {
		return ResearchThemeDetail{}, mapRepositoryError(err)
	}
	return ResearchThemeDetail{Theme: themeDTO(item.ThemeSummaryRecord), Events: eventDTOs(item.Events)}, nil
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

func themeDTO(item ThemeSummaryRecord) ResearchTheme {
	return ResearchTheme{
		ID: item.ID, AnalysisBatchID: item.AnalysisBatchID, Title: item.Title,
		OneLineConclusion: item.OneLineConclusion, ConclusionDirection: item.ConclusionDirection,
		ImpactStrength: item.ImpactStrength, AttentionLevel: item.AttentionLevel,
		ConclusionStatus: item.ConclusionStatus, TransmissionStage: item.TransmissionStage,
		InvestmentGuidanceAction:  item.InvestmentGuidanceAction,
		InvestmentGuidanceSummary: item.InvestmentGuidanceSummary,
		TimeHorizonCategory:       item.TimeHorizonCategory, TimeHorizonSummary: item.TimeHorizonSummary,
		TransmissionSummary: item.TransmissionSummary, CheckpointSummary: item.CheckpointSummary,
		RiskSummary: item.RiskSummary, AnalysisAsOf: item.AnalysisAsOf.UTC(),
		WindowStart: item.WindowStart.UTC(), WindowEnd: item.WindowEnd.UTC(),
		PublishedAt: item.PublishedAt.UTC(), Impacts: impactDTOs(item.Impacts),
		EvidenceEventCount: item.EvidenceEventCount, ReasoningTreeCount: item.ReasoningTreeCount,
	}
}

func impactDTOs(values []ThemeImpactRecord) []ResearchThemeImpact {
	result := make([]ResearchThemeImpact, 0, len(values))
	for _, value := range values {
		result = append(result, ResearchThemeImpact{
			ChainNodeEntityID: value.ChainNodeEntityID, Name: value.Name,
			RelationRole: value.RelationRole, ImpactDirection: value.ImpactDirection,
			ImpactSummary: value.ImpactSummary, DisplayOrder: value.DisplayOrder,
		})
	}
	return result
}

func eventDTOs(values []EventRecord) []ResearchEvent {
	result := make([]ResearchEvent, 0, len(values))
	for index, value := range values {
		var eventTime *time.Time
		if value.EventTime != nil {
			formatted := value.EventTime.UTC()
			eventTime = &formatted
		}
		displayOrder := value.DisplayOrder
		if displayOrder == 0 {
			displayOrder = index + 1
		}
		result = append(result, ResearchEvent{
			EventID: value.EventID, Title: value.Title, Summary: value.Summary,
			EventTime: eventTime, EvidenceRole: value.EvidenceRole,
			SupportedClaim: value.SupportedClaim, DisplayOrder: displayOrder,
		})
	}
	return result
}

func mapRepositoryError(err error) error {
	switch {
	case errors.Is(err, ErrResearchNotFound), errors.Is(err, ErrResearchThemeNotFound):
		return ErrNotFound
	default:
		return fmt.Errorf("%w: %v", ErrRepository, err)
	}
}
