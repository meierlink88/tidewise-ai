package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	dataapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	evidencebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/evidence"
	evidencedata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/evidence"
	evidenceservice "github.com/meierlink88/tidewise-ai/data-service/backend/internal/service/evidence"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/research"
)

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
		serverTestEventSemanticService{}, application, serverTestRawDocumentService{},
		serverTestCountryService{}, serverTestOrganizationService{}, authenticator, nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	payload := `{
		"raw_evidence": {
			"raw_evidence_id":"RAW_category_000000000000000000",
			"source_id":"SRC_category_000000000000000000",
			"source_name":"Example Wire",
			"source_level":"L2_WIRE",
			"source_url":"https://example.test/category",
			"is_original":true,
			"raw_text":"Powell expects another rate increase this year.",
			"collected_at":"2026-08-14T01:05:00Z",
			"keywords":["美联储","加息"],
			"category_ids":["EVC_009","EVC_005"]
		}
	}`
	created := productionEvidenceRequest(t, server, http.MethodPost, dataapi.APIPrefix+"/raw-evidence-publications", "raw-evidence-write-token", payload, http.StatusCreated)
	if created["result"].(map[string]any)["raw_evidence_id"] != "RAW_category_000000000000000000" {
		t.Fatalf("created Raw Evidence envelope = %#v", created)
	}

	detail := productionEvidenceRequest(t, server, http.MethodGet, dataapi.APIPrefix+"/raw-evidences/RAW_category_000000000000000000", "raw-evidence-read-token", "", http.StatusOK)
	raw := detail["result"].(map[string]any)["raw_evidence"].(map[string]any)
	categories := raw["categories"].([]any)
	if len(categories) != 2 || categories[0].(map[string]any)["id"] != "EVC_005" || categories[1].(map[string]any)["id"] != "EVC_009" {
		t.Fatalf("Raw Evidence categories = %#v", categories)
	}
	if categories[0].(map[string]any)["code"] != "FORECAST_PLAN_OUTLOOK" || categories[0].(map[string]any)["description"] == "" {
		t.Fatalf("resolved Evidence Category = %#v", categories[0])
	}

	reversed := bytes.Replace([]byte(payload), []byte(`"category_ids":["EVC_009","EVC_005"]`), []byte(`"category_ids":["EVC_005","EVC_009"]`), 1)
	productionEvidenceRequest(t, server, http.MethodPost, dataapi.APIPrefix+"/raw-evidence-publications", "raw-evidence-write-token", string(reversed), http.StatusCreated)
	var linkCount int
	if err := db.QueryRow(`SELECT count(*) FROM raw_evidence_category_links WHERE raw_evidence_id = 'RAW_category_000000000000000000'`).Scan(&linkCount); err != nil {
		t.Fatal(err)
	}
	if linkCount != 2 {
		t.Fatalf("Raw Evidence Category link count = %d, want 2", linkCount)
	}
	var earliestLink time.Time
	if err := db.QueryRow(`SELECT min(created_at) FROM raw_evidence_category_links WHERE raw_evidence_id = 'RAW_category_000000000000000000'`).Scan(&earliestLink); err != nil {
		t.Fatal(err)
	}
	if earliestLink.IsZero() {
		t.Fatal("Raw Evidence Category link has zero created_at")
	}

	drift := bytes.Replace([]byte(payload), []byte(`"category_ids":["EVC_009","EVC_005"]`), []byte(`"category_ids":["EVC_009"]`), 1)
	productionEvidenceError(t, server, http.MethodPost, dataapi.APIPrefix+"/raw-evidence-publications", "raw-evidence-write-token", string(drift), http.StatusConflict, "EVIDENCE_PUBLICATION_CONFLICT")
	unknown := bytes.Replace([]byte(payload), []byte("RAW_category_000000000000000000"), []byte("RAW_unknown_0000000000000000000"), 1)
	unknown = bytes.Replace(unknown, []byte(`"EVC_009","EVC_005"`), []byte(`"EVC_999"`), 1)
	productionEvidenceError(t, server, http.MethodPost, dataapi.APIPrefix+"/raw-evidence-publications", "raw-evidence-write-token", string(unknown), http.StatusUnprocessableEntity, "EVIDENCE_PUBLICATION_REFERENCE_INVALID")
	duplicate := bytes.Replace([]byte(payload), []byte("RAW_category_000000000000000000"), []byte("RAW_duplicate_00000000000000000"), 1)
	duplicate = bytes.Replace(duplicate, []byte(`"EVC_009","EVC_005"`), []byte(`"EVC_005","EVC_005"`), 1)
	productionEvidenceError(t, server, http.MethodPost, dataapi.APIPrefix+"/raw-evidence-publications", "raw-evidence-write-token", string(duplicate), http.StatusBadRequest, "INVALID_REQUEST")
	malformed := bytes.Replace([]byte(payload), []byte("RAW_category_000000000000000000"), []byte("RAW_badcategory_000000000000000"), 1)
	malformed = bytes.Replace(malformed, []byte(`"EVC_009","EVC_005"`), []byte(`"BAD_ID"`), 1)
	malformedError := productionEvidenceError(t, server, http.MethodPost, dataapi.APIPrefix+"/raw-evidence-publications", "raw-evidence-write-token", string(malformed), http.StatusBadRequest, "INVALID_REQUEST")
	malformedIssues := malformedError["error"].(map[string]any)["details"].(map[string]any)["issues"].([]any)
	if malformedIssues[0].(map[string]any)["path"] != "raw_evidence.category_ids[0]" {
		t.Fatalf("malformed category issue = %#v", malformedIssues[0])
	}
	var failedWriteCount int
	if err := db.QueryRow(`SELECT count(*) FROM raw_evidences WHERE raw_evidence_id IN ('RAW_unknown_0000000000000000000', 'RAW_duplicate_00000000000000000', 'RAW_badcategory_000000000000000')`).Scan(&failedWriteCount); err != nil {
		t.Fatal(err)
	}
	if failedWriteCount != 0 {
		t.Fatalf("failed category publications left %d Raw Evidence rows", failedWriteCount)
	}

	productionEvidenceError(t, server, http.MethodGet, dataapi.APIPrefix+"/raw-evidences/RAW_missing_0000000000000000000", "raw-evidence-read-token", "", http.StatusNotFound, "RAW_EVIDENCE_NOT_FOUND")
	productionEvidenceError(t, server, http.MethodGet, dataapi.APIPrefix+"/raw-evidences/"+string(bytes.Repeat([]byte("X"), 33)), "raw-evidence-read-token", "", http.StatusBadRequest, "INVALID_REQUEST")
	productionEvidenceError(t, server, http.MethodGet, dataapi.APIPrefix+"/raw-evidences/RAW_category_000000000000000000", "raw-evidence-write-token", "", http.StatusForbidden, "FORBIDDEN")
	productionEvidenceError(t, server, http.MethodGet, dataapi.APIPrefix+"/raw-evidences/RAW_category_000000000000000000", "", "", http.StatusUnauthorized, "UNAUTHENTICATED")

	legacy := bytes.Replace([]byte(payload), []byte("RAW_category_000000000000000000"), []byte("RAW_uncategorized_00000000000000"), 1)
	legacy = bytes.Replace(legacy, []byte(`,
			"category_ids":["EVC_009","EVC_005"]`), nil, 1)
	productionEvidenceRequest(t, server, http.MethodPost, dataapi.APIPrefix+"/raw-evidence-publications", "raw-evidence-write-token", string(legacy), http.StatusCreated)
	legacyDetail := productionEvidenceRequest(t, server, http.MethodGet, dataapi.APIPrefix+"/raw-evidences/RAW_uncategorized_00000000000000", "raw-evidence-read-token", "", http.StatusOK)
	legacyCategories := legacyDetail["result"].(map[string]any)["raw_evidence"].(map[string]any)["categories"].([]any)
	if len(legacyCategories) != 0 {
		t.Fatalf("uncategorized Raw Evidence categories = %#v", legacyCategories)
	}

	if _, err := db.Exec(`INSERT INTO raw_evidence_category_links (raw_evidence_id, category_id) VALUES ('RAW_category_000000000000000000', 'EVC_005')`); err == nil {
		t.Fatal("database accepted a duplicate Raw Evidence Category link")
	}
	if _, err := db.Exec(`INSERT INTO raw_evidence_category_links (raw_evidence_id, category_id) VALUES ('RAW_missing_0000000000000000000', 'EVC_001')`); err == nil {
		t.Fatal("database accepted a Raw Evidence Category link with a missing Raw Evidence")
	}
	if _, err := db.Exec(`DELETE FROM evidence_categories WHERE id = 'EVC_005'`); err == nil {
		t.Fatal("database deleted a referenced Evidence Category")
	}
}

func assertEvidenceCategoryCatalog(t *testing.T, db interface {
	Query(string, ...any) (*sql.Rows, error)
}) {
	t.Helper()
	want := [][3]string{
		{"EVC_001", "EVENT_BRIEF", "事件快讯"},
		{"EVC_002", "FINANCIAL_REPORT_DATA_SUMMARY", "财报数据摘要"},
		{"EVC_003", "MARKET_MOVEMENT_BRIEF", "行情异动简讯"},
		{"EVC_004", "MARKET_MOVEMENT_ANALYSIS", "市场异动分析"},
		{"EVC_005", "FORECAST_PLAN_OUTLOOK", "预测/计划/展望"},
		{"EVC_006", "INDUSTRY_THEME_ANALYSIS", "行业/主题分析"},
		{"EVC_007", "IN_DEPTH_REPORT", "专题/深度报道"},
		{"EVC_008", "POLICY_DOCUMENT_SUMMARY", "政策文件摘要"},
		{"EVC_009", "INTERVIEW_OR_STATEMENT", "人物访谈/表态"},
		{"EVC_010", "SOCIAL_MEDIA_BRIEF", "社交媒体快讯"},
		{"EVC_011", "COMMENTARY_EDITORIAL_OPINION", "评论/社论/观点"},
	}
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
