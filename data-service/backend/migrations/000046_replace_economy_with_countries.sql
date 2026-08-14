-- +goose Up
CREATE TABLE countries (
    id VARCHAR(32) PRIMARY KEY,
    code CHAR(3) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    name_en VARCHAR(100) NOT NULL,
    strategic_positioning TEXT,
    key_resources TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_countries_code CHECK (code ~ '^[A-Z]{3}$'),
    CONSTRAINT chk_countries_identity CHECK (id = 'COU_' || code),
    CONSTRAINT chk_countries_names CHECK (btrim(name) <> '' AND btrim(name_en) <> ''),
    CONSTRAINT chk_countries_strategic_positioning CHECK (
        strategic_positioning IS NULL OR btrim(strategic_positioning) <> ''
    ),
    CONSTRAINT chk_countries_key_resources CHECK (
        key_resources IS NULL OR btrim(key_resources) <> ''
    )
);

CREATE TABLE country_region_links (
    id SERIAL PRIMARY KEY,
    country_id VARCHAR(32) NOT NULL REFERENCES countries(id) ON DELETE RESTRICT,
    region_id VARCHAR(32) NOT NULL REFERENCES regions(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_country_region_links UNIQUE (country_id, region_id)
);

CREATE INDEX idx_country_region_links_region_country
    ON country_region_links (region_id, country_id);

ALTER TABLE event_entity_links
    ALTER COLUMN entity_id DROP NOT NULL,
    ADD COLUMN country_id VARCHAR(32) REFERENCES countries(id) ON DELETE RESTRICT,
    ADD CONSTRAINT chk_event_entity_links_object_reference CHECK (
        (entity_id IS NOT NULL)::integer + (country_id IS NOT NULL)::integer = 1
    );

CREATE INDEX idx_event_entity_links_country_id
    ON event_entity_links (country_id)
    WHERE country_id IS NOT NULL;

CREATE UNIQUE INDEX ux_event_entity_link_country_active_accepted
    ON event_entity_links(event_id, country_id, entity_role)
    WHERE review_status = 'accepted' AND country_id IS NOT NULL;

ALTER TABLE market_profiles
    DROP COLUMN economy_entity_id,
    ADD COLUMN country_id VARCHAR(32) REFERENCES countries(id) ON DELETE RESTRICT;

ALTER TABLE company_profiles
    DROP COLUMN registration_economy_entity_id,
    ADD COLUMN registration_country_id VARCHAR(32) REFERENCES countries(id) ON DELETE RESTRICT;

ALTER TABLE person_profiles
    DROP COLUMN economy_entity_id,
    ADD COLUMN country_id VARCHAR(32) REFERENCES countries(id) ON DELETE RESTRICT;

ALTER TABLE industry_chain_definitions
    ADD COLUMN primary_country_id VARCHAR(32) REFERENCES countries(id) ON DELETE RESTRICT;

CREATE INDEX idx_industry_chain_definitions_primary_country
    ON industry_chain_definitions (primary_country_id)
    WHERE primary_country_id IS NOT NULL;

CREATE TEMP TABLE retired_economy_entity_ids ON COMMIT DROP AS
SELECT id
FROM entity_nodes
WHERE entity_type = 'economy';

-- No legacy Economy row is guessed into Country. References whose only meaning was
-- the retired Economy identity are removed under the snapshot-protected cutover.
CREATE TEMP TABLE retired_economy_edge_ids ON COMMIT DROP AS
SELECT id
FROM entity_edges
WHERE from_entity_id IN (SELECT id FROM retired_economy_entity_ids)
   OR to_entity_id IN (SELECT id FROM retired_economy_entity_ids);

CREATE TEMP TABLE retired_economy_event_link_seed_ids ON COMMIT DROP AS
SELECT id
FROM event_entity_links
WHERE entity_id IN (SELECT id FROM retired_economy_entity_ids);

CREATE TEMP TABLE retired_economy_variable_signal_seed_ids ON COMMIT DROP AS
SELECT id
FROM variable_signals
WHERE subject_event_entity_link_id IN (SELECT id FROM retired_economy_event_link_seed_ids);

CREATE TEMP TABLE retired_economy_direct_impact_seed_ids ON COMMIT DROP AS
SELECT impact.id
FROM direct_impact_assertions impact
LEFT JOIN direct_transmission_rules rule
  ON rule.rule_key = impact.rule_key AND rule.version = impact.rule_version
WHERE impact.target_entity_id IN (SELECT id FROM retired_economy_entity_ids)
   OR impact.source_variable_signal_id IN (SELECT id FROM retired_economy_variable_signal_seed_ids)
   OR impact.entity_relation_id IN (SELECT id FROM retired_economy_edge_ids)
   OR rule.source_entity_type = 'economy'
   OR rule.target_entity_type = 'economy';

-- A semantic submission is one immutable aggregate. If any candidate depends on
-- Economy, retire the whole submission instead of leaving snapshots that name
-- candidates whose persisted Link/Signal/Impact rows were removed.
CREATE TEMP TABLE retired_economy_semantic_submission_ids ON COMMIT DROP AS
SELECT semantic_submission_id id
FROM event_entity_links
WHERE id IN (SELECT id FROM retired_economy_event_link_seed_ids)
  AND semantic_submission_id IS NOT NULL
UNION
SELECT semantic_submission_id
FROM variable_signals
WHERE id IN (SELECT id FROM retired_economy_variable_signal_seed_ids)
UNION
SELECT semantic_submission_id
FROM direct_impact_assertions
WHERE id IN (SELECT id FROM retired_economy_direct_impact_seed_ids)
UNION
SELECT semantic_submission_id
FROM event_semantic_resolution_bindings
WHERE anchor_entity_id IN (SELECT id FROM retired_economy_entity_ids)
   OR target_entity_id IN (SELECT id FROM retired_economy_entity_ids);

CREATE TEMP TABLE retired_economy_event_link_ids ON COMMIT DROP AS
SELECT id FROM retired_economy_event_link_seed_ids
UNION
SELECT id
FROM event_entity_links
WHERE semantic_submission_id IN (SELECT id FROM retired_economy_semantic_submission_ids);

CREATE TEMP TABLE retired_economy_variable_signal_ids ON COMMIT DROP AS
SELECT id FROM retired_economy_variable_signal_seed_ids
UNION
SELECT id
FROM variable_signals
WHERE semantic_submission_id IN (SELECT id FROM retired_economy_semantic_submission_ids);

CREATE TEMP TABLE retired_economy_direct_impact_ids ON COMMIT DROP AS
SELECT id FROM retired_economy_direct_impact_seed_ids
UNION
SELECT id
FROM direct_impact_assertions
WHERE semantic_submission_id IN (SELECT id FROM retired_economy_semantic_submission_ids)
   OR source_variable_signal_id IN (SELECT id FROM retired_economy_variable_signal_ids);

CREATE TEMP TABLE retired_economy_research_theme_seed_ids ON COMMIT DROP AS
SELECT DISTINCT tree.theme_id
FROM research_reasoning_trees tree
JOIN research_reasoning_tree_nodes node ON node.reasoning_tree_id = tree.id
WHERE node.direct_impact_assertion_id IN (SELECT id FROM retired_economy_direct_impact_ids)
   OR node.direct_impact_semantic_submission_id IN (SELECT id FROM retired_economy_semantic_submission_ids)
   OR node.inference_upstream_direct_impact_assertion_id IN (SELECT id FROM retired_economy_direct_impact_ids)
   OR node.inference_upstream_variable_signal_id IN (SELECT id FROM retired_economy_variable_signal_ids)
   OR node.inference_entity_relation_id IN (SELECT id FROM retired_economy_edge_ids)
UNION
SELECT DISTINCT tree.theme_id
FROM research_reasoning_trees tree
JOIN research_reasoning_tree_nodes node ON node.reasoning_tree_id = tree.id
JOIN research_reasoning_tree_node_signals signal ON signal.reasoning_tree_node_id = node.id
WHERE signal.variable_signal_id IN (SELECT id FROM retired_economy_variable_signal_ids)
   OR signal.semantic_submission_id IN (SELECT id FROM retired_economy_semantic_submission_ids)
   OR signal.upstream_variable_signal_id IN (SELECT id FROM retired_economy_variable_signal_ids)
   OR signal.upstream_direct_impact_assertion_id IN (SELECT id FROM retired_economy_direct_impact_ids)
   OR signal.entity_relation_id IN (SELECT id FROM retired_economy_edge_ids);

-- One publication receipt can own more than one historical Theme. Retire the whole
-- immutable publication aggregate rather than leave a partially mutated receipt.
CREATE TEMP TABLE retired_economy_research_theme_ids ON COMMIT DROP AS
SELECT theme.id
FROM research_themes theme
WHERE theme.import_receipt_id IN (
    SELECT affected.import_receipt_id
    FROM research_themes affected
    WHERE affected.id IN (SELECT theme_id FROM retired_economy_research_theme_seed_ids)
);

CREATE TEMP TABLE retired_economy_research_tree_receipt_ids ON COMMIT DROP AS
SELECT import_receipt_id
FROM research_reasoning_trees
WHERE theme_id IN (SELECT id FROM retired_economy_research_theme_ids);

CREATE TEMP TABLE retired_economy_research_theme_receipt_ids ON COMMIT DROP AS
SELECT DISTINCT import_receipt_id
FROM research_themes
WHERE id IN (SELECT id FROM retired_economy_research_theme_ids);

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
  AND tree.theme_id IN (SELECT id FROM retired_economy_research_theme_ids);

DELETE FROM research_reasoning_tree_nodes node
USING research_reasoning_trees tree
WHERE node.reasoning_tree_id = tree.id
  AND tree.theme_id IN (SELECT id FROM retired_economy_research_theme_ids);

DELETE FROM research_reasoning_tree_events event
USING research_reasoning_trees tree
WHERE event.reasoning_tree_id = tree.id
  AND tree.theme_id IN (SELECT id FROM retired_economy_research_theme_ids);

DELETE FROM research_reasoning_trees
WHERE theme_id IN (SELECT id FROM retired_economy_research_theme_ids);

DELETE FROM research_reasoning_tree_import_receipts
WHERE id IN (SELECT import_receipt_id FROM retired_economy_research_tree_receipt_ids);

DELETE FROM research_theme_impacts
WHERE theme_id IN (SELECT id FROM retired_economy_research_theme_ids);

DELETE FROM research_theme_events
WHERE theme_id IN (SELECT id FROM retired_economy_research_theme_ids);

DELETE FROM research_themes
WHERE id IN (SELECT id FROM retired_economy_research_theme_ids);

DELETE FROM research_theme_import_receipts
WHERE id IN (SELECT import_receipt_id FROM retired_economy_research_theme_receipt_ids);

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
WHERE variable_signal_id IN (SELECT id FROM retired_economy_variable_signal_ids);

DELETE FROM direct_impact_assertions
WHERE id IN (SELECT id FROM retired_economy_direct_impact_ids);

DELETE FROM variable_signals
WHERE id IN (SELECT id FROM retired_economy_variable_signal_ids);

DELETE FROM event_semantic_resolution_bindings
WHERE semantic_submission_id IN (SELECT id FROM retired_economy_semantic_submission_ids)
   OR anchor_entity_id IN (SELECT id FROM retired_economy_entity_ids)
   OR target_entity_id IN (SELECT id FROM retired_economy_entity_ids);

DELETE FROM entity_edges
WHERE id IN (SELECT id FROM retired_economy_edge_ids);

DELETE FROM entity_redirects
WHERE source_entity_id IN (SELECT id FROM retired_economy_entity_ids)
   OR target_entity_id IN (SELECT id FROM retired_economy_entity_ids);

UPDATE index_profiles
SET market_entity_id = NULL
WHERE market_entity_id IN (SELECT id FROM retired_economy_entity_ids);

UPDATE instrument_profiles
SET underlying_entity_id = NULL
WHERE underlying_entity_id IN (SELECT id FROM retired_economy_entity_ids);

UPDATE person_profiles
SET organization_entity_id = NULL
WHERE organization_entity_id IN (SELECT id FROM retired_economy_entity_ids);

UPDATE security_profiles
SET issuer_company_entity_id = NULL
WHERE issuer_company_entity_id IN (SELECT id FROM retired_economy_entity_ids);

DELETE FROM event_entity_links
WHERE id IN (SELECT id FROM retired_economy_event_link_ids);

DELETE FROM event_semantic_candidate_snapshots
WHERE semantic_submission_id IN (SELECT id FROM retired_economy_semantic_submission_ids);

DELETE FROM event_semantic_review_snapshots
WHERE semantic_submission_id IN (SELECT id FROM retired_economy_semantic_submission_ids);

CREATE TEMP TABLE retired_economy_context_lease_ids ON COMMIT DROP AS
SELECT context_lease_id
FROM event_semantic_submissions
WHERE id IN (SELECT id FROM retired_economy_semantic_submission_ids);

UPDATE event_semantic_context_leases
SET supersedes_submission_id = NULL
WHERE supersedes_submission_id IN (SELECT id FROM retired_economy_semantic_submission_ids);

UPDATE event_semantic_submissions
SET supersedes_submission_id = NULL
WHERE supersedes_submission_id IN (SELECT id FROM retired_economy_semantic_submission_ids)
  AND id NOT IN (SELECT id FROM retired_economy_semantic_submission_ids);

DELETE FROM event_semantic_submissions
WHERE id IN (SELECT id FROM retired_economy_semantic_submission_ids);

DELETE FROM event_semantic_context_leases
WHERE id IN (SELECT context_lease_id FROM retired_economy_context_lease_ids);

DELETE FROM direct_transmission_rules
WHERE source_entity_type = 'economy' OR target_entity_type = 'economy';

DELETE FROM variable_definition_entity_types
WHERE entity_type = 'economy';

DROP TABLE economy_profiles;

DELETE FROM entity_nodes
WHERE id IN (SELECT id FROM retired_economy_entity_ids);

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000046 is forward-only; restore the pre-Country PostgreSQL snapshot and previous applications';
END;
$$;
-- +goose StatementEnd
