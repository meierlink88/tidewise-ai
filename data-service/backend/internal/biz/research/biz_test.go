package research

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type graphStoreStub struct {
	query GraphQuery
	graph GraphSubgraph
	err   error
}

func (s *graphStoreStub) SearchResearchGraph(_ context.Context, query GraphQuery) (GraphSubgraph, error) {
	s.query = query
	return s.graph, s.err
}

func validGraphRequest() GraphSearchRequest {
	return GraphSearchRequest{
		AnalysisAsOf:  "2026-07-30T00:00:00Z",
		SeedEntityIDs: []string{"ENT11111111-1111-4111-8111-111111111111"},
		RelationFilters: []RelationFilter{{
			RelationType: "produces",
			Direction:    DirectionOutgoing,
		}},
		MaxDepth:   1,
		NodeBudget: 10,
		EdgeBudget: 10,
	}
}

func TestNewUseCaseRequiresGraphStore(t *testing.T) {
	if _, err := NewUseCase(nil); err == nil {
		t.Fatal("nil Research Graph store was accepted")
	}
	if _, err := NewUseCase(&graphStoreStub{}); err != nil {
		t.Fatal(err)
	}
}

func TestGraphReturnsDeterministicReferenceCompleteGraph(t *testing.T) {
	store := &graphStoreStub{graph: GraphSubgraph{
		ActualDepth: 1,
		Entities: []GraphEntity{
			{EntityID: "ENT11111111-1111-4111-8111-111111111111", EntityType: "company", Name: "Producer", CanonicalName: "producer", Status: "active"},
			{EntityID: "ICH22222222-2222-4222-8222-222222222222", EntityType: "industry_chain", Name: "Product", CanonicalName: "product", Status: "active"},
		},
		RelationDefinitions: []GraphRelationDefinition{{RelationType: "produces", Direction: "directed"}},
		EntityRelations: []GraphEntityRelation{{
			EntityRelationID: "ERL33333333-3333-4333-8333-333333333333",
			FromEntityID:     "ENT11111111-1111-4111-8111-111111111111",
			ToEntityID:       "ICH22222222-2222-4222-8222-222222222222",
			RelationType:     "produces",
			Status:           "active",
		}},
		IndustryChains:           []GraphIndustryChain{},
		IndustryChainMemberships: []GraphIndustryChainMembership{},
		IndustryChainGraphEdges:  []GraphIndustryChainEdge{},
	}}
	useCase, err := NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	result, err := useCase.Search(context.Background(), validGraphRequest())
	if err != nil {
		t.Fatal(err)
	}
	if result.ContractVersion != GraphContractVersion || result.ActualDepth != 1 ||
		!testGraphHashPattern(result.QueryFingerprint) || !testGraphHashPattern(result.GraphFingerprint) {
		t.Fatalf("result = %#v", result)
	}
	if len(result.Entities) != 2 || len(result.EntityRelations) != 1 ||
		store.query.RelationFilters[0].Direction != DirectionOutgoing {
		t.Fatalf("result = %#v query = %#v", result, store.query)
	}
}

func testGraphHashPattern(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func TestGraphAcceptsIndependentCountryAndRegionObjects(t *testing.T) {
	store := &graphStoreStub{graph: GraphSubgraph{
		ActualDepth: 1,
		Entities: []GraphEntity{
			{EntityID: "COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b", EntityType: "country", Name: "中国", CanonicalName: "中国", Status: "active"},
			{EntityID: "REG88d53cc8-1c75-57e6-a02c-56f9a4bc13c4", EntityType: "region", Name: "亚太地区", CanonicalName: "亚太地区", Status: "active"},
		},
		RelationDefinitions: []GraphRelationDefinition{{RelationType: "belongs_to_region", Direction: "directed"}},
		EntityRelations: []GraphEntityRelation{{
			EntityRelationID: "ERL33333333-3333-4333-8333-333333333333",
			FromEntityID:     "COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b",
			ToEntityID:       "REG88d53cc8-1c75-57e6-a02c-56f9a4bc13c4",
			RelationType:     "belongs_to_region",
			Status:           "active",
		}},
		IndustryChains:           []GraphIndustryChain{},
		IndustryChainMemberships: []GraphIndustryChainMembership{},
		IndustryChainGraphEdges:  []GraphIndustryChainEdge{},
	}}
	request := validGraphRequest()
	request.AnalysisAsOf = "2026-08-14T00:00:00Z"
	request.SeedEntityIDs = []string{"COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b"}
	request.RelationFilters = []RelationFilter{{RelationType: "belongs_to_region", Direction: DirectionOutgoing}}
	result, err := (&UseCase{graphStore: store}).Search(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entities) != 2 || store.query.SeedEntityIDs[0] != request.SeedEntityIDs[0] {
		t.Fatalf("Country graph result=%#v query=%#v", result, store.query)
	}
}

func TestGraphRejectsInvalidOrOrphanedGraphRequests(t *testing.T) {
	valid := validGraphRequest()
	for _, mutate := range []func(*GraphSearchRequest){
		func(request *GraphSearchRequest) { request.SeedEntityIDs = nil },
		func(request *GraphSearchRequest) {
			request.SeedEntityIDs = append(request.SeedEntityIDs, request.SeedEntityIDs[0])
		},
		func(request *GraphSearchRequest) { request.RelationFilters[0].Direction = "sideways" },
		func(request *GraphSearchRequest) {
			request.RelationFilters = append(request.RelationFilters, RelationFilter{RelationType: "produces", Direction: DirectionIncoming})
		},
		func(request *GraphSearchRequest) { request.MaxDepth = GraphMaxDepth + 1 },
		func(request *GraphSearchRequest) { request.NodeBudget = 0 },
	} {
		request := valid
		request.SeedEntityIDs = append([]string(nil), valid.SeedEntityIDs...)
		request.RelationFilters = append([]RelationFilter(nil), valid.RelationFilters...)
		mutate(&request)
		if _, err := (&UseCase{graphStore: &graphStoreStub{}}).Search(context.Background(), request); err == nil {
			t.Fatalf("request unexpectedly accepted: %#v", request)
		}
	}

	store := &graphStoreStub{graph: GraphSubgraph{
		Entities:            []GraphEntity{{EntityID: "ENT11111111-1111-4111-8111-111111111111", EntityType: "company"}},
		RelationDefinitions: []GraphRelationDefinition{{RelationType: "produces", Direction: "directed"}},
		EntityRelations: []GraphEntityRelation{{
			EntityRelationID: "ERL33333333-3333-4333-8333-333333333333",
			FromEntityID:     "ENT11111111-1111-4111-8111-111111111111",
			ToEntityID:       "ICH22222222-2222-4222-8222-222222222222",
			RelationType:     "produces",
		}},
	}}
	if _, err := (&UseCase{graphStore: store}).Search(context.Background(), valid); err == nil {
		t.Fatal("orphaned graph edge was accepted")
	}
}

func TestGraphReportsExceededGraphBudgetDimension(t *testing.T) {
	valid := validGraphRequest()
	valid.EdgeBudget = 1
	entities := []GraphEntity{
		{EntityID: "ENT11111111-1111-4111-8111-111111111111"},
		{EntityID: "ICH22222222-2222-4222-8222-222222222222"},
	}
	relations := []GraphEntityRelation{
		{EntityRelationID: "ERL33333333-3333-4333-8333-333333333333", FromEntityID: entities[0].EntityID, ToEntityID: entities[1].EntityID, RelationType: "produces"},
		{EntityRelationID: "ERL44444444-4444-4444-8444-444444444444", FromEntityID: entities[0].EntityID, ToEntityID: entities[1].EntityID, RelationType: "produces"},
	}
	_, err := (&UseCase{graphStore: &graphStoreStub{graph: GraphSubgraph{
		Entities: entities, RelationDefinitions: []GraphRelationDefinition{{RelationType: "produces"}}, EntityRelations: relations,
	}}}).Search(context.Background(), valid)
	var resourceLimit *ResearchResourceLimitError
	if !errors.As(err, &resourceLimit) || resourceLimit.Component != "research_graph_edges" ||
		resourceLimit.ActualRows == nil || resourceLimit.MaxRows == nil ||
		*resourceLimit.ActualRows != 2 || *resourceLimit.MaxRows != 1 {
		t.Fatalf("edge budget error = %#v", err)
	}

	valid.EdgeBudget = 10
	_, err = (&UseCase{graphStore: &graphStoreStub{graph: GraphSubgraph{Entities: []GraphEntity{{
		EntityID: "ENT11111111-1111-4111-8111-111111111111", Name: strings.Repeat("x", GraphMaxResultBytes),
	}}}}}).Search(context.Background(), valid)
	if !errors.As(err, &resourceLimit) || resourceLimit.Component != "research_graph_result" ||
		resourceLimit.ActualBytes == nil || resourceLimit.MaxBytes == nil {
		t.Fatalf("response budget error = %#v", err)
	}
}

var _ GraphStore = (*graphStoreStub)(nil)
