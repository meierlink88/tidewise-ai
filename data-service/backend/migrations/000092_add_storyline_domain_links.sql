-- +goose Up
SET LOCAL lock_timeout = '5s';

CREATE TABLE geopolitic_rivalry_domain_links (
    id VARCHAR(39) PRIMARY KEY CHECK (id ~ '^GRD[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
    geopolitic_rivalry_id VARCHAR(39) NOT NULL REFERENCES geopolitic_rivalries(id) ON DELETE RESTRICT,
    geopolitic_domain_id VARCHAR(39) NOT NULL REFERENCES geopolitic_domains(id) ON DELETE RESTRICT,
    UNIQUE (geopolitic_rivalry_id, geopolitic_domain_id)
);
CREATE INDEX idx_geopolitic_rivalry_domain_links_domain ON geopolitic_rivalry_domain_links (geopolitic_domain_id, geopolitic_rivalry_id);

CREATE TABLE macro_economic_domain_links (
    id VARCHAR(39) PRIMARY KEY CHECK (id ~ '^MED[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
    macro_economic_id VARCHAR(39) NOT NULL REFERENCES macro_economics(id) ON DELETE RESTRICT,
    macro_economic_domain_id VARCHAR(39) NOT NULL REFERENCES macro_economics_domain(id) ON DELETE RESTRICT,
    UNIQUE (macro_economic_id, macro_economic_domain_id)
);
CREATE INDEX idx_macro_economic_domain_links_domain ON macro_economic_domain_links (macro_economic_domain_id, macro_economic_id);

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION 'storyline domain migration is forward-only; restore a reviewed backup with its matching application' USING ERRCODE = '55000';
END;
$$;
-- +goose StatementEnd
