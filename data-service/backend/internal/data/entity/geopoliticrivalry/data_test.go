package geopoliticrivalry

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
	geopoliticdomaindata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/geopoliticdomain"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

func TestStorePersistsGeopoliticalStorylineWithOneDomain(t *testing.T) {
	db := openGeopoliticRivalryTestDatabase(t)
	domainID := createDomain(t, db, "MILITARY", "军事/防务线")
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	input := CreateInput{
		Name: "俄乌战争", Category: "俄乌及欧洲安全", GeopoliticDomainID: domainID,
		CoreProposition:  "俄罗斯与乌克兰之间的战场进程、领土控制和停火安排发生变化",
		CoreActors:       "俄罗斯、乌克兰及直接军援方",
		MainTransmission: "战争进程→风险偏好、军工需求及地区基础设施风险变化",
		CandidateAssets:  []string{"黄金", "原油", "国防军工板块"},
	}
	created, err := store.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !coreid.Is(created.ID, coreid.GeopoliticRivalry) || created.GeopoliticDomainID != domainID {
		t.Fatalf("Create() = %#v", created)
	}
	if created.CreatedAt.IsZero() || !created.CreatedAt.Equal(created.UpdatedAt) || time.Since(created.CreatedAt) > time.Minute {
		t.Fatalf("Create() times = %s, %s", created.CreatedAt, created.UpdatedAt)
	}

	if created.ShortName != nil {
		t.Fatal("new storyline must have null short_name")
	}
	for _, invalid := range []string{"", "   ", "一二三四五六"} {
		if _, err := db.ExecContext(context.Background(), "UPDATE geopolitic_rivalries SET short_name=$1 WHERE id=$2", invalid, created.ID); err == nil {
			t.Fatal("accepted invalid short_name")
		}
		corrupt := created
		corrupt.ShortName = &invalid
		if err := validateStored(corrupt); err == nil {
			t.Fatal("accepted corrupted stored short_name")
		}
	}
	label := "俄乌战争"
	if _, err := db.ExecContext(context.Background(), "UPDATE geopolitic_rivalries SET short_name=$1 WHERE id=$2", label, created.ID); err != nil {
		t.Fatal(err)
	}
	created.ShortName = &label

	got, err := store.Get(context.Background(), created.ID)
	if err != nil || !reflect.DeepEqual(got, created) {
		t.Fatalf("Get() = %#v, %v; want %#v", got, err, created)
	}
	updated, err := store.Update(context.Background(), UpdateInput{
		ID: created.ID, Name: input.Name, Category: input.Category, GeopoliticDomainID: input.GeopoliticDomainID,
		CoreProposition: input.CoreProposition, CoreActors: input.CoreActors,
		MainTransmission: input.MainTransmission, CandidateAssets: []string{"原油", "黄金", "VIX指数"},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if !reflect.DeepEqual(updated.CandidateAssets, []string{"原油", "黄金", "VIX指数"}) {
		t.Fatalf("Update() candidate assets = %#v", updated.CandidateAssets)
	}
	if updated.ShortName == nil || *updated.ShortName != label {
		t.Fatal("update erased short_name")
	}

	listed, err := store.List(context.Background(), Filter{})
	if err != nil || len(listed) != 1 || !reflect.DeepEqual(listed[0], updated) {
		t.Fatalf("List() = %#v, %v; want %#v", listed, err, updated)
	}
}

func TestStoreFiltersGeopoliticalStorylinesAndEnforcesDomainReference(t *testing.T) {
	db := openGeopoliticRivalryTestDatabase(t)
	militaryID := createDomain(t, db, "MILITARY", "军事/防务线")
	financeID := createDomain(t, db, "FINANCE_MONETARY", "金融/货币线")
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, input := range []CreateInput{
		validInput("俄乌战争", "俄乌及欧洲安全", militaryID),
		validInput("美伊军事冲突", "中东安全", militaryID),
		validInput("美国对伊经济制裁", "中东安全", financeID),
	} {
		if _, err := store.Create(ctx, input); err != nil {
			t.Fatal(err)
		}
	}
	byDomain, err := store.List(ctx, Filter{GeopoliticDomainID: &militaryID})
	if err != nil || len(byDomain) != 2 || byDomain[0].Name != "俄乌战争" || byDomain[1].Name != "美伊军事冲突" {
		t.Fatalf("List(domain) = %#v, %v", byDomain, err)
	}
	category := "中东安全"
	byCategory, err := store.List(ctx, Filter{Category: &category})
	if err != nil || len(byCategory) != 2 {
		t.Fatalf("List(category) = %#v, %v", byCategory, err)
	}

	missingDomain, err := coreid.New(coreid.GeopoliticDomain)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create(ctx, validInput("无领域故事线", "测试", missingDomain)); !errors.Is(err, ErrInvalidGeopoliticRivalry) {
		t.Fatalf("Create(missing domain) error = %v, want ErrInvalidGeopoliticRivalry", err)
	}
	if _, err := store.Create(ctx, validInput("俄乌战争", "重复", militaryID)); !errors.Is(err, ErrConflict) {
		t.Fatalf("Create(duplicate name) error = %v, want ErrConflict", err)
	}
}

func TestStoreRejectsInvalidCandidateAssets(t *testing.T) {
	db := openGeopoliticRivalryTestDatabase(t)
	domainID := createDomain(t, db, "MILITARY", "军事/防务线")
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, candidateAssets := range [][]string{
		nil,
		{},
		{" "},
		{"黄金", "黄金"},
		{strings.Repeat("资", 101)},
	} {
		input := validInput("无效候选资产故事线", "测试", domainID)
		input.CandidateAssets = candidateAssets
		if _, err := store.Create(ctx, input); !errors.Is(err, ErrInvalidGeopoliticRivalry) {
			t.Fatalf("Create(candidate assets %#v) error = %v, want ErrInvalidGeopoliticRivalry", candidateAssets, err)
		}
	}

	created, err := store.Create(ctx, validInput("数据库候选资产约束", "测试", domainID))
	if err != nil {
		t.Fatal(err)
	}
	for _, payload := range []string{`null`, `[]`, `[1]`, `[""]`, `["黄金","黄金"]`} {
		if _, err := db.ExecContext(ctx, `UPDATE geopolitic_rivalries SET candidate_assets = $2::jsonb WHERE id = $1`, created.ID, payload); err == nil {
			t.Fatalf("database accepted invalid candidate_assets %s", payload)
		}
	}
}

func validInput(name, category, domainID string) CreateInput {
	return CreateInput{
		Name: name, Category: category, GeopoliticDomainID: domainID,
		CoreProposition:  "每条故事线只表达一个核心命题",
		CoreActors:       "核心参与方",
		MainTransmission: "直接影响→对中国经济的主要传导",
		CandidateAssets:  []string{"黄金", "VIX指数"},
	}
}

func createDomain(t *testing.T, db *sql.DB, code, name string) string {
	t.Helper()
	store, err := geopoliticdomaindata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	domain, err := store.Create(context.Background(), geopoliticdomaindata.CreateInput{
		Code: code, Name: name, Description: name + "定义",
		Tactics: []geopoliticdomaindata.Tactic{{Name: "手段", Description: "手段描述"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return domain.ID
}

func openGeopoliticRivalryTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, "tw_geopolitic_rivalry", migrationDir, 0)
}
