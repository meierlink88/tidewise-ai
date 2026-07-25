# Data Domain Service Kratos 迁移设计 V1

## 状态

已确认，由 GitHub Issue #105 冻结；实现基线为已合并 AgentRun 应用后的
`main@3568f72`。`entity-seed` 保留，现有
`graph-projector` 删除，Neo4j 作为未来可重建投影基础设施保留，并解除与 Data
Server 的强启动依赖。命令生命周期、单任务/单 PR 的原子交付方式、共享集成顺序、
capability-oriented Biz/Data 分包、失败、安全、发布和验证方案均已冻结。

## 1. 结果与非目标

Data Domain Service 一次性迁移为完整 Kratos v3 Application：

```text
analyse-data-service/backend/
├── api/data/v1/
├── cmd/server/
├── configs/
└── internal/
    ├── conf/
    ├── biz/
    ├── data/
    ├── service/
    └── server/
```

“功能不变”冻结的是迁移前后可观察的业务与运行合同，不要求保留现有源码组织或内部
实现。迁移必须像 Miniapp Service 与 Admin Portal Service 一样，完整采用 Tidewise
Kratos 规范：

- 使用 `kratos.App` 管理进程生命周期；
- 使用 Kratos HTTP Server、Router、Filter 和 Middleware；
- 使用显式构造函数装配，不使用 Wire；
- 将 API 绑定、配置、业务规则、基础设施 Adapter、API/Biz 转换和 Server 机制分别
  收敛到标准目录；
- 删除旧顶层 `config`、`domain`、`repositories`、`adapters`、`transport`、
  `usecase`、旧 HTTP 入口和根级装配文件，不保留双运行时、wrapper 或兼容目录；
- 允许为满足依赖方向、错误处理、资源生命周期和测试 seam 而重写内部实现。

迁移期间必须保持：

- 现有 OpenAPI paths、methods、DTO、status、error envelope、Request ID、时间、
  顺序、空值和分页语义；
- 现有认证主体、Service Token 来源、权限和安全错误边界；
- Data 的业务规则、数据库 schema、migration 历史、数据和持久化语义；
- Data 与 AgentRun、Miniapp、Admin Portal 的事实 ownership 和版本化 REST API
  边界；
- 现有 binary、容器、配置、健康检查和部署能力；其工程路径可以按 Kratos 规范
  一次切换。

非目标：

- 不修正已知业务或 HTTP 合同偏差；
- 不新增、删除或改变数据库 schema 与 migration；
- 不改变 Event Publication、Research Theme、Research Anchor、查询或投影业务规则；
- 不改变 Data 与 AgentRun 的 Source、Artifact、Execution、Event 或 Research 事实
  ownership；
- 不引入 gRPC、Protobuf、Wire、服务发现、远程配置、Kubernetes、Redis、MQ、
  Service Mesh 或跨应用共享运行时 package；
- 不在本迁移中合并 AgentRun 应用或修改 AgentRun 实现。

## 2. Owner Map

| 责任 | Owner |
| --- | --- |
| Data HTTP API、认证与线协议 | Data Domain Service |
| 投研领域规则与事实 | Data Domain Service |
| PostgreSQL schema、migration、repository | Data Domain Service |
| Neo4j 可重建投影 | Data Domain Service |
| Event Publication 接纳合同与事务 | Data Domain Service |
| Source、完整 Artifact、Agent 与 Execution 事实 | AgentRun |
| Data 查询调用方 | Miniapp 与 Admin Portal Application Backend Service |
| Event Publication 调用方 | AgentRun |

共仓不改变所有权。不同 Backend 继续只通过版本化 REST API 协作，不共享数据库、
Repository、Go model、Service Token 或实现 package。

## 3. 当前冻结合同

迁移前 OpenAPI 位于 `analyse-data-service/backend/api/openapi.yaml`，以下运行时路由
必须保持：

- `GET /healthz`
- `GET /readyz`
- `GET /api/data/v1/research/themes`
- `GET /api/data/v1/research/themes/{theme_id}`
- `GET /api/data/v1/research/themes/{theme_id}/reasoning-trees`
- `GET /api/data/v1/research/themes/{theme_id}/reasoning-trees/{anchor_id}`
- `GET /api/data/v1/raw-documents`
- `GET /api/data/v1/events`
- `POST /api/data/v1/reviewed-event-imports`
- `POST /api/data/v1/research-theme-imports`
- `POST /api/data/v1/research-anchor-imports`

OpenAPI 文件迁移到 `api/data/v1/openapi.yaml` 只改变工程位置，不改变线协议。详细
请求、响应、鉴权、错误、幂等和事务语义继续以 Data Context、当前 OpenAPI、冻结
fixture 与现有合同测试为准。

## 4. 采用的参考

- `docs/contexts/data/CONTEXT.md`
- `docs/adr/0002-backend-service-architecture.md`
- `docs/adr/0006-kratos-official-service-layout.md`
- `docs/adr/0007-app-oriented-monorepo.md`
- `docs/architecture/kratos-backend-development-standard-v1.md`
- `docs/architecture/miniapp-kratos-pilot-v1.md`
- `docs/architecture/admin-portal-kratos-migration-v1.md`
- `github.com/go-kratos/kratos/v3 v3.0.0`

采用 Miniapp 与 Admin Portal 已验证的 Tidewise Kratos Application 模式；不复制
官方示例中的 Proto、gRPC、Wire、Discovery 或远程配置。

## 5. 目标落位与依赖

```text
api/data/v1 <- internal/service -> internal/biz <- internal/data
      ^                    ^
      └──── internal/server

cmd/server -> conf + data + biz + service + server + kratos.App
cmd/<tool> -> conf + required biz/data capabilities
```

Data 不使用单一巨型 `biz` 或 `data` package 机械承接旧目录，而是在标准层内按
capability 分包：

```text
internal/biz/
  eventpublication/
  researchthemeimport/
  researchanchorimport/
  research/
  adminquery/
  entityseed/

internal/data/
  postgres/
  dbmigration/
  eventpublication/
  researchthemeimport/
  researchanchorimport/
  research/
  adminquery/
  entityseed/
```

- 每个 Biz capability 拥有自己的最小 Port、模型、规则、Use Case 和稳定错误。
- `internal/data` 分别实现 Biz Port；可以在 Data 层内部复用连接池、事务创建等无
  业务语义的 PostgreSQL 机制，但不得继续暴露覆盖全部能力的巨型 Repository。
- `internal/service` 按 API capability 完成 wire DTO、transport 校验和 Biz
  输入输出转换，不访问数据库或具体 Adapter。
- `api/data/v1` 拥有 OpenAPI、wire DTO、API interface 和薄 Kratos HTTP 绑定。
- `internal/server` 拥有 Kratos HTTP Server、Filter、Middleware、认证、Request
  ID、Recovery、日志、envelope、health/readiness 和文档。
- `internal/conf` 拥有 YAML/env 加载、默认值、Secret 注入和启动校验。
- 每个 `cmd/<tool>` 只装配该命令需要的 capability，不获得整个 Data Service
  依赖集合。

## 6. 已确认命令与基础设施边界

已确认：

- `graph-projector` 当前 Neo4j 投影规则已经落后于多轮实体关系迭代，本任务删除
  现有 `cmd/graph-projector`、`usecase/graphprojection`、`adapters/graphdb` 及其
  专属测试和失效的运行说明。未来按新的业务规则重新设计 projector，不继承本次
  删除实现的兼容责任。
- Neo4j 仍由 Data Domain Service 拥有，是 PostgreSQL 事实的未来可重建投影目标；
  删除旧 projector 不表示放弃 Neo4j，也不把 Neo4j 提升为事实源。
- 没有 projector 的过渡期内，Neo4j 只作为可选本地投影基础设施保留。Data Server
  不再 `depends_on` Neo4j，不要求 Neo4j 凭据，Kratos Server 配置也不携带未使用的
  Neo4j 字段。未来 projector 必须独立拥有 Neo4j 配置、连接、失败处理和生命周期。
- `entity-seed` 当前负责校验并幂等写入 Data 主数据基线，包括实体、类型 Profile、
  实体关系、外部标识和若干经人工冻结的批次。现有架构审计还要求在 fresh database
  可完整重建前保留其历史模式，因此不能按普通开发脚本直接删除。
- `entity-seed` 定义为仓库支持的离线 Data 运维命令，保留现有行为和
  `cmd/entity-seed` 入口，但不要求进入 Data Server 生产镜像。业务规则迁入
  `internal/biz`，PostgreSQL Adapter 迁入 `internal/data`。
- `dbmigrate` 是生产发布命令，保留现有 binary 名称、check-only、`-apply`、
  `-target-version`、报告和失败语义，并继续进入 Data 镜像供 UAT migration
  preflight、风险门禁和部署流程调用。
- `research-theme-dev-reset` 与 `research-theme-dev-seed` 是仓库支持的本地开发
  命令，保留现有行为，但不进入生产镜像；两者在非 local 环境继续 fail closed。
- 迁移后保留 `cmd/server`、`cmd/dbmigrate`、`cmd/entity-seed`、
  `cmd/research-theme-dev-reset` 和 `cmd/research-theme-dev-seed`。只有长期运行的
  `cmd/server` 创建 `kratos.App`；其余命令是复用 `internal/conf`、
  `internal/biz` 与 `internal/data` 的短生命周期 composition root。

## 7. 交付边界

本任务使用一个 GitHub Issue、一个实现分支和一个 PR，一次合并完成切换。PR 内部
必须按可独立审查的提交隔离：

1. 迁移设计与现有合同冻结；
2. 旧 `graph-projector` 退役及 Data Server/Neo4j 解耦；
3. Data 剩余能力迁移到 Kratos 官方 Application Layout；
4. 应用级合同、生命周期和架构验证；
5. 基于 AgentRun 合并后的最新 `main` 收口 CI、Compose、仓库契约和根级文档。

最终 PR 唯一允许的功能差异是旧 `graph-projector` 不再存在。其余 HTTP、数据库、
命令、认证、配置和部署合同以 projector 退役后的基线为准保持不变。源码、binary、
Docker、Compose 和 CI 最终一次切换，不保留旧目录、旧入口或双运行时。

## 8. AgentRun 并仓后的实施基线

GitHub Issue #104 / PR #106 已完成 AgentRun 历史迁入及根 `go.mod/go.sum`、CI、
Local/UAT Compose、UAT CD、Context Map 和仓库级契约，并以
`main@3568f72` 通过 Data、Miniapp、Admin Portal、AgentRun 与 Security CI。

Issue #105 已 rebase 到该提交。实现必须保留共仓后的四 Context、独立数据库与
Artifact ownership、根 module 和共享交付结构，不恢复并仓前的外部仓库假设。
当前 AgentRun 尚未实现 Data Event Publication HTTP client；本迁移只保持 Data
provider OpenAPI、fixture 和现有消费方合同，不以框架迁移为由新增 AgentRun
publication capability。

## 9. 失败、安全与生命周期

- 保留四个 Service Identity 与 Scope：
  - `agent-run`：`data.reviewed-events.import`
  - `research-theme-publisher`：`data.research.import`
  - `miniapp-bff`：`data.research.read`
  - `admin-portal-bff`：`data.admin.read`
- Service Token 继续只来自环境变量；使用常量时间比较，不写入配置样例、响应或日志。
- 认证和授权继续 fail closed；浏览器凭据不得作为 Data Service 凭据使用或转发。
- `X-Request-ID` 与响应 `request_id` 保持一致；缺失或超过 128 字符时由 Data 生成
  新 ID。不得暴露调用方之外的内部或下游 ID。
- 写请求继续使用严格 JSON 解码、拒绝未知字段，并保持 1 MiB request body 上限。
- 业务错误继续映射为当前稳定 status、code、message 和 details；SQL、连接字符串、
  Token、请求正文和内部错误不得进入响应。
- Request ID 与 panic recovery 使用最外层 Filter 覆盖完整请求链；认证、访问日志等
 使用明确的 Kratos Middleware。手写路由必须设置稳定 Operation 并实际执行
  Middleware。
- Kratos HTTP request timeout 显式设为 `0`，避免框架默认 1 秒改变现有长事务行为；
  `net/http` read/write timeout 保留现有配置：local 为 5/15 秒，UAT 为 5/30 秒。
- 配置、PostgreSQL 连接和只读 migration readiness 检查必须在监听前成功；失败时
  进程退出，不提供部分可用 Server。
- `/healthz` 与 `/readyz` 的现有 wire contract 不变；`/readyz` 继续报告
  `config=ok`，不在每次探测时访问 PostgreSQL 或 Neo4j。
- local/UAT 继续提供 OpenAPI 与 Swagger UI，prod 不注册文档路由。
- `kratos.App` 负责信号和 Server 生命周期，并设置有限 `StopTimeout`；停止监听后
  显式、有界地关闭 PostgreSQL 等 Adapter 资源。
- Neo4j 不参与 Data Server 配置校验、readiness 或 shutdown。

## 10. 发布与回滚

- Data PR 必须保持 `main@3568f72` 之后形成的 AgentRun 根 module、CI 和交付合同。
- Data 源码、OpenAPI 位置、binary build path、Docker、Compose、CI 和文档一次切换
  到新结构，不维护旧入口或双栈。
- Data OpenAPI、数据库 schema、migration ledger 和调用方合同不变，因此 Miniapp、
  Admin Portal 与 AgentRun 不要求与 Data 同时发布。
- UAT 继续先使用目标 Data 镜像中的 `dbmigrate` 执行 check-only、风险门禁和
  `-apply`，再启动新 Data Server。
- 本任务不新增 SQL migration，不转换或删除 PostgreSQL/Neo4j 数据。
- 应用回滚使用迁移前已知良好镜像或 Git revision，不执行 down migration。
- 旧 projector 被删除后，回滚到迁移前镜像只恢复旧命令代码；不得在规则已知过时的
  情况下自动执行旧投影。

## 11. 验证

迁移开始前先固定现有可观察基线；迁移完成后对相同合同重新执行。

默认 seam：

- Biz：Event Publication、Research Theme Import、Research Anchor Import、Research
  Query、Admin Query、Entity Seed 的现有规则、事务前校验、幂等、冲突和稳定错误。
- API/HTTP：全部 OpenAPI 路由、DTO、严格绑定、分页/时间校验、status、envelope、
  Request ID、认证/授权、1 MiB 限制、404/405、panic recovery 和文档路由。

本迁移触发的条件 seam：

- Data：拆分后的 capability Port 与 PostgreSQL Adapter 保持现有 SQL、事务、锁、
  顺序、null 和错误清洗语义；保留受影响的 PostgreSQL 集成测试。
- Migration：`dbmigrate` check/apply/target-version、ledger、锁、read-only readiness
  和 UAT 风险报告保持兼容；SQL migration 文件内容和顺序不得改变。
- Conf/Lifecycle：local/UAT 默认值、环境 Secret 校验、启动前依赖失败、Kratos
  App、信号、有界 shutdown 和 PostgreSQL cleanup。
- Architecture：完整官方 Layout、Biz 不反向依赖 Data/Service/Server、无跨应用
  实现 import、无 Wire/双栈/旧目录/根级运行时文件。
- Projector 退役：旧 command、graph projection Use Case、Neo4j Adapter 和失效运行
  说明不存在；Neo4j local 基础设施仍可选存在，但 Data Server 不依赖其健康或凭据。
- Provider/Consumer：Miniapp/Admin Data client 与 Data OpenAPI/fixture 合同保持；
  Data Event Publication provider OpenAPI 与冻结 fixture 保持。当前 AgentRun 没有
  Data publication client，本迁移不新增消费者实现。
- Delivery：Data Server 与 `dbmigrate` binary、Data image、local/UAT Compose config、
  Miniapp→Data smoke 及 Data 不可用安全失败路径。

完成门禁：

- Go format、受影响 package vet/test/race；
- Data 全量测试与仓库契约测试；
- 所有受影响 Backend binary 和 image 构建；
- local/UAT Compose config；
- 最高可观察 provider-consumer smoke；
- 确认根 `go.mod/go.sum` 中只保留迁移后实际引用的依赖。

不为简单构造函数、机械 DTO/PO 复制、框架注册胶水或无校验配置读取新增单元测试。

## 12. 开放决策

无实现级开放决策。Issue #104 门禁已满足并完成合并后基线复核；实现期间如发现会
改变已冻结合同、数据库、所有权或交付顺序的新事实，必须返回设计门禁。
