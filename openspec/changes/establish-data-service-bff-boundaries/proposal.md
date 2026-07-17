## Why

当前 change 的范围不只是把 Data API/BFF 拆出边界，还必须退役 Tidewise 仓库内的 ingestion scheduler/runtime。`cmd/ingestion-scheduler`、`cmd/source-ingest`、`cmd/ingest-smoke` 与其 application/runtime 会直接执行 connector、写入 Data PostgreSQL；这与新的 ownership 及外部 `agent-run` 执行边界冲突。若只删 command 而不处理 Admin 控制面、配置、repository/domain、文档和测试，会留下不可运行或可误用的旧路径。

本 Proposal 仍停在 Proposal Review：先固定删除/保留清单、Agent 交接合同、兼容性、Gate Map、测试/testdata 证据与恢复策略；不进入 Apply，不修改 `agent-run`，不执行数据库或部署写入。

## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---:|---|---|---|
| 1 | Proposal Review | R0 | yes | SPEC_SEMANTICS | 批准三服务边界、scheduler/runtime 退役、Agent import contract、保留/删除清单、兼容策略与 non-goals |
| 2 | Inventory/architecture | R1 | no | NONE | 冻结 caller/import/table/test/testdata manifest；新增 BFF→Data DB/repository 与旧 runtime 禁止检查 |
| 3 | Service skeleton | R1 | no | NONE | 三个 service entry/facade/health、单 Go module、薄 compatibility wiring；不 bulk move 业务代码 |
| 4 | Data import/API contract | R1 | no | NONE | scoped source-metadata read、raw-document/reviewed-event HTTP contracts 与 client/tests；未通过不得删 runtime |
| 5 | Miniapp BFF decoupling | R1 | no | NONE | Miniapp 通过 DataServiceClient，保持外部页面 API |
| 6 | Admin retirement/decoupling | R1 | no | NONE | 保留 raw/event/source；scheduler API 认证后返回一个部署窗口的 `410 Gone` tombstone，移除 scheduler UI |
| 7 | Scheduler/runtime retirement | R1 | no | NONE | 删除精确列出的 commands/packages/config/repository/domain/tests/docs；保留 connectors、import、tables、migrations |
| 8 | Assets/local/CI/docs | R1 | no | NONE | service-owned build/health、local dry config、CI/reference cleanup；排除 UAT/prod/shared |
| 9 | Data PostgreSQL role/credential | R2 | yes | DEPLOYMENT_SECURITY | 仅另行批准后执行 local roles/grants/credential cutover；不得由 R1 推定授权 |
| 10 | Apply-final Review | R1 | yes | APPLY_FINAL | 验收 scoped diff、兼容性、测试计数/fixture 审计、完整验证和未验证风险 |

## Complexity Budget

| Key | Value |
|---|---|
| human_gates | 3 |
| stateful_layers | 2 |
| checkpoints | 4 |
| full_test_runs | 1 |
| continuous_automation_scope | packages:2-8 |

## Stateful Layer Map

| Layer | Package | Environment | Order | Scope | Exclusions | Recovery Evidence | Recovery Baseline | Expected Counts/Hash/Schema | Before Assertions | After Assertions | Stop Conditions |
|---|---:|---|---:|---|---|---|---|---|---|---|---|
| local-data-db-role-boundary | 9 | local | 1 | `data_service_rw/migrate/ro` 最小 grants；记录 role/grant manifest | UAT/prod/shared、业务数据、历史 scheduler tables/migrations | backup | new:local-data-db-security-baseline | counts=review-gated;hash=review-gated;schema=role-grant-manifest-v1 | identity、现有 grants、owner、counts/hash/schema 与 backup 可恢复 | 批准 manifest 一致，业务 schema/data counts/hash 不变 | identity/grant/owner/count/hash/schema 漂移或 backup 不可恢复立即停止 |
| local-service-credential-cutover | 9 | local | 2 | Data Service 使用批准 role；BFF/Agent 无 DB credential | migration/seed/import/projection/secret 入库 | backup | reuse:local-data-db-security-baseline | counts=review-gated;hash=review-gated;schema=role-grant-manifest-v1 | 同一 baseline identity/scope/count/hash/schema 未漂移，旧 credential 可回切 | Data 最小权限通过，BFF/Agent 启动配置无 DB credential | health/权限/回切失败立即停止并使下一层授权失效 |

## What Changes

- 在 monorepo、单一 `backend/go.mod` 内建立 Data Service、Miniapp BFF、Admin Portal BFF；BFF 只通过版本化 HTTP REST + JSON/OpenAPI 调用 Data Service。
- Data Service 独占 Data domain、repository、PostgreSQL、Neo4j/向量投影、受控 raw-document/reviewed-event import；Miniapp/Admin/Agent 不持有 Data DB 凭据。
- 先实现并测试 Data source-metadata 只读 contract 与 raw-document/reviewed-event 两个 import contract，再删除 Tidewise scheduler/runtime；外部 `agent-run` 负责 schedule、connector execution、provider retry/rate-limit，并只经 scoped Data API 读取批准 metadata、提交导入产物。
- 退役 scheduler Admin API/UI、runtime-only config、直接 scheduler/run repository/domain、启动装配和文档。兼容窗口固定为认证后 machine-readable `410 Gone`，持续一个实际部署窗口且不读写 scheduler 表；consumer/log 审计为零后由后继 change 删除 route。
- 保留 source catalog seed、connectors/parsers、raw-document/source/event repositories、event-import、graph projector、entity seed、dbmigrate、所有历史 migration 与数据表；connectors 短期归 Data Service、但 Tidewise 不再组装或运行它们。
- 用 architecture/reference tests 禁止 BFF 直接依赖 Data DB/repository、禁止旧 runtime/反向 caller 复活；按 production counterpart 清理失效测试，保留有效 domain/repository/connector/API/migration/idempotency/transaction/security 覆盖。

## Frozen Proposal Review Decisions

- Data API namespace 固定为 `/internal/data/v1`，由网络边界和 service identity 共同限制。
- 本 change 固定使用各消费方拥有的小型受控手写 typed client + OpenAPI schema drift test，不引入生成器。
- 三个 Admin scheduler endpoints 固定为认证后的 machine-readable `410 Gone` tombstone并保留一个实际部署窗口；本 change 不采用直接 404。
- raw-document import 固定为有界 whole-batch validation + atomic write；任一 item 非法时整批在业务写入前拒绝，不提供逐 item 部分成功。

## Capabilities

### New Capabilities

- `backend-service-boundaries`: 建立 Data、Miniapp BFF、Admin BFF 的 runtime/API/data ownership、部署资产、迁移与测试清理边界。

### Modified Capabilities

- `backend-subsystem-boundaries`: 将共享模块化单体收窄为三服务 ownership，禁止 Tidewise 采集 runner 与 BFF→Data 内部依赖。
- `technical-architecture`: 固定外部 `agent-run`/Agent Server 只经 Data API 读取批准 metadata并导入 raw/reviewed-event。
- `persistence-and-contracts`: 固定 Data PostgreSQL 独占、历史 scheduler tables/migrations 保留与独立 R2 权限切换。
- `event-import-and-tag-catalog`: 保留 reviewed-event transaction/receipt/CLI compatibility并新增受控 HTTP import。
- `data-ingestion-layer`: 保留 source catalog/connectors/parsers，新增 raw import，移除 Tidewise execution/runtime 语义。
- `admin-console`: 移除 scheduler UI/config/run 行为并定义认证后的 410 兼容窗口。
- `ingestion-scheduler`: 删除全部现行 scheduler requirements；不在 Tidewise 建立替代 scheduler。

## Exact Retirement / Keep Manifest

### 删除或替换

| Area | Scope | Reason |
|---|---|---|
| Commands | `backend/cmd/{ingestion-scheduler,source-ingest,ingest-smoke}/**` | 都会执行或触发真实 connector/runtime；不得留下手动后门 |
| Application/runtime | `backend/internal/apps/ingestion/{scheduler,runtime}/**`、`backend/internal/apps/ingestion/health/doc.go` | 调度决策、source selection、connector→parser→writer、并发/失败隔离已移出 Tidewise |
| Config | scheduler tick/timezone 与 runtime-only ingestion keys、三套 templates 中对应死键 | 删除无受支持 caller 的配置；Data API 使用独立 timeout/import 配置 |
| Admin surface | scheduler DTO/handlers/repository methods；前端 scheduler client/page/tab/styles | scheduler capability 退役；兼容窗口只允许无 DB 的 410 tombstone |
| Persistence adapters | `repositories/scheduler.go`、`ingestion_run.go` 及 memory scheduler/run state；domain scheduler/run symbols | 不再创建/更新/读取 scheduler state；历史表和 migration 不动 |
| Tests | 对应上述 production code 的 23 direct tests；mixed domain 5、core 4、Admin 4 改删/替换；frontend scheduler-only 10，3 mixed cases 改写 | 每一项绑定 production removal 或新 retirement/architecture contract |

### 必须保留

| Capability | Files/data | Boundary |
|---|---|---|
| Historical state | `ingestion_scheduler_configs`, `ingestion_runs`, `ingestion_run_sources`；`000005_add_ingestion_scheduler.sql` 及全部 21 migrations | Data-owned historical data；不 drop/truncate/rewrite/删除 rows，无新 scheduler API |
| Source catalog | `backend/cmd/source-seed`、`backend/internal/apps/ingestion/sourcecatalog/**`、`backend/data/source_catalogs/**` | 版本化 metadata/seed，不执行 connector |
| Connector/parser adapters | `backend/internal/apps/ingestion/connectors/**`、`backend/internal/apps/ingestion/parsers/**`、仍被使用的 `backend/internal/apps/ingestion/core` contracts、`backend/data/prompts/ingestion/**` | Data-owned、受测、短期无 Tidewise production runner；外部 repo 不得 import Go `internal` |
| Import/data access | `backend/cmd/event-import`、`backend/internal/apps/ingestion/eventimport/**`、`backend/internal/domain/eventimport/**`、event/raw/source/admin query repositories与contracts | reviewed event 与 raw-document 是独立受控 API；幂等、receipt、事务合同保留 |
| Other Data jobs | entity seed、graph projector、dbmigrate、migration/platform tests | 非采集执行；本 change 不运行真实写入 |
| Fixtures | `backend/testdata/event-import/reviewed-outbox-v1.json`、architecture task-design fixtures、source catalogs/prompts | 引用有效；当前 testdata 删除计划为 0 |

## Compatibility and Agent-run Handoff

删除 runtime 后允许存在明确记录的采集停止窗口；不得在 Tidewise 留 scheduler、worker、真实 smoke 或“临时手动” source-ingest。`agent-run` 后继 change 负责 schedule/connector execution，并以 `/internal/data/v1`、scoped service identity、脱敏 source metadata、bounded batch、source attribution、payload hash/idempotency、request-id、timeout/retry 和 receipt/result 验收；它不能调用 Tidewise connector `internal` package、BFF 或 PostgreSQL。reviewed event import 继续写 raw/event/source/tag/receipt 的单 transaction；raw-document import 只写/复用 raw document，不自动创建 event/tag/research 结论。

## Impact / Non-goals

受影响区域包括 backend commands/apps/config/domain/repositories、Admin API/UI、local README、architecture/CI/reference tests、service assets 和 OpenSpec specs。唯一 repo-template 窄例外是删除 `backend/config/config.{local,uat,prod}.yaml` 中已无 caller 的 ingestion keys；这不授权修改真实环境、secret或部署行为。当前 `.github/workflows/deploy-uat.yml`、`infra/uat/**`、prod/shared runtime、真实 DB/Neo4j/deploy 均不在范围内；UAT active change 完成后另行规划三服务 rollout。不开 multi-module/repo，不复制 connectors 到外部 repo，不重写 migration/搬表/删数据，不改变保留的 miniapp/admin raw/event/source API 语义。

## Verification and Effort

删除前基线：114 个 Go test files、541 个 `func Test`；frontend 合计 8 个 test files、44 个 `it/test` cases，其中 Admin 7/26、Miniapp 1/18；13 个落盘 testdata files 均有引用。实施必须提供逐文件 before/after manifest：旧 scheduler/runtime 行为 36 个 backend tests、frontend scheduler-only 10 个 cases 的删/改原因；Miniapp 现有 18 cases 预期零删除；新增 import/410/architecture tests 单独计入；fixture 引用审计必须为零悬空且本次预计删除 0 个有效 fixture。运行 connectors/parsers/sourcecatalog/eventimport/raw-doc/migration/idempotency/transaction/security targeted suites、受影响服务完整 suites、architecture/reference checks，最后一次 `go test ./...`、前端 test/build、OpenSpec strict validate、diff/scope/secret checks。

| Package | Estimate |
|---|---:|
| 1-3 Review/inventory/binaries | 5-7 days |
| 4 Data import/API | 5-8 days |
| 5-6 BFF/admin retirement | 4-8 days |
| 7 runtime/test cleanup | 3-5 days |
| 8 assets/docs/CI | 2-4 days |
| 9 DB roles (授权后) | 1-2 days |
| 10 final review | 0.5-1 day |

总计约 22-35 engineer-days；外部 `agent-run`、真实 UAT/prod rollout、multi-module/repo split 不计入。
