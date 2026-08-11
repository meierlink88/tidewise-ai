package entity

import (
	"context"
	"database/sql"
	"fmt"

	biz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/entity"
	bizidentity "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/identity"
)

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, fmt.Errorf("Entity database is required")
	}
	return &Store{db: db}, nil
}

func (s *Store) UpsertBenchmarkObservation(ctx context.Context, observation biz.BenchmarkObservation) (biz.BenchmarkObservationWriteResult, error) {
	if err := observation.Validate(); err != nil {
		return biz.BenchmarkObservationWriteResult{}, err
	}
	if err := s.ensureBenchmarkEntity(ctx, observation.BenchmarkEntityID); err != nil {
		return biz.BenchmarkObservationWriteResult{}, err
	}
	row := s.db.QueryRowContext(ctx, `WITH upsert AS (INSERT INTO benchmark_observations (id, benchmark_entity_id, observed_at, value, unit, source_name, source_url, external_series_code, quality_status) VALUES ($1,$2,$3,$4::numeric,$5,$6,$7,$8,$9) ON CONFLICT (benchmark_entity_id, observed_at, source_name) DO UPDATE SET value=EXCLUDED.value, unit=EXCLUDED.unit, source_url=EXCLUDED.source_url, external_series_code=EXCLUDED.external_series_code, quality_status=EXCLUDED.quality_status, updated_at=now() RETURNING id, benchmark_entity_id, observed_at, value::text, unit, source_name, source_url, external_series_code, quality_status, xmax=0 AS inserted) SELECT id, benchmark_entity_id, observed_at, value, unit, source_name, source_url, external_series_code, quality_status, inserted FROM upsert`, bizidentity.NormalizeUUID(observation.ID), bizidentity.NormalizeUUID(observation.BenchmarkEntityID), observation.ObservedAt, observation.Value, observation.Unit, observation.SourceName, observation.SourceURL, observation.ExternalSeriesCode, observation.QualityStatus)
	saved, created, err := scanBenchmarkObservationWrite(row)
	if err != nil {
		return biz.BenchmarkObservationWriteResult{}, fmt.Errorf("upsert benchmark observation: %w", err)
	}
	return biz.BenchmarkObservationWriteResult{Observation: saved, Created: created}, nil
}

func (s *Store) ListBenchmarkObservations(ctx context.Context, filter biz.BenchmarkObservationFilter) ([]biz.BenchmarkObservation, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, benchmark_entity_id, observed_at, value::text, unit, source_name, source_url, external_series_code, quality_status FROM benchmark_observations WHERE ($1::uuid IS NULL OR benchmark_entity_id=$1::uuid) ORDER BY observed_at DESC, source_name, id LIMIT CASE WHEN $2 > 0 THEN $2 ELSE 2147483647 END`, optionalUUID(filter.BenchmarkEntityID), filter.Limit)
	if err != nil {
		return nil, fmt.Errorf("query benchmark observations: %w", err)
	}
	defer rows.Close()
	items := make([]biz.BenchmarkObservation, 0)
	for rows.Next() {
		item, err := scanBenchmarkObservation(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate benchmark observations: %w", err)
	}
	return items, nil
}

func (s *Store) ensureBenchmarkEntity(ctx context.Context, id string) error {
	var entityType biz.EntityType
	if err := s.db.QueryRowContext(ctx, `SELECT entity_type FROM entity_nodes WHERE id=$1`, bizidentity.NormalizeUUID(id)).Scan(&entityType); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("benchmark entity %q not found", id)
		}
		return fmt.Errorf("query benchmark entity %q: %w", id, err)
	}
	if entityType != biz.EntityTypeBenchmark {
		return fmt.Errorf("entity %q type %q is not benchmark", id, entityType)
	}
	return nil
}

type scanner interface{ Scan(...any) error }

func scanBenchmarkObservation(row scanner) (biz.BenchmarkObservation, error) {
	var item biz.BenchmarkObservation
	if err := row.Scan(&item.ID, &item.BenchmarkEntityID, &item.ObservedAt, &item.Value, &item.Unit, &item.SourceName, &item.SourceURL, &item.ExternalSeriesCode, &item.QualityStatus); err != nil {
		return biz.BenchmarkObservation{}, fmt.Errorf("scan benchmark observation: %w", err)
	}
	if err := item.Validate(); err != nil {
		return biz.BenchmarkObservation{}, fmt.Errorf("validate persisted benchmark observation: %w", err)
	}
	return item, nil
}
func scanBenchmarkObservationWrite(row scanner) (biz.BenchmarkObservation, bool, error) {
	var item biz.BenchmarkObservation
	var created bool
	if err := row.Scan(&item.ID, &item.BenchmarkEntityID, &item.ObservedAt, &item.Value, &item.Unit, &item.SourceName, &item.SourceURL, &item.ExternalSeriesCode, &item.QualityStatus, &created); err != nil {
		return biz.BenchmarkObservation{}, false, err
	}
	if err := item.Validate(); err != nil {
		return biz.BenchmarkObservation{}, false, fmt.Errorf("validate persisted benchmark observation: %w", err)
	}
	return item, created, nil
}
func optionalUUID(value string) any {
	if value == "" {
		return nil
	}
	return bizidentity.NormalizeUUID(value)
}
