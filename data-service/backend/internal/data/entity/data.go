package entity

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	biz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity"
	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

const researchGraphCTE = `
	WITH RECURSIVE
	all_entities(id, entity_type, name, canonical_name, aliases, status, created_at, updated_at) AS MATERIALIZED (
	    SELECT id, entity_type::text, name, canonical_name, aliases, status::text, created_at, updated_at
	    FROM entity_nodes
	    UNION ALL
	    SELECT id, 'industry', name, name, aliases, 'active', created_at, updated_at
	    FROM industry
	    UNION ALL
	    SELECT id, 'concept', name, name, aliases, 'active', created_at, updated_at
	    FROM concept
	    UNION ALL
	    SELECT id, 'chain_node', name, name, aliases, 'active', created_at, updated_at
	    FROM chain_node
	    UNION ALL
	    SELECT id, 'industry_chain', name, name, aliases, 'active', created_at, updated_at
	    FROM industry_chain
	),
	all_entity_relations(id, from_entity_id, to_entity_id, relation_type, status, created_at, updated_at) AS MATERIALIZED (
	    SELECT id, from_entity_id, to_entity_id, relation_type, status::text, created_at, updated_at
	    FROM entity_edges
	    UNION ALL
	    SELECT id, industry_chain_id, industry_id, 'mapped_to_industry', 'active', created_at, created_at
	    FROM industry_chain_industry_links
	    UNION ALL
	    SELECT id, industry_chain_id, concept_id, 'mapped_to_concept', 'active', created_at, created_at
	    FROM industry_chain_concept_links
	),
	requested_seeds(entity_id) AS (
	    SELECT unnest($2::text[])
	),
	requested_filters(relation_type, direction) AS (
	    SELECT *
	    FROM unnest($3::text[], $4::text[])
	),
	eligible_edges AS NOT MATERIALIZED (
	    SELECT
	        'entity_relation'::text edge_kind,
	        relation.id edge_id,
	        NULL::text industry_chain_id,
	        relation.from_entity_id,
	        relation.to_entity_id,
	        relation.relation_type,
	        ''::text mechanism,
	        NULL::text condition_note,
	        ''::text segment_kind,
	        NULL::text omitted_step_note
	    FROM all_entity_relations relation
	    JOIN all_entities from_entity
	      ON from_entity.id = relation.from_entity_id
	     AND from_entity.status = $9
	     AND from_entity.created_at <= $1
	     AND from_entity.updated_at <= $1
	    JOIN all_entities to_entity
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
	        edge.id::text,
	        edge.industry_chain_id,
	        edge.from_chain_node_id,
	        edge.to_chain_node_id,
	        edge.relation_type,
	        edge.mechanism,
	        edge.condition_note,
	        edge.segment_kind,
	        edge.omitted_step_note
	    FROM industry_chain_graph_edges edge
	    JOIN industry_chain definition
	      ON definition.id = edge.industry_chain_id
	     AND definition.review_status = $11
	     AND definition.created_at <= $1
	     AND definition.updated_at <= $1
	    JOIN industry_chain_node_memberships from_membership
	      ON from_membership.industry_chain_id = edge.industry_chain_id
	     AND from_membership.chain_node_id = edge.from_chain_node_id
	     AND from_membership.review_status = $12
	     AND from_membership.status = $13
	     AND from_membership.created_at <= $1
	     AND from_membership.updated_at <= $1
	    JOIN industry_chain_node_memberships to_membership
	      ON to_membership.industry_chain_id = edge.industry_chain_id
	     AND to_membership.chain_node_id = edge.to_chain_node_id
	     AND to_membership.review_status = $12
	     AND to_membership.status = $13
	     AND to_membership.created_at <= $1
	     AND to_membership.updated_at <= $1
	    JOIN all_entities from_entity
	      ON from_entity.id = edge.from_chain_node_id
	     AND from_entity.status = $9
	     AND from_entity.created_at <= $1
	     AND from_entity.updated_at <= $1
	    JOIN all_entities to_entity
	      ON to_entity.id = edge.to_chain_node_id
	     AND to_entity.status = $9
	     AND to_entity.created_at <= $1
	     AND to_entity.updated_at <= $1
	    JOIN requested_filters filter
	      ON filter.relation_type = edge.relation_type
	    WHERE edge.status = $15
	      AND edge.review_status = $14
	      AND edge.created_at <= $1
	      AND edge.updated_at <= $1
	      AND ($6::text IS NULL OR edge.industry_chain_id = $6::text)
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
	            '{}'::text[]
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
	    FROM all_entity_relations relation
	    JOIN used_edges used
	      ON used.edge_kind = 'entity_relation'
	     AND used.edge_id = relation.id
	),
	selected_graph_edges AS MATERIALIZED (
	    SELECT edge.*
	    FROM industry_chain_graph_edges edge
	    JOIN used_edges used
	      ON used.edge_kind = 'industry_chain_graph_edge'
	     AND used.edge_id = edge.id::text
	),
	selected_chain_ids(industry_chain_id) AS MATERIALIZED (
	    SELECT DISTINCT industry_chain_id
	    FROM selected_graph_edges
	    UNION
	    SELECT $6::text
	    WHERE $6::text IS NOT NULL
	),
	selected_entity_ids(entity_id) AS MATERIALIZED (
	    SELECT entity_id FROM reached_entities
	    UNION
	    SELECT industry_chain_id FROM selected_chain_ids
	),
	selected_entities AS MATERIALIZED (
	    SELECT entity.*
	    FROM all_entities entity
	    JOIN selected_entity_ids selected ON selected.entity_id = entity.id
	    WHERE entity.status = $9
	      AND entity.created_at <= $1
	      AND entity.updated_at <= $1
	),
	selected_industry_chains AS MATERIALIZED (
	    SELECT definition.*
	    FROM industry_chain definition
	    JOIN selected_chain_ids selected
	      ON selected.industry_chain_id = definition.id
	    WHERE definition.review_status = $11
	      AND definition.created_at <= $1
	      AND definition.updated_at <= $1
	),
	selected_memberships AS MATERIALIZED (
	    SELECT membership.*
	    FROM industry_chain_node_memberships membership
	    JOIN selected_chain_ids selected_chain
	      ON selected_chain.industry_chain_id = membership.industry_chain_id
	    JOIN reached_entities selected_node
	      ON selected_node.entity_id = membership.chain_node_id
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
	if containsCountrySeed(query.SeedEntityIDs) {
		return s.searchCountryResearchGraph(ctx, query)
	}
	if containsOrganizationSeed(query.SeedEntityIDs) {
		return s.searchOrganizationResearchGraph(ctx, query)
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
		query.IndustryChainID,
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
		            'industry_chain_id', definition.id,
		            'scope', definition.scope,
		            'target_output', definition.target_output,
			            'end_use', definition.end_use,
			            'geography', definition.geography,
			            'primary_country_id', definition.primary_country_id,
			            'as_of_date', definition.as_of_date,
		            'review_status', definition.review_status
		        ) ORDER BY definition.id)
		        FROM selected_industry_chains definition
		    ), '[]'::jsonb),
		    'industry_chain_memberships', COALESCE((
		        SELECT jsonb_agg(jsonb_build_object(
		            'industry_chain_id', membership.industry_chain_id,
		            'chain_node_id', membership.chain_node_id,
		            'position', membership.position,
		            'contextual_stage', membership.contextual_stage,
		            'review_status', membership.review_status,
		            'status', membership.status
		        ) ORDER BY membership.industry_chain_id, membership.position, membership.chain_node_id)
		        FROM selected_memberships membership
		    ), '[]'::jsonb),
		    'industry_chain_graph_edges', COALESCE((
		        SELECT jsonb_agg(jsonb_build_object(
		            'industry_chain_graph_edge_id', edge.id,
		            'industry_chain_id', edge.industry_chain_id,
		            'from_chain_node_id', edge.from_chain_node_id,
		            'to_chain_node_id', edge.to_chain_node_id,
		            'relation_type', edge.relation_type,
		            'mechanism', edge.mechanism,
		            'condition_note', edge.condition_note,
		            'segment_kind', edge.segment_kind,
		            'omitted_step_note', edge.omitted_step_note,
		            'review_status', edge.review_status,
		            'status', edge.status
		        ) ORDER BY edge.industry_chain_id, edge.from_chain_node_id, edge.to_chain_node_id, edge.id)
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

const countryRegionRelationType = "belongs_to_region"
const organizationMemberRelationType = "has_member"

func containsCountrySeed(ids []string) bool {
	for _, id := range ids {
		if biz.IsCountryID(id) {
			return true
		}
	}
	return false
}

func containsOrganizationSeed(ids []string) bool {
	for _, id := range ids {
		if biz.IsOrganizationID(id) {
			return true
		}
	}
	return false
}

func validResearchGraphIdentity(id, objectType string) bool {
	return biz.ObjectTypeMatchesID(objectType, id)
}

func (s *Store) searchOrganizationResearchGraph(ctx context.Context, query biz.ResearchGraphQuery) (biz.ResearchGraphSubgraph, error) {
	if query.IndustryChainID != nil {
		return biz.ResearchGraphSubgraph{}, &biz.ResearchGraphValidationError{Reason: "Organization seeds cannot use an Industry Chain scope"}
	}
	for _, id := range query.SeedEntityIDs {
		if !biz.IsOrganizationID(id) {
			return biz.ResearchGraphSubgraph{}, &biz.ResearchGraphValidationError{Reason: "Organization and other Object seeds cannot be mixed"}
		}
	}
	includeMembers := false
	for _, filter := range query.RelationFilters {
		if filter.RelationType != organizationMemberRelationType {
			return biz.ResearchGraphSubgraph{}, &biz.ResearchGraphValidationError{Reason: "Organization graph supports only has_member"}
		}
		includeMembers = filter.Direction == biz.ResearchGraphDirectionOutgoing || filter.Direction == biz.ResearchGraphDirectionBoth
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,name_en,created_at,updated_at FROM organizations WHERE id=ANY($1::text[]) ORDER BY code,id`, query.SeedEntityIDs)
	if err != nil {
		return biz.ResearchGraphSubgraph{}, err
	}
	defer rows.Close()
	graph := biz.ResearchGraphSubgraph{Entities: []biz.ResearchGraphEntity{}, RelationDefinitions: []biz.ResearchGraphRelation{{RelationType: organizationMemberRelationType, Direction: "directed"}}, EntityRelations: []biz.ResearchGraphEntityRelation{}, IndustryChains: []biz.ResearchGraphIndustryChain{}, IndustryChainMemberships: []biz.ResearchGraphMembership{}, IndustryChainGraphEdges: []biz.ResearchGraphIndustryEdge{}}
	for rows.Next() {
		var id, name, nameEn string
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &name, &nameEn, &createdAt, &updatedAt); err != nil {
			return graph, err
		}
		if createdAt.After(query.AnalysisAsOf) {
			continue
		}
		if updatedAt.After(query.AnalysisAsOf) {
			return graph, biz.ErrResearchHistoricalReferencesUnavailable
		}
		graph.Entities = append(graph.Entities, biz.ResearchGraphEntity{EntityID: id, EntityType: biz.ObjectTypeOrganization, Name: name, CanonicalName: name, Aliases: []string{nameEn}, Status: "active"})
	}
	if err := rows.Err(); err != nil {
		return graph, err
	}
	if len(graph.Entities) != len(query.SeedEntityIDs) {
		return graph, &biz.ResearchGraphValidationError{Reason: "one or more seed organizations are unavailable"}
	}
	if includeMembers {
		memberRows, err := s.db.QueryContext(ctx, `SELECT member.id,member.organization_id,country.id,country.name,country.name_en FROM organization_members member JOIN countries country ON country.id=member.country_id WHERE member.organization_id=ANY($1::text[]) AND (member.effective_date IS NULL OR member.effective_date <= $2::date) AND (member.expiry_date IS NULL OR member.expiry_date >= $2::date) ORDER BY member.organization_id,country.code,member.id`, query.SeedEntityIDs, query.AnalysisAsOf)
		if err != nil {
			return graph, err
		}
		defer memberRows.Close()
		countries := map[string]biz.ResearchGraphEntity{}
		for memberRows.Next() {
			var memberID string
			var organizationID, countryID, name, nameEn string
			if err := memberRows.Scan(&memberID, &organizationID, &countryID, &name, &nameEn); err != nil {
				return graph, err
			}
			countries[countryID] = biz.ResearchGraphEntity{EntityID: countryID, EntityType: biz.ObjectTypeCountry, Name: name, CanonicalName: name, Aliases: []string{nameEn}, Status: "active"}
			relationID, err := coreid.Derive(biz.EntityRelationIDPrefix, "organization-member", memberID)
			if err != nil {
				return graph, err
			}
			graph.EntityRelations = append(graph.EntityRelations, biz.ResearchGraphEntityRelation{EntityRelationID: relationID, FromEntityID: organizationID, ToEntityID: countryID, RelationType: organizationMemberRelationType, Status: "active"})
		}
		if err := memberRows.Err(); err != nil {
			return graph, err
		}
		for _, country := range countries {
			graph.Entities = append(graph.Entities, country)
		}
		if len(graph.EntityRelations) > 0 {
			graph.ActualDepth = 1
		}
	}
	sort.Slice(graph.Entities, func(i, j int) bool {
		if graph.Entities[i].EntityType != graph.Entities[j].EntityType {
			return graph.Entities[i].EntityType < graph.Entities[j].EntityType
		}
		if graph.Entities[i].CanonicalName != graph.Entities[j].CanonicalName {
			return graph.Entities[i].CanonicalName < graph.Entities[j].CanonicalName
		}
		return graph.Entities[i].EntityID < graph.Entities[j].EntityID
	})
	if len(graph.Entities) > query.NodeBudget {
		maximum := int64(query.NodeBudget)
		return graph, &biz.ResearchGraphResourceLimitError{Reason: "research graph result exceeds the requested node budget", Component: "research_graph_nodes", MaxRows: &maximum, RetryGuidance: "reduce_depth_or_seed_count"}
	}
	if len(graph.EntityRelations) > query.EdgeBudget {
		maximum := int64(query.EdgeBudget)
		return graph, &biz.ResearchGraphResourceLimitError{Reason: "research graph result exceeds the requested edge budget", Component: "research_graph_edges", MaxRows: &maximum, RetryGuidance: "reduce_seed_count"}
	}
	if err := validatePersistedResearchGraph(graph, query.MaxDepth); err != nil {
		return graph, err
	}
	return graph, nil
}

func (s *Store) searchCountryResearchGraph(
	ctx context.Context,
	query biz.ResearchGraphQuery,
) (biz.ResearchGraphSubgraph, error) {
	if query.IndustryChainID != nil {
		return biz.ResearchGraphSubgraph{}, &biz.ResearchGraphValidationError{Reason: "Country seeds cannot use an Industry Chain scope"}
	}
	for _, id := range query.SeedEntityIDs {
		if !biz.IsCountryID(id) {
			return biz.ResearchGraphSubgraph{}, &biz.ResearchGraphValidationError{Reason: "Country and legacy Entity seeds cannot be mixed"}
		}
	}
	includeRegions := false
	for _, filter := range query.RelationFilters {
		if filter.RelationType != countryRegionRelationType {
			return biz.ResearchGraphSubgraph{}, &biz.ResearchGraphValidationError{Reason: "Country graph supports only belongs_to_region"}
		}
		includeRegions = filter.Direction == biz.ResearchGraphDirectionOutgoing || filter.Direction == biz.ResearchGraphDirectionBoth
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, name_en, created_at, updated_at
		FROM countries
		WHERE id = ANY($1::text[])
		ORDER BY code, id`, query.SeedEntityIDs)
	if err != nil {
		return biz.ResearchGraphSubgraph{}, err
	}
	defer rows.Close()
	graph := biz.ResearchGraphSubgraph{
		Entities:                 make([]biz.ResearchGraphEntity, 0, len(query.SeedEntityIDs)),
		RelationDefinitions:      []biz.ResearchGraphRelation{{RelationType: countryRegionRelationType, Direction: "directed"}},
		EntityRelations:          []biz.ResearchGraphEntityRelation{},
		IndustryChains:           []biz.ResearchGraphIndustryChain{},
		IndustryChainMemberships: []biz.ResearchGraphMembership{},
		IndustryChainGraphEdges:  []biz.ResearchGraphIndustryEdge{},
	}
	for rows.Next() {
		var id, name, nameEn string
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &name, &nameEn, &createdAt, &updatedAt); err != nil {
			return biz.ResearchGraphSubgraph{}, err
		}
		if createdAt.After(query.AnalysisAsOf) {
			continue
		}
		if updatedAt.After(query.AnalysisAsOf) {
			return biz.ResearchGraphSubgraph{}, biz.ErrResearchHistoricalReferencesUnavailable
		}
		graph.Entities = append(graph.Entities, biz.ResearchGraphEntity{
			EntityID: id, EntityType: biz.ObjectTypeCountry, Name: name, CanonicalName: name,
			Aliases: []string{nameEn}, Status: "active",
		})
	}
	if err := rows.Err(); err != nil {
		return biz.ResearchGraphSubgraph{}, err
	}
	if len(graph.Entities) != len(query.SeedEntityIDs) {
		return biz.ResearchGraphSubgraph{}, &biz.ResearchGraphValidationError{Reason: "one or more seed countries are unavailable"}
	}

	if includeRegions {
		linkRows, err := s.db.QueryContext(ctx, `
			SELECT link.id, link.country_id, region.id, region.name, region.name_en
			FROM country_region_links link
			JOIN regions region ON region.id = link.region_id
			WHERE link.country_id = ANY($1::text[])
			  AND link.created_at <= $2
			  AND region.created_at <= $2
			ORDER BY link.country_id, region.code, region.id`, query.SeedEntityIDs, query.AnalysisAsOf)
		if err != nil {
			return biz.ResearchGraphSubgraph{}, err
		}
		defer linkRows.Close()
		regions := map[string]biz.ResearchGraphEntity{}
		for linkRows.Next() {
			var linkID string
			var countryID, regionID, name, nameEn string
			if err := linkRows.Scan(&linkID, &countryID, &regionID, &name, &nameEn); err != nil {
				return biz.ResearchGraphSubgraph{}, err
			}
			regions[regionID] = biz.ResearchGraphEntity{
				EntityID: regionID, EntityType: "region", Name: name, CanonicalName: name,
				Aliases: []string{nameEn}, Status: "active",
			}
			relationID, err := coreid.Derive(biz.EntityRelationIDPrefix, "country-region", linkID)
			if err != nil {
				return biz.ResearchGraphSubgraph{}, err
			}
			graph.EntityRelations = append(graph.EntityRelations, biz.ResearchGraphEntityRelation{
				EntityRelationID: relationID,
				FromEntityID:     countryID, ToEntityID: regionID,
				RelationType: countryRegionRelationType, Status: "active",
			})
		}
		if err := linkRows.Err(); err != nil {
			return biz.ResearchGraphSubgraph{}, err
		}
		for _, region := range regions {
			graph.Entities = append(graph.Entities, region)
		}
		if len(graph.EntityRelations) > 0 {
			graph.ActualDepth = 1
		}
	}

	sort.Slice(graph.Entities, func(i, j int) bool {
		if graph.Entities[i].EntityType != graph.Entities[j].EntityType {
			return graph.Entities[i].EntityType < graph.Entities[j].EntityType
		}
		if graph.Entities[i].CanonicalName != graph.Entities[j].CanonicalName {
			return graph.Entities[i].CanonicalName < graph.Entities[j].CanonicalName
		}
		return graph.Entities[i].EntityID < graph.Entities[j].EntityID
	})
	if len(graph.Entities) > query.NodeBudget {
		maximum := int64(query.NodeBudget)
		return biz.ResearchGraphSubgraph{}, &biz.ResearchGraphResourceLimitError{
			Reason: "research graph result exceeds the requested node budget", Component: "research_graph_nodes",
			MaxRows: &maximum, RetryGuidance: "reduce_depth_relation_types_or_chain_scope",
		}
	}
	if len(graph.EntityRelations) > query.EdgeBudget {
		maximum := int64(query.EdgeBudget)
		return biz.ResearchGraphSubgraph{}, &biz.ResearchGraphResourceLimitError{
			Reason: "research graph result exceeds the requested edge budget", Component: "research_graph_edges",
			MaxRows: &maximum, RetryGuidance: "reduce_depth_relation_types_or_chain_scope",
		}
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
		if !validResearchGraphIdentity(entity.EntityID, entity.EntityType) ||
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
		if !biz.IsEntityRelationID(relation.EntityRelationID) || relation.Status != "active" {
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
		if !biz.IsIndustryChainID(chain.IndustryChainID) || chain.ReviewStatus != "approved" ||
			strings.TrimSpace(chain.Scope) == "" || strings.TrimSpace(chain.TargetOutput) == "" ||
			strings.TrimSpace(chain.EndUse) == "" || strings.TrimSpace(chain.Geography) == "" ||
			strings.TrimSpace(chain.AsOfDate) == "" ||
			(chain.PrimaryCountryID != "" && !biz.IsCountryID(chain.PrimaryCountryID)) {
			return errors.New("persisted Research Graph Industry Chain violates invariants")
		}
		if _, ok := entities[chain.IndustryChainID]; !ok {
			return errors.New("persisted Research Graph Industry Chain Entity is unavailable")
		}
		chains[chain.IndustryChainID] = struct{}{}
	}
	for _, membership := range graph.IndustryChainMemberships {
		if membership.Position <= 0 || !oneOfResearchGraph(membership.ContextualStage, "upstream", "midstream", "downstream") ||
			membership.ReviewStatus != "approved" || membership.Status != "active" {
			return errors.New("persisted Research Graph membership violates invariants")
		}
		if _, ok := chains[membership.IndustryChainID]; !ok {
			return errors.New("persisted Research Graph membership Chain is unavailable")
		}
		if _, ok := entities[membership.ChainNodeID]; !ok {
			return errors.New("persisted Research Graph membership Entity is unavailable")
		}
	}
	edgeIDs := make(map[string]struct{}, len(graph.IndustryChainGraphEdges))
	for _, edge := range graph.IndustryChainGraphEdges {
		if !coreid.Is(edge.IndustryChainGraphEdgeID, coreid.IndustryChainGraphEdge) || strings.TrimSpace(edge.Mechanism) == "" ||
			!oneOfResearchGraph(edge.SegmentKind, "direct_candidate", "compressed_candidate") ||
			edge.ReviewStatus != "approved" || edge.Status != "active" || edge.FromChainNodeID == edge.ToChainNodeID {
			return errors.New("persisted Research Graph Industry Chain edge violates invariants")
		}
		if _, ok := chains[edge.IndustryChainID]; !ok {
			return errors.New("persisted Research Graph Industry Chain edge Chain is unavailable")
		}
		if _, ok := entities[edge.FromChainNodeID]; !ok {
			return errors.New("persisted Research Graph Industry Chain edge source is unavailable")
		}
		if _, ok := entities[edge.ToChainNodeID]; !ok {
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
	legacyEntityIDs := make([]string, 0, len(query.EntityIDs))
	countryIDs := make([]string, 0, len(query.EntityIDs))
	for _, id := range query.EntityIDs {
		if biz.IsCountryID(id) {
			countryIDs = append(countryIDs, id)
		} else {
			legacyEntityIDs = append(legacyEntityIDs, id)
		}
	}
	var historicalGap bool
	if err := s.db.QueryRowContext(ctx, `WITH
requested_entities(id) AS (SELECT unnest($2::text[])),
requested_relations(id) AS (SELECT unnest($3::text[])),
all_entities(id, created_at, updated_at) AS (
    SELECT id, created_at, updated_at FROM entity_nodes
    UNION ALL SELECT id, created_at, updated_at FROM industry
    UNION ALL SELECT id, created_at, updated_at FROM concept
	UNION ALL SELECT id, created_at, updated_at FROM chain_node
	UNION ALL SELECT id, created_at, updated_at FROM industry_chain
),
all_entity_relations(id, created_at, updated_at) AS (
    SELECT id, created_at, updated_at FROM entity_edges
    UNION ALL SELECT id, created_at, created_at FROM industry_chain_industry_links
    UNION ALL SELECT id, created_at, created_at FROM industry_chain_concept_links
)
SELECT EXISTS (
    SELECT 1 FROM all_entities entity JOIN requested_entities requested ON requested.id = entity.id
    WHERE entity.created_at <= $1 AND entity.updated_at > $1
) OR EXISTS (
    SELECT 1 FROM all_entity_relations relation JOIN requested_relations requested ON requested.id = relation.id
    WHERE relation.created_at <= $1 AND relation.updated_at > $1
) OR EXISTS (
    SELECT 1 FROM industry_chain definition JOIN requested_entities requested ON requested.id = definition.id
    WHERE definition.created_at <= $1 AND definition.updated_at > $1
)`, query.AnalysisAsOf, legacyEntityIDs, query.EntityRelationIDs).Scan(&historicalGap); err != nil {
		return biz.ResearchReferenceDictionaries{}, fmt.Errorf("check historical Entity reference closure: %w", err)
	}
	if historicalGap {
		return biz.ResearchReferenceDictionaries{}, biz.ErrResearchHistoricalReferencesUnavailable
	}
	var payload []byte
	err := s.db.QueryRowContext(ctx, `WITH
requested_entities(id) AS (SELECT unnest($2::text[])),
requested_relations(id) AS (SELECT unnest($3::text[])),
requested_relation_types(relation_type) AS (SELECT unnest($4::text[])),
all_entities(id, entity_type, name, canonical_name, aliases, status, created_at, updated_at) AS (
    SELECT id, entity_type::text, name, canonical_name, aliases, status::text, created_at, updated_at
    FROM entity_nodes
    UNION ALL SELECT id, 'industry', name, name, aliases, 'active', created_at, updated_at FROM industry
    UNION ALL SELECT id, 'concept', name, name, aliases, 'active', created_at, updated_at FROM concept
	UNION ALL SELECT id, 'chain_node', name, name, aliases, 'active', created_at, updated_at FROM chain_node
	UNION ALL SELECT id, 'industry_chain', name, name, aliases, 'active', created_at, updated_at FROM industry_chain
),
all_entity_relations(id, from_entity_id, to_entity_id, relation_type, status, created_at, updated_at) AS (
    SELECT id, from_entity_id, to_entity_id, relation_type, status::text, created_at, updated_at
    FROM entity_edges
    UNION ALL
    SELECT id, industry_chain_id, industry_id, 'mapped_to_industry', 'active', created_at, created_at
    FROM industry_chain_industry_links
    UNION ALL
    SELECT id, industry_chain_id, concept_id, 'mapped_to_concept', 'active', created_at, created_at
    FROM industry_chain_concept_links
),
selected_relations AS MATERIALIZED (
    SELECT relation.* FROM all_entity_relations relation
    JOIN requested_relations requested ON requested.id = relation.id
    WHERE relation.status = 'active' AND relation.created_at <= $1 AND relation.updated_at <= $1
),
selected_entity_ids(id) AS MATERIALIZED (
    SELECT id FROM requested_entities
    UNION SELECT from_entity_id FROM selected_relations
    UNION SELECT to_entity_id FROM selected_relations
),
selected_entities AS MATERIALIZED (
    SELECT entity.* FROM all_entities entity
    JOIN selected_entity_ids selected ON selected.id = entity.id
    WHERE entity.status = 'active' AND entity.created_at <= $1 AND entity.updated_at <= $1
),
selected_industry_chains AS MATERIALIZED (
    SELECT definition.* FROM industry_chain definition
    JOIN selected_entity_ids selected ON selected.id = definition.id
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
        'industry_chain_id', definition.id, 'scope', definition.scope,
		'target_output', definition.target_output, 'end_use', definition.end_use,
		'geography', definition.geography, 'primary_country_id', definition.primary_country_id,
		'as_of_date', definition.as_of_date,
        'review_status', definition.review_status
    ) ORDER BY definition.id) FROM selected_industry_chains definition), '[]'::jsonb),
    'industry_chain_memberships', '[]'::jsonb,
    'industry_chain_graph_edges', '[]'::jsonb
)`, query.AnalysisAsOf, legacyEntityIDs, query.EntityRelationIDs, query.RelationTypes).Scan(&payload)
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
	}
	countries, err := s.researchCountryReferences(ctx, query.AnalysisAsOf, countryIDs)
	if err != nil {
		return biz.ResearchReferenceDictionaries{}, err
	}
	dictionaries.Entities = append(dictionaries.Entities, countries...)
	sort.Slice(dictionaries.Entities, func(i, j int) bool {
		if dictionaries.Entities[i].EntityType != dictionaries.Entities[j].EntityType {
			return dictionaries.Entities[i].EntityType < dictionaries.Entities[j].EntityType
		}
		if dictionaries.Entities[i].CanonicalName != dictionaries.Entities[j].CanonicalName {
			return dictionaries.Entities[i].CanonicalName < dictionaries.Entities[j].CanonicalName
		}
		return dictionaries.Entities[i].EntityID < dictionaries.Entities[j].EntityID
	})
	if err := validateResearchReferenceDictionaries(dictionaries); err != nil {
		return biz.ResearchReferenceDictionaries{}, err
	}
	return dictionaries, nil
}

func (s *Store) researchCountryReferences(
	ctx context.Context,
	analysisAsOf time.Time,
	ids []string,
) ([]biz.ResearchGraphEntity, error) {
	if len(ids) == 0 {
		return []biz.ResearchGraphEntity{}, nil
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, name_en, created_at, updated_at
		FROM countries
		WHERE id = ANY($1::text[])
		ORDER BY code, id`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]biz.ResearchGraphEntity, 0, len(ids))
	for rows.Next() {
		var id, name, nameEn string
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &name, &nameEn, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		if createdAt.After(analysisAsOf) {
			continue
		}
		if updatedAt.After(analysisAsOf) {
			return nil, biz.ErrResearchHistoricalReferencesUnavailable
		}
		result = append(result, biz.ResearchGraphEntity{
			EntityID: id, EntityType: biz.ObjectTypeCountry, Name: name, CanonicalName: name,
			Aliases: []string{nameEn}, Status: "active",
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(result) != len(ids) {
		return nil, errors.New("one or more Country references are unavailable")
	}
	return result, nil
}

type persistedResearchReferenceDictionaries struct {
	Entities                 []biz.ResearchGraphEntity         `json:"entities"`
	RelationDefinitions      []biz.ResearchGraphRelation       `json:"relation_definitions"`
	EntityRelations          []biz.ResearchGraphEntityRelation `json:"entity_relations"`
	IndustryChains           []biz.ResearchGraphIndustryChain  `json:"industry_chains"`
	IndustryChainMemberships []biz.ResearchGraphMembership     `json:"industry_chain_memberships"`
	IndustryChainGraphEdges  []biz.ResearchGraphIndustryEdge   `json:"industry_chain_graph_edges"`
}

func validateResearchReferenceDictionaries(value biz.ResearchReferenceDictionaries) error {
	entities := make(map[string]struct{}, len(value.Entities))
	for _, entity := range value.Entities {
		if !validResearchGraphIdentity(entity.EntityID, entity.EntityType) || strings.TrimSpace(entity.Name) == "" ||
			strings.TrimSpace(entity.CanonicalName) == "" || entity.Status != "active" {
			return errors.New("persisted Research Entity violates invariants")
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
		if !biz.IsEntityRelationID(relation.EntityRelationID) || relation.Status != "active" {
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
		if _, ok := entities[chain.IndustryChainID]; !ok || chain.ReviewStatus != "approved" ||
			strings.TrimSpace(chain.Scope) == "" || strings.TrimSpace(chain.TargetOutput) == "" ||
			strings.TrimSpace(chain.AsOfDate) == "" ||
			(chain.PrimaryCountryID != "" && !biz.IsCountryID(chain.PrimaryCountryID)) {
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
		WITH all_entities(id, status, created_at, updated_at) AS MATERIALIZED (
		    SELECT id, status::text, created_at, updated_at FROM entity_nodes
		    UNION ALL SELECT id, 'active', created_at, updated_at FROM industry
		    UNION ALL SELECT id, 'active', created_at, updated_at FROM concept
		    UNION ALL SELECT id, 'active', created_at, updated_at FROM chain_node
		    UNION ALL SELECT id, 'active', created_at, updated_at FROM industry_chain
		),
		all_entity_relations(id, from_entity_id, to_entity_id, relation_type, status, created_at, updated_at) AS MATERIALIZED (
		    SELECT id, from_entity_id, to_entity_id, relation_type, status::text, created_at, updated_at
		    FROM entity_edges
		    UNION ALL
		    SELECT id, industry_chain_id, industry_id, 'mapped_to_industry', 'active', created_at, created_at
		    FROM industry_chain_industry_links
		    UNION ALL
		    SELECT id, industry_chain_id, concept_id, 'mapped_to_concept', 'active', created_at, created_at
		    FROM industry_chain_concept_links
		)
		SELECT
		    (
		        SELECT count(*)
		        FROM all_entities entity
		        WHERE entity.id = ANY($2::text[])
		          AND entity.status = $5
		          AND entity.created_at <= $1
		          AND entity.updated_at <= $1
		    ),
		    (
		        SELECT count(DISTINCT requested.relation_type)
		        FROM unnest($3::text[]) requested(relation_type)
		        WHERE EXISTS (
		            SELECT 1
		            FROM all_entity_relations relation
		            JOIN all_entities from_entity
		              ON from_entity.id = relation.from_entity_id
		             AND from_entity.status = $5
		             AND from_entity.created_at <= $1
		             AND from_entity.updated_at <= $1
		            JOIN all_entities to_entity
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
		            JOIN industry_chain definition
		              ON definition.id = edge.industry_chain_id
		             AND definition.review_status = $7
		             AND definition.created_at <= $1
		             AND definition.updated_at <= $1
		            JOIN industry_chain_node_memberships from_membership
		              ON from_membership.industry_chain_id = edge.industry_chain_id
		             AND from_membership.chain_node_id = edge.from_chain_node_id
		             AND from_membership.review_status = $8
		             AND from_membership.status = $9
		             AND from_membership.created_at <= $1
		             AND from_membership.updated_at <= $1
		            JOIN industry_chain_node_memberships to_membership
		              ON to_membership.industry_chain_id = edge.industry_chain_id
		             AND to_membership.chain_node_id = edge.to_chain_node_id
		             AND to_membership.review_status = $8
		             AND to_membership.status = $9
		             AND to_membership.created_at <= $1
		             AND to_membership.updated_at <= $1
		            JOIN all_entities from_entity
		              ON from_entity.id = edge.from_chain_node_id
		             AND from_entity.status = $5
		             AND from_entity.created_at <= $1
		             AND from_entity.updated_at <= $1
		            JOIN all_entities to_entity
		              ON to_entity.id = edge.to_chain_node_id
		             AND to_entity.status = $5
		             AND to_entity.created_at <= $1
		             AND to_entity.updated_at <= $1
		            WHERE edge.relation_type = requested.relation_type
		              AND edge.status = $11
		              AND edge.review_status = $10
		              AND edge.created_at <= $1
		              AND edge.updated_at <= $1
		              AND ($4::text IS NULL OR edge.industry_chain_id = $4::text)
		        )
		    ),
		    CASE
		        WHEN $4::text IS NULL THEN 1
		        ELSE (
		            SELECT count(*)
		            FROM industry_chain definition
		            WHERE definition.id = $4::text
		              AND definition.review_status = $7
		              AND definition.created_at <= $1
		              AND definition.updated_at <= $1
		        )
		    END
	`,
		query.AnalysisAsOf,
		query.SeedEntityIDs,
		relationTypes,
		query.IndustryChainID,
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

type scanner interface{ Scan(...any) error }
