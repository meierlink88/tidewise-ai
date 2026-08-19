package geopoliticrivalry

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

func TestStoreCreatesAndGetsGeopoliticRivalry(t *testing.T) {
	db := openGeopoliticRivalryTestDatabase(t)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	peripheralActors := "代理组织与区域伙伴"
	regions := []string{"中东", "波斯湾"}

	created, err := store.Create(context.Background(), CreateInput{
		Name:              "美伊对抗",
		NameEn:            "United States-Iran rivalry",
		RivalryType:       RivalryTypeGeopolitical,
		Description:       "围绕地区安全、核问题与制裁形成的长期地缘政治对抗。",
		CoreActors:        "美国；伊朗",
		PeripheralActors:  &peripheralActors,
		InfluencedRegions: regions,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !coreid.Is(created.ID, coreid.GeopoliticRivalry) || len(created.ID) != 39 {
		t.Fatalf("Create() ID = %q, want canonical GPR identity", created.ID)
	}
	if created.Status != StatusActive || created.PeripheralActors == nil || *created.PeripheralActors != peripheralActors {
		t.Fatalf("Create() = %+v", created)
	}
	if !reflect.DeepEqual(created.InfluencedRegions, regions) {
		t.Fatalf("Create() influenced regions = %#v, want %#v", created.InfluencedRegions, regions)
	}
	if created.CreatedAt.IsZero() || !created.CreatedAt.Equal(created.UpdatedAt) || time.Since(created.CreatedAt) > time.Minute {
		t.Fatalf("Create() times = %s, %s", created.CreatedAt, created.UpdatedAt)
	}

	got, err := store.Get(context.Background(), created.ID)
	if err != nil || !reflect.DeepEqual(got, created) {
		t.Fatalf("Get() = %+v, %v; want %+v", got, err, created)
	}
}

func TestStoreListsFiltersAndUpdatesGeopoliticRivalries(t *testing.T) {
	db := openGeopoliticRivalryTestDatabase(t)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	inputs := []CreateInput{
		{Name: "Beta rivalry", NameEn: "Shared rivalry", RivalryType: RivalryTypeGeopolitical, Description: "地区安全竞争。", CoreActors: "国家甲；国家乙"},
		{Name: "边境战争", NameEn: "Border war", RivalryType: RivalryTypeMilitaryWar, Description: "持续中的边境战争。", CoreActors: "国家丙；国家丁", Status: StatusDormant, InfluencedRegions: []string{}},
		{Name: "Alpha rivalry", NameEn: "Shared rivalry", RivalryType: RivalryTypeGeopolitical, Description: "海上通道竞争。", CoreActors: "国家戊；国家己", Status: StatusDormant},
		{Name: "Alpha rivalry", NameEn: "Shared rivalry", RivalryType: RivalryTypeGeopolitical, Description: "另一份海上通道竞争蓝图。", CoreActors: "国家庚；国家辛", Status: StatusDormant},
	}
	created := make([]GeopoliticRivalry, 0, len(inputs))
	for _, input := range inputs {
		item, err := store.Create(ctx, input)
		if err != nil {
			t.Fatal(err)
		}
		created = append(created, item)
	}
	if created[0].InfluencedRegions != nil {
		t.Fatalf("nil influenced regions = %#v", created[0].InfluencedRegions)
	}
	if created[1].InfluencedRegions == nil || len(created[1].InfluencedRegions) != 0 {
		t.Fatalf("empty influenced regions = %#v", created[1].InfluencedRegions)
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
	byType, err := store.List(ctx, Filter{RivalryType: rivalryTypePointer(RivalryTypeGeopolitical)})
	if err != nil || len(byType) != 3 {
		t.Fatalf("List(type) = %+v, %v", byType, err)
	}
	byStatus, err := store.List(ctx, Filter{Status: statusPointer(StatusDormant)})
	if err != nil || len(byStatus) != 3 {
		t.Fatalf("List(status) = %+v, %v", byStatus, err)
	}
	combined, err := store.List(ctx, Filter{
		RivalryType: rivalryTypePointer(RivalryTypeGeopolitical),
		Status:      statusPointer(StatusDormant),
	})
	if err != nil || len(combined) != 2 || combined[0].ID != tiedIDs[0] || combined[1].ID != tiedIDs[1] {
		t.Fatalf("List(type AND status) = %+v, %v", combined, err)
	}

	time.Sleep(time.Millisecond)
	updated, err := store.Update(ctx, UpdateInput{
		ID:          created[0].ID,
		Name:        "中东地缘竞争",
		NameEn:      "Middle East geopolitical rivalry",
		RivalryType: RivalryTypeGeopolitical,
		Description: "更新后的地区安全竞争蓝图。",
		CoreActors:  "国家甲；国家乙",
		Status:      StatusResolved,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.ID != created[0].ID || !updated.CreatedAt.Equal(created[0].CreatedAt) || !updated.UpdatedAt.After(created[0].UpdatedAt) || updated.Status != StatusResolved {
		t.Fatalf("Update() = %+v; created = %+v", updated, created[0])
	}
}

func TestStoreEnforcesGeopoliticRivalryContracts(t *testing.T) {
	db := openGeopoliticRivalryTestDatabase(t)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	valid := CreateInput{
		Name: "重复名称", NameEn: "Duplicate name", RivalryType: RivalryTypeGeopolitical,
		Description: "合法描述。", CoreActors: "参与方甲；参与方乙",
	}
	first, err := store.Create(ctx, valid)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create(ctx, valid); err != nil {
		t.Fatalf("Create(duplicate names) error = %v", err)
	}

	invalid := []CreateInput{
		{Name: " ", NameEn: "Blank", RivalryType: RivalryTypeGeopolitical, Description: "描述", CoreActors: "甲；乙"},
		{Name: "未知类型", NameEn: "Unknown type", RivalryType: RivalryType("geopolitical"), Description: "描述", CoreActors: "甲；乙"},
		{Name: "未知状态", NameEn: "Unknown status", RivalryType: RivalryTypeGeopolitical, Description: "描述", CoreActors: "甲；乙", Status: Status("ARCHIVED")},
		{Name: "无核心参与方", NameEn: "Missing core actors", RivalryType: RivalryTypeGeopolitical, Description: "描述", CoreActors: " "},
	}
	for _, input := range invalid {
		if _, err := store.Create(ctx, input); !errors.Is(err, ErrInvalidGeopoliticRivalry) {
			t.Errorf("Create(%+v) error = %v, want ErrInvalidGeopoliticRivalry", input, err)
		}
	}
	if _, err := store.Get(ctx, "MEC11111111-1111-4111-8111-111111111111"); !errors.Is(err, ErrInvalidGeopoliticRivalry) {
		t.Fatalf("Get(wrong identity) error = %v", err)
	}
	if _, err := store.Get(ctx, "GPR11111111-1111-4111-8111-111111111111"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(missing) error = %v", err)
	}
	if _, err := store.Update(ctx, UpdateInput{
		ID: "GPR11111111-1111-4111-8111-111111111111", Name: "缺失", NameEn: "Missing",
		RivalryType: RivalryTypeGeopolitical, Description: "描述", CoreActors: "甲；乙", Status: StatusActive,
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update(missing) error = %v", err)
	}
	if _, err := store.List(ctx, Filter{Status: statusPointer(Status("active"))}); !errors.Is(err, ErrInvalidGeopoliticRivalry) {
		t.Fatalf("List(invalid status) error = %v", err)
	}
	if _, err := store.List(ctx, Filter{RivalryType: rivalryTypePointer(RivalryType("geopolitical"))}); !errors.Is(err, ErrInvalidGeopoliticRivalry) {
		t.Fatalf("List(invalid type) error = %v", err)
	}

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := store.Get(cancelled, first.ID); !errors.Is(err, context.Canceled) {
		t.Fatalf("Get(cancelled) error = %v", err)
	}
}

func TestStoreFailsClosedForUnknownPersistedGeopoliticRivalryEnums(t *testing.T) {
	db := openGeopoliticRivalryTestDatabase(t)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	created, err := store.Create(ctx, CreateInput{
		Name: "枚举漂移", NameEn: "Enum drift", RivalryType: RivalryTypeGeopolitical,
		Description: "用于测试持久化枚举漂移。", CoreActors: "甲；乙",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE geopolitic_rivalries ALTER COLUMN rivalry_type TYPE TEXT USING rivalry_type::text`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE geopolitic_rivalries SET rivalry_type = 'UNKNOWN' WHERE id = $1`, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, created.ID); !errors.Is(err, ErrPersistence) {
		t.Fatalf("Get(unknown persisted rivalry type) error = %v, want ErrPersistence", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE geopolitic_rivalries SET rivalry_type = 'GEOPOLITICAL' WHERE id = $1`, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE geopolitic_rivalries ALTER COLUMN status DROP DEFAULT`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE geopolitic_rivalries ALTER COLUMN status TYPE TEXT USING status::text`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE geopolitic_rivalries SET status = 'UNKNOWN' WHERE id = $1`, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, created.ID); !errors.Is(err, ErrPersistence) {
		t.Fatalf("Get(unknown persisted status) error = %v, want ErrPersistence", err)
	}
}

func rivalryTypePointer(value RivalryType) *RivalryType { return &value }

func statusPointer(value Status) *Status { return &value }

func openGeopoliticRivalryTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, "tw_geopolitic_rivalry", migrationDir, 0)
}
