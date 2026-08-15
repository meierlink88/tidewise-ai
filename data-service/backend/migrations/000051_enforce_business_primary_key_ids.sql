-- +goose Up
-- Coordinated stop-write migration. Every independent Data Application row
-- keeps its UUID suffix and receives its reviewed object prefix. Legacy
-- sequence keys are deterministically converted and their sequences removed.

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;

-- +goose StatementBegin
CREATE FUNCTION migration_000051_derived_id(prefix TEXT, namespace TEXT, part TEXT)
RETURNS TEXT
LANGUAGE plpgsql
IMMUTABLE
STRICT
AS $$
DECLARE
    value BYTEA;
    encoded TEXT;
BEGIN
    value := substring(
        public.digest(
            uuid_send('6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid)
            || convert_to(prefix, 'UTF8') || '\x00'::bytea
            || convert_to(namespace, 'UTF8') || '\x00'::bytea
            || convert_to(part, 'UTF8'),
            'sha1'
        )
        FROM 1 FOR 16
    );
    value := set_byte(value, 6, (get_byte(value, 6) & 15) | 80);
    value := set_byte(value, 8, (get_byte(value, 8) & 63) | 128);
    encoded := encode(value, 'hex');
    RETURN prefix
        || substring(encoded FROM 1 FOR 8) || '-'
        || substring(encoded FROM 9 FOR 4) || '-'
        || substring(encoded FROM 13 FOR 4) || '-'
        || substring(encoded FROM 17 FOR 4) || '-'
        || substring(encoded FROM 21 FOR 12);
END;
$$;
-- +goose StatementEnd

CREATE TEMP TABLE migration_000051_id_map (
    identity_kind TEXT NOT NULL,
    old_id TEXT NOT NULL,
    new_id VARCHAR(39) NOT NULL,
    PRIMARY KEY (identity_kind, old_id),
    UNIQUE (identity_kind, new_id)
) ON COMMIT DROP;

INSERT INTO migration_000051_id_map
SELECT 'chain_node_physical_constraint', id::text, 'CPC' || id::text FROM chain_node_physical_constraints;
INSERT INTO migration_000051_id_map
SELECT 'chain_node_relation', id::text, 'CNR' || id::text FROM chain_node_relations;
INSERT INTO migration_000051_id_map
SELECT 'country_region_link', id::text, migration_000051_derived_id('CRL', 'country-region-link-legacy', id::text) FROM country_region_links;
INSERT INTO migration_000051_id_map
SELECT 'direct_impact_assertion', id::text, 'DIA' || id::text FROM direct_impact_assertions;
INSERT INTO migration_000051_id_map
SELECT 'entity_external_identifier', id::text, 'EEI' || id::text FROM entity_external_identifiers;
INSERT INTO migration_000051_id_map
SELECT 'event_entity_link', id::text, 'ENL' || id::text FROM event_entity_links;
INSERT INTO migration_000051_id_map
SELECT 'event_publication_receipt', id::text, 'EPR' || id::text FROM event_publication_receipts;
INSERT INTO migration_000051_id_map
SELECT 'event_semantic_candidate_snapshot', id::text, 'ECS' || id::text FROM event_semantic_candidate_snapshots;
INSERT INTO migration_000051_id_map
SELECT 'event_semantic_context_lease', id::text, 'SCL' || id::text FROM event_semantic_context_leases;
INSERT INTO migration_000051_id_map
SELECT 'event_semantic_resolution_binding', id::text, 'ERB' || id::text FROM event_semantic_resolution_bindings;
INSERT INTO migration_000051_id_map
SELECT 'event_semantic_review_snapshot', id::text, 'ERS' || id::text FROM event_semantic_review_snapshots;
INSERT INTO migration_000051_id_map
SELECT 'event_semantic_submission', id::text, 'ESS' || id::text FROM event_semantic_submissions;
INSERT INTO migration_000051_id_map
SELECT 'event_evidence_link', id::text, 'EEL' || id::text FROM event_sources;
INSERT INTO migration_000051_id_map
SELECT 'event_tag_definition', id::text, 'ETD' || id::text FROM event_tag_defs;
INSERT INTO migration_000051_id_map
SELECT 'event_tag_assignment', id::text, 'ETA' || id::text FROM event_tag_maps;
INSERT INTO migration_000051_id_map
SELECT 'event', id::text, 'EVT' || id::text FROM events;
INSERT INTO migration_000051_id_map
SELECT 'industry_chain_graph_edge', id::text, 'IGE' || id::text FROM industry_chain_graph_edges;
INSERT INTO migration_000051_id_map
SELECT 'industry_relationship_import_receipt', id::text, 'IRI' || id::text FROM industry_relationship_import_receipts;
INSERT INTO migration_000051_id_map
SELECT 'organization_membership', id::text, migration_000051_derived_id('OMB', 'organization-membership-legacy', id::text) FROM organization_members;
INSERT INTO migration_000051_id_map
SELECT 'event_evidence_record', id::text, 'EER' || id::text FROM raw_documents;
INSERT INTO migration_000051_id_map
SELECT 'research_reasoning_tree_receipt', id::text, 'RRI' || id::text FROM research_reasoning_tree_import_receipts;
INSERT INTO migration_000051_id_map
SELECT 'research_reasoning_tree_node', id::text, 'RRN' || id::text FROM research_reasoning_tree_nodes;
INSERT INTO migration_000051_id_map
SELECT 'research_reasoning_tree', id::text, 'RRT' || id::text FROM research_reasoning_trees;
INSERT INTO migration_000051_id_map
SELECT 'research_theme_receipt', id::text, 'RTI' || id::text FROM research_theme_import_receipts;
INSERT INTO migration_000051_id_map
SELECT 'research_theme', id::text, 'RTH' || id::text FROM research_themes;
INSERT INTO migration_000051_id_map
SELECT 'variable_signal_measurement', id::text, 'VSM' || id::text FROM variable_signal_measurements;
INSERT INTO migration_000051_id_map
SELECT 'variable_signal', id::text, 'VSG' || id::text FROM variable_signals;

CREATE TEMP TABLE migration_000051_identity_columns (
    table_oid OID NOT NULL,
    column_attnum SMALLINT NOT NULL,
    table_identity TEXT NOT NULL,
    column_name TEXT NOT NULL,
    depth INTEGER NOT NULL,
    PRIMARY KEY (table_oid, column_attnum)
) ON COMMIT DROP;

CREATE TEMP TABLE migration_000051_constraint_definitions (
    table_oid OID NOT NULL,
    table_identity TEXT NOT NULL,
    constraint_name TEXT NOT NULL,
    constraint_definition TEXT NOT NULL,
    PRIMARY KEY (table_oid, constraint_name)
) ON COMMIT DROP;

CREATE TEMP TABLE migration_000051_trigger_definitions (
    table_oid OID NOT NULL,
    table_identity TEXT NOT NULL,
    trigger_name TEXT NOT NULL,
    trigger_definition TEXT NOT NULL,
    PRIMARY KEY (table_oid, trigger_name)
) ON COMMIT DROP;

-- +goose StatementBegin
CREATE FUNCTION migration_000051_rewrite_identity(root_table REGCLASS, root_column TEXT, selected_kind TEXT)
RETURNS VOID
LANGUAGE plpgsql
AS $$
DECLARE
    inserted_count INTEGER;
    item RECORD;
    missing_count BIGINT;
BEGIN
    TRUNCATE migration_000051_identity_columns;
    TRUNCATE migration_000051_constraint_definitions;
    TRUNCATE migration_000051_trigger_definitions;

    INSERT INTO migration_000051_identity_columns
    SELECT relation.oid, attribute.attnum,
           format('%I.%I', namespace.nspname, relation.relname), attribute.attname, 0
    FROM pg_class relation
    JOIN pg_namespace namespace ON namespace.oid = relation.relnamespace
    JOIN pg_attribute attribute ON attribute.attrelid = relation.oid
    WHERE relation.oid = root_table AND attribute.attname = root_column;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'identity root %.% is unavailable', root_table, root_column;
    END IF;

    LOOP
        INSERT INTO migration_000051_identity_columns
        SELECT constraint_row.conrelid, child_key.attnum,
               format('%I.%I', namespace.nspname, relation.relname), attribute.attname, parent.depth + 1
        FROM migration_000051_identity_columns parent
        JOIN pg_constraint constraint_row
          ON constraint_row.contype = 'f' AND constraint_row.confrelid = parent.table_oid
        JOIN LATERAL unnest(constraint_row.confkey) WITH ORDINALITY parent_key(attnum, position)
          ON parent_key.attnum = parent.column_attnum
        JOIN LATERAL unnest(constraint_row.conkey) WITH ORDINALITY child_key(attnum, position)
          ON child_key.position = parent_key.position
        JOIN pg_class relation ON relation.oid = constraint_row.conrelid
        JOIN pg_namespace namespace ON namespace.oid = relation.relnamespace
        JOIN pg_attribute attribute ON attribute.attrelid = relation.oid AND attribute.attnum = child_key.attnum
        ON CONFLICT (table_oid, column_attnum) DO NOTHING;
        GET DIAGNOSTICS inserted_count = ROW_COUNT;
        EXIT WHEN inserted_count = 0;
    END LOOP;

    INSERT INTO migration_000051_constraint_definitions
    SELECT DISTINCT constraint_row.conrelid,
           format('%I.%I', namespace.nspname, relation.relname), constraint_row.conname,
           pg_get_constraintdef(constraint_row.oid)
    FROM migration_000051_identity_columns identity_column
    JOIN pg_constraint constraint_row
      ON constraint_row.contype IN ('f', 'c')
     AND (
         constraint_row.conrelid = identity_column.table_oid
         OR (constraint_row.contype = 'f' AND constraint_row.confrelid = identity_column.table_oid)
     )
    JOIN pg_class relation ON relation.oid = constraint_row.conrelid
    JOIN pg_namespace namespace ON namespace.oid = relation.relnamespace
    ON CONFLICT (table_oid, constraint_name) DO NOTHING;

    INSERT INTO migration_000051_trigger_definitions
    SELECT DISTINCT trigger_row.tgrelid, identity_column.table_identity,
           trigger_row.tgname, pg_get_triggerdef(trigger_row.oid)
    FROM migration_000051_identity_columns identity_column
    JOIN pg_trigger trigger_row
      ON trigger_row.tgrelid = identity_column.table_oid AND NOT trigger_row.tgisinternal;

    FOR item IN SELECT * FROM migration_000051_constraint_definitions ORDER BY table_identity, constraint_name LOOP
        EXECUTE format('ALTER TABLE %s DROP CONSTRAINT %I', item.table_identity, item.constraint_name);
    END LOOP;
    FOR item IN SELECT * FROM migration_000051_trigger_definitions ORDER BY table_identity, trigger_name LOOP
        EXECUTE format('DROP TRIGGER %I ON %s', item.trigger_name, item.table_identity);
    END LOOP;

    FOR item IN SELECT * FROM migration_000051_identity_columns ORDER BY depth DESC, table_identity, column_name LOOP
        EXECUTE format('ALTER TABLE %s ALTER COLUMN %I DROP DEFAULT', item.table_identity, item.column_name);
        EXECUTE format(
            'ALTER TABLE %s ALTER COLUMN %I TYPE VARCHAR(39) USING %I::text',
            item.table_identity, item.column_name, item.column_name
        );
        EXECUTE format(
            'UPDATE %s target SET %I = mapping.new_id FROM migration_000051_id_map mapping '
            'WHERE mapping.identity_kind = %L AND target.%I::text = mapping.old_id',
            item.table_identity, item.column_name, selected_kind, item.column_name
        );
        EXECUTE format(
            'SELECT count(*) FROM %s target WHERE target.%I IS NOT NULL AND NOT EXISTS ('
            'SELECT 1 FROM migration_000051_id_map mapping '
            'WHERE mapping.identity_kind = %L AND mapping.new_id = target.%I::text)',
            item.table_identity, item.column_name, selected_kind, item.column_name
        ) INTO missing_count;
        IF missing_count <> 0 THEN
            RAISE EXCEPTION '% has % unmapped % identities', item.table_identity, missing_count, selected_kind;
        END IF;
    END LOOP;

    FOR item IN SELECT * FROM migration_000051_constraint_definitions ORDER BY table_identity, constraint_name LOOP
        EXECUTE format('ALTER TABLE %s ADD CONSTRAINT %I %s', item.table_identity, item.constraint_name, item.constraint_definition);
    END LOOP;
    FOR item IN SELECT * FROM migration_000051_trigger_definitions ORDER BY table_identity, trigger_name LOOP
        EXECUTE item.trigger_definition;
    END LOOP;
END;
$$;
-- +goose StatementEnd

SELECT migration_000051_rewrite_identity('chain_node_physical_constraints', 'id', 'chain_node_physical_constraint');
SELECT migration_000051_rewrite_identity('chain_node_relations', 'id', 'chain_node_relation');
SELECT migration_000051_rewrite_identity('country_region_links', 'id', 'country_region_link');
SELECT migration_000051_rewrite_identity('direct_impact_assertions', 'id', 'direct_impact_assertion');
SELECT migration_000051_rewrite_identity('entity_external_identifiers', 'id', 'entity_external_identifier');
SELECT migration_000051_rewrite_identity('event_entity_links', 'id', 'event_entity_link');
SELECT migration_000051_rewrite_identity('event_publication_receipts', 'id', 'event_publication_receipt');
SELECT migration_000051_rewrite_identity('event_semantic_candidate_snapshots', 'id', 'event_semantic_candidate_snapshot');
SELECT migration_000051_rewrite_identity('event_semantic_context_leases', 'id', 'event_semantic_context_lease');
SELECT migration_000051_rewrite_identity('event_semantic_resolution_bindings', 'id', 'event_semantic_resolution_binding');
SELECT migration_000051_rewrite_identity('event_semantic_review_snapshots', 'id', 'event_semantic_review_snapshot');
SELECT migration_000051_rewrite_identity('event_semantic_submissions', 'id', 'event_semantic_submission');
SELECT migration_000051_rewrite_identity('event_sources', 'id', 'event_evidence_link');
SELECT migration_000051_rewrite_identity('event_tag_defs', 'id', 'event_tag_definition');
SELECT migration_000051_rewrite_identity('event_tag_maps', 'id', 'event_tag_assignment');
SELECT migration_000051_rewrite_identity('events', 'id', 'event');
SELECT migration_000051_rewrite_identity('industry_chain_graph_edges', 'id', 'industry_chain_graph_edge');
SELECT migration_000051_rewrite_identity('industry_relationship_import_receipts', 'id', 'industry_relationship_import_receipt');
SELECT migration_000051_rewrite_identity('organization_members', 'id', 'organization_membership');
SELECT migration_000051_rewrite_identity('raw_documents', 'id', 'event_evidence_record');
SELECT migration_000051_rewrite_identity('research_reasoning_tree_import_receipts', 'id', 'research_reasoning_tree_receipt');
SELECT migration_000051_rewrite_identity('research_reasoning_tree_nodes', 'id', 'research_reasoning_tree_node');
SELECT migration_000051_rewrite_identity('research_reasoning_trees', 'id', 'research_reasoning_tree');
SELECT migration_000051_rewrite_identity('research_theme_import_receipts', 'id', 'research_theme_receipt');
SELECT migration_000051_rewrite_identity('research_themes', 'id', 'research_theme');
SELECT migration_000051_rewrite_identity('variable_signal_measurements', 'id', 'variable_signal_measurement');
SELECT migration_000051_rewrite_identity('variable_signals', 'id', 'variable_signal');

-- +goose StatementBegin
DO $$
DECLARE
    item RECORD;
BEGIN
    FOR item IN
        SELECT * FROM (VALUES
            ('chain_node_physical_constraints', 'id', 'CPC'),
            ('chain_node_relations', 'id', 'CNR'),
            ('country_region_links', 'id', 'CRL'),
            ('direct_impact_assertions', 'id', 'DIA'),
            ('entity_external_identifiers', 'id', 'EEI'),
            ('event_entity_links', 'id', 'ENL'),
            ('event_publication_receipts', 'id', 'EPR'),
            ('event_semantic_candidate_snapshots', 'id', 'ECS'),
            ('event_semantic_context_leases', 'id', 'SCL'),
            ('event_semantic_resolution_bindings', 'id', 'ERB'),
            ('event_semantic_review_snapshots', 'id', 'ERS'),
            ('event_semantic_submissions', 'id', 'ESS'),
            ('event_sources', 'id', 'EEL'),
            ('event_tag_defs', 'id', 'ETD'),
            ('event_tag_maps', 'id', 'ETA'),
            ('events', 'id', 'EVT'),
            ('industry_chain_graph_edges', 'id', 'IGE'),
            ('industry_relationship_import_receipts', 'id', 'IRI'),
            ('organization_members', 'id', 'OMB'),
            ('raw_documents', 'id', 'EER'),
            ('research_reasoning_tree_import_receipts', 'id', 'RRI'),
            ('research_reasoning_tree_nodes', 'id', 'RRN'),
            ('research_reasoning_trees', 'id', 'RRT'),
            ('research_theme_import_receipts', 'id', 'RTI'),
            ('research_themes', 'id', 'RTH'),
            ('variable_signal_measurements', 'id', 'VSM'),
            ('variable_signals', 'id', 'VSG')
        ) AS identifiers(table_name, column_name, prefix)
    LOOP
        EXECUTE format(
            'ALTER TABLE %I ADD CONSTRAINT %I CHECK (%I ~ %L)',
            item.table_name,
            'chk_' || item.table_name || '_system_id',
            item.column_name,
            '^' || item.prefix || '[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
        );
    END LOOP;
END;
$$;
-- +goose StatementEnd

DROP SEQUENCE IF EXISTS country_region_links_id_seq;
DROP SEQUENCE IF EXISTS organization_members_id_seq;
DROP FUNCTION migration_000051_rewrite_identity(REGCLASS, TEXT, TEXT);
DROP FUNCTION migration_000051_derived_id(TEXT, TEXT, TEXT);

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000051 is forward-only; restore the reviewed pre-migration snapshot';
END;
$$;
-- +goose StatementEnd
