## ADDED Requirements

### Requirement: 版本化采集源清单
系统 SHALL 使用 repo 内版本化采集源清单维护可接入来源，并通过统一 seed 流程把内容类、HTTP 行情类、板块类和本地回灌类来源写入采集源目录。

#### Scenario: 加载调研来源
- **WHEN** 系统执行采集源 seed
- **THEN** 系统必须从 repo 内结构化清单加载 Vibe-Research、Vibe-Trading 和 Stock 中可纳入观潮家的来源，并映射为 `source_catalogs` 记录

#### Scenario: 达到来源接入数量目标
- **WHEN** 系统完成本 change 的来源 seed
- **THEN** 系统必须接入 Vibe-Research 的 108 条 RSS 配置并报告 106 个唯一 URL、Vibe-Trading 排除 `auto` 和 SDK-only loader 后的非 SDK loader source、以及 Stock 的新闻网页、东方财富股票/指数/板块、本地历史文件等非 SDK 来源条目，并报告 SDK 排除口径

#### Scenario: 区分来源类型
- **WHEN** 来源清单包含 RSS、网页新闻、RSSHub route、Eastmoney HTTP、行情 provider、板块代码或本地文件
- **THEN** 系统必须通过 `ingest_channel`、`provider_key`、`connector_key`、`parser_key`、`source_type`、`source_config`、`usage_policy` 和 `status` 表达来源用途、类型、执行路径和当前启用状态

#### Scenario: 校验来源清单
- **WHEN** seed 流程读取来源清单
- **THEN** 系统必须校验来源 ID、名称、通道、provider、connector、parser、来源类型、授权策略、限流策略、状态、阶段和使用说明，遇到无效配置时拒绝写入并返回明确错误

#### Scenario: 幂等写入来源
- **WHEN** 同一来源清单被重复执行 seed
- **THEN** 系统必须按稳定来源 ID 幂等 upsert `source_catalogs`，不得创建重复来源记录

#### Scenario: 统计接入来源
- **WHEN** 来源 seed 完成或开发者查询来源目录
- **THEN** 系统必须能够按来源系统、provider、通道、来源类型、用途和状态统计当前接入的数据源数量，并输出 Vibe-Research、Vibe-Trading 和 Stock 三组来源的实际计数与统计口径

### Requirement: 采集源扩展配置
系统 SHALL 支持通过 `source_config` 保存来源专属结构化参数，使不同 connector 和 parser 可以在不频繁修改表结构的情况下读取扩展配置。

#### Scenario: 保存扩展配置
- **WHEN** 采集源包含 RSSHub route 参数、网页解析策略、分类标签、分页参数、股票/指数/板块代码列表、市场范围、数据频率、字段映射或 fallback 策略
- **THEN** 系统必须把这些非敏感结构化参数保存到 `source_config`，并在读取 active source 时还原给采集执行路径

#### Scenario: 禁止保存敏感信息
- **WHEN** 采集源需要 API key、cookie、bearer token 或私有 RSSHub 访问凭证
- **THEN** `source_config` 不得保存真实敏感值，必须只保存 `credential_ref` 或非敏感配置

### Requirement: 分阶段 connector 接入
系统 SHALL 允许不同类型来源按阶段接入 connector，并用明确状态表达已可运行、待凭证或暂不可用。

#### Scenario: 内容来源可运行
- **WHEN** 来源使用 `rss_feed`、`rsshub_feed`、`web_fetch` 或 `local_file` 连接器且不需要私有凭证
- **THEN** 系统必须能够通过现有或新增 parser 把内容标准化为原始文档候选对象

#### Scenario: HTTP 行情和板块来源
- **WHEN** 来源使用 Eastmoney、Sina、Tencent、Yahoo、Stooq 或类似 HTTP provider
- **THEN** 系统必须通过 provider 专属 connector/parser 或通用 HTTP connector/parser 表达采集路径，并保留限流、字段映射和数据频率配置

### Requirement: 多来源并发采集
系统 SHALL 支持对多个 active source 进行可配置并发采集，并保持单源失败隔离、provider 限流和可测试的汇总报告。

#### Scenario: 并发执行多个来源
- **WHEN** 采集任务读取到多个 active source 且并发数大于 1
- **THEN** 系统必须并发执行来源采集，并输出包含总来源数、成功数、失败数和错误明细的 report

#### Scenario: 单源失败隔离
- **WHEN** 某个来源连接、解析或写入失败
- **THEN** 系统必须把该来源计入失败并记录错误，同时继续处理其他来源

#### Scenario: 保持 provider 限流
- **WHEN** 多个并发 worker 访问同一 `provider_key`
- **THEN** 系统必须继续通过统一 rate limiter 执行 provider 级限流，不得因为并发执行绕过限流边界

#### Scenario: 保持串行兼容
- **WHEN** 采集任务并发数设置为 1
- **THEN** 系统必须保持与现有串行执行等价的处理语义和 report 结构
