package stock

import (
	"context"
	"errors"
	"strings"
	"testing"

	biz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/stock"
	fixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

func TestClassificationPublicationAtomicReplayAndRelations(t *testing.T) {
	db := fixture.OpenIsolated(t, "stock_classification", "../../../migrations", 0)
	ctx := context.Background()
	s, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	input := `{"meta":{"record_count":1,"universe":1},"stocks":[{"code":"000001","exchange":"SZ","industry_l1":"金融","industry_l2":"银行","market_concepts":[{"name":"市场主题","code":"pt123"}],"industry_chain":["金融科技"]}]}`
	if _, err = biz.PublishClassifications(ctx, s, strings.NewReader(input), true); err == nil {
		t.Fatal("missing stock accepted")
	}
	var count int
	if err = db.QueryRow(`SELECT count(*) FROM s_industry`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("partial catalog: %d %v", count, err)
	}
	items, err := biz.DecodeCatalog(strings.NewReader(`{"meta":{"as_of":"2026-09-22","total":1,"by_exchange":{"SZ":1},"by_board":{"主板":1}},"stocks":[{"code":"000001.SZ","name":"平安银行","exchange":"SZ","board":"主板"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Upsert(ctx, items); err != nil {
		t.Fatal(err)
	}
	result, err := biz.PublishClassifications(ctx, s, strings.NewReader(input), false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Stocks != 1 || result.Industries != 2 || result.ConceptLinks != 1 || result.ChainLinks != 1 {
		t.Fatal(result)
	}
	_ = db.QueryRow(`SELECT count(*) FROM s_industry`).Scan(&count)
	if count != 0 {
		t.Fatal("dry run wrote catalog")
	}
	if _, err = biz.PublishClassifications(ctx, s, strings.NewReader(input), true); err != nil {
		t.Fatal(err)
	}
	fingerprint := func() string {
		var value string
		err := db.QueryRow(`SELECT md5(jsonb_build_object('industry',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM s_industry t),'concept',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM s_concept t),'chain',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM s_industry_chain t),'il',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM stock_s_industry_links t),'cl',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM stock_s_concept_links t),'chl',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM stock_s_industry_chain_links t),'stock',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM stock t))::text)`).Scan(&value)
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	before := fingerprint()
	if _, err = biz.PublishClassifications(ctx, s, strings.NewReader(input), true); err != nil {
		t.Fatal(err)
	}
	if fingerprint() != before {
		t.Fatal("replay changed rows or timestamps")
	}
	if err = db.QueryRow(`SELECT count(*) FROM stock_s_industry_links`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("industry links %d %v", count, err)
	}
	if err = db.QueryRow(`SELECT count(*) FROM s_industry child JOIN s_industry parent ON parent.id=child.parent_industry_id WHERE child.name='银行' AND parent.name='金融' AND child.hierarchy_path_codes=parent.hierarchy_path_codes||child.industry_code`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("hierarchy %d %v", count, err)
	}
	if err = db.QueryRow(`SELECT count(*) FROM s_industry_chain WHERE scope IS NULL AND target_output IS NULL AND review_status='candidate'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("missing values fabricated: %d %v", count, err)
	}
	if _, err = db.Exec(`DELETE FROM s_concept`); err == nil {
		t.Fatal("referenced concept deleted")
	}
	// A malformed final row must not partially change existing data.
	bad := strings.Replace(input, `"universe":1`, `"universe":2`, 1)
	if _, err = biz.PublishClassifications(ctx, s, strings.NewReader(bad), true); err == nil {
		t.Fatal("incomplete input accepted")
	}
	if fingerprint() != before {
		t.Fatal("invalid input changed data")
	}
	// Force a late failure after all master rows and earlier links were written.
	if _, err = db.Exec(`CREATE FUNCTION reject_classification_link() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'test failure'; END $$; CREATE TRIGGER reject_classification_link BEFORE INSERT ON stock_s_industry_chain_links FOR EACH ROW EXECUTE FUNCTION reject_classification_link()`); err != nil {
		t.Fatal(err)
	}
	changed := strings.ReplaceAll(input, "金融科技", "新产业链")
	changed = strings.ReplaceAll(changed, "市场主题", "新概念")
	if _, err = biz.PublishClassifications(ctx, s, strings.NewReader(changed), true); err == nil {
		t.Fatal("late failure accepted")
	}
	if fingerprint() != before {
		t.Fatal("transaction failure leaked partial writes")
	}
	if _, err = db.Exec(`DROP TRIGGER reject_classification_link ON stock_s_industry_chain_links`); err != nil {
		t.Fatal(err)
	}
	empty := strings.Replace(input, `[{"name":"市场主题","code":"pt123"}]`, `[]`, 1)
	if _, err = biz.PublishClassifications(ctx, s, strings.NewReader(empty), true); err != nil {
		t.Fatal(err)
	}
	_ = db.QueryRow(`SELECT count(*) FROM stock_s_concept_links`).Scan(&count)
	if count != 0 {
		t.Fatal("stock links not synchronized")
	}
	_ = db.QueryRow(`SELECT count(*) FROM s_concept`).Scan(&count)
	if count != 1 {
		t.Fatal("unreferenced master deleted")
	}
}

func TestDailyQuotesAtomicPrecisionReplayAndConstraints(t *testing.T) {
	db := fixture.OpenIsolated(t, "daily_quote", "../../../migrations", 0)
	ctx := context.Background()
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	input := `{"meta":{"period":"day","universe":2,"record_count":2},"stocks":[{"code":"601091","exchange":"SH","name":"样本","kline":[{"date":"2026-09-16","open":0,"high":0,"low":0,"close":4.39,"volume":0,"turnover":0,"change_pct":0},{"date":"2026-09-17","open":20.0001,"high":21,"low":19.1234,"close":20.8,"volume":287081713,"turnover":95.17,"change_pct":373.8041002278}]},{"code":"000016","exchange":"SZ","name":"无行情","kline":[]}]}`
	decode := func(s string) biz.DailyQuoteBatch {
		t.Helper()
		b, e := biz.DecodeDailyQuotes(strings.NewReader(s))
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	batch := decode(input)
	if _, err = biz.PublishDailyQuotes(ctx, store, batch, true); !errors.Is(err, biz.ErrInvalid) {
		t.Fatalf("missing stock: %v", err)
	}
	catalog := `{"meta":{"as_of":"2026-09-30","total":2,"by_exchange":{"SH":1,"SZ":1},"by_board":{"主板":2}},"stocks":[{"code":"601091.SH","name":"样本","exchange":"SH","board":"主板"},{"code":"000016.SZ","name":"无行情","exchange":"SZ","board":"主板"}]}`
	stocks, err := biz.DecodeCatalog(strings.NewReader(catalog))
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Upsert(ctx, stocks); err != nil {
		t.Fatal(err)
	}
	fingerprint := func(table string) string {
		t.Helper()
		var result string
		if e := db.QueryRow(`SELECT md5(coalesce(jsonb_agg(to_jsonb(t) ORDER BY id)::text,'')) FROM ` + table + ` t`).Scan(&result); e != nil {
			t.Fatal(e)
		}
		return result
	}
	originalStock := fingerprint("stock")
	result, err := biz.PublishDailyQuotes(ctx, store, batch, false)
	if err != nil || result.New != 2 || result.Applied {
		t.Fatalf("dry run: %+v %v", result, err)
	}
	var count int
	if err = db.QueryRow(`SELECT count(*) FROM stock_daily_quote`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("dry run wrote: %d %v", count, err)
	}
	result, err = biz.PublishDailyQuotes(ctx, store, batch, true)
	if err != nil || result.New != 2 || result.Placeholders != 1 || result.Observed != 1 || result.StocksWithQuotes != 1 || len(result.Normalizations) != 1 {
		t.Fatalf("publish: %+v %v", result, err)
	}
	var change, price string
	if err = db.QueryRow(`SELECT change_pct::text,open_price::text FROM stock_daily_quote WHERE trade_date='2026-09-17'`).Scan(&change, &price); err != nil || change != "373.8041002278" || price != "20.000100" {
		t.Fatalf("precision: %s %s %v", change, price, err)
	}
	quoteFingerprint := fingerprint("stock_daily_quote")
	result, err = biz.PublishDailyQuotes(ctx, store, batch, true)
	if err != nil || result.New != 0 || result.Unchanged != 2 || fingerprint("stock_daily_quote") != quoteFingerprint {
		t.Fatalf("replay: %+v %v", result, err)
	}
	conflict := decode(strings.Replace(input, `"close":20.8`, `"close":20.9`, 1))
	if _, err = biz.PublishDailyQuotes(ctx, store, conflict, true); !errors.Is(err, biz.ErrConflict) {
		t.Fatalf("conflict: %v", err)
	}
	if fingerprint("stock_daily_quote") != quoteFingerprint {
		t.Fatal("conflict changed records")
	}
	// A real database failure must roll back all rows in the attempted batch.
	if _, err = db.Exec(`CREATE FUNCTION reject_quote() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.trade_date='2026-09-19' THEN RAISE EXCEPTION 'test'; END IF; RETURN NEW; END $$; CREATE TRIGGER reject_quote BEFORE INSERT ON stock_daily_quote FOR EACH ROW EXECUTE FUNCTION reject_quote()`); err != nil {
		t.Fatal(err)
	}
	newer := decode(strings.ReplaceAll(strings.ReplaceAll(input, "2026-09-16", "2026-09-18"), "2026-09-17", "2026-09-19"))
	if _, err = biz.PublishDailyQuotes(ctx, store, newer, true); !errors.Is(err, biz.ErrPersistence) {
		t.Fatalf("trigger failure: %v", err)
	}
	if fingerprint("stock_daily_quote") != quoteFingerprint {
		t.Fatal("failed batch partially persisted")
	}
	if _, err = db.Exec(`DROP TRIGGER reject_quote ON stock_daily_quote`); err != nil {
		t.Fatal(err)
	}
	// Concurrent identical publishers must yield one insert and one replay.
	errs := make(chan error, 2)
	results := make(chan biz.DailyQuoteResult, 2)
	for i := 0; i < 2; i++ {
		go func() { r, e := biz.PublishDailyQuotes(ctx, store, newer, true); results <- r; errs <- e }()
	}
	totalNew, totalUnchanged := 0, 0
	for i := 0; i < 2; i++ {
		if e := <-errs; e != nil {
			t.Fatal(e)
		}
		r := <-results
		totalNew += r.New
		totalUnchanged += r.Unchanged
	}
	if totalNew != 2 || totalUnchanged != 2 {
		t.Fatalf("concurrent publication: %d %d", totalNew, totalUnchanged)
	}
	for _, sql := range []string{
		`UPDATE stock_daily_quote SET high_price=1 WHERE record_status='observed'`,
		`UPDATE stock_daily_quote SET low_price=NULL WHERE record_status='observed'`,
		`UPDATE stock_daily_quote SET close_price='NaN'::numeric`,
		`UPDATE stock_daily_quote SET volume_lots=-1`,
		`UPDATE stock_daily_quote SET change_pct='NaN'::numeric`,
		`UPDATE stock_daily_quote SET open_price=1 WHERE record_status='placeholder'`,
		`UPDATE stock_daily_quote SET stock_id='STK00000000-0000-4000-8000-000000000000'`,
		`UPDATE stock_daily_quote SET trade_date='2026-09-17'`,
		`DELETE FROM stock WHERE code='601091'`,
	} {
		if _, e := db.Exec(sql); e == nil {
			t.Fatalf("constraint accepted: %s", sql)
		}
	}
	if fingerprint("stock") != originalStock {
		t.Fatal("stock was modified")
	}
}
