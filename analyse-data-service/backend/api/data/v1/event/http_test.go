package event

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
)

func TestHTTPRoutesPreserveEventContract(t *testing.T) {
	for _, test := range []struct {
		method, path, body, operation string
	}{
		{http.MethodPost, v1.APIPrefix + "/reviewed-event-imports", `{}`, OperationPublishReviewedEvents},
		{http.MethodGet, v1.APIPrefix + "/event-tags?active=true", "", OperationListActiveEventTags},
		{http.MethodGet, v1.APIPrefix + "/events", "", OperationListAdminEvents},
	} {
		t.Run(test.operation, func(t *testing.T) {
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
			RegisterHTTPServer(server, testService{})
			response := httptest.NewRecorder()
			server.ServeHTTP(response, httptest.NewRequest(test.method, test.path, strings.NewReader(test.body)))
			if response.Code != http.StatusNoContent {
				t.Fatalf("status = %d, want 204: %s", response.Code, response.Body.String())
			}
			if operation != test.operation {
				t.Fatalf("operation = %q, want %q", operation, test.operation)
			}
		})
	}
}

func TestPublicationBindingAllowsArbitraryFactPayloadButRejectsUnknownFields(t *testing.T) {
	server := kratoshttp.NewServer()
	RegisterHTTPServer(server, testService{})
	valid := `{"package_id":"package","provenance":{"extractor_execution_id":"execution","extractor_agent_version":"v1","collector_executions":[]},"raw_documents":[],"events":[{"dedupe_key":"event","title":"title","factual_summary":"summary","fact_payload":{"nested":{"value":[1,true,null]}},"evidence":[],"tags":[],"review":{"review_id":"review","evidence_grade":"A","reasons":[]}}]}`
	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodPost, v1.APIPrefix+"/reviewed-event-imports", strings.NewReader(valid)))
	if response.Code != http.StatusNoContent {
		t.Fatalf("valid status = %d: %s", response.Code, response.Body.String())
	}

	var request PublicationRequest
	if err := v1.DecodeStrictJSON([]byte(`{"package_id":"package","unexpected":true}`), publicationShape(), &request); err == nil {
		t.Fatal("unknown top-level field was accepted")
	}
	if err := v1.DecodeStrictJSON([]byte(`{"package_id":null}`), publicationShape(), &request); err != nil {
		t.Fatalf("legacy null scalar was rejected at binding: %v", err)
	}
}

type testService struct{}

func (testService) PublishReviewedEvents(context.Context, *PublicationRequest) (*v1.Response[PublicationResult], error) {
	return testResponse[PublicationResult]()
}

func (testService) ListActiveEventTags(context.Context, *TagCatalogRequest) (*v1.Response[TagCatalog], error) {
	return testResponse[TagCatalog]()
}

func (testService) ListEvents(context.Context, *ListRequest) (*v1.Response[Page], error) {
	return testResponse[Page]()
}

func testResponse[T any]() (*v1.Response[T], error) {
	return &v1.Response[T]{Status: http.StatusNoContent}, nil
}
