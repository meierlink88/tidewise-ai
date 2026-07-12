package migrations_test

import (
	"strings"
	"testing"
)

func TestSectorConvergenceMigrationDefinesAppendOnlyVersionedAudit(t *testing.T) {
	sql := strings.ToLower(readMigration(t, "000011_add_sector_convergence.sql"))
	for _, fragment := range []string{
		"create table entity_convergence_manifests",
		"manifest_version bigint primary key",
		"previous_manifest_version bigint",
		"manifest_checksum text not null unique",
		"create table entity_convergences",
		"id uuid primary key",
		"legacy_entity_id uuid not null references entity_nodes(id)",
		"target_entity_id uuid references entity_nodes(id)",
		"unique (legacy_entity_id, manifest_version)",
		"create table entity_convergence_reference_moves",
		"create table entity_convergence_alias_moves",
		"raise exception 'convergence audit is append-only'",
		"before update or delete on entity_convergence_manifests",
		"before update or delete on entity_convergences",
		"before update or delete on entity_convergence_reference_moves",
		"before update or delete on entity_convergence_alias_moves",
		"-- +goose down",
		"reviewed forward migration or restored backup",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("convergence migration missing %q", fragment)
		}
	}
}
