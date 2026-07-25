CREATE TABLE IF NOT EXISTS agent_definitions (
    agent_key text PRIMARY KEY,
    display_name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS agent_versions (
    version text PRIMARY KEY,
    agent_key text NOT NULL REFERENCES agent_definitions(agent_key),
    created_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO agent_definitions (agent_key, display_name)
VALUES ('collector', 'Collector Agent')
ON CONFLICT (agent_key) DO NOTHING;

INSERT INTO agent_versions (version, agent_key)
VALUES ('collector.v1', 'collector')
ON CONFLICT (version) DO NOTHING;
