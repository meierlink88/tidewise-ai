# 项目 Agent 工作规范

## 强制入口

任何会写入本仓库的任务，在提出方案或修改文件前都必须先调用
`$eino-reference-first`。该要求适用于源代码、测试、配置、文档、提示词、
skills、脚本、依赖、生成物和工程结构。即使最终判断不需要使用 Eino，也必须
完成与任务相关的 `eino-ext`、`eino-examples`、`eino` 三仓审计。

只读检查可以先执行，以确定审计范围。`.reference/cloudwego/` 仅作为只读上游
学习资料，禁止编辑、暂存、提交或发布其中的任何文件。

## 工程流程

所有仓库改动都必须通过命名的 OpenSpec change 执行，完整生命周期为：

`Explore -> Delegate -> Propose -> Leader Review -> Apply -> Validate -> Leader Acceptance -> Sync -> Archive -> Deliver -> Merge -> Cleanup`

开发 Leader 位于主 Codex 任务，只负责 Explore、使用 `create_thread` 创建和委派
独立执行任务、评审、批准、监控、Leader Acceptance 及合并后清理；不得实施具体
代码、测试、配置、文档或 change 产物。执行 Agent 位于独立 Desktop-managed
worktree，负责 Eino audit、OpenSpec、TDD、实现、验证、Sync、Archive、提交、推送
和 Pull Request。

开始工作前必须根据任务读取下列规则：

| 范围 | 必读文件 |
|---|---|
| 技能选择 | `.agents/skill-routing.md` |
| OpenSpec 生命周期与评审门禁 | `.agents/openspec-workflow.md` |
| 分支、worktree、提交与 Pull Request | `.agents/git-workflow.md` |
| TDD 与验证证据 | `.agents/testing-tdd.md` |
| 三个任务 Agent 的工程边界 | `.agents/architecture-boundaries.md` |

提案材料必须获得开发 Leader 明确完成 Leader Review 后才能进入 Apply；实现和验证
完成后，必须再次获得开发 Leader 明确完成 Leader Acceptance，才能执行 Sync、Archive
和最终 Pull Request 交付。执行 Agent不得自行批准这两个门禁。用户控制 PR merge；
PR merged 后由 Leader 在主任务中执行并报告 Cleanup，不回写已归档 change 的 tasks。

## 通用约束

- 新 change 自动使用 `codex/<change-name>` 分支和独立 worktree；不得直接在
  `main` 上实施。
- 行为变更必须按 `RED -> GREEN -> REFACTOR` 执行并保留可复核证据。
- OpenSpec 固定结构标记、规范关键字及不可翻译技术标识保留原文，其余 change
  产物中文优先。
- `.env`、密钥、运行产物、操作系统文件和 `.reference/cloudwego/` 不得进入
  commit 或 Pull Request。
- 完成 change 前必须提交、推送 change 分支并创建 Pull Request；合并由用户执行。
