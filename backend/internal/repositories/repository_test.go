package repositories

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
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

func TestInMemoryRepositoryUpsertsRawDocumentByExternalID(t *testing.T) {
	repo := NewInMemoryRepository(nil)
	doc := validRawDocument("raw-1")
	doc.SourceExternalID = "external-1"

	first, err := repo.UpsertRawDocument(context.Background(), doc)
	if err != nil {
		t.Fatalf("UpsertRawDocument() error = %v", err)
	}
	if !first.Created {
		t.Fatal("first write should create document")
	}

	duplicate := validRawDocument("raw-2")
	duplicate.SourceExternalID = "external-1"
	duplicate.ContentHash = "hash-2"

	second, err := repo.UpsertRawDocument(context.Background(), duplicate)
	if err != nil {
		t.Fatalf("UpsertRawDocument() duplicate error = %v", err)
	}
	if second.Created {
		t.Fatal("duplicate external id should not create document")
	}
	if second.Document.ID != "raw-1" {
		t.Fatalf("duplicate document id = %q, want raw-1", second.Document.ID)
	}
}

func TestInMemoryRepositoryUpsertsRawDocumentByContentHash(t *testing.T) {
	repo := NewInMemoryRepository(nil)
	doc := validRawDocument("raw-1")
	doc.SourceExternalID = ""

	if _, err := repo.UpsertRawDocument(context.Background(), doc); err != nil {
		t.Fatalf("UpsertRawDocument() error = %v", err)
	}

	duplicate := validRawDocument("raw-2")
	duplicate.SourceExternalID = ""

	result, err := repo.UpsertRawDocument(context.Background(), duplicate)
	if err != nil {
		t.Fatalf("UpsertRawDocument() duplicate error = %v", err)
	}
	if result.Created {
		t.Fatal("duplicate content hash should not create document")
	}
	if result.DuplicateOf != "raw-1" {
		t.Fatalf("DuplicateOf = %q, want raw-1", result.DuplicateOf)
	}
}

func TestInMemoryRepositoryUpdatesRawDocumentStatus(t *testing.T) {
	repo := NewInMemoryRepository(nil)
	doc := validRawDocument("raw-1")
	if _, err := repo.UpsertRawDocument(context.Background(), doc); err != nil {
		t.Fatalf("UpsertRawDocument() error = %v", err)
	}

	if err := repo.UpdateRawDocumentStatus(context.Background(), "raw-1", domain.IngestStatusFailed); err != nil {
		t.Fatalf("UpdateRawDocumentStatus() error = %v", err)
	}

	stored, ok := repo.RawDocument("raw-1")
	if !ok {
		t.Fatal("raw document not found")
	}
	if stored.IngestStatus != domain.IngestStatusFailed {
		t.Fatalf("IngestStatus = %q, want %q", stored.IngestStatus, domain.IngestStatusFailed)
	}
}

func TestInMemoryRepositoryLoadsDefaultSchedulerConfig(t *testing.T) {
	repo := NewInMemoryRepository(nil)

	config, err := repo.LoadSchedulerConfig(context.Background())
	if err != nil {
		t.Fatalf("LoadSchedulerConfig() error = %v", err)
	}

	if config.Enabled {
		t.Fatal("default scheduler config must be disabled")
	}
	if config.Mode != domain.SchedulerModeInterval {
		t.Fatalf("Mode = %q, want %q", config.Mode, domain.SchedulerModeInterval)
	}
	if config.IntervalMinutes != 60 {
		t.Fatalf("IntervalMinutes = %d, want 60", config.IntervalMinutes)
	}
	if config.SourceFilter.ProviderKey != "" || config.SourceFilter.IngestChannel != "" || config.SourceFilter.SourceType != "" {
		t.Fatalf("SourceFilter = %+v, want empty global filter", config.SourceFilter)
	}
}

func TestInMemoryRepositorySavesSchedulerConfig(t *testing.T) {
	repo := NewInMemoryRepository(nil)
	config := domain.SchedulerConfig{
		ID:             "default",
		Enabled:        true,
		Mode:           domain.SchedulerModeFixedTimes,
		FixedTimes:     []string{"09:00", "12:00", "15:00", "18:00", "21:00"},
		Concurrency:    3,
		BatchSize:      30,
		TimeoutSeconds: 240,
		Timezone:       "Asia/Shanghai",
		SourceFilter: domain.SchedulerSourceFilter{
			ProviderKey:   "llm_web_research",
			IngestChannel: "ai_web_research",
			SourceType:    "news",
		},
	}

	saved, err := repo.SaveSchedulerConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("SaveSchedulerConfig() error = %v", err)
	}
	loaded, err := repo.LoadSchedulerConfig(context.Background())
	if err != nil {
		t.Fatalf("LoadSchedulerConfig() error = %v", err)
	}

	if saved.ConfigVersion != 1 {
		t.Fatalf("saved ConfigVersion = %d, want 1", saved.ConfigVersion)
	}
	if !reflect.DeepEqual(loaded.FixedTimes, config.FixedTimes) {
		t.Fatalf("FixedTimes = %v, want %v", loaded.FixedTimes, config.FixedTimes)
	}
	if loaded.SourceFilter.ProviderKey != "llm_web_research" {
		t.Fatalf("ProviderKey = %q, want llm_web_research", loaded.SourceFilter.ProviderKey)
	}
}

func TestInMemoryRepositoryRecordsIngestionRuns(t *testing.T) {
	repo := NewInMemoryRepository(nil)
	started := time.Now().Add(-time.Minute)
	finished := time.Now()
	run := domain.IngestionRun{
		ID:          "run-1",
		TriggerType: domain.SchedulerTriggerManualOnce,
		Status:      domain.SchedulerRunStatusRunning,
		StartedAt:   started,
	}

	if _, err := repo.CreateIngestionRun(context.Background(), run); err != nil {
		t.Fatalf("CreateIngestionRun() error = %v", err)
	}

	sourceResult := domain.IngestionRunSource{
		ID:                 "run-source-1",
		RunID:              "run-1",
		SourceID:           "source-1",
		Status:             domain.SchedulerSourceRunStatusSucceeded,
		DocumentsWritten:   5,
		DocumentsDuplicate: 2,
		StartedAt:          started,
		FinishedAt:         &finished,
		DurationMillis:     120,
	}
	if err := repo.RecordIngestionRunSource(context.Background(), sourceResult); err != nil {
		t.Fatalf("RecordIngestionRunSource() error = %v", err)
	}

	run.Status = domain.SchedulerRunStatusSucceeded
	run.FinishedAt = &finished
	run.TotalSources = 1
	run.SucceededSources = 1
	if err := repo.CompleteIngestionRun(context.Background(), run); err != nil {
		t.Fatalf("CompleteIngestionRun() error = %v", err)
	}

	runs, err := repo.RecentIngestionRuns(context.Background(), 5)
	if err != nil {
		t.Fatalf("RecentIngestionRuns() error = %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("recent runs length = %d, want 1", len(runs))
	}
	if runs[0].Status != domain.SchedulerRunStatusSucceeded {
		t.Fatalf("run status = %q, want succeeded", runs[0].Status)
	}
	if got := repo.IngestionRunSources("run-1"); len(got) != 1 {
		t.Fatalf("run source results length = %d, want 1", len(got))
	}
}

func validRawDocument(id string) domain.RawDocument {
	return domain.RawDocument{
		ID:            id,
		SourceID:      "source-1",
		IngestChannel: "rss_feed",
		SourceType:    "news",
		SourceName:    "示例来源",
		Title:         "示例标题",
		ContentHash:   "hash-1",
		CollectedAt:   time.Now(),
		IngestStatus:  domain.IngestStatusCollected,
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
