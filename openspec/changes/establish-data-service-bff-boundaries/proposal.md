## Why

当前 change 的范围不只是把 Data API/BFF 拆出边界，还必须退役 Tidewise 仓库内的 ingestion scheduler/runtime。`cmd/ingestion-scheduler`、`cmd/source-ingest`、`cmd/ingest-smoke` 与其 application/runtime 会直接执行 connector、写入 Data PostgreSQL；这与新的 ownership 及外部 `agent-run` 执行边界冲突。若只删 command 而不处理 Admin 控制面、配置、repository/domain、文档和测试，会留下不可运行或可误用的旧路径。

本 Proposal amendment 仍停在 Proposal Review：先固定删除/保留清单、Agent 交接合同、兼容性、Gate Map、raw receipt schema、测试/testdata 证据与恢复策略；本次 amendment 不恢复 Apply、不修改 `agent-run`、不创建 migration 文件，也不执行数据库或部署写入。用户已于 2026-07-17 明确批准新增 `raw_document_import_receipts` migration，并授权在 amendment 获批且 Package 5 order 1 preflight evidence 精确通过后自动对 local PostgreSQL执行一次该 migration；该条件式授权不扩展到其他 SQL、环境或状态层。

## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---:|---|---|---|
| 1 | Proposal Review | R0 | yes | SPEC_SEMANTICS | 批准三服务边界、raw receipt amendment、scheduler/runtime 退役、Agent import contract、保留/删除清单、兼容策略与 non-goals |
| 2 | Inventory/architecture (already complete) | R1 | no | NONE | 已冻结 caller/import/table/test/testdata manifest；已新增 BFF→Data DB/repository 与旧 runtime 禁止检查 |
| 3 | Service skeleton | R1 | no | NONE | 三个 service entry/facade/health、单 Go module、薄 compatibility wiring；不 bulk move 业务代码 |
| 4 | Data API/import code + migration artifact | R1 | no | NONE | scoped source metadata、raw/reviewed import code、client/tests及`000022` artifact；禁止执行 SQL |
| 5 | Local raw receipt schema apply/integration | R2 | yes | DRIFT_RECOVERY | 两层 fail-closed local preflight/backup 后仅执行一次已批准的`000022`并做只读schema/零行/rollback assertions |
| 6 | Miniapp BFF decoupling | R1 | no | NONE | Miniapp 通过 DataServiceClient，保持外部页面 API |
| 7 | Admin retirement/decoupling | R1 | no | NONE | 保留 raw/event/source；scheduler API 认证后返回一个部署窗口的`410 Gone` tombstone，移除 scheduler UI |
| 8 | Scheduler/runtime retirement | R1 | no | NONE | 删除精确列出的 commands/packages/config/repository/domain/tests/docs；保留 connectors、import、tables、migrations |
| 9 | Assets/local/CI/docs | R1 | no | NONE | service-owned build/health、local dry config、CI/reference cleanup；排除 UAT/prod/shared |
| 10 | Local Data DB roles/credential | R2 | yes | DEPLOYMENT_SECURITY | 独立 local roles/grants/owner/credential cutover；不得与 Package 5 migration授权或执行合并 |
| 11 | Apply-final Review | R1 | yes | APPLY_FINAL | 验收 scoped diff、migration manifest、兼容性、测试计数/fixture 审计、完整验证和未验证风险 |

## Complexity Budget

| Key | Value |
|---|---|
| human_gates | 4 |
| stateful_layers | 4 |
| checkpoints | 5 |
| full_test_runs | 1 |
| continuous_automation_scope | packages:2-4,6-9 |

## Stateful Layer Map

| Layer | Package | Environment | Order | Scope | Exclusions | Recovery Evidence | Recovery Baseline | Expected Counts/Hash/Schema | Before Assertions | After Assertions | Stop Conditions |
|---|---:|---|---:|---|---|---|---|---|---|---|---|
| local-raw-receipt-schema-preflight | 5 | local | 1 | 只读核验数据库identity、21-migration baseline/hash、clean backup、schema/count invariants及`000022`候选hash和dry contract；只允许只读SQL，禁止migration/DDL/DML | seed、import业务写入、Neo4j、UAT/prod/shared、部署、roles/credentials、restore、retry、forward-fix | backup | new:local-raw-receipt-schema-baseline | counts=21-migrations;hash=historical-21+000022-candidate;schema=pre-000022-manifest-v1 | local identity/scope、已应用migration=21、历史hash、仅`000022`pending、全部固定对象名缺失、DDL/ledger权限、schema/counts及完整可读backup+documented recovery精确匹配；不声称restore-tested | 冻结可复核preflight/backup与`000022`exact hash，确认数据库schema/data未变化 | identity/scope/count/hash/schema/privilege、pending/object collision或backup任一漂移立即停止，禁止进入order 2 |
| local-raw-receipt-schema-apply | 5 | local | 2 | 复验baseline后只执行一次已批准的transactional`000022`，随后只读断言schema/constraints/indexes/trigger/零初始receipt rows并运行全rollback integration tests | seed、业务数据持久化、Neo4j、UAT/prod/shared、部署、roles/credentials、restore、retry、forward-fix、真实commit-race fixture | backup | reuse:local-raw-receipt-schema-baseline | counts=22-migrations+0-receipts;hash=historical-21-unchanged+000022-approved;schema=raw-document-import-receipts-v1 | 复验同一local identity/scope/count/hash/schema、privilege、backup及exact`000022`hash，无任何新增pending/object collision或数据漂移 | 仅ledger新增000022且仅列明新对象出现；22 migrations、冻结表合同、零初始receipt rows与transaction rollback后零残留成立，pre-existing业务schema/data不变 | apply或任一assertion失败、partial state、非零receipt rows或漂移立即停止；不得restore、retry或forward-fix |
| local-data-db-role-boundary | 10 | local | 1 | `data_service_rw/migrate/ro`最小grants；receipt table/function owner转给`data_service_migrate`，收敛PUBLIC/function privilege，runtime仅必要SELECT/INSERT；记录manifest | UAT/prod/shared、业务数据、历史 scheduler tables/migrations、Package 5 schema apply | backup | new:local-data-db-security-baseline | counts=22-migrations+review-gated-business-rows;hash=historical-21-unchanged+000022-applied;schema=role-grant-manifest-v2 | identity、现有grants、owner、counts/hash/schema与backup可恢复 | 批准owner/grant manifest一致，receipt不可UPDATE/DELETE/TRUNCATE，业务schema/data counts/hash不变 | identity/grant/owner/count/hash/schema漂移或backup不可恢复立即停止 |
| local-service-credential-cutover | 10 | local | 2 | Data Service使用批准role；BFF/Agent无DB credential | migration/seed/import/projection/secret入库、Package 5 schema apply | backup | reuse:local-data-db-security-baseline | counts=22-migrations+review-gated-business-rows;hash=historical-21-unchanged+000022-applied;schema=role-grant-manifest-v2 | 复验同一local identity/scope/count/hash/schema与backup，旧credential可回切 | Data最小权限通过，BFF/Agent启动配置无DB credential | health/权限/回切失败立即停止并使未执行授权失效 |

## What Changes

- 在 monorepo、单一 `backend/go.mod` 内建立 Data Service、Miniapp BFF、Admin Portal BFF；BFF 只通过版本化 HTTP REST + JSON/OpenAPI 调用 Data Service。
- Data Service 独占 Data domain、repository、PostgreSQL、Neo4j/向量投影、受控 raw-document/reviewed-event import；raw import使用独立、caller-scoped、不可变的`raw_document_import_receipts`审计合同，Miniapp/Admin/Agent不持有Data DB凭据。
- 先实现并测试 Data source-metadata 只读 contract 与 raw-document/reviewed-event 两个 import contract，再删除 Tidewise scheduler/runtime；外部 `agent-run` 负责 schedule、connector execution、provider retry/rate-limit，并只经 scoped Data API 读取批准 metadata、提交导入产物。
- 退役 scheduler Admin API/UI、runtime-only config、直接 scheduler/run repository/domain、启动装配和文档。兼容窗口固定为认证后 machine-readable `410 Gone`，持续一个实际部署窗口且不读写 scheduler 表；consumer/log 审计为零后由后继 change 删除 route。
- 保留 source catalog seed、connectors/parsers、raw-document/source/event repositories、event-import、graph projector、entity seed、dbmigrate、所有历史 migration 与数据表；connectors 短期归 Data Service、但 Tidewise 不再组装或运行它们。
- 用 architecture/reference tests 禁止 BFF 直接依赖 Data DB/repository、禁止旧 runtime/反向 caller 复活；按 production counterpart 清理失效测试，保留有效 domain/repository/connector/API/migration/idempotency/transaction/security 覆盖。

## Frozen Proposal Review Decisions

- Data API namespace 固定为 `/internal/data/v1`，由网络边界和 service identity 共同限制。
- 本 change 固定使用各消费方拥有的小型受控手写 typed client + OpenAPI schema drift test，不引入生成器。
- 三个 Admin scheduler endpoints 固定为认证后的 machine-readable `410 Gone` tombstone并保留一个实际部署窗口；本 change 不采用直接 404。
- raw-document import 固定为有界 whole-batch validation + atomic write；任一 item 非法时整批在业务写入前拒绝，不提供逐 item 部分成功。

### Proposed Package 6 Research Contract Correction

Package 4 的 Research Data OpenAPI/typed client 值域与已发布主规格 `openspec/specs/research-theme-anchor-foundation/spec.md`、`000021_add_research_theme_anchor_foundation.sql`、domain/repository 的一致语义发生漂移，本 amendment checkpoint 拟在恢复 Package 6 Apply 前将权威合同纠正为：`impact_level=high|focus|watch`、`transmission_stage=upstream|midstream|downstream|infrastructure|service`、`anchor_type=policy|supply|demand|technology|cost|geopolitics|market_structure`、`importance=primary|secondary|contextual`、index `impact_direction=positive|negative|mixed|neutral`、`evidence_role=driver|supporting|contradicting|context`；`trading_direction`保持trim后非空的自然语言string而非enum。Theme chain node与Anchor chain node必须使用两个明确DTO/schema，前者输出`impact_summary`、后者输出`relation_summary`，不得丢字段或做隐式语义映射。批准后Package 6只同步修正Data OpenAPI、Miniapp handwritten dataclient DTO/contract drift tests、Data handler非空golden及Miniapp BFF public golden/call-count tests，并保持现有`/api/v1/miniapp/research/*` JSON、cursor、排序与400/404/500错误语义；本checkpoint及后续纠偏均不变更数据库schema、migration或数据。

### Approved Raw Receipt Amendment

用户于2026-07-17明确批准新增forward-only `backend/migrations/000022_add_raw_document_import_receipts.sql`，并授权在本amendment获批且Package 5 order 1 evidence无漂移通过后自动执行一次local apply。当前21个历史migration文件的冻结聚合SHA-256为`2ed0dd004ab3b0bad633af5f0107a9ddffa28b83422b643f821c2cd47fb02dfc`（对按路径排序的21行`shasum -a 256` manifest再次执行SHA-256）；Package 4只能新增第22个文件并记录其独立exact hash，禁止改动前21个文件。

最小不可变表合同固定为：`id UUID PRIMARY KEY`、`caller_identity TEXT NOT NULL`、`idempotency_key TEXT NOT NULL`、`payload_hash CHAR(64) NOT NULL`、`raw_document_ids UUID[] NOT NULL`、`result_payload JSONB NOT NULL`、`imported_at TIMESTAMPTZ NOT NULL DEFAULT now()`；`caller_identity`来自认证principal而非request body，`UNIQUE(caller_identity,idempotency_key)`、1..200 caller/key、lowercase 64-hex hash、一维/无NULL/非空raw IDs与required/cross-field-consistent JSON result checks均使用固定命名constraint，另有imported-time index和拒绝`UPDATE/DELETE/TRUNCATE`的表专用statement trigger/function。`result_payload`保存首次commit的完整稳定业务result（request id属于每次transport envelope，不写入该snapshot），后续同caller+key+hash必须精确重放该snapshot；同caller+key不同hash返回409且不修改；status查询命中row即`completed`并以stored snapshot为权威，缺row即`unknown`/尚未commit。`000022`的Goose Up必须保持默认单transaction且禁止`NO TRANSACTION`，Down必须以`RAISE EXCEPTION`失败而不能成功降ledger。该表不关联或复用events、sources、tags、review状态或`event_import_receipts`。

## Capabilities

### New Capabilities

- `backend-service-boundaries`: 建立 Data、Miniapp BFF、Admin BFF 的 runtime/API/data ownership、部署资产、迁移与测试清理边界。

### Modified Capabilities

- `backend-subsystem-boundaries`: 将共享模块化单体收窄为三服务 ownership，禁止 Tidewise 采集 runner 与 BFF→Data 内部依赖。
- `technical-architecture`: 固定外部 `agent-run`/Agent Server 只经 Data API 读取批准 metadata并导入 raw/reviewed-event。
- `persistence-and-contracts`: 固定 Data PostgreSQL 独占、21个历史migration保留、独立raw receipt forward migration/local apply与另行R2权限切换。
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
| Historical state | `ingestion_scheduler_configs`, `ingestion_runs`, `ingestion_run_sources`；`000005_add_ingestion_scheduler.sql`及当前21个历史migrations | Data-owned historical data；21个文件byte-for-byte保留，不drop/truncate/rewrite/删除rows，无新scheduler API |
| Source catalog | `backend/cmd/source-seed`、`backend/internal/apps/ingestion/sourcecatalog/**`、`backend/data/source_catalogs/**` | 版本化 metadata/seed，不执行 connector |
| Connector/parser adapters | `backend/internal/apps/ingestion/connectors/**`、`backend/internal/apps/ingestion/parsers/**`、仍被使用的 `backend/internal/apps/ingestion/core` contracts、`backend/data/prompts/ingestion/**` | Data-owned、受测、短期无 Tidewise production runner；外部 repo 不得 import Go `internal` |
| Import/data access | `backend/cmd/event-import`、`backend/internal/apps/ingestion/eventimport/**`、`backend/internal/domain/eventimport/**`、event/raw/source/admin query repositories与contracts | reviewed event与raw-document是独立受控API；event receipt合同保留，raw import新增独立immutable receipt合同 |
| Raw import receipt | Package 4新增`000022_add_raw_document_import_receipts.sql`及Data-owned repository/API；Package 5只在local应用一次 | caller-scoped key/hash、原result重放、raw batch membership和import/transaction timestamp可审计；不创建event/source/tag/review placeholder |
| Other Data jobs | entity seed、graph projector、dbmigrate、migration/platform tests | 非采集执行；本 change 不运行真实写入 |
| Fixtures | `backend/testdata/event-import/reviewed-outbox-v1.json`、architecture task-design fixtures、source catalogs/prompts | 引用有效；当前 testdata 删除计划为 0 |

## Compatibility and Agent-run Handoff

删除 runtime 后允许存在明确记录的采集停止窗口；不得在 Tidewise 留 scheduler、worker、真实 smoke 或“临时手动” source-ingest。`agent-run` 后继 change 负责 schedule/connector execution，并以 `/internal/data/v1`、scoped service identity、脱敏 source metadata、bounded batch、source attribution、payload hash/idempotency、request-id、timeout/retry 和 receipt/result 验收；它不能调用 Tidewise connector `internal` package、BFF 或 PostgreSQL。reviewed event import继续写raw/event/source/tag/event receipt的单transaction；raw-document import在whole-batch validation后以一个PostgreSQL transaction写/复用raw documents并插入独立immutable raw receipt，不自动创建event/tag/research结论。未知网络结果先按认证caller+idempotency key查询：已有receipt为completed并返回原result，缺失为unknown/尚未commit。

## Impact / Non-goals

受影响区域包括 backend commands/apps/config/domain/repositories、Admin API/UI、local README、architecture/CI/reference tests、service assets 和 OpenSpec specs。唯一 repo-template 窄例外是删除 `backend/config/config.{local,uat,prod}.yaml` 中已无 caller 的 ingestion keys；这不授权修改真实环境、secret或部署行为。当前 `.github/workflows/deploy-uat.yml`、`infra/uat/**`、prod/shared runtime、Neo4j/deploy均不在范围内；唯一数据库schema写入是Package 5在order 1 preflight evidence精确通过后对local PostgreSQL应用一次`000022`，唯一后续权限写入是完全分离的Package 10。不开multi-module/repo，不复制connectors到外部repo，不改前21个migration、不搬表/删数据，不改变保留的miniapp/admin raw/event/source API语义。

## Verification and Effort

删除前基线：114个Go test files、541个`func Test`；frontend合计8个test files、44个`it/test` cases，其中Admin 7/26、Miniapp 1/18；13个落盘testdata files均有引用。实施必须提供逐文件before/after manifest：旧scheduler/runtime行为36个backend tests、frontend scheduler-only 10个cases的删/改原因；Miniapp现有18 cases预期零删除；新增raw receipt schema/repository/API/transaction-race/status、reviewed import、410与architecture tests单独计入；fixture引用审计必须为零悬空且本次预计删除0个有效fixture。migration manifest固定为repo before=`21`且历史hash=`2ed0dd004ab3b0bad633af5f0107a9ddffa28b83422b643f821c2cd47fb02dfc`，Package 4 after=`22`且前21 hash不变、`000022`独立exact SHA-256已记录并贯穿最终manifest，Package 5 local before=`21`/table absent、after=`22`/table contract exact/初始rows=`0`。运行connectors/parsers/sourcecatalog/eventimport/raw-doc/raw-receipt/migration/idempotency/transaction/security targeted suites、受影响服务完整suites、architecture/reference checks，最后一次`go test ./...`、前端test/build、OpenSpec strict validate、diff/scope/secret checks。

| Package | Estimate |
|---|---:|
| 1-3 Review/inventory/binaries | 5-7 days |
| 4 Data API/import + migration artifact | 6-10 days |
| 5 local raw receipt schema apply/integration | 1-2 days |
| 6-7 BFF/admin retirement | 4-8 days |
| 8 runtime/test cleanup | 3-5 days |
| 9 assets/docs/CI | 2-4 days |
| 10 DB roles (独立授权) | 1-2 days |
| 11 final review | 0.5-1 day |

总计约24-39 engineer-days；新增工作量来自durable raw receipt schema/transaction/race/status实现、forward migration evidence与独立local integration层；外部`agent-run`、真实UAT/prod rollout、multi-module/repo split不计入。
