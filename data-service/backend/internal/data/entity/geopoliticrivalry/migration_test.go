package geopoliticrivalry

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
	"github.com/pressly/goose/v3"
)

func TestCandidateAssetMigrationFailsClosedWhenStorylinesExist(t *testing.T) {
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	db := postgresfixture.OpenIsolated(t, "tw_geopolitical_candidate_assets_cutover", migrationDir, 82)
	domainID := createDomain(t, db, "MILITARY", "军事/防务线")
	storylineID, err := coreid.New(coreid.GeopoliticRivalry)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO geopolitic_rivalries (
    id, name, category, geopolitic_domain_id, core_proposition, core_actors, main_transmission
) VALUES ($1, '俄乌战争', '俄乌及欧洲安全', $2, '核心命题', '核心参与方', '主要传导')`, storylineID, domainID); err != nil {
		t.Fatal(err)
	}
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	err = goose.UpToContext(context.Background(), db, migrationDir, 83)
	if err == nil || !strings.Contains(err.Error(), "SQLSTATE 55000") {
		t.Fatalf("migration 83 error = %v, want SQLSTATE 55000", err)
	}
	version, err := goose.GetDBVersion(db)
	if err != nil {
		t.Fatal(err)
	}
	if version != 82 {
		t.Fatalf("migration version = %d, want 82", version)
	}
	var candidateAssetsColumnExists bool
	if err := db.QueryRow(`SELECT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = current_schema()
      AND table_name = 'geopolitic_rivalries'
      AND column_name = 'candidate_assets'
)`).Scan(&candidateAssetsColumnExists); err != nil {
		t.Fatal(err)
	}
	if candidateAssetsColumnExists {
		t.Fatal("failed migration left candidate_assets behind")
	}
	var storylineCount int
	if err := db.QueryRow(`SELECT count(*) FROM geopolitic_rivalries`).Scan(&storylineCount); err != nil {
		t.Fatal(err)
	}
	if storylineCount != 1 {
		t.Fatalf("failed migration preserved %d storylines, want 1", storylineCount)
	}
}
