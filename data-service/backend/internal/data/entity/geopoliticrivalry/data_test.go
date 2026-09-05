package geopoliticrivalry

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
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

	got, err := store.Get(context.Background(), created.ID)
	if err != nil || !reflect.DeepEqual(got, created) {
		t.Fatalf("Get() = %#v, %v; want %#v", got, err, created)
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

func validInput(name, category, domainID string) CreateInput {
	return CreateInput{
		Name: name, Category: category, GeopoliticDomainID: domainID,
		CoreProposition:  "每条故事线只表达一个核心命题",
		CoreActors:       "核心参与方",
		MainTransmission: "直接影响→对中国经济的主要传导",
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
