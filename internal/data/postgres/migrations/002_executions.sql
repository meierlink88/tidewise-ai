CREATE TABLE IF NOT EXISTS agent_executions (
    execution_id uuid PRIMARY KEY,
    agent_version text NOT NULL REFERENCES agent_versions(version),
    idempotency_key text NOT NULL UNIQUE,
    prompt text NOT NULL,
    prompt_sha256 char(64) NOT NULL,
    prompt_bytes integer NOT NULL CHECK (prompt_bytes >= 0),
    status text NOT NULL CHECK (status IN ('queued', 'planning', 'collecting', 'materializing', 'succeeded', 'succeeded_no_change', 'partially_succeeded', 'failed')),
    error_code text,
    error_summary text,
    candidate_counts jsonb NOT NULL DEFAULT '{}'::jsonb,
    artifacts jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL,
    started_at timestamptz,
    completed_at timestamptz,
    updated_at timestamptz NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS agent_executions_one_active
ON agent_executions ((true))
WHERE status IN ('queued', 'planning', 'collecting', 'materializing');

CREATE TABLE IF NOT EXISTS connector_invocations (
    execution_id uuid NOT NULL REFERENCES agent_executions(execution_id) ON DELETE CASCADE,
    connector_key text NOT NULL,
    position smallint NOT NULL,
    status text NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    result_count integer NOT NULL DEFAULT 0 CHECK (result_count >= 0),
    error_code text,
    error_summary text,
    started_at timestamptz,
    completed_at timestamptz,
    PRIMARY KEY (execution_id, connector_key),
    UNIQUE (execution_id, position)
);
