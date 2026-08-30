-- +goose Up

-- Event metric content is validated by the typed HTTP and Biz boundaries.
-- PostgreSQL protects the Event semantic envelope and projection invariants,
-- but must not duplicate optional metric-field validation with fragile JSONPath.
ALTER TABLE events DROP CONSTRAINT chk_events_semantic;
ALTER TABLE events ADD CONSTRAINT chk_events_semantic CHECK (
    jsonb_typeof(semantic) = 'object'
    AND jsonb_array_length(jsonb_path_query_array(semantic, '$.keyvalue()')) = 10
    AND semantic ?& ARRAY['actors','action','objects','stage','modality','time','jurisdictions','reason','method','metrics']
    AND jsonb_typeof(semantic -> 'actors') = 'array'
    AND jsonb_array_length(semantic -> 'actors') > 0
    AND NOT jsonb_path_exists(semantic, '$.actors[*] ? (@.type() != "string" || @ == "")')
    AND jsonb_typeof(semantic -> 'action') = 'string'
    AND btrim(semantic ->> 'action') <> ''
    AND jsonb_typeof(semantic -> 'objects') = 'array'
    AND jsonb_array_length(semantic -> 'objects') > 0
    AND NOT jsonb_path_exists(semantic, '$.objects[*] ? (@.type() != "string" || @ == "")')
    AND semantic ->> 'stage' IN ('OCCURRED','ANNOUNCED','EFFECTIVE','IMPLEMENTED','UPDATED','SUSPENDED','TERMINATED','EXPECTED')
    AND semantic ->> 'modality' IN ('FACT','PLAN','SPEC')
    AND jsonb_typeof(semantic -> 'time') = 'object'
    AND jsonb_array_length(jsonb_path_query_array(semantic -> 'time', '$.keyvalue()')) = 4
    AND semantic -> 'time' ?& ARRAY['occurred_at','announced_at','effective_at','precision']
    AND jsonb_typeof(semantic #> '{time,occurred_at}') IN ('string','null')
    AND jsonb_typeof(semantic #> '{time,announced_at}') IN ('string','null')
    AND jsonb_typeof(semantic #> '{time,effective_at}') IN ('string','null')
    AND (
        jsonb_typeof(semantic #> '{time,occurred_at}') = 'string'
        OR jsonb_typeof(semantic #> '{time,announced_at}') = 'string'
        OR jsonb_typeof(semantic #> '{time,effective_at}') = 'string'
    )
    AND (
        (semantic #>> '{time,occurred_at}') IS NULL
        OR (semantic #>> '{time,occurred_at}') ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T.*Z$'
    )
    AND (
        (semantic #>> '{time,announced_at}') IS NULL
        OR (semantic #>> '{time,announced_at}') ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T.*Z$'
    )
    AND (
        (semantic #>> '{time,effective_at}') IS NULL
        OR (semantic #>> '{time,effective_at}') ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T.*Z$'
    )
    AND semantic #>> '{time,precision}' IN ('INSTANT','DAY','RANGE','MONTH','QUARTER','YEAR','UNKNOWN')
    AND modality = semantic ->> 'modality'
    AND occurred_at IS NOT DISTINCT FROM (semantic #>> '{time,occurred_at}')::timestamptz
    AND announced_at IS NOT DISTINCT FROM (semantic #>> '{time,announced_at}')::timestamptz
    AND jsonb_typeof(semantic -> 'jurisdictions') = 'array'
    AND NOT jsonb_path_exists(semantic, '$.jurisdictions[*] ? (@.type() != "string" || @ == "")')
    AND jsonb_typeof(semantic -> 'reason') IN ('string','null')
    AND jsonb_typeof(semantic -> 'method') IN ('string','null')
    AND jsonb_typeof(semantic -> 'metrics') = 'array'
    AND NOT jsonb_path_exists(semantic, '$.metrics[*] ? (@.type() != "object")')
);

-- +goose Down

-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000076 is a forward-only Event metric constraint repair; restore a pre-migration recovery point instead of reinstating the defective constraint';
END
$$;
-- +goose StatementEnd
