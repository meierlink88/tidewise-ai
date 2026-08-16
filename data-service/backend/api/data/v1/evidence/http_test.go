package evidence

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

type evidencePublicationHTTPStub struct {
	rawRequest      *RawEvidencePublicationRequest
	evidenceRequest *EvidencePublicationRequest
	deadline        time.Time
	blockUntilDone  bool
}

func (s *evidencePublicationHTTPStub) ListEvidenceCategories(ctx context.Context) (*v1.Response[EvidenceCategoryCatalog], error) {
	s.deadline, _ = ctx.Deadline()
	if s.blockUntilDone {
		<-ctx.Done()
		return nil, v1.NewPublicError(v1.StatusServiceUnavailable, ErrorEvidenceCategoryCatalogTimeout, "Evidence Category Catalog execution budget exceeded", nil)
	}
	return &v1.Response[EvidenceCategoryCatalog]{Status: v1.StatusOK, Result: EvidenceCategoryCatalog{
		Categories: []EvidenceCategory{{
			ID: "EVCc18ddddb-14bc-5496-99ea-963ee2c25597", Code: "EVENT_BRIEF",
			Name: "事件快讯", Description: "简短报告已经发生或正在发生的事件，核心目的是说明发生了什么。",
		}},
	}}, nil
}

func TestEvidenceCategoryCatalogHTTPRejectsQueryParameters(t *testing.T) {
	for _, rawQuery := range []string{"limit=1", "bad;param"} {
		t.Run(rawQuery, func(t *testing.T) {
			server := kratoshttp.NewServer(kratoshttp.ErrorEncoder(func(response http.ResponseWriter, _ *http.Request, err error) {
				public, ok := err.(*v1.PublicError)
				if !ok {
					t.Fatalf("error = %T %v, want PublicError", err, err)
				}
				response.WriteHeader(public.Status)
			}))
			RegisterHTTPServer(server, &evidencePublicationHTTPStub{})
			response := httptest.NewRecorder()
			server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, v1.APIPrefix+"/evidence-categories?"+rawQuery, nil))
			if response.Code != v1.StatusBadRequest {
				t.Fatalf("status=%d body=%s, want 400", response.Code, response.Body)
			}
		})
	}
}

func TestEvidenceCategoryCatalogHTTPReturnsSafe503WhenInternalBudgetExpires(t *testing.T) {
	server := kratoshttp.NewServer(kratoshttp.ErrorEncoder(func(response http.ResponseWriter, request *http.Request, err error) {
		public, ok := err.(*v1.PublicError)
		if !ok {
			t.Fatalf("error = %T %v, want PublicError", err, err)
		}
		response.WriteHeader(public.Status)
		_ = json.NewEncoder(response).Encode(map[string]any{
			"request_id": request.Header.Get("X-Request-ID"),
			"error":      map[string]any{"code": public.Code, "message": public.Message, "details": public.Details},
		})
	}))
	registerHTTPServer(server, &evidencePublicationHTTPStub{blockUntilDone: true}, 5*time.Millisecond)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, v1.APIPrefix+"/evidence-categories", nil)
	request.Header.Set("X-Request-ID", "catalog-timeout-request")
	server.ServeHTTP(response, request)
	if response.Code != v1.StatusServiceUnavailable ||
		!strings.Contains(response.Body.String(), `"code":"`+ErrorEvidenceCategoryCatalogTimeout+`"`) ||
		!strings.Contains(response.Body.String(), `"request_id":"catalog-timeout-request"`) {
		t.Fatalf("timeout response status=%d body=%s", response.Code, response.Body)
	}
}

func TestEvidenceCategoryCatalogHTTPRunsStableOperationWithBudget(t *testing.T) {
	var operation string
	recorder := func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, request any) (any, error) {
			if serverTransport, ok := transport.FromServerContext(ctx); ok {
				operation = serverTransport.Operation()
			}
			return next(ctx, request)
		}
	}
	stub := &evidencePublicationHTTPStub{}
	server := kratoshttp.NewServer(kratoshttp.Middleware(recorder))
	startedAt := time.Now()
	RegisterHTTPServer(server, stub)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, v1.APIPrefix+"/evidence-categories", nil))
	if response.Code != v1.StatusOK || operation != OperationListEvidenceCategories {
		t.Fatalf("status=%d operation=%q body=%s", response.Code, operation, response.Body)
	}
	if stub.deadline.IsZero() || stub.deadline.Sub(startedAt) <= 0 || stub.deadline.Sub(startedAt) > ExecutionBudget {
		t.Fatalf("Evidence Category Catalog deadline = %s", stub.deadline)
	}
	var catalog EvidenceCategoryCatalog
	if err := json.Unmarshal(response.Body.Bytes(), &catalog); err != nil {
		t.Fatal(err)
	}
	if len(catalog.Categories) != 1 || catalog.Categories[0].Code != "EVENT_BRIEF" {
		t.Fatalf("catalog = %#v", catalog)
	}
}

func TestEvidencePublicationHTTPRunsMiddlewareWithStableOperation(t *testing.T) {
	for _, test := range []struct {
		path      string
		body      string
		operation string
	}{
		{
			path:      v1.APIPrefix + "/raw-evidence-publications",
			body:      `{"raw_evidence":{"publication_key":"example-article-1","source_id":"SRC_example_00000000000000000000","source_name":"Example Wire","source_level":"L2_WIRE","source_url":"https://example.test/article/1","is_original":true,"raw_text":"article","collected_at":"2026-08-11T01:05:00Z","keywords":[]}}`,
			operation: OperationPublishRawEvidence,
		},
		{
			path:      v1.APIPrefix + "/evidence-publications",
			body:      `{"raw_evidence_id":"RAW15bec7e3-998c-5434-aa5d-29712c4c67cf","evidences":[{"split_order":0,"layer_type":"SINGLE","source_what":"fact","expression_fingerprint":"fact","expression_key":"fact-v1","fingerprint_version":"v1"}]}`,
			operation: OperationPublishEvidence,
		},
	} {
		var operation string
		recorder := func(next middleware.Handler) middleware.Handler {
			return func(ctx context.Context, request any) (any, error) {
				if serverTransport, ok := transport.FromServerContext(ctx); ok {
					operation = serverTransport.Operation()
				}
				return next(ctx, request)
			}
		}
		server := kratoshttp.NewServer(kratoshttp.Middleware(recorder))
		RegisterHTTPServer(server, &evidencePublicationHTTPStub{})
		response := httptest.NewRecorder()
		server.ServeHTTP(response, httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body)))
		if response.Code != v1.StatusCreated || operation != test.operation {
			t.Errorf("%s status=%d operation=%q, want 201 %q", test.path, response.Code, operation, test.operation)
		}
	}
}

func (s *evidencePublicationHTTPStub) PublishRawEvidence(ctx context.Context, request *RawEvidencePublicationRequest) (*v1.Response[RawEvidencePublicationResult], error) {
	s.rawRequest = request
	s.deadline, _ = ctx.Deadline()
	if s.blockUntilDone {
		<-ctx.Done()
		return nil, v1.NewPublicError(v1.StatusServiceUnavailable, ErrorEvidencePublicationTimeout, "Evidence Publication execution budget exceeded", nil)
	}
	return &v1.Response[RawEvidencePublicationResult]{Status: v1.StatusCreated, Result: RawEvidencePublicationResult{
		ID: "RAW15bec7e3-998c-5434-aa5d-29712c4c67cf",
	}}, nil
}

func (s *evidencePublicationHTTPStub) GetRawEvidence(ctx context.Context, request *GetRawEvidenceRequest) (*v1.Response[RawEvidenceReadResult], error) {
	s.deadline, _ = ctx.Deadline()
	if s.blockUntilDone {
		<-ctx.Done()
		return nil, v1.NewPublicError(v1.StatusServiceUnavailable, ErrorRawEvidenceReadTimeout, "Raw Evidence read execution budget exceeded", nil)
	}
	return &v1.Response[RawEvidenceReadResult]{Status: v1.StatusOK, Result: RawEvidenceReadResult{
		RawEvidence: RawEvidenceRead{ID: request.ID, Keywords: []string{}, Categories: []EvidenceCategory{}},
	}}, nil
}

func (s *evidencePublicationHTTPStub) PublishEvidence(ctx context.Context, request *EvidencePublicationRequest) (*v1.Response[EvidencePublicationResult], error) {
	s.evidenceRequest = request
	s.deadline, _ = ctx.Deadline()
	if s.blockUntilDone {
		<-ctx.Done()
		return nil, v1.NewPublicError(v1.StatusServiceUnavailable, ErrorEvidencePublicationTimeout, "Evidence Publication execution budget exceeded", nil)
	}
	return &v1.Response[EvidencePublicationResult]{Status: v1.StatusCreated, Result: EvidencePublicationResult{
		RawEvidenceID: request.RawEvidenceID,
		IDs:           []string{"EVD5cb71bef-5b1d-5995-add0-7408eaa2be15"},
	}}, nil
}

func TestRawEvidencePublicationHTTPPreservesPublisherKeywords(t *testing.T) {
	stub := &evidencePublicationHTTPStub{}
	server := kratoshttp.NewServer()
	RegisterHTTPServer(server, stub)
	body := `{"raw_evidence":{"publication_key":"example-article-1","source_id":"SRC_example_00000000000000000000","source_name":"Example Wire","source_level":"L2_WIRE","source_url":"https://example.test/article/1","is_original":true,"raw_text":"Complete original article.","collected_at":"2026-08-11T01:05:00Z","keywords":[" AI芯片 ","供应链","AI芯片"]}}`
	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodPost, v1.APIPrefix+"/raw-evidence-publications", strings.NewReader(body)))

	if response.Code != v1.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", response.Code, response.Body.String())
	}
	if stub.rawRequest == nil || !equalStrings(stub.rawRequest.RawEvidence.Keywords, []string{" AI芯片 ", "供应链", "AI芯片"}) {
		t.Fatalf("bound keywords = %#v", stub.rawRequest)
	}
	var result RawEvidencePublicationResult
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.ID != "RAW15bec7e3-998c-5434-aa5d-29712c4c67cf" {
		t.Fatalf("response ID = %q", result.ID)
	}
}

func TestEvidencePublicationHTTPRejectsDuplicateAndUnknownFields(t *testing.T) {
	for _, body := range []string{
		`{"raw_evidence_id":"RAW15bec7e3-998c-5434-aa5d-29712c4c67cf","raw_evidence_id":"RAW15bec7e3-998c-5434-aa5d-29712c4c67cf","evidences":[]}`,
		`{"raw_evidence_id":"RAW15bec7e3-998c-5434-aa5d-29712c4c67cf","evidences":[],"group_id":"not-a-resource"}`,
		`{"raw_evidence_id":"RAW15bec7e3-998c-5434-aa5d-29712c4c67cf","evidences":[{"evidence_id":"EVD5cb71bef-5b1d-5995-add0-7408eaa2be15","split_order":null,"layer_type":"SINGLE","source_what":"fact","expression_fingerprint":"fact","expression_key":"fact-v1","fingerprint_version":"v1"}]}`,
	} {
		server := kratoshttp.NewServer(kratoshttp.ErrorEncoder(func(response http.ResponseWriter, _ *http.Request, err error) {
			public, ok := err.(*v1.PublicError)
			if !ok {
				t.Fatalf("error = %T %v", err, err)
			}
			response.WriteHeader(public.Status)
		}))
		RegisterHTTPServer(server, &evidencePublicationHTTPStub{})
		response := httptest.NewRecorder()
		server.ServeHTTP(response, httptest.NewRequest(http.MethodPost, v1.APIPrefix+"/evidence-publications", strings.NewReader(body)))
		if response.Code != v1.StatusBadRequest {
			t.Fatalf("status = %d, want 400: %s", response.Code, response.Body.String())
		}
	}
}

func TestEvidencePublicationHTTPAppliesInternalExecutionBudget(t *testing.T) {
	for _, test := range []struct {
		name string
		path string
		body string
	}{
		{
			name: "Raw Evidence",
			path: v1.APIPrefix + "/raw-evidence-publications",
			body: `{"raw_evidence":{"publication_key":"example-article-1","source_id":"SRC_example_00000000000000000000","source_name":"Example Wire","source_level":"L2_WIRE","source_url":"https://example.test/article/1","is_original":true,"raw_text":"article","collected_at":"2026-08-11T01:05:00Z","keywords":[]}}`,
		},
		{
			name: "Evidence",
			path: v1.APIPrefix + "/evidence-publications",
			body: `{"raw_evidence_id":"RAW15bec7e3-998c-5434-aa5d-29712c4c67cf","evidences":[{"split_order":0,"layer_type":"SINGLE","source_what":"fact","expression_fingerprint":"fact","expression_key":"fact-v1","fingerprint_version":"v1"}]}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			stub := &evidencePublicationHTTPStub{}
			server := kratoshttp.NewServer()
			startedAt := time.Now()
			RegisterHTTPServer(server, stub)
			server.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body)))
			if stub.deadline.IsZero() {
				t.Fatal("Evidence service context has no deadline")
			}
			remaining := stub.deadline.Sub(startedAt)
			if remaining <= 0 || remaining > ExecutionBudget {
				t.Fatalf("Evidence execution budget = %s, want (0, %s]", remaining, ExecutionBudget)
			}
		})
	}
}

func TestEvidencePublicationHTTPReturnsSafe503WhenInternalBudgetExpires(t *testing.T) {
	for _, test := range []struct {
		path string
		body string
	}{
		{
			path: v1.APIPrefix + "/raw-evidence-publications",
			body: `{"raw_evidence":{"publication_key":"example-article-1","source_id":"SRC_example_00000000000000000000","source_name":"Example Wire","source_level":"L2_WIRE","source_url":"https://example.test/article/1","is_original":true,"raw_text":"article","collected_at":"2026-08-11T01:05:00Z","keywords":[]}}`,
		},
		{
			path: v1.APIPrefix + "/evidence-publications",
			body: `{"raw_evidence_id":"RAW15bec7e3-998c-5434-aa5d-29712c4c67cf","evidences":[{"split_order":0,"layer_type":"SINGLE","source_what":"fact","expression_fingerprint":"fact","expression_key":"fact-v1","fingerprint_version":"v1"}]}`,
		},
	} {
		server := kratoshttp.NewServer(kratoshttp.ErrorEncoder(func(response http.ResponseWriter, request *http.Request, err error) {
			public, ok := err.(*v1.PublicError)
			if !ok {
				t.Fatalf("error = %T %v, want PublicError", err, err)
			}
			response.WriteHeader(public.Status)
			_ = json.NewEncoder(response).Encode(map[string]any{
				"request_id": request.Header.Get("X-Request-ID"),
				"error":      map[string]any{"code": public.Code, "message": public.Message, "details": public.Details},
			})
		}))
		registerHTTPServer(server, &evidencePublicationHTTPStub{blockUntilDone: true}, 5*time.Millisecond)
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body))
		request.Header.Set("X-Request-ID", "timeout-request")
		server.ServeHTTP(response, request)
		if response.Code != v1.StatusServiceUnavailable ||
			!strings.Contains(response.Body.String(), `"code":"`+ErrorEvidencePublicationTimeout+`"`) ||
			!strings.Contains(response.Body.String(), `"request_id":"timeout-request"`) {
			t.Errorf("%s timeout response status=%d body=%s", test.path, response.Code, response.Body)
		}
	}
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
