# AgentRun Kratos Service 迁移 Spec

状态：Implemented and validated; awaiting PR merge
日期：2026-07-25
涉及系统：`tidewise-ai-agentrun`
GitHub Issue：https://github.com/meierlink88/tidewise-ai-agentrun/issues/26

## Outcome

把 AgentRun 迁移为一个符合观潮家 Backend Service 规范的 Kratos v3 Application，
统一进程生命周期、HTTP/OpenAPI、配置、业务规则、基础设施 Adapter 和 API Service
的工程边界，同时完整保留各 Agent 能力内部使用 CloudWeGo Eino 的机制。

## Non-goals

- 不用 Kratos 替换 Eino。
- 不改变 Collector 的查询规划、七通道采集、确定性 Candidate 处理或 Artifact 合同。
- 不引入 gRPC、Protobuf、Wire、服务发现、配置中心、Service Mesh、Redis、MQ 或
  Kubernetes 运行前提。
- 不因框架迁移新增 Agent、Connector、模型能力或跨仓库业务能力。

## Confirmed Decisions

### One repository, one Kratos Application

本仓库本身就是一个可独立构建和部署的 AgentRun Service，因此以仓库根作为一个
Kratos Application，不再增加 `services/agentrun` 包装目录。

目标稳定结构为：

```text
api/agentrun/v1/
cmd/
├── server/
├── migrate/
├── config/
└── artifacts/
configs/
internal/
├── conf/
├── biz/
│   ├── platform/
│   └── agents/
│       └── collector/
├── data/
│   ├── postgres/
│   ├── modelprovider/deepseek/
│   ├── connectors/
│   ├── artifacts/
│   └── scheduler/
├── service/
└── server/
```

Kratos 的 `biz/data/service/server` 是顶层工程共同语言。Agent 能力继续作为
`internal/biz/agents` 下的独立能力包；后续 Event Extractor、Analyst 等 Agent
各自拥有 Eino Workflow/Graph，不进入 AgentRun 平台业务包。

当前 `internal/architecture/layout_test.go` 所要求的 capability-first 顶层目录与
目标 Kratos Layout 冲突。迁移完成时必须用新的 Kratos 依赖方向和目录门禁替换该
规则，不保留旧目录或兼容转发包。

### Full Kratos HTTP contract cutover

本次迁移完整采用观潮家 Kratos Backend Service 规范，不保留现有 `net/http`
Router、响应 body 或错误格式的历史兼容：

- OpenAPI 3.0.4 移至 `api/agentrun/v1/` 并继续作为唯一 HTTP wire contract。
- API 注册使用 Kratos v3 Router；手写 Handler 必须设置稳定 operation，并显式执行
  Kratos Middleware。
- 业务成功响应统一为 `request_id + result` 信封，业务错误统一为
  `request_id + error` 信封；`X-Request-ID` 与 body 中的 `request_id` 一致。
- `/healthz` 与 `/readyz` 保持直接运维响应，不套业务信封。
- 统一实现安全的 404、405、panic recovery、认证和响应编码，不返回 Kratos 默认
  Error JSON。
- 不注册旧响应格式、旧 Router 或兼容 alias，不运行 Kratos 与旧 HTTP 双栈。
- 调用方必须依据新 OpenAPI 更新；本仓库只拥有 AgentRun Provider 合同，不直接修改
  Admin Portal Service 或 Tidewise Data Service。

该选择允许 breaking change。迁移以一次性切换完成，回滚使用迁移前二进制或镜像，
而不是在源码中保留永久兼容层。

### Repository and consumer scope

本任务只修改 `tidewise-ai-agentrun`。AgentRun 发布新的 OpenAPI、请求/响应 fixture
和消费方迁移说明，但不修改 `tidewise-ai` 中的 Admin Portal Service 或 Tidewise
Data Service 客户端。

Kratos 本身不要求改变 API 语义；本次消费方影响来自已确认采用的 Tidewise
`request_id + result/error` HTTP 信封。新 AgentRun 不保证与旧客户端兼容，消费方
适配和部署编排属于后续跨仓任务。

### Kratos and Go baseline

按统一规范固定 `github.com/go-kratos/kratos/v3 v3.0.0`，并把根 `go.mod`、CI 和
后续构建镜像的 Go 基线升级到 Go 1.25。不得跟随 `latest`，也不得使用 Kratos v2
示例或 import。

### Responsibility split, not package relocation

迁移必须拆开当前混合在同一 package 中的业务规则、Eino 编排和基础设施，不得把
现有 package 原样移动到 Kratos 目录后继续反向依赖。

目标职责为：

```text
internal/biz/
├── platform/
└── agents/collector/
    ├── planning/
    ├── workflow/
    ├── materialization/
    └── usecase.go

internal/data/
├── postgres/
├── modelprovider/deepseek/
├── connectors/
├── artifacts/
└── scheduler/
```

- `internal/biz/platform` 拥有 Agent Definition、Agent Version、Agent Execution、
  Agent Schedule 和当前 Provider/Connector 配置的规则、模型、错误与 Port。
- `internal/biz/agents/collector` 拥有 Collector Use Case、查询规划合同、Eino
  `compose.Workflow`、Candidate 处理和确定性物化规则。
- Collector Biz 可以依赖 Eino core 的 `compose`、`model` 和 `schema`，但不得依赖
  Kratos HTTP、pgx、文件系统、gocron、DeepSeek SDK 或 Eino Ext Provider 类型。
- `internal/data/modelprovider/deepseek` 使用官方 Eino Ext DeepSeek 组件实现
  Biz-owned Model Factory/Provider Port，并只向 Biz 暴露 Eino core interface。
- 七个 Connector、PostgreSQL、Artifact 文件/index 和 gocron 分别作为 Data
  Adapter 实现 Biz-owned Port。
- canonical URL、正文选择、时间门禁、SHA-256、SimHash、Candidate 终态和守恒属于
  Collector Biz；目录、原子文件、TSV 读写与数据库 SQL 属于 Data。
- `internal/service` 只做 API DTO 与 Biz 输入输出转换；`internal/server` 只做 Kratos
  HTTP、Middleware、认证、健康、就绪和文档发布。

迁移必须保持 Collector 已冻结的查询规划、七通道、直接结果、确定性去重、
Candidate、Artifact 和审计行为，不借职责拆分修改业务语义。

### Kratos-owned process lifecycle

`cmd/server` 显式构造配置、Data Adapter、Biz Use Case、Service、Kratos HTTP
Server 和 `kratos.App`。不得继续自行监听 Signal 或调用
`http.Server.ListenAndServe`。

- 静态 Service 配置、数据库连接、兼容 Schema、Artifact 目录和 enabled Schedule
  runtime projection 全部验证成功后才允许 HTTP Server 监听。
- Model Provider/Connector 当前配置不是进程监听前提；Admin API 必须能在新环境中
  完成首次配置。配置不完整时 `/healthz` 正常、Admin API 可用、`/readyz` 返回未
  就绪，Collector 创建安全失败。
- `kratos.App` 固定 Name、Version、Logger、HTTP Server 和有限 StopTimeout。
- 启动时保留现有 prepared Artifact publication、stale Agent Execution、缺失终态
  Audit 和 enabled Schedule runtime projection 对账。
- 停机先停止新的 Schedule 触发，再停止接收 HTTP 请求，并取消当前进程拥有的活跃
  Agent Execution。
- 最多等待 10 秒让进程内 Execution 写入安全终态，然后关闭 Scheduler 和
  PostgreSQL 连接池；所有清理都有有限超时。
- 不新增任务队列、跨进程 Workflow resume 或 Eino checkpoint。意外退出继续由现有
  启动对账处理，不扩大为通用恢复平台。

## Compatibility Finding

Kratos 与 Eino 没有运行时或职责冲突：

- Kratos 拥有 Service 进程生命周期、HTTP/OpenAPI、配置、认证、数据库和外部
  Adapter 装配。
- Eino 只拥有一个具体 Agent Execution 内部的模型和 Workflow/Graph 编排。
- Agent 业务代码依赖 Eino core interface；DeepSeek 等 Eino Ext Provider 构造留在
  基础设施或 composition root。
- Kratos HTTP Handler 不构建 Eino Workflow，Eino 节点也不拥有 HTTP、Schedule、
  PostgreSQL 或 Service 生命周期。

版本层面也没有已知冲突：

- Kratos v3.0.0 的 `go.mod` 要求 Go 1.25。
- Eino v0.9.12 的 module 基线是 Go 1.18。
- Eino Ext DeepSeek v0.1.7 的 module 基线是 Go 1.24，并要求 Eino v0.7.13；Go MVS
  继续选择本项目更高的 Eino v0.9.12。
- Go 1.25 能构建较低 Go 基线的 Eino modules。Kratos 会提升部分
  `golang.org/x/*` 间接依赖，实施时必须通过 `go mod tidy`、`go list -m`、全量测试
  和 race test 验证，不使用 `replace` 或降级 Eino。

## Current Conflicts To Remove

当前实现能运行，但与目标 Kratos 规范存在以下结构冲突：

1. `cmd/agentrun-server` 自行监听 Signal 和调用 `http.Server.ListenAndServe`。
2. `internal/architecture/layout_test.go` 强制 capability-first 顶层目录。
3. OpenAPI、Swagger 和 `net/http` Handler 位于 `internal/agentrun`、
   `internal/collector`，没有 API/Service/Server 分层。
4. Collector HTTP Handler 直接识别 `pgx.ErrNoRows`，Transport 知道数据库错误。
5. Collector runtime 同时构造 Eino Workflow、DeepSeek、Connector 和文件物化。
6. Artifact package 同时包含 Candidate 规则、hash/dedup 和文件/index I/O。
7. Scheduling package 同时包含 Agent Schedule 规则和 gocron runtime Adapter。
8. YAML 位于 `internal/agentrun/config`，服务、CLI 和基础设施使用旧 package 路径。
9. 根 Go 基线为 1.24.7，不能直接引入 Kratos v3.0.0。
10. 当前 HTTP 成功/错误 body 不符合已确认采用的 Tidewise Kratos 信封。

## Owner Map

| Concern | Owner |
|---|---|
| HTTP wire contract、DTO、operation | `api/agentrun/v1` |
| API↔Biz 转换 | `internal/service` |
| Agent Definition/Version/Execution/Schedule | `internal/biz/platform` |
| Provider/Connector 当前配置规则 | `internal/biz/platform` |
| Collector 输入、执行、Candidate、确定性物化 | `internal/biz/agents/collector` |
| Eino Workflow 和 Query Planner | `internal/biz/agents/collector` |
| PostgreSQL、gocron、DeepSeek、Connector HTTP、Artifact 文件 | `internal/data` |
| Kratos HTTP、Middleware、认证、编码、健康、文档 | `internal/server` |
| 构造、Kratos App、有限清理 | `cmd/server` |
| AgentRun 数据库与 Artifact Volume | AgentRun |
| Admin Portal/Data Service 消费方适配 | 各自仓库的后续任务 |

## Target Layout And Package Mapping

```text
api/agentrun/v1/
├── contract.go
├── http.go
├── document.go
├── openapi.yaml
└── testdata/

cmd/
├── server/
├── migrate/
├── config/
└── artifacts/

configs/
├── config.dev.yaml
└── config.uat.yaml

internal/
├── conf/
├── biz/
│   ├── platform/
│   └── agents/collector/
│       ├── planning/
│       ├── workflow/
│       └── materialization/
├── data/
│   ├── postgres/migrations/
│   ├── modelprovider/deepseek/
│   ├── connectors/
│   ├── artifacts/
│   └── scheduler/
├── service/
├── server/
├── architecture/
└── testsupport/
```

现有文件按职责迁移：

| Current | Target |
|---|---|
| `internal/agentrun/execution.go`、`schedule.go`、`configuration*.go` | `internal/biz/platform` |
| `internal/agentrun/admin/service.go` | Platform Biz Use Case + `internal/service` |
| `internal/agentrun/scheduling/service.go` | Schedule 规则进入 Biz；gocron 进入 Data |
| `internal/agentrun/persistence/postgres` | `internal/data/postgres` |
| `internal/collector/types.go`、`configuration.go` | `internal/biz/agents/collector` |
| `internal/collector/planning`、`workflow` | 对应 Collector Biz 子包 |
| `internal/collector/application` | Collector Use Case、Ports、Execution Supervisor |
| `internal/collector/connectors` | `internal/data/connectors` |
| `internal/collector/artifacts` | 规则进入 Biz materialization；I/O 进入 Data artifacts |
| 两个 `httpapi` package | API binding + Service + Server |
| `internal/agentrun/openapi` | `api/agentrun/v1` + Server 文档注册 |
| `internal/agentrun/config` | `internal/conf` + `configs` |
| `cmd/agentrun-*` | `cmd/server`、`cmd/migrate`、`cmd/config`、`cmd/artifacts` |

迁移完成后删除旧目录和 pass-through package，不保留同义层。

## Dependency Direction

```text
api/agentrun/v1 <- internal/service -> internal/biz <- internal/data
       ^                    ^
       └──── internal/server

cmd/server -> conf + data + biz + service + server + kratos.App
```

- Biz 不 import `internal/data`、`internal/service`、`internal/server`、Kratos HTTP、
  pgx、gocron 或文件系统。
- Data 可以 import Biz 以实现 Port，但不 import Service、Server 或 API。
- Service 只 import API 和 Biz。
- Server import API、Service、Conf 和 Kratos，不 import PostgreSQL/Provider SDK。
- `cmd/server` 是唯一 composition root；不得在 Handler 内构造 Agent Workflow。
- Eino core 依赖只允许出现在具体 Agent capability；Eino Ext Provider 只允许出现在
 对应 Data Adapter。

## Eino Runtime Contract

Collector 继续使用 typed `compose.Workflow`：

```text
Collection Prompt
  -> Query Planner
  -> seven deterministic Connector nodes
  -> deterministic materialization
  -> Artifact publication
```

- Planner 依赖 `model.BaseChatModel`，不依赖 DeepSeek concrete type。
- Eino Workflow 仍由 Collector capability 构建和编译，Kratos Handler 不参与。
- 每次 Execution 先冻结当前 Provider/Connector 配置，再构造本次运行依赖；运行中
  配置更新不得改变已启动 Execution。
- Go 程序继续拥有 Connector 集合、并发、时间窗默认、排序、hash、dedup、Candidate
  终态、Publication 和状态迁移。
- 不引入 ADK、Tool calling、multi-Agent、checkpoint 或模型选择 Connector。
- Agent Execution 使用由 Kratos App 生命周期拥有的 execution context；HTTP
  请求结束不会取消已接受的异步 Execution，App 停机则会有界取消。

## HTTP And OpenAPI Contract

现有业务/Admin 路径和方法保持其语义，但所有业务响应切换到 Kratos 信封：

```json
{
  "request_id": "req-...",
  "result": {}
}
```

```json
{
  "request_id": "req-...",
  "error": {
    "code": "INVALID_REQUEST",
    "message": "request parameter is invalid",
    "details": {}
  }
}
```

- Collector、Admin 路径继续使用 `/api/v1` 与 `/api/admin/v1`。
- Service/Admin Bearer Token 保持独立；认证通过 Kratos Middleware 按 operation
  启用，不在每个 Handler 复制。
- `Idempotency-Key`、64 KiB Prompt/Input、严格 JSON、未知/尾随字段拒绝等合同保留。
- HTTP status、分页、时间、排序、null 和安全错误语义以更新后的 OpenAPI 为准。
- `X-Request-ID` 缺失或非法时生成安全 ID；header 与响应 body 一致。
- Request ID Filter 和完整链 panic recovery 覆盖路由、绑定、Middleware、Service
  与编码。
- 手写 API Handler 在调用 `ctx.Middleware` 前设置稳定 operation。
- 自定义 response/error encoder 保证不输出 Kratos 默认 Error JSON。
- `/healthz` 只表示进程和 HTTP transport 存活。
- `/readyz` 表示数据库/schema、配置、Artifact 和 Scheduler runtime 可接流量。
- local/UAT 注册 `/openapi.yaml`、`/docs`、`/docs/`；不依赖 CDN。由于 plaintext
  Credential ADR，当前没有受支持的 prod 环境。

## Configuration, Commands And Migrations

- YAML 移至 `configs/config.dev.yaml` 与 `configs/config.uat.yaml`。
- `internal/conf` 保留手写 typed YAML、环境覆盖、默认值和启动校验；不因 Kratos
  引入 Protobuf 配置或远程配置中心。
- Token、数据库密码和以后新增 Secret 继续只从环境变量或批准的 Secret Provider
  注入。
- Service 入口改为 `go run ./cmd/server`。
- Utility CLI 改为 `./cmd/migrate`、`./cmd/config`、`./cmd/artifacts`，复用 Conf、
  Biz 和 Data 构造，不复制规则。
- 旧 `cmd/agentrun-*` 入口删除，不留 wrapper。
- 现有 `001`–`006` SQL 原样移动到 `internal/data/postgres/migrations`。embed 相对路径
  仍为 `migrations/<file>`，因此 `schema_migrations.version` 不变。
- 本次没有 Schema 变化，不新增 `007` migration，不修改已有 SQL。
- 现有 PostgreSQL 数据、Schedule、Execution、Provider/Connector 配置和 Artifact
  文件不重写、不删除。

## Logging And Safe Errors

- 使用 Kratos v3 `log/slog`，输出 stdout/stderr。
- 请求日志至少包含 service、environment、operation、request_id、status 和 duration。
- Biz 不依赖全局 logger；后台 Execution 只注入最小事件日志 Port。
- 不记录完整 Prompt、Authorization、Provider/Connector Key、模型原始响应、Connector
  原始 body、数据库 DSN 或内部错误。
- API、CLI、Artifact 和日志继续执行现有 Credential redaction 合同。

## Verification Contract

最高可观察验收缝是一个真实 Kratos AgentRun Service 黑盒测试：

1. 使用隔离 PostgreSQL database 执行现有增量 migration。
2. 使用真实 Kratos v3 HTTP Server、真实 Router/Middleware/Encoder 和随机本地端口。
3. 使用 fake Eino model endpoint、七个 fake Connector endpoint 和临时 Artifact
   目录。
4. 通过 Admin API 配置运行依赖和 Schedule。
5. 通过 Collector API 与 Schedule 分别创建 Execution，观察终态、Invocation、
   Candidate 守恒和 Artifact。
6. 验证新成功/错误信封、Request ID、认证、严格 JSON、404、405、panic recovery、
   OpenAPI/runtime parity 和 Swagger 环境策略。
7. 触发 Kratos App shutdown，验证停止新触发、取消/等待活跃 Execution、关闭
   Scheduler 和 PostgreSQL。

附加门禁：

- 架构测试验证官方 Layout、依赖方向、旧目录删除、无 Wire/Protobuf/gRPC 业务入口，
  以及每个手写 route 实际执行 Middleware。
- 保留并移动现有 Collector deterministic golden、Connector fixture、Artifact
  publication、index rebuild/verify、failure injection 和 PostgreSQL migration 测试。
- API fixture 冻结 Collector/Admin 的新信封，供后续消费方适配。
- `gofmt`、`go vet ./...`、`go build ./cmd/...`、`go test -count=1 ./...`、
  `go test -race -count=1 ./...`、`docker build --tag tidewise-ai-agentrun:ci .`。
- 设置 `AGENTRUN_TEST_DATABASE_URL` 运行全部 PostgreSQL 和 Service 黑盒用例，skip
  不算通过。
- `go mod tidy`、`go list -m` 验证 Kratos v3.0.0、Eino v0.9.12、DeepSeek
  v0.1.7，无 `replace`、无 Eino 降级。
- 不要求真实外部 Provider smoke 才能验收，因为本次不改变 Provider/Connector
  业务合同；已有运行能力由 fake 黑盒和原有 golden 合同保护。

## Rollout And Rollback

- 在单一 feature branch 中一次完成目标 Layout、依赖拆分和 Kratos runtime 切换；
  不提交长期可运行的双栈中间状态。
- PR 只包含本仓库迁移文件，不包含用户已有 `AGENTS.md` 修改或无关研究 Spec。
- 新 OpenAPI 是 breaking contract；AgentRun 部署前，消费方必须在各自仓库完成适配。
- 数据库无需迁移到新 Schema。迁移 CLI 仍验证现有 `001`–`006` ledger。
- 回滚使用迁移前二进制/镜像；因为没有数据库或 Artifact 格式变化，回滚不需要数据
  补偿。

## Risks And Mitigations

- **大范围移动掩盖行为变化**：用现有 golden、最高层黑盒和 diff scope review
  锁定功能。
- **API breaking change**：发布完整 OpenAPI、fixture 和消费方迁移说明，不声称旧
  客户端兼容。
- **Go/依赖图提升**：固定 Kratos 版本，执行 MVS、全量、race 和构建验证。
- **Artifact 规则拆分产生 hash 漂移**：cross-language/deterministic golden 必须
  字节级不变。
- **优雅停机留下非终态**：execution supervisor 有界取消和等待，现有启动对账继续
  兜底。
- **框架侵入 Agent 逻辑**：架构测试禁止 Kratos HTTP/Data 依赖进入 Agent Biz，
  禁止 Eino Ext concrete Provider 进入 Planner/Workflow。

## Reference Audit

### Kratos

- Tidewise Kratos Backend Standard：
  `/Users/meierlink/Documents/david/创业项目/观潮家/tidewise-ai/docs/architecture/kratos-backend-development-standard-v1.md`
  ，参考仓库 commit `cdc67cdce45c373a403231343a43b6ac43eb15c1`。
- Kratos v3.0.0 module source：检查 `app.go`、`options.go`、
  `transport/http/server.go`、`transport/http/context.go`。
- Tidewise Miniapp 已迁移样板：检查 `cmd/server/app.go`、`internal/server/http.go`、
  `api/miniapp/v1/http.go`。
- 采用：`kratos.App` 生命周期、显式构造、Kratos Router、operation + Middleware、
  自定义 encoder、OpenAPI-first。
- 拒绝：gRPC、Protobuf、Wire、默认 1 秒 timeout、默认 Error JSON、registry、
  远程 config 和复制其他 Service 的业务 package。

### Eino

- `cloudwego/eino` commit
  `922b6a8a233b5233fe47eecee6cd2c005e8c39cd`：检查
  `compose/doc.go`、`compose/workflow.go` 和 typed Runnable/Workflow 合同。
- `cloudwego/eino-ext` commit
  `9137edd89e72b72735ede69db1c5ae29178a6e41`：检查
  `components/model/deepseek/deepseek.go` 与 module `go.mod`。
- `cloudwego/eino-examples` commit
  `171220631fb7068ead50b7cd964b8c471647117d`：检查 typed Workflow field mapping
  和 HTTP SSE Agent 示例。
- 采用：capability-local typed Workflow、core interface、显式输入/依赖和 context
  传播。
- 拒绝：示例级 global callback/model、Handler 内构建 Agent、Hertz runtime、
  `context.Background` 节点、ADK/multi-Agent 和把 Eino 当仓库顶层架构。

## Out Of Scope

- 修改 Tidewise Data Service、Admin Portal Service 或前端。
- 新增 Agent、模型、Connector、Tool、MCP、ADK 或 multi-Agent。
- 修改 Collector Prompt、查询规划 Schema、Candidate/Artifact 数据格式。
- 新数据库表、Schema migration、数据清洗或历史 Artifact 重写。
- gRPC、Protobuf、Wire、服务发现、远程配置、Kubernetes、Redis、MQ、Trace/Metrics
  Backend。
- 旧 HTTP body、旧命令路径、旧目录或旧 runtime 的兼容层。

## Open Decisions

无。用户授权在满足 Kratos 规范、Eino 最佳实践和功能不丢失的前提下，由本设计选择
可逆实现细节。
