> **状态：SUPERSEDED（2026-07-13）**
>
> 用户已决定由新的“统一产业链节点”架构整体替代本 proposal：取消 sector 逻辑实体、取消 `industry_chain` 容器、取消独立 membership，改为粗细粒度统一 `chain_node` 与单一 typed edge。本 change 只保留已实现代码、migration、seed 及实际 PG/Neo4j 状态的迁移谱系，不再代表目标架构；delta specs 不同步到主规格，后续只能由新 change 通过 forward migration 接管。

## Why

现有实体基础库只有 33 个扁平 `chain_node`，缺少独立产业链身份、链内成员、稳定拓扑和经证据审阅的物理约束，无法为全球事件到中国市场板块的产业链传导提供可靠静态骨架。本 change 收缩为静态产业链基础；动态 observation 平台必须由后续独立 change 设计，避免本 change 同时拥有主数据、采集治理和时序写入。

## What Changes

- 新增 `industry_chain` 实体及 `industry_chain_profiles`，保存稳定代码、定义、范围、版本、来源和 Review 状态。
- 增量改进既有 `chain_node_profiles`，补充节点分类、定义、分析单位和粒度说明；链内阶段与角色进入独立 membership。
- 新增 `industry_chain_memberships`，表达同一节点在不同产业链中的阶段、角色、顺序和核心性。
- 新增 `industry_chain_topology_edges`，以 `supplies_to`、`depends_on`、`substitutes_for` 表达链内稳定客观拓扑；商品投入/产出继续复用 `entity_edges`。
- 新增 `industry_chain_physical_constraints`，只保存经权威技术证据和人工 Review 批准的相对稳定物理/工程机制，不保存市场结构、认证、监管、融资、当前严重度、评分或投研结论。
- 扩展 `entity_edges` relationship policy，连接产业链/节点与 economy、commodity、benchmark、sector；全球 benchmark 必须经 chain/node 映射到中国 sector，海外 market 不得错误 `COVERS_SECTOR` 中国板块。
- 首批只以 AI 算力基础设施、半导体制造两条产业链作为 Review 候选，每链约 10–15 个节点，去重后约 20–30 个节点；机器人、新能源汽车/储能、创新药/生物制造移到第二批。
- Serenity 只提供“从系统变化识别物理卡点”的启发，不提供正式数据库 schema；本 change 的物理约束枚举是 Tidewise adaptation。

## Capabilities

### New Capabilities

- `industry-chain-foundation`: 定义静态产业链主数据、链内成员、稳定拓扑、物理约束、两条试点 Review 和未来 observation/reasoning 消费边界。

### Modified Capabilities

- `event-knowledge-schema`: 增加静态产业链 profile、membership、topology 和 physical constraint 的 PostgreSQL 增量 schema。
- `entity-foundation-seeds`: 将 `industry_chain` 纳入实体基础库，改进 `chain_node` profile，并为两条试点 seed 增加人工 Review 门禁。
- `entity-relationship-curation`: 增加产业链相关客观跨实体关系的类型、方向、来源和分层写入规则。
- `neo4j-graph-projection-foundation`: 增加产业链定义、membership、稳定拓扑及已审阅跨实体关系的 active-only 投影规则；物理约束保留在 PostgreSQL，不进入当前图投影。

## Impact

- 预期 Apply 只影响 `backend/migrations/`、`backend/internal/domain/`、`backend/internal/apps/entityfoundation/seed/`、`backend/internal/repositories/`、`backend/internal/apps/graphprojection/`、`backend/data/entity_foundation/` 及测试。
- `industry_chain_metric_definitions`、`industry_chain_metric_bindings`、通用 `observation_records`、node/flow observations、revision/quality/idempotency governance、ingestion observation writer 全部移到后续 `add-industry-chain-observation-foundation`。
- 本 change 不实现事件抽取、推理结果持久化、API、UI 或真实 observation connector；不修改 `frontend/`、`prototype/`、项目外 `doc/` 或其他 OpenSpec change。
- 不改变 PostgreSQL 事实源、Neo4j 可重建投影和 `Review → Write → Rebuild → Query` 分层门禁。
- Propose 阶段不执行任何有状态操作；进入 Apply 后，migration、seed/write、Neo4j rebuild 和 query 仍必须按层分别取得用户明确授权。
