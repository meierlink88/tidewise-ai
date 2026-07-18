package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
)

type BenchmarkObservationRepository interface {
	UpsertBenchmarkObservation(context.Context, domain.BenchmarkObservation) (BenchmarkObservationWriteResult, error)
	ListBenchmarkObservations(context.Context, BenchmarkObservationFilter) ([]domain.BenchmarkObservation, error)
}

type BenchmarkObservationWriteResult struct {
	Observation domain.BenchmarkObservation
	Created     bool
}

type BenchmarkObservationFilter struct {
	BenchmarkEntityID string
	Limit             int
}

func (r *InMemoryRepository) UpsertBenchmarkObservation(_ context.Context, observation domain.BenchmarkObservation) (BenchmarkObservationWriteResult, error) {
	if err := observation.Validate(); err != nil {
		return BenchmarkObservationWriteResult{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	entity, ok := r.graphEntities[observation.BenchmarkEntityID]
	if !ok {
		return BenchmarkObservationWriteResult{}, fmt.Errorf("benchmark entity %q not found", observation.BenchmarkEntityID)
	}
	if entity.EntityType != domain.EntityTypeBenchmark {
		return BenchmarkObservationWriteResult{}, fmt.Errorf("entity %q type %q is not benchmark", observation.BenchmarkEntityID, entity.EntityType)
	}

	key := benchmarkObservationKey(observation.BenchmarkEntityID, observation.ObservedAt, observation.SourceName)
	existing, ok := r.observations[key]
	if ok {
		observation.ID = existing.ID
		r.observations[key] = observation
		return BenchmarkObservationWriteResult{Observation: observation, Created: false}, nil
	}
	r.observations[key] = observation
	return BenchmarkObservationWriteResult{Observation: observation, Created: true}, nil
}

func (r *InMemoryRepository) ListBenchmarkObservations(_ context.Context, filter BenchmarkObservationFilter) ([]domain.BenchmarkObservation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	observations := make([]domain.BenchmarkObservation, 0, len(r.observations))
	for _, observation := range r.observations {
		if filter.BenchmarkEntityID != "" && observation.BenchmarkEntityID != filter.BenchmarkEntityID {
			continue
		}
		observations = append(observations, observation)
	}
	sort.SliceStable(observations, func(i, j int) bool {
		if !observations[i].ObservedAt.Equal(observations[j].ObservedAt) {
			return observations[i].ObservedAt.After(observations[j].ObservedAt)
		}
		return observations[i].SourceName < observations[j].SourceName
	})
	if filter.Limit > 0 && len(observations) > filter.Limit {
		observations = observations[:filter.Limit]
	}
	return observations, nil
}

func benchmarkObservationKey(benchmarkEntityID string, observedAt time.Time, sourceName string) string {
	return strings.Join([]string{benchmarkEntityID, observedAt.UTC().Format(time.RFC3339Nano), strings.ToLower(strings.TrimSpace(sourceName))}, "|")
}

func (r PostgresRepository) UpsertBenchmarkObservation(ctx context.Context, observation domain.BenchmarkObservation) (BenchmarkObservationWriteResult, error) {
	if err := observation.Validate(); err != nil {
		return BenchmarkObservationWriteResult{}, err
	}
	if err := r.ensureBenchmarkEntity(ctx, observation.BenchmarkEntityID); err != nil {
		return BenchmarkObservationWriteResult{}, err
	}

	row := r.db.QueryRowContext(ctx, `
WITH upsert AS (
    INSERT INTO benchmark_observations (
        id, benchmark_entity_id, observed_at, value, unit, source_name,
        source_url, external_series_code, quality_status
    ) VALUES (
        $1, $2, $3, $4::numeric, $5, $6,
        $7, $8, $9
    )
    ON CONFLICT (benchmark_entity_id, observed_at, source_name) DO UPDATE SET
        value = EXCLUDED.value,
        unit = EXCLUDED.unit,
        source_url = EXCLUDED.source_url,
        external_series_code = EXCLUDED.external_series_code,
        quality_status = EXCLUDED.quality_status,
        updated_at = now()
    RETURNING id, benchmark_entity_id, observed_at, value::text, unit, source_name,
              source_url, external_series_code, quality_status, xmax = 0 AS inserted
)
SELECT id, benchmark_entity_id, observed_at, value, unit, source_name,
       source_url, external_series_code, quality_status, inserted
FROM upsert
`, NormalizeUUID(observation.ID), NormalizeUUID(observation.BenchmarkEntityID), observation.ObservedAt, observation.Value, observation.Unit, observation.SourceName,
		observation.SourceURL, observation.ExternalSeriesCode, observation.QualityStatus)

	saved, created, err := scanBenchmarkObservationWrite(row)
	if err != nil {
		return BenchmarkObservationWriteResult{}, fmt.Errorf("upsert benchmark observation: %w", err)
	}
	return BenchmarkObservationWriteResult{Observation: saved, Created: created}, nil
}

func (r PostgresRepository) ListBenchmarkObservations(ctx context.Context, filter BenchmarkObservationFilter) ([]domain.BenchmarkObservation, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, benchmark_entity_id, observed_at, value::text, unit, source_name,
       source_url, external_series_code, quality_status
FROM benchmark_observations
WHERE ($1::uuid IS NULL OR benchmark_entity_id = $1::uuid)
ORDER BY observed_at DESC, source_name, id
LIMIT CASE WHEN $2 > 0 THEN $2 ELSE 2147483647 END
`, benchmarkObservationFilterEntityID(filter.BenchmarkEntityID), filter.Limit)
	if err != nil {
		return nil, fmt.Errorf("query benchmark observations: %w", err)
	}
	defer rows.Close()

	observations := make([]domain.BenchmarkObservation, 0)
	for rows.Next() {
		observation, err := scanBenchmarkObservation(rows)
		if err != nil {
			return nil, err
		}
		observations = append(observations, observation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate benchmark observations: %w", err)
	}
	return observations, nil
}

func (r PostgresRepository) ensureBenchmarkEntity(ctx context.Context, entityID string) error {
	var entityType domain.EntityType
	if err := r.db.QueryRowContext(ctx, `
SELECT entity_type
FROM entity_nodes
WHERE id = $1
`, NormalizeUUID(entityID)).Scan(&entityType); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("benchmark entity %q not found", entityID)
		}
		return fmt.Errorf("query benchmark entity %q: %w", entityID, err)
	}
	if entityType != domain.EntityTypeBenchmark {
		return fmt.Errorf("entity %q type %q is not benchmark", entityID, entityType)
	}
	return nil
}

func scanBenchmarkObservation(scanner rawDocumentScanner) (domain.BenchmarkObservation, error) {
	var observation domain.BenchmarkObservation
	if err := scanner.Scan(
		&observation.ID,
		&observation.BenchmarkEntityID,
		&observation.ObservedAt,
		&observation.Value,
		&observation.Unit,
		&observation.SourceName,
		&observation.SourceURL,
		&observation.ExternalSeriesCode,
		&observation.QualityStatus,
	); err != nil {
		return domain.BenchmarkObservation{}, fmt.Errorf("scan benchmark observation: %w", err)
	}
	return observation, nil
}

func scanBenchmarkObservationWrite(scanner rawDocumentScanner) (domain.BenchmarkObservation, bool, error) {
	var observation domain.BenchmarkObservation
	var created bool
	if err := scanner.Scan(
		&observation.ID,
		&observation.BenchmarkEntityID,
		&observation.ObservedAt,
		&observation.Value,
		&observation.Unit,
		&observation.SourceName,
		&observation.SourceURL,
		&observation.ExternalSeriesCode,
		&observation.QualityStatus,
		&created,
	); err != nil {
		return domain.BenchmarkObservation{}, false, fmt.Errorf("scan benchmark observation write: %w", err)
	}
	return observation, created, nil
}

func benchmarkObservationFilterEntityID(value string) any {
	if value == "" {
		return nil
	}
	return NormalizeUUID(value)
}
