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

## 1. Proposal Review Package

- [ ] 1.1 审阅并批准 proposal/design/specs/tasks 的 Data/Miniapp/Admin ownership、目标目录、HTTP contract、non-goals、采集停止窗口和 `agent-run` 交接边界；只允许进入 Packages 2-8，不授权数据库、Neo4j、部署、Sync、Archive 或 Deliver。
- [ ] 1.2 确认四项冻结决策：Data API namespace 为 `/internal/data/v1`；client 为小型受控手写 typed client + OpenAPI drift test；scheduler API 为认证后一个实际部署窗口的 `410 Gone` tombstone；raw-document batch 为有界 whole-batch validation + atomic write。

## 2. Inventory And Architecture Package

- [ ] 2.1 冻结删除前 manifest：精确 production files/symbols、caller/import/reference、三张历史 scheduler tables/21 migrations、保留 repositories/connectors/import、114 Go test files/541 Go tests、frontend 8 test files/44 cases（Admin 7/26、Miniapp 1/18）与 13 个 testdata files。
- [ ] 2.2 先写 RED architecture/reference tests：Miniapp/Admin 不得 import Data DB/repository/domain/migration 或 connector/parser；旧 scheduler/runtime/health/commands/config/caller 不得在退役后出现；platform 不得拥有业务 DTO/repository/client。
- [ ] 2.3 对 `backend/testdata`、package fixtures、`backend/data/source_catalogs`、prompts 做引用审计；区分 testdata 与版本化业务 asset，当前无零引用 fixture 可安全删除的假设必须由命令输出证明。
- [ ] 2.4 记录 package baseline、输入 commit `origin/main=3f0f779d2c332a74f31fd398adb47adb306a60c3` 与 scope/secret checks；manifest 漂移先停下更新 design，不扩大删除范围。

## 3. Service Skeleton Package

- [ ] 3.1 建立 Data、Miniapp、Admin service-owned entry/facade/health，入口只负责 config、依赖组装和启动；保持单 `backend/go.mod`，旧 `cmd/api`/`cmd/admin-api` 只作薄 compatibility wiring。
- [ ] 3.2 让 Data Service 拥有 Data domain/repository/migration/projection；BFF 仅拥有 channel DTO/权限/编排；不为目录美观 bulk move 全部 domain/repositories/data。
- [ ] 3.3 运行三个 binary build、architecture targeted suites、`git diff --check` 与 reference/scope checks；失败时回退 facade wiring，不触碰数据。

## 4. Data Import/API Contract Package

- [ ] 4.1 先写 OpenAPI/handler/手写 typed client tests，固定 `/internal/data/v1`、UUID/UTC/enums、bounded batch、structured error/request-id、timeout、service identity 和 retryability，并以 OpenAPI drift test 防止 contract/client 漂移。
- [ ] 4.2 先以contract tests定义当前Research页面、Admin raw/event/source aggregates，以及`agent-run`只读source-metadata endpoint；后者使用独立scope和脱敏DTO，只返回批准非敏感配置/`credential_ref`名称，不返回provider secret。
- [ ] 4.3 定义并实现 raw-document batch import：来源归因、external ID/content hash/稳定 UUID 幂等、whole-batch validation/atomic transaction、receipt/result；不得自动创建 event/tag/research。
- [ ] 4.4 保留并暴露 reviewed-event import：canonical payload hash、review 状态、event/source/tag/raw/receipt 单 transaction、同 key 同 hash 重放与不同 hash conflict。
- [ ] 4.5 为 Miniapp/Admin 提供本地 `DataServiceClient` ports、手写 HTTP adapter 与 fake；覆盖 timeout、connection failure、4xx/409/5xx、有限 retry、request-id/trace 和日志脱敏；contract 未通过禁止删除 runtime。
- [ ] 4.6 保持 `event-import` CLI dry-run/input/machine JSON/exit code；非 dry-run 迁移期只能作为受控 Data client/maintenance adapter，不能绕过 service ownership。

## 5. Miniapp BFF Decoupling Package

- [ ] 5.1 以现有 `/api/v1/miniapp/research/*` golden/contract tests + fake Data client 固定 DTO、cursor、错误和有界调用次数。
- [ ] 5.2 移除 Miniapp command/config/image 的 PostgreSQL migration check、repository wiring、Data DB credential；保留 BFF 自身 identity/timeout/health。
- [ ] 5.3 运行 Miniapp backend/frontend affected suites、architecture import tests 和 timeout/call-count assertions；不恢复 repository import 作为 rollback。

## 6. Admin Retirement And Decoupling Package

- [ ] 6.1 先将 `GET/PUT /admin/scheduler/config`、`GET /admin/scheduler/runs` 固定为认证后无 DB 读写的 machine-readable `410 Gone` contract，并保留一个实际部署窗口；保留 raw/event/source/auth/pagination 行为。
- [ ] 6.2 删除 Admin scheduler DTO/handlers/repository contract 的业务行为；前端移除 `api/scheduler.ts`、`SchedulerSettings.tsx`、scheduler tab/styles，混合 tests 改为三 tab/source/raw/event assertions；无关 React transitive `scheduler` packages 保留。
- [ ] 6.3 让 Admin 通过 DataServiceClient 查询/编排，移除 Data DB credential 与 migration wiring；运行 Admin backend 完整 suite、frontend test/build、410/调用次数/timeout tests。

## 7. Scheduler/Runtime Retirement And Test Cleanup Package

- [ ] 7.1 在删除前复核 caller/reference manifest 与两类 import contract；停止条件为仍有 production caller、未通过替代 contract、历史 migration/table diff 或保留能力覆盖下降。
- [ ] 7.2 删除 `backend/cmd/{ingestion-scheduler,source-ingest,ingest-smoke}/**`、`backend/internal/apps/ingestion/{scheduler,runtime}/**` 与 `backend/internal/apps/ingestion/health/doc.go`。不得迁移到 Data Service 或新建替代 scheduler/worker。
- [ ] 7.3 删除 runtime-only scheduler tick/timezone/config、scheduler/run domain symbols、`repositories/scheduler.go`、`ingestion_run.go` 与 memory scheduler/run state；保留所有历史 tables/rows/migrations，禁止 drop/truncate/rewrite。
- [ ] 7.4 只在 symbol/caller 证明无保留消费者后移除 core 的 `SourceRegistry`、`RateLimiter`、`LocalRawObjectStore`、`RawDocumentWriter`；保留 Connector/Parser/Registry/EnvCredentialResolver、connectors/parsers/sourcecatalog/prompt contracts 与 tests。
- [ ] 7.5 精确清理测试：删除对应旧 production 的 23 direct backend tests、domain 5、core 4；Admin 4 改为 410/无写入 retirement tests；frontend scheduler-only 10 删除，3 个 mixed cases 改写；不得删有效 domain/repository/connector/API/migration/idempotency/transaction/security tests。
- [ ] 7.6 更新 `infra/local/README.md`、ingestion README、`.agents/backend-boundaries.md` 和 command/architecture/config tests，移除不可运行命令/旧 owner；CI/Docker/UAT 中无 scheduler service 时不凭名称删除。
- [ ] 7.7 运行 connectors/parsers/sourcecatalog/eventimport/raw-doc/migration targeted suites、受影响 service suites、reference scan、testdata loader scan；重新输出 before/after manifest，testdata 删除预期为 0 且无悬空引用。

## 8. Assets, Local, CI, And Docs Package

- [ ] 8.1 为 Data/Miniapp/Admin 建 service-owned Dockerfile/CMD/health/readiness/start assets；旧 `backend/Dockerfile` 只有在 local/CI consumers 切换后才移除。
- [ ] 8.2 更新 `infra/local` 只编排三服务、DB、Neo4j、network/observability；验证 Miniapp/Admin 无 Data DB credential，运行 compose config/dry checks，不启动真实采集或数据库写入。
- [ ] 8.3 更新 CI 测试/build 三服务、OpenAPI/client drift、architecture/reference contracts；明确 `.github/workflows/deploy-uat.yml`、`infra/uat/**`、prod/shared 为 excluded。

## 9. Local Data PostgreSQL Security Package

- [ ] 9.1 在任何写操作前重新提交 database identity、role/grant manifest、schema owner、counts/hash/schema、可恢复 backup、旧 credential 回切、逐层命令和 stop conditions；未获明确批准不得继续。
- [ ] 9.2 仅授权后创建/收敛 `data_service_rw`、`data_service_migrate`、`data_service_ro`，只读断言；任何 drift/failure 立即停止。不得 drop scheduler tables 或改历史 migrations。
- [ ] 9.3 仅在上一层通过且授权仍有效时切换 Data Service local credential，验证 BFF/Agent 无 DB credential、最小权限与回切；不得执行 migration/seed/import/projection/业务写入。

## 10. Apply-final Review Package

- [ ] 10.1 对照 artifacts 复核三服务拓扑、精确退役/保留清单、Agent handoff、410 窗口、R2 状态、测试计数和 fixture manifest；修复漂移。
- [ ] 10.2 运行一次 `go test ./...`、受影响前端完整 test/build、三个 binary/image builds、architecture/contract/reference checks、OpenAPI drift、local compose config、`openspec validate establish-data-service-bff-boundaries --strict`、`git diff --check`、scope/secret checks，并保留新鲜输出。
- [ ] 10.3 提交 scoped diff、删除/保留/替换理由、before/after test count、zero dangling testdata、兼容/性能/认证/事务/停止窗口风险、rollback 和未验证项；等待 Apply-final 明确批准，批准前不得 Sync/Archive/Deliver。
