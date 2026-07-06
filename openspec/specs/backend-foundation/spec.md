## Purpose

定义观潮家 Go 后端 API/BFF 工程骨架、健康检查、环境配置、secret 边界和后续服务端扩展边界，作为后端能力继续演进的当前系统事实。

## Requirements

### Requirement: Go API/BFF 工程骨架
系统 SHALL 在 `backend/` 下提供 Go + Gin API/BFF 工程骨架，让服务端能力可以在统一位置编译、测试和启动。

#### Scenario: 后端工程存在
- **WHEN** 开发者打开源码工程根目录
- **THEN** Go 后端源码必须位于 `backend/`

#### Scenario: 后端可编译测试
- **WHEN** 开发者在 `backend/` 下运行 Go 测试命令
- **THEN** 后端骨架必须可以完成编译和测试

### Requirement: 后端 HTTP 健康检查
系统 SHALL 提供最小 HTTP 健康检查接口，用于本地开发、UAT 和生产运行探针。

#### Scenario: 检查进程存活
- **WHEN** 调用 `GET /healthz`
- **THEN** 后端必须返回成功状态和结构化健康响应

#### Scenario: 检查服务就绪
- **WHEN** 调用 `GET /readyz`
- **THEN** 后端必须返回服务配置和基础依赖就绪状态

### Requirement: 后端环境配置加载
系统 SHALL 通过统一强类型 config 支持 `local`、`uat`、`prod` 三类环境，并在启动阶段校验必填配置。

#### Scenario: 按环境加载配置
- **WHEN** Go 后端以 `APP_ENV=local`、`APP_ENV=uat` 或 `APP_ENV=prod` 启动
- **THEN** 服务必须加载对应环境配置文件并生成统一 config 对象

#### Scenario: 拒绝未知环境
- **WHEN** Go 后端以未知 `APP_ENV` 启动
- **THEN** 服务必须返回配置错误，而不是静默使用错误环境

#### Scenario: 校验必填配置
- **WHEN** 必填非敏感配置缺失或格式错误
- **THEN** 服务必须在启动阶段失败并返回明确错误

### Requirement: 后端 secret 边界
系统 SHALL 保证真实 secret 不进入 repo，并且只能通过环境变量或部署平台 secret 注入。

#### Scenario: 提供配置模板
- **WHEN** 开发者查看 `backend/config` 或环境变量示例
- **THEN** 文件中只能出现非敏感默认值或占位名称，不能出现真实密钥、token、密码或生产连接串

#### Scenario: 使用敏感配置
- **WHEN** 后端需要数据库密码、Agent 平台 API key、JWT secret、支付密钥或云厂商密钥
- **THEN** 代码必须通过统一 config 读取注入后的值，而不是在业务模块中直接读取环境变量或硬编码

### Requirement: 后端扩展边界
系统 SHALL 为 API handler、外部集成、数据访问和异步任务预留清晰目录边界，避免未来能力直接堆叠在入口或页面服务中。

#### Scenario: 添加业务 API
- **WHEN** 后续 change 添加真实业务 API
- **THEN** 该 API 必须接入后端 HTTP 路由边界，而不是绕过 API/BFF 直接暴露内部模块

#### Scenario: 添加外部能力
- **WHEN** 后续 change 添加 Agent 平台、数据库、缓存、队列、图谱或 RAG 数据处理
- **THEN** 该能力必须放入对应后端边界，并保持前端无法直接访问凭证或内部实现

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
