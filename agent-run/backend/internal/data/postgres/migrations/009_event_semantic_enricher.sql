INSERT INTO agent_definitions (agent_key, display_name)
VALUES ('event-semantic-enricher', 'Event Semantic Enricher Agent')
ON CONFLICT (agent_key) DO UPDATE
SET display_name = EXCLUDED.display_name;

INSERT INTO agent_versions (version, agent_key)
VALUES ('event-semantic-enricher.v1', 'event-semantic-enricher')
ON CONFLICT (version) DO NOTHING;

CREATE TABLE event_semantic_work_items (
    work_item_id uuid PRIMARY KEY,
    event_id uuid NOT NULL,
    supersedes_submission_id uuid,
    trigger_source text NOT NULL,
    reason text NOT NULL DEFAULT '',
    idempotency_key text NOT NULL UNIQUE,
    status text NOT NULL DEFAULT 'pending',
    attempt_count integer NOT NULL DEFAULT 0,
    max_attempts integer NOT NULL DEFAULT 2,
    lease_expires_at timestamptz,
    current_execution_id uuid REFERENCES agent_executions(execution_id),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT event_semantic_work_items_trigger_check
        CHECK (trigger_source IN ('eligible_event', 'explicit_reanalysis')),
    CONSTRAINT event_semantic_work_items_status_check
        CHECK (status IN ('pending', 'running', 'succeeded', 'failed')),
    CONSTRAINT event_semantic_work_items_attempt_check
        CHECK (attempt_count >= 0 AND max_attempts >= 1 AND attempt_count <= max_attempts),
    CONSTRAINT event_semantic_work_items_reanalysis_check
        CHECK (
            (trigger_source = 'eligible_event' AND supersedes_submission_id IS NULL)
            OR
            (trigger_source = 'explicit_reanalysis' AND supersedes_submission_id IS NOT NULL)
        ),
    CONSTRAINT event_semantic_work_items_lease_check
        CHECK (
            (status = 'running' AND lease_expires_at IS NOT NULL AND current_execution_id IS NOT NULL)
            OR
            (status <> 'running' AND lease_expires_at IS NULL)
        )
);

CREATE INDEX event_semantic_work_items_pending_idx
    ON event_semantic_work_items (created_at, work_item_id)
    WHERE status = 'pending';

CREATE INDEX event_semantic_work_items_event_idx
    ON event_semantic_work_items (event_id, created_at DESC);
