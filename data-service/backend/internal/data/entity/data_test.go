package entity

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	domain "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

func TestChainNodeIndustryChainOpenSPGSchemas(t *testing.T) {
	for fileName, properties := range map[string][]string{
		"chain-node.schema": {
			"id(产业链节点标识): Text", "name(产业链节点名称): Text", "aliases(产业链节点别名): Text",
			"definition(产业链节点定义): Text", "reviewStatus(审核状态): Text", "createdAt(创建时间): Text", "updatedAt(更新时间): Text",
		},
		"industry-chain.schema": {
			"id(产业链标识): Text", "name(产业链名称): Text", "aliases(产业链别名): Text", "scope(产业链范围): Text",
			"targetOutput(目标产出): Text", "endUse(终端用途): Text", "geography(地域范围): Text", "asOfDate(截至日期): Text",
			"reviewStatus(审核状态): Text", "reviewNote(审核说明): Text", "technologyRouteQualifier(技术路线限定): Text",
			"observableVariables(可观察变量): Text", "createdAt(创建时间): Text", "updatedAt(更新时间): Text",
			"primaryCountry(主要国家): Country", "chainNodes(产业链节点): ChainNode",
		},
	} {
		path, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "doctype", fileName))
		if err != nil {
			t.Fatal(err)
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, property := range properties {
			if !strings.Contains(string(contents), property) {
				t.Errorf("%s is missing %q", fileName, property)
			}
		}
	}
}

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
			EntityID: entityID, EntityType: "security", Name: "Security", CanonicalName: "Security", Status: "active",
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
	if _, err := db.ExecContext(ctx, `INSERT INTO organization_categories(id,code,name_zh) VALUES('OCA7cf04802-4d04-5a8c-9a10-7d805cf29a4d','INTERGOVERNMENTAL','政府间国际组织'); INSERT INTO organization_functions(id,code,name_zh) VALUES('OFN72d5d191-1510-5f5b-a2ab-0cc3a8919107','GOVERNANCE','治理与协调'); INSERT INTO organizations(id,code,name,name_en,category_code,function_code) VALUES('ORG3fb9e7ff-2222-57fa-b306-c223ce3af549','UN','联合国','United Nations','INTERGOVERNMENTAL','GOVERNANCE'); INSERT INTO countries(id,code,name,name_en) VALUES('COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b','CN','中国','China'); INSERT INTO organization_members(id,organization_id,country_id,membership_type,effective_date) VALUES('OMB11111111-1111-4111-8111-111111111111','ORG3fb9e7ff-2222-57fa-b306-c223ce3af549','COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b','FULL_MEMBER','1945-10-24')`); err != nil {
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

func TestResearchGraphResolvesIndustryChainTypedLinksWithoutShadowEntities(t *testing.T) {
	db := openEntityTestDatabase(t)
	ctx := context.Background()
	const (
		industryID     = "IND11111111-1111-4111-8111-111111111111"
		conceptID      = "CON22222222-2222-4222-8222-222222222222"
		chainID        = "ICH33333333-3333-4333-8333-333333333333"
		industryLinkID = "ERL44444444-4444-4444-8444-444444444444"
		conceptLinkID  = "ERL55555555-5555-4555-8555-555555555555"
	)
	if _, err := db.ExecContext(ctx, `
		INSERT INTO industry (
			id, name, aliases, classification_system, industry_code,
			parent_industry_id, hierarchy_path_codes, definition, review_status
		) VALUES (
			'`+industryID+`', '半导体', ARRAY['集成电路'], 'sw', '801000',
			NULL, ARRAY['801000'], '半导体行业', 'approved'
		);
		INSERT INTO concept (
			id, name, aliases, concept_type, definition, review_status
		) VALUES (
			'`+conceptID+`', '先进制程', ARRAY['先进节点'], 'technology', '先进制程概念', 'approved'
		);
		INSERT INTO industry_chain (
			id, name, aliases, scope, target_output, end_use, observable_variables,
			geography, as_of_date, review_status
		) VALUES (
			'`+chainID+`', '半导体产业链', '{}', '半导体产业链', '芯片', '计算', ARRAY['supply'],
			'global', CURRENT_DATE, 'approved'
		);
		INSERT INTO industry_chain_industry_links (
			id, industry_chain_id, industry_id
		) VALUES (
			'`+industryLinkID+`', '`+chainID+`', '`+industryID+`'
		);
		INSERT INTO industry_chain_concept_links (
			id, industry_chain_id, concept_id
		) VALUES (
			'`+conceptLinkID+`', '`+chainID+`', '`+conceptID+`'
		)`); err != nil {
		t.Fatal(err)
	}

	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, seedID, relationType, relationID, targetType, targetName string
	}{
		{name: "industry", seedID: industryID, relationType: "mapped_to_industry", relationID: industryLinkID, targetType: "industry", targetName: "半导体"},
		{name: "concept", seedID: conceptID, relationType: "mapped_to_concept", relationID: conceptLinkID, targetType: "concept", targetName: "先进制程"},
	} {
		t.Run(test.name, func(t *testing.T) {
			graph, err := store.SearchResearchGraph(ctx, domain.ResearchGraphQuery{
				AnalysisAsOf: time.Now().UTC().Add(time.Minute), SeedEntityIDs: []string{test.seedID},
				RelationFilters: []domain.ResearchGraphRelationFilter{{
					RelationType: test.relationType, Direction: domain.ResearchGraphDirectionIncoming,
				}},
				MaxDepth: 1, NodeBudget: 10, EdgeBudget: 10,
				FactPolicy: domain.ApprovedActiveResearchGraphFactPolicy(),
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(graph.Entities) != 2 || len(graph.EntityRelations) != 1 {
				t.Fatalf("%s Research Graph = %#v", test.name, graph)
			}
			relation := graph.EntityRelations[0]
			if relation.EntityRelationID != test.relationID || relation.FromEntityID != chainID ||
				relation.ToEntityID != test.seedID || relation.RelationType != test.relationType || relation.Status != "active" {
				t.Fatalf("%s Research Graph relation = %#v", test.name, relation)
			}
			foundTarget := false
			for _, entity := range graph.Entities {
				if entity.EntityID == test.seedID {
					foundTarget = entity.EntityType == test.targetType && entity.Name == test.targetName
				}
			}
			if !foundTarget {
				t.Fatalf("independent %s missing from graph: %#v", test.targetType, graph.Entities)
			}
		})
	}
	closure, err := store.ResearchReferenceClosure(ctx, domain.ResearchReferenceQuery{
		AnalysisAsOf:      time.Now().UTC().Add(time.Minute),
		EntityRelationIDs: []string{industryLinkID, conceptLinkID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(closure.EntityRelations) != 2 || len(closure.Entities) != 3 {
		t.Fatalf("typed Link reference closure = %#v", closure)
	}
	for name, statement := range map[string]string{
		"duplicate Industry endpoint pair": `INSERT INTO industry_chain_industry_links (id, industry_chain_id, industry_id)
			VALUES ('ERL66666666-6666-4666-8666-666666666666', '` + chainID + `', '` + industryID + `')`,
		"reserved generic mapping type": `INSERT INTO entity_edges (id, from_entity_id, to_entity_id, relation_type, evidence_note, status)
			VALUES ('ERL77777777-7777-4777-8777-777777777777', '` + chainID + `', '` + industryID + `', 'mapped_to_industry', 'legacy write', 'active')`,
		"cross-store ERL identity": `INSERT INTO entity_edges (id, from_entity_id, to_entity_id, relation_type, evidence_note, status)
			VALUES ('` + industryLinkID + `', '` + chainID + `', '` + conceptID + `', 'related_to', 'identity collision', 'active')`,
		"unknown typed Industry": `INSERT INTO industry_chain_industry_links (id, industry_chain_id, industry_id)
			VALUES ('ERL88888888-8888-4888-8888-888888888888', '` + chainID + `', 'IND99999999-9999-4999-8999-999999999999')`,
	} {
		if _, err := db.ExecContext(ctx, statement); err == nil {
			t.Fatalf("%s write succeeded", name)
		}
	}

	foundIndustry := false
	for _, entity := range closure.Entities {
		if entity.EntityID == industryID {
			foundIndustry = entity.EntityType == "industry" && entity.Name == "半导体" &&
				len(entity.Aliases) == 1 && entity.Aliases[0] == "集成电路"
		}
	}
	if !foundIndustry {
		t.Fatalf("independent Industry missing from closure: %#v", closure.Entities)
	}
	var shadowRows int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM entity_nodes WHERE id = $1`, industryID).Scan(&shadowRows); err != nil {
		t.Fatal(err)
	}
	if shadowRows != 0 {
		t.Fatalf("shadow Entity rows = %d", shadowRows)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM industry WHERE id = $1`, industryID); err == nil ||
		!strings.Contains(err.Error(), "is still referenced and cannot change identity or be deleted") {
		t.Fatalf("delete referenced Industry error = %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM industry_chain WHERE id = $1`, chainID); err == nil ||
		!strings.Contains(err.Error(), "is still referenced and cannot change identity or be deleted") {
		t.Fatalf("delete referenced IndustryChain error = %v", err)
	}
	if _, err := db.ExecContext(ctx, `TRUNCATE industry, company_industry_links, industry_chain_industry_links`); err == nil ||
		!strings.Contains(err.Error(), "still owns referenced facts and cannot be truncated") {
		t.Fatalf("truncate referenced Industry table error = %v", err)
	}
	var industryRows int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM industry WHERE id = $1`, industryID).Scan(&industryRows); err != nil {
		t.Fatal(err)
	}
	if industryRows != 1 {
		t.Fatalf("referenced Industry rows after rejected delete = %d", industryRows)
	}
}

func TestIndependentEntityPersistenceSchemasStayAligned(t *testing.T) {
	db := openEntityTestDatabase(t)
	wantColumns := map[string]map[string]string{
		"industry": {
			"id": "NO", "name": "NO", "aliases": "NO", "classification_system": "NO",
			"industry_code": "NO", "parent_industry_id": "YES", "hierarchy_path_codes": "NO",
			"definition": "NO", "review_status": "NO", "created_at": "NO", "updated_at": "NO",
		},
		"concept": {
			"id": "NO", "name": "NO", "aliases": "NO", "concept_type": "NO",
			"definition": "NO", "review_status": "NO", "created_at": "NO", "updated_at": "NO",
		},
		"chain_node": {
			"id": "NO", "name": "NO", "aliases": "NO", "definition": "NO",
			"review_status": "NO", "created_at": "NO", "updated_at": "NO",
		},
		"industry_chain": {
			"id": "NO", "name": "NO", "aliases": "NO", "scope": "NO", "target_output": "NO",
			"end_use": "NO", "geography": "NO", "as_of_date": "NO", "review_status": "NO",
			"review_note": "YES", "created_at": "NO", "updated_at": "NO",
			"technology_route_qualifier": "YES", "observable_variables": "NO", "primary_country_id": "YES",
		},
		"industry_chain_industry_links": {
			"id": "NO", "industry_chain_id": "NO", "industry_id": "NO", "created_at": "NO",
		},
		"industry_chain_concept_links": {
			"id": "NO", "industry_chain_id": "NO", "concept_id": "NO", "created_at": "NO",
		},
	}
	for tableName, want := range wantColumns {
		rows, err := db.QueryContext(context.Background(), `
			SELECT column_name, is_nullable
			FROM information_schema.columns
			WHERE table_schema = current_schema() AND table_name = $1`, tableName)
		if err != nil {
			t.Fatal(err)
		}
		got := make(map[string]string)
		for rows.Next() {
			var name, nullable string
			if err := rows.Scan(&name, &nullable); err != nil {
				rows.Close()
				t.Fatal(err)
			}
			got[name] = nullable
		}
		if err := rows.Close(); err != nil {
			t.Fatal(err)
		}
		if len(got) != len(want) {
			t.Fatalf("%s columns = %#v, want %#v", tableName, got, want)
		}
		for name, nullable := range want {
			if got[name] != nullable {
				t.Fatalf("%s.%s nullable = %q, want %q", tableName, name, got[name], nullable)
			}
		}
		for _, retired := range []string{"entity_id", "classification_version", "classification_level", "boundary_note"} {
			if _, ok := got[retired]; ok {
				t.Fatalf("%s retains retired column %s", tableName, retired)
			}
		}
	}

	var legacyTables int
	if err := db.QueryRowContext(context.Background(), `
		SELECT count(*) FROM pg_class
		WHERE oid IN (to_regclass('industry_profiles'), to_regclass('concept_profiles'),
		              to_regclass('chain_node_profiles'), to_regclass('industry_chain_definitions'))`).Scan(&legacyTables); err != nil {
		t.Fatal(err)
	}
	if legacyTables != 0 {
		t.Fatalf("legacy profile tables = %d", legacyTables)
	}

	for name, statement := range map[string]string{
		"blank ChainNode alias": `INSERT INTO chain_node (
			id, name, aliases, definition, review_status
		) VALUES ('CND60000000-0000-4000-8000-000000000001', 'Blank Alias', ARRAY[''], 'definition', 'candidate')`,
		"duplicate ChainNode alias": `INSERT INTO chain_node (
			id, name, aliases, definition, review_status
		) VALUES ('CND60000000-0000-4000-8000-000000000002', 'Duplicate Alias', ARRAY['same','same'], 'definition', 'candidate')`,
		"inverted ChainNode timestamps": `INSERT INTO chain_node (
			id, name, aliases, definition, review_status, created_at, updated_at
		) VALUES ('CND60000000-0000-4000-8000-000000000003', 'Inverted Time', '{}', 'definition', 'candidate', now(), now() - interval '1 day')`,
		"blank IndustryChain variable": `INSERT INTO industry_chain (
			id, name, aliases, scope, target_output, end_use, geography, as_of_date,
			review_status, observable_variables
		) VALUES ('ICH60000000-0000-4000-8000-000000000004', 'Blank Variable', '{}', 'scope', 'output', 'use', 'global', CURRENT_DATE, 'candidate', ARRAY[''])`,
		"duplicate IndustryChain variable": `INSERT INTO industry_chain (
			id, name, aliases, scope, target_output, end_use, geography, as_of_date,
			review_status, observable_variables
		) VALUES ('ICH60000000-0000-4000-8000-000000000005', 'Duplicate Variable', '{}', 'scope', 'output', 'use', 'global', CURRENT_DATE, 'candidate', ARRAY['same','same'])`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := db.ExecContext(context.Background(), statement); err == nil {
				t.Fatal("invalid independent object write succeeded")
			}
		})
	}
}

func TestIndependentObjectPrefixesSeparateOwnersWithTheSameUUIDSuffix(t *testing.T) {
	db := openEntityTestDatabase(t)
	ctx := context.Background()
	const suffix = "44444444-4444-4444-8444-444444444444"
	if _, err := db.ExecContext(ctx, `INSERT INTO industry (
		id, name, aliases, classification_system, industry_code,
		parent_industry_id, hierarchy_path_codes, definition, review_status
	) VALUES ($1, '独立行业', '{}', 'test', 'industry', NULL, ARRAY['industry'], '测试行业', 'approved')`, "IND"+suffix); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO concept (
		id, name, aliases, concept_type, definition, review_status
	) VALUES ($1, '独立概念', '{}', 'technology', '测试概念', 'approved')`, "CON"+suffix); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO entity_nodes (
		id, entity_key, entity_type, layer_code, name, canonical_name, aliases, status
	) VALUES ($1, 'security:same-suffix', 'security', 'security', '独立证券', '独立证券', '{}', 'active')`, "ENT"+suffix); err != nil {
		t.Fatal(err)
	}
	for objectID, wantType := range map[string]string{
		"ENT" + suffix: "security",
		"IND" + suffix: "industry",
		"CON" + suffix: "concept",
	} {
		var gotType string
		if err := db.QueryRowContext(ctx, `SELECT data_object_type($1)`, objectID).Scan(&gotType); err != nil {
			t.Fatal(err)
		}
		if gotType != wantType {
			t.Fatalf("data_object_type(%q) = %q, want %q", objectID, gotType, wantType)
		}
	}
}

func TestResearchGraphSearchHonorsChainScopeAndBoundsCycles(t *testing.T) {
	db := openEntityTestDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	const (
		chainA = "ICH20000000-0000-4000-8000-000000000001"
		chainB = "ICH20000000-0000-4000-8000-000000000002"
		nodeA  = "CND20000000-0000-4000-8000-000000000003"
		nodeB  = "CND20000000-0000-4000-8000-000000000004"
		nodeC  = "CND20000000-0000-4000-8000-000000000005"
	)
	for _, statement := range []string{
		`INSERT INTO chain_node (id, name, aliases, definition, review_status) VALUES
		    ('` + nodeA + `', 'Graph Node A', '{}', 'Graph Node A definition', 'approved'),
		    ('` + nodeB + `', 'Graph Node B', '{}', 'Graph Node B definition', 'approved'),
		    ('` + nodeC + `', 'Graph Node C', '{}', 'Graph Node C definition', 'approved')`,
		`INSERT INTO industry_chain (
		    id, name, aliases, scope, target_output, end_use, observable_variables,
		    geography, as_of_date, review_status
		) VALUES
		    ('` + chainA + `', 'Graph Chain A', '{}', 'Graph Chain A scope', 'A', 'A use', ARRAY['supply'], 'CN', CURRENT_DATE, 'approved'),
		    ('` + chainB + `', 'Graph Chain B', '{}', 'Graph Chain B scope', 'B', 'B use', ARRAY['demand'], 'CN', CURRENT_DATE, 'approved')`,
		`INSERT INTO industry_chain_node_memberships (
		    industry_chain_id, chain_node_id, position, contextual_stage,
		    review_status, status, inclusion_reason, evidence_ids, source_name, source_url, verified_at
		) VALUES
		    ('` + chainA + `', '` + nodeA + `', 1, 'upstream', 'approved', 'active', 'A seed', ARRAY['evidence:a'], 'integration', 'artifact://graph-a', now()),
		    ('` + chainA + `', '` + nodeB + `', 2, 'midstream', 'approved', 'active', 'A target', ARRAY['evidence:b'], 'integration', 'artifact://graph-a', now()),
		    ('` + chainB + `', '` + nodeA + `', 1, 'upstream', 'approved', 'active', 'B seed', ARRAY['evidence:c'], 'integration', 'artifact://graph-b', now()),
		    ('` + chainB + `', '` + nodeC + `', 2, 'downstream', 'approved', 'active', 'B target', ARRAY['evidence:d'], 'integration', 'artifact://graph-b', now())`,
		`INSERT INTO industry_chain_graph_edges (
		    id, industry_chain_id, from_chain_node_id, to_chain_node_id,
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
		MaxDepth:        5,
		IndustryChainID: graphStringPointer(chainA),
		NodeBudget:      10,
		EdgeBudget:      10,
		FactPolicy:      domain.ApprovedActiveResearchGraphFactPolicy(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if graph.ActualDepth != 1 ||
		len(graph.Entities) != 3 ||
		len(graph.IndustryChains) != 1 ||
		graph.IndustryChains[0].IndustryChainID != chainA ||
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
