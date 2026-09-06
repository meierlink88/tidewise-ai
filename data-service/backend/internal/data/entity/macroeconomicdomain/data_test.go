package macroeconomicdomain

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

func TestStorePersistsMacroEconomicDomainTacticArray(t *testing.T) {
	db := openMacroEconomicDomainTestDatabase(t)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	tactics := []Tactic{
		{Name: "降息", Description: "芯片、EDA和 AI 等关键降息"},
		{Name: "前瞻指引", Description: "央行政策意图引导"},
	}

	created, err := store.Create(context.Background(), CreateInput{
		Code: "MONETARY", Name: "货币政策线",
		Description: "芯片、AI、通信技术、技术标准争夺、降息、科技人才争夺",
		Tactics:     tactics,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !coreid.Is(created.ID, coreid.MacroEconomicDomain) || created.Code != "MONETARY" ||
		!reflect.DeepEqual(created.Tactics, tactics) {
		t.Fatalf("Create() = %#v", created)
	}
	if created.CreatedAt.IsZero() || !created.CreatedAt.Equal(created.UpdatedAt) || time.Since(created.CreatedAt) > time.Minute {
		t.Fatalf("Create() times = %s, %s", created.CreatedAt, created.UpdatedAt)
	}

	got, err := store.Get(context.Background(), created.ID)
	if err != nil || !reflect.DeepEqual(got, created) {
		t.Fatalf("Get() = %#v, %v; want %#v", got, err, created)
	}
}

func TestStoreEnforcesMacroEconomicDomainContracts(t *testing.T) {
	db := openMacroEconomicDomainTestDatabase(t)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	valid := CreateInput{
		Code: "FISCAL", Name: "财政政策线", Description: "财政支出、税收政策与财政赤字",
		Tactics: []Tactic{{Name: "财政刺激", Description: "扩大财政支出或减税"}},
	}
	if _, err := store.Create(ctx, valid); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create(ctx, valid); !errors.Is(err, ErrConflict) {
		t.Fatalf("Create(duplicate code) error = %v, want ErrConflict", err)
	}

	invalid := []CreateInput{
		{Code: "fiscal", Name: "财政线", Description: "描述", Tactics: valid.Tactics},
		{Code: "FISCAL", Name: " ", Description: "描述", Tactics: valid.Tactics},
		{Code: "FISCAL", Name: "财政线", Description: "描述"},
		{Code: "FISCAL", Name: "财政线", Description: "描述", Tactics: []Tactic{{Name: " ", Description: "描述"}}},
		{Code: "FISCAL", Name: "财政线", Description: "描述", Tactics: []Tactic{{Name: "财政刺激", Description: "描述"}, {Name: "财政刺激", Description: "重复"}}},
	}
	for _, input := range invalid {
		if _, err := store.Create(ctx, input); !errors.Is(err, ErrInvalidMacroEconomicDomain) {
			t.Errorf("Create(%#v) error = %v, want ErrInvalidMacroEconomicDomain", input, err)
		}
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO macro_economics_domain (
    id, code, name, description, tactics
) VALUES (
    'MCD11111111-1111-4111-8111-111111111111', 'INVALID_JSON', '非法数组', '非法持久化数据',
    '[{"name":"财政刺激","description":"描述","unexpected":true}]'::jsonb
)`); err == nil {
		t.Fatal("database accepted a tactic object with an extra field")
	}
	if _, err := db.ExecContext(ctx, `UPDATE macro_economics_domain SET code = 'FISCAL_CHANGED' WHERE code = 'FISCAL'`); err == nil {
		t.Fatal("database accepted MacroEconomicDomain code mutation")
	}
}

func openMacroEconomicDomainTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, "tw_macroeconomic_domain", migrationDir, 0)
}
