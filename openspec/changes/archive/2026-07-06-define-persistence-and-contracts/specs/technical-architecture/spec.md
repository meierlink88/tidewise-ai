## ADDED Requirements

### Requirement: MVP 数据基础设施选择
系统 SHALL 将 PostgreSQL 定义为 MVP 阶段结构化主存储，将 Redis 定义为缓存、限流、幂等和短期任务状态基础设施，并将独立图数据库和向量数据库延后到明确需求后的独立 change。

#### Scenario: 规划持久化模块
- **WHEN** 开发者规划需要保存业务事实数据的新 change
- **THEN** 该 change 必须优先基于 PostgreSQL 设计结构化数据边界，并说明是否需要 Redis 短期状态

#### Scenario: 规划图谱或向量能力
- **WHEN** 开发者规划图谱查询、RAG 检索、向量召回或复杂关系推理能力
- **THEN** 该 change 必须说明该能力由 PostgreSQL、外部 Agent 平台或未来独立图/向量基础设施承载

### Requirement: API 契约优先开发
系统 SHALL 在真实前后端集成或 Agent 回写实现前，先定义可审阅的 API 契约作为跨端和跨系统协作边界。

#### Scenario: 开始真实 API 开发
- **WHEN** 后续 change 准备把前端 mock-first service 替换为真实 Go API 调用
- **THEN** 该 change 必须先定义或引用 API 契约，而不是直接让页面或 handler 推断字段结构

#### Scenario: 变更已发布契约
- **WHEN** 后续 change 修改已被前端、后端或 Agent 平台使用的契约
- **THEN** 该 change 必须说明兼容性影响、迁移方式和版本处理

### Requirement: 模块化单体演进边界
系统 SHALL 在 MVP 阶段采用 Go 后端模块化单体，将 HTTP、应用服务、领域模型、数据访问、外部集成和异步任务分层组织，并在容量或团队边界明确后再拆分独立服务。

#### Scenario: 新增服务端模块
- **WHEN** 后续 change 新增事件、报告、订阅、Agent 分析或支付回调等服务端模块
- **THEN** 该模块必须映射到 HTTP、应用服务、领域、repository、integration 或 job 边界，而不是把业务逻辑直接堆叠在入口或 handler 中

#### Scenario: 拆分独立服务
- **WHEN** 后续 change 拟将某个能力从模块化单体拆分为独立服务
- **THEN** 该 change 必须说明部署、契约、数据所有权和回滚影响

### Requirement: 数据采集层架构边界
系统 SHALL 将数据采集层定义为服务端输入层，由 ingestion 编排清洗标准化流程，由 integrations 适配自研爬虫和外部 Agent API，由 jobs 调度采集任务，由 repositories 保存标准化结果。

#### Scenario: 规划采集能力
- **WHEN** 后续 change 规划热点事件、政策事件、市场异动、公告、产业动态或热度信号采集
- **THEN** 该 change 必须说明自研爬虫、外部 Agent API、清洗标准化、入库和后续分析触发之间的边界

#### Scenario: 使用采集数据
- **WHEN** 后续 change 需要把采集结果用于事件流、Agent 分析、报告生成、订阅推送或图谱关系
- **THEN** 该 change 必须使用经过清洗标准化和来源追踪的数据，而不是直接使用原始爬虫结果或外部 Agent 原始响应
