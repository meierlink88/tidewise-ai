package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

var (
	ErrResearchThemeNotFound          = errors.New("research theme not found")
	ErrResearchReasoningTreesNotFound = errors.New("research reasoning trees not found")
	ErrResearchReasoningTreeNotFound  = errors.New("research reasoning tree not found")
	ErrResearchReasoningTreeInvariant = errors.New("research reasoning tree invariant violation")
)

type ResearchReasoningTreeSummary struct {
	AnchorID            string `json:"anchor_id"`
	CenterChainNodeID   string `json:"center_chain_node_id"`
	CenterChainNodeName string `json:"center_chain_node_name"`
}

type ResearchReasoningTreeList struct {
	Theme          ResearchThemeSummary
	ReasoningTrees []ResearchReasoningTreeSummary
}

type ResearchReasoningTreeEvent struct {
	EventID         string     `json:"event_id"`
	Title           string     `json:"title"`
	Summary         string     `json:"summary"`
	EventTime       *time.Time `json:"event_time"`
	EvidenceRole    string     `json:"evidence_role"`
	EvidenceSummary string     `json:"evidence_summary"`
}

type ResearchReasoningTreePathNode struct {
	Position                      int     `json:"position"`
	ChainNodeID                   string  `json:"chain_node_id"`
	Name                          string  `json:"name"`
	ChangeDirection               string  `json:"change_direction"`
	ChangeSummary                 string  `json:"change_summary"`
	ImpactSummary                 string  `json:"impact_summary"`
	IncomingTransmissionMechanism *string `json:"incoming_transmission_mechanism"`
}

type ResearchReasoningTree struct {
	AnchorID            string
	CenterChainNodeID   string
	CenterChainNodeName string
	OneLineConclusion   string
	FactSummary         string
	NetDirectionSummary string
	SupportSummary      string
	CounterSummary      *string
	TradingDirection    string
	NextCheckpoint      string
	Events              []ResearchReasoningTreeEvent
	PathNodes           []ResearchReasoningTreePathNode
}

type ResearchReasoningTreeDetail struct {
	ThemeID       string
	ReasoningTree ResearchReasoningTree
}

const getResearchReasoningTreeThemeQuery = `
SELECT t.id, t.name, t.one_line_conclusion, t.impact_level, t.transmission_path,
       t.trading_direction, t.transmission_stage, t.next_checkpoint,
       t.market_confirmation_summary, t.published_at,
       COALESCE((SELECT jsonb_agg(jsonb_build_object('id', n.chain_node_entity_id, 'name', e.name, 'relation_role', n.relation_role, 'impact_summary', n.impact_summary) ORDER BY e.name, n.chain_node_entity_id)
                 FROM research_theme_chain_nodes n JOIN entity_nodes e ON e.id = n.chain_node_entity_id WHERE n.theme_id = t.id), '[]'::jsonb),
       COALESCE((SELECT jsonb_agg(jsonb_build_object('id', i.index_entity_id, 'name', e.name, 'impact_direction', i.impact_direction, 'impact_summary', i.impact_summary) ORDER BY e.name, i.index_entity_id)
                 FROM research_theme_indices i JOIN entity_nodes e ON e.id = i.index_entity_id WHERE i.theme_id = t.id), '[]'::jsonb),
       (SELECT COUNT(DISTINCT event_id) FROM research_theme_events WHERE theme_id = t.id AND evidence_role IN ('driver', 'supporting')),
       (SELECT COUNT(DISTINCT event_id) FROM research_theme_events WHERE theme_id = t.id AND evidence_role = 'contradicting')
FROM research_themes t
WHERE t.id = $1 AND t.published_at IS NOT NULL`

const getResearchReasoningTreePublicationQuery = `
SELECT r.id::text, r.anchor_ids_by_center_chain_node_id, r.write_counts,
       COALESCE((
           SELECT jsonb_agg(
               jsonb_build_object(
                   'anchor_id', a.id,
                   'center_chain_node_id', a.center_chain_node_entity_id,
                   'center_chain_node_name', e.name
               )
               ORDER BY e.name COLLATE "C" ASC, a.center_chain_node_entity_id ASC
           )
           FROM research_anchors a
           JOIN entity_nodes e ON e.id = a.center_chain_node_entity_id
           WHERE a.theme_id = r.theme_id AND a.import_receipt_id = r.id
       ), '[]'::jsonb),
       (SELECT COUNT(*)
        FROM research_anchor_events ae
        JOIN research_anchors a ON a.id = ae.anchor_id
        WHERE a.theme_id = r.theme_id AND a.import_receipt_id = r.id),
       (SELECT COUNT(*)
        FROM research_anchor_chain_nodes pn
        JOIN research_anchors a ON a.id = pn.anchor_id
        WHERE a.theme_id = r.theme_id AND a.import_receipt_id = r.id)
FROM research_anchor_import_receipts r
WHERE r.theme_id = $1`

const getResearchReasoningTreeDetailQuery = `
SELECT a.id, a.theme_id, a.center_chain_node_entity_id, center.name,
       a.one_line_conclusion, a.fact_summary, a.net_direction_summary,
       a.support_summary, a.counter_summary, a.trading_direction, a.next_checkpoint,
       COALESCE((
           SELECT jsonb_agg(
               jsonb_build_object(
                   'event_id', e.id,
                   'title', e.title,
                   'summary', e.summary,
                   'event_time', e.event_time,
                   'evidence_role', ae.evidence_role,
                   'evidence_summary', ae.evidence_summary
               )
               ORDER BY e.event_time ASC NULLS LAST, e.id ASC
           )
           FROM research_anchor_events ae
           JOIN events e ON e.id = ae.event_id
           WHERE ae.anchor_id = a.id
       ), '[]'::jsonb),
       COALESCE((
           SELECT jsonb_agg(
               jsonb_build_object(
                   'position', pn.position,
                   'chain_node_id', pn.chain_node_entity_id,
                   'name', node.name,
                   'change_direction', pn.change_direction,
                   'change_summary', pn.change_summary,
                   'impact_summary', pn.impact_summary,
                   'incoming_transmission_mechanism', pn.incoming_transmission_mechanism
               )
               ORDER BY pn.position ASC
           )
	           FROM research_anchor_chain_nodes pn
	           JOIN entity_nodes node ON node.id = pn.chain_node_entity_id
	           WHERE pn.anchor_id = a.id
	       ), '[]'::jsonb),
	       (SELECT COUNT(*)
	        FROM research_anchor_events ae
	        WHERE ae.anchor_id = a.id
	          AND NOT EXISTS (
	              SELECT 1
	              FROM research_theme_events te
	              WHERE te.theme_id = a.theme_id AND te.event_id = ae.event_id
	          ))
FROM research_anchors a
JOIN entity_nodes center ON center.id = a.center_chain_node_entity_id
WHERE a.theme_id = $1 AND a.id = $2 AND a.import_receipt_id = $3`

type researchReasoningTreePublication struct {
	ReceiptID string
	Mapping   map[string]string
	Counts    ResearchAnchorImportCounts
	Trees     []ResearchReasoningTreeSummary
}

func (r PostgresRepository) ListResearchThemeReasoningTrees(ctx context.Context, themeID string) (ResearchReasoningTreeList, error) {
	theme, err := r.readResearchReasoningTreeTheme(ctx, themeID)
	if err != nil {
		return ResearchReasoningTreeList{}, err
	}
	publication, err := r.readResearchReasoningTreePublication(ctx, themeID)
	if err != nil {
		return ResearchReasoningTreeList{}, err
	}
	if !reasoningTreeCentersMatchTheme(theme.ChainNodes, publication.Trees) {
		return ResearchReasoningTreeList{}, ErrResearchReasoningTreeInvariant
	}
	return ResearchReasoningTreeList{Theme: theme, ReasoningTrees: publication.Trees}, nil
}

func (r PostgresRepository) GetResearchThemeReasoningTree(ctx context.Context, themeID, anchorID string) (ResearchReasoningTreeDetail, error) {
	theme, err := r.readResearchReasoningTreeTheme(ctx, themeID)
	if err != nil {
		return ResearchReasoningTreeDetail{}, err
	}
	publication, err := r.readResearchReasoningTreePublication(ctx, themeID)
	if err != nil {
		return ResearchReasoningTreeDetail{}, err
	}
	if !reasoningTreeCentersMatchTheme(theme.ChainNodes, publication.Trees) {
		return ResearchReasoningTreeDetail{}, ErrResearchReasoningTreeInvariant
	}
	if !publicationContainsAnchor(publication.Mapping, anchorID) {
		return ResearchReasoningTreeDetail{}, ErrResearchReasoningTreeNotFound
	}

	var tree ResearchReasoningTree
	var returnedThemeID string
	var eventsJSON, pathNodesJSON []byte
	var invalidThemeEventCount int
	err = r.db.QueryRowContext(ctx, getResearchReasoningTreeDetailQuery, themeID, anchorID, publication.ReceiptID).Scan(
		&tree.AnchorID, &returnedThemeID, &tree.CenterChainNodeID, &tree.CenterChainNodeName,
		&tree.OneLineConclusion, &tree.FactSummary, &tree.NetDirectionSummary,
		&tree.SupportSummary, &tree.CounterSummary, &tree.TradingDirection, &tree.NextCheckpoint,
		&eventsJSON, &pathNodesJSON, &invalidThemeEventCount,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ResearchReasoningTreeDetail{}, ErrResearchReasoningTreeInvariant
	}
	if err != nil {
		return ResearchReasoningTreeDetail{}, fmt.Errorf("get research reasoning tree: %w", err)
	}
	if err := json.Unmarshal(eventsJSON, &tree.Events); err != nil {
		return ResearchReasoningTreeDetail{}, fmt.Errorf("decode research reasoning tree events: %w", err)
	}
	if err := json.Unmarshal(pathNodesJSON, &tree.PathNodes); err != nil {
		return ResearchReasoningTreeDetail{}, fmt.Errorf("decode research reasoning tree path: %w", err)
	}
	if returnedThemeID != themeID || publication.Mapping[tree.CenterChainNodeID] != tree.AnchorID || invalidThemeEventCount != 0 || !validReasoningTree(tree) {
		return ResearchReasoningTreeDetail{}, ErrResearchReasoningTreeInvariant
	}
	return ResearchReasoningTreeDetail{ThemeID: themeID, ReasoningTree: tree}, nil
}

func (r PostgresRepository) readResearchReasoningTreeTheme(ctx context.Context, themeID string) (ResearchThemeSummary, error) {
	theme, err := scanResearchThemeSummary(r.db.QueryRowContext(ctx, getResearchReasoningTreeThemeQuery, themeID))
	if errors.Is(err, sql.ErrNoRows) {
		return ResearchThemeSummary{}, ErrResearchThemeNotFound
	}
	if err != nil {
		return ResearchThemeSummary{}, fmt.Errorf("get research reasoning tree theme: %w", err)
	}
	return theme, nil
}

func (r PostgresRepository) readResearchReasoningTreePublication(ctx context.Context, themeID string) (researchReasoningTreePublication, error) {
	var publication researchReasoningTreePublication
	var mappingJSON, countsJSON, treesJSON []byte
	var eventCount, pathNodeCount int
	err := r.db.QueryRowContext(ctx, getResearchReasoningTreePublicationQuery, themeID).Scan(
		&publication.ReceiptID, &mappingJSON, &countsJSON, &treesJSON, &eventCount, &pathNodeCount,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return researchReasoningTreePublication{}, ErrResearchReasoningTreesNotFound
	}
	if err != nil {
		return researchReasoningTreePublication{}, fmt.Errorf("get research reasoning tree publication: %w", err)
	}
	if err := json.Unmarshal(mappingJSON, &publication.Mapping); err != nil {
		return researchReasoningTreePublication{}, fmt.Errorf("decode research reasoning tree mapping: %w", err)
	}
	if err := json.Unmarshal(countsJSON, &publication.Counts); err != nil {
		return researchReasoningTreePublication{}, fmt.Errorf("decode research reasoning tree counts: %w", err)
	}
	if err := json.Unmarshal(treesJSON, &publication.Trees); err != nil {
		return researchReasoningTreePublication{}, fmt.Errorf("decode research reasoning tree tabs: %w", err)
	}
	if !validReasoningTreePublication(publication, eventCount, pathNodeCount) {
		return researchReasoningTreePublication{}, ErrResearchReasoningTreeInvariant
	}
	return publication, nil
}

func validReasoningTreePublication(publication researchReasoningTreePublication, eventCount, pathNodeCount int) bool {
	if strings.TrimSpace(publication.ReceiptID) == "" || len(publication.Mapping) == 0 || len(publication.Trees) == 0 {
		return false
	}
	if publication.Counts != (ResearchAnchorImportCounts{
		Anchors: len(publication.Trees), EventAssociations: eventCount,
		PathNodes: pathNodeCount, Receipts: 1,
	}) {
		return false
	}
	actual := make(map[string]string, len(publication.Trees))
	anchorIDs := make(map[string]struct{}, len(publication.Trees))
	for _, tree := range publication.Trees {
		if strings.TrimSpace(tree.AnchorID) == "" || strings.TrimSpace(tree.CenterChainNodeID) == "" || strings.TrimSpace(tree.CenterChainNodeName) == "" {
			return false
		}
		if _, exists := actual[tree.CenterChainNodeID]; exists {
			return false
		}
		if _, exists := anchorIDs[tree.AnchorID]; exists {
			return false
		}
		actual[tree.CenterChainNodeID] = tree.AnchorID
		anchorIDs[tree.AnchorID] = struct{}{}
	}
	return reflect.DeepEqual(actual, publication.Mapping)
}

func reasoningTreeCentersMatchTheme(nodes []ResearchChainNode, trees []ResearchReasoningTreeSummary) bool {
	centers := make(map[string]struct{}, len(trees))
	for _, tree := range trees {
		centers[tree.CenterChainNodeID] = struct{}{}
	}
	if len(nodes) != len(centers) {
		return false
	}
	for _, node := range nodes {
		if _, exists := centers[node.ID]; !exists {
			return false
		}
	}
	return true
}

func publicationContainsAnchor(mapping map[string]string, anchorID string) bool {
	for _, value := range mapping {
		if value == anchorID {
			return true
		}
	}
	return false
}

func validReasoningTree(tree ResearchReasoningTree) bool {
	for _, value := range []string{
		tree.AnchorID, tree.CenterChainNodeID, tree.CenterChainNodeName, tree.OneLineConclusion,
		tree.FactSummary, tree.NetDirectionSummary, tree.SupportSummary, tree.TradingDirection, tree.NextCheckpoint,
	} {
		if strings.TrimSpace(value) == "" {
			return false
		}
	}
	if len(tree.Events) == 0 || len(tree.PathNodes) < 2 {
		return false
	}
	eventIDs := make(map[string]struct{}, len(tree.Events))
	hasDriver := false
	hasContradiction := false
	for _, event := range tree.Events {
		if strings.TrimSpace(event.EventID) == "" || strings.TrimSpace(event.EvidenceSummary) == "" {
			return false
		}
		if _, exists := eventIDs[event.EventID]; exists {
			return false
		}
		eventIDs[event.EventID] = struct{}{}
		switch event.EvidenceRole {
		case "driver":
			hasDriver = true
		case "supporting", "contradicting", "context":
			if event.EvidenceRole == "contradicting" {
				hasContradiction = true
			}
		default:
			return false
		}
	}
	if !hasDriver {
		return false
	}
	if hasContradiction && (tree.CounterSummary == nil || strings.TrimSpace(*tree.CounterSummary) == "") {
		return false
	}
	if !hasContradiction && tree.CounterSummary != nil {
		return false
	}

	nodeIDs := make(map[string]struct{}, len(tree.PathNodes))
	centerCount := 0
	for index, node := range tree.PathNodes {
		if node.Position != index+1 || strings.TrimSpace(node.ChainNodeID) == "" || strings.TrimSpace(node.Name) == "" ||
			strings.TrimSpace(node.ChangeSummary) == "" || strings.TrimSpace(node.ImpactSummary) == "" {
			return false
		}
		switch node.ChangeDirection {
		case "increase", "decrease", "mixed", "unchanged", "uncertain":
		default:
			return false
		}
		if _, exists := nodeIDs[node.ChainNodeID]; exists {
			return false
		}
		nodeIDs[node.ChainNodeID] = struct{}{}
		if node.ChainNodeID == tree.CenterChainNodeID {
			centerCount++
		}
		if index == 0 && node.IncomingTransmissionMechanism != nil {
			return false
		}
		if index > 0 && (node.IncomingTransmissionMechanism == nil || strings.TrimSpace(*node.IncomingTransmissionMechanism) == "") {
			return false
		}
	}
	return centerCount == 1
}
