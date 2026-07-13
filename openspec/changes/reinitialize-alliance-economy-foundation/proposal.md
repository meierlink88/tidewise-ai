## Why

现有联盟基础仅覆盖 10 个组织、50 个 economy 和 223 条 `member_of`，联盟 profile 仍使用 `org_code/org_type/primary_domain/scope_region/official_url`，无法表达已批准的多值分类、简称识别、主导方摘要与影响范围，也没有把联盟确认、经济体补齐和成员关系建立串成可审计闭环。与此同时，联盟研究 CSV 混有组织、战略资源、成员数快照和分析评级，必须先建立严格的 identity、字段与逐层 Review 契约，避免候选材料直接变成 seed 或关系事实。

## What Changes

- **BREAKING**：将 `alliance_org_profiles` 收敛为 `entity_id`、`abbreviation`、`categories TEXT[]`、`leadership_summary`、`influence_scope_summary`；联盟名称、规范名称、aliases 和状态继续只保存在 `entity_nodes`。非空简称同时进入 aliases，分类使用受控多值，禁止保存“政治 / 军事”拼接字符串。
- 复核现有 50 个 economy，建立 `country_code`、identity kind、ISO 3166、currency 和 region 的身份边界；主权国家、地区经济体、`economy:eu` 超国家聚合与 `economy:global` 全球聚合不得混淆，已有合格 identity 必须复用。
- 以 CSV 第 1—68 条作为联盟候选参考，但不生成可执行 seed；第 69—85 条战略矿产与农产品不属于 `alliance_org`，排除出本 change，仅记录为未来 `chain_node`、`commodity` 或 observation 候选，不自行创建后续 change。
- 明确不入库：子类、CSV 成员数、全球占比、约束力级别、影响力评级。正式成员数未来仅由 active `member_of` 计算；占比属于 observation；约束力和影响力属于后续分析评价。
- 建立不可跳过的依赖链：联盟 schema/data contract Review → 联盟候选逐项 Review → 从官方来源形成已批准联盟的正式成员全集 → 与现有 economy 做差异审计 → economy 候选 Review → `member_of` 候选 Review。联盟清单未确认不得冻结 economy 范围，economy 清单未确认不得生成关系候选。
- `member_of` 固定为 `economy -> alliance_org`，仅表达 active 正式成员；候选逐条展示成员身份与冲突报告，正式 edge 保存官方来源和核验时间。观察员、伙伴国、申请国、暂停成员和退出成员不得混入。CSV“成员数”仅供 Review 对照，不是事实源。
- `led_by` 固定为 `alliance_org -> economy/alliance_org`，只为可解析且有证据的核心主导方建边；“多边”“轮值”等保留在 `leadership_summary`，不得伪造实体。`part_of` 固定为下属 `alliance_org -> alliance_org` 上级组织。两者作为独立候选层，不阻塞 alliance/economy/`member_of` MVP。
- 后续有状态执行固定为 alliance Write → Query → economy Write → Query → `member_of` Review → Write → Query；上一层 Query 未经人工验收不得进入下一层。PostgreSQL 全部验收后，如需 Neo4j，必须单独 Review → Rebuild → Query。
- 不新增实体标签机制、不复用事件标签；PostgreSQL 继续作为事实源，Neo4j 保持单一 `Entity` label 与 `projection_namespace=tidewise`。
- 本 change 当前只进入 proposal Review。由于与 `refactor-industry-chain-node-foundation` 共享 entityfoundation seed/repository/migration 测试和 PostgreSQL 状态，源码实现、migration、seed、PostgreSQL/Neo4j 写入必须等待该 change 完成 Deliver，并重新基于最新 `origin/main` 评估后排队。

## Capabilities

### New Capabilities

- `alliance-economy-foundation`: 定义联盟最小 profile、经济体 identity/ISO 边界、候选来源与“联盟确认 → 补齐经济体 → 建立成员关系”的完整性门禁。

### Modified Capabilities

- `event-knowledge-schema`: 将联盟 profile 收敛到已批准字段，并补充 economy 聚合身份与 ISO 边界。
- `entity-foundation-seeds`: 将联盟和 economy 候选转换为正式 seed 前的逐层 Review、复用、差异审计与幂等校验要求。
- `entity-relationship-curation`: 增加 `member_of` 正式成员边界，以及独立的 `led_by`、`part_of` 关系契约和候选 Review 顺序。
- `neo4j-graph-projection-foundation`: 在 PostgreSQL 全部验收后，按独立授权投影已审阅联盟关系，并继续使用单一 `Entity` label。

## Impact

- 未来 Apply 预计影响 `backend/migrations/`、`backend/data/entity_foundation/`、`backend/internal/apps/entityfoundation/seed/`、`backend/internal/repositories/`、`backend/internal/apps/graphprojection/` 及对应 Go/migration 测试；当前 checkpoint 只新增 `openspec/changes/reinitialize-alliance-economy-foundation/` artifacts。
- CSV 仅作为只读候选材料，不复制到可执行 seed；其 68 条组织候选仍须逐项人工 Review，成员来源须另取可审计正式来源。
- 不修改 `prototype/` 或项目外 `doc/`，不涉及产业链实现、市场、benchmark/index、事件抽取/推理、观测数据实现、实体标签机制或股票推荐。
- 不执行 migration、seed、PostgreSQL/Neo4j 写入、投影重建或清理，不创建完成态 PR。
