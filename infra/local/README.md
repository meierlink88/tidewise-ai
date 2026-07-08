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
  -max-documents 3
```

复跑 smoke 应保持幂等：相同 source external id 或 content hash 不应制造重复 `raw_documents`。

## 常见失败

- `ping postgres`：本地 PostgreSQL 未启动、端口不对、数据库不存在或 password 未注入。
- `pending migrations exist`：当前环境关闭了 `migration.auto_apply`，需要先运行 `dbmigrate -apply`。
- `fetch url status` 或 `parse rss feed`：公开来源临时不可用、限流或返回格式变化。可以用 `-source-url` 临时替换来源。
- `insert raw document`：通常表示 migration 未执行、source seed 失败或 schema 与 repository 不一致。
