ALTER TABLE agent_executions
    DROP CONSTRAINT agent_executions_status_check;

ALTER TABLE agent_executions
    ADD CONSTRAINT agent_executions_status_check
    CHECK (status IN (
        'queued', 'planning', 'collecting', 'materializing',
        'succeeded', 'succeeded_no_change', 'partially_succeeded', 'failed', 'skipped'
    )),
    ADD COLUMN stop_reason text,
    ADD COLUMN blocked_by_execution_id uuid REFERENCES agent_executions(execution_id);

UPDATE agent_executions
SET stop_reason = CASE status
    WHEN 'succeeded' THEN 'connectors_completed'
    WHEN 'succeeded_no_change' THEN 'connectors_completed'
    WHEN 'partially_succeeded' THEN 'completed_with_connector_failures'
    WHEN 'failed' THEN CASE
        WHEN error_code = 'all_connectors_failed' THEN 'completed_with_connector_failures'
        ELSE 'agent_or_tool_limit'
    END
END
WHERE stop_reason IS NULL
  AND status IN ('succeeded', 'succeeded_no_change', 'partially_succeeded', 'failed');

ALTER TABLE connector_invocations
    DROP CONSTRAINT connector_invocations_status_check;

ALTER TABLE connector_invocations
    ADD CONSTRAINT connector_invocations_status_check
    CHECK (status IN ('pending', 'running', 'completed', 'failed', 'not_invoked'));

DROP INDEX agent_executions_one_active;

CREATE UNIQUE INDEX agent_executions_one_active
ON agent_executions ((true))
WHERE status IN ('queued', 'planning', 'collecting', 'materializing');

CREATE TABLE collector_artifact_publications (
    execution_id uuid PRIMARY KEY REFERENCES agent_executions(execution_id) ON DELETE CASCADE,
    plan_path text NOT NULL,
    plan_sha256 char(64) NOT NULL,
    prepared_at timestamptz NOT NULL
);
