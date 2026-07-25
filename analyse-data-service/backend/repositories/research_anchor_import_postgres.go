package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

func (r PostgresRepository) InResearchAnchorImportTransaction(ctx context.Context, fn func(ResearchAnchorImportTransaction) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin Research Anchor import transaction: %w", err)
	}
	wrapper := &postgresResearchAnchorImportTx{tx: tx}
	if err := fn(wrapper); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit Research Anchor import transaction: %w", err)
	}
	return nil
}

type postgresResearchAnchorImportTx struct{ tx *sql.Tx }

func (t *postgresResearchAnchorImportTx) LockResearchAnchorImportTheme(ctx context.Context, themeID string) error {
	lockKey := "research-anchor:" + themeID
	if _, err := t.tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, lockKey); err != nil {
		return fmt.Errorf("lock Research Anchor Theme %q: %w", themeID, err)
	}
	return nil
}

func (t *postgresResearchAnchorImportTx) ResearchAnchorImportReceipt(ctx context.Context, themeID string) (*ResearchAnchorImportReceipt, error) {
	var receipt ResearchAnchorImportReceipt
	var anchorIDsJSON []byte
	var countsJSON []byte
	err := t.tx.QueryRowContext(ctx, `SELECT id, theme_id, publisher_subject, payload_hash,
       anchor_ids_by_center_chain_node_id, write_counts, published_at, imported_at
FROM research_anchor_import_receipts WHERE theme_id = $1`, themeID).Scan(
		&receipt.ID, &receipt.ThemeID, &receipt.PublisherSubject, &receipt.PayloadHash,
		&anchorIDsJSON, &countsJSON, &receipt.PublishedAt, &receipt.ImportedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read Research Anchor import receipt: %w", err)
	}
	if err := json.Unmarshal(anchorIDsJSON, &receipt.AnchorIDsByCenterChainNodeID); err != nil {
		return nil, fmt.Errorf("decode receipt Anchor mapping: %w", err)
	}
	if err := json.Unmarshal(countsJSON, &receipt.Counts); err != nil {
		return nil, fmt.Errorf("decode receipt write counts: %w", err)
	}
	if len(receipt.AnchorIDsByCenterChainNodeID) == 0 || receipt.Counts.Anchors != len(receipt.AnchorIDsByCenterChainNodeID) {
		return nil, fmt.Errorf("receipt Anchor mapping does not match its Anchor count")
	}
	return &receipt, nil
}

func (t *postgresResearchAnchorImportTx) ResearchAnchorImportThemePublication(ctx context.Context, themeID string) (*ResearchAnchorImportThemePublication, error) {
	var publication ResearchAnchorImportThemePublication
	var receiptID sql.NullString
	var publisher sql.NullString
	err := t.tx.QueryRowContext(ctx, `SELECT t.id::text, t.import_receipt_id::text, r.publisher_subject
FROM research_themes t
LEFT JOIN research_theme_import_receipts r ON r.id = t.import_receipt_id
WHERE t.id = $1`, themeID).Scan(&publication.ThemeID, &receiptID, &publisher)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read parent Theme publication: %w", err)
	}
	if receiptID.Valid {
		if !publisher.Valid || publisher.String == "" {
			return nil, fmt.Errorf("parent Theme receipt has no publisher subject")
		}
		publication.ThemeImportReceiptID = receiptID.String
		publication.PublisherSubject = publisher.String
	}
	return &publication, nil
}

func (t *postgresResearchAnchorImportTx) ResearchAnchorImportThemeChainNodes(ctx context.Context, themeID string) (map[string]struct{}, error) {
	return queryResearchAnchorSet(ctx, t.tx, `SELECT chain_node_entity_id::text FROM research_theme_chain_nodes WHERE theme_id = $1`, themeID)
}

func (t *postgresResearchAnchorImportTx) ResearchAnchorImportThemeEvents(ctx context.Context, themeID string) (map[string]struct{}, error) {
	return queryResearchAnchorSet(ctx, t.tx, `SELECT event_id::text FROM research_theme_events WHERE theme_id = $1`, themeID)
}

func (t *postgresResearchAnchorImportTx) ExistingResearchAnchorChainNodes(ctx context.Context, ids []string) (map[string]struct{}, error) {
	return queryExistingResearchThemeIDs(ctx, t.tx, `SELECT entity_id::text FROM chain_node_profiles WHERE entity_id = ANY($1::uuid[])`, ids)
}

func (t *postgresResearchAnchorImportTx) ExistingResearchAnchorEvents(ctx context.Context, ids []string) (map[string]struct{}, error) {
	return queryExistingResearchThemeIDs(ctx, t.tx, `SELECT id::text FROM events WHERE id = ANY($1::uuid[])`, ids)
}

func queryResearchAnchorSet(ctx context.Context, tx *sql.Tx, query string, argument any) (map[string]struct{}, error) {
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (t *postgresResearchAnchorImportTx) InsertResearchAnchorImportReceipt(ctx context.Context, receipt ResearchAnchorImportReceipt) error {
	anchorIDs, err := json.Marshal(receipt.AnchorIDsByCenterChainNodeID)
	if err != nil {
		return fmt.Errorf("encode receipt Anchor mapping: %w", err)
	}
	counts, err := json.Marshal(receipt.Counts)
	if err != nil {
		return fmt.Errorf("encode receipt counts: %w", err)
	}
	_, err = t.tx.ExecContext(ctx, `INSERT INTO research_anchor_import_receipts (
    id, theme_id, publisher_subject, payload_hash,
    anchor_ids_by_center_chain_node_id, write_counts, published_at, imported_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		receipt.ID, receipt.ThemeID, receipt.PublisherSubject, receipt.PayloadHash,
		anchorIDs, counts, receipt.PublishedAt, receipt.ImportedAt,
	)
	return err
}

func (t *postgresResearchAnchorImportTx) InsertResearchAnchor(ctx context.Context, anchor ResearchAnchorImportAnchor) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO research_anchors (
    id, theme_id, center_chain_node_entity_id, import_receipt_id,
    one_line_conclusion, fact_summary, net_direction_summary, support_summary,
    counter_summary, trading_direction, next_checkpoint
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		anchor.ID, anchor.ThemeID, anchor.CenterChainNodeEntityID, anchor.ImportReceiptID,
		anchor.OneLineConclusion, anchor.FactSummary, anchor.NetDirectionSummary,
		anchor.SupportSummary, anchor.CounterSummary, anchor.TradingDirection, anchor.NextCheckpoint,
	)
	return err
}

func (t *postgresResearchAnchorImportTx) InsertResearchAnchorEvent(ctx context.Context, event ResearchAnchorImportEvent) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO research_anchor_events (
    anchor_id, event_id, evidence_role, evidence_summary
) VALUES ($1,$2,$3,$4)`, event.AnchorID, event.EventID, event.EvidenceRole, event.EvidenceSummary)
	return err
}

func (t *postgresResearchAnchorImportTx) InsertResearchAnchorPathNode(ctx context.Context, node ResearchAnchorImportPathNode) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO research_anchor_chain_nodes (
    anchor_id, position, chain_node_entity_id, change_direction,
    change_summary, impact_summary, incoming_transmission_mechanism
) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		node.AnchorID, node.Position, node.ChainNodeEntityID, node.ChangeDirection,
		node.ChangeSummary, node.ImpactSummary, node.IncomingTransmissionMechanism,
	)
	return err
}

func (t *postgresResearchAnchorImportTx) VerifyResearchAnchorImportReceipt(ctx context.Context, receipt ResearchAnchorImportReceipt) error {
	var anchorIDsJSON []byte
	if err := t.tx.QueryRowContext(ctx, `SELECT COALESCE(jsonb_object_agg(center_chain_node_entity_id::text, id::text), '{}'::jsonb)
FROM research_anchors WHERE import_receipt_id = $1`, receipt.ID).Scan(&anchorIDsJSON); err != nil {
		return fmt.Errorf("verify receipt Anchor IDs: %w", err)
	}
	var anchorIDs map[string]string
	if err := json.Unmarshal(anchorIDsJSON, &anchorIDs); err != nil {
		return fmt.Errorf("decode verified receipt Anchor IDs: %w", err)
	}
	if !reflect.DeepEqual(anchorIDs, receipt.AnchorIDsByCenterChainNodeID) {
		return fmt.Errorf("receipt Anchor IDs are not all present")
	}

	var counts ResearchAnchorImportCounts
	if err := t.tx.QueryRowContext(ctx, `SELECT
    (SELECT count(*) FROM research_anchors WHERE import_receipt_id = $1),
    (SELECT count(*) FROM research_anchor_events e JOIN research_anchors a ON a.id = e.anchor_id WHERE a.import_receipt_id = $1),
    (SELECT count(*) FROM research_anchor_chain_nodes n JOIN research_anchors a ON a.id = n.anchor_id WHERE a.import_receipt_id = $1)`, receipt.ID).Scan(
		&counts.Anchors, &counts.EventAssociations, &counts.PathNodes,
	); err != nil {
		return fmt.Errorf("verify receipt write counts: %w", err)
	}
	counts.Receipts = 1
	if counts != receipt.Counts {
		return fmt.Errorf("receipt write counts do not match persisted rows")
	}
	return nil
}
