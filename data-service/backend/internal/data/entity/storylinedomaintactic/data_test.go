package storylinedomaintactic

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

func TestStoreCreatesGetsListsAndUpdatesStorylineDomainTactics(t *testing.T) {
	db := openStorylineDomainTacticTestDatabase(t)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	inputs := []CreateInput{
		{Key: "SUPPLY_SHOCK", Name: "供给冲击", NameEn: "Supply shock", Description: "追踪关键供给中断及其传导。"},
		{Key: "HOT_WAR", Name: "热战", NameEn: "Hot war", Description: "追踪公开军事冲突及其升级。"},
		{Key: "CAPITAL_CONTROL", Name: "资本管制", NameEn: "Capital control", Description: "追踪跨境资本流动限制。"},
	}
	created := make([]StorylineDomainTactic, 0, len(inputs))
	for _, input := range inputs {
		item, err := store.Create(ctx, input)
		if err != nil {
			t.Fatal(err)
		}
		created = append(created, item)
	}
	if !coreid.Is(created[0].ID, coreid.StorylineDomainTactic) || len(created[0].ID) != 39 {
		t.Fatalf("Create() ID = %q, want canonical SDT identity", created[0].ID)
	}
	if created[0].CreatedAt.IsZero() || !created[0].CreatedAt.Equal(created[0].UpdatedAt) || time.Since(created[0].CreatedAt) > time.Minute {
		t.Fatalf("Create() = %+v", created[0])
	}
	got, err := store.Get(ctx, created[0].ID)
	if err != nil || !reflect.DeepEqual(got, created[0]) {
		t.Fatalf("Get() = %+v, %v; want %+v", got, err, created[0])
	}

	all, err := store.List(ctx)
	wantOrder := []string{created[2].ID, created[1].ID, created[0].ID}
	if err != nil || len(all) != len(wantOrder) {
		t.Fatalf("List() = %+v, %v", all, err)
	}
	for index, wantID := range wantOrder {
		if all[index].ID != wantID {
			t.Fatalf("List()[%d].ID = %q, want %q; full list = %+v", index, all[index].ID, wantID, all)
		}
	}

	time.Sleep(time.Millisecond)
	updated, err := store.Update(ctx, UpdateInput{
		ID: created[1].ID, Name: "高烈度战争", NameEn: "High-intensity war",
		Description: "更新后的公开军事冲突手段定义。",
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.ID != created[1].ID || updated.Key != created[1].Key || !updated.CreatedAt.Equal(created[1].CreatedAt) || !updated.UpdatedAt.After(created[1].UpdatedAt) {
		t.Fatalf("Update() = %+v; created = %+v", updated, created[1])
	}
}

func TestStoreEnforcesStorylineDomainTacticContracts(t *testing.T) {
	db := openStorylineDomainTacticTestDatabase(t)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	valid := CreateInput{Key: "VALID_KEY", Name: "重复名称", NameEn: "Duplicate name", Description: "合法描述。"}
	first, err := store.Create(ctx, valid)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create(ctx, CreateInput{Key: "ANOTHER_KEY", Name: valid.Name, NameEn: valid.NameEn, Description: valid.Description}); err != nil {
		t.Fatalf("Create(duplicate names) error = %v", err)
	}
	if _, err := store.Create(ctx, valid); !errors.Is(err, ErrConflict) {
		t.Fatalf("Create(duplicate key) error = %v, want ErrConflict", err)
	}

	invalid := []CreateInput{
		{Key: "lower_case", Name: "小写", NameEn: "Lower case", Description: "描述"},
		{Key: "_LEADING", Name: "前导下划线", NameEn: "Leading underscore", Description: "描述"},
		{Key: strings.Repeat("A", 31), Name: "超长", NameEn: "Too long", Description: "描述"},
		{Key: "VALID", Name: " ", NameEn: "Blank", Description: "描述"},
		{Key: "VALID", Name: "无描述", NameEn: "Missing description", Description: " "},
	}
	for _, input := range invalid {
		if _, err := store.Create(ctx, input); !errors.Is(err, ErrInvalidStorylineDomainTactic) {
			t.Errorf("Create(%+v) error = %v, want ErrInvalidStorylineDomainTactic", input, err)
		}
	}
	if _, err := store.Get(ctx, "SLD11111111-1111-4111-8111-111111111111"); !errors.Is(err, ErrInvalidStorylineDomainTactic) {
		t.Fatalf("Get(wrong identity) error = %v", err)
	}
	if _, err := store.Get(ctx, "SDT11111111-1111-4111-8111-111111111111"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(missing) error = %v", err)
	}
	if _, err := store.Update(ctx, UpdateInput{
		ID: "SDT11111111-1111-4111-8111-111111111111", Name: "缺失", NameEn: "Missing", Description: "描述",
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update(missing) error = %v", err)
	}

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := store.Get(cancelled, first.ID); !errors.Is(err, context.Canceled) {
		t.Fatalf("Get(cancelled) error = %v", err)
	}
}

func openStorylineDomainTacticTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, "tw_storyline_domain_tactic", migrationDir, 0)
}
