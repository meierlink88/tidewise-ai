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
