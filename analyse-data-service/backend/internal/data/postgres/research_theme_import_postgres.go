package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

func (r repository) InResearchThemeImportTransaction(ctx context.Context, fn func(ResearchThemeImportTransaction) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin research Theme import transaction: %w", err)
	}
	wrapper := &postgresResearchThemeImportTx{tx: tx}
	if err := fn(wrapper); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit research Theme import transaction: %w", err)
	}
	return nil
}

type postgresResearchThemeImportTx struct{ tx *sql.Tx }

const existingResearchThemeEventsSQL = `SELECT id::text
FROM events
WHERE id = ANY($1::uuid[])
  AND event_status = 'confirmed'
  AND fact_status = 'verified'`

func (t *postgresResearchThemeImportTx) LockResearchThemeImportBatch(ctx context.Context, analysisBatchID string) error {
	if _, err := t.tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, analysisBatchID); err != nil {
		return fmt.Errorf("lock analysis batch %q: %w", analysisBatchID, err)
	}
	return nil
}

func (t *postgresResearchThemeImportTx) ResearchThemeImportReceipt(ctx context.Context, analysisBatchID string) (*ResearchThemeImportReceipt, error) {
	var receipt ResearchThemeImportReceipt
	var themeIDsJSON []byte
	var countsJSON []byte
	err := t.tx.QueryRowContext(ctx, `SELECT id, analysis_batch_id, publisher_subject, payload_hash,
       theme_ids_by_key, write_counts, published_at, imported_at
FROM research_theme_import_receipts WHERE analysis_batch_id = $1`, analysisBatchID).Scan(
		&receipt.ID, &receipt.AnalysisBatchID, &receipt.PublisherSubject, &receipt.PayloadHash,
		&themeIDsJSON, &countsJSON, &receipt.PublishedAt, &receipt.ImportedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read research Theme import receipt: %w", err)
	}
	if err := json.Unmarshal(themeIDsJSON, &receipt.ThemeIDsByKey); err != nil {
		return nil, fmt.Errorf("decode receipt theme_ids_by_key: %w", err)
	}
	if err := json.Unmarshal(countsJSON, &receipt.Counts); err != nil {
		return nil, fmt.Errorf("decode receipt write_counts: %w", err)
	}
	if len(receipt.ThemeIDsByKey) == 0 || receipt.Counts.Themes != len(receipt.ThemeIDsByKey) {
		return nil, fmt.Errorf("receipt Theme mapping does not match its Theme count")
	}
	return &receipt, nil
}

func (t *postgresResearchThemeImportTx) ExistingResearchThemeImpactNodes(ctx context.Context, ids []string) (map[string]struct{}, error) {
	return queryExistingResearchThemeIDs(ctx, t.tx, `SELECT profile.entity_id::text
FROM chain_node_profiles profile
JOIN entity_nodes node ON node.id = profile.entity_id
WHERE profile.entity_id = ANY($1::uuid[])
  AND node.status = 'active'
  AND profile.review_status = 'approved'`, ids)
}

func (t *postgresResearchThemeImportTx) ExistingResearchThemeEvents(ctx context.Context, ids []string) (map[string]struct{}, error) {
	return queryExistingResearchThemeIDs(ctx, t.tx, existingResearchThemeEventsSQL, ids)
}

func queryExistingResearchThemeIDs(ctx context.Context, tx *sql.Tx, query string, ids []string) (map[string]struct{}, error) {
	rows, err := tx.QueryContext(ctx, query, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]struct{}, len(ids))
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (t *postgresResearchThemeImportTx) InsertResearchThemeImportReceipt(ctx context.Context, receipt ResearchThemeImportReceipt) error {
	themeIDs, err := json.Marshal(receipt.ThemeIDsByKey)
	if err != nil {
		return fmt.Errorf("encode receipt Theme mapping: %w", err)
	}
	counts, err := json.Marshal(receipt.Counts)
	if err != nil {
		return fmt.Errorf("encode receipt counts: %w", err)
	}
	_, err = t.tx.ExecContext(ctx, `INSERT INTO research_theme_import_receipts (
    id, analysis_batch_id, publisher_subject, payload_hash, theme_ids_by_key,
    write_counts, published_at, imported_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		receipt.ID, receipt.AnalysisBatchID, receipt.PublisherSubject, receipt.PayloadHash,
		themeIDs, counts, receipt.PublishedAt, receipt.ImportedAt,
	)
	return err
}

func (t *postgresResearchThemeImportTx) InsertResearchTheme(ctx context.Context, theme ResearchThemeImportTheme) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO research_themes (
    id, import_receipt_id, analysis_batch_id, theme_key, title, one_line_conclusion,
    conclusion_direction, impact_strength, attention_level, conclusion_status,
    transmission_stage, investment_guidance_action, investment_guidance_summary,
    time_horizon_category, time_horizon_summary, transmission_summary,
    checkpoint_summary, risk_summary, analysis_as_of, window_start, window_end, published_at
) VALUES (
    $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22
)`,
		theme.ID, theme.ImportReceiptID, theme.AnalysisBatchID, theme.ThemeKey, theme.Title,
		theme.OneLineConclusion, theme.ConclusionDirection, theme.ImpactStrength,
		theme.AttentionLevel, theme.ConclusionStatus, theme.TransmissionStage,
		theme.InvestmentGuidanceAction, theme.InvestmentGuidanceSummary,
		theme.TimeHorizonCategory, theme.TimeHorizonSummary, theme.TransmissionSummary,
		theme.CheckpointSummary, theme.RiskSummary, theme.AnalysisAsOf,
		theme.WindowStart, theme.WindowEnd, theme.PublishedAt,
	)
	return err
}

func (t *postgresResearchThemeImportTx) InsertResearchThemeImpact(ctx context.Context, impact ResearchThemeImportImpact) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO research_theme_impacts (
    theme_id, chain_node_entity_id, relation_role, impact_direction, impact_summary, display_order
) VALUES ($1,$2,$3,$4,$5,$6)`, impact.ThemeID, impact.ChainNodeEntityID,
		impact.RelationRole, impact.ImpactDirection, impact.ImpactSummary, impact.DisplayOrder)
	return err
}

func (t *postgresResearchThemeImportTx) InsertResearchThemeEvent(ctx context.Context, event ResearchThemeImportEvent) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO research_theme_events (
    theme_id, event_id, evidence_role, supported_claim
) VALUES ($1,$2,$3,$4)`, event.ThemeID, event.EventID, event.EvidenceRole, event.SupportedClaim)
	return err
}

func (t *postgresResearchThemeImportTx) VerifyResearchThemeImportReceipt(ctx context.Context, receipt ResearchThemeImportReceipt) error {
	var themeIDsJSON []byte
	if err := t.tx.QueryRowContext(ctx, `SELECT COALESCE(jsonb_object_agg(theme_key, id::text), '{}'::jsonb)
FROM research_themes WHERE import_receipt_id = $1`, receipt.ID).Scan(&themeIDsJSON); err != nil {
		return fmt.Errorf("verify receipt Theme IDs: %w", err)
	}
	var themeIDs map[string]string
	if err := json.Unmarshal(themeIDsJSON, &themeIDs); err != nil {
		return fmt.Errorf("decode verified receipt Theme IDs: %w", err)
	}
	if !reflect.DeepEqual(themeIDs, receipt.ThemeIDsByKey) {
		return fmt.Errorf("receipt Theme IDs are not all present")
	}

	var counts ResearchThemeImportCounts
	if err := t.tx.QueryRowContext(ctx, `SELECT
    (SELECT count(*) FROM research_themes WHERE import_receipt_id = $1),
    (SELECT count(*) FROM research_theme_impacts i JOIN research_themes t ON t.id = i.theme_id WHERE t.import_receipt_id = $1),
    (SELECT count(*) FROM research_theme_events e JOIN research_themes t ON t.id = e.theme_id WHERE t.import_receipt_id = $1)`, receipt.ID).Scan(
		&counts.Themes, &counts.Impacts, &counts.EventAssociations,
	); err != nil {
		return fmt.Errorf("verify receipt write counts: %w", err)
	}
	counts.Receipts = 1
	if counts != receipt.Counts {
		return fmt.Errorf("receipt write counts do not match persisted rows")
	}
	return nil
}
