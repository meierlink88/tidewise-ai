-- +goose Up
DROP TABLE entity_type_definitions;

CREATE TYPE region_type AS ENUM (
    'CONTINENT',
    'GEOGRAPHIC',
    'MULTILATERAL',
    'INVESTMENT'
);

CREATE TABLE regions (
    id VARCHAR(32) PRIMARY KEY,
    code VARCHAR(20) NOT NULL UNIQUE,
    name VARCHAR(50) NOT NULL,
    name_en VARCHAR(100) NOT NULL,
    region_type region_type NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_regions_code CHECK (
        code ~ '^[A-Z][A-Z0-9_]*$'
    ),
    CONSTRAINT chk_regions_identity CHECK (
        id = 'REG_' || code
    ),
    CONSTRAINT chk_regions_names CHECK (
        btrim(name) <> '' AND btrim(name_en) <> ''
    )
);

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000045 is forward-only; restore a pre-cutover database snapshot to roll back Tidewise AI 2.0';
END;
$$;
-- +goose StatementEnd
