package storylinedomain

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

func TestStoreCreatesGetsListsAndUpdatesStorylineDomains(t *testing.T) {
	db := openStorylineDomainTestDatabase(t)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	inputs := []CreateInput{
		{Name: "Beta domain", NameEn: "Shared domain", Description: "地缘政治叙事领域。", ScopeDefinition: "覆盖国家间政治与安全竞争。", DomainCategory: DomainCategoryGeopolitical},
		{Name: "公司治理", NameEn: "Corporate governance", Description: "公司治理叙事领域。", ScopeDefinition: "覆盖公司控制、经营与治理议题。", DomainCategory: DomainCategoryCorporate, IsActive: boolPointer(false)},
		{Name: "Alpha domain", NameEn: "Shared domain", Description: "宏观叙事领域。", ScopeDefinition: "覆盖货币、财政与经济数据议题。", DomainCategory: DomainCategoryMacro, IsActive: boolPointer(false)},
		{Name: "Alpha domain", NameEn: "Shared domain", Description: "第二份宏观叙事领域。", ScopeDefinition: "覆盖另一组宏观经济边界。", DomainCategory: DomainCategoryMacro, IsActive: boolPointer(false)},
	}
	created := make([]StorylineDomain, 0, len(inputs))
	for _, input := range inputs {
		item, err := store.Create(ctx, input)
		if err != nil {
			t.Fatal(err)
		}
		created = append(created, item)
	}
	if !coreid.Is(created[0].ID, coreid.StorylineDomain) || len(created[0].ID) != 39 {
		t.Fatalf("Create() ID = %q, want canonical SLD identity", created[0].ID)
	}
	if !created[0].IsActive || created[0].CreatedAt.IsZero() || !created[0].CreatedAt.Equal(created[0].UpdatedAt) || time.Since(created[0].CreatedAt) > time.Minute {
		t.Fatalf("Create() = %+v", created[0])
	}
	got, err := store.Get(ctx, created[0].ID)
	if err != nil || !reflect.DeepEqual(got, created[0]) {
		t.Fatalf("Get() = %+v, %v; want %+v", got, err, created[0])
	}

	all, err := store.List(ctx, Filter{})
	tiedIDs := []string{created[2].ID, created[3].ID}
	sort.Strings(tiedIDs)
	wantOrder := []string{created[1].ID, tiedIDs[0], tiedIDs[1], created[0].ID}
	if err != nil || len(all) != len(wantOrder) {
		t.Fatalf("List() = %+v, %v", all, err)
	}
	for index, wantID := range wantOrder {
		if all[index].ID != wantID {
			t.Fatalf("List()[%d].ID = %q, want %q; full list = %+v", index, all[index].ID, wantID, all)
		}
	}
	byCategory, err := store.List(ctx, Filter{DomainCategory: domainCategoryPointer(DomainCategoryMacro)})
	if err != nil || len(byCategory) != 2 {
		t.Fatalf("List(category) = %+v, %v", byCategory, err)
	}
	inactive := false
	byActive, err := store.List(ctx, Filter{IsActive: &inactive})
	if err != nil || len(byActive) != 3 {
		t.Fatalf("List(is_active) = %+v, %v", byActive, err)
	}
	combined, err := store.List(ctx, Filter{DomainCategory: domainCategoryPointer(DomainCategoryMacro), IsActive: &inactive})
	if err != nil || len(combined) != 2 || combined[0].ID != tiedIDs[0] || combined[1].ID != tiedIDs[1] {
		t.Fatalf("List(category AND is_active) = %+v, %v", combined, err)
	}

	time.Sleep(time.Millisecond)
	updated, err := store.Update(ctx, UpdateInput{
		ID: created[0].ID, Name: "全球地缘政治", NameEn: "Global geopolitics",
		Description: "更新后的地缘政治领域。", ScopeDefinition: "覆盖全球国家间竞争。",
		DomainCategory: DomainCategoryGeopolitical, IsActive: false,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.ID != created[0].ID || !updated.CreatedAt.Equal(created[0].CreatedAt) || !updated.UpdatedAt.After(created[0].UpdatedAt) || updated.IsActive {
		t.Fatalf("Update() = %+v; created = %+v", updated, created[0])
	}
}

func TestStoreEnforcesStorylineDomainContracts(t *testing.T) {
	db := openStorylineDomainTestDatabase(t)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	valid := CreateInput{
		Name: "重复名称", NameEn: "Duplicate name", Description: "合法描述。",
		ScopeDefinition: "合法边界。", DomainCategory: DomainCategoryIndustry,
	}
	first, err := store.Create(ctx, valid)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create(ctx, valid); err != nil {
		t.Fatalf("Create(duplicate names) error = %v", err)
	}

	invalid := []CreateInput{
		{Name: " ", NameEn: "Blank", Description: "描述", ScopeDefinition: "边界", DomainCategory: DomainCategoryIndustry},
		{Name: "超长", NameEn: strings.Repeat("x", 51), Description: "描述", ScopeDefinition: "边界", DomainCategory: DomainCategoryIndustry},
		{Name: "未知分类", NameEn: "Unknown", Description: "描述", ScopeDefinition: "边界", DomainCategory: DomainCategory("TECHNOLOGY")},
		{Name: "无描述", NameEn: "Missing description", Description: " ", ScopeDefinition: "边界", DomainCategory: DomainCategoryIndustry},
		{Name: "无边界", NameEn: "Missing scope", Description: "描述", ScopeDefinition: " ", DomainCategory: DomainCategoryIndustry},
	}
	for _, input := range invalid {
		if _, err := store.Create(ctx, input); !errors.Is(err, ErrInvalidStorylineDomain) {
			t.Errorf("Create(%+v) error = %v, want ErrInvalidStorylineDomain", input, err)
		}
	}
	if _, err := store.Get(ctx, "SDT11111111-1111-4111-8111-111111111111"); !errors.Is(err, ErrInvalidStorylineDomain) {
		t.Fatalf("Get(wrong identity) error = %v", err)
	}
	if _, err := store.Get(ctx, "SLD11111111-1111-4111-8111-111111111111"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(missing) error = %v", err)
	}
	if _, err := store.Update(ctx, UpdateInput{
		ID: "SLD11111111-1111-4111-8111-111111111111", Name: "缺失", NameEn: "Missing",
		Description: "描述", ScopeDefinition: "边界", DomainCategory: DomainCategoryIndustry, IsActive: true,
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update(missing) error = %v", err)
	}
	if _, err := store.List(ctx, Filter{DomainCategory: domainCategoryPointer(DomainCategory("technology"))}); !errors.Is(err, ErrInvalidStorylineDomain) {
		t.Fatalf("List(invalid category) error = %v", err)
	}

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := store.Get(cancelled, first.ID); !errors.Is(err, context.Canceled) {
		t.Fatalf("Get(cancelled) error = %v", err)
	}
}

func TestStoreFailsClosedForUnknownPersistedStorylineDomainCategory(t *testing.T) {
	db := openStorylineDomainTestDatabase(t)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	created, err := store.Create(ctx, CreateInput{
		Name: "枚举漂移", NameEn: "Enum drift", Description: "用于测试持久化枚举漂移。",
		ScopeDefinition: "测试边界。", DomainCategory: DomainCategoryGeopolitical,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE storyline_domains ALTER COLUMN domain_category TYPE TEXT USING domain_category::text`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE storyline_domains SET domain_category = 'UNKNOWN' WHERE id = $1`, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, created.ID); !errors.Is(err, ErrPersistence) {
		t.Fatalf("Get(unknown persisted category) error = %v, want ErrPersistence", err)
	}
}

func domainCategoryPointer(value DomainCategory) *DomainCategory { return &value }

func boolPointer(value bool) *bool { return &value }

func openStorylineDomainTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, "tw_storyline_domain", migrationDir, 0)
}
