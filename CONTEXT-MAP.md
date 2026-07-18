# Context Map

观潮家当前使用一个 repository 和一个 Go module，包含三个可独立构建、启动和部署的 Backend Service。源码是否在同一仓库，不改变 Service 之间只能通过远程 API 协作的边界。

## Contexts

| Context | 类型 | 主要职责 | 详细说明 |
|---|---|---|---|
| Miniapp | 产品系统 | Miniapp Frontend 与 Miniapp Application Backend Service | `docs/contexts/miniapp/CONTEXT.md` |
| Admin Portal | 产品系统 | Admin Portal Frontend 与 Admin Application Backend Service | `docs/contexts/adminportal/CONTEXT.md` |
| Data | 领域系统 | Data Domain Service、数据事实、导入和查询能力 | `docs/contexts/data/CONTEXT.md` |

未来可以新增 User、Payment 等 Domain Service，但不得把领域能力放回没有 owner 的共享业务目录。

## Allowed Dependencies

```text
Miniapp Frontend
  -> Miniapp Application Backend Service
      -> Data Domain Service REST API
      -> future User/Payment Domain Service REST API

Admin Portal Frontend
  -> Admin Application Backend Service
      -> Data Domain Service REST API
      -> future User/Payment Domain Service REST API
```

禁止依赖：

- Frontend 直接调用 Domain Service。
- Application Backend Service 直接访问 Domain Service 数据库或 import 其实现。
- 不同 Backend Service 通过 Go 方法、共享 repository 或共享 domain model 协作。
- `internal/platform` import 任一 Service 的业务代码。

## Canonical Terms

- **Application Backend Service**：面向一个 Frontend 的可部署后端进程。当前为 Miniapp 和 Admin Portal。
- **Domain Service**：拥有领域规则、事实数据和持久化的可部署后端进程。当前为 Data。
- **Use Case Layer**：单个 Service 内部的业务编排层。源码目录统一使用 `usecase/`，避免与 Application Backend Service 混淆。
- **Platform**：没有业务语义的通用机制，例如配置文件读取、HTTP server/client 基础能力。

## System Decisions

- `docs/adr/0001-product-source-root.md`
- `docs/adr/0002-backend-service-architecture.md`
