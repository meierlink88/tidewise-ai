# Context Map

观潮家当前使用一个 repository 和一个根 Go module，按应用垂直组织三个 Context、四个可
独立构建和启动的应用服务。源码是否在同一仓库，不改变 Service 之间只能通过远程 API
协作的边界。

## Contexts

| Context      | 类型     | 主要职责                                          | 详细说明                               |
| ------------ | -------- | ------------------------------------------------- | -------------------------------------- |
| Miniapp      | 产品系统 | `miniapp/frontend` 与 `miniapp/backend`           | `docs/contexts/miniapp/CONTEXT.md`     |
| Admin Portal | 产品系统 | `admin-portal/frontend` 与 `admin-portal/backend` | `docs/contexts/adminportal/CONTEXT.md` |
| Data         | 领域系统 | `data-service/backend`、正式事实与查询            | `docs/contexts/data/CONTEXT.md`        |

`data-service` 是工程应用名，领域术语仍为 Data Domain Service。Agent OS 位于本仓库之外，
需要发布数据时只能消费 Data 的版本化 REST API，不属于本仓库 Context、部署单元或数据
所有权边界。

## Allowed Dependencies

```text
Miniapp Frontend
  -> Miniapp Application Backend Service
      -> Data Domain Service REST API

Admin Portal Frontend
  -> Admin Application Backend Service
      -> Data Domain Service REST API, including Source management

External Agent OS
  -> Data Domain Service versioned Source/Company snapshots and publication/read REST APIs

Data Domain Service
  -> no external Agent runtime, database, Artifact, or code dependency
```

禁止 Frontend 直连 Domain Service、Application Backend 访问下游数据库、不同 Service
通过 Go 方法或共享 repository 协作，以及在根目录增加多应用 import 的共享运行时源码。

## Runtime

四个应用服务为 Data Service、Miniapp Backend、Admin Backend 与 Admin Web。它们通过
Docker image 和 Compose 运行。本地共享 PostgreSQL 与 MinIO 属于独立基础设施项目，
不属于应用发布单元。本地推理专用 Neo4j 由外部 `tidewise-reason`
repository 独立拥有，四个应用服务不依赖它。Admin Web 是浏览器唯一 Admin origin，并把相对
`/api/admin/*` 请求代理到内部 Admin Backend。

系统级退役决策见 `docs/adr/0027-retire-agent-run.md`。Source 所有权、Admin Backend 管理边界与
AgentOS 运行时快照见 `docs/adr/0031-data-owns-source-management.md`。
Company 投影只能通过 Data 版本化快照 API，不得直连 PostgreSQL；该边界见
`docs/adr/0050-expose-company-projection-snapshot.md`。

UAT ECS 只保留四个应用服务与历史 `tidewise-infra-uat` MinIO/raw-evidence 存储；历史链接
继续沿用既有访问方式。新 Raw Evidence 文档由 DGX AgentOS 自有 MinIO 归档，Data Service
只保存不含 origin 的相对路径，不访问该 MinIO；是否提供浏览器可达 origin 不属于 Data 或
Admin Portal 合同。
AgentOS、Reason/OpenSPG、KAG、MySQL、Neo4j 与 Qdrant 已迁出或退役，不属于当前 ECS、RDS
或本仓库 UAT 生命周期；详见 `docs/adr/0055-retire-uat-reasoning-runtime.md`。
