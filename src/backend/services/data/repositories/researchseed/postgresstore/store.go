package postgresstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/data/usecase/researchseed"
)

const resolveChainNodeQuery = `SELECT min(e.id::text)
FROM entity_nodes e
JOIN chain_node_profiles p ON p.entity_id = e.id
WHERE e.entity_type = 'chain_node' AND e.status = 'active' AND e.name = $1
HAVING count(*) = 1`

const resolveEventQuery = `SELECT EXISTS (SELECT 1 FROM events WHERE id = $1)`

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

type resolvedTheme struct {
	theme        researchseed.Theme
	chainNodeIDs map[string]string
}

func (s *Store) Apply(ctx context.Context, manifest researchseed.Manifest, publishedAt time.Time) (researchseed.Report, error) {
	if s == nil || s.db == nil {
		return researchseed.Report{}, fmt.Errorf("database is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return researchseed.Report{}, fmt.Errorf("begin research theme seed transaction: %w", err)
	}
	defer tx.Rollback()

	resolved, err := resolveReferences(ctx, tx, manifest)
	if err != nil {
		return researchseed.Report{}, err
	}

	report := researchseed.Report{AnalysisBatchID: manifest.AnalysisBatchID, PublishedAt: publishedAt}
	for _, item := range resolved {
		if err := upsertTheme(ctx, tx, manifest.AnalysisBatchID, item.theme, publishedAt); err != nil {
			return researchseed.Report{}, err
		}
		if err := replaceRelations(ctx, tx, item); err != nil {
			return researchseed.Report{}, err
		}
		report.ThemeCount++
		report.ChainNodeCount += len(item.theme.ChainNodes)
		report.EventCount += len(item.theme.Events)
	}
	if err := tx.Commit(); err != nil {
		return researchseed.Report{}, fmt.Errorf("commit research theme seed transaction: %w", err)
	}
	return report, nil
}

func resolveReferences(ctx context.Context, tx *sql.Tx, manifest researchseed.Manifest) ([]resolvedTheme, error) {
	resolved := make([]resolvedTheme, 0, len(manifest.Themes))
	for _, theme := range manifest.Themes {
		item := resolvedTheme{theme: theme, chainNodeIDs: make(map[string]string, len(theme.ChainNodes))}
		for _, node := range theme.ChainNodes {
			var id string
			if err := tx.QueryRowContext(ctx, resolveChainNodeQuery, node.Name).Scan(&id); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return nil, fmt.Errorf("active chain node %q was not found uniquely", node.Name)
				}
				return nil, fmt.Errorf("resolve chain node %q: %w", node.Name, err)
			}
			item.chainNodeIDs[node.Name] = id
		}
		for _, event := range theme.Events {
			var exists bool
			if err := tx.QueryRowContext(ctx, resolveEventQuery, event.ID).Scan(&exists); err != nil {
				return nil, fmt.Errorf("resolve event %q: %w", event.ID, err)
			}
			if !exists {
				return nil, fmt.Errorf("event %q was not found", event.ID)
			}
		}
		resolved = append(resolved, item)
	}
	return resolved, nil
}

func upsertTheme(ctx context.Context, tx *sql.Tx, batchID string, theme researchseed.Theme, publishedAt time.Time) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO research_themes (
    id, analysis_batch_id, name, one_line_conclusion, impact_level,
    transmission_path, trading_direction, transmission_stage, next_checkpoint,
    index_impact_summary, window_start, window_end, published_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
ON CONFLICT (id) DO UPDATE SET
    analysis_batch_id = EXCLUDED.analysis_batch_id,
    name = EXCLUDED.name,
    one_line_conclusion = EXCLUDED.one_line_conclusion,
    impact_level = EXCLUDED.impact_level,
    transmission_path = EXCLUDED.transmission_path,
    trading_direction = EXCLUDED.trading_direction,
    transmission_stage = EXCLUDED.transmission_stage,
    next_checkpoint = EXCLUDED.next_checkpoint,
    index_impact_summary = EXCLUDED.index_impact_summary,
    window_start = EXCLUDED.window_start,
    window_end = EXCLUDED.window_end,
    published_at = EXCLUDED.published_at,
    updated_at = now()`,
		theme.ID, batchID, theme.Name, theme.OneLineConclusion, theme.ImpactLevel,
		theme.TransmissionPath, theme.TradingDirection, theme.TransmissionStage, theme.NextCheckpoint,
		theme.IndexImpactSummary, publishedAt.Add(-72*time.Hour), publishedAt, publishedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert research theme %q: %w", theme.Name, err)
	}
	return nil
}

func replaceRelations(ctx context.Context, tx *sql.Tx, item resolvedTheme) error {
	for _, table := range []string{"research_theme_chain_nodes", "research_theme_indices", "research_theme_events"} {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+table+" WHERE theme_id = $1", item.theme.ID); err != nil {
			return fmt.Errorf("clear %s for theme %q: %w", table, item.theme.Name, err)
		}
	}
	for _, node := range item.theme.ChainNodes {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO research_theme_chain_nodes (theme_id, chain_node_entity_id, relation_role, impact_summary)
VALUES ($1,$2,$3,$4)`, item.theme.ID, item.chainNodeIDs[node.Name], node.RelationRole, node.ImpactSummary); err != nil {
			return fmt.Errorf("insert chain node %q for theme %q: %w", node.Name, item.theme.Name, err)
		}
	}
	for _, event := range item.theme.Events {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO research_theme_events (theme_id, event_id, evidence_role, supported_claim)
VALUES ($1,$2,$3,$4)`, item.theme.ID, event.ID, event.EvidenceRole, event.SupportedClaim); err != nil {
			return fmt.Errorf("insert event %q for theme %q: %w", event.ID, item.theme.Name, err)
		}
	}
	return nil
}
