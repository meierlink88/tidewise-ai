package v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
)

type researchAnalysisContextHTTPStub struct {
	testDataHTTPServer
	request *ResearchAnalysisContextRequest
}

func (s *researchAnalysisContextHTTPStub) ListResearchAnalysisContext(
	_ context.Context,
	request *ResearchAnalysisContextRequest,
) (*Response[ResearchAnalysisContext], error) {
	s.request = request
	return &Response[ResearchAnalysisContext]{
		Status: StatusOK,
		Result: ResearchAnalysisContext{
			ContractVersion:             "research-analysis-context.v1",
			TBoxContractVersion:         "event-semantics.phase-one@1",
			TemporalSemantics:           "retrospective_reconstruction",
			TemporalLimitation:          "current-state dictionaries",
			EventPageFingerprint:        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			ReferenceClosureFingerprint: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			DiscoveryWindowStart:        request.DiscoveryWindowStart,
			DiscoveryWindowEnd:          request.DiscoveryWindowEnd,
			AnalysisAsOf:                request.AnalysisAsOf,
			EventSemanticBundles:        []ResearchAnalysisEventSemanticBundle{},
			Dictionaries:                ResearchAnalysisDictionaries{},
			HasMore:                     false,
		},
	}, nil
}

func TestResearchAnalysisContextBindsExplicitWindowAndPageSize(t *testing.T) {
	stub := &researchAnalysisContextHTTPStub{}
	server := kratoshttp.NewServer()
	RegisterDataHTTPServer(server, stub)
	request := httptest.NewRequest(
		http.MethodGet,
		APIPrefix+"/research-analysis-context"+
			"?discovery_window_start=2026-07-28T00%3A00%3A00Z"+
			"&discovery_window_end=2026-07-29T00%3A00%3A00Z"+
			"&analysis_as_of=2026-07-29T00%3A00%3A00Z&page_size=20",
		nil,
	)
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK || stub.request == nil ||
		stub.request.PageSize != 20 ||
		stub.request.DiscoveryWindowStart != "2026-07-28T00:00:00Z" {
		t.Fatalf("status=%d request=%#v body=%s", recorder.Code, stub.request, recorder.Body)
	}
}

func TestResearchAnalysisContextRejectsMissingOrUnknownQueryParameters(t *testing.T) {
	for _, target := range []string{
		APIPrefix + "/research-analysis-context?analysis_as_of=2026-07-29T00%3A00%3A00Z&page_size=20",
		APIPrefix + "/research-analysis-context?discovery_window_start=2026-07-28T00%3A00%3A00Z&discovery_window_end=2026-07-29T00%3A00%3A00Z&analysis_as_of=2026-07-29T00%3A00%3A00Z&page_size=20&invented=true",
	} {
		stub := &researchAnalysisContextHTTPStub{}
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
		recorder := httptest.NewRecorder()

		server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))

		if recorder.Code != http.StatusBadRequest || stub.request != nil {
			t.Fatalf("target=%s status=%d request=%#v", target, recorder.Code, stub.request)
		}
	}
}
