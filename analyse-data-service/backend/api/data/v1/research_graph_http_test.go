package v1

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
)

type researchGraphHTTPStub struct {
	testDataHTTPServer
	request *ResearchGraphSearchRequest
}

func (s *researchGraphHTTPStub) SearchResearchGraph(
	_ context.Context,
	request *ResearchGraphSearchRequest,
) (*Response[ResearchGraphSearchResult], error) {
	s.request = request
	return &Response[ResearchGraphSearchResult]{
		Status: StatusOK,
		Result: ResearchGraphSearchResult{
			ContractVersion:          "research-graph-search.v1",
			AnalysisAsOf:             request.AnalysisAsOf,
			QueryFingerprint:         "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			GraphFingerprint:         "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			Entities:                 []ResearchAnalysisEntity{},
			RelationDefinitions:      []ResearchAnalysisRelationDefinition{},
			EntityRelations:          []ResearchAnalysisEntityRelation{},
			IndustryChains:           []ResearchAnalysisIndustryChain{},
			IndustryChainMemberships: []ResearchAnalysisIndustryChainMembership{},
			IndustryChainGraphEdges:  []ResearchAnalysisIndustryChainGraphEdge{},
		},
	}, nil
}

func TestResearchGraphSearchBindsStrictStructuredRequest(t *testing.T) {
	stub := &researchGraphHTTPStub{}
	server := kratoshttp.NewServer()
	RegisterDataHTTPServer(server, stub)
	request := httptest.NewRequest(
		http.MethodPost,
		APIPrefix+"/research-graph:search",
		bytes.NewBufferString(`{
			"analysis_as_of":"2026-07-30T00:00:00Z",
			"seed_entity_ids":["11111111-1111-4111-8111-111111111111"],
			"relation_filters":[{"relation_type":"produces","direction":"outgoing"}],
			"max_depth":2,
			"node_budget":100,
			"edge_budget":200
		}`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK ||
		stub.request == nil ||
		stub.request.MaxDepth != 2 ||
		len(stub.request.RelationFilters) != 1 ||
		stub.request.RelationFilters[0].Direction != "outgoing" {
		t.Fatalf("status=%d request=%#v body=%s", recorder.Code, stub.request, recorder.Body)
	}
}

func TestResearchGraphSearchRejectsUnknownFields(t *testing.T) {
	stub := &researchGraphHTTPStub{}
	server := kratoshttp.NewServer(
		kratoshttp.ErrorEncoder(func(response http.ResponseWriter, _ *http.Request, err error) {
			public, ok := err.(*PublicError)
			if !ok {
				t.Fatalf("error = %T %v", err, err)
			}
			response.WriteHeader(public.Status)
		}),
	)
	RegisterDataHTTPServer(server, stub)
	request := httptest.NewRequest(
		http.MethodPost,
		APIPrefix+"/research-graph:search",
		bytes.NewBufferString(`{"analysis_as_of":"2026-07-30T00:00:00Z","invented":true}`),
	)
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest || stub.request != nil {
		t.Fatalf("status=%d request=%#v", recorder.Code, stub.request)
	}
}

func TestResearchGraphSearchRejectsOversizedBody(t *testing.T) {
	stub := &researchGraphHTTPStub{}
	server := kratoshttp.NewServer(
		kratoshttp.ErrorEncoder(func(response http.ResponseWriter, _ *http.Request, err error) {
			public, ok := err.(*PublicError)
			if !ok {
				t.Fatalf("error = %T %v", err, err)
			}
			if public.Code != "PAYLOAD_TOO_LARGE" {
				t.Fatalf("error code = %q", public.Code)
			}
			response.WriteHeader(public.Status)
		}),
	)
	RegisterDataHTTPServer(server, stub)
	request := httptest.NewRequest(
		http.MethodPost,
		APIPrefix+"/research-graph:search",
		bytes.NewReader(bytes.Repeat([]byte("x"), MaxRequestBodySize+1)),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusRequestEntityTooLarge || stub.request != nil {
		t.Fatalf("status=%d request=%#v", recorder.Code, stub.request)
	}
}
