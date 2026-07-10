## Why

当前系统已经在 PostgreSQL 中保存实体、实体关系、事件知识 schema 和图谱投影来源，但还没有真正的图数据库投影层。为了支撑后续多跳关系查询、路径分析、图谱推理和事件图谱可视化，需要先建立 Neo4j 图谱投影基础设施，并把 PostgreSQL 中的实体和基础关系投影为可重建的图结构。

本 change 是对既有“未来图谱数据库边界”的正式落地：PostgreSQL 继续作为权威事实源，Neo4j 只作为从 PostgreSQL 派生、可清理重建的图谱查询库。

## What Changes

- 新增 Neo4j 图谱投影基础能力，支持通过统一配置连接 Neo4j，并在启动或命令执行时检查连通性。
- 新增图谱投影子系统边界，用于从 PostgreSQL 读取 `entity_nodes`、实体 profile 和 `entity_edges`，写入 Neo4j 的实体节点和实体关系。
- 新增 Neo4j 节点和关系模型规范，要求节点使用稳定 `entity_id`、`entity_key`、`entity_type` 和名称属性，关系使用稳定关系类型、来源和状态属性。
- 新增可重复执行的实体图投影命令或 worker 入口，支持幂等 upsert、按实体关系更新时间增量投影、以及本地显式重建。
- 新增投影运行记录和错误报告边界，使开发者能确认投影数量、跳过数量、失败数量和错误原因。
- 明确 PostgreSQL 是事实源，Neo4j 是图谱投影视图；Neo4j 数据丢失、清空或结构调整后，必须能从 PostgreSQL 重新投影恢复。
- 暂不在本 change 中实现 AI event 提取、事件图谱投影、图谱可视化、图谱查询 API、投资建议、关系强度推理或向量数据库。

## Capabilities

### New Capabilities

- `neo4j-graph-projection-foundation`: 定义 Neo4j 图谱投影基础能力，覆盖 Neo4j 配置连接、PG 到 Neo4j 的实体图投影、幂等重建、投影运行记录和事实源边界。

### Modified Capabilities

- `technical-architecture`: 将原先“未来独立图数据库边界”细化为 Neo4j 图谱投影库，并明确 PostgreSQL 仍是权威事实源。
- `persistence-and-contracts`: 增加非 PostgreSQL 图谱存储的引入规则，要求 Neo4j 投影必须可从 PostgreSQL 重建，不得作为事实源。
- `event-knowledge-schema`: 明确 `entity_nodes`、`entity_relationships` 和后续事件关系表是 Neo4j 投影来源。
- `backend-subsystem-boundaries`: 增加图谱投影子系统和 Neo4j 平台连接归属边界，避免投影逻辑散落在采集器、管理后台 API 或实体 seed 中。

## Impact

- 影响 `backend/config`：新增 local、uat、prod Neo4j 非敏感配置模板，例如启用开关、URI、数据库名、连接超时和连接池参数；真实用户名、密码或 token 必须通过环境变量或部署 secret 注入。
- 影响 `backend/internal/config`：新增 Go 强类型 Neo4j 配置读取和校验。
- 影响 `backend/internal/platform`：新增 Neo4j 连接基础设施，例如驱动创建、连通性检查和关闭流程。
- 影响 `backend/internal/apps`：新增 `graphprojection` 或等价业务子系统，负责实体图投影编排、幂等写入、运行报告和错误处理。
- 影响 `backend/internal/repositories`：补充读取 `entity_nodes`、`entity_edges` 和投影来源数据的查询边界；如需要保存投影运行记录，则通过 PostgreSQL migration 增量创建表。
- 影响 `backend/cmd`：新增 `graph-projector` 或等价命令入口，只负责配置加载、依赖组装和启动投影流程。
- 影响 `backend/migrations`：如需要记录投影运行状态，必须通过版本化 SQL migration 增量新增结构，不得清空已有业务数据。
- 影响 `openspec/specs`：归档后将新增 Neo4j 图谱投影主规格，并补充技术架构、持久化、事件知识和后端边界主规格。
- 不影响 `frontend/miniapp/`、`frontend/admin/`、`prototype/` 和 `doc/`；本 change 不实现任何前端展示、设计稿、报告生成或用户可见图谱查询页面。
