-- +goose Up
CREATE TYPE ministry_agency_level AS ENUM (
    'CABINET_LEVEL',
    'SUB_CABINET',
    'INDEPENDENT_REGULATOR'
);

CREATE TYPE ministry_jurisdiction_scope AS ENUM (
    'FEDERAL',
    'STATE',
    'SUPRANATIONAL'
);

CREATE TYPE institution_type AS ENUM (
    'CENTRAL_BANK',
    'COMMERCIAL_BANK',
    'CLEARING_HOUSE',
    'PAYMENT_SYSTEM',
    'DEVELOPMENT_BANK',
    'INTERNATIONAL_FINANCIAL_INSTITUTION'
);

CREATE TYPE institution_systemic_importance AS ENUM (
    'G_SIB',
    'D_SIB',
    'NON_SIB'
);

CREATE TABLE ministries (
    id VARCHAR(39) PRIMARY KEY,
    code VARCHAR(30) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    name_en VARCHAR(100) NOT NULL,
    country_id VARCHAR(39),
    org_id VARCHAR(39),
    is_supranational BOOLEAN NOT NULL DEFAULT FALSE,
    parent_ministry_id VARCHAR(39),
    agency_level ministry_agency_level NOT NULL,
    has_sanction_power BOOLEAN NOT NULL,
    has_regulatory_power BOOLEAN NOT NULL,
    has_enforcement_power BOOLEAN NOT NULL,
    jurisdiction_scope ministry_jurisdiction_scope,
    domain_tags TEXT[],
    strategic_positioning TEXT,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_ministries_country
        FOREIGN KEY (country_id) REFERENCES countries(id) ON DELETE RESTRICT,
    CONSTRAINT fk_ministries_organization
        FOREIGN KEY (org_id) REFERENCES organizations(id) ON DELETE RESTRICT,
    CONSTRAINT fk_ministries_parent
        FOREIGN KEY (parent_ministry_id) REFERENCES ministries(id) ON DELETE RESTRICT,
    CONSTRAINT chk_ministries_identity CHECK (
        id ~ '^MIN[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    ),
    CONSTRAINT chk_ministries_required_text CHECK (
        btrim(code) <> '' AND btrim(name) <> '' AND btrim(name_en) <> ''
    ),
    CONSTRAINT chk_ministries_owner CHECK (
        (country_id IS NOT NULL AND org_id IS NULL AND is_supranational = FALSE) OR
        (country_id IS NULL AND org_id IS NOT NULL AND is_supranational = TRUE)
    )
);

CREATE INDEX idx_ministries_country
    ON ministries (country_id, code, id)
    WHERE country_id IS NOT NULL;
CREATE INDEX idx_ministries_organization
    ON ministries (org_id, code, id)
    WHERE org_id IS NOT NULL;
CREATE INDEX idx_ministries_parent
    ON ministries (parent_ministry_id, code, id)
    WHERE parent_ministry_id IS NOT NULL;

CREATE TABLE institutions (
    id VARCHAR(39) PRIMARY KEY,
    code VARCHAR(30) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    name_en VARCHAR(100) NOT NULL,
    country_id VARCHAR(39),
    org_id VARCHAR(39),
    is_supranational BOOLEAN NOT NULL DEFAULT FALSE,
    institution_type institution_type NOT NULL,
    clearing_currency CHAR(3),
    swift_bic CHAR(11),
    lei_code CHAR(20),
    systemic_importance institution_systemic_importance,
    strategic_positioning TEXT,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_institutions_country
        FOREIGN KEY (country_id) REFERENCES countries(id) ON DELETE RESTRICT,
    CONSTRAINT fk_institutions_organization
        FOREIGN KEY (org_id) REFERENCES organizations(id) ON DELETE RESTRICT,
    CONSTRAINT chk_institutions_identity CHECK (
        id ~ '^INS[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    ),
    CONSTRAINT chk_institutions_required_text CHECK (
        btrim(code) <> '' AND btrim(name) <> '' AND btrim(name_en) <> ''
    ),
    CONSTRAINT chk_institutions_owner CHECK (
        (country_id IS NOT NULL AND org_id IS NULL AND is_supranational = FALSE) OR
        (country_id IS NULL AND org_id IS NOT NULL AND is_supranational = TRUE)
    )
);

CREATE INDEX idx_institutions_country
    ON institutions (country_id, code, id)
    WHERE country_id IS NOT NULL;
CREATE INDEX idx_institutions_organization
    ON institutions (org_id, code, id)
    WHERE org_id IS NOT NULL;

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000063 is forward-only; restore a pre-migration database snapshot to roll back Ministry and Institution persistence';
END;
$$;
-- +goose StatementEnd
