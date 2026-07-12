## Why

规则精简后，`.agents/skill-routing.md` 遗漏了 approved fallback worktree 对 `superpowers:using-git-worktrees` 的条件映射，导致架构测试与 GitHub Actions 失败。现有测试还依赖已被当前规则语义替代的脆弱字面措辞，需要改为验证 Desktop 强制受管、fallback 双条件和完整生命周期等稳定契约。

## What Changes

- 在 Skill 路由中恢复条件映射：Codex Desktop 可用时继续使用 Desktop 新任务创建受管 worktree；仅当 Desktop 机制不可用且用户明确批准 fallback 时，才使用 `superpowers:using-git-worktrees`。
- 更新架构规则测试，使其验证 Desktop-managed 默认路径、approved fallback 双条件、OpenSpec 阶段顺序与人工 Review、sequential/parallel 分流、两类 cleanup 顺序，以及 Archive 后才进入 finishing branch。
- 移除测试对“tasks 全部完成后”“Sync delta specs”“Archive change”“原生 worktree”等旧字面短语的依赖，不恢复旧的默认手工 worktree 路径。
- 保持 `streamline-agent-rules` 归档历史不变，不修改其他 active change、数据库状态、产品代码、API、`prototype/` 或 `doc/`。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `skill-driven-development-workflow`: 明确 Desktop-managed 与 approved fallback 两条 worktree 路径的 Skill 条件映射，并以自动化架构测试验证当前生命周期和隔离语义。

## Impact

- 规则文件：`.agents/skill-routing.md`。
- 自动化测试：`backend/internal/architecture/workflow_rules_test.go`。
- OpenSpec：新增本 change 的 proposal、design、delta spec 和 tasks；后续经 Review/Apply/Sync 后增量更新 `skill-driven-development-workflow` 主规格。
- 不新增依赖，不影响运行时 API、数据库、前端、`prototype/` 或 `doc/`。
