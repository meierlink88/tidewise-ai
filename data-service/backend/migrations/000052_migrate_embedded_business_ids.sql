-- +goose Up
-- Convert embedded collections that cannot participate in PostgreSQL foreign
-- keys but carry the same system-owned identities migrated by 000051.

-- +goose StatementBegin
CREATE FUNCTION migration_000052_prefix_uuid_array(prefix TEXT, identities UUID[])
RETURNS TEXT[]
LANGUAGE sql
IMMUTABLE
STRICT
AS $$
    SELECT COALESCE(array_agg(prefix || value::text ORDER BY position), '{}'::text[])
    FROM unnest(identities) WITH ORDINALITY item(value, position)
$$;
-- +goose StatementEnd

-- PostgreSQL prevents type changes while a trigger definition references the
-- column. Preserve the reviewed trigger contracts around the coordinated rewrite.
DROP TRIGGER trg_event_semantic_measurement_evidence_ids_compat ON variable_signal_measurements;
DROP TRIGGER trg_event_publication_receipts_immutable ON event_publication_receipts;
DROP TRIGGER trg_research_theme_events_immutable ON research_theme_events;
DROP TRIGGER trg_research_reasoning_tree_events_immutable ON research_reasoning_tree_events;
DROP TRIGGER trg_research_theme_receipts_immutable ON research_theme_import_receipts;
DROP TRIGGER trg_research_reasoning_tree_receipts_immutable ON research_reasoning_tree_import_receipts;

ALTER TABLE direct_impact_assertions DROP CONSTRAINT chk_direct_impact_evidence;
ALTER TABLE variable_signals DROP CONSTRAINT chk_variable_signal_evidence;
ALTER TABLE variable_signal_measurements DROP CONSTRAINT chk_variable_signal_measurement_evidence_ids;
ALTER TABLE event_publication_receipts
    DROP CONSTRAINT chk_event_publication_receipts_event_ids,
    DROP CONSTRAINT chk_event_publication_receipts_raw_document_ids,
    DROP CONSTRAINT chk_event_publication_receipts_event_source_ids,
    DROP CONSTRAINT chk_event_publication_receipts_event_tag_map_ids;

ALTER TABLE direct_impact_assertions
    ALTER COLUMN evidence_ids TYPE TEXT[]
    USING migration_000052_prefix_uuid_array('EEL', evidence_ids);
ALTER TABLE event_entity_links
    ALTER COLUMN evidence_ids TYPE TEXT[]
    USING migration_000052_prefix_uuid_array('EEL', evidence_ids);
ALTER TABLE variable_signals
    ALTER COLUMN evidence_ids TYPE TEXT[]
    USING migration_000052_prefix_uuid_array('EEL', evidence_ids);
ALTER TABLE variable_signal_measurements
    ALTER COLUMN evidence_ids TYPE TEXT[]
    USING migration_000052_prefix_uuid_array('EEL', evidence_ids);
ALTER TABLE research_theme_events
    ALTER COLUMN evidence_ids TYPE TEXT[]
    USING migration_000052_prefix_uuid_array('EEL', evidence_ids);
ALTER TABLE research_reasoning_tree_events
    ALTER COLUMN evidence_ids TYPE TEXT[]
    USING migration_000052_prefix_uuid_array('EEL', evidence_ids);

ALTER TABLE event_publication_receipts
    ALTER COLUMN event_ids TYPE TEXT[]
    USING migration_000052_prefix_uuid_array('EVT', event_ids),
    ALTER COLUMN raw_document_ids TYPE TEXT[]
    USING migration_000052_prefix_uuid_array('EER', raw_document_ids),
    ALTER COLUMN event_source_ids TYPE TEXT[]
    USING migration_000052_prefix_uuid_array('EEL', event_source_ids),
    ALTER COLUMN event_tag_map_ids TYPE TEXT[]
    USING migration_000052_prefix_uuid_array('ETA', event_tag_map_ids);

ALTER TABLE direct_impact_assertions ADD CONSTRAINT chk_direct_impact_evidence
    CHECK (cardinality(evidence_ids) > 0 AND array_position(evidence_ids, NULL::text) IS NULL);
ALTER TABLE variable_signals ADD CONSTRAINT chk_variable_signal_evidence
    CHECK (cardinality(evidence_ids) > 0 AND array_position(evidence_ids, NULL::text) IS NULL);
ALTER TABLE variable_signal_measurements ADD CONSTRAINT chk_variable_signal_measurement_evidence_ids
    CHECK (cardinality(evidence_ids) > 0 AND array_position(evidence_ids, NULL::text) IS NULL);
ALTER TABLE event_publication_receipts
    ADD CONSTRAINT chk_event_publication_receipts_event_ids
        CHECK (array_ndims(event_ids) = 1 AND cardinality(event_ids) BETWEEN 1 AND 10 AND array_position(event_ids, NULL::text) IS NULL),
    ADD CONSTRAINT chk_event_publication_receipts_raw_document_ids
        CHECK (array_ndims(raw_document_ids) = 1 AND cardinality(raw_document_ids) >= 1 AND array_position(raw_document_ids, NULL::text) IS NULL),
    ADD CONSTRAINT chk_event_publication_receipts_event_source_ids
        CHECK (array_ndims(event_source_ids) = 1 AND cardinality(event_source_ids) >= 1 AND array_position(event_source_ids, NULL::text) IS NULL),
    ADD CONSTRAINT chk_event_publication_receipts_event_tag_map_ids
        CHECK (array_ndims(event_tag_map_ids) = 1 AND cardinality(event_tag_map_ids) >= 1 AND array_position(event_tag_map_ids, NULL::text) IS NULL);

ALTER TABLE research_theme_import_receipts
    ALTER COLUMN aggregate_theme_id TYPE VARCHAR(39)
    USING CASE WHEN aggregate_theme_id IS NULL THEN NULL ELSE 'RTH' || aggregate_theme_id::text END;

UPDATE research_theme_import_receipts
SET reasoning_tree_ids_by_industry_chain_entity_id = COALESCE((
    SELECT jsonb_object_agg(entry.key, to_jsonb('RRT' || trim(both '"' from entry.value::text)))
    FROM jsonb_each(reasoning_tree_ids_by_industry_chain_entity_id) entry
), '{}'::jsonb);

UPDATE research_reasoning_tree_import_receipts
SET reasoning_tree_ids_by_industry_chain_entity_id = COALESCE((
    SELECT jsonb_object_agg(entry.key, to_jsonb('RRT' || trim(both '"' from entry.value::text)))
    FROM jsonb_each(reasoning_tree_ids_by_industry_chain_entity_id) entry
), '{}'::jsonb);

CREATE TRIGGER trg_research_theme_receipts_immutable
BEFORE DELETE OR UPDATE OR TRUNCATE ON research_theme_import_receipts
FOR EACH STATEMENT EXECUTE FUNCTION prevent_research_publication_mutation();
CREATE TRIGGER trg_research_reasoning_tree_receipts_immutable
BEFORE DELETE OR UPDATE OR TRUNCATE ON research_reasoning_tree_import_receipts
FOR EACH STATEMENT EXECUTE FUNCTION prevent_research_publication_mutation();
CREATE TRIGGER trg_research_theme_events_immutable
BEFORE DELETE OR UPDATE OR TRUNCATE ON research_theme_events
FOR EACH STATEMENT EXECUTE FUNCTION prevent_research_publication_mutation();
CREATE TRIGGER trg_research_reasoning_tree_events_immutable
BEFORE DELETE OR UPDATE OR TRUNCATE ON research_reasoning_tree_events
FOR EACH STATEMENT EXECUTE FUNCTION prevent_research_publication_mutation();
CREATE TRIGGER trg_event_publication_receipts_immutable
BEFORE UPDATE OR DELETE OR TRUNCATE ON event_publication_receipts
FOR EACH STATEMENT EXECUTE FUNCTION prevent_event_publication_receipt_mutation();
CREATE TRIGGER trg_event_semantic_measurement_evidence_ids_compat
BEFORE INSERT OR UPDATE OF evidence_id, evidence_ids ON variable_signal_measurements
FOR EACH ROW EXECUTE FUNCTION event_semantic_measurement_evidence_ids_compat();

DROP FUNCTION migration_000052_prefix_uuid_array(TEXT, UUID[]);

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000052 is forward-only; restore the reviewed pre-migration snapshot';
END;
$$;
-- +goose StatementEnd
