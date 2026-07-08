## Purpose

定义观潮家后端数据采集层的当前系统事实，覆盖采集源目录、连接器、解析器、第一批采集通道、原始文档幂等写入、凭证限流安全和采集职责边界。

## Requirements

### Requirement: 采集源目录驱动采集
系统 SHALL 通过采集源目录管理外部信息源，并使用 `ingest_channel`、`provider_key`、`connector_key`、`parser_key`、授权策略、限流策略和状态字段驱动采集流程。

#### Scenario: 选择采集源
- **WHEN** 采集任务启动
- **THEN** 系统必须从采集源目录读取 active 状态的来源，并根据通道、提供方、连接器和解析器选择执行路径

#### Scenario: 管理采集源元数据
- **WHEN** 新增 RSS、HTTP API、RSSHub 路由、网页或本地文件来源
- **THEN** 系统必须记录来源名称、来源类型、来源 URL、主题提示、默认来源等级、授权方式、凭证引用、限流策略和使用授权

### Requirement: 连接器和解析器注册
系统 SHALL 将外部数据源连接器和内容解析器解耦，并将只服务采集链路的 connector/parser 归属到 `internal/apps/ingestion` 子系统，使不同 provider 的获取逻辑和返回内容标准化逻辑可以通过 `internal/apps/ingestion/core` 独立注册、测试和替换。

#### Scenario: 执行连接器
- **WHEN** 采集源指定 `connector_key`
- **THEN** 系统必须通过采集子系统 `core` 注册边界找到对应连接器，并返回原始响应、原始内容类型和采集元数据

#### Scenario: 执行解析器
- **WHEN** 连接器返回 RSS、JSON、HTML、PDF、CSV 或本地文件内容
- **THEN** 系统必须通过采集子系统 `core` 注册边界把内容转换为统一原始文档候选对象

#### Scenario: 未注册实现
- **WHEN** 采集源引用未注册的连接器或解析器
- **THEN** 系统必须把该采集源标记为失败状态或跳过，并记录明确错误，而不是静默写入不完整原始文档

### Requirement: 第一批采集通道
系统 SHALL 支持第一批采集通道：标准 RSS/Atom、Eastmoney HTTP、RSSHub、网页抓取和本地文件回灌，并为 Tushare 与 AKShare SDK 通道保留独立边界。

#### Scenario: 标准 RSS 采集
- **WHEN** 采集源使用 `rss_feed` 连接器和 `rss_item` 解析器
- **THEN** 系统必须解析 RSS 或 Atom 条目，并标准化标题、链接、摘要、发布时间、来源和正文候选内容

#### Scenario: Eastmoney HTTP 采集
- **WHEN** 采集源使用 `http_eastmoney` 连接器
- **THEN** 系统必须通过统一限流、浏览器 User-Agent、超时和错误处理访问 Eastmoney 公共接口，并将返回 JSON 标准化为原始文档候选对象

#### Scenario: RSSHub 采集
- **WHEN** 采集源使用 `rsshub_feed` 连接器
- **THEN** 系统必须支持 `RSSHUB_BASE_URL`、`route_template`、`code_style`、超时和安全 XML 解析，并把条目标准化为原始文档候选对象

#### Scenario: 网页抓取
- **WHEN** 采集源使用 `web_fetch` 连接器
- **THEN** 系统必须保存原始 HTML、PDF 或网页快照，并提取可读正文用于 `content_text`

#### Scenario: 本地文件回灌
- **WHEN** 采集源使用 `local_file` 连接器
- **THEN** 系统必须读取本地 CSV、JSON 或文本文件，并按配置解析为原始文档候选对象

#### Scenario: SDK 通道边界
- **WHEN** 采集源使用 `sdk_tushare` 或 `sdk_akshare`
- **THEN** 本阶段系统必须识别该通道和配置，但不得要求 Go 主服务直接加载 Python SDK；真实 SDK 执行必须留给后续 worker、sidecar 或内部 HTTP wrapper

### Requirement: 原始文档幂等写入
系统 SHALL 根据采集源、外部源 ID 和内容哈希对原始文档进行幂等写入，避免重复采集造成重复事实基础。

#### Scenario: 写入新文档
- **WHEN** 标准化后的原始文档候选对象在数据库中不存在
- **THEN** 系统必须创建新的原始文档记录，并保存采集状态、内容哈希和原始对象 URI

#### Scenario: 重复采集
- **WHEN** 同一采集源返回相同外部源 ID 或相同内容哈希
- **THEN** 系统必须复用或更新已有原始文档记录，而不是创建无意义重复记录

### Requirement: 真实采集 smoke 入库
系统 SHALL 提供显式运行的真实采集 smoke，使无需凭证的公开来源可以经过 connector、parser、writer 和 repository 写入本地 PostgreSQL 原始文档边界。

#### Scenario: 写入真实采集文档
- **WHEN** 开发者在已完成 migration 的 local PostgreSQL 上运行采集 smoke
- **THEN** 系统必须从公开来源采集少量真实文档，并在 `raw_documents` 中保存标题、来源、外部 ID 或内容哈希、发布时间、采集时间和入库状态

#### Scenario: 输出 smoke 结果
- **WHEN** 采集 smoke 运行完成
- **THEN** 命令必须输出结构化结果，包含成功、失败、重复和当前原始文档数量，便于人工 review

#### Scenario: 外部来源失败
- **WHEN** smoke 来源超时、不可达、限流或返回无法解析内容
- **THEN** 系统必须返回明确失败原因，不得写入伪造文档或把失败标记为成功

### Requirement: 真实 repository 幂等写入
系统 SHALL 通过 PostgreSQL repository 对原始文档执行幂等写入，避免重复 smoke 或重复采集造成重复事实基础。

#### Scenario: 重复外部 ID
- **WHEN** 同一采集源返回相同外部 ID 的文档
- **THEN** repository 必须复用或更新已有原始文档，而不是插入重复记录

#### Scenario: 重复内容哈希
- **WHEN** 同一采集源返回内容哈希相同但外部 ID 不可用的文档
- **THEN** repository 必须复用或更新已有原始文档，而不是插入重复记录

### Requirement: 采集写库 UUID 稳定性
系统 SHALL 在真实写库前为采集源和原始文档生成稳定 UUID，确保 PostgreSQL 主键类型和采集幂等策略一致。

#### Scenario: 重复生成文档 ID
- **WHEN** 同一采集源、外部 ID 和内容哈希多次进入写库流程
- **THEN** 系统必须生成相同的原始文档 UUID

#### Scenario: 接收非 UUID 候选 ID
- **WHEN** connector 或 parser 生成的候选文档 ID 不是合法 UUID
- **THEN** repository 或 ingestion helper 必须把它稳定映射为合法 UUID 后再写入 PostgreSQL

### Requirement: 凭证和限流安全
系统 SHALL 将真实凭证从代码、配置文件和数据库中隔离，并通过统一限流边界控制外部 provider 的访问频率。

#### Scenario: 解析凭证引用
- **WHEN** 采集源需要 API key、bearer token、cookie、自建 RSSHub base URL 或其他授权信息
- **THEN** 系统必须通过 `credential_ref` 从环境变量或部署平台 secret 解析凭证，不得把真实值保存到 PostgreSQL 或提交到 repo

#### Scenario: provider 限流
- **WHEN** 多个采集源访问同一 provider
- **THEN** 系统必须根据 `provider_key` 和 `rate_limit_policy` 执行统一限流，而不是由各连接器散落 sleep

### Requirement: 采集层职责边界
系统 SHALL 只负责获取原始材料、保存原始对象、清洗正文、记录来源、去重和写入原始文档，不得在本阶段生成投资判断或推理结论；采集层实现必须位于 backend 的 `internal/apps/ingestion` 子系统内。

#### Scenario: 处理采集材料
- **WHEN** 采集层成功获取外部内容
- **THEN** 系统必须输出可复核的原始文档和采集状态，而不是直接输出利好利空、评分、传导强度或投资建议

#### Scenario: 失败处理
- **WHEN** 外部来源超时、限流、解析失败或返回空内容
- **THEN** 系统必须记录失败状态和错误原因，并允许后续重试，而不是伪造成功文档

#### Scenario: 保持子系统边界
- **WHEN** 后续 change 新增采集 runtime、scheduler、connector、parser、source catalog 或来源健康能力
- **THEN** 该能力必须进入 `internal/apps/ingestion` 子系统，而不是进入小程序 API、管理后台 API 或全局 integrations 杂项包

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
系统 SHALL 允许不同类型来源按阶段接入 connector，并用明确状态表达已可运行、待凭证或暂不可用；只服务采集链路的 connector 必须归属到采集子系统。

#### Scenario: 内容来源可运行
- **WHEN** 来源使用 `rss_feed`、`rsshub_feed`、`web_fetch` 或 `local_file` 连接器且不需要私有凭证
- **THEN** 系统必须能够通过采集子系统 connector 和 parser 把内容标准化为原始文档候选对象

#### Scenario: HTTP 行情和板块来源
- **WHEN** 来源使用 Eastmoney、Sina、Tencent、Yahoo、Stooq 或类似 HTTP provider
- **THEN** 系统必须通过采集子系统 provider 专属 connector/parser 或通用 HTTP connector/parser 表达采集路径，并保留限流、字段映射和数据频率配置

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
