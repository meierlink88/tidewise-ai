package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	dataapi "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/conf"
	"gopkg.in/yaml.v3"
)

func TestHealthAndReadiness(t *testing.T) {
	assertServiceHealth(t, testHTTPHandler(testConfig(), nil), ServiceName)
}

func TestHandlerPublishesEmbeddedOpenAPIOutsideProduction(t *testing.T) {
	for _, environment := range []conf.Environment{conf.EnvLocal, conf.EnvUAT} {
		cfg := testConfig()
		cfg.App.Env = environment
		response := httptest.NewRecorder()
		testHTTPHandler(cfg, nil).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("%s openapi status = %d, want %d", environment, response.Code, http.StatusOK)
		}
		if !strings.HasPrefix(response.Body.String(), "openapi: 3.0.4\n") {
			t.Fatalf("%s openapi document does not declare 3.0.4", environment)
		}
	}
}

func TestHandlerDoesNotPublishOpenAPIInProduction(t *testing.T) {
	cfg := testConfig()
	cfg.App.Env = conf.EnvProd

	response := httptest.NewRecorder()
	testHTTPHandler(cfg, nil).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("production openapi status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestOperationalResponseFieldsMatchOpenAPI(t *testing.T) {
	var document map[string]any
	if err := yaml.Unmarshal(dataapi.Document(), &document); err != nil {
		t.Fatalf("parse OpenAPI: %v", err)
	}
	paths := document["paths"].(map[string]any)
	handler := testHTTPHandler(testConfig(), nil)
	for _, path := range []string{"/healthz", "/readyz"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		var body map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		operation := paths[path].(map[string]any)["get"].(map[string]any)
		responses := operation["responses"].(map[string]any)
		content := responses["200"].(map[string]any)["content"].(map[string]any)
		schema := content["application/json"].(map[string]any)["schema"].(map[string]any)
		assertJSONKeysMatchProperties(t, path, body, schema["properties"].(map[string]any))
	}
}

func TestServerComposesDataAPIWithHealth(t *testing.T) {
	api := serverTestDataService{}
	handler := testHTTPHandler(testConfig(), api)
	assertServiceHealth(t, handler, ServiceName)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/data/v1/events", nil)
	request.Header.Set("Authorization", "Bearer admin-token")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("data API status = %d, want %d", response.Code, http.StatusNoContent)
	}
}

func TestServerOwnsAuthenticationAuthorizationAndPrincipalInjection(t *testing.T) {
	authenticator, err := NewAuthenticator([]Credential{
		{
			Secret: "admin-token",
			Principal: dataapi.Principal{
				Identity: "admin-portal-bff",
				Scopes:   []string{ScopeAdminRead},
			},
		},
		{
			Secret: "research-token",
			Principal: dataapi.Principal{
				Identity: "miniapp-bff",
				Scopes:   []string{ScopeResearchRead},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	application := &principalRecordingDataService{}
	handler := NewHTTPServer(testConfig(), application, authenticator, nil).Server.Handler

	for _, test := range []struct {
		name       string
		token      string
		wantStatus int
		wantCode   string
	}{
		{name: "missing credential", wantStatus: http.StatusUnauthorized, wantCode: "UNAUTHENTICATED"},
		{name: "wrong scope", token: "research-token", wantStatus: http.StatusForbidden, wantCode: "FORBIDDEN"},
		{name: "authorized", token: "admin-token", wantStatus: http.StatusNoContent},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, dataapi.APIPrefix+"/events", nil)
			if test.token != "" {
				request.Header.Set("Authorization", "Bearer "+test.token)
			}
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, test.wantStatus, response.Body)
			}
			if test.wantCode != "" && !strings.Contains(response.Body.String(), `"code":"`+test.wantCode+`"`) {
				t.Fatalf("body = %s, want error code %s", response.Body, test.wantCode)
			}
		})
	}

	if application.identity != "admin-portal-bff" {
		t.Fatalf("service principal = %q, want admin-portal-bff", application.identity)
	}
}

func TestAuthenticatorRejectsInvalidCredentials(t *testing.T) {
	valid := Credential{
		Secret: "token",
		Principal: dataapi.Principal{
			Identity: "caller",
			Scopes:   []string{ScopeAdminRead},
		},
	}
	for _, test := range []struct {
		name        string
		credentials []Credential
	}{
		{name: "missing secret", credentials: []Credential{{Principal: valid.Principal}}},
		{name: "missing identity", credentials: []Credential{{Secret: valid.Secret, Principal: dataapi.Principal{Scopes: valid.Principal.Scopes}}}},
		{name: "missing scope", credentials: []Credential{{Secret: valid.Secret, Principal: dataapi.Principal{Identity: valid.Principal.Identity}}}},
		{name: "oversized identity", credentials: []Credential{{
			Secret: valid.Secret,
			Principal: dataapi.Principal{
				Identity: strings.Repeat("界", 201),
				Scopes:   valid.Principal.Scopes,
			},
		}}},
		{name: "duplicate secret", credentials: []Credential{valid, {
			Secret: valid.Secret,
			Principal: dataapi.Principal{
				Identity: "other-caller",
				Scopes:   []string{ScopeResearchRead},
			},
		}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewAuthenticator(test.credentials); err == nil {
				t.Fatal("NewAuthenticator() error = nil")
			}
		})
	}
}

func TestEveryBusinessOperationHasAnAuthenticationScope(t *testing.T) {
	var document map[string]any
	if err := yaml.Unmarshal(dataapi.Document(), &document); err != nil {
		t.Fatalf("parse OpenAPI: %v", err)
	}
	openAPIOperations := map[string]struct{}{}
	for path, rawPath := range document["paths"].(map[string]any) {
		if !strings.HasPrefix(path, dataapi.APIPrefix) {
			continue
		}
		pathItem := rawPath.(map[string]any)
		for _, method := range []string{"get", "post", "put", "patch", "delete"} {
			rawOperation, exists := pathItem[method]
			if !exists {
				continue
			}
			operation, ok := rawOperation.(map[string]any)
			if !ok {
				continue
			}
			anchor, ok := operation["x-client-drift-anchor"].(string)
			if !ok || anchor == "" {
				t.Fatalf("OpenAPI business route %q has no client drift anchor", path)
			}
			if _, duplicate := openAPIOperations[anchor]; duplicate {
				t.Fatalf("OpenAPI business operation %q is duplicated", anchor)
			}
			openAPIOperations[anchor] = struct{}{}
		}
	}
	for _, operation := range dataapi.BusinessOperations {
		if _, exists := openAPIOperations[operation]; !exists {
			t.Errorf("business operation %q is absent from OpenAPI", operation)
		}
		if scope, ok := requiredScope(operation); !ok || scope == "" {
			t.Errorf("business operation %q has no authentication scope", operation)
		}
		delete(openAPIOperations, operation)
	}
	for operation := range openAPIOperations {
		t.Errorf("OpenAPI business operation %q is absent from BusinessOperations", operation)
	}
	if _, ok := requiredScope("data.v1.futureOperation"); ok {
		t.Fatal("unknown business operation was implicitly authorized")
	}
}

func TestServerRecoveryPreservesStableErrorEnvelope(t *testing.T) {
	authenticator, err := NewAuthenticator([]Credential{{
		Secret: "admin-token",
		Principal: dataapi.Principal{
			Identity: "admin-portal-bff",
			Scopes:   []string{ScopeAdminRead},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHTTPServer(testConfig(), panickingDataService{}, authenticator, nil).Server.Handler
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, dataapi.APIPrefix+"/events", nil)
	request.Header.Set("Authorization", "Bearer admin-token")
	request.Header.Set("X-Request-ID", "panic-request")
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", response.Code, response.Body)
	}
	if !strings.Contains(response.Body.String(), `"request_id":"panic-request"`) ||
		!strings.Contains(response.Body.String(), `"code":"INTERNAL_ERROR"`) {
		t.Fatalf("panic envelope = %s", response.Body)
	}
}

func testHTTPHandler(config conf.Config, application dataapi.DataHTTPServer) http.Handler {
	if application == nil {
		return NewHTTPServer(config, nil, nil, nil).Server.Handler
	}
	authenticator, err := NewAuthenticator([]Credential{{
		Secret: "admin-token",
		Principal: dataapi.Principal{
			Identity: "admin-portal-bff",
			Scopes:   []string{ScopeAdminRead},
		},
	}})
	if err != nil {
		panic(err)
	}
	return NewHTTPServer(config, application, authenticator, nil).Server.Handler
}

type serverTestDataService struct{}

func serverTestResponse[T any]() (*dataapi.Response[T], error) {
	return &dataapi.Response[T]{Status: http.StatusNoContent}, nil
}
func (serverTestDataService) ImportReviewedEvents(context.Context, *dataapi.EventPublicationRequest) (*dataapi.Response[dataapi.EventPublicationResult], error) {
	return serverTestResponse[dataapi.EventPublicationResult]()
}
func (serverTestDataService) ListActiveEventTags(context.Context, *dataapi.EventTagCatalogRequest) (*dataapi.Response[dataapi.EventTagCatalog], error) {
	return serverTestResponse[dataapi.EventTagCatalog]()
}
func (serverTestDataService) ImportResearchThemes(context.Context, *dataapi.ResearchThemeImportRequest) (*dataapi.Response[dataapi.ResearchThemeImportResult], error) {
	return serverTestResponse[dataapi.ResearchThemeImportResult]()
}
func (serverTestDataService) ImportResearchReasoningTrees(context.Context, *dataapi.ResearchReasoningTreeImportRequest) (*dataapi.Response[dataapi.ResearchReasoningTreeImportResult], error) {
	return serverTestResponse[dataapi.ResearchReasoningTreeImportResult]()
}
func (serverTestDataService) ListResearchThemes(context.Context, *dataapi.ListResearchThemesRequest) (*dataapi.Response[dataapi.ResearchThemePage], error) {
	return serverTestResponse[dataapi.ResearchThemePage]()
}
func (serverTestDataService) GetResearchTheme(context.Context, *dataapi.GetResearchThemeRequest) (*dataapi.Response[dataapi.ResearchThemeDetail], error) {
	return serverTestResponse[dataapi.ResearchThemeDetail]()
}
func (serverTestDataService) ListResearchReasoningTrees(context.Context, *dataapi.ReasoningTreeListRequest) (*dataapi.Response[dataapi.ResearchReasoningTreeList], error) {
	return serverTestResponse[dataapi.ResearchReasoningTreeList]()
}
func (serverTestDataService) GetResearchReasoningTree(context.Context, *dataapi.ReasoningTreeDetailRequest) (*dataapi.Response[dataapi.ResearchReasoningTreeDetail], error) {
	return serverTestResponse[dataapi.ResearchReasoningTreeDetail]()
}
func (serverTestDataService) ListRawDocuments(context.Context, *dataapi.RawDocumentListRequest) (*dataapi.Response[dataapi.AdminRawDocumentPage], error) {
	return serverTestResponse[dataapi.AdminRawDocumentPage]()
}
func (serverTestDataService) ListEvents(context.Context, *dataapi.EventListRequest) (*dataapi.Response[dataapi.AdminEventPage], error) {
	return serverTestResponse[dataapi.AdminEventPage]()
}

type principalRecordingDataService struct {
	serverTestDataService
	identity string
}

func (s *principalRecordingDataService) ListEvents(ctx context.Context, _ *dataapi.EventListRequest) (*dataapi.Response[dataapi.AdminEventPage], error) {
	principal, ok := dataapi.PrincipalFromContext(ctx)
	if ok {
		s.identity = principal.Identity
	}
	return serverTestResponse[dataapi.AdminEventPage]()
}

type panickingDataService struct{ serverTestDataService }

func (panickingDataService) ListEvents(context.Context, *dataapi.EventListRequest) (*dataapi.Response[dataapi.AdminEventPage], error) {
	panic("sensitive panic detail")
}

func assertJSONKeysMatchProperties(t *testing.T, name string, body, properties map[string]any) {
	t.Helper()
	got := make([]string, 0, len(body))
	for key := range body {
		got = append(got, key)
	}
	want := make([]string, 0, len(properties))
	for key := range properties {
		want = append(want, key)
	}
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("%s JSON fields = %v, OpenAPI fields = %v", name, got, want)
	}
}

func assertServiceHealth(t *testing.T, handler http.Handler, service string) {
	t.Helper()
	for _, test := range []struct {
		path       string
		wantStatus string
	}{
		{path: "/healthz", wantStatus: "ok"},
		{path: "/readyz", wantStatus: "ready"},
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want %d", test.path, response.Code, http.StatusOK)
		}
		var body struct {
			Status      string            `json:"status"`
			Service     string            `json:"service"`
			Environment conf.Environment  `json:"environment"`
			Checks      map[string]string `json:"checks"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode %s: %v", test.path, err)
		}
		if body.Status != test.wantStatus || body.Service != service || body.Environment != conf.EnvLocal {
			t.Fatalf("%s response = %+v", test.path, body)
		}
		if test.path == "/readyz" && body.Checks["config"] != "ok" {
			t.Fatalf("%s checks = %v, want config=ok", test.path, body.Checks)
		}
	}
}

func testConfig() conf.Config {
	return conf.Config{
		App: conf.AppConfig{Env: conf.EnvLocal},
		Server: conf.ServerConfig{
			Host:                "127.0.0.1",
			Port:                18081,
			ReadTimeoutSeconds:  5,
			WriteTimeoutSeconds: 10,
		},
	}
}
