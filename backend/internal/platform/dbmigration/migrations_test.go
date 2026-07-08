package dbmigration

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestMigrationFilesAreVersionedAndGooseCompatible(t *testing.T) {
	files := migrationFiles(t)
	versionPattern := regexp.MustCompile(`^\d{6}_[a-z0-9_]+\.sql$`)

	seenVersions := map[string]string{}
	for _, file := range files {
		name := filepath.Base(file)
		if !versionPattern.MatchString(name) {
			t.Fatalf("migration file %q must use 000001_name.sql format", name)
		}

		version := name[:6]
		if previous, ok := seenVersions[version]; ok {
			t.Fatalf("migration version %s is duplicated by %q and %q", version, previous, name)
		}
		seenVersions[version] = name

		content := readMigration(t, file)
		if !strings.Contains(content, "-- +goose Up") {
			t.Fatalf("migration %q must include -- +goose Up", name)
		}
		if !strings.Contains(content, "-- +goose Down") {
			t.Fatalf("migration %q must include -- +goose Down", name)
		}
	}
}

func TestInitialEventKnowledgeMigrationDefinesCoreTables(t *testing.T) {
	content := combinedMigrations(t)

	for _, table := range []string{
		"entity_nodes",
		"entity_edges",
		"alliance_org_profiles",
		"economy_profiles",
		"policy_body_profiles",
		"market_profiles",
		"index_profiles",
		"sector_profiles",
		"chain_node_profiles",
		"company_profiles",
		"security_profiles",
		"instrument_profiles",
		"metric_profiles",
		"commodity_profiles",
		"person_profiles",
		"source_catalogs",
		"raw_documents",
		"events",
		"event_sources",
		"event_tag_defs",
		"event_tag_maps",
		"event_entity_links",
	} {
		if !strings.Contains(content, "create table "+table) {
			t.Fatalf("initial migration must create table %s", table)
		}
	}
}

func TestInitialEventKnowledgeMigrationDefinesCriticalConstraints(t *testing.T) {
	content := combinedMigrations(t)

	for _, fragment := range []string{
		"primary key",
		"references entity_nodes",
		"unique (org_code)",
		"rank_snapshot integer not null default 0",
		"snapshot_date date",
		"references source_catalogs",
		"references raw_documents",
		"references events",
		"on raw_documents (source_id, source_external_id)",
		"on raw_documents (source_id, content_hash)",
		"unique (tag_kind, code)",
		"unique (event_id, tag_id)",
		"unique (event_id, entity_id, entity_role)",
	} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("initial migration must contain constraint fragment %q", fragment)
		}
	}
}

func TestInitialEventKnowledgeMigrationDefinesQueryIndexes(t *testing.T) {
	content := combinedMigrations(t)

	for _, indexName := range []string{
		"idx_entity_edges_from_entity_id",
		"idx_entity_edges_to_entity_id",
		"idx_alliance_org_profiles_org_type",
		"idx_alliance_org_profiles_primary_domain",
		"idx_source_catalogs_status",
		"idx_source_catalogs_provider_channel",
		"idx_raw_documents_source_id",
		"idx_raw_documents_published_at",
		"idx_raw_documents_ingest_status",
		"idx_events_dedupe_key",
		"idx_events_event_time",
		"idx_event_sources_event_id",
		"idx_event_sources_raw_document_id",
		"idx_event_tag_maps_event_id",
		"idx_event_entity_links_event_id",
	} {
		if !strings.Contains(content, indexName) {
			t.Fatalf("initial migration must define index %s", indexName)
		}
	}
}

func TestMigrationsDoNotUseDataDestructiveResetStatements(t *testing.T) {
	content := combinedMigrations(t)

	for _, forbidden := range []string{
		"drop database",
		"drop schema",
		"truncate table",
		"delete from",
	} {
		if strings.Contains(content, forbidden) {
			t.Fatalf("migration must not contain destructive reset statement %q", forbidden)
		}
	}
}

func TestSourceCatalogSourceConfigMigrationIsAdditive(t *testing.T) {
	content := strings.ToLower(readMigration(t, filepath.Join("..", "..", "..", "migrations", "000004_add_source_catalog_source_config.sql")))

	for _, fragment := range []string{
		"alter table source_catalogs",
		"add column if not exists source_config jsonb not null default '{}'::jsonb",
		"source catalog source_config rollback requires a reviewed forward migration or restored backup",
	} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("source_config migration must contain fragment %q", fragment)
		}
	}

	for _, forbidden := range []string{
		"create table source_catalogs",
		"drop table",
		"drop column",
		"truncate table",
		"delete from",
	} {
		if strings.Contains(content, forbidden) {
			t.Fatalf("source_config migration must not contain destructive fragment %q", forbidden)
		}
	}
}

func migrationFiles(t *testing.T) []string {
	t.Helper()

	matches, err := filepath.Glob(filepath.Join("..", "..", "..", "migrations", "*.sql"))
	if err != nil {
		t.Fatalf("find migration files: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("expected at least one migration file")
	}

	return matches
}

func combinedMigrations(t *testing.T) string {
	t.Helper()

	var builder strings.Builder
	for _, file := range migrationFiles(t) {
		builder.WriteString(readMigration(t, file))
		builder.WriteString("\n")
	}

	return strings.ToLower(builder.String())
}

func readMigration(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration %s: %v", path, err)
	}

	return string(content)
}
