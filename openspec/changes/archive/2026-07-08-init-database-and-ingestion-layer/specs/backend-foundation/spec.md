## ADDED Requirements

### Requirement: 采集层代码边界
系统 SHALL 在 Go 后端中新增 `internal/ingestion` 边界，用于承载采集编排、清洗、标准化、去重和写入流程，并通过 `integrations`、`repositories`、`domain` 和 `jobs` 协作。

#### Scenario: 新增采集模块
- **WHEN** 本 change 添加采集层代码
- **THEN** 采集编排必须位于后端 ingestion 边界，而不是放入 HTTP handler、main 入口或前端服务中

#### Scenario: 协作外部连接器
- **WHEN** 采集流程需要访问 RSS、HTTP API、RSSHub、网页、本地文件或未来 SDK worker
- **THEN** 访问外部来源的实现必须通过 integrations 或 connector 边界，而不是散落在 repository 或 handler 中

### Requirement: 数据库迁移实现边界
系统 SHALL 在后端或基础设施区域提供数据库迁移来源，使 PostgreSQL schema 可以在 local、uat、prod 环境一致创建、审阅和增量更新。

#### Scenario: 添加迁移文件
- **WHEN** 本 change 创建事件知识和采集相关表
- **THEN** 必须在 repo 内提供迁移文件或迁移 runner 可识别的来源，并说明执行方式

#### Scenario: 启动检查迁移版本
- **WHEN** Go 后端启动并连接 PostgreSQL
- **THEN** 后端必须能够检查数据库已执行 migration 版本和 repo 内 pending migration，并按环境配置决定是否自动执行

#### Scenario: 避免并发迁移
- **WHEN** 多个后端实例同时启动并发现 pending migration
- **THEN** migration runner 必须通过 PostgreSQL advisory lock、迁移工具锁或等价机制避免并发 DDL

#### Scenario: 测试迁移来源
- **WHEN** 开发者验证后端持久化基础
- **THEN** 必须能够通过自动化测试或可重复命令确认迁移文件可解析且结构符合预期

### Requirement: 采集配置扩展
系统 SHALL 扩展后端强类型配置，使采集层可以读取非敏感的 provider、限流、对象存储和本地运行配置。

#### Scenario: 加载采集配置
- **WHEN** Go 后端启动并加载 local、uat 或 prod 配置
- **THEN** config 对象必须包含采集层运行所需的非敏感配置，并在启动阶段校验必填字段

#### Scenario: 注入采集凭证
- **WHEN** 采集连接器需要真实 API key、token、cookie、数据库密码或云服务密钥
- **THEN** 这些值必须通过环境变量或部署平台 secret 注入，不得写入环境配置文件

### Requirement: 采集层测试边界
系统 SHALL 为采集层 registry、parser、writer、credential resolver 和 rate limiter 提供可自动化验证的单元测试或集成测试边界。

#### Scenario: 验证标准化
- **WHEN** 测试 RSS、Eastmoney JSON、RSSHub XML、网页内容或本地文件输入
- **THEN** 采集层必须能生成统一原始文档候选对象，并处理空值、时间、语言、哈希和来源字段

#### Scenario: 验证安全边界
- **WHEN** 测试凭证解析和配置加载
- **THEN** 测试不得包含真实密钥，且代码不得绕过统一 config 和 credential resolver 读取敏感值

### Requirement: Go 后端 TDD 测试先行
系统 SHALL 在后端功能开发中采用测试先行方式，先设计并实现 Go 单元测试或可自动化测试用例，再编写生产代码。

#### Scenario: 开发新功能点
- **WHEN** 本 change 实现 migration checker、config、domain、repository、ingestion、connector、parser 或 jobs 功能点
- **THEN** 必须先提交对应 Go 测试代码或测试 fixture，再提交满足测试的生产实现

#### Scenario: 使用 Go 测试生态
- **WHEN** 后端测试需要验证纯函数、接口行为、HTTP 交互、解析器或错误处理
- **THEN** 应优先使用 Go 官方 `testing`、table-driven tests、`httptest`、fake 或成熟断言工具，并避免单元测试依赖真实外部网络

#### Scenario: 验证完成标准
- **WHEN** 本 change 声明实现完成
- **THEN** `backend/` 下 `go test ./...` 必须通过，除非存在明确记录且经人工确认的外部环境阻塞
