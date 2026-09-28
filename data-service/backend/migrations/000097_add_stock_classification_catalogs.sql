-- +goose Up
-- Isolated stock taxonomies retain original column names and types. Missing
-- research definitions remain NULL; this migration never copies business data.
CREATE TABLE s_industry (LIKE industry INCLUDING DEFAULTS INCLUDING STORAGE INCLUDING COMMENTS);
ALTER TABLE s_industry
 ALTER COLUMN id TYPE VARCHAR(40),
 ALTER COLUMN parent_industry_id TYPE VARCHAR(40),
 ALTER COLUMN definition DROP NOT NULL,
 ADD PRIMARY KEY (id),
 ADD CHECK (id ~ '^SIND[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
 ADD CHECK (btrim(name)<>'' AND btrim(classification_system)<>'' AND btrim(industry_code)<>''),
 ADD CHECK (valid_independent_object_text_set(aliases,TRUE)),
 ADD CHECK (definition IS NULL OR btrim(definition)<>''),
 ADD CHECK (review_status IN ('candidate','approved')),
 ADD CHECK (review_status<>'approved' OR definition IS NOT NULL),
 ADD CHECK (updated_at>=created_at),
 ADD CHECK (valid_independent_object_text_set(hierarchy_path_codes,FALSE)),
 ADD CHECK (hierarchy_path_codes[cardinality(hierarchy_path_codes)]=industry_code),
 ADD CHECK ((parent_industry_id IS NULL AND cardinality(hierarchy_path_codes)=1) OR (parent_industry_id IS NOT NULL AND cardinality(hierarchy_path_codes)=2)),
 ADD UNIQUE (classification_system,industry_code),
 ADD UNIQUE (classification_system,hierarchy_path_codes),
 ADD FOREIGN KEY (parent_industry_id) REFERENCES s_industry(id) ON DELETE RESTRICT;
CREATE INDEX idx_s_industry_parent ON s_industry(parent_industry_id);
-- +goose StatementBegin
CREATE FUNCTION validate_s_industry_parent() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.parent_industry_id IS NOT NULL AND NOT EXISTS (
   SELECT 1 FROM s_industry p WHERE p.id=NEW.parent_industry_id
   AND p.parent_industry_id IS NULL AND p.classification_system=NEW.classification_system
   AND NEW.hierarchy_path_codes=p.hierarchy_path_codes||NEW.industry_code
 ) THEN RAISE EXCEPTION 's_industry parent/path mismatch'; END IF;
 IF EXISTS (SELECT 1 FROM s_industry c WHERE c.parent_industry_id=NEW.id
   AND (c.classification_system<>NEW.classification_system OR c.hierarchy_path_codes<>NEW.hierarchy_path_codes||c.industry_code))
 THEN RAISE EXCEPTION 's_industry child/path mismatch'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER trg_s_industry_parent BEFORE INSERT OR UPDATE ON s_industry FOR EACH ROW EXECUTE FUNCTION validate_s_industry_parent();

CREATE TABLE s_concept (LIKE concept INCLUDING DEFAULTS INCLUDING STORAGE INCLUDING COMMENTS);
ALTER TABLE s_concept
 ALTER COLUMN id TYPE VARCHAR(40),
 ALTER COLUMN definition DROP NOT NULL,
 ADD PRIMARY KEY(id),
 ADD UNIQUE(name),
 ADD CHECK (id ~ '^SCON[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
 ADD CHECK (btrim(name)<>''),
 ADD CHECK (valid_independent_object_text_set(aliases,TRUE)),
 ADD CHECK (definition IS NULL OR btrim(definition)<>''),
 ADD CHECK (concept_type IN ('technology','policy','application','demand','business_model','company_ecosystem','product_ecosystem','event_narrative','market_theme')),
 ADD CHECK (review_status IN ('candidate','approved')),
 ADD CHECK (review_status<>'approved' OR definition IS NOT NULL),
 ADD CHECK (updated_at>=created_at);

CREATE TABLE s_industry_chain (LIKE industry_chain INCLUDING DEFAULTS INCLUDING STORAGE INCLUDING COMMENTS);
ALTER TABLE s_industry_chain
 ALTER COLUMN id TYPE VARCHAR(40),
 ALTER COLUMN scope DROP NOT NULL,
 ALTER COLUMN target_output DROP NOT NULL,
 ALTER COLUMN end_use DROP NOT NULL,
 ALTER COLUMN geography DROP NOT NULL,
 ALTER COLUMN as_of_date DROP NOT NULL,
 ALTER COLUMN observable_variables DROP NOT NULL,
 ALTER COLUMN observable_variables DROP DEFAULT,
 ADD PRIMARY KEY(id),
 ADD UNIQUE(name),
 ADD CHECK (id ~ '^SICH[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
 ADD CHECK (btrim(name)<>''),
 ADD CHECK (valid_independent_object_text_set(aliases,TRUE)),
 ADD CHECK (scope IS NULL OR btrim(scope)<>''),
 ADD CHECK (target_output IS NULL OR btrim(target_output)<>''),
 ADD CHECK (end_use IS NULL OR btrim(end_use)<>''),
 ADD CHECK (geography IS NULL OR btrim(geography)<>''),
 ADD CHECK (observable_variables IS NULL OR valid_independent_object_text_set(observable_variables,FALSE)),
 ADD CHECK (review_status IN ('candidate','approved')),
 ADD CHECK (review_status<>'approved' OR (scope IS NOT NULL AND target_output IS NOT NULL AND end_use IS NOT NULL AND geography IS NOT NULL AND as_of_date IS NOT NULL AND observable_variables IS NOT NULL)),
 ADD CHECK (short_name IS NULL OR (btrim(short_name)<>'' AND char_length(short_name)<=5)),
 ADD CHECK (review_note IS NULL OR btrim(review_note)<>''),
 ADD CHECK (technology_route_qualifier IS NULL OR btrim(technology_route_qualifier)<>''),
 ADD CHECK (updated_at>=created_at),
 ADD FOREIGN KEY (primary_country_id) REFERENCES countries(id) ON DELETE RESTRICT;

CREATE TABLE stock_s_industry_links (
 id VARCHAR(64) PRIMARY KEY CHECK (id ~ '^SIL[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
 stock_id VARCHAR(64) NOT NULL REFERENCES stock(id) ON DELETE RESTRICT,
 industry_id VARCHAR(64) NOT NULL REFERENCES s_industry(id) ON DELETE RESTRICT,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 UNIQUE(stock_id,industry_id)
);
CREATE INDEX idx_stock_s_industry_reverse ON stock_s_industry_links(industry_id,stock_id);

CREATE TABLE stock_s_concept_links (
 id VARCHAR(64) PRIMARY KEY CHECK (id ~ '^SCL[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
 stock_id VARCHAR(64) NOT NULL REFERENCES stock(id) ON DELETE RESTRICT,
 concept_id VARCHAR(64) NOT NULL REFERENCES s_concept(id) ON DELETE RESTRICT,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 UNIQUE(stock_id,concept_id)
);
CREATE INDEX idx_stock_s_concept_reverse ON stock_s_concept_links(concept_id,stock_id);

CREATE TABLE stock_s_industry_chain_links (
 id VARCHAR(64) PRIMARY KEY CHECK (id ~ '^SICL[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
 stock_id VARCHAR(64) NOT NULL REFERENCES stock(id) ON DELETE RESTRICT,
 industry_chain_id VARCHAR(64) NOT NULL REFERENCES s_industry_chain(id) ON DELETE RESTRICT,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 UNIQUE(stock_id,industry_chain_id)
);
CREATE INDEX idx_stock_s_industry_chain_reverse ON stock_s_industry_chain_links(industry_chain_id,stock_id);

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'stock classification migration is forward-only'; END $$;
-- +goose StatementEnd
