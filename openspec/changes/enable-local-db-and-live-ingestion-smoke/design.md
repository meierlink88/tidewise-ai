## Context

当前 `init-database-and-ingestion-layer` 已经完成事件知识 schema migration、迁移 runner 抽象、采集 source registry、connector/parser registry、采集 job、fake repository 和 Go 单元测试。缺口是这些能力还停留在文件、接口和 fake 层面：

- `backend/migrations/000001_init_event_knowledge_schema.sql` 使用 goose 格式，但当前没有真实 PostgreSQL 执行器。
- `backend/internal/migrations` 可以列出和模拟执行迁移，但还没有真实数据库版本表、advisory lock 或启动迁移检查。
- `backend/internal/repositories` 只有 `InMemoryRepository`，无法把 `source_catalogs` 和 `raw_documents` 写入 PostgreSQL。
- `backend/internal/jobs/IngestionJob` 可以调度采集流程，但没有本地数据库 seed 和真实采集 smoke 入口。
- 现有 domain ID 是 string，但数据库主键是 UUID。真实写库前必须统一稳定 UUID 生成或映射策略。

本 change 需要把“DDL 可执行”和“采集可入库”变成开发者本地可重复验证的事实，但默认单元测试仍不得依赖真实数据库或真实网络。

## Goals / Non-Goals

**Goals:**

- 提供真实 PostgreSQL 连接边界，并通过统一 config 读取 host、port、database、user、ssl mode 等非敏感配置，通过环境变量读取 password。
- 使用 repo 内 `backend/migrations` 创建真实 PostgreSQL 表，并记录迁移版本。
- 在 API 启动或独立迁移命令中执行迁移检查，并按环境配置决定是否自动 apply。
- 为 migration 执行增加 PostgreSQL advisory lock 或等价机制，避免多个实例并发执行 DDL。
- 增加 PostgreSQL repository 最小实现，覆盖采集源目录读取、采集源 seed/upsert、原始文档幂等写入和状态更新。
- 增加真实采集 smoke 入口，从无需凭证的公开来源采集少量真实文档，写入本地 PostgreSQL，并输出可审阅的成功、失败、重复和入库数量。
- 增加测试先行验证：默认测试用 fake、fixture、`httptest` 和纯函数测试；真实 PostgreSQL/网络 smoke 通过显式命令或环境变量开启。

**Non-Goals:**

- 不实现前端页面、真实业务 API、报告、订阅、支付或用户体系。
- 不实现 Agent 推理、RAG、Prompt 编排、图数据库、向量数据库或事件抽取。
- 不把采集结果表达为利好利空、投资建议、预测结论或交易信号。
- 不连接生产数据库，不提交真实数据库密码、token、cookie、API key 或生产连接串。
- 不直接修改 `prototype/` 或 `../doc/`。

## Decisions

### Decision 1: PostgreSQL 访问使用 `database/sql` + `pgx` driver

本 change 使用 Go 标准 `database/sql` 作为上层连接抽象，使用 `github.com/jackc/pgx/v5/stdlib` 作为 PostgreSQL driver。这样可以同时服务 goose migration 和 repository 查询，并保持实现对 AI 编程更友好：接口简单、测试边界清晰、错误路径明确。

替代方案：

- 直接使用 `pgxpool`：性能和 PostgreSQL 特性更强，但 goose 与 `database/sql` 集成更直接；本阶段先降低集成复杂度。
- 使用 ORM：会引入较多隐式行为，不适合当前 schema-first 和 migration-first 的工程策略。

### Decision 2: config 扩展数据库运行参数，secret 仍走环境变量

`backend/config/config.*.yaml` 增加非敏感字段，例如 database user、ssl mode、连接池大小和超时。数据库 password 继续来自 `DATABASE_PASSWORD`，或者本地 smoke 通过显式 `DATABASE_URL`/`TIDEWISE_DATABASE_URL` 这类环境变量注入完整连接串。配置文件和示例文件只保留占位说明。

本地 Docker Compose 如需出现，将放在 `infra/local/`，并通过 `.env.example` 提示变量名；真实 `.env.local` 必须保持未提交。

### Decision 3: migration 执行使用 goose，并包一层项目迁移服务

现有 SQL 文件已经采用 `-- +goose Up/Down` 标记，因此选择 `github.com/pressly/goose/v3` 作为 migration 执行库。项目内部新增 migration service：

- 读取统一 config 和迁移目录。
- 打开 PostgreSQL 连接并 ping。
- 使用 PostgreSQL advisory lock 包住 goose 执行。
- 支持 check-only 和 apply 两种模式。
- 输出 pending/applied/current version 信息，供命令行、启动日志和测试使用。

现有 `internal/migrations` 的 Source/Runner 抽象继续用于静态验证和纯单元测试；真实数据库执行由新 PostgreSQL runner 适配，不强行把 goose 塞进 fake store 接口。

### Decision 4: API 启动迁移检查和独立命令并存

新增 `backend/cmd/dbmigrate` 独立命令用于本地、UAT 和部署流水线显式执行迁移。`backend/cmd/api` 启动时也调用同一个 migration service：

- `migration.auto_apply=true` 时，启动阶段自动应用 pending migration。
- `migration.auto_apply=false` 时，只检查 pending migration，并在存在 pending 时失败或返回不可就绪，避免服务在未知 schema 上运行。

这样既满足“系统启动检查数据库更新”，也保留部署流程中手动执行/审阅 migration 的能力。

### Decision 5: PostgreSQL repository 先覆盖采集最小闭环

本 change 不实现所有领域表 repository，只实现当前 smoke 必需的最小边界：

- `source_catalogs`: active source 查询、按固定 ID seed/upsert smoke source。
- `raw_documents`: 幂等 upsert、重复检测、状态更新、smoke 计数查询。

幂等策略优先使用 `(source_id, source_external_id)`，当外部 ID 不存在时使用 `(source_id, content_hash)`。由于当前唯一索引包含 partial index，repository 会先查询已有记录再插入/更新，避免依赖复杂的 partial `ON CONFLICT` 写法。

### Decision 6: 写库 ID 统一生成稳定 UUID

数据库主键是 UUID，现有采集候选 ID 是自由字符串。真实写库时引入稳定 UUID 生成策略：

- seed source 使用固定 UUID，保证本地 smoke 可重复。
- raw document ID 由 source ID、source external ID 或 content hash 生成稳定 UUID。
- 如果上游 candidate 已提供合法 UUID，则直接使用；否则由 repository 或 ingestion ID helper 映射为稳定 UUID。

该策略保证重复采集不会制造新主键，也避免为了真实入库修改 schema 主键类型。

### Decision 7: 真实采集 smoke 显式运行，不进入默认测试

新增 `backend/cmd/ingest-smoke` 或等价命令：

- 加载 config 和数据库连接。
- 确认 migration 已执行。
- seed 一个无需凭证的公开 RSS/Atom 或等价来源。
- 注册现有 RSS connector/parser。
- 采集最多 N 条真实文档并写入 PostgreSQL。
- 输出 JSON report，至少包含 sources、fetched、created、duplicates、failed、raw_document_count。

默认 `go test ./...` 不访问真实网络和真实数据库。真实 smoke 通过命令执行，或通过 `TIDEWISE_RUN_LIVE_SMOKE=1`、`TIDEWISE_DATABASE_URL` 等显式环境变量开启集成测试。

## Risks / Trade-offs

- 真实外部 RSS 或网页来源可能临时不可用 → smoke source 必须可配置，默认来源失败时应输出明确错误，不伪造成成功。
- 本地 PostgreSQL 不存在或密码未配置 → 命令必须快速失败，并提示需要设置的环境变量，不把 secret 打印出来。
- goose down migration 对初始 schema 不适合自动回滚 → 回滚策略以恢复备份或追加 forward migration 为主，不在有数据环境下清空表。
- API 启动自动迁移可能影响生产发布策略 → `prod` 可通过 config 关闭 auto apply，部署流水线先运行 `dbmigrate`。
- ID 映射策略如果不稳定会破坏幂等 → 必须用 table-driven tests 覆盖同一 source/external ID/hash 多次生成相同 UUID 的场景。
- 真实数据库 integration tests 运行成本高 → 默认测试只跑纯单元和 fake，真实 DB 测试由显式环境变量触发。

## Migration Plan

1. 先新增配置、DSN/连接构建、ID helper、migration service 和 repository 的测试。
2. 实现 PostgreSQL 连接、goose migration、advisory lock、repository 和 smoke 命令。
3. 在本地 PostgreSQL 上运行 `dbmigrate`，确认关键表和 migration 版本存在。
4. 运行 `ingest-smoke`，确认至少 1 条真实 raw document 写入本地库。
5. 默认验证运行 `go test ./...`、`openspec validate enable-local-db-and-live-ingestion-smoke` 和 `openspec validate --all`。
6. 若真实 smoke 失败，只能以明确外部环境阻塞记录，不能把 change 标记为完成态。

## Open Questions

- 默认公开 smoke source 在 apply 阶段需要最终确认。优先选择无需凭证、稳定、允许抓取、RSS/Atom 格式清晰的来源；如果默认来源不稳定，应允许通过环境变量覆盖。
- 本地 PostgreSQL 是优先使用已有本机服务，还是在 `infra/local/` 提供 Docker Compose 模板并由开发者选择启动。实现上应同时支持二者，只把 Docker Compose 作为可选便利入口。
