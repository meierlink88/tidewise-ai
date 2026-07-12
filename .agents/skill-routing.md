# Skill Routing

正式研发必须优先调用与当前任务匹配的已安装 Skill。OpenSpec 是唯一正式 change 生命周期和 artifacts 的所有者；Superpowers 提供执行纪律；GitHub plugin 负责远端协作操作。

## Priority

规则冲突时按以下顺序处理：

```text
用户当前明确指令
> AGENTS.md 与 .agents 项目规则
> 已批准 OpenSpec artifacts
> 已安装 Skills
> Agent 临时判断
```

Skill 不得覆盖项目的 artifact 路径、change 顺序、分支命名、安全边界和技术约束。

## OpenSpec Lifecycle

| 阶段 | 必须使用的主 Skill 或机制 | 执行辅助 |
|---|---|---|
| Explore | `openspec-explore` | `superpowers:brainstorming` |
| Propose | `openspec-propose` | `superpowers:writing-plans` 的任务拆分方法 |
| Review | 用户人工确认 OpenSpec artifacts | 必要时 `superpowers:requesting-code-review` |
| Apply | `openspec-apply-change` | TDD、debug、计划执行或 Subagent Skills |
| Validate | OpenSpec CLI 和项目验证命令 | `superpowers:verification-before-completion` |
| Sync | `openspec-sync-specs` | 完成前验证 |
| Archive | `openspec-archive-change` | 完成前验证 |
| Branch finish | `superpowers:finishing-a-development-branch` | GitHub plugin |

不得跳过人工 Review 直接从 Propose 进入 Apply。只有 tasks 全部完成、主规格已同步、change 已归档且 `openspec validate --all` 通过后，才能进入 branch finish、PR 或 merge。

## Artifact Ownership

- `superpowers:brainstorming` 的确认结论写入当前 change 的 `design.md`。
- `superpowers:writing-plans` 的可执行计划写入当前 change 的 `tasks.md`。
- 默认不得创建 `docs/superpowers/specs/` 或 `docs/superpowers/plans/`。
- 用户明确要求额外计划文档时可以创建，但它不得成为正式状态来源，也不得替代 OpenSpec artifacts。

## Engineering Skills

- 功能、bugfix、重构或行为变更：使用 `superpowers:test-driven-development`。
- 缺陷、测试失败或异常行为：先使用 `superpowers:systematic-debugging`，基于证据定位原因。
- 声明完成、commit、push、PR、sync 或 archive 前：使用 `superpowers:verification-before-completion`，读取新鲜验证结果。
- 主要功能完成或准备合并：使用 `superpowers:requesting-code-review`；收到意见后使用 `superpowers:receiving-code-review` 验证再修改。
- 已有书面实施计划且需要跨检查点执行：使用 `superpowers:executing-plans`。
- 两个以上无共享写状态、无顺序依赖的任务：可使用 `superpowers:dispatching-parallel-agents`。
- 任务边界、文件所有权和合并方式明确时：可使用 `superpowers:subagent-driven-development`。
- 多 Agent 不得同时修改同一 OpenSpec artifact、tasks 文件、数据库状态或同一源码文件。

## Git And GitHub Skills

- 新 change 的最新 `origin/main` 基线、branch、worktree 和 commit 规则见 `.agents/git-workflow.md`。
- 并行 change 或长任务隔离使用 `superpowers:using-git-worktrees`；在 Codex Desktop 中优先使用与独立任务绑定的原生 worktree。
- branch 收尾使用 `superpowers:finishing-a-development-branch`，但必须位于 `openspec-archive-change` 之后。
- commit、push 和创建 PR 优先使用 `github:yeet`。
- CI 失败使用 `github:gh-fix-ci`。
- PR review comments 使用 `github:gh-address-comments`，并结合 `superpowers:receiving-code-review`。

## Lightweight Tasks

纯解释、只读查询或单条命令不需要机械创建 OpenSpec change，也不要求调用所有 Skills。只在 Skill 的触发条件与任务匹配时调用。
