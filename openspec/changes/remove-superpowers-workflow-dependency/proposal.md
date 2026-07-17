## Why

当前正式研发规则把 Superpowers plugin 的存在误当成 TDD、系统化调试、完成前验证、Git 收尾和部分 OpenSpec 路由的执行前提。用户将卸载该 plugin，因此需要把这些约束归还到项目原生规则、OpenSpec CLI、Codex Desktop worktree 机制和 GitHub/`gh` 交付路径中，确保卸载后仍可完整执行生命周期。

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

## What Changes

- 从 `.agents/skill-routing.md` 移除全部 `superpowers:*` 强制路由；OpenSpec Skills 保留 Explore/Propose/Apply/Sync/Archive 路由，Review 由用户人工确认，TDD/debug/verification 由项目原生规则执行，Deliver 使用 GitHub plugin 或 `gh` fallback。
- 从 `.agents/testing-tdd.md` 移除 Superpowers 调用依赖，保留 RED→GREEN→REFACTOR、测试先行、失败/通过证据、边界验证、fake/fixture 和 R2/R3 规则。
- 从 `.agents/git-workflow.md` 移除 `finishing-a-development-branch` 与 `using-git-worktrees` 依赖，保留 Desktop-managed worktree 硬门、经明确批准的 project-owned fallback，以及 branch/PR/merge/cleanup 顺序。
- 更新 `skill-driven-development-workflow` delta spec，使 Superpowers 变为可选辅助而不是项目执行依赖，并禁止 `docs/superpowers` 作为事实来源。
- 更新 workflow architecture targeted assertions：负向断言正式规则/主 spec 不含 Superpowers 强制引用，正向断言所有 OpenSpec、TDD、debug diagnosis、fresh verification、Desktop worktree、branch/cleanup 和 Review gates 仍存在。
- **BREAKING**：卸载 Superpowers 后，不再提供其全局“skills before response”行为；本 change 不模拟该 plugin 的全局行为，也不创建替代 Skill 或执行器。

## Capabilities

### New Capabilities

- 无。

### Modified Capabilities

- `skill-driven-development-workflow`：将 Superpowers 从必要执行机制降级为可选辅助，并要求项目原生规则与 architecture assertions 独立保证 workflow 硬门。

## Impact

- 影响范围仅为 `.agents/skill-routing.md`、`.agents/testing-tdd.md`、`.agents/git-workflow.md`、`openspec/specs/skill-driven-development-workflow/spec.md`、本 change artifacts，以及 `backend/internal/architecture/workflow_rules_test.go` 及其必要的 workflow assertion 测试数据。
- 不修改业务代码、API、数据库、图谱、部署、UAT/prod/shared、`prototype/` 或 `doc/`；不创建平行 workflow、替代执行器或 `docs/superpowers` 文件。
- 卸载顺序：Apply-final Review 通过 → Sync/Archive/Deliver/PR merge+cleanup → 独立核验 `main` 无 Superpowers 强制引用 → 用户再卸载 plugin。
