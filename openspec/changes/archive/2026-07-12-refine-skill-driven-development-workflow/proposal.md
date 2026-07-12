## Why

项目已经同时安装 OpenSpec 1.5.0、Superpowers 6.1.1 和 GitHub plugin，但现有规则仍重复描述部分 Skill 已提供的流程，且没有完整定义 Skill 路由、artifact 归属和冲突覆盖，容易导致后续 Agent 生成平行设计文档、跳过 Skill 或过早进入 PR/merge。`openspec/config.yaml` 还包含 OpenSpec 1.5.0 不支持的 `rules.language` 并保留部分过期架构上下文，需要同步治理。

## What Changes

- 新增统一的 Skill 路由规则，明确 OpenSpec、Superpowers、GitHub plugin、项目规则和用户指令的职责与优先级。
- 将 `Explore -> Propose -> Review -> Apply -> Validate -> Sync -> Archive` 各阶段显式映射到已安装的 OpenSpec 和 Superpowers skills。
- 明确 `brainstorming` 和 `writing-plans` 的内容必须落入 OpenSpec `design.md` 与 `tasks.md`，默认禁止创建平行的 `docs/superpowers/specs/` 和 `docs/superpowers/plans/` artifacts。
- 将 TDD、系统化调试、完成前验证、代码审查、并行 Agent、worktree 和分支收尾路由到对应 Superpowers skills，并保留项目特有约束。
- 将 GitHub push、PR、CI 修复和 review comments 路由到对应 GitHub plugin skills。
- 明确每个新 change 必须先更新远端引用，并基于最新 `origin/main` 创建独立 branch；并行 change 优先使用 Codex Desktop 原生 worktree 任务。
- 精简 `AGENTS.md`、OpenSpec、Git 和 TDD 规则中的重复流程语言，使规则文件主要承担 Skill 路由和项目差异约束。
- 修复 `openspec/config.yaml` 中不受支持的 `rules.language`，并更新明显过期的工程上下文描述。

## Capabilities

### New Capabilities

- `skill-driven-development-workflow`: 定义正式研发 change 如何组合 OpenSpec、Superpowers、GitHub plugin、Git branch/worktree 和项目规则完成全生命周期交付。

### Modified Capabilities

无。

## Impact

- 影响 `AGENTS.md`、`.agents/openspec-workflow.md`、`.agents/git-workflow.md`、`.agents/testing-tdd.md` 和新增的 `.agents/skill-routing.md`。
- 影响 `openspec/config.yaml` 的规则结构和工程上下文，不改变 OpenSpec schema。
- 不修改 `frontend/`、后端业务运行时代码、`prototype/`、业务 API、数据库或运行时行为；仅在 `backend/internal/architecture` 增加规则静态测试。
- 后续 Agent 将优先调用匹配的已安装 Skill，项目规则只保留路由、优先级、artifact 所有权和项目特有约束。
