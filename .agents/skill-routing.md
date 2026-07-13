# Skill Routing

正式研发必须优先调用与任务匹配的已安装 Skill。OpenSpec 拥有唯一正式 change 生命周期和 artifacts；Superpowers 提供工程执行纪律；GitHub plugin 负责远端协作。

## Priority

规则冲突时按以下顺序处理：

```text
用户当前明确指令
> AGENTS.md 与 .agents 项目规则
> 已批准 OpenSpec artifacts
> 已安装 Skills
> Agent 临时判断
```

Skill 不得覆盖项目 artifact 路径、生命周期顺序、分支命名、安全边界和技术约束。生命周期门禁见 `.agents/openspec-workflow.md`，Git 与 Desktop worktree 规则见 `.agents/git-workflow.md`。

## Lifecycle Skill Map

| 阶段 | 主 Skill 或机制 | 执行辅助 |
|---|---|---|
| Explore | `openspec-explore` | `superpowers:brainstorming` |
| Propose | `openspec-propose` | `superpowers:writing-plans` 的任务拆分方法 |
| Review | 用户人工确认 OpenSpec artifacts | 必要时 `superpowers:requesting-code-review` |
| Apply | `openspec-apply-change` | TDD、debug、计划执行或 Subagent Skills |
| Validate | OpenSpec CLI 和项目验证命令 | `superpowers:verification-before-completion` |
| Sync | `openspec-sync-specs` | 完成前验证 |
| Archive | `openspec-archive-change` | 完成前验证 |
| Deliver | `superpowers:finishing-a-development-branch` | GitHub plugin 与 scoped Git 交付 |

完整阶段语义、顺序和完成条件只在 `.agents/openspec-workflow.md` 维护。

## Artifact Ownership

- Brainstorming 的确认结论进入当前 change 的 `design.md`。
- Writing Plans 的可执行计划进入当前 change 的 `tasks.md`。
- 默认不得创建 `docs/superpowers/specs/` 或 `docs/superpowers/plans/`；用户明确要求的额外文档也不得替代 OpenSpec artifacts。

## Engineering Skills

- 功能、bugfix、重构或行为变更：`superpowers:test-driven-development`。
- 缺陷、测试失败或异常：先使用 `superpowers:systematic-debugging`。
- 声明完成、commit、push、PR、sync 或 archive 前：`superpowers:verification-before-completion`。
- 主要功能完成或准备合并：`superpowers:requesting-code-review`；处理反馈：`superpowers:receiving-code-review`。
- 同一风险边界的 contract、实现、测试、dry-run 和只读 preflight 可以组成阶段 Review package；task agent 在通知主对话前必须完成 self-review/code review，并先整改阻断问题。风险等级、授权和执行包正文只在 `.agents/openspec-workflow.md` 维护。
- 已有书面计划且跨检查点执行：`superpowers:executing-plans`。
- 两个以上无共享写状态且无顺序依赖的任务：`superpowers:dispatching-parallel-agents`。
- 边界和所有权明确的独立任务：`superpowers:subagent-driven-development`。
- 多 Agent 不得同时修改同一 OpenSpec artifact、tasks、数据库状态或源码文件。

## Worktree Skill Routing

- Codex Desktop 可用时，由 Desktop 新任务机制创建受管 worktree；不得使用 Skill 将 project-owned worktree 作为默认路径。
- 只有 Codex Desktop 受管机制不可用且用户明确批准 fallback 时，才使用 `superpowers:using-git-worktrees` 创建 project-owned worktree；完整条件和操作服从 `.agents/git-workflow.md`。

## Git And GitHub Skills

- 新 change、branch、Desktop worktree、commit 和 cleanup 服从 `.agents/git-workflow.md`。
- branch 收尾使用 `superpowers:finishing-a-development-branch`，但只能在 Archive 之后。
- commit、push 和 PR 优先使用 `github:yeet`；CI 失败使用 `github:gh-fix-ci`；PR 评论使用 `github:gh-address-comments`。

## Lightweight Tasks

纯解释、只读查询或单条命令不需要机械创建 OpenSpec change；仅在 Skill 触发条件匹配时调用。
