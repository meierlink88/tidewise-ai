# Git 分支、worktree 与 Pull Request

## Leader 直接交付工作流规则修改

当且仅当变更范围限于工程协作规则、正式工作流规格、策略测试，以及直接支撑该流程的
`.agents/skills/` 项目级工程协作 skill 或脚本时，Leader 无需创建独立执行任务或
OpenSpec change，直接在当前主任务中从最新 `origin/main` 创建
`codex/<workflow-rule-name>` 分支。完成策略测试 RED/GREEN/REFACTOR、全量验证、差异和
凭据检查后，由 Leader 提交、推送并创建 base 为 `main` 的 Pull Request。用户仍控制
Pull Request merge。若修改范围扩展到业务代码、运行时配置、普通产品文档、提示词、
运行时 Agent skill、依赖或工程结构，则停止直接流程，改用下述新 change 自动初始化流程。

## 新 change 自动初始化

开发 Leader接到新的仓库改动任务后，先检查 active changes 和现有 worktrees。若不存在
匹配项，Leader使用 Codex `create_thread` 创建独立执行任务和 Desktop-managed worktree；
执行 Agent在该 worktree 中将创建隔离工作空间视为正常实施步骤，无需额外询问：

用户提出需要新 OpenSpec change 的修改请求即授权创建该任务。“独立执行任务”不包括
`multi_agent` 或内部 sub-agent。Leader 调用 `create_thread` 时必须指定 worktree 环境、
`model: gpt-5.6-sol` 和 `thinking: medium`；只有取得 `threadId` 或 `clientThreadId` 及
`hostId` 后才算委派成功。任一条件不可用时停止并报告，不得改用其他任务类型。

1. 运行 `git fetch origin main` 获取最新基线。
2. 使用分支 `codex/<change-name>`，基线必须是最新 `origin/main`。
3. 优先使用 Codex Desktop 管理的 worktree。
4. Desktop-managed worktree 不可用时，使用：
   `git worktree add ../tidewise-ai-agentrun-worktrees/<change-name> -b codex/<change-name> origin/main`。
5. 运行 `git branch --show-current`、`git worktree list` 和 `git status --short` 验证环境。

如果同名 branch 或 worktree 已存在，Leader和执行 Agent先确认其对应同一个 active
change，然后恢复该工作空间；不得创建重复 change，也不得覆盖用户已有修改。

## 分支约束

- `main` 只接受 Pull Request 合并，不直接实施 change。
- 每个 change 对应一个 `codex/<change-name>` 分支和一个 worktree。
- 不在一个分支中混入多个无关 change。
- 不使用 `git reset --hard`、强制推送或其他破坏性命令处理用户修改。
- 需要同步主线时，先确认工作区干净，再采用可审计的 rebase 或 merge。

## 提交约束

- 仅暂存当前 change 涉及的文件，提交前查看 `git status --short` 和 staged diff。
- 执行 `git diff --cached --check`。
- 扫描凭据格式并确认 `.env`、`data/`、`.reference/cloudwego/` 等被忽略。
- 使用 conventional commit 风格，描述实际结果，不使用含糊的“update files”。
- 未通过测试和评审门禁时，不创建表示已完成的提交或 PR。

## Pull Request 交付

获得 Leader Acceptance、完成 Sync、Archive 和最终验证后，执行 Agent：

1. 提交当前 change 的全部必要文件。
2. 推送 `codex/<change-name>`。
3. 创建 GitHub Pull Request，base 为 `main`。
4. PR 正文列出 change 名称、主要改动、TDD/测试证据、OpenSpec 验证和风险。
5. 报告 PR 地址并向 Leader交付 cleanup handoff；合并始终由用户决定。

可以为协作提前创建 Draft Pull Request，但只有完成前评审、Sync、Archive 和最终
验证全部通过，才能将其视为正式交付。

## PR merged 后的 Cleanup

用户确认 PR merged 后，开发 Leader从主项目执行 Cleanup；该状态由主 Codex 任务跟踪，
不回写已归档 change 的 tasks。Leader必须按下列顺序操作：

1. 获取最新 `origin/main`，识别 change branch、必要 commits 和 Desktop-managed worktree 路径。
2. 检查 change worktree clean；任何 tracked 或 untracked 变更都必须停止 Cleanup 并报告。
3. 验证必要 commits 已进入 `origin/main`；验证失败必须停止并报告。
4. 删除已验证的 change worktree。
5. 删除本地 `codex/<change-name>` 分支。
6. 若远端分支仍存在，删除远端同名分支。
7. 运行 `git worktree prune`，并报告最终 worktree/branch 状态。

每一步仅在前一步成功后进行。任何 Cleanup 失败必须报告失败命令、错误、已完成步骤和
剩余资源，不得静默完成。
