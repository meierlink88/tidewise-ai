# Local PostgreSQL, Neo4j And Ingestion Smoke

本目录提供 local PostgreSQL 和真实采集 smoke 的本地运行说明。这里的模板只用于开发环境，不得复用为 uat 或 prod secret 来源。

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

`project-entities` 会读取 PostgreSQL 的 `entity_nodes` 和 `entity_edges`，写入 Neo4j 的 `TidewiseEntity` 命名空间。`rebuild-entities` 只清理本系统 `projection_namespace=tidewise` 的实体图，不会清空整个 Neo4j database。

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

## 运行真实采集 smoke

先确认 migration 已完成，再执行：

```bash
APP_ENV=local DATABASE_PASSWORD=<local-password> go run ./cmd/ingest-smoke -max-documents 3
```

默认来源是无需凭证的公开 RSS。可以覆盖来源：

```bash
APP_ENV=local DATABASE_PASSWORD=<local-password> go run ./cmd/ingest-smoke \
  -source-url 'https://feeds.bbci.co.uk/news/business/rss.xml' \
  -source-name 'BBC Business RSS' \
  -max-documents 3 \
  -concurrency 1
```

复跑 smoke 应保持幂等：相同 source external id 或 content hash 不应制造重复 `raw_documents`。

## 运行 source catalog 多来源采集

`cmd/source-ingest` 会读取数据库中已经 seed 且状态为 active 的来源，并通过当前已注册的 connector/parser 写入 `raw_documents`。默认并发为 1，避免本地开发时一次打满外部来源。

运行全部 active 内容来源：

```bash
APP_ENV=local DATABASE_PASSWORD=<local-password> go run ./cmd/source-ingest \
  -source-type news \
  -concurrency 1
```

只运行 RSS 通道：

```bash
APP_ENV=local DATABASE_PASSWORD=<local-password> go run ./cmd/source-ingest \
  -channel rss_feed \
  -source-type news \
  -concurrency 2
```

显式验证 Eastmoney HTTP 来源前，应先在本地数据库中只启用少量 Eastmoney source，避免误抓大量 inactive 行情/板块来源；再按 provider 和 channel 过滤运行：

```bash
APP_ENV=local DATABASE_PASSWORD=<local-password> go run ./cmd/source-ingest \
  -provider eastmoney \
  -channel eastmoney \
  -concurrency 1
```

如果需要 RSSHub 来源，必须显式提供非敏感 base URL：

```bash
APP_ENV=local DATABASE_PASSWORD=<local-password> go run ./cmd/source-ingest \
  -channel rsshub_feed \
  -rsshub-base-url 'https://rsshub.app' \
  -concurrency 1
```

## 运行采集调度器

调度器读取 PostgreSQL 中的全局调度配置，默认关闭。先执行 migration，再用 SQL 或管理后台启用调度配置。建议本地先限制到 AI Web Research 或少量低风险来源：

```sql
UPDATE ingestion_scheduler_configs
SET enabled = true,
    mode = 'interval',
    interval_minutes = 60,
    concurrency = 1,
    batch_size = 10,
    timeout_seconds = 180,
    source_filter = '{"provider_key":"llm_web_research","ingest_channel":"ai_web_research","source_type":"news"}'::jsonb,
    timezone = 'Asia/Shanghai',
    updated_at = now()
WHERE id = 'default';
```

单轮运行用于 smoke 和排障：

```bash
APP_ENV=local DATABASE_PASSWORD=<local-password> go run ./cmd/ingestion-scheduler -once
```

查看当前调度配置，不触发采集：

```bash
APP_ENV=local DATABASE_PASSWORD=<local-password> go run ./cmd/ingestion-scheduler -dry-run
```

持续运行用于模拟生产进程。进程会按 `ingestion.scheduler_tick_seconds` 检查是否到期，只有全局配置到期时才触发采集：

```bash
APP_ENV=local DATABASE_PASSWORD=<local-password> go run ./cmd/ingestion-scheduler
```

也可以覆盖 tick 间隔：

```bash
APP_ENV=local DATABASE_PASSWORD=<local-password> go run ./cmd/ingestion-scheduler -tick-seconds 15
```

验证调度 run 和 source 级结果：

```sql
SELECT id, trigger_type, status, started_at, finished_at, total_sources, succeeded_sources, failed_sources
FROM ingestion_runs
ORDER BY started_at DESC
LIMIT 5;

SELECT run_id, source_id, status, documents_written, documents_duplicate, error_message
FROM ingestion_run_sources
ORDER BY started_at DESC
LIMIT 20;
```

## 运行 Admin API 和管理后台

Admin API 使用 `ADMIN_API_TOKEN` 鉴权。真实 token 只通过环境变量注入，不写入 repo：

```bash
APP_ENV=local DATABASE_PASSWORD=<local-password> ADMIN_API_TOKEN=<local-admin-token> go run ./cmd/admin-api
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

页面右上角输入 `ADMIN_API_TOKEN` 后，可以在“调度器设置”中读取和保存全局调度配置。本阶段管理后台只包含调度器设置菜单，后续采集源管理、原始数据列表和事件列表通过独立 change 扩展。

## 常见失败

- `ping postgres`：本地 PostgreSQL 未启动、端口不对、数据库不存在或 password 未注入。
- `pending migrations exist`：当前环境关闭了 `migration.auto_apply`，需要先运行 `dbmigrate -apply`。
- `fetch url status` 或 `parse rss feed`：公开来源临时不可用、限流或返回格式变化。可以用 `-source-url` 临时替换来源。
- `source ingest failed`：通常表示某些 active source 的 connector/parser 未注册、外部来源不可达或被限流；先缩小 `-provider`、`-channel`、`-source-type` 过滤范围排查。
- `insert raw document`：通常表示 migration 未执行、source seed 失败或 schema 与 repository 不一致。
- `admin token is not configured`：启动 `cmd/admin-api` 时没有注入 `ADMIN_API_TOKEN`。
