package repositories

import (
	"context"
	"reflect"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

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
