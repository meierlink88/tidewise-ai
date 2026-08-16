package entity

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	domain "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

func TestResearchGraphIdentityRejectsLegacyUUIDAsOrganization(t *testing.T) {
	legacyUUID := "6f845f9f-10e2-44dd-b08a-e482e32d3558"
	if validResearchGraphIdentity(legacyUUID, domain.ObjectTypeOrganization) {
		t.Fatal("legacy UUID must not be accepted as an independent Organization")
	}
	if !validResearchGraphIdentity("ORG3fb9e7ff-2222-57fa-b306-c223ce3af549", domain.ObjectTypeOrganization) {
		t.Fatal("stable ORG_ identity must be accepted as an independent Organization")
	}
}

func TestResearchGraphAdapterRejectsMalformedPersistedSubgraph(t *testing.T) {
	entityID := "ENT11111111-1111-4111-8111-111111111111"
	valid := domain.ResearchGraphSubgraph{
		Entities: []domain.ResearchGraphEntity{{
			EntityID: entityID, EntityType: "company", Name: "Company", CanonicalName: "Company", Status: "active",
		}},
	}
	if err := validatePersistedResearchGraph(valid, 2); err != nil {
		t.Fatalf("valid persisted Research Graph rejected: %v", err)
	}
	invalid := valid
	invalid.Entities = append([]domain.ResearchGraphEntity(nil), valid.Entities...)
	invalid.Entities[0].Status = "invented"
	if err := validatePersistedResearchGraph(invalid, 2); err == nil {
		t.Fatal("malformed persisted Research Graph Entity status was accepted")
	}
	invalid = valid
	invalid.EntityRelations = []domain.ResearchGraphEntityRelation{{
		EntityRelationID: "ERL22222222-2222-4222-8222-222222222222",
		FromEntityID:     entityID, ToEntityID: "ENT33333333-3333-4333-8333-333333333333",
		RelationType: "supplies", Status: "active",
	}}
	invalid.RelationDefinitions = []domain.ResearchGraphRelation{{RelationType: "supplies", Direction: "directed"}}
	if err := validatePersistedResearchGraph(invalid, 2); err == nil {
		t.Fatal("dangling persisted Research Graph relation was accepted")
	}
}

func openEntityTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, "tw_entity", migrationDir, 0)
}

func TestResearchGraphResolvesIndependentOrganizationMembership(t *testing.T) {
	db := openEntityTestDatabase(t)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `INSERT INTO organization_categories(id,code,name_zh) VALUES('OCA7cf04802-4d04-5a8c-9a10-7d805cf29a4d','INTERGOVERNMENTAL','政府间国际组织'); INSERT INTO organization_functions(code,name_zh) VALUES('GOVERNANCE','治理与协调'); INSERT INTO organizations(id,code,name,name_en,category_code,function_code) VALUES('ORG3fb9e7ff-2222-57fa-b306-c223ce3af549','UN','联合国','United Nations','INTERGOVERNMENTAL','GOVERNANCE'); INSERT INTO countries(id,code,name,name_en) VALUES('COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b','CN','中国','China'); INSERT INTO organization_members(id,organization_id,country_id,membership_type,effective_date) VALUES('OMB11111111-1111-4111-8111-111111111111','ORG3fb9e7ff-2222-57fa-b306-c223ce3af549','COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b','FULL_MEMBER','1945-10-24')`); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := store.SearchResearchGraph(ctx, domain.ResearchGraphQuery{AnalysisAsOf: time.Now().UTC().Add(time.Minute), SeedEntityIDs: []string{"ORG3fb9e7ff-2222-57fa-b306-c223ce3af549"}, RelationFilters: []domain.ResearchGraphRelationFilter{{RelationType: "has_member", Direction: domain.ResearchGraphDirectionOutgoing}}, MaxDepth: 1, NodeBudget: 10, EdgeBudget: 10, FactPolicy: domain.ApprovedActiveResearchGraphFactPolicy()})
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Entities) != 2 || len(graph.EntityRelations) != 1 || graph.Entities[1].EntityID != "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549" && graph.Entities[0].EntityID != "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549" {
		t.Fatalf("Organization Research Graph = %#v", graph)
	}
	if !domain.IsEntityRelationID(graph.EntityRelations[0].EntityRelationID) {
		t.Fatalf("Organization membership relation ID = %q", graph.EntityRelations[0].EntityRelationID)
	}
}
func TestResearchGraphSearchHonorsChainScopeAndBoundsCycles(t *testing.T) {
	db := openEntityTestDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	const (
		chainA = "ENT20000000-0000-4000-8000-000000000001"
		chainB = "ENT20000000-0000-4000-8000-000000000002"
		nodeA  = "ENT20000000-0000-4000-8000-000000000003"
		nodeB  = "ENT20000000-0000-4000-8000-000000000004"
		nodeC  = "ENT20000000-0000-4000-8000-000000000005"
	)
	for _, statement := range []string{
		`INSERT INTO entity_nodes (
		    id, entity_key, entity_type, layer_code, name, canonical_name, aliases, status
		) VALUES
		    ('` + chainA + `', 'industry-chain:graph-a', 'industry_chain', 'industry_chain', 'Graph Chain A', 'Graph Chain A', '{}', 'active'),
		    ('` + chainB + `', 'industry-chain:graph-b', 'industry_chain', 'industry_chain', 'Graph Chain B', 'Graph Chain B', '{}', 'active'),
		    ('` + nodeA + `', 'chain-node:graph-a', 'chain_node', 'chain_node', 'Graph Node A', 'Graph Node A', '{}', 'active'),
		    ('` + nodeB + `', 'chain-node:graph-b', 'chain_node', 'chain_node', 'Graph Node B', 'Graph Node B', '{}', 'active'),
		    ('` + nodeC + `', 'chain-node:graph-c', 'chain_node', 'chain_node', 'Graph Node C', 'Graph Node C', '{}', 'active')`,
		`INSERT INTO chain_node_profiles (entity_id, definition, boundary_note, review_status) VALUES
		    ('` + nodeA + `', 'Graph Node A definition', 'Graph Node A boundary', 'approved'),
		    ('` + nodeB + `', 'Graph Node B definition', 'Graph Node B boundary', 'approved'),
		    ('` + nodeC + `', 'Graph Node C definition', 'Graph Node C boundary', 'approved')`,
		`INSERT INTO industry_chain_definitions (
		    entity_id, scope, target_output, end_use, observable_variables,
		    geography, as_of_date, review_status
		) VALUES
		    ('` + chainA + `', 'Graph Chain A scope', 'A', 'A use', ARRAY['supply'], 'CN', CURRENT_DATE, 'approved'),
		    ('` + chainB + `', 'Graph Chain B scope', 'B', 'B use', ARRAY['demand'], 'CN', CURRENT_DATE, 'approved')`,
		`INSERT INTO industry_chain_node_memberships (
		    industry_chain_entity_id, chain_node_entity_id, position, contextual_stage,
		    review_status, status, inclusion_reason, evidence_ids, source_name, source_url, verified_at
		) VALUES
		    ('` + chainA + `', '` + nodeA + `', 1, 'upstream', 'approved', 'active', 'A seed', ARRAY['evidence:a'], 'integration', 'artifact://graph-a', now()),
		    ('` + chainA + `', '` + nodeB + `', 2, 'midstream', 'approved', 'active', 'A target', ARRAY['evidence:b'], 'integration', 'artifact://graph-a', now()),
		    ('` + chainB + `', '` + nodeA + `', 1, 'upstream', 'approved', 'active', 'B seed', ARRAY['evidence:c'], 'integration', 'artifact://graph-b', now()),
		    ('` + chainB + `', '` + nodeC + `', 2, 'downstream', 'approved', 'active', 'B target', ARRAY['evidence:d'], 'integration', 'artifact://graph-b', now())`,
		`INSERT INTO industry_chain_graph_edges (
		    id, industry_chain_entity_id, from_chain_node_entity_id, to_chain_node_entity_id,
		    relation_type, mechanism, segment_kind, review_status, status,
		    evidence_ids, source_name, source_url, verified_at
		) VALUES
		    ('IGE20000000-0000-4000-8000-000000000006', '` + chainA + `', '` + nodeA + `', '` + nodeB + `',
		     'input_to', 'A feeds B', 'direct_candidate', 'approved', 'active',
		     ARRAY['evidence:e'], 'integration', 'artifact://graph-a', now()),
		    ('IGE20000000-0000-4000-8000-000000000008', '` + chainB + `', '` + nodeA + `', '` + nodeC + `',
		     'input_to', 'A feeds C in another chain', 'direct_candidate', 'approved', 'active',
		     ARRAY['evidence:g'], 'integration', 'artifact://graph-b', now())`,
		`INSERT INTO entity_edges (
		    id, from_entity_id, to_entity_id, relation_type, evidence_note, status
		) VALUES (
		    'ERL20000000-0000-4000-8000-000000000007', '` + nodeB + `', '` + nodeA + `',
		    'depends_on', 'Cycle fixture outside the chain topology table', 'active'
		)`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("seed scoped graph: %v\n%s", err, statement)
		}
	}

	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := store.SearchResearchGraph(ctx, domain.ResearchGraphQuery{
		AnalysisAsOf:  time.Now().UTC().Add(time.Minute),
		SeedEntityIDs: []string{nodeA},
		RelationFilters: []domain.ResearchGraphRelationFilter{
			{RelationType: "input_to", Direction: domain.ResearchGraphDirectionOutgoing},
			{RelationType: "depends_on", Direction: domain.ResearchGraphDirectionOutgoing},
		},
		MaxDepth:              5,
		IndustryChainEntityID: graphStringPointer(chainA),
		NodeBudget:            10,
		EdgeBudget:            10,
		FactPolicy:            domain.ApprovedActiveResearchGraphFactPolicy(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if graph.ActualDepth != 1 ||
		len(graph.Entities) != 3 ||
		len(graph.IndustryChains) != 1 ||
		graph.IndustryChains[0].IndustryChainEntityID != chainA ||
		len(graph.IndustryChainMemberships) != 2 ||
		len(graph.IndustryChainGraphEdges) != 1 ||
		len(graph.EntityRelations) != 1 {
		t.Fatalf("scoped cyclic graph = %#v", graph)
	}
	for _, entity := range graph.Entities {
		if entity.EntityID == chainB || entity.EntityID == nodeC {
			t.Fatalf("out-of-scope entity returned: %#v", entity)
		}
	}
}

func graphStringPointer(value string) *string {
	return &value
}
