package seed

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func TestPostgresIndexTargetRedirectsPolicyCompatibleGenericEdge(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	legacyID, targetID := entitySeedUUID("sector:legacy"), entitySeedUUID("index:csi300")
	verifiedAt := time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT e.id, fn.entity_key, fn.entity_type, tn.entity_key, tn.entity_type,")).
		WithArgs(legacyID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "from_key", "from_type", "to_key", "to_type", "relation_type", "evidence_note", "source_name", "source_url", "verified_at", "status"}).
			AddRow("11111111-1111-1111-1111-111111111111", "market:test", domain.EntityTypeMarket, "sector:legacy", domain.EntityTypeSector, "tracks_index", "reviewed identity correction", "review", "https://example.com/review", verifiedAt, domain.StatusActive))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE entity_edges SET to_entity_id=$1,updated_at=NOW() WHERE id=$2 AND to_entity_id=$3")).
		WithArgs(targetID, "11111111-1111-1111-1111-111111111111", legacyID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO entity_convergence_reference_moves").WillReturnResult(sqlmock.NewResult(0, 1))
	moved, err := redirectLegacyEdgesToIndex(context.Background(), tx, "22222222-2222-2222-2222-222222222222", legacyID, "sector:legacy", targetID, "index:csi300")
	if err != nil {
		t.Fatal(err)
	}
	if moved != 1 {
		t.Fatalf("moved = %d", moved)
	}
	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresNoTargetDeactivatesActiveEdge(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	legacyID := entitySeedUUID("sector:legacy")
	mock.ExpectQuery(regexp.QuoteMeta("UPDATE entity_edges SET status='inactive',updated_at=NOW() WHERE status='active' AND (from_entity_id=$1 OR to_entity_id=$1) RETURNING id")).WithArgs(legacyID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("33333333-3333-3333-3333-333333333333"))
	mock.ExpectExec("INSERT INTO entity_convergence_reference_moves").WillReturnResult(sqlmock.NewResult(0, 1))
	moved, err := deactivateLegacyEdges(context.Background(), tx, "22222222-2222-2222-2222-222222222222", legacyID)
	if err != nil {
		t.Fatal(err)
	}
	if moved != 1 {
		t.Fatalf("moved = %d", moved)
	}
	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresIndexTargetBlocksSectorOnlyForeignKey(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT tc.table_name, kcu.column_name FROM information_schema.table_constraints").
		WillReturnRows(sqlmock.NewRows([]string{"table_name", "column_name"}).AddRow("sector_source_mappings", "sector_entity_id"))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS (SELECT 1 FROM "sector_source_mappings" WHERE "sector_entity_id"=$1)`)).WithArgs(sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	manifest := SectorConvergenceManifest{Convergences: []SectorConvergence{{LegacyEntityKey: "sector:legacy", TargetEntityKey: "index:csi300", TargetEntityType: domain.EntityTypeIndex}}}
	err = preflightSectorReferences(context.Background(), tx, manifest, NewSectorReferenceRegistry())
	if err == nil || !strings.Contains(err.Error(), "sector-only") {
		t.Fatalf("error = %v", err)
	}
	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresConvergenceAuditSnapshotsTargetTypeAndReason(t *testing.T) {
	item := SectorConvergence{LegacyTaxonomy: "index", Action: SectorConvergenceReplaceWithExistingIndex, TargetEntityType: domain.EntityTypeIndex, Reason: "reviewed equivalent index identity"}
	manifest := SectorConvergenceManifest{ManifestVersion: 1, ManifestChecksum: "checksum"}
	statement, args := buildConvergenceAuditInsert("audit-id", "legacy-id", "target-id", manifest, item, SectorConvergenceModeInitial)
	for _, fragment := range []string{"target_entity_type", "reason", "mutation_provenance"} {
		if !strings.Contains(statement, fragment) {
			t.Fatalf("audit SQL missing %q", fragment)
		}
	}
	if got := args[3]; got != domain.EntityTypeIndex {
		t.Fatalf("target type arg = %#v", got)
	}
	if got := args[7]; got != item.Reason {
		t.Fatalf("reason arg = %#v", got)
	}
}
