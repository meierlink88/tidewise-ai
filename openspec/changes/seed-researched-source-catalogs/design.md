## Context

当前 `enable-local-db-and-live-ingestion-smoke` 已经让采集层具备本地 PostgreSQL 连接、migration、`source_catalogs`、`raw_documents`、真实 RSS smoke 和幂等写入能力。现有 `IngestionJob.Run` 会读取多个 active source，但实现上通过 `for _, source := range sources` 顺序执行，因此当前支持一次任务处理多个来源，不支持多个来源并发抓取。

前期调研的三个系统里，可进入观潮家的来源不是单一粒度：Vibe-Research 主要是策展 RSS 内容源；Vibe-Trading 主要是行情、SDK、RSSHub 和 loader provider；Stock 同时包含新闻网页、Eastmoney 行情、指数、概念板块、Tushare、AKShare 和本地文件输出。这个 change 的核心不应只是“跑通一批 RSS”，而是建立统一来源目录能力，让系统后续能回答“我们一共接入了多少数据源、每个来源的用途、类型、provider、凭证、限流、连接器和状态是什么”。

因此本 change 改为全类型来源纳入 `source_catalogs`，但真实采集按阶段落地：内容/事件类来源优先可运行；Eastmoney HTTP 行情和板块类来源在同一 change 内建立可测试 connector/parser 边界；Tushare、AKShare 等 SDK 来源先完成元信息登记和 worker/wrapper connector 边界，真实 Python SDK worker 可作为后续 change 独立实现。

本 change 的数据源接入完成目标采用以下计数口径：

| 来源系统 | 完成目标 | 口径 |
| --- | ---: | --- |
| Vibe-Research | 108 条配置，106 个唯一 URL | 读取 `/Users/meierlink/Documents/股票趋势验证/Vibe-Research/backend/news_sources.json` 的 `sources` 数组；seed 保留 108 条配置记录，同时在 report 中输出 URL 去重数量 |
| Vibe-Trading | 18 个 loader source | 读取 `/Users/meierlink/Documents/股票趋势验证/Vibe-Trading/agent/backtest/loaders/registry.py` 中 `VALID_SOURCES`，排除 `auto` |
| Stock | 约 78 个来源条目 | 覆盖新闻网页、东方财富股票/指数/板块、AkShare 样例股票/指数、Tushare 动态 provider、本地历史文件等可治理来源；允许 provider 级来源、endpoint 来源和代码列表型来源并存，但必须在 `source_config.kind` 中说明粒度 |

## Goals / Non-Goals

**Goals:**

- 将调研得到的内容类、行情类、板块类、SDK 类和本地回灌类来源整理为 repo 内版本化 source seed 文件。
- 完成 Vibe-Research 108 条、Vibe-Trading 18 条、Stock 约 78 条来源的 seed 接入，并输出可审阅统计。
- 通过命令把 source seed 幂等写入 PostgreSQL `source_catalogs`。
- 为 `source_catalogs` 增加 `source_config` JSONB 字段，用来承载来源特有参数。
- 提供来源目录统计能力，使系统可以按 provider、通道、类型、用途和状态统计当前接入来源。
- 分阶段实现 connector 边界：内容源可真实运行，Eastmoney HTTP 行情/板块源具备可测试实现，SDK 源具备 worker/wrapper 型边界和明确错误。
- 让采集任务支持可配置并发执行多个 active source，并保持单源失败隔离。
- 保持 provider 级限流语义，不让并发执行绕过 `provider_key` 和 `rate_limit_policy`。
- 按 TDD 方式先写 fixture/fake/单元测试，再实现生产代码，并通过 `go test ./...`。

**Non-Goals:**

- 不在 Go API 主进程中直接加载 Python SDK，不把 Tushare、AKShare 等 SDK 运行时嵌入主服务。
- 不在本 change 中建设外部 Agent API 回写、Agent 推理、RAG 或报告生成。
- 不修改 `prototype` 和 `doc` 目录。
- 不把外部样例系统源码复制进本仓库，只沉淀可审计的来源配置和必要出处说明。
- 不在单元测试中访问真实 RSS、网页、Eastmoney、RSSHub 或生产数据库。

## Decisions

### Decision: 全类型来源统一登记，真实采集分阶段实现

本 change 将 Vibe-Research、Vibe-Trading 和 Stock 中可治理的数据源统一整理为 `source_catalogs` seed。来源类型至少包括：

- 内容/事件类：RSS、网页新闻、RSSHub route、本地历史材料回灌。
- 行情类：Eastmoney、Sina、Tencent、Yahoo、Stooq、Finnhub、FMP、Tiingo、Alpha Vantage、OKX、CCXT 等 provider 或 loader 来源。
- 板块类：Eastmoney 概念板块列表、板块 K 线、Stock 中预定义重点板块、AKShare 板块列表。
- SDK 类：Tushare、AKShare、Baostock、Futu、Mootdx 等需要 SDK 或本地服务的来源。

真实执行分三阶段：

1. 内容/事件类来源完成真实采集闭环。
2. Eastmoney HTTP 行情和板块类来源完成 Go connector/parser 和 fixture 测试。
3. SDK 类来源完成元信息登记、凭证引用和 worker/wrapper connector stub，真实 SDK worker 后续独立实现。

备选方案是只导入内容源。该方案短期更快，但会让行情、板块、SDK 来源继续散落在调研结论和脚本里，无法统一统计、治理和审计。另一个备选方案是一次性真实实现所有 provider 采集，这会显著扩大外部依赖和凭证风险，不适合当前 change。

### Decision: 数量验收以 seed report 为准

来源接入完成不只看清单文件存在，还必须由 seed loader 或 seed command 输出结构化 report，至少包含：

- `total_configured_sources`: 已解析来源条目总数。
- `by_origin_system`: 按 Vibe-Research、Vibe-Trading、Stock 分组的数量。
- `unique_urls`: 至少覆盖 Vibe-Research 的 URL 去重计数。
- `by_source_type`: 内容、行情、板块、SDK、本地回灌等分类数量。
- `by_status`: active、inactive、disabled 等状态数量。
- `by_provider`: provider 维度数量。

Vibe-Research 的 108 条配置和 106 个唯一 URL 是硬性验收；Vibe-Trading 的 18 个 loader source 是硬性验收；Stock 的“约 78”需要在 seed report 中给出实际解析数量和来源说明，因为 Stock 同时存在代码列表、样例股票、动态 provider 和历史输出文件，最终条目数由配置粒度决定。

### Decision: 使用 repo 内 seed 文件作为来源目录资产

新增 `backend/internal/sourcecatalog/testdata` 或 `backend/data/source_catalogs` 类似目录保存版本化来源清单，生产代码通过 seed service 读取并写入 repository。seed 文件必须是结构化格式，包含 `id`、`source_name`、`source_url`、`topic_hint`、`provider_key`、`connector_key`、`parser_key`、`source_config`、`rate_limit_policy`、`usage_policy`、`source_level`、`source_type`、`status` 和 `stage`。

备选方案是把来源直接写进 SQL migration。这样初始化简单，但来源变更会频繁制造 schema migration，而且不利于单独测试和环境差异控制。来源清单作为数据资产更适合独立加载、幂等更新和后续管理。

### Decision: 为 `source_catalogs` 增加 `source_config`

现有表字段已经能表达通用来源身份，但 RSSHub route、网页解析策略、分页参数、分类标签、股票/指数/板块代码列表、市场范围、数据频率、SDK 方法名、字段映射、fallback 优先级等都不适合塞进 `source_url` 或 `topic_hint`。`source_config JSONB NOT NULL DEFAULT '{}'::jsonb` 用于保存来源专属的结构化参数。

备选方案是为每类来源持续新增专用列。该方案查询便利，但在采集源快速扩展阶段会导致表结构频繁变更。MVP 阶段先使用 `source_config` 承载低频读取的来源参数，后续如果某些字段进入高频查询再独立列化。

### Decision: SDK 来源采用 worker/wrapper connector 边界

Tushare、AKShare、Baostock、Futu 和 Mootdx 的运行方式差异明显，有的需要 Python SDK，有的需要本地服务，有的需要 token。Go 主服务不直接加载这些 SDK，而是在 `source_catalogs` 里统一记录来源元信息、授权方式、凭证引用和 `source_config`，并通过 `sdk_worker`、`sdk_tushare`、`sdk_akshare` 或具体 wrapper connector 表达执行边界。

备选方案是在 Go 主服务中直接通过 shell 或嵌入方式调用 Python SDK。该方案会提高部署耦合和故障面，不符合模块化单体边界。worker/wrapper 方式让主服务保留统一治理和任务调度，SDK 执行可以后续按能力拆分。

### Decision: 来源目录必须支持统计和审计

新增 seed 和 repository 能力时，除了写入 active sources，还要支持按 `provider_key`、`ingest_channel`、`source_type`、`status` 和 `usage_policy` 统计来源。这样后续可以明确展示系统当前接入了多少来源、哪些可运行、哪些只是元信息登记、哪些需要凭证或 worker。

备选方案是只通过 SQL 手工统计。该方案能临时回答问题，但不利于 API、运维和后续管理页面复用。

### Decision: 并发采集采用固定 worker pool

`IngestionJob` 增加可配置并发数，默认保持保守值，按 active sources 投递到 worker pool。每个 source 独立执行 connector、parser、writer，report 聚合 total、succeeded、failed、errors。并发数必须可测试，并允许设置为 `1` 保持串行语义。

备选方案是为每个 source 创建 goroutine。这样实现更短，但在 100+ 来源时容易造成外部站点压力和本地资源波动。固定 worker pool 更容易限制并发和测试错误聚合。

### Decision: provider 限流保持在 job 执行路径中

并发执行时仍必须在每个 source fetch 前调用统一 rate limiter，并以 `provider_key` 作为限流 key。后续可将 rate limiter 从内存实现替换为 Redis 支持跨进程限流，本 change 不引入 Redis 运行依赖。

备选方案是每个 connector 自己 sleep。该方案会散落策略，无法统一测试，也无法在后续跨 provider 观察和调整。

## Risks / Trade-offs

- [Risk] 外部来源 URL、provider API 或 SDK 可用性可能变化。→ Mitigation：seed 测试只验证配置结构，采集单元测试使用 `httptest`、fixture 和 fake SDK worker；真实连通性作为显式 smoke，不作为单元测试。
- [Risk] 并发采集可能触发 provider 限流或反爬。→ Mitigation：默认并发保守，按 `provider_key` 执行限流，单源失败不影响整体任务完成。
- [Risk] `source_config` 过度自由导致配置质量参差。→ Mitigation：seed loader 必须做结构校验，至少校验必填字段、connector/parser 组合、URL/route 形式和 JSON 对象类型。
- [Risk] Stock 来源目标是“约 78”，实现时可能因去重、代码列表合并或 provider 级登记导致数量有轻微差异。→ Mitigation：seed report 必须输出实际数量、统计口径和差异说明；Vibe-Research 108/106 与 Vibe-Trading 18 保持硬约束。
- [Risk] 行情类和板块类来源粒度不一致，有的是 provider，有的是 endpoint，有的是代码列表。→ Mitigation：用 `source_type`、`source_config.kind` 和 `stage` 明确粒度；provider 级来源和具体 endpoint/代码列表来源都允许登记，但必须能解释用途。
- [Risk] SDK 类来源登记后暂时不能真实执行。→ Mitigation：通过 `status`、`stage`、`auth_required`、`credential_ref` 和明确 connector 错误表达“已治理但待 worker 实现”，不得伪造成采集成功。
- [Risk] 来源中存在重复 URL 或同源不同主题。→ Mitigation：seed 文件允许稳定 ID，但测试必须检查重复 ID，重复 URL 需要明确允许或归并。
- [Risk] 直接从外部样例系统复制内容会带来上下文漂移。→ Mitigation：只复制来源清单和必要出处元数据，不复制执行脚本；后续通过本工程 connector/parser 执行。

## Migration Plan

1. 增加 `source_config` 的增量 SQL migration。
2. 扩展 domain model、repository 和测试，使 seed/upsert/scan 能保存并读取 `source_config`。
3. 增加 repo 内 source seed 清单和 loader 测试，按内容、行情、板块、SDK、本地回灌分组。
4. 增加 seed command，把清单幂等写入目标 PostgreSQL，并输出来源统计。
5. 为内容源、Eastmoney HTTP 行情/板块源和 SDK worker/wrapper stub 增加 connector/parser 测试。
6. 为 `IngestionJob` 增加并发执行测试，再实现 worker pool。
7. 使用 fixture/httptest/fake worker 验证多来源采集和 report 聚合。
8. 运行 `go test ./...` 和 `openspec validate seed-researched-source-catalogs`。

回滚策略：如果 migration 已执行但功能需要回退，可以保留 `source_config` 空字段，不影响已有查询；seed 写入的来源可以通过 `status=disabled` 停用，不需要删除历史配置。

## Open Questions

- 全量来源 seed 后，哪些默认 `active`、哪些默认 `inactive` 或 `disabled`？默认建议内容源和可测试 HTTP 源可 active，SDK 源先 inactive 或 active 但 connector 返回明确 worker 缺失错误。
- 来源统计是否需要在本 change 提供 HTTP API，还是先提供 repository/command report？默认建议先提供命令 report，管理 API 后续独立 change。
