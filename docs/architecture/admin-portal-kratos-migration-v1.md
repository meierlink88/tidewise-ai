# Admin Portal Service Kratos 迁移设计 V1

## 状态

已实施。功能范围由 GitHub Issue #97 冻结，工程基线采用 `origin/main@8127814`，
其中 Issue #100 / PR #101 已完成按应用根目录重组，Miniapp Kratos 试点由
`44191c6` 落地。

## 1. 结果与非目标

Admin Portal Backend 一次性迁移为完整 Kratos v3 Application：

```text
admin-portal/backend/
├── api/admin/v1/
├── cmd/server/
├── configs/
└── internal/{conf,biz,data,service,server}
```

迁移保持 Issue #97 的 AgentRun 管理代理、现有 Data 管理查询、HTTP paths、DTO、
状态码、错误 envelope、Request ID、认证、CORS、OpenAPI、Swagger UI 和部署行为不变。

非目标：

- 不修改 AgentRun、Data Domain Service 或 Admin Frontend 的功能合同；
- 不新增数据库、共享运行时 package、gRPC、Protobuf、Wire、服务发现、远程配置、
  Kubernetes、Redis、MQ 或 Service Mesh；
- 不保留 Gin/Kratos 双栈、旧目录 wrapper 或旧入口；
- 不修改 Schedule、Execution、Provider、Connector 的领域规则和事实 owner。

## 2. Owner Map

| 责任 | Owner |
| --- | --- |
| 管理员界面 | `admin-portal/frontend` |
| 浏览器 API、认证、前端 DTO、错误映射 | Admin Application Backend Service |
| AgentRun consumer port 与页面编排 | `admin-portal/backend/internal/biz` |
| Data/AgentRun HTTP adapter | `admin-portal/backend/internal/data` |
| Schedule、Execution、Provider、Connector 事实 | AgentRun |
| Raw Document、Event 事实 | Data Domain Service |
| 调用方 | Admin Frontend 只调用 Admin Backend |

不同 Backend 继续只通过版本化 REST API 协作；Admin Backend 不获得 Data 或 AgentRun
数据库凭据，也不 import 其实现。

## 3. 当前合同

- Admin API 前缀保持 `/api/admin/v1`。
- Data 查询保持 Raw Document 与 Event 的现有分页、筛选、时间和错误语义。
- AgentRun 管理保持 Schedule 查询/完整保存/局部启停、Execution 每页 20 条、
  Provider 与 Connector 列表/读取/更新。
- 保存 Schedule 不改变既有 `enabled`；首次创建默认停止。
- 浏览器 Bearer Token 只用于 Admin API；出站 Data/AgentRun 分别注入独立 Service
  Token，浏览器 Token 不转发。
- AgentRun Secret 永不进入响应或日志；上游错误正文、URL、Token 和 request ID 不透传。
- 成功与失败继续使用 Tidewise envelope，`X-Request-ID` 与响应体一致。
- `/healthz`、`/readyz` 保持直接运维对象；local/UAT 提供 OpenAPI 与 Swagger，
  prod 不注册文档路由。
- AgentRun 不可用只影响采集器配置接口，Data 查询仍独立可用。

OpenAPI 3.0.4 与现有 HTTP/Frontend 测试是迁移期间的线协议冻结依据。

## 4. 采用的参考

- `docs/adr/0006-kratos-official-service-layout.md`
- `docs/architecture/kratos-backend-development-standard-v1.md`
- `docs/architecture/miniapp-kratos-pilot-v1.md`
- `miniapp/backend` 在 `origin/main@8127814` 的已实施结构
- `github.com/go-kratos/kratos/v3 v3.0.0`

采用 Kratos App、HTTP Server/Router、显式 middleware、固定 URL HTTP Client、
显式构造和有界清理。拒绝 Miniapp 未采用的 Proto/gRPC、Wire、Discovery、远程配置
与平台控制面。

## 5. 目标落位与依赖

```text
api/admin/v1 <- internal/service -> internal/biz <- internal/data
      ^                    ^
      └──── internal/server

cmd/server -> conf + data + biz + service + server + kratos.App
```

- `api/admin/v1`：OpenAPI、wire DTO、API interface、Kratos HTTP 绑定。
- `internal/conf`：YAML/env 加载、默认值和启动校验。
- `internal/biz`：Admin 用例、业务模型、Data/AgentRun consumer-owned ports。
- `internal/data`：Data 与 AgentRun 的 Kratos HTTP adapters。
- `internal/service`：API DTO 与 Biz 输入输出转换、稳定错误分类。
- `internal/server`：Kratos HTTP Server、认证/CORS、Request ID、Recovery、日志、
  envelope、health/readiness 和文档。
- `cmd/server`：显式装配、Kratos App、运行和有限资源清理。

迁移完成后删除旧 `api/`、`cmd/`、`config/`、`dataclient/`、`agentrunclient/`、
`transport/`、`usecase/`、根 `service.go` 与 `http_runtime.go`。

## 6. 失败与安全

- Admin API 认证继续 fail closed；CORS 只允许配置的单一 origin。
- Kratos HTTP request timeout 显式设为 `0`，由两个下游 adapter 的 5 秒总预算控制；
  read/write timeout 保持现有配置。
- 只读 GET 按既有规则最多两次，写请求不自动重试；所有尝试共享单一 deadline。
- 远程调用显式声明 Kratos Operation 与 PathTemplate；Data 层使用自有 wire DTO
  解码、校验并映射为 Biz 模型，Biz 不持有 URL、HTTP 状态码或 JSON 传输标签。
- 成功/错误正文继续限制为 1 MiB，request body 限制为 128 KiB。
- 404/405、panic、认证失败和上游失败都返回安全 Admin envelope。
- 结构化日志不记录 Token、Authorization、Prompt、API Key 或上游错误正文。

## 7. 发布与回滚

源码、binary、Docker、Compose 和 CI 一次切换到 `cmd/server`，不维护双入口。
HTTP 与环境变量合同不变，因此 Frontend、Data 和 AgentRun 不要求同时发布。
回滚使用迁移前已知良好镜像或 Git revision，不执行数据 rollback。

## 8. 验证

默认 seam：

- Biz：Schedule 保存保留启停状态、首次创建停止、Collector Execution 固定分页、
  Data 与 AgentRun 故障隔离。
- API/HTTP：认证、CORS、绑定、状态码、envelope、Request ID、404/405、OpenAPI/runtime。

条件 seam：

- Data Adapter：两个远端 client 的 Token、request ID、timeout、重试、正文限制、
  decode、关闭与安全错误。
- Conf/Lifecycle：环境校验、必需 Secret、启动前失败、Kratos App、有界 shutdown/cleanup。
- Architecture：官方 Layout、依赖方向、无旧目录、无 Gin/Wire/共享运行时 import。
- Provider/Consumer 与部署：Data/AgentRun 合同漂移、三个 Backend binary、Admin image、
  local/UAT Compose。
- Frontend：保留 Issue #97 已确认的页面到模拟 Admin API 行为测试。

## 9. 开放决策

无。用户已明确要求保持 Issue #97 功能需求并按 Miniapp 同等 Kratos 规范迁移
Admin Portal；实现不得借迁移改变业务合同。
