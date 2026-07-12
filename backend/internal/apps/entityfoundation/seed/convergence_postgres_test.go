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
	for _, fragment := range []string{"$9::text", "$10::text"} {
		if !strings.Contains(statement, fragment) {
			t.Fatalf("audit SQL missing cast %q", fragment)
		}
	}
	if got := args[3]; got != domain.EntityTypeIndex {
		t.Fatalf("target type arg = %#v", got)
	}
	if got := args[7]; got != item.Reason {
		t.Fatalf("reason arg = %#v", got)
	}
}

func TestPostgresCorrectionRedirectsRecordedSectorEdgeToIndex(t *testing.T) {
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
	previousID, currentID := "11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222"
	oldID, newID := entitySeedUUID("sector:old"), entitySeedUUID("index:new")
	edgeID := "33333333-3333-3333-3333-333333333333"
	mock.ExpectQuery("SELECT target_entity_id, target_entity_type FROM entity_convergences").WithArgs(previousID).WillReturnRows(sqlmock.NewRows([]string{"target_entity_id", "target_entity_type"}).AddRow(oldID, domain.EntityTypeSector))
	mock.ExpectQuery("SELECT reference_table,reference_column,reference_row_id,from_entity_id,to_entity_id").WithArgs(previousID).WillReturnRows(sqlmock.NewRows([]string{"reference_table", "reference_column", "reference_row_id", "from_entity_id", "to_entity_id"}).AddRow("entity_edges", "to_entity_id", edgeID, entitySeedUUID("sector:legacy"), oldID))
	mock.ExpectQuery("SELECT e.id, fn.id, fn.entity_key, fn.entity_type, tn.id, tn.entity_key, tn.entity_type").WithArgs(edgeID).WillReturnRows(correctionEdgeRows().AddRow(edgeID, entitySeedUUID("market:test"), "market:test", domain.EntityTypeMarket, oldID, "sector:old", domain.EntityTypeSector, "tracks_index", "reviewed", "review", "https://example.com/review", time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC), domain.StatusActive))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE entity_edges SET to_entity_id=$1,status='active',updated_at=NOW() WHERE id=$2 AND to_entity_id=$3 AND status=$4")).WithArgs(newID, edgeID, oldID, domain.StatusActive).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO entity_convergence_reference_moves").WithArgs(sqlmock.AnyArg(), currentID, "entity_edges", "to_entity_id", edgeID, oldID, newID, "forward_correction_redirect").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT alias,to_entity_id FROM entity_convergence_alias_moves").WithArgs(previousID).WillReturnRows(sqlmock.NewRows([]string{"alias", "to_entity_id"}))
	counts, err := applyRecordedCorrectionMoves(context.Background(), tx, previousID, currentID, domain.EntityTypeIndex, newID, "index:new")
	if err != nil {
		t.Fatal(err)
	}
	if counts.references != 1 {
		t.Fatalf("counts = %+v", counts)
	}
	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresCorrectionReactivatesRecordedNoTargetEdgeToIndex(t *testing.T) {
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
	previousID, currentID := "11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222"
	legacyID, newID := entitySeedUUID("sector:legacy"), entitySeedUUID("index:new")
	edgeID := "33333333-3333-3333-3333-333333333333"
	mock.ExpectQuery("SELECT target_entity_id, target_entity_type FROM entity_convergences").WithArgs(previousID).WillReturnRows(sqlmock.NewRows([]string{"target_entity_id", "target_entity_type"}).AddRow(nil, nil))
	mock.ExpectQuery("SELECT reference_table,reference_column,reference_row_id,from_entity_id,to_entity_id").WithArgs(previousID).WillReturnRows(sqlmock.NewRows([]string{"reference_table", "reference_column", "reference_row_id", "from_entity_id", "to_entity_id"}).AddRow("entity_edges", "status", edgeID, legacyID, nil))
	mock.ExpectQuery("SELECT e.id, fn.id, fn.entity_key, fn.entity_type, tn.id, tn.entity_key, tn.entity_type").WithArgs(edgeID).WillReturnRows(correctionEdgeRows().AddRow(edgeID, entitySeedUUID("market:test"), "market:test", domain.EntityTypeMarket, legacyID, "sector:legacy", domain.EntityTypeSector, "tracks_index", "reviewed", "review", "https://example.com/review", time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC), domain.StatusInactive))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE entity_edges SET to_entity_id=$1,status='active',updated_at=NOW() WHERE id=$2 AND to_entity_id=$3 AND status=$4")).WithArgs(newID, edgeID, legacyID, domain.StatusInactive).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO entity_convergence_reference_moves").WithArgs(sqlmock.AnyArg(), currentID, "entity_edges", "to_entity_id", edgeID, legacyID, newID, "forward_correction_reactivate_redirect").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT alias,to_entity_id FROM entity_convergence_alias_moves").WithArgs(previousID).WillReturnRows(sqlmock.NewRows([]string{"alias", "to_entity_id"}))
	counts, err := applyRecordedCorrectionMoves(context.Background(), tx, previousID, currentID, domain.EntityTypeIndex, newID, "index:new")
	if err != nil {
		t.Fatal(err)
	}
	if counts.references != 1 {
		t.Fatalf("counts = %+v", counts)
	}
	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresCorrectionDeactivatesRecordedTargetEdgeForNoTarget(t *testing.T) {
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
	previousID, currentID := "11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222"
	oldID := entitySeedUUID("sector:old")
	edgeID := "33333333-3333-3333-3333-333333333333"
	mock.ExpectQuery("SELECT target_entity_id, target_entity_type FROM entity_convergences").WithArgs(previousID).WillReturnRows(sqlmock.NewRows([]string{"target_entity_id", "target_entity_type"}).AddRow(oldID, domain.EntityTypeSector))
	mock.ExpectQuery("SELECT reference_table,reference_column,reference_row_id,from_entity_id,to_entity_id").WithArgs(previousID).WillReturnRows(sqlmock.NewRows([]string{"reference_table", "reference_column", "reference_row_id", "from_entity_id", "to_entity_id"}).AddRow("entity_edges", "to_entity_id", edgeID, entitySeedUUID("sector:legacy"), oldID))
	mock.ExpectQuery("SELECT e.id, fn.id, fn.entity_key, fn.entity_type, tn.id, tn.entity_key, tn.entity_type").WithArgs(edgeID).WillReturnRows(correctionEdgeRows().AddRow(edgeID, entitySeedUUID("market:test"), "market:test", domain.EntityTypeMarket, oldID, "sector:old", domain.EntityTypeSector, "covers_sector", "reviewed", "review", "https://example.com/review", time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC), domain.StatusActive))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE entity_edges SET status='inactive',updated_at=NOW() WHERE id=$1 AND status=$2")).WithArgs(edgeID, domain.StatusActive).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO entity_convergence_reference_moves").WithArgs(sqlmock.AnyArg(), currentID, "entity_edges", "status", edgeID, oldID, nil, "forward_correction_deactivate").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT alias,to_entity_id FROM entity_convergence_alias_moves").WithArgs(previousID).WillReturnRows(sqlmock.NewRows([]string{"alias", "to_entity_id"}))
	counts, err := applyRecordedCorrectionMoves(context.Background(), tx, previousID, currentID, "", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if counts.references != 1 {
		t.Fatalf("counts = %+v", counts)
	}
	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresCorrectionIncompatiblePolicyRollsBack(t *testing.T) {
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
	previousID, currentID := "11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222"
	oldID, newID := entitySeedUUID("sector:old"), entitySeedUUID("index:new")
	edgeID := "33333333-3333-3333-3333-333333333333"
	mock.ExpectQuery("SELECT target_entity_id, target_entity_type FROM entity_convergences").WithArgs(previousID).WillReturnRows(sqlmock.NewRows([]string{"target_entity_id", "target_entity_type"}).AddRow(oldID, domain.EntityTypeSector))
	mock.ExpectQuery("SELECT reference_table,reference_column,reference_row_id,from_entity_id,to_entity_id").WithArgs(previousID).WillReturnRows(sqlmock.NewRows([]string{"reference_table", "reference_column", "reference_row_id", "from_entity_id", "to_entity_id"}).AddRow("entity_edges", "to_entity_id", edgeID, entitySeedUUID("sector:legacy"), oldID))
	mock.ExpectQuery("SELECT e.id, fn.id, fn.entity_key, fn.entity_type, tn.id, tn.entity_key, tn.entity_type").WithArgs(edgeID).WillReturnRows(correctionEdgeRows().AddRow(edgeID, entitySeedUUID("market:test"), "market:test", domain.EntityTypeMarket, oldID, "sector:old", domain.EntityTypeSector, "covers_sector", "reviewed", "review", "https://example.com/review", time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC), domain.StatusActive))
	if _, err := applyRecordedCorrectionMoves(context.Background(), tx, previousID, currentID, domain.EntityTypeIndex, newID, "index:new"); err == nil || !strings.Contains(err.Error(), "incompatible") {
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

func TestPostgresCorrectionDriftRollsBack(t *testing.T) {
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
	previousID, currentID := "11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222"
	oldID, newID := entitySeedUUID("sector:old"), entitySeedUUID("sector:new")
	edgeID := "33333333-3333-3333-3333-333333333333"
	mock.ExpectQuery("SELECT target_entity_id, target_entity_type FROM entity_convergences").WithArgs(previousID).WillReturnRows(sqlmock.NewRows([]string{"target_entity_id", "target_entity_type"}).AddRow(oldID, domain.EntityTypeSector))
	mock.ExpectQuery("SELECT reference_table,reference_column,reference_row_id,from_entity_id,to_entity_id").WithArgs(previousID).WillReturnRows(sqlmock.NewRows([]string{"reference_table", "reference_column", "reference_row_id", "from_entity_id", "to_entity_id"}).AddRow("entity_edges", "to_entity_id", edgeID, entitySeedUUID("sector:legacy"), oldID))
	mock.ExpectQuery("SELECT e.id, fn.id, fn.entity_key, fn.entity_type, tn.id, tn.entity_key, tn.entity_type").WithArgs(edgeID).WillReturnRows(correctionEdgeRows().AddRow(edgeID, entitySeedUUID("market:test"), "market:test", domain.EntityTypeMarket, oldID, "sector:old", domain.EntityTypeSector, "covers_sector", "reviewed", "review", "https://example.com/review", time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC), domain.StatusInactive))
	if _, err := applyRecordedCorrectionMoves(context.Background(), tx, previousID, currentID, domain.EntityTypeSector, newID, "sector:new"); err == nil || !strings.Contains(err.Error(), "drifted") {
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

func correctionEdgeRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "from_id", "from_key", "from_type", "to_id", "to_key", "to_type", "relation_type", "evidence_note", "source_name", "source_url", "verified_at", "status"})
}
