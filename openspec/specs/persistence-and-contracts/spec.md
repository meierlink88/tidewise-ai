## Purpose

定义观潮家正式模块开发前的持久化、缓存、图谱/向量演进、数据采集、API 契约、Agent 回写和异步任务边界，作为后续事件、报告、订阅、采集和 AI 分析模块的当前系统事实。

## Requirements

### Requirement: MVP 持久化基础设施
系统 SHALL 使用 PostgreSQL 作为 MVP 阶段结构化主存储，并使用 Redis 承载缓存、限流、幂等和短期任务状态。

#### Scenario: 选择结构化主存储
- **WHEN** 后续 change 引入用户、事件、市场指标、板块、资产、订阅、报告、Agent 分析结果或任务记录
- **THEN** 该 change 必须默认以 PostgreSQL 作为结构化事实数据的主存储

#### Scenario: 使用短期状态
- **WHEN** 后续 change 需要缓存、限流、幂等键、短期会话状态或任务进度
- **THEN** 该 change 必须通过 Redis 边界表达短期状态，而不是把短期状态散落在内存或页面中

### Requirement: 图谱和向量能力演进边界
系统 SHALL 在 MVP 阶段不直接引入独立图数据库或向量数据库，并通过 PostgreSQL 结构化关系和外部 Agent 平台结果承载初始图谱/RAG 数据边界。

#### Scenario: 存储初始关系数据
- **WHEN** 后续 change 需要保存事件、实体、板块、资产或报告之间的关系
- **THEN** 该 change 必须优先通过 PostgreSQL 结构化关系表达，并保留未来迁移到图数据库的边界

#### Scenario: 使用 RAG 或向量检索
- **WHEN** 后续 change 需要 RAG 检索、向量召回或 Prompt 编排
- **THEN** 该能力必须优先经由外部 Agent 平台或明确的服务端集成边界承载，而不是在前端或业务 handler 中直接实现

### Requirement: 数据采集输入边界
系统 SHALL 将热点事件和外部信号采集定义为后端 ingestion 能力，并支持自研爬虫脚本采集和外部 Agent API 采集结果接入两种输入路径。

#### Scenario: 自研爬虫采集数据
- **WHEN** 后续 change 通过自研爬虫脚本采集新闻、公告、政策、市场异动、行业事件或热度信号
- **THEN** 采集结果必须进入后端 ingestion 清洗和标准化边界，而不是由前端或业务 handler 直接使用

#### Scenario: 接入 Agent 采集结果
- **WHEN** 后续 change 通过外部 Agent API 获取已经采集或初步分析后的事件数据
- **THEN** 后端必须通过 integration 边界接入该结果，并交由 ingestion 进行结构化校验、清洗、标准化和入库

### Requirement: 采集数据标准化和存储分流
系统 SHALL 对采集后的原始数据执行来源追踪、去重、清洗、标准化和质量标记，并按数据形态进入关系型、向量或图谱存储边界。

#### Scenario: 标准化采集结果
- **WHEN** 采集层收到自研爬虫或外部 Agent API 的原始结果
- **THEN** 系统必须生成标准化事件、来源、时间、标签、实体、置信度和处理状态，而不是直接把原始结果作为系统事实

#### Scenario: 分流不同数据形态
- **WHEN** 标准化后的数据包含结构化事件、语义向量、实体关系或事件传导链
- **THEN** 系统必须根据数据形态进入 PostgreSQL、未来向量数据库或未来图谱数据库边界，并在独立 change 中定义非 PostgreSQL 存储的引入方式

### Requirement: API 契约边界
系统 SHALL 在真实业务 API 实现前定义 API 契约边界，覆盖请求响应 DTO、错误结构、分页、时间、ID、枚举和 Agent 回写 payload。

#### Scenario: 定义业务接口
- **WHEN** 后续 change 新增面向小程序的业务 API
- **THEN** 该 API 必须先定义契约，包含请求、响应、错误、分页和字段语义

#### Scenario: 定义 Agent 回写接口
- **WHEN** 后续 change 新增 Agent 平台结构化结果回写
- **THEN** 该回写必须先定义 payload 契约、幂等字段、状态字段和校验规则

### Requirement: Agent 平台回写治理
系统 SHALL 要求 Agent 平台回写必须经过后端鉴权、幂等、结构化校验、状态流转和落库边界。

#### Scenario: 接收 Agent 回写
- **WHEN** Agent 平台向本系统回写分析结果、报告内容、图谱关系或任务状态
- **THEN** 后端必须校验调用方身份、幂等键、payload 结构和状态流转后再保存结果

#### Scenario: 展示 Agent 输出
- **WHEN** 前端展示 Agent 平台生成的分析、报告或市场解读
- **THEN** 展示内容必须保持决策辅助定位，不得表达为直接投资建议

### Requirement: 异步任务边界
系统 SHALL 将事件采集、Agent 分析、报告生成、图谱更新、通知投递和支付/订阅后处理等长时间流程表达为服务端 job 边界。

#### Scenario: 识别长时间任务
- **WHEN** 某个能力无法在一次普通 HTTP 请求中稳定完成
- **THEN** 该能力必须通过服务端 job、任务状态和可查询进度表达，而不是阻塞小程序请求

#### Scenario: 查询任务状态
- **WHEN** 小程序需要展示报告生成、AI 分析或订阅后处理进度
- **THEN** 小程序必须通过 API 契约查询任务状态，而不是直接读取内部队列、数据库或 Agent 平台状态

### Requirement: 采集原始数据持久化
系统 SHALL 将采集层接收到的原始外部材料标准化并保存到 PostgreSQL 的采集源目录和原始文档边界中。

#### Scenario: 保存采集源目录
- **WHEN** 系统注册外部来源
- **THEN** 必须保存来源通道、provider、connector、parser、来源类型、来源 URL、主题提示、授权策略、限流策略、凭证引用和状态

#### Scenario: 保存原始文档
- **WHEN** 采集连接器返回可解析内容
- **THEN** 必须保存对应原始文档，并保留来源、发布时间、采集时间、内容哈希、原始对象 URI、内容类型和入库状态

#### Scenario: 通过 migration 创建持久化结构
- **WHEN** 采集源目录、原始文档或事件证据相关结构需要创建或调整
- **THEN** 必须通过 repo 内版本化 SQL migration 创建或增量修改，不得只在代码模型中表达数据库结构

### Requirement: 本地持久化 smoke 链路
系统 SHALL 提供本地可重复运行的持久化 smoke 链路，覆盖数据库连接、migration、采集源 seed、真实采集、原始文档入库和幂等复跑验证。

#### Scenario: 完成本地持久化闭环
- **WHEN** 开发者按本地说明配置 PostgreSQL 并运行迁移和采集 smoke
- **THEN** 系统必须能在 local 数据库中看到采集源记录、原始文档记录和迁移版本记录

#### Scenario: 复跑 smoke
- **WHEN** 开发者在已有 smoke 数据的 local 数据库上再次运行采集 smoke
- **THEN** 系统必须保持幂等，不得因为同一来源同一文档重复创建多条事实基础记录

### Requirement: 本地基础设施配置边界
系统 SHALL 将本地 PostgreSQL 运行所需的非敏感配置和示例模板保存在 repo 内，并将真实 secret 留给环境变量或未提交文件。

#### Scenario: 查看本地配置模板
- **WHEN** 开发者查看本地数据库或 smoke 运行模板
- **THEN** 模板必须说明需要的变量名和用途，但不得包含真实密码、真实 token 或生产连接串

#### Scenario: 切换 local、uat、prod
- **WHEN** 后续环境需要执行同一套 migration
- **THEN** 系统必须通过环境配置和 secret 注入切换连接目标，而不是修改 migration 文件或业务代码

### Requirement: 采集结果结构化校验
系统 SHALL 在原始文档进入数据库前执行结构化校验和质量标记，确保后续事件抽取不依赖未校验的外部响应。

#### Scenario: 校验必填字段
- **WHEN** 原始文档候选对象缺少标题、来源、内容哈希或可识别来源信息
- **THEN** 系统必须拒绝成功入库或标记为失败状态，并记录明确错误

#### Scenario: 标记处理状态
- **WHEN** 原始文档完成写入、解析失败、重复跳过或等待后续抽取
- **THEN** 系统必须保存可查询的入库状态，而不是只依赖进程日志

### Requirement: Agent 和采集结果边界
系统 SHALL 保持自研采集、外部 Agent 采集结果和后续 Agent 推理结果的边界清晰，避免原始响应绕过 ingestion 直接成为系统事实。

#### Scenario: 接收外部 Agent 采集结果
- **WHEN** 外部 Agent API 返回已经采集或初步整理的事件材料
- **THEN** 后端必须通过 integration 边界接收，并交由 ingestion 标准化、校验和写入原始文档或后续结构化表

#### Scenario: 展示分析结果
- **WHEN** 后续前端或 API 展示基于采集数据生成的分析内容
- **THEN** 展示内容必须保持决策辅助定位，不得表达为直接投资建议

### Requirement: 版本化实体基础库初始化
系统 SHALL 通过 repo 内版本化实体 seed 初始化和更新实体基础库，而不是依赖手工数据库操作、临时 SQL 或散落脚本。

#### Scenario: 初始化实体基础库
- **WHEN** local、uat 或 prod 环境需要初始化一阶段实体基础库
- **THEN** 运维或开发者必须能够运行统一 seed 命令，将 repo 内实体清单写入 PostgreSQL，并得到实体总数、类型分布、profile 写入数量和关系写入数量的统计报告

#### Scenario: 覆盖所有实体 profile
- **WHEN** 一阶段实体 seed 完成
- **THEN** PostgreSQL 中必须至少完成联盟组织、经济体、政策机构、市场、指数、板块、产业链节点、公司、证券、交易工具、指标、商品和人物这些实体 profile 表的初始化验证

#### Scenario: 幂等更新实体基础库
- **WHEN** 同一套实体 seed 在同一数据库中重复执行
- **THEN** 系统必须按稳定实体 key 幂等 upsert，不得创建重复实体、重复 profile 或重复关系

#### Scenario: 审计实体基础库变更
- **WHEN** 后续新增、禁用或修改基础实体、profile 属性或基础关系
- **THEN** 该变更必须体现在 repo 内实体 seed、测试和必要 migration 中，而不是只修改数据库现状

#### Scenario: 保持决策辅助边界
- **WHEN** 实体 seed 初始化基础实体和关系
- **THEN** seed 数据不得包含投资建议、预测结论、利好利空判断、传导强度或事件评分

### Requirement: 采集源扩展配置持久化
系统 SHALL 在 PostgreSQL 采集源目录中持久化非敏感扩展配置，使来源参数可以随 `source_catalogs` 一起查询、审计和迁移。

#### Scenario: 创建扩展配置字段
- **WHEN** 数据库迁移执行到本 change 的版本
- **THEN** PostgreSQL 必须为 `source_catalogs` 提供 `source_config` JSONB 字段，默认值为空 JSON 对象，并且不得影响既有来源记录读取

#### Scenario: 读取扩展配置
- **WHEN** repository 查询 active source 或 seed 后读取来源记录
- **THEN** 系统必须返回 `source_config` 中的非敏感结构化参数，供后续 connector、parser 或 job 使用

### Requirement: 版本化来源初始化
系统 SHALL 通过 repo 内版本化来源清单初始化和更新统一采集源目录，而不是依赖手工数据库操作或散落脚本。

#### Scenario: 初始化来源目录
- **WHEN** local、uat 或 prod 环境需要初始化第一批采集源
- **THEN** 运维或开发者必须能够运行统一 seed 命令，将 repo 内来源清单写入目标 PostgreSQL，并得到接入来源总数和分类统计

#### Scenario: 审计来源变更
- **WHEN** 后续新增、禁用或修改采集来源
- **THEN** 该变更必须体现在 repo 内来源清单、测试和必要 migration 中，而不是只修改数据库现状

#### Scenario: 保护敏感配置
- **WHEN** 来源初始化涉及需要凭证的 provider
- **THEN** seed 数据必须只写入授权类型和凭证引用，不得写入真实 API key、cookie、token 或私有数据库连接信息

#### Scenario: 管理多类型来源
- **WHEN** 来源清单包含内容、行情、板块或本地回灌来源
- **THEN** PostgreSQL 中的 `source_catalogs` 必须能够保存这些来源的用途、类型、provider、connector、parser、扩展配置、授权策略、限流策略和状态，便于统一查询和治理

### Requirement: 统一 migration 和 seed 数据边界
系统 SHALL 在 MVP 阶段保持 backend 统一 PostgreSQL migration 和 repo 内 seed 数据资产边界，后端子系统不得在普通功能 change 中创建独立 migration 根或独立数据库。

#### Scenario: 使用统一 migration
- **WHEN** 小程序 API、管理后台 API、采集子系统或后端运维命令需要调整 PostgreSQL schema
- **THEN** 该变更必须使用 `backend/migrations` 作为统一 migration 来源，除非已有独立 OpenSpec change 决定拆分数据库

#### Scenario: 区分 migration 文件和执行器
- **WHEN** 后端需要读取、检查或执行 PostgreSQL migration
- **THEN** Go 执行器代码必须位于 `backend/internal/platform/dbmigration`，并且 SQL migration 文件必须继续位于 `backend/migrations`

#### Scenario: 管理 seed 数据资产
- **WHEN** 后端需要维护采集源清单、实体基础库或其他 repo 内长期 seed 数据
- **THEN** 数据资产必须放在 `backend/data/<data-domain>` 并通过对应 seed 命令和 repository 边界写入数据库

#### Scenario: 禁止隐式拆库
- **WHEN** 普通功能 change 修改某个后端子系统
- **THEN** 不得顺手创建子系统私有数据库、私有 migration 目录或私有 schema 管理机制

### Requirement: AI connector 非敏感配置持久化
系统 SHALL 允许 `source_catalogs.source_config` 保存 AI Web Research connector 的非敏感运行参数，并保持真实凭证隔离。

#### Scenario: 保存 AI connector 配置
- **WHEN** AI Web Research source 被 seed 到 `source_catalogs`
- **THEN** PostgreSQL 必须保存该 source 的 `collection_mode`、`search_plan_mode`、固定查询计划、`web_search_plan`、搜索选项、来源偏好、可信域名、LLM planner provider、API base URL、API 协议、模型名、`prompt_ref`、`prompt_version`、`prompt_variables`、时间窗口、结果上限、语言和输出 schema 等非敏感配置

#### Scenario: 引用真实凭证
- **WHEN** AI Web Research source 需要调用真实 Web Search API 或真实模型 API
- **THEN** `source_catalogs` 只能保存 `credential_ref` 或 `source_config.credential_refs` 这类凭证引用名，真实 API key 必须来自环境变量或部署平台 secret

### Requirement: AI 采集元数据追踪
系统 SHALL 在原始文档或其 raw metadata 中保留 AI Web Research 的采集上下文，使后续审计能够追踪材料来源、模型配置和提示词版本。

#### Scenario: 保存采集上下文
- **WHEN** AI Web Research item 写入原始文档边界
- **THEN** 系统必须保留 search_plan_mode、查询计划来源、web_search_plan 摘要、参与召回的 search tool、llm_provider、model、api_protocol、prompt_ref、prompt_version、prompt_purpose、search_options、source_preferences、trusted_domain_match、content_origin、retrieval_method、source_attribution_type、来源说明、provider 搜索元数据和原始返回片段等非敏感元数据

#### Scenario: 排除敏感元数据
- **WHEN** 保存 AI Web Research 原始返回或请求元数据
- **THEN** 系统不得保存真实 API key、Authorization header、cookie、私有 token 或其他敏感凭证
