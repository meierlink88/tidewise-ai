package seed

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func TestFrozenChainNodeRelationManifestMatchesApprovedHundredRelationArtifact(t *testing.T) {
	path := frozenChainNodeRelationManifestPath(t)
	manifest, err := LoadFrozenChainNodeRelationManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(manifest.Relations); got != 100 {
		t.Fatalf("relations = %d, want 100", got)
	}
}

func TestFrozenAdditiveChainNodeRelationManifestCombinesAcceptedBaselineExactly(t *testing.T) {
	accepted, err := LoadFrozenChainNodeRelationManifest(frozenChainNodeRelationManifestPath(t))
	if err != nil {
		t.Fatal(err)
	}
	combined, err := LoadFrozenAdditiveChainNodeRelationManifest(frozenAdditiveChainNodeRelationManifestPath(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(combined.Relations) != 212 {
		t.Fatalf("relations = %d, want 212", len(combined.Relations))
	}
	if !reflect.DeepEqual(combined.Relations[:100], accepted.Relations) {
		t.Fatal("accepted 100 relation baseline changed in additive manifest")
	}
	counts := map[domain.ChainNodeRelationType]int{}
	for _, relation := range combined.Relations {
		counts[relation.RelationType]++
	}
	if counts[domain.ChainNodeRelationSubcategoryOf] != 108 || counts[domain.ChainNodeRelationComponentOf] != 3 || counts[domain.ChainNodeRelationInputTo] != 93 || counts[domain.ChainNodeRelationDependsOn] != 8 {
		t.Fatalf("relation type counts = %+v, want 108/3/93/8", counts)
	}
}

func TestFrozenAdditiveChainNodeRelationManifestRejectsPathAndContentDrift(t *testing.T) {
	path := frozenAdditiveChainNodeRelationManifestPath(t)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateFrozenAdditiveChainNodeRelationFileIdentity(path, content); err != nil {
		t.Fatal(err)
	}
	drifted := append([]byte(nil), content...)
	drifted[len(drifted)-2] ^= 1
	if err := validateFrozenAdditiveChainNodeRelationFileIdentity(path, drifted); err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("content drift error = %v", err)
	}
	copyPath := filepath.Join(t.TempDir(), "additive-final-candidate-manifest.json")
	if err := os.WriteFile(copyPath, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateFrozenAdditiveChainNodeRelationFileIdentity(copyPath, content); err == nil || !strings.Contains(err.Error(), "path") {
		t.Fatalf("path drift error = %v", err)
	}
}

func TestFrozenChainNodeRelationManifestRejectsPathAndContentDrift(t *testing.T) {
	path := frozenChainNodeRelationManifestPath(t)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateFrozenChainNodeRelationFileIdentity(path, content); err != nil {
		t.Fatal(err)
	}
	drifted := append([]byte(nil), content...)
	drifted[len(drifted)-2] ^= 1
	if err := validateFrozenChainNodeRelationFileIdentity(path, drifted); err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("content drift error = %v", err)
	}
	copyPath := filepath.Join(t.TempDir(), "approved-candidate-manifest.json")
	if err := os.WriteFile(copyPath, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateFrozenChainNodeRelationFileIdentity(copyPath, content); err == nil || !strings.Contains(err.Error(), "path") {
		t.Fatalf("path drift error = %v", err)
	}
}

func TestFrozenChainNodeRelationManifestRejectsEndpointOutsideFrozen842(t *testing.T) {
	manifest, err := LoadFrozenChainNodeRelationManifest(frozenChainNodeRelationManifestPath(t))
	if err != nil {
		t.Fatal(err)
	}
	endpoints, err := loadFrozenChainNodeEndpointBaseline()
	if err != nil {
		t.Fatal(err)
	}
	delete(endpoints, manifest.Relations[0].FromChainNodeEntityID)
	if err := validateFrozenChainNodeRelationEndpoints(manifest.Relations, endpoints); err == nil {
		t.Fatal("endpoint baseline drift error = nil")
	}
}

func TestFrozenChainNodeRelationManifestRejectsCountAndTypeDrift(t *testing.T) {
	content, err := os.ReadFile(frozenChainNodeRelationManifestPath(t))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		mutate func(*frozenChainNodeRelationManifest, *ChainNodeRelationManifest)
	}{
		{name: "baseline count", mutate: func(frozen *frozenChainNodeRelationManifest, _ *ChainNodeRelationManifest) {
			frozen.BaselineNodeCount = 841
		}},
		{name: "relation count", mutate: func(frozen *frozenChainNodeRelationManifest, _ *ChainNodeRelationManifest) { frozen.RelationCount = 99 }},
		{name: "type count", mutate: func(frozen *frozenChainNodeRelationManifest, _ *ChainNodeRelationManifest) {
			frozen.ByRelationType[domain.ChainNodeRelationInputTo] = 2
		}},
		{name: "relation rows", mutate: func(_ *frozenChainNodeRelationManifest, manifest *ChainNodeRelationManifest) {
			manifest.Relations = manifest.Relations[:99]
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var frozen frozenChainNodeRelationManifest
			if err := decodeFrozenChainNodeJSON(content, &frozen); err != nil {
				t.Fatal(err)
			}
			manifest := ChainNodeRelationManifest{Relations: make([]domain.ChainNodeRelation, 0, len(frozen.Relations))}
			for _, relation := range frozen.Relations {
				manifest.Relations = append(manifest.Relations, relation.ChainNodeRelation)
			}
			test.mutate(&frozen, &manifest)
			if err := validateFrozenChainNodeRelationMetadata(frozen, manifest); err == nil {
				t.Fatal("metadata drift error = nil")
			}
		})
	}
}

func TestPreflightFrozenChainNodeRelationDataRequiresExactGoose18SchemaAndBaseline(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery(relationDataBaselineSQL).WillReturnRows(sqlmock.NewRows([]string{"database", "version", "goose", "nodes", "profiles", "external", "edges", "relations", "subcategory", "component", "input", "depends", "constraints"}).AddRow("tidewise_local", "16.14", 18, 842, 842, 1169, 241, 100, 95, 1, 3, 1, 0))
	mock.ExpectQuery(relationDataSchemaSQL).WillReturnRows(sqlmock.NewRows([]string{"relation_columns", "constraint_columns", "relation_checks", "relation_fks", "relation_pks", "relation_uniques", "constraint_checks", "constraint_fks", "constraint_pks", "relation_indexes", "constraint_indexes", "triggers"}).AddRow(relationColumnSignature, physicalConstraintColumnSignature, 7, 2, 1, 1, 7, 2, 1, 4, 3, 0))
	report, err := NewPostgresRepository(db).PreflightFrozenChainNodeRelationData(context.Background())
	if err != nil || !report.SchemaValid || report.GooseVersion != 18 || report.ExistingRelations != 100 {
		t.Fatalf("report=%+v err=%v", report, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPreflightFrozenChainNodeRelationDataRejectsSchemaDrift(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery(relationDataBaselineSQL).WillReturnRows(sqlmock.NewRows([]string{"database", "version", "goose", "nodes", "profiles", "external", "edges", "relations", "subcategory", "component", "input", "depends", "constraints"}).AddRow("tidewise_local", "16.14", 18, 842, 842, 1169, 241, 100, 95, 1, 3, 1, 0))
	mock.ExpectQuery(relationDataSchemaSQL).WillReturnRows(sqlmock.NewRows([]string{"relation_columns", "constraint_columns", "relation_checks", "relation_fks", "relation_pks", "relation_uniques", "constraint_checks", "constraint_fks", "constraint_pks", "relation_indexes", "constraint_indexes", "triggers"}).AddRow(relationColumnSignature, physicalConstraintColumnSignature, 6, 2, 1, 1, 7, 2, 1, 4, 3, 0))
	if _, err := NewPostgresRepository(db).PreflightFrozenChainNodeRelationData(context.Background()); err == nil {
		t.Fatal("schema drift error = nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyFrozenChainNodeRelationPostWriteChecksProtectedBaselineAndExactAggregate(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery(relationDataBaselineSQL).WillReturnRows(sqlmock.NewRows([]string{"database", "version", "goose", "nodes", "profiles", "external", "edges", "relations", "subcategory", "component", "input", "depends", "constraints"}).AddRow("tidewise_local", "16.14", 18, 842, 842, 1169, 241, 212, 108, 3, 93, 8, 0))
	mock.ExpectQuery(relationDataSchemaSQL).WillReturnRows(sqlmock.NewRows([]string{"relation_columns", "constraint_columns", "relation_checks", "relation_fks", "relation_pks", "relation_uniques", "constraint_checks", "constraint_fks", "constraint_pks", "relation_indexes", "constraint_indexes", "triggers"}).AddRow(relationColumnSignature, physicalConstraintColumnSignature, 7, 2, 1, 1, 7, 2, 1, 4, 3, 0))
	mock.ExpectQuery(frozenChainNodeRelationAggregateSQL).WillReturnRows(sqlmock.NewRows([]string{"total", "subcategory", "component", "input", "depends", "incomplete", "self", "duplicate", "orphan"}).AddRow(212, 108, 3, 93, 8, 0, 0, 0, 0))
	if err := verifyFrozenChainNodeRelationPostWrite(context.Background(), tx); err != nil {
		t.Fatal(err)
	}
	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func frozenChainNodeRelationManifestPath(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "..", "..", "..", "..", "src", "backend", "data", "entity_foundation", "relationships", "reviewed_chain_node_relations", "approved-candidate-manifest.json")
}

func frozenAdditiveChainNodeRelationManifestPath(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "..", "..", "..", "..", "src", "backend", "data", "entity_foundation", "relationships", "reviewed_chain_node_relations", "additive-final-candidate-manifest.json")
}
