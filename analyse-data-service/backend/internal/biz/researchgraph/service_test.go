package researchgraph

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type graphStoreStub struct {
	query Query
	graph Subgraph
}

func (s *graphStoreStub) Search(_ context.Context, query Query) (Subgraph, error) {
	s.query = query
	return s.graph, nil
}

func TestServiceReturnsDeterministicReferenceCompleteGraph(t *testing.T) {
	store := &graphStoreStub{graph: Subgraph{
		ActualDepth: 1,
		Entities: []Entity{
			{
				EntityID:   "11111111-1111-4111-8111-111111111111",
				EntityType: "company",
				Name:       "Producer", CanonicalName: "producer", Status: "active",
			},
			{
				EntityID:   "22222222-2222-4222-8222-222222222222",
				EntityType: "product",
				Name:       "Product", CanonicalName: "product", Status: "active",
			},
		},
		RelationDefinitions: []RelationDefinition{{
			RelationType: "produces", Direction: "directed",
		}},
		EntityRelations: []EntityRelation{{
			EntityRelationID: "33333333-3333-4333-8333-333333333333",
			FromEntityID:     "11111111-1111-4111-8111-111111111111",
			ToEntityID:       "22222222-2222-4222-8222-222222222222",
			RelationType:     "produces",
			Status:           "active",
		}},
		IndustryChains:           []IndustryChain{},
		IndustryChainMemberships: []IndustryChainMembership{},
		IndustryChainGraphEdges:  []IndustryChainGraphEdge{},
	}}
	result, err := NewService(store).Search(context.Background(), Request{
		AnalysisAsOf:  "2026-07-30T00:00:00Z",
		SeedEntityIDs: []string{"11111111-1111-4111-8111-111111111111"},
		RelationFilters: []RelationFilter{{
			RelationType: "produces",
			Direction:    DirectionOutgoing,
		}},
		MaxDepth:   1,
		NodeBudget: 10,
		EdgeBudget: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ContractVersion != "research-graph-search.v1" ||
		result.ActualDepth != 1 ||
		!testHashPattern(result.QueryFingerprint) ||
		!testHashPattern(result.GraphFingerprint) {
		t.Fatalf("result = %#v", result)
	}
	if len(result.Entities) != 2 ||
		len(result.EntityRelations) != 1 ||
		store.query.RelationFilters[0].Direction != DirectionOutgoing {
		t.Fatalf("result = %#v query = %#v", result, store.query)
	}
}

func testHashPattern(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') &&
			(character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func TestServiceRejectsInvalidOrOrphanedGraphRequests(t *testing.T) {
	valid := Request{
		AnalysisAsOf:  "2026-07-30T00:00:00Z",
		SeedEntityIDs: []string{"11111111-1111-4111-8111-111111111111"},
		RelationFilters: []RelationFilter{{
			RelationType: "produces",
			Direction:    DirectionOutgoing,
		}},
		MaxDepth:   1,
		NodeBudget: 10,
		EdgeBudget: 10,
	}
	for _, mutate := range []func(*Request){
		func(request *Request) { request.SeedEntityIDs = nil },
		func(request *Request) {
			request.SeedEntityIDs = append(request.SeedEntityIDs, request.SeedEntityIDs[0])
		},
		func(request *Request) { request.RelationFilters[0].Direction = "sideways" },
		func(request *Request) {
			request.RelationFilters = append(
				request.RelationFilters,
				RelationFilter{
					RelationType: "produces",
					Direction:    DirectionIncoming,
				},
			)
		},
		func(request *Request) { request.MaxDepth = MaxDepth + 1 },
		func(request *Request) { request.NodeBudget = 0 },
	} {
		request := valid
		request.SeedEntityIDs = append([]string(nil), valid.SeedEntityIDs...)
		request.RelationFilters = append([]RelationFilter(nil), valid.RelationFilters...)
		mutate(&request)
		if _, err := NewService(&graphStoreStub{}).Search(
			context.Background(),
			request,
		); err == nil {
			t.Fatalf("request unexpectedly accepted: %#v", request)
		}
	}

	store := &graphStoreStub{graph: Subgraph{
		Entities: []Entity{{
			EntityID:   "11111111-1111-4111-8111-111111111111",
			EntityType: "company",
		}},
		RelationDefinitions: []RelationDefinition{{
			RelationType: "produces", Direction: "directed",
		}},
		EntityRelations: []EntityRelation{{
			EntityRelationID: "33333333-3333-4333-8333-333333333333",
			FromEntityID:     "11111111-1111-4111-8111-111111111111",
			ToEntityID:       "22222222-2222-4222-8222-222222222222",
			RelationType:     "produces",
		}},
	}}
	if _, err := NewService(store).Search(context.Background(), valid); err == nil {
		t.Fatal("orphaned graph edge was accepted")
	}
}

func TestServiceReportsTheExceededGraphBudgetDimension(t *testing.T) {
	valid := Request{
		AnalysisAsOf:  "2026-07-30T00:00:00Z",
		SeedEntityIDs: []string{"11111111-1111-4111-8111-111111111111"},
		RelationFilters: []RelationFilter{{
			RelationType: "produces",
			Direction:    DirectionOutgoing,
		}},
		MaxDepth: 1, NodeBudget: 10, EdgeBudget: 1,
	}
	entities := []Entity{
		{EntityID: "11111111-1111-4111-8111-111111111111"},
		{EntityID: "22222222-2222-4222-8222-222222222222"},
	}
	relations := []EntityRelation{
		{
			EntityRelationID: "33333333-3333-4333-8333-333333333333",
			FromEntityID:     entities[0].EntityID,
			ToEntityID:       entities[1].EntityID,
			RelationType:     "produces",
		},
		{
			EntityRelationID: "44444444-4444-4444-8444-444444444444",
			FromEntityID:     entities[0].EntityID,
			ToEntityID:       entities[1].EntityID,
			RelationType:     "produces",
		},
	}
	_, err := NewService(&graphStoreStub{graph: Subgraph{
		Entities:            entities,
		RelationDefinitions: []RelationDefinition{{RelationType: "produces"}},
		EntityRelations:     relations,
	}}).Search(context.Background(), valid)
	var resourceLimit *ResourceLimitError
	if !errors.As(err, &resourceLimit) ||
		resourceLimit.Component != "research_graph_edges" ||
		resourceLimit.ActualRows == nil ||
		resourceLimit.MaxRows == nil ||
		*resourceLimit.ActualRows != 2 ||
		*resourceLimit.MaxRows != 1 {
		t.Fatalf("edge budget error = %#v", err)
	}

	valid.EdgeBudget = 10
	_, err = NewService(&graphStoreStub{graph: Subgraph{
		Entities: []Entity{{
			EntityID: "11111111-1111-4111-8111-111111111111",
			Name:     strings.Repeat("x", MaxResultBytes),
		}},
	}}).Search(context.Background(), valid)
	if !errors.As(err, &resourceLimit) ||
		resourceLimit.Component != "research_graph_result" ||
		resourceLimit.ActualBytes == nil ||
		resourceLimit.MaxBytes == nil {
		t.Fatalf("response budget error = %#v", err)
	}
}
