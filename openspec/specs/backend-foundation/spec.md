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

### Requirement: 数据库迁移命令入口
系统 SHALL 提供独立数据库迁移命令，使开发者和部署流程可以不启动 API 服务也能检查或应用 migration。

#### Scenario: 执行本地建表
- **WHEN** 开发者在 local 环境运行数据库迁移命令
- **THEN** 命令必须使用 repo 内 migration 来源在 PostgreSQL 中创建事件知识相关表，并输出可审阅的迁移结果

#### Scenario: 检查迁移状态
- **WHEN** 开发者以 check-only 模式运行迁移命令
- **THEN** 命令必须报告已应用版本和 pending migration，而不是修改数据库结构

### Requirement: 后端数据访问边界
Data Service SHALL 通过repository边界访问PostgreSQL；Miniapp/Admin SHALL 通过consumer-owned `DataServiceClient`访问Data API，并通过明确的缓存或短期状态边界访问各自获准状态。

#### Scenario: 访问结构化数据
- **WHEN** 后端业务模块需要读取或写入用户、事件、市场、板块、订阅、报告、Agent 结果或任务记录
- **THEN** Data-owned业务代码必须通过repository访问PostgreSQL，BFF必须调用Data API；任何handler都不得直接执行SQL

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

#### Scenario: 添加迁移文件
- **WHEN** 后续 change 创建或调整事件知识和采集相关表
- **THEN** 必须在 repo 内提供迁移文件或迁移 runner 可识别的来源，并说明执行方式

#### Scenario: 启动检查迁移版本
- **WHEN** Data Service启动并连接PostgreSQL
- **THEN** 它必须只读检查既有ledger和pending migration且不得自动执行；Miniapp/Admin不得连接Data DB，显式apply只能由单独授权的migration command执行

#### Scenario: 避免并发迁移
- **WHEN** 多个后端实例同时启动并发现 pending migration
- **THEN** migration runner 必须通过 PostgreSQL advisory lock、迁移工具锁或等价机制避免并发 DDL

#### Scenario: 测试迁移来源
- **WHEN** 开发者验证后端持久化基础
- **THEN** 必须能够通过自动化测试或可重复命令确认迁移文件可解析且结构符合预期

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

### Requirement: Go 后端 TDD 测试先行
系统 SHALL 在后端功能开发中采用测试先行方式，先设计并实现 Go 单元测试或可自动化测试用例，再编写生产代码。

#### Scenario: 开发新功能点
- **WHEN** change 实现 migration checker、config、domain、repository、ingestion、connector、parser 或 jobs 功能点
- **THEN** 必须先提交对应 Go 测试代码或测试 fixture，再提交满足测试的生产实现

#### Scenario: 使用 Go 测试生态
- **WHEN** 后端测试需要验证纯函数、接口行为、HTTP 交互、解析器或错误处理
- **THEN** 应优先使用 Go 官方 `testing`、table-driven tests、`httptest`、fake 或成熟断言工具，并避免单元测试依赖真实外部网络

#### Scenario: 验证完成标准
- **WHEN** 后端 change 声明实现完成
- **THEN** `backend/` 下 `go test ./...` 必须通过，除非存在明确记录且经人工确认的外部环境阻塞

### Requirement: 后端 UAT 镜像打包
系统 SHALL 为 Go 后端提供 UAT 可部署镜像，使管理后台 API 和数据库 migration 命令可以作为同一提交版本的服务端产物发布。

#### Scenario: 构建后端镜像
- **WHEN** GitHub Actions 构建 backend UAT 镜像
- **THEN** 镜像必须从 `backend/` Go module 构建后端可执行文件
- **AND** 镜像必须包含运行 `admin-api` 所需的产物
- **AND** 镜像必须包含运行数据库 migration 命令所需的产物

#### Scenario: 使用 UAT 配置启动后端
- **WHEN** UAT 环境启动 backend 服务
- **THEN** 服务必须以 `APP_ENV=uat` 加载 UAT 配置
- **AND** 必须通过环境变量或 UAT 未提交配置注入数据库密码和 Admin Token

### Requirement: UAT migration 部署执行
系统 SHALL 在 UAT backend 服务更新前通过版本化后端 migration 命令执行数据库迁移。

#### Scenario: 部署时执行 migration
- **WHEN** UAT 部署 job 更新 backend 服务
- **THEN** 部署 job 必须在启动新服务前运行后端 migration apply 命令
- **AND** migration 必须使用 repo 内版本化 migration 来源

#### Scenario: migration 失败阻止部署
- **WHEN** UAT migration 执行失败
- **THEN** 部署 job 必须失败
- **AND** 不得继续启动或替换 backend 服务

### Requirement: 后端 UAT 健康探针
系统 SHALL 在 UAT 部署流程中验证后端 HTTP 健康检查和就绪检查。

#### Scenario: 验证 UAT backend 存活
- **WHEN** backend UAT 服务启动完成
- **THEN** 部署流程必须调用 `/healthz` 并确认成功响应

#### Scenario: 验证 UAT backend 就绪
- **WHEN** backend UAT 服务启动完成
- **THEN** 部署流程必须调用 `/readyz` 或等价就绪检查并确认成功响应
