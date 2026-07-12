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

func TestMemoryRepositoryUpdatesReviewFieldsWithoutReplacingNewerSnapshot(t *testing.T) {
	repo := NewMemoryRepository()
	for _, entity := range []Entity{
		{Key: "sector:theme_ai", EntityType: domain.EntityTypeSector, LayerCode: "sector", Name: "人工智能", CanonicalName: "人工智能", Aliases: []string{"Artificial Intelligence"}},
		{Key: "sector:theme_ai_merged", EntityType: domain.EntityTypeSector, LayerCode: "sector", Name: "人工智能产业", CanonicalName: "人工智能产业", Aliases: []string{"Artificial Intelligence Industry"}},
	} {
		if _, err := repo.UpsertEntity(context.Background(), entity); err != nil {
			t.Fatalf("UpsertEntity() error = %v", err)
		}
	}
	current := SectorSourceMapping{
		SectorEntityKey: "sector:theme_ai", SourceSystem: "ths", SourceTaxonomyType: "concept",
		SourceSectorCode: "885001", SourceSectorName: "人工智能", SourceMarketScope: "cn_a_share",
		SourceURL: "https://example.com/newer", RankSnapshot: 2, SnapshotDate: "2026-07-12",
		MappingStatus: "candidate",
	}
	if _, err := repo.UpsertSectorSourceMapping(context.Background(), current); err != nil {
		t.Fatalf("UpsertSectorSourceMapping(current) error = %v", err)
	}
	olderReview := current
	olderReview.SectorEntityKey = "sector:theme_ai_merged"
	olderReview.SourceURL = "https://example.com/older"
	olderReview.RankSnapshot = 9
	olderReview.SnapshotDate = "2026-07-01"
	olderReview.MappingStatus = "merged"
	olderReview.ReviewNote = "reviewed canonical merge"
	result, err := repo.UpsertSectorSourceMapping(context.Background(), olderReview)
	if err != nil {
		t.Fatalf("UpsertSectorSourceMapping(older review) error = %v", err)
	}
	if result.Action != WriteUpdated {
		t.Fatalf("older review action = %q, want updated", result.Action)
	}
	stored := repo.sectorSourceMappings[sectorSourceMappingIdentity(normalizeSectorSourceMapping(olderReview))]
	if stored.SectorEntityKey != olderReview.SectorEntityKey || stored.MappingStatus != "merged" || stored.ReviewNote != olderReview.ReviewNote {
		t.Fatalf("review fields not updated: %+v", stored)
	}
	if stored.RankSnapshot != current.RankSnapshot || stored.SnapshotDate != current.SnapshotDate || stored.SourceURL != current.SourceURL {
		t.Fatalf("newer snapshot fields replaced: %+v", stored)
	}
	unchanged, err := repo.UpsertSectorSourceMapping(context.Background(), SectorSourceMapping{
		SectorEntityKey: olderReview.SectorEntityKey, SourceSystem: olderReview.SourceSystem,
		SourceTaxonomyType: olderReview.SourceTaxonomyType, SourceSectorCode: olderReview.SourceSectorCode,
		SourceSectorName: olderReview.SourceSectorName, SourceMarketScope: olderReview.SourceMarketScope,
		MappingStatus: olderReview.MappingStatus, ReviewNote: olderReview.ReviewNote,
	})
	if err != nil {
		t.Fatalf("UpsertSectorSourceMapping(no snapshot unchanged) error = %v", err)
	}
	if unchanged.Action != WriteUnchanged {
		t.Fatalf("no snapshot unchanged action = %q, want unchanged", unchanged.Action)
	}
}

func TestMemoryRepositoryAllowsApprovalWithoutReplacingExistingSnapshot(t *testing.T) {
	repo := NewMemoryRepository()
	entity := Entity{Key: "sector:theme_ai", EntityType: domain.EntityTypeSector, LayerCode: "sector", Name: "人工智能", CanonicalName: "人工智能", Aliases: []string{"Artificial Intelligence"}}
	if _, err := repo.UpsertEntity(context.Background(), entity); err != nil {
		t.Fatalf("UpsertEntity() error = %v", err)
	}
	current := SectorSourceMapping{
		SectorEntityKey: entity.Key, SourceSystem: "ths", SourceTaxonomyType: "concept",
		SourceSectorCode: "885001", SourceSectorName: "人工智能", SourceURL: "https://example.com/latest",
		RankSnapshot: 2, SnapshotDate: "2026-07-12", MappingStatus: "candidate",
	}
	if _, err := repo.UpsertSectorSourceMapping(context.Background(), current); err != nil {
		t.Fatalf("UpsertSectorSourceMapping(current) error = %v", err)
	}
	approval := current
	approval.SourceURL = ""
	approval.RankSnapshot = 0
	approval.SnapshotDate = ""
	approval.MappingStatus = "approved"
	approval.ReviewNote = "approved by user"
	result, err := repo.UpsertSectorSourceMapping(context.Background(), approval)
	if err != nil {
		t.Fatalf("UpsertSectorSourceMapping(approval) error = %v", err)
	}
	if result.Action != WriteUpdated {
		t.Fatalf("approval action = %q, want updated", result.Action)
	}
	stored := repo.sectorSourceMappings[sectorSourceMappingIdentity(normalizeSectorSourceMapping(approval))]
	if stored.MappingStatus != "approved" || stored.SnapshotDate != current.SnapshotDate || stored.SourceURL != current.SourceURL {
		t.Fatalf("approval/snapshot state = %+v", stored)
	}
}
