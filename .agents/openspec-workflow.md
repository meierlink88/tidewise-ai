# OpenSpec 工程流程

## 生命周期

每个正式变更都遵循：

`Explore -> Delegate -> Propose -> Leader Review -> Apply -> Validate -> Leader Acceptance -> Sync -> Archive -> Deliver -> Merge -> Cleanup`

阶段不能倒置。开发 Leader 只负责 Explore、Delegate、评审、批准、监控、验收和
Cleanup；执行 Agent负责 Propose、Apply、Validate、Sync、Archive 和 Deliver。评审
门禁只能由开发 Leader 明确批准，执行 Agent不得自行批准，也不得以“测试通过”替代
人工批准。用户控制 PR merge。

## 1. Explore

- 开发 Leader只读检查现状、关联规格、代码边界和风险。
- 先完成 `$eino-reference-first` 的三仓审计。
- 检查是否已有同范围 active change，避免重复创建。
- 尚未确定修改范围时使用 `$openspec-explore`，不得提前写实现。

## 2. Delegate

- 开发 Leader使用 Codex `create_thread` 创建独立执行任务和 Desktop-managed worktree。
- 执行 Agent确认独立 `codex/<change-name>` 分支、worktree 与 active change 匹配后，才
  写入 proposal 或实现文件。
- Leader不得在主任务中接管具体 change 实施；阻塞时应补充委派、重新委派或报告阻塞。

## 3. Propose

- change 名称使用简短的 kebab-case，并与分支 `codex/<change-name>` 一致。
- 执行 Agent使用 `$openspec-propose`，按 CLI 返回的依赖顺序生成 proposal、design、specs、tasks。
- 每个 capability 使用独立 `specs/<capability>/spec.md`。
- 需求必须可测试，每个 `Requirement` 至少包含一个 `Scenario`。
- 所有 change 产物执行中文优先规则。
- 行为变更的 tasks 必须明确 RED、GREEN、REFACTOR 和验证命令。

## 4. Leader Review

这是第一道人工门禁，即“提案评审”。执行 Agent必须向开发 Leader提交：

- 变更目标、范围和非目标；
- capability requirements 与关键 scenarios；
- 设计选择、风险和回滚方式；
- tasks、TDD 计划和验证方式；
- 分支与 worktree 状态。

只有开发 Leader明确表达“通过”“批准实施”等同意后，执行 Agent才能调用
`$openspec-apply-change`。执行 Agent不得自行批准。
若开发 Leader要求调整，先更新 change 产物并重新严格验证。

## 5. Apply

- 执行 Agent从 `openspec instructions apply --change <change-name> --json` 获取上下文和任务。
- 完整读取命令返回的所有 context files。
- 按 tasks 顺序执行；每项完成后立即更新 checkbox。
- 行为变更严格执行 RED、GREEN、REFACTOR，不得先写实现再补测试。
- 实现发现设计问题时，先更新 OpenSpec 产物，必要时重新请求提案评审。

## 6. Validate

至少执行：

- 与变更直接相关的聚焦测试；
- `go test ./...`；
- `openspec validate <change-name> --strict`；
- `git diff --check`；
- 变更范围、密钥和 `.reference/cloudwego/` 检查。

## 7. Leader Acceptance

这是第二道人工门禁，即“完成前评审”。执行 Agent必须向开发 Leader提交实现摘要、
完整 diff 范围、RED/GREEN/REFACTOR 证据、测试与 OpenSpec 验证结果、残余风险。只有
Leader明确完成 Leader Acceptance 后，执行 Agent才能执行 Sync 和 Archive；执行
Agent不得自行批准。

## 8. Sync

使用 `$openspec-sync-specs` 将已批准的 delta specs 同步到 `openspec/specs/`。同步后
检查正式规格与已实现行为一致，不得丢失原有 requirement 或 scenario。

## 9. Archive

使用 `$openspec-archive-change` 归档 change，再执行 OpenSpec 和工程最终验证。归档
失败时不得继续宣称完成。

## 10. Deliver

执行 Agent有选择地暂存当前 change 文件，执行密钥检查，提交、推送 change 分支并创建
或更新 Pull Request。PR 描述必须包含范围、测试证据、OpenSpec change 和已知风险。
执行 Agent在 PR 交付时向 Leader提交 cleanup handoff；该 handoff 后 OpenSpec tasks
可完成。合并始终由用户控制。

## 11. Merge 与 Cleanup

PR merged 后，Leader在主 Codex 任务中从主项目核验 worktree clean 和 change 提交已
进入 `origin/main`，再按安全顺序清理 worktree、本地分支、可选远端分支并 prune。该
operational state 不回写已归档 tasks；失败必须在主任务报告，不得静默完成。
