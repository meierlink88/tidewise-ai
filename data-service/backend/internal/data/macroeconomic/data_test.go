package macroeconomic

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

func TestStoreCreatesAndGetsMacroEconomic(t *testing.T) {
	db := openMacroEconomicTestDatabase(t)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}

	created, err := store.Create(context.Background(), CreateInput{
		Name:        "美国货币政策",
		NameEn:      "United States monetary policy",
		MacroType:   MacroTypeMonetary,
		Description: "描述美国货币政策取向、工具及其宏观传导的静态叙事蓝图。",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !coreid.Is(created.ID, coreid.MacroEconomic) || len(created.ID) != 39 {
		t.Fatalf("Create() ID = %q, want canonical MEC identity", created.ID)
	}
	if created.Status != StatusActive || created.CreatedAt.IsZero() || !created.CreatedAt.Equal(created.UpdatedAt) || time.Since(created.CreatedAt) > time.Minute {
		t.Fatalf("Create() = %+v", created)
	}

	got, err := store.Get(context.Background(), created.ID)
	if err != nil || !reflect.DeepEqual(got, created) {
		t.Fatalf("Get() = %+v, %v; want %+v", got, err, created)
	}
}

func TestStoreListsFiltersAndUpdatesMacroEconomics(t *testing.T) {
	db := openMacroEconomicTestDatabase(t)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	inputs := []CreateInput{
		{Name: "贸易政策", NameEn: "Trade policy", MacroType: MacroTypeTradePolicy, Description: "贸易政策叙事蓝图。"},
		{Name: "财政政策", NameEn: "Fiscal policy", MacroType: MacroTypeFiscal, Description: "财政政策叙事蓝图。", Status: StatusDormant},
		{Name: "Beta framework", NameEn: "Shared framework", MacroType: MacroTypeDataEconomic, Description: "经济数据叙事蓝图。"},
		{Name: "Alpha framework", NameEn: "Shared framework", MacroType: MacroTypeDataEconomic, Description: "区域经济数据叙事蓝图。", Status: StatusDormant},
		{Name: "Alpha framework", NameEn: "Shared framework", MacroType: MacroTypeDataEconomic, Description: "全球经济数据叙事蓝图。", Status: StatusDormant},
	}
	created := make([]MacroEconomic, 0, len(inputs))
	for _, input := range inputs {
		item, err := store.Create(ctx, input)
		if err != nil {
			t.Fatal(err)
		}
		created = append(created, item)
	}

	all, err := store.List(ctx, Filter{})
	tiedIDs := []string{created[3].ID, created[4].ID}
	sort.Strings(tiedIDs)
	wantOrder := []string{created[1].ID, tiedIDs[0], tiedIDs[1], created[2].ID, created[0].ID}
	if err != nil || len(all) != len(wantOrder) {
		t.Fatalf("List() = %+v, %v", all, err)
	}
	for index, wantID := range wantOrder {
		if all[index].ID != wantID {
			t.Fatalf("List()[%d].ID = %q, want %q; full list = %+v", index, all[index].ID, wantID, all)
		}
	}
	byType, err := store.List(ctx, Filter{MacroType: macroTypePointer(MacroTypeDataEconomic)})
	if err != nil || len(byType) != 3 {
		t.Fatalf("List(type) = %+v, %v", byType, err)
	}
	byStatus, err := store.List(ctx, Filter{Status: statusPointer(StatusDormant)})
	if err != nil || len(byStatus) != 3 {
		t.Fatalf("List(status) = %+v, %v", byStatus, err)
	}
	combined, err := store.List(ctx, Filter{
		MacroType: macroTypePointer(MacroTypeDataEconomic),
		Status:    statusPointer(StatusDormant),
	})
	if err != nil || len(combined) != 2 || combined[0].ID != tiedIDs[0] || combined[1].ID != tiedIDs[1] {
		t.Fatalf("List(type AND status) = %+v, %v", combined, err)
	}

	time.Sleep(time.Millisecond)
	updated, err := store.Update(ctx, UpdateInput{
		ID: created[0].ID, Name: "国际贸易政策", NameEn: "International trade policy",
		MacroType: MacroTypeTradePolicy, Description: "更新后的贸易政策叙事蓝图。", Status: StatusArchived,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.ID != created[0].ID || !updated.CreatedAt.Equal(created[0].CreatedAt) || !updated.UpdatedAt.After(created[0].UpdatedAt) || updated.Status != StatusArchived {
		t.Fatalf("Update() = %+v; created = %+v", updated, created[0])
	}
}

func TestStoreEnforcesMacroEconomicContracts(t *testing.T) {
	db := openMacroEconomicTestDatabase(t)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	valid := CreateInput{Name: "重复名称", NameEn: "Duplicate name", MacroType: MacroTypeRegulatory, Description: "合法描述。"}
	first, err := store.Create(ctx, valid)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create(ctx, valid); err != nil {
		t.Fatalf("Create(duplicate names) error = %v", err)
	}

	invalid := []CreateInput{
		{Name: " ", NameEn: "Blank", MacroType: MacroTypeMonetary, Description: "描述"},
		{Name: "未知类型", NameEn: "Unknown type", MacroType: MacroType("ECONOMIC_DATA"), Description: "描述"},
		{Name: "未知状态", NameEn: "Unknown status", MacroType: MacroTypeFiscal, Description: "描述", Status: Status("RESOLVED")},
		{Name: "无描述", NameEn: "Missing description", MacroType: MacroTypeFiscal, Description: " "},
	}
	for _, input := range invalid {
		if _, err := store.Create(ctx, input); !errors.Is(err, ErrInvalidMacroEconomic) {
			t.Errorf("Create(%+v) error = %v, want ErrInvalidMacroEconomic", input, err)
		}
	}
	if _, err := store.Get(ctx, "GPR11111111-1111-4111-8111-111111111111"); !errors.Is(err, ErrInvalidMacroEconomic) {
		t.Fatalf("Get(wrong identity) error = %v", err)
	}
	if _, err := store.Get(ctx, "MEC11111111-1111-4111-8111-111111111111"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(missing) error = %v", err)
	}
	if _, err := store.Update(ctx, UpdateInput{
		ID: "MEC11111111-1111-4111-8111-111111111111", Name: "缺失", NameEn: "Missing",
		MacroType: MacroTypeMonetary, Description: "描述", Status: StatusActive,
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update(missing) error = %v", err)
	}
	if _, err := store.List(ctx, Filter{MacroType: macroTypePointer(MacroType("monetary"))}); !errors.Is(err, ErrInvalidMacroEconomic) {
		t.Fatalf("List(invalid type) error = %v", err)
	}
	if _, err := store.List(ctx, Filter{Status: statusPointer(Status("active"))}); !errors.Is(err, ErrInvalidMacroEconomic) {
		t.Fatalf("List(invalid status) error = %v", err)
	}

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := store.Get(cancelled, first.ID); !errors.Is(err, context.Canceled) {
		t.Fatalf("Get(cancelled) error = %v", err)
	}
}

func TestStoreFailsClosedForUnknownPersistedMacroEconomicEnums(t *testing.T) {
	db := openMacroEconomicTestDatabase(t)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	created, err := store.Create(ctx, CreateInput{
		Name: "枚举漂移", NameEn: "Enum drift", MacroType: MacroTypeDataEconomic,
		Description: "用于测试持久化枚举漂移。",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE macro_economics ALTER COLUMN macro_type TYPE TEXT USING macro_type::text`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE macro_economics SET macro_type = 'UNKNOWN' WHERE id = $1`, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, created.ID); !errors.Is(err, ErrPersistence) {
		t.Fatalf("Get(unknown persisted macro type) error = %v, want ErrPersistence", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE macro_economics SET macro_type = 'DATA_ECONOMIC' WHERE id = $1`, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE macro_economics ALTER COLUMN status DROP DEFAULT`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE macro_economics ALTER COLUMN status TYPE TEXT USING status::text`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE macro_economics SET status = 'UNKNOWN' WHERE id = $1`, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, created.ID); !errors.Is(err, ErrPersistence) {
		t.Fatalf("Get(unknown persisted status) error = %v, want ErrPersistence", err)
	}
}

func macroTypePointer(value MacroType) *MacroType { return &value }

func statusPointer(value Status) *Status { return &value }

func openMacroEconomicTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, "tw_macro_economic", migrationDir, 0)
}
