package macroeconomic

import (
	"context"
	"errors"
	"reflect"
	"testing"

	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
	domaindata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/macroeconomicdomain"
)

func TestStoreStorylineContracts(t *testing.T) {
	db := openMacroeconomicCatalogTestDatabase(t, "tw_macro_store")
	ctx := context.Background()
	ds, err := domaindata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	d, err := ds.Create(ctx, domaindata.CreateInput{Code: "MONETARY", Name: "货币政策线", Description: "利率与准备金政策", Tactics: []domaindata.Tactic{{Name: "降息", Description: "下调政策利率"}}})
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	input := CreateInput{Name: "中国政策利率调整", MacroEconomicDomainID: d.ID, CoreProposition: "政策利率通过融资成本影响投资。", CandidateAssets: []string{"国债ETF", "银行板块"}}
	created, err := s.Create(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if !coreid.Is(created.ID, coreid.MacroEconomic) || created.Name != input.Name || created.CoreProposition != input.CoreProposition || !reflect.DeepEqual(created.CandidateAssets, input.CandidateAssets) {
		t.Fatalf("created: %#v", created)
	}
	got, err := s.Get(ctx, created.ID)
	if err != nil || !reflect.DeepEqual(got, created) {
		t.Fatalf("get: %#v %v", got, err)
	}
	items, err := s.List(ctx, Filter{MacroEconomicDomainID: &d.ID})
	if err != nil || len(items) != 1 {
		t.Fatalf("list: %#v %v", items, err)
	}
	if _, err := s.Create(ctx, input); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate: %v", err)
	}
	missing, _ := coreid.New(coreid.MacroEconomicDomain)
	invalid := input
	invalid.Name = "未知领域"
	invalid.MacroEconomicDomainID = missing
	if _, err := s.Create(ctx, invalid); !errors.Is(err, ErrInvalidMacroEconomic) {
		t.Fatalf("missing domain: %v", err)
	}
	for _, assets := range [][]string{nil, {}, {"黄金", "黄金"}, {" 黄金"}} {
		invalid = input
		invalid.CandidateAssets = assets
		if _, err := s.Create(ctx, invalid); !errors.Is(err, ErrInvalidMacroEconomic) {
			t.Fatalf("assets: %v", err)
		}
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM macro_economics_domain WHERE id=$1", d.ID); err == nil {
		t.Fatal("accepted referenced domain deletion")
	}
	updated, err := s.Update(ctx, UpdateInput{ID: created.ID, Name: created.Name, MacroEconomicDomainID: d.ID, CoreProposition: "利率变化影响融资成本与资产估值。", CandidateAssets: []string{"国债ETF"}})
	if err != nil || updated.ID != created.ID || !updated.CreatedAt.Equal(created.CreatedAt) || updated.CoreProposition == created.CoreProposition || len(updated.CandidateAssets) != 1 {
		t.Fatalf("update: %#v %v", updated, err)
	}
	for _, payload := range []string{`{}`, `null`, `[]`, `["黄金","黄金"]`, `[1]`} {
		if _, err := db.ExecContext(ctx, "UPDATE macro_economics SET candidate_assets=$1::jsonb WHERE id=$2", payload, created.ID); err == nil {
			t.Fatalf("database accepted %s", payload)
		}
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := s.Get(cancelled, created.ID); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
	// Corrupt storage after deliberately removing the constraint to prove fail-closed reads.
	if _, err := db.ExecContext(ctx, "ALTER TABLE macro_economics DROP CONSTRAINT macro_economics_candidate_assets_check"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "UPDATE macro_economics SET candidate_assets='[]'::jsonb WHERE id=$1", created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(ctx, created.ID); !errors.Is(err, ErrPersistence) {
		t.Fatalf("corrupt storage: %v", err)
	}
}
