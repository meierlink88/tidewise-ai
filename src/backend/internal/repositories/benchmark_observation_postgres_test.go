package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func TestPostgresRepositoryBenchmarkObservationIntegration(t *testing.T) {
	dsn := os.Getenv("TIDEWISE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TIDEWISE_TEST_DATABASE_URL to run PostgreSQL benchmark observation repository integration test")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close db: %v", err)
		}
	})

	ctx := context.Background()
	repo := NewPostgresRepository(db)
	runID := fmt.Sprintf("benchmark-observation-integration-%d", time.Now().UnixNano())
	benchmarkID := NormalizeUUID(runID, "benchmark")
	otherBenchmarkID := NormalizeUUID(runID, "other-benchmark")
	indexID := NormalizeUUID(runID, "index")
	entityIDs := []string{benchmarkID, otherBenchmarkID, indexID}
	t.Cleanup(func() {
		if _, err := db.ExecContext(context.Background(), `DELETE FROM benchmark_observations WHERE benchmark_entity_id = ANY($1::uuid[])`, entityIDs); err != nil {
			t.Errorf("cleanup benchmark observations: %v", err)
		}
		if _, err := db.ExecContext(context.Background(), `DELETE FROM entity_nodes WHERE id = ANY($1::uuid[])`, entityIDs); err != nil {
			t.Errorf("cleanup benchmark entities: %v", err)
		}
	})

	if _, err := db.ExecContext(ctx, `
INSERT INTO entity_nodes (
    id, entity_key, entity_type, layer_code, name, canonical_name, aliases, status
) VALUES
    ($1, $2, 'benchmark', 'market', 'Benchmark A', 'Benchmark A', '{}'::text[], 'active'),
    ($3, $4, 'benchmark', 'market', 'Benchmark B', 'Benchmark B', '{}'::text[], 'active'),
    ($5, $6, 'index', 'market', 'Index A', 'Index A', '{}'::text[], 'active')
`, benchmarkID, runID+":benchmark-a", otherBenchmarkID, runID+":benchmark-b", indexID, runID+":index-a"); err != nil {
		t.Fatalf("insert benchmark observation entities: %v", err)
	}

	observedAt := time.Date(2026, 7, 12, 9, 30, 0, 0, time.UTC)
	first, err := repo.UpsertBenchmarkObservation(ctx, domain.BenchmarkObservation{
		ID:                 runID + "-observation-first",
		BenchmarkEntityID:  benchmarkID,
		ObservedAt:         observedAt,
		Value:              "4.25",
		Unit:               "percent",
		SourceName:         runID + "-source-a",
		SourceURL:          "https://example.com/source-a",
		ExternalSeriesCode: "SERIES-A",
		QualityStatus:      domain.BenchmarkObservationQualityRaw,
	})
	if err != nil {
		t.Fatalf("UpsertBenchmarkObservation(first) error = %v", err)
	}
	if !first.Created {
		t.Fatal("first UpsertBenchmarkObservation() should create a row")
	}

	updated, err := repo.UpsertBenchmarkObservation(ctx, domain.BenchmarkObservation{
		ID:                 runID + "-observation-retry",
		BenchmarkEntityID:  benchmarkID,
		ObservedAt:         observedAt,
		Value:              "4.30",
		Unit:               "percent",
		SourceName:         runID + "-source-a",
		SourceURL:          "https://example.com/source-a-updated",
		ExternalSeriesCode: "SERIES-A",
		QualityStatus:      domain.BenchmarkObservationQualityValidated,
	})
	if err != nil {
		t.Fatalf("UpsertBenchmarkObservation(conflict) error = %v", err)
	}
	if updated.Created {
		t.Fatal("same benchmark/time/source should update the existing row")
	}
	if updated.Observation.ID != first.Observation.ID {
		t.Fatalf("updated observation ID = %q, want original ID %q", updated.Observation.ID, first.Observation.ID)
	}
	if updated.Observation.Value != "4.30" || updated.Observation.QualityStatus != domain.BenchmarkObservationQualityValidated {
		t.Fatalf("updated observation = %+v, want updated value and quality status", updated.Observation)
	}

	for _, observation := range []domain.BenchmarkObservation{
		{
			ID:                runID + "-observation-source-b",
			BenchmarkEntityID: benchmarkID,
			ObservedAt:        observedAt,
			Value:             "4.31",
			Unit:              "percent",
			SourceName:        runID + "-source-b",
			SourceURL:         "https://example.com/source-b",
			QualityStatus:     domain.BenchmarkObservationQualityRaw,
		},
		{
			ID:                runID + "-observation-latest",
			BenchmarkEntityID: benchmarkID,
			ObservedAt:        observedAt.Add(time.Hour),
			Value:             "4.32",
			Unit:              "percent",
			SourceName:        runID + "-source-a",
			SourceURL:         "https://example.com/source-a",
			QualityStatus:     domain.BenchmarkObservationQualityRaw,
		},
		{
			ID:                runID + "-observation-other-benchmark",
			BenchmarkEntityID: otherBenchmarkID,
			ObservedAt:        observedAt.Add(2 * time.Hour),
			Value:             "3.80",
			Unit:              "percent",
			SourceName:        runID + "-source-a",
			SourceURL:         "https://example.com/source-a",
			QualityStatus:     domain.BenchmarkObservationQualityRaw,
		},
	} {
		result, err := repo.UpsertBenchmarkObservation(ctx, observation)
		if err != nil {
			t.Fatalf("UpsertBenchmarkObservation(%s) error = %v", observation.ID, err)
		}
		if !result.Created {
			t.Fatalf("UpsertBenchmarkObservation(%s) should create a distinct row", observation.ID)
		}
	}

	filtered, err := repo.ListBenchmarkObservations(ctx, BenchmarkObservationFilter{BenchmarkEntityID: benchmarkID})
	if err != nil {
		t.Fatalf("ListBenchmarkObservations(filtered) error = %v", err)
	}
	if got, want := len(filtered), 3; got != want {
		t.Fatalf("filtered observations length = %d, want %d", got, want)
	}
	if !filtered[0].ObservedAt.Equal(observedAt.Add(time.Hour)) {
		t.Fatalf("first filtered observed_at = %s, want latest %s", filtered[0].ObservedAt, observedAt.Add(time.Hour))
	}
	if filtered[1].SourceName == filtered[2].SourceName {
		t.Fatalf("same-time observations should preserve different sources: %+v", filtered)
	}

	all, err := repo.ListBenchmarkObservations(ctx, BenchmarkObservationFilter{})
	if err != nil {
		t.Fatalf("ListBenchmarkObservations(empty filter) error = %v", err)
	}
	positions := map[string]int{}
	for index, observation := range all {
		if observation.BenchmarkEntityID == benchmarkID || observation.BenchmarkEntityID == otherBenchmarkID {
			positions[observation.ID] = index
		}
	}
	if got, want := len(positions), 4; got != want {
		t.Fatalf("empty filter returned %d integration observations, want %d", got, want)
	}
	if positions[NormalizeUUID(runID+"-observation-other-benchmark")] >= positions[NormalizeUUID(runID+"-observation-latest")] {
		t.Fatalf("empty-filter observations are not ordered by observed_at descending: %+v", all)
	}

	_, err = repo.UpsertBenchmarkObservation(ctx, domain.BenchmarkObservation{
		ID:                runID + "-observation-index",
		BenchmarkEntityID: indexID,
		ObservedAt:        observedAt,
		Value:             "20",
		Unit:              "points",
		SourceName:        runID + "-source-a",
		QualityStatus:     domain.BenchmarkObservationQualityRaw,
	})
	if err == nil {
		t.Fatal("UpsertBenchmarkObservation(index) error = nil, want non-benchmark entity rejection")
	}
}
