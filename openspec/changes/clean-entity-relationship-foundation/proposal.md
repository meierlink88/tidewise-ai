## Why

当前实体主数据已具备可用基础，但 `relationships.json` 中仅有 78 条样例关系，成员关系不完整，部分关系缺少明确语义和可审计来源。事件提取流水线开始前，需要先建立空关系基线，再按实体层级逐批审阅、写入 PostgreSQL 并重建 Neo4j，确保后续事件关联建立在可信实体关系之上。

## What Changes

- **BREAKING（仅 local 关系数据）**：保留 PostgreSQL `entity_nodes` 和各实体 profile，清空 local `entity_edges`，同时清空可重建的 local Neo4j 投影数据。
- 将 repo 中现有 78 条样例关系从默认 seed 基线移除，避免后续运行 `entity-seed` 时自动恢复未审阅关系。
- 建立按关系族分批审阅的实体关系清洗流程；第一批为联盟组织与国家/经济体的 `member_of` 关系，其余关系族在前一批验收后依次推进。
- 为实体关系补充最小来源元数据，至少保存来源名称、来源 URL 和核验时间，使关系事实可以追溯和复核。
- 增加关系方向、允许的实体类型组合、重复关系、悬空实体、缺少来源和禁止推理结论的校验规则。
- 每批关系必须先完成人工 review，再写入 PostgreSQL；PG 验收通过后运行 `graph-projector rebuild-entities`，从 PG 重建 Neo4j 实体图。
- 本 change 不实现事件提取、事件图谱、自动队列触发、生产环境关系清空入口，也不修改已经确认的实体主数据内容。
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
- 影响 `backend/internal/apps/entityfoundation/seed` 的关系模型、校验、report 和 PostgreSQL upsert。
- 影响 `backend/migrations/` 中 `entity_edges` 的增量字段定义，以及 `internal/repositories` 和 `graphprojection` 的关系读取模型。
- local PostgreSQL 的实体关系数据和 local Neo4j 投影将被清空；PostgreSQL 实体节点、实体 profile、采集数据、事件数据及其他业务表必须保留。
- 不新增外部依赖，不修改前端和公开 API。
