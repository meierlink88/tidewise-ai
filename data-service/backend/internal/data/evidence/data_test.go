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
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	evidencebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/evidence"
	"github.com/pressly/goose/v3"
)

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
	created, err := publication.PublishRawEvidence(ctx, raw)
	if err != nil {
		t.Fatalf("publish Raw Evidence: %v", err)
	}
	rawCreatedAt := storedCreationTime(t, db, "raw_evidences", "raw_evidence_id", raw.RawEvidenceID)
	replayed, err := publication.PublishRawEvidence(ctx, raw)
	if err != nil {
		t.Fatalf("replay Raw Evidence: %v", err)
	}
	if replayedCreatedAt := storedCreationTime(t, db, "raw_evidences", "raw_evidence_id", raw.RawEvidenceID); !replayedCreatedAt.Equal(rawCreatedAt) {
		t.Fatalf("replayed Raw Evidence created_at = %s, want %s", replayedCreatedAt, rawCreatedAt)
	}
	if created.RawEvidenceID != raw.RawEvidenceID || replayed != created {
		t.Fatalf("Raw Evidence results created=%#v replayed=%#v", created, replayed)
	}

	items := []evidencebiz.Evidence{
		postgresEvidence("EVDe29312f1-33fb-5d44-8cfb-2b455b50533b", 0),
		postgresEvidence("EVD8fea9496-3764-53c2-ab57-5b1ff87b7581", 1),
	}
	items[1].SourceWhat = "A second source statement supports the same normalized fact."
	published, err := publication.PublishEvidence(ctx, raw.RawEvidenceID, items)
	if err != nil {
		t.Fatalf("publish Evidence set: %v", err)
	}
	evidenceCreatedAt := make(map[string]time.Time, len(items))
	for _, item := range items {
		evidenceCreatedAt[item.EvidenceID] = storedCreationTime(t, db, "evidences", "evidence_id", item.EvidenceID)
	}
	reused, err := publication.PublishEvidence(ctx, raw.RawEvidenceID, items)
	if err != nil {
		t.Fatalf("replay Evidence set: %v", err)
	}
	for _, item := range items {
		if replayedCreatedAt := storedCreationTime(t, db, "evidences", "evidence_id", item.EvidenceID); !replayedCreatedAt.Equal(evidenceCreatedAt[item.EvidenceID]) {
			t.Fatalf("replayed Evidence %q created_at = %s, want %s", item.EvidenceID, replayedCreatedAt, evidenceCreatedAt[item.EvidenceID])
		}
	}
	if published.RawEvidenceID != raw.RawEvidenceID || !sameTestStrings(published.EvidenceIDs, reused.EvidenceIDs) || len(published.EvidenceIDs) != 2 {
		t.Fatalf("Evidence results published=%#v reused=%#v", published, reused)
	}

	var keywords []string
	var keywordsJSON []byte
	var evidenceCount int
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

	drift := append([]evidencebiz.Evidence(nil), items...)
	drift[0].SourceWhat = "drifted"
	_, err = publication.PublishEvidence(ctx, raw.RawEvidenceID, drift)
	var conflict *evidencebiz.ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("drift error = %v, want ConflictError", err)
	}

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
			"raw_evidence_id", "source_id", "source_name", "source_level", "source_url", "is_original",
			"quoted_source_id", "quoted_source_name", "title", "raw_text", "published_at", "collected_at",
			"content_hash", "keywords",
		}).AddRow(
			rawEvidenceID, "SRC_0000000000000000000000000000", "Source", "INVALID",
			"https://example.test/article", true, nil, nil, nil, "hello", nil,
			time.Date(2026, 8, 11, 1, 2, 3, 0, time.UTC),
			"2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824", []byte(`[]`),
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
	rows := sqlmock.NewRows([]string{
		"evidence_id", "raw_evidence_id", "split_order", "is_split", "layer_type",
		"source_who", "source_what", "source_when", "source_when_raw", "source_where", "source_why", "source_how",
		"source_who_core", "source_what_core", "source_when_core", "source_when_raw_core",
		"source_where_core", "source_why_core", "source_how_core",
		"expression_fingerprint", "expression_key", "fingerprint_version",
	})
	rows.AddRow(persistedEvidenceRow("EVDc8222fc3-a24f-5d44-b204-09dfb2b8960f", rawEvidenceID, 0, "first fact")...)
	rows.AddRow(persistedEvidenceRow("EVD0f10cab3-e6ca-5bbc-ac33-5b09d3ff1602", rawEvidenceID, 2, "second fact")...)
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
	if errors.Is(err, accepted) || !errors.As(err, &invariantErr) || invariantErr.field != "split_order" {
		t.Fatalf("EvidencesByRawEvidence() error = %v, want persisted split_order invariant error", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func persistedEvidenceRow(id, rawEvidenceID string, splitOrder int, sourceWhat string) []driver.Value {
	return []driver.Value{
		id, rawEvidenceID, splitOrder, true, "SINGLE",
		nil, sourceWhat, nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil, nil, nil,
		sourceWhat + " normalized", id + "-key", "v1",
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
