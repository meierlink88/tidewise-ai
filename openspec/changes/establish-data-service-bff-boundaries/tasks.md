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

## 1. Proposal Review Package

- [x] 1.1 已审阅并批准初始 proposal/design/specs/tasks 的 Data/Miniapp/Admin ownership、目标目录、HTTP contract、non-goals、采集停止窗口和`agent-run`交接边界；该批准未授权数据库、Neo4j、部署、Sync、Archive或Deliver。
- [x] 1.2 确认四项冻结决策：Data API namespace 为 `/internal/data/v1`；client 为小型受控手写 typed client + OpenAPI drift test；scheduler API 为认证后一个实际部署窗口的 `410 Gone` tombstone；raw-document batch 为有界 whole-batch validation + atomic write。
- [x] 1.3 记录用户于2026-07-17明确批准新增forward-only `backend/migrations/000022_add_raw_document_import_receipts.sql`，并授权在amendment获批且Package 5 order 1 evidence全部精确通过后自动对local PostgreSQL执行一次该migration；该授权不含本轮SQL、其他migration、seed、业务写入、Neo4j、环境、部署或Package 10权限层。
- [x] 1.4 审阅本amendment checkpoint，确认11-package Gate Map、raw receipt不可变表/事务/status合同、历史21 migration hash、Package 5两层fail-closed计划与Package 10独立性；批准前不得恢复Apply或创建`000022`文件。

## 2. Inventory And Architecture Package

- [x] 2.1 冻结删除前 manifest：精确 production files/symbols、caller/import/reference、三张历史 scheduler tables/21 migrations、保留 repositories/connectors/import、114 Go test files/541 Go tests、frontend 8 test files/44 cases（Admin 7/26、Miniapp 1/18）与 13 个 testdata files。
- [x] 2.2 先写 RED architecture/reference tests：Miniapp/Admin 不得 import Data DB/repository/domain/migration 或 connector/parser；旧 scheduler/runtime/health/commands/config/caller 不得在退役后出现；platform 不得拥有业务 DTO/repository/client。
- [x] 2.3 对 `backend/testdata`、package fixtures、`backend/data/source_catalogs`、prompts 做引用审计；区分 testdata 与版本化业务 asset，当前无零引用 fixture 可安全删除的假设必须由命令输出证明。
- [x] 2.4 记录 package baseline、输入 commit `origin/main=3f0f779d2c332a74f31fd398adb47adb306a60c3` 与 scope/secret checks；manifest 漂移先停下更新 design，不扩大删除范围。

## 3. Service Skeleton Package

- [x] 3.1 建立 Data、Miniapp、Admin service-owned entry/facade/health，入口只负责 config、依赖组装和启动；保持单 `backend/go.mod`，旧 `cmd/api`/`cmd/admin-api` 只作薄 compatibility wiring。
- [x] 3.2 让 Data Service 拥有 Data domain/repository/migration/projection；BFF 仅拥有 channel DTO/权限/编排；不为目录美观 bulk move 全部 domain/repositories/data。
- [x] 3.3 运行三个 binary build、architecture targeted suites、`git diff --check` 与 reference/scope checks；失败时回退 facade wiring，不触碰数据。

## 4. Data API/Import Code And Migration Artifact Package

- [x] 4.1 先写OpenAPI/handler/手写typed client tests，固定`/internal/data/v1`、UUID/UTC/enums、bounded batch、structured error/request-id、timeout、service identity和retryability，并以OpenAPI drift test防止contract/client漂移。
- [x] 4.2 先以contract tests定义当前Research页面、Admin raw/event/source aggregates，以及`agent-run`只读source-metadata endpoint；后者使用独立scope和脱敏DTO，只返回批准非敏感配置/`credential_ref`名称，不返回provider secret。
- [x] 4.3 先写raw import RED tests：认证principal派生`caller_identity`及1..200 key边界、`raw-document-import-v1` canonical lowercase SHA-256 test vectors、现有`NormalizeUUID("raw_document_import_receipt",caller,key)`确定性receipt ID、caller-scoped幂等、required/cross-field-consistent original result、status existing=`completed`/missing=`unknown`、same caller+key changed hash=`409`且无mutation。
- [x] 4.4 只新增forward-only `backend/migrations/000022_add_raw_document_import_receipts.sql` artifact：精确七列、named PK/unique/checks/index、固定function与拒绝UPDATE/DELETE/TRUNCATE的statement trigger；Goose Up默认single transaction且禁止`NO TRANSACTION`/`IF NOT EXISTS`/`OR REPLACE`，Down必须`RAISE EXCEPTION`失败。不得改21个历史文件、seed既有表、创建event/source/tag/review placeholder、执行migration或连接数据库。
- [x] 4.5 实现whole-batch validation-before-DML与一个PostgreSQL transaction：先caller+key advisory xact lock/receipt lookup，receipt miss才做可变source/business validation；length-prefixed sorted raw external-ID/content-hash locks、separate identity queries、converged reuse/divergent 409、`ON CONFLICT DO NOTHING`+winner re-read覆盖cross-key/cross-caller overlap；全部resolution后再次要求resolved IDs数量等于candidates且唯一，折叠冲突409 rollback。raw documents与immutable receipt同生共死，不得部分item成功、内存receipt、jobs/runs或generic polymorphic receipt table。
- [x] 4.6 保留并暴露reviewed-event import：canonical payload hash、review状态、event/source/tag/raw/`event_import_receipts`单transaction、同key同hash重放与不同hash conflict；raw receipt与event receipt repository/API/schema/tests完全分离。
- [x] 4.7 为Miniapp/Admin提供本地`DataServiceClient` ports、手写HTTP adapter与fake；覆盖timeout、connection failure、4xx/409/5xx、有限retry、request-id/trace和日志脱敏；contract未通过禁止删除runtime。
- [x] 4.8 保持`event-import` CLI dry-run/input/machine JSON/exit code；非dry-run迁移期只能作为受控Data client/maintenance adapter，不能绕过service ownership。
- [x] 4.9 提交Package 4 checkpoint evidence：repo migration files从21变22、前21聚合hash仍为`2ed0dd004ab3b0bad633af5f0107a9ddffa28b83422b643f821c2cd47fb02dfc`、记录`000022`exact SHA-256、migration/static/API/repository/race/rollback unit tests及strict/scope/secret checks；明确未执行SQL。

## 5. Local Raw Receipt Schema Apply And Integration Package

- [x] 5.1 执行order 1 `local-raw-receipt-schema-preflight`：用server-enforced`BEGIN READ ONLY`直接核验local identity/scope与既有Goose ledger；ledger缺失即停，禁止运行可能`EnsureDBVersion`写入的dbmigrate check mode。确认applied=21、repo仅`000022`pending、排除new file后的21-file聚合hash、new file exact hash/transactional Up/failing Down、全部固定对象名不存在、DDL/ledger privilege及business schema/data counts；生成完整可读且有hash/documented recovery的clean backup evidence但不声称restore-tested，任一不一致立即停止。
- [x] 5.2 只有amendment已获批准且5.1 evidence全部精确通过时，依据2026-07-17已记录的用户条件式授权自动进入order 2 `local-raw-receipt-schema-apply`，从`backend/`仅执行一次`go run ./cmd/dbmigrate -apply -target-version 000022`；不得使用无target apply、再次执行、扩大pending集合、seed、持久化业务数据、触碰Neo4j/UAT/prod/shared/deploy/roles/credentials。
- [x] 5.3 apply后只读断言migration count=22、前21 hash不变、`000022`已应用一次、七列类型/default、named PK/unique/checks/index/function/statement trigger与API contract完全一致、初始receipt rows=0；除Goose ledger新增`000022`及列明新对象外，pre-existing business schema/data counts/hash不变。
- [x] 5.4 在已应用schema上运行可全部rollback的raw receipt PostgreSQL integration：atomic failure rollback、valid insert/read/cross-field result、constraint rejection、savepoint内UPDATE/DELETE/TRUNCATE拒绝、两connection advisory-lock阻塞/释放与sorted overlapping raw-identity guard；两connection都rollback且结束rows=0。winner commit→loser replay由Package 4 state-machine/SQL winner-re-read tests覆盖，不得在curated local伪称真实commit-race或为其持久化fixture。
- [x] 5.5 记录两层命令、输入指纹、backup hash、before/after query输出和targeted tests；apply/断言/测试任一失败或出现partial state时立即停止并报告，不得restore、retry、forward-fix、切换credential或继续后续package。

## 6. Miniapp BFF Decoupling Package

- [x] 6.0 在恢复Package 6 Apply前取得本Research contract amendment的明确Review批准；未批准即停止，不得修改production Go/OpenAPI/client、数据库、Neo4j、部署或继续6.1-6.3。
- [x] 6.1 先修正`backend/services/data/api/openapi.yaml`、Miniapp `dataclient/port.go`与contract drift/HTTP tests、Data handler非空golden，再以现有`/api/v1/miniapp/research/*` public golden + fake Data client固定权威值域、Theme `impact_summary`/Anchor `relation_summary`两个DTO、现有JSON/cursor/排序/错误语义和每请求一个aggregate call；禁止隐式枚举映射或字段丢失。
- [x] 6.2 移除Miniapp command/config/image的PostgreSQL migration check、repository wiring、Data DB credential；保留BFF自身identity/timeout/health。
- [x] 6.3 运行Data OpenAPI/client drift与handler golden、Miniapp backend/frontend affected suites、architecture import tests和timeout/call-count assertions；不恢复repository import作为rollback，且不执行数据库/schema/migration/data操作。

## 7. Admin Retirement And Decoupling Package

- [x] 7.1 先将`GET/PUT /admin/scheduler/config`、`GET /admin/scheduler/runs`固定为认证后无DB读写的machine-readable`410 Gone` contract，并保留一个实际部署窗口；保留raw/event/source/auth/pagination行为。
- [x] 7.2 删除Admin scheduler DTO/handlers/repository contract的业务行为；前端移除`api/scheduler.ts`、`SchedulerSettings.tsx`、scheduler tab/styles，混合tests改为三tab/source/raw/event assertions；无关React transitive`scheduler` packages保留。
- [x] 7.3 让Admin通过DataServiceClient查询/编排，移除Data DB credential与migration wiring；运行Admin backend完整suite、frontend test/build、410/调用次数/timeout tests。

## 8. Scheduler/Runtime Retirement And Test Cleanup Package

- [x] 8.1 删除前冻结并复核keep/remove/replace caller/reference manifest、Package 4两类import contract coverage与Package 5 raw receipt schema/integration evidence；仍有production caller、替代contract未通过、历史21 migration hash/table data漂移或保留能力覆盖下降即停止。
- [x] 8.2 删除`backend/cmd/{ingestion-scheduler,source-ingest,ingest-smoke}/**`、`backend/internal/apps/ingestion/{scheduler,runtime}/**`与`backend/internal/apps/ingestion/health/doc.go`。不得迁移到Data Service或新建替代scheduler/worker。
- [x] 8.3 删除runtime-only scheduler tick/timezone/config、scheduler/run domain symbols、`repositories/scheduler.go`、`ingestion_run.go`与memory scheduler/run state；保留所有历史tables/rows及21个历史migrations，禁止drop/truncate/rewrite。
- [x] 8.4 只在symbol/caller证明无保留消费者后移除core的`SourceRegistry`、`RateLimiter`、`LocalRawObjectStore`、`RawDocumentWriter`；保留Connector/Parser/Registry/EnvCredentialResolver、connectors/parsers/sourcecatalog/prompt contracts与tests。
- [x] 8.5 精确清理测试：删除对应旧production的23 direct backend tests、domain 5、core 4；Admin 4改为410/无写入retirement tests；frontend scheduler-only 10删除，3个mixed cases改写；不得删有效domain/repository/connector/API/migration/idempotency/transaction/security/raw receipt tests。
- [x] 8.6 更新`infra/local/README.md`、ingestion README、`.agents/backend-boundaries.md`和command/architecture/config tests，移除不可运行命令/旧owner；CI/Docker/UAT中无scheduler service时不凭名称删除。
- [x] 8.7 运行connectors/parsers/sourcecatalog/eventimport/raw-doc/raw-receipt/migration targeted suites、受影响service suites、reference scan、testdata loader scan；重新输出before/after manifest，testdata删除预期为0且无悬空引用。

## 9. Assets, Local, CI, And Docs Package

- [x] 9.1 为Data/Miniapp/Admin建service-owned Dockerfile/CMD/health/readiness/start assets；旧`backend/Dockerfile`只有在local/CI consumers切换后才移除。
- [x] 9.2 更新`infra/local`只编排三服务、DB、Neo4j、network/observability；验证Miniapp/Admin无Data DB credential，运行compose config/dry checks，不启动真实采集或数据库写入。
- [x] 9.3 更新CI测试/build三服务、OpenAPI/client drift、architecture/reference contracts；明确`.github/workflows/deploy-uat.yml`、`infra/uat/**`、prod/shared为excluded。

## 10. Local Data DB Roles And Credential Package

- [x] 10.1 在任何权限写操作前重新提交database identity、role/grant manifest、schema owner、22-migration及业务counts/hash/schema、可恢复backup、旧credential回切、逐层命令和stop conditions，并确认raw/reviewed-event receipt均为advisory xact lock后的plain `SELECT`且无row-locking clause；该R2与Package 5 migration层完全分离，未获权限层明确批准不得继续。
- [x] 10.2 仅在Package 10授权后创建/收敛`data_service_rw`、`data_service_migrate`、`data_service_ro`，把raw receipt table/function owner转给`data_service_migrate`并收敛PUBLIC/function privilege；runtime仅必要SELECT/INSERT且不可UPDATE/DELETE/TRUNCATE，随后只读断言owner/grant manifest。任何drift/failure立即停止，不得执行migration、drop scheduler tables或改历史SQL。执行后数据库合同断言及Leader独立只读复核均通过；临时runtime credential在后续无状态路径错误退出时被清除，credential recovery/cutover仍归10.3且未执行。
- [x] 10.3 仅在上一层通过且授权仍有效时切换Data Service local credential，验证BFF/Agent无DB credential、最小权限与回切；不得执行migration/seed/import/projection/业务写入或重跑Package 5。一次process-only runtime password recovery transaction、post-contract/实际密码登录最小权限断言及单一临时Data Service health/readiness/read-only endpoint验证全部通过；进程已停止、listener消失且secret已清除。

## 11. Apply-final Review Package

- [x] 11.1 对照artifacts复核三服务拓扑、raw receipt表/transaction/status合同、精确退役/保留清单、Agent handoff、410窗口、两个独立R2状态、测试计数和fixture manifest；修复漂移。
- [x] 11.2 提交最终before/after manifest：repo migration 21→22且前21聚合hash不变、Package 4冻结的`000022`exact SHA-256与applied文件一致、local applied 21→22且`000022`一次、Package 5初始/测试后receipt rows=0；114/541 Go与frontend 8/44基线按`after=before-removed+replacement/new`逐文件解释，13 testdata预计零删除/零悬空。
- [x] 11.3 运行一次`go test ./...`、受影响前端完整test/build、三个binary/image builds、architecture/contract/reference checks、OpenAPI drift、local compose config、`openspec validate establish-data-service-bff-boundaries --strict`、显式task-design lint、artifact status、`git diff --check`、whitespace/scope/secret checks，并保留新鲜输出。
- [x] 11.4 提交scoped diff、删除/保留/替换理由、migration/schema/rollback evidence、兼容/性能/认证/事务/race/停止窗口风险和未验证项；等待Apply-final明确批准，批准前不得Sync/Archive/Deliver。
