INSERT INTO agent_versions (version, agent_key)
VALUES ('event-fact-extractor.v2', 'event-fact-extractor')
ON CONFLICT (version) DO NOTHING;
