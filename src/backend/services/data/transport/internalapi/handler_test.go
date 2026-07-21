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
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
	domainimport "github.com/meierlink88/tidewise-ai/backend/services/data/domain/eventimport"
	researchanchordomainimport "github.com/meierlink88/tidewise-ai/backend/services/data/domain/researchanchorimport"
	researchdomainimport "github.com/meierlink88/tidewise-ai/backend/services/data/domain/researchthemeimport"
	"github.com/meierlink88/tidewise-ai/backend/services/data/usecase/adminquery"
	eventapp "github.com/meierlink88/tidewise-ai/backend/services/data/usecase/eventimport"
	"github.com/meierlink88/tidewise-ai/backend/services/data/usecase/rawimport"
	"github.com/meierlink88/tidewise-ai/backend/services/data/usecase/research"
	researchanchorimportapp "github.com/meierlink88/tidewise-ai/backend/services/data/usecase/researchanchorimport"
	researchimportapp "github.com/meierlink88/tidewise-ai/backend/services/data/usecase/researchthemeimport"
	"github.com/meierlink88/tidewise-ai/backend/services/data/usecase/sourcemetadata"
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
	fixture, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "testdata", "event-import", "reviewed-outbox-v1.json"))
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
	fixture, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "testdata", "event-import", "reviewed-outbox-v1.json"))
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
	fixture, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "testdata", "event-import", "reviewed-outbox-v1.json"))
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

func TestResearchThemeImportUsesDedicatedPublisherIdentityAndFrozenResult(t *testing.T) {
	now := time.Date(2026, 7, 19, 9, 30, 0, 0, time.UTC)
	importer := &fakeResearchThemeImporter{result: researchimportapp.Result{
		ReceiptID: "11111111-1111-4111-8111-111111111111", AnalysisBatchID: "batch-1",
		PayloadHash: strings.Repeat("a", 64), ThemeIDsByKey: map[string]string{"theme:a": "22222222-2222-4222-8222-222222222222"},
		Counts:      researchimportapp.Counts{Themes: 1, ChainNodeAssociations: 1, EventAssociations: 1, Receipts: 1},
		PublishedAt: now, ImportedAt: now,
	}}
	handler := testHandler(t, Dependencies{ResearchThemeImports: importer})
	body := `{"analysis_batch_id":"batch-1","window_start":"2026-07-15T00:00:00Z","window_end":"2026-07-18T00:00:00Z","themes":[{"theme_key":"theme:a","name":"主题","one_line_conclusion":"结论","impact_level":"high","transmission_path":"事件 → 影响","trading_direction":"研究方向","transmission_stage":"validation","next_checkpoint":"跟踪指标","market_confirmation_summary":"当前没有可归属的正式市场观测","chain_nodes":[{"chain_node_id":"33333333-3333-4333-8333-333333333333","relation_role":"driver","impact_summary":"驱动"}],"events":[{"event_id":"44444444-4444-4444-8444-444444444444","evidence_role":"driver","supported_claim":"支持结论"}]}]}`

	request := httptest.NewRequest(http.MethodPost, Namespace+"/research-theme-imports", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer cred-research-publisher")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || importer.calls != 1 || importer.publisher != "research-theme-publisher" {
		t.Fatalf("response=%d calls=%d publisher=%q body=%s", response.Code, importer.calls, importer.publisher, response.Body.String())
	}
	for _, expected := range []string{
		`"theme_ids_by_key":{"theme:a":"22222222-2222-4222-8222-222222222222"}`,
		`"chain_node_associations":1`, `"event_associations":1`,
		`"published_at":"2026-07-19T09:30:00Z"`, `"imported_at":"2026-07-19T09:30:00Z"`, `"replayed":false`,
	} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Fatalf("response missing %s: %s", expected, response.Body.String())
		}
	}

	importer.result.Replayed = true
	request = httptest.NewRequest(http.MethodPost, Namespace+"/research-theme-imports", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer cred-research-publisher")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"replayed":true`) {
		t.Fatalf("replay response=%d body=%s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, Namespace+"/research-theme-imports", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer cred-agent")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || importer.calls != 2 {
		t.Fatalf("wrong-scope response=%d calls=%d body=%s", response.Code, importer.calls, response.Body.String())
	}
}

func TestResearchAnchorImportUsesDedicatedPublisherIdentityAndFrozenResult(t *testing.T) {
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC)
	importer := &fakeResearchAnchorImporter{result: researchanchorimportapp.Result{
		ReceiptID: "99999999-9999-4999-8999-999999999999", ThemeID: "11111111-1111-4111-8111-111111111111",
		PayloadHash: "e006ca80db77df2b07e0028d3b499b664956fd9ff0d5b57e2d00ccc6c19741a2",
		AnchorIDsByCenterChainNodeID: map[string]string{
			"22222222-2222-4222-8222-222222222222": "534d83be-774b-51d9-ad00-cdee4ba91799",
			"33333333-3333-4333-8333-333333333333": "5c18fc57-6bd8-5612-9a24-01a4e928b761",
		},
		Counts:      researchanchorimportapp.Counts{Anchors: 2, EventAssociations: 4, PathNodes: 4, Receipts: 1},
		PublishedAt: now, ImportedAt: now,
	}}
	handler := testHandler(t, Dependencies{ResearchAnchorImports: importer})
	body, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "testdata", "reasoning-tree-v1", "01-multi-anchor-import-request.json"))
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, Namespace+"/research-anchor-imports", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer cred-research-publisher")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || importer.calls != 1 || importer.publisher != "research-theme-publisher" {
		t.Fatalf("response=%d calls=%d publisher=%q body=%s", response.Code, importer.calls, importer.publisher, response.Body.String())
	}
	for _, expected := range []string{
		`"anchor_ids_by_center_chain_node_id":{"22222222-2222-4222-8222-222222222222":"534d83be-774b-51d9-ad00-cdee4ba91799","33333333-3333-4333-8333-333333333333":"5c18fc57-6bd8-5612-9a24-01a4e928b761"}`,
		`"anchors":2`, `"event_associations":4`, `"path_nodes":4`,
		`"published_at":"2026-07-20T09:00:00Z"`, `"imported_at":"2026-07-20T09:00:00Z"`, `"replayed":false`,
	} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Fatalf("response missing %s: %s", expected, response.Body.String())
		}
	}

	importer.result.Replayed = true
	request = httptest.NewRequest(http.MethodPost, Namespace+"/research-anchor-imports", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer cred-research-publisher")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"replayed":true`) {
		t.Fatalf("replay response=%d body=%s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, Namespace+"/research-anchor-imports", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer cred-agent")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || importer.calls != 2 {
		t.Fatalf("wrong-scope response=%d calls=%d body=%s", response.Code, importer.calls, response.Body.String())
	}
}

func TestResearchAnchorImportReturnsFrozenActionableErrors(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "testdata", "reasoning-tree-v1", "01-multi-anchor-import-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	importer := &fakeResearchAnchorImporter{err: &researchanchorimportapp.ReferenceError{
		Kind: researchanchorimportapp.ReferenceInvalid, CenterChainNodeID: "22222222-2222-4222-8222-222222222222",
		Path: "anchors[0].events[0].event_id", Reference: "55555555-5555-4555-8555-555555555555",
	}}
	handler := testHandler(t, Dependencies{ResearchAnchorImports: importer})

	request := httptest.NewRequest(http.MethodPost, Namespace+"/research-anchor-imports", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer cred-research-publisher")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	for _, expected := range []string{`"code":"RESEARCH_ANCHOR_REFERENCE_INVALID"`, `"center_chain_node_id":"22222222-2222-4222-8222-222222222222"`, `"path":"anchors[0].events[0].event_id"`} {
		if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), expected) {
			t.Fatalf("reference response=%d missing %s: %s", response.Code, expected, response.Body.String())
		}
	}

	importer.err = &researchanchorimportapp.ContractError{Path: "anchors", Reference: "33333333-3333-4333-8333-333333333333"}
	request = httptest.NewRequest(http.MethodPost, Namespace+"/research-anchor-imports", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer cred-research-publisher")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"RESEARCH_ANCHOR_IMPORT_REJECTED"`) {
		t.Fatalf("contract response=%d body=%s", response.Code, response.Body.String())
	}

	importer.err = researchanchorimportapp.ErrPayloadConflict
	request = httptest.NewRequest(http.MethodPost, Namespace+"/research-anchor-imports", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer cred-research-publisher")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), `"code":"RESEARCH_ANCHOR_PAYLOAD_CONFLICT"`) {
		t.Fatalf("conflict response=%d body=%s", response.Code, response.Body.String())
	}

	calls := importer.calls
	request = httptest.NewRequest(http.MethodPost, Namespace+"/research-anchor-imports", strings.NewReader(`{"theme_id":null,"anchors":[]}`))
	request.Header.Set("Authorization", "Bearer cred-research-publisher")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || importer.calls != calls || !strings.Contains(response.Body.String(), `"path":"theme_id"`) {
		t.Fatalf("strict decode response=%d calls=%d body=%s", response.Code, importer.calls, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, Namespace+"/research-anchor-imports", strings.NewReader(`{"theme_id":"AAAAAAAA-AAAA-4AAA-8AAA-AAAAAAAAAAAA","anchors":[]}`))
	request.Header.Set("Authorization", "Bearer cred-research-publisher")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || importer.calls != calls || !strings.Contains(response.Body.String(), `"code":"INVALID_REQUEST"`) || !strings.Contains(response.Body.String(), `"path":"theme_id"`) {
		t.Fatalf("UUID format response=%d calls=%d body=%s", response.Code, importer.calls, response.Body.String())
	}
}

func TestResearchThemeImportReturnsActionableValidationAndConflictErrors(t *testing.T) {
	importer := &fakeResearchThemeImporter{err: &researchimportapp.ReferenceError{
		ThemeKey: "theme:a", Path: "themes[0].events[0].event_id", Reference: "44444444-4444-4444-8444-444444444444",
	}}
	handler := testHandler(t, Dependencies{ResearchThemeImports: importer})
	body := `{"analysis_batch_id":"batch-1","window_start":"2026-07-15T00:00:00Z","window_end":"2026-07-18T00:00:00Z","themes":[{"theme_key":"theme:a","name":"主题","one_line_conclusion":"结论","impact_level":"high","transmission_path":"事件 → 影响","trading_direction":"研究方向","transmission_stage":"validation","next_checkpoint":"跟踪指标","market_confirmation_summary":"未观察到市场证据","chain_nodes":[{"chain_node_id":"33333333-3333-4333-8333-333333333333","relation_role":"driver","impact_summary":"驱动"}],"events":[{"event_id":"44444444-4444-4444-8444-444444444444","evidence_role":"driver","supported_claim":"支持结论"}]}]}`

	request := httptest.NewRequest(http.MethodPost, Namespace+"/research-theme-imports", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer cred-research-publisher")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), `"theme_key":"theme:a"`) || !strings.Contains(response.Body.String(), `"path":"themes[0].events[0].event_id"`) {
		t.Fatalf("validation response=%d body=%s", response.Code, response.Body.String())
	}

	importer.err = &researchdomainimport.ValidationError{ThemeKey: "theme:a", Path: "themes[0].transmission_stage", Reference: "unknown"}
	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, Namespace+"/research-theme-imports", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer cred-research-publisher")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "RESEARCH_THEME_IMPORT_REJECTED") || !strings.Contains(response.Body.String(), `"path":"themes[0].transmission_stage"`) {
		t.Fatalf("contract validation response=%d body=%s", response.Code, response.Body.String())
	}

	importer.err = researchimportapp.ErrPayloadConflict
	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, Namespace+"/research-theme-imports", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer cred-research-publisher")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "RESEARCH_THEME_PAYLOAD_CONFLICT") {
		t.Fatalf("conflict response=%d body=%s", response.Code, response.Body.String())
	}
}

func TestResearchThemeImportReturnsActionableStrictDecodeError(t *testing.T) {
	importer := &fakeResearchThemeImporter{}
	handler := testHandler(t, Dependencies{ResearchThemeImports: importer})
	body := `{"analysis_batch_id":"batch-1","window_start":"2026-07-15T00:00:00Z","window_end":"2026-07-18T00:00:00Z","themes":[{"Theme_Key":"shadow","theme_key":"theme:a","name":"主题","one_line_conclusion":"结论","impact_level":"high","transmission_path":"事件 → 影响","trading_direction":"研究方向","transmission_stage":"validation","next_checkpoint":"跟踪指标","market_confirmation_summary":"未观察到市场证据","chain_nodes":[{"chain_node_id":"33333333-3333-4333-8333-333333333333","relation_role":"driver","impact_summary":"驱动"}],"events":[{"event_id":"44444444-4444-4444-8444-444444444444","evidence_role":"driver","supported_claim":"支持结论"}]}]}`

	request := httptest.NewRequest(http.MethodPost, Namespace+"/research-theme-imports", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer cred-research-publisher")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest || importer.calls != 0 {
		t.Fatalf("response=%d calls=%d body=%s", response.Code, importer.calls, response.Body.String())
	}
	for _, expected := range []string{`"code":"INVALID_REQUEST"`, `"theme_key":"theme:a"`, `"path":"themes[0].Theme_Key"`} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Fatalf("response missing %s: %s", expected, response.Body.String())
		}
	}
}

func TestResearchAdminAndAgentMetadataAreSingleAggregateCallsAndMetadataIsRedacted(t *testing.T) {
	research := &fakeResearchService{
		themes: research.ResearchThemePage{WindowStart: time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC), WindowEnd: time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC), AsOf: time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC), Items: []research.ResearchTheme{}},
	}
	admin := &fakeAdminStore{
		rawPage:   adminquery.RawDocumentPage{Page: 1, PageSize: 20, Items: []domain.RawDocument{{ID: "11111111-1111-5111-8111-111111111111", SourceID: "22222222-2222-5222-8222-222222222222", Title: "Raw", CollectedAt: time.Date(2026, 7, 17, 1, 0, 0, 0, time.UTC), IngestStatus: domain.IngestStatusCollected}}},
		eventPage: adminquery.EventPage{Page: 1, PageSize: 20, Items: []domain.Event{}},
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
		domain.ImpactLevelHigh,
	}
	themeStages := []domain.TransmissionStage{
		domain.TransmissionStageIdentification, domain.TransmissionStageValidation,
		domain.TransmissionStageDiffusion, domain.TransmissionStageDampening,
	}
	themes := make([]research.ResearchTheme, 0, len(themeStages))
	for index, stage := range themeStages {
		themes = append(themes, research.ResearchTheme{
			ID: "11111111-1111-4111-8111-111111111111", Name: "主题", OneLineConclusion: "结论",
			ImpactLevel: themeImpacts[index], TransmissionPath: "供给到价格", TradingDirection: "关注供需验证后的方向变化",
			TransmissionStage: stage, NextCheckpoint: "下次数据", MarketConfirmationSummary: "市场验证摘要", PublishedAt: now,
			AffectedChainNodes: []research.ResearchThemeChainNode{}, RelatedIndices: []research.ResearchIndex{},
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
	assertStringSet(t, "transmission_stage", themeStageStrings(themeEnvelope.Result.Items), []string{"identification", "validation", "diffusion", "dampening"})
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

func TestResearchReasoningTreeHandlersExposeThemeSubresourcesAndRemoveLegacyDataRoutes(t *testing.T) {
	themeID := "11111111-1111-4111-8111-111111111111"
	anchorID := "22222222-2222-4222-8222-222222222222"
	fake := &fakeResearchService{
		reasoningTrees: research.ResearchReasoningTreeList{
			Theme: research.ResearchTheme{ID: themeID, RelatedIndices: []research.ResearchIndex{}, AffectedChainNodes: []research.ResearchThemeChainNode{}},
			ReasoningTrees: []research.ResearchReasoningTreeSummary{{
				AnchorID:        anchorID,
				CenterChainNode: research.ResearchReasoningTreeChainNode{ID: "33333333-3333-4333-8333-333333333333", Name: "光模块"},
			}},
		},
		reasoningTree: research.ResearchReasoningTreeDetail{
			ThemeID: themeID,
			ReasoningTree: research.ResearchReasoningTree{
				AnchorID:        anchorID,
				CenterChainNode: research.ResearchReasoningTreeChainNode{ID: "33333333-3333-4333-8333-333333333333", Name: "光模块"},
				EventCount:      1,
				Events:          []research.ResearchReasoningTreeEvent{{EventID: "44444444-4444-4444-8444-444444444444", EvidenceRole: "driver", EvidenceSummary: "直接驱动"}},
				PathNodes:       []research.ResearchReasoningTreePathNode{},
			},
		},
	}
	handler := testHandler(t, Dependencies{Research: fake})

	request := httptest.NewRequest(http.MethodGet, Namespace+"/research/themes/"+themeID+"/reasoning-trees", nil)
	request.Header.Set("Authorization", "Bearer cred-miniapp")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || fake.reasoningTreeListCalls != 1 || !strings.Contains(response.Body.String(), `"center_chain_node":{"id":"33333333-3333-4333-8333-333333333333","name":"光模块"}`) {
		t.Fatalf("list response=%d calls=%d body=%s", response.Code, fake.reasoningTreeListCalls, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, Namespace+"/research/themes/"+themeID+"/reasoning-trees/"+anchorID, nil)
	request.Header.Set("Authorization", "Bearer cred-miniapp")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || fake.reasoningTreeDetailCalls != 1 || !strings.Contains(response.Body.String(), `"evidence_summary":"直接驱动"`) {
		t.Fatalf("detail response=%d calls=%d body=%s", response.Code, fake.reasoningTreeDetailCalls, response.Body.String())
	}

	for _, path := range []string{Namespace + "/research/anchors", Namespace + "/research/anchors/" + anchorID} {
		request = httptest.NewRequest(http.MethodGet, path, nil)
		request.Header.Set("Authorization", "Bearer cred-miniapp")
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Fatalf("legacy path %s status=%d body=%s", path, response.Code, response.Body.String())
		}
	}
}

func TestResearchReasoningTreeHandlerMapsFrozenErrors(t *testing.T) {
	themeID := "11111111-1111-4111-8111-111111111111"
	for _, test := range []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "theme missing", err: research.ErrThemeNotFound, wantStatus: http.StatusNotFound, wantCode: "RESEARCH_THEME_NOT_FOUND"},
		{name: "trees missing", err: research.ErrReasoningTreesNotFound, wantStatus: http.StatusNotFound, wantCode: "RESEARCH_REASONING_TREES_NOT_FOUND"},
		{name: "tree missing", err: research.ErrReasoningTreeNotFound, wantStatus: http.StatusNotFound, wantCode: "RESEARCH_REASONING_TREE_NOT_FOUND"},
		{name: "invariant", err: research.ErrReasoningTreeInvariantViolation, wantStatus: http.StatusInternalServerError, wantCode: "RESEARCH_REASONING_TREE_INVARIANT_VIOLATION"},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler := testHandler(t, Dependencies{Research: &fakeResearchService{err: test.err}})
			request := httptest.NewRequest(http.MethodGet, Namespace+"/research/themes/"+themeID+"/reasoning-trees", nil)
			request.Header.Set("Authorization", "Bearer cred-miniapp")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus || !strings.Contains(response.Body.String(), `"code":"`+test.wantCode+`"`) {
				t.Fatalf("response=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
}

func TestResearchReasoningTreeHandlersMatchSharedSuccessFixtures(t *testing.T) {
	themeID := "11111111-1111-4111-8111-111111111111"
	fixtureDirectory := filepath.Join("..", "..", "..", "..", "..", "testdata", "reasoning-tree-v1")

	var listEnvelope struct {
		RequestID string                             `json:"request_id"`
		Result    research.ResearchReasoningTreeList `json:"result"`
	}
	loadJSONFixture(t, filepath.Join(fixtureDirectory, "01-reasoning-tree-list-result.json"), &listEnvelope)
	listHandler := testHandlerWithRequestID(t, Dependencies{Research: &fakeResearchService{reasoningTrees: listEnvelope.Result}}, listEnvelope.RequestID)
	assertHandlerJSONMatchesFixture(t, listHandler, Namespace+"/research/themes/"+themeID+"/reasoning-trees", filepath.Join(fixtureDirectory, "01-reasoning-tree-list-result.json"))

	for _, fixture := range []struct {
		name     string
		anchorID string
		file     string
	}{
		{name: "with contradiction", anchorID: "534d83be-774b-51d9-ad00-cdee4ba91799", file: "02-reasoning-tree-with-contradiction-result.json"},
		{name: "without contradiction and unquantified", anchorID: "5c18fc57-6bd8-5612-9a24-01a4e928b761", file: "03-reasoning-tree-without-contradiction-unquantified-result.json"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			var detailEnvelope struct {
				RequestID string                               `json:"request_id"`
				Result    research.ResearchReasoningTreeDetail `json:"result"`
			}
			fixturePath := filepath.Join(fixtureDirectory, fixture.file)
			loadJSONFixture(t, fixturePath, &detailEnvelope)
			detailHandler := testHandlerWithRequestID(t, Dependencies{Research: &fakeResearchService{reasoningTree: detailEnvelope.Result}}, detailEnvelope.RequestID)
			assertHandlerJSONMatchesFixture(t, detailHandler, Namespace+"/research/themes/"+themeID+"/reasoning-trees/"+fixture.anchorID, fixturePath)
		})
	}
}

func TestResearchReasoningTreeHandlerMatchesSharedMissingTreesFixture(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "..", "..", "..", "testdata", "reasoning-tree-v1", "04-theme-without-reasoning-trees-error.json")
	var fixture struct {
		Request struct {
			Method string `json:"method"`
			Path   string `json:"path"`
		} `json:"request"`
		Response struct {
			Status int             `json:"status"`
			Body   json.RawMessage `json:"body"`
		} `json:"response"`
	}
	loadJSONFixture(t, fixturePath, &fixture)
	var expectedBody struct {
		RequestID string `json:"request_id"`
	}
	if err := json.Unmarshal(fixture.Response.Body, &expectedBody); err != nil {
		t.Fatal(err)
	}
	handler := testHandlerWithRequestID(t, Dependencies{Research: &fakeResearchService{err: research.ErrReasoningTreesNotFound}}, expectedBody.RequestID)
	request := httptest.NewRequest(fixture.Request.Method, fixture.Request.Path, nil)
	request.Header.Set("Authorization", "Bearer cred-miniapp")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != fixture.Response.Status {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var got, want any
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(fixture.Response.Body, &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("response drifted\ngot:  %s\nwant: %s", response.Body.String(), fixture.Response.Body)
	}
}

func loadJSONFixture(t *testing.T, path string, target any) {
	t.Helper()
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		t.Fatal(err)
	}
}

func assertHandlerJSONMatchesFixture(t *testing.T, handler http.Handler, path, fixturePath string) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("Authorization", "Bearer cred-miniapp")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("GET %s status=%d body=%s", path, response.Code, response.Body.String())
	}
	var got, want any
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(payload, &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GET %s response drifted\ngot:  %s\nwant: %s", path, response.Body.String(), payload)
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

func testHandler(t *testing.T, dependencies Dependencies) http.Handler {
	return testHandlerWithRequestID(t, dependencies, "generated-request")
}

func testHandlerWithRequestID(t *testing.T, dependencies Dependencies, requestID string) http.Handler {
	t.Helper()
	authenticator, err := NewAuthenticator([]Credential{
		{Secret: "cred-agent", Principal: Principal{Identity: "agent-run", Scopes: []string{ScopeRawImport, ScopeReviewedEventImport, ScopeSourceMetadataRead}}},
		{Secret: "cred-research-publisher", Principal: Principal{Identity: "research-theme-publisher", Scopes: []string{ScopeResearchImport}}},
		{Secret: "cred-miniapp", Principal: Principal{Identity: "miniapp-service", Scopes: []string{ScopeResearchRead}}},
		{Secret: "cred-admin", Principal: Principal{Identity: "adminportal-service", Scopes: []string{ScopeAdminRead}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	dependencies.Authenticator = authenticator
	dependencies.NewRequestID = func() string { return requestID }
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

type fakeResearchThemeImporter struct {
	result    researchimportapp.Result
	batch     researchdomainimport.Batch
	publisher string
	err       error
	calls     int
}

type fakeResearchAnchorImporter struct {
	result      researchanchorimportapp.Result
	publication researchanchordomainimport.Publication
	publisher   string
	err         error
	calls       int
}

func (f *fakeResearchAnchorImporter) Import(_ context.Context, publisher string, publication researchanchordomainimport.Publication) (researchanchorimportapp.Result, error) {
	f.calls++
	f.publisher = publisher
	f.publication = publication
	return f.result, f.err
}

func (f *fakeResearchThemeImporter) Import(_ context.Context, publisher string, batch researchdomainimport.Batch) (researchimportapp.Result, error) {
	f.calls++
	f.publisher = publisher
	f.batch = batch
	return f.result, f.err
}

func (f *fakeReviewedImporter) Import(_ context.Context, pkg domainimport.Package) (eventapp.Result, error) {
	f.calls++
	f.pkg = pkg
	return f.result, f.err
}

type fakeResearchService struct {
	themes                   research.ResearchThemePage
	theme                    research.ResearchThemeDetail
	reasoningTrees           research.ResearchReasoningTreeList
	reasoningTree            research.ResearchReasoningTreeDetail
	err                      error
	themeCalls               int
	themeDetailCalls         int
	reasoningTreeListCalls   int
	reasoningTreeDetailCalls int
	lastThemeRequest         research.ResearchListRequest
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
func (f *fakeResearchService) ListReasoningTrees(context.Context, string) (research.ResearchReasoningTreeList, error) {
	f.reasoningTreeListCalls++
	return f.reasoningTrees, f.err
}
func (f *fakeResearchService) GetReasoningTree(context.Context, string, string) (research.ResearchReasoningTreeDetail, error) {
	f.reasoningTreeDetailCalls++
	return f.reasoningTree, f.err
}

type fakeAdminStore struct {
	rawPage     adminquery.RawDocumentPage
	eventPage   adminquery.EventPage
	sources     []domain.SourceCatalog
	rawFilter   adminquery.RawDocumentListRequest
	rawCalls    int
	eventCalls  int
	sourceCalls int
}

func (f *fakeAdminStore) ListRawDocuments(_ context.Context, filter adminquery.RawDocumentListRequest) (adminquery.RawDocumentPage, error) {
	f.rawCalls++
	f.rawFilter = filter
	return f.rawPage, nil
}
func (f *fakeAdminStore) ListEvents(context.Context, adminquery.EventListRequest) (adminquery.EventPage, error) {
	f.eventCalls++
	return f.eventPage, nil
}
func (f *fakeAdminStore) ListSourceCatalogs(context.Context, adminquery.SourceCatalogListRequest) ([]domain.SourceCatalog, error) {
	f.sourceCalls++
	return f.sources, nil
}

func (f *fakeAdminStore) List(_ context.Context, request sourcemetadata.ListRequest) (sourcemetadata.Page, error) {
	f.sourceCalls++
	if request.Cursor != "" && len(f.sources) > 0 && request.Cursor == f.sources[len(f.sources)-1].ID {
		return sourcemetadata.Page{Items: []domain.SourceCatalog{}}, nil
	}
	return sourcemetadata.Page{Items: f.sources}, nil
}

func decodeJSON(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	return value
}
