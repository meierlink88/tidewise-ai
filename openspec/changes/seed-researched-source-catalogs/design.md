## Context

当前 `enable-local-db-and-live-ingestion-smoke` 已经让采集层具备本地 PostgreSQL 连接、migration、`source_catalogs`、`raw_documents`、真实 RSS smoke 和幂等写入能力。现有 `IngestionJob.Run` 会读取多个 active source，但实现上通过 `for _, source := range sources` 顺序执行，因此当前支持一次任务处理多个来源，不支持多个来源并发抓取。

前期调研的三个系统里，最适合作为第一批进入观潮家的是内容/事件类来源：Vibe-Research 的 108 个策展 RSS 源，以及 Stock 中 12 个去重后的新闻网页来源。Vibe-Trading 的 loader、Stock 的行情/指数/板块代码、Tushare 和 AKShare 更偏行情或 SDK provider，不适合和第一批内容源混在一个 change 中做真实采集闭环。

## Goals / Non-Goals

**Goals:**

- 将第一批内容/事件类来源整理为 repo 内版本化 source seed 文件。
- 通过命令把 source seed 幂等写入 PostgreSQL `source_catalogs`。
- 为 `source_catalogs` 增加 `source_config` JSONB 字段，用来承载来源特有参数。
- 让采集任务支持可配置并发执行多个 active source，并保持单源失败隔离。
- 保持 provider 级限流语义，不让并发执行绕过 `provider_key` 和 `rate_limit_policy`。
- 按 TDD 方式先写 fixture/fake/单元测试，再实现生产代码，并通过 `go test ./...`。

**Non-Goals:**

- 不真实接入 Tushare、AKShare、Futu、Yahoo、Finnhub、FMP、Tiingo、Alpha Vantage、OKX、CCXT 等行情或 SDK provider。
- 不在本 change 中建设外部 Agent API 回写、Agent 推理、RAG 或报告生成。
- 不修改 `prototype` 和 `doc` 目录。
- 不把外部样例系统源码复制进本仓库，只沉淀可审计的来源配置和必要出处说明。
- 不在单元测试中访问真实 RSS、网页、Eastmoney、RSSHub 或生产数据库。

## Decisions

### Decision: 第一批只导入内容/事件类来源

本 change 第一批导入目标为 Vibe-Research 的 RSS 源和 Stock 的新闻网页源。它们与观潮家的事件理解目标直接相关，能够进入 `raw_documents` 并成为后续事件抽取和证据链的事实基础。

备选方案是一次性导入全部约 204 个来源条目。该方案看似完整，但会把 RSS、网页、行情、SDK、板块代码、股票代码和 provider loader 混成一类，导致 connector/parser、凭证、限流和验收标准过大。本 change 先接入内容源，行情和 SDK 通过后续 change 逐步接入。

### Decision: 使用 repo 内 seed 文件作为来源目录资产

新增 `backend/internal/sourcecatalog/testdata` 或 `backend/data/source_catalogs` 类似目录保存版本化来源清单，生产代码通过 seed service 读取并写入 repository。seed 文件必须是结构化格式，包含 `id`、`source_name`、`source_url`、`topic_hint`、`provider_key`、`connector_key`、`parser_key`、`source_config`、`rate_limit_policy`、`usage_policy` 和 `status`。

备选方案是把来源直接写进 SQL migration。这样初始化简单，但来源变更会频繁制造 schema migration，而且不利于单独测试和环境差异控制。来源清单作为数据资产更适合独立加载、幂等更新和后续管理。

### Decision: 为 `source_catalogs` 增加 `source_config`

现有表字段已经能表达通用来源身份，但 RSSHub route、网页解析策略、分页参数、分类标签、行情代码列表等都不适合塞进 `source_url` 或 `topic_hint`。`source_config JSONB NOT NULL DEFAULT '{}'::jsonb` 用于保存来源专属的结构化参数。

备选方案是为每类来源持续新增专用列。该方案查询便利，但在采集源快速扩展阶段会导致表结构频繁变更。MVP 阶段先使用 `source_config` 承载低频读取的来源参数，后续如果某些字段进入高频查询再独立列化。

### Decision: 并发采集采用固定 worker pool

`IngestionJob` 增加可配置并发数，默认保持保守值，按 active sources 投递到 worker pool。每个 source 独立执行 connector、parser、writer，report 聚合 total、succeeded、failed、errors。并发数必须可测试，并允许设置为 `1` 保持串行语义。

备选方案是为每个 source 创建 goroutine。这样实现更短，但在 100+ 来源时容易造成外部站点压力和本地资源波动。固定 worker pool 更容易限制并发和测试错误聚合。

### Decision: provider 限流保持在 job 执行路径中

并发执行时仍必须在每个 source fetch 前调用统一 rate limiter，并以 `provider_key` 作为限流 key。后续可将 rate limiter 从内存实现替换为 Redis 支持跨进程限流，本 change 不引入 Redis 运行依赖。

备选方案是每个 connector 自己 sleep。该方案会散落策略，无法统一测试，也无法在后续跨 provider 观察和调整。

## Risks / Trade-offs

- [Risk] 外部来源 URL 可能失效或临时不可达。→ Mitigation：seed 测试只验证配置结构，采集单元测试使用 `httptest` 和 fixture；真实连通性作为显式 smoke，不作为单元测试。
- [Risk] 并发采集可能触发 provider 限流或反爬。→ Mitigation：默认并发保守，按 `provider_key` 执行限流，单源失败不影响整体任务完成。
- [Risk] `source_config` 过度自由导致配置质量参差。→ Mitigation：seed loader 必须做结构校验，至少校验必填字段、connector/parser 组合、URL/route 形式和 JSON 对象类型。
- [Risk] 第一批来源中存在重复 URL 或同源不同主题。→ Mitigation：seed 文件允许稳定 ID，但测试必须检查重复 ID，重复 URL 需要明确允许或归并。
- [Risk] 直接从外部样例系统复制内容会带来上下文漂移。→ Mitigation：只复制来源清单和必要出处元数据，不复制执行脚本；后续通过本工程 connector/parser 执行。

## Migration Plan

1. 增加 `source_config` 的增量 SQL migration。
2. 扩展 domain model、repository 和测试，使 seed/upsert/scan 能保存并读取 `source_config`。
3. 增加 repo 内 source seed 清单和 loader 测试。
4. 增加 seed command，把清单幂等写入目标 PostgreSQL。
5. 为 `IngestionJob` 增加并发执行测试，再实现 worker pool。
6. 使用 fixture/httptest 验证多来源采集和 report 聚合。
7. 运行 `go test ./...` 和 `openspec validate seed-researched-source-catalogs`。

回滚策略：如果 migration 已执行但功能需要回退，可以保留 `source_config` 空字段，不影响已有查询；seed 写入的来源可以通过 `status=disabled` 停用，不需要删除历史配置。

## Open Questions

- 第一批 Vibe-Research RSS 源是否全部启用为 `active`，还是按主题分批启用？默认建议全部 seed，但初始只对部分来源运行 smoke。
- Stock 中新闻网页源是否全部作为 `web_fetch` 接入，还是优先只接入东方财富和新浪财经？默认建议全部 seed，但通过较低并发和限流控制。
