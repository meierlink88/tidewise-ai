package seed

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func TestHasRetiredIndustryEntitiesChecksAllSectorAndIndustryChainRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery("entity_type IN \\('sector', 'industry_chain'\\)").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	found, err := NewPostgresRepository(db).HasRetiredIndustryEntities(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("retired industry entities were not detected")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

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

func TestRelationshipUpsertSQLPersistsProvenance(t *testing.T) {
	verifiedAt := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	statement, args := buildRelationshipUpsert(Relationship{
		Key:          "relationship:cn_member_of_g20",
		From:         "economy:cn",
		To:           "alliance_org:g20",
		RelationType: "member_of",
		EvidenceNote: "成员列表",
		SourceName:   "G20",
		SourceURL:    "https://g20.org/members/",
		VerifiedAt:   verifiedAt,
		Status:       domain.StatusActive,
	})

	for _, fragment := range []string{
		"source_name", "source_url", "verified_at",
		"source_name = excluded.source_name",
		"source_url = excluded.source_url",
		"verified_at = excluded.verified_at",
	} {
		if !strings.Contains(strings.ToLower(statement), fragment) {
			t.Fatalf("relationship upsert statement missing %q: %s", fragment, statement)
		}
	}
	if got, want := len(args), 9; got != want {
		t.Fatalf("relationship upsert args = %d, want %d", got, want)
	}
	if args[6] != "G20" || args[7] != "https://g20.org/members/" || args[8] != verifiedAt {
		t.Fatalf("relationship provenance args = %#v", args[6:])
	}
}

func TestRelationshipUpsertSQLRejectsIdentityChanges(t *testing.T) {
	statement, _ := buildRelationshipUpsert(Relationship{Key: "relationship:test", From: "chain_node:test", To: "sector:test", RelationType: "mapped_to_sector"})
	normalized := strings.ToLower(statement)
	for _, fragment := range []string{"identity_conflict", "from_entity_id is distinct from", "to_entity_id is distinct from", "relation_type is distinct from"} {
		if !strings.Contains(normalized, fragment) {
			t.Fatalf("relationship upsert missing immutable identity guard %q", fragment)
		}
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

func TestEntityUpsertSQLScopesAliasOwnershipToCurrentConvergence(t *testing.T) {
	statement := strings.ToLower(buildEntityUpsert())
	for _, fragment := range []string{"entity_convergence_alias_moves", "entity_convergences", "max(manifest_version)", "am.to_entity_id = $1", "coalesce($7::text[]", "order by am.moved_at,am.id", "sa.aliases || oa.aliases"} {
		if !strings.Contains(statement, fragment) {
			t.Fatalf("entity upsert missing %q", fragment)
		}
	}
}

func TestProfileTableName(t *testing.T) {
	cases := map[domain.EntityType]string{
		domain.EntityTypeAllianceOrg:   "alliance_org_profiles",
		domain.EntityTypeEconomy:       "economy_profiles",
		domain.EntityTypeSector:        "sector_profiles",
		domain.EntityTypeIndustryChain: "industry_chain_profiles",
		domain.EntityTypeSecurity:      "security_profiles",
		domain.EntityTypePerson:        "person_profiles",
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

func TestIndustryChainAndChainNodeProfileUpsertFields(t *testing.T) {
	chainSQL, _, err := buildProfileUpsert("industry_chain:test", domain.EntityTypeIndustryChain, []byte(`{"chain_code":"test","definition":"test","boundary_note":"","scope_type":"global","version":1,"review_status":"approved","source_name":"review","source_url":"https://example.com","verified_at":"2026-07-12T00:00:00Z"}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"industry_chain_profiles", "chain_code", "scope_type", "review_status", "verified_at"} {
		if !strings.Contains(chainSQL, fragment) {
			t.Fatalf("industry chain profile SQL missing %q", fragment)
		}
	}
	nodeSQL, _, err := buildProfileUpsert("chain_node:test", domain.EntityTypeChainNode, []byte(`{"chain_position":"midstream","node_category":"component","definition":"test","unit_of_analysis":"component","granularity_note":"test"}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"chain_node_profiles", "definition", "boundary_note"} {
		if !strings.Contains(nodeSQL, fragment) {
			t.Fatalf("chain node profile SQL missing %q", fragment)
		}
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

func TestProfileUpsertSQLIncludesMarketSectorFields(t *testing.T) {
	statement, _, err := buildProfileUpsert("sector:theme_ai", domain.EntityTypeSector, []byte(`{
		"sector_system":"tidewise","sector_code":"","sector_type":"theme",
		"classification_code":"theme_sector","primary_market_entity_id":"market:cn_a_share",
		"primary_economy_entity_id":"economy:cn","methodology_url":"https://example.com/methodology",
		"review_status":"approved"
	}`))
	if err != nil {
		t.Fatalf("buildProfileUpsert() error = %v", err)
	}
	for _, fragment := range []string{"classification_code", "primary_market_entity_id", "primary_economy_entity_id", "methodology_url", "review_status"} {
		if !strings.Contains(statement, fragment) {
			t.Fatalf("sector profile upsert missing %q", fragment)
		}
	}
}

func TestSectorSourceMappingUpsertUsesStableIdentityAndLatestSnapshot(t *testing.T) {
	mapping := SectorSourceMapping{
		SectorEntityKey: "sector:theme_ai", SourceSystem: "ths", SourceTaxonomyType: "concept",
		SourceSectorCode: "885001", SourceSectorName: "人工智能", SourceURL: "https://example.com/latest",
		RankSnapshot: 2, SnapshotDate: "2026-07-12", MappingStatus: "approved",
	}
	statement, args, err := buildSectorSourceMappingUpsert(mapping)
	if err != nil {
		t.Fatalf("buildSectorSourceMappingUpsert() error = %v", err)
	}
	for _, fragment := range []string{
		"insert into sector_source_mappings", "on conflict (id) do update set",
		"sector_entity_id = excluded.sector_entity_id",
		"mapping_status = excluded.mapping_status", "review_note = excluded.review_note",
		"rank_snapshot = case", "snapshot_date = case", "source_url = case",
		"excluded.snapshot_date >= sector_source_mappings.snapshot_date",
		"sector_source_mappings.mapping_status is distinct from excluded.mapping_status",
		"sector_source_mappings.review_note is distinct from excluded.review_note",
		"updated_at = now()",
	} {
		if !strings.Contains(strings.ToLower(statement), fragment) {
			t.Fatalf("source mapping upsert missing %q", fragment)
		}
	}
	if strings.Contains(strings.ToLower(statement), ")\n      and (sector_source_mappings.snapshot_date") {
		t.Fatal("snapshot recency must not gate non-snapshot review field updates")
	}
	if got, want := len(args), 13; got != want {
		t.Fatalf("source mapping args = %d, want %d", got, want)
	}
	firstID := args[0]
	mapping.RankSnapshot = 9
	mapping.SnapshotDate = "2026-07-19"
	_, laterArgs, err := buildSectorSourceMappingUpsert(mapping)
	if err != nil {
		t.Fatalf("buildSectorSourceMappingUpsert(later) error = %v", err)
	}
	if laterArgs[0] != firstID {
		t.Fatalf("mapping id changed across snapshots: %v != %v", laterArgs[0], firstID)
	}
}
