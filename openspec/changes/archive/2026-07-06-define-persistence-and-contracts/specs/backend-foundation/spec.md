## ADDED Requirements

### Requirement: 后端正式模块分层
系统 SHALL 在 Go 后端正式模块开发中区分 HTTP、应用服务、领域模型、数据访问、外部集成、异步任务和契约边界。

#### Scenario: 添加业务 handler
- **WHEN** 后续 change 添加业务 HTTP handler
- **THEN** handler 必须负责协议适配和 DTO 映射，不得直接承载复杂业务编排、数据库访问或 Agent 平台调用

#### Scenario: 添加业务编排
- **WHEN** 后续 change 添加跨 repository、integration 或 job 的业务流程
- **THEN** 该流程必须放入应用服务边界，并保持领域规则和基础设施细节可测试

#### Scenario: 添加采集流程
- **WHEN** 后续 change 添加自研爬虫采集、外部 Agent API 采集结果接入、清洗、标准化或去重流程
- **THEN** 该流程必须放入 ingestion 边界，并通过 integrations、jobs 和 repositories 协作完成

### Requirement: 后端数据访问边界
系统 SHALL 通过 repository 边界访问 PostgreSQL，并通过明确的缓存或短期状态边界访问 Redis。

#### Scenario: 访问结构化数据
- **WHEN** 后端业务模块需要读取或写入用户、事件、市场、板块、订阅、报告、Agent 结果或任务记录
- **THEN** 业务代码必须通过 repository 边界访问 PostgreSQL，而不是在 handler 中直接执行 SQL

#### Scenario: 访问缓存或幂等状态
- **WHEN** 后端业务模块需要缓存、限流、幂等或短期任务状态
- **THEN** 业务代码必须通过 Redis 访问边界或封装后的服务访问，而不是散落操作 Redis key

### Requirement: 后端迁移边界
系统 SHALL 将数据库 schema 变更纳入 migration 边界，并保证 local、uat、prod 使用一致的迁移来源。

#### Scenario: 新增数据库表
- **WHEN** 后续 change 新增或修改 PostgreSQL schema
- **THEN** 该 change 必须提供 migration，并说明 local、uat、prod 的执行方式

#### Scenario: 回滚 schema 变更
- **WHEN** 某个 schema 变更需要回滚
- **THEN** 该 change 必须提供可审阅的回滚策略或兼容迁移策略

### Requirement: 后端契约来源
系统 SHALL 在后端保留 API 契约来源，用于约束小程序请求响应、Agent 回写 payload、错误结构和分页结构。

#### Scenario: 新增 API 契约
- **WHEN** 后续 change 新增业务 API 或 Agent 回写接口
- **THEN** 契约必须先在后端契约边界中定义，并能被实现和测试引用

#### Scenario: 校验契约一致性
- **WHEN** 后续 change 修改 handler、service 或前端 service 的请求响应结构
- **THEN** 该 change 必须验证实现没有偏离已定义契约
