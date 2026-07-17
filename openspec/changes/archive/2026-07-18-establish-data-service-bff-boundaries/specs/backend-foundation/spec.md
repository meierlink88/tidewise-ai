## MODIFIED Requirements

### Requirement: 后端扩展边界
系统 SHALL 为 Data Service、Miniapp BFF、Admin BFF、Data-owned import/projection/maintenance command、共享技术基础层和外部平台集成预留清晰目录边界，避免未来能力直接堆叠在入口、handler 或全局杂项包中；采集scheduler与connector execution归外部`agent-run`。

#### Scenario: 添加可运行子系统
- **WHEN** 后续 change 添加小程序 API、管理后台 API、Data API、event import、graph projection或Data-owned运维命令
- **THEN** 进程入口必须位于对应service-owned command边界，业务逻辑必须下沉到对应service application，不得在command中承载复杂流程

#### Scenario: 添加采集执行能力
- **WHEN** 后续 change 需要scheduler、source worker、connector execution、provider retry/rate-limit或真实ingest smoke
- **THEN** 该能力默认属于外部`agent-run`，Tidewise不得在`backend/cmd/*`、BFF、Data Service或platform恢复采集runtime

#### Scenario: 添加共享基础能力
- **WHEN** 多个后端服务需要共享config、logging、observability、request id或HTTP bootstrap
- **THEN** 只有无业务语义的技术能力可以进入platform；Data domain、repository、DTO与业务client必须留在owner/consumer边界并避免反向依赖

#### Scenario: 添加外部能力
- **WHEN** 后续 change 添加 Agent 平台、数据库、缓存、队列、图谱或 RAG 数据处理
- **THEN** 该能力必须放入对应后端边界，并保持前端无法直接访问凭证或内部实现

### Requirement: 后端正式模块分层
系统 SHALL 在 Go 后端正式模块开发中区分service-owned command、Data/Miniapp/Admin application、协议适配、Data领域与repository、外部平台集成、受控import和契约边界；采集执行不属于Tidewise runtime。

#### Scenario: 添加业务 handler
- **WHEN** 后续 change 添加业务 HTTP handler
- **THEN** handler 必须负责协议适配和 DTO 映射，不得直接承载复杂业务编排、数据库访问或 Agent 平台调用

#### Scenario: 添加业务编排
- **WHEN** 后续 change 添加跨 repository、integration 或 job 的业务流程
- **THEN** 该流程必须放入对应service application边界，并保持领域规则和基础设施细节可测试；BFF不得直接编排Data repository

#### Scenario: 添加采集流程
- **WHEN** 后续 change 添加自研爬虫采集、外部 Agent API 采集结果接入、清洗、标准化或去重流程
- **THEN** 来源访问、schedule和connector execution必须位于外部`agent-run`；Tidewise只可在Data-owned边界保留source metadata、受测adapter contract及raw/reviewed-event import validation与persistence，不得组装采集runtime

### Requirement: 真实 PostgreSQL 连接边界
系统 SHALL 让 Data Service及显式Data migration/maintenance command通过统一config和secret注入访问PostgreSQL；Miniapp/Admin BFF与Agent不得持有Data DB credential或建立连接。

#### Scenario: 构建本地数据库连接
- **WHEN** 开发者以 `APP_ENV=local` 启动Data Service或显式Data migration command并通过进程环境注入数据库凭据
- **THEN** Data-owned bootstrap必须能够构建 PostgreSQL 连接并完成就绪校验，且不得从业务模块散落读取环境变量

#### Scenario: BFF 不连接 Data DB
- **WHEN** Miniapp或Admin BFF启动
- **THEN** 配置不得要求PostgreSQL凭据或migration/repository wiring，Data读取必须通过`DataServiceClient`

#### Scenario: 避免提交数据库 secret
- **WHEN** 开发者查看 repo 内配置文件、本地基础设施模板或示例环境文件
- **THEN** 文件中不得包含真实数据库密码、生产连接串、token 或私有凭证

### Requirement: 后端启动迁移检查
Data Service SHALL 在启动阶段以只读、fail-closed方式检查既有PostgreSQL migration ledger；启动路径不得创建ledger或自动应用migration，Miniapp/Admin BFF不得执行Data migration检查。

#### Scenario: 只读检查 migration readiness
- **WHEN** Data Service启动并检查数据库版本
- **THEN** 必须使用server-enforced read-only ledger查询；ledger缺失、pending或版本异常时返回不可就绪，且不得调用会自动创建ledger的检查路径

#### Scenario: 拒绝未知 schema 启动
- **WHEN** Data Service发现pending migration或未知schema
- **THEN** 必须拒绝提供ready状态；只有另行授权的显式migration command可以应用DDL

### Requirement: 后端数据访问边界
Data Service SHALL 通过repository边界访问PostgreSQL；Miniapp/Admin SHALL 通过consumer-owned `DataServiceClient`访问Data API，并通过明确的缓存或短期状态边界访问各自获准状态。

#### Scenario: 访问结构化数据
- **WHEN** 后端业务模块需要读取或写入用户、事件、市场、板块、订阅、报告、Agent 结果或任务记录
- **THEN** Data-owned业务代码必须通过repository访问PostgreSQL，BFF必须调用Data API；任何handler都不得直接执行SQL

#### Scenario: 访问缓存或幂等状态
- **WHEN** 后端业务模块需要缓存、限流、幂等或短期任务状态
- **THEN** 业务代码必须通过 Redis 访问边界或封装后的服务访问，而不是散落操作 Redis key

### Requirement: 采集层代码边界
系统 SHALL 在 Data-owned边界保留source catalog、connector/parser、`core`中仍有调用方的adapter contract、凭证引用解析和raw/reviewed-event import能力；Tidewise SHALL NOT保留采集编排、来源健康、scheduler、source worker或等价production execution wiring。

#### Scenario: 新增采集模块
- **WHEN** 后续 change 添加或扩展source metadata、adapter contract或Data import能力
- **THEN** 代码必须归Data-owned边界并保持可独立测试，不得把connector execution放入HTTP handler、command、BFF或platform

#### Scenario: 新增采集核心抽象
- **WHEN** 后续 change 添加采集 connector/parser 接口、注册表、凭证解析、限流策略、原始响应、原始文档候选对象或 raw document 写入抽象
- **THEN** 只有仍被保留adapter/import contract使用的抽象可以位于`internal/apps/ingestion/core`；runtime-only limiter/writer/orchestration不得恢复，也不得重新创建顶层`internal/ingestion`

#### Scenario: 协作外部连接器
- **WHEN** 外部`agent-run`需要执行RSS、HTTP API、RSSHub、网页、本地文件或SDK adapter
- **THEN** 它必须在外部repo复制/适配并通过Data API提交结果，不得直接import Tidewise Go `internal` connector或访问Data DB

### Requirement: 数据库迁移实现边界
系统 SHALL 在后端或基础设施区域提供数据库迁移来源，使 PostgreSQL schema 可以在 local、uat、prod 环境一致创建、审阅和增量更新。

#### Scenario: 启动检查迁移版本
- **WHEN** Data Service启动并连接PostgreSQL
- **THEN** 它必须只读检查既有ledger和pending migration且不得自动执行；Miniapp/Admin不得连接Data DB，显式apply只能由单独授权的migration command执行

### Requirement: 采集配置扩展
系统 SHALL 通过source catalog和Data metadata contract保存采集adapter的非敏感provider、限流策略、对象引用和配置；Tidewise service config不得重新加入scheduler/runtime执行参数。

#### Scenario: 加载采集配置
- **WHEN** Data Service或adapter contract test读取source metadata
- **THEN** 必须获得经scope过滤的非敏感配置和`credential_ref`名称，不得因此启动connector或provider limiter

#### Scenario: 注入采集凭证
- **WHEN** production connector execution需要真实 API key、token、cookie或云服务密钥
- **THEN** 凭据必须由外部`agent-run`自身secret边界管理；Tidewise只保存引用名，不得把provider secret写入service config或Data DB

### Requirement: 采集层测试边界
系统 SHALL 为保留的connector/parser/registry、credential resolver、source metadata与raw/reviewed-event import contract提供可自动化验证的测试边界；删除runtime-only测试不得降低这些合同覆盖。

#### Scenario: 验证标准化
- **WHEN** 测试 RSS、Eastmoney JSON、RSSHub XML、网页内容或本地文件输入
- **THEN** 采集层必须能生成统一原始文档候选对象，并处理空值、时间、语言、哈希和来源字段

#### Scenario: 验证安全边界
- **WHEN** 测试凭证解析和配置加载
- **THEN** 测试不得包含真实密钥，且代码不得绕过统一 config 和 credential resolver 读取敏感值
