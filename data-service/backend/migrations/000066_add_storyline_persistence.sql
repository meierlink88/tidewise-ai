-- +goose Up
CREATE TYPE storyline_type AS ENUM (
    'GEOPOLITICAL',
    'MACRO',
    'INDUSTRY',
    'CORPORATE'
);

CREATE TYPE storyline_status AS ENUM (
    'EMERGING',
    'ACTIVE',
    'DORMANT',
    'ARCHIVED'
);

CREATE TYPE storyline_data_alignment_status AS ENUM (
    'ALIGNED',
    'LAGGING',
    'ACCUMULATING',
    'DIVERGING',
    'NEW_FACTOR'
);

CREATE TABLE storylines (
    id VARCHAR(39) PRIMARY KEY,
    storyline_name VARCHAR(200) NOT NULL,
    storyline_type storyline_type NOT NULL,
    rivalry_id VARCHAR(39) REFERENCES geopolitic_rivalries(id) ON DELETE RESTRICT,
    macro_economic_id VARCHAR(39) REFERENCES macro_economics(id) ON DELETE RESTRICT,
    industry_chain_id VARCHAR(39) REFERENCES industry_chain(id) ON DELETE RESTRICT,
    company_entity_id VARCHAR(39) REFERENCES company_profiles(entity_id) ON DELETE RESTRICT,
    summary TEXT NOT NULL,
    current_stage VARCHAR(50) NOT NULL,
    status storyline_status NOT NULL DEFAULT 'EMERGING',
    confidence NUMERIC(3,2) NOT NULL,
    data_alignment_status storyline_data_alignment_status NOT NULL,
    data_alignment_score NUMERIC(3,2) NOT NULL,
    data_alignment_reason TEXT NOT NULL,
    last_alignment_checked_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_storylines_identity CHECK (
        id ~ '^STL[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    ),
    CONSTRAINT chk_storylines_required_text CHECK (
        btrim(storyline_name) <> '' AND btrim(summary) <> '' AND
        btrim(current_stage) <> '' AND btrim(data_alignment_reason) <> ''
    ),
    CONSTRAINT chk_storylines_confidence CHECK (confidence BETWEEN 0.00 AND 0.99),
    CONSTRAINT chk_storylines_alignment_score CHECK (data_alignment_score BETWEEN 0.00 AND 1.00),
    CONSTRAINT chk_storylines_anchor CHECK (
        (storyline_type = 'GEOPOLITICAL' AND rivalry_id IS NOT NULL AND
            macro_economic_id IS NULL AND industry_chain_id IS NULL AND company_entity_id IS NULL)
        OR
        (storyline_type = 'MACRO' AND rivalry_id IS NULL AND
            macro_economic_id IS NOT NULL AND industry_chain_id IS NULL AND company_entity_id IS NULL)
        OR
        (storyline_type = 'INDUSTRY' AND rivalry_id IS NULL AND
            macro_economic_id IS NULL AND industry_chain_id IS NOT NULL AND company_entity_id IS NULL)
        OR
        (storyline_type = 'CORPORATE' AND rivalry_id IS NULL AND
            macro_economic_id IS NULL AND industry_chain_id IS NULL AND company_entity_id IS NOT NULL)
    ),
    CONSTRAINT chk_storylines_timestamp_order CHECK (updated_at >= created_at)
);

CREATE INDEX idx_storylines_list ON storylines (storyline_name, id);
CREATE INDEX idx_storylines_type_list ON storylines (storyline_type, storyline_name, id);
CREATE INDEX idx_storylines_status_list ON storylines (status, storyline_name, id);

CREATE TABLE storyline_event_links (
    id VARCHAR(39) PRIMARY KEY,
    storyline_id VARCHAR(39) NOT NULL REFERENCES storylines(id) ON DELETE RESTRICT,
    event_id VARCHAR(39) NOT NULL REFERENCES events(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_storyline_event_links_identity CHECK (
        id ~ '^SLE[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    ),
    CONSTRAINT uq_storyline_event_links_endpoints UNIQUE (storyline_id, event_id)
);

CREATE INDEX idx_storyline_event_links_event_id ON storyline_event_links (event_id);

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000066 is forward-only; restore a pre-migration database snapshot to roll back Storyline persistence';
END;
$$;
-- +goose StatementEnd
