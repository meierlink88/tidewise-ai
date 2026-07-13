## Why

前序 `add-industry-chain-node-foundation` 将产业链表达拆成 sector、industry_chain、membership 与 topology 多套并行结构，造成身份重复、层级固化、关系语义重叠，也难以支撑事件实体链接与可解释推理。本 change 将先备份并审计所有引用，再以版本化 migration/受控命令完整清除 PostgreSQL 中的旧产业模型及其遗留事实，随后只从最终人工批准的新清单全新导入 chain_node，并把 Tidewise 自有投研视角从产业分类中明确分离。

## What Changes

- **BREAKING**：取消 `sector` 逻辑实体、独立 `industry_chain` 容器实体与 `industry_chain_memberships`；粗细产业概念统一为 `entity_type=chain_node`，不存全局固定层级、父节点或产业链名称。
- 将 `chain_node_profiles` 收敛为 `entity_id`、必填 `definition` 与可选 `boundary_note`；身份、中文名、英文及其他 aliases、状态和时间继续复用 `entity_nodes`。
- 新增 `entity_type=theme` 与最小 `theme_profiles`，中文展示为“投研主题”；theme 是 Tidewise 自有分析视角，不是 sector、指数、产业链容器或证券集合。本 change 不确定任何具体 theme 实例，也不保留 `research_theme` 兼容命名。
- 不建设来源映射表。同花顺、东方财富分类只作为候选节点 Review 与 seed 评审证据，过滤涨停、融资融券、高股息等非产业标签，不进入生产主数据。
- **BREAKING**：以唯一关系事实表 `chain_node_relations` 取代 `industry_chain_topology_edges`，不复用 `entity_edges`，也不恢复 membership/topology 双表。MVP 仅允许 `is_subcategory_of`、`is_component_of`、`input_to`、`depends_on` 四类有向关系。
- 删除 `contains`、`supplies_to`、`substitutes_for`、`transmits_to`。事件传导方向、强度、时滞和结论由事件沿静态关系动态推理，不写入关系主数据；同一机制不得同时登记为 `input_to` 与 `depends_on`。
- 历史 physical constraints 与旧 topology subject 一并进入清理范围，不迁移旧 ID 或旧语义；未来若需要节点/关系约束，只能基于新节点和新关系重新 Review 后创建。不设计观测数据。
- 先生成可恢复备份和完整引用审计，再通过版本化、幂等 migration/受控命令清除旧 PostgreSQL facts 与专属表；禁止手工清库、历史回滚或把旧 UUID/entity_key 收敛到新节点。
- 最新候选工作簿包含 955 个“初步保留”、34 个“待复核”和 202 个“明确排除”，仍不是最终 seed。待复核、同义归并、粒度及层级关系全部由主对话 Review 完成后，才能生成全新 chain_node seed。
- 将 Apply 严格拆成 Phase A“cleanup 后全新节点模型/初始化”和 Phase B“基于新节点建立关系”。cleanup 与 final seed 分别独立执行 `Review -> Write -> Query`，两次 Write 分别授权；Phase A 完整验收前不得进入 Phase B。
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
- PostgreSQL 仍是实体与关系事实源；旧 sector、industry_chain、chain_node、membership、topology、physical constraint、相关 `entity_edges`、event links 与 convergence/audit 引用必须按审计结果受控删除，不复用旧 stable ID/key。
- Neo4j 不在本 change 清理、写入或重建；PG cleanup 后既有 projection 将明确处于陈旧状态，后续须由独立 change 重新投影。
- 不修改 `prototype/` 或项目外 `doc/`，不调整 alliance、economy/country、market、benchmark/index，不包含事件提取/推理、观测数据或股票推荐。
