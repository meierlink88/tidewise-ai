# Git 分支、worktree 与 Pull Request

## 新 change 自动初始化

Agent 接到新的仓库改动任务后，先检查 active changes 和现有 worktrees。若不存在
匹配项，应将创建隔离工作空间视为正常实施步骤，无需额外询问：

1. 运行 `git fetch origin main` 获取最新基线。
2. 使用分支 `codex/<change-name>`，基线必须是最新 `origin/main`。
3. 优先使用 Codex Desktop 管理的 worktree。
4. Desktop worktree 不可用时，使用：
   `git worktree add ../tidewise-ai-agentrun-worktrees/<change-name> -b codex/<change-name> origin/main`。
5. 运行 `git branch --show-current`、`git worktree list` 和 `git status --short` 验证环境。

如果同名 branch 或 worktree 已存在，先确认其对应同一个 active change，然后恢复该
工作空间；不得创建重复 change，也不得覆盖用户已有修改。

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

完成 Sync、Archive 和最终验证后：

1. 提交当前 change 的全部必要文件。
2. 推送 `codex/<change-name>`。
3. 创建 GitHub Pull Request，base 为 `main`。
4. PR 正文列出 change 名称、主要改动、TDD/测试证据、OpenSpec 验证和风险。
5. 报告 PR 地址；合并和删除分支由用户决定。

可以为协作提前创建 Draft Pull Request，但只有完成前评审、Sync、Archive 和最终
验证全部通过，才能将其视为正式交付。
