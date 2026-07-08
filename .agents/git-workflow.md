# Git Workflow

OpenSpec change 是本项目的正式工作单元。Git branch 是 change 的交付边界，commit 是阶段性检查点，worktree 只用于并行隔离。

## Branch Rules

除项目初始 baseline、紧急小修或用户明确要求直接在 `main` 操作外，正式 OpenSpec change 必须从 `main` 创建独立分支：

```text
codex/<change-name>
```

标准分支流程：

```text
1. 确认 `main` 干净，并从 `main` 开始。
2. 创建或切换到 `codex/<change-name>`。
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
13. 提交 archive 检查点，推荐 commit message：`spec: archive <change-name>`。
14. 通过 PR 或明确确认后合并回 `main`。
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
- worktree 目录建议放在 `tidewise-ai` 同级目录，并带 `tidewise-ai-wt-<change-name>` 前缀。
- 不要在两个 worktree 中同时修改同一个 OpenSpec change，避免 tasks 状态和 specs delta 冲突。

## Main Rules

- `main` 应保持可验证、可恢复的稳定状态。
- `main` 可以包含已 propose 但未 apply 的轻量 active change；不应长期包含半实现代码。
- 合并回 `main` 前应确认 OpenSpec 校验通过，且相关 tasks、sync/archive 状态与代码实现一致。

## Safety Rules

- 不要 revert 用户或其他 agent 的无关改动。
- 不要使用 `git reset --hard`、`git checkout --` 等破坏性命令，除非用户明确要求。
- 如果工作区存在无关变更，忽略它们；如果相关变更影响当前任务，先读懂并兼容。
- push、PR、merge 前必须先运行与当前 change 匹配的验证。
