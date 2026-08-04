package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/research"
)

var ErrResearchNotFound = research.ErrResearchNotFound

type ResearchThemeImpact = research.ThemeImpactRecord
type ResearchEvent = research.EventRecord
type ResearchThemeSummary = research.ThemeSummaryRecord
type ResearchThemeDetail = research.ThemeDetailRecord
type ResearchThemeListFilter = research.ThemeListFilter
type ResearchThemePage = research.ThemeStorePage
type ResearchReadRepository = research.Repository

const researchThemeSummaryColumns = `
t.id, t.analysis_batch_id, t.title, t.one_line_conclusion,
t.conclusion_direction, t.impact_strength, t.attention_level, t.conclusion_status,
t.transmission_stage, t.investment_guidance_action, t.investment_guidance_summary,
t.time_horizon_category, t.time_horizon_summary, t.transmission_summary,
t.checkpoint_summary, t.risk_summary, t.analysis_as_of, t.window_start, t.window_end,
t.published_at,
COALESCE((
    SELECT jsonb_agg(jsonb_build_object(
		'node_key', COALESCE(impact.node_key, impact.chain_node_entity_id::text),
		'display_name', COALESCE(impact.display_name, node.name),
		'chain_node_entity_id', COALESCE(impact.chain_node_entity_id::text, ''),
		'name', COALESCE(node.name, impact.display_name),
        'relation_role', impact.relation_role,
        'impact_direction', impact.impact_direction,
        'impact_summary', impact.impact_summary,
        'display_order', impact.display_order
    ) ORDER BY impact.display_order)
    FROM research_theme_impacts impact
	LEFT JOIN entity_nodes node ON node.id = impact.chain_node_entity_id
    WHERE impact.theme_id = t.id
), '[]'::jsonb),
(SELECT count(*) FROM research_theme_events event WHERE event.theme_id = t.id),
(SELECT count(*) FROM research_reasoning_trees tree WHERE tree.theme_id = t.id)`

const listResearchThemesQuery = `
WITH page AS (
    SELECT theme.*
    FROM research_themes theme
    WHERE theme.published_at >= $1
      AND theme.published_at < $2
      AND ($3::timestamptz IS NULL
        OR theme.published_at < $3
        OR (theme.published_at = $3 AND theme.id > $4))
    ORDER BY theme.published_at DESC, theme.id ASC
    LIMIT $5
)
SELECT ` + researchThemeSummaryColumns + `
FROM page t
ORDER BY t.published_at DESC, t.id ASC`

const countResearchThemesQuery = `
SELECT count(DISTINCT theme.id), count(DISTINCT event.event_id)
FROM research_themes theme
LEFT JOIN research_theme_events event ON event.theme_id = theme.id
WHERE theme.published_at >= $1 AND theme.published_at < $2`

const getResearchThemeQuery = `
SELECT ` + researchThemeSummaryColumns + `,
COALESCE((
    SELECT jsonb_agg(jsonb_build_object(
        'event_id', event.id,
		'evidence_ids', association.evidence_ids,
        'title', event.title,
        'summary', event.summary,
        'event_time', event.event_time,
        'evidence_role', association.evidence_role,
        'supported_claim', association.supported_claim
    ) ORDER BY event.event_time DESC NULLS LAST, event.id)
    FROM research_theme_events association
    JOIN events event ON event.id = association.event_id
    WHERE association.theme_id = t.id
), '[]'::jsonb),
t.theme_key,
(SELECT receipt.publication_mode FROM research_theme_import_receipts receipt WHERE receipt.id = t.import_receipt_id),
(SELECT receipt.publication_contract_version FROM research_theme_import_receipts receipt WHERE receipt.id = t.import_receipt_id)
FROM research_themes t
WHERE t.id = $1`

const getResearchThemeByIDQuery = `
SELECT ` + researchThemeSummaryColumns + `
FROM research_themes t
WHERE t.id = $1`

func (r repository) ListResearchThemes(ctx context.Context, filter ResearchThemeListFilter) (ResearchThemePage, error) {
	rows, err := r.db.QueryContext(ctx, listResearchThemesQuery,
		filter.WindowStart, filter.WindowEnd, nullableTime(filter.CursorPublishedAt),
		nullableString(filter.CursorID), filter.Limit+1)
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
	if err := r.db.QueryRowContext(ctx, countResearchThemesQuery,
		filter.WindowStart, filter.WindowEnd).Scan(&themeCount, &eventCount); err != nil {
		return ResearchThemePage{}, fmt.Errorf("count research themes: %w", err)
	}
	hasMore := len(items) > filter.Limit
	if hasMore {
		items = items[:filter.Limit]
	}
	return ResearchThemePage{
		AsOf: filter.AsOf, WindowStart: filter.WindowStart, WindowEnd: filter.WindowEnd,
		ThemeCount: themeCount, EventCount: eventCount, Items: items, HasMore: hasMore,
	}, nil
}

func (r repository) GetResearchTheme(ctx context.Context, id string) (ResearchThemeDetail, error) {
	item, err := scanResearchThemeDetail(r.db.QueryRowContext(ctx, getResearchThemeQuery, id))
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
	var impacts []byte
	if err := row.Scan(
		&item.ID, &item.AnalysisBatchID, &item.Title, &item.OneLineConclusion,
		&item.ConclusionDirection, &item.ImpactStrength, &item.AttentionLevel,
		&item.ConclusionStatus, &item.TransmissionStage, &item.InvestmentGuidanceAction,
		&item.InvestmentGuidanceSummary, &item.TimeHorizonCategory, &item.TimeHorizonSummary,
		&item.TransmissionSummary, &item.CheckpointSummary, &item.RiskSummary,
		&item.AnalysisAsOf, &item.WindowStart, &item.WindowEnd, &item.PublishedAt,
		&impacts, &item.EvidenceEventCount, &item.ReasoningTreeCount,
	); err != nil {
		return ResearchThemeSummary{}, err
	}
	if err := json.Unmarshal(impacts, &item.Impacts); err != nil {
		return ResearchThemeSummary{}, fmt.Errorf("decode research Theme Impacts: %w", err)
	}
	if item.Impacts == nil || len(item.Impacts) == 0 {
		return ResearchThemeSummary{}, fmt.Errorf("research Theme %s has no Impacts", item.ID)
	}
	return item, nil
}

func scanResearchThemeDetail(row researchRow) (ResearchThemeDetail, error) {
	var item ResearchThemeDetail
	var impacts, events []byte
	if err := row.Scan(
		&item.ID, &item.AnalysisBatchID, &item.Title, &item.OneLineConclusion,
		&item.ConclusionDirection, &item.ImpactStrength, &item.AttentionLevel,
		&item.ConclusionStatus, &item.TransmissionStage, &item.InvestmentGuidanceAction,
		&item.InvestmentGuidanceSummary, &item.TimeHorizonCategory, &item.TimeHorizonSummary,
		&item.TransmissionSummary, &item.CheckpointSummary, &item.RiskSummary,
		&item.AnalysisAsOf, &item.WindowStart, &item.WindowEnd, &item.PublishedAt,
		&impacts, &item.EvidenceEventCount, &item.ReasoningTreeCount, &events,
		&item.ThemeKey, &item.PublicationMode, &item.PublicationContractVersion,
	); err != nil {
		return ResearchThemeDetail{}, err
	}
	if err := json.Unmarshal(impacts, &item.Impacts); err != nil {
		return ResearchThemeDetail{}, fmt.Errorf("decode research Theme Impacts: %w", err)
	}
	if len(item.Impacts) == 0 {
		return ResearchThemeDetail{}, fmt.Errorf("research Theme %s has no Impacts", item.ID)
	}
	if err := json.Unmarshal(events, &item.Events); err != nil {
		return ResearchThemeDetail{}, fmt.Errorf("decode research Theme Events: %w", err)
	}
	if item.Events == nil {
		item.Events = []ResearchEvent{}
	}
	return item, nil
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
