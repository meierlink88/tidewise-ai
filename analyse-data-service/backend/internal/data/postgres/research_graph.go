package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchgraph"
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

type ResearchGraphStore struct {
	db *sql.DB
}

func NewResearchGraphStore(db *sql.DB) *ResearchGraphStore {
	return &ResearchGraphStore{db: db}
}

func (s *ResearchGraphStore) Search(
	ctx context.Context,
	query researchgraph.Query,
) (researchgraph.Subgraph, error) {
	if s == nil || s.db == nil {
		return researchgraph.Subgraph{}, errors.New("research graph database is required")
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
		return researchgraph.Subgraph{}, err
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
		return researchgraph.Subgraph{}, err
	}
	if nodeCount > int64(query.NodeBudget) {
		maximum := int64(query.NodeBudget)
		return researchgraph.Subgraph{}, &researchgraph.ResourceLimitError{
			Reason:        "research graph result exceeds the requested node budget",
			Component:     "research_graph_nodes",
			MaxRows:       &maximum,
			RetryGuidance: "reduce_depth_relation_types_or_chain_scope",
		}
	}
	if edgeCount > int64(query.EdgeBudget) {
		maximum := int64(query.EdgeBudget)
		return researchgraph.Subgraph{}, &researchgraph.ResourceLimitError{
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
		return researchgraph.Subgraph{}, err
	}
	var graph researchgraph.Subgraph
	if err := strictDecodeResearchContext(payload, &graph); err != nil {
		return researchgraph.Subgraph{}, err
	}
	return graph, nil
}

func (s *ResearchGraphStore) validateResearchGraphReferences(
	ctx context.Context,
	query researchgraph.Query,
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
		return &researchgraph.ValidationError{Reason: "one or more seed entities are unavailable"}
	}
	if relationTypeCount != len(relationTypes) {
		return &researchgraph.ValidationError{Reason: "one or more relation types are unavailable"}
	}
	if chainCount != 1 {
		return &researchgraph.ValidationError{Reason: "industry chain scope is unavailable"}
	}
	return nil
}

var _ researchgraph.Store = (*ResearchGraphStore)(nil)
