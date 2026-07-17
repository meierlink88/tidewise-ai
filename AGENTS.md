# 项目 Agent 工作规范

## 强制入口

任何会写入本仓库的任务，在提出方案或修改文件前都必须先调用
`$eino-reference-first`。该要求适用于源代码、测试、配置、文档、提示词、
skills、脚本、依赖、生成物和工程结构。即使最终判断不需要使用 Eino，也必须
完成与任务相关的 `eino-ext`、`eino-examples`、`eino` 三仓审计。

只读检查可以先执行，以确定审计范围。`.reference/cloudwego/` 仅作为只读上游
学习资料，禁止编辑、暂存、提交或发布其中的任何文件。

## 工程流程

除下述“工作流规则例外”外，所有仓库改动都必须通过命名的 OpenSpec change 执行，
完整生命周期为：

`Explore -> Delegate -> Propose -> Leader Review -> Apply -> Validate -> Leader Acceptance -> Sync -> Archive -> Deliver -> Merge -> Cleanup`

开发 Leader 位于主 Codex 任务，只负责 Explore、使用 `create_thread` 创建和委派
独立执行任务、评审、批准、监控、Leader Acceptance 及合并后清理；不得实施具体
代码、测试、配置、文档或 change 产物。执行 Agent 位于独立 Desktop-managed
worktree，负责 Eino audit、OpenSpec、TDD、实现、验证、Sync、Archive、提交、推送
和 Pull Request。

本规范中的“独立执行任务”和“执行 Agent”专指通过 Codex `create_thread` 创建、在
侧边栏可见并拥有 Desktop-managed worktree 的独立 Codex 任务，不包括 `multi_agent`
或任何内部 sub-agent。用户提出需要新 OpenSpec change 的仓库修改请求时，该 change 请求即视为授权
Leader 创建对应独立 Codex 任务，无需再次询问是否创建任务；Leader 不得使用内部
sub-agent 承载或替代 Propose、Apply、Validate、Sync、Archive、Deliver。
委派只有在 `create_thread` 返回 `threadId` 或 `clientThreadId`、`hostId`，且请求明确
指定 worktree 环境、`model: gpt-5.6-sol`、`thinking: medium` 后才算完成。任一条件不可用
时 Leader 必须停止并报告，不得静默改用其他任务类型、模型或执行环境。

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

## 工作流规则例外

仅修改工程协作工作流规则、对应正式工作流规格、策略测试，以及直接支撑该流程的
`.agents/skills/` 项目级工程协作 skill 或脚本时，由开发 Leader 在当前主 Codex 对话
直接完成，无需委派独立执行 Agent、无需创建 OpenSpec change，也不经过 Leader Review
或 Leader Acceptance。该例外不得用于业务代码、运行时配置、普通产品文档、提示词、
运行时 Agent skill、依赖或工程结构变更；范围一旦超出工作流规则及其策略保护文件，
必须恢复上述完整委派式 OpenSpec 生命周期。

Leader 直接修改工作流规则时仍须先完成 `$eino-reference-first`，从最新 `origin/main`
创建独立 `codex/<workflow-rule-name>` 分支，按 `RED -> GREEN -> REFACTOR` 更新策略测试，
执行聚焦与全量验证、密钥和禁止路径检查，再由 Leader 提交、推送并直接创建 Pull
Request；Leader 直接创建 Pull Request 不代表合并，用户控制 PR merge。

Leader 委派普通 change 任务时，默认使用产品口径“gpt 6 sol medium”对应的当前可执行
参数 `model: gpt-5.6-sol` 与 `thinking: medium`。如果该组合在目标 host 不可用，Leader
必须停止委派并报告，不得静默降级或改用其他模型。

## 通用约束

- 新 change 自动使用 `codex/<change-name>` 分支和独立 worktree；不得直接在
  `main` 上实施。
- 行为变更必须按 `RED -> GREEN -> REFACTOR` 执行并保留可复核证据。
- OpenSpec 固定结构标记、规范关键字及不可翻译技术标识保留原文，其余 change
  产物中文优先。
- `.env`、密钥、运行产物、操作系统文件和 `.reference/cloudwego/` 不得进入
  commit 或 Pull Request。
- 完成 change 前必须提交、推送 change 分支并创建 Pull Request；合并由用户执行。
