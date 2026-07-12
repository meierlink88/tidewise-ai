# Git Workflow

OpenSpec change 是正式工作单元，`codex/<change-name>` branch 是交付边界，commit 是检查点，worktree 用于 change 隔离。OpenSpec 阶段语义见 `.agents/openspec-workflow.md`。

## Desktop-Managed Worktree Gate

- 在 Codex Desktop 可用时，所有新 change 和并行 change 必须先通过 Desktop 新任务创建受管 worktree；agent 不得手工执行 `git worktree add`。
- branch 必须在 Desktop 受管任务内从最新 `origin/main` 创建或切换为 `codex/<change-name>`，不得把当前 worktree 手工创建 branch/worktree 作为等价默认路径。
- 只有 Codex Desktop 受管机制不可用且用户明确批准 fallback 时，agent 才可创建项目自有 Git worktree。两个条件缺一不可。
- 不得在两个 worktree 修改同一 change，也不得把其他 active change 的文件混入当前 branch。

## New Change Gate

创建或实现新 change 前必须：

1. 确认上一 change 已有 archive commit，且没有未提交的源码、测试、主规格或 archive 文件。
2. 执行 `git fetch origin`，确认基线为最新 `origin/main`，不得依赖可能过期的本地 `main`。
3. 在 Desktop 新任务创建的受管 worktree 内创建或切换 `codex/<change-name>`。
4. 确认 `git status --short` 干净，branch 名称匹配且不包含其他 change 修改。

建议检查：

```bash
git status --short
git branch --show-current
git log -1 --oneline
git merge-base HEAD origin/main
git worktree list
```

任一条件不满足时不得继续。无法使用 Desktop 时必须先报告原因并取得用户对 project-owned fallback 的明确批准。

## Approved Fallback Worktree

仅在 fallback 获批后，才可按以下边界创建项目自有 worktree：

- 基于最新 `origin/main` 创建 `codex/<change-name>`。
- 路径必须位于项目明确管理的 worktree 区域，并记录路径与所有权。
- 创建前后检查其他 worktree 与 branch，避免同一 change 被重复占用。
- fallback worktree 必须保持 scoped changes；不得移动或复制其他 active change 的未提交文件。

## Commit Checkpoints

- Propose artifacts 完整且 `openspec validate <change-name>` 通过后：`spec: propose <change-name>`。
- Apply 中每组独立可验证任务可以提交阶段性 checkpoint；tasks 必须同步更新。
- Sync/Archive 完成且 `openspec validate --all` 通过后：`spec: archive <change-name>`。
- commit 前运行新鲜验证、`git diff --check` 和 `git status --short`，只暂存当前 change 文件；不得加入依赖目录、构建产物、缓存、secret 或无关文件。

## Push, PR And Merge Gate

- Propose checkpoint 可以 push 供 Review；不得把它表达为完成态 PR。
- tasks、Apply 后人工 Review、Sync、Archive、`openspec validate --all` 和 archive commit 未完成前，不得创建完成态 PR 或 merge。
- Deliver 使用 `superpowers:finishing-a-development-branch`，commit/push/PR 优先使用 GitHub plugin。
- CI 失败使用 `github:gh-fix-ci`；PR review comments 使用 `github:gh-address-comments` 并先验证意见。
- merge 前确认主规格、归档和代码一致；merge 后先验证 `origin/main` 包含最终 archive commit，再开始 cleanup。

## Desktop-Managed Cleanup

Desktop-managed worktree 必须严格按以下顺序清理：

1. PR merge 后验证 `origin/main` 包含 change 的最终 archive commit。
2. 删除远端 `codex/<change-name>` branch。
3. 归档或关闭对应 Codex Desktop 任务，让 Desktop 释放托管 worktree。
4. 使用 Desktop 状态与 `git worktree list` 验证托管 worktree 已释放。
5. 仅在释放后删除仍存在的本地 change branch。

Agent 不得对 Desktop-managed worktree 执行 `rm`、`git worktree remove` 或其他手工目录移除。若 Desktop 尚未释放，必须记录待清理状态，不得删除仍被占用的本地 branch，也不得宣称 cleanup、Deliver 或 change closure 完成。

## Project-Owned Fallback Cleanup

经用户批准创建的 project-owned fallback worktree 必须严格按以下顺序清理：

1. PR merge 后验证 `origin/main` 包含 change 的最终 archive commit。
2. 删除远端 `codex/<change-name>` branch。
3. 确认 worktree 路径、所有权、branch 和工作区状态；只有路径与所有权均明确且无未提交/未跟踪文件时，才执行 `git worktree remove <path>`。
4. 删除本地 change branch。
5. 对已移除的项目自有 worktree 执行 `git worktree prune`，再用 `git worktree list` 验证无残留。

不得删除 Codex Desktop、其他工具或所有权不明的 worktree。发现未提交或未跟踪文件时必须暂停并迁移到对应 change，不得强制删除。

## Delivered State

只有同时满足以下条件，change 才是 `delivered`：

- `origin/main` 包含最终 archive commit。
- 远端 change branch 已删除。
- 对应 Desktop-managed 或 project-owned worktree 已按所属路径完整释放。
- 不再有被占用或待清理的本地 change branch/worktree。
- 当前 change 没有未提交文件。

Archive 成功只代表 `archived`，不代表 `delivered`。Deliver 未完成不得声明 change 关闭或开始下一 change。

## Safety

- 不 revert 用户或其他 agent 的无关改动。
- 不使用 `git reset --hard`、`git checkout --` 或强制删除，除非用户明确要求。
- push、PR、merge 和 cleanup 前必须运行与当前阶段匹配的新鲜验证。
