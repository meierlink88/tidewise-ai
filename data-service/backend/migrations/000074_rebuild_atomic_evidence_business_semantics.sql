-- +goose Up
-- Issue #351 authorizes a zero-compatibility Evidence contract cutover.
-- Stop Raw Evidence, Evidence and Event writers and preserve a reviewed
-- PostgreSQL recovery point before applying this migration.

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM raw_evidences)
       OR EXISTS (SELECT 1 FROM evidences)
       OR EXISTS (SELECT 1 FROM events)
       OR EXISTS (SELECT 1 FROM event_evidence_links)
       OR EXISTS (SELECT 1 FROM event_actor_links)
       OR EXISTS (SELECT 1 FROM event_asset_links)
       OR EXISTS (SELECT 1 FROM event_publication_receipts) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'migration 000074 requires empty Raw Evidence, Evidence and Event history; preserve the recovery point and clear the coordinated dataset before retrying';
    END IF;
END;
$$;
-- +goose StatementEnd

ALTER TABLE raw_evidences
    DROP CONSTRAINT chk_raw_evidences_keywords_one_dimension,
    DROP COLUMN keywords;

ALTER TABLE evidences
    DROP CONSTRAINT chk_evidences_semantic,
    ADD COLUMN keywords TEXT[] NOT NULL,
    ADD CONSTRAINT chk_evidences_keywords CHECK (
        COALESCE(array_ndims(keywords), 1) = 1
        AND cardinality(keywords) BETWEEN 1 AND 5
        AND array_position(keywords, NULL::TEXT) IS NULL
    ),
    ADD CONSTRAINT chk_evidences_semantic CHECK (
        jsonb_typeof(semantic) = 'object'
        AND semantic = jsonb_build_object(
            'actors', semantic -> 'actors',
            'action', semantic -> 'action',
            'objects', semantic -> 'objects',
            'stage', semantic -> 'stage',
            'modality', semantic -> 'modality',
            'time', semantic -> 'time',
            'jurisdictions', semantic -> 'jurisdictions',
            'reason', semantic -> 'reason',
            'method', semantic -> 'method',
            'metrics', semantic -> 'metrics',
            'attribution', semantic -> 'attribution'
        )
        AND jsonb_typeof(semantic -> 'actors') = 'array'
        AND jsonb_array_length(semantic -> 'actors') BETWEEN 1 AND 20
        AND jsonb_typeof(semantic -> 'action') = 'string'
        AND btrim(semantic ->> 'action') <> ''
        AND jsonb_typeof(semantic -> 'objects') = 'array'
        AND jsonb_array_length(semantic -> 'objects') BETWEEN 1 AND 20
        AND semantic ->> 'stage' IN ('OCCURRED', 'ANNOUNCED', 'EFFECTIVE', 'IMPLEMENTED', 'UPDATED', 'SUSPENDED', 'TERMINATED', 'EXPECTED')
        AND semantic ->> 'modality' IN ('FACT', 'PLAN', 'SPEC')
        AND jsonb_typeof(semantic -> 'jurisdictions') = 'array'
        AND jsonb_array_length(semantic -> 'jurisdictions') <= 20
        AND jsonb_typeof(semantic -> 'metrics') = 'array'
        AND (jsonb_typeof(semantic -> 'reason') = 'null' OR jsonb_typeof(semantic -> 'reason') = 'string')
        AND (jsonb_typeof(semantic -> 'method') = 'null' OR jsonb_typeof(semantic -> 'method') = 'string')
        AND jsonb_typeof(semantic -> 'time') = 'object'
        AND semantic -> 'time' = jsonb_build_object(
            'raw', semantic #> '{time,raw}',
            'start_at', semantic #> '{time,start_at}',
            'end_at', semantic #> '{time,end_at}',
            'precision', semantic #> '{time,precision}'
        )
        AND semantic #>> '{time,precision}' IN ('INSTANT', 'DAY', 'RANGE', 'MONTH', 'QUARTER', 'YEAR', 'UNKNOWN')
        AND (jsonb_typeof(semantic #> '{time,raw}') = 'null' OR jsonb_typeof(semantic #> '{time,raw}') = 'string')
        AND (jsonb_typeof(semantic #> '{time,start_at}') = 'null' OR jsonb_typeof(semantic #> '{time,start_at}') = 'string')
        AND (jsonb_typeof(semantic #> '{time,end_at}') = 'null' OR jsonb_typeof(semantic #> '{time,end_at}') = 'string')
        AND jsonb_typeof(semantic -> 'attribution') = 'object'
        AND semantic -> 'attribution' = jsonb_build_object(
            'reported_by', semantic #> '{attribution,reported_by}',
            'claimed_by', semantic #> '{attribution,claimed_by}'
        )
        AND (jsonb_typeof(semantic #> '{attribution,reported_by}') = 'null' OR jsonb_typeof(semantic #> '{attribution,reported_by}') = 'string')
        AND (jsonb_typeof(semantic #> '{attribution,claimed_by}') = 'null' OR jsonb_typeof(semantic #> '{attribution,claimed_by}') = 'string')
    );

COMMENT ON COLUMN evidences.keywords IS '发布方生成的 Evidence 阅读辅助关键词；Data 按给定顺序保存。';
COMMENT ON COLUMN evidences.semantic IS '最小完整业务命题语义：主体、动作、对象、阶段、模态、时间、辖区、原因、方式、指标与归因。';

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000074 is a forward-only Evidence contract cutover; restore the pre-migration recovery point with the previous applications';
END;
$$;
-- +goose StatementEnd
