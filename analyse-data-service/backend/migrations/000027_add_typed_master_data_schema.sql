-- +goose Up
CREATE UNIQUE INDEX ux_entity_nodes_nonblank_key
    ON entity_nodes (entity_key)
    WHERE btrim(entity_key) <> '';

CREATE INDEX idx_entity_nodes_aliases_gin
    ON entity_nodes USING GIN (aliases);

ALTER TABLE chain_node_profiles
    ADD COLUMN review_status VARCHAR(32),
    ADD CONSTRAINT chk_chain_node_profile_review_status
        CHECK (review_status IS NULL OR review_status IN ('candidate', 'approved'));

CREATE TABLE industry_profiles (
    entity_id UUID PRIMARY KEY REFERENCES entity_nodes(id) ON DELETE RESTRICT,
    classification_system TEXT NOT NULL,
    classification_version TEXT NOT NULL,
    industry_code TEXT NOT NULL,
    classification_level SMALLINT NOT NULL,
    parent_industry_entity_id UUID REFERENCES industry_profiles(entity_id) ON DELETE RESTRICT,
    hierarchy_path_codes TEXT[] NOT NULL,
    definition TEXT NOT NULL,
    boundary_note TEXT NOT NULL,
    review_status VARCHAR(32) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_industry_profile_classification_identity
        UNIQUE (classification_system, classification_version, industry_code),
    CONSTRAINT chk_industry_profile_system_nonblank CHECK (btrim(classification_system) <> ''),
    CONSTRAINT chk_industry_profile_version_nonblank CHECK (btrim(classification_version) <> ''),
    CONSTRAINT chk_industry_profile_code_nonblank CHECK (btrim(industry_code) <> ''),
    CONSTRAINT chk_industry_profile_level CHECK (classification_level IN (1, 2, 3)),
    CONSTRAINT chk_industry_profile_parent_presence CHECK (
        (classification_level = 1 AND parent_industry_entity_id IS NULL)
        OR (classification_level IN (2, 3) AND parent_industry_entity_id IS NOT NULL)
    ),
    CONSTRAINT chk_industry_profile_path_length CHECK (
        cardinality(hierarchy_path_codes) = classification_level
    ),
    CONSTRAINT chk_industry_profile_path_leaf CHECK (
        hierarchy_path_codes[cardinality(hierarchy_path_codes)] = industry_code
    ),
    CONSTRAINT chk_industry_profile_definition_nonblank CHECK (btrim(definition) <> ''),
    CONSTRAINT chk_industry_profile_boundary_nonblank CHECK (btrim(boundary_note) <> ''),
    CONSTRAINT chk_industry_profile_review_status CHECK (review_status IN ('candidate', 'approved'))
);

CREATE INDEX idx_industry_profiles_parent
    ON industry_profiles (parent_industry_entity_id);
CREATE INDEX idx_industry_profiles_classification_level
    ON industry_profiles (classification_system, classification_version, classification_level, industry_code);
CREATE INDEX idx_industry_profiles_review_status
    ON industry_profiles (review_status, classification_system, classification_version);

CREATE TABLE concept_profiles (
    entity_id UUID PRIMARY KEY REFERENCES entity_nodes(id) ON DELETE RESTRICT,
    concept_type VARCHAR(32) NOT NULL,
    definition TEXT NOT NULL,
    boundary_note TEXT NOT NULL,
    review_status VARCHAR(32) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_concept_profile_type CHECK (concept_type IN (
        'technology',
        'policy',
        'application',
        'demand',
        'business_model',
        'company_ecosystem',
        'product_ecosystem',
        'event_narrative',
        'market_theme'
    )),
    CONSTRAINT chk_concept_profile_definition_nonblank CHECK (btrim(definition) <> ''),
    CONSTRAINT chk_concept_profile_boundary_nonblank CHECK (btrim(boundary_note) <> ''),
    CONSTRAINT chk_concept_profile_review_status CHECK (review_status IN ('candidate', 'approved'))
);

CREATE INDEX idx_concept_profiles_type_review
    ON concept_profiles (concept_type, review_status);

CREATE TABLE industry_chain_definitions (
    entity_id UUID PRIMARY KEY REFERENCES entity_nodes(id) ON DELETE RESTRICT,
    scope TEXT NOT NULL,
    target_output TEXT NOT NULL,
    end_use TEXT NOT NULL,
    geography TEXT NOT NULL,
    as_of_date DATE NOT NULL,
    review_status VARCHAR(32) NOT NULL,
    review_note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_industry_chain_definition_scope_nonblank CHECK (btrim(scope) <> ''),
    CONSTRAINT chk_industry_chain_definition_target_output_nonblank CHECK (btrim(target_output) <> ''),
    CONSTRAINT chk_industry_chain_definition_end_use_nonblank CHECK (btrim(end_use) <> ''),
    CONSTRAINT chk_industry_chain_definition_geography_nonblank CHECK (btrim(geography) <> ''),
    CONSTRAINT chk_industry_chain_definition_review_status CHECK (review_status IN ('candidate', 'approved')),
    CONSTRAINT chk_industry_chain_definition_review_note_nonblank CHECK (
        review_note IS NULL OR btrim(review_note) <> ''
    )
);

CREATE INDEX idx_industry_chain_definitions_review_date
    ON industry_chain_definitions (review_status, as_of_date DESC);

CREATE TABLE industry_chain_node_memberships (
    industry_chain_entity_id UUID NOT NULL REFERENCES industry_chain_definitions(entity_id) ON DELETE RESTRICT,
    chain_node_entity_id UUID NOT NULL REFERENCES chain_node_profiles(entity_id) ON DELETE RESTRICT,
    position INTEGER NOT NULL,
    contextual_stage VARCHAR(32) NOT NULL,
    review_status VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (industry_chain_entity_id, chain_node_entity_id),
    CONSTRAINT chk_industry_chain_node_membership_position CHECK (position > 0),
    CONSTRAINT chk_industry_chain_node_membership_stage CHECK (
        contextual_stage IN ('upstream', 'midstream', 'downstream')
    ),
    CONSTRAINT chk_industry_chain_node_membership_review_status CHECK (
        review_status IN ('candidate', 'approved')
    ),
    CONSTRAINT chk_industry_chain_node_membership_status CHECK (status IN ('active', 'inactive'))
);

CREATE INDEX idx_industry_chain_node_memberships_chain_status_position
    ON industry_chain_node_memberships (industry_chain_entity_id, status, position, chain_node_entity_id);
CREATE INDEX idx_industry_chain_node_memberships_node_status
    ON industry_chain_node_memberships (chain_node_entity_id, status);

CREATE TABLE industry_chain_graph_edges (
    id UUID PRIMARY KEY,
    industry_chain_entity_id UUID NOT NULL,
    from_chain_node_entity_id UUID NOT NULL,
    to_chain_node_entity_id UUID NOT NULL,
    relation_type VARCHAR(32) NOT NULL,
    mechanism TEXT NOT NULL,
    condition_note TEXT,
    segment_kind VARCHAR(32) NOT NULL,
    omitted_step_note TEXT,
    review_status VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_industry_chain_graph_from_membership
        FOREIGN KEY (industry_chain_entity_id, from_chain_node_entity_id)
        REFERENCES industry_chain_node_memberships (industry_chain_entity_id, chain_node_entity_id)
        ON DELETE RESTRICT,
    CONSTRAINT fk_industry_chain_graph_to_membership
        FOREIGN KEY (industry_chain_entity_id, to_chain_node_entity_id)
        REFERENCES industry_chain_node_memberships (industry_chain_entity_id, chain_node_entity_id)
        ON DELETE RESTRICT,
    CONSTRAINT uq_industry_chain_graph_semantic_edge
        UNIQUE (industry_chain_entity_id, from_chain_node_entity_id, to_chain_node_entity_id, relation_type),
    CONSTRAINT chk_industry_chain_graph_self CHECK (
        from_chain_node_entity_id <> to_chain_node_entity_id
    ),
    CONSTRAINT chk_industry_chain_graph_relation_type CHECK (
        relation_type IN ('input_to', 'is_component_of', 'depends_on')
    ),
    CONSTRAINT chk_industry_chain_graph_mechanism_nonblank CHECK (btrim(mechanism) <> ''),
    CONSTRAINT chk_industry_chain_graph_condition_nonblank CHECK (
        condition_note IS NULL OR btrim(condition_note) <> ''
    ),
    CONSTRAINT chk_industry_chain_graph_segment_kind CHECK (
        segment_kind IN ('direct_candidate', 'compressed_candidate')
    ),
    CONSTRAINT chk_industry_chain_graph_omitted_step CHECK (
        (segment_kind = 'direct_candidate' AND omitted_step_note IS NULL)
        OR (
            segment_kind = 'compressed_candidate'
            AND omitted_step_note IS NOT NULL
            AND btrim(omitted_step_note) <> ''
        )
    ),
    CONSTRAINT chk_industry_chain_graph_review_status CHECK (
        review_status IN ('candidate', 'approved')
    ),
    CONSTRAINT chk_industry_chain_graph_status CHECK (status IN ('active', 'inactive'))
);

CREATE INDEX idx_industry_chain_graph_chain_status
    ON industry_chain_graph_edges (industry_chain_entity_id, status, from_chain_node_entity_id);
CREATE INDEX idx_industry_chain_graph_to_node_status
    ON industry_chain_graph_edges (to_chain_node_entity_id, status);

CREATE TABLE entity_redirects (
    source_entity_id UUID PRIMARY KEY REFERENCES entity_nodes(id) ON DELETE RESTRICT,
    target_entity_id UUID NOT NULL REFERENCES entity_nodes(id) ON DELETE RESTRICT,
    redirect_kind VARCHAR(32) NOT NULL,
    reason TEXT NOT NULL,
    review_status VARCHAR(32) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_entity_redirect_self CHECK (source_entity_id <> target_entity_id),
    CONSTRAINT chk_entity_redirect_kind CHECK (redirect_kind IN ('merge', 'reclassification')),
    CONSTRAINT chk_entity_redirect_reason_nonblank CHECK (btrim(reason) <> ''),
    CONSTRAINT chk_entity_redirect_review_status CHECK (review_status IN ('candidate', 'approved'))
);

CREATE INDEX idx_entity_redirects_target
    ON entity_redirects (target_entity_id);
CREATE INDEX idx_entity_redirects_review_status
    ON entity_redirects (review_status, redirect_kind);

-- +goose StatementBegin
CREATE FUNCTION assert_entity_profile_type()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    actual_type TEXT;
    actual_key TEXT;
BEGIN
    SELECT entity_type, entity_key
    INTO actual_type, actual_key
    FROM entity_nodes
    WHERE id = NEW.entity_id
    FOR SHARE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'profile entity % does not exist', NEW.entity_id;
    END IF;
    IF actual_type IS DISTINCT FROM TG_ARGV[0] THEN
        RAISE EXCEPTION 'profile entity % has type %, expected %', NEW.entity_id, actual_type, TG_ARGV[0];
    END IF;
    IF btrim(actual_key) = '' THEN
        RAISE EXCEPTION 'typed profile entity % requires a nonblank entity_key', NEW.entity_id;
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER trg_industry_profile_entity_type
BEFORE INSERT OR UPDATE OF entity_id ON industry_profiles
FOR EACH ROW EXECUTE FUNCTION assert_entity_profile_type('industry');

CREATE TRIGGER trg_concept_profile_entity_type
BEFORE INSERT OR UPDATE OF entity_id ON concept_profiles
FOR EACH ROW EXECUTE FUNCTION assert_entity_profile_type('concept');

CREATE TRIGGER trg_chain_node_profile_entity_type
BEFORE INSERT OR UPDATE OF entity_id ON chain_node_profiles
FOR EACH ROW EXECUTE FUNCTION assert_entity_profile_type('chain_node');

CREATE TRIGGER trg_industry_chain_definition_entity_type
BEFORE INSERT OR UPDATE OF entity_id ON industry_chain_definitions
FOR EACH ROW EXECUTE FUNCTION assert_entity_profile_type('industry_chain');

-- +goose StatementBegin
CREATE FUNCTION protect_profiled_entity_identity()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM industry_profiles WHERE entity_id = OLD.id)
       AND (NEW.entity_type <> 'industry' OR btrim(NEW.entity_key) = '') THEN
        RAISE EXCEPTION 'industry profile identity cannot change type or use a blank key';
    END IF;
    IF EXISTS (SELECT 1 FROM concept_profiles WHERE entity_id = OLD.id)
       AND (NEW.entity_type <> 'concept' OR btrim(NEW.entity_key) = '') THEN
        RAISE EXCEPTION 'concept profile identity cannot change type or use a blank key';
    END IF;
    IF EXISTS (SELECT 1 FROM chain_node_profiles WHERE entity_id = OLD.id)
       AND (NEW.entity_type <> 'chain_node' OR btrim(NEW.entity_key) = '') THEN
        RAISE EXCEPTION 'chain node profile identity cannot change type or use a blank key';
    END IF;
    IF EXISTS (SELECT 1 FROM industry_chain_definitions WHERE entity_id = OLD.id)
       AND (NEW.entity_type <> 'industry_chain' OR btrim(NEW.entity_key) = '') THEN
        RAISE EXCEPTION 'industry chain profile identity cannot change type or use a blank key';
    END IF;

    IF NEW.entity_type IS DISTINCT FROM OLD.entity_type THEN
        PERFORM pg_advisory_xact_lock(hashtext('entity_redirects'));

        IF EXISTS (
            SELECT 1
            FROM entity_redirects redirect
            JOIN entity_nodes target ON target.id = redirect.target_entity_id
            WHERE redirect.source_entity_id = OLD.id
              AND (
                  (redirect.redirect_kind = 'merge' AND NEW.entity_type <> target.entity_type)
                  OR (redirect.redirect_kind = 'reclassification' AND NEW.entity_type = target.entity_type)
              )
            UNION ALL
            SELECT 1
            FROM entity_redirects redirect
            JOIN entity_nodes source ON source.id = redirect.source_entity_id
            WHERE redirect.target_entity_id = OLD.id
              AND (
                  (redirect.redirect_kind = 'merge' AND source.entity_type <> NEW.entity_type)
                  OR (redirect.redirect_kind = 'reclassification' AND source.entity_type = NEW.entity_type)
              )
        ) THEN
            RAISE EXCEPTION 'entity type change would invalidate an existing redirect';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER trg_protect_profiled_entity_identity
BEFORE UPDATE OF entity_type, entity_key ON entity_nodes
FOR EACH ROW EXECUTE FUNCTION protect_profiled_entity_identity();

-- +goose StatementBegin
CREATE FUNCTION validate_industry_profile_hierarchy()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    parent_system TEXT;
    parent_version TEXT;
    parent_level SMALLINT;
    parent_path TEXT[];
BEGIN
	    IF NEW.classification_level = 1 THEN
	        IF NEW.hierarchy_path_codes <> ARRAY[NEW.industry_code]::TEXT[] THEN
	            RAISE EXCEPTION 'level 1 industry path must contain only its own code';
	        END IF;
	    ELSE
	        IF NEW.parent_industry_entity_id = NEW.entity_id THEN
	            RAISE EXCEPTION 'industry cannot be its own parent';
	        END IF;

	        SELECT classification_system, classification_version, classification_level, hierarchy_path_codes
        INTO parent_system, parent_version, parent_level, parent_path
        FROM industry_profiles
        WHERE entity_id = NEW.parent_industry_entity_id
        FOR SHARE;

        IF NOT FOUND THEN
            RAISE EXCEPTION 'industry parent % does not exist', NEW.parent_industry_entity_id;
        END IF;
        IF parent_system <> NEW.classification_system OR parent_version <> NEW.classification_version THEN
            RAISE EXCEPTION 'industry parent must use the same classification system and version';
        END IF;
        IF parent_level <> NEW.classification_level - 1 THEN
            RAISE EXCEPTION 'industry parent level must be exactly one level above child';
        END IF;
        IF NEW.hierarchy_path_codes <> (parent_path || NEW.industry_code) THEN
            RAISE EXCEPTION 'industry hierarchy path must extend the direct parent path';
        END IF;
    END IF;

    IF TG_OP = 'UPDATE' THEN
        PERFORM 1
        FROM industry_profiles child
        WHERE child.parent_industry_entity_id = OLD.entity_id
        FOR SHARE;

        IF EXISTS (
            SELECT 1
            FROM industry_profiles child
            WHERE child.parent_industry_entity_id = OLD.entity_id
              AND (
                  child.classification_system <> NEW.classification_system
                  OR child.classification_version <> NEW.classification_version
                  OR child.classification_level <> NEW.classification_level + 1
                  OR child.hierarchy_path_codes <> (NEW.hierarchy_path_codes || child.industry_code)
              )
        ) THEN
            RAISE EXCEPTION 'industry update would invalidate an existing child hierarchy';
        END IF;
    END IF;

    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER trg_validate_industry_profile_hierarchy
BEFORE INSERT OR UPDATE ON industry_profiles
FOR EACH ROW EXECUTE FUNCTION validate_industry_profile_hierarchy();

-- +goose StatementBegin
CREATE FUNCTION protect_active_industry_chain_membership()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF OLD.status = 'active' AND NEW.status = 'inactive' THEN
        PERFORM pg_advisory_xact_lock(hashtext('industry_chain_topology:' || NEW.industry_chain_entity_id::TEXT));

        IF EXISTS (
            SELECT 1
            FROM industry_chain_graph_edges edge
            WHERE edge.industry_chain_entity_id = NEW.industry_chain_entity_id
              AND edge.status = 'active'
              AND NEW.chain_node_entity_id IN (
                  edge.from_chain_node_entity_id,
                  edge.to_chain_node_entity_id
              )
        ) THEN
            RAISE EXCEPTION 'active industry chain graph edges require active memberships';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER trg_protect_active_industry_chain_membership
BEFORE UPDATE OF status ON industry_chain_node_memberships
FOR EACH ROW EXECUTE FUNCTION protect_active_industry_chain_membership();

-- +goose StatementBegin
CREATE FUNCTION reject_industry_chain_graph_cycle()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.status = 'active' THEN
        PERFORM 1
        FROM industry_chain_node_memberships membership
        WHERE membership.industry_chain_entity_id = NEW.industry_chain_entity_id
          AND membership.chain_node_entity_id IN (
              NEW.from_chain_node_entity_id,
              NEW.to_chain_node_entity_id
          )
        ORDER BY membership.chain_node_entity_id
        FOR SHARE;

        IF EXISTS (
            SELECT 1
            FROM industry_chain_node_memberships membership
            WHERE membership.industry_chain_entity_id = NEW.industry_chain_entity_id
              AND membership.chain_node_entity_id IN (
                  NEW.from_chain_node_entity_id,
                  NEW.to_chain_node_entity_id
              )
              AND membership.status <> 'active'
        ) THEN
            RAISE EXCEPTION 'active industry chain graph edges require active memberships';
        END IF;

        PERFORM pg_advisory_xact_lock(hashtext('industry_chain_topology:' || NEW.industry_chain_entity_id::TEXT));
    END IF;

    IF NEW.status = 'active' AND EXISTS (
        WITH RECURSIVE reachable(node_id) AS (
            SELECT NEW.to_chain_node_entity_id
            UNION
            SELECT edge.to_chain_node_entity_id
            FROM industry_chain_graph_edges edge
            JOIN reachable current_path ON edge.from_chain_node_entity_id = current_path.node_id
            WHERE edge.industry_chain_entity_id = NEW.industry_chain_entity_id
              AND edge.status = 'active'
              AND edge.id <> NEW.id
        )
        SELECT 1 FROM reachable WHERE node_id = NEW.from_chain_node_entity_id
    ) THEN
        RAISE EXCEPTION 'industry chain topology must remain acyclic';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER trg_reject_industry_chain_graph_cycle
BEFORE INSERT OR UPDATE ON industry_chain_graph_edges
FOR EACH ROW EXECUTE FUNCTION reject_industry_chain_graph_cycle();

-- +goose StatementBegin
CREATE FUNCTION validate_entity_redirect()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    source_type TEXT;
    target_type TEXT;
BEGIN
    SELECT entity_type INTO source_type
    FROM entity_nodes
    WHERE id = NEW.source_entity_id
    FOR SHARE;

    SELECT entity_type INTO target_type
    FROM entity_nodes
    WHERE id = NEW.target_entity_id
    FOR SHARE;

    PERFORM pg_advisory_xact_lock(hashtext('entity_redirects'));

    IF source_type IS NULL OR target_type IS NULL THEN
        RAISE EXCEPTION 'redirect source and target identities must exist';
    END IF;
    IF NEW.redirect_kind = 'merge' AND source_type <> target_type THEN
        RAISE EXCEPTION 'merge redirect requires source and target of the same type';
    END IF;
    IF NEW.redirect_kind = 'reclassification' AND source_type = target_type THEN
        RAISE EXCEPTION 'reclassification redirect requires source and target of different types';
    END IF;
    IF EXISTS (
        WITH RECURSIVE reachable(entity_id) AS (
            SELECT NEW.target_entity_id
            UNION
            SELECT redirect.target_entity_id
            FROM entity_redirects redirect
            JOIN reachable current_path ON redirect.source_entity_id = current_path.entity_id
            WHERE redirect.source_entity_id <> NEW.source_entity_id
        )
        SELECT 1 FROM reachable WHERE entity_id = NEW.source_entity_id
    ) THEN
        RAISE EXCEPTION 'entity redirect graph must remain acyclic';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER trg_validate_entity_redirect
BEFORE INSERT OR UPDATE ON entity_redirects
FOR EACH ROW EXECUTE FUNCTION validate_entity_redirect();

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION 'migration 000027 is forward-only; apply a reviewed forward-fix migration';
END;
$$;
-- +goose StatementEnd
