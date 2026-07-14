# Active change workflow adoption Review

## Adoption record

- source workflow change：`optimize-risk-tiered-development-workflow`，已 Deliver；`origin/main` merge commit 为 `4b3df5c`。
- current change：`refactor-industry-chain-node-foundation`，Desktop-managed worktree 与 `codex/refactor-industry-chain-node-foundation` branch 保持不变。
- 已执行 `git fetch origin`，确认 `origin/main@4b3df5c`，并通过 merge commit `2256c00` 非破坏性吸收最新主分支；未重写、删除或覆盖本 change checkpoint。
- 本 adoption diff 只修改当前 change 的未来流程标注；1.1—1.12 的 checkbox、Review 结果与授权保持不变。主对话已批准 checkpoint `0b21f74`、task 1.13 Cleanup Readiness Review，并已验收 task 1.14 cleanup execution checkpoint `f2bc90a`；后续 schema/seed/mapping/relation 层仍须独立授权。

## Package scope

- 风险等级：**R0**，因为只修改 OpenSpec artifacts 并执行 scoped validation。
- scope：声明 change R3 基线、映射剩余 tasks 到 R0/R1/R2/R3、定义阶段 Review package、R2 命名条件式执行包和 R3 cleanup 独立授权。
- non-goals：不运行 restore rehearsal，不重复 migration 15，不运行 migration 16、seed、cleanup、mapping、relation，不连接或写入 Neo4j，不 rebuild，不修改主规格。
- 未验证项：所有未来执行包的环境快照、recovery evidence、预计 counts 与 before/after assertions 必须在各包提交时刷新；本 adoption 不复用旧证据作为未来授权。
- 当前阻断：`backup_verified=false` 保持历史事实；当前下一入口是独立 `phase-a-external-identifier-schema` R2 package，尚未授权或执行。

## 剩余阶段映射

| 范围 | 风险 | Review / Authorization package | 通过后允许 | 明确不授权 |
|---|---|---|---|---|
| workflow adoption | R0 | 本文件、tasks/design diff、OpenSpec validate、scope/secret/diff 检查 | adoption Review 后按新语义准备未来 package | 任何 DB/Neo4j 操作；改变历史状态 |
| 1.13 | R0，已批准 | Cleanup Readiness Review package | 准备独立 restore rehearsal authorization package | restore、cleanup Write、migration 16、seed、mapping、Neo4j |
| restore rehearsal | R0 artifact / R2 execution | `phase-a-backup-restore-rehearsal` 独立条件式执行包 | 获独立授权后只在隔离 disposable PG16 恢复并验证 backup；成功升级 `backup_verified` | migration 15、cleanup、seed、现有 PG/Neo4j Write |
| 1.14 | R3，已验收 `f2bc90a` | `phase-a-legacy-industry-cleanup` 独立授权包 | 已完成 migration 15 与立即 Query/assert | migration 16、seed、mapping、relation、Neo4j |
| 1.15 | R2，Write/Query 已完成待验收 | `phase-a-external-identifier-schema` 条件式执行包 | 已完成 migration 16 schema Write 与立即 Query/assert | 1.16、node/profile、mapping、relation、Neo4j |
| 1.16 | R0 | Final seed candidate Review package | 组装 node/profile R2 执行包 | seed Write |
| 1.17 | R2 | `phase-a-chain-node-seed` 条件式执行包 | 842 node/profile Write 后立即 Query/assert | mapping data、theme、relation |
| 1.18 | R0→R2 | Mapping candidate Review；随后 `phase-a-external-identifier-mapping` 条件式执行包 | 1,156 mapping Write 后立即 Query/assert | 未审阅 mapping、其他实体 mapping |
| 1.19 | R0 | Phase A Acceptance Review package | 全部 pre/post evidence 验收后进入 Phase B R1 package | 自动授权 Phase B Write |
| 2.2—2.5 | R1 | relation contract/implementation/tests/candidate Review package | package 验收后准备 relation schema R2 包 | schema/data Write、Neo4j |
| 2.6 | R2 | `phase-b-relation-schema` 条件式执行包 | relation schema Write 后立即 Query/assert | relation/constraint data |
| 2.7 | R2 | `phase-b-relation-data` 条件式执行包 | 已审阅 relation/新 constraint data Write 后立即 Query/assert | 旧 edge 迁移、theme link、Neo4j |
| 2.8—2.9 | R1 Review | Apply-final Review package，聚合 R3/R2 实际证据 | 人工验收后才可进入 Sync | 自动 Sync/Archive/Deliver |

## R2 条件式执行包统一契约

每个命名 R2 包必须逐层提交并由用户明确授权：环境身份、顺序、精确范围、排除范围、`backup` 或逐层批准的 `approved disposable recovery`、预计 counts、before/after assertions、停止条件和写后 Query/assert。curated local PostgreSQL 不自动视为 disposable。范围漂移、recovery evidence 无效、断言失败或停止条件触发时 fail-closed；当前层停止，包内尚未执行的授权自动失效，重试必须重新授权。

本 change 采用一层一包，保持既有顺序与人工 Review：schema、node/profile、mapping、relation schema、relation data 不互相推定授权。R3 cleanup 绝不进入 R2 包。

## Verification boundary

本 checkpoint 只修改当前 change 的 OpenSpec artifacts，属于 R0；运行 OpenSpec strict validation、scoped diff、`git diff --check` 与 secret scan。未修改源码、共享规则或 architecture tests，因此不触发 targeted Go tests、Apply-final suite 或 repo-wide `go test ./...`。

## Review request

主对话已验收本 active-change workflow adoption、task 1.13 与 task 1.14。当前下一授权入口仅为 [phase-a-external-identifier-schema-authorization.md](phase-a-external-identifier-schema-authorization.md)；既有批准不授权 migration 16 或任何后续有状态操作。
