# Local PostgreSQL And Neo4j

本目录提供 local PostgreSQL 和 Neo4j 的本地运行说明。这里的模板只用于开发环境，不得复用为 uat 或 prod secret 来源。采集调度与运行由独立 agent-run 项目负责，不在本仓库提供本地 scheduler、source-ingest 或 ingest-smoke 命令。

## 本地 PostgreSQL

如果使用 Docker Compose：

```bash
cp infra/local/.env.example infra/local/.env.local
```

修改 `infra/local/.env.local` 中的本地 password 后启动：

```bash
docker compose --env-file infra/local/.env.local -f infra/local/docker-compose.postgres.yaml up -d
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

Neo4j 是从 PostgreSQL 事实源投影出来的图谱查询库。local 环境默认在 `backend/config/config.local.yaml` 中启用 Neo4j，但真实用户名和密码只通过环境变量注入。

如果使用 Docker Compose：

```bash
cp infra/local/.env.example infra/local/.env.local
```

修改 `infra/local/.env.local` 中的 `NEO4J_USERNAME` 和 `NEO4J_PASSWORD` 后启动：

```bash
docker compose --env-file infra/local/.env.local -f infra/local/docker-compose.neo4j.yaml up -d
```

Neo4j Browser 默认访问：

```text
http://localhost:7474
```

Bolt 连接地址与 local config 对齐：

```text
bolt://localhost:7687
```

图谱投影命令位于 `backend/cmd/graph-projector`。真实 Neo4j smoke 必须显式启用，普通 `go test ./...` 不会连接 Neo4j：

```bash
APP_ENV=local \
DATABASE_PASSWORD=<local-postgres-password> \
NEO4J_USERNAME=<local-neo4j-user> \
NEO4J_PASSWORD=<local-neo4j-password> \
TIDEWISE_ENABLE_NEO4J_SMOKE=true \
go test ./cmd/graph-projector ./internal/platform/graphdb ./internal/apps/graphprojection
```

手动检查连接和投影：

```bash
APP_ENV=local NEO4J_USERNAME=<local-neo4j-user> NEO4J_PASSWORD=<local-neo4j-password> go run ./cmd/graph-projector check
APP_ENV=local DATABASE_PASSWORD=<local-postgres-password> NEO4J_USERNAME=<local-neo4j-user> NEO4J_PASSWORD=<local-neo4j-password> go run ./cmd/graph-projector project-entities
APP_ENV=local DATABASE_PASSWORD=<local-postgres-password> NEO4J_USERNAME=<local-neo4j-user> NEO4J_PASSWORD=<local-neo4j-password> go run ./cmd/graph-projector rebuild-entities
```

`project-entities` 会读取 PostgreSQL 的 `entity_nodes` 和 `entity_edges`，写入 Neo4j 的 `Entity` 标签，并通过 `projection_namespace=tidewise` 标识本系统投影。`rebuild-entities` 只清理该命名空间的实体图，不会清空整个 Neo4j database。

## 执行 migration

在 `backend/` 目录执行：

```bash
APP_ENV=local DATABASE_PASSWORD=<local-password> go run ./cmd/dbmigrate -apply
```

也可以用完整连接串覆盖 host/user/password：

```bash
APP_ENV=local TIDEWISE_DATABASE_URL='postgres://tidewise:<local-password>@localhost:5432/tidewise_local?sslmode=disable' go run ./cmd/dbmigrate -apply
```

检查模式不会修改 schema：

```bash
APP_ENV=local DATABASE_PASSWORD=<local-password> go run ./cmd/dbmigrate
```

## 初始化 source catalog

先确认 migration 已完成，再把 repo 内版本化来源清单写入 `source_catalogs`：

```bash
APP_ENV=local DATABASE_PASSWORD=<local-password> go run ./cmd/source-seed
```

该命令默认读取：

```text
backend/data/source_catalogs/
```

输出 report 应包含 Vibe-Research、Vibe-Trading、Stock 的来源数量、Vibe-Research 108 条配置和 106 个唯一 URL、总来源数量、provider/type/status 分布以及 SDK 排除口径。

可用以下 SQL 快速核验来源目录：

```sql
SELECT COUNT(*) FROM source_catalogs;
SELECT provider_key, COUNT(*) FROM source_catalogs GROUP BY provider_key ORDER BY COUNT(*) DESC, provider_key;
SELECT source_type, COUNT(*) FROM source_catalogs GROUP BY source_type ORDER BY source_type;
SELECT status, COUNT(*) FROM source_catalogs GROUP BY status ORDER BY status;
SELECT origin_system, COUNT(*) FROM source_catalogs GROUP BY origin_system ORDER BY origin_system;
```

## 采集运行边界

本仓库保留 source metadata、connector/parser/sourcecatalog/prompt 代码与 Data Service 的 raw-document/reviewed-event 受控导入合同，但不再拥有采集调度或执行编排。独立 agent-run 项目负责调度、调用 connector/parser，并通过 `/internal/data/v1` 的受认证接口导入结果；不得绕过 Data Service 直接写 Data DB。

历史 scheduler/run 表及 migrations 为审计兼容保留，不代表仍可从 Tidewise 启动 scheduler，也不得通过本说明直接修改这些历史表。

## 运行 Admin API 和管理后台

Admin API 使用 `ADMIN_API_TOKEN` 鉴权。真实 token 只通过环境变量注入，不写入 repo：

```bash
APP_ENV=local ADMIN_API_TOKEN=<local-admin-token> DATA_SERVICE_BASE_URL=http://127.0.0.1:8082 DATA_SERVICE_ADMIN_TOKEN=<local-data-token> go run ./cmd/admin-api
```

管理后台位于：

```text
frontend/admin/
```

首次运行需要安装依赖：

```bash
cd frontend/admin
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

页面右上角输入 `ADMIN_API_TOKEN` 后，可以查询采集源、原始数据和事件。已退役的 scheduler 路由在一个发布窗口内返回认证后的 machine-readable `410 Gone`。

## 常见失败

- `ping postgres`：本地 PostgreSQL 未启动、端口不对、数据库不存在或 password 未注入。
- `pending migrations exist`：当前环境关闭了 `migration.auto_apply`，需要先运行 `dbmigrate -apply`。
- `insert raw document`：通常表示 migration 未执行、source seed 失败或 schema 与 repository 不一致。
- `admin token is not configured`：启动 `cmd/admin-api` 时没有注入 `ADMIN_API_TOKEN`。
