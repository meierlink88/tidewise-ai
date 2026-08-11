package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchgraph"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/postgres"
	eventsemanticfixture "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/testsupport/eventsemantic"
)

func TestPostgresResearchGraphSearchTraversesDeterministicallyAndFailsClosedOnBudget(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	eventsemanticfixture.SeedScenario(t, db, true)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	const downstreamID = "10000000-0000-4000-8000-000000000007"
	const inactiveID = "10000000-0000-4000-8000-000000000009"
	if _, err := db.ExecContext(ctx, `
		INSERT INTO entity_nodes (
		    id, entity_key, entity_type, layer_code, name, canonical_name, aliases, status
		) VALUES (
		    $1, 'company:integration-downstream', 'company', 'company',
		    'Integration Downstream', 'Integration Downstream', '{}', 'active'
		), (
		    $2, 'product:integration-inactive', 'product', 'product',
		    'Integration Inactive', 'Integration Inactive', '{}', 'inactive'
		)
	`, downstreamID, inactiveID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO entity_edges (
		    id, from_entity_id, to_entity_id, relation_type, evidence_note, status
		) VALUES (
		    '10000000-0000-4000-8000-000000000008',
		    $2,
		    $1,
		    'used_as_input_by',
		    'Integration graph fixture',
		    'active'
		)
	`, downstreamID, eventsemanticfixture.ProductID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO entity_edges (
		    id, from_entity_id, to_entity_id, relation_type, evidence_note, status
		) VALUES (
		    '10000000-0000-4000-8000-000000000010',
		    $1,
		    $2,
		    'produces',
		    'Inactive endpoints must never enter the reachable graph',
		    'active'
		)
	`, eventsemanticfixture.CompanyID, inactiveID); err != nil {
		t.Fatal(err)
	}
	if _, err := postgres.NewResearchGraphStore(db).Search(ctx, researchgraph.Query{
		AnalysisAsOf:  time.Date(2030, 7, 30, 0, 0, 0, 0, time.UTC),
		SeedEntityIDs: []string{eventsemanticfixture.CompanyID},
		RelationFilters: []researchgraph.RelationFilter{
			{RelationType: "produces", Direction: researchgraph.DirectionOutgoing},
			{RelationType: "used_as_input_by", Direction: researchgraph.DirectionOutgoing},
		},
		MaxDepth: 2, NodeBudget: 3, EdgeBudget: 2,
		FactPolicy: researchgraph.ApprovedActiveFactPolicy(),
	}); err != nil {
		t.Fatalf("direct PostgreSQL graph query: %v", err)
	}

	handler := dataServiceTestHandler(Dependencies{
		ResearchGraph: researchgraph.NewService(postgres.NewResearchGraphStore(db)),
	}, map[string]v1.Principal{
		"read-token": {
			Identity: "codex-analyst",
			Scopes:   []string{ScopeResearchRead},
		},
	}, "request-research-graph")
	requestBody := []byte(`{
		"analysis_as_of":"2030-07-30T00:00:00Z",
		"seed_entity_ids":["10000000-0000-4000-8000-000000000004"],
		"relation_filters":[
			{"relation_type":"produces","direction":"outgoing"},
			{"relation_type":"used_as_input_by","direction":"outgoing"}
		],
		"max_depth":2,
		"node_budget":3,
		"edge_budget":2
	}`)
	request := httptest.NewRequest(
		http.MethodPost,
		Namespace+"/research-graph:search",
		bytes.NewReader(requestBody),
	)
	request.Header.Set("Authorization", "Bearer read-token")
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body)
	}
	var response struct {
		Result v1.ResearchGraphSearchResult `json:"result"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Result.ActualDepth != 2 ||
		len(response.Result.Entities) != 3 ||
		len(response.Result.EntityRelations) != 2 ||
		len(response.Result.RelationDefinitions) != 2 {
		t.Fatalf("result = %#v", response.Result)
	}
	for _, entity := range response.Result.Entities {
		if entity.EntityID == inactiveID {
			t.Fatalf("inactive endpoint entered the graph: %#v", response.Result)
		}
	}

	store := postgres.NewResearchGraphStore(db)
	incoming, err := store.Search(ctx, researchgraph.Query{
		AnalysisAsOf:  time.Date(2030, 7, 30, 0, 0, 0, 0, time.UTC),
		SeedEntityIDs: []string{downstreamID},
		RelationFilters: []researchgraph.RelationFilter{{
			RelationType: "used_as_input_by",
			Direction:    researchgraph.DirectionIncoming,
		}},
		MaxDepth: 1, NodeBudget: 10, EdgeBudget: 10,
		FactPolicy: researchgraph.ApprovedActiveFactPolicy(),
	})
	if err != nil || incoming.ActualDepth != 1 ||
		len(incoming.Entities) != 2 ||
		len(incoming.EntityRelations) != 1 {
		t.Fatalf("incoming graph = %#v err=%v", incoming, err)
	}
	bidirectional, err := store.Search(ctx, researchgraph.Query{
		AnalysisAsOf:  time.Date(2030, 7, 30, 0, 0, 0, 0, time.UTC),
		SeedEntityIDs: []string{eventsemanticfixture.CompanyID, downstreamID},
		RelationFilters: []researchgraph.RelationFilter{
			{RelationType: "produces", Direction: researchgraph.DirectionBoth},
			{RelationType: "used_as_input_by", Direction: researchgraph.DirectionBoth},
		},
		MaxDepth: 1, NodeBudget: 10, EdgeBudget: 10,
		FactPolicy: researchgraph.ApprovedActiveFactPolicy(),
	})
	if err != nil || bidirectional.ActualDepth != 1 ||
		len(bidirectional.Entities) != 3 ||
		len(bidirectional.EntityRelations) != 2 {
		t.Fatalf("bidirectional multi-seed graph = %#v err=%v", bidirectional, err)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO entity_edges (
		    id, from_entity_id, to_entity_id, relation_type, evidence_note, status
		) VALUES (
		    '10000000-0000-4000-8000-000000000011',
		    $1,
		    $2,
		    'used_as_input_by',
		    'Alternate direct path for minimum-depth verification',
		    'active'
		)
	`, eventsemanticfixture.CompanyID, downstreamID); err != nil {
		t.Fatal(err)
	}
	alternatePath, err := store.Search(ctx, researchgraph.Query{
		AnalysisAsOf:  time.Date(2030, 7, 30, 0, 0, 0, 0, time.UTC),
		SeedEntityIDs: []string{eventsemanticfixture.CompanyID},
		RelationFilters: []researchgraph.RelationFilter{
			{RelationType: "produces", Direction: researchgraph.DirectionOutgoing},
			{RelationType: "used_as_input_by", Direction: researchgraph.DirectionOutgoing},
		},
		MaxDepth: 5, NodeBudget: 10, EdgeBudget: 10,
		FactPolicy: researchgraph.ApprovedActiveFactPolicy(),
	})
	if err != nil || alternatePath.ActualDepth != 1 ||
		len(alternatePath.Entities) != 3 ||
		len(alternatePath.EntityRelations) != 3 {
		t.Fatalf("alternate-path graph = %#v err=%v", alternatePath, err)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO entity_nodes (
		    id, entity_key, entity_type, layer_code, name, canonical_name,
		    aliases, status
		)
		SELECT
		    md5('research-graph-fanout-node-' || series)::uuid,
		    'product:research-graph-fanout-' || series,
		    'product',
		    'product',
		    'Research Graph Fanout ' || series,
		    'research graph fanout ' || series,
		    '{}',
		    'active'
		FROM generate_series(1, 100) series;
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO entity_edges (
		    id, from_entity_id, to_entity_id, relation_type, evidence_note, status
		)
		SELECT
		    md5('research-graph-fanout-edge-' || series)::uuid,
		    $1,
		    md5('research-graph-fanout-node-' || series)::uuid,
		    'produces',
		    'Bounded fanout fixture',
		    'active'
		FROM generate_series(1, 100) series
	`, eventsemanticfixture.CompanyID); err != nil {
		t.Fatal(err)
	}
	_, err = store.Search(ctx, researchgraph.Query{
		AnalysisAsOf:  time.Date(2030, 7, 30, 0, 0, 0, 0, time.UTC),
		SeedEntityIDs: []string{eventsemanticfixture.CompanyID},
		RelationFilters: []researchgraph.RelationFilter{{
			RelationType: "produces",
			Direction:    researchgraph.DirectionOutgoing,
		}},
		MaxDepth: 5, NodeBudget: 10, EdgeBudget: 100,
		FactPolicy: researchgraph.ApprovedActiveFactPolicy(),
	})
	var fanoutLimit *researchgraph.ResourceLimitError
	if !errors.As(err, &fanoutLimit) ||
		fanoutLimit.Component != "research_graph_nodes" ||
		fanoutLimit.ActualRows != nil ||
		fanoutLimit.MaxRows == nil ||
		*fanoutLimit.MaxRows != 10 {
		t.Fatalf("fanout limit error = %#v", err)
	}

	limitedBody := bytes.Replace(
		requestBody,
		[]byte(`"node_budget":3`),
		[]byte(`"node_budget":2`),
		1,
	)
	limitedRequest := httptest.NewRequest(
		http.MethodPost,
		Namespace+"/research-graph:search",
		bytes.NewReader(limitedBody),
	)
	limitedRequest.Header.Set("Authorization", "Bearer read-token")
	limitedRequest.Header.Set("Content-Type", "application/json")
	limitedRecorder := httptest.NewRecorder()

	handler.ServeHTTP(limitedRecorder, limitedRequest)

	if limitedRecorder.Code != http.StatusTooManyRequests {
		t.Fatalf("limited status=%d body=%s", limitedRecorder.Code, limitedRecorder.Body)
	}
	var limitedResponse struct {
		Error struct {
			Code    string                          `json:"code"`
			Details v1.ResearchResourceLimitDetails `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(limitedRecorder.Body.Bytes(), &limitedResponse); err != nil {
		t.Fatal(err)
	}
	if limitedResponse.Error.Code != "RESEARCH_GRAPH_RESOURCE_LIMIT" ||
		limitedResponse.Error.Details.Component != "research_graph_nodes" ||
		limitedResponse.Error.Details.ActualRows != nil ||
		limitedResponse.Error.Details.MaxRows == nil ||
		*limitedResponse.Error.Details.MaxRows != 2 {
		t.Fatalf("limited response = %#v", limitedResponse)
	}
}

func TestPostgresResearchGraphSearchHonorsChainScopeAndBoundsCycles(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	const (
		chainA = "20000000-0000-4000-8000-000000000001"
		chainB = "20000000-0000-4000-8000-000000000002"
		nodeA  = "20000000-0000-4000-8000-000000000003"
		nodeB  = "20000000-0000-4000-8000-000000000004"
		nodeC  = "20000000-0000-4000-8000-000000000005"
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
		    ('20000000-0000-4000-8000-000000000006', '` + chainA + `', '` + nodeA + `', '` + nodeB + `',
		     'input_to', 'A feeds B', 'direct_candidate', 'approved', 'active',
		     ARRAY['evidence:e'], 'integration', 'artifact://graph-a', now()),
		    ('20000000-0000-4000-8000-000000000008', '` + chainB + `', '` + nodeA + `', '` + nodeC + `',
		     'input_to', 'A feeds C in another chain', 'direct_candidate', 'approved', 'active',
		     ARRAY['evidence:g'], 'integration', 'artifact://graph-b', now())`,
		`INSERT INTO entity_edges (
		    id, from_entity_id, to_entity_id, relation_type, evidence_note, status
		) VALUES (
		    '20000000-0000-4000-8000-000000000007', '` + nodeB + `', '` + nodeA + `',
		    'depends_on', 'Cycle fixture outside the chain topology table', 'active'
		)`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("seed scoped graph: %v\n%s", err, statement)
		}
	}

	graph, err := postgres.NewResearchGraphStore(db).Search(ctx, researchgraph.Query{
		AnalysisAsOf:  time.Now().UTC().Add(time.Minute),
		SeedEntityIDs: []string{nodeA},
		RelationFilters: []researchgraph.RelationFilter{
			{RelationType: "input_to", Direction: researchgraph.DirectionOutgoing},
			{RelationType: "depends_on", Direction: researchgraph.DirectionOutgoing},
		},
		MaxDepth:              5,
		IndustryChainEntityID: graphStringPointer(chainA),
		NodeBudget:            10,
		EdgeBudget:            10,
		FactPolicy:            researchgraph.ApprovedActiveFactPolicy(),
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
