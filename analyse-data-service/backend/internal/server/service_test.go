package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	dataapi "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	eventapi "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1/event"
	eventsemanticapi "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1/eventsemantic"
	evidenceapi "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1/evidence"
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
	application := &principalRecordingEventService{}
	handler := newTestHTTPServerWithEvent(testConfig(), serverTestDataService{}, application, serverTestEvidenceService{}, authenticator).Server.Handler

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

func TestNewHTTPServerRejectsMissingRequiredApplications(t *testing.T) {
	authenticator, err := NewAuthenticator([]Credential{{
		Secret:    "admin-token",
		Principal: dataapi.Principal{Identity: "admin", Scopes: []string{ScopeAdminRead}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name          string
		application   dataapi.DataHTTPServer
		event         eventapi.Service
		eventSemantic eventsemanticapi.Service
		evidence      evidenceapi.Service
		auth          *Authenticator
	}{
		{name: "Data API", event: serverTestEventService{}, eventSemantic: serverTestEventSemanticService{}, evidence: serverTestEvidenceService{}, auth: authenticator},
		{name: "Event API", application: serverTestDataService{}, eventSemantic: serverTestEventSemanticService{}, evidence: serverTestEvidenceService{}, auth: authenticator},
		{name: "Event Semantic API", application: serverTestDataService{}, event: serverTestEventService{}, evidence: serverTestEvidenceService{}, auth: authenticator},
		{name: "Evidence API", application: serverTestDataService{}, event: serverTestEventService{}, eventSemantic: serverTestEventSemanticService{}, auth: authenticator},
		{name: "authenticator", application: serverTestDataService{}, event: serverTestEventService{}, eventSemantic: serverTestEventSemanticService{}, evidence: serverTestEvidenceService{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewHTTPServer(testConfig(), test.application, test.event, test.eventSemantic, test.evidence, test.auth, nil); err == nil {
				t.Fatal("NewHTTPServer() error = nil")
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
	businessOperations := append(dataapi.BusinessOperations(), eventapi.BusinessOperations()...)
	businessOperations = append(businessOperations, eventsemanticapi.BusinessOperations()...)
	businessOperations = append(businessOperations, evidenceapi.BusinessOperations()...)
	for _, operation := range businessOperations {
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
	handler := newTestHTTPServerWithEvent(testConfig(), serverTestDataService{}, panickingEventService{}, serverTestEvidenceService{}, authenticator).Server.Handler
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
		application = serverTestDataService{}
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
	return newTestHTTPServer(config, application, serverTestEvidenceService{}, authenticator).Server.Handler
}

func newTestHTTPServer(config conf.Config, application dataapi.DataHTTPServer, evidenceApplication evidenceapi.Service, authenticator *Authenticator) *kratoshttp.Server {
	return newTestHTTPServerWithEvent(config, application, serverTestEventService{}, evidenceApplication, authenticator)
}

func newTestHTTPServerWithEvent(config conf.Config, application dataapi.DataHTTPServer, eventApplication eventapi.Service, evidenceApplication evidenceapi.Service, authenticator *Authenticator) *kratoshttp.Server {
	server, err := NewHTTPServer(config, application, eventApplication, serverTestEventSemanticService{}, evidenceApplication, authenticator, nil)
	if err != nil {
		panic(err)
	}
	return server
}

type serverTestDataService struct{}

type serverTestEventService struct{}

type serverTestEventSemanticService struct{}

type serverTestEvidenceService struct{}

func (serverTestEventService) PublishReviewedEvents(context.Context, *eventapi.PublicationRequest) (*dataapi.Response[eventapi.PublicationResult], error) {
	return serverTestResponse[eventapi.PublicationResult]()
}

func (serverTestEventService) ListActiveEventTags(context.Context, *eventapi.TagCatalogRequest) (*dataapi.Response[eventapi.TagCatalog], error) {
	return serverTestResponse[eventapi.TagCatalog]()
}

func (serverTestEventService) ListEvents(context.Context, *eventapi.ListRequest) (*dataapi.Response[eventapi.Page], error) {
	return serverTestResponse[eventapi.Page]()
}

func (serverTestEventSemanticService) ListEligibleEventSemanticEvents(context.Context, *eventsemanticapi.EligibleEventSemanticEventsRequest) (*dataapi.Response[eventsemanticapi.EligibleEventSemanticEvents], error) {
	return serverTestResponse[eventsemanticapi.EligibleEventSemanticEvents]()
}
func (serverTestEventSemanticService) CreateEventSemanticContextLease(context.Context, *eventsemanticapi.EventSemanticContextLeaseRequest) (*dataapi.Response[eventsemanticapi.EventSemanticContextLease], error) {
	return serverTestResponse[eventsemanticapi.EventSemanticContextLease]()
}
func (serverTestEventSemanticService) GetEventSemanticContext(context.Context, *eventsemanticapi.EventSemanticContextRequest) (*dataapi.Response[eventsemanticapi.EventSemanticContext], error) {
	return serverTestResponse[eventsemanticapi.EventSemanticContext]()
}
func (serverTestEventSemanticService) CreateEventSemanticSubmission(context.Context, *eventsemanticapi.EventSemanticSubmissionRequest) (*dataapi.Response[eventsemanticapi.EventSemanticSubmissionResult], error) {
	return serverTestResponse[eventsemanticapi.EventSemanticSubmissionResult]()
}
func (serverTestEventSemanticService) SubmitEventSemanticReview(context.Context, *eventsemanticapi.EventSemanticReviewRequest) (*dataapi.Response[eventsemanticapi.EventSemanticSubmissionResult], error) {
	return serverTestResponse[eventsemanticapi.EventSemanticSubmissionResult]()
}
func (serverTestEventSemanticService) GetEventSemantics(context.Context, *eventsemanticapi.GetEventSemanticsRequest) (*dataapi.Response[eventsemanticapi.EventSemanticsResult], error) {
	return serverTestResponse[eventsemanticapi.EventSemanticsResult]()
}

func (serverTestEvidenceService) PublishRawEvidence(context.Context, *evidenceapi.RawEvidencePublicationRequest) (*dataapi.Response[evidenceapi.RawEvidencePublicationResult], error) {
	return serverTestResponse[evidenceapi.RawEvidencePublicationResult]()
}

func (serverTestEvidenceService) PublishEvidence(context.Context, *evidenceapi.EvidencePublicationRequest) (*dataapi.Response[evidenceapi.EvidencePublicationResult], error) {
	return serverTestResponse[evidenceapi.EvidencePublicationResult]()
}

func serverTestResponse[T any]() (*dataapi.Response[T], error) {
	return &dataapi.Response[T]{Status: http.StatusNoContent}, nil
}
func (serverTestDataService) PublishResearchTheme(context.Context, *dataapi.ResearchThemeImportRequest) (*dataapi.Response[dataapi.ResearchThemeImportResult], error) {
	return serverTestResponse[dataapi.ResearchThemeImportResult]()
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
func (serverTestDataService) ListResearchAnalysisContext(context.Context, *dataapi.ResearchAnalysisContextRequest) (*dataapi.Response[dataapi.ResearchAnalysisContext], error) {
	return serverTestResponse[dataapi.ResearchAnalysisContext]()
}
func (serverTestDataService) SearchResearchGraph(context.Context, *dataapi.ResearchGraphSearchRequest) (*dataapi.Response[dataapi.ResearchGraphSearchResult], error) {
	return serverTestResponse[dataapi.ResearchGraphSearchResult]()
}
func (serverTestDataService) GetRuntimeHealth(context.Context, *dataapi.RuntimeHealthRequest) (*dataapi.Response[dataapi.RuntimeHealth], error) {
	return serverTestResponse[dataapi.RuntimeHealth]()
}

type principalRecordingEventService struct {
	serverTestEventService
	identity string
}

func (s *principalRecordingEventService) ListEvents(ctx context.Context, _ *eventapi.ListRequest) (*dataapi.Response[eventapi.Page], error) {
	principal, ok := dataapi.PrincipalFromContext(ctx)
	if ok {
		s.identity = principal.Identity
	}
	return serverTestResponse[eventapi.Page]()
}

type panickingEventService struct{ serverTestEventService }

func (panickingEventService) ListEvents(context.Context, *eventapi.ListRequest) (*dataapi.Response[eventapi.Page], error) {
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
