## Purpose

定义 Go 后端三服务、多子系统、共享基础层、实体基础库、Data-owned adapter 与外部采集执行边界和依赖方向约束，作为后续后端 change 判断代码归属的当前系统事实。

## Requirements

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

### Requirement: 实体基础库子系统边界
系统 SHALL 将实体基础库初始化定义为 backend 内的独立业务子系统能力，不得归入数据采集子系统。

#### Scenario: 初始化实体基础库
- **WHEN** 后端需要初始化经济体、联盟组织、政策机构、市场、指数、板块、产业链节点、公司、证券、指标、商品、KOL 或实体关系
- **THEN** 实体 seed 逻辑必须位于 `backend/internal/apps/entityfoundation/seed`，并通过 `backend/cmd/entity-seed` 运行

#### Scenario: 采集子系统引用实体
- **WHEN** 采集子系统需要把原始数据关联到实体基础库
- **THEN** 采集子系统可以读取或引用实体结果，但不得拥有实体 seed 逻辑或把实体初始化代码放入 `internal/apps/ingestion`

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

### Requirement: Data-owned adapter 与外部采集执行边界
Data Service SHALL 暂时拥有source catalog、connectors、parsers和仍有调用方的adapter core contract，但Tidewise SHALL NOT组装或运行这些adapter；外部`agent-run`拥有schedule与execution并只通过scoped Data API读取批准metadata、提交产物。

#### Scenario: 测试保留adapter
- **WHEN** connector/parser/sourcecatalog测试运行
- **THEN** adapter和source metadata contract必须继续通过，不得因runtime删除而删除有效测试

#### Scenario: 在外部repo使用adapter
- **WHEN** 后续change需要让`agent-run`执行现有adapter逻辑
- **THEN** 必须在外部repo复制或适配并定义版本/credential/rate-limit/import验收，不得直接import Tidewise Go `internal`代码

### Requirement: 图谱投影子系统边界
系统 SHALL 在 backend 内提供独立图谱投影子系统，用于把 PostgreSQL 事实数据投影到 Neo4j，并与采集、实体 seed、管理后台 API 和小程序 API 保持边界隔离。

#### Scenario: 放置图谱投影业务逻辑
- **WHEN** 后端实现 PG 到 Neo4j 的投影编排、运行报告、重建和错误处理
- **THEN** 业务逻辑必须位于 `backend/internal/apps/graphprojection` 或等价图谱投影子系统，而不是放入 `cmd/*`、`internal/apps/ingestion`、`internal/apps/entityfoundation/seed` 或 `internal/apps/adminapi`

#### Scenario: 放置 Neo4j 连接基础设施
- **WHEN** 后端实现 Neo4j driver 创建、连接池、连通性检查和关闭流程
- **THEN** 基础设施能力必须位于 `backend/internal/platform/graphdb` 或等价 platform 包，不得反向依赖业务子系统

#### Scenario: 提供图谱投影命令入口
- **WHEN** 开发者需要显式运行图谱投影、重建或连通性检查
- **THEN** 进程入口必须位于 `backend/cmd/graph-projector` 或等价 `cmd/*` 目录，并且入口只负责配置加载、依赖组装和启动流程

#### Scenario: 禁止采集器直接写 Neo4j
- **WHEN** 采集 connector、parser、runtime 或 scheduler 处理外部原始材料
- **THEN** 采集子系统不得直接写入 Neo4j，必须先通过 PostgreSQL 原始文档和后续事实表形成投影来源

### Requirement: 共享 repository adapter 按业务职责组织
系统 SHALL 在 `backend/internal/repositories` 单一 package 中保留调用方所需的业务小接口，并以共享 `PostgresRepository` 和共享 `InMemoryRepository` 实现这些接口；源码文件 SHALL 按业务职责组织，不得为机械拆分引入 ORM、codegen、repository framework、新 package 或一组业务专用具体 adapter。

#### Scenario: 定位业务 repository 能力
- **WHEN** 开发者需要查看 source catalog、raw document、event/import receipt、benchmark observation、admin query、research read、graph projection 或 identity 的 repository 契约与实现
- **THEN** 对应业务文件必须集中该职责的接口/DTO、参数转换、具体 adapter 方法、Scan/结果映射和专属 helper
- **AND** `PostgresRepository` 与 `InMemoryRepository` 的 constructor 和共享 state 必须保持唯一

#### Scenario: 自动验证行为保持
- **WHEN** 开发者运行 repository targeted tests、受影响 app tests 和 backend 完整测试
- **THEN** 现有业务接口、SQL 参数化、错误语义、结果映射、稳定 ID 和调用方行为必须保持通过

#### Scenario: 阻止架构扩张
- **WHEN** repository 文件整理不需要新的运行时能力
- **THEN** 变更不得新增 GORM、sqlc、ORM、codegen、query DSL、base repository、service 或 transaction framework
