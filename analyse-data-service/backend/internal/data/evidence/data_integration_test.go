package evidence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	evidencebiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/evidence"
	"github.com/pressly/goose/v3"
)

func TestPostgresEvidencePublicationNaturalIdentityTransactionsAndSchema(t *testing.T) {
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

	raw := postgresEvidenceRaw("RAW_postgres_0000000000000000000")
	created, err := publication.PublishRawEvidence(ctx, "neutral-publisher", raw)
	if err != nil {
		t.Fatalf("publish Raw Evidence: %v", err)
	}
	replayed, err := publication.PublishRawEvidence(ctx, "neutral-publisher", raw)
	if err != nil {
		t.Fatalf("replay Raw Evidence: %v", err)
	}
	if created.RawEvidence.Disposition != evidencebiz.DispositionCreated ||
		replayed.RawEvidence.Disposition != evidencebiz.DispositionReused ||
		created.ReceiptID == replayed.ReceiptID {
		t.Fatalf("Raw Evidence results created=%#v replayed=%#v", created, replayed)
	}

	items := []evidencebiz.Evidence{
		postgresEvidence("EVD_postgres_0000000000000000000", 0),
		postgresEvidence("EVD_postgres_0000000000000000001", 1),
	}
	items[1].SourceWhat = "A second source statement supports the same normalized fact."
	published, err := publication.PublishEvidence(ctx, "neutral-publisher", raw.RawEvidenceID, items)
	if err != nil {
		t.Fatalf("publish Evidence set: %v", err)
	}
	reused, err := publication.PublishEvidence(ctx, "neutral-publisher", raw.RawEvidenceID, items)
	if err != nil {
		t.Fatalf("replay Evidence set: %v", err)
	}
	if published.Counts.Created != 2 || reused.Counts.Reused != 2 || published.ReceiptID == reused.ReceiptID {
		t.Fatalf("Evidence results published=%#v reused=%#v", published, reused)
	}

	var keywords []string
	var keywordsJSON []byte
	var evidenceCount, expressionIndexCount, expressionUniqueCount int
	if err := db.QueryRowContext(ctx, `SELECT array_to_json(keywords) FROM raw_evidences WHERE raw_evidence_id = $1`, raw.RawEvidenceID).Scan(&keywordsJSON); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(keywordsJSON, &keywords); err != nil {
		t.Fatal(err)
	}
	if !sameTestStrings(keywords, raw.Keywords) {
		t.Fatalf("stored keywords = %#v, want %#v", keywords, raw.Keywords)
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM evidences WHERE expression_key = $1`, items[0].ExpressionKey).Scan(&evidenceCount); err != nil {
		t.Fatal(err)
	}
	if evidenceCount != 2 {
		t.Fatalf("Evidence rows sharing expression_key = %d, want 2", evidenceCount)
	}
	if err := db.QueryRowContext(ctx, `
SELECT count(*), count(*) FILTER (WHERE NOT i.indisunique)
FROM pg_index i
JOIN pg_class c ON c.oid = i.indexrelid
WHERE c.relname = 'idx_evidences_expression_key'`).Scan(&expressionIndexCount, &expressionUniqueCount); err != nil {
		t.Fatal(err)
	}
	if expressionIndexCount != 1 || expressionUniqueCount != 1 {
		t.Fatalf("expression key index count=%d nonunique=%d", expressionIndexCount, expressionUniqueCount)
	}
	assertEvidenceReplicatedSchema(t, db)

	drift := append([]evidencebiz.Evidence(nil), items...)
	drift[0].SourceWhat = "drifted"
	_, err = publication.PublishEvidence(ctx, "neutral-publisher", raw.RawEvidenceID, drift)
	var conflict *evidencebiz.ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("drift error = %v, want ConflictError", err)
	}

	assertConcurrentRawEvidenceConvergence(t, publication, db)
	assertConcurrentRawEvidenceConflict(t, publication, db)
	assertConcurrentEvidenceConvergenceAndConflict(t, publication, db)
	assertEvidenceTransactionRollback(t, publication, store, db)
	assertEvidenceDeadlineCancelsQueryAndRollsBack(t, store, db)
	assertEvidenceReceiptsImmutable(t, db, created.ReceiptID, published.ReceiptID)
}

func assertEvidenceReplicatedSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	var splitDefault, rawTextComment string
	var restrictFKs, quotedChecks int
	if err := db.QueryRow(`
SELECT column_default
FROM information_schema.columns
WHERE table_schema = current_schema() AND table_name = 'evidences' AND column_name = 'split_order'`).Scan(&splitDefault); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`
SELECT col_description('raw_evidences'::regclass, a.attnum)
FROM pg_attribute a
WHERE a.attrelid = 'raw_evidences'::regclass AND a.attname = 'raw_text'`).Scan(&rawTextComment); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`
SELECT count(*) FROM pg_constraint
WHERE conrelid = 'evidences'::regclass AND contype = 'f' AND confdeltype = 'r'`).Scan(&restrictFKs); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`
SELECT count(*) FROM pg_constraint
WHERE conrelid = 'raw_evidences'::regclass
  AND conname IN ('chk_raw_evidences_quoted_source_id', 'chk_raw_evidences_quoted_source_name')`).Scan(&quotedChecks); err != nil {
		t.Fatal(err)
	}
	if splitDefault != "0" || rawTextComment != "原始文章完整正文。" || restrictFKs != 1 || quotedChecks != 2 {
		t.Fatalf("replicated schema default=%q comment=%q restrict_fks=%d quoted_checks=%d", splitDefault, rawTextComment, restrictFKs, quotedChecks)
	}
}

func assertConcurrentRawEvidenceConflict(t *testing.T, publication *evidencebiz.UseCase, db *sql.DB) {
	t.Helper()
	left := postgresEvidenceRaw("RAW_race_drift_00000000000000000")
	right := left
	right.RawText = "A different concurrent article."
	start := make(chan struct{})
	errorsChannel := make(chan error, 2)
	for _, input := range []evidencebiz.RawEvidence{left, right} {
		go func(raw evidencebiz.RawEvidence) {
			<-start
			_, err := publication.PublishRawEvidence(context.Background(), "neutral-publisher", raw)
			errorsChannel <- err
		}(input)
	}
	close(start)
	succeeded, conflicted := 0, 0
	for count := 0; count < 2; count++ {
		err := <-errorsChannel
		if err == nil {
			succeeded++
			continue
		}
		var conflict *evidencebiz.ConflictError
		if errors.As(err, &conflict) {
			conflicted++
			continue
		}
		t.Fatalf("concurrent Raw drift error = %v", err)
	}
	var rows, receipts int
	if err := db.QueryRow(`SELECT count(*) FROM raw_evidences WHERE raw_evidence_id = $1`, left.RawEvidenceID).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM raw_evidence_publication_receipts WHERE raw_evidence_id = $1`, left.RawEvidenceID).Scan(&receipts); err != nil {
		t.Fatal(err)
	}
	if succeeded != 1 || conflicted != 1 || rows != 1 || receipts != 1 {
		t.Fatalf("Raw drift convergence succeeded=%d conflicted=%d rows=%d receipts=%d", succeeded, conflicted, rows, receipts)
	}
}

func assertConcurrentEvidenceConvergenceAndConflict(t *testing.T, publication *evidencebiz.UseCase, db *sql.DB) {
	t.Helper()
	for _, test := range []struct {
		rawID         string
		evidenceID    string
		drift         bool
		wantCreated   int
		wantReused    int
		wantConflicts int
		wantReceipts  int
	}{
		{rawID: "RAW_evrace_ok_000000000000000000", evidenceID: "EVD_evrace_ok_000000000000000000", wantCreated: 1, wantReused: 1, wantReceipts: 2},
		{rawID: "RAW_evrace_no_000000000000000000", evidenceID: "EVD_evrace_no_000000000000000000", drift: true, wantCreated: 1, wantConflicts: 1, wantReceipts: 1},
	} {
		raw := postgresEvidenceRaw(test.rawID)
		if _, err := publication.PublishRawEvidence(context.Background(), "neutral-publisher", raw); err != nil {
			t.Fatal(err)
		}
		left := []evidencebiz.Evidence{postgresEvidence(test.evidenceID, 0)}
		right := []evidencebiz.Evidence{postgresEvidence(test.evidenceID, 0)}
		if test.drift {
			right[0].SourceWhat = "Concurrent semantic drift."
		}
		start := make(chan struct{})
		type outcome struct {
			result evidencebiz.EvidenceResult
			err    error
		}
		outcomes := make(chan outcome, 2)
		for _, input := range [][]evidencebiz.Evidence{left, right} {
			go func(items []evidencebiz.Evidence) {
				<-start
				result, err := publication.PublishEvidence(context.Background(), "neutral-publisher", raw.RawEvidenceID, items)
				outcomes <- outcome{result: result, err: err}
			}(input)
		}
		close(start)
		created, reused, conflicted := 0, 0, 0
		for count := 0; count < 2; count++ {
			completed := <-outcomes
			if completed.err == nil {
				created += completed.result.Counts.Created
				reused += completed.result.Counts.Reused
				continue
			}
			var conflict *evidencebiz.ConflictError
			if errors.As(completed.err, &conflict) {
				conflicted++
				continue
			}
			t.Fatalf("concurrent Evidence error = %v", completed.err)
		}
		var rows, receipts int
		if err := db.QueryRow(`SELECT count(*) FROM evidences WHERE raw_evidence_id = $1`, raw.RawEvidenceID).Scan(&rows); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow(`SELECT count(*) FROM evidence_publication_receipts WHERE raw_evidence_id = $1`, raw.RawEvidenceID).Scan(&receipts); err != nil {
			t.Fatal(err)
		}
		if created != test.wantCreated || reused != test.wantReused || conflicted != test.wantConflicts || rows != 1 || receipts != test.wantReceipts {
			t.Fatalf("Evidence convergence created=%d reused=%d conflicts=%d rows=%d receipts=%d", created, reused, conflicted, rows, receipts)
		}
	}
}

func assertConcurrentRawEvidenceConvergence(t *testing.T, publication *evidencebiz.UseCase, db *sql.DB) {
	t.Helper()
	raw := postgresEvidenceRaw("RAW_concurrent_00000000000000000")
	start := make(chan struct{})
	results := make(chan evidencebiz.RawEvidenceResult, 2)
	errorsChannel := make(chan error, 2)
	var workers sync.WaitGroup
	for index := 0; index < 2; index++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			result, err := publication.PublishRawEvidence(context.Background(), "neutral-publisher", raw)
			results <- result
			errorsChannel <- err
		}()
	}
	close(start)
	workers.Wait()
	close(results)
	close(errorsChannel)
	for err := range errorsChannel {
		if err != nil {
			t.Fatalf("concurrent publication: %v", err)
		}
	}
	created, reused := 0, 0
	for result := range results {
		switch result.RawEvidence.Disposition {
		case evidencebiz.DispositionCreated:
			created++
		case evidencebiz.DispositionReused:
			reused++
		}
	}
	var rows, receipts int
	if err := db.QueryRow(`SELECT count(*) FROM raw_evidences WHERE raw_evidence_id = $1`, raw.RawEvidenceID).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM raw_evidence_publication_receipts WHERE raw_evidence_id = $1`, raw.RawEvidenceID).Scan(&receipts); err != nil {
		t.Fatal(err)
	}
	if created != 1 || reused != 1 || rows != 1 || receipts != 2 {
		t.Fatalf("convergence created=%d reused=%d rows=%d receipts=%d", created, reused, rows, receipts)
	}
}

func assertEvidenceTransactionRollback(t *testing.T, publication *evidencebiz.UseCase, store evidencebiz.Store, db *sql.DB) {
	t.Helper()
	raw := postgresEvidenceRaw("RAW_rollback_0000000000000000000")
	err := store.InTransaction(context.Background(), func(tx evidencebiz.Transaction) error {
		if err := tx.InsertRawEvidence(context.Background(), evidencebiz.StoredRawEvidence{
			RawEvidence: raw, ContentHash: "unused-generated-column-value",
		}); err != nil {
			return err
		}
		return tx.InsertRawEvidenceReceipt(context.Background(), evidencebiz.RawEvidencePublicationReceipt{
			ID: "not-a-uuid", CallerSubject: "neutral-publisher", RawEvidenceID: raw.RawEvidenceID,
			Disposition: evidencebiz.DispositionCreated, ImportedAt: time.Now().UTC(),
		})
	})
	if err == nil {
		t.Fatal("transaction with invalid receipt ID unexpectedly succeeded")
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM raw_evidences WHERE raw_evidence_id = $1`, raw.RawEvidenceID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("rolled-back Raw Evidence count = %d", count)
	}

	evidenceRaw := postgresEvidenceRaw("RAW_evrollback_00000000000000000")
	if _, err := publication.PublishRawEvidence(context.Background(), "neutral-publisher", evidenceRaw); err != nil {
		t.Fatal(err)
	}
	evidence := postgresEvidence("EVD_evrollback_00000000000000000", 0)
	err = store.InTransaction(context.Background(), func(tx evidencebiz.Transaction) error {
		if err := tx.InsertEvidence(context.Background(), evidencebiz.StoredEvidence{
			Evidence: evidence, RawEvidenceID: evidenceRaw.RawEvidenceID, IsSplit: false,
		}); err != nil {
			return err
		}
		return tx.InsertEvidenceReceipt(context.Background(), evidencebiz.EvidencePublicationReceipt{
			ID: "not-a-uuid", CallerSubject: "neutral-publisher", RawEvidenceID: evidenceRaw.RawEvidenceID,
			EvidenceIDs: []string{evidence.EvidenceID}, Counts: evidencebiz.EvidenceCounts{Created: 1},
			ImportedAt: time.Now().UTC(),
		})
	})
	if err == nil {
		t.Fatal("partial Evidence transaction unexpectedly succeeded")
	}
	if err := db.QueryRow(`SELECT count(*) FROM evidences WHERE raw_evidence_id = $1`, evidenceRaw.RawEvidenceID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("rolled-back Evidence count = %d", count)
	}
}

func assertEvidenceDeadlineCancelsQueryAndRollsBack(t *testing.T, store Store, db *sql.DB) {
	t.Helper()
	lockKey := "raw-evidence:deadline-rollback"
	lockHolder, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := lockHolder.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			t.Errorf("release Evidence deadline test lock: %v", err)
		}
	}()
	if _, err := lockHolder.Exec(`SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, lockKey); err != nil {
		t.Fatal(err)
	}

	raw := postgresEvidenceRaw("RAW_deadline_0000000000000000000")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err = store.InTransaction(ctx, func(tx evidencebiz.Transaction) error {
		if err := tx.InsertRawEvidence(ctx, evidencebiz.StoredRawEvidence{RawEvidence: raw}); err != nil {
			return err
		}
		if err := tx.InsertRawEvidenceReceipt(ctx, evidencebiz.RawEvidencePublicationReceipt{
			ID: "33333333-3333-4333-8333-333333333333", CallerSubject: "neutral-publisher",
			RawEvidenceID: raw.RawEvidenceID, Disposition: evidencebiz.DispositionCreated,
			ImportedAt: time.Now().UTC(),
		}); err != nil {
			return err
		}
		return tx.LockIdentities(ctx, []string{lockKey})
	})
	if err == nil || !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatalf("deadline transaction error=%v context=%v", err, ctx.Err())
	}
	var rawCount, receiptCount int
	if err := db.QueryRow(`SELECT count(*) FROM raw_evidences WHERE raw_evidence_id = $1`, raw.RawEvidenceID).Scan(&rawCount); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM raw_evidence_publication_receipts WHERE raw_evidence_id = $1`, raw.RawEvidenceID).Scan(&receiptCount); err != nil {
		t.Fatal(err)
	}
	if rawCount != 0 || receiptCount != 0 {
		t.Fatalf("deadline rollback left raw=%d receipts=%d", rawCount, receiptCount)
	}
}

func assertEvidenceReceiptsImmutable(t *testing.T, db *sql.DB, rawReceiptID, evidenceReceiptID string) {
	t.Helper()
	if _, err := db.Exec(`UPDATE raw_evidence_publication_receipts SET caller_subject = 'mutated' WHERE id = $1`, rawReceiptID); err == nil {
		t.Fatal("Raw Evidence receipt update unexpectedly succeeded")
	}
	if _, err := db.Exec(`DELETE FROM evidence_publication_receipts WHERE id = $1`, evidenceReceiptID); err == nil {
		t.Fatal("Evidence receipt delete unexpectedly succeeded")
	}
}

func postgresEvidenceRaw(id string) evidencebiz.RawEvidence {
	publishedAt := time.Date(2026, 8, 11, 1, 0, 0, 123456789, time.UTC)
	return evidencebiz.RawEvidence{
		RawEvidenceID: id, SourceID: "SRC_postgres_0000000000000000000", SourceName: "Example Wire",
		SourceLevel: "L2_WIRE", SourceURL: "https://example.test/evidence", IsOriginal: true,
		RawText: "Complete PostgreSQL Evidence Publication article.", PublishedAt: &publishedAt,
		CollectedAt: time.Date(2026, 8, 11, 1, 5, 0, 987654321, time.UTC),
		Keywords:    []string{" AI芯片 ", "供应链", "AI芯片"},
	}
}

func postgresEvidence(id string, order int) evidencebiz.Evidence {
	return evidencebiz.Evidence{
		EvidenceID: id, SplitOrder: order, LayerType: "SINGLE",
		SourceWhat:            "Example Corp expanded production.",
		ExpressionFingerprint: "Example Corp expands production",
		ExpressionKey:         "shared-expression-key-v1", FingerprintVersion: "evidence-expression.v1",
	}
}

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
