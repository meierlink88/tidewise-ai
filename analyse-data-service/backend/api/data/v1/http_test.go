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

	if response.Code != 204 {
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
	}{
		"POST " + APIPrefix + "/reviewed-event-imports": {
			requestPath: APIPrefix + "/reviewed-event-imports",
			operation:   "data.v1.publishReviewedEvents",
		},
		"POST " + APIPrefix + "/research-theme-imports": {
			requestPath: APIPrefix + "/research-theme-imports",
			operation:   "data.v1.importResearchThemes",
		},
		"POST " + APIPrefix + "/research-anchor-imports": {
			requestPath: APIPrefix + "/research-anchor-imports",
			operation:   "data.v1.importResearchAnchors",
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
		"GET " + APIPrefix + "/research/themes/{theme_id}/reasoning-trees/{anchor_id}": {
			requestPath: APIPrefix + "/research/themes/theme-1/reasoning-trees/anchor-1",
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
		server := kratoshttp.NewServer()
		RegisterDataHTTPServer(server, testDataHTTPServer{})
		response := httptest.NewRecorder()
		server.ServeHTTP(response, httptest.NewRequest(method, runtime.requestPath, nil))
		if response.Code != 204 {
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

func (testDataHTTPServer) respond(ctx kratoshttp.Context) error { return ctx.JSON(204, nil) }
func (s testDataHTTPServer) ImportReviewedEvents(ctx kratoshttp.Context) error {
	return s.respond(ctx)
}
func (s testDataHTTPServer) ImportResearchThemes(ctx kratoshttp.Context) error {
	return s.respond(ctx)
}
func (s testDataHTTPServer) ImportResearchAnchors(ctx kratoshttp.Context) error {
	return s.respond(ctx)
}
func (s testDataHTTPServer) ListResearchThemes(ctx kratoshttp.Context) error {
	return s.respond(ctx)
}
func (s testDataHTTPServer) GetResearchTheme(ctx kratoshttp.Context) error {
	return s.respond(ctx)
}
func (s testDataHTTPServer) ListResearchReasoningTrees(ctx kratoshttp.Context) error {
	return s.respond(ctx)
}
func (s testDataHTTPServer) GetResearchReasoningTree(ctx kratoshttp.Context) error {
	return s.respond(ctx)
}
func (s testDataHTTPServer) ListRawDocuments(ctx kratoshttp.Context) error {
	return s.respond(ctx)
}
func (s testDataHTTPServer) ListEvents(ctx kratoshttp.Context) error { return s.respond(ctx) }
