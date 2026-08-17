-- +goose Up
-- Replace the legacy shared ENT prefix on four independent objects while
-- preserving every canonical UUID suffix and all supported references.

CREATE TEMP TABLE migration_000058_id_rewrite (
    old_id TEXT PRIMARY KEY,
    new_id TEXT NOT NULL UNIQUE,
    object_type TEXT NOT NULL CHECK (object_type IN ('industry', 'concept', 'chain_node', 'industry_chain'))
) ON COMMIT DROP;

INSERT INTO migration_000058_id_rewrite (old_id, new_id, object_type)
SELECT id, 'IND' || substring(id FROM 4), 'industry' FROM industry
UNION ALL
SELECT id, 'CON' || substring(id FROM 4), 'concept' FROM concept
UNION ALL
SELECT id, 'CND' || substring(id FROM 4), 'chain_node' FROM chain_node
UNION ALL
SELECT id, 'ICH' || substring(id FROM 4), 'industry_chain' FROM industry_chain;

-- Detect key collisions recursively before jsonb_object_agg can collapse two
-- distinct business values into one rewritten key.
-- +goose StatementBegin
CREATE FUNCTION migration_000058_jsonb_key_collision(input JSONB)
RETURNS BOOLEAN
LANGUAGE plpgsql
STABLE
STRICT
AS $$
DECLARE
    child JSONB;
BEGIN
    CASE jsonb_typeof(input)
        WHEN 'object' THEN
            IF EXISTS (
                SELECT 1
                FROM jsonb_object_keys(input) entry(key)
                JOIN migration_000058_id_rewrite rewrite ON rewrite.old_id = entry.key
                WHERE input ? rewrite.new_id
            ) THEN
                RETURN TRUE;
            END IF;
            FOR child IN SELECT value FROM jsonb_each(input) LOOP
                IF migration_000058_jsonb_key_collision(child) THEN
                    RETURN TRUE;
                END IF;
            END LOOP;
        WHEN 'array' THEN
            FOR child IN SELECT value FROM jsonb_array_elements(input) LOOP
                IF migration_000058_jsonb_key_collision(child) THEN
                    RETURN TRUE;
                END IF;
            END LOOP;
        ELSE
            NULL;
    END CASE;
    RETURN FALSE;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE
    column_record RECORD;
    contains_collision BOOLEAN;
BEGIN
    IF EXISTS (
        SELECT 1
        FROM migration_000058_id_rewrite
        WHERE old_id !~ '^ENT[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
           OR substring(old_id FROM 4) <> substring(new_id FROM 4)
           OR (object_type = 'industry' AND new_id !~ '^IND[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$')
           OR (object_type = 'concept' AND new_id !~ '^CON[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$')
           OR (object_type = 'chain_node' AND new_id !~ '^CND[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$')
           OR (object_type = 'industry_chain' AND new_id !~ '^ICH[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$')
    ) THEN
        RAISE EXCEPTION 'independent object IDs must start with legacy ENT and retain canonical UUID suffixes';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM migration_000058_id_rewrite rewrite
        WHERE data_object_type(rewrite.new_id) IS NOT NULL
    ) THEN
        RAISE EXCEPTION 'an independent object target identity already belongs to a Data object';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM research_theme_import_receipts receipt
        CROSS JOIN LATERAL jsonb_object_keys(receipt.reasoning_tree_ids_by_industry_chain_id) entry(key)
        LEFT JOIN migration_000058_id_rewrite rewrite
          ON rewrite.object_type = 'industry_chain'
         AND (rewrite.old_id = entry.key OR substring(rewrite.old_id FROM 4) = entry.key)
        WHERE rewrite.old_id IS NULL
        UNION ALL
        SELECT 1
        FROM research_reasoning_tree_import_receipts receipt
        CROSS JOIN LATERAL jsonb_object_keys(receipt.reasoning_tree_ids_by_industry_chain_id) entry(key)
        LEFT JOIN migration_000058_id_rewrite rewrite
          ON rewrite.object_type = 'industry_chain'
         AND (rewrite.old_id = entry.key OR substring(rewrite.old_id FROM 4) = entry.key)
        WHERE rewrite.old_id IS NULL
    ) THEN
        RAISE EXCEPTION 'Research receipt contains an unknown IndustryChain identity';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM (
            SELECT 'theme' receipt_type, receipt.id, rewrite.new_id, count(*) occurrences
            FROM research_theme_import_receipts receipt
            CROSS JOIN LATERAL jsonb_object_keys(receipt.reasoning_tree_ids_by_industry_chain_id) entry(key)
            JOIN migration_000058_id_rewrite rewrite
              ON rewrite.object_type = 'industry_chain'
             AND (rewrite.old_id = entry.key OR substring(rewrite.old_id FROM 4) = entry.key)
            GROUP BY receipt.id, rewrite.new_id
            UNION ALL
            SELECT 'tree', receipt.id, rewrite.new_id, count(*)
            FROM research_reasoning_tree_import_receipts receipt
            CROSS JOIN LATERAL jsonb_object_keys(receipt.reasoning_tree_ids_by_industry_chain_id) entry(key)
            JOIN migration_000058_id_rewrite rewrite
              ON rewrite.object_type = 'industry_chain'
             AND (rewrite.old_id = entry.key OR substring(rewrite.old_id FROM 4) = entry.key)
            GROUP BY receipt.id, rewrite.new_id
        ) mapped
        WHERE mapped.occurrences > 1
    ) THEN
        RAISE EXCEPTION 'Research receipt contains colliding legacy IndustryChain identities';
    END IF;

    FOR column_record IN
        SELECT column_value.table_name, column_value.column_name
        FROM information_schema.columns column_value
        JOIN information_schema.tables table_value
          ON table_value.table_schema = column_value.table_schema
         AND table_value.table_name = column_value.table_name
        WHERE column_value.table_schema = current_schema()
          AND column_value.data_type = 'jsonb'
          AND table_value.table_type = 'BASE TABLE'
        ORDER BY column_value.table_name, column_value.ordinal_position
    LOOP
        EXECUTE format(
            'SELECT EXISTS (SELECT 1 FROM %I.%I target WHERE target.%I IS NOT NULL '
            'AND migration_000058_jsonb_key_collision(target.%I))',
            current_schema(), column_record.table_name, column_record.column_name,
            column_record.column_name
        ) INTO contains_collision;
        IF contains_collision THEN
            RAISE EXCEPTION 'independent object JSON key rewrite collision in %.%',
                column_record.table_name, column_record.column_name;
        END IF;
    END LOOP;
END;
$$;
-- +goose StatementEnd

-- Rewrite exact object IDs wherever they occur as JSON object keys or string
-- values. This covers retained Event Semantic audit snapshots in addition to
-- the two formal Research receipt maps.
-- +goose StatementBegin
CREATE FUNCTION migration_000058_rewrite_jsonb(input JSONB)
RETURNS JSONB
LANGUAGE plpgsql
STABLE
STRICT
AS $$
DECLARE
    result JSONB;
    scalar_value TEXT;
    rewritten_value TEXT;
BEGIN
    CASE jsonb_typeof(input)
        WHEN 'object' THEN
            SELECT COALESCE(
                jsonb_object_agg(COALESCE(rewrite.new_id, entry.key), migration_000058_rewrite_jsonb(entry.value)),
                '{}'::jsonb
            )
            INTO result
            FROM jsonb_each(input) entry
            LEFT JOIN migration_000058_id_rewrite rewrite ON rewrite.old_id = entry.key;
            RETURN result;
        WHEN 'array' THEN
            SELECT COALESCE(
                jsonb_agg(migration_000058_rewrite_jsonb(entry.value) ORDER BY entry.position),
                '[]'::jsonb
            )
            INTO result
            FROM jsonb_array_elements(input) WITH ORDINALITY entry(value, position);
            RETURN result;
        WHEN 'string' THEN
            scalar_value := input #>> '{}';
            SELECT rewrite.new_id INTO rewritten_value
            FROM migration_000058_id_rewrite rewrite
            WHERE rewrite.old_id = scalar_value;
            IF FOUND THEN
                RETURN to_jsonb(rewritten_value);
            END IF;
            RETURN input;
        ELSE
            RETURN input;
    END CASE;
END;
$$;
-- +goose StatementEnd

-- Every scalar reference that this migration is allowed to rewrite is listed
-- explicitly. An exact legacy object ID in any other scalar/array column means
-- a new reference owner was added without a reviewed migration rule, so fail
-- before the first durable UPDATE.
CREATE TEMP TABLE migration_000058_supported_scalar_references (
    table_name TEXT NOT NULL,
    column_name TEXT NOT NULL,
    PRIMARY KEY (table_name, column_name)
) ON COMMIT DROP;

INSERT INTO migration_000058_supported_scalar_references (table_name, column_name) VALUES
    ('industry', 'id'),
    ('industry', 'parent_industry_id'),
    ('concept', 'id'),
    ('chain_node', 'id'),
    ('industry_chain', 'id'),
    ('industry_chain_node_memberships', 'industry_chain_id'),
    ('industry_chain_node_memberships', 'chain_node_id'),
    ('industry_chain_graph_edges', 'industry_chain_id'),
    ('industry_chain_graph_edges', 'from_chain_node_id'),
    ('industry_chain_graph_edges', 'to_chain_node_id'),
    ('chain_node_relations', 'from_chain_node_id'),
    ('chain_node_relations', 'to_chain_node_id'),
    ('chain_node_physical_constraints', 'chain_node_id'),
    ('research_theme_impacts', 'chain_node_id'),
    ('research_reasoning_trees', 'industry_chain_id'),
    ('research_reasoning_tree_nodes', 'chain_node_id'),
    ('entity_edges', 'from_entity_id'),
    ('entity_edges', 'to_entity_id'),
    ('entity_external_identifiers', 'entity_id'),
    ('event_entity_links', 'entity_id'),
    ('entity_redirects', 'source_entity_id'),
    ('entity_redirects', 'target_entity_id'),
    ('direct_impact_assertions', 'target_entity_id'),
    ('event_semantic_resolution_bindings', 'anchor_entity_id'),
    ('event_semantic_resolution_bindings', 'target_entity_id');

-- +goose StatementBegin
DO $$
DECLARE
    column_record RECORD;
    contains_unreviewed_reference BOOLEAN;
BEGIN
    FOR column_record IN
        SELECT column_value.table_name, column_value.column_name,
               column_value.data_type, column_value.udt_name
        FROM information_schema.columns column_value
        JOIN information_schema.tables table_value
          ON table_value.table_schema = column_value.table_schema
         AND table_value.table_name = column_value.table_name
        LEFT JOIN migration_000058_supported_scalar_references supported
          ON supported.table_name = column_value.table_name
         AND supported.column_name = column_value.column_name
        WHERE column_value.table_schema = current_schema()
          AND table_value.table_type = 'BASE TABLE'
          AND supported.table_name IS NULL
          AND (column_value.data_type IN ('text', 'character varying', 'character')
               OR column_value.udt_name = '_text')
        ORDER BY column_value.table_name, column_value.ordinal_position
    LOOP
        IF column_record.udt_name = '_text' THEN
            EXECUTE format(
                'SELECT EXISTS (SELECT 1 FROM %I.%I target '
                'CROSS JOIN LATERAL unnest(target.%I) value '
                'JOIN migration_000058_id_rewrite rewrite ON rewrite.old_id = value)',
                current_schema(), column_record.table_name, column_record.column_name
            ) INTO contains_unreviewed_reference;
        ELSE
            EXECUTE format(
                'SELECT EXISTS (SELECT 1 FROM %I.%I target '
                'JOIN migration_000058_id_rewrite rewrite ON rewrite.old_id = target.%I)',
                current_schema(), column_record.table_name, column_record.column_name
            ) INTO contains_unreviewed_reference;
        END IF;
        IF contains_unreviewed_reference THEN
            RAISE EXCEPTION 'unreviewed independent object reference in %.%',
                column_record.table_name, column_record.column_name;
        END IF;
    END LOOP;
END;
$$;
-- +goose StatementEnd

CREATE TEMP TABLE migration_000058_affected_tables (
    table_name TEXT PRIMARY KEY
) ON COMMIT DROP;

INSERT INTO migration_000058_affected_tables (table_name) VALUES
    ('industry'),
    ('concept'),
    ('chain_node'),
    ('industry_chain'),
    ('industry_chain_node_memberships'),
    ('industry_chain_graph_edges'),
    ('chain_node_relations'),
    ('chain_node_physical_constraints'),
    ('research_theme_impacts'),
    ('research_reasoning_trees'),
    ('research_reasoning_tree_nodes'),
    ('entity_edges'),
    ('entity_external_identifiers'),
    ('event_entity_links'),
    ('entity_redirects'),
    ('direct_impact_assertions'),
    ('event_semantic_resolution_bindings');

INSERT INTO migration_000058_affected_tables (table_name)
SELECT DISTINCT column_value.table_name
FROM information_schema.columns column_value
JOIN information_schema.tables table_value
  ON table_value.table_schema = column_value.table_schema
 AND table_value.table_name = column_value.table_name
WHERE column_value.table_schema = current_schema()
  AND column_value.data_type = 'jsonb'
  AND table_value.table_type = 'BASE TABLE'
ON CONFLICT DO NOTHING;

CREATE TEMP TABLE migration_000058_trigger_state (
    table_name TEXT NOT NULL,
    trigger_name TEXT NOT NULL,
    enabled "char" NOT NULL,
    PRIMARY KEY (table_name, trigger_name)
) ON COMMIT DROP;

INSERT INTO migration_000058_trigger_state (table_name, trigger_name, enabled)
SELECT table_value.relname, trigger_value.tgname, trigger_value.tgenabled
FROM pg_trigger trigger_value
JOIN pg_class table_value ON table_value.oid = trigger_value.tgrelid
JOIN pg_namespace schema_value ON schema_value.oid = table_value.relnamespace
JOIN migration_000058_affected_tables affected ON affected.table_name = table_value.relname
WHERE schema_value.nspname = current_schema()
  AND NOT trigger_value.tgisinternal;

-- +goose StatementBegin
DO $$
DECLARE
    trigger_value RECORD;
BEGIN
    FOR trigger_value IN SELECT * FROM migration_000058_trigger_state LOOP
        EXECUTE format(
			'ALTER TABLE %I.%I DISABLE TRIGGER %I',
			current_schema(),
            trigger_value.table_name,
            trigger_value.trigger_name
        );
    END LOOP;
END;
$$;
-- +goose StatementEnd

ALTER TABLE industry DROP CONSTRAINT chk_industry_identity, DROP CONSTRAINT fk_industry_parent;
ALTER TABLE concept DROP CONSTRAINT chk_concept_identity;
ALTER TABLE chain_node DROP CONSTRAINT chk_chain_node_identity;
ALTER TABLE industry_chain DROP CONSTRAINT chk_industry_chain_identity;
ALTER TABLE industry_chain_node_memberships
    DROP CONSTRAINT industry_chain_node_memberships_chain_node_entity_id_fkey,
    DROP CONSTRAINT industry_chain_node_memberships_industry_chain_entity_id_fkey;
ALTER TABLE industry_chain_graph_edges
    DROP CONSTRAINT fk_industry_chain_graph_from_membership,
    DROP CONSTRAINT fk_industry_chain_graph_to_membership;
ALTER TABLE chain_node_relations
    DROP CONSTRAINT chain_node_relations_from_chain_node_entity_id_fkey,
    DROP CONSTRAINT chain_node_relations_to_chain_node_entity_id_fkey;
ALTER TABLE chain_node_physical_constraints
    DROP CONSTRAINT chain_node_physical_constraints_chain_node_entity_id_fkey;
ALTER TABLE research_theme_impacts
    DROP CONSTRAINT research_theme_impacts_chain_node_entity_id_fkey;
ALTER TABLE research_reasoning_trees
    DROP CONSTRAINT research_reasoning_trees_industry_chain_entity_id_fkey;
ALTER TABLE research_reasoning_tree_nodes
    DROP CONSTRAINT research_reasoning_tree_nodes_chain_node_entity_id_fkey;

UPDATE entity_edges target SET from_entity_id = rewrite.new_id
FROM migration_000058_id_rewrite rewrite WHERE target.from_entity_id = rewrite.old_id;
UPDATE entity_edges target SET to_entity_id = rewrite.new_id
FROM migration_000058_id_rewrite rewrite WHERE target.to_entity_id = rewrite.old_id;
UPDATE entity_external_identifiers target SET entity_id = rewrite.new_id
FROM migration_000058_id_rewrite rewrite WHERE target.entity_id = rewrite.old_id;
UPDATE event_entity_links target SET entity_id = rewrite.new_id
FROM migration_000058_id_rewrite rewrite WHERE target.entity_id = rewrite.old_id;
UPDATE entity_redirects target SET source_entity_id = rewrite.new_id
FROM migration_000058_id_rewrite rewrite WHERE target.source_entity_id = rewrite.old_id;
UPDATE entity_redirects target SET target_entity_id = rewrite.new_id
FROM migration_000058_id_rewrite rewrite WHERE target.target_entity_id = rewrite.old_id;
UPDATE direct_impact_assertions target SET target_entity_id = rewrite.new_id
FROM migration_000058_id_rewrite rewrite WHERE target.target_entity_id = rewrite.old_id;
UPDATE event_semantic_resolution_bindings target SET anchor_entity_id = rewrite.new_id
FROM migration_000058_id_rewrite rewrite WHERE target.anchor_entity_id = rewrite.old_id;
UPDATE event_semantic_resolution_bindings target SET target_entity_id = rewrite.new_id
FROM migration_000058_id_rewrite rewrite WHERE target.target_entity_id = rewrite.old_id;

UPDATE industry target SET parent_industry_id = rewrite.new_id
FROM migration_000058_id_rewrite rewrite WHERE target.parent_industry_id = rewrite.old_id;
UPDATE industry_chain_node_memberships target SET industry_chain_id = rewrite.new_id
FROM migration_000058_id_rewrite rewrite WHERE target.industry_chain_id = rewrite.old_id;
UPDATE industry_chain_node_memberships target SET chain_node_id = rewrite.new_id
FROM migration_000058_id_rewrite rewrite WHERE target.chain_node_id = rewrite.old_id;
UPDATE industry_chain_graph_edges target SET industry_chain_id = rewrite.new_id
FROM migration_000058_id_rewrite rewrite WHERE target.industry_chain_id = rewrite.old_id;
UPDATE industry_chain_graph_edges target SET from_chain_node_id = rewrite.new_id
FROM migration_000058_id_rewrite rewrite WHERE target.from_chain_node_id = rewrite.old_id;
UPDATE industry_chain_graph_edges target SET to_chain_node_id = rewrite.new_id
FROM migration_000058_id_rewrite rewrite WHERE target.to_chain_node_id = rewrite.old_id;
UPDATE chain_node_relations target SET from_chain_node_id = rewrite.new_id
FROM migration_000058_id_rewrite rewrite WHERE target.from_chain_node_id = rewrite.old_id;
UPDATE chain_node_relations target SET to_chain_node_id = rewrite.new_id
FROM migration_000058_id_rewrite rewrite WHERE target.to_chain_node_id = rewrite.old_id;
UPDATE chain_node_physical_constraints target SET chain_node_id = rewrite.new_id
FROM migration_000058_id_rewrite rewrite WHERE target.chain_node_id = rewrite.old_id;
UPDATE research_theme_impacts target SET chain_node_id = rewrite.new_id
FROM migration_000058_id_rewrite rewrite WHERE target.chain_node_id = rewrite.old_id;
UPDATE research_reasoning_trees target SET industry_chain_id = rewrite.new_id
FROM migration_000058_id_rewrite rewrite WHERE target.industry_chain_id = rewrite.old_id;
UPDATE research_reasoning_tree_nodes target SET chain_node_id = rewrite.new_id
FROM migration_000058_id_rewrite rewrite WHERE target.chain_node_id = rewrite.old_id;

-- Migration 000050 prefixed owner identities but intentionally retained the
-- original UUID-only keys in immutable Research audit receipts. Repair those
-- reviewed keys explicitly; do not treat arbitrary bare UUIDs in other JSON as
-- Data object references.
UPDATE research_theme_import_receipts receipt
SET reasoning_tree_ids_by_industry_chain_id = (
    SELECT COALESCE(jsonb_object_agg(COALESCE(rewrite.new_id, entry.key), entry.value), '{}'::jsonb) value
    FROM jsonb_each(receipt.reasoning_tree_ids_by_industry_chain_id) entry
    LEFT JOIN migration_000058_id_rewrite rewrite
      ON rewrite.object_type = 'industry_chain'
     AND (rewrite.old_id = entry.key OR substring(rewrite.old_id FROM 4) = entry.key)
);

UPDATE research_reasoning_tree_import_receipts receipt
SET reasoning_tree_ids_by_industry_chain_id = (
    SELECT COALESCE(jsonb_object_agg(COALESCE(rewrite.new_id, entry.key), entry.value), '{}'::jsonb) value
    FROM jsonb_each(receipt.reasoning_tree_ids_by_industry_chain_id) entry
    LEFT JOIN migration_000058_id_rewrite rewrite
      ON rewrite.object_type = 'industry_chain'
     AND (rewrite.old_id = entry.key OR substring(rewrite.old_id FROM 4) = entry.key)
);

-- +goose StatementBegin
DO $$
DECLARE
	column_record RECORD;
BEGIN
	FOR column_record IN
        SELECT column_value.table_name, column_value.column_name
        FROM information_schema.columns column_value
        JOIN information_schema.tables table_value
          ON table_value.table_schema = column_value.table_schema
         AND table_value.table_name = column_value.table_name
		WHERE column_value.table_schema = current_schema()
          AND column_value.data_type = 'jsonb'
          AND table_value.table_type = 'BASE TABLE'
        ORDER BY column_value.table_name, column_value.ordinal_position
    LOOP
        EXECUTE format(
			'WITH rewritten AS MATERIALIZED ('
			'SELECT tableoid, ctid, migration_000058_rewrite_jsonb(%I) value '
			'FROM %I.%I WHERE %I IS NOT NULL) '
			'UPDATE %I.%I target SET %I = rewritten.value FROM rewritten '
			'WHERE target.tableoid = rewritten.tableoid AND target.ctid = rewritten.ctid '
			'AND target.%I IS DISTINCT FROM rewritten.value',
			column_record.column_name,
			current_schema(),
			column_record.table_name,
			column_record.column_name,
			current_schema(),
			column_record.table_name,
			column_record.column_name,
			column_record.column_name
        );
    END LOOP;
END;
$$;
-- +goose StatementEnd

UPDATE industry target SET id = rewrite.new_id
FROM migration_000058_id_rewrite rewrite
WHERE target.id = rewrite.old_id AND rewrite.object_type = 'industry';
UPDATE concept target SET id = rewrite.new_id
FROM migration_000058_id_rewrite rewrite
WHERE target.id = rewrite.old_id AND rewrite.object_type = 'concept';
UPDATE chain_node target SET id = rewrite.new_id
FROM migration_000058_id_rewrite rewrite
WHERE target.id = rewrite.old_id AND rewrite.object_type = 'chain_node';
UPDATE industry_chain target SET id = rewrite.new_id
FROM migration_000058_id_rewrite rewrite
WHERE target.id = rewrite.old_id AND rewrite.object_type = 'industry_chain';

ALTER TABLE industry
    ADD CONSTRAINT chk_industry_identity CHECK (
        id ~ '^IND[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    ),
    ADD CONSTRAINT fk_industry_parent
        FOREIGN KEY (parent_industry_id) REFERENCES industry(id) ON DELETE RESTRICT;
ALTER TABLE concept
    ADD CONSTRAINT chk_concept_identity CHECK (
        id ~ '^CON[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    );
ALTER TABLE chain_node
    ADD CONSTRAINT chk_chain_node_identity CHECK (
        id ~ '^CND[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    );
ALTER TABLE industry_chain
    ADD CONSTRAINT chk_industry_chain_identity CHECK (
        id ~ '^ICH[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    );

ALTER TABLE industry_chain_node_memberships
    ADD CONSTRAINT industry_chain_node_memberships_chain_node_entity_id_fkey
        FOREIGN KEY (chain_node_id) REFERENCES chain_node(id) ON DELETE RESTRICT,
    ADD CONSTRAINT industry_chain_node_memberships_industry_chain_entity_id_fkey
        FOREIGN KEY (industry_chain_id) REFERENCES industry_chain(id) ON DELETE RESTRICT;
ALTER TABLE industry_chain_graph_edges
    ADD CONSTRAINT fk_industry_chain_graph_from_membership
        FOREIGN KEY (industry_chain_id, from_chain_node_id)
        REFERENCES industry_chain_node_memberships(industry_chain_id, chain_node_id) ON DELETE RESTRICT,
    ADD CONSTRAINT fk_industry_chain_graph_to_membership
        FOREIGN KEY (industry_chain_id, to_chain_node_id)
        REFERENCES industry_chain_node_memberships(industry_chain_id, chain_node_id) ON DELETE RESTRICT;
ALTER TABLE chain_node_relations
    ADD CONSTRAINT chain_node_relations_from_chain_node_entity_id_fkey
        FOREIGN KEY (from_chain_node_id) REFERENCES chain_node(id) ON DELETE RESTRICT,
    ADD CONSTRAINT chain_node_relations_to_chain_node_entity_id_fkey
        FOREIGN KEY (to_chain_node_id) REFERENCES chain_node(id) ON DELETE RESTRICT;
ALTER TABLE chain_node_physical_constraints
    ADD CONSTRAINT chain_node_physical_constraints_chain_node_entity_id_fkey
        FOREIGN KEY (chain_node_id) REFERENCES chain_node(id) ON DELETE RESTRICT;
ALTER TABLE research_theme_impacts
    ADD CONSTRAINT research_theme_impacts_chain_node_entity_id_fkey
        FOREIGN KEY (chain_node_id) REFERENCES chain_node(id);
ALTER TABLE research_reasoning_trees
    ADD CONSTRAINT research_reasoning_trees_industry_chain_entity_id_fkey
        FOREIGN KEY (industry_chain_id) REFERENCES industry_chain(id);
ALTER TABLE research_reasoning_tree_nodes
    ADD CONSTRAINT research_reasoning_tree_nodes_chain_node_entity_id_fkey
        FOREIGN KEY (chain_node_id) REFERENCES chain_node(id);

-- Fail the transaction if any mapped legacy identity survived in a textual,
-- array, or JSONB value that is not part of the explicitly reviewed rewrite.
-- +goose StatementBegin
DO $$
DECLARE
	column_record RECORD;
	contains_legacy BOOLEAN;
BEGIN
	FOR column_record IN
		SELECT column_value.table_name, column_value.column_name,
		       column_value.data_type, column_value.udt_name
        FROM information_schema.columns column_value
        JOIN information_schema.tables table_value
          ON table_value.table_schema = column_value.table_schema
         AND table_value.table_name = column_value.table_name
		WHERE column_value.table_schema = current_schema()
          AND table_value.table_type = 'BASE TABLE'
		  AND (column_value.data_type IN ('text', 'character varying', 'character', 'jsonb')
		       OR column_value.udt_name = '_text')
        ORDER BY column_value.table_name, column_value.ordinal_position
    LOOP
		IF column_record.data_type = 'jsonb' THEN
			EXECUTE format(
				'SELECT EXISTS (SELECT 1 FROM %I.%I target '
				'WHERE target.%I IS NOT NULL '
				'AND migration_000058_rewrite_jsonb(target.%I) IS DISTINCT FROM target.%I)',
				current_schema(), column_record.table_name, column_record.column_name,
				column_record.column_name, column_record.column_name
			) INTO contains_legacy;
		ELSIF column_record.udt_name = '_text' THEN
			EXECUTE format(
				'SELECT EXISTS (SELECT 1 FROM %I.%I target '
				'CROSS JOIN LATERAL unnest(target.%I) value '
				'JOIN migration_000058_id_rewrite rewrite ON rewrite.old_id = value)',
				current_schema(), column_record.table_name, column_record.column_name
			) INTO contains_legacy;
		ELSE
			EXECUTE format(
				'SELECT EXISTS (SELECT 1 FROM %I.%I target '
				'JOIN migration_000058_id_rewrite rewrite ON rewrite.old_id = target.%I)',
				current_schema(), column_record.table_name, column_record.column_name
			) INTO contains_legacy;
		END IF;
		IF contains_legacy THEN
			RAISE EXCEPTION 'legacy independent object identity remains in %.%',
				column_record.table_name, column_record.column_name;
        END IF;
    END LOOP;

    IF EXISTS (
        SELECT 1
        FROM migration_000058_id_rewrite rewrite
        WHERE data_object_type(rewrite.new_id) IS DISTINCT FROM rewrite.object_type
           OR substring(rewrite.old_id FROM 4) <> substring(rewrite.new_id FROM 4)
    ) THEN
        RAISE EXCEPTION 'independent object identity rewrite did not preserve owner type and UUID suffix';
    END IF;

    IF EXISTS (
        SELECT 1 FROM industry WHERE id !~ '^IND[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
        UNION ALL SELECT 1 FROM concept WHERE id !~ '^CON[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
        UNION ALL SELECT 1 FROM chain_node WHERE id !~ '^CND[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
        UNION ALL SELECT 1 FROM industry_chain WHERE id !~ '^ICH[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    ) THEN
        RAISE EXCEPTION 'independent object owner table contains an invalid target identity';
    END IF;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE
    trigger_value RECORD;
BEGIN
    FOR trigger_value IN SELECT * FROM migration_000058_trigger_state LOOP
        CASE trigger_value.enabled
            WHEN 'O' THEN
				EXECUTE format('ALTER TABLE %I.%I ENABLE TRIGGER %I', current_schema(), trigger_value.table_name, trigger_value.trigger_name);
            WHEN 'R' THEN
				EXECUTE format('ALTER TABLE %I.%I ENABLE REPLICA TRIGGER %I', current_schema(), trigger_value.table_name, trigger_value.trigger_name);
            WHEN 'A' THEN
				EXECUTE format('ALTER TABLE %I.%I ENABLE ALWAYS TRIGGER %I', current_schema(), trigger_value.table_name, trigger_value.trigger_name);
            ELSE
				EXECUTE format('ALTER TABLE %I.%I DISABLE TRIGGER %I', current_schema(), trigger_value.table_name, trigger_value.trigger_name);
        END CASE;
    END LOOP;
END;
$$;
-- +goose StatementEnd

DROP FUNCTION migration_000058_rewrite_jsonb(JSONB);
DROP FUNCTION migration_000058_jsonb_key_collision(JSONB);

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000058 is forward-only; restore the reviewed pre-migration snapshot and previous application';
END;
$$;
-- +goose StatementEnd
