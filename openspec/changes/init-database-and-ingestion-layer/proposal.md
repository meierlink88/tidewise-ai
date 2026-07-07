## Why

当前后端已经具备 Go API/BFF 骨架、环境配置和持久化边界规格，但还没有真实 PostgreSQL schema、数据库迁移来源，也没有可运行的数据采集层。新的事件驱动能力需要先把 ER 设计、采集源目录、原始文档落库和采集连接器边界固化下来，才能支撑后续事件抽取、实体链接、Agent 分析和前端真实数据接入。

本 change 基于外部 ER 与采集层设计文档 `/Users/meierlink/Documents/股票趋势验证/theme-reasoning-er-diagram.md`，并参考三个历史系统的数据采集实现路径：

- `/Users/meierlink/Documents/股票趋势验证/Vibe-Research`
- `/Users/meierlink/Documents/股票趋势验证/Vibe-Trading`
- `/Users/meierlink/Documents/股票趋势验证/Stock`

## What Changes

- 新增 PostgreSQL 数据库迁移基础，覆盖实体图谱基础表、采集源目录、原始文档、事件事实、事件证据、事件标签和事件实体关联。
- 新增后端 ingestion 代码骨架，建立 `source_registry`、`connector_registry`、`parser_registry`、`credential_resolver`、`rate_limiter`、`raw_object_store` 和 `raw_document_writer` 等边界。
- 新增第一批采集通道设计与实现范围：`rss_feed`、`http_eastmoney`、`rsshub_feed`、`web_fetch`、`local_file`。
- 为 `sdk_tushare` 和 `sdk_akshare` 建立配置、接口和任务边界，但本 change 不在 Go 主服务中直接嵌入 Python SDK 运行时。
- 将采集结果统一标准化为 `RAW_DOCUMENT`，支持来源追踪、内容哈希、幂等写入、原始对象保存、清洗正文和采集状态。
- 建立采集层的凭证引用、限流策略、授权策略和安全边界，真实 token、cookie、API key、数据库密码和云服务密钥不得进入 repo 或 PostgreSQL。
- 保留事件抽取之后的表结构边界，但本 change 不实现 AI 事件抽取、影响方向判断、评分、传导强度、预测结论、图数据库投影或向量数据库写入。

## Capabilities

### New Capabilities

- `event-knowledge-schema`: 定义 MVP 阶段 PostgreSQL 中的实体、关系、事件、证据、标签和事件实体关联 schema 能力。
- `data-ingestion-layer`: 定义后端采集层如何从采集源目录读取配置、执行连接器、解析外部内容、保存原始对象并幂等写入原始文档。

### Modified Capabilities

- `backend-foundation`: 增加数据库迁移、采集层代码骨架和后端分层实现要求。
- `persistence-and-contracts`: 将数据采集输入边界推进到具体 `SOURCE_CATALOG`、`RAW_DOCUMENT` 和 PostgreSQL schema 落地要求。
- `technical-architecture`: 明确本阶段采集通道、SDK 边界、PostgreSQL 主存储和未来图/向量投影之间的职责划分。

## Impact

- 影响仓库区域：`backend/`、未来新增的 `infra/` 或迁移目录、`openspec/changes/init-database-and-ingestion-layer/`。
- 影响后端工程：需要新增数据库迁移来源、采集层 Go 包、领域模型、repository 边界、integration connector 边界和相关测试。
- 影响配置：需要扩展 local/uat/prod 非敏感配置，表达数据库、Redis、采集限流、对象存储和外部源基础配置；敏感凭证仍通过环境变量或部署平台 secret 注入。
- 影响外部系统：参考 Vibe-Research、Vibe-Trading、Stock 的采集通道设计，但不直接复制这些系统的业务推理、前端代码、模拟数据或真实凭证。
- 不影响前端页面，不修改 `frontend/miniapp/src`。
- 不修改外部参考项目，不修改 `/Users/meierlink/Documents/股票趋势验证/*` 下的源文件。
