-- +goose Up
ALTER TABLE sector_profiles
    ADD COLUMN IF NOT EXISTS classification_code TEXT NOT NULL DEFAULT 'market_sector',
    ADD COLUMN IF NOT EXISTS primary_market_entity_id UUID REFERENCES entity_nodes(id),
    ADD COLUMN IF NOT EXISTS primary_economy_entity_id UUID REFERENCES entity_nodes(id),
    ADD COLUMN IF NOT EXISTS methodology_url TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS review_status TEXT NOT NULL DEFAULT 'candidate';

ALTER TABLE sector_profiles
    ADD CONSTRAINT chk_sector_profiles_classification_code CHECK (
        classification_code IN ('industry_sector', 'theme_sector', 'market_sector', 'style_sector', 'region_sector')
    ),
    ADD CONSTRAINT chk_sector_profiles_review_status CHECK (
        review_status IN ('candidate', 'approved', 'rejected')
    );

CREATE TABLE IF NOT EXISTS sector_source_mappings (
    id UUID PRIMARY KEY,
    sector_entity_id UUID NOT NULL REFERENCES entity_nodes(id),
    source_system TEXT NOT NULL,
    source_taxonomy_type TEXT NOT NULL,
    source_sector_code TEXT NOT NULL DEFAULT '',
    source_sector_name TEXT NOT NULL,
    source_sector_name_normalized TEXT NOT NULL,
    source_market_scope TEXT NOT NULL DEFAULT '',
    source_url TEXT NOT NULL DEFAULT '',
    rank_snapshot INTEGER NOT NULL DEFAULT 0,
    snapshot_date DATE,
    mapping_status TEXT NOT NULL DEFAULT 'candidate',
    review_note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_sector_source_mappings_taxonomy CHECK (
        source_taxonomy_type IN ('concept', 'industry', 'index_sector')
    ),
    CONSTRAINT chk_sector_source_mappings_status CHECK (
        mapping_status IN ('candidate', 'approved', 'rejected', 'merged')
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_sector_source_mappings_code
    ON sector_source_mappings (source_system, source_taxonomy_type, source_sector_code)
    WHERE source_sector_code <> '';

CREATE UNIQUE INDEX IF NOT EXISTS uq_sector_source_mappings_name_scope
    ON sector_source_mappings (source_system, source_taxonomy_type, source_sector_name_normalized, source_market_scope)
    WHERE source_sector_code = '';

CREATE INDEX IF NOT EXISTS idx_sector_source_mappings_sector_entity_id
    ON sector_source_mappings (sector_entity_id);

-- +goose Down
SELECT 'market sector foundation rollback requires a reviewed forward migration or restored backup';
