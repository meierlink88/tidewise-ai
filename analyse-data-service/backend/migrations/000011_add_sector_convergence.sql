-- +goose Up
CREATE TABLE entity_convergence_manifests (
    manifest_version BIGINT PRIMARY KEY,
    previous_manifest_version BIGINT REFERENCES entity_convergence_manifests(manifest_version),
    manifest_checksum TEXT NOT NULL UNIQUE,
    review_source_url TEXT NOT NULL,
    reviewed_at TIMESTAMPTZ NOT NULL,
    applied_mode TEXT NOT NULL CHECK (applied_mode IN ('initial', 'correction')),
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE entity_convergences (
    id UUID PRIMARY KEY,
    legacy_entity_id UUID NOT NULL REFERENCES entity_nodes(id),
    target_entity_id UUID REFERENCES entity_nodes(id),
    target_entity_type TEXT,
    manifest_version BIGINT NOT NULL REFERENCES entity_convergence_manifests(manifest_version),
    action TEXT NOT NULL CHECK (action IN ('replace', 'merge', 'retire_without_canonical', 'replace_with_existing_index', 'retire_without_target')),
    legacy_taxonomy TEXT NOT NULL,
    reason TEXT NOT NULL CHECK (btrim(reason) <> ''),
    mutation_provenance JSONB NOT NULL DEFAULT '{}'::jsonb,
    converged_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (legacy_entity_id, manifest_version),
    CONSTRAINT chk_entity_convergences_target_snapshot CHECK (
        (action IN ('replace', 'merge') AND target_entity_id IS NOT NULL AND target_entity_type = 'sector')
        OR (action = 'replace_with_existing_index' AND target_entity_id IS NOT NULL AND target_entity_type = 'index')
        OR (action IN ('retire_without_canonical', 'retire_without_target') AND target_entity_id IS NULL AND target_entity_type IS NULL)
    )
);

CREATE TABLE entity_convergence_reference_moves (
    id UUID PRIMARY KEY,
    convergence_id UUID NOT NULL REFERENCES entity_convergences(id),
    reference_table TEXT NOT NULL,
    reference_column TEXT NOT NULL,
    reference_row_id UUID NOT NULL,
    from_entity_id UUID NOT NULL REFERENCES entity_nodes(id),
    to_entity_id UUID REFERENCES entity_nodes(id),
    mutation_provenance JSONB NOT NULL,
    moved_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (convergence_id, reference_table, reference_column, reference_row_id)
);

CREATE TABLE entity_convergence_alias_moves (
    id UUID PRIMARY KEY,
    convergence_id UUID NOT NULL REFERENCES entity_convergences(id),
    alias TEXT NOT NULL,
    from_entity_id UUID NOT NULL REFERENCES entity_nodes(id),
    to_entity_id UUID NOT NULL REFERENCES entity_nodes(id),
    mutation_provenance JSONB NOT NULL,
    moved_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (convergence_id, alias)
);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION prevent_entity_convergence_audit_mutation()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'convergence audit is append-only';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER entity_convergence_manifests_append_only
BEFORE UPDATE OR DELETE ON entity_convergence_manifests
FOR EACH ROW EXECUTE FUNCTION prevent_entity_convergence_audit_mutation();

CREATE TRIGGER entity_convergences_append_only
BEFORE UPDATE OR DELETE ON entity_convergences
FOR EACH ROW EXECUTE FUNCTION prevent_entity_convergence_audit_mutation();

CREATE TRIGGER entity_convergence_reference_moves_append_only
BEFORE UPDATE OR DELETE ON entity_convergence_reference_moves
FOR EACH ROW EXECUTE FUNCTION prevent_entity_convergence_audit_mutation();

CREATE TRIGGER entity_convergence_alias_moves_append_only
BEFORE UPDATE OR DELETE ON entity_convergence_alias_moves
FOR EACH ROW EXECUTE FUNCTION prevent_entity_convergence_audit_mutation();

-- +goose Down
SELECT 'sector convergence rollback requires a reviewed forward migration or restored backup';
