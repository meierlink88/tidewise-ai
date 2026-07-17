## Purpose

定义观潮家后端数据采集层的当前系统事实，覆盖采集源目录、连接器、解析器、第一批采集通道、原始文档幂等写入、凭证限流安全和采集职责边界。

## Requirements

### Requirement: 采集源目录驱动采集
系统 SHALL 由 Data Service 拥有并维护采集源目录及其 `ingest_channel`、`provider_key`、`connector_key`、`parser_key`、授权策略、限流策略和状态字段；Tidewise SHALL NOT 使用该目录启动采集任务，外部 `agent-run` 对批准目录信息的读取必须通过本change定义的scoped `/internal/data/v1` Data API contract。

#### Scenario: 查询采集源元数据
- **WHEN** Admin BFF 或已授权外部执行系统需要来源名称、通道、provider、adapter key、非敏感配置与状态
- **THEN** 它必须通过 Data Service 拥有的版本化 API 查询，不得直连 `source_catalogs`；外部scope只能获得批准的非敏感字段和`credential_ref`名称，不得获得secret

#### Scenario: 管理采集源元数据
- **WHEN** 新增 RSS、HTTP API、RSSHub 路由、网页或本地文件来源
- **THEN** Data Service必须继续记录来源名称、类型、URL、主题提示、默认等级、授权方式、凭证引用、限流策略和使用授权，且不得自动执行该来源

### Requirement: 连接器和解析器注册
系统 SHALL 在 Data Service 边界保留 `internal/apps/ingestion/connectors`、`parsers` 与 `core` 中仍有调用方的 connector/parser contract、注册表和凭证引用解析，使 adapter 可以独立单测和供未来迁移参考；Tidewise SHALL NOT 提供调用这些 adapter 的 production scheduler/runtime/standalone command。

#### Scenario: 验证连接器
- **WHEN** connector unit/contract test按 `connector_key` 调用受保留实现
- **THEN** adapter必须继续返回可校验的原始响应、内容类型与采集元数据，不需要真实数据库或Tidewise runtime

#### Scenario: 验证解析器
- **WHEN** parser unit/contract test接收 RSS、JSON、HTML、PDF、CSV或本地文件fixture
- **THEN** parser必须继续转换为统一原始文档候选对象并保持既有安全/归因规则

#### Scenario: 生产运行限制
- **WHEN** Tidewise binary、BFF或Data API组装生产依赖
- **THEN** 它不得启动connector/parser执行链、source worker、provider limiter或scheduler；实际执行属于外部 `agent-run`

### Requirement: 第一批采集通道
系统 SHALL 保留标准 RSS/Atom、Eastmoney HTTP、RSSHub、网页抓取、本地文件和 AI Web Research 的受测 adapter代码与非敏感配置资产，但这些代码 SHALL NOT 被解释为 Tidewise 内可运行的 production采集通道；Tushare/AKShare及未来provider execution均属于外部执行边界。

#### Scenario: 保留 adapter contract
- **WHEN** 开发者运行第一批connector/parser targeted tests
- **THEN** RSS、Eastmoney、RSSHub、web fetch、local file与AI adapter必须继续满足各自解析、timeout、错误、安全和归因contract

#### Scenario: 禁止内部执行入口
- **WHEN** 开发者检查Tidewise commands与service wiring
- **THEN** 不得存在`ingestion-scheduler`、`source-ingest`、`ingest-smoke`或等价替代runner来执行这些adapters

#### Scenario: 未来外部迁移
- **WHEN** 后续change把某一adapter迁往`agent-run`
- **THEN** 必须在外部repo复制或适配并以source catalog、credential、rate-limit和Data import contract验收，不得直接import Tidewise Go `internal` package

### Requirement: 原始文档幂等写入
Data Service SHALL通过版本化、认证、bounded batch且幂等的raw-document import API，根据来源、外部源ID、稳定UUID和内容哈希写入原始文档；系统SHALL以认证principal派生的1..200 character caller identity、1..200 character idempotency key及Data Service按`raw-document-import-v1` normalization计算的canonical 64-char lowercase SHA-256识别batch，并以既有`NormalizeUUID("raw_document_import_receipt",caller,key)`算法生成receipt ID，在`raw_document_import_receipts`保存不可变审计/result；Miniapp、Admin、Agent/`agent-run`MUST NOT直接调用repository或SQL。

#### Scenario: 导入新文档
- **WHEN** 已授权service identity提交通过whole-batch validation且不存在completed receipt的原始文档batch
- **THEN** Data Service必须在一个PostgreSQL transaction内写入或复用全部raw documents并插入独立immutable receipt，然后一次commit并返回带request id envelope的结构化result

#### Scenario: 重复导入
- **WHEN** 同一认证caller使用相同idempotency key重试且server canonical payload hash与既有receipt相同
- **THEN** Data Service必须精确返回receipt保存的首次business result，不得因当前source eligibility或raw row状态变化重新拒绝/合成不同结果或创建重复事实；每次transport request id可以不同

#### Scenario: 同 key 的 payload 发生变化
- **WHEN** 同一认证caller使用已有idempotency key提交的canonical payload hash与receipt不同
- **THEN** Data Service必须返回machine-readable 409且不插入、更新、删除或truncate raw document/receipt；不同caller的同名key必须位于独立identity scope

#### Scenario: 查询未知网络结果
- **WHEN** caller未收到import响应并按自身identity与idempotency key查询status
- **THEN** 可见receipt必须返回`completed`及stored original result，缺失receipt必须返回`unknown`/尚未commit且不得创建placeholder、job、run或内存状态

#### Scenario: 并发使用相同 key
- **WHEN** 两个transaction并发提交同一caller与idempotency key
- **THEN** caller-scoped unique constraint和`pg_advisory_xact_lock(hashtextextended(raw-receipt caller/key lock text,0))`必须产生一个winner；取得advisory transaction lock后必须以plain `SELECT`读取receipt且不得使用`FOR UPDATE`、`FOR SHARE`或其他row-locking clause；loser必须重读completed receipt并按same-hash replay或different-hash 409处理，不得部分commit

#### Scenario: 跨 key 的 raw identity 并发
- **WHEN** 不同caller/key的batch并发包含相同source external ID或content hash
- **THEN** Data Service必须按sorted raw-identity lock texts串行化并使用conflict-safe insert/winner re-read；external-ID与hash命中两个不同rows时必须返回409并整批零mutation，不得无序选择一个row

#### Scenario: 不同 candidate 折叠到同一 raw row
- **WHEN** 所有candidate identity resolution完成后resolved raw ID数量小于candidate数量或存在重复ID
- **THEN** Data Service必须返回`RAW_DOCUMENT_BATCH_COLLISION` 409并整批rollback，不得把duplicate IDs或少于candidate数的membership写入receipt

#### Scenario: 非法批次
- **WHEN** batch超限、payload无有效来源/归因、字段无效或调用方scope不匹配
- **THEN** Data Service必须在任何raw document或receipt DML前拒绝整批请求并返回结构化错误，不得提供逐item部分成功

#### Scenario: receipt miss 后才验证可变来源状态
- **WHEN** auth/scope/bounds/canonical hash通过且caller/key没有completed receipt
- **THEN** Data Service必须在DML前验证全部item、current source eligibility/attribution及batch duplicate identities；receipt hit必须先按stored snapshot replay，不能让后续source状态变化破坏原结果重放

#### Scenario: receipt 保存 exact batch result
- **WHEN** 首次raw batch commit
- **THEN** receipt必须保存一维无NULL且按canonical candidate order的非空/无重复raw IDs，以及逐字段匹配columns的`receipt_id`、`payload_hash`、`raw_document_ids`、`items[{raw_document_id,disposition}]`和`imported_at`；disposition只能为`created`或`reused`

#### Scenario: raw receipt 与 event receipt 分离
- **WHEN** raw-document import成功或重放
- **THEN** 它只能使用`raw_document_import_receipts`和raw document repository，不得写入或复用`event_import_receipts`、event、source/tag mapping或review状态

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
Tidewise Data Service SHALL 只保存外部provider的非敏感配置和`credential_ref`，不得持有或执行采集provider凭证/限流runtime；Data import caller使用独立service identity，未来 `agent-run` 的provider secret与rate limiting由其自身边界管理。

#### Scenario: 查看来源配置
- **WHEN** Admin或外部执行系统查询source catalog
- **THEN** Data Service只能返回批准的非敏感配置和凭证引用名，不得返回真实API key、token、cookie或secret

#### Scenario: 调用Data import
- **WHEN** `agent-run`向Data Service导入材料
- **THEN** 请求必须使用Data import scope的service identity，且Data Service不得因此调用任何外部provider

### Requirement: 采集层职责边界
系统 SHALL 将采集scheduling、外部来源访问、connector execution、provider retry/rate-limit与模型查询计划运行归外部 `agent-run`；Tidewise Data Service只拥有source catalog、暂存adapter代码、raw/event import validation、去重、归因和持久化，不得输出投资建议。

#### Scenario: 外部系统提交采集材料
- **WHEN** `agent-run`完成外部获取与标准化并提交raw-document batch
- **THEN** Data Service必须验证可复核来源/归因、幂等与安全contract后持久化，不得重新执行connector

#### Scenario: Tidewise应用依赖采集数据
- **WHEN** Miniapp或Admin需要raw document/event数据
- **THEN** BFF必须调用Data API，不得调用外部`agent-run`、connector或repository

#### Scenario: 后续新增采集执行能力
- **WHEN** 后续需求需要scheduler、worker、connector runtime或来源健康执行
- **THEN** 默认归`agent-run` repo；如确需回到Tidewise，必须新建OpenSpec change并明确推翻本边界，不得顺手添加到BFF或platform

### Requirement: 外部采集受控交接
系统 SHALL 让仓库外的 `agent-run`/Agent Server承担采集schedule与execution，并 SHALL 让其只经Data Service受控API读取批准的source metadata和导入raw document/reviewed event；外部系统MUST NOT访问Tidewise PostgreSQL、Neo4j或BFF内部接口。

#### Scenario: agent-run导入raw documents
- **WHEN** 已授权`agent-run`提交有界raw-document batch
- **THEN** Data Service必须验证identity、scope、caller-scoped idempotency、canonical payload hash、来源归因和whole batch，并返回可status查询/原result重放的durable receipt，不得把连接数据库作为客户端前提

#### Scenario: Agent导入reviewed event
- **WHEN** Agent Server提交reviewed-outbox package
- **THEN** Data Service必须使用独立event import transaction/receipt contract；raw import不得被当作绕过review状态的event入口

#### Scenario: 外部项目尚未交付
- **WHEN** Tidewise runtime已退役而`agent-run`采集尚未上线
- **THEN** 系统必须保持Data读取/import能力与历史数据完整，并明确处于采集停止窗口，不得静默恢复旧command或新建替代scheduler

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
- **THEN** 系统必须把这些非敏感结构化参数保存到 `source_config`，并通过scoped Data metadata API、adapter contract test或未来外部`agent-run`执行适配读取；不得因此在Tidewise恢复采集执行路径

#### Scenario: 禁止保存敏感信息
- **WHEN** 采集源需要 API key、cookie、bearer token 或私有 RSSHub 访问凭证
- **THEN** `source_config` 不得保存真实敏感值，必须只保存 `credential_ref` 或非敏感配置

### Requirement: 分阶段 connector 接入
系统 SHALL 用明确状态和versioned source metadata表达adapter已实现、待凭证或暂不可用，并保留只服务采集链路的connector/parser源代码与tests；adapter可测试不等于Tidewise提供production execution。

#### Scenario: adapter代码可验证
- **WHEN** 来源使用 `rss_feed`、`rsshub_feed`、`web_fetch` 或 `local_file` 连接器且不需要私有凭证
- **THEN** 相应unit/contract tests必须验证fetch/parse/config/security行为，且不得要求Tidewise scheduler/runtime或真实DB

#### Scenario: production source状态
- **WHEN** source catalog记录某adapter为active、inactive或disabled
- **THEN** 该状态只表示source metadata，不得使Tidewise自动或手动启动采集

### Requirement: AI Web Research 采集通道
系统 SHALL 保留`llm_web_research` connector、planner、Web Search adapter、parser、prompt引用和程序化标准化代码作为Data-owned受测adapter资产；Tidewise SHALL NOT 通过runtime或scheduler执行它，未来execution归外部`agent-run`。

#### Scenario: 验证AI adapter
- **WHEN** unit/contract test使用fake search/LLM client和source config调用AI adapter
- **THEN** adapter必须继续校验查询计划、执行多个Web Search tool、隔离provider错误并程序化映射结构化items

#### Scenario: 禁止Tidewise统一调度
- **WHEN** AI source在source catalog中为active
- **THEN** Tidewise不得因此调用模型、Web Search、connector或parser；外部执行产物只能经Data raw-document/event import contract进入

#### Scenario: 未来迁往agent-run
- **WHEN** 后继change在`agent-run`实现AI Web Research执行
- **THEN** 必须迁移/适配prompt、credential、tool、normalization contract并保留来源归因，Tidewise本change不跨repo移动代码

### Requirement: AI 采集源配置校验
系统 SHALL 对 AI Web Research source 的 `source_config` 执行 connector 专属校验，确保采集运行参数完整、非敏感且可审计。

#### Scenario: 校验必填参数
- **WHEN** seed、adapter contract test或未来外部执行适配加载 AI Web Research source
- **THEN** 系统必须校验 `collection_mode`、`search_plan_mode`、`search_queries`、`web_search_plan`、tool provider、tool credential ref、tool options、source_preferences、trusted_domains、`max_results` 和 `output_schema` 的类型与取值；LLM provider、API base URL、模型名、`prompt_ref`、`prompt_version`、`prompt_variables`、planner 凭证引用和查询数量上限只在启用 LLM 查询计划或兼容 normalizer 模式时作为必填项校验

#### Scenario: 保护提示词和模型参数
- **WHEN** 开发者查看 repo 内 AI Web Research source seed
- **THEN** seed 可以包含提示词引用、模型名、base URL、Web Search tool 参数、来源偏好、可信域名和凭证引用名，但不得包含完整长提示词正文、真实 API key、token、cookie 或私有凭证值

#### Scenario: 搜索工具失败隔离
- **WHEN** 任一 Web Search provider 返回鉴权失败、参数错误、限流、超时或空结果
- **THEN** adapter contract必须隔离并返回source级失败/跳过原因，未来外部执行系统负责保证同批其他source继续；Tidewise不得因此组装production batch runtime

### Requirement: AI 搜索结果原始文档标准化
系统 SHALL 将 AI Web Research 返回的结构化 items 标准化为与其他 connector 一致的原始文档候选对象。

#### Scenario: 标准化结构化 item
- **WHEN** AI Web Research parser 接收到已校验的 item
- **THEN** 系统必须生成包含标题、正文或摘要、来源 URL 或来源说明、来源名称、发布时间、采集时间、内容哈希、来源等级、内容来源类型、来源归因类型和 raw metadata 的原始文档候选对象

#### Scenario: 幂等处理重复结果
- **WHEN** AI Web Research 多次返回相同来源 URL、外部 ID 或内容哈希的 item
- **THEN** 系统必须复用现有 raw document 幂等策略，不得创建重复事实基础

#### Scenario: 处理无 URL 但有来源说明的结果
- **WHEN** AI Web Research item 没有来源 URL 但有来源名称、来源说明、引用文本或 provider 来源描述
- **THEN** 系统必须使用内容哈希、标题、发布时间和来源归因信息参与幂等判断，并在 raw metadata 中保留原始来源说明

### Requirement: AI 查询计划与原始文档标准化分离
系统 SHALL 保持 AI Web Research 的查询计划生成和原始文档标准化分离，避免采集层回退到由模型格式化原始文档。

#### Scenario: Go 程序化标准化搜索结果
- **WHEN** AI Web Research source 通过 LLM planner 生成查询计划并完成 Web Search
- **THEN** 系统必须继续由 Go 程序把搜索结果映射为 parser 可校验的结构化 items，而不是要求 LLM 根据搜索结果生成 items

#### Scenario: prompt 文件不承载标准化 schema
- **WHEN** 开发者查看 AI Web Research repo prompt
- **THEN** active 查询计划 prompt 不得成为 raw document schema 的事实来源，raw document 候选对象格式必须继续由 Go parser、测试和 OpenSpec 主规格约束
