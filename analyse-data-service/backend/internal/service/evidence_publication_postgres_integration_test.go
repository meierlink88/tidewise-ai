package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/evidencepublication"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/postgres"
)

func TestPostgresEvidencePublicationNaturalIdentityTransactionsAndSchema(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	store := postgres.NewEvidencePublicationStore(db)
	publication := evidencepublication.NewService(store)
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
	if created.RawEvidence.Disposition != evidencepublication.DispositionCreated ||
		replayed.RawEvidence.Disposition != evidencepublication.DispositionReused ||
		created.ReceiptID == replayed.ReceiptID {
		t.Fatalf("Raw Evidence results created=%#v replayed=%#v", created, replayed)
	}

	items := []evidencepublication.Evidence{
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

	drift := append([]evidencepublication.Evidence(nil), items...)
	drift[0].SourceWhat = "drifted"
	_, err = publication.PublishEvidence(ctx, "neutral-publisher", raw.RawEvidenceID, drift)
	var conflict *evidencepublication.ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("drift error = %v, want ConflictError", err)
	}

	assertConcurrentRawEvidenceConvergence(t, publication, db)
	assertEvidenceTransactionRollback(t, store, db)
	assertEvidenceReceiptsImmutable(t, db, created.ReceiptID, published.ReceiptID)
}

func assertConcurrentRawEvidenceConvergence(t *testing.T, publication *evidencepublication.Service, db *sql.DB) {
	t.Helper()
	raw := postgresEvidenceRaw("RAW_concurrent_00000000000000000")
	start := make(chan struct{})
	results := make(chan evidencepublication.RawEvidenceResult, 2)
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
		case evidencepublication.DispositionCreated:
			created++
		case evidencepublication.DispositionReused:
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

func assertEvidenceTransactionRollback(t *testing.T, store evidencepublication.Store, db *sql.DB) {
	t.Helper()
	raw := postgresEvidenceRaw("RAW_rollback_0000000000000000000")
	err := store.InTransaction(context.Background(), func(tx evidencepublication.Transaction) error {
		if err := tx.InsertRawEvidence(context.Background(), evidencepublication.StoredRawEvidence{
			RawEvidence: raw, ContentHash: "unused-generated-column-value",
		}); err != nil {
			return err
		}
		return tx.InsertRawEvidenceReceipt(context.Background(), evidencepublication.RawEvidencePublicationReceipt{
			ID: "not-a-uuid", CallerSubject: "neutral-publisher", RawEvidenceID: raw.RawEvidenceID,
			Disposition: evidencepublication.DispositionCreated, ImportedAt: time.Now().UTC(),
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

func postgresEvidenceRaw(id string) evidencepublication.RawEvidence {
	publishedAt := time.Date(2026, 8, 11, 1, 0, 0, 123456789, time.UTC)
	return evidencepublication.RawEvidence{
		RawEvidenceID: id, SourceID: "SRC_postgres_0000000000000000000", SourceName: "Example Wire",
		SourceLevel: "L2_WIRE", SourceURL: "https://example.test/evidence", IsOriginal: true,
		RawText: "Complete PostgreSQL Evidence Publication article.", PublishedAt: &publishedAt,
		CollectedAt: time.Date(2026, 8, 11, 1, 5, 0, 987654321, time.UTC),
		Keywords:    []string{" AI芯片 ", "供应链", "AI芯片"},
	}
}

func postgresEvidence(id string, order int) evidencepublication.Evidence {
	return evidencepublication.Evidence{
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
