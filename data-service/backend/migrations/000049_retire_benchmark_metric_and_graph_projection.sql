-- +goose Up
CREATE TEMP TABLE retired_entity_ids ON COMMIT DROP AS
SELECT id FROM entity_nodes WHERE entity_type IN ('benchmark', 'metric');

CREATE TEMP TABLE retired_edge_ids ON COMMIT DROP AS
SELECT id FROM entity_edges
WHERE from_entity_id IN (SELECT id FROM retired_entity_ids)
   OR to_entity_id IN (SELECT id FROM retired_entity_ids);

CREATE TEMP TABLE retired_event_link_ids ON COMMIT DROP AS
SELECT id FROM event_entity_links WHERE entity_id IN (SELECT id FROM retired_entity_ids);

CREATE TEMP TABLE retired_semantic_submission_ids ON COMMIT DROP AS
SELECT semantic_submission_id AS id FROM event_entity_links
WHERE id IN (SELECT id FROM retired_event_link_ids) AND semantic_submission_id IS NOT NULL
UNION
SELECT semantic_submission_id FROM variable_signals
WHERE subject_event_entity_link_id IN (SELECT id FROM retired_event_link_ids)
UNION
SELECT semantic_submission_id FROM direct_impact_assertions
WHERE target_entity_id IN (SELECT id FROM retired_entity_ids)
   OR entity_relation_id IN (SELECT id FROM retired_edge_ids)
UNION
SELECT semantic_submission_id FROM event_semantic_resolution_bindings
WHERE anchor_entity_id IN (SELECT id FROM retired_entity_ids)
   OR target_entity_id IN (SELECT id FROM retired_entity_ids);

CREATE TEMP TABLE retired_variable_signal_ids ON COMMIT DROP AS
SELECT id FROM variable_signals
WHERE semantic_submission_id IN (SELECT id FROM retired_semantic_submission_ids)
   OR subject_event_entity_link_id IN (SELECT id FROM retired_event_link_ids);

CREATE TEMP TABLE retired_direct_impact_ids ON COMMIT DROP AS
SELECT id FROM direct_impact_assertions
WHERE semantic_submission_id IN (SELECT id FROM retired_semantic_submission_ids)
   OR target_entity_id IN (SELECT id FROM retired_entity_ids)
   OR entity_relation_id IN (SELECT id FROM retired_edge_ids)
   OR source_variable_signal_id IN (SELECT id FROM retired_variable_signal_ids);

CREATE TEMP TABLE retired_research_theme_seed_ids ON COMMIT DROP AS
SELECT DISTINCT tree.theme_id AS id
FROM research_reasoning_trees tree
JOIN research_reasoning_tree_nodes node ON node.reasoning_tree_id = tree.id
WHERE node.direct_impact_assertion_id IN (SELECT id FROM retired_direct_impact_ids)
   OR node.direct_impact_semantic_submission_id IN (SELECT id FROM retired_semantic_submission_ids)
   OR node.inference_upstream_direct_impact_assertion_id IN (SELECT id FROM retired_direct_impact_ids)
   OR node.inference_upstream_variable_signal_id IN (SELECT id FROM retired_variable_signal_ids)
   OR node.inference_entity_relation_id IN (SELECT id FROM retired_edge_ids)
UNION
SELECT DISTINCT tree.theme_id AS id
FROM research_reasoning_trees tree
JOIN research_reasoning_tree_nodes node ON node.reasoning_tree_id = tree.id
JOIN research_reasoning_tree_node_signals signal ON signal.reasoning_tree_node_id = node.id
WHERE signal.variable_signal_id IN (SELECT id FROM retired_variable_signal_ids)
   OR signal.semantic_submission_id IN (SELECT id FROM retired_semantic_submission_ids)
   OR signal.upstream_variable_signal_id IN (SELECT id FROM retired_variable_signal_ids)
   OR signal.upstream_direct_impact_assertion_id IN (SELECT id FROM retired_direct_impact_ids)
   OR signal.entity_relation_id IN (SELECT id FROM retired_edge_ids);

-- Theme receipts are immutable publication aggregates. Retire every Theme owned
-- by an affected receipt rather than leaving a receipt with only some of its facts.
CREATE TEMP TABLE retired_research_theme_ids ON COMMIT DROP AS
SELECT theme.id
FROM research_themes theme
WHERE theme.import_receipt_id IN (
    SELECT affected.import_receipt_id
    FROM research_themes affected
    WHERE affected.id IN (SELECT id FROM retired_research_theme_seed_ids)
);

CREATE TEMP TABLE retired_research_tree_receipt_ids ON COMMIT DROP AS
SELECT import_receipt_id AS id FROM research_reasoning_trees
WHERE theme_id IN (SELECT id FROM retired_research_theme_ids);

CREATE TEMP TABLE retired_research_theme_receipt_ids ON COMMIT DROP AS
SELECT DISTINCT import_receipt_id AS id FROM research_themes
WHERE id IN (SELECT id FROM retired_research_theme_ids);

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
WHERE signal.reasoning_tree_node_id = node.id AND node.reasoning_tree_id = tree.id
  AND tree.theme_id IN (SELECT retired_research_theme_ids.id FROM retired_research_theme_ids);
DELETE FROM research_reasoning_tree_nodes node USING research_reasoning_trees tree
WHERE node.reasoning_tree_id = tree.id AND tree.theme_id IN (SELECT retired_research_theme_ids.id FROM retired_research_theme_ids);
DELETE FROM research_reasoning_tree_events event USING research_reasoning_trees tree
WHERE event.reasoning_tree_id = tree.id AND tree.theme_id IN (SELECT retired_research_theme_ids.id FROM retired_research_theme_ids);
DELETE FROM research_reasoning_trees WHERE theme_id IN (SELECT retired_research_theme_ids.id FROM retired_research_theme_ids);
DELETE FROM research_reasoning_tree_import_receipts WHERE id IN (SELECT id FROM retired_research_tree_receipt_ids);
DELETE FROM research_theme_impacts WHERE theme_id IN (SELECT retired_research_theme_ids.id FROM retired_research_theme_ids);
DELETE FROM research_theme_events WHERE theme_id IN (SELECT retired_research_theme_ids.id FROM retired_research_theme_ids);
DELETE FROM research_themes WHERE id IN (SELECT retired_research_theme_ids.id FROM retired_research_theme_ids);
DELETE FROM research_theme_import_receipts WHERE id IN (SELECT id FROM retired_research_theme_receipt_ids);

ALTER TABLE research_reasoning_tree_node_signals ENABLE TRIGGER trg_research_reasoning_tree_node_signals_immutable;
ALTER TABLE research_reasoning_tree_nodes ENABLE TRIGGER trg_research_reasoning_tree_nodes_immutable;
ALTER TABLE research_reasoning_tree_events ENABLE TRIGGER trg_research_reasoning_tree_events_immutable;
ALTER TABLE research_reasoning_trees ENABLE TRIGGER trg_research_reasoning_trees_immutable;
ALTER TABLE research_reasoning_tree_import_receipts ENABLE TRIGGER trg_research_reasoning_tree_receipts_immutable;
ALTER TABLE research_theme_impacts ENABLE TRIGGER trg_research_theme_impacts_immutable;
ALTER TABLE research_theme_events ENABLE TRIGGER trg_research_theme_events_immutable;
ALTER TABLE research_themes ENABLE TRIGGER trg_research_themes_immutable;
ALTER TABLE research_theme_import_receipts ENABLE TRIGGER trg_research_theme_receipts_immutable;

DELETE FROM variable_signal_measurements WHERE variable_signal_id IN (SELECT id FROM retired_variable_signal_ids);
DELETE FROM direct_impact_assertions WHERE id IN (SELECT id FROM retired_direct_impact_ids);
DELETE FROM variable_signals WHERE id IN (SELECT id FROM retired_variable_signal_ids);
DELETE FROM event_entity_links WHERE id IN (SELECT id FROM retired_event_link_ids);
DELETE FROM event_semantic_resolution_bindings
WHERE semantic_submission_id IN (SELECT id FROM retired_semantic_submission_ids)
   OR anchor_entity_id IN (SELECT id FROM retired_entity_ids)
   OR target_entity_id IN (SELECT id FROM retired_entity_ids);
DELETE FROM event_semantic_candidate_snapshots WHERE semantic_submission_id IN (SELECT id FROM retired_semantic_submission_ids);
DELETE FROM event_semantic_review_snapshots WHERE semantic_submission_id IN (SELECT id FROM retired_semantic_submission_ids);
CREATE TEMP TABLE retired_context_lease_ids ON COMMIT DROP AS
SELECT context_lease_id FROM event_semantic_submissions WHERE id IN (SELECT id FROM retired_semantic_submission_ids);
UPDATE event_semantic_context_leases
SET supersedes_submission_id = NULL
WHERE supersedes_submission_id IN (SELECT id FROM retired_semantic_submission_ids);
UPDATE event_semantic_submissions
SET supersedes_submission_id = NULL
WHERE supersedes_submission_id IN (SELECT id FROM retired_semantic_submission_ids)
  AND id NOT IN (SELECT id FROM retired_semantic_submission_ids);
DELETE FROM event_semantic_submissions WHERE id IN (SELECT id FROM retired_semantic_submission_ids);
DELETE FROM event_semantic_context_leases WHERE id IN (SELECT context_lease_id FROM retired_context_lease_ids);
DELETE FROM entity_edges WHERE id IN (SELECT id FROM retired_edge_ids);
DELETE FROM entity_redirects WHERE source_entity_id IN (SELECT id FROM retired_entity_ids) OR target_entity_id IN (SELECT id FROM retired_entity_ids);
DROP TABLE IF EXISTS benchmark_observations;
DROP TABLE IF EXISTS benchmark_profiles;
DROP TABLE IF EXISTS metric_profiles;
DELETE FROM entity_nodes WHERE id IN (SELECT id FROM retired_entity_ids);

DROP TABLE IF EXISTS graph_projection_run_items;
DROP TABLE IF EXISTS graph_projection_runs;

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000049 is forward-only; restore the pre-retirement PostgreSQL snapshot and previous applications';
END;
$$;
-- +goose StatementEnd
