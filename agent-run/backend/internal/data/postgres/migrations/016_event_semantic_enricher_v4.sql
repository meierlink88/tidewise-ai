INSERT INTO agent_versions (version, agent_key)
VALUES ('event-semantic-enricher.v4', 'event-semantic-enricher')
ON CONFLICT (version) DO NOTHING;
