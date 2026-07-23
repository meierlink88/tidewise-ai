# Local Three-Service Stack

本目录只编排 Data、Miniapp、Admin Portal 三个服务，以及 PostgreSQL、Neo4j、共享 network/volumes。这里的模板只用于开发环境，不得复用为 uat 或 prod secret 来源。采集调度与运行由独立 agent-run 项目负责，不在本仓库提供本地 scheduler、source-ingest 或 ingest-smoke 命令。

## 静态检查与服务入口

先创建不提交的本地环境文件：

```bash
cp infra/local/.env.example infra/local/.env.local
```

修改全部 `replace-with-local-*` 占位值后，可在不创建或启动任何容器的情况下检查最终编排：

```bash
docker compose --env-file infra/local/.env.local -f infra/local/docker-compose.yaml config
```

统一编排使用三个 service-owned Dockerfile；默认端口为 Data `9011`、Miniapp `9012`、Admin `9013`、PostgreSQL `5432`、Neo4j Browser `7474`、Neo4j Bolt `7687`。Miniapp/Admin只获得各自Data Service identity token，不携带Data PostgreSQL或Neo4j凭据。

## 本地 PostgreSQL

需要启动local stack时使用统一编排文件：

```bash
docker compose --env-file infra/local/.env.local -f infra/local/docker-compose.yaml up -d
```

如果使用本机已有 PostgreSQL，只要保证 local 配置能连接到：

```text
host: localhost
port: 5432
database: tidewise_local
user: tidewise
```

真实 password 通过环境变量注入，不写入 repo。

## 本地 Neo4j

Neo4j 是从 PostgreSQL 事实源投影出来的图谱查询库。local 环境默认在 `src/backend/services/data/config/config.local.yaml` 中启用 Neo4j，但真实用户名和密码只通过环境变量注入。

Neo4j Browser 默认访问：

```text
http://localhost:7474
```

Bolt 连接地址与 local config 对齐：

```text
bolt://localhost:7687
```

图谱投影命令位于 `src/backend/services/data/cmd/graph-projector`。真实 Neo4j smoke 必须显式启用，普通 `go test ./...` 不会连接 Neo4j：

```bash
APP_ENV=local \
DATABASE_PASSWORD=<local-postgres-password> \
NEO4J_USERNAME=<local-neo4j-user> \
NEO4J_PASSWORD=<local-neo4j-password> \
TIDEWISE_ENABLE_NEO4J_SMOKE=true \
go test ./services/data/cmd/graph-projector ./services/data/adapters/graphdb ./services/data/usecase/graphprojection
```

手动检查连接和投影：

```bash
APP_ENV=local NEO4J_USERNAME=<local-neo4j-user> NEO4J_PASSWORD=<local-neo4j-password> go run ./services/data/cmd/graph-projector check
APP_ENV=local DATABASE_PASSWORD=<local-postgres-password> NEO4J_USERNAME=<local-neo4j-user> NEO4J_PASSWORD=<local-neo4j-password> go run ./services/data/cmd/graph-projector project-entities
APP_ENV=local DATABASE_PASSWORD=<local-postgres-password> NEO4J_USERNAME=<local-neo4j-user> NEO4J_PASSWORD=<local-neo4j-password> go run ./services/data/cmd/graph-projector rebuild-entities
```

`project-entities` 会读取 PostgreSQL 的 `entity_nodes` 和 `entity_edges`，写入 Neo4j 的 `Entity` 标签，并通过 `projection_namespace=tidewise` 标识本系统投影。`rebuild-entities` 只清理该命名空间的实体图，不会清空整个 Neo4j database。

## 执行 migration

在 `src/backend/` 目录执行：

```bash
APP_ENV=local DATABASE_PASSWORD=<local-password> go run ./services/data/cmd/dbmigrate -apply
```

也可以用完整连接串覆盖 host/user/password：

```bash
APP_ENV=local TIDEWISE_DATABASE_URL='postgres://tidewise:<local-password>@localhost:5432/tidewise_local?sslmode=disable' go run ./services/data/cmd/dbmigrate -apply
```

检查模式不会修改 schema：

```bash
APP_ENV=local DATABASE_PASSWORD=<local-password> go run ./services/data/cmd/dbmigrate
```

## 初始化本地 Research Theme

应用 migration 后，可将版本化的首页开发批次通过正式 Research Theme Import Service 写入本地库：

```bash
APP_ENV=local \
TIDEWISE_DATABASE_URL='postgres://tidewise:<local-password>@localhost:5432/tidewise_local?sslmode=disable' \
go run ./services/data/cmd/research-theme-dev-seed
```

该命令只允许连接 `tidewise_local`，默认读取 `src/backend/data/research_themes/local_homepage.json`。文件使用生产 V1 导入合同；命令不会直接 upsert Theme 或清空关联表。首次执行创建不可变 receipt，重复执行返回原结果并标记 `replayed: true`。

## 采集运行边界

Source 主数据、connector、parser、prompt、完整 Markdown Artifact 与采集编排归属独立 AgentRun 项目。Tidewise Data 只通过受认证的 `POST /internal/data/v2/reviewed-event-imports` 原子接纳正式 Event 及其轻量证据记录；AgentRun 不得绕过 Data Service 直接写 Data DB。

历史 Source、scheduler/run 表只存在于旧 migration 历史中；当前 Schema 和运行时不再提供对应控制面。

## 运行 Admin 前端

Admin Portal BFF由统一compose在`9013`提供，并使用`ADMIN_API_TOKEN`鉴权。真实token只通过未提交的`.env.local`注入，不写入repo；本地只允许 `http://127.0.0.1:5174` Origin。

管理后台位于：

```text
src/frontend/admin/
```

首次运行需要安装依赖：

```bash
npm install
```

本地启动：

```bash
npm run dev -- --port 5174
```

默认访问：

```text
http://127.0.0.1:5174/
```

页面右上角输入 `ADMIN_API_TOKEN` 后，可以查询采集源、原始数据和事件。已退役的 scheduler 路由已经删除。

## 常见失败

- `ping postgres`：本地 PostgreSQL 未启动、端口不对、数据库不存在或 password 未注入。
- `pending migrations exist`：当前环境关闭了 `migration.auto_apply`，需要先运行 `dbmigrate -apply`。
- `insert raw document`：通常表示 migration 未执行、source seed 失败或 schema 与 repository 不一致。
- `admin token is not configured`：启动 `services/adminportal/cmd` 时没有注入 `ADMIN_API_TOKEN`。
