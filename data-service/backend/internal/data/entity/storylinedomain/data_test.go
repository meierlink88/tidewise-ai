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
		{Code: "BETA_DOMAIN", Name: "Beta domain", NameEn: "Shared domain", Description: "地缘政治叙事领域。", ScopeDefinition: "覆盖国家间政治与安全竞争。", DomainCategory: DomainCategoryGeopolitical},
		{Code: "CORPORATE_GOVERNANCE", Name: "公司治理", NameEn: "Corporate governance", Description: "公司治理叙事领域。", ScopeDefinition: "覆盖公司控制、经营与治理议题。", DomainCategory: DomainCategoryCorporate, IsActive: boolPointer(false)},
		{Code: "ALPHA_DOMAIN_ONE", Name: "Alpha domain", NameEn: "Shared domain", Description: "宏观叙事领域。", ScopeDefinition: "覆盖货币、财政与经济数据议题。", DomainCategory: DomainCategoryMacro, IsActive: boolPointer(false)},
		{Code: "ALPHA_DOMAIN_TWO", Name: "Alpha domain", NameEn: "Shared domain", Description: "第二份宏观叙事领域。", ScopeDefinition: "覆盖另一组宏观经济边界。", DomainCategory: DomainCategoryMacro, IsActive: boolPointer(false)},
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
	if created[0].Code != inputs[0].Code {
		t.Fatalf("Create() code = %q, want %q", created[0].Code, inputs[0].Code)
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
	if updated.Code != created[0].Code {
		t.Fatalf("Update() code = %q, want immutable %q", updated.Code, created[0].Code)
	}
}

func TestPublishCurrentCatalogInitializesStorylineDomains(t *testing.T) {
	db := openStorylineDomainTestDatabase(t)
	ctx := context.Background()
	publication := CurrentCatalog()
	if len(publication) != 35 {
		t.Fatalf("CurrentCatalog() length = %d, want 35", len(publication))
	}
	if err := PublishCatalog(ctx, db, publication); err != nil {
		t.Fatalf("PublishCatalog() error = %v", err)
	}
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.List(ctx, Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 35 {
		t.Fatalf("List() length = %d, want 35", len(first))
	}
	counts := map[DomainCategory]int{}
	byCode := make(map[string]StorylineDomain, len(first))
	codes := make([]string, 0, len(first))
	for _, item := range first {
		counts[item.DomainCategory]++
		byCode[item.Code] = item
		codes = append(codes, item.Code)
		if !item.IsActive || item.ScopeDefinition != item.Description {
			t.Fatalf("published StorylineDomain = %+v", item)
		}
	}
	if counts[DomainCategoryGeopolitical] != 7 || counts[DomainCategoryMacro] != 12 ||
		counts[DomainCategoryIndustry] != 8 || counts[DomainCategoryCorporate] != 8 {
		t.Fatalf("category counts = %#v", counts)
	}
	sort.Strings(codes)
	wantCodes := []string{
		"ASSET_PURCHASE", "CAPITAL_MARKET", "COMPETITION_LANDSCAPE", "CYBER_SPACE",
		"DATA_RELEASE", "DEBT_MANAGEMENT", "DOWNSTREAM_DEMAND", "EARNINGS", "ENERGY",
		"EXPECTATION_GAP", "FINANCE", "FISCAL_SPENDING", "GOVERNANCE", "IDEOLOGY",
		"INVENTORY_CYCLE", "IPO_PROGRESS", "LEGAL_COMPLIANCE", "MARKET_PERFORMANCE",
		"MARKET_REACTION", "MIDSTREAM_MANUFACTURING", "MILITARY", "M_A", "POLICY_GUIDANCE",
		"PRICE_TRANSMISSION", "PRODUCT_TECH", "RATE_DECISION", "REGULATION", "TARIFF",
		"TAX_POLICY", "TECHNOLOGY", "TECHNOLOGY_BREAKTHROUGH", "TRADE", "TRADE_AGREEMENT",
		"TRADE_POLICY_IMPACT", "UPSTREAM_SUPPLY",
	}
	if !reflect.DeepEqual(codes, wantCodes) {
		t.Fatalf("published codes = %#v, want %#v", codes, wantCodes)
	}
	for _, want := range []struct {
		code, name, nameEn, description string
		category                        DomainCategory
	}{
		{"TECHNOLOGY", "科技线", "Technology", "围绕芯片、AI、技术标准、人才争夺的博弈", DomainCategoryGeopolitical},
		{"RATE_DECISION", "利率决策线", "Rate Decision", "政策利率调整、降息/加息决策", DomainCategoryMacro},
		{"UPSTREAM_SUPPLY", "上游供给线", "Upstream Supply", "原材料供应、矿产出产、上游产能", DomainCategoryIndustry},
		{"IPO_PROGRESS", "IPO进程线", "IPO Progress", "招股书提交、过会、定价、上市、首日表现", DomainCategoryCorporate},
	} {
		got, ok := byCode[want.code]
		wantID, idErr := coreid.Derive(coreid.StorylineDomain, "storyline-domain", want.code)
		if idErr != nil {
			t.Fatal(idErr)
		}
		if !ok || got.ID != wantID || got.Name != want.name || got.NameEn != want.nameEn ||
			got.Description != want.description || got.ScopeDefinition != want.description || got.DomainCategory != want.category {
			t.Fatalf("published %s = %+v, want ID=%q name=%q name_en=%q description=%q category=%q", want.code, got, wantID, want.name, want.nameEn, want.description, want.category)
		}
	}
	if err := PublishCatalog(ctx, db, publication); err != nil {
		t.Fatalf("PublishCatalog(repeat) error = %v", err)
	}
	second, err := store.List(ctx, Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(second, first) {
		t.Fatalf("repeated publication changed catalog:\nfirst=%+v\nsecond=%+v", first, second)
	}
	time.Sleep(time.Millisecond)
	if _, err := db.ExecContext(ctx, `
UPDATE storyline_domains
SET name = '漂移科技线',
    name_en = 'Drifted technology',
    description = '漂移描述',
    scope_definition = '漂移边界',
    domain_category = 'MACRO',
    is_active = FALSE
WHERE code = 'TECHNOLOGY'`); err != nil {
		t.Fatalf("drift TECHNOLOGY facts: %v", err)
	}
	if err := PublishCatalog(ctx, db, publication); err != nil {
		t.Fatalf("PublishCatalog(reconcile drift) error = %v", err)
	}
	afterDrift, err := store.List(ctx, Filter{})
	if err != nil {
		t.Fatal(err)
	}
	driftedByCode := make(map[string]StorylineDomain, len(afterDrift))
	for _, item := range afterDrift {
		driftedByCode[item.Code] = item
	}
	technology := driftedByCode["TECHNOLOGY"]
	wantTechnology := byCode["TECHNOLOGY"]
	if technology.ID != wantTechnology.ID || technology.Code != wantTechnology.Code ||
		technology.Name != wantTechnology.Name || technology.NameEn != wantTechnology.NameEn ||
		technology.Description != wantTechnology.Description || technology.ScopeDefinition != wantTechnology.Description ||
		technology.DomainCategory != DomainCategoryGeopolitical || !technology.IsActive ||
		!technology.UpdatedAt.After(wantTechnology.UpdatedAt) {
		t.Fatalf("reconciled TECHNOLOGY = %+v, want stable identity and restored descriptive/category/active facts", technology)
	}
}

func TestPublishCatalogFailsClosedOnStorylineDomainIdentityDrift(t *testing.T) {
	db := openStorylineDomainTestDatabase(t)
	ctx := context.Background()
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	existing, err := store.Create(ctx, CreateInput{
		Code: "TECHNOLOGY", Name: "冲突科技线", NameEn: "Conflicting technology",
		Description: "冲突描述", ScopeDefinition: "冲突边界", DomainCategory: DomainCategoryGeopolitical,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := PublishCatalog(ctx, db, CurrentCatalog()); !errors.Is(err, ErrConflict) {
		t.Fatalf("PublishCatalog(identity drift) error = %v, want ErrConflict", err)
	}
	listed, err := store.List(ctx, Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0] != existing {
		t.Fatalf("failed publication changed facts: got %+v, want only %+v", listed, existing)
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
		Code: "DUPLICATE_NAME_ONE", Name: "重复名称", NameEn: "Duplicate name", Description: "合法描述。",
		ScopeDefinition: "合法边界。", DomainCategory: DomainCategoryIndustry,
	}
	first, err := store.Create(ctx, valid)
	if err != nil {
		t.Fatal(err)
	}
	duplicateName := valid
	duplicateName.Code = "DUPLICATE_NAME_TWO"
	if _, err := store.Create(ctx, duplicateName); err != nil {
		t.Fatalf("Create(duplicate names) error = %v", err)
	}
	if _, err := store.Create(ctx, valid); !errors.Is(err, ErrConflict) {
		t.Fatalf("Create(duplicate code) error = %v, want ErrConflict", err)
	}

	invalid := []CreateInput{
		{Code: "", Name: "空编码", NameEn: "Blank code", Description: "描述", ScopeDefinition: "边界", DomainCategory: DomainCategoryIndustry},
		{Code: "lowercase", Name: "小写编码", NameEn: "Lowercase code", Description: "描述", ScopeDefinition: "边界", DomainCategory: DomainCategoryIndustry},
		{Code: strings.Repeat("A", 31), Name: "过长编码", NameEn: "Long code", Description: "描述", ScopeDefinition: "边界", DomainCategory: DomainCategoryIndustry},
		{Code: "BLANK_NAME", Name: " ", NameEn: "Blank", Description: "描述", ScopeDefinition: "边界", DomainCategory: DomainCategoryIndustry},
		{Code: "LONG_NAME_EN", Name: "超长", NameEn: strings.Repeat("x", 51), Description: "描述", ScopeDefinition: "边界", DomainCategory: DomainCategoryIndustry},
		{Code: "UNKNOWN_CATEGORY", Name: "未知分类", NameEn: "Unknown", Description: "描述", ScopeDefinition: "边界", DomainCategory: DomainCategory("TECHNOLOGY")},
		{Code: "NO_DESCRIPTION", Name: "无描述", NameEn: "Missing description", Description: " ", ScopeDefinition: "边界", DomainCategory: DomainCategoryIndustry},
		{Code: "NO_SCOPE", Name: "无边界", NameEn: "Missing scope", Description: "描述", ScopeDefinition: " ", DomainCategory: DomainCategoryIndustry},
	}
	for _, input := range invalid {
		if _, err := store.Create(ctx, input); !errors.Is(err, ErrInvalidStorylineDomain) {
			t.Errorf("Create(%+v) error = %v, want ErrInvalidStorylineDomain", input, err)
		}
	}
	for _, query := range []string{
		`INSERT INTO storyline_domains (id,code,name,name_en,description,scope_definition,domain_category) VALUES ('SLD11111111-1111-4111-8111-111111111111','lowercase','无效','Invalid','描述','边界','INDUSTRY')`,
		`INSERT INTO storyline_domains (id,code,name,name_en,description,scope_definition,domain_category) VALUES ('SLD11111111-1111-4111-8111-111111111111',NULL,'无效','Invalid','描述','边界','INDUSTRY')`,
	} {
		if _, err := db.ExecContext(ctx, query); err == nil {
			t.Fatalf("invalid database code accepted: %s", query)
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
		Code: "ENUM_DRIFT", Name: "枚举漂移", NameEn: "Enum drift", Description: "用于测试持久化枚举漂移。",
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
