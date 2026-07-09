package sourcecatalog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVibeResearchRSSSeedMatchesResearchedSourceCounts(t *testing.T) {
	manifest, err := LoadFile(filepath.Join("..", "..", "..", "..", "data", "source_catalogs", "vibe_research_rss.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	if got, want := len(manifest.Sources), 108; got != want {
		t.Fatalf("sources = %d, want %d", got, want)
	}

	uniqueURLs := map[string]int{}
	for _, source := range manifest.Sources {
		if source.OriginSystem != "Vibe-Research" {
			t.Fatalf("OriginSystem = %q, want Vibe-Research", source.OriginSystem)
		}
		if source.ConnectorKey != "rss_feed" || source.ParserKey != "rss_item" {
			t.Fatalf("source %q connector/parser = %s/%s, want rss_feed/rss_item", source.ID, source.ConnectorKey, source.ParserKey)
		}
		if source.SourceConfig["kind"] != "rss_feed" {
			t.Fatalf("source %q SourceConfig[kind] = %v, want rss_feed", source.ID, source.SourceConfig["kind"])
		}
		uniqueURLs[source.SourceURL]++
	}

	if got, want := len(uniqueURLs), 106; got != want {
		t.Fatalf("unique URLs = %d, want %d", got, want)
	}
	assertDuplicateURL(t, uniqueURLs, "https://www.engadget.com/rss.xml", 2)
	assertDuplicateURL(t, uniqueURLs, "https://sspai.com/feed", 2)
}

func TestVibeTradingNonSDKSeedMatchesRegistryScope(t *testing.T) {
	manifest, err := LoadFile(filepath.Join("..", "..", "..", "..", "data", "source_catalogs", "vibe_trading_non_sdk.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	if got, want := len(manifest.Sources), 13; got != want {
		t.Fatalf("sources = %d, want %d", got, want)
	}

	seen := map[string]bool{}
	for _, source := range manifest.Sources {
		if source.OriginSystem != "Vibe-Trading" {
			t.Fatalf("OriginSystem = %q, want Vibe-Trading", source.OriginSystem)
		}
		if source.ConnectorKey != "market_provider" || source.ParserKey != "provider_metadata" {
			t.Fatalf("source %q connector/parser = %s/%s, want market_provider/provider_metadata", source.ID, source.ConnectorKey, source.ParserKey)
		}
		loader, ok := source.SourceConfig["loader_source"].(string)
		if !ok || loader == "" {
			t.Fatalf("source %q loader_source is missing", source.ID)
		}
		seen[loader] = true
	}

	for _, included := range []string{"okx", "yfinance", "tencent", "ccxt", "eastmoney", "sina", "stooq", "yahoo", "finnhub", "alphavantage", "tiingo", "fmp", "local"} {
		if !seen[included] {
			t.Fatalf("expected included loader source %q", included)
		}
	}
	for _, excluded := range []string{"auto", "tushare", "akshare", "baostock", "futu", "mootdx"} {
		if seen[excluded] {
			t.Fatalf("loader source %q must be excluded from non-SDK seed", excluded)
		}
	}
}

func TestStockNonSDKSeedMatchesResearchedSourceScope(t *testing.T) {
	manifest, err := LoadFile(filepath.Join("..", "..", "..", "..", "data", "source_catalogs", "stock_non_sdk.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	if got, want := len(manifest.Sources), 56; got != want {
		t.Fatalf("sources = %d, want %d", got, want)
	}

	byGroup := map[string]int{}
	for _, source := range manifest.Sources {
		if source.OriginSystem != "Stock" {
			t.Fatalf("OriginSystem = %q, want Stock", source.OriginSystem)
		}
		if source.ConnectorKey == "sdk_stub" {
			t.Fatalf("source %q must not include SDK connector in non-SDK seed", source.ID)
		}
		byGroup[source.SourceGroup]++
	}

	assertGroupCount(t, byGroup, "stock_news", 6)
	assertGroupCount(t, byGroup, "market_data", 16)
	assertGroupCount(t, byGroup, "sector_data", 30)
	assertGroupCount(t, byGroup, "local_backfill", 4)
}

func TestContentEventSourceGroupCoversExpectedConnectorKinds(t *testing.T) {
	manifest, err := LoadFile(filepath.Join("..", "..", "..", "..", "data", "source_catalogs", "content_event_sources.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	if got, want := len(manifest.Sources), 4; got != want {
		t.Fatalf("sources = %d, want %d", got, want)
	}
	assertConnectorPresent(t, manifest, "rss_feed")
	assertConnectorPresent(t, manifest, "web_fetch")
	assertConnectorPresent(t, manifest, "rsshub_feed")
	assertConnectorPresent(t, manifest, "local_backfill")
}

func TestAIWebResearchSourceGroupCoversInitialSearchProviders(t *testing.T) {
	manifest, err := LoadFile(filepath.Join("..", "..", "..", "..", "data", "source_catalogs", "ai_web_research_sources.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	if got, want := len(manifest.Sources), 2; got != want {
		t.Fatalf("sources = %d, want %d", got, want)
	}

	for _, source := range manifest.Sources {
		if source.ConnectorKey != "llm_web_research" || source.ParserKey != "llm_research_items" {
			t.Fatalf("source %q connector/parser = %s/%s, want llm_web_research/llm_research_items", source.ID, source.ConnectorKey, source.ParserKey)
		}
		if source.SourceConfig["kind"] != "llm_web_research" {
			t.Fatalf("source %q kind = %v, want llm_web_research", source.ID, source.SourceConfig["kind"])
		}
		if source.SourceConfig["search_plan_mode"] != "llm_query_plan" {
			t.Fatalf("source %q search_plan_mode = %v, want llm_query_plan", source.ID, source.SourceConfig["search_plan_mode"])
		}
		credentialRefs, ok := source.SourceConfig["credential_refs"].(map[string]any)
		if !ok || credentialRefs["planner"] == "" {
			t.Fatalf("source %q credential_refs.planner is missing", source.ID)
		}
		plan, ok := source.SourceConfig["web_search_plan"].(map[string]any)
		if !ok {
			t.Fatalf("source %q web_search_plan is missing", source.ID)
		}
		tools, ok := plan["tools"].([]any)
		if !ok || len(tools) != 2 {
			t.Fatalf("source %q tools = %v, want two tools", source.ID, plan["tools"])
		}
		seen := map[string]bool{}
		for _, tool := range tools {
			toolConfig, ok := tool.(map[string]any)
			if !ok {
				t.Fatalf("source %q tool = %v, want object", source.ID, tool)
			}
			provider, _ := toolConfig["provider"].(string)
			seen[provider] = true
		}
		for _, provider := range []string{"tavily", "bocha_web_search"} {
			if !seen[provider] {
				t.Fatalf("source %q expected web search provider %q", source.ID, provider)
			}
		}
		promptRef, ok := source.SourceConfig["prompt_ref"].(string)
		if !ok || promptRef == "" {
			t.Fatalf("source %q prompt_ref is missing", source.ID)
		}
		if promptRef != "ingestion/ai_web_research/search-plan.v1.md" {
			t.Fatalf("source %q prompt_ref = %q, want search plan prompt", source.ID, promptRef)
		}
		promptPath := filepath.Join("..", "..", "..", "..", "data", "prompts", filepath.FromSlash(promptRef))
		if _, err := os.Stat(promptPath); err != nil {
			t.Fatalf("source %q prompt file %q error = %v", source.ID, promptRef, err)
		}
	}
}

func TestMarketSourceGroupCoversResearchedHTTPProviders(t *testing.T) {
	manifest, err := LoadFile(filepath.Join("..", "..", "..", "..", "data", "source_catalogs", "market_sources.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	seen := map[string]bool{}
	for _, source := range manifest.Sources {
		if source.SourceGroup != "market_data" {
			t.Fatalf("source %q group = %q, want market_data", source.ID, source.SourceGroup)
		}
		seen[source.ProviderKey] = true
	}
	for _, provider := range []string{"eastmoney", "sina", "tencent", "yahoo", "stooq", "finnhub", "fmp", "tiingo", "alphavantage", "okx", "ccxt"} {
		if !seen[provider] {
			t.Fatalf("expected market provider %q", provider)
		}
	}
}

func TestSectorSourceGroupCoversEastmoneyBoardsAndIndexes(t *testing.T) {
	manifest, err := LoadFile(filepath.Join("..", "..", "..", "..", "data", "source_catalogs", "sector_sources.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	seenKinds := map[string]bool{}
	for _, source := range manifest.Sources {
		if source.SourceGroup != "sector_data" {
			t.Fatalf("source %q group = %q, want sector_data", source.ID, source.SourceGroup)
		}
		kind, _ := source.SourceConfig["kind"].(string)
		seenKinds[kind] = true
	}
	for _, kind := range []string{"eastmoney_concept_board_list", "eastmoney_concept_board_kline", "stock_predefined_board_universe", "eastmoney_major_index_metadata"} {
		if !seenKinds[kind] {
			t.Fatalf("expected sector source kind %q", kind)
		}
	}
}

func TestLocalBackfillSourceGroupCoversHistoricalMaterialTypes(t *testing.T) {
	manifest, err := LoadFile(filepath.Join("..", "..", "..", "..", "data", "source_catalogs", "local_backfill_sources.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	seenKinds := map[string]bool{}
	for _, source := range manifest.Sources {
		if source.SourceGroup != "local_backfill" {
			t.Fatalf("source %q group = %q, want local_backfill", source.ID, source.SourceGroup)
		}
		kind, _ := source.SourceConfig["kind"].(string)
		seenKinds[kind] = true
	}
	for _, kind := range []string{"stock_csv", "index_csv", "sector_file", "news_file"} {
		if !seenKinds[kind] {
			t.Fatalf("expected local backfill kind %q", kind)
		}
	}
}

func assertDuplicateURL(t *testing.T, counts map[string]int, url string, want int) {
	t.Helper()

	if got := counts[url]; got != want {
		t.Fatalf("URL %q count = %d, want %d", url, got, want)
	}
}

func assertConnectorPresent(t *testing.T, manifest Manifest, connector string) {
	t.Helper()

	for _, source := range manifest.Sources {
		if source.ConnectorKey == connector {
			return
		}
	}
	t.Fatalf("expected connector %q", connector)
}

func assertGroupCount(t *testing.T, counts map[string]int, group string, want int) {
	t.Helper()

	if got := counts[group]; got != want {
		t.Fatalf("group %q count = %d, want %d", group, got, want)
	}
}
