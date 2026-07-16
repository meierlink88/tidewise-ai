package dbmigration

import (
	"strings"
	"testing"
)

func TestEventImportMigrationFreezesSourceReceiptAndTagSeed(t *testing.T) {
	sql := strings.ToLower(readMigration(t, "000020_add_event_import_receipts_and_tag_seed.sql"))
	for _, required := range []string{
		"add column if not exists content_level varchar(32)",
		"create table if not exists event_import_receipts",
		"idempotency_key text not null unique",
		"payload_hash char(64) not null",
		"event_id uuid not null references events(id)",
		"chk_event_import_receipts_payload_hash",
		"cd209afe-2ea9-54b8-bdd7-db64eebf0d71",
		"manifest_identity",
		"insert into event_tag_defs",
		"on conflict (tag_kind, code) do update",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration missing %q", required)
		}
	}
	for _, code := range []string{
		"geopolitics", "macroeconomy", "monetary_policy", "fiscal_trade", "usd_fx", "commodities", "market_indices", "executive_commentary", "capital_markets", "technology_industry",
		"macro_economic_index", "inflation_price_index", "interest_credit_index", "fx_index", "equity_broad_index", "industry_theme_index", "commodity_index", "market_sentiment_index", "stock_trading_data", "futures_contract", "fund_etf_index", "options_derivatives",
	} {
		if !strings.Contains(sql, "'"+code+"'") {
			t.Fatalf("migration missing frozen tag code %q", code)
		}
	}
	if strings.Contains(sql, "drop table") || strings.Contains(sql, "truncate") {
		t.Fatal("event import migration must not destructively reset data")
	}
}
