INSERT INTO agent_versions (version, agent_key)
VALUES ('event-semantic-enricher.v3', 'event-semantic-enricher')
ON CONFLICT (version) DO NOTHING;

CREATE TABLE IF NOT EXISTS event_semantic_stage_audits (
    execution_id uuid PRIMARY KEY REFERENCES agent_executions(execution_id) ON DELETE RESTRICT,
    event_id uuid NOT NULL,
    contract_version text NOT NULL,
    summary jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT event_semantic_stage_audits_contract_check
        CHECK (contract_version = 'event-semantic-stage-audit.v1'),
    CONSTRAINT event_semantic_stage_audits_summary_check
        CHECK (jsonb_typeof(summary) = 'object')
);

CREATE INDEX IF NOT EXISTS event_semantic_stage_audits_event_idx
    ON event_semantic_stage_audits (event_id, updated_at DESC);
