package internalapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	eventapp "github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/eventimport"
	"github.com/meierlink88/tidewise-ai/backend/internal/apps/miniappapi"
	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	domainimport "github.com/meierlink88/tidewise-ai/backend/internal/domain/eventimport"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
	"github.com/meierlink88/tidewise-ai/backend/services/data/rawimport"
)

func TestRawImportDerivesCallerFromAuthenticatedPrincipalAndReturnsEnvelopes(t *testing.T) {
	raw := &fakeRawImporter{result: rawimport.Result{
		ReceiptID:      "11111111-1111-5111-8111-111111111111",
		PayloadHash:    strings.Repeat("a", 64),
		RawDocumentIDs: []string{"22222222-2222-5222-8222-222222222222"},
		Items:          []rawimport.ItemResult{{RawDocumentID: "22222222-2222-5222-8222-222222222222", Disposition: rawimport.DispositionCreated}},
		ImportedAt:     time.Date(2026, 7, 17, 3, 0, 0, 0, time.UTC),
	}}
	handler := testHandler(t, Dependencies{RawImports: raw})
	body := `{"idempotency_key":"batch-1","items":[{"source_id":"22222222-2222-5222-8222-222222222222","ingest_channel":"rss","source_type":"news","source_name":"Example","source_url":"https://example.test/feed","source_external_id":"story-1","title":"Title","content_text":"Body","content_level":"full","raw_object_uri":"","raw_mime_type":"","language":"en","published_at":"2026-07-17T00:00:00Z","collected_at":"2026-07-17T01:00:00Z","content_hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}`
	request := httptest.NewRequest(http.MethodPost, Namespace+"/raw-document-imports", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer cred-agent")
	request.Header.Set("X-Request-ID", "request-123")
	request.Header.Set("X-Caller-Identity", "spoofed")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	if raw.caller != "agent-run" || raw.key != "batch-1" {
		t.Fatalf("raw caller/key = %q/%q", raw.caller, raw.key)
	}
	assertEnvelopeRequestID(t, response, "request-123")

	raw.result.Replayed = true
	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, Namespace+"/raw-document-imports", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer cred-agent")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("replay status = %d, body=%s", response.Code, response.Body.String())
	}
	assertNonemptyGeneratedRequestID(t, response)

	raw.status = rawimport.ImportStatus{State: rawimport.StatusCompleted, Result: &raw.result}
	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, Namespace+"/raw-document-imports/batch-1", nil)
	request.Header.Set("Authorization", "Bearer cred-agent")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"status":"completed"`) {
		t.Fatalf("status lookup = %d %s", response.Code, response.Body.String())
	}
	if raw.statusCaller != "agent-run" || raw.statusKey != "batch-1" {
		t.Fatalf("status caller/key = %q/%q", raw.statusCaller, raw.statusKey)
	}
	statusCalls := raw.statusCalls
	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, Namespace+"/raw-document-imports/"+strings.Repeat("k", 201), nil)
	request.Header.Set("Authorization", "Bearer cred-agent")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || raw.statusCalls != statusCalls {
		t.Fatalf("oversized status key response=%d calls=%d body=%s", response.Code, raw.statusCalls, response.Body.String())
	}
}

func TestAuthenticationScopeBodyLimitAndStructuredConflictFailClosed(t *testing.T) {
	raw := &fakeRawImporter{err: &rawimport.ContractError{Code: rawimport.CodeIdempotencyConflict, Message: "conflict"}}
	handler := testHandler(t, Dependencies{RawImports: raw})

	for _, test := range []struct {
		name       string
		credential string
		want       int
	}{
		{name: "missing", want: http.StatusUnauthorized},
		{name: "wrong scope", credential: "cred-miniapp", want: http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, Namespace+"/raw-document-imports", strings.NewReader(`{}`))
			if test.credential != "" {
				request.Header.Set("Authorization", "Bearer "+test.credential)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.want || !strings.Contains(response.Body.String(), `"request_id"`) || !strings.Contains(response.Body.String(), `"error"`) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
		})
	}

	oversized := `{"idempotency_key":"x","items":[],"padding":"` + strings.Repeat("x", MaxRequestBodyBytes+1) + `"}`
	request := httptest.NewRequest(http.MethodPost, Namespace+"/raw-document-imports", strings.NewReader(oversized))
	request.Header.Set("Authorization", "Bearer cred-agent")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge || raw.calls != 0 {
		t.Fatalf("oversized response=%d calls=%d body=%s", response.Code, raw.calls, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, Namespace+"/raw-document-imports", strings.NewReader(`{"idempotency_key":"key","items":[]}`))
	request.Header.Set("Authorization", "Bearer cred-agent")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), rawimport.CodeIdempotencyConflict) {
		t.Fatalf("conflict response=%d %s", response.Code, response.Body.String())
	}
}

func TestReviewedEventImportUsesSeparateReviewedService(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "event-import", "reviewed-outbox-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	reviewed := &fakeReviewedImporter{result: eventapp.Result{
		PackageID: "pkg", ReceiptID: "11111111-1111-5111-8111-111111111111",
		EventID:        "22222222-2222-5222-8222-222222222222",
		RawDocumentIDs: []string{"33333333-3333-5333-8333-333333333333"},
		EventSourceIDs: []string{"44444444-4444-5444-8444-444444444444"},
		EventTagMapIDs: []string{"55555555-5555-5555-8555-555555555555"},
		PayloadHash:    strings.Repeat("b", 64),
	}}
	handler := testHandler(t, Dependencies{ReviewedEvents: reviewed})
	request := httptest.NewRequest(http.MethodPost, Namespace+"/reviewed-event-imports", bytes.NewReader(fixture))
	request.Header.Set("Authorization", "Bearer cred-agent")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || reviewed.calls != 1 || reviewed.pkg.Review.ReviewID == "" {
		t.Fatalf("reviewed response=%d calls=%d body=%s", response.Code, reviewed.calls, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "raw_document_import_receipts") {
		t.Fatal("reviewed event response leaked the raw receipt contract")
	}

	reviewed.err = eventapp.ErrIdempotencyConflict
	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, Namespace+"/reviewed-event-imports", bytes.NewReader(fixture))
	request.Header.Set("Authorization", "Bearer cred-agent")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("reviewed conflict=%d %s", response.Code, response.Body.String())
	}
}

func TestImportHandlersDoNotLeakUnexpectedStoreErrors(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "event-import", "reviewed-outbox-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	internalText := "pq: connection failed for password=must-not-leak"
	reviewed := &fakeReviewedImporter{err: errors.New(internalText)}
	raw := &fakeRawImporter{err: errors.New(internalText)}
	handler := testHandler(t, Dependencies{ReviewedEvents: reviewed, RawImports: raw})

	request := httptest.NewRequest(http.MethodPost, Namespace+"/reviewed-event-imports", bytes.NewReader(fixture))
	request.Header.Set("Authorization", "Bearer cred-agent")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError || strings.Contains(response.Body.String(), internalText) || strings.Contains(response.Body.String(), "password") {
		t.Fatalf("reviewed store failure response=%d body=%s", response.Code, response.Body.String())
	}

	rawBody := `{"idempotency_key":"batch-1","items":[]}`
	request = httptest.NewRequest(http.MethodPost, Namespace+"/raw-document-imports", strings.NewReader(rawBody))
	request.Header.Set("Authorization", "Bearer cred-agent")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError || strings.Contains(response.Body.String(), internalText) || strings.Contains(response.Body.String(), "password") {
		t.Fatalf("raw store failure response=%d body=%s", response.Code, response.Body.String())
	}
}

func TestReviewedEventKnownValidationFailsSafelyBeforeServiceCall(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "event-import", "reviewed-outbox-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixture = bytes.Replace(fixture, []byte(`"decision": "auto_approved"`), []byte(`"decision": "unknown"`), 1)
	reviewed := &fakeReviewedImporter{}
	handler := testHandler(t, Dependencies{ReviewedEvents: reviewed})
	request := httptest.NewRequest(http.MethodPost, Namespace+"/reviewed-event-imports", bytes.NewReader(fixture))
	request.Header.Set("Authorization", "Bearer cred-agent")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnprocessableEntity || reviewed.calls != 0 || !strings.Contains(response.Body.String(), "REVIEWED_EVENT_IMPORT_REJECTED") {
		t.Fatalf("known validation response=%d calls=%d body=%s", response.Code, reviewed.calls, response.Body.String())
	}
}

func TestResearchAdminAndAgentMetadataAreSingleAggregateCallsAndMetadataIsRedacted(t *testing.T) {
	research := &fakeResearchService{
		themes: miniappapi.ResearchThemeListResponse{WindowStart: "2026-07-16T00:00:00Z", WindowEnd: "2026-07-17T00:00:00Z", AsOf: "2026-07-17T00:00:00Z", Items: []miniappapi.ResearchThemeItem{}},
	}
	admin := &fakeAdminStore{
		rawPage:   repositories.RawDocumentPage{Page: 1, PageSize: 20, Items: []domain.RawDocument{{ID: "11111111-1111-5111-8111-111111111111", SourceID: "22222222-2222-5222-8222-222222222222", Title: "Raw", CollectedAt: time.Date(2026, 7, 17, 1, 0, 0, 0, time.UTC), IngestStatus: domain.IngestStatusCollected}}},
		eventPage: repositories.EventPage{Page: 1, PageSize: 20, Items: []domain.Event{}},
		sources: []domain.SourceCatalog{{
			ID: "22222222-2222-5222-8222-222222222222", IngestChannel: "rss", ProviderKey: "provider", ConnectorKey: "connector", ParserKey: "parser", SourceType: "news", SourceName: "Example", SourceURL: "https://example.test/feed", AuthType: "api_key", CredentialRef: "PROVIDER_CREDENTIAL", UsagePolicy: "research-only", Status: domain.SourceCatalogStatusActive,
			SourceConfig: map[string]any{"language": "zh", "secret": "must-not-leak", "api_key": "must-not-leak"}, RateLimitPolicy: map[string]any{"requests_per_minute": 10, "token": "must-not-leak"},
		}},
	}
	handler := testHandler(t, Dependencies{Research: research, Admin: admin, SourceMetadata: admin})

	request := httptest.NewRequest(http.MethodGet, Namespace+"/research/themes?window_hours=24&limit=20", nil)
	request.Header.Set("Authorization", "Bearer cred-miniapp")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || research.themeCalls != 1 {
		t.Fatalf("research response=%d calls=%d body=%s", response.Code, research.themeCalls, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, Namespace+"/admin/raw-documents?source_id=22222222-2222-5222-8222-222222222222&ingest_status=collected&page=1&page_size=20", nil)
	request.Header.Set("Authorization", "Bearer cred-admin")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || admin.rawCalls != 1 || admin.rawFilter.SourceID == "" || admin.rawFilter.IngestStatus != domain.IngestStatusCollected {
		t.Fatalf("admin raw response=%d calls=%d filter=%#v body=%s", response.Code, admin.rawCalls, admin.rawFilter, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, Namespace+"/agent-run/source-metadata?limit=20&status=active", nil)
	request.Header.Set("Authorization", "Bearer cred-agent")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || admin.sourceCalls != 1 {
		t.Fatalf("metadata response=%d calls=%d body=%s", response.Code, admin.sourceCalls, response.Body.String())
	}
	body := response.Body.String()
	for _, forbidden := range []string{"must-not-leak", `"secret"`, `"credential_value"`} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("metadata contains %q: %s", forbidden, body)
		}
	}
	for _, required := range []string{"PROVIDER_CREDENTIAL", `"language":"zh"`, `"parser_key":"parser"`} {
		if !strings.Contains(body, required) {
			t.Fatalf("metadata missing %q: %s", required, body)
		}
	}

	request = httptest.NewRequest(http.MethodGet, Namespace+"/agent-run/source-metadata?limit=20&cursor=22222222-2222-5222-8222-222222222222", nil)
	request.Header.Set("Authorization", "Bearer cred-agent")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"items":[]`) {
		t.Fatalf("last cursor response=%d body=%s", response.Code, response.Body.String())
	}
}

func testHandler(t *testing.T, dependencies Dependencies) http.Handler {
	t.Helper()
	authenticator, err := NewAuthenticator([]Credential{
		{Secret: "cred-agent", Principal: Principal{Identity: "agent-run", Scopes: []string{ScopeRawImport, ScopeReviewedEventImport, ScopeSourceMetadataRead}}},
		{Secret: "cred-miniapp", Principal: Principal{Identity: "miniapp-service", Scopes: []string{ScopeResearchRead}}},
		{Secret: "cred-admin", Principal: Principal{Identity: "adminportal-service", Scopes: []string{ScopeAdminRead}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	dependencies.Authenticator = authenticator
	dependencies.NewRequestID = func() string { return "generated-request" }
	return NewHandler(dependencies)
}

func assertEnvelopeRequestID(t *testing.T, response *httptest.ResponseRecorder, want string) {
	t.Helper()
	if response.Header().Get("X-Request-ID") != want || !strings.Contains(response.Body.String(), `"request_id":"`+want+`"`) {
		t.Fatalf("request id response: header=%q body=%s", response.Header().Get("X-Request-ID"), response.Body.String())
	}
}

func assertNonemptyGeneratedRequestID(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	assertEnvelopeRequestID(t, response, "generated-request")
}

type fakeRawImporter struct {
	result                  rawimport.Result
	status                  rawimport.ImportStatus
	err                     error
	calls                   int
	caller, key             string
	statusCaller, statusKey string
	statusCalls             int
}

func (f *fakeRawImporter) Import(_ context.Context, caller, key string, _ rawimport.Batch) (rawimport.Result, error) {
	f.calls++
	f.caller, f.key = caller, key
	return f.result, f.err
}

func (f *fakeRawImporter) Status(_ context.Context, caller, key string) (rawimport.ImportStatus, error) {
	f.statusCalls++
	f.statusCaller, f.statusKey = caller, key
	return f.status, f.err
}

type fakeReviewedImporter struct {
	result eventapp.Result
	pkg    domainimport.Package
	err    error
	calls  int
}

func (f *fakeReviewedImporter) Import(_ context.Context, pkg domainimport.Package) (eventapp.Result, error) {
	f.calls++
	f.pkg = pkg
	return f.result, f.err
}

type fakeResearchService struct {
	themes     miniappapi.ResearchThemeListResponse
	themeCalls int
}

func (f *fakeResearchService) ListThemes(context.Context, miniappapi.ResearchListRequest) (miniappapi.ResearchThemeListResponse, error) {
	f.themeCalls++
	return f.themes, nil
}
func (f *fakeResearchService) GetTheme(context.Context, string, miniappapi.ResearchDetailRequest) (miniappapi.ResearchThemeDetailResponse, error) {
	return miniappapi.ResearchThemeDetailResponse{}, errors.New("not configured")
}
func (f *fakeResearchService) ListAnchors(context.Context, miniappapi.ResearchListRequest) (miniappapi.ResearchAnchorListResponse, error) {
	return miniappapi.ResearchAnchorListResponse{}, errors.New("not configured")
}
func (f *fakeResearchService) GetAnchor(context.Context, string, miniappapi.ResearchDetailRequest) (miniappapi.ResearchAnchorDetailResponse, error) {
	return miniappapi.ResearchAnchorDetailResponse{}, errors.New("not configured")
}

type fakeAdminStore struct {
	rawPage     repositories.RawDocumentPage
	eventPage   repositories.EventPage
	sources     []domain.SourceCatalog
	rawFilter   repositories.RawDocumentListFilter
	rawCalls    int
	eventCalls  int
	sourceCalls int
}

func (f *fakeAdminStore) ListRawDocuments(_ context.Context, filter repositories.RawDocumentListFilter) (repositories.RawDocumentPage, error) {
	f.rawCalls++
	f.rawFilter = filter
	return f.rawPage, nil
}
func (f *fakeAdminStore) ListEvents(context.Context, repositories.EventListFilter) (repositories.EventPage, error) {
	f.eventCalls++
	return f.eventPage, nil
}
func (f *fakeAdminStore) ListSourceCatalogs(context.Context, repositories.SourceCatalogListFilter) ([]domain.SourceCatalog, error) {
	f.sourceCalls++
	return f.sources, nil
}

func decodeJSON(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	return value
}
