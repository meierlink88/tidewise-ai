-- +goose Up
-- Issue #367 authorizes the destructive, zero-compatibility retirement of the
-- Research Theme and Reason Tree publication domain. Stop the old publisher
-- and readers and preserve a reviewed PostgreSQL recovery point first.

DROP TRIGGER IF EXISTS trg_research_theme_receipts_immutable ON research_theme_import_receipts;
DROP TRIGGER IF EXISTS trg_research_themes_immutable ON research_themes;
DROP TRIGGER IF EXISTS trg_research_theme_impacts_immutable ON research_theme_impacts;
DROP TRIGGER IF EXISTS trg_research_theme_events_immutable ON research_theme_events;
DROP TRIGGER IF EXISTS trg_research_reasoning_tree_receipts_immutable ON research_reasoning_tree_import_receipts;
DROP TRIGGER IF EXISTS trg_research_reasoning_trees_immutable ON research_reasoning_trees;
DROP TRIGGER IF EXISTS trg_research_reasoning_tree_events_immutable ON research_reasoning_tree_events;
DROP TRIGGER IF EXISTS trg_research_reasoning_tree_nodes_immutable ON research_reasoning_tree_nodes;
DROP TRIGGER IF EXISTS trg_research_reasoning_tree_node_signals_immutable ON research_reasoning_tree_node_signals;

DROP TABLE research_reasoning_tree_node_signals;
DROP TABLE research_reasoning_tree_nodes;
DROP TABLE research_reasoning_tree_events;
DROP TABLE research_reasoning_trees;
DROP TABLE research_reasoning_tree_import_receipts;
DROP TABLE research_theme_events;
DROP TABLE research_theme_impacts;
DROP TABLE research_themes;
DROP TABLE research_theme_import_receipts;

DROP FUNCTION prevent_research_publication_mutation();

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000078 is a destructive forward-only Research Theme and Reason Tree retirement; restore the reviewed pre-migration snapshot with the previous application releases';
END;
$$;
-- +goose StatementEnd
