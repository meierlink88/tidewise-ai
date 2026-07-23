# OpenAPI 与 Swagger UI V1

## 状态

已实施。

## 目的

为 Tidewise AI 的三个可独立部署 Backend Service 建立各自拥有、可验证、可浏览的 HTTP API 合同：

- Data Domain Service
- Miniapp Application Backend Service
- Admin Portal Application Backend Service

OpenAPI 文档描述服务对外暴露的 HTTP 合同；Swagger UI 只负责浏览和调试这些合同，不成为 API 事实来源。

## 已确认决策

### 每个 Service 独立拥有合同

每个 Backend Service 都维护自己的 OpenAPI 3.0.4 文档，不建立中央聚合合同：

```text
src/backend/
├── internal/platform/
│   └── apidocs/
│       ├── handler.go
│       └── handler_test.go
└── services/
    ├── data/
    │   └── api/
    │       ├── openapi.yaml
    │       ├── document.go
    │       └── openapi_test.go
    ├── miniapp/
    │   └── api/
    │       ├── openapi.yaml
    │       ├── document.go
    │       └── openapi_test.go
    └── adminportal/
        └── api/
            ├── openapi.yaml
            ├── document.go
            └── openapi_test.go
```

`internal/platform/apidocs` 只提供无业务语义的 OpenAPI 文档与 Swagger UI HTTP 交付机制。每份 `openapi.yaml` 及其合同测试由所属 Service 维护。

### 固定入口

每个 Service 使用相同的文档入口：

```text
GET /openapi.yaml
GET /docs
```

`/docs` 重定向到 Swagger UI 使用的规范目录入口 `/docs/`。

### Swagger UI 依赖

固定使用：

```text
github.com/swaggest/swgui v1.8.9
github.com/swaggest/swgui/v5emb
```

`v5emb` 在 Go 二进制中嵌入 Swagger UI v5.32.8 静态资源。项目不复制 Swagger UI 资源文件，不使用 CDN，也不新增前端构建步骤。

### API 路径规范

三个 Service 统一使用：

```text
/api/{service}/v{major}/{resource}
```

当前目标前缀：

```text
Data Service:         /api/data/v1/...
Miniapp Service:      /api/miniapp/v1/...
Admin Portal Service: /api/admin/v1/...
```

Data Service 的内部调用边界不再通过 `/internal` 路径表达，而由部署网络、Bearer service token、权限 Scope 和架构依赖规则保证。路径统一不授权 Miniapp Frontend 或 Admin Portal Frontend 直接调用 Data Service；Frontend 仍只能调用所属 Application Backend Service。

Data Service 当前只维护一个服务级 API 主版本 `v1`。现有
`/internal/data/v2/reviewed-event-imports` 中仍有效的 Event Publication 合同迁移到
`/api/data/v1/reviewed-event-imports`；这次路径收敛不改变其请求、响应和事务语义。
当前有效合同在代码与 OpenAPI 中使用中性名称，例如
`EventPublicationRequest`、`EventPublicationResult` 和
`publishReviewedEvents`，不再把历史合同代际 `V2` 写入当前类型名或
`operationId`。已接受 ADR 和历史设计文档继续保留 “V2” 名称，用于记录合同演进。

Data 路径按 Data 自身资源命名，不包含调用方名称。现有
`/internal/data/v1/admin/raw-documents` 和
`/internal/data/v1/admin/events` 分别迁移为：

```text
/api/data/v1/raw-documents
/api/data/v1/events
```

它们继续使用 `data.admin.read` 权限。Admin Portal BFF 面向自身前端提供独立的
`/api/admin/v1/raw-documents` 和 `/api/admin/v1/events`。

### 环境策略

| 环境 | `/openapi.yaml` | `/docs` | Swagger UI `Try it out` |
|---|---|---|---|
| local | 开放 | 开放 | 开放 |
| uat | 开放 | 开放 | 开放 |
| prod | 不注册 | 不注册 | 不适用 |

生产环境不通过隐藏按钮或运行时拒绝实现关闭，而是不注册两个文档路由。

V1 不为 UAT 的 `/openapi.yaml` 或 `/docs/` 增加应用层认证。UAT 文档入口依赖部署网络边界控制；该决定不表示文档适合公开互联网访问。未来如需公网长期开放，应单独增加访问保护。

### Swagger UI 凭据

- Data Service 使用 OpenAPI Bearer security scheme；Local 和 UAT 使用者通过 Swagger UI `Authorize` 手工输入对应环境的 service token。
- Admin Portal 使用 OpenAPI Bearer security scheme；Local 和 UAT 使用者手工输入对应环境的 admin token。
- Miniapp Service 按当前运行时合同不声明鉴权。
- OpenAPI、Swagger UI HTML、配置文件和构建产物均不得预置 token。
- Swagger UI 使用 `persistAuthorization=false`，刷新页面后不保留凭据。
- Local 和 UAT 均允许 `Try it out`，实际请求仍必须通过服务自身的真实鉴权。

### Server 地址

每份 OpenAPI 文档只使用相对同源地址：

```yaml
servers:
  - url: /
```

合同不写死 Local、UAT 的主机、IP 或端口。Swagger UI 从哪个 Service origin 打开，就向该 origin 发起 `Try it out` 请求；未来增加域名或反向代理时不需要改写合同。

### 合同独立性

三个 Service 的 OpenAPI 文档和 Schema 完全独立：

- 每个 Service V1 使用单个 `api/openapi.yaml` 入口文档。
- 不建立跨 Service 的公共 OpenAPI Schema 目录。
- 不使用跨 Service 文件 `$ref`。
- 相似的健康检查、错误响应或分页 Schema 由每个 Service 在自己的合同中定义。
- Miniapp 和 Admin Portal 不通过引用 Data OpenAPI 来构造自身前端合同。

该边界允许任一 Service 独立构建、部署或拆分仓库。

### Service 合同范围

- Data OpenAPI 只描述 `/api/data/v1/...`。
- Miniapp OpenAPI 只描述 `/api/miniapp/v1/...`。
- Admin Portal OpenAPI 只描述 `/api/admin/v1/...`。
- Application Backend Service 不复制其调用的 Data API；Data API 只在 Data Swagger UI 中展示。

### 错误响应

三个 Service 的 JSON 错误响应统一为：

```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "request parameter is invalid",
    "details": {}
  },
  "request_id": "req-..."
}
```

- `error.code` 是稳定、机器可读的错误码。
- `error.message` 是安全的对外说明，不泄露内部错误或凭据。
- `error.details` 始终存在；没有附加信息时返回空对象。
- `request_id` 同时写入 `X-Request-ID` 响应头，供调用链和日志关联。
- Data、Miniapp 和 Admin Portal 在各自 OpenAPI 中独立定义该 Schema；
  不使用跨 Service `$ref`。
- Go 实现可以复用 `internal/platform` 下的错误响应工具，但各 Service
  仍独立决定业务错误码与 HTTP 状态映射。

### 成功响应

三个 Service 的业务 JSON 成功响应统一为：

```json
{
  "request_id": "req-...",
  "result": {}
}
```

- `request_id` 同时写入 `X-Request-ID` 响应头。
- `result` 保存该 operation 的业务结果；列表、详情和写入回执均遵循该信封。
- Miniapp Frontend、Admin Portal Frontend 和 Service 间 typed client 随路径迁移
  同步适配。
- `GET /healthz` 和 `GET /readyz` 是运维端点，继续直接返回运维状态对象，
  不套业务成功信封。

### 兼容现有结构

- Data Service 现有 OpenAPI 3.0.3 文档升级到 3.0.4，不重写已有业务合同。
- Miniapp 和 Admin Portal 分别新增自己的 OpenAPI 3.0.4 文档。
- 本次任务同时统一三个 Service 的运行时 API 路径、OpenAPI paths、内部客户端常量、测试、文档和 UAT 调用脚本：
  - Data：仍有效的 `/internal/data/v1/...` 接口迁移为 `/api/data/v1/...`；
    `/internal/data/v2/reviewed-event-imports` 收敛为
    `/api/data/v1/reviewed-event-imports`。
    Data 资源路径中的调用方片段 `/admin` 同时移除。
  - Miniapp：`/api/v1/miniapp/...` 迁移为 `/api/miniapp/v1/...`。
  - Admin Portal：`/admin/...` 迁移为 `/api/admin/v1/...`。
- 已退役且仅返回 `410 Gone` 的
  `/internal/data/v1/raw-document-imports`、
  `/internal/data/v1/raw-document-imports/{idempotency_key}` 和
  `/internal/data/v1/reviewed-event-imports` 直接删除，不迁移到新命名空间。
- 所有旧路径硬切换下线，不注册兼容别名或重定向。
- AgentRun 等仓外调用方不由本仓修改，但必须获得明确的旧路径到新路径迁移清单，并在部署前完成适配。
- 不移动现有 `transport/`、`usecase/`、`domain/`、repository 或 migration。
- 不通过 Go handler 注释反向生成合同。
- Swagger UI 通过固定的 `swgui v1.8.9` 及其 `v5emb` 包随服务构建，不依赖运行时外部 CDN。

### 合同权威与防漂移

每个 Service 的 `api/openapi.yaml` 是该 Service HTTP API 合同的唯一权威来源。Handler、请求 DTO、响应 DTO 和客户端继续手写，不从 OpenAPI 生成生产代码。

每个 Service 的 `api/document.go` 使用 Go `embed` 将所属 `openapi.yaml`
编译进服务二进制。`GET /openapi.yaml` 直接返回这份嵌入合同，Swagger UI
也只读取该地址。运行时不依赖部署主机文件，不复制 YAML，不生成或提交派生
OpenAPI 构建产物。

新增或修改 HTTP API 时，必须在同一变更中更新所属 OpenAPI 文档。CI 至少验证：

- OpenAPI 3.0.4 文档可解析且不存在悬空本地引用。
- 运行时业务路径和 HTTP 方法与合同一致。
- `operationId` 在单个 Service 内唯一且稳定。
- 请求参数、请求体和响应 DTO 的 JSON 字段与 Schema 一致。
- 主要成功与失败状态码和安全声明与 Handler 行为一致。

合同与代码发生漂移时 CI 失败，不以 Swagger UI 能否渲染作为合同正确性的判断。
这些检查作为普通 Go 测试纳入现有 `go test ./...`，不增加独立生成步骤。

### 运维端点

每个 Service 的 OpenAPI 文档包含：

- `GET /healthz`
- `GET /readyz`

两者归入 `Operations` 标签，并参与路由与响应 DTO 防漂移验证。

以下文档交付入口不写入 OpenAPI paths，并从运行时业务路由覆盖检查中明确排除：

- `GET /openapi.yaml`
- `GET /docs`
- `GET /docs/` 及其 Swagger UI 静态资源子路径

## 实施范围

- 为 Data、Miniapp 和 Admin Portal 分别补齐 OpenAPI 3.0.4 合同与合同测试。
- 通过 Go `embed` 和 `swgui v1.8.9` 为三个 Service 提供文档入口。
- 统一三个 Service 的 API 路径、业务成功信封、错误信封和 request ID 行为。
- 更新本仓运行时路由、Service 间 typed client、Miniapp Frontend 调用、
  测试、架构文档和 UAT 脚本。
- 输出 AgentRun 等仓外消费者所需的新旧 Data 路径迁移清单。

## 范围外

- 不新增业务 API。
- 不改变现有业务请求字段、业务结果字段、分页语义或权限 Scope。
- 不修改数据库 Schema 或业务数据。
- 不修改 AgentRun 仓库。
- 不引入 Handler 注解、OpenAPI 代码生成、聚合网关或统一文档门户。
- 不在生产环境注册 OpenAPI 或 Swagger UI 路由。
