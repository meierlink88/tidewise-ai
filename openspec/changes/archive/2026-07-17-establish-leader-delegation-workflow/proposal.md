## Why

现有工程变更流程要求隔离 worktree、两次人工评审和 Pull Request 交付，但没有明确区分开发 Leader 与执行 Agent，也没有规定必须通过 Codex 新任务委派实施或在合并后安全清理 change 资源。随着多个 change 并行推进，需要把职责分离、任务隔离和可审计清理固化为可测试的仓库治理规则。

## What Changes

- 将生命周期扩展为 `Explore -> Delegate -> Propose -> Leader Review -> Apply -> Validate -> Leader Acceptance -> Sync -> Archive -> Deliver -> Merge -> Cleanup`。
- 规定开发 Leader（主 Codex 任务）仅负责探索、创建和委派新 Codex 任务、评审、批准、监控、验收，以及合并后的清理协调；不得承担具体代码或文档实施。
- 规定每个新 change 必须由 Leader 使用 Codex `create_thread` 创建独立执行任务和 Desktop-managed worktree，并继续使用 `codex/<change-name>` feature branch。
- 规定执行 Agent 独立完成 Eino audit、OpenSpec、TDD、实现、验证、Sync、Archive、提交、推送和 PR，但不得自行批准 Proposal Review 或 Apply-final Review；用户继续控制 PR merge。
- 规定确认 PR merged 后，从主项目依次核验 change worktree clean、change 提交已进入 `origin/main`，再删除 worktree、本地分支、远端分支（若存在），最后执行 prune；任何清理失败必须明确报告。
- 以 TDD 扩展现有 workflow policy Go 测试，保护 Leader/执行 Agent 职责、Codex 任务委派、两道 Leader 门禁和 post-merge cleanup 规则。
- 不修改 AI 采集器、AI 事件提取器或 AI 投研报告分析师的运行行为，不引入 Eino 运行时组件、外部服务或自动合并能力，也不处理已有 `adopt-openspec-tdd-workflow` worktree。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `engineering-change-workflow`：增加 Leader/执行 Agent 职责分离、Codex Desktop 任务委派、Leader 审批门禁，以及 PR 合并后的安全资源清理要求。

## Impact

- 影响 `AGENTS.md`、`.agents/` 下的工作流与 Git 规则文档、`openspec/config.yaml`、`openspec/specs/engineering-change-workflow/spec.md` 及 `internal/architecture/workflow_policy_test.go`。
- 后续所有正式 change 都必须由主任务 Leader 委派给独立执行任务；现有 `codex/<change-name>` 分支命名和用户控制合并保持兼容。
- 主要风险是过度依赖稳定文案的策略测试、误清理未合并或不干净的 worktree，以及 Codex Desktop 状态与 Git 状态不一致；通过测试稳定标记、严格清理前置检查和失败显式报告降低风险。
