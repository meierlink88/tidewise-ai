package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestEntityExternalIdentifiersSchema(t *testing.T) {
	data, err := os.ReadFile("000016_add_entity_external_identifiers.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(data))
	for _, required := range []string{
		"current_setting('tidewise.external_identifier_schema_write_authorized', true)",
		"external identifier schema write is not authorized",
		"create table entity_external_identifiers",
		"id uuid primary key",
		"entity_id uuid not null references entity_nodes(id) on delete cascade",
		"source_system text not null",
		"source_taxonomy_type text not null",
		"external_code text not null",
		"external_name text not null",
		"status varchar(32) not null default 'active'",
		"created_at timestamptz not null default now()",
		"updated_at timestamptz not null default now()",
		"unique (source_system, source_taxonomy_type, external_code)",
		"create index idx_entity_external_identifiers_entity_source",
		"on entity_external_identifiers (entity_id, source_system, source_taxonomy_type)",
		"check (btrim(source_system) <> '')",
		"check (btrim(source_taxonomy_type) <> '')",
		"check (btrim(external_code) <> '')",
		"check (btrim(external_name) <> '')",
		"check (status in ('active', 'inactive'))",
		"migration 000016 is irreversible after external identifiers exist",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("external identifier migration missing %q", required)
		}
	}
	for _, forbidden := range []string{
		"chain_node_source_mappings",
		"sector_source_mappings",
		"jsonb",
		"unique (entity_id, source_system, source_taxonomy_type, external_code)",
		"insert into entity_external_identifiers",
		"create trigger",
		"create function",
		"create view",
	} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("external identifier migration contains forbidden %q", forbidden)
		}
	}
}
