package evidence

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
	evidencebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/evidence"
	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
	"github.com/pressly/goose/v3"
)

func TestEvidenceListTextFiltersUseLiteralCaseInsensitiveContains(t *testing.T) {
	for _, expression := range []string{
		"strpos(lower(raw.title), lower($1)) > 0",
		"strpos(lower(evidence.summary), lower($2)) > 0",
		"strpos(lower(raw.source_name), lower($5)) > 0",
	} {
		if !strings.Contains(evidenceListWhere, expression) {
			t.Fatalf("Evidence list predicate is missing %q", expression)
		}
	}
	if strings.Contains(evidenceListWhere, "ILIKE") {
		t.Fatal("Evidence list text filters must not interpret LIKE wildcard characters")
	}
	if !strings.Contains(evidenceListWhere, "raw.source_id = $4") || strings.Contains(evidenceListWhere, "lower(raw.source_id)") {
		t.Fatal("Evidence source_id filter must use case-sensitive exact matching")
	}
}

func TestPostgresEvidencePublicationNaturalIdentityAndPersistence(t *testing.T) {
	db := openEvidencePublicationTestDatabase(t)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	publication, err := evidencebiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	raw := postgresEvidenceRaw("RAW6d88a7c8-da68-5dbc-b6ed-ca4b1a6cf175")
	raw.CategoryIDs = []evidencebiz.CategoryID{"EVCc18ddddb-14bc-5496-99ea-963ee2c25597"}
	created, err := publication.PublishRawEvidence(ctx, raw)
	if err != nil {
		t.Fatalf("publish Raw Evidence: %v", err)
	}
	rawID := created.ID
	rawCreatedAt := storedCreationTime(t, db, "raw_evidences", "id", rawID)
	replayed, err := publication.PublishRawEvidence(ctx, raw)
	if err != nil {
		t.Fatalf("replay Raw Evidence: %v", err)
	}
	if replayedCreatedAt := storedCreationTime(t, db, "raw_evidences", "id", rawID); !replayedCreatedAt.Equal(rawCreatedAt) {
		t.Fatalf("replayed Raw Evidence created_at = %s, want %s", replayedCreatedAt, rawCreatedAt)
	}
	if created.ID != postgresRawEvidenceID(raw) || replayed != created {
		t.Fatalf("Raw Evidence results created=%#v replayed=%#v", created, replayed)
	}
	var storedRawText string
	if err := db.QueryRowContext(ctx, `SELECT raw_text FROM raw_evidences WHERE id = $1`, rawID).Scan(&storedRawText); err != nil {
		t.Fatal(err)
	}
	if storedRawText != raw.RawText {
		t.Fatalf("stored raw_text = %q, want unchanged object path %q", storedRawText, raw.RawText)
	}

	items := []evidencebiz.Evidence{
		postgresEvidence(0),
		postgresEvidence(1),
	}
	items[1].Summary = "A second source statement supports a different atomic fact."
	published, err := publication.PublishEvidence(ctx, rawID, items)
	if err != nil {
		t.Fatalf("publish Evidence set: %v", err)
	}
	evidenceCreatedAt := make(map[string]time.Time, len(items))
	for _, evidenceID := range published.IDs {
		evidenceCreatedAt[evidenceID] = storedCreationTime(t, db, "evidences", "id", evidenceID)
	}
	reused, err := publication.PublishEvidence(ctx, rawID, items)
	if err != nil {
		t.Fatalf("replay Evidence set: %v", err)
	}
	for _, evidenceID := range published.IDs {
		if replayedCreatedAt := storedCreationTime(t, db, "evidences", "id", evidenceID); !replayedCreatedAt.Equal(evidenceCreatedAt[evidenceID]) {
			t.Fatalf("replayed Evidence %q created_at = %s, want %s", evidenceID, replayedCreatedAt, evidenceCreatedAt[evidenceID])
		}
	}
	if published.RawEvidenceID != rawID || !sameTestStrings(published.IDs, reused.IDs) || len(published.IDs) != 2 {
		t.Fatalf("Evidence results published=%#v reused=%#v", published, reused)
	}

	var keywords []string
	var keywordsJSON []byte
	var evidenceCount int
	if err := db.QueryRowContext(ctx, `SELECT array_to_json(keywords) FROM evidences WHERE id = $1`, published.Items[0].ID).Scan(&keywordsJSON); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(keywordsJSON, &keywords); err != nil {
		t.Fatal(err)
	}
	if !sameTestStrings(keywords, items[0].Keywords) {
		t.Fatalf("stored keywords = %#v, want %#v", keywords, items[0].Keywords)
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM evidences WHERE raw_evidence_id = $1`, rawID).Scan(&evidenceCount); err != nil {
		t.Fatal(err)
	}
	if evidenceCount != 2 {
		t.Fatalf("stored Evidence rows = %d, want 2", evidenceCount)
	}
	var storedSummary string
	var storedSemanticJSON []byte
	var storedIsSplit bool
	if err := db.QueryRowContext(ctx, `SELECT summary, semantic, is_split FROM evidences WHERE summary = $1`, items[0].Summary).
		Scan(&storedSummary, &storedSemanticJSON, &storedIsSplit); err != nil {
		t.Fatal(err)
	}
	var storedSemantic evidencebiz.Semantic
	if err := json.Unmarshal(storedSemanticJSON, &storedSemantic); err != nil {
		t.Fatal(err)
	}
	var storedSemanticFields map[string]json.RawMessage
	if err := json.Unmarshal(storedSemanticJSON, &storedSemanticFields); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"actors", "action", "objects", "stage", "modality", "time", "jurisdictions", "reason", "method", "metrics", "attribution"} {
		if _, exists := storedSemanticFields[field]; !exists {
			t.Fatalf("stored Evidence semantic is missing %q: %s", field, storedSemanticJSON)
		}
	}
	if storedSummary != items[0].Summary || !storedIsSplit || len(storedSemanticFields) != 11 ||
		!sameTestSemantic(storedSemantic, items[0].Semantic) {
		t.Fatalf("stored Evidence summary=%q semantic=%#v is_split=%t", storedSummary, storedSemantic, storedIsSplit)
	}
	singleRaw := postgresEvidenceRaw("single-evidence-is-split-false")
	singleRawResult, err := publication.PublishRawEvidence(ctx, singleRaw)
	if err != nil {
		t.Fatal(err)
	}
	singleResult, err := publication.PublishEvidence(ctx, singleRawResult.ID, []evidencebiz.Evidence{postgresEvidence(2)})
	if err != nil {
		t.Fatal(err)
	}
	var singleIsSplit bool
	if err := db.QueryRowContext(ctx, `SELECT is_split FROM evidences WHERE id = $1`, singleResult.IDs[0]).Scan(&singleIsSplit); err != nil {
		t.Fatal(err)
	}
	if singleIsSplit {
		t.Fatal("single Atomic Evidence persisted is_split=true, want false")
	}
	var categoryLinkID string
	if err := db.QueryRowContext(ctx, `SELECT id FROM raw_evidence_category_links WHERE raw_evidence_id = $1`, rawID).Scan(&categoryLinkID); err != nil {
		t.Fatal(err)
	}
	expectedCategoryLinkID, err := coreid.Derive(coreid.RawEvidenceCategoryLink, "raw-evidence-category-link", rawID, "EVCc18ddddb-14bc-5496-99ea-963ee2c25597")
	if err != nil {
		t.Fatal(err)
	}
	if categoryLinkID != expectedCategoryLinkID {
		t.Fatalf("Raw Evidence Category Link ID = %q", categoryLinkID)
	}
	assertPostgresCode(t, db, "23514", `
INSERT INTO raw_evidences(id,source_id,source_name,source_level,source_url,is_original,raw_text,collected_at)
VALUES('BAD','SRC_bad_raw_identity','Bad Source','L1_OFFICIAL','https://example.test/bad',true,'bad identity',now())`)
	assertPostgresCode(t, db, "23514", `
INSERT INTO evidences(id,raw_evidence_id,is_split,summary,keywords,semantic)
VALUES('BAD',$1,false,'bad identity',ARRAY['无效'],'{"actors":["Example"],"action":"bad identity","objects":["record"],"stage":"OCCURRED","modality":"FACT","time":{"raw":null,"start_at":null,"end_at":null,"precision":"UNKNOWN"},"jurisdictions":[],"reason":null,"method":null,"metrics":[],"attribution":{"reported_by":null,"claimed_by":null}}')`, rawID)
	assertPostgresCode(t, db, "23514", `
INSERT INTO raw_evidence_category_links(id,raw_evidence_id,category_id)
VALUES('BAD',$1,'EVCc18ddddb-14bc-5496-99ea-963ee2c25597')`, rawID)
	assertPostgresCode(t, db, "23505", `
INSERT INTO raw_evidence_category_links(id,raw_evidence_id,category_id)
VALUES('RCL11111111-1111-4111-8111-111111111111',$1,'EVCc18ddddb-14bc-5496-99ea-963ee2c25597')`, rawID)
	assertPostgresCode(t, db, "23503", `
INSERT INTO evidences(id,raw_evidence_id,is_split,summary,keywords,semantic)
VALUES('EVD11111111-1111-4111-8111-111111111111','RAW11111111-1111-4111-8111-111111111111',false,'missing parent',ARRAY['缺失'],'{"actors":["Example"],"action":"missing parent","objects":["record"],"stage":"OCCURRED","modality":"FACT","time":{"raw":null,"start_at":null,"end_at":null,"precision":"UNKNOWN"},"jurisdictions":[],"reason":null,"method":null,"metrics":[],"attribution":{"reported_by":null,"claimed_by":null}}')`)
	assertPostgresCode(t, db, "23514", `
INSERT INTO evidences(id,raw_evidence_id,is_split,summary,keywords,semantic)
VALUES('EVD11111111-1111-4111-8111-111111111112',$1,false,'invalid semantic',ARRAY['无效'],'{"actors":[],"action":"","objects":[],"stage":"OCCURRED","modality":"FACT","time":{"raw":null,"start_at":null,"end_at":null,"precision":"UNKNOWN"},"jurisdictions":[],"reason":null,"method":null,"metrics":[],"attribution":{"reported_by":null,"claimed_by":null},"summary":"duplicate"}')`, rawID)

	drift := append([]evidencebiz.Evidence(nil), items...)
	drift[0].Semantic.Action = "drifted"
	_, err = publication.PublishEvidence(ctx, rawID, drift)
	var conflict *evidencebiz.ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("semantic drift error = %v, want ConflictError", err)
	}
	summaryDrift := append([]evidencebiz.Evidence(nil), items...)
	summaryDrift[0].Summary = "drifted summary"
	_, err = publication.PublishEvidence(ctx, rawID, summaryDrift)
	if !errors.As(err, &conflict) {
		t.Fatalf("summary drift error = %v, want ConflictError", err)
	}

}

func assertPostgresCode(t *testing.T, db *sql.DB, want, query string, args ...any) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), query, args...)
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) || postgresError.Code != want {
		t.Fatalf("PostgreSQL error = %T %v, want SQLSTATE %s", err, err, want)
	}
}

func TestPostgresTargetTablesUseOnlyIDAsPrimaryKey(t *testing.T) {
	db := openEvidencePublicationTestDatabase(t)
	rows, err := db.Query(`
SELECT tc.table_name, kcu.column_name
FROM information_schema.table_constraints tc
JOIN information_schema.key_column_usage kcu
  ON kcu.constraint_catalog = tc.constraint_catalog
 AND kcu.constraint_schema = tc.constraint_schema
 AND kcu.constraint_name = tc.constraint_name
WHERE tc.constraint_schema = current_schema()
  AND tc.constraint_type = 'PRIMARY KEY'
  AND tc.table_name IN (
    'organization_categories', 'organization_domain_tags', 'organization_domain_tag_links',
    'raw_evidences', 'evidences', 'raw_evidence_category_links'
  )
ORDER BY tc.table_name, kcu.ordinal_position`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	primaryKeys := make(map[string][]string)
	for rows.Next() {
		var table, column string
		if err := rows.Scan(&table, &column); err != nil {
			t.Fatal(err)
		}
		primaryKeys[table] = append(primaryKeys[table], column)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{
		"organization_categories", "organization_domain_tags", "organization_domain_tag_links",
		"raw_evidences", "evidences", "raw_evidence_category_links",
	} {
		columns := primaryKeys[table]
		if len(columns) != 1 || columns[0] != "id" {
			t.Errorf("%s primary key columns = %#v, want [id]", table, columns)
		}
	}
}

func TestPostgresAtomicEvidenceSchemaUsesSummaryKeywordsAndSemantic(t *testing.T) {
	db := openEvidencePublicationTestDatabase(t)
	rows, err := db.Query(`
SELECT column_name, data_type
FROM information_schema.columns
WHERE table_schema = current_schema() AND table_name = 'evidences'`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	columns := make(map[string]string)
	for rows.Next() {
		var name, dataType string
		if err := rows.Scan(&name, &dataType); err != nil {
			t.Fatal(err)
		}
		columns[name] = dataType
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"id": "character varying", "raw_evidence_id": "character varying", "is_split": "boolean",
		"summary": "character varying", "keywords": "ARRAY", "semantic": "jsonb", "created_at": "timestamp with time zone",
	}
	if len(columns) != len(want) {
		t.Fatalf("Evidence columns = %#v, want exactly %#v", columns, want)
	}
	for name, dataType := range want {
		if columns[name] != dataType {
			t.Fatalf("Evidence column %s type = %q, want %q", name, columns[name], dataType)
		}
	}
}

func TestPostgresEvidenceCategoryCatalogReturnsCurrentFixedCategories(t *testing.T) {
	db := openEvidencePublicationTestDatabase(t)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	categories, err := store.ListCategories(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	fixturePayload, err := os.ReadFile(filepath.Join("..", "..", "..", "api", "data", "v1", "evidence", "testdata", "evidence-category-catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Result struct {
			Categories []struct {
				ID          string `json:"id"`
				Code        string `json:"code"`
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"categories"`
		} `json:"result"`
	}
	if err := json.Unmarshal(fixturePayload, &fixture); err != nil {
		t.Fatal(err)
	}
	want := fixture.Result.Categories
	if len(want) != 11 || len(categories) != len(want) {
		t.Fatalf("Evidence Category count = %d, fixture count %d", len(categories), len(want))
	}
	for index, expected := range want {
		category := categories[index]
		if string(category.ID) != expected.ID || category.Code != expected.Code || category.Name != expected.Name || category.Description != expected.Description {
			t.Fatalf("category[%d] = %#v, want exact fixture %#v", index, category, expected)
		}
	}
}

func TestListEvidenceReturnsJoinedRawEvidenceAndCompleteCategories(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	publishedFrom := time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC)
	publishedTo := time.Date(2026, 8, 19, 0, 0, 0, 0, time.UTC)
	collectedFrom := publishedFrom
	collectedTo := publishedTo
	categoryID := evidencebiz.CategoryID("EVCc18ddddb-14bc-5496-99ea-963ee2c25597")
	filter := evidencebiz.EvidenceListFilter{
		Title: "Source", Summary: "Atomic", CategoryID: categoryID,
		SourceID: "SRC_example_00000000000000000000", SourceName: "Example",
		SourceLevel: evidencebiz.SourceLevelWire, IsSplit: testBoolPointer(true),
		PublishedFrom: &publishedFrom, PublishedTo: &publishedTo,
		CollectedFrom: &collectedFrom, CollectedTo: &collectedTo, Page: 2, PageSize: 10,
	}
	args := []driver.Value{
		"Source", "Atomic", string(categoryID), "SRC_example_00000000000000000000", "Example", string(evidencebiz.SourceLevelWire), true,
		publishedFrom, publishedTo, collectedFrom, collectedTo,
	}
	mock.ExpectQuery("SELECT COUNT\\(\\*\\).*raw\\.source_id = \\$4").WithArgs(args...).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	listArgs := append(append([]driver.Value{}, args...), int64(10), int64(10))
	categoryCreatedAt := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	categories := fmt.Sprintf(`[{"id":"%s","code":"EVENT_BRIEF","name":"事件快讯","description":"事件材料","created_at":"%s"}]`, categoryID, categoryCreatedAt.Format(time.RFC3339))
	semantic := []byte(`{"actors":["Example Corp"],"action":"announced a production line","objects":["production line"],"stage":"ANNOUNCED","modality":"FACT","time":{"raw":null,"start_at":null,"end_at":null,"precision":"UNKNOWN"},"jurisdictions":["China"],"reason":null,"method":"through an exchange filing","metrics":[],"attribution":{"reported_by":"Example Wire","claimed_by":"Example Corp"}}`)
	keywords := []byte(`["扩产","产能"]`)
	quotedSourceName := "Example Corp filing"
	mock.ExpectQuery("SELECT evidence.id, evidence.raw_evidence_id").WithArgs(listArgs...).WillReturnRows(sqlmock.NewRows([]string{
		"id", "raw_evidence_id", "is_split", "summary", "semantic", "title", "source_id", "source_name", "source_level", "source_url", "is_original", "quoted_source_name", "published_at", "collected_at", "keywords", "categories",
	}).AddRow(
		"EVD5cb71bef-5b1d-5995-add0-7408eaa2be15", "RAW15bec7e3-998c-5434-aa5d-29712c4c67cf", true, "Atomic fact", semantic,
		"Source title", "SRC_example_00000000000000000000", "Example Wire", "L2_WIRE", "https://example.com/report", false, quotedSourceName,
		publishedFrom.Add(time.Hour), collectedFrom.Add(65*time.Minute), keywords, []byte(categories),
	))

	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	page, err := store.ListEvidence(context.Background(), filter)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || page.Page != 2 || page.PageSize != 10 || len(page.Items) != 1 {
		t.Fatalf("page = %#v", page)
	}
	item := page.Items[0]
	if item.Title == nil || *item.Title != "Source title" || item.Summary != "Atomic fact" ||
		item.SourceID != "SRC_example_00000000000000000000" || item.SourceName != "Example Wire" || item.SourceLevel != evidencebiz.SourceLevelWire || !item.IsSplit ||
		len(item.Semantic.Actors) != 1 || item.Semantic.Actors[0] != "Example Corp" || item.Semantic.Action != "announced a production line" ||
		item.SourceURL != "https://example.com/report" || item.IsOriginal || item.QuotedSourceName == nil || *item.QuotedSourceName != quotedSourceName ||
		len(item.Keywords) != 2 || item.Keywords[0] != "扩产" ||
		len(item.Categories) != 1 || item.Categories[0].ID != categoryID {
		t.Fatalf("item = %#v", item)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func testBoolPointer(value bool) *bool { return &value }

func TestEvidenceCategoryCatalogRejectsInvalidPersistedCollection(t *testing.T) {
	now := time.Date(2026, 8, 16, 8, 0, 0, 0, time.UTC)
	validID := "EVCc18ddddb-14bc-5496-99ea-963ee2c25597"
	otherID := "EVC5b12ffce-178d-56ed-a54f-c01696c486f4"
	tests := []struct {
		name string
		rows *sqlmock.Rows
	}{
		{name: "empty", rows: categoryCatalogRows()},
		{name: "invalid row", rows: categoryCatalogRows().AddRow("EVC_001", "EVENT_BRIEF", "事件快讯", "事件快讯定义", now)},
		{name: "duplicate ID", rows: categoryCatalogRows().
			AddRow(validID, "EVENT_BRIEF", "事件快讯", "事件快讯定义", now).
			AddRow(validID, "SECOND_CODE", "第二分类", "第二分类定义", now)},
		{name: "duplicate code", rows: categoryCatalogRows().
			AddRow(validID, "EVENT_BRIEF", "事件快讯", "事件快讯定义", now).
			AddRow(otherID, "EVENT_BRIEF", "第二分类", "第二分类定义", now)},
		{name: "unstable order", rows: categoryCatalogRows().
			AddRow(otherID, "IN_DEPTH_REPORT", "专题/深度报道", "专题定义", now).
			AddRow(validID, "EVENT_BRIEF", "事件快讯", "事件快讯定义", now)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = db.Close() })
			mock.ExpectQuery(regexp.QuoteMeta(categoryCatalogQuery)).WillReturnRows(test.rows)
			store, err := NewStore(db)
			if err != nil {
				t.Fatal(err)
			}
			categories, err := store.ListCategories(context.Background())
			var invariant *persistedInvariantError
			if err == nil || !errors.As(err, &invariant) || len(categories) != 0 {
				t.Fatalf("categories=%#v error=%v, want persisted invariant failure", categories, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestEvidenceCategoryCatalogRejectsRepositoryFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectQuery(regexp.QuoteMeta(categoryCatalogQuery)).
		WillReturnError(errors.New("database unavailable"))
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	categories, err := store.ListCategories(context.Background())
	if err == nil || len(categories) != 0 {
		t.Fatalf("categories=%#v error=%v, want closed repository failure", categories, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func categoryCatalogRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "code", "name", "description", "created_at"})
}

func storedCreationTime(t *testing.T, db *sql.DB, table, identityColumn, identity string) time.Time {
	t.Helper()
	var createdAt time.Time
	query := fmt.Sprintf(`SELECT created_at FROM %s WHERE %s = $1`, table, identityColumn)
	if err := db.QueryRow(query, identity).Scan(&createdAt); err != nil {
		t.Fatalf("read %s created_at: %v", table, err)
	}
	if createdAt.IsZero() {
		t.Fatalf("%s created_at is zero", table)
	}
	return createdAt
}

func TestEvidenceTransactionRejectsInvalidPersistedRawEvidence(t *testing.T) {
	const rawEvidenceID = "RAW5b6ecd34-8a1a-56e4-8a7c-79efd7843473"
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectBegin()
	mock.ExpectQuery("FROM raw_evidences").
		WithArgs(rawEvidenceID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "source_id", "source_name", "source_level", "source_url", "is_original",
			"quoted_source_id", "quoted_source_name", "title", "raw_text", "published_at", "collected_at",
			"content_hash",
		}).AddRow(
			rawEvidenceID, "SRC_0000000000000000000000000000", "Source", "INVALID",
			"https://example.test/article", true, nil, nil, nil, "hello", nil,
			time.Date(2026, 8, 11, 1, 2, 3, 0, time.UTC),
			"2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
		))
	mock.ExpectRollback()

	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	accepted := errors.New("invalid persisted Raw Evidence was accepted")
	ctx := context.Background()
	err = store.InTransaction(ctx, func(tx evidencebiz.Transaction) error {
		_, readErr := tx.RawEvidence(ctx, rawEvidenceID)
		if readErr == nil {
			return accepted
		}
		return readErr
	})
	var invariantErr *persistedInvariantError
	if errors.Is(err, accepted) || !errors.As(err, &invariantErr) || invariantErr.field != "source_level" {
		t.Fatalf("RawEvidence() error = %v, want persisted source_level invariant error", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEvidenceTransactionRejectsInvalidPersistedEvidenceSet(t *testing.T) {
	const rawEvidenceID = "RAW5b6ecd34-8a1a-56e4-8a7c-79efd7843473"
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectBegin()
	rows := sqlmock.NewRows([]string{"id", "raw_evidence_id", "is_split", "summary", "keywords", "semantic"})
	rows.AddRow(persistedEvidenceRow("EVD0f10cab3-e6ca-5bbc-ac33-5b09d3ff1602", rawEvidenceID, false, "first fact")...)
	rows.AddRow(persistedEvidenceRow("EVDc8222fc3-a24f-5d44-b204-09dfb2b8960f", rawEvidenceID, false, "second fact")...)
	mock.ExpectQuery("FROM evidences").
		WithArgs(rawEvidenceID).
		WillReturnRows(rows)
	mock.ExpectRollback()

	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	accepted := errors.New("invalid persisted Evidence set was accepted")
	ctx := context.Background()
	err = store.InTransaction(ctx, func(tx evidencebiz.Transaction) error {
		_, readErr := tx.EvidencesByRawEvidence(ctx, rawEvidenceID)
		if readErr == nil {
			return accepted
		}
		return readErr
	})
	var invariantErr *persistedInvariantError
	if errors.Is(err, accepted) || !errors.As(err, &invariantErr) || invariantErr.field != "is_split" {
		t.Fatalf("EvidencesByRawEvidence() error = %v, want persisted is_split invariant error", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEvidenceTransactionRejectsPersistedSemanticOutsidePublicLimits(t *testing.T) {
	const rawEvidenceID = "RAW5b6ecd34-8a1a-56e4-8a7c-79efd7843473"
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectBegin()
	semantic := fmt.Sprintf(`{"actors":[%q],"action":"atomic fact","objects":["object"],"stage":"OCCURRED","modality":"FACT","time":{"raw":null,"start_at":null,"end_at":null,"precision":"UNKNOWN"},"jurisdictions":[],"reason":null,"method":null,"metrics":[],"attribution":{"reported_by":null,"claimed_by":null}}`, strings.Repeat("a", 101))
	rows := sqlmock.NewRows([]string{"id", "raw_evidence_id", "is_split", "summary", "keywords", "semantic"}).
		AddRow("EVD0f10cab3-e6ca-5bbc-ac33-5b09d3ff1602", rawEvidenceID, false, "fact", []byte(`["事实"]`), []byte(semantic))
	mock.ExpectQuery("FROM evidences").WithArgs(rawEvidenceID).WillReturnRows(rows)
	mock.ExpectRollback()

	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	err = store.InTransaction(context.Background(), func(tx evidencebiz.Transaction) error {
		_, readErr := tx.EvidencesByRawEvidence(context.Background(), rawEvidenceID)
		return readErr
	})
	var invariantErr *persistedInvariantError
	if !errors.As(err, &invariantErr) || invariantErr.field != "semantic.actors" {
		t.Fatalf("EvidencesByRawEvidence() error = %v, want persisted semantic.actors invariant error", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func persistedEvidenceRow(id, rawEvidenceID string, isSplit bool, summary string) []driver.Value {
	return []driver.Value{
		id, rawEvidenceID, isSplit, summary, []byte(`["事实"]`),
		[]byte(`{"actors":["Example"],"action":"atomic fact","objects":["object"],"stage":"OCCURRED","modality":"FACT","time":{"raw":null,"start_at":null,"end_at":null,"precision":"UNKNOWN"},"jurisdictions":[],"reason":null,"method":null,"metrics":[],"attribution":{"reported_by":null,"claimed_by":null}}`),
	}
}

func postgresEvidenceRaw(publicationKey string) evidencebiz.RawEvidence {
	publishedAt := time.Date(2026, 8, 11, 1, 0, 0, 123456789, time.UTC)
	return evidencebiz.RawEvidence{
		PublicationKey: publicationKey, SourceID: "SRC_postgres_0000000000000000000", SourceName: "Example Wire",
		SourceLevel: "L2_WIRE", SourceURL: "https://example.test/evidence", IsOriginal: true,
		RawText: "/raw-evidence/documents/2026/08/11/1111111111111111111111111111111111111111111111111111111111111111.md", PublishedAt: &publishedAt,
		CollectedAt: time.Date(2026, 8, 11, 1, 5, 0, 987654321, time.UTC),
	}
}

func postgresRawEvidenceID(raw evidencebiz.RawEvidence) string {
	value, _ := coreid.Derive(coreid.RawEvidence, "raw-evidence-publication", raw.PublicationKey)
	return value
}

func postgresEvidence(variant int) evidencebiz.Evidence {
	return evidencebiz.Evidence{
		Summary:  fmt.Sprintf("Example Corp expands production %d", variant),
		Keywords: []string{"扩产", "产能"},
		Semantic: evidencebiz.Semantic{
			Actors: []string{"Example Corp"}, Action: fmt.Sprintf("expanded production line %d", variant),
			Objects: []string{"production capacity"}, Stage: evidencebiz.EvidenceStageOccurred, Modality: evidencebiz.EvidenceModalityFact,
			Time:          evidencebiz.EvidenceTime{Raw: testStringPointer("2026-08-10"), Precision: evidencebiz.EvidenceTimeDay},
			Jurisdictions: []string{}, Method: testStringPointer("by adding capacity"), Metrics: []evidencebiz.EvidenceMetric{},
			Attribution: &evidencebiz.EvidenceAttribution{ReportedBy: testStringPointer("Example Wire")},
		},
	}
}

func testStringPointer(value string) *string { return &value }

func sameTestStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func sameTestSemantic(left, right evidencebiz.Semantic) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftJSON) == string(rightJSON)
}

func sameTestOptionalString(left, right *string) bool {
	return left == nil && right == nil || left != nil && right != nil && *left == *right
}

func openEvidencePublicationTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	databaseURL := os.Getenv("TIDEWISE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TIDEWISE_TEST_DATABASE_URL to run Evidence Publication integration tests")
	}
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	host := parsed.Hostname()
	if host != "localhost" && host != "127.0.0.1" && host != "::1" {
		t.Fatalf("Evidence Publication integration database must use a loopback host, got %q", host)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	admin, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("tw_evidence_publication_%d", time.Now().UnixNano())
	if _, err := admin.ExecContext(ctx, `CREATE SCHEMA `+schema); err != nil {
		closeErr := admin.Close()
		if closeErr != nil {
			t.Errorf("close Evidence Publication integration admin after schema failure: %v", closeErr)
		}
		t.Fatal(err)
	}
	var db *sql.DB
	t.Cleanup(func() {
		var dbCloseErr error
		if db != nil {
			dbCloseErr = db.Close()
		}
		_, dropErr := admin.ExecContext(context.Background(), `DROP SCHEMA `+schema+` CASCADE`)
		adminCloseErr := admin.Close()
		if cleanupErr := errors.Join(dbCloseErr, dropErr, adminCloseErr); cleanupErr != nil {
			t.Errorf("clean Evidence Publication integration database: %v", cleanupErr)
		}
	})

	config, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	config.RuntimeParams["search_path"] = schema
	config.RuntimeParams["tidewise.phase_a_cleanup_write_authorized"] = "reviewed_backup_verified"
	config.RuntimeParams["tidewise.external_identifier_schema_write_authorized"] = "reviewed_backup_verified"
	config.RuntimeParams["tidewise.alliance_economy_schema_write_authorized"] = "reviewed_local_cleanup_verified"
	db = stdlib.OpenDB(*config)
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if err := goose.UpContext(ctx, db, migrationDir); err != nil {
		t.Fatalf("apply migrations in isolated schema: %v", err)
	}

	return db
}
