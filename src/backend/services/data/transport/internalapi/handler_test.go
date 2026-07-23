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
	publicationdomain "github.com/meierlink88/tidewise-ai/backend/services/data/domain/eventpublication"
	researchanchordomainimport "github.com/meierlink88/tidewise-ai/backend/services/data/domain/researchanchorimport"
	researchdomainimport "github.com/meierlink88/tidewise-ai/backend/services/data/domain/researchthemeimport"
	"github.com/meierlink88/tidewise-ai/backend/services/data/usecase/adminquery"
	eventpublicationapp "github.com/meierlink88/tidewise-ai/backend/services/data/usecase/eventpublication"
	"github.com/meierlink88/tidewise-ai/backend/services/data/usecase/research"
	researchanchorimportapp "github.com/meierlink88/tidewise-ai/backend/services/data/usecase/researchanchorimport"
	researchimportapp "github.com/meierlink88/tidewise-ai/backend/services/data/usecase/researchthemeimport"
)

func TestUnifiedDataNamespaceExposesOnlySupportedOperations(t *testing.T) {
	if Namespace != "/api/data/v1" {
		t.Fatalf("Namespace = %q, want /api/data/v1", Namespace)
	}

	handler := testHandler(t, Dependencies{})
	for _, active := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: Namespace + "/reviewed-event-imports"},
		{method: http.MethodGet, path: Namespace + "/research/themes"},
	} {
		request := httptest.NewRequest(active.method, active.path, strings.NewReader(`{}`))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s response=%d body=%s", active.method, active.path, response.Code, response.Body.String())
		}
	}

	for _, retired := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/internal/data/v1/research/themes"},
		{method: http.MethodPost, path: "/internal/data/v2/reviewed-event-imports"},
		{method: http.MethodPost, path: Namespace + "/raw-document-imports"},
		{method: http.MethodGet, path: Namespace + "/raw-document-imports/legacy-key"},
	} {
		request := httptest.NewRequest(retired.method, retired.path, strings.NewReader(`{}`))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Fatalf("%s %s response=%d, want 404; body=%s", retired.method, retired.path, response.Code, response.Body.String())
		}
	}
}

func TestEventPublicationValidationDetailsRemainAnObject(t *testing.T) {
	importer := &fakeEventPublicationImporter{err: &publicationdomain.ValidationError{Issues: []publicationdomain.ValidationIssue{{
		Path: "events[0].title", Code: "required", Message: "title is required",
	}}}}
	payload, err := json.Marshal(eventPublicationFixture("error-details"))
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, Namespace+"/reviewed-event-imports", bytes.NewReader(payload))
	request.Header.Set("Authorization", "Bearer cred-agent")
	response := httptest.NewRecorder()

	testHandler(t, Dependencies{EventPublications: importer}).ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusUnprocessableEntity, response.Body.String())
	}
	var envelope struct {
		Error struct {
			Details struct {
				Issues []publicationdomain.ValidationIssue `json:"issues"`
			} `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if len(envelope.Error.Details.Issues) != 1 || envelope.Error.Details.Issues[0].Path != "events[0].title" {
		t.Fatalf("details = %#v", envelope.Error.Details)
	}
}

func TestNewAuthenticatorRejectsOversizedServiceIdentity(t *testing.T) {
	_, err := NewAuthenticator([]Credential{{
		Secret: "secret",
		Principal: Principal{
			Identity: strings.Repeat("a", 201),
			Scopes:   []string{ScopeReviewedEventImport},
		},
	}})
	if err == nil || !strings.Contains(err.Error(), "at most 200 characters") {
		t.Fatalf("NewAuthenticator error = %v, want identity length error", err)
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

func TestResearchAndAdminQueriesUseSingleAggregateCalls(t *testing.T) {
	research := &fakeResearchService{
		themes: research.ResearchThemePage{WindowStart: time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC), WindowEnd: time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC), AsOf: time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC), Items: []research.ResearchTheme{}},
	}
	admin := &fakeAdminStore{
		rawPage:   adminquery.RawDocumentPage{Page: 1, PageSize: 20, Items: []domain.RawDocument{{ID: "11111111-1111-5111-8111-111111111111", ContractVersion: 2, ArtifactID: "artifact-1", SourceRef: "source:reuters:world", Title: "Raw", CollectedAt: time.Date(2026, 7, 17, 1, 0, 0, 0, time.UTC), IngestStatus: domain.IngestStatusCollected}}},
		eventPage: adminquery.EventPage{Page: 1, PageSize: 20, Items: []domain.Event{}},
	}
	handler := testHandler(t, Dependencies{Research: research, Admin: admin})

	request := httptest.NewRequest(http.MethodGet, Namespace+"/research/themes?window_hours=24&limit=20", nil)
	request.Header.Set("Authorization", "Bearer cred-miniapp")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || research.themeCalls != 1 {
		t.Fatalf("research response=%d calls=%d body=%s", response.Code, research.themeCalls, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, Namespace+"/raw-documents?source_ref=source%3Areuters%3Aworld&ingest_status=collected&page=1&page_size=20", nil)
	request.Header.Set("Authorization", "Bearer cred-admin")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || admin.rawCalls != 1 || admin.rawFilter.SourceRef != "source:reuters:world" || admin.rawFilter.IngestStatus != domain.IngestStatusCollected {
		t.Fatalf("admin raw response=%d calls=%d filter=%#v body=%s", response.Code, admin.rawCalls, admin.rawFilter, response.Body.String())
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
		{Secret: "cred-agent", Principal: Principal{Identity: "agent-run", Scopes: []string{ScopeReviewedEventImport}}},
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

type fakeResearchThemeImporter struct {
	result    researchimportapp.Result
	batch     researchdomainimport.Batch
	publisher string
	err       error
	calls     int
}

type fakeEventPublicationImporter struct {
	err error
}

func (f *fakeEventPublicationImporter) Import(context.Context, string, publicationdomain.Publication) (eventpublicationapp.Result, error) {
	return eventpublicationapp.Result{}, f.err
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
	rawPage    adminquery.RawDocumentPage
	eventPage  adminquery.EventPage
	rawFilter  adminquery.RawDocumentListRequest
	rawCalls   int
	eventCalls int
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
func decodeJSON(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	return value
}
