-- +goose Up
ALTER TABLE variable_signal_measurements
    DROP CONSTRAINT chk_variable_signal_measurement_role,
    DROP CONSTRAINT chk_variable_signal_measurement_shape,
    DROP CONSTRAINT chk_variable_signal_measurement_values,
    DROP CONSTRAINT chk_variable_signal_measurement_change_unit,
    DROP CONSTRAINT chk_variable_signal_measurement_units,
    DROP CONSTRAINT chk_variable_signal_measurement_conversion,
    DROP CONSTRAINT chk_variable_signal_measurement_raw_range,
    DROP CONSTRAINT chk_variable_signal_measurement_canonical_range,
    ALTER COLUMN measurement_role DROP NOT NULL,
    ALTER COLUMN value_shape DROP NOT NULL,
    ALTER COLUMN raw_unit DROP NOT NULL,
    ALTER COLUMN canonical_unit DROP NOT NULL,
    ADD COLUMN evidence_ids UUID[];

UPDATE variable_signal_measurements
SET evidence_ids = ARRAY[evidence_id];

-- +goose StatementBegin
CREATE FUNCTION event_semantic_measurement_evidence_ids_compat()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.evidence_ids IS NULL OR cardinality(NEW.evidence_ids) = 0 THEN
        NEW.evidence_ids := ARRAY[NEW.evidence_id];
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER trg_event_semantic_measurement_evidence_ids_compat
BEFORE INSERT OR UPDATE OF evidence_id, evidence_ids
ON variable_signal_measurements
FOR EACH ROW
EXECUTE FUNCTION event_semantic_measurement_evidence_ids_compat();

ALTER TABLE variable_signal_measurements
    ALTER COLUMN evidence_ids SET NOT NULL,
    ADD CONSTRAINT chk_variable_signal_measurement_evidence_ids CHECK (
        cardinality(evidence_ids) > 0
        AND array_position(evidence_ids, NULL::uuid) IS NULL
    );

ALTER TABLE entity_type_definitions
    ADD COLUMN allowed_event_roles TEXT[] NOT NULL DEFAULT '{}';

UPDATE entity_type_definitions
SET allowed_event_roles = CASE
    WHEN signal_subject_allowed THEN ARRAY[
        'event_subject', 'actor', 'affected_entity', 'statement_source', 'event_object', 'context'
    ]::text[]
    ELSE ARRAY['actor', 'statement_source', 'context']::text[]
END;

ALTER TABLE entity_type_definitions
    ADD CONSTRAINT chk_entity_type_definition_event_roles CHECK (
        cardinality(allowed_event_roles) > 0
        AND allowed_event_roles <@ ARRAY[
            'event_subject', 'actor', 'affected_entity', 'statement_source', 'event_object', 'context'
        ]::text[]
        AND array_position(allowed_event_roles, NULL::text) IS NULL
    );

ALTER TABLE variable_definitions
    ADD COLUMN allowed_units TEXT[] NOT NULL DEFAULT '{}';

UPDATE variable_definitions
SET allowed_units = ARRAY[canonical_unit]
WHERE canonical_unit IS NOT NULL AND btrim(canonical_unit) <> '';

ALTER TABLE variable_definitions
    ADD CONSTRAINT chk_variable_definition_allowed_units CHECK (
        array_position(allowed_units, NULL::text) IS NULL
    );

INSERT INTO event_semantic_acceptance_policies(
    policy_key, version, retry_budget, status, policy
) VALUES (
    'event-semantics.objective-v2', 1, 1, 'active',
    '{
      "requires_independent_reviewer": true,
      "assertion_modalities": ["actual", "stated_intent", "source_forecast"],
      "measurement_contract": {
        "representation": "evidence_grounded_narrative",
        "max_items_per_signal": 8,
        "max_text_characters": 2000,
        "requires_evidence_ids": true,
        "numeric_validation": false
      }
    }'::jsonb
);

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000036 is forward-only; roll back application binaries while retaining the relaxed schema';
END;
$$;
-- +goose StatementEnd
