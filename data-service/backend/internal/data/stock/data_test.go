package stock

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	biz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/stock"
	fixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

func TestCatalogImportReplayConflictAndSearch(t *testing.T) {
	db := fixture.OpenIsolated(t, "stock_catalog", "../../../migrations", 0)
	ctx := context.Background()
	s, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open("../../../../initdata/stocks-a-share-20260921.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	items, err := biz.DecodeCatalog(f)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Upsert(ctx, items); err != nil {
		t.Fatal(err)
	}
	var before string
	if err = db.QueryRow(`SELECT md5(string_agg(row_to_json(s)::text,',' ORDER BY id)) FROM stock s`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if err = s.Upsert(ctx, items); err != nil {
		t.Fatal(err)
	}
	var after string
	_ = db.QueryRow(`SELECT md5(string_agg(row_to_json(s)::text,',' ORDER BY id)) FROM stock s`).Scan(&after)
	if before != after {
		t.Fatal("replay changed rows")
	}
	for _, tc := range []struct {
		query string
		code  string
		count int
	}{{"000001", "000001", 1}, {"600519.SH", "600519", 1}, {"贵州茅台", "600519", 1}, {"%", "", 0}, {"_", "", 0}, {"不存在公司", "", 0}} {
		page, err := s.Search(ctx, biz.Query{Text: tc.query, Limit: 20})
		if err != nil || len(page.Items) != tc.count {
			t.Fatalf("%s: %#v %v", tc.query, page, err)
		}
		if tc.count > 0 && page.Items[0].Code != tc.code {
			t.Fatal(page)
		}
	}
	page, err := s.Search(ctx, biz.Query{Text: "0", Limit: 1})
	if err != nil || !page.HasMore {
		t.Fatalf("pagination %v %v", page, err)
	}
	changed := append([]biz.Stock(nil), items...)
	changed[0].Name = "改名"
	changed[0].AsOf = changed[0].AsOf.Add(24 * time.Hour)
	changed[len(changed)-1].AsOf = changed[len(changed)-1].AsOf.Add(-24 * time.Hour)
	if err = s.Upsert(ctx, changed); !errors.Is(err, biz.ErrConflict) {
		t.Fatalf("stale import %v", err)
	}
	_ = db.QueryRow(`SELECT md5(string_agg(row_to_json(s)::text,',' ORDER BY id)) FROM stock s`).Scan(&after)
	if before != after {
		t.Fatal("conflicting import partially committed")
	}
	rows, err := db.Query(`SELECT id,code,name,exchange,board,as_of FROM stock ORDER BY exchange,code`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	i := 0
	for rows.Next() {
		var got biz.Stock
		if err = rows.Scan(&got.ID, &got.Code, &got.Name, &got.Exchange, &got.Board, &got.AsOf); err != nil {
			t.Fatal(err)
		}
		want := items[i]
		if got.ID != want.ID || got.Code != want.Code || got.Name != want.Name || got.Exchange != want.Exchange || got.Board != want.Board || !got.AsOf.Equal(want.AsOf) {
			t.Fatalf("readback mismatch %d", i)
		}
		i++
	}
	if rows.Err() != nil || i != 5565 {
		t.Fatalf("readback count %d err %v", i, rows.Err())
	}
}

func TestBusinessFieldsPreserveUnknownEmptyAndCatalogCompatibility(t *testing.T) {
	db := fixture.OpenIsolated(t, "stock_business", "../../../migrations", 0)
	ctx := context.Background()
	s, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	item := biz.Stock{ID: "STKf4a8eb61-c352-5980-91b1-9da6eba8f8af", Code: "000001", Name: "平安银行", Exchange: "SZ", Board: "主板", AsOf: time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)}
	if err := s.Upsert(ctx, []biz.Stock{item}); err != nil {
		t.Fatal(err)
	}
	var unknown bool
	if err := db.QueryRow(`SELECT full_name IS NULL AND former_name IS NULL AND list_date IS NULL AND established IS NULL AND industry_l1 IS NULL AND industry_l2 IS NULL AND main_business IS NULL AND main_product_type IS NULL AND index_core IS NULL AND concepts IS NULL FROM stock WHERE id=$1`, item.ID).Scan(&unknown); err != nil || !unknown {
		t.Fatalf("unknown fields: %v, %v", unknown, err)
	}
	if _, err := db.Exec(`UPDATE stock SET main_business='[]', main_product_type='{}', index_core='{}', concepts='{}' WHERE id=$1`, item.ID); err != nil {
		t.Fatal(err)
	}
	var empty bool
	if err := db.QueryRow(`SELECT main_business='[]'::jsonb AND cardinality(main_product_type)=0 AND cardinality(index_core)=0 AND cardinality(concepts)=0 FROM stock WHERE id=$1`, item.ID).Scan(&empty); err != nil || !empty {
		t.Fatalf("known empty: %v, %v", empty, err)
	}
	// Negative eliminations and non-100 totals are source facts, not invalid weights.
	if _, err := db.Exec(`UPDATE stock SET full_name='测试银行股份有限公司', former_name='甲公司,乙公司', list_date='1991-04-03', established='1987-12-22', industry_l1='金融', industry_l2='银行', main_business='[{"name":"利息收入:贷款","pct":120.25},{"name":"抵销","pct":-14.17}]', main_product_type=ARRAY['贷款','存款'], index_core=ARRAY['沪深300'], concepts=ARRAY['跨境支付','银'] WHERE id=$1`, item.ID); err != nil {
		t.Fatal(err)
	}
	profileSQL := `SELECT jsonb_build_array(full_name,former_name,list_date,established,industry_l1,industry_l2,main_business,main_product_type,index_core,concepts)::text FROM stock WHERE id=$1`
	var before, after string
	if err := db.QueryRow(profileSQL, item.ID).Scan(&before); err != nil {
		t.Fatal(err)
	}
	item.AsOf = item.AsOf.Add(24 * time.Hour)
	if err := s.Upsert(ctx, []biz.Stock{item}); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(profileSQL, item.ID).Scan(&after); err != nil || after != before {
		t.Fatalf("catalog overwrote profile: %v", err)
	}
	page, err := s.Search(ctx, biz.Query{Text: "000001", Limit: 20})
	if err != nil || len(page.Items) != 1 || page.Items[0].ID != item.ID {
		t.Fatalf("catalog search: %v %v", page, err)
	}
	for _, invalid := range []string{`null`, `{}`, `[null]`, `[1]`, `[{}]`, `[{"name":"收入"}]`, `[{"pct":1}]`, `[{"name":null,"pct":1}]`, `[{"name":" ","pct":1}]`, `[{"name":[],"pct":1}]`, `[{"name":"收入","pct":"1"}]`, `[{"name":"收入","pct":null}]`, `[{"name":"收入","pct":[1]}]`} {
		if _, err := db.Exec(`UPDATE stock SET main_business=$1::jsonb WHERE id=$2`, invalid, item.ID); err == nil {
			t.Errorf("accepted malformed breakdown %s", invalid)
		}
	}
	for _, column := range []string{"main_product_type", "index_core", "concepts"} {
		for _, invalid := range []string{`ARRAY['x',NULL]`, `ARRAY[['x'],['y']]`} {
			if _, err := db.Exec(`UPDATE stock SET `+column+`=`+invalid+` WHERE id=$1`, item.ID); err == nil {
				t.Errorf("accepted invalid %s: %s", column, invalid)
			}
		}
	}
}
