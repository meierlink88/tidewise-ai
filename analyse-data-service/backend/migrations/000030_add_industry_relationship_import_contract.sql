-- +goose Up
ALTER TABLE industry_chain_definitions
    ADD COLUMN technology_route_qualifier TEXT,
    ADD COLUMN observable_variables TEXT[];

ALTER TABLE industry_chain_node_memberships
    ADD COLUMN inclusion_reason TEXT,
    ADD COLUMN evidence_ids TEXT[],
    ADD COLUMN source_name TEXT,
    ADD COLUMN source_url TEXT,
    ADD COLUMN verified_at TIMESTAMPTZ;

ALTER TABLE industry_chain_graph_edges
    ADD COLUMN evidence_ids TEXT[],
    ADD COLUMN source_name TEXT,
    ADD COLUMN source_url TEXT,
    ADD COLUMN verified_at TIMESTAMPTZ;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM industry_chain_definitions) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'industry chain definitions exist without V1 route/observable variables; audit them before migration 000030';
    END IF;
    IF EXISTS (SELECT 1 FROM industry_chain_node_memberships) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'industry chain memberships exist without V1 inclusion/provenance; audit them before migration 000030';
    END IF;
    IF EXISTS (SELECT 1 FROM industry_chain_graph_edges) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'industry chain graph edges exist without V1 provenance; audit them before migration 000030';
    END IF;
END;
$$;
-- +goose StatementEnd

ALTER TABLE industry_chain_definitions
    ALTER COLUMN observable_variables SET NOT NULL,
    ADD CONSTRAINT chk_industry_chain_definition_route_nonblank
        CHECK (
            technology_route_qualifier IS NULL
            OR btrim(technology_route_qualifier) <> ''
        ),
    ADD CONSTRAINT chk_industry_chain_definition_observable_variables
        CHECK (
            cardinality(observable_variables) > 0
            AND array_position(observable_variables, NULL) IS NULL
        );

ALTER TABLE industry_chain_node_memberships
    ALTER COLUMN inclusion_reason SET NOT NULL,
    ALTER COLUMN evidence_ids SET NOT NULL,
    ALTER COLUMN source_name SET NOT NULL,
    ALTER COLUMN source_url SET NOT NULL,
    ALTER COLUMN verified_at SET NOT NULL,
    ADD CONSTRAINT chk_industry_chain_membership_inclusion_nonblank
        CHECK (btrim(inclusion_reason) <> ''),
    ADD CONSTRAINT chk_industry_chain_membership_evidence_ids
        CHECK (
            cardinality(evidence_ids) > 0
            AND array_position(evidence_ids, NULL) IS NULL
        ),
    ADD CONSTRAINT chk_industry_chain_membership_source_name_nonblank
        CHECK (btrim(source_name) <> ''),
    ADD CONSTRAINT chk_industry_chain_membership_source_locator
        CHECK (source_url ~ '^(https?://|artifact://)');

ALTER TABLE industry_chain_graph_edges
    ALTER COLUMN evidence_ids SET NOT NULL,
    ALTER COLUMN source_name SET NOT NULL,
    ALTER COLUMN source_url SET NOT NULL,
    ALTER COLUMN verified_at SET NOT NULL,
    ADD CONSTRAINT chk_industry_chain_graph_evidence_ids
        CHECK (
            cardinality(evidence_ids) > 0
            AND array_position(evidence_ids, NULL) IS NULL
        ),
    ADD CONSTRAINT chk_industry_chain_graph_source_name_nonblank
        CHECK (btrim(source_name) <> ''),
    ADD CONSTRAINT chk_industry_chain_graph_source_locator
        CHECK (source_url ~ '^(https?://|artifact://)');

CREATE TABLE industry_relationship_import_receipts (
    id UUID PRIMARY KEY,
    package_sha256 TEXT NOT NULL UNIQUE,
    manifest_sha256 TEXT NOT NULL,
    relation_spec_sha256 TEXT NOT NULL,
    approval_basis TEXT NOT NULL,
    package_counts JSONB NOT NULL,
    caller_subject TEXT NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_industry_relationship_receipt_package_sha
        CHECK (package_sha256 ~ '^[0-9a-f]{64}$'),
    CONSTRAINT chk_industry_relationship_receipt_manifest_sha
        CHECK (manifest_sha256 ~ '^[0-9a-f]{64}$'),
    CONSTRAINT chk_industry_relationship_receipt_spec_sha
        CHECK (relation_spec_sha256 ~ '^[0-9a-f]{64}$'),
    CONSTRAINT chk_industry_relationship_receipt_approval
        CHECK (approval_basis = 'user_explicit_delegated_review'),
    CONSTRAINT chk_industry_relationship_receipt_counts
        CHECK (jsonb_typeof(package_counts) = 'object'),
    CONSTRAINT chk_industry_relationship_receipt_caller
        CHECK (
            char_length(caller_subject) BETWEEN 1 AND 200
            AND btrim(caller_subject) <> ''
        )
);

-- +goose StatementBegin
CREATE FUNCTION prevent_industry_relationship_import_receipt_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'industry relationship import receipts are immutable';
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER trg_industry_relationship_import_receipts_immutable
BEFORE UPDATE OR DELETE OR TRUNCATE ON industry_relationship_import_receipts
FOR EACH STATEMENT
EXECUTE FUNCTION prevent_industry_relationship_import_receipt_mutation();

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000030 is forward-only; use a reviewed forward migration or restore the pre-migration backup';
END;
$$;
-- +goose StatementEnd
