package entity

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	biz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/entity"
	bizidentity "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/identity"
)

const researchGraphCTE = `
	WITH RECURSIVE
	requested_seeds(entity_id) AS (
	    SELECT unnest($2::uuid[])
	),
	requested_filters(relation_type, direction) AS (
	    SELECT *
	    FROM unnest($3::text[], $4::text[])
	),
	eligible_edges AS NOT MATERIALIZED (
	    SELECT
	        'entity_relation'::text edge_kind,
	        relation.id edge_id,
	        NULL::uuid industry_chain_entity_id,
	        relation.from_entity_id,
	        relation.to_entity_id,
	        relation.relation_type,
	        ''::text mechanism,
	        NULL::text condition_note,
	        ''::text segment_kind,
	        NULL::text omitted_step_note
	    FROM entity_edges relation
	    JOIN entity_nodes from_entity
	      ON from_entity.id = relation.from_entity_id
	     AND from_entity.status = $9
	     AND from_entity.created_at <= $1
	     AND from_entity.updated_at <= $1
	    JOIN entity_nodes to_entity
	      ON to_entity.id = relation.to_entity_id
	     AND to_entity.status = $9
	     AND to_entity.created_at <= $1
	     AND to_entity.updated_at <= $1
	    JOIN requested_filters filter
	      ON filter.relation_type = relation.relation_type
	    WHERE relation.status = $10
	      AND relation.created_at <= $1
	      AND relation.updated_at <= $1
	    UNION
	    SELECT
	        'industry_chain_graph_edge'::text,
	        edge.id,
	        edge.industry_chain_entity_id,
	        edge.from_chain_node_entity_id,
	        edge.to_chain_node_entity_id,
	        edge.relation_type,
	        edge.mechanism,
	        edge.condition_note,
	        edge.segment_kind,
	        edge.omitted_step_note
	    FROM industry_chain_graph_edges edge
	    JOIN industry_chain_definitions definition
	      ON definition.entity_id = edge.industry_chain_entity_id
	     AND definition.review_status = $11
	     AND definition.created_at <= $1
	     AND definition.updated_at <= $1
	    JOIN industry_chain_node_memberships from_membership
	      ON from_membership.industry_chain_entity_id = edge.industry_chain_entity_id
	     AND from_membership.chain_node_entity_id = edge.from_chain_node_entity_id
	     AND from_membership.review_status = $12
	     AND from_membership.status = $13
	     AND from_membership.created_at <= $1
	     AND from_membership.updated_at <= $1
	    JOIN industry_chain_node_memberships to_membership
	      ON to_membership.industry_chain_entity_id = edge.industry_chain_entity_id
	     AND to_membership.chain_node_entity_id = edge.to_chain_node_entity_id
	     AND to_membership.review_status = $12
	     AND to_membership.status = $13
	     AND to_membership.created_at <= $1
	     AND to_membership.updated_at <= $1
	    JOIN entity_nodes from_entity
	      ON from_entity.id = edge.from_chain_node_entity_id
	     AND from_entity.status = $9
	     AND from_entity.created_at <= $1
	     AND from_entity.updated_at <= $1
	    JOIN entity_nodes to_entity
	      ON to_entity.id = edge.to_chain_node_entity_id
	     AND to_entity.status = $9
	     AND to_entity.created_at <= $1
	     AND to_entity.updated_at <= $1
	    JOIN requested_filters filter
	      ON filter.relation_type = edge.relation_type
	    WHERE edge.status = $15
	      AND edge.review_status = $14
	      AND edge.created_at <= $1
	      AND edge.updated_at <= $1
	      AND ($6::uuid IS NULL OR edge.industry_chain_entity_id = $6::uuid)
	),
	traversal_options AS NOT MATERIALIZED (
	    SELECT
	        edge.edge_kind,
	        edge.edge_id,
	        edge.from_entity_id traversal_from,
	        edge.to_entity_id traversal_to
	    FROM eligible_edges edge
	    JOIN requested_filters filter
	      ON filter.relation_type = edge.relation_type
	     AND filter.direction IN ('outgoing', 'both')
	    UNION
	    SELECT
	        edge.edge_kind,
	        edge.edge_id,
	        edge.to_entity_id,
	        edge.from_entity_id
	    FROM eligible_edges edge
	    JOIN requested_filters filter
	      ON filter.relation_type = edge.relation_type
	     AND filter.direction IN ('incoming', 'both')
	),
	walk(depth, entity_ids, entity_depths, frontier) AS (
	    SELECT
	        0,
	        array_agg(seed.entity_id ORDER BY seed.entity_id),
	        array_agg(0 ORDER BY seed.entity_id),
	        array_agg(seed.entity_id ORDER BY seed.entity_id)
	    FROM requested_seeds seed
	    UNION ALL
	    SELECT
	        walk.depth + 1,
	        walk.entity_ids || expansion.new_entities,
	        walk.entity_depths || array_fill(
	            walk.depth + 1,
	            ARRAY[cardinality(expansion.new_entities)]
	        ),
	        expansion.new_entities
	    FROM walk
	    CROSS JOIN LATERAL (
	        SELECT COALESCE(
	            array_agg(candidate.entity_id ORDER BY candidate.entity_id),
	            '{}'::uuid[]
	        ) new_entities
	        FROM (
	            SELECT DISTINCT option.traversal_to entity_id
	            FROM traversal_options option
	            WHERE option.traversal_from = ANY(walk.frontier)
	              AND NOT option.traversal_to = ANY(walk.entity_ids)
	            ORDER BY option.traversal_to
	            LIMIT GREATEST($7 + 1 - cardinality(walk.entity_ids), 0)
	        ) candidate
	    ) expansion
	    WHERE walk.depth < $5
	      AND cardinality(walk.frontier) > 0
	      AND cardinality(walk.entity_ids) <= $7
	      AND cardinality(expansion.new_entities) > 0
	),
	final_walk AS MATERIALIZED (
	    SELECT * FROM walk ORDER BY depth DESC LIMIT 1
	),
	reached_entities(entity_id, depth) AS MATERIALIZED (
	    SELECT reached.entity_id, reached.depth
	    FROM final_walk,
	         unnest(final_walk.entity_ids, final_walk.entity_depths)
	         AS reached(entity_id, depth)
	),
	used_edges(edge_kind, edge_id) AS MATERIALIZED (
	    SELECT DISTINCT option.edge_kind, option.edge_id
	    FROM reached_entities reached
	    JOIN traversal_options option
	      ON option.traversal_from = reached.entity_id
	    JOIN reached_entities target
	      ON target.entity_id = option.traversal_to
	    WHERE reached.depth < $5
	    ORDER BY option.edge_kind, option.edge_id
	    LIMIT $8 + 1
	),
	selected_entity_relations AS MATERIALIZED (
	    SELECT relation.*
	    FROM entity_edges relation
	    JOIN used_edges used
	      ON used.edge_kind = 'entity_relation'
	     AND used.edge_id = relation.id
	),
	selected_graph_edges AS MATERIALIZED (
	    SELECT edge.*
	    FROM industry_chain_graph_edges edge
	    JOIN used_edges used
	      ON used.edge_kind = 'industry_chain_graph_edge'
	     AND used.edge_id = edge.id
	),
	selected_chain_ids(industry_chain_entity_id) AS MATERIALIZED (
	    SELECT DISTINCT industry_chain_entity_id
	    FROM selected_graph_edges
	    UNION
	    SELECT $6::uuid
	    WHERE $6::uuid IS NOT NULL
	),
	selected_entity_ids(entity_id) AS MATERIALIZED (
	    SELECT entity_id FROM reached_entities
	    UNION
	    SELECT industry_chain_entity_id FROM selected_chain_ids
	),
	selected_entities AS MATERIALIZED (
	    SELECT entity.*
	    FROM entity_nodes entity
	    JOIN selected_entity_ids selected ON selected.entity_id = entity.id
	    WHERE entity.status = $9
	      AND entity.created_at <= $1
	      AND entity.updated_at <= $1
	),
	selected_industry_chains AS MATERIALIZED (
	    SELECT definition.*
	    FROM industry_chain_definitions definition
	    JOIN selected_chain_ids selected
	      ON selected.industry_chain_entity_id = definition.entity_id
	    WHERE definition.review_status = $11
	      AND definition.created_at <= $1
	      AND definition.updated_at <= $1
	),
	selected_memberships AS MATERIALIZED (
	    SELECT membership.*
	    FROM industry_chain_node_memberships membership
	    JOIN selected_chain_ids selected_chain
	      ON selected_chain.industry_chain_entity_id = membership.industry_chain_entity_id
	    JOIN reached_entities selected_node
	      ON selected_node.entity_id = membership.chain_node_entity_id
	    WHERE membership.review_status = $12
	      AND membership.status = $13
	      AND membership.created_at <= $1
	      AND membership.updated_at <= $1
	),
	selected_relation_types(relation_type) AS MATERIALIZED (
	    SELECT relation_type FROM selected_entity_relations
	    UNION
	    SELECT relation_type FROM selected_graph_edges
	)
`

func (s *Store) SearchResearchGraph(
	ctx context.Context,
	query biz.ResearchGraphQuery,
) (biz.ResearchGraphSubgraph, error) {
	if s == nil || s.db == nil {
		return biz.ResearchGraphSubgraph{}, errors.New("research graph database is required")
	}
	relationTypes := make([]string, 0, len(query.RelationFilters))
	directions := make([]string, 0, len(query.RelationFilters))
	for _, filter := range query.RelationFilters {
		relationTypes = append(relationTypes, filter.RelationType)
		directions = append(directions, string(filter.Direction))
	}
	args := []any{
		query.AnalysisAsOf,
		query.SeedEntityIDs,
		relationTypes,
		directions,
		query.MaxDepth,
		query.IndustryChainEntityID,
		query.NodeBudget,
		query.EdgeBudget,
		query.FactPolicy.EntityStatus,
		query.FactPolicy.EntityRelationStatus,
		query.FactPolicy.IndustryChainReviewStatus,
		query.FactPolicy.MembershipReviewStatus,
		query.FactPolicy.MembershipStatus,
		query.FactPolicy.GraphEdgeReviewStatus,
		query.FactPolicy.GraphEdgeStatus,
	}
	if err := s.validateResearchGraphReferences(
		ctx,
		query,
		relationTypes,
	); err != nil {
		return biz.ResearchGraphSubgraph{}, err
	}
	var nodeCount, edgeCount int64
	if err := s.db.QueryRowContext(
		ctx,
		researchGraphCTE+`
		SELECT
		    (SELECT count(*) FROM selected_entities),
		    (SELECT count(*) FROM selected_entity_relations)
		      + (SELECT count(*) FROM selected_graph_edges)
	`,
		args...,
	).Scan(&nodeCount, &edgeCount); err != nil {
		return biz.ResearchGraphSubgraph{}, err
	}
	if nodeCount > int64(query.NodeBudget) {
		maximum := int64(query.NodeBudget)
		return biz.ResearchGraphSubgraph{}, &biz.ResearchGraphResourceLimitError{
			Reason:        "research graph result exceeds the requested node budget",
			Component:     "research_graph_nodes",
			MaxRows:       &maximum,
			RetryGuidance: "reduce_depth_relation_types_or_chain_scope",
		}
	}
	if edgeCount > int64(query.EdgeBudget) {
		maximum := int64(query.EdgeBudget)
		return biz.ResearchGraphSubgraph{}, &biz.ResearchGraphResourceLimitError{
			Reason:        "research graph result exceeds the requested edge budget",
			Component:     "research_graph_edges",
			MaxRows:       &maximum,
			RetryGuidance: "reduce_depth_relation_types_or_chain_scope",
		}
	}
	var payload []byte
	if err := s.db.QueryRowContext(
		ctx,
		researchGraphCTE+`
		SELECT jsonb_build_object(
			    'actual_depth', COALESCE((SELECT max(depth) FROM reached_entities), 0),
		    'entities', COALESCE((
		        SELECT jsonb_agg(jsonb_build_object(
		            'entity_id', entity.id,
		            'entity_type', entity.entity_type,
		            'name', entity.name,
		            'canonical_name', entity.canonical_name,
		            'aliases', entity.aliases,
		            'status', entity.status
		        ) ORDER BY entity.entity_type, entity.canonical_name, entity.id)
		        FROM selected_entities entity
		    ), '[]'::jsonb),
		    'relation_definitions', COALESCE((
		        SELECT jsonb_agg(jsonb_build_object(
		            'relation_type', relation_type,
		            'direction', 'directed'
		        ) ORDER BY relation_type)
		        FROM selected_relation_types
		    ), '[]'::jsonb),
		    'entity_relations', COALESCE((
		        SELECT jsonb_agg(jsonb_build_object(
		            'entity_relation_id', relation.id,
		            'from_entity_id', relation.from_entity_id,
		            'to_entity_id', relation.to_entity_id,
		            'relation_type', relation.relation_type,
		            'status', relation.status
		        ) ORDER BY relation.relation_type, relation.from_entity_id, relation.to_entity_id, relation.id)
		        FROM selected_entity_relations relation
		    ), '[]'::jsonb),
		    'industry_chains', COALESCE((
		        SELECT jsonb_agg(jsonb_build_object(
		            'industry_chain_entity_id', definition.entity_id,
		            'scope', definition.scope,
		            'target_output', definition.target_output,
		            'end_use', definition.end_use,
		            'geography', definition.geography,
		            'as_of_date', definition.as_of_date,
		            'review_status', definition.review_status
		        ) ORDER BY definition.entity_id)
		        FROM selected_industry_chains definition
		    ), '[]'::jsonb),
		    'industry_chain_memberships', COALESCE((
		        SELECT jsonb_agg(jsonb_build_object(
		            'industry_chain_entity_id', membership.industry_chain_entity_id,
		            'chain_node_entity_id', membership.chain_node_entity_id,
		            'position', membership.position,
		            'contextual_stage', membership.contextual_stage,
		            'review_status', membership.review_status,
		            'status', membership.status
		        ) ORDER BY membership.industry_chain_entity_id, membership.position, membership.chain_node_entity_id)
		        FROM selected_memberships membership
		    ), '[]'::jsonb),
		    'industry_chain_graph_edges', COALESCE((
		        SELECT jsonb_agg(jsonb_build_object(
		            'industry_chain_graph_edge_id', edge.id,
		            'industry_chain_entity_id', edge.industry_chain_entity_id,
		            'from_chain_node_entity_id', edge.from_chain_node_entity_id,
		            'to_chain_node_entity_id', edge.to_chain_node_entity_id,
		            'relation_type', edge.relation_type,
		            'mechanism', edge.mechanism,
		            'condition_note', edge.condition_note,
		            'segment_kind', edge.segment_kind,
		            'omitted_step_note', edge.omitted_step_note,
		            'review_status', edge.review_status,
		            'status', edge.status
		        ) ORDER BY edge.industry_chain_entity_id, edge.from_chain_node_entity_id, edge.to_chain_node_entity_id, edge.id)
		        FROM selected_graph_edges edge
		    ), '[]'::jsonb)
		)
	`,
		args...,
	).Scan(&payload); err != nil {
		return biz.ResearchGraphSubgraph{}, err
	}
	var graph biz.ResearchGraphSubgraph
	if err := strictDecodeResearchGraph(payload, &graph); err != nil {
		return biz.ResearchGraphSubgraph{}, err
	}
	if err := validatePersistedResearchGraph(graph, query.MaxDepth); err != nil {
		return biz.ResearchGraphSubgraph{}, err
	}
	return graph, nil
}

func validatePersistedResearchGraph(graph biz.ResearchGraphSubgraph, maxDepth int) error {
	if graph.ActualDepth < 0 || graph.ActualDepth > maxDepth {
		return errors.New("persisted Research Graph depth violates invariants")
	}
	entities := make(map[string]struct{}, len(graph.Entities))
	for _, entity := range graph.Entities {
		if !bizidentity.IsUUID(entity.EntityID) || strings.TrimSpace(entity.EntityType) == "" ||
			strings.TrimSpace(entity.Name) == "" || strings.TrimSpace(entity.CanonicalName) == "" ||
			entity.Status != "active" {
			return errors.New("persisted Research Graph Entity violates invariants")
		}
		if _, duplicate := entities[entity.EntityID]; duplicate {
			return errors.New("persisted Research Graph Entity identity is duplicated")
		}
		entities[entity.EntityID] = struct{}{}
	}
	relationTypes := make(map[string]struct{}, len(graph.RelationDefinitions))
	for _, definition := range graph.RelationDefinitions {
		if strings.TrimSpace(definition.RelationType) == "" || definition.Direction != "directed" {
			return errors.New("persisted Research Graph relation definition violates invariants")
		}
		if _, duplicate := relationTypes[definition.RelationType]; duplicate {
			return errors.New("persisted Research Graph relation definition is duplicated")
		}
		relationTypes[definition.RelationType] = struct{}{}
	}
	relationIDs := make(map[string]struct{}, len(graph.EntityRelations))
	for _, relation := range graph.EntityRelations {
		if !bizidentity.IsUUID(relation.EntityRelationID) || relation.Status != "active" {
			return errors.New("persisted Research Graph Entity relation violates invariants")
		}
		if _, ok := entities[relation.FromEntityID]; !ok {
			return errors.New("persisted Research Graph Entity relation source is unavailable")
		}
		if _, ok := entities[relation.ToEntityID]; !ok {
			return errors.New("persisted Research Graph Entity relation target is unavailable")
		}
		if _, ok := relationTypes[relation.RelationType]; !ok {
			return errors.New("persisted Research Graph Entity relation type is unavailable")
		}
		if _, duplicate := relationIDs[relation.EntityRelationID]; duplicate {
			return errors.New("persisted Research Graph Entity relation identity is duplicated")
		}
		relationIDs[relation.EntityRelationID] = struct{}{}
	}
	chains := make(map[string]struct{}, len(graph.IndustryChains))
	for _, chain := range graph.IndustryChains {
		if !bizidentity.IsUUID(chain.IndustryChainEntityID) || chain.ReviewStatus != "approved" ||
			strings.TrimSpace(chain.Scope) == "" || strings.TrimSpace(chain.TargetOutput) == "" ||
			strings.TrimSpace(chain.EndUse) == "" || strings.TrimSpace(chain.Geography) == "" ||
			strings.TrimSpace(chain.AsOfDate) == "" {
			return errors.New("persisted Research Graph Industry Chain violates invariants")
		}
		if _, ok := entities[chain.IndustryChainEntityID]; !ok {
			return errors.New("persisted Research Graph Industry Chain Entity is unavailable")
		}
		chains[chain.IndustryChainEntityID] = struct{}{}
	}
	for _, membership := range graph.IndustryChainMemberships {
		if membership.Position <= 0 || !oneOfResearchGraph(membership.ContextualStage, "upstream", "midstream", "downstream") ||
			membership.ReviewStatus != "approved" || membership.Status != "active" {
			return errors.New("persisted Research Graph membership violates invariants")
		}
		if _, ok := chains[membership.IndustryChainEntityID]; !ok {
			return errors.New("persisted Research Graph membership Chain is unavailable")
		}
		if _, ok := entities[membership.ChainNodeEntityID]; !ok {
			return errors.New("persisted Research Graph membership Entity is unavailable")
		}
	}
	edgeIDs := make(map[string]struct{}, len(graph.IndustryChainGraphEdges))
	for _, edge := range graph.IndustryChainGraphEdges {
		if !bizidentity.IsUUID(edge.IndustryChainGraphEdgeID) || strings.TrimSpace(edge.Mechanism) == "" ||
			!oneOfResearchGraph(edge.SegmentKind, "direct_candidate", "compressed_candidate") ||
			edge.ReviewStatus != "approved" || edge.Status != "active" || edge.FromChainNodeEntityID == edge.ToChainNodeEntityID {
			return errors.New("persisted Research Graph Industry Chain edge violates invariants")
		}
		if _, ok := chains[edge.IndustryChainEntityID]; !ok {
			return errors.New("persisted Research Graph Industry Chain edge Chain is unavailable")
		}
		if _, ok := entities[edge.FromChainNodeEntityID]; !ok {
			return errors.New("persisted Research Graph Industry Chain edge source is unavailable")
		}
		if _, ok := entities[edge.ToChainNodeEntityID]; !ok {
			return errors.New("persisted Research Graph Industry Chain edge target is unavailable")
		}
		if _, ok := relationTypes[edge.RelationType]; !ok {
			return errors.New("persisted Research Graph Industry Chain edge type is unavailable")
		}
		if _, duplicate := edgeIDs[edge.IndustryChainGraphEdgeID]; duplicate {
			return errors.New("persisted Research Graph Industry Chain edge identity is duplicated")
		}
		edgeIDs[edge.IndustryChainGraphEdgeID] = struct{}{}
	}
	return nil
}

func oneOfResearchGraph(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func (s *Store) ResearchReferenceClosure(
	ctx context.Context,
	query biz.ResearchReferenceQuery,
) (biz.ResearchReferenceDictionaries, error) {
	if s == nil || s.db == nil {
		return biz.ResearchReferenceDictionaries{}, errors.New("Entity reference database is required")
	}
	var historicalGap bool
	if err := s.db.QueryRowContext(ctx, `WITH
requested_entities(id) AS (SELECT unnest($2::uuid[])),
requested_relations(id) AS (SELECT unnest($3::uuid[]))
SELECT EXISTS (
    SELECT 1 FROM entity_nodes entity JOIN requested_entities requested ON requested.id = entity.id
    WHERE entity.created_at <= $1 AND entity.updated_at > $1
) OR EXISTS (
    SELECT 1 FROM entity_edges relation JOIN requested_relations requested ON requested.id = relation.id
    WHERE relation.created_at <= $1 AND relation.updated_at > $1
) OR EXISTS (
    SELECT 1 FROM industry_chain_definitions definition JOIN requested_entities requested ON requested.id = definition.entity_id
    WHERE definition.created_at <= $1 AND definition.updated_at > $1
)`, query.AnalysisAsOf, query.EntityIDs, query.EntityRelationIDs).Scan(&historicalGap); err != nil {
		return biz.ResearchReferenceDictionaries{}, fmt.Errorf("check historical Entity reference closure: %w", err)
	}
	if historicalGap {
		return biz.ResearchReferenceDictionaries{}, biz.ErrResearchHistoricalReferencesUnavailable
	}
	var payload []byte
	err := s.db.QueryRowContext(ctx, `WITH
requested_entities(id) AS (SELECT unnest($2::uuid[])),
requested_relations(id) AS (SELECT unnest($3::uuid[])),
requested_relation_types(relation_type) AS (SELECT unnest($4::text[])),
requested_entity_types(type_key) AS (SELECT unnest($5::text[])),
selected_relations AS MATERIALIZED (
    SELECT relation.* FROM entity_edges relation
    JOIN requested_relations requested ON requested.id = relation.id
    WHERE relation.status = 'active' AND relation.created_at <= $1 AND relation.updated_at <= $1
),
selected_entity_ids(id) AS MATERIALIZED (
    SELECT id FROM requested_entities
    UNION SELECT from_entity_id FROM selected_relations
    UNION SELECT to_entity_id FROM selected_relations
),
selected_entities AS MATERIALIZED (
    SELECT entity.* FROM entity_nodes entity
    JOIN selected_entity_ids selected ON selected.id = entity.id
    WHERE entity.status = 'active' AND entity.created_at <= $1 AND entity.updated_at <= $1
),
selected_entity_type_definitions AS MATERIALIZED (
    SELECT definition.* FROM entity_type_definitions definition
    JOIN (
        SELECT DISTINCT entity_type type_key FROM selected_entities
        UNION SELECT type_key FROM requested_entity_types
    ) selected ON selected.type_key = definition.type_key
    WHERE definition.status = 'active'
      AND definition.created_at <= $1
),
selected_industry_chains AS MATERIALIZED (
    SELECT definition.* FROM industry_chain_definitions definition
    JOIN selected_entity_ids selected ON selected.id = definition.entity_id
    WHERE definition.review_status = 'approved' AND definition.created_at <= $1 AND definition.updated_at <= $1
),
selected_relation_types(relation_type) AS (
    SELECT relation_type FROM requested_relation_types
    UNION SELECT relation_type FROM selected_relations
)
SELECT jsonb_build_object(
    'entities', COALESCE((SELECT jsonb_agg(jsonb_build_object(
        'entity_id', entity.id, 'entity_type', entity.entity_type, 'name', entity.name,
        'canonical_name', entity.canonical_name, 'aliases', entity.aliases, 'status', entity.status
    ) ORDER BY entity.entity_type, entity.canonical_name, entity.id) FROM selected_entities entity), '[]'::jsonb),
    'relation_definitions', COALESCE((SELECT jsonb_agg(jsonb_build_object(
        'relation_type', definition.relation_type, 'direction', 'directed'
    ) ORDER BY definition.relation_type) FROM selected_relation_types definition), '[]'::jsonb),
    'entity_relations', COALESCE((SELECT jsonb_agg(jsonb_build_object(
        'entity_relation_id', relation.id, 'from_entity_id', relation.from_entity_id,
        'to_entity_id', relation.to_entity_id, 'relation_type', relation.relation_type, 'status', relation.status
    ) ORDER BY relation.relation_type, relation.from_entity_id, relation.to_entity_id, relation.id) FROM selected_relations relation), '[]'::jsonb),
    'industry_chains', COALESCE((SELECT jsonb_agg(jsonb_build_object(
        'industry_chain_entity_id', definition.entity_id, 'scope', definition.scope,
        'target_output', definition.target_output, 'end_use', definition.end_use,
        'geography', definition.geography, 'as_of_date', definition.as_of_date,
        'review_status', definition.review_status
    ) ORDER BY definition.entity_id) FROM selected_industry_chains definition), '[]'::jsonb),
    'industry_chain_memberships', '[]'::jsonb,
    'industry_chain_graph_edges', '[]'::jsonb,
    'entity_type_definitions', COALESCE((SELECT jsonb_agg(jsonb_build_object(
        'type_key', definition.type_key, 'version', definition.version, 'name_zh', definition.name_zh,
        'name_en', definition.name_en, 'business_definition', definition.business_definition,
        'inclusion_criteria', definition.inclusion_criteria, 'exclusion_criteria', definition.exclusion_criteria,
		'event_link_allowed', definition.event_link_allowed, 'signal_subject_allowed', definition.signal_subject_allowed,
		'direct_target_mode', definition.direct_target_mode,
		'allowed_event_roles', definition.allowed_event_roles, 'status', definition.status
    ) ORDER BY definition.type_key, definition.version) FROM selected_entity_type_definitions definition), '[]'::jsonb)
)`, query.AnalysisAsOf, query.EntityIDs, query.EntityRelationIDs, query.RelationTypes, query.EntityTypes).Scan(&payload)
	if err != nil {
		return biz.ResearchReferenceDictionaries{}, fmt.Errorf("query Entity reference closure: %w", err)
	}
	var persisted persistedResearchReferenceDictionaries
	if err := strictDecodeResearchGraph(payload, &persisted); err != nil {
		return biz.ResearchReferenceDictionaries{}, err
	}
	dictionaries := biz.ResearchReferenceDictionaries{
		Entities: persisted.Entities, RelationDefinitions: persisted.RelationDefinitions,
		EntityRelations: persisted.EntityRelations, IndustryChains: persisted.IndustryChains,
		IndustryChainMemberships: persisted.IndustryChainMemberships,
		IndustryChainGraphEdges:  persisted.IndustryChainGraphEdges,
		EntityTypeDefinitions:    make([]biz.ResearchEntityTypeContext, 0, len(persisted.EntityTypeDefinitions)),
	}
	for _, definition := range persisted.EntityTypeDefinitions {
		candidate := biz.EntityTypeDefinition{
			TypeKey: definition.TypeKey, Version: definition.Version, NameZH: definition.NameZH,
			NameEN: definition.NameEN, BusinessDefinition: definition.BusinessDefinition,
			InclusionCriteria: definition.InclusionCriteria, ExclusionCriteria: definition.ExclusionCriteria,
			EventLinkAllowed: definition.EventLinkAllowed, SignalSubjectAllowed: definition.SignalSubjectAllowed,
			DirectTargetMode: definition.DirectTargetMode, AllowedEventRoles: definition.AllowedEventRoles,
			Status: biz.EntityTypeDefinitionStatus(definition.Status),
		}
		if err := candidate.Validate(); err != nil || candidate.Status != biz.EntityTypeDefinitionActive {
			return biz.ResearchReferenceDictionaries{}, errors.New("persisted Research Entity Type Definition violates invariants")
		}
		dictionaries.EntityTypeDefinitions = append(dictionaries.EntityTypeDefinitions, definition.ResearchEntityTypeContext)
	}
	if err := validateResearchReferenceDictionaries(dictionaries); err != nil {
		return biz.ResearchReferenceDictionaries{}, err
	}
	return dictionaries, nil
}

type persistedResearchReferenceDictionaries struct {
	Entities                 []biz.ResearchGraphEntity         `json:"entities"`
	RelationDefinitions      []biz.ResearchGraphRelation       `json:"relation_definitions"`
	EntityRelations          []biz.ResearchGraphEntityRelation `json:"entity_relations"`
	IndustryChains           []biz.ResearchGraphIndustryChain  `json:"industry_chains"`
	IndustryChainMemberships []biz.ResearchGraphMembership     `json:"industry_chain_memberships"`
	IndustryChainGraphEdges  []biz.ResearchGraphIndustryEdge   `json:"industry_chain_graph_edges"`
	EntityTypeDefinitions    []persistedResearchEntityType     `json:"entity_type_definitions"`
}

type persistedResearchEntityType struct {
	biz.ResearchEntityTypeContext
	AllowedEventRoles []string `json:"allowed_event_roles"`
}

func validateResearchReferenceDictionaries(value biz.ResearchReferenceDictionaries) error {
	entities := make(map[string]struct{}, len(value.Entities))
	entityTypes := make(map[string]struct{}, len(value.EntityTypeDefinitions))
	for _, definition := range value.EntityTypeDefinitions {
		entityTypes[definition.TypeKey] = struct{}{}
	}
	for _, entity := range value.Entities {
		if !bizidentity.IsUUID(entity.EntityID) || strings.TrimSpace(entity.Name) == "" ||
			strings.TrimSpace(entity.CanonicalName) == "" || entity.Status != "active" {
			return errors.New("persisted Research Entity violates invariants")
		}
		if _, ok := entityTypes[entity.EntityType]; !ok {
			return errors.New("persisted Research Entity type reference is unavailable")
		}
		entities[entity.EntityID] = struct{}{}
	}
	relationTypes := make(map[string]struct{}, len(value.RelationDefinitions))
	for _, definition := range value.RelationDefinitions {
		if strings.TrimSpace(definition.RelationType) == "" || definition.Direction != "directed" {
			return errors.New("persisted Research relation definition violates invariants")
		}
		relationTypes[definition.RelationType] = struct{}{}
	}
	for _, relation := range value.EntityRelations {
		if !bizidentity.IsUUID(relation.EntityRelationID) || relation.Status != "active" {
			return errors.New("persisted Research Entity relation violates invariants")
		}
		if _, ok := entities[relation.FromEntityID]; !ok {
			return errors.New("persisted Research Entity relation source is unavailable")
		}
		if _, ok := entities[relation.ToEntityID]; !ok {
			return errors.New("persisted Research Entity relation target is unavailable")
		}
		if _, ok := relationTypes[relation.RelationType]; !ok {
			return errors.New("persisted Research Entity relation type is unavailable")
		}
	}
	for _, chain := range value.IndustryChains {
		if _, ok := entities[chain.IndustryChainEntityID]; !ok || chain.ReviewStatus != "approved" ||
			strings.TrimSpace(chain.Scope) == "" || strings.TrimSpace(chain.TargetOutput) == "" ||
			strings.TrimSpace(chain.AsOfDate) == "" {
			return errors.New("persisted Research Industry Chain violates invariants")
		}
	}
	return nil
}

func (s *Store) validateResearchGraphReferences(
	ctx context.Context,
	query biz.ResearchGraphQuery,
	relationTypes []string,
) error {
	var seedCount, relationTypeCount, chainCount int
	if err := s.db.QueryRowContext(ctx, `
		SELECT
		    (
		        SELECT count(*)
		        FROM entity_nodes entity
		        WHERE entity.id = ANY($2::uuid[])
		          AND entity.status = $5
		          AND entity.created_at <= $1
		          AND entity.updated_at <= $1
		    ),
		    (
		        SELECT count(DISTINCT requested.relation_type)
		        FROM unnest($3::text[]) requested(relation_type)
		        WHERE EXISTS (
		            SELECT 1
		            FROM entity_edges relation
		            JOIN entity_nodes from_entity
		              ON from_entity.id = relation.from_entity_id
		             AND from_entity.status = $5
		             AND from_entity.created_at <= $1
		             AND from_entity.updated_at <= $1
		            JOIN entity_nodes to_entity
		              ON to_entity.id = relation.to_entity_id
		             AND to_entity.status = $5
		             AND to_entity.created_at <= $1
		             AND to_entity.updated_at <= $1
		            WHERE relation.relation_type = requested.relation_type
		              AND relation.status = $6
		              AND relation.created_at <= $1
		              AND relation.updated_at <= $1
		            UNION ALL
		            SELECT 1
		            FROM industry_chain_graph_edges edge
		            JOIN industry_chain_definitions definition
		              ON definition.entity_id = edge.industry_chain_entity_id
		             AND definition.review_status = $7
		             AND definition.created_at <= $1
		             AND definition.updated_at <= $1
		            JOIN industry_chain_node_memberships from_membership
		              ON from_membership.industry_chain_entity_id = edge.industry_chain_entity_id
		             AND from_membership.chain_node_entity_id = edge.from_chain_node_entity_id
		             AND from_membership.review_status = $8
		             AND from_membership.status = $9
		             AND from_membership.created_at <= $1
		             AND from_membership.updated_at <= $1
		            JOIN industry_chain_node_memberships to_membership
		              ON to_membership.industry_chain_entity_id = edge.industry_chain_entity_id
		             AND to_membership.chain_node_entity_id = edge.to_chain_node_entity_id
		             AND to_membership.review_status = $8
		             AND to_membership.status = $9
		             AND to_membership.created_at <= $1
		             AND to_membership.updated_at <= $1
		            JOIN entity_nodes from_entity
		              ON from_entity.id = edge.from_chain_node_entity_id
		             AND from_entity.status = $5
		             AND from_entity.created_at <= $1
		             AND from_entity.updated_at <= $1
		            JOIN entity_nodes to_entity
		              ON to_entity.id = edge.to_chain_node_entity_id
		             AND to_entity.status = $5
		             AND to_entity.created_at <= $1
		             AND to_entity.updated_at <= $1
		            WHERE edge.relation_type = requested.relation_type
		              AND edge.status = $11
		              AND edge.review_status = $10
		              AND edge.created_at <= $1
		              AND edge.updated_at <= $1
		              AND ($4::uuid IS NULL OR edge.industry_chain_entity_id = $4::uuid)
		        )
		    ),
		    CASE
		        WHEN $4::uuid IS NULL THEN 1
		        ELSE (
		            SELECT count(*)
		            FROM industry_chain_definitions definition
		            WHERE definition.entity_id = $4::uuid
		              AND definition.review_status = $7
		              AND definition.created_at <= $1
		              AND definition.updated_at <= $1
		        )
		    END
	`,
		query.AnalysisAsOf,
		query.SeedEntityIDs,
		relationTypes,
		query.IndustryChainEntityID,
		query.FactPolicy.EntityStatus,
		query.FactPolicy.EntityRelationStatus,
		query.FactPolicy.IndustryChainReviewStatus,
		query.FactPolicy.MembershipReviewStatus,
		query.FactPolicy.MembershipStatus,
		query.FactPolicy.GraphEdgeReviewStatus,
		query.FactPolicy.GraphEdgeStatus,
	).Scan(&seedCount, &relationTypeCount, &chainCount); err != nil {
		return err
	}
	if seedCount != len(query.SeedEntityIDs) {
		return &biz.ResearchGraphValidationError{Reason: "one or more seed entities are unavailable"}
	}
	if relationTypeCount != len(relationTypes) {
		return &biz.ResearchGraphValidationError{Reason: "one or more relation types are unavailable"}
	}
	if chainCount != 1 {
		return &biz.ResearchGraphValidationError{Reason: "industry chain scope is unavailable"}
	}
	return nil
}

func strictDecodeResearchGraph(payload []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode persisted Research graph: %w", err)
	}
	if decoder.More() {
		return errors.New("persisted Research graph contains trailing JSON")
	}
	return nil
}

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, fmt.Errorf("Entity database is required")
	}
	return &Store{db: db}, nil
}

func (s *Store) UpsertBenchmarkObservation(ctx context.Context, observation biz.BenchmarkObservation) (biz.BenchmarkObservationWriteResult, error) {
	if err := observation.Validate(); err != nil {
		return biz.BenchmarkObservationWriteResult{}, err
	}
	if err := s.ensureBenchmarkEntity(ctx, observation.BenchmarkEntityID); err != nil {
		return biz.BenchmarkObservationWriteResult{}, err
	}
	row := s.db.QueryRowContext(ctx, `WITH upsert AS (INSERT INTO benchmark_observations (id, benchmark_entity_id, observed_at, value, unit, source_name, source_url, external_series_code, quality_status) VALUES ($1,$2,$3,$4::numeric,$5,$6,$7,$8,$9) ON CONFLICT (benchmark_entity_id, observed_at, source_name) DO UPDATE SET value=EXCLUDED.value, unit=EXCLUDED.unit, source_url=EXCLUDED.source_url, external_series_code=EXCLUDED.external_series_code, quality_status=EXCLUDED.quality_status, updated_at=now() RETURNING id, benchmark_entity_id, observed_at, value::text, unit, source_name, source_url, external_series_code, quality_status, xmax=0 AS inserted) SELECT id, benchmark_entity_id, observed_at, value, unit, source_name, source_url, external_series_code, quality_status, inserted FROM upsert`, bizidentity.NormalizeUUID(observation.ID), bizidentity.NormalizeUUID(observation.BenchmarkEntityID), observation.ObservedAt, observation.Value, observation.Unit, observation.SourceName, observation.SourceURL, observation.ExternalSeriesCode, observation.QualityStatus)
	saved, created, err := scanBenchmarkObservationWrite(row)
	if err != nil {
		return biz.BenchmarkObservationWriteResult{}, fmt.Errorf("upsert benchmark observation: %w", err)
	}
	return biz.BenchmarkObservationWriteResult{Observation: saved, Created: created}, nil
}

func (s *Store) ListBenchmarkObservations(ctx context.Context, filter biz.BenchmarkObservationFilter) ([]biz.BenchmarkObservation, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, benchmark_entity_id, observed_at, value::text, unit, source_name, source_url, external_series_code, quality_status FROM benchmark_observations WHERE ($1::uuid IS NULL OR benchmark_entity_id=$1::uuid) ORDER BY observed_at DESC, source_name, id LIMIT CASE WHEN $2 > 0 THEN $2 ELSE 2147483647 END`, optionalUUID(filter.BenchmarkEntityID), filter.Limit)
	if err != nil {
		return nil, fmt.Errorf("query benchmark observations: %w", err)
	}
	defer rows.Close()
	items := make([]biz.BenchmarkObservation, 0)
	for rows.Next() {
		item, err := scanBenchmarkObservation(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate benchmark observations: %w", err)
	}
	return items, nil
}

func (s *Store) ensureBenchmarkEntity(ctx context.Context, id string) error {
	var entityType biz.EntityType
	if err := s.db.QueryRowContext(ctx, `SELECT entity_type FROM entity_nodes WHERE id=$1`, bizidentity.NormalizeUUID(id)).Scan(&entityType); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("benchmark entity %q not found", id)
		}
		return fmt.Errorf("query benchmark entity %q: %w", id, err)
	}
	if entityType != biz.EntityTypeBenchmark {
		return fmt.Errorf("entity %q type %q is not benchmark", id, entityType)
	}
	return nil
}

type scanner interface{ Scan(...any) error }

func scanBenchmarkObservation(row scanner) (biz.BenchmarkObservation, error) {
	var item biz.BenchmarkObservation
	if err := row.Scan(&item.ID, &item.BenchmarkEntityID, &item.ObservedAt, &item.Value, &item.Unit, &item.SourceName, &item.SourceURL, &item.ExternalSeriesCode, &item.QualityStatus); err != nil {
		return biz.BenchmarkObservation{}, fmt.Errorf("scan benchmark observation: %w", err)
	}
	if err := item.Validate(); err != nil {
		return biz.BenchmarkObservation{}, fmt.Errorf("validate persisted benchmark observation: %w", err)
	}
	return item, nil
}
func scanBenchmarkObservationWrite(row scanner) (biz.BenchmarkObservation, bool, error) {
	var item biz.BenchmarkObservation
	var created bool
	if err := row.Scan(&item.ID, &item.BenchmarkEntityID, &item.ObservedAt, &item.Value, &item.Unit, &item.SourceName, &item.SourceURL, &item.ExternalSeriesCode, &item.QualityStatus, &created); err != nil {
		return biz.BenchmarkObservation{}, false, err
	}
	if err := item.Validate(); err != nil {
		return biz.BenchmarkObservation{}, false, fmt.Errorf("validate persisted benchmark observation: %w", err)
	}
	return item, created, nil
}
func optionalUUID(value string) any {
	if value == "" {
		return nil
	}
	return bizidentity.NormalizeUUID(value)
}
