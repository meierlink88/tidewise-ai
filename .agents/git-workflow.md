# Git Workflow

OpenSpec change 是本项目的正式工作单元。Git branch 是 change 的交付边界，commit 是阶段性检查点，worktree 只用于并行隔离。

## Branch Rules

除项目初始 baseline、紧急小修或用户明确要求直接在 `main` 操作外，正式 OpenSpec change 必须从最新 `origin/main` 创建独立分支：

```text
codex/<change-name>
```

标准分支流程：

```text
1. 执行 `git fetch origin`，更新远端引用。
2. 基于最新 `origin/main` 创建 `codex/<change-name>`，不得仅依赖可能过期的本地 `main`。
3. 执行 Explore/Propose，生成或更新 `openspec/changes/<change-name>/` artifacts。
4. 运行 `openspec validate <change-name>`。
5. 提交 propose 检查点，推荐 commit message：`spec: propose <change-name>`。
6. 等待或完成 Review，确认 artifacts 可以进入实现。
7. 执行 Apply，严格按 `tasks.md` 顺序实现。
8. 每完成一组可验证任务后更新 checkbox，并按需要提交阶段性 commit。
9. tasks 全部完成后运行适当验证。
10. Sync delta specs 到 `openspec/specs/`。
11. Archive change 到 `openspec/changes/archive/`。
12. 运行 `openspec validate --all`。
13. 检查 scoped diff，只暂存当前 change 文件并提交 archive 检查点，推荐 commit message：`spec: archive <change-name>`。
14. 验证 `git status --short` 不再包含当前 change 的未提交文件，并用 `git log -1` 确认 archive commit 存在。
15. 使用 `superpowers:finishing-a-development-branch` 和 GitHub plugin，通过 PR 或明确确认后合并回 `main`。
16. PR 合并后确认默认分支已包含 change 的 archive commit，并删除远端 change branch。
17. 切换出 change worktree 后删除本地 change branch；如果该 worktree 由本项目创建，则移除 worktree 并执行 `git worktree prune`。
18. 只有完成 archive commit、分支交付和 branch/worktree cleanup 后，change 才进入 `delivered` 状态并允许启动下一 change。
```

## Commit Rules

- propose artifacts 完整且 `openspec validate <change-name>` 通过后，应提交一次 `spec: propose <change-name>`。
- apply 过程中，完成一组有独立验证意义的任务后可以提交一次，例如 `chore: add backend config foundation` 或 `feat: add backend health endpoints`。
- 不要在 tasks 未更新、验证未运行或 artifacts 明显过期时提交完成态代码。
- sync/archive 完成且 `openspec validate --all` 通过后，应提交一次 `spec: archive <change-name>`。
- commit 前必须检查 `git status --short`，确认没有把 `node_modules`、构建产物、缓存、真实 secret 或无关文件加入提交。

## Worktree Rules

- 默认一个 change 使用当前 worktree 加一个独立 branch 即可。
- 当多个 change 并行、某个 change 长期未完成但需要切换任务、或多个 Codex 线程同时工作时，才创建额外 worktree。
- 创建并行 worktree 时使用 `superpowers:using-git-worktrees`，并确保它基于最新 `origin/main`。
- 在 Codex Desktop 中优先创建与独立任务绑定的原生 worktree；手工 worktree 只作为原生能力不可用时的替代方案。
- 不要在两个 worktree 中同时修改同一个 OpenSpec change，避免 tasks 状态和 specs delta 冲突。

## New Change Gate

创建或实现新 OpenSpec change 前必须同时满足：

- 上一个 change 的 archive commit 已存在，且其源码、测试、主规格和 archive 文件均被 Git 跟踪。
- 当前 branch 为 `codex/<new-change-name>`，并基于最新 `origin/main`；不得沿用上一个 change 的 branch。
- 当前 worktree 不包含其他 change 的未提交修改。
- 如果另一个 change 仍在 review、PR 或实现中，必须为新 change 使用独立 worktree，禁止在同一 worktree 混合两个 change。

建议检查命令：

```bash
git status --short
git branch --show-current
git log -1 --oneline
git merge-base HEAD origin/main
```

任一条件不满足时，必须先完成上一 change 的 scoped commit/交付，或创建并切换到新 change 的独立 worktree。OpenSpec archive 成功只代表 `archived`，不代表 Git 已 `delivered`。

## Delivered Change Cleanup

PR 合并或本地合并成功后必须完成：

- 验证 `origin/main` 已包含 change 的最终 commit。
- 删除远端 `codex/<change-name>` branch。
- 确认没有 worktree 正在使用该 branch 后，删除本地 branch。
- 仅清理本项目创建且路径可确认的 change worktree；不得删除 Codex Desktop 或其他工具拥有的未知 worktree。
- 对已移除的项目自有 worktree 运行 `git worktree prune`，并用 `git worktree list` 验证没有残留。

删除 branch 或 worktree 前必须确认其中没有未提交和未跟踪文件。若存在文件，先迁移到对应新 change 的 branch/worktree，不得强制删除。

## GitHub Rules

- commit、push 和创建 PR 优先使用 `github:yeet`。
- CI 失败使用 `github:gh-fix-ci`。
- PR review comments 使用 `github:gh-address-comments`，并先按 `superpowers:receiving-code-review` 验证意见。
- tasks、sync、archive 和 `openspec validate --all` 未完成前，不得进入 `superpowers:finishing-a-development-branch` 或创建完成态 PR。

## Main Rules

- `main` 应保持可验证、可恢复的稳定状态。
- `main` 可以包含已 propose 但未 apply 的轻量 active change；不应长期包含半实现代码。
- 合并回 `main` 前应确认 OpenSpec 校验通过，且相关 tasks、sync/archive 状态与代码实现一致。

## Safety Rules

- 不要 revert 用户或其他 agent 的无关改动。
- 不要使用 `git reset --hard`、`git checkout --` 等破坏性命令，除非用户明确要求。
- 如果工作区存在无关变更，忽略它们；如果相关变更影响当前任务，先读懂并兼容。
- push、PR、merge 前必须先运行与当前 change 匹配的验证。
