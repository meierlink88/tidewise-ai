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

func TestAllianceOrgProfileValidate(t *testing.T) {
	profile := AllianceOrgProfile{
		EntityID:      "entity-1",
		OrgCode:       "OPEC_PLUS",
		OrgType:       "energy_alliance",
		PrimaryDomain: "energy",
		ScopeRegion:   "global",
		OfficialURL:   "https://www.opec.org",
	}

	if err := profile.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	profile.OrgCode = ""
	if err := profile.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want missing org code error")
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
		ID:                "run-source-1",
		RunID:             "run-1",
		SourceID:          "source-1",
		Status:            SchedulerSourceRunStatusSucceeded,
		DocumentsWritten:  2,
		DocumentsDuplicate: 1,
		StartedAt:         time.Now(),
		DurationMillis:    120,
	}

	if err := result.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	result.DocumentsWritten = -1
	if err := result.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want negative document count error")
	}
}
