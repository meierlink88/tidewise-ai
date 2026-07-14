## Why

当前 workflow 文本允许 R3 使用“备份/恢复或等价灾难恢复证据”，但 Stateful Layer Map 与 task-design lint 将 `approved-disposable-recovery` 硬限制为 local R2，导致以 PostgreSQL 为唯一事实源且可完整重建的 local Neo4j disposable projection 无法诚实表达。`rebuild-foundation-graph-and-enrich-chain-data` 的 explicit lint 因此仅产生 Proposal/tasks 各三条、共六条同根因 `stateful-recovery` 错误，必须用独立轻量规则 change 修正后才能继续 Review。

## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | Proposal 规则语义 Review | R1 | yes | SPEC_SEMANTICS | 仅确认 local Neo4j disposable R3 的窄例外、禁止矩阵、adoption 与验证边界 |
| 2 | workflow、lint 与 tests 连续实现 | R1 | no | NONE | 仅修改 workflow 正文、task-design lint、对应 tests/fixtures 与本 change tasks |
| 3 | Apply-final Review | R1 | yes | APPLY_FINAL | 仅审阅 scoped diff、完整验证与受阻 change 的只读 adoption 证据 |

## Complexity Budget

| Key | Value |
|---|---|
| human_gates | 2 |
| stateful_layers | 0 |
| checkpoints | 2 |
| full_test_runs | 1 |
| continuous_automation_scope | packages:2 |

## What Changes

- 保留 `approved-disposable-recovery` 对 local R2 的现有支持，并新增唯一窄例外：仅当 layer 同时是 `Environment=local`、所属 Gate `Risk=R3`、scope 明确为 Neo4j projection 的 `cleanup`、`rebuild` 或 `sync`，且 PostgreSQL baseline 已冻结、已验收并可完整重建该投影时允许使用。
- 明确 before/after assertions、stop conditions、命名 layer 的独立 R3 授权、逐层执行、失败停止与禁止跨层批量仍是强制门禁；该例外只改变 recovery evidence 的可表达性，不授权任何操作。
- 对 `shared-local`、UAT、prod、shared/生产、非 Neo4j R3、非 projection 操作或不能从 PostgreSQL 完整重建的状态继续 fail-closed，必须使用 `backup` 或等价正式灾备。
- 以 Go table-driven tests/fixtures 锁定允许与禁止矩阵，不新增依赖、列、通用 policy framework 或平行 CI job。
- 规则 Deliver 后，受阻 change 在其独立 branch/worktree 中更新到最新 `origin/main`，在自己的 Proposal/tasks 中把三个 Scope 补成明确的 `Neo4j cleanup`、`Neo4j rebuild`、`Neo4j sync` 机器锚点并经 Review 后复验 explicit lint；本 change 不直接修改其 artifacts，也不追认或执行任何 R3 layer。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `skill-driven-development-workflow`: 扩展 Stateful Layer Map 的 recovery contract，使严格受限的 local Neo4j R3 disposable projection layer 可使用 `approved-disposable-recovery`，同时保持其他 R3 recovery 与独立授权门禁不变。

## Impact

- 计划修改：`.agents/openspec-workflow.md`、`backend/internal/architecture/task_design_lint.go`、`backend/internal/architecture/task_design_lint_test.go` 及必要的 `backend/internal/architecture/testdata/task_design/` fixture。
- 本 change artifacts 位于 `openspec/changes/allow-local-neo4j-disposable-r3-recovery/`；不修改 `rebuild-foundation-graph-and-enrich-chain-data` artifacts。
- 不修改业务源码、数据库、Neo4j、CI、依赖、`doc/` 或 `prototype/`，不执行 cleanup、rebuild、sync 或其他有状态操作。
- 基线风险为 R1；规则影响共享 workflow，因此 Apply final 计划运行 backend repo-wide `go test ./...`、OpenSpec strict、当前 change explicit task-design lint 与 diff/scope/secret 检查。
