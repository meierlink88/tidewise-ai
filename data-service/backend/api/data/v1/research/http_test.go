package research

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

type httpStub struct {
	graphRequest  *ResearchGraphSearchRequest
	graphDeadline time.Time
}

func (s *httpStub) SearchResearchGraph(ctx context.Context, request *ResearchGraphSearchRequest) (*v1.Response[ResearchGraphSearchResult], error) {
	s.graphRequest = request
	s.graphDeadline, _ = ctx.Deadline()
	return &v1.Response[ResearchGraphSearchResult]{Status: v1.StatusOK, Result: ResearchGraphSearchResult{
		ContractVersion:          "research-graph-search.v2",
		AnalysisAsOf:             request.AnalysisAsOf,
		Entities:                 []ResearchGraphEntity{},
		RelationDefinitions:      []ResearchGraphRelationDefinition{},
		EntityRelations:          []ResearchGraphEntityRelation{},
		IndustryChains:           []ResearchGraphIndustryChain{},
		IndustryChainMemberships: []ResearchGraphIndustryChainMembership{},
		IndustryChainGraphEdges:  []ResearchGraphIndustryChainGraphEdge{},
	}}, nil
}

func TestHTTPRegistersOnlyResearchGraphContract(t *testing.T) {
	application := &httpStub{}
	server := kratoshttp.NewServer()
	RegisterHTTPServer(server, application)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, v1.APIPrefix+"/research-graph:search", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	server.ServeHTTP(recorder, request)
	if recorder.Code == http.StatusNotFound {
		t.Fatal("Research Graph route is not registered")
	}

	for _, route := range []struct {
		method string
		path   string
	}{
		{http.MethodPost, v1.APIPrefix + "/research-theme-imports"},
		{http.MethodGet, v1.APIPrefix + "/research/themes"},
		{http.MethodGet, v1.APIPrefix + "/research/themes/theme-id"},
		{http.MethodGet, v1.APIPrefix + "/research/themes/theme-id/reasoning-trees"},
		{http.MethodGet, v1.APIPrefix + "/research/themes/theme-id/reasoning-trees/tree-id"},
	} {
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, httptest.NewRequest(route.method, route.path, nil))
		if recorder.Code != http.StatusNotFound {
			t.Errorf("retired route %s %s returned %d, want 404", route.method, route.path, recorder.Code)
		}
	}
}

func TestResearchGraphHTTPBindsStrictRequestAndAppliesBudget(t *testing.T) {
	valid := `{"analysis_as_of":"2026-07-30T00:00:00Z","seed_entity_ids":["11111111-1111-4111-8111-111111111111"],"relation_filters":[{"relation_type":"produces","direction":"outgoing"}],"max_depth":2,"node_budget":100,"edge_budget":200}`
	application := &httpStub{}
	server := kratoshttp.NewServer()
	RegisterHTTPServer(server, application)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, v1.APIPrefix+"/research-graph:search", strings.NewReader(valid))
	request.Header.Set("Content-Type", "application/json")
	server.ServeHTTP(recorder, request)
	remaining := time.Until(application.graphDeadline)
	if recorder.Code != http.StatusOK || application.graphRequest == nil || application.graphRequest.MaxDepth != 2 {
		t.Fatalf("status=%d request=%#v body=%s", recorder.Code, application.graphRequest, recorder.Body.String())
	}
	if application.graphDeadline.IsZero() || remaining <= 0 || remaining > HeavyExecutionBudget {
		t.Fatalf("graph deadline remaining=%s", remaining)
	}
}

func TestResearchGraphHTTPRejectsUnknownOrOversizedBodies(t *testing.T) {
	for _, test := range []struct {
		name, payload string
		want          int
	}{
		{name: "unknown", payload: `{"analysis_as_of":"2026-07-30T00:00:00Z","invented":true}`, want: http.StatusBadRequest},
		{name: "oversized", payload: string(bytes.Repeat([]byte("x"), v1.MaxRequestBodySize+1)), want: http.StatusRequestEntityTooLarge},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := kratoshttp.NewServer(kratoshttp.ErrorEncoder(publicErrorStatusEncoder))
			RegisterHTTPServer(server, &httpStub{})
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, v1.APIPrefix+"/research-graph:search", strings.NewReader(test.payload))
			request.Header.Set("Content-Type", "application/json")
			server.ServeHTTP(recorder, request)
			if recorder.Code != test.want {
				t.Fatalf("status=%d, want=%d body=%s", recorder.Code, test.want, recorder.Body.String())
			}
		})
	}
}

func publicErrorStatusEncoder(writer http.ResponseWriter, _ *http.Request, err error) {
	var public *v1.PublicError
	if errors.As(err, &public) {
		writer.WriteHeader(public.Status)
		return
	}
	writer.WriteHeader(http.StatusInternalServerError)
}

var _ Service = (*httpStub)(nil)
