## Why

现有实体基础库已经有 `sector` 实体和 60 条同花顺风格板块快照，但它们仍偏向行情源清单，没有明确市场板块分类法、稳定标识策略、中文主名/英文别名规则，以及与市场、经济体、benchmark 的客观关系边界。事件驱动投研需要先把“板块”变成可审阅、可投影、可被事件推理引用的基础实体，否则后续事件抽取、产业链节点、商品和指标分析容易把 sector、industry-chain node、benchmark、commodity、metric 混成一层。

本 change 在提案阶段先完成 Explore -> Propose -> Validate；经后续逐阶段 Review 批准后，已实现 migration 文件、loader/repository、关系策略、投影映射和正式 seed，但尚未执行 migration apply、PostgreSQL 写入或 Neo4j 重建。

## What Changes

- 定义第一版市场板块实体基础能力，覆盖外部来源分类、语义板块分类、稳定 `entity_key`、中文主名称、英文 aliases、来源系统、市场范围、父子层级和可审阅快照字段。
- 明确同花顺“概念板块、行业板块、指数板块”三类在产品语义里都可以作为 `sector` 候选；`index_sector` 不因名称带“指数”而自动降级为 benchmark。
- 采纳已确认的 MVP 板块选择方法：三类各 Top 20 形成约 60 个原始候选池，按事件可解释性、传导独立性、行情敏感度、数据完整性、长期稳定性和市场代表性评分，原则上 70 分以上进入 MVP，最终形成约 50-60 个正式 sector。
- 明确板块与 `market`、`economy`、`benchmark` 的关系草案：PG 仍为实体与关系事实源，Neo4j 只投影已审阅的实体关系，不投影行情时序或事件推理结论。
- 收紧 external/source taxonomy、semantic sector 和 market benchmark 三层概念边界，避免把来源分类、可被事件影响的产业/主题暴露、用于量化验证的行情标尺混成同一个实体职责。
- 设计实现阶段应复用 `backend/data/entity_foundation/sectors.json`、`sector_profiles`、`entity_edges`、`relationship_policy.go` 和 graph projection 映射，不创建平行 seed、平行 profile 或平行图谱写入路径。
- 明确投研安全边界：板块基础数据只能表达客观分类、来源、市场范围和审阅关系，不表达具体股票推荐、买卖建议、涨跌预测、受益承压或传导强度。

## Capabilities

### New Capabilities
- `market-sector-foundation`: 定义市场板块基础实体、分类法、稳定标识、命名规则、客观关系边界、PG/Neo4j 边界和投研安全边界。

### Modified Capabilities
- `entity-foundation-seeds`: 将现有一阶段实体 seed 中的 `sector` 从快照清单收紧为可审阅的市场板块基础实体，并补充市场板块关系族的 seed/校验边界。
- `neo4j-graph-projection-foundation`: 补充板块实体和已审阅板块关系的 Neo4j 投影边界，明确不投影板块行情时序、事件推理结论或股票推荐。
- `market-benchmark-foundation`: 明确板块与 benchmark 的关系只能通过新的 `tracked_by_benchmark` 表达 `sector -> benchmark` 客观跟踪关系，不改变既有 `market -> observes_benchmark -> benchmark` 语义，不能把 benchmark 当板块、指标、商品或事件结论复制。

## Impact

- 仓库区域：涉及 `tidewise-ai` 源码工程内 OpenSpec artifacts，以及经阶段 Review 批准的 `backend/` 实现、`backend/data/entity_foundation/` seed、`backend/migrations/` 和相关测试。
- 不涉及 `prototype` 目录，不从高保真原型复制 HTML、DOM 操作或内联脚本。
- 不涉及上级 `doc` 目录；长期产品文档如需更新，应由独立文档 change 处理。
- 不修改或混入 active change `add-ai-event-extraction-pipeline`，也不触碰 `add-sdk-source-worker-connectors` worktree。
- 后续实现会影响后端实体基础库 seed、Go loader/validator、repository/migration、graph projection 映射和 OpenSpec 主规格；不新增前端 API、不新增小程序页面、不接入真实行情源；候选评分和核心/扩展/观察分层用于 Review 与推理调度，不得被误固化为实体身份。
