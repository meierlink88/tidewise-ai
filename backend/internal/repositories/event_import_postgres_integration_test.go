package repositories_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"reflect"
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
	var results []app.Result
	t.Cleanup(func() { cleanupEventImportFixtures(t, ctx, db, results) })

	firstPackage := reviewedPackage(runKey + "-one")
	first, err := service.Import(ctx, firstPackage)
	if err != nil {
		t.Fatal(err)
	}
	results = append(results, first)
	second, err := service.Import(ctx, firstPackage)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(second, first) {
		t.Fatalf("same-hash replay = %#v, want %#v", second, first)
	}

	changed := firstPackage
	changed.Event.Title = "different payload"
	if _, err := service.Import(ctx, changed); !errors.Is(err, app.ErrIdempotencyConflict) {
		t.Fatalf("different-hash import error = %v, want idempotency conflict", err)
	}

	secondPackage := reviewedPackage(runKey + "-two")
	third, err := service.Import(ctx, secondPackage)
	if err != nil {
		t.Fatal(err)
	}
	results = append(results, third)
	assertReceiptVerificationFailures(t, ctx, repo, first, third)
	assertRollbackRemovesSyntheticRawDocument(t, ctx, db, repo, runKey)
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
	for _, column := range []string{"raw_document_ids", "event_source_ids", "event_tag_map_ids"} {
		var count int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM information_schema.columns WHERE table_name = 'event_import_receipts' AND column_name = $1`, column).Scan(&count); err != nil || count != 1 {
			t.Fatalf("receipt column %q count=%d err=%v", column, count, err)
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
	var foreignKeys int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM pg_constraint WHERE conrelid = 'event_import_receipts'::regclass AND contype = 'f'`).Scan(&foreignKeys); err != nil || foreignKeys != 1 {
		t.Fatalf("receipt foreign key count=%d err=%v", foreignKeys, err)
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
