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
  -> Data Domain Service versioned Source snapshot and publication/read REST APIs

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

UAT 中的 OpenSPG MySQL 与共享 MinIO 由独立 `tidewise-infra-uat` 项目运行，不属于四个
应用服务、AgentOS 或 Reason Server 的发布事务；详见
`docs/adr/0029-independent-uat-mysql-and-minio.md`。Huawei RDS、主机级 Neo4j 和独立 Qdrant
保持独立 ownership 与生命周期。其中专用 UAT Neo4j 的版本、插件、认证、备份/清理、
网络暴露与基础设施验证由本仓库统一管理；Reason Server 只消费且验证该合同，详见
`docs/adr/0030-own-uat-neo4j-for-reason.md`。
