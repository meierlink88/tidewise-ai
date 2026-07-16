package eventimport

import "fmt"

type FrozenTag struct {
	ID           string `json:"id"`
	Kind         string `json:"tag_kind"`
	Code         string `json:"code"`
	Name         string `json:"name"`
	DisplayOrder int    `json:"display_order"`
}

// FrozenTags is the runtime authority for the 22 reviewed-outbox tag identities.
// The migration contract test asserts that 000020 contains the same tuples.
var FrozenTags = []FrozenTag{
	{ID: "b0fe1994-0db2-526c-a57f-97fa73c1b595", Kind: "news_category", Code: "geopolitics", Name: "地缘政治", DisplayOrder: 1},
	{ID: "b1a5438f-6e81-55e7-8ecb-33230b9ae965", Kind: "news_category", Code: "macroeconomy", Name: "宏观经济", DisplayOrder: 2},
	{ID: "19fb07c0-aed3-5a1a-99b4-bba004cf2d00", Kind: "news_category", Code: "monetary_policy", Name: "货币政策", DisplayOrder: 3},
	{ID: "80f6cb51-38ed-5fcc-8037-3aff25d1b767", Kind: "news_category", Code: "fiscal_trade", Name: "财政贸易", DisplayOrder: 4},
	{ID: "06d1e3f4-ba81-5903-80d0-daabb27421af", Kind: "news_category", Code: "usd_fx", Name: "美元汇率", DisplayOrder: 5},
	{ID: "80155a2e-33a9-545a-b57e-7bb253af699d", Kind: "news_category", Code: "commodities", Name: "大宗商品", DisplayOrder: 6},
	{ID: "2b775f7a-24de-5b44-9fef-dd18f7480148", Kind: "news_category", Code: "market_indices", Name: "指数行情", DisplayOrder: 7},
	{ID: "79b73443-5cc4-589b-9dd0-720d2af61e14", Kind: "news_category", Code: "executive_commentary", Name: "高层评论", DisplayOrder: 8},
	{ID: "7947aa41-be9c-52ea-816e-8513b6c18d7d", Kind: "news_category", Code: "capital_markets", Name: "资本市场", DisplayOrder: 9},
	{ID: "22a5afc5-20ed-55ce-bf77-54c26bbcc6ea", Kind: "news_category", Code: "technology_industry", Name: "科技产业", DisplayOrder: 10},
	{ID: "173cabde-c2bf-5cdc-a026-08cd52a953f0", Kind: "index_category", Code: "macro_economic_index", Name: "宏观经济指数", DisplayOrder: 1},
	{ID: "71e1deff-56b8-5f70-88ae-fcd4e267c429", Kind: "index_category", Code: "inflation_price_index", Name: "通胀物价指数", DisplayOrder: 2},
	{ID: "d9a25979-00e6-5fe4-8807-4ac455d275cd", Kind: "index_category", Code: "interest_credit_index", Name: "利率与信用指数", DisplayOrder: 3},
	{ID: "896f457d-3c40-5bad-bb91-3c7f196287c5", Kind: "index_category", Code: "fx_index", Name: "外汇汇率指数", DisplayOrder: 4},
	{ID: "87de7402-7632-5a61-8f16-1432f9112c7e", Kind: "index_category", Code: "equity_broad_index", Name: "股票宽基指数", DisplayOrder: 5},
	{ID: "22bf6fe5-7b11-5e80-abfa-430713657426", Kind: "index_category", Code: "industry_theme_index", Name: "行业主题指数", DisplayOrder: 6},
	{ID: "ba56c6f1-2dfb-5f4c-a769-b95570e0a830", Kind: "index_category", Code: "commodity_index", Name: "大宗商品指数", DisplayOrder: 7},
	{ID: "d4616900-4234-578b-9f35-2364c1009634", Kind: "index_category", Code: "market_sentiment_index", Name: "市场情绪指数", DisplayOrder: 8},
	{ID: "b67b9650-7460-5708-9c10-089d566682b0", Kind: "index_category", Code: "stock_trading_data", Name: "个股与成交数据", DisplayOrder: 9},
	{ID: "4f9ffa47-39c7-5a86-90a4-5ad06d91de4b", Kind: "index_category", Code: "futures_contract", Name: "期货合约品种", DisplayOrder: 10},
	{ID: "e95a831e-f852-5838-a739-dbc59726a059", Kind: "index_category", Code: "fund_etf_index", Name: "基金与 ETF 指数", DisplayOrder: 11},
	{ID: "6b2cf910-6aa3-5f8d-8016-8e9c0c4a2b09", Kind: "index_category", Code: "options_derivatives", Name: "期权与衍生品", DisplayOrder: 12},
}

func LookupFrozenTag(kind, id, code string) (FrozenTag, error) {
	for _, tag := range FrozenTags {
		if tag.Kind == kind && tag.Code == code && tag.ID == id {
			return tag, nil
		}
	}
	return FrozenTag{}, fmt.Errorf("unknown event tag identity %q/%q/%q", id, kind, code)
}
