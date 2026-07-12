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

func TestInMemoryRepositoryUpsertsBenchmarkObservationsIdempotently(t *testing.T) {
	repo := NewInMemoryRepository(nil)
	repo.SeedGraphEntity(GraphEntityNode{
		ID:         "benchmark-1",
		EntityKey:  "benchmark:us_10y",
		EntityType: domain.EntityTypeBenchmark,
		Status:     domain.StatusActive,
	})
	observedAt := time.Date(2026, 7, 12, 9, 30, 0, 0, time.UTC)
	observation := domain.BenchmarkObservation{
		ID:                "observation-1",
		BenchmarkEntityID: "benchmark-1",
		ObservedAt:        observedAt,
		Value:             "4.25",
		Unit:              "percent",
		SourceName:        "US Treasury",
		QualityStatus:     domain.BenchmarkObservationQualityRaw,
	}

	first, err := repo.UpsertBenchmarkObservation(context.Background(), observation)
	if err != nil {
		t.Fatalf("UpsertBenchmarkObservation(first) error = %v", err)
	}
	observation.ID = "observation-2"
	observation.Value = "4.30"
	observation.QualityStatus = domain.BenchmarkObservationQualityValidated
	second, err := repo.UpsertBenchmarkObservation(context.Background(), observation)
	if err != nil {
		t.Fatalf("UpsertBenchmarkObservation(second) error = %v", err)
	}

	if !first.Created {
		t.Fatal("first write should create observation")
	}
	if second.Created {
		t.Fatal("same benchmark/time/source should update existing observation")
	}
	if second.Observation.ID != "observation-1" || second.Observation.Value != "4.30" {
		t.Fatalf("updated observation = %+v, want original id with updated value", second.Observation)
	}
}

func TestInMemoryRepositoryAllowsDifferentBenchmarkObservationSourcesAndSortsDescending(t *testing.T) {
	repo := NewInMemoryRepository(nil)
	repo.SeedGraphEntity(GraphEntityNode{
		ID:         "benchmark-1",
		EntityKey:  "benchmark:us_10y",
		EntityType: domain.EntityTypeBenchmark,
		Status:     domain.StatusActive,
	})
	firstTime := time.Date(2026, 7, 12, 9, 30, 0, 0, time.UTC)
	secondTime := firstTime.Add(time.Hour)
	for _, observation := range []domain.BenchmarkObservation{
		{ID: "observation-1", BenchmarkEntityID: "benchmark-1", ObservedAt: firstTime, Value: "4.25", Unit: "percent", SourceName: "US Treasury", QualityStatus: domain.BenchmarkObservationQualityRaw},
		{ID: "observation-2", BenchmarkEntityID: "benchmark-1", ObservedAt: firstTime, Value: "4.26", Unit: "percent", SourceName: "Market Data Vendor", QualityStatus: domain.BenchmarkObservationQualityRaw},
		{ID: "observation-3", BenchmarkEntityID: "benchmark-1", ObservedAt: secondTime, Value: "4.27", Unit: "percent", SourceName: "US Treasury", QualityStatus: domain.BenchmarkObservationQualityRaw},
	} {
		if _, err := repo.UpsertBenchmarkObservation(context.Background(), observation); err != nil {
			t.Fatalf("UpsertBenchmarkObservation(%s) error = %v", observation.ID, err)
		}
	}

	observations, err := repo.ListBenchmarkObservations(context.Background(), BenchmarkObservationFilter{BenchmarkEntityID: "benchmark-1"})
	if err != nil {
		t.Fatalf("ListBenchmarkObservations() error = %v", err)
	}

	if got, want := len(observations), 3; got != want {
		t.Fatalf("observations length = %d, want %d", got, want)
	}
	if observations[0].ID != "observation-3" {
		t.Fatalf("first observation id = %q, want latest observation-3", observations[0].ID)
	}
	if observations[1].SourceName == observations[2].SourceName {
		t.Fatalf("same-time observations should preserve different sources: %+v", observations)
	}
}

func TestInMemoryRepositoryRejectsInvalidBenchmarkObservation(t *testing.T) {
	repo := NewInMemoryRepository(nil)
	repo.SeedGraphEntity(GraphEntityNode{ID: "index-1", EntityKey: "index:vix", EntityType: domain.EntityTypeIndex, Status: domain.StatusActive})
	invalid := domain.BenchmarkObservation{
		ID:                "observation-1",
		BenchmarkEntityID: "index-1",
		ObservedAt:        time.Now(),
		Value:             "20",
		Unit:              "points",
		SourceName:        "Cboe",
		QualityStatus:     domain.BenchmarkObservationQualityRaw,
	}
	if _, err := repo.UpsertBenchmarkObservation(context.Background(), invalid); err == nil {
		t.Fatal("UpsertBenchmarkObservation() error = nil, want non-benchmark entity rejection")
	}

	invalid.BenchmarkEntityID = "benchmark-1"
	invalid.QualityStatus = "estimated"
	repo.SeedGraphEntity(GraphEntityNode{ID: "benchmark-1", EntityKey: "benchmark:test", EntityType: domain.EntityTypeBenchmark, Status: domain.StatusActive})
	if _, err := repo.UpsertBenchmarkObservation(context.Background(), invalid); err == nil {
		t.Fatal("UpsertBenchmarkObservation() error = nil, want invalid quality status rejection")
	}
}

func TestBenchmarkObservationFilterEntityIDUsesNullableUUID(t *testing.T) {
	if got := benchmarkObservationFilterEntityID(""); got != nil {
		t.Fatalf("empty filter entity id = %#v, want nil to avoid empty string UUID comparison", got)
	}
	if got := benchmarkObservationFilterEntityID("benchmark-1"); got != NormalizeUUID("benchmark-1") {
		t.Fatalf("non-empty filter entity id = %#v, want normalized UUID", got)
	}
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

func TestInMemoryRepositoryListsEntityGraphSnapshot(t *testing.T) {
	repo := NewInMemoryRepository(nil)
	now := time.Date(2026, 7, 10, 9, 30, 0, 0, time.UTC)

	repo.SeedGraphEntity(GraphEntityNode{
		ID:            "entity-1",
		EntityKey:     "economy:cn",
		EntityType:    domain.EntityTypeEconomy,
		LayerCode:     "economy",
		Name:          "中国",
		CanonicalName: "中国",
		Aliases:       []string{"China"},
		Status:        domain.StatusActive,
		UpdatedAt:     now,
	})
	repo.SeedGraphEntity(GraphEntityNode{
		ID:            "entity-2",
		EntityType:    domain.EntityTypeAllianceOrg,
		LayerCode:     "alliance",
		Name:          "G20",
		CanonicalName: "二十国集团",
		Status:        domain.StatusActive,
		UpdatedAt:     now.Add(time.Minute),
	})
	repo.SeedGraphEdge(GraphEntityEdge{
		ID:           "edge-1",
		FromEntityID: "entity-1",
		ToEntityID:   "entity-2",
		RelationType: "member_of",
		EvidenceNote: "基础关系",
		Status:       domain.StatusActive,
		UpdatedAt:    now.Add(2 * time.Minute),
	})

	nodes, err := repo.ListGraphEntityNodes(context.Background())
	if err != nil {
		t.Fatalf("ListGraphEntityNodes() error = %v", err)
	}
	edges, err := repo.ListGraphEntityEdges(context.Background())
	if err != nil {
		t.Fatalf("ListGraphEntityEdges() error = %v", err)
	}

	if got, want := graphEntityNodeIDs(nodes), []string{"entity-1", "entity-2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("node ids = %v, want %v", got, want)
	}
	if nodes[1].EntityKey != "alliance_org:entity-2" {
		t.Fatalf("fallback entity key = %q, want alliance_org:entity-2", nodes[1].EntityKey)
	}
	if !reflect.DeepEqual(nodes[0].Aliases, []string{"China"}) {
		t.Fatalf("node aliases = %v, want [China]", nodes[0].Aliases)
	}
	if len(edges) != 1 || edges[0].RelationType != "member_of" {
		t.Fatalf("edges = %+v, want member_of edge", edges)
	}
}

func TestInMemoryRepositoryListsOnlyActiveGraphProjectionInputs(t *testing.T) {
	repo := NewInMemoryRepository(nil)
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	for _, node := range []GraphEntityNode{
		{ID: "active-a", EntityType: domain.EntityTypeEconomy, LayerCode: "economy", Name: "A", CanonicalName: "A", Status: domain.StatusActive, UpdatedAt: now},
		{ID: "active-b", EntityType: domain.EntityTypeMarket, LayerCode: "market", Name: "B", CanonicalName: "B", Status: domain.StatusActive, UpdatedAt: now},
		{ID: "inactive", EntityType: domain.EntityTypeSector, LayerCode: "sector", Name: "旧板块", CanonicalName: "旧板块", Status: domain.StatusInactive, UpdatedAt: now},
	} {
		repo.SeedGraphEntity(node)
	}
	for _, edge := range []GraphEntityEdge{
		{ID: "active-edge", FromEntityID: "active-a", ToEntityID: "active-b", RelationType: "has_market", Status: domain.StatusActive, UpdatedAt: now},
		{ID: "inactive-endpoint", FromEntityID: "active-b", ToEntityID: "inactive", RelationType: "covers_sector", Status: domain.StatusActive, UpdatedAt: now},
		{ID: "inactive-edge", FromEntityID: "active-a", ToEntityID: "active-b", RelationType: "has_market", Status: domain.StatusInactive, UpdatedAt: now},
	} {
		repo.SeedGraphEdge(edge)
	}

	nodes, err := repo.ListGraphEntityNodes(context.Background())
	if err != nil {
		t.Fatalf("ListGraphEntityNodes() error = %v", err)
	}
	edges, err := repo.ListGraphEntityEdges(context.Background())
	if err != nil {
		t.Fatalf("ListGraphEntityEdges() error = %v", err)
	}
	if got, want := graphEntityNodeIDs(nodes), []string{"active-a", "active-b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("node ids = %v, want %v", got, want)
	}
	if len(edges) != 1 || edges[0].ID != "active-edge" {
		t.Fatalf("edges = %+v, want only active-edge", edges)
	}
}

func TestInMemoryRepositoryPreservesSectorClassificationForProjection(t *testing.T) {
	repo := NewInMemoryRepository(nil)
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	repo.SeedGraphEntity(GraphEntityNode{
		ID: "sector", EntityKey: "sector:industry_test", EntityType: domain.EntityTypeSector,
		LayerCode: "sector", Name: "测试行业", CanonicalName: "测试行业",
		ClassificationCode: domain.SectorClassificationIndustry, Status: domain.StatusActive, UpdatedAt: now,
	})
	repo.SeedGraphEntity(GraphEntityNode{
		ID: "market", EntityKey: "market:test", EntityType: domain.EntityTypeMarket,
		LayerCode: "market", Name: "测试市场", CanonicalName: "测试市场", Status: domain.StatusActive, UpdatedAt: now,
	})

	nodes, err := repo.ListGraphEntityNodes(context.Background())
	if err != nil {
		t.Fatalf("ListGraphEntityNodes() error = %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("nodes = %+v, want two active nodes", nodes)
	}
	if nodes[1].ClassificationCode != domain.SectorClassificationIndustry {
		t.Fatalf("sector classification = %q", nodes[1].ClassificationCode)
	}
	if nodes[0].ClassificationCode != "" {
		t.Fatalf("non-sector classification = %q, want empty", nodes[0].ClassificationCode)
	}
}

func TestInMemoryRepositoryRecordsGraphProjectionRuns(t *testing.T) {
	repo := NewInMemoryRepository(nil)
	started := time.Date(2026, 7, 10, 10, 0, 0, 0, time.UTC)
	finished := started.Add(2 * time.Second)

	run, err := repo.CreateGraphProjectionRun(context.Background(), GraphProjectionRun{
		ID:             "run-1",
		ProjectionType: GraphProjectionTypeEntityGraph,
		Mode:           GraphProjectionModeProjectEntities,
		Status:         GraphProjectionRunStatusRunning,
		StartedAt:      started,
		ConfigSummary:  map[string]any{"namespace": "tidewise"},
	})
	if err != nil {
		t.Fatalf("CreateGraphProjectionRun() error = %v", err)
	}
	if run.Status != GraphProjectionRunStatusRunning {
		t.Fatalf("created run status = %q", run.Status)
	}

	if err := repo.RecordGraphProjectionRunItem(context.Background(), GraphProjectionRunItem{
		ID:           "item-1",
		RunID:        "run-1",
		ItemType:     GraphProjectionRunItemTypeRelationship,
		ItemKey:      "edge-1",
		Status:       GraphProjectionRunItemStatusFailed,
		ErrorMessage: "missing endpoint",
	}); err != nil {
		t.Fatalf("RecordGraphProjectionRunItem() error = %v", err)
	}

	run.Status = GraphProjectionRunStatusPartial
	run.FinishedAt = &finished
	run.SourceRowCount = 3
	run.ProjectedCount = 2
	run.SkippedCount = 0
	run.FailedCount = 1
	run.ErrorSummary = "1 relationship failed"
	if err := repo.CompleteGraphProjectionRun(context.Background(), run); err != nil {
		t.Fatalf("CompleteGraphProjectionRun() error = %v", err)
	}

	runs, err := repo.RecentGraphProjectionRuns(context.Background(), 5)
	if err != nil {
		t.Fatalf("RecentGraphProjectionRuns() error = %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("recent run count = %d, want 1", len(runs))
	}
	if runs[0].Status != GraphProjectionRunStatusPartial || runs[0].FailedCount != 1 {
		t.Fatalf("recent run = %+v, want partial with one failure", runs[0])
	}
}

func TestInMemoryRepositoryListsRawDocumentsWithPaginationAndTitleSearch(t *testing.T) {
	repo := NewInMemoryRepository(nil)
	base := time.Date(2026, 7, 9, 10, 0, 0, 0, time.UTC)
	for _, doc := range []domain.RawDocument{
		validRawDocumentWithTitle("raw-1", "央行公布金融数据", base.Add(2*time.Minute)),
		validRawDocumentWithTitle("raw-2", "美联储维持利率不变", base.Add(time.Minute)),
		validRawDocumentWithTitle("raw-3", "央行开展逆回购操作", base),
	} {
		if _, err := repo.UpsertRawDocument(context.Background(), doc); err != nil {
			t.Fatalf("UpsertRawDocument(%s) error = %v", doc.ID, err)
		}
	}

	page, err := repo.ListRawDocuments(context.Background(), RawDocumentListFilter{Title: "央行", Page: 1, PageSize: 1})
	if err != nil {
		t.Fatalf("ListRawDocuments() error = %v", err)
	}

	if page.Total != 2 || page.Page != 1 || page.PageSize != 1 {
		t.Fatalf("page = %+v, want total/page/page_size 2/1/1", page)
	}
	if got := rawDocumentIDs(page.Items); !reflect.DeepEqual(got, []string{"raw-1"}) {
		t.Fatalf("page ids = %v, want newest matching raw-1", got)
	}
}

func TestInMemoryRepositoryListsEventsWithFilters(t *testing.T) {
	repo := NewInMemoryRepository(nil)
	eventTime := time.Date(2026, 7, 9, 10, 0, 0, 0, time.UTC)
	if err := repo.SeedEvent(context.Background(), domain.Event{
		ID:          "event-1",
		Title:       "美联储维持利率不变",
		Summary:     "FOMC 会议维持联邦基金利率目标区间不变。",
		EventTime:   &eventTime,
		FirstSeenAt: eventTime.Add(10 * time.Minute),
		EventStatus: domain.EventStatusConfirmed,
		FactStatus:  domain.FactStatusVerified,
		DedupeKey:   "fed-rate-hold",
	}); err != nil {
		t.Fatalf("SeedEvent(event-1) error = %v", err)
	}
	if err := repo.SeedEvent(context.Background(), domain.Event{
		ID:          "event-2",
		Title:       "欧洲央行政策表态",
		Summary:     "欧洲央行官员表态。",
		EventTime:   ptrTimeForRepositoryTest(eventTime.Add(-24 * time.Hour)),
		FirstSeenAt: eventTime.Add(-23 * time.Hour),
		EventStatus: domain.EventStatusCandidate,
		FactStatus:  domain.FactStatusUnverified,
		DedupeKey:   "ecb-policy",
	}); err != nil {
		t.Fatalf("SeedEvent(event-2) error = %v", err)
	}

	page, err := repo.ListEvents(context.Background(), EventListFilter{
		Title:         "美联储",
		EventStatus:   domain.EventStatusConfirmed,
		FactStatus:    domain.FactStatusVerified,
		EventTimeFrom: ptrTimeForRepositoryTest(eventTime.Add(-time.Hour)),
		EventTimeTo:   ptrTimeForRepositoryTest(eventTime.Add(time.Hour)),
		FirstSeenFrom: ptrTimeForRepositoryTest(eventTime),
		FirstSeenTo:   ptrTimeForRepositoryTest(eventTime.Add(time.Hour)),
		Page:          1,
		PageSize:      50,
	})
	if err != nil {
		t.Fatalf("ListEvents() error = %v", err)
	}

	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].ID != "event-1" {
		t.Fatalf("event page = %+v, want only event-1", page)
	}
}

func TestInMemoryRepositoryListsSourceCatalogsByStatus(t *testing.T) {
	repo := NewInMemoryRepository([]domain.SourceCatalog{
		{ID: "source-1", ProviderKey: "rss", IngestChannel: "rss_feed", SourceType: "news", SourceName: "RSS 1", Status: domain.SourceCatalogStatusActive},
		{ID: "source-2", ProviderKey: "rss", IngestChannel: "rss_feed", SourceType: "news", SourceName: "RSS 2", Status: domain.SourceCatalogStatusInactive},
		{ID: "source-3", ProviderKey: "eastmoney", IngestChannel: "http_api", SourceType: "market", SourceName: "东方财富", Status: domain.SourceCatalogStatusDisabled},
	})

	sources, err := repo.ListSourceCatalogs(context.Background(), SourceCatalogListFilter{Status: domain.SourceCatalogStatusInactive})
	if err != nil {
		t.Fatalf("ListSourceCatalogs() error = %v", err)
	}

	if got := sourceIDs(sources); !reflect.DeepEqual(got, []string{"source-2"}) {
		t.Fatalf("source ids = %v, want inactive source-2", got)
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

func validRawDocumentWithTitle(id string, title string, collectedAt time.Time) domain.RawDocument {
	doc := validRawDocument(id)
	doc.Title = title
	doc.ContentHash = id + "-hash"
	doc.CollectedAt = collectedAt
	return doc
}

func rawDocumentIDs(items []domain.RawDocument) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func graphEntityNodeIDs(items []GraphEntityNode) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func ptrTimeForRepositoryTest(value time.Time) *time.Time {
	return &value
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
