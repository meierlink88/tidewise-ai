package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
)

var ErrResearchNotFound = errors.New("research result not found")

type ResearchChainNode struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	RelationRole string `json:"relation_role"`
	Summary      string `json:"impact_summary"`
}

type ResearchIndex struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	ImpactDirection string `json:"impact_direction"`
	Summary         string `json:"impact_summary"`
}

type ResearchEvent struct {
	EventID        string     `json:"event_id"`
	Title          string     `json:"title"`
	Summary        string     `json:"summary"`
	EventTime      *time.Time `json:"event_time,omitempty"`
	EvidenceRole   string     `json:"evidence_role"`
	SupportedClaim string     `json:"supported_claim"`
}

type ResearchThemeSummary struct {
	ID                        string
	Name                      string
	OneLineConclusion         string
	ImpactLevel               domain.ImpactLevel
	TransmissionPath          string
	TradingDirection          string
	TransmissionStage         domain.TransmissionStage
	NextCheckpoint            string
	MarketConfirmationSummary string
	PublishedAt               time.Time
	ChainNodes                []ResearchChainNode
	Indices                   []ResearchIndex
	SupportingEventCount      int
	ContradictingEventCount   int
}

type ResearchThemeDetail struct {
	ResearchThemeSummary
	Events []ResearchEvent
}

type ResearchThemeListFilter struct {
	WindowStart       time.Time
	AsOf              time.Time
	Limit             int
	CursorRank        int
	CursorPublishedAt *time.Time
	CursorID          string
}

type ResearchDetailFilter struct {
	WindowStart time.Time
	AsOf        time.Time
}

type ResearchThemePage struct {
	AsOf        time.Time
	WindowStart time.Time
	WindowEnd   time.Time
	ThemeCount  int
	EventCount  int
	Items       []ResearchThemeSummary
	HasMore     bool
}

type ResearchReadRepository interface {
	ListResearchThemes(context.Context, ResearchThemeListFilter) (ResearchThemePage, error)
	GetResearchTheme(context.Context, string, ResearchDetailFilter) (ResearchThemeDetail, error)
	ListResearchThemeReasoningTrees(context.Context, string) (ResearchReasoningTreeList, error)
	GetResearchThemeReasoningTree(context.Context, string, string) (ResearchReasoningTreeDetail, error)
}

const listResearchThemesQuery = `
WITH latest_batch AS MATERIALIZED (
    SELECT id
    FROM research_theme_import_receipts
    WHERE published_at >= $1 AND published_at <= $2
    ORDER BY published_at DESC, id DESC
    LIMIT 1
), visible AS MATERIALIZED (
    SELECT t.*,
           CASE t.impact_level WHEN 'high' THEN 3 WHEN 'focus' THEN 2 ELSE 1 END AS impact_rank
    FROM research_themes t
    JOIN latest_batch b ON b.id = t.import_receipt_id
), page AS (
    SELECT * FROM visible
    WHERE ($3 = 0 OR impact_rank < $3 OR (impact_rank = $3 AND (published_at < $4 OR (published_at = $4 AND id > $5))))
    ORDER BY impact_rank DESC, published_at DESC, id ASC
    LIMIT $6
)
SELECT p.id, p.name, p.one_line_conclusion, p.impact_level, p.transmission_path,
       p.trading_direction, p.transmission_stage, p.next_checkpoint,
	       p.market_confirmation_summary, p.published_at,
       COALESCE((SELECT jsonb_agg(jsonb_build_object('id', n.chain_node_entity_id, 'name', e.name, 'relation_role', n.relation_role, 'impact_summary', n.impact_summary) ORDER BY e.name, n.chain_node_entity_id)
                 FROM research_theme_chain_nodes n JOIN entity_nodes e ON e.id = n.chain_node_entity_id WHERE n.theme_id = p.id), '[]'::jsonb),
       COALESCE((SELECT jsonb_agg(jsonb_build_object('id', i.index_entity_id, 'name', e.name, 'impact_direction', i.impact_direction, 'impact_summary', i.impact_summary) ORDER BY e.name, i.index_entity_id)
                 FROM research_theme_indices i JOIN entity_nodes e ON e.id = i.index_entity_id WHERE i.theme_id = p.id), '[]'::jsonb),
       (SELECT COUNT(DISTINCT event_id) FROM research_theme_events WHERE theme_id = p.id AND evidence_role IN ('driver', 'supporting')),
       (SELECT COUNT(DISTINCT event_id) FROM research_theme_events WHERE theme_id = p.id AND evidence_role = 'contradicting')
FROM page p
ORDER BY p.impact_rank DESC, p.published_at DESC, p.id ASC`

const countResearchThemesQuery = `
WITH latest_batch AS MATERIALIZED (
    SELECT id
    FROM research_theme_import_receipts
    WHERE published_at >= $1 AND published_at <= $2
    ORDER BY published_at DESC, id DESC
    LIMIT 1
)
SELECT COUNT(DISTINCT t.id), COUNT(DISTINCT e.event_id)
FROM research_themes t
JOIN latest_batch b ON b.id = t.import_receipt_id
LEFT JOIN research_theme_events e ON e.theme_id = t.id
`

const getResearchThemeQuery = `
SELECT t.id, t.name, t.one_line_conclusion, t.impact_level, t.transmission_path,
       t.trading_direction, t.transmission_stage, t.next_checkpoint,
	       t.market_confirmation_summary, t.published_at,
       COALESCE((SELECT jsonb_agg(jsonb_build_object('id', n.chain_node_entity_id, 'name', e.name, 'relation_role', n.relation_role, 'impact_summary', n.impact_summary) ORDER BY e.name, n.chain_node_entity_id)
                 FROM research_theme_chain_nodes n JOIN entity_nodes e ON e.id = n.chain_node_entity_id WHERE n.theme_id = t.id), '[]'::jsonb),
       COALESCE((SELECT jsonb_agg(jsonb_build_object('id', i.index_entity_id, 'name', e.name, 'impact_direction', i.impact_direction, 'impact_summary', i.impact_summary) ORDER BY e.name, i.index_entity_id)
                 FROM research_theme_indices i JOIN entity_nodes e ON e.id = i.index_entity_id WHERE i.theme_id = t.id), '[]'::jsonb),
       COALESCE((SELECT jsonb_agg(jsonb_build_object('event_id', e.id, 'title', e.title, 'summary', e.summary, 'event_time', e.event_time, 'evidence_role', r.evidence_role, 'supported_claim', r.supported_claim) ORDER BY e.event_time DESC NULLS LAST, e.id)
                 FROM research_theme_events r JOIN events e ON e.id = r.event_id WHERE r.theme_id = t.id), '[]'::jsonb),
       (SELECT COUNT(DISTINCT event_id) FROM research_theme_events WHERE theme_id = t.id AND evidence_role IN ('driver', 'supporting')),
       (SELECT COUNT(DISTINCT event_id) FROM research_theme_events WHERE theme_id = t.id AND evidence_role = 'contradicting')
FROM research_themes t
WHERE t.id = $1 AND t.published_at IS NOT NULL AND t.published_at >= $2 AND t.published_at <= $3`

func (r PostgresRepository) ListResearchThemes(ctx context.Context, filter ResearchThemeListFilter) (ResearchThemePage, error) {
	rows, err := r.db.QueryContext(ctx, listResearchThemesQuery, filter.WindowStart, filter.AsOf, filter.CursorRank, nullableTime(filter.CursorPublishedAt), nullableString(filter.CursorID), filter.Limit+1)
	if err != nil {
		return ResearchThemePage{}, fmt.Errorf("list research themes: %w", err)
	}
	defer rows.Close()
	items := make([]ResearchThemeSummary, 0, filter.Limit+1)
	for rows.Next() {
		item, err := scanResearchThemeSummary(rows)
		if err != nil {
			return ResearchThemePage{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return ResearchThemePage{}, fmt.Errorf("iterate research themes: %w", err)
	}
	var themeCount, eventCount int
	if err := r.db.QueryRowContext(ctx, countResearchThemesQuery, filter.WindowStart, filter.AsOf).Scan(&themeCount, &eventCount); err != nil {
		return ResearchThemePage{}, fmt.Errorf("count research themes: %w", err)
	}
	hasMore := len(items) > filter.Limit
	if hasMore {
		items = items[:filter.Limit]
	}
	return ResearchThemePage{AsOf: filter.AsOf, WindowStart: filter.WindowStart, WindowEnd: filter.AsOf, ThemeCount: themeCount, EventCount: eventCount, Items: items, HasMore: hasMore}, nil
}

func (r PostgresRepository) GetResearchTheme(ctx context.Context, id string, filter ResearchDetailFilter) (ResearchThemeDetail, error) {
	row := r.db.QueryRowContext(ctx, getResearchThemeQuery, id, filter.WindowStart, filter.AsOf)
	item, err := scanResearchThemeDetail(row)
	if errors.Is(err, sql.ErrNoRows) {
		return ResearchThemeDetail{}, ErrResearchNotFound
	}
	if err != nil {
		return ResearchThemeDetail{}, fmt.Errorf("get research theme: %w", err)
	}
	return item, nil
}

type researchRow interface{ Scan(...any) error }

func scanResearchThemeSummary(row researchRow) (ResearchThemeSummary, error) {
	var item ResearchThemeSummary
	var nodes, indices []byte
	if err := row.Scan(&item.ID, &item.Name, &item.OneLineConclusion, &item.ImpactLevel, &item.TransmissionPath, &item.TradingDirection, &item.TransmissionStage, &item.NextCheckpoint, &item.MarketConfirmationSummary, &item.PublishedAt, &nodes, &indices, &item.SupportingEventCount, &item.ContradictingEventCount); err != nil {
		return ResearchThemeSummary{}, err
	}
	if err := json.Unmarshal(nodes, &item.ChainNodes); err != nil {
		return ResearchThemeSummary{}, fmt.Errorf("decode research theme nodes: %w", err)
	}
	if err := json.Unmarshal(indices, &item.Indices); err != nil {
		return ResearchThemeSummary{}, fmt.Errorf("decode research theme indices: %w", err)
	}
	if item.ChainNodes == nil {
		item.ChainNodes = []ResearchChainNode{}
	}
	if item.Indices == nil {
		item.Indices = []ResearchIndex{}
	}
	return item, nil
}

func scanResearchThemeDetail(row researchRow) (ResearchThemeDetail, error) {
	var item ResearchThemeDetail
	var nodes, indices, events []byte
	if err := row.Scan(&item.ID, &item.Name, &item.OneLineConclusion, &item.ImpactLevel, &item.TransmissionPath, &item.TradingDirection, &item.TransmissionStage, &item.NextCheckpoint, &item.MarketConfirmationSummary, &item.PublishedAt, &nodes, &indices, &events, &item.SupportingEventCount, &item.ContradictingEventCount); err != nil {
		return ResearchThemeDetail{}, err
	}
	if err := decodeResearchCollections(nodes, indices, events, &item.ChainNodes, &item.Indices, &item.Events); err != nil {
		return ResearchThemeDetail{}, err
	}
	return item, nil
}

func decodeResearchCollections(nodes, indices, events []byte, nodeTarget *[]ResearchChainNode, indexTarget *[]ResearchIndex, eventTarget *[]ResearchEvent) error {
	if nodeTarget != nil {
		if err := json.Unmarshal(nodes, nodeTarget); err != nil {
			return fmt.Errorf("decode research nodes: %w", err)
		}
		if *nodeTarget == nil {
			*nodeTarget = []ResearchChainNode{}
		}
	}
	if indexTarget != nil {
		if err := json.Unmarshal(indices, indexTarget); err != nil {
			return fmt.Errorf("decode research indices: %w", err)
		}
		if *indexTarget == nil {
			*indexTarget = []ResearchIndex{}
		}
	}
	if eventTarget != nil {
		if err := json.Unmarshal(events, eventTarget); err != nil {
			return fmt.Errorf("decode research events: %w", err)
		}
		if *eventTarget == nil {
			*eventTarget = []ResearchEvent{}
		}
	}
	return nil
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
