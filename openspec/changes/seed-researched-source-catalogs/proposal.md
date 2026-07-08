## Why

当前采集层已经具备本地 PostgreSQL 写入、采集源目录、连接器、解析器和真实 RSS smoke 闭环，但可复用的数据源仍停留在调研结论和外部样例系统中，没有形成 repo 内可审计、可初始化、可测试、可统一治理的采集源资产。现在需要把前期调研中适合观潮家的内容类、行情类、板块类和 SDK 类来源统一纳入 `source_catalogs` 管理，并让采集任务能够稳定处理多个来源。

## What Changes

- 新增调研来源的版本化 seed 数据，统一覆盖 Vibe-Research、Vibe-Trading 和 Stock 中可纳入本系统的数据源，包括内容/事件类、行情类、板块类、SDK 类和本地回灌类来源。
- 新增采集源 seed 运行入口，使 local/uat/prod 可以通过同一套结构把来源写入 PostgreSQL 的 `source_catalogs`，且不得提交真实凭证。
- 为 `source_catalogs` 增加可扩展的 `source_config` 结构，用于表达 RSSHub route 参数、网页解析策略、分页参数、股票/指数/板块代码列表、市场范围、数据频率、SDK 方法名、字段映射、采集类别等不适合硬塞进单一 URL 的配置。
- 将真实采集能力分阶段落地：第一阶段保证内容/事件类来源可运行；第二阶段接入 Eastmoney 等 HTTP 行情和板块来源；第三阶段为 Tushare、AKShare 等 SDK 来源提供 worker/wrapper 型 connector 边界和统一元信息治理。
- 将当前串行采集 job 升级为可配置并发采集，支持同时处理多个 active source，并保持按 `provider_key` 的限流、错误汇总和单源失败隔离。
- 增加测试先行覆盖：seed 数据解析、重复 seed 幂等、`source_config` 持久化、来源统计查询、分阶段 connector 注册、并发采集 report、provider 限流边界和无真实网络的 connector/job 测试。
- 不在本 change 中建设外部 Agent 回写、Agent 推理、RAG 或报告生成；外部 Agent 来源可以作为采集源元信息登记，但回写结果契约仍由后续 change 定义。

## Capabilities

### New Capabilities

- 无

### Modified Capabilities

- `data-ingestion-layer`: 采集源目录需要支持版本化来源 seed、`source_config` 扩展配置、全类型来源治理、分阶段 connector 接入和多来源并发采集。
- `persistence-and-contracts`: PostgreSQL 持久化边界需要保存采集源扩展配置，并支持通过版本化工程资产初始化、更新、统计和审计来源目录。

## Impact

- 影响 `backend/migrations/`：需要增加增量 SQL migration，为 `source_catalogs` 增加 `source_config` JSONB 字段。
- 影响 `backend/internal/domain`、`backend/internal/repositories`：需要扩展 `SourceCatalog` 模型和 PostgreSQL repository 的 seed/upsert/scan 行为。
- 影响 `backend/internal/ingestion`、`backend/internal/jobs`：需要支持多来源并发执行，并保持 provider 级限流、来源类型过滤和稳定 report。
- 影响 `backend/internal/integrations`：需要确保内容来源、HTTP 行情/板块来源和 SDK worker/wrapper 来源使用清晰 connector/parser 边界，不引入真实外部网络单元测试。
- 影响 `backend/cmd`：需要新增或扩展采集源 seed 命令，用于把 repo 内来源清单写入目标环境数据库。
- 影响 repo 内新增数据资产：需要增加分阶段来源清单文件，作为工程可审计输入，并能统计系统当前接入来源总数、类型、用途、provider 和状态。
- 不修改 `prototype` 和 `doc` 目录，不直接迁移外部样例系统源码，不提交任何 API key、cookie、token 或生产连接信息。
