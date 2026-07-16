package repositories_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	app "github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/eventimport"
	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	domainimport "github.com/meierlink88/tidewise-ai/backend/internal/domain/eventimport"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

func TestEventImportPostgresIntegration(t *testing.T) {
	dsn := os.Getenv("TIDEWISE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TIDEWISE_TEST_DATABASE_URL to run event import PostgreSQL integration test")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	var databaseName string
	if err := db.QueryRowContext(ctx, `SELECT current_database()`).Scan(&databaseName); err != nil {
		t.Fatal(err)
	}
	if databaseName != "tidewise_local" {
		t.Fatalf("integration database = %q, want tidewise_local", databaseName)
	}
	assertEventImportSchema(t, ctx, db)

	runKey := fmt.Sprintf("event-import-integration-%d", time.Now().UTC().UnixNano())
	repo := repositories.NewPostgresRepository(db)
	service := app.NewService(repo)
	var cleanupResults []app.Result
	t.Cleanup(func() { cleanupEventImportFixtures(t, ctx, db, cleanupResults) })

	firstPackage := reviewedPackage(runKey + "-one")
	cleanupResults = append(cleanupResults, cleanupResultFromPlan(t, service, firstPackage))
	first := importConcurrentlyAndAssertSingleResult(t, ctx, db, service, firstPackage)

	changed := firstPackage
	changed.Event.Title = "different payload"
	if _, err := service.Import(ctx, changed); !errors.Is(err, app.ErrIdempotencyConflict) {
		t.Fatalf("different-hash import error = %v, want idempotency conflict", err)
	}

	secondPackage := reviewedPackage(runKey + "-two")
	cleanupResults = append(cleanupResults, cleanupResultFromPlan(t, service, secondPackage))
	third, err := service.Import(ctx, secondPackage)
	if err != nil {
		t.Fatal(err)
	}
	assertReceiptVerificationFailures(t, ctx, repo, first, third)
	assertRollbackRemovesSyntheticRawDocument(t, ctx, db, repo, runKey)
}

func TestCleanupResultFromPlanUsesDeterministicPlanIDs(t *testing.T) {
	plan := app.Plan{
		PackageID: "package", ReceiptID: "receipt", EventID: "event", PayloadHash: "hash",
		RawDocumentIDs: []string{"raw"}, EventSourceIDs: []string{"source"}, EventTagMapIDs: []string{"tag"},
	}
	got := cleanupResultFromPlanValue(plan)
	want := app.Result{
		PackageID: "package", ReceiptID: "receipt", EventID: "event", PayloadHash: "hash",
		RawDocumentIDs: []string{"raw"}, EventSourceIDs: []string{"source"}, EventTagMapIDs: []string{"tag"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("cleanup scope = %#v, want deterministic plan IDs %#v", got, want)
	}
}

func cleanupResultFromPlan(t *testing.T, service *app.Service, pkg domainimport.Package) app.Result {
	t.Helper()
	plan, err := service.Plan(pkg)
	if err != nil {
		t.Fatal(err)
	}
	return cleanupResultFromPlanValue(plan)
}

func cleanupResultFromPlanValue(plan app.Plan) app.Result {
	return app.Result{
		PackageID: plan.PackageID, ReceiptID: plan.ReceiptID, EventID: plan.EventID, PayloadHash: plan.PayloadHash,
		RawDocumentIDs: plan.RawDocumentIDs, EventSourceIDs: plan.EventSourceIDs, EventTagMapIDs: plan.EventTagMapIDs,
	}
}

func importConcurrentlyAndAssertSingleResult(t *testing.T, ctx context.Context, db *sql.DB, service *app.Service, input domainimport.Package) app.Result {
	t.Helper()
	ready := make(chan struct{}, 2)
	start := make(chan struct{})
	results := make(chan app.Result, 2)
	errs := make(chan error, 2)
	var workers sync.WaitGroup
	for range 2 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			ready <- struct{}{}
			<-start
			result, err := service.Import(ctx, input)
			if err != nil {
				errs <- err
				return
			}
			results <- result
		}()
	}
	<-ready
	<-ready
	close(start)
	workers.Wait()
	close(results)
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	var imported []app.Result
	for result := range results {
		imported = append(imported, result)
	}
	if len(imported) != 2 {
		t.Fatalf("concurrent imports = %d successful results, want 2", len(imported))
	}
	if !reflect.DeepEqual(imported[1], imported[0]) {
		t.Fatalf("concurrent same-hash replay = %#v, want %#v", imported[1], imported[0])
	}
	assertSinglePersistedResultSet(t, ctx, db, imported[0])
	return imported[0]
}

func assertSinglePersistedResultSet(t *testing.T, ctx context.Context, db *sql.DB, result app.Result) {
	t.Helper()
	for _, check := range []struct {
		name  string
		query string
		arg   any
	}{
		{"receipt", `SELECT count(*) FROM event_import_receipts WHERE id = $1`, result.ReceiptID},
		{"event", `SELECT count(*) FROM events WHERE id = $1`, result.EventID},
		{"raw document", `SELECT count(*) FROM raw_documents WHERE id = ANY($1::uuid[])`, result.RawDocumentIDs},
		{"event source", `SELECT count(*) FROM event_sources WHERE id = ANY($1::uuid[])`, result.EventSourceIDs},
		{"event tag map", `SELECT count(*) FROM event_tag_maps WHERE id = ANY($1::uuid[])`, result.EventTagMapIDs},
	} {
		var count int
		if err := db.QueryRowContext(ctx, check.query, check.arg).Scan(&count); err != nil {
			t.Fatal(err)
		}
		want := 1
		if check.name == "raw document" {
			want = len(result.RawDocumentIDs)
		} else if check.name == "event source" {
			want = len(result.EventSourceIDs)
		} else if check.name == "event tag map" {
			want = len(result.EventTagMapIDs)
		}
		if count != want {
			t.Fatalf("concurrent replay %s count = %d, want %d", check.name, count, want)
		}
	}
}

func reviewedPackage(key string) domainimport.Package {
	now := time.Now().UTC().Truncate(time.Second)
	published := now.Add(-time.Minute)
	documentID := "synthetic:" + key
	return domainimport.Package{
		IdempotencyKey: key,
		PackageID:      key,
		RawDocuments: []domainimport.RawDocumentInput{{
			DocumentID: documentID, SourceName: "synthetic integration fixture", SourceURL: "https://example.invalid/" + key,
			Title: "synthetic event document", ContentText: "synthetic only", ContentLevel: "summary",
			PublishedAt: &published, CollectedAt: now, ContentHash: "sha256:" + key,
		}},
		Event: domainimport.EventInput{
			DedupeKey: "synthetic:" + key, Title: "synthetic event", FactualSummary: "synthetic factual summary",
			FactStatus: "verified", EventStatus: "confirmed", FactPayload: map[string]any{"synthetic": true},
		},
		EventSources: []domainimport.EventSourceInput{{
			DocumentID: documentID, EvidenceExcerpt: "synthetic evidence", SourceURL: "https://example.invalid/" + key,
			EvidenceRelation: "supports", SupportsFields: []string{"title"}, SourceLevel: "secondary",
			ContentLevel: "summary", EvidenceHash: "sha256:evidence:" + key,
		}},
		EventTags: []domainimport.EventTagInput{{
			TagID: "b0fe1994-0db2-526c-a57f-97fa73c1b595", TagKind: "news_category", TagCode: "geopolitics",
			Confidence: "0.99", ReviewStatus: "approved", AssignmentReason: "synthetic", AssignSource: "rule",
		}},
		Review: domainimport.ReviewInput{
			ReviewID: key + "-review", PackageID: key, Decision: "auto_approved", EventStatus: "confirmed",
			FactStatus: "verified", EvidenceGrade: "synthetic", Reasons: []string{"synthetic"}, ComponentVersions: map[string]string{"integration": "v1"},
		},
	}
}

func assertReceiptVerificationFailures(t *testing.T, ctx context.Context, repo repositories.PostgresRepository, first, second app.Result) {
	t.Helper()
	err := repo.InTransaction(ctx, func(tx repositories.EventImportTransaction) error {
		missing := repositories.EventImportReceipt{
			EventID: first.EventID, RawDocumentIDs: []string{"00000000-0000-0000-0000-000000000000"},
			EventSourceIDs: first.EventSourceIDs, EventTagMapIDs: first.EventTagMapIDs,
		}
		if err := tx.VerifyReceiptResults(ctx, missing); err == nil {
			return errors.New("missing receipt raw document unexpectedly verified")
		}
		crossEvent := repositories.EventImportReceipt{
			EventID: second.EventID, RawDocumentIDs: first.RawDocumentIDs,
			EventSourceIDs: first.EventSourceIDs, EventTagMapIDs: first.EventTagMapIDs,
		}
		if err := tx.VerifyReceiptResults(ctx, crossEvent); err == nil {
			return errors.New("cross-event receipt results unexpectedly verified")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func assertRollbackRemovesSyntheticRawDocument(t *testing.T, ctx context.Context, db *sql.DB, repo repositories.PostgresRepository, runKey string) {
	t.Helper()
	docID := repositories.RawDocumentUUID(app.FixedSourceID, "rollback:"+runKey, "rollback:"+runKey, "sha256:rollback:"+runKey)
	wantRollback := errors.New("synthetic rollback")
	err := repo.InTransaction(ctx, func(tx repositories.EventImportTransaction) error {
		source, err := tx.Source(ctx, app.FixedSourceID)
		if err != nil {
			return err
		}
		if _, err := tx.UpsertRawDocument(ctx, domain.RawDocument{
			ID: docID, SourceID: app.FixedSourceID, IngestChannel: source.IngestChannel, SourceType: source.SourceType,
			SourceName: "synthetic rollback", SourceURL: "https://example.invalid/rollback", SourceExternalID: "rollback:" + runKey,
			Title: "rollback", ContentText: "rollback", ContentLevel: "summary", CollectedAt: time.Now().UTC(),
			ContentHash: "sha256:rollback:" + runKey, IngestStatus: domain.IngestStatusCollected,
		}); err != nil {
			return err
		}
		return wantRollback
	})
	if !errors.Is(err, wantRollback) {
		t.Fatalf("rollback transaction error = %v, want %v", err, wantRollback)
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM raw_documents WHERE id = $1`, docID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("rollback raw document count = %d, want 0", count)
	}
}

func assertEventImportSchema(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	for _, column := range []struct{ name, typeName string }{
		{"id", "uuid"}, {"idempotency_key", "text"}, {"package_id", "text"}, {"review_id", "text"},
		{"review_decision", "character varying(32)"}, {"payload_hash", "character(64)"}, {"event_id", "uuid"},
		{"raw_document_ids", "uuid[]"}, {"event_source_ids", "uuid[]"}, {"event_tag_map_ids", "uuid[]"},
		{"review_metadata", "jsonb"}, {"imported_at", "timestamp with time zone"},
	} {
		var actualType string
		var notNull bool
		err := db.QueryRowContext(ctx, `SELECT format_type(a.atttypid, a.atttypmod), a.attnotnull FROM pg_attribute a WHERE a.attrelid = 'event_import_receipts'::regclass AND a.attname = $1 AND a.attnum > 0 AND NOT a.attisdropped`, column.name).Scan(&actualType, &notNull)
		if err != nil || actualType != column.typeName || !notNull {
			t.Fatalf("receipt column %q type/not-null = %q/%t err=%v, want %q/true", column.name, actualType, notNull, err, column.typeName)
		}
	}
	for _, column := range []struct{ name, defaultFragment string }{
		{"review_metadata", "'{}'::jsonb"}, {"imported_at", "now()"},
	} {
		var defaultExpr sql.NullString
		if err := db.QueryRowContext(ctx, `SELECT pg_get_expr(d.adbin, d.adrelid) FROM pg_attrdef d JOIN pg_attribute a ON a.attrelid=d.adrelid AND a.attnum=d.adnum WHERE d.adrelid='event_import_receipts'::regclass AND a.attname=$1`, column.name).Scan(&defaultExpr); err != nil || !defaultExpr.Valid || !strings.Contains(defaultExpr.String, column.defaultFragment) {
			t.Fatalf("receipt column %q default = %q err=%v, want fragment %q", column.name, defaultExpr.String, err, column.defaultFragment)
		}
	}
	for _, name := range []string{"idx_event_import_receipts_event_id", "idx_event_import_receipts_package_id", "idx_event_import_receipts_imported_at"} {
		var count int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM pg_indexes WHERE schemaname='public' AND indexname=$1`, name).Scan(&count); err != nil || count != 1 {
			t.Fatalf("receipt index %q count=%d err=%v", name, count, err)
		}
	}
	for _, name := range []string{"chk_event_import_receipts_decision", "chk_event_import_receipts_payload_hash", "chk_event_import_receipts_raw_ids", "chk_event_import_receipts_source_ids", "chk_event_import_receipts_tag_ids", "chk_event_import_receipts_metadata"} {
		var count int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM pg_constraint WHERE conrelid = 'event_import_receipts'::regclass AND conname = $1`, name).Scan(&count); err != nil || count != 1 {
			t.Fatalf("receipt constraint %q count=%d err=%v", name, count, err)
		}
	}
	for _, constraint := range []struct{ name, typeCode string }{
		{"event_import_receipts_pkey", "p"}, {"event_import_receipts_idempotency_key_key", "u"}, {"event_import_receipts_event_id_fkey", "f"},
	} {
		var count int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM pg_constraint WHERE conrelid='event_import_receipts'::regclass AND conname=$1 AND contype=$2`, constraint.name, constraint.typeCode).Scan(&count); err != nil || count != 1 {
			t.Fatalf("receipt constraint %q/%q count=%d err=%v", constraint.name, constraint.typeCode, count, err)
		}
	}
}

func cleanupEventImportFixtures(t *testing.T, ctx context.Context, db *sql.DB, results []app.Result) {
	t.Helper()
	for _, result := range results {
		for _, statement := range []struct {
			query string
			arg   any
		}{
			{`DELETE FROM event_import_receipts WHERE id = $1`, result.ReceiptID},
			{`DELETE FROM event_tag_maps WHERE id = ANY($1::uuid[])`, result.EventTagMapIDs},
			{`DELETE FROM event_sources WHERE id = ANY($1::uuid[])`, result.EventSourceIDs},
			{`DELETE FROM events WHERE id = $1`, result.EventID},
			{`DELETE FROM raw_documents WHERE id = ANY($1::uuid[])`, result.RawDocumentIDs},
		} {
			if _, err := db.ExecContext(ctx, statement.query, statement.arg); err != nil {
				t.Fatalf("cleanup synthetic fixture: %v", err)
			}
		}
	}
	for _, result := range results {
		for _, check := range []struct {
			query string
			arg   any
		}{
			{`SELECT count(*) FROM event_import_receipts WHERE id = $1`, result.ReceiptID},
			{`SELECT count(*) FROM event_tag_maps WHERE id = ANY($1::uuid[])`, result.EventTagMapIDs},
			{`SELECT count(*) FROM event_sources WHERE id = ANY($1::uuid[])`, result.EventSourceIDs},
			{`SELECT count(*) FROM events WHERE id = $1`, result.EventID},
			{`SELECT count(*) FROM raw_documents WHERE id = ANY($1::uuid[])`, result.RawDocumentIDs},
		} {
			var count int
			if err := db.QueryRowContext(ctx, check.query, check.arg).Scan(&count); err != nil || count != 0 {
				t.Fatalf("synthetic fixture residual count=%d err=%v", count, err)
			}
		}
	}
}
