-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF current_setting('tidewise.external_identifier_schema_write_authorized', true) IS DISTINCT FROM 'reviewed_backup_verified' THEN
        RAISE EXCEPTION 'external identifier schema write is not authorized; complete schema Review and backup verification before applying migration 000016';
    END IF;
END;
$$;
-- +goose StatementEnd

CREATE TABLE entity_external_identifiers (
    id UUID PRIMARY KEY,
    entity_id UUID NOT NULL REFERENCES entity_nodes(id) ON DELETE CASCADE,
    source_system TEXT NOT NULL,
    source_taxonomy_type TEXT NOT NULL,
    external_code TEXT NOT NULL,
    external_name TEXT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_entity_external_identifier_identity UNIQUE (source_system, source_taxonomy_type, external_code),
    CONSTRAINT chk_entity_external_identifier_source_system CHECK (btrim(source_system) <> ''),
    CONSTRAINT chk_entity_external_identifier_taxonomy CHECK (btrim(source_taxonomy_type) <> ''),
    CONSTRAINT chk_entity_external_identifier_code CHECK (btrim(external_code) <> ''),
    CONSTRAINT chk_entity_external_identifier_name CHECK (btrim(external_name) <> ''),
    CONSTRAINT chk_entity_external_identifier_status CHECK (status IN ('active', 'inactive'))
);

CREATE INDEX idx_entity_external_identifiers_entity_source
    ON entity_external_identifiers (entity_id, source_system, source_taxonomy_type);

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION 'migration 000016 is irreversible after external identifiers exist; apply a reviewed forward-fix migration';
END;
$$;
-- +goose StatementEnd
