package report

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
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

func TestPostgresReportPublicationReplayReadsAndImmutableRelationships(t *testing.T) {
	db := openReportTestDatabase(t, 0)
	evidenceIDs := publishReportEvidence(t, db)
	content := contentWithEvidenceIDs(t, evidenceIDs[0], evidenceIDs[1])
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	publishedAt := time.Date(2026, 9, 1, 1, 2, 3, 456789000, time.UTC)
	useCase, err := reportbiz.NewUseCase(store, func() time.Time { return publishedAt })
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	first, err := useCase.Publish(ctx, reportbiz.ContractVersion, "agentos-report-2026-09-01-a", content)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := useCase.Publish(ctx, reportbiz.ContractVersion, "agentos-report-2026-09-01-a", content)
	if err != nil {
		t.Fatal(err)
	}
	if first.Replayed || !replay.Replayed || first.Record.ID != replay.Record.ID || first.ContentHash != replay.ContentHash {
		t.Fatalf("publication results first=%#v replay=%#v", first, replay)
	}
	var reports, links int
	if err := db.QueryRow(`SELECT count(*) FROM reports WHERE publisher_report_id=$1`, "agentos-report-2026-09-01-a").Scan(&reports); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM report_evidence_links WHERE report_id=$1`, first.Record.ID).Scan(&links); err != nil {
		t.Fatal(err)
	}
	if reports != 1 || links == 0 {
		t.Fatalf("persisted reports=%d links=%d", reports, links)
	}

	changed := content
	changed.Title = "变更内容"
	_, err = useCase.Publish(ctx, reportbiz.ContractVersion, "agentos-report-2026-09-01-a", changed)
	if !errors.Is(err, reportbiz.ErrPublicationConflict) {
		t.Fatalf("conflicting replay error=%v", err)
	}
	missing := contentWithEvidenceIDs(t, evidenceIDs[0], "EVD33333333-3333-4333-8333-333333333333")
	_, err = useCase.Publish(ctx, reportbiz.ContractVersion, "agentos-report-missing-evidence", missing)
	var reference *reportbiz.ReferenceError
	if !errors.As(err, &reference) {
		t.Fatalf("missing Evidence error=%T %v", err, err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM reports WHERE publisher_report_id=$1`, "agentos-report-missing-evidence").Scan(&reports); err != nil || reports != 0 {
		t.Fatalf("failed publication retained Reports=%d error=%v", reports, err)
	}

	stored, err := useCase.Get(ctx, first.Record.ID)
	if err != nil || stored.ContentHash != first.ContentHash || stored.Content.Title != content.Title {
		t.Fatalf("Get()=%#v error=%v", stored, err)
	}
	home, err := useCase.GetHome(ctx, first.Record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if home.Geopolitics == nil || home.Geopolitics.Key != "geopolitics" || home.Macroeconomics == nil || home.Macroeconomics.Key != "macroeconomics" {
		t.Fatalf("GetHome()=%#v", home)
	}
	_, layer, related, err := useCase.GetLayer(ctx, first.Record.ID, "geopolitics")
	if err != nil || layer.Key != "geopolitics" || len(related) != 1 || related[0].Key != "chain-01" ||
		len(related[0].ImpactItems) != 1 || related[0].ImpactItems[0].Name != "产业节点 01" {
		t.Fatalf("GetLayer() layer=%#v related=%#v error=%v", layer, related, err)
	}
	_, chain, err := useCase.GetIndustryChain(ctx, first.Record.ID, "chain-01")
	if err != nil || chain.Key != "chain-01" || len(chain.Detail.NodeImpacts) != 1 {
		t.Fatalf("GetIndustryChain() chain=%#v error=%v", chain, err)
	}
	chainPage, err := useCase.ListIndustryChains(ctx, reportbiz.IndustryChainListRequest{ReportID: first.Record.ID, Limit: 1})
	if err != nil || len(chainPage.Items) != 1 || chainPage.Items[0].Claim.Key != "C-CHAIN-01" ||
		len(chainPage.Items[0].ImpactItems) != 1 || chainPage.Items[0].ImpactItems[0].Name != "产业节点 01" || chainPage.NextCursor != nil {
		t.Fatalf("ListIndustryChains()=%#v error=%v", chainPage, err)
	}
	evidence, err := useCase.ListEvidence(ctx, first.Record.ID, reportbiz.ScopeIndustryChainNode, "impact-01")
	if err != nil {
		t.Fatal(err)
	}
	if len(evidence) != 1 || evidence[0].EvidenceID != evidenceIDs[1] || evidence[0].DisplayOrder != 1 ||
		strings.TrimSpace(evidence[0].Summary) == "" || len(evidence[0].Keywords) != 1 ||
		evidence[0].PublishedAt == nil || !evidence[0].PublishedAt.Equal(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("ListEvidence()=%#v", evidence)
	}
	_, err = useCase.ListEvidence(ctx, first.Record.ID, reportbiz.ScopeAnchor, "unknown-anchor")
	if !errors.Is(err, reportbiz.ErrEvidenceScopeNotFound) {
		t.Fatalf("unknown Report Evidence scope error=%v", err)
	}

	second, err := useCase.Publish(ctx, reportbiz.ContractVersion, "agentos-report-2026-09-01-b", content)
	if err != nil {
		t.Fatal(err)
	}
	pageOne, err := useCase.List(ctx, reportbiz.ListRequest{Limit: 1})
	if err != nil || len(pageOne.Items) != 1 || pageOne.NextCursor == nil {
		t.Fatalf("first list page=%#v error=%v", pageOne, err)
	}
	pageTwo, err := useCase.List(ctx, reportbiz.ListRequest{Limit: 1, Cursor: *pageOne.NextCursor})
	if err != nil || len(pageTwo.Items) != 1 || pageTwo.Items[0].ID == pageOne.Items[0].ID {
		t.Fatalf("second list page=%#v error=%v", pageTwo, err)
	}
	wantIDs := []string{first.Record.ID, second.Record.ID}
	sort.Strings(wantIDs)
	if pageOne.Items[0].ID != wantIDs[0] || pageTwo.Items[0].ID != wantIDs[1] {
		t.Fatalf("same-time Report order=%q,%q want=%q,%q", pageOne.Items[0].ID, pageTwo.Items[0].ID, wantIDs[0], wantIDs[1])
	}

	type concurrentPublication struct {
		result reportbiz.PublicationResult
		err    error
	}
	const attempts = 6
	start := make(chan struct{})
	completed := make(chan concurrentPublication, attempts)
	for index := 0; index < attempts; index++ {
		go func() {
			<-start
			result, err := useCase.Publish(ctx, reportbiz.ContractVersion, "agentos-report-2026-09-01-concurrent", content)
			completed <- concurrentPublication{result: result, err: err}
		}()
	}
	close(start)
	createdCount := 0
	concurrentReportID := ""
	for index := 0; index < attempts; index++ {
		publication := <-completed
		if publication.err != nil {
			t.Fatalf("concurrent Report publication failed: %v", publication.err)
		}
		if !publication.result.Replayed {
			createdCount++
		}
		if concurrentReportID == "" {
			concurrentReportID = publication.result.Record.ID
		}
		if publication.result.Record.ID != concurrentReportID || publication.result.ContentHash != first.ContentHash {
			t.Fatalf("concurrent Report publication=%#v want report_id=%q content_hash=%q", publication.result, concurrentReportID, first.ContentHash)
		}
	}
	if createdCount != 1 {
		t.Fatalf("concurrent Report first publications=%d want=1", createdCount)
	}
	if err := db.QueryRow(`SELECT count(*) FROM reports WHERE publisher_report_id=$1`, "agentos-report-2026-09-01-concurrent").Scan(&reports); err != nil || reports != 1 {
		t.Fatalf("concurrent Report persisted rows=%d error=%v", reports, err)
	}

	assertPostgresCode(t, db, "23503", `INSERT INTO report_evidence_links
        (id,report_id,evidence_id,scope_type,scope_key,role,display_order)
		VALUES('RPE33333333-3333-4333-8333-333333333333',$1,'EVD33333333-3333-4333-8333-333333333333','section_summary','geopolitics','supports_claim',2)`, first.Record.ID)
	assertPostgresCode(t, db, "23505", `INSERT INTO report_evidence_links
		(id,report_id,evidence_id,scope_type,scope_key,role,display_order)
		SELECT 'RPE44444444-4444-4444-8444-444444444444',report_id,evidence_id,scope_type,scope_key,role,99
		FROM report_evidence_links WHERE report_id=$1 AND scope_type='section_summary' AND scope_key='geopolitics'`, first.Record.ID)
	assertPostgresCode(t, db, "23505", `INSERT INTO report_evidence_links
		(id,report_id,evidence_id,scope_type,scope_key,role,display_order)
		VALUES('RPE55555555-5555-4555-8555-555555555555',$1,$2,'section_summary','geopolitics','supports_claim',1)`, first.Record.ID, evidenceIDs[0])
	assertPostgresCode(t, db, "23514", `INSERT INTO report_evidence_links
		(id,report_id,evidence_id,scope_type,scope_key,role,display_order)
		VALUES('RPE66666666-6666-4666-8666-666666666666',$1,$2,'section_summary','Bad/Key','invalid',3)`, first.Record.ID, evidenceIDs[0])
	if _, err := db.Exec(`INSERT INTO report_evidence_links
		(id,report_id,evidence_id,scope_type,scope_key,role,display_order)
		VALUES('RPE77777777-7777-5777-8777-777777777777',$1,$2,'section_summary','v5-link','supports_claim',1)`, first.Record.ID, evidenceIDs[0]); err != nil {
		t.Fatalf("canonical UUID v5 Report Evidence Link ID was rejected: %v", err)
	}
	assertPostgresCode(t, db, "23503", `DELETE FROM evidences WHERE id=$1`, evidenceIDs[0])
	assertPostgresCode(t, db, "55000", `UPDATE reports SET content_hash=repeat('b',64) WHERE id=$1`, first.Record.ID)
	assertPostgresCode(t, db, "55000", `DELETE FROM reports WHERE id=$1`, first.Record.ID)
	assertPostgresCode(t, db, "55000", `TRUNCATE reports, report_evidence_links`)
	assertPostgresCode(t, db, "55000", `UPDATE report_evidence_links SET role='changed' WHERE report_id=$1`, first.Record.ID)
	assertPostgresCode(t, db, "55000", `DELETE FROM report_evidence_links WHERE report_id=$1`, first.Record.ID)
	assertPostgresCode(t, db, "55000", `TRUNCATE report_evidence_links`)
}

func TestPostgresReportIndustryChainCursorPagesFiftyFourSummaries(t *testing.T) {
	db := openReportTestDatabase(t, 0)
	evidenceIDs := publishReportEvidence(t, db)
	content := contentWithManyChainsAndEvidenceIDs(t, 54, evidenceIDs[0], evidenceIDs[1])
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := reportbiz.NewUseCase(store, func() time.Time {
		return time.Date(2026, 9, 2, 1, 2, 3, 0, time.UTC)
	})
	if err != nil {
		t.Fatal(err)
	}
	published, err := useCase.Publish(context.Background(), reportbiz.ContractVersion, "agentos-report-54-chains", content)
	if err != nil {
		t.Fatal(err)
	}

	request := reportbiz.IndustryChainListRequest{ReportID: published.Record.ID, Limit: 20}
	gotKeys := make([]string, 0, 54)
	pageSizes := []int{}
	for {
		page, err := useCase.ListIndustryChains(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		pageSizes = append(pageSizes, len(page.Items))
		for _, item := range page.Items {
			gotKeys = append(gotKeys, item.Key)
			if len(item.ImpactItems) != 1 || item.ImpactItems[0].NodeKey == "" {
				t.Fatalf("chain summary projection=%#v", item)
			}
		}
		if page.NextCursor == nil {
			break
		}
		request.Cursor = *page.NextCursor
	}
	if !reflect.DeepEqual(pageSizes, []int{20, 20, 14}) {
		t.Fatalf("page sizes=%v want=[20 20 14]", pageSizes)
	}
	if len(gotKeys) != 54 || gotKeys[0] != "chain-01" || gotKeys[53] != "chain-54" {
		t.Fatalf("paged keys count=%d first=%q last=%q", len(gotKeys), gotKeys[0], gotKeys[len(gotKeys)-1])
	}
}

func TestMigration79CreatesOnlyReviewedReportTablesAndCanonicalIdentityConstraints(t *testing.T) {
	db := openReportTestDatabase(t, 78)
	before := currentTables(t, db)
	migrationDir := reportMigrationDir(t)
	postgresfixture.ApplyMigration(t, db, migrationDir, 79)
	after := currentTables(t, db)
	created := difference(after, before)
	if !reflect.DeepEqual(created, []string{"report_evidence_links", "reports"}) {
		t.Fatalf("migration 79 created tables=%#v", created)
	}

	wantColumns := map[string][]string{
		"reports":               {"id", "source_report_id", "contract_version", "content_hash", "content", "published_at"},
		"report_evidence_links": {"id", "report_id", "evidence_id", "scope_type", "scope_key", "role", "display_order"},
	}
	for table, want := range wantColumns {
		if got := tableColumns(t, db, table); !reflect.DeepEqual(got, want) {
			t.Errorf("%s columns=%#v want=%#v", table, got, want)
		}
	}
	for _, retired := range []string{"report_publication_receipts", "report_projection_rows"} {
		if tableExists(t, db, retired) {
			t.Errorf("retired auxiliary table %q exists", retired)
		}
	}

	if _, err := db.Exec(`INSERT INTO reports
        (id,source_report_id,contract_version,content_hash,content,published_at)
        VALUES('RPT55555555-5555-5555-8555-555555555555','source-v5','report-publication.v1',repeat('a',64),'{}',now())`); err != nil {
		t.Fatalf("canonical UUID v5 Report ID was rejected: %v", err)
	}
	assertPostgresCode(t, db, "23514", `INSERT INTO reports
        (id,source_report_id,contract_version,content_hash,content,published_at)
        VALUES('RPT55555555-5555-4555-8555-555555555555',' source-with-whitespace','report-publication.v1',repeat('a',64),'{}',now())`)
	assertPostgresCode(t, db, "55000", `UPDATE reports SET content='{"x":1}' WHERE source_report_id='source-v5'`)

	var indexDefinition string
	if err := db.QueryRow(`SELECT indexdef FROM pg_indexes
        WHERE schemaname=current_schema() AND indexname='idx_reports_published_at_id'`).Scan(&indexDefinition); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(indexDefinition, "published_at DESC, id") {
		t.Fatalf("Report homepage index=%q", indexDefinition)
	}
	var reverseIndexDefinition string
	if err := db.QueryRow(`SELECT indexdef FROM pg_indexes
		WHERE schemaname=current_schema() AND indexname='idx_report_evidence_links_evidence_id'`).Scan(&reverseIndexDefinition); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(reverseIndexDefinition, "(evidence_id)") {
		t.Fatalf("Report Evidence reverse index=%q", reverseIndexDefinition)
	}
	var restrictiveForeignKeys int
	if err := db.QueryRow(`SELECT count(*) FROM pg_constraint
		WHERE connamespace=(SELECT oid FROM pg_namespace WHERE nspname=current_schema())
		  AND conrelid='report_evidence_links'::regclass AND contype='f' AND confdeltype='r'`).Scan(&restrictiveForeignKeys); err != nil {
		t.Fatal(err)
	}
	if restrictiveForeignKeys != 2 {
		t.Fatalf("Report Evidence RESTRICT foreign keys=%d want=2", restrictiveForeignKeys)
	}
	var triggerCount int
	if err := db.QueryRow(`SELECT count(*) FROM information_schema.triggers
        WHERE trigger_schema=current_schema() AND trigger_name IN ('trg_reports_immutable','trg_report_evidence_links_immutable')
          AND action_orientation='STATEMENT' AND event_manipulation IN ('UPDATE','DELETE','TRUNCATE')`).Scan(&triggerCount); err != nil {
		t.Fatal(err)
	}
	if triggerCount != 4 {
		t.Fatalf("immutable UPDATE/DELETE trigger event rows=%d want=4", triggerCount)
	}
}

func TestMigration79DownIsExplicitlyForwardOnly(t *testing.T) {
	migrationPath := filepath.Join(reportMigrationDir(t), "000079_add_report_publications.sql")
	payload, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatal(err)
	}
	sqlText := string(payload)
	downMarker := strings.Index(sqlText, "-- +goose Down")
	if downMarker < 0 {
		t.Fatal("migration 79 is missing its Down section")
	}
	downSQL := sqlText[downMarker:]
	if !strings.Contains(downSQL, "ERRCODE = '55000'") ||
		!strings.Contains(downSQL, "forward-only Report publication schema") {
		t.Fatalf("migration 79 Down section is not an explicit forward-only failure: %q", downSQL)
	}
	for _, destructiveStatement := range []string{"DROP TABLE", "DROP FUNCTION", "DELETE FROM", "TRUNCATE"} {
		if strings.Contains(strings.ToUpper(downSQL), destructiveStatement) {
			t.Fatalf("migration 79 Down section contains destructive statement %q", destructiveStatement)
		}
	}
}

func TestMigration80CutsEmptyReportStoreOverToV2(t *testing.T) {
	db := openReportTestDatabase(t, 79)
	postgresfixture.ApplyMigration(t, db, reportMigrationDir(t), 80)
	if got := tableColumns(t, db, "reports"); !reflect.DeepEqual(got, []string{"id", "publisher_report_id", "contract_version", "content_hash", "content", "published_at"}) {
		t.Fatalf("reports columns=%#v", got)
	}
	assertPostgresCode(t, db, "23514", `INSERT INTO reports
        (id,publisher_report_id,contract_version,content_hash,content,published_at)
        VALUES('RPT55555555-5555-5555-8555-555555555555','publisher','report-publication.v1',repeat('a',64),'{}',now())`)
}

func TestMigration80RefusesLossyCutover(t *testing.T) {
	db := openReportTestDatabase(t, 79)
	if _, err := db.Exec(`INSERT INTO reports
        (id,source_report_id,contract_version,content_hash,content,published_at)
        VALUES('RPT55555555-5555-5555-8555-555555555555','publisher','report-publication.v1',repeat('a',64),'{}',now())`); err != nil {
		t.Fatal(err)
	}
	assertPostgresCode(t, db, "55000", string(mustMigrationUp(t, filepath.Join(reportMigrationDir(t), "000080_upgrade_report_publication_v2.sql"))))
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

func publishReportEvidence(t *testing.T, db *sql.DB) []string {
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
	result, err := useCase.PublishEvidence(context.Background(), raw.ID, []evidencebiz.Evidence{
		reportEvidence("第一条报告依据", "依据一", "supports the first report claim"),
		reportEvidence("第二条报告依据", "依据二", "supports the second report claim"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.IDs) != 2 {
		t.Fatalf("published Evidence IDs=%#v", result.IDs)
	}
	return result.IDs
}

func reportEvidence(summary, keyword, action string) evidencebiz.Evidence {
	return evidencebiz.Evidence{Summary: summary, Keywords: []string{keyword}, Semantic: evidencebiz.Semantic{
		Actors: []string{"Example actor"}, Action: action, Objects: []string{"Report claim"},
		Stage: evidencebiz.EvidenceStageOccurred, Modality: evidencebiz.EvidenceModalityFact,
		Time: evidencebiz.EvidenceTime{Precision: evidencebiz.EvidenceTimeUnknown}, Jurisdictions: []string{},
		Metrics: []evidencebiz.EvidenceMetric{}, Attribution: &evidencebiz.EvidenceAttribution{},
	}}
}

func contentWithEvidenceIDs(t *testing.T, first, second string) reportbiz.Content {
	return replaceContentEvidenceIDs(t, reportfixture.Content(), first, second)
}

func contentWithManyChainsAndEvidenceIDs(t *testing.T, count int, first, second string) reportbiz.Content {
	return replaceContentEvidenceIDs(t, reportfixture.ContentWithManyChains(count), first, second)
}

func replaceContentEvidenceIDs(t *testing.T, source reportbiz.Content, first, second string) reportbiz.Content {
	t.Helper()
	payload, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	replaced := strings.ReplaceAll(string(payload), reportfixture.EvidenceOne, first)
	replaced = strings.ReplaceAll(replaced, reportfixture.EvidenceTwo, second)
	var content reportbiz.Content
	if err := json.Unmarshal([]byte(replaced), &content); err != nil {
		t.Fatal(err)
	}
	return content
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

func currentTables(t *testing.T, db *sql.DB) []string {
	t.Helper()
	rows, err := db.Query(`SELECT table_name FROM information_schema.tables
        WHERE table_schema=current_schema() AND table_type='BASE TABLE' ORDER BY table_name`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	result := []string{}
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			t.Fatal(err)
		}
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return result
}

func tableColumns(t *testing.T, db *sql.DB, table string) []string {
	t.Helper()
	rows, err := db.Query(`SELECT column_name FROM information_schema.columns
        WHERE table_schema=current_schema() AND table_name=$1 ORDER BY ordinal_position`, table)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	result := []string{}
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			t.Fatal(err)
		}
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return result
}

func difference(after, before []string) []string {
	seen := make(map[string]struct{}, len(before))
	for _, value := range before {
		seen[value] = struct{}{}
	}
	result := []string{}
	for _, value := range after {
		if _, exists := seen[value]; !exists {
			result = append(result, value)
		}
	}
	return result
}

func tableExists(t *testing.T, db *sql.DB, table string) bool {
	t.Helper()
	var exists bool
	if err := db.QueryRow(`SELECT to_regclass(current_schema() || '.' || $1) IS NOT NULL`, table).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	return exists
}

func assertPostgresCode(t *testing.T, db *sql.DB, want, query string, args ...any) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), query, args...)
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) || postgresError.Code != want {
		t.Fatalf("PostgreSQL error=%T %v, want SQLSTATE %s", err, err, want)
	}
}
