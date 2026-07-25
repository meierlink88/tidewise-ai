---
status: accepted
date: 2026-07-25
---

# 可部署 Go Service 采用 Kratos 官方 Application Layout

## 背景

Tidewise AI 当前在一个 repository 和一个根 Go module 内维护 Data、Miniapp、
Admin Portal 三个可独立部署的 Backend Service。ADR-0007 将它们分别放在
`analyse-data-service/backend`、`miniapp/backend` 与 `admin-portal/backend`。原有
`cmd/usecase/transport/dataclient/config` 顶层布局能够表达服务 ownership，
但没有完整采用 Kratos 的 Application 组织方式，进程生命周期、HTTP Server、
Router 和 Client 仍由自定义 `net/http`/Gin 代码承担。

本决策首先在 Miniapp Service 落地，随后应用于 Admin Portal。Data 仍保持现状，
直到其独立迁移任务完成。

## 决策

每个迁移后的可部署 Go Service 在自己的 service 根目录内采用完整 Kratos
Application Layout：

```text
<application>/backend/
├── api/<service>/v1/
├── cmd/server/
├── configs/
└── internal/
    ├── conf/
    ├── biz/
    ├── data/
    ├── service/
    └── server/
```

职责固定为：

- `api/<service>/v1`：OpenAPI、wire DTO、Service interface 与 HTTP 绑定；
- `cmd/server`：显式构造依赖并运行 `kratos.App`；
- `configs` 与 `internal/conf`：非敏感配置样例、配置类型、加载与校验；
- `internal/biz`：Use Case、业务不变量和 consumer-owned Port；
- `internal/data`：Port 的外部系统适配器；
- `internal/service`：API 与 Biz 之间的转换和 API interface 实现；
- `internal/server`：Kratos HTTP Server、Router、Middleware、Filter 和文档交付。

服务之间仍只通过版本化 REST API 协作。当前保持：

- 单 repository、单 Go module；
- HTTP-only、OpenAPI-first；
- 显式 Go 构造函数，不使用 Wire；
- 固定服务 URL，不引入注册中心；
- 不引入 gRPC、Protobuf、Kubernetes、远程配置中心或 Service Mesh。

当前固定使用 Go 1.25 与 Kratos v3.0.0。Miniapp 与 Admin Portal 二进制依赖闭包
均不得包含 Gin；Admin Portal 完成迁移后根 `go.mod` 不再保留 Gin 直接依赖。

## 取舍

选择“每个 Service 都是完整 Kratos Application”，而不是在仓库根部建立一套
共享 `api/biz/data/service/server`：

- 服务边界在未来拆仓时可以原样搬迁；
- 同名 Kratos 层不会混合不同服务的业务；
- 单仓内会存在少量相似装配代码，但避免过早创建跨服务框架层。

选择手写 HTTP 绑定而不是 Proto/gRPC 代码生成：

- 保持现有 OpenAPI 3.0.4 为 HTTP 合同权威；
- 避免一次框架迁移同时改变协议工具链；
- 以后若确有 gRPC 需求，可在独立决策中引入。

## 影响

- Miniapp 与 Admin Portal 的旧顶层自定义分层和根装配文件被完整替代，不保留
  Gin/Kratos 双栈。
- 两个 Application Backend 的现有路径、DTO、状态码、错误语义、Request ID、
  OpenAPI 和 Swagger UI 行为保持不变。
- Data 是否迁移由后续独立任务决定；本 ADR 不授权顺带修改其业务结构。
- 架构测试必须验证 Kratos 目录存在、服务间无实现依赖，以及 Miniapp/Admin Portal
  binary closure 不包含 Gin。
