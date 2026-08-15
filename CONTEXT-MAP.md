# Context Map

观潮家当前使用一个 repository 和一个根 Go module，按应用垂直组织四个可独立构建、
启动和部署的 Backend Service。源码是否在同一仓库，不改变 Service 之间只能通过远程
API 协作的边界。

## Contexts

| Context      | 类型            | 主要职责                                                 | 详细说明                               |
| ------------ | --------------- | -------------------------------------------------------- | -------------------------------------- |
| Miniapp      | 产品系统        | `miniapp/frontend` 与 `miniapp/backend`                  | `docs/contexts/miniapp/CONTEXT.md`     |
| Admin Portal | 产品系统        | `admin-portal/frontend` 与 `admin-portal/backend`        | `docs/contexts/adminportal/CONTEXT.md` |
| Data         | 领域系统        | `data-service/backend`、数据事实、导入和查询能力 | `docs/contexts/data/CONTEXT.md`        |
| AgentRun     | 遗留 Agent 平台 | `agent-run/backend`、待独立退役的执行与 Artifact 能力    | `docs/contexts/agentrun/CONTEXT.md`    |

`data-service` 是工程应用名，领域术语仍为 Data Domain Service。AgentRun 的既有
运行时、PostgreSQL、migration ledger 与 Artifact 在退役前继续物理隔离，但新的完整
Raw Evidence 与原子 Evidence 已由 Data 拥有；旧 AgentRun 数据的搬迁和运行时下架属于
独立 rollout。也可以新增 User、Payment 等 Domain Service，但不得把领域能力放回没有
owner 的共享业务目录。

## Allowed Dependencies

```text
Miniapp Frontend
  -> Miniapp Application Backend Service
      -> Data Domain Service REST API
      -> future User/Payment Domain Service REST API

Admin Portal Frontend
  -> Admin Application Backend Service
      -> Data Domain Service REST API
      -> AgentRun Admin REST API
      -> future User/Payment Domain Service REST API

AgentOS or legacy AgentRun
  -> Data Domain Service versioned publication/read REST APIs

Data Domain Service
  -> no AgentOS/AgentRun runtime, database, Artifact, or code dependency
```

禁止依赖：

- Frontend 直接调用 Domain Service。
- Application Backend Service 直接访问 Domain Service 数据库或 import 其实现。
- 不同 Backend Service 通过 Go 方法、共享 repository 或共享 domain model 协作。
- 根目录出现被多个应用 import 的共享运行时源码；相似机制必须由各应用独立拥有。

## Canonical Terms

- **Application Backend Service**：面向一个 Frontend 的可部署后端进程。当前为 Miniapp 和 Admin Portal。
- **Domain Service**：拥有领域规则、事实数据和持久化的可部署后端进程。当前为 Data。
- **Use Case Layer**：单个 Service 内部的业务编排层。Kratos Service 使用
  `internal/biz/`；尚未迁移的 Service 可以暂时保留 `usecase/`。二者是同一
  逻辑层的不同工程布局，不是跨 Service 共享层。
- **Platform**：没有业务语义的通用机制，例如配置文件读取、HTTP server/client 基础能力。

## System Decisions

- `docs/adr/0001-product-source-root.md`
- `docs/adr/0002-backend-service-architecture.md`
- `docs/adr/0006-kratos-official-service-layout.md`
- `docs/adr/0007-app-oriented-monorepo.md`
- `docs/adr/0011-data-owns-raw-evidence-and-evidence-publication.md`
- `docs/adr/0012-docker-only-service-runtime.md`
- `docs/adr/0013-data-entity-domain-and-projection-retirement.md`
- `docs/adr/0014-agentrun-consumes-retained-qdrant-snapshot.md`
- `docs/adr/0015-retire-root-shared-test-fixtures.md`
- `docs/adr/0020-local-docker-application-and-infrastructure-projects.md`

## Runtime

All deployable application services and service-owned operational processes run through their
Docker images and Docker Compose. Backend services keep one YAML per application environment;
local/development YAML uses container-reachable external infrastructure endpoints and no
host-runtime configuration tree is maintained. The local developer environment declares its
independently operated PostgreSQL, MySQL, Neo4j, MinIO and Qdrant containers in the
`tidewise-infra` Compose project; they remain outside the `tidewise-app` application Compose and
all UAT/release artifacts. Tidewise services and local middleware use the `tidewise-local` network
and Compose DNS.
Miniapp Frontend is not an HTTP service and is excluded from the Docker runtime contract;
repository-pinned Node/Taro build/watch commands generate `dist` for the host platform developer
tools. Admin Web is the browser's single Admin origin and proxies relative `/api/admin/*` requests
to the internal Admin Backend service.
