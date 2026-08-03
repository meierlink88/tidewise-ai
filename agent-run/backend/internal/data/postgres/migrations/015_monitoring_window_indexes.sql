CREATE INDEX IF NOT EXISTS agent_executions_monitoring_window_idx
    ON agent_executions(agent_key, triggered_at DESC, execution_id DESC);

CREATE INDEX IF NOT EXISTS event_artifact_extraction_units_monitoring_window_idx
    ON event_artifact_extraction_units(updated_at DESC, unit_key DESC);

CREATE INDEX IF NOT EXISTS event_semantic_work_items_monitoring_window_idx
    ON event_semantic_work_items(updated_at DESC, work_item_id DESC);
