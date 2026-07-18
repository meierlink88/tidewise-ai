package sourcecatalog

import "path/filepath"

const DefaultSeedDir = "data/source_catalogs"

func DefaultSeedPaths(seedDir string) []string {
	if seedDir == "" {
		seedDir = DefaultSeedDir
	}
	return []string{
		filepath.Join(seedDir, "vibe_research_rss.json"),
		filepath.Join(seedDir, "vibe_trading_non_sdk.json"),
		filepath.Join(seedDir, "stock_non_sdk.json"),
		filepath.Join(seedDir, "content_event_sources.json"),
		filepath.Join(seedDir, "ai_web_research_sources.json"),
		filepath.Join(seedDir, "market_sources.json"),
		filepath.Join(seedDir, "sector_sources.json"),
		filepath.Join(seedDir, "local_backfill_sources.json"),
	}
}
