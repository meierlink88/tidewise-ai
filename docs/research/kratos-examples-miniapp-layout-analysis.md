# Kratos Examples 与 Miniapp Service 目录研究

日期：2026-07-25

## 结论

Kratos 官方材料支持以下判断：

1. `go-kratos/examples` 是按技术主题组织的混合示例库，不是一个可以整体复制的业务 Monorepo 模板。
2. 完整 Kratos Application 的稳定骨架是 `api`、`cmd`、`configs` 和
   `internal/{conf,biz,data,service,server}`。`internal` 不是一个额外业务层，而是包住五个实现层的
   Go 私有边界。
3. `examples` 中的 `http/*`、`swagger`、`auth/jwt`、`registry/*` 等只演示单个组件，不能用来否定
   或替代完整 Application Layout。
4. Tidewise 应把 `miniapp/backend` 视为一个完整 Kratos Application 根目录，在当前
   Monorepo 中保持独立二进制、配置、镜像和私有实现边界；不需要为了采用 Kratos 立即拆成独立仓库。
5. Tidewise 采用官方分层职责，但不采用官方模板中的 Protobuf API/配置、gRPC 和 Wire。Miniapp
   继续保持 HTTP-only、OpenAPI 3.0.4-first、显式构造，并保留 Swagger UI。

## 调查范围与版本

本次只使用 Kratos 第一方材料：

- [`go-kratos/examples` README](https://github.com/go-kratos/examples/blob/61daed1ec4d5a94d689bc8fab9bc960c6af73ead/README.md)
- [`go-kratos/examples` 调查快照](https://github.com/go-kratos/examples/tree/61daed1ec4d5a94d689bc8fab9bc960c6af73ead)
- [Kratos 官方 Layout 文档](https://go-kratos.dev/docs/intro/layout/)
- [Kratos 官方 Design 文档](https://go-kratos.dev/docs/intro/design/)
- [Kratos 官方 Usage 文档中的大仓模式](https://go-kratos.dev/docs/getting-started/usage/)
- [`go-kratos/kratos-layout` V3 调查快照](https://github.com/go-kratos/kratos-layout/tree/94dbfcc4264a6be8e7b6c4929923c1e1f738b980)

需要特别注意版本差异：

- `examples` 根 [`go.mod`](https://github.com/go-kratos/examples/blob/61daed1ec4d5a94d689bc8fab9bc960c6af73ead/go.mod)
  仍是 Go 1.16 和 Kratos v2.8.0，适合研究组件用法和分层落位，不应复制其中的 `/v2` API。
- `kratos-layout` 调查快照的
  [`go.mod`](https://github.com/go-kratos/kratos-layout/blob/94dbfcc4264a6be8e7b6c4929923c1e1f738b980/go.mod)
  使用 Go 1.25.7 和 Kratos v3.0.0，是本次目录和 V3 生命周期判断的更高优先级依据。

## 1. Examples 顶层如何组织

`examples` 顶层按“要演示什么”划分，而不是按统一业务系统划分：

- 单组件或单传输示例：`auth`、`config`、`errors`、`helloworld`、`http`、`metadata`、
  `metrics`、`middleware`、`registry`、`selector`、`swagger`、`tls`、`validate`、`ws`。
- 较完整的分层应用：[`blog`](https://github.com/go-kratos/examples/tree/61daed1ec4d5a94d689bc8fab9bc960c6af73ead/blog)、
  [`i18n`](https://github.com/go-kratos/examples/tree/61daed1ec4d5a94d689bc8fab9bc960c6af73ead/i18n)。
- 同一能力的不同基础设施实现：
  [`transaction/{ent,gorm,mongo,sqlc}`](https://github.com/go-kratos/examples/tree/61daed1ec4d5a94d689bc8fab9bc960c6af73ead/transaction)。
- 包含多个 Application 的方案级示例：
  [`cqrs`](https://github.com/go-kratos/examples/tree/61daed1ec4d5a94d689bc8fab9bc960c6af73ead/cqrs)、
  `casbin`、`chatroom`、`realtimemap`。

仓库根部有一个共享 `go.mod`，但 `casbin`、`chatroom`、`cqrs`、`grpc-vite/server`、`otel`、
`realtimemap` 又拥有自己的 `go.mod`。这说明该仓库优先服务于示例独立性，并不要求所有示例使用一种
模块组织方式。

`cqrs` 进一步证明一个仓库可以包含多个可部署 Application：例如
[`app/logger/service`](https://github.com/go-kratos/examples/tree/61daed1ec4d5a94d689bc8fab9bc960c6af73ead/cqrs/app/logger/service)
和 `app/logger/job` 分别拥有自己的 `cmd`、`configs` 和 `internal`。因此“独立 Service”与“独立
Repository”不是同一个决定。

Kratos 官方 Usage 还明确提供 `kratos new app/user --nomod` 的“大仓模式”。Tidewise 当前共享一个
Backend `go.mod`、在 `services/` 下放置多个 Application，与这个方向兼容。

## 2. 完整 Application 中各目录的真实放置

### 官方 V3 Layout

当前 `kratos-layout` 是最直接的完整 V3 Application 依据：

```text
.
├── api/todo/v1/
├── cmd/server/
├── configs/
└── internal/
    ├── conf/
    ├── biz/
    ├── data/
    ├── service/
    └── server/
```

具体第一方实现：

| 目录 | 官方真实内容 | 目录职责 |
|---|---|---|
| [`api/todo/v1`](https://github.com/go-kratos/kratos-layout/tree/94dbfcc4264a6be8e7b6c4929923c1e1f738b980/api/todo/v1) | Proto API 和生成的 HTTP/gRPC binding | 进程对外 API 合同 |
| [`cmd/server`](https://github.com/go-kratos/kratos-layout/tree/94dbfcc4264a6be8e7b6c4929923c1e1f738b980/cmd/server) | 配置加载、logger、`kratos.App` 和依赖装配 | Application composition root |
| [`configs`](https://github.com/go-kratos/kratos-layout/tree/94dbfcc4264a6be8e7b6c4929923c1e1f738b980/configs) | 本地运行配置 | 配置样例，不承载业务逻辑 |
| [`internal/conf`](https://github.com/go-kratos/kratos-layout/tree/94dbfcc4264a6be8e7b6c4929923c1e1f738b980/internal/conf) | 配置结构 | 配置解析所需的私有类型 |
| [`internal/biz`](https://github.com/go-kratos/kratos-layout/tree/94dbfcc4264a6be8e7b6c4929923c1e1f738b980/internal/biz) | 业务对象、错误、Use Case、Repository Port | 领域规则与依赖倒置边界 |
| [`internal/data`](https://github.com/go-kratos/kratos-layout/tree/94dbfcc4264a6be8e7b6c4929923c1e1f738b980/internal/data) | Repository 实现和资源生命周期 | 数据库、缓存或远端服务 Adapter |
| [`internal/service`](https://github.com/go-kratos/kratos-layout/tree/94dbfcc4264a6be8e7b6c4929923c1e1f738b980/internal/service) | API 实现、DTO 与 Biz 转换 | Transport-facing Application Service |
| [`internal/server`](https://github.com/go-kratos/kratos-layout/tree/94dbfcc4264a6be8e7b6c4929923c1e1f738b980/internal/server) | HTTP/gRPC Server 创建及 API 注册 | 传输服务器和中间件组装 |

官方 [`internal/biz/todo.go`](https://github.com/go-kratos/kratos-layout/blob/94dbfcc4264a6be8e7b6c4929923c1e1f738b980/internal/biz/todo.go)
把 `TodoRepo` 接口定义在 Biz；`internal/data/todo.go` 实现该接口；`internal/service/todo.go`
完成 API DTO 与 Biz 对象转换；`internal/server/http.go` 创建 Kratos HTTP Server 并注册 API。这是
本次 Miniapp 改造应保留的核心依赖方向：

```text
api <- internal/service -> internal/biz <- internal/data
 ^             ^
 └── internal/server

cmd/server -> conf + data + biz + service + server + kratos.App
```

### Blog 完整业务示例

`examples/blog` 虽然仍使用 Kratos v2，但展示了比 Todo 内存仓储更接近真实业务的落位：

- [`api/blog/v1`](https://github.com/go-kratos/examples/tree/61daed1ec4d5a94d689bc8fab9bc960c6af73ead/blog/api/blog/v1)
  放 API 定义和生成 binding。
- [`internal/biz/article.go`](https://github.com/go-kratos/examples/blob/61daed1ec4d5a94d689bc8fab9bc960c6af73ead/blog/internal/biz/article.go)
  放 `Article`、`ArticleRepo` 和 `ArticleUsecase`。
- [`internal/data/article.go`](https://github.com/go-kratos/examples/blob/61daed1ec4d5a94d689bc8fab9bc960c6af73ead/blog/internal/data/article.go)
  实现 Biz 拥有的 Port，并把持久化对象转换成 Biz 对象。
- [`internal/service/article.go`](https://github.com/go-kratos/examples/blob/61daed1ec4d5a94d689bc8fab9bc960c6af73ead/blog/internal/service/article.go)
  实现 API 方法并调用 Use Case。
- [`internal/server/http.go`](https://github.com/go-kratos/examples/blob/61daed1ec4d5a94d689bc8fab9bc960c6af73ead/blog/internal/server/http.go)
  创建 HTTP Server、组合 Middleware 并注册服务。
- [`cmd/blog/main.go`](https://github.com/go-kratos/examples/blob/61daed1ec4d5a94d689bc8fab9bc960c6af73ead/blog/cmd/blog/main.go)
  加载配置并运行 `kratos.App`。

`cmd/blog` 与当前模板的 `cmd/server` 名称不同，但职责相同。Tidewise 统一选择 `cmd/server`，避免三个
Backend Service 产生不必要的入口命名差异。

### Transaction 与 CQRS

[`transaction/sqlc`](https://github.com/go-kratos/examples/tree/61daed1ec4d5a94d689bc8fab9bc960c6af73ead/transaction/sqlc)
仍使用 `cmd`、`configs`、`internal/conf|biz|data|service|server`，SQLC 生成物属于 `internal/data`，
而不是 Biz 或 Service。

[`cqrs/app/logger/service`](https://github.com/go-kratos/examples/tree/61daed1ec4d5a94d689bc8fab9bc960c6af73ead/cqrs/app/logger/service)
展示了方案级 Monorepo 内的独立 Application。它可以支持 Tidewise 保持单仓的判断，但其中 Kafka、
Consul、gRPC 和远程配置只是 CQRS 示例自己的需求，不能反推为 Miniapp 的必要组件。

## 3. 哪些 Examples 不是完整 Application Layout

以下目录主要回答某个 Kratos API“怎么用”，不回答完整服务“怎么分层”：

- [`helloworld`](https://github.com/go-kratos/examples/tree/61daed1ec4d5a94d689bc8fab9bc960c6af73ead/helloworld)：
  只有简单 client/server 和生成合同。
- [`http/*`](https://github.com/go-kratos/examples/tree/61daed1ec4d5a94d689bc8fab9bc960c6af73ead/http)：
  分别演示 CORS、下载、Gin、Mux、PProf、重定向、Session、静态文件、上传等传输能力。
- [`auth/jwt`](https://github.com/go-kratos/examples/tree/61daed1ec4d5a94d689bc8fab9bc960c6af73ead/auth/jwt)：
  只演示 JWT Middleware。
- [`config`](https://github.com/go-kratos/examples/tree/61daed1ec4d5a94d689bc8fab9bc960c6af73ead/config)：
  只演示配置源。
- [`errors`](https://github.com/go-kratos/examples/tree/61daed1ec4d5a94d689bc8fab9bc960c6af73ead/errors)：
  只演示 Kratos 错误。
- [`registry`](https://github.com/go-kratos/examples/tree/61daed1ec4d5a94d689bc8fab9bc960c6af73ead/registry)：
  分别演示 Consul、etcd、Nacos、Zookeeper 等注册发现实现。
- [`swagger`](https://github.com/go-kratos/examples/tree/61daed1ec4d5a94d689bc8fab9bc960c6af73ead/swagger)：
  只演示嵌入 Swagger API。
- `metadata`、`metrics`、`middleware`、`selector`、`sse`、`stream`、`tls`、`validate`、`ws`
  同样属于局部能力演示。

这些示例可以帮助实现 `internal/server` 或 `internal/data` 内的一项机制，但不能直接复制为 Tidewise
Miniapp 的整体目录。尤其不能因为 `http/gin` 或 `swagger/server` 只有一个 `main.go`，就让正式
Miniapp 绕过 Biz、Data、Service 分层。

## 4. Tidewise Miniapp 的可迁移目录建议

### 推荐目录

```text
miniapp/backend/
├── Dockerfile
├── api/
│   └── miniapp/
│       └── v1/
│           ├── openapi.yaml
│           ├── document.go
│           ├── contract.go
│           ├── http.go
│           └── *_test.go
├── cmd/
│   └── server/
│       ├── main.go
│       ├── app.go
│       └── app_test.go
├── configs/
│   ├── config.local.yaml
│   └── config.uat.yaml
└── internal/
    ├── conf/
    │   ├── config.go
    │   └── config_test.go
    ├── biz/
    │   ├── research.go
    │   ├── reasoning_tree.go
    │   ├── research_repo.go
    │   ├── errors.go
    │   └── *_test.go
    ├── data/
    │   ├── data.go
    │   ├── research_http.go
    │   └── research_http_test.go
    ├── service/
    │   ├── service.go
    │   ├── research.go
    │   ├── reasoning_tree.go
    │   └── *_test.go
    └── server/
        ├── http.go
        ├── middleware.go
        ├── docs.go
        └── *_test.go
```

`miniapp/backend` 是一个完整 Application 根目录；`internal` 包含其五个私有实现层。未来需要独立
仓库时，可以整体移动该目录，而不需要重新拆分业务层。

### 业务组件归属

| Miniapp 能力 | 目标目录 | 原因 |
|---|---|---|
| OpenAPI 3.0.4、HTTP DTO、API interface、Kratos 路由绑定 | `api/miniapp/v1` | 对外线协议和薄 binding |
| Theme 列表/详情与推理树的业务对象、校验、Use Case、Research Port | `internal/biz` | Miniapp 产品规则及 consumer-owned Port |
| Data Service HTTP Client、鉴权、超时、重试、响应限制、上游 DTO 转换 | `internal/data` | 实现 Biz Port 的远端 Adapter |
| API DTO 与 Biz 对象转换、调用 Use Case、错误分类交接 | `internal/service` | API-facing Application Service |
| Kratos HTTP Server、Request ID、Recovery、日志、Envelope、404/405、健康检查、Swagger UI | `internal/server` | 传输服务器和横切机制组装 |
| YAML/env 加载、默认值和启动前校验 | `configs` + `internal/conf` | 配置样例与私有配置逻辑分离 |
| 显式构造、`kratos.App`、运行、优雅停机和 cleanup | `cmd/server` | 唯一 composition root |

### 当前目录迁移映射

| 当前目录/文件 | 迁移目标 |
|---|---|
| `api/openapi.yaml`、`api/document.go` | `api/miniapp/v1` |
| `cmd/main.go`、根 `service.go` | `cmd/server/main.go`、`cmd/server/app.go` |
| `config/*.yaml` | `configs/` |
| `config/config.go` | `internal/conf` |
| `usecase/*` | `internal/biz` |
| `dataclient/port.go` 中的 Research Port 和业务模型 | `internal/biz` |
| `dataclient/http.go` 及上游 wire DTO | `internal/data` |
| `transport` 中 path/query 绑定和 HTTP DTO | `api/miniapp/v1` |
| `transport` 中 API↔Use Case 转换 | `internal/service` |
| `transport` 中 router、middleware、health、docs、response | `internal/server` |

迁移完成后不保留 `config`、`dataclient`、`usecase`、`transport` 或根 `service.go` 兼容目录，避免同一
职责出现两套共同语言。

### Monorepo 选择

当前继续使用：

```text
.
├── miniapp/
│   └── backend/
├── admin-portal/
│   └── backend/
└── analyse-data-service/
    └── backend/
```

每个目录未来都采用完整 Application Layout，分别构建二进制和镜像。三个服务共享 Backend
`go.mod` 不会破坏 `internal` 隔离：Go 编译器只允许 `miniapp/backend` 目录树内的包导入
`miniapp/backend/internal/*`，兄弟 Service 不能直接导入。

官方大仓示例倾向把 API 放在仓库公共根目录；Tidewise 则把 API 放回每个 Service 根目录，以满足独立
部署和未来整目录拆仓的目标。Kratos 官方 Design 文档明确 Layout 是推荐模板而不是框架强制，因此这项
调整保留了官方分层和依赖方向，不构成偏离 Kratos 设计理念。

## 5. 本项目明确不采用的官方惯例

官方 Layout 同时展示了若干可选技术。它们不是目录分层成立的前提。

| 官方模板惯例 | Tidewise Miniapp 决定 |
|---|---|
| Protobuf 定义 API并生成 HTTP/gRPC binding | 不采用；保留 OpenAPI 3.0.4-first 和手写薄 binding |
| Protobuf 定义配置并生成 `conf.pb.go` | 不采用；`internal/conf` 使用手写强类型配置 |
| 同时创建 HTTP 和 gRPC Server | 不采用；Miniapp 只创建 Kratos HTTP Server |
| Wire `ProviderSet`、`wire.go`、`wire_gen.go` | 不采用；`cmd/server` 使用普通构造函数显式装配 |
| `third_party` Proto 依赖目录 | 不采用；当前无 Proto 编译链 |
| 注册中心、服务发现、远程配置 | 不采用；Miniapp 使用固定 Data Service URL 和本地 YAML/env |
| 官方生成 OpenAPI 放在 Application 根目录 | 调整为 `api/miniapp/v1/openapi.yaml`，由该版本 API 与 Swagger UI 共用 |
| 官方默认错误 JSON | 不直接采用；继续保持 Tidewise Envelope、状态码和 Request ID 合同 |

Kratos v3 仍负责 Application 生命周期、HTTP Server/Router/Client、Middleware 和错误基础能力；不采用
上述可选惯例，不等于只把目录改名或绕过 Kratos Runtime。

## 最终建议

本次 Miniapp 改造应以 `kratos-layout` V3 为结构基线，以 `examples/blog` 验证真实业务分层，以
`examples` 的组件目录按需学习单项能力。目录决定冻结为：

```text
miniapp/backend/
├── api/miniapp/v1
├── cmd/server
├── configs
└── internal/{conf,biz,data,service,server}
```

这既符合 Kratos 官方 Application Layout 的核心理念，也满足 Tidewise 已冻结的 HTTP/OpenAPI、显式
依赖构造、Swagger UI、服务独立部署和未来可拆仓要求。
