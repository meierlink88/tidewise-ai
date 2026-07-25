-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF current_setting('tidewise.phase_a_cleanup_write_authorized', true) IS DISTINCT FROM 'reviewed_backup_verified' THEN
        RAISE EXCEPTION 'cleanup write is not authorized; complete Review and backup verification before applying migration 000015';
    END IF;
END;
$$;
-- +goose StatementEnd

CREATE TEMP TABLE phase_a_retired_entity_ids ON COMMIT DROP AS
SELECT id
FROM entity_nodes
WHERE entity_type IN ('sector', 'industry_chain', 'chain_node');

DELETE FROM event_entity_links
WHERE entity_id IN (SELECT id FROM phase_a_retired_entity_ids);

DELETE FROM entity_edges
WHERE from_entity_id IN (SELECT id FROM phase_a_retired_entity_ids)
   OR to_entity_id IN (SELECT id FROM phase_a_retired_entity_ids);

DROP TABLE entity_convergence_reference_moves;
DROP TABLE entity_convergence_alias_moves;
DROP TABLE entity_convergences;
DROP TABLE entity_convergence_manifests;
DROP FUNCTION prevent_entity_convergence_audit_mutation();

DROP TABLE industry_chain_physical_constraints;
DROP TABLE industry_chain_topology_edges;
DROP TABLE industry_chain_memberships;
DROP TABLE industry_chain_profiles;

DROP TABLE sector_source_mappings;
DROP TABLE sector_profiles;

DELETE FROM chain_node_profiles
WHERE entity_id IN (SELECT id FROM phase_a_retired_entity_ids);

ALTER TABLE chain_node_profiles
    DROP COLUMN chain_position,
    DROP COLUMN node_category,
    DROP COLUMN unit_of_analysis,
    DROP COLUMN granularity_note,
    ALTER COLUMN definition DROP DEFAULT,
    ALTER COLUMN definition SET NOT NULL,
    ADD COLUMN boundary_note TEXT,
    ADD CONSTRAINT chk_chain_node_definition_nonblank CHECK (btrim(definition) <> ''),
    ADD CONSTRAINT chk_chain_node_boundary_nonblank CHECK (boundary_note IS NULL OR btrim(boundary_note) <> '');

-- +goose StatementBegin
DO $$
DECLARE
    ref RECORD;
    reference_count BIGINT;
BEGIN
    FOR ref IN
        SELECT
            source_namespace.nspname AS schema_name,
            source_table.relname AS table_name,
            source_column.attname AS column_name
        FROM pg_constraint fk
        JOIN pg_class source_table ON source_table.oid = fk.conrelid
        JOIN pg_namespace source_namespace ON source_namespace.oid = source_table.relnamespace
        JOIN pg_class target_table ON target_table.oid = fk.confrelid
        JOIN pg_namespace target_namespace ON target_namespace.oid = target_table.relnamespace
        JOIN LATERAL unnest(fk.conkey) WITH ORDINALITY source_key(attnum, ordinal) ON TRUE
        JOIN LATERAL unnest(fk.confkey) WITH ORDINALITY target_key(attnum, ordinal) ON target_key.ordinal = source_key.ordinal
        JOIN pg_attribute source_column ON source_column.attrelid = source_table.oid AND source_column.attnum = source_key.attnum
        JOIN pg_attribute target_column ON target_column.attrelid = target_table.oid AND target_column.attnum = target_key.attnum
        WHERE fk.contype = 'f'
          AND target_namespace.nspname = current_schema()
          AND target_table.relname = 'entity_nodes'
          AND target_column.attname = 'id'
          AND array_length(fk.conkey, 1) = 1
    LOOP
        EXECUTE format(
            'SELECT count(*) FROM %I.%I WHERE %I IN (SELECT id FROM phase_a_retired_entity_ids)',
            ref.schema_name,
            ref.table_name,
            ref.column_name
        ) INTO reference_count;
        IF reference_count > 0 THEN
            RAISE EXCEPTION 'unexpected reference %.% column % still points to % retired industry entities',
                ref.schema_name,
                ref.table_name,
                ref.column_name,
                reference_count;
        END IF;
    END LOOP;
END;
$$;
-- +goose StatementEnd

DELETE FROM entity_nodes
WHERE id IN (SELECT id FROM phase_a_retired_entity_ids);

CREATE TABLE theme_profiles (
    entity_id UUID PRIMARY KEY REFERENCES entity_nodes(id),
    definition TEXT NOT NULL,
    boundary_note TEXT NOT NULL,
    CONSTRAINT chk_theme_definition_nonblank CHECK (btrim(definition) <> ''),
    CONSTRAINT chk_theme_boundary_nonblank CHECK (btrim(boundary_note) <> '')
);

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION 'migration 000015 is irreversible; restore the reviewed backup or apply a forward-fix migration';
END;
$$;
-- +goose StatementEnd
