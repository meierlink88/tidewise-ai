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
	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	domainimport "github.com/meierlink88/tidewise-ai/backend/internal/domain/eventimport"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
	"github.com/meierlink88/tidewise-ai/backend/services/data/rawimport"
	"github.com/meierlink88/tidewise-ai/backend/services/data/research"
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
		themes: research.ResearchThemePage{WindowStart: time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC), WindowEnd: time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC), AsOf: time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC), Items: []research.ResearchTheme{}},
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

func TestResearchHandlerNonEmptyGoldenPreservesAuthoritativeContract(t *testing.T) {
	now := time.Date(2026, 7, 17, 1, 2, 3, 0, time.UTC)
	nextCursor := "opaque-next"
	themeImpacts := []domain.ImpactLevel{
		domain.ImpactLevelHigh, domain.ImpactLevelFocus, domain.ImpactLevelWatch,
		domain.ImpactLevelHigh, domain.ImpactLevelFocus,
	}
	themeStages := []domain.TransmissionStage{
		domain.TransmissionStageUpstream, domain.TransmissionStageMidstream,
		domain.TransmissionStageDownstream, domain.TransmissionStageInfrastructure,
		domain.TransmissionStageService,
	}
	themes := make([]research.ResearchTheme, 0, len(themeStages))
	for index, stage := range themeStages {
		themes = append(themes, research.ResearchTheme{
			ID: "11111111-1111-4111-8111-111111111111", Name: "主题", OneLineConclusion: "结论",
			ImpactLevel: themeImpacts[index], TransmissionPath: "供给到价格", TradingDirection: "关注供需验证后的方向变化",
			TransmissionStage: stage, NextCheckpoint: "下次数据", IndexImpactSummary: "指数摘要", PublishedAt: now,
			AffectedChainNodes: []research.ResearchThemeChainNode{}, RelatedIndices: []research.ResearchIndex{}, HasMoreDetail: true,
		})
	}
	themes[0].AffectedChainNodes = []research.ResearchThemeChainNode{{
		ID: "22222222-2222-4222-8222-222222222222", Name: "主题节点", RelationRole: "driver", ImpactSummary: "主题影响摘要",
	}}
	themes[0].RelatedIndices = []research.ResearchIndex{
		{ID: "33333333-3333-4333-8333-333333333333", Name: "正向", ImpactDirection: domain.ResearchImpactPositive, ImpactSummary: "正向摘要"},
		{ID: "44444444-4444-4444-8444-444444444444", Name: "负向", ImpactDirection: domain.ResearchImpactNegative, ImpactSummary: "负向摘要"},
		{ID: "55555555-5555-4555-8555-555555555555", Name: "混合", ImpactDirection: domain.ResearchImpactMixed, ImpactSummary: "混合摘要"},
		{ID: "66666666-6666-4666-8666-666666666666", Name: "中性", ImpactDirection: domain.ResearchImpactNeutral, ImpactSummary: "中性摘要"},
	}

	anchorTypes := []domain.AnchorType{
		domain.AnchorTypePolicy, domain.AnchorTypeSupply, domain.AnchorTypeDemand,
		domain.AnchorTypeTechnology, domain.AnchorTypeCost, domain.AnchorTypeGeopolitics,
		domain.AnchorTypeMarketStructure,
	}
	importance := []domain.ResearchImportance{
		domain.ResearchImportancePrimary, domain.ResearchImportanceSecondary, domain.ResearchImportanceContextual,
	}
	anchors := make([]research.ResearchAnchor, 0, len(anchorTypes))
	for index, anchorType := range anchorTypes {
		anchors = append(anchors, research.ResearchAnchor{
			ID: "77777777-7777-4777-8777-777777777777", AnchorType: anchorType, Name: "锚点", OneLineConclusion: "锚点结论",
			Importance: importance[index%len(importance)], TransmissionPath: "政策到预期", TradingDirection: "观察政策兑现后的市场反馈",
			PublishedAt: now, RelatedChainNodes: []research.ResearchAnchorChainNode{}, RelatedIndices: []research.ResearchIndex{},
		})
	}
	anchors[0].RelatedChainNodes = []research.ResearchAnchorChainNode{{
		ID: "88888888-8888-4888-8888-888888888888", Name: "锚点节点", RelationRole: "constraint", RelationSummary: "锚点关系摘要",
	}}

	events := []research.ResearchEvent{
		{EventID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", Title: "驱动", Summary: "摘要", EventTime: &now, EvidenceRole: domain.ResearchEvidenceDriver, SupportedClaim: "主张"},
		{EventID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", Title: "支持", Summary: "摘要", EvidenceRole: domain.ResearchEvidenceSupporting, SupportedClaim: "主张"},
		{EventID: "cccccccc-cccc-4ccc-8ccc-cccccccccccc", Title: "反证", Summary: "摘要", EvidenceRole: domain.ResearchEvidenceContradicting, SupportedClaim: "主张"},
		{EventID: "dddddddd-dddd-4ddd-8ddd-dddddddddddd", Title: "背景", Summary: "摘要", EvidenceRole: domain.ResearchEvidenceContext, SupportedClaim: "主张"},
	}
	fake := &fakeResearchService{
		themes: research.ResearchThemePage{
			WindowStart: now.Add(-24 * time.Hour), WindowEnd: now, AsOf: now,
			ThemeCount: len(themes), EventCount: len(events), Items: themes, NextCursor: &nextCursor,
		},
		theme: research.ResearchThemeDetail{Theme: themes[0], Events: events},
		anchors: research.ResearchAnchorPage{
			WindowStart: now.Add(-24 * time.Hour), WindowEnd: now, AsOf: now,
			AnchorCount: len(anchors), EventCount: len(events), Items: anchors, NextCursor: &nextCursor,
		},
		anchor: research.ResearchAnchorDetail{Anchor: anchors[0], Events: events},
	}
	handler := testHandler(t, Dependencies{Research: fake})

	request := httptest.NewRequest(http.MethodGet, Namespace+"/research/themes?window_hours=24&limit=20&cursor=incoming", nil)
	request.Header.Set("Authorization", "Bearer cred-miniapp")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || fake.themeCalls != 1 {
		t.Fatalf("theme list response=%d calls=%d body=%s", response.Code, fake.themeCalls, response.Body.String())
	}
	if fake.lastThemeRequest.Cursor != "incoming" || fake.lastThemeRequest.WindowHours != 24 || fake.lastThemeRequest.Limit != 20 {
		t.Fatalf("theme request = %#v", fake.lastThemeRequest)
	}
	var themeEnvelope struct {
		RequestID string                     `json:"request_id"`
		Result    research.ResearchThemePage `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &themeEnvelope); err != nil {
		t.Fatal(err)
	}
	assertStringSet(t, "impact_level", themeImpactStrings(themeEnvelope.Result.Items), []string{"high", "focus", "watch"})
	assertStringSet(t, "transmission_stage", themeStageStrings(themeEnvelope.Result.Items), []string{"upstream", "midstream", "downstream", "infrastructure", "service"})
	assertStringSet(t, "impact_direction", indexDirectionStrings(themeEnvelope.Result.Items[0].RelatedIndices), []string{"positive", "negative", "mixed", "neutral"})
	if themeEnvelope.Result.Items[0].TradingDirection == "" || themeEnvelope.Result.NextCursor == nil || *themeEnvelope.Result.NextCursor != nextCursor {
		t.Fatalf("theme natural-language/cursor contract drifted: %#v", themeEnvelope.Result.Items[0])
	}
	themeNodeJSON, err := json.Marshal(themeEnvelope.Result.Items[0].AffectedChainNodes[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(themeNodeJSON), `"impact_summary":"主题影响摘要"`) || strings.Contains(string(themeNodeJSON), "relation_summary") {
		t.Fatalf("theme node JSON = %s", themeNodeJSON)
	}

	request = httptest.NewRequest(http.MethodGet, Namespace+"/research/themes/11111111-1111-4111-8111-111111111111?window_hours=24", nil)
	request.Header.Set("Authorization", "Bearer cred-miniapp")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	var themeDetailEnvelope struct {
		Result research.ResearchThemeDetail `json:"result"`
	}
	if response.Code != http.StatusOK || fake.themeDetailCalls != 1 {
		t.Fatalf("theme detail response=%d calls=%d body=%s", response.Code, fake.themeDetailCalls, response.Body.String())
	}
	if err := json.Unmarshal(response.Body.Bytes(), &themeDetailEnvelope); err != nil {
		t.Fatal(err)
	}
	assertStringSet(t, "evidence_role", eventRoleStrings(themeDetailEnvelope.Result.Events), []string{"driver", "supporting", "contradicting", "context"})

	request = httptest.NewRequest(http.MethodGet, Namespace+"/research/anchors?window_hours=24&limit=20", nil)
	request.Header.Set("Authorization", "Bearer cred-miniapp")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	var anchorEnvelope struct {
		Result research.ResearchAnchorPage `json:"result"`
	}
	if response.Code != http.StatusOK || fake.anchorCalls != 1 {
		t.Fatalf("anchor list response=%d calls=%d body=%s", response.Code, fake.anchorCalls, response.Body.String())
	}
	if err := json.Unmarshal(response.Body.Bytes(), &anchorEnvelope); err != nil {
		t.Fatal(err)
	}
	assertStringSet(t, "anchor_type", anchorTypeStrings(anchorEnvelope.Result.Items), []string{"policy", "supply", "demand", "technology", "cost", "geopolitics", "market_structure"})
	assertStringSet(t, "importance", anchorImportanceStrings(anchorEnvelope.Result.Items), []string{"primary", "secondary", "contextual"})
	anchorNodeJSON, err := json.Marshal(anchorEnvelope.Result.Items[0].RelatedChainNodes[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(anchorNodeJSON), `"relation_summary":"锚点关系摘要"`) || strings.Contains(string(anchorNodeJSON), "impact_summary") {
		t.Fatalf("anchor node JSON = %s", anchorNodeJSON)
	}

	request = httptest.NewRequest(http.MethodGet, Namespace+"/research/anchors/77777777-7777-4777-8777-777777777777?window_hours=24", nil)
	request.Header.Set("Authorization", "Bearer cred-miniapp")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || fake.anchorDetailCalls != 1 || !strings.Contains(response.Body.String(), `"relation_summary":"锚点关系摘要"`) {
		t.Fatalf("anchor detail response=%d calls=%d body=%s", response.Code, fake.anchorDetailCalls, response.Body.String())
	}
}

func TestResearchHandlerMapsKnownErrorsWithoutLeakingRepositoryFailures(t *testing.T) {
	for _, test := range []struct {
		name          string
		err           error
		wantStatus    int
		wantCode      string
		forbiddenText string
	}{
		{name: "invalid", err: research.ErrInvalidRequest, wantStatus: http.StatusBadRequest, wantCode: "INVALID_REQUEST"},
		{name: "not found", err: research.ErrNotFound, wantStatus: http.StatusNotFound, wantCode: "NOT_FOUND"},
		{name: "repository", err: errors.New("pq: password authentication failed"), wantStatus: http.StatusInternalServerError, wantCode: "DATA_REPOSITORY_FAILURE", forbiddenText: "password"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fake := &fakeResearchService{err: test.err}
			handler := testHandler(t, Dependencies{Research: fake})
			request := httptest.NewRequest(http.MethodGet, Namespace+"/research/themes", nil)
			request.Header.Set("Authorization", "Bearer cred-miniapp")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus || !strings.Contains(response.Body.String(), `"code":"`+test.wantCode+`"`) {
				t.Fatalf("response=%d body=%s", response.Code, response.Body.String())
			}
			if test.forbiddenText != "" && strings.Contains(response.Body.String(), test.forbiddenText) {
				t.Fatalf("response leaked internal error: %s", response.Body.String())
			}
		})
	}
}

func assertStringSet(t *testing.T, name string, got, want []string) {
	t.Helper()
	gotSet := map[string]bool{}
	for _, value := range got {
		gotSet[value] = true
	}
	for _, value := range want {
		if !gotSet[value] {
			t.Fatalf("%s missing %q: %v", name, value, got)
		}
	}
	if len(gotSet) != len(want) {
		t.Fatalf("%s contains unexpected values: %v", name, got)
	}
}

func themeImpactStrings(items []research.ResearchTheme) []string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, string(item.ImpactLevel))
	}
	return values
}

func themeStageStrings(items []research.ResearchTheme) []string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, string(item.TransmissionStage))
	}
	return values
}

func indexDirectionStrings(items []research.ResearchIndex) []string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, string(item.ImpactDirection))
	}
	return values
}

func eventRoleStrings(items []research.ResearchEvent) []string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, string(item.EvidenceRole))
	}
	return values
}

func anchorTypeStrings(items []research.ResearchAnchor) []string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, string(item.AnchorType))
	}
	return values
}

func anchorImportanceStrings(items []research.ResearchAnchor) []string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, string(item.Importance))
	}
	return values
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
	themes            research.ResearchThemePage
	theme             research.ResearchThemeDetail
	anchors           research.ResearchAnchorPage
	anchor            research.ResearchAnchorDetail
	err               error
	themeCalls        int
	themeDetailCalls  int
	anchorCalls       int
	anchorDetailCalls int
	lastThemeRequest  research.ResearchListRequest
}

func (f *fakeResearchService) ListThemes(_ context.Context, request research.ResearchListRequest) (research.ResearchThemePage, error) {
	f.themeCalls++
	f.lastThemeRequest = request
	return f.themes, f.err
}
func (f *fakeResearchService) GetTheme(context.Context, string, research.ResearchDetailRequest) (research.ResearchThemeDetail, error) {
	f.themeDetailCalls++
	return f.theme, f.err
}
func (f *fakeResearchService) ListAnchors(context.Context, research.ResearchListRequest) (research.ResearchAnchorPage, error) {
	f.anchorCalls++
	return f.anchors, f.err
}
func (f *fakeResearchService) GetAnchor(context.Context, string, research.ResearchDetailRequest) (research.ResearchAnchorDetail, error) {
	f.anchorDetailCalls++
	return f.anchor, f.err
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
