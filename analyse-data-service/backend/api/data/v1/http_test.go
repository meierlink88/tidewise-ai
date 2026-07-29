package v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"gopkg.in/yaml.v3"
)

func TestDataBindingRunsKratosMiddlewareWithStableOperation(t *testing.T) {
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
	RegisterDataHTTPServer(server, testDataHTTPServer{})

	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, APIPrefix+"/events", nil))

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", response.Code)
	}
	if operation != "data.v1.listAdminEvents" {
		t.Fatalf("operation = %q, want data.v1.listAdminEvents", operation)
	}
}

func TestDataRuntimeRoutesMatchOpenAPIContract(t *testing.T) {
	routes := map[string]struct {
		requestPath string
		operation   string
		body        string
	}{
		"POST " + APIPrefix + "/reviewed-event-imports": {
			requestPath: APIPrefix + "/reviewed-event-imports",
			operation:   "data.v1.publishReviewedEvents",
		},
		"POST " + APIPrefix + "/research-theme-imports": {
			requestPath: APIPrefix + "/research-theme-imports",
			operation:   "data.v1.importResearchThemes",
		},
		"POST " + APIPrefix + "/research-reasoning-tree-imports": {
			requestPath: APIPrefix + "/research-reasoning-tree-imports",
			operation:   "data.v1.importResearchReasoningTrees",
			body:        `{"theme_id":"11111111-1111-4111-8111-111111111111","reasoning_trees":[]}`,
		},
		"GET " + APIPrefix + "/event-tags": {
			requestPath: APIPrefix + "/event-tags?active=true",
			operation:   "data.v1.listActiveEventTags",
		},
		"GET " + APIPrefix + "/research/themes": {
			requestPath: APIPrefix + "/research/themes",
			operation:   "data.v1.listResearchThemes",
		},
		"GET " + APIPrefix + "/research/themes/{theme_id}": {
			requestPath: APIPrefix + "/research/themes/theme-1",
			operation:   "data.v1.getResearchTheme",
		},
		"GET " + APIPrefix + "/research/themes/{theme_id}/reasoning-trees": {
			requestPath: APIPrefix + "/research/themes/theme-1/reasoning-trees",
			operation:   "data.v1.listResearchThemeReasoningTrees",
		},
		"GET " + APIPrefix + "/research/themes/{theme_id}/reasoning-trees/{reasoning_tree_id}": {
			requestPath: APIPrefix + "/research/themes/theme-1/reasoning-trees/tree-1",
			operation:   "data.v1.getResearchThemeReasoningTree",
		},
		"GET " + APIPrefix + "/raw-documents": {
			requestPath: APIPrefix + "/raw-documents",
			operation:   "data.v1.listAdminRawDocuments",
		},
		"GET " + APIPrefix + "/events": {
			requestPath: APIPrefix + "/events",
			operation:   "data.v1.listAdminEvents",
		},
		"GET " + APIPrefix + "/event-semantics/eligible-events": {
			requestPath: APIPrefix + "/event-semantics/eligible-events?limit=20",
			operation:   "data.v1.listEligibleEventSemanticEvents",
		},
		"POST " + APIPrefix + "/event-semantics/context-leases": {
			requestPath: APIPrefix + "/event-semantics/context-leases",
			operation:   "data.v1.createEventSemanticContextLease",
			body:        `{"event_id":"22222222-2222-4222-8222-222222222222","agent_execution_id":"semantic-execution-1","worker_id":"semantic-worker","lease_seconds":300}`,
		},
		"GET " + APIPrefix + "/event-semantics/context-leases/{context_lease_id}/context": {
			requestPath: APIPrefix + "/event-semantics/context-leases/11111111-1111-4111-8111-111111111111/context",
			operation:   "data.v1.getEventSemanticContext",
		},
		"POST " + APIPrefix + "/event-semantics/entity-resolutions": {
			requestPath: APIPrefix + "/event-semantics/entity-resolutions",
			operation:   "data.v1.resolveEventSemanticEntities",
			body:        `{"context_lease_id":"11111111-1111-4111-8111-111111111111","mentions":[]}`,
		},
		"POST " + APIPrefix + "/event-semantics/direct-targets:search": {
			requestPath: APIPrefix + "/event-semantics/direct-targets:search",
			operation:   "data.v1.searchEventSemanticDirectTargets",
			body:        `{"context_lease_id":"11111111-1111-4111-8111-111111111111","subject_entity_id":"22222222-2222-4222-8222-222222222222","allowed_target_types":["product"]}`,
		},
		"POST " + APIPrefix + "/event-semantics/submissions": {
			requestPath: APIPrefix + "/event-semantics/submissions",
			operation:   "data.v1.createEventSemanticSubmission",
		},
		"POST " + APIPrefix + "/event-semantics/submissions/{submission_id}/reviews": {
			requestPath: APIPrefix + "/event-semantics/submissions/11111111-1111-4111-8111-111111111111/reviews",
			operation:   "data.v1.submitEventSemanticReview",
		},
		"GET " + APIPrefix + "/events/{event_id}/semantics": {
			requestPath: APIPrefix + "/events/22222222-2222-4222-8222-222222222222/semantics",
			operation:   "data.v1.getEventSemantics",
		},
	}

	contractRoutes := make(map[string]string)
	for path, pathItemValue := range httpContractObject(t, loadHTTPContract(t)["paths"], "paths") {
		if !strings.HasPrefix(path, APIPrefix) {
			continue
		}
		pathItem := httpContractObject(t, pathItemValue, "path "+path)
		for _, method := range []string{"get", "post", "put", "patch", "delete"} {
			operationValue, exists := pathItem[method]
			if !exists {
				continue
			}
			operation := httpContractObject(t, operationValue, method+" "+path)
			contractRoutes[strings.ToUpper(method)+" "+path] = httpContractString(
				t,
				operation["x-client-drift-anchor"],
				method+" "+path+" x-client-drift-anchor",
			)
		}
	}
	if len(contractRoutes) != len(routes) {
		t.Fatalf("OpenAPI business route count = %d, runtime route count = %d", len(contractRoutes), len(routes))
	}

	for route, runtime := range routes {
		contractOperation, exists := contractRoutes[route]
		if !exists {
			t.Errorf("runtime route %q is absent from OpenAPI", route)
			continue
		}
		if contractOperation != runtime.operation {
			t.Errorf("%s operation = %q, want OpenAPI anchor %q", route, runtime.operation, contractOperation)
		}

		method, _, _ := strings.Cut(route, " ")
		body := runtime.body
		if body == "" {
			body = `{}`
		}
		server := kratoshttp.NewServer()
		RegisterDataHTTPServer(server, testDataHTTPServer{})
		response := httptest.NewRecorder()
		server.ServeHTTP(response, httptest.NewRequest(method, runtime.requestPath, strings.NewReader(body)))
		if response.Code != http.StatusNoContent {
			t.Errorf("%s returned status %d, want 204", route, response.Code)
		}
	}
}

func loadHTTPContract(t *testing.T) map[string]any {
	t.Helper()
	content, err := os.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatalf("read openapi.yaml: %v", err)
	}
	var document map[string]any
	if err := yaml.Unmarshal(content, &document); err != nil {
		t.Fatalf("parse openapi.yaml: %v", err)
	}
	return document
}

func httpContractObject(t *testing.T, value any, label string) map[string]any {
	t.Helper()
	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("%s = %#v, want object", label, value)
	}
	return object
}

func httpContractString(t *testing.T, value any, label string) string {
	t.Helper()
	text, ok := value.(string)
	if !ok || text == "" {
		t.Fatalf("%s = %#v, want non-empty string", label, value)
	}
	return text
}

type testDataHTTPServer struct{}

func testResponse[T any]() (*Response[T], error) {
	return &Response[T]{Status: http.StatusNoContent}, nil
}
func (testDataHTTPServer) ImportReviewedEvents(context.Context, *EventPublicationRequest) (*Response[EventPublicationResult], error) {
	return testResponse[EventPublicationResult]()
}
func (testDataHTTPServer) ListActiveEventTags(context.Context, *EventTagCatalogRequest) (*Response[EventTagCatalog], error) {
	return testResponse[EventTagCatalog]()
}
func (testDataHTTPServer) ImportResearchThemes(context.Context, *ResearchThemeImportRequest) (*Response[ResearchThemeImportResult], error) {
	return testResponse[ResearchThemeImportResult]()
}
func (testDataHTTPServer) ImportResearchReasoningTrees(context.Context, *ResearchReasoningTreeImportRequest) (*Response[ResearchReasoningTreeImportResult], error) {
	return testResponse[ResearchReasoningTreeImportResult]()
}
func (testDataHTTPServer) ListResearchThemes(context.Context, *ListResearchThemesRequest) (*Response[ResearchThemePage], error) {
	return testResponse[ResearchThemePage]()
}
func (testDataHTTPServer) GetResearchTheme(context.Context, *GetResearchThemeRequest) (*Response[ResearchThemeDetail], error) {
	return testResponse[ResearchThemeDetail]()
}
func (testDataHTTPServer) ListResearchReasoningTrees(context.Context, *ReasoningTreeListRequest) (*Response[ResearchReasoningTreeList], error) {
	return testResponse[ResearchReasoningTreeList]()
}
func (testDataHTTPServer) GetResearchReasoningTree(context.Context, *ReasoningTreeDetailRequest) (*Response[ResearchReasoningTreeDetail], error) {
	return testResponse[ResearchReasoningTreeDetail]()
}
func (testDataHTTPServer) ListRawDocuments(context.Context, *RawDocumentListRequest) (*Response[AdminRawDocumentPage], error) {
	return testResponse[AdminRawDocumentPage]()
}
func (testDataHTTPServer) ListEvents(context.Context, *EventListRequest) (*Response[AdminEventPage], error) {
	return testResponse[AdminEventPage]()
}
