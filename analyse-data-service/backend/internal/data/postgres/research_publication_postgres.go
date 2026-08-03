package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchpublication"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchreasoningtreeimport"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchthemeimport"
)

func (r repository) InResearchPublicationTransaction(
	ctx context.Context,
	fn func(researchpublication.Transaction) error,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin research publication transaction: %w", err)
	}
	wrapper := &postgresResearchPublicationTx{tx: tx}
	if err := fn(wrapper); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit research publication transaction: %w", err)
	}
	return nil
}

type postgresResearchPublicationTx struct{ tx *sql.Tx }

func (t *postgresResearchPublicationTx) Lock(ctx context.Context, analysisBatchID string) error {
	_, err := t.tx.ExecContext(ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`,
		"research-publication-v2:"+analysisBatchID,
	)
	return err
}

func (t *postgresResearchPublicationTx) Receipt(ctx context.Context, analysisBatchID string) (*researchpublication.Receipt, error) {
	var receipt researchpublication.Receipt
	var treeIDsJSON, treeKeyIDsJSON, countsJSON []byte
	err := t.tx.QueryRowContext(ctx, `SELECT
    id::text, analysis_batch_id, publisher_subject, payload_hash,
    publication_contract_version, publication_mode, COALESCE(aggregate_theme_id::text, ''),
    reasoning_tree_ids_by_industry_chain_entity_id, reasoning_tree_ids_by_tree_key, aggregate_write_counts,
    published_at, imported_at
FROM research_theme_import_receipts
WHERE analysis_batch_id = $1`, analysisBatchID).Scan(
		&receipt.ID, &receipt.AnalysisBatchID, &receipt.PublisherSubject, &receipt.PayloadHash,
		&receipt.ContractVersion, &receipt.PublicationMode, &receipt.ThemeID, &treeIDsJSON, &treeKeyIDsJSON, &countsJSON,
		&receipt.PublishedAt, &receipt.ImportedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(treeIDsJSON, &receipt.ReasoningTreeIDsByIndustryChainEntityID); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(treeKeyIDsJSON, &receipt.ReasoningTreeIDsByTreeKey); err != nil {
		return nil, err
	}
	if receipt.ContractVersion >= 2 {
		if err := json.Unmarshal(countsJSON, &receipt.Counts); err != nil {
			return nil, err
		}
		if err := t.tx.QueryRowContext(ctx,
			`SELECT theme_key FROM research_themes WHERE id = $1`, receipt.ThemeID,
		).Scan(&receipt.ThemeKey); err != nil {
			return nil, err
		}
	}
	return &receipt, nil
}

func (t *postgresResearchPublicationTx) ReferenceFacts(
	ctx context.Context,
	query researchpublication.ReferenceQuery,
) (researchpublication.ReferenceFacts, error) {
	facts := researchpublication.ReferenceFacts{}
	var err error
	facts.ChainNodeIDs, err = queryResearchPublicationTemporalFacts(ctx, t.tx, `SELECT profile.entity_id::text,
       node.created_at, node.updated_at
FROM chain_node_profiles profile
JOIN entity_nodes node ON node.id = profile.entity_id
WHERE profile.entity_id = ANY($1::uuid[])
  AND node.status = 'active'
  AND profile.review_status = 'approved'`, query.ChainNodeIDs)
	if err != nil {
		return facts, err
	}
	facts.Events, err = t.events(ctx, query.EventIDs, query.SnapshotEventExistenceOnly)
	if err != nil {
		return facts, err
	}
	facts.IndustryChainIDs, err = queryResearchPublicationTemporalFacts(ctx, t.tx, `SELECT definition.entity_id::text,
       GREATEST(node.created_at, definition.created_at),
       GREATEST(node.updated_at, definition.updated_at)
FROM industry_chain_definitions definition
JOIN entity_nodes node ON node.id = definition.entity_id
WHERE definition.entity_id = ANY($1::uuid[])
  AND definition.review_status = 'approved'
  AND node.status = 'active'`, query.IndustryChainIDs)
	if err != nil {
		return facts, err
	}
	facts.Memberships, err = t.memberships(ctx, query.IndustryChainIDs)
	if err != nil {
		return facts, err
	}
	facts.GraphEdges, err = t.graphEdges(ctx, query.GraphEdgeIDs)
	if err != nil {
		return facts, err
	}
	facts.Signals, err = t.signals(ctx, query.SignalIDs)
	if err != nil {
		return facts, err
	}
	facts.Impacts, err = t.impacts(ctx, query.ImpactIDs)
	if err != nil {
		return facts, err
	}
	facts.Evidences, err = t.evidences(ctx, query.EvidenceIDs)
	if err != nil {
		return facts, err
	}
	facts.EntityRelations, err = t.entityRelations(ctx, query.EntityRelationIDs)
	return facts, err
}

func queryResearchPublicationTemporalFacts(
	ctx context.Context,
	tx *sql.Tx,
	statement string,
	ids []string,
) (map[string]researchpublication.TemporalFact, error) {
	rows, err := tx.QueryContext(ctx, statement, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]researchpublication.TemporalFact, len(ids))
	for rows.Next() {
		var id string
		var temporal researchpublication.TemporalFact
		if err := rows.Scan(&id, &temporal.CreatedAt, &temporal.UpdatedAt); err != nil {
			return nil, err
		}
		result[id] = temporal
	}
	return result, rows.Err()
}

func (t *postgresResearchPublicationTx) events(
	ctx context.Context,
	ids []string,
	existenceOnly bool,
) (map[string]researchpublication.EventFact, error) {
	statement := `SELECT id::text,
       COALESCE(knowable_at, first_seen_at)
FROM events
WHERE id = ANY($1::uuid[])
  AND event_status = 'confirmed'
	  AND fact_status = 'verified'`
	if existenceOnly {
		statement = `SELECT id::text, COALESCE(knowable_at, first_seen_at)
FROM events WHERE id = ANY($1::uuid[])`
	}
	rows, err := t.tx.QueryContext(ctx, statement, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]researchpublication.EventFact)
	for rows.Next() {
		var value researchpublication.EventFact
		if err := rows.Scan(&value.ID, &value.KnowledgeAvailableAt); err != nil {
			return nil, err
		}
		result[value.ID] = value
	}
	return result, rows.Err()
}

func (t *postgresResearchPublicationTx) entityRelations(
	ctx context.Context,
	ids []string,
) (map[string]researchpublication.EntityRelationFact, error) {
	rows, err := t.tx.QueryContext(ctx, `SELECT edge.id::text,
       edge.from_entity_id::text, edge.to_entity_id::text,
       GREATEST(edge.created_at, source.created_at, target.created_at),
       GREATEST(edge.updated_at, source.updated_at, target.updated_at)
FROM entity_edges edge
JOIN entity_nodes source ON source.id = edge.from_entity_id AND source.status = 'active'
JOIN entity_nodes target ON target.id = edge.to_entity_id AND target.status = 'active'
WHERE edge.id = ANY($1::uuid[]) AND edge.status = 'active'`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]researchpublication.EntityRelationFact)
	for rows.Next() {
		var value researchpublication.EntityRelationFact
		if err := rows.Scan(
			&value.ID, &value.FromEntityID, &value.ToEntityID,
			&value.CreatedAt, &value.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result[value.ID] = value
	}
	return result, rows.Err()
}

func (t *postgresResearchPublicationTx) memberships(
	ctx context.Context,
	chainIDs []string,
) (map[string]map[string]researchpublication.TemporalFact, error) {
	rows, err := t.tx.QueryContext(ctx, `SELECT membership.industry_chain_entity_id::text,
       membership.chain_node_entity_id::text,
       GREATEST(membership.created_at, node.created_at),
       GREATEST(membership.updated_at, node.updated_at)
FROM industry_chain_node_memberships membership
JOIN entity_nodes node ON node.id = membership.chain_node_entity_id
JOIN chain_node_profiles profile ON profile.entity_id = membership.chain_node_entity_id
WHERE membership.industry_chain_entity_id = ANY($1::uuid[])
  AND membership.status = 'active' AND membership.review_status = 'approved'
  AND node.status = 'active' AND profile.review_status = 'approved'`, chainIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]map[string]researchpublication.TemporalFact)
	for rows.Next() {
		var chainID, nodeID string
		var temporal researchpublication.TemporalFact
		if err := rows.Scan(&chainID, &nodeID, &temporal.CreatedAt, &temporal.UpdatedAt); err != nil {
			return nil, err
		}
		if result[chainID] == nil {
			result[chainID] = make(map[string]researchpublication.TemporalFact)
		}
		result[chainID][nodeID] = temporal
	}
	return result, rows.Err()
}

func (t *postgresResearchPublicationTx) graphEdges(
	ctx context.Context,
	ids []string,
) (map[string]researchpublication.GraphEdgeFact, error) {
	rows, err := t.tx.QueryContext(ctx, `SELECT id::text, industry_chain_entity_id::text,
       from_chain_node_entity_id::text, to_chain_node_entity_id::text,
       created_at, updated_at
FROM industry_chain_graph_edges
WHERE id = ANY($1::uuid[]) AND status = 'active' AND review_status = 'approved'`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]researchpublication.GraphEdgeFact)
	for rows.Next() {
		var value researchpublication.GraphEdgeFact
		if err := rows.Scan(
			&value.ID, &value.IndustryChainEntityID,
			&value.FromChainNodeEntityID, &value.ToChainNodeEntityID,
			&value.CreatedAt, &value.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result[value.ID] = value
	}
	return result, rows.Err()
}

func (t *postgresResearchPublicationTx) signals(ctx context.Context, ids []string) (map[string]researchpublication.SignalFact, error) {
	rows, err := t.tx.QueryContext(ctx, `SELECT signal.id::text, signal.semantic_submission_id::text,
       signal.source_event_id::text, link.entity_id::text, signal.variable_key, signal.direction,
       to_json(signal.evidence_ids), COALESCE(submission.finalized_at, submission.created_at)
FROM variable_signals signal
JOIN event_semantic_submissions submission
  ON submission.id = signal.semantic_submission_id AND submission.status = 'accepted'
JOIN event_entity_links link
  ON link.id = signal.subject_event_entity_link_id
 AND link.review_status = 'accepted'
 AND link.semantic_submission_id = signal.semantic_submission_id
WHERE signal.id = ANY($1::uuid[]) AND signal.review_status = 'accepted'`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]researchpublication.SignalFact)
	for rows.Next() {
		var value researchpublication.SignalFact
		var evidenceJSON []byte
		if err := rows.Scan(&value.ID, &value.SemanticSubmissionID, &value.EventID,
			&value.SubjectEntityID, &value.VariableKey, &value.Direction,
			&evidenceJSON, &value.AcceptedAt); err != nil {
			return nil, err
		}
		value.EvidenceIDs, err = decodeIDSet(evidenceJSON)
		if err != nil {
			return nil, err
		}
		result[value.ID] = value
	}
	return result, rows.Err()
}

func (t *postgresResearchPublicationTx) impacts(ctx context.Context, ids []string) (map[string]researchpublication.ImpactFact, error) {
	rows, err := t.tx.QueryContext(ctx, `SELECT impact.id::text, impact.semantic_submission_id::text,
       impact.source_variable_signal_id::text, impact.target_entity_id::text,
       impact.affected_variable_key, impact.affected_direction, signal.source_event_id::text,
       source_link.entity_id::text,
       to_json(impact.evidence_ids), COALESCE(submission.finalized_at, submission.created_at)
FROM direct_impact_assertions impact
JOIN event_semantic_submissions submission
  ON submission.id = impact.semantic_submission_id AND submission.status = 'accepted'
JOIN variable_signals signal
  ON signal.id = impact.source_variable_signal_id
 AND signal.review_status = 'accepted'
 AND signal.semantic_submission_id = impact.semantic_submission_id
JOIN event_entity_links source_link
  ON source_link.id = signal.subject_event_entity_link_id
 AND source_link.semantic_submission_id = signal.semantic_submission_id
 AND source_link.review_status = 'accepted'
LEFT JOIN direct_transmission_rules rule
  ON rule.rule_key = impact.rule_key AND rule.version = impact.rule_version
LEFT JOIN entity_edges relation ON relation.id = impact.entity_relation_id
WHERE impact.id = ANY($1::uuid[])
  AND impact.review_status = 'accepted'
  AND (
      impact.derivation_type = 'event_explicit'
      OR (
          impact.derivation_type = 'rule_inferred'
          AND rule.status = 'approved'
          AND relation.status = 'active'
      )
  )`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]researchpublication.ImpactFact)
	for rows.Next() {
		var value researchpublication.ImpactFact
		var evidenceJSON []byte
		if err := rows.Scan(&value.ID, &value.SemanticSubmissionID,
			&value.SourceVariableSignalID, &value.TargetEntityID,
			&value.AffectedVariableKey, &value.AffectedDirection, &value.SourceEventID,
			&value.SourceEntityID,
			&evidenceJSON, &value.AcceptedAt); err != nil {
			return nil, err
		}
		value.EvidenceIDs, err = decodeIDSet(evidenceJSON)
		if err != nil {
			return nil, err
		}
		result[value.ID] = value
	}
	return result, rows.Err()
}

func (t *postgresResearchPublicationTx) evidences(ctx context.Context, ids []string) (map[string]researchpublication.EvidenceFact, error) {
	rows, err := t.tx.QueryContext(ctx, `SELECT source.id::text, source.event_id::text,
       source.evidence_hash,
       GREATEST(COALESCE(document.published_at, document.collected_at), document.collected_at)
FROM event_sources source
JOIN raw_documents document ON document.id = source.raw_document_id
WHERE source.id = ANY($1::uuid[])`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]researchpublication.EvidenceFact)
	for rows.Next() {
		var value researchpublication.EvidenceFact
		if err := rows.Scan(&value.ID, &value.EventID, &value.Hash, &value.KnowledgeAvailableAt); err != nil {
			return nil, err
		}
		result[value.ID] = value
	}
	return result, rows.Err()
}

func decodeIDSet(raw []byte) (map[string]struct{}, error) {
	var ids []string
	if err := json.Unmarshal(raw, &ids); err != nil {
		return nil, err
	}
	result := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		result[id] = struct{}{}
	}
	return result, nil
}

func (t *postgresResearchPublicationTx) InsertThemeReceipt(ctx context.Context, receipt researchpublication.Receipt) error {
	themeIDs, _ := json.Marshal(map[string]string{receipt.ThemeKey: receipt.ThemeID})
	treeIDs, _ := json.Marshal(receipt.ReasoningTreeIDsByIndustryChainEntityID)
	treeKeyIDs, _ := json.Marshal(cloneStringMapOrEmpty(receipt.ReasoningTreeIDsByTreeKey))
	legacyCounts, _ := json.Marshal(researchthemeimport.Counts{
		Themes: 1, Impacts: receipt.Counts.Impacts,
		EventAssociations: receipt.Counts.ThemeEventAssociations, Receipts: 1,
	})
	aggregateCounts, _ := json.Marshal(receipt.Counts)
	_, err := t.tx.ExecContext(ctx, `INSERT INTO research_theme_import_receipts (
    id, analysis_batch_id, publisher_subject, payload_hash, theme_ids_by_key,
    write_counts, published_at, imported_at, publication_contract_version,
    aggregate_theme_id, reasoning_tree_ids_by_industry_chain_entity_id,
    reasoning_tree_ids_by_tree_key, aggregate_write_counts, publication_mode
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		receipt.ID, receipt.AnalysisBatchID, receipt.PublisherSubject, receipt.PayloadHash,
		themeIDs, legacyCounts, receipt.PublishedAt, receipt.ImportedAt,
		receipt.ContractVersion, receipt.ThemeID, treeIDs, treeKeyIDs, aggregateCounts, receipt.PublicationMode,
	)
	return err
}

func cloneStringMapOrEmpty(values map[string]string) map[string]string {
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func (t *postgresResearchPublicationTx) InsertTheme(ctx context.Context, record researchthemeimport.ThemeRecord) error {
	return (&postgresResearchThemeImportTx{tx: t.tx}).InsertResearchTheme(ctx, record)
}
func (t *postgresResearchPublicationTx) InsertThemeImpact(ctx context.Context, record researchthemeimport.ImpactRecord) error {
	return (&postgresResearchThemeImportTx{tx: t.tx}).InsertResearchThemeImpact(ctx, record)
}
func (t *postgresResearchPublicationTx) InsertSnapshotThemeImpact(ctx context.Context, record researchpublication.SnapshotImpactRecord) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO research_theme_impacts (
    theme_id, node_key, display_name, relation_role, impact_direction, impact_summary, display_order
) VALUES ($1,$2,$3,$4,$5,$6,$7)`, record.ThemeID, record.NodeKey, record.DisplayName,
		record.RelationRole, record.ImpactDirection, record.ImpactSummary, record.DisplayOrder)
	return err
}
func (t *postgresResearchPublicationTx) InsertThemeEvent(ctx context.Context, record researchthemeimport.EventRecord) error {
	return (&postgresResearchThemeImportTx{tx: t.tx}).InsertResearchThemeEvent(ctx, record)
}
func (t *postgresResearchPublicationTx) InsertTreeReceipt(ctx context.Context, record researchreasoningtreeimport.Receipt) error {
	return (&postgresResearchReasoningTreeImportTx{tx: t.tx}).InsertResearchReasoningTreeImportReceipt(ctx, record)
}
func (t *postgresResearchPublicationTx) InsertSnapshotTreeReceipt(ctx context.Context, record researchpublication.SnapshotTreeReceipt) error {
	treeIDs, _ := json.Marshal(record.ReasoningTreeIDsByTreeKey)
	counts, _ := json.Marshal(record.Counts)
	_, err := t.tx.ExecContext(ctx, `INSERT INTO research_reasoning_tree_import_receipts (
    id, theme_id, publisher_subject, payload_hash,
    reasoning_tree_ids_by_industry_chain_entity_id, reasoning_tree_ids_by_tree_key,
    write_counts, published_at, imported_at, publication_contract_version, publication_mode
) VALUES ($1,$2,$3,$4,'{}'::jsonb,$5,$6,$7,$8,3,'analyst_snapshot')`,
		record.ID, record.ThemeID, record.PublisherSubject, record.PayloadHash,
		treeIDs, counts, record.PublishedAt, record.ImportedAt)
	return err
}
func (t *postgresResearchPublicationTx) InsertTree(ctx context.Context, record researchreasoningtreeimport.ReasoningTreeRecord) error {
	return (&postgresResearchReasoningTreeImportTx{tx: t.tx}).InsertResearchReasoningTree(ctx, record)
}
func (t *postgresResearchPublicationTx) InsertSnapshotTree(ctx context.Context, record researchpublication.SnapshotTreeRecord) error {
	if record.InvalidationConditions == nil {
		record.InvalidationConditions = []string{}
	}
	if record.Checkpoints == nil {
		record.Checkpoints = []researchreasoningtreeimport.Checkpoint{}
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
func (t *postgresResearchPublicationTx) InsertTreeEvent(ctx context.Context, record researchreasoningtreeimport.EventRecord) error {
	return (&postgresResearchReasoningTreeImportTx{tx: t.tx}).InsertResearchReasoningTreeEvent(ctx, record)
}

func (t *postgresResearchPublicationTx) InsertNode(ctx context.Context, record researchpublication.NodeRecord) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO research_reasoning_tree_nodes (
    id, reasoning_tree_id, position, chain_node_entity_id, state_summary,
    impact_direction, impact_strength, impact_summary, reasoning_basis_summary,
    evidence_gap_summary, incoming_industry_chain_graph_edge_id,
    incoming_transmission_title, incoming_transmission_mechanism, incoming_condition_summary,
    incoming_source_kind, direct_impact_assertion_id, direct_impact_semantic_submission_id,
    direct_impact_evidence_id, direct_impact_evidence_hash,
    direct_impact_affected_variable_key, direct_impact_affected_direction,
    inference_upstream_variable_signal_id, inference_upstream_direct_impact_assertion_id,
    inference_entity_relation_id
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24)`,
		record.ID, record.ReasoningTreeID, record.Position, record.ChainNodeEntityID,
		record.StateSummary, record.ImpactDirection, record.ImpactStrength, record.ImpactSummary,
		record.ReasoningBasisSummary, record.EvidenceGapSummary,
		record.IncomingIndustryChainGraphEdgeID, record.IncomingTransmissionTitle,
		record.IncomingTransmissionMechanism, record.IncomingConditionSummary,
		record.IncomingSourceKind, record.DirectImpactAssertionID,
		record.DirectImpactSemanticSubmissionID, record.DirectImpactEvidenceID,
		record.DirectImpactEvidenceHash, record.DirectImpactAffectedVariableKey,
		record.DirectImpactAffectedDirection, record.InferenceUpstreamVariableSignalID,
		record.InferenceUpstreamDirectImpactAssertionID, record.InferenceEntityRelationID,
	)
	return err
}

func (t *postgresResearchPublicationTx) InsertSnapshotNode(ctx context.Context, record researchpublication.SnapshotNodeRecord) error {
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

func (t *postgresResearchPublicationTx) InsertSignal(ctx context.Context, record researchpublication.SignalRecord) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO research_reasoning_tree_node_signals (
    reasoning_tree_node_id, variable_signal_key, signal_role, signal_direction,
    display_summary, display_order, source_kind, variable_signal_id,
    semantic_submission_id, evidence_id, evidence_hash, upstream_variable_signal_id,
    upstream_direct_impact_assertion_id, entity_relation_id, industry_chain_graph_edge_id
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		record.ReasoningTreeNodeID, record.VariableSignalKey, record.SignalRole,
		record.SignalDirection, record.DisplaySummary, record.DisplayOrder,
		record.SourceKind, record.VariableSignalID, record.SemanticSubmissionID,
		record.EvidenceID, record.EvidenceHash, record.UpstreamVariableSignalID,
		record.UpstreamDirectImpactAssertionID, record.EntityRelationID,
		record.IndustryChainGraphEdgeID,
	)
	return err
}

func (t *postgresResearchPublicationTx) InsertSnapshotSignal(ctx context.Context, record researchpublication.SnapshotSignalRecord) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO research_reasoning_tree_node_signals (
    reasoning_tree_node_id, signal_key, variable_name, signal_role, signal_direction,
    display_summary, display_order, source_kind
) VALUES ($1,$2,$3,$4,$5,$6,$7,'analyst_snapshot')`,
		record.ReasoningTreeNodeID, record.SignalKey, record.VariableName, record.SignalRole,
		record.SignalDirection, record.DisplaySummary, record.DisplayOrder)
	return err
}

func (t *postgresResearchPublicationTx) Verify(ctx context.Context, receipt researchpublication.Receipt) error {
	var counts researchpublication.Counts
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
	identityColumn := "industry_chain_entity_id::text"
	expected := receipt.ReasoningTreeIDsByIndustryChainEntityID
	if receipt.PublicationMode == researchpublication.SnapshotPublicationMode {
		identityColumn = "tree_key"
		expected = receipt.ReasoningTreeIDsByTreeKey
	}
	var treeIDsJSON []byte
	query := fmt.Sprintf(`SELECT COALESCE(jsonb_object_agg(%s, id::text), '{}'::jsonb)
FROM research_reasoning_trees WHERE theme_id = $1`, identityColumn)
	if err := t.tx.QueryRowContext(ctx, query, receipt.ThemeID).Scan(&treeIDsJSON); err != nil {
		return err
	}
	var treeIDs map[string]string
	if err := json.Unmarshal(treeIDsJSON, &treeIDs); err != nil {
		return err
	}
	if len(treeIDs) != len(expected) {
		return fmt.Errorf("aggregate Reason Tree identities do not match persisted rows")
	}
	for key, id := range expected {
		if treeIDs[key] != id {
			return fmt.Errorf("aggregate Reason Tree identity mismatch")
		}
	}
	return nil
}
