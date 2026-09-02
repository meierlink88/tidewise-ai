package research

import (
	"context"
	"errors"
	"testing"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	researchapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/research"
	researchbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/research"
)

type useCaseStub struct {
	result  researchbiz.GraphSearchResult
	err     error
	request researchbiz.GraphSearchRequest
}

func (s *useCaseStub) Search(_ context.Context, request researchbiz.GraphSearchRequest) (researchbiz.GraphSearchResult, error) {
	s.request = request
	return s.result, s.err
}

func TestServiceRequiresResearchGraphUseCase(t *testing.T) {
	if _, err := NewService(nil); err == nil {
		t.Fatal("NewService(nil) error = nil")
	}
}

func TestServiceMapsResearchGraphResult(t *testing.T) {
	useCase := &useCaseStub{result: researchbiz.GraphSearchResult{
		ContractVersion:  researchbiz.GraphContractVersion,
		AnalysisAsOf:     "2026-07-30T00:00:00Z",
		QueryFingerprint: "query",
		GraphFingerprint: "graph",
		ActualDepth:      1,
		Entities: []researchbiz.GraphEntity{{
			EntityID: "ENT11111111-1111-4111-8111-111111111111",
			Name:     "Producer",
		}},
	}}
	service, err := NewService(useCase)
	if err != nil {
		t.Fatal(err)
	}
	response, err := service.SearchResearchGraph(context.Background(), &researchapi.ResearchGraphSearchRequest{
		AnalysisAsOf:  "2026-07-30T00:00:00Z",
		SeedEntityIDs: []string{"ENT11111111-1111-4111-8111-111111111111"},
		RelationFilters: []researchapi.ResearchGraphRelationFilter{{
			RelationType: "produces",
			Direction:    "outgoing",
		}},
		MaxDepth: 1, NodeBudget: 10, EdgeBudget: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Status != v1.StatusOK || response.Result.ContractVersion != researchbiz.GraphContractVersion ||
		len(response.Result.Entities) != 1 || useCase.request.RelationFilters[0].Direction != researchbiz.DirectionOutgoing {
		t.Fatalf("response=%#v request=%#v", response, useCase.request)
	}
}

func TestServiceKeepsStableResearchGraphErrorContracts(t *testing.T) {
	actual := int64(11)
	maximum := int64(10)
	tests := []struct {
		name       string
		domainErr  error
		wantStatus int
		wantCode   string
	}{
		{name: "validation", domainErr: &researchbiz.GraphValidationError{Reason: "invalid graph query"}, wantStatus: v1.StatusBadRequest, wantCode: "RESEARCH_GRAPH_INVALID"},
		{name: "resource", domainErr: &researchbiz.ResearchResourceLimitError{
			Reason: "too many nodes", Component: "research_graph_nodes", ActualRows: &actual, MaxRows: &maximum,
		}, wantStatus: v1.StatusTooManyRequests, wantCode: "RESEARCH_GRAPH_RESOURCE_LIMIT"},
		{name: "internal", domainErr: errors.New("database unavailable"), wantStatus: v1.StatusInternalServerError, wantCode: "RESEARCH_GRAPH_FAILED"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, err := NewService(&useCaseStub{err: test.domainErr})
			if err != nil {
				t.Fatal(err)
			}
			_, err = service.SearchResearchGraph(context.Background(), &researchapi.ResearchGraphSearchRequest{})
			var public *v1.PublicError
			if !errors.As(err, &public) || public.Status != test.wantStatus || public.Code != test.wantCode {
				t.Fatalf("public error = %#v, want status=%d code=%s", err, test.wantStatus, test.wantCode)
			}
		})
	}
}

var _ UseCase = (*useCaseStub)(nil)
