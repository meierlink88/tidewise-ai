-- +goose Up
INSERT INTO entity_type_definitions (
    type_key,
    version,
    signal_subject_allowed,
    direct_target_mode,
    status,
    allowed_event_roles,
    name_zh,
    name_en,
    business_definition,
    inclusion_criteria,
    exclusion_criteria,
    event_link_allowed
) VALUES
    (
        'economy',
        1,
        FALSE,
        'context',
        'active',
        ARRAY['event_subject', 'affected_entity', 'event_object', 'context']::text[],
        '经济体',
        'Economy',
        '用于宏观经济分析和统计的国家、地区或跨国家经济区域。',
        ARRAY[
            '作为宏观经济、贸易、通胀、增长或统计口径对象的国家与地区',
            '欧元区、欧盟等具有稳定经济分析口径的跨国家区域'
        ]::text[],
        ARRAY[
            '政府部门、中央银行或监管机构',
            '证券市场、交易所、指数、企业或产业链',
            '仅作为地理位置或政治行为主体且未表达经济体语义的国家提及'
        ]::text[],
        TRUE
    ),
    (
        'index',
        1,
        FALSE,
        'deny',
        'active',
        ARRAY['event_subject', 'affected_entity', 'event_object', 'context']::text[],
        '市场指数',
        'Market Index',
        '按照稳定编制规则计算，用于衡量一组证券、商品、汇率或市场表现的指标化对象。',
        ARRAY[
            '股票、债券、商品、汇率或波动率等正式编制指数',
            '具有稳定名称、代码或编制口径并可持续观测的市场指数'
        ]::text[],
        ARRAY[
            '单一证券、企业、商品或金融合约',
            '普通经营指标、临时统计值、交易所或市场板块'
        ]::text[],
        TRUE
    ),
    (
        'instrument',
        1,
        FALSE,
        'deny',
        'active',
        ARRAY['event_subject', 'affected_entity', 'event_object', 'context']::text[],
        '金融工具',
        'Financial Instrument',
        '具有可交易或可订立合约属性的金融资产类别或标准化金融工具。',
        ARRAY[
            '股票、外汇、商品期货和数字资产等受控金融工具类别',
            '能够作为交易、监管或资金配置对象的标准化金融工具'
        ]::text[],
        ARRAY[
            '具体发行企业、单只证券或市场指数',
            '标的商品、交易场所、研究概念或市场板块'
        ]::text[],
        TRUE
    ),
    (
        'market',
        1,
        FALSE,
        'deny',
        'active',
        ARRAY[
            'event_subject',
            'actor',
            'affected_entity',
            'statement_source',
            'event_object',
            'context'
        ]::text[],
        '交易市场',
        'Trading Market',
        '具有稳定交易范围、组织规则或交易场所身份的正式市场或交易所。',
        ARRAY[
            '证券交易所、期货交易所及其他正式交易场所',
            '按地域或资产类别形成且具有稳定边界的股票、债券、外汇或商品交易市场'
        ]::text[],
        ARRAY[
            '交易所运营企业的证券',
            '单一证券、指数、金融工具或市场板块',
            '泛指的市场情绪、需求或价格变化'
        ]::text[],
        TRUE
    )
ON CONFLICT (type_key, version) DO UPDATE
SET signal_subject_allowed = EXCLUDED.signal_subject_allowed,
    direct_target_mode = EXCLUDED.direct_target_mode,
    status = EXCLUDED.status,
    allowed_event_roles = EXCLUDED.allowed_event_roles,
    name_zh = EXCLUDED.name_zh,
    name_en = EXCLUDED.name_en,
    business_definition = EXCLUDED.business_definition,
    inclusion_criteria = EXCLUDED.inclusion_criteria,
    exclusion_criteria = EXCLUDED.exclusion_criteria,
    event_link_allowed = EXCLUDED.event_link_allowed;

-- +goose StatementBegin
DO $$
BEGIN
    IF (
        SELECT count(*)
        FROM entity_type_definitions
        WHERE version = 1
          AND status = 'active'
          AND type_key IN ('economy', 'index', 'instrument', 'market')
          AND event_link_allowed
          AND NULLIF(btrim(name_zh), '') IS NOT NULL
          AND NULLIF(btrim(name_en), '') IS NOT NULL
          AND NULLIF(btrim(business_definition), '') IS NOT NULL
          AND cardinality(inclusion_criteria) > 0
          AND cardinality(exclusion_criteria) > 0
          AND cardinality(allowed_event_roles) > 0
    ) <> 4 THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            MESSAGE = 'Event Semantic Entity Type catalog completion failed';
    END IF;
END;
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000040 is forward-only; retain the completed Entity Type catalog during application rollback';
END;
$$;
-- +goose StatementEnd
