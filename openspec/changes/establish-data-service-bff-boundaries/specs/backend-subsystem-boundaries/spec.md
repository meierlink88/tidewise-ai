## MODIFIED Requirements

### Requirement: 后端多子系统边界
系统 SHALL 在单一 Go module 内支持 Data Service、Miniapp Service/BFF、Admin Portal Service/BFF 及 Data-owned jobs 等多个可独立运行的后端子系统，并通过 service-owned command 与 application/internal package 表达进程入口和 ownership 边界。

#### Scenario: 表达可运行进程
- **WHEN** 后端新增或迁移 Data API、小程序 BFF、管理后台 BFF、采集调度器或运维命令
- **THEN** 进程入口必须位于对应 service-owned command 边界，并且入口只负责配置加载、依赖组装和启动流程

#### Scenario: 表达业务子系统
- **WHEN** 后端新增或修改 Data、Miniapp 或 Admin 应用逻辑
- **THEN** 业务逻辑必须位于对应 service ownership 内，不得直接堆叠在 command 入口或无 owner 的共享层

#### Scenario: 限制共享基础层
- **WHEN** 多个服务需要共享 config、logging、observability 或 HTTP bootstrap
- **THEN** 共享 platform 只能包含无业务技术能力，Data domain/repository/DTO 和业务 client 方法必须归对应 owner 或 consumer service

#### Scenario: 区分数据库连接和业务数据访问
- **WHEN** Data Service 需要创建 PostgreSQL 连接、配置连接池或检查数据库连通性
- **THEN** database bootstrap 可以由 platform 提供，但业务 repository、migration ownership 与数据库凭据必须由 Data Service 独占

### Requirement: 采集器可运行子系统
系统 SHALL 将 ingestion、event import、source catalog 与 scheduler 定义为 Data Service 拥有的独立可运行 job/command，允许独立进程或容器 entrypoint 部署，但不得在本 change 拆成独立 Go module、repo 或数据库。

#### Scenario: 独立运行采集器
- **WHEN** 部署或本地开发需要启动 ingestion scheduler 或采集命令
- **THEN** 系统必须能够通过 Data Service-owned command 运行采集子系统，而不要求启动 Miniapp 或 Admin BFF

#### Scenario: 使用 Data 基础设施
- **WHEN** 采集器读取配置、访问 PostgreSQL、使用 Data Domain 或执行受控写入
- **THEN** 采集器必须复用 Data Service 内部 application/repository contract，不得通过 BFF 或 platform 业务包访问

#### Scenario: 禁止提前拆分模块
- **WHEN** 普通功能 change 修改采集器
- **THEN** 不得顺手创建独立 `go.mod`、独立 repo、独立数据库 migration 根或独立版本发布边界

### Requirement: 后端依赖方向约束
系统 SHALL 通过自动化 architecture tests 约束服务依赖方向：Miniapp/Admin 只能依赖各自 BFF application 与本地 Data client port/adapter，Data Service 拥有 Data domain/application/repository，platform 不得包含或反向依赖业务能力。

#### Scenario: BFF 不依赖 Data 内部包
- **WHEN** 运行架构边界测试
- **THEN** Miniapp/Admin 对 Data Service domain/application/repository、数据库或 migration 内部包的 import 必须失败

#### Scenario: BFF 不直接调用采集 connector
- **WHEN** 运行架构边界测试
- **THEN** Miniapp/Admin 不得直接 import Data-owned ingestion connector/parser

#### Scenario: command 不承载业务逻辑
- **WHEN** 新增或修改任一 service command
- **THEN** 入口代码必须只负责解析配置、组装依赖和启动进程，复杂业务流程必须下沉到对应 service application

#### Scenario: platform 不拥有业务能力
- **WHEN** 运行架构边界测试
- **THEN** platform 对任一 service application/domain/repository 的 import以及 Event/Research/Entity DTO 或业务 client 方法必须失败
