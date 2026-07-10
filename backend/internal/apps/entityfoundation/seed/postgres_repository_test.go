package seed

import (
	"strings"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func TestEntitySeedUUIDsAreStable(t *testing.T) {
	first := entitySeedUUID("economy:cn")
	second := entitySeedUUID("economy:cn")
	if first != second {
		t.Fatalf("entitySeedUUID() = %q and %q, want stable id", first, second)
	}
	if first == entitySeedUUID("economy:us") {
		t.Fatalf("entitySeedUUID() should differ for different keys")
	}
}

func TestPostgresEntityWriteNormalizesNilAliases(t *testing.T) {
	entity := Entity{
		Key:           "sector:test",
		EntityType:    domain.EntityTypeSector,
		LayerCode:     "sector",
		Name:          "测试板块",
		CanonicalName: "测试板块",
	}
	if entity.Aliases != nil {
		t.Fatal("test setup expected nil aliases")
	}

	entity.Aliases = normalizeEntityAliases(entity.Aliases)
	if entity.Aliases == nil {
		t.Fatal("aliases should be normalized to an empty slice")
	}
}

func TestEntityUpsertSQLPersistsBusinessKey(t *testing.T) {
	statement := buildEntityUpsert()

	for _, fragment := range []string{
		"entity_key",
		"entity_key = excluded.entity_key",
		"entity_nodes.entity_key is distinct from excluded.entity_key",
	} {
		if !strings.Contains(strings.ToLower(statement), fragment) {
			t.Fatalf("entity upsert statement missing %q: %s", fragment, statement)
		}
	}
}

func TestProfileTableName(t *testing.T) {
	cases := map[domain.EntityType]string{
		domain.EntityTypeAllianceOrg: "alliance_org_profiles",
		domain.EntityTypeEconomy:     "economy_profiles",
		domain.EntityTypeSector:      "sector_profiles",
		domain.EntityTypeSecurity:    "security_profiles",
		domain.EntityTypePerson:      "person_profiles",
	}

	for entityType, want := range cases {
		t.Run(string(entityType), func(t *testing.T) {
			got, err := profileTableName(entityType)
			if err != nil {
				t.Fatalf("profileTableName() error = %v", err)
			}
			if got != want {
				t.Fatalf("profileTableName() = %q, want %q", got, want)
			}
		})
	}
}

func TestProfileUpsertSQLIncludesSectorSnapshotFields(t *testing.T) {
	statement, _, err := buildProfileUpsert("sector:ai", domain.EntityTypeSector, []byte(`{
		"sector_system": "ths",
		"sector_code": "concept_001",
		"sector_type": "concept",
		"exchange_scope": "CN",
		"rank_snapshot": 1,
		"snapshot_date": "2026-07-08"
	}`))
	if err != nil {
		t.Fatalf("buildProfileUpsert() error = %v", err)
	}

	for _, fragment := range []string{"sector_profiles", "rank_snapshot", "snapshot_date"} {
		if !strings.Contains(statement, fragment) {
			t.Fatalf("statement missing %q: %s", fragment, statement)
		}
	}
}
