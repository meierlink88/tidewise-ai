package stock

import (
	"context"
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
