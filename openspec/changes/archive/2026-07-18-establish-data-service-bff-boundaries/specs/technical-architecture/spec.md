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
系统 SHALL 将 `agent-run`/Agent Server视为独立外部系统：外部系统拥有采集schedule、connector execution、模型/Prompt/RAG编排，且只通过Data Service受控source-metadata/read与raw-document/reviewed-event导入API交互；它不得直接访问Tidewise PostgreSQL、Neo4j、BFF内部接口或Go `internal` packages。

#### Scenario: Agent 导入 reviewed-outbox
- **WHEN** Agent Server 提交已审阅的结构化事件 package
- **THEN** Data Service 必须验证 service identity、幂等键、payload、review 状态并在自身事务边界持久化

#### Scenario: agent-run 导入原始材料
- **WHEN** 外部`agent-run`完成来源访问和标准化但尚未形成reviewed event
- **THEN** 它必须通过bounded raw-document import提交，Data Service验证来源归因、caller-scoped幂等、canonical batch hash与scope，并在raw documents+immutable receipt单transaction内持久化且不得重新执行connector

#### Scenario: agent-run 查询未知导入结果
- **WHEN** 外部`agent-run`因timeout不知道raw import是否commit
- **THEN** 它必须经同一Data API namespace和自身service identity按idempotency key查询status；completed返回stored original result，missing返回unknown，外部系统不得查询数据库或要求Tidewise创建job/run状态

#### Scenario: Tidewise 不运行采集器
- **WHEN** 任一Tidewise service或command启动
- **THEN** 不得启动scheduler、source worker、connector/parser execution、provider retry/rate-limit或真实ingest smoke

#### Scenario: Agent 尝试数据库连接
- **WHEN** Agent、Miniapp 或 Admin 配置请求 Data DB 凭据
- **THEN** 配置与 architecture/security checks 必须拒绝该 ownership 越界
