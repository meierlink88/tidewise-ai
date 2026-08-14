package research

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	bizidentity "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/identity"
	researchbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/research"
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
		"research-publication-v2:"+analysisBatchID,
	)
	return err
}

func (t *publicationTransaction) Receipt(ctx context.Context, analysisBatchID string) (*researchbiz.Receipt, error) {
	var receipt researchbiz.Receipt
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
	if err := validatePersistedResearchReceipt(receipt); err != nil {
		return nil, err
	}
	return &receipt, nil
}

func validatePersistedResearchReceipt(receipt researchbiz.Receipt) error {
	invalid := func(reason string) error {
		return fmt.Errorf("persisted Research publication Receipt %q violates invariants: %s", receipt.ID, reason)
	}
	if !bizidentity.IsUUID(receipt.ID) || !bizidentity.IsUUID(receipt.ThemeID) ||
		strings.TrimSpace(receipt.AnalysisBatchID) == "" || strings.TrimSpace(receipt.PublisherSubject) == "" ||
		!researchHashPattern.MatchString(receipt.PayloadHash) || !researchKeyPattern.MatchString(receipt.ThemeKey) {
		return invalid("required identity is malformed")
	}
	if (receipt.ContractVersion != 2 && receipt.ContractVersion != 3) ||
		!oneOf(receipt.PublicationMode, "formal", researchbiz.SnapshotPublicationMode) ||
		(receipt.ContractVersion == 2) != (receipt.PublicationMode == "formal") {
		return invalid("contract version and publication mode are inconsistent")
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
	identities := receipt.ReasoningTreeIDsByIndustryChainEntityID
	if receipt.PublicationMode == researchbiz.SnapshotPublicationMode {
		if len(receipt.ReasoningTreeIDsByIndustryChainEntityID) != 0 {
			return invalid("snapshot Receipt contains formal Reason Tree identities")
		}
		identities = receipt.ReasoningTreeIDsByTreeKey
	} else if len(receipt.ReasoningTreeIDsByTreeKey) != 0 {
		return invalid("formal Receipt contains snapshot Reason Tree identities")
	}
	if len(identities) != counts.ReasoningTrees {
		return invalid("Reason Tree identity count does not match write counts")
	}
	for key, id := range identities {
		if key == "" || !bizidentity.IsUUID(id) {
			return invalid("a Reason Tree identity is malformed")
		}
		if receipt.PublicationMode == researchbiz.SnapshotPublicationMode && !researchKeyPattern.MatchString(key) {
			return invalid("a snapshot Reason Tree key is malformed")
		}
		if receipt.PublicationMode == "formal" && !bizidentity.IsUUID(key) {
			return invalid("a formal Industry Chain identity is malformed")
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
) (map[string]researchbiz.TemporalFact, error) {
	rows, err := tx.QueryContext(ctx, statement, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]researchbiz.TemporalFact, len(ids))
	for rows.Next() {
		var id string
		var temporal researchbiz.TemporalFact
		if err := rows.Scan(&id, &temporal.CreatedAt, &temporal.UpdatedAt); err != nil {
			return nil, err
		}
		result[id] = temporal
	}
	return result, rows.Err()
}

func (t *publicationTransaction) events(
	ctx context.Context,
	ids []string,
	existenceOnly bool,
) (map[string]researchbiz.EventFact, error) {
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

func (t *publicationTransaction) entityRelations(
	ctx context.Context,
	ids []string,
) (map[string]researchbiz.EntityRelationFact, error) {
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
	result := make(map[string]researchbiz.EntityRelationFact)
	for rows.Next() {
		var value researchbiz.EntityRelationFact
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

func (t *publicationTransaction) memberships(
	ctx context.Context,
	chainIDs []string,
) (map[string]map[string]researchbiz.TemporalFact, error) {
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
	result := make(map[string]map[string]researchbiz.TemporalFact)
	for rows.Next() {
		var chainID, nodeID string
		var temporal researchbiz.TemporalFact
		if err := rows.Scan(&chainID, &nodeID, &temporal.CreatedAt, &temporal.UpdatedAt); err != nil {
			return nil, err
		}
		if result[chainID] == nil {
			result[chainID] = make(map[string]researchbiz.TemporalFact)
		}
		result[chainID][nodeID] = temporal
	}
	return result, rows.Err()
}

func (t *publicationTransaction) graphEdges(
	ctx context.Context,
	ids []string,
) (map[string]researchbiz.GraphEdgeFact, error) {
	rows, err := t.tx.QueryContext(ctx, `SELECT id::text, industry_chain_entity_id::text,
       from_chain_node_entity_id::text, to_chain_node_entity_id::text,
       created_at, updated_at
FROM industry_chain_graph_edges
WHERE id = ANY($1::uuid[]) AND status = 'active' AND review_status = 'approved'`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]researchbiz.GraphEdgeFact)
	for rows.Next() {
		var value researchbiz.GraphEdgeFact
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

func (t *publicationTransaction) signals(ctx context.Context, ids []string) (map[string]researchbiz.SignalFact, error) {
	rows, err := t.tx.QueryContext(ctx, `SELECT signal.id::text, signal.semantic_submission_id::text,
	       signal.source_event_id::text, COALESCE(link.country_id, link.organization_id, link.entity_id::text), signal.variable_key, signal.direction,
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
	result := make(map[string]researchbiz.SignalFact)
	for rows.Next() {
		var value researchbiz.SignalFact
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

func (t *publicationTransaction) impacts(ctx context.Context, ids []string) (map[string]researchbiz.ImpactFact, error) {
	rows, err := t.tx.QueryContext(ctx, `SELECT impact.id::text, impact.semantic_submission_id::text,
       impact.source_variable_signal_id::text, impact.target_entity_id::text,
       impact.affected_variable_key, impact.affected_direction, signal.source_event_id::text,
	       COALESCE(source_link.country_id, source_link.organization_id, source_link.entity_id::text),
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
	result := make(map[string]researchbiz.ImpactFact)
	for rows.Next() {
		var value researchbiz.ImpactFact
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

func (t *publicationTransaction) evidences(ctx context.Context, ids []string) (map[string]researchbiz.EvidenceFact, error) {
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
	result := make(map[string]researchbiz.EvidenceFact)
	for rows.Next() {
		var value researchbiz.EvidenceFact
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

func (t *publicationTransaction) InsertThemeReceipt(ctx context.Context, receipt researchbiz.Receipt) error {
	themeIDs, err := json.Marshal(map[string]string{receipt.ThemeKey: receipt.ThemeID})
	if err != nil {
		return fmt.Errorf("encode Research Theme receipt IDs: %w", err)
	}
	treeIDs, err := json.Marshal(receipt.ReasoningTreeIDsByIndustryChainEntityID)
	if err != nil {
		return fmt.Errorf("encode Research Reason Tree receipt IDs: %w", err)
	}
	treeKeyIDs, err := json.Marshal(cloneStringMapOrEmpty(receipt.ReasoningTreeIDsByTreeKey))
	if err != nil {
		return fmt.Errorf("encode Research Reason Tree key IDs: %w", err)
	}
	legacyCounts, err := json.Marshal(struct {
		Themes            int `json:"themes"`
		Impacts           int `json:"impacts"`
		EventAssociations int `json:"event_associations"`
		Receipts          int `json:"receipts"`
	}{1, receipt.Counts.Impacts, receipt.Counts.ThemeEventAssociations, 1})
	if err != nil {
		return fmt.Errorf("encode legacy Research Theme receipt counts: %w", err)
	}
	aggregateCounts, err := json.Marshal(receipt.Counts)
	if err != nil {
		return fmt.Errorf("encode Research aggregate receipt counts: %w", err)
	}
	_, err = t.tx.ExecContext(ctx, `INSERT INTO research_theme_import_receipts (
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
func (t *publicationTransaction) InsertThemeImpact(ctx context.Context, record researchbiz.PublicationThemeImpactRecord) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO research_theme_impacts (
    theme_id, chain_node_entity_id, relation_role, impact_direction, impact_summary, display_order
) VALUES ($1,$2,$3,$4,$5,$6)`, record.ThemeID, record.ChainNodeEntityID,
		record.RelationRole, record.ImpactDirection, record.ImpactSummary, record.DisplayOrder)
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
) VALUES ($1,$2,$3,$4,COALESCE($5::uuid[], '{}'::uuid[]))`, record.ThemeID, record.EventID,
		record.EvidenceRole, record.SupportedClaim, record.EvidenceIDs)
	return err
}
func (t *publicationTransaction) InsertTreeReceipt(ctx context.Context, record researchbiz.ReasonTreeReceipt) error {
	treeIDs, err := json.Marshal(record.ReasoningTreeIDsByIndustryChainEntityID)
	if err != nil {
		return err
	}
	counts, err := json.Marshal(record.Counts)
	if err != nil {
		return err
	}
	_, err = t.tx.ExecContext(ctx, `INSERT INTO research_reasoning_tree_import_receipts (
    id, theme_id, publisher_subject, payload_hash,
    reasoning_tree_ids_by_industry_chain_entity_id, write_counts, published_at, imported_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, record.ID, record.ThemeID, record.PublisherSubject,
		record.PayloadHash, treeIDs, counts, record.PublishedAt, record.ImportedAt)
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
    reasoning_tree_ids_by_industry_chain_entity_id, reasoning_tree_ids_by_tree_key,
    write_counts, published_at, imported_at, publication_contract_version, publication_mode
) VALUES ($1,$2,$3,$4,'{}'::jsonb,$5,$6,$7,$8,3,'analyst_snapshot')`,
		record.ID, record.ThemeID, record.PublisherSubject, record.PayloadHash,
		treeIDs, counts, record.PublishedAt, record.ImportedAt)
	return err
}
func (t *publicationTransaction) InsertTree(ctx context.Context, record researchbiz.ReasonTreeRecord) error {
	conditions, err := json.Marshal(record.InvalidationConditions)
	if err != nil {
		return err
	}
	checkpoints, err := json.Marshal(record.Checkpoints)
	if err != nil {
		return err
	}
	_, err = t.tx.ExecContext(ctx, `INSERT INTO research_reasoning_trees (
    id, theme_id, import_receipt_id, industry_chain_entity_id, title, display_order,
    one_line_conclusion, fact_summary, transmission_summary, impact_direction,
    impact_strength, impact_summary, conclusion_boundary_summary, support_summary,
    counter_summary, invalidation_conditions, checkpoints
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		record.ID, record.ThemeID, record.ImportReceiptID, record.IndustryChainEntityID,
		record.Title, record.DisplayOrder, record.OneLineConclusion, record.FactSummary,
		record.TransmissionSummary, record.ImpactDirection, record.ImpactStrength,
		record.ImpactSummary, record.ConclusionBoundarySummary, record.SupportSummary,
		record.CounterSummary, conditions, checkpoints)
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
) VALUES ($1,$2,$3,$4,COALESCE($5::uuid[], '{}'::uuid[]))`, record.ReasoningTreeID,
		record.EventID, record.EvidenceRole, record.DisplayOrder, record.EvidenceIDs)
	return err
}

func (t *publicationTransaction) InsertNode(ctx context.Context, record researchbiz.NodeRecord) error {
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

func (t *publicationTransaction) InsertSignal(ctx context.Context, record researchbiz.PublicationSignalRecord) error {
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

func (t *publicationTransaction) InsertSnapshotSignal(ctx context.Context, record researchbiz.SnapshotSignalRecord) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO research_reasoning_tree_node_signals (
    reasoning_tree_node_id, signal_key, variable_name, signal_role, signal_direction,
    display_summary, display_order, source_kind
) VALUES ($1,$2,$3,$4,$5,$6,$7,'analyst_snapshot')`,
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
	identityColumn := "industry_chain_entity_id::text"
	expected := receipt.ReasoningTreeIDsByIndustryChainEntityID
	if receipt.PublicationMode == researchbiz.SnapshotPublicationMode {
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
