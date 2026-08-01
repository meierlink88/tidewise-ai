-- +goose Up
ALTER TABLE entity_type_definitions
    ADD COLUMN name_zh TEXT,
    ADD COLUMN name_en TEXT,
    ADD COLUMN business_definition TEXT,
    ADD COLUMN inclusion_criteria TEXT[],
    ADD COLUMN exclusion_criteria TEXT[],
    ADD COLUMN event_link_allowed BOOLEAN NOT NULL DEFAULT FALSE;

WITH definitions(
    type_key, version, name_zh, name_en, business_definition,
    inclusion_criteria, exclusion_criteria, event_link_allowed
) AS (
    VALUES
    ('alliance_org', 1, '联盟组织', 'Alliance Organization',
     '由多个独立成员基于共同章程、协议或长期协作机制组成，具有可识别组织身份的联盟或协会。',
     ARRAY['正式联盟、协会、商会、标准组织或跨机构协作组织', '具有稳定名称、成员边界或治理安排的多方组织']::text[],
     ARRAY['单次合作项目或临时会议', '国家、政府部门、企业集团、产业链或泛化行业集合']::text[], TRUE),
    ('chain_node', 1, '产业链节点', 'Industry Chain Node',
     '产业链中表示投入、制造、工艺、组件、设备、服务或关键经济环节的规范节点。',
     ARRAY['可在产业链中承担明确功能或生产环节的技术、工艺、设备、材料或服务类别', '可被多个企业参与且不等同于单一企业的经济环节']::text[],
     ARRAY['具体企业、品牌或证券', '独立可交易大宗商品', '完整产业链、泛化行业或一次性事件行为']::text[], TRUE),
    ('commodity', 1, '大宗商品', 'Commodity',
     '具有标准化品类、可形成公开或合同市场价格并可被交易或交付的基础商品。',
     ARRAY['能源、金属、农产品、化工原料等标准化可交易品类', '能够以数量、质量等级和单位价格描述的基础商品']::text[],
     ARRAY['具体品牌产品或企业', '证券、指数、期货合约本身', '产业链工艺节点或泛化原材料概念']::text[], TRUE),
    ('company', 1, '企业', 'Company',
     '具有独立经营身份的企业、集团或法人经营主体。',
     ARRAY['上市公司、非上市公司、集团公司及可识别经营主体', '拥有稳定企业名称并承担经营活动的法人或组织']::text[],
     ARRAY['企业发行的证券', '品牌、产品、产业链节点', '政府机构、联盟组织或临时项目']::text[], TRUE),
    ('concept', 1, '研究概念', 'Research Concept',
     '用于组织具有共同技术、商业模式或市场叙事特征对象的受控研究概念。',
     ARRAY['经正式目录治理的技术概念、商业模式概念或主题性研究标签', '可跨企业、产品或行业复用的稳定概念']::text[],
     ARRAY['具体企业、产品、证券或事件结论', '完整行业、产业链或产业链节点', '模型临时生成的热点或投资判断']::text[], TRUE),
    ('industry', 1, '行业', 'Industry',
     '按主要经济活动、产品或服务供给方式划分的稳定产业分类。',
     ARRAY['具有明确经营活动边界的正式行业分类', '可包含多家企业和多个产品类别的经济活动集合']::text[],
     ARRAY['证券市场板块、研究概念或完整产业链', '单一企业、产品或产业链节点', '临时热点和投资主题']::text[], TRUE),
    ('industry_chain', 1, '产业链', 'Industry Chain',
     '围绕明确目标产出与终端用途，由多个经济节点通过投入、组成、技术支撑或依赖形成的有边界研究子图。',
     ARRAY['具有明确目标产出、终端用途和成员节点边界的正式产业链', '包含多个相互依赖经济环节的受控研究对象']::text[],
     ARRAY['单一行业、概念、企业或产业链节点', '未定义边界的上下游泛称', '一次事件中的临时传导路径']::text[], TRUE),
    ('person', 1, '人物', 'Person',
     '能够以稳定姓名或正式身份识别的自然人。',
     ARRAY['企业负责人、政府官员、研究人员及其他可识别自然人', '原文中以姓名或无歧义正式称谓出现的个人']::text[],
     ARRAY['机构、职位本身、匿名群体或虚构主体', '仅能推断但原文未指明的个人']::text[], TRUE),
    ('policy_body', 1, '政策与监管机构', 'Policy and Regulatory Body',
     '依法或依正式授权制定、执行、解释政策并实施公共监管的政府部门、中央银行、法院或监管机构。',
     ARRAY['政府部门、中央银行、监管机关、法院及具有公共政策职能的正式机构', '能够发布政策、监管决定、司法裁决或官方公共措施的机构']::text[],
     ARRAY['商业公司、交易所、行业协会或临时工作组', '国家、地区或经济体本身', '机构负责人个人']::text[], TRUE),
    ('product', 1, '产品', 'Product',
     '企业生产、销售、采购或被市场需求的可识别产品、产品族或商业化服务对象。',
     ARRAY['具有稳定名称、型号、功能或商业用途的产品及产品族', '可由特定企业生产、销售或采购的商业化对象']::text[],
     ARRAY['生产工艺或产业链经济环节', '标准化大宗商品', '企业、品牌、证券或抽象研究概念']::text[], TRUE),
    ('sector', 1, '市场板块', 'Market Sector',
     '由市场或研究分类规则聚合的一组证券或企业板块，用于描述共同市场属性。',
     ARRAY['经正式目录治理的地域、风格、主题或行业市场板块', '由多个证券或企业构成的稳定市场分类']::text[],
     ARRAY['单一证券或企业', '实体行业、产业链或研究概念', '指数、交易所和临时行情描述']::text[], TRUE),
    ('security', 1, '证券', 'Security',
     '由发行主体发行并具有独立交易身份的股票、债券、基金份额或其他正式证券。',
     ARRAY['具有稳定证券身份、代码或发行关系的股票、债券和基金份额', '可与发行企业明确区分的交易标的']::text[],
     ARRAY['发行企业本身', '市场指数、期货合约或交易所', '商品、产品、行业和市场板块']::text[], TRUE)
)
UPDATE entity_type_definitions target
SET name_zh = definitions.name_zh,
    name_en = definitions.name_en,
    business_definition = definitions.business_definition,
    inclusion_criteria = definitions.inclusion_criteria,
    exclusion_criteria = definitions.exclusion_criteria,
    event_link_allowed = definitions.event_link_allowed
FROM definitions
WHERE target.type_key = definitions.type_key
  AND target.version = definitions.version;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM entity_type_definitions
        WHERE status = 'active'
          AND (
              NULLIF(btrim(name_zh), '') IS NULL
              OR NULLIF(btrim(name_en), '') IS NULL
              OR NULLIF(btrim(business_definition), '') IS NULL
              OR COALESCE(cardinality(inclusion_criteria), 0) = 0
              OR COALESCE(cardinality(exclusion_criteria), 0) = 0
          )
    ) THEN
        RAISE EXCEPTION 'every active Entity Type Definition requires complete V3 semantics';
    END IF;
END;
$$;
-- +goose StatementEnd

ALTER TABLE entity_type_definitions
    ALTER COLUMN name_zh SET NOT NULL,
    ALTER COLUMN name_en SET NOT NULL,
    ALTER COLUMN business_definition SET NOT NULL,
    ALTER COLUMN inclusion_criteria SET NOT NULL,
    ALTER COLUMN exclusion_criteria SET NOT NULL,
    ADD CONSTRAINT chk_entity_type_definition_names_v3 CHECK (
        btrim(name_zh) <> '' AND btrim(name_en) <> ''
    ),
    ADD CONSTRAINT chk_entity_type_definition_business_definition_v3 CHECK (
        btrim(business_definition) <> ''
    ),
    ADD CONSTRAINT chk_entity_type_definition_boundaries_v3 CHECK (
        cardinality(inclusion_criteria) > 0
        AND cardinality(exclusion_criteria) > 0
        AND array_position(inclusion_criteria, NULL::text) IS NULL
        AND array_position(exclusion_criteria, NULL::text) IS NULL
        AND array_to_string(inclusion_criteria, E'\x1F')
            !~ ('(^|' || E'\x1F' || ')[[:space:]]*(' || E'\x1F' || '|$)')
        AND array_to_string(exclusion_criteria, E'\x1F')
            !~ ('(^|' || E'\x1F' || ')[[:space:]]*(' || E'\x1F' || '|$)')
    );

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000037 is forward-only; roll back application binaries while retaining V3 Entity Type definitions';
END;
$$;
-- +goose StatementEnd
