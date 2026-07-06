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
