# Local PostgreSQL And Ingestion Smoke

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

## 常见失败

- `ping postgres`：本地 PostgreSQL 未启动、端口不对、数据库不存在或 password 未注入。
- `pending migrations exist`：当前环境关闭了 `migration.auto_apply`，需要先运行 `dbmigrate -apply`。
- `fetch url status` 或 `parse rss feed`：公开来源临时不可用、限流或返回格式变化。可以用 `-source-url` 临时替换来源。
- `source ingest failed`：通常表示某些 active source 的 connector/parser 未注册、外部来源不可达或被限流；先缩小 `-provider`、`-channel`、`-source-type` 过滤范围排查。
- `insert raw document`：通常表示 migration 未执行、source seed 失败或 schema 与 repository 不一致。
