-- +goose Up
CREATE TABLE IF NOT EXISTS benchmark_profiles (
    entity_id UUID PRIMARY KEY REFERENCES entity_nodes(id),
    benchmark_type TEXT NOT NULL,
    official_series_code TEXT,
    provider TEXT NOT NULL,
    tenor TEXT,
    underlying_symbol TEXT,
    currency_code TEXT NOT NULL,
    unit TEXT NOT NULL,
    frequency TEXT NOT NULL,
    source_url TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_benchmark_profiles_benchmark_type CHECK (
        benchmark_type IN ('government_bond_yield', 'futures_price', 'spot_price', 'reference_rate')
    )
);

CREATE TABLE IF NOT EXISTS benchmark_observations (
    id UUID PRIMARY KEY,
    benchmark_entity_id UUID NOT NULL REFERENCES entity_nodes(id),
    observed_at TIMESTAMPTZ NOT NULL,
    value NUMERIC NOT NULL,
    unit TEXT NOT NULL,
    source_name TEXT NOT NULL,
    source_url TEXT NOT NULL DEFAULT '',
    external_series_code TEXT NOT NULL DEFAULT '',
    quality_status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_benchmark_observations_source_time UNIQUE (benchmark_entity_id, observed_at, source_name),
    CONSTRAINT chk_benchmark_observations_quality_status CHECK (
        quality_status IN ('raw', 'validated', 'suspect', 'rejected')
    )
);

CREATE INDEX IF NOT EXISTS idx_benchmark_observations_benchmark_time
    ON benchmark_observations (benchmark_entity_id, observed_at DESC);

-- +goose Down
SELECT 'benchmark foundation rollback requires a reviewed forward migration or restored backup';
