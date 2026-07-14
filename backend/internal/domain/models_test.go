package domain

import (
	"reflect"
	"testing"
	"time"
)

func TestEntityNodeValidate(t *testing.T) {
	node := EntityNode{
		ID:            "entity-1",
		EntityType:    EntityTypeCompany,
		LayerCode:     "company",
		Name:          "示例公司",
		CanonicalName: "示例公司",
		Status:        StatusActive,
	}

	if err := node.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	node.Status = "unknown"
	if err := node.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want invalid status error")
	}
}

func TestEntityNodeValidateRejectsUnsupportedEntityType(t *testing.T) {
	node := EntityNode{
		ID:            "entity-1",
		EntityType:    "unknown",
		LayerCode:     "unknown",
		Name:          "未知实体",
		CanonicalName: "未知实体",
		Status:        StatusActive,
	}

	if err := node.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want unsupported entity type error")
	}
}

func TestBenchmarkEntityTypeAndProfileValidate(t *testing.T) {
	node := EntityNode{
		ID:            "benchmark-1",
		EntityType:    EntityTypeBenchmark,
		LayerCode:     "benchmark",
		Name:          "美国10年期国债收益率",
		CanonicalName: "美国10年期国债收益率",
		Status:        StatusActive,
	}
	if err := node.Validate(); err != nil {
		t.Fatalf("benchmark EntityNode.Validate() error = %v", err)
	}

	profile := BenchmarkProfile{
		EntityID:           "benchmark-1",
		BenchmarkType:      BenchmarkTypeGovernmentBondYield,
		OfficialSeriesCode: "",
		Provider:           "us_treasury",
		Tenor:              "10Y",
		CurrencyCode:       "USD",
		Unit:               "percent",
		Frequency:          "daily",
		SourceURL:          "https://home.treasury.gov/",
	}
	if err := profile.Validate(); err != nil {
		t.Fatalf("BenchmarkProfile.Validate() error = %v", err)
	}

	profile.BenchmarkType = "index"
	if err := profile.Validate(); err == nil {
		t.Fatal("BenchmarkProfile.Validate() error = nil, want invalid benchmark type error")
	}
}

func TestBenchmarkObservationQualityStatusValidate(t *testing.T) {
	validStatuses := []BenchmarkObservationQualityStatus{
		BenchmarkObservationQualityRaw,
		BenchmarkObservationQualityValidated,
		BenchmarkObservationQualitySuspect,
		BenchmarkObservationQualityRejected,
	}
	for _, status := range validStatuses {
		observation := BenchmarkObservation{
			ID:                "observation-1",
			BenchmarkEntityID: "benchmark-1",
			ObservedAt:        time.Now(),
			Value:             "4.25",
			Unit:              "percent",
			SourceName:        "US Treasury",
			QualityStatus:     status,
		}
		if err := observation.Validate(); err != nil {
			t.Fatalf("BenchmarkObservation.Validate() status %q error = %v", status, err)
		}
	}

	observation := BenchmarkObservation{
		ID:                "observation-1",
		BenchmarkEntityID: "benchmark-1",
		ObservedAt:        time.Now(),
		Value:             "4.25",
		Unit:              "percent",
		SourceName:        "US Treasury",
		QualityStatus:     "estimated",
	}
	if err := observation.Validate(); err == nil {
		t.Fatal("BenchmarkObservation.Validate() error = nil, want invalid quality status error")
	}
}

func TestRawDocumentValidate(t *testing.T) {
	document := RawDocument{
		ID:            "raw-1",
		SourceID:      "source-1",
		IngestChannel: "rss_feed",
		SourceType:    "news",
		SourceName:    "示例来源",
		Title:         "示例标题",
		ContentHash:   "hash-1",
		CollectedAt:   time.Now(),
		IngestStatus:  IngestStatusCollected,
	}

	if err := document.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	document.ContentHash = ""
	if err := document.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want missing content hash error")
	}
}

func TestEventValidate(t *testing.T) {
	event := Event{
		ID:          "event-1",
		Title:       "示例事件",
		FirstSeenAt: time.Now(),
		EventStatus: EventStatusCandidate,
		FactStatus:  FactStatusUnverified,
		DedupeKey:   "event:demo",
	}

	if err := event.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	event.FactStatus = "certain"
	if err := event.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want invalid fact status error")
	}
}

func TestSchedulerConfigValidateIntervalMode(t *testing.T) {
	config := SchedulerConfig{
		ID:              "default",
		Enabled:         false,
		Mode:            SchedulerModeInterval,
		IntervalMinutes: 60,
		Concurrency:     2,
		BatchSize:       20,
		TimeoutSeconds:  180,
		Timezone:        "Asia/Shanghai",
		SourceFilter: SchedulerSourceFilter{
			ProviderKey:   "llm_web_research",
			IngestChannel: "ai_web_research",
			SourceType:    "news",
		},
	}

	if err := config.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	config.IntervalMinutes = 0
	if err := config.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want invalid interval error")
	}
}

func TestSchedulerConfigValidateFixedTimes(t *testing.T) {
	config := SchedulerConfig{
		ID:             "default",
		Enabled:        true,
		Mode:           SchedulerModeFixedTimes,
		FixedTimes:     []string{"09:00", "12:00", "15:00", "18:00", "21:00"},
		Concurrency:    1,
		BatchSize:      10,
		TimeoutSeconds: 180,
		Timezone:       "Asia/Shanghai",
	}

	if err := config.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	config.FixedTimes = []string{"09:00", "09:00"}
	if err := config.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want duplicate fixed time error")
	}
}

func TestSchedulerSourceFilterDoesNotExposeSingleSource(t *testing.T) {
	filterType := reflect.TypeOf(SchedulerSourceFilter{})
	if _, ok := filterType.FieldByName("SourceID"); ok {
		t.Fatal("SchedulerSourceFilter must not expose SourceID")
	}
}

func TestIngestionRunValidate(t *testing.T) {
	run := IngestionRun{
		ID:           "run-1",
		TriggerType:  SchedulerTriggerManualOnce,
		Status:       SchedulerRunStatusRunning,
		StartedAt:    time.Now(),
		TotalSources: 3,
	}

	if err := run.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	run.Status = "unknown"
	if err := run.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want invalid run status error")
	}
}

func TestIngestionRunSourceValidate(t *testing.T) {
	result := IngestionRunSource{
		ID:                 "run-source-1",
		RunID:              "run-1",
		SourceID:           "source-1",
		Status:             SchedulerSourceRunStatusSucceeded,
		DocumentsWritten:   2,
		DocumentsDuplicate: 1,
		StartedAt:          time.Now(),
		DurationMillis:     120,
	}

	if err := result.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	result.DocumentsWritten = -1
	if err := result.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want negative document count error")
	}
}

func TestSectorProfileAndSourceMappingValidateSemanticBoundaries(t *testing.T) {
	profile := SectorProfile{EntityID: "sector-id", ClassificationCode: SectorClassificationTheme, ReviewStatus: SectorReviewApproved}
	if err := profile.Validate(); err != nil {
		t.Fatalf("SectorProfile.Validate() error = %v", err)
	}
	profile.ClassificationCode = SectorClassification("index_sector")
	if err := profile.Validate(); err == nil {
		t.Fatal("SectorProfile.Validate() expected index_sector rejection")
	}
	mapping := SectorSourceMapping{
		ID: "mapping-id", SectorEntityID: "sector-id", SourceSystem: "ths",
		SourceTaxonomyType: SectorSourceTaxonomyIndexSector,
		SourceSectorName:   "人工智能", SourceSectorNameNormalized: "人工智能",
		MappingStatus: SectorSourceMappingApproved,
	}
	if err := mapping.Validate(); err != nil {
		t.Fatalf("SectorSourceMapping.Validate() error = %v", err)
	}
}
