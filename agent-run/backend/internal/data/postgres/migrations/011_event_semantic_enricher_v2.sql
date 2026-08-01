INSERT INTO agent_versions (version, agent_key)
VALUES ('event-semantic-enricher.v2', 'event-semantic-enricher')
ON CONFLICT (version) DO NOTHING;
