package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

func (r repository) InResearchReasoningTreeImportTransaction(ctx context.Context, fn func(ResearchReasoningTreeImportTransaction) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin Research Reason Tree import transaction: %w", err)
	}
	wrapper := &postgresResearchReasoningTreeImportTx{tx: tx}
	if err := fn(wrapper); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit Research Reason Tree import transaction: %w", err)
	}
	return nil
}

type postgresResearchReasoningTreeImportTx struct{ tx *sql.Tx }

func (t *postgresResearchReasoningTreeImportTx) LockResearchReasoningTreeImportTheme(ctx context.Context, themeID string) error {
	if _, err := t.tx.ExecContext(ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, "research-reasoning-tree:"+themeID); err != nil {
		return fmt.Errorf("lock Research Reason Tree Theme %q: %w", themeID, err)
	}
	return nil
}

func (t *postgresResearchReasoningTreeImportTx) LockResearchReasoningTreeAnalysisBatch(ctx context.Context, analysisBatchID string) error {
	if _, err := t.tx.ExecContext(ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, "research-reasoning-tree-batch:"+analysisBatchID); err != nil {
		return fmt.Errorf("lock Research Reason Tree analysis batch %q: %w", analysisBatchID, err)
	}
	return nil
}

func (t *postgresResearchReasoningTreeImportTx) ResearchReasoningTreeImportReceipt(ctx context.Context, themeID string) (*ResearchReasoningTreeImportReceipt, error) {
	var receipt ResearchReasoningTreeImportReceipt
	var treeIDsJSON, countsJSON []byte
	err := t.tx.QueryRowContext(ctx, `SELECT id, theme_id, publisher_subject, payload_hash,
       reasoning_tree_ids_by_industry_chain_entity_id, write_counts, published_at, imported_at
FROM research_reasoning_tree_import_receipts WHERE theme_id = $1`, themeID).Scan(
		&receipt.ID, &receipt.ThemeID, &receipt.PublisherSubject, &receipt.PayloadHash,
		&treeIDsJSON, &countsJSON, &receipt.PublishedAt, &receipt.ImportedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read Research Reason Tree import receipt: %w", err)
	}
	if err := json.Unmarshal(treeIDsJSON, &receipt.ReasoningTreeIDsByIndustryChainEntityID); err != nil {
		return nil, fmt.Errorf("decode receipt Reason Tree mapping: %w", err)
	}
	if err := json.Unmarshal(countsJSON, &receipt.Counts); err != nil {
		return nil, fmt.Errorf("decode receipt write counts: %w", err)
	}
	if len(receipt.ReasoningTreeIDsByIndustryChainEntityID) == 0 ||
		receipt.Counts.ReasoningTrees != len(receipt.ReasoningTreeIDsByIndustryChainEntityID) {
		return nil, fmt.Errorf("receipt Reason Tree mapping does not match its Tree count")
	}
	return &receipt, nil
}

func (t *postgresResearchReasoningTreeImportTx) ResearchReasoningTreeImportThemePublication(ctx context.Context, themeID string) (*ResearchReasoningTreeImportThemePublication, error) {
	var publication ResearchReasoningTreeImportThemePublication
	err := t.tx.QueryRowContext(ctx, `SELECT theme.id::text, theme.analysis_batch_id,
       theme.import_receipt_id::text, receipt.publisher_subject
FROM research_themes theme
JOIN research_theme_import_receipts receipt ON receipt.id = theme.import_receipt_id
WHERE theme.id = $1`, themeID).Scan(
		&publication.ThemeID, &publication.AnalysisBatchID,
		&publication.ThemeImportReceiptID, &publication.PublisherSubject,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read parent Theme publication: %w", err)
	}
	publication.ImpactNodeIDs, err = queryResearchSet(ctx, t.tx,
		`SELECT chain_node_entity_id::text FROM research_theme_impacts WHERE theme_id = $1`, themeID)
	if err != nil {
		return nil, fmt.Errorf("read parent Theme Impacts: %w", err)
	}
	publication.EventIDs, err = queryResearchSet(ctx, t.tx,
		`SELECT event_id::text FROM research_theme_events WHERE theme_id = $1`, themeID)
	if err != nil {
		return nil, fmt.Errorf("read parent Theme Events: %w", err)
	}
	return &publication, nil
}

func (t *postgresResearchReasoningTreeImportTx) ResearchReasoningTreeSignalSnapshots(
	ctx context.Context,
	analysisBatchID string,
	keys []string,
) (map[string]ResearchReasoningTreeImportSignalSnapshot, error) {
	rows, err := t.tx.QueryContext(ctx, `SELECT signal.variable_signal_key,
       signal.signal_direction, signal.display_summary
FROM research_reasoning_tree_node_signals signal
JOIN research_reasoning_tree_nodes node ON node.id = signal.reasoning_tree_node_id
JOIN research_reasoning_trees tree ON tree.id = node.reasoning_tree_id
JOIN research_themes theme ON theme.id = tree.theme_id
WHERE theme.analysis_batch_id = $1
  AND signal.variable_signal_key = ANY($2::text[])`, analysisBatchID, keys)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]ResearchReasoningTreeImportSignalSnapshot, len(keys))
	for rows.Next() {
		var key string
		var snapshot ResearchReasoningTreeImportSignalSnapshot
		if err := rows.Scan(&key, &snapshot.SignalDirection, &snapshot.DisplaySummary); err != nil {
			return nil, err
		}
		if prior, exists := result[key]; exists && prior != snapshot {
			return nil, fmt.Errorf("persisted Variable Signal snapshot %q is inconsistent within analysis batch", key)
		}
		result[key] = snapshot
	}
	return result, rows.Err()
}

func (t *postgresResearchReasoningTreeImportTx) ExistingResearchReasoningTreeIndustryChains(ctx context.Context, ids []string) (map[string]struct{}, error) {
	return queryResearchSet(ctx, t.tx, `SELECT definition.entity_id::text
FROM industry_chain_definitions definition
JOIN entity_nodes node ON node.id = definition.entity_id
WHERE definition.entity_id = ANY($1::uuid[])
  AND definition.review_status = 'approved'
  AND node.status = 'active'`, ids)
}

func (t *postgresResearchReasoningTreeImportTx) ResearchReasoningTreeChainMemberships(ctx context.Context, ids []string) (map[string]map[string]struct{}, error) {
	rows, err := t.tx.QueryContext(ctx, `SELECT membership.industry_chain_entity_id::text,
       membership.chain_node_entity_id::text
FROM industry_chain_node_memberships membership
JOIN entity_nodes node ON node.id = membership.chain_node_entity_id
JOIN chain_node_profiles profile ON profile.entity_id = membership.chain_node_entity_id
WHERE membership.industry_chain_entity_id = ANY($1::uuid[])
  AND membership.review_status = 'approved'
  AND membership.status = 'active'
  AND node.status = 'active'
  AND profile.review_status = 'approved'`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]map[string]struct{}, len(ids))
	for rows.Next() {
		var chainID, nodeID string
		if err := rows.Scan(&chainID, &nodeID); err != nil {
			return nil, err
		}
		if result[chainID] == nil {
			result[chainID] = make(map[string]struct{})
		}
		result[chainID][nodeID] = struct{}{}
	}
	return result, rows.Err()
}

func (t *postgresResearchReasoningTreeImportTx) ResearchReasoningTreeGraphEdges(ctx context.Context, ids []string) (map[string]ResearchReasoningTreeImportGraphEdge, error) {
	rows, err := t.tx.QueryContext(ctx, `SELECT id::text, industry_chain_entity_id::text,
       from_chain_node_entity_id::text, to_chain_node_entity_id::text
FROM industry_chain_graph_edges
WHERE id = ANY($1::uuid[]) AND review_status = 'approved' AND status = 'active'`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]ResearchReasoningTreeImportGraphEdge, len(ids))
	for rows.Next() {
		var edge ResearchReasoningTreeImportGraphEdge
		if err := rows.Scan(&edge.ID, &edge.IndustryChainEntityID,
			&edge.FromChainNodeEntityID, &edge.ToChainNodeEntityID); err != nil {
			return nil, err
		}
		result[edge.ID] = edge
	}
	return result, rows.Err()
}

func queryResearchSet(ctx context.Context, tx *sql.Tx, query string, argument any) (map[string]struct{}, error) {
	rows, err := tx.QueryContext(ctx, query, argument)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]struct{})
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result[id] = struct{}{}
	}
	return result, rows.Err()
}

func (t *postgresResearchReasoningTreeImportTx) InsertResearchReasoningTreeImportReceipt(ctx context.Context, receipt ResearchReasoningTreeImportReceipt) error {
	treeIDs, err := json.Marshal(receipt.ReasoningTreeIDsByIndustryChainEntityID)
	if err != nil {
		return err
	}
	counts, err := json.Marshal(receipt.Counts)
	if err != nil {
		return err
	}
	_, err = t.tx.ExecContext(ctx, `INSERT INTO research_reasoning_tree_import_receipts (
    id, theme_id, publisher_subject, payload_hash,
    reasoning_tree_ids_by_industry_chain_entity_id, write_counts, published_at, imported_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		receipt.ID, receipt.ThemeID, receipt.PublisherSubject, receipt.PayloadHash,
		treeIDs, counts, receipt.PublishedAt, receipt.ImportedAt)
	return err
}

func (t *postgresResearchReasoningTreeImportTx) InsertResearchReasoningTree(ctx context.Context, tree ResearchReasoningTreeImportTree) error {
	conditions, err := json.Marshal(tree.InvalidationConditions)
	if err != nil {
		return err
	}
	checkpoints, err := json.Marshal(tree.Checkpoints)
	if err != nil {
		return err
	}
	_, err = t.tx.ExecContext(ctx, `INSERT INTO research_reasoning_trees (
    id, theme_id, import_receipt_id, industry_chain_entity_id, title, display_order,
    one_line_conclusion, fact_summary, transmission_summary, impact_direction,
    impact_strength, impact_summary, conclusion_boundary_summary, support_summary,
    counter_summary, invalidation_conditions, checkpoints
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		tree.ID, tree.ThemeID, tree.ImportReceiptID, tree.IndustryChainEntityID,
		tree.Title, tree.DisplayOrder, tree.OneLineConclusion, tree.FactSummary,
		tree.TransmissionSummary, tree.ImpactDirection, tree.ImpactStrength,
		tree.ImpactSummary, tree.ConclusionBoundarySummary, tree.SupportSummary,
		tree.CounterSummary, conditions, checkpoints)
	return err
}

func (t *postgresResearchReasoningTreeImportTx) InsertResearchReasoningTreeEvent(ctx context.Context, event ResearchReasoningTreeImportEvent) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO research_reasoning_tree_events (
    reasoning_tree_id, event_id, evidence_role, display_order, evidence_ids
) VALUES ($1,$2,$3,$4,COALESCE($5::uuid[], '{}'::uuid[]))`, event.ReasoningTreeID, event.EventID, event.EvidenceRole, event.DisplayOrder, event.EvidenceIDs)
	return err
}

func (t *postgresResearchReasoningTreeImportTx) InsertResearchReasoningTreeNode(ctx context.Context, node ResearchReasoningTreeImportNode) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO research_reasoning_tree_nodes (
    id, reasoning_tree_id, position, chain_node_entity_id, state_summary,
    impact_direction, impact_strength, impact_summary, reasoning_basis_summary,
    evidence_gap_summary, incoming_industry_chain_graph_edge_id,
    incoming_transmission_title, incoming_transmission_mechanism, incoming_condition_summary
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		node.ID, node.ReasoningTreeID, node.Position, node.ChainNodeEntityID,
		node.StateSummary, node.ImpactDirection, node.ImpactStrength, node.ImpactSummary,
		node.ReasoningBasisSummary, node.EvidenceGapSummary,
		node.IncomingIndustryChainGraphEdgeID, node.IncomingTransmissionTitle,
		node.IncomingTransmissionMechanism, node.IncomingConditionSummary)
	return err
}

func (t *postgresResearchReasoningTreeImportTx) InsertResearchReasoningTreeNodeSignal(ctx context.Context, signal ResearchReasoningTreeImportSignal) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO research_reasoning_tree_node_signals (
    reasoning_tree_node_id, variable_signal_key, signal_role, signal_direction,
    display_summary, display_order
) VALUES ($1,$2,$3,$4,$5,$6)`, signal.ReasoningTreeNodeID, signal.VariableSignalKey,
		signal.SignalRole, signal.SignalDirection, signal.DisplaySummary, signal.DisplayOrder)
	return err
}

func (t *postgresResearchReasoningTreeImportTx) VerifyResearchReasoningTreeImportReceipt(ctx context.Context, receipt ResearchReasoningTreeImportReceipt) error {
	var treeIDsJSON []byte
	if err := t.tx.QueryRowContext(ctx, `SELECT COALESCE(
    jsonb_object_agg(industry_chain_entity_id::text, id::text), '{}'::jsonb)
FROM research_reasoning_trees WHERE import_receipt_id = $1`, receipt.ID).Scan(&treeIDsJSON); err != nil {
		return fmt.Errorf("verify receipt Reason Tree IDs: %w", err)
	}
	var treeIDs map[string]string
	if err := json.Unmarshal(treeIDsJSON, &treeIDs); err != nil {
		return err
	}
	if !reflect.DeepEqual(treeIDs, receipt.ReasoningTreeIDsByIndustryChainEntityID) {
		return fmt.Errorf("receipt Reason Tree IDs are not all present")
	}
	var counts ResearchReasoningTreeImportCounts
	if err := t.tx.QueryRowContext(ctx, `SELECT
    (SELECT count(*) FROM research_reasoning_trees WHERE import_receipt_id = $1),
    (SELECT count(*) FROM research_reasoning_tree_nodes n
       JOIN research_reasoning_trees t ON t.id = n.reasoning_tree_id WHERE t.import_receipt_id = $1),
    (SELECT count(*) FROM research_reasoning_tree_events e
       JOIN research_reasoning_trees t ON t.id = e.reasoning_tree_id WHERE t.import_receipt_id = $1),
    (SELECT count(*) FROM research_reasoning_tree_node_signals s
       JOIN research_reasoning_tree_nodes n ON n.id = s.reasoning_tree_node_id
       JOIN research_reasoning_trees t ON t.id = n.reasoning_tree_id WHERE t.import_receipt_id = $1)`,
		receipt.ID).Scan(&counts.ReasoningTrees, &counts.Nodes,
		&counts.EventAssociations, &counts.SignalAssociations); err != nil {
		return fmt.Errorf("verify receipt write counts: %w", err)
	}
	counts.Receipts = 1
	if counts != receipt.Counts {
		return fmt.Errorf("receipt write counts do not match persisted rows")
	}
	return nil
}
