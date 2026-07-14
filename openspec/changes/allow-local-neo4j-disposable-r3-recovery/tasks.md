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

## 1. Proposal 规则语义 Review Package

- [x] 1.1 人工 Review proposal、design、delta spec 与 tasks，确认精确合取谓词、允许/禁止矩阵、文件范围、受阻 change adoption 方式及“lint 通过不等于 R3 授权”；批准前不得进入 Package 2。

## 2. Workflow、Lint 与 Tests 连续实现 Package

- [x] 2.1 先在 `backend/internal/architecture/task_design_lint_test.go` 与必要 fixture 增加 table-driven RED cases：保留 local R2；允许 Scope 显式锚定的 local Neo4j R3 projection cleanup/rebuild/sync；拒绝 shared-local、UAT、prod、Layer 合法但 Scope 非 Neo4j/非允许 operation、缺 PG/PostgreSQL baseline 与 Human=no，并以受阻 change adoption-ready Scope 锁定回归。
- [x] 2.2 运行 targeted test 读取 RED 结果后，最小修改 `backend/internal/architecture/task_design_lint.go` 的 recovery 判定，并同步更新 `.agents/openspec-workflow.md`；不新增 schema 列、依赖、通用 framework、CI 或业务源码。
- [x] 2.3 运行 targeted GREEN、architecture suite、backend `go test ./...`、OpenSpec strict、当前 change explicit task-design lint、diff/scope/secret 检查并完成 self-review；任何失败留在本 package 修复并刷新证据，不新增人工 gate。

## 3. Apply-final、Sync、Archive、Deliver Package

- [x] 3.1 汇总 scoped diff、允许/禁止矩阵、TDD RED/GREEN、repo-wide backend 验证、未验证项与 adoption 证据，提交 Apply-final Review checkpoint 后停止等待人工 Review；不得把该 Review 解释为任何 Neo4j R3 layer 授权。
- [ ] 3.2 仅在 Apply-final Review 通过后依序 Sync、Archive、运行 `openspec validate --all` 并完成 archive checkpoint；再按 Git 门禁 Deliver。规则进入最新 `origin/main` 前，受阻 change 不得 adoption 或继续 Apply。
