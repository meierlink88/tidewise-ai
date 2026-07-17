# Skill Routing

正式研发优先调用与任务匹配的已安装 Skill。OpenSpec 拥有唯一正式 artifacts；本文件只维护 Skill 路由，不重复生命周期、Git 或测试详述。可选 Skill 不得成为项目执行依赖。

## Workflow-Only Routing Exception

仅修改 `AGENTS.md`、`.agents/**`、项目 workflow/Skill 路由规则，以及这些规则专用的 lint 或 architecture tests 时，由当前 Leader 主对话直接实施，不创建独立 Desktop 任务，也不调用 OpenSpec propose/apply/sync/archive Skills。Leader 只运行命中规则的 targeted validation、规则或 architecture tests、链接检查、`git diff --check`、scope 与 secret 检查，然后按 `.agents/git-workflow.md` 直接提交 PR。

若改动同时触及产品或业务规格、生产代码、API、数据库 migration/seed、业务数据、部署或运行环境，则不得拆分或伪装为 workflow-only change，必须恢复标准 Skill 路由和 OpenSpec 生命周期。

## Priority

规则冲突时按以下顺序处理：

```text
用户当前明确指令
> AGENTS.md 与 .agents
> 已批准 OpenSpec artifacts
> 已安装 Skills
> Agent 临时判断
```

Skill 不得覆盖项目 artifact 路径、生命周期顺序、分支命名、安全边界和技术约束；操作规则分别见 `.agents/openspec-workflow.md`、`.agents/git-workflow.md` 和 `.agents/testing-tdd.md`。

## Lifecycle Skill Map

| 阶段 | 主 Skill 或机制 | 执行辅助 |
|---|---|---|
| Explore | `openspec-explore` | 可选 brainstorming 辅助 |
| Propose | `openspec-propose` | 可选任务拆分辅助 |
| Review | 用户人工确认 OpenSpec artifacts | 项目原生 self-review；必要时 GitHub review |
| Apply | `openspec-apply-change` | 项目原生 TDD、debug、计划执行或 Subagent Skills |
| Validate | OpenSpec CLI 和项目验证命令 | 项目原生 fresh verification |
| Sync | `openspec-sync-specs` | 项目原生完成前验证 |
| Archive | `openspec-archive-change` | 项目原生完成前验证 |
| Deliver | GitHub plugin 优先，`gh` fallback | `.agents/git-workflow.md` 交付规则 |

## Artifact Ownership

OpenSpec `proposal.md`、`design.md`、delta specs、`tasks.md` 和归档历史是唯一正式 artifacts。不得创建平行设计、计划或执行事实来源。

## Engineering Skills

- 功能、bugfix、重构或行为变更：遵循 `.agents/testing-tdd.md` 的测试先行与 TDD gate；异常或测试失败：先完成项目原生 diagnosis。
- 声明完成、commit、push、PR、sync 或 archive 前：遵循 `.agents/testing-tdd.md` 的 fresh verification。
- 主要功能完成或准备合并：完成项目原生 self-review；GitHub review feedback 按对应 GitHub workflow 处理。
- 同一风险边界的 contract、实现、测试、dry-run 和只读 preflight 可组成一个阶段 Review package；task agent 在通知主对话前完成 self-review/code review。
- 只有两个以上无共享写状态、无顺序依赖的任务才使用并行 Agent；不得并行修改同一 artifact、tasks、源码文件或数据库状态。

## Worktree Skill Routing

Codex Desktop 可用时，由 Desktop 新任务机制创建受管 worktree；不得以任意 Skill 替代该入口。只有 Codex Desktop 受管机制不可用且用户明确批准 fallback 时，才按 `.agents/git-workflow.md` 的项目原生规则创建 project-owned worktree。

## Git And GitHub Skills

新 change、branch、Desktop worktree、commit 和 cleanup 服从 `.agents/git-workflow.md`。commit、push 和 PR 优先使用 `github:yeet`；CI 失败使用 `github:gh-fix-ci`；PR 评论使用 `github:gh-address-comments`。

## Lightweight Tasks

纯解释、只读查询或单条命令不需要机械创建 OpenSpec change；仅在 Skill 触发条件匹配时调用。
