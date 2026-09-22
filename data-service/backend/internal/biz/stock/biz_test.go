package stock

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestCatalogRejectsCorruptInputAndKeepsLeadingZeros(t *testing.T) {
	raw, err := os.ReadFile("../../../../initdata/stocks-a-share-20260921.json")
	if err != nil {
		t.Fatal(err)
	}
	items, err := DecodeCatalog(bytes.NewReader(raw))
	if err != nil || len(items) != 5565 {
		t.Fatalf("decode=%d %v", len(items), err)
	}
	found := false
	for _, item := range items {
		if item.Code == "000001" && item.Exchange == "SZ" {
			found = true
		}
	}
	if !found {
		t.Fatal("leading zeros lost")
	}
	for _, change := range []func(*Catalog){func(c *Catalog) { c.Stocks[1] = c.Stocks[0] }, func(c *Catalog) { c.Meta.Total-- }, func(c *Catalog) { c.Stocks[0].Exchange = "SH" }, func(c *Catalog) { c.Meta.ByBoard["主板"]-- }} {
		var c Catalog
		if err = json.Unmarshal(raw, &c); err != nil {
			t.Fatal(err)
		}
		change(&c)
		bad, _ := json.Marshal(c)
		if _, err = DecodeCatalog(bytes.NewReader(bad)); err == nil {
			t.Fatal("corrupt catalog accepted")
		}
	}
}
