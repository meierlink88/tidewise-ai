## ADDED Requirements

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
