package report

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	evidencebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/evidence"
	reportbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/report"
	evidencedata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/evidence"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
	reportfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/report"
)

func TestPostgresReportPublicationReplayAndReadProjections(t *testing.T) {
	db := openReportTestDatabase(t, 0)
	evidenceIDs := publishReportEvidence(t, db)
	report := reportWithEvidenceIDs(t, reportfixture.Report(), evidenceIDs[0], evidenceIDs[1])
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

	first, err := useCase.Publish(ctx, "agentos-report-2026-09-02", report)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := useCase.Publish(ctx, "agentos-report-2026-09-02", report)
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
	if reportCount != 1 || linkCount != 6 {
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
	if err != nil || len(chain.AffectedNodes) != 1 || chain.AffectedNodes[0].EvidenceScopeToken == nil {
		t.Fatalf("chain=%#v err=%v", chain, err)
	}
	evidence, err := useCase.ListEvidence(ctx, first.Record.ID, *chain.AffectedNodes[0].EvidenceScopeToken)
	if err != nil || len(evidence) != 1 || evidence[0].Summary != "第二条报告依据" || !reflect.DeepEqual(evidence[0].Keywords, []string{"依据二"}) {
		t.Fatalf("evidence=%#v err=%v", evidence, err)
	}
	_, err = useCase.ListEvidence(ctx, first.Record.ID, "RPE33333333-3333-4333-8333-333333333333")
	if !errors.Is(err, reportbiz.ErrEvidenceScopeNotFound) {
		t.Fatalf("unknown token error=%v", err)
	}

	changed := report
	changed.IndustryChains[0].Name = "变更后的产业链"
	_, err = useCase.Publish(ctx, "agentos-report-2026-09-02", changed)
	if !errors.Is(err, reportbiz.ErrPublicationConflict) {
		t.Fatalf("conflict error=%v", err)
	}
	missing := reportWithEvidenceIDs(t, reportfixture.Report(), evidenceIDs[0], "EVD33333333-3333-4333-8333-333333333333")
	_, err = useCase.Publish(ctx, "agentos-report-missing", missing)
	var reference *reportbiz.ReferenceError
	if !errors.As(err, &reference) {
		t.Fatalf("missing Evidence error=%v", err)
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
	for {
		page, err := useCase.ListIndustryChains(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		pageSizes = append(pageSizes, len(page.Items))
		for _, item := range page.Items {
			keys = append(keys, item.LocalKey)
		}
		if page.NextCursor == nil {
			break
		}
		request.Cursor = *page.NextCursor
	}
	if !reflect.DeepEqual(pageSizes, []int{20, 20, 14}) || len(keys) != 54 || keys[0] != "chain-01" || keys[53] != "chain-54" {
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
