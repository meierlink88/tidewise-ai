## MODIFIED Requirements

### Requirement: 后端多子系统边界
系统 SHALL 在单一Go module内支持Data Service、Miniapp Service/BFF、Admin Portal Service/BFF以及Data-owned import/projection/maintenance commands等可独立运行的后端子系统；Tidewise MUST NOT继续提供采集scheduler/runtime/source-ingest/ingest-smoke子系统。

#### Scenario: 表达可运行进程
- **WHEN** 后端新增或迁移Data API、小程序BFF、管理后台BFF、event import、projection或maintenance command
- **THEN** 进程入口必须位于对应service-owned command边界，并且入口只负责配置加载、依赖组装和启动流程

#### Scenario: 禁止采集执行进程复活
- **WHEN** 后端新增或修改command、worker或application package
- **THEN** 不得重新引入Tidewise ingestion schedule、connector execution、source worker、手动source-ingest或真实ingest smoke；该执行ownership属于外部`agent-run`

#### Scenario: 表达业务子系统
- **WHEN** 后端新增或修改Data、Miniapp或Admin应用逻辑
- **THEN** 业务逻辑必须位于对应service ownership内，不得堆叠在command或无owner共享层

#### Scenario: 限制共享基础层
- **WHEN** 多个服务需要共享config、logging、observability或HTTP bootstrap
- **THEN** platform只能包含无业务技术能力，Data domain/repository/DTO、connector contract和业务client方法必须归对应owner/consumer

#### Scenario: 区分数据库连接和业务数据访问
- **WHEN** Data Service需要创建PostgreSQL连接、配置连接池或检查连通性
- **THEN** database bootstrap可以由platform提供，但repository、migration ownership和数据库凭据必须由Data Service独占

### Requirement: 后端依赖方向约束
系统 SHALL 通过automated architecture tests约束服务依赖方向：Miniapp/Admin只能依赖各自BFF application与本地Data client port/adapter，Data Service拥有Data domain/application/repository，platform不得包含或反向依赖业务能力；tests还必须阻止已删除采集runtime路径或反向caller复活。

#### Scenario: BFF 不依赖 Data 内部包
- **WHEN** 运行架构边界测试
- **THEN** Miniapp/Admin对Data Service domain/application/repository、数据库或migration内部包的import必须失败

#### Scenario: BFF 不直接调用采集 connector
- **WHEN** 运行架构边界测试
- **THEN** Miniapp/Admin不得直接import Data-owned ingestion connector/parser

#### Scenario: 旧runtime路径不存在
- **WHEN** 运行架构与reference tests
- **THEN** 旧scheduler/runtime/health packages、三个退役commands、ingestion runtime config和任何production caller必须不存在

#### Scenario: command 不承载业务逻辑
- **WHEN** 新增或修改任一service command
- **THEN** 入口代码必须只负责解析配置、组装依赖和启动进程，复杂流程必须下沉到对应service application

#### Scenario: platform 不拥有业务能力
- **WHEN** 运行架构边界测试
- **THEN** platform对任一service application/domain/repository的import以及Event/Research/Entity DTO、connector或业务client方法必须失败

## ADDED Requirements

### Requirement: Data-owned adapter 与外部采集执行边界
Data Service SHALL 暂时拥有source catalog、connectors、parsers和仍有调用方的adapter core contract，但Tidewise SHALL NOT组装或运行这些adapter；外部`agent-run`拥有schedule与execution并只通过scoped Data API读取批准metadata、提交产物。

#### Scenario: 测试保留adapter
- **WHEN** connector/parser/sourcecatalog测试运行
- **THEN** adapter和source metadata contract必须继续通过，不得因runtime删除而删除有效测试

#### Scenario: 在外部repo使用adapter
- **WHEN** 后续change需要让`agent-run`执行现有adapter逻辑
- **THEN** 必须在外部repo复制或适配并定义版本/credential/rate-limit/import验收，不得直接import Tidewise Go `internal`代码

## REMOVED Requirements

### Requirement: 采集器可运行子系统
**Reason**: 新架构把采集scheduling与实际execution交给外部`agent-run`；把scheduler改名或迁入Data Service仍会保留错误ownership。
**Migration**: 先建立Data raw-document/reviewed-event受控import，随后删除Tidewise scheduler/source-ingest/ingest-smoke/runtime/health和其装配；保留source seed、connectors/parsers、import、表与历史migration。
