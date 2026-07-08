package entityseed

import (
	"context"
	"testing"

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
