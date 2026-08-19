-- +goose Up
CREATE TYPE subdivision_type AS ENUM (
    'PROVINCE',
    'STATE',
    'SAR',
    'TERRITORY'
);

CREATE TABLE subdivisions (
    id VARCHAR(39) PRIMARY KEY,
    code VARCHAR(10) NOT NULL,
    name VARCHAR(100) NOT NULL,
    name_en VARCHAR(100) NOT NULL,
    country_id VARCHAR(39) NOT NULL REFERENCES countries(id) ON DELETE RESTRICT,
    subdivision_type subdivision_type NOT NULL,
    strategic_positioning TEXT,
    key_resources TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_subdivisions_country_code UNIQUE (country_id, code),
    CONSTRAINT chk_subdivisions_identity CHECK (
        id ~ '^SUB[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    ),
    CONSTRAINT chk_subdivisions_code CHECK (
        code ~ '^[A-Z0-9]+$'
    ),
    CONSTRAINT chk_subdivisions_names CHECK (
        btrim(name) <> '' AND btrim(name_en) <> ''
    ),
    CONSTRAINT chk_subdivisions_optional_text CHECK (
        (strategic_positioning IS NULL OR btrim(strategic_positioning) <> '') AND
        (key_resources IS NULL OR btrim(key_resources) <> '')
    )
);

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000062 is forward-only; restore a pre-migration database snapshot to roll back Subdivision persistence';
END;
$$;
-- +goose StatementEnd
