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

func TestProfilePackagePreservesSourceSemanticsAndRejectsMissingFields(t *testing.T) {
	raw := []byte(`{"meta":{"kind":"SAMPLE","schema_version":"v4","sample_size":1,"as_of":"2026-09-22"},"stocks":[{"code":"000001","name":"平安银行","exchange":"SZ","board":"主板","full_name":"平安银行股份有限公司","former_name":null,"list_date":"1991-04-03","established":null,"industry_l1":"金融","industry_l2":"银行","main_business":[{"name":"抵销","pct":-3.2}],"main_product_type":null,"index_core":[],"concepts":["跨境支付","银"]}]}`)
	batch, err := DecodeProfiles(bytes.NewReader(raw))
	if err != nil || len(batch.Items) != 1 {
		t.Fatal(batch, err)
	}
	p := batch.Items[0]
	if p.IndexCore == nil || p.MainProductType != nil || p.MainBusiness[0].Pct == nil || *p.MainBusiness[0].Pct != -3.2 {
		t.Fatal("source null/array/negative pct changed")
	}
	var wire map[string]any
	if json.Unmarshal(raw, &wire) != nil {
		t.Fatal("fixture")
	}
	item := wire["stocks"].([]any)[0].(map[string]any)
	delete(item, "full_name")
	bad, _ := json.Marshal(wire)
	if _, err = DecodeProfiles(bytes.NewReader(bad)); err == nil {
		t.Fatal("missing field accepted as null")
	}
}
