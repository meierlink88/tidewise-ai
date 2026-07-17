## ADDED Requirements

### Requirement: 三服务运行时边界
系统 SHALL 在 monorepo 与单一 Go module 内提供 Data Service、Miniapp Service/BFF、Admin Portal Service/BFF 三个独立运行入口；Miniapp 与 Admin SHALL 为平行 channel/application 入口，且不得相互调用。

#### Scenario: 启动三个服务入口
- **WHEN** 开发者构建或启动后端服务
- **THEN** Data、Miniapp 与 Admin 必须能以独立 binary 和独立健康检查运行

#### Scenario: BFF 间编排
- **WHEN** Miniapp 或 Admin 需要另一 channel 已展示的数据
- **THEN** 它必须调用 Data Service 拥有的 contract，不得调用另一 BFF

### Requirement: Data Domain ownership
Data Service SHALL 独占 Entity、Chain Node、Raw Document、Event、Event Tag、Research Theme、Research Anchor、Index 以及 PostgreSQL、Neo4j/向量投影相关领域、application 与 repository 能力；Miniapp/Admin MUST NOT import 或调用这些 Data 内部包。

#### Scenario: BFF 读取 Data Domain
- **WHEN** Miniapp/Admin 页面需要 Data Domain 数据
- **THEN** BFF 必须通过本地 `DataServiceClient` port 调用 Data Service API，不得直接调用 repository 或 SQL

#### Scenario: 检查禁止依赖
- **WHEN** CI 运行 architecture tests
- **THEN** 任何 BFF 到 Data domain/application/repository 内部包的 import 必须失败

### Requirement: 跨服务 HTTP contract
系统 SHALL 将生产跨 Service 通信统一为 HTTP REST + JSON + OpenAPI，并 SHALL 将 service 内部调用保持为 Go interface/方法调用；本 change MUST NOT 要求 gRPC、Kitex、Kratos、服务注册中心、Service Mesh、Kafka 或分布式事务。

#### Scenario: 生产 BFF 调用 Data Service
- **WHEN** Miniapp/Admin 在生产运行时请求 Data 能力
- **THEN** production adapter 必须经版本化 HTTP API 调用，并使用 Data Service 拥有的 OpenAPI contract

#### Scenario: 单元测试 BFF 编排
- **WHEN** BFF application service 运行单元测试
- **THEN** 测试必须能注入 fake `DataServiceClient`，不得依赖真实网络或数据库

### Requirement: Data API 治理
Data Service SHALL 拥有 API DTO、版本、分页、错误、时间、ID、枚举、超时、幂等、service identity、request id/trace 和向后兼容规则；transport DTO MUST NOT 直接暴露 repository model。

#### Scenario: 读取分页集合
- **WHEN** BFF 请求 Data 集合
- **THEN** API 必须返回稳定 cursor、limit 语义、UTC 时间和带 request id 的结构化错误边界

#### Scenario: 重试写操作
- **WHEN** client 因网络故障无法确认写操作结果
- **THEN** 只有携带幂等 identity 的操作可以有限重试，并能通过 receipt/status 区分成功、冲突和未知结果

#### Scenario: 变更已消费 contract
- **WHEN** Data Service 需要删除字段、改变字段语义或引入 breaking change
- **THEN** change 必须提供新版本或 additive compatibility window，并等待独立 Review

### Requirement: 页面级聚合与性能边界
Data Service SHALL 提供当前 Miniapp/Admin 页面和管理操作需要的聚合 API，避免 BFF 为单次页面渲染执行 N+1 或不受控 chatty calls。

#### Scenario: 请求页面聚合
- **WHEN** 页面需要主题、关联事件、产业链节点与指数摘要
- **THEN** BFF 必须能通过有界数量的 Data API calls 获得聚合结果，而不是逐条查询关联对象

#### Scenario: Data API 性能回归
- **WHEN** contract 或实现增加下游调用次数或响应延迟
- **THEN** 受影响 package 必须以调用次数和 timeout/性能基线识别回归

### Requirement: Platform 无业务边界
`backend/platform` SHALL 只包含 config、logging、observability、通用 HTTP client/server bootstrap、request id/trace 等无业务技术能力；Event/Research/Entity DTO、repository、领域规则和业务 client 方法 MUST NOT 位于 platform。

#### Scenario: 提议共享代码
- **WHEN** 两个服务出现重复代码
- **THEN** 只有不含业务语义且已稳定复用的 bootstrap 才能进入 platform，业务 DTO/port 必须留在 owner/consumer service

#### Scenario: 检查 platform import
- **WHEN** CI 运行 architecture tests
- **THEN** platform 对 service application/domain/repository 的反向依赖必须失败

### Requirement: Service-owned deploy assets
每个服务自己的 Dockerfile、binary command、健康检查和启动配置 SHALL 跟随服务目录；根 `infra` SHALL 只拥有跨服务环境编排、网络、数据库和观测配置。

#### Scenario: 构建单一服务
- **WHEN** CI 构建 Data、Miniapp 或 Admin 镜像
- **THEN** 构建必须使用该服务目录拥有的 build/start/health assets，不依赖另一服务的 binary

#### Scenario: 编排 local 环境
- **WHEN** 开发者启动 local 多服务环境
- **THEN** 根 `infra/local` 必须只编排服务、数据库、网络与观测，不承载业务代码或服务内部启动实现

### Requirement: 渐进迁移与 module split gate
系统 SHALL 以 architecture tests、兼容入口与逐消费者切换渐进建立服务边界，MUST NOT 为目录美观一次性搬迁全部 domain/repository/migration；本 change SHALL 保持单一 `backend/go.mod`。

#### Scenario: 迁移现有入口
- **WHEN** `cmd/api` 或 `cmd/admin-api` 迁往目标服务目录
- **THEN** 现有外部 API 行为必须由 contract tests 保持，并允许短期薄 compatibility entrypoint

#### Scenario: 评估 multi-module
- **WHEN** 尚未出现独立版本/发布、独立团队、显著依赖差异、稳定远程 contract 或单 module 构建冲突
- **THEN** change 不得新增 `go.mod`；条件真实出现时必须另开 OpenSpec change
