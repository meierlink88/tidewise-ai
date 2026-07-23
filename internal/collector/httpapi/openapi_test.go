package httpapi_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector/httpapi"
	"gopkg.in/yaml.v3"
)

type documentedApplication struct{}

func (documentedApplication) Ready(context.Context) error {
	return nil
}

func (documentedApplication) CreateCollectorRun(context.Context, string, string) (agentrun.Execution, agentrun.CreateDisposition, error) {
	return agentrun.Execution{}, agentrun.ExecutionCreated, nil
}

func (documentedApplication) GetCollectorRun(context.Context, string) (agentrun.Execution, error) {
	return agentrun.Execution{}, nil
}

func newDocumentedHTTPServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(httpapi.NewHandler(documentedApplication{}, "service-test-token"))
	t.Cleanup(server.Close)
	return server
}

func getDocumentedResponse(t *testing.T, server *httptest.Server, path string) (*http.Response, []byte) {
	t.Helper()
	response, err := server.Client().Get(server.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response, body
}

func TestAgentRunServesEmbeddedOpenAPIDocument(t *testing.T) {
	server := newDocumentedHTTPServer(t)

	response, body := getDocumentedResponse(t, server, "/openapi.yaml")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET /openapi.yaml status = %d, want 200", response.StatusCode)
	}
	if contentType := response.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "application/yaml") {
		t.Fatalf("GET /openapi.yaml Content-Type = %q", contentType)
	}
	var metadata struct {
		OpenAPI string `yaml:"openapi"`
		Info    struct {
			Version string `yaml:"version"`
		} `yaml:"info"`
	}
	if err := yaml.Unmarshal(body, &metadata); err != nil {
		t.Fatalf("decode OpenAPI document: %v", err)
	}
	if metadata.OpenAPI != "3.0.4" || metadata.Info.Version != "1.0.0" {
		t.Fatalf("OpenAPI metadata = %#v", metadata)
	}
	if strings.Contains(string(body), "service-test-token") {
		t.Fatal("OpenAPI document contains the runtime service token")
	}
}

func TestAgentRunOpenAPIContractMatchesHTTPInterface(t *testing.T) {
	server := newDocumentedHTTPServer(t)

	_, body := getDocumentedResponse(t, server, "/openapi.yaml")

	document, err := openapi3.NewLoader().LoadFromData(body)
	if err != nil {
		t.Fatalf("load OpenAPI document: %v", err)
	}
	if err := document.Validate(context.Background()); err != nil {
		t.Fatalf("validate OpenAPI document: %v", err)
	}
	if len(document.Security) != 0 {
		t.Fatalf("OpenAPI document unexpectedly declares global security: %#v", document.Security)
	}

	requiredOperations := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/healthz"},
		{method: http.MethodGet, path: "/readyz"},
		{method: http.MethodPost, path: "/internal/agent-run/v1/collector/runs"},
		{method: http.MethodGet, path: "/internal/agent-run/v1/collector/runs/{execution_id}"},
		{method: http.MethodGet, path: "/openapi.yaml"},
		{method: http.MethodGet, path: "/docs/"},
	}
	for _, required := range requiredOperations {
		path := document.Paths.Value(required.path)
		if path == nil || path.GetOperation(required.method) == nil {
			t.Errorf("OpenAPI document is missing %s %s", required.method, required.path)
		}
	}
	if t.Failed() {
		t.FailNow()
	}

	bearer := document.Components.SecuritySchemes["bearerAuth"]
	if bearer == nil || bearer.Value == nil || bearer.Value.Type != "http" || bearer.Value.Scheme != "bearer" {
		t.Fatalf("bearerAuth = %#v", bearer)
	}
	for _, public := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/healthz"},
		{method: http.MethodGet, path: "/readyz"},
		{method: http.MethodGet, path: "/openapi.yaml"},
		{method: http.MethodGet, path: "/docs/"},
	} {
		security := document.Paths.Value(public.path).GetOperation(public.method).Security
		if security == nil || len(*security) != 0 {
			t.Errorf("%s %s must explicitly override security with an empty requirement", public.method, public.path)
		}
	}

	create := document.Paths.Value("/internal/agent-run/v1/collector/runs").Post
	get := document.Paths.Value("/internal/agent-run/v1/collector/runs/{execution_id}").Get
	for name, operation := range map[string]*openapi3.Operation{"create": create, "get": get} {
		if operation.Security == nil || len(*operation.Security) != 1 {
			t.Fatalf("%s security = %#v", name, operation.Security)
		}
		if _, ok := (*operation.Security)[0]["bearerAuth"]; !ok {
			t.Fatalf("%s does not require bearerAuth", name)
		}
	}

	idempotencyKey := create.Parameters.GetByInAndName(openapi3.ParameterInHeader, "Idempotency-Key")
	if idempotencyKey == nil || !idempotencyKey.Required {
		t.Fatalf("Idempotency-Key parameter = %#v", idempotencyKey)
	}
	if create.RequestBody == nil || create.RequestBody.Value == nil || !create.RequestBody.Value.Required {
		t.Fatalf("Collector request body = %#v", create.RequestBody)
	}
	requestMedia := create.RequestBody.Value.Content.Get("application/json")
	if requestMedia == nil || requestMedia.Schema == nil || requestMedia.Schema.Value == nil {
		t.Fatalf("Collector JSON request schema = %#v", requestMedia)
	}
	requestSchema := requestMedia.Schema.Value
	if len(requestSchema.Properties) != 1 || requestSchema.Properties["prompt"] == nil ||
		len(requestSchema.Required) != 1 || requestSchema.Required[0] != "prompt" ||
		requestSchema.AdditionalProperties.Has == nil || *requestSchema.AdditionalProperties.Has {
		t.Fatalf("Collector request schema = %#v", requestSchema)
	}

	executionID := get.Parameters.GetByInAndName(openapi3.ParameterInPath, "execution_id")
	if executionID == nil || !executionID.Required || executionID.Schema == nil ||
		executionID.Schema.Value == nil || executionID.Schema.Value.Format != "uuid" {
		t.Fatalf("execution_id parameter = %#v", executionID)
	}

	for name, contract := range map[string]struct {
		operation *openapi3.Operation
		statuses  []string
	}{
		"create": {operation: create, statuses: []string{"202", "400", "401", "409", "413", "500", "503"}},
		"get":    {operation: get, statuses: []string{"200", "401", "404", "500"}},
	} {
		for _, status := range contract.statuses {
			if contract.operation.Responses.Value(status) == nil {
				t.Errorf("%s response is missing status %s", name, status)
			}
		}
	}

	for name, expected := range map[string][]any{
		"ExecutionStatus":  {"queued", "planning", "collecting", "materializing", "succeeded", "succeeded_no_change", "partially_succeeded", "failed", "skipped"},
		"InvocationStatus": {"pending", "running", "completed", "failed", "not_invoked"},
	} {
		schema := document.Components.Schemas[name]
		if schema == nil || schema.Value == nil {
			t.Fatalf("%s schema is missing", name)
		}
		if len(schema.Value.Enum) != len(expected) {
			t.Fatalf("%s enum = %#v", name, schema.Value.Enum)
		}
		for index := range expected {
			if schema.Value.Enum[index] != expected[index] {
				t.Fatalf("%s enum = %#v", name, schema.Value.Enum)
			}
		}
	}
}

func TestDocumentationMountPreservesExistingHTTPRoutes(t *testing.T) {
	server := newDocumentedHTTPServer(t)

	for _, request := range []struct {
		method     string
		path       string
		wantStatus int
	}{
		{method: http.MethodGet, path: "/healthz", wantStatus: http.StatusOK},
		{method: http.MethodGet, path: "/readyz", wantStatus: http.StatusOK},
		{method: http.MethodPost, path: "/internal/agent-run/v1/collector/runs", wantStatus: http.StatusUnauthorized},
		{method: http.MethodGet, path: "/internal/agent-run/v1/collector/runs/00000000-0000-0000-0000-000000000000", wantStatus: http.StatusUnauthorized},
	} {
		httpRequest, err := http.NewRequest(request.method, server.URL+request.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := server.Client().Do(httpRequest)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != request.wantStatus {
			t.Errorf("%s %s status = %d, want %d", request.method, request.path, response.StatusCode, request.wantStatus)
		}
	}
}

func TestAgentRunServesEmbeddedSwaggerUI(t *testing.T) {
	server := newDocumentedHTTPServer(t)

	response, body := getDocumentedResponse(t, server, "/docs/")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET /docs/ status = %d, want 200", response.StatusCode)
	}
	if contentType := response.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
		t.Fatalf("GET /docs/ Content-Type = %q", contentType)
	}
	html := string(body)
	if !strings.Contains(html, "/openapi.yaml") {
		t.Fatal("Swagger UI does not load /openapi.yaml")
	}
	if externalAsset := regexp.MustCompile(`(?i)(?:src|href)=["']https?://`); externalAsset.MatchString(html) {
		t.Fatal("Swagger UI HTML references an external asset")
	}

	for _, asset := range []struct {
		path        string
		contentType string
	}{
		{path: "/docs/swagger-ui-bundle.js", contentType: "javascript"},
		{path: "/docs/swagger-ui.css", contentType: "text/css"},
	} {
		assetResponse, _ := getDocumentedResponse(t, server, asset.path)
		if assetResponse.StatusCode != http.StatusOK {
			t.Fatalf("GET %s status = %d, want 200", asset.path, assetResponse.StatusCode)
		}
		if contentType := assetResponse.Header.Get("Content-Type"); !strings.Contains(contentType, asset.contentType) {
			t.Fatalf("GET %s Content-Type = %q", asset.path, contentType)
		}
	}
}
