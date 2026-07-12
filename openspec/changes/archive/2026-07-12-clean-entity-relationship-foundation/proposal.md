## Why

当前实体主数据已具备可用基础，但 `relationships.json` 中仅有 78 条样例关系，成员关系不完整，部分关系缺少明确语义和可审计来源。事件提取流水线开始前，需要先建立空关系基线，再按实体层级逐批审阅、写入 PostgreSQL 并重建 Neo4j，确保后续事件关联建立在可信实体关系之上。

## What Changes

- **BREAKING（仅 local 关系数据）**：保留 PostgreSQL `entity_nodes` 和各实体 profile，清空 local `entity_edges`，同时清空可重建的 local Neo4j 投影数据。
- 将 repo 中现有 78 条样例关系从默认 seed 基线移除，避免后续运行 `entity-seed` 时自动恢复未审阅关系。
- 建立按关系族分批审阅的实体关系清洗流程；本 change 完成 `member_of`、`has_market` 和 `tracks_index` 三个投研优先批次，其余关系族保留校验能力但按事件推导价值重新排期。
- 为实体关系补充最小来源元数据，至少保存来源名称、来源 URL 和核验时间，使关系事实可以追溯和复核。
- 增加关系方向、允许的实体类型组合、重复关系、悬空实体、缺少来源和禁止推理结论的校验规则。
- 每批关系必须先完成人工 review，再写入 PostgreSQL；PG 验收通过后运行 `graph-projector rebuild-entities`，从 PG 重建 Neo4j 实体图。
- 第二批 `has_market` 在关系写入前补充第一版事件投研所需的最小市场实体，优先覆盖核心主权债券市场、关键商品交易场所和高事件敏感区域市场；新增实体必须先通过独立审阅清单确认。
- 第三批 `tracks_index` 只保留具有明确编制方法的正式指数及其市场归属，纠正道琼斯、MSCI、中债综合指数等错误归属，并补充全球股票分析市场和三个高事件敏感区域股票指数；价格、收益率和参考利率等可观测基准不再作为 `index` 或 `tracks_index` 写入。
- 经投研优先级复核，`issues`、`participates_in`、`affiliated_with` 和 `applies_to` 不在本 change 写入数据：先建设 benchmark，再建设市场、板块、商品和产业链传导层，最后补公司和证券映射。
- 本 change 不实现事件提取、事件图谱、自动队列触发、benchmark 实体与观测值存储，或生产环境关系清空入口；除已审阅通过的市场和正式指数补充外，不修改已经确认的实体主数据内容。
- 已整理的 `issues` 候选清单作为后续输入保留，但不视为 review 通过，不进入 seed、PostgreSQL 或 Neo4j。
- 变更范围仅包含 `tidewise-ai` 后端 seed、migration、校验、测试、OpenSpec 和本地验证说明；不修改 `prototype` 和 `doc`。

## Capabilities

### New Capabilities

- `entity-relationship-curation`: 定义实体关系空基线、分批人工审阅、关系类型约束、来源审计、PG 写入验收和 Neo4j 重建流程。

### Modified Capabilities

- `entity-foundation-seeds`: 将实体主数据 seed 与关系 seed 解耦，禁止未审阅关系随实体主数据自动恢复，并要求关系数据通过分批校验和 review。
- `event-knowledge-schema`: 为 `entity_edges` 增加最小来源元数据，并保证通过增量 migration 保留实体主数据和其他业务数据。
- `neo4j-graph-projection-foundation`: 明确 Neo4j 可以从空投影开始，并只能从已审阅的 PostgreSQL 实体关系事实重建图谱。

## Impact

- 影响 `backend/data/entity_foundation/relationships.json` 及可能新增的按关系族拆分数据文件。
- 影响 `backend/data/entity_foundation/markets.json`，仅增加第二批 review 通过的最小市场实体，不改写其他已确认实体。
- 影响 `backend/data/entity_foundation/indices.json`，仅增加或修正第三批 review 通过的正式指数；10 个价格、收益率或参考利率概念延后到独立 benchmark change。
- 影响 `backend/internal/apps/entityfoundation/seed` 的关系模型、校验、report 和 PostgreSQL upsert。
- 影响 `backend/migrations/` 中 `entity_edges` 的增量字段定义，以及 `internal/repositories` 和 `graphprojection` 的关系读取模型。
- local PostgreSQL 的实体关系数据和 local Neo4j 投影将被清空；PostgreSQL 实体节点、实体 profile、采集数据、事件数据及其他业务表必须保留。
- 不新增外部依赖，不修改前端和公开 API。
