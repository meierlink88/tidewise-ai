package seed

import (
	"context"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func TestMemoryRepositoryUpsertsEntityIdempotently(t *testing.T) {
	repo := NewMemoryRepository()
	entity := Entity{
		Key:           "economy:cn",
		EntityType:    domain.EntityTypeEconomy,
		LayerCode:     "economy",
		Name:          "中国",
		CanonicalName: "中国",
		Status:        domain.StatusActive,
	}

	first, err := repo.UpsertEntity(context.Background(), entity)
	if err != nil {
		t.Fatalf("UpsertEntity(first) error = %v", err)
	}
	second, err := repo.UpsertEntity(context.Background(), entity)
	if err != nil {
		t.Fatalf("UpsertEntity(second) error = %v", err)
	}

	if first.Action != WriteCreated {
		t.Fatalf("first action = %q, want %q", first.Action, WriteCreated)
	}
	if second.Action != WriteUnchanged {
		t.Fatalf("second action = %q, want %q", second.Action, WriteUnchanged)
	}
	if got, want := repo.EntityCount(), 1; got != want {
		t.Fatalf("entity count = %d, want %d", got, want)
	}
}

func TestMemoryRepositoryUpdatesChangedEntity(t *testing.T) {
	repo := NewMemoryRepository()
	entity := Entity{
		Key:           "economy:cn",
		EntityType:    domain.EntityTypeEconomy,
		LayerCode:     "economy",
		Name:          "中国",
		CanonicalName: "中国",
		Status:        domain.StatusActive,
	}
	if _, err := repo.UpsertEntity(context.Background(), entity); err != nil {
		t.Fatalf("UpsertEntity(first) error = %v", err)
	}
	entity.Aliases = []string{"中华人民共和国", "China"}

	result, err := repo.UpsertEntity(context.Background(), entity)
	if err != nil {
		t.Fatalf("UpsertEntity(update) error = %v", err)
	}

	if result.Action != WriteUpdated {
		t.Fatalf("action = %q, want %q", result.Action, WriteUpdated)
	}
}

func TestMemoryRepositoryUpsertsProfilesAndRelationshipsIdempotently(t *testing.T) {
	repo := NewMemoryRepository()
	entity := Entity{
		Key:           "economy:cn",
		EntityType:    domain.EntityTypeEconomy,
		LayerCode:     "economy",
		Name:          "中国",
		CanonicalName: "中国",
		Status:        domain.StatusActive,
	}
	alliance := Entity{
		Key:           "alliance_org:g20",
		EntityType:    domain.EntityTypeAllianceOrg,
		LayerCode:     "alliance",
		Name:          "二十国集团",
		CanonicalName: "二十国集团",
		Status:        domain.StatusActive,
	}
	if _, err := repo.UpsertEntity(context.Background(), entity); err != nil {
		t.Fatalf("UpsertEntity(entity) error = %v", err)
	}
	if _, err := repo.UpsertEntity(context.Background(), alliance); err != nil {
		t.Fatalf("UpsertEntity(alliance) error = %v", err)
	}

	profile := Profile{
		EntityKey:  "economy:cn",
		EntityType: domain.EntityTypeEconomy,
		Data:       []byte(`{"country_code":"CN","currency_code":"CNY"}`),
	}
	relationship := Relationship{
		Key:          "relationship:cn_member_of_g20",
		From:         "economy:cn",
		To:           "alliance_org:g20",
		RelationType: "member_of",
		SourceName:   "G20",
		SourceURL:    "https://g20.org/members/",
		VerifiedAt:   time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		Status:       domain.StatusActive,
	}

	cases := []struct {
		name string
		run  func() (WriteResult, error)
	}{
		{name: "create profile", run: func() (WriteResult, error) {
			return repo.UpsertProfile(context.Background(), profile)
		}},
		{name: "unchanged profile", run: func() (WriteResult, error) {
			return repo.UpsertProfile(context.Background(), profile)
		}},
		{name: "create relationship", run: func() (WriteResult, error) {
			return repo.UpsertRelationship(context.Background(), relationship)
		}},
		{name: "unchanged relationship", run: func() (WriteResult, error) {
			return repo.UpsertRelationship(context.Background(), relationship)
		}},
	}
	wantActions := []WriteAction{WriteCreated, WriteUnchanged, WriteCreated, WriteUnchanged}

	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.run()
			if err != nil {
				t.Fatalf("run() error = %v", err)
			}
			if result.Action != wantActions[index] {
				t.Fatalf("action = %q, want %q", result.Action, wantActions[index])
			}
		})
	}
}

func TestMemoryRepositoryUpsertsBenchmarkProfileIdempotently(t *testing.T) {
	repo := NewMemoryRepository()
	entity := Entity{
		Key:           "benchmark:us_10y_treasury_yield",
		EntityType:    domain.EntityTypeBenchmark,
		LayerCode:     "benchmark",
		Name:          "美国10年期国债收益率",
		CanonicalName: "美国10年期国债收益率",
		Aliases:       []string{"US 10Y Treasury Yield"},
		Status:        domain.StatusActive,
	}
	if _, err := repo.UpsertEntity(context.Background(), entity); err != nil {
		t.Fatalf("UpsertEntity() error = %v", err)
	}

	profile := Profile{
		EntityKey:  entity.Key,
		EntityType: domain.EntityTypeBenchmark,
		Data: []byte(`{
		  "benchmark_type":"government_bond_yield",
		  "official_series_code":null,
		  "provider":"us_treasury",
		  "tenor":"10Y",
		  "currency_code":"USD",
		  "unit":"percent",
		  "frequency":"daily",
		  "source_url":"https://home.treasury.gov/"
		}`),
	}

	first, err := repo.UpsertProfile(context.Background(), profile)
	if err != nil {
		t.Fatalf("UpsertProfile(first) error = %v", err)
	}
	second, err := repo.UpsertProfile(context.Background(), profile)
	if err != nil {
		t.Fatalf("UpsertProfile(second) error = %v", err)
	}
	if first.Action != WriteCreated || second.Action != WriteUnchanged {
		t.Fatalf("actions = %q/%q, want created/unchanged", first.Action, second.Action)
	}
}

func TestMemoryRepositoryUpdatesRelationshipProvenance(t *testing.T) {
	repo := NewMemoryRepository()
	for _, entity := range []Entity{
		{Key: "economy:cn", EntityType: domain.EntityTypeEconomy, LayerCode: "economy", Name: "中国", CanonicalName: "中国"},
		{Key: "alliance_org:g20", EntityType: domain.EntityTypeAllianceOrg, LayerCode: "alliance", Name: "二十国集团", CanonicalName: "二十国集团"},
	} {
		if _, err := repo.UpsertEntity(context.Background(), entity); err != nil {
			t.Fatalf("UpsertEntity() error = %v", err)
		}
	}
	relationship := Relationship{
		Key: "relationship:cn_member_of_g20", From: "economy:cn", To: "alliance_org:g20", RelationType: "member_of",
		SourceName: "G20", SourceURL: "https://g20.org/members/", VerifiedAt: time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
	}
	if result, err := repo.UpsertRelationship(context.Background(), relationship); err != nil || result.Action != WriteCreated {
		t.Fatalf("UpsertRelationship(create) = %+v, %v", result, err)
	}
	relationship.SourceURL = "https://www.g20.org/en/about-the-g20/members/"
	if result, err := repo.UpsertRelationship(context.Background(), relationship); err != nil || result.Action != WriteUpdated {
		t.Fatalf("UpsertRelationship(update) = %+v, %v", result, err)
	}
}

func TestMemoryRepositoryRejectsDanglingWrites(t *testing.T) {
	repo := NewMemoryRepository()

	if _, err := repo.UpsertProfile(context.Background(), Profile{
		EntityKey:  "economy:missing",
		EntityType: domain.EntityTypeEconomy,
		Data:       []byte(`{"country_code":"CN","currency_code":"CNY"}`),
	}); err == nil {
		t.Fatal("UpsertProfile() expected dangling entity error")
	}

	if _, err := repo.UpsertRelationship(context.Background(), Relationship{
		Key:          "relationship:missing",
		From:         "economy:cn",
		To:           "alliance_org:g20",
		RelationType: "member_of",
	}); err == nil {
		t.Fatal("UpsertRelationship() expected dangling relationship error")
	}
}

func TestMemoryRepositoryUpdatesLatestSectorSourceMappingSnapshotIdempotently(t *testing.T) {
	repo := NewMemoryRepository()
	entity := Entity{Key: "sector:theme_ai", EntityType: domain.EntityTypeSector, LayerCode: "sector", Name: "人工智能", CanonicalName: "人工智能", Aliases: []string{"Artificial Intelligence"}}
	if _, err := repo.UpsertEntity(context.Background(), entity); err != nil {
		t.Fatalf("UpsertEntity() error = %v", err)
	}
	mapping := SectorSourceMapping{
		SectorEntityKey: entity.Key, SourceSystem: "ths", SourceTaxonomyType: "concept",
		SourceSectorName: "人工 智能", SourceMarketScope: "cn_a_share",
		RankSnapshot: 1, SnapshotDate: "2026-07-01", MappingStatus: "approved",
	}
	first, err := repo.UpsertSectorSourceMapping(context.Background(), mapping)
	if err != nil {
		t.Fatalf("UpsertSectorSourceMapping(first) error = %v", err)
	}
	mapping.SourceSectorName = "人工智能"
	mapping.RankSnapshot = 4
	mapping.SnapshotDate = "2026-07-08"
	mapping.SourceURL = "https://example.com/latest"
	second, err := repo.UpsertSectorSourceMapping(context.Background(), mapping)
	if err != nil {
		t.Fatalf("UpsertSectorSourceMapping(second) error = %v", err)
	}
	third, err := repo.UpsertSectorSourceMapping(context.Background(), mapping)
	if err != nil {
		t.Fatalf("UpsertSectorSourceMapping(third) error = %v", err)
	}
	if first.Action != WriteCreated || second.Action != WriteUpdated || third.Action != WriteUnchanged {
		t.Fatalf("actions = %q/%q/%q, want created/updated/unchanged", first.Action, second.Action, third.Action)
	}
	older := mapping
	older.RankSnapshot = 1
	older.SnapshotDate = "2026-07-01"
	older.SourceURL = "https://example.com/older"
	olderResult, err := repo.UpsertSectorSourceMapping(context.Background(), older)
	if err != nil {
		t.Fatalf("UpsertSectorSourceMapping(older) error = %v", err)
	}
	if olderResult.Action != WriteUnchanged {
		t.Fatalf("older snapshot action = %q, want unchanged", olderResult.Action)
	}
	if got := repo.SectorSourceMappingCount(); got != 1 {
		t.Fatalf("source mapping count = %d, want 1", got)
	}
}
