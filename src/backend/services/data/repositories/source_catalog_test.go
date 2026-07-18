package repositories

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
)

func TestInMemoryRepositoryListsActiveSources(t *testing.T) {
	repo := NewInMemoryRepository([]domain.SourceCatalog{
		{
			ID:            "source-1",
			IngestChannel: "rss_feed",
			ProviderKey:   "rss",
			SourceType:    "news",
			Status:        domain.SourceCatalogStatusActive,
		},
		{
			ID:            "source-2",
			IngestChannel: "rss_feed",
			ProviderKey:   "rss",
			SourceType:    "news",
			Status:        domain.SourceCatalogStatusDisabled,
		},
		{
			ID:            "source-3",
			IngestChannel: "web_fetch",
			ProviderKey:   "web",
			SourceType:    "news",
			Status:        domain.SourceCatalogStatusActive,
		},
		{
			ID:            "source-4",
			IngestChannel: "rss_feed",
			ProviderKey:   "rss",
			SourceType:    "market",
			Status:        domain.SourceCatalogStatusActive,
		},
	})

	sources, err := repo.ActiveSources(context.Background(), SourceCatalogFilter{
		SourceID:      "source-1",
		ProviderKey:   "rss",
		IngestChannel: "rss_feed",
		SourceType:    "news",
	})
	if err != nil {
		t.Fatalf("ActiveSources() error = %v", err)
	}

	if got, want := sourceIDs(sources), []string{"source-1"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("source ids = %v, want %v", got, want)
	}
}

func TestInMemoryRepositoryPreservesSourceConfigAndDefaultsEmptyConfig(t *testing.T) {
	repo := NewInMemoryRepository(nil)

	configured := domain.SourceCatalog{
		ID:            "source-configured",
		IngestChannel: "web_fetch",
		ProviderKey:   "web",
		SourceName:    "配置化网页来源",
		SourceConfig: map[string]any{
			"kind":     "web_page",
			"selector": ".article",
		},
		Status: domain.SourceCatalogStatusActive,
	}
	empty := domain.SourceCatalog{
		ID:            "source-empty",
		IngestChannel: "rss_feed",
		ProviderKey:   "rss",
		SourceName:    "空配置来源",
		Status:        domain.SourceCatalogStatusActive,
	}

	if err := repo.SeedSource(context.Background(), configured); err != nil {
		t.Fatalf("SeedSource(configured) error = %v", err)
	}
	if err := repo.SeedSource(context.Background(), empty); err != nil {
		t.Fatalf("SeedSource(empty) error = %v", err)
	}

	sources, err := repo.ActiveSources(context.Background(), SourceCatalogFilter{})
	if err != nil {
		t.Fatalf("ActiveSources() error = %v", err)
	}

	byID := map[string]domain.SourceCatalog{}
	for _, source := range sources {
		byID[source.ID] = source
	}

	if got := byID["source-configured"].SourceConfig["kind"]; got != "web_page" {
		t.Fatalf("configured SourceConfig[kind] = %v, want web_page", got)
	}
	if got := byID["source-configured"].SourceConfig["selector"]; got != ".article" {
		t.Fatalf("configured SourceConfig[selector] = %v, want .article", got)
	}
	if byID["source-empty"].SourceConfig == nil {
		t.Fatal("empty SourceConfig must default to an empty map")
	}
	if len(byID["source-empty"].SourceConfig) != 0 {
		t.Fatalf("empty SourceConfig length = %d, want 0", len(byID["source-empty"].SourceConfig))
	}
}

func TestInMemoryRepositoryBuildsSourceCatalogStats(t *testing.T) {
	repo := NewInMemoryRepository([]domain.SourceCatalog{
		{
			ID:            "source-1",
			IngestChannel: "rss_feed",
			ProviderKey:   "rss",
			SourceType:    "news",
			UsagePolicy:   "research",
			Status:        domain.SourceCatalogStatusActive,
		},
		{
			ID:            "source-2",
			IngestChannel: "rss_feed",
			ProviderKey:   "rss",
			SourceType:    "news",
			UsagePolicy:   "research",
			Status:        domain.SourceCatalogStatusInactive,
		},
		{
			ID:            "source-3",
			IngestChannel: "http_api",
			ProviderKey:   "eastmoney",
			SourceType:    "sector",
			UsagePolicy:   "market-data",
			Status:        domain.SourceCatalogStatusDisabled,
		},
	})

	stats, err := repo.SourceCatalogStats(context.Background())
	if err != nil {
		t.Fatalf("SourceCatalogStats() error = %v", err)
	}

	if stats.Total != 3 {
		t.Fatalf("Total = %d, want 3", stats.Total)
	}
	assertStatsCount(t, stats.ByProviderKey, "rss", 2)
	assertStatsCount(t, stats.ByProviderKey, "eastmoney", 1)
	assertStatsCount(t, stats.ByIngestChannel, "rss_feed", 2)
	assertStatsCount(t, stats.ByIngestChannel, "http_api", 1)
	assertStatsCount(t, stats.BySourceType, "news", 2)
	assertStatsCount(t, stats.BySourceType, "sector", 1)
	assertStatsCount(t, stats.ByUsagePolicy, "research", 2)
	assertStatsCount(t, stats.ByUsagePolicy, "market-data", 1)
	assertStatsCount(t, stats.ByStatus, string(domain.SourceCatalogStatusActive), 1)
	assertStatsCount(t, stats.ByStatus, string(domain.SourceCatalogStatusInactive), 1)
	assertStatsCount(t, stats.ByStatus, string(domain.SourceCatalogStatusDisabled), 1)
}

func TestScanSourceReadsSourceConfigAndDefaultsEmptyConfig(t *testing.T) {
	source, err := scanSource(fixedSourceScanner{values: []any{
		"source-1",
		"web_fetch",
		"web",
		"web_fetch",
		"web_article",
		"news",
		"示例网页",
		"https://example.com/news",
		"secondary",
		"宏观",
		"",
		"",
		false,
		"none",
		"",
		[]byte(`{"kind":"web_page","selectors":["article",".content"]}`),
		[]byte(`{}`),
		"research",
		domain.SourceCatalogStatusActive,
	}})
	if err != nil {
		t.Fatalf("scanSource() error = %v", err)
	}
	if got := source.SourceConfig["kind"]; got != "web_page" {
		t.Fatalf("SourceConfig[kind] = %v, want web_page", got)
	}
	if got := source.SourceConfig["selectors"].([]any)[1]; got != ".content" {
		t.Fatalf("SourceConfig[selectors][1] = %v, want .content", got)
	}

	empty, err := scanSource(fixedSourceScanner{values: []any{
		"source-2",
		"rss_feed",
		"rss",
		"rss_feed",
		"rss_item",
		"news",
		"示例 RSS",
		"https://example.com/feed.xml",
		"secondary",
		"",
		"",
		"",
		false,
		"none",
		"",
		[]byte(`{}`),
		[]byte(`{}`),
		"research",
		domain.SourceCatalogStatusActive,
	}})
	if err != nil {
		t.Fatalf("scanSource(empty) error = %v", err)
	}
	if empty.SourceConfig == nil {
		t.Fatal("empty scanned SourceConfig must default to an empty map")
	}
	if len(empty.SourceConfig) != 0 {
		t.Fatalf("empty scanned SourceConfig length = %d, want 0", len(empty.SourceConfig))
	}
}

func sourceIDs(sources []domain.SourceCatalog) []string {
	ids := make([]string, 0, len(sources))
	for _, source := range sources {
		ids = append(ids, source.ID)
	}
	return ids
}

func assertStatsCount(t *testing.T, counts map[string]int, key string, want int) {
	t.Helper()

	if got := counts[key]; got != want {
		t.Fatalf("stats[%q] = %d, want %d", key, got, want)
	}
}

type fixedSourceScanner struct {
	values []any
}

func (s fixedSourceScanner) Scan(dest ...any) error {
	if len(dest) != len(s.values) {
		return fmt.Errorf("scan destinations = %d, want %d", len(dest), len(s.values))
	}
	for index, value := range s.values {
		switch target := dest[index].(type) {
		case *string:
			*target = value.(string)
		case *bool:
			*target = value.(bool)
		case *[]byte:
			*target = value.([]byte)
		case *domain.SourceCatalogStatus:
			*target = value.(domain.SourceCatalogStatus)
		default:
			return fmt.Errorf("unsupported scan target %T at index %d", target, index)
		}
	}
	return nil
}
