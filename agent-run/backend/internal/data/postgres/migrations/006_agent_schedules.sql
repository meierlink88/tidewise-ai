ALTER TABLE agent_versions
    ADD CONSTRAINT agent_versions_version_agent_key_key
    UNIQUE (version, agent_key);

CREATE TABLE agent_schedules (
    schedule_id uuid PRIMARY KEY,
    agent_key text NOT NULL UNIQUE,
    agent_version text NOT NULL,
    schedule_type text NOT NULL,
    cron_expression text,
    daily_times jsonb,
    input_payload jsonb NOT NULL,
    enabled boolean NOT NULL,
    last_triggered_at timestamptz,
    next_run_at timestamptz,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT agent_schedules_agent_version_agent_key_fkey
        FOREIGN KEY (agent_version, agent_key)
        REFERENCES agent_versions (version, agent_key),
    CONSTRAINT agent_schedules_type_check
        CHECK (schedule_type IN ('cron', 'daily')),
    CONSTRAINT agent_schedules_policy_check
        CHECK (
            (
                schedule_type = 'cron'
                AND cron_expression IS NOT NULL
                AND btrim(cron_expression) <> ''
                AND daily_times IS NULL
            )
            OR
            (
                schedule_type = 'daily'
                AND cron_expression IS NULL
                AND daily_times IS NOT NULL
                AND jsonb_typeof(daily_times) = 'array'
                AND jsonb_array_length(daily_times) > 0
            )
        ),
    CONSTRAINT agent_schedules_input_object_check
        CHECK (jsonb_typeof(input_payload) = 'object')
);

CREATE INDEX agent_schedules_enabled_idx
    ON agent_schedules (enabled, agent_key);

ALTER TABLE agent_executions
    ADD COLUMN agent_key text,
    ADD COLUMN input_payload jsonb,
    ADD COLUMN trigger_source text,
    ADD COLUMN schedule_id uuid REFERENCES agent_schedules(schedule_id),
    ADD COLUMN triggered_at timestamptz;

UPDATE agent_executions AS execution
SET agent_key = version.agent_key,
    input_payload = jsonb_build_object('prompt', execution.prompt),
    trigger_source = 'api',
    triggered_at = execution.created_at
FROM agent_versions AS version
WHERE version.version = execution.agent_version;

ALTER TABLE agent_executions
    ALTER COLUMN agent_key SET NOT NULL,
    ALTER COLUMN input_payload SET NOT NULL,
    ALTER COLUMN trigger_source SET NOT NULL,
    ALTER COLUMN triggered_at SET NOT NULL,
    ALTER COLUMN prompt DROP NOT NULL,
    ALTER COLUMN prompt_sha256 DROP NOT NULL,
    ALTER COLUMN prompt_bytes DROP NOT NULL,
    ADD CONSTRAINT agent_executions_agent_version_agent_key_fkey
        FOREIGN KEY (agent_version, agent_key)
        REFERENCES agent_versions (version, agent_key),
    ADD CONSTRAINT agent_executions_trigger_source_check
        CHECK (trigger_source IN ('api', 'schedule')),
    ADD CONSTRAINT agent_executions_trigger_schedule_check
        CHECK (
            (trigger_source = 'api' AND schedule_id IS NULL)
            OR
            (trigger_source = 'schedule' AND schedule_id IS NOT NULL)
        ),
    ADD CONSTRAINT agent_executions_input_object_check
        CHECK (jsonb_typeof(input_payload) = 'object'),
    ADD CONSTRAINT agent_executions_collector_prompt_check
        CHECK (
            agent_key <> 'collector'
            OR (
                prompt IS NOT NULL
                AND prompt_sha256 IS NOT NULL
                AND prompt_bytes IS NOT NULL
            )
        );

DROP INDEX agent_executions_one_active;

CREATE UNIQUE INDEX agent_executions_one_active
    ON agent_executions (agent_key)
    WHERE status IN ('queued', 'planning', 'collecting', 'materializing');

CREATE INDEX agent_executions_created_at_id_idx
    ON agent_executions (created_at DESC, execution_id DESC);

CREATE INDEX agent_executions_agent_key_created_at_id_idx
    ON agent_executions (agent_key, created_at DESC, execution_id DESC);
