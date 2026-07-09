package sourcecatalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func TestLoadManifestValidatesAndMapsSources(t *testing.T) {
	path := writeManifest(t, `{
		"sources": [
			{
				"id": "vibe-research:bbc-business",
				"origin_system": "Vibe-Research",
				"stage": "content",
				"ingest_channel": "rss_feed",
				"provider_key": "bbc",
				"connector_key": "rss_feed",
				"parser_key": "rss_item",
				"source_type": "news",
				"source_group": "content_events",
				"source_name": "BBC Business RSS",
				"source_url": "https://feeds.bbci.co.uk/news/business/rss.xml",
				"source_level": "primary",
				"topic_hint": "全球商业新闻",
				"auth_required": false,
				"auth_type": "none",
				"credential_ref": "",
				"rate_limit_policy": {"requests_per_minute": 30},
				"usage_policy": "research_and_event_detection",
				"source_config": {"kind": "rss_feed", "language": "en"},
				"status": "active"
			}
		]
	}`)

	manifest, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	if got, want := len(manifest.Sources), 1; got != want {
		t.Fatalf("sources = %d, want %d", got, want)
	}

	source := manifest.Sources[0]
	if source.OriginSystem != "Vibe-Research" {
		t.Fatalf("OriginSystem = %q, want Vibe-Research", source.OriginSystem)
	}
	if source.SourceConfig["kind"] != "rss_feed" {
		t.Fatalf("SourceConfig[kind] = %v, want rss_feed", source.SourceConfig["kind"])
	}

	catalog := source.SourceCatalog()
	if catalog.ID != "vibe-research:bbc-business" {
		t.Fatalf("catalog ID = %q, want source id", catalog.ID)
	}
	if catalog.SourceConfig["language"] != "en" {
		t.Fatalf("catalog SourceConfig[language] = %v, want en", catalog.SourceConfig["language"])
	}
	if catalog.Status != domain.SourceCatalogStatusActive {
		t.Fatalf("catalog status = %q, want active", catalog.Status)
	}
}

func TestLoadManifestValidatesAIWebResearchSourceConfig(t *testing.T) {
	path := writeManifest(t, `{
		"sources": [
			{
				"id": "tidewise:ai-web-research:cn-finance-daily",
				"origin_system": "Tidewise",
				"stage": "ai_web_research",
				"ingest_channel": "ai_web_research",
				"provider_key": "llm_web_research",
				"connector_key": "llm_web_research",
				"parser_key": "llm_research_items",
				"source_type": "news",
				"source_group": "ai_web_research",
				"source_name": "AI Web Research 中文财经日度采集",
				"source_url": "tidewise://ai-web-research/cn-finance-daily",
				"source_level": "secondary",
				"topic_hint": "近24小时中国财经政经热点",
				"auth_required": true,
				"auth_type": "api_key",
				"credential_ref": "",
				"rate_limit_policy": {"requests_per_minute": 6},
				"usage_policy": "ai_web_research_governance",
				"source_config": {
					"kind": "llm_web_research",
					"web_search_plan": {
						"mode": "parallel",
						"tools": [
							{"provider": "tavily", "credential_ref": "env:TAVILY_API_KEY", "max_results": 10},
							{"provider": "bocha_web_search", "credential_ref": "env:BOCHA_API_KEY", "max_results": 10}
						]
					},
					"credential_refs": {
						"llm": "env:DEEPSEEK_API_KEY"
					},
					"llm_provider": "deepseek",
					"api_base_url": "https://api.deepseek.com",
					"api_protocol": "openai_compatible",
					"model": "deepseek-v4-pro",
					"prompt_ref": "ingestion/ai_web_research/cn-finance-daily.v1.md",
					"prompt_version": "v1",
					"prompt_variables": {"language": "zh-CN", "time_window": "24h"},
					"max_results": 20,
					"output_schema": {"type": "llm_research_items.v1"},
					"source_preferences": {"region": "china_finance"},
					"trusted_domains": ["pbc.gov.cn", "sse.com.cn"]
				},
				"status": "inactive"
			}
		]
	}`)

	manifest, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	source := manifest.Sources[0]
	if source.ConnectorKey != "llm_web_research" || source.ParserKey != "llm_research_items" {
		t.Fatalf("connector/parser = %s/%s, want llm_web_research/llm_research_items", source.ConnectorKey, source.ParserKey)
	}
	if source.SourceConfig["prompt_ref"] != "ingestion/ai_web_research/cn-finance-daily.v1.md" {
		t.Fatalf("prompt_ref = %v, want repo prompt reference", source.SourceConfig["prompt_ref"])
	}
}

func TestLoadManifestRejectsAIWebResearchSourceWithoutSearchPlan(t *testing.T) {
	path := writeManifest(t, `{
		"sources": [
			{
				"id": "tidewise:ai-web-research:missing-plan",
				"origin_system": "Tidewise",
				"stage": "ai_web_research",
				"ingest_channel": "ai_web_research",
				"provider_key": "llm_web_research",
				"connector_key": "llm_web_research",
				"parser_key": "llm_research_items",
				"source_type": "news",
				"source_group": "ai_web_research",
				"source_name": "AI Web Research 缺少搜索计划",
				"source_url": "tidewise://ai-web-research/missing-plan",
				"source_level": "secondary",
				"topic_hint": "测试",
				"auth_required": true,
				"auth_type": "api_key",
				"rate_limit_policy": {"requests_per_minute": 6},
				"usage_policy": "ai_web_research_governance",
				"source_config": {
					"kind": "llm_web_research",
					"credential_refs": {"llm": "env:DEEPSEEK_API_KEY"},
					"llm_provider": "deepseek",
					"api_base_url": "https://api.deepseek.com",
					"api_protocol": "openai_compatible",
					"model": "deepseek-v4-pro",
					"prompt_ref": "ingestion/ai_web_research/cn-finance-daily.v1.md",
					"prompt_version": "v1",
					"max_results": 20,
					"output_schema": {"type": "llm_research_items.v1"}
				},
				"status": "inactive"
			}
		]
	}`)

	if _, err := LoadFile(path); err == nil || !strings.Contains(err.Error(), "web_search_plan is required") {
		t.Fatalf("LoadFile() error = %v, want missing web_search_plan", err)
	}
}

func TestLoadManifestRejectsMissingRequiredFields(t *testing.T) {
	path := writeManifest(t, `{
		"sources": [
			{
				"id": "missing-name",
				"origin_system": "Vibe-Research",
				"stage": "content",
				"ingest_channel": "rss_feed",
				"provider_key": "rss",
				"connector_key": "rss_feed",
				"parser_key": "rss_item",
				"source_type": "news",
				"source_group": "content_events",
				"source_url": "https://example.com/rss.xml",
				"source_level": "secondary",
				"topic_hint": "事件",
				"usage_policy": "research",
				"status": "active"
			}
		]
	}`)

	if _, err := LoadFile(path); err == nil || !strings.Contains(err.Error(), "source_name is required") {
		t.Fatalf("LoadFile() error = %v, want missing source_name", err)
	}
}

func TestLoadManifestRejectsDuplicateIDs(t *testing.T) {
	path := writeManifest(t, `{
		"sources": [
			{"id": "duplicate-source", "origin_system": "Vibe-Research", "stage": "content", "ingest_channel": "rss_feed", "provider_key": "rss", "connector_key": "rss_feed", "parser_key": "rss_item", "source_type": "news", "source_group": "content_events", "source_name": "A", "source_url": "https://example.com/a.xml", "source_level": "secondary", "topic_hint": "事件", "usage_policy": "research", "status": "active"},
			{"id": "duplicate-source", "origin_system": "Vibe-Research", "stage": "content", "ingest_channel": "rss_feed", "provider_key": "rss", "connector_key": "rss_feed", "parser_key": "rss_item", "source_type": "news", "source_group": "content_events", "source_name": "B", "source_url": "https://example.com/b.xml", "source_level": "secondary", "topic_hint": "事件", "usage_policy": "research", "status": "active"}
		]
	}`)

	if _, err := LoadFile(path); err == nil || !strings.Contains(err.Error(), "duplicate source id") {
		t.Fatalf("LoadFile() error = %v, want duplicate source id", err)
	}
}

func TestLoadFilesMergesDefaultSeedManifests(t *testing.T) {
	manifest, err := LoadFiles(DefaultSeedPaths(filepath.Join("..", "..", "..", "..", "data", "source_catalogs"))...)
	if err != nil {
		t.Fatalf("LoadFiles() error = %v", err)
	}

	if got, want := len(manifest.Sources), 202; got != want {
		t.Fatalf("sources = %d, want %d", got, want)
	}
}

func TestLoadFilesRejectsDuplicateIDsAcrossFiles(t *testing.T) {
	first := writeManifest(t, `{
		"sources": [
			{"id": "duplicate-across-files", "origin_system": "Tidewise", "stage": "content", "ingest_channel": "rss_feed", "provider_key": "rss", "connector_key": "rss_feed", "parser_key": "rss_item", "source_type": "news", "source_group": "content_events", "source_name": "A", "source_url": "https://example.com/a.xml", "source_level": "secondary", "topic_hint": "事件", "usage_policy": "research", "status": "active"}
		]
	}`)
	second := writeManifest(t, `{
		"sources": [
			{"id": "duplicate-across-files", "origin_system": "Tidewise", "stage": "content", "ingest_channel": "rss_feed", "provider_key": "rss", "connector_key": "rss_feed", "parser_key": "rss_item", "source_type": "news", "source_group": "content_events", "source_name": "B", "source_url": "https://example.com/b.xml", "source_level": "secondary", "topic_hint": "事件", "usage_policy": "research", "status": "active"}
		]
	}`)

	if _, err := LoadFiles(first, second); err == nil || !strings.Contains(err.Error(), "duplicate source id") {
		t.Fatalf("LoadFiles() error = %v, want duplicate source id", err)
	}
}

func TestLoadManifestRejectsInvalidURL(t *testing.T) {
	path := writeManifest(t, `{
		"sources": [
			{"id": "invalid-url", "origin_system": "Vibe-Research", "stage": "content", "ingest_channel": "rss_feed", "provider_key": "rss", "connector_key": "rss_feed", "parser_key": "rss_item", "source_type": "news", "source_group": "content_events", "source_name": "Bad URL", "source_url": "not-a-url", "source_level": "secondary", "topic_hint": "事件", "usage_policy": "research", "status": "active"}
		]
	}`)

	if _, err := LoadFile(path); err == nil || !strings.Contains(err.Error(), "source_url must be an absolute http url") {
		t.Fatalf("LoadFile() error = %v, want invalid URL", err)
	}
}

func TestLoadManifestRejectsInvalidConnectorParser(t *testing.T) {
	path := writeManifest(t, `{
		"sources": [
			{"id": "invalid-connector", "origin_system": "Vibe-Research", "stage": "content", "ingest_channel": "rss_feed", "provider_key": "rss", "connector_key": "rss_feed", "parser_key": "text", "source_type": "news", "source_group": "content_events", "source_name": "Bad Parser", "source_url": "https://example.com/rss.xml", "source_level": "secondary", "topic_hint": "事件", "usage_policy": "research", "status": "active"}
		]
	}`)

	if _, err := LoadFile(path); err == nil || !strings.Contains(err.Error(), "unsupported connector/parser combination") {
		t.Fatalf("LoadFile() error = %v, want connector/parser error", err)
	}
}

func TestLoadManifestRejectsSensitiveFields(t *testing.T) {
	path := writeManifest(t, `{
		"sources": [
			{
				"id": "secret-source",
				"origin_system": "Stock",
				"stage": "market",
				"ingest_channel": "http_eastmoney",
				"provider_key": "eastmoney",
				"connector_key": "eastmoney",
				"parser_key": "eastmoney_json",
				"source_type": "market",
				"source_group": "market_data",
				"source_name": "Secret Source",
				"source_url": "https://example.com/api",
				"source_level": "secondary",
				"topic_hint": "行情",
				"usage_policy": "research",
				"source_config": {"api_key": "do-not-commit"},
				"status": "active"
			}
		]
	}`)

	if _, err := LoadFile(path); err == nil || !strings.Contains(err.Error(), "forbidden sensitive field") {
		t.Fatalf("LoadFile() error = %v, want sensitive field error", err)
	}
}

func writeManifest(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "source_catalogs.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return path
}
