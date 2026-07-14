package seed

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestApplyExternalIdentifierBatchRollsBackBeforeWritesWhenTargetIsInvalid(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	item := firstBatchExternalIdentifier("chain_node:3d_printing", "eastmoney", "concept_sector", "BK0619", "3D打印")
	mock.ExpectBegin()
	mock.ExpectExec("pg_advisory_xact_lock").WithArgs(externalIdentifierIdentity(item.SourceSystem, item.SourceTaxonomyType, item.ExternalCode)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT id FROM entity_nodes").WithArgs(item.EntityID).WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectRollback()
	if _, err := NewPostgresRepository(db).ApplyExternalIdentifierBatch(context.Background(), []ExternalIdentifierMapping{mappingFromIdentifier(item)}); err == nil {
		t.Fatal("ApplyExternalIdentifierBatch() error = nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLoadExternalIdentifierMappingFileDecodesSnakeCaseAndRejectsDuplicateTriple(t *testing.T) {
	item := firstBatchExternalIdentifier("chain_node:3d_printing", "eastmoney", "concept_sector", "BK0619", "3D打印")
	path := filepath.Join(t.TempDir(), "mappings.json")
	content := `{"mappings":[{"id":"` + item.ID + `","entity_id":"` + item.EntityID + `","source_system":"eastmoney","source_taxonomy_type":"concept_sector","external_code":"BK0619","external_name":"3D打印","status":"active"}]}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := LoadExternalIdentifierMappingFile(path)
	if err != nil || len(manifest.Mappings) != 1 || manifest.Mappings[0].EntityID != item.EntityID {
		t.Fatalf("LoadExternalIdentifierMappingFile() = %+v, %v", manifest, err)
	}
	duplicate := strings.TrimSuffix(content, `]}`) + `,` + strings.TrimPrefix(content, `{"mappings":[`)
	if err := os.WriteFile(path, []byte(duplicate), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadExternalIdentifierMappingFile(path); err == nil || !strings.Contains(err.Error(), "duplicate external identifier") {
		t.Fatalf("duplicate manifest error = %v", err)
	}
}
