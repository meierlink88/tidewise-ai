## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | Proposal Review | R0 | yes | SPEC_SEMANTICS | 仅批准本 change 的服务边界、API/DB ownership、迁移顺序与非目标 |
| 2 | 边界与进程骨架连续实施 | R1 | no | NONE | 允许 architecture tests、单 module 目录边界与三个 binary 归属调整 |
| 3 | Data API contract 连续实施 | R1 | no | NONE | 允许最小 OpenAPI、Data Service handler/client、fake 与 contract tests |
| 4 | BFF 去数据库依赖连续实施 | R1 | no | NONE | 允许 Miniapp/Admin 通过 DataServiceClient 编排并保持现有外部 API 行为 |
| 5 | Data jobs 归属连续实施 | R1 | no | NONE | 允许 ingestion、event-import、seed、migration、graph projector 归属 Data Service 边界 |
| 6 | service-owned assets 与 local orchestration 连续实施 | R1 | no | NONE | 仅允许 local 与 repo 内构建资产；排除 active UAT、prod/shared 和真实部署 |
| 7 | Data PostgreSQL 角色与凭据切换授权 | R2 | yes | DEPLOYMENT_SECURITY | 仅允许另行审阅后的 local Data DB roles/grants 与服务凭据切换 |
| 8 | Apply-final Review | R1 | yes | APPLY_FINAL | 仅验收 scoped diff、兼容性、验证证据和未验证风险 |

## Complexity Budget

| Key | Value |
|---|---|
| human_gates | 3 |
| stateful_layers | 2 |
| checkpoints | 3 |
| full_test_runs | 1 |
| continuous_automation_scope | packages:2-6 |

## Stateful Layer Map

| Layer | Package | Environment | Order | Scope | Exclusions | Recovery Evidence | Recovery Baseline | Expected Counts/Hash/Schema | Before Assertions | After Assertions | Stop Conditions |
|---|---|---|---|---|---|---|---|---|---|---|---|
| local-data-db-role-boundary | 7 | local | 1 | 在已确认的 local Data PostgreSQL 中创建或收敛 `data_service_rw`、`data_service_migrate`、`data_service_ro` roles 与最小 grants | UAT、prod/shared、业务表结构与业务数据、跨数据库角色 | backup | new:local-data-db-security-baseline | counts=review-gated;hash=review-gated;schema=role-grant-manifest-v1 | 已复验 local 数据库 identity、现有 roles/grants、schema owner、连接计数、manifest hash，并验证可恢复 backup | 三个 Data roles 的 owner/grants 与批准 manifest 一致，业务 schema/data counts/hash 不变 | 环境、role、grant、owner、count/hash/schema 漂移，备份不可恢复或权限断言失败立即停止 |
| local-service-credential-cutover | 7 | local | 2 | 将 local Data Service 切换到批准的 Data PostgreSQL role，并确认 Miniapp/Admin/Agent 无 Data DB 凭据 | UAT、prod/shared、secret 入库、业务数据写入和 migration 执行 | backup | reuse:local-data-db-security-baseline | counts=review-gated;hash=review-gated;schema=role-grant-manifest-v1 | 复验 local identity、scope、role/grant counts、manifest hash、schema 与 security baseline 未漂移，旧凭据回切方式可用 | Data Service 最小读写与 migration 职责通过只读/受控验证，Miniapp/Admin/Agent 启动配置不含 Data DB 凭据 | credential scope、identity、count/hash/schema 漂移，健康检查、权限或回切验证失败立即停止 |

## 1. Proposal Review Package

- [ ] 1.1 审阅并明确批准 proposal、design、五个 delta specs 与本 tasks 的 Data/Miniapp/Admin ownership、目标目录、禁止依赖、迁移兼容策略和 non-goals；该批准只允许进入 Packages 2-6 的 R1 Apply，不授权 Package 7、任何数据库/Neo4j/部署写入、Sync、Archive 或 Deliver。
- [ ] 1.2 固定首批 Data API namespace、service identity 方式、client 生成/手写策略、`cmd/api`/`cmd/admin-api` compatibility window，以及首批只覆盖现有 Research/Admin/Agent consumers 的 API surface；未决项必须在 Apply 前更新 artifacts 并重新 Review。

## 2. Boundary And Binaries Package

- [ ] 2.1 按 TDD 先增加 architecture tests，验证 Miniapp/Admin 不得 import Data domain/application/repository、database/migration 或 ingestion connector/parser，platform 不得拥有业务 DTO/repository/client method，三个 service command 不得承载业务逻辑。
- [ ] 2.2 建立 Data、Miniapp、Admin Portal 的 service-owned package/facade 与三个可编译 HTTP binary；先复用现有实现并保留薄 compatibility entrypoints，不 bulk move `internal/domain`、`internal/repositories`、`migrations` 或 `data`。
- [ ] 2.3 明确单一 `backend/go.mod`、Data-owned legacy paths 与 compatibility adapter 的删除条件；增加结构/导入测试，防止新代码继续写入无 owner 的共享业务层。
- [ ] 2.4 运行 architecture targeted suites、三个 binary build、受影响 package tests、`git diff --check` 与 scope/secret checks，记录输入 commit 和 Packages 2 连续执行证据。

## 3. Data API Contract Package

- [ ] 3.1 先编写 OpenAPI schema/contract tests，固定内部版本 namespace、UUID/UTC/enums、cursor pagination、结构化 error/request id、timeout、service identity、idempotency 与向后兼容规则。
- [ ] 3.2 先以 handler/use-case tests 定义当前 Research page aggregates、Admin raw/event/source/scheduler aggregates/commands 与 Agent reviewed-outbox import 的最小 Data API；明确每页有界调用次数，禁止无消费者 CRUD。
- [ ] 3.3 实现 Data Service HTTP adapters 和 repository-backed application wiring，使聚合 command 在单一 Data PostgreSQL transaction 内完成；不得暴露 repository models 或执行真实 database writes 作为测试前提。
- [ ] 3.4 在 Miniapp/Admin 各自定义本地 `DataServiceClient` port，提供 production HTTP client 与单元测试 fake；若使用生成 client，增加 OpenAPI drift check，生成业务 DTO 不得进入 platform。
- [ ] 3.5 覆盖网络 timeout、connection failure、4xx/409/5xx、有限 retry、同 key 同/different hash、service identity、request id/trace propagation 与日志脱敏测试，并运行 Data API/clients 的完整 package suites。

## 4. BFF Database Decoupling Package

- [ ] 4.1 先用现有 `/api/v1/miniapp/research/*` golden/contract tests 和 fake Data client 固定页面 DTO、cursor、错误与聚合行为，再把 `miniappapi` 的 repository/model 依赖替换为本地 Data client port。
- [ ] 4.2 先用现有 `/admin/*` contract tests 和 fake Data client 固定 auth、pagination、query/update 行为，再把 `adminapi` 的 scheduler/raw/event/source repository 依赖替换为本地 Data client port。
- [ ] 4.3 从 Miniapp/Admin command/config/image contract 中移除 PostgreSQL migration check、repository wiring 与 Data DB credential requirement；保留 BFF 自身 config、service identity、timeout 和 health/readiness。
- [ ] 4.4 运行 miniapp/admin backend suites、前端 API contract tests、architecture import tests 与调用次数/timeout assertions，确认两个 BFF 平行且外部 API 无 breaking behavior。

## 5. Data Jobs Ownership Package

- [ ] 5.1 先以 command/architecture tests 固定 ingestion scheduler、source ingest/smoke/seed、entity seed、dbmigrate、event import 与 graph projector 的 CLI、exit code、transaction/owner 行为，再标记或渐进迁入 Data Service ownership。
- [ ] 5.2 将 event import application/transaction 明确归 Data Service；保留 CLI input/dry-run/machine JSON/exit code 兼容，使生产 Agent path 使用受控 HTTP import API，CLI 非 dry-run 只作为明确记录 mode 的 compatibility/maintenance adapter。
- [ ] 5.3 保持 `backend/migrations`、版本化 seed/data 和 Data PostgreSQL tables 原位且归 Data Service；不得重写历史 migration、搬表、拆 schema、执行 migration/seed 或写业务数据。
- [ ] 5.4 保持 graph projector 为 Data-owned job、PostgreSQL 为事实源、Neo4j 为可重建投影；运行 fake/fixture suites，不执行 Neo4j cleanup/rebuild/project。
- [ ] 5.5 运行各 command/app targeted suites、architecture tests、event import contract/idempotency tests 与 scope/secret checks，确认没有 Agent/BFF direct DB path。

## 6. Service Assets And Local Orchestration Package

- [ ] 6.1 先写构建与 health contract tests，为 Data、Miniapp、Admin 各自建立 service-owned Dockerfile/CMD/health/readiness/startup assets；旧 `backend/Dockerfile` 只在新 assets 与 local/CI consumers 切换后移除。
- [ ] 6.2 更新 `infra/local` 仅编排三服务、PostgreSQL、Neo4j、网络与观测，验证 Miniapp/Admin 不注入 Data DB credential，Data Service 使用唯一 Data DB runtime credential；只运行 compose config/dry checks，不启动或写入真实环境。
- [ ] 6.3 更新 `.github/workflows/ci.yml` 以测试/build 三个服务、OpenAPI/client drift 与 architecture contracts；根据 path/affected boundary 选择 targeted jobs，并保留 Apply-final 一次完整验证。
- [ ] 6.4 明确排除 `.github/workflows/deploy-uat.yml`、`infra/uat/**`、prod/shared 与真实部署；等待 `migrate-uat-to-linux-amd64` Deliver 后另开或更新已审阅 plan 处理三服务 UAT rollout。
- [ ] 6.5 运行三个镜像 build、health contract、local compose config、CI assertions、受影响前端 suites/build 与 scope/secret checks，记录 service-owned/cross-service asset ownership。

## 7. Local Data DB Security Package

- [ ] 7.1 在任何写操作前重新提交独立 R2 authorization package：精确 local database identity、role/grant manifest、schema owner、现有连接、预计 counts/hash/schema、可恢复 backup 证据、旧凭据回切、逐层命令与停止条件；未获明确批准不得继续。
- [ ] 7.2 经授权后只执行 `local-data-db-role-boundary`：复验 baseline 后创建/收敛 `data_service_rw`、`data_service_migrate`、`data_service_ro` 与最小 grants，立即只读查询/断言；任何 drift/failure 立即停止并使下一层授权失效。
- [ ] 7.3 仅在 7.2 全部断言通过且授权仍有效时执行 `local-service-credential-cutover`：复验同一 baseline identity/scope/count/hash/schema，切换 Data Service local credential，验证 Data 最小权限与 BFF/Agent 无 DB credential；失败立即回切并停止。
- [ ] 7.4 记录 before/after role/grant manifest、业务 schema/data counts/hash 不变、backup/recovery 与回切证据；不得扩展到 migration、seed、业务写入、UAT、prod/shared 或 secret 入库。

## 8. Apply-final Review Package

- [ ] 8.1 对照 proposal/design/delta specs 复核三服务拓扑、精确迁移映射、API/DB ownership、platform/infra 边界、兼容窗口、Package 7 结果或延期状态，修复 artifacts/implementation 漂移。
- [ ] 8.2 运行一次受影响交付边界完整验证：`go test ./...`、三个 binary/image builds、共享 architecture/contract tests、受影响 miniapp/admin suites/build、OpenAPI/client drift、local compose config、`openspec validate establish-data-service-bff-boundaries --strict`、`git diff --check`、scope 与 secret checks，并读取新鲜输出。
- [ ] 8.3 完成 self-review/code review，提交 scoped diff、验证证据、性能/网络/认证/事务/CI/CD/rollout 风险、未验证项、blockers 与 rollback；等待用户 Apply-final 明确批准，批准前不得 Sync、Archive、Deliver、修改 UAT/prod/shared 或创建完成态 PR。
