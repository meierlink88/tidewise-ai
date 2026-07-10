## ADDED Requirements

### Requirement: 图谱投影子系统边界
系统 SHALL 在 backend 内提供独立图谱投影子系统，用于把 PostgreSQL 事实数据投影到 Neo4j，并与采集、实体 seed、管理后台 API 和小程序 API 保持边界隔离。

#### Scenario: 放置图谱投影业务逻辑
- **WHEN** 后端实现 PG 到 Neo4j 的投影编排、运行报告、重建和错误处理
- **THEN** 业务逻辑必须位于 `backend/internal/apps/graphprojection` 或等价图谱投影子系统，而不是放入 `cmd/*`、`internal/apps/ingestion`、`internal/apps/entityfoundation/seed` 或 `internal/apps/adminapi`

#### Scenario: 放置 Neo4j 连接基础设施
- **WHEN** 后端实现 Neo4j driver 创建、连接池、连通性检查和关闭流程
- **THEN** 基础设施能力必须位于 `backend/internal/platform/graphdb` 或等价 platform 包，不得反向依赖业务子系统

#### Scenario: 提供图谱投影命令入口
- **WHEN** 开发者需要显式运行图谱投影、重建或连通性检查
- **THEN** 进程入口必须位于 `backend/cmd/graph-projector` 或等价 `cmd/*` 目录，并且入口只负责配置加载、依赖组装和启动流程

#### Scenario: 禁止采集器直接写 Neo4j
- **WHEN** 采集 connector、parser、runtime 或 scheduler 处理外部原始材料
- **THEN** 采集子系统不得直接写入 Neo4j，必须先通过 PostgreSQL 原始文档和后续事实表形成投影来源
