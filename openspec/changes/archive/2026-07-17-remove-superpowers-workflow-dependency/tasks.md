## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | Proposal Review 通过后才允许 Apply 规则与断言变更 | R1 | yes | SPEC_SEMANTICS | 仅限本 change 的 OpenSpec artifacts、`.agents` 路由/测试契约设计；不得修改生产代码、数据库、部署或其他 active change |
| 2 | Apply-final Review 前完成 scoped workflow 变更与新鲜验证 | R1 | yes | APPLY_FINAL | 仅限本 change 已批准的 `.agents`、主 workflow spec 与 workflow architecture assertions；不得 Sync、Archive、Deliver、PR 或 merge |

## Complexity Budget

| Key | Value |
|---|---|
| human_gates | 2 |
| stateful_layers | 0 |
| checkpoints | 2 |
| full_test_runs | 0 |
| continuous_automation_scope | packages:none |

## 1. 解耦正式路由与规则 Package

- [x] 1.1 Proposal Review 通过后，按 before/after 路由矩阵更新 `.agents/skill-routing.md`，移除全部 `superpowers:*` 强制路由并保留 OpenSpec、人工 Review、项目原生规则和 GitHub/`gh` 路径。
- [x] 1.2 更新 `.agents/testing-tdd.md` 与 `.agents/git-workflow.md`，将 TDD/debug/verification、Desktop-managed worktree、批准 fallback、branch/PR/merge/cleanup 直接写成项目原生约束。
- [x] 1.3 更新 `openspec/specs/skill-driven-development-workflow/spec.md` 对应 delta 落地所需的长期可验证行为，明确 Superpowers 可选且不构成执行依赖。

## 2. Workflow assertions 与 Proposal Review Package

- [x] 2.1 更新 `backend/internal/architecture/workflow_rules_test.go` 及必要测试数据：负向拒绝 `superpowers:*`、`docs/superpowers`、必须安装 Superpowers 的正式引用，正向覆盖 OpenSpec 生命周期、TDD/debug/fresh verification、Desktop worktree、branch/cleanup、Review gates、风险/事实源/安全边界。
- [x] 2.2 运行精确 `OPENSPEC_TASK_LINT_CHANGE=remove-superpowers-workflow-dependency go test ./internal/architecture -run '^TestOpenSpecTaskDesignLint$' -count=1`、`openspec validate remove-superpowers-workflow-dependency --strict`、`git diff --check` 及 scoped scope/secret/link/reference checks；不运行业务 `go test ./...`。
- [x] 2.3 形成 Apply-final Review package：记录 scoped diff、验证输出、未验证项和阻断项；Apply-final Review 前不得 Sync、Archive、Deliver、PR 或 merge。
