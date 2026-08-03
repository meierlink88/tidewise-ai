package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/research"
	researchimport "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchreasoningtreeimport"
)

var (
	ErrResearchThemeNotFound          = research.ErrResearchThemeNotFound
	ErrResearchReasoningTreesNotFound = research.ErrResearchReasoningTreesNotFound
	ErrResearchReasoningTreeNotFound  = research.ErrResearchReasoningTreeNotFound
	ErrResearchReasoningTreeInvariant = research.ErrResearchReasoningTreeInvariant
)

type ResearchReasoningTreeSummary = research.ReasoningTreeSummaryRecord
type ResearchReasoningTreeList = research.ReasoningTreeListRecord
type ResearchReasoningTree = research.ReasoningTreeRecord
type ResearchReasoningTreeDetail = research.ReasoningTreeDetailRecord

const getResearchReasoningTreePublicationQuery = `
SELECT receipt.id::text,
	   CASE WHEN receipt.publication_mode = 'analyst_snapshot'
	        THEN receipt.reasoning_tree_ids_by_tree_key
	        ELSE receipt.reasoning_tree_ids_by_industry_chain_entity_id END,
       receipt.write_counts,
       COALESCE((
           SELECT jsonb_agg(jsonb_build_object(
               'ReasoningTreeID', tree.id,
			   'TreeKey', COALESCE(tree.tree_key, tree.industry_chain_entity_id::text),
			   'DisplayName', COALESCE(tree.display_name, chain.name),
			   'IndustryChainEntityID', COALESCE(tree.industry_chain_entity_id::text, ''),
			   'IndustryChainName', COALESCE(chain.name, tree.display_name),
               'Title', tree.title,
               'DisplayOrder', tree.display_order,
               'EventCount', (SELECT count(*) FROM research_reasoning_tree_events e
                              WHERE e.reasoning_tree_id = tree.id),
               'PublishedAt', receipt.published_at
           ) ORDER BY tree.display_order)
           FROM research_reasoning_trees tree
		   LEFT JOIN entity_nodes chain ON chain.id = tree.industry_chain_entity_id
           WHERE tree.theme_id = receipt.theme_id AND tree.import_receipt_id = receipt.id
       ), '[]'::jsonb),
       (SELECT count(*) FROM research_reasoning_tree_nodes node
        JOIN research_reasoning_trees tree ON tree.id = node.reasoning_tree_id
        WHERE tree.import_receipt_id = receipt.id),
       (SELECT count(*) FROM research_reasoning_tree_events event
        JOIN research_reasoning_trees tree ON tree.id = event.reasoning_tree_id
        WHERE tree.import_receipt_id = receipt.id),
       (SELECT count(*) FROM research_reasoning_tree_node_signals signal
        JOIN research_reasoning_tree_nodes node ON node.id = signal.reasoning_tree_node_id
        JOIN research_reasoning_trees tree ON tree.id = node.reasoning_tree_id
        WHERE tree.import_receipt_id = receipt.id)
FROM research_reasoning_tree_import_receipts receipt
WHERE receipt.theme_id = $1`

const getResearchReasoningTreeDetailQuery = `
SELECT theme.theme_key, receipt.publication_mode, receipt.publication_contract_version,
	   tree.id::text, tree.theme_id::text,
	   COALESCE(tree.tree_key, tree.industry_chain_entity_id::text),
	   COALESCE(tree.display_name, chain.name),
	   COALESCE(tree.industry_chain_entity_id::text, ''),
	   COALESCE(chain.name, tree.display_name), tree.title, tree.display_order, tree.one_line_conclusion,
       tree.fact_summary, tree.transmission_summary, tree.impact_direction,
       tree.impact_strength, tree.impact_summary, tree.conclusion_boundary_summary,
       tree.support_summary, tree.counter_summary, tree.invalidation_conditions,
       tree.checkpoints, receipt.published_at,
       COALESCE((
           SELECT jsonb_agg(jsonb_build_object(
               'event_id', event.id,
			   'evidence_ids', association.evidence_ids,
               'title', event.title,
               'summary', event.summary,
               'event_time', event.event_time,
               'evidence_role', association.evidence_role,
               'display_order', association.display_order
           ) ORDER BY association.display_order)
           FROM research_reasoning_tree_events association
           JOIN events event ON event.id = association.event_id
           WHERE association.reasoning_tree_id = tree.id
       ), '[]'::jsonb),
       COALESCE((
           SELECT jsonb_agg(jsonb_build_object(
               'ID', node.id,
			   'NodeKey', COALESCE(node.node_key, node.chain_node_entity_id::text),
			   'DisplayName', COALESCE(node.display_name, chain_node.name),
               'Position', node.position,
			   'ChainNodeEntityID', COALESCE(node.chain_node_entity_id::text, ''),
			   'Name', COALESCE(chain_node.name, node.display_name),
               'StateSummary', node.state_summary,
               'ImpactDirection', node.impact_direction,
               'ImpactStrength', node.impact_strength,
               'ImpactSummary', node.impact_summary,
               'ReasoningBasisSummary', node.reasoning_basis_summary,
               'EvidenceGapSummary', node.evidence_gap_summary,
               'IncomingIndustryChainGraphEdgeID', node.incoming_industry_chain_graph_edge_id,
               'IncomingTransmissionTitle', node.incoming_transmission_title,
               'IncomingTransmissionMechanism', node.incoming_transmission_mechanism,
               'IncomingConditionSummary', node.incoming_condition_summary,
               'IncomingGraphEdge', CASE WHEN edge.id IS NULL THEN NULL ELSE jsonb_build_object(
                   'ID', edge.id,
                   'RelationType', edge.relation_type,
                   'ReviewStatus', edge.review_status,
                   'Status', edge.status
               ) END,
               'Signals', COALESCE((
                   SELECT jsonb_agg(jsonb_build_object(
                       'VariableSignalKey', signal.variable_signal_key,
					   'SignalKey', COALESCE(signal.signal_key, signal.variable_signal_key),
					   'VariableName', signal.variable_name,
					   'Direction', signal.signal_direction,
                       'SignalRole', signal.signal_role,
                       'SignalDirection', signal.signal_direction,
                       'DisplaySummary', signal.display_summary,
                       'DisplayOrder', signal.display_order
                   ) ORDER BY signal.display_order)
                   FROM research_reasoning_tree_node_signals signal
                   WHERE signal.reasoning_tree_node_id = node.id
               ), '[]'::jsonb)
           ) ORDER BY node.position)
           FROM research_reasoning_tree_nodes node
		   LEFT JOIN entity_nodes chain_node ON chain_node.id = node.chain_node_entity_id
           LEFT JOIN industry_chain_graph_edges edge
             ON edge.id = node.incoming_industry_chain_graph_edge_id
           WHERE node.reasoning_tree_id = tree.id
       ), '[]'::jsonb),
       COALESCE((
		   SELECT jsonb_agg(COALESCE(impact.node_key, impact.chain_node_entity_id::text) ORDER BY impact.display_order)
           FROM research_theme_impacts impact WHERE impact.theme_id = tree.theme_id
       ), '[]'::jsonb),
       (SELECT count(*) FROM research_reasoning_tree_events event
        WHERE event.reasoning_tree_id = tree.id
          AND NOT EXISTS (
              SELECT 1 FROM research_theme_events theme_event
              WHERE theme_event.theme_id = tree.theme_id
                AND theme_event.event_id = event.event_id
          ))
FROM research_reasoning_trees tree
JOIN research_reasoning_tree_import_receipts receipt ON receipt.id = tree.import_receipt_id
	JOIN research_themes theme ON theme.id = tree.theme_id
	LEFT JOIN entity_nodes chain ON chain.id = tree.industry_chain_entity_id
WHERE tree.theme_id = $1 AND tree.id = $2`

type researchReasoningTreePublication struct {
	ReceiptID string
	Mapping   map[string]string
	Counts    researchimport.Counts
	Trees     []ResearchReasoningTreeSummary
}

func (r repository) ListResearchThemeReasoningTrees(ctx context.Context, themeID string) (ResearchReasoningTreeList, error) {
	theme, err := r.readResearchReasoningTreeTheme(ctx, themeID)
	if err != nil {
		return ResearchReasoningTreeList{}, err
	}
	publication, err := r.readResearchReasoningTreePublication(ctx, themeID)
	if err != nil {
		return ResearchReasoningTreeList{}, err
	}
	return ResearchReasoningTreeList{Theme: theme, ReasoningTrees: publication.Trees}, nil
}

func (r repository) GetResearchThemeReasoningTree(ctx context.Context, themeID, reasoningTreeID string) (ResearchReasoningTreeDetail, error) {
	if _, err := r.readResearchReasoningTreeTheme(ctx, themeID); err != nil {
		return ResearchReasoningTreeDetail{}, err
	}
	publication, err := r.readResearchReasoningTreePublication(ctx, themeID)
	if err != nil {
		return ResearchReasoningTreeDetail{}, err
	}
	if !publicationContainsReasoningTree(publication.Mapping, reasoningTreeID) {
		return ResearchReasoningTreeDetail{}, ErrResearchReasoningTreeNotFound
	}
	var result ResearchReasoningTreeDetail
	var tree ResearchReasoningTree
	var invalidationJSON, checkpointsJSON, eventsJSON, nodesJSON, impactIDsJSON []byte
	var invalidThemeEventCount int
	err = r.db.QueryRowContext(ctx, getResearchReasoningTreeDetailQuery,
		themeID, reasoningTreeID).Scan(
		&result.ThemeKey, &result.PublicationMode, &result.PublicationContractVersion,
		&tree.ReasoningTreeID, &tree.ThemeID, &tree.TreeKey, &tree.DisplayName,
		&tree.IndustryChainEntityID,
		&tree.IndustryChainName, &tree.Title, &tree.DisplayOrder,
		&tree.OneLineConclusion, &tree.FactSummary, &tree.TransmissionSummary,
		&tree.ImpactDirection, &tree.ImpactStrength, &tree.ImpactSummary,
		&tree.ConclusionBoundarySummary, &tree.SupportSummary, &tree.CounterSummary,
		&invalidationJSON, &checkpointsJSON, &tree.PublishedAt,
		&eventsJSON, &nodesJSON, &impactIDsJSON, &invalidThemeEventCount,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ResearchReasoningTreeDetail{}, ErrResearchReasoningTreeInvariant
	}
	if err != nil {
		return ResearchReasoningTreeDetail{}, fmt.Errorf("get research reasoning tree: %w", err)
	}
	var impactNodeIDs []string
	if err := json.Unmarshal(invalidationJSON, &tree.InvalidationConditions); err != nil {
		return ResearchReasoningTreeDetail{}, ErrResearchReasoningTreeInvariant
	}
	if err := json.Unmarshal(checkpointsJSON, &tree.Checkpoints); err != nil {
		return ResearchReasoningTreeDetail{}, ErrResearchReasoningTreeInvariant
	}
	if err := json.Unmarshal(eventsJSON, &tree.Events); err != nil {
		return ResearchReasoningTreeDetail{}, ErrResearchReasoningTreeInvariant
	}
	if err := json.Unmarshal(nodesJSON, &tree.Nodes); err != nil {
		return ResearchReasoningTreeDetail{}, ErrResearchReasoningTreeInvariant
	}
	if err := json.Unmarshal(impactIDsJSON, &impactNodeIDs); err != nil {
		return ResearchReasoningTreeDetail{}, ErrResearchReasoningTreeInvariant
	}
	tree.EventCount = len(tree.Events)
	if tree.ThemeID != themeID || invalidThemeEventCount != 0 ||
		!validReasoningTreeDetail(tree, impactNodeIDs) {
		return ResearchReasoningTreeDetail{}, ErrResearchReasoningTreeInvariant
	}
	result.ThemeID = themeID
	result.ImpactNodeIDs = impactNodeIDs
	result.ReasoningTree = tree
	return result, nil
}

func (r repository) readResearchReasoningTreeTheme(ctx context.Context, themeID string) (ResearchThemeSummary, error) {
	theme, err := scanResearchThemeSummary(r.db.QueryRowContext(ctx, getResearchThemeByIDQuery, themeID))
	if errors.Is(err, sql.ErrNoRows) {
		return ResearchThemeSummary{}, ErrResearchThemeNotFound
	}
	if err != nil {
		return ResearchThemeSummary{}, fmt.Errorf("get research reasoning tree Theme: %w", err)
	}
	return theme, nil
}

func (r repository) readResearchReasoningTreePublication(ctx context.Context, themeID string) (researchReasoningTreePublication, error) {
	var publication researchReasoningTreePublication
	var mappingJSON, countsJSON, treesJSON []byte
	var nodeCount, eventCount, signalCount int
	err := r.db.QueryRowContext(ctx, getResearchReasoningTreePublicationQuery, themeID).Scan(
		&publication.ReceiptID, &mappingJSON, &countsJSON, &treesJSON,
		&nodeCount, &eventCount, &signalCount,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return researchReasoningTreePublication{}, ErrResearchReasoningTreesNotFound
	}
	if err != nil {
		return researchReasoningTreePublication{}, fmt.Errorf("get research reasoning tree publication: %w", err)
	}
	if json.Unmarshal(mappingJSON, &publication.Mapping) != nil ||
		json.Unmarshal(countsJSON, &publication.Counts) != nil ||
		json.Unmarshal(treesJSON, &publication.Trees) != nil ||
		!validReasoningTreePublication(publication, nodeCount, eventCount, signalCount) {
		return researchReasoningTreePublication{}, ErrResearchReasoningTreeInvariant
	}
	return publication, nil
}

func validReasoningTreePublication(publication researchReasoningTreePublication, nodeCount, eventCount, signalCount int) bool {
	if strings.TrimSpace(publication.ReceiptID) == "" || len(publication.Mapping) == 0 ||
		len(publication.Trees) == 0 {
		return false
	}
	if publication.Counts != (researchimport.Counts{
		ReasoningTrees: len(publication.Trees), Nodes: nodeCount,
		EventAssociations: eventCount, SignalAssociations: signalCount, Receipts: 1,
	}) {
		return false
	}
	actual := make(map[string]string, len(publication.Trees))
	for index, tree := range publication.Trees {
		if tree.DisplayOrder != index+1 || strings.TrimSpace(tree.Title) == "" ||
			strings.TrimSpace(tree.IndustryChainName) == "" {
			return false
		}
		actual[tree.TreeKey] = tree.ReasoningTreeID
	}
	return reflect.DeepEqual(actual, publication.Mapping)
}

func publicationContainsReasoningTree(mapping map[string]string, reasoningTreeID string) bool {
	for _, value := range mapping {
		if value == reasoningTreeID {
			return true
		}
	}
	return false
}

func validReasoningTreeDetail(tree ResearchReasoningTree, impactNodeIDs []string) bool {
	if len(tree.Nodes) == 0 || len(impactNodeIDs) == 0 {
		return false
	}
	impactSet := make(map[string]struct{}, len(impactNodeIDs))
	for _, id := range impactNodeIDs {
		impactSet[id] = struct{}{}
	}
	hasImpact := false
	for index, node := range tree.Nodes {
		if node.Position != index+1 || len(node.Signals) < 1 || len(node.Signals) > 5 {
			return false
		}
		if _, impact := impactSet[node.NodeKey]; impact {
			hasImpact = true
		}
		primaryCount := 0
		for signalIndex, signal := range node.Signals {
			if signal.DisplayOrder != signalIndex+1 {
				return false
			}
			if signal.SignalRole == "primary" {
				primaryCount++
				if signal.DisplayOrder != 1 {
					return false
				}
			}
		}
		if primaryCount != 1 {
			return false
		}
	}
	return hasImpact
}
