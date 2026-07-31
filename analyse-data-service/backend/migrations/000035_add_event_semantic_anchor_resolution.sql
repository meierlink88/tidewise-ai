-- +goose Up
ALTER TABLE event_semantic_context_leases
    ADD COLUMN context_manifest JSONB,
    ALTER COLUMN context_snapshot DROP NOT NULL,
    ADD CONSTRAINT chk_event_semantic_context_lease_manifest
        CHECK (context_manifest IS NULL OR jsonb_typeof(context_manifest) = 'object'),
    ADD CONSTRAINT chk_event_semantic_context_lease_payload
        CHECK (context_manifest IS NOT NULL OR context_snapshot IS NOT NULL);

CREATE TABLE event_semantic_resolution_bindings (
    id UUID PRIMARY KEY,
    semantic_submission_id UUID NOT NULL
        REFERENCES event_semantic_submissions(id) ON DELETE RESTRICT,
    context_lease_id UUID NOT NULL
        REFERENCES event_semantic_context_leases(id) ON DELETE RESTRICT,
    candidate_key TEXT NOT NULL,
    mention TEXT NOT NULL,
    anchor_entity_id UUID NOT NULL REFERENCES entity_nodes(id) ON DELETE RESTRICT,
    target_entity_id UUID NOT NULL REFERENCES entity_nodes(id) ON DELETE RESTRICT,
    route_id VARCHAR(128) NOT NULL,
    route_contract_version VARCHAR(64) NOT NULL,
    path_fingerprint CHAR(64) NOT NULL,
    resolution_receipt JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_event_semantic_resolution_binding_candidate
        UNIQUE (semantic_submission_id, candidate_key),
    CONSTRAINT chk_event_semantic_resolution_binding_text
        CHECK (
            btrim(candidate_key) <> '' AND btrim(mention) <> ''
            AND btrim(route_id) <> '' AND btrim(route_contract_version) <> ''
        ),
    CONSTRAINT chk_event_semantic_resolution_binding_hash
        CHECK (path_fingerprint ~ '^[0-9a-f]{64}$'),
    CONSTRAINT chk_event_semantic_resolution_binding_receipt
        CHECK (jsonb_typeof(resolution_receipt) = 'object')
);

CREATE INDEX idx_event_semantic_resolution_bindings_target
    ON event_semantic_resolution_bindings(target_entity_id, semantic_submission_id);

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000035 is forward-only; use a reviewed forward migration or restore the pre-migration backup';
END;
$$;
-- +goose StatementEnd
