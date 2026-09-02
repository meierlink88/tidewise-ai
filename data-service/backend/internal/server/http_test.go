package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	dataapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	companyapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/company"
	evidenceapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/evidence"
	sourceapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/source"
	evidencebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/evidence"
	sourcebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/source"
	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
	evidencedata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/evidence"
	sourcedata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/source"
	evidenceservice "github.com/meierlink88/tidewise-ai/data-service/backend/internal/service/evidence"
	sourceservice "github.com/meierlink88/tidewise-ai/data-service/backend/internal/service/source"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/research"
)

func TestProductionServerSourceManagementAndSnapshotContract(t *testing.T) {
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	db := postgresfixture.OpenIsolated(t, "tw_source_server", migrationDir, 0)
	store, err := sourcedata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := sourcebiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	plainKey := "plain-bocha-key"
	fixed, err := useCase.PublishFixed(context.Background(), sourcebiz.CurrentFixedManifest(sourcebiz.FixedManifestOptions{AppKeys: map[string]string{"bocha": plainKey}}))
	if err != nil {
		t.Fatal(err)
	}
	application, err := sourceservice.NewService(useCase)
	if err != nil {
		t.Fatal(err)
	}
	authenticator, err := NewAuthenticator([]Credential{
		{Secret: "source-read-token", Principal: dataapi.Principal{Identity: "agentos", Scopes: []string{ScopeSourceRead}}},
		{Secret: "source-write-token", Principal: dataapi.Principal{Identity: "admin-backend", Scopes: []string{ScopeSourceWrite}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	server, err := NewHTTPServer(
		testConfig(), serverTestDataService{}, research.Service{}, serverTestEventService{}, serverTestEvidenceService{},
		serverTestCountryService{}, serverTestIndustryService{}, serverTestConceptService{}, serverTestChainNodeService{},
		serverTestIndustryChainService{}, serverTestOrganizationService{}, application, serverTestCompanyService{}, serverTestReportService{}, authenticator, nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	snapshot := productionContractRequest(t, server, http.MethodGet, dataapi.APIPrefix+"/source-snapshot", "source-read-token", "", "source-snapshot", http.StatusOK)
	active := snapshot["result"].(map[string]any)["sources"].([]any)
	if len(active) != 5 || active[0].(map[string]any)["channel_type"] != "api" || active[len(active)-1].(map[string]any)["channel_type"] != "web_search" {
		t.Fatalf("active snapshot = %#v", active)
	}
	if active[len(active)-1].(map[string]any)["app_key"] != plainKey {
		t.Fatalf("snapshot did not return plaintext app_key: %#v", active[len(active)-1])
	}
	productionContractError(t, server, http.MethodPost, dataapi.APIPrefix+"/sources", "source-write-token", `{"code":"missing_enabled","name":"Missing","endpoint":"https://example.com/feed.xml","app_key":null,"config":{},"priority":1,"timeout_seconds":30,"max_results":10,"default_source_level":"L3_MEDIA"}`, "source-create-required", http.StatusBadRequest, "INVALID_REQUEST")

	created := productionContractRequest(t, server, http.MethodPost, dataapi.APIPrefix+"/sources", "source-write-token", `{"code":"example_feed","name":"Example Feed","enabled":true,"endpoint":"https://example.com/feed.xml","app_key":null,"config":{"max_bytes":5000000},"priority":2,"timeout_seconds":20,"max_results":25,"default_source_level":"L3_MEDIA"}`, "source-create", http.StatusCreated)
	dynamicID := created["result"].(map[string]any)["id"].(string)
	if created["result"].(map[string]any)["ownership_type"] != "dynamic" || created["result"].(map[string]any)["adapter_key"] != "generic_rss" {
		t.Fatalf("dynamic create = %#v", created)
	}

	bochaID := ""
	for _, item := range fixed {
		if item.Code == "bocha" {
			bochaID = item.ID
		}
	}
	updated := productionContractRequest(t, server, http.MethodPut, dataapi.APIPrefix+"/sources/"+bochaID, "source-write-token", `{"name":"博查","adapter_key":"generic_rss","enabled":true,"endpoint":"https://api.bochaai.com/v1/web-search","app_key":"plain-bocha-key","config":{},"priority":1,"timeout_seconds":30,"max_results":10,"default_source_level":"L3_MEDIA"}`, "source-update", http.StatusOK)
	if updated["result"].(map[string]any)["channel_type"] != "web_search" || updated["result"].(map[string]any)["adapter_key"] != "generic_rss" {
		t.Fatalf("fixed adapter mismatch update = %#v", updated)
	}
	productionContractError(t, server, http.MethodDelete, dataapi.APIPrefix+"/sources/"+bochaID, "source-write-token", "", "source-fixed-delete", http.StatusConflict, "SOURCE_FIXED_DELETE_FORBIDDEN")
	productionContractRequest(t, server, http.MethodDelete, dataapi.APIPrefix+"/sources/"+dynamicID, "source-write-token", "", "source-dynamic-delete", http.StatusOK)
	productionContractError(t, server, http.MethodGet, dataapi.APIPrefix+"/source-snapshot?page=1", "source-read-token", "", "source-query", http.StatusBadRequest, "INVALID_REQUEST")
	productionContractError(t, server, http.MethodGet, dataapi.APIPrefix+"/source-snapshot", "source-write-token", "", "source-scope", http.StatusForbidden, "FORBIDDEN")
	if _, err := db.Exec(`UPDATE sources SET config='{"source_levels":{"example.com":"INVALID"}}'::jsonb WHERE code='cls_telegraph'`); err != nil {
		t.Fatal(err)
	}
	failedSnapshot := productionContractError(t, server, http.MethodGet, dataapi.APIPrefix+"/source-snapshot", "source-read-token", "", "source-invalid-state", http.StatusServiceUnavailable, "SOURCE_SNAPSHOT_FAILED")
	if _, exists := failedSnapshot["result"]; exists {
		t.Fatalf("failed snapshot exposed partial result: %#v", failedSnapshot)
	}
}

func TestSourceProviderFixtureMatchesRuntimeHTTPOutput(t *testing.T) {
	payload, err := os.ReadFile(filepath.Join("..", "..", "api", "data", "v1", "source", "testdata", "source-snapshot.v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		RequestID string                   `json:"request_id"`
		Result    sourceapi.SourceSnapshot `json:"result"`
	}
	var expected map[string]any
	if err := json.Unmarshal(payload, &fixture); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(payload, &expected); err != nil {
		t.Fatal(err)
	}
	server := sourceSnapshotContractServer(t, fixtureSourceService{sources: fixture.Result.Sources})
	actual := productionContractRequest(t, server, http.MethodGet, dataapi.APIPrefix+"/source-snapshot", "source-read-token", "", fixture.RequestID, http.StatusOK)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("runtime Source snapshot differs from provider fixture:\nactual=%#v\nexpected=%#v", actual, expected)
	}
}

func TestCompanyProjectionProviderFixtureMatchesRuntimeHTTPOutput(t *testing.T) {
	payload, err := os.ReadFile(filepath.Join("..", "..", "api", "data", "v1", "entity", "company", "testdata", "company-projection-page.v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		RequestID string                           `json:"request_id"`
		Result    companyapi.CompanyProjectionPage `json:"result"`
	}
	var expected map[string]any
	if err := json.Unmarshal(payload, &fixture); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(payload, &expected); err != nil {
		t.Fatal(err)
	}
	authenticator, err := NewAuthenticator([]Credential{{
		Secret: "company-read-token", Principal: dataapi.Principal{Identity: "agentos", Scopes: []string{ScopeCompanyRead}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	server, err := NewHTTPServer(
		testConfig(), serverTestDataService{}, research.Service{}, serverTestEventService{}, serverTestEvidenceService{},
		serverTestCountryService{}, serverTestIndustryService{}, serverTestConceptService{}, serverTestChainNodeService{},
		serverTestIndustryChainService{}, serverTestOrganizationService{}, serverTestSourceService{},
		fixtureCompanyService{page: fixture.Result}, serverTestReportService{}, authenticator, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	actual := productionContractRequest(t, server, http.MethodGet, dataapi.APIPrefix+"/entities/companies", "company-read-token", "", fixture.RequestID, http.StatusOK)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("runtime Company projection differs from provider fixture:\nactual=%#v\nexpected=%#v", actual, expected)
	}
}

type fixtureCompanyService struct {
	serverTestCompanyService
	page companyapi.CompanyProjectionPage
}

func (s fixtureCompanyService) List(context.Context, *companyapi.ListRequest) (*dataapi.Response[companyapi.CompanyProjectionPage], error) {
	return &dataapi.Response[companyapi.CompanyProjectionPage]{Status: dataapi.StatusOK, Result: s.page}, nil
}

func TestEvidencePublicationProviderFixtureMatchesRuntimeHTTPOutput(t *testing.T) {
	requestPayload, err := os.ReadFile(filepath.Join("..", "..", "api", "data", "v1", "evidence", "testdata", "evidence-publication.json"))
	if err != nil {
		t.Fatal(err)
	}
	responsePayload, err := os.ReadFile(filepath.Join("..", "..", "api", "data", "v1", "evidence", "testdata", "evidence-publication-response.v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		RequestID string                                `json:"request_id"`
		Result    evidenceapi.EvidencePublicationResult `json:"result"`
	}
	var expected map[string]any
	if err := json.Unmarshal(responsePayload, &fixture); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(responsePayload, &expected); err != nil {
		t.Fatal(err)
	}
	application := &fixtureEvidencePublicationService{result: fixture.Result}
	server := evidencePublicationContractServer(t, application)
	actual := productionContractRequest(
		t, server, http.MethodPost, dataapi.APIPrefix+"/evidence-publications", "evidence-write-token",
		string(requestPayload), fixture.RequestID, http.StatusCreated,
	)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("runtime Evidence Publication response differs from provider fixture:\nactual=%#v\nexpected=%#v", actual, expected)
	}
	if application.request == nil || len(application.request.Evidences) != 2 {
		t.Fatalf("runtime request binding = %#v", application.request)
	}
}

type fixtureEvidencePublicationService struct {
	serverTestEvidenceService
	request *evidenceapi.EvidencePublicationRequest
	result  evidenceapi.EvidencePublicationResult
}

func (s *fixtureEvidencePublicationService) PublishEvidence(_ context.Context, request *evidenceapi.EvidencePublicationRequest) (*dataapi.Response[evidenceapi.EvidencePublicationResult], error) {
	s.request = request
	return &dataapi.Response[evidenceapi.EvidencePublicationResult]{Status: http.StatusCreated, Result: s.result}, nil
}

func evidencePublicationContractServer(t *testing.T, application evidenceapi.Service) http.Handler {
	t.Helper()
	authenticator, err := NewAuthenticator([]Credential{{
		Secret: "evidence-write-token", Principal: dataapi.Principal{Identity: "agentos", Scopes: []string{ScopeEvidenceImport}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	server, err := NewHTTPServer(
		testConfig(), serverTestDataService{}, research.Service{}, serverTestEventService{}, application,
		serverTestCountryService{}, serverTestIndustryService{}, serverTestConceptService{}, serverTestChainNodeService{},
		serverTestIndustryChainService{}, serverTestOrganizationService{}, serverTestSourceService{}, serverTestCompanyService{}, serverTestReportService{}, authenticator, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return server
}

func TestSourceSnapshotHTTPReturnsTwoHundredSourcesWithinEnvelopeAndTimeBudgets(t *testing.T) {
	sources := make([]sourceapi.Source, 0, sourcebiz.MaxSources)
	for index := 0; index < sourcebiz.MaxSources; index++ {
		code := fmt.Sprintf("source-%03d", index)
		id, err := coreid.Derive(coreid.Source, "source", code)
		if err != nil {
			t.Fatal(err)
		}
		sources = append(sources, sourceapi.Source{
			ID: id, Code: code, Name: code, OwnershipType: "dynamic", ChannelType: "rss", AdapterKey: "generic_rss",
			Enabled: true, Endpoint: "https://example.com/feed.xml", Config: json.RawMessage(`{}`), Priority: 1,
			TimeoutSeconds: 30, MaxResults: 10, DefaultSourceLevel: "L3_MEDIA",
			CreatedAt: "2026-08-19T00:00:00Z", UpdatedAt: "2026-08-19T00:00:00Z",
		})
	}
	server := sourceSnapshotContractServer(t, fixtureSourceService{sources: sources})
	request := httptest.NewRequest(http.MethodGet, dataapi.APIPrefix+"/source-snapshot", nil)
	request.Header.Set("Authorization", "Bearer source-read-token")
	request.Header.Set("X-Request-ID", "source-200-http")
	response := httptest.NewRecorder()
	started := time.Now()
	server.ServeHTTP(response, request)
	if elapsed := time.Since(started); elapsed >= 3*time.Second {
		t.Fatalf("200-Source HTTP snapshot took %s", elapsed)
	}
	if response.Code != http.StatusOK {
		t.Fatalf("snapshot status=%d body=%s", response.Code, response.Body.String())
	}
	if response.Body.Len() > sourcebiz.MaxSnapshotEnvelopeSize {
		t.Fatalf("serialized snapshot envelope=%d bytes", response.Body.Len())
	}
	var envelope struct {
		Result sourceapi.SourceSnapshot `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Result.Sources) != sourcebiz.MaxSources {
		t.Fatalf("HTTP snapshot Source count=%d", len(envelope.Result.Sources))
	}
}

type fixtureSourceService struct {
	serverTestSourceService
	sources []sourceapi.Source
}

func (s fixtureSourceService) Snapshot(context.Context) (*dataapi.Response[sourceapi.SourceSnapshot], error) {
	return &dataapi.Response[sourceapi.SourceSnapshot]{Status: dataapi.StatusOK, Result: sourceapi.SourceSnapshot{Sources: s.sources}}, nil
}

func sourceSnapshotContractServer(t *testing.T, application sourceapi.Service) http.Handler {
	t.Helper()
	authenticator, err := NewAuthenticator([]Credential{{
		Secret: "source-read-token", Principal: dataapi.Principal{Identity: "agentos", Scopes: []string{ScopeSourceRead}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	server, err := NewHTTPServer(
		testConfig(), serverTestDataService{}, research.Service{}, serverTestEventService{}, serverTestEvidenceService{},
		serverTestCountryService{}, serverTestIndustryService{}, serverTestConceptService{}, serverTestChainNodeService{},
		serverTestIndustryChainService{}, serverTestOrganizationService{}, application, serverTestCompanyService{}, serverTestReportService{}, authenticator, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return server
}

func TestProductionServerRawEvidenceCategoriesUsePostgresAndPublicContract(t *testing.T) {
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	db := postgresfixture.OpenIsolated(t, "tw_raw_evidence_category_server", migrationDir, 0)
	assertEvidenceCategoryCatalog(t, db)
	store, err := evidencedata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := evidencebiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	application, err := evidenceservice.NewService(useCase)
	if err != nil {
		t.Fatal(err)
	}
	authenticator, err := NewAuthenticator([]Credential{
		{Secret: "raw-evidence-write-token", Principal: dataapi.Principal{Identity: "raw-evidence-writer", Scopes: []string{ScopeRawEvidenceImport}}},
		{Secret: "raw-evidence-read-token", Principal: dataapi.Principal{Identity: "raw-evidence-reader", Scopes: []string{ScopeRawEvidenceRead}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	server, err := NewHTTPServer(
		testConfig(), serverTestDataService{}, research.Service{}, serverTestEventService{},
		application,
		serverTestCountryService{}, serverTestIndustryService{}, serverTestConceptService{}, serverTestChainNodeService{}, serverTestIndustryChainService{}, serverTestOrganizationService{}, serverTestSourceService{}, serverTestCompanyService{}, serverTestReportService{}, authenticator, nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	publicationKey := "raw-evidence-category-contract"
	rawEvidenceID, err := coreid.Derive(coreid.RawEvidence, "raw-evidence-publication", publicationKey)
	if err != nil {
		t.Fatal(err)
	}
	payload := fmt.Sprintf(`{
		"raw_evidence": {
			"publication_key":%q,
			"source_id":"SRC_category_000000000000000000",
			"source_name":"Example Wire",
			"source_level":"L2_WIRE",
			"source_url":"https://example.test/category",
			"is_original":true,
			"raw_text":"Powell expects another rate increase this year.",
			"collected_at":"2026-08-14T01:05:00Z",
			"keywords":["美联储","加息"],
			"category_ids":["EVC083b086f-c9ee-504c-85e9-639fa8d39e8f","EVC097bf77a-fb8a-5756-ae47-e122c4367985"]
		}
	}`, publicationKey)
	created := productionEvidenceRequest(t, server, http.MethodPost, dataapi.APIPrefix+"/raw-evidence-publications", "raw-evidence-write-token", payload, http.StatusCreated)
	if created["result"].(map[string]any)["id"] != rawEvidenceID {
		t.Fatalf("created Raw Evidence envelope = %#v", created)
	}

	detail := productionEvidenceRequest(t, server, http.MethodGet, dataapi.APIPrefix+"/raw-evidences/"+rawEvidenceID, "raw-evidence-read-token", "", http.StatusOK)
	raw := detail["result"].(map[string]any)["raw_evidence"].(map[string]any)
	if raw["id"] != rawEvidenceID {
		t.Fatalf("Raw Evidence identity = %#v", raw["id"])
	}
	categories := raw["categories"].([]any)
	if len(categories) != 2 || categories[0].(map[string]any)["id"] != "EVC083b086f-c9ee-504c-85e9-639fa8d39e8f" || categories[1].(map[string]any)["id"] != "EVC097bf77a-fb8a-5756-ae47-e122c4367985" {
		t.Fatalf("Raw Evidence categories = %#v", categories)
	}
	if categories[0].(map[string]any)["code"] != "INTERVIEW_OR_STATEMENT" || categories[0].(map[string]any)["description"] == "" {
		t.Fatalf("resolved Evidence Category = %#v", categories[0])
	}

	reversed := bytes.Replace([]byte(payload), []byte(`"category_ids":["EVC083b086f-c9ee-504c-85e9-639fa8d39e8f","EVC097bf77a-fb8a-5756-ae47-e122c4367985"]`), []byte(`"category_ids":["EVC097bf77a-fb8a-5756-ae47-e122c4367985","EVC083b086f-c9ee-504c-85e9-639fa8d39e8f"]`), 1)
	productionEvidenceRequest(t, server, http.MethodPost, dataapi.APIPrefix+"/raw-evidence-publications", "raw-evidence-write-token", string(reversed), http.StatusCreated)
	var linkCount int
	if err := db.QueryRow(`SELECT count(*) FROM raw_evidence_category_links WHERE raw_evidence_id = $1`, rawEvidenceID).Scan(&linkCount); err != nil {
		t.Fatal(err)
	}
	if linkCount != 2 {
		t.Fatalf("Raw Evidence Category link count = %d, want 2", linkCount)
	}
	var earliestLink time.Time
	if err := db.QueryRow(`SELECT min(created_at) FROM raw_evidence_category_links WHERE raw_evidence_id = $1`, rawEvidenceID).Scan(&earliestLink); err != nil {
		t.Fatal(err)
	}
	if earliestLink.IsZero() {
		t.Fatal("Raw Evidence Category link has zero created_at")
	}

	drift := bytes.Replace([]byte(payload), []byte(`"category_ids":["EVC083b086f-c9ee-504c-85e9-639fa8d39e8f","EVC097bf77a-fb8a-5756-ae47-e122c4367985"]`), []byte(`"category_ids":["EVC083b086f-c9ee-504c-85e9-639fa8d39e8f"]`), 1)
	productionEvidenceError(t, server, http.MethodPost, dataapi.APIPrefix+"/raw-evidence-publications", "raw-evidence-write-token", string(drift), http.StatusConflict, "EVIDENCE_PUBLICATION_CONFLICT")
	unknownKey := "raw-evidence-unknown-category"
	unknown := bytes.Replace([]byte(payload), []byte(publicationKey), []byte(unknownKey), 1)
	unknown = bytes.Replace(unknown, []byte(`"EVC083b086f-c9ee-504c-85e9-639fa8d39e8f","EVC097bf77a-fb8a-5756-ae47-e122c4367985"`), []byte(`"EVCca2c5f7c-b52a-5c5b-ba14-b38252e4f738"`), 1)
	productionEvidenceError(t, server, http.MethodPost, dataapi.APIPrefix+"/raw-evidence-publications", "raw-evidence-write-token", string(unknown), http.StatusUnprocessableEntity, "EVIDENCE_PUBLICATION_REFERENCE_INVALID")
	duplicateKey := "raw-evidence-duplicate-category"
	duplicate := bytes.Replace([]byte(payload), []byte(publicationKey), []byte(duplicateKey), 1)
	duplicate = bytes.Replace(duplicate, []byte(`"EVC083b086f-c9ee-504c-85e9-639fa8d39e8f","EVC097bf77a-fb8a-5756-ae47-e122c4367985"`), []byte(`"EVC097bf77a-fb8a-5756-ae47-e122c4367985","EVC097bf77a-fb8a-5756-ae47-e122c4367985"`), 1)
	productionEvidenceError(t, server, http.MethodPost, dataapi.APIPrefix+"/raw-evidence-publications", "raw-evidence-write-token", string(duplicate), http.StatusBadRequest, "INVALID_REQUEST")
	malformedKey := "raw-evidence-malformed-category"
	malformed := bytes.Replace([]byte(payload), []byte(publicationKey), []byte(malformedKey), 1)
	malformed = bytes.Replace(malformed, []byte(`"EVC083b086f-c9ee-504c-85e9-639fa8d39e8f","EVC097bf77a-fb8a-5756-ae47-e122c4367985"`), []byte(`"BAD_ID"`), 1)
	malformedError := productionEvidenceError(t, server, http.MethodPost, dataapi.APIPrefix+"/raw-evidence-publications", "raw-evidence-write-token", string(malformed), http.StatusBadRequest, "INVALID_REQUEST")
	malformedIssues := malformedError["error"].(map[string]any)["details"].(map[string]any)["issues"].([]any)
	if malformedIssues[0].(map[string]any)["path"] != "raw_evidence.category_ids[0]" {
		t.Fatalf("malformed category issue = %#v", malformedIssues[0])
	}
	var failedWriteCount int
	failedIDs := make([]string, 0, 3)
	for _, key := range []string{unknownKey, duplicateKey, malformedKey} {
		id, deriveErr := coreid.Derive(coreid.RawEvidence, "raw-evidence-publication", key)
		if deriveErr != nil {
			t.Fatal(deriveErr)
		}
		failedIDs = append(failedIDs, id)
	}
	if err := db.QueryRow(`SELECT count(*) FROM raw_evidences WHERE id IN ($1, $2, $3)`, failedIDs[0], failedIDs[1], failedIDs[2]).Scan(&failedWriteCount); err != nil {
		t.Fatal(err)
	}
	if failedWriteCount != 0 {
		t.Fatalf("failed category publications left %d Raw Evidence rows", failedWriteCount)
	}

	productionEvidenceError(t, server, http.MethodGet, dataapi.APIPrefix+"/raw-evidences/RAW6dcc2dc9-8595-5e56-a521-a8967abd8bb9", "raw-evidence-read-token", "", http.StatusNotFound, "RAW_EVIDENCE_NOT_FOUND")
	productionEvidenceError(t, server, http.MethodGet, dataapi.APIPrefix+"/raw-evidences/"+string(bytes.Repeat([]byte("X"), 33)), "raw-evidence-read-token", "", http.StatusBadRequest, "INVALID_REQUEST")
	productionEvidenceError(t, server, http.MethodGet, dataapi.APIPrefix+"/raw-evidences/"+rawEvidenceID, "raw-evidence-write-token", "", http.StatusForbidden, "FORBIDDEN")
	productionEvidenceError(t, server, http.MethodGet, dataapi.APIPrefix+"/raw-evidences/"+rawEvidenceID, "", "", http.StatusUnauthorized, "UNAUTHENTICATED")

	uncategorizedKey := "raw-evidence-uncategorized"
	uncategorizedID, err := coreid.Derive(coreid.RawEvidence, "raw-evidence-publication", uncategorizedKey)
	if err != nil {
		t.Fatal(err)
	}
	legacy := bytes.Replace([]byte(payload), []byte(publicationKey), []byte(uncategorizedKey), 1)
	legacy = bytes.Replace(legacy, []byte(`,
			"category_ids":["EVC083b086f-c9ee-504c-85e9-639fa8d39e8f","EVC097bf77a-fb8a-5756-ae47-e122c4367985"]`), nil, 1)
	productionEvidenceRequest(t, server, http.MethodPost, dataapi.APIPrefix+"/raw-evidence-publications", "raw-evidence-write-token", string(legacy), http.StatusCreated)
	legacyDetail := productionEvidenceRequest(t, server, http.MethodGet, dataapi.APIPrefix+"/raw-evidences/"+uncategorizedID, "raw-evidence-read-token", "", http.StatusOK)
	legacyCategories := legacyDetail["result"].(map[string]any)["raw_evidence"].(map[string]any)["categories"].([]any)
	if len(legacyCategories) != 0 {
		t.Fatalf("uncategorized Raw Evidence categories = %#v", legacyCategories)
	}

	if _, err := db.Exec(`INSERT INTO raw_evidence_category_links (id, raw_evidence_id, category_id) VALUES ('RCL11111111-1111-4111-8111-111111111111', $1, 'EVC097bf77a-fb8a-5756-ae47-e122c4367985')`, rawEvidenceID); err == nil {
		t.Fatal("database accepted a duplicate Raw Evidence Category link")
	}
	if _, err := db.Exec(`INSERT INTO raw_evidence_category_links (id, raw_evidence_id, category_id) VALUES ('RCL22222222-2222-4222-8222-222222222222', 'RAW6dcc2dc9-8595-5e56-a521-a8967abd8bb9', 'EVCc18ddddb-14bc-5496-99ea-963ee2c25597')`); err == nil {
		t.Fatal("database accepted a Raw Evidence Category link with a missing Raw Evidence")
	}
	if _, err := db.Exec(`DELETE FROM evidence_categories WHERE id = 'EVC097bf77a-fb8a-5756-ae47-e122c4367985'`); err == nil {
		t.Fatal("database deleted a referenced Evidence Category")
	}
}

func assertEvidenceCategoryCatalog(t *testing.T, db interface {
	Query(string, ...any) (*sql.Rows, error)
}) {
	t.Helper()
	want := [][3]string{
		{"EVCc18ddddb-14bc-5496-99ea-963ee2c25597", "EVENT_BRIEF", "事件快讯"},
		{"EVC3a321d2c-1a58-5fde-bc69-fcdbb6d5180c", "FINANCIAL_REPORT_DATA_SUMMARY", "财报数据摘要"},
		{"EVC02a34cd9-b141-5a77-a628-c4458b77dd4e", "MARKET_MOVEMENT_BRIEF", "行情异动简讯"},
		{"EVC22332f49-2236-5cc4-aa2c-c53c29f09906", "MARKET_MOVEMENT_ANALYSIS", "市场异动分析"},
		{"EVC097bf77a-fb8a-5756-ae47-e122c4367985", "FORECAST_PLAN_OUTLOOK", "预测/计划/展望"},
		{"EVCed6e9380-8b20-53d5-b748-fa45c774fa67", "INDUSTRY_THEME_ANALYSIS", "行业/主题分析"},
		{"EVC5b12ffce-178d-56ed-a54f-c01696c486f4", "IN_DEPTH_REPORT", "专题/深度报道"},
		{"EVC5965b67d-1578-54a6-9b81-900a61a47a3f", "POLICY_DOCUMENT_SUMMARY", "政策文件摘要"},
		{"EVC083b086f-c9ee-504c-85e9-639fa8d39e8f", "INTERVIEW_OR_STATEMENT", "人物访谈/表态"},
		{"EVCb7505968-f65e-55a5-9294-b8a7be829a4a", "SOCIAL_MEDIA_BRIEF", "社交媒体快讯"},
		{"EVC7af2e623-fb5d-5145-91c8-dc308f2a1dce", "COMMENTARY_EDITORIAL_OPINION", "评论/社论/观点"},
	}
	sort.Slice(want, func(left, right int) bool { return want[left][0] < want[right][0] })
	rows, err := db.Query(`SELECT id, code, name, description, created_at FROM evidence_categories ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	index := 0
	for rows.Next() {
		var id, code, name, description string
		var createdAt time.Time
		if err := rows.Scan(&id, &code, &name, &description, &createdAt); err != nil {
			t.Fatal(err)
		}
		if index >= len(want) || [3]string{id, code, name} != want[index] || description == "" || createdAt.IsZero() {
			t.Fatalf("Evidence Category row %d = %q/%q/%q description=%q created_at=%s", index, id, code, name, description, createdAt)
		}
		index++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if index != len(want) {
		t.Fatalf("Evidence Category count = %d, want %d", index, len(want))
	}
}

func productionEvidenceRequest(t *testing.T, handler http.Handler, method, path, token, body string, wantStatus int) map[string]any {
	t.Helper()
	return productionContractRequest(t, handler, method, path, token, body, "raw-evidence-category-contract", wantStatus)
}

func productionEvidenceError(t *testing.T, handler http.Handler, method, path, token, body string, wantStatus int, wantCode string) map[string]any {
	t.Helper()
	return productionContractError(t, handler, method, path, token, body, "raw-evidence-category-contract", wantStatus, wantCode)
}

func productionContractRequest(t *testing.T, handler http.Handler, method, path, token, body, requestID string, wantStatus int) map[string]any {
	t.Helper()
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Request-ID", requestID)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != wantStatus {
		t.Fatalf("%s %s status=%d want=%d body=%s", method, path, response.Code, wantStatus, response.Body.String())
	}
	if response.Header().Get("X-Request-ID") != requestID {
		t.Fatalf("%s %s X-Request-ID=%q", method, path, response.Header().Get("X-Request-ID"))
	}
	var envelope map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode %s %s response: %v; body=%s", method, path, err, response.Body.String())
	}
	if envelope["request_id"] != requestID {
		t.Fatalf("%s %s envelope request_id=%#v", method, path, envelope["request_id"])
	}
	return envelope
}

func productionContractError(t *testing.T, handler http.Handler, method, path, token, body, requestID string, wantStatus int, wantCode string) map[string]any {
	t.Helper()
	envelope := productionContractRequest(t, handler, method, path, token, body, requestID, wantStatus)
	errorValue, ok := envelope["error"].(map[string]any)
	if !ok || errorValue["code"] != wantCode {
		t.Fatalf("%s %s error envelope=%#v, want code %s", method, path, envelope, wantCode)
	}
	return envelope
}
