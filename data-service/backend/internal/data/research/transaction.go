package research

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	researchbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/research"
	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

func (r Store) InResearchPublicationTransaction(
	ctx context.Context,
	fn func(researchbiz.PublicationTransaction) error,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin research publication transaction: %w", err)
	}
	wrapper := &publicationTransaction{tx: tx}
	if err := fn(wrapper); err != nil {
		return errors.Join(err, tx.Rollback())
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit research publication transaction: %w", err)
	}
	return nil
}

type publicationTransaction struct{ tx *sql.Tx }

func (t *publicationTransaction) Lock(ctx context.Context, analysisBatchID string) error {
	_, err := t.tx.ExecContext(ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`,
		"research-publication-v3:"+analysisBatchID,
	)
	return err
}

func (t *publicationTransaction) Receipt(ctx context.Context, analysisBatchID string) (*researchbiz.Receipt, error) {
	var receipt researchbiz.Receipt
	var treeIDsJSON, countsJSON []byte
	err := t.tx.QueryRowContext(ctx, `SELECT
    id::text, analysis_batch_id, publisher_subject, payload_hash,
    publication_contract_version, publication_mode, aggregate_theme_id::text,
    reasoning_tree_ids_by_tree_key, aggregate_write_counts,
    published_at, imported_at
FROM research_theme_import_receipts
WHERE analysis_batch_id = $1`, analysisBatchID).Scan(
		&receipt.ID, &receipt.AnalysisBatchID, &receipt.PublisherSubject, &receipt.PayloadHash,
		&receipt.ContractVersion, &receipt.PublicationMode, &receipt.ThemeID,
		&treeIDsJSON, &countsJSON, &receipt.PublishedAt, &receipt.ImportedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(treeIDsJSON, &receipt.ReasoningTreeIDsByTreeKey); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(countsJSON, &receipt.Counts); err != nil {
		return nil, err
	}
	if err := t.tx.QueryRowContext(ctx,
		`SELECT theme_key FROM research_themes WHERE id = $1`, receipt.ThemeID,
	).Scan(&receipt.ThemeKey); err != nil {
		return nil, err
	}
	if err := validatePersistedResearchReceipt(receipt); err != nil {
		return nil, err
	}
	return &receipt, nil
}

func validatePersistedResearchReceipt(receipt researchbiz.Receipt) error {
	invalid := func(reason string) error {
		return fmt.Errorf("persisted Research publication Receipt %q violates invariants: %s", receipt.ID, reason)
	}
	if !coreid.Is(receipt.ID, coreid.ResearchThemeReceipt) || !coreid.Is(receipt.ThemeID, coreid.ResearchTheme) ||
		strings.TrimSpace(receipt.AnalysisBatchID) == "" || strings.TrimSpace(receipt.PublisherSubject) == "" ||
		!researchHashPattern.MatchString(receipt.PayloadHash) || !researchKeyPattern.MatchString(receipt.ThemeKey) {
		return invalid("required identity is malformed")
	}
	if receipt.ContractVersion != 3 || receipt.PublicationMode != researchbiz.SnapshotPublicationMode {
		return invalid("only analyst_snapshot contract version 3 is supported")
	}
	if receipt.PublishedAt.IsZero() || receipt.ImportedAt.IsZero() || receipt.ImportedAt.Before(receipt.PublishedAt) {
		return invalid("publication timestamps are invalid")
	}
	counts := receipt.Counts
	if counts.Themes != 1 || counts.Receipts != 2 || counts.Impacts < 1 ||
		counts.ThemeEventAssociations < 0 || counts.ReasoningTrees < 1 || counts.Nodes < 1 ||
		counts.TreeEventAssociations < 0 || counts.SignalAssociations < 1 {
		return invalid("aggregate write counts are invalid")
	}
	if len(receipt.ReasoningTreeIDsByTreeKey) != counts.ReasoningTrees {
		return invalid("Reason Tree identity count does not match write counts")
	}
	for key, id := range receipt.ReasoningTreeIDsByTreeKey {
		if !researchKeyPattern.MatchString(key) || !coreid.Is(id, coreid.ResearchReasoningTree) {
			return invalid("a snapshot Reason Tree identity is malformed")
		}
	}
	return nil
}

func (t *publicationTransaction) ReferenceFacts(
	ctx context.Context,
	query researchbiz.ReferenceQuery,
) (researchbiz.ReferenceFacts, error) {
	facts := researchbiz.ReferenceFacts{}
	var err error
	facts.Events, err = t.events(ctx, query.EventIDs)
	if err != nil {
		return facts, err
	}
	facts.Evidences, err = t.evidences(ctx, query.EvidenceIDs)
	return facts, err
}

func (t *publicationTransaction) events(ctx context.Context, ids []string) (map[string]researchbiz.EventFact, error) {
	rows, err := t.tx.QueryContext(ctx, `SELECT event.id,
       MIN(GREATEST(COALESCE(raw.published_at, raw.collected_at), raw.collected_at))
FROM events event
JOIN event_evidence_links link ON link.event_id = event.id
JOIN evidences evidence ON evidence.id = link.evidence_id
JOIN raw_evidences raw ON raw.id = evidence.raw_evidence_id
WHERE event.id = ANY($1::text[]) AND event.status = 'ACTIVE'
GROUP BY event.id`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]researchbiz.EventFact)
	for rows.Next() {
		var value researchbiz.EventFact
		if err := rows.Scan(&value.ID, &value.KnowledgeAvailableAt); err != nil {
			return nil, err
		}
		result[value.ID] = value
	}
	return result, rows.Err()
}

func (t *publicationTransaction) evidences(ctx context.Context, ids []string) (map[string]researchbiz.EvidenceFact, error) {
	rows, err := t.tx.QueryContext(ctx, `SELECT link.id, link.event_id,
       GREATEST(COALESCE(raw.published_at, raw.collected_at), raw.collected_at)
FROM event_evidence_links link
JOIN evidences evidence ON evidence.id = link.evidence_id
JOIN raw_evidences raw ON raw.id = evidence.raw_evidence_id
WHERE link.id = ANY($1::text[])`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]researchbiz.EvidenceFact)
	for rows.Next() {
		var value researchbiz.EvidenceFact
		if err := rows.Scan(&value.ID, &value.EventID, &value.KnowledgeAvailableAt); err != nil {
			return nil, err
		}
		result[value.ID] = value
	}
	return result, rows.Err()
}

func (t *publicationTransaction) InsertThemeReceipt(ctx context.Context, receipt researchbiz.Receipt) error {
	treeIDs, err := json.Marshal(receipt.ReasoningTreeIDsByTreeKey)
	if err != nil {
		return fmt.Errorf("encode Research snapshot Reason Tree IDs: %w", err)
	}
	aggregateCounts, err := json.Marshal(receipt.Counts)
	if err != nil {
		return fmt.Errorf("encode Research aggregate receipt counts: %w", err)
	}
	_, err = t.tx.ExecContext(ctx, `INSERT INTO research_theme_import_receipts (
    id, analysis_batch_id, publisher_subject, payload_hash,
    published_at, imported_at, publication_contract_version,
    aggregate_theme_id, reasoning_tree_ids_by_tree_key,
    aggregate_write_counts, publication_mode
) VALUES ($1,$2,$3,$4,$5,$6,3,$7,$8,$9,'analyst_snapshot')`,
		receipt.ID, receipt.AnalysisBatchID, receipt.PublisherSubject, receipt.PayloadHash,
		receipt.PublishedAt, receipt.ImportedAt, receipt.ThemeID, treeIDs, aggregateCounts,
	)
	return err
}

func (t *publicationTransaction) InsertTheme(ctx context.Context, record researchbiz.PublicationThemeRecord) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO research_themes (
    id, import_receipt_id, analysis_batch_id, theme_key, title, one_line_conclusion,
    conclusion_direction, impact_strength, attention_level, conclusion_status,
    transmission_stage, investment_guidance_action, investment_guidance_summary,
    time_horizon_category, time_horizon_summary, transmission_summary,
    checkpoint_summary, risk_summary, analysis_as_of, window_start, window_end, published_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)`,
		record.ID, record.ImportReceiptID, record.AnalysisBatchID, record.ThemeKey, record.Title,
		record.OneLineConclusion, record.ConclusionDirection, record.ImpactStrength,
		record.AttentionLevel, record.ConclusionStatus, record.TransmissionStage,
		record.InvestmentGuidanceAction, record.InvestmentGuidanceSummary,
		record.TimeHorizonCategory, record.TimeHorizonSummary, record.TransmissionSummary,
		record.CheckpointSummary, record.RiskSummary, record.AnalysisAsOf,
		record.WindowStart, record.WindowEnd, record.PublishedAt)
	return err
}

func (t *publicationTransaction) InsertSnapshotThemeImpact(ctx context.Context, record researchbiz.SnapshotImpactRecord) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO research_theme_impacts (
    theme_id, node_key, display_name, relation_role, impact_direction, impact_summary, display_order
) VALUES ($1,$2,$3,$4,$5,$6,$7)`, record.ThemeID, record.NodeKey, record.DisplayName,
		record.RelationRole, record.ImpactDirection, record.ImpactSummary, record.DisplayOrder)
	return err
}

func (t *publicationTransaction) InsertThemeEvent(ctx context.Context, record researchbiz.PublicationThemeEventRecord) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO research_theme_events (
    theme_id, event_id, evidence_role, supported_claim, evidence_ids
) VALUES ($1,$2,$3,$4,COALESCE($5::text[], '{}'::text[]))`, record.ThemeID, record.EventID,
		record.EvidenceRole, record.SupportedClaim, record.EvidenceIDs)
	return err
}

func (t *publicationTransaction) InsertSnapshotTreeReceipt(ctx context.Context, record researchbiz.SnapshotTreeReceipt) error {
	treeIDs, err := json.Marshal(record.ReasoningTreeIDsByTreeKey)
	if err != nil {
		return fmt.Errorf("encode Research snapshot Reason Tree IDs: %w", err)
	}
	counts, err := json.Marshal(record.Counts)
	if err != nil {
		return fmt.Errorf("encode Research snapshot Reason Tree counts: %w", err)
	}
	_, err = t.tx.ExecContext(ctx, `INSERT INTO research_reasoning_tree_import_receipts (
    id, theme_id, publisher_subject, payload_hash,
    reasoning_tree_ids_by_tree_key, write_counts, published_at, imported_at,
    publication_contract_version, publication_mode
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,3,'analyst_snapshot')`,
		record.ID, record.ThemeID, record.PublisherSubject, record.PayloadHash,
		treeIDs, counts, record.PublishedAt, record.ImportedAt)
	return err
}

func (t *publicationTransaction) InsertSnapshotTree(ctx context.Context, record researchbiz.SnapshotTreeRecord) error {
	if record.InvalidationConditions == nil {
		record.InvalidationConditions = []string{}
	}
	if record.Checkpoints == nil {
		record.Checkpoints = []researchbiz.ReasonTreeCheckpoint{}
	}
	conditions, err := json.Marshal(record.InvalidationConditions)
	if err != nil {
		return err
	}
	checkpoints, err := json.Marshal(record.Checkpoints)
	if err != nil {
		return err
	}
	_, err = t.tx.ExecContext(ctx, `INSERT INTO research_reasoning_trees (
    id, theme_id, import_receipt_id, tree_key, display_name, title, display_order,
    one_line_conclusion, fact_summary, transmission_summary, impact_direction,
    impact_strength, impact_summary, conclusion_boundary_summary, support_summary,
    counter_summary, invalidation_conditions, checkpoints
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
		record.ID, record.ThemeID, record.ImportReceiptID, record.TreeKey, record.DisplayName,
		record.Title, record.DisplayOrder, record.OneLineConclusion, record.FactSummary,
		record.TransmissionSummary, record.ImpactDirection, record.ImpactStrength,
		record.ImpactSummary, record.ConclusionBoundarySummary, record.SupportSummary,
		record.CounterSummary, conditions, checkpoints)
	return err
}

func (t *publicationTransaction) InsertTreeEvent(ctx context.Context, record researchbiz.ReasonTreeEventRecord) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO research_reasoning_tree_events (
    reasoning_tree_id, event_id, evidence_role, display_order, evidence_ids
) VALUES ($1,$2,$3,$4,COALESCE($5::text[], '{}'::text[]))`, record.ReasoningTreeID,
		record.EventID, record.EvidenceRole, record.DisplayOrder, record.EvidenceIDs)
	return err
}

func (t *publicationTransaction) InsertSnapshotNode(ctx context.Context, record researchbiz.SnapshotNodeRecord) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO research_reasoning_tree_nodes (
    id, reasoning_tree_id, position, node_key, display_name, state_summary,
    impact_direction, impact_strength, impact_summary, reasoning_basis_summary,
    evidence_gap_summary, incoming_transmission_title,
    incoming_transmission_mechanism, incoming_condition_summary
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		record.ID, record.ReasoningTreeID, record.Position, record.NodeKey, record.DisplayName,
		record.StateSummary, record.ImpactDirection, record.ImpactStrength, record.ImpactSummary,
		record.ReasoningBasisSummary, record.EvidenceGapSummary, record.IncomingTransmissionTitle,
		record.IncomingTransmissionMechanism, record.IncomingConditionSummary)
	return err
}

func (t *publicationTransaction) InsertSnapshotSignal(ctx context.Context, record researchbiz.SnapshotSignalRecord) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO research_reasoning_tree_node_signals (
    reasoning_tree_node_id, signal_key, variable_name, signal_role, signal_direction,
    display_summary, display_order
) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		record.ReasoningTreeNodeID, record.SignalKey, record.VariableName, record.SignalRole,
		record.SignalDirection, record.DisplaySummary, record.DisplayOrder)
	return err
}

func (t *publicationTransaction) Verify(ctx context.Context, receipt researchbiz.Receipt) error {
	var counts researchbiz.Counts
	if err := t.tx.QueryRowContext(ctx, `SELECT
    (SELECT count(*) FROM research_themes WHERE import_receipt_id = $1),
    (SELECT count(*) FROM research_theme_impacts WHERE theme_id = $2),
    (SELECT count(*) FROM research_theme_events WHERE theme_id = $2),
    (SELECT count(*) FROM research_reasoning_trees WHERE theme_id = $2),
    (SELECT count(*) FROM research_reasoning_tree_nodes node
       JOIN research_reasoning_trees tree ON tree.id = node.reasoning_tree_id WHERE tree.theme_id = $2),
    (SELECT count(*) FROM research_reasoning_tree_events event
       JOIN research_reasoning_trees tree ON tree.id = event.reasoning_tree_id WHERE tree.theme_id = $2),
    (SELECT count(*) FROM research_reasoning_tree_node_signals signal
       JOIN research_reasoning_tree_nodes node ON node.id = signal.reasoning_tree_node_id
       JOIN research_reasoning_trees tree ON tree.id = node.reasoning_tree_id WHERE tree.theme_id = $2)`,
		receipt.ID, receipt.ThemeID,
	).Scan(&counts.Themes, &counts.Impacts, &counts.ThemeEventAssociations,
		&counts.ReasoningTrees, &counts.Nodes, &counts.TreeEventAssociations,
		&counts.SignalAssociations); err != nil {
		return err
	}
	counts.Receipts = 2
	if counts != receipt.Counts {
		return fmt.Errorf("aggregate write counts do not match persisted rows")
	}
	var treeIDsJSON []byte
	if err := t.tx.QueryRowContext(ctx, `SELECT COALESCE(jsonb_object_agg(tree_key, id::text), '{}'::jsonb)
FROM research_reasoning_trees WHERE theme_id = $1`, receipt.ThemeID).Scan(&treeIDsJSON); err != nil {
		return err
	}
	var treeIDs map[string]string
	if err := json.Unmarshal(treeIDsJSON, &treeIDs); err != nil {
		return err
	}
	if len(treeIDs) != len(receipt.ReasoningTreeIDsByTreeKey) {
		return fmt.Errorf("aggregate Reason Tree identities do not match persisted rows")
	}
	for key, id := range receipt.ReasoningTreeIDsByTreeKey {
		if treeIDs[key] != id {
			return fmt.Errorf("aggregate Reason Tree identity mismatch")
		}
	}
	return nil
}
