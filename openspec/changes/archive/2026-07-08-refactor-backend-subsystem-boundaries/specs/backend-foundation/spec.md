## MODIFIED Requirements

### Requirement: 后端扩展边界
系统 SHALL 为多个可运行后端子系统、共享基础层、外部平台集成、数据访问和异步/采集任务预留清晰目录边界，避免未来能力直接堆叠在入口、handler 或全局杂项包中。

#### Scenario: 添加可运行子系统
- **WHEN** 后续 change 添加小程序 API、管理后台 API、采集调度器或运维命令
- **THEN** 进程入口必须放在 `backend/cmd/*`，业务逻辑必须放在 `backend/internal/apps/<subsystem>`，不得在 `cmd/*` 中承载复杂业务流程

#### Scenario: 添加共享基础能力
- **WHEN** 多个后端子系统需要共享领域模型、数据库访问、配置、数据库连接、平台运行能力或跨子系统外部平台集成
- **THEN** 该能力必须放入 `domain`、`repositories`、`config`、`platform` 或 `integrations` 等共享基础层，并避免反向依赖具体业务子系统

#### Scenario: 添加外部能力
- **WHEN** 后续 change 添加 Agent 平台、支付、消息、对象存储、第三方认证、数据库、缓存、队列、图谱或 RAG 数据处理
- **THEN** 该能力必须放入对应后端边界，并保持前端无法直接访问凭证或内部实现

### Requirement: 后端正式模块分层
系统 SHALL 在 Go 后端正式模块开发中区分进程入口、业务子系统应用逻辑、协议适配、领域模型、数据访问、外部平台集成、采集子系统和契约边界。

#### Scenario: 添加业务 handler
- **WHEN** 后续 change 添加业务 HTTP handler
- **THEN** handler 必须负责协议适配和 DTO 映射，不得直接承载复杂业务编排、数据库访问或 Agent 平台调用

#### Scenario: 添加业务编排
- **WHEN** 后续 change 添加跨 repository、integration 或 job 的业务流程
- **THEN** 该流程必须放入对应 `internal/apps/<subsystem>` 应用服务边界，并保持领域规则和基础设施细节可测试

#### Scenario: 添加采集流程
- **WHEN** 后续 change 添加自研爬虫采集、外部 Agent API 采集结果接入、清洗、标准化或去重流程
- **THEN** 该流程必须放入 `internal/apps/ingestion` 子系统边界，并通过采集 runtime、connector、parser、repository 和共享 integration 协作完成

### Requirement: 采集层代码边界
系统 SHALL 在 Go 后端中提供 `internal/apps/ingestion` 子系统边界，用于承载采集核心接口、采集编排、source catalog、connector、parser、来源健康、清洗、标准化、去重和写入流程，并通过 `core`、共享 `repositories`、`domain`、`config` 和必要的共享 `integrations` 协作。

#### Scenario: 新增采集模块
- **WHEN** 后续 change 添加或扩展采集层代码
- **THEN** 采集编排必须位于后端 `internal/apps/ingestion` 边界，而不是放入 HTTP handler、main 入口或前端服务中

#### Scenario: 新增采集核心抽象
- **WHEN** 后续 change 添加采集 connector/parser 接口、注册表、凭证解析、限流策略、原始响应、原始文档候选对象或 raw document 写入抽象
- **THEN** 该能力必须位于 `internal/apps/ingestion/core`，不得重新创建顶层 `internal/ingestion`

#### Scenario: 协作外部连接器
- **WHEN** 采集流程需要访问 RSS、HTTP API、RSSHub、网页、本地文件或未来 SDK worker
- **THEN** 访问外部数据源的实现必须通过 `internal/apps/ingestion/connectors` 或等价采集 connector 边界，而不是散落在 repository、handler 或全局 integrations 中
