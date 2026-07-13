## Why

前序 `add-industry-chain-node-foundation` 将产业链表达拆成 sector、industry_chain、membership 与 topology 多套并行结构，造成身份重复、层级固化、关系语义重叠，也难以支撑事件实体链接与可解释推理。本 change 以前序已交付的 PostgreSQL facts 和既有 Neo4j projection 为前向迁移输入，建立单一产业节点事实模型，并把 Tidewise 自有投研视角从产业分类中明确分离。

## What Changes

- **BREAKING**：取消 `sector` 逻辑实体、独立 `industry_chain` 容器实体与 `industry_chain_memberships`；粗细产业概念统一为 `entity_type=chain_node`，不存全局固定层级、父节点或产业链名称。
- 将 `chain_node_profiles` 收敛为 `entity_id`、必填 `definition` 与可选 `boundary_note`；身份、中文名、英文及其他 aliases、状态和时间继续复用 `entity_nodes`。
- 新增 `entity_type=theme` 与最小 `theme_profiles`，中文展示为“投研主题”；theme 是 Tidewise 自有分析视角，不是 sector、指数、产业链容器或证券集合。本 change 不确定任何具体 theme 实例，也不保留 `research_theme` 兼容命名。
- 不建设来源映射表。同花顺、东方财富分类只作为候选节点 Review 与 seed 评审证据，过滤涨停、融资融券、高股息等非产业标签，不进入生产主数据。
- **BREAKING**：以唯一关系事实表 `chain_node_relations` 取代 `industry_chain_topology_edges`，不复用 `entity_edges`，也不恢复 membership/topology 双表。MVP 仅允许 `is_subcategory_of`、`is_component_of`、`input_to`、`depends_on` 四类有向关系。
- 删除 `contains`、`supplies_to`、`substitutes_for`、`transmits_to`。事件传导方向、强度、时滞和结论由事件沿静态关系动态推理，不写入关系主数据；同一机制不得同时登记为 `input_to` 与 `depends_on`。
- physical constraints 继续独立保存，并将旧 topology edge 引用经审批映射到 `chain_node_relations.id`；不设计观测数据。
- 通过版本化、幂等的前向迁移处理既有 PostgreSQL facts；既有 Neo4j projection 仅作为待后续 change 处理的旧投影，不清库、不回滚历史，也不在本 change 重建。
- 将 Apply 严格拆成 Phase A“统一节点模型与节点初始化”和 Phase B“节点关系建立”。每阶段均设置候选数据人工 Review、PostgreSQL Write 单独授权与写后 Query 验收，且 Phase A 完整验收前不得进入 Phase B。
- `entity_key` 全局唯一约束仅在全库 preflight 证明安全后才允许实施，否则保持现状并记录阻断项。

## Capabilities

### New Capabilities

- `industry-chain-node-foundation`: 定义统一 chain_node 身份、最小 profile、四类静态关系、physical constraint 迁移边界及分阶段有状态操作门禁。
- `theme-foundation`: 定义 Tidewise 投研主题实体的最小主数据契约、与 chain_node/sector/index/证券集合的边界，以及未来 theme-node 关联的隔离要求。

### Modified Capabilities

- `event-knowledge-schema`: 将产业实体类型收敛为 chain_node，并增加 theme 实体类型，避免事件实体链接继续产生 sector 或 industry_chain 容器身份。
- `entity-foundation-seeds`: 将产业基础数据初始化改为统一 chain_node 候选 Review 与分层写入，不再初始化 sector 来源映射、industry_chain 容器或 memberships。
- `entity-relationship-curation`: 将产业链静态关系限定为独立 `chain_node_relations` 的四类可判定语义，并明确它与通用 `entity_edges` 及动态事件传导的边界。
- `market-sector-foundation`: 移除 sector 作为生产逻辑实体、sector profile、外部分类映射与 sector 图投影输入的要求。

## Impact

- 未来 Apply 预计影响 `backend/migrations/`、`backend/internal/domain/`、实体 seed 与 repository、产业关系与 physical constraint 数据访问、相关测试及受影响 seed manifests；本 Proposal checkpoint 不修改这些源码。
- PostgreSQL 仍是实体与关系事实源；迁移必须保留或显式收敛旧 stable ID/key、来源证据和 constraint subject，且所有 schema/data Write 均需独立授权。
- Neo4j 不在本 change 构建或重建；旧 projection 不得清理，后续须由独立 change 消化 PostgreSQL 新事实模型。
- 不修改 `prototype/` 或项目外 `doc/`，不调整 alliance、economy/country、market、benchmark/index，不包含事件提取/推理、观测数据或股票推荐。
