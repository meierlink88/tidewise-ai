## MODIFIED Requirements

### Requirement: 采集源目录驱动采集
系统 SHALL 由 Data Service 拥有并维护采集源目录及其 `ingest_channel`、`provider_key`、`connector_key`、`parser_key`、授权策略、限流策略和状态字段；Tidewise SHALL NOT 使用该目录启动采集任务，外部 `agent-run` 对批准目录信息的读取必须通过本change定义的scoped `/internal/data/v1` Data API contract。

#### Scenario: 查询采集源元数据
- **WHEN** Admin BFF 或已授权外部执行系统需要来源名称、通道、provider、adapter key、非敏感配置与状态
- **THEN** 它必须通过 Data Service 拥有的版本化 API 查询，不得直连 `source_catalogs`；外部scope只能获得批准的非敏感字段和`credential_ref`名称，不得获得secret

#### Scenario: 管理采集源元数据
- **WHEN** source seed 新增 RSS、HTTP API、RSSHub route、网页或本地文件来源
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
Data Service SHALL 通过版本化、认证、bounded batch且幂等的 raw-document import API，根据来源、外部源 ID、稳定 UUID 和内容哈希写入原始文档；Miniapp、Admin、Agent/`agent-run` MUST NOT 直接调用 repository 或 SQL。

#### Scenario: 导入新文档
- **WHEN** 已授权service identity提交通过validation且数据库中不存在的原始文档candidate
- **THEN** Data Service必须在自身transaction边界创建记录并返回带request id的结构化receipt/result

#### Scenario: 重复导入
- **WHEN** 同一来源以相同idempotency key/payload或相同外部ID/content hash重试
- **THEN** Data Service必须返回已有结果或复用记录，不得创建无意义重复事实；相同key不同payload必须409且不修改原结果

#### Scenario: 非法批次
- **WHEN** batch超限、payload无有效来源/归因、字段无效或调用方scope不匹配
- **THEN** Data Service必须在业务写入前拒绝整批请求并返回结构化错误

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

### Requirement: 分阶段 connector 接入
系统 SHALL 用明确状态和versioned source metadata表达adapter已实现、待凭证或暂不可用，并保留只服务采集链路的connector/parser源代码与tests；adapter可测试不等于Tidewise提供production execution。

#### Scenario: adapter代码可验证
- **WHEN** 来源使用`rss_feed`、`rsshub_feed`、`web_fetch`、`local_file`、Eastmoney或AI adapter
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

## ADDED Requirements

### Requirement: 外部采集受控交接
系统 SHALL 让仓库外的 `agent-run`/Agent Server承担采集schedule与execution，并 SHALL 让其只经Data Service受控API读取批准的source metadata和导入raw document/reviewed event；外部系统MUST NOT访问Tidewise PostgreSQL、Neo4j或BFF内部接口。

#### Scenario: agent-run导入raw documents
- **WHEN** 已授权`agent-run`提交有界raw-document batch
- **THEN** Data Service必须验证identity、scope、idempotency、来源归因和payload并返回receipt，不得把连接数据库作为客户端前提

#### Scenario: Agent导入reviewed event
- **WHEN** Agent Server提交reviewed-outbox package
- **THEN** Data Service必须使用独立event import transaction/receipt contract；raw import不得被当作绕过review状态的event入口

#### Scenario: 外部项目尚未交付
- **WHEN** Tidewise runtime已退役而`agent-run`采集尚未上线
- **THEN** 系统必须保持Data读取/import能力与历史数据完整，并明确处于采集停止窗口，不得静默恢复旧command或新建替代scheduler

## REMOVED Requirements

### Requirement: 真实采集 smoke 入库
**Reason**: `ingest-smoke`实际联网并写Data DB，属于迁往外部`agent-run`的execution；保留会形成绕过Data import ownership的入口。
**Migration**: 删除command/runtime与本地说明；保留connector/parser fixtures和raw-document repository/import contract tests，不执行真实来源或数据库写入。

### Requirement: 多来源并发采集
**Reason**: worker concurrency、单源失败隔离、provider限流与scheduler filter兼容都是采集runtime语义，Tidewise不再执行。
**Migration**: 删除`IngestionJob`、runtime limiter/filter/report与对应tests；未来由`agent-run`实现并通过Data import receipt验收，Tidewise保留adapter合同而非runner。
