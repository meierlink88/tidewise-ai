package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
)

func TestInMemoryRepositoryUpsertsBenchmarkObservationsIdempotently(t *testing.T) {
	repo := NewInMemoryRepository()
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
	repo := NewInMemoryRepository()
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
	repo := NewInMemoryRepository()
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
