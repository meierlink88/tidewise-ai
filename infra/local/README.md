# Local Application Stack

本目录编排 Data、Miniapp、Admin Portal、AgentRun 四个后端服务，以及 Admin 前端、PostgreSQL、Neo4j 和共享 network/volumes。这里的模板只用于开发环境，不得复用为 uat 或 prod secret 来源。AgentRun 在单仓中保持独立数据库、Artifact 卷与 API 边界，不与 Data Service 共享表。

## 静态检查与服务入口

先创建不提交的本地环境文件：

```bash
cp infra/local/.env.example infra/local/.env.local
```

修改全部 `replace-with-local-*` 占位值后，可在不创建或启动任何容器的情况下检查最终编排：

```bash
docker compose --env-file infra/local/.env.local -f infra/local/docker-compose.yaml config
```

统一编排使用四个 backend service-owned Dockerfile；默认端口为 Data `9011`、Miniapp `9012`、Admin `9013`、AgentRun `9080`、PostgreSQL `5432`、Qdrant `6333/6334`、Neo4j Browser `7474`、Neo4j Bolt `7687`。Miniapp/Admin 只获得各自下游 Service identity token，不携带 Data 或 AgentRun 的数据库凭据。

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
Data、AgentRun 和 Admin Portal Backend 分别只使用一个服务身份令牌：
`DATA_SERVICE_TOKEN`、`AGENTRUN_SERVICE_TOKEN`、`ADMIN_SERVICE_TOKEN`。数据库密码使用
`TIDEWISW_DB_PASSWORD` 与 `AGENTRUN_DB_PASSWORD`，容器内外名称保持一致。

## 本地 Neo4j

Neo4j 是 PostgreSQL 事实未来可重建的图谱查询库。local Compose 保留独立 Neo4j
服务和持久化卷，但 Data Server 不连接它、不等待其健康，也不接收其凭据。Neo4j
用户名和密码只用于 Neo4j 容器本身，通过本地 `NEO4J_USERNAME` 和
`NEO4J_PASSWORD` 环境变量注入。

Neo4j Browser 默认访问：

```text
http://localhost:7474
```

从宿主机访问 Bolt：

```text
bolt://localhost:7687
```

旧的通用实体图投影规则仍然退役，不得恢复或手工复制 PostgreSQL 事实。产业链关系
V1 使用独立、受支持的 local-only projector；PostgreSQL 是事实源，Neo4j 只是可重建
查询投影。

先执行只读核验：

```bash
APP_ENV=local \
TIDEWISW_DB_PASSWORD=<local-postgres-password> \
NEO4J_USERNAME=<local-neo4j-user> \
NEO4J_PASSWORD=<local-neo4j-password> \
NEO4J_URI=bolt://localhost:7687 \
NEO4J_DATABASE=neo4j \
go run ./analyse-data-service/backend/cmd/industry-graph-projector \
  -expected-sha256 7c737410ac6af562af19f8b9dad9e8e1c802f8f782625bd360bb2e8f20768608 \
  -dry-run
```

确认 PostgreSQL 与冻结 CSV 的语义指纹一致后，原子替换固定命名空间：

```bash
APP_ENV=local \
TIDEWISW_DB_PASSWORD=<local-postgres-password> \
NEO4J_USERNAME=<local-neo4j-user> \
NEO4J_PASSWORD=<local-neo4j-password> \
NEO4J_URI=bolt://localhost:7687 \
NEO4J_DATABASE=neo4j \
go run ./analyse-data-service/backend/cmd/industry-graph-projector \
  -expected-sha256 7c737410ac6af562af19f8b9dad9e8e1c802f8f782625bd360bb2e8f20768608 \
  -apply -allow-env local
```

命令只允许 loopback `tidewise_local` PostgreSQL、loopback Neo4j 和数据库 `neo4j`。
固定命名空间为 `tidewise-industry-v1`；历史 `projection_namespace=tidewise` 图不会被
删除。首次成功结果应为 4,449 个节点和 7,867 条关系；同包再次执行应返回
`unchanged: true`。

最小独立验收：

```cypher
MATCH (n:TidewiseEntity {projection_namespace: 'tidewise-industry-v1'})
RETURN n.entity_type, count(*) ORDER BY n.entity_type;

MATCH (:TidewiseEntity {projection_namespace: 'tidewise-industry-v1'})
      -[r]->
      (:TidewiseEntity {projection_namespace: 'tidewise-industry-v1'})
WHERE r.projection_namespace = 'tidewise-industry-v1'
RETURN type(r), count(*) ORDER BY type(r);

MATCH (n:TidewiseEntity {projection_namespace: 'tidewise-industry-v1'})
WHERE NOT (n)--()
RETURN count(n); // 必须为 0
```

完整合同、类型分布和事务门禁见
`docs/architecture/local-industry-graph-projection-v1.md`。

## 执行 migration

在仓库根目录执行：

```bash
APP_ENV=local TIDEWISW_DB_PASSWORD=<local-password> go run ./analyse-data-service/backend/cmd/dbmigrate -apply
```

检查模式不会修改 schema：

```bash
APP_ENV=local TIDEWISW_DB_PASSWORD=<local-password> go run ./analyse-data-service/backend/cmd/dbmigrate
```

host、port、database、user 和 SSL 均由 `analyse-data-service/backend/configs/config.local.yaml`
提供，运行时不接受完整数据库 URL 覆盖。宿主机配置使用 `localhost`；Local Compose
显式选择镜像内 `configs/compose/config.local.yaml`，并通过 Docker DNS 使用 `postgres`。

## 初始化本地 Research Theme

应用 migration 后，可将版本化的首页开发批次通过正式 Research Theme Import Service 写入本地库：

```bash
APP_ENV=local \
TIDEWISW_DB_PASSWORD=<local-password> \
go run ./analyse-data-service/backend/cmd/research-theme-dev-seed
```

该命令只允许连接 `tidewise_local`，默认读取 `analyse-data-service/backend/data/research_themes/local_homepage.json`。文件使用生产 V1 导入合同；命令不会直接 upsert Theme 或清空关联表。首次执行创建不可变 receipt，重复执行返回原结果并标记 `replayed: true`。

## AgentRun

AgentRun 源码位于 `agent-run/backend/`，与其他 Go 服务共享根 `go.mod`，但拥有独立 Context、配置、PostgreSQL database 和 Artifact 目录。统一 Compose 会先幂等创建 AgentRun 的数据库身份、应用 AgentRun migration，再启动服务。

单独运行代码：

```bash
npm run backend:dev:agentrun
```

只读检查或应用 AgentRun migration：

```bash
APP_ENV=dev AGENTRUN_DB_PASSWORD=<password> \
go run ./agent-run/backend/cmd/migrate --check-only

APP_ENV=dev AGENTRUN_DB_PASSWORD=<password> \
go run ./agent-run/backend/cmd/migrate
```

完整的配置和 Artifact 运维命令见 `agent-run/backend/README.md`。

### Event Semantic Qdrant 投影

PostgreSQL 是唯一事实源；Qdrant 只保存 Data-owned 可重建语义投影。先应用 Data
migration，再从 Data Service 所有的 CLI 显式全量重建：

```bash
APP_ENV=local TIDEWISE_CONFIG_DIR=configs \
TIDEWISW_DB_PASSWORD=<local-postgres-password> \
QDRANT_URL=http://127.0.0.1:6333 \
EMBEDDING_BASE_URL=https://dashscope.aliyuncs.com/compatible-mode/v1 \
EMBEDDING_API_KEY=<runtime-dashscope-key> \
go run ./cmd/event-semantic-projector -apply -allow-env local
```

该命令在 `analyse-data-service/backend/` 中执行，固定重建
`entity_semantic_v1` 和 `variable_definition_semantic_v1`，使用
`text-embedding-v4 / 1024 / cosine`。AgentRun 仅通过 Eino Embedder 生成查询向量并直接批量
查 Qdrant；它不读 Data PostgreSQL，Data 也不代理查询。真实 Key 只写入未提交的
`.env.local` 或当前进程环境。

## 采集运行边界

Source 主数据、connector、parser、prompt、完整 Markdown Artifact 与采集编排归属 AgentRun 应用。Tidewise Data 只通过受认证的 `POST /api/data/v1/reviewed-event-imports` 原子接纳正式 Event 及其轻量证据记录；AgentRun 不得绕过 Data Service 直接写 Data DB。

历史 Source、scheduler/run 表只存在于旧 migration 历史中；当前 Schema 和运行时不再提供对应控制面。

## 运行 Admin 前端

Admin Portal BFF由统一compose在`9013`提供，并使用`ADMIN_SERVICE_TOKEN`鉴权。真实token只通过未提交的`.env.local`注入，不写入repo；本地只允许 `http://127.0.0.1:5174` Origin。采集器管理功能由 BFF 使用统一的 `AGENTRUN_SERVICE_TOKEN` 访问 `AGENTRUN_BASE_URL`，该服务令牌不会下发浏览器。

管理后台位于：

```text
admin-portal/frontend/
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

页面右上角输入 `ADMIN_SERVICE_TOKEN` 后，可以查询采集源、原始数据和事件。已退役的 scheduler 路由已经删除。

## 常见失败

- `ping postgres`：本地 PostgreSQL 未启动、端口不对、数据库不存在或 password 未注入。
- `pending migrations exist`：当前环境关闭了 `migration.auto_apply`，需要先运行 `dbmigrate -apply`。
- `insert raw document`：通常表示 migration 未执行、source seed 失败或 schema 与 repository 不一致。
- `admin token is not configured`：启动 `admin-portal/backend/cmd/server` 时没有注入 `ADMIN_SERVICE_TOKEN`。
