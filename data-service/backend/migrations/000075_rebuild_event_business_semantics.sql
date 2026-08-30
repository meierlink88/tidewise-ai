-- +goose Up

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM events) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'migration 000075 requires empty Event data; preserve a recovery point and republish Events through the coordinated business-semantic contract';
    END IF;
END
$$;
-- +goose StatementEnd

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
    AND NOT jsonb_path_exists(semantic, '$.time.* ? (@.type() == "string" && !(@ like_regex "^[0-9]{4}-[0-9]{2}-[0-9]{2}T.*Z$"))')
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
    AND NOT jsonb_path_exists(semantic, '$.metrics[*] ? (@.keyvalue().size() != 5)')
    AND NOT jsonb_path_exists(semantic, '$.metrics[*] ? (!exists(@.name) || !exists(@.value) || !exists(@.unit) || !exists(@.change) || !exists(@.period))')
    AND NOT jsonb_path_exists(semantic, '$.metrics[*] ? (@.name.type() != "string" || @.name == "")')
    AND NOT jsonb_path_exists(semantic, '$.metrics[*] ? (@.value.type() != "string" && @.value.type() != "null")')
    AND NOT jsonb_path_exists(semantic, '$.metrics[*] ? (@.unit.type() != "string" && @.unit.type() != "null")')
    AND NOT jsonb_path_exists(semantic, '$.metrics[*] ? (@.change.type() != "string" && @.change.type() != "null")')
    AND NOT jsonb_path_exists(semantic, '$.metrics[*] ? (@.period.type() != "string" && @.period.type() != "null")')
    AND NOT jsonb_path_exists(semantic, '$.metrics[*] ? (@.value.type() == "null" && @.change.type() == "null")')
);

-- +goose Down

-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000075 is a forward-only Event semantic cutover; restore the pre-migration recovery point with the previous application and AgentOS releases';
END
$$;
-- +goose StatementEnd
