package evidence

import (
	"context"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

func TestRecoverPreV74EvidenceClearsOnlyEvidenceDataset(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(preV74RecoveryLockSQL)).
		WithArgs(preV74RecoveryLockName).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(preV74MigrationVersionSQL)).
		WillReturnRows(sqlmock.NewRows([]string{"version_id", "is_applied"}).AddRow(int64(73), true))
	mock.ExpectQuery(regexp.QuoteMeta(preV74CountsSQL)).
		WillReturnRows(preV74CountRows(2, 4, 2, 0, 0, 0, 0, 0))
	mock.ExpectExec(regexp.QuoteMeta(deleteRawEvidenceCategoryLinksSQL)).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(regexp.QuoteMeta(deleteEvidenceSQL)).
		WillReturnResult(sqlmock.NewResult(0, 4))
	mock.ExpectExec(regexp.QuoteMeta(deleteRawEvidenceSQL)).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectQuery(regexp.QuoteMeta(preV74CountsSQL)).
		WillReturnRows(preV74CountRows(0, 0, 0, 0, 0, 0, 0, 0))
	mock.ExpectCommit()

	report, err := RecoverPreV74Evidence(context.Background(), db, true)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Applied || report.MigrationVersion != 73 {
		t.Fatalf("unexpected report: %+v", report)
	}
	if report.Before.RawEvidences != 2 || report.Before.Evidences != 4 || report.Before.RawEvidenceCategoryLinks != 2 {
		t.Fatalf("unexpected before counts: %+v", report.Before)
	}
	if !report.After.empty() {
		t.Fatalf("unexpected after counts: %+v", report.After)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRecoverPreV74EvidenceCheckOnlyDoesNotDelete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(preV74RecoveryLockSQL)).
		WithArgs(preV74RecoveryLockName).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(preV74MigrationVersionSQL)).
		WillReturnRows(sqlmock.NewRows([]string{"version_id", "is_applied"}).AddRow(int64(73), true))
	mock.ExpectQuery(regexp.QuoteMeta(preV74CountsSQL)).
		WillReturnRows(preV74CountRows(2, 4, 2, 0, 0, 0, 0, 0))
	mock.ExpectRollback()

	report, err := RecoverPreV74Evidence(context.Background(), db, false)
	if err != nil {
		t.Fatal(err)
	}
	if report.Applied || report.Before != report.After {
		t.Fatalf("unexpected check-only report: %+v", report)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRecoverPreV74EvidenceRejectsUnexpectedEventDataset(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(preV74RecoveryLockSQL)).
		WithArgs(preV74RecoveryLockName).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(preV74MigrationVersionSQL)).
		WillReturnRows(sqlmock.NewRows([]string{"version_id", "is_applied"}).AddRow(int64(73), true))
	mock.ExpectQuery(regexp.QuoteMeta(preV74CountsSQL)).
		WillReturnRows(preV74CountRows(2, 4, 2, 1, 1, 0, 0, 1))
	mock.ExpectRollback()

	_, err = RecoverPreV74Evidence(context.Background(), db, true)
	if err == nil || !strings.Contains(err.Error(), "Event dataset") {
		t.Fatalf("error = %v, want Event dataset guard", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRecoverPreV74EvidenceRequiresMigration73(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(preV74RecoveryLockSQL)).
		WithArgs(preV74RecoveryLockName).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(preV74MigrationVersionSQL)).
		WillReturnRows(sqlmock.NewRows([]string{"version_id", "is_applied"}).AddRow(int64(72), true))
	mock.ExpectRollback()

	_, err = RecoverPreV74Evidence(context.Background(), db, true)
	if err == nil || !strings.Contains(err.Error(), "migration 73") {
		t.Fatalf("error = %v, want migration guard", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRecoverPreV74EvidenceAllowsMigration74AfterCleanup(t *testing.T) {
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	db := postgresfixture.OpenIsolated(t, "tw_pre_v74_evidence_recovery", migrationDir, 73)
	ctx := context.Background()

	var categoryID string
	if err := db.QueryRowContext(ctx, `SELECT id FROM evidence_categories ORDER BY id LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO raw_evidences (id, source_id, source_name, source_level, source_url, is_original, raw_text, collected_at)
VALUES ('RAW11111111-1111-4111-8111-111111111111', 'SRC-RECOVERY-TEST', 'Recovery Test', 'L1_OFFICIAL', 'https://example.test/recovery', true, 'legacy raw evidence', now())`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO evidences (id, raw_evidence_id, is_split, summary, semantic)
VALUES (
    'EVD11111111-1111-4111-8111-111111111111',
    'RAW11111111-1111-4111-8111-111111111111',
    false,
    'legacy evidence',
    '{"who":"Example","what":"legacy evidence","when":null,"where":null,"why":null,"how":null}'
)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO raw_evidence_category_links (id, raw_evidence_id, category_id)
VALUES ('RCL11111111-1111-4111-8111-111111111111', 'RAW11111111-1111-4111-8111-111111111111', $1)`, categoryID); err != nil {
		t.Fatal(err)
	}

	report, err := RecoverPreV74Evidence(ctx, db, true)
	if err != nil {
		t.Fatal(err)
	}
	if report.Before.RawEvidences != 1 || report.Before.Evidences != 1 || report.Before.RawEvidenceCategoryLinks != 1 || !report.After.empty() {
		t.Fatalf("unexpected recovery report: %+v", report)
	}
	var categories int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM evidence_categories`).Scan(&categories); err != nil {
		t.Fatal(err)
	}
	if categories == 0 {
		t.Fatal("Evidence Category Catalog was unexpectedly cleared")
	}

	postgresfixture.ApplyMigration(t, db, migrationDir, 74)
}

func preV74CountRows(raw, atomic, categoryLinks, events, evidenceLinks, actorLinks, assetLinks, receipts int64) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"raw_evidences", "evidences", "raw_evidence_category_links", "events",
		"event_evidence_links", "event_actor_links", "event_asset_links", "event_publication_receipts",
	}).AddRow(raw, atomic, categoryLinks, events, evidenceLinks, actorLinks, assetLinks, receipts)
}
