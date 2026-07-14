# 第一批 chain_node 数据契约审阅

## 审阅状态与边界

- 本材料只用于 first-batch data contract Review，不是可执行 seed，也不授权 migration、cleanup、schema Write、seed Write、PostgreSQL/Neo4j 写入或关系阶段。
- structure implementation checkpoint `0f20171` 已由主对话复验通过；该结论只允许继续本契约审阅。
- 审阅输入为 `/Users/meierlink/.codex/visualizations/2026/07/12/019f5477-445d-75d3-acf2-61a4fdd5b1d4/outputs/产业链节点候选-稳定节点宽口径筛选与合并.xlsx`，第一批范围只取 Sheet「标准化保留」。

## 只读核验结果

| 断言 | 只读结果 | 后续 Query / report 口径 |
|---|---:|---|
| 标准化节点 | 842 | `canonical_name` 互异；第一批 node/profile 目标数 |
| 原始名称 | 950 | 互异；canonical/name/aliases 合计覆盖 |
| 同义合并减少 | 108 | `950 - 842` |
| 宽边界审阅节点 | 79 | 已保留子集，不是排除清单 |
| 外部代码 | 1,156 | 每个代码拆成一条 identifier row |
| 东方财富代码 | 811 | `source_system=eastmoney` |
| 同花顺代码 | 345 | `source_system=ths` |
| 双来源节点 | 241 | 同一 canonical 同时存在 eastmoney/ths |
| 跨节点代码冲突 | 0 | 按来源+代码复核；生产唯一键还包含 taxonomy |

已批准的名称归并示例保持不变：冰雪经济归入冰雪产业；白酒Ⅱ、白酒Ⅲ、白酒概念归入白酒；航空发动机作为稳定产品/设备节点保留。

## 对旧决策的技术核对

旧 artifacts 中“不建立 source mapping”的目的，是禁止恢复 `sector_source_mappings`、禁止创建 `chain_node_source_mappings`，以及禁止把 provider/code/source 塞进 `chain_node_profiles`。它不应扩大解释为“外部 identity 永远不能进入生产库”。

本次调整采用通用 `entity_external_identifiers`：只表示内部实体与外部系统标识的绑定，不表达 sector 语义等价、成员关系、层级、市场归属或来源证据链。因此：

- `chain_node_profiles(entity_id, definition, boundary_note)` 保持不变；
- 旧 `sector_source_mappings` 继续受控 cleanup，不迁移；
- 不创建任何 chain_node 专属 mapping 表；
- 本 change 首批只允许 chain_node 的 eastmoney/ths 标识，不顺带处理其他实体；
- 外部筛选证据、链接、快照和 reviewer 结论仍留在 Review/seed report，不进入 identifier 主表。

## 推荐 schema

```text
entity_external_identifiers
  id                   UUID         NOT NULL PRIMARY KEY
  entity_id            UUID         NOT NULL REFERENCES entity_nodes(id) ON DELETE CASCADE
  source_system        TEXT         NOT NULL CHECK (btrim(source_system) <> '')
  source_taxonomy_type TEXT         NOT NULL CHECK (btrim(source_taxonomy_type) <> '')
  external_code        TEXT         NOT NULL CHECK (btrim(external_code) <> '')
  external_name        TEXT         NOT NULL CHECK (btrim(external_name) <> '')
  status               VARCHAR(32)  NOT NULL DEFAULT 'active' CHECK (status IN ('active','inactive'))
  created_at           TIMESTAMPTZ  NOT NULL DEFAULT now()
  updated_at           TIMESTAMPTZ  NOT NULL DEFAULT now()

UNIQUE (source_system, source_taxonomy_type, external_code)
INDEX  (entity_id, source_system, source_taxonomy_type)
```

推荐不建立 `UNIQUE(entity_id, source_system, source_taxonomy_type, external_code)`：外部 identity 唯一约束已经逻辑蕴含四列 tuple 唯一，第二个唯一索引不会增加一致性，只增加存储和写放大。若主对话要求保留，须在 schema Review 明确接受该成本。

不采用 PostgreSQL enum：通用表未来可能接入其他来源或 taxonomy；数据库只约束非空，首批 application/seed validator 严格限定 `eastmoney`、`ths` 与 `industry_sector`、`concept_sector`、`index_sector`。不采用 JSONB：字段均参与唯一性、过滤、校验或审计，强类型列更直接。

## Excel 到 seed report 的转换契约

1. 冻结「标准化保留」842 个 canonical，拒绝其他 Sheet 或新增名称进入本批。
2. `entity_nodes.name` 与 `canonical_name` 使用「标准化节点名」。
3. 拆分「原始别名」；去首尾空白、稳定排序、去重。与 canonical 相同的名称由 name/canonical_name 保存，不在 aliases 重复；其余审阅原名全部进入 aliases。
4. 使用获批契约生成全新 UUID/entity_key；禁止读取、复用或收敛旧 sector/industry_chain/chain_node ID/key。
5. 逐项生成 definition draft 与必要 boundary draft，经人工 Review 后才能进入 final seed report。
6. 拆分「来源代码」，从「原名保留明细」或来源侧逐代码证据恢复 source_system、source_taxonomy_type、external_code、external_name；先绑定已批准的新 entity identity，再生成逐行 mapping report。
7. dry-run 只输出 create/unchanged/conflict、预计 counts、identity/profile/identifier 校验和与阻断项，不写数据库。

不得将 `东方财富:BK...；同花顺:...` 保存为一个字段值，也不得用标准化 canonical 覆盖平台原始 `external_name`。

## Taxonomy 消歧阻断项

「标准化保留」没有逐代码 taxonomy；「原名保留明细」中 6 条记录把来源分类聚合成组合值，共涉及 13 个代码：白酒、家用电器、跨境电商、汽车整车、燃料电池、物业管理。组合字符串不能证明每个代码属于 industry/concept/index，尤其跨境电商包含两个东方财富代码。

实现不得依靠代码前缀猜测 taxonomy。schema/TDD implementation 可以先实现“未决 taxonomy 阻断”规则，但 final seed dry-run 前必须补齐逐代码来源证据并人工 Review。任一记录仍为组合值时，报告必须失败而不是写入含混 taxonomy。

## Definition / boundary Review 策略

- definition 必须回答“该节点是什么”，采用“上位类别或对象 + 可判定差异/核心功能或产出”结构。
- 禁止空值、仅复制 canonical/alias、以及“与 X 相关的产业链节点”“X 相关产业”等循环模板。
- 每条 draft 至少通过：非空、非名称复刻、非模板、与相邻/同义节点不冲突、reviewer 明确通过。
- 名称合并、同名消歧、粗细范围重叠与 79 个宽边界节点优先填写 boundary_note，明确包含与排除；边界清晰节点保持 NULL，不默认空字符串。
- 工作簿的 GB/T、ISIC、NAICS 候选只能辅助定义和边界 Review，不自动转成层级、parent、relation 或生产 taxonomy。
- definition/boundary 审阅材料保留 draft、证据与 reviewer 状态；生产 `chain_node_profiles` 不增加 evidence JSONB 或 review workflow 字段。

## 幂等与冲突规则

- node/profile：以最终获批的新 entity_key 作为幂等键；UUID/key/canonical 任一交叉冲突都必须报告 conflict，禁止静默换 ID 或复用旧身份。
- external identifier：以 `(source_system, source_taxonomy_type, external_code)` 为冲突键；同 entity 可 unchanged 或更新 external_name/status，entity_id 变化必须阻断，禁止静默换绑。
- 预期 counts 不一致、alias 丢失、definition/boundary 不合格、taxonomy 未决、代码跨实体冲突或非目标实体变化时，dry-run 与 Write 都必须停止。

## 待主对话确认

1. 批准通用 `entity_external_identifiers` 字段与“单一外部 identity 唯一约束 + entity 查询索引”的推荐方案。
2. 批准 842/950/1,156 的转换与校验口径，以及 13 个组合 taxonomy 代码必须逐项消歧的阻断规则。
3. 批准 definition/boundary 生成与人工 Review 策略、全新 UUID/entity_key 和 dry-run/report 作为后续实现门禁。
4. 批准顺序：contract Review → schema/TDD implementation Review → cleanup Review/Write/Query → external identifier schema Review/Write/Query → final node/profile dry-run Review/Write/Query → mapping data Review/Write/Query → Phase A 验收。
