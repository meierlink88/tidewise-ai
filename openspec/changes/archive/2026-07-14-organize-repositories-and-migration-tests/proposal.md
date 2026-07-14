## Why

一句话目标：在不改变任何运行时行为、API 或数据库语义的前提下，把当前集中在两个超大文件中的 repository 实现和散落在 SQL 目录中的 migration 测试整理为可按业务职责定位、复用和验证的最小结构。

## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | Proposal Review 已通过且 fresh overlap audit 无共享文件后连续执行 R1 整理 | R1 | no | NONE | 仅机械拆分 `backend/internal/repositories`、迁移有效测试并删除已确认无效测试；不得改变行为或执行有状态操作 |
| 2 | Apply-final Review | R1 | yes | APPLY_FINAL | 汇总 scoped diff、测试与验证证据；通过后仅允许 Sync、Archive |
| 3 | Git completion | R0 | yes | GIT_COMPLETION | Archive checkpoint 后才允许 PR merge 与 Desktop-owned cleanup；不得扩大实现范围 |

## Complexity Budget

| Key | Value |
|---|---|
| human_gates | 2 |
| stateful_layers | 0 |
| checkpoints | 3 |
| full_test_runs | 1 |
| continuous_automation_scope | packages:1 |

## What Changes

- 保留业务小接口、共享具体 `PostgresRepository` 和共享测试 adapter `InMemoryRepository`，按 source catalog、raw document、benchmark observation、admin query/event、scheduler、ingestion run、graph projection、identity 职责组织接口、DTO、参数化 SQL、Scan/normalize 与内存实现。
- 将 `postgres.go` 限定为 `db *sql.DB`、constructor 和极少量跨业务数据库基础能力，将 `memory.go` 限定为 state、mutex 和 constructor；不创建一组业务专用 PostgreSQL 具体类型。
- 删除空 `doc.go`，审计 `NormalizeUUID` 的命名和归属；仅在行为、调用点与稳定 ID 契约保持不变时进行最小归位或重命名。
- 将 `backend/migrations` 收敛为 SQL 与 `README.md`，把仍有效的静态 contract、Goose 执行和可选 PostgreSQL integration tests 迁入现有 `backend/internal/platform/dbmigration` 测试边界；删除只保护已废止最终 schema 的测试，不削弱迁移安全契约。
- 本 change 与 `rebuild-foundation-graph-and-enrich-chain-data` 无前置依赖；fresh overlap audit 已确认双方仅有各自 OpenSpec artifacts、无共享文件和数据库写状态，本 change 可独立执行 R1 Apply。若并行期间出现真实文件重叠则立即停止并重新排序。
- 实施时先用现有测试锁定行为，再连续完成机械移动、必要修复和验证；普通文件移动、测试、修复、commit/push 不设置逐文件人工 gate。
- 本 change 仅修改 `tidewise-ai` 的 OpenSpec artifacts，Apply 获批后才可修改后端源码和测试；不修改只读的 `prototype/` 或 `doc/`。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `backend-subsystem-boundaries`: 明确共享 repository 具体 adapter、业务小接口、按职责分文件和禁止新增 ORM/codegen/framework 的组织契约。
- `persistence-and-contracts`: 明确 migration SQL 目录与统一测试边界的职责分离，以及迁移安全契约不得因移动测试而弱化。

## Impact

- 影响：`backend/internal/repositories`、`backend/internal/platform/dbmigration`、`backend/migrations/*_test.go`，以及上述两个主规格的 delta specs。
- 保持不变：业务 API、repository 对外方法集合与调用语义、PostgreSQL schema、migration SQL 语义、数据、Neo4j 投影和依赖版本。
- 不引入 GORM、sqlc、ORM、codegen、新 repository framework、新 service、新 package 或抽象平台。
- 风险等级为 R1；本 change 不执行 migration、seed、PostgreSQL/Neo4j write，也不创建 PR。
