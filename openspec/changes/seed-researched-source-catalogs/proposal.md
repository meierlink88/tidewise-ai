## Why

当前采集层已经具备本地 PostgreSQL 写入、采集源目录、连接器、解析器和真实 RSS smoke 闭环，但可复用的数据源仍停留在调研结论和外部样例系统中，没有形成 repo 内可审计、可初始化、可测试的采集源资产。现在需要把第一批适合观潮家事件采集的来源接入 `source_catalogs`，并让采集任务能够稳定处理多个来源。

## What Changes

- 新增第一批调研来源的版本化 seed 数据，优先接入内容/事件类来源：Vibe-Research 的策展 RSS 源，以及 Stock 中可复用的新闻网页来源。
- 新增采集源 seed 运行入口，使 local/uat/prod 可以通过同一套结构把来源写入 PostgreSQL 的 `source_catalogs`，且不得提交真实凭证。
- 为 `source_catalogs` 增加可扩展的 `source_config` 结构，用于表达 RSSHub route 参数、网页解析策略、分页参数、代码列表、采集类别等不适合硬塞进单一 URL 的配置。
- 将当前串行采集 job 升级为可配置并发采集，支持同时处理多个 active source，并保持按 `provider_key` 的限流、错误汇总和单源失败隔离。
- 增加测试先行覆盖：seed 数据解析、重复 seed 幂等、`source_config` 持久化、并发采集 report、provider 限流边界和无真实网络的 connector/job 测试。
- 不在本 change 中真实接入 Tushare、AKShare、行情 K 线、板块历史行情或外部 Agent 回写；这些来源只作为后续扩展边界。

## Capabilities

### New Capabilities

- 无

### Modified Capabilities

- `data-ingestion-layer`: 采集源目录需要支持版本化来源 seed、`source_config` 扩展配置和多来源并发采集。
- `persistence-and-contracts`: PostgreSQL 持久化边界需要保存采集源扩展配置，并支持通过版本化工程资产初始化来源目录。

## Impact

- 影响 `backend/migrations/`：需要增加增量 SQL migration，为 `source_catalogs` 增加 `source_config` JSONB 字段。
- 影响 `backend/internal/domain`、`backend/internal/repositories`：需要扩展 `SourceCatalog` 模型和 PostgreSQL repository 的 seed/upsert/scan 行为。
- 影响 `backend/internal/ingestion`、`backend/internal/jobs`：需要支持多来源并发执行，并保持 provider 级限流和稳定 report。
- 影响 `backend/internal/integrations`：需要确保第一批来源使用已有 `rss_feed`、`rss_item`、`web_fetch`、`text` 等边界，不引入真实外部网络单元测试。
- 影响 `backend/cmd`：需要新增或扩展采集源 seed 命令，用于把 repo 内来源清单写入目标环境数据库。
- 影响 repo 内新增数据资产：需要增加第一批来源清单文件，作为工程可审计输入。
- 不修改 `prototype` 和 `doc` 目录，不直接迁移外部样例系统源码，不提交任何 API key、cookie、token 或生产连接信息。
