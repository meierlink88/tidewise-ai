package report

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/jackc/pgx/v5/pgconn"
	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	reportapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/report"
	evidencebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/evidence"
	reportbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/report"
	evidencedata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/evidence"
	reportservice "github.com/meierlink88/tidewise-ai/data-service/backend/internal/service/report"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
	reportfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/report"
)

func TestPostgresReportPublicationReplayAndReadProjections(t *testing.T) {
	db := openReportTestDatabase(t, 0)
	evidenceIDs := publishReportEvidence(t, db)
	report := reportWithEvidenceIDs(t, agentOSFixtureReport(t), evidenceIDs[0], evidenceIDs[1])
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	publishedAt := time.Date(2026, 9, 2, 1, 2, 3, 456789000, time.UTC)
	useCase, err := reportbiz.NewUseCase(store, func() time.Time { return publishedAt })
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	first, err := useCase.Publish(ctx, "agentos-investment-run-001", report)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := useCase.Publish(ctx, "agentos-investment-run-001", report)
	if err != nil {
		t.Fatal(err)
	}
	if first.Replayed || !replay.Replayed || first.Record.ID != replay.Record.ID {
		t.Fatalf("first=%#v replay=%#v", first, replay)
	}

	var reportCount, linkCount int
	if err := db.QueryRow(`SELECT count(*) FROM reports`).Scan(&reportCount); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM report_evidence_links WHERE report_id=$1`, first.Record.ID).Scan(&linkCount); err != nil {
		t.Fatal(err)
	}
	if reportCount != 1 || linkCount != 5 {
		t.Fatalf("reports=%d links=%d", reportCount, linkCount)
	}

	stored, err := useCase.Get(ctx, first.Record.ID)
	if err != nil || stored.ContentHash != first.Record.ContentHash || len(stored.Report.IndustryChains) != 1 {
		t.Fatalf("stored=%#v err=%v", stored, err)
	}
	home, err := useCase.GetHome(ctx, first.Record.ID)
	if err != nil || home.Geopolitics == nil || home.Macroeconomics == nil || home.Report.IndustryChainCount != 1 {
		t.Fatalf("home=%#v err=%v", home, err)
	}
	if home.Geopolitics.Summary.EvidenceScopeToken == nil {
		t.Fatal("summary Evidence token was not projected")
	}

	_, layer, err := useCase.GetLayer(ctx, first.Record.ID, "geopolitics")
	if err != nil || len(layer.AffectedAnchors) != 1 || layer.AffectedAnchors[0].EvidenceScopeToken == nil {
		t.Fatalf("layer=%#v err=%v", layer, err)
	}
	_, chain, err := useCase.GetIndustryChain(ctx, first.Record.ID, "chain-01")
	if err != nil || len(chain.AffectedNodes) != 2 || chain.AffectedNodes[0].EvidenceScopeToken == nil || chain.AffectedNodes[1].EvidenceScopeToken != nil {
		t.Fatalf("chain=%#v err=%v", chain, err)
	}
	evidence, err := useCase.ListEvidence(ctx, first.Record.ID, *chain.AffectedNodes[0].EvidenceScopeToken)
	if err != nil || len(evidence) != 1 || evidence[0].Summary != "第一条报告依据" || !reflect.DeepEqual(evidence[0].Keywords, []string{"依据一"}) {
		t.Fatalf("evidence=%#v err=%v", evidence, err)
	}
	if len(evidence[0].SemanticTags) != 3 || evidence[0].SemanticTags[0].Text != "Example actor" || evidence[0].SemanticTags[1].Kind != "action" || evidence[0].SemanticTags[2].Text != "Report claim" {
		t.Fatalf("persisted semantic tags=%#v", evidence[0].SemanticTags)
	}
	_, err = useCase.ListEvidence(ctx, first.Record.ID, "RPE33333333-3333-4333-8333-333333333333")
	if !errors.Is(err, reportbiz.ErrEvidenceScopeNotFound) {
		t.Fatalf("unknown token error=%v", err)
	}

	changed := report
	changed.IndustryChains[0].Name = "变更后的产业链"
	_, err = useCase.Publish(ctx, "agentos-investment-run-001", changed)
	if !errors.Is(err, reportbiz.ErrPublicationConflict) {
		t.Fatalf("conflict error=%v", err)
	}
	missing := reportWithEvidenceIDs(t, agentOSFixtureReport(t), "EVD33333333-3333-4333-8333-333333333333", evidenceIDs[1])
	_, err = useCase.Publish(ctx, "agentos-report-missing", missing)
	var reference *reportbiz.ReferenceError
	if !errors.As(err, &reference) {
		t.Fatalf("missing Evidence error=%v", err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM reports`).Scan(&reportCount); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM report_evidence_links`).Scan(&linkCount); err != nil {
		t.Fatal(err)
	}
	if reportCount != 1 || linkCount != 5 {
		t.Fatalf("missing Evidence was not rolled back: reports=%d links=%d", reportCount, linkCount)
	}
	assertPostgresCode(t, db, "55000", `UPDATE reports SET content_hash=repeat('b',64) WHERE id=$1`, first.Record.ID)
}

func TestPostgresReportIndustryChainCursorPagesFiftyFourSummaries(t *testing.T) {
	db := openReportTestDatabase(t, 0)
	placeholderIDs := reportfixture.FrozenScaleBaselineEvidenceIDs()
	evidenceIDs := publishReportEvidenceCount(t, db, len(placeholderIDs))
	report := reportWithEvidenceIDMap(t, reportfixture.FrozenScaleBaselineReport(), placeholderIDs, evidenceIDs)
	store, _ := NewStore(db)
	useCase, _ := reportbiz.NewUseCase(store, func() time.Time { return time.Date(2026, 9, 2, 1, 2, 3, 0, time.UTC) })
	published, err := useCase.Publish(context.Background(), "agentos-report-54-chains", report)
	if err != nil {
		t.Fatal(err)
	}
	var linkCount int
	if err := db.QueryRow(`SELECT count(*) FROM report_evidence_links WHERE report_id=$1`, published.Record.ID).Scan(&linkCount); err != nil {
		t.Fatal(err)
	}
	if linkCount != 265 {
		t.Fatalf("links=%d want=265", linkCount)
	}
	request := reportbiz.IndustryChainListRequest{ReportID: published.Record.ID, Limit: 20}
	pageSizes := []int{}
	keys := []string{}
	uniqueKeys := map[string]struct{}{}
	for {
		page, err := useCase.ListIndustryChains(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		pageSizes = append(pageSizes, len(page.Items))
		for _, item := range page.Items {
			keys = append(keys, item.LocalKey)
			uniqueKeys[item.LocalKey] = struct{}{}
		}
		if page.NextCursor == nil {
			break
		}
		request.Cursor = *page.NextCursor
	}
	if !reflect.DeepEqual(pageSizes, []int{20, 20, 14}) || len(keys) != 54 || len(uniqueKeys) != 54 || keys[0] != "chain-01" || keys[53] != "chain-54" {
		t.Fatalf("sizes=%v keys=%d first=%q last=%q", pageSizes, len(keys), keys[0], keys[len(keys)-1])
	}
}

func TestMigration81CutsEmptyReportStoreToFinalShape(t *testing.T) {
	db := openReportTestDatabase(t, 80)
	postgresfixture.ApplyMigration(t, db, reportMigrationDir(t), 81)
	if got := tableColumns(t, db, "reports"); !reflect.DeepEqual(got, []string{"id", "publisher_report_id", "content_hash", "report", "published_at"}) {
		t.Fatalf("reports columns=%#v", got)
	}
	if got := tableColumns(t, db, "report_evidence_links"); !reflect.DeepEqual(got, []string{"id", "report_id", "evidence_id", "scope_type", "scope_path", "position"}) {
		t.Fatalf("links columns=%#v", got)
	}
}

func TestMigration81RefusesLossyCutover(t *testing.T) {
	db := openReportTestDatabase(t, 80)
	if _, err := db.Exec(`INSERT INTO reports (id,publisher_report_id,contract_version,content_hash,content,published_at)
        VALUES('RPT55555555-5555-5555-8555-555555555555','publisher','report-publication.v2',repeat('a',64),'{}',now())`); err != nil {
		t.Fatal(err)
	}
	assertPostgresCode(t, db, "55000", string(mustMigrationUp(t, filepath.Join(reportMigrationDir(t), "000081_finalize_report_publication_contract.sql"))))
}

func TestMigration81DownIsForwardOnly(t *testing.T) {
	payload, err := os.ReadFile(filepath.Join(reportMigrationDir(t), "000081_finalize_report_publication_contract.sql"))
	if err != nil {
		t.Fatal(err)
	}
	down := strings.Split(string(payload), "-- +goose Down")
	if len(down) != 2 || !strings.Contains(down[1], "ERRCODE = '55000'") || !strings.Contains(down[1], "forward-only") {
		t.Fatalf("invalid Down section: %q", down)
	}
}

func publishReportEvidence(t *testing.T, db *sql.DB) []string {
	return publishReportEvidenceCount(t, db, 2)
}

func publishReportEvidenceCount(t *testing.T, db *sql.DB, count int) []string {
	t.Helper()
	store, err := evidencedata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := evidencebiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	publishedAt := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	raw, err := useCase.PublishRawEvidence(context.Background(), evidencebiz.RawEvidence{
		PublicationKey: "report-data-test-evidence", SourceID: "SRC_report_data_test", SourceName: "Example Wire",
		SourceLevel: evidencebiz.SourceLevelWire, SourceURL: "https://example.test/report", IsOriginal: true,
		RawText: "Two persisted facts support the test Report.", PublishedAt: &publishedAt, CollectedAt: publishedAt.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	items := make([]evidencebiz.Evidence, count)
	for index := range items {
		summary := fmt.Sprintf("第 %d 条报告依据", index+1)
		keyword := fmt.Sprintf("依据%d", index+1)
		if index == 0 {
			summary, keyword = "第一条报告依据", "依据一"
		} else if index == 1 {
			summary, keyword = "第二条报告依据", "依据二"
		}
		items[index] = reportEvidence(
			summary,
			keyword,
			fmt.Sprintf("supports report claim %d", index+1),
		)
	}
	result, err := useCase.PublishEvidence(context.Background(), raw.ID, items)
	if err != nil {
		t.Fatal(err)
	}
	evidenceIDs := make([]string, len(result.Items))
	for _, item := range result.Items {
		if item.InputIndex < 0 || item.InputIndex >= len(evidenceIDs) {
			t.Fatalf("Evidence result input_index=%d out of range", item.InputIndex)
		}
		evidenceIDs[item.InputIndex] = item.ID
	}
	for index, id := range evidenceIDs {
		if id == "" {
			t.Fatalf("Evidence result is missing input_index=%d", index)
		}
	}
	return evidenceIDs
}

func reportWithEvidenceIDMap(t *testing.T, source reportbiz.Report, placeholders, replacements []string) reportbiz.Report {
	t.Helper()
	if len(placeholders) != len(replacements) {
		t.Fatalf("placeholder count=%d replacement count=%d", len(placeholders), len(replacements))
	}
	payload, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	replaced := string(payload)
	for index := range placeholders {
		replaced = strings.ReplaceAll(replaced, placeholders[index], replacements[index])
	}
	var report reportbiz.Report
	if err := json.Unmarshal([]byte(replaced), &report); err != nil {
		t.Fatal(err)
	}
	return report
}

func reportEvidence(summary, keyword, action string) evidencebiz.Evidence {
	return evidencebiz.Evidence{Summary: summary, Keywords: []string{keyword}, Semantic: evidencebiz.Semantic{
		Actors: []string{"Example actor"}, Action: action, Objects: []string{"Report claim"},
		Stage: evidencebiz.EvidenceStageOccurred, Modality: evidencebiz.EvidenceModalityFact,
		Time: evidencebiz.EvidenceTime{Precision: evidencebiz.EvidenceTimeUnknown}, Jurisdictions: []string{},
		Metrics: []evidencebiz.EvidenceMetric{}, Attribution: &evidencebiz.EvidenceAttribution{},
	}}
}

func reportWithEvidenceIDs(t *testing.T, source reportbiz.Report, first, second string) reportbiz.Report {
	t.Helper()
	payload, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	replaced := strings.ReplaceAll(string(payload), reportfixture.EvidenceOne, first)
	replaced = strings.ReplaceAll(replaced, reportfixture.EvidenceTwo, second)
	var report reportbiz.Report
	if err := json.Unmarshal([]byte(replaced), &report); err != nil {
		t.Fatal(err)
	}
	return report
}

func agentOSFixtureReport(t *testing.T) reportbiz.Report {
	t.Helper()
	payload, err := os.ReadFile(filepath.Join("..", "..", "..", "api", "data", "v1", "report", "testdata", "investment-report-publication-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	var request struct {
		Report reportbiz.Report `json:"report"`
	}
	if err := json.Unmarshal(payload, &request); err != nil {
		t.Fatal(err)
	}
	return request.Report
}

func openReportTestDatabase(t *testing.T, version int64) *sql.DB {
	t.Helper()
	return postgresfixture.OpenIsolated(t, "tw_report", reportMigrationDir(t), version)
}

func reportMigrationDir(t *testing.T) string {
	t.Helper()
	directory, err := filepath.Abs(filepath.Join("..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return directory
}

func mustMigrationUp(t *testing.T, path string) []byte {
	t.Helper()
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(string(payload), "-- +goose Down")
	if len(parts) != 2 {
		t.Fatal("migration has no Down marker")
	}
	return []byte(strings.TrimPrefix(parts[0], "-- +goose Up"))
}

func tableColumns(t *testing.T, db *sql.DB, table string) []string {
	t.Helper()
	rows, err := db.Query(`SELECT column_name FROM information_schema.columns WHERE table_schema=current_schema() AND table_name=$1 ORDER BY ordinal_position`, table)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	columns := []string{}
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			t.Fatal(err)
		}
		columns = append(columns, value)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return columns
}

func assertPostgresCode(t *testing.T, db *sql.DB, code, statement string, args ...any) {
	t.Helper()
	_, err := db.Exec(statement, args...)
	var pgError *pgconn.PgError
	if !errors.As(err, &pgError) || pgError.Code != code {
		t.Fatalf("error=%T %v want PostgreSQL code %s", err, err, code)
	}
}

func TestPostgresStoryConceptPublicationAndScopedReads(t *testing.T) {
	db := openReportTestDatabase(t, 0)
	ids := publishReportEvidence(t, db)
	payload, err := os.ReadFile("../../../api/data/v1/report/testdata/story-concept-publication-request.json")
	if err != nil {
		t.Fatal(err)
	}
	payload = []byte(strings.ReplaceAll(string(payload), "EVD11111111-1111-4111-8111-111111111111", ids[0]))
	var request struct {
		Report reportbiz.Report `json:"report"`
	}
	if err := json.Unmarshal(payload, &request); err != nil {
		t.Fatal(err)
	}
	store, _ := NewStore(db)
	uc, _ := reportbiz.NewUseCase(store, time.Now)
	ctx := context.Background()
	first, err := uc.Publish(ctx, "story-concept", request.Report)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := uc.Publish(ctx, "story-concept", request.Report)
	if err != nil || !replay.Replayed || replay.Record.ID != first.Record.ID {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}

	app, _ := reportservice.NewService(uc)
	httpServer := kratoshttp.NewServer()
	reportapi.RegisterHTTPServer(httpServer, app)
	wire, _ := json.Marshal(map[string]any{"publisher_report_id": "story-concept", "report": request.Report})
	httpRequest := httptest.NewRequest(http.MethodPost, "/api/data/v1/report-publications", bytes.NewReader(wire))
	httpRequest.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	httpServer.ServeHTTP(response, httpRequest)
	if response.Code != http.StatusOK {
		t.Fatalf("HTTP replay status=%d body=%s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	httpServer.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/data/v1/reports/"+first.Record.ID+"/analyses/concept_analyses", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "concept-a") || strings.Contains(response.Body.String(), "EVD") {
		t.Fatalf("HTTP projection status=%d body=%s", response.Code, response.Body.String())
	}
	stored, err := uc.Get(ctx, first.Record.ID)
	if err != nil || stored.ContentHash != first.Record.ContentHash {
		t.Fatalf("stored hash error %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM report_evidence_links WHERE report_id=$1`, first.Record.ID).Scan(&count); err != nil || count != 14 {
		t.Fatalf("links=%d err=%v", count, err)
	}
	page, err := uc.ListAnalyses(ctx, reportbiz.AnalysisListRequest{ReportID: first.Record.ID, Kind: "geopolitical_stories", Limit: 1})
	if err != nil || len(page.Items) != 1 || page.NextCursor == nil {
		t.Fatalf("page=%+v err=%v", page, err)
	}
	next, err := uc.ListAnalyses(ctx, reportbiz.AnalysisListRequest{ReportID: first.Record.ID, Kind: "geopolitical_stories", Limit: 1, Cursor: *page.NextCursor})
	if err != nil || len(next.Items) != 1 || next.Items[0].LocalKey != "geo-b" || next.NextCursor != nil {
		t.Fatalf("next=%+v err=%v", next, err)
	}
	for _, q := range []reportbiz.AnalysisListRequest{{ReportID: first.Record.ID, Kind: "concept_analyses", Cursor: *page.NextCursor}, {ReportID: "RPT22222222-2222-4222-8222-222222222222", Kind: "geopolitical_stories", Cursor: *page.NextCursor}} {
		if _, err := uc.ListAnalyses(ctx, q); err == nil {
			t.Fatal("accepted wrong cursor scope")
		}
	}
	concept, err := uc.GetAnalysis(ctx, first.Record.ID, "concept_analyses", "concept-a")
	if err != nil || len(concept.IndustryChains) != 2 || len(concept.Summary.AffectedAnchors) != 2 {
		t.Fatalf("concept=%+v err=%v", concept, err)
	}
	chain, err := uc.GetAnalysisChain(ctx, first.Record.ID, "concept_analyses", "concept-a", "chain-a")
	if err != nil || len(chain.AffectedNodes) != 1 || chain.AffectedNodes[0].EvidenceScopeToken == nil {
		t.Fatalf("chain=%+v err=%v", chain, err)
	}
	evidence, err := uc.ListEvidence(ctx, first.Record.ID, *chain.AffectedNodes[0].EvidenceScopeToken)
	if err != nil || len(evidence) != 1 {
		t.Fatalf("evidence=%+v err=%v", evidence, err)
	}
	encoded, _ := json.Marshal(concept)
	if strings.Contains(string(encoded), "EVD") || strings.Contains(string(encoded), "evidence_refs") || strings.Contains(string(encoded), "node_local_key") == false {
		t.Fatalf("unsafe/incomplete projection: %s", encoded)
	}
	_, err = uc.GetAnalysisChain(ctx, first.Record.ID, "concept_analyses", "wrong-concept", "chain-a")
	if !errors.Is(err, reportbiz.ErrChainNotFound) {
		t.Fatalf("cross Concept chain: %v", err)
	}
	legacy := reportWithEvidenceIDs(t, agentOSFixtureReport(t), ids[0], ids[1])
	old, err := uc.Publish(ctx, "legacy-alongside-v3", legacy)
	if err != nil {
		t.Fatal(err)
	}
	list, err := uc.List(ctx, reportbiz.ListRequest{SchemaVersion: "legacy"})
	if err != nil || len(list.Items) != 1 || list.Items[0].ID != old.Record.ID {
		t.Fatalf("legacy selection=%+v err=%v", list, err)
	}
	list, err = uc.List(ctx, reportbiz.ListRequest{SchemaVersion: reportbiz.AnalysisSchemaVersion})
	if err != nil || len(list.Items) != 1 || list.Items[0].IndustryChainCount != 2 {
		t.Fatalf("v3 selection=%+v err=%v", list, err)
	}
	_, err = uc.GetHome(ctx, old.Record.ID)
	if err != nil {
		t.Fatal(err)
	}
	request.Report.GeopoliticalStories[0].Summary.Conclusion = "不同结论"
	if _, err := uc.Publish(ctx, "story-concept", request.Report); !errors.Is(err, reportbiz.ErrPublicationConflict) {
		t.Fatalf("conflict=%v", err)
	}
	request.Report.GeopoliticalStories[0].Summary.EvidenceRefs[0].EvidenceID = "EVD33333333-3333-4333-8333-333333333333"
	if _, err := uc.Publish(ctx, "missing-analysis-evidence", request.Report); err == nil {
		t.Fatal("missing Evidence accepted")
	}
	if err := db.QueryRow(`SELECT count(*) FROM reports`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("atomicity count=%d err=%v", count, err)
	}
	request.Report.GeopoliticalStories[0].Summary.EvidenceRefs[0].EvidenceID = ids[0]
	request.Report.ConceptAnalyses = []reportbiz.AnalysisUnit{}
	onlyStory, err := uc.Publish(ctx, "story-only", request.Report)
	if err != nil {
		t.Fatal(err)
	}
	list, err = uc.List(ctx, reportbiz.ListRequest{SchemaVersion: reportbiz.AnalysisSchemaVersion})
	if err != nil || len(list.Items) != 2 {
		t.Fatalf("story only list %v", err)
	}
	empty, err := uc.ListAnalyses(ctx, reportbiz.AnalysisListRequest{ReportID: onlyStory.Record.ID, Kind: "concept_analyses"})
	if err != nil || len(empty.Items) != 0 {
		t.Fatalf("empty group %v", err)
	}
}

func TestPostgresStoryChainHTTPPublicationAndRead(t *testing.T) {
	db := openReportTestDatabase(t, 0)
	ids := publishReportEvidence(t, db)
	payload, err := os.ReadFile("../../../api/data/v1/report/testdata/story-chain-publication-request.json")
	if err != nil {
		t.Fatal(err)
	}
	payload = bytes.ReplaceAll(payload, []byte("EVD11111111-1111-4111-8111-111111111111"), []byte(ids[0]))
	store, _ := NewStore(db)
	uc, _ := reportbiz.NewUseCase(store, time.Now)
	app, _ := reportservice.NewService(uc)
	server := kratoshttp.NewServer()
	reportapi.RegisterHTTPServer(server, app)
	call := func(method, path string, body []byte) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.ServeHTTP(w, r)
		return w
	}
	for i := 0; i < 2; i++ {
		w := call(http.MethodPost, "/api/data/v1/report-publications", payload)
		want := http.StatusCreated
		if i == 1 {
			want = http.StatusOK
		}
		if w.Code != want {
			t.Fatalf("publish/replay: %d %s", w.Code, w.Body.String())
		}
	}
	var id string
	if err := db.QueryRow(`SELECT id FROM reports WHERE publisher_report_id='story-chain-example'`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	prefix := "/api/data/v1/reports/" + id
	for _, x := range []struct{ kind, key string }{{"geopolitical_stories", "geo"}, {"macroeconomic_stories", "macro"}} {
		path := prefix + "/analyses/" + x.kind + "/" + x.key + "-story"
		w := call(http.MethodGet, path, nil)
		if w.Code != 200 {
			t.Fatalf("unit: %d %s", w.Code, w.Body.String())
		}
		var unit reportapi.AnalysisUnitDetail
		if err := json.Unmarshal(w.Body.Bytes(), &unit); err != nil {
			t.Fatal(err)
		}
		if len(unit.IndustryChains) != 1 || len(unit.Summary.AffectedAnchors) != 1 || unit.Summary.AffectedAnchors[0].TargetType.Code != "industry_chain" || strings.Contains(w.Body.String(), `"graph"`) {
			t.Fatalf("unit projection: %s", w.Body.String())
		}
		w = call(http.MethodGet, path+"/industry-chains/"+x.key+"-chain-a", nil)
		if w.Code != 200 {
			t.Fatalf("chain: %d %s", w.Code, w.Body.String())
		}
		var chain reportapi.ChainAnalysisDetail
		if err := json.Unmarshal(w.Body.Bytes(), &chain); err != nil {
			t.Fatal(err)
		}
		c := chain
		if len(c.Graph.Nodes) != 2 || len(c.Graph.Edges) != 1 || len(c.AffectedNodes) != 1 || len(c.ReasoningSteps) != 1 || c.TransmissionLogic != "供给变化 → 服务成本变化" {
			t.Fatalf("incomplete chain: %s", w.Body.String())
		}
		for _, token := range []*string{c.EvidenceScopeToken, c.AffectedNodes[0].EvidenceScopeToken, c.ReasoningSteps[0].EvidenceScopeToken} {
			if token == nil {
				t.Fatal("missing Evidence scope")
			}
			evidence, err := uc.ListEvidence(context.Background(), id, *token)
			if err != nil || len(evidence) != 1 || evidence[0].Summary == "" {
				t.Fatalf("evidence=%+v err=%v", evidence, err)
			}
		}
		// Same chain identity in another story is a separate scoped snapshot.
		_, err := uc.GetAnalysisChain(context.Background(), id, x.kind, x.key+"-story", "chain-a")
		if !errors.Is(err, reportbiz.ErrChainNotFound) {
			t.Fatalf("cross-unit lookup: %v", err)
		}
	}
	for _, path := range []string{prefix + "/concept-analyses/concept-a/industry-chains/chain-a", prefix + "/analyses/concept_analyses/concept-a/industry-chains/chain-a"} {
		w := call(http.MethodGet, path, nil)
		if w.Code != 200 {
			t.Fatalf("Concept compatibility: %d %s", w.Code, w.Body.String())
		}
	}
	_, err = uc.GetAnalysisChain(context.Background(), id, "invalid", "geo-story", "geo-chain-a")
	var validation *reportbiz.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("invalid kind: %v", err)
	}
	for _, x := range []struct{ kind, key, level string }{{"geopolitical_stories", "geo-story", "high"}, {"macroeconomic_stories", "macro-story", "medium"}, {"concept_analyses", "concept-a", "low"}} {
		detail, err := uc.GetAnalysis(context.Background(), id, x.kind, x.key)
		if err != nil {
			t.Fatal(err)
		}
		a := detail.Summary.ImpactAssessment
		if a == nil || a.Level.Code != x.level || a.Rationale == "" || a.EvidenceScopeToken == nil {
			t.Fatalf("impact missing: %+v", a)
		}
		ev, err := uc.ListEvidence(context.Background(), id, *a.EvidenceScopeToken)
		if err != nil || len(ev) != 1 {
			t.Fatalf("impact evidence: %+v %v", ev, err)
		}
		page, err := uc.ListAnalyses(context.Background(), reportbiz.AnalysisListRequest{ReportID: id, Kind: x.kind})
		if err != nil || len(page.Items) != 1 || !reflect.DeepEqual(page.Items[0].ImpactAssessment, a) {
			t.Fatalf("list impact: %+v %v", page, err)
		}
		w := call(http.MethodGet, prefix+"/analyses/"+x.kind+"/"+x.key, nil)
		var wire reportapi.AnalysisUnitDetail
		if err := json.Unmarshal(w.Body.Bytes(), &wire); err != nil {
			t.Fatal(err)
		}
		if w.Code != 200 || wire.Summary.ImpactAssessment == nil || wire.Summary.ImpactAssessment.Level.Code != x.level || strings.Contains(w.Body.String(), "EVD") || strings.Contains(w.Body.String(), "evidence_refs") {
			t.Fatalf("impact wire: %d %s", w.Code, w.Body.String())
		}
	}

	var request struct {
		PublisherReportID string           `json:"publisher_report_id"`
		Report            reportbiz.Report `json:"report"`
	}
	if err := json.Unmarshal(payload, &request); err != nil {
		t.Fatal(err)
	}
	request.PublisherReportID = "missing-story-node-evidence"
	request.Report.MacroeconomicStories[0].Detail.IndustryChains[0].AffectedNodes[0].EvidenceRefs[0].EvidenceID = "EVD33333333-3333-4333-8333-333333333333"
	_, err = uc.Publish(context.Background(), request.PublisherReportID, request.Report)
	var reference *reportbiz.ReferenceError
	if !errors.As(err, &reference) {
		t.Fatalf("missing nested Evidence: %v", err)
	}

	request.Report.MacroeconomicStories[0].Detail.IndustryChains[0].AffectedNodes[0].EvidenceRefs[0].EvidenceID = ids[0]
	request.PublisherReportID = "missing-impact-evidence"
	request.Report.MacroeconomicStories[0].Summary.ImpactAssessment.EvidenceRefs[0].EvidenceID = "EVD33333333-3333-4333-8333-333333333333"
	_, err = uc.Publish(context.Background(), request.PublisherReportID, request.Report)
	if !errors.As(err, &reference) {
		t.Fatalf("missing impact Evidence accepted: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM reports`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("atomic report count=%d err=%v", count, err)
	}
}

func TestPostgresNormalizedReportHTTPRoundTrip(t *testing.T) { testNormalizedHTTPRoundTrip(t, false) }
func TestPostgresSignalReportHTTPRoundTrip(t *testing.T)     { testNormalizedHTTPRoundTrip(t, true) }
func testNormalizedHTTPRoundTrip(t *testing.T, signals bool) {
	db := openReportTestDatabase(t, 0)
	ids := publishReportEvidence(t, db)
	fixture := "normalized-publication-request.json"
	publisher := "normalized-contract-example"
	if signals {
		fixture = "signal-publication-request.json"
		publisher = "synthetic-signal-report-v5"
	}
	payload, err := os.ReadFile("../../../api/data/v1/report/testdata/" + fixture)
	if err != nil {
		t.Fatal(err)
	}
	payload = bytes.ReplaceAll(payload, []byte("EVD11111111-1111-4111-8111-111111111111"), []byte(ids[0]))
	store, _ := NewStore(db)
	publicationTime := time.Now()
	uc, _ := reportbiz.NewUseCase(store, func() time.Time { return publicationTime })
	app, _ := reportservice.NewService(uc)
	server := kratoshttp.NewServer(kratoshttp.ErrorEncoder(func(w http.ResponseWriter, r *http.Request, err error) {
		var p *v1.PublicError
		if errors.As(err, &p) {
			w.WriteHeader(p.Status)
			json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": p.Code, "message": p.Message}})
			return
		}
		w.WriteHeader(500)
	}))
	reportapi.RegisterHTTPServer(server, app)
	document, err := openapi3.NewLoader().LoadFromFile("../../../api/data/v1/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	call := func(method, path string, body []byte) *httptest.ResponseRecorder {
		t.Helper()
		q := httptest.NewRequest(method, path, bytes.NewReader(body))
		q.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.ServeHTTP(w, q)
		if signals && method == "GET" && w.Code == 200 {
			var value map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &value); err != nil {
				t.Fatal(err)
			}
			schema := "SignalDetailProjection"
			switch {
			case strings.Contains(path, "/home"):
				schema = "SignalHomeProjection"
			case strings.Contains(path, "/industry-chains/"):
				schema = "SignalChainRead"
			case strings.Contains(path, "/company_analyses/"):
				schema = "SignalCompanyProjection"
			}
			if items, ok := value["items"].([]any); ok {
				for _, item := range items {
					n := "SignalSummaryProjection"
					if strings.Contains(path, "company_analyses") {
						n = "SignalCompanySummary"
					}
					if err := document.Components.Schemas[n].Value.VisitJSON(item); err != nil {
						t.Fatalf("%s: %v", path, err)
					}
				}
			} else if err := document.Components.Schemas[schema].Value.VisitJSON(value); err != nil {
				t.Fatalf("%s: %v", path, err)
			}
		}
		return w
	}
	for _, want := range []int{201, 200} {
		w := call("POST", "/api/data/v1/report-publications", payload)
		if w.Code != want {
			t.Fatalf("publish %d %s", w.Code, w.Body.String())
		}
	}
	var id string
	if err := db.QueryRow(`SELECT id FROM reports WHERE publisher_report_id=$1`, publisher).Scan(&id); err != nil {
		t.Fatal(err)
	}
	var storedCounts []byte
	if err := db.QueryRow(`SELECT evidence_counts FROM reports WHERE id=$1`, id).Scan(&storedCounts); err != nil {
		t.Fatal(err)
	}
	var counts map[string]int
	if err := json.Unmarshal(storedCounts, &counts); err != nil || len(counts) == 0 {
		t.Fatalf("missing publication counts: %s %v", storedCounts, err)
	}
	for _, count := range counts {
		if count != 1 {
			t.Fatalf("scope must not aggregate shared evidence across objects: %d", count)
		}
	}

	var req struct {
		Report reportbiz.Report `json:"report"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatal(err)
	}
	var checkCounts func(any)
	checkCounts = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			if token, exists := x["evidence_scope_token"]; exists {
				want := 0
				if token != nil {
					items, err := uc.ListEvidence(context.Background(), id, token.(string))
					if err != nil {
						t.Fatal(err)
					}
					want = len(items)
				}
				if x["evidence_count"] != float64(want) {
					t.Fatalf("count/list mismatch: %+v want %d", x, want)
				}
			}
			for _, v := range x {
				checkCounts(v)
			}
		case []any:
			for _, v := range x {
				checkCounts(v)
			}
		}
	}

	root := "/api/data/v1/reports/" + id
	groups := []struct {
		kind  string
		units []reportbiz.V4Unit
	}{{"geopolitical_stories", req.Report.V4.GeopoliticalStories}, {"macroeconomic_stories", req.Report.V4.MacroeconomicStories}, {"concept_analyses", req.Report.V4.ConceptAnalyses}}
	if signals {
		groups = append(groups, struct {
			kind  string
			units []reportbiz.V4Unit
		}{"industry_chain_analyses", *req.Report.V4.IndustryChainAnalyses})
	}
	for _, g := range groups {
		w := call("GET", root+"/analyses/"+g.kind+"?limit=1", nil)
		if w.Code != 200 || strings.Contains(w.Body.String(), `"evidence_ids"`) {
			t.Fatalf("list %d %s", w.Code, w.Body.String())
		}
		var page reportapi.AnalysisCollection
		if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(w.Body.String(), "variable_signals") {
			t.Fatal("signals leaked into summary")
		}
		if len(page.Items) != 1 || page.Items[0].V4 == nil {
			t.Fatal("missing normalized summary")
		}
		for _, u := range g.units {
			prefix := root + "/analyses/" + g.kind + "/" + u.LocalKey
			w := call("GET", prefix, nil)
			if w.Code != 200 {
				t.Fatal(w.Body.String())
			}
			var detail reportapi.AnalysisUnitDetail
			if err := json.Unmarshal(w.Body.Bytes(), &detail); err != nil {
				t.Fatal(err)
			}
			if detail.V4 == nil || detail.V4.Summary.Summary.Conclusion != u.Summary.Conclusion || strings.Contains(w.Body.String(), `"graph"`) {
				t.Fatal("invalid unit header projection")
			}
			if signals {
				expected := map[string]any{"variable_signals": u.Detail.VariableSignals, "companies": u.Detail.Companies, "macro_impacts": u.Detail.MacroImpacts, "reasoning_sources": u.ReasoningSources, "judgment_origin": u.JudgmentOrigin}
				var actual map[string]any
				json.Unmarshal(w.Body.Bytes(), &actual)
				checkCounts(actual)
				for k, v := range expected {
					raw, _ := json.Marshal(v)
					var want any
					json.Unmarshal(raw, &want)
					if !reflect.DeepEqual(withoutEvidenceFields(want), withoutEvidenceFields(actual[k])) {
						t.Fatalf("unit lost %s", k)
					}
				}
			}
			for _, c := range u.Detail.IndustryChains {
				w := call("GET", prefix+"/industry-chains/"+c.LocalKey, nil)
				if w.Code != 200 {
					t.Fatalf("chain %d %s", w.Code, w.Body.String())
				}
				var chain reportapi.ChainAnalysisDetail
				if err := json.Unmarshal(w.Body.Bytes(), &chain); err != nil {
					t.Fatal(err)
				}
				if chain.V4 == nil || chain.V4.Assessment.Conclusion != c.Assessment.Conclusion || len(chain.V4.AffectedNodes) != len(c.AffectedNodes) {
					t.Fatal("chain fields missing")
				}
				// Compare every non-Evidence field, not just selected display columns.
				expectedBytes, _ := json.Marshal(c)
				var expected, actual any
				json.Unmarshal(expectedBytes, &expected)
				json.Unmarshal(w.Body.Bytes(), &actual)
				checkCounts(actual)
				if !reflect.DeepEqual(withoutEvidenceFields(expected), withoutEvidenceFields(actual)) {
					t.Fatal("chain round trip lost or changed fields")
				}
				if strings.Contains(w.Body.String(), "EVD") || strings.Contains(w.Body.String(), "evidence_ids") {
					t.Fatal("raw Evidence leaked")
				}
				token := chain.V4.ReasoningSummary.Support.EvidenceScopeToken
				if token == nil {
					t.Fatal("missing support token")
				}
				ev, err := uc.ListEvidence(context.Background(), id, *token)
				if err != nil || len(ev) != 1 {
					t.Fatalf("evidence %v", err)
				}
				for i, n := range chain.V4.AffectedNodes {
					if n.Assessment.Conclusion != c.AffectedNodes[i].Assessment.Conclusion || n.Assessment.ForecastWindow.Description != c.AffectedNodes[i].Assessment.ForecastWindow.Description {
						t.Fatal("node fields changed")
					}
				}
			}
		}
	}
	if signals {
		for _, co := range *req.Report.V4.CompanyAnalyses {
			w := call("GET", root+"/analyses/company_analyses/"+co.LocalKey, nil)
			if w.Code != 200 {
				t.Fatalf("company: %d %s", w.Code, w.Body.String())
			}
			var actual map[string]any
			json.Unmarshal(w.Body.Bytes(), &actual)
			checkCounts(actual)
			raw, _ := json.Marshal(co)
			var want any
			json.Unmarshal(raw, &want)
			if !reflect.DeepEqual(withoutEvidenceFields(want), withoutEvidenceFields(actual["company"])) {
				t.Fatal("company fields changed")
			}
		}
		w := call("GET", root+"/analyses/company_analyses?limit=1", nil)
		if w.Code != 200 || strings.Contains(w.Body.String(), "variable_signals") {
			t.Fatalf("company summary %d %s", w.Code, w.Body.String())
		}
	}
	w := call("GET", root+"/home", nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"observations"`) || strings.Contains(w.Body.String(), "evidence_ids") {
		t.Fatal("missing normalized metadata")
	}
	// A newer normalized report must be visible without selecting a format.
	page, err := uc.List(context.Background(), reportbiz.ListRequest{})
	if err != nil || len(page.Items) != 1 || page.Items[0].ID != id {
		t.Fatalf("all-version selection: %+v %v", page, err)
	}
	changed := bytes.ReplaceAll(payload, []byte(req.Report.V4.GeopoliticalStories[0].Summary.Conclusion), []byte("Different conclusion"))
	if w := call("POST", "/api/data/v1/report-publications", changed); w.Code != 409 {
		t.Fatalf("changed replay: %d", w.Code)
	}
	// Missing/unknown/null fields fail before storage; missing Evidence rolls back.
	for _, change := range []func(map[string]any){func(v map[string]any) { v["unknown"] = true }, func(v map[string]any) { delete(v["report"].(map[string]any), "observations") }, func(v map[string]any) { v["report"].(map[string]any)["limitations"] = nil }} {
		var v map[string]any
		json.Unmarshal(payload, &v)
		change(v)
		b, _ := json.Marshal(v)
		if w := call("POST", "/api/data/v1/report-publications", b); w.Code != 400 {
			t.Fatalf("invalid shape accepted: %d %s", w.Code, w.Body.String())
		}
	}
	missing := bytes.ReplaceAll(payload, []byte(ids[0]), []byte("EVD33333333-3333-4333-8333-333333333333"))
	missing = bytes.ReplaceAll(missing, []byte(publisher), []byte("normalized-missing"))
	if w := call("POST", "/api/data/v1/report-publications", missing); w.Code != 422 {
		t.Fatalf("missing Evidence: %d %s", w.Code, w.Body.String())
	}
	var count int
	db.QueryRow(`SELECT count(*) FROM reports`).Scan(&count)
	if count != 1 {
		t.Fatal("publication was not atomic")
	}
	publicationTime = publicationTime.Add(-24 * time.Hour)
	legacy := reportWithEvidenceIDs(t, agentOSFixtureReport(t), ids[0], ids[1])
	old, err := uc.Publish(context.Background(), "old-format", legacy)
	if err != nil {
		t.Fatal(err)
	}
	v3bytes, err := os.ReadFile("../../../api/data/v1/report/testdata/story-concept-publication-request.json")
	if err != nil {
		t.Fatal(err)
	}
	v3bytes = bytes.ReplaceAll(v3bytes, []byte("EVD11111111-1111-4111-8111-111111111111"), []byte(ids[0]))
	var v3req struct {
		Report reportbiz.Report `json:"report"`
	}
	if err := json.Unmarshal(v3bytes, &v3req); err != nil {
		t.Fatal(err)
	}
	third, err := uc.Publish(context.Background(), "v3-format", v3req.Report)
	if err != nil {
		t.Fatal(err)
	}
	page, err = uc.List(context.Background(), reportbiz.ListRequest{Limit: 1})
	if err != nil || len(page.Items) != 1 || page.Items[0].ID != id || page.NextCursor == nil {
		t.Fatalf("mixed latest selection: %+v %v", page, err)
	}
	if _, err := uc.List(context.Background(), reportbiz.ListRequest{SchemaVersion: "legacy", Cursor: *page.NextCursor}); err == nil {
		t.Fatal("accepted cursor with changed version filter")
	}
	for version, want := range map[string]string{"legacy": old.Record.ID, reportbiz.AnalysisSchemaVersion: third.Record.ID, req.Report.SchemaVersion: id} {
		result, err := uc.List(context.Background(), reportbiz.ListRequest{SchemaVersion: version})
		if err != nil || len(result.Items) != 1 || result.Items[0].ID != want {
			t.Fatalf("version %s: %+v %v", version, result, err)
		}
	}
	// Simulate a pre-metadata immutable publication without modifying the original row.
	historicalID := "RPTaaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	if _, err := db.Exec(`INSERT INTO reports(id,publisher_report_id,content_hash,report,published_at)
 SELECT $2,'historical-count-fallback',content_hash,report,published_at - interval '1 day' FROM reports WHERE id=$1`, id, historicalID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO report_evidence_links(id,report_id,evidence_id,scope_type,scope_path,position)
 SELECT 'RPE'||gen_random_uuid()::text,$2,evidence_id,scope_type,scope_path,position FROM report_evidence_links WHERE report_id=$1`, id, historicalID); err != nil {
		t.Fatal(err)
	}
	historical, err := uc.ListAnalyses(context.Background(), reportbiz.AnalysisListRequest{ReportID: historicalID, Kind: "geopolitical_stories"})
	if err != nil || historical.Items[0].V4.Summary.EvidenceCount != 1 {
		t.Fatalf("historical count fallback: %+v %v", historical, err)
	}

}

func withoutEvidenceFields(v any) any {
	switch x := v.(type) {
	case map[string]any:
		delete(x, "evidence_ids")
		delete(x, "evidence_scope_token")
		delete(x, "evidence_count")
		for k, child := range x {
			x[k] = withoutEvidenceFields(child)
		}
	case []any:
		for i, child := range x {
			x[i] = withoutEvidenceFields(child)
		}
	}
	return v
}
