-- +goose Up
SELECT pg_advisory_xact_lock(hashtextextended('tidewise:migration:btree_gist', 0));
CREATE EXTENSION IF NOT EXISTS btree_gist WITH SCHEMA public;

CREATE TYPE organization_binding_power_level AS ENUM ('HIGH', 'MEDIUM', 'LOW');
CREATE TYPE organization_influence_rating AS ENUM ('S', 'A', 'B');
CREATE TYPE organization_membership_type AS ENUM (
    'FULL_MEMBER',
    'OBSERVER',
    'ASSOCIATE',
    'PARTNER',
    'CANDIDATE'
);

CREATE TABLE organization_categories (
    code VARCHAR(30) PRIMARY KEY,
    name_zh VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_organization_categories_code CHECK (code ~ '^[A-Z][A-Z0-9_]*$'),
    CONSTRAINT chk_organization_categories_name CHECK (btrim(name_zh) <> '')
);

CREATE TABLE organization_functions (
    code VARCHAR(30) PRIMARY KEY,
    name_zh VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_organization_functions_code CHECK (code ~ '^[A-Z][A-Z0-9_]*$'),
    CONSTRAINT chk_organization_functions_name CHECK (btrim(name_zh) <> '')
);

CREATE TABLE organization_domain_tags (
    code VARCHAR(50) PRIMARY KEY,
    function_code VARCHAR(30) NOT NULL REFERENCES organization_functions(code) ON DELETE RESTRICT,
    name_zh VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_organization_domain_tags_code_function UNIQUE (code, function_code),
    CONSTRAINT chk_organization_domain_tags_code CHECK (code ~ '^[A-Z][A-Z0-9_]*$'),
    CONSTRAINT chk_organization_domain_tags_name CHECK (btrim(name_zh) <> '')
);

CREATE TABLE organizations (
    id VARCHAR(32) PRIMARY KEY,
    code VARCHAR(30) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    name_en VARCHAR(100) NOT NULL,
    region_id VARCHAR(32) REFERENCES regions(id) ON DELETE RESTRICT,
    category_code VARCHAR(30) NOT NULL REFERENCES organization_categories(code) ON DELETE RESTRICT,
    function_code VARCHAR(30) NOT NULL REFERENCES organization_functions(code) ON DELETE RESTRICT,
    legal_entity_code CHAR(20),
    dominant_party_id VARCHAR(32) REFERENCES countries(id) ON DELETE RESTRICT,
    binding_power_level organization_binding_power_level,
    influence_rating organization_influence_rating,
    strategic_positioning TEXT,
    core_impact_scope TEXT,
    founding_document VARCHAR(200),
    established_date DATE,
    headquarters_city VARCHAR(100),
    headquarters_country_id VARCHAR(32) REFERENCES countries(id) ON DELETE RESTRICT,
    headquarters_subdivision_id VARCHAR(32),
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_organizations_id_function UNIQUE (id, function_code),
    CONSTRAINT chk_organizations_code CHECK (code ~ '^[A-Z][A-Z0-9_]*$'),
    CONSTRAINT chk_organizations_identity CHECK (id = 'ORG_' || code),
    CONSTRAINT chk_organizations_names CHECK (btrim(name) <> '' AND btrim(name_en) <> ''),
    CONSTRAINT chk_organizations_legal_entity_code CHECK (
        legal_entity_code IS NULL OR legal_entity_code ~ '^[A-Z0-9]{20}$'
    ),
    CONSTRAINT chk_organizations_optional_text CHECK (
        (strategic_positioning IS NULL OR btrim(strategic_positioning) <> '') AND
        (core_impact_scope IS NULL OR btrim(core_impact_scope) <> '') AND
        (founding_document IS NULL OR btrim(founding_document) <> '') AND
        (headquarters_city IS NULL OR btrim(headquarters_city) <> '') AND
        (headquarters_subdivision_id IS NULL OR btrim(headquarters_subdivision_id) <> '') AND
        (description IS NULL OR btrim(description) <> '')
    )
);

CREATE INDEX idx_organizations_category_code ON organizations (category_code, code, id);
CREATE INDEX idx_organizations_function_code ON organizations (function_code, code, id);
CREATE INDEX idx_organizations_region_id ON organizations (region_id, code, id) WHERE region_id IS NOT NULL;
CREATE INDEX idx_organizations_dominant_party_id ON organizations (dominant_party_id, code, id) WHERE dominant_party_id IS NOT NULL;

CREATE TABLE organization_domain_tag_links (
    organization_id VARCHAR(32) NOT NULL,
    function_code VARCHAR(30) NOT NULL,
    domain_tag_code VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, domain_tag_code),
    CONSTRAINT fk_organization_domain_tag_links_organization_function
        FOREIGN KEY (organization_id, function_code)
        REFERENCES organizations(id, function_code) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT fk_organization_domain_tag_links_tag_function
        FOREIGN KEY (domain_tag_code, function_code)
        REFERENCES organization_domain_tags(code, function_code) ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX idx_organization_domain_tag_links_tag
    ON organization_domain_tag_links (domain_tag_code, organization_id);

CREATE TABLE organization_members (
    id BIGSERIAL PRIMARY KEY,
    organization_id VARCHAR(32) NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    country_id VARCHAR(32) NOT NULL REFERENCES countries(id) ON DELETE RESTRICT,
    membership_type organization_membership_type NOT NULL,
    effective_date DATE,
    expiry_date DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_organization_members_dates CHECK (
        expiry_date IS NULL OR effective_date IS NULL OR expiry_date >= effective_date
    ),
    CONSTRAINT ex_organization_members_no_overlap EXCLUDE USING gist (
        organization_id WITH =,
        country_id WITH =,
        daterange(
            COALESCE(effective_date, '-infinity'::date),
            COALESCE(expiry_date, 'infinity'::date),
            '[]'
        ) WITH &&
    )
);

CREATE INDEX idx_organization_members_organization_dates
    ON organization_members (organization_id, effective_date, expiry_date, country_id, id);
CREATE INDEX idx_organization_members_country_dates
    ON organization_members (country_id, effective_date, expiry_date, organization_id, id);

ALTER TABLE event_entity_links
    DROP CONSTRAINT chk_event_entity_links_object_reference,
    ADD COLUMN organization_id VARCHAR(32) REFERENCES organizations(id) ON DELETE RESTRICT,
    ADD CONSTRAINT chk_event_entity_links_object_reference CHECK (
        (entity_id IS NOT NULL)::integer +
        (country_id IS NOT NULL)::integer +
        (organization_id IS NOT NULL)::integer = 1
    );

CREATE INDEX idx_event_entity_links_organization_id
    ON event_entity_links (organization_id)
    WHERE organization_id IS NOT NULL;

CREATE UNIQUE INDEX ux_event_entity_link_organization_active_accepted
    ON event_entity_links(event_id, organization_id, entity_role)
    WHERE review_status = 'accepted' AND organization_id IS NOT NULL;

-- No legacy Alliance row is guessed into Organization. Snapshot-protected cutover
-- retires every immutable aggregate that still depends on an Alliance identity.
CREATE TEMP TABLE retired_alliance_entity_ids ON COMMIT DROP AS
SELECT id
FROM entity_nodes
WHERE entity_type = 'alliance_org';

-- References whose only meaning was
-- the retired Alliance identity are removed under the snapshot-protected cutover.
CREATE TEMP TABLE retired_alliance_edge_ids ON COMMIT DROP AS
SELECT id
FROM entity_edges
WHERE from_entity_id IN (SELECT id FROM retired_alliance_entity_ids)
   OR to_entity_id IN (SELECT id FROM retired_alliance_entity_ids);

CREATE TEMP TABLE retired_alliance_event_link_seed_ids ON COMMIT DROP AS
SELECT id
FROM event_entity_links
WHERE entity_id IN (SELECT id FROM retired_alliance_entity_ids);

CREATE TEMP TABLE retired_alliance_variable_signal_seed_ids ON COMMIT DROP AS
SELECT id
FROM variable_signals
WHERE subject_event_entity_link_id IN (SELECT id FROM retired_alliance_event_link_seed_ids);

CREATE TEMP TABLE retired_alliance_direct_impact_seed_ids ON COMMIT DROP AS
SELECT impact.id
FROM direct_impact_assertions impact
LEFT JOIN direct_transmission_rules rule
  ON rule.rule_key = impact.rule_key AND rule.version = impact.rule_version
WHERE impact.target_entity_id IN (SELECT id FROM retired_alliance_entity_ids)
   OR impact.source_variable_signal_id IN (SELECT id FROM retired_alliance_variable_signal_seed_ids)
   OR impact.entity_relation_id IN (SELECT id FROM retired_alliance_edge_ids)
   OR rule.source_entity_type = 'alliance_org'
   OR rule.target_entity_type = 'alliance_org';

-- A semantic submission is one immutable aggregate. If any candidate depends on
-- Alliance, retire the whole submission instead of leaving snapshots that name
-- candidates whose persisted Link/Signal/Impact rows were removed.
CREATE TEMP TABLE retired_alliance_semantic_submission_ids ON COMMIT DROP AS
SELECT semantic_submission_id id
FROM event_entity_links
WHERE id IN (SELECT id FROM retired_alliance_event_link_seed_ids)
  AND semantic_submission_id IS NOT NULL
UNION
SELECT semantic_submission_id
FROM variable_signals
WHERE id IN (SELECT id FROM retired_alliance_variable_signal_seed_ids)
UNION
SELECT semantic_submission_id
FROM direct_impact_assertions
WHERE id IN (SELECT id FROM retired_alliance_direct_impact_seed_ids)
UNION
SELECT semantic_submission_id
FROM event_semantic_resolution_bindings
WHERE anchor_entity_id IN (SELECT id FROM retired_alliance_entity_ids)
   OR target_entity_id IN (SELECT id FROM retired_alliance_entity_ids);

CREATE TEMP TABLE retired_alliance_event_link_ids ON COMMIT DROP AS
SELECT id FROM retired_alliance_event_link_seed_ids
UNION
SELECT id
FROM event_entity_links
WHERE semantic_submission_id IN (SELECT id FROM retired_alliance_semantic_submission_ids);

CREATE TEMP TABLE retired_alliance_variable_signal_ids ON COMMIT DROP AS
SELECT id FROM retired_alliance_variable_signal_seed_ids
UNION
SELECT id
FROM variable_signals
WHERE semantic_submission_id IN (SELECT id FROM retired_alliance_semantic_submission_ids);

CREATE TEMP TABLE retired_alliance_direct_impact_ids ON COMMIT DROP AS
SELECT id FROM retired_alliance_direct_impact_seed_ids
UNION
SELECT id
FROM direct_impact_assertions
WHERE semantic_submission_id IN (SELECT id FROM retired_alliance_semantic_submission_ids)
   OR source_variable_signal_id IN (SELECT id FROM retired_alliance_variable_signal_ids);

CREATE TEMP TABLE retired_alliance_research_theme_seed_ids ON COMMIT DROP AS
SELECT DISTINCT tree.theme_id
FROM research_reasoning_trees tree
JOIN research_reasoning_tree_nodes node ON node.reasoning_tree_id = tree.id
WHERE node.direct_impact_assertion_id IN (SELECT id FROM retired_alliance_direct_impact_ids)
   OR node.direct_impact_semantic_submission_id IN (SELECT id FROM retired_alliance_semantic_submission_ids)
   OR node.inference_upstream_direct_impact_assertion_id IN (SELECT id FROM retired_alliance_direct_impact_ids)
   OR node.inference_upstream_variable_signal_id IN (SELECT id FROM retired_alliance_variable_signal_ids)
   OR node.inference_entity_relation_id IN (SELECT id FROM retired_alliance_edge_ids)
UNION
SELECT DISTINCT tree.theme_id
FROM research_reasoning_trees tree
JOIN research_reasoning_tree_nodes node ON node.reasoning_tree_id = tree.id
JOIN research_reasoning_tree_node_signals signal ON signal.reasoning_tree_node_id = node.id
WHERE signal.variable_signal_id IN (SELECT id FROM retired_alliance_variable_signal_ids)
   OR signal.semantic_submission_id IN (SELECT id FROM retired_alliance_semantic_submission_ids)
   OR signal.upstream_variable_signal_id IN (SELECT id FROM retired_alliance_variable_signal_ids)
   OR signal.upstream_direct_impact_assertion_id IN (SELECT id FROM retired_alliance_direct_impact_ids)
   OR signal.entity_relation_id IN (SELECT id FROM retired_alliance_edge_ids);

-- One publication receipt can own more than one historical Theme. Retire the whole
-- immutable publication aggregate rather than leave a partially mutated receipt.
CREATE TEMP TABLE retired_alliance_research_theme_ids ON COMMIT DROP AS
SELECT theme.id
FROM research_themes theme
WHERE theme.import_receipt_id IN (
    SELECT affected.import_receipt_id
    FROM research_themes affected
    WHERE affected.id IN (SELECT theme_id FROM retired_alliance_research_theme_seed_ids)
);

CREATE TEMP TABLE retired_alliance_research_tree_receipt_ids ON COMMIT DROP AS
SELECT import_receipt_id
FROM research_reasoning_trees
WHERE theme_id IN (SELECT id FROM retired_alliance_research_theme_ids);

CREATE TEMP TABLE retired_alliance_research_theme_receipt_ids ON COMMIT DROP AS
SELECT DISTINCT import_receipt_id
FROM research_themes
WHERE id IN (SELECT id FROM retired_alliance_research_theme_ids);

ALTER TABLE research_reasoning_tree_node_signals DISABLE TRIGGER trg_research_reasoning_tree_node_signals_immutable;
ALTER TABLE research_reasoning_tree_nodes DISABLE TRIGGER trg_research_reasoning_tree_nodes_immutable;
ALTER TABLE research_reasoning_tree_events DISABLE TRIGGER trg_research_reasoning_tree_events_immutable;
ALTER TABLE research_reasoning_trees DISABLE TRIGGER trg_research_reasoning_trees_immutable;
ALTER TABLE research_reasoning_tree_import_receipts DISABLE TRIGGER trg_research_reasoning_tree_receipts_immutable;
ALTER TABLE research_theme_impacts DISABLE TRIGGER trg_research_theme_impacts_immutable;
ALTER TABLE research_theme_events DISABLE TRIGGER trg_research_theme_events_immutable;
ALTER TABLE research_themes DISABLE TRIGGER trg_research_themes_immutable;
ALTER TABLE research_theme_import_receipts DISABLE TRIGGER trg_research_theme_receipts_immutable;

DELETE FROM research_reasoning_tree_node_signals signal
USING research_reasoning_tree_nodes node, research_reasoning_trees tree
WHERE signal.reasoning_tree_node_id = node.id
  AND node.reasoning_tree_id = tree.id
  AND tree.theme_id IN (SELECT id FROM retired_alliance_research_theme_ids);

DELETE FROM research_reasoning_tree_nodes node
USING research_reasoning_trees tree
WHERE node.reasoning_tree_id = tree.id
  AND tree.theme_id IN (SELECT id FROM retired_alliance_research_theme_ids);

DELETE FROM research_reasoning_tree_events event
USING research_reasoning_trees tree
WHERE event.reasoning_tree_id = tree.id
  AND tree.theme_id IN (SELECT id FROM retired_alliance_research_theme_ids);

DELETE FROM research_reasoning_trees
WHERE theme_id IN (SELECT id FROM retired_alliance_research_theme_ids);

DELETE FROM research_reasoning_tree_import_receipts
WHERE id IN (SELECT import_receipt_id FROM retired_alliance_research_tree_receipt_ids);

DELETE FROM research_theme_impacts
WHERE theme_id IN (SELECT id FROM retired_alliance_research_theme_ids);

DELETE FROM research_theme_events
WHERE theme_id IN (SELECT id FROM retired_alliance_research_theme_ids);

DELETE FROM research_themes
WHERE id IN (SELECT id FROM retired_alliance_research_theme_ids);

DELETE FROM research_theme_import_receipts
WHERE id IN (SELECT import_receipt_id FROM retired_alliance_research_theme_receipt_ids);

ALTER TABLE research_reasoning_tree_node_signals ENABLE TRIGGER trg_research_reasoning_tree_node_signals_immutable;
ALTER TABLE research_reasoning_tree_nodes ENABLE TRIGGER trg_research_reasoning_tree_nodes_immutable;
ALTER TABLE research_reasoning_tree_events ENABLE TRIGGER trg_research_reasoning_tree_events_immutable;
ALTER TABLE research_reasoning_trees ENABLE TRIGGER trg_research_reasoning_trees_immutable;
ALTER TABLE research_reasoning_tree_import_receipts ENABLE TRIGGER trg_research_reasoning_tree_receipts_immutable;
ALTER TABLE research_theme_impacts ENABLE TRIGGER trg_research_theme_impacts_immutable;
ALTER TABLE research_theme_events ENABLE TRIGGER trg_research_theme_events_immutable;
ALTER TABLE research_themes ENABLE TRIGGER trg_research_themes_immutable;
ALTER TABLE research_theme_import_receipts ENABLE TRIGGER trg_research_theme_receipts_immutable;

DELETE FROM variable_signal_measurements
WHERE variable_signal_id IN (SELECT id FROM retired_alliance_variable_signal_ids);

DELETE FROM direct_impact_assertions
WHERE id IN (SELECT id FROM retired_alliance_direct_impact_ids);

DELETE FROM variable_signals
WHERE id IN (SELECT id FROM retired_alliance_variable_signal_ids);

DELETE FROM event_semantic_resolution_bindings
WHERE semantic_submission_id IN (SELECT id FROM retired_alliance_semantic_submission_ids)
   OR anchor_entity_id IN (SELECT id FROM retired_alliance_entity_ids)
   OR target_entity_id IN (SELECT id FROM retired_alliance_entity_ids);

DELETE FROM entity_edges
WHERE id IN (SELECT id FROM retired_alliance_edge_ids);

DELETE FROM entity_redirects
WHERE source_entity_id IN (SELECT id FROM retired_alliance_entity_ids)
   OR target_entity_id IN (SELECT id FROM retired_alliance_entity_ids);

UPDATE index_profiles
SET market_entity_id = NULL
WHERE market_entity_id IN (SELECT id FROM retired_alliance_entity_ids);

UPDATE instrument_profiles
SET underlying_entity_id = NULL
WHERE underlying_entity_id IN (SELECT id FROM retired_alliance_entity_ids);

UPDATE person_profiles
SET organization_entity_id = NULL
WHERE organization_entity_id IN (SELECT id FROM retired_alliance_entity_ids);

UPDATE security_profiles
SET issuer_company_entity_id = NULL
WHERE issuer_company_entity_id IN (SELECT id FROM retired_alliance_entity_ids);

DELETE FROM event_entity_links
WHERE id IN (SELECT id FROM retired_alliance_event_link_ids);

DELETE FROM event_semantic_candidate_snapshots
WHERE semantic_submission_id IN (SELECT id FROM retired_alliance_semantic_submission_ids);

DELETE FROM event_semantic_review_snapshots
WHERE semantic_submission_id IN (SELECT id FROM retired_alliance_semantic_submission_ids);

CREATE TEMP TABLE retired_alliance_context_lease_ids ON COMMIT DROP AS
SELECT context_lease_id
FROM event_semantic_submissions
WHERE id IN (SELECT id FROM retired_alliance_semantic_submission_ids);

UPDATE event_semantic_context_leases
SET supersedes_submission_id = NULL
WHERE supersedes_submission_id IN (SELECT id FROM retired_alliance_semantic_submission_ids);

UPDATE event_semantic_submissions
SET supersedes_submission_id = NULL
WHERE supersedes_submission_id IN (SELECT id FROM retired_alliance_semantic_submission_ids)
  AND id NOT IN (SELECT id FROM retired_alliance_semantic_submission_ids);

DELETE FROM event_semantic_submissions
WHERE id IN (SELECT id FROM retired_alliance_semantic_submission_ids);

DELETE FROM event_semantic_context_leases
WHERE id IN (SELECT context_lease_id FROM retired_alliance_context_lease_ids);

DELETE FROM direct_transmission_rules
WHERE source_entity_type = 'alliance_org' OR target_entity_type = 'alliance_org';

DELETE FROM variable_definition_entity_types
WHERE entity_type = 'alliance_org';

DROP TABLE alliance_org_profiles;

DELETE FROM entity_nodes
WHERE id IN (SELECT id FROM retired_alliance_entity_ids);

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000048 is forward-only; restore the pre-Organization PostgreSQL snapshot and previous applications';
END;
$$;
-- +goose StatementEnd
