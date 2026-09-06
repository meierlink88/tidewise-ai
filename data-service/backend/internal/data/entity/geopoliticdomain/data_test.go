package geopoliticdomain

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

func TestStorePersistsGeopoliticDomainTacticArray(t *testing.T) {
	db := openGeopoliticDomainTestDatabase(t)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	tactics := []Tactic{
		{Name: "技术出口管制", Description: "芯片、EDA和 AI 等关键技术出口管制"},
		{Name: "实体清单", Description: "实体清单、SDN 清单纳入"},
	}

	created, err := store.Create(context.Background(), CreateInput{
		Code: "TECHNOLOGY_STANDARDS", Name: "科技/标准线",
		Description: "芯片、AI、通信技术、技术标准争夺、技术出口管制、科技人才争夺",
		Tactics:     tactics,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !coreid.Is(created.ID, coreid.GeopoliticDomain) || created.Code != "TECHNOLOGY_STANDARDS" ||
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

func TestStoreEnforcesGeopoliticDomainContracts(t *testing.T) {
	db := openGeopoliticDomainTestDatabase(t)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	valid := CreateInput{
		Code: "MILITARY", Name: "军事/防务线", Description: "军事行动、军备竞赛与军事同盟",
		Tactics: []Tactic{{Name: "热战", Description: "正面战场、直接交火、军事打击"}},
	}
	if _, err := store.Create(ctx, valid); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create(ctx, valid); !errors.Is(err, ErrConflict) {
		t.Fatalf("Create(duplicate code) error = %v, want ErrConflict", err)
	}

	invalid := []CreateInput{
		{Code: "military", Name: "军事线", Description: "描述", Tactics: valid.Tactics},
		{Code: "MILITARY", Name: " ", Description: "描述", Tactics: valid.Tactics},
		{Code: "MILITARY", Name: "军事线", Description: "描述"},
		{Code: "MILITARY", Name: "军事线", Description: "描述", Tactics: []Tactic{{Name: " ", Description: "描述"}}},
		{Code: "MILITARY", Name: "军事线", Description: "描述", Tactics: []Tactic{{Name: "热战", Description: "描述"}, {Name: "热战", Description: "重复"}}},
	}
	for _, input := range invalid {
		if _, err := store.Create(ctx, input); !errors.Is(err, ErrInvalidGeopoliticDomain) {
			t.Errorf("Create(%#v) error = %v, want ErrInvalidGeopoliticDomain", input, err)
		}
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO geopolitic_domains (
    id, code, name, description, tactics
) VALUES (
    'GPD11111111-1111-4111-8111-111111111111', 'INVALID_JSON', '非法数组', '非法持久化数据',
    '[{"name":"热战","description":"描述","unexpected":true}]'::jsonb
)`); err == nil {
		t.Fatal("database accepted a tactic object with an extra field")
	}
	if _, err := db.ExecContext(ctx, `UPDATE geopolitic_domains SET code = 'MILITARY_CHANGED' WHERE code = 'MILITARY'`); err == nil {
		t.Fatal("database accepted GeopoliticDomain code mutation")
	}
}

func openGeopoliticDomainTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, "tw_geopolitic_domain", migrationDir, 0)
}
