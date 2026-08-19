-- +goose Up
CREATE TYPE geopolitic_rivalry_type AS ENUM (
    'GEOPOLITICAL',
    'MILITARY_WAR'
);

CREATE TYPE geopolitic_rivalry_status AS ENUM (
    'ACTIVE',
    'DORMANT',
    'RESOLVED'
);

CREATE TABLE geopolitic_rivalries (
    id VARCHAR(39) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    name_en VARCHAR(100) NOT NULL,
    rivalry_type geopolitic_rivalry_type NOT NULL,
    description TEXT NOT NULL,
    core_actors TEXT NOT NULL,
    peripheral_actors TEXT,
    influenced_regions TEXT[],
    status geopolitic_rivalry_status NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_geopolitic_rivalries_identity CHECK (
        id ~ '^GPR[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    ),
    CONSTRAINT chk_geopolitic_rivalries_required_text CHECK (
        btrim(name) <> '' AND btrim(name_en) <> '' AND
        btrim(description) <> '' AND btrim(core_actors) <> ''
    )
);

CREATE INDEX idx_geopolitic_rivalries_list
    ON geopolitic_rivalries (name_en, name, id);
CREATE INDEX idx_geopolitic_rivalries_type_list
    ON geopolitic_rivalries (rivalry_type, name_en, name, id);
CREATE INDEX idx_geopolitic_rivalries_status_list
    ON geopolitic_rivalries (status, name_en, name, id);

CREATE TYPE macro_economic_type AS ENUM (
    'MONETARY',
    'FISCAL',
    'TRADE_POLICY',
    'REGULATORY',
    'DATA_ECONOMIC'
);

CREATE TYPE macro_economic_status AS ENUM (
    'ACTIVE',
    'DORMANT',
    'ARCHIVED'
);

CREATE TABLE macro_economics (
    id VARCHAR(39) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    name_en VARCHAR(100) NOT NULL,
    macro_type macro_economic_type NOT NULL,
    description TEXT NOT NULL,
    status macro_economic_status NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_macro_economics_identity CHECK (
        id ~ '^MEC[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    ),
    CONSTRAINT chk_macro_economics_required_text CHECK (
        btrim(name) <> '' AND btrim(name_en) <> '' AND btrim(description) <> ''
    )
);

CREATE INDEX idx_macro_economics_list
    ON macro_economics (name_en, name, id);
CREATE INDEX idx_macro_economics_type_list
    ON macro_economics (macro_type, name_en, name, id);
CREATE INDEX idx_macro_economics_status_list
    ON macro_economics (status, name_en, name, id);

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000064 is forward-only; restore a pre-migration database snapshot to roll back narrative blueprint persistence';
END;
$$;
-- +goose StatementEnd
