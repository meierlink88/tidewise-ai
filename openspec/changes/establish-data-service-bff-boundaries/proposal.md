## Why

当前 `cmd/api`、`cmd/admin-api`、`internal/apps/miniappapi` 与 `internal/apps/adminapi` 都能直接组装或调用 Data PostgreSQL repository，Agent reviewed-outbox 也通过本地 CLI 直写同一数据库；这使部署进程看似分离，但数据所有权、API ownership 和故障边界仍是共享模块化单体。随着 Research Theme/Anchor、Event Import、图谱投影和管理后台能力增长，需要先建立 Miniapp BFF、Admin Portal BFF 与 Data Service 的最小可执行边界，避免继续扩大跨入口共享 repository。

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

## What Changes

- 在 monorepo 和单一 `backend/go.mod` 内建立三条可部署服务边界：Data Service、Miniapp Service/BFF、Admin Portal Service/BFF；Miniapp 与 Admin 平行，原则上不互调。
- Data Service 拥有 Entity、Chain Node、Raw Document、Event、Event Tag、Research Theme/Anchor、Index，以及 PostgreSQL、Neo4j/向量投影相关领域与 repository；Miniapp/Admin 不得 import Data 的 domain/application/repository 内部包，也不得持有 Data DB 凭据。
- 生产跨 Service 统一使用 HTTP REST + JSON + OpenAPI；Service 内部继续使用 Go interface/方法调用。Miniapp/Admin 通过本地 `DataServiceClient` interface 编排，生产注入 HTTP client，测试注入 fake。
- Data Service 提供页面级聚合 API 和 Agent 受控导入 API，并拥有 DTO、版本、分页、错误、超时、幂等、service identity、request id/trace 与兼容策略；不引入 gRPC、Kitex、Kratos、服务注册中心、Service Mesh、Kafka 或分布式事务。
- 采用渐进迁移：先固化依赖禁令与 contract，再建立三个 binary/目录归属，随后迁移 Data API/client、解除 BFF 直接 repository/DB 依赖，最后收敛 data jobs 与 deploy assets；现有 miniapp/admin 外部 API 行为保持兼容。
- `platform` 只保留 config/logging/observability/httpclient/database bootstrap 等无业务技术能力；Event/Research/Entity DTO、repository、领域规则和业务 client 方法必须留在服务所有边界。
- 每个服务的 Dockerfile、健康检查与启动配置跟随服务目录；根 `infra` 仅保留跨服务 local/UAT/prod 环境编排、网络、数据库和观测配置。本 change 不修改或运行 UAT/prod/shared 环境，且在 active `migrate-uat-to-linux-amd64` 完成前不触碰其文件所有权。
- PostgreSQL 当前整体归 Data Service 独占，不搬表、不拆 schema、不重写既有 migration；实际 Data DB role/grant/credential 切换作为独立 R2 package，必须再次明确授权并具备 recovery evidence。
- 评估 multi-module 触发条件，但本 change 不新增 `go.mod`；只有独立版本/发布、团队、依赖或构建冲突成为真实约束时，才另开 change。

## Capabilities

### New Capabilities

- `backend-service-boundaries`: 定义 Data Service、Miniapp BFF、Admin Portal BFF 的运行时拓扑、依赖禁令、HTTP/OpenAPI 通信、API/DB ownership、platform/infra 边界和渐进迁移兼容要求。

### Modified Capabilities

- `backend-subsystem-boundaries`: 将允许 API 子系统共享 `domain/repositories` 的旧规则收窄为 BFF 只能通过 Data Service API 访问 Data Domain，并明确 data jobs 的服务归属。
- `technical-architecture`: 将“共享 repository 的模块化单体”演进为 monorepo、单 Go module 下的三服务边界，并把 Agent 回写入口明确归 Data Service。
- `persistence-and-contracts`: 明确现有 PostgreSQL 的 Data Service 独占 ownership、数据库角色边界、跨服务 API contract 和未来跨领域独立数据库规则。
- `event-import-and-tag-catalog`: 将 Agent reviewed-outbox 的正式生产交互从直连数据库/仅本地 CLI 边界演进为 Data Service 受控 HTTP 导入 API，同时保留迁移期 CLI contract 与幂等语义。

## Impact

- 受影响区域：`backend/cmd/*`、`backend/internal/apps/*`、`backend/internal/domain`、`backend/internal/repositories`、`backend/internal/http`、`backend/internal/platform`、`backend/migrations`（只保留原位并验证 ownership，不改历史 SQL）、`frontend/miniapp`、`frontend/admin`、服务构建资产、`infra/local`、`.github/workflows/ci.yml` 与相关 OpenSpec 主规格。
- 运行时影响：最终新增 Data Service 网络 hop；Miniapp/Admin 的数据库故障将转化为 Data API 超时/错误处理，需要连接池、timeout budget、request id/trace、重试与幂等边界。
- 兼容性：现有 `/api/v1/miniapp/research/*` 与 `/admin/*` 对前端保持行为兼容；Data API 使用独立版本 namespace，不把 repository model 暴露为 transport DTO。
- 依赖与避让：不与 active `migrate-uat-to-linux-amd64` 并行修改 `.github/workflows/deploy-uat.yml`、`infra/uat/**` 或 UAT runtime；该 change Deliver 后再基于最新 `origin/main` 处理 UAT 编排适配。
- 非目标：不拆 Git repo、不切 multi-module、不新增 Identity/Membership/Billing/Subscription 服务、不修改 Agent Server repo、不创建平行业务模型、不搬表/拆 schema/重写 migration、不执行数据库/图谱/部署操作、不修改 `prototype/` 或 `doc/`。
