ALTER TABLE event_semantic_work_items
    DROP CONSTRAINT event_semantic_work_items_status_check;

ALTER TABLE event_semantic_work_items
    ADD CONSTRAINT event_semantic_work_items_status_check
    CHECK (status IN ('pending', 'running', 'succeeded', 'failed', 'skipped'));
