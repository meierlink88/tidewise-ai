## Why

当前工程已经具备事件知识 schema migration 文件、迁移 runner 抽象、采集层接口和 fake repository，但还不能把表真正创建到 PostgreSQL，也不能证明真实采集结果可以经过采集层写入数据库。正式模块开发前需要补齐这条本地可重复验证链路，避免后续 API、事件抽取和报告能力建立在未验证的持久化基础上。

## What Changes

- 增加真实 PostgreSQL 连接、迁移执行和迁移版本检查能力，使 `backend/migrations` 中的 DDL 能在 local 环境创建数据库表，并为 uat/prod 保留同一套执行边界。
- 将现有 migration runner 从 fake store 扩展到真实数据库执行路径，确保增量迁移、版本记录和并发迁移保护可验证。
- 增加最小 PostgreSQL repository 实现，覆盖 `source_catalogs` 和 `raw_documents` 的源目录读取、seed/upsert 和原始文档幂等写入。
- 增加真实采集 smoke 命令或等价运行入口，从一个无需凭证的公开来源采集少量真实文档，经过 connector、parser、writer 写入本地 PostgreSQL。
- 增加本地运行配置和说明，使开发者可以在不提交真实密码、token 或生产连接串的前提下完成建表和 smoke 入库。
- 增加测试先行验证：普通 Go 单元测试使用 fake、fixture 或 `httptest`，真实 PostgreSQL 和真实网络 smoke 通过显式环境变量或命令开启，不进入默认单元测试路径。

## Capabilities

### New Capabilities

- 无。

### Modified Capabilities

- `backend-foundation`: 增加真实 PostgreSQL 连接、迁移执行入口、启动迁移检查和本地 smoke 运行边界。
- `event-knowledge-schema`: 明确事件知识 schema 必须能通过 repo 内 migration 在真实 PostgreSQL 中创建，并可查询迁移版本和关键表存在性。
- `data-ingestion-layer`: 增加真实来源采集 smoke，要求采集结果可以经过现有采集层写入本地 PostgreSQL 的原始文档边界。
- `persistence-and-contracts`: 增加本地持久化验证链路，覆盖采集源目录、原始文档、幂等写入和真实数据库 smoke。
- `technical-architecture`: 补充本地数据库和真实采集 smoke 在后端模块化单体、基础设施和部署边界中的位置。

## Impact

- 影响源码区域：`backend/`、`infra/`、`openspec/changes/enable-local-db-and-live-ingestion-smoke/`。
- 可能新增 Go 依赖：PostgreSQL driver、migration 执行库或等价轻量实现，具体在 `design.md` 中确认。
- 可能新增命令入口：数据库迁移命令、采集 smoke 命令或 API 启动迁移检查入口。
- 可能新增本地基础设施模板：用于 local PostgreSQL 的非敏感配置、`.env.example` 或 Docker Compose 示例；真实密码和生产连接串不得提交。
- 不修改 `prototype/`，不把原型 HTML、DOM 行为或设计稿内容引入生产代码。
- 不修改 `../doc/`，除非后续单独 change 明确需要同步长期文档。
- 不实现事件抽取、Agent 推理、RAG、图数据库、向量数据库、投资判断或前端页面能力。
