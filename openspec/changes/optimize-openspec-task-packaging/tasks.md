## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | Proposal Review | R1 | yes | SPEC_SEMANTICS | 仅允许进入 package 2；不授权 R2/R3、数据库、Neo4j、部署、Sync、Archive 或 Deliver |
| 2 | R0/R1 Apply package | R1 | no | NONE | 完成测试先行、规则/lint、targeted 验证、修复和一个 scoped checkpoint |
| 3 | Apply-final Review | R1 | yes | APPLY_FINAL | 审阅通过后才允许 Sync、Archive、Deliver；不授权任何有状态操作 |

## Complexity Budget

| Key | Value |
|---|---|
| human_gates | 2 |
| stateful_layers | 0 |
| checkpoints | 2 |
| full_test_runs | 1 |
| continuous_automation_scope | packages:2 |

## 1. Proposal Review Package

- [x] 1.1 完成中文 proposal、design、delta spec 与 tasks，运行 `openspec validate optimize-openspec-task-packaging --strict`、scoped diff/secret/所有权检查并创建、推送 `spec: propose optimize-openspec-task-packaging` checkpoint；主对话人工确认 Gate Map、复杂度预算、合法 gate、R2 baseline 复用、lint fail/warning 边界、active change 不 adoption 与全部 non-goals 后，才允许进入第 2 个 package。

## 2. R0/R1 Apply Package

- [x] 2.1 测试先行：在 `backend/internal/architecture/testdata/task_design/` 增加合规 zero-stateful、合规 multi-layer stateful 与违规 fixture；先用失败测试覆盖 Gate Map 固定列/枚举/package 映射、Complexity Budget 固定键/整数/selector、Stateful Layer Map 缺失/字段/order/package 映射、baseline valid-skip/stale/duplicate/unknown/archived warning、explicit mode 绕过 baseline，以及微型 Review/checkpoint/commit/push 只 warning 的边界。
- [x] 2.2 在 `.agents/openspec-workflow.md` 落实一级 package、Gate Map、复杂度预算、合法人工 gate、package 验证、候选全量机器校验与例外人工审阅、local-only R2 多层执行和 recovery baseline 复验；仅在必要时对根 `AGENTS.md` 保留短摘要，不复制详细流程。
- [x] 2.3 复用 `backend/internal/architecture/` 的 Go 标准库测试模式实现 task-design lint：active mode 在 `OPENSPEC_TASK_LINT_CHANGE` 未设置时扫描 active changes 并读取 `.agents/openspec-task-lint-baseline.tsv`；explicit mode 设置该变量后只校验指定 change 且不跳过 baseline。两种模式都排除 archive；通过现有 backend `go test ./...` 自动接入 CI，不新增依赖、wrapper 或平行 job。
- [x] 2.4 完成 targeted `go test ./internal/architecture -run '^TestOpenSpecTaskDesignLint' -count=1`，并从 `backend/` 运行 `OPENSPEC_TASK_LINT_CHANGE=optimize-openspec-task-packaging go test ./internal/architecture -run '^TestOpenSpecTaskDesignLint$' -count=1` 作为当前 change 的精确 Proposal/package checkpoint 命令；同时运行 OpenSpec strict validation、规则链接/重复检查、`git diff --check`、scope/secret 检查和 self-review，失败与修复留在本 package 内。
- [x] 2.5 汇总一个 R1 package evidence，记录受影响边界、fixture、fail/warning 结果、legacy active baseline、未验证项和阻断项；创建并推送一个 scoped Apply package checkpoint，不为 2.1—2.4 分别制造 Review、commit 或 push。

## 3. Apply-final Review Package

- [ ] 3.1 运行一次受影响边界完整验证：backend `go test ./...`、共享 architecture/contract tests、`openspec validate optimize-openspec-task-packaging --strict`、规则链接/重复检查、`git diff --check`、scope/secret 检查；复读新鲜结果并确认 stateful layers 为 0、无 R2/R3 状态变化。
- [ ] 3.2 提交 scoped diff、完整验证、self-review 和第 2 个 package checkpoint 证据，等待主对话 Apply-final 人工 Review；通过前不得 Sync、Archive、Deliver、创建完成态 PR，或修改任何 active change artifacts/tasks。
