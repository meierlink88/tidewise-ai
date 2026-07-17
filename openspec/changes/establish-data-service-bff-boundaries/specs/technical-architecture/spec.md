## MODIFIED Requirements

### Requirement: 模块化单体演进边界
系统 SHALL 在当前阶段采用 monorepo、单 Go module 下的服务化单体：Data Service、Miniapp Service/BFF 与 Admin Portal Service/BFF 通过稳定 HTTP contract 分离运行时和数据 ownership，并在版本、团队、依赖或容量边界明确后再拆分 module/repo。

#### Scenario: 新增服务端模块
- **WHEN** 后续 change 新增事件、报告、订阅、Agent 分析或支付回调等服务端能力
- **THEN** 该模块必须先明确属于 Data Domain、Miniapp channel、Admin channel、外部 Agent 或未来独立领域，不得放入无 owner 的共享业务层

#### Scenario: BFF 访问 Data 能力
- **WHEN** Miniapp/Admin 需要 Data Domain 查询或写入
- **THEN** 生产运行时必须通过版本化 HTTP REST + JSON contract 调用 Data Service，Service 内部才能使用 Go interface/方法调用

#### Scenario: 拆分独立 module 或 repo
- **WHEN** 后续 change 拟将某个服务拆成独立 Go module 或 repo
- **THEN** 该 change 必须证明独立版本/发布、团队、依赖、资源或构建冲突条件，并说明 contract、数据 ownership、CI/CD 和回滚影响

## ADDED Requirements

### Requirement: 外部 Agent 到 Data Service 边界
系统 SHALL 将 Agent Server 视为独立外部系统，并 SHALL 让 Agent 只通过 Data Service 受控导入/回写 API 交互，不得直接访问 Tidewise PostgreSQL、Neo4j 或 BFF 内部接口。

#### Scenario: Agent 导入 reviewed-outbox
- **WHEN** Agent Server 提交已审阅的结构化事件 package
- **THEN** Data Service 必须验证 service identity、幂等键、payload、review 状态并在自身事务边界持久化

#### Scenario: Agent 尝试数据库连接
- **WHEN** Agent、Miniapp 或 Admin 配置请求 Data DB 凭据
- **THEN** 配置与 architecture/security checks 必须拒绝该 ownership 越界
