-- +goose Up
CREATE TYPE storyline_domain_category AS ENUM (
    'GEOPOLITICAL',
    'MACRO',
    'INDUSTRY',
    'CORPORATE'
);

CREATE TABLE storyline_domains (
    id VARCHAR(39) PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    name_en VARCHAR(50) NOT NULL,
    description TEXT NOT NULL,
    scope_definition TEXT NOT NULL,
    domain_category storyline_domain_category NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_storyline_domains_identity CHECK (
        id ~ '^SLD[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    ),
    CONSTRAINT chk_storyline_domains_required_text CHECK (
        btrim(name) <> '' AND btrim(name_en) <> '' AND
        btrim(description) <> '' AND btrim(scope_definition) <> ''
    )
);

CREATE INDEX idx_storyline_domains_list
    ON storyline_domains (name_en, name, id);
CREATE INDEX idx_storyline_domains_category_list
    ON storyline_domains (domain_category, name_en, name, id);
CREATE INDEX idx_storyline_domains_active_list
    ON storyline_domains (is_active, name_en, name, id);

CREATE TABLE storyline_domain_tactics (
    id VARCHAR(39) PRIMARY KEY,
    key VARCHAR(30) NOT NULL UNIQUE,
    name VARCHAR(50) NOT NULL,
    name_en VARCHAR(50) NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_storyline_domain_tactics_identity CHECK (
        id ~ '^SDT[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    ),
    CONSTRAINT chk_storyline_domain_tactics_key CHECK (
        key ~ '^[A-Z][A-Z0-9_]{0,29}$'
    ),
    CONSTRAINT chk_storyline_domain_tactics_required_text CHECK (
        btrim(name) <> '' AND btrim(name_en) <> '' AND btrim(description) <> ''
    )
);

CREATE INDEX idx_storyline_domain_tactics_list
    ON storyline_domain_tactics (key, id);

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000065 is forward-only; restore a pre-migration database snapshot to roll back StorylineDomain and StorylineDomainTactic persistence';
END;
$$;
-- +goose StatementEnd
