package seed

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func TestLoadChainNodeRelationManifestRejectsLegacyFieldsAndTypes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "relations.json")
	content := `{"relations":[{"id":"r","from_chain_node_entity_id":"a","to_chain_node_entity_id":"b","relation_type":"contains","mechanism":"分类","evidence_note":"定义证据","provenance":"review","verified_at":"2026-07-14T00:00:00Z","status":"active","topology_edge_id":"legacy"}]}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadChainNodeRelationManifest(path); err == nil {
		t.Fatal("LoadChainNodeRelationManifest() error = nil")
	}
}

func TestLoadChainNodeRelationManifestDecodesSnakeCaseContract(t *testing.T) {
	path := filepath.Join(t.TempDir(), "relations.json")
	content := `{"relations":[{"id":"r","from_chain_node_entity_id":"a","to_chain_node_entity_id":"b","relation_type":"is_subcategory_of","mechanism":"定义范围从属","condition_note":"稳定语义范围","evidence_note":"定义证据","provenance":"review","verified_at":"2026-07-14T00:00:00Z","status":"active"}]}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := LoadChainNodeRelationManifest(path)
	if err != nil || len(manifest.Relations) != 1 || manifest.Relations[0].FromChainNodeEntityID != "a" {
		t.Fatalf("manifest=%+v err=%v", manifest, err)
	}
}

func TestChainNodeRelationDryRunReportsCreateUnchangedUpdateAndConflict(t *testing.T) {
	now := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	base := domain.ChainNodeRelation{ID: "r1", FromChainNodeEntityID: "a", ToChainNodeEntityID: "b", RelationType: domain.ChainNodeRelationSubcategoryOf, Mechanism: "定义范围从属", EvidenceNote: "定义与边界直接支持", Provenance: "approved-node-manifest", VerifiedAt: now, Status: domain.StatusActive}
	repo := newChainNodeRelationMemoryRepository([]string{"a", "b", "c"})
	report, err := DryRunChainNodeRelations(context.Background(), repo, []domain.ChainNodeRelation{base})
	if err != nil || report.Created != 1 {
		t.Fatalf("create report = %+v, %v", report, err)
	}
	if _, err := repo.UpsertChainNodeRelation(context.Background(), base); err != nil {
		t.Fatal(err)
	}
	report, err = DryRunChainNodeRelations(context.Background(), repo, []domain.ChainNodeRelation{base})
	if err != nil || report.Unchanged != 1 {
		t.Fatalf("unchanged report = %+v, %v", report, err)
	}
	updated := base
	updated.EvidenceNote = "补充定义证据"
	report, err = DryRunChainNodeRelations(context.Background(), repo, []domain.ChainNodeRelation{updated})
	if err != nil || report.Updated != 1 {
		t.Fatalf("updated report = %+v, %v", report, err)
	}
	conflict := base
	conflict.FromChainNodeEntityID = "c"
	if _, err := DryRunChainNodeRelations(context.Background(), repo, []domain.ChainNodeRelation{conflict}); err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("conflict error = %v", err)
	}
	tupleConflict := base
	tupleConflict.ID = "r2"
	if _, err := DryRunChainNodeRelations(context.Background(), repo, []domain.ChainNodeRelation{tupleConflict}); err == nil || !strings.Contains(err.Error(), "tuple conflict") {
		t.Fatalf("tuple conflict error = %v", err)
	}
}

func TestChainNodeRelationDryRunRejectsInactiveOrMissingEndpoint(t *testing.T) {
	relation := domain.ChainNodeRelation{ID: "r", FromChainNodeEntityID: "a", ToChainNodeEntityID: "missing", RelationType: domain.ChainNodeRelationInputTo, Mechanism: "直接投入", EvidenceNote: "证据", Provenance: "review", VerifiedAt: time.Now(), Status: domain.StatusActive}
	_, err := DryRunChainNodeRelations(context.Background(), newChainNodeRelationMemoryRepository([]string{"a"}), []domain.ChainNodeRelation{relation})
	if err == nil || !strings.Contains(err.Error(), "active chain_node") {
		t.Fatalf("error = %v", err)
	}
}

func TestPostgresChainNodeRelationDryRunUsesRepeatableReadOnlySnapshot(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	relation := domain.ChainNodeRelation{ID: "r", FromChainNodeEntityID: "a", ToChainNodeEntityID: "b", RelationType: domain.ChainNodeRelationSubcategoryOf, Mechanism: "定义范围从属", EvidenceNote: "定义证据", Provenance: "review", VerifiedAt: time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC), Status: domain.StatusActive}
	mock.ExpectBegin()
	for _, endpoint := range []string{"a", "b"} {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS (SELECT 1 FROM entity_nodes WHERE id=$1::uuid AND entity_type='chain_node' AND status='active')")).WithArgs(endpoint).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	}
	mock.ExpectQuery("FROM chain_node_relations WHERE id").WithArgs("r").WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("FROM chain_node_relations WHERE from_chain_node_entity_id").WithArgs("a", "b", domain.ChainNodeRelationSubcategoryOf).WillReturnError(sql.ErrNoRows)
	mock.ExpectCommit()
	report, err := NewPostgresRepository(db).DryRunChainNodeRelationBatch(context.Background(), []domain.ChainNodeRelation{relation})
	if err != nil || report.Created != 1 {
		t.Fatalf("report=%+v err=%v", report, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
